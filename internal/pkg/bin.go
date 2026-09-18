package pkg

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

// M-PKG-BIN-ENTRYPOINTS (v0.40.0): a package's [bin] table names commands;
// `ailang install` turns each into a shim on PATH. A shim is a generated
// script that execs `ailang run --package-dir <pkg> <file> -- "$@"`, so the
// installed package directory is the program root: its own ailang.toml is
// found beside the file (ailang#671 anchoring), the lock EnsureLock wrote
// beside it resolves the dependencies, and PackageDir lets MOD010 accept the
// package's canonical module paths. Nothing here resolves modules or
// dependencies itself — it reuses ResolveModuleToFile and ResolveDependencies.

// ShimHeaderPrefix is the marker line every shim carries; `ailang bin list`
// and `ailang bin uninstall` identify shims by it, so there is no separate
// registry of installed bins to drift from the files on disk.
const ShimHeaderPrefix = "# ailang-bin: "

// binNameRe is what a bin name may look like: it becomes a file on PATH, so
// it is restricted to what every shell and filesystem accepts unquoted.
var binNameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

// ValidateBinName rejects names that could not be a portable command name.
func ValidateBinName(name string) error {
	if !binNameRe.MatchString(name) {
		return fmt.Errorf("[bin].%s: bin names must match %s (lowercase, digits, '.', '_', '-'; no leading dot or dash)", name, binNameRe.String())
	}
	if name == "." || name == ".." || name == "ailang" {
		return fmt.Errorf("[bin].%s: reserved name", name)
	}
	return nil
}

// ResolveBinFile returns the source file a bin entry names, or an error that
// says which candidate paths were tried. The module is resolved with the same
// rules as an import (flat root, src/, canonical layout, module_prefix dir).
func ResolveBinFile(pkgDir string, manifest *PackageManifest, name string, spec BinSpec) (string, error) {
	file := ResolveModuleToFile(pkgDir, manifest.Package.Name, spec.Module)
	if file == "" && manifest.Package.ModulePrefix != "" && !strings.HasPrefix(spec.Module, manifest.Package.ModulePrefix+"/") {
		file = ResolveModuleToFile(pkgDir, manifest.Package.Name, manifest.Package.ModulePrefix+"/"+spec.Module)
	}
	if file == "" {
		return "", fmt.Errorf("bin %q: module %q does not resolve to a file under %s (tried src/<module>.ail, <module>.ail, <package>/<module>.ail)", name, spec.Module, pkgDir)
	}
	abs, err := filepath.Abs(file)
	if err != nil {
		return "", err
	}
	return abs, nil
}

// VerifyBinEntrypoints checks every [bin] entry resolves to a file that
// exports its entry function. Run by `ailang publish` before a tarball exists
// and by `ailang install` before a shim is written, so a bin that cannot run
// is refused at the step that can name the fix rather than at first use.
func VerifyBinEntrypoints(pkgDir string, manifest *PackageManifest) error {
	names := make([]string, 0, len(manifest.Bin))
	for name := range manifest.Bin {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		spec := manifest.Bin[name]
		file, err := ResolveBinFile(pkgDir, manifest, name, spec)
		if err != nil {
			return err
		}
		ok, err := fileExportsFunc(file, spec.EffectiveEntry())
		if err != nil {
			return fmt.Errorf("bin %q: %w", name, err)
		}
		if !ok {
			rel, _ := filepath.Rel(pkgDir, file)
			return fmt.Errorf("bin %q: module %s resolves to %s but it does not export func %s", name, spec.Module, rel, spec.EffectiveEntry())
		}
	}
	return nil
}

// fileExportsFunc reports whether the file declares `export [pure] func <name>`.
// A text scan, not a parse: publish already compiles the package through the
// quality report, and install runs against a tarball publish accepted.
func fileExportsFunc(file, fn string) (bool, error) {
	f, err := os.Open(file)
	if err != nil {
		return false, err
	}
	defer f.Close()
	re := regexp.MustCompile(`^\s*export\s+(pure\s+)?func\s+` + regexp.QuoteMeta(fn) + `\s*[\[(]`)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		if re.MatchString(sc.Text()) {
			return true, nil
		}
	}
	return false, sc.Err()
}

