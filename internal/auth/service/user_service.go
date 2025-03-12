package service

import (
	"context"
	"database/sql"
	"errors"
	"log"

	authctx "github.com/unsavory/silocore-go/internal/auth/context"
)

// Common errors
var (
	ErrUserNotFound = errors.New("user not found")
	ErrDBOperation  = errors.New("database operation failed")
)

// User represents a user in the system
type User struct {
	ID           int64
	Email        string
	FirstName    string
	LastName     string
	PasswordHash string
}

// UserService defines the interface for user-related operations
type UserService interface {
	// GetUserRoles retrieves all roles for a user, both system-wide and tenant-specific
	GetUserRoles(ctx context.Context, userID int64) ([]authctx.Role, error)

	// GetUserTenantRoles retrieves tenant-specific roles for a user
	GetUserTenantRoles(ctx context.Context, userID int64, tenantID int64) ([]authctx.Role, error)

	// GetUserByEmail retrieves a user by their email address
	GetUserByEmail(ctx context.Context, email string) (*User, error)
}

// DBUserService implements UserService using a database
type DBUserService struct {
	db *sql.DB
}

// NewDBUserService creates a new DBUserService
func NewDBUserService(db *sql.DB) *DBUserService {
	return &DBUserService{db: db}
}

// GetUserByEmail retrieves a user by their email address
func (s *DBUserService) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT user_id, email, first_name, last_name, password_hash
		FROM usr
		WHERE email = $1
	`

	var user User
	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.FirstName,
		&user.LastName,
		&user.PasswordHash,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		log.Printf("[ERROR] Database error when getting user by email %s: %v", email, err)
		return nil, ErrDBOperation
	}

	return &user, nil
}

// GetUserRoles retrieves all system-wide roles for a user
func (s *DBUserService) GetUserRoles(ctx context.Context, userID int64) ([]authctx.Role, error) {
	// Query to get system-wide roles from user_role table
	query := `
		SELECT r.name 
		FROM user_role ur
		JOIN role r ON ur.role_id = r.id
		WHERE ur.user_id = $1
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, ErrDBOperation
	}
	defer rows.Close()

	var roles []authctx.Role
	for rows.Next() {
		var roleName string
		if err := rows.Scan(&roleName); err != nil {
			return nil, ErrDBOperation
		}
		roles = append(roles, authctx.Role(roleName))
	}

	if err := rows.Err(); err != nil {
		return nil, ErrDBOperation
	}

	if len(roles) == 0 {
		log.Printf("[INFO] No roles found for user ID %d", userID)
	}

	return roles, nil
}

// GetUserTenantRoles retrieves tenant-specific roles for a user
func (s *DBUserService) GetUserTenantRoles(ctx context.Context, userID int64, tenantID int64) ([]authctx.Role, error) {
	// Query to get tenant-specific roles from tenant_role table
	query := `
		SELECT r.name 
		FROM tenant_role tr
		JOIN role r ON tr.role_id = r.id
		WHERE tr.user_id = $1 AND tr.tenant_id = $2
	`

	rows, err := s.db.QueryContext(ctx, query, userID, tenantID)
	if err != nil {
		return nil, ErrDBOperation
	}
	defer rows.Close()

	var roles []authctx.Role
	for rows.Next() {
		var roleName string
		if err := rows.Scan(&roleName); err != nil {
			return nil, ErrDBOperation
		}
		roles = append(roles, authctx.Role(roleName))
	}

	if err := rows.Err(); err != nil {
		return nil, ErrDBOperation
	}

	if len(roles) == 0 {
		log.Printf("[INFO] No tenant roles found for user ID %d in tenant ID %d", userID, tenantID)
	}

	return roles, nil
}
