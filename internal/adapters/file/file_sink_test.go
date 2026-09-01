package file

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/engine"
)

// TestFileSinkBasic tests basic file creation, message writing, and JSONL format
func TestFileSinkBasic(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_messages.jsonl")

	// Create a channel and a file sink
	inChan := engine.NewChannel("test-sink", 10)
	sink, err := NewFileSink(filePath, inChan)
	if err != nil {
		t.Fatalf("NewFileSink failed: %v", err)
	}
	defer sink.Close()

	// Start the sink in the background
	ctx, cancel := context.WithCancel(context.Background())
	sink.Start(ctx)

	// Create and send a test message
	testMsg := engine.NewMessage(
		map[string]interface{}{"test": "data"},
		"test-route",
		"v1",
	)

	// Send the message through the channel
	if err := inChan.Send(ctx, testMsg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Give the sink time to process the message
	time.Sleep(100 * time.Millisecond)

	// Close the input channel to signal EOF
	inChan.Close()

	// Cancel context to stop the sink
	cancel()

	// Give the sink time to shut down
	time.Sleep(100 * time.Millisecond)

	// Read and verify the file contents
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	// The file should not be empty
	if len(data) == 0 {
		t.Fatal("file is empty, expected JSONL content")
	}

	// Split by newline and verify the first line is valid JSON
	lines := string(data)
	if lines[len(lines)-1] != '\n' {
		t.Errorf("file does not end with newline")
	}

	// Parse the JSON to verify structure
	var result engine.Message
	err = json.Unmarshal([]byte(lines), &result)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Verify the message content
	if result.Body.(map[string]interface{})["test"] != "data" {
		t.Errorf("message body mismatch: got %v", result.Body)
	}
	if result.Metadata.Route != "test-route" {
		t.Errorf("route mismatch: got %q, want %q", result.Metadata.Route, "test-route")
	}
	if result.Metadata.RouteVersion != "v1" {
		t.Errorf("route version mismatch: got %q, want %q", result.Metadata.RouteVersion, "v1")
	}
}

// TestFileSinkMultipleMessages tests appending multiple messages to the same file
func TestFileSinkMultipleMessages(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_multi.jsonl")

	// Create a channel and sink
	inChan := engine.NewChannel("test-sink", 10)
	sink, err := NewFileSink(filePath, inChan)
	if err != nil {
		t.Fatalf("NewFileSink failed: %v", err)
	}
	defer sink.Close()

	ctx, cancel := context.WithCancel(context.Background())
	sink.Start(ctx)

	// Send multiple messages
	msgCount := 5
	for i := 0; i < msgCount; i++ {
		msg := engine.NewMessage(
			map[string]interface{}{"index": i},
			"test-route",
			"v1",
		)
		if err := inChan.Send(ctx, msg); err != nil {
			t.Fatalf("Send failed on iteration %d: %v", i, err)
		}
	}

	// Give time for processing
	time.Sleep(200 * time.Millisecond)

	// Close and cancel
	inChan.Close()
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Read and verify file contents
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	// Split by lines and verify we have the expected number of lines
	lines := make([]string, 0)
	var lineStart int
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			lines = append(lines, string(data[lineStart:i]))
			lineStart = i + 1
		}
	}

	if len(lines) != msgCount {
		t.Errorf("expected %d lines, got %d", msgCount, len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var msg engine.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			t.Errorf("line %d failed to unmarshal: %v", i, err)
		}
		// Verify the index matches
		if msg.Body.(map[string]interface{})["index"] != float64(i) {
			t.Errorf("line %d: expected index %d, got %v", i, i, msg.Body.(map[string]interface{})["index"])
		}
	}
}

// TestFileSinkAppendMode tests that the sink appends to existing files
func TestFileSinkAppendMode(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_append.jsonl")

	// Write an initial line to the file
	initialMsg := engine.NewMessage(
		map[string]interface{}{"initial": true},
		"test-route",
		"v1",
	)
	initialData, _ := json.Marshal(initialMsg)
	if err := os.WriteFile(filePath, append(initialData, '\n'), 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	// Create a sink and append to the file
	inChan := engine.NewChannel("test-sink", 10)
	sink, err := NewFileSink(filePath, inChan)
	if err != nil {
		t.Fatalf("NewFileSink failed: %v", err)
	}
	defer sink.Close()

	ctx, cancel := context.WithCancel(context.Background())
	sink.Start(ctx)

	// Send a new message
	newMsg := engine.NewMessage(
		map[string]interface{}{"appended": true},
		"test-route",
		"v1",
	)
	if err := inChan.Send(ctx, newMsg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Close and cancel
	inChan.Close()
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Read and verify the file has both lines
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	lines := make([]string, 0)
	var lineStart int
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			lines = append(lines, string(data[lineStart:i]))
			lineStart = i + 1
		}
	}

	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}

	// Verify first line has "initial: true"
	var firstMsg engine.Message
	if err := json.Unmarshal([]byte(lines[0]), &firstMsg); err != nil {
		t.Errorf("failed to unmarshal first line: %v", err)
	}
	if firstMsg.Body.(map[string]interface{})["initial"] != true {
		t.Errorf("first message body mismatch")
	}

	// Verify second line has "appended: true"
	var secondMsg engine.Message
	if err := json.Unmarshal([]byte(lines[1]), &secondMsg); err != nil {
		t.Errorf("failed to unmarshal second line: %v", err)
	}
	if secondMsg.Body.(map[string]interface{})["appended"] != true {
		t.Errorf("second message body mismatch")
	}
}

