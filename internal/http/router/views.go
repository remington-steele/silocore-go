package router

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	authctx "github.com/unsavory/silocore-go/internal/auth/context"
	"github.com/unsavory/silocore-go/internal/auth/jwt"
	"github.com/unsavory/silocore-go/internal/auth/service"
	"github.com/unsavory/silocore-go/internal/order"
	orderService "github.com/unsavory/silocore-go/internal/order/service"
	"github.com/unsavory/silocore-go/internal/views/pages"
)

// ViewsRouter handles all view-related routes
type ViewsRouter struct {
	authService         service.AuthService
	orderService        orderService.OrderService
	registrationService service.RegistrationService
	jwtService          *jwt.Service
}

// NewViewsRouter creates a new ViewsRouter
func NewViewsRouter(authService service.AuthService, orderService orderService.OrderService, registrationService service.RegistrationService, jwtService *jwt.Service) *ViewsRouter {
	return &ViewsRouter{
		authService:         authService,
		orderService:        orderService,
		registrationService: registrationService,
		jwtService:          jwtService,
	}
}

// Routes returns all view routes
func (vr *ViewsRouter) Routes() chi.Router {
	r := chi.NewRouter()

	// Public routes
	r.Group(func(r chi.Router) {
		r.Get("/login", vr.LoginPage)
		r.Post("/login", vr.HandleLogin)
		r.Get("/register", vr.RegisterPage)
		r.Post("/register", vr.HandleRegister)
	})

	// Protected routes
	r.Group(func(r chi.Router) {
		// Add authentication middleware
		r.Use(vr.authMiddleware)

		r.Get("/", vr.HomePage)
		r.Get("/orders", vr.OrdersPage)
		r.Get("/orders/{id}", vr.OrderDetailPage)
		r.Post("/logout", vr.HandleLogout)
	})

	return r
}

// LoginPage renders the login page
func (vr *ViewsRouter) LoginPage(w http.ResponseWriter, r *http.Request) {
	data := pages.LoginData{}

	// Check if there's a message in the query string
	if message := r.URL.Query().Get("message"); message != "" {
		// In a real app, you might want to validate/sanitize this message
		data.Error = message
	}

	component := pages.Login(data)
	component.Render(r.Context(), w)
}

// RegisterPage renders the registration page
func (vr *ViewsRouter) RegisterPage(w http.ResponseWriter, r *http.Request) {
	data := pages.RegisterData{}
	component := pages.Register(data)
	component.Render(r.Context(), w)
}

