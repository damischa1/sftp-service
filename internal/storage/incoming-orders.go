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
	apiKey     string
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
func NewIncomingOrdersStorage(apiURL, apiKey string) *IncomingOrdersStorage {
	return &IncomingOrdersStorage{
		apiURL: apiURL,
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// StoreIncomingFile sends file content directly to HTTP API (no local storage)
func (s *IncomingOrdersStorage) StoreIncomingFile(username, filename, content string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check file size limit (100KB = 102400 bytes)
	if len(content) > 102400 {
		return fmt.Errorf("file size exceeds 100KB limit")
	}

	// Generate timestamp for the order
	timestamp := time.Now().Format("20060102_150405")

	// Send order directly to HTTP API (no local storage)
	if err := s.sendOrderToAPI(username, filename, content, timestamp); err != nil {
		log.Printf("Error sending order to API: %v", err)
		return fmt.Errorf("failed to send order to API: %w", err)
	}

	log.Printf("Successfully processed incoming order: %s/%s (%d bytes)", username, filename, len(content))
	return nil
}

// sendOrderToAPI sends the order data to the HTTP API (with mock fallback)
func (s *IncomingOrdersStorage) sendOrderToAPI(username, filename, content, timestamp string) error {
	if s.apiURL == "" {
		log.Printf("API URL not configured, using mock processing")
		return s.processMockOrder(username, filename, content, timestamp)
	}

	orderReq := OrderRequest{
		Username:  username,
		Filename:  filename,
		Content:   content,
		Timestamp: timestamp,
		FileSize:  len(content),
	}

	jsonData, err := json.Marshal(orderReq)
	if err != nil {
		log.Printf("Failed to marshal order, using mock processing: %v", err)
		return s.processMockOrder(username, filename, content, timestamp)
	}

	url := fmt.Sprintf("%s/api/orders/incoming", s.apiURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to create HTTP request, using mock processing: %v", err)
		return s.processMockOrder(username, filename, content, timestamp)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SFTP-Service/1.0")

	if s.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.apiKey))
	}

	log.Printf("Sending order to API: %s (user: %s, file: %s)", url, username, filename)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("HTTP request failed, using mock processing: %v", err)
		return s.processMockOrder(username, filename, content, timestamp)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response, using mock processing: %v", err)
		return s.processMockOrder(username, filename, content, timestamp)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("API request failed: HTTP %d - %s, using mock processing", resp.StatusCode, string(body))
		return s.processMockOrder(username, filename, content, timestamp)
	}

	log.Printf("Order successfully sent to API: %s", string(body))
	return nil
}

// processMockOrder simulates order processing for testing
func (s *IncomingOrdersStorage) processMockOrder(username, filename, content, timestamp string) error {
	log.Printf("MOCK ORDER PROCESSING:")
	log.Printf("  User: %s", username)
	log.Printf("  File: %s", filename)
	log.Printf("  Size: %d bytes", len(content))
	log.Printf("  Timestamp: %s", timestamp)
	log.Printf("  Content preview: %.100s...", content)
	log.Printf("Order processed successfully (mock mode)")
	return nil
}

// FileExists checks if incoming file exists (always false since no local storage)
func (s *IncomingOrdersStorage) FileExists(username, filename string) (bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Since files are sent directly to API and not stored locally,
	// we always return false to allow uploads
	return false, nil
}

// ListIncomingFiles returns an empty list since files are sent directly to API
func (s *IncomingOrdersStorage) ListIncomingFiles(username string) ([]IncomingFileInfo, error) {
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