// TestFileSinkContextCancellation tests that context cancellation stops the sink gracefully
func TestFileSinkContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_cancel.jsonl")

	inChan := engine.NewChannel("test-sink", 10)
	sink, err := NewFileSink(filePath, inChan)
	if err != nil {
		t.Fatalf("NewFileSink failed: %v", err)
	}
	defer sink.Close()

	ctx, cancel := context.WithCancel(context.Background())
	sink.Start(ctx)

	// Send a message
	msg := engine.NewMessage(
		map[string]interface{}{"test": "data"},
		"test-route",
		"v1",
	)
	if err := inChan.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Verify the file exists and has content
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("file is empty after context cancellation")
	}
}

// TestFileSinkCreateParentDirectories tests that parent directories are created if needed
func TestFileSinkCreateParentDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a nested path that doesn't exist yet
	filePath := filepath.Join(tmpDir, "nested", "deep", "path", "test.jsonl")

	inChan := engine.NewChannel("test-sink", 10)
	sink, err := NewFileSink(filePath, inChan)
	if err != nil {
		t.Fatalf("NewFileSink failed: %v", err)
	}
	defer sink.Close()

	// Verify that the parent directories were created
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); err != nil {
		t.Errorf("parent directory was not created: %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("file was not created")
	}
}

// TestFileSinkFilePermissions tests that the file is created with reasonable permissions (0644)
func TestFileSinkFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_perms.jsonl")

	inChan := engine.NewChannel("test-sink", 10)
	sink, err := NewFileSink(filePath, inChan)
	if err != nil {
		t.Fatalf("NewFileSink failed: %v", err)
	}
	defer sink.Close()

	// Check file permissions
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	// 0644 = rw-r--r--
	// The permissions should allow owner to read/write and others to read
	mode := info.Mode().Perm()
	expected := os.FileMode(0644)
	if mode != expected {
		t.Errorf("file permissions mismatch: got %o, want %o", mode, expected)
	}
}

// TestFileSinkNilChannelError tests that NewFileSink returns an error for nil channel
func TestFileSinkNilChannelError(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.jsonl")

	sink, err := NewFileSink(filePath, nil)
	if err == nil {
		t.Fatal("expected error for nil channel, got nil")
	}
	if sink != nil {
		t.Fatal("expected nil sink for nil channel")
	}
}

// TestFileSinkMessageMetadata tests that message metadata is correctly written to JSONL
func TestFileSinkMessageMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_metadata.jsonl")

	inChan := engine.NewChannel("test-sink", 10)
	sink, err := NewFileSink(filePath, inChan)
	if err != nil {
		t.Fatalf("NewFileSink failed: %v", err)
	}
	defer sink.Close()

	ctx, cancel := context.WithCancel(context.Background())
	sink.Start(ctx)

	// Create a message with all metadata fields
	msg := engine.NewMessage(
		map[string]interface{}{"payload": "test"},
		"order-processing",
		"v1.2.3",
	)
	msg.Metadata.Stage = "translate"
	msg.Headers["X-Custom"] = "value"

	if err := inChan.Send(ctx, msg); err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	inChan.Close()
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Read and verify
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var result engine.Message
	if err := json.Unmarshal(data[:len(data)-1], &result); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// Verify all fields
	if result.Body.(map[string]interface{})["payload"] != "test" {
		t.Errorf("body mismatch")
	}
	if result.Metadata.Route != "order-processing" {
		t.Errorf("route mismatch: %q", result.Metadata.Route)
	}
	if result.Metadata.RouteVersion != "v1.2.3" {
		t.Errorf("route version mismatch: %q", result.Metadata.RouteVersion)
	}
	if result.Metadata.Stage != "translate" {
		t.Errorf("stage mismatch: %q", result.Metadata.Stage)
	}
	if result.Headers["X-Custom"] != "value" {
		t.Errorf("headers mismatch: %v", result.Headers)
	}
}

// TestFileSinkJSONLFormat tests that the output is strictly JSONL format (one JSON per line)
func TestFileSinkJSONLFormat(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_format.jsonl")

	inChan := engine.NewChannel("test-sink", 10)
	sink, err := NewFileSink(filePath, inChan)
	if err != nil {
		t.Fatalf("NewFileSink failed: %v", err)
	}
	defer sink.Close()

	ctx, cancel := context.WithCancel(context.Background())
	sink.Start(ctx)

	// Send 3 messages
	for i := 0; i < 3; i++ {
		msg := engine.NewMessage(
			map[string]interface{}{"n": i},
			"test",
			"v1",
		)
		if err := inChan.Send(ctx, msg); err != nil {
			t.Fatalf("Send failed: %v", err)
		}
	}

	time.Sleep(200 * time.Millisecond)

	inChan.Close()
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Read raw file
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	// Every line should be valid JSON, and file should end with newline
	if data[len(data)-1] != '\n' {
		t.Fatal("file does not end with newline")
	}

	// Extract lines and verify each is valid JSON
	lineCount := 0
	var lineStart int
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			line := data[lineStart:i]
			var msg engine.Message
			if err := json.Unmarshal(line, &msg); err != nil {
				t.Errorf("line is not valid JSON: %v", err)
			}
			lineCount++
			lineStart = i + 1
		}
	}

	if lineCount != 3 {
		t.Errorf("expected 3 lines, got %d", lineCount)
	}
}
