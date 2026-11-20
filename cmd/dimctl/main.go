package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/daemon"
	"github.com/naren-chakraview/dim/internal/factory"
	"github.com/naren-chakraview/dim/internal/lineage"
	"github.com/naren-chakraview/dim/internal/observability"
	"github.com/naren-chakraview/dim/internal/testing"
	"github.com/naren-chakraview/dim/internal/validation"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "dimctl",
	Short: "dim middleware control CLI",
	Long:  "dimctl is the command-line control tool for the dim integration middleware engine",
}

var replayCmd = &cobra.Command{
	Use:   "replay [options] <target-route>",
	Short: "Replay messages from a DLQ",
	Long:  "Recover and replay messages from a dead-letter queue back into a target route with optional filtering and rate limiting",
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Replay command needs update for current engine API
		// engine.NewEngine and related APIs were refactored in the current version
		_ = args // Avoid unused variable
		fmt.Fprintf(os.Stderr, "replay command: not yet updated for current engine API\n")
		return fmt.Errorf("replay command requires API update")
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate <file>",
	Short: "Validate a route configuration file",
	Long:  "Validate a route configuration YAML file against the JSON Schema. Returns exit code 0 on success, 1 on validation failure.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := args[0]
		strict, _ := cmd.Flags().GetBool("strict")

		// Load and validate the configuration
		cfg, err := config.LoadRouteConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "validation error: %v\n", err)
			return err
		}

		// Validate auth declarations
		mode := config.AuthValidationWarn
		if strict {
			mode = config.AuthValidationStrict
		}
		if err := config.ValidateAuthDeclarations(cfg, mode); err != nil {
			fmt.Fprintf(os.Stderr, "auth validation error: %v\n", err)
			return err
		}

		// Perform static contract conformance checks (M2.7.3)
		staticCheckWarnings := validation.PerformStaticContractChecks(cfg)

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

		// Print static contract conformance check results (M2.7.3)
		if len(staticCheckWarnings) > 0 {
			fmt.Fprintf(os.Stdout, "\nStatic contract conformance checks:\n")
			for _, warning := range staticCheckWarnings {
				fmt.Fprintf(os.Stdout, "  %s\n", warning)
			}
			// Print mandatory caveat (M2.7.3)
			fmt.Fprintf(os.Stdout, "%s\n", validation.StaticCheckCaveat)
		}

		return nil
	},
}

var resolveCmd = &cobra.Command{
	Use:   "resolve <file>",
	Short: "Resolve and display a route configuration after all imports and fragments are expanded",
	Long:  "Load a route configuration YAML, resolve all imports and fragment references, and output the fully-resolved route configuration as YAML. Useful for debugging imports and fragment expansion.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := args[0]

		// Load config (this internally resolves all imports and fragments)
		cfg, err := config.LoadRouteConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "resolve error: %v\n", err)
			return err
		}

		// Convert back to YAML and output
		output, err := yaml.Marshal(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "resolve error: failed to marshal resolved config: %v\n", err)
			return err
		}

		fmt.Fprintf(os.Stdout, "%s", output)
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

		return daemon.Serve(configPath, cfg)
	},
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

// Lineage command group
var lineageCmd = &cobra.Command{
	Use:   "lineage",
	Short: "Manage message lineage and audit trails",
	Long:  "Commands for querying, purging, and exporting message lineage records from the embedded SQLite store",
}

var lineagePurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge lineage records",
	Long:  "Manually delete lineage records by subject ID or time range",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, _ := cmd.Flags().GetString("db")
		if dbPath == "" {
			dbPath = "./lineage.db"
		}

		ctx := context.Background()
		store, err := lineage.NewStore(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open lineage store: %w", err)
		}
		defer store.Close()

		subjectID, _ := cmd.Flags().GetString("subject")
		if subjectID == "" {
			return fmt.Errorf("--subject is required for purge")
		}

		opts := lineage.PurgeOpts{SubjectID: subjectID}
		count, err := lineage.Purge(ctx, store, opts)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stdout, "Purged %d records for subject %q\n", count, subjectID)
		return nil
	},
}

var lineageExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export lineage records",
	Long:  "Export lineage records in CSV or NDJSON format",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, _ := cmd.Flags().GetString("db")
		if dbPath == "" {
			dbPath = "./lineage.db"
		}

		format, _ := cmd.Flags().GetString("format")
		if format != "csv" && format != "ndjson" {
			return fmt.Errorf("format must be 'csv' or 'ndjson'")
		}

		sinceStr, _ := cmd.Flags().GetString("since")
		untilStr, _ := cmd.Flags().GetString("until")

		var since, until time.Time
		if sinceStr != "" {
			var err error
			since, err = time.Parse(time.RFC3339, sinceStr)
			if err != nil {
				return fmt.Errorf("invalid --since format: %w", err)
			}
		} else {
			since = time.Now().UTC().Add(-30 * 24 * time.Hour) // Default: last 30 days
		}

		if untilStr != "" {
			var err error
			until, err = time.Parse(time.RFC3339, untilStr)
			if err != nil {
				return fmt.Errorf("invalid --until format: %w", err)
			}
		} else {
			until = time.Now().UTC()
		}

		ctx := context.Background()
		store, err := lineage.NewStore(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open lineage store: %w", err)
		}
		defer store.Close()

		filename := fmt.Sprintf("lineage-export.%s", format)
		f, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to create export file: %w", err)
		}
		defer f.Close()

		err = lineage.Export(ctx, store, format, since, until, f)
		if err != nil {
			return err
		}

		fmt.Fprintf(os.Stdout, "Exported lineage records to %s\n", filename)
		return nil
	},
}

var provenanceCmd = &cobra.Command{
	Use:   "provenance",
	Short: "Query message provenance",
	Long:  "Retrieve the complete provenance chain for a message or subject",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath, _ := cmd.Flags().GetString("db")
		if dbPath == "" {
			dbPath = "./lineage.db"
		}

		ctx := context.Background()
		store, err := lineage.NewStore(dbPath)
		if err != nil {
			return fmt.Errorf("failed to open lineage store: %w", err)
		}
		defer store.Close()

		messageID, _ := cmd.Flags().GetString("message-id")
		subjectID, _ := cmd.Flags().GetString("subject-id")

		if messageID == "" && subjectID == "" {
			return fmt.Errorf("either --message-id or --subject-id is required")
		}

		var chain *lineage.ProvenanceChain
		var chainErr error

		if messageID != "" {
			chain, chainErr = lineage.GetProvenance(ctx, store, messageID)
		} else {
			chain, chainErr = lineage.GetProvenanceBySubject(ctx, store, subjectID)
		}

		if chainErr != nil {
			return chainErr
		}

		fmt.Fprintf(os.Stdout, "Provenance chain: %d records\n", len(chain.Records))
		for i, record := range chain.Records {
			fmt.Fprintf(os.Stdout, "  [%d] %s (route=%s, created=%s)\n",
				i, record.ID, record.RouteName, record.CreatedAt.Format(time.RFC3339))
		}

		return nil
	},
}

// Trace command group
var traceCmd = &cobra.Command{
	Use:   "trace",
	Short: "Trace management commands",
	Long:  "Commands for querying, tailing, and analyzing OpenTelemetry spans",
}

