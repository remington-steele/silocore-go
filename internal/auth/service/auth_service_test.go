package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	authctx "github.com/unsavory/silocore-go/internal/auth/context"
	"github.com/unsavory/silocore-go/internal/auth/jwt"
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

func (m *MockUserService) IsTenantMember(ctx context.Context, userID int64, tenantID int64) (bool, error) {
	args := m.Called(ctx, userID, tenantID)
	return args.Bool(0), args.Error(1)
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

func TestSwitchTenantContext(t *testing.T) {
	// Setup
	mockUserService := new(MockUserService)
	mockJWTService := new(MockJWTService)
	authService := NewDefaultAuthService(mockUserService, mockJWTService)
	ctx := context.Background()
	userID := int64(1)
	currentToken := "current-token"

	t.Run("Switch to global context with admin role", func(t *testing.T) {
		// Setup expectations
		mockUserService.On("GetUserRoles", ctx, userID).Return([]authctx.Role{authctx.RoleAdmin}, nil)
		mockJWTService.On("SwitchTenantContext", currentToken, (*int64)(nil)).Return("new-token", nil)

		// Execute
		token, err := authService.SwitchTenantContext(ctx, userID, currentToken, nil)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "new-token", token)
		mockUserService.AssertExpectations(t)
		mockJWTService.AssertExpectations(t)
	})

	t.Run("Switch to global context without admin role", func(t *testing.T) {
		// Setup expectations
		mockUserService.On("GetUserRoles", ctx, userID).Return([]authctx.Role{authctx.RoleTenantSuper}, nil)

		// Execute
		_, err := authService.SwitchTenantContext(ctx, userID, currentToken, nil)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
		mockUserService.AssertExpectations(t)
	})

	t.Run("Switch to tenant context as member", func(t *testing.T) {
		// Setup
		tenantID := int64(2)

		// Setup expectations
		mockUserService.On("IsTenantMember", ctx, userID, tenantID).Return(true, nil)
		mockJWTService.On("SwitchTenantContext", currentToken, &tenantID).Return("new-tenant-token", nil)

		// Execute
		token, err := authService.SwitchTenantContext(ctx, userID, currentToken, &tenantID)

		// Assert
		assert.NoError(t, err)
		assert.Equal(t, "new-tenant-token", token)
		mockUserService.AssertExpectations(t)
		mockJWTService.AssertExpectations(t)
	})

	t.Run("Switch to tenant context as non-member", func(t *testing.T) {
		// Setup
		tenantID := int64(3)

		// Setup expectations
		mockUserService.On("IsTenantMember", ctx, userID, tenantID).Return(false, nil)

		// Execute
		_, err := authService.SwitchTenantContext(ctx, userID, currentToken, &tenantID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
		mockUserService.AssertExpectations(t)
	})

	t.Run("Error checking tenant membership", func(t *testing.T) {
		// Setup
		tenantID := int64(4)
		expectedErr := errors.New("database error")

		// Setup expectations
		mockUserService.On("IsTenantMember", ctx, userID, tenantID).Return(false, expectedErr)

		// Execute
		_, err := authService.SwitchTenantContext(ctx, userID, currentToken, &tenantID)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), expectedErr.Error())
		mockUserService.AssertExpectations(t)
	})
}

