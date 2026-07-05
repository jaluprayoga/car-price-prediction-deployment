package main

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/api"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/config"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/db"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/gcs"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/model"
)

func main() {
	log.Println("Initializing API application lifecycle...")

	// 1. Load Configurations
	cfg := config.LoadConfig()

	// 2. Initialize Database and Seed Dummy User
	if err := db.InitDB(); err != nil {
		log.Fatalf("Failed to initialize SQLite Database: %v", err)
	}
	db.SeedDummyUser()

	// 3. Download/Locate ONNX Model Artifact
	modelPath := filepath.Join("models", "model.onnx")
	ensureModelArtifact(modelPath)

	// 4. Initialize ONNX Runtime environment
	log.Printf("Initializing ONNX Runtime using shared library: %s", cfg.OnnxSharedLibPath)
	if err := model.InitONNX(cfg.OnnxSharedLibPath); err != nil {
		log.Fatalf("Failed to initialize ONNX Runtime: %v. Please make sure the shared library is correctly installed.", err)
	}
	defer model.CleanupONNX()

	// 5. Load model predictor
	log.Printf("Loading ONNX model from: %s", modelPath)
	predictor, err := model.NewPredictor(modelPath)
	if err != nil {
		log.Fatalf("Failed to load predictor session: %v", err)
	}
	defer predictor.Destroy()
	log.Println("ModelPredictor preloaded and stored in app state.")

	// 6. Setup Fiber Server
	app := fiber.New(fiber.Config{
		AppName: "Car Price Prediction ML API (Golang)",
	})

	// Add request logger middleware
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	server := &api.Server{
		Predictor: predictor,
	}
	server.SetupRouter(app)

	// 7. Run Server on port 8000
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	log.Printf("Starting Fiber Server on port %s...", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Server failed to run: %v", err)
	}
}

// ensureModelArtifact checks for model.onnx and resolves it via GCS or local copy.
func ensureModelArtifact(modelPath string) {
	if _, err := os.Stat(modelPath); err == nil {
		log.Printf("Model artifact already exists at: %s", modelPath)
		return
	}

	// 1. Try downloading from GCS
	log.Println("Model artifact not found locally. Attempting to download from GCS...")
	gcsSuccess := gcs.DownloadFromGCS(modelPath, "models/model.onnx", "")
	if gcsSuccess {
		return
	}

	// 2. Local fallback to sibling training folder
	log.Println("GCS download failed or skipped. Searching for sibling training folder fallback...")
	wd, _ := os.Getwd()
	siblingModelPath := filepath.Join(filepath.Dir(wd), "training", "models", "model.onnx")
	if _, err := os.Stat(siblingModelPath); err == nil {
		log.Printf("Found fallback training model at: %s. Copying...", siblingModelPath)
		err = copyFile(siblingModelPath, modelPath)
		if err == nil {
			log.Println("Successfully copied local fallback model.")
			return
		}
		log.Printf("Failed to copy local fallback model: %v", err)
	}

	log.Fatalf("CRITICAL: Model artifact could not be downloaded from GCS and fallback not found. Please train the model or upload it.")
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Sync()
}
