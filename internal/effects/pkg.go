package effects

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
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
// vendor/name format. When the program runs under an ailang.lock, the locked
// directory is used (the same one its modules were compiled from, #1772);
// otherwise the highest semver installed under the registry cache
// (~/.ailang/cache/registry, or AILANG_PACKAGE_ROOT) is used.
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
	clean := filepath.Clean(filepath.FromSlash(strings.ReplaceAll(relPath, `\`, "/")))
	if strings.HasPrefix(clean, "..") || strings.Contains(clean, "../") {
		return fsMakeErr(fmt.Sprintf("invalid relative path: %q must not escape assets/", relPath)), nil
	}

	base, label, errVal, err := resolvePkgAssetBase(ctx, pkgName, parts)
	if err != nil || errVal != nil {
		return errVal, err
	}

	full := filepath.Join(base, AssetsDir, clean)
	if info, err := os.Stat(full); err != nil || info.IsDir() {
		return fsMakeErr(fmt.Sprintf("asset not found: %s in %s@%s", relPath, pkgName, label)), nil
	}
	return fsMakeOk(&eval.StringValue{Value: full}), nil
}

// AssetsDir is duplicated from internal/pkg to preserve layering (effects does
// not depend on the package loader). Keep in sync with pkg.AssetsDir.
const AssetsDir = "assets"

// resolvePkgAssetBase picks the package directory whose assets/ to read, and a
// label for messages. The lock wins: two machines with different versions
// cached must read the same data (A1). Without a lock it falls back to the
// highest installed version by semver, not by string (0.10.0 > 0.9.0).
func resolvePkgAssetBase(ctx *EffContext, pkgName string, parts []string) (string, string, eval.Value, error) {
	if target, ok := ctx.lockedPkgAsset(pkgName); ok {
		label := target.Version
		installName := pkgName
		if label != "" {
			installName += "@" + label
		} else {
			label = "locked"
		}
		info, err := os.Stat(target.Dir)
		if target.Dir == "" || err != nil || !info.IsDir() {
			return "", "", fsMakeErr(fmt.Sprintf("package not installed: %s (locked by ailang.lock); run 'ailang install %s'", installName, installName)), nil
		}
		return target.Dir, label, nil, nil
	}
	root := config.PackageRoot()
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", nil, fmt.Errorf("pkgAssetPath: cannot resolve user home dir: %w", err)
		}
		root = filepath.Join(home, ".ailang", "cache", "registry")
	}
	pkgDir := filepath.Join(root, parts[0], parts[1])
	versions, err := listInstalledVersions(pkgDir)
	if err != nil {
		return "", "", fsMakeErr(fmt.Sprintf("cannot list installed versions: %s: %v", pkgName, err)), nil
	}
	if len(versions) == 0 {
		return "", "", fsMakeErr(fmt.Sprintf("package not installed: %s", pkgName)), nil
	}
	chosen := versions[len(versions)-1]
	if _, valid := parseSemverParts(chosen); !valid {
		return "", "", fsMakeErr(fmt.Sprintf("no valid installed versions: %s", pkgName)), nil
	}
	return filepath.Join(pkgDir, chosen), chosen, nil, nil
}

// listInstalledVersions returns subdirectory names under pkgDir, sorted by
// semver ascending (non-semver names sort first, by string).
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
	sort.Slice(out, func(i, j int) bool { return semverLess(out[i], out[j]) })
	return out, nil
}

// semverLess orders MAJOR.MINOR.PATCH numerically; a pre-release sorts before
// its release per SemVer 2.0.0 §11.4 (dot-separated numeric identifiers
// compared as integers, numeric < non-numeric, shorter prefix < longer),
// build metadata (+...) is ignored, and names that do not parse sort before
// any that do.
func semverLess(a, b string) bool {
	av, aok := parseSemverParts(a)
	bv, bok := parseSemverParts(b)
	if aok != bok {
		return !aok
	}
	if !aok {
		return a < b
	}
	for i := 0; i < 3; i++ {
		if av.nums[i] != bv.nums[i] {
			return av.nums[i] < bv.nums[i]
		}
	}
	if (av.pre == "") != (bv.pre == "") {
		return av.pre != ""
	}
	if av.pre == "" {
		return false
	}
	return semverPrereleaseLess(av.pre, bv.pre)
}

// semverPrereleaseLess compares two non-empty SemVer pre-release strings per
// SemVer 2.0.0 §11.4.
func semverPrereleaseLess(a, b string) bool {
	aIDs := strings.Split(a, ".")
	bIDs := strings.Split(b, ".")
	for i := 0; i < len(aIDs) && i < len(bIDs); i++ {
		ai, bi := aIDs[i], bIDs[i]
		if ai == bi {
			continue
		}
		aNum, bNum := isDigitsOnly(ai), isDigitsOnly(bi)
		if aNum && bNum {
			if cmp := compareDigitStrings(ai, bi); cmp != 0 {
				return cmp < 0
			}
			continue
		}
		if aNum != bNum {
			// SemVer 2.0.0 §11.4.3: numeric identifiers always have lower
			// precedence than non-numeric identifiers.
			return aNum
		}
		// SemVer 2.0.0 §11.4.2: non-numeric identifiers compare lexically in ASCII.
		return ai < bi
	}
	// SemVer 2.0.0 §11.4.4: a smaller set of pre-release fields has lower
	// precedence when all preceding identifiers are equal.
	return len(aIDs) < len(bIDs)
}

func isDigitsOnly(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func compareDigitStrings(a, b string) int {
	at := strings.TrimLeft(a, "0")
	bt := strings.TrimLeft(b, "0")
	if len(at) != len(bt) {
		if len(at) < len(bt) {
			return -1
		}
		return 1
	}
	if at < bt {
		return -1
	}
	if at > bt {
		return 1
	}
	return 0
}

type semverParts struct {
	nums [3]int
	pre  string
}

func parseSemverParts(v string) (semverParts, bool) {
	var p semverParts
	v = strings.TrimPrefix(v, "v")
	// SemVer 2.0.0 §10: strip build metadata (+...) before splitting pre-release.
	if plus := strings.IndexByte(v, '+'); plus >= 0 {
		if plus == len(v)-1 {
			return p, false
		}
		v = v[:plus]
	}
	if dash := strings.IndexByte(v, '-'); dash >= 0 {
		p.pre = v[dash+1:]
		if p.pre == "" {
			return p, false
		}
		for _, id := range strings.Split(p.pre, ".") {
			if id == "" {
				return p, false
			}
		}
		v = v[:dash]
	}
	fields := strings.Split(v, ".")
	if len(fields) != 3 {
		return p, false
	}
	for i, f := range fields {
		n, err := strconv.Atoi(f)
		if err != nil || n < 0 {
			return p, false
		}
		p.nums[i] = n
	}
	return p, true
}

// lockedPkgAsset is nil-safe. Versioned snapshots take precedence; PkgDirs
// remains supported for existing embedders that supply directory-only targets.
func (ctx *EffContext) lockedPkgAsset(pkgName string) (PkgAssetTarget, bool) {
	if ctx == nil {
		return PkgAssetTarget{}, false
	}
	if target, ok := ctx.PkgAssets[pkgName]; ok {
		return target, true
	}
	dir, ok := ctx.PkgDirs[pkgName]
	return PkgAssetTarget{Dir: dir}, ok
}

// validAssetPackagePart keeps package identities inside the cache namespace.
func validAssetPackagePart(part string) bool {
	return part != "" && part != "." && part != ".." && !strings.ContainsAny(part, `\:`)
}
