package storage

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type FileInfo struct {
	Name         string
	Size         int64
	LastModified time.Time
	IsDir        bool
}

type PricelistWebAPIStorage struct {
	baseURL    string
	username   string
	apiKey     string
	timeout    time.Duration
	httpClient *http.Client
}

// NewPricelistWebAPIStorage creates a new web API storage client for pricelist files
func NewPricelistWebAPIStorage(baseURL, username, apiKey string) *PricelistWebAPIStorage {
	// API key is optional - user's password will be used as X-ApiKey header

	return &PricelistWebAPIStorage{
		baseURL:  baseURL,
		username: username,
		apiKey:   apiKey, // This can be empty now
		timeout:  30 * time.Second,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DownloadFile fetches pricelist data from the web API
func (s *PricelistWebAPIStorage) DownloadFile(remotePath string) ([]byte, error) {
	// Only allow access to the specific pricelist file
	if remotePath != "/Hinnat/salhydro_kaikki.zip" && remotePath != "salhydro_kaikki.zip" {
		return nil, fmt.Errorf("access denied: only salhydro_kaikki.zip is available")
	}

	url := fmt.Sprintf("%s/api/futur/pricelist", s.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add API key from user (password) to request headers
	req.Header.Set("X-ApiKey", s.apiKey)
	req.Header.Set("User-Agent", "SFTP-Service/1.0")

	log.Printf("Downloading pricelist for user %s from web API: %s", s.username, url)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed: HTTP %d", resp.StatusCode)
	}

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("Successfully downloaded pricelist: %d bytes", len(data))
	return data, nil
}

// FileExists checks if the pricelist file exists (always true for salhydro_kaikki.zip)
func (s *PricelistWebAPIStorage) FileExists(remotePath string) (bool, error) {
	// Only the specific pricelist file exists
	exists := remotePath == "salhydro_kaikki.zip"
	log.Printf("File exists check for %s/%s: %v", s.username, remotePath, exists)
	return exists, nil
}

// GetFileInfo returns information about the pricelist file
func (s *PricelistWebAPIStorage) GetFileInfo(remotePath string) (*FileInfo, error) {
	return nil, fmt.Errorf("use UserPricelistStorage instead")
}

// GetFileInfoForUser returns information about the pricelist file
func (s *PricelistWebAPIStorage) GetFileInfoForUser(remotePath, username string) (*FileInfo, error) {
	if remotePath != "salhydro_kaikki.zip" {
		return nil, fmt.Errorf("file not found: %s", remotePath)
	}

	// Return basic file info
	return &FileInfo{
		Name:         "salhydro_kaikki.zip",
		Size:         2 * 1024 * 1024, // 2MB
		LastModified: time.Now(),
		IsDir:        false,
	}, nil
}

// SetTimeout allows customizing the HTTP timeout
func (s *PricelistWebAPIStorage) SetTimeout(timeout time.Duration) {
	s.timeout = timeout
	s.httpClient.Timeout = timeout
}
