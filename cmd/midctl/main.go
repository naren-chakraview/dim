package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/engine"
	"github.com/naren-chakraview/dim/internal/factory"
	"github.com/naren-chakraview/dim/internal/observability"
	"github.com/naren-chakraview/dim/internal/testing"
	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "midctl",
	Short: "dim middleware control CLI",
	Long:  "midctl is the command-line control tool for the dim integration middleware engine",
}

var validateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate a route configuration file",
	Long:  "Validate a route configuration YAML file against the JSON Schema. Returns exit code 0 on success, 1 on validation failure.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := args[0]

		// Load and validate the configuration
		cfg, err := config.LoadRouteConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "validation error: %v\n", err)
			return err
		}

		// Success message
		fmt.Fprintf(os.Stdout, "OK: valid route configuration\n")
		fmt.Fprintf(os.Stdout, "  Version: %d\n", cfg.Version)
		fmt.Fprintf(os.Stdout, "  Sources: %d\n", len(cfg.Sources))
		fmt.Fprintf(os.Stdout, "  Sinks: %d\n", len(cfg.Sinks))
		fmt.Fprintf(os.Stdout, "  Routes: %d\n", len(cfg.Routes))

		// Print route versions for lineage tracking
		if len(cfg.Routes) > 0 {
			fmt.Fprintf(os.Stdout, "  Route versions:\n")
			for name, route := range cfg.Routes {
				if route.RouteVersion != "" {
					fmt.Fprintf(os.Stdout, "    %s: %s\n", name, route.RouteVersion)
				}
			}
		}

		return nil
	},
}

var runCmd = &cobra.Command{
	Use:   "run <file>",
	Short: "Run a route configuration",
	Long:  "Load a route configuration YAML and run the integration pipeline. Listens for messages on HTTP /ingest, processes through steps, outputs to file. Supports single or multiple independent routes.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := args[0]

		// Load config
		cfg, err := config.LoadRouteConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Check if single-route or multi-route
		if len(cfg.Routes) == 1 {
			return runSingleRoute(cfg)
		}

		return runMultiRoute(cfg)
	},
}

// runSingleRoute runs a single-route configuration (backward compatible)
func runSingleRoute(cfg *config.RouteConfig) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize observability
	obsConfig := observability.FromRouteConfig(cfg)
	if err := obsConfig.Validate(); err != nil {
		return fmt.Errorf("observability config validation failed: %w", err)
	}

	var metricsCollector *observability.MetricsCollector
	var tracingProvider *observability.TracingProvider
	var metricsServer *http.Server

	// Initialize metrics if enabled
	if obsConfig.MetricsEnabled {
		metricsCollector = observability.NewMetricsCollector(obsConfig.PrometheusAddr)
		metricsServer = startMetricsServer(ctx, obsConfig.PrometheusAddr, metricsCollector)
		fmt.Fprintf(os.Stderr, "metrics enabled on %s/metrics\n", obsConfig.PrometheusAddr)
	}

	// Initialize tracing if enabled
	if obsConfig.TracingEnabled {
		tracingProvider = observability.NewTracingProvider(obsConfig.OtelExporter, obsConfig.JaegerEndpoint, obsConfig.SampleRate)
		fmt.Fprintf(os.Stderr, "tracing enabled with %s exporter (sample_rate=%.2f)\n", obsConfig.OtelExporter, obsConfig.SampleRate)
	}

	executor, _, _, _, sources, sinks, err := factory.BuildSingleRoutePipeline(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to build pipeline: %w", err)
	}

	// Store metrics and tracing in context for executor access (optional enhancement)
	_ = metricsCollector
	_ = tracingProvider

	// Start all adapters
	for _, source := range sources {
		if err := source.Start(ctx); err != nil {
			return fmt.Errorf("source startup failed: %w", err)
		}
	}
	for _, sink := range sinks {
		if err := sink.Start(ctx); err != nil {
			return fmt.Errorf("sink startup failed: %w", err)
		}
	}

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	// Start executor in goroutine
	executorErr := make(chan error, 1)
	go func() {
		executorErr <- executor.Run(ctx)
	}()

	// Print startup message
	fmt.Fprintf(os.Stderr, "listening on http://localhost:8080/ingest\n")
	fmt.Fprintf(os.Stderr, "press Ctrl+C to stop\n")

	// Wait for signal or executor error
	select {
	case err := <-executorErr:
		if err != nil && !strings.Contains(err.Error(), "context canceled") {
			return err
		}
	case <-sigCh:
		fmt.Fprintf(os.Stderr, "\nshutting down...\n")
		signal.Stop(sigCh)

		// Shutdown metrics server if running
		if metricsServer != nil {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
			metricsServer.Shutdown(shutdownCtx)
			shutdownCancel()
		}

		// Cancel context to stop executor
		cancel()

		// Wait for executor to finish
		select {
		case <-executorErr:
		case <-time.After(5 * time.Second):
			fmt.Fprintf(os.Stderr, "shutdown timeout\n")
		}
	}

	return nil
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

