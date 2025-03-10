package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Common errors
var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrExpiredToken      = errors.New("token has expired")
	ErrMissingClaim      = errors.New("missing required claim")
	ErrInvalidSigningKey = errors.New("invalid signing key")
)

// Service provides JWT token operations
type Service struct {
	config Config
}

// Ensure Service implements JWTService
var _ JWTService = (*Service)(nil)

// NewService creates a new JWT service with the provided configuration
func NewService(config Config) *Service {
	return &Service{
		config: config,
	}
}

// GenerateTokenPair creates a new access and refresh token pair for a user
func (s *Service) GenerateTokenPair(userID int64, username string, tenantID *int64) (*TokenPair, error) {
	// Generate access token
	accessToken, accessExpiry, err := s.generateToken(userID, username, tenantID, s.config.AccessExpiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token (without tenant context for security)
	refreshToken, _, err := s.generateToken(userID, username, nil, s.config.RefreshExpiration)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(time.Until(accessExpiry).Seconds()),
	}, nil
}

// generateToken creates a new JWT token with the provided claims
func (s *Service) generateToken(userID int64, username string, tenantID *int64, expirationSeconds int64) (string, time.Time, error) {
	now := time.Now()
	expiryTime := now.Add(time.Duration(expirationSeconds) * time.Second)

	claims := CustomClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiryTime),
		},
		UserID:   userID,
		Username: username,
		TenantID: tenantID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(s.config.Secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, expiryTime, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *Service) ValidateToken(tokenString string) (*CustomClaims, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method: %v", ErrInvalidToken, token.Header["alg"])
		}
		return []byte(s.config.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Extract claims
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Validate required claims
	if claims.UserID == 0 {
		return nil, fmt.Errorf("%w: user_id", ErrMissingClaim)
	}

	return claims, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *Service) RefreshToken(refreshToken string, tenantID *int64) (*TokenPair, error) {
	// Parse the refresh token
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Generate a new token pair
	return s.GenerateTokenPair(claims.UserID, claims.Username, tenantID)
}

// SwitchTenantContext generates a new access token with a different tenant context
func (s *Service) SwitchTenantContext(currentToken string, newTenantID *int64) (string, error) {
	// Validate the current token
	claims, err := s.ValidateToken(currentToken)
	if err != nil {
		return "", err
	}

	// Generate a new token with the new tenant context
	token, _, err := s.generateToken(claims.UserID, claims.Username, newTenantID, s.config.AccessExpiration)
	if err != nil {
		return "", fmt.Errorf("failed to generate token with new tenant context: %w", err)
	}

	return token, nil
}
