package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/pkg"
)

func TestPkgLockPreservesMetadata(t *testing.T) {
	for _, withDep := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty", true: "path"}[withDep], func(t *testing.T) {
			root := lockPreserveFixture(t, withDep)
			t.Chdir(root)
			if err := pkgLockCommand(nil); err != nil {
				t.Fatal(err)
			}
			old, err := pkg.LoadLockFile(root)
			if err != nil {
				t.Fatal(err)
			}
			old.GeneratedAt = time.Date(2001, 2, 3, 4, 5, 6, 0, time.UTC)
			old.Generator = "ailang lock old-version"
			old.AILANGVersion = "old-version"
			if err := old.Save(root); err != nil {
				t.Fatal(err)
			}
			before := readPreserveLock(t, root)
			for i := 0; i < 2; i++ {
				if err := pkgLockCommand(nil); err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(before, readPreserveLock(t, root)) {
					t.Fatal("unchanged resolution rewrote lock metadata")
				}
			}
			if err := pkgLockCommand([]string{"--check"}); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, readPreserveLock(t, root)) {
				t.Fatal("--check wrote lock")
			}
			if withDep {
				depManifest := filepath.Join(root, "dep", pkg.ManifestFile)
				data, err := os.ReadFile(depManifest)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(depManifest, []byte(strings.ReplaceAll(string(data), "0.1.0", "0.2.0")), 0644); err != nil {
					t.Fatal(err)
				}
				if err := pkgLockCommand([]string{"--check"}); err == nil {
					t.Fatal("--check accepted drift")
				}
				if !bytes.Equal(before, readPreserveLock(t, root)) {
					t.Fatal("drift check wrote lock")
				}
				if err := pkgLockCommand(nil); err != nil {
					t.Fatal(err)
				}
				fresh, err := pkg.LoadLockFile(root)
				if err != nil {
					t.Fatal(err)
				}
				if fresh.GeneratedAt.Equal(old.GeneratedAt) || fresh.Generator == old.Generator || fresh.AILANGVersion != Version {
					t.Fatal("changed resolution retained stale metadata")
				}
				if fresh.Packages[0].Path != "dep" {
					t.Fatalf("relative path lost: %q", fresh.Packages[0].Path)
				}
			}
		})
	}
}

func TestPkgLockInvalidPreviousLock(t *testing.T) {
	for _, kind := range []string{"malformed", "invalid-schema", "unreadable"} {
		t.Run(kind, func(t *testing.T) {
			root := lockPreserveFixture(t, false)
			t.Chdir(root)
			path := filepath.Join(root, pkg.LockFileName)
			if kind == "unreadable" {
				// A directory is an unreadable lock on every platform, including root CI.
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
			} else {
				data := []byte("{")
				if kind == "invalid-schema" {
					data = []byte(`{"schema":"unknown"}`)
				}
				if err := os.WriteFile(path, data, 0644); err != nil {
					t.Fatal(err)
				}
			}
			err := pkgLockCommand(nil)
			if err == nil || !strings.Contains(err.Error(), "cannot update existing "+pkg.LockFileName) {
				t.Fatalf("expected actionable existing lock error, got %v", err)
			}
			if kind != "unreadable" {
				want := "{"
				if kind == "invalid-schema" {
					want = `{"schema":"unknown"}`
				}
				if string(readPreserveLock(t, root)) != want {
					t.Fatal("invalid lock overwritten")
				}
			}
		})
	}
}

func lockPreserveFixture(t *testing.T, withDep bool) string {
	t.Helper()
	root := t.TempDir()
	manifest := "[package]\nname = \"test/app\"\nversion = \"0.1.0\"\nedition = \"1\"\n"
	if withDep {
		dep := filepath.Join(root, "dep")
		if err := os.Mkdir(dep, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dep, pkg.ManifestFile), []byte("[package]\nname = \"test/dep\"\nversion = \"0.1.0\"\nedition = \"1\"\n"), 0644); err != nil {
			t.Fatal(err)
		}
		manifest += "[dependencies]\n\"test/dep\" = { path = \"dep\" }\n"
	}
	if err := os.WriteFile(filepath.Join(root, pkg.ManifestFile), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

func readPreserveLock(t *testing.T, root string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, pkg.LockFileName))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
