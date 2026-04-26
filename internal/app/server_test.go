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

	"github.com/DavitHakobyan/shipper-to-carrier/internal/carrieridentity"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/externalevidence"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/identity"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/platform/config"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/trust"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/verification"
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

type carrierStub struct {
	createStatus   carrieridentity.OnboardingStatus
	createErr      error
	currentStatus  carrieridentity.OnboardingStatus
	ownerCarrierID string
}

func (c carrierStub) CreateCarrier(_ context.Context, _ identity.AuthenticatedAccount, _ carrieridentity.CreateCarrierInput) (carrieridentity.OnboardingStatus, error) {
	return c.createStatus, c.createErr
}

func (c carrierStub) AddOwner(_ context.Context, _ identity.AuthenticatedAccount, _ string, _ carrieridentity.AddOwnerInput) (carrieridentity.OnboardingStatus, error) {
	return c.currentStatus, nil
}

func (c carrierStub) UpsertAuthority(_ context.Context, _ identity.AuthenticatedAccount, _ string, _ carrieridentity.UpsertAuthorityInput) (carrieridentity.OnboardingStatus, error) {
	return carrieridentity.OnboardingStatus{}, nil
}

func (c carrierStub) AddInsurance(_ context.Context, _ identity.AuthenticatedAccount, _ string, _ carrieridentity.AddInsuranceInput) (carrieridentity.OnboardingStatus, error) {
	return carrieridentity.OnboardingStatus{}, nil
}

func (c carrierStub) GetOnboardingStatus(_ context.Context, _ identity.AuthenticatedAccount, _ string) (carrieridentity.OnboardingStatus, error) {
	return carrieridentity.OnboardingStatus{}, nil
}

func (c carrierStub) GetCurrentOnboardingStatus(_ context.Context, _ identity.AuthenticatedAccount) (carrieridentity.OnboardingStatus, error) {
	return c.currentStatus, nil
}

type evidenceStub struct {
	latest externalevidence.FMCSAData
}

func (e evidenceStub) RefreshFMCSA(_ context.Context, _ carrieridentity.OnboardingStatus) (externalevidence.FMCSAData, error) {
	return e.latest, nil
}

func (e evidenceStub) LatestFMCSA(_ context.Context, _ string) (externalevidence.FMCSAData, error) {
	return e.latest, nil
}

type trustStub struct {
	latest trust.TrustStatus
}

func (t trustStub) Evaluate(_ context.Context, _ carrieridentity.OnboardingStatus, _ externalevidence.FMCSAData) (trust.TrustStatus, error) {
	return t.latest, nil
}

func (t trustStub) LatestTrust(_ context.Context, _ string) (trust.TrustStatus, error) {
	return t.latest, nil
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
	}, carrierStub{}, evidenceStub{}, trustStub{})
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
	}, carrierStub{}, evidenceStub{}, trustStub{})
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
	}, authStub{}, carrierStub{}, evidenceStub{}, trustStub{})
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

func TestCreateCarrierReturnsOnboardingStatus(t *testing.T) {
	t.Parallel()

	handler, err := NewServer(config.Config{
		AppName:           "Test App",
		SessionCookieName: "test_session",
	}, authStub{
		currentAccount: identity.AuthenticatedAccount{
			AccountID:   "acct_1",
			Email:       "carrier@example.com",
			DisplayName: "Carrier One",
			Role:        identity.RoleCarrier,
		},
	}, carrierStub{
		createStatus: carrieridentity.OnboardingStatus{
			Carrier: carrieridentity.CarrierAccount{
				ID:              "carrier_1",
				LegalName:       "Carrier LLC",
				Status:          carrieridentity.CarrierStatusActive,
				OnboardingStage: carrieridentity.OnboardingStageBusinessSubmitted,
			},
			Profile: carrieridentity.CarrierProfile{
				ContactPhone: "555-555-5555",
				ContactEmail: "carrier@example.com",
			},
			VerificationCase: verification.Case{
				ID:       "case_1",
				CaseType: verification.CaseTypeOnboarding,
				Status:   verification.CaseStatusOpen,
				OpenedAt: time.Now().UTC(),
			},
		},
	}, evidenceStub{}, trustStub{})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	body, err := json.Marshal(createCarrierRequest{
		LegalName:    "Carrier LLC",
		ContactPhone: "555-555-5555",
		Address: carrierAddressRequest{
			Line1:      "1 Main St",
			City:       "Phoenix",
			State:      "AZ",
			PostalCode: "85001",
			Country:    "US",
		},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/carriers", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "test_session", Value: "token"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var response onboardingStatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if response.Carrier.ID != "carrier_1" {
		t.Fatalf("carrier id = %q, want %q", response.Carrier.ID, "carrier_1")
	}
}

func TestIntelligenceReturnsScorecard(t *testing.T) {
	t.Parallel()

	handler, err := NewServer(config.Config{
		AppName:           "Test App",
		SessionCookieName: "test_session",
	}, authStub{
		currentAccount: identity.AuthenticatedAccount{
			AccountID:   "acct_1",
			Email:       "carrier@example.com",
			DisplayName: "Carrier One",
			Role:        identity.RoleCarrier,
		},
	}, carrierStub{
		currentStatus: carrieridentity.OnboardingStatus{
			Carrier: carrieridentity.CarrierAccount{
				ID: "carrier_1",
			},
		},
	}, evidenceStub{
		latest: externalevidence.FMCSAData{
			Snapshot: externalevidence.ExternalRecordSnapshot{
				Status:    externalevidence.SnapshotStatusMatched,
				SourceKey: "123456",
				FetchedAt: time.Now().UTC(),
			},
			Registration: externalevidence.FMCSARegistrationRecord{
				LegalName:       "Carrier LLC",
				AuthorityStatus: "active",
				OperatingStatus: "active",
			},
			Safety: externalevidence.FMCSASafetyRecord{
				SafetyRating:    "satisfactory",
				CrashCount:      1,
				InspectionCount: 12,
			},
		},
	}, trustStub{
		latest: trust.TrustStatus{
			Scorecard: trust.CarrierScorecard{
				EligibilityTier: trust.EligibilityTierTier0,
				ScoreValue:      80,
				ScoreBand:       trust.ScoreBandHigh,
				GeneratedAt:     time.Now().UTC(),
			},
			AccessGrants: []trust.AccessGrant{{GrantType: "load_value_cap", GrantValue: "2500"}},
		},
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/carriers/current/intelligence", nil)
	req.SetPathValue("carrierID", "current")
	req.AddCookie(&http.Cookie{Name: "test_session", Value: "token"})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var response intelligenceResponse
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if response.Scorecard.EligibilityTier != trust.EligibilityTierTier0 {
		t.Fatalf("eligibility = %q, want %q", response.Scorecard.EligibilityTier, trust.EligibilityTierTier0)
	}
}
