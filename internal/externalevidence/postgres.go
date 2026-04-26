package externalevidence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) SaveFMCSA(ctx context.Context, data FMCSAData) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO external_record_snapshots (id, carrier_account_id, source, source_key, fetched_at, status, payload_json, checksum)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, data.Snapshot.ID, data.Snapshot.CarrierAccountID, data.Snapshot.Source, data.Snapshot.SourceKey, data.Snapshot.FetchedAt, data.Snapshot.Status, data.Snapshot.PayloadJSON, data.Snapshot.Checksum); err != nil {
		return fmt.Errorf("insert external snapshot: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO fmcsa_registration_records (snapshot_id, dot_number, legal_name, address, entity_type, authority_status, out_of_service, operating_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, data.Registration.SnapshotID, data.Registration.DOTNumber, data.Registration.LegalName, data.Registration.Address, data.Registration.EntityType, data.Registration.AuthorityStatus, data.Registration.OutOfService, data.Registration.OperatingStatus); err != nil {
		return fmt.Errorf("insert fmcsa registration: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO fmcsa_safety_records (snapshot_id, safety_rating, crash_count, inspection_count, oos_rate, incident_window_start, incident_window_end)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, data.Safety.SnapshotID, data.Safety.SafetyRating, data.Safety.CrashCount, data.Safety.InspectionCount, data.Safety.OOSRate, data.Safety.IncidentWindowStart, data.Safety.IncidentWindowEnd); err != nil {
		return fmt.Errorf("insert fmcsa safety: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresRepository) LatestFMCSA(ctx context.Context, carrierAccountID string) (FMCSAData, error) {
	var data FMCSAData

	err := r.pool.QueryRow(ctx, `
		SELECT
			s.id,
			s.carrier_account_id,
			s.source,
			s.source_key,
			s.fetched_at,
			s.status,
			s.payload_json,
			s.checksum,
			r.dot_number,
			r.legal_name,
			r.address,
			r.entity_type,
			r.authority_status,
			r.out_of_service,
			r.operating_status,
			f.safety_rating,
			f.crash_count,
			f.inspection_count,
			f.oos_rate,
			f.incident_window_start,
			f.incident_window_end
		FROM external_record_snapshots s
		JOIN fmcsa_registration_records r ON r.snapshot_id = s.id
		JOIN fmcsa_safety_records f ON f.snapshot_id = s.id
		WHERE s.carrier_account_id = $1
		  AND s.source = $2
		ORDER BY s.fetched_at DESC
		LIMIT 1
	`, carrierAccountID, SourceFMCSA).Scan(
		&data.Snapshot.ID,
		&data.Snapshot.CarrierAccountID,
		&data.Snapshot.Source,
		&data.Snapshot.SourceKey,
		&data.Snapshot.FetchedAt,
		&data.Snapshot.Status,
		&data.Snapshot.PayloadJSON,
		&data.Snapshot.Checksum,
		&data.Registration.DOTNumber,
		&data.Registration.LegalName,
		&data.Registration.Address,
		&data.Registration.EntityType,
		&data.Registration.AuthorityStatus,
		&data.Registration.OutOfService,
		&data.Registration.OperatingStatus,
		&data.Safety.SafetyRating,
		&data.Safety.CrashCount,
		&data.Safety.InspectionCount,
		&data.Safety.OOSRate,
		&data.Safety.IncidentWindowStart,
		&data.Safety.IncidentWindowEnd,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FMCSAData{}, ErrNoSnapshot
		}
		return FMCSAData{}, fmt.Errorf("query latest fmcsa snapshot: %w", err)
	}

	data.Registration.SnapshotID = data.Snapshot.ID
	data.Safety.SnapshotID = data.Snapshot.ID
	return data, nil
}

func staleAfter(fetchedAt time.Time, freshnessWindow time.Duration) bool {
	return time.Now().UTC().After(fetchedAt.Add(freshnessWindow))
}
