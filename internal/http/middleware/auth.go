package middleware

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	authctx "github.com/unsavory/silocore-go/internal/auth/context"
	"github.com/unsavory/silocore-go/internal/auth/jwt"
	"github.com/unsavory/silocore-go/internal/auth/service"
	tenantservice "github.com/unsavory/silocore-go/internal/tenant/service"
)

// JWTService defines the interface for JWT operations
type JWTService interface {
	ValidateToken(tokenString string) (*jwt.CustomClaims, error)
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
					log.Printf("[DEBUG] Token extracted from Authorization header: %s", r.URL.Path)
				}
			}

			// If no token in header, try to extract from cookie
			if tokenString == "" {
				cookie, err := r.Cookie("auth_token")
				if err == nil && cookie.Value != "" {
					tokenString = cookie.Value
					log.Printf("[DEBUG] Token extracted from cookie: %s", r.URL.Path)
				}
			}

			// If no token found, return unauthorized
			if tokenString == "" {
				log.Printf("[WARN] Authentication required but no token found: %s %s", r.Method, r.URL.Path)
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Validate the token
			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				log.Printf("[WARN] Invalid or expired token: %s %s - %v", r.Method, r.URL.Path, err)
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
				log.Printf("[DEBUG] User ID %d authenticated with tenant context %d: %s", claims.UserID, *claims.TenantID, r.URL.Path)
			} else {
				log.Printf("[DEBUG] User ID %d authenticated without tenant context: %s", claims.UserID, r.URL.Path)
			}

			// Continue with the updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RoleMiddleware creates middleware to fetch and set user roles in the context
