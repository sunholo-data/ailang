package mission

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// rulingsFixture builds a throwaway registry root for the M-FLEET-RULINGS-LEDGER tests.
type rulingsFixture struct {
	t    *testing.T
	root string
	n    int
}

func newRulingsFixture(t *testing.T) *rulingsFixture {
	t.Helper()
	return &rulingsFixture{t: t, root: t.TempDir()}
}

// write creates rel under the fixture root and returns the absolute path.
func (f *rulingsFixture) write(rel, body string) string {
	f.t.Helper()
	abs := filepath.Join(f.root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		f.t.Fatal(err)
	}
	return abs
}

// mission writes missions/<name>.toml (distinct boot_offset per call so Load() sees no collision) and LoadFile()s it.
func (f *rulingsFixture) mission(name, repo, workdir, doc, extra string) *Mission {
	f.t.Helper()
	f.n++
	body := fmt.Sprintf("name = %q\nrepo = %q\nworkdir = %q\ndoc = %q\n%s\n"+
		"[schedule]\nmode = \"keepalive\"\nthrottle_seconds = 60\nboot_offset = %d\n",
		name, repo, workdir, doc, extra, f.n*60)
	path := f.write(filepath.Join("missions", name+".toml"), body)
	m, err := LoadFile(path)
	if err != nil {
		f.t.Fatalf("LoadFile(%s): %v", path, err)
	}
	return m
}

func TestRulingsRoot_Derivation(t *testing.T) {
	root, skipped := rulingsRoot(nil)
	if root != "" || skipped == nil || skipped.Detail != "registry has no missions" {
		t.Fatalf("nil missions: root=%q skipped=%+v", root, skipped)
	}
	if skipped.Mission != rulingsMission || skipped.Kind != "rulings-skipped" || skipped.Severity != Note {
		t.Fatalf("skipped finding shape wrong: %+v", skipped)
	}

	root, skipped = rulingsRoot([]*Mission{{Name: "a"}})
	if skipped == nil || skipped.Detail != "no mission carries a registry root" || root != "" {
		t.Fatalf("hand-built mission: root=%q skipped=%+v", root, skipped)
	}

	f1, f2 := newRulingsFixture(t), newRulingsFixture(t)
	m1 := f1.mission("a", sharedDriverRepo, filepath.Join(f1.root, "wd", "a"), "doc.md", "")
	m2 := f2.mission("b", sharedDriverRepo, filepath.Join(f2.root, "wd", "b"), "doc.md", "")
	root, skipped = rulingsRoot([]*Mission{m1, m2})
	if skipped == nil || root != "" {
		t.Fatalf("disagreeing roots must be skipped, got root=%q skipped=%+v", root, skipped)
	}
	if !strings.Contains(skipped.Detail, f1.root) || !strings.Contains(skipped.Detail, f2.root) {
		t.Fatalf("disagreement detail must name both roots: %q", skipped.Detail)
	}

	root, skipped = rulingsRoot([]*Mission{m1})
	if skipped != nil || root != m1.Root() {
		t.Fatalf("single mission: root=%q (want %q) skipped=%+v", root, m1.Root(), skipped)
	}
}

func TestRulingsCharterPath_Location(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "reg")
	wd := filepath.Join(string(filepath.Separator), "wd")

	inRepo := &Mission{Repo: sharedDriverRepo, Workdir: wd, Doc: "design_docs/x.md"}
	if got, want := charterPath(root, inRepo), filepath.Join(root, "design_docs/x.md"); got != want {
		t.Fatalf("in-repo charter: got %q want %q", got, want)
	}

	foreign := &Mission{Repo: "other/repo", Workdir: wd, Doc: "c.md"}
	if got, want := charterPath(root, foreign), filepath.Join(wd, "c.md"); got != want {
		t.Fatalf("foreign charter: got %q want %q", got, want)
	}

	abs := filepath.Join(string(filepath.Separator), "abs", "c.md")
	absolute := &Mission{Repo: sharedDriverRepo, Workdir: wd, Doc: abs}
	if got := charterPath(root, absolute); got != abs {
		t.Fatalf("absolute doc must be unchanged: got %q want %q", got, abs)
	}
}

