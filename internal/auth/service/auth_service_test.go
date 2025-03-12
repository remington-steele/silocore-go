package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	authctx "github.com/unsavory/silocore-go/internal/auth/context"
	"github.com/unsavory/silocore-go/internal/auth/jwt"
	tenantservice "github.com/unsavory/silocore-go/internal/tenant/service"
)

// MockUserService is a mock implementation of UserService
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetUserRoles(ctx context.Context, userID int64) ([]authctx.Role, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]authctx.Role), args.Error(1)
}

func (m *MockUserService) GetUserTenantRoles(ctx context.Context, userID int64, tenantID int64) ([]authctx.Role, error) {
	args := m.Called(ctx, userID, tenantID)
	return args.Get(0).([]authctx.Role), args.Error(1)
}

func (m *MockUserService) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

// MockTenantMemberService is a mock implementation of TenantMemberService
type MockTenantMemberService struct {
	mock.Mock
}

func (m *MockTenantMemberService) GetUserTenantMemberships(ctx context.Context, userID int64) ([]tenantservice.TenantMembership, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]tenantservice.TenantMembership), args.Error(1)
}

func (m *MockTenantMemberService) GetUserDefaultTenant(ctx context.Context, userID int64) (*int64, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*int64), args.Error(1)
}

func (m *MockTenantMemberService) IsTenantMember(ctx context.Context, userID int64, tenantID int64) (bool, error) {
	args := m.Called(ctx, userID, tenantID)
	return args.Bool(0), args.Error(1)
}

func (m *MockTenantMemberService) AddTenantMember(ctx context.Context, userID int64, tenantID int64) error {
	args := m.Called(ctx, userID, tenantID)
	return args.Error(0)
}

func (m *MockTenantMemberService) RemoveTenantMember(ctx context.Context, userID int64, tenantID int64) error {
	args := m.Called(ctx, userID, tenantID)
	return args.Error(0)
}

// MockJWTService is a mock implementation of jwt.JWTService
type MockJWTService struct {
	mock.Mock
}

func (m *MockJWTService) SwitchTenantContext(currentToken string, newTenantID *int64) (string, error) {
	args := m.Called(currentToken, newTenantID)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) GenerateTokenPair(userID int64, username string, tenantID *int64) (*jwt.TokenPair, error) {
	args := m.Called(userID, username, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwt.TokenPair), args.Error(1)
}

func (m *MockJWTService) ValidateToken(tokenString string) (*jwt.CustomClaims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwt.CustomClaims), args.Error(1)
}

func (m *MockJWTService) RefreshToken(refreshToken string, tenantID *int64) (*jwt.TokenPair, error) {
	args := m.Called(refreshToken, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*jwt.TokenPair), args.Error(1)
}

// MockPasswordVerifier is a mock implementation for password verification
type MockPasswordVerifier struct {
	ShouldSucceed bool
	Error         error
}

func (m *MockPasswordVerifier) VerifyPassword(storedHash, password string) (bool, error) {
	return m.ShouldSucceed, m.Error
}

