package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/pkg"
)

func pkgDocsCommand(args []string) error {
	flagSet := flag.NewFlagSet("docs", flag.ExitOnError)
	helpFlag := flagSet.Bool("help", false, "Show help")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if *helpFlag || flagSet.NArg() < 1 {
		fmt.Println("Usage: ailang docs <vendor/name>")
		fmt.Println()
		fmt.Println("Display the AGENT.md usage guide for a package.")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang docs sunholo/auth")
		fmt.Println("  ailang docs sunholo/gcp-auth")
		return nil
	}

	name := flagSet.Arg(0)
	nameParts := strings.SplitN(name, "/", 2)
	if len(nameParts) != 2 {
		return fmt.Errorf("invalid package name: %q (must be vendor/name)", name)
	}

	// Try to find AGENT.md in local cache (any version)
	cacheDir, err := pkg.RegistryCacheDir()
	if err != nil {
		return err
	}

	// Search cache for any version of this package
	pkgCacheDir := filepath.Join(cacheDir, nameParts[0], nameParts[1])
	entries, err := os.ReadDir(pkgCacheDir)
	if err == nil && len(entries) > 0 {
		// Use latest cached version
		latestDir := filepath.Join(pkgCacheDir, entries[len(entries)-1].Name())
		agentMD := filepath.Join(latestDir, "AGENT.md")
		if data, err := os.ReadFile(agentMD); err == nil {
			fmt.Println(string(data))
			return nil
		}
	}

	// Not in registry cache — check git cache
	home, _ := os.UserHomeDir()
	gitCacheBase := filepath.Join(home, ".ailang", "cache", "git")
	if entries, err := os.ReadDir(gitCacheBase); err == nil {
		for _, entry := range entries {
			// Check each git cache dir for this package
			candidate := filepath.Join(gitCacheBase, entry.Name(), "packages", nameParts[1], "AGENT.md")
			if data, err := os.ReadFile(candidate); err == nil {
				fmt.Println(string(data))
				return nil
			}
		}
	}

	// Not cached — try fetching from registry
	client := pkg.NewRegistryClient()
	index, err := client.FetchIndex()
	if err != nil {
		return fmt.Errorf("package %s not found locally. Registry unavailable: %w", name, err)
	}

	// Find package in index to get latest version
	for _, p := range index.Packages {
		if p.Name == name {
			tarball, err := client.FetchPackage(name, p.Latest)
			if err != nil {
				return fmt.Errorf("failed to download %s: %w", name, err)
			}
			// Extract to cache and display AGENT.md
			cachePath, _ := pkg.CachedPackagePath(name, p.Latest)
			os.MkdirAll(cachePath, 0755)
			pkg.ExtractTarball(tarball, cachePath)

			agentMD := filepath.Join(cachePath, "AGENT.md")
			if data, err := os.ReadFile(agentMD); err == nil {
				fmt.Println(string(data))
				return nil
			}
			return fmt.Errorf("package %s@%s has no AGENT.md", name, p.Latest)
		}
	}

	return fmt.Errorf("package %s not found in registry or local cache", name)
}
