package jwt

import (
	"fmt"
	"os"
	"strconv"
)

const (
	// Default values
	defaultExpiration int64 = 86400 // 24 hours in seconds
	defaultIssuer           = "silocore"

	// Environment variable names
	envJWTSecret         = "JWT_SECRET"
	envJWTExpirationSecs = "JWT_EXPIRATION_SECONDS"
	envJWTRefreshExpSecs = "JWT_REFRESH_EXPIRATION_SECONDS"
	envJWTIssuer         = "JWT_ISSUER"
)

// LoadConfig loads JWT configuration from environment variables
func LoadConfig() (Config, error) {
	// Get JWT secret (required)
	secret := os.Getenv(envJWTSecret)
	if secret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	// Get JWT expiration (optional, default to 24 hours)
	accessExpStr := os.Getenv(envJWTExpirationSecs)
	var accessExp int64 = defaultExpiration
	if accessExpStr != "" {
		var err error
		accessExp, err = strconv.ParseInt(accessExpStr, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("invalid JWT_EXPIRATION_SECONDS value: %w", err)
		}
	}

	// Get JWT refresh expiration (optional, default to 7 days)
	refreshExpStr := os.Getenv(envJWTRefreshExpSecs)
	var refreshExp int64 = accessExp * 7 // Default refresh token expiration is 7x the access token
	if refreshExpStr != "" {
		var err error
		refreshExp, err = strconv.ParseInt(refreshExpStr, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("invalid JWT_REFRESH_EXPIRATION_SECONDS value: %w", err)
		}
	}

	// Get JWT issuer (optional)
	issuer := os.Getenv(envJWTIssuer)
	if issuer == "" {
		issuer = defaultIssuer
	}

	return Config{
		Secret:            secret,
		AccessExpiration:  accessExp,
		RefreshExpiration: refreshExp,
		Issuer:            issuer,
	}, nil
}
