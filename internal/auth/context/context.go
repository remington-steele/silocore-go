package context

import (
	"context"
	"errors"
)

// Key type for context values
type contextKey string

// Context keys
const (
	userIDKey   contextKey = "user_id"
	tenantIDKey contextKey = "tenant_id"
	usernameKey contextKey = "username"
	rolesKey    contextKey = "roles"
)

// Common errors
var (
	ErrNoUserID   = errors.New("user ID not found in context")
	ErrNoTenantID = errors.New("tenant ID not found in context")
	ErrNoUsername = errors.New("username not found in context")
	ErrNoRoles    = errors.New("roles not found in context")
)

// Role represents a system role
type Role string

// System roles
const (
	RoleAdmin       Role = "ADMIN"
	RoleInternal    Role = "INTERNAL"
	RoleTenantSuper Role = "TENANT_SUPER"
)

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID retrieves the user ID from the context
func GetUserID(ctx context.Context) (int64, error) {
	userID, ok := ctx.Value(userIDKey).(int64)
	if !ok {
		return 0, ErrNoUserID
	}
	return userID, nil
}

// WithTenantID adds a tenant ID to the context
func WithTenantID(ctx context.Context, tenantID *int64) context.Context {
	return context.WithValue(ctx, tenantIDKey, tenantID)
}

// GetTenantID retrieves the tenant ID from the context
func GetTenantID(ctx context.Context) (*int64, error) {
	tenantID, ok := ctx.Value(tenantIDKey).(*int64)
	if !ok {
		return nil, ErrNoTenantID
	}
	return tenantID, nil
}

// WithUsername adds a username to the context
func WithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey, username)
}

// GetUsername retrieves the username from the context
func GetUsername(ctx context.Context) (string, error) {
	username, ok := ctx.Value(usernameKey).(string)
	if !ok {
		return "", ErrNoUsername
	}
	return username, nil
}

// WithRoles adds roles to the context
func WithRoles(ctx context.Context, roles []Role) context.Context {
	return context.WithValue(ctx, rolesKey, roles)
}

// GetRoles retrieves the roles from the context
func GetRoles(ctx context.Context) ([]Role, error) {
	roles, ok := ctx.Value(rolesKey).([]Role)
	if !ok {
		return nil, ErrNoRoles
	}
	return roles, nil
}

// HasRole checks if the context has a specific role
func HasRole(ctx context.Context, role Role) bool {
	roles, err := GetRoles(ctx)
	if err != nil {
		return false
	}

	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

// IsAdmin checks if the context has the ADMIN role
func IsAdmin(ctx context.Context) bool {
	return HasRole(ctx, RoleAdmin)
}

// IsTenantSuper checks if the context has the TENANT_SUPER role
func IsTenantSuper(ctx context.Context) bool {
	return HasRole(ctx, RoleTenantSuper)
}

// IsInternal checks if the context has the INTERNAL role
func IsInternal(ctx context.Context) bool {
	return HasRole(ctx, RoleInternal)
}
