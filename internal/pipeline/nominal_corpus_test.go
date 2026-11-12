package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestNominalIdentity_NoUnqualifiedNominalInCorpus compiles every runnable
// example and every std module with the §4.1 test hook on (TestMain): a
// nominal type that reaches unification without its defining module panics,
// so a minting site M-V1-NOMINAL-TYPE-IDENTITY missed fails here by name
// instead of silently forming an identity class of its own. Whether a file
// type-checks is not asserted (make verify-examples owns that); only that no
// unqualified nominal type reaches the unifier.
func TestNominalIdentity_NoUnqualifiedNominalInCorpus(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("repository go.mod not found")
		}
		root = parent
	}
	t.Chdir(root)

	var files []string
	for _, pattern := range []string{"examples/runnable/*.ail", "examples/runnable/*/*.ail", "std/*.ail", "std/*/*.ail"} {
		m, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, m...)
	}
	sort.Strings(files)
	if len(files) < 100 {
		t.Fatalf("corpus too small (%d files): the glob is wrong", len(files))
	}

	var panics []string
	for _, f := range files {
		func() {
			defer func() {
				if r := recover(); r != nil {
					msg := fmt.Sprint(r)
					if strings.Contains(msg, "M-V1-NOMINAL-TYPE-IDENTITY") {
						panics = append(panics, f+": "+msg)
						return
					}
					panic(r)
				}
			}()
			_, _ = Run(Config{Mode: ModeCheck, NoCache: true, RelaxModules: true}, Source{Filename: f})
		}()
	}
	if len(panics) > 0 {
		t.Fatalf("%d of %d files reached unification with an unqualified nominal type:\n%s",
			len(panics), len(files), strings.Join(panics, "\n"))
	}
	t.Logf("%d files compiled, no unqualified nominal type reached unification", len(files))
}
