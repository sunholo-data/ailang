package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPipelineIFCFieldSinks(t *testing.T) {
	for _, tc := range []struct {
		name, decl, body string
		reject           bool
	}{
		{"plain", `type Box = {payload: string{not secret}}`, `pure func f() -> Box = {payload: source()}`, true},
		{"domain", `type Handler = {run: (string{not secret}) -> string}`, `pure func f(h: Handler) -> string = h.run(source())`, true},
		{"codomain", `type Maker = {mk: () -> string{not secret}}`, `pure func f() -> int = 1`, true},
		{"safe", `type Box = {payload: string{not secret}}`, `pure func f() -> Box = {payload: "safe"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
			src := "module field_sinks\n" + tc.decl + "\npure func source() -> string<secret> = \"secret\"\n" + tc.body + "\n"
			if err := os.WriteFile("field_sinks.ail", []byte(src), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "field_sinks.ail"}, productionCacheDependencies())
			if tc.reject {
				if err == nil || (!strings.Contains(err.Error(), "information-flow") && !strings.Contains(err.Error(), "not enforceable")) {
					t.Fatalf("sink silently accepted or wrong failure: %v", err)
				}
			} else if err != nil {
				t.Fatalf("safe field rejected: %v", err)
			}
		})
	}
}
