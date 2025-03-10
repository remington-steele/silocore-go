package transaction

import (
	"log"
	"net/http"

	authctx "github.com/unsavory/silocore-go/internal/auth/context"
)

// Middleware creates middleware for transaction management
func (m *Manager) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Start a new transaction
			ctx, tx, err := m.Begin(r.Context())
			if err != nil {
				log.Printf("Error starting transaction: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Create a response writer that captures the status code
			rw := newResponseWriter(w)

			// Set tenant context if available
			tenantID, err := authctx.GetTenantID(ctx)
			if err == nil && tenantID != nil {
				if err := m.SetTenantContext(ctx, *tenantID); err != nil {
					log.Printf("Error setting tenant context: %v", err)
					tx.Rollback()
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}
			}

			// Update the request with the new context
			r = r.WithContext(ctx)

			// Call the next handler
			defer func() {
				// Recover from panics
				if rec := recover(); rec != nil {
					log.Printf("Panic in handler: %v", rec)
					tx.Rollback()
					panic(rec) // Re-panic after rollback
				}

				// Clear tenant context
				if tenantID != nil {
					if err := m.ClearTenantContext(ctx); err != nil {
						log.Printf("Error clearing tenant context: %v", err)
					}
				}

				// Commit or rollback based on the response status
				if rw.statusCode >= 200 && rw.statusCode < 500 {
					// Success or client error, commit the transaction
					if err := tx.Commit(); err != nil {
						log.Printf("Error committing transaction: %v", err)
						http.Error(w, "Internal server error", http.StatusInternalServerError)
					}
				} else {
					// Server error, rollback the transaction
					if err := tx.Rollback(); err != nil {
						log.Printf("Error rolling back transaction: %v", err)
					}
				}
			}()

			// Serve the request
			next.ServeHTTP(rw, r)
		})
	}
}

// responseWriter is a wrapper around http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// newResponseWriter creates a new responseWriter
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

// WriteHeader captures the status code and calls the underlying WriteHeader
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush implements the http.Flusher interface
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements the http.Hijacker interface
func (rw *responseWriter) Hijack() (interface{}, interface{}, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}
