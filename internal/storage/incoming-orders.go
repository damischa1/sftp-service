package storage

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type IncomingOrdersStorage struct {
	storageDir string
	mutex      sync.RWMutex
}

// NewIncomingOrdersStorage creates a new file-based storage for /in/ directory orders
func NewIncomingOrdersStorage(storageDir string) *IncomingOrdersStorage {
	if storageDir == "" {
		storageDir = "./incoming_orders"
	}
	
	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		log.Printf("Warning: Failed to create storage directory %s: %v", storageDir, err)
	}
	
	return &IncomingOrdersStorage{
		storageDir: storageDir,
	}
}

// StoreIncomingFile stores file content to local file system
func (s *IncomingOrdersStorage) StoreIncomingFile(username, filename, content string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check file size limit (100KB = 102400 bytes)
	if len(content) > 102400 {
		return fmt.Errorf("file size exceeds 100KB limit")
	}

	// Create user directory if it doesn't exist
	userDir := filepath.Join(s.storageDir, username)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return fmt.Errorf("failed to create user directory %s: %w", userDir, err)
	}

	// Add timestamp to filename to avoid conflicts
	timestamp := time.Now().Format("20060102_150405")
	baseFilename := filepath.Base(filename)
	ext := filepath.Ext(baseFilename)
	nameWithoutExt := baseFilename[:len(baseFilename)-len(ext)]
	timestampedFilename := fmt.Sprintf("%s_%s%s", nameWithoutExt, timestamp, ext)
	
	filePath := filepath.Join(userDir, timestampedFilename)

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to store incoming file %s: %w", filename, err)
	}

	log.Printf("Stored incoming file to filesystem: %s (%d bytes)", filePath, len(content))
	return nil
}

// FileExists checks if incoming file exists (not applicable for timestamped files)
func (s *IncomingOrdersStorage) FileExists(username, filename string) (bool, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Since we add timestamps to files, we'll always return false to allow new uploads
	return false, nil
}

// ListIncomingFiles lists files in the user's directory (for directory listing)
func (s *IncomingOrdersStorage) ListIncomingFiles(username string) ([]IncomingFileInfo, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	userDir := filepath.Join(s.storageDir, username)
	
	// Check if user directory exists
	if _, err := os.Stat(userDir); os.IsNotExist(err) {
		return []IncomingFileInfo{}, nil // Empty directory
	}

	entries, err := os.ReadDir(userDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list incoming files: %w", err)
	}

	var files []IncomingFileInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				log.Printf("Warning: Failed to get file info for %s: %v", entry.Name(), err)
				continue
			}
			
			files = append(files, IncomingFileInfo{
				Name:    info.Name(),
				Size:    info.Size(),
				ModTime: info.ModTime(),
			})
		}
	}

	return files, nil
}

type IncomingFileInfo struct {
	Name    string
	Size    int64
	ModTime time.Time
}