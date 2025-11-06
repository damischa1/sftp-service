package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type IncomingOrdersStorage struct {
	apiURL     string
	apiKey     string // User's password used as API key
	username   string // Current user
	httpClient *http.Client
	mutex      sync.RWMutex
}

type OrderRequest struct {
	Username  string `json:"username"`
	Filename  string `json:"filename"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
	FileSize  int    `json:"file_size"`
}

// NewIncomingOrdersStorage creates a new API-based storage for /in/ directory orders
func NewIncomingOrdersStorage(apiURL, username, apiKey string) *IncomingOrdersStorage {
	return &IncomingOrdersStorage{
		apiURL:   apiURL,
		apiKey:   apiKey, // Store user's API key (password) in apiKey field
		username: username,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// StoreIncomingFile sends file content directly to HTTP API (no local storage)
func (s *IncomingOrdersStorage) StoreIncomingFile(filename, content string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check file size limit (100KB = 102400 bytes)
	if len(content) > 102400 {
		return fmt.Errorf("file size exceeds 100KB limit")
	}

	// Generate timestamp for the order
	timestamp := time.Now().Format("20060102_150405")

	// Send order directly to HTTP API (no local storage)
	if err := s.sendOrderToAPI(filename, content, timestamp); err != nil {
		log.Printf("Error sending order to API: %v", err)
		return fmt.Errorf("failed to send order to API: %w", err)
	}

	log.Printf("Successfully processed incoming order: %s/%s (%d bytes)", s.username, filename, len(content))
	return nil
}

// sendOrderToAPI sends the order data to the HTTP API
func (s *IncomingOrdersStorage) sendOrderToAPI(filename, content, timestamp string) error {
	if s.apiURL == "" {
		return fmt.Errorf("API URL not configured")
	}

	orderReq := OrderRequest{
		Username:  s.username,
		Filename:  filename,
		Content:   content,
		Timestamp: timestamp,
		FileSize:  len(content),
	}

	jsonData, err := json.Marshal(orderReq)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	url := fmt.Sprintf("%s/api/futur/order", s.apiURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SFTP-Service/1.0")
	req.Header.Set("X-ApiKey", s.apiKey)

	log.Printf("Sending order to API: %s (user: %s, file: %s)", url, s.username, filename)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API request failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	log.Printf("Order successfully sent to API: %s", string(body))
	return nil
}

// FileExists checks if incoming file exists (always false since no local storage)
func (s *IncomingOrdersStorage) FileExists(filename string) (bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Since files are sent directly to API and not stored locally,
	// we always return false to allow uploads
	return false, nil
}

// ListIncomingFiles returns an empty list since files are sent directly to API
func (s *IncomingOrdersStorage) ListIncomingFiles() ([]IncomingFileInfo, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Since files are sent directly to API and not stored locally,
	// we return an empty directory listing
	return []IncomingFileInfo{}, nil
}

type IncomingFileInfo struct {
	Name    string
	Size    int64
	ModTime time.Time
}
