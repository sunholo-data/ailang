package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTryLoadPackageResolver_BrokenManifestSurfacesParseError(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "ailang.toml")
	manifest := `[package]
name = "sunholo/broken"
version = "0.1.0"
edition = "1"

[dependencies]
"sunholo/duplicate" = "0.1.0"
"sunholo/duplicate" = "0.2.0"
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, err := tryLoadPackageResolver(dir)
	if err == nil {
		t.Fatal("broken manifest unexpectedly loaded")
	}
	message := err.Error()
	if !strings.Contains(message, manifestPath) {
		t.Fatalf("error does not name manifest path %q: %s", manifestPath, message)
	}
	if !strings.Contains(message, "already been defined") || !strings.Contains(message, "sunholo/duplicate") {
		t.Fatalf("error does not carry duplicate-key TOML detail: %s", message)
	}
}

func TestTryLoadPackageResolver_NoManifestUsesLegacyResolution(t *testing.T) {
	dir := t.TempDir()
	resolver, err := tryLoadPackageResolver(dir)
	if err != nil || resolver != nil {
		t.Fatalf("no manifest: resolver = %T, error = %v; want nil, nil", resolver, err)
	}

	sourcePath := filepath.Join(dir, "main.ail")
	source := "module main\n\npure func answer() -> int = 42\n"
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if _, err := Run(Config{Mode: ModeCheck, PackageDir: dir}, Source{Filename: sourcePath}); err != nil {
		t.Fatalf("legacy bare-project pipeline failed: %v", err)
	}
}

// The lock-file load is the widest-blast-radius half of this change: it used to
// degrade to nil for ANY error, and now only a MISSING lock does. Both directions
// need an arm, because getting the split wrong breaks either every package
// project without a lock (authoring) or every project with a corrupt one
// (silently, which is the bug being fixed).

const validPackageManifest = `[package]
name = "sunholo/lockarms"
version = "0.1.0"
edition = "1"
`

// A missing ailang.lock stays OPTIONAL — a package being authored before its
// first `ailang lock` must still compile.
//
// Killing mutation: drop the errors.Is(err, os.ErrNotExist) branch so any lock
// error is fatal.
func TestTryLoadPackageResolver_MissingLockIsOptional(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte(validPackageManifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	// Control: the manifest itself is loadable, so a failure below is about the
	// absent lock and not about the fixture.
	if _, err := os.Stat(filepath.Join(dir, "ailang.toml")); err != nil {
		t.Fatalf("instrument failure: fixture manifest missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "ailang.lock")); !os.IsNotExist(err) {
		t.Fatalf("instrument failure: fixture must have no lock file, got %v", err)
	}

	resolver, err := tryLoadPackageResolver(dir)
	if err != nil {
		t.Fatalf("a missing lock file must not be an error: %v", err)
	}
	if resolver != nil {
		t.Errorf("resolver = %v, want nil so the self-only path takes over", resolver)
	}
}

// An ailang.lock that EXISTS and is malformed is loud, naming its path and the
// underlying parse error — the same Critical-Principle-2 correction the manifest
// half makes. Asserting on the message text, not on err != nil: every failure
// path in this function produces a non-nil error, so `err != nil` is satisfied
// by the missing-lock case too and would pass for the wrong reason.
//
// Killing mutation: return (nil, nil) from the malformed-lock branch.
func TestTryLoadPackageResolver_MalformedLockSurfacesParseError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte(validPackageManifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	lockPath := filepath.Join(dir, "ailang.lock")
	if err := os.WriteFile(lockPath, []byte("{ this is not json"), 0o644); err != nil {
		t.Fatalf("write lock: %v", err)
	}

	resolver, err := tryLoadPackageResolver(dir)
	if err == nil {
		t.Fatalf("malformed lock file was swallowed; resolver = %v", resolver)
	}
	message := err.Error()
	if !strings.Contains(message, lockPath) {
		t.Errorf("error does not name the lock path %q: %s", lockPath, message)
	}
	if !strings.Contains(message, "parse") {
		t.Errorf("error does not carry the underlying parse detail: %s", message)
	}
}
