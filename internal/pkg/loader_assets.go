package pkg

import "sort"

// AssetTarget returns the locked directory and version, and whether the package
// is in the lock. Missing installs retain their version and return an empty dir;
// callers must preserve that entry rather than choosing another cached version.
func (pl *PackageLoader) AssetTarget(name string) (dir, version string, locked bool) {
	entry, found := pl.lockFile.FindPackage(name)
	if !found {
		manifest, err := LoadManifest(pl.rootDir)
		if err == nil && manifest.Package.Name == name {
			return pl.rootDir, manifest.Package.Version, true
		}
		return "", "", false
	}
	dir, err := pl.packageDir(entry)
	if err != nil {
		dir = ""
	}
	return dir, entry.Version, true
}

// LockedPackageNames returns every resolved package name, including transitive
// dependencies and the root package itself (also before a lockfile exists).
func (pl *PackageLoader) LockedPackageNames() []string {
	names := make([]string, 0, len(pl.lockFile.Packages))
	for _, entry := range pl.lockFile.Packages {
		names = append(names, entry.Name)
	}
	if manifest, err := LoadManifest(pl.rootDir); err == nil && manifest.Package.Name != "" {
		if _, locked := pl.lockFile.FindPackage(manifest.Package.Name); !locked {
			names = append(names, manifest.Package.Name)
		}
	}
	sort.Strings(names)
	return names
}
