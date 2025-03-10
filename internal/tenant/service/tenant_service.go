package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Common errors
var (
	ErrTenantNotFound = errors.New("tenant not found")
	ErrDBOperation    = errors.New("database operation failed")
	ErrInvalidInput   = errors.New("invalid input")
)

// Tenant represents a tenant in the system
type Tenant struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TenantMember represents a user's membership in a tenant
type TenantMember struct {
	UserID    int64     `json:"user_id"`
	TenantID  int64     `json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
}

// TenantService defines the interface for tenant-related operations
type TenantService interface {
	// GetTenant retrieves a tenant by ID
	GetTenant(ctx context.Context, tenantID int64) (*Tenant, error)

	// ListTenants retrieves all tenants
	ListTenants(ctx context.Context) ([]Tenant, error)

	// CreateTenant creates a new tenant
	CreateTenant(ctx context.Context, tenant *Tenant) (*Tenant, error)

	// UpdateTenant updates an existing tenant
	UpdateTenant(ctx context.Context, tenant *Tenant) error

	// DeleteTenant deletes a tenant
	DeleteTenant(ctx context.Context, tenantID int64) error

	// GetTenantMembers retrieves all members of a tenant
	GetTenantMembers(ctx context.Context, tenantID int64) ([]TenantMember, error)

	// AddTenantMember adds a user to a tenant
	AddTenantMember(ctx context.Context, userID int64, tenantID int64) error

	// RemoveTenantMember removes a user from a tenant
	RemoveTenantMember(ctx context.Context, userID int64, tenantID int64) error

	// GetUserTenants retrieves all tenants a user is a member of
	GetUserTenants(ctx context.Context, userID int64) ([]Tenant, error)
}

// DBTenantService implements TenantService using a database
type DBTenantService struct {
	db *sql.DB
}

// NewDBTenantService creates a new DBTenantService
func NewDBTenantService(db *sql.DB) *DBTenantService {
	return &DBTenantService{db: db}
}

// GetTenant retrieves a tenant by ID
func (s *DBTenantService) GetTenant(ctx context.Context, tenantID int64) (*Tenant, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM tenant
		WHERE id = $1
	`

	var tenant Tenant
	err := s.db.QueryRowContext(ctx, query, tenantID).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Description,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return &tenant, nil
}

// ListTenants retrieves all tenants
func (s *DBTenantService) ListTenants(ctx context.Context) ([]Tenant, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM tenant
		ORDER BY name
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var tenant Tenant
		if err := rows.Scan(
			&tenant.ID,
			&tenant.Name,
			&tenant.Description,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
		}
		tenants = append(tenants, tenant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return tenants, nil
}

// CreateTenant creates a new tenant
func (s *DBTenantService) CreateTenant(ctx context.Context, tenant *Tenant) (*Tenant, error) {
	if tenant.Name == "" {
		return nil, fmt.Errorf("%w: tenant name is required", ErrInvalidInput)
	}

	query := `
		INSERT INTO tenant (name, description)
		VALUES ($1, $2)
		RETURNING id, name, description, created_at, updated_at
	`

	err := s.db.QueryRowContext(ctx, query, tenant.Name, tenant.Description).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Description,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return tenant, nil
}

// UpdateTenant updates an existing tenant
func (s *DBTenantService) UpdateTenant(ctx context.Context, tenant *Tenant) error {
	if tenant.ID == 0 {
		return fmt.Errorf("%w: tenant ID is required", ErrInvalidInput)
	}

	if tenant.Name == "" {
		return fmt.Errorf("%w: tenant name is required", ErrInvalidInput)
	}

	query := `
		UPDATE tenant
		SET name = $1, description = $2, updated_at = NOW()
		WHERE id = $3
	`

	result, err := s.db.ExecContext(ctx, query, tenant.Name, tenant.Description, tenant.ID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	if rowsAffected == 0 {
		return ErrTenantNotFound
	}

	return nil
}

// DeleteTenant deletes a tenant
func (s *DBTenantService) DeleteTenant(ctx context.Context, tenantID int64) error {
	// Start a transaction to ensure atomicity
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}
	defer tx.Rollback()

	// Delete tenant members
	_, err = tx.ExecContext(ctx, "DELETE FROM tenant_member WHERE tenant_id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	// Delete tenant roles
	_, err = tx.ExecContext(ctx, "DELETE FROM tenant_role WHERE tenant_id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	// Delete tenant
	result, err := tx.ExecContext(ctx, "DELETE FROM tenant WHERE id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	if rowsAffected == 0 {
		return ErrTenantNotFound
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return nil
}

// GetTenantMembers retrieves all members of a tenant
func (s *DBTenantService) GetTenantMembers(ctx context.Context, tenantID int64) ([]TenantMember, error) {
	query := `
		SELECT user_id, tenant_id, created_at
		FROM tenant_member
		WHERE tenant_id = $1
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}
	defer rows.Close()

	var members []TenantMember
	for rows.Next() {
		var member TenantMember
		if err := rows.Scan(
			&member.UserID,
			&member.TenantID,
			&member.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
		}
		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return members, nil
}

// AddTenantMember adds a user to a tenant
func (s *DBTenantService) AddTenantMember(ctx context.Context, userID int64, tenantID int64) error {
	query := `
		INSERT INTO tenant_member (user_id, tenant_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, tenant_id) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, userID, tenantID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return nil
}

// RemoveTenantMember removes a user from a tenant
func (s *DBTenantService) RemoveTenantMember(ctx context.Context, userID int64, tenantID int64) error {
	// Start a transaction to ensure atomicity
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}
	defer tx.Rollback()

	// Remove tenant roles
	_, err = tx.ExecContext(ctx, "DELETE FROM tenant_role WHERE user_id = $1 AND tenant_id = $2", userID, tenantID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	// Remove tenant membership
	result, err := tx.ExecContext(ctx, "DELETE FROM tenant_member WHERE user_id = $1 AND tenant_id = $2", userID, tenantID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	if rowsAffected == 0 {
		return ErrTenantNotFound
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return nil
}

// GetUserTenants retrieves all tenants a user is a member of
func (s *DBTenantService) GetUserTenants(ctx context.Context, userID int64) ([]Tenant, error) {
	query := `
		SELECT t.id, t.name, t.description, t.created_at, t.updated_at
		FROM tenant t
		JOIN tenant_member tm ON t.id = tm.tenant_id
		WHERE tm.user_id = $1
		ORDER BY t.name
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var tenant Tenant
		if err := rows.Scan(
			&tenant.ID,
			&tenant.Name,
			&tenant.Description,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
		}
		tenants = append(tenants, tenant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return tenants, nil
}
