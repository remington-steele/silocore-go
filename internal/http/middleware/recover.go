package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

// Recover is a middleware that recovers from panics and logs the error
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log the error and stack trace
				log.Printf("[ERROR] Panic recovered: %v\n%s", err, debug.Stack())

				// Return a 500 Internal Server Error response
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, "Internal Server Error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
