package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/factory"
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

		return nil
	},
}

var runCmd = &cobra.Command{
	Use:   "run <file>",
	Short: "Run a route configuration",
	Long:  "Load a route configuration YAML and run the integration pipeline. Listens for messages on HTTP /ingest, processes through steps, outputs to file.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := args[0]

		// Load config
		cfg, err := config.LoadRouteConfig(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Build pipeline
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		executor, _, _, _, sources, sinks, err := factory.BuildPipeline(ctx, cfg)
		if err != nil {
			return fmt.Errorf("failed to build pipeline: %w", err)
		}

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
			cancel()

			// Wait for executor to finish
			select {
			case <-executorErr:
			case <-time.After(5 * time.Second):
				fmt.Fprintf(os.Stderr, "shutdown timeout\n")
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(runCmd)
}
