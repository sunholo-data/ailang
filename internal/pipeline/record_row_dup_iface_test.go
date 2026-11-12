package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/iface"
)

// PR #2288 review, round 3: a duplicate label (a field a record has, and again
// in the row its tail is bound to) that has been unified and agrees must be
// folded, so no consumer sees the field twice. Before, `h` below exported
// `(())->{x: int, ...<*types.Row>}`: interface_hash v2 hashes that rendered
// string, so every concrete row hashed alike.
const dupexpLib = `module lib/dupexp

export func f(p: {y: int | r}, q: {x: int | r}) -> string = p.x

export func g(p: {y: int | r}, q: {x: int | r}) -> {x: int | r} = q

export func h() = g({y: 1, x: 5}, {x: 2})

export func k() -> int = h().x
`

// The same module with h's return type written out.
const dupexpAnnotated = `module lib/dupexp

export func f(p: {y: int | r}, q: {x: int | r}) -> string = p.x

export func g(p: {y: int | r}, q: {x: int | r}) -> {x: int | r} = q

export func h() -> {x: int} = g({y: 1, x: 5}, {x: 2})

export func k() -> int = h().x
`

// ifaceOf checks one module (cache off) and returns its normalized iface JSON.
func ifaceOf(t *testing.T, path, src string) []byte {
	t.Helper()
	res, err := writeAndCheck(t, map[string]string{path: src}, path)
	if err != nil {
		t.Fatalf("check %s: %v", path, err)
	}
	data, err := res.Interface.ToNormalizedJSON()
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func renderedTypes(t *testing.T, data []byte) map[string]string {
	t.Helper()
	var j iface.InterfaceJSON
	if err := json.Unmarshal(data, &j); err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range j.Funcs {
		out[f.Name] = f.Type
	}
	return out
}

func hashProjection(t *testing.T, data []byte) string {
	t.Helper()
	var j iface.InterfaceJSON
	if err := json.Unmarshal(data, &j); err != nil {
		t.Fatal(err)
	}
	p, err := iface.HashProjection(&j)
	if err != nil {
		t.Fatal(err)
	}
	return string(p)
}

func TestRecordRows_AgreeingDuplicateIsFoldedInTheInterface(t *testing.T) {
	data := ifaceOf(t, "lib/dupexp.ail", dupexpLib)
	types := renderedTypes(t, data)
	if got := types["h"]; got != "(())->{x: int}" {
		t.Fatalf("h renders %q, want (())->{x: int}", got)
	}
	// f is uncallable (its q has x as int and, through r, as string): that
	// duplicate is never unified, so it stays unfolded and must still render
	// structurally, not as <*types.Row>.
	if strings.Contains(string(data), "<*types.") {
		t.Fatalf("interface leaks a Go type name:\n%s", data)
	}
	// interface_hash v2 hashes the rendered type string (HashProjection keeps
	// FuncJSON.Type), so h must hash exactly as h written `-> {x: int}`.
	if a, b := hashProjection(t, data), hashProjection(t, ifaceOf(t, "lib/dupexp.ail", dupexpAnnotated)); a != b {
		t.Fatalf("hash projection differs from the annotated h:\n%s\n%s", a, b)
	}
	// Deterministic over repeated cold checks.
	for i := 0; i < 10; i++ {
		if again := ifaceOf(t, "lib/dupexp.ail", dupexpLib); string(again) != string(data) {
			t.Fatalf("run %d: iface differs:\n%s\n%s", i, again, data)
		}
	}
}

// An importer sees the same verdicts cold and warm: h().x is an int, h().y does
// not exist.
func TestRecordRows_DuplicateExportColdAndWarm(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("AILANG_CACHE_DIR", cacheDir)
	t.Setenv("AILANG_NO_CACHE", "")
	good := "module app/good\n\nimport lib/dupexp (h)\n\nexport func main() -> () ! {IO} = println(show(h().x + 1))\n"
	bad := "module app/bad\n\nimport lib/dupexp (h)\n\nexport func main() -> () ! {IO} = println(show(h().y))\n"
	dir := t.TempDir()
	for rel, src := range map[string]string{"lib/dupexp.ail": dupexpLib, "app/good.ail": good, "app/bad.ail": bad} {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
	for _, run := range []string{"cold", "warm"} {
		if _, err := Run(Config{Mode: ModeCheck}, Source{Filename: "app/good.ail"}); err != nil {
			t.Fatalf("%s: good importer rejected: %v", run, err)
		}
		_, err := Run(Config{Mode: ModeCheck}, Source{Filename: "app/bad.ail"})
		if err == nil || !strings.Contains(err.Error(), "field 'y' not found") {
			t.Fatalf("%s: h().y accepted or mis-reported: %v", run, err)
		}
	}
	if len(collectIfaceJSON(t, cacheDir)) == 0 {
		t.Fatal("nothing was cached: the warm run measured nothing")
	}
}

// A duplicate in parameter and result position (dup2.g3), used at two record
// shapes, and a wrong use of its result; and a local type error does not show
// the field twice (c21).
func TestRecordRows_DuplicateInParamAndResultPositions(t *testing.T) {
	lib := `module lib/dup2

export func g3(p: {y: int | r}, q: {x: int | r}) -> {x: int | r} = { let a = p.x + 1; q }
`
	ok := `module app/two

import lib/dup2 (g3)

export func main() -> () ! {IO} {
  println(g3({y: 1, x: 5, z: "a"}, {x: 2, z: "b"}).z);
  println(show(g3({y: 1, x: 5, w: true}, {x: 2, w: false}).w))
}
`
	bad := `module app/bad

import lib/dup2 (g3)

export func main() -> () ! {IO} = println(show(g3({y: 1, x: 5, z: "a"}, {x: 2, z: "b"}).z + 1))
`
	if _, err := writeAndCheck(t, map[string]string{"lib/dup2.ail": lib, "app/two.ail": ok}, "app/two.ail"); err != nil {
		t.Fatalf("g3 at two shapes rejected: %v", err)
	}
	if _, err := writeAndCheck(t, map[string]string{"lib/dup2.ail": lib, "app/bad.ail": bad}, "app/bad.ail"); err == nil || !strings.Contains(err.Error(), "Num[string]") {
		t.Fatalf("g3(...).z + 1 accepted or mis-reported: %v", err)
	}
	c21 := `module m

func g(p: {y: int | r}, q: {x: int | r}) -> {x: int | r} = q

export func main() -> () ! {IO} = println(g({y: 1, x: 5}, {x: 2}))
`
	err := checkModule(t, c21)
	if err == nil {
		t.Fatal("println of a record accepted")
	}
	if msg := err.Error(); strings.Count(msg, "x: int") != 1 {
		t.Fatalf("the field is printed more than once: %v", msg)
	}
}
