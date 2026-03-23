package pipeline

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/loader"
	"github.com/sunholo/ailang/internal/pkg"
)

// currentPackageManifest holds the manifest for the current package being compiled.
// Set by tryLoadPackageResolver, read by validateEffectCeiling.
var currentPackageManifest *pkg.PackageManifest

// currentModulePrefixMap holds module_prefix mappings for all loaded packages.
// Set by tryLoadPackageResolver, read by MOD010 validation.
// Key: package name (e.g., "sunholo/docparse"), Value: module_prefix (e.g., "docparse").
var currentModulePrefixMap map[string]string

// tryLoadPackageResolver attempts to set up a package resolver from
// ailang.toml + ailang.lock in the given directory. Returns nil if
// no package manifest exists (backward compatible — bare projects work unchanged).
func tryLoadPackageResolver(dir string) loader.PackageResolver {
	// Check if ailang.toml exists
	manifestDir := pkg.FindManifest(dir)
	if manifestDir == "" {
		currentPackageManifest = nil
		return nil // No package manifest — use legacy module resolution
	}

	// Load manifest for effect ceiling checks
	manifest, err := pkg.LoadManifest(manifestDir)
	if err != nil {
		currentPackageManifest = nil
		return nil
	}
	currentPackageManifest = manifest

	// Build module_prefix map from current package and dependencies
	currentModulePrefixMap = make(map[string]string)
	if manifest.Package.ModulePrefix != "" {
		currentModulePrefixMap[manifest.Package.Name] = manifest.Package.ModulePrefix
	}

	// Load lock file
	lf, err := pkg.LoadLockFile(manifestDir)
	if err != nil {
		return nil // No lock file or invalid — skip package resolution
	}

	// Validate content hashes — detect stale lock files
	if err := lf.ValidateContentHashes(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	// Load module_prefix from dependency manifests
	pkgLoader := pkg.NewPackageLoader(lf, manifestDir)
	for depName := range manifest.Dependencies {
		if depManifest, err := loadDepManifest(pkgLoader, depName); err == nil {
			if depManifest.Package.ModulePrefix != "" {
				currentModulePrefixMap[depName] = depManifest.Package.ModulePrefix
			}
		}
	}

	return pkgLoader
}

// loadDepManifest loads a dependency's manifest via the package loader.
func loadDepManifest(pkgLoader *pkg.PackageLoader, depName string) (*pkg.PackageManifest, error) {
	// Use the loader's internal manifest cache
	return pkgLoader.LoadManifestByName(depName)
}

// validateEffectCeiling checks that a module's declared function effects
// do not exceed the current package's [effects].max ceiling.
// Only applies when an ailang.toml exists; bare projects are unchecked.
func validateEffectCeiling(surfaceAST *ast.File, modID string) error {
	if currentPackageManifest == nil {
		return nil // No package — no ceiling to enforce
	}
	if surfaceAST == nil {
		return nil
	}

	// Only check modules belonging to the current package (not dependencies)
	// Dependencies are imported via pkg/ prefix; the current package's own
	// modules use local paths.
	if strings.HasPrefix(modID, "pkg/") {
		return nil // This is a dependency module, not ours
	}

	maxEffects := currentPackageManifest.Effects.Max
	if maxEffects == nil {
		return nil // No ceiling declared
	}

	// Check each function's declared effects
	for _, fn := range surfaceAST.Funcs {
		declaredEffects := ast.EffectNames(fn.Effects)
		if err := pkg.CheckEffectCeiling(currentPackageManifest.Package.Name, declaredEffects, maxEffects); err != nil {
			return fmt.Errorf("in function %s in %s: %w", fn.Name, modID, err)
		}
	}

	return nil
}
