package runner

import (
	"path/filepath"
	"sort"
	"testing"

	"github.com/sunholo-data/ailang/internal/gen/lower"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/types"
)

// TestLoweredLocalsAreUnique checks the invariant the VM's flat name space
// needs (#2375) on real programs: every runnable example and every module it
// imports (the stdlib included), lowered as `run --bytecode` lowers them,
// declares each source-named local once per function
// (lower.CheckUniqueLocals).
func TestLoweredLocalsAreUnique(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	files, err := filepath.Glob("examples/runnable/*.ail")
	if err != nil || len(files) == 0 {
		t.Fatalf("no examples found (%v)", err)
	}
	checkedMods := map[string]bool{}
	funcs := 0
	for _, f := range files {
		res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck}, pipeline.Source{Filename: f})
		if err != nil || len(res.Errors) > 0 {
			continue // examples that need flags or packages; not this test's concern
		}
		funcs += checkLoweredUnique(t, f, res, checkedMods)
	}
	if funcs < 500 {
		t.Fatalf("checked only %d functions in %d modules: the corpus did not load", funcs, len(checkedMods))
	}
	t.Logf("%d functions in %d modules", funcs, len(checkedMods))
}

// checkLoweredUnique lowers every module of res not yet in done, as the VM
// does, and reports each function that breaks lower.CheckUniqueLocals. It
// returns the number of functions checked.
func checkLoweredUnique(t *testing.T, label string, res pipeline.Result, done map[string]bool) int {
	t.Helper()
	ids := make([]string, 0, len(res.Modules))
	for id := range res.Modules {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	funcs := 0
	for _, id := range ids {
		mod := res.Modules[id]
		if mod == nil || mod.Core == nil || done[id] {
			continue
		}
		done[id] = true
		cti, _ := mod.CoreTI.(types.CoreTypeInfo)
		prog, err := lower.LowerProgram(mod.Core, cti, mod.File, "p")
		if err != nil {
			t.Errorf("%s: lower %s: %v", label, id, err)
			continue
		}
		for _, fd := range prog.FuncDecls {
			funcs++
			if err := lower.CheckUniqueLocals(fd); err != nil {
				t.Errorf("%s (module %s): %v", label, id, err)
			}
		}
	}
	return funcs
}
