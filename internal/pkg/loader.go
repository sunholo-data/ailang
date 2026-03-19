package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PackageLoader resolves package imports against a lock file.
type PackageLoader struct {
	lockFile  *LockFile
	manifests map[string]*PackageManifest // cached manifests by package name
	rootDir   string
}

// NewPackageLoader creates a loader from a lock file and root directory.
func NewPackageLoader(lf *LockFile, rootDir string) *PackageLoader {
	return &PackageLoader{
		lockFile:  lf,
		manifests: make(map[string]*PackageManifest),
		rootDir:   rootDir,
	}
}

// ResolveImport resolves a package import path to a source file path.
// importPath is the path without the pkg/ prefix: "vendor/name/module"
// Returns the absolute path to the .ail source file.
func (pl *PackageLoader) ResolveImport(importPath string) (string, error) {
	// Extract package name (first two segments)
	parts := strings.SplitN(importPath, "/", 3)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid package import path %q: must be vendor/name or vendor/name/module", importPath)
	}
	pkgName := parts[0] + "/" + parts[1]

	// Look up in lock file
	locked, found := pl.lockFile.FindPackage(pkgName)
	if !found {
		return "", fmt.Errorf("package %q not found in ailang.lock; run 'ailang lock' to resolve dependencies", pkgName)
	}

	// Get the package directory
	pkgDir, err := pl.packageDir(locked)
	if err != nil {
		return "", err
	}

	// Check export visibility
	if err := pl.checkExportVisibility(pkgName, importPath, pkgDir); err != nil {
		return "", err
	}

	// Map import path to file path
	// vendor/name/module → src/module.ail (relative to package dir)
	// vendor/name → src/core.ail (default module)
	var modulePath string
	if len(parts) == 2 {
		// Import the package root module
		modulePath = "core.ail"
	} else {
		// Import a specific module within the package
		modulePath = parts[2] + ".ail"
	}

	// Try src/ subdirectory first, then root
	candidates := []string{
		filepath.Join(pkgDir, "src", modulePath),
		filepath.Join(pkgDir, modulePath),
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("module %q not found in package %q (tried %s)", importPath, pkgName, strings.Join(candidates, ", "))
}

// packageDir returns the directory for a locked package.
func (pl *PackageLoader) packageDir(locked *LockedPackage) (string, error) {
	switch locked.Source {
	case "path":
		dir := locked.Path
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(pl.rootDir, dir)
		}
		if _, err := os.Stat(dir); err != nil {
			return "", fmt.Errorf("package directory not found: %s", dir)
		}
		return dir, nil
	case "registry":
		return "", fmt.Errorf("registry packages not yet supported (package %s); use path dependencies", locked.Name)
	default:
		return "", fmt.Errorf("unknown package source: %s", locked.Source)
	}
}

// checkExportVisibility verifies that the imported module is exported by the package.
func (pl *PackageLoader) checkExportVisibility(pkgName, importPath, pkgDir string) error {
	manifest, err := pl.loadManifest(pkgName, pkgDir)
	if err != nil {
		return err
	}

	// If no exports are declared, all modules are accessible (permissive default)
	if len(manifest.Exports.Modules) == 0 {
		return nil
	}

	// Check if the import path matches an exported module
	for _, exported := range manifest.Exports.Modules {
		if importPath == exported {
			return nil
		}
	}

	// Build helpful error message with available exports
	return fmt.Errorf("module %q is not exported by package %q\nAvailable exports:\n  %s",
		importPath, pkgName, strings.Join(manifest.Exports.Modules, "\n  "))
}

// loadManifest loads and caches a package's manifest.
func (pl *PackageLoader) loadManifest(pkgName, pkgDir string) (*PackageManifest, error) {
	if m, ok := pl.manifests[pkgName]; ok {
		return m, nil
	}

	m, err := LoadManifest(pkgDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load manifest for %s: %w", pkgName, err)
	}

	pl.manifests[pkgName] = m
	return m, nil
}

// HasPackage returns true if the named package exists in the lock file.
func (pl *PackageLoader) HasPackage(pkgName string) bool {
	_, found := pl.lockFile.FindPackage(pkgName)
	return found
}
