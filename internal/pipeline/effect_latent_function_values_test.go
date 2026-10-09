package pipeline

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLatentFunctionValues(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	tests := []struct{ name, source, blame string }{
		{"direct", `func loud(x: int) -> int ! {IO} { let _ = println("EFFECT PERFORMED"); x }
 pure func sneaky() -> int = loud(1)`, "sneaky"},
		{"local_hof", `func loud(x: int) -> int ! {IO} { let _ = println("EFFECT PERFORMED"); x }
 func applyTo(f: int -> int, x: int) -> int = f(x)
 pure func sneaky() -> int = applyTo(loud, 1)`, "sneaky"},
		{"wrong_row", `func loud(x: int) -> int ! {IO} { let _ = println("EFFECT PERFORMED"); x }
 func applyTo(f: int -> int, x: int) -> int = f(x)
 func sneaky() -> int ! {FS} = applyTo(loud, 1)`, "sneaky"},
		{"field", `type Hooks = { f: int -> int ! {IO} }
 func rowless(h: Hooks) -> int = h.f(1)`, "rowless"},
		{"field_wrong_row", `type Hooks = { f: int -> int ! {IO} }
 func rowless(h: Hooks) -> int ! {FS} = h.f(1)`, "rowless"},
		{"inline_field", `func rowless(h: {f: int -> int ! {IO}}) -> int = h.f(1)`, "rowless"},
		{"parameter", `func rowless(f: int -> int ! {IO}) -> int = f(1)`, "rowless"},
		{"return", `func loud(x: int) -> int ! {IO} { let _ = println("EFFECT PERFORMED"); x }
 func get() -> int -> int ! {IO} = loud
 func rowless() -> int = get()(1)`, "rowless"},
		{"let", `func loud(x: int) -> int ! {IO} { let _ = println("EFFECT PERFORMED"); x }
 func rowless() -> int { let f: int -> int ! {IO} = loud; f(1) }`, "rowless"},
		{"alias", `type Callback = int -> int ! {IO}
 func rowless(f: Callback) -> int = f(1)`, "rowless"},
		{"adt", `type Hook = Hook(int -> int ! {IO})
 func rowless(h: Hook) -> int = match h { Hook(f) => f(1) }`, "rowless"},
		{"update", `type Hooks = { f: int -> int ! {IO} }
 func quiet(x: int) -> int = x
 func loud(x: int) -> int ! {IO} { let _ = println("EFFECT PERFORMED"); x }
 func base() -> Hooks = {f: loud}
 func rowless() -> int { let h = {base() | f: loud}; h.f(1) }`, "rowless"},
		{"storage", `type Hooks = { f: int -> int ! {IO} }
 func loud(x: int) -> int ! {IO} { let _ = println("EFFECT PERFORMED"); x }
 func store() -> Hooks = {f: loud}`, ""},
		{"closed_storage", `type Hooks = { f: int -> int ! {IO} }
 func loud(x: int) -> int ! {IO} { let _ = println("EFFECT PERFORMED"); x }
 func box(f: int -> int ! {IO}) -> Hooks = {f: f}
 func store() -> Hooks = box(loud)`, ""},
		{"shadow_parameter", `func step(x: int) -> int ! {IO} { let _ = println("EFFECT PERFORMED"); x }
 pure func applyStep(step: int -> int, x: int) -> int = step(x)`, ""},
		{"shadow_let", `func step(x: int) -> int ! {IO} { let _ = println("EFFECT PERFORMED"); x }
 pure func use(x: int) -> int { let step = \y. y; step(x) }`, ""},
		{"declared_hof", `func loud(x: int) -> int ! {IO} { let _ = println("EFFECT PERFORMED"); x }
 func applyTo(f: int -> int, x: int) -> int = f(x)
 func use() -> int ! {IO} = applyTo(loud, 1)`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkModules(t, map[string]string{"main.ail": "module main\nimport std/io (println)\n" + tt.source})
			if tt.blame == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("accepted undeclared IO")
			}
			for _, want := range []string{"Effect checking failed for function '" + tt.blame + "'", "Missing effects: IO"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("want %q: %v", want, err)
				}
			}
		})
	}
}

func TestLatentStdHOFs(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, tt := range []struct{ name, callback, call, result string }{
		{"sortBy", "func cb(a: int,b: int) -> int ! {IO} {let _ = println(\"E\"); a-b}", "sortBy(cb,[2,1])", "[int]"},
		{"any", "func cb(a: int) -> bool ! {IO} {let _ = println(\"E\"); true}", "any(cb,[1])", "bool"},
		{"findIndex", "func cb(a: int) -> bool ! {IO} {let _ = println(\"E\"); true}", "findIndex(cb,[1])", "Option[int]"},
		{"foldr", "func cb(a: int,b: int) -> int ! {IO} {let _ = println(\"E\"); a+b}", "foldr(cb,0,[1])", "int"},
		{"zipWith", "func cb(a: int,b: int) -> int ! {IO} {let _ = println(\"E\"); a+b}", "zipWith(cb,[1],[2])", "[int]"},
		{"flatMap", "func cb(a: int) -> [int] ! {IO} {let _ = println(\"E\"); [a]}", "flatMap(cb,[1])", "[int]"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, row := range []string{"", " ! {IO}"} {
				source := "module main\nimport std/io (println)\nimport std/option (Option)\nimport std/list (" + tt.name + ")\n" + tt.callback + "\nfunc sneaky() -> " + tt.result + row + " = " + tt.call
				err := checkModules(t, map[string]string{"main.ail": source})
				if tt.name == "any" || tt.name == "findIndex" || tt.name == "foldr" {
					if err == nil || !strings.Contains(err.Error(), "incompatible closed rows") {
						t.Fatalf("existing closed-callback contract changed: %v", err)
					}
					continue
				}
				if row != "" {
					if err != nil {
						t.Fatal(err)
					}
					continue
				}
				assertLatentIO(t, err, "sneaky")
			}
		})
	}
}

