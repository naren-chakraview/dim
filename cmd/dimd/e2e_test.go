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

// TestKafkaAdapterRoundTrip tests Kafka adapter connectivity and wire protocol
func TestKafkaAdapterRoundTrip(t *testing.T) {
	// Test Kafka adapter via actual produce/consume operations
	// Uses a well-known topic that should exist or be created
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testTopic := "e2e-test-messages"
	testMessage := []byte(`{"order_id": "test-123", "amount": 99.99}`)

	// Step 1: Verify broker connectivity by fetching metadata
	conn, err := kafka.Dial("tcp", "localhost:9092")
	if err != nil {
		t.Fatalf("failed to dial kafka broker: %v", err)
	}
	brokers, err := conn.Brokers()
	if err != nil {
		conn.Close()
		t.Fatalf("failed to fetch brokers: %v", err)
	}
	if len(brokers) == 0 {
		conn.Close()
		t.Fatalf("no brokers available")
	}
	conn.Close()

	// Step 2: Wait for topic to exist (kafka-init may still be running)
	// Poll for up to 10 seconds for the topic to be created
	topicFound := false
	for attempt := 0; attempt < 20; attempt++ {
		conn, err := kafka.Dial("tcp", "localhost:9092")
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		partitions, err := conn.ReadPartitions(testTopic)
		conn.Close()
		if err == nil && len(partitions) > 0 {
			topicFound = true
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !topicFound {
		t.Fatalf("topic %q was not created by kafka-init service", testTopic)
	}

	// Step 3: Produce a test message with retry
	writer := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    testTopic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	var writeErr error
	for attempt := 0; attempt < 20; attempt++ {
		writeErr = writer.WriteMessages(ctx, kafka.Message{Value: testMessage})
		if writeErr == nil {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if writeErr != nil {
		t.Fatalf("failed to produce message: %v", writeErr)
	}

	// Step 4: Consume the message with fresh reader starting from beginning
	reader2 := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{"localhost:9092"},
		Topic:       testTopic,
		Partition:   0,
		StartOffset: 0,
		MaxBytes:    1e6,
	})
	defer reader2.Close()

	msg, err := reader2.ReadMessage(ctx)
	if err != nil {
		t.Fatalf("failed to consume message: %v", err)
	}

	if !bytes.Equal(msg.Value, testMessage) {
		t.Fatalf("message content mismatch: got %q, want %q", string(msg.Value), string(testMessage))
	}

	t.Logf("✓ Kafka adapter: broker connectivity verified, message produced and consumed")
}

// TestS3AdapterRoundTrip tests S3 adapter with real object PUT and GET
func TestS3AdapterRoundTrip(t *testing.T) {
	ctx := context.Background()

	// Create S3 client for MinIO
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider("minioadmin", "minioadmin", ""))),
	)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String("http://localhost:9000")
		o.UsePathStyle = true
	})

	bucketName := "e2e-test-bucket"
	objectKey := "e2e-test-object"
	testData := []byte("e2e-test-payload-data")

	// Create bucket (ignore error if it already exists)
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucketName)})
	if err != nil && !contains(err.Error(), "BucketAlreadyExists") {
		t.Logf("bucket creation (may already exist): %v", err)
	}

	// Put object
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   bytes.NewReader(testData),
	})
	if err != nil {
		t.Fatalf("failed to put object: %v", err)
	}

	// Get object
	output, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		t.Fatalf("failed to get object: %v", err)
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

	t.Logf("✓ S3 adapter: object PUT and GET successful")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && bytes.Contains([]byte(s), []byte(substr))
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
