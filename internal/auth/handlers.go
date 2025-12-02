package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/i-christian/fileShare/internal/database"
	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/utils/security"
	"github.com/i-christian/fileShare/internal/validator"
	"github.com/i-christian/fileShare/internal/worker"
)

type AuthHandler struct {
	authService     *AuthService
	apiKeyService   *APIKeyService
	logger          *slog.Logger
	distributor     worker.Distributor
	refreshTokenTTL time.Duration
}

func NewAuthHandler(authService *AuthService, apiKeyService *APIKeyService, refreshTokenTTL time.Duration, logger *slog.Logger, distributor worker.Distributor) *AuthHandler {
	return &AuthHandler{
		authService:     authService,
		apiKeyService:   apiKeyService,
		refreshTokenTTL: refreshTokenTTL,
		logger:          logger,
		distributor:     distributor,
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

	data := map[string]any{
		"AppName":         utils.GetEnvOrFile("PROJECT_NAME"),
		"FirstName":       user.FirstName,
		"LastName":        user.LastName,
		"Email":           user.Email,
		"ActivationToken": user.ActivationToken,
		"Year":            time.Now().Year(),
	}
	payload := &worker.EmailPayload{
		Recipient:    user.Email,
		UserID:       user.UserID,
		TemplateFile: "user_welcome.tmpl",
		Data:         data,
	}
	opts := []asynq.Option{
		asynq.Queue("critical"),
		asynq.MaxRetry(5),
	}

	err = h.distributor.DistributeSendEmail(context.Background(), payload, opts...)
	if err != nil {
		utils.WriteServerError(h.logger, "failed to queue welcome email", err)
	}

	err = utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"user": user}, nil)
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

	err = utils.WriteJSON(w, http.StatusOK, utils.Envelope{"tokens": data}, nil)
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

	err = utils.WriteJSON(w, http.StatusOK, utils.Envelope{"access_token": accessToken}, nil)
	if err != nil {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
		utils.WriteServerError(h.logger, "failed to encode json response", err)
	}
}

func (h *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	user, ok := security.GetUserFromContext(r)
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

	fullKey, err := h.apiKeyService.GenerateAPIKey(r.Context(), user.UserID, req.KeyName, expires, newKeyScope())
	if err != nil {
		utils.ServerErrorResponse(w, "could not generate api key")
		utils.WriteServerError(h.logger, "could not generate api key", err)
		return
	}

	err = utils.WriteJSON(w, http.StatusCreated, utils.Envelope{
		"apiKey": fullKey,
	}, nil)
	if err != nil {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
		utils.WriteServerError(h.logger, "failed to encode json response", err)
	}
}

func (h *AuthHandler) SendPasswordResetLink(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}

	err := utils.ReadJSON(w, r, &input)
	if err != nil {
		utils.BadRequestResponse(w, err)
		return
	}

	v := validator.New()
	if validator.ValidateEmail(v, input.Email); !v.Valid() {
		utils.FailedValidationResponse(w, v.Errors)
	}

	userID, firstName, lastName, resetLink, err := h.authService.SendPasswordResetLink(r.Context(), input.Email)
	if err != nil {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
		utils.WriteServerError(h.logger, "failed to send reset link", err)
		return
	}

	data := map[string]any{
		"AppName":    utils.GetEnvOrFile("PROJECT_NAME"),
		"FirstName":  firstName,
		"LastName":   lastName,
		"Email":      input.Email,
		"UserID":     userID.String(),
		"ResetToken": resetLink,
		"Year":       time.Now().Year(),
	}
	payload := &worker.EmailPayload{
		Recipient:    input.Email,
		UserID:       userID,
		TemplateFile: "reset_password.tmpl",
		Data:         data,
	}
	opts := []asynq.Option{
		asynq.Queue("critical"),
		asynq.MaxRetry(5),
	}

	err = h.distributor.DistributeSendEmail(context.Background(), payload, opts...)
	if err != nil {
		utils.WriteServerError(h.logger, "failed to queue welcome email", err)
	}

	err = utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "Check your email for a reset link"}, nil)
	if err != nil {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
	}
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token       string `json:"token"`
		UserID      string `json:"user_id"`
		NewPassword string `json:"password"`
	}

	err := utils.ReadJSON(w, r, &input)
	if err != nil {
		utils.BadRequestResponse(w, err)
		return
	}

	parsedUserID, err := uuid.Parse(input.UserID)
	if err != nil {
		utils.FailedValidationResponse(w, map[string]string{"user_id": "a valid value must be provided"})
		return
	}

	v := validator.New()
	validator.ValidateResetPassword(v, input.Token, input.NewPassword)
	if !v.Valid() {
		utils.FailedValidationResponse(w, v.Errors)
		return
	}

	status, err := h.authService.VerifyPasswordReset(r.Context(), parsedUserID, input.NewPassword, input.Token, v)
	if err != nil {
		switch {
		case errors.Is(err, utils.ErrRecordNotFound):
			utils.FailedValidationResponse(w, v.Errors)
		case errors.Is(err, utils.ErrEditConflict):
			utils.EditConflictResponse(w)
		default:
			utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
			utils.WriteServerError(h.logger, "failed to reset password", err)
		}
		return
	}

	if !status {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
		utils.WriteServerError(h.logger, "failed to change password", err)

		return
	}

	err = utils.WriteJSON(w, http.StatusOK, utils.Envelope{"message": "Password successfully changed"}, nil)
	if err != nil {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
	}
}
