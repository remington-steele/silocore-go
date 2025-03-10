package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *DBTenantService) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	service := NewDBTenantService(db)
	return db, mock, service
}

func TestGetTenant(t *testing.T) {
	db, mock, service := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	tenantID := int64(1)

	t.Run("Successful retrieval", func(t *testing.T) {
		// Setup mock expectations
		rows := sqlmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"}).
			AddRow(tenantID, "Test Tenant", "Test Description", time.Now(), time.Now())

		mock.ExpectQuery("SELECT id, name, description, created_at, updated_at FROM tenant WHERE id = \\$1").
			WithArgs(tenantID).
			WillReturnRows(rows)

		// Execute
		tenant, err := service.GetTenant(ctx, tenantID)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, tenant)
		assert.Equal(t, tenantID, tenant.ID)
		assert.Equal(t, "Test Tenant", tenant.Name)
		assert.Equal(t, "Test Description", tenant.Description)
	})

	t.Run("Tenant not found", func(t *testing.T) {
		// Setup mock expectations
		mock.ExpectQuery("SELECT id, name, description, created_at, updated_at FROM tenant WHERE id = \\$1").
			WithArgs(tenantID).
			WillReturnError(sql.ErrNoRows)

		// Execute
		tenant, err := service.GetTenant(ctx, tenantID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, tenant)
		assert.Equal(t, ErrTenantNotFound, err)
	})

	t.Run("Database error", func(t *testing.T) {
		// Setup mock expectations
		dbErr := errors.New("database error")
		mock.ExpectQuery("SELECT id, name, description, created_at, updated_at FROM tenant WHERE id = \\$1").
			WithArgs(tenantID).
			WillReturnError(dbErr)

		// Execute
		tenant, err := service.GetTenant(ctx, tenantID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, tenant)
		assert.True(t, errors.Is(err, ErrDBOperation))
	})
}

func TestListTenants(t *testing.T) {
	db, mock, service := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()

	t.Run("Successful retrieval", func(t *testing.T) {
		// Setup mock expectations
		rows := sqlmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"}).
			AddRow(1, "Tenant 1", "Description 1", time.Now(), time.Now()).
			AddRow(2, "Tenant 2", "Description 2", time.Now(), time.Now())

		mock.ExpectQuery("SELECT id, name, description, created_at, updated_at FROM tenant ORDER BY name").
			WillReturnRows(rows)

		// Execute
		tenants, err := service.ListTenants(ctx)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, tenants, 2)
		assert.Equal(t, int64(1), tenants[0].ID)
		assert.Equal(t, "Tenant 1", tenants[0].Name)
		assert.Equal(t, int64(2), tenants[1].ID)
		assert.Equal(t, "Tenant 2", tenants[1].Name)
	})

	t.Run("Empty result", func(t *testing.T) {
		// Setup mock expectations
		rows := sqlmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"})

		mock.ExpectQuery("SELECT id, name, description, created_at, updated_at FROM tenant ORDER BY name").
			WillReturnRows(rows)

		// Execute
		tenants, err := service.ListTenants(ctx)

		// Assert
		assert.NoError(t, err)
		assert.Empty(t, tenants)
	})

	t.Run("Database error", func(t *testing.T) {
		// Setup mock expectations
		dbErr := errors.New("database error")
		mock.ExpectQuery("SELECT id, name, description, created_at, updated_at FROM tenant ORDER BY name").
			WillReturnError(dbErr)

		// Execute
		tenants, err := service.ListTenants(ctx)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, tenants)
		assert.True(t, errors.Is(err, ErrDBOperation))
	})
}