// EnsureLock makes pkgDir a runnable program root: when no ailang.lock is
// beside the manifest, it resolves the manifest's dependencies (fetching
// missing registry packages into the cache, as `ailang lock` does) and writes
// one. An existing lock is kept — for a registry package it is a pure
// function of the pinned manifest, and for a developer's checkout it is the
// developer's. Returns the number of packages locked and whether it wrote.
func EnsureLock(pkgDir, generator, ailangVersion string) (locked int, wrote bool, err error) {
	if lf, err := LoadLockFile(pkgDir); err == nil {
		return len(lf.Packages), false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return 0, false, fmt.Errorf("existing %s in %s is unreadable: %w", LockFileName, pkgDir, err)
	}
	manifest, err := LoadManifest(pkgDir)
	if err != nil {
		return 0, false, err
	}
	var packages []LockedPackage
	if len(manifest.Dependencies) > 0 {
		resolved, err := ResolveDependencies(manifest, pkgDir)
		if err != nil {
			return 0, false, fmt.Errorf("resolving dependencies of %s: %w", manifest.Package.Name, err)
		}
		packages = make([]LockedPackage, len(resolved))
		for i, r := range resolved {
			packages[i] = LockedPackage(r)
		}
	}
	lf := NewLockFile(packages, generator)
	lf.AILANGVersion = ailangVersion
	if err := lf.Save(pkgDir); err != nil {
		return 0, false, err
	}
	return len(packages), true, nil
}

// DefaultBinDir is where shims go: ~/.ailang/bin, beside the package cache.
func DefaultBinDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ailang", "bin"), nil
}

// ShimPath is the file a bin installs to; Windows shims are .cmd files.
func ShimPath(binDir, name string) string { return shimPath(runtime.GOOS, binDir, name) }

func shimPath(goos, binDir, name string) string {
	if goos == "windows" {
		return filepath.Join(binDir, name+".cmd")
	}
	return filepath.Join(binDir, name)
}

// Shim describes one installed bin, as parsed back from its file.
type Shim struct {
	Name    string // command name
	Package string // vendor/name
	Version string
	Path    string // shim file
	Ailang  string // interpreter the shim execs
	PkgDir  string // package root the shim runs from
}

// WriteShim writes the shim for one [bin] entry and returns its path. The
// file execs the installing ailang binary with the package as program root;
// everything it needs is in its own header, so `bin list`/`bin uninstall`
// read the file rather than a registry.
func WriteShim(binDir, ailangBin, pkgDir string, manifest *PackageManifest, name string, spec BinSpec) (string, error) {
	file, err := ResolveBinFile(pkgDir, manifest, name, spec)
	if err != nil {
		return "", err
	}
	absPkgDir, err := filepath.Abs(pkgDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", fmt.Errorf("creating bin dir %s: %w", binDir, err)
	}
	path := ShimPath(binDir, name)
	body := shimBody(runtime.GOOS, ailangBin, absPkgDir, file, manifest, name, spec)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		return "", fmt.Errorf("writing shim %s: %w", path, err)
	}
	return path, nil
}

