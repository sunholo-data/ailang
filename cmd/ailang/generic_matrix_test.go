package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// A generic body is compiled once, at its defaulted type (`Num a` := int), and
// every consumer of a recorded node type must still give the run-time value's
// answer (#2273 review): the class methods are corrected from the run-time
// values, ShowNormalizer leaves show dynamic on defaulted nodes, and mod_Int
// is corrected like the Num methods. Each module in testdata/generic_matrix/mx
// computes one operator shape with a generic `gen` and its monomorphic twins
// (`twinI`, `twinF`, sometimes `twinS`) and prints `genX=…` / `twinX=…` pairs;
// on the evaluator and the strict VM every pair must agree. A new consumer that
// picks an implementation from a defaulted type fails here.
func TestGenericBodiesMatchTheirMonomorphicTwins(t *testing.T) {
	bin := buildAilang(t)
	dir, err := filepath.Abs(filepath.Join("testdata", "generic_matrix"))
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "mx", "*.ail"))
	if err != nil || len(files) < 30 {
		t.Fatalf("matrix modules missing (%d found): %v", len(files), err)
	}
	pair := regexp.MustCompile(`^\s*(gen|twin)([A-Z])=(.*)$`)
	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".ail")
		t.Run(name, func(t *testing.T) {
			for _, engine := range [][]string{nil, {"--strict-bytecode"}} {
				args := append([]string{"run", "--quiet"}, engine...)
				args = append(args, "--caps", "IO", "--entry", "main", "mx/"+name+".ail")
				out, stderr, code := runAilangIn(t, bin, dir, args...)
				if code != 0 {
					t.Fatalf("%s %v: exit %d: %s", name, engine, code, stderr)
				}
				gen, twin := map[string]string{}, map[string]string{}
				for _, line := range strings.Split(out, "\n") {
					m := pair.FindStringSubmatch(line)
					if m == nil {
						continue
					}
					if m[1] == "gen" {
						gen[m[2]] = m[3]
					} else {
						twin[m[2]] = m[3]
					}
				}
				if len(gen) == 0 {
					t.Fatalf("%s %v: no gen/twin lines in %q", name, engine, out)
				}
				for k, g := range gen {
					if tw, ok := twin[k]; !ok || tw != g {
						t.Errorf("%s %v: gen%s=%q but twin%s=%q", name, engine, k, g, k, tw)
					}
				}
			}
		})
	}
}

// runAilangIn runs the binary with cwd dir (the matrix modules are `mx/<name>`).
func runAilangIn(t *testing.T, bin, dir string, args ...string) (string, string, int) {
	t.Helper()
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(repo, "std"))
	if _, err := os.Stat(dir); err != nil {
		t.Fatal(err)
	}
	return testutil.RunBounded(t, dir, 60*time.Second, bin, args...)
}
