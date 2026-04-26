package carrieridentity

import (
	"context"
	"testing"
	"time"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/identity"
)

type repoStub struct {
	createCalled bool
}

func (r *repoStub) CreateCarrier(_ context.Context, _ identity.AuthenticatedAccount, _ CreateCarrierInput, _ time.Time) (OnboardingStatus, error) {
	r.createCalled = true
	return OnboardingStatus{}, nil
}

func (r *repoStub) AddOwner(_ context.Context, _ identity.AuthenticatedAccount, _ string, _ AddOwnerInput, _ time.Time) (OnboardingStatus, error) {
	return OnboardingStatus{}, nil
}

func (r *repoStub) UpsertAuthority(_ context.Context, _ identity.AuthenticatedAccount, _ string, _ UpsertAuthorityInput, _ time.Time) (OnboardingStatus, error) {
	return OnboardingStatus{}, nil
}

func (r *repoStub) AddInsurance(_ context.Context, _ identity.AuthenticatedAccount, _ string, _ AddInsuranceInput, _ time.Time) (OnboardingStatus, error) {
	return OnboardingStatus{}, nil
}

func (r *repoStub) GetOnboardingStatus(_ context.Context, _ identity.AuthenticatedAccount, _ string) (OnboardingStatus, error) {
	return OnboardingStatus{}, nil
}

func (r *repoStub) GetCurrentOnboardingStatus(_ context.Context, _ identity.AuthenticatedAccount) (OnboardingStatus, error) {
	return OnboardingStatus{}, nil
}

func TestCreateCarrierRejectsShipper(t *testing.T) {
	t.Parallel()

	service := NewService(&repoStub{})
	_, err := service.CreateCarrier(context.Background(), identity.AuthenticatedAccount{
		AccountID: "acct_1",
		Email:     "shipper@example.com",
		Role:      identity.RoleShipper,
	}, CreateCarrierInput{
		LegalName:    "Carrier LLC",
		ContactPhone: "555-555-5555",
		Address: CarrierAddressInput{
			Line1:      "1 Main St",
			City:       "Phoenix",
			State:      "AZ",
			PostalCode: "85001",
			Country:    "US",
		},
	})

	if err != ErrForbidden {
		t.Fatalf("CreateCarrier() error = %v, want %v", err, ErrForbidden)
	}
}
