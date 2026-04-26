package verification

import "time"

type CaseType string
type CaseStatus string
type RequirementType string
type RequirementStatus string

const (
	CaseTypeOnboarding    CaseType   = "onboarding"
	CaseStatusOpen        CaseStatus = "open"
	CaseStatusReviewReady CaseStatus = "review_pending"

	RequirementTypeBusinessProfile RequirementType = "business_profile"
	RequirementTypeOwnerIdentity   RequirementType = "owner_identity"
	RequirementTypeOperatingAddr   RequirementType = "operating_address"
	RequirementTypeAuthorityLink   RequirementType = "authority_link"
	RequirementTypeInsurancePolicy RequirementType = "insurance_policy"

	RequirementStatusPending   RequirementStatus = "pending"
	RequirementStatusSatisfied RequirementStatus = "satisfied"
)

type Case struct {
	ID               string
	CarrierAccountID string
	CaseType         CaseType
	Status           CaseStatus
	OpenedAt         time.Time
	ClosedAt         *time.Time
	AssignedAdminID  *string
}

type Requirement struct {
	ID                 string
	VerificationCaseID string
	RequirementType    RequirementType
	Status             RequirementStatus
	SatisfiedAt        *time.Time
	Notes              string
}

func DefaultRequirementTypes() []RequirementType {
	return []RequirementType{
		RequirementTypeBusinessProfile,
		RequirementTypeOwnerIdentity,
		RequirementTypeOperatingAddr,
		RequirementTypeAuthorityLink,
		RequirementTypeInsurancePolicy,
	}
}
