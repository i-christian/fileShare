package user

import (
	"net/http"

	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/utils/security"
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
	ctxUser, ok := security.GetUserFromContext(r)
	if !ok {
		utils.UnauthorisedResponse(w, "try to login first")
		return
	}

	userDetails, err := h.userService.GetUserInfo(r.Context(), ctxUser)
	if err != nil {
		utils.WriteServerError(h.userService.logger, "failed to get user information", err)
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, utils.Envelope{"user": userDetails}, nil)
	if err != nil {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
		utils.WriteServerError(h.userService.logger, "failed to encode a json response", err)
	}
}
