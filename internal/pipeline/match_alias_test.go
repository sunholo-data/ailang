package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// #2244: a type alias of an ADT is that ADT. `type Hue = Color` adds a name,
// not a type, so a value of type Hue is matched with Color's constructors,
// and every check that asks which ADT a value belongs to — the constructor
// ownership check, constructor arity, exhaustiveness — asks it after alias
// expansion. A newtype ADT (`type W = W(...)`) is a type of its own and is
// never expanded.

const aliasColors = "module lib/colors\n\nimport std/option (Option)\n\n" +
	"export type Color = Red | Green | Blue\nexport type Hue = Color\nexport type MaybeHue = Option[Hue]\n\n" +
	"export pure func warm() -> Hue = Red\n"

func TestMatchOnADTAliasAccepted(t *testing.T) {
	const local = "type Color = Red | Green | Blue\ntype Hue = Color\n\n"
	cases := []struct{ name, src string }{
		{"issue repro", "module main\n\n" + local +
			"export pure func code(h: Hue) -> int = match h {\n  Red => 1,\n  Green => 2,\n  Blue => 3\n}\n"},
		{"alias of a prelude type", "module main\n\nimport std/option (Option, Some, None)\n\ntype MaybeInt = Option[int]\n\n" +
			"export pure func get(m: MaybeInt) -> int = match m {\n  Some(n) => n,\n  None => 0\n}\n"},
		{"alias of an alias", "module main\n\n" + local + "type Tint = Hue\n\n" +
			"export pure func code(t: Tint) -> int = match t {\n  Red => 1,\n  Green => 2,\n  Blue => 3\n}\n"},
		{"parameterised alias, generic", "module main\n\nimport std/option (Option, Some, None)\n\ntype Opt[a] = Option[a]\n\n" +
			"export pure func get[a](o: Opt[a], d: a) -> a = match o {\n  Some(x) => x,\n  None => d\n}\n"},
		{"alias nested in a constructor argument", "module main\n\nimport std/option (Option, Some, None)\n\n" + local + "type Opt[a] = Option[a]\n\n" +
			"export pure func f(o: Opt[Hue]) -> int = match o {\n  Some(Red) => 1,\n  Some(Green) => 2,\n  Some(Blue) => 3,\n  None => 0\n}\n"},
		{"annotated and unannotated lambdas", "module main\n\n" + local +
			"export pure func f(h: Hue) -> int {\n  let a = func(x: Hue) -> int {\n    match x {\n      Red => 1,\n      Green => 2,\n      Blue => 3\n    }\n  };\n" +
			"  let b = \\x. match x {\n    Red => 1,\n    Green => 2,\n    Blue => 3\n  };\n  a(h) + b(h)\n}\n"},
		{"two aliases of one ADT are interchangeable", "module main\n\n" + local + "type Shade = Color\n\n" +
			"export pure func toShade(h: Hue) -> Shade = h\n\nexport pure func toHue(s: Shade) -> Hue = s\n\nexport pure func toColor(s: Shade) -> Color = toHue(s)\n"},
		{"exported alias, constructors imported", "module main\n\nimport std/option (Some, None)\nimport lib/colors (Hue, MaybeHue, Red, Green, Blue)\n\n" +
			"pure func code(h: Hue) -> int = match h {\n  Red => 1,\n  Green => 2,\n  Blue => 3\n}\n\n" +
			"export pure func maybe(m: MaybeHue) -> int = match m {\n  Some(h) => code(h),\n  None => 0\n}\n"},
		{"exported alias, constructors matched by name", "module main\n\nimport lib/colors (Hue)\n\n" +
			"export pure func code(h: Hue) -> int = match h {\n  Red => 1,\n  Green => 2,\n  Blue => 3\n}\n"},
		{"alias of a newtype stays the newtype", "module main\n\ntype W = W({ x: int })\ntype V = W\n\n" +
			"export pure func get(v: V) -> int = match v {\n  W(r) => r.x\n}\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			errs, _, err := checkMatchModules(t, map[string]string{"lib/colors.ail": aliasColors, "main.ail": c.src}, "main.ail")
			if err != nil {
				t.Fatalf("rejected: %v %v", errs, err)
			}
		})
	}
}

