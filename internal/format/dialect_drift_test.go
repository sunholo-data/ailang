package format

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// TestFmtDoesNotDriftFromTeachingPrompt is the gate that would have caught two
// weeks of wasted eval time.
//
// # WHY THIS EXISTS
//
// `ailang prompt` is the canonical teaching text handed to every model in every
// eval. `ailang fmt` tells a model what canonical AILANG looks like. If the two
// disagree, the model is given contradictory instructions on every write — and
// it obeys the formatter, because the formatter speaks last.
//
// That is not hypothetical. Measured 2026-07-30, from a model's own context:
//
//	fmt: your file is UNCHANGED on disk. Canonical AILANG would differ here:
//	+pure func contains(x: int, xs: list[int]) -> bool = match xs {
//	+  ::(h, t) => if x == h then true else contains(x, t)
//	  (replacing:)
//	-pure func contains(x: int, xs: [int]) -> bool =
//	-    h :: t => if x == h then true else contains(x, t)
//	  ailang check: OK - file type-checks.
//
// `check` says the code is fine; fmt calls it non-canonical. The model wrote
// "the file type-checks but the formatter wants canonical style", switched
// dialect, and broke working code. Cost: +62% output tokens (p=0.011) and a
// fortnight spent blaming the model.
//
// # WHY A TYPE-CHECK CANNOT CATCH THIS
//
// Both spellings PARSE and mean the same thing. `Parse(fmt(x)) ≡ Parse(x)`
// holds. Only an assertion about DIALECT — which of several valid spellings we
// teach — can catch it, which is why this test compares token streams against
// the prompt rather than checking validity.
func TestFmtDoesNotDriftFromTeachingPrompt(t *testing.T) {
	blocks := teachingPromptBlocks(t)
	if len(blocks) < 20 {
		t.Fatalf("only %d ailang blocks found in prompts/ — the extractor is broken, "+
			"and a silently-empty corpus would make this gate pass vacuously", len(blocks))
	}

	var drifted []string
	compared := 0
	for _, b := range blocks {
		prog, ok := tryParse(b.src)
		if !ok {
			continue // fragments and snippets are not formattable; not drift
		}
		out, err := Source(prog, Options{Indent: "  "})
		if err != nil {
			continue
		}
		compared++
		before, after := dialectTokens(b.src), dialectTokens(string(out))
		// Compare token MULTISETS, not sequences. Reordering is layout — the
		// formatter is allowed to move a `match` onto the `=` line or restructure
		// a block. Dialect drift is tokens APPEARING or DISAPPEARING: `[int]`
		// becoming `list[int]`, `h :: t` becoming `::(h, t)`, an interpolation
		// becoming a concat_String chain. Comparing sequences flagged 8 pure
		// reorderings for every 1 real substitution.
		if sameTokenMultiset(before, after) {
			continue
		}
		drifted = append(drifted, b.name)
	}

	if compared < 10 {
		t.Fatalf("only %d blocks were parseable; the gate is not actually comparing anything", compared)
	}
	// RATCHET, not a clean gate. When you fix one, LOWER this number; it must
	// never rise.
	//
	// 2026-07-31, M-FMT-DIALECT-ALIGNMENT: 9 → 4. What was fixed, and why each
	// was safe — in every case the two spellings parse to the IDENTICAL AST, so
	// re-emitting the taught one round-trips exactly:
	//
	//	interpolation   concat_String(concat_String(show(a)," x"),…) → "${a} x…"  (interp.go)
	//	function type   (int) -> int                                 → int -> int (types.go)
	//	unit call       now(())                                      → now()      (expr.go)
	//	record pattern  {name: name, age: age}                       → {name, age}(pattern.go)
	//	open record     { email: string | _r0 }                      → { email: string, ... }
	//
	// The previous count of 9 was measured with a `knownDivergence` allowlist that
	// excused any block whose output gained a concat_String. That hid COMPOUND
	// cases: blocks drifting for interpolation AND for another reason were waved
	// through on the interpolation alone, so the true residual was 16, not 9. The
	// allowlist is gone — it now excuses nothing, and an allowlist that excuses
	// nothing is a rubber stamp.
	//
	// 2026-07-31, second pass: 4 → 2. Two of those four were the PROMPT being
	// wrong, not fmt, and were fixed by cutting prompt v0.16.4:
	//
	//	#19  v0.16.3 taught that `letrec` with a `= ...` body is a PARSE ERROR. It
	//	     type-checks AND runs (verified with `ailang check` + `ailang run`);
	//	     the claim went stale when M-SYNTAX-AI-FORGIVING R1 made an equation
	//	     body a `;`-separated statement sequence. Teaching a model to avoid
	//	     valid syntax is the same failure class as fmt contradicting the prompt.
	//	#71  `Result[List[string], string]` → `Result[[string], string]`. `[T]` is
	//	     taught 64 times and is what fmt emits, so `List[T]` was a spelling fmt
	//	     would rewrite. Both parse to the identical TypeApp.
	//
	// The 2 that remain are diagnosed, and neither is fixable in the formatter:
	//
	//	#63  `=> { -- comment \n expr }` unwraps to `=> expr`. The block existed
	//	     only to host a comment; Source() drops comments by construction, so
	//	     the wrapper has nothing left to hold.
	//	#86  `price - (price * pct) / 100` → `price - price * pct / 100`. The AST
	//	     has no ParenExpr node (design V20), so redundant source parens are
	//	     simply absent from the tree and cannot be recovered. Unfixable without
	//	     changing the AST; semantically identical.
	const knownDrift = 2
	if len(drifted) > knownDrift {
		t.Errorf("`ailang fmt` now rewrites %d teaching-prompt examples into a different dialect, up from %d: %v\n\n"+
			"A NEW divergence has landed. Every one of these tells a model its correct code is\n"+
			"non-canonical — that is how the fmt extension cost +62%% output tokens (p=0.011).\n"+
			"Either fix the formatter to emit what the prompt teaches, or update the prompt and\n"+
			"add the case to knownDivergence with a reason.",
			len(drifted), knownDrift, drifted)
	}
	if len(drifted) < knownDrift {
		t.Errorf("drift is down to %d from %d — good. Lower knownDrift to %d to lock it in.",
			len(drifted), knownDrift, len(drifted))
	}
}

