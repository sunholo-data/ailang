package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The import resolution rule (M1b of m-v1-nominal-type-identity).
//
// A spelling S written by a module of package P (the root package, or a
// dependency) is resolved in P's SCOPE — P itself, the packages P declares in
// its [dependencies], and std — in this order:
//
//  1. std/… and file paths are not package spellings.
//  2. pkg/<vendor>/<name>/<m> names that package; it must be P or one P
//     declares (else LDR006).
//  3. <vendor>/<name>/<m> with <vendor>/<name> == P is P's bare self-import.
//  4. <prefix>/<m> names P's own module when <prefix> is P's module_prefix and
//     P has the module; otherwise the module of the ONE declared dependency
//     whose module_prefix is <prefix> and that has it. Two such dependencies
//     is LDR005 (ambiguous); none is LDR001 for a dependency.
//  5. Nothing else. A dependency's spelling that no package in its scope
//     claims fails in its scope: LDR006 when some other package in the build
//     (or the root) would have provided it, LDR001 otherwise. It is never
//     retried in the root's scope. The ROOT keeps its project-relative
//     fallback: a spelling that names a file of the root package is
//     canonicalised to that module; any other path is loaded by path.
//
// Without a PackageLayoutResolver (a resolver that cannot describe packages),
// or outside any package, the pre-scoping rule applies unchanged.

// scopedModule is the package module a spelling resolved to: the package,
// the module path inside it ("" or "/<path>", derived from the file) and the
// file.
type scopedModule struct {
	pkg, rest, file string
}

// resolveInScope resolves canonPath as written by a module of package owner.
// (nil, nil) means the spelling is not a package spelling in this scope and
// Load resolves it as before (std, a file path, the root's project-relative
// fallback, or a resolver error Load reports itself).
func (ml *ModuleLoader) resolveInScope(owner, canonPath string) (*scopedModule, error) {
	if strings.HasPrefix(canonPath, "../") || strings.HasPrefix(canonPath, "std/") || strings.HasSuffix(canonPath, ".ail") {
		return nil, nil
	}
	lr, hasLayouts := ml.pkgLoader.(PackageLayoutResolver)
	unscoped := func() (*scopedModule, error) {
		res := ml.resolveUnscoped(canonPath)
		if res == nil || !hasLayouts {
			return res, nil
		}
		return ml.finishScoped(res.pkg, res.rest, res.file, canonPath)
	}
	if !hasLayouts || owner == "" {
		return unscoped()
	}
	_, ownPrefix, deps, ok := lr.PackageLayout(owner)
	if !ok {
		return unscoped()
	}
	isRoot := owner == ml.currentPackageName
	declared := map[string]bool{owner: true}
	for _, d := range deps {
		declared[d] = true
	}

	// 2. The published path.
	if strings.HasPrefix(canonPath, "pkg/") {
		bare := strings.TrimPrefix(canonPath, "pkg/")
		parts := strings.SplitN(bare, "/", 3)
		if len(parts) < 2 {
			return nil, nil
		}
		name := parts[0] + "/" + parts[1]
		f, err := ml.pkgLoader.ResolveImport(bare)
		if err != nil {
			return nil, nil // Load reports the resolver's error, as before
		}
		if !declared[name] {
			return nil, ml.undeclaredError(owner, canonPath, name)
		}
		return ml.finishScoped(name, strings.TrimPrefix(bare, name), f, canonPath)
	}

	// 3. The bare self-import, scoped to the importer.
	if pathMatchesPackagePrefix(canonPath, owner) {
		f, err := ml.pkgLoader.ResolveImport(canonPath)
		if err != nil {
			if isRoot {
				return nil, nil // Load's self-import branch reports it, as before
			}
			return nil, notInScopeError(owner, canonPath, "pkg/"+canonPath, err)
		}
		return ml.finishScoped(owner, strings.TrimPrefix(canonPath, owner), f, canonPath)
	}

	// 4. A module_prefix spelling: the importer's own prefix, then its declared
	// dependencies' (sorted, so diagnostics are stable).
	first, rest := splitFirstSegment(canonPath)
	// claimers: the packages in scope whose module_prefix is first, with the
	// resolver's error for each that does not have the module.
	var claimers []string
	missErr := map[string]error{}
	if ownPrefix != "" && ownPrefix == first {
		f, err := ml.pkgLoader.ResolveImport(owner + rest)
		if err == nil {
			return ml.finishScoped(owner, rest, f, canonPath)
		}
		claimers = append(claimers, owner)
		missErr[owner] = err
	}
	sortedDeps := append([]string(nil), deps...)
	sort.Strings(sortedDeps)
	var hits []string
	hitFile := ""
	for _, d := range sortedDeps {
		if d == owner {
			continue
		}
		if _, p, _, ok := lr.PackageLayout(d); ok && p == first {
			f, err := ml.pkgLoader.ResolveImport(d + rest)
			if err == nil {
				hits = append(hits, d)
				hitFile = f
				continue
			}
			claimers = append(claimers, d)
			missErr[d] = err
		}
	}
	switch {
	case len(hits) > 1:
		return nil, fmt.Errorf("LDR005: package %s imports %q, but its dependencies %s all provide it under module_prefix %q; "+
			"import the one you mean by its published path (%s)",
			owner, canonPath, strings.Join(hits, " and "), first, "pkg/"+hits[0]+rest)
	case len(hits) == 1:
		return ml.finishScoped(hits[0], rest, hitFile, canonPath)
	}

	// 5. Nothing in the importer's scope provides it.
	if isRoot {
		return ml.rootFile(canonPath)
	}
	if len(claimers) > 0 {
		// The first claimer is the importer itself when its prefix matched,
		// else the first declared dependency with that prefix.
		c := claimers[0]
		return nil, notInScopeError(owner, canonPath, "pkg/"+c+rest, missErr[c])
	}
	if other := ml.providerOutsideScope(canonPath); other != "" {
		return nil, ml.undeclaredError(owner, canonPath, other)
	}
	return nil, notInScopeError(owner, canonPath, "", nil)
}

