package order

import (
	"github.com/go-chi/chi/v5"
	"github.com/unsavory/silocore-go/internal/http/middleware"
	orderservice "github.com/unsavory/silocore-go/internal/order/service"
	"github.com/unsavory/silocore-go/internal/service"
)

// OrderRouter handles order-related routes
type OrderRouter struct {
	handler *Handler
}

// NewOrderRouter creates a new OrderRouter with the required dependencies
func NewOrderRouter(orderService orderservice.OrderService) *OrderRouter {
	return &OrderRouter{
		handler: NewHandler(orderService),
	}
}

// RegisterRoutes registers order routes
func RegisterRoutes(r chi.Router, factory *service.Factory) {
	// Create order router with only the dependencies it needs
	orderRouter := NewOrderRouter(factory.OrderService())

	// Register routes
	r.Route("/orders", func(r chi.Router) {
		// Apply middleware - these should already be applied at a higher level
		// in the router hierarchy, but we include them here for completeness
		// and to ensure proper security even if the parent router changes
		r.Use(middleware.AuthMiddleware(factory.JWTService()))
		r.Use(middleware.RoleMiddleware(factory.UserService()))
		r.Use(middleware.RequireTenantContext)

		// GET /orders - View page
		r.Get("/", orderRouter.handler.OrdersPage)

		// API routes
		r.Route("/api", func(r chi.Router) {
			// GET /orders/api
			r.Get("/", orderRouter.handler.ListOrders)

			// GET /orders/api/count
			r.Get("/count", orderRouter.handler.CountOrders)

			// POST /orders/api
			r.Post("/", orderRouter.handler.CreateOrder)

			// GET /orders/api/{id}
			r.Get("/{id}", orderRouter.handler.GetOrder)

			// PUT /orders/api/{id}
			r.Put("/{id}", orderRouter.handler.UpdateOrder)

			// DELETE /orders/api/{id}
			r.Delete("/{id}", orderRouter.handler.DeleteOrder)
		})
	})

	// Register user orders route
	r.Route("/users/{id}/orders", func(r chi.Router) {
		// Apply middleware
		r.Use(middleware.AuthMiddleware(factory.JWTService()))
		r.Use(middleware.RoleMiddleware(factory.UserService()))
		r.Use(middleware.RequireTenantContext)

		// GET /users/{id}/orders
		r.Get("/", orderRouter.handler.ListUserOrders)
	})
}
