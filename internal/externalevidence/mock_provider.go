package externalevidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type MockProvider struct {
	now func() time.Time
}

func NewMockProvider() *MockProvider {
	return &MockProvider{now: time.Now}
}

func (p *MockProvider) FetchFMCSA(_ context.Context, input FMCSARequest) (FMCSAData, error) {
	sourceKey := input.DOTNumber
	if sourceKey == "" {
		sourceKey = input.MCNumber
	}
	if sourceKey == "" {
		return FMCSAData{}, ErrNoAuthorityLink
	}

	digit := trailingDigit(sourceKey)
	status := SnapshotStatusMatched
	legalName := input.LegalName
	if digit == 9 {
		status = SnapshotStatusMismatch
		legalName = "Different Carrier Logistics"
	}

	safetyRating := "satisfactory"
	switch {
	case digit >= 7:
		safetyRating = "conditional"
	case digit >= 8:
		safetyRating = "unsatisfactory"
	}
	if digit == 8 {
		safetyRating = "unsatisfactory"
	}

	authorityStatus := strings.TrimSpace(input.USDOTStatus)
	if authorityStatus == "" {
		authorityStatus = "active"
	}

	operatingStatus := "active"
	outOfService := false
	if digit == 8 {
		operatingStatus = "restricted"
		outOfService = true
	}

	now := p.now().UTC()
	payload := map[string]any{
		"sourceKey":       sourceKey,
		"status":          status,
		"legalName":       legalName,
		"authorityStatus": authorityStatus,
		"safetyRating":    safetyRating,
		"outOfService":    outOfService,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return FMCSAData{}, fmt.Errorf("marshal payload: %w", err)
	}

	snapshotID := uuid.NewString()
	return FMCSAData{
		Snapshot: ExternalRecordSnapshot{
			ID:               snapshotID,
			CarrierAccountID: input.CarrierAccountID,
			Source:           SourceFMCSA,
			SourceKey:        sourceKey,
			FetchedAt:        now,
			Status:           status,
			PayloadJSON:      payloadJSON,
			Checksum:         checksum(payloadJSON),
		},
		Registration: FMCSARegistrationRecord{
			SnapshotID:      snapshotID,
			DOTNumber:       input.DOTNumber,
			LegalName:       legalName,
			Address:         strings.TrimSpace(strings.Join([]string{input.AddressLine1, input.City, input.State}, ", ")),
			EntityType:      "for_hire_carrier",
			AuthorityStatus: authorityStatus,
			OutOfService:    outOfService,
			OperatingStatus: operatingStatus,
		},
		Safety: FMCSASafetyRecord{
			SnapshotID:          snapshotID,
			SafetyRating:        safetyRating,
			CrashCount:          digit % 4,
			InspectionCount:     12 + digit,
			OOSRate:             float64(digit) / 100,
			IncidentWindowStart: now.AddDate(-2, 0, 0),
			IncidentWindowEnd:   now,
		},
	}, nil
}

func trailingDigit(value string) int {
	for index := len(value) - 1; index >= 0; index-- {
		if value[index] < '0' || value[index] > '9' {
			continue
		}

		digit, err := strconv.Atoi(string(value[index]))
		if err == nil {
			return digit
		}
	}

	return 4
}

func checksum(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
