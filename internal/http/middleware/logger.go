package middleware

import (
	"log"
	"net/http"
	"time"
)

// Logger is a middleware that logs HTTP requests
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer to capture the status code
		crw := &customResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default status code
		}

		// Process the request
		next.ServeHTTP(crw, r)

		// Calculate request duration
		duration := time.Since(start)

		// Log the request details
		log.Printf(
			"[INFO] %s - %s %s %d %s",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			crw.statusCode,
			duration,
		)
	})
}

// customResponseWriter is a wrapper for http.ResponseWriter that captures the status code
type customResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before writing it
func (crw *customResponseWriter) WriteHeader(code int) {
	crw.statusCode = code
	crw.ResponseWriter.WriteHeader(code)
}

// Write captures the default status code (200) if WriteHeader hasn't been called
func (crw *customResponseWriter) Write(b []byte) (int, error) {
	return crw.ResponseWriter.Write(b)
}
