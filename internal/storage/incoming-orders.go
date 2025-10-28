package storage

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type IncomingOrdersStorage struct {
	db *sql.DB
}

// NewIncomingOrdersStorage creates a new PostgreSQL file storage for /in/ directory orders
func NewIncomingOrdersStorage(db *sql.DB) *IncomingOrdersStorage {
	return &IncomingOrdersStorage{db: db}
}

// StoreIncomingFile stores file content to incoming_files table
func (s *IncomingOrdersStorage) StoreIncomingFile(username, filename, content string) error {
	// Check file size limit (100KB = 102400 bytes)
	if len(content) > 102400 {
		return fmt.Errorf("file size exceeds 100KB limit")
	}

	query := `
		INSERT INTO incoming_files (username, filename, file_content, file_size)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (username, filename) 
		DO UPDATE SET 
			file_content = EXCLUDED.file_content,
			file_size = EXCLUDED.file_size,
			created_at = CURRENT_TIMESTAMP`

	_, err := s.db.Exec(query, username, filename, content, len(content))
	if err != nil {
		return fmt.Errorf("failed to store incoming file %s: %w", filename, err)
	}

	log.Printf("Stored incoming file to database: %s/%s (%d bytes)", username, filename, len(content))
	return nil
}

// FileExists checks if incoming file exists
func (s *IncomingOrdersStorage) FileExists(username, filename string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM incoming_files WHERE username = $1 AND filename = $2)`
	
	err := s.db.QueryRow(query, username, filename).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check incoming file existence: %w", err)
	}

	return exists, nil
}

// ListIncomingFiles lists files in incoming_files table for user (for directory listing)
func (s *IncomingOrdersStorage) ListIncomingFiles(username string) ([]IncomingFileInfo, error) {
	query := `
		SELECT filename, file_size, created_at
		FROM incoming_files 
		WHERE username = $1 
		ORDER BY filename ASC`

	rows, err := s.db.Query(query, username)
	if err != nil {
		return nil, fmt.Errorf("failed to list incoming files: %w", err)
	}
	defer rows.Close()

	var files []IncomingFileInfo
	for rows.Next() {
		var file IncomingFileInfo
		err := rows.Scan(&file.Name, &file.Size, &file.ModTime)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incoming file info: %w", err)
		}
		files = append(files, file)
	}

	return files, nil
}

type IncomingFileInfo struct {
	Name    string
	Size    int64
	ModTime time.Time
}