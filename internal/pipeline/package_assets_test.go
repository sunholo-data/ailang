package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/testutil/pkgassets"
)

func TestAssetTargetsForDiscovery(t *testing.T) {
	dir := t.TempDir()
	targets, err := AssetTargetsFor(dir)
	if err != nil || targets != nil {
		t.Fatalf("bare: %v %v", targets, err)
	}
	manifest := filepath.Join(dir, pkg.ManifestFile)
	if err := os.WriteFile(manifest, []byte(validPackageManifest), 0600); err != nil {
		t.Fatal(err)
	}
	targets, err = AssetTargetsFor(dir)
	if err != nil || len(targets) != 1 {
		t.Fatalf("no lock should retain root assets: %v %v", targets, err)
	}
	for _, target := range targets {
		if target.Dir != dir {
			t.Fatalf("root target directory = %q, want %q", target.Dir, dir)
		}
	}
	lf := pkg.NewLockFile([]pkg.LockedPackage{{Name: "test/transitive", Version: "0.3.0", Source: "path", Path: "missing", ContentHash: "test"}}, "test")
	if err := lf.Save(dir); err != nil {
		t.Fatal(err)
	}
	targets, err = AssetTargetsFor(dir)
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := targets["test/transitive"]; !ok || v.Version != "0.3.0" || v.Dir != "" {
		t.Fatalf("lost unresolved target: %v", targets)
	}
	if err := os.WriteFile(filepath.Join(dir, pkg.LockFileName), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = AssetTargetsFor(dir); err == nil || !strings.Contains(err.Error(), pkg.LockFileName) {
		t.Fatalf("malformed lock: %v", err)
	}
	if err := os.WriteFile(manifest, []byte("[package"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = AssetTargetsFor(dir); err == nil || !strings.Contains(err.Error(), pkg.ManifestFile) {
		t.Fatalf("malformed manifest: %v", err)
	}
}

func TestPipelineLockedAssetTargets(t *testing.T) {
	root, locked := pkgassets.New(t)
	result, err := Run(Config{Mode: ModeCheck, RelaxModules: true}, Source{Filename: filepath.Join(root, "asset_fixture.ail")})
	if err != nil || len(result.Errors) > 0 {
		t.Fatalf("compile: %v %v", err, result.Errors)
	}
	target, ok := result.PkgAssets["test/data"]
	if !ok || target.Dir != locked || target.Version != "0.3.0" {
		t.Fatalf("lost locked target: %v", result.PkgAssets)
	}
}
