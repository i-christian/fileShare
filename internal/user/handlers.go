package user

import (
	"errors"
	"net/http"

	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/utils/security"
	"github.com/i-christian/fileShare/internal/validator"
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

func (h *UserHandler) ActivateUserHandler(w http.ResponseWriter, r *http.Request) {
	userIDFromContext, ok := security.GetUserFromContext(r)
	if !ok {
		utils.UnauthorisedResponse(w, "unauthorized")
	}

	var input struct {
		TokenPlainText string `json:"token"`
	}

	err := utils.ReadJSON(w, r, &input)
	if err != nil {
		utils.BadRequestResponse(w, err)
		return
	}

	v := validator.New()
	if validator.ValidateTokenPlainText(v, input.TokenPlainText); !v.Valid() {
		utils.FailedValidationResponse(w, v.Errors)
		return
	}

	_, activated, err := h.userService.ActivateUser(r.Context(), userIDFromContext, input.TokenPlainText, v)
	if !v.Valid() {
		utils.FailedValidationResponse(w, v.Errors)
		return
	}
	if err != nil {
		switch {
		case errors.Is(err, utils.ErrEditConflict):
			utils.EditConflictResponse(w)
		default:
			utils.ServerErrorResponse(w, "failed to verify user email")
		}

		utils.WriteServerError(h.userService.logger, "failed to verify user email", err)
		return
	}

	if !activated {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
		utils.WriteServerError(h.userService.logger, "failed to verify user email", err)
		return
	}

	userDetails, err := h.userService.GetUserInfo(r.Context(), userIDFromContext)
	if err != nil {
		utils.WriteServerError(h.userService.logger, "failed to get user information", err)
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
		return
	}

	err = utils.WriteJSON(w, http.StatusOK, utils.Envelope{"user": userDetails}, nil)
	if err != nil {
		utils.ServerErrorResponse(w, utils.ErrUnexpectedError.Error())
		utils.WriteServerError(h.userService.logger, "failed to send json response", err)
	}
}
