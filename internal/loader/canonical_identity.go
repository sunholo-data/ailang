package loader

import (
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// One module file, one identity (M1b of m-v1-nominal-type-identity).
//
// A package's module can be named three ways: by its published path
// (`pkg/<vendor>/<name>/<module>`), by the package's `module_prefix`
// (`<prefix>/<module>`, what the package's own files write), and — inside the
// root package — by the bare self-import (`<vendor>/<name>/<module>`). The
// loader used to key its cache by the spelling, so one file imported two ways
// was parsed, type-checked and evaluated twice under two module IDs: two
// copies of every type it declares, and its module-level code run twice.
//
// canonicalImportPath maps every spelling that the package resolver answers to
// ONE identity, the published `pkg/<vendor>/<name>/<module>` path:
//
//   - it is unique in a build: package names are unique in the lock file, while
//     a module_prefix may be shared by several packages;
//   - it is the same whether the package is the root being checked in its own
//     repository or a dependency installed under ~/.ailang;
//   - it is already the identity the loader gives relative (`./x`) imports.
//
// The mapping is logical (package name + module path), never the file's
// realpath, so two vendored copies of different packages stay distinct even
// when a symlink makes them share bytes. A spelling the resolver rejects is
// returned unchanged, so Load reports exactly the error it reported before.
func (ml *ModuleLoader) canonicalImportPath(path string) string {
	canonPath := canonicalizeModulePath(path)
	if ml.pkgLoader == nil {
		return canonPath
	}
	if canon, ok := ml.canonicalOf[canonPath]; ok {
		return canon
	}
	canon := ml.resolveCanonicalSpelling(canonPath)
	if ml.canonicalOf == nil {
		ml.canonicalOf = make(map[string]string)
	}
	ml.canonicalOf[canonPath] = canon
	return canon
}

// resolveCanonicalSpelling is canonicalImportPath without the memo. It tries
// the spellings in the order Load's own branches try them, so the file it
// names is the file Load would have read for the original spelling.
func (ml *ModuleLoader) resolveCanonicalSpelling(canonPath string) string {
	switch {
	case strings.HasPrefix(canonPath, "./"), strings.HasPrefix(canonPath, "../"),
		strings.HasPrefix(canonPath, "pkg/"), strings.HasPrefix(canonPath, "std/"),
		strings.HasSuffix(canonPath, ".ail"):
		return canonPath
	}
	// Bare self-import: `import sunholo/linkedin/types` inside sunholo/linkedin.
	if ml.currentPackageName != "" && pathMatchesPackagePrefix(canonPath, ml.currentPackageName) {
		if _, err := ml.pkgLoader.ResolveImport(canonPath); err == nil {
			return "pkg/" + canonPath
		}
		return canonPath
	}
	// module_prefix spelling: `import docparse/types/document` where
	// sunholo/ailang_parse declares module_prefix = "docparse".
	if ml.modulePrefixMap == nil {
		return canonPath
	}
	firstSeg, rest := canonPath, ""
	if idx := strings.Index(canonPath, "/"); idx >= 0 {
		firstSeg, rest = canonPath[:idx], canonPath[idx:]
	}
	for _, pkgName := range ml.modulePrefixMap[firstSeg] {
		if _, err := ml.pkgLoader.ResolveImport(pkgName + rest); err == nil {
			return "pkg/" + pkgName + rest
		}
	}
	return canonPath
}

// canonicalizeImportDecl rewrites one import to its canonical identity,
// keeping the author's spelling in WrittenPath for diagnostics. Relative
// imports are normalised by extractImports before this runs.
func (ml *ModuleLoader) canonicalizeImportDecl(imp *ast.ImportDecl) {
	if imp == nil || imp.IsRelative {
		return
	}
	canon := ml.canonicalImportPath(imp.Path)
	if canon == imp.Path || !strings.HasPrefix(canon, "pkg/") {
		return
	}
	imp.WrittenPath = imp.Path
	imp.Path = canon
	imp.IsPackage = true
	parts := strings.SplitN(strings.TrimPrefix(canon, "pkg/"), "/", 3)
	if len(parts) >= 2 {
		imp.PackageName = parts[0] + "/" + parts[1]
	}
}

// ImportDisplay names an import in a diagnostic: the canonical module, plus
// the spelling the author wrote when the loader rewrote it.
func ImportDisplay(imp *ast.ImportDecl) string {
	if imp == nil {
		return ""
	}
	return DisplayModule(imp.Path, imp.WrittenPath)
}

// DisplayModule renders a canonical module ID with the spelling that reached
// it, when the two differ: `pkg/acme/shapes/types/shape (imported as
// shp/types/shape)`.
func DisplayModule(canonical, written string) string {
	if written == "" || written == canonical {
		return canonical
	}
	return canonical + " (imported as " + written + ")"
}
