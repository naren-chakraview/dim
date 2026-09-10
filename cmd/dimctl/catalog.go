package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var catalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Discover existing contracts, connections, and data products",
	Long:  "Query Apicurio registry and OpenLineage catalog to discover what you can build on",
}

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search catalog for contracts and data products",
	Args:  cobra.ExactArgs(1),
	RunE:  runSearch,
}

var (
	searchType   string
	searchDomain string
	outputJSON   bool
)

func init() {
	searchCmd.Flags().StringVar(&searchType, "type", "", "Filter by type: contract, product")
	searchCmd.Flags().StringVar(&searchDomain, "domain", "", "Filter by domain")
	searchCmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")

	catalogCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := args[0]

	results, err := Search(context.Background(), query, searchType, searchDomain)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if outputJSON {
		return json.NewEncoder(os.Stdout).Encode(results)
	}

	fmt.Printf("Found %d items matching '%s':\n\n", results.Count, query)
	if results.Count == 0 {
		fmt.Println("No results found.")
		return nil
	}

	for _, r := range results.Results {
		fmt.Printf("  %s/%s [%s]\n", r.Domain, r.Name, r.Type)
		if r.Subject != "" {
			fmt.Printf("    Subject: %s\n", r.Subject)
		}
		fmt.Printf("    Updated: %s\n", r.Updated)
		if r.URL != "" {
			fmt.Printf("    URL: %s\n", r.URL)
		}
	}

	return nil
}
