package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateAccountWithMembership(ctx context.Context, account Account, membership Membership) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, `
		INSERT INTO accounts (id, email, display_name, password_hash, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, account.ID, account.Email, account.DisplayName, account.PasswordHash, account.CreatedAt, account.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrDuplicateEmail
		}

		return fmt.Errorf("insert account: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO memberships (id, account_id, role, created_at)
		VALUES ($1, $2, $3, $4)
	`, membership.ID, membership.AccountID, membership.Role, membership.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert membership: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresRepository) FindAccountByEmail(ctx context.Context, email string) (Account, Membership, error) {
	var account Account
	var membership Membership

	err := r.pool.QueryRow(ctx, `
		SELECT
			a.id,
			a.email,
			a.display_name,
			a.password_hash,
			a.created_at,
			a.updated_at,
			m.id,
			m.role,
			m.created_at
		FROM accounts a
		JOIN memberships m ON m.account_id = a.id
		WHERE a.email = $1
		LIMIT 1
	`, email).Scan(
		&account.ID,
		&account.Email,
		&account.DisplayName,
		&account.PasswordHash,
		&account.CreatedAt,
		&account.UpdatedAt,
		&membership.ID,
		&membership.Role,
		&membership.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Account{}, Membership{}, ErrUnauthorized
		}

		return Account{}, Membership{}, fmt.Errorf("query account: %w", err)
	}

	membership.AccountID = account.ID
	return account, membership, nil
}

func (r *PostgresRepository) CreateSession(ctx context.Context, session Session) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sessions (id, account_id, token_hash, expires_at, created_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, session.ID, session.AccountID, session.TokenHash, session.ExpiresAt, session.CreatedAt, session.LastSeenAt)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	return nil
}

func (r *PostgresRepository) FindAuthenticatedByTokenHash(ctx context.Context, tokenHash string) (AuthenticatedAccount, Session, error) {
	var account AuthenticatedAccount
	var session Session

	err := r.pool.QueryRow(ctx, `
		SELECT
			a.id,
			a.email,
			a.display_name,
			m.role,
			s.id,
			s.account_id,
			s.token_hash,
			s.expires_at,
			s.created_at,
			s.last_seen_at
		FROM sessions s
		JOIN accounts a ON a.id = s.account_id
		JOIN memberships m ON m.account_id = a.id
		WHERE s.token_hash = $1
		  AND s.expires_at > $2
		LIMIT 1
	`, tokenHash, time.Now().UTC()).Scan(
		&account.AccountID,
		&account.Email,
		&account.DisplayName,
		&account.Role,
		&session.ID,
		&session.AccountID,
		&session.TokenHash,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.LastSeenAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AuthenticatedAccount{}, Session{}, ErrUnauthorized
		}

		return AuthenticatedAccount{}, Session{}, fmt.Errorf("query session: %w", err)
	}

	return account, session, nil
}

func (r *PostgresRepository) TouchSession(ctx context.Context, sessionID string, seenAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE sessions
		SET last_seen_at = $2
		WHERE id = $1
	`, sessionID, seenAt)
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}

	return nil
}

func (r *PostgresRepository) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM sessions
		WHERE token_hash = $1
	`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}

	return false
}
