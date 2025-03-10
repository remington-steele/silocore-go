package order

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	authctx "github.com/unsavory/silocore-go/internal/auth/context"
	orderservice "github.com/unsavory/silocore-go/internal/order/service"
)

// Handler handles HTTP requests for orders
type Handler struct {
	orderService orderservice.OrderService
}

// NewHandler creates a new order handler
func NewHandler(orderService orderservice.OrderService) *Handler {
	return &Handler{
		orderService: orderService,
	}
}

// GetOrder handles GET /orders/{id}
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	// Parse order ID from URL
	orderIDStr := chi.URLParam(r, "id")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	// Verify tenant context
	tenantID, err := authctx.GetTenantID(r.Context())
	if err != nil || tenantID == nil {
		http.Error(w, "Tenant context required", http.StatusForbidden)
		return
	}

	// Get order from service
	order, err := h.orderService.GetOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, orderservice.ErrOrderNotFound) {
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, orderservice.ErrNoTenantContext) {
			http.Error(w, "Tenant context required", http.StatusForbidden)
			return
		}
		log.Printf("Error getting order: %v", err)
		http.Error(w, "Failed to get order", http.StatusInternalServerError)
		return
	}

	// Verify order belongs to the tenant in context
	if order.TenantID != *tenantID {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Return order as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// ListOrders handles GET /orders
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	// Verify tenant context
	tenantID, err := authctx.GetTenantID(r.Context())
	if err != nil || tenantID == nil {
		http.Error(w, "Tenant context required", http.StatusForbidden)
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	userIDStr := r.URL.Query().Get("user_id")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	// Create filter
	filter := orderservice.OrderFilter{
		Status: status,
	}

	// Parse user ID if provided
	if userIDStr != "" {
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		filter.UserID = &userID
	}

	// Parse limit if provided
	if limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			http.Error(w, "Invalid limit", http.StatusBadRequest)
			return
		}
		filter.Limit = limit
	} else {
		// Default limit
		filter.Limit = 50
	}

	// Parse offset if provided
	if offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil {
			http.Error(w, "Invalid offset", http.StatusBadRequest)
			return
		}
		filter.Offset = offset
	}

	// Get orders from service
	orders, err := h.orderService.ListOrders(r.Context(), filter)
	if err != nil {
		if errors.Is(err, orderservice.ErrNoTenantContext) {
			http.Error(w, "Tenant context required", http.StatusForbidden)
			return
		}
		log.Printf("Error listing orders: %v", err)
		http.Error(w, "Failed to list orders", http.StatusInternalServerError)
		return
	}

	// Return orders as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

// ListUserOrders handles GET /users/{id}/orders
func (h *Handler) ListUserOrders(w http.ResponseWriter, r *http.Request) {
	// Verify tenant context
	tenantID, err := authctx.GetTenantID(r.Context())
	if err != nil || tenantID == nil {
		http.Error(w, "Tenant context required", http.StatusForbidden)
		return
	}

	// Parse user ID from URL
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Get orders from service
	orders, err := h.orderService.ListUserOrders(r.Context(), userID)
	if err != nil {
		if errors.Is(err, orderservice.ErrNoTenantContext) {
			http.Error(w, "Tenant context required", http.StatusForbidden)
			return
		}
		log.Printf("Error listing user orders: %v", err)
		http.Error(w, "Failed to list user orders", http.StatusInternalServerError)
		return
	}

	// Return orders as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

// CreateOrder handles POST /orders
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	// Verify tenant context
	tenantID, err := authctx.GetTenantID(r.Context())
	if err != nil || tenantID == nil {
		http.Error(w, "Tenant context required", http.StatusForbidden)
		return
	}

	// Parse request body
	var order orderservice.Order
	err = json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set tenant ID from context
	order.TenantID = *tenantID

	// Get user ID from context
	userID, err := authctx.GetUserID(r.Context())
	if err != nil {
		http.Error(w, "User ID not found in context", http.StatusUnauthorized)
		return
	}
	order.UserID = userID

	// Create order
	createdOrder, err := h.orderService.CreateOrder(r.Context(), &order)
	if err != nil {
		if errors.Is(err, orderservice.ErrInvalidInput) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, orderservice.ErrNoTenantContext) {
			http.Error(w, "Tenant context required", http.StatusForbidden)
			return
		}
		log.Printf("Error creating order: %v", err)
		http.Error(w, "Failed to create order", http.StatusInternalServerError)
		return
	}

	// Return created order as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createdOrder)
}

// UpdateOrder handles PUT /orders/{id}
func (h *Handler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	// Verify tenant context
	tenantID, err := authctx.GetTenantID(r.Context())
	if err != nil || tenantID == nil {
		http.Error(w, "Tenant context required", http.StatusForbidden)
		return
	}

	// Parse order ID from URL
	orderIDStr := chi.URLParam(r, "id")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	// Parse request body
	var order orderservice.Order
	err = json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set order ID and tenant ID
	order.ID = orderID
	order.TenantID = *tenantID

	// Update order
	err = h.orderService.UpdateOrder(r.Context(), &order)
	if err != nil {
		if errors.Is(err, orderservice.ErrOrderNotFound) {
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, orderservice.ErrInvalidInput) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, orderservice.ErrNoTenantContext) {
			http.Error(w, "Tenant context required", http.StatusForbidden)
			return
		}
		log.Printf("Error updating order: %v", err)
		http.Error(w, "Failed to update order", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusNoContent)
}

// DeleteOrder handles DELETE /orders/{id}
func (h *Handler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	// Verify tenant context
	tenantID, err := authctx.GetTenantID(r.Context())
	if err != nil || tenantID == nil {
		http.Error(w, "Tenant context required", http.StatusForbidden)
		return
	}

	// Parse order ID from URL
	orderIDStr := chi.URLParam(r, "id")
	orderID, err := strconv.ParseInt(orderIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	// Delete order
	err = h.orderService.DeleteOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, orderservice.ErrOrderNotFound) {
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}
		if errors.Is(err, orderservice.ErrNoTenantContext) {
			http.Error(w, "Tenant context required", http.StatusForbidden)
			return
		}
		log.Printf("Error deleting order: %v", err)
		http.Error(w, "Failed to delete order", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusNoContent)
}

// CountOrders handles GET /orders/count
func (h *Handler) CountOrders(w http.ResponseWriter, r *http.Request) {
	// Verify tenant context
	tenantID, err := authctx.GetTenantID(r.Context())
	if err != nil || tenantID == nil {
		http.Error(w, "Tenant context required", http.StatusForbidden)
		return
	}

	// Parse query parameters
	status := r.URL.Query().Get("status")
	userIDStr := r.URL.Query().Get("user_id")

	// Create filter
	filter := orderservice.OrderFilter{
		Status: status,
	}

	// Parse user ID if provided
	if userIDStr != "" {
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}
		filter.UserID = &userID
	}

	// Count orders
	count, err := h.orderService.CountOrders(r.Context(), filter)
	if err != nil {
		if errors.Is(err, orderservice.ErrNoTenantContext) {
			http.Error(w, "Tenant context required", http.StatusForbidden)
			return
		}
		log.Printf("Error counting orders: %v", err)
		http.Error(w, "Failed to count orders", http.StatusInternalServerError)
		return
	}

	// Return count as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"count": count})
}
