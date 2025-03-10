package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	authctx "github.com/unsavory/silocore-go/internal/auth/context"
	"github.com/unsavory/silocore-go/internal/auth/jwt"
)

// Common errors
var (
	ErrUnauthorized        = errors.New("unauthorized access")
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrInvalidTenantSwitch = errors.New("invalid tenant switch")
)

// AuthService defines the interface for authentication and authorization operations
type AuthService interface {
	// SwitchTenantContext switches the tenant context for a user
	SwitchTenantContext(ctx context.Context, userID int64, currentToken string, newTenantID *int64) (string, error)

	// ValidateAccess checks if a user has access to a specific resource
	ValidateAccess(ctx context.Context, userID int64, tenantID *int64, requiredRoles []authctx.Role) error

	// BuildAuthContext builds an authentication context with user roles
	BuildAuthContext(ctx context.Context, userID int64, tenantID *int64) (context.Context, error)
}

// DefaultAuthService implements AuthService
type DefaultAuthService struct {
	userService UserService
	jwtService  jwt.JWTService
}

// NewDefaultAuthService creates a new DefaultAuthService
func NewDefaultAuthService(userService UserService, jwtService jwt.JWTService) *DefaultAuthService {
	return &DefaultAuthService{
		userService: userService,
		jwtService:  jwtService,
	}
}

// SwitchTenantContext switches the tenant context for a user
func (s *DefaultAuthService) SwitchTenantContext(ctx context.Context, userID int64, currentToken string, newTenantID *int64) (string, error) {
	// If switching to no tenant context (global access)
	if newTenantID == nil {
		// Check if user has admin role which allows global access
		roles, err := s.userService.GetUserRoles(ctx, userID)
		if err != nil {
			return "", fmt.Errorf("failed to get user roles: %w", err)
		}

		hasAdminRole := false
		for _, role := range roles {
			if role == authctx.RoleAdmin {
				hasAdminRole = true
				break
			}
		}

		if !hasAdminRole {
			return "", ErrUnauthorized
		}

		// Generate new token without tenant context
		return s.jwtService.SwitchTenantContext(currentToken, nil)
	}

	// Check if user is a member of the requested tenant
	isMember, err := s.userService.IsTenantMember(ctx, userID, *newTenantID)
	if err != nil {
		return "", fmt.Errorf("failed to check tenant membership: %w", err)
	}

	if !isMember {
		return "", ErrUnauthorized
	}

	// Generate new token with the new tenant context
	return s.jwtService.SwitchTenantContext(currentToken, newTenantID)
}

// ValidateAccess checks if a user has access to a specific resource
func (s *DefaultAuthService) ValidateAccess(ctx context.Context, userID int64, tenantID *int64, requiredRoles []authctx.Role) error {
	// Get user's system-wide roles
	systemRoles, err := s.userService.GetUserRoles(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user roles: %w", err)
	}

	// Admin role has access to everything
	for _, role := range systemRoles {
		if role == authctx.RoleAdmin {
			return nil
		}
	}

	// If tenant-specific access is required
	if tenantID != nil {
		// Check if user is a member of the tenant
		isMember, err := s.userService.IsTenantMember(ctx, userID, *tenantID)
		if err != nil {
			return fmt.Errorf("failed to check tenant membership: %w", err)
		}

		if !isMember {
			return ErrUnauthorized
		}

		// If specific roles are required, check tenant-specific roles
		if len(requiredRoles) > 0 {
			tenantRoles, err := s.userService.GetUserTenantRoles(ctx, userID, *tenantID)
			if err != nil {
				return fmt.Errorf("failed to get tenant roles: %w", err)
			}

			// Check if user has any of the required roles
			hasRequiredRole := false
			for _, required := range requiredRoles {
				for _, role := range tenantRoles {
					if role == required {
						hasRequiredRole = true
						break
					}
				}
				if hasRequiredRole {
					break
				}
			}

			if !hasRequiredRole {
				return ErrUnauthorized
			}
		}
	}

	return nil
}

// BuildAuthContext builds an authentication context with user roles
func (s *DefaultAuthService) BuildAuthContext(ctx context.Context, userID int64, tenantID *int64) (context.Context, error) {
	// Add user ID to context
	ctx = authctx.WithUserID(ctx, userID)

	// Add tenant ID to context if provided
	if tenantID != nil {
		ctx = authctx.WithTenantID(ctx, tenantID)
	}

	// Get user's system-wide roles
	systemRoles, err := s.userService.GetUserRoles(ctx, userID)
	if err != nil {
		log.Printf("Failed to get user roles: %v", err)
		return ctx, fmt.Errorf("failed to get user roles: %w", err)
	}

	// If tenant context is provided, get tenant-specific roles
	var allRoles []authctx.Role
	allRoles = append(allRoles, systemRoles...)

	if tenantID != nil {
		tenantRoles, err := s.userService.GetUserTenantRoles(ctx, userID, *tenantID)
		if err != nil {
			log.Printf("Failed to get tenant roles: %v", err)
			return ctx, fmt.Errorf("failed to get tenant roles: %w", err)
		}
		allRoles = append(allRoles, tenantRoles...)
	}

	// Add roles to context
	ctx = authctx.WithRoles(ctx, allRoles)

	return ctx, nil
}
