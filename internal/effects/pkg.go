package effects

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval"
)

// init registers package-asset effect operations.
func init() {
	RegisterOp("FS", "pkgAssetPath", pkgAssetPath)
}

// pkgAssetPath implements FS.pkgAssetPath(pkgName: string, relPath: string) -> Result[string, string].
//
// Resolves to the absolute filesystem path of an asset shipped inside an
// installed AILANG package's assets/ subdirectory. The package must be
// vendor/name format and present under ~/.ailang/cache/registry/. The most
// recent installed version is selected using package compatibility semver
// ordering when no locked dependency target is available.
//
// Returns:
//   - Ok(absolutePath) when the asset exists
//   - Err("invalid package name: ...") when pkgName is not vendor/name
//   - Err("invalid relative path: ...") when relPath escapes assets/
//   - Err("package not installed: ...") when no version is cached
//   - Err("asset not found: ...") when the file is absent
func pkgAssetPath(ctx *EffContext, args []eval.Value) (eval.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("pkgAssetPath: expected 2 arguments, got %d", len(args))
	}
	pkgArg, ok := args[0].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("pkgAssetPath: arg 0 (pkg_name) must be string, got %T", args[0])
	}
	relArg, ok := args[1].(*eval.StringValue)
	if !ok {
		return nil, fmt.Errorf("pkgAssetPath: arg 1 (rel_path) must be string, got %T", args[1])
	}
	pkgName := pkgArg.Value
	relPath := relArg.Value

	// Package assets live in the user's registry cache, outside any FS
	// sandbox. A sandboxed program could not open the returned path anyway
	// (the root handle refuses it), so say so here instead of handing out
	// a host path it cannot use (M-EXECUTOR-POLICY-HARDENING M1).
	if ctx.fsSandboxed() {
		return fsMakeErr(fmt.Sprintf("package assets are outside the FS sandbox %s: %s", ctx.Env.Sandbox, pkgName)), nil
	}

	parts := strings.SplitN(pkgName, "/", 3)
	if len(parts) != 2 || !validAssetPackagePart(parts[0]) || !validAssetPackagePart(parts[1]) {
		return fsMakeErr(fmt.Sprintf("invalid package name: %q (want vendor/name)", pkgName)), nil
	}

	if relPath == "" {
		return fsMakeErr("invalid relative path: empty"), nil
	}
	if isAbsoluteCrossPlatform(relPath) {
		// Reject paths that would be absolute on ANY mainstream host, not just
		// the current one. A package compiled on Windows might call this with
		// "/etc/passwd" — that's still escape on Linux, even though Windows's
		// filepath.IsAbs would say false.
		return fsMakeErr(fmt.Sprintf("invalid relative path: %q must be relative to assets/", relPath)), nil
	}
	clean := filepath.Clean(relPath)
	if strings.HasPrefix(clean, "..") || strings.Contains(clean, "../") {
		return fsMakeErr(fmt.Sprintf("invalid relative path: %q must not escape assets/", relPath)), nil
	}

	var target PkgAssetTarget
	locked := false
	if ctx != nil {
		target, locked = ctx.PkgAssets[pkgName]
	}
	if locked {
		info, err := os.Stat(target.Dir)
		if target.Dir == "" || err != nil || !info.IsDir() {
			return fsMakeErr(fmt.Sprintf("package not installed: %s@%s (locked by ailang.lock); run 'ailang install %s@%s'", pkgName, target.Version, pkgName, target.Version)), nil
		}
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("pkgAssetPath: cannot resolve user home dir: %w", err)
		}
		pkgDir := filepath.Join(home, ".ailang", "cache", "registry", parts[0], parts[1])
		versions, err := listInstalledVersions(pkgDir)
		if err != nil {
			return fsMakeErr(fmt.Sprintf("cannot list installed versions: %s: %v", pkgName, err)), nil
		}
		if len(versions) == 0 {
			return fsMakeErr(fmt.Sprintf("package not installed: %s", pkgName)), nil
		}
		valid := versions[:0]
		for _, v := range versions {
			if _, err := assetVersion(v); err == nil {
				valid = append(valid, v)
			}
		}
		if len(valid) == 0 {
			return fsMakeErr(fmt.Sprintf("no valid installed versions: %s", pkgName)), nil
		}
		sort.Slice(valid, func(i, j int) bool { return compareAssetVersions(valid[i], valid[j]) < 0 })
		target.Version = valid[len(valid)-1]
		target.Dir = filepath.Join(pkgDir, target.Version)
	}
	chosen := target.Version
	full := filepath.Join(target.Dir, AssetsDir, clean)
	if info, err := os.Stat(full); err != nil || info.IsDir() {
		return fsMakeErr(fmt.Sprintf("asset not found: %s in %s@%s", relPath, pkgName, chosen)), nil
	}
	return fsMakeOk(&eval.StringValue{Value: full}), nil
}

// AssetsDir is duplicated from internal/pkg to preserve layering (effects does
// not depend on the package loader). Keep in sync with pkg.AssetsDir.
const AssetsDir = "assets"

// listInstalledVersions returns subdirectory names under pkgDir, sorted ascending.
// Returns an empty slice (not an error) if pkgDir does not exist.
func listInstalledVersions(pkgDir string) ([]string, error) {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

// validAssetPackagePart keeps package names inside the cache namespace.
func validAssetPackagePart(part string) bool {
	return part != "" && part != "." && part != ".." && !strings.ContainsAny(part, `\:`)
}
