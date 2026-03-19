package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/pkg"
)

func pkgAddCommand(args []string) error {
	flagSet := flag.NewFlagSet("add", flag.ExitOnError)
	pathFlag := flagSet.Bool("path", false, "Add as path dependency")
	helpFlag := flagSet.Bool("help", false, "Show help")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if *helpFlag || flagSet.NArg() < 1 {
		fmt.Println("Usage: ailang add <dependency> [--path]")
		fmt.Println()
		fmt.Println("Add a dependency to ailang.toml.")
		fmt.Println()
		fmt.Println("Flags:")
		fmt.Println("  --path    Add as a path dependency (local directory)")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang add ../shared/json --path")
		fmt.Println("  ailang add sunholo/json@0.3.1")
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	manifest, err := pkg.LoadManifest(cwd)
	if err != nil {
		return fmt.Errorf("no ailang.toml found in current directory: %w\nRun 'ailang init package' first", err)
	}

	arg := flagSet.Arg(0)

	if *pathFlag {
		return addPathDep(cwd, manifest, arg)
	}
	return addVersionDep(cwd, manifest, arg)
}

func addPathDep(cwd string, manifest *pkg.PackageManifest, depPath string) error {
	// Load the dependency's manifest to get its name
	depManifest, err := pkg.LoadManifest(depPath)
	if err != nil {
		// Try relative to cwd
		absPath := depPath
		if !strings.HasPrefix(depPath, "/") {
			absPath = cwd + "/" + depPath
		}
		depManifest, err = pkg.LoadManifest(absPath)
		if err != nil {
			return fmt.Errorf("no ailang.toml found at %s: %w", depPath, err)
		}
	}

	name := depManifest.Package.Name

	// Check for duplicate
	if _, exists := manifest.Dependencies[name]; exists {
		return fmt.Errorf("dependency %s already exists in ailang.toml", name)
	}

	// Re-read the manifest file to append the dependency
	return appendDependencyToFile(cwd, name, depPath, true)
}

func addVersionDep(cwd string, manifest *pkg.PackageManifest, spec string) error {
	// Parse name@version
	parts := strings.SplitN(spec, "@", 2)
	if len(parts) != 2 {
		return fmt.Errorf("version dependency must be name@version format, got %q\nExample: ailang add sunholo/json@0.3.1", spec)
	}
	name, version := parts[0], parts[1]

	// Check for duplicate
	if _, exists := manifest.Dependencies[name]; exists {
		return fmt.Errorf("dependency %s already exists in ailang.toml", name)
	}

	return appendDependencyToFile(cwd, name, version, false)
}

// appendDependencyToFile appends a dependency line to ailang.toml.
func appendDependencyToFile(dir, name, value string, isPath bool) error {
	path := dir + "/" + pkg.ManifestFile
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)

	var depLine string
	if isPath {
		depLine = fmt.Sprintf("\"%s\" = { path = %q }\n", name, value)
	} else {
		depLine = fmt.Sprintf("\"%s\" = %q\n", name, value)
	}

	// Find [dependencies] section and append
	if strings.Contains(content, "[dependencies]") {
		// Insert after [dependencies] line
		idx := strings.Index(content, "[dependencies]")
		lineEnd := strings.Index(content[idx:], "\n")
		insertAt := idx + lineEnd + 1
		content = content[:insertAt] + depLine + content[insertAt:]
	} else {
		// Add new [dependencies] section
		content += "\n[dependencies]\n" + depLine
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}

	label := value
	if isPath {
		label = fmt.Sprintf("path: %s", value)
	}
	fmt.Printf("%s Added %s (%s)\n", green("✓"), name, label)
	return nil
}

func pkgLockCommand(args []string) error {
	flagSet := flag.NewFlagSet("lock", flag.ExitOnError)
	helpFlag := flagSet.Bool("help", false, "Show help")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if *helpFlag {
		fmt.Println("Usage: ailang lock")
		fmt.Println()
		fmt.Println("Resolve dependencies and generate ailang.lock.")
		fmt.Println("Reads ailang.toml and writes a deterministic lock file")
		fmt.Println("with content hashes for all resolved packages.")
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	manifest, err := pkg.LoadManifest(cwd)
	if err != nil {
		return fmt.Errorf("no ailang.toml found: %w\nRun 'ailang init package' first", err)
	}

	if len(manifest.Dependencies) == 0 {
		fmt.Printf("%s No dependencies to resolve\n", yellow("⚠"))
		// Still create an empty lock file for consistency
		lf := pkg.NewLockFile(nil, fmt.Sprintf("ailang lock %s", Version))
		if err := lf.Save(cwd); err != nil {
			return fmt.Errorf("failed to write lock file: %w", err)
		}
		fmt.Printf("%s Generated %s (0 packages)\n", green("✓"), pkg.LockFileName)
		return nil
	}

	resolved, err := pkg.ResolveDependencies(manifest, cwd)
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %w", err)
	}

	// Convert to LockedPackages
	locked := make([]pkg.LockedPackage, len(resolved))
	for i, r := range resolved {
		locked[i] = pkg.LockedPackage(r)
	}

	lf := pkg.NewLockFile(locked, fmt.Sprintf("ailang lock %s", Version))
	if err := lf.Save(cwd); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	fmt.Printf("%s Generated %s (%d packages)\n", green("✓"), pkg.LockFileName, len(resolved))
	for _, r := range resolved {
		source := r.Source
		if r.Path != "" {
			source = "path: " + r.Path
		}
		fmt.Printf("  %s %s@%s (%s)\n", cyan("→"), r.Name, r.Version, source)
	}

	return nil
}

func pkgTreeCommand(args []string) error {
	flagSet := flag.NewFlagSet("tree", flag.ExitOnError)
	helpFlag := flagSet.Bool("help", false, "Show help")

	if err := flagSet.Parse(args); err != nil {
		return err
	}

	if *helpFlag {
		fmt.Println("Usage: ailang tree")
		fmt.Println()
		fmt.Println("Display the dependency tree for the current package.")
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	manifest, err := pkg.LoadManifest(cwd)
	if err != nil {
		return fmt.Errorf("no ailang.toml found: %w\nRun 'ailang init package' first", err)
	}

	tree, err := pkg.BuildDependencyTree(manifest, cwd)
	if err != nil {
		return fmt.Errorf("failed to build dependency tree: %w", err)
	}

	fmt.Print(tree)
	return nil
}
