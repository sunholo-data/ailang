package pipeline

import (
	"strings"
	"testing"
)

const closureA = `module a
export type Item = {x: int, w: int}
export type Box = {items: [Item]}
export pure func mk() -> Box { {items: [{x: 1, w: 2}]} }
export pure func count(b: Box) -> int { match b.items { [] => 0, _ :: _ => 1 } }
export pure func mkItem() -> Item { {x: 1, w: 2} }
export pure func countItem(i: Item) -> int { i.x }
`
const closureB = `module b
export type Item = {y: string}
export pure func one() -> Item { {y: "b"} }
`

func TestAliasBodyClosure_ImportMatrix(t *testing.T) {
	for _, imports := range []string{
		"import a (Box, mk, count, mkItem, countItem)\nimport b (Item, one)",
		"import b (one)\nimport a (Box, mk, count, mkItem, countItem)",
		"import a (Box, mk, count, mkItem, countItem)\nimport b (one)",
	} {
		for _, expr := range []string{"count(mk())", "countItem(mkItem())"} {
			t.Run(imports+expr, func(t *testing.T) {
				files := map[string]string{"a.ail": closureA, "b.ail": closureB,
					"main.ail": "module main\n" + imports + "\nexport pure func main() -> int { let y = one().y; " + expr + " }"}
				if err := checkModules(t, files); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestAliasBodyClosure_NavigationSolTrappistShape(t *testing.T) {
	files := map[string]string{
		"sol.ail": `module sol
export type Planet = {mass: int, orbit: int}
export type System = {planets: [Planet]}
export pure func solSystem() -> System { {planets: [{mass: 1, orbit: 2}]} }
`,
		"navigation.ail": `module navigation
import sol (System)
export pure func trappistFacts(s: System) -> int { match s.planets { [] => 0, p :: _ => p.mass } }
`,
		"trappist.ail": `module trappist
export type Planet = {name: string}
export pure func planet() -> Planet { {name: "b"} }
`,
		"main.ail": `module main
import navigation (trappistFacts)
import sol (solSystem)
import trappist (Planet, planet)
export pure func main() -> int { trappistFacts(solSystem()) }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatal(err)
	}
}

func TestAliasBodyClosure_UnexportedDependencyAndLocalPrecedence(t *testing.T) {
	a := strings.Replace(closureA, "export type Item", "type Item", 1)
	a = strings.Replace(a, "module a", "module a\nimport b (one)", 1)
	files := map[string]string{"a.ail": a, "b.ail": closureB,
		"main.ail": `module main
import a (Box, mk, count)
import b (Item, one)
export pure func main() -> int { count(mk()) }
`}
	if err := checkModules(t, files); err != nil {
		t.Fatal(err)
	}
}

func TestAliasBodyClosure_IndependentRecordsStayDistinct(t *testing.T) {
	files := map[string]string{"a.ail": closureA, "b.ail": closureB,
		"main.ail": `module main
import a (countItem)
import b (Item, one)
export pure func main() -> int { countItem(one()) }
`}
	if err := checkModules(t, files); err == nil {
		t.Fatal("unrelated record shapes must not unify")
	}
}

func TestAliasBodyClosure_ConstructorFieldsAndPatterns(t *testing.T) {
	a := closureA + `
export type Wrapped = Wrap(Item)
export pure func wrapped() -> Wrapped { Wrap(mkItem()) }
`
	files := map[string]string{"a.ail": a, "b.ail": closureB, "main.ail": `module main
import a (Wrapped, Wrap, wrapped)
import b (Item, one)
export pure func main() -> int { match wrapped() { Wrap(i) => i.x } }
`}
	if err := checkModules(t, files); err != nil {
		t.Fatal(err)
	}
}

// M-V1-NOMINAL-TYPE-IDENTITY replaced M-TYPE-NAME-SHADOW's residual-capture
// error (TC_TYPE_SHADOW_001): an imported alias body names a.Item, which a
// local `type Item` can no longer capture, so using the alias is legal and
// expands to a's Item. Putting the LOCAL Item where a's is expected is still
// rejected, now as a nominal mismatch.
func TestAliasBodyClosure_ImportedBodyNotCapturedByLocalName(t *testing.T) {
	a := `module a
export type Item = Empty | Full(int)
export type Box = {items: [Item]}
export pure func mk() -> Box { {items:[Full(1)]} }
`
	ok := map[string]string{"a.ail": a, "main.ail": `module main
import a (Box, mk)
type Item = {y: string}
export pure func main(b: Box) -> int { match b.items { [] => 0, _ :: _ => 1 } }
`}
	if err := checkModules(t, ok); err != nil {
		t.Fatalf("an imported alias whose body names a.Item must be usable next to a local Item: %v", err)
	}
	bad := map[string]string{"a.ail": a, "main.ail": `module main
import a (Box, mk)
type Item = {y: string}
pure func mine() -> Item { {y: "s"} }
export pure func main() -> Box { {items: [mine()]} }
`}
	err := checkModules(t, bad)
	if err == nil {
		t.Fatal("a local Item must not be accepted where a's Item is expected")
	}
	if !strings.Contains(err.Error(), "Item") {
		t.Fatalf("diagnostic should name Item: %v", err)
	}
}
