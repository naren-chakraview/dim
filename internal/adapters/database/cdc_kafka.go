package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/segmentio/kafka-go"
	"github.com/naren-chakraview/dim/internal/engine"
)

// LogBasedCDCSource consumes CDC events from Kafka topics (M2.3.4b)
// Supports Debezium and Maxwell formatted CDC records
type LogBasedCDCSource struct {
	reader    *kafka.Reader
	config    *LogBasedCDCConfig
	outChan   *engine.Channel
	closed    chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
	committed int64
}

// LogBasedCDCConfig defines configuration for Kafka-fed CDC
type LogBasedCDCConfig struct {
	Brokers       []string // Kafka broker addresses
	Topic         string   // CDC topic name (e.g., debezium.public.users)
	ConsumerGroup string   // Consumer group ID
	CDCFormat     string   // "debezium", "maxwell", etc.
	StartOffset   string   // "latest" or "earliest"
	BatchSize     int      // Messages per batch
	TimeoutSec    int      // Read timeout
}

// DebeziumEvent represents a Debezium CDC record
type DebeziumEvent struct {
	Before    map[string]interface{} `json:"before"`
	After     map[string]interface{} `json:"after"`
	Source    Source                 `json:"source"`
	Op        string                 `json:"op"`
	Timestamp int64                  `json:"ts_ms"`
}

// Source contains Debezium source metadata
type Source struct {
	Table string `json:"table"`
}

// NewLogBasedCDCSource creates a new log-based CDC source consuming from Kafka
func NewLogBasedCDCSource(config *LogBasedCDCConfig, outChan *engine.Channel) (*LogBasedCDCSource, error) {
	if len(config.Brokers) == 0 {
		return nil, fmt.Errorf("Brokers is required")
	}
	if config.Topic == "" {
		return nil, fmt.Errorf("Topic is required")
	}
	if config.ConsumerGroup == "" {
		return nil, fmt.Errorf("ConsumerGroup is required")
	}

	if config.CDCFormat == "" {
		config.CDCFormat = "debezium"
	}
	if config.StartOffset == "" {
		config.StartOffset = "latest"
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.TimeoutSec == 0 {
		config.TimeoutSec = 30
	}

	// Determine start offset
	startOffset := kafka.LastOffset
	if config.StartOffset == "earliest" {
		startOffset = kafka.FirstOffset
	}

	// Create Kafka reader
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:         config.Brokers,
		Topic:           config.Topic,
		GroupID:         config.ConsumerGroup,
		StartOffset:     startOffset,
		CommitInterval:  10,
		MaxBytes:        10e6,
		PartitionWatchInterval: 30,
	})

	source := &LogBasedCDCSource{
		reader:  reader,
		config:  config,
		outChan: outChan,
		closed:  make(chan struct{}),
	}

	log.Printf("[INFO] Log-based CDC source created: topic=%s brokers=%v format=%s", config.Topic, config.Brokers, config.CDCFormat)

	return source, nil
}

// Start begins consuming CDC events from Kafka
func (s *LogBasedCDCSource) Start(ctx context.Context) error {
	s.wg.Add(1)
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			close(s.closed)
			return ctx.Err()
		default:
		}

		msg, err := s.reader.FetchMessage(ctx)
		if err != nil {
			if err == context.Canceled {
				return err
			}
			log.Printf("[WARN] Kafka fetch failed: %v", err)
			continue
		}

		// Parse CDC record based on format
		cdcEvent, err := s.parseRecord(msg.Value)
		if err != nil {
			log.Printf("[WARN] Failed to parse CDC record: %v", err)
			continue
		}

		// Create message
		outMsg := engine.NewMessage(cdcEvent, "database-cdc-kafka", "")
		outMsg.Metadata.Stage = fmt.Sprintf("cdc_kafka topic=%s partition=%d offset=%d", s.config.Topic, msg.Partition, msg.Offset)

		// Send to output channel
		if err := s.outChan.Send(ctx, outMsg); err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}

		// Commit offset
		if err := s.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("[WARN] Failed to commit offset: %v", err)
		}

		s.mu.Lock()
		s.committed++
		s.mu.Unlock()
	}
}

// parseRecord parses CDC record based on configured format
func (s *LogBasedCDCSource) parseRecord(data []byte) (map[string]interface{}, error) {
	switch s.config.CDCFormat {
	case "debezium":
		return s.parseDebezium(data)
	case "maxwell":
		return s.parseMaxwell(data)
	default:
		return nil, fmt.Errorf("unsupported CDC format: %s", s.config.CDCFormat)
	}
}

// parseDebezium parses a Debezium CDC record
func (s *LogBasedCDCSource) parseDebezium(data []byte) (map[string]interface{}, error) {
	var event DebeziumEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Debezium event: %w", err)
	}

	// Map Debezium operations to standard names
	operation := event.Op
	switch event.Op {
	case "c":
		operation = "INSERT"
	case "u":
		operation = "UPDATE"
	case "d":
		operation = "DELETE"
	case "r":
		operation = "READ"
	}

	cdcEvent := map[string]interface{}{
		"table":      event.Source.Table,
		"operation":  operation,
		"timestamp":  event.Timestamp,
	}

	// Include before state for UPDATE/DELETE
	if event.Before != nil && len(event.Before) > 0 {
		beforeJSON, _ := json.Marshal(event.Before)
		cdcEvent["before"] = string(beforeJSON)
	}

	// Include after state for INSERT/UPDATE
	if event.After != nil && len(event.After) > 0 {
		afterJSON, _ := json.Marshal(event.After)
		cdcEvent["after"] = string(afterJSON)
	}

	return cdcEvent, nil
}

// parseMaxwell parses a Maxwell CDC record
func (s *LogBasedCDCSource) parseMaxwell(data []byte) (map[string]interface{}, error) {
	var rawEvent map[string]interface{}
	if err := json.Unmarshal(data, &rawEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Maxwell event: %w", err)
	}

	cdcEvent := map[string]interface{}{
		"table":      rawEvent["table"],
		"operation":  rawEvent["type"],
	}

	// Maxwell format includes data and old fields
	if data, ok := rawEvent["data"]; ok {
		dataJSON, _ := json.Marshal(data)
		cdcEvent["after"] = string(dataJSON)
	}
	if old, ok := rawEvent["old"]; ok {
		oldJSON, _ := json.Marshal(old)
		cdcEvent["before"] = string(oldJSON)
	}

	if ts, ok := rawEvent["ts"]; ok {
		cdcEvent["timestamp"] = ts
	}

	return cdcEvent, nil
}

// Close closes the CDC source and Kafka reader
func (s *LogBasedCDCSource) Close() error {
	select {
	case <-s.closed:
	default:
	}
	s.wg.Wait()

	if s.reader != nil {
		return s.reader.Close()
	}
	return nil
}

// GetCommittedCount returns total CDC records consumed
func (s *LogBasedCDCSource) GetCommittedCount() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.committed
}

// HealthCheck verifies Kafka connection
func (s *LogBasedCDCSource) HealthCheck(ctx context.Context) error {
	_, err := s.reader.FetchMessage(ctx)
	if err == context.Canceled || err == context.DeadlineExceeded {
		return err
	}
	return nil
}
