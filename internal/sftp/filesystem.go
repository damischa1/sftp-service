package sftp

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
)

// S3FileSystem implements sftp.FileLister, sftp.FileReader, sftp.FileWriter, and sftp.FileCmder interfaces
type S3FileSystem struct {
	storage         Storage
	incomingStorage IncomingStorage  // PostgreSQL storage for /in/ directory
	username        string
	allowedDirs     []string  // Allowed directories for this user
	allowedOps      []string  // Allowed operations
}

// IncomingStorage interface for PostgreSQL file storage (/in/ directory)
type IncomingStorage interface {
	StoreIncomingFile(username, filename, content string) error
	FileExists(username, filename string) (bool, error)
	ListIncomingFiles(username string) ([]IncomingFileInfo, error)
}

type IncomingFileInfo struct {
	Name    string
	Size    int64
	ModTime time.Time
}

// Storage interface defines the methods needed for file operations
type Storage interface {
	UploadFile(username, remotePath string, content io.Reader) error
	DownloadFile(username, remotePath string) ([]byte, error)
	DeleteFile(username, remotePath string) error
	ListFiles(username, remotePath string) ([]FileInfo, error)
	CreateDirectory(username, remotePath string) error
	FileExists(username, remotePath string) (bool, error)
	GetFileInfo(username, remotePath string) (*FileInfo, error)
}

type FileInfo struct {
	Name         string
	Size         int64
	LastModified time.Time
	IsDir        bool
}

// NewS3FileSystem creates a new S3-backed file system with restricted access
func NewS3FileSystem(storage Storage, incomingStorage IncomingStorage, username string) *S3FileSystem {
	return &S3FileSystem{
		storage:         storage,
		incomingStorage: incomingStorage,
		username:        username,
		allowedDirs:     []string{"/", "/in", "/Hinnat"},  // Only root, in, and Hinnat directories
		allowedOps:      []string{"list", "read", "write"}, // Only list, read and write operations
	}
}

// isPathAllowed checks if the given path is allowed for the user
func (fs *S3FileSystem) isPathAllowed(path string) bool {
	// Normalize path
	if path == "" || path == "." {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	
	// Check if path starts with any allowed directory
	for _, allowedDir := range fs.allowedDirs {
		if path == allowedDir || strings.HasPrefix(path, allowedDir+"/") {
			return true
		}
	}
	
	return false
}

// isWriteAllowed checks if writing is allowed in the given path
func (fs *S3FileSystem) isWriteAllowed(path string) bool {
	// Only allow writing to /in and /Hinnat directories (not root)
	if path == "/" {
		return false
	}
	
	return strings.HasPrefix(path, "/in/") || strings.HasPrefix(path, "/Hinnat/") ||
		   path == "/in" || path == "/Hinnat"
}

// getDirectoryFromPath extracts the base directory from a file path
func (fs *S3FileSystem) getDirectoryFromPath(path string) string {
	if path == "/" {
		return "/"
	}
	
	// Remove filename, get directory
	dir := filepath.Dir(path)
	if dir == "." {
		return "/"
	}
	return dir
}

// isInIncomingDirectory checks if path is in /in/ directory
func (fs *S3FileSystem) isInIncomingDirectory(path string) bool {
	return strings.HasPrefix(path, "/in/") && !strings.Contains(strings.TrimPrefix(path, "/in/"), "/")
}

// Fileread implements sftp.FileReader
func (fs *S3FileSystem) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	log.Printf("Reading file: %s for user: %s", r.Filepath, fs.username)
	
	// Check if path is allowed
	if !fs.isPathAllowed(r.Filepath) {
		log.Printf("Access denied: user %s tried to read %s", fs.username, r.Filepath)
		return nil, fmt.Errorf("access denied: path not allowed")
	}
	
	// Deny reading from /in/ directory (write-only)
	if fs.isInIncomingDirectory(r.Filepath) {
		log.Printf("Read denied from /in/: user %s tried to read %s", fs.username, r.Filepath)
		return nil, fmt.Errorf("access denied: /in/ directory is write-only")
	}
	
	data, err := fs.storage.DownloadFile(fs.username, r.Filepath)
	if err != nil {
		return nil, err
	}
	
	return &bytesReaderAt{data: data}, nil
}

