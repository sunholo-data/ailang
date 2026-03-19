package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo/ailang/internal/pkg"
)

func initPackageCommand(args []string) error {
	flagSet := flag.NewFlagSet("init package", flag.ExitOnError)
	nameFlag := flagSet.String("name", "", "Package name (vendor/name format)")
	helpFlag := flagSet.Bool("help", false, "Show help")

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

	if err := pkg.InitManifest(cwd, name); err != nil {
		return err
	}

	fmt.Printf("%s Created %s\n", green("✓"), pkg.ManifestFile)
	fmt.Printf("  Package: %s\n", cyan(name))
	fmt.Printf("  Version: 0.1.0\n")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit ailang.toml to configure exports and dependencies")
	fmt.Println("  2. Create source files in src/")
	fmt.Printf("  3. Run %s to resolve dependencies\n", bold("ailang lock"))

	return nil
}

func printInitPackageHelp() {
	fmt.Println("Usage: ailang init package [--name vendor/name]")
	fmt.Println()
	fmt.Println("Initialize a new AILANG package in the current directory.")
	fmt.Println("Creates an ailang.toml manifest file.")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --name    Package name in vendor/name format (default: local/<dirname>)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang init package --name sunholo/docparse")
	fmt.Println("  ailang init package")
}