var traceTailCmd = &cobra.Command{
	Use:   "tail",
	Short: "Stream live spans to terminal",
	Long:  "Stream live OpenTelemetry spans to the terminal in real-time for debugging. Shows recent spans first, then streams new ones.",
	RunE: func(cmd *cobra.Command, args []string) error {
		route, _ := cmd.Flags().GetString("route")
		service, _ := cmd.Flags().GetString("service")
		limit, _ := cmd.Flags().GetInt("limit")

		// Get tracing provider from environment or use a default one
		// In a running system, this would come from the observability system
		jaegerEndpoint := os.Getenv("JAEGER_ENDPOINT")
		if jaegerEndpoint == "" {
			jaegerEndpoint = "http://localhost:16686"
		}

		otelExporter := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		if otelExporter == "" {
			otelExporter = "http://localhost:4317"
		}

		// Create a tracing provider for tail operations
		// Note: In production, this would connect to the actual OTEL collector
		// For MVP, we create a local provider that can query spans from the system
		tp := observability.NewTracingProvider("stdout", jaegerEndpoint, 1.0)
		streamer := observability.NewSpanStreamer(tp)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Setup signal handling for graceful shutdown
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

		opts := observability.TailOptions{
			Route:    route,
			Service:  service,
			Limit:    limit,
			Endpoint: otelExporter,
			Ctx:      ctx,
		}

		fmt.Fprintf(os.Stderr, "Tailing spans from %s (service=%s, limit=%d)\n", service, service, limit)
		if route != "" {
			fmt.Fprintf(os.Stderr, "Filtering to route: %s\n", route)
		}
		fmt.Fprintf(os.Stderr, "Press Ctrl+C to stop\n\n")

		// Query recent spans first
		recentSpans, err := streamer.QueryRecentSpans(ctx, limit, route)
		if err != nil {
			return fmt.Errorf("failed to query recent spans: %w", err)
		}

		if len(recentSpans) > 0 {
			fmt.Fprintf(os.Stderr, "Recent spans:\n")
			for _, span := range recentSpans {
				fmt.Fprintf(os.Stdout, "%s\n", observability.FormatSpan(span, false))
			}
			fmt.Fprintf(os.Stderr, "\nStreaming live spans...\n")
		}

		// Stream live spans
		spanCh, err := streamer.StreamSpans(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to start span stream: %w", err)
		}

		// Process spans until cancelled
		for {
			select {
			case <-sigCh:
				fmt.Fprintf(os.Stderr, "\nshutting down span tail\n")
				cancel()
				return nil
			case span, ok := <-spanCh:
				if !ok {
					return nil
				}
				if span != nil {
					fmt.Fprintf(os.Stdout, "%s\n", observability.FormatSpan(span, true))
				}
			case <-ctx.Done():
				return nil
			}
		}
	},
}

var studioCmd = &cobra.Command{
	Use:   "studio [dir]",
	Short: "Launch visual route editor (local web UI)",
	Long:  "Start a local web UI for editing routes visually at localhost:7070",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		workDir := "."
		if len(args) > 0 {
			workDir = args[0]
		}
		return StartServer(7070, workDir)
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(resolveCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(replayCmd)
	rootCmd.AddCommand(scaffoldCmd)
	rootCmd.AddCommand(catalogCmd)
	rootCmd.AddCommand(lineageCmd)
	rootCmd.AddCommand(traceCmd)
	rootCmd.AddCommand(studioCmd)
	rootCmd.AddCommand(secretAuditCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(manifestCmd)

	// Add lineage subcommands
	lineageCmd.AddCommand(lineagePurgeCmd)
	lineageCmd.AddCommand(lineageExportCmd)
	lineageCmd.AddCommand(provenanceCmd)

	// Add trace subcommands
	traceCmd.AddCommand(traceTailCmd)

	// Add flags to validate command
	validateCmd.Flags().BoolP("strict", "s", false, "Treat missing auth declarations as errors instead of warnings")

	// Add --config flag to test command
	testCmd.Flags().StringP("config", "c", "", "Path to route configuration file (required for test command)")

	// Add flags to lineage commands
	lineagePurgeCmd.Flags().String("db", "./lineage.db", "Path to lineage database")
	lineagePurgeCmd.Flags().String("subject", "", "Subject ID to purge (required)")

	lineageExportCmd.Flags().String("db", "./lineage.db", "Path to lineage database")
	lineageExportCmd.Flags().String("format", "csv", "Export format (csv or ndjson)")
	lineageExportCmd.Flags().String("since", "", "Start time (RFC3339 format; default: 30 days ago)")
	lineageExportCmd.Flags().String("until", "", "End time (RFC3339 format; default: now)")

	provenanceCmd.Flags().String("db", "./lineage.db", "Path to lineage database")
	provenanceCmd.Flags().String("message-id", "", "Message correlation ID")
	provenanceCmd.Flags().String("subject-id", "", "Subject ID")

	// Add flags to trace tail command
	traceTailCmd.Flags().StringP("route", "r", "", "Filter to specific route (optional)")
	traceTailCmd.Flags().StringP("service", "s", "dim", "Service name (default: dim)")
	traceTailCmd.Flags().IntP("limit", "l", 20, "Number of recent spans to show before streaming (default: 20)")

	// Add flags to secret-audit command
	secretAuditCmd.Flags().BoolP("json", "j", false, "Output results as JSON")
}
