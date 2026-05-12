package loader

import "strings"

// SetCurrentPackageName records the <vendor>/<name> of the package being
// compiled. When set, bare canonical imports whose first two segments match
// this name (e.g. `import sunholo/linkedin/types` from within sunholo/linkedin)
// route through the package resolver's self-reference path instead of falling
// through to project-relative resolution. Pass "" to clear.
func (ml *ModuleLoader) SetCurrentPackageName(name string) {
	ml.currentPackageName = name
}

// pathMatchesPackagePrefix reports whether canonPath is a self-reference to
// pkgName. pkgName is "<vendor>/<name>"; a match requires canonPath to equal
// pkgName or extend it with `/<submodule>...`. Bare "sunholo/linkedin" matches
// pkgName "sunholo/linkedin"; "sunholo/linkedin/types" matches too;
// "sunholo/linkedin_other" does NOT (boundary check).
func pathMatchesPackagePrefix(canonPath, pkgName string) bool {
	if canonPath == pkgName {
		return true
	}
	return strings.HasPrefix(canonPath, pkgName+"/")
}