func TestLogin(t *testing.T) {
	// Setup
	mockUserService := new(MockUserService)
	mockTenantMemberService := new(MockTenantMemberService)
	mockJWTService := new(MockJWTService)

	ctx := context.Background()

	t.Run("Successful login", func(t *testing.T) {
		// Setup test data
		email := "test@example.com"
		password := "password123"
		userID := int64(1)
		passwordHash := "salt:hash" // This would be a real scrypt hash in production

		// Create a mock user
		user := &User{
			ID:           userID,
			Email:        email,
			FirstName:    "Test",
			LastName:     "User",
			PasswordHash: passwordHash,
		}

		// Setup tenant ID
		tenantID := int64(2)

		// Setup token pair
		tokenPair := &jwt.TokenPair{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
		}

		// Setup expectations
		mockUserService.On("GetUserByEmail", ctx, email).Return(user, nil).Once()
		mockTenantMemberService.On("GetUserDefaultTenant", ctx, userID).Return(&tenantID, nil).Once()
		mockJWTService.On("GenerateTokenPair", userID, email, &tenantID).Return(tokenPair, nil).Once()

		// Create a custom auth service with mocked password verification
		customAuthService := &DefaultAuthService{
			userService:         mockUserService,
			tenantMemberService: mockTenantMemberService,
			jwtService:          mockJWTService,
		}

		// Override the VerifyPassword function for this test
		verifyPasswordFunc := func(storedHash, pwd string) (bool, error) {
			return true, nil
		}

		// Execute with custom verification
		resultTokenPair, resultUserID, err := customAuthService.loginWithVerifier(ctx, email, password, verifyPasswordFunc)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, tokenPair, resultTokenPair)
		assert.Equal(t, userID, resultUserID)
		mockUserService.AssertExpectations(t)
		mockTenantMemberService.AssertExpectations(t)
		mockJWTService.AssertExpectations(t)
	})

	t.Run("User not found", func(t *testing.T) {
		// Setup test data
		email := "nonexistent@example.com"
		password := "password123"

		// Setup expectations
		mockUserService.On("GetUserByEmail", ctx, email).Return(nil, ErrUserNotFound).Once()

		// Create a custom auth service with mocked password verification
		customAuthService := &DefaultAuthService{
			userService:         mockUserService,
			tenantMemberService: mockTenantMemberService,
			jwtService:          mockJWTService,
		}

		// Override the VerifyPassword function for this test
		verifyPasswordFunc := func(storedHash, pwd string) (bool, error) {
			return true, nil
		}

		// Execute with custom verification
		resultTokenPair, resultUserID, err := customAuthService.loginWithVerifier(ctx, email, password, verifyPasswordFunc)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCredentials, err)
		assert.Nil(t, resultTokenPair)
		assert.Equal(t, int64(0), resultUserID)
		mockUserService.AssertExpectations(t)
	})

	t.Run("Invalid password", func(t *testing.T) {
		// Setup test data
		email := "test@example.com"
		password := "wrongpassword"
		userID := int64(1)
		passwordHash := "salt:hash" // This would be a real scrypt hash in production

		// Create a mock user
		user := &User{
			ID:           userID,
			Email:        email,
			FirstName:    "Test",
			LastName:     "User",
			PasswordHash: passwordHash,
		}

		// Setup expectations
		mockUserService.On("GetUserByEmail", ctx, email).Return(user, nil).Once()

		// Create a custom auth service with mocked password verification
		customAuthService := &DefaultAuthService{
			userService:         mockUserService,
			tenantMemberService: mockTenantMemberService,
			jwtService:          mockJWTService,
		}

		// Override the VerifyPassword function for this test
		verifyPasswordFunc := func(storedHash, pwd string) (bool, error) {
			return false, nil
		}

		// Execute with custom verification
		resultTokenPair, resultUserID, err := customAuthService.loginWithVerifier(ctx, email, password, verifyPasswordFunc)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCredentials, err)
		assert.Nil(t, resultTokenPair)
		assert.Equal(t, int64(0), resultUserID)
		mockUserService.AssertExpectations(t)
	})

	t.Run("No tenant memberships", func(t *testing.T) {
		// Setup test data
		email := "test@example.com"
		password := "password123"
		userID := int64(1)
		passwordHash := "salt:hash" // This would be a real scrypt hash in production

		// Create a mock user
		user := &User{
			ID:           userID,
			Email:        email,
			FirstName:    "Test",
			LastName:     "User",
			PasswordHash: passwordHash,
		}

		// Setup token pair
		tokenPair := &jwt.TokenPair{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
		}

		// Setup expectations
		mockUserService.On("GetUserByEmail", ctx, email).Return(user, nil).Once()
		mockTenantMemberService.On("GetUserDefaultTenant", ctx, userID).Return(nil, nil).Once()
		mockJWTService.On("GenerateTokenPair", userID, email, mock.Anything).Return(tokenPair, nil).Once()

		// Create a custom auth service with mocked password verification
		customAuthService := &DefaultAuthService{
			userService:         mockUserService,
			tenantMemberService: mockTenantMemberService,
			jwtService:          mockJWTService,
		}

		// Override the VerifyPassword function for this test
		verifyPasswordFunc := func(storedHash, pwd string) (bool, error) {
			return true, nil
		}

		// Execute with custom verification
		resultTokenPair, resultUserID, err := customAuthService.loginWithVerifier(ctx, email, password, verifyPasswordFunc)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, tokenPair, resultTokenPair)
		assert.Equal(t, userID, resultUserID)
		mockUserService.AssertExpectations(t)
		mockTenantMemberService.AssertExpectations(t)
		mockJWTService.AssertExpectations(t)
	})

	t.Run("Database error during user lookup", func(t *testing.T) {
		// Setup test data
		email := "test@example.com"
		password := "password123"
		dbError := errors.New("database connection error")

		// Setup expectations
		mockUserService.On("GetUserByEmail", ctx, email).Return(nil, dbError).Once()

		// Create a custom auth service with mocked password verification
		customAuthService := &DefaultAuthService{
			userService:         mockUserService,
			tenantMemberService: mockTenantMemberService,
			jwtService:          mockJWTService,
		}

		// Override the VerifyPassword function for this test
		verifyPasswordFunc := func(storedHash, pwd string) (bool, error) {
			return true, nil
		}

		// Execute with custom verification
		resultTokenPair, resultUserID, err := customAuthService.loginWithVerifier(ctx, email, password, verifyPasswordFunc)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, dbError, err)
		assert.Nil(t, resultTokenPair)
		assert.Equal(t, int64(0), resultUserID)
		mockUserService.AssertExpectations(t)
	})

	t.Run("Error generating token", func(t *testing.T) {
		// Setup test data
		email := "test@example.com"
		password := "password123"
		userID := int64(1)
		passwordHash := "salt:hash" // This would be a real scrypt hash in production
		tokenError := errors.New("token generation error")

		// Create a mock user
		user := &User{
			ID:           userID,
			Email:        email,
			FirstName:    "Test",
			LastName:     "User",
			PasswordHash: passwordHash,
		}

		// Setup tenant ID
		tenantID := int64(2)

		// Setup expectations
		mockUserService.On("GetUserByEmail", ctx, email).Return(user, nil).Once()
		mockTenantMemberService.On("GetUserDefaultTenant", ctx, userID).Return(&tenantID, nil).Once()
		mockJWTService.On("GenerateTokenPair", userID, email, &tenantID).Return(nil, tokenError).Once()

		// Create a custom auth service with mocked password verification
		customAuthService := &DefaultAuthService{
			userService:         mockUserService,
			tenantMemberService: mockTenantMemberService,
			jwtService:          mockJWTService,
		}

		// Override the VerifyPassword function for this test
		verifyPasswordFunc := func(storedHash, pwd string) (bool, error) {
			return true, nil
		}

		// Execute with custom verification
		resultTokenPair, resultUserID, err := customAuthService.loginWithVerifier(ctx, email, password, verifyPasswordFunc)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, tokenError, err)
		assert.Nil(t, resultTokenPair)
		assert.Equal(t, int64(0), resultUserID)
		mockUserService.AssertExpectations(t)
		mockTenantMemberService.AssertExpectations(t)
		mockJWTService.AssertExpectations(t)
	})
}

