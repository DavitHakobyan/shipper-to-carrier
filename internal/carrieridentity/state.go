package carrieridentity

import "github.com/DavitHakobyan/shipper-to-carrier/internal/verification"

type OnboardingFacts struct {
	HasBusinessProfile bool
	HasOperatingAddr   bool
	HasOwnerIdentity   bool
	HasAuthorityLink   bool
	HasInsurance       bool
}

func DeriveOnboardingStage(facts OnboardingFacts) OnboardingStage {
	stage := OnboardingStageDraft

	if facts.HasBusinessProfile && facts.HasOperatingAddr {
		stage = OnboardingStageBusinessSubmitted
	}

	if facts.HasAuthorityLink {
		stage = OnboardingStageAuthorityLinked
	}

	if facts.HasInsurance {
		stage = OnboardingStageInsuranceSubmitted
	}

	if allRequirementsSatisfied(facts) {
		stage = OnboardingStageReviewPending
	}

	return stage
}

func MissingRequirements(facts OnboardingFacts) []verification.RequirementType {
	missing := make([]verification.RequirementType, 0, 5)

	if !facts.HasBusinessProfile {
		missing = append(missing, verification.RequirementTypeBusinessProfile)
	}
	if !facts.HasOwnerIdentity {
		missing = append(missing, verification.RequirementTypeOwnerIdentity)
	}
	if !facts.HasOperatingAddr {
		missing = append(missing, verification.RequirementTypeOperatingAddr)
	}
	if !facts.HasAuthorityLink {
		missing = append(missing, verification.RequirementTypeAuthorityLink)
	}
	if !facts.HasInsurance {
		missing = append(missing, verification.RequirementTypeInsurancePolicy)
	}

	return missing
}

func allRequirementsSatisfied(facts OnboardingFacts) bool {
	return facts.HasBusinessProfile &&
		facts.HasOperatingAddr &&
		facts.HasOwnerIdentity &&
		facts.HasAuthorityLink &&
		facts.HasInsurance
}
