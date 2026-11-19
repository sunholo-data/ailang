package pipeline

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Every diagnostic this change added must not depend on the compile cache.
// Marking a function deprecated does not change its module's interface digest,
// so a dependant stays a cache hit after a dependency's annotations change.
// Each case compiles cold, edits ONLY deprecation/experimental annotations in a
// dependency, and requires the warm result (dependant served from the cache) to
// equal a cold compile of the edited sources: same verdict, same diagnostics.
func TestDeprecationDiagnosticsAreCacheStable(t *testing.T) {
	lib := func(ann map[string]string) string {
		var b strings.Builder
		b.WriteString("module lib\n\n")
		for _, fn := range []string{"f", "g", "h"} {
			if a := ann[fn]; a != "" {
				b.WriteString(a + "\n")
			}
			fmt.Fprintf(&b, "export pure func %s(x: int) -> int = x\n\n", fn)
		}
		return b.String()
	}
	appUses := "module app\nimport lib (f)\nexport pure func k() -> int = f(1)\n"
	cases := []struct {
		name   string
		before map[string]string // path -> source
		after  map[string]string // only annotation edits in a dependency
		check  string            // file to check
		want   string            // substring the edited result must contain ("" = no diagnostics)
	}{
		{"use: dependency gains @deprecated",
			map[string]string{"lib.ail": lib(nil), "app.ail": appUses},
			map[string]string{"lib.ail": lib(map[string]string{"f": `@deprecated("g")`})},
			"app.ail", "lib.f is deprecated; use g"},
		{"use: dependency loses @deprecated",
			map[string]string{"lib.ail": lib(map[string]string{"f": `@deprecated("g")`}), "app.ail": appUses},
			map[string]string{"lib.ail": lib(nil)},
			"app.ail", ""},
		{"use: target changes",
			map[string]string{"lib.ail": lib(map[string]string{"f": `@deprecated("g")`}), "app.ail": appUses},
			map[string]string{"lib.ail": lib(map[string]string{"f": `@deprecated("h")`})},
			"app.ail", "lib.f is deprecated; use h"},
		{"use: the replacement is deprecated later (chain grows)",
			map[string]string{"lib.ail": lib(map[string]string{"f": `@deprecated("g")`}), "app.ail": appUses},
			map[string]string{"lib.ail": lib(map[string]string{"f": `@deprecated("g")`, "g": `@deprecated("h")`})},
			"app.ail", "lib.f is deprecated; use h"},
		{"use: chain shrinks",
			map[string]string{"lib.ail": lib(map[string]string{"f": `@deprecated("g")`, "g": `@deprecated("h")`}), "app.ail": appUses},
			map[string]string{"lib.ail": lib(map[string]string{"f": `@deprecated("g")`})},
			"app.ail", "lib.f is deprecated; use g"},
		{"use: chain becomes a cycle",
			map[string]string{"lib.ail": lib(map[string]string{"f": `@deprecated("g")`}), "app.ail": appUses},
			map[string]string{"lib.ail": lib(map[string]string{"f": `@deprecated("g")`, "g": `@deprecated("f")`})},
			"app.ail", "lib.f is deprecated; use g"},
		{"unused import: dependency gains @deprecated",
			map[string]string{"lib.ail": lib(nil), "app.ail": "module app\nimport lib (f)\nexport pure func k() -> int = 1\n"},
			map[string]string{"lib.ail": lib(map[string]string{"f": `@deprecated("g")`})},
			"app.ail", "it is imported here and never used"},
		{"test block: dependency gains @deprecated",
			map[string]string{"lib.ail": lib(nil), "app.ail": "module app\nimport lib (f)\nexport pure func k() -> int = 1\ntest \"t\" {\n  assert f(1) == 1\n}\n"},
			map[string]string{"lib.ail": lib(map[string]string{"f": `@deprecated("g")`})},
			"app.ail", "lib.f is deprecated; use g"},
		// The reviewer's repro: a cached dependant names a target that later
		// becomes deprecated. Legal now, cold or warm.
		{"002: a qualified target becomes deprecated",
			map[string]string{"lib.ail": lib(nil),
				"user.ail": "module user\nimport lib (g)\nexport pure func g2(x: int) -> int = g(x)\n@deprecated(\"lib.g\")\nexport pure func old(x: int) -> int = g(x)\n",
				"app.ail":  "module app\nimport user (g2)\nexport pure func k() -> int = g2(1)\n"},
			map[string]string{"lib.ail": lib(map[string]string{"g": `@deprecated("h")`})},
			"app.ail", "lib.g is deprecated; use h"},
		{"002: a dependency's own annotation gets a typo",
			map[string]string{"lib.ail": lib(map[string]string{"f": `@deprecated("g")`}), "app.ail": appUses},
			map[string]string{"lib.ail": lib(map[string]string{"f": `@deprecated("gx")`})},
			"app.ail", "TC_DEPRECATED_002"},
		{"PAR_EXPERIMENTAL_ARG in a dependency",
			map[string]string{"lib.ail": lib(nil), "app.ail": appUses},
			map[string]string{"lib.ail": lib(map[string]string{"f": `@experimental("x")`})},
			"app.ail", "PAR_EXPERIMENTAL_ARG"},
		{"@experimental added in a dependency",
			map[string]string{"lib.ail": lib(nil), "app.ail": appUses},
			map[string]string{"lib.ail": lib(map[string]string{"f": `@experimental`})},
			"app.ail", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			for p, src := range c.before {
				writeModule(t, p, src)
			}
			warmCache := filepath.Join(root, "cache-warm")
			_ = cacheStableCheck(t, c.check, warmCache) // cold, fills the cache
			for p, src := range c.after {
				writeModule(t, p, src)
			}
			warm := cacheStableCheck(t, c.check, warmCache)
			cold := cacheStableCheck(t, c.check, filepath.Join(root, "cache-cold"))
			if warm != cold {
				t.Fatalf("warm and cold disagree after a dependency's annotations changed\nwarm:\n%s\ncold:\n%s", warm, cold)
			}
			if c.want == "" && cold != "" {
				t.Errorf("want no diagnostics, got:\n%s", cold)
			}
			if strings.Contains(cold, "ERROR") && !strings.Contains(c.want, "_0") && !strings.Contains(c.want, "_ARG") {
				t.Errorf("want it to compile, got:\n%s", cold)
			}
			if c.want != "" && !strings.Contains(cold, c.want) {
				t.Errorf("want a diagnostic containing %q, got:\n%s", c.want, cold)
			}
		})
	}
}

// cacheStableCheck compiles file with the given cache dir and renders the
// verdict and every TC_DEPRECATED_* diagnostic as one comparable string.
func cacheStableCheck(t *testing.T, file, cacheDir string) string {
	t.Helper()
	t.Setenv("AILANG_CACHE_DIR", cacheDir)
	res, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: file}, productionCacheDependencies())
	var lines []string
	if err != nil {
		lines = append(lines, "ERROR "+err.Error())
	}
	for _, w := range deprecatedWarnings(res.Warnings) {
		line, col := w.Position()
		lines = append(lines, fmt.Sprintf("%s %s:%d:%d %s | %s", w.Code(), filepath.Base(w.WarningFile()), line, col, w.Message(), w.Suggestion()))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}
