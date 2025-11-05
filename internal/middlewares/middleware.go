package middlewares

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/i-christian/fileShare/internal/auth"
	"github.com/i-christian/fileShare/internal/utils"
	"github.com/i-christian/fileShare/internal/utils/security"
	"golang.org/x/time/rate"
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
				user, err := authService.ValidateToken(credential)
				if err != nil {
					if errors.Is(err, auth.ErrExpiredToken) {
						utils.UnauthorisedResponse(w, "token has expired")
					} else {
						utils.UnauthorisedResponse(w, "invalid or expired token")
					}
					return
				}

				req := security.SetContextUser(r, user)

				next.ServeHTTP(w, req)

			case "ApiKey":
				user, err := apiKeyService.ValidateAPIKey(r.Context(), credential)
				if err != nil {
					if errors.Is(err, utils.ErrUnexpectedError) {
						utils.UnauthorisedResponse(w, utils.ErrUnexpectedError.Error())
						utils.WriteServerError(authService.Logger, "api key authorisation failed", err)
						return

					} else {
						utils.UnauthorisedResponse(w, err.Error())
					}
					return
				}

				req := security.SetContextUser(r, user)
				next.ServeHTTP(w, req)

			default:
				utils.UnauthorisedResponse(w, "unsupported authorization scheme")
				return
			}
		})
	}
}

// Limiter is a config struct for rate limit middleware
type Limiter struct {
	Rps     float64
	Burst   int
	Enabled bool
}

// RateLimit middleware sets the Ip-based request rate limits
func RateLimit(rps float64, burst int, enabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		type client struct {
			limiter  *rate.Limiter
			lastSeen time.Time
		}

		var (
			mu      sync.Mutex
			clients = make(map[string]*client)
		)

		go func() {
			for {
				time.Sleep(time.Minute)
				mu.Lock()

				for ip, client := range clients {
					if time.Since(client.lastSeen) > 3*time.Minute {
						delete(clients, ip)
					}
				}

				mu.Unlock()
			}
		}()

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if enabled {
				ip, _ := security.GetIPAddress(r)

				mu.Lock()

				if _, found := clients[ip]; !found {
					clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
				}

				clients[ip].lastSeen = time.Now()

				if !clients[ip].limiter.Allow() {
					mu.Unlock()
					utils.RateLimitExcededResponse(w)
					return
				}

				mu.Unlock()
			}

			next.ServeHTTP(w, r)
		})
	}
}
