package runner

import (
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// lockedPackageDirs returns the package directories pinned by the ailang.lock
// that governs filename (or the explicit packageDir), plus the root package
// itself, so std/package.assetPath reads the same version the program was
// compiled against (#1772). nil when there is no manifest; a manifest without
// a lock yields just the root package.
func lockedPackageDirs(filename, packageDir string) map[string]string {
	searchDir := packageDir
	if searchDir == "" {
		searchDir = filepath.Dir(filename)
	}
	manifestDir := pkg.FindManifest(searchDir)
	if manifestDir == "" {
		return nil
	}
	dirs := map[string]string{}
	if lf, err := pkg.LoadLockFile(manifestDir); err == nil {
		dirs = pkg.NewPackageLoader(lf, manifestDir).LockedPackageDirs()
	}
	if manifest, err := pkg.LoadManifest(manifestDir); err == nil && manifest.Package.Name != "" {
		dirs[manifest.Package.Name] = manifestDir
	}
	return dirs
}
