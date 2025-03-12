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

func TestGetUserByEmail(t *testing.T) {
	// Create a new mock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock database: %v", err)
	}
	defer db.Close()

	// Create a new user service with the mock database
	userService := NewDBUserService(db)

	// Set up test data
	email := "test@example.com"
	expectedUser := &User{
		ID:           1,
		Email:        email,
		FirstName:    "Test",
		LastName:     "User",
		PasswordHash: "hash",
	}

	// Set up mock expectations
	rows := sqlmock.NewRows([]string{"user_id", "email", "first_name", "last_name", "password_hash"}).
		AddRow(expectedUser.ID, expectedUser.Email, expectedUser.FirstName, expectedUser.LastName, expectedUser.PasswordHash)

	mock.ExpectQuery("SELECT user_id, email, first_name, last_name, password_hash FROM usr").
		WithArgs(email).
		WillReturnRows(rows)

	// Call the method being tested
	user, err := userService.GetUserByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("GetUserByEmail returned an error: %v", err)
	}

	// Check the results
	if user.ID != expectedUser.ID {
		t.Errorf("Expected user ID %d, got %d", expectedUser.ID, user.ID)
	}
	if user.Email != expectedUser.Email {
		t.Errorf("Expected user email %s, got %s", expectedUser.Email, user.Email)
	}
	if user.FirstName != expectedUser.FirstName {
		t.Errorf("Expected user first name %s, got %s", expectedUser.FirstName, user.FirstName)
	}
	if user.LastName != expectedUser.LastName {
		t.Errorf("Expected user last name %s, got %s", expectedUser.LastName, user.LastName)
	}
	if user.PasswordHash != expectedUser.PasswordHash {
		t.Errorf("Expected user password hash %s, got %s", expectedUser.PasswordHash, user.PasswordHash)
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
	email := "test@example.com"

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

	// Test GetUserByEmail with database error
	mock.ExpectQuery("SELECT user_id, email, first_name, last_name, password_hash FROM usr").
		WithArgs(email).
		WillReturnError(sql.ErrConnDone)

	_, err = userService.GetUserByEmail(context.Background(), email)
	if err != ErrDBOperation {
		t.Errorf("Expected ErrDBOperation, got %v", err)
	}

	// Ensure all expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %s", err)
	}
}
