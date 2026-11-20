package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/naren-chakraview/dim/internal/config"
	"github.com/naren-chakraview/dim/internal/factory"
	"github.com/spf13/cobra"
)

var sourcesCmd = &cobra.Command{
	Use:   "sources <route-file>",
	Short: "Report whether each configured source is reachable",
	Long: "Load a route configuration and probe each of its sources: a file path, or a " +
		"TCP connection to a broker or host. Each source is reported as reachable, " +
		"unreachable, or unchecked (with the reason). The command exits non-zero if any " +
		"source is unreachable.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadRouteConfig(args[0])
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if len(cfg.Sources) == 0 {
			fmt.Println("no sources configured (routes use the HTTP ingest source)")
			return nil
		}

		timeout, _ := cmd.Flags().GetDuration("timeout")
		statuses := factory.CheckSourceReachability(cfg, timeout)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SOURCE\tTYPE\tROUTES\tREACHABILITY\tDETAIL")
		unreachable := 0
		for _, s := range statuses {
			if s.Reachability == factory.ReachabilityUnreachable {
				unreachable++
			}
			routes := strings.Join(s.Route, ",")
			if routes == "" {
				routes = "(none)"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.Name, s.Type, routes, s.Reachability, s.Detail)
		}
		w.Flush()

		if unreachable > 0 {
			cmd.SilenceUsage = true
			return fmt.Errorf("%d of %d source(s) unreachable", unreachable, len(statuses))
		}
		return nil
	},
}

func init() {
	sourcesCmd.Flags().Duration("timeout", 3*time.Second, "timeout for each TCP probe")
	rootCmd.AddCommand(sourcesCmd)
}
