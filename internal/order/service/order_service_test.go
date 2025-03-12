package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	authctx "github.com/unsavory/silocore-go/internal/auth/context"
	"github.com/unsavory/silocore-go/internal/database/transaction"
)

func setupMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *DBOrderService) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	service := NewDBOrderService(db)
	return db, mock, service
}

// createContextWithTenant creates a context with tenant ID
func createContextWithTenant(tenantID int64) context.Context {
	ctx := context.Background()
	return authctx.WithTenantID(ctx, &tenantID)
}

// setupTransaction sets up a transaction in the context
func setupTransaction(ctx context.Context, mock sqlmock.Sqlmock) context.Context {
	// We don't need to create a real transaction, just mock the expectations
	// The actual transaction will be created by the service when it calls Begin
	mockTx := mock.ExpectBegin()

	// Use the mock transaction directly
	return context.WithValue(ctx, transaction.TxKey, mockTx)
}

func TestGetOrder(t *testing.T) {
	db, mock, service := setupMock(t)
	defer db.Close()

	// Setup test data
	orderID := int64(1)
	tenantID := int64(42)
	userID := int64(100)
	now := time.Now()

	// Create context with tenant
	ctx := createContextWithTenant(tenantID)

	// Setup expectations for transaction
	mock.ExpectBegin()

	// Expect set_tenant_context call
	mock.ExpectExec("SELECT set_tenant_context\\(\\$1\\)").
		WithArgs(tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect query for order
	mock.ExpectQuery("SELECT order_id, tenant_id, user_id, order_number, status, total_amount, notes, created_at, updated_at").
		WithArgs(orderID, tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"order_id", "tenant_id", "user_id", "order_number", "status", "total_amount", "notes", "created_at", "updated_at"}).
			AddRow(orderID, tenantID, userID, "ORD-001", "pending", 100.50, "Test order", now, now))

	// Expect clear_tenant_context call
	mock.ExpectExec("SELECT clear_tenant_context\\(\\)").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect commit
	mock.ExpectCommit()

	// Setup transaction in context
	ctx = setupTransaction(ctx, mock)

	// Execute test
	order, err := service.GetOrder(ctx, orderID)

	// Verify results
	require.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, orderID, order.ID)
	assert.Equal(t, tenantID, order.TenantID)
	assert.Equal(t, userID, order.UserID)
	assert.Equal(t, "ORD-001", order.OrderNumber)
	assert.Equal(t, "pending", order.Status)
	assert.Equal(t, 100.50, order.TotalAmount)
	assert.Equal(t, "Test order", order.Notes)

	// Verify all expectations were met
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

