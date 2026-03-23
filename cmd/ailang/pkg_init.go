package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/pkg"
)

// depSlice implements flag.Value for repeated --dep flags.
type depSlice []string

func (d *depSlice) String() string { return strings.Join(*d, ", ") }
func (d *depSlice) Set(value string) error {
	*d = append(*d, value)
	return nil
}

func initPackageCommand(args []string) error {
	flagSet := flag.NewFlagSet("init package", flag.ExitOnError)
	nameFlag := flagSet.String("name", "", "Package name (vendor/name format)")
	helpFlag := flagSet.Bool("help", false, "Show help")
	var deps depSlice
	flagSet.Var(&deps, "dep", "Add dependency (vendor/name format, repeatable)")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if *helpFlag {
		printInitPackageHelp()
		return nil
	}

	name := *nameFlag

	// If no --name flag, try to derive from directory name
	if name == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		dirName := filepath.Base(cwd)
		// Use a default vendor prefix
		name = "local/" + dirName
	}

	// Validate name format
	parts := strings.SplitN(name, "/", 3)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("package name must be vendor/name format, got %q\nExample: ailang init package --name sunholo/mylib", name)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	if err := pkg.InitManifest(cwd, name, Version); err != nil {
		return err
	}

	fmt.Printf("%s Created %s\n", green("✓"), pkg.ManifestFile)
	fmt.Printf("  Package: %s\n", cyan(name))
	fmt.Printf("  Version: 0.1.0\n")

	// Add dependencies if specified
	for _, dep := range deps {
		// Parse dep as "vendor/name" or "vendor/name@version"
		depName, depVersion := parseDep(dep)
		if err := appendDependencyToFile(cwd, depName, depVersion, false); err != nil {
			return fmt.Errorf("failed to add dependency %s: %w", dep, err)
		}
	}

	fmt.Println()
	if len(deps) == 0 {
		fmt.Println("Next steps:")
		fmt.Println("  1. Edit ailang.toml to configure exports and dependencies")
		fmt.Println("  2. Create source files in src/")
		fmt.Printf("  3. Run %s to resolve dependencies\n", bold("ailang lock"))
	} else {
		fmt.Println("Next steps:")
		fmt.Println("  1. Edit ailang.toml to configure exports")
		fmt.Println("  2. Create source files in src/")
		fmt.Printf("  3. Run %s to resolve dependencies\n", bold("ailang lock"))
	}

	return nil
}

// parseDep splits "vendor/name@version" into name and version parts.
// If no version is specified, defaults to "*".
func parseDep(dep string) (name, version string) {
	if idx := strings.LastIndex(dep, "@"); idx > 0 {
		return dep[:idx], dep[idx+1:]
	}
	return dep, "*"
}

func printInitPackageHelp() {
	fmt.Println("Usage: ailang init package [--name vendor/name] [--dep vendor/name[@version]]...")
	fmt.Println()
	fmt.Println("Initialize a new AILANG package in the current directory.")
	fmt.Println("Creates an ailang.toml manifest file.")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --name    Package name in vendor/name format (default: local/<dirname>)")
	fmt.Println("  --dep     Add a dependency (repeatable). Format: vendor/name or vendor/name@version")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang init package --name sunholo/docparse")
	fmt.Println("  ailang init package --name sunholo/billing_store --dep sunholo/config --dep sunholo/gcp_auth")
	fmt.Println("  ailang init package --dep sunholo/http_helpers@0.2.0")
	fmt.Println("  ailang init package")
}
