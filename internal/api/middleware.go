package api

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/config"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/db"
)

// createAccessToken generates a JWT access token.
func createAccessToken(username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(time.Minute * time.Duration(config.AppConfig.AccessTokenExpireMinutes)).Unix(),
	})
	return token.SignedString(config.AppConfig.JWTSecretKey)
}

// verifyAccessToken decodes and validates a JWT token.
func verifyAccessToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return config.AppConfig.JWTSecretKey, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		sub, ok := claims["sub"].(string)
		if !ok {
			return "", fmt.Errorf("invalid sub claim")
		}
		return sub, nil
	}

	return "", fmt.Errorf("invalid token")
}

// AuthMiddleware handles API validation using either API key or JWT token.
func AuthMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		apiKey := c.Get("X-API-Key")
		authHeader := c.Get("Authorization")

		var token string
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}

		if apiKey == "" && token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"detail": "Authentication required. Provide a valid JWT token or X-API-Key header.",
			})
		}

		// Try validating API Key first if present
		if apiKey != "" {
			if config.AppConfig.APIKeys[apiKey] {
				c.Locals("authenticated_as", fmt.Sprintf("api_key:%s", apiKey))
				return c.Next()
			}
			// If key is invalid and there's no JWT, return 403 Forbidden
			if token == "" {
				log.Printf("[Auth] Unsuccessful API Key verification attempt.")
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"detail": "Invalid API Key.",
				})
			}
		}

		// Fallback to validating JWT token
		if token != "" {
			username, err := verifyAccessToken(token)
			if err != nil {
				log.Printf("[Auth] JWT validation failed: %v", err)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"detail": "Invalid JWT token or API Key.",
				})
			}

			// Validate user exists in DB
			user, err := db.GetUser(username)
			if err != nil || user == nil {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"detail": "Could not validate credentials.",
				})
			}

			c.Locals("authenticated_as", fmt.Sprintf("user:%s", username))
			return c.Next()
		}

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"detail": "Could not validate authentication credentials.",
		})
	}
}
