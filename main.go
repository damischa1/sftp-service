package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"sftp-service/internal/auth"
	"sftp-service/internal/config"
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

	// Initialize web API authenticator
	authenticator := auth.NewWebAPIAuthenticator(cfg.FuturAPIURL)

	// Initialize Web API storage for pricelist
	pricelistStorage, err := storage.NewPricelistWebAPIStorage(
		cfg.FuturAPIURL,
		"", // No separate API key needed - using user password
	)
	if err != nil {
		log.Fatalf("Failed to initialize pricelist API storage: %v", err)
	}

	// Initialize API storage for /in/ directory orders
	incomingStorage := storage.NewIncomingOrdersStorage(
		cfg.FuturAPIURL,
		"", // No separate API key needed - using user password
	)

	// Create SFTP server
	sftpServer, err := sftp.NewServer(&sftp.Config{
		Authenticator:   authenticator,
		Storage:         pricelistStorage,
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
