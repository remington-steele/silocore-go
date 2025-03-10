package main

import (
	"flag"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/unsavory/silocore-go/internal/database"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Define command-line flags
	migrationsPath := flag.String("path", "sql/migrations", "Path to migration files")
	down := flag.Bool("down", false, "Migrate down instead of up")
	steps := flag.Int("steps", 0, "Number of migrations to apply (0 means all)")
	flag.Parse()

	// Get database URL from environment variables
	dbURL := os.Getenv("DATABASE_ADMIN_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_ADMIN_URL environment variable is required")
	}

	// Set up migration options
	opts := database.MigrateOptions{
		DatabaseURL:    dbURL,
		MigrationsPath: *migrationsPath,
		MigrateUp:      !*down,
		Steps:          *steps,
	}

	// Run migrations
	if err := database.RunMigrations(opts); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration completed successfully")
}
