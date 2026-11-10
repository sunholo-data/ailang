package loader

import (
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// One module file, one identity (M1b of m-v1-nominal-type-identity).
//
// A package's module can be named several ways: by its published path
// (`pkg/<vendor>/<name>/<module>`), by the package's `module_prefix`
// (`<prefix>/<module>`, what the package's own files write), and — inside the
// root package — by the bare self-import (`<vendor>/<name>/<module>`). The
// loader used to key its cache by the spelling, so one file imported two ways
// was parsed, type-checked and evaluated twice under two module IDs: two
// copies of every type it declares, and its module-level code run twice.
//
// canonicalImportPath maps every spelling the package resolver answers to ONE
// identity per build:
//
//   - a DEPENDENCY's module is `pkg/<vendor>/<name>/<module>`, the path its
//     consumers write. It is unique in a build (package names are unique in
//     the lock; a module_prefix may be shared by several dependencies), and it
//     is the identity `./x` imports inside a dependency already had.
//   - the ROOT package's own module keeps the spelling the package itself
//     writes: `<prefix>/<module>` when it declares a module_prefix, otherwise
//     `<vendor>/<name>/<module>`. That is the ID those modules always had when
//     reached by their usual spelling (and the path-derived ID of the same file
//     checked as an entry), so diagnostics, serve-api dispatch and traces of a
//     package checked in its own repository do not change; only the other
//     spellings (`pkg/<root>/…`, the bare form of a prefixed root) fold into it.
//     MOD013 already forbids a dependency to share the root's prefix.
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
	canon := canonPath
	if pkgName, rest, ok := ml.resolvePackageSpelling(canonPath); ok {
		canon = ml.packageModuleID(pkgName, rest)
	}
	if ml.canonicalOf == nil {
		ml.canonicalOf = make(map[string]string)
	}
	ml.canonicalOf[canonPath] = canon
	return canon
}

// resolvePackageSpelling reports which package and module a spelling names,
// when the package resolver answers it. rest is "" for the package's root
// module or "/<module path>". It tries the spellings in the order Load's own
// branches try them, so the file it names is the file Load would read for the
// original spelling.
func (ml *ModuleLoader) resolvePackageSpelling(canonPath string) (pkgName, rest string, ok bool) {
	switch {
	case strings.HasPrefix(canonPath, "../"), strings.HasPrefix(canonPath, "std/"),
		strings.HasSuffix(canonPath, ".ail"):
		return "", "", false
	case strings.HasPrefix(canonPath, "pkg/"):
		// The published path. ResolveImport takes it without "pkg/".
		bare := strings.TrimPrefix(canonPath, "pkg/")
		parts := strings.SplitN(bare, "/", 3)
		if len(parts) < 2 {
			return "", "", false
		}
		if _, err := ml.pkgLoader.ResolveImport(bare); err != nil {
			return "", "", false
		}
		return parts[0] + "/" + parts[1], strings.TrimPrefix(bare, parts[0]+"/"+parts[1]), true
	}
	// Bare self-import: `import sunholo/linkedin/types` inside sunholo/linkedin.
	// Load does not fall through when this fails, so neither do we.
	if ml.currentPackageName != "" && pathMatchesPackagePrefix(canonPath, ml.currentPackageName) {
		if _, err := ml.pkgLoader.ResolveImport(canonPath); err != nil {
			return "", "", false
		}
		return ml.currentPackageName, strings.TrimPrefix(canonPath, ml.currentPackageName), true
	}
	// module_prefix spelling: `import docparse/types/document` where
	// sunholo/ailang_parse declares module_prefix = "docparse".
	firstSeg, rest := canonPath, ""
	if idx := strings.Index(canonPath, "/"); idx >= 0 {
		firstSeg, rest = canonPath[:idx], canonPath[idx:]
	}
	for _, name := range ml.modulePrefixMap[firstSeg] {
		if _, err := ml.pkgLoader.ResolveImport(name + rest); err == nil {
			return name, rest, true
		}
	}
	return "", "", false
}

// packageModuleID is the canonical module ID of module rest of package
// pkgName in this build (see canonicalImportPath).
func (ml *ModuleLoader) packageModuleID(pkgName, rest string) string {
	if pkgName != ml.currentPackageName || ml.currentPackageName == "" {
		return "pkg/" + pkgName + rest
	}
	if prefix := ml.prefixOf(pkgName); prefix != "" {
		return prefix + rest
	}
	return pkgName + rest
}

// prefixOf returns the module_prefix registered for pkgName, or "".
func (ml *ModuleLoader) prefixOf(pkgName string) string {
	for prefix, names := range ml.modulePrefixMap {
		for _, name := range names {
			if name == pkgName {
				return prefix
			}
		}
	}
	return ""
}

// canonicalizeImportDecl rewrites one import to its canonical identity,
// keeping the author's spelling in WrittenPath for diagnostics. A relative
// import arrives here already turned into `pkg/<declared dir>/<name>` by
// extractImports; its written form is `./<name>`.
func (ml *ModuleLoader) canonicalizeImportDecl(imp *ast.ImportDecl) {
	if imp == nil {
		return
	}
	canon := ml.canonicalImportPath(imp.Path)
	if canon == imp.Path {
		return
	}
	written := imp.Path
	if imp.IsRelative {
		written = "./" + imp.RelativePath
	}
	if imp.WrittenPath == "" {
		imp.WrittenPath = written
	}
	imp.Path = canon
	imp.IsPackage = strings.HasPrefix(canon, "pkg/")
	imp.PackageName = ""
	if imp.IsPackage {
		parts := strings.SplitN(strings.TrimPrefix(canon, "pkg/"), "/", 3)
		if len(parts) >= 2 {
			imp.PackageName = parts[0] + "/" + parts[1]
		}
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