// runMultiRoute runs a multi-route configuration with coordinated lifecycle
func runMultiRoute(cfg *config.RouteConfig) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize observability
	obsConfig := observability.FromRouteConfig(cfg)
	if err := obsConfig.Validate(); err != nil {
		return fmt.Errorf("observability config validation failed: %w", err)
	}

	var metricsCollector *observability.MetricsCollector
	var tracingProvider *observability.TracingProvider
	var metricsServer *http.Server

	// Initialize metrics if enabled
	if obsConfig.MetricsEnabled {
		metricsCollector = observability.NewMetricsCollector(obsConfig.PrometheusAddr)
		metricsServer = startMetricsServer(ctx, obsConfig.PrometheusAddr, metricsCollector)
		fmt.Fprintf(os.Stderr, "metrics enabled on %s/metrics\n", obsConfig.PrometheusAddr)
	}

	// Initialize tracing if enabled
	if obsConfig.TracingEnabled {
		tracingProvider = observability.NewTracingProvider(obsConfig.OtelExporter, obsConfig.JaegerEndpoint, obsConfig.SampleRate)
		fmt.Fprintf(os.Stderr, "tracing enabled with %s exporter (sample_rate=%.2f)\n", obsConfig.OtelExporter, obsConfig.SampleRate)
	}

	executors, router, sources, sinks, err := factory.BuildMultiRoutePipeline(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to build multi-route pipeline: %w", err)
	}

	// Store metrics and tracing in context for executor access (optional enhancement)
	_ = metricsCollector
	_ = tracingProvider

	fmt.Fprintf(os.Stderr, "configured %d routes\n", len(executors))

	// Start all adapters
	for _, source := range sources {
		if err := source.Start(ctx); err != nil {
			return fmt.Errorf("source startup failed: %w", err)
		}
	}
	for _, sink := range sinks {
		if err := sink.Start(ctx); err != nil {
			return fmt.Errorf("sink startup failed: %w", err)
		}
	}

	// Start the message router
	if err := router.Start(ctx); err != nil {
		return fmt.Errorf("failed to start message router: %w", err)
	}

	// Setup signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	// Start all executors in goroutines
	executorErrorChannels := make([]chan error, 0, len(executors))
	routeNames := make([]string, 0, len(executors))
	for routeName, executor := range executors {
		ch := make(chan error, 1)
		executorErrorChannels = append(executorErrorChannels, ch)
		routeNames = append(routeNames, routeName)
		go func(name string, exec *engine.Executor) {
			ch <- exec.Run(ctx)
		}(routeName, executor)
	}

	// Print startup message
	fmt.Fprintf(os.Stderr, "listening on http://localhost:8080/ingest\n")
	fmt.Fprintf(os.Stderr, "press Ctrl+C to stop\n")

	// Wait for signal or executor error
	select {
	case <-sigCh:
		fmt.Fprintf(os.Stderr, "\nshutting down...\n")
		signal.Stop(sigCh)

		// Shutdown metrics server if running
		if metricsServer != nil {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
			metricsServer.Shutdown(shutdownCtx)
			shutdownCancel()
		}

		// Cancel the context to stop all executors and the router
		cancel()

		// Wait for all executors to finish (with timeout)
		timeout := time.After(10 * time.Second)
		completed := 0
		for i, errCh := range executorErrorChannels {
			select {
			case err := <-errCh:
				if err != nil && !strings.Contains(err.Error(), "context canceled") {
					fmt.Fprintf(os.Stderr, "executor %q error: %v\n", routeNames[i], err)
				}
				completed++
			case <-timeout:
				fmt.Fprintf(os.Stderr, "executor shutdown timeout (completed %d/%d)\n", completed, len(executorErrorChannels))
				break
			}
		}

		fmt.Fprintf(os.Stderr, "shutdown complete\n")
	}

	return nil
}

