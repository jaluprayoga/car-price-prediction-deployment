package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/jaluprayoga/car-price-prediction-deployment/internal/config"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// DB is the global database connection pool.
var DB *sql.DB

// User represents a user record in the database.
type User struct {
	ID             int
	Username       string
	Email          string
	HashedPassword string // Map to password_hash database column
	FullName       string
	Role           string
	IsActive       bool
}

// InitDB initializes the PostgreSQL database and creates the schema and tables if they do not exist.
func InitDB() error {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.AppConfig.DBHost,
		config.AppConfig.DBPort,
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		config.AppConfig.DBName,
	)

	log.Printf("[DB] Initializing PostgreSQL database connection with host=%s port=%s dbname=%s",
		config.AppConfig.DBHost, config.AppConfig.DBPort, config.AppConfig.DBName)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("[DB] Failed to open database: %v", err)
		return err
	}

	// Ping the database to verify the connection
	if err = DB.Ping(); err != nil {
		log.Printf("[DB] Failed to ping database: %v", err)
		return err
	}

	// Create auth schema
	schemaQuery := `CREATE SCHEMA IF NOT EXISTS auth;`
	if _, err = DB.Exec(schemaQuery); err != nil {
		log.Printf("[DB] Failed to create schema 'auth': %v", err)
		return err
	}

	// Create users table inside auth schema matching requested columns
	tableQuery := `
	CREATE TABLE IF NOT EXISTS auth.users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		full_name VARCHAR(100),
		role VARCHAR(50) DEFAULT 'user',
		is_active BOOLEAN DEFAULT true,
		created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err = DB.Exec(tableQuery); err != nil {
		log.Printf("[DB] Failed to create table 'auth.users': %v", err)
		return err
	}

	log.Printf("[DB] PostgreSQL database schema and auth.users table verified successfully.")
	return nil
}

// HashPassword hashes a plain-text password using bcrypt.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// VerifyPassword verifies a plain-text password against a bcrypt hash.
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// CreateUser registers a new user in the database. Returns true on success, false if user already exists.
func CreateUser(username, hashedPassword string) (bool, error) {
	email := username
	if !strings.Contains(username, "@") {
		email = username + "@local.com"
	}
	return CreateUserFull(username, email, hashedPassword, username, "user", true)
}

// CreateUserFull registers a user with full details (useful for separate seeding).
func CreateUserFull(username, email, hashedPassword, fullName, role string, isActive bool) (bool, error) {
	if DB == nil {
		return false, errors.New("database connection not initialized")
	}

	query := `
		INSERT INTO auth.users (username, email, password_hash, full_name, role, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := DB.Exec(query, username, email, hashedPassword, fullName, role, isActive)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok {
			if pgErr.Code == "23505" { // unique_violation code
				log.Printf("[DB] User registration aborted. Username or Email already exists: %v", pgErr.Detail)
				return false, nil
			}
		}
		log.Printf("[DB] Failed to create user: %v", err)
		return false, err
	}

	log.Printf("[DB] Successfully registered user: %s (email: %s)", username, email)
	return true, nil
}

// GetUser retrieves user info from the database. It queries both username and email. Returns nil if not found.
func GetUser(identifier string) (*User, error) {
	if DB == nil {
		return nil, errors.New("database connection not initialized")
	}

	query := `
		SELECT id, username, email, password_hash, COALESCE(full_name, ''), COALESCE(role, 'user'), COALESCE(is_active, true)
		FROM auth.users
		WHERE username = $1 OR email = $1
	`
	row := DB.QueryRow(query, identifier)

	var u User
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.HashedPassword, &u.FullName, &u.Role, &u.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &u, nil
}

// SeedDummyUser seeds the dummy user configured in env settings.
func SeedDummyUser() {
	username := config.AppConfig.DummyUserUsername
	password := config.AppConfig.DummyUserPassword

	if username == "" || password == "" {
		return
	}

	existing, err := GetUser(username)
	if err != nil {
		log.Printf("[DB] Error checking for dummy user: %v", err)
		return
	}

	if existing == nil {
		log.Printf("[DB] Seeding dummy user from env settings: %s", username)
		hashed, err := HashPassword(password)
		if err != nil {
			log.Printf("[DB] Failed to hash dummy user password: %v", err)
			return
		}
		email := username
		if !strings.Contains(username, "@") {
			email = username + "@example.com"
		}
		_, _ = CreateUserFull(username, email, hashed, "Administrator", "admin", true)
	} else {
		log.Printf("[DB] Dummy user '%s' already exists in database. Skipping seed.", username)
	}
}
