package externalevidence

import (
	"context"
	"errors"
	"time"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/carrieridentity"
)

type Source string
type SnapshotStatus string

const (
	SourceFMCSA Source = "fmcsa"

	SnapshotStatusMatched     SnapshotStatus = "matched"
	SnapshotStatusMismatch    SnapshotStatus = "mismatch"
	SnapshotStatusUnavailable SnapshotStatus = "unavailable"
)

var (
	ErrNoAuthorityLink = errors.New("carrier authority link is required")
	ErrNoSnapshot      = errors.New("external record snapshot not found")
)

type ExternalRecordSnapshot struct {
	ID               string
	CarrierAccountID string
	Source           Source
	SourceKey        string
	FetchedAt        time.Time
	Status           SnapshotStatus
	PayloadJSON      []byte
	Checksum         string
}

type FMCSARegistrationRecord struct {
	SnapshotID      string
	DOTNumber       string
	LegalName       string
	Address         string
	EntityType      string
	AuthorityStatus string
	OutOfService    bool
	OperatingStatus string
}

type FMCSASafetyRecord struct {
	SnapshotID          string
	SafetyRating        string
	CrashCount          int
	InspectionCount     int
	OOSRate             float64
	IncidentWindowStart time.Time
	IncidentWindowEnd   time.Time
}

type FMCSAData struct {
	Snapshot     ExternalRecordSnapshot
	Registration FMCSARegistrationRecord
	Safety       FMCSASafetyRecord
}

type FMCSARequest struct {
	CarrierAccountID string
	LegalName        string
	DOTNumber        string
	MCNumber         string
	USDOTStatus      string
	AuthorityType    string
	AddressLine1     string
	City             string
	State            string
}

type Provider interface {
	FetchFMCSA(ctx context.Context, input FMCSARequest) (FMCSAData, error)
}

type Repository interface {
	SaveFMCSA(ctx context.Context, data FMCSAData) error
	LatestFMCSA(ctx context.Context, carrierAccountID string) (FMCSAData, error)
}

type Service struct {
	repo     Repository
	provider Provider
}

func NewService(repo Repository, provider Provider) *Service {
	return &Service{
		repo:     repo,
		provider: provider,
	}
}

func (s *Service) RefreshFMCSA(ctx context.Context, status carrieridentity.OnboardingStatus) (FMCSAData, error) {
	if status.AuthorityLink == nil {
		return FMCSAData{}, ErrNoAuthorityLink
	}

	request := FMCSARequest{
		CarrierAccountID: status.Carrier.ID,
		LegalName:        status.Carrier.LegalName,
		DOTNumber:        status.AuthorityLink.DOTNumber,
		MCNumber:         status.AuthorityLink.MCNumber,
		USDOTStatus:      status.AuthorityLink.USDOTStatus,
		AuthorityType:    status.AuthorityLink.AuthorityType,
	}
	if len(status.Addresses) > 0 {
		request.AddressLine1 = status.Addresses[0].Line1
		request.City = status.Addresses[0].City
		request.State = status.Addresses[0].State
	}

	data, err := s.provider.FetchFMCSA(ctx, request)
	if err != nil {
		return FMCSAData{}, err
	}
	if err := s.repo.SaveFMCSA(ctx, data); err != nil {
		return FMCSAData{}, err
	}

	return data, nil
}

func (s *Service) LatestFMCSA(ctx context.Context, carrierAccountID string) (FMCSAData, error) {
	return s.repo.LatestFMCSA(ctx, carrierAccountID)
}
