package s3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/naren-chakraview/dim/internal/engine"
)

type S3Sink struct {
	client *s3.Client
	bucket string
	prefix string
	mu     sync.Mutex
	closed bool
	metrics struct {
		written int64
		failed  int64
	}
}

type SinkConfig struct {
	Bucket string
	Prefix string
	Region string
}

func NewS3Sink(cfg SinkConfig) (*S3Sink, error) {
	return &S3Sink{
		bucket: cfg.Bucket,
		prefix: cfg.Prefix,
	}, nil
}

func (s *S3Sink) Write(ctx context.Context, messages ...*engine.Message) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("sink is closed")
	}
	s.mu.Unlock()

	for i, msg := range messages {
		var body []byte
		switch v := msg.Body.(type) {
		case string:
			body = []byte(v)
		case []byte:
			body = v
		default:
			var err error
			body, err = json.Marshal(msg.Body)
			if err != nil {
				s.mu.Lock()
				s.metrics.failed++
				s.mu.Unlock()
				return fmt.Errorf("marshal failed: %w", err)
			}
		}

		key := fmt.Sprintf("%s/%d.json", s.prefix, i)

		_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
			Bucket:      aws.String(s.bucket),
			Key:         aws.String(key),
			Body:        bytes.NewReader(body),
			ContentType: aws.String("application/json"),
		})
		if err != nil {
			s.mu.Lock()
			s.metrics.failed++
			s.mu.Unlock()
			return fmt.Errorf("put failed: %w", err)
		}

		s.mu.Lock()
		s.metrics.written++
		s.mu.Unlock()
	}

	return nil
}

func (s *S3Sink) Start(ctx context.Context) error {
	return nil
}

func (s *S3Sink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	if s.client != nil {
		s.client.Close()
	}
	return nil
}
