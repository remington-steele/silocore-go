package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/crypto/scrypt"
)

// Registration errors
var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrPasswordTooWeak    = errors.New("password is too weak")
	ErrRegistrationFailed = errors.New("registration failed")
)

// Scrypt parameters
const (
	ScryptN      = 32768 // CPU/memory cost parameter (power of 2)
	ScryptR      = 8     // Block size parameter
	ScryptP      = 1     // Parallelization parameter
	ScryptKeyLen = 32    // Key length
	SaltSize     = 16    // Salt size in bytes
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

// ValidatePassword checks if a password meets the minimum requirements
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooWeak
	}

	// Additional password strength checks could be added here
	// For example, requiring a mix of uppercase, lowercase, numbers, and special characters

	return nil
}

// VerifyPassword verifies a password against a stored hash
// This is useful for login functionality
func VerifyPassword(storedHash, password string) (bool, error) {
	// Split the stored hash into salt and hash components
	parts := strings.Split(storedHash, ":")
	if len(parts) != 2 {
		return false, errors.New("invalid hash format")
	}

	// Decode the salt and hash
	salt, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return false, fmt.Errorf("error decoding salt: %w", err)
	}

	storedHashBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false, fmt.Errorf("error decoding hash: %w", err)
	}

	// Hash the provided password with the same salt
	hashedPassword, err := scrypt.Key([]byte(password), salt, ScryptN, ScryptR, ScryptP, ScryptKeyLen)
	if err != nil {
		return false, fmt.Errorf("error hashing password: %w", err)
	}

	// Compare the hashes in constant time to prevent timing attacks
	return subtle.ConstantTimeCompare(storedHashBytes, hashedPassword) == 1, nil
}
