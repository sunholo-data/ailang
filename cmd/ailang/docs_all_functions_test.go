package main

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// stdlibDirForTest returns the repo-root std/ directory. The in-process docs
// helpers resolve stdlib relative to CWD (./std or ../std) which does not match
// the test CWD (cmd/ailang), so tests pass the absolute std path directly.
func stdlibDirForTest(t *testing.T) string {
	t.Helper()
	root := findRepoRootForTest(t)
	std := filepath.Join(root, "std")
	if _, err := os.Stat(filepath.Join(std, "io.ail")); err != nil {
		t.Fatalf("stdlib not found at %s: %v", std, err)
	}
	return std
}

// scanExportNames independently scans a std/*.ail file for `export [pure] func <name>`
// declarations, returning names in file order. This is deliberately a SEPARATE
// implementation from parseExportSignatures (which uses the AST) so the name-set
// equality test cross-checks two independent extractors.
var scanExportRe = regexp.MustCompile(`^export\s+(?:pure\s+)?func\s+(\w+)`)

func scanExportNames(t *testing.T, filePath string) []string {
	t.Helper()
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("open %s: %v", filePath, err)
	}
	defer f.Close()
	var names []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if m := scanExportRe.FindStringSubmatch(line); m != nil {
			names = append(names, m[1])
		}
	}
	return names
}

// allStdlibFiles returns the sorted list of std/*.ail file paths.
func allStdlibFiles(t *testing.T, stdDir string) []string {
	t.Helper()
	entries, err := os.ReadDir(stdDir)
	if err != nil {
		t.Fatalf("readdir %s: %v", stdDir, err)
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ail") {
			continue
		}
		files = append(files, filepath.Join(stdDir, e.Name()))
	}
	sort.Strings(files)
	return files
}

// TestAllFunctions_ExactNameSet asserts the --all-functions emitted export name
// set EQUALS an independent scan of std/*.ail — BOTH directions (missing OR extra
// fails). Prelude entries are excluded from this stdlib-only comparison.
func TestAllFunctions_ExactNameSet(t *testing.T) {
	stdDir := stdlibDirForTest(t)

	// Independent scan: module.name for every export.
	want := map[string]bool{}
	for _, fp := range allStdlibFiles(t, stdDir) {
		// derive module name from `module std/x` line
		mod := moduleNameOf(t, fp)
		for _, n := range scanExportNames(t, fp) {
			want[mod+"."+n] = true
		}
	}

	// Emitted set (strip signature/doc; drop prelude.* pseudo-module).
	got := map[string]bool{}
	for _, line := range buildAllFunctionsLines(stdDir) {
		key := line[:strings.Index(line, ":")]
		if strings.HasPrefix(key, "prelude.") {
			continue
		}
		got[key] = true
	}

	for k := range want {
		if !got[k] {
			t.Errorf("export %q present in independent scan but MISSING from --all-functions", k)
		}
	}
	for k := range got {
		if !want[k] {
			t.Errorf("export %q emitted by --all-functions but ABSENT from independent scan", k)
		}
	}
}

func moduleNameOf(t *testing.T, filePath string) string {
	t.Helper()
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatalf("open %s: %v", filePath, err)
	}
	defer f.Close()
	re := regexp.MustCompile(`^module\s+(std/\w+)`)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := re.FindStringSubmatch(strings.TrimSpace(sc.Text())); m != nil {
			return m[1]
		}
	}
	t.Fatalf("no module declaration in %s", filePath)
	return ""
}

// TestAllFunctions_EverySignatureNonEmpty asserts every emitted line carries a
// non-empty signature (not `[signature unparsed]`, not blank).
func TestAllFunctions_EverySignatureNonEmpty(t *testing.T) {
	stdDir := stdlibDirForTest(t)
	for _, line := range buildAllFunctionsLines(stdDir) {
		colon := strings.Index(line, ": ")
		if colon < 0 {
			t.Errorf("line has no signature separator: %q", line)
			continue
		}
		rest := line[colon+2:]
		// strip the doc suffix
		if i := strings.Index(rest, " -- "); i >= 0 {
			rest = rest[:i]
		}
		sig := strings.TrimSpace(rest)
		if sig == "" || strings.Contains(sig, "[signature unparsed]") {
			t.Errorf("empty/unparsed signature for line: %q", line)
		}
	}
}