func TestSwitchTenantContext(t *testing.T) {
	// Setup
	mockUserService := new(MockUserService)
	mockTenantMemberService := new(MockTenantMemberService)
	mockJWTService := new(MockJWTService)
	authService := NewDefaultAuthService(mockUserService, mockTenantMemberService, mockJWTService)

	ctx := context.Background()
	userID := int64(1)
	currentToken := "current-token"
	newToken := "new-token"

	t.Run("Switch to global context with admin role", func(t *testing.T) {
		// Setup expectations
		mockUserService.On("GetUserRoles", mock.Anything, userID).Return([]authctx.Role{authctx.RoleAdmin}, nil).Once()
		mockJWTService.On("SwitchTenantContext", currentToken, mock.Anything).Return(newToken, nil).Once()

		// Execute
		token, err := authService.SwitchTenantContext(ctx, userID, currentToken, nil)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, newToken, token)
		mockUserService.AssertExpectations(t)
		mockJWTService.AssertExpectations(t)
	})

	t.Run("Switch to global context without admin role", func(t *testing.T) {
		// Setup expectations
		mockUserService.On("GetUserRoles", mock.Anything, userID).Return([]authctx.Role{authctx.RoleTenantSuper}, nil).Once()

		// Execute
		token, err := authService.SwitchTenantContext(ctx, userID, currentToken, nil)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
		assert.Empty(t, token)
		mockUserService.AssertExpectations(t)
	})

	t.Run("Switch to tenant context as member", func(t *testing.T) {
		// Setup
		tenantID := int64(2)

		// Setup expectations
		mockTenantMemberService.On("IsTenantMember", mock.Anything, userID, tenantID).Return(true, nil).Once()
		mockJWTService.On("SwitchTenantContext", currentToken, &tenantID).Return(newToken, nil).Once()

		// Execute
		token, err := authService.SwitchTenantContext(ctx, userID, currentToken, &tenantID)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, newToken, token)
		mockTenantMemberService.AssertExpectations(t)
		mockJWTService.AssertExpectations(t)
	})

	t.Run("Switch to tenant context as non-member", func(t *testing.T) {
		// Setup
		tenantID := int64(3)

		// Setup expectations
		mockTenantMemberService.On("IsTenantMember", mock.Anything, userID, tenantID).Return(false, nil).Once()

		// Execute
		token, err := authService.SwitchTenantContext(ctx, userID, currentToken, &tenantID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
		assert.Empty(t, token)
		mockTenantMemberService.AssertExpectations(t)
	})
}

