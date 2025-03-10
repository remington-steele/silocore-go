package context

import (
	"context"
	"testing"
)

func TestAuthContext(t *testing.T) {
	// No need to create an unused context

	t.Run("UserID", func(t *testing.T) {
		// Test with valid user ID
		userID := int64(123)
		ctx := WithUserID(context.Background(), userID)

		retrievedID, err := GetUserID(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if retrievedID != userID {
			t.Errorf("Expected user ID %d, got %d", userID, retrievedID)
		}

		// Test with missing user ID
		_, err = GetUserID(context.Background())
		if err != ErrNoUserID {
			t.Errorf("Expected error %v, got %v", ErrNoUserID, err)
		}
	})

	t.Run("TenantID", func(t *testing.T) {
		// Test with valid tenant ID
		tenantIDValue := int64(456)
		tenantID := &tenantIDValue
		ctx := WithTenantID(context.Background(), tenantID)

		retrievedID, err := GetTenantID(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if retrievedID == nil || *retrievedID != *tenantID {
			t.Errorf("Expected tenant ID %d, got %v", *tenantID, retrievedID)
		}

		// Test with nil tenant ID
		ctx = WithTenantID(context.Background(), nil)
		retrievedID, err = GetTenantID(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if retrievedID != nil {
			t.Errorf("Expected nil tenant ID, got %v", retrievedID)
		}

		// Test with missing tenant ID
		_, err = GetTenantID(context.Background())
		if err != ErrNoTenantID {
			t.Errorf("Expected error %v, got %v", ErrNoTenantID, err)
		}
	})

	t.Run("Username", func(t *testing.T) {
		// Test with valid username
		username := "testuser"
		ctx := WithUsername(context.Background(), username)

		retrievedUsername, err := GetUsername(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if retrievedUsername != username {
			t.Errorf("Expected username %s, got %s", username, retrievedUsername)
		}

		// Test with missing username
		_, err = GetUsername(context.Background())
		if err != ErrNoUsername {
			t.Errorf("Expected error %v, got %v", ErrNoUsername, err)
		}
	})

	t.Run("Roles", func(t *testing.T) {
		// Test with valid roles
		roles := []Role{RoleAdmin, RoleTenantSuper}
		ctx := WithRoles(context.Background(), roles)

		retrievedRoles, err := GetRoles(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if len(retrievedRoles) != len(roles) {
			t.Errorf("Expected %d roles, got %d", len(roles), len(retrievedRoles))
		}
		for i, role := range roles {
			if retrievedRoles[i] != role {
				t.Errorf("Expected role %s at index %d, got %s", role, i, retrievedRoles[i])
			}
		}

		// Test with missing roles
		_, err = GetRoles(context.Background())
		if err != ErrNoRoles {
			t.Errorf("Expected error %v, got %v", ErrNoRoles, err)
		}
	})

	t.Run("HasRole", func(t *testing.T) {
		// Test with roles
		roles := []Role{RoleAdmin, RoleTenantSuper}
		ctx := WithRoles(context.Background(), roles)

		// Test with existing role
		if !HasRole(ctx, RoleAdmin) {
			t.Error("Expected HasRole to return true for ADMIN role")
		}

		// Test with non-existing role
		if HasRole(ctx, RoleInternal) {
			t.Error("Expected HasRole to return false for INTERNAL role")
		}

		// Test with missing roles
		if HasRole(context.Background(), RoleAdmin) {
			t.Error("Expected HasRole to return false for missing roles")
		}
	})

	t.Run("RoleHelpers", func(t *testing.T) {
		// Test with admin role
		ctx := WithRoles(context.Background(), []Role{RoleAdmin})
		if !IsAdmin(ctx) {
			t.Error("Expected IsAdmin to return true")
		}
		if IsTenantSuper(ctx) {
			t.Error("Expected IsTenantSuper to return false")
		}
		if IsInternal(ctx) {
			t.Error("Expected IsInternal to return false")
		}

		// Test with tenant super role
		ctx = WithRoles(context.Background(), []Role{RoleTenantSuper})
		if IsAdmin(ctx) {
			t.Error("Expected IsAdmin to return false")
		}
		if !IsTenantSuper(ctx) {
			t.Error("Expected IsTenantSuper to return true")
		}
		if IsInternal(ctx) {
			t.Error("Expected IsInternal to return false")
		}

		// Test with internal role
		ctx = WithRoles(context.Background(), []Role{RoleInternal})
		if IsAdmin(ctx) {
			t.Error("Expected IsAdmin to return false")
		}
		if IsTenantSuper(ctx) {
			t.Error("Expected IsTenantSuper to return false")
		}
		if !IsInternal(ctx) {
			t.Error("Expected IsInternal to return true")
		}

		// Test with multiple roles
		ctx = WithRoles(context.Background(), []Role{RoleAdmin, RoleTenantSuper})
		if !IsAdmin(ctx) {
			t.Error("Expected IsAdmin to return true")
		}
		if !IsTenantSuper(ctx) {
			t.Error("Expected IsTenantSuper to return true")
		}
		if IsInternal(ctx) {
			t.Error("Expected IsInternal to return false")
		}
	})
}
