package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Each self-call shares the recursive function's declared-closed effect row.
// A long conditional chain must publish the same solved row as a short one.
func TestRecursiveConditionalApplicationPublication(t *testing.T) {
	for _, count := range []int{4, 5, 8} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			branch := "frame(x)"
			for i := count - 1; i >= 0; i-- {
				branch = fmt.Sprintf("if x == %d then frame(x + 1) else %s", i, branch)
			}
			code := "module recursive_rows\nexport pure func frame(x:int)->int = " + branch + "\n"
			path := filepath.Join(t.TempDir(), "recursive_rows.ail")
			if err := os.WriteFile(path, []byte(code), 0600); err != nil {
				t.Fatal(err)
			}
			result, err := Run(Config{Mode: ModeCheck, TransientRoot: true, NoCache: true}, Source{Filename: path, Code: code})
			if err != nil || len(result.Errors) != 0 {
				t.Fatalf("valid recursive pure function rejected: %v %v", err, result.Errors)
			}
		})
	}
}

func TestRecursiveConditionalStillRejectsRealEffect(t *testing.T) {
	branch := "frame(x)"
	for i := 5; i >= 0; i-- {
		branch = fmt.Sprintf("if x == %d then frame(x + 1) else %s", i, branch)
	}
	code := "module recursive_rows_bad\nimport std/io (println)\nexport pure func frame(x:int)->int = if x < 0 then {println(\"effect\"); x} else " + branch + "\n"
	path := filepath.Join(t.TempDir(), "recursive_rows_bad.ail")
	if err := os.WriteFile(path, []byte(code), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := Run(Config{Mode: ModeCheck, TransientRoot: true, NoCache: true}, Source{Filename: path, Code: code})
	if err == nil || !strings.Contains(err.Error(), "Missing effects: IO") {
		t.Fatalf("real effect was hidden or misdiagnosed: %v", err)
	}
}
