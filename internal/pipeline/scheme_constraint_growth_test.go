package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// nestedLetEqChain builds n local lambdas where each reuses the previous one
// three times and the first carries a non-ground Eq constraint.
func nestedLetEqChain(n int) string {
	var b strings.Builder
	b.WriteString("module chain\nexport pure func f(x: int, y: string) -> bool {\n")
	b.WriteString("  let g1 = \\z. z == z;\n")
	for i := 2; i <= n; i++ {
		fmt.Fprintf(&b, "  let g%d = \\z. g%d(z) && g%d(y) && g%d(x);\n", i, i-1, i-1, i-1)
	}
	fmt.Fprintf(&b, "  g%d(x)\n}\n", n)
	return b.String()
}

// TestSchemeConstraints_NestedLetsStayLinear guards #1811: let-bound schemes
// copied EVERY unsolved constraint of the context and inferVar re-emitted them
// at each use, so constraints grew exponentially with let depth (this fixture
// allocated ~500 MB at depth 10 and ~4x more per two levels; sunholo/crew_core
// reached 14 GB and OOM-killed the registry validator). With relevance
// filtering it allocates a few MB at any depth.
func TestSchemeConstraints_NestedLetsStayLinear(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	src := nestedLetEqChain(12)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "chain.ail"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, err := Run(Config{DryLink: true}, Source{Code: src, Filename: "chain.ail"})
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatalf("check failed: %v", err)
	}

	const budget = 64 << 20
	if alloc := after.TotalAlloc - before.TotalAlloc; alloc > budget {
		t.Fatalf("type-checking a depth-12 let chain allocated %d MB (budget %d MB): scheme constraints are multiplying again", alloc>>20, budget>>20)
	}
}
