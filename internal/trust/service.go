package trust

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/carrieridentity"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/externalevidence"
	"github.com/google/uuid"
)

func (s *Service) buildEvaluation(onboarding carrieridentity.OnboardingStatus, fmcsa externalevidence.FMCSAData, now time.Time) Evaluation {
	scorecardID := uuid.NewString()
	completeness := verificationCompleteness(onboarding)
	fraudSignals := buildFraudSignals(onboarding, fmcsa, scorecardID, now)
	scoreValue := calculateScore(completeness, fmcsa, fraudSignals)
	scoreBand := bandForScore(scoreValue)
	eligibilityTier := eligibilityFor(onboarding, fmcsa, fraudSignals)
	reasonSummary := fmt.Sprintf("FMCSA=%s, completeness=%.0f%%, signals=%d", fmcsa.Snapshot.Status, completeness*100, len(fraudSignals))

	scorecard := CarrierScorecard{
		ID:                       scorecardID,
		CarrierAccountID:         onboarding.Carrier.ID,
		ScoreVersion:             "v1",
		ScoreValue:               scoreValue,
		ScoreBand:                scoreBand,
		EligibilityTier:          eligibilityTier,
		VerificationCompleteness: completeness,
		ReasonSummary:            reasonSummary,
		GeneratedAt:              now,
	}

	inputs := []CarrierScoreInput{
		{
			ID:                uuid.NewString(),
			CarrierAccountID:  onboarding.Carrier.ID,
			SourceScorecardID: scorecardID,
			InputType:         "verification_completeness",
			Source:            "platform",
			ValueNumeric:      completeness,
			ValueText:         fmt.Sprintf("%.2f", completeness),
			EffectiveAt:       now,
		},
		{
			ID:                uuid.NewString(),
			CarrierAccountID:  onboarding.Carrier.ID,
			SourceScorecardID: scorecardID,
			InputType:         "fmcsa_match_status",
			Source:            "fmcsa",
			ValueNumeric:      0,
			ValueText:         string(fmcsa.Snapshot.Status),
			EffectiveAt:       now,
		},
		{
			ID:                uuid.NewString(),
			CarrierAccountID:  onboarding.Carrier.ID,
			SourceScorecardID: scorecardID,
			InputType:         "safety_rating",
			Source:            "fmcsa",
			ValueNumeric:      float64(fmcsa.Safety.CrashCount),
			ValueText:         fmcsa.Safety.SafetyRating,
			EffectiveAt:       now,
		},
		{
			ID:                uuid.NewString(),
			CarrierAccountID:  onboarding.Carrier.ID,
			SourceScorecardID: scorecardID,
			InputType:         "insurance_policy_count",
			Source:            "platform",
			ValueNumeric:      float64(len(onboarding.InsurancePolicies)),
			ValueText:         "",
			EffectiveAt:       now,
		},
	}

	return Evaluation{
		Inputs:       inputs,
		Scorecard:    scorecard,
		AccessGrants: grantsFor(scorecard, now),
		FraudSignals: fraudSignals,
	}
}

func verificationCompleteness(onboarding carrieridentity.OnboardingStatus) float64 {
	if len(onboarding.Requirements) == 0 {
		return 0
	}

	satisfied := 0
	for _, requirement := range onboarding.Requirements {
		if requirement.Status == "satisfied" {
			satisfied++
		}
	}

	return float64(satisfied) / float64(len(onboarding.Requirements))
}

func calculateScore(completeness float64, fmcsa externalevidence.FMCSAData, fraudSignals []FraudSignal) int {
	score := 35 + int(completeness*35)

	switch fmcsa.Snapshot.Status {
	case externalevidence.SnapshotStatusMatched:
		score += 15
	case externalevidence.SnapshotStatusMismatch:
		score -= 20
	}

	switch strings.ToLower(fmcsa.Safety.SafetyRating) {
	case "satisfactory":
		score += 10
	case "conditional":
		score += 2
	case "unsatisfactory":
		score -= 20
	}

	if !fmcsa.Registration.OutOfService {
		score += 5
	} else {
		score -= 25
	}

	score -= fmcsa.Safety.CrashCount * 3
	for _, signal := range fraudSignals {
		switch signal.Severity {
		case FraudSeverityHigh:
			score -= 20
		case FraudSeverityMedium:
			score -= 10
		default:
			score -= 3
		}
	}

	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}

	return score
}

