package carrieridentity

import (
	"testing"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/verification"
)

func TestDeriveOnboardingStage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		facts OnboardingFacts
		want  OnboardingStage
	}{
		{
			name:  "draft",
			facts: OnboardingFacts{},
			want:  OnboardingStageDraft,
		},
		{
			name:  "business submitted",
			facts: OnboardingFacts{HasBusinessProfile: true, HasOperatingAddr: true},
			want:  OnboardingStageBusinessSubmitted,
		},
		{
			name:  "authority linked",
			facts: OnboardingFacts{HasBusinessProfile: true, HasOperatingAddr: true, HasOwnerIdentity: true, HasAuthorityLink: true},
			want:  OnboardingStageAuthorityLinked,
		},
		{
			name:  "review pending",
			facts: OnboardingFacts{HasBusinessProfile: true, HasOperatingAddr: true, HasOwnerIdentity: true, HasAuthorityLink: true, HasInsurance: true},
			want:  OnboardingStageReviewPending,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := DeriveOnboardingStage(test.facts); got != test.want {
				t.Fatalf("DeriveOnboardingStage() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestMissingRequirements(t *testing.T) {
	t.Parallel()

	missing := MissingRequirements(OnboardingFacts{
		HasBusinessProfile: true,
		HasOperatingAddr:   true,
	})

	want := []verification.RequirementType{
		verification.RequirementTypeOwnerIdentity,
		verification.RequirementTypeAuthorityLink,
		verification.RequirementTypeInsurancePolicy,
	}

	if len(missing) != len(want) {
		t.Fatalf("len(missing) = %d, want %d", len(missing), len(want))
	}

	for index, requirement := range want {
		if missing[index] != requirement {
			t.Fatalf("missing[%d] = %q, want %q", index, missing[index], requirement)
		}
	}
}
