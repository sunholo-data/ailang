package loader

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// One module file, one identity (M1b of m-v1-nominal-type-identity).
//
// A package's module can be named several ways: by its published path
// (`pkg/<vendor>/<name>/<module>`), by the package's `module_prefix`
// (`<prefix>/<module>`, what the package's own files write), inside the root
// package by the bare self-import (`<vendor>/<name>/<module>`), and — in a
// package with no [exports] list — by any other path the resolver maps to the
// same file (`pkg/<vendor>/<name>/<prefix>/<module>`). The loader used to key
// its cache by the spelling, so one file imported two ways was parsed,
// type-checked and evaluated twice under two module IDs.
//
// The identity is now derived from what a spelling RESOLVES to — the package
// and the file — never from the spelling:
//
//  1. Scope. A spelling is read in the scope of the package whose module wrote
//     it. A module_prefix spelling names the importing package's own module
//     when the prefix is its own, else a module of one of its declared
//     dependencies with that prefix; only when neither claims the prefix does
//     the build-wide prefix table (root + direct dependencies) apply. So a
//     dependency's `px/types/t` is its own file even when another package
//     shares "px", and a transitive dependency resolves its own prefix.
//  2. Module path. From the resolved file, the module's path inside its
//     package is the shortest of (prefix stripped, src/ stripped, checkout
//     <vendor>/<name>/ stripped, the path as is) that the resolver maps back to
//     the SAME file. Every spelling of one file therefore yields one path.
//  3. Name. A DEPENDENCY's module is `pkg/<vendor>/<name>/<path>`, the path its
//     consumers write; unique in a build because package names are unique in
//     the lock. The ROOT package's own module keeps the spelling the package
//     writes — `<prefix>/<path>`, else `<vendor>/<name>/<path>` — the ID it had
//     before, so a package checked in its own repository reports its modules
//     by the names it declares (MOD013 forbids a dependency to share the
//     root's prefix, so this is unique too).
//
// A spelling the resolver rejects is returned unchanged, so Load reports the
// error it reported before — except a dependency's own prefix spelling, which
// is pinned to that dependency (`pkg/<owner>/<path>`) so it fails naming the
// dependency instead of silently loading another package's module.
func (ml *ModuleLoader) canonicalImportPath(path string) string {
	return ml.canonicalImportPathFor(ml.currentPackageName, path)
}

// resetCanonical drops the spelling memo; called by every setter that changes
// an input of the mapping.
func (ml *ModuleLoader) resetCanonical() {
	ml.canonicalOf = nil
	ml.resolvedByID = nil
}

// resolvedModule is a package module a spelling resolved to.
type resolvedModule struct {
	pkg  string // <vendor>/<name>
	file string // the file ResolveImport returned
}

// PackageLayoutResolver is implemented by package resolvers that can describe
// a package visible to the build: its directory, module_prefix and declared
// dependency names (pkg.PackageLoader). Without it the loader falls back to
// the build-wide prefix table and to spelling-derived module paths.
type PackageLayoutResolver interface {
	PackageLayout(pkgName string) (dir, prefix string, deps []string, ok bool)
}

// canonicalImportPathFor is canonicalImportPath for a spelling written by a
// module of package owner ("" = no package context).
func (ml *ModuleLoader) canonicalImportPathFor(owner, path string) string {
	canonPath := canonicalizeModulePath(path)
	if ml.pkgLoader == nil {
		return canonPath
	}
	key := owner + "\x00" + canonPath
	if canon, ok := ml.canonicalOf[key]; ok {
		return canon
	}
	canon := canonPath
	if pkgName, rest, file, ok := ml.resolvePackageSpelling(owner, canonPath); ok {
		canon = ml.packageModuleID(pkgName, ml.modulePathOf(pkgName, file, rest))
		if ml.resolvedByID == nil {
			ml.resolvedByID = make(map[string]resolvedModule)
		}
		ml.resolvedByID[CanonicalModuleID(canon)] = resolvedModule{pkg: pkgName, file: file}
	} else if pinned, ok := ml.pinOwnPrefix(owner, canonPath); ok {
		canon = pinned
	}
	if ml.canonicalOf == nil {
		ml.canonicalOf = make(map[string]string)
	}
	ml.canonicalOf[key] = canon
	return canon
}

