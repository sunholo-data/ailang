package pkg

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// M-PKG-BIN-ENTRYPOINTS tests: the [bin] table, entry verification, the
// shim round-trip and the lock a bin install writes.

func writeBinFixture(t *testing.T, manifestExtra string) string {
	t.Helper()
	dir := t.TempDir()
	manifest := `
[package]
name = "test/greeter"
version = "0.1.0"
edition = "1"

[exports]
modules = ["test/greeter/lib"]
` + manifestExtra
	if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	cli := `module test/greeter/cli
import std/io (println)

export func main() -> () ! {IO} = println("hi")

export pure func version() -> string = "0.1.0"
`
	if err := os.WriteFile(filepath.Join(dir, "cli.ail"), []byte(cli), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadManifest_BinForms(t *testing.T) {
	dir := writeBinFixture(t, `
[bin]
greet = "cli"
greet-v = { module = "test/greeter/cli", entry = "version", caps = "IO", run_flags = ["--max-recursion-depth", "50000"] }
`)
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	short := m.Bin["greet"]
	if short.Module != "cli" || short.EffectiveEntry() != "main" || short.EffectiveCaps() != "auto" {
		t.Errorf("string form: %+v (entry %q caps %q)", short, short.EffectiveEntry(), short.EffectiveCaps())
	}
	long := m.Bin["greet-v"]
	if long.Module != "test/greeter/cli" || long.Entry != "version" || long.Caps != "IO" || len(long.RunFlags) != 2 {
		t.Errorf("table form: %+v", long)
	}
}

func TestLoadManifest_BinRejects(t *testing.T) {
	cases := map[string]string{
		"uppercase name":    "[bin]\nGreet = \"cli\"\n",
		"leading dash":      "[bin]\n\"-x\" = \"cli\"\n",
		"space in name":     "[bin]\n\"a b\" = \"cli\"\n",
		"reserved ailang":   "[bin]\nailang = \"cli\"\n",
		"empty module":      "[bin]\ngreet = { entry = \"main\" }\n",
		"run_flags --caps":  "[bin]\ngreet = { module = \"cli\", run_flags = [\"--caps\", \"IO\"] }\n",
		"run_flags --entry": "[bin]\ngreet = { module = \"cli\", run_flags = [\"--entry=main\"] }\n",
		"run_flags quiet":   "[bin]\ngreet = { module = \"cli\", run_flags = [\"-quiet\"] }\n",
		"run_flags pkgdir":  "[bin]\ngreet = { module = \"cli\", run_flags = [\"--package-dir\", \"/x\"] }\n",
	}
	for name, extra := range cases {
		t.Run(name, func(t *testing.T) {
			dir := writeBinFixture(t, extra)
			if _, err := LoadManifest(dir); err == nil {
				t.Fatalf("expected %s to be rejected", name)
			} else if !strings.Contains(err.Error(), "[bin]") {
				t.Fatalf("error should name [bin]: %v", err)
			}
		})
	}
}

func TestVerifyBinEntrypoints(t *testing.T) {
	t.Run("ok, string and table, pure entry", func(t *testing.T) {
		dir := writeBinFixture(t, "[bin]\ngreet = \"cli\"\nv = { module = \"cli\", entry = \"version\" }\n")
		m, _ := LoadManifest(dir)
		if err := VerifyBinEntrypoints(dir, m); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("missing module", func(t *testing.T) {
		dir := writeBinFixture(t, "[bin]\ngreet = \"nowhere\"\n")
		m, _ := LoadManifest(dir)
		err := VerifyBinEntrypoints(dir, m)
		if err == nil || !strings.Contains(err.Error(), `module "nowhere" does not resolve`) {
			t.Fatalf("want missing-module error, got %v", err)
		}
	})
	t.Run("missing entry", func(t *testing.T) {
		dir := writeBinFixture(t, "[bin]\ngreet = { module = \"cli\", entry = \"nope\" }\n")
		m, _ := LoadManifest(dir)
		err := VerifyBinEntrypoints(dir, m)
		if err == nil || !strings.Contains(err.Error(), "does not export func nope") {
			t.Fatalf("want missing-entry error, got %v", err)
		}
	})
	t.Run("module_prefix layout", func(t *testing.T) {
		dir := t.TempDir()
		manifest := "[package]\nname = \"sunholo/ailang_parse\"\nversion = \"1.0.0\"\nedition = \"1\"\nmodule_prefix = \"docparse\"\n[exports]\nmodules = [\"docparse/main\"]\n[bin]\ndocparse = \"main\"\ndocparse2 = \"docparse/main\"\n"
		os.WriteFile(filepath.Join(dir, ManifestFile), []byte(manifest), 0o644)
		os.MkdirAll(filepath.Join(dir, "docparse"), 0o755)
		os.WriteFile(filepath.Join(dir, "docparse", "main.ail"), []byte("module docparse/main\nexport func main() -> () ! {IO} = ()\n"), 0o644)
		m, err := LoadManifest(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := VerifyBinEntrypoints(dir, m); err != nil {
			t.Fatal(err)
		}
	})
}

func TestShim_RoundTrip(t *testing.T) {
	dir := writeBinFixture(t, "[bin]\ngreet = \"cli\"\nv = { module = \"cli\", entry = \"version\", caps = \"IO\", run_flags = [\"--seed\", \"1\"] }\n")
	m, _ := LoadManifest(dir)
	binDir := filepath.Join(t.TempDir(), "bin")
	ailang := filepath.Join(t.TempDir(), "ailang")

	path, err := WriteShim(binDir, ailang, dir, m, "greet", m.Bin["greet"])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteShim(binDir, ailang, dir, m, "v", m.Bin["v"]); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(path)
	if runtime.GOOS != "windows" {
		if !strings.HasPrefix(string(body), "#!/bin/sh\n") {
			t.Errorf("shim must start with a shebang:\n%s", body)
		}
		for _, want := range []string{"--quiet", "--package-dir " + dir, "--entry main", "--caps auto", filepath.Join(dir, "cli.ail") + " -- \"$@\""} {
			if !strings.Contains(string(body), want) {
				t.Errorf("shim missing %q:\n%s", want, body)
			}
		}
		info, _ := os.Stat(path)
		if info.Mode()&0o111 == 0 {
			t.Errorf("shim is not executable: %v", info.Mode())
		}
		vBody, _ := os.ReadFile(ShimPath(binDir, "v"))
		if !strings.Contains(string(vBody), "--caps IO --seed 1 ") {
			t.Errorf("run_flags must follow the shim's own flags and precede the file:\n%s", vBody)
		}
	}

	s, err := ReadShim(path)
	if err != nil || s == nil {
		t.Fatalf("ReadShim: %v %v", s, err)
	}
	if s.Name != "greet" || s.Package != "test/greeter" || s.Version != "0.1.0" || s.Ailang != ailang || s.PkgDir != dir {
		t.Errorf("parsed shim: %+v", s)
	}

	// A user's own script in the same directory is not a shim.
	os.WriteFile(filepath.Join(binDir, "mine"), []byte("#!/bin/sh\necho mine\n"), 0o755)
	shims, err := ListShims(binDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(shims) != 2 || shims[0].Name != "greet" || shims[1].Name != "v" {
		t.Errorf("ListShims: %+v", shims)
	}
	if _, err := RemoveShim(binDir, "mine"); err == nil {
		t.Error("RemoveShim must refuse a file that is not an ailang shim")
	}
	if _, err := RemoveShim(binDir, "greet"); err != nil {
		t.Fatal(err)
	}
	if _, err := RemoveShim(binDir, "greet"); err == nil {
		t.Error("second RemoveShim must report the bin is not installed")
	}
	shims, _ = ListShims(binDir)
	if len(shims) != 1 {
		t.Errorf("after remove: %+v", shims)
	}
	if got, _ := ListShims(filepath.Join(binDir, "absent")); got != nil {
		t.Errorf("missing dir must list as empty, got %+v", got)
	}
}

func TestEnsureLock(t *testing.T) {
	dir := writeBinFixture(t, "[bin]\ngreet = \"cli\"\n")
	n, wrote, err := EnsureLock(dir, "test", "v0.0.0")
	if err != nil || !wrote || n != 0 {
		t.Fatalf("first EnsureLock: n=%d wrote=%v err=%v", n, wrote, err)
	}
	if _, err := os.Stat(filepath.Join(dir, LockFileName)); err != nil {
		t.Fatalf("lock not written: %v", err)
	}
	n, wrote, err = EnsureLock(dir, "test", "v0.0.0")
	if err != nil || wrote {
		t.Fatalf("second EnsureLock must keep the existing lock: n=%d wrote=%v err=%v", n, wrote, err)
	}
	os.WriteFile(filepath.Join(dir, LockFileName), []byte("{not json"), 0o644)
	if _, _, err := EnsureLock(dir, "test", "v0.0.0"); err == nil {
		t.Fatal("a corrupt lock must fail loudly, not be silently regenerated")
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"plain":     "plain",
		"--flag=1":  "--flag=1",
		"/a/b.ail":  "/a/b.ail",
		"two words": "'two words'",
		"it's":      `'it'\''s'`,
		"$HOME":     "'$HOME'",
		"":          "''",
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestShimBody_WindowsAndPosixParseAlike(t *testing.T) {
	dir := writeBinFixture(t, "[bin]\ngreet = { module = \"cli\", run_flags = [\"--seed\", \"1\"] }\n")
	m, _ := LoadManifest(dir)
	file := filepath.Join(dir, "cli.ail")

	win := shimBody("windows", `C:\tools\ailang.exe`, dir, file, m, "greet", m.Bin["greet"])
	if !strings.HasPrefix(win, "@echo off\r\n") || !strings.Contains(win, "rem ailang-bin: test/greeter@0.1.0 greet\r\n") || !strings.HasSuffix(win, " %*\r\n") {
		t.Errorf("windows shim:\n%s", win)
	}
	for _, want := range []string{`"--package-dir" "` + dir + `"`, `"--caps" "auto"`, `"--seed" "1" "` + file + `" "--"`} {
		if !strings.Contains(win, want) {
			t.Errorf("windows shim missing %s:\n%s", want, win)
		}
	}
	posix := shimBody("darwin", "/usr/local/bin/ailang", dir, file, m, "greet", m.Bin["greet"])
	if !strings.HasPrefix(posix, "#!/bin/sh\n") || !strings.HasSuffix(posix, ` "$@"`+"\n") {
		t.Errorf("posix shim:\n%s", posix)
	}

	// Both bodies parse back to the same Shim through the same reader.
	binDir := t.TempDir()
	for goos, body := range map[string]string{"windows": win, "linux": posix} {
		path := shimPath(goos, binDir, "greet")
		if goos == "windows" && !strings.HasSuffix(path, ".cmd") {
			t.Errorf("windows shim path must end in .cmd: %s", path)
		}
		os.WriteFile(path, []byte(body), 0o755)
		s, err := ReadShim(path)
		if err != nil || s == nil || s.Name != "greet" || s.Package != "test/greeter" || s.Version != "0.1.0" || s.PkgDir != dir {
			t.Errorf("%s shim did not parse back: %+v %v", goos, s, err)
		}
	}
}

func TestEnsureLock_ResolvesPathDependencies(t *testing.T) {
	root := t.TempDir()
	dep := filepath.Join(root, "dep")
	os.MkdirAll(dep, 0o755)
	os.WriteFile(filepath.Join(dep, ManifestFile), []byte("[package]\nname = \"test/dep\"\nversion = \"0.2.0\"\nedition = \"1\"\n[exports]\nmodules = [\"test/dep/core\"]\n"), 0o644)
	os.WriteFile(filepath.Join(dep, "core.ail"), []byte("module test/dep/core\nexport pure func one() -> int = 1\n"), 0o644)

	app := filepath.Join(root, "app")
	os.MkdirAll(app, 0o755)
	os.WriteFile(filepath.Join(app, ManifestFile), []byte("[package]\nname = \"test/app\"\nversion = \"0.1.0\"\nedition = \"1\"\n[dependencies]\n\"test/dep\" = { path = \"../dep\" }\n[bin]\napp = \"cli\"\n"), 0o644)
	os.WriteFile(filepath.Join(app, "cli.ail"), []byte("module test/app/cli\nexport func main() -> () ! {IO} = ()\n"), 0o644)

	n, wrote, err := EnsureLock(app, "test", "v0.0.0")
	if err != nil || !wrote || n != 1 {
		t.Fatalf("EnsureLock: n=%d wrote=%v err=%v", n, wrote, err)
	}
	lf, err := LoadLockFile(app)
	if err != nil {
		t.Fatal(err)
	}
	if lf.AILANGVersion != "v0.0.0" || lf.Generator != "test" || len(lf.Packages) != 1 || lf.Packages[0].Name != "test/dep" || lf.Packages[0].Source != "path" {
		t.Errorf("lock: %+v", lf)
	}

	// A manifest that cannot resolve fails loudly and writes nothing.
	os.WriteFile(filepath.Join(app, ManifestFile), []byte("[package]\nname = \"test/app\"\nversion = \"0.1.0\"\nedition = \"1\"\n[dependencies]\n\"test/missing\" = { path = \"../nowhere\" }\n"), 0o644)
	os.Remove(filepath.Join(app, LockFileName))
	if _, _, err := EnsureLock(app, "test", "v0.0.0"); err == nil {
		t.Fatal("an unresolvable dependency must fail EnsureLock")
	}
	if _, err := os.Stat(filepath.Join(app, LockFileName)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("no lock may be written on failure: %v", err)
	}
	if _, _, err := EnsureLock(filepath.Join(root, "absent"), "test", "v0.0.0"); err == nil {
		t.Fatal("a directory without a manifest must fail EnsureLock")
	}
}

func TestDefaultBinDir(t *testing.T) {
	dir, err := DefaultBinDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "bin" || filepath.Base(filepath.Dir(dir)) != ".ailang" {
		t.Errorf("DefaultBinDir = %s, want <home>/.ailang/bin", dir)
	}
}
