package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/cors"
	"github.com/go-chi/chi/v5/middleware"
)

func registerRoutes(domain string) http.Handler {
	r := chi.NewRouter()

	// Global middlewares
	r.Use(middleware.CleanPath)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

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
			r.Post("/signup", nil)
			r.Post("/login", nil)
			r.With(nil).Post("/refresh", nil)
		})

		// Authorised routes
		r.Group(func(r chi.Router) {
			r.Use(nil)
			r.Get("/me", nil)
		})
	})

	return r
}
