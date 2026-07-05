package db

import (
	"database/sql"
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/jaluprayoga/car-price-prediction-deployment/internal/config"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// DBPath is the path to the SQLite database file. Can be overridden in tests.
var DBPath = filepath.Join("data", "users.db")

// InitDB initializes the SQLite database and creates the users table if it does not exist.
func InitDB() error {
	dir := filepath.Dir(DBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[DB] Failed to create directory '%s': %v", dir, err)
		return err
	}

	log.Printf("[DB] Initializing SQLite database at: %s", DBPath)
	db, err := sql.Open("sqlite3", DBPath)
	if err != nil {
		log.Printf("[DB] Failed to open database: %v", err)
		return err
	}
	defer db.Close()

	query := `
	CREATE TABLE IF NOT EXISTS users (
		username TEXT PRIMARY KEY,
		hashed_password TEXT NOT NULL
	);`

	_, err = db.Exec(query)
	if err != nil {
		log.Printf("[DB] Failed to create table: %v", err)
		return err
	}

	log.Printf("[DB] Users table verified/created successfully.")
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
	db, err := sql.Open("sqlite3", DBPath)
	if err != nil {
		return false, err
	}
	defer db.Close()

	query := "INSERT INTO users (username, hashed_password) VALUES (?, ?)"
	_, err = db.Exec(query, username, hashedPassword)
	if err != nil {
		// Check for SQLite constraint violation (username already exists)
		// Error code 19 is SQLITE_CONSTRAINT
		log.Printf("[DB] Failed to create user '%s': %v", username, err)
		return false, nil
	}

	log.Printf("[DB] Successfully registered user: %s", username)
	return true, nil
}

// User represents a user record in the database.
type User struct {
	Username       string
	HashedPassword string
}

// GetUser retrieves user info from the database. Returns nil if not found.
func GetUser(username string) (*User, error) {
	db, err := sql.Open("sqlite3", DBPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := "SELECT username, hashed_password FROM users WHERE username = ?"
	row := db.QueryRow(query, username)

	var u User
	err = row.Scan(&u.Username, &u.HashedPassword)
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
		_, _ = CreateUser(username, hashed)
	} else {
		log.Printf("[DB] Dummy user '%s' already exists in database. Skipping seed.", username)
	}
}
