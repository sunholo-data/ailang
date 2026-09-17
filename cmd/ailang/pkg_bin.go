package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// M-PKG-BIN-ENTRYPOINTS (v0.40.0): `ailang install` shims a package's [bin]
// commands onto PATH; `ailang bin list|uninstall` manages the shims.

// resolveBinDir applies the --bin-dir override or the default ~/.ailang/bin.
func resolveBinDir(flagValue string) (string, error) {
	if flagValue != "" {
		return filepath.Abs(flagValue)
	}
	return pkg.DefaultBinDir()
}

// installingAilangBinary is the interpreter a shim execs: this binary, with
// symlinks resolved so a `go install` that replaces the file keeps working.
func installingAilangBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating the ailang binary for the shim: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

// installLocalBins is `ailang install --path <dir>`: the developer loop. The
// checkout itself is the program root, so the developer's lock is used when
// present and generated otherwise.
func installLocalBins(dir, binDirFlag string) error {
	pkgDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	manifest, err := pkg.LoadManifest(pkgDir)
	if err != nil {
		return fmt.Errorf("--path %s: %w", dir, err)
	}
	if len(manifest.Bin) == 0 {
		return fmt.Errorf("%s@%s declares no [bin] commands in %s — nothing to install", manifest.Package.Name, manifest.Package.Version, filepath.Join(pkgDir, pkg.ManifestFile))
	}
	return installBins(pkgDir, binDirFlag)
}

// installBins writes a shim for every [bin] entry of the package at pkgDir.
// A package without [bin] is a no-op, so registry installs of libraries are
// unchanged.
func installBins(pkgDir, binDirFlag string) error {
	manifest, err := pkg.LoadManifest(pkgDir)
	if err != nil {
		return err
	}
	if len(manifest.Bin) == 0 {
		return nil
	}
	binDir, err := resolveBinDir(binDirFlag)
	if err != nil {
		return err
	}
	ailangBin, err := installingAilangBinary()
	if err != nil {
		return err
	}
	return installBinsTo(os.Stdout, pkgDir, binDir, ailangBin, manifest)
}

// installBinsTo is the testable core: lock, verify, shim, PATH hint.
func installBinsTo(out io.Writer, pkgDir, binDir, ailangBin string, manifest *pkg.PackageManifest) error {
	locked, wrote, err := pkg.EnsureLock(pkgDir, fmt.Sprintf("ailang install %s", Version), Version)
	if err != nil {
		return err
	}
	lockPath := filepath.Join(pkgDir, pkg.LockFileName)
	if wrote {
		fmt.Fprintf(out, "%s Lock: %s (%d packages)\n", green("✓"), lockPath, locked)
	} else {
		fmt.Fprintf(out, "%s Lock: %s (existing, %d packages)\n", green("✓"), lockPath, locked)
	}

	if err := pkg.VerifyBinEntrypoints(pkgDir, manifest); err != nil {
		return err
	}

	names := make([]string, 0, len(manifest.Bin))
	for name := range manifest.Bin {
		names = append(names, name)
	}
	sort.Strings(names)
	notOnPath := false
	for _, name := range names {
		spec := manifest.Bin[name]
		path, err := pkg.WriteShim(binDir, ailangBin, pkgDir, manifest, name, spec)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s bin: %s → %s  (%s@%s, module %s, entry %s, caps %s)\n",
			green("✓"), name, path, manifest.Package.Name, manifest.Package.Version, spec.Module, spec.EffectiveEntry(), spec.EffectiveCaps())
		if !reportPathStatus(out, name, path) {
			notOnPath = true
		}
	}
	if notOnPath {
		fmt.Fprintf(out, "%s %s is not on your PATH. Add:  export PATH=\"%s:$PATH\"\n", yellow("⚠"), binDir, binDir)
	}
	return nil
}

