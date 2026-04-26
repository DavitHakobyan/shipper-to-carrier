package carrieridentity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/identity"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/verification"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

type dbtx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateCarrier(ctx context.Context, actor identity.AuthenticatedAccount, input CreateCarrierInput, now time.Time) (OnboardingStatus, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return OnboardingStatus{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	exists, err := r.carrierExistsForAccount(ctx, tx, actor.AccountID)
	if err != nil {
		return OnboardingStatus{}, err
	}
	if exists {
		return OnboardingStatus{}, ErrCarrierExists
	}

	carrierID := uuid.NewString()
	caseID := uuid.NewString()

	if _, err := tx.Exec(ctx, `
		INSERT INTO carrier_accounts (id, account_id, legal_name, doing_business_as, status, onboarding_stage, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, carrierID, actor.AccountID, input.LegalName, input.DoingBusinessAs, CarrierStatusActive, OnboardingStageBusinessSubmitted, now, now); err != nil {
		return OnboardingStatus{}, fmt.Errorf("insert carrier account: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO carrier_profiles (carrier_account_id, contact_phone, contact_email, fleet_size_declared, operating_regions, preferred_load_types)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, carrierID, input.ContactPhone, actor.Email, input.FleetSizeDeclared, input.OperatingRegions, input.PreferredLoadTypes); err != nil {
		return OnboardingStatus{}, fmt.Errorf("insert carrier profile: %w", err)
	}

	addressID := uuid.NewString()
	if _, err := tx.Exec(ctx, `
		INSERT INTO carrier_addresses (id, carrier_account_id, address_type, line1, line2, city, state, postal_code, country, valid_from, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10)
	`, addressID, carrierID, input.Address.AddressType, input.Address.Line1, input.Address.Line2, input.Address.City, input.Address.State, input.Address.PostalCode, input.Address.Country, now); err != nil {
		return OnboardingStatus{}, fmt.Errorf("insert carrier address: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO verification_cases (id, carrier_account_id, case_type, status, opened_at)
		VALUES ($1, $2, $3, $4, $5)
	`, caseID, carrierID, verification.CaseTypeOnboarding, verification.CaseStatusOpen, now); err != nil {
		return OnboardingStatus{}, fmt.Errorf("insert verification case: %w", err)
	}

	for _, requirementType := range verification.DefaultRequirementTypes() {
		status := verification.RequirementStatusPending
		var satisfiedAt any = nil
		if requirementType == verification.RequirementTypeBusinessProfile || requirementType == verification.RequirementTypeOperatingAddr {
			status = verification.RequirementStatusSatisfied
			satisfiedAt = now
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO verification_requirements (id, verification_case_id, requirement_type, status, satisfied_at, notes)
			VALUES ($1, $2, $3, $4, $5, '')
		`, uuid.NewString(), caseID, requirementType, status, satisfiedAt); err != nil {
			return OnboardingStatus{}, fmt.Errorf("insert verification requirement: %w", err)
		}
	}

	if err := r.insertEvent(ctx, tx, carrierID, "carrier.created", map[string]any{
		"onboardingStage": OnboardingStageBusinessSubmitted,
	}); err != nil {
		return OnboardingStatus{}, err
	}

	status, err := r.loadOnboardingStatusByCarrierID(ctx, tx, actor.AccountID, carrierID)
	if err != nil {
		return OnboardingStatus{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return OnboardingStatus{}, fmt.Errorf("commit transaction: %w", err)
	}

	return status, nil
}

func (r *PostgresRepository) AddOwner(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string, input AddOwnerInput, now time.Time) (OnboardingStatus, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return OnboardingStatus{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	carrier, err := r.findCarrier(ctx, tx, actor.AccountID, carrierID)
	if err != nil {
		return OnboardingStatus{}, err
	}

	if input.IsPrimaryContact {
		if _, err := tx.Exec(ctx, `
			UPDATE carrier_owner_identities
			SET is_primary_contact = FALSE
			WHERE carrier_account_id = $1
		`, carrier.ID); err != nil {
			return OnboardingStatus{}, fmt.Errorf("clear primary owner: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO carrier_owner_identities (id, carrier_account_id, full_name, phone, email, ownership_role, is_primary_contact, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, uuid.NewString(), carrier.ID, input.FullName, input.Phone, input.Email, input.OwnershipRole, input.IsPrimaryContact, now); err != nil {
		return OnboardingStatus{}, fmt.Errorf("insert owner: %w", err)
	}

	if err := r.insertEvent(ctx, tx, carrier.ID, "carrier.owner_added", map[string]any{
		"ownershipRole": input.OwnershipRole,
	}); err != nil {
		return OnboardingStatus{}, err
	}

	status, err := r.recomputeAndLoad(ctx, tx, actor.AccountID, carrier.ID, now)
	if err != nil {
		return OnboardingStatus{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return OnboardingStatus{}, fmt.Errorf("commit transaction: %w", err)
	}

	return status, nil
}

func (r *PostgresRepository) UpsertAuthority(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string, input UpsertAuthorityInput, now time.Time) (OnboardingStatus, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return OnboardingStatus{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	carrier, err := r.findCarrier(ctx, tx, actor.AccountID, carrierID)
	if err != nil {
		return OnboardingStatus{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO carrier_authority_links (id, carrier_account_id, dot_number, mc_number, usdot_status, authority_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		ON CONFLICT (carrier_account_id)
		DO UPDATE SET
			dot_number = EXCLUDED.dot_number,
			mc_number = EXCLUDED.mc_number,
			usdot_status = EXCLUDED.usdot_status,
			authority_type = EXCLUDED.authority_type,
			updated_at = EXCLUDED.updated_at
	`, uuid.NewString(), carrier.ID, input.DOTNumber, input.MCNumber, input.USDOTStatus, input.AuthorityType, now); err != nil {
		return OnboardingStatus{}, fmt.Errorf("upsert authority link: %w", err)
	}

	if err := r.insertEvent(ctx, tx, carrier.ID, "carrier.authority_linked", map[string]any{
		"dotNumber": input.DOTNumber,
		"mcNumber":  input.MCNumber,
	}); err != nil {
		return OnboardingStatus{}, err
	}

	status, err := r.recomputeAndLoad(ctx, tx, actor.AccountID, carrier.ID, now)
	if err != nil {
		return OnboardingStatus{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return OnboardingStatus{}, fmt.Errorf("commit transaction: %w", err)
	}

	return status, nil
}

func (r *PostgresRepository) AddInsurance(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string, input AddInsuranceInput, now time.Time) (OnboardingStatus, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return OnboardingStatus{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	carrier, err := r.findCarrier(ctx, tx, actor.AccountID, carrierID)
	if err != nil {
		return OnboardingStatus{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO carrier_insurance_policies (
			id, carrier_account_id, provider_name, policy_number_hash, coverage_type, effective_at, expires_at, verification_status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
	`, uuid.NewString(), carrier.ID, input.ProviderName, hashValue(input.PolicyNumber), input.CoverageType, input.EffectiveAt.UTC(), input.ExpiresAt.UTC(), input.VerificationStatus, now); err != nil {
		return OnboardingStatus{}, fmt.Errorf("insert insurance policy: %w", err)
	}

	if err := r.insertEvent(ctx, tx, carrier.ID, "carrier.insurance_added", map[string]any{
		"providerName": input.ProviderName,
		"coverageType": input.CoverageType,
	}); err != nil {
		return OnboardingStatus{}, err
	}

	status, err := r.recomputeAndLoad(ctx, tx, actor.AccountID, carrier.ID, now)
	if err != nil {
		return OnboardingStatus{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return OnboardingStatus{}, fmt.Errorf("commit transaction: %w", err)
	}

	return status, nil
}

func (r *PostgresRepository) GetOnboardingStatus(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string) (OnboardingStatus, error) {
	return r.loadOnboardingStatusByCarrierID(ctx, r.pool, actor.AccountID, carrierID)
}

func (r *PostgresRepository) GetCurrentOnboardingStatus(ctx context.Context, actor identity.AuthenticatedAccount) (OnboardingStatus, error) {
	return r.loadOnboardingStatusByAccountID(ctx, r.pool, actor.AccountID)
}

func (r *PostgresRepository) carrierExistsForAccount(ctx context.Context, q dbtx, accountID string) (bool, error) {
	var exists bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM carrier_accounts
			WHERE account_id = $1
		)
	`, accountID).Scan(&exists); err != nil {
		return false, fmt.Errorf("query carrier existence: %w", err)
	}

	return exists, nil
}

func (r *PostgresRepository) findCarrier(ctx context.Context, q dbtx, accountID string, carrierID string) (CarrierAccount, error) {
	var carrier CarrierAccount
	err := q.QueryRow(ctx, `
		SELECT id, account_id, legal_name, doing_business_as, status, onboarding_stage, created_at, updated_at
		FROM carrier_accounts
		WHERE id = $1 AND account_id = $2
	`, carrierID, accountID).Scan(
		&carrier.ID,
		&carrier.AccountID,
		&carrier.LegalName,
		&carrier.DoingBusinessAs,
		&carrier.Status,
		&carrier.OnboardingStage,
		&carrier.CreatedAt,
		&carrier.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CarrierAccount{}, ErrCarrierNotFound
		}
		return CarrierAccount{}, fmt.Errorf("query carrier: %w", err)
	}

	return carrier, nil
}

func (r *PostgresRepository) loadOnboardingStatusByAccountID(ctx context.Context, q dbtx, accountID string) (OnboardingStatus, error) {
	var carrierID string
	if err := q.QueryRow(ctx, `
		SELECT id
		FROM carrier_accounts
		WHERE account_id = $1
		LIMIT 1
	`, accountID).Scan(&carrierID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OnboardingStatus{}, ErrCarrierNotFound
		}
		return OnboardingStatus{}, fmt.Errorf("query current carrier: %w", err)
	}

	return r.loadOnboardingStatusByCarrierID(ctx, q, accountID, carrierID)
}

func (r *PostgresRepository) loadOnboardingStatusByCarrierID(ctx context.Context, q dbtx, accountID string, carrierID string) (OnboardingStatus, error) {
	carrier, err := r.findCarrier(ctx, q, accountID, carrierID)
	if err != nil {
		return OnboardingStatus{}, err
	}

	var status OnboardingStatus
	status.Carrier = carrier

	if err := q.QueryRow(ctx, `
		SELECT carrier_account_id, contact_phone, contact_email, fleet_size_declared, operating_regions, preferred_load_types
		FROM carrier_profiles
		WHERE carrier_account_id = $1
	`, carrier.ID).Scan(
		&status.Profile.CarrierAccountID,
		&status.Profile.ContactPhone,
		&status.Profile.ContactEmail,
		&status.Profile.FleetSizeDeclared,
		&status.Profile.OperatingRegions,
		&status.Profile.PreferredLoadTypes,
	); err != nil {
		return OnboardingStatus{}, fmt.Errorf("query carrier profile: %w", err)
	}

	rows, err := q.Query(ctx, `
		SELECT id, carrier_account_id, address_type, line1, line2, city, state, postal_code, country, valid_from, created_at
		FROM carrier_addresses
		WHERE carrier_account_id = $1
		ORDER BY created_at ASC
	`, carrier.ID)
	if err != nil {
		return OnboardingStatus{}, fmt.Errorf("query addresses: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var address CarrierAddress
		if err := rows.Scan(
			&address.ID,
			&address.CarrierAccountID,
			&address.AddressType,
			&address.Line1,
			&address.Line2,
			&address.City,
			&address.State,
			&address.PostalCode,
			&address.Country,
			&address.ValidFrom,
			&address.CreatedAt,
		); err != nil {
			return OnboardingStatus{}, fmt.Errorf("scan address: %w", err)
		}
		status.Addresses = append(status.Addresses, address)
	}
	rows.Close()

	rows, err = q.Query(ctx, `
		SELECT id, carrier_account_id, full_name, phone, email, ownership_role, is_primary_contact, created_at
		FROM carrier_owner_identities
		WHERE carrier_account_id = $1
		ORDER BY created_at ASC
	`, carrier.ID)
	if err != nil {
		return OnboardingStatus{}, fmt.Errorf("query owners: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var owner CarrierOwnerIdentity
		if err := rows.Scan(
			&owner.ID,
			&owner.CarrierAccountID,
			&owner.FullName,
			&owner.Phone,
			&owner.Email,
			&owner.OwnershipRole,
			&owner.IsPrimaryContact,
			&owner.CreatedAt,
		); err != nil {
			return OnboardingStatus{}, fmt.Errorf("scan owner: %w", err)
		}
		status.Owners = append(status.Owners, owner)
	}
	rows.Close()

	var authority CarrierAuthorityLink
	err = q.QueryRow(ctx, `
		SELECT id, carrier_account_id, dot_number, mc_number, usdot_status, authority_type, created_at, updated_at
		FROM carrier_authority_links
		WHERE carrier_account_id = $1
		LIMIT 1
	`, carrier.ID).Scan(
		&authority.ID,
		&authority.CarrierAccountID,
		&authority.DOTNumber,
		&authority.MCNumber,
		&authority.USDOTStatus,
		&authority.AuthorityType,
		&authority.CreatedAt,
		&authority.UpdatedAt,
	)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return OnboardingStatus{}, fmt.Errorf("query authority link: %w", err)
	default:
		status.AuthorityLink = &authority
	}

	rows, err = q.Query(ctx, `
		SELECT id, carrier_account_id, provider_name, policy_number_hash, coverage_type, effective_at, expires_at, verification_status, created_at, updated_at
		FROM carrier_insurance_policies
		WHERE carrier_account_id = $1
		ORDER BY created_at ASC
	`, carrier.ID)
	if err != nil {
		return OnboardingStatus{}, fmt.Errorf("query insurance policies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var policy CarrierInsurancePolicy
		if err := rows.Scan(
			&policy.ID,
			&policy.CarrierAccountID,
			&policy.ProviderName,
			&policy.PolicyNumberHash,
			&policy.CoverageType,
			&policy.EffectiveAt,
			&policy.ExpiresAt,
			&policy.VerificationStatus,
			&policy.CreatedAt,
			&policy.UpdatedAt,
		); err != nil {
			return OnboardingStatus{}, fmt.Errorf("scan insurance policy: %w", err)
		}
		status.InsurancePolicies = append(status.InsurancePolicies, policy)
	}
	rows.Close()

	if err := q.QueryRow(ctx, `
		SELECT id, carrier_account_id, case_type, status, opened_at, closed_at, assigned_admin_id
		FROM verification_cases
		WHERE carrier_account_id = $1
		LIMIT 1
	`, carrier.ID).Scan(
		&status.VerificationCase.ID,
		&status.VerificationCase.CarrierAccountID,
		&status.VerificationCase.CaseType,
		&status.VerificationCase.Status,
		&status.VerificationCase.OpenedAt,
		&status.VerificationCase.ClosedAt,
		&status.VerificationCase.AssignedAdminID,
	); err != nil {
		return OnboardingStatus{}, fmt.Errorf("query verification case: %w", err)
	}

	rows, err = q.Query(ctx, `
		SELECT id, verification_case_id, requirement_type, status, satisfied_at, notes
		FROM verification_requirements
		WHERE verification_case_id = $1
		ORDER BY requirement_type ASC
	`, status.VerificationCase.ID)
	if err != nil {
		return OnboardingStatus{}, fmt.Errorf("query verification requirements: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var req verification.Requirement
		if err := rows.Scan(
			&req.ID,
			&req.VerificationCaseID,
			&req.RequirementType,
			&req.Status,
			&req.SatisfiedAt,
			&req.Notes,
		); err != nil {
			return OnboardingStatus{}, fmt.Errorf("scan verification requirement: %w", err)
		}
		status.Requirements = append(status.Requirements, req)
		if req.Status != verification.RequirementStatusSatisfied {
			status.MissingRequirements = append(status.MissingRequirements, req.RequirementType)
		}
	}
	rows.Close()

	return status, nil
}

func (r *PostgresRepository) recomputeAndLoad(ctx context.Context, tx pgx.Tx, accountID string, carrierID string, now time.Time) (OnboardingStatus, error) {
	facts, caseID, err := r.loadFacts(ctx, tx, carrierID)
	if err != nil {
		return OnboardingStatus{}, err
	}

	stage := DeriveOnboardingStage(facts)
	caseStatus := verification.CaseStatusOpen
	if len(MissingRequirements(facts)) == 0 {
		caseStatus = verification.CaseStatusReviewReady
	}

	if _, err := tx.Exec(ctx, `
		UPDATE carrier_accounts
		SET onboarding_stage = $2, updated_at = $3
		WHERE id = $1
	`, carrierID, stage, now); err != nil {
		return OnboardingStatus{}, fmt.Errorf("update carrier stage: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE verification_cases
		SET status = $2
		WHERE id = $1
	`, caseID, caseStatus); err != nil {
		return OnboardingStatus{}, fmt.Errorf("update verification case: %w", err)
	}

	requirementStates := map[verification.RequirementType]bool{
		verification.RequirementTypeBusinessProfile: facts.HasBusinessProfile,
		verification.RequirementTypeOwnerIdentity:   facts.HasOwnerIdentity,
		verification.RequirementTypeOperatingAddr:   facts.HasOperatingAddr,
		verification.RequirementTypeAuthorityLink:   facts.HasAuthorityLink,
		verification.RequirementTypeInsurancePolicy: facts.HasInsurance,
	}

	for requirementType, satisfied := range requirementStates {
		reqStatus := verification.RequirementStatusPending
		var satisfiedAt any = nil
		if satisfied {
			reqStatus = verification.RequirementStatusSatisfied
			satisfiedAt = now
		}

		if _, err := tx.Exec(ctx, `
			UPDATE verification_requirements
			SET status = $3, satisfied_at = $4
			WHERE verification_case_id = $1 AND requirement_type = $2
		`, caseID, requirementType, reqStatus, satisfiedAt); err != nil {
			return OnboardingStatus{}, fmt.Errorf("update verification requirement: %w", err)
		}
	}

	return r.loadOnboardingStatusByCarrierID(ctx, tx, accountID, carrierID)
}

func (r *PostgresRepository) loadFacts(ctx context.Context, q dbtx, carrierID string) (OnboardingFacts, string, error) {
	var facts OnboardingFacts
	var caseID string

	if err := q.QueryRow(ctx, `
		SELECT
			EXISTS (SELECT 1 FROM carrier_profiles WHERE carrier_account_id = $1),
			EXISTS (SELECT 1 FROM carrier_addresses WHERE carrier_account_id = $1 AND address_type = 'operating'),
			EXISTS (SELECT 1 FROM carrier_owner_identities WHERE carrier_account_id = $1),
			EXISTS (SELECT 1 FROM carrier_authority_links WHERE carrier_account_id = $1),
			EXISTS (SELECT 1 FROM carrier_insurance_policies WHERE carrier_account_id = $1),
			(SELECT id FROM verification_cases WHERE carrier_account_id = $1 LIMIT 1)
	`, carrierID).Scan(
		&facts.HasBusinessProfile,
		&facts.HasOperatingAddr,
		&facts.HasOwnerIdentity,
		&facts.HasAuthorityLink,
		&facts.HasInsurance,
		&caseID,
	); err != nil {
		return OnboardingFacts{}, "", fmt.Errorf("query onboarding facts: %w", err)
	}

	return facts, caseID, nil
}

func (r *PostgresRepository) insertEvent(ctx context.Context, q dbtx, carrierID string, eventType string, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal verification event payload: %w", err)
	}

	if _, err := q.Exec(ctx, `
		INSERT INTO verification_events (id, carrier_account_id, event_type, event_payload_json, occurred_at)
		VALUES ($1, $2, $3, $4, $5)
	`, uuid.NewString(), carrierID, eventType, body, time.Now().UTC()); err != nil {
		return fmt.Errorf("insert verification event: %w", err)
	}

	return nil
}

func hashValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
