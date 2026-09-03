package bench

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
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
		config.AuthorizeSpec{
			Mode:       "rbac",
			Required:   false,
			AllowRoles: []string{"*"},
		},
	)

	stepInstances := []steps.Step{filterStep, translateStep, authorizeStep}
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
			msg := engine.NewMessage()
			msg.ID = fmt.Sprintf("msg-%d", i)
			msg.Body = map[string]interface{}{"id": i, "value": "test"}

			inputCh.Send(msg)

			// Wait for result with timeout
			select {
			case result := <-outputCh.Recv():
				if result == nil {
					b.Errorf("nil result received")
				}
			case <-time.After(5 * time.Second):
				b.Errorf("message processing timeout")
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
		msg := engine.NewMessage()
		msg.Body = map[string]interface{}{"amount": 150}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := filterStep.Process(ctx, msg)
			if err != nil {
				b.Errorf("filter error: %v", err)
			}
		}
	})

	b.Run("Translate", func(b *testing.B) {
		translateStep, _ := steps.NewTranslateStep("{\"id\": body.id, \"processed\": true}")
		msg := engine.NewMessage()
		msg.Body = map[string]interface{}{"id": "test-123"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := translateStep.Process(ctx, msg)
			if err != nil {
				b.Errorf("translate error: %v", err)
			}
		}
	})

	b.Run("Authorize", func(b *testing.B) {
		authorizeStep, _ := steps.NewAuthorizeStep(
			config.AuthorizeSpec{
				Mode:       "rbac",
				Required:   false,
				AllowRoles: []string{"*"},
			},
		)
		msg := engine.NewMessage()
		msg.Principal = &engine.Principal{Subject: "user-123", Roles: []string{"admin"}}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := authorizeStep.Process(ctx, msg)
			if err != nil {
				b.Errorf("authorize error: %v", err)
			}
		}
	})

	b.Run("Idempotent", func(b *testing.B) {
		idempotentStep, _ := steps.NewIdempotentStep("body.transaction_id", 60)
		msg := engine.NewMessage()
		msg.Body = map[string]interface{}{"transaction_id": "txn-123"}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := idempotentStep.Process(ctx, msg)
			if err != nil {
				b.Errorf("idempotent error: %v", err)
			}
		}
	})
}

// BenchmarkContractValidation measures contract validation overhead
func BenchmarkContractValidation(b *testing.B) {
	ctx := context.Background()

	// Simple schema
	simpleSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id":   map[string]interface{}{"type": "string"},
			"name": map[string]interface{}{"type": "string"},
		},
		"required": []string{"id", "name"},
	}

	// Complex schema
	complexSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{"type": "string"},
			"nested": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"field1": map[string]interface{}{"type": "string"},
					"field2": map[string]interface{}{"type": "number"},
					"field3": map[string]interface{}{"type": "array"},
				},
			},
		},
		"required": []string{"id", "nested"},
	}

	msg := engine.NewMessage()
	msg.Body = map[string]interface{}{
		"id":   "test-123",
		"name": "Test Object",
		"nested": map[string]interface{}{
			"field1": "value1",
			"field2": 42.0,
			"field3": []string{"a", "b", "c"},
		},
	}

	b.Run("SimpleSchema", func(b *testing.B) {
		contractStep, _ := steps.NewContractStep(simpleSchema, "warn")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := contractStep.Process(ctx, msg)
			if err != nil {
				b.Errorf("contract validation error: %v", err)
			}
		}
	})

	b.Run("ComplexSchema", func(b *testing.B) {
		contractStep, _ := steps.NewContractStep(complexSchema, "warn")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := contractStep.Process(ctx, msg)
			if err != nil {
				b.Errorf("contract validation error: %v", err)
			}
		}
	})
}

// BenchmarkChannelOperations measures channel send/recv overhead
func BenchmarkChannelOperations(b *testing.B) {
	b.Run("Send", func(b *testing.B) {
		ch := engine.NewChannel("bench", 1000)
		msg := engine.NewMessage()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ch.Send(msg)
		}
	})

	b.Run("Recv", func(b *testing.B) {
		ch := engine.NewChannel("bench", 1000)
		msg := engine.NewMessage()

		// Pre-fill channel
		for i := 0; i < b.N; i++ {
			ch.Send(msg)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			<-ch.Recv()
		}
	})

	b.Run("SendRecv", func(b *testing.B) {
		ch := engine.NewChannel("bench", 100)
		msg := engine.NewMessage()

		go func() {
			for i := 0; i < b.N; i++ {
				<-ch.Recv()
			}
		}()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ch.Send(msg)
		}
	})
}

// BenchmarkMessageCloning measures message deep copy overhead
func BenchmarkMessageCloning(b *testing.B) {
	original := engine.NewMessage()
	original.Body = map[string]interface{}{
		"id":    "test-123",
		"amount": 1000.50,
		"items": []string{"a", "b", "c"},
	}
	original.Headers = map[string]string{
		"x-trace-id":  "trace-123",
		"x-user-id":   "user-456",
		"content-type": "application/json",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = original.Clone()
	}
}
