package api

import (
	"fmt"
	"log"
	"math"

	"github.com/gofiber/fiber/v2"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/db"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/model"
)

// handleRegister handles user registrations.
func (s *Server) handleRegister(c *fiber.Ctx) error {
	var body UserRegister
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"detail": "Invalid JSON body",
		})
	}

	if len(body.Username) < 3 || len(body.Username) > 50 || len(body.Password) < 6 || len(body.Password) > 100 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"detail": "Username must be 3-50 chars, password 6-100 chars.",
		})
	}

	log.Printf("[API] Received registration request for username: %s", body.Username)
	hashed, err := db.HashPassword(body.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"detail": "Failed to process password",
		})
	}

	success, err := db.CreateUser(body.Username, hashed)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"detail": fmt.Sprintf("Database error: %v", err),
		})
	}

	if !success {
		log.Printf("[API] Registration aborted. Username already taken: %s", body.Username)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"detail": "Username already registered.",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully.",
	})
}

// handleToken handles OAuth2 password flow login.
func (s *Server) handleToken(c *fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	log.Printf("[API] Login attempt for username: %s", username)
	user, err := db.GetUser(username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"detail": fmt.Sprintf("Database error: %v", err),
		})
	}

	if user == nil || !db.VerifyPassword(password, user.HashedPassword) {
		log.Printf("[API] Failed login attempt for username: %s", username)
		c.Set("WWW-Authenticate", "Bearer")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"detail": "Incorrect username or password.",
		})
	}

	token, err := createAccessToken(username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"detail": "Failed to generate token",
		})
	}

	log.Printf("[API] Successful login. Token generated for: %s", username)
	return c.JSON(fiber.Map{
		"access_token": token,
		"token_type":   "bearer",
	})
}

// handlePredict processes car physical and market features to generate price predictions.
func (s *Server) handlePredict(c *fiber.Ctx) error {
	authInfo := c.Locals("authenticated_as").(string)
	log.Printf("[API] Prediction requested by %s", authInfo)

	var payload CarPredictionInput
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"detail": []string{"Failed to parse body"},
		})
	}

	validationErrs := payload.Validate()
	if len(validationErrs) > 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"detail": validationErrs,
		})
	}

	if s.Predictor == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"detail": "Inference execution failed: Model is not loaded.",
		})
	}

	carAge := 2026 - payload.Year
	features := model.CarFeatures{
		KmDriven:     float32(payload.KmDriven),
		Age:          float32(carAge),
		Mileage:      float32(payload.Mileage),
		Engine:       float32(payload.Engine),
		MaxPower:     float32(payload.MaxPower),
		Seats:        float32(payload.Seats),
		Fuel:         payload.Fuel,
		SellerType:   payload.SellerType,
		Transmission: payload.Transmission,
		Owner:        payload.Owner,
	}

	prediction, err := s.Predictor.Predict(features)
	if err != nil {
		log.Printf("[API] Error during endpoint model prediction: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"detail": fmt.Sprintf("Inference execution failed: %v", err),
		})
	}

	// Avoid negative prices
	if prediction < 0 {
		prediction = 0
	}

	PredictionCounter.Inc()
	PredictedPriceHistogram.Observe(float64(prediction))

	log.Printf("[API] Inference completed. Predicted Selling Price: $%.2f USD", prediction)
	return c.JSON(fiber.Map{
		"predicted_price_usd: ": math.Round(float64(prediction)*100) / 100, // round to 2 decimal places
		"predicted_price_usd":   math.Round(float64(prediction)*100) / 100, // duplicate or round as requested
		"currency":              "USD (Dollars)",
		"authenticated_as":      authInfo,
	})
}
