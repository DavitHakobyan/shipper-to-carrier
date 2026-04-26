package trust

import (
	"context"
	"testing"
	"time"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/carrieridentity"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/externalevidence"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/verification"
)

type repoStub struct {
	saved Evaluation
}

func (r *repoStub) SaveEvaluation(_ context.Context, evaluation Evaluation) error {
	r.saved = evaluation
	return nil
}

func (r *repoStub) LatestTrust(_ context.Context, _ string) (TrustStatus, error) {
	return TrustStatus{Scorecard: r.saved.Scorecard}, nil
}

func TestEvaluateReturnsTier0WhenMatchedAndComplete(t *testing.T) {
	t.Parallel()

	repo := &repoStub{}
	service := NewService(repo)
	status, err := service.Evaluate(context.Background(), carrieridentity.OnboardingStatus{
		Carrier: carrieridentity.CarrierAccount{
			ID:              "carrier_1",
			OnboardingStage: carrieridentity.OnboardingStageReviewPending,
		},
		Requirements: []verification.Requirement{
			{Status: verification.RequirementStatusSatisfied},
			{Status: verification.RequirementStatusSatisfied},
			{Status: verification.RequirementStatusSatisfied},
			{Status: verification.RequirementStatusSatisfied},
			{Status: verification.RequirementStatusSatisfied},
		},
	}, externalevidence.FMCSAData{
		Snapshot: externalevidence.ExternalRecordSnapshot{
			Status: externalevidence.SnapshotStatusMatched,
		},
		Registration: externalevidence.FMCSARegistrationRecord{
			AuthorityStatus: "active",
		},
		Safety: externalevidence.FMCSASafetyRecord{
			SafetyRating:    "satisfactory",
			CrashCount:      0,
			InspectionCount: 12,
			OOSRate:         0.02,
		},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	if status.Scorecard.EligibilityTier != EligibilityTierTier0 {
		t.Fatalf("EligibilityTier = %q, want %q", status.Scorecard.EligibilityTier, EligibilityTierTier0)
	}

	if len(status.AccessGrants) == 0 {
		t.Fatal("AccessGrants = empty, want grants")
	}
}

func TestEvaluateRestrictsMismatch(t *testing.T) {
	t.Parallel()

	service := NewService(&repoStub{})
	status, err := service.Evaluate(context.Background(), carrieridentity.OnboardingStatus{
		Carrier: carrieridentity.CarrierAccount{
			ID:              "carrier_1",
			LegalName:       "Carrier LLC",
			OnboardingStage: carrieridentity.OnboardingStageReviewPending,
		},
		Requirements: []verification.Requirement{
			{Status: verification.RequirementStatusSatisfied},
		},
	}, externalevidence.FMCSAData{
		Snapshot: externalevidence.ExternalRecordSnapshot{
			Status: externalevidence.SnapshotStatusMismatch,
		},
		Registration: externalevidence.FMCSARegistrationRecord{
			LegalName: "Different Carrier Logistics",
		},
		Safety: externalevidence.FMCSASafetyRecord{
			SafetyRating: "satisfactory",
		},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}

	if status.Scorecard.EligibilityTier != EligibilityTierRestricted {
		t.Fatalf("EligibilityTier = %q, want %q", status.Scorecard.EligibilityTier, EligibilityTierRestricted)
	}

	if len(status.FraudSignals) == 0 {
		t.Fatal("FraudSignals = empty, want mismatch signal")
	}
}

func TestVerificationCompleteness(t *testing.T) {
	t.Parallel()

	got := verificationCompleteness(carrieridentity.OnboardingStatus{
		Requirements: []verification.Requirement{
			{Status: verification.RequirementStatusSatisfied},
			{Status: verification.RequirementStatusPending},
			{Status: verification.RequirementStatusSatisfied},
		},
	})
	want := 2.0 / 3.0
	if got != want {
		t.Fatalf("verificationCompleteness() = %f, want %f", got, want)
	}
}

func TestNewGrantUsesScorecard(t *testing.T) {
	t.Parallel()

	grant := newGrant(CarrierScorecard{
		ID:               "score_1",
		CarrierAccountID: "carrier_1",
	}, "load_access", "pending_review", time.Now().UTC())

	if grant.SourceScorecardID != "score_1" {
		t.Fatalf("SourceScorecardID = %q, want %q", grant.SourceScorecardID, "score_1")
	}
}
