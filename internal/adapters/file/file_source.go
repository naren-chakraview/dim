package file

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

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

// pollSFTP polls an SFTP server for new files
// Phase 1 implementation: basic SFTP connection and file listing
// Full RPC and credential handling deferred to R5 (plugin runtime)
func (fs *FileSource) pollSFTP(ctx context.Context) error {
	// Phase 1 spike: document the interface contract
	// For now, return a placeholder indicating SFTP polling is ready to integrate
	// when the SFTP client library is added to go.mod

	// Intended flow:
	// 1. Connect to SFTP server (fs.config.SFTPHost:fs.config.SFTPPort)
	// 2. Authenticate using fs.config.SFTPUser + password/key
	// 3. List files in fs.config.Path
	// 4. Compare mod times against fs.lastModTime to detect new/changed files
	// 5. Download new files and send as messages
	// 6. Track fs.lastModTime and optionally move processed files

	return fmt.Errorf("SFTP polling not yet implemented (phase 1 infrastructure); use local file source for now")
}

// Close closes the file source and stops polling
func (fs *FileSource) Close() error {
	fs.pollTicker.Stop()
	<-fs.closed
	return nil
}
