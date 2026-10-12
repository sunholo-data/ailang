package pipeline

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/types"
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
			if fn.EffectRow != nil && len(fn.EffectRow.Labels) > 0 {
				labelled++
			}
			if fn.EffectRow != nil && fn.EffectRow.Tail != nil {
				open++
			}
			if fd.IsPure {
				keyword++
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
	sort.Strings(pins)
	for _, pin := range pins {
		t.Log(pin)
	}
	t.Logf("census: modules=%d functions=%d labelled=%d open=%d keyword-pure=%d combinators=%d", modules, functions, labelled, open, keyword, combinators)
}
