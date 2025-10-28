package storage

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3Storage struct {
	client     *s3.S3
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
	bucket     string
}

type FileInfo struct {
	Name         string
	Size         int64
	LastModified time.Time
	IsDir        bool
}

// NewS3Storage creates a new S3 storage client
func NewS3Storage(region, accessKey, secretKey, bucket string) (*S3Storage, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	client := s3.New(sess)
	uploader := s3manager.NewUploader(sess)
	downloader := s3manager.NewDownloader(sess)

	// Test connection by listing bucket
	_, err = client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to access S3 bucket %s: %w", bucket, err)
	}

	log.Printf("Successfully connected to S3 bucket: %s", bucket)
	return &S3Storage{
		client:     client,
		uploader:   uploader,
		downloader: downloader,
		bucket:     bucket,
	}, nil
}

// getUserPath constructs the S3 key with user prefix
func (s *S3Storage) getUserPath(username, path string) string {
	// Clean the path and ensure it starts with username
	cleanPath := strings.TrimPrefix(path, "/")
	if cleanPath == "" {
		return username + "/"
	}
	return username + "/" + cleanPath
}

// UploadFile is a no-op - file upload to S3 is not allowed
func (s *S3Storage) UploadFile(username, remotePath string, content io.Reader) error {
	// No operation - file upload to S3 is disabled for security reasons
	log.Printf("Upload operation attempted on file %s for user %s - operation blocked", remotePath, username)
	return nil
}

// DownloadFile downloads only the specific allowed file from S3
func (s *S3Storage) DownloadFile(username, remotePath string) ([]byte, error) {
	// Only allow downloading the specific file
	if remotePath != "Hinnat/salhydro_kaikki.zip" {
		return nil, fmt.Errorf("file not found: %s", remotePath)
	}
	
	key := s.getUserPath(username, remotePath)
	
	buf := aws.NewWriteAtBuffer([]byte{})
	_, err := s.downloader.Download(buf, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to download file %s: %w", key, err)
	}
	
	log.Printf("Successfully downloaded file: %s for user: %s", remotePath, username)
	return buf.Bytes(), nil
}

// DeleteFile is a no-op - file deletion is not allowed
func (s *S3Storage) DeleteFile(username, remotePath string) error {
	// No operation - file deletion is disabled for security reasons
	log.Printf("Delete operation attempted on file %s for user %s - operation blocked", remotePath, username)
	return nil
}

// ListFiles returns only the specific allowed file: salhydro_kaikki.zip
func (s *S3Storage) ListFiles(username, remotePath string) ([]FileInfo, error) {
	// Only show the specific file in Hinnat directory
	if remotePath == "Hinnat" || remotePath == "Hinnat/" {
		// Check if the specific file exists
		key := s.getUserPath(username, "Hinnat/salhydro_kaikki.zip")
		
		result, err := s.client.HeadObject(&s3.HeadObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		})
		
		if err != nil {
			// If file doesn't exist, return empty list
			if strings.Contains(err.Error(), "NotFound") {
				return []FileInfo{}, nil
			}
			return nil, fmt.Errorf("failed to check file existence: %w", err)
		}
		
		// Return only this specific file
		return []FileInfo{
			{
				Name:         "salhydro_kaikki.zip",
				Size:         *result.ContentLength,
				LastModified: *result.LastModified,
				IsDir:        false,
			},
		}, nil
	}
	
	// For root directory, show only Hinnat folder if it exists
	if remotePath == "" || remotePath == "/" {
		return []FileInfo{
			{
				Name:  "Hinnat",
				IsDir: true,
			},
		}, nil
	}
	
	// For any other path, return empty
	return []FileInfo{}, nil
}

// CreateDirectory creates a directory marker in S3
func (s *S3Storage) CreateDirectory(username, remotePath string) error {
	key := s.getUserPath(username, remotePath)
	if !strings.HasSuffix(key, "/") {
		key += "/"
	}
	
	_, err := s.client.PutObject(&s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader([]byte("")),
	})
	
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", key, err)
	}
	
	log.Printf("Successfully created directory: %s", key)
	return nil
}

// FileExists checks if the specific allowed file exists in S3
func (s *S3Storage) FileExists(username, remotePath string) (bool, error) {
	// Only allow checking for the specific file or Hinnat directory
	if remotePath == "Hinnat" {
		return true, nil // Directory always "exists"
	}
	
	if remotePath != "Hinnat/salhydro_kaikki.zip" {
		return false, nil // All other files don't exist from client perspective
	}
	
	key := s.getUserPath(username, remotePath)
	
	_, err := s.client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	
	if err != nil {
		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check file existence %s: %w", key, err)
	}
	
	return true, nil
}

// GetFileInfo gets information about the specific allowed file
func (s *S3Storage) GetFileInfo(username, remotePath string) (*FileInfo, error) {
	// Only allow getting info for the specific file or Hinnat directory
	if remotePath == "Hinnat" {
		return &FileInfo{
			Name:  "Hinnat",
			IsDir: true,
		}, nil
	}
	
	if remotePath != "Hinnat/salhydro_kaikki.zip" {
		return nil, fmt.Errorf("file not found: %s", remotePath)
	}
	
	key := s.getUserPath(username, remotePath)
	
	result, err := s.client.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for %s: %w", key, err)
	}
	
	return &FileInfo{
		Name:         "salhydro_kaikki.zip",
		Size:         *result.ContentLength,
		LastModified: *result.LastModified,
		IsDir:        false,
	}, nil
}