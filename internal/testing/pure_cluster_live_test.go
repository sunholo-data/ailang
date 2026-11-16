package testing

import (
	"errors"
	"strings"
	"testing"
)

// effectfulClusterSource covers the three shapes the live purity gate must
// tell apart:
//   - work -> logIt, both ! {IO}: an effectful function with an effectful
//     dependency (rejected);
//   - selfEff, ! {IO}, no dependencies (rejected);
//   - answer, pure, whose dependency closure holds the effectful render only as
//     a stored record field it never calls (accepted: the shape of
//     examples/runnable/explicit_empty_callback_contracts.ail);
//   - usesHelper -> helper, both pure (accepted, the plain positive control).
const effectfulClusterSource = `module effectful_cluster

import std/io (println)

func logIt(x: int) -> int ! {IO} {
  println("logging");
  x
}

func work(x: int) -> int ! {IO}
  tests [ (1, 1) ] {
  logIt(x)
}

func selfEff(x: int) -> int ! {IO}
  tests [ (3, 3) ] {
  println("self");
  x
}

type Hooks = { probe: (int) -> int ! {}, render: (int) -> int ! {IO} }

pure func inc(x: int) -> int = x + 1
func render(x: int) -> int ! {IO} = { println("stored callback invoked"); x }
pure func build() -> Hooks = {probe: inc, render: render}

pure func answer() -> int
  tests [ ((), 42) ] {
  let h = build();
  h.probe(41)
}

pure func helper(x: int) -> int { x + 1 }

pure func usesHelper(x: int) -> int
  tests [ (1, 2) ] {
  helper(x)
}
`

// TestExtractPureClusterForFunction_RejectsEffectful is the live-path purity
// gate: the extractor the runner calls must not return a "pure" cluster for an
// effectful function, and must not reject a pure one for storing an effectful
// value it never calls.
func TestExtractPureClusterForFunction_RejectsEffectful(t *testing.T) {
	r, _ := parseDeriveFixture(t, effectfulClusterSource)

	for _, fn := range []string{"work", "selfEff"} {
		cluster, err := r.executor.ExtractPureClusterForFunction(fn)
		var purityErr *PurityError
		if !errors.As(err, &purityErr) {
			t.Fatalf("%s: want *PurityError, got cluster=%v err=%v", fn, cluster, err)
		}
		if cluster != nil {
			t.Errorf("%s: want nil cluster alongside a PurityError, got %v", fn, cluster.Names)
		}
		if purityErr.FuncUnderTest != fn || len(purityErr.Effects) != 1 || purityErr.Effects[0] != "IO" {
			t.Errorf("%s: PurityError = %+v, want FuncUnderTest=%s Effects=[IO]", fn, purityErr, fn)
		}
	}

	for fn, wantNames := range map[string][]string{
		"usesHelper": {"usesHelper", "helper"},
		"answer":     {"answer", "build", "inc", "render"},
	} {
		cluster, err := r.executor.ExtractPureClusterForFunction(fn)
		if err != nil {
			t.Fatalf("%s: unexpected error %v", fn, err)
		}
		for _, name := range wantNames {
			if !cluster.Names[name] {
				t.Errorf("%s cluster = %v, missing %s", fn, cluster.Names, name)
			}
		}
	}
}

// TestRunner_EffectfulInlineTestReportsPurityError checks the caller side:
// the runner reports the purity rejection as the test's failure instead of
// swallowing it and falling back to a harness that dies on a missing
// capability, and pure functions still pass.
func TestRunner_EffectfulInlineTestReportsPurityError(t *testing.T) {
	result := runInlineTestsOnSource(t, effectfulClusterSource)

	byName := map[string]TestResult{}
	for _, tr := range result.Tests {
		byName[tr.Name] = tr
	}
	for _, fn := range []string{"work", "selfEff"} {
		tr, ok := byName[fn+"_test_1"]
		if !ok {
			t.Fatalf("missing result for %s_test_1; have %v", fn, byName)
		}
		if tr.Status != StatusFail || !strings.Contains(tr.Error, "cannot test '"+fn+"': it has effects [IO]") {
			t.Errorf("%s_test_1: status=%v error=%q, want FAIL naming the purity rejection", fn, tr.Status, tr.Error)
		}
		if strings.Contains(tr.Error, "requires capability") {
			t.Errorf("%s_test_1: error still comes from the capability-less harness: %q", fn, tr.Error)
		}
	}
	for _, name := range []string{"usesHelper_test_1", "answer_test_1"} {
		if tr := byName[name]; tr.Status != StatusPass {
			t.Errorf("%s: status=%v error=%q, want PASS", name, tr.Status, tr.Error)
		}
	}
}
