package runner

import (
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/testutil/pkgassets"
)

func TestRunnerLockedAssetBytes(t *testing.T) {
	root, _ := pkgassets.New(t)
	filename := filepath.Join(root, "asset_fixture.ail")
	result, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, RelaxModules: true}, pipeline.Source{Filename: filename})
	if err != nil || len(result.Errors) > 0 {
		t.Fatalf("compile: %v %v", err, result.Errors)
	}
	// Changing the on-disk lock after compilation must not change runtime assets.
	lf, err := pkg.LoadLockFile(root)
	if err != nil {
		t.Fatal(err)
	}
	lf.Packages[0].Version = "0.4.0"
	newer, err := pkg.PackageDir("test/data", "0.4.0")
	if err != nil {
		t.Fatal(err)
	}
	lf.Packages[0].ContentHash, err = pkg.ContentHash(newer)
	if err != nil {
		t.Fatal(err)
	}
	if err := lf.Save(root); err != nil {
		t.Fatal(err)
	}
	opts := Options{Filename: filename, Entry: "main", ArgsJSON: "null", Caps: "FS,IO", Quiet: true}
	if code := runSingle(t.Context(), result, opts, nil, "FS,IO"); code != -1 {
		t.Fatalf("single read wrong bytes: %d", code)
	}
	if err := ExecuteBatchItem(t.Context(), result, "input", opts, nil, nil); err != nil {
		t.Fatalf("batch read wrong bytes: %v", err)
	}
}
