package pipeline

import (
	"os"
	"strings"
	"testing"
)

// M-IFC-CROSS-MODULE M3 (#1134): a caller is checked against its imports'
// IFC summaries, cold and warm. The verdict must not depend on the cache:
// a cached caller (served without re-running the check) must be recompiled
// when the library adds a sink refinement, and a rejected caller stays
// rejected on every run.
func TestImportedSinkVerdictColdWarmAndLibraryEdit(t *testing.T) {
	ifcCacheFixture(t, "module lib\nexport func provide(x: string) -> string { x }\n")
	app := `module app
import lib (provide)
func tag() -> string<secret> { "k" }
export func main() -> string { provide(tag()) }
`
	if err := os.WriteFile("app.ail", []byte(app), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := Config{Mode: ModeCheck}
	src := Source{Filename: "app.ail"}
	for _, run := range []string{"cold", "warm"} {
		if _, err := runModuleWithCacheDependencies(t.Context(), cfg, src, productionCacheDependencies()); err != nil {
			t.Fatalf("%s: a transparent library parameter accepts <secret>: %v", run, err)
		}
	}
	// The library now forbids <secret> on that parameter: the cached caller's
	// accepted verdict must not be served.
	if err := os.WriteFile("lib.ail", []byte("module lib\nexport func provide(x: string{not secret}) -> string { x }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	want := `reaches parameter "x" of imported function lib.provide, which forbids it (string{not secret})`
	for _, run := range []string{"after edit", "after edit, warm"} {
		_, err := runModuleWithCacheDependencies(t.Context(), cfg, src, productionCacheDependencies())
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: want the imported sink refusal %q, got %v", run, want, err)
		}
	}
}

// A library wrapping ANOTHER library's labelled producer exports that taint
// as its own intrinsic label: the dependency's summaries reach its interface
// build, so a caller two modules away still sees <secret>.
func TestImportedTaintCrossesTwoModules(t *testing.T) {
	ifcCacheFixture(t, "module lib\nexport func token() -> string<secret> { \"k\" }\n")
	mid := "module mid\nimport lib (token)\nexport func fetch() -> string { token() }\n"
	app := `module app
import mid (fetch)
func sink(x: string{not secret}) -> string { x }
export func main() -> string { sink(fetch()) }
`
	for name, src := range map[string]string{"mid.ail": mid, "app.ail": app} {
		if err := os.WriteFile(name, []byte(src), 0600); err != nil {
			t.Fatal(err)
		}
	}
	_, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "app.ail"}, productionCacheDependencies())
	if err == nil || !strings.Contains(err.Error(), `value labelled <secret> reaches parameter "x"`) {
		t.Fatalf("want lib's <secret> to reach app's sink through mid.fetch, got %v", err)
	}
}

// The labelled RESULT of an imported function taints the caller's own sink,
// whether the library declares the label or only computes it.
func TestImportedResultLabelReachesLocalSink(t *testing.T) {
	for _, lib := range []string{
		`export func token() -> string<secret> { "k" }`,
		`func raw() -> string<secret> { "k" }
export func token() -> string { raw() }`,
	} {
		ifcCacheFixture(t, "module lib\n"+lib+"\n")
		app := `module app
import lib (token)
func sink(x: string{not secret}) -> string { x }
export func main() -> string { sink(token()) }
`
		if err := os.WriteFile("app.ail", []byte(app), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "app.ail"}, productionCacheDependencies())
		if err == nil || !strings.Contains(err.Error(), `value labelled <secret> reaches parameter "x"`) {
			t.Fatalf("%s: want the local sink to reject the imported <secret> result, got %v", lib, err)
		}
	}
}