// TestAllFunctions_SignatureFidelity re-parses every std file independently in
// the test and asserts the emitted signature EQUALS the in-test AST rendering
// (corpus-wide fidelity, not a golden subset).
func TestAllFunctions_SignatureFidelity(t *testing.T) {
	stdDir := stdlibDirForTest(t)

	// Build emitted map: module.name -> signature (without leading name/doc).
	emitted := map[string]string{}
	for _, line := range buildAllFunctionsLines(stdDir) {
		key := line[:strings.Index(line, ":")]
		if strings.HasPrefix(key, "prelude.") {
			continue
		}
		rest := line[strings.Index(line, ": ")+2:]
		if i := strings.Index(rest, " -- "); i >= 0 {
			rest = rest[:i]
		}
		emitted[key] = strings.TrimSpace(rest)
	}

	for _, fp := range allStdlibFiles(t, stdDir) {
		mod := moduleNameOf(t, fp)
		src, err := os.ReadFile(fp)
		if err != nil {
			t.Fatalf("read %s: %v", fp, err)
		}
		l := lexer.New(string(src), fp)
		p := parser.New(l)
		prog := p.Parse()
		if errs := p.Errors(); len(errs) > 0 {
			t.Fatalf("independent parse of %s failed: %v", fp, errs[0])
		}
		for _, fn := range prog.File.Funcs {
			if fn == nil || !fn.IsExport {
				continue
			}
			// The emitted line drops the leading name (formatFunctionLine), so
			// compare against the rendered signature minus its leading name.
			fullSig := renderFuncSignature(fn)
			wantSig := strings.TrimSpace(strings.TrimPrefix(fullSig, fn.Name))
			key := mod + "." + fn.Name
			if emitted[key] != wantSig {
				t.Errorf("signature mismatch for %s:\n  emitted: %q\n  re-parse: %q", key, emitted[key], wantSig)
			}
		}
	}
}

// TestAllFunctions_V16EffectRow asserts the V16 truncation is fixed: std/clock.now
// renders the full `! {Clock}` effect row in --all-functions.
func TestAllFunctions_V16EffectRow(t *testing.T) {
	stdDir := stdlibDirForTest(t)
	var nowLine string
	for _, line := range buildAllFunctionsLines(stdDir) {
		if strings.HasPrefix(line, "std/clock.now:") {
			nowLine = line
			break
		}
	}
	if nowLine == "" {
		t.Fatal("std/clock.now not found in --all-functions output")
	}
	if !strings.Contains(nowLine, "! {Clock}") {
		t.Errorf("V16 regression: std/clock.now missing full effect row, got: %q", nowLine)
	}
}

// TestPerModuleDocs_V16EffectRow asserts the V16 fix also applies to per-module
// `ailang docs std/clock` output (rendered from the AST, not the truncating regex).
func TestPerModuleDocs_V16EffectRow(t *testing.T) {
	stdDir := stdlibDirForTest(t)
	sigs, _, err := parseExportSignatures(filepath.Join(stdDir, "clock.ail"))
	if err != nil {
		t.Fatalf("parseExportSignatures(clock): %v", err)
	}
	now, ok := sigs["now"]
	if !ok {
		t.Fatal("clock.now not found in parsed signatures")
	}
	if !strings.Contains(now, "! {Clock}") {
		t.Errorf("V16 regression in per-module render: %q", now)
	}
}

