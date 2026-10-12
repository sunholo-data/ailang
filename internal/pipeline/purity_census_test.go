package pipeline

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
	"github.com/sunholo-data/ailang/internal/types/traverse"
)

// TestIfacePurity_StdCensus measures every std module; failures are never skipped.
func TestIfacePurity_StdCensus(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
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
	std := filepath.Join(root, "std")
	t.Chdir(filepath.Dir(std))
	modules, functions, labelled, open, keyword, combinators := 0, 0, 0, 0, 0, 0
	publishedPure, declaredOpen, streaming := 0, 0, 0
	expected := map[string]bool{
		"std/list.any":             false,
		"std/list.filter":          false,
		"std/list.findIndex":       false,
		"std/list.flatMap":         true,
		"std/list.foldl":           false,
		"std/list.foldr":           false,
		"std/list.map":             false,
		"std/list.mapAccumL":       false,
		"std/list.sortBy":          true,
		"std/list.takeFlatMap":     false,
		"std/list.takeMap":         false,
		"std/list.zipWith":         false,
		"std/option.filter":        false,
		"std/option.flatMap":       true,
		"std/option.map":           false,
		"std/result.flatMap":       false,
		"std/result.map":           false,
		"std/result.mapErr":        false,
		"std/string.foldChars":     false,
		"std/string.foldSlices":    false,
		"std/string.mapSlicesJoin": false,
		"std/xml.flatMapChildren":  false,
		"std/xml.foldChildren":     false,
		"std/xml.foldChildrenStep": false,
		"std/xml.mapChildren":      false,
		"std/xml.parseFold":        false,
		"std/xml.parseFoldStep":    false,
	}
	seen := map[string]bool{}
	var pins []string
	err = filepath.WalkDir(std, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".ail") {
			return nil
		}
		rel, err := filepath.Rel(filepath.Dir(std), path)
		if err != nil {
			return err
		}
		result, err := Run(Config{Mode: ModeCheck}, Source{Filename: rel})
		if err != nil {
			t.Fatalf("compile %s: %v", path, err)
		}
		modules++
		for _, fd := range result.Artifacts.AST.Funcs {
			item := result.Interface.Exports[fd.Name]
			if item == nil {
				continue
			}
			fn, ok := item.Type.Type.(*types.TFunc2)
			if !ok {
				continue
			}
			functions++
			if item.Purity {
				publishedPure++
			}
			key := result.Interface.Module + "." + fd.Name
			for _, effect := range fd.Effects {
				if effect.IsRowVar {
					declaredOpen++
					if item.Purity {
						t.Errorf("declared-open export %s reports pure", key)
					}
					break
				}
			}
			if result.Interface.Module == "std/ai/streaming" {
				streaming++
				if item.Purity {
					t.Errorf("streaming export %s reports pure", key)
				}
			}
			if key == "std/list.mapE" && item.Purity {
				t.Error("mapE reports pure")
			}
			if wantOpen, pinned := expected[key]; pinned {
				seen[key] = true
				if !item.Purity {
					t.Errorf("combinator %s reports impure", key)
				}
				gotOpen := fn.EffectRow != nil && fn.EffectRow.Tail != nil
				if gotOpen != wantOpen {
					t.Errorf("%s outer tail changed: %s", key, fn)
				}
				if wantOpen {
					shared := false
					for _, param := range fn.Params {
						traverse.Walk(param, func(typ types.Type) {
							if cb, ok := typ.(*types.TFunc2); ok && cb.EffectRow != nil && cb.EffectRow.Tail != nil && cb.EffectRow.Tail.Name == fn.EffectRow.Tail.Name {
								shared = true
							}
						})
					}
					if !shared {
						t.Errorf("%s tail no longer shared with callback", key)
					}
				}
			}
			if fn.EffectRow != nil && len(fn.EffectRow.Labels) > 0 {
				labelled++
				if item.Purity {
					t.Errorf("labelled export %s reports pure", key)
				}
			}
			if fn.EffectRow != nil && fn.EffectRow.Tail != nil {
				open++
			}
			if fd.IsPure {
				keyword++
				if !item.Purity {
					t.Errorf("keyword-pure export %s reports impure", key)
				}
			}
			unannotated := false
			for _, p := range fd.Params {
				if callback, ok := p.Type.(*ast.FuncType); ok && callback.Effects == nil {
					unannotated = true
				}
			}
			if unannotated && len(fd.Effects) == 0 {
				combinators++
				pins = append(pins, result.Interface.Module+"."+fd.Name+" : "+fn.String())
			}
			if fd.Name == "mapE" || result.Interface.Module == "std/ai/streaming" {
				pins = append(pins, result.Interface.Module+"."+fd.Name+" : "+fn.String())
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if modules == 0 || functions == 0 {
		t.Fatal("empty std census")
	}
	for key := range expected {
		if !seen[key] {
			t.Errorf("missing combinator pin %s", key)
		}
	}
	if streaming != 6 {
		t.Errorf("streaming exports=%d want 6", streaming)
	}
	sort.Strings(pins)
	for _, pin := range pins {
		t.Log(pin)
	}
	t.Logf("census: modules=%d functions=%d labelled=%d open=%d keyword-pure=%d combinators=%d declared-open=%d published-pure=%d published-impure=%d", modules, functions, labelled, open, keyword, combinators, declaredOpen, publishedPure, functions-publishedPure)
}
