package pipeline

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/loader"
)

// runModuleWithContext runs the pipeline for a module with dependencies
func runModuleWithContext(ctx context.Context, cfg Config, src Source) (Result, error) {
	return runModuleWithCacheDependencies(ctx, cfg, src, productionCacheDependencies())
}

func runModuleWithCacheDependencies(ctx context.Context, cfg Config, src Source, cacheDeps cacheDependencies, afterLoad ...func(map[string]*loader.LoadedModule)) (Result, error) {
	// The orchestration lives in pipeline_module_phases.go (per-call state +
	// ordered phase methods); this signature and its callers are unchanged.
	st := newModulePipelineState(ctx, cfg, src, cacheDeps, afterLoad)
	return st.run()
}

// detectModulePathCollisions returns an error (MOD011) if two *different*
// source files declare the same `module X` header.
//
// Runtime dispatch in serve-api and elsewhere looks up functions by the
// declared module path. If two different files claim the same path, whichever
// one wins the loader cache silently shadows the other — a silent footgun.
//
// The SAME file loaded under two canonical IDs is NOT a collision. This
// happens routinely with `module_prefix`-aliased packages: e.g. the package
// file at `~/.ailang/pkg/.../services/csv_parser.ail` can be loaded under
// both `pkg/sunholo/ailang_parse/services/csv_parser` (direct pkg/ import)
// and `docparse/services/csv_parser` (alias import resolved via
// module_prefix). Both entries point to the same physical file.
//
// We distinguish the two cases by comparing the module's resolved disk path
// (absolute filesystem path after symlinks are followed), which is populated
// by the loader at parse time.
func detectModulePathCollisions(modules map[string]*loader.LoadedModule) error {
	// Step 1: dedupe entries that point to the same physical file. Two
	// canonical IDs backed by the same `filepath.EvalSymlinks(absPath)`
	// represent the same source — keep only the lexically smallest canonical
	// ID for each physical file, so error messages remain deterministic.
	type entry struct {
		canonicalID string
		filePath    string // resolved absolute path (may be "" if unavailable)
		declared    string
	}
	byFile := make(map[string]entry) // resolvedFilePath -> best entry
	var unresolved []entry           // entries where we couldn't determine a disk path

	canonicalIDs := make([]string, 0, len(modules))
	for id := range modules {
		canonicalIDs = append(canonicalIDs, id)
	}
	sort.Strings(canonicalIDs)

	for _, canonicalID := range canonicalIDs {
		mod := modules[canonicalID]
		if mod == nil || mod.File == nil || mod.File.Module == nil {
			continue
		}
		declared := mod.File.Module.Path
		if declared == "" {
			continue
		}

		// Resolve to an absolute, symlink-free path so that two different
		// spellings of the same file are treated as identical.
		resolved := ""
		if mod.File.Path != "" {
			if abs, err := filepath.Abs(mod.File.Path); err == nil {
				if eval, err := filepath.EvalSymlinks(abs); err == nil {
					resolved = eval
				} else {
					resolved = abs
				}
			}
		}

		e := entry{canonicalID: canonicalID, filePath: resolved, declared: declared}
		if resolved == "" {
			// Without a resolvable disk path we can't safely dedupe,
			// so keep the entry around for downstream comparison by
			// canonical ID.
			unresolved = append(unresolved, e)
			continue
		}
		if _, seen := byFile[resolved]; !seen {
			byFile[resolved] = e // first (lexically smallest) canonical ID wins
		}
	}

	// Step 2: check for two *different* files claiming the same declared
	// module path. Iterate in sorted order for stable error messages.
	declaredBy := make(map[string]entry) // declared path -> first claimant

	// Resolved entries first, in deterministic order.
	resolvedPaths := make([]string, 0, len(byFile))
	for p := range byFile {
		resolvedPaths = append(resolvedPaths, p)
	}
	sort.Strings(resolvedPaths)

	checkCollision := func(e entry) error {
		if existing, seen := declaredBy[e.declared]; seen {
			return fmt.Errorf(
				"Error MOD011: module %q is declared in two different files:\n"+
					"  1. %s (canonical: %s)\n"+
					"  2. %s (canonical: %s)\n"+
					"  Fix: rename one of the module declarations so each module path is unique.\n"+
					"  Note: this commonly happens when a local file and a `module_prefix`-aliased\n"+
					"  package file claim the same namespace — runtime dispatch would be ambiguous.",
				e.declared, existing.filePath, existing.canonicalID, e.filePath, e.canonicalID)
		}
		declaredBy[e.declared] = e
		return nil
	}

	for _, p := range resolvedPaths {
		if err := checkCollision(byFile[p]); err != nil {
			return err
		}
	}
	for _, e := range unresolved {
		if err := checkCollision(e); err != nil {
			return err
		}
	}
	return nil
}

