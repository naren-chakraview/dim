package file

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/secrets"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// FileSource is a file polling source adapter that monitors a directory for new files
// and converts them into messages sent to an output channel.
// Supports both local filesystem and SFTP sources via the SourceConfig.
type FileSource struct {
	config         SourceConfig
	outChan        *engine.Channel
	closed         chan struct{}
	lifecycle      sync.Mutex // guards running and stopped
	running        bool       // Start's loop is active
	stopped        bool       // Close has been called
	closeOnce      sync.Once
	pollTicker     *time.Ticker
	lastModTime    map[string]time.Time
	pending        map[string]pendingFile // versions seen but not yet settled
	settle         time.Duration          // settle window; zero means defaultSettleWindow
	domain         string                 // domain context for this source (used for secret resolution)
	secretResolver *secrets.Resolver      // optional domain-scoped secret resolver
}

// SourceConfig defines configuration for file polling
type SourceConfig struct {
	Path         string // Local directory path or SFTP path
	Type         string // "local" or "sftp"
	Schedule     string // Poll interval (e.g., "30s", "1m")
	SFTPHost     string // SFTP host (for type: sftp)
	SFTPPort     int    // SFTP port (default 22)
	SFTPUser     string // SFTP username
	SFTPPassword string // SFTP password (or use key auth)
	SFTPKeyFile  string // Path to SSH private key
	ProcessedDir string // Move processed files here (optional)
	FilePattern  string // Match files by pattern (e.g., "*.json")
}

// NewFileSource creates a local file polling source (backward compatible, no domain scoping)
func NewFileSource(path string, schedule string, outChan *engine.Channel) (*FileSource, error) {
	return NewFileSourceWithDomain(path, schedule, outChan, "", nil)
}

// NewFileSourceWithDomain creates a local file polling source with domain-scoped secret resolution
func NewFileSourceWithDomain(path string, schedule string, outChan *engine.Channel, domain string, resolver *secrets.Resolver) (*FileSource, error) {
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
		outChan:        outChan,
		closed:         make(chan struct{}),
		pollTicker:     time.NewTicker(duration),
		lastModTime:    make(map[string]time.Time),
		domain:         domain,
		secretResolver: resolver,
	}, nil
}

// NewSFTPSource creates an SFTP file polling source (backward compatible, no domain scoping)
func NewSFTPSource(host string, port int, user string, path string, schedule string, outChan *engine.Channel) (*FileSource, error) {
	return NewSFTPSourceWithDomain(host, port, user, "", path, schedule, outChan, "", nil)
}

// NewSFTPSourceWithDomain creates an SFTP file polling source with domain-scoped secret resolution.
// password is a secret reference name (not the secret itself). When a resolver and domain are
// configured, the reference is checked with ValidateSecretReferences at construction, so a
// denied or missing reference fails here instead of at first poll.
func NewSFTPSourceWithDomain(host string, port int, user string, password string, path string, schedule string, outChan *engine.Channel, domain string, resolver *secrets.Resolver) (*FileSource, error) {
	if outChan == nil {
		return nil, fmt.Errorf("output channel cannot be nil")
	}

	duration, err := time.ParseDuration(schedule)
	if err != nil {
		return nil, fmt.Errorf("invalid schedule duration: %w", err)
	}

	if password != "" && resolver != nil && domain != "" {
		if err := resolver.ValidateSecretReferences(domain, fmt.Sprintf("${SECRET:%s}", password)); err != nil {
			return nil, fmt.Errorf("invalid SFTP password reference: %w", err)
		}
	}

	return &FileSource{
		config: SourceConfig{
			Path:         path,
			Type:         "sftp",
			Schedule:     schedule,
			SFTPHost:     host,
			SFTPPort:     port,
			SFTPUser:     user,
			SFTPPassword: password,
		},
		outChan:        outChan,
		closed:         make(chan struct{}),
		pollTicker:     time.NewTicker(duration),
		lastModTime:    make(map[string]time.Time),
		domain:         domain,
		secretResolver: resolver,
	}, nil
}

// Start begins polling the configured directory for new files.
// It blocks until the context is cancelled.
func (fs *FileSource) Start(ctx context.Context) error {
	fs.lifecycle.Lock()
	if fs.stopped {
		fs.lifecycle.Unlock()
		return nil // Close ran before Start: nothing to poll
	}
	fs.running = true
	fs.lifecycle.Unlock()

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
			fs.signalClosed()
			return ctx.Err()
		}
	}
}

