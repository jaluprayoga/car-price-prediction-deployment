package tests

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/jaluprayoga/car-price-prediction-deployment/internal/config"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/model"
)

func TestModelInference(t *testing.T) {
	// Load config
	config.LoadConfig()

	// Initialize ONNX Runtime
	err := model.InitONNX(config.AppConfig.OnnxSharedLibPath)
	if err != nil {
		t.Fatalf("Failed to initialize ONNX Runtime: %v", err)
	}
	defer model.CleanupONNX()

	// Check if local model.onnx exists (either in current directory/models or parent/models)
	modelPath := filepath.Join("models", "model.onnx")
	if _, err := os.Stat(modelPath); err != nil {
		// When running tests, Cwd is the directory containing the test file (tests/)
		modelPath = filepath.Join("..", "models", "model.onnx")
		if _, err := os.Stat(modelPath); err != nil {
			modelPath = filepath.Join("..", "..", "training", "models", "model.onnx")
			if _, err := os.Stat(modelPath); err != nil {
				t.Skip("ONNX model file not found, skipping inference test.")
				return
			}
		}
	}

	predictor, err := model.NewPredictor(modelPath)
	if err != nil {
		t.Fatalf("Failed to load predictor: %v", err)
	}
	defer predictor.Destroy()

	features := model.CarFeatures{
		KmDriven:     27000.0,
		Age:          2026 - 2014,
		Mileage:      23.4,
		Engine:       1248.0,
		MaxPower:     74.0,
		Seats:        5.0,
		Fuel:         "Petrol",
		SellerType:   "Dealer",
		Transmission: "Manual",
		Owner:        "First Owner",
	}

	price, err := predictor.Predict(features)
	if err != nil {
		t.Fatalf("Predict failed: %v", err)
	}

	if price < 0 {
		t.Errorf("Expected price to be non-negative, got %f", price)
	}
	log.Printf("Successfully predicted price: %f", price)
}
