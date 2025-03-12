package service

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"

	authctx "github.com/unsavory/silocore-go/internal/auth/context"
	"github.com/unsavory/silocore-go/internal/auth/jwt"
	"golang.org/x/crypto/scrypt"
)

// Common errors
var (
	ErrUnauthorized        = errors.New("unauthorized access")
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrInvalidTenantSwitch = errors.New("invalid tenant switch")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrPasswordTooWeak     = errors.New("password is too weak")
)

// Scrypt parameters
const (
	ScryptN      = 32768 // CPU/memory cost parameter (power of 2)
	ScryptR      = 8     // Block size parameter
	ScryptP      = 1     // Parallelization parameter
	ScryptKeyLen = 32    // Key length
	SaltSize     = 16    // Salt size in bytes
)

// TenantMemberService defines the interface for tenant membership operations
type TenantMemberService interface {
	// GetUserDefaultTenant retrieves a user's default tenant ID (first tenant in membership list)
	GetUserDefaultTenant(ctx context.Context, userID int64) (*int64, error)

	// IsTenantMember checks if a user is a member of a specific tenant
	IsTenantMember(ctx context.Context, userID int64, tenantID int64) (bool, error)
}

// AuthService defines the interface for authentication and authorization operations
type AuthService interface {
	// SwitchTenantContext switches the tenant context for a user
	SwitchTenantContext(ctx context.Context, userID int64, currentToken string, newTenantID *int64) (string, error)

	// ValidateAccess checks if a user has access to a specific resource
	ValidateAccess(ctx context.Context, userID int64, tenantID *int64, requiredRoles []authctx.Role) error

	// BuildAuthContext builds an authentication context with user roles
	BuildAuthContext(ctx context.Context, userID int64, tenantID *int64) (context.Context, error)

	// Login authenticates a user with email and password, returning a JWT token pair
	Login(ctx context.Context, email, password string) (*jwt.TokenPair, int64, error)
}

// DefaultAuthService implements AuthService
type DefaultAuthService struct {
	userService         UserService
	tenantMemberService TenantMemberService
	jwtService          jwt.JWTService
}

// NewDefaultAuthService creates a new DefaultAuthService
func NewDefaultAuthService(userService UserService, tenantMemberService TenantMemberService, jwtService jwt.JWTService) *DefaultAuthService {
	return &DefaultAuthService{
		userService:         userService,
		tenantMemberService: tenantMemberService,
		jwtService:          jwtService,
	}
}

// Login authenticates a user with email and password
func (s *DefaultAuthService) Login(ctx context.Context, email, password string) (*jwt.TokenPair, int64, error) {
	return s.loginWithVerifier(ctx, email, password, VerifyPassword)
}

// loginWithVerifier is a helper method for testing that allows injecting a custom password verification function
func (s *DefaultAuthService) loginWithVerifier(ctx context.Context, email, password string, verifyFunc func(string, string) (bool, error)) (*jwt.TokenPair, int64, error) {
	// Get user by email
	user, err := s.userService.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			log.Printf("[WARN] Login attempt for non-existent user: %s", email)
			return nil, 0, ErrInvalidCredentials
		}
		log.Printf("[ERROR] Database error during login for %s: %v", email, err)
		return nil, 0, err
	}

	// Verify password
	isValid, err := verifyFunc(user.PasswordHash, password)
	if err != nil {
		log.Printf("[ERROR] Error verifying password for user %s: %v", email, err)
		return nil, 0, err
	}

	if !isValid {
		log.Printf("[WARN] Invalid password attempt for user: %s", email)
		return nil, 0, ErrInvalidCredentials
	}

	// Get user's default tenant (if any)
	defaultTenant, err := s.tenantMemberService.GetUserDefaultTenant(ctx, user.ID)
	if err != nil {
		log.Printf("[ERROR] Error getting default tenant for user %s: %v", email, err)
		return nil, 0, err
	}

	if defaultTenant == nil {
		log.Printf("[INFO] User %s has no tenant memberships", email)
	}

	// Generate token pair
	tokenPair, err := s.jwtService.GenerateTokenPair(user.ID, user.Email, defaultTenant)
	if err != nil {
		log.Printf("[ERROR] Error generating token for user %s: %v", email, err)
		return nil, 0, err
	}

	log.Printf("[INFO] User %s successfully authenticated", email)
	return tokenPair, user.ID, nil
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
	isMember, err := s.tenantMemberService.IsTenantMember(ctx, userID, *newTenantID)
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
		isMember, err := s.tenantMemberService.IsTenantMember(ctx, userID, *tenantID)
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

// ValidatePassword checks if a password meets the minimum requirements
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooWeak
	}

	// Additional password strength checks could be added here
	// For example, requiring a mix of uppercase, lowercase, numbers, and special characters

	return nil
}

// VerifyPassword verifies a password against a stored hash
func VerifyPassword(storedHash, password string) (bool, error) {
	// Split the stored hash into salt and hash components
	parts := strings.Split(storedHash, ":")
	if len(parts) != 2 {
		return false, errors.New("invalid hash format")
	}

	// Decode the salt and hash
	salt, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return false, fmt.Errorf("error decoding salt: %w", err)
	}

	storedHashBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false, fmt.Errorf("error decoding hash: %w", err)
	}

	// Hash the provided password with the same salt
	hashedPassword, err := scrypt.Key([]byte(password), salt, ScryptN, ScryptR, ScryptP, ScryptKeyLen)
	if err != nil {
		return false, fmt.Errorf("error hashing password: %w", err)
	}

	// Compare the hashes in constant time to prevent timing attacks
	return subtle.ConstantTimeCompare(storedHashBytes, hashedPassword) == 1, nil
}