func TestValidateAccess(t *testing.T) {
	// Setup
	mockUserService := new(MockUserService)
	mockTenantMemberService := new(MockTenantMemberService)
	mockJWTService := new(MockJWTService)
	authService := NewDefaultAuthService(mockUserService, mockTenantMemberService, mockJWTService)

	ctx := context.Background()
	userID := int64(1)

	t.Run("Admin has access to everything", func(t *testing.T) {
		// Setup expectations
		mockUserService.On("GetUserRoles", mock.Anything, userID).Return([]authctx.Role{authctx.RoleAdmin}, nil).Once()

		// Execute
		err := authService.ValidateAccess(ctx, userID, nil, []authctx.Role{authctx.RoleTenantSuper})

		// Assert
		assert.NoError(t, err)
		mockUserService.AssertExpectations(t)
	})

	t.Run("Tenant member has access to tenant", func(t *testing.T) {
		// Setup
		tenantID := int64(2)

		// Setup expectations
		mockUserService.On("GetUserRoles", mock.Anything, userID).Return([]authctx.Role{authctx.RoleInternal}, nil).Once()
		mockTenantMemberService.On("IsTenantMember", mock.Anything, userID, tenantID).Return(true, nil).Once()

		// Execute
		err := authService.ValidateAccess(ctx, userID, &tenantID, nil)

		// Assert
		assert.NoError(t, err)
		mockUserService.AssertExpectations(t)
		mockTenantMemberService.AssertExpectations(t)
	})

	t.Run("Non-member has no access to tenant", func(t *testing.T) {
		// Setup
		tenantID := int64(3)

		// Setup expectations
		mockUserService.On("GetUserRoles", mock.Anything, userID).Return([]authctx.Role{authctx.RoleInternal}, nil).Once()
		mockTenantMemberService.On("IsTenantMember", mock.Anything, userID, tenantID).Return(false, nil).Once()

		// Execute
		err := authService.ValidateAccess(ctx, userID, &tenantID, nil)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
		mockUserService.AssertExpectations(t)
		mockTenantMemberService.AssertExpectations(t)
	})

	t.Run("Tenant member with required role has access", func(t *testing.T) {
		// Setup
		tenantID := int64(2)

		// Setup expectations
		mockUserService.On("GetUserRoles", mock.Anything, userID).Return([]authctx.Role{authctx.RoleInternal}, nil).Once()
		mockTenantMemberService.On("IsTenantMember", mock.Anything, userID, tenantID).Return(true, nil).Once()
		mockUserService.On("GetUserTenantRoles", mock.Anything, userID, tenantID).Return([]authctx.Role{authctx.RoleTenantSuper}, nil).Once()

		// Execute
		err := authService.ValidateAccess(ctx, userID, &tenantID, []authctx.Role{authctx.RoleTenantSuper})

		// Assert
		assert.NoError(t, err)
		mockUserService.AssertExpectations(t)
		mockTenantMemberService.AssertExpectations(t)
	})

	t.Run("Tenant member without required role has no access", func(t *testing.T) {
		// Setup
		tenantID := int64(2)

		// Setup expectations
		mockUserService.On("GetUserRoles", mock.Anything, userID).Return([]authctx.Role{authctx.RoleInternal}, nil).Once()
		mockTenantMemberService.On("IsTenantMember", mock.Anything, userID, tenantID).Return(true, nil).Once()
		mockUserService.On("GetUserTenantRoles", mock.Anything, userID, tenantID).Return([]authctx.Role{}, nil).Once()

		// Execute
		err := authService.ValidateAccess(ctx, userID, &tenantID, []authctx.Role{authctx.RoleTenantSuper})

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
		mockUserService.AssertExpectations(t)
		mockTenantMemberService.AssertExpectations(t)
	})
}

