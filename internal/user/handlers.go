package user

import (
	"net/http"

	"github.com/i-christian/fileShare/internal/middlewares"
	"github.com/i-christian/fileShare/internal/utils"
)

type UserHandler struct {
	userService *UserService
}

func NewUserHandler(u *UserService) *UserHandler {
	return &UserHandler{
		userService: u,
	}
}

func (h *UserHandler) MyProfile(w http.ResponseWriter, r *http.Request) {
	ctxUser, ok := middlewares.GetUserFromContext(r)
	if !ok {
		utils.WriteErrorJSON(w, http.StatusUnauthorized, "unauthorized", h.userService.logger)
		return
	}

	userDetails, err := h.userService.GetUserInfo(r.Context(), ctxUser)
	if err != nil {
		h.userService.logger.Error("failed to get user information", "details", err.Error())
		utils.WriteErrorJSON(w, http.StatusInternalServerError, "internal server error", h.userService.logger)
		return
	}

	utils.WriteJSON(w, http.StatusOK, userDetails, nil, h.userService.logger)
}
