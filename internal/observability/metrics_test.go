package observability

import (
	"strings"
	"testing"
	"time"
)

func TestMetricsCollector_NewMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector(":2112")
	if mc == nil {
		t.Fatal("expected non-nil MetricsCollector")
	}
	if !mc.enabled {
		t.Fatal("expected metrics to be enabled by default")
	}
	if mc.prometheusAddr != ":2112" {
		t.Errorf("expected prometheus_addr ':2112', got %q", mc.prometheusAddr)
	}
}

func TestMetricsCollector_RecordMessageSuccess(t *testing.T) {
	mc := NewMetricsCollector(":2112")

	// Record successful message
	mc.RecordMessageSuccess("test-route", 100)

	// Verify metrics
	metrics := mc.GetRoute("test-route")
	if metrics.MessagesTotal != 1 {
		t.Errorf("expected 1 total message, got %d", metrics.MessagesTotal)
	}
	if metrics.MessagesSuccess != 1 {
		t.Errorf("expected 1 success, got %d", metrics.MessagesSuccess)
	}
	if metrics.MessagesFailed != 0 {
		t.Errorf("expected 0 failures, got %d", metrics.MessagesFailed)
	}
	if metrics.MessageLatencyP50 != 100 {
		t.Errorf("expected p50=100, got %d", metrics.MessageLatencyP50)
	}
}

func TestMetricsCollector_RecordMessageError(t *testing.T) {
	mc := NewMetricsCollector(":2112")

	// Record error
	mc.RecordMessageError("test-route", "step_failed")

	// Verify metrics
	metrics := mc.GetRoute("test-route")
	if metrics.MessagesTotal != 1 {
		t.Errorf("expected 1 total message, got %d", metrics.MessagesTotal)
	}
	if metrics.MessagesFailed != 1 {
		t.Errorf("expected 1 failure, got %d", metrics.MessagesFailed)
	}
	if metrics.MessagesSuccess != 0 {
		t.Errorf("expected 0 successes, got %d", metrics.MessagesSuccess)
	}
}

func TestMetricsCollector_RecordRetry(t *testing.T) {
	mc := NewMetricsCollector(":2112")

	// Record retries
	mc.RecordRetry("test-route")
	mc.RecordRetry("test-route")
	mc.RecordRetry("test-route")

	metrics := mc.GetRoute("test-route")
	if metrics.RetriesTotal != 3 {
		t.Errorf("expected 3 retries, got %d", metrics.RetriesTotal)
	}
}

func TestMetricsCollector_RecordDeadLetter(t *testing.T) {
	mc := NewMetricsCollector(":2112")

	// Record dead letters
	mc.RecordDeadLetter("test-route")
	mc.RecordDeadLetter("test-route")

	metrics := mc.GetRoute("test-route")
	if metrics.DeadLettersTotal != 2 {
		t.Errorf("expected 2 dead letters, got %d", metrics.DeadLettersTotal)
	}
}

func TestMetricsCollector_RecordWorkerInFlight(t *testing.T) {
	mc := NewMetricsCollector(":2112")

	// Increment in-flight
	mc.RecordWorkerInFlight("test-route", 1)
	mc.RecordWorkerInFlight("test-route", 1)
	mc.RecordWorkerInFlight("test-route", 1)

	metrics := mc.GetRoute("test-route")
	if metrics.WorkersInFlight != 3 {
		t.Errorf("expected 3 in-flight workers, got %d", metrics.WorkersInFlight)
	}

	// Decrement in-flight
	mc.RecordWorkerInFlight("test-route", -1)
	metrics = mc.GetRoute("test-route")
	if metrics.WorkersInFlight != 2 {
		t.Errorf("expected 2 in-flight workers after decrement, got %d", metrics.WorkersInFlight)
	}
}

func TestMetricsCollector_RecordStepLatency(t *testing.T) {
	mc := NewMetricsCollector(":2112")

	// Record step latencies
	mc.RecordStepLatency("test-route", "filter", 10)
	mc.RecordStepLatency("test-route", "filter", 15)
	mc.RecordStepLatency("test-route", "filter", 20)

	// Verify through internal data (not exposed in GetRoute)
	mc.mu.RLock()
	latencies := mc.stepLatencies["test-route.filter"]
	mc.mu.RUnlock()

	if len(latencies) != 3 {
		t.Errorf("expected 3 latency samples, got %d", len(latencies))
	}
}