func assertLatentIO(t *testing.T, err error, name string) {
	t.Helper()
	if err == nil {
		t.Fatal("accepted undeclared IO")
	}
	for _, want := range []string{"Effect checking failed for function '" + name + "'", "Missing effects: IO"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("want %q: %v", want, err)
		}
	}
}

func TestLatentCrossModule(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, tt := range []struct{ name, lib, call, result string }{
		{"recursive", `export pure func anyR(f: int -> bool, xs: [int]) -> bool = match xs { [] => false, x :: rest => if f(x) then true else anyR(f, rest) }`, "anyR(cb,[1])", "bool"},
		{"forwarding", `func applyTo(f: int -> bool,x: int) -> bool = f(x)
export pure func applyVia(f: int -> bool,x: int) -> bool = applyTo(f,x)`, "applyVia(cb,1)", "bool"},
		{"std_forwarding", `import std/list (flatMap)
export pure func fm(f: int -> bool,xs: [int]) -> [bool] = flatMap(\x. [f(x)],xs)`, "fm(cb,[1])", "[bool]"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			exported := map[string]string{"recursive": "anyR", "forwarding": "applyVia", "std_forwarding": "fm"}[tt.name]
			for _, row := range []string{"", " ! {IO}"} {
				files := map[string]string{
					"helper.ail": "module helper\n" + tt.lib,
					"main.ail":   "module main\nimport helper (" + exported + ")\nimport std/io (println)\nfunc cb(x: int) -> bool ! {IO} {let _ = println(\"E\"); true}\nfunc sneaky() -> " + tt.result + row + " = " + tt.call,
				}
				err := checkModules(t, files)
				if row != "" {
					if err != nil {
						t.Fatal(err)
					}
					continue
				}
				assertLatentIO(t, err, "sneaky")
			}
		})
	}
	// A source concrete annotation must stay a storage contract across imports.
	err := checkModules(t, map[string]string{
		"helper.ail": `module helper
export type Hooks = {f: int -> int ! {IO}}
export func box(f: int -> int ! {IO}) -> Hooks = {f:f}`,
		"main.ail": `module main
import helper (box, Hooks)
import std/io (println)
func loud(x:int) -> int ! {IO} {let _ = println("E"); x}
func store() -> Hooks = box(loud)`,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestLatentComputedCalleesAndShadowing(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, tt := range []struct{ name, body, blame string }{
		{"lambda", `func sneaky() -> int = (\f. f(1))(loud)`, "sneaky"},
		{"local_alias", `func applyTo(f: int -> int,x:int) -> int = f(x)
func sneaky() -> int {let apply = applyTo; apply(loud,1)}`, "sneaky"},
		{"hof_field", `func applyTo(f: int -> int,x:int) -> int = f(x)
func sneaky() -> int {let h = {apply:applyTo}; h.apply(loud,1)}`, "sneaky"},
		{"computed", `func applyTo(f: int -> int,x:int) -> int = f(x)
func sneaky() -> int = (if true then applyTo else applyTo)(loud,1)`, "sneaky"},
		{"pattern_shadow", `type Box = Box(int -> int)
pure func use(b: Box) -> int = match b {Box(loud) => loud(1)}`, ""},
		{"annotated_named_tail", `func applyE[e](f: int -> int ! {e},x:int) -> int ! {e} = f(x)
func sneaky() -> int = applyE(loud,1)`, "sneaky"},
		{"mixed_named_tail", `func applyE[e](f: int -> int ! {IO,e},x:int) -> int ! {IO,e} = f(x)
func sneaky() -> int = applyE(loud,1)`, "sneaky"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := checkModules(t, map[string]string{"main.ail": `module main
import std/io (println)
func loud(x:int) -> int ! {IO} {let _ = println("E"); x}
` + tt.body})
			if tt.blame == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else {
				assertLatentIO(t, err, tt.blame)
			}
		})
	}
}

func TestLatentStorageAlias(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, body := range []string{
		"{let mk = box; mk(loud)}",
		"= (if true then box else box)(loud)",
		"{let h = {mk:box}; h.mk(loud)}",
		"{let mk: (int -> int ! {IO}) -> Hooks = box; mk(loud)}",
	} {
		t.Run(body, func(t *testing.T) {
			err := checkModules(t, map[string]string{"main.ail": `module main
import std/io (println)
type Hooks = {f:int -> int ! {IO}}
func loud(x:int) -> int ! {IO} {let _ = println("E"); x}
func box(f:int -> int ! {IO}) -> Hooks = {f:f}
func store() -> Hooks ` + body})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLatentImportedFunctionAlias(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	err := checkModules(t, map[string]string{
		"helper.ail": `module helper
export type Callback = int -> int ! {IO}`,
		"main.ail": `module main
import helper (Callback)
func rowless(f:Callback) -> int = f(1)`,
	})
	assertLatentIO(t, err, "rowless")
}

func TestLatentDOMReplay(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	path, err := filepath.Abs(filepath.Join("..", "..", "examples", "cognitive_os", "single_agent_replay.ail"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Config{Mode: ModeCheck, RelaxModules: true, NoCache: true}, Source{Filename: path}); err != nil {
		t.Fatal(err)
	}
}