// finishScoped completes a resolution: the spelling must match the file's
// case on disk, and the module path is derived from the file.
func (ml *ModuleLoader) finishScoped(pkgName, rest, file, spelling string) (*scopedModule, error) {
	if lr, ok := ml.pkgLoader.(PackageLayoutResolver); ok {
		if dir, _, _, ok := lr.PackageLayout(pkgName); ok {
			absDir, _ := filepath.Abs(dir)
			absFile, _ := filepath.Abs(file)
			if written, actual, bad := caseOnDisk(absDir, absFile); bad {
				return nil, fmt.Errorf("LDR001: import %q names %q in package %s, but the file on disk is spelled %q; "+
					"module paths are case-sensitive (on a case-sensitive filesystem this import is not found)",
					spelling, written, pkgName, actual)
			}
		}
	}
	if p, ok := ml.modulePathOf(pkgName, file); ok {
		rest = p
	}
	return &scopedModule{pkg: pkgName, rest: rest, file: file}, nil
}

// rootFile canonicalises a root-package spelling that is a project-relative
// path to a file of the root package (`import util` for util.ail in an
// unprefixed root): it names that module, so it shares the identity of the
// package spellings of the same file. Any other path is left to Load.
func (ml *ModuleLoader) rootFile(canonPath string) (*scopedModule, error) {
	root := ml.currentPackageName
	if root == "" {
		return nil, nil
	}
	proj := filepath.Join(ml.basePath, canonPath) + ".ail"
	if _, err := os.Stat(proj); err != nil {
		return nil, nil
	}
	if _, ok := ml.modulePathOf(root, proj); !ok {
		return nil, nil
	}
	return ml.finishScoped(root, "", proj, canonPath)
}

// providerOutsideScope reports which package — outside the importer's scope —
// the pre-scoping rule would have resolved canonPath to: the root (bare
// self-import) or a package in the build-wide prefix table. "" if none.
func (ml *ModuleLoader) providerOutsideScope(canonPath string) string {
	if res := ml.resolveUnscoped(canonPath); res != nil {
		return res.pkg
	}
	return ""
}

// resolveUnscoped is the pre-scoping rule: the published path, the ROOT's bare
// self-import, then the build-wide prefix table (root + its direct
// dependencies, sorted per prefix). The module path is the spelling's.
func (ml *ModuleLoader) resolveUnscoped(canonPath string) *scopedModule {
	if strings.HasPrefix(canonPath, "pkg/") {
		bare := strings.TrimPrefix(canonPath, "pkg/")
		parts := strings.SplitN(bare, "/", 3)
		if len(parts) < 2 {
			return nil
		}
		f, err := ml.pkgLoader.ResolveImport(bare)
		if err != nil {
			return nil
		}
		name := parts[0] + "/" + parts[1]
		return &scopedModule{pkg: name, rest: strings.TrimPrefix(bare, name), file: f}
	}
	if ml.currentPackageName != "" && pathMatchesPackagePrefix(canonPath, ml.currentPackageName) {
		f, err := ml.pkgLoader.ResolveImport(canonPath)
		if err != nil {
			return nil
		}
		return &scopedModule{pkg: ml.currentPackageName, rest: strings.TrimPrefix(canonPath, ml.currentPackageName), file: f}
	}
	first, rest := splitFirstSegment(canonPath)
	for _, name := range ml.modulePrefixMap[first] {
		if f, err := ml.pkgLoader.ResolveImport(name + rest); err == nil {
			return &scopedModule{pkg: name, rest: rest, file: f}
		}
	}
	return nil
}

func splitFirstSegment(p string) (first, rest string) {
	if idx := strings.Index(p, "/"); idx >= 0 {
		return p[:idx], p[idx:]
	}
	return p, ""
}

// undeclaredError is LDR006: owner's import resolves to a package outside its
// scope.
func (ml *ModuleLoader) undeclaredError(owner, spelling, other string) error {
	if other == ml.currentPackageName && owner != other {
		return fmt.Errorf("LDR006: package %s imports %q, which is a module of the root package %s; "+
			"a dependency cannot import the package that depends on it — move the code into %s or into a package both declare",
			owner, spelling, other, owner)
	}
	return fmt.Errorf("LDR006: package %s imports %q from package %s, which it does not declare; "+
		"add %q to [dependencies] in %s's ailang.toml, or fix the import",
		owner, spelling, other, other, owner)
}

// notInScopeError is LDR001 for a spelling that does not resolve in owner's
// scope; attempt is the published path it was looked up as, when known.
func notInScopeError(owner, spelling, attempt string, cause error) error {
	name := spelling
	if attempt != "" {
		name = DisplayModule(attempt, spelling)
	}
	msg := fmt.Sprintf("LDR001: module %s, imported by package %s, was not found in its scope (%s itself and the packages it declares)",
		name, owner, owner)
	if cause != nil {
		msg += ": " + cause.Error()
	}
	return fmt.Errorf("%s", msg)
}
