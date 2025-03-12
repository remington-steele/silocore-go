package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

// Common errors
var (
	ErrMemberNotFound = errors.New("tenant member not found")
	ErrDBOperationTM  = errors.New("database operation failed")
)

// TenantMembership represents a user's membership in a tenant
type TenantMembership struct {
	UserID    int64     `json:"user_id"`
	TenantID  int64     `json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
}

// TenantMemberService defines the interface for tenant membership operations
type TenantMemberService interface {
	// GetUserTenantMemberships retrieves all tenant memberships for a user
	GetUserTenantMemberships(ctx context.Context, userID int64) ([]TenantMembership, error)

	// GetUserDefaultTenant retrieves a user's default tenant ID (first tenant in membership list)
	GetUserDefaultTenant(ctx context.Context, userID int64) (*int64, error)

	// IsTenantMember checks if a user is a member of a specific tenant
	IsTenantMember(ctx context.Context, userID int64, tenantID int64) (bool, error)

	// AddTenantMember adds a user to a tenant
	AddTenantMember(ctx context.Context, userID int64, tenantID int64) error

	// RemoveTenantMember removes a user from a tenant
	RemoveTenantMember(ctx context.Context, userID int64, tenantID int64) error
}

// DBTenantMemberService implements TenantMemberService using a database
type DBTenantMemberService struct {
	db *sql.DB
}

// NewDBTenantMemberService creates a new DBTenantMemberService
func NewDBTenantMemberService(db *sql.DB) *DBTenantMemberService {
	return &DBTenantMemberService{db: db}
}

// GetUserTenantMemberships retrieves all tenant memberships for a user
func (s *DBTenantMemberService) GetUserTenantMemberships(ctx context.Context, userID int64) ([]TenantMembership, error) {
	query := `
		SELECT tenant_id, user_id, created_at
		FROM tenant_member
		WHERE user_id = $1
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		log.Printf("[ERROR] Database error when getting tenant memberships for user %d: %v", userID, err)
		return nil, fmt.Errorf("%w: %v", ErrDBOperationTM, err)
	}
	defer rows.Close()

	var memberships []TenantMembership
	for rows.Next() {
		var membership TenantMembership
		if err := rows.Scan(
			&membership.TenantID,
			&membership.UserID,
			&membership.CreatedAt,
		); err != nil {
			log.Printf("[ERROR] Error scanning tenant membership row for user %d: %v", userID, err)
			return nil, fmt.Errorf("%w: %v", ErrDBOperationTM, err)
		}
		memberships = append(memberships, membership)
	}

	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] Error iterating tenant membership rows for user %d: %v", userID, err)
		return nil, fmt.Errorf("%w: %v", ErrDBOperationTM, err)
	}

	return memberships, nil
}

// GetUserDefaultTenant retrieves a user's default tenant ID (first tenant in membership list)
func (s *DBTenantMemberService) GetUserDefaultTenant(ctx context.Context, userID int64) (*int64, error) {
	// Get the first tenant membership for the user (ordered by created_at)
	query := `
		SELECT tenant_id
		FROM tenant_member
		WHERE user_id = $1
		ORDER BY created_at ASC
		LIMIT 1
	`

	var tenantID int64
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// User has no tenant memberships, which is allowed
			log.Printf("[INFO] No tenant memberships found for user %d", userID)
			return nil, nil
		}
		log.Printf("[ERROR] Database error when getting default tenant for user %d: %v", userID, err)
		return nil, fmt.Errorf("%w: %v", ErrDBOperationTM, err)
	}

	return &tenantID, nil
}

// IsTenantMember checks if a user is a member of a specific tenant
func (s *DBTenantMemberService) IsTenantMember(ctx context.Context, userID int64, tenantID int64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM tenant_member 
			WHERE user_id = $1 AND tenant_id = $2
		)
	`

	var isMember bool
	err := s.db.QueryRowContext(ctx, query, userID, tenantID).Scan(&isMember)
	if err != nil {
		log.Printf("[ERROR] Database error when checking tenant membership for user %d in tenant %d: %v", userID, tenantID, err)
		return false, fmt.Errorf("%w: %v", ErrDBOperationTM, err)
	}

	return isMember, nil
}

// AddTenantMember adds a user to a tenant
func (s *DBTenantMemberService) AddTenantMember(ctx context.Context, userID int64, tenantID int64) error {
	query := `
		INSERT INTO tenant_member (user_id, tenant_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, tenant_id) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query, userID, tenantID)
	if err != nil {
		log.Printf("[ERROR] Database error when adding user %d to tenant %d: %v", userID, tenantID, err)
		return fmt.Errorf("%w: %v", ErrDBOperationTM, err)
	}

	log.Printf("[INFO] User %d successfully added to tenant %d", userID, tenantID)
	return nil
}

// RemoveTenantMember removes a user from a tenant
func (s *DBTenantMemberService) RemoveTenantMember(ctx context.Context, userID int64, tenantID int64) error {
	// Start a transaction to ensure atomicity
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("[ERROR] Failed to begin transaction when removing user %d from tenant %d: %v", userID, tenantID, err)
		return fmt.Errorf("%w: %v", ErrDBOperationTM, err)
	}
	defer tx.Rollback()

	// Remove tenant roles
	_, err = tx.ExecContext(ctx, "DELETE FROM tenant_role WHERE user_id = $1 AND tenant_id = $2", userID, tenantID)
	if err != nil {
		log.Printf("[ERROR] Failed to delete tenant roles for user %d in tenant %d: %v", userID, tenantID, err)
		return fmt.Errorf("%w: %v", ErrDBOperationTM, err)
	}

	// Remove tenant membership
	result, err := tx.ExecContext(ctx, "DELETE FROM tenant_member WHERE user_id = $1 AND tenant_id = $2", userID, tenantID)
	if err != nil {
		log.Printf("[ERROR] Failed to delete tenant membership for user %d in tenant %d: %v", userID, tenantID, err)
		return fmt.Errorf("%w: %v", ErrDBOperationTM, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[ERROR] Failed to get rows affected when removing user %d from tenant %d: %v", userID, tenantID, err)
		return fmt.Errorf("%w: %v", ErrDBOperationTM, err)
	}

	if rowsAffected == 0 {
		log.Printf("[WARN] User %d is not a member of tenant %d", userID, tenantID)
		return ErrMemberNotFound
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Printf("[ERROR] Failed to commit transaction when removing user %d from tenant %d: %v", userID, tenantID, err)
		return fmt.Errorf("%w: %v", ErrDBOperationTM, err)
	}

	log.Printf("[INFO] User %d successfully removed from tenant %d", userID, tenantID)
	return nil
}
