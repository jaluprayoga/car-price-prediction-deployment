package gcs

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/config"
	"google.golang.org/api/option"
)

// getGCSClient initializes a Google Cloud Storage client based on env configuration.
func getGCSClient(ctx context.Context) (*storage.Client, error) {
	credsPath := config.AppConfig.GoogleCredentials
	if credsPath != "" {
		// Resolve relative path based on current directory
		if !filepath.IsAbs(credsPath) {
			// Check if file exists relative to working dir
			if _, err := os.Stat(credsPath); err != nil {
				log.Printf("[GCS] Credentials file specified not found: %s, attempting default auth", credsPath)
				return storage.NewClient(ctx)
			}
		}
		log.Printf("[GCS] Initializing client using credentials from: %s", credsPath)
		return storage.NewClient(ctx, option.WithCredentialsFile(credsPath))
	}

	log.Printf("[GCS] Initializing client using default credentials")
	return storage.NewClient(ctx)
}

// DownloadFromGCS downloads a file from GCS. Returns true on success, false on failure.
func DownloadFromGCS(localFilePath, gcsBlobName, bucketName string) bool {
	if bucketName == "" {
		bucketName = config.AppConfig.GCSBucketName
	}

	ctx := context.Background()
	client, err := getGCSClient(ctx)
	if err != nil {
		log.Printf("[GCS] Client unavailable. Skipping download of '%s' from cloud storage: %v", gcsBlobName, err)
		return false
	}
	defer client.Close()

	log.Printf("[GCS] Fetching blob '%s' from bucket '%s'...", gcsBlobName, bucketName)
	bucket := client.Bucket(bucketName)
	blob := bucket.Object(gcsBlobName)

	// Open reader
	reader, err := blob.NewReader(ctx)
	if err != nil {
		log.Printf("[GCS] Failed to open reader for blob '%s': %v", gcsBlobName, err)
		return false
	}
	defer reader.Close()

	// Ensure destination directory exists
	dir := filepath.Dir(localFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[GCS] Failed to create directory '%s': %v", dir, err)
		return false
	}

	// Open destination file
	outFile, err := os.Create(localFilePath)
	if err != nil {
		log.Printf("[GCS] Failed to create destination file '%s': %v", localFilePath, err)
		return false
	}
	defer outFile.Close()

	// Copy content
	_, err = io.Copy(outFile, reader)
	if err != nil {
		log.Printf("[GCS] Failed to write downloaded content to file: %v", err)
		return false
	}

	log.Printf("[GCS] Successfully downloaded '%s' from GCS to '%s'", gcsBlobName, localFilePath)
	return true
}
