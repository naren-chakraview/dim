package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/secrets"
	"github.com/spf13/cobra"
)

// AuditResult represents the result of a secret audit
type AuditResult struct {
	TotalSecrets              int
	DomainScopedSecrets       int
	GlobalSecrets             int
	RoutesNeedingMigration    []RouteMigrationIssue `json:"routes_needing_migration,omitempty"`
	CrossDomainReferences     []CrossDomainRef     `json:"cross_domain_references,omitempty"`
	UnresolvedReferences      []UnresolvedRef      `json:"unresolved_references,omitempty"`
	MigrationReadiness        string               `json:"migration_readiness"`
	RecommendedNextSteps      []string             `json:"recommended_next_steps"`
}

// RouteMigrationIssue represents a route that needs secret migration
type RouteMigrationIssue struct {
	RouteName             string   `json:"route_name"`
	Domain                string   `json:"domain"`
	ImplicitCrossDomain   []string `json:"implicit_cross_domain,omitempty"`
	UnresolvedSecrets     []string `json:"unresolved_secrets,omitempty"`
}

// CrossDomainRef represents an explicit cross-domain secret reference
type CrossDomainRef struct {
	RouteName      string `json:"route_name"`
	SourceDomain   string `json:"source_domain"`
	TargetDomain   string `json:"target_domain"`
	SecretName     string `json:"secret_name"`
}

// UnresolvedRef represents a secret reference that cannot be resolved
type UnresolvedRef struct {
	RouteName  string `json:"route_name"`
	Domain     string `json:"domain"`
	SecretRef  string `json:"secret_ref"`
}

var secretAuditCmd = &cobra.Command{
	Use:   "secret-audit <path>",
	Short: "Audit secrets for domain-scoping readiness",
	Long:  "Scan routes and secrets to identify migration targets and cross-domain dependencies. Helps plan the transition to domain-scoped secrets.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		jsonOutput, _ := cmd.Flags().GetBool("json")

		// Load all route configs from path (file or directory)
		var configs []*config.RouteConfig
		var configPaths []string

		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to access path: %w", err)
		}

		if info.IsDir() {
			// Scan directory for YAML files
			entries, err := os.ReadDir(path)
			if err != nil {
				return fmt.Errorf("failed to read directory: %w", err)
			}

			for _, entry := range entries {
				if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
					continue
				}

				filePath := filepath.Join(path, entry.Name())
				cfg, err := config.LoadRouteConfig(filePath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to load %s: %v\n", filePath, err)
					continue
				}

				configs = append(configs, cfg)
				configPaths = append(configPaths, filePath)
			}
		} else {
			cfg, err := config.LoadRouteConfig(path)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			configs = append(configs, cfg)
			configPaths = append(configPaths, path)
		}

		if len(configs) == 0 {
			return fmt.Errorf("no configuration files found")
		}

		// Initialize secret store
		store := secrets.NewInMemoryStore()

		// Analyze routes
		result := &AuditResult{
			RoutesNeedingMigration: []RouteMigrationIssue{},
			CrossDomainReferences:  []CrossDomainRef{},
			UnresolvedReferences:   []UnresolvedRef{},
		}

		// Scan all routes for secret references
		secretRefs := make(map[string][]string) // route -> list of secret refs
		routeDomains := make(map[string]string)  // route -> domain

		for _, cfg := range configs {
			for routeName, route := range cfg.Routes {
				routeDomains[routeName] = route.Domain

				// Scan route spec for ${SECRET:...} patterns
				refs := extractSecretReferences(route)
				if len(refs) > 0 {
					secretRefs[routeName] = refs
				}
			}
		}

		// Validate secret references
		for routeName, refs := range secretRefs {
			domain := routeDomains[routeName]

			for _, ref := range refs {
				// Check if explicit cross-domain (contains dot)
				refDomain, refName := parseSecretRefSimple(ref)

				if refDomain != "" {
					// Explicit cross-domain reference
					result.CrossDomainReferences = append(result.CrossDomainReferences, CrossDomainRef{
						RouteName:    routeName,
						SourceDomain: domain,
						TargetDomain: refDomain,
						SecretName:   refName,
					})
				} else {
					// Check if resolvable in current domain
					_, err := store.Resolve(domain, ref)
					if err != nil {
						// Unresolved - might be implicit cross-domain or truly missing
						result.UnresolvedReferences = append(result.UnresolvedReferences, UnresolvedRef{
							RouteName: routeName,
							Domain:    domain,
							SecretRef: ref,
						})
					}
				}
			}
		}

		// Build migration issues
		for routeName, refs := range secretRefs {
			domain := routeDomains[routeName]
			var implicitCrossDomain []string
			var unresolved []string

			for _, ref := range refs {
				refDomain, _ := parseSecretRefSimple(ref)
				if refDomain == "" {
					// Check if it's truly resolvable
					_, err := store.Resolve(domain, ref)
					if err != nil {
						unresolved = append(unresolved, ref)
					}
				}
			}

			if len(implicitCrossDomain) > 0 || len(unresolved) > 0 {
				result.RoutesNeedingMigration = append(result.RoutesNeedingMigration, RouteMigrationIssue{
					RouteName:           routeName,
					Domain:              domain,
					ImplicitCrossDomain: implicitCrossDomain,
					UnresolvedSecrets:   unresolved,
				})
			}
		}

		// Determine migration readiness
		if len(result.RoutesNeedingMigration) == 0 && len(result.CrossDomainReferences) == 0 {
			result.MigrationReadiness = "READY"
			result.RecommendedNextSteps = []string{
				"All routes are ready for domain-scoped secrets",
				"Run 'dimctl secret-migrate' to apply domain scoping",
			}
		} else if len(result.RoutesNeedingMigration) > 0 {
			result.MigrationReadiness = "NOT_READY"
			result.RecommendedNextSteps = []string{
				fmt.Sprintf("Fix %d route(s) with unresolved or cross-domain secrets", len(result.RoutesNeedingMigration)),
				"See 'routes_needing_migration' for details",
			}
		} else {
			result.MigrationReadiness = "READY_WITH_SHARED"
			result.RecommendedNextSteps = []string{
				fmt.Sprintf("%d route(s) use explicit cross-domain secret references (audit these)", len(result.CrossDomainReferences)),
				"Verify that cross-domain sharing is intentional",
			}
		}

		// Output results
		if jsonOutput {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			return encoder.Encode(result)
		}

		// Human-readable output
		fmt.Printf("═══════════════════════════════════════════════════════════════\n")
		fmt.Printf("SECRET AUDIT REPORT\n")
		fmt.Printf("═══════════════════════════════════════════════════════════════\n\n")

		fmt.Printf("Scanned:   %d configuration file(s), %d route(s)\n", len(configs), len(secretRefs))
		fmt.Printf("Readiness: %s\n\n", result.MigrationReadiness)

		if len(result.RoutesNeedingMigration) > 0 {
			fmt.Printf("⚠️  ROUTES NEEDING MIGRATION (%d):\n", len(result.RoutesNeedingMigration))
			for _, issue := range result.RoutesNeedingMigration {
				fmt.Printf("  • %s (domain: %s)\n", issue.RouteName, issue.Domain)
				if len(issue.UnresolvedSecrets) > 0 {
					fmt.Printf("    Unresolved: %v\n", issue.UnresolvedSecrets)
				}
				if len(issue.ImplicitCrossDomain) > 0 {
					fmt.Printf("    Implicit cross-domain: %v\n", issue.ImplicitCrossDomain)
				}
			}
			fmt.Printf("\n")
		}

		if len(result.CrossDomainReferences) > 0 {
			fmt.Printf("🔗 CROSS-DOMAIN REFERENCES (%d):\n", len(result.CrossDomainReferences))
			for _, ref := range result.CrossDomainReferences {
				fmt.Printf("  • %s (%s → %s.%s)\n", ref.RouteName, ref.SourceDomain, ref.TargetDomain, ref.SecretName)
			}
			fmt.Printf("\n")
		}

		fmt.Printf("📋 NEXT STEPS:\n")
		for i, step := range result.RecommendedNextSteps {
			fmt.Printf("  %d. %s\n", i+1, step)
		}
		fmt.Printf("\n")

		return nil
	},
}

