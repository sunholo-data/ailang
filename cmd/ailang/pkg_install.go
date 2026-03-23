package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/pkg"
)

func pkgInstallCommand(args []string) error {
	flagSet := flag.NewFlagSet("install", flag.ExitOnError)
	helpFlag := flagSet.Bool("help", false, "Show help")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if *helpFlag || flagSet.NArg() < 1 {
		fmt.Println("Usage: ailang install <vendor/name@version>")
		fmt.Println()
		fmt.Println("Download a package from the registry, verify its hash, and add it to ailang.toml.")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang install sunholo/auth@0.1.0")
		fmt.Println("  ailang install sunholo/gcp-auth@0.1.0")
		fmt.Println()
		fmt.Println("Registry: $AILANG_REGISTRY (default: https://storage.googleapis.com/ailang-registry)")
		return nil
	}

	spec := flagSet.Arg(0)
	parts := strings.SplitN(spec, "@", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid package spec: %q (must be vendor/name@version)\nExample: ailang install sunholo/auth@0.1.0", spec)
	}
	name, version := parts[0], parts[1]

	// Validate name format
	nameParts := strings.SplitN(name, "/", 2)
	if len(nameParts) != 2 {
		return fmt.Errorf("invalid package name: %q (must be vendor/name)", name)
	}

	client := pkg.NewRegistryClient()

	// Download metadata for hash verification
	fmt.Printf("Fetching %s@%s from %s...\n", name, version, client.BaseURL)
	meta, err := client.FetchMetadata(name, version)
	if err != nil {
		return fmt.Errorf("failed to fetch package metadata: %w", err)
	}

	// Check AILANG version compatibility
	if meta.Manifest.AILANG != "" {
		ok, checkErr := pkg.SatisfiesAILANGVersion(meta.Manifest.AILANG, Version)
		if checkErr != nil {
			fmt.Fprintf(os.Stderr, "%s version check: %v\n", yellow("⚠"), checkErr)
		} else if !ok {
			return fmt.Errorf("%s@%s requires AILANG %s (you have %s)\n\n  Options:\n    1. Upgrade AILANG:  go install github.com/sunholo/ailang@latest\n    2. Use older version: ailang install %s@<older-version>",
				name, version, meta.Manifest.AILANG, Version, name)
		} else {
			fmt.Printf("  AILANG compatibility: %s (you have %s) %s\n", meta.Manifest.AILANG, Version, green("✓"))
		}
	}

	// Download tarball
	tarballData, err := client.FetchPackage(name, version)
	if err != nil {
		return err
	}

	// Verify tarball hash
	actualHash := pkg.TarballHash(tarballData)
	if meta.TarballHash != "" && actualHash != meta.TarballHash {
		return fmt.Errorf("tarball hash mismatch for %s@%s: expected %s, got %s\nThe package may have been tampered with", name, version, meta.TarballHash, actualHash)
	}

	// Extract to cache
	cachePath, err := pkg.CachedPackagePath(name, version)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return fmt.Errorf("failed to create cache dir: %w", err)
	}

	if err := pkg.ExtractTarball(tarballData, cachePath); err != nil {
		return fmt.Errorf("failed to extract package: %w", err)
	}

	fmt.Printf("%s Downloaded %s@%s (%d bytes)\n", green("✓"), name, version, len(tarballData))

	// Add to ailang.toml if we're in a package project
	cwd, err := os.Getwd()
	if err != nil {
		return nil // non-fatal
	}

	if _, err := pkg.LoadManifest(cwd); err == nil {
		// We're in a package project — add the dep
		return appendDependencyToFile(cwd, name, version, false)
	}

	fmt.Println()
	fmt.Println("To use this package, add to your ailang.toml:")
	fmt.Printf("  \"%s\" = %q\n", name, version)
	fmt.Println("Then run: ailang lock")

	return nil
}
