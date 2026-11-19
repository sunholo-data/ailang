package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// ailangBin is the binary the fixtures run against. make test-check-doc-samples sets
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

func repoStd(t *testing.T) string {
	p, err := filepath.Abs("../../std")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// copyTree copies testdata/site into a temp dir.
func copyTree(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir("testdata/site", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel("testdata/site", p)
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return writeFile(dst, filepath.ToSlash(rel), string(raw))
	})
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func writeLists(t *testing.T, root, baseline, unchecked string) {
	t.Helper()
	if err := writeFile(root, "scripts/doc_samples_baseline.txt", baselineHeader+baseline); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(root, "scripts/doc_samples_unchecked.txt", uncheckedHeader+unchecked); err != nil {
		t.Fatal(err)
	}
}

func opts(t *testing.T, root string) options {
	return options{root: root, bin: ailangBin(t), baseline: "scripts/doc_samples_baseline.txt",
		unchecked: "scripts/doc_samples_unchecked.txt", stdlib: repoStd(t), jobs: 4}
}

// TestFixtureVerdicts is the control: one block per verdict, checked by the real
// binary, each asserted exactly — and the report names page:line, tag, code and repro.
func TestFixtureVerdicts(t *testing.T) {
	root := copyTree(t)
	bs := fixtureBlocks(t, root, fixturePage)
	writeLists(t, root,
		fixturePage+" "+bs["baselined-failure"].Key+" TC_UNDEFINED_VAR\n"+
			fixturePage+" "+bs["baselined-code-changed"].Key+" PAR015\n"+
			fixturePage+" "+bs["stale-baseline"].Key+" TC_UNDEFINED_VAR\n"+
			fixturePage+" 000000000000 PAR015\n",
		fixturePage+" "+bs["sketch"].Key+" sketch\n"+fixturePage+" "+bs["repl"].Key+" repl\n"+fixturePage+" 111111111111 sketch\n")
	var buf bytes.Buffer
	o := opts(t, root)
	o.jsonOut = filepath.Join(t.TempDir(), "out.json")
	if rc := run(o, &buf); rc != 1 {
		t.Fatalf("exit %d, want 1 (the fixture has deliberate failures)\n%s", rc, buf.String())
	}
	out := buf.String()
	rows := readRows(t, o.jsonOut)
	want := map[string]string{
		"pass-program": "ok", "pass-header": "ok synth", "main-needs-module": "FAIL MOD014",
		"fail-untagged": "FAIL TC_UNDEFINED_VAR", "fragment-parse": "ok", "error-fails-correctly": "ok",
		"error-wrongly-passes": "FAIL DOC_EXPECTED_FAIL", "error-wrong-code": "FAIL DOC_WRONG_ERROR",
		"error-prefix-only": "FAIL DOC_WRONG_ERROR", "error-second-code": "FAIL DOC_WRONG_ERROR",
		"error-pseudo-code": "FAIL DOC_BADTAG", "baselined-failure": "baselined",
		"baselined-code-changed": "FAIL TC_UNDEFINED_VAR", "stale-baseline": "STALE", "sketch": "skipped",
		"sketch-unlisted": "FAIL", "repl": "skipped", "file-lib": "ok", "file-user": "ok",
		"tilde-fence": "FAIL TC_UNDEFINED_VAR", "misfenced": "FAIL DOC_MISFENCED",
		"misfenced-untagged": "FAIL DOC_MISFENCED", "misfenced-parse-probe": "FAIL DOC_MISFENCED",
		"bad-tag": "FAIL DOC_BADTAG", "stray-fence": "FAIL DOC_FENCE_PAIRING", "unterminated": "FAIL DOC_UNTERMINATED",
		"probe-untagged": "FAIL DOC_MISFENCED", "hint-match": "FAIL DOC_MISFENCED", "hint-adt-arm": "FAIL DOC_MISFENCED",
	}
	for m, w := range want {
		r, ok := rows[fixturePage+":"+strconv.Itoa(bs[m].Line)]
		if !ok {
			t.Errorf("%s (line %d): no verdict", m, bs[m].Line)
			continue
		}
		if got := describe(r, w); got != w {
			t.Errorf("%s (line %d): %q, want %q (%s %s)", m, bs[m].Line, got, w, r.FirstErr, r.Why)
		}
	}
	others := map[string]string{
		"docs/docs/reference/imports.mdx:8":  "FAIL DOC_MISSING_IMPORT",
		"docs/docs/reference/imports.mdx:14": "FAIL DOC_MISFENCED",
		"docs/docs/reference/imports.mdx:16": "FAIL TC_UNDEFINED_VAR", // a block under reference/ is scanned
		"docs/src/pages/index.jsx:5":         "FAIL TC_UNDEFINED_VAR", // AILANG in a JSX literal
		"examples/fixture/bad.ail:1":         "FAIL TC_UNDEFINED_VAR", // an imported file
		"examples/fixture/good.ail:1":        "ok",
	}
	for k, w := range others {
		if r, ok := rows[k]; !ok {
			t.Errorf("%s: no verdict", k)
		} else if got := describe(r, w); got != w {
			t.Errorf("%s: %q, want %q (%s)", k, got, w, r.FirstErr)
		}
	}
	if r := rows[fixturePage+":"+strconv.Itoa(bs["sketch-unlisted"].Line)]; !strings.Contains(r.Why, "not in the unchecked list") {
		t.Errorf("an unlisted sketch must say so: %q", r.Why)
	}
	if len(rows) != len(want)+len(others) {
		t.Errorf("%d block verdicts, want %d (output and non-AILANG blocks must be ignored)", len(rows), len(want)+len(others))
	}
	line := strconv.Itoa(bs["fail-untagged"].Line)
	for _, frag := range []string{
		"FAIL      docs/docs/fixture.md:" + line + " [ailang] TC_UNDEFINED_VAR: ",
		"repro: make check-doc-samples DOC_SAMPLES_ARGS='-only docs/docs/fixture.md:" + line,
		"ORPHAN    docs/docs/fixture.md 000000000000", "ORPHAN    docs/docs/fixture.md 111111111111",
		"baselined as PAR015 but now fails with TC_UNDEFINED_VAR",
		"only with a synthesised module line 1",
	} {
		if !strings.Contains(out, frag) {
			t.Errorf("report lacks %q:\n%s", frag, out)
		}
	}
}

func describe(r jsonRow, want string) string {
	got := r.Verdict
	if strings.HasPrefix(want, "FAIL ") {
		got += " " + r.Code
	}
	if strings.HasSuffix(want, " synth") && r.Synth {
		got += " synth"
	}
	return got
}

// cleanPage is a page with no structural defect: a passing block, two failing ones and
// a sketch — the shape the ratchet tests mutate.
const cleanPage = "---\ntitle: t\n---\n\n" +
	"```ailang\n-- marker: pass\nexport func ok() -> int = 1\n```\n\n" +
	"```ailang\n-- marker: fail-one\nexport func f1() -> int = missingOne\n```\n\n" +
	"```ailang\n-- marker: fail-two\nexport func f2() -> int = missingTwo\n```\n\n" +
	"```ailang sketch\n-- marker: sketch\nfunc sig(x: int) -> int\n```\n"

func cleanRepo(t *testing.T) (string, options) {
	root := t.TempDir()
	if err := writeFile(root, fixturePage, cleanPage); err != nil {
		t.Fatal(err)
	}
	git(t, root, "init", "-q")
	o := opts(t, root)
	o.write = true
	var buf bytes.Buffer
	if rc := run(o, &buf); rc != 0 {
		t.Fatalf("seed: exit %d\n%s", rc, buf.String())
	}
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "seed")
	o.write = false
	if rc := run(o, &buf); rc != 0 {
		t.Fatalf("a seeded tree must be green: exit %d\n%s", rc, buf.String())
	}
	return root, o
}

