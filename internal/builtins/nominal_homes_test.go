package builtins

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
	"github.com/sunholo-data/ailang/internal/types/traverse"
)

// TestBuiltinHomes_Complete is the D4 completeness test of
// M-V1-NOMINAL-TYPE-IDENTITY: every nominal type constructor in every builtin
// signature carries a defining module, and every std home it names really
// declares that type. A builtin that minted `Foo` with no home would be a
// distinct `$builtin` type that no user-visible `Foo` unifies with — a false
// rejection of every valid program calling it.
func TestBuiltinHomes_Complete(t *testing.T) {
	root := repoRoot(t)
	declared := map[string]bool{} // "std/x::T"
	declRe := regexp.MustCompile(`(?m)^\s*(?:export\s+)?type\s+([A-Z][A-Za-z0-9_]*)\b`)
	err := filepath.Walk(filepath.Join(root, "std"), func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".ail") {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		mod := strings.TrimSuffix(filepath.ToSlash(rel), ".ail")
		for _, m := range declRe.FindAllStringSubmatch(string(src), -1) {
			declared[mod+"::"+m[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(declared) < 50 {
		t.Fatalf("found only %d std type declarations: the scan is wrong", len(declared))
	}

	seen := map[string]bool{}
	var problems []string
	for name, spec := range AllSpecs() {
		if spec.Type == nil {
			continue
		}
		traverse.Walk(spec.Type(), func(n types.Type) {
			c, ok := n.(*types.TCon)
			if !ok {
				return
			}
			switch {
			case types.IsModuleless(c.Name):
				if c.Module != "" {
					problems = append(problems, name+": module-less "+c.Name+" carries module "+c.Module)
				}
			case c.Module == "":
				problems = append(problems, name+": nominal "+c.Name+" has no defining module")
			case c.Module == types.BuiltinModule:
				seen[c.Name] = true
			default:
				seen[c.Name] = true
				if !declared[c.Module+"::"+c.Name] {
					problems = append(problems, name+": "+c.Name+" is homed in "+c.Module+", which does not declare it")
				}
			}
		})
	}
	// Every table entry is either used by a builtin or declared where it says.
	for _, n := range types.BuiltinHomeNames() {
		home, _ := types.BuiltinHome(n)
		if home != types.BuiltinModule && !declared[home+"::"+n] {
			problems = append(problems, "home table: "+n+" -> "+home+", which does not declare it")
		}
		if home == types.BuiltinModule {
			for k := range declared {
				if strings.HasSuffix(k, "::"+n) {
					problems = append(problems, "home table: "+n+" is $builtin but "+k+" declares it")
				}
			}
		}
	}
	sort.Strings(problems)
	if len(problems) > 0 {
		t.Fatalf("%d builtin nominal-home problems:\n%s", len(problems), strings.Join(problems, "\n"))
	}
	if len(seen) < 10 {
		t.Fatalf("saw only %d nominal builtin types: the walk is wrong", len(seen))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
