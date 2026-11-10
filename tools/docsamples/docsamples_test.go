package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const fixturePage = "docs/docs/fixture.md"

var markerRe = regexp.MustCompile(`marker: ([a-z-]+)`)

// fixtureBlocks extracts a fixture page and indexes its blocks by marker.
func fixtureBlocks(t *testing.T, root, page string) map[string]*Block {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, page))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*Block{}
	for _, b := range ExtractBlocks(page, page, string(raw)) {
		switch {
		case markerRe.MatchString(b.Body):
			out[markerRe.FindStringSubmatch(b.Body)[1]] = b
		case b.Class == ClassRepl:
			out["repl"] = b
		case strings.HasPrefix(b.Body, "let answer"): // marker-less: a `--` marker is itself a hint
			out["probe-untagged"] = b
		case strings.HasPrefix(b.Body, "match parse"):
			out["hint-match"] = b
		case strings.HasPrefix(b.Body, "let r = run"):
			out["hint-adt-arm"] = b
		}
	}
	return out
}

func TestExtractClassifiesTheInfoString(t *testing.T) {
	bs := fixtureBlocks(t, "testdata/site", fixturePage)
	want := map[string]Class{
		"pass-program": ClassProgram, "pass-header": ClassProgram, "main-needs-module": ClassProgram,
		"fail-untagged": ClassProgram, "fragment-parse": ClassParse, "error-fails-correctly": ClassError,
		"error-wrongly-passes": ClassError, "error-wrong-code": ClassError, "error-prefix-only": ClassError,
		"error-second-code": ClassError, "error-pseudo-code": ClassBadTag, "baselined-failure": ClassProgram,
		"baselined-code-changed": ClassProgram, "stale-baseline": ClassProgram, "sketch": ClassSketch,
		"sketch-unlisted": ClassSketch, "repl": ClassRepl, "file-lib": ClassProgram, "file-user": ClassProgram,
		"tilde-fence": ClassProgram, "misfenced": ClassMisfenced, "misfenced-untagged": ClassMisfenced,
		"misfenced-parse-probe": ClassCandidate, "bad-tag": ClassBadTag, "stray-fence": ClassStructural,
		"unterminated": ClassStructural, "probe-untagged": ClassCandidate, "hint-match": ClassMisfenced,
		"hint-adt-arm": ClassMisfenced,
	}
	for m, c := range want {
		if b := bs[m]; b == nil {
			t.Errorf("marker %s: block not extracted", m)
		} else if b.Class != c {
			t.Errorf("marker %s: class %s, want %s", m, b.Class, c)
		}
	}
	if len(bs) != len(want) {
		t.Fatalf("extracted %d marked blocks, want %d (TypeScript, bash and output blocks must be ignored)", len(bs), len(want))
	}
	if bs["error-fails-correctly"].WantCode != "PAR_UNEXPECTED_TOKEN" || bs["file-lib"].File != "fixture/lib.ail" {
		t.Errorf("error=/file= values not parsed: %+v %+v", bs["error-fails-correctly"], bs["file-lib"])
	}
	if bs["stray-fence"].Defect != "DOC_FENCE_PAIRING" || bs["unterminated"].Defect != "DOC_UNTERMINATED" {
		t.Errorf("structural defects: %q %q", bs["stray-fence"].Defect, bs["unterminated"].Defect)
	}
}

// A CRLF checkout (Windows, autocrlf) must yield the same blocks and the same keys,
// or a baseline written on one OS would read as all-orphaned on the other.
func TestExtractIsLineEndingIndependent(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata/site", fixturePage))
	if err != nil {
		t.Fatal(err)
	}
	lf := strings.ReplaceAll(string(raw), "\r\n", "\n")
	a := ExtractBlocks(fixturePage, fixturePage, lf)
	b := ExtractBlocks(fixturePage, fixturePage, strings.ReplaceAll(lf, "\n", "\r\n"))
	if len(a) == 0 || len(a) != len(b) {
		t.Fatalf("LF: %d blocks, CRLF: %d blocks", len(a), len(b))
	}
	for i := range a {
		if a[i].Key != b[i].Key || a[i].Class != b[i].Class || a[i].Line != b[i].Line {
			t.Errorf("block %d differs: LF %+v vs CRLF %+v", i, *a[i], *b[i])
		}
	}
}

