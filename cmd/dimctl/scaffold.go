package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold data-product --domain <name> [--source <type>] [--sink <type>] [--template <shape>] [--with-contract]",
	Short: "Generate a domain directory scaffold with starter route",
	Long:  "Generate a complete domain directory pre-wired with governance fragments and starter routes",
	RunE:  runScaffold,
}

var (
	domainName   string
	sourceType   string
	sinkType     string
	withContract bool
	templateType string
)

func init() {
	scaffoldCmd.Flags().StringVar(&domainName, "domain", "", "Domain name (required)")
	scaffoldCmd.Flags().StringVar(&sourceType, "source", "http", "Source type (http, file, kafka, amqp)")
	scaffoldCmd.Flags().StringVar(&sinkType, "sink", "file", "Sink type (file, kafka, http)")
	scaffoldCmd.Flags().BoolVar(&withContract, "with-contract", false, "Generate contract enforcement template")
	scaffoldCmd.Flags().StringVar(&templateType, "template", "passthrough", "Template shape (passthrough, transform, contract)")

	scaffoldCmd.MarkFlagRequired("domain")
}

func runScaffold(cmd *cobra.Command, args []string) error {
	if domainName == "" {
		return fmt.Errorf("--domain is required")
	}

	// Create domain directory
	domainPath := filepath.Join("domains", domainName)
	if _, err := os.Stat(domainPath); err == nil {
		return fmt.Errorf("domain directory already exists: %s (refusing to overwrite)", domainPath)
	}

	if err := os.MkdirAll(domainPath, 0755); err != nil {
		return fmt.Errorf("failed to create domain directory: %w", err)
	}

	// Generate DOMAIN.yaml
	if err := generateDomainMetadata(domainPath, domainName); err != nil {
		return fmt.Errorf("failed to generate domain metadata: %w", err)
	}

	// Generate starter route
	if err := generateRoute(domainPath, domainName, templateType); err != nil {
		return fmt.Errorf("failed to generate route: %w", err)
	}

	// Generate error path template if requested
	if withContract {
		if err := generateErrorPathTemplate(domainPath); err != nil {
			return fmt.Errorf("failed to generate error path template: %w", err)
		}
	}

	fmt.Printf("✓ Generated domain scaffold at %s/\n", domainPath)
	fmt.Printf("✓ Run 'dimctl validate domains/%s/*.yaml' to verify\n", domainName)
	fmt.Printf("✓ Next: edit routes and run 'dimctl test domains/%s/<route>.route_test.yaml'\n", domainName)

	return nil
}

func generateDomainMetadata(domainPath, domainName string) error {
	data := map[string]string{
		"DomainName": domainName,
	}

	tmpl, err := template.New("domain").Parse(domainMetadataTemplate)
	if err != nil {
		return err
	}

	filePath := filepath.Join(domainPath, "DOMAIN.yaml")
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}

func generateRoute(domainPath, domainName, templateShape string) error {
	var routeYAML string
	switch templateShape {
	case "passthrough":
		routeYAML = passthroughTemplate
	case "transform":
		routeYAML = transformTemplate
	case "contract":
		routeYAML = contractTemplate
	default:
		return fmt.Errorf("unknown template shape: %s (valid: passthrough, transform, contract)", templateShape)
	}

	routePath := filepath.Join(domainPath, fmt.Sprintf("%s-route.yaml", domainName))
	return os.WriteFile(routePath, []byte(routeYAML), 0644)
}

func generateErrorPathTemplate(domainPath string) error {
	errorPath := filepath.Join(domainPath, "error-path.yaml")
	return os.WriteFile(errorPath, []byte(errorPathTemplate), 0644)
}
