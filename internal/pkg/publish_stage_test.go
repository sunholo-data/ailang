package pkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

func smokeStageFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS("testdata/smoke_staged_lock")); err != nil {
		t.Fatal(err)
	}
	testutil.SetHomeDir(t, t.TempDir())
	t.Setenv("AILANG_PACKAGE_ROOT", filepath.Join(root, "registry"))
	t.Setenv("AILANG_REGISTRY", "http://127.0.0.1:1")
	t.Setenv("AILANG_NO_CACHE", "1")
	std, err := filepath.Abs("../../std")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", std)
	author := filepath.Join(root, "a")
	m, err := LoadManifest(author)
	if err != nil {
		t.Fatal(err)
	}
	lf, err := ResolveLock(m, author, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if err := lf.Save(author); err != nil {
		t.Fatal(err)
	}
	return root, author
}

func TestSmokeStage_PublishedSourceAndAssetBytes(t *testing.T) {
	_, author := smokeStageFixture(t)
	beforeManifest, _ := os.ReadFile(filepath.Join(author, ManifestFile))
	beforeLock, _ := os.ReadFile(filepath.Join(author, LockFileName))
	result, err := RunSmokeInTempDir(author, findAilangBinary(t), 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Passed {
		t.Fatalf("published stage failed: exit=%d output=%s", result.ExitCode, result.Output)
	}
	for _, marker := range []string{"REGISTRY_SOURCE", "REGISTRY_SOURCE_ASSET", "STAGED_ROOT"} {
		if !strings.Contains(result.Output, marker) {
			t.Errorf("missing %q: %s", marker, result.Output)
		}
	}
	if strings.Contains(result.Output, "LOCAL_SOURCE") {
		t.Fatal("smoke read author dependency instead of published bytes")
	}
	afterManifest, _ := os.ReadFile(filepath.Join(author, ManifestFile))
	afterLock, _ := os.ReadFile(filepath.Join(author, LockFileName))
	if string(afterManifest) != string(beforeManifest) || string(afterLock) != string(beforeLock) {
		t.Fatal("staging mutated author manifest/lock")
	}
}

func TestSmokeStage_RegeneratesRegistryLock(t *testing.T) {
	root, author := smokeStageFixture(t)
	stage := t.TempDir()
	if err := stagePublishedPackage(author, stage); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(stage)
	if err != nil {
		t.Fatal(err)
	}
	if m.Dependencies["test/b"].Path != "" || m.Dependencies["test/b"].Version != "0.1.0" {
		t.Fatalf("not published manifest: %+v", m.Dependencies)
	}
	lf, err := LoadLockFile(stage)
	if err != nil {
		t.Fatal(err)
	}
	b, ok := lf.FindPackage("test/b")
	if !ok || b.Source != "registry" || b.Path != "" {
		t.Fatalf("copied path lock survived: %+v", b)
	}
	published := filepath.Join(root, "registry", "test", "b", "0.1.0")
	expected, err := ContentHash(published)
	if err != nil {
		t.Fatal(err)
	}
	local, err := ContentHash(filepath.Join(root, "b"))
	if err != nil {
		t.Fatal(err)
	}
	pm, err := LoadManifest(published)
	if err != nil {
		t.Fatal(err)
	}
	if b.ContentHash != expected || b.ContentHash == local || b.InterfaceHash != InterfaceHash(pm) {
		t.Fatalf("not published hashes: %+v", b)
	}
	loader := NewPackageLoader(lf, stage)
	self, _, ok := loader.AssetTarget("test/a")
	if !ok || self != stage {
		t.Fatalf("self target not staged: %q", self)
	}
	dep, _, ok := loader.AssetTarget("test/b")
	if !ok || dep != published {
		t.Fatalf("dependency target: %q", dep)
	}
}

func TestSmokeStage_FailClosedPreparation(t *testing.T) {
	for _, kind := range []string{"malformed_lock", "invalid_schema", "missing_exact_registry"} {
		t.Run(kind, func(t *testing.T) {
			root, author := smokeStageFixture(t)
			switch kind {
			case "malformed_lock":
				if err := os.WriteFile(filepath.Join(author, LockFileName), []byte("{"), 0600); err != nil {
					t.Fatal(err)
				}
			case "invalid_schema":
				if err := os.WriteFile(filepath.Join(author, LockFileName), []byte(`{"schema":"bad","schema_version":"1.0.0","packages":[]}`), 0600); err != nil {
					t.Fatal(err)
				}
			case "missing_exact_registry":
				old := filepath.Join(root, "registry", "test", "b", "0.1.0")
				newer := filepath.Join(root, "registry", "test", "b", "0.2.0")
				if err := os.Rename(old, newer); err != nil {
					t.Fatal(err)
				}
			}
			stage := t.TempDir()
			err := stagePublishedPackage(author, stage)
			if err == nil {
				t.Fatal("invalid preparation accepted")
			}
			want := LockFileName
			if kind == "missing_exact_registry" {
				want = "test/b@0.1.0"
			}
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("expected named cause %q: %v", want, err)
			}
		})
	}
}
