package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	authctx "github.com/unsavory/silocore-go/internal/auth/context"
	"github.com/unsavory/silocore-go/internal/auth/jwt"
)

// JWTService defines the interface for JWT operations
type JWTService interface {
	ValidateToken(tokenString string) (*jwt.CustomClaims, error)
}

// UserService defines the interface for user role operations
type UserService interface {
	GetUserRoles(ctx context.Context, userID int64) ([]authctx.Role, error)
	GetUserTenantRoles(ctx context.Context, userID int64, tenantID int64) ([]authctx.Role, error)
	IsTenantMember(ctx context.Context, userID int64, tenantID int64) (bool, error)
}

// AuthMiddleware creates middleware for JWT authentication
func AuthMiddleware(jwtService JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenString string

			// First try to extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				// Check if the header has the Bearer prefix
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenString = parts[1]
				}
			}

			// If no token in header, try to extract from cookie
			if tokenString == "" {
				cookie, err := r.Cookie("auth_token")
				if err == nil && cookie.Value != "" {
					tokenString = cookie.Value
				}
			}

			// If no token found, return unauthorized
			if tokenString == "" {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Validate the token
			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Add user information to request context
			ctx := r.Context()
			ctx = authctx.WithUserID(ctx, claims.UserID)
			ctx = authctx.WithUsername(ctx, claims.Username)

			// Add tenant context if present
			if claims.TenantID != nil {
				ctx = authctx.WithTenantID(ctx, claims.TenantID)
			}

			// Continue with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RoleMiddleware creates middleware to fetch and set user roles in the context
func RoleMiddleware(userService UserService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Get user ID from context
			userID, err := authctx.GetUserID(ctx)
			if err != nil {
				http.Error(w, "User ID not found in context", http.StatusUnauthorized)
				return
			}

			// Fetch system-wide roles for the user
			roles, err := userService.GetUserRoles(ctx, userID)
			if err != nil {
				// Log the error but continue with empty roles
				// This allows regular users with no roles to proceed
				roles = []authctx.Role{}
			}

			// Add roles to context (even if empty)
			ctx = authctx.WithRoles(ctx, roles)

			// If tenant context is present, fetch tenant-specific roles
			tenantID, err := authctx.GetTenantID(ctx)
			if err == nil && tenantID != nil {
				// Check if user is a member of this tenant or has admin role
				isMember, err := userService.IsTenantMember(ctx, userID, *tenantID)
				if err != nil {
					// Log the error but assume not a member
					isMember = false
				}

				// Admin users can access any tenant context
				isAdmin := authctx.IsAdmin(ctx)

				if !isMember && !isAdmin {
					// Non-admin users must be members of the tenant they're accessing
					http.Error(w, "Access denied: not a member of this tenant", http.StatusForbidden)
					return
				}

				// Fetch tenant-specific roles only if user is a member
				if isMember {
					tenantRoles, err := userService.GetUserTenantRoles(ctx, userID, *tenantID)
					if err != nil {
						// Log the error but continue with empty tenant roles
						tenantRoles = []authctx.Role{}
					}

					// Merge tenant roles with system roles
					allRoles := append(roles, tenantRoles...)
					ctx = authctx.WithRoles(ctx, allRoles)
				}
			}

			// Continue with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin middleware ensures the user has the ADMIN role
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !authctx.IsAdmin(r.Context()) {
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireTenantContext middleware ensures a tenant context is present
func RequireTenantContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, err := authctx.GetTenantID(r.Context())
		if err != nil || tenantID == nil {
			http.Error(w, "Tenant context required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireTenantSuper middleware ensures the user has the TENANT_SUPER role for the current tenant
func RequireTenantSuper(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// First ensure tenant context exists
		tenantID, err := authctx.GetTenantID(ctx)
		if err != nil || tenantID == nil {
			http.Error(w, "Tenant context required", http.StatusForbidden)
			return
		}

		// Admin users can access any tenant admin functionality
		if authctx.IsAdmin(ctx) {
			next.ServeHTTP(w, r)
			return
		}

		// Then check if user has TENANT_SUPER role
		if !authctx.IsTenantSuper(ctx) {
			http.Error(w, "Tenant super access required", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireTenantMember middleware ensures the user is a member of the current tenant
// This is less restrictive than RequireTenantSuper
func RequireTenantMember(userService UserService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// First ensure tenant context exists
			tenantID, err := authctx.GetTenantID(ctx)
			if err != nil || tenantID == nil {
				http.Error(w, "Tenant context required", http.StatusForbidden)
				return
			}

			// Get user ID from context
			userID, err := authctx.GetUserID(ctx)
			if err != nil {
				http.Error(w, "User ID not found in context", http.StatusUnauthorized)
				return
			}

			// Admin users can access any tenant
			if authctx.IsAdmin(ctx) {
				next.ServeHTTP(w, r)
				return
			}

			// Check if user is a member of this tenant
			isMember, err := userService.IsTenantMember(ctx, userID, *tenantID)
			if err != nil {
				http.Error(w, "Failed to verify tenant membership", http.StatusInternalServerError)
				return
			}

			if !isMember {
				http.Error(w, "Access denied: not a member of this tenant", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// TenantIDFromURL extracts the tenant ID from the URL parameter and adds it to the context
func TenantIDFromURL(paramName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantIDStr := chi.URLParam(r, paramName)
			if tenantIDStr == "" {
				http.Error(w, "Tenant ID parameter is required", http.StatusBadRequest)
				return
			}

			// Convert tenantIDStr to int64
			tenantID, err := strconv.ParseInt(tenantIDStr, 10, 64)
			if err != nil {
				http.Error(w, "Invalid tenant ID format", http.StatusBadRequest)
				return
			}

			// Set tenant ID in context
			ctx := r.Context()
			ctx = authctx.WithTenantID(ctx, &tenantID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
