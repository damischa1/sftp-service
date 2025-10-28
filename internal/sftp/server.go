package sftp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"	"sftp-service/internal/database"
	"sftp-service/internal/storage""
	"sftp-service/internal/storage"
)

type Server struct {
	db              *database.DB
	storage         *storage.PricelistS3Storage
	incomingStorage *storage.IncomingOrdersStorage
	hostKey         ssh.Signer
	port            string
}

type Config struct {
	DB              *database.DB
	Storage         *storage.PricelistS3Storage
	IncomingStorage *storage.IncomingOrdersStorage
	HostKeyPath     string
	Port            string
}

// NewServer creates a new SFTP server
func NewServer(config *Config) (*Server, error) {
	hostKey, err := loadOrCreateHostKey(config.HostKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load host key: %w", err)
	}

	return &Server{
		db:              config.DB,
		storage:         config.Storage,
		incomingStorage: config.IncomingStorage,
		hostKey:         hostKey,
		port:            config.Port,
	}, nil
}

// Start starts the SFTP server
func (s *Server) Start() error {
	log.Printf("Starting SFTP server on port %s", s.port)

	// Configure SSH server
	sshConfig := &ssh.ServerConfig{
		PasswordCallback: s.passwordCallback,
	}
	sshConfig.AddHostKey(s.hostKey)

	// Listen for connections
	listener, err := net.Listen("tcp", ":"+s.port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", s.port, err)
	}
	defer listener.Close()

	log.Printf("SFTP server listening on port %s", s.port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle connection in a goroutine
		go s.handleConnection(conn, sshConfig)
	}
}

func (s *Server) passwordCallback(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	username := conn.User()
	log.Printf("Authentication attempt for user: %s", username)

	user, err := s.db.AuthenticateUser(username, string(password))
	if err != nil {
		log.Printf("Authentication failed for user %s: %v", username, err)
		return nil, fmt.Errorf("authentication failed")
	}

	log.Printf("Authentication successful for user: %s", username)
	
	// Store username in permissions for later use
	return &ssh.Permissions{
		Extensions: map[string]string{
			"username": user.Username,
		},
	}, nil
}

func (s *Server) handleConnection(conn net.Conn, sshConfig *ssh.ServerConfig) {
	defer conn.Close()

	// Perform SSH handshake
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, sshConfig)
	if err != nil {
		log.Printf("SSH handshake failed: %v", err)
		return
	}
	defer sshConn.Close()

	// Get username from permissions
	username := sshConn.Permissions.Extensions["username"]
	log.Printf("New SSH connection from %s for user %s", conn.RemoteAddr(), username)

	// Handle global requests
	go ssh.DiscardRequests(reqs)

	// Handle channels
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Failed to accept channel: %v", err)
			continue
		}

		// Handle channel requests
		go func(in <-chan *ssh.Request) {
			for req := range in {
				switch req.Type {
				case "subsystem":
					if string(req.Payload[4:]) == "sftp" {
						req.Reply(true, nil)
						s.handleSFTP(channel, username)
					} else {
						req.Reply(false, nil)
					}
				default:
					req.Reply(false, nil)
				}
			}
		}(requests)
	}
}

func (s *Server) handleSFTP(channel ssh.Channel, username string) {
	defer channel.Close()

	log.Printf("Starting SFTP session for user: %s", username)

	// Create S3-backed file system for the user
	filesystem := NewS3FileSystem(s.storage, s.incomingStorage, username)

	// Create SFTP server
	sftpServer, err := sftp.NewServer(
		channel,
		sftp.WithServerWorkingDirectory("/"),
		sftp.WithReadOnly(false),
	)
	if err != nil {
		log.Printf("Failed to create SFTP server: %v", err)
		return
	}
	defer sftpServer.Close()

	// Set handlers
	sftpServer.Handlers = sftp.Handlers{
		FileGet:  filesystem,
		FilePut:  filesystem,
		FileCmd:  filesystem,
		FileList: filesystem,
	}

	// Serve SFTP requests
	if err := sftpServer.Serve(); err != nil && err != io.EOF {
		log.Printf("SFTP server error: %v", err)
	}

	log.Printf("SFTP session ended for user: %s", username)
}

// loadOrCreateHostKey loads an existing host key or creates a new one
func loadOrCreateHostKey(hostKeyPath string) (ssh.Signer, error) {
	// Try to load existing key
	if _, err := os.Stat(hostKeyPath); err == nil {
		keyData, err := os.ReadFile(hostKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read host key: %w", err)
		}

		key, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse host key: %w", err)
		}

		log.Printf("Loaded existing host key from %s", hostKeyPath)
		return key, nil
	}

	// Create new key
	log.Printf("Creating new host key at %s", hostKeyPath)
	
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	privateKeyBytes, err := x509.MarshalPKCS1PrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	keyFile, err := os.Create(hostKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create host key file: %w", err)
	}
	defer keyFile.Close()

	if err := pem.Encode(keyFile, privateKeyPEM); err != nil {
		return nil, fmt.Errorf("failed to write host key: %w", err)
	}

	// Set proper permissions
	if err := os.Chmod(hostKeyPath, 0600); err != nil {
		return nil, fmt.Errorf("failed to set host key permissions: %w", err)
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer from key: %w", err)
	}

	return signer, nil
}
