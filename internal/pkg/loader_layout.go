package pkg

import "sort"

// PackageLayout describes a package visible to this loader — a locked
// dependency, or the root package itself — for the module loader's canonical
// module identity (internal/loader/canonical_identity.go): the directory its
// modules load from, its module_prefix, and the names of its declared
// dependencies (sorted). The lookup order is ResolveImport's: the lock first,
// then the root by self-reference. ok is false for a package the loader
// cannot see or whose manifest does not load.
func (pl *PackageLoader) PackageLayout(pkgName string) (dir, prefix string, deps []string, ok bool) {
	if locked, found := pl.lockFile.FindPackage(pkgName); found {
		d, err := pl.packageDir(locked)
		if err != nil {
			return "", "", nil, false
		}
		dir = d
	} else if pl.isSelfReference(pkgName) {
		dir = pl.rootDir
	} else {
		return "", "", nil, false
	}
	m, err := pl.loadManifest(pkgName, dir)
	if err != nil {
		return "", "", nil, false
	}
	for name := range m.Dependencies {
		deps = append(deps, name)
	}
	sort.Strings(deps)
	return dir, m.Package.ModulePrefix, deps, true
}