var testCmd = &cobra.Command{
	Use:   "test <path>",
	Short: "Run fixture-based integration tests",
	Long:  "Load and execute fixture-based integration tests from a YAML file or directory. Each fixture defines an input message and expected output/error outcomes.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fixturesPath := args[0]

		// Load fixtures from file or directory
		var fixtures []*testing.Fixture
		var err error

		// Check if path is a file or directory
		info, err := os.Stat(fixturesPath)
		if err != nil {
			return fmt.Errorf("failed to access fixtures path: %w", err)
		}

		if info.IsDir() {
			fixtures, err = testing.LoadFixturesFromDirectory(fixturesPath)
		} else {
			fixtures, err = testing.LoadFixturesFromFile(fixturesPath)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load fixtures: %v\n", err)
			return err
		}

		if len(fixtures) == 0 {
			fmt.Fprintf(os.Stderr, "no fixtures found\n")
			return fmt.Errorf("no fixtures found in %s", fixturesPath)
		}

		fmt.Fprintf(os.Stderr, "loaded %d fixture(s)\n", len(fixtures))

		// Validate fixtures
		if err := testing.ValidateFixtures(fixtures); err != nil {
			fmt.Fprintf(os.Stderr, "fixture validation failed: %v\n", err)
			return err
		}

		// Load route configuration - use a default minimal config
		// For fixture testing, we need a route config to build the pipeline
		// We'll create a synthetic config from fixtures or load from a specified file
		configPath := cmd.Flag("config").Value.String()
		if configPath == "" {
			configPath = "route.yaml" // Default config file
		}

		cfg, err := config.LoadRouteConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load route config: %v\n", err)
			return err
		}

		// Build pipeline for testing
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		executor, _, _, _, _, _, err := factory.BuildSingleRoutePipeline(ctx, cfg)
		if err != nil {
			return fmt.Errorf("failed to build pipeline: %w", err)
		}

		// Create fixture runner
		runner := testing.NewFixtureRunner(executor, 30000) // 30 second default timeout

		// Run all fixtures
		results := runner.RunFixtures(ctx, fixtures)

		// Print results
		fmt.Fprintf(os.Stdout, "\n--- Test Results ---\n")
		_, failed, errCount, summary := testing.SummarizeResults(results)
		fmt.Fprintf(os.Stdout, "%s\n\n", summary)

		// Print detailed results
		for _, result := range results {
			if result.Passed {
				fmt.Fprintf(os.Stdout, "✓ %s (%dms)\n", result.Name, result.DurationMs)
			} else {
				fmt.Fprintf(os.Stdout, "✗ %s (%dms)\n", result.Name, result.DurationMs)
				fmt.Fprintf(os.Stdout, "  Reason: %s\n", result.FailureReason)
				if result.ActualError != nil {
					fmt.Fprintf(os.Stdout, "  Error: %v\n", result.ActualError)
				}
			}
		}

		// Exit with appropriate code
		if failed > 0 || errCount > 0 {
			return fmt.Errorf("tests failed: %d failed, %d errors", failed, errCount)
		}

		fmt.Fprintf(os.Stdout, "\nAll tests passed!\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(testCmd)

	// Add --config flag to test command
	testCmd.Flags().StringP("config", "c", "", "Path to route configuration file (required for test command)")
}
