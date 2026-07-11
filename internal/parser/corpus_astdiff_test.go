package parser

// M-SYNTAX-AI-FORGIVING corpus AST-diff fuzz gate.
//
// The hand-picked fixtures in syntax_ai_forgiving_test.go prove the new forms are
// ACCEPTED. This gate proves the change is BACKWARD-COMPATIBLE: for every currently
// valid .ail file in the corpus (benchmarks/**/*.ail + examples/**/*.ail), the NEW
// parser must produce a byte-identical AST to the OLD (pre-change) parser. A single
// meaning-diff on a currently-valid program fails the gate.
//
// Precedent (M-TAINT): a 2-token lookahead that passed hand-picked tests still
// mis-parsed ~14 real programs. The corpus diff — not the fixtures — is the merge gate.
//
// Mechanism: build the astdump command (cmd/astdump) from BOTH a git worktree at the
// pre-change base commit (the "old" parser) and the current worktree (the "new"
// parser), run both over every corpus file, and diff their output. Files the OLD
// parser rejected are skipped (they are not "currently valid"); the new parser is
// allowed to newly ACCEPT them — that is the whole point of the feature.
//
// This test is heavy (it compiles the toolchain twice and parses ~400 files). It is
// gated behind AILANG_CORPUS_FUZZ=1 so it does not run on every `go test` invocation,
// but IS the explicit M2 (R1) and M4 (R2) merge gate.

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// corpusBaseEnv names the git ref of the pre-change parser. The sprint plan pins
// the base at the sprint-plan commit; override for local runs if the base moves.
const corpusBaseEnv = "AILANG_CORPUS_BASE"

func TestCorpusASTDiff(t *testing.T) {
	if os.Getenv("AILANG_CORPUS_FUZZ") != "1" {
		t.Skip("set AILANG_CORPUS_FUZZ=1 to run the corpus AST-diff fuzz gate")
	}

	base := os.Getenv(corpusBaseEnv)
	if base == "" {
		base = "a7bd8257c" // sprint-plan commit (pre M-SYNTAX-AI-FORGIVING)
	}

	repoRoot := repoRootDir(t)

	// Build the NEW-parser dumper from the current tree.
	newBin := buildAstdump(t, repoRoot, "new")

	// Materialise the OLD tree in a temp git worktree and build the OLD dumper.
	oldRoot := t.TempDir()
	gitRun(t, repoRoot, "worktree", "add", "--detach", oldRoot, base)
	t.Cleanup(func() {
		// Best-effort removal; ignore errors so a leftover worktree never fails CI.
		_ = exec.Command("git", "-C", repoRoot, "worktree", "remove", "--force", oldRoot).Run()
	})
	// The base commit predates cmd/astdump — copy it in so both trees share the dumper.
	copyAstdumpInto(t, repoRoot, oldRoot)
	oldBin := buildAstdump(t, oldRoot, "old")

	// Collect the corpus.
	files := collectCorpus(t, repoRoot)
	if len(files) == 0 {
		t.Fatal("empty corpus — glob is wrong")
	}
	t.Logf("corpus: %d files", len(files))

	var (
		checked   int
		newlyOK   int
		stillFail int
	)
	for _, f := range files {
		oldOut := runDump(t, oldBin, f)
		newOut := runDump(t, newBin, f)

		oldValid := !strings.HasPrefix(oldOut, "ERRORS")
		newValid := !strings.HasPrefix(newOut, "ERRORS")

		if !oldValid {
			// Not a currently-valid program. The feature is allowed to newly accept
			// it; we don't gate on files the old parser already rejected.
			if newValid {
				newlyOK++
			} else {
				stillFail++
			}
			continue
		}

		checked++
		if oldOut != newOut {
			rel, _ := filepath.Rel(repoRoot, f)
			t.Errorf("AST DIFF on currently-valid file %s\n%s", rel, firstDiff(oldOut, newOut))
		}
	}

	t.Logf("checked=%d identical; newly-accepted=%d; still-invalid=%d",
		checked, newlyOK, stillFail)
}

// --- helpers -----------------------------------------------------------------

func repoRootDir(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func buildAstdump(t *testing.T, root, tag string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "astdump_"+tag)
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/astdump")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build astdump (%s) in %s: %v\n%s", tag, root, err, out)
	}
	return bin
}

// copyAstdumpInto copies cmd/astdump/main.go from the new tree into the old tree so
// the old (pre-change) source can be compiled with the same dumper entrypoint.
func copyAstdumpInto(t *testing.T, srcRoot, dstRoot string) {
	t.Helper()
	src := filepath.Join(srcRoot, "cmd", "astdump", "main.go")
	dstDir := filepath.Join(dstRoot, "cmd", "astdump")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dstDir, err)
	}
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(filepath.Join(dstDir, "main.go"), data, 0o644); err != nil {
		t.Fatalf("write astdump into old tree: %v", err)
	}
}

func collectCorpus(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	for _, dir := range []string{"benchmarks", "examples"} {
		err := filepath.Walk(filepath.Join(root, dir), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".ail") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	return files
}

func runDump(t *testing.T, bin, file string) string {
	t.Helper()
	out, err := exec.Command(bin, file).Output()
	if err != nil {
		// A non-zero exit means the dumper itself failed (bad usage / read error),
		// not a parse error (those are reported on stdout with exit 0).
		t.Fatalf("astdump %s: %v", file, err)
	}
	return string(out)
}

// firstDiff returns a short context around the first differing line, to keep
// failure output actionable rather than dumping two full ASTs.
func firstDiff(a, b string) string {
	al := strings.Split(a, "\n")
	bl := strings.Split(b, "\n")
	n := len(al)
	if len(bl) < n {
		n = len(bl)
	}
	for i := 0; i < n; i++ {
		if al[i] != bl[i] {
			start := i - 2
			if start < 0 {
				start = 0
			}
			var sb strings.Builder
			sb.WriteString("first diff at line ")
			sb.WriteString(itoa(i + 1))
			sb.WriteString(":\n")
			for j := start; j <= i; j++ {
				sb.WriteString("  old: " + al[j] + "\n")
			}
			sb.WriteString("  ---\n")
			for j := start; j <= i; j++ {
				sb.WriteString("  new: " + bl[j] + "\n")
			}
			return sb.String()
		}
	}
	if len(al) != len(bl) {
		return "outputs differ in length (old=" + itoa(len(al)) + " new=" + itoa(len(bl)) + " lines)"
	}
	return "(identical prefix; trailing whitespace differs)"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
