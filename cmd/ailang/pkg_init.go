package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/pkg"
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
	modulePrefixFlag := flagSet.String("module-prefix", "", "Module prefix mapping (e.g., 'docparse' if modules use 'docparse/...')")
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

	// Add module_prefix if specified
	if *modulePrefixFlag != "" {
		if strings.Contains(*modulePrefixFlag, "/") {
			return fmt.Errorf("module-prefix must be a single segment (no slashes), got %q", *modulePrefixFlag)
		}
		if err := appendModulePrefixToFile(cwd, *modulePrefixFlag); err != nil {
			return fmt.Errorf("failed to add module_prefix: %w", err)
		}
	}

	fmt.Printf("%s Created %s\n", green("✓"), pkg.ManifestFile)
	fmt.Printf("  Package: %s\n", cyan(name))
	if *modulePrefixFlag != "" {
		fmt.Printf("  Module prefix: %s\n", cyan(*modulePrefixFlag))
	}
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

// appendModulePrefixToFile adds module_prefix to the [package] section of ailang.toml.
func appendModulePrefixToFile(dir, prefix string) error {
	path := filepath.Join(dir, pkg.ManifestFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	// Insert module_prefix after the name line
	content = strings.Replace(content,
		"\nversion = ",
		fmt.Sprintf("\nmodule_prefix = %q\nversion = ", prefix),
		1)
	return os.WriteFile(path, []byte(content), 0644)
}

func printInitPackageHelp() {
	fmt.Println("Usage: ailang init package [--name vendor/name] [--module-prefix prefix] [--dep vendor/name[@version]]...")
	fmt.Println()
	fmt.Println("Initialize a new AILANG package in the current directory.")
	fmt.Println("Creates an ailang.toml manifest file.")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --name           Package name in vendor/name format (default: local/<dirname>)")
	fmt.Println("  --module-prefix  Map existing module paths (e.g., 'docparse' if modules use 'docparse/...')")
	fmt.Println("  --dep            Add a dependency (repeatable). Format: vendor/name or vendor/name@version")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang init package --name sunholo/docparse --module-prefix docparse")
	fmt.Println("  ailang init package --name sunholo/billing_store --dep sunholo/config --dep sunholo/gcp_auth")
	fmt.Println("  ailang init package --dep sunholo/http_helpers@0.2.0")
	fmt.Println("  ailang init package")
}
