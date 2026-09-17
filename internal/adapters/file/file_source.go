package file

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// resolveSecret resolves ${SECRET:name} references from environment variables.
// Returns the environment variable value, or empty string if not found.
//
// TODO (T1.13): Wire domain-scoped secret resolver from internal/secrets instead of unscoped os.Getenv.
// Current limitation: this adapter cannot enforce domain-scoped secret isolation or cross-domain
// authorization checks. Once domain context is available to adapters, call secrets.Resolver.ResolveInRoute
// with the route's domain to enable M4.4 domain-scoped secret resolution.
// This requires: (1) passing domain through adapter interfaces, (2) adding authorization checks
// for cross-domain.name syntax, (3) rejecting cross-domain access without @shared syntax.
func resolveSecret(ref string) string {
	return os.Getenv(ref)
}

// FileSource is a file polling source adapter that monitors a directory for new files
// and converts them into messages sent to an output channel.
// Supports both local filesystem and SFTP sources via the SourceConfig.
type FileSource struct {
	config      SourceConfig
	outChan     *engine.Channel
	closed      chan struct{}
	pollTicker  *time.Ticker
	lastModTime map[string]time.Time
}

// SourceConfig defines configuration for file polling
type SourceConfig struct {
	Path          string        // Local directory path or SFTP path
	Type          string        // "local" or "sftp"
	Schedule      string        // Poll interval (e.g., "30s", "1m")
	SFTPHost      string        // SFTP host (for type: sftp)
	SFTPPort      int           // SFTP port (default 22)
	SFTPUser      string        // SFTP username
	SFTPPassword  string        // SFTP password (or use key auth)
	SFTPKeyFile   string        // Path to SSH private key
	ProcessedDir  string        // Move processed files here (optional)
	FilePattern   string        // Match files by pattern (e.g., "*.json")
}

// NewFileSource creates a local file polling source
func NewFileSource(path string, schedule string, outChan *engine.Channel) (*FileSource, error) {
	if outChan == nil {
		return nil, fmt.Errorf("output channel cannot be nil")
	}

	duration, err := time.ParseDuration(schedule)
	if err != nil {
		return nil, fmt.Errorf("invalid schedule duration: %w", err)
	}

	return &FileSource{
		config: SourceConfig{
			Path:     path,
			Type:     "local",
			Schedule: schedule,
		},
		outChan:     outChan,
		closed:      make(chan struct{}),
		pollTicker:  time.NewTicker(duration),
		lastModTime: make(map[string]time.Time),
	}, nil
}

// NewSFTPSource creates an SFTP file polling source
func NewSFTPSource(host string, port int, user string, path string, schedule string, outChan *engine.Channel) (*FileSource, error) {
	if outChan == nil {
		return nil, fmt.Errorf("output channel cannot be nil")
	}

	duration, err := time.ParseDuration(schedule)
	if err != nil {
		return nil, fmt.Errorf("invalid schedule duration: %w", err)
	}

	return &FileSource{
		config: SourceConfig{
			Path:     path,
			Type:     "sftp",
			Schedule: schedule,
			SFTPHost: host,
			SFTPPort: port,
			SFTPUser: user,
		},
		outChan:     outChan,
		closed:      make(chan struct{}),
		pollTicker:  time.NewTicker(duration),
		lastModTime: make(map[string]time.Time),
	}, nil
}

// Start begins polling the configured directory for new files.
// It blocks until the context is cancelled.
func (fs *FileSource) Start(ctx context.Context) error {
	// Initial poll
	if err := fs.poll(ctx); err != nil {
		log.Printf("[WARN] initial poll failed: %v", err)
	}

	// Periodic polling
	for {
		select {
		case <-fs.pollTicker.C:
			if err := fs.poll(ctx); err != nil {
				log.Printf("[WARN] poll error: %v", err)
			}
		case <-ctx.Done():
			fs.pollTicker.Stop()
			close(fs.closed)
			return ctx.Err()
		}
	}
}

// poll checks for new or modified files and sends them as messages
func (fs *FileSource) poll(ctx context.Context) error {
	switch fs.config.Type {
	case "local":
		return fs.pollLocal(ctx)
	case "sftp":
		return fs.pollSFTP(ctx)
	default:
		return fmt.Errorf("unsupported source type: %s", fs.config.Type)
	}
}