// resolvePackageSpelling reports which package module a spelling names, when
// the package resolver answers it: the package, the module path the spelling
// wrote ("" for the package's root module, else "/<path>"), and the file.
func (ml *ModuleLoader) resolvePackageSpelling(owner, canonPath string) (pkgName, rest, file string, ok bool) {
	switch {
	case strings.HasPrefix(canonPath, "../"), strings.HasPrefix(canonPath, "std/"),
		strings.HasSuffix(canonPath, ".ail"):
		return "", "", "", false
	case strings.HasPrefix(canonPath, "pkg/"):
		// The published path. ResolveImport takes it without "pkg/".
		bare := strings.TrimPrefix(canonPath, "pkg/")
		parts := strings.SplitN(bare, "/", 3)
		if len(parts) < 2 {
			return "", "", "", false
		}
		f, err := ml.pkgLoader.ResolveImport(bare)
		if err != nil {
			return "", "", "", false
		}
		name := parts[0] + "/" + parts[1]
		return name, strings.TrimPrefix(bare, name), f, true
	}
	// Bare self-import: `import sunholo/linkedin/types` inside sunholo/linkedin.
	// Load does not fall through when this fails, so neither do we.
	if ml.currentPackageName != "" && pathMatchesPackagePrefix(canonPath, ml.currentPackageName) {
		f, err := ml.pkgLoader.ResolveImport(canonPath)
		if err != nil {
			return "", "", "", false
		}
		return ml.currentPackageName, strings.TrimPrefix(canonPath, ml.currentPackageName), f, true
	}
	// module_prefix spelling, read in the importing package's scope.
	firstSeg, rest := splitFirstSegment(canonPath)
	for _, name := range ml.prefixCandidates(owner, firstSeg) {
		if f, err := ml.pkgLoader.ResolveImport(name + rest); err == nil {
			return name, rest, f, true
		}
	}
	return "", "", "", false
}

func splitFirstSegment(p string) (first, rest string) {
	if idx := strings.Index(p, "/"); idx >= 0 {
		return p[:idx], p[idx:]
	}
	return p, ""
}

// prefixCandidates lists the packages a module_prefix spelling written by a
// module of owner may name, in the order they are tried: owner itself if the
// prefix is its own, then owner's declared dependencies with that prefix
// (sorted). Only when neither claims the prefix does the build-wide table
// (root + its direct dependencies, sorted per prefix) apply — the behaviour
// before scoping, kept for packages whose resolver cannot describe layouts.
func (ml *ModuleLoader) prefixCandidates(owner, prefix string) []string {
	var scoped []string
	if lr, ok := ml.pkgLoader.(PackageLayoutResolver); ok && owner != "" {
		if _, own, deps, ok := lr.PackageLayout(owner); ok {
			if own == prefix {
				scoped = append(scoped, owner)
			}
			sorted := append([]string(nil), deps...)
			sort.Strings(sorted)
			for _, d := range sorted {
				if _, p, _, ok := lr.PackageLayout(d); ok && p == prefix && d != owner {
					scoped = append(scoped, d)
				}
			}
		}
	}
	if len(scoped) > 0 {
		return scoped
	}
	return ml.modulePrefixMap[prefix]
}

// pinOwnPrefix maps a DEPENDENCY's own prefix spelling that did not resolve to
// `pkg/<owner><rest>`, so Load fails naming the dependency rather than falling
// back to the build-wide prefix table or the root's directory. The root keeps
// its old fallback (project-relative), which its unexported modules use.
func (ml *ModuleLoader) pinOwnPrefix(owner, canonPath string) (string, bool) {
	if owner == "" || owner == ml.currentPackageName || strings.HasPrefix(canonPath, "pkg/") {
		return "", false
	}
	lr, ok := ml.pkgLoader.(PackageLayoutResolver)
	if !ok {
		return "", false
	}
	_, own, _, ok := lr.PackageLayout(owner)
	firstSeg, rest := splitFirstSegment(canonPath)
	if !ok || own == "" || own != firstSeg {
		return "", false
	}
	return "pkg/" + owner + rest, true
}

// modulePathOf is the module path ("" or "/<path>") of file inside package
// pkgName, derived from the file so that every spelling of it agrees: the
// first candidate the resolver maps back to the same file. spellingRest (what
// the spelling wrote) is the fallback when the layout is unknown or no
// candidate round-trips.
func (ml *ModuleLoader) modulePathOf(pkgName, file, spellingRest string) string {
	lr, ok := ml.pkgLoader.(PackageLayoutResolver)
	if !ok {
		return spellingRest
	}
	dir, prefix, _, ok := lr.PackageLayout(pkgName)
	if !ok {
		return spellingRest
	}
	want := realFile(file)
	rel, err := filepath.Rel(realFile(dir), want)
	if err != nil || strings.HasPrefix(rel, "..") {
		return spellingRest
	}
	rel = strings.TrimSuffix(filepath.ToSlash(rel), ".ail")
	for _, cand := range modulePathCandidates(rel, prefix, pkgName) {
		f, err := ml.pkgLoader.ResolveImport(pkgName + cand)
		if err == nil && realFile(f) == want {
			return cand
		}
	}
	return spellingRest
}

