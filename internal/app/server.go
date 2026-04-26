package app

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	platformauth "github.com/DavitHakobyan/shipper-to-carrier/internal/platform/auth"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/platform/config"
	"github.com/DavitHakobyan/shipper-to-carrier/internal/platform/web"

	"github.com/DavitHakobyan/shipper-to-carrier/internal/identity"
)

type Authenticator interface {
	Register(rctx context.Context, input identity.RegisterInput) (identity.SessionResult, error)
	Login(rctx context.Context, input identity.LoginInput) (identity.SessionResult, error)
	Current(rctx context.Context, sessionToken string) (identity.AuthenticatedAccount, error)
	Logout(rctx context.Context, sessionToken string) error
}

type Server struct {
	config        config.Config
	authenticator Authenticator
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

func NewServer(cfg config.Config, authenticator Authenticator) (http.Handler, error) {
	assetHandler, err := web.NewHandler()
	if err != nil {
		return nil, err
	}

	server := &Server{
		config:        cfg,
		authenticator: authenticator,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", server.handleHealth)
	mux.HandleFunc("GET /api/v1/config", server.handleConfig)
	mux.HandleFunc("POST /api/v1/accounts/register", server.handleRegister)
	mux.HandleFunc("POST /api/v1/sessions", server.handleLogin)
	mux.HandleFunc("POST /api/v1/sessions/logout", server.handleLogout)
	mux.HandleFunc("GET /api/v1/me", server.handleCurrent)
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
