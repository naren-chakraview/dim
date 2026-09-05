package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/factory"
	"github.com/naren-chakraview/dim/internal/observability"
	"github.com/naren-chakraview/dim/internal/observability/viewer"
	"github.com/spf13/cobra"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "dimd",
	Short: "dim integration middleware engine daemon",
	Long:  "dimd is the engine daemon for the dim declarative integration middleware. It loads route configurations and processes messages through the integrated pipeline.",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		if configPath == "" {
			return fmt.Errorf("--config flag is required")
		}
		return runDaemon(configPath)
	},
}

var startCmd = &cobra.Command{
	Use:   "start <config-file>",
	Short: "Start the engine daemon with a route configuration",
	Long:  "Load a route configuration YAML and run the integration pipeline as a daemon. Listens for messages on HTTP /message, processes through routes, outputs to configured sinks.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDaemon(args[0])
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate a route configuration file",
	Long:  "Validate a route configuration YAML file against the JSON Schema.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := args[0]
		strict, _ := cmd.Flags().GetBool("strict")

		cfg, err := config.LoadRouteConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "validation error: %v\n", err)
			return err
		}

		mode := config.AuthValidationWarn
		if strict {
			mode = config.AuthValidationStrict
		}
		if err := config.ValidateAuthDeclarations(cfg, mode); err != nil {
			fmt.Fprintf(os.Stderr, "auth validation error: %v\n", err)
			return err
		}

		fmt.Fprintf(os.Stdout, "OK: valid route configuration\n")
		fmt.Fprintf(os.Stdout, "  Version: %d\n", cfg.Version)
		fmt.Fprintf(os.Stdout, "  Sources: %d\n", len(cfg.Sources))
		fmt.Fprintf(os.Stdout, "  Sinks: %d\n", len(cfg.Sinks))
		fmt.Fprintf(os.Stdout, "  Routes: %d\n", len(cfg.Routes))

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

func init() {
	rootCmd.Flags().StringP("config", "c", "", "Path to route configuration file")
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(validateCmd)

	validateCmd.Flags().BoolP("strict", "s", false, "Require strict auth declaration validation")
}

// runDaemon starts the engine daemon with the given configuration
func runDaemon(configPath string) error {
	log.Printf("[INFO] dim engine daemon starting with config: %s", configPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg, err := config.LoadRouteConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	log.Printf("[INFO] loaded configuration with %d routes", len(cfg.Routes))

	// Initialize observability
	obsConfig := observability.FromRouteConfig(cfg)
	if err := obsConfig.Validate(); err != nil {
		return fmt.Errorf("observability config validation failed: %w", err)
	}

	var metricsCollector *observability.MetricsCollector
	var tracingProvider *observability.TracingProvider
	var metricsServer *http.Server
	var viewerServer *http.Server
	var viewerSrv *viewer.ViewerServer

	// Initialize metrics if enabled
	if obsConfig.MetricsEnabled {
		metricsCollector = observability.NewMetricsCollector(obsConfig.PrometheusAddr)
		metricsServer = startMetricsServer(ctx, obsConfig.PrometheusAddr, metricsCollector)
		log.Printf("[INFO] metrics enabled on %s/metrics", obsConfig.PrometheusAddr)
	}

	// Initialize tracing if enabled
	if obsConfig.TracingEnabled {
		tracingProvider = observability.NewTracingProvider(obsConfig.OtelExporter, obsConfig.JaegerEndpoint, obsConfig.SampleRate)
		log.Printf("[INFO] tracing enabled with %s exporter (sample_rate=%.2f)", obsConfig.OtelExporter, obsConfig.SampleRate)
	}

	// Initialize Tier 1 viewer
	var err2 error
	viewerSrv, viewerServer, err2 = viewer.StartServer(":8081")
	if err2 != nil {
		return fmt.Errorf("failed to start viewer server: %w", err2)
	}
	log.Printf("[INFO] Tier 1 viewer enabled on http://localhost:8081/debug/routes")
	_ = viewerSrv

	// Build and start pipeline
	if len(cfg.Routes) == 1 {
		if err := runSingleRoute(ctx, cfg, tracingProvider, metricsCollector, viewerSrv); err != nil {
			return err
		}
	} else {
		if err := runMultiRoute(ctx, cfg, configPath, tracingProvider, metricsCollector, viewerSrv); err != nil {
			return err
		}
	}

	// Cleanup
	if metricsServer != nil {
		metricsServer.Shutdown(ctx)
	}
	if viewerServer != nil {
		viewerServer.Shutdown(ctx)
	}

	return nil
}

// runSingleRoute runs a single-route configuration
func runSingleRoute(ctx context.Context, cfg *config.RouteConfig, tracingProvider *observability.TracingProvider, metricsCollector *observability.MetricsCollector, viewerSrv *viewer.ViewerServer) error {
	executor, _, _, _, sources, sinks, err := factory.BuildSingleRoutePipelineWithTracing(ctx, cfg, tracingProvider)
	if err != nil {
		return fmt.Errorf("failed to build pipeline: %w", err)
	}

	// Start executor and sources
	go executor.Run(ctx)
	for _, source := range sources {
		go source.Start(ctx)
	}
	for _, sink := range sinks {
		go sink.Start(ctx)
	}

	log.Printf("[INFO] engine daemon started successfully. Listening on http://localhost:8080/message")
	log.Printf("[INFO] Press Ctrl+C to stop")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("[INFO] shutdown signal received, stopping daemon")

	// Stop sources and sinks
	for _, source := range sources {
		if err := source.Stop(); err != nil {
			log.Printf("[WARN] error stopping source: %v", err)
		}
	}
	for _, sink := range sinks {
		if err := sink.Stop(); err != nil {
			log.Printf("[WARN] error stopping sink: %v", err)
		}
	}

	log.Printf("[INFO] daemon shutdown complete")
	return nil
}

// runMultiRoute runs a multi-route configuration
func runMultiRoute(ctx context.Context, cfg *config.RouteConfig, configPath string, tracingProvider *observability.TracingProvider, metricsCollector *observability.MetricsCollector, viewerSrv *viewer.ViewerServer) error {
	_, _, sources, sinks, err := factory.BuildMultiRoutePipelineWithTracing(ctx, cfg, tracingProvider)
	if err != nil {
		return fmt.Errorf("failed to build multi-route pipeline: %w", err)
	}
	// Start all sources and sinks
	// TODO: multi-route executor API changed in current version
	// The old executor.Run pattern is no longer used; MessageRouter manages execution
	for _, source := range sources {
		go source.Start(ctx)
	}
	for _, sink := range sinks {
		go sink.Start(ctx)
	}

	log.Printf("[INFO] multi-route engine daemon started successfully. Listening on http://localhost:8080/message")
	log.Printf("[INFO] Press Ctrl+C to stop")

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("[INFO] shutdown signal received, stopping daemon")

	// Stop sources and sinks
	for _, source := range sources {
		if err := source.Stop(); err != nil {
			log.Printf("[WARN] error stopping source: %v", err)
		}
	}
	for _, sink := range sinks {
		if err := sink.Stop(); err != nil {
			log.Printf("[WARN] error stopping sink: %v", err)
		}
	}

	log.Printf("[INFO] daemon shutdown complete")
	return nil
}

// startMetricsServer starts the Prometheus metrics HTTP server
func startMetricsServer(ctx context.Context, addr string, metricsCollector *observability.MetricsCollector) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		w.Write([]byte(metricsCollector.GetMetrics()))
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("[DEBUG] metrics server listening on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[ERROR] metrics server error: %v", err)
		}
	}()

	return server
}
