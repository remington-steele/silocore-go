package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	authctx "github.com/unsavory/silocore-go/internal/auth/context"
	"github.com/unsavory/silocore-go/internal/database/transaction"
)

// Common errors
var (
	ErrOrderNotFound   = errors.New("order not found")
	ErrDBOperation     = errors.New("database operation failed")
	ErrInvalidInput    = errors.New("invalid input")
	ErrNoTenantContext = errors.New("tenant context is required")
)

// Order represents an order in the system
type Order struct {
	ID          int64     `json:"id"`
	TenantID    int64     `json:"tenant_id"`
	UserID      int64     `json:"user_id"`
	OrderNumber string    `json:"order_number"`
	Status      string    `json:"status"`
	TotalAmount float64   `json:"total_amount"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// OrderFilter represents filters for listing orders
type OrderFilter struct {
	Status string
	UserID *int64
	Limit  int
	Offset int
}

// OrderService defines the interface for order-related operations
type OrderService interface {
	// GetOrder retrieves an order by ID
	GetOrder(ctx context.Context, orderID int64) (*Order, error)

	// ListOrders retrieves orders for the current tenant with optional filters
	ListOrders(ctx context.Context, filter OrderFilter) ([]Order, error)

	// ListUserOrders retrieves orders for a specific user in the current tenant
	ListUserOrders(ctx context.Context, userID int64) ([]Order, error)

	// CreateOrder creates a new order
	CreateOrder(ctx context.Context, order *Order) (*Order, error)

	// UpdateOrder updates an existing order
	UpdateOrder(ctx context.Context, order *Order) error

	// DeleteOrder deletes an order
	DeleteOrder(ctx context.Context, orderID int64) error

	// CountOrders counts orders for the current tenant with optional filters
	CountOrders(ctx context.Context, filter OrderFilter) (int, error)
}

// DBOrderService implements OrderService using a database
type DBOrderService struct {
	txManager *transaction.Manager
}

// NewDBOrderService creates a new DBOrderService
func NewDBOrderService(db *sql.DB) *DBOrderService {
	return &DBOrderService{
		txManager: transaction.NewManager(db),
	}
}

// GetOrder retrieves an order by ID
func (s *DBOrderService) GetOrder(ctx context.Context, orderID int64) (*Order, error) {
	// Verify tenant context
	tenantID, err := authctx.GetTenantID(ctx)
	if err != nil || tenantID == nil {
		return nil, ErrNoTenantContext
	}

	// Get transaction from context
	tx, err := s.txManager.GetTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	// Query with explicit tenant_id filter for additional security
	query := `
		SELECT order_id, tenant_id, user_id, order_number, status, total_amount, notes, created_at, updated_at
		FROM "order"
		WHERE order_id = $1 AND tenant_id = $2
	`

	var order Order
	err = tx.QueryRowContext(ctx, query, orderID, *tenantID).Scan(
		&order.ID,
		&order.TenantID,
		&order.UserID,
		&order.OrderNumber,
		&order.Status,
		&order.TotalAmount,
		&order.Notes,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return &order, nil
}

// ListOrders retrieves orders for the current tenant with optional filters
func (s *DBOrderService) ListOrders(ctx context.Context, filter OrderFilter) ([]Order, error) {
	// Verify tenant context
	tenantID, err := authctx.GetTenantID(ctx)
	if err != nil || tenantID == nil {
		return nil, ErrNoTenantContext
	}

	// Get transaction from context
	tx, err := s.txManager.GetTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	// Base query with explicit tenant_id filter
	query := `
		SELECT order_id, tenant_id, user_id, order_number, status, total_amount, notes, created_at, updated_at
		FROM "order"
		WHERE tenant_id = $1
	`

	// Build query with additional filters
	var args []interface{}
	args = append(args, *tenantID)
	argPos := 2

	// Add status filter if provided
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filter.Status)
		argPos++
	}

	// Add user filter if provided
	if filter.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argPos)
		args = append(args, *filter.UserID)
		argPos++
	}

	// Add order by
	query += " ORDER BY created_at DESC"

	// Add limit and offset
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argPos)
		args = append(args, filter.Limit)
		argPos++

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argPos)
			args = append(args, filter.Offset)
		}
	}

	// Execute query
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}
	defer rows.Close()

	// Process results
	var orders []Order
	for rows.Next() {
		var order Order
		err := rows.Scan(
			&order.ID,
			&order.TenantID,
			&order.UserID,
			&order.OrderNumber,
			&order.Status,
			&order.TotalAmount,
			&order.Notes,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return orders, nil
}

// ListUserOrders retrieves orders for a specific user in the current tenant
func (s *DBOrderService) ListUserOrders(ctx context.Context, userID int64) ([]Order, error) {
	filter := OrderFilter{
		UserID: &userID,
	}
	return s.ListOrders(ctx, filter)
}

// CreateOrder creates a new order
func (s *DBOrderService) CreateOrder(ctx context.Context, order *Order) (*Order, error) {
	// Validate input
	if order.TenantID <= 0 {
		return nil, fmt.Errorf("%w: tenant ID is required", ErrInvalidInput)
	}
	if order.UserID <= 0 {
		return nil, fmt.Errorf("%w: user ID is required", ErrInvalidInput)
	}
	if order.OrderNumber == "" {
		return nil, fmt.Errorf("%w: order number is required", ErrInvalidInput)
	}
	if order.Status == "" {
		// Set default status if not provided
		order.Status = "pending"
	}
	if order.TotalAmount < 0 {
		return nil, fmt.Errorf("%w: total amount cannot be negative", ErrInvalidInput)
	}

	// Ensure the tenant ID in the order matches the tenant ID in the context
	tenantID, err := authctx.GetTenantID(ctx)
	if err != nil || tenantID == nil {
		return nil, ErrNoTenantContext
	}

	if order.TenantID != *tenantID {
		return nil, fmt.Errorf("%w: tenant ID in order does not match tenant context", ErrInvalidInput)
	}

	// Set timestamps
	now := time.Now()
	order.CreatedAt = now
	order.UpdatedAt = now

	// Get transaction from context
	tx, err := s.txManager.GetTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	// Insert order
	query := `
		INSERT INTO "order" (tenant_id, user_id, order_number, status, total_amount, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING order_id
	`

	err = tx.QueryRowContext(
		ctx,
		query,
		order.TenantID,
		order.UserID,
		order.OrderNumber,
		order.Status,
		order.TotalAmount,
		order.Notes,
		order.CreatedAt,
		order.UpdatedAt,
	).Scan(&order.ID)

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return order, nil
}

// UpdateOrder updates an existing order
func (s *DBOrderService) UpdateOrder(ctx context.Context, order *Order) error {
	// Validate input
	if order.ID <= 0 {
		return fmt.Errorf("%w: order ID is required", ErrInvalidInput)
	}
	if order.TenantID <= 0 {
		return fmt.Errorf("%w: tenant ID is required", ErrInvalidInput)
	}
	if order.UserID <= 0 {
		return fmt.Errorf("%w: user ID is required", ErrInvalidInput)
	}
	if order.OrderNumber == "" {
		return fmt.Errorf("%w: order number is required", ErrInvalidInput)
	}
	if order.Status == "" {
		return fmt.Errorf("%w: status is required", ErrInvalidInput)
	}
	if order.TotalAmount < 0 {
		return fmt.Errorf("%w: total amount cannot be negative", ErrInvalidInput)
	}

	// Ensure the tenant ID in the order matches the tenant ID in the context
	tenantID, err := authctx.GetTenantID(ctx)
	if err != nil || tenantID == nil {
		return ErrNoTenantContext
	}

	if order.TenantID != *tenantID {
		return fmt.Errorf("%w: tenant ID in order does not match tenant context", ErrInvalidInput)
	}

	// Update timestamp
	order.UpdatedAt = time.Now()

	// Get transaction from context
	tx, err := s.txManager.GetTx(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	// Update order with explicit tenant_id filter
	query := `
		UPDATE "order"
		SET user_id = $1, order_number = $2, status = $3, total_amount = $4, notes = $5, updated_at = $6
		WHERE order_id = $7 AND tenant_id = $8
	`

	result, err := tx.ExecContext(
		ctx,
		query,
		order.UserID,
		order.OrderNumber,
		order.Status,
		order.TotalAmount,
		order.Notes,
		order.UpdatedAt,
		order.ID,
		order.TenantID,
	)

	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	// Check if the order was found
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	if rowsAffected == 0 {
		return ErrOrderNotFound
	}

	return nil
}

// DeleteOrder deletes an order
func (s *DBOrderService) DeleteOrder(ctx context.Context, orderID int64) error {
	// Verify tenant context
	tenantID, err := authctx.GetTenantID(ctx)
	if err != nil || tenantID == nil {
		return ErrNoTenantContext
	}

	// Get transaction from context
	tx, err := s.txManager.GetTx(ctx)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	// Delete with explicit tenant_id filter
	query := `
		DELETE FROM "order"
		WHERE order_id = $1 AND tenant_id = $2
	`

	result, err := tx.ExecContext(ctx, query, orderID, *tenantID)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	// Check if the order was found
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	if rowsAffected == 0 {
		return ErrOrderNotFound
	}

	return nil
}

// CountOrders counts orders for the current tenant with optional filters
func (s *DBOrderService) CountOrders(ctx context.Context, filter OrderFilter) (int, error) {
	// Verify tenant context
	tenantID, err := authctx.GetTenantID(ctx)
	if err != nil || tenantID == nil {
		return 0, ErrNoTenantContext
	}

	// Get transaction from context
	tx, err := s.txManager.GetTx(ctx)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	// Base query with explicit tenant_id filter
	query := `
		SELECT COUNT(*)
		FROM "order"
		WHERE tenant_id = $1
	`

	// Build query with additional filters
	var args []interface{}
	args = append(args, *tenantID)
	argPos := 2

	// Add status filter if provided
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, filter.Status)
		argPos++
	}

	// Add user filter if provided
	if filter.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argPos)
		args = append(args, *filter.UserID)
	}

	// Execute query
	var count int
	err = tx.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return count, nil
}