func TestCreateTenant(t *testing.T) {
	db, mock, service := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	now := time.Now()

	t.Run("Successful creation", func(t *testing.T) {
		// Setup
		tenant := &Tenant{
			Name:        "New Tenant",
			Description: "New Description",
		}

		// Setup mock expectations
		rows := sqlmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"}).
			AddRow(1, tenant.Name, tenant.Description, now, now)

		mock.ExpectQuery("INSERT INTO tenant \\(name, description\\) VALUES \\(\\$1, \\$2\\) RETURNING id, name, description, created_at, updated_at").
			WithArgs(tenant.Name, tenant.Description).
			WillReturnRows(rows)

		// Execute
		createdTenant, err := service.CreateTenant(ctx, tenant)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, createdTenant)
		assert.Equal(t, int64(1), createdTenant.ID)
		assert.Equal(t, tenant.Name, createdTenant.Name)
		assert.Equal(t, tenant.Description, createdTenant.Description)
	})

	t.Run("Invalid input - empty name", func(t *testing.T) {
		// Setup
		tenant := &Tenant{
			Description: "New Description",
		}

		// Execute
		createdTenant, err := service.CreateTenant(ctx, tenant)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, createdTenant)
		assert.True(t, errors.Is(err, ErrInvalidInput))
	})

	t.Run("Database error", func(t *testing.T) {
		// Setup
		tenant := &Tenant{
			Name:        "New Tenant",
			Description: "New Description",
		}

		// Setup mock expectations
		dbErr := errors.New("database error")
		mock.ExpectQuery("INSERT INTO tenant \\(name, description\\) VALUES \\(\\$1, \\$2\\) RETURNING id, name, description, created_at, updated_at").
			WithArgs(tenant.Name, tenant.Description).
			WillReturnError(dbErr)

		// Execute
		createdTenant, err := service.CreateTenant(ctx, tenant)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, createdTenant)
		assert.True(t, errors.Is(err, ErrDBOperation))
	})
}

func TestUpdateTenant(t *testing.T) {
	db, mock, service := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()

	t.Run("Successful update", func(t *testing.T) {
		// Setup
		tenant := &Tenant{
			ID:          1,
			Name:        "Updated Tenant",
			Description: "Updated Description",
		}

		// Setup mock expectations
		mock.ExpectExec("UPDATE tenant SET name = \\$1, description = \\$2, updated_at = NOW\\(\\) WHERE id = \\$3").
			WithArgs(tenant.Name, tenant.Description, tenant.ID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Execute
		err := service.UpdateTenant(ctx, tenant)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Invalid input - missing ID", func(t *testing.T) {
		// Setup
		tenant := &Tenant{
			Name:        "Updated Tenant",
			Description: "Updated Description",
		}

		// Execute
		err := service.UpdateTenant(ctx, tenant)

		// Assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidInput))
	})

	t.Run("Invalid input - empty name", func(t *testing.T) {
		// Setup
		tenant := &Tenant{
			ID:          1,
			Description: "Updated Description",
		}

		// Execute
		err := service.UpdateTenant(ctx, tenant)

		// Assert
		assert.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidInput))
	})

	t.Run("Tenant not found", func(t *testing.T) {
		// Setup
		tenant := &Tenant{
			ID:          999,
			Name:        "Updated Tenant",
			Description: "Updated Description",
		}

		// Setup mock expectations
		mock.ExpectExec("UPDATE tenant SET name = \\$1, description = \\$2, updated_at = NOW\\(\\) WHERE id = \\$3").
			WithArgs(tenant.Name, tenant.Description, tenant.ID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// Execute
		err := service.UpdateTenant(ctx, tenant)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrTenantNotFound, err)
	})
}

