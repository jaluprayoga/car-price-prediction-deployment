package main

import (
	"log"

	"github.com/jaluprayoga/car-price-prediction-deployment/internal/config"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/db"
)

func main() {
	log.Println("[Seeder] Starting database seeding process...")

	// 1. Load configuration parameters from env / .env
	config.LoadConfig()

	// 2. Initialize connection pool and create tables
	if err := db.InitDB(); err != nil {
		log.Fatalf("[Seeder] Failed to connect and initialize database: %v", err)
	}
	defer func() {
		if db.DB != nil {
			log.Println("[Seeder] Closing database connection pool.")
			_ = db.DB.Close()
		}
	}()

	// 3. Seed the configured dummy user
	log.Println("[Seeder] Checking and seeding dummy user account...")
	db.SeedDummyUser()

	// 4. Seed user1
	log.Println("[Seeder] Checking and seeding 'user1' account...")
	seedUser1()

	log.Println("[Seeder] Seeding process completed successfully.")
}

func seedUser1() {
	username := "user1"
	email := "user1@example.com"
	password := "user1password"
	fullName := "User One"
	role := "user"
	isActive := true

	existing, err := db.GetUser(username)
	if err != nil {
		log.Printf("[Seeder] Error checking for user1: %v", err)
		return
	}

	if existing == nil {
		log.Printf("[Seeder] Seeding user1...")
		hashed, err := db.HashPassword(password)
		if err != nil {
			log.Printf("[Seeder] Failed to hash password for user1: %v", err)
			return
		}
		_, err = db.CreateUserFull(username, email, hashed, fullName, role, isActive)
		if err != nil {
			log.Printf("[Seeder] Failed to create user1: %v", err)
		}
	} else {
		log.Printf("[Seeder] User '%s' already exists in database. Skipping seed.", username)
	}
}
