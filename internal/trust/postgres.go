package trust

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) SaveEvaluation(ctx context.Context, evaluation Evaluation) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := evaluation.Scorecard.GeneratedAt
	if _, err := tx.Exec(ctx, `
		UPDATE access_grants
		SET revoked_at = $2
		WHERE carrier_account_id = $1
		  AND revoked_at IS NULL
	`, evaluation.Scorecard.CarrierAccountID, now); err != nil {
		return fmt.Errorf("revoke prior grants: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO carrier_scorecards (
			id, carrier_account_id, score_version, score_value, score_band, eligibility_tier, verification_completeness, reason_summary, generated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, evaluation.Scorecard.ID, evaluation.Scorecard.CarrierAccountID, evaluation.Scorecard.ScoreVersion, evaluation.Scorecard.ScoreValue, evaluation.Scorecard.ScoreBand, evaluation.Scorecard.EligibilityTier, evaluation.Scorecard.VerificationCompleteness, evaluation.Scorecard.ReasonSummary, evaluation.Scorecard.GeneratedAt); err != nil {
		return fmt.Errorf("insert scorecard: %w", err)
	}

	for _, input := range evaluation.Inputs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO carrier_score_inputs (id, carrier_account_id, source_scorecard_id, input_type, source, value_numeric, value_text, effective_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, input.ID, input.CarrierAccountID, input.SourceScorecardID, input.InputType, input.Source, input.ValueNumeric, input.ValueText, input.EffectiveAt); err != nil {
			return fmt.Errorf("insert score input: %w", err)
		}
	}

	for _, grant := range evaluation.AccessGrants {
		if _, err := tx.Exec(ctx, `
			INSERT INTO access_grants (id, carrier_account_id, grant_type, grant_value, granted_at, revoked_at, source_scorecard_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, grant.ID, grant.CarrierAccountID, grant.GrantType, grant.GrantValue, grant.GrantedAt, grant.RevokedAt, grant.SourceScorecardID); err != nil {
			return fmt.Errorf("insert access grant: %w", err)
		}
	}

	for _, signal := range evaluation.FraudSignals {
		if _, err := tx.Exec(ctx, `
			INSERT INTO fraud_signals (id, carrier_account_id, source_scorecard_id, signal_type, severity, status, detected_at, evidence_json, reviewed_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, signal.ID, signal.CarrierAccountID, signal.SourceScorecardID, signal.SignalType, signal.Severity, signal.Status, signal.DetectedAt, signal.EvidenceJSON, signal.ReviewedAt); err != nil {
			return fmt.Errorf("insert fraud signal: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresRepository) LatestTrust(ctx context.Context, carrierAccountID string) (TrustStatus, error) {
	var status TrustStatus

	err := r.pool.QueryRow(ctx, `
		SELECT id, carrier_account_id, score_version, score_value, score_band, eligibility_tier, verification_completeness, reason_summary, generated_at
		FROM carrier_scorecards
		WHERE carrier_account_id = $1
		ORDER BY generated_at DESC
		LIMIT 1
	`, carrierAccountID).Scan(
		&status.Scorecard.ID,
		&status.Scorecard.CarrierAccountID,
		&status.Scorecard.ScoreVersion,
		&status.Scorecard.ScoreValue,
		&status.Scorecard.ScoreBand,
		&status.Scorecard.EligibilityTier,
		&status.Scorecard.VerificationCompleteness,
		&status.Scorecard.ReasonSummary,
		&status.Scorecard.GeneratedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TrustStatus{}, ErrNoScorecard
		}
		return TrustStatus{}, fmt.Errorf("query latest scorecard: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, carrier_account_id, source_scorecard_id, input_type, source, value_numeric, value_text, effective_at
		FROM carrier_score_inputs
		WHERE source_scorecard_id = $1
		ORDER BY input_type ASC
	`, status.Scorecard.ID)
	if err != nil {
		return TrustStatus{}, fmt.Errorf("query score inputs: %w", err)
	}
	for rows.Next() {
		var input CarrierScoreInput
		if err := rows.Scan(&input.ID, &input.CarrierAccountID, &input.SourceScorecardID, &input.InputType, &input.Source, &input.ValueNumeric, &input.ValueText, &input.EffectiveAt); err != nil {
			rows.Close()
			return TrustStatus{}, fmt.Errorf("scan score input: %w", err)
		}
		status.Inputs = append(status.Inputs, input)
	}
	rows.Close()

	rows, err = r.pool.Query(ctx, `
		SELECT id, carrier_account_id, grant_type, grant_value, granted_at, revoked_at, source_scorecard_id
		FROM access_grants
		WHERE source_scorecard_id = $1
		ORDER BY grant_type ASC
	`, status.Scorecard.ID)
	if err != nil {
		return TrustStatus{}, fmt.Errorf("query access grants: %w", err)
	}
	for rows.Next() {
		var grant AccessGrant
		if err := rows.Scan(&grant.ID, &grant.CarrierAccountID, &grant.GrantType, &grant.GrantValue, &grant.GrantedAt, &grant.RevokedAt, &grant.SourceScorecardID); err != nil {
			rows.Close()
			return TrustStatus{}, fmt.Errorf("scan access grant: %w", err)
		}
		status.AccessGrants = append(status.AccessGrants, grant)
	}
	rows.Close()

	rows, err = r.pool.Query(ctx, `
		SELECT id, carrier_account_id, source_scorecard_id, signal_type, severity, status, detected_at, evidence_json, reviewed_at
		FROM fraud_signals
		WHERE source_scorecard_id = $1
		ORDER BY detected_at ASC
	`, status.Scorecard.ID)
	if err != nil {
		return TrustStatus{}, fmt.Errorf("query fraud signals: %w", err)
	}
	for rows.Next() {
		var signal FraudSignal
		if err := rows.Scan(&signal.ID, &signal.CarrierAccountID, &signal.SourceScorecardID, &signal.SignalType, &signal.Severity, &signal.Status, &signal.DetectedAt, &signal.EvidenceJSON, &signal.ReviewedAt); err != nil {
			rows.Close()
			return TrustStatus{}, fmt.Errorf("scan fraud signal: %w", err)
		}
		status.FraudSignals = append(status.FraudSignals, signal)
	}
	rows.Close()

	return status, nil
}
