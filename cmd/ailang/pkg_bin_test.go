package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// M-PKG-BIN-ENTRYPOINTS: `ailang install` turns a package's [bin] table into
// shims that run from any cwd, and refuses a bin that could not run.

// writeGreeterPackage is a package with a [bin] and no external
// dependencies, so EnsureLock never touches the network.
func writeGreeterPackage(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `[package]
name = "test/greeter"
version = "0.1.0"
edition = "1"

[exports]
modules = ["test/greeter/lib"]

[bin]
greet = "cli"
greet-args = { module = "cli", entry = "echoArgs", caps = "IO,Env" }
`
	lib := "module test/greeter/lib\nexport pure func greeting(name: string) -> string = \"hello, ${name}\"\n"
	cli := `module test/greeter/cli
import std/io (println)
import std/env (getArgs)
import std/list (length)
import ./lib (greeting)

export func main() -> () ! {IO, Env} =
  let a = getArgs(()) in
  println("${greeting("bin")} n=${show(length(a))}")

export func echoArgs() -> () ! {IO, Env} =
  println(show(getArgs(())))
`
	for name, body := range map[string]string{pkg.ManifestFile: manifest, "lib.ail": lib, "cli.ail": cli} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInstallPath_ShimRunsFromAnyCwd(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh shims; the .cmd shim is verified by inspection until a Windows runner exercises it")
	}
	ailang := buildAilang(t)
	pkgDir := filepath.Join(t.TempDir(), "greeter")
	writeGreeterPackage(t, pkgDir)
	binDir := filepath.Join(t.TempDir(), "bin")

	manifest, err := pkg.LoadManifest(pkgDir)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := installBinsTo(&out, pkgDir, binDir, ailang, manifest); err != nil {
		t.Fatalf("installBinsTo: %v\n%s", err, out.String())
	}
	for _, want := range []string{"Lock: " + filepath.Join(pkgDir, pkg.LockFileName), "bin: greet →", "bin: greet-args →", "not on your PATH"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("install output missing %q:\n%s", want, out.String())
		}
	}

	// The shim runs from an unrelated cwd, receives positional args verbatim
	// (including ones that look like flags), and keeps stderr clean.
	elsewhere := t.TempDir()
	stdout, stderr, code := testutil.RunBounded(t, elsewhere, 60*time.Second, filepath.Join(binDir, "greet"), "--flag", "two words")
	if code != 0 || strings.TrimSpace(stdout) != "hello, bin n=2" {
		t.Fatalf("greet: exit %d stdout %q stderr %q", code, stdout, stderr)
	}
	if strings.Contains(stderr, "auto-granted") || strings.Contains(stderr, "Type checking") {
		t.Errorf("shim stderr must not carry runner progress: %q", stderr)
	}
	stdout, stderr, code = testutil.RunBounded(t, elsewhere, 60*time.Second, filepath.Join(binDir, "greet-args"), "a", "--b=c")
	if code != 0 || strings.TrimSpace(stdout) != "[a, --b=c]" {
		t.Fatalf("greet-args: exit %d stdout %q stderr %q", code, stdout, stderr)
	}

	// Reinstall is idempotent and uninstall removes exactly the one shim.
	out.Reset()
	if err := installBinsTo(&out, pkgDir, binDir, ailang, manifest); err != nil {
		t.Fatalf("second install: %v", err)
	}
	if !strings.Contains(out.String(), "existing, 0 packages") {
		t.Errorf("second install must keep the lock:\n%s", out.String())
	}
	if _, err := pkg.RemoveShim(binDir, "greet-args"); err != nil {
		t.Fatal(err)
	}
	shims, _ := pkg.ListShims(binDir)
	if len(shims) != 1 || shims[0].Name != "greet" {
		t.Errorf("after uninstall: %+v", shims)
	}
}

