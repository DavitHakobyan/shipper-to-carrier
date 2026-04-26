package carrieridentity

import (
	"context"
	"errors"
	"time"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/identity"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/verification"
)

type OnboardingStage string
type CarrierStatus string

const (
	CarrierStatusActive CarrierStatus = "active"

	OnboardingStageDraft              OnboardingStage = "draft"
	OnboardingStageBusinessSubmitted  OnboardingStage = "business_submitted"
	OnboardingStageAuthorityLinked    OnboardingStage = "authority_linked"
	OnboardingStageInsuranceSubmitted OnboardingStage = "insurance_submitted"
	OnboardingStageReviewPending      OnboardingStage = "review_pending"
)

var (
	ErrCarrierExists   = errors.New("carrier account already exists for this actor")
	ErrCarrierNotFound = errors.New("carrier account not found")
	ErrForbidden       = errors.New("forbidden")
)

type CarrierAccount struct {
	ID              string
	AccountID       string
	LegalName       string
	DoingBusinessAs string
	Status          CarrierStatus
	OnboardingStage OnboardingStage
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CarrierProfile struct {
	CarrierAccountID   string
	ContactPhone       string
	ContactEmail       string
	FleetSizeDeclared  int
	OperatingRegions   []string
	PreferredLoadTypes []string
}

type CarrierAddress struct {
	ID               string
	CarrierAccountID string
	AddressType      string
	Line1            string
	Line2            string
	City             string
	State            string
	PostalCode       string
	Country          string
	ValidFrom        time.Time
	CreatedAt        time.Time
}

type CarrierOwnerIdentity struct {
	ID               string
	CarrierAccountID string
	FullName         string
	Phone            string
	Email            string
	OwnershipRole    string
	IsPrimaryContact bool
	CreatedAt        time.Time
}

type CarrierAuthorityLink struct {
	ID               string
	CarrierAccountID string
	DOTNumber        string
	MCNumber         string
	USDOTStatus      string
	AuthorityType    string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type CarrierInsurancePolicy struct {
	ID                 string
	CarrierAccountID   string
	ProviderName       string
	PolicyNumberHash   string
	CoverageType       string
	EffectiveAt        time.Time
	ExpiresAt          time.Time
	VerificationStatus string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CreateCarrierInput struct {
	LegalName          string
	DoingBusinessAs    string
	ContactPhone       string
	FleetSizeDeclared  int
	OperatingRegions   []string
	PreferredLoadTypes []string
	Address            CarrierAddressInput
}

type CarrierAddressInput struct {
	AddressType string
	Line1       string
	Line2       string
	City        string
	State       string
	PostalCode  string
	Country     string
}

type AddOwnerInput struct {
	FullName         string
	Phone            string
	Email            string
	OwnershipRole    string
	IsPrimaryContact bool
}

type UpsertAuthorityInput struct {
	DOTNumber     string
	MCNumber      string
	USDOTStatus   string
	AuthorityType string
}

type AddInsuranceInput struct {
	ProviderName       string
	PolicyNumber       string
	CoverageType       string
	EffectiveAt        time.Time
	ExpiresAt          time.Time
	VerificationStatus string
}

type OnboardingStatus struct {
	Carrier             CarrierAccount
	Profile             CarrierProfile
	Addresses           []CarrierAddress
	Owners              []CarrierOwnerIdentity
	AuthorityLink       *CarrierAuthorityLink
	InsurancePolicies   []CarrierInsurancePolicy
	VerificationCase    verification.Case
	Requirements        []verification.Requirement
	MissingRequirements []verification.RequirementType
}

type Repository interface {
	CreateCarrier(ctx context.Context, actor identity.AuthenticatedAccount, input CreateCarrierInput, now time.Time) (OnboardingStatus, error)
	AddOwner(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string, input AddOwnerInput, now time.Time) (OnboardingStatus, error)
	UpsertAuthority(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string, input UpsertAuthorityInput, now time.Time) (OnboardingStatus, error)
	AddInsurance(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string, input AddInsuranceInput, now time.Time) (OnboardingStatus, error)
	GetOnboardingStatus(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string) (OnboardingStatus, error)
	GetCurrentOnboardingStatus(ctx context.Context, actor identity.AuthenticatedAccount) (OnboardingStatus, error)
}
