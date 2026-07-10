package bestof

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// stubVerifier returns canned verdicts keyed by path — lets us unit-test the ranking
// logic without invoking the compiler.
type stubVerifier struct{ m map[string]Verdict }

func (s stubVerifier) Verify(p string) Verdict { return s.m[p] }

func TestSelectBest_Ranking(t *testing.T) {
	paths := []string{"neither", "typechecks", "runs"}
	v := stubVerifier{m: map[string]Verdict{
		"neither":    {},
		"typechecks": {TypeChecks: true},
		"runs":       {TypeChecks: true, Runs: true},
	}}
	best, verdicts := SelectBest(paths, v)
	if best != 2 {
		t.Errorf("best=%d, want 2 (the running candidate outranks typechecks-only and neither)", best)
	}
	if len(verdicts) != 3 {
		t.Errorf("verdicts len=%d, want 3", len(verdicts))
	}
}

func TestSelectBest_TieKeepsEarliest(t *testing.T) {
	// Two equally-good candidates → keep the model's own ordering (earliest).
	v := stubVerifier{m: map[string]Verdict{"a": {TypeChecks: true, Runs: true}, "b": {TypeChecks: true, Runs: true}}}
	if best, _ := SelectBest([]string{"a", "b"}, v); best != 0 {
		t.Errorf("best=%d, want 0 (earliest on tie)", best)
	}
}

func TestSelectBest_AllFailFallsBackToFirst(t *testing.T) {
	v := stubVerifier{m: map[string]Verdict{"a": {}, "b": {}}}
	if best, _ := SelectBest([]string{"a", "b"}, v); best != 0 {
		t.Errorf("best=%d, want 0 (submit a guess rather than nothing)", best)
	}
}

func TestSelectBest_Empty(t *testing.T) {
	if best, _ := SelectBest(nil, stubVerifier{}); best != -1 {
		t.Errorf("best=%d, want -1 for empty list", best)
	}
}

// TestAilangVerifier_Integration exercises the real selector against the installed
// compiler: a good candidate typechecks+runs, a Num[string] candidate fails check, and
// SelectBest picks the good one out of the pair. Skips if ailang isn't on PATH.
func TestAilangVerifier_Integration(t *testing.T) {
	bin := testutil.FindAilangBinary(t)
	dir := t.TempDir()
	good := filepath.Join(dir, "good.ail")
	bad := filepath.Join(dir, "bad.ail")
	if err := os.WriteFile(good, []byte("module good\nimport std/io (println)\nexport func main() -> () ! {IO} { println(\"hi\") }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bad, []byte("module bad\nimport std/io (println)\nexport func main() -> () ! {IO} { println(show(\"a\" + \"b\")) }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	v := AilangVerifier{Bin: bin}
	if gv := v.Verify(good); !gv.TypeChecks || !gv.Runs {
		t.Errorf("good candidate verdict=%+v, want TypeChecks && Runs", gv)
	}
	if bv := v.Verify(bad); bv.TypeChecks {
		t.Errorf("bad candidate verdict=%+v, want TypeChecks=false (Num[string])", bv)
	}
	if best, _ := SelectBest([]string{bad, good}, v); best != 1 {
		t.Errorf("best=%d, want 1 (the verified-good candidate)", best)
	}
}
