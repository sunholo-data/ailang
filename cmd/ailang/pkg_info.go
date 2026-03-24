package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultRegistryAPI = "https://ailang-registry-validator-mdpoxgrptq-ew.a.run.app"

func registryAPIURL() string {
	if url := os.Getenv("AILANG_REGISTRY_API"); url != "" {
		return strings.TrimRight(url, "/")
	}
	return defaultRegistryAPI
}

func registryAPIGet(path string) ([]byte, error) {
	url := registryAPIURL() + path
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url) //nolint:gosec // URL from trusted source
	if err != nil {
		return nil, fmt.Errorf("registry API unavailable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry API returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// pkgInfoCommand shows detailed information about a package.
// Usage: ailang pkg info vendor/name
func pkgInfoCommand(args []string) error {
	if len(args) == 0 || args[0] == "--help" {
		fmt.Println("Usage: ailang pkg info <vendor/name>")
		fmt.Println()
		fmt.Println("Show detailed package information from the registry API.")
		fmt.Println("Displays manifest, versions, dependents, and hashes.")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang pkg info sunholo/auth")
		fmt.Println("  ailang pkg info sunholo/firestore")
		return nil
	}

	name := args[0]
	if !strings.Contains(name, "/") {
		return fmt.Errorf("expected vendor/name format, got %q", name)
	}

	data, err := registryAPIGet("/api/packages/" + name)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", name, err)
	}

	var detail struct {
		Index struct {
			Name       string   `json:"name"`
			Latest     string   `json:"latest"`
			Versions   []string `json:"versions"`
			AISummary  string   `json:"ai_summary"`
			Tags       []string `json:"tags"`
			Effects    []string `json:"effects"`
			Stability  string   `json:"stability"`
			Exports    []string `json:"exports"`
			HasAgent   bool     `json:"has_agent_doc"`
			Deps       []string `json:"dependencies"`
			LastUpdate string   `json:"last_updated"`
			UpdatedBy  string   `json:"updated_by"`
		} `json:"index"`
		Versions []struct {
			Version  string `json:"version"`
			Metadata *struct {
				PublishedAt string `json:"published_at"`
				PublishedBy string `json:"published_by"`
				ContentHash string `json:"content_hash"`
				InterfHash  string `json:"interface_hash"`
				TarballHash string `json:"tarball_hash"`
				TarballSize int64  `json:"tarball_size_bytes"`
				Validation  struct {
					Compiles          bool   `json:"compiles"`
					EffectsValid      bool   `json:"effects_valid"`
					ContractsVerified int    `json:"contracts_verified"`
					ContractsTotal    int    `json:"contracts_total"`
					AILANGVersion     string `json:"ailang_version"`
				} `json:"validation"`
			} `json:"metadata,omitempty"`
		} `json:"versions"`
		Dependents []string `json:"dependents"`
	}

	if err := json.Unmarshal(data, &detail); err != nil {
		return fmt.Errorf("invalid response: %w", err)
	}

	idx := detail.Index

	// Header
	effects := "Pure"
	if len(idx.Effects) > 0 {
		effects = strings.Join(idx.Effects, ", ")
	}
	fmt.Printf("%s %s\n", bold(idx.Name), cyan("v"+idx.Latest))
	fmt.Printf("%s\n\n", idx.AISummary)

	// Manifest
	fmt.Printf("  Stability:  %s\n", idx.Stability)
	fmt.Printf("  Effects:    %s\n", effects)
	if len(idx.Tags) > 0 {
		fmt.Printf("  Tags:       %s\n", strings.Join(idx.Tags, ", "))
	}
	fmt.Printf("  AGENT.md:   %v\n", idx.HasAgent)
	fmt.Println()

	// Exports
	fmt.Printf("  %s\n", bold("Exports"))
	for _, exp := range idx.Exports {
		fmt.Printf("    %s %s\n", cyan("›"), exp)
	}
	fmt.Println()

	// Dependencies
	if len(idx.Deps) > 0 {
		fmt.Printf("  %s\n", bold("Dependencies"))
		for _, dep := range idx.Deps {
			fmt.Printf("    %s %s\n", cyan("→"), dep)
		}
		fmt.Println()
	}

	// Dependents
	if len(detail.Dependents) > 0 {
		fmt.Printf("  %s\n", bold("Used By"))
		for _, dep := range detail.Dependents {
			fmt.Printf("    %s %s\n", cyan("←"), dep)
		}
		fmt.Println()
	}

	// Versions
	fmt.Printf("  %s (%d versions)\n", bold("Versions"), len(detail.Versions))
	for i := len(detail.Versions) - 1; i >= 0; i-- {
		v := detail.Versions[i]
		latest := ""
		if v.Version == idx.Latest {
			latest = green(" (latest)")
		}
		if v.Metadata != nil {
			pubDate := v.Metadata.PublishedAt
			if t, err := time.Parse(time.RFC3339, pubDate); err == nil {
				pubDate = t.Format("2006-01-02")
			}
			sizeKB := fmt.Sprintf("%.1f KB", float64(v.Metadata.TarballSize)/1024)
			fmt.Printf("    %s  %s  %s%s\n", cyan("v"+v.Version), pubDate, sizeKB, latest)
		} else {
			fmt.Printf("    %s%s\n", cyan("v"+v.Version), latest)
		}
	}

	// Browse link
	fmt.Printf("\n  %s https://ailang.sunholo.com/docs/packages/%s\n", bold("Browse:"), idx.Name)

	return nil
}

// pkgVersionsCommand shows all versions of a package with hashes.
// Usage: ailang pkg versions vendor/name
func pkgVersionsCommand(args []string) error {
	if len(args) == 0 || args[0] == "--help" {
		fmt.Println("Usage: ailang pkg versions <vendor/name>")
		fmt.Println()
		fmt.Println("Show all published versions with hashes and validation status.")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang pkg versions sunholo/auth")
		return nil
	}

	name := args[0]
	if !strings.Contains(name, "/") {
		return fmt.Errorf("expected vendor/name format, got %q", name)
	}

	data, err := registryAPIGet("/api/packages/" + name)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", name, err)
	}

	var detail struct {
		Index struct {
			Latest string `json:"latest"`
		} `json:"index"`
		Versions []struct {
			Version  string `json:"version"`
			Metadata *struct {
				PublishedAt string `json:"published_at"`
				PublishedBy string `json:"published_by"`
				ContentHash string `json:"content_hash"`
				InterfHash  string `json:"interface_hash"`
				TarballHash string `json:"tarball_hash"`
				TarballSize int64  `json:"tarball_size_bytes"`
				Validation  struct {
					Compiles          bool `json:"compiles"`
					EffectsValid      bool `json:"effects_valid"`
					ContractsVerified int  `json:"contracts_verified"`
					ContractsTotal    int  `json:"contracts_total"`
				} `json:"validation"`
			} `json:"metadata,omitempty"`
		} `json:"versions"`
	}

	if err := json.Unmarshal(data, &detail); err != nil {
		return fmt.Errorf("invalid response: %w", err)
	}

	fmt.Printf("%s — %d version(s)\n\n", bold(name), len(detail.Versions))

	for i := len(detail.Versions) - 1; i >= 0; i-- {
		v := detail.Versions[i]
		latest := ""
		if v.Version == detail.Index.Latest {
			latest = green(" ← latest")
		}
		fmt.Printf("%s%s\n", bold("v"+v.Version), latest)

		if v.Metadata != nil {
			m := v.Metadata
			pubDate := m.PublishedAt
			if t, err := time.Parse(time.RFC3339, pubDate); err == nil {
				pubDate = t.Format("2006-01-02 15:04")
			}
			fmt.Printf("  Published:  %s", pubDate)
			if m.PublishedBy != "" {
				fmt.Printf(" by %s", m.PublishedBy)
			}
			fmt.Println()
			fmt.Printf("  Size:       %.1f KB\n", float64(m.TarballSize)/1024)

			// Validation
			compileIcon := green("✓")
			if !m.Validation.Compiles {
				compileIcon = red("✗")
			}
			effectIcon := green("✓")
			if !m.Validation.EffectsValid {
				effectIcon = red("✗")
			}
			fmt.Printf("  Validation: %s compiles  %s effects  %d/%d contracts\n",
				compileIcon, effectIcon, m.Validation.ContractsVerified, m.Validation.ContractsTotal)

			// Hashes
			fmt.Printf("  Content:    %s\n", truncHash(m.ContentHash))
			fmt.Printf("  Interface:  %s\n", truncHash(m.InterfHash))
			fmt.Printf("  Tarball:    %s\n", truncHash(m.TarballHash))
		}
		fmt.Println()
	}

	return nil
}

