package jwt

import (
	"testing"
)

func TestJWTService(t *testing.T) {
	// Test configuration
	config := Config{
		Secret:            "test-secret-key-for-jwt-token-generation",
		AccessExpiration:  300,  // 5 minutes
		RefreshExpiration: 3600, // 1 hour
		Issuer:            "test-issuer",
	}

	// Create service
	service := NewService(config)

	// Test user data
	userID := int64(123)
	username := "testuser"
	var tenantID *int64
	tenantIDValue := int64(456)
	tenantID = &tenantIDValue

	t.Run("GenerateTokenPair", func(t *testing.T) {
		// Generate token pair
		tokenPair, err := service.GenerateTokenPair(userID, username, tenantID)
		if err != nil {
			t.Fatalf("Failed to generate token pair: %v", err)
		}

		// Verify token pair
		if tokenPair.AccessToken == "" {
			t.Error("Access token is empty")
		}
		if tokenPair.RefreshToken == "" {
			t.Error("Refresh token is empty")
		}
		if tokenPair.ExpiresIn <= 0 || tokenPair.ExpiresIn > 300 {
			t.Errorf("Expected expiration between 0 and 300 seconds, got %d", tokenPair.ExpiresIn)
		}
	})

	t.Run("ValidateToken", func(t *testing.T) {
		// Generate token
		token, _, err := service.generateToken(userID, username, tenantID, config.AccessExpiration)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		// Validate token
		claims, err := service.ValidateToken(token)
		if err != nil {
			t.Fatalf("Failed to validate token: %v", err)
		}

		// Verify claims
		if claims.UserID != userID {
			t.Errorf("Expected user ID %d, got %d", userID, claims.UserID)
		}
		if claims.Username != username {
			t.Errorf("Expected username %s, got %s", username, claims.Username)
		}
		if claims.TenantID == nil || *claims.TenantID != *tenantID {
			t.Errorf("Expected tenant ID %d, got %v", *tenantID, claims.TenantID)
		}
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		// Generate token with negative expiration
		token, _, err := service.generateToken(userID, username, tenantID, -10)
		if err != nil {
			t.Fatalf("Failed to generate expired token: %v", err)
		}

		// Validate token
		_, err = service.ValidateToken(token)
		if err == nil {
			t.Fatal("Expected error for expired token, got nil")
		}
		if err != ErrExpiredToken {
			t.Errorf("Expected error %v, got %v", ErrExpiredToken, err)
		}
	})

	t.Run("SwitchTenantContext", func(t *testing.T) {
		// Generate token with tenant context
		token, _, err := service.generateToken(userID, username, tenantID, config.AccessExpiration)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		// Switch tenant context
		newTenantID := int64(789)
		newToken, err := service.SwitchTenantContext(token, &newTenantID)
		if err != nil {
			t.Fatalf("Failed to switch tenant context: %v", err)
		}

		// Validate new token
		claims, err := service.ValidateToken(newToken)
		if err != nil {
			t.Fatalf("Failed to validate new token: %v", err)
		}

		// Verify claims
		if claims.UserID != userID {
			t.Errorf("Expected user ID %d, got %d", userID, claims.UserID)
		}
		if claims.TenantID == nil || *claims.TenantID != newTenantID {
			t.Errorf("Expected tenant ID %d, got %v", newTenantID, claims.TenantID)
		}
	})

	t.Run("RefreshToken", func(t *testing.T) {
		// Generate refresh token
		refreshToken, _, err := service.generateToken(userID, username, nil, config.RefreshExpiration)
		if err != nil {
			t.Fatalf("Failed to generate refresh token: %v", err)
		}

		// Refresh token with tenant context
		tokenPair, err := service.RefreshToken(refreshToken, tenantID)
		if err != nil {
			t.Fatalf("Failed to refresh token: %v", err)
		}

		// Validate new access token
		claims, err := service.ValidateToken(tokenPair.AccessToken)
		if err != nil {
			t.Fatalf("Failed to validate new access token: %v", err)
		}

		// Verify claims
		if claims.UserID != userID {
			t.Errorf("Expected user ID %d, got %d", userID, claims.UserID)
		}
		if claims.TenantID == nil || *claims.TenantID != *tenantID {
			t.Errorf("Expected tenant ID %d, got %v", *tenantID, claims.TenantID)
		}
	})
}
