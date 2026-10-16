package pkg

import "sort"

// AssetTarget returns the locked directory and version, and whether the package
// is in the lock. Missing installs retain their version and return an empty dir;
// callers must preserve that entry rather than choosing another cached version.
func (pl *PackageLoader) AssetTarget(name string) (dir, version string, locked bool) {
	entry, found := pl.lockFile.FindPackage(name)
	if !found {
		return "", "", false
	}
	dir, err := pl.packageDir(entry)
	if err != nil {
		dir = ""
	}
	return dir, entry.Version, true
}

// LockedPackageNames returns all dependency names, including transitive entries.
func (pl *PackageLoader) LockedPackageNames() []string {
	names := make([]string, 0, len(pl.lockFile.Packages))
	for _, entry := range pl.lockFile.Packages {
		names = append(names, entry.Name)
	}
	sort.Strings(names)
	return names
}
