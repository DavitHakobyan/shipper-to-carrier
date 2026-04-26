package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/identity"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/platform/config"
)

type authStub struct {
	registerResult identity.SessionResult
	registerErr    error
	loginResult    identity.SessionResult
	loginErr       error
	currentAccount identity.AuthenticatedAccount
	currentErr     error
	logoutErr      error
}

func (a authStub) Register(_ context.Context, _ identity.RegisterInput) (identity.SessionResult, error) {
	return a.registerResult, a.registerErr
}

func (a authStub) Login(_ context.Context, _ identity.LoginInput) (identity.SessionResult, error) {
	return a.loginResult, a.loginErr
}

func (a authStub) Current(_ context.Context, _ string) (identity.AuthenticatedAccount, error) {
	return a.currentAccount, a.currentErr
}

func (a authStub) Logout(_ context.Context, _ string) error {
	return a.logoutErr
}

func TestRegisterSetsSessionCookie(t *testing.T) {
	t.Parallel()

	handler, err := NewServer(config.Config{
		AppName:           "Test App",
		SessionCookieName: "test_session",
	}, authStub{
		registerResult: identity.SessionResult{
			Account: identity.AuthenticatedAccount{
				AccountID:   "acct_1",
				Email:       "carrier@example.com",
				DisplayName: "Carrier One",
				Role:        identity.RoleCarrier,
			},
			SessionToken:     "session-token",
			SessionExpiresAt: time.Now().Add(24 * time.Hour).UTC(),
		},
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	body, err := json.Marshal(registerRequest{
		Email:       "carrier@example.com",
		Password:    "super-secret",
		DisplayName: "Carrier One",
		Role:        identity.RoleCarrier,
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/accounts/register", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	if cookie := rec.Header().Get("Set-Cookie"); !strings.Contains(cookie, "test_session=session-token") {
		t.Fatalf("Set-Cookie = %q, want session cookie", cookie)
	}
}

func TestCurrentReturnsAuthenticatedCarrier(t *testing.T) {
	t.Parallel()

	handler, err := NewServer(config.Config{
		AppName:           "Test App",
		SessionCookieName: "test_session",
	}, authStub{
		currentAccount: identity.AuthenticatedAccount{
			AccountID:   "acct_1",
			Email:       "shipper@example.com",
			DisplayName: "Shipper One",
			Role:        identity.RoleShipper,
		},
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.AddCookie(&http.Cookie{Name: "test_session", Value: "token"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response authResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if response.Account.Role != identity.RoleShipper {
		t.Fatalf("role = %q, want %q", response.Account.Role, identity.RoleShipper)
	}
}

func TestRootServesDashboardShell(t *testing.T) {
	t.Parallel()

	handler, err := NewServer(config.Config{
		AppName:           "Test App",
		SessionCookieName: "test_session",
	}, authStub{})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	if !strings.Contains(rec.Body.String(), "Shipper to Carrier") {
		t.Fatalf("body = %q, want dashboard shell content", rec.Body.String())
	}
}
