package carrieridentity

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/identity"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  time.Now,
	}
}

func (s *Service) CreateCarrier(ctx context.Context, actor identity.AuthenticatedAccount, input CreateCarrierInput) (OnboardingStatus, error) {
	if err := validateCarrierActor(actor); err != nil {
		return OnboardingStatus{}, err
	}
	if err := validateCreateCarrierInput(input); err != nil {
		return OnboardingStatus{}, err
	}

	return s.repo.CreateCarrier(ctx, actor, normalizeCreateCarrierInput(input), s.now().UTC())
}

func (s *Service) AddOwner(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string, input AddOwnerInput) (OnboardingStatus, error) {
	if err := validateCarrierActor(actor); err != nil {
		return OnboardingStatus{}, err
	}
	if strings.TrimSpace(carrierID) == "" {
		return OnboardingStatus{}, ErrCarrierNotFound
	}
	if err := validateAddOwnerInput(input); err != nil {
		return OnboardingStatus{}, err
	}

	return s.repo.AddOwner(ctx, actor, carrierID, normalizeOwnerInput(input), s.now().UTC())
}

func (s *Service) UpsertAuthority(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string, input UpsertAuthorityInput) (OnboardingStatus, error) {
	if err := validateCarrierActor(actor); err != nil {
		return OnboardingStatus{}, err
	}
	if strings.TrimSpace(carrierID) == "" {
		return OnboardingStatus{}, ErrCarrierNotFound
	}
	if err := validateAuthorityInput(input); err != nil {
		return OnboardingStatus{}, err
	}

	return s.repo.UpsertAuthority(ctx, actor, carrierID, normalizeAuthorityInput(input), s.now().UTC())
}

func (s *Service) AddInsurance(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string, input AddInsuranceInput) (OnboardingStatus, error) {
	if err := validateCarrierActor(actor); err != nil {
		return OnboardingStatus{}, err
	}
	if strings.TrimSpace(carrierID) == "" {
		return OnboardingStatus{}, ErrCarrierNotFound
	}
	if err := validateInsuranceInput(input); err != nil {
		return OnboardingStatus{}, err
	}

	return s.repo.AddInsurance(ctx, actor, carrierID, input, s.now().UTC())
}

func (s *Service) GetOnboardingStatus(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string) (OnboardingStatus, error) {
	if err := validateCarrierActor(actor); err != nil {
		return OnboardingStatus{}, err
	}
	if strings.TrimSpace(carrierID) == "" {
		return OnboardingStatus{}, ErrCarrierNotFound
	}

	return s.repo.GetOnboardingStatus(ctx, actor, carrierID)
}

func (s *Service) GetCurrentOnboardingStatus(ctx context.Context, actor identity.AuthenticatedAccount) (OnboardingStatus, error) {
	if err := validateCarrierActor(actor); err != nil {
		return OnboardingStatus{}, err
	}

	return s.repo.GetCurrentOnboardingStatus(ctx, actor)
}

func validateCarrierActor(actor identity.AuthenticatedAccount) error {
	if actor.Role != identity.RoleCarrier {
		return ErrForbidden
	}

	return nil
}

func validateCreateCarrierInput(input CreateCarrierInput) error {
	switch {
	case strings.TrimSpace(input.LegalName) == "":
		return errors.New("legal name is required")
	case strings.TrimSpace(input.ContactPhone) == "":
		return errors.New("contact phone is required")
	case strings.TrimSpace(input.Address.Line1) == "":
		return errors.New("address line1 is required")
	case strings.TrimSpace(input.Address.City) == "":
		return errors.New("address city is required")
	case strings.TrimSpace(input.Address.State) == "":
		return errors.New("address state is required")
	case strings.TrimSpace(input.Address.PostalCode) == "":
		return errors.New("address postal code is required")
	case strings.TrimSpace(input.Address.Country) == "":
		return errors.New("address country is required")
	default:
		return nil
	}
}

func validateAddOwnerInput(input AddOwnerInput) error {
	switch {
	case strings.TrimSpace(input.FullName) == "":
		return errors.New("owner full name is required")
	case strings.TrimSpace(input.Email) == "":
		return errors.New("owner email is required")
	case strings.TrimSpace(input.OwnershipRole) == "":
		return errors.New("owner role is required")
	default:
		return nil
	}
}

func validateAuthorityInput(input UpsertAuthorityInput) error {
	if strings.TrimSpace(input.DOTNumber) == "" && strings.TrimSpace(input.MCNumber) == "" {
		return errors.New("dot number or mc number is required")
	}

	return nil
}

func validateInsuranceInput(input AddInsuranceInput) error {
	switch {
	case strings.TrimSpace(input.ProviderName) == "":
		return errors.New("insurance provider name is required")
	case strings.TrimSpace(input.PolicyNumber) == "":
		return errors.New("insurance policy number is required")
	case strings.TrimSpace(input.CoverageType) == "":
		return errors.New("insurance coverage type is required")
	case input.ExpiresAt.IsZero():
		return errors.New("insurance expiration is required")
	case input.EffectiveAt.IsZero():
		return errors.New("insurance effective date is required")
	case !input.ExpiresAt.After(input.EffectiveAt):
		return errors.New("insurance expiration must be after effective date")
	default:
		return nil
	}
}

func normalizeCreateCarrierInput(input CreateCarrierInput) CreateCarrierInput {
	input.LegalName = strings.TrimSpace(input.LegalName)
	input.DoingBusinessAs = strings.TrimSpace(input.DoingBusinessAs)
	input.ContactPhone = strings.TrimSpace(input.ContactPhone)
	input.Address.AddressType = strings.TrimSpace(input.Address.AddressType)
	if input.Address.AddressType == "" {
		input.Address.AddressType = "operating"
	}
	input.Address.Line1 = strings.TrimSpace(input.Address.Line1)
	input.Address.Line2 = strings.TrimSpace(input.Address.Line2)
	input.Address.City = strings.TrimSpace(input.Address.City)
	input.Address.State = strings.TrimSpace(input.Address.State)
	input.Address.PostalCode = strings.TrimSpace(input.Address.PostalCode)
	input.Address.Country = strings.TrimSpace(input.Address.Country)
	input.OperatingRegions = normalizeSlice(input.OperatingRegions)
	input.PreferredLoadTypes = normalizeSlice(input.PreferredLoadTypes)
	return input
}

func normalizeOwnerInput(input AddOwnerInput) AddOwnerInput {
	input.FullName = strings.TrimSpace(input.FullName)
	input.Phone = strings.TrimSpace(input.Phone)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.OwnershipRole = strings.TrimSpace(input.OwnershipRole)
	return input
}

func normalizeAuthorityInput(input UpsertAuthorityInput) UpsertAuthorityInput {
	input.DOTNumber = strings.TrimSpace(input.DOTNumber)
	input.MCNumber = strings.TrimSpace(input.MCNumber)
	input.USDOTStatus = strings.TrimSpace(input.USDOTStatus)
	input.AuthorityType = strings.TrimSpace(input.AuthorityType)
	return input
}

func normalizeSlice(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}
