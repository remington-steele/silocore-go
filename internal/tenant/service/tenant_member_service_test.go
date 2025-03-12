package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestGetUserDefaultTenant(t *testing.T) {
	// Create a new mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Create a new tenant member service with the mock database
	tenantMemberService := NewDBTenantMemberService(db)

	// Set up test data
	userID := int64(1)
	expectedTenantID := int64(2)

	t.Run("User has tenant memberships", func(t *testing.T) {
		// Set up mock expectations
		rows := sqlmock.NewRows([]string{"tenant_id"}).
			AddRow(expectedTenantID)

		mock.ExpectQuery("SELECT tenant_id FROM tenant_member").
			WithArgs(userID).
			WillReturnRows(rows)

		// Call the method being tested
		tenantID, err := tenantMemberService.GetUserDefaultTenant(context.Background(), userID)
		assert.NoError(t, err)
		assert.NotNil(t, tenantID)
		assert.Equal(t, expectedTenantID, *tenantID)

		// Ensure all expectations were met
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("User has no tenant memberships", func(t *testing.T) {
		// Set up mock expectations
		rows := sqlmock.NewRows([]string{"tenant_id"})

		mock.ExpectQuery("SELECT tenant_id FROM tenant_member").
			WithArgs(userID).
			WillReturnRows(rows)

		// Call the method being tested
		tenantID, err := tenantMemberService.GetUserDefaultTenant(context.Background(), userID)
		assert.NoError(t, err)
		assert.Nil(t, tenantID)

		// Ensure all expectations were met
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Database error", func(t *testing.T) {
		// Set up mock expectations
		mock.ExpectQuery("SELECT tenant_id FROM tenant_member").
			WithArgs(userID).
			WillReturnError(sql.ErrConnDone)

		// Call the method being tested
		tenantID, err := tenantMemberService.GetUserDefaultTenant(context.Background(), userID)
		assert.Error(t, err)
		assert.Nil(t, tenantID)

		// Ensure all expectations were met
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestIsTenantMember(t *testing.T) {
	// Create a new mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Create a new tenant member service with the mock database
	tenantMemberService := NewDBTenantMemberService(db)

	// Set up test data
	userID := int64(1)
	tenantID := int64(2)
	expectedIsMember := true

	t.Run("User is a tenant member", func(t *testing.T) {
		// Set up mock expectations
		rows := sqlmock.NewRows([]string{"exists"}).
			AddRow(expectedIsMember)

		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID, tenantID).
			WillReturnRows(rows)

		// Call the method being tested
		isMember, err := tenantMemberService.IsTenantMember(context.Background(), userID, tenantID)
		assert.NoError(t, err)
		assert.Equal(t, expectedIsMember, isMember)

		// Ensure all expectations were met
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("User is not a tenant member", func(t *testing.T) {
		// Set up mock expectations
		rows := sqlmock.NewRows([]string{"exists"}).
			AddRow(false)

		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID, tenantID).
			WillReturnRows(rows)

		// Call the method being tested
		isMember, err := tenantMemberService.IsTenantMember(context.Background(), userID, tenantID)
		assert.NoError(t, err)
		assert.False(t, isMember)

		// Ensure all expectations were met
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Database error", func(t *testing.T) {
		// Set up mock expectations
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(userID, tenantID).
			WillReturnError(sql.ErrConnDone)

		// Call the method being tested
		isMember, err := tenantMemberService.IsTenantMember(context.Background(), userID, tenantID)
		assert.Error(t, err)
		assert.False(t, isMember)

		// Ensure all expectations were met
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetUserTenantMemberships(t *testing.T) {
	// Create a new mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Create a new tenant member service with the mock database
	tenantMemberService := NewDBTenantMemberService(db)

	// Set up test data
	userID := int64(1)
	now := sql.NullTime{Time: sql.NullTime{}.Time, Valid: true}

	t.Run("User has tenant memberships", func(t *testing.T) {
		// Set up mock expectations
		rows := sqlmock.NewRows([]string{"tenant_id", "user_id", "created_at"}).
			AddRow(1, userID, now).
			AddRow(2, userID, now)

		mock.ExpectQuery("SELECT tenant_id, user_id, created_at FROM tenant_member").
			WithArgs(userID).
			WillReturnRows(rows)

		// Call the method being tested
		memberships, err := tenantMemberService.GetUserTenantMemberships(context.Background(), userID)
		assert.NoError(t, err)
		assert.Len(t, memberships, 2)
		assert.Equal(t, int64(1), memberships[0].TenantID)
		assert.Equal(t, int64(2), memberships[1].TenantID)

		// Ensure all expectations were met
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("User has no tenant memberships", func(t *testing.T) {
		// Set up mock expectations
		rows := sqlmock.NewRows([]string{"tenant_id", "user_id", "created_at"})

		mock.ExpectQuery("SELECT tenant_id, user_id, created_at FROM tenant_member").
			WithArgs(userID).
			WillReturnRows(rows)

		// Call the method being tested
		memberships, err := tenantMemberService.GetUserTenantMemberships(context.Background(), userID)
		assert.NoError(t, err)
		assert.Empty(t, memberships)

		// Ensure all expectations were met
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Database error", func(t *testing.T) {
		// Set up mock expectations
		mock.ExpectQuery("SELECT tenant_id, user_id, created_at FROM tenant_member").
			WithArgs(userID).
			WillReturnError(sql.ErrConnDone)

		// Call the method being tested
		memberships, err := tenantMemberService.GetUserTenantMemberships(context.Background(), userID)
		assert.Error(t, err)
		assert.Nil(t, memberships)

		// Ensure all expectations were met
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
