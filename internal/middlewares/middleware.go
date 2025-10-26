package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/i-christian/fileShare/internal/auth"
	"github.com/i-christian/fileShare/internal/utils"
)

type contextKey string

const UserIDKey contextKey = "userID"

func AuthMiddleware(authService *auth.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.UnauthorisedResponse(w, "authorization header required")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				utils.UnauthorisedResponse(w, "invalid authorization format")
				return
			}

			tokenString := parts[1]
			claims, err := authService.ValidateToken(tokenString)
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

			ctx := context.WithValue(r.Context(), UserIDKey, userID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserFromContext(r *http.Request) (uuid.UUID, bool) {
	userID, ok := r.Context().Value(UserIDKey).(uuid.UUID)

	return userID, ok
}