func TestValidateAccess(t *testing.T) {
	// Setup
	mockUserService := new(MockUserService)
	mockJWTService := new(MockJWTService)
	authService := NewDefaultAuthService(mockUserService, mockJWTService)
	ctx := context.Background()
	userID := int64(1)

	t.Run("Admin has access to everything", func(t *testing.T) {
		// Setup expectations
		mockUserService.On("GetUserRoles", ctx, userID).Return([]authctx.Role{authctx.RoleAdmin}, nil)

		// Execute
		tenantID := int64(2)
		err := authService.ValidateAccess(ctx, userID, &tenantID, []authctx.Role{authctx.RoleTenantSuper})

		// Assert
		assert.NoError(t, err)
		mockUserService.AssertExpectations(t)
	})

	t.Run("Non-admin accessing tenant resource with required role", func(t *testing.T) {
		// Setup
		tenantID := int64(2)
		requiredRoles := []authctx.Role{authctx.RoleTenantSuper}

		// Setup expectations
		mockUserService.On("GetUserRoles", ctx, userID).Return([]authctx.Role{authctx.RoleInternal}, nil)
		mockUserService.On("IsTenantMember", ctx, userID, tenantID).Return(true, nil)
		mockUserService.On("GetUserTenantRoles", ctx, userID, tenantID).Return([]authctx.Role{authctx.RoleTenantSuper}, nil)

		// Execute
		err := authService.ValidateAccess(ctx, userID, &tenantID, requiredRoles)

		// Assert
		assert.NoError(t, err)
		mockUserService.AssertExpectations(t)
	})

	t.Run("Non-admin accessing tenant resource without required role", func(t *testing.T) {
		// Setup
		tenantID := int64(2)
		requiredRoles := []authctx.Role{authctx.RoleTenantSuper}

		// Setup expectations
		mockUserService.On("GetUserRoles", ctx, userID).Return([]authctx.Role{authctx.RoleInternal}, nil)
		mockUserService.On("IsTenantMember", ctx, userID, tenantID).Return(true, nil)
		mockUserService.On("GetUserTenantRoles", ctx, userID, tenantID).Return([]authctx.Role{authctx.RoleInternal}, nil)

		// Execute
		err := authService.ValidateAccess(ctx, userID, &tenantID, requiredRoles)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
		mockUserService.AssertExpectations(t)
	})

	t.Run("Non-member accessing tenant resource", func(t *testing.T) {
		// Setup
		tenantID := int64(2)

		// Setup expectations
		mockUserService.On("GetUserRoles", ctx, userID).Return([]authctx.Role{authctx.RoleInternal}, nil)
		mockUserService.On("IsTenantMember", ctx, userID, tenantID).Return(false, nil)

		// Execute
		err := authService.ValidateAccess(ctx, userID, &tenantID, []authctx.Role{})

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
		mockUserService.AssertExpectations(t)
	})
}

func TestBuildAuthContext(t *testing.T) {
	// Setup
	mockUserService := new(MockUserService)
	mockJWTService := new(MockJWTService)
	authService := NewDefaultAuthService(mockUserService, mockJWTService)
	ctx := context.Background()
	userID := int64(1)

	t.Run("Build context with tenant ID", func(t *testing.T) {
		// Setup
		tenantID := int64(2)
		systemRoles := []authctx.Role{authctx.RoleInternal}
		tenantRoles := []authctx.Role{authctx.RoleTenantSuper}

		// Setup expectations
		mockUserService.On("GetUserRoles", mock.Anything, userID).Return(systemRoles, nil)
		mockUserService.On("GetUserTenantRoles", mock.Anything, userID, tenantID).Return(tenantRoles, nil)

		// Execute
		newCtx, err := authService.BuildAuthContext(ctx, userID, &tenantID)

		// Assert
		assert.NoError(t, err)

		// Verify context values
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

	t.Run("Build context without tenant ID", func(t *testing.T) {
		// Setup
		systemRoles := []authctx.Role{authctx.RoleInternal}

		// Setup expectations
		mockUserService.On("GetUserRoles", mock.Anything, userID).Return(systemRoles, nil)

		// Execute
		newCtx, err := authService.BuildAuthContext(ctx, userID, nil)

		// Assert
		assert.NoError(t, err)

		// Verify context values
		ctxUserID, err := authctx.GetUserID(newCtx)
		assert.NoError(t, err)
		assert.Equal(t, userID, ctxUserID)

		_, err = authctx.GetTenantID(newCtx)
		assert.Error(t, err)

		ctxRoles, err := authctx.GetRoles(newCtx)
		assert.NoError(t, err)
		assert.Contains(t, ctxRoles, authctx.RoleInternal)

		mockUserService.AssertExpectations(t)
	})
}
