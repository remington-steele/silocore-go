package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/scrypt"
)

// Registration errors
var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrRegistrationFailed = errors.New("registration failed")
)

// RegistrationService defines the interface for user registration
type RegistrationService interface {
	// RegisterUser registers a new user
	RegisterUser(ctx context.Context, firstName, lastName, email, password string) (int64, error)
}

// DBRegistrationService implements RegistrationService using a database
type DBRegistrationService struct {
	db *sql.DB
}

// NewDBRegistrationService creates a new DBRegistrationService
func NewDBRegistrationService(db *sql.DB) *DBRegistrationService {
	return &DBRegistrationService{db: db}
}

// RegisterUser registers a new user
func (s *DBRegistrationService) RegisterUser(ctx context.Context, firstName, lastName, email, password string) (int64, error) {
	// Check if email already exists
	var exists bool
	err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM usr WHERE email = $1)", email).Scan(&exists)
	if err != nil {
		log.Printf("Error checking if email exists: %v", err)
		return 0, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	if exists {
		return 0, ErrEmailAlreadyExists
	}

	// Validate password
	if err := ValidatePassword(password); err != nil {
		return 0, err
	}

	// Generate a random salt
	salt := make([]byte, SaltSize)
	_, err = rand.Read(salt)
	if err != nil {
		log.Printf("Error generating salt: %v", err)
		return 0, fmt.Errorf("%w: %v", ErrRegistrationFailed, err)
	}

	// Hash the password using scrypt
	hashedPassword, err := scrypt.Key([]byte(password), salt, ScryptN, ScryptR, ScryptP, ScryptKeyLen)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return 0, fmt.Errorf("%w: %v", ErrRegistrationFailed, err)
	}

	// Encode the salt and hashed password for storage
	// Format: base64(salt):base64(hash)
	saltBase64 := base64.StdEncoding.EncodeToString(salt)
	hashBase64 := base64.StdEncoding.EncodeToString(hashedPassword)
	passwordHash := fmt.Sprintf("%s:%s", saltBase64, hashBase64)

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("Error beginning transaction: %v", err)
		return 0, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}
	defer tx.Rollback()

	// Insert user - using the correct column names from the database schema
	var userID int64
	now := time.Now()
	err = tx.QueryRowContext(
		ctx,
		`INSERT INTO usr (first_name, last_name, email, password_hash, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6) 
		 RETURNING user_id`,
		firstName, lastName, email, passwordHash, now, now,
	).Scan(&userID)

	if err != nil {
		log.Printf("Error inserting user: %v", err)
		return 0, fmt.Errorf("%w: %v", ErrRegistrationFailed, err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		return 0, fmt.Errorf("%w: %v", ErrDBOperation, err)
	}

	return userID, nil
}
