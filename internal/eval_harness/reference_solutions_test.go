package eval_harness

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// referenceSolution describes one benchmark/language pair.
type referenceSolution struct {
	bench    string
	lang     string
	file     string
	runner   func() LanguageRunner
	expected string
}

func jsRunner() LanguageRunner { return NewJSRunner() }
func goRunner() LanguageRunner { return NewGoRunner() }

// referenceSolutions lists all reference solution pairs.
// Expected output is trimmed on comparison.
var referenceSolutionsTable = []referenceSolution{
	{"fizzbuzz", "javascript", "main.js", jsRunner, "1\n2\nFizz\n4\nBuzz\nFizz\n7\n8\nFizz\nBuzz\n11\nFizz\n13\n14\nFizzBuzz"},
	{"fizzbuzz", "go", "main.go", goRunner, "1\n2\nFizz\n4\nBuzz\nFizz\n7\n8\nFizz\nBuzz\n11\nFizz\n13\n14\nFizzBuzz"},
	{"recursion_fibonacci", "javascript", "main.js", jsRunner, "6765"},
	{"recursion_fibonacci", "go", "main.go", goRunner, "6765"},
	{"graph_bfs", "javascript", "main.js", jsRunner, "1\n2\n3\n4\n5"},
	{"graph_bfs", "go", "main.go", goRunner, "1\n2\n3\n4\n5"},
	{"binary_tree_sum", "javascript", "main.js", jsRunner, "31"},
	{"binary_tree_sum", "go", "main.go", goRunner, "31"},
	{"balanced_parens", "javascript", "main.js", jsRunner, "true\nfalse\nfalse\ntrue"},
	{"balanced_parens", "go", "main.go", goRunner, "true\nfalse\nfalse\ntrue"},
	{"csv_to_json_converter", "javascript", "main.js", jsRunner, "Converted 3 valid rows to users.json"},
	{"csv_to_json_converter", "go", "main.go", goRunner, "Converted 3 valid rows to users.json"},
	{"expression_evaluator", "javascript", "main.js", jsRunner, "49"},
	{"expression_evaluator", "go", "main.go", goRunner, "49"},
	{"gcd_lcm", "javascript", "main.js", jsRunner, "6\n144"},
	{"gcd_lcm", "go", "main.go", goRunner, "6\n144"},
	{"fold_reduce", "javascript", "main.js", jsRunner, "Sum: 30\nProduct: 3840\nMax: 11"},
	{"fold_reduce", "go", "main.go", goRunner, "Sum: 30\nProduct: 3840\nMax: 11"},
	{"higher_order_functions", "javascript", "main.js", jsRunner, "Result: 14"},
	{"higher_order_functions", "go", "main.go", goRunner, "Result: 14"},
}

func TestReferenceSolutions_JS(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not available, skipping JS reference solution tests")
	}
	testReferenceSolutions(t, "javascript")
}

func TestReferenceSolutions_Go(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not available, skipping Go reference solution tests")
	}
	testReferenceSolutions(t, "go")
}

func testReferenceSolutions(t *testing.T, lang string) {
	t.Helper()
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Skipf("cannot find repo root: %v", err)
	}

	for _, rs := range referenceSolutionsTable {
		if rs.lang != lang {
			continue
		}
		rs := rs
		t.Run(rs.bench, func(t *testing.T) {
			srcPath := filepath.Join(repoRoot, "examples", "reference", rs.bench, rs.file)
			code, err := os.ReadFile(srcPath)
			if err != nil {
				t.Fatalf("reference solution not found: %s: %v", srcPath, err)
			}

			// Reference solutions are tiny programs; wall-clock is dominated
			// by interpreter startup (~slow on Windows CI runners — node alone
			// can take >20s cold). recursion_fibonacci was getting the only
			// 60s slot, but fizzbuzz hits the same 30s cliff on Windows. Give
			// every benchmark the same generous slot — the only thing the
			// shorter limit was buying was faster failure on a hang, and any
			// real hang would still time out well before 60s of useful work.
			timeout := 60 * time.Second

			runner := rs.runner()
			result, err := runner.Run(string(code), timeout)
			if err != nil {
				t.Fatalf("runner error: %v", err)
			}
			if !result.RuntimeOk {
				t.Errorf("expected RuntimeOk=true\nstdout: %s\nstderr: %s", result.Stdout, result.Stderr)
				return
			}

			got := strings.TrimSpace(result.Stdout)
			// For fizzbuzz, only verify the first 15 lines match (output is 100 lines).
			if rs.bench == "fizzbuzz" {
				lines := strings.Split(got, "\n")
				if len(lines) > 15 {
					got = strings.Join(lines[:15], "\n")
				}
			}
			if got != rs.expected {
				t.Errorf("output mismatch\ngot:  %q\nwant: %q", got, rs.expected)
			}
		})
	}
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