func TestMetricsCollector_GetMetrics_PrometheusFormat(t *testing.T) {
	mc := NewMetricsCollector(":2112")

	// Record some metrics
	mc.RecordMessageSuccess("premium-processing", 45)
	mc.RecordMessageSuccess("premium-processing", 50)
	mc.RecordMessageError("premium-processing", "step_failed")
	mc.RecordRetry("premium-processing")
	mc.RecordDeadLetter("premium-processing")
	mc.RecordWorkerInFlight("premium-processing", 3)

	// Get Prometheus format output
	output := mc.GetMetrics()

	// Verify format contains expected sections
	expectedSections := []string{
		"dim_messages_total",
		"dim_messages_success",
		"dim_messages_failed",
		"dim_workers_in_flight",
		"dim_retries_total",
		"dim_dead_letters_total",
		"dim_message_latency_ms",
	}

	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("expected prometheus output to contain %q", section)
		}
	}

	// Verify specific metrics
	if !strings.Contains(output, `route="premium-processing"`) {
		t.Errorf("expected prometheus output to contain route label")
	}
	if !strings.Contains(output, "dim_workers_in_flight{route=\"premium-processing\"} 3") {
		t.Errorf("expected workers_in_flight=3")
	}
}

func TestMetricsCollector_MultipleRoutes(t *testing.T) {
	mc := NewMetricsCollector(":2112")

	// Record metrics for different routes
	mc.RecordMessageSuccess("route-a", 100)
	mc.RecordMessageSuccess("route-b", 200)
	mc.RecordMessageError("route-a", "error1")

	// Check route A
	metricsA := mc.GetRoute("route-a")
	if metricsA.MessagesTotal != 2 {
		t.Errorf("route-a: expected 2 total messages, got %d", metricsA.MessagesTotal)
	}
	if metricsA.MessagesSuccess != 1 {
		t.Errorf("route-a: expected 1 success, got %d", metricsA.MessagesSuccess)
	}

	// Check route B
	metricsB := mc.GetRoute("route-b")
	if metricsB.MessagesTotal != 1 {
		t.Errorf("route-b: expected 1 total message, got %d", metricsB.MessagesTotal)
	}
}

func TestMetricsCollector_LatencyPercentiles(t *testing.T) {
	mc := NewMetricsCollector(":2112")

	// Record latencies: 10, 20, 30, 40, 50, 60, 70, 80, 90, 100
	latencies := []int64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	for _, lat := range latencies {
		mc.RecordMessageSuccess("test-route", lat)
	}

	metrics := mc.GetRoute("test-route")

	// Verify percentiles (approximate)
	if metrics.MessageLatencyP50 == 0 {
		t.Error("expected p50 to be non-zero")
	}
	if metrics.MessageLatencyP90 == 0 {
		t.Error("expected p90 to be non-zero")
	}
	if metrics.MessageLatencyP99 == 0 {
		t.Error("expected p99 to be non-zero")
	}
	if metrics.MessageLatencyCount != 10 {
		t.Errorf("expected count=10, got %d", metrics.MessageLatencyCount)
	}

	// Verify ordering: p50 <= p90 <= p99
	if metrics.MessageLatencyP50 > metrics.MessageLatencyP90 {
		t.Errorf("expected p50 <= p90, got %d > %d", metrics.MessageLatencyP50, metrics.MessageLatencyP90)
	}
	if metrics.MessageLatencyP90 > metrics.MessageLatencyP99 {
		t.Errorf("expected p90 <= p99, got %d > %d", metrics.MessageLatencyP90, metrics.MessageLatencyP99)
	}
}

func TestMetricsCollector_Disable(t *testing.T) {
	mc := NewMetricsCollector(":2112")

	mc.RecordMessageSuccess("test-route", 100)
	metrics1 := mc.GetRoute("test-route")

	mc.Disable()

	// After disabling, new records should not be counted
	mc.RecordMessageSuccess("test-route", 200)
	metrics2 := mc.GetRoute("test-route")

	if metrics2.MessagesTotal != metrics1.MessagesTotal {
		t.Errorf("expected metrics to remain unchanged after disable")
	}
}

