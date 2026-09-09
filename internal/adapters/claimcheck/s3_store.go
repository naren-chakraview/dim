package claimcheck

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3ClaimCheckStore is an S3-backed implementation of ClaimCheckStore (M3.2)
// Stores large payloads in S3 and returns lightweight ticket references
type S3ClaimCheckStore struct {
	client     *s3.Client
	bucket     string
	prefix     string
	defaultTTL time.Duration
	mu         sync.RWMutex
	closed     bool

	// Metadata cache for performance (avoid repeated GetObject for metadata)
	metadataCache map[string]ClaimCheckMetadata
}

// S3StoreConfig configures the S3-backed claim-check store
type S3StoreConfig struct {
	Bucket      string        // S3 bucket name (required)
	Region      string        // AWS region (e.g., "us-east-1")
	Prefix      string        // Key prefix for claim-check objects (e.g., "claim-check/")
	DefaultTTL  time.Duration // Default TTL for stored payloads (e.g., 30 days)
	Endpoint    string        // Optional: custom S3 endpoint (for MinIO testing)
	AccessKey   string        // Optional: S3 access key (for custom endpoints)
	SecretKey   string        // Optional: S3 secret key (for custom endpoints)
}

// NewS3ClaimCheckStore creates a new S3-backed claim-check store (M3.2)
func NewS3ClaimCheckStore(cfg S3StoreConfig) (*S3ClaimCheckStore, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("S3 bucket name is required")
	}

	if cfg.Region == "" {
		cfg.Region = "us-east-1" // Default region
	}

	if cfg.DefaultTTL == 0 {
		cfg.DefaultTTL = 30 * 24 * time.Hour // Default: 30 days
	}

	if cfg.Prefix == "" {
		cfg.Prefix = "claim-check/"
	}

	// Load AWS SDK configuration
	ctx := context.Background()
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}

	// Support custom S3 endpoint for MinIO in tests
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = os.Getenv("S3_ENDPOINT")
	}

	accessKey := cfg.AccessKey
	if accessKey == "" {
		accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
		if accessKey == "" {
			accessKey = os.Getenv("S3_ACCESS_KEY")
		}
	}

	secretKey := cfg.SecretKey
	if secretKey == "" {
		secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
		if secretKey == "" {
			secretKey = os.Getenv("S3_SECRET_KEY")
		}
	}

	if endpoint != "" && accessKey != "" && secretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(aws.NewCredentialsCache(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""))))
	}

	sdkConfig, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Configure custom endpoint for MinIO if provided
	clientOpts := func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	}

	client := s3.NewFromConfig(sdkConfig, clientOpts)

	store := &S3ClaimCheckStore{
		client:         client,
		bucket:         cfg.Bucket,
		prefix:         cfg.Prefix,
		defaultTTL:     cfg.DefaultTTL,
		metadataCache:  make(map[string]ClaimCheckMetadata),
	}

	// Verify S3 connectivity with a health check
	if err := store.HealthCheck(context.Background()); err != nil {
		return nil, fmt.Errorf("S3 health check failed: %w", err)
	}

	return store, nil
}

// Store saves a payload to S3 and returns a ticket reference
func (s *S3ClaimCheckStore) Store(ctx context.Context, payload []byte, metadata ClaimCheckMetadata) (string, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return "", fmt.Errorf("store is closed")
	}
	s.mu.RUnlock()

	// Generate ticket
	ticket := fmt.Sprintf("s3://%s/%s%s", s.bucket, s.prefix, generateRandomID())

	// Compute hash for integrity
	hash := computeHash(payload)
	if metadata.ContentType == "" {
		metadata.ContentType = "application/octet-stream"
	}
	metadata.Hash = hash
	metadata.Size = int64(len(payload))
	metadata.Timestamp = time.Now()

	// Set expiration if TTL provided
	if s.defaultTTL > 0 {
		expiresAt := time.Now().Add(s.defaultTTL)
		metadata.ExpiresAt = &expiresAt
	}

	// Upload to S3
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(s.prefix + getTicketKey(ticket)),
		Body:        bytes.NewReader(payload),
		ContentType: aws.String(metadata.ContentType),
		Metadata: map[string]string{
			"hash":      metadata.Hash,
			"size":      fmt.Sprintf("%d", metadata.Size),
			"timestamp": metadata.Timestamp.Format(time.RFC3339),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to store payload in S3: %w", err)
	}

	// Cache metadata
	s.mu.Lock()
	s.metadataCache[ticket] = metadata
	s.mu.Unlock()

	return ticket, nil
}

