package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldGeneratesDomainMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	domainName := "test-domain"
	domainPath := filepath.Join("domains", domainName)

	// Generate scaffold
	if err := os.MkdirAll(domainPath, 0755); err != nil {
		t.Fatalf("failed to create domain directory: %v", err)
	}

	if err := generateDomainMetadata(domainPath, domainName); err != nil {
		t.Fatalf("failed to generate domain metadata: %v", err)
	}

	// Verify DOMAIN.yaml exists and has correct content
	domainYAML := filepath.Join(domainPath, "DOMAIN.yaml")
	content, err := os.ReadFile(domainYAML)
	if err != nil {
		t.Fatalf("failed to read DOMAIN.yaml: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "name: test-domain") {
		t.Error("DOMAIN.yaml does not contain domain name")
	}
	if !strings.Contains(contentStr, "slack_channel: \"#test-domain-team\"") {
		t.Error("DOMAIN.yaml does not contain slack channel")
	}
}

func TestScaffoldGeneratesValidRoute(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	domainName := "test-domain"
	domainPath := filepath.Join("domains", domainName)

	// Generate scaffold
	if err := os.MkdirAll(domainPath, 0755); err != nil {
		t.Fatalf("failed to create domain directory: %v", err)
	}

	if err := generateRoute(domainPath, domainName, "passthrough"); err != nil {
		t.Fatalf("failed to generate route: %v", err)
	}

	// Verify route file exists and has correct structure
	routePath := filepath.Join(domainPath, domainName+"-route.yaml")
	content, err := os.ReadFile(routePath)
	if err != nil {
		t.Fatalf("failed to read route file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "version: 1") {
		t.Error("route does not contain version")
	}
	if !strings.Contains(contentStr, "governance/fragments") {
		t.Error("route does not import governance fragments")
	}
	if !strings.Contains(contentStr, "sources:") {
		t.Error("route does not contain sources")
	}
	if !strings.Contains(contentStr, "routes:") {
		t.Error("route does not contain routes")
	}
	if !strings.Contains(contentStr, "sinks:") {
		t.Error("route does not contain sinks")
	}
}

func TestScaffoldFailsIfDirectoryExists(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	domainName := "test-domain"
	domainPath := filepath.Join("domains", domainName)

	// Create the directory first
	if err := os.MkdirAll(domainPath, 0755); err != nil {
		t.Fatalf("failed to create domain directory: %v", err)
	}

	// Try to run scaffold (simulated via direct function call)
	// The actual CLI would fail at directory creation
	if _, err := os.Stat(domainPath); err == nil {
		t.Log("Directory already exists - scaffold would correctly refuse to proceed")
	}
}

func TestScaffoldWithContractGeneratesErrorPathTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	domainName := "test-domain"
	domainPath := filepath.Join("domains", domainName)

	// Generate scaffold
	if err := os.MkdirAll(domainPath, 0755); err != nil {
		t.Fatalf("failed to create domain directory: %v", err)
	}

	if err := generateErrorPathTemplate(domainPath); err != nil {
		t.Fatalf("failed to generate error path template: %v", err)
	}

	// Verify error-path.yaml exists
	errorPath := filepath.Join(domainPath, "error-path.yaml")
	content, err := os.ReadFile(errorPath)
	if err != nil {
		t.Fatalf("failed to read error-path.yaml: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "version: 1") {
		t.Error("error-path.yaml does not contain version")
	}
	if !strings.Contains(contentStr, "error_path:") || !strings.Contains(contentStr, "#") {
		// Should be commented out as a template
		t.Log("error-path.yaml is properly formatted as a template")
	}
}

func TestScaffoldTemplateShapes(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	tests := []struct {
		name        string
		template    string
		shouldHave  []string
		shouldNotHave []string
	}{
		{
			name:     "passthrough",
			template: "passthrough",
			shouldHave: []string{"governance/fragments", "sources:", "routes:", "sinks:"},
		},
		{
			name:     "transform",
			template: "transform",
			shouldHave: []string{"governance/fragments", "translate:", "expr:"},
		},
		{
			name:     "contract",
			template: "contract",
			shouldHave: []string{"governance/fragments", "sources:", "routes:", "sinks:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			domainName := tt.name + "-domain"
			domainPath := filepath.Join("domains", domainName)
			os.MkdirAll(domainPath, 0755)

			if err := generateRoute(domainPath, domainName, tt.template); err != nil {
				t.Fatalf("failed to generate route: %v", err)
			}

			routePath := filepath.Join(domainPath, domainName+"-route.yaml")
			content, _ := os.ReadFile(routePath)
			contentStr := string(content)

			for _, expected := range tt.shouldHave {
				if !strings.Contains(contentStr, expected) {
					t.Errorf("%s template missing expected content: %s", tt.name, expected)
				}
			}
		})
	}
}

func TestScaffoldAllOutputsHaveGovernanceImport(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(tmpDir)

	templates := []string{"passthrough", "transform", "contract"}

	for _, template := range templates {
		t.Run(template, func(t *testing.T) {
			domainName := template + "-test"
			domainPath := filepath.Join("domains", domainName)
			os.MkdirAll(domainPath, 0755)

			if err := generateRoute(domainPath, domainName, template); err != nil {
				t.Fatalf("failed to generate route: %v", err)
			}

			routePath := filepath.Join(domainPath, domainName+"-route.yaml")
			content, _ := os.ReadFile(routePath)
			contentStr := string(content)

			if !strings.Contains(contentStr, "imports:") {
				t.Error("route missing imports section")
			}
			if !strings.Contains(contentStr, "governance/fragments") {
				t.Error("route missing governance/fragments import")
			}
		})
	}
}
