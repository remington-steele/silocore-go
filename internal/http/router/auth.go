package router

import (
	"context"
	"errors"
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
	log.Printf("[INFO] Initializing AuthRouter")
	return &AuthRouter{
		authService:         authService,
		registrationService: registrationService,
		jwtService:          jwtService,
	}
}

// LoginPage renders the login page
func (ar *AuthRouter) LoginPage(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DEBUG] Rendering login page: %s", r.URL.String())
	data := pages.LoginData{}

	// Check if there's a message in the query string
	if message := r.URL.Query().Get("message"); message != "" {
		// In a real app, you might want to validate/sanitize this message
		log.Printf("[DEBUG] Login page message: %s", message)
		data.Error = message
	}

	component := pages.Login(data)
	component.Render(r.Context(), w)
}

// HandleLogin processes login form submission
func (ar *AuthRouter) HandleLogin(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Processing login request from %s", r.RemoteAddr)

	if err := r.ParseForm(); err != nil {
		log.Printf("[WARN] Invalid login form submission: %v", err)
		data := pages.LoginData{Error: "Invalid form submission"}
		component := pages.Login(data)
		component.Render(r.Context(), w)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password") // Don't log passwords

	log.Printf("[DEBUG] Login attempt for email: %s", email)

	// Validate inputs
	if email == "" || password == "" {
		log.Printf("[WARN] Login attempt with empty email or password")
		data := pages.LoginData{Error: "Email and password are required"}
		component := pages.Login(data)
		component.Render(r.Context(), w)
		return
	}

	// Check if authentication services are available
	if ar.authService == nil || ar.jwtService == nil {
		log.Printf("[ERROR] Authentication service unavailable for login request")
		http.Error(w, "Authentication service unavailable", http.StatusInternalServerError)
		return
	}

	// Authenticate the user
	tokenPair, userID, err := ar.authService.Login(r.Context(), email, password)
	if err != nil {
		log.Printf("[WARN] Failed login attempt for user %s: %v", email, err)

		var errorMessage string
		if errors.Is(err, service.ErrInvalidCredentials) {
			errorMessage = "Invalid email or password"
		} else {
			errorMessage = "Authentication failed. Please try again."
		}

		data := pages.LoginData{Error: errorMessage}
		component := pages.Login(data)
		component.Render(r.Context(), w)
		return
	}

	tokenString := tokenPair.AccessToken
	log.Printf("[INFO] Successfully authenticated user: %s (ID: %d)", email, userID)

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
	log.Printf("[DEBUG] Set auth_token cookie for user %s, expires in 24 hours", email)

	// Redirect to orders page instead of home page
	log.Printf("[DEBUG] Redirecting authenticated user %s to /orders", email)
	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

// RegisterPage renders the registration page
func (ar *AuthRouter) RegisterPage(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DEBUG] Rendering registration page: %s", r.URL.String())
	data := pages.RegisterData{}
	component := pages.Register(data)
	component.Render(r.Context(), w)
}

// HandleRegister processes registration form submission
func (ar *AuthRouter) HandleRegister(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Processing registration request from %s", r.RemoteAddr)

	if err := r.ParseForm(); err != nil {
		log.Printf("[WARN] Invalid registration form submission: %v", err)
		data := pages.RegisterData{Error: "Invalid form submission"}
		component := pages.Register(data)
		component.Render(r.Context(), w)
		return
	}

	// Log all form values for debugging (except password fields)
	formValues := make(map[string][]string)
	for key, value := range r.Form {
		if !strings.Contains(strings.ToLower(key), "password") {
			formValues[key] = value
		} else {
			formValues[key] = []string{"[REDACTED]"}
		}
	}
	log.Printf("[DEBUG] Registration form values: %+v", formValues)

	firstName := strings.TrimSpace(r.FormValue("first_name"))
	lastName := strings.TrimSpace(r.FormValue("last_name"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")                // Don't log passwords
	confirmPassword := r.FormValue("confirm_password") // Don't log passwords

	// Log extracted values (except passwords)
	log.Printf("[DEBUG] Registration attempt - firstName: %s, lastName: %s, email: %s", firstName, lastName, email)

	// Validate inputs
	if firstName == "" || lastName == "" || email == "" || password == "" || confirmPassword == "" {
		log.Printf("[WARN] Registration attempt with missing required fields")
		data := pages.RegisterData{Error: "All fields are required"}
		component := pages.Register(data)
		component.Render(r.Context(), w)
		return
	}

	if len(password) < 8 {
		log.Printf("[WARN] Registration attempt with password too short for email: %s", email)
		data := pages.RegisterData{Error: "Password must be at least 8 characters"}
		component := pages.Register(data)
		component.Render(r.Context(), w)
		return
	}

	if password != confirmPassword {
		log.Printf("[WARN] Registration attempt with mismatched passwords for email: %s", email)
		data := pages.RegisterData{Error: "Passwords do not match"}
		component := pages.Register(data)
		component.Render(r.Context(), w)
		return
	}

	// Check if the auth service is available
	if ar.registrationService == nil {
		log.Printf("[ERROR] Registration service not available for registration request")
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
		log.Printf("[ERROR] Failed to register user %s: %v", email, err)
		data := pages.RegisterData{Error: "Failed to register user: " + err.Error()}
		component := pages.Register(data)
		component.Render(r.Context(), w)
		return
	}

	log.Printf("[INFO] Successfully registered new user: %s", email)

	// Redirect to login page with success message
	log.Printf("[DEBUG] Redirecting newly registered user %s to login page", email)
	http.Redirect(w, r, "/login?message=Registration+successful!+You+can+now+log+in.", http.StatusSeeOther)
}

// registerUser is a helper method to register a user
func (ar *AuthRouter) registerUser(ctx context.Context, firstName, lastName, email, password string) error {
	// Validate password
	if err := service.ValidatePassword(password); err != nil {
		log.Printf("[WARN] Password validation failed for email %s: %v", email, err)
		return err
	}

	log.Printf("[DEBUG] Attempting to register user with email: %s", email)

	// Register the user
	userID, err := ar.registrationService.RegisterUser(ctx, firstName, lastName, email, password)
	if err != nil {
		log.Printf("[ERROR] User registration failed for email %s: %v", email, err)
		return err
	}

	log.Printf("[INFO] User registered successfully with ID: %d, email: %s", userID, email)
	return nil
}

// HandleLogout processes logout requests
func (ar *AuthRouter) HandleLogout(w http.ResponseWriter, r *http.Request) {
	log.Printf("[INFO] Processing logout request from %s", r.RemoteAddr)

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

	log.Printf("[DEBUG] Cleared auth_token cookie for user")

	// Redirect to login page
	log.Printf("[DEBUG] Redirecting logged out user to login page")
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
