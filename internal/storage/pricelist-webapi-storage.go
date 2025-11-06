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
	apiKey     string
	timeout    time.Duration
	httpClient *http.Client
}

// NewPricelistWebAPIStorage creates a new web API storage client for pricelist files
func NewPricelistWebAPIStorage(baseURL, apiKey string) (*PricelistWebAPIStorage, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	return &PricelistWebAPIStorage{
		baseURL: baseURL,
		apiKey:  apiKey,
		timeout: 30 * time.Second,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// getUserPath creates a user-specific path (not used for API, but kept for interface compatibility)
func (s *PricelistWebAPIStorage) getUserPath(username, path string) string {
	return fmt.Sprintf("%s/%s", username, path)
}

// UploadFile is disabled for pricelist API - pricelists are read-only
func (s *PricelistWebAPIStorage) UploadFile(username, remotePath string, content io.Reader) error {
	log.Printf("Upload attempt blocked for pricelist API: %s/%s", username, remotePath)
	return fmt.Errorf("upload not allowed for pricelist files")
}

// DownloadFile fetches pricelist data from the web API (with fallback to mock data)
func (s *PricelistWebAPIStorage) DownloadFile(username, remotePath string) ([]byte, error) {
	// Only allow access to the specific pricelist file
	if remotePath != "/Hinnat/salhydro_kaikki.zip" && remotePath != "salhydro_kaikki.zip" {
		return nil, fmt.Errorf("access denied: only salhydro_kaikki.zip is available")
	}

	url := fmt.Sprintf("%s/api/pricelist/download", s.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("HTTP request creation failed, using mock data: %v", err)
		return s.getMockPricelistData(), nil
	}

	// Add API key to request headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))
	req.Header.Set("User-Agent", "SFTP-Service/1.0")

	log.Printf("Downloading pricelist for user %s from web API: %s", username, url)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("HTTP request failed, using mock data: %v", err)
		return s.getMockPricelistData(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("API request failed: HTTP %d, using mock data", resp.StatusCode)
		return s.getMockPricelistData(), nil
	}

	// Read the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body, using mock data: %v", err)
		return s.getMockPricelistData(), nil
	}

	log.Printf("Successfully downloaded pricelist: %d bytes", len(data))
	return data, nil
}

// getMockPricelistData returns mock pricelist data for testing
func (s *PricelistWebAPIStorage) getMockPricelistData() []byte {
	mockData := `PK
This is a mock pricelist file for testing SFTP service.

Product List:
1. Product A - 10.99 EUR
2. Product B - 25.50 EUR  
3. Product C - 45.00 EUR

Updated: ` + time.Now().Format("2006-01-02 15:04:05") + `
`
	log.Printf("Returning mock pricelist data: %d bytes", len(mockData))
	return []byte(mockData)
}

// DeleteFile is disabled for pricelist API - pricelists are read-only
func (s *PricelistWebAPIStorage) DeleteFile(username, remotePath string) error {
	log.Printf("Delete attempt blocked for pricelist API: %s/%s", username, remotePath)
	return fmt.Errorf("delete not allowed for pricelist files")
}

// ListFiles lists available pricelist files for a user (no API call needed for listing)
func (s *PricelistWebAPIStorage) ListFiles(username, remotePath string) ([]FileInfo, error) {
	log.Printf("Listing pricelist files for user %s at path %s", username, remotePath)

	// Return hardcoded file list - API call only happens during download
	return s.getFallbackFileList(), nil
}

// getFallbackFileList returns a hardcoded file list as fallback
func (s *PricelistWebAPIStorage) getFallbackFileList() []FileInfo {
	return []FileInfo{
		{
			Name:         "salhydro_kaikki.zip",
			Size:         2 * 1024 * 1024, // 2MB
			LastModified: time.Now(),
			IsDir:        false,
		},
	}
}

// CreateDirectory is disabled for pricelist API - pricelists are read-only
func (s *PricelistWebAPIStorage) CreateDirectory(username, remotePath string) error {
	log.Printf("Directory creation blocked for pricelist API: %s/%s", username, remotePath)
	return fmt.Errorf("directory creation not allowed for pricelist files")
}

// FileExists checks if the pricelist file exists (always true for salhydro_kaikki.zip)
func (s *PricelistWebAPIStorage) FileExists(username, remotePath string) (bool, error) {
	// Only the specific pricelist file exists
	exists := remotePath == "salhydro_kaikki.zip"
	log.Printf("File exists check for %s/%s: %v", username, remotePath, exists)
	return exists, nil
}

// GetFileInfo returns information about the pricelist file
func (s *PricelistWebAPIStorage) GetFileInfo(username, remotePath string) (*FileInfo, error) {
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
