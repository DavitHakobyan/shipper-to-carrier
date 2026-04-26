package identity

import (
	"context"
	"testing"
	"time"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/platform/auth"
)

type memoryRepository struct {
	accountByEmail map[string]Account
	roleByAccount  map[string]Membership
	sessions       map[string]Session
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		accountByEmail: map[string]Account{},
		roleByAccount:  map[string]Membership{},
		sessions:       map[string]Session{},
	}
}

func (m *memoryRepository) CreateAccountWithMembership(_ context.Context, account Account, membership Membership) error {
	if _, exists := m.accountByEmail[account.Email]; exists {
		return ErrDuplicateEmail
	}

	m.accountByEmail[account.Email] = account
	m.roleByAccount[account.ID] = membership
	return nil
}

func (m *memoryRepository) FindAccountByEmail(_ context.Context, email string) (Account, Membership, error) {
	account, ok := m.accountByEmail[email]
	if !ok {
		return Account{}, Membership{}, ErrUnauthorized
	}

	return account, m.roleByAccount[account.ID], nil
}

func (m *memoryRepository) CreateSession(_ context.Context, session Session) error {
	m.sessions[session.TokenHash] = session
	return nil
}

func (m *memoryRepository) FindAuthenticatedByTokenHash(_ context.Context, tokenHash string) (AuthenticatedAccount, Session, error) {
	session, ok := m.sessions[tokenHash]
	if !ok {
		return AuthenticatedAccount{}, Session{}, ErrUnauthorized
	}

	for _, account := range m.accountByEmail {
		if account.ID != session.AccountID {
			continue
		}

		membership := m.roleByAccount[account.ID]
		return AuthenticatedAccount{
			AccountID:   account.ID,
			Email:       account.Email,
			DisplayName: account.DisplayName,
			Role:        membership.Role,
		}, session, nil
	}

	return AuthenticatedAccount{}, Session{}, ErrUnauthorized
}

func (m *memoryRepository) TouchSession(_ context.Context, sessionID string, seenAt time.Time) error {
	for tokenHash, session := range m.sessions {
		if session.ID != sessionID {
			continue
		}

		session.LastSeenAt = seenAt
		m.sessions[tokenHash] = session
		return nil
	}

	return ErrUnauthorized
}

func (m *memoryRepository) DeleteSessionByTokenHash(_ context.Context, tokenHash string) error {
	delete(m.sessions, tokenHash)
	return nil
}

func TestRegisterCreatesShipperSession(t *testing.T) {
	t.Parallel()

	repo := newMemoryRepository()
	service := NewService(repo, 12*time.Hour)

	result, err := service.Register(context.Background(), RegisterInput{
		Email:       "Shipper@Example.com",
		Password:    "super-secret",
		DisplayName: "Shipper One",
		Role:        RoleShipper,
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if result.Account.Role != RoleShipper {
		t.Fatalf("role = %q, want %q", result.Account.Role, RoleShipper)
	}

	if result.SessionToken == "" {
		t.Fatal("SessionToken = empty, want non-empty")
	}

	account, ok := repo.accountByEmail["shipper@example.com"]
	if !ok {
		t.Fatal("account not stored by normalized email")
	}

	if account.DisplayName != "Shipper One" {
		t.Fatalf("DisplayName = %q, want %q", account.DisplayName, "Shipper One")
	}
}

func TestLoginAndCurrentReturnCarrier(t *testing.T) {
	t.Parallel()

	repo := newMemoryRepository()
	passwordHash, err := auth.HashPassword("super-secret")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	account := Account{
		ID:           "acct_1",
		Email:        "carrier@example.com",
		DisplayName:  "Carrier One",
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	repo.accountByEmail[account.Email] = account
	repo.roleByAccount[account.ID] = Membership{
		ID:        "membership_1",
		AccountID: account.ID,
		Role:      RoleCarrier,
		CreatedAt: time.Now().UTC(),
	}

	service := NewService(repo, 12*time.Hour)
	loginResult, err := service.Login(context.Background(), LoginInput{
		Email:    account.Email,
		Password: "super-secret",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	current, err := service.Current(context.Background(), loginResult.SessionToken)
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	if current.Role != RoleCarrier {
		t.Fatalf("role = %q, want %q", current.Role, RoleCarrier)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	t.Parallel()

	repo := newMemoryRepository()
	service := NewService(repo, 12*time.Hour)

	input := RegisterInput{
		Email:       "carrier@example.com",
		Password:    "super-secret",
		DisplayName: "Carrier One",
		Role:        RoleCarrier,
	}

	if _, err := service.Register(context.Background(), input); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	if _, err := service.Register(context.Background(), input); err != ErrDuplicateEmail {
		t.Fatalf("second Register() error = %v, want %v", err, ErrDuplicateEmail)
	}
}