func git(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=t", "-c", "user.email=t@example.invalid"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func edit(t *testing.T, root, from, to string) {
	t.Helper()
	p := filepath.Join(root, fixturePage)
	raw, _ := os.ReadFile(p)
	if !strings.Contains(string(raw), from) {
		t.Fatalf("fixture lacks %q", from)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(string(raw), from, to, 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// rewriteAndRatchet re-baselines (what an author silencing a failure would do) and then
// runs the gate against the seed commit, as CI does.
func rewriteAndRatchet(t *testing.T, root string, o options) (int, string) {
	t.Helper()
	var buf bytes.Buffer
	o.write = true
	run(o, &buf)
	o.write, o.baseRef = false, "HEAD"
	buf.Reset()
	rc := run(o, &buf)
	return rc, buf.String()
}

// The reviewer's two simulations: editing a baselined block and re-baselining, and
// breaking a passing block while deleting a different failing one (count unchanged).
func TestRatchetIsASetRule(t *testing.T) {
	t.Run("edit a baselined block", func(t *testing.T) {
		root, o := cleanRepo(t)
		edit(t, root, "-- marker: fail-one", "-- marker: fail-one (comment edited)")
		if rc, out := rewriteAndRatchet(t, root, o); rc != 1 || !strings.Contains(out, "FAIL      shrink-only") {
			t.Fatalf("exit %d, want 1 with a shrink-only failure\n%s", rc, out)
		}
	})
	t.Run("swap: break a passing block, delete another failing one", func(t *testing.T) {
		root, o := cleanRepo(t)
		edit(t, root, "export func ok() -> int = 1", "export func ok() -> int = nowBroken")
		edit(t, root, "```ailang\n-- marker: fail-two\nexport func f2() -> int = missingTwo\n```\n", "")
		rc, out := rewriteAndRatchet(t, root, o)
		if rc != 1 || !strings.Contains(out, "shrink-only: scripts/doc_samples_baseline.txt 2 entries (HEAD: 2), 1 new") {
			t.Fatalf("exit %d, want 1: same count, one new key\n%s", rc, out)
		}
	})
	t.Run("new sketch needs a reviewed marker", func(t *testing.T) {
		root, o := cleanRepo(t)
		edit(t, root, "```ailang\n-- marker: fail-one", "```ailang sketch\n-- marker: fail-one")
		rc, out := rewriteAndRatchet(t, root, o)
		if rc != 1 || !strings.Contains(out, "needs `# reviewed:") {
			t.Fatalf("tagging a failing block sketch: exit %d, want 1\n%s", rc, out)
		}
		p := filepath.Join(root, o.unchecked)
		raw, _ := os.ReadFile(p)
		marked := strings.Replace(string(raw), " sketch\n", " sketch # reviewed: a type signature with no body\n", -1)
		if err := os.WriteFile(p, []byte(marked), 0o644); err != nil {
			t.Fatal(err)
		}
		o.baseRef = "HEAD"
		var buf bytes.Buffer
		if rc := run(o, &buf); rc != 0 {
			t.Fatalf("with a reviewed marker: exit %d, want 0\n%s", rc, buf.String())
		}
	})
	t.Run("unresolvable base fails closed", func(t *testing.T) {
		root, o := cleanRepo(t)
		_ = root
		o.baseRef = "0000000000000000000000000000000000000001"
		var buf bytes.Buffer
		if rc := run(o, &buf); rc != 2 {
			t.Fatalf("exit %d, want 2\n%s", rc, buf.String())
		}
	})
}

func TestStructuralExemptionIsNamedAndExpires(t *testing.T) {
	root := t.TempDir()
	page := "```\nstray\n```ailang\nexport func x() -> int = 1\n```\n\n```ailang\nexport func ok() -> int = 1\n```\n"
	if err := writeFile(root, fixturePage, page); err != nil {
		t.Fatal(err)
	}
	stray := ExtractBlocks(fixturePage, fixturePage, page)[0]
	writeLists(t, root, "", "")
	saved := structuralExemptions
	defer func() { structuralExemptions = saved }()
	var buf bytes.Buffer
	structuralExemptions = map[string]string{}
	if rc := run(opts(t, root), &buf); rc != 1 {
		t.Fatalf("an unexempted stray fence: exit %d, want 1\n%s", rc, buf.String())
	}
	structuralExemptions = map[string]string{fixturePage + " " + stray.Key: "test exemption"}
	buf.Reset()
	if rc := run(opts(t, root), &buf); rc != 0 || !strings.Contains(buf.String(), "EXEMPT") {
		t.Fatalf("exempted: exit %d, want 0 with EXEMPT\n%s", rc, buf.String())
	}
	structuralExemptions = map[string]string{fixturePage + " 0123456789ab": "fixed long ago"}
	if err := writeFile(root, fixturePage, "```ailang\nexport func ok() -> int = 1\n```\n"); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if rc := run(opts(t, root), &buf); rc != 1 || !strings.Contains(buf.String(), "matches no defect any more") {
		t.Fatalf("a stale exemption on an in-scope file: exit %d, want 1\n%s", rc, buf.String())
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

// readRows loads the -json output, indexed by source:line.
func readRows(t *testing.T, path string) map[string]jsonRow {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var rows []jsonRow
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	out := map[string]jsonRow{}
	for _, r := range rows {
		if r.Line > 0 {
			out[r.Source+":"+strconv.Itoa(r.Line)] = r
		}
	}
	return out
}