func TestInfoStringGrammar(t *testing.T) {
	cases := map[string]Class{
		``:                                ClassProgram,
		`title="a b.ail" showLineNumbers`: ClassProgram,
		`{1,3-4} parse`:                   ClassParse,
		`error=PAR015`:                    ClassError,
		`error=PAR_UNEXPECTED`:            ClassError, // a well-formed code; whether it is THE code is the check's job
		`file=a/b.ail error=MOD014`:       ClassError,
		`error=OTHER`:                     ClassBadTag, // the gate's own fallback class, not a binary code
		`error=TC_UNDEFINED_VAR`:          ClassBadTag,
		`error=DOC_MISFENCED`:             ClassBadTag,
		`error=par015`:                    ClassBadTag,
		`parse sketch`:                    ClassBadTag, // two classes
		`file=a/b.ail sketch`:             ClassBadTag, // a file must be checked
		`file=../x.ail`:                   ClassBadTag, // escapes the stage
		`error=`:                          ClassBadTag,
		`fragment`:                        ClassBadTag, // unknown word
	}
	for info, want := range cases {
		b := &Block{Info: info}
		classify(b)
		if b.Class != want {
			t.Errorf("info %q: class %s, want %s", info, b.Class, want)
		}
	}
}

func TestKeyIgnoresWhitespaceNotContent(t *testing.T) {
	a := KeyOf("export func f() -> int = 1\n")
	if KeyOf("\n\nexport func f() -> int = 1   \n\n") != a {
		t.Error("surrounding blank lines / trailing spaces changed the key")
	}
	if KeyOf("export func f() -> int = 2\n") == a {
		t.Error("a content change kept the key")
	}
	if len(a) != 12 {
		t.Errorf("key %q is not 12 hex chars", a)
	}
}

func TestScopeCoversEveryDocsDirectory(t *testing.T) {
	for rel, want := range map[string]bool{
		"docs/docs/reference/stability.md": false, "docs/docs/guides/x.mdx": false, "docs/docs/intro.mdx": false,
		"docs/docs/prompts/index.md": false, "docs/docs/prompts/v0.16.12.md": true, "docs/docs/prompts/current.md": true,
		"docs/docs/packages/index.mdx": false, "docs/docs/packages/sunholo/x.md": true,
	} {
		if excluded(rel) != want {
			t.Errorf("excluded(%s) = %v, want %v", rel, !want, want)
		}
	}
	s, err := Scope("testdata/site")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(s.Pages, ",") != "docs/docs/fixture.md,docs/docs/reference/imports.mdx" || strings.Join(s.JS, ",") != "docs/src/pages/index.jsx" {
		t.Errorf("scope: pages %v, js %v", s.Pages, s.JS)
	}
}

func TestImportsAndLiterals(t *testing.T) {
	raw, _ := os.ReadFile("testdata/site/docs/docs/reference/imports.mdx")
	imps, defects := ImportsOf("testdata/site", "docs/docs/reference/imports.mdx", string(raw))
	if len(imps) != 2 || imps[0].Path != "examples/fixture/good.ail" || imps[1].Path != "examples/fixture/bad.ail" {
		t.Errorf("imports: %+v", imps)
	}
	codes := []string{}
	for _, d := range defects {
		codes = append(codes, d.Defect)
	}
	if strings.Join(codes, ",") != "DOC_MISSING_IMPORT,DOC_MISFENCED" {
		t.Errorf("defects: %v", codes)
	}
	jsx, _ := os.ReadFile("testdata/site/docs/src/pages/index.jsx")
	lits := LiteralsOf("docs/src/pages/index.jsx", string(jsx))
	if len(lits) != 1 || !strings.Contains(lits[0].Body, "module fixture/hero") || lits[0].Line != 5 {
		t.Errorf("literals (want the AILANG one only, not the CSS): %+v", lits)
	}
}

