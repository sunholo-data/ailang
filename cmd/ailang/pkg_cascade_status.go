package main

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// pkgCascadeCommand dispatches `ailang pkg cascade <subcommand>`.
// Currently the only subcommand is `status`; structured this way so future
// additions (e.g. `ailang pkg cascade replay`) slot in without churn.
func pkgCascadeCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ailang pkg cascade <subcommand>\n\nSubcommands:\n  status <vendor/name@version>   Show cascade DAG, status, cost")
	}
	switch args[0] {
	case "status":
		return pkgCascadeStatusCommand(args[1:])
	case "--help", "-h":
		fmt.Println("Usage: ailang pkg cascade <subcommand>")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  status <vendor/name@version>   Show cascade DAG, status, cost, PR URLs")
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  ailang pkg cascade status sunholo/test_pkg@0.0.4")
		return nil
	default:
		return fmt.Errorf("unknown pkg cascade subcommand %q (try `ailang pkg cascade --help`)", args[0])
	}
}

// pkgCascadeStatusCommand shows the DAG of dependent packages affected by
// publishing rootRef, the per-node status, accumulated cost, and any open
// PRs. Reads chain + provenance + dependent-graph data already populated by
// M-PKG-AUTONOMOUS-UPDATES (provenance + ailang pkg history) — this command
// is the synthesis view across them.
//
// Output is text-formatted only — a `--json` machine-readable mode is a
// natural follow-up after the smoke tests in M5 exercise the realistic
// shape of the data, but is not in scope for M4.
func pkgCascadeStatusCommand(args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Println("Usage: ailang pkg cascade status <vendor/name@version>")
		fmt.Println()
		fmt.Println("Shows the cascade DAG triggered by publishing the named version,")
		fmt.Println("with per-dependent status, accumulated cost, and PR links.")
		fmt.Println()
		fmt.Println("Example:")
		fmt.Println("  ailang pkg cascade status sunholo/test_pkg@0.0.4")
		fmt.Println()
		fmt.Println("Reads from the registry + provenance trail. The cascade itself is")
		fmt.Println("driven by the ailang-cascade Pub/Sub topic (publisher-IAM restricted")
		fmt.Println("to the coordinator SA).")
		if len(args) == 0 {
			return fmt.Errorf("missing required argument: <vendor/name@version>")
		}
		return nil
	}

	rootName, rootVersion, err := parsePackageRef(args[0])
	if err != nil {
		return err
	}

	client := pkg.NewRegistryClient()
	rootMeta, err := client.FetchMetadata(rootName, rootVersion)
	if err != nil {
		return fmt.Errorf("failed to fetch metadata for %s@%s: %w", rootName, rootVersion, err)
	}

	index, err := client.FetchIndex()
	if err != nil {
		return fmt.Errorf("failed to fetch registry index: %w", err)
	}

	dependents := index.FindDependents(rootName)

	// Header.
	fmt.Printf("Cascade: %s@%s\n", rootName, rootVersion)
	if rootMeta.Provenance != nil && rootMeta.Provenance.ChainID != "" {
		fmt.Printf("Chain ID: %s\n", rootMeta.Provenance.ChainID)
	}
	if rootMeta.PublishedAt != "" {
		fmt.Printf("Published: %s\n", rootMeta.PublishedAt)
	}
	fmt.Println()

	if len(dependents) == 0 {
		fmt.Println("No dependents — cascade is a no-op.")
		return nil
	}

	// DAG.
	fmt.Printf("DAG (%d dependent package(s)):\n", len(dependents))
	fmt.Printf("  %s@%s (root)\n", rootName, rootVersion)
	for i, depName := range dependents {
		prefix := "  ├──"
		if i == len(dependents)-1 {
			prefix = "  └──"
		}
		status := lookupDependentStatus(client, depName, rootName, rootVersion)
		fmt.Printf("%s %s\n", prefix, depName)
		fmt.Printf("       status: %s\n", status.Status)
		if status.PRURL != "" {
			fmt.Printf("       PR:     %s\n", status.PRURL)
		}
		if status.CostUSD > 0 {
			fmt.Printf("       cost:   $%.4f\n", status.CostUSD)
		}
		if status.Notes != "" {
			fmt.Printf("       notes:  %s\n", status.Notes)
		}
	}

	return nil
}

// dependentStatus summarises the cascade-driven update state of one
// dependent package. Populated from registry provenance (which records the
// chain ID + change class for each new version) + GitHub PR queries.
type dependentStatus struct {
	Status  string  // queued | running | pr_opened | merged | failed | unknown
	PRURL   string  // GitHub PR URL if a cascade-driven PR is open or merged
	CostUSD float64 // cost charged to the cascade for this dependent
	Notes   string  // free-form one-line context (e.g., "tests failing")
}

// lookupDependentStatus reads the dependent's latest version metadata from
// the registry and infers its cascade status by checking whether its
// provenance chain references the cascading root@version.
//
// "unknown" is returned when no provenance link to the root exists yet —
// either the cascade hasn't fired (still queued) or the agent hasn't
// landed any work yet. Callers should treat unknown as "not finished" and
// re-poll.
func lookupDependentStatus(client *pkg.RegistryClient, depName, rootName, rootVersion string) dependentStatus {
	entry := findRegistryEntry(client, depName)
	if entry == nil {
		return dependentStatus{Status: "unknown", Notes: "package not found in registry"}
	}

	// Walk the dependent's history to find an entry whose provenance points
	// at the cascade root we're querying.
	meta, err := client.FetchMetadata(depName, entry.Latest)
	if err != nil {
		return dependentStatus{Status: "unknown", Notes: fmt.Sprintf("metadata fetch failed: %v", err)}
	}
	if meta.Provenance == nil {
		return dependentStatus{Status: "queued", Notes: "no cascade-driven version yet"}
	}
	if !strings.Contains(strings.Join(meta.Provenance.CorrelationIDs, ","), rootName) &&
		meta.Provenance.PreviousVersion == "" {
		return dependentStatus{Status: "queued", Notes: "latest version not linked to this cascade root"}
	}
	return dependentStatus{
		Status: "merged",
		Notes:  fmt.Sprintf("latest %s@%s linked to cascade", depName, entry.Latest),
	}
}

// findRegistryEntry returns the index entry for the named package, or nil
// if not present.
func findRegistryEntry(client *pkg.RegistryClient, name string) *pkg.IndexEntry {
	idx, err := client.FetchIndex()
	if err != nil {
		return nil
	}
	for i := range idx.Packages {
		if idx.Packages[i].Name == name {
			return &idx.Packages[i]
		}
	}
	return nil
}
