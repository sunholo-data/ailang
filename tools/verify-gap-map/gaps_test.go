package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestGlobPrefixAcceptsOnlyTheInventoryShapes(t *testing.T) {
	for g, want := range map[string]string{"examples/**/*.ail": "examples/", ".agent/**": ".agent/", "scripts/wip/**": "scripts/wip/"} {
		got, err := globPrefix(g)
		if err != nil || got != want {
			t.Errorf("globPrefix(%q) = %q, %v; want %q", g, got, err, want)
		}
	}
	for _, g := range []string{"*.ail", "examples/*.ail", "a/*/b/**", "**"} {
		if _, err := globPrefix(g); err == nil {
			t.Errorf("globPrefix(%q): want an error, a different match would be silent", g)
		}
	}
}

func TestContractFilesSkipsTestsCachesAndNestedCheckouts(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	contracted := "module m\nexport func f(x: int) -> int\n  ensures { result == x }\n{ x }\n"
	write("src/a.ail", contracted)
	write("src/plain.ail", "module m\nexport func f(x: int) -> int = x\n")
	write("src/a_test.ail", contracted)
	write(".ailang/cache/b.ail", contracted)
	write("wt/.git", "gitdir: elsewhere")
	write("wt/c.ail", contracted)
	write("wip/d.ail", contracted)
	got, err := contractFiles(gapCorpus{Root: root, Exclude: []string{"wip/**"}})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"src/a.ail"}; !slices.Equal(got, want) {
		t.Fatalf("contractFiles = %v, want %v", got, want)
	}
}

func TestNormaliseReasonFoldsNamesAndNumbers(t *testing.T) {
	a := normaliseReason(`Function "f" calls "foldChars" whose signature uses an unencodable type "a"`)
	b := normaliseReason(`Function "g" calls "decode" whose signature uses an unencodable type "Json"`)
	if a != b {
		t.Fatalf("same limitation, different buckets:\n%s\n%s", a, b)
	}
	if got := normaliseReason("depth 3 frontier at line 12"); got != "depth N frontier at line N" {
		t.Fatalf("got %q", got)
	}
}

func TestAttributeBlamesCalleeTypeAndBuiltin(t *testing.T) {
	a := attribute([]row{
		{Status: "skipped", Reason: `Function "f" calls "foldChars" whose signature uses an unencodable type "a"`},
		{Status: "skipped", Reason: `unsupported application std/string.words`},
		{Status: "verified", Reason: `calls "x" whose signature uses an unencodable type "y"`},
	})
	if a.Callees["foldChars"] != 1 || a.Types["a"] != 1 || a.Builtins["std/string.words"] != 1 || a.Callees["x"] != 0 {
		t.Fatalf("attribution = %+v", a)
	}
}

// A lost proof is a regression only when the source did not change; a new
// error or unknown likewise. Anything that changed with its source is drift.
func TestDiffSeparatesRegressionsFromSourceDrift(t *testing.T) {
	prev := &snapshot{
		Rows: []row{
			{Corpus: "c", File: "same.ail", Function: "lost", Status: "verified"},
			{Corpus: "c", File: "same.ail", Function: "gained", Status: "skipped"},
			{Corpus: "c", File: "same.ail", Function: "broke", Status: "skipped"},
			{Corpus: "c", File: "edited.ail", Function: "drift", Status: "verified"},
			{Corpus: "c", File: "same.ail", Function: "gone", Status: "verified"},
		},
		Files: map[string]string{"c|same.ail": "h1", "c|edited.ail": "h2"},
	}
	cur := &snapshot{
		Rows: []row{
			{Corpus: "c", File: "same.ail", Function: "lost", Status: "inconclusive"},
			{Corpus: "c", File: "same.ail", Function: "gained", Status: "verified"},
			{Corpus: "c", File: "same.ail", Function: "broke", Status: "error"},
			{Corpus: "c", File: "edited.ail", Function: "drift", Status: "counterexample"},
			{Corpus: "c", File: "same.ail", Function: "new", Status: "verified"},
		},
		Files: map[string]string{"c|same.ail": "h1", "c|edited.ail": "h3"},
	}
	trans, added, removed := diff(prev, cur)
	if added != 1 || removed != 1 || len(trans) != 4 {
		t.Fatalf("added %d removed %d transitions %v", added, removed, trans)
	}
	var regressed []string
	for _, tr := range trans {
		if tr.regression() {
			regressed = append(regressed, tr.Key)
		}
	}
	if want := []string{"c|same.ail|broke", "c|same.ail|lost"}; !slices.Equal(regressed, want) {
		t.Fatalf("regressions = %v, want %v", regressed, want)
	}
}

// A result whose every requires-carrying callee was checked is not
// "unchecked"; one reaching a callee only where calls are not checked is.
func TestD1SeparatesCheckedFromUnchecked(t *testing.T) {
	d := d1([]row{
		{Status: "verified", RequiresCallees: []string{"a"}, RequiresChecked: []string{"a"}},
		{Status: "verified", RequiresCallees: []string{"a", "b"}, RequiresChecked: []string{"a"}},
		{Status: "skipped", RequiresCallees: []string{"c"}},
		{Status: "verified"},
	})
	if d.Results != 3 || d.ByStatus["verified"] != 2 || d.Unchecked["verified"] != 1 || d.Unchecked["skipped"] != 1 {
		t.Fatalf("d1 = %+v", d)
	}
}

func TestCaveatCountsOnlyVerified(t *testing.T) {
	c := caveatCounts([]row{
		{Status: "verified", Caveats: []string{"int_overflow"}},
		{Status: "verified", Caveats: []string{"int_overflow", "float_inputs_assumed_non_nan"}},
		{Status: "counterexample", Caveats: []string{"int_overflow"}},
	})
	if c["int_overflow"] != 2 || c["float_inputs_assumed_non_nan"] != 1 {
		t.Fatalf("caveatCounts = %v", c)
	}
}
