package router

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/unsavory/silocore-go/internal/auth/jwt"
	"github.com/unsavory/silocore-go/internal/auth/service"
	"github.com/unsavory/silocore-go/internal/views/pages"
)

// AuthRouter handles authentication-related routes
type AuthRouter struct {
	authService         service.AuthService
	registrationService service.RegistrationService
	jwtService          *jwt.Service
}

// NewAuthRouter creates a new AuthRouter with the required dependencies
func NewAuthRouter(authService service.AuthService, registrationService service.RegistrationService, jwtService *jwt.Service) *AuthRouter {
	return &AuthRouter{
		authService:         authService,
		registrationService: registrationService,
		jwtService:          jwtService,
	}
}

// LoginPage renders the login page
func (ar *AuthRouter) LoginPage(w http.ResponseWriter, r *http.Request) {
	data := pages.LoginData{}

	// Check if there's a message in the query string
	if message := r.URL.Query().Get("message"); message != "" {
		// In a real app, you might want to validate/sanitize this message
		data.Error = message
	}

	component := pages.Login(data)
	component.Render(r.Context(), w)
}

// HandleLogin processes login form submission
func (ar *AuthRouter) HandleLogin(w http.ResponseWriter, r *http.Request) {
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

	// Generate a JWT token using the JWT service
	if ar.jwtService == nil {
		http.Error(w, "Authentication service unavailable", http.StatusInternalServerError)
		return
	}

	// In a real implementation, you would get the user ID and roles from the database
	// For now, we're using placeholder values that should be replaced with actual user data
	userID := int64(1)
	username := "user@example.com" // This should come from the form or user record
	var tenantID *int64            // This would be set based on the user's default tenant

	// Generate token pair using the JWT service
	tokenPair, err := ar.jwtService.GenerateTokenPair(userID, username, tenantID)
	if err != nil {
		http.Error(w, "Failed to generate authentication token", http.StatusInternalServerError)
		return
	}

	tokenString := tokenPair.AccessToken

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

	// Redirect to orders page instead of home page
	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

// RegisterPage renders the registration page
func (ar *AuthRouter) RegisterPage(w http.ResponseWriter, r *http.Request) {
	data := pages.RegisterData{}
	component := pages.Register(data)
	component.Render(r.Context(), w)
}

// HandleRegister processes registration form submission
func (ar *AuthRouter) HandleRegister(w http.ResponseWriter, r *http.Request) {
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
	if ar.registrationService == nil {
		log.Printf("Error: Registration service not available")
		data := pages.RegisterData{Error: "Registration service unavailable"}
		component := pages.Register(data)
		component.Render(r.Context(), w)
		return
	}

	// Create user registration request
	ctx := r.Context()

	// Attempt to register the user
	err := ar.registerUser(ctx, firstName, lastName, email, password)
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
func (ar *AuthRouter) registerUser(ctx context.Context, firstName, lastName, email, password string) error {
	// Validate password
	if err := service.ValidatePassword(password); err != nil {
		return err
	}

	// Register the user
	_, err := ar.registrationService.RegisterUser(ctx, firstName, lastName, email, password)
	if err != nil {
		return err
	}

	return nil
}

// HandleLogout processes logout requests
func (ar *AuthRouter) HandleLogout(w http.ResponseWriter, r *http.Request) {
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