func TestMatchOnADTAliasStillChecked(t *testing.T) {
	const local = "type Color = Red | Green | Blue\ntype Hue = Color\n\n"
	t.Run("missing arm names the ADT's constructor and the alias", func(t *testing.T) {
		errs, _, err := checkMatchModules(t, map[string]string{"main.ail": "module main\n\n" + local +
			"export pure func code(h: Hue) -> int = match h {\n  Red => 1,\n  Green => 2\n}\n"}, "main.ail")
		if len(errs) != 1 || errs[0].Code != types.TC_MATCH_002 ||
			errs[0].Message != "non-exhaustive match on Hue (= Color) — no arm matches: Blue" {
			t.Fatalf("got %v %v", errs, err)
		}
	})
	t.Run("parameterised alias missing arm", func(t *testing.T) {
		errs, _, err := checkMatchModules(t, map[string]string{"main.ail": "module main\n\nimport std/option (Option, Some, None)\n\ntype IntOpt = Option[int]\n\n" +
			"export pure func get(o: IntOpt) -> int = match o {\n  Some(n) => n\n}\n"}, "main.ail")
		if len(errs) != 1 || errs[0].Code != types.TC_MATCH_002 ||
			errs[0].Message != "non-exhaustive match on IntOpt (= Option[int]) — no arm matches: None" {
			t.Fatalf("got %v %v", errs, err)
		}
	})
	t.Run("exported alias, by-name arms, missing arm", func(t *testing.T) {
		errs, _, err := checkMatchModules(t, map[string]string{"lib/colors.ail": aliasColors, "main.ail": "module main\n\nimport lib/colors (Hue)\n\n" +
			"export pure func code(h: Hue) -> int = match h {\n  Red => 1,\n  Blue => 3\n}\n"}, "main.ail")
		if len(errs) != 1 || errs[0].Code != types.TC_MATCH_002 || !strings.HasSuffix(errs[0].Message, "no arm matches: Green") {
			t.Fatalf("got %v %v", errs, err)
		}
	})
	t.Run("re-exported alias, constructors matched by name", func(t *testing.T) {
		// lib/b re-exports lib/color's Color as Hue: main loads Color's
		// constructors only transitively, so the elaborator stamps none of
		// them and the check resolves them through the alias (review p32).
		files := map[string]string{
			"lib/color.ail": "module lib/color\n\nexport type Color = Red | Green | Blue\n",
			"lib/b.ail":     "module lib/b\n\nimport lib/color (Color)\n\nexport type Hue = Color\n",
		}
		files["main.ail"] = "module main\n\nimport lib/b (Hue)\n\nexport pure func code(h: Hue) -> int = match h {\n  Red => 1,\n  Green => 2,\n  Blue => 3\n}\n"
		if errs, _, err := checkMatchModules(t, files, "main.ail"); err != nil {
			t.Fatalf("exhaustive: %v %v", errs, err)
		}
		files["main.ail"] = "module main\n\nimport lib/b (Hue)\n\nexport pure func code(h: Hue) -> int = match h {\n  Red => 1,\n  Blue => 3\n}\n"
		errs, _, err := checkMatchModules(t, files, "main.ail")
		if len(errs) != 1 || errs[0].Message != "non-exhaustive match on Hue (= Color) — no arm matches: Green" {
			t.Fatalf("missing arm: %v %v", errs, err)
		}
	})
	t.Run("constructor two imports offer, chosen through an alias", func(t *testing.T) {
		// Importing the types brings in their constructors, so Red comes from
		// lib/a and lib/b; the scrutinee's type, an alias of lib/a's Color,
		// decides (ambiguous_ctor.go choiceHead).
		files := map[string]string{
			"lib/a.ail": "module lib/a\n\nexport type Color = Green | Red | Blue\nexport type Hue = Color\n",
			"lib/b.ail": "module lib/b\n\nexport type Mood = Red | Calm\n",
			"main.ail": "module main\n\nimport lib/a (Color, Hue)\nimport lib/b (Mood)\n\n" +
				"export pure func code(h: Hue) -> int = match h {\n  Red => 1,\n  Green => 2,\n  Blue => 3\n}\n\n" +
				"export pure func mood(m: Mood) -> int = match m {\n  Red => 1,\n  Calm => 2\n}\n",
		}
		if errs, _, err := checkMatchModules(t, files, "main.ail"); err != nil {
			t.Fatalf("rejected: %v %v", errs, err)
		}
	})
	t.Run("foreign constructor on an alias", func(t *testing.T) {
		_, _, err := checkMatchModules(t, map[string]string{"main.ail": "module main\n\nimport std/option (Option, Some, None)\n\n" + local +
			"export pure func code(h: Hue) -> int = match h {\n  Red => 1,\n  Some(_) => 2,\n  _ => 3\n}\n"}, "main.ail")
		want := "match arm constructor 'Some' belongs to ADT 'Option', not 'Color' (the scrutinee's type, written as its alias Hue).\n  Color's constructors are: Blue, Green, Red"
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("want %q, got %v", want, err)
		}
	})
	t.Run("foreign constructor matched by name on an exported alias", func(t *testing.T) {
		_, _, err := checkMatchModules(t, map[string]string{"lib/colors.ail": aliasColors,
			"lib/other.ail": "module lib/other\n\nexport type Mood = Calm | Angry\n",
			"lib/hold.ail":  "module lib/hold\n\nimport lib/other (Mood)\n\nexport pure func one() -> int = 1\n",
			"main.ail": "module main\n\nimport lib/colors (Hue)\nimport lib/hold (one)\n\n" +
				"export pure func code(h: Hue) -> int = match h {\n  Calm => 1,\n  _ => 2\n}\n"}, "main.ail")
		want := "match arm constructor 'Calm' belongs to ADT 'Mood', not 'Color' (the scrutinee's type, written as its alias Hue)"
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("want %q, got %v", want, err)
		}
	})
	t.Run("an alias does not make a newtype an alias", func(t *testing.T) {
		_, _, err := checkMatchModules(t, map[string]string{"main.ail": "module main\n\ntype Meters = Meters(int)\ntype Feet = Feet(int)\ntype M = Meters\n\n" +
			"export pure func m(d: M) -> int = match d {\n  Feet(n) => n\n}\n"}, "main.ail")
		want := "match arm constructor 'Feet' belongs to ADT 'Feet', not 'Meters' (the scrutinee's type, written as its alias M)"
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("want %q, got %v", want, err)
		}
	})
	t.Run("an alias of a record newtype is that newtype, not its record", func(t *testing.T) {
		_, _, err := checkMatchModules(t, map[string]string{"main.ail": "module main\n\ntype W = W({ x: int })\ntype U = U({ x: int })\ntype V = W\n\n" +
			"export pure func get(v: V) -> int = match v {\n  U(r) => r.x\n}\n"}, "main.ail")
		want := "match arm constructor 'U' belongs to ADT 'U', not 'W' (the scrutinee's type, written as its alias V)"
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("want %q, got %v", want, err)
		}
	})
	t.Run("alias of one ADT is not another ADT", func(t *testing.T) {
		_, _, err := checkMatchModules(t, map[string]string{"main.ail": "module main\n\n" + local + "type Mood = Calm | Angry\n\n" +
			"export pure func f(h: Hue) -> Mood = h\n"}, "main.ail")
		if err == nil {
			t.Fatal("a Hue (= Color) was accepted as a Mood")
		}
	})
}