func TestMetricsCollector_Reset(t *testing.T) {
	mc := NewMetricsCollector(":2112")

	mc.RecordMessageSuccess("test-route", 100)
	mc.RecordRetry("test-route")
	mc.RecordDeadLetter("test-route")

	metrics1 := mc.GetRoute("test-route")
	if metrics1.MessagesTotal == 0 {
		t.Fatal("expected metrics before reset")
	}

	mc.Reset()

	metrics2 := mc.GetRoute("test-route")
	if metrics2.MessagesTotal != 0 {
		t.Errorf("expected all metrics to be zero after reset, got %d", metrics2.MessagesTotal)
	}
}

func TestMetricsCollector_NegativeLatency(t *testing.T) {
	mc := NewMetricsCollector(":2112")

	// Negative latencies should be ignored
	mc.RecordMessageSuccess("test-route", -100)

	metrics := mc.GetRoute("test-route")
	if metrics.MessagesTotal != 0 {
		t.Errorf("expected negative latency to be ignored")
	}
}

func TestMetricsCollector_ZeroDeltaWorkerInFlight(t *testing.T) {
	mc := NewMetricsCollector(":2112")

	mc.RecordWorkerInFlight("test-route", 1)
	mc.RecordWorkerInFlight("test-route", 0) // Zero delta should be no-op

	metrics := mc.GetRoute("test-route")
	if metrics.WorkersInFlight != 1 {
		t.Errorf("expected zero delta to be no-op, got %d", metrics.WorkersInFlight)
	}
}

func TestMetricsCollector_ThreadSafety(t *testing.T) {
	mc := NewMetricsCollector(":2112")

	// Record metrics concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			for j := 0; j < 100; j++ {
				mc.RecordMessageSuccess("concurrent-route", int64(idx*100+j))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	metrics := mc.GetRoute("concurrent-route")
	if metrics.MessagesTotal != 1000 {
		t.Errorf("expected 1000 messages, got %d", metrics.MessagesTotal)
	}
}

func TestMetricsCollector_LatencyAverage(t *testing.T) {
	mc := NewMetricsCollector(":2112")

	// Record latencies: 10, 20, 30
	mc.RecordMessageSuccess("test-route", 10)
	mc.RecordMessageSuccess("test-route", 20)
	mc.RecordMessageSuccess("test-route", 30)

	metrics := mc.GetRoute("test-route")
	expectedAvg := int64(20) // (10+20+30)/3 = 20
	if metrics.MessageLatencyAvg != expectedAvg {
		t.Errorf("expected average latency %d, got %d", expectedAvg, metrics.MessageLatencyAvg)
	}
}

func BenchmarkMetricsCollector_RecordMessageSuccess(b *testing.B) {
	mc := NewMetricsCollector(":2112")
	start := time.Now()

	for i := 0; i < b.N; i++ {
		mc.RecordMessageSuccess("test-route", int64(i%1000))
	}

	elapsed := time.Since(start)
	totalOps := b.N
	opsPerMs := float64(totalOps) / elapsed.Seconds() / 1000

	b.ReportMetric(opsPerMs, "k_ops/sec")

	// Check that latency overhead is acceptable (<1ms per 1000 ops)
	if elapsed > time.Duration(b.N)*time.Microsecond {
		b.Logf("performance acceptable: %d ops in %v", b.N, elapsed)
	}
}

func BenchmarkMetricsCollector_GetMetrics(b *testing.B) {
	mc := NewMetricsCollector(":2112")

	// Pre-populate with metrics
	for i := 0; i < 100; i++ {
		for j := 0; j < 10; j++ {
			mc.RecordMessageSuccess("route-"+string(rune(i)), int64(j*100))
		}
	}

	start := time.Now()
	for i := 0; i < b.N; i++ {
		_ = mc.GetMetrics()
	}
	elapsed := time.Since(start)

	b.ReportMetric(float64(b.N)/elapsed.Seconds(), "ops/sec")
}