// pollLocal polls the local filesystem for new files
func (fs *FileSource) pollLocal(ctx context.Context) error {
	entries, err := os.ReadDir(fs.config.Path)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", fs.config.Path, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		fullPath := filepath.Join(fs.config.Path, filename)

		// Check if we've already processed this file
		info, err := os.Stat(fullPath)
		if err != nil {
			log.Printf("[WARN] failed to stat file %s: %v", fullPath, err)
			continue
		}

		lastMod := info.ModTime()
		if lastModTime, exists := fs.lastModTime[fullPath]; exists && lastMod.Equal(lastModTime) {
			// File hasn't been modified since last poll
			continue
		}

		// Read and send file contents as message
		data, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("[WARN] failed to read file %s: %v", fullPath, err)
			continue
		}

		// Create a message from the file content
		// Route name defaults to "file-source", route version is empty (set by executor)
		msg := engine.NewMessage(string(data), "file-source", "")
		msg.Headers["source_file"] = filename
		msg.Headers["source_path"] = fullPath

		// Send to output channel
		if err := fs.outChan.Send(ctx, msg); err != nil {
			log.Printf("[WARN] failed to send file message: %v", err)
			continue
		}
		fs.lastModTime[fullPath] = lastMod

		// Move file to processed directory if configured
		if fs.config.ProcessedDir != "" {
			processedPath := filepath.Join(fs.config.ProcessedDir, filename)
			if err := os.Rename(fullPath, processedPath); err != nil {
				log.Printf("[WARN] failed to move file %s to %s: %v", fullPath, processedPath, err)
			}
		}
	}

	return nil
}

// pollSFTP polls an SFTP server for new files (R21.2).
// Supports both password and private-key authentication.
// Credentials are resolved from environment variables via resolveSecret.
func (fs *FileSource) pollSFTP(ctx context.Context) error {
	// Build SSH authentication config
	config := &ssh.ClientConfig{
		User:            fs.config.SFTPUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: use known_hosts for production
		Timeout:         10 * time.Second,
	}

	// Try password auth first if password is set
	if fs.config.SFTPPassword != "" {
		password := resolveSecret(fs.config.SFTPPassword)
		if password != "" {
			config.Auth = []ssh.AuthMethod{ssh.Password(password)}
		}
	}

	// Try key auth if key file is set
	if fs.config.SFTPKeyFile != "" {
		keyData, err := os.ReadFile(fs.config.SFTPKeyFile)
		if err != nil {
			return fmt.Errorf("failed to read SSH key file: %w", err)
		}

		// Try to parse as private key
		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			return fmt.Errorf("failed to parse SSH key: %w", err)
		}

		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	}

	if len(config.Auth) == 0 {
		return fmt.Errorf("no SFTP authentication method configured (password or key required)")
	}

	// Set default port if not specified
	port := fs.config.SFTPPort
	if port == 0 {
		port = 22
	}

	// Connect to SSH server
	addr := fmt.Sprintf("%s:%d", fs.config.SFTPHost, port)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return fmt.Errorf("failed to connect to SFTP server %s: %w", addr, err)
	}
	defer conn.Close()

	// Create SFTP client
	client, err := sftp.NewClient(conn)
	if err != nil {
		return fmt.Errorf("failed to create SFTP client: %w", err)
	}
	defer client.Close()

	// List files in the remote directory
	entries, err := client.ReadDir(fs.config.Path)
	if err != nil {
		return fmt.Errorf("failed to list SFTP directory %s: %w", fs.config.Path, err)
	}

	// Process each file
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		remotePath := filepath.Join(fs.config.Path, filename)

		// Check if we've already processed this file
		lastMod := entry.ModTime()
		if lastModTime, exists := fs.lastModTime[remotePath]; exists && lastMod.Equal(lastModTime) {
			// File hasn't been modified since last poll
			continue
		}

		// Download file
		remoteFile, err := client.Open(remotePath)
		if err != nil {
			log.Printf("[WARN] failed to open SFTP file %s: %v", remotePath, err)
			continue
		}

		data, err := io.ReadAll(remoteFile)
		remoteFile.Close()
		if err != nil {
			log.Printf("[WARN] failed to read SFTP file %s: %v", remotePath, err)
			continue
		}

		// Create a message from the file content
		msg := engine.NewMessage(string(data), "sftp-source", "")
		msg.Headers["source_file"] = filename
		msg.Headers["source_path"] = remotePath
		msg.Headers["source_host"] = fs.config.SFTPHost

		// Send to output channel
		if err := fs.outChan.Send(ctx, msg); err != nil {
			log.Printf("[WARN] failed to send SFTP file message: %v", err)
			continue
		}
		fs.lastModTime[remotePath] = lastMod
	}

	return nil
}

// Close closes the file source and stops polling
func (fs *FileSource) Close() error {
	fs.pollTicker.Stop()
	<-fs.closed
	return nil
}