// Retrieve loads a payload from S3 using the ticket
func (s *S3ClaimCheckStore) Retrieve(ctx context.Context, ticket string) ([]byte, ClaimCheckMetadata, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, ClaimCheckMetadata{}, fmt.Errorf("store is closed")
	}
	s.mu.RUnlock()

	// Check cache first
	s.mu.RLock()
	if meta, ok := s.metadataCache[ticket]; ok {
		s.mu.RUnlock()
		// Still need to fetch payload from S3
		result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(s.prefix + getTicketKey(ticket)),
		})
		if err != nil {
			return nil, ClaimCheckMetadata{}, fmt.Errorf("failed to retrieve payload from S3: %w", err)
		}
		defer result.Body.Close()

		payload, err := io.ReadAll(result.Body)
		if err != nil {
			return nil, ClaimCheckMetadata{}, fmt.Errorf("failed to read payload from S3: %w", err)
		}

		return payload, meta, nil
	}
	s.mu.RUnlock()

	// Fetch from S3 (not in cache)
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.prefix + getTicketKey(ticket)),
	})
	if err != nil {
		return nil, ClaimCheckMetadata{}, fmt.Errorf("failed to retrieve payload from S3: %w", err)
	}
	defer result.Body.Close()

	payload, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, ClaimCheckMetadata{}, fmt.Errorf("failed to read payload from S3: %w", err)
	}

	// Reconstruct metadata from response
	metadata := ClaimCheckMetadata{
		ContentType: *result.ContentType,
		Size:        *result.ContentLength,
		Timestamp:   *result.LastModified,
	}
	if hash, ok := result.Metadata["hash"]; ok {
		metadata.Hash = hash
	}

	// Cache metadata
	s.mu.Lock()
	s.metadataCache[ticket] = metadata
	s.mu.Unlock()

	return payload, metadata, nil
}

// Delete removes a payload from S3
func (s *S3ClaimCheckStore) Delete(ctx context.Context, ticket string) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("store is closed")
	}
	s.mu.RUnlock()

	// Delete from S3
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.prefix + getTicketKey(ticket)),
	})
	if err != nil {
		return fmt.Errorf("failed to delete payload from S3: %w", err)
	}

	// Remove from cache
	s.mu.Lock()
	delete(s.metadataCache, ticket)
	s.mu.Unlock()

	return nil
}

// Verify checks the integrity of a stored payload
func (s *S3ClaimCheckStore) Verify(ctx context.Context, ticket string, expectedHash string) (bool, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return false, fmt.Errorf("store is closed")
	}
	s.mu.RUnlock()

	// Retrieve to get actual hash
	_, metadata, err := s.Retrieve(ctx, ticket)
	if err != nil {
		return false, err
	}

	return metadata.Hash == expectedHash, nil
}

// Close gracefully shuts down the S3 store
func (s *S3ClaimCheckStore) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.closed = true
	s.metadataCache = nil
	// AWS SDK v2 S3 client doesn't require explicit close

	return nil
}

// HealthCheck verifies S3 connectivity and creates bucket if needed
func (s *S3ClaimCheckStore) HealthCheck(ctx context.Context) error {
	if s.client == nil {
		return fmt.Errorf("S3 client not initialized")
	}
	if s.bucket == "" {
		return fmt.Errorf("S3 bucket not configured")
	}

	// Try to access the bucket
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})

	// If bucket doesn't exist (404), try to create it
	if err != nil {
		if contains(err.Error(), "NotFound") || contains(err.Error(), "404") {
			_, createErr := s.client.CreateBucket(ctx, &s3.CreateBucketInput{
				Bucket: aws.String(s.bucket),
			})
			if createErr != nil && !contains(createErr.Error(), "BucketAlreadyExists") && !contains(createErr.Error(), "409") {
				return fmt.Errorf("failed to create S3 bucket: %w", createErr)
			}
			return nil // Bucket now exists or already existed
		}
		return fmt.Errorf("failed to access S3 bucket: %w", err)
	}

	return nil
}

// contains checks if string s contains substring substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && bytes.Contains([]byte(s), []byte(substr))
}

// Helper function to extract key from ticket
func getTicketKey(ticket string) string {
	// Extract key portion from ticket (e.g., "s3://bucket/prefix/key" -> "key")
	// For simplicity, just use the timestamp and random ID that were generated
	// This is already in the right format from generateRandomID()
	for i := len(ticket) - 1; i >= 0; i-- {
		if ticket[i] == '/' {
			return ticket[i+1:]
		}
	}
	return ticket
}