// Filewrite implements sftp.FileWriter
func (fs *S3FileSystem) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	log.Printf("Writing file: %s for user: %s", r.Filepath, fs.username)
	
	// Check if path is allowed for writing
	if !fs.isPathAllowed(r.Filepath) || !fs.isWriteAllowed(r.Filepath) {
		log.Printf("Write access denied: user %s tried to write to %s", fs.username, r.Filepath)
		return nil, fmt.Errorf("access denied: write not allowed to this path")
	}
	
	// Handle /in/ directory separately (PostgreSQL storage)
	if fs.isInIncomingDirectory(r.Filepath) {
		filename := filepath.Base(r.Filepath)
		return &incomingWriterAt{
			incomingStorage: fs.incomingStorage,
			username:        fs.username,
			filename:        filename,
		}, nil
	}
	
	// Handle /Hinnat/ directory (S3 storage)
	return &s3WriterAt{
		storage:  fs.storage,
		username: fs.username,
		path:     r.Filepath,
	}, nil
}

// Filecmd implements sftp.FileCmder
func (fs *S3FileSystem) Filecmd(r *sftp.Request) error {
	log.Printf("File command: %s %s for user: %s", r.Method, r.Filepath, fs.username)
	
	// Check if path is allowed
	if !fs.isPathAllowed(r.Filepath) {
		log.Printf("Command access denied: user %s tried %s on %s", fs.username, r.Method, r.Filepath)
		return fmt.Errorf("access denied: path not allowed")
	}
	
	switch r.Method {
	case "Remove":
		// Deny all delete operations
		log.Printf("Delete denied: user %s tried to delete %s", fs.username, r.Filepath)
		return fmt.Errorf("access denied: delete operations not allowed")
	case "Mkdir":
		// Only allow mkdir in /in and /hinnat directories
		if !fs.isWriteAllowed(r.Filepath) {
			log.Printf("Mkdir denied: user %s tried to create directory %s", fs.username, r.Filepath)
			return fmt.Errorf("access denied: directory creation not allowed in this location")
		}
		return fs.storage.CreateDirectory(fs.username, r.Filepath)
	case "Rename":
		// Deny all rename operations
		return fmt.Errorf("access denied: rename operations not allowed")
	case "Rmdir":
		// Deny all directory removal operations
		log.Printf("Rmdir denied: user %s tried to remove directory %s", fs.username, r.Filepath)
		return fmt.Errorf("access denied: directory removal not allowed")
	default:
		return sftp.ErrSSHFxOpUnsupported
	}
}

// Filelist implements sftp.FileLister
func (fs *S3FileSystem) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	log.Printf("Listing directory: %s for user: %s", r.Filepath, fs.username)
	
	// Check if path is allowed
	if !fs.isPathAllowed(r.Filepath) {
		log.Printf("List access denied: user %s tried to list %s", fs.username, r.Filepath)
		return nil, fmt.Errorf("access denied: path not allowed")
	}
	
	// Handle root directory specially - show only allowed subdirectories
	if r.Filepath == "/" || r.Filepath == "" {
		return fs.listRootDirectory()
	}
	
	// Handle /in/ directory specially (PostgreSQL storage)
	if r.Filepath == "/in" {
		return fs.listIncomingDirectory()
	}
	
	files, err := fs.storage.ListFiles(fs.username, r.Filepath)
	if err != nil {
		return nil, err
	}
	
	var fileInfos []os.FileInfo
	for _, file := range files {
		fileInfos = append(fileInfos, &s3FileInfo{
			name:    file.Name,
			size:    file.Size,
			modTime: file.LastModified,
			isDir:   file.IsDir,
		})
	}
	
	return &listerat{files: fileInfos}, nil
}

