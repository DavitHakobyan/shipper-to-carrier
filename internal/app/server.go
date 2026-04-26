package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/carrieridentity"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/externalevidence"
	platformauth "github.com/DavitHakobyan/shipper-to-carrier/internal/platform/auth"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/platform/config"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/platform/web"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/trust"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/verification"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/identity"
)

type Authenticator interface {
	Register(rctx context.Context, input identity.RegisterInput) (identity.SessionResult, error)
	Login(rctx context.Context, input identity.LoginInput) (identity.SessionResult, error)
	Current(rctx context.Context, sessionToken string) (identity.AuthenticatedAccount, error)
	Logout(rctx context.Context, sessionToken string) error
}

type CarrierOnboarder interface {
	CreateCarrier(ctx context.Context, actor identity.AuthenticatedAccount, input carrieridentity.CreateCarrierInput) (carrieridentity.OnboardingStatus, error)
	AddOwner(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string, input carrieridentity.AddOwnerInput) (carrieridentity.OnboardingStatus, error)
	UpsertAuthority(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string, input carrieridentity.UpsertAuthorityInput) (carrieridentity.OnboardingStatus, error)
	AddInsurance(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string, input carrieridentity.AddInsuranceInput) (carrieridentity.OnboardingStatus, error)
	GetOnboardingStatus(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string) (carrieridentity.OnboardingStatus, error)
	GetCurrentOnboardingStatus(ctx context.Context, actor identity.AuthenticatedAccount) (carrieridentity.OnboardingStatus, error)
}

type EvidenceService interface {
	RefreshFMCSA(ctx context.Context, status carrieridentity.OnboardingStatus) (externalevidence.FMCSAData, error)
	LatestFMCSA(ctx context.Context, carrierAccountID string) (externalevidence.FMCSAData, error)
}

type TrustService interface {
	Evaluate(ctx context.Context, onboarding carrieridentity.OnboardingStatus, fmcsa externalevidence.FMCSAData) (trust.TrustStatus, error)
	LatestTrust(ctx context.Context, carrierAccountID string) (trust.TrustStatus, error)
}

type Server struct {
	config           config.Config
	authenticator    Authenticator
	carrierOnboarder CarrierOnboarder
	evidenceService  EvidenceService
	trustService     TrustService
}

type configResponse struct {
	AppName string          `json:"appName"`
	Roles   []identity.Role `json:"roles"`
}

type authResponse struct {
	Account   accountResponse `json:"account"`
	ExpiresAt time.Time       `json:"expiresAt,omitempty"`
}

type accountResponse struct {
	ID          string        `json:"id"`
	Email       string        `json:"email"`
	DisplayName string        `json:"displayName"`
	Role        identity.Role `json:"role"`
}

