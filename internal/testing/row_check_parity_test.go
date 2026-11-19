package testing

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/pipeline"
)

// #2232 review: an inline row the harness runs successfully must never be a
// TC_TEST_001 from check. Both build the call with ast.RowInput/RowCallArgs;
// these tests hold the two paths to it, per row shape and over every tracked
// file with rows.

type rowParity struct {
	harness  map[string]TestStatus // "<fn>_test_<n>" -> harness verdict
	checkErr error
}

func runBothPaths(t *testing.T, path string) rowParity {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	p := parser.New(lexer.New(string(src), path))
	file := p.ParseFile()
	if len(p.Errors()) > 0 {
		return rowParity{} // not a row-check question
	}
	res, err := RunTestsFromFile(path, file)
	if err != nil {
		t.Fatalf("%s: RunTestsFromFile: %v", path, err)
	}
	out := rowParity{harness: map[string]TestStatus{}}
	for _, r := range res.Tests {
		out.harness[r.Name] = r.Status
	}
	_, out.checkErr = pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: true, RelaxModules: true},
		pipeline.Source{Filename: path})
	return out
}

var rowNameRe = regexp.MustCompile(`^(\S+) row (\d+)$`)

// rowCheckFailure returns the harness name of the row check rejected, if the
// check error is a TC_TEST_001 for a row.
func rowCheckFailure(err error) (string, bool) {
	rep, ok := ailerrors.AsReport(err)
	if !ok || rep.Code != "TC_TEST_001" || rep.Data["kind"] != "test row" {
		return "", false
	}
	name, _ := rep.Data["name"].(string)
	m := rowNameRe.FindStringSubmatch(name)
	if m == nil {
		return "", false
	}
	return m[1] + "_test_" + m[2], true
}

func TestRowCheckParity_Shapes(t *testing.T) {
	cases := []struct {
		name, src string
		wantPass  bool // harness and check agree on this verdict
	}{
		{"two_args", "module m\nexport pure func add2(a: int, b: int) -> int\n  tests [((1, 2), 3)]\n{ a + b }\n", true},
		{"two_args_extra_parens", "module m\nexport pure func add2(a: int, b: int) -> int\n  tests [(((1, 2)), 3)]\n{ a + b }\n", true},
		{"three_args", "module m\nexport pure func add3(a: int, b: int, c: int) -> int\n  tests [((1, 2, 3), 6)]\n{ a + b + c }\n", true},
		{"three_args_extra_parens", "module m\nexport pure func add3(a: int, b: int, c: int) -> int\n  tests [(((1, 2, 3)), 6)]\n{ a + b + c }\n", true},
		{"one_arg", "module m\nexport pure func inc(a: int) -> int\n  tests [(1, 2), ((4), 5)]\n{ a + 1 }\n", true},
		// The harness spreads a tuple input over the parameters, so a single
		// tuple parameter cannot be called from a row: both paths fail it.
		{"tuple_param", "module m\nexport pure func psum(p: (int, int)) -> int\n  tests [((1, 2), 3)]\n{ match p { (a, b) => a + b } }\n", false},
		{"tuple_param_extra_parens", "module m\nexport pure func psum(p: (int, int)) -> int\n  tests [(((1, 2)), 3)]\n{ match p { (a, b) => a + b } }\n", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "m.ail")
			if err := os.WriteFile(path, []byte(c.src), 0o644); err != nil {
				t.Fatal(err)
			}
			got := runBothPaths(t, path)
			harnessPass := len(got.harness) > 0
			for _, st := range got.harness {
				harnessPass = harnessPass && st == StatusPass
			}
			if harnessPass != c.wantPass || (got.checkErr == nil) != c.wantPass {
				t.Fatalf("harness pass=%v (%v), check error=%v; want both pass=%v", harnessPass, got.harness, got.checkErr, c.wantPass)
			}
		})
	}
}

// Every tracked file with inline rows: a row the harness passes is never the
// row check reports.
func TestRowCheckParity_RepoFiles(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var files []string
	for _, dir := range []string{"examples", "internal/testing/testdata", "tests"} {
		_ = filepath.Walk(filepath.Join(root, dir), func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(p, ".ail") || strings.Contains(p, "/archive/") {
				return nil
			}
			if b, rerr := os.ReadFile(p); rerr == nil && strings.Contains(string(b), "tests [") {
				files = append(files, p)
			}
			return nil
		})
	}
	if len(files) < 20 {
		t.Fatalf("found only %d row-bearing files; the walk is not seeing the repo", len(files))
	}
	t.Chdir(root)
	for _, f := range files {
		rel, _ := filepath.Rel(root, f)
		got := runBothPaths(t, rel)
		if row, ok := rowCheckFailure(got.checkErr); ok && got.harness[row] == StatusPass {
			t.Errorf("%s: the harness passes %s but check rejects it: %v", rel, row, got.checkErr)
		}
	}
}
