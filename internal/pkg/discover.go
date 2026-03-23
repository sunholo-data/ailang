package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PackageSources holds the discovered source files and metadata for a package.
type PackageSources struct {
	// Dir is the absolute path to the package directory.
	Dir string

	// Manifest is the parsed ailang.toml.
	Manifest *PackageManifest

	// SourceFiles maps module paths to their .ail file paths (absolute).
	// e.g., "sunholo/billing_store/core" -> "/path/to/src/core.ail"
	SourceFiles map[string]string

	// TestFiles lists discovered *_test.ail files (absolute paths).
	TestFiles []string

	// OrphanFiles lists .ail files not matching any exported module.
	OrphanFiles []string

	// MissingModules lists modules declared in [exports] but without source files.
	MissingModules []string
}

// AllSourcePaths returns all source file paths (not test files) in deterministic order.
func (ps *PackageSources) AllSourcePaths() []string {
	paths := make([]string, 0, len(ps.SourceFiles))
	for _, p := range ps.SourceFiles {
		paths = append(paths, p)
	}
	return paths
}

// DiscoverPackageSources reads ailang.toml from dir and discovers all source and test files.
// It maps exported module paths to source files, identifies test files, and reports
// missing modules and orphan files.
func DiscoverPackageSources(dir string) (*PackageSources, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot resolve directory: %w", err)
	}

	// Load manifest
	manifest, err := LoadManifest(absDir)
	if err != nil {
		return nil, fmt.Errorf("cannot load package manifest: %w", err)
	}

	result := &PackageSources{
		Dir:         absDir,
		Manifest:    manifest,
		SourceFiles: make(map[string]string),
	}

	// Discover all .ail files in the package directory
	var allFiles []string
	err = filepath.Walk(absDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// Skip hidden directories and common non-source dirs
		if info.IsDir() {
			base := filepath.Base(path)
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".ail") {
			allFiles = append(allFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot scan package directory: %w", err)
	}

	// Separate test files from source files
	var sourceFiles []string
	for _, f := range allFiles {
		base := filepath.Base(f)
		if strings.HasSuffix(base, "_test.ail") {
			result.TestFiles = append(result.TestFiles, f)
		} else {
			sourceFiles = append(sourceFiles, f)
		}
	}

	// Build expected module-to-file mapping from exports
	// Module path "vendor/name/module" -> expected file "module.ail" or "src/module.ail"
	pkgName := manifest.Package.Name
	expectedModules := make(map[string]bool)
	for _, mod := range manifest.Exports.Modules {
		expectedModules[mod] = true
	}

	// Map source files to module paths
	matchedFiles := make(map[string]bool) // track which files matched a module
	for _, mod := range manifest.Exports.Modules {
		filePath := resolveModuleToFile(absDir, pkgName, mod)
		if filePath != "" {
			result.SourceFiles[mod] = filePath
			matchedFiles[filePath] = true
		} else {
			result.MissingModules = append(result.MissingModules, mod)
		}
	}

	// Identify orphan files (source files not matching any exported module)
	for _, f := range sourceFiles {
		if !matchedFiles[f] {
			result.OrphanFiles = append(result.OrphanFiles, f)
		}
	}

	return result, nil
}

// resolveModuleToFile finds the .ail file for a given module path within a package.
// Returns empty string if not found.
func resolveModuleToFile(pkgDir, pkgName, modulePath string) string {
	// Strip package name prefix to get the relative module name
	// "sunholo/billing_store/core" -> "core"
	// "sunholo/billing_store" -> "" (root module -> core.ail)
	var relModule string
	if modulePath == pkgName {
		relModule = "core"
	} else if strings.HasPrefix(modulePath, pkgName+"/") {
		relModule = strings.TrimPrefix(modulePath, pkgName+"/")
	} else {
		// Module path doesn't match package name — try using it directly
		// This handles cases where module_prefix remaps paths
		relModule = modulePath
	}

	// Convert module path to file path: "store/core" -> "store/core.ail"
	fileName := relModule + ".ail"

	// Search candidates: src/ first, then root
	candidates := []string{
		filepath.Join(pkgDir, "src", fileName),
		filepath.Join(pkgDir, fileName),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return ""
}
