package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/naren-chakraview/dim/internal/cluster"
	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/daemon"
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

	// Initialize cluster if in cluster mode
	var clusterInstance *cluster.Cluster
	clusterConfig, clusterErr := cluster.NewClusterConfigFromEnv()
	if clusterErr == nil && clusterConfig.IsClustered() {
		// Cluster mode enabled
		clusterInstance, err = cluster.NewCluster(clusterConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize cluster: %w", err)
		}
		defer clusterInstance.Close(ctx)
		log.Printf("[INFO] cluster mode initialized with %d instances", len(clusterConfig.Instances))
	} else if clusterErr != nil {
		// Cluster env vars not set or invalid: single-instance mode
		log.Printf("[DEBUG] cluster mode not configured (single-instance mode)")
	}

	return daemon.Serve(configPath, cfg)
}
