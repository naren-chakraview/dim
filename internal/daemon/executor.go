package daemon

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/observability"
	"github.com/naren-chakraview/dim/internal/ordering"
	"github.com/naren-chakraview/dim/internal/steps"
)

func truncateVersion(version string) string {
	if len(version) > 16 {
		return version[:16]
	}
	return version
}

// buildExecutorForRoute builds a new executor for a given route.
// Used during hot reload to create executors for updated route configs.
// M0.2.10: Helper for SIGHUP reload.
// BuildExecutorForRoute builds the executor for one route from its config. Used at
// startup and on hot reload.
//
// outputCh and errorCh are the channels the route's sinks read. A reload passes
// the previous executor's channels, so the new generation writes to the same
// sinks. Startup passes nil and gets new channels.
func BuildExecutorForRoute(ctx context.Context, routeName string, routeSpec config.RouteSpec, outputCh, errorCh *engine.Channel) (*engine.Executor, error) {
	// Create executor channels
	// Note: channel names won't change for a given route
	inputCh := engine.NewChannel(fmt.Sprintf("router-to-executor-%s", routeName), 100)
	if outputCh == nil {
		outputCh = engine.NewChannel(fmt.Sprintf("executor-%s-to-sink", routeName), 100)
	}

	// Create error channel if error path is configured
	if routeSpec.ErrorPath != nil {
		if errorCh == nil {
			errorCh = engine.NewChannel(fmt.Sprintf("executor-%s-to-error-sink", routeName), 100)
		}
	}

	// Build contract store from route config (M0.3.5)
	contractStore := config.NewContractStore()
	if len(routeSpec.Contracts) > 0 {
		if err := contractStore.LoadContracts(routeName, routeSpec.Contracts); err != nil {
			return nil, fmt.Errorf("failed to load contracts for route %q: %w", routeName, err)
		}
	}

	// Build steps from route config (dedupStore is nil for CLI tool)
	stepsInstances, stepNames, err := steps.BuildStepsFromSpec(routeSpec.Steps, contractStore, routeName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build steps for route %q: %w", routeName, err)
	}

	// Determine worker count based on ordering requirement
	defaultWorkers := 4
	numWorkers := ordering.GetWorkerCount(&routeSpec, defaultWorkers)
	if ordering.IsOrderingRequired(&routeSpec) {
		log.Printf("[INFO] Route %q has ordering=required, constraining to 1 worker for serial processing", routeName)
	}

	// Create executor with appropriate worker count and error channel
	var executor *engine.Executor
	if errorCh != nil {
		executor = engine.NewExecutorWithWorkers(routeName, inputCh, outputCh, errorCh, stepsInstances, stepNames, numWorkers)
	} else {
		executor = engine.NewExecutorWithWorkers(routeName, inputCh, outputCh, nil, stepsInstances, stepNames, numWorkers)
	}

	return executor, nil
}

// startMetricsServer starts the Prometheus metrics HTTP server
func startMetricsServer(ctx context.Context, addr string, mc *observability.MetricsCollector) *http.Server {
	server := &http.Server{
		Addr: addr,
	}

	// Register metrics endpoint
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.Write([]byte(mc.GetMetrics()))
	})

	// Start server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[WARN] metrics server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	return server
}

// runMultiRoute runs a multi-route configuration with coordinated lifecycle and hot reload support.
// M0.2.10: Added SIGHUP handler for hot reload with graceful in-flight message draining.
