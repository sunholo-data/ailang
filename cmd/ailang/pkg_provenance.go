package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// pkgProvenanceCommand shows the provenance chain for a published package version.
// Usage: ailang pkg provenance vendor/name@version
func pkgProvenanceCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ailang pkg provenance <vendor/name@version>")
	}

	name, version, err := parsePackageRef(args[0])
	if err != nil {
		return err
	}

	client := pkg.NewRegistryClient()
	meta, err := client.FetchMetadata(name, version)
	if err != nil {
		return fmt.Errorf("failed to fetch metadata for %s@%s: %w", name, version, err)
	}

	fmt.Printf("Version: %s\n", meta.Version)
	fmt.Printf("Published: %s\n", meta.PublishedAt)
	fmt.Printf("Published By: %s\n", meta.PublishedBy)
	fmt.Println()

	if meta.Provenance == nil {
		fmt.Println("Provenance: none recorded")
		return nil
	}

	p := meta.Provenance
	fmt.Println("Provenance:")
	if p.TriggerMessageID != "" {
		fmt.Printf("  Trigger: %s\n", p.TriggerMessageID)
	}
	if len(p.CorrelationIDs) > 0 {
		fmt.Printf("  Correlation: %v\n", p.CorrelationIDs)
	}
	if p.ChangeClass != "" {
		fmt.Printf("  Change Class: %s\n", p.ChangeClass)
	}
	if p.AutoApproved {
		fmt.Println("  Auto Approved: true")
	} else if p.ApprovedBy != "" {
		fmt.Printf("  Approved By: %s\n", p.ApprovedBy)
		if p.ApprovedAt != "" {
			fmt.Printf("  Approved At: %s\n", p.ApprovedAt)
		}
	}
	if p.AgentTraceID != "" {
		fmt.Printf("  Agent Trace: %s\n", p.AgentTraceID)
	}
	if p.ChainID != "" {
		fmt.Printf("  Chain ID: %s\n", p.ChainID)
	}
	if p.PreviousVersion != "" {
		fmt.Printf("  Previous Version: %s\n", p.PreviousVersion)
	}

	return nil
}

// pkgHistoryCommand shows the version history timeline for a published package version.
// Usage: ailang pkg history vendor/name@version
func pkgHistoryCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ailang pkg history <vendor/name@version>")
	}

	name, version, err := parsePackageRef(args[0])
	if err != nil {
		return err
	}

	// Fetch history.json from registry
	client := pkg.NewRegistryClient()
	historyURL := fmt.Sprintf("%s/packages/%s/%s/history.json", client.BaseURL, name, version)

	resp, err := http.Get(historyURL) //nolint:gosec // URL is derived from trusted registry
	if err != nil {
		return fmt.Errorf("failed to fetch history for %s@%s: %w", name, version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		fmt.Printf("No history recorded for %s@%s\n", name, version)
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry returned %d for history.json", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read history response: %w", err)
	}

	var history pkg.VersionHistory
	if err := json.Unmarshal(body, &history); err != nil {
		return fmt.Errorf("failed to parse history.json: %w", err)
	}

	fmt.Printf("Package: %s\n", history.Package)
	fmt.Printf("Version: %s\n", history.Version)
	if history.Previous != "" {
		fmt.Printf("Previous: %s\n", history.Previous)
	}
	if history.Summary != "" {
		fmt.Printf("Summary: %s\n", history.Summary)
	}
	fmt.Println()

	if len(history.Messages) == 0 {
		fmt.Println("No events recorded.")
		return nil
	}

	fmt.Println("Timeline:")
	for _, entry := range history.Messages {
		status := ""
		if entry.Status != "" {
			status = fmt.Sprintf(" [%s]", entry.Status)
		}
		fmt.Printf("  %s  %-25s %s%s\n", entry.Timestamp, entry.Kind, entry.Title, status)
	}

	return nil
}

// parsePackageRef parses "vendor/name@version" into name and version parts.
func parsePackageRef(ref string) (string, string, error) {
	parts := strings.SplitN(ref, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected format: vendor/name@version, got %q", ref)
	}
	return parts[0], parts[1], nil
}
