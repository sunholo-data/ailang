package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
)

// One module file, one identity (M1b of m-v1-nominal-type-identity).
//
// A package's module can be named several ways: by its published path
// (`pkg/<vendor>/<name>/<module>`), by the package's `module_prefix`
// (`<prefix>/<module>`), by the bare self-import (`<vendor>/<name>/<module>`),
// inside the root package by a project-relative path (`util` for util.ail),
// and — in a package with no [exports] list — by any other path the resolver
// maps to the same file. The loader used to key its cache by the spelling, so
// one file imported two ways was compiled and evaluated twice.
//
// The identity is derived from what a spelling RESOLVES to, never from the
// spelling (canonical_scope.go has the resolution rule):
//
//  1. Scope. A spelling is resolved in the scope of the package whose module
//     wrote it: that package itself, its DECLARED dependencies and std. A
//     spelling that resolves to any other package is an error (LDR006), a
//     prefix two declared dependencies both provide is an error (LDR005), and
//     a dependency's spelling that does not resolve in its scope fails there
//     (LDR001) — it is never retried in another package's scope.
//  2. Module path. From the resolved file, the module's path inside its
//     package is the first of (prefix stripped, src/ stripped, checkout
//     <vendor>/<name>/ stripped, the path as is) that the resolver maps back to
//     the SAME file, so every spelling of one file yields one path. A spelling
//     whose case differs from the file on disk is rejected on every host.
//  3. Name. A DEPENDENCY's module is `pkg/<vendor>/<name>/<path>`. The ROOT
//     package's own module keeps the spelling the package writes —
//     `<prefix>/<path>`, else `<vendor>/<name>/<path>` — the ID it had before.
//
// What is left — a file reached through a path the package resolver does not
// govern (a symlink into a package, a file-path import of another package's
// file) — is caught by the file↔identity record: the pipeline fails the build
// with LDR003 if any file was read under two identities (IdentityConflicts).
func (ml *ModuleLoader) canonicalImportPath(path string) (string, error) {
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

// canonicalResult is a memoised canonicalisation: the canonical spelling, or
// the error the spelling must fail with in its importer's scope.
type canonicalResult struct {
	canon string
	err   error
}

// PackageLayoutResolver is implemented by package resolvers that can describe
// a package visible to the build: its directory, module_prefix and declared
// dependency names (pkg.PackageLoader). Without it the loader keeps the
// pre-scoping behaviour: the build-wide prefix table and spelling-derived
// module paths.
type PackageLayoutResolver interface {
	PackageLayout(pkgName string) (dir, prefix string, deps []string, ok bool)
}

// canonicalImportPathFor canonicalises a spelling written by a module of
// package owner ("" = no package context).
func (ml *ModuleLoader) canonicalImportPathFor(owner, path string) (string, error) {
	canonPath := canonicalizeModulePath(path)
	if ml.pkgLoader == nil {
		return canonPath, nil
	}
	key := owner + "\x00" + canonPath
	if r, ok := ml.canonicalOf[key]; ok {
		return r.canon, r.err
	}
	r := canonicalResult{canon: canonPath}
	res, err := ml.resolveInScope(owner, canonPath)
	switch {
	case err != nil:
		r.err = err
	case res != nil:
		r.canon = ml.packageModuleID(res.pkg, res.rest)
		if ml.resolvedByID == nil {
			ml.resolvedByID = make(map[string]resolvedModule)
		}
		ml.resolvedByID[CanonicalModuleID(r.canon)] = resolvedModule{pkg: res.pkg, file: res.file}
	}
	if ml.canonicalOf == nil {
		ml.canonicalOf = make(map[string]canonicalResult)
	}
	ml.canonicalOf[key] = r
	return r.canon, r.err
}

// modulePathOf is the module path ("" or "/<path>") of file inside package
// pkgName, derived from the file so that every spelling of it agrees: the
// first candidate the resolver maps back to the same file. ok is false when
// the layout is unknown, the file is outside the package, or no candidate
// round-trips.
func (ml *ModuleLoader) modulePathOf(pkgName, file string) (string, bool) {
	lr, ok := ml.pkgLoader.(PackageLayoutResolver)
	if !ok {
		return "", false
	}
	dir, prefix, _, ok := lr.PackageLayout(pkgName)
	if !ok {
		return "", false
	}
	want := realFile(file)
	rel, err := filepath.Rel(realFile(dir), want)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	rel = strings.TrimSuffix(filepath.ToSlash(rel), ".ail")
	for _, cand := range modulePathCandidates(rel, prefix, pkgName) {
		f, err := ml.pkgLoader.ResolveImport(pkgName + cand)
		if err == nil && realFile(f) == want {
			return cand, true
		}
	}
	return "", false
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

// caseOnDisk reports the first path component of file below dir whose name on
// disk differs from the spelling only in letter case, as (written, actual).
// On a case-sensitive filesystem such a file does not resolve at all; on a
// case-insensitive one it does, so without this check `cc/Init` and `cc/init`
// would be two modules of one file on macOS and a missing module on Linux.
func caseOnDisk(dir, file string) (written, actual string, mismatch bool) {
	rel, err := filepath.Rel(dir, file)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", "", false
	}
	parent := dir
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		entries, err := os.ReadDir(parent)
		if err != nil {
			return "", "", false
		}
		exact, folded := false, ""
		for _, e := range entries {
			if e.Name() == part {
				exact = true
				break
			}
			if strings.EqualFold(e.Name(), part) {
				folded = e.Name()
			}
		}
		if !exact && folded != "" {
			return part, folded, true
		}
		parent = filepath.Join(parent, part)
	}
	return "", "", false
}

// packageModuleID is the canonical module ID of module rest of package
// pkgName in this build.
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
// `./<name>`. It returns the error the import fails with in owner's scope.
func (ml *ModuleLoader) canonicalizeImportDecl(owner string, imp *ast.ImportDecl) error {
	if imp == nil {
		return nil
	}
	written := imp.Path
	if imp.IsRelative {
		written = "./" + imp.RelativePath
	}
	canon, err := ml.canonicalImportPathFor(owner, imp.Path)
	if err != nil {
		return err
	}
	if canon == imp.Path {
		return nil
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
	return nil
}

// recordIdentity notes that module id was read from file. A file read under
// a second id, or an id read from a second file, is an identity conflict.
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
		ml.identityConflicts = append(ml.identityConflicts,
			fmt.Sprintf("module file %s is loaded under two identities, %s and %s", rf, prev, id))
	} else if !ok {
		ml.identityOfFile[rf] = id
	}
	if prev, ok := ml.fileOfIdentity[id]; ok && prev != rf {
		ml.identityConflicts = append(ml.identityConflicts,
			fmt.Sprintf("module %s is read from two files, %s and %s", id, prev, rf))
	} else if !ok {
		ml.fileOfIdentity[id] = rf
	}
}

// IdentityConflicts reports every source file this loader read under more
// than one module ID, and every module ID it read from more than one file.
// Empty when module identity is single-valued and injective over the build;
// the module pipeline fails the build (LDR003) otherwise.
func (ml *ModuleLoader) IdentityConflicts() []string {
	return append([]string(nil), ml.identityConflicts...)
}

// DisplayModule renders a canonical module ID with the spelling that reached
// it, when the two differ: `pkg/acme/shapes/types/shape (imported as
// shp/types/shape)`. Used by IMP010 (loader.getExport), LDR001 in a package's
// scope (canonical_scope.go) and LDR002 (link.CycleError).
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
