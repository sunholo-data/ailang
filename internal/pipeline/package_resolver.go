package pipeline

import (
	"github.com/sunholo/ailang/internal/loader"
	"github.com/sunholo/ailang/internal/pkg"
)

// tryLoadPackageResolver attempts to set up a package resolver from
// ailang.toml + ailang.lock in the given directory. Returns nil if
// no package manifest exists (backward compatible — bare projects work unchanged).
func tryLoadPackageResolver(dir string) loader.PackageResolver {
	// Check if ailang.toml exists
	manifestDir := pkg.FindManifest(dir)
	if manifestDir == "" {
		return nil // No package manifest — use legacy module resolution
	}

	// Load lock file
	lf, err := pkg.LoadLockFile(manifestDir)
	if err != nil {
		return nil // No lock file or invalid — skip package resolution
	}

	return pkg.NewPackageLoader(lf, manifestDir)
}
