package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

// MigrateOptions contains options for running migrations
type MigrateOptions struct {
	// DatabaseURL is the connection string for the database
	DatabaseURL string
	// MigrationsPath is the path to the migrations directory
	MigrationsPath string
	// MigrateUp indicates whether to migrate up or down
	MigrateUp bool
	// Steps is the number of migrations to apply (0 means all)
	Steps int
}

// RunMigrationsUp is a convenience function to run all migrations up
// It uses the provided database URL and migrations path
func RunMigrationsUp(dbURL, migrationsPath string) error {
	opts := MigrateOptions{
		DatabaseURL:    dbURL,
		MigrationsPath: migrationsPath,
		MigrateUp:      true,
		Steps:          0, // Run all pending migrations
	}
	return RunMigrations(opts)
}

// RunMigrationsDown is a convenience function to run all migrations down
// It uses the provided database URL and migrations path
func RunMigrationsDown(dbURL, migrationsPath string) error {
	opts := MigrateOptions{
		DatabaseURL:    dbURL,
		MigrationsPath: migrationsPath,
		MigrateUp:      false,
		Steps:          0, // Run all pending migrations
	}
	return RunMigrations(opts)
}

// RunMigrations runs database migrations based on the provided options
func RunMigrations(opts MigrateOptions) error {
	log.Printf("Running migrations with options: path=%s, up=%t, steps=%d",
		opts.MigrationsPath, opts.MigrateUp, opts.Steps)

	// Connect to the database
	db, err := sql.Open("postgres", opts.DatabaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Ping the database to ensure the connection is valid
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Get the absolute path to the migrations directory
	absPath, err := filepath.Abs(opts.MigrationsPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for migrations: %w", err)
	}

	// Check if the migrations directory exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory does not exist: %s", absPath)
	}

	// Create a new postgres driver instance
	driver, err := postgres.WithInstance(db, &postgres.Config{
		MigrationsTable: "_migration",
		// Set the search path to public schema
		SchemaName: "public",
	})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver instance: %w", err)
	}

	// Create a new migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file:///%s", absPath),
		"postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Set up a function to log migration version
	logVersion := func() {
		version, dirty, err := m.Version()
		if err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				log.Println("No migration has been applied yet")
				return
			}
			log.Printf("Failed to get migration version: %v", err)
			return
		}
		log.Printf("Current migration version: %d, dirty: %t", version, dirty)
	}

	// Log the current version before migration
	logVersion()

	// Count and log the number of migrations to be applied
	files, err := os.ReadDir(absPath)
	if err != nil {
		log.Printf("Warning: Failed to read migrations directory: %v", err)
	} else {
		var migrationFiles []string
		for _, file := range files {
			if !file.IsDir() {
				migrationFiles = append(migrationFiles, file.Name())
				log.Printf("Found migration file: %s", file.Name())
			}
		}
		log.Printf("Found %d migration files", len(migrationFiles))
	}

	// Start time for measuring migration duration
	startTime := time.Now()

	// Run the migration
	var migrationErr error
	if opts.MigrateUp {
		if opts.Steps > 0 {
			log.Printf("Running %d migrations up", opts.Steps)
			migrationErr = m.Steps(opts.Steps)
		} else {
			log.Println("Running all pending migrations up")
			migrationErr = m.Up()
		}
	} else {
		if opts.Steps > 0 {
			log.Printf("Running %d migrations down", opts.Steps)
			migrationErr = m.Steps(-opts.Steps)
		} else {
			log.Println("Running all migrations down")
			migrationErr = m.Down()
		}
	}

	// Check for migration errors
	if migrationErr != nil {
		if errors.Is(migrationErr, migrate.ErrNoChange) {
			log.Println("No migration needed, database is up to date")
		} else {
			return fmt.Errorf("migration failed: %w", migrationErr)
		}
	}

	// Log the duration and new version
	duration := time.Since(startTime)
	log.Printf("Migration completed in %v", duration)
	logVersion()

	return nil
}
