package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/i-christian/fileShare/internal/auth"
	"github.com/i-christian/fileShare/internal/middlewares"
	"github.com/i-christian/fileShare/internal/user"
)

func RegisterRoutes(domain string, aH *auth.AuthHandler, authService *auth.AuthService, apiKeyService *auth.ApiKeyService, uH *user.UserHandler) http.Handler {
	r := chi.NewRouter()

	// Global middlewares
	r.Use(middleware.CleanPath)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middlewares.RateLimit)

	// CORS setup
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{domain},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Route("/api/v1", func(r chi.Router) {
		// Unauthorised routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/signup", aH.Signup)
			r.Post("/login", aH.LoginWithRefresh)
			r.Post("/refresh", aH.Refresh)
		})

		// Authorised routes
		r.Route("/user", func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware(authService, apiKeyService))
			r.Get("/me", uH.MyProfile)
			r.Post("/api-keys", aH.CreateAPIKey)
		})
	})

	return r
}
