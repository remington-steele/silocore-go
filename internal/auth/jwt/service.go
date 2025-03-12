package jwt

import (
	"errors"
	"fmt"
	"log"
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
	log.Printf("[INFO] Initializing JWT service with issuer: %s", config.Issuer)
	return &Service{
		config: config,
	}
}

// GenerateTokenPair creates a new access and refresh token pair for a user
func (s *Service) GenerateTokenPair(userID int64, username string, tenantID *int64) (*TokenPair, error) {
	// Generate access token
	log.Printf("[DEBUG] Generating access token for user ID %d, username %s", userID, username)
	accessToken, accessExpiry, err := s.generateToken(userID, username, tenantID, s.config.AccessExpiration)
	if err != nil {
		log.Printf("[ERROR] Failed to generate access token for user ID %d: %v", userID, err)
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token (without tenant context for security)
	log.Printf("[DEBUG] Generating refresh token for user ID %d", userID)
	refreshToken, _, err := s.generateToken(userID, username, nil, s.config.RefreshExpiration)
	if err != nil {
		log.Printf("[ERROR] Failed to generate refresh token for user ID %d: %v", userID, err)
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	expiresIn := int64(time.Until(accessExpiry).Seconds())
	log.Printf("[INFO] Generated token pair for user ID %d, expires in %d seconds", userID, expiresIn)

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
	}, nil
}

// generateToken creates a new JWT token with the provided claims
func (s *Service) generateToken(userID int64, username string, tenantID *int64, expirationSeconds int64) (string, time.Time, error) {
	now := time.Now()
	expiryTime := now.Add(time.Duration(expirationSeconds) * time.Second)

	tenantIDLog := "<nil>"
	if tenantID != nil {
		tenantIDLog = fmt.Sprintf("%d", *tenantID)
	}
	log.Printf("[DEBUG] Creating token with claims: userID=%d, username=%s, tenantID=%s, expiry=%s",
		userID, username, tenantIDLog, expiryTime.Format(time.RFC3339))

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
		log.Printf("[ERROR] Failed to sign token for user ID %d: %v", userID, err)
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	log.Printf("[DEBUG] Successfully signed token for user ID %d", userID)
	return signedToken, expiryTime, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *Service) ValidateToken(tokenString string) (*CustomClaims, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			log.Printf("[WARN] Token validation failed: unexpected signing method: %v", token.Header["alg"])
			return nil, fmt.Errorf("%w: unexpected signing method: %v", ErrInvalidToken, token.Header["alg"])
		}
		return []byte(s.config.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			log.Printf("[WARN] Token validation failed: token has expired")
			return nil, ErrExpiredToken
		}
		log.Printf("[WARN] Token validation failed: %v", err)
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	// Extract claims
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		log.Printf("[WARN] Token validation failed: invalid claims or token")
		return nil, ErrInvalidToken
	}

	// Validate required claims
	if claims.UserID == 0 {
		log.Printf("[WARN] Token validation failed: missing required claim: user_id")
		return nil, fmt.Errorf("%w: user_id", ErrMissingClaim)
	}

	tenantIDLog := "<nil>"
	if claims.TenantID != nil {
		tenantIDLog = fmt.Sprintf("%d", *claims.TenantID)
	}
	log.Printf("[DEBUG] Token validated successfully for user ID %d, username %s, tenant ID %s",
		claims.UserID, claims.Username, tenantIDLog)

	return claims, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *Service) RefreshToken(refreshToken string, tenantID *int64) (*TokenPair, error) {
	// Parse the refresh token
	log.Printf("[DEBUG] Attempting to refresh token with tenant ID: %v", tenantID)
	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		log.Printf("[WARN] Token refresh failed: invalid refresh token: %v", err)
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	log.Printf("[INFO] Refreshing token for user ID %d, username %s", claims.UserID, claims.Username)

	// Generate a new token pair
	return s.GenerateTokenPair(claims.UserID, claims.Username, tenantID)
}

// SwitchTenantContext generates a new access token with a different tenant context
func (s *Service) SwitchTenantContext(currentToken string, newTenantID *int64) (string, error) {
	// Validate the current token
	tenantIDLog := "<nil>"
	if newTenantID != nil {
		tenantIDLog = fmt.Sprintf("%d", *newTenantID)
	}
	log.Printf("[DEBUG] Attempting to switch tenant context to %s", tenantIDLog)

	claims, err := s.ValidateToken(currentToken)
	if err != nil {
		log.Printf("[WARN] Tenant context switch failed: invalid token: %v", err)
		return "", err
	}

	// Generate a new token with the new tenant context
	log.Printf("[INFO] Switching tenant context for user ID %d from %v to %v",
		claims.UserID, claims.TenantID, newTenantID)

	token, _, err := s.generateToken(claims.UserID, claims.Username, newTenantID, s.config.AccessExpiration)
	if err != nil {
		log.Printf("[ERROR] Failed to generate token with new tenant context for user ID %d: %v", claims.UserID, err)
		return "", fmt.Errorf("failed to generate token with new tenant context: %w", err)
	}

	log.Printf("[INFO] Successfully switched tenant context for user ID %d to %s", claims.UserID, tenantIDLog)
	return token, nil
}