func TestInstallPath_RefusesBrokenBin(t *testing.T) {
	pkgDir := filepath.Join(t.TempDir(), "greeter")
	writeGreeterPackage(t, pkgDir)
	manifestPath := filepath.Join(pkgDir, pkg.ManifestFile)
	data, _ := os.ReadFile(manifestPath)
	broken := strings.Replace(string(data), `entry = "echoArgs"`, `entry = "missing"`, 1)
	if err := os.WriteFile(manifestPath, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest, err := pkg.LoadManifest(pkgDir)
	if err != nil {
		t.Fatal(err)
	}
	binDir := filepath.Join(t.TempDir(), "bin")
	var out bytes.Buffer
	err = installBinsTo(&out, pkgDir, binDir, "/nonexistent/ailang", manifest)
	if err == nil || !strings.Contains(err.Error(), "does not export func missing") {
		t.Fatalf("want refusal naming the missing entry, got %v", err)
	}
	if entries, _ := os.ReadDir(binDir); len(entries) != 0 {
		t.Errorf("no shim may be written when any bin is broken: %v", entries)
	}
}

// TestInstallRegistry_WritesLockAndShims drives the real `ailang install
// vendor/name` path against a fake registry: the tarball lands in the
// per-user cache, the lock is written beside it, and the shims point there.
func TestInstallRegistry_WritesLockAndShims(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("HOME-relative cache; covered on POSIX runners")
	}
	src := filepath.Join(t.TempDir(), "src")
	writeGreeterPackage(t, src)
	tarball, err := pkg.CreateTarball(src)
	if err != nil {
		t.Fatal(err)
	}
	meta := pkg.PackageMetadata{
		Schema:      "ailang.package-metadata/v1",
		Name:        "test/greeter",
		Version:     "0.1.0",
		TarballHash: pkg.TarballHash(tarball),
	}
	metaJSON, _ := json.Marshal(meta)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/packages/test/greeter/0.1.0/metadata.json":
			w.Write(metaJSON)
		case "/packages/test/greeter/0.1.0/package.tar.gz":
			w.Write(tarball)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv("AILANG_REGISTRY", server.URL)
	binDir := filepath.Join(home, "shims")

	// Run from a directory with no ailang.toml so install does not append a
	// dependency anywhere.
	cwd, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	if err := pkgInstallCommand([]string{"--bin-dir", binDir, "test/greeter@0.1.0"}); err != nil {
		t.Fatalf("install: %v", err)
	}

	cacheDir := filepath.Join(home, ".ailang", "cache", "registry", "test", "greeter", "0.1.0")
	if _, err := os.Stat(filepath.Join(cacheDir, pkg.LockFileName)); err != nil {
		t.Errorf("lock must be written beside the cached manifest: %v", err)
	}
	shims, err := pkg.ListShims(binDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(shims) != 2 || shims[0].Name != "greet" || shims[0].PkgDir != cacheDir || shims[0].Version != "0.1.0" {
		t.Errorf("shims: %+v", shims)
	}

	// --no-bin is the library-only install: cache populated, no shims.
	noBin := filepath.Join(home, "nobin")
	if err := pkgInstallCommand([]string{"--no-bin", "--bin-dir", noBin, "test/greeter@0.1.0"}); err != nil {
		t.Fatalf("install --no-bin: %v", err)
	}
	if _, err := os.Stat(noBin); !os.IsNotExist(err) {
		t.Errorf("--no-bin must not create a bin dir: %v", err)
	}
}

// TestBinCommand_ListAndUninstall drives `ailang bin` end to end through the
// command function: `install --path` writes the shims, `bin list` reads them
// back (flags accepted after the subcommand), `bin uninstall` removes exactly
// one and refuses a name that is not installed.
func TestBinCommand_ListAndUninstall(t *testing.T) {
	pkgDir := filepath.Join(t.TempDir(), "greeter")
	writeGreeterPackage(t, pkgDir)
	binDir := filepath.Join(t.TempDir(), "bin")

	out := captureStdout(t, func() {
		if err := pkgInstallCommand([]string{"--path", pkgDir, "--bin-dir", binDir}); err != nil {
			t.Errorf("install --path: %v", err)
		}
	})
	if !strings.Contains(out, "bin: greet →") || !strings.Contains(out, "bin: greet-args →") {
		t.Fatalf("install --path output:\n%s", out)
	}
	if err := pkgInstallCommand([]string{"--path", pkgDir, "some/name"}); err == nil {
		t.Error("--path with a vendor/name argument must be refused")
	}
	if err := pkgInstallCommand([]string{"--path", t.TempDir()}); err == nil {
		t.Error("--path on a directory without ailang.toml must be refused")
	}
	noBin := filepath.Join(t.TempDir(), "lib")
	os.MkdirAll(noBin, 0o755)
	os.WriteFile(filepath.Join(noBin, pkg.ManifestFile), []byte("[package]\nname = \"test/lib\"\nversion = \"0.1.0\"\nedition = \"1\"\n"), 0o644)
	if err := pkgInstallCommand([]string{"--path", noBin}); err == nil || !strings.Contains(err.Error(), "no [bin]") {
		t.Errorf("--path on a package without [bin] must say so, got %v", err)
	}

	out = captureStdout(t, func() {
		if err := binCommand([]string{"list", "--bin-dir", binDir}); err != nil {
			t.Errorf("bin list: %v", err)
		}
	})
	for _, want := range []string{"greet ", "greet-args", "test/greeter@0.1.0", "not on your PATH"} {
		if !strings.Contains(out, want) {
			t.Errorf("bin list missing %q:\n%s", want, out)
		}
	}

	out = captureStdout(t, func() {
		if err := binCommand([]string{"uninstall", "greet-args", "--bin-dir", binDir}); err != nil {
			t.Errorf("bin uninstall: %v", err)
		}
	})
	if !strings.Contains(out, "Removed") {
		t.Errorf("uninstall output:\n%s", out)
	}
	if err := binCommand([]string{"--bin-dir", binDir, "uninstall", "greet-args"}); err == nil {
		t.Error("uninstalling twice must fail")
	}
	if err := binCommand([]string{"--bin-dir", binDir, "uninstall"}); err == nil {
		t.Error("uninstall without a name must fail")
	}
	if err := binCommand([]string{"--bin-dir", binDir, "frobnicate"}); err == nil {
		t.Error("unknown subcommand must fail")
	}
	shims, _ := pkg.ListShims(binDir)
	if len(shims) != 1 || shims[0].Name != "greet" {
		t.Errorf("after uninstall: %+v", shims)
	}

	out = captureStdout(t, func() {
		if err := binCommand([]string{"list", "--bin-dir", filepath.Join(binDir, "absent")}); err != nil {
			t.Errorf("bin list on an empty dir: %v", err)
		}
	})
	if !strings.Contains(out, "No ailang bins installed") {
		t.Errorf("empty list output:\n%s", out)
	}
	out = captureStdout(t, func() { _ = binCommand([]string{"--help"}) })
	if !strings.Contains(out, "Usage: ailang bin list") {
		t.Errorf("help output:\n%s", out)
	}
}

// TestReportPathStatus covers the three PATH outcomes: absent, the shim
// itself, and shadowed by an earlier entry.
func TestReportPathStatus(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("PATH lookup semantics differ on Windows")
	}
	binDir := t.TempDir()
	shim := filepath.Join(binDir, "greet")
	os.WriteFile(shim, []byte("#!/bin/sh\n"), 0o755)

	var out bytes.Buffer
	t.Setenv("PATH", t.TempDir())
	if reportPathStatus(&out, "greet", shim) {
		t.Error("absent from PATH must report false")
	}

	t.Setenv("PATH", binDir)
	out.Reset()
	if !reportPathStatus(&out, "greet", shim) || out.Len() != 0 {
		t.Errorf("the shim itself must be found silently, got %q", out.String())
	}

	earlier := t.TempDir()
	os.WriteFile(filepath.Join(earlier, "greet"), []byte("#!/bin/sh\n"), 0o755)
	t.Setenv("PATH", earlier+string(os.PathListSeparator)+binDir)
	out.Reset()
	shadow, _ := filepath.EvalSymlinks(filepath.Join(earlier, "greet")) // macOS: /var → /private/var
	if !reportPathStatus(&out, "greet", shim) || !strings.Contains(out.String(), "resolves to "+shadow) {
		t.Errorf("a shadowing entry must be named, got %q", out.String())
	}
}
