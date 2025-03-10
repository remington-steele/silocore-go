package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// Options contains configuration for the router
type Options struct {
	EnableCORS        bool
	EnableCompression bool
	Timeout           time.Duration
	Dependencies      RouterDependencies
}

// DefaultOptions returns the default router options
func DefaultOptions() Options {
	return Options{
		EnableCORS:        true,
		EnableCompression: true,
		Timeout:           60 * time.Second,
		Dependencies:      RouterDependencies{}, // This should be provided by the caller
	}
}

// New creates a new Chi router with the given options
func New(opts Options) *chi.Mux {
	r := chi.NewRouter()

	// Apply global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(opts.Timeout))

	if opts.EnableCompression {
		r.Use(middleware.Compress(5))
	}

	if opts.EnableCORS {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"}, // For development; restrict in production
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300, // Maximum value not readily exceeded by browsers
		}))
	}

	// Health check endpoint (no auth required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return r
}
