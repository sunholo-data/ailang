package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/pkg"
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
	cacheDir := config.PackageRoot()
	var err error
	if cacheDir == "" {
		cacheDir, err = pkg.RegistryCacheDir()
	}
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
			printAgentDoc(os.Stdout, data, latestDir)
			return nil
		}
	}

	if config.PackageRoot() != "" {
		return fmt.Errorf("package %s is not provisioned in AILANG_PACKAGE_ROOT at %s; ask the operator to provision it", name, pkgCacheDir)
	}

	// Not in registry cache — check git cache
	gitCacheBase, _ := pkg.UserDir("cache/git") // pure: computes, never creates
	if entries, err := os.ReadDir(gitCacheBase); gitCacheBase != "" && err == nil {
		for _, entry := range entries {
			// Check each git cache dir for this package
			candidate := filepath.Join(gitCacheBase, entry.Name(), "packages", nameParts[1], "AGENT.md")
			if data, err := os.ReadFile(candidate); err == nil {
				printAgentDoc(os.Stdout, data, filepath.Dir(candidate))
				return nil
			}
		}
	}

	if err := pkg.RefuseIfConfined("fetching from the registry"); err != nil {
		return fmt.Errorf("package %s not provisioned: %w", name, err)
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
			if _, err := pkg.EnsureRegistryCacheDir(); err != nil {
				return err
			}
			cachePath, err := pkg.CachedPackagePath(name, p.Latest)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(cachePath, 0755); err != nil {
				return err
			}
			if err := pkg.ExtractTarball(tarball, cachePath); err != nil {
				return err
			}

			agentMD := filepath.Join(cachePath, "AGENT.md")
			if data, err := os.ReadFile(agentMD); err == nil {
				printAgentDoc(os.Stdout, data, cachePath)
				return nil
			}
			return fmt.Errorf("package %s@%s has no AGENT.md", name, p.Latest)
		}
	}

	return fmt.Errorf("package %s not found in registry or local cache", name)
}

// printAgentDoc prints a package's AGENT.md followed by the canonical import
// line for each exported module, read from the package's own ailang.toml.
// The bare module_prefix spelling also resolves, but the loader lets several
// dependencies share a prefix; only pkg/<vendor>/<name>/... names exactly one
// package (M-PKG-PARALLEL-AGENTS M6). A missing or unreadable manifest prints
// the guide alone.
func printAgentDoc(w io.Writer, agentDoc []byte, pkgDir string) {
	fmt.Fprintln(w, string(agentDoc))
	m, err := pkg.LoadManifest(pkgDir)
	if err != nil || len(m.Exports.Modules) == 0 {
		return
	}
	fmt.Fprintln(w, "## Imports (canonical)")
	fmt.Fprintln(w)
	for _, mod := range m.Exports.Modules {
		fmt.Fprintf(w, "import pkg/%s\n", m.MapModuleToImportPath(mod))
	}
}
