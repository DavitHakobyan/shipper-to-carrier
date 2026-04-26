package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/platform/auth"
	"github.com/google/uuid"
)

type Service struct {
	repo       Repository
	now        func() time.Time
	sessionTTL time.Duration
}

func NewService(repo Repository, sessionTTL time.Duration) *Service {
	return &Service{
		repo:       repo,
		now:        time.Now,
		sessionTTL: sessionTTL,
	}
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (SessionResult, error) {
	if err := validateRegisterInput(input); err != nil {
		return SessionResult{}, err
	}

	now := s.now().UTC()
	account := Account{
		ID:           uuid.NewString(),
		Email:        normalizeEmail(input.Email),
		DisplayName:  strings.TrimSpace(input.DisplayName),
		PasswordHash: "",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		return SessionResult{}, fmt.Errorf("hash password: %w", err)
	}
	account.PasswordHash = passwordHash

	membership := Membership{
		ID:        uuid.NewString(),
		AccountID: account.ID,
		Role:      input.Role,
		CreatedAt: now,
	}

	if err := s.repo.CreateAccountWithMembership(ctx, account, membership); err != nil {
		return SessionResult{}, err
	}

	return s.createSession(ctx, account, membership.Role, now)
}

func (s *Service) Login(ctx context.Context, input LoginInput) (SessionResult, error) {
	email := normalizeEmail(input.Email)
	if email == "" || strings.TrimSpace(input.Password) == "" {
		return SessionResult{}, errors.New("email and password are required")
	}

	account, membership, err := s.repo.FindAccountByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			return SessionResult{}, ErrInvalidCredentials
		}

		return SessionResult{}, err
	}

	if err := auth.ComparePassword(account.PasswordHash, input.Password); err != nil {
		return SessionResult{}, ErrInvalidCredentials
	}

	return s.createSession(ctx, account, membership.Role, s.now().UTC())
}

func (s *Service) Current(ctx context.Context, sessionToken string) (AuthenticatedAccount, error) {
	if strings.TrimSpace(sessionToken) == "" {
		return AuthenticatedAccount{}, ErrUnauthorized
	}

	tokenHash := auth.HashToken(sessionToken)
	account, session, err := s.repo.FindAuthenticatedByTokenHash(ctx, tokenHash)
	if err != nil {
		return AuthenticatedAccount{}, err
	}

	if err := s.repo.TouchSession(ctx, session.ID, s.now().UTC()); err != nil {
		return AuthenticatedAccount{}, err
	}

	return account, nil
}

func (s *Service) Logout(ctx context.Context, sessionToken string) error {
	if strings.TrimSpace(sessionToken) == "" {
		return ErrUnauthorized
	}

	return s.repo.DeleteSessionByTokenHash(ctx, auth.HashToken(sessionToken))
}

func (s *Service) createSession(ctx context.Context, account Account, role Role, now time.Time) (SessionResult, error) {
	sessionToken, tokenHash, err := auth.NewSessionToken()
	if err != nil {
		return SessionResult{}, fmt.Errorf("create session token: %w", err)
	}

	session := Session{
		ID:         uuid.NewString(),
		AccountID:  account.ID,
		TokenHash:  tokenHash,
		ExpiresAt:  now.Add(s.sessionTTL),
		CreatedAt:  now,
		LastSeenAt: now,
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return SessionResult{}, err
	}

	return SessionResult{
		Account: AuthenticatedAccount{
			AccountID:   account.ID,
			Email:       account.Email,
			DisplayName: account.DisplayName,
			Role:        role,
		},
		SessionToken:     sessionToken,
		SessionExpiresAt: session.ExpiresAt,
	}, nil
}

func validateRegisterInput(input RegisterInput) error {
	switch {
	case normalizeEmail(input.Email) == "":
		return errors.New("email is required")
	case strings.TrimSpace(input.DisplayName) == "":
		return errors.New("display name is required")
	case len(strings.TrimSpace(input.Password)) < 8:
		return errors.New("password must be at least 8 characters")
	case !input.Role.Valid() || input.Role == RoleAdmin:
		return errors.New("role must be carrier or shipper")
	default:
		return nil
	}
}
