package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AuthAPIURL       string
	PricelistAPIURL  string
	PricelistAPIKey  string
	SFTPHostKeyPath  string
	SFTPPort         string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist)
	_ = godotenv.Load()

	config := &Config{
		AuthAPIURL:      getEnv("AUTH_API_URL", "http://localhost:8080"),
		PricelistAPIURL: getEnv("PRICELIST_API_URL", "http://localhost:8081"),
		PricelistAPIKey: getEnv("PRICELIST_API_KEY", ""),
		SFTPHostKeyPath: getEnv("SFTP_HOST_KEY_PATH", "./host_key"),
		SFTPPort:        getEnv("SFTP_PORT", "2222"),
	}

	// Validate required configuration
	if config.AuthAPIURL == "" {
		return nil, fmt.Errorf("AUTH_API_URL is required")
	}
	if config.PricelistAPIURL == "" {
		return nil, fmt.Errorf("PRICELIST_API_URL is required")
	}
	if config.PricelistAPIKey == "" {
		return nil, fmt.Errorf("PRICELIST_API_KEY is required")
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
