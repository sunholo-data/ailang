package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

func TestPkgLockShadowWarning_RealCLI(t *testing.T) {
	bin := buildAilang(t)
	for _, shadow := range []bool{false, true} {
		name := "direct_path"
		if shadow {
			name = "transitive_registry_shadow"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			fixture, err := filepath.Abs("../../internal/pkg/testdata/smoke_staged_lock")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.CopyFS(root, os.DirFS(fixture)); err != nil {
				t.Fatal(err)
			}
			aManifest := filepath.Join(root, "a", pkg.ManifestFile)
			a, err := os.ReadFile(aManifest)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(aManifest, []byte(strings.Replace(string(a), `{path = "../b"}`, `"0.1.0"`, 1)), 0600); err != nil {
				t.Fatal(err)
			}
			deps := "\"test/b\" = {path = \"b\"}\n"
			if shadow {
				deps = "\"test/a\" = {path = \"a\"}\n" + deps
			}
			if err := os.WriteFile(filepath.Join(root, pkg.ManifestFile), []byte("[package]\nname=\"test/consumer\"\nversion=\"0.1.0\"\nedition=\"1\"\n[dependencies]\n"+deps), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("AILANG_PACKAGE_ROOT", filepath.Join(root, "registry"))
			t.Setenv("AILANG_REGISTRY", "http://127.0.0.1:1")
			manifest, err := pkg.LoadManifest(root)
			if err != nil {
				t.Fatal(err)
			}
			expected, err := pkg.ResolveLock(manifest, root, "without warning")
			if err != nil {
				t.Fatal(err)
			}
			home := t.TempDir()
			result := runCLIIn(t, bin, home, root, "lock")
			if result.exitCode != 0 {
				t.Fatalf("lock failed: %+v", result)
			}
			got, err := pkg.LoadLockFile(root)
			if err != nil {
				t.Fatal(err)
			}
			if drift := pkg.LockDrift(got, expected); len(drift) != 0 {
				t.Fatalf("warning changed resolution: %v", drift)
			}
			warned := strings.Contains(result.stderr, "test/b") && strings.Contains(result.stderr, "registry") && strings.Contains(result.stderr, "NOT in this lock")
			if warned != shadow {
				t.Fatalf("shadow=%v wrong stderr: %s", shadow, result.stderr)
			}
			if !shadow {
				return
			}
			before, err := os.ReadFile(filepath.Join(root, pkg.LockFileName))
			if err != nil {
				t.Fatal(err)
			}
			bManifest := filepath.Join(root, "b", pkg.ManifestFile)
			data, err := os.ReadFile(bManifest)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(bManifest, []byte(strings.Replace(string(data), `version = "0.1.0"`, `version = "0.2.0"`, 1)), 0600); err != nil {
				t.Fatal(err)
			}
			conflict := runCLIIn(t, bin, home, root, "lock")
			if conflict.exitCode == 0 || !strings.Contains(conflict.stderr, "version conflict") || strings.Contains(conflict.stderr, "NOT in this lock") {
				t.Fatalf("conflict semantics changed: %+v", conflict)
			}
			after, _ := os.ReadFile(filepath.Join(root, pkg.LockFileName))
			if string(after) != string(before) {
				t.Fatal("conflict rewrote committed lock")
			}
		})
	}
}

func TestPkgLockShadowWarning_ActualSourcesAndReadOnly(t *testing.T) {
	manifest := &pkg.PackageManifest{Dependencies: map[string]pkg.Dependency{
		"test/registry": {Path: "registry-local"}, "test/git": {Path: "git-local"},
		"test/path": {Path: "selected"}, "test/direct": {Version: "0.1.0"},
		"test/absent": {Path: "absent"},
	}}
	resolved := []pkg.ResolvedPackage{
		{Name: "test/registry", Source: "registry", Version: "0.1.0"},
		{Name: "test/git", Source: "git", Version: "0.1.0"},
		{Name: "test/path", Source: "path", Version: "0.1.0"},
		{Name: "test/direct", Source: "registry", Version: "0.1.0"},
	}
	beforeManifest, _ := json.Marshal(manifest)
	beforeResolved, _ := json.Marshal(resolved)
	var out bytes.Buffer
	warnShadowedPathDeps(&out, manifest, resolved)
	text := out.String()
	for _, want := range []string{"dependency test/git is declared as a path dep but was resolved from git", "dependency test/registry is declared as a path dep but was resolved from registry"} {
		if !strings.Contains(text, want) {
			t.Fatalf("actual source missing: %s", text)
		}
	}
	for _, unwanted := range []string{"dependency test/path", "dependency test/direct", "dependency test/absent"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("false warning: %s", text)
		}
	}
	afterManifest, _ := json.Marshal(manifest)
	afterResolved, _ := json.Marshal(resolved)
	if !bytes.Equal(beforeManifest, afterManifest) || !bytes.Equal(beforeResolved, afterResolved) {
		t.Fatal("warning mutated resolver inputs")
	}
}
