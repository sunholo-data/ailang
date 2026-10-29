package pipeline

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// lockedAssetTargets shares construction between compile-time resolution and
// pre-evaluation callers. No package-global state is used for asset resolution.
func lockedAssetTargets(pl *pkg.PackageLoader) map[string]effects.PkgAssetTarget {
	targets := make(map[string]effects.PkgAssetTarget)
	for _, name := range pl.LockedPackageNames() {
		dir, version, _ := pl.AssetTarget(name)
		targets[name] = effects.PkgAssetTarget{Dir: dir, Version: version}
	}
	return targets
}

// AssetTargetsFor resolves the package context rooted at dir, without fetching
// dependencies. No manifest means no context; a manifest without a lock retains
// the root package for authoring. Malformed existing files fail explicitly.
func AssetTargetsFor(dir string) (map[string]effects.PkgAssetTarget, error) {
	root := pkg.FindManifest(dir)
	if root == "" {
		return nil, nil
	}
	if _, err := pkg.LoadManifest(root); err != nil {
		return nil, fmt.Errorf("cannot load package manifest %s: %w", filepath.Join(root, pkg.ManifestFile), err)
	}
	lf, err := pkg.LoadLockFile(root)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			self, selfErr := pkg.NewSelfOnlyPackageLoader(root)
			if selfErr != nil {
				return nil, selfErr
			}
			return lockedAssetTargets(self), nil
		}
		return nil, fmt.Errorf("cannot load package lock file %s: %w", filepath.Join(root, pkg.LockFileName), err)
	}
	return lockedAssetTargets(pkg.NewPackageLoader(lf, root)), nil
}
