package jwt

import (
	"github.com/golang-jwt/jwt/v5"
)

// JWTService defines the interface for JWT operations
type JWTService interface {
	// GenerateTokenPair creates a new access and refresh token pair for a user
	GenerateTokenPair(userID int64, username string, tenantID *int64) (*TokenPair, error)

	// ValidateToken validates a token and returns its claims
	ValidateToken(tokenString string) (*CustomClaims, error)

	// RefreshToken refreshes an access token using a refresh token
	RefreshToken(refreshToken string, tenantID *int64) (*TokenPair, error)

	// SwitchTenantContext switches the tenant context in a token
	SwitchTenantContext(currentToken string, newTenantID *int64) (string, error)
}

// CustomClaims extends the standard JWT claims with our custom claims
type CustomClaims struct {
	jwt.RegisteredClaims
	UserID   int64  `json:"user_id"`
	TenantID *int64 `json:"tenant_id,omitempty"` // Optional tenant context
	Username string `json:"username"`
}

// TokenPair represents an access token and refresh token pair
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // Expiration time in seconds
}

// Config holds JWT configuration settings
type Config struct {
	Secret            string
	AccessExpiration  int64
	RefreshExpiration int64
	Issuer            string
}
