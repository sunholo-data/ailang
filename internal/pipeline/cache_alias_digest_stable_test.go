package pipeline

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

// A record alias must digest identically every time. Map iteration order is
// randomized per range, so an unsorted record print gave every module whose
// closure holds a record alias (std/process.ProcessOutput, most packages) a new
// cache key per run: `ailang run` recompiled every time (Daneel, 2026-09-28).
func TestAliasDigest_RecordAliasIsStable(t *testing.T) {
	fields := map[string]types.Type{}
	for _, n := range []string{"stdout", "stderr", "exitCode", "truncated", "resolvedPath", "a", "b", "c"} {
		fields[n] = &types.TCon{Name: "int"}
	}
	ifc := &iface.Iface{
		Module: "m",
		TypeAliases: map[string]types.Type{
			"Closed": &types.TRecord{Fields: fields},
			"Open":   &types.TRecordOpen{Fields: fields, Row: &types.TVar2{Name: "r"}},
		},
	}
	want := aliasDigest(ifc)
	for i := 0; i < 200; i++ {
		if got := aliasDigest(ifc); got != want {
			t.Fatalf("alias digest changed on call %d: %s vs %s", i, got, want)
		}
	}
}
