package pipeline

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// envScanModules writes a two-module fixture: lib exports m functions; app
// imports all of them, declares n top-level functions and one body making k
// applications. Checking app type-checks both.
func envScanModules(t *testing.T, n, m, k int) {
	t.Helper()
	var lib strings.Builder
	lib.WriteString("module lib\n")
	names := make([]string, m)
	for i := range m {
		names[i] = fmt.Sprintf("l%d", i)
		fmt.Fprintf(&lib, "export pure func l%d(x: int) -> int { x + %d }\n", i, i)
	}
	var app strings.Builder
	fmt.Fprintf(&app, "module app\nimport lib (%s)\n", strings.Join(names, ", "))
	for i := range n {
		fmt.Fprintf(&app, "export pure func t%d(x: int) -> int { x * %d }\n", i, i+1)
	}
	app.WriteString("export pure func body(x: int) -> int {\n  let a0 = x;\n")
	for i := 1; i <= k; i++ {
		fmt.Fprintf(&app, "  let a%d = t%d(l%d(a%d));\n", i, i%n, i%m, i-1)
	}
	fmt.Fprintf(&app, "  a%d\n}\n", k)
	for name, src := range map[string]string{"lib.ail": lib.String(), "app.ail": app.String()} {
		if err := os.WriteFile(name, []byte(src), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
}

// envScanWorkFor cold-checks the fixture and returns the bindings visited by
// environment free-variable walks.
func envScanWorkFor(t *testing.T, n, m, k int) uint64 {
	t.Helper()
	t.Chdir(t.TempDir())
	t.Setenv("AILANG_NO_CACHE", "1")
	envScanModules(t, n, m, k)
	prev := types.SetEnvVarCrossCheck(false) // the cross-check re-walks everything
	defer types.SetEnvVarCrossCheck(prev)
	types.ResetEnvScanWork()
	if _, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "app.ail"}, productionCacheDependencies()); err != nil {
		t.Fatalf("check (n=%d m=%d k=%d): %v", n, m, k, err)
	}
	return types.EnvScanWork()
}

// TestEnvScan_WorkPerApplication measures what one more application costs in
// environment walks, as top-level declarations (n) and imports (m) grow. Each
// application used to walk the whole environment, so that marginal cost grew
// with n + m (M-TYPECHECK-ENV-SCAN-COST). Not parallel: the counter is global.
func TestEnvScan_WorkPerApplication(t *testing.T) {
	const k = 20
	marginal := func(n, m int) uint64 {
		lo := envScanWorkFor(t, n, m, k)
		hi := envScanWorkFor(t, n, m, 2*k)
		return (hi - lo) / k
	}
	small := marginal(10, 10)
	wideTop := marginal(160, 10)
	wideImports := marginal(10, 160)
	t.Logf("bindings visited per extra application: n=10,m=10: %d; n=160,m=10: %d; n=10,m=160: %d",
		small, wideTop, wideImports)
}