// TestAllFunctions_Filter asserts the positional filter keeps matching lines and
// drops unrelated modules (case-insensitive substring over the full line).
func TestAllFunctions_Filter(t *testing.T) {
	stdDir := stdlibDirForTest(t)
	lines := buildAllFunctionsLines(stdDir)

	// Apply the same filter allFunctionsCommand applies, in-process.
	filter := "timestamp"
	var kept []string
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), filter) {
			kept = append(kept, line)
		}
	}
	if len(kept) == 0 {
		t.Fatal("filter 'timestamp' matched nothing")
	}
	// Every kept line must actually contain the filter substring.
	for _, line := range kept {
		if !strings.Contains(strings.ToLower(line), filter) {
			t.Errorf("filter leaked non-matching line: %q", line)
		}
	}
	// A clearly unrelated module (std/ai callResult) must be excluded.
	for _, line := range kept {
		if strings.HasPrefix(line, "std/ai.call") {
			t.Errorf("filter 'timestamp' should not include std/ai.call lines: %q", line)
		}
	}
}

// TestRenderFuncSignature_TrickyForms locks the AST rendering of tricky shapes:
// generics, effect rows, multi-effect, pure (no effect row), zero/unit return.
func TestRenderFuncSignature_TrickyForms(t *testing.T) {
	src := `module std/trickytest

export pure func idfn[a](x: a) -> a { x }

export func doIO(msg: string) -> () ! {IO} { () }

export func multi(x: int) -> int ! {IO, FS} { x }

export pure func add(a: int, b: int) -> int { a + b }
`
	l := lexer.New(src, "trickytest.ail")
	p := parser.New(l)
	prog := p.Parse()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("parse tricky forms: %v", errs[0])
	}
	got := map[string]string{}
	for _, fn := range prog.File.Funcs {
		if fn == nil || !fn.IsExport {
			continue
		}
		got[fn.Name] = renderFuncSignature(fn)
	}

	checks := map[string]string{
		"idfn":  "idfn[a](a) -> a",
		"doIO":  "doIO(string) -> () ! {IO}",
		"multi": "multi(int) -> int ! {IO, FS}",
		"add":   "add(int, int) -> int",
	}
	for name, want := range checks {
		if got[name] != want {
			t.Errorf("renderFuncSignature(%s) = %q, want %q", name, got[name], want)
		}
	}
}

// TestParseExportSignatures_UnparseableFileErrors asserts a deliberately broken
// .ail file makes parseExportSignatures return an error naming the file (the
// caller then fails loudly, non-zero) — never a silent drop.
func TestParseExportSignatures_UnparseableFileErrors(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "broken.ail")
	if err := os.WriteFile(bad, []byte("module std/broken\nexport func f( -> {{{ !!!\n"), 0o644); err != nil {
		t.Fatalf("write bad file: %v", err)
	}
	_, _, err := parseExportSignatures(bad)
	if err == nil {
		t.Fatal("expected parse error for broken .ail, got nil")
	}
	if !strings.Contains(err.Error(), bad) {
		t.Errorf("error should name the file %q, got: %v", bad, err)
	}
}

// TestAllFunctions_UnparseableFileExitsNonZero drives the actual binary against a
// stdlib dir containing one broken file and asserts a non-zero exit that names
// the file on stderr (no silent partial render).
func TestAllFunctions_UnparseableFileExitsNonZero(t *testing.T) {
	bin := testutil.FindAilangBinary(t)
	realStd := stdlibDirForTest(t)

	// Build a temp stdlib dir: copy io.ail (so isStdlibDir passes) + a broken file.
	tmpStd := t.TempDir()
	copyFile(t, filepath.Join(realStd, "io.ail"), filepath.Join(tmpStd, "io.ail"))
	if err := os.WriteFile(filepath.Join(tmpStd, "broken.ail"),
		[]byte("module std/broken\nexport func f( -> {{{ !!!\n"), 0o644); err != nil {
		t.Fatalf("write broken: %v", err)
	}

	cmd := exec.Command(bin, "docs", "--all-functions")
	cmd.Env = append(os.Environ(), "AILANG_STDLIB_PATH="+tmpStd)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit for broken stdlib file, got success:\n%s", out)
	}
	if !strings.Contains(string(out), "broken.ail") {
		t.Errorf("expected broken.ail named on stderr, got:\n%s", out)
	}
}
