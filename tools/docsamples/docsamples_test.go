package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

const fixturePage = "docs/docs/fixture.md"

var markerRe = regexp.MustCompile(`marker: ([a-z-]+)`)

// fixtureBlocks extracts the fixture page and indexes its blocks by marker.
func fixtureBlocks(t *testing.T, root string) map[string]*Block {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, fixturePage))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*Block{}
	for _, b := range ExtractBlocks(fixturePage, fixturePage, string(raw)) {
		if m := markerRe.FindStringSubmatch(b.Body); m != nil {
			out[m[1]] = b
		} else if b.Class == ClassRepl {
			out["repl"] = b
		}
	}
	return out
}

func TestExtractClassifiesTheInfoString(t *testing.T) {
	bs := fixtureBlocks(t, "testdata/site")
	want := map[string]Class{
		"pass-program": ClassProgram, "pass-header": ClassProgram, "fail-untagged": ClassProgram,
		"fragment-parse": ClassParse, "error-fails-correctly": ClassError, "error-wrongly-passes": ClassError,
		"error-wrong-code": ClassError, "baselined-failure": ClassProgram, "stale-baseline": ClassProgram,
		"sketch": ClassSketch, "repl": ClassRepl, "file-lib": ClassProgram, "file-user": ClassProgram,
		"misfenced": ClassMisfenced, "bad-tag": ClassBadTag,
	}
	for m, c := range want {
		b := bs[m]
		if b == nil {
			t.Errorf("marker %s: block not extracted", m)
			continue
		}
		if b.Class != c {
			t.Errorf("marker %s: class %s, want %s", m, b.Class, c)
		}
	}
	if len(bs) != len(want) {
		t.Fatalf("extracted %d marked blocks, want %d (the TypeScript and bash blocks must be ignored)", len(bs), len(want))
	}
	if bs["error-fails-correctly"].WantCode != "PAR_UNEXPECTED_TOKEN" || bs["file-lib"].File != "fixture/lib.ail" {
		t.Errorf("error=/file= values not parsed: %+v %+v", bs["error-fails-correctly"], bs["file-lib"])
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
		`file=a/b.ail error=MOD014`:       ClassError,
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

func TestJudgeAppliesTheBaseline(t *testing.T) {
	mk := func(key string, c Class) *Block { return &Block{Page: "p.md", Key: key, Class: c} }
	pass, fail, listedFail, listedPass, sketchListed, skipped :=
		mk("aaaaaaaaaaaa", ClassProgram), mk("bbbbbbbbbbbb", ClassProgram), mk("cccccccccccc", ClassProgram),
		mk("dddddddddddd", ClassProgram), mk("eeeeeeeeeeee", ClassSketch), mk("ffffffffffff", ClassRepl)
	blocks := []*Block{pass, fail, listedFail, listedPass, sketchListed, skipped}
	out := map[*Block]*Outcome{
		pass: {Ran: true, Pass: true}, fail: {Ran: true, Code: "X"}, listedFail: {Ran: true, Code: "Y"},
		listedPass: {Ran: true, Pass: true}, sketchListed: {}, skipped: {},
	}
	bl := &Baseline{Entries: map[string]Entry{}}
	for _, e := range []Entry{{"p.md", "cccccccccccc", "Z"}, {"p.md", "dddddddddddd", "X"}, {"p.md", "eeeeeeeeeeee", "X"}, {"p.md", "999999999999", "X"}} {
		bl.Entries[e.id()] = e
		bl.Order = append(bl.Order, e.id())
	}
	got := Judge(blocks, out, bl)
	want := []Verdict{VOK, VFail, VBaselined, VStale, VStale, VSkipped, VOrphan}
	if len(got) != len(want) {
		t.Fatalf("got %d verdicts, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Verdict != w {
			t.Errorf("verdict %d: %s, want %s", i, got[i].Verdict, w)
		}
	}
	if got[2].CodeNote == "" {
		t.Error("a baselined block whose first code changed carries no note")
	}
}

// ailangBin is the binary the fixture runs against. make test-check-doc-samples sets
// DOCSAMPLES_AILANG to a binary built from this commit and then a missing binary is
// a FAILURE; under a plain `go test ./...` the shared stale-aware finder may skip.
func ailangBin(t *testing.T) string {
	if p := os.Getenv("DOCSAMPLES_AILANG"); p != "" {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("DOCSAMPLES_AILANG=%s: %v", p, err)
		}
		return p
	}
	return testutil.FindAilangBinary(t)
}

// copyFixture copies testdata/site into a temp dir and writes the fixture baseline:
// the baselined-failure block, the stale-baseline block (which passes), and one
// orphan entry for a block that does not exist.
func copyFixture(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	raw, err := os.ReadFile(filepath.Join("testdata/site", fixturePage))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dst, "docs/docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, fixturePage), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	bs := fixtureBlocks(t, dst)
	bl := baselineHeader +
		fixturePage + " " + bs["baselined-failure"].Key + " TC_UNDEFINED_VAR\n" +
		fixturePage + " " + bs["stale-baseline"].Key + " TC_UNDEFINED_VAR\n" +
		fixturePage + " 000000000000 PAR015\n"
	if err := os.MkdirAll(filepath.Join(dst, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "scripts/doc_samples_baseline.txt"), []byte(bl), 0o644); err != nil {
		t.Fatal(err)
	}
	return dst
}

func repoStd(t *testing.T) string {
	p, err := filepath.Abs("../../std")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// TestFixtureVerdicts is the control: one block per verdict, checked by the real
// binary, each asserted exactly — including the gate's exit code and that the
// failure lines name page:line, the tag, the code and a repro command.
func TestFixtureVerdicts(t *testing.T) {
	bin := ailangBin(t)
	root := copyFixture(t)
	var buf bytes.Buffer
	jsonPath := filepath.Join(t.TempDir(), "out.json")
	rc := run(options{root: root, bin: bin, baseline: "scripts/doc_samples_baseline.txt", stdlib: repoStd(t), jobs: 4, jsonOut: jsonPath}, &buf)
	out := buf.String()
	if rc != 1 {
		t.Fatalf("exit %d, want 1 (the fixture has deliberate failures)\n%s", rc, out)
	}
	bs := fixtureBlocks(t, root)
	want := map[string]string{
		"pass-program":          "ok",
		"pass-header":           "ok",
		"fail-untagged":         "FAIL TC_UNDEFINED_VAR",
		"fragment-parse":        "ok",
		"error-fails-correctly": "ok",
		"error-wrongly-passes":  "FAIL DOC_EXPECTED_FAIL",
		"error-wrong-code":      "FAIL DOC_WRONG_ERROR",
		"baselined-failure":     "baselined",
		"stale-baseline":        "STALE",
		"sketch":                "skipped",
		"repl":                  "skipped",
		"file-lib":              "ok",
		"file-user":             "ok",
		"misfenced":             "FAIL DOC_MISFENCED",
		"bad-tag":               "FAIL DOC_BADTAG",
	}
	rows := readRows(t, jsonPath)
	for m, w := range want {
		b := bs[m]
		r, ok := rows[b.Line]
		if !ok {
			t.Errorf("%s (line %d): no verdict", m, b.Line)
			continue
		}
		got := r.Verdict
		if strings.HasPrefix(w, "FAIL ") {
			got += " " + r.Code
		}
		if got != w {
			t.Errorf("%s (line %d): %q, want %q (%s)", m, b.Line, got, w, r.FirstErr)
		}
	}
	if !strings.Contains(out, "ORPHAN    docs/docs/fixture.md 000000000000") {
		t.Errorf("orphan baseline entry not reported:\n%s", out)
	}
	line := bs["fail-untagged"].Line
	for _, frag := range []string{
		"FAIL      docs/docs/fixture.md:" + itoa(line) + " [ailang] TC_UNDEFINED_VAR: ",
		"repro: make check-doc-samples DOC_SAMPLES_ARGS='-only docs/docs/fixture.md:" + itoa(line),
		"NEW failures 5", "stale baseline 1, orphaned 1",
	} {
		if !strings.Contains(out, frag) {
			t.Errorf("report lacks %q:\n%s", frag, out)
		}
	}
}

// TestFixtureGreenOnceBaselined: the same tree, with the failures listed and the
// stale/orphan entries gone, exits 0 — so the gate is not simply always red.
func TestFixtureGreenOnceBaselined(t *testing.T) {
	bin := ailangBin(t)
	root := copyFixture(t)
	var buf bytes.Buffer
	o := options{root: root, bin: bin, baseline: "scripts/doc_samples_baseline.txt", stdlib: repoStd(t), jobs: 4, write: true}
	if rc := run(o, &buf); rc != 0 {
		t.Fatalf("after -write-baseline: exit %d, want 0\n%s", rc, buf.String())
	}
	o.write = false
	buf.Reset()
	if rc := run(o, &buf); rc != 0 {
		t.Fatalf("re-run against the written baseline: exit %d, want 0\n%s", rc, buf.String())
	}
	raw, _ := os.ReadFile(filepath.Join(root, "scripts/doc_samples_baseline.txt"))
	if strings.Contains(string(raw), "000000000000") || strings.Count(string(raw), fixturePage) != 6 {
		t.Errorf("written baseline should list exactly the 6 failing blocks:\n%s", raw)
	}
}

func TestShrinkOnlyAgainstBaseRef(t *testing.T) {
	bin := ailangBin(t)
	root := copyFixture(t)
	git := func(args ...string) {
		cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@example.invalid"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	o := options{root: root, bin: bin, baseline: "scripts/doc_samples_baseline.txt", stdlib: repoStd(t), jobs: 4, write: true}
	var buf bytes.Buffer
	run(o, &buf) // seed: 6 entries
	git("add", "-A")
	git("commit", "-q", "-m", "seed")
	// Grow the baseline by one line: the ratchet must refuse it against HEAD's count.
	o.write, o.baseRef = false, "HEAD"
	bl := filepath.Join(root, o.baseline)
	raw, _ := os.ReadFile(bl)
	if err := os.WriteFile(bl, append(raw, []byte(fixturePage+" 0123456789ab EXTRA\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if rc := run(o, &buf); rc != 1 || !strings.Contains(buf.String(), "FAIL shrink-only") {
		t.Fatalf("a grown baseline: exit %d, want 1 with a shrink-only failure\n%s", rc, buf.String())
	}
	o.baseRef = "no-such-ref"
	buf.Reset()
	if rc := run(o, &buf); rc != 2 {
		t.Fatalf("a missing base ref: exit %d, want 2 (instrument failure, never a silent pass)\n%s", rc, buf.String())
	}
}

func TestRefusesVacuousAndMissingInputs(t *testing.T) {
	var buf bytes.Buffer
	if rc := run(options{root: t.TempDir(), bin: ""}, &buf); rc != 2 {
		t.Errorf("no binary: exit %d, want 2", rc)
	}
	empty := t.TempDir()
	if err := os.MkdirAll(filepath.Join(empty, "docs/docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	self, _ := os.Executable()
	if rc := run(options{root: empty, bin: self, baseline: "b.txt"}, &buf); rc != 2 {
		t.Errorf("no blocks in scope: exit %d, want 2", rc)
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

// readRows loads the -json output, indexed by line (fixture blocks only).
func readRows(t *testing.T, path string) map[int]jsonRow {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rows []jsonRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	out := map[int]jsonRow{}
	for _, r := range rows {
		if r.Line > 0 {
			out[r.Line] = r
		}
	}
	return out
}