func bandForScore(score int) ScoreBand {
	switch {
	case score >= 75:
		return ScoreBandHigh
	case score >= 45:
		return ScoreBandMedium
	default:
		return ScoreBandLow
	}
}

func eligibilityFor(onboarding carrieridentity.OnboardingStatus, fmcsa externalevidence.FMCSAData, fraudSignals []FraudSignal) EligibilityTier {
	for _, signal := range fraudSignals {
		if signal.Severity == FraudSeverityHigh {
			return EligibilityTierRestricted
		}
	}

	if onboarding.Carrier.OnboardingStage != carrieridentity.OnboardingStageReviewPending {
		return EligibilityTierReviewPending
	}
	if fmcsa.Snapshot.Status != externalevidence.SnapshotStatusMatched {
		return EligibilityTierReviewPending
	}

	return EligibilityTierTier0
}

func grantsFor(scorecard CarrierScorecard, now time.Time) []AccessGrant {
	switch scorecard.EligibilityTier {
	case EligibilityTierTier0:
		return []AccessGrant{
			newGrant(scorecard, "load_value_cap", "2500", now),
			newGrant(scorecard, "allowed_load_risk_band", "low", now),
			newGrant(scorecard, "requires_manual_dispatch_review", "true", now),
			newGrant(scorecard, "max_open_loads", "1", now),
		}
	case EligibilityTierRestricted:
		return []AccessGrant{
			newGrant(scorecard, "load_access", "blocked", now),
			newGrant(scorecard, "payout_hold", "true", now),
		}
	default:
		return []AccessGrant{
			newGrant(scorecard, "load_access", "pending_review", now),
		}
	}
}

func newGrant(scorecard CarrierScorecard, grantType string, grantValue string, now time.Time) AccessGrant {
	return AccessGrant{
		ID:                uuid.NewString(),
		CarrierAccountID:  scorecard.CarrierAccountID,
		GrantType:         grantType,
		GrantValue:        grantValue,
		GrantedAt:         now,
		SourceScorecardID: scorecard.ID,
	}
}

func buildFraudSignals(onboarding carrieridentity.OnboardingStatus, fmcsa externalevidence.FMCSAData, scorecardID string, now time.Time) []FraudSignal {
	signals := make([]FraudSignal, 0, 3)

	if fmcsa.Snapshot.Status == externalevidence.SnapshotStatusMismatch {
		signals = append(signals, newSignal(onboarding.Carrier.ID, scorecardID, "fmcsa_identity_mismatch", FraudSeverityHigh, map[string]any{
			"carrierLegalName": onboarding.Carrier.LegalName,
			"fmcsaLegalName":   fmcsa.Registration.LegalName,
		}, now))
	}
	if fmcsa.Registration.OutOfService {
		signals = append(signals, newSignal(onboarding.Carrier.ID, scorecardID, "fmcsa_out_of_service", FraudSeverityHigh, map[string]any{
			"operatingStatus": fmcsa.Registration.OperatingStatus,
		}, now))
	}
	if fmcsa.Safety.CrashCount >= 3 {
		signals = append(signals, newSignal(onboarding.Carrier.ID, scorecardID, "elevated_crash_history", FraudSeverityMedium, map[string]any{
			"crashCount": fmcsa.Safety.CrashCount,
		}, now))
	}

	return signals
}

func newSignal(carrierID string, scorecardID string, signalType string, severity FraudSeverity, evidence map[string]any, now time.Time) FraudSignal {
	body, _ := json.Marshal(evidence)
	return FraudSignal{
		ID:                uuid.NewString(),
		CarrierAccountID:  carrierID,
		SourceScorecardID: scorecardID,
		SignalType:        signalType,
		Severity:          severity,
		Status:            FraudSignalStatusOpen,
		DetectedAt:        now,
		EvidenceJSON:      body,
	}
}
