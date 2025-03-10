package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
)

// Common errors
var (
	ErrNoTransaction = errors.New("no transaction in context")
)

// Manager provides transaction management functionality
type Manager struct {
	db *sql.DB
}

// NewManager creates a new transaction manager
func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// GetDB returns the database connection
func (m *Manager) GetDB() *sql.DB {
	return m.db
}

// Begin starts a new transaction and adds it to the context
func (m *Manager) Begin(ctx context.Context) (context.Context, *sql.Tx, error) {
	// Check if there's already a transaction in the context
	if tx, ok := ctx.Value(TxKey).(*sql.Tx); ok {
		// Return the existing transaction
		return ctx, tx, nil
	}

	// Start a new transaction
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return ctx, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Add the transaction to the context
	ctx = context.WithValue(ctx, TxKey, tx)
	return ctx, tx, nil
}

// GetTx retrieves the transaction from the context
func (m *Manager) GetTx(ctx context.Context) (*sql.Tx, error) {
	tx, ok := ctx.Value(TxKey).(*sql.Tx)
	if !ok {
		return nil, ErrNoTransaction
	}
	return tx, nil
}

// Commit commits the transaction in the context
func (m *Manager) Commit(ctx context.Context) error {
	tx, err := m.GetTx(ctx)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Rollback rolls back the transaction in the context
func (m *Manager) Rollback(ctx context.Context) error {
	tx, err := m.GetTx(ctx)
	if err != nil {
		return err
	}
	return tx.Rollback()
}

// WithTransaction executes a function within a transaction
// If there's already a transaction in the context, it will use that transaction
// Otherwise, it will start a new transaction
func (m *Manager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// Check if there's already a transaction in the context
	_, ok := ctx.Value(TxKey).(*sql.Tx)
	if ok {
		// Use the existing transaction
		return fn(ctx)
	}

	// Start a new transaction
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Add the transaction to the context
	ctx = context.WithValue(ctx, TxKey, tx)

	// Execute the function
	err = fn(ctx)
	if err != nil {
		// Rollback the transaction on error
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("Error rolling back transaction: %v", rbErr)
		}
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// SetTenantContext sets the tenant context for the current database session
func (m *Manager) SetTenantContext(ctx context.Context, tenantID int64) error {
	tx, err := m.GetTx(ctx)
	if err != nil {
		return err
	}

	// Set tenant context in the database session
	_, err = tx.ExecContext(ctx, "SELECT set_tenant_context($1)", tenantID)
	if err != nil {
		return fmt.Errorf("failed to set tenant context: %w", err)
	}

	return nil
}

// ClearTenantContext clears the tenant context for the current database session
func (m *Manager) ClearTenantContext(ctx context.Context) error {
	tx, err := m.GetTx(ctx)
	if err != nil {
		return err
	}

	// Clear tenant context in the database session
	_, err = tx.ExecContext(ctx, "SELECT clear_tenant_context()")
	if err != nil {
		return fmt.Errorf("failed to clear tenant context: %w", err)
	}

	return nil
}
