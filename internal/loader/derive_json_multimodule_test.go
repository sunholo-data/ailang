package loader_test

// DERIVE-JSON (#2404 RC1 M2): `export type T = ... deriving (Json)` in a library
// module — one that imports only std/json and never names Option/Result — must
// (a) export the synthesized encodeT/decodeT to sibling modules before the
// defining module is elaborated, and (b) receive the std/option + std/result
// prelude its synthesized decoder needs, without colliding with a local
// declaration of Option (TST-RC1-07).

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runner"
	"github.com/sunholo-data/ailang/internal/runtime"
)

const djTypesSrc = `module dj2/types

import std/json (Json)

export type Item = { name: string, count: int } deriving (Json)

type Hidden = { secret: string } deriving (Json)
`

const djMainSrc = `module dj2/main

import std/io (println)
import std/json (Json, encode)
import dj2/types (Item, encodeItem, decodeItem)

export func main() -> () ! {IO} {
  let it: Item = { name: "widget", count: 3 };
  let j = encodeItem(it);
  println(encode(j));
  match decodeItem(j) {
    Ok(x) => println("${x.name}:${show(x.count)}"),
    Err(e) => println("error: ${e}")
  }
}
`

// A library module that declares its own Option (shadowing the prelude type)
// alongside deriving (Json). main does not reach Algo: a user ADT named Option
// is a separate, pre-existing strict-VM gap ("unknown ADT tag Option.Algo").
const djsTypesSrc = `module djs/types

import std/json (Json)

export type Option = Nada | Algo(int)

export type Item = { name: string, count: int } deriving (Json)

export pure func pick(o: Option) -> int = match o { Nada => 0, Algo(n) => n }
`

const djsMainSrc = `module djs/main

import std/io (println)
import std/json (Json, encode)
import djs/types (Item, encodeItem, decodeItem)

export func main() -> () ! {IO} {
  let it: Item = { name: "local", count: 7 };
  match decodeItem(encodeItem(it)) {
    Ok(x) => println("${x.name}:${show(x.count + 1)}"),
    Err(e) => println("error: ${e}")
  }
}
`

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TST-RC1-06
func TestDerivedJsonCodecsExportedBeforeElaboration(t *testing.T) {
	root := writeTree(t, map[string]string{"dj2/types.ail": djTypesSrc})
	ml := loader.NewModuleLoader(root)
	for _, name := range []string{"encodeItem", "decodeItem"} {
		decl, err := ml.GetExport("dj2/types", name)
		if err != nil {
			t.Fatalf("GetExport(dj2/types, %s): %v", name, err)
		}
		if decl == nil || decl.Name != name || !decl.IsExport || !decl.IsPure || len(decl.Params) != 1 {
			t.Fatalf("GetExport(dj2/types, %s) = %+v, want a pure exported 1-param codec", name, decl)
		}
	}
	// A private type's codecs stay private.
	if _, err := ml.GetExport("dj2/types", "encodeHidden"); err == nil {
		t.Error("encodeHidden of an unexported type must not be exported")
	}
}

// TST-RC1-06
func TestDerivingJsonLibraryModuleGetsOptionResultPrelude(t *testing.T) {
	root := writeTree(t, map[string]string{"dj2/types.ail": djTypesSrc})
	loaded, err := loader.NewModuleLoader(root).Load("dj2/types")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"std/option", "std/result"} {
		found := false
		for _, imp := range loaded.File.Imports {
			if imp.Path == want && imp.Implicit {
				found = true
			}
		}
		if !found {
			t.Errorf("deriving (Json) library module lacks the implicit %s import its decoder needs", want)
		}
	}
}

// TST-RC1-06
func TestDerivingJsonSiblingImportRunsOnBothEngines(t *testing.T) {
	root := writeTree(t, map[string]string{"dj2/types.ail": djTypesSrc, "dj2/main.ail": djMainSrc})
	t.Chdir(root)
	runMainOnEngines(t, "dj2/main.ail", "{\"name\":\"widget\",\"count\":3}\nwidget:3\n")
}

// TST-RC1-07
func TestDerivingJsonWithLocalOptionDoesNotCollide(t *testing.T) {
	root := writeTree(t, map[string]string{"djs/types.ail": djsTypesSrc, "djs/main.ail": djsMainSrc})
	loaded, err := loader.NewModuleLoader(root).Load("djs/types")
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range loaded.File.Imports {
		if imp.Path != "std/option" {
			continue
		}
		for _, sym := range imp.Symbols {
			if alias := imp.SymbolAliases[sym]; alias == "" || alias == sym {
				t.Errorf("std/option's %s reaches a module declaring its own Option unaliased (%q)", sym, alias)
			}
		}
	}
	t.Chdir(root)
	runMainOnEngines(t, "djs/main.ail", "local:8\n")
}

// runMainOnEngines compiles entry and runs its main on the evaluator and the
// strict bytecode VM, requiring stdout == want on both.
func runMainOnEngines(t *testing.T, entry, want string) {
	t.Helper()
	res, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: true}, pipeline.Source{Filename: entry})
	if err != nil || len(res.Errors) > 0 {
		t.Fatalf("compile %s: %v %v", entry, err, res.Errors)
	}
	for _, engine := range []string{"eval", "strict-bytecode"} {
		var out bytes.Buffer
		rt := runtime.NewModuleRuntime(".")
		if res.DictReg != nil {
			rt.GetEvaluator().SetDictionaryRegistry(res.DictReg)
		}
		ctx := effects.NewEffContext(nil)
		ctx.Grant(effects.Capability{Name: "IO"})
		ctx.IOWriter = &out
		rt.GetEvaluator().SetEffContext(ctx)
		ctx.FnCaller = rt.GetEvaluator().CallValue
		ctx.FnCallerN = rt.GetEvaluator().CallValueN
		err := runner.ExecuteModuleEntrypoint(rt, runner.ModuleExecParams{
			Filename: entry, Iface: res.Interface, Modules: res.Modules, Entry: "main", ArgsJSON: "null",
			NoPrint: true, Quiet: true, PipelineResult: &res,
			BytecodeMode: engine != "eval", StrictBytecode: engine != "eval",
		})
		if err != nil {
			t.Fatalf("%s: %v", engine, err)
		}
		if out.String() != want {
			t.Fatalf("%s: stdout = %q, want %q", engine, out.String(), want)
		}
	}
}
