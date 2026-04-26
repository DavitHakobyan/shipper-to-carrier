package trust

import (
	"context"
	"errors"
	"time"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/carrieridentity"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/externalevidence"
)

type ScoreBand string
type EligibilityTier string
type FraudSeverity string
type FraudSignalStatus string

const (
	ScoreBandLow    ScoreBand = "low"
	ScoreBandMedium ScoreBand = "medium"
	ScoreBandHigh   ScoreBand = "high"

	EligibilityTierReviewPending EligibilityTier = "review_pending"
	EligibilityTierTier0         EligibilityTier = "tier_0"
	EligibilityTierRestricted    EligibilityTier = "restricted"

	FraudSeverityLow    FraudSeverity = "low"
	FraudSeverityMedium FraudSeverity = "medium"
	FraudSeverityHigh   FraudSeverity = "high"

	FraudSignalStatusOpen FraudSignalStatus = "open"
)

var ErrNoScorecard = errors.New("carrier scorecard not found")

type CarrierScoreInput struct {
	ID                string
	CarrierAccountID  string
	SourceScorecardID string
	InputType         string
	Source            string
	ValueNumeric      float64
	ValueText         string
	EffectiveAt       time.Time
}

type CarrierScorecard struct {
	ID                       string
	CarrierAccountID         string
	ScoreVersion             string
	ScoreValue               int
	ScoreBand                ScoreBand
	EligibilityTier          EligibilityTier
	VerificationCompleteness float64
	ReasonSummary            string
	GeneratedAt              time.Time
}

type AccessGrant struct {
	ID                string
	CarrierAccountID  string
	GrantType         string
	GrantValue        string
	GrantedAt         time.Time
	RevokedAt         *time.Time
	SourceScorecardID string
}

type FraudSignal struct {
	ID                string
	CarrierAccountID  string
	SourceScorecardID string
	SignalType        string
	Severity          FraudSeverity
	Status            FraudSignalStatus
	DetectedAt        time.Time
	EvidenceJSON      []byte
	ReviewedAt        *time.Time
}

type TrustStatus struct {
	Inputs       []CarrierScoreInput
	Scorecard    CarrierScorecard
	AccessGrants []AccessGrant
	FraudSignals []FraudSignal
}

type Evaluation struct {
	Inputs       []CarrierScoreInput
	Scorecard    CarrierScorecard
	AccessGrants []AccessGrant
	FraudSignals []FraudSignal
}

type Repository interface {
	SaveEvaluation(ctx context.Context, evaluation Evaluation) error
	LatestTrust(ctx context.Context, carrierAccountID string) (TrustStatus, error)
}

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  time.Now,
	}
}

func (s *Service) Evaluate(ctx context.Context, onboarding carrieridentity.OnboardingStatus, fmcsa externalevidence.FMCSAData) (TrustStatus, error) {
	evaluation := s.buildEvaluation(onboarding, fmcsa, s.now().UTC())
	if err := s.repo.SaveEvaluation(ctx, evaluation); err != nil {
		return TrustStatus{}, err
	}

	return TrustStatus{
		Inputs:       evaluation.Inputs,
		Scorecard:    evaluation.Scorecard,
		AccessGrants: evaluation.AccessGrants,
		FraudSignals: evaluation.FraudSignals,
	}, nil
}

func (s *Service) LatestTrust(ctx context.Context, carrierAccountID string) (TrustStatus, error) {
	return s.repo.LatestTrust(ctx, carrierAccountID)
}
