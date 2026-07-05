package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration parameters for the application.
type Config struct {
	GoogleCredentials        string
	GCSBucketName            string
	JWTSecretKey             []byte
	JWTAlgorithm             string
	AccessTokenExpireMinutes int
	APIKeys                  map[string]bool
	DummyUserUsername        string
	DummyUserPassword        string
	LogLevel                 string
	OnnxSharedLibPath        string
}

// Global configuration instance
var AppConfig *Config

// LoadConfig loads configuration from the environment and optional .env file.
func LoadConfig() *Config {
	projectRoot := ""
	dir, err := os.Getwd()
	if err == nil {
		for i := 0; i < 4; i++ {
			envPath := filepath.Join(dir, ".env")
			if _, err := os.Stat(envPath); err == nil {
				projectRoot = dir
				if err := godotenv.Load(envPath); err == nil {
					log.Printf("Loaded environment variables from: %s", envPath)
					break
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	if projectRoot == "" {
		projectRoot, _ = os.Getwd()
	}

	apiKeysStr := getEnv("API_KEYS", "test-api-key-12345,external-service-key-999")
	apiKeysMap := make(map[string]bool)
	for _, key := range strings.Split(apiKeysStr, ",") {
		trimmed := strings.TrimSpace(key)
		if trimmed != "" {
			apiKeysMap[trimmed] = true
		}
	}

	expireMinStr := getEnv("ACCESS_TOKEN_EXPIRE_MINUTES", "30")
	expireMin, err := strconv.Atoi(expireMinStr)
	if err != nil {
		expireMin = 30
	}

	libPath := getEnv("ONNXRUNTIME_SHARED_LIB_PATH", "lib/onnxruntime.dll")
	if libPath != "" && !filepath.IsAbs(libPath) {
		libPath = filepath.Join(projectRoot, libPath)
	}

	AppConfig = &Config{
		GoogleCredentials:        getEnv("GOOGLE_APPLICATION_CREDENTIALS", "gcp-key.json"),
		GCSBucketName:            getEnv("GCS_BUCKET_NAME", "car-price-prediction-501013"),
		JWTSecretKey:             []byte(getEnv("JWT_SECRET_KEY", "9a1506b12a52dfdb9c6e3b5e40e2b4d45d62590fae1a2f6fb38a2c20a4b73bfa")),
		JWTAlgorithm:             getEnv("JWT_ALGORITHM", "HS256"),
		AccessTokenExpireMinutes: expireMin,
		APIKeys:                  apiKeysMap,
		DummyUserUsername:        getEnv("DUMMY_USER_USERNAME", "admin"),
		DummyUserPassword:        getEnv("DUMMY_USER_PASSWORD", "adminpassword"),
		LogLevel:                 getEnv("LOG_LEVEL", "INFO"),
		OnnxSharedLibPath:        libPath,
	}

	return AppConfig
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}