// shimBody renders the shim for goos: POSIX sh, or a .cmd batch file on
// Windows. Both carry the same header lines ("# k: v" / "rem k: v") so
// ReadShim parses either.
func shimBody(goos, ailangBin, absPkgDir, file string, manifest *PackageManifest, name string, spec BinSpec) string {
	header := fmt.Sprintf("%s%s@%s %s", ShimHeaderPrefix, manifest.Package.Name, manifest.Package.Version, name)
	regen := fmt.Sprintf("# regenerate: ailang install %s@%s", manifest.Package.Name, manifest.Package.Version)
	runArgs := []string{"run", "--quiet", "--package-dir", absPkgDir, "--entry", spec.EffectiveEntry(), "--caps", spec.EffectiveCaps()}
	runArgs = append(runArgs, spec.RunFlags...)
	runArgs = append(runArgs, file, "--")

	var sb strings.Builder
	if goos == "windows" {
		sb.WriteString("@echo off\r\n")
		sb.WriteString("rem " + strings.TrimPrefix(header, "# ") + "\r\n")
		sb.WriteString("rem ailang: " + ailangBin + "\r\n")
		sb.WriteString("rem package-dir: " + absPkgDir + "\r\n")
		sb.WriteString("rem " + strings.TrimPrefix(regen, "# ") + "\r\n")
		sb.WriteString(fmt.Sprintf("\"%s\"", ailangBin))
		for _, a := range runArgs {
			sb.WriteString(fmt.Sprintf(" \"%s\"", a))
		}
		sb.WriteString(" %*\r\n")
		return sb.String()
	}
	sb.WriteString("#!/bin/sh\n")
	sb.WriteString(header + "\n")
	sb.WriteString("# ailang: " + ailangBin + "\n")
	sb.WriteString("# package-dir: " + absPkgDir + "\n")
	sb.WriteString(regen + "\n")
	sb.WriteString("exec " + shellQuote(ailangBin))
	for _, a := range runArgs {
		sb.WriteString(" " + shellQuote(a))
	}
	sb.WriteString(" \"$@\"\n")
	return sb.String()
}

// shellQuote single-quotes s for POSIX sh; "--" and plain words stay readable.
func shellQuote(s string) string {
	if s != "" && !strings.ContainsAny(s, " \t\n'\"\\$`!*?[](){}<>|&;#~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ReadShim parses a shim file. Returns (nil, nil) for a file that is not an
// ailang shim, so a user's own scripts in the same directory are ignored.
func ReadShim(path string) (*Shim, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := &Shim{Path: path}
	sc := bufio.NewScanner(f)
	// Header lines are "# key: value" (sh) or "rem key: value" (cmd).
	for i := 0; sc.Scan() && i < 8; i++ {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "# ") {
			line = strings.TrimPrefix(line, "# ")
		} else if strings.HasPrefix(line, "rem ") {
			line = strings.TrimPrefix(line, "rem ")
		} else {
			continue
		}
		key, value, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}
		switch key {
		case "ailang-bin":
			fields := strings.Fields(value)
			if len(fields) != 2 {
				return nil, fmt.Errorf("%s: malformed shim header %q", path, line)
			}
			pkgVer := strings.SplitN(fields[0], "@", 2)
			if len(pkgVer) != 2 {
				return nil, fmt.Errorf("%s: malformed shim header %q", path, line)
			}
			s.Package, s.Version, s.Name = pkgVer[0], pkgVer[1], fields[1]
		case "ailang":
			s.Ailang = value
		case "package-dir":
			s.PkgDir = value
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if s.Name == "" {
		return nil, nil
	}
	return s, nil
}

// ListShims returns every ailang shim in binDir, sorted by name. A missing
// directory is an empty list, not an error.
func ListShims(binDir string) ([]Shim, error) {
	entries, err := os.ReadDir(binDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var shims []Shim
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		s, err := ReadShim(filepath.Join(binDir, e.Name()))
		if err != nil {
			return nil, err
		}
		if s != nil {
			shims = append(shims, *s)
		}
	}
	sort.Slice(shims, func(i, j int) bool { return shims[i].Name < shims[j].Name })
	return shims, nil
}

// RemoveShim deletes the shim for name, refusing to delete a file in binDir
// that is not an ailang shim.
func RemoveShim(binDir, name string) (*Shim, error) {
	path := ShimPath(binDir, name)
	s, err := ReadShim(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("no bin %q installed in %s", name, binDir)
	}
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("%s exists but is not an ailang shim; not removing it", path)
	}
	if err := os.Remove(path); err != nil {
		return nil, err
	}
	return s, nil
}