type registerRequest struct {
	Email       string        `json:"email"`
	Password    string        `json:"password"`
	DisplayName string        `json:"displayName"`
	Role        identity.Role `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type carrierAddressRequest struct {
	AddressType string `json:"addressType"`
	Line1       string `json:"line1"`
	Line2       string `json:"line2"`
	City        string `json:"city"`
	State       string `json:"state"`
	PostalCode  string `json:"postalCode"`
	Country     string `json:"country"`
}

type createCarrierRequest struct {
	LegalName          string                `json:"legalName"`
	DoingBusinessAs    string                `json:"doingBusinessAs"`
	ContactPhone       string                `json:"contactPhone"`
	FleetSizeDeclared  int                   `json:"fleetSizeDeclared"`
	OperatingRegions   []string              `json:"operatingRegions"`
	PreferredLoadTypes []string              `json:"preferredLoadTypes"`
	Address            carrierAddressRequest `json:"address"`
}

type addOwnerRequest struct {
	FullName         string `json:"fullName"`
	Phone            string `json:"phone"`
	Email            string `json:"email"`
	OwnershipRole    string `json:"ownershipRole"`
	IsPrimaryContact bool   `json:"isPrimaryContact"`
}

type upsertAuthorityRequest struct {
	DOTNumber     string `json:"dotNumber"`
	MCNumber      string `json:"mcNumber"`
	USDOTStatus   string `json:"usdotStatus"`
	AuthorityType string `json:"authorityType"`
}

type addInsuranceRequest struct {
	ProviderName       string    `json:"providerName"`
	PolicyNumber       string    `json:"policyNumber"`
	CoverageType       string    `json:"coverageType"`
	EffectiveAt        time.Time `json:"effectiveAt"`
	ExpiresAt          time.Time `json:"expiresAt"`
	VerificationStatus string    `json:"verificationStatus"`
}

type onboardingStatusResponse struct {
	Carrier             carrierResponse                   `json:"carrier"`
	Profile             carrierProfileResponse            `json:"profile"`
	Addresses           []carrierAddressResponse          `json:"addresses"`
	Owners              []ownerResponse                   `json:"owners"`
	AuthorityLink       *authorityResponse                `json:"authorityLink,omitempty"`
	InsurancePolicies   []insuranceResponse               `json:"insurancePolicies"`
	VerificationCase    verificationCaseResponse          `json:"verificationCase"`
	Requirements        []verificationRequirementResponse `json:"requirements"`
	MissingRequirements []verification.RequirementType    `json:"missingRequirements"`
}

type carrierResponse struct {
	ID              string                          `json:"id"`
	LegalName       string                          `json:"legalName"`
	DoingBusinessAs string                          `json:"doingBusinessAs"`
	Status          carrieridentity.CarrierStatus   `json:"status"`
	OnboardingStage carrieridentity.OnboardingStage `json:"onboardingStage"`
}

type carrierProfileResponse struct {
	ContactPhone       string   `json:"contactPhone"`
	ContactEmail       string   `json:"contactEmail"`
	FleetSizeDeclared  int      `json:"fleetSizeDeclared"`
	OperatingRegions   []string `json:"operatingRegions"`
	PreferredLoadTypes []string `json:"preferredLoadTypes"`
}

type carrierAddressResponse struct {
	AddressType string `json:"addressType"`
	Line1       string `json:"line1"`
	Line2       string `json:"line2"`
	City        string `json:"city"`
	State       string `json:"state"`
	PostalCode  string `json:"postalCode"`
	Country     string `json:"country"`
}

type ownerResponse struct {
	FullName         string `json:"fullName"`
	Phone            string `json:"phone"`
	Email            string `json:"email"`
	OwnershipRole    string `json:"ownershipRole"`
	IsPrimaryContact bool   `json:"isPrimaryContact"`
}

type authorityResponse struct {
	DOTNumber     string `json:"dotNumber"`
	MCNumber      string `json:"mcNumber"`
	USDOTStatus   string `json:"usdotStatus"`
	AuthorityType string `json:"authorityType"`
}

type insuranceResponse struct {
	ProviderName       string    `json:"providerName"`
	CoverageType       string    `json:"coverageType"`
	EffectiveAt        time.Time `json:"effectiveAt"`
	ExpiresAt          time.Time `json:"expiresAt"`
	VerificationStatus string    `json:"verificationStatus"`
}

type verificationCaseResponse struct {
	ID       string                  `json:"id"`
	CaseType verification.CaseType   `json:"caseType"`
	Status   verification.CaseStatus `json:"status"`
	OpenedAt time.Time               `json:"openedAt"`
}

type verificationRequirementResponse struct {
	RequirementType verification.RequirementType   `json:"requirementType"`
	Status          verification.RequirementStatus `json:"status"`
	SatisfiedAt     *time.Time                     `json:"satisfiedAt,omitempty"`
}

type intelligenceResponse struct {
	FMCSA     fmcsaResponse     `json:"fmcsa"`
	Scorecard scorecardResponse `json:"scorecard"`
}

type fmcsaResponse struct {
	Status          externalevidence.SnapshotStatus `json:"status"`
	FetchedAt       time.Time                       `json:"fetchedAt"`
	SourceKey       string                          `json:"sourceKey"`
	LegalName       string                          `json:"legalName"`
	AuthorityStatus string                          `json:"authorityStatus"`
	OperatingStatus string                          `json:"operatingStatus"`
	OutOfService    bool                            `json:"outOfService"`
	SafetyRating    string                          `json:"safetyRating"`
	CrashCount      int                             `json:"crashCount"`
	InspectionCount int                             `json:"inspectionCount"`
	OOSRate         float64                         `json:"oosRate"`
}

type scorecardResponse struct {
	ScoreValue               int                   `json:"scoreValue"`
	ScoreBand                trust.ScoreBand       `json:"scoreBand"`
	EligibilityTier          trust.EligibilityTier `json:"eligibilityTier"`
	VerificationCompleteness float64               `json:"verificationCompleteness"`
	ReasonSummary            string                `json:"reasonSummary"`
	GeneratedAt              time.Time             `json:"generatedAt"`
	AccessGrants             []accessGrantResponse `json:"accessGrants"`
	FraudSignals             []fraudSignalResponse `json:"fraudSignals"`
}

type accessGrantResponse struct {
	GrantType  string `json:"grantType"`
	GrantValue string `json:"grantValue"`
}

type fraudSignalResponse struct {
	SignalType string                  `json:"signalType"`
	Severity   trust.FraudSeverity     `json:"severity"`
	Status     trust.FraudSignalStatus `json:"status"`
}

func NewServer(cfg config.Config, authenticator Authenticator, carrierOnboarder CarrierOnboarder, evidenceService EvidenceService, trustService TrustService) (http.Handler, error) {
	assetHandler, err := web.NewHandler()
	if err != nil {
		return nil, err
	}

	server := &Server{
		config:           cfg,
		authenticator:    authenticator,
		carrierOnboarder: carrierOnboarder,
		evidenceService:  evidenceService,
		trustService:     trustService,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.handleHealth)
	mux.HandleFunc("GET /api/v1/config", server.handleConfig)
	mux.HandleFunc("POST /api/v1/accounts/register", server.handleRegister)
	mux.HandleFunc("POST /api/v1/sessions", server.handleLogin)
	mux.HandleFunc("POST /api/v1/sessions/logout", server.handleLogout)
	mux.HandleFunc("GET /api/v1/me", server.handleCurrent)
	mux.HandleFunc("POST /api/v1/carriers", server.handleCreateCarrier)
	mux.HandleFunc("POST /api/v1/carriers/{carrierID}/owners", server.handleAddOwner)
	mux.HandleFunc("POST /api/v1/carriers/{carrierID}/authority", server.handleUpsertAuthority)
	mux.HandleFunc("POST /api/v1/carriers/{carrierID}/insurance", server.handleAddInsurance)
	mux.HandleFunc("GET /api/v1/carriers/{carrierID}/onboarding-status", server.handleOnboardingStatus)
	mux.HandleFunc("POST /api/v1/carriers/{carrierID}/fmcsa-refresh", server.handleFMCSARefresh)
	mux.HandleFunc("GET /api/v1/carriers/{carrierID}/intelligence", server.handleIntelligence)
	mux.Handle("/", assetHandler)

	return mux, nil
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, configResponse{
		AppName: s.config.AppName,
		Roles:   []identity.Role{identity.RoleCarrier, identity.RoleShipper},
	})
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var input registerRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	result, err := s.authenticator.Register(r.Context(), identity.RegisterInput{
		Email:       input.Email,
		Password:    input.Password,
		DisplayName: input.DisplayName,
		Role:        input.Role,
	})
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	platformauth.SetSessionCookie(w, s.config.SessionCookieName, result.SessionToken, result.SessionExpiresAt)
	writeJSON(w, http.StatusCreated, authResponse{
		Account:   accountFromAuthenticated(result.Account),
		ExpiresAt: result.SessionExpiresAt,
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var input loginRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	result, err := s.authenticator.Login(r.Context(), identity.LoginInput{
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	platformauth.SetSessionCookie(w, s.config.SessionCookieName, result.SessionToken, result.SessionExpiresAt)
	writeJSON(w, http.StatusOK, authResponse{
		Account:   accountFromAuthenticated(result.Account),
		ExpiresAt: result.SessionExpiresAt,
	})
}

func (s *Server) handleCurrent(w http.ResponseWriter, r *http.Request) {
	sessionToken, err := sessionTokenFromRequest(r, s.config.SessionCookieName)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: identity.ErrUnauthorized.Error()})
		return
	}

	account, err := s.authenticator.Current(r.Context(), sessionToken)
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, authResponse{
		Account: accountFromAuthenticated(account),
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	sessionToken, err := sessionTokenFromRequest(r, s.config.SessionCookieName)
	if err == nil {
		_ = s.authenticator.Logout(r.Context(), sessionToken)
	}

	platformauth.ClearSessionCookie(w, s.config.SessionCookieName)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateCarrier(w http.ResponseWriter, r *http.Request) {
	var input createCarrierRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	actor, err := s.authenticatedActor(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}

	status, err := s.carrierOnboarder.CreateCarrier(r.Context(), actor, carrieridentity.CreateCarrierInput{
		LegalName:          input.LegalName,
		DoingBusinessAs:    input.DoingBusinessAs,
		ContactPhone:       input.ContactPhone,
		FleetSizeDeclared:  input.FleetSizeDeclared,
		OperatingRegions:   input.OperatingRegions,
		PreferredLoadTypes: input.PreferredLoadTypes,
		Address: carrieridentity.CarrierAddressInput{
			AddressType: input.Address.AddressType,
			Line1:       input.Address.Line1,
			Line2:       input.Address.Line2,
			City:        input.Address.City,
			State:       input.Address.State,
			PostalCode:  input.Address.PostalCode,
			Country:     input.Address.Country,
		},
	})
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, onboardingStatusFromDomain(status))
}

func (s *Server) handleAddOwner(w http.ResponseWriter, r *http.Request) {
	var input addOwnerRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	actor, err := s.authenticatedActor(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}

	carrierID, err := s.resolveCarrierID(r.Context(), actor, r.PathValue("carrierID"))
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	status, err := s.carrierOnboarder.AddOwner(r.Context(), actor, carrierID, carrieridentity.AddOwnerInput{
		FullName:         input.FullName,
		Phone:            input.Phone,
		Email:            input.Email,
		OwnershipRole:    input.OwnershipRole,
		IsPrimaryContact: input.IsPrimaryContact,
	})
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, onboardingStatusFromDomain(status))
}

func (s *Server) handleUpsertAuthority(w http.ResponseWriter, r *http.Request) {
	var input upsertAuthorityRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	actor, err := s.authenticatedActor(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}

	carrierID, err := s.resolveCarrierID(r.Context(), actor, r.PathValue("carrierID"))
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	status, err := s.carrierOnboarder.UpsertAuthority(r.Context(), actor, carrierID, carrieridentity.UpsertAuthorityInput{
		DOTNumber:     input.DOTNumber,
		MCNumber:      input.MCNumber,
		USDOTStatus:   input.USDOTStatus,
		AuthorityType: input.AuthorityType,
	})
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	_, _ = s.syncCarrierIntelligence(r.Context(), actor, status)
	writeJSON(w, http.StatusOK, onboardingStatusFromDomain(status))
}

func (s *Server) handleAddInsurance(w http.ResponseWriter, r *http.Request) {
	var input addInsuranceRequest
	if err := decodeJSON(r, &input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	actor, err := s.authenticatedActor(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}

	carrierID, err := s.resolveCarrierID(r.Context(), actor, r.PathValue("carrierID"))
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	status, err := s.carrierOnboarder.AddInsurance(r.Context(), actor, carrierID, carrieridentity.AddInsuranceInput{
		ProviderName:       input.ProviderName,
		PolicyNumber:       input.PolicyNumber,
		CoverageType:       input.CoverageType,
		EffectiveAt:        input.EffectiveAt,
		ExpiresAt:          input.ExpiresAt,
		VerificationStatus: input.VerificationStatus,
	})
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	if _, err := s.tryEvaluateLatest(r.Context(), status); err != nil && !errors.Is(err, externalevidence.ErrNoSnapshot) {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, onboardingStatusFromDomain(status))
}

func (s *Server) handleOnboardingStatus(w http.ResponseWriter, r *http.Request) {
	actor, err := s.authenticatedActor(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}

	carrierID := r.PathValue("carrierID")
	var status carrieridentity.OnboardingStatus
	if carrierID == "current" {
		status, err = s.carrierOnboarder.GetCurrentOnboardingStatus(r.Context(), actor)
	} else {
		status, err = s.carrierOnboarder.GetOnboardingStatus(r.Context(), actor, carrierID)
	}
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, onboardingStatusFromDomain(status))
}

func (s *Server) handleFMCSARefresh(w http.ResponseWriter, r *http.Request) {
	actor, err := s.authenticatedActor(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}

	carrierID, err := s.resolveCarrierID(r.Context(), actor, r.PathValue("carrierID"))
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	status, err := s.carrierOnboarder.GetOnboardingStatus(r.Context(), actor, carrierID)
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	response, err := s.syncCarrierIntelligence(r.Context(), actor, status)
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) handleIntelligence(w http.ResponseWriter, r *http.Request) {
	actor, err := s.authenticatedActor(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: err.Error()})
		return
	}

	carrierID, err := s.resolveCarrierID(r.Context(), actor, r.PathValue("carrierID"))
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	onboarding, err := s.carrierOnboarder.GetOnboardingStatus(r.Context(), actor, carrierID)
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	fmcsa, err := s.evidenceService.LatestFMCSA(r.Context(), onboarding.Carrier.ID)
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	trustStatus, err := s.trustService.LatestTrust(r.Context(), onboarding.Carrier.ID)
	if err != nil {
		writeJSON(w, statusForError(err), errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, intelligenceFromDomain(fmcsa, trustStatus))
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func statusForError(err error) int {
	switch {
	case errors.Is(err, identity.ErrDuplicateEmail):
		return http.StatusConflict
	case errors.Is(err, identity.ErrInvalidCredentials), errors.Is(err, identity.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, carrieridentity.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, carrieridentity.ErrCarrierExists):
		return http.StatusConflict
	case errors.Is(err, carrieridentity.ErrCarrierNotFound):
		return http.StatusNotFound
	case errors.Is(err, externalevidence.ErrNoAuthorityLink):
		return http.StatusBadRequest
	case errors.Is(err, externalevidence.ErrNoSnapshot), errors.Is(err, trust.ErrNoScorecard):
		return http.StatusNotFound
	default:
		return http.StatusBadRequest
	}
}

func sessionTokenFromRequest(r *http.Request, cookieName string) (string, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return "", identity.ErrUnauthorized
	}

	return cookie.Value, nil
}

func accountFromAuthenticated(account identity.AuthenticatedAccount) accountResponse {
	return accountResponse{
		ID:          account.AccountID,
		Email:       account.Email,
		DisplayName: account.DisplayName,
		Role:        account.Role,
	}
}

func (s *Server) authenticatedActor(r *http.Request) (identity.AuthenticatedAccount, error) {
	sessionToken, err := sessionTokenFromRequest(r, s.config.SessionCookieName)
	if err != nil {
		return identity.AuthenticatedAccount{}, err
	}

	return s.authenticator.Current(r.Context(), sessionToken)
}

func (s *Server) resolveCarrierID(ctx context.Context, actor identity.AuthenticatedAccount, carrierID string) (string, error) {
	if carrierID != "current" {
		return carrierID, nil
	}

	status, err := s.carrierOnboarder.GetCurrentOnboardingStatus(ctx, actor)
	if err != nil {
		return "", err
	}

	return status.Carrier.ID, nil
}

func (s *Server) syncCarrierIntelligence(ctx context.Context, actor identity.AuthenticatedAccount, onboarding carrieridentity.OnboardingStatus) (intelligenceResponse, error) {
	fmcsa, err := s.evidenceService.RefreshFMCSA(ctx, onboarding)
	if err != nil {
		return intelligenceResponse{}, err
	}

	trustStatus, err := s.trustService.Evaluate(ctx, onboarding, fmcsa)
	if err != nil {
		return intelligenceResponse{}, err
	}

	return intelligenceFromDomain(fmcsa, trustStatus), nil
}

func (s *Server) tryEvaluateLatest(ctx context.Context, onboarding carrieridentity.OnboardingStatus) (intelligenceResponse, error) {
	fmcsa, err := s.evidenceService.LatestFMCSA(ctx, onboarding.Carrier.ID)
	if err != nil {
		return intelligenceResponse{}, err
	}

	trustStatus, err := s.trustService.Evaluate(ctx, onboarding, fmcsa)
	if err != nil {
		return intelligenceResponse{}, err
	}

	return intelligenceFromDomain(fmcsa, trustStatus), nil
}

func onboardingStatusFromDomain(status carrieridentity.OnboardingStatus) onboardingStatusResponse {
	response := onboardingStatusResponse{
		Carrier: carrierResponse{
			ID:              status.Carrier.ID,
			LegalName:       status.Carrier.LegalName,
			DoingBusinessAs: status.Carrier.DoingBusinessAs,
			Status:          status.Carrier.Status,
			OnboardingStage: status.Carrier.OnboardingStage,
		},
		Profile: carrierProfileResponse{
			ContactPhone:       status.Profile.ContactPhone,
			ContactEmail:       status.Profile.ContactEmail,
			FleetSizeDeclared:  status.Profile.FleetSizeDeclared,
			OperatingRegions:   status.Profile.OperatingRegions,
			PreferredLoadTypes: status.Profile.PreferredLoadTypes,
		},
		VerificationCase: verificationCaseResponse{
			ID:       status.VerificationCase.ID,
			CaseType: status.VerificationCase.CaseType,
			Status:   status.VerificationCase.Status,
			OpenedAt: status.VerificationCase.OpenedAt,
		},
		MissingRequirements: status.MissingRequirements,
	}

	for _, address := range status.Addresses {
		response.Addresses = append(response.Addresses, carrierAddressResponse{
			AddressType: address.AddressType,
			Line1:       address.Line1,
			Line2:       address.Line2,
			City:        address.City,
			State:       address.State,
			PostalCode:  address.PostalCode,
			Country:     address.Country,
		})
	}

	for _, owner := range status.Owners {
		response.Owners = append(response.Owners, ownerResponse{
			FullName:         owner.FullName,
			Phone:            owner.Phone,
			Email:            owner.Email,
			OwnershipRole:    owner.OwnershipRole,
			IsPrimaryContact: owner.IsPrimaryContact,
		})
	}

	if status.AuthorityLink != nil {
		response.AuthorityLink = &authorityResponse{
			DOTNumber:     status.AuthorityLink.DOTNumber,
			MCNumber:      status.AuthorityLink.MCNumber,
			USDOTStatus:   status.AuthorityLink.USDOTStatus,
			AuthorityType: status.AuthorityLink.AuthorityType,
		}
	}

	for _, policy := range status.InsurancePolicies {
		response.InsurancePolicies = append(response.InsurancePolicies, insuranceResponse{
			ProviderName:       policy.ProviderName,
			CoverageType:       policy.CoverageType,
			EffectiveAt:        policy.EffectiveAt,
			ExpiresAt:          policy.ExpiresAt,
			VerificationStatus: policy.VerificationStatus,
		})
	}

	for _, requirement := range status.Requirements {
		response.Requirements = append(response.Requirements, verificationRequirementResponse{
			RequirementType: requirement.RequirementType,
			Status:          requirement.Status,
			SatisfiedAt:     requirement.SatisfiedAt,
		})
	}

	return response
}

func intelligenceFromDomain(fmcsa externalevidence.FMCSAData, trustStatus trust.TrustStatus) intelligenceResponse {
	response := intelligenceResponse{
		FMCSA: fmcsaResponse{
			Status:          fmcsa.Snapshot.Status,
			FetchedAt:       fmcsa.Snapshot.FetchedAt,
			SourceKey:       fmcsa.Snapshot.SourceKey,
			LegalName:       fmcsa.Registration.LegalName,
			AuthorityStatus: fmcsa.Registration.AuthorityStatus,
			OperatingStatus: fmcsa.Registration.OperatingStatus,
			OutOfService:    fmcsa.Registration.OutOfService,
			SafetyRating:    fmcsa.Safety.SafetyRating,
			CrashCount:      fmcsa.Safety.CrashCount,
			InspectionCount: fmcsa.Safety.InspectionCount,
			OOSRate:         fmcsa.Safety.OOSRate,
		},
		Scorecard: scorecardResponse{
			ScoreValue:               trustStatus.Scorecard.ScoreValue,
			ScoreBand:                trustStatus.Scorecard.ScoreBand,
			EligibilityTier:          trustStatus.Scorecard.EligibilityTier,
			VerificationCompleteness: trustStatus.Scorecard.VerificationCompleteness,
			ReasonSummary:            trustStatus.Scorecard.ReasonSummary,
			GeneratedAt:              trustStatus.Scorecard.GeneratedAt,
		},
	}

	for _, grant := range trustStatus.AccessGrants {
		response.Scorecard.AccessGrants = append(response.Scorecard.AccessGrants, accessGrantResponse{
			GrantType:  grant.GrantType,
			GrantValue: grant.GrantValue,
		})
	}

	for _, signal := range trustStatus.FraudSignals {
		response.Scorecard.FraudSignals = append(response.Scorecard.FraudSignals, fraudSignalResponse{
			SignalType: signal.SignalType,
			Severity:   signal.Severity,
			Status:     signal.Status,
		})
	}

	return response
}
