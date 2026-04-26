package externalevidence

import (
	"context"
	"testing"
	"time"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/carrieridentity"
)

type repoStub struct {
	saved FMCSAData
}

func (r *repoStub) SaveFMCSA(_ context.Context, data FMCSAData) error {
	r.saved = data
	return nil
}

func (r *repoStub) LatestFMCSA(_ context.Context, _ string) (FMCSAData, error) {
	return r.saved, nil
}

type providerStub struct {
	data FMCSAData
}

func (p providerStub) FetchFMCSA(_ context.Context, _ FMCSARequest) (FMCSAData, error) {
	return p.data, nil
}

func TestRefreshFMCSARequiresAuthority(t *testing.T) {
	t.Parallel()

	service := NewService(&repoStub{}, providerStub{})
	_, err := service.RefreshFMCSA(context.Background(), carrieridentity.OnboardingStatus{})
	if err != ErrNoAuthorityLink {
		t.Fatalf("RefreshFMCSA() error = %v, want %v", err, ErrNoAuthorityLink)
	}
}

func TestMockProviderReturnsSnapshot(t *testing.T) {
	t.Parallel()

	provider := NewMockProvider()
	data, err := provider.FetchFMCSA(context.Background(), FMCSARequest{
		CarrierAccountID: "carrier_1",
		LegalName:        "Carrier LLC",
		DOTNumber:        "1234567",
		USDOTStatus:      "active",
		AddressLine1:     "1 Main St",
		City:             "Phoenix",
		State:            "AZ",
	})
	if err != nil {
		t.Fatalf("FetchFMCSA() error = %v", err)
	}

	if data.Snapshot.Source != SourceFMCSA {
		t.Fatalf("Snapshot.Source = %q, want %q", data.Snapshot.Source, SourceFMCSA)
	}

	if data.Snapshot.FetchedAt.Before(time.Now().UTC().Add(-time.Minute)) {
		t.Fatalf("Snapshot.FetchedAt = %s, want recent timestamp", data.Snapshot.FetchedAt)
	}
}
