package pipeline

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// coreTICapturingDeps records every CoreTypeInfo the cache publishes, by module directory.
func coreTICapturingDeps(captured map[string]types.CoreTypeInfo) cacheDependencies {
	return cacheDependencies{stderr: io.Discard, newStore: func(projectDir string) (*CacheStore, error) {
		store, err := NewCacheStore(projectDir)
		if err == nil {
			encode := store.artifactCodec.encodeCoreTI
			n := 0
			store.artifactCodec.encodeCoreTI = func(info types.CoreTypeInfo) ([]byte, error) {
				n++
				captured[fmt.Sprintf("module-%03d", n)] = info
				return encode(info)
			}
		}
		return store, err
	}}
}

// Every CoreTypeInfo that real compiles publish decodes, through the DAG codec,
// to exactly what a plain gob round trip yields.
func TestCoreTIDAG_StdCorpusMatchesGob(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", "")
	stdlib := []string{"ai", "array", "bytes", "datetime", "fs", "io", "iter", "json", "list", "map", "math", "option", "regex", "result", "stream", "string"}
	var src strings.Builder
	src.WriteString("module main\n")
	for _, m := range stdlib {
		fmt.Fprintf(&src, "import std/%s as M%s\n", m, m)
	}
	src.WriteString("export pure func main() -> int { 1 }\n")
	writeFileForTest(t, "main.ail", src.String())
	writeFileForTest(t, "a.ail", closureA)
	writeFileForTest(t, "b.ail", closureB)
	writeFileForTest(t, "closure.ail", "module closure\nimport a (Box, mk, count, mkItem, countItem)\nimport b (Item, one)\nexport pure func main() -> int { count(mk()) + countItem(mkItem()) }\n")

	captured := map[string]types.CoreTypeInfo{}
	for _, file := range []string{"main.ail", "closure.ail"} {
		if _, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: file}, coreTICapturingDeps(captured)); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}
	if len(captured) < len(stdlib) {
		t.Fatalf("captured %d CoreTypeInfo blobs, want at least %d", len(captured), len(stdlib))
	}
	nodes := 0
	for name, info := range captured {
		t.Run(name, func(t *testing.T) { assertCoreTIEquivalent(t, info) })
		nodes += len(info)
	}
	t.Logf("%d modules, %d CoreTypeInfo entries", len(captured), nodes)
}

// The #1778 guard: nested record aliases imported from another module, used
// across many functions. Plain gob writes each expansion under every node that
// mentions it; the DAG blob must stay small, so the next expansion change shows
// up here rather than in motoko's startup time.
func TestCoreTIDAG_NestedImportedAliasesStayCacheable(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", "")
	var lib strings.Builder
	lib.WriteString("module lib\nexport type L0 = {id: string, n: int}\n")
	for depth := 1; depth <= 6; depth++ {
		fmt.Fprintf(&lib, "export type L%d = {a: [L%d], b: [L%d], c: [L%d]}\n", depth, depth-1, depth-1, depth-1)
	}
	lib.WriteString("export pure func empty() -> L6 { {a: [], b: [], c: []} }\n")
	writeFileForTest(t, "lib.ail", lib.String())
	var app strings.Builder
	app.WriteString("module app\nimport lib (L6, empty)\n")
	const funcs = 200
	for i := 0; i < funcs; i++ {
		fmt.Fprintf(&app, "pure func step%d(x: L6) -> L6 { {x | a: x.b} }\n", i)
	}
	app.WriteString("export pure func main() -> int { match step0(step1(empty())).a { [] => 0, _ :: _ => 1 } }\n")
	writeFileForTest(t, "app.ail", app.String())

	captured := map[string]types.CoreTypeInfo{}
	var warnings bytes.Buffer
	deps := coreTICapturingDeps(captured)
	deps.stderr = &warnings
	if _, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "app.ail"}, deps); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(warnings.String(), "CACHE_WRITE_FAILED") {
		t.Fatalf("cache write failed: %s", warnings.String())
	}
	blob, err := os.ReadFile(filepath.Join(root, ".ailang", "cache", "compile", "modules", "app", artifactCoreTIName))
	if err != nil {
		t.Fatal(err)
	}
	var appInfo types.CoreTypeInfo
	for _, info := range captured {
		if len(info) > len(appInfo) {
			appInfo = info
		}
	}
	var plain bytes.Buffer
	if err := gob.NewEncoder(&plain).Encode(appInfo); err != nil {
		t.Fatal(err)
	}
	t.Logf("app: %d entries, plain gob %d bytes, written dag %d bytes", len(appInfo), plain.Len(), len(blob))
	// Budget: ~4x the 8.1 KB measured at introduction (plain gob: 16.3 MB).
	const budget = 32 << 10
	if len(blob) > budget {
		t.Fatalf("app coretypeinfo.gob is %d bytes, budget %d: a type-expansion change has made CoreTypeInfo grow with expanded rather than distinct structure", len(blob), budget)
	}
	if len(blob)*10 > plain.Len() {
		t.Fatalf("dag %d bytes is not 10x smaller than plain gob %d bytes", len(blob), plain.Len())
	}
}

func writeFileForTest(t *testing.T, name, content string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