// pkgStatsCommand shows ecosystem-wide registry statistics.
// Usage: ailang pkg stats
func pkgStatsCommand(args []string) error {
	if len(args) > 0 && args[0] == "--help" {
		fmt.Println("Usage: ailang pkg stats")
		fmt.Println()
		fmt.Println("Show ecosystem-wide statistics from the package registry.")
		fmt.Println("Includes package counts, effect distribution, dependency depth, etc.")
		return nil
	}

	data, err := registryAPIGet("/api/stats")
	if err != nil {
		return fmt.Errorf("failed to fetch stats: %w", err)
	}

	var stats struct {
		TotalPackages  int            `json:"total_packages"`
		TotalVersions  int            `json:"total_versions"`
		EffectDist     map[string]int `json:"effect_distribution"`
		StabilityBreak map[string]int `json:"stability_breakdown"`
		DepDepthMax    int            `json:"dependency_depth_max"`
		AvgExports     float64        `json:"avg_exports_per_package"`
		PassRate       float64        `json:"validation_pass_rate"`
		TopDeps        []struct {
			Name  string `json:"name"`
			Count int    `json:"dependent_count"`
		} `json:"top_depended_on"`
		PurePackages int `json:"pure_packages"`
		AgentVsHuman struct {
			Agent int `json:"agent"`
			Human int `json:"human"`
		} `json:"agent_vs_human"`
	}

	if err := json.Unmarshal(data, &stats); err != nil {
		return fmt.Errorf("invalid response: %w", err)
	}

	fmt.Printf("%s\n\n", bold("AILANG Package Registry — Ecosystem Stats"))

	// Overview
	fmt.Printf("  Packages:      %s\n", cyan(fmt.Sprintf("%d", stats.TotalPackages)))
	fmt.Printf("  Versions:      %s\n", cyan(fmt.Sprintf("%d", stats.TotalVersions)))
	fmt.Printf("  Pure (no fx):  %s\n", green(fmt.Sprintf("%d", stats.PurePackages)))
	fmt.Printf("  Avg Exports:   %.1f per package\n", stats.AvgExports)
	fmt.Printf("  Max Dep Depth: %d\n", stats.DepDepthMax)
	fmt.Printf("  Validation:    %s pass rate\n", green(fmt.Sprintf("%.0f%%", stats.PassRate*100)))
	fmt.Println()

	// Effect distribution
	if len(stats.EffectDist) > 0 {
		fmt.Printf("  %s\n", bold("Effect Distribution"))
		for effect, count := range stats.EffectDist {
			bar := strings.Repeat("█", count)
			fmt.Printf("    %-4s %s %d\n", effect, cyan(bar), count)
		}
		fmt.Println()
	}

	// Stability breakdown
	if len(stats.StabilityBreak) > 0 {
		fmt.Printf("  %s\n", bold("Stability"))
		for stability, count := range stats.StabilityBreak {
			fmt.Printf("    %-14s %d\n", stability, count)
		}
		fmt.Println()
	}

	// Top depended-on
	if len(stats.TopDeps) > 0 {
		fmt.Printf("  %s\n", bold("Most Depended On"))
		for _, dep := range stats.TopDeps {
			fmt.Printf("    %s %s (%d dependents)\n", cyan("→"), dep.Name, dep.Count)
		}
		fmt.Println()
	}

	// Agent vs human
	if stats.AgentVsHuman.Agent > 0 || stats.AgentVsHuman.Human > 0 {
		fmt.Printf("  %s\n", bold("Updates By"))
		fmt.Printf("    Agent: %d   Human: %d\n", stats.AgentVsHuman.Agent, stats.AgentVsHuman.Human)
		fmt.Println()
	}

	fmt.Printf("  %s https://ailang.sunholo.com/docs/packages/explorer\n", bold("Browse:"))

	return nil
}

func truncHash(hash string) string {
	if len(hash) > 20 {
		return hash[:20] + "..."
	}
	return hash
}
