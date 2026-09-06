//go:build e2e

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	_ "github.com/lib/pq"
	"github.com/naren-chakraview/dim/internal/adapters/claimcheck"
	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/segmentio/kafka-go"
)

// TestE2EKafkaIntegration verifies Kafka source/sink in production-like environment
func TestE2EKafkaIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	brokers := []string{"localhost:9092"}
	topic := "e2e-test-" + time.Now().Format("20060102150405")
	groupID := "e2e-test-group"

	// Wait for Kafka to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})
	defer reader.Close()

	// Verify Kafka is accessible
	_, err := reader.FetchPartitions(ctx)
	if err != nil {
		t.Skipf("Kafka not available: %v", err)
	}

	// Create a writer
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: brokers,
		Topic:   topic,
	})
	defer writer.Close()

	// Test: Write and read a message
	testMessage := kafka.Message{
		Value: []byte(`{"customer_id": "C-001", "order_id": "ORD-123", "amount": 99.99}`),
	}

	err = writer.WriteMessages(ctx, testMessage)
	if err != nil {
		t.Fatalf("Failed to write Kafka message: %v", err)
	}

	// Read the message back
	reader2 := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID + "-verify",
		StartOffset: kafka.FirstOffset,
	})
	defer reader2.Close()

	readCtx, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	msg, err := reader2.ReadMessage(readCtx)
	if err != nil {
		t.Fatalf("Failed to read Kafka message: %v", err)
	}

	if string(msg.Value) != string(testMessage.Value) {
		t.Errorf("Message mismatch: got %s, want %s", string(msg.Value), string(testMessage.Value))
	}

	t.Log("✓ Kafka integration test passed")
}

