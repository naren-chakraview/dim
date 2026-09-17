package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

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
	Long:  "Generate the agent capability manifest from schemas/route.schema.json",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		schemaPath := args[0]
		outputPath := "docs/schemas/capability-manifest.json"
		if len(args) > 1 {
			outputPath = args[1]
		}

		// Load the schema
		schemaData, err := ioutil.ReadFile(schemaPath)
		if err != nil {
			return fmt.Errorf("failed to read schema: %w", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaData, &schema); err != nil {
			return fmt.Errorf("failed to parse schema: %w", err)
		}

		// Extract capabilities from schema
		manifest := generateManifestFromSchema(schema)

		// Write manifest
		manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal manifest: %w", err)
		}

		if err := ioutil.WriteFile(outputPath, manifestJSON, 0644); err != nil {
			return fmt.Errorf("failed to write manifest: %w", err)
		}

		fmt.Printf("✅ Generated capability manifest: %s\n", outputPath)
		fmt.Printf("   Capabilities: %d\n", len(manifest["capabilities"].([]interface{})))
		return nil
	},
}

// generateManifestFromSchema extracts capabilities from the route schema
func generateManifestFromSchema(schema map[string]interface{}) map[string]interface{} {
	capabilities := []interface{}{}

	// Extract from definitions (adapters, steps, etc.)
	if definitions, ok := schema["definitions"].(map[string]interface{}); ok {
		// Extract step types from the oneOf in definitions.step
		if stepDef, ok := definitions["step"].(map[string]interface{}); ok {
			if oneOf, ok := stepDef["oneOf"].([]interface{}); ok {
				for _, stepRef := range oneOf {
					if refMap, ok := stepRef.(map[string]interface{}); ok {
						if ref, ok := refMap["$ref"].(string); ok {
							// Extract step type from reference (e.g., "#/definitions/translate_step" -> "translate")
							stepType := extractStepType(ref)
							if stepType != "" {
								if stepDef, ok := definitions[stepType+"_step"].(map[string]interface{}); ok {
									capability := map[string]interface{}{
										"name":           stepType,
										"type":           "step",
										"description":    getDescription(stepDef),
										"config_schema": extractSchema(stepDef),
									}
									capabilities = append(capabilities, capability)
								}
							}
						}
					}
				}
			}
		}

		// Extract adapter types (http-source, http-sink, file-source, file-sink, s3-source, s3-sink, etc.)
		adapterTypes := []string{
			"http-source", "http-sink",
			"file-source", "file-sink",
			"s3-source", "s3-sink",
			"sftp-source", "sftp-sink",
			"kafka-source", "kafka-sink",
		}

		for _, adapterType := range adapterTypes {
			capability := map[string]interface{}{
				"name":        adapterType,
				"type":        "adapter",
				"description": fmt.Sprintf("%s adapter", adapterType),
				"config_schema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"type": map[string]interface{}{
							"type": "string",
							"enum": []string{extractAdapterTypeEnum(adapterType)},
						},
					},
					"required": []string{"type"},
				},
			}
			capabilities = append(capabilities, capability)
		}
	}

	return map[string]interface{}{
		"capabilities": capabilities,
		"version":      "1.0.0",
		"generated":    true,
	}
}

func extractStepType(ref string) string {
	// Convert "#/definitions/translate_step" to "translate"
	if len(ref) > len("#/definitions/") && len(ref) > len("_step") {
		start := len("#/definitions/")
		end := len(ref) - len("_step")
		if end > start {
			return ref[start:end]
		}
	}
	return ""
}

func extractAdapterTypeEnum(adapterType string) string {
	// Convert "http-source" to "http"
	if len(adapterType) > 7 {
		return adapterType[:len(adapterType)-7]
	}
	return adapterType
}

func getDescription(def map[string]interface{}) string {
	if desc, ok := def["description"].(string); ok {
		return desc
	}
	return ""
}

func extractSchema(def map[string]interface{}) map[string]interface{} {
	// For now, return a minimal schema with just the type
	return map[string]interface{}{
		"type": "object",
	}
}

func init() {
	manifestCmd.AddCommand(manifestGenerateCmd)
}
