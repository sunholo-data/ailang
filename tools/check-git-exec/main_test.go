package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, name, src string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestEnumerator_AdditionAndBlindspots(t *testing.T) {
	d := t.TempDir()
	write(t, d, "cmd/base.go", "package p\nimport \"os/exec\"\nfunc f(){ exec.Command(\"git\", \"status\") }\n")
	fs, _, err := enumerate(d, []string{"cmd"})
	if err != nil || len(fs) != 1 {
		t.Fatalf("base = %v, %v", fs, err)
	}
	write(t, d, "cmd/multi.go", "package p\nimport \"os/exec\"\nfunc f(){ exec.Command(\n\"git\", \"status\") }\n")
	fs, _, err = enumerate(d, []string{"cmd"})
	if err != nil || len(fs) != 2 {
		t.Fatalf("addition count = %d, %v", len(fs), err)
	}
	write(t, d, "cmd/blind.go", "package p\nimport \"os/exec\"\nfunc f(){ g:=\"git\"; exec.Command(g); exec.Command(\"bash\",\"-c\",\"git status\") }\n")
	fs, _, err = enumerate(d, []string{"cmd"})
	if err != nil || len(fs) != 2 {
		t.Fatalf("blindspots added findings: %d, %v", len(fs), err)
	}
}

func TestEnumerator_CommittedFixtures(t *testing.T) {
	d := t.TempDir()
	for _, tc := range []struct {
		name string
		want int
	}{
		{"git_exec_gate_positive.txt", 2},
		{"git_exec_gate_blindspot.txt", 0},
	} {
		data, err := os.ReadFile(filepath.Join("..", "..", "scripts", "testdata", tc.name))
		if err != nil {
			t.Fatal(err)
		}
		write(t, d, filepath.Join("cmd", tc.name+".go"), string(data))
		fs, _, err := enumerate(d, []string{"cmd"})
		if err != nil {
			t.Fatal(err)
		}
		if len(fs) != tc.want {
			t.Fatalf("%s findings = %d, want %d", tc.name, len(fs), tc.want)
		}
		if err := os.RemoveAll(filepath.Join(d, "cmd")); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEnumerator_ExactNamesAndArgumentPositions(t *testing.T) {
	d := t.TempDir()
	write(t, d, "cmd/x.go", "package p\nimport (\"context\"; \"os/exec\")\nfunc f(ctx context.Context){ exec.Command(\"git-lfs\"); exec.Command(\"gitfoo\"); exec.Command(\"echo\",\"git\"); exec.CommandContext(ctx,\"git\"); exec.CommandContext(ctx,\"echo\",\"git\") }\n")
	fs, _, err := enumerate(d, []string{"cmd"})
	if err != nil || len(fs) != 1 {
		t.Fatalf("findings = %v, %v", fs, err)
	}
}

func TestEnumerator_ExcludesTests(t *testing.T) {
	d := t.TempDir()
	src := "package p\nimport \"os/exec\"\nfunc f(){exec.Command(\"git\")}\n"
	write(t, d, "cmd/x_test.go", src)
	fs, _, err := enumerate(d, []string{"cmd"})
	if err != nil || len(fs) != 0 {
		t.Fatalf("test file = %v, %v", fs, err)
	}
	write(t, d, "cmd/x.go", src)
	fs, _, err = enumerate(d, []string{"cmd"})
	if err != nil || len(fs) != 1 {
		t.Fatalf("go file = %v, %v", fs, err)
	}
}

func TestEnumerator_ParseErrorNamesFile(t *testing.T) {
	d := t.TempDir()
	write(t, d, "cmd/bad.go", "package p\nfunc (")
	_, _, err := enumerate(d, []string{"cmd"})
	if err == nil || !strings.Contains(err.Error(), "bad.go") {
		t.Fatalf("error = %v", err)
	}
}

// TestEnumerator_LookPathIsASTBased pins the iteration-298 evaluator's BLOCKING
// finding. The LookPath invariant used to be a line-anchored grep, while the
// exec.Command enumeration was AST-based for the express reason that grep cannot
// see the shape gofmt produces (design HID-6). That reasoning was never applied
// to the sibling check inside the same gate, so a gofmt-canonical multi-line
// duplicate resolver — and an aliased os/exec import — both passed with rc=0.
//
// Every arm below is gofmt-canonical: none is a shape `make fmt-check` would
// reformat away. The last arm asserts a DECLARED residual (dataflow through a
// constant), so the known miss is pinned deliberately rather than left silent.
func TestEnumerator_LookPathIsASTBased(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want int
	}{
		{"single line", "package p\n\nimport \"os/exec\"\n\nfunc f() { _, _ = exec.LookPath(\"git\") }\n", 1},
		{"multi line", "package p\n\nimport \"os/exec\"\n\nfunc f() {\n\t_, _ = exec.LookPath(\n\t\t\"git\")\n}\n", 1},
		{"aliased import", "package p\n\nimport exe \"os/exec\"\n\nfunc f() { _, _ = exe.LookPath(\"git\") }\n", 1},
		{"not git", "package p\n\nimport \"os/exec\"\n\nfunc f() { _, _ = exec.LookPath(\"hg\") }\n", 0},
		{"declared residual: constant not literal", "package p\n\nimport \"os/exec\"\n\nconst n = \"git\"\n\nfunc f() { _, _ = exec.LookPath(n) }\n", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := t.TempDir()
			write(t, d, "cmd/x.go", tc.src)
			_, lp, err := enumerate(d, []string{"cmd"})
			if err != nil {
				t.Fatal(err)
			}
			if len(lp) != tc.want {
				t.Fatalf("LookPath findings = %d, want %d (%v)", len(lp), tc.want, lp)
			}
		})
	}
}