func TestBuildAuthContext(t *testing.T) {
	// Setup
	mockUserService := new(MockUserService)
	mockTenantMemberService := new(MockTenantMemberService)
	mockJWTService := new(MockJWTService)
	authService := NewDefaultAuthService(mockUserService, mockTenantMemberService, mockJWTService)

	ctx := context.Background()
	userID := int64(1)

	t.Run("Build context with system roles only", func(t *testing.T) {
		// Setup expectations
		mockUserService.On("GetUserRoles", mock.Anything, userID).Return([]authctx.Role{authctx.RoleAdmin}, nil).Once()

		// Execute
		newCtx, err := authService.BuildAuthContext(ctx, userID, nil)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, newCtx)

		// Check context values
		ctxUserID, err := authctx.GetUserID(newCtx)
		assert.NoError(t, err)
		assert.Equal(t, userID, ctxUserID)

		_, err = authctx.GetTenantID(newCtx)
		assert.Error(t, err) // Should error since no tenant ID was set

		ctxRoles, err := authctx.GetRoles(newCtx)
		assert.NoError(t, err)
		assert.Contains(t, ctxRoles, authctx.RoleAdmin)
		mockUserService.AssertExpectations(t)
	})

	t.Run("Build context with tenant roles", func(t *testing.T) {
		// Setup
		tenantID := int64(2)

		// Setup expectations
		mockUserService.On("GetUserRoles", mock.Anything, userID).Return([]authctx.Role{authctx.RoleInternal}, nil).Once()
		mockUserService.On("GetUserTenantRoles", mock.Anything, userID, tenantID).Return([]authctx.Role{authctx.RoleTenantSuper}, nil).Once()

		// Execute
		newCtx, err := authService.BuildAuthContext(ctx, userID, &tenantID)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, newCtx)

		// Check context values
		ctxUserID, err := authctx.GetUserID(newCtx)
		assert.NoError(t, err)
		assert.Equal(t, userID, ctxUserID)

		ctxTenantID, err := authctx.GetTenantID(newCtx)
		assert.NoError(t, err)
		assert.Equal(t, &tenantID, ctxTenantID)

		ctxRoles, err := authctx.GetRoles(newCtx)
		assert.NoError(t, err)
		assert.Contains(t, ctxRoles, authctx.RoleInternal)
		assert.Contains(t, ctxRoles, authctx.RoleTenantSuper)
		mockUserService.AssertExpectations(t)
	})
}
