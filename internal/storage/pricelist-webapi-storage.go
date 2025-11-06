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

// UploadFile is disabled for pricelist API - pricelists are read-only
func (s *PricelistWebAPIStorage) UploadFile(remotePath string, content io.Reader) error {
	return fmt.Errorf("use UserPricelistStorage instead")
}

// UploadFileForUser is disabled for pricelist API - pricelists are read-only
func (s *PricelistWebAPIStorage) UploadFileForUser(remotePath string, content io.Reader, username string) error {
	log.Printf("Upload attempt blocked for pricelist API: %s/%s", username, remotePath)
	return fmt.Errorf("upload not allowed for pricelist files")
}

// DownloadFile fetches pricelist data from the web API
func (s *PricelistWebAPIStorage) DownloadFile(remotePath string) ([]byte, error) {
	// This should not be called directly - use UserPricelistStorage instead
	return nil, fmt.Errorf("use UserPricelistStorage instead")
}

// DownloadFileForUser fetches pricelist data from the web API
func (s *PricelistWebAPIStorage) DownloadFileForUser(remotePath, username, userApiKey string) ([]byte, error) {
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
	req.Header.Set("X-ApiKey", userApiKey)
	req.Header.Set("User-Agent", "SFTP-Service/1.0")

	log.Printf("Downloading pricelist for user %s from web API: %s", username, url)

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

// DeleteFile is disabled for pricelist API - pricelists are read-only
func (s *PricelistWebAPIStorage) DeleteFile(remotePath string) error {
	return fmt.Errorf("use UserPricelistStorage instead")
}

// DeleteFileForUser is disabled for pricelist API - pricelists are read-only
func (s *PricelistWebAPIStorage) DeleteFileForUser(remotePath, username string) error {
	log.Printf("Delete attempt blocked for pricelist API: %s/%s", username, remotePath)
	return fmt.Errorf("delete not allowed for pricelist files")
}

// ListFiles lists available pricelist files for a user (no API call needed for listing)
func (s *PricelistWebAPIStorage) ListFiles(remotePath string) ([]FileInfo, error) {
	return nil, fmt.Errorf("use UserPricelistStorage instead")
}

// ListFilesForUser lists available pricelist files for a user (no API call needed for listing)
func (s *PricelistWebAPIStorage) ListFilesForUser(remotePath, username string) ([]FileInfo, error) {
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
func (s *PricelistWebAPIStorage) CreateDirectory(remotePath string) error {
	return fmt.Errorf("use UserPricelistStorage instead")
}

// CreateDirectoryForUser is disabled for pricelist API - pricelists are read-only
func (s *PricelistWebAPIStorage) CreateDirectoryForUser(remotePath, username string) error {
	log.Printf("Directory creation blocked for pricelist API: %s/%s", username, remotePath)
	return fmt.Errorf("directory creation not allowed for pricelist storage")
}

// FileExists checks if the pricelist file exists (always true for salhydro_kaikki.zip)
func (s *PricelistWebAPIStorage) FileExists(remotePath string) (bool, error) {
	return false, fmt.Errorf("use UserPricelistStorage instead")
}

// FileExistsForUser checks if the pricelist file exists (always true for salhydro_kaikki.zip)
func (s *PricelistWebAPIStorage) FileExistsForUser(remotePath, username string) (bool, error) {
	// Only the specific pricelist file exists
	exists := remotePath == "salhydro_kaikki.zip"
	log.Printf("File exists check for %s/%s: %v", username, remotePath, exists)
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
