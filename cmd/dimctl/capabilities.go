package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/naren-chakraview/dim/internal/agent"
	"github.com/spf13/cobra"
)

var capabilitiesCmd = &cobra.Command{
	Use:   "capabilities [--filter adapter|step|eip]",
	Short: "Query supported capabilities (adapters, steps, EIPs)",
	Long:  "Query the enumerable catalog of all supported adapters, steps, and Enterprise Integration Patterns with their JSON schemas",
	RunE: func(cmd *cobra.Command, args []string) error {
		filterType, _ := cmd.Flags().GetString("filter")

		ctx := context.Background()
		req := agent.CapabilitiesRequest{
			FilterType: filterType,
		}

		resp, err := agent.GetCapabilities(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to get capabilities: %v", err)
		}

		// Create manifest with interface version and timestamp
		manifest := map[string]interface{}{
			"interface_version": agent.AgentInterfaceVersion,
			"generated_at":      time.Now().UTC().Format(time.RFC3339),
			"capabilities":      resp.Capabilities,
			"schema_version":    resp.SchemaVersion,
		}

		// Output as JSON
		data, marshalErr := json.MarshalIndent(manifest, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal manifest: %v", marshalErr)
		}

		fmt.Println(string(data))
		return nil
	},
}

func init() {
	capabilitiesCmd.Flags().StringP("filter", "f", "", "Filter capabilities by type: 'adapter', 'step', or 'eip'")
	rootCmd.AddCommand(capabilitiesCmd)
}