// listRootDirectory returns only the allowed directories in root
func (fs *S3FileSystem) listRootDirectory() (sftp.ListerAt, error) {
	var fileInfos []os.FileInfo
	
	// Add the allowed directories
	fileInfos = append(fileInfos, &s3FileInfo{
		name:    "in",
		size:    0,
		modTime: time.Now(),
		isDir:   true,
	})
	
	fileInfos = append(fileInfos, &s3FileInfo{
		name:    "Hinnat",
		size:    0,
		modTime: time.Now(),
		isDir:   true,
	})
	
	return &listerat{files: fileInfos}, nil
}

// listIncomingDirectory returns files from PostgreSQL incoming_files table
func (fs *S3FileSystem) listIncomingDirectory() (sftp.ListerAt, error) {
	files, err := fs.incomingStorage.ListIncomingFiles(fs.username)
	if err != nil {
		return nil, fmt.Errorf("failed to list incoming files: %w", err)
	}
	
	var fileInfos []os.FileInfo
	for _, file := range files {
		fileInfos = append(fileInfos, &s3FileInfo{
			name:    file.Name,
			size:    file.Size,
			modTime: file.ModTime,
			isDir:   false, // Files in /in/ are never directories
		})
	}
	
	return &listerat{files: fileInfos}, nil
}

// bytesReaderAt implements io.ReaderAt for byte slices
type bytesReaderAt struct {
	data []byte
}

func (r *bytesReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(r.data)) {
		return 0, io.EOF
	}
	
	n := copy(p, r.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// s3WriterAt implements io.WriterAt for S3 uploads
type s3WriterAt struct {
	storage  Storage
	username string
	path     string
	data     []byte
}

func (w *s3WriterAt) WriteAt(p []byte, off int64) (int, error) {
	// Extend data slice if necessary
	needed := int(off) + len(p)
	if needed > len(w.data) {
		newData := make([]byte, needed)
		copy(newData, w.data)
		w.data = newData
	}
	
	copy(w.data[off:], p)
	return len(p), nil
}

func (w *s3WriterAt) Close() error {
	if len(w.data) > 0 {
		return w.storage.UploadFile(w.username, w.path, strings.NewReader(string(w.data)))
	}
	return nil
}

// incomingWriterAt implements io.WriterAt for PostgreSQL /in/ directory
type incomingWriterAt struct {
	incomingStorage IncomingStorage
	username        string
	filename        string
	data            []byte
}

func (w *incomingWriterAt) WriteAt(p []byte, off int64) (int, error) {
	// Extend data slice if necessary
	needed := int(off) + len(p)
	if needed > len(w.data) {
		newData := make([]byte, needed)
		copy(newData, w.data)
		w.data = newData
	}
	
	copy(w.data[off:], p)
	return len(p), nil
}

func (w *incomingWriterAt) Close() error {
	if len(w.data) > 0 {
		content := string(w.data)
		return w.incomingStorage.StoreIncomingFile(w.username, w.filename, content)
	}
	return nil
}

// s3FileInfo implements os.FileInfo for S3 files
type s3FileInfo struct {
	name    string
	size    int64
	modTime time.Time
	isDir   bool
}

func (fi *s3FileInfo) Name() string       { return fi.name }
func (fi *s3FileInfo) Size() int64        { return fi.size }
func (fi *s3FileInfo) Mode() os.FileMode  { 
	if fi.isDir {
		return os.ModeDir | 0755
	}
	return 0644
}
func (fi *s3FileInfo) ModTime() time.Time { return fi.modTime }
func (fi *s3FileInfo) IsDir() bool        { return fi.isDir }
func (fi *s3FileInfo) Sys() interface{}   { return nil }

// listerat implements sftp.ListerAt
type listerat struct {
	files []os.FileInfo
}

func (l *listerat) ListAt(f []os.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l.files)) {
		return 0, io.EOF
	}
	
	n := copy(f, l.files[offset:])
	if offset+int64(n) >= int64(len(l.files)) {
		return n, io.EOF
	}
	return n, nil
}