// detectModulePrefixOverlap returns an error (MOD013) when the root project and
// one or more dependency packages share the same module_prefix value.
//
// When root and dep share module_prefix = "src", imports of root-only modules
// (e.g. rpc.ail) silently cross the package boundary: the compiler strips the
// shared prefix and builds a pkg/-qualified path that resolves to the dep instead
// of the root, yielding an opaque "not exported by package" error with no pointer
// to the ambiguity. The motoko_agent scenario (root + sunholo/motoko_core both
// using module_prefix = "src") is the canonical reproduction.
//
// Only fires when the root package is involved in the overlap — two deps sharing
// a prefix without the root is allowed (they import each other via pkg/ explicitly).
func detectModulePrefixOverlap(prefixMap map[string]string, rootPkgName string) error {
	if rootPkgName == "" || len(prefixMap) == 0 {
		return nil
	}

	// Group package names by their module_prefix value.
	byPrefix := make(map[string][]string)
	for pkg, prefix := range prefixMap {
		if prefix != "" {
			byPrefix[prefix] = append(byPrefix[prefix], pkg)
		}
	}

	for prefix, pkgs := range byPrefix {
		if len(pkgs) < 2 {
			continue
		}
		// Only fire when the root package is one of the claimants.
		rootInGroup := false
		for _, p := range pkgs {
			if p == rootPkgName {
				rootInGroup = true
				break
			}
		}
		if !rootInGroup {
			continue
		}

		// Collect the dep names (everything except the root) for the message.
		deps := make([]string, 0, len(pkgs)-1)
		for _, p := range pkgs {
			if p != rootPkgName {
				deps = append(deps, p)
			}
		}
		sort.Strings(deps)

		return fmt.Errorf(
			"Error MOD013: ambiguous module ownership under shared module_prefix\n\n"+
				"  Root project:  %s\n"+
				"  Dependency:    %s\n"+
				"  Shared prefix: %q\n\n"+
				"  Both the root project and the dependency use module_prefix = %q.\n"+
				"  Imports of root-only modules (e.g. `import src/core/rpc`) can silently\n"+
				"  cross the package boundary and resolve against the dep instead of the root.\n\n"+
				"  Fix one of:\n"+
				"    1. Remove %s from your [dependencies] if your project IS\n"+
				"       the canonical source of those modules.\n"+
				"    2. Change one side's module_prefix to a distinct value (e.g. rename\n"+
				"       the dep's prefix from %q to a longer, unique segment).\n"+
				"    3. Use explicit pkg/ imports in extension packages that need the dep:\n"+
				"       `import pkg/%s/core/tool_contract (...)`",
			rootPkgName, strings.Join(deps, ", "), prefix, prefix,
			strings.Join(deps, " or "), prefix, strings.Join(deps, "/"),
		)
	}
	return nil
}

// validateModulePath validates that the module declaration matches the canonical path (MOD010).
func validateModulePath(mod *loader.LoadedModule, modID string, cfg *Config) error {
	if mod.File.Module == nil {
		// MOD014: A file with top-level *function declarations* but no `module`
		// header exports nothing, so the entry (e.g. `main`) never runs and the
		// runner silently falls through to a non-module print of unit — exit 0,
		// no output. Fail loudly with an actionable fix instead.
		//
		// Guard on Funcs ONLY, never Statements/Decls. A file that is a lone
		// bare expression to be evaluated (e.g. `1 + 1` -> 2) is parsed into
		// Statements (and, for back-compat, ALSO into Decls — see
		// parser_file.go), with Funcs empty. Gating on Decls would break that
		// eval path. `func main`-style footguns always populate Funcs.
		if len(mod.File.Funcs) > 0 {
			canonicalID := loader.CanonicalModuleID(modID)
			return fmt.Errorf("Error MOD014: no 'module' declaration — this file has top-level "+
				"declarations but no module, so nothing is exported and the entry never runs.\n"+
				"  Fix: add 'module %s' as the first line of the file", canonicalID)
		}
		return nil
	}

	canonicalID := loader.CanonicalModuleID(modID)
	// Exception: std/* modules bypass this check
	if strings.HasPrefix(canonicalID, "std/") || mod.File.Module.Path == canonicalID {
		return nil
	}
	// Exception: pkg/* imports — strip pkg/ prefix before comparing
	// The module declares "vendor/name/module" but the import path is "pkg/vendor/name/module"
	if strings.HasPrefix(canonicalID, "pkg/") {
		stripped := strings.TrimPrefix(canonicalID, "pkg/")
		if mod.File.Module.Path == stripped {
			return nil
		}
		// Also check module_prefix mapping: if the package has module_prefix="docparse",
		// then pkg/sunholo/docparse/services/api can declare "module docparse/services/api"
		if currentModulePrefixMap != nil {
			parts := strings.SplitN(stripped, "/", 3)
			if len(parts) >= 2 {
				pkgName := parts[0] + "/" + parts[1]
				if prefix, ok := currentModulePrefixMap[pkgName]; ok && len(parts) == 3 {
					prefixedPath := prefix + "/" + parts[2]
					if mod.File.Module.Path == prefixedPath {
						return nil
					}
				}
			}
		}
	}

	// Check if relaxation applies
	isTempPath := loader.IsTempPath(modID)
	shouldRelax := cfg.RelaxModules || isTempPath

	if shouldRelax {
		// Emit warning — process-level dedup: each named-test body gets a fresh
		// pipeline cfg, so per-cfg dedup still emitted N copies of the same
		// relaxed-MOD010 warning (observed 2026-08-28: 6 warnings for 1 module).
		warnMOD010Relaxed(mod.File.Module.Path, canonicalID, mod010Reason(isTempPath))
		return nil
	}

	// Strict mode: lead with actionable fix, no search trace noise
	return fmt.Errorf("Error MOD010: module '%s' doesn't match file path '%s'.\n  Fix: use --relax-modules flag or set AILANG_RELAX_MODULES=1\n  Alt: rename module declaration to: module %s\n  Alt: move file to: %s.ail",
		mod.File.Module.Path, canonicalID, canonicalID, mod.File.Module.Path)
}