// extractSecretReferences finds all ${SECRET:...} patterns in a route
func extractSecretReferences(route config.RouteSpec) []string {
	var refs []string
	seenRefs := make(map[string]bool)

	// Check Auth field
	if authStr, ok := route.Auth.(string); ok {
		refs = appendUniqueRefs(refs, extractSecretsFromText(authStr), seenRefs)
	}

	// Check Steps
	for _, step := range route.Steps {
		if step.Filter != nil && step.Filter.Expr != "" {
			refs = appendUniqueRefs(refs, extractSecretsFromText(step.Filter.Expr), seenRefs)
		}
		if step.Translate != nil && step.Translate.Expr != "" {
			refs = appendUniqueRefs(refs, extractSecretsFromText(step.Translate.Expr), seenRefs)
		}
	}

	return refs
}

// extractSecretsFromText finds all ${SECRET:ref} patterns in a string
func extractSecretsFromText(text string) []string {
	var refs []string
	for {
		start := strings.Index(text, "${SECRET:")
		if start == -1 {
			break
		}

		end := strings.Index(text[start:], "}")
		if end == -1 {
			break
		}

		ref := text[start+9 : start+end]
		refs = append(refs, ref)
		text = text[start+end+1:]
	}

	return refs
}

// appendUniqueRefs adds new refs to the list, avoiding duplicates
func appendUniqueRefs(existing []string, new []string, seen map[string]bool) []string {
	for _, ref := range new {
		if !seen[ref] {
			existing = append(existing, ref)
			seen[ref] = true
		}
	}
	return existing
}

// parseSecretRefSimple parses secret reference into domain and name
func parseSecretRefSimple(ref string) (string, string) {
	parts := strings.Split(ref, ".")
	if len(parts) == 1 {
		return "", parts[0]
	}
	return parts[0], strings.Join(parts[1:], ".")
}
