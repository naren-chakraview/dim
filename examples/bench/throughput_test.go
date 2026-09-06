package bench

import (
	"context"
	"testing"

	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/steps"
)

// BenchmarkMessageThroughput measures messages/sec through full pipeline
// Single route, 3 steps (filter, translate, authorize)
func BenchmarkMessageThroughput(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Build a simple 3-step pipeline
	inputCh := engine.NewChannel("benchmark-input", 1000)
	outputCh := engine.NewChannel("benchmark-output", 1000)

	// Create steps
	filterStep, _ := steps.NewFilterStep("true")  // Always pass
	translateStep, _ := steps.NewTranslateStep("{\"id\": body.id, \"ts\": $now()}")
	authorizeStep, _ := steps.NewAuthorizeStep(
		"rbac",
		[]string{},
		"",
		"",
		5000,
	)

	stepInstances := []engine.Step{filterStep, translateStep, authorizeStep}
	stepNames := []string{"filter", "translate", "authorize"}

	executor := engine.NewExecutorWithWorkers("bench-route", inputCh, outputCh, nil, stepInstances, stepNames, 4)

	// Run executor in background
	executorErr := make(chan error, 1)
	go func() {
		executorErr <- executor.Run(ctx)
	}()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			msg := engine.NewMessage(
				map[string]interface{}{"id": i, "value": "test"},
				"bench-route",
				"v1",
			)

			inputCh.Send(ctx, msg)

			// Wait for result with timeout
			result, err := outputCh.Recv(ctx)
			if err != nil {
				b.Errorf("recv error: %v", err)
			} else if result == nil {
				b.Errorf("nil result received")
			}
			i++
		}
	})

	b.StopTimer()
	cancel()

	// Wait for executor to finish
	<-executorErr

	// Calculate throughput
	throughput := float64(b.N) / b.Elapsed().Seconds()
	b.ReportMetric(throughput, "msg/sec")
}

// BenchmarkStepExecution measures latency of individual steps
func BenchmarkStepExecution(b *testing.B) {
	ctx := context.Background()

	b.Run("Filter", func(b *testing.B) {
		filterStep, _ := steps.NewFilterStep("body.amount > 100")
		msg := engine.NewMessage(
			map[string]interface{}{"amount": 150},
			"bench",
			"v1",
		)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := filterStep.Execute(ctx, msg)
			if err != nil {
				b.Errorf("filter error: %v", err)
			}
		}
	})

	b.Run("Translate", func(b *testing.B) {
		translateStep, _ := steps.NewTranslateStep("{\"id\": body.id, \"processed\": true}")
		msg := engine.NewMessage(
			map[string]interface{}{"id": "test-123"},
			"bench",
			"v1",
		)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := translateStep.Execute(ctx, msg)
			if err != nil {
				b.Errorf("translate error: %v", err)
			}
		}
	})

	b.Run("Authorize", func(b *testing.B) {
		authorizeStep, _ := steps.NewAuthorizeStep(
			"rbac",
			[]string{},
			"",
			"",
			5000,
		)
		msg := engine.NewMessage(
			map[string]interface{}{},
			"bench",
			"v1",
		)
		msg.Metadata.Principal = &engine.Principal{Subject: "user-123", Roles: []string{"admin"}}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := authorizeStep.Execute(ctx, msg)
			if err != nil {
				b.Errorf("authorize error: %v", err)
			}
		}
	})

	b.Run("Idempotent", func(b *testing.B) {
		idempotentStep, _ := steps.NewIdempotentStep("body.transaction_id", 60)
		msg := engine.NewMessage(
			map[string]interface{}{"transaction_id": "txn-123"},
			"bench",
			"v1",
		)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := idempotentStep.Execute(ctx, msg)
			if err != nil {
				b.Errorf("idempotent error: %v", err)
			}
		}
	})
}

// BenchmarkChannelOperations measures channel send/recv overhead
func BenchmarkChannelOperations(b *testing.B) {
	ctx := context.Background()

	b.Run("Send", func(b *testing.B) {
		ch := engine.NewChannel("bench", 1000)
		msg := engine.NewMessage(map[string]interface{}{}, "bench", "v1")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ch.Send(ctx, msg)
		}
	})

	b.Run("Recv", func(b *testing.B) {
		ch := engine.NewChannel("bench", 1000)
		msg := engine.NewMessage(map[string]interface{}{}, "bench", "v1")

		// Pre-fill channel
		for i := 0; i < b.N; i++ {
			ch.Send(ctx, msg)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ch.Recv(ctx)
		}
	})

	b.Run("SendRecv", func(b *testing.B) {
		ch := engine.NewChannel("bench", 100)
		msg := engine.NewMessage(map[string]interface{}{}, "bench", "v1")

		go func() {
			for i := 0; i < b.N; i++ {
				ch.Recv(ctx)
			}
		}()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ch.Send(ctx, msg)
		}
	})
}

// BenchmarkMessageCloning measures message deep copy overhead
func BenchmarkMessageCloning(b *testing.B) {
	original := engine.NewMessage(
		map[string]interface{}{
			"id":    "test-123",
			"amount": 1000.50,
			"items": []string{"a", "b", "c"},
		},
		"bench",
		"v1",
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = original.Clone()
	}
}
