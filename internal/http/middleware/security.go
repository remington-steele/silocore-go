package middleware

import (
	"net/http"
	"strconv"
)

// SecurityConfig holds configuration for security middleware
type SecurityConfig struct {
	XSSProtection             string
	ContentTypeNosniff        string
	XFrameOptions             string
	HSTSMaxAge                int
	HSTSIncludeSubdomains     bool
	ContentSecurityPolicy     string
	ReferrerPolicy            string
	PermissionsPolicy         string
	CrossOriginEmbedderPolicy string
	CrossOriginOpenerPolicy   string
	CrossOriginResourcePolicy string
}

// DefaultSecurityConfig returns a default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		XSSProtection:             "1; mode=block",
		ContentTypeNosniff:        "nosniff",
		XFrameOptions:             "SAMEORIGIN",
		HSTSMaxAge:                31536000, // 1 year
		HSTSIncludeSubdomains:     true,
		ContentSecurityPolicy:     "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		PermissionsPolicy:         "camera=(), microphone=(), geolocation=()",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
	}
}

// Security middleware adds security headers to responses
func Security(config *SecurityConfig) func(http.Handler) http.Handler {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set X-XSS-Protection header
			if config.XSSProtection != "" {
				w.Header().Set("X-XSS-Protection", config.XSSProtection)
			}

			// Set X-Content-Type-Options header
			if config.ContentTypeNosniff != "" {
				w.Header().Set("X-Content-Type-Options", config.ContentTypeNosniff)
			}

			// Set X-Frame-Options header
			if config.XFrameOptions != "" {
				w.Header().Set("X-Frame-Options", config.XFrameOptions)
			}

			// Set Strict-Transport-Security header
			if config.HSTSMaxAge > 0 {
				hstsValue := "max-age=" + strconv.Itoa(config.HSTSMaxAge)
				if config.HSTSIncludeSubdomains {
					hstsValue += "; includeSubDomains"
				}
				w.Header().Set("Strict-Transport-Security", hstsValue)
			}

			// Set Content-Security-Policy header
			if config.ContentSecurityPolicy != "" {
				w.Header().Set("Content-Security-Policy", config.ContentSecurityPolicy)
			}

			// Set Referrer-Policy header
			if config.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", config.ReferrerPolicy)
			}

			// Set Permissions-Policy header
			if config.PermissionsPolicy != "" {
				w.Header().Set("Permissions-Policy", config.PermissionsPolicy)
			}

			// Set Cross-Origin-Embedder-Policy header
			if config.CrossOriginEmbedderPolicy != "" {
				w.Header().Set("Cross-Origin-Embedder-Policy", config.CrossOriginEmbedderPolicy)
			}

			// Set Cross-Origin-Opener-Policy header
			if config.CrossOriginOpenerPolicy != "" {
				w.Header().Set("Cross-Origin-Opener-Policy", config.CrossOriginOpenerPolicy)
			}

			// Set Cross-Origin-Resource-Policy header
			if config.CrossOriginResourcePolicy != "" {
				w.Header().Set("Cross-Origin-Resource-Policy", config.CrossOriginResourcePolicy)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecureHeaders is a simplified version of Security middleware with default settings
func SecureHeaders(next http.Handler) http.Handler {
	return Security(DefaultSecurityConfig())(next)
}
