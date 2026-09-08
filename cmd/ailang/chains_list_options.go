package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/sunholo-data/ailang/internal/observatory"
)

type chainsListFlags struct {
	Query   observatory.ChainListOptions
	Remote  string
	JSON    bool
	FullIDs bool
}

// parseChainsListFlags validates the complete request before opening any backend.
func parseChainsListFlags(args []string, output io.Writer) (chainsListFlags, error) {
	var options chainsListFlags
	fs := flag.NewFlagSet("chains list", flag.ContinueOnError)
	fs.SetOutput(output)
	status := fs.String("status", "", "Filter by status (active, pending_approval, completed, failed)")
	fs.StringVar(&options.Query.SourceType, "source", "", "Filter by source type (github_issue, message, manual)")
	fs.StringVar(&options.Query.AgentID, "agent", "", "Filter by agent ID (e.g., design-doc-creator)")
	fs.StringVar(&options.Query.WorkspaceID, "workspace", "", "Filter by workspace ID")
	fs.StringVar(&options.Query.GitHubRepo, "repo", "", "Filter by GitHub repository (owner/repo)")
	since := fs.String("since", "", "Show chains created after (e.g., 24h, 7d, 2026-02-01)")
	fs.IntVar(&options.Query.Limit, "limit", 20, "Maximum number of chains to show (positive)")
	fs.IntVar(&options.Query.Offset, "offset", 0, "Skip this many matching chains (non-negative; newest first)")
	fs.BoolVar(&options.JSON, "json", false, "Output as JSON")
	fs.BoolVar(&options.FullIDs, "full", false, "Show full chain IDs (for copy-paste)")
	fs.StringVar(&options.Remote, "remote", "", "Read from this observatory storage mode (gcp). Default: $AILANG_CHAINS_READ")
	if err := fs.Parse(args); err != nil {
		return options, err
	}
	if fs.NArg() != 0 {
		return options, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	if options.Query.Limit <= 0 {
		return options, fmt.Errorf("--limit must be positive")
	}
	if options.Query.Offset < 0 {
		return options, fmt.Errorf("--offset must be non-negative")
	}
	options.Query.Status = observatory.ChainStatus(*status)
	if *since != "" {
		createdAfter, err := parseSinceFlag(*since)
		if err != nil {
			return options, fmt.Errorf("invalid --since value %q: %w", *since, err)
		}
		options.Query.CreatedAfter = &createdAfter
	}
	return options, nil
}