func TestDeleteTenant(t *testing.T) {
	db, mock, service := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	tenantID := int64(1)

	t.Run("Successful deletion", func(t *testing.T) {
		// Setup mock expectations
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM tenant_member WHERE tenant_id = \\$1").
			WithArgs(tenantID).
			WillReturnResult(sqlmock.NewResult(0, 2))
		mock.ExpectExec("DELETE FROM tenant_role WHERE tenant_id = \\$1").
			WithArgs(tenantID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("DELETE FROM tenant WHERE id = \\$1").
			WithArgs(tenantID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		// Execute
		err := service.DeleteTenant(ctx, tenantID)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Tenant not found", func(t *testing.T) {
		// Setup mock expectations
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM tenant_member WHERE tenant_id = \\$1").
			WithArgs(tenantID).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("DELETE FROM tenant_role WHERE tenant_id = \\$1").
			WithArgs(tenantID).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("DELETE FROM tenant WHERE id = \\$1").
			WithArgs(tenantID).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		// Execute
		err := service.DeleteTenant(ctx, tenantID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrTenantNotFound, err)
	})
}

func TestGetTenantMembers(t *testing.T) {
	db, mock, service := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	tenantID := int64(1)
	now := time.Now()

	t.Run("Successful retrieval", func(t *testing.T) {
		// Setup mock expectations
		rows := sqlmock.NewRows([]string{"user_id", "tenant_id", "created_at"}).
			AddRow(1, tenantID, now).
			AddRow(2, tenantID, now)

		mock.ExpectQuery("SELECT user_id, tenant_id, created_at FROM tenant_member WHERE tenant_id = \\$1").
			WithArgs(tenantID).
			WillReturnRows(rows)

		// Execute
		members, err := service.GetTenantMembers(ctx, tenantID)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, members, 2)
		assert.Equal(t, int64(1), members[0].UserID)
		assert.Equal(t, tenantID, members[0].TenantID)
		assert.Equal(t, int64(2), members[1].UserID)
	})

	t.Run("Empty result", func(t *testing.T) {
		// Setup mock expectations
		rows := sqlmock.NewRows([]string{"user_id", "tenant_id", "created_at"})

		mock.ExpectQuery("SELECT user_id, tenant_id, created_at FROM tenant_member WHERE tenant_id = \\$1").
			WithArgs(tenantID).
			WillReturnRows(rows)

		// Execute
		members, err := service.GetTenantMembers(ctx, tenantID)

		// Assert
		assert.NoError(t, err)
		assert.Empty(t, members)
	})
}

func TestAddTenantMember(t *testing.T) {
	db, mock, service := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	userID := int64(1)
	tenantID := int64(2)

	t.Run("Successful addition", func(t *testing.T) {
		// Setup mock expectations
		mock.ExpectExec("INSERT INTO tenant_member \\(user_id, tenant_id\\) VALUES \\(\\$1, \\$2\\) ON CONFLICT").
			WithArgs(userID, tenantID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Execute
		err := service.AddTenantMember(ctx, userID, tenantID)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Already a member (no error)", func(t *testing.T) {
		// Setup mock expectations
		mock.ExpectExec("INSERT INTO tenant_member \\(user_id, tenant_id\\) VALUES \\(\\$1, \\$2\\) ON CONFLICT").
			WithArgs(userID, tenantID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// Execute
		err := service.AddTenantMember(ctx, userID, tenantID)

		// Assert
		assert.NoError(t, err)
	})
}

func TestRemoveTenantMember(t *testing.T) {
	db, mock, service := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	userID := int64(1)
	tenantID := int64(2)

	t.Run("Successful removal", func(t *testing.T) {
		// Setup mock expectations
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM tenant_role WHERE user_id = \\$1 AND tenant_id = \\$2").
			WithArgs(userID, tenantID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec("DELETE FROM tenant_member WHERE user_id = \\$1 AND tenant_id = \\$2").
			WithArgs(userID, tenantID).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		// Execute
		err := service.RemoveTenantMember(ctx, userID, tenantID)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("Not a member", func(t *testing.T) {
		// Setup mock expectations
		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM tenant_role WHERE user_id = \\$1 AND tenant_id = \\$2").
			WithArgs(userID, tenantID).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("DELETE FROM tenant_member WHERE user_id = \\$1 AND tenant_id = \\$2").
			WithArgs(userID, tenantID).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		// Execute
		err := service.RemoveTenantMember(ctx, userID, tenantID)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, ErrTenantNotFound, err)
	})
}

func TestGetUserTenants(t *testing.T) {
	db, mock, service := setupMockDB(t)
	defer db.Close()

	ctx := context.Background()
	userID := int64(1)
	now := time.Now()

	t.Run("Successful retrieval", func(t *testing.T) {
		// Setup mock expectations
		rows := sqlmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"}).
			AddRow(1, "Tenant 1", "Description 1", now, now).
			AddRow(2, "Tenant 2", "Description 2", now, now)

		mock.ExpectQuery("SELECT t.id, t.name, t.description, t.created_at, t.updated_at FROM tenant t JOIN tenant_member tm ON t.id = tm.tenant_id WHERE tm.user_id = \\$1 ORDER BY t.name").
			WithArgs(userID).
			WillReturnRows(rows)

		// Execute
		tenants, err := service.GetUserTenants(ctx, userID)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, tenants, 2)
		assert.Equal(t, int64(1), tenants[0].ID)
		assert.Equal(t, "Tenant 1", tenants[0].Name)
		assert.Equal(t, int64(2), tenants[1].ID)
		assert.Equal(t, "Tenant 2", tenants[1].Name)
	})

	t.Run("No tenants", func(t *testing.T) {
		// Setup mock expectations
		rows := sqlmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"})

		mock.ExpectQuery("SELECT t.id, t.name, t.description, t.created_at, t.updated_at FROM tenant t JOIN tenant_member tm ON t.id = tm.tenant_id WHERE tm.user_id = \\$1 ORDER BY t.name").
			WithArgs(userID).
			WillReturnRows(rows)

		// Execute
		tenants, err := service.GetUserTenants(ctx, userID)

		// Assert
		assert.NoError(t, err)
		assert.Empty(t, tenants)
	})
}
