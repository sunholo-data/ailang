package pipeline

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// #2239: a newtype-record ADT (`type W = W({ x: int })`) is nominal. It never
// unifies with a record — neither a bare record where a W is expected nor a W
// where a record is — so two same-shaped newtypes cannot reach each other
// through a record-typed value. Field access `w.x` keeps working: it is a
// projection decided once the receiver's type is known
// (types/field_access_deferred.go), not a conversion.

const ntHdr = "type W = W({ x: int, y: int })\ntype V = V({ x: int, y: int })\n" +
	"pure func getw(w: W) -> int = match w { W(r) => r.x * 3 }\n"

// ntOne is a one-module case of the #2239 table.
func ntOne(name, body string, reject ...string) nominalCase {
	src := "module nt1/main\nimport std/list (map, foldl)\n" + ntHdr + body
	return nominalCase{name: name, files: map[string]string{"nt1/main.ail": src}, entry: "nt1/main.ail", reject: reject}
}

func newtypeRecordCases() []nominalCase {
	reject := []string{types.TC_NEWTYPE_001}
	return []nominalCase{
		// --- the issue's repros and the holes the old expansion left ---
		ntOne("pnt3: a bare record where a W is expected",
			"export func main() -> () ! {IO} = println(show(getw({ x: 5, y: 1 })))\n", reject...),
		ntOne("pnt2: V reaches W through a record-typed function",
			"pure func launder(r: { x: int, y: int }) -> { x: int, y: int } = r\n"+
				"export func main() -> () ! {IO} = println(show(getw(launder(V({ x: 1000, y: 0 })))))\n", reject...),
		ntOne("a W returned as its record",
			"pure func toRec(w: W) -> { x: int, y: int } = w\nexport pure func main() -> int = toRec(W({ x: 1, y: 2 })).x\n", reject...),
		ntOne("a let annotated W bound to a record",
			"export pure func main() -> int { let w: W = { x: 1, y: 2 }; w.x }\n", reject...),
		ntOne("a lambda that reads a field and passes its argument as a W, called with a record",
			"export pure func main() -> int { let f = \\r. { let _ = r.x; getw(r) }; f({ x: 5, y: 1 }) }\n", reject...),
		ntOne("a field-reading lambda applied to a V and a W",
			"export pure func main() -> int { let pick = \\a b. if a.x > 0 then a else b; getw(pick(V({ x: 1, y: 2 }), W({ x: 3, y: 4 }))) }\n",
			"V vs W"),
		ntOne("a let-bound field reader returning its argument launders a record into a W",
			"export pure func main() -> int { let f = \\r. { let _ = r.x; r }; getw(f({ x: 5, y: 1 })) }\n", reject...),
		ntOne("a let-bound field reader keeps its access at every use (a string has no field)",
			"export pure func main() -> int { let gx = \\t. t.x; gx(W({ x: 1, y: 2 })) + gx(\"s\") }\n", "field access at"),
		ntOne("record update on a W (crashed at run time on both engines)",
			"export pure func main() -> int { let w = W({ x: 1, y: 2 }); let w2 = { w | x: 9 }; w2.x }\n", reject...),
		ntOne("a field the newtype's record does not have",
			"export pure func main() -> int = W({ x: 1, y: 2 }).z\n", "field `z` of W"),
		// --- controls: field access on a newtype keeps working ---
		ntOne("field access on an annotated W",
			"pure func sumw(w: W) -> int = w.x + w.y\nexport pure func main() -> int = sumw(W({ x: 3, y: 4 }))\n"),
		ntOne("field access on a let-bound W",
			"export pure func main() -> int { let w = W({ x: 1, y: 2 }); w.x + w.y }\n"),
		ntOne("prompt example: map(\\t. t.x, ws)",
			"export func main() -> () ! {IO} = println(show(map(\\t. t.x, [W({ x: 1, y: 2 }), W({ x: 3, y: 4 })])))\n"),
		ntOne("a let-bound field reader used twice at W",
			"export func main() -> () ! {IO} { let gx = \\t. t.x; println(show(map(gx, [W({ x: 1, y: 2 })]))); println(show(gx(W({ x: 7, y: 0 })))) }\n"),
		ntOne("a let-bound field reader stays polymorphic over records and newtypes",
			"export pure func main() -> int { let gx = \\t. t.x; gx(W({ x: 1, y: 2 })) + gx({ x: 3, z: true }) + gx(V({ x: 5, y: 0 })) }\n"),
		ntOne("foldl picking the best W by a field",
			"export pure func main() -> int { let best = foldl(\\acc w. if w.x > acc.x then w else acc, W({ x: 0, y: 0 }), [W({ x: 2, y: 1 }), W({ x: 5, y: 1 })]); getw(best) }\n"),
		ntOne("match W(r) and the record inside",
			"export pure func main() -> int = getw(W({ x: 5, y: 1 }))\n"),
		ntOne("records and record aliases stay structural",
			"type P = { x: int, y: int }\npure func gp(p: P) -> int = p.x\n"+
				"export pure func main() -> int { let r = \\q. q.x + q.y; gp({ x: 1, y: 2 }) + r({ x: 3, y: 4 }) }\n"),
		ntOne("a generic field reader at a top-level declaration is still a record reader",
			"pure func getx(r: { x: int, y: int }) -> int = r.x\nexport pure func main() -> int = getx({ x: 1, y: 2 })\n"),
		// --- across modules (also run warm: dependencies served from the cache) ---
		{
			name: "pnt: a's W reaches b's W through a record (cross-module)",
			files: map[string]string{
				"pnt/a.ail":     "module pnt/a\nexport type W = W({ x: int })\nexport pure func mk() -> W = W({ x: 1 })\n",
				"pnt/b.ail":     "module pnt/b\nexport type W = W({ x: int })\nexport pure func getb(w: W) -> int = match w { W(r) => r.x + 100 }\n",
				"pnt/main.ail":  "module pnt/main\nimport pnt/a (mk)\nimport pnt/b (getb)\npure func launder(r: { x: int }) -> { x: int } = r\nexport func main() -> () ! {IO} = println(show(getb(launder(mk()))))\n",
				"pnt/prime.ail": "module pnt/prime\nimport pnt/a (mk)\nimport pnt/b (getb)\nexport pure func main() -> int = 1\n",
			},
			entry: "pnt/main.ail", prime: "pnt/prime.ail",
			reject: []string{types.TC_NEWTYPE_001, "W (defined in pnt/a)"},
		},
		{
			name: "an imported W's fields through an inferred lambda (cross-module)",
			files: map[string]string{
				"ntx/a.ail":     "module ntx/a\nexport type W = W({ x: int })\nexport pure func mk(n: int) -> W = W({ x: n })\n",
				"ntx/main.ail":  "module ntx/main\nimport std/list (map)\nimport ntx/a (mk)\nexport pure func main() -> [int] = map(\\w. w.x, [mk(1), mk(2)])\n",
				"ntx/prime.ail": "module ntx/prime\nimport ntx/a (mk)\nexport pure func main() -> int = 1\n",
			},
			entry: "ntx/main.ail", prime: "ntx/prime.ail",
		},
	}
}

func TestNewtypeRecord_RowsCold(t *testing.T) { runNominalRowsCold(t, newtypeRecordCases()) }

func TestNewtypeRecord_RowsWarm(t *testing.T) {
	var warm []nominalCase
	for _, c := range newtypeRecordCases() {
		if c.prime != "" {
			warm = append(warm, c)
		}
	}
	runNominalRowsWarm(t, warm)
}
