package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/naren-chakraview/dim/internal/agent"
	"github.com/spf13/cobra"
)

var manifestCmd = &cobra.Command{
	Use:   "manifest",
	Short: "Generate or verify the capability manifest",
	Long:  "Generate the agent capability manifest from schemas/route.schema.json",
}

var manifestGenerateCmd = &cobra.Command{
	Use:   "generate <schema-file> [output-file]",
	Short: "Generate capability manifest from schema",
	Long: "Generate the agent capability manifest from a route JSON schema. " +
		"Uses the same builder as the served GetCapabilities, so the output " +
		"is the served catalog for that schema file.",
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		schemaPath := args[0]
		outputPath := "docs/schemas/capability-manifest.json"
		if len(args) > 1 {
			outputPath = args[1]
		}

		schemaData, err := os.ReadFile(schemaPath)
		if err != nil {
			return fmt.Errorf("failed to read schema: %w", err)
		}

		capabilities, err := agent.BuildCapabilities(schemaData, "")
		if err != nil {
			return fmt.Errorf("failed to build capabilities from %s: %w", schemaPath, err)
		}

		manifest := map[string]interface{}{
			"schema_version": agent.CapabilitySchemaVersion,
			"capabilities":   capabilities,
		}

		manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal manifest: %w", err)
		}
		manifestJSON = append(manifestJSON, '\n')

		if err := os.WriteFile(outputPath, manifestJSON, 0644); err != nil {
			return fmt.Errorf("failed to write manifest: %w", err)
		}

		fmt.Printf("✅ Generated capability manifest: %s\n", outputPath)
		fmt.Printf("   Capabilities: %d\n", len(capabilities))
		return nil
	},
}

func init() {
	manifestCmd.AddCommand(manifestGenerateCmd)
}