func list(es ...Entry) *List {
	l := &List{Entries: map[string]Entry{}}
	for _, e := range es {
		l.Entries[e.id()] = e
		l.Order = append(l.Order, e.id())
	}
	return l
}

func TestJudgeAppliesTheLists(t *testing.T) {
	mk := func(key string, c Class) *Block { return &Block{Page: "p.md", Source: "p.md", Key: key, Class: c} }
	blocks := []*Block{
		mk("aaaaaaaaaaaa", ClassProgram), mk("bbbbbbbbbbbb", ClassProgram), mk("cccccccccccc", ClassProgram),
		mk("dddddddddddd", ClassProgram), mk("eeeeeeeeeeee", ClassSketch), mk("ffffffffffff", ClassRepl),
		mk("111111111111", ClassProgram), mk("222222222222", ClassMisfenced), mk("333333333333", ClassSketch),
	}
	out := map[*Block]*Outcome{
		blocks[0]: {Ran: true, Pass: true}, blocks[1]: {Ran: true, Code: "X"}, blocks[2]: {Ran: true, Code: "Z"},
		blocks[3]: {Ran: true, Pass: true}, blocks[4]: {}, blocks[5]: {},
		blocks[6]: {Ran: true, Code: "NEW"}, blocks[7]: {Ran: true, Code: "DOC_MISFENCED"}, blocks[8]: {},
	}
	bl := list(Entry{"p.md", "cccccccccccc", "Z", ""}, Entry{"p.md", "dddddddddddd", "X", ""}, Entry{"p.md", "eeeeeeeeeeee", "X", ""},
		Entry{"p.md", "111111111111", "OLD", ""}, Entry{"p.md", "222222222222", "TC_TYPE", ""}, Entry{"p.md", "999999999999", "X", ""})
	un := list(Entry{"p.md", "ffffffffffff", "repl", ""}, Entry{"p.md", "333333333333", "repl", ""}, Entry{"p.md", "888888888888", "sketch", ""})
	got := Judge(blocks, out, bl, un)
	want := []Verdict{VOK, VFail, VBaselined, VStale, VStale, VSkipped, VFail, VFail, VFail, VOrphan, VOrphan}
	if len(got) != len(want) {
		t.Fatalf("got %d verdicts, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Verdict != w {
			t.Errorf("verdict %d: %s, want %s (%s)", i, got[i].Verdict, w, got[i].Why)
		}
	}
	// A baselined block re-fenced as typescript keeps its key: it must be judged as a
	// structural defect (no list can excuse it), not as an ordinary code change.
	if got[7].Why != "" {
		t.Errorf("a listed block that turned structural must fail as structural, got %q", got[7].Why)
	}
	if !strings.Contains(got[6].Why, "baselined as OLD but now fails with NEW") {
		t.Errorf("a changed code must say so: %q", got[6].Why)
	}
}

func TestListsRefuseStructuralCodes(t *testing.T) {
	p := filepath.Join(t.TempDir(), "b.txt")
	if err := os.WriteFile(p, []byte("p.md 0123456789ab DOC_MISFENCED\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadList(p, false); err == nil || !strings.Contains(err.Error(), "cannot be baselined") {
		t.Errorf("a DOC_* line must be refused, got %v", err)
	}
}

func TestAddedIsASetRule(t *testing.T) {
	base := list(Entry{"p.md", "aaaaaaaaaaaa", "X", ""}, Entry{"p.md", "bbbbbbbbbbbb", "X", ""})
	swapped := list(Entry{"p.md", "aaaaaaaaaaaa", "X", ""}, Entry{"p.md", "cccccccccccc", "X", ""})
	if got := Added(swapped, base, nil); len(got) != 1 || got[0] != "p.md cccccccccccc" {
		t.Errorf("a swap keeps the count but adds a key: Added = %v", got)
	}
	marked := list(Entry{"p.md", "cccccccccccc", "sketch", "reviewed: body-less signature"}, Entry{"p.md", "dddddddddddd", "sketch", "reviewed:"})
	if got := Added(marked, base, reviewed); len(got) != 1 || got[0] != "p.md dddddddddddd" {
		t.Errorf("only a non-empty reviewed: marker admits a new unchecked line: %v", got)
	}
}