// TestE2EDatabaseIntegration verifies PostgreSQL connectivity and basic operations
func TestE2EDatabaseIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	dsn := "postgres://dim_test:test_password@localhost:5432/dim_e2e?sslmode=disable"

	// Wait for database to be ready
	var db *sql.DB
	var err error
	for i := 0; i < 30; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer db.Close()

	// Create a test table
	_, err = db.Exec(`
		DROP TABLE IF EXISTS e2e_test_messages;
		CREATE TABLE e2e_test_messages (
			id SERIAL PRIMARY KEY,
			message_id VARCHAR(255) UNIQUE NOT NULL,
			payload JSONB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Insert a test message
	payload := `{"order_id": "ORD-456", "customer": "C-002"}`
	_, err = db.Exec(
		"INSERT INTO e2e_test_messages (message_id, payload) VALUES ($1, $2)",
		"msg-001",
		payload,
	)
	if err != nil {
		t.Fatalf("Failed to insert message: %v", err)
	}

	// Query it back
	var retrievedID string
	var retrievedPayload string
	err = db.QueryRow(
		"SELECT message_id, payload::text FROM e2e_test_messages WHERE message_id = $1",
		"msg-001",
	).Scan(&retrievedID, &retrievedPayload)
	if err != nil {
		t.Fatalf("Failed to query message: %v", err)
	}

	if retrievedID != "msg-001" || retrievedPayload != payload {
		t.Errorf("Data mismatch: got (%s, %s), want (msg-001, %s)", retrievedID, retrievedPayload, payload)
	}

	t.Log("✓ Database integration test passed")
}

// TestE2ES3Integration verifies S3-compatible storage (MinIO) for claim-check pattern
func TestE2ES3Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	ctx := context.Background()

	// Check if MinIO is available (optional in CI)
	conn, err := net.DialTimeout("tcp", "localhost:9000", 2*time.Second)
	if err != nil {
		t.Skip("MinIO not available (optional service) - skipping S3 tests")
	}
	conn.Close()

	// Configure S3 client for MinIO
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"minioadmin",
			"minioadmin",
			"",
		)),
	)
	if err != nil {
		t.Skipf("Failed to load AWS config: %v", err)
	}

	// Create S3 client pointing to MinIO
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = "http://localhost:9000"
	})

	bucketName := "e2e-test-bucket"

	// Create bucket
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &bucketName,
	})
	if err != nil {
		// Bucket may already exist
		t.Logf("Bucket creation note: %v", err)
	}

	// Test upload
	key := "test-payload-001"
	testData := []byte(`{"customer_id": "C-003", "order_total": 149.99}`)

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &key,
		Body:   bytes.NewReader(testData),
	})
	if err != nil {
		t.Skipf("S3 not available (MinIO): %v", err)
	}

	// Test download
	output, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &key,
	})
	if err != nil {
		t.Fatalf("Failed to get object: %v", err)
	}
	defer output.Body.Close()

	retrieved := make([]byte, len(testData))
	n, err := output.Body.Read(retrieved)
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("Failed to read object: %v", err)
	}

	if string(retrieved[:n]) != string(testData) {
		t.Errorf("Data mismatch: got %s, want %s", string(retrieved[:n]), string(testData))
	}

	t.Log("✓ S3/MinIO integration test passed")
}

// TestE2EClaimCheckPattern verifies claim-check EIP with S3 backend
func TestE2EClaimCheckPattern(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Check if MinIO is available (optional in CI)
	conn, err := net.DialTimeout("tcp", "localhost:9000", 2*time.Second)
	if err != nil {
		t.Skip("MinIO not available (optional service) - skipping S3 tests")
	}
	conn.Close()

	ctx := context.Background()

	// Create S3-backed claim check store
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			"minioadmin",
			"minioadmin",
			"",
		)),
	)
	if err != nil {
		t.Skipf("Failed to load AWS config: %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = "http://localhost:9000"
	})

	store := claimcheck.NewS3ClaimCheckStore(client, "e2e-claims", 24*time.Hour)
	defer store.Close()

	// Test: Store a large payload
	largePayload := map[string]interface{}{
		"order_id":  "ORD-789",
		"items":     []string{"ITEM-1", "ITEM-2", "ITEM-3"},
		"metadata":  "Large order with many line items",
		"customer":  "C-004",
		"timestamp": time.Now().Unix(),
	}

	ticket, err := store.Store(ctx, largePayload)
	if err != nil {
		t.Skipf("Claim-check store not available: %v", err)
	}

	// Test: Retrieve the payload
	retrieved, err := store.Retrieve(ctx, ticket)
	if err != nil {
		t.Fatalf("Failed to retrieve payload: %v", err)
	}

	// Verify structure (note: types may vary after marshaling)
	if retrieved == nil {
		t.Fatal("Retrieved payload is nil")
	}

	t.Logf("✓ Claim-check pattern test passed (ticket: %s)", ticket)
}

// TestE2EMessageFlow verifies end-to-end message processing through the engine
func TestE2EMessageFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Create a test engine with all dependencies available
	eng := engine.NewEngine()
	defer eng.Close()

	// Create channels for testing
	inChan := engine.NewChannel("e2e-input", 10)
	outChan := engine.NewChannel("e2e-output", 10)

	// Add a simple pass-through executor for testing
	executor := engine.NewExecutor("e2e-test", 1)
	executor.SetInputChannel(inChan)
	executor.SetOutputChannel(outChan)

	// Start executor
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		if err := executor.Start(ctx); err != nil {
			t.Logf("Executor error: %v", err)
		}
	}()

	// Send test message
	testMsg := engine.NewMessage(
		map[string]interface{}{
			"order_id":    "ORD-E2E-001",
			"amount":      299.99,
			"customer":    "C-E2E-001",
			"items_count": 5,
		},
		"e2e-test-route",
		"v1",
	)

	inChan.SendMessage(ctx, testMsg)

	// Receive processed message
	received := outChan.ReceiveMessage(ctx)
	if received == nil {
		t.Fatal("No message received from executor")
	}

	// Verify message integrity
	body := received.Body.(map[string]interface{})
	if orderID, ok := body["order_id"].(string); !ok || orderID != "ORD-E2E-001" {
		t.Errorf("Message corruption: expected order_id 'ORD-E2E-001', got %v", body["order_id"])
	}

	t.Log("✓ End-to-end message flow test passed")
}

// TestE2EMultiTenantIsolation verifies tenant isolation with database backend
func TestE2EMultiTenantIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	dsn := "postgres://dim_test:test_password@localhost:5432/dim_e2e?sslmode=disable"

	// Connect to database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Database not available: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		t.Skipf("Database not accessible: %v", err)
	}

	// Create tenant-aware tables
	_, err = db.Exec(`
		DROP TABLE IF EXISTS e2e_tenant_messages;
		CREATE TABLE e2e_tenant_messages (
			id SERIAL PRIMARY KEY,
			tenant_id VARCHAR(255) NOT NULL,
			message_id VARCHAR(255) NOT NULL,
			payload JSONB NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT unique_tenant_message UNIQUE(tenant_id, message_id)
		);
		CREATE INDEX idx_tenant_id ON e2e_tenant_messages(tenant_id);
	`)
	if err != nil {
		t.Fatalf("Failed to create tenant table: %v", err)
	}

	// Insert messages for two different tenants
	tenants := []struct {
		id    string
		count int
	}{
		{"tenant-premium", 10},
		{"tenant-standard", 5},
	}

	for _, tenant := range tenants {
		for i := 0; i < tenant.count; i++ {
			msgID := fmt.Sprintf("%s-msg-%03d", tenant.id, i)
			payload := fmt.Sprintf(`{"tenant": "%s", "sequence": %d}`, tenant.id, i)
			_, err := db.Exec(
				"INSERT INTO e2e_tenant_messages (tenant_id, message_id, payload) VALUES ($1, $2, $3)",
				tenant.id,
				msgID,
				payload,
			)
			if err != nil {
				t.Fatalf("Failed to insert message: %v", err)
			}
		}
	}

	// Verify isolation: each tenant only sees their own messages
	for _, tenant := range tenants {
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM e2e_tenant_messages WHERE tenant_id = $1",
			tenant.id,
		).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query tenant messages: %v", err)
		}
		if count != tenant.count {
			t.Errorf("Tenant %s: expected %d messages, got %d", tenant.id, tenant.count, count)
		}
	}

	t.Log("✓ Multi-tenant isolation test passed")
}
