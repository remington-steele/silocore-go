package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Role represents a role in the system
type Role struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserRole represents a user's system-wide role
type UserRole struct {
	UserID    int64     `json:"user_id"`
	RoleID    int64     `json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

// TenantRole represents a user's tenant-specific role
type TenantRole struct {
	UserID    int64     `json:"user_id"`
	TenantID  int64     `json:"tenant_id"`
	RoleID    int64     `json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

// RoleService defines the interface for role-related operations
type RoleService interface {
	// GetRoles retrieves all roles in the system
	GetRoles(ctx context.Context) ([]Role, error)

	// GetRole retrieves a role by ID
	GetRole(ctx context.Context, roleID int64) (*Role, error)

	// GetRoleByName retrieves a role by name
	GetRoleByName(ctx context.Context, name string) (*Role, error)

	// AssignUserRole assigns a system-wide role to a user
	AssignUserRole(ctx context.Context, userID int64, roleID int64) error

	// RevokeUserRole revokes a system-wide role from a user
	RevokeUserRole(ctx context.Context, userID int64, roleID int64) error

	// GetUserRoles retrieves all system-wide roles for a user
	GetUserRoles(ctx context.Context, userID int64) ([]Role, error)

	// AssignTenantRole assigns a tenant-specific role to a user
	AssignTenantRole(ctx context.Context, userID int64, tenantID int64, roleID int64) error

	// RevokeTenantRole revokes a tenant-specific role from a user
	RevokeTenantRole(ctx context.Context, userID int64, tenantID int64, roleID int64) error

	// GetUserTenantRoles retrieves all tenant-specific roles for a user
	GetUserTenantRoles(ctx context.Context, userID int64, tenantID int64) ([]Role, error)
}

// DBRoleService implements RoleService using a database
type DBRoleService struct {
	db *sql.DB
}

// NewDBRoleService creates a new DBRoleService
func NewDBRoleService(db *sql.DB) *DBRoleService {
	return &DBRoleService{db: db}
}

// GetRoles retrieves all roles in the system
func (s *DBRoleService) GetRoles(ctx context.Context) ([]Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM role
		ORDER BY name
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.Description,
			&role.CreatedAt,
			&role.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return roles, nil
}

// GetRole retrieves a role by ID
func (s *DBRoleService) GetRole(ctx context.Context, roleID int64) (*Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM role
		WHERE id = $1
	`

	var role Role
	err := s.db.QueryRowContext(ctx, query, roleID).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.CreatedAt,
		&role.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("role not found: %d", roleID)
		}
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return &role, nil
}

// GetRoleByName retrieves a role by name
func (s *DBRoleService) GetRoleByName(ctx context.Context, name string) (*Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM role
		WHERE name = $1
	`

	var role Role
	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&role.ID,
		&role.Name,
		&role.Description,
		&role.CreatedAt,
		&role.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("role not found: %s", name)
		}
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return &role, nil
}

// AssignUserRole assigns a system-wide role to a user
func (s *DBRoleService) AssignUserRole(ctx context.Context, userID int64, roleID int64) error {
	query := `
		INSERT INTO user_role (user_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, userID, roleID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return nil
}

// RevokeUserRole revokes a system-wide role from a user
func (s *DBRoleService) RevokeUserRole(ctx context.Context, userID int64, roleID int64) error {
	query := `
		DELETE FROM user_role
		WHERE user_id = $1 AND role_id = $2
	`

	result, err := s.db.ExecContext(ctx, query, userID, roleID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user %d does not have role %d", userID, roleID)
	}

	return nil
}

// GetUserRoles retrieves all system-wide roles for a user
func (s *DBRoleService) GetUserRoles(ctx context.Context, userID int64) ([]Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.created_at, r.updated_at
		FROM role r
		JOIN user_role ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY r.name
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.Description,
			&role.CreatedAt,
			&role.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return roles, nil
}

// AssignTenantRole assigns a tenant-specific role to a user
func (s *DBRoleService) AssignTenantRole(ctx context.Context, userID int64, tenantID int64, roleID int64) error {
	// Start a transaction to ensure atomicity
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}
	defer tx.Rollback()

	// Ensure user is a member of the tenant
	var isMember bool
	err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM tenant_member WHERE user_id = $1 AND tenant_id = $2)", userID, tenantID).Scan(&isMember)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	if !isMember {
		// Add user as a tenant member first
		_, err = tx.ExecContext(ctx, "INSERT INTO tenant_member (user_id, tenant_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", userID, tenantID)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrDBOperation, err)
		}
	}

	// Assign the tenant role
	_, err = tx.ExecContext(ctx, "INSERT INTO tenant_role (user_id, tenant_id, role_id) VALUES ($1, $2, $3) ON CONFLICT (user_id, tenant_id, role_id) DO NOTHING", userID, tenantID, roleID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return nil
}

// RevokeTenantRole revokes a tenant-specific role from a user
func (s *DBRoleService) RevokeTenantRole(ctx context.Context, userID int64, tenantID int64, roleID int64) error {
	query := `
		DELETE FROM tenant_role
		WHERE user_id = $1 AND tenant_id = $2 AND role_id = $3
	`

	result, err := s.db.ExecContext(ctx, query, userID, tenantID, roleID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user %d does not have role %d for tenant %d", userID, roleID, tenantID)
	}

	return nil
}

// GetUserTenantRoles retrieves all tenant-specific roles for a user
func (s *DBRoleService) GetUserTenantRoles(ctx context.Context, userID int64, tenantID int64) ([]Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.created_at, r.updated_at
		FROM role r
		JOIN tenant_role tr ON r.id = tr.role_id
		WHERE tr.user_id = $1 AND tr.tenant_id = $2
		ORDER BY r.name
	`

	rows, err := s.db.QueryContext(ctx, query, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.Description,
			&role.CreatedAt,
			&role.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return roles, nil
}
