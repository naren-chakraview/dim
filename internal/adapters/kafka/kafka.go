package kafka

// Kafka adapter package provides:
// - KafkaSource: Consumer group-based message ingestion from Kafka topics
// - KafkaSink: Producer-based message output to Kafka topics
// - Offset tracking and consumer group management
// - Phase 1 implementation using Shopify/sarama or segmentio/kafka-go
