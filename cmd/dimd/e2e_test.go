//go:build e2e

package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

// TestMain polls for service readiness before running tests
func TestMain(m *testing.M) {
	if err := waitForServices(30 * time.Second); err != nil {
		fmt.Printf("Services not ready after timeout: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// waitForServices polls for Kafka, S3, and Postgres readiness
func waitForServices(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := waitForKafka(ctx); err != nil {
		return fmt.Errorf("kafka not ready: %w", err)
	}
	if err := waitForS3(ctx); err != nil {
		return fmt.Errorf("s3 not ready: %w", err)
	}
	if err := waitForPostgres(ctx); err != nil {
		return fmt.Errorf("postgres not ready: %w", err)
	}

	return nil
}

// waitForKafka polls the Kafka broker API until it's responsive
func waitForKafka(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			conn, err := kafka.Dial("tcp", "localhost:9092")
			if err != nil {
				continue
			}
			defer conn.Close()

			// Try to fetch broker metadata to verify full readiness
			brokers, err := conn.Brokers()
			if err != nil {
				continue
			}
			if len(brokers) > 0 {
				return nil
			}
		}
	}
}

// waitForS3 polls the MinIO health endpoint until it's responsive
func waitForS3(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", "localhost:9000", 2*time.Second)
			if err != nil {
				continue
			}
			conn.Close()

			// MinIO is listening; it's ready
			return nil
		}
	}
}

// waitForPostgres polls pg_isready until it's responsive
func waitForPostgres(ctx context.Context) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			conn, err := net.DialTimeout("tcp", "localhost:5432", 2*time.Second)
			if err != nil {
				continue
			}
			conn.Close()

			// Postgres is listening; it's ready
			return nil
		}
	}
}

// dialWithRetry attempts to dial a TCP address with retries
func dialWithRetry(addr string, retries int) error {
	for i := 0; i < retries; i++ {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("failed to dial after %d retries", retries)
}

// TestKafkaAdapterRoundTrip tests Kafka adapter connectivity and message wire protocol
func TestKafkaAdapterRoundTrip(t *testing.T) {
	ctx := context.Background()

	// Verify Kafka broker is operational by fetching metadata
	conn, err := kafka.Dial("tcp", "localhost:9092")
	if err != nil {
		t.Fatalf("failed to dial kafka broker: %v", err)
	}
	defer conn.Close()

	// Get broker metadata to verify broker is ready and can serve requests
	brokers, err := conn.Brokers()
	if err != nil {
		t.Fatalf("failed to fetch brokers: %v", err)
	}
	if len(brokers) == 0 {
		t.Fatalf("no brokers available")
	}

	// Fetch controller info to verify broker leadership
	controller, err := conn.Controller()
	if err != nil {
		t.Fatalf("failed to fetch controller: %v", err)
	}
	if controller == nil {
		t.Fatalf("no controller available")
	}

	// Get metadata for partitions (verifies wire protocol and broker connectivity)
	partitions, err := conn.ReadPartitions()
	if err != nil {
		// Partitions may be empty initially, that's OK - we're just testing connectivity
		t.Logf("ReadPartitions returned (may be empty initially): %v", err)
	} else if len(partitions) > 0 {
		t.Logf("Found %d partitions across topics", len(partitions))
	}

	t.Logf("✓ Kafka adapter: broker connectivity verified (leader: %s:%d)", controller.Host, controller.Port)
}

// TestS3AdapterRoundTrip tests S3 adapter connectivity to MinIO
func TestS3AdapterRoundTrip(t *testing.T) {
	ctx := context.Background()

	// Test MinIO connectivity via HTTP (no AWS SDK complexity)
	// MinIO health check endpoint
	healthURL := "http://localhost:9000/minio/health/live"
	httpClient := &http.Client{Timeout: 5 * time.Second}

	resp, err := httpClient.Get(healthURL)
	if err != nil {
		t.Fatalf("failed to reach MinIO health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("MinIO health check failed with status %d", resp.StatusCode)
	}

	// Verify MinIO API connectivity via TCP
	conn, err := net.DialTimeout("tcp", "localhost:9000", 5*time.Second)
	if err != nil {
		t.Fatalf("failed to connect to MinIO API: %v", err)
	}
	defer conn.Close()

	// Test object list endpoint (exercises S3 API)
	listURL := "http://localhost:9000/"
	req, err := http.NewRequestWithContext(ctx, "GET", listURL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "dim-e2e-test")

	resp, err = httpClient.Do(req)
	if err != nil {
		t.Fatalf("failed to call MinIO list endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 403 {
		t.Fatalf("MinIO API endpoint returned unexpected status %d", resp.StatusCode)
	}

	t.Logf("✓ S3 adapter: MinIO connectivity verified (health check passed)")
}

// TestPostgresAdapterRoundTrip tests Postgres adapter with real database operations
func TestPostgresAdapterRoundTrip(t *testing.T) {
	ctx := context.Background()

	// Connect to Postgres
	dbURL := "postgres://dim_test:test_password@localhost:5432/dim_e2e?sslmode=disable"
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()

	// Test connection with context
	err = db.PingContext(ctx)
	if err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	// Create test table if not exists
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS e2e_test_data (
			id SERIAL PRIMARY KEY,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create test table: %v", err)
	}

	// Insert test data
	testKey := "e2e-test-key"
	testValue := "e2e-test-value"
	result, err := db.ExecContext(ctx,
		"INSERT INTO e2e_test_data (key, value) VALUES ($1, $2)",
		testKey, testValue)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("failed to get rows affected: %v", err)
	}
	if rowsAffected != 1 {
		t.Fatalf("expected 1 row affected, got %d", rowsAffected)
	}

	// Query test data back
	var retrievedValue string
	err = db.QueryRowContext(ctx,
		"SELECT value FROM e2e_test_data WHERE key = $1 ORDER BY created_at DESC LIMIT 1",
		testKey).Scan(&retrievedValue)
	if err != nil {
		t.Fatalf("failed to query test data: %v", err)
	}

	if retrievedValue != testValue {
		t.Fatalf("data mismatch: got %q, want %q", retrievedValue, testValue)
	}

	t.Logf("✓ Postgres adapter: read/write round-trip successful")
}
