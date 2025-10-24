package auth

import (
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
		utils.WriteErrorJSON(w, http.StatusBadRequest, "invalid request", h.logger)
		return
	}

	user, err := h.service.Register(r.Context(), req.Email, req.FirstName, req.LastName, req.Password)
	if err != nil {
		utils.WriteErrorJSON(w, http.StatusInternalServerError, "failed to create user", h.logger)
		h.logger.Error("failed to create user", "details", err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, user, nil, h.logger)
}

func (h *AuthHandler) LoginWithRefresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := utils.ReadJSON(w, r, &req); err != nil {
		utils.WriteErrorJSON(w, http.StatusBadRequest, "invalid request", h.logger)
		return
	}

	accessToken, refreshToken, err := h.service.LoginWithRefresh(r.Context(), req.Email, req.Password, h.refreshTokenTTL)
	if err != nil {
		utils.WriteErrorJSON(w, http.StatusUnauthorized, ErrInvalidCredentials.Error(), h.logger)
		h.logger.Error("login failure", "details", err.Error())
		return
	}

	data := map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	}
	utils.WriteJSON(w, http.StatusOK, data, nil, h.logger)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := utils.ReadJSON(w, r, &req); err != nil {
		utils.WriteErrorJSON(w, http.StatusBadRequest, "invalid request", h.logger)
		return
	}

	accessToken, err := h.service.RefreshAccessToken(r.Context(), req.RefreshToken)
	if err != nil {
		utils.WriteErrorJSON(w, http.StatusUnauthorized, "failed to refresh token", h.logger)
		return
	}

	utils.WriteJSON(w, http.StatusOK, map[string]string{"access_token": accessToken}, nil, h.logger)
}