// reportPathStatus reports whether `name` is found on PATH at all, and warns
// when it is found but shadowed by another file earlier on PATH (the repo's
// own launcher, typically) — silence there would leave the user running the
// old command and blaming the new one.
func reportPathStatus(out io.Writer, name, shimPath string) bool {
	found, err := exec.LookPath(name)
	if err != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(found); err == nil {
		found = resolved
	}
	want := shimPath
	if resolved, err := filepath.EvalSymlinks(shimPath); err == nil {
		want = resolved
	}
	if filepath.Clean(found) != filepath.Clean(want) {
		fmt.Fprintf(out, "%s `%s` on your PATH resolves to %s, not the shim; %s must come earlier in PATH\n", yellow("⚠"), name, found, filepath.Dir(shimPath))
	}
	return true
}

// binCommand is `ailang bin list|uninstall <name> [--bin-dir DIR]`.
func binCommand(args []string) error {
	flagSet := flag.NewFlagSet("bin", flag.ExitOnError)
	helpFlag := flagSet.Bool("help", false, "Show help")
	binDirFlag := flagSet.String("bin-dir", "", "Shim directory (default: ~/.ailang/bin)")
	if err := flagSet.Parse(args); err != nil {
		return err
	}
	usage := func() {
		fmt.Println("Usage: ailang bin list [--bin-dir DIR]")
		fmt.Println("       ailang bin uninstall <name> [--bin-dir DIR]")
		fmt.Println()
		fmt.Println("Manage the commands `ailang install` shimmed onto PATH from packages that")
		fmt.Println("declare [bin] in ailang.toml. To (re)install one: ailang install vendor/name")
	}
	if *helpFlag || flagSet.NArg() < 1 {
		usage()
		return nil
	}
	// Flags may follow the subcommand (`bin uninstall x --bin-dir X`): the
	// flag package stops at the first positional, so hoist flags to the front
	// and parse once more.
	if err := flagSet.Parse(hoistFlagsWith(flagSet.Args(), map[string]bool{"--bin-dir": true, "-bin-dir": true})); err != nil {
		return err
	}
	sub := flagSet.Arg(0)
	binDir, err := resolveBinDir(*binDirFlag)
	if err != nil {
		return err
	}
	switch sub {
	case "list":
		shims, err := pkg.ListShims(binDir)
		if err != nil {
			return err
		}
		if len(shims) == 0 {
			fmt.Printf("No ailang bins installed in %s\n", binDir)
			return nil
		}
		for _, s := range shims {
			line := fmt.Sprintf("%-20s %s@%s  %s", s.Name, s.Package, s.Version, s.Path)
			if s.Ailang != "" {
				if _, err := os.Stat(s.Ailang); err != nil {
					line += fmt.Sprintf("  %s interpreter missing: %s (reinstall: ailang install %s@%s)", yellow("⚠"), s.Ailang, s.Package, s.Version)
				}
			}
			if s.PkgDir != "" {
				if _, err := os.Stat(filepath.Join(s.PkgDir, pkg.ManifestFile)); err != nil {
					line += fmt.Sprintf("  %s package missing: %s", yellow("⚠"), s.PkgDir)
				}
			}
			fmt.Println(line)
		}
		notOnPath := false
		for _, s := range shims {
			if !reportPathStatus(os.Stdout, s.Name, s.Path) {
				notOnPath = true
			}
		}
		if notOnPath {
			fmt.Printf("%s %s is not on your PATH. Add:  export PATH=\"%s:$PATH\"\n", yellow("⚠"), binDir, binDir)
		}
		return nil
	case "uninstall", "remove", "rm":
		if flagSet.NArg() != 2 {
			return fmt.Errorf("usage: ailang bin uninstall <name>")
		}
		s, err := pkg.RemoveShim(binDir, flagSet.Arg(1))
		if err != nil {
			return err
		}
		fmt.Printf("%s Removed %s (%s@%s)\n", green("✓"), s.Path, s.Package, s.Version)
		return nil
	default:
		usage()
		return fmt.Errorf("unknown bin subcommand %q", sub)
	}
}
