package service

import (
	"database/sql"

	"github.com/unsavory/silocore-go/internal/auth/jwt"
	authservice "github.com/unsavory/silocore-go/internal/auth/service"
	"github.com/unsavory/silocore-go/internal/database/transaction"
	orderservice "github.com/unsavory/silocore-go/internal/order/service"
	tenantservice "github.com/unsavory/silocore-go/internal/tenant/service"
)

// Factory provides access to all services
type Factory struct {
	db *sql.DB

	// Transaction manager
	txManager *transaction.Manager

	// Auth services
	userService         authservice.UserService
	authService         authservice.AuthService
	roleService         authservice.RoleService
	registrationService authservice.RegistrationService
	jwtService          *jwt.Service

	// Tenant services
	tenantService       tenantservice.TenantService
	tenantMemberService tenantservice.TenantMemberService

	// Order services
	orderService orderservice.OrderService
}

// NewFactory creates a new service factory
func NewFactory(db *sql.DB, jwtConfig jwt.Config) *Factory {
	// Create transaction manager
	txManager := transaction.NewManager(db)

	// Create JWT service
	jwtService := jwt.NewService(jwtConfig)

	// Create user service
	userService := authservice.NewDBUserService(db)

	// Create role service
	roleService := authservice.NewDBRoleService(db)

	// Create registration service
	registrationService := authservice.NewDBRegistrationService(db)

	// Create tenant service
	tenantService := tenantservice.NewDBTenantService(db)

	// Create tenant member service
	tenantMemberService := tenantservice.NewDBTenantMemberService(db)

	// Create auth service
	authService := authservice.NewDefaultAuthService(userService, tenantMemberService, jwtService)

	// Create order service
	orderService := orderservice.NewDBOrderService(db)

	return &Factory{
		db:                  db,
		txManager:           txManager,
		userService:         userService,
		authService:         authService,
		roleService:         roleService,
		registrationService: registrationService,
		jwtService:          jwtService,
		tenantService:       tenantService,
		tenantMemberService: tenantMemberService,
		orderService:        orderService,
	}
}

// UserService returns the user service
func (f *Factory) UserService() authservice.UserService {
	return f.userService
}

// AuthService returns the auth service
func (f *Factory) AuthService() authservice.AuthService {
	return f.authService
}

// RoleService returns the role service
func (f *Factory) RoleService() authservice.RoleService {
	return f.roleService
}

// RegistrationService returns the registration service
func (f *Factory) RegistrationService() authservice.RegistrationService {
	return f.registrationService
}

// JWTService returns the JWT service
func (f *Factory) JWTService() *jwt.Service {
	return f.jwtService
}

// TenantService returns the tenant service
func (f *Factory) TenantService() tenantservice.TenantService {
	return f.tenantService
}

// TenantMemberService returns the tenant member service
func (f *Factory) TenantMemberService() tenantservice.TenantMemberService {
	return f.tenantMemberService
}

// OrderService returns the order service
func (f *Factory) OrderService() orderservice.OrderService {
	return f.orderService
}

// TransactionManager returns the transaction manager
func (f *Factory) TransactionManager() *transaction.Manager {
	return f.txManager
}

// DB returns the database connection
func (f *Factory) DB() *sql.DB {
	return f.db
}