type promptBlock struct {
	name string
	src  string
}

var fenceRe = regexp.MustCompile("(?s)```ailang\n(.*?)```")

// teachingPromptBlocks collects the fenced ```ailang examples from the ACTIVE
// prompt versions only — the exact text `ailang prompt` serves to models today.
//
// Walking every historical version instead reports ~457 "divergences", almost
// all of them prompts written for syntax that has since changed. Those are not
// drift; they are history. Only what we teach NOW can contradict what fmt emits
// now.
func teachingPromptBlocks(t *testing.T) []promptBlock {
	t.Helper()
	var out []promptBlock
	for _, spec := range []struct{ index, dir string }{
		{filepath.Join("..", "..", "prompts", "versions.json"), filepath.Join("..", "..", "prompts")},
		{filepath.Join("..", "..", "prompts", "agent", "versions.json"), filepath.Join("..", "..", "prompts", "agent")},
	} {
		raw, err := os.ReadFile(spec.index)
		if err != nil {
			t.Fatalf("reading %s: %v", spec.index, err)
		}
		var idx struct {
			Active string `json:"active"`
		}
		if err := json.Unmarshal(raw, &idx); err != nil {
			t.Fatalf("parsing %s: %v", spec.index, err)
		}
		if idx.Active == "" {
			t.Fatalf("%s has no `active` version — cannot tell what we teach", spec.index)
		}
		path := filepath.Join(spec.dir, idx.Active+".md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("active prompt %s missing: %v", path, err)
		}
		for i, m := range fenceRe.FindAllStringSubmatch(string(data), -1) {
			if strings.TrimSpace(m[1]) == "" {
				continue
			}
			out = append(out, promptBlock{name: idx.Active + ".md#" + itoa(i), src: m[1]})
		}
	}
	return out
}

var commentRe = regexp.MustCompile(`--[^\n]*`)
var tokenRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*|::|->|=>|\|\||&&|[^\sA-Za-z0-9_]`)

// dialectTokens reduces source to a whitespace- and comment-insensitive token
// stream, with statement-terminating `;` dropped.
//
// Layout is the formatter's job and differences there are not drift. `;` is
// excluded because the prompt documents both `;`-separated and newline-separated
// statements as equivalent (v0.29+ forgiving separators), so removing one is a
// layout choice rather than a change of dialect.
func dialectTokens(src string) string {
	src = commentRe.ReplaceAllString(src, "")
	toks := tokenRe.FindAllString(src, -1)
	kept := toks[:0]
	for _, tk := range toks {
		if tk == ";" {
			continue
		}
		kept = append(kept, tk)
	}
	return strings.Join(kept, " ")
}

// tryParse parses a prompt snippet, reporting ok=false for fragments that were
// never meant to stand alone (illustrative one-liners, `...` elisions). Those
// are not drift and must not be counted either way.
func tryParse(src string) (*ast.Program, bool) {
	p := parser.New(lexer.New(src, "test://dialect"))
	prog := p.Parse()
	if len(p.Errors()) > 0 || prog == nil || prog.File == nil {
		return nil, false
	}
	return prog, true
}

// sameTokenMultiset reports whether two token streams contain the same tokens
// with the same counts, ignoring order.
func sameTokenMultiset(a, b string) bool {
	counts := map[string]int{}
	for _, t := range strings.Fields(a) {
		counts[t]++
	}
	for _, t := range strings.Fields(b) {
		counts[t]--
	}
	for _, n := range counts {
		if n != 0 {
			return false
		}
	}
	return true
}