// resolveSecret resolves a secret reference (T1.13 domain-scoped secret resolution).
//
// The environment-variable fallback is used ONLY when no domain context is configured
// (no resolver, or no domain). Once domain scoping is in effect, any resolution failure,
// including a cross-domain denial, is returned as an error and never falls back to env.
func (fs *FileSource) resolveSecret(ref string) (string, error) {
	if fs.secretResolver == nil || fs.domain == "" {
		return os.Getenv(ref), nil
	}

	value, err := fs.secretResolver.ResolveInRoute(fs.domain, fmt.Sprintf("${SECRET:%s}", ref))
	if err != nil {
		return "", fmt.Errorf("resolve secret %q for domain %q: %w", ref, fs.domain, err)
	}
	return value, nil
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
			// This version of the file has already been emitted
			continue
		}

		// Wait until the file is settled before reading it. A producer writing in place
		// is still writing while its size or mtime changes, so reading it then would emit
		// partial (or empty) content.
		if !fs.isSettled(fullPath, info.Size(), lastMod) {
			continue
		}

		if info.Size() == 0 {
			log.Printf("[WARN] skipping empty file %s (settled at size 0)", fullPath)
			fs.lastModTime[fullPath] = lastMod
			delete(fs.pending, fullPath)
			continue
		}

		// Read and send file contents as message
		data, err := os.ReadFile(fullPath)
		if err != nil {
			log.Printf("[WARN] failed to read file %s: %v", fullPath, err)
			continue
		}

		// The file must not have changed while it was being read. If it did, its new
		// version starts a new settle window on the next poll.
		if after, err := os.Stat(fullPath); err != nil || after.Size() != int64(len(data)) || !after.ModTime().Equal(lastMod) {
			delete(fs.pending, fullPath)
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
		delete(fs.pending, fullPath)

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

// isSettled reports whether a file has kept the same size and mtime for at least the
// settle window. The first time a version is seen, it starts the window and is not
// yet settled.
func (fs *FileSource) isSettled(path string, size int64, mtime time.Time) bool {
	if fs.pending == nil {
		fs.pending = make(map[string]pendingFile)
	}
	p, seen := fs.pending[path]
	if !seen || p.size != size || !p.mtime.Equal(mtime) {
		fs.pending[path] = pendingFile{size: size, mtime: mtime, since: time.Now()}
		return false
	}
	return time.Since(p.since) >= fs.settleWindow()
}

func (fs *FileSource) settleWindow() time.Duration {
	if fs.settle > 0 {
		return fs.settle
	}
	return defaultSettleWindow
}

// pendingFile is a file version that has been seen but not yet settled.
type pendingFile struct {
	size  int64
	mtime time.Time
	since time.Time
}

// defaultSettleWindow is how long a file must stay unchanged before it is read.
const defaultSettleWindow = time.Second

// pollSFTP polls an SFTP server for new files (R21.2).
// Supports both password and private-key authentication.
// Credentials are resolved with domain-scoped secret resolver when available, otherwise from environment variables.
func (fs *FileSource) pollSFTP(ctx context.Context) error {
	// Build SSH authentication config
	config := &ssh.ClientConfig{
		User:            fs.config.SFTPUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: use known_hosts for production
		Timeout:         10 * time.Second,
	}

	// Try password auth first if password is set
	if fs.config.SFTPPassword != "" {
		password, err := fs.resolveSecret(fs.config.SFTPPassword)
		if err != nil {
			// Fail the poll before any network connection is attempted
			return fmt.Errorf("SFTP password: %w", err)
		}
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

// Close stops polling. If Start is running, Close waits for its loop to exit, which
// happens once its context is cancelled. A source that was built but never started
// (test_route builds pipelines this way) returns at once, instead of waiting for a
// loop that never runs.
func (fs *FileSource) Close() error {
	fs.lifecycle.Lock()
	fs.stopped = true
	running := fs.running
	fs.lifecycle.Unlock()

	fs.pollTicker.Stop()
	if !running {
		fs.signalClosed()
		return nil
	}
	<-fs.closed
	return nil
}

// signalClosed closes fs.closed exactly once.
func (fs *FileSource) signalClosed() {
	fs.closeOnce.Do(func() { close(fs.closed) })
}
