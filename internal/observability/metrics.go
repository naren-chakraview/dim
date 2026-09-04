package observability

import (
	"fmt"
	"log"
	"sync"
)

// MetricsCollector tracks Prometheus metrics for the middleware.
// It records counters, gauges, and histograms for message processing.
// All operations are thread-safe.
type MetricsCollector struct {
	mu                 sync.RWMutex
	enabled            bool
	prometheusAddr     string

	// Counters: route -> count
	messagesTotal map[string]int64
	messagesSuccess map[string]int64
	messagesFailed map[string]int64
	retriesTotal map[string]int64
	deadLettersTotal map[string]int64

	// Gauges: route -> value
	workersInFlight map[string]int32

	// Histograms: route -> list of latencies (in milliseconds)
	// We store raw samples and compute percentiles on demand
	messageLatencies map[string][]int64
	stepLatencies map[string][]int64

	// Error types tracking: route -> error_type -> count
	errorsByType map[string]map[string]int64
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(prometheusAddr string) *MetricsCollector {
	return &MetricsCollector{
		enabled:          true,
		prometheusAddr:   prometheusAddr,
		messagesTotal:    make(map[string]int64),
		messagesSuccess:  make(map[string]int64),
		messagesFailed:   make(map[string]int64),
		retriesTotal:     make(map[string]int64),
		deadLettersTotal: make(map[string]int64),
		workersInFlight:  make(map[string]int32),
		messageLatencies: make(map[string][]int64),
		stepLatencies:    make(map[string][]int64),
		errorsByType:     make(map[string]map[string]int64),
	}
}

// RecordMessageSuccess records a successful message processing
func (mc *MetricsCollector) RecordMessageSuccess(route string, latencyMs int64) {
	if !mc.enabled || latencyMs < 0 {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Increment total and success counters
	mc.messagesTotal[route]++
	mc.messagesSuccess[route]++

	// Record latency
	if _, exists := mc.messageLatencies[route]; !exists {
		mc.messageLatencies[route] = make([]int64, 0)
	}
	mc.messageLatencies[route] = append(mc.messageLatencies[route], latencyMs)
}

// RecordMessageError records a failed message processing
func (mc *MetricsCollector) RecordMessageError(route string, errorType string) {
	if !mc.enabled {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Increment total and failed counters
	mc.messagesTotal[route]++
	mc.messagesFailed[route]++

	// Record error type
	if _, exists := mc.errorsByType[route]; !exists {
		mc.errorsByType[route] = make(map[string]int64)
	}
	mc.errorsByType[route][errorType]++
}

// RecordRetry records a retry attempt for a message
func (mc *MetricsCollector) RecordRetry(route string) {
	if !mc.enabled {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.retriesTotal[route]++
}

// RecordDeadLetter records a dead-letter envelope
func (mc *MetricsCollector) RecordDeadLetter(route string) {
	if !mc.enabled {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.deadLettersTotal[route]++
}

// RecordWorkerInFlight adjusts the in-flight worker count
// delta: positive to increment, negative to decrement
func (mc *MetricsCollector) RecordWorkerInFlight(route string, delta int32) {
	if !mc.enabled || delta == 0 {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	current := mc.workersInFlight[route]
	mc.workersInFlight[route] = current + delta
}

// RecordStepLatency records the latency of a single step execution
func (mc *MetricsCollector) RecordStepLatency(route string, stepName string, latencyMs int64) {
	if !mc.enabled || latencyMs < 0 {
		return
	}

	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := fmt.Sprintf("%s.%s", route, stepName)
	if _, exists := mc.stepLatencies[key]; !exists {
		mc.stepLatencies[key] = make([]int64, 0)
	}
	mc.stepLatencies[key] = append(mc.stepLatencies[key], latencyMs)
}

// GetMetrics returns a snapshot of all metrics in Prometheus text format
func (mc *MetricsCollector) GetMetrics() string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var output string

	// Write counters: messages_total
	if len(mc.messagesTotal) > 0 {
		output += "# HELP dim_messages_total Total number of messages processed\n"
		output += "# TYPE dim_messages_total counter\n"
		for route, count := range mc.messagesTotal {
			output += fmt.Sprintf("dim_messages_total{route=\"%s\"} %d\n", route, count)
		}
		output += "\n"
	}

	// Write counters: messages_success
	if len(mc.messagesSuccess) > 0 {
		output += "# HELP dim_messages_success Total number of successfully processed messages\n"
		output += "# TYPE dim_messages_success counter\n"
		for route, count := range mc.messagesSuccess {
			output += fmt.Sprintf("dim_messages_success{route=\"%s\"} %d\n", route, count)
		}
		output += "\n"
	}

	// Write counters: messages_failed
	if len(mc.messagesFailed) > 0 {
		output += "# HELP dim_messages_failed Total number of failed messages\n"
		output += "# TYPE dim_messages_failed counter\n"
		for route, count := range mc.messagesFailed {
			output += fmt.Sprintf("dim_messages_failed{route=\"%s\"} %d\n", route, count)
		}
		output += "\n"
	}

	// Write gauges: workers_in_flight
	if len(mc.workersInFlight) > 0 {
		output += "# HELP dim_workers_in_flight Number of messages currently being processed\n"
		output += "# TYPE dim_workers_in_flight gauge\n"
		for route, count := range mc.workersInFlight {
			output += fmt.Sprintf("dim_workers_in_flight{route=\"%s\"} %d\n", route, count)
		}
		output += "\n"
	}

	// Write counters: retries_total
	if len(mc.retriesTotal) > 0 {
		output += "# HELP dim_retries_total Total number of retry attempts\n"
		output += "# TYPE dim_retries_total counter\n"
		for route, count := range mc.retriesTotal {
			output += fmt.Sprintf("dim_retries_total{route=\"%s\"} %d\n", route, count)
		}
		output += "\n"
	}

	// Write counters: dead_letters_total
	if len(mc.deadLettersTotal) > 0 {
		output += "# HELP dim_dead_letters_total Total number of dead-letter envelopes\n"
		output += "# TYPE dim_dead_letters_total counter\n"
		for route, count := range mc.deadLettersTotal {
			output += fmt.Sprintf("dim_dead_letters_total{route=\"%s\"} %d\n", route, count)
		}
		output += "\n"
	}

	// Write summaries: message_latency_ms with quantiles
	// Fixed from histogram to summary (matches the quantile-based output format)
	if len(mc.messageLatencies) > 0 {
		output += "# HELP dim_message_latency_ms Message processing latency in milliseconds\n"
		output += "# TYPE dim_message_latency_ms summary\n"
		for route, latencies := range mc.messageLatencies {
			if len(latencies) > 0 {
				p50 := getPercentile(latencies, 0.50)
				p90 := getPercentile(latencies, 0.90)
				p99 := getPercentile(latencies, 0.99)
				sum := int64(0)
				for _, l := range latencies {
					sum += l
				}

				output += fmt.Sprintf("dim_message_latency_ms{route=\"%s\",quantile=\"0.5\"} %d\n", route, p50)
				output += fmt.Sprintf("dim_message_latency_ms{route=\"%s\",quantile=\"0.9\"} %d\n", route, p90)
				output += fmt.Sprintf("dim_message_latency_ms{route=\"%s\",quantile=\"0.99\"} %d\n", route, p99)
				output += fmt.Sprintf("dim_message_latency_ms_sum{route=\"%s\"} %d\n", route, sum)
				output += fmt.Sprintf("dim_message_latency_ms_count{route=\"%s\"} %d\n", route, len(latencies))
			}
		}
		output += "\n"
	}

	// Write error types
	if len(mc.errorsByType) > 0 {
		output += "# HELP dim_errors_total Total number of errors by type\n"
		output += "# TYPE dim_errors_total counter\n"
		for route, errors := range mc.errorsByType {
			for errorType, count := range errors {
				output += fmt.Sprintf("dim_errors_total{route=\"%s\",error_type=\"%s\"} %d\n", route, errorType, count)
			}
		}
		output += "\n"
	}

	return output
}

// GetRoute returns metrics for a specific route
func (mc *MetricsCollector) GetRoute(route string) RouteMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics := RouteMetrics{
		Route:            route,
		MessagesTotal:    mc.messagesTotal[route],
		MessagesSuccess:  mc.messagesSuccess[route],
		MessagesFailed:   mc.messagesFailed[route],
		RetriesTotal:     mc.retriesTotal[route],
		DeadLettersTotal: mc.deadLettersTotal[route],
		WorkersInFlight:  mc.workersInFlight[route],
	}

	// Compute latency percentiles
	if latencies, exists := mc.messageLatencies[route]; exists && len(latencies) > 0 {
		metrics.MessageLatencyP50 = getPercentile(latencies, 0.50)
		metrics.MessageLatencyP90 = getPercentile(latencies, 0.90)
		metrics.MessageLatencyP99 = getPercentile(latencies, 0.99)
		sum := int64(0)
		for _, l := range latencies {
			sum += l
		}
		metrics.MessageLatencySum = sum
		metrics.MessageLatencyCount = int64(len(latencies))
		if metrics.MessageLatencyCount > 0 {
			metrics.MessageLatencyAvg = sum / metrics.MessageLatencyCount
		}
	}

	return metrics
}

// RouteMetrics holds aggregated metrics for a single route
type RouteMetrics struct {
	Route              string
	MessagesTotal      int64
	MessagesSuccess    int64
	MessagesFailed     int64
	RetriesTotal       int64
	DeadLettersTotal   int64
	WorkersInFlight    int32
	MessageLatencyP50  int64
	MessageLatencyP90  int64
	MessageLatencyP99  int64
	MessageLatencySum  int64
	MessageLatencyCount int64
	MessageLatencyAvg  int64
}

// getPercentile computes the p-th percentile of a sorted list
func getPercentile(values []int64, p float64) int64 {
	if len(values) == 0 {
		return 0
	}

	// Copy and sort
	sorted := make([]int64, len(values))
	copy(sorted, values)
	// Simple bubble sort for small datasets
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Compute percentile index
	idx := int(float64(len(sorted)-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}

	return sorted[idx]
}

// Disable disables metrics collection
func (mc *MetricsCollector) Disable() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.enabled = false
	log.Println("[DEBUG] metrics collection disabled")
}

// Reset clears all metrics (for testing)
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.messagesTotal = make(map[string]int64)
	mc.messagesSuccess = make(map[string]int64)
	mc.messagesFailed = make(map[string]int64)
	mc.retriesTotal = make(map[string]int64)
	mc.deadLettersTotal = make(map[string]int64)
	mc.workersInFlight = make(map[string]int32)
	mc.messageLatencies = make(map[string][]int64)
	mc.stepLatencies = make(map[string][]int64)
	mc.errorsByType = make(map[string]map[string]int64)
}
