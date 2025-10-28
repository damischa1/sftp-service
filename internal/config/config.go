package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL      string
	AWSRegion        string
	AWSAccessKeyID   string
	AWSSecretKey     string
	S3BucketName     string
	SFTPHostKeyPath  string
	SFTPPort         string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	config := &Config{
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://username:password@localhost:5432/sftpdb?sslmode=disable"),
		AWSRegion:       getEnv("AWS_REGION", "eu-west-1"),
		AWSAccessKeyID:  getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretKey:    getEnv("AWS_SECRET_ACCESS_KEY", ""),
		S3BucketName:    getEnv("S3_BUCKET_NAME", ""),
		SFTPHostKeyPath: getEnv("SFTP_HOST_KEY_PATH", "./host_key"),
		SFTPPort:        getEnv("SFTP_PORT", "2222"),
	}

	// Validate required configuration
	if config.AWSAccessKeyID == "" {
		return nil, fmt.Errorf("AWS_ACCESS_KEY_ID is required")
	}
	if config.AWSSecretKey == "" {
		return nil, fmt.Errorf("AWS_SECRET_ACCESS_KEY is required")
	}
	if config.S3BucketName == "" {
		return nil, fmt.Errorf("S3_BUCKET_NAME is required")
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
