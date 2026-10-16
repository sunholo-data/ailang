package runner

import (
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/testutil/pkgassets"
)

func TestRunnerLockedAssetBytes(t *testing.T) {
	root, _ := pkgassets.New(t)
	filename := filepath.Join(root, "asset_fixture.ail")
	result, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, RelaxModules: true}, pipeline.Source{Filename: filename})
	if err != nil || len(result.Errors) > 0 {
		t.Fatalf("compile: %v %v", err, result.Errors)
	}
	opts := Options{Filename: filename, Entry: "main", ArgsJSON: "null", Caps: "FS,IO", Quiet: true}
	if code := runSingle(t.Context(), result, opts, nil, "FS,IO"); code != -1 {
		t.Fatalf("single read wrong bytes: %d", code)
	}
	if err := ExecuteBatchItem(t.Context(), result, "input", opts, nil, nil); err != nil {
		t.Fatalf("batch read wrong bytes: %v", err)
	}
}
