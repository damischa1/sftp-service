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
	// API key is optional - user's password will be used as X-ApiKey header

	return &PricelistWebAPIStorage{
		baseURL: baseURL,
		apiKey:  apiKey, // This can be empty now
		timeout: 30 * time.Second,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// UserPricelistStorage wraps PricelistWebAPIStorage with user-specific context
type UserPricelistStorage struct {
	storage  *PricelistWebAPIStorage
	username string
	apiKey   string
}

// NewUserPricelistStorage creates a user-specific wrapper around PricelistWebAPIStorage
func NewUserPricelistStorage(storage *PricelistWebAPIStorage, username, apiKey string) *UserPricelistStorage {
	return &UserPricelistStorage{
		storage:  storage,
		username: username,
		apiKey:   apiKey,
	}
}

// DownloadFile delegates to the underlying storage with user context
func (s *UserPricelistStorage) DownloadFile(remotePath string) ([]byte, error) {
	return s.storage.DownloadFileForUser(remotePath, s.username, s.apiKey)
}

// UploadFile delegates to the underlying storage
func (s *UserPricelistStorage) UploadFile(remotePath string, content io.Reader) error {
	return s.storage.UploadFileForUser(remotePath, content, s.username)
}

// DeleteFile delegates to the underlying storage
func (s *UserPricelistStorage) DeleteFile(remotePath string) error {
	return s.storage.DeleteFileForUser(remotePath, s.username)
}

// ListFiles delegates to the underlying storage
func (s *UserPricelistStorage) ListFiles(remotePath string) ([]FileInfo, error) {
	return s.storage.ListFilesForUser(remotePath, s.username)
}

// CreateDirectory delegates to the underlying storage
func (s *UserPricelistStorage) CreateDirectory(remotePath string) error {
	return s.storage.CreateDirectoryForUser(remotePath, s.username)
}

// FileExists delegates to the underlying storage
func (s *UserPricelistStorage) FileExists(remotePath string) (bool, error) {
	return s.storage.FileExistsForUser(remotePath, s.username)
}

// GetFileInfo delegates to the underlying storage
func (s *UserPricelistStorage) GetFileInfo(remotePath string) (*FileInfo, error) {
	return s.storage.GetFileInfoForUser(remotePath, s.username)
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

// DownloadFile fetches pricelist data from the web API (with fallback to mock data)
func (s *PricelistWebAPIStorage) DownloadFile(remotePath string) ([]byte, error) {
	// This should not be called directly - use UserPricelistStorage instead
	return nil, fmt.Errorf("use UserPricelistStorage instead")
}

// DownloadFileForUser fetches pricelist data from the web API (with fallback to mock data)
func (s *PricelistWebAPIStorage) DownloadFileForUser(remotePath, username, userApiKey string) ([]byte, error) {
	// Only allow access to the specific pricelist file
	if remotePath != "/Hinnat/salhydro_kaikki.zip" && remotePath != "salhydro_kaikki.zip" {
		return nil, fmt.Errorf("access denied: only salhydro_kaikki.zip is available")
	}

	url := fmt.Sprintf("%s/api/futur/pricelist", s.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("HTTP request creation failed, using mock data: %v", err)
		return s.getMockPricelistData(), nil
	}

	// Add API key from user (password) to request headers
	req.Header.Set("X-ApiKey", userApiKey)
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
