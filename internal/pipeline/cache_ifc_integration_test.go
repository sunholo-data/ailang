package pipeline

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
)

func ifcCacheFixture(t *testing.T, lib string) {
	t.Helper()
	t.Chdir(t.TempDir())
	t.Setenv("AILANG_CACHE_DIR", "")
	if err := os.WriteFile("lib.ail", []byte(lib), 0600); err != nil {
		t.Fatal(err)
	}
	app := `module app
import lib (provide)
export func main() -> string { provide("hello") }
`
	if err := os.WriteFile("app.ail", []byte(app), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestCacheIFCLibraryEditsInvalidateCaller(t *testing.T) {
	for _, tc := range []struct{ name, before, after string }{
		{"refinement", `export func provide(x: string{not email}) -> string { x }`, `export func provide(x: string{not secret}) -> string { x }`},
		{"intrinsic", `export func provide(x: string) -> string { x }`, `export func provide(x: string) -> string { let tagged: string<email> = x in tagged }`},
		{"return", `export func provide(x: string) -> string { x }`, `export func provide(x: string) -> string<email> { x }`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ifcCacheFixture(t, "module lib\n"+tc.before+"\n")
			cfg := Config{Mode: ModeCheck}
			src := Source{Filename: "app.ail"}
			if _, err := runModuleWithCacheDependencies(t.Context(), cfg, src, productionCacheDependencies()); err != nil {
				t.Fatal(err)
			}
			manifestPath := filepath.Join(".ailang", "cache", "compile", "manifest.json")
			before := readCacheManifest(t, manifestPath)
			var encodes int
			deps := cacheDependencies{stderr: io.Discard, newStore: func(dir string) (*CacheStore, error) {
				cs, err := NewCacheStore(dir)
				if err == nil {
					encode := cs.artifactCodec.encodeCore
					cs.artifactCodec.encodeCore = func(p *core.Program) ([]byte, error) { encodes++; return encode(p) }
				}
				return cs, err
			}}
			// A cached caller and dependency both reuse their authenticated verdicts.
			if _, err := runModuleWithCacheDependencies(t.Context(), cfg, src, deps); err != nil {
				t.Fatal(err)
			}
			if encodes != 0 {
				t.Fatalf("warm caller compiled %d modules", encodes)
			}
			if err := os.WriteFile("lib.ail", []byte("module lib\n"+tc.after+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := runModuleWithCacheDependencies(t.Context(), cfg, src, deps); err != nil {
				t.Fatal(err)
			}
			after := readCacheManifest(t, manifestPath)
			if before.Entries["lib"].IfaceDigest == after.Entries["lib"].IfaceDigest {
				t.Fatal("IFC-only edit did not change library digest")
			}
			if before.Entries["app"].CacheKey == after.Entries["app"].CacheKey {
				t.Fatal("IFC-only edit served stale caller")
			}
			if encodes != 2 {
				t.Fatalf("library edit compiled %d modules, want library and caller", encodes)
			}
			encodes = 0
			if _, err := runModuleWithCacheDependencies(t.Context(), cfg, src, deps); err != nil {
				t.Fatal(err)
			}
			if encodes != 0 {
				t.Fatal("edited warm verdict did not reuse cache")
			}
		})
	}
}

func TestCacheIFCLegacySchemaRecompiles(t *testing.T) {
	ifcCacheFixture(t, "module lib\nexport func provide(x: string{not email}) -> string { x }\n")
	cfg := Config{Mode: ModeCheck}
	src := Source{Filename: "app.ail"}
	if _, err := runModuleWithCacheDependencies(t.Context(), cfg, src, productionCacheDependencies()); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(".ailang", "cache", "compile", "modules", "lib")
	rewriteIFCArtifact(t, dir, func(p map[string]any) { p["schema"] = "ailang.iface/v1" })
	var warnings bytes.Buffer
	if _, err := runModuleWithCacheDependencies(t.Context(), cfg, src, cacheDependencies{newStore: NewCacheStore, stderr: &warnings}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(warnings.String(), "CACHE_INVALID module=lib") {
		t.Fatalf("unobservable schema miss: %s", warnings.String())
	}
	raw, err := os.ReadFile(filepath.Join(dir, artifactIfaceName))
	if err != nil {
		t.Fatal(err)
	}
	repaired, err := unmarshalIfaceFull(raw)
	if err != nil {
		t.Fatal(err)
	}
	if repaired.Exports["provide"].IFC.Params[0].Forbidden[0] != "email" {
		t.Fatal("source recompilation lost refinement")
	}
	// The loader must fail with a source diagnostic when no valid interface/source
	// can be supplied; the legacy cache cannot serve as a clean substitute.
	if err := os.Remove("lib.ail"); err != nil {
		t.Fatal(err)
	}
	if _, err := runModuleWithCacheDependencies(t.Context(), cfg, src, productionCacheDependencies()); err == nil {
		t.Fatal("missing source accepted through cache")
	}
}

func TestCacheIFCColdWarmLocalVerdict(t *testing.T) {
	ifcCacheFixture(t, "module lib\nexport func provide(x: string) -> string { x }\n")
	cfg := Config{Mode: ModeCheck}
	src := Source{Filename: "app.ail"}
	if _, err := runModuleWithCacheDependencies(t.Context(), cfg, src, productionCacheDependencies()); err != nil {
		t.Fatal(err)
	}
	bad := `module app
import lib (provide)
func sink(x: string{not email}) -> string { x }
export func main() -> string { let x: string<email> = provide("hello") in sink(x) }
`
	if err := os.WriteFile("app.ail", []byte(bad), 0600); err != nil {
		t.Fatal(err)
	}
	_, warmErr := runModuleWithCacheDependencies(t.Context(), cfg, src, productionCacheDependencies())
	_, coldErr := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck, NoCache: true}, src, productionCacheDependencies())
	if warmErr == nil || coldErr == nil || !strings.Contains(warmErr.Error(), "sink") || !strings.Contains(coldErr.Error(), "sink") {
		t.Fatalf("cold/warm IFC mismatch: %v / %v", coldErr, warmErr)
	}
}