// A cold and a warm check of a module matching on an alias agree.
func TestMatchOnADTAliasWarmCache(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
	if err := os.MkdirAll("lib", 0o755); err != nil {
		t.Fatal(err)
	}
	for path, src := range map[string]string{
		"lib/colors.ail": aliasColors,
		"main.ail": "module main\n\nimport lib/colors (Hue, Red, Green)\n\n" +
			"export pure func code(h: Hue) -> int = match h {\n  Red => 1,\n  Green => 2\n}\n",
	} {
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, run := range []string{"cold", "warm"} {
		_, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "main.ail"}, productionCacheDependencies())
		if err == nil || !strings.Contains(err.Error(), "non-exhaustive match on Hue (= Color) — no arm matches: Blue") {
			t.Errorf("%s: got %v", run, err)
		}
	}
	// The exhaustive twin passes cold, and again when served from the cache.
	if err := os.WriteFile("main.ail", []byte("module main\n\nimport lib/colors (Hue, Red, Green, Blue)\n\n"+
		"export pure func code(h: Hue) -> int = match h {\n  Red => 1,\n  Green => 2,\n  Blue => 3\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, run := range []string{"cold", "warm"} {
		if _, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: "main.ail"}, productionCacheDependencies()); err != nil {
			t.Errorf("%s: %v", run, err)
		}
	}
}
