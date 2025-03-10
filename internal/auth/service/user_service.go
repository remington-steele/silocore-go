package service

import (
	"context"
	"database/sql"
	"errors"

	authctx "github.com/unsavory/silocore-go/internal/auth/context"
)

// Common errors
var (
	ErrUserNotFound = errors.New("user not found")
	ErrDBOperation  = errors.New("database operation failed")
)

// UserService defines the interface for user-related operations
type UserService interface {
	// GetUserRoles retrieves all roles for a user, both system-wide and tenant-specific
	GetUserRoles(ctx context.Context, userID int64) ([]authctx.Role, error)

	// GetUserTenantRoles retrieves tenant-specific roles for a user
	GetUserTenantRoles(ctx context.Context, userID int64, tenantID int64) ([]authctx.Role, error)

	// IsTenantMember checks if a user is a member of a specific tenant
	IsTenantMember(ctx context.Context, userID int64, tenantID int64) (bool, error)
}

// DBUserService implements UserService using a database
type DBUserService struct {
	db *sql.DB
}

// NewDBUserService creates a new DBUserService
func NewDBUserService(db *sql.DB) *DBUserService {
	return &DBUserService{db: db}
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

	return roles, nil
}

// IsTenantMember checks if a user is a member of a specific tenant
func (s *DBUserService) IsTenantMember(ctx context.Context, userID int64, tenantID int64) (bool, error) {
	// Query to check if user is a member of the tenant
	query := `
		SELECT EXISTS(
			SELECT 1 FROM tenant_member 
			WHERE user_id = $1 AND tenant_id = $2
		)
	`

	var isMember bool
	err := s.db.QueryRowContext(ctx, query, userID, tenantID).Scan(&isMember)
	if err != nil {
		return false, ErrDBOperation
	}

	return isMember, nil
}
