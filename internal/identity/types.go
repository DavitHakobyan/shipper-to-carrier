package identity

import (
	"context"
	"errors"
	"strings"
	"time"
)

type Role string

const (
	RoleCarrier Role = "carrier"
	RoleShipper Role = "shipper"
	RoleAdmin   Role = "admin"
)

var (
	ErrDuplicateEmail     = errors.New("account email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthorized       = errors.New("unauthorized")
)

type Account struct {
	ID           string
	Email        string
	DisplayName  string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Membership struct {
	ID        string
	AccountID string
	Role      Role
	CreatedAt time.Time
}

type Session struct {
	ID         string
	AccountID  string
	TokenHash  string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	LastSeenAt time.Time
}

type AuthenticatedAccount struct {
	AccountID   string
	Email       string
	DisplayName string
	Role        Role
}

type RegisterInput struct {
	Email       string
	Password    string
	DisplayName string
	Role        Role
}

type LoginInput struct {
	Email    string
	Password string
}

type SessionResult struct {
	Account          AuthenticatedAccount
	SessionToken     string
	SessionExpiresAt time.Time
}

type Repository interface {
	CreateAccountWithMembership(ctx context.Context, account Account, membership Membership) error
	FindAccountByEmail(ctx context.Context, email string) (Account, Membership, error)
	CreateSession(ctx context.Context, session Session) error
	FindAuthenticatedByTokenHash(ctx context.Context, tokenHash string) (AuthenticatedAccount, Session, error)
	TouchSession(ctx context.Context, sessionID string, seenAt time.Time) error
	DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error
}

func (r Role) Valid() bool {
	switch r {
	case RoleCarrier, RoleShipper, RoleAdmin:
		return true
	default:
		return false
	}
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}
