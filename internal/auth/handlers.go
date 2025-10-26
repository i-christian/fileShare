package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/i-christian/fileShare/internal/utils"
)

type AuthHandler struct {
	service         *AuthService
	logger          *slog.Logger
	refreshTokenTTL time.Duration
}

func NewAuthHandler(service *AuthService, refreshTokenTTL time.Duration, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		service:         service,
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

	user, err := h.service.Register(r.Context(), req.Email, req.FirstName, req.LastName, req.Password)
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

	accessToken, refreshToken, err := h.service.LoginWithRefresh(r.Context(), req.Email, req.Password, h.refreshTokenTTL)
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

	accessToken, err := h.service.RefreshAccessToken(r.Context(), req.RefreshToken)
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
