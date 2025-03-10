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
)

// RouterDependencies contains all dependencies needed for the router
type RouterDependencies struct {
	Factory             *service.Factory
	JWTService          custommw.JWTService
	UserService         custommw.UserService
	AuthService         authservice.AuthService
	OrderService        orderservice.OrderService
	RegistrationService authservice.RegistrationService
	JWTAuthService      *jwt.Service
}

// RegisterRoutesWithAuth registers all application routes with authentication
func RegisterRoutesWithAuth(r chi.Router, deps RouterDependencies) {
	// Create a new router to apply middleware
	router := chi.NewRouter()

	// Apply transaction middleware to all routes if factory is available
	if deps.Factory != nil {
		router.Use(deps.Factory.TransactionManager().Middleware())
	}

	// Register view routes
	registerViewRoutes(router, deps)

	// Register API routes
	router.Route("/api", func(r chi.Router) {
		// Public routes
		registerPublicRoutes(r)

		// Protected routes (require authentication)
		r.Group(func(r chi.Router) {
			// Apply authentication middleware to all routes in this group
			r.Use(custommw.AuthMiddleware(deps.JWTService))

			// Apply role middleware to fetch and set user roles
			r.Use(custommw.RoleMiddleware(deps.UserService))

			// Admin routes
			registerAdminRoutes(r)

			// Tenant routes
			registerTenantRoutes(r, deps.UserService)

			// Order routes
			if deps.Factory != nil {
				order.RegisterRoutes(r, deps.Factory)
			}
		})
	})

	// Serve static files
	fileServer := http.FileServer(http.Dir("./internal/static"))
	router.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Mount the router with middleware to the parent router
	r.Mount("/", router)
}

// RegisterRoutes registers all application routes without authentication (for development/testing)
func RegisterRoutes(r chi.Router) {
	// Register view routes (without authentication for development/testing)
	registerViewRoutes(r, RouterDependencies{})

	// Serve static files
	fileServer := http.FileServer(http.Dir("./internal/static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Register API routes
	r.Route("/api", func(r chi.Router) {
		// Public routes
		registerPublicRoutes(r)

		// Protected routes (without authentication for development/testing)
		r.Group(func(r chi.Router) {
			// Admin routes
			registerAdminRoutes(r)

			// Tenant routes
			registerTenantRoutes(r, nil)

			// Order routes (without factory for development/testing)
			// Note: This won't work without a factory, but is included for completeness
		})
	})
}

// registerViewRoutes registers all view routes
func registerViewRoutes(r chi.Router, deps RouterDependencies) {
	// Create a views router with available services
	// If services are not available, use nil to allow development mode
	viewsRouter := NewViewsRouter(deps.AuthService, deps.OrderService, deps.RegistrationService, deps.JWTAuthService)

	// Mount the views router
	r.Mount("/", viewsRouter.Routes())
}

// registerPublicRoutes registers routes that don't require authentication
func registerPublicRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Welcome to SiloCore API"))
		})

		// Auth routes for login, registration, etc.
		r.Post("/login", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Login endpoint"))
		})

		r.Post("/register", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Register endpoint"))
		})
	})
}

// registerAdminRoutes registers routes that require ADMIN role
func registerAdminRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		// Apply admin middleware to all routes in this group
		r.Use(custommw.RequireAdmin)

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Admin Dashboard"))
		})

		// Tenant management
		r.Route("/tenants", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("List of all tenants"))
			})

			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Create new tenant"))
			})

			r.Route("/{tenantID}", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Get tenant details"))
				})

				r.Put("/", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Update tenant"))
				})

				r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Delete tenant"))
				})
			})
		})

		// User management
		r.Route("/users", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("List of all users"))
			})

			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Create new user"))
			})

			r.Route("/{userID}", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Get user details"))
				})

				r.Put("/", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Update user"))
				})

				r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Delete user"))
				})
			})
		})
	})
}

// registerTenantRoutes registers routes that require tenant context
func registerTenantRoutes(r chi.Router, userService custommw.UserService) {
	r.Route("/tenant", func(r chi.Router) {
		// Apply tenant context middleware to all routes in this group
		r.Use(custommw.RequireTenantContext)

		// If userService is provided (in auth mode), require tenant membership
		if userService != nil {
			r.Use(custommw.RequireTenantMember(userService))
		}

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Tenant Dashboard"))
		})

		// Tenant profile
		r.Route("/profile", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Get tenant profile"))
			})

			r.Put("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Update tenant profile"))
			})
		})

		// Tenant members
		r.Route("/members", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("List tenant members"))
			})

			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("Add tenant member"))
			})

			// Tenant super routes
			r.Route("/admin", func(r chi.Router) {
				// Apply tenant super middleware
				r.Use(custommw.RequireTenantSuper)

				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Tenant Admin Dashboard"))
				})
			})

			r.Route("/{memberID}", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Get member details"))
				})

				r.Put("/", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Update member"))
				})

				r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("Remove member"))
				})
			})
		})
	})
}
