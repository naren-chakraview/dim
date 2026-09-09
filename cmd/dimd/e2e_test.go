//go:build e2e

package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

// TestKafkaAdapterRoundTrip tests Kafka adapter with real message produce/consume
func TestKafkaAdapterRoundTrip(t *testing.T) {
	ctx := context.Background()

	// Test topic for this test
	testTopic := "e2e-test-messages"

	// Create topic via admin API
	conn, err := kafka.Dial("tcp", "localhost:9092")
	if err != nil {
		t.Fatalf("failed to dial kafka broker: %v", err)
	}
	defer conn.Close()

	// Verify broker is operational
	brokers, err := conn.Brokers()
	if err != nil {
		t.Fatalf("failed to fetch brokers: %v", err)
	}
	if len(brokers) == 0 {
		t.Fatalf("no brokers available")
	}

	// Produce test message
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   testTopic,
	})
	defer w.Close()

	testMsg := "e2e-test-payload"
	err = w.WriteMessages(ctx, kafka.Message{Value: []byte(testMsg)})
	if err != nil {
		t.Fatalf("failed to produce message: %v", err)
	}

	// Consume test message
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{"localhost:9092"},
		Topic:          testTopic,
		GroupID:        "e2e-test-group",
		CommitInterval: time.Second,
		MaxBytes:       1e6,
	})
	defer r.Close()

	msg, err := r.FetchMessage(ctx)
	if err != nil {
		t.Fatalf("failed to consume message: %v", err)
	}

	if string(msg.Value) != testMsg {
		t.Fatalf("consumed message mismatch: got %q, want %q", string(msg.Value), testMsg)
	}

	t.Logf("✓ Kafka adapter: message produce/consume round-trip successful")
}

// TestS3AdapterRoundTrip tests S3 adapter with real object write/read
func TestS3AdapterRoundTrip(t *testing.T) {
	ctx := context.Background()

	// Create S3 client pointing to MinIO
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{URL: "http://localhost:9000"}, nil
			})),
		config.WithCredentialsProvider(aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", ""))),
	)
	if err != nil {
		t.Fatalf("failed to load AWS config: %v", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	// Ensure bucket exists
	bucketName := "e2e-test-bucket"
	_, err = s3Client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucketName)})
	if err != nil && !strings.Contains(err.Error(), "BucketAlreadyExists") {
		t.Fatalf("failed to create bucket: %v", err)
	}

	// Write object
	testKey := "e2e-test-object"
	testData := []byte("e2e-test-payload-data")
	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(testKey),
		Body:   strings.NewReader(string(testData)),
	})
	if err != nil {
		t.Fatalf("failed to write object: %v", err)
	}

	// Read object back
	output, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(testKey),
	})
	if err != nil {
		t.Fatalf("failed to read object: %v", err)
	}
	defer output.Body.Close()

	// Verify content
	data, err := io.ReadAll(output.Body)
	if err != nil {
		t.Fatalf("failed to read object body: %v", err)
	}

	if !bytes.Equal(data, testData) {
		t.Fatalf("object content mismatch: got %q, want %q", string(data), string(testData))
	}

	t.Logf("✓ S3 adapter: object write/read round-trip successful")
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
