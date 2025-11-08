package public

import (
	"log/slog"
	"net/http"

	"github.com/i-christian/fileShare/internal/utils"
)

type PublicHandler struct {
	logger  *slog.Logger
	env     string
	version string
}

func NewPublicHandler(env, version string, logger *slog.Logger) *PublicHandler {
	return &PublicHandler{
		env:     env,
		version: version,
		logger:  logger,
	}
}

// HealthStatus function handles an app health check endpoint
func (h *PublicHandler) HealthStatus(w http.ResponseWriter, r *http.Request) {
	env := utils.GetEnvOrFile("ENV")
	vers := utils.GetEnvOrFile("VERSION")
	data := utils.Envelope{
		"status": "available",
		"system_info": map[string]any{
			"environment": env,
			"version":     vers,
		},
	}

	err := utils.WriteJSON(w, http.StatusOK, data, nil)
	if err != nil {
		utils.ServerErrorResponse(w, "failed to process request, try again later.")
		utils.WriteServerError(h.logger, "failed to send a response", err)
	}
}
