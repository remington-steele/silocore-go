package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/unsavory/silocore-go/internal/auth/jwt"
	authservice "github.com/unsavory/silocore-go/internal/auth/service"
	custommw "github.com/unsavory/silocore-go/internal/http/middleware"
	"github.com/unsavory/silocore-go/internal/http/router/order"
	orderservice "github.com/unsavory/silocore-go/internal/order/service"
	"github.com/unsavory/silocore-go/internal/service"
	tenantservice "github.com/unsavory/silocore-go/internal/tenant/service"
)

// RouterDependencies contains all dependencies needed for the router
type RouterDependencies struct {
	Factory             *service.Factory
	JWTService          custommw.JWTService
	UserService         authservice.UserService
	AuthService         authservice.AuthService
	OrderService        orderservice.OrderService
	RegistrationService authservice.RegistrationService
	JWTAuthService      *jwt.Service
	TenantMemberService tenantservice.TenantMemberService
}

// RegisterRoutes registers all application routes with proper authentication and authorization
func RegisterRoutes(r chi.Router, deps RouterDependencies) {
	// Create a new router to apply middleware
	router := chi.NewRouter()

	// Apply transaction middleware to all routes if factory is available
	if deps.Factory != nil {
		router.Use(deps.Factory.TransactionManager().Middleware())
	}

	// Register public routes (no authentication required)
	registerPublicRoutes(router, deps)

	// Register protected routes (require authentication)
	router.Group(func(r chi.Router) {
		// Apply authentication middleware to all routes in this group
		r.Use(custommw.AuthMiddleware(deps.JWTService))

		// Apply role middleware to fetch and set user roles
		r.Use(custommw.RoleMiddleware(deps.UserService, deps.TenantMemberService))

		// Admin routes
		registerAdminRoutes(r)

		// Tenant routes
		registerTenantRoutes(r, deps.UserService, deps.TenantMemberService)

		// Order routes
		if deps.Factory != nil {
			order.RegisterRoutes(r, deps.Factory)
		}
	})

	// Mount the router
	r.Mount("/", router)
}

// registerPublicRoutes registers routes that don't require authentication
func registerPublicRoutes(r chi.Router, deps RouterDependencies) {
	// Home page
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		// This could be a templ template rendering the home page
		w.Write([]byte("Welcome to SiloCore"))
	})

	// Authentication routes
	if deps.AuthService != nil && deps.JWTAuthService != nil {
		// Create auth router with only the dependencies it needs
		authRouter := NewAuthRouter(deps.AuthService, deps.RegistrationService, deps.JWTAuthService)

		// Mount auth routes
		r.Get("/login", authRouter.LoginPage)
		r.Post("/login", authRouter.HandleLogin)
		r.Get("/register", authRouter.RegisterPage)
		r.Post("/register", authRouter.HandleRegister)
		r.Get("/logout", authRouter.HandleLogout)
	} else {
		// Fallback for when services aren't available
		r.Get("/login", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Login Page"))
		})
		r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Login Handler"))
		})
		r.Get("/register", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Register Page"))
		})
		r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Register Handler"))
		})
	}

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
}

// registerAdminRoutes registers routes that require ADMIN role
func registerAdminRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		// Apply admin middleware to all routes in this group
		r.Use(custommw.RequireAdmin)

		// Create admin router with only the dependencies it needs
		adminRouter := NewAdminRouter()

		// Dashboard
		r.Get("/", adminRouter.Dashboard)

		// Tenant management
		r.Route("/tenants", func(r chi.Router) {
			r.Get("/", adminRouter.ListTenants)
			r.Post("/", adminRouter.CreateTenant)

			r.Route("/{tenantID}", func(r chi.Router) {
				r.Get("/", adminRouter.GetTenant)
				r.Put("/", adminRouter.UpdateTenant)
				r.Delete("/", adminRouter.DeleteTenant)
			})
		})

		// User management
		r.Route("/users", func(r chi.Router) {
			r.Get("/", adminRouter.ListUsers)
			r.Post("/", adminRouter.CreateUser)

			r.Route("/{userID}", func(r chi.Router) {
				r.Get("/", adminRouter.GetUser)
				r.Put("/", adminRouter.UpdateUser)
				r.Delete("/", adminRouter.DeleteUser)
			})
		})
	})
}

// registerTenantRoutes registers routes that require tenant context
func registerTenantRoutes(r chi.Router, userService authservice.UserService, tenantMemberService tenantservice.TenantMemberService) {
	r.Route("/tenant", func(r chi.Router) {
		// Apply tenant context middleware to all routes in this group
		r.Use(custommw.RequireTenantContext)

		// If tenantMemberService is provided, require tenant membership
		if tenantMemberService != nil {
			r.Use(custommw.RequireTenantMember(tenantMemberService))
		}

		// Create tenant router with only the dependencies it needs
		tenantRouter := NewTenantRouter(userService)

		// Dashboard
		r.Get("/", tenantRouter.Dashboard)

		// Tenant profile
		r.Route("/profile", func(r chi.Router) {
			r.Get("/", tenantRouter.GetProfile)
			r.Put("/", tenantRouter.UpdateProfile)
		})

		// Tenant members
		r.Route("/members", func(r chi.Router) {
			r.Get("/", tenantRouter.ListMembers)
			r.Post("/", tenantRouter.AddMember)

			// Tenant super routes
			r.Route("/admin", func(r chi.Router) {
				// Apply tenant super middleware
				r.Use(custommw.RequireTenantSuper)

				r.Get("/", tenantRouter.AdminDashboard)
			})

			r.Route("/{memberID}", func(r chi.Router) {
				r.Get("/", tenantRouter.GetMember)
				r.Put("/", tenantRouter.UpdateMember)
				r.Delete("/", tenantRouter.RemoveMember)
			})
		})
	})
}
