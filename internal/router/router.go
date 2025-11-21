package router

import (
	"expvar"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/i-christian/fileShare/internal/auth"
	"github.com/i-christian/fileShare/internal/files"
	"github.com/i-christian/fileShare/internal/middlewares"
	"github.com/i-christian/fileShare/internal/public"
	"github.com/i-christian/fileShare/internal/user"
)

type RoutesConfig struct {
	Domain         string
	Rps            float64
	Burst          int
	LimiterEnabled bool
}

func RegisterRoutes(config *RoutesConfig, aH *auth.AuthHandler, authService *auth.AuthService, apiKeyService *auth.ApiKeyService, uH *user.UserHandler, pH *public.PublicHandler, fH *files.FileHandler) http.Handler {
	r := chi.NewRouter()

	// Global middlewares
	r.Use(middlewares.Metrics)
	r.Use(middleware.CleanPath)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middlewares.RateLimit(config.Rps, config.Burst, config.LimiterEnabled))

	// CORS setup
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{config.Domain},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Handle("/debug/vars", expvar.Handler())

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/healthcheck", pH.HealthStatus)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/signup", aH.Signup)
			r.Post("/login", aH.LoginWithRefresh)
			r.Post("/refresh", aH.Refresh)
		})

		r.Route("/user", func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware(authService, apiKeyService))
			r.Put("/activated", uH.ActivateUserHandler)

			r.Group(func(r chi.Router) {
				r.Use(middlewares.RequireActivatedUser)
				r.Get("/me", uH.MyProfile)
				r.Post("/api-keys", aH.CreateAPIKey)
			})
		})

		r.Route("/files", func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware(authService, apiKeyService))
			r.Get("/", fH.ListPublicFiles)
			r.Get("/{id}/download", fH.Download)

			r.Group(func(r chi.Router) {
				r.Use(middlewares.RequireActivatedUser)

				r.Post("/upload", fH.Upload)
				r.Get("/me", fH.ListMyFiles)
				r.Get("/{id}", fH.GetMetadata)
				r.Put("/{id}", fH.Delete)
			})
		})
	})

	return r
}
