package runtime

import (
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// loadViaPipeline compiles modulePath (relative to the repo root, without
// ".ail") through the pipeline and preloads every resulting module before
// LoadAndEvaluate — the same sequence the production entry path uses
// (internal/runner/run.go + entrypoint.go). Use it for any fixture that
// imports constructors or relies on OpLowering / type-class dictionaries;
// raw LoadAndEvaluate only elaborates trivial modules itself (#324).
func loadViaPipeline(t *testing.T, modulePath string) (*ModuleRuntime, *ModuleInstance) {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("abs repo root: %v", err)
	}
	// The pipeline derives canonical module paths from the working directory,
	// as `ailang run` from the repo root does.
	t.Chdir(root)
	res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck},
		pipeline.Source{Filename: modulePath + ".ail"})
	if err != nil {
		t.Fatalf("pipeline %s: %v", modulePath, err)
	}
	if len(res.Errors) > 0 {
		t.Fatalf("pipeline errors %s: %v", modulePath, res.Errors)
	}

	rt := NewModuleRuntime(root)
	if res.DictReg != nil {
		rt.GetEvaluator().SetDictionaryRegistry(res.DictReg)
	}
	for path, loaded := range res.Modules {
		rt.PreloadModule(path, loaded)
	}
	inst, err := rt.LoadAndEvaluate(res.Interface.Module)
	if err != nil {
		t.Fatalf("LoadAndEvaluate %s: %v", modulePath, err)
	}
	return rt, inst
}