func TestGetOrderNotFound(t *testing.T) {
	db, mock, service := setupMock(t)
	defer db.Close()

	// Test data
	orderID := int64(999)
	tenantID := int64(2)

	// Create context with tenant
	ctx := createContextWithTenant(tenantID)

	// Setup expectations for transaction
	mock.ExpectBegin()

	// Expect set_tenant_context call
	mock.ExpectExec("SELECT set_tenant_context\\(\\$1\\)").
		WithArgs(tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect query for order (not found)
	mock.ExpectQuery("SELECT order_id, tenant_id, user_id, order_number, status, total_amount, notes, created_at, updated_at").
		WithArgs(orderID, tenantID).
		WillReturnError(sql.ErrNoRows)

	// Expect clear_tenant_context call
	mock.ExpectExec("SELECT clear_tenant_context\\(\\)").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect rollback (since we're returning an error)
	mock.ExpectRollback()

	// Setup transaction in context
	ctx = setupTransaction(ctx, mock)

	// Execute test
	order, err := service.GetOrder(ctx, orderID)

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, order)
	assert.ErrorIs(t, err, ErrOrderNotFound)

	// Verify all expectations were met
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestListOrders(t *testing.T) {
	db, mock, service := setupMock(t)
	defer db.Close()

	// Test data
	tenantID := int64(42)
	now := time.Now()

	// Create context with tenant
	ctx := createContextWithTenant(tenantID)

	// Setup expectations for transaction
	mock.ExpectBegin()

	// Expect set_tenant_context call
	mock.ExpectExec("SELECT set_tenant_context\\(\\$1\\)").
		WithArgs(tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect query for orders
	mock.ExpectQuery("SELECT order_id, tenant_id, user_id, order_number, status, total_amount, notes, created_at, updated_at").
		WithArgs(tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"order_id", "tenant_id", "user_id", "order_number", "status", "total_amount", "notes", "created_at", "updated_at"}).
			AddRow(1, tenantID, 100, "ORD-001", "pending", 100.50, "Test order 1", now, now).
			AddRow(2, tenantID, 101, "ORD-002", "completed", 200.75, "Test order 2", now, now))

	// Expect clear_tenant_context call
	mock.ExpectExec("SELECT clear_tenant_context\\(\\)").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect commit
	mock.ExpectCommit()

	// Setup transaction in context
	ctx = setupTransaction(ctx, mock)

	// Execute test
	orders, err := service.ListOrders(ctx, OrderFilter{})

	// Verify results
	require.NoError(t, err)
	assert.Len(t, orders, 2)
	assert.Equal(t, int64(1), orders[0].ID)
	assert.Equal(t, int64(2), orders[1].ID)

	// Verify all expectations were met
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

func TestListOrdersWithFilters(t *testing.T) {
	db, mock, service := setupMock(t)
	defer db.Close()

	// Test data
	tenantID := int64(2)
	userID := int64(3)
	status := "pending"
	now := time.Now()

	// Create context with tenant ID
	ctx := createContextWithTenant(tenantID)

	// Setup expectations for transaction
	mock.ExpectBegin()

	// Expect set_tenant_context call
	mock.ExpectExec("SELECT set_tenant_context\\(\\$1\\)").
		WithArgs(tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Setup expectations for query
	rows := sqlmock.NewRows([]string{
		"order_id", "tenant_id", "user_id", "order_number", "status", "total_amount", "notes", "created_at", "updated_at",
	}).AddRow(
		1, tenantID, userID, "ORD-001", status, 100.50, "Test order", now, now,
	)

	mock.ExpectQuery(`SELECT order_id, tenant_id, user_id, order_number, status, total_amount, notes, created_at, updated_at FROM "order" WHERE tenant_id = \$1 AND status = \$2 AND user_id = \$3 ORDER BY created_at DESC`).
		WithArgs(tenantID, status, userID).
		WillReturnRows(rows)

	// Expect clear_tenant_context call
	mock.ExpectExec("SELECT clear_tenant_context\\(\\)").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect commit
	mock.ExpectCommit()

	// Execute test
	filter := OrderFilter{
		Status: status,
		UserID: &userID,
	}
	result, err := service.ListOrders(ctx, filter)

	// Verify results
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, userID, result[0].UserID)
	assert.Equal(t, status, result[0].Status)

	// Verify all expectations were met
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

func TestListUserOrders(t *testing.T) {
	db, mock, service := setupMock(t)
	defer db.Close()

	// Test data
	tenantID := int64(2)
	userID := int64(3)
	now := time.Now()

	// Create context with tenant ID
	ctx := createContextWithTenant(tenantID)

	// Setup expectations for transaction
	mock.ExpectBegin()

	// Expect set_tenant_context call
	mock.ExpectExec("SELECT set_tenant_context\\(\\$1\\)").
		WithArgs(tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Setup expectations for query
	rows := sqlmock.NewRows([]string{
		"order_id", "tenant_id", "user_id", "order_number", "status", "total_amount", "notes", "created_at", "updated_at",
	}).AddRow(
		1, tenantID, userID, "ORD-001", "pending", 100.50, "Test order", now, now,
	)

	mock.ExpectQuery(`SELECT order_id, tenant_id, user_id, order_number, status, total_amount, notes, created_at, updated_at FROM "order" WHERE tenant_id = \$1 AND user_id = \$2 ORDER BY created_at DESC`).
		WithArgs(tenantID, userID).
		WillReturnRows(rows)

	// Expect clear_tenant_context call
	mock.ExpectExec("SELECT clear_tenant_context\\(\\)").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect commit
	mock.ExpectCommit()

	// Execute test
	result, err := service.ListUserOrders(ctx, userID)

	// Verify results
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, userID, result[0].UserID)

	// Verify all expectations were met
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

func TestCreateOrder(t *testing.T) {
	db, mock, service := setupMock(t)
	defer db.Close()

	// Test data
	tenantID := int64(42)
	userID := int64(100)
	now := time.Now()
	order := &Order{
		TenantID:    tenantID,
		UserID:      userID,
		OrderNumber: "ORD-003",
		Status:      "pending",
		TotalAmount: 150.25,
		Notes:       "New test order",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Create context with tenant
	ctx := createContextWithTenant(tenantID)

	// Setup expectations for transaction
	mock.ExpectBegin()

	// Expect set_tenant_context call
	mock.ExpectExec("SELECT set_tenant_context\\(\\$1\\)").
		WithArgs(tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect insert query
	mock.ExpectQuery("INSERT INTO \"order\"").
		WithArgs(
			order.TenantID,
			order.UserID,
			order.OrderNumber,
			order.Status,
			order.TotalAmount,
			order.Notes,
			order.CreatedAt,
			order.UpdatedAt,
		).
		WillReturnRows(sqlmock.NewRows([]string{"order_id"}).AddRow(1))

	// Expect clear_tenant_context call
	mock.ExpectExec("SELECT clear_tenant_context\\(\\)").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect commit
	mock.ExpectCommit()

	// Setup transaction in context
	ctx = setupTransaction(ctx, mock)

	// Execute test
	createdOrder, err := service.CreateOrder(ctx, order)

	// Verify results
	require.NoError(t, err)
	assert.NotNil(t, createdOrder)
	assert.Equal(t, int64(1), createdOrder.ID)
	assert.Equal(t, tenantID, createdOrder.TenantID)
	assert.Equal(t, userID, createdOrder.UserID)
	assert.Equal(t, "ORD-003", createdOrder.OrderNumber)

	// Verify all expectations were met
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

func TestCreateOrderValidationErrors(t *testing.T) {
	db, _, service := setupMock(t)
	defer db.Close()

	// Test data
	tenantID := int64(2)

	// Create context with tenant ID
	ctx := createContextWithTenant(tenantID)

	testCases := []struct {
		name  string
		order *Order
	}{
		{
			name: "Missing tenant ID",
			order: &Order{
				UserID:      3,
				OrderNumber: "ORD-001",
				Status:      "pending",
				TotalAmount: 100.50,
			},
		},
		{
			name: "Missing user ID",
			order: &Order{
				TenantID:    tenantID,
				OrderNumber: "ORD-001",
				Status:      "pending",
				TotalAmount: 100.50,
			},
		},
		{
			name: "Missing order number",
			order: &Order{
				TenantID:    tenantID,
				UserID:      3,
				Status:      "pending",
				TotalAmount: 100.50,
			},
		},
		{
			name: "Negative total amount",
			order: &Order{
				TenantID:    tenantID,
				UserID:      3,
				OrderNumber: "ORD-001",
				Status:      "pending",
				TotalAmount: -10.0,
			},
		},
		{
			name: "Tenant ID mismatch",
			order: &Order{
				TenantID:    tenantID + 1, // Different from context
				UserID:      3,
				OrderNumber: "ORD-001",
				Status:      "pending",
				TotalAmount: 100.50,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.CreateOrder(ctx, tc.order)
			assert.Nil(t, result)
			assert.ErrorIs(t, err, ErrInvalidInput)
		})
	}
}

func TestUpdateOrder(t *testing.T) {
	db, mock, service := setupMock(t)
	defer db.Close()

	// Test data
	tenantID := int64(42)
	userID := int64(100)
	now := time.Now()
	order := &Order{
		ID:          1,
		TenantID:    tenantID,
		UserID:      userID,
		OrderNumber: "ORD-001",
		Status:      "completed",
		TotalAmount: 120.75,
		Notes:       "Updated test order",
		UpdatedAt:   now,
	}

	// Create context with tenant
	ctx := createContextWithTenant(tenantID)

	// Setup expectations for transaction
	mock.ExpectBegin()

	// Expect set_tenant_context call
	mock.ExpectExec("SELECT set_tenant_context\\(\\$1\\)").
		WithArgs(tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect update query
	mock.ExpectExec("UPDATE \"order\"").
		WithArgs(
			order.UserID,
			order.OrderNumber,
			order.Status,
			order.TotalAmount,
			order.Notes,
			order.UpdatedAt,
			order.ID,
			order.TenantID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect clear_tenant_context call
	mock.ExpectExec("SELECT clear_tenant_context\\(\\)").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect commit
	mock.ExpectCommit()

	// Setup transaction in context
	ctx = setupTransaction(ctx, mock)

	// Execute test
	err := service.UpdateOrder(ctx, order)

	// Verify results
	require.NoError(t, err)

	// Verify all expectations were met
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

func TestDeleteOrder(t *testing.T) {
	db, mock, service := setupMock(t)
	defer db.Close()

	// Test data
	orderID := int64(1)
	tenantID := int64(42)

	// Create context with tenant
	ctx := createContextWithTenant(tenantID)

	// Setup expectations for transaction
	mock.ExpectBegin()

	// Expect set_tenant_context call
	mock.ExpectExec("SELECT set_tenant_context\\(\\$1\\)").
		WithArgs(tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect delete query
	mock.ExpectExec("DELETE FROM \"order\"").
		WithArgs(orderID, tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect clear_tenant_context call
	mock.ExpectExec("SELECT clear_tenant_context\\(\\)").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect commit
	mock.ExpectCommit()

	// Setup transaction in context
	ctx = setupTransaction(ctx, mock)

	// Execute test
	err := service.DeleteOrder(ctx, orderID)

	// Verify results
	require.NoError(t, err)

	// Verify all expectations were met
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

func TestDeleteOrderNotFound(t *testing.T) {
	db, mock, service := setupMock(t)
	defer db.Close()

	// Test data
	orderID := int64(999)
	tenantID := int64(2)

	// Create context with tenant ID
	ctx := createContextWithTenant(tenantID)

	// Setup expectations for transaction
	mock.ExpectBegin()

	// Expect set_tenant_context call
	mock.ExpectExec("SELECT set_tenant_context\\(\\$1\\)").
		WithArgs(tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Setup expectations for DeleteOrder - no rows affected
	mock.ExpectExec(`DELETE FROM "order" WHERE order_id = \$1 AND tenant_id = \$2`).
		WithArgs(orderID, tenantID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	// Expect clear_tenant_context call
	mock.ExpectExec("SELECT clear_tenant_context\\(\\)").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect rollback due to error
	mock.ExpectRollback()

	// Execute test
	err := service.DeleteOrder(ctx, orderID)

	// Verify results
	assert.ErrorIs(t, err, ErrOrderNotFound)

	// Verify all expectations were met
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

func TestCountOrders(t *testing.T) {
	db, mock, service := setupMock(t)
	defer db.Close()

	// Test data
	tenantID := int64(42)

	// Create context with tenant
	ctx := createContextWithTenant(tenantID)

	// Setup expectations for transaction
	mock.ExpectBegin()

	// Expect set_tenant_context call
	mock.ExpectExec("SELECT set_tenant_context\\(\\$1\\)").
		WithArgs(tenantID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect count query
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").
		WithArgs(tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

	// Expect clear_tenant_context call
	mock.ExpectExec("SELECT clear_tenant_context\\(\\)").
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect commit
	mock.ExpectCommit()

	// Setup transaction in context
	ctx = setupTransaction(ctx, mock)

	// Execute test
	count, err := service.CountOrders(ctx, OrderFilter{})

	// Verify results
	require.NoError(t, err)
	assert.Equal(t, 5, count)

	// Verify all expectations were met
	err = mock.ExpectationsWereMet()
	require.NoError(t, err)
}

func TestNoTenantContext(t *testing.T) {
	db, _, service := setupMock(t)
	defer db.Close()

	// Create context without tenant ID
	ctx := context.Background()

	// Test various methods
	t.Run("GetOrder", func(t *testing.T) {
		order, err := service.GetOrder(ctx, 1)
		assert.Nil(t, order)
		assert.ErrorIs(t, err, ErrNoTenantContext)
	})

	t.Run("ListOrders", func(t *testing.T) {
		orders, err := service.ListOrders(ctx, OrderFilter{})
		assert.Nil(t, orders)
		assert.ErrorIs(t, err, ErrNoTenantContext)
	})

	t.Run("CreateOrder", func(t *testing.T) {
		order, err := service.CreateOrder(ctx, &Order{TenantID: 1, UserID: 1, OrderNumber: "ORD-001"})
		assert.Nil(t, order)
		assert.ErrorIs(t, err, ErrNoTenantContext)
	})

	t.Run("UpdateOrder", func(t *testing.T) {
		err := service.UpdateOrder(ctx, &Order{ID: 1})
		assert.ErrorIs(t, err, ErrNoTenantContext)
	})

	t.Run("DeleteOrder", func(t *testing.T) {
		err := service.DeleteOrder(ctx, 1)
		assert.ErrorIs(t, err, ErrNoTenantContext)
	})

	t.Run("CountOrders", func(t *testing.T) {
		count, err := service.CountOrders(ctx, OrderFilter{})
		assert.Equal(t, 0, count)
		assert.ErrorIs(t, err, ErrNoTenantContext)
	})
}