// modulePathCandidates lists, most canonical first, the module paths a file
// at rel (relative to its package, no extension) may be published under.
func modulePathCandidates(rel, prefix, pkgName string) []string {
	base := strings.TrimPrefix(rel, "src/")
	var out []string
	add := func(p string) {
		for _, seen := range out {
			if seen == p {
				return
			}
		}
		out = append(out, p)
	}
	if base == "core" {
		add("") // the package's root module: pkg/<vendor>/<name>
	}
	if prefix != "" && strings.HasPrefix(base, prefix+"/") {
		add("/" + strings.TrimPrefix(base, prefix+"/"))
	}
	if strings.HasPrefix(base, pkgName+"/") {
		add("/" + strings.TrimPrefix(base, pkgName+"/"))
	}
	add("/" + base)
	add("/" + rel)
	return out
}

// realFile is p made absolute and symlink-resolved where possible.
func realFile(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	return filepath.Clean(p)
}

// packageModuleID is the canonical module ID of module rest of package
// pkgName in this build (see canonicalImportPath).
func (ml *ModuleLoader) packageModuleID(pkgName, rest string) string {
	if pkgName != ml.currentPackageName || ml.currentPackageName == "" {
		return "pkg/" + pkgName + rest
	}
	if prefix := ml.prefixOf(pkgName); prefix != "" {
		if rest == "" {
			return pkgName // the root module has no prefix form
		}
		return prefix + rest
	}
	return pkgName + rest
}

// prefixOf returns the module_prefix registered for pkgName, or "".
func (ml *ModuleLoader) prefixOf(pkgName string) string {
	if lr, ok := ml.pkgLoader.(PackageLayoutResolver); ok {
		if _, prefix, _, ok := lr.PackageLayout(pkgName); ok {
			return prefix
		}
	}
	for prefix, names := range ml.modulePrefixMap {
		for _, name := range names {
			if name == pkgName {
				return prefix
			}
		}
	}
	return ""
}

// canonicalizeImportDecl rewrites one import, written by a module of package
// owner, to its canonical identity, keeping the author's spelling in
// WrittenPath for diagnostics. A relative import arrives here already turned
// into `pkg/<declared dir>/<name>` by extractImports; its written form is
// `./<name>`.
func (ml *ModuleLoader) canonicalizeImportDecl(owner string, imp *ast.ImportDecl) {
	if imp == nil {
		return
	}
	canon := ml.canonicalImportPathFor(owner, imp.Path)
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

// recordIdentity notes that module id was read from file. A file read under
// a second id, or an id read from a second file, is an identity conflict:
// IdentityConflicts reports them (tests assert there are none).
func (ml *ModuleLoader) recordIdentity(id, file string) {
	if file == "" {
		return
	}
	rf := realFile(file)
	if ml.identityOfFile == nil {
		ml.identityOfFile = make(map[string]string)
		ml.fileOfIdentity = make(map[string]string)
	}
	if prev, ok := ml.identityOfFile[rf]; ok && prev != id {
		ml.identityConflicts = append(ml.identityConflicts, rf+": loaded as "+prev+" and as "+id)
	} else if !ok {
		ml.identityOfFile[rf] = id
	}
	if prev, ok := ml.fileOfIdentity[id]; ok && prev != rf {
		ml.identityConflicts = append(ml.identityConflicts, id+": read from "+prev+" and from "+rf)
	} else if !ok {
		ml.fileOfIdentity[id] = rf
	}
}

// IdentityConflicts reports every source file this loader read under more
// than one module ID, and every module ID it read from more than one file.
// Empty when module identity is single-valued and injective over the build.
func (ml *ModuleLoader) IdentityConflicts() []string {
	return append([]string(nil), ml.identityConflicts...)
}

// DisplayModule renders a canonical module ID with the spelling that reached
// it, when the two differ: `pkg/acme/shapes/types/shape (imported as
// shp/types/shape)`. Used by IMP010 (loader.getExport) and LDR002
// (link.CycleError).
func DisplayModule(canonical, written string) string {
	if written == "" || written == canonical {
		return canonical
	}
	return canonical + " (imported as " + written + ")"
}

// DeclaredModuleDisplay renders a DEPENDENCY module's ID with the path its
// file declares, when they differ: `pkg/acme/shapes/types/shape (module
// shp/types/shape)`. Used by diagnostics that name the module being compiled.
// Other IDs (the root's own modules, std, entry files) are returned as they
// are: they are already the name their author wrote or the file's path.
func DeclaredModuleDisplay(id string, file *ast.File) string {
	if !strings.HasPrefix(id, "pkg/") || file == nil || file.Module == nil || file.Module.Path == "" ||
		file.Module.Path == id || "pkg/"+file.Module.Path == id {
		return id
	}
	return id + " (module " + file.Module.Path + ")"
}
