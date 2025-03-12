package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/unsavory/silocore-go/internal/auth/jwt"
	"github.com/unsavory/silocore-go/internal/database"
	"github.com/unsavory/silocore-go/internal/http/router"
	orderservice "github.com/unsavory/silocore-go/internal/order/service"
	appservice "github.com/unsavory/silocore-go/internal/service"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Run database migrations at startup using the admin connection string
	adminDbUrl := os.Getenv("DATABASE_ADMIN_URL")
	if adminDbUrl == "" {
		log.Fatal("DATABASE_ADMIN_URL environment variable is required for migrations")
	}

	// Set up migration options
	opts := database.MigrateOptions{
		DatabaseURL:    adminDbUrl,
		MigrationsPath: "sql/migrations",
		MigrateUp:      true,
		Steps:          0, // Run all pending migrations
	}

	// Run migrations
	if err := database.RunMigrations(opts); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations completed successfully")

	// Initialize database connection
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize JWT service
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	jwtConfig := jwt.Config{
		Secret:            jwtSecret,
		AccessExpiration:  3600,          // 1 hour
		RefreshExpiration: 7 * 24 * 3600, // 7 days
		Issuer:            "silocore-go",
	}

	// Create service factory
	serviceFactory := appservice.NewFactory(db, jwtConfig)

	// Initialize user service from factory
	userService := serviceFactory.UserService()

	// Initialize JWT service from factory
	jwtService := serviceFactory.JWTService()

	// Initialize auth service from factory
	authService := serviceFactory.AuthService()

	// Initialize order service
	orderService := orderservice.NewDBOrderService(db)

	// Initialize registration service
	registrationService := serviceFactory.RegistrationService()

	// Initialize tenant member service
	tenantMemberService := serviceFactory.TenantMemberService()

	// Create router dependencies
	routerDeps := router.RouterDependencies{
		Factory:             serviceFactory,
		JWTService:          jwtService,
		UserService:         userService,
		AuthService:         authService,
		OrderService:        orderService,
		RegistrationService: registrationService,
		JWTAuthService:      jwtService,
		TenantMemberService: tenantMemberService,
	}

	// Initialize Chi router with default options and dependencies
	routerOpts := router.DefaultOptions()
	routerOpts.Dependencies = routerDeps
	r := router.New(routerOpts)

	// Register application routes
	router.RegisterRoutes(r, routerDeps)

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
