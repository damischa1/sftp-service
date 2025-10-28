package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"sftp-service/internal/config"
	"sftp-service/internal/database"
	"sftp-service/internal/sftp"
	"sftp-service/internal/storage"
)

func main() {
	log.Println("Starting SFTP service...")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	db, err := database.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize S3 storage
	s3Storage, err := storage.NewS3Storage(
		cfg.AWSRegion,
		cfg.AWSAccessKeyID,
		cfg.AWSSecretKey,
		cfg.S3BucketName,
	)
	if err != nil {
		log.Fatalf("Failed to initialize S3 storage: %v", err)
	}

	// Initialize PostgreSQL file storage for /in/ directory
	incomingStorage := storage.NewPostgreSQLFileStorage(db.GetConnection())

	// Create SFTP server
	sftpServer, err := sftp.NewServer(&sftp.Config{
		DB:              db,
		Storage:         s3Storage,
		IncomingStorage: incomingStorage,
		HostKeyPath:     cfg.SFTPHostKeyPath,
		Port:            cfg.SFTPPort,
	})
	if err != nil {
		log.Fatalf("Failed to create SFTP server: %v", err)
	}

	// Set up graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := sftpServer.Start(); err != nil {
			log.Fatalf("SFTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-c
	log.Println("Shutting down SFTP service...")
}
