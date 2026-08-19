package pipeline

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// currentPackageManifest holds the manifest for the current package being compiled.
// Set by tryLoadPackageResolver, read by validateEffectCeiling.
var currentPackageManifest *pkg.PackageManifest

// currentModulePrefixMap holds module_prefix mappings for all loaded packages.
// Set by tryLoadPackageResolver, read by MOD010 validation and MOD013 detection.
// Key: package name (e.g., "sunholo/docparse"), Value: module_prefix (e.g., "docparse").
var currentModulePrefixMap map[string]string

// currentRootPkgName is the package name of the root project (from ailang.toml).
// Set by tryLoadPackageResolver, read by MOD013 detection to identify which entry
// in currentModulePrefixMap belongs to the root vs a dependency.
var currentRootPkgName string

// tryLoadPackageResolver attempts to set up a package resolver from
// ailang.toml + ailang.lock in the given directory. Returns (nil, nil) if no
// package manifest exists (backward compatible — bare projects work unchanged).
func tryLoadPackageResolver(dir string) (loader.PackageResolver, error) {
	// Check if ailang.toml exists
	manifestDir := pkg.FindManifest(dir)
	if manifestDir == "" {
		currentPackageManifest = nil
		currentModulePrefixMap = nil
		currentRootPkgName = ""
		return nil, nil // No package manifest — use legacy module resolution
	}

	// Load manifest for effect ceiling checks
	manifest, err := pkg.LoadManifest(manifestDir)
	if err != nil {
		currentPackageManifest = nil
		currentModulePrefixMap = nil
		currentRootPkgName = ""
		return nil, fmt.Errorf("cannot load package manifest %s: %w", filepath.Join(manifestDir, pkg.ManifestFile), err)
	}
	currentPackageManifest = manifest
	currentRootPkgName = manifest.Package.Name

	// Build module_prefix map from current package and dependencies
	currentModulePrefixMap = make(map[string]string)
	if manifest.Package.ModulePrefix != "" {
		currentModulePrefixMap[manifest.Package.Name] = manifest.Package.ModulePrefix
	}

	// Load lock file
	lf, err := pkg.LoadLockFile(manifestDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil // Lock files are optional while authoring a package.
		}
		return nil, fmt.Errorf("cannot load package lock file %s: %w", filepath.Join(manifestDir, pkg.LockFileName), err)
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

	return pkgLoader, nil
}

// loadDepManifest loads a dependency's manifest via the package loader.
func loadDepManifest(pkgLoader *pkg.PackageLoader, depName string) (*pkg.PackageManifest, error) {
	// Use the loader's internal manifest cache
	return pkgLoader.LoadManifestByName(depName)
}

// tryLoadSelfOnlyPackageResolver attempts to set up a self-only package
// resolver from ailang.toml alone (no ailang.lock). Returns nil if no
// manifest exists. This is the authoring fallback: a freshly-initialized
// package can resolve sibling imports before `ailang lock` has been run.
// External pkg/<other>/... imports under this resolver fail with a clear
// "run ailang lock" error rather than LDR001.
func tryLoadSelfOnlyPackageResolver(dir string) loader.PackageResolver {
	manifestDir := pkg.FindManifest(dir)
	if manifestDir == "" {
		return nil
	}

	manifest, err := pkg.LoadManifest(manifestDir)
	if err != nil {
		return nil
	}
	currentPackageManifest = manifest
	currentRootPkgName = manifest.Package.Name

	currentModulePrefixMap = make(map[string]string)
	if manifest.Package.ModulePrefix != "" {
		currentModulePrefixMap[manifest.Package.Name] = manifest.Package.ModulePrefix
	}

	pkgLoader, err := pkg.NewSelfOnlyPackageLoader(manifestDir)
	if err != nil {
		return nil
	}
	return pkgLoader
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

// packageSearchDir decides where to look for ailang.toml/ailang.lock.
//
// The package manifest belongs to the SOURCE FILE, not to the process CWD.
// Anchoring the search at "." meant an installed AILANG CLI invoked from
// anywhere but its own project root could not resolve its own package imports
// — and the failure blamed a missing ailang.toml/ailang.lock that were sitting
// next to the source file (ailang#671). The named-test harness already passed
// PackageDir for exactly this reason; every other entry point (run, check,
// serve-api, the MCP server, embed) did not.
//
// FindManifest walks upward, so anchoring at the file's directory is a superset
// of the old behaviour for any file inside its own project. An explicit
// PackageDir always wins.
func packageSearchDir(explicitPackageDir, loaderBaseDir, entryFilename string) string {
	if explicitPackageDir != "" {
		return explicitPackageDir
	}
	if dir := entrySourceDir(entryFilename); dir != "" {
		return dir
	}
	return loaderBaseDir
}

// entrySourceDir returns the directory holding the entry source file, or ""
// when there is no real file on disk (code compiled from a string, a REPL
// buffer, or a synthetic "<stdin>"-style name). Returning "" leaves the
// caller on its previous CWD-anchored behaviour rather than inventing a
// directory from a name that never named a file.
func entrySourceDir(filename string) string {
	if filename == "" {
		return ""
	}
	st, err := os.Stat(filename)
	if err != nil || st.IsDir() {
		return ""
	}
	return filepath.Dir(filename)
}

// packageResolverAbsentReason explains why no package resolver could be wired
// for dir. The two causes need different user actions, and conflating them is
// what made ailang#671 tell users to create files that already existed.
func packageResolverAbsentReason(dir string) string {
	if dir == "" {
		dir = "."
	}
	searched := dir
	if abs, err := filepath.Abs(dir); err == nil {
		searched = abs
	}
	if manifestDir := pkg.FindManifest(dir); manifestDir != "" {
		return fmt.Sprintf("the package manifest %s exists but could not be loaded as a package",
			filepath.Join(manifestDir, pkg.ManifestFile))
	}
	return fmt.Sprintf("no %s was found in %s or any parent directory; run 'ailang init package' there, or invoke ailang from inside the package",
		pkg.ManifestFile, searched)
}
