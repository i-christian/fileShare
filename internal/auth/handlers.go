package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/i-christian/fileShare/internal/database"
	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/utils/security"
	"github.com/i-christian/fileShare/internal/validator"
)

type AuthHandler struct {
	authService     *AuthService
	apiKeyService   *ApiKeyService
	logger          *slog.Logger
	refreshTokenTTL time.Duration
}

func NewAuthHandler(authService *AuthService, apiKeyService *ApiKeyService, refreshTokenTTL time.Duration, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		authService:     authService,
		apiKeyService:   apiKeyService,
		refreshTokenTTL: refreshTokenTTL,
		logger:          logger,
	}
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Password  string `json:"password"`
	}

	if err := utils.ReadJSON(w, r, &req); err != nil {
		utils.BadRequestResponse(w, err)
		return
	}

	userVerify := &validator.User{
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Password:  req.Password,
	}

	v := validator.New()
	if validator.ValidateUser(v, userVerify); !v.Valid() {
		utils.FailedValidationResponse(w, v.Errors)
		return
	}

	user, err := h.authService.Register(r.Context(), req.Email, req.FirstName, req.LastName, req.Password)
	if err != nil {
		utils.WriteServerError(h.logger, "failed to create user", err)
		utils.ServerErrorResponse(w, "failed to create user")
		return
	}

	err = utils.WriteJSON(w, http.StatusCreated, user, nil)
	if err != nil {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
		utils.WriteServerError(h.logger, "failed to encode json response", err)
	}
}

func (h *AuthHandler) LoginWithRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := utils.ReadJSON(w, r, &req); err != nil {
		utils.BadRequestResponse(w, err)
		return
	}

	loginVerify := &validator.User{
		Email:    req.Email,
		Password: req.Password,
	}

	v := validator.New()
	if validator.ValidateBasicLogin(v, loginVerify); !v.Valid() {
		utils.FailedValidationResponse(w, v.Errors)
		return
	}

	accessToken, refreshToken, err := h.authService.LoginWithRefresh(r.Context(), req.Email, req.Password, h.refreshTokenTTL)
	if err != nil {
		utils.UnauthorisedResponse(w, ErrInvalidCredentials.Error())
		utils.WriteServerError(h.logger, "login failure", err)
		return
	}

	data := map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	}

	err = utils.WriteJSON(w, http.StatusOK, data, nil)
	if err != nil {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
		utils.WriteServerError(h.logger, "failed to encode a json response", err)
	}
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := utils.ReadJSON(w, r, &req); err != nil {
		utils.BadRequestResponse(w, err)
		return
	}

	accessToken, err := h.authService.RefreshAccessToken(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, utils.ErrUnexpectedError) {
			utils.UnauthorisedResponse(w, utils.ErrUnexpectedError.Error())
			utils.WriteServerError(h.logger, "failed to refresh access token", err)
			return
		}

		utils.UnauthorisedResponse(w, err.Error())
		utils.WriteServerError(h.logger, "failed to refresh access token", err)
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, map[string]string{"access_token": accessToken}, nil)
	if err != nil {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
		utils.WriteServerError(h.logger, "failed to encode json response", err)
	}
}

func (h *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := security.GetUserFromContext(r)
	if !ok {
		utils.UnauthorisedResponse(w, "authentication required")
		return
	}

	var req struct {
		KeyName string    `json:"key_name"`
		Expires time.Time `json:"expires_at,omitzero"`
		Scope   []string  `json:"scope"`
	}

	if err := utils.ReadJSON(w, r, &req); err != nil {
		utils.BadRequestResponse(w, err)
		return
	}

	apiKeyLogin := &validator.ApiKey{
		KeyName: req.KeyName,
		Expires: req.Expires,
		Scope:   req.Scope,
	}

	v := validator.New()
	if validator.ValidateAPIKeyLogin(v, apiKeyLogin); !v.Valid() {
		utils.FailedValidationResponse(w, v.Errors)
		return
	}

	newKeyScope := func() []database.ApiScope {
		newScope := make([]database.ApiScope, 0)
		supportedScope := map[database.ApiScope]struct{}{
			database.ApiScopeRead:  {},
			database.ApiScopeWrite: {},
			database.ApiScopeSuper: {},
		}

		for _, scope := range req.Scope {
			if _, exists := supportedScope[database.ApiScope(scope)]; exists {
				newScope = append(newScope, database.ApiScope(scope))
			}
		}

		return newScope
	}

	expires := time.Now().Add(90 * 24 * time.Hour)
	if !req.Expires.IsZero() {
		expires = req.Expires
	}

	fullKey, err := h.apiKeyService.GenerateAPIKey(r.Context(), userID, req.KeyName, expires, newKeyScope())
	if err != nil {
		utils.ServerErrorResponse(w, "could not generate api key")
		utils.WriteServerError(h.logger, "could not generate api key", err)
		return
	}

	err = utils.WriteJSON(w, http.StatusCreated, map[string]string{
		"apiKey": fullKey,
	}, nil)
	if err != nil {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
		utils.WriteServerError(h.logger, "failed to encode json response", err)
	}
}
