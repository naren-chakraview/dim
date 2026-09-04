package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/factory"
	"gopkg.in/yaml.v3"
)

// TestRunCommandEndToEnd tests the full pipeline from HTTP ingestion to file output
func TestRunCommandEndToEnd(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test config
	cfg := &config.RouteConfig{
		Version: 1,
		Sources: map[string]config.SourceSpec{
			"http-source": {
				Type: "http",
			},
		},
		Sinks: map[string]config.SinkSpec{
			"output": {
				Type: "file",
				Path: filepath.Join(tmpDir, "output.jsonl"),
			},
		},
		Routes: map[string]config.RouteSpec{
			"main": {
				From:  "http-source",
				Steps: []config.StepSpec{},
			},
		},
	}

	// Write config to file
	configPath := filepath.Join(tmpDir, "config.yaml")
	configData, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Start the pipeline with a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	executor, _, _, _, sources, sinks, err := factory.BuildPipeline(ctx, cfg)
	if err != nil {
		t.Fatalf("BuildPipeline failed: %v", err)
	}

	// Start source in goroutine (Start() is blocking)
	sourceErr := make(chan error, 1)
	for _, source := range sources {
		go func(src factory.SourceAdapter) {
			sourceErr <- src.Start(ctx)
		}(source)
	}

	// Start sinks
	for _, sink := range sinks {
		if err := sink.Start(ctx); err != nil {
			t.Fatalf("sink startup failed: %v", err)
		}
	}

	// Run executor in goroutine
	executorDone := make(chan error, 1)
	go func() {
		executorDone <- executor.Run(ctx)
	}()

	// Give the server time to start
	time.Sleep(100 * time.Millisecond)

	// Send test message via HTTP
	payload := map[string]interface{}{
		"test": "data",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	resp, err := http.Post("http://localhost:8080/ingest", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", resp.StatusCode)
	}

	// Give sink time to write
	time.Sleep(100 * time.Millisecond)

	// Shutdown - context timeout will trigger graceful shutdown
	<-executorDone
	<-sourceErr

	// Verify output file exists and contains data
	outputPath := filepath.Join(tmpDir, "output.jsonl")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("output file is empty")
	}

	// Verify the line is valid JSON
	var msg engine.Message
	lineEnd := bytes.IndexByte(data, '\n')
	if lineEnd == -1 {
		lineEnd = len(data)
	}
	if err := json.Unmarshal(data[:lineEnd], &msg); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}

	// Clean up
	for _, sink := range sinks {
		sink.Stop()
	}
	for _, source := range sources {
		source.Stop()
	}
}

// TestMissingConfigFile tests error handling for missing config
func TestMissingConfigFile(t *testing.T) {
	_, err := config.LoadRouteConfig("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

// TestInvalidConfigFile tests error handling for invalid config
func TestInvalidConfigFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an invalid config file
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(configPath, []byte("invalid: yaml: : :"), 0644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	_, err := config.LoadRouteConfig(configPath)
	if err == nil {
		t.Fatal("expected error for invalid config")
	}
}