func RoleMiddleware(userService service.UserService, tenantMemberService tenantservice.TenantMemberService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Get user ID from context
			userID, err := authctx.GetUserID(ctx)
			if err != nil {
				log.Printf("[ERROR] User ID not found in context: %s %s", r.Method, r.URL.Path)
				http.Error(w, "User ID not found in context", http.StatusUnauthorized)
				return
			}

			// Fetch user's system-wide roles
			roles, err := userService.GetUserRoles(ctx, userID)
			if err != nil {
				log.Printf("[ERROR] Failed to fetch roles for user ID %d: %v", userID, err)
				roles = []authctx.Role{}
			} else {
				log.Printf("[DEBUG] Fetched %d system roles for user ID %d", len(roles), userID)
			}

			// Add roles to context (even if empty)
			ctx = authctx.WithRoles(ctx, roles)

			// If tenant context is present, fetch tenant-specific roles
			tenantID, err := authctx.GetTenantID(ctx)
			if err == nil && tenantID != nil {
				log.Printf("[DEBUG] Processing tenant context %d for user ID %d", *tenantID, userID)

				// Check if user is a member of this tenant or has admin role
				isMember, err := tenantMemberService.IsTenantMember(ctx, userID, *tenantID)
				if err != nil {
					// Log the error but assume not a member
					log.Printf("[WARN] Failed to verify tenant membership for user ID %d, tenant ID %d: %v", userID, *tenantID, err)
					isMember = false
				}

				// Admin users can access any tenant context
				isAdmin := authctx.IsAdmin(ctx)

				if !isMember && !isAdmin {
					// Non-admin users must be members of the tenant they're accessing
					log.Printf("[WARN] Access denied: User ID %d is not a member of tenant ID %d and is not an admin", userID, *tenantID)
					http.Error(w, "Access denied: not a member of this tenant", http.StatusForbidden)
					return
				}

				// Fetch tenant-specific roles
				tenantRoles, err := userService.GetUserTenantRoles(ctx, userID, *tenantID)
				if err != nil {
					log.Printf("[ERROR] Failed to fetch tenant roles for user ID %d, tenant ID %d: %v", userID, *tenantID, err)
				} else {
					log.Printf("[DEBUG] Fetched %d tenant roles for user ID %d, tenant ID %d", len(tenantRoles), userID, *tenantID)
					// Add tenant roles to existing roles
					roles = append(roles, tenantRoles...)
					// Update roles in context
					ctx = authctx.WithRoles(ctx, roles)
				}
			}

			// Continue with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdmin middleware ensures the user has the ADMIN role
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID, _ := authctx.GetUserID(ctx)

		if !authctx.IsAdmin(ctx) {
			log.Printf("[WARN] Admin access required but user ID %d does not have admin role: %s %s", userID, r.Method, r.URL.Path)
			http.Error(w, "Admin access required", http.StatusForbidden)
			return
		}

		log.Printf("[DEBUG] Admin access granted to user ID %d: %s %s", userID, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// RequireTenantContext middleware ensures a tenant context is present
func RequireTenantContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID, _ := authctx.GetUserID(ctx)

		tenantID, err := authctx.GetTenantID(ctx)
		if err != nil || tenantID == nil {
			log.Printf("[WARN] Tenant context required but not found for user ID %d: %s %s", userID, r.Method, r.URL.Path)
			http.Error(w, "Tenant context required", http.StatusForbidden)
			return
		}

		log.Printf("[DEBUG] Tenant context %d verified for user ID %d: %s %s", *tenantID, userID, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// RequireTenantSuper middleware ensures the user has the TENANT_SUPER role for the current tenant
func RequireTenantSuper(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID, _ := authctx.GetUserID(ctx)

		// First ensure tenant context exists
		tenantID, err := authctx.GetTenantID(ctx)
		if err != nil || tenantID == nil {
			log.Printf("[WARN] Tenant context required but not found for user ID %d: %s %s", userID, r.Method, r.URL.Path)
			http.Error(w, "Tenant context required", http.StatusForbidden)
			return
		}

		// Admin users can access any tenant admin functionality
		if authctx.IsAdmin(ctx) {
			log.Printf("[DEBUG] Admin user ID %d granted tenant super access for tenant ID %d: %s %s", userID, *tenantID, r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
			return
		}

		// Then check if user has TENANT_SUPER role
		if !authctx.IsTenantSuper(ctx) {
			log.Printf("[WARN] Tenant super access required but user ID %d does not have the role for tenant ID %d: %s %s", userID, *tenantID, r.Method, r.URL.Path)
			http.Error(w, "Tenant super access required", http.StatusForbidden)
			return
		}

		log.Printf("[DEBUG] Tenant super access granted to user ID %d for tenant ID %d: %s %s", userID, *tenantID, r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// RequireTenantMember middleware ensures the user is a member of the current tenant
// This is less restrictive than RequireTenantSuper
func RequireTenantMember(tenantMemberService tenantservice.TenantMemberService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// First ensure tenant context exists
			tenantID, err := authctx.GetTenantID(ctx)
			if err != nil || tenantID == nil {
				log.Printf("[WARN] Tenant context required but not found: %s %s", r.Method, r.URL.Path)
				http.Error(w, "Tenant context required", http.StatusForbidden)
				return
			}

			// Get user ID from context
			userID, err := authctx.GetUserID(ctx)
			if err != nil {
				log.Printf("[ERROR] User ID not found in context: %s %s", r.Method, r.URL.Path)
				http.Error(w, "User ID not found in context", http.StatusUnauthorized)
				return
			}

			// Admin users can access any tenant
			if authctx.IsAdmin(ctx) {
				log.Printf("[DEBUG] Admin user ID %d granted tenant member access for tenant ID %d: %s %s", userID, *tenantID, r.Method, r.URL.Path)
				next.ServeHTTP(w, r)
				return
			}

			// Check if user is a member of this tenant
			isMember, err := tenantMemberService.IsTenantMember(ctx, userID, *tenantID)
			if err != nil {
				log.Printf("[ERROR] Failed to verify tenant membership for user ID %d, tenant ID %d: %v", userID, *tenantID, err)
				http.Error(w, "Failed to verify tenant membership", http.StatusInternalServerError)
				return
			}

			if !isMember {
				log.Printf("[WARN] Access denied: User ID %d is not a member of tenant ID %d: %s %s", userID, *tenantID, r.Method, r.URL.Path)
				http.Error(w, "Access denied: not a member of this tenant", http.StatusForbidden)
				return
			}

			// User is a member of this tenant, continue
			log.Printf("[DEBUG] User ID %d verified as member of tenant ID %d: %s %s", userID, *tenantID, r.Method, r.URL.Path)
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
				log.Printf("[WARN] Tenant ID parameter '%s' is required but not found: %s %s", paramName, r.Method, r.URL.Path)
				http.Error(w, "Tenant ID parameter is required", http.StatusBadRequest)
				return
			}

			// Convert tenantIDStr to int64
			tenantID, err := strconv.ParseInt(tenantIDStr, 10, 64)
			if err != nil {
				log.Printf("[WARN] Invalid tenant ID format '%s': %s %s - %v", tenantIDStr, r.Method, r.URL.Path, err)
				http.Error(w, "Invalid tenant ID format", http.StatusBadRequest)
				return
			}

			// Set tenant ID in context
			ctx := r.Context()
			ctx = authctx.WithTenantID(ctx, &tenantID)
			log.Printf("[DEBUG] Tenant ID %d extracted from URL parameter '%s': %s %s", tenantID, paramName, r.Method, r.URL.Path)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