// HandleRegister processes registration form submission
func (vr *ViewsRouter) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		data := pages.RegisterData{Error: "Invalid form submission"}
		component := pages.Register(data)
		component.Render(r.Context(), w)
		return
	}

	// Log all form values for debugging
	log.Printf("Form values: %+v", r.Form)

	firstName := strings.TrimSpace(r.FormValue("first_name"))
	lastName := strings.TrimSpace(r.FormValue("last_name"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	// Log extracted values
	log.Printf("Extracted values - firstName: %s, lastName: %s, email: %s", firstName, lastName, email)

	// Validate inputs
	if firstName == "" || lastName == "" || email == "" || password == "" || confirmPassword == "" {
		data := pages.RegisterData{Error: "All fields are required"}
		component := pages.Register(data)
		component.Render(r.Context(), w)
		return
	}

	if len(password) < 8 {
		data := pages.RegisterData{Error: "Password must be at least 8 characters"}
		component := pages.Register(data)
		component.Render(r.Context(), w)
		return
	}

	if password != confirmPassword {
		data := pages.RegisterData{Error: "Passwords do not match"}
		component := pages.Register(data)
		component.Render(r.Context(), w)
		return
	}

	// Check if the auth service is available
	if vr.registrationService == nil {
		log.Printf("Error: Registration service not available")
		data := pages.RegisterData{Error: "Registration service unavailable"}
		component := pages.Register(data)
		component.Render(r.Context(), w)
		return
	}

	// Create user registration request
	ctx := r.Context()

	// Attempt to register the user
	err := vr.registerUser(ctx, firstName, lastName, email, password)
	if err != nil {
		log.Printf("Error registering user: %v", err)
		data := pages.RegisterData{Error: "Failed to register user: " + err.Error()}
		component := pages.Register(data)
		component.Render(r.Context(), w)
		return
	}

	// Redirect to login page with success message
	http.Redirect(w, r, "/login?message=Registration+successful!+You+can+now+log+in.", http.StatusSeeOther)
}

// registerUser is a helper method to register a user
func (vr *ViewsRouter) registerUser(ctx context.Context, firstName, lastName, email, password string) error {
	// Validate password
	if err := service.ValidatePassword(password); err != nil {
		return err
	}

	// Register the user
	_, err := vr.registrationService.RegisterUser(ctx, firstName, lastName, email, password)
	if err != nil {
		return err
	}

	return nil
}

// HandleLogin processes login form submission
func (vr *ViewsRouter) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		data := pages.LoginData{Error: "Invalid form submission"}
		component := pages.Login(data)
		component.Render(r.Context(), w)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Validate inputs
	if email == "" || password == "" {
		data := pages.LoginData{Error: "Email and password are required"}
		component := pages.Login(data)
		component.Render(r.Context(), w)
		return
	}

	// In a real implementation, you would:
	// 1. Query the database to find the user by email
	// 2. Verify the password hash
	// 3. Generate a JWT token with the user's ID and roles

	// For now, we'll use a sample token for development
	// In production, this would be a real JWT token generated by the JWT service
	tokenString := "sample_token"

	// Set the token as a cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	// Redirect to orders page after successful login
	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

// HomePage renders the home page
func (vr *ViewsRouter) HomePage(w http.ResponseWriter, r *http.Request) {
	// Redirect to orders page for now
	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

// OrdersPage renders the orders page
func (vr *ViewsRouter) OrdersPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get user ID from context
	userID, err := authctx.GetUserID(ctx)
	if err != nil {
		log.Printf("Error: User ID not found in context: %v", err)

		// Redirect to login page
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get tenant ID from context
	tenantID, err := authctx.GetTenantID(ctx)
	if err != nil || tenantID == nil {
		log.Printf("Error: Tenant ID not found in context: %v", err)

		// Redirect to login page
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get username from context
	username, err := authctx.GetUsername(ctx)
	if err != nil {
		log.Printf("Warning: Username not found in context: %v", err)
		// Continue with empty username rather than failing
		username = ""
	}

	// Create filter for orders
	filter := orderService.OrderFilter{
		UserID: &userID,
		Limit:  50,
		Offset: 0,
	}

	// Get orders from service
	serviceOrders, err := vr.orderService.ListOrders(ctx, filter)
	if err != nil {
		log.Printf("Error fetching orders: %v", err)

		// For development purposes, use empty orders list instead of failing
		serviceOrders = []orderService.Order{}
	}

	// Convert service orders to view model orders
	viewOrders := make([]order.Order, len(serviceOrders))
	for i, o := range serviceOrders {
		viewOrders[i] = order.Order{
			ID:        fmt.Sprintf("ORD-%d", o.ID),
			TenantID:  strconv.FormatInt(o.TenantID, 10),
			UserID:    strconv.FormatInt(o.UserID, 10),
			Status:    o.Status,
			Total:     o.TotalAmount,
			CreatedAt: o.CreatedAt,
			UpdatedAt: o.UpdatedAt,
		}
	}

	data := pages.OrdersPageData{
		Orders: viewOrders,
		User: struct {
			Name string
		}{
			Name: username,
		},
	}

	component := pages.Orders(data)
	if err := component.Render(ctx, w); err != nil {
		log.Printf("Error rendering orders page: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// OrderDetailPage renders the order detail page
func (vr *ViewsRouter) OrderDetailPage(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")

	// Convert string ID to int64, removing any prefix
	numericID := orderID
	if len(orderID) > 4 && orderID[:4] == "ORD-" {
		numericID = orderID[4:]
	}

	id, err := strconv.ParseInt(numericID, 10, 64)
	if err != nil {
		log.Printf("Invalid order ID: %v", orderID)
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Get tenant ID from context
	tenantID, err := authctx.GetTenantID(ctx)
	if err != nil || tenantID == nil {
		log.Printf("Error: Tenant ID not found in context: %v", err)
		http.Error(w, "Tenant context required", http.StatusBadRequest)
		return
	}

	// Get order from service
	serviceOrder, err := vr.orderService.GetOrder(ctx, id)
	if err != nil {
		log.Printf("Error fetching order: %v", err)
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// TODO: Implement order detail page rendering
	// For now, just return a simple message
	w.Write([]byte(fmt.Sprintf("Order details for %s (ID: %d)", serviceOrder.OrderNumber, serviceOrder.ID)))
}

// HandleLogout processes logout requests
func (vr *ViewsRouter) HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Clear the auth cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	// Redirect to login page
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// authMiddleware is a simple middleware to check for authentication
// In a real application, this would verify the JWT token
func (vr *ViewsRouter) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the auth token from the cookie
		cookie, err := r.Cookie("auth_token")
		if err != nil || cookie.Value == "" {
			// No auth token, redirect to login
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// In a real implementation, you would:
		// 1. Validate the JWT token
		// 2. Extract the user ID, tenant ID, and roles
		// 3. Set them in the request context

		// For development purposes, we'll use sample values
		// This is a placeholder - replace with actual JWT validation
		userID := int64(1)   // Sample user ID
		tenantID := int64(1) // Sample tenant ID
		username := "Sample User"

		// Create a new context with authentication information
		ctx := r.Context()
		ctx = authctx.WithUserID(ctx, userID)
		ctx = authctx.WithTenantID(ctx, &tenantID)
		ctx = authctx.WithUsername(ctx, username)
		ctx = authctx.WithRoles(ctx, []authctx.Role{authctx.RoleAdmin}) // Sample role

		// Create a new request with the updated context
		r = r.WithContext(ctx)

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
