package testing

import (
	"fmt"
	"strings"
	"testing"
)

func TestNamedTestEffect_Diagnostics(t *testing.T) {
	for _, helper := range []string{"func check() -> bool ! {FS} = readFile(\"missing\") != \"\"", "export func check() -> bool ! {FS} = readFile(\"missing\") != \"\"", ""} {
		body := "check()"
		if helper == "" {
			body = `readFile("missing") != ""`
		}
		path := writeEngineSource(t, "effects.ail", fmt.Sprintf(`module effects
import std/fs (readFile)
%s
test "effectful helper" { %s }
test "pure sibling" { true }
property "effectful property" { forall(n: int) => %s }
`, helper, body, body))
		for _, mode := range [][2]bool{{false, false}, {true, false}, {true, true}} {
			res, _ := runWithExecutor(t, path, mode[0], mode[1])
			tests := byName(res)
			if tests["pure sibling"].Status != StatusPass {
				t.Fatalf("sibling: %+v", res.Tests)
			}
			tr := tests["effectful helper"]
			if tr.Status != StatusFail {
				t.Fatalf("effectful: %+v", tr)
			}
			for _, text := range []string{`named test "effectful helper"`, path + ":4", "checked pure", "Missing effects: FS", "ailang run --caps FS"} {
				if !strings.Contains(tr.Error, text) {
					t.Errorf("missing %q: %s", text, tr.Error)
				}
			}
			pr := propsByName(res)["effectful property"]
			if pr.Status != StatusFail || !strings.Contains(pr.Error, `property "effectful property"`) {
				t.Errorf("property: %+v", pr)
			}
			all := tr.Error + pr.Error + res.Engine.Summary()
			for _, f := range res.NamedBatchFailures {
				all += f.Notice() + f.Reason
				if f.HarnessBug {
					t.Error("user error blamed on harness")
				}
			}
			for _, forbidden := range []string{"__namedtest_", "$tmp", "ailang-namedtest-", "std/debug", "Suggested fix:"} {
				if strings.Contains(all, forbidden) {
					t.Errorf("leaked %q: %s", forbidden, all)
				}
			}
		}
	}
}

func TestNamedTestEffect_Mapper(t *testing.T) {
	e := &Executor{modulePath: "repro.ail"}
	ent := namedEntry{title: "example", testLine: 7, label: "test body"}
	for _, symbol := range []string{"__namedtest_0", "__namedtest_entry", "__namedtest_prop_2", "$tmp19"} {
		input := fmt.Sprintf("pipeline error: Effect checking failed for function '%s'\n  Missing effects: FS, IO\n  Current signature: func %s(...) -> T\n  Suggested fix: func %s(...) -> T ! {FS, IO}\n", symbol, symbol, symbol)
		got := e.mapEntryEffectError(input, ent)
		if !strings.Contains(got, "Missing effects: FS, IO") || !strings.Contains(got, "--caps FS,IO") || strings.Contains(got, symbol) {
			t.Errorf("mapper %s: %s", symbol, got)
		}
	}
	for _, msg := range []string{"Effect checking failed for function 'main'\n  Missing effects: FS", "undefined variable: check", "unrelated '__namedtest_0'", "Effect checking failed for function '__namedtest_bad'\n", "quoted Effect checking failed for function '__namedtest_0'\n"} {
		if got := e.mapEntryEffectError(msg, ent); got != msg {
			t.Errorf("rewrote control %q as %q", msg, got)
		}
	}
}

func TestNamedTestEffect_InvalidHelperPreservesModuleError(t *testing.T) {
	path := writeEngineSource(t, "invalid.ail", `module invalid
import std/fs (readFile)
func check() -> bool = readFile("missing") != ""
test "uses helper" { check() }
`)
	for _, mode := range [][2]bool{{false, false}, {true, false}, {true, true}} {
		res, _ := runWithExecutor(t, path, mode[0], mode[1])
		msg := res.Tests[0].Error
		if !strings.Contains(msg, "Effect checking failed for function 'check'") || !strings.Contains(msg, "Missing effects: FS") || strings.Contains(msg, "undefined variable") || strings.Contains(msg, "checked pure") || strings.Contains(msg, "ailang-namedtest-") {
			t.Fatalf("invalid helper: %s", msg)
		}
	}
}

func TestNamedTestEffect_StrictFailureIsUserError(t *testing.T) {
	path := engineFixtureFrom(t, "testdata/strip/named_test_effectful_direct.ail")
	res, _ := runWithExecutor(t, path, true, true)
	if len(res.NamedBatchFailures) != 1 || res.NamedBatchFailures[0].HarnessBug || res.Engine.StrictFailures != 1 || res.Engine.VMBodies != 0 {
		t.Fatalf("strict routing: %+v, %+v", res.NamedBatchFailures, res.Engine)
	}
}
