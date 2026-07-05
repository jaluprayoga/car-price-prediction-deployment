package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/api"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/config"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/db"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/model"
)

type MockPredictor struct{}

func (m *MockPredictor) Predict(features model.CarFeatures) (float32, error) {
	return 5000.0, nil
}

func setupTestDB(t *testing.T) func() {
	// Set test database path
	db.DBPath = filepath.Join("..", "data", "test_users.db")
	_ = os.Remove(db.DBPath)

	err := db.InitDB()
	if err != nil {
		t.Fatalf("Failed to initialize test DB: %v", err)
	}

	// Return cleanup function
	return func() {
		_ = os.Remove(db.DBPath)
	}
}

func getTestApp() *fiber.App {
	config.LoadConfig()
	app := fiber.New()
	server := &api.Server{
		Predictor: &MockPredictor{},
	}
	server.SetupRouter(app)
	return app
}

func TestRootEndpoint(t *testing.T) {
	app := getTestApp()

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	_ = json.Unmarshal(bodyBytes, &result)

	if result["status"] != "healthy" {
		t.Errorf("Expected status to be healthy, got %v", result["status"])
	}
}

func TestUserFlowAuthAndPrediction(t *testing.T) {
	cleanup := setupTestDB(t)
	defer cleanup()

	app := getTestApp()

	username := fmt.Sprintf("user_%d", time.Now().UnixNano())
	password := "secretpassword"

	// 1. Register User
	regPayload := fmt.Sprintf(`{"username":"%s","password":"%s"}`, username, password)
	req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(regPayload))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d", resp.StatusCode)
	}

	// Try duplicate registration
	reqDup := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(regPayload))
	reqDup.Header.Set("Content-Type", "application/json")
	respDup, _ := app.Test(reqDup, -1)
	defer respDup.Body.Close()
	if respDup.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for duplicate user, got %d", respDup.StatusCode)
	}

	// 2. Get Access Token (Login)
	loginPayload := fmt.Sprintf("username=%s&password=%s", username, password)
	reqToken := httptest.NewRequest("POST", "/api/auth/token", strings.NewReader(loginPayload))
	reqToken.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	respToken, err := app.Test(reqToken, -1)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	defer respToken.Body.Close()

	if respToken.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", respToken.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(respToken.Body)
	var tokenRes map[string]interface{}
	_ = json.Unmarshal(bodyBytes, &tokenRes)

	token, ok := tokenRes["access_token"].(string)
	if !ok || token == "" {
		t.Fatalf("Failed to retrieve access token: %v", tokenRes)
	}

	if tokenRes["token_type"] != "bearer" {
		t.Errorf("Expected bearer token type, got %v", tokenRes["token_type"])
	}

	// Try invalid login
	badLoginPayload := fmt.Sprintf("username=%s&password=wrongpassword", username)
	reqBadLogin := httptest.NewRequest("POST", "/api/auth/token", strings.NewReader(badLoginPayload))
	reqBadLogin.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respBadLogin, _ := app.Test(reqBadLogin, -1)
	defer respBadLogin.Body.Close()
	if respBadLogin.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected 401 for wrong credentials, got %d", respBadLogin.StatusCode)
	}

	// 3. Request Prediction with valid token
	dummyCar := `{"year":2014,"km_driven":27000,"fuel":"Petrol","seller_type":"Dealer","transmission":"Manual","owner":"First Owner","mileage":23.4,"engine":1248.0,"max_power":74.0,"seats":5}`
	reqPredict := httptest.NewRequest("POST", "/api/predict", strings.NewReader(dummyCar))
	reqPredict.Header.Set("Content-Type", "application/json")
	reqPredict.Header.Set("Authorization", "Bearer "+token)

	respPredict, err := app.Test(reqPredict, -1)
	if err != nil {
		t.Fatalf("Prediction request failed: %v", err)
	}
	defer respPredict.Body.Close()

	if respPredict.StatusCode != http.StatusOK {
		t.Fatalf("Expected prediction success 200, got %d", respPredict.StatusCode)
	}

	bodyPredBytes, _ := io.ReadAll(respPredict.Body)
	var predRes map[string]interface{}
	_ = json.Unmarshal(bodyPredBytes, &predRes)

	priceVal, hasPrice := predRes["predicted_price_usd"]
	if !hasPrice {
		t.Errorf("Expected response to contain predicted_price_usd, got %v", predRes)
	}
	price, ok := priceVal.(float64)
	if !ok || price < 0 {
		t.Errorf("Invalid price value: %v", priceVal)
	}
}

func TestPredictAPIKeyAuth(t *testing.T) {
	app := getTestApp()

	dummyCar := `{"year":2014,"km_driven":27000,"fuel":"Petrol","seller_type":"Dealer","transmission":"Manual","owner":"First Owner","mileage":23.4,"engine":1248.0,"max_power":74.0,"seats":5}`
	req := httptest.NewRequest("POST", "/api/predict", strings.NewReader(dummyCar))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-api-key-12345") // configured key from .env

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Prediction request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	var predRes map[string]interface{}
	_ = json.Unmarshal(bodyBytes, &predRes)

	priceVal, hasPrice := predRes["predicted_price_usd"]
	if !hasPrice || priceVal.(float64) < 0 {
		t.Errorf("Invalid prediction response: %v", predRes)
	}

	authAs, hasAuth := predRes["authenticated_as"]
	if !hasAuth || !strings.Contains(authAs.(string), "api_key") {
		t.Errorf("Expected authenticated_as to show api_key: %v", authAs)
	}
}

func TestPredictUnauthorized(t *testing.T) {
	app := getTestApp()

	dummyCar := `{"year":2014,"km_driven":27000,"fuel":"Petrol","seller_type":"Dealer","transmission":"Manual","owner":"First Owner","mileage":23.4,"engine":1248.0,"max_power":74.0,"seats":5}`

	// No headers
	req1 := httptest.NewRequest("POST", "/api/predict", strings.NewReader(dummyCar))
	req1.Header.Set("Content-Type", "application/json")
	resp1, _ := app.Test(req1, -1)
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for no auth, got %d", resp1.StatusCode)
	}

	// Invalid API Key
	req2 := httptest.NewRequest("POST", "/api/predict", strings.NewReader(dummyCar))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-API-Key", "invalid-key-here")
	resp2, _ := app.Test(req2, -1)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403 for invalid API Key, got %d", resp2.StatusCode)
	}
}

func TestPredictInvalidPayload(t *testing.T) {
	app := getTestApp()

	// Invalid fields (e.g. year: 1800, km_driven: -100, seats: 15)
	invalidCar := `{"year":1800,"km_driven":-100,"fuel":"Water","seller_type":"Individual","transmission":"Manual","owner":"First Owner","mileage":23.4,"engine":1248.0,"max_power":74.0,"seats":15}`
	req := httptest.NewRequest("POST", "/api/predict", strings.NewReader(invalidCar))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-api-key-12345")

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Prediction request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("Expected status 422 for invalid payload, got %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	var errRes map[string]interface{}
	_ = json.Unmarshal(bodyBytes, &errRes)

	errors, hasErrors := errRes["detail"]
	if !hasErrors {
		t.Errorf("Expected response to contain detail, got %v", errRes)
	}

	errList, ok := errors.([]interface{})
	if !ok || len(errList) == 0 {
		t.Errorf("Expected list of validation messages, got: %v", errors)
	}
}
