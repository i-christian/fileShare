package middlewares

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/i-christian/fileShare/internal/auth"
	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/utils/security"
)

func AuthMiddleware(authService *auth.AuthService, apiKeyService *auth.ApiKeyService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.UnauthorisedResponse(w, "authorization header required")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 {
				utils.UnauthorisedResponse(w, "invalid authorization format")
				return
			}

			scheme, credential := parts[0], parts[1]

			switch scheme {
			case "Bearer":
				claims, err := authService.ValidateToken(credential)
				if err != nil {
					utils.UnauthorisedResponse(w, "invalid or expired token")
					return
				}

				userIDStr, ok := claims["sub"].(string)
				if !ok {
					utils.UnauthorisedResponse(w, "invalid token claims")
					return
				}

				userID, err := uuid.Parse(userIDStr)
				if err != nil {
					utils.UnauthorisedResponse(w, "invalid userID in token")
					return
				}

				ctx := context.WithValue(r.Context(), security.UserIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))

			case "ApiKey":
				userID, err := apiKeyService.ValidateAPIKey(r.Context(), credential)
				if err != nil {
					if errors.Is(err, utils.ErrUnexpectedError) {
						utils.UnauthorisedResponse(w, utils.ErrUnexpectedError.Error())
						utils.WriteServerError(authService.Logger, "api key authorisation failed", err)

					}
					utils.UnauthorisedResponse(w, err.Error())
					utils.WriteServerError(authService.Logger, "api key authorisation failed", err)
					return
				}

				ctx := context.WithValue(r.Context(), security.UserIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))

			default:
				utils.UnauthorisedResponse(w, "unsupported authorization scheme")
				return
			}
		})
	}
}
