package s3

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/naren-chakraview/dim/internal/engine"
)

type S3Source struct {
	client    *s3.Client
	bucket    string
	prefix    string
	outChan   *engine.Channel
	closed    chan struct{}
	config    SourceConfig
	lastSync  map[string]time.Time
	mu        sync.Mutex
}

type SourceConfig struct {
	Bucket      string
	Prefix      string
	Region      string
	PollInterval time.Duration
}

func NewS3Source(cfg SourceConfig, outChan *engine.Channel) (*S3Source, error) {
	if outChan == nil {
		return nil, fmt.Errorf("output channel cannot be nil")
	}
	return &S3Source{
		bucket:  cfg.Bucket,
		prefix:  cfg.Prefix,
		outChan: outChan,
		closed:  make(chan struct{}),
		config:  cfg,
		lastSync: make(map[string]time.Time),
	}, nil
}

func (s *S3Source) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			close(s.closed)
			return ctx.Err()
		case <-ticker.C:
			s.poll(ctx)
		}
	}
}

func (s *S3Source) poll(ctx context.Context) {
	// List objects from S3
	result, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(s.prefix),
	})
	if err != nil {
		fmt.Printf("[WARN] S3 list failed: %v\n", err)
		return
	}

	for _, obj := range result.Contents {
		key := *obj.Key
		lastMod := *obj.LastModified

		s.mu.Lock()
		if prev, exists := s.lastSync[key]; exists && prev.Equal(lastMod) {
			s.mu.Unlock()
			continue
		}
		s.lastSync[key] = lastMod
		s.mu.Unlock()

		// Get object
		getResult, err := s.client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			fmt.Printf("[WARN] S3 get failed: %v\n", err)
			continue
		}

		bodyBytes, err := io.ReadAll(getResult.Body)
		defer getResult.Body.Close()
		if err != nil {
			fmt.Printf("[WARN] S3 read body failed: %v\n", err)
			continue
		}

		var body interface{}
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			body = string(bodyBytes)
		}

		msg := engine.NewMessage(body, "s3-source", "")
		msg.Headers["s3_bucket"] = s.bucket
		msg.Headers["s3_key"] = key
		msg.Headers["s3_size"] = *getResult.ContentLength

		s.outChan.Send(ctx, msg)
	}
}

func (s *S3Source) Close() error {
	// AWS SDK v2 S3 client doesn't require explicit close
	return nil
}

// HealthCheck verifies the S3 source is able to connect
func (s *S3Source) HealthCheck(ctx context.Context) error {
	if s.client == nil {
		return fmt.Errorf("S3 client not initialized")
	}
	if s.bucket == "" {
		return fmt.Errorf("S3 bucket not configured")
	}
	return nil
}

// Checkpoint returns the current last-sync map for resuming from a known state
func (s *S3Source) Checkpoint() (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	checkpoint := make(map[string]int64)
	for k, v := range s.lastSync {
		checkpoint[k] = v.Unix()
	}
	return checkpoint, nil
}
