package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	authctx "github.com/unsavory/silocore-go/internal/auth/context"
)

func TestGetUserRoles(t *testing.T) {
	// Create a new mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Create a new user service with the mock database
	userService := NewDBUserService(db)

	// Set up test data
	userID := int64(1)
	expectedRoles := []authctx.Role{authctx.RoleAdmin, authctx.RoleInternal}

	// Set up mock expectations
	rows := sqlmock.NewRows([]string{"name"}).
		AddRow(string(authctx.RoleAdmin)).
		AddRow(string(authctx.RoleInternal))

	mock.ExpectQuery("SELECT r.name FROM user_role").
		WithArgs(userID).
		WillReturnRows(rows)

	// Call the method being tested
	roles, err := userService.GetUserRoles(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetUserRoles returned an error: %v", err)
	}

	// Check the results
	if len(roles) != len(expectedRoles) {
		t.Errorf("Expected %d roles, got %d", len(expectedRoles), len(roles))
	}

	for i, role := range roles {
		if role != expectedRoles[i] {
			t.Errorf("Expected role %s, got %s", expectedRoles[i], role)
		}
	}

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestGetUserTenantRoles(t *testing.T) {
	// Create a new mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Create a new user service with the mock database
	userService := NewDBUserService(db)

	// Set up test data
	userID := int64(1)
	tenantID := int64(2)
	expectedRoles := []authctx.Role{authctx.RoleTenantSuper}

	// Set up mock expectations
	rows := sqlmock.NewRows([]string{"name"}).
		AddRow(string(authctx.RoleTenantSuper))

	mock.ExpectQuery("SELECT r.name FROM tenant_role").
		WithArgs(userID, tenantID).
		WillReturnRows(rows)

	// Call the method being tested
	roles, err := userService.GetUserTenantRoles(context.Background(), userID, tenantID)
	if err != nil {
		t.Fatalf("GetUserTenantRoles returned an error: %v", err)
	}

	// Check the results
	if len(roles) != len(expectedRoles) {
		t.Errorf("Expected %d roles, got %d", len(expectedRoles), len(roles))
	}

	for i, role := range roles {
		if role != expectedRoles[i] {
			t.Errorf("Expected role %s, got %s", expectedRoles[i], role)
		}
	}

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestIsTenantMember(t *testing.T) {
	// Create a new mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Create a new user service with the mock database
	userService := NewDBUserService(db)

	// Set up test data
	userID := int64(1)
	tenantID := int64(2)
	expectedIsMember := true

	// Set up mock expectations
	rows := sqlmock.NewRows([]string{"exists"}).
		AddRow(expectedIsMember)

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(userID, tenantID).
		WillReturnRows(rows)

	// Call the method being tested
	isMember, err := userService.IsTenantMember(context.Background(), userID, tenantID)
	if err != nil {
		t.Fatalf("IsTenantMember returned an error: %v", err)
	}

	// Check the results
	if isMember != expectedIsMember {
		t.Errorf("Expected isMember to be %v, got %v", expectedIsMember, isMember)
	}

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}

func TestDBErrors(t *testing.T) {
	// Create a new mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Create a new user service with the mock database
	userService := NewDBUserService(db)

	// Set up test data
	userID := int64(1)
	tenantID := int64(2)

	// Test GetUserRoles with database error
	mock.ExpectQuery("SELECT r.name FROM user_role").
		WithArgs(userID).
		WillReturnError(sql.ErrConnDone)

	_, err = userService.GetUserRoles(context.Background(), userID)
	if err != ErrDBOperation {
		t.Errorf("Expected ErrDBOperation, got %v", err)
	}

	// Test GetUserTenantRoles with database error
	mock.ExpectQuery("SELECT r.name FROM tenant_role").
		WithArgs(userID, tenantID).
		WillReturnError(sql.ErrConnDone)

	_, err = userService.GetUserTenantRoles(context.Background(), userID, tenantID)
	if err != ErrDBOperation {
		t.Errorf("Expected ErrDBOperation, got %v", err)
	}

	// Test IsTenantMember with database error
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(userID, tenantID).
		WillReturnError(sql.ErrConnDone)

	_, err = userService.IsTenantMember(context.Background(), userID, tenantID)
	if err != ErrDBOperation {
		t.Errorf("Expected ErrDBOperation, got %v", err)
	}

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}
