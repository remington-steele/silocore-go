package order

import (
	"github.com/go-chi/chi/v5"
	"github.com/unsavory/silocore-go/internal/http/middleware"
	"github.com/unsavory/silocore-go/internal/service"
)

// RegisterRoutes registers order routes
func RegisterRoutes(r chi.Router, factory *service.Factory) {
	// Create handler
	handler := NewHandler(factory.OrderService())

	// Register routes
	r.Route("/orders", func(r chi.Router) {
		// Apply middleware
		r.Use(middleware.AuthMiddleware(factory.JWTService()))
		r.Use(middleware.RoleMiddleware(factory.UserService()))
		r.Use(middleware.RequireTenantContext)

		// GET /orders
		r.Get("/", handler.ListOrders)

		// GET /orders/count
		r.Get("/count", handler.CountOrders)

		// POST /orders
		r.Post("/", handler.CreateOrder)

		// GET /orders/{id}
		r.Get("/{id}", handler.GetOrder)

		// PUT /orders/{id}
		r.Put("/{id}", handler.UpdateOrder)

		// DELETE /orders/{id}
		r.Delete("/{id}", handler.DeleteOrder)
	})

	// Register user orders route
	r.Route("/users/{id}/orders", func(r chi.Router) {
		// Apply middleware
		r.Use(middleware.AuthMiddleware(factory.JWTService()))
		r.Use(middleware.RoleMiddleware(factory.UserService()))
		r.Use(middleware.RequireTenantContext)

		// GET /users/{id}/orders
		r.Get("/", handler.ListUserOrders)
	})
}