func TestRulingsSkillFiles_WalkDirDepthAndFilter(t *testing.T) {
	f := newRulingsFixture(t)
	want := []string{
		f.write(".claude/skills/mission-a/SKILL.md", "x"),
		f.write(".claude/skills/mission-a/resources/g.md", "x"),
		f.write(".claude/skills/mission-a/resources/deep/x.md", "x"),
		f.write(".agents/skills/mission-a/resources/g.md", "x"),
	}
	f.write(".claude/skills/mission-a/notes.txt", "x")
	f.write(".claude/skills/other/SKILL.md", "x")
	sort.Strings(want)

	got, err := rulingSkillFiles(f.root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("skill files:\n got %q\nwant %q", got, want)
	}

	bare := newRulingsFixture(t)
	bare.write(".claude/skills/mission-a/SKILL.md", "x")
	if _, err := rulingSkillFiles(bare.root); err != nil {
		t.Fatalf("root without .agents/ must not error: %v", err)
	}
	empty := newRulingsFixture(t)
	if files, err := rulingSkillFiles(empty.root); err != nil || len(files) != 0 {
		t.Fatalf("root with no skills dirs: files=%q err=%v", files, err)
	}
}

func TestRulingsLiteralHits_LedgerExclusion(t *testing.T) {
	const lit = "retired-phrase-x"
	lines := []string{
		"# charter",
		"outside " + lit,
		"<!-- decision-ledger:start -->",
		"| ID | Status | Ruling | Evidence |",
		"| D-FR-1 | RESOLVED | " + lit + " | e |",
		"<!-- decision-ledger:end -->",
	}
	if got := literalHits(lines, lit, true); !reflect.DeepEqual(got, []int{2}) {
		t.Fatalf("charter=true: got %v want [2]", got)
	}
	if got := literalHits(lines, lit, false); !reflect.DeepEqual(got, []int{2, 5}) {
		t.Fatalf("charter=false: got %v want [2 5]", got)
	}
}

func TestRulingsLiteralHits_ByteExactCaseSensitive(t *testing.T) {
	lines := []string{
		"WORLD IS A FORK here",
		"generator≠judge guard (HARD",
		"retired-literal-x and retired-literal-x",
	}
	if got := literalHits(lines, "World is a fork", false); len(got) != 0 {
		t.Fatalf("case-folding matcher: got %v want none", got)
	}
	if got := literalHits(lines, "WORLD IS A FORK", false); !reflect.DeepEqual(got, []int{1}) {
		t.Fatalf("exact match: got %v want [1]", got)
	}
	if got := literalHits(lines, "generator≠judge guard (HARD", false); !reflect.DeepEqual(got, []int{2}) {
		t.Fatalf("exact bytes: got %v want [2]", got)
	}
	if got := literalHits(lines, "retired-literal-x", false); !reflect.DeepEqual(got, []int{3}) {
		t.Fatalf("per-line counting: got %v want [3] (one hit per line)", got)
	}
}

func TestRulingsDisplayPath(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "reg")
	in := filepath.Join(root, ".claude", "skills", "mission-a", "SKILL.md")
	if got, want := displayPath(root, in), filepath.Join(".claude", "skills", "mission-a", "SKILL.md"); got != want {
		t.Fatalf("under root: got %q want %q", got, want)
	}
	out := filepath.Join(string(filepath.Separator), "elsewhere", "x.md")
	if got := displayPath(root, out); got != out {
		t.Fatalf("outside root: got %q want %q", got, out)
	}
	if got := displayPath(root, "rel.md"); got != "rel.md" {
		t.Fatalf("relative unchanged: got %q", got)
	}
}
