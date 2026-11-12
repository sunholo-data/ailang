package types_test

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// #2239: a newtype-record ADT (`type W = W({ x: int })`) is nominal. It never
// unifies with a record — neither a bare record where a W is expected nor a W
// where a record is — so two same-shaped newtypes cannot reach each other
// through a record-typed value. Field access `w.x` keeps working: it is a
// projection decided once the receiver's type is known
// (field_access_deferred.go), not a conversion. Seen through the pipeline so
// the deferred access, its scheme copies and TC_NEWTYPE_001 are exercised in
// this package; the cross-module rows (cold and warm cache) are in
// internal/pipeline/newtype_record_test.go.

const ntHdr = "module m/main\nimport std/list (map, foldl)\n" +
	"type W = W({ x: int, y: int })\ntype V = V({ x: int, y: int })\n" +
	"pure func getw(w: W) -> int = match w { W(r) => r.x * 3 }\n"

func TestNewtypeRecord_FieldAccessIsAProjection(t *testing.T) {
	reject := types.TC_NEWTYPE_001
	for _, c := range []struct {
		name, body, want string // want "" = must check
	}{
		// --- the issue's repros and the holes the old expansion left ---
		{"pnt3: a bare record where a W is expected",
			"export func main() -> () ! {IO} = println(show(getw({ x: 5, y: 1 })))\n", reject},
		{"pnt2: V reaches W through a record-typed function",
			"pure func launder(r: { x: int, y: int }) -> { x: int, y: int } = r\n" +
				"export func main() -> () ! {IO} = println(show(getw(launder(V({ x: 1000, y: 0 })))))\n", reject},
		{"a W returned as its record",
			"pure func toRec(w: W) -> { x: int, y: int } = w\nexport pure func main() -> int = toRec(W({ x: 1, y: 2 })).x\n", reject},
		{"a let annotated W bound to a record",
			"export pure func main() -> int { let w: W = { x: 1, y: 2 }; w.x }\n", reject},
		{"a lambda that reads a field and passes its argument as a W, called with a record",
			"export pure func main() -> int { let f = \\r. { let _ = r.x; getw(r) }; f({ x: 5, y: 1 }) }\n", reject},
		{"a field-reading lambda applied to a V and a W",
			"export pure func main() -> int { let pick = \\a b. if a.x > 0 then a else b; getw(pick(V({ x: 1, y: 2 }), W({ x: 3, y: 4 }))) }\n",
			"V vs W"},
		{"a let-bound field reader returning its argument launders a record into a W",
			"export pure func main() -> int { let f = \\r. { let _ = r.x; r }; getw(f({ x: 5, y: 1 })) }\n", reject},
		{"a let-bound field reader keeps its access at every use (a string has no field)",
			"export pure func main() -> int { let gx = \\t. t.x; gx(W({ x: 1, y: 2 })) + gx(\"s\") }\n", "field access at"},
		{"record update on a W (crashed at run time on both engines)",
			"export pure func main() -> int { let w = W({ x: 1, y: 2 }); let w2 = { w | x: 9 }; w2.x }\n", reject},
		{"a field the newtype's record does not have",
			"export pure func main() -> int = W({ x: 1, y: 2 }).z\n", "field `z` of W"},
		// --- controls: field access on a newtype keeps working ---
		{"field access on an annotated W",
			"pure func sumw(w: W) -> int = w.x + w.y\nexport pure func main() -> int = sumw(W({ x: 3, y: 4 }))\n", ""},
		{"field access on a let-bound W",
			"export pure func main() -> int { let w = W({ x: 1, y: 2 }); w.x + w.y }\n", ""},
		{"prompt example: map(\\t. t.x, ws)",
			"export func main() -> () ! {IO} = println(show(map(\\t. t.x, [W({ x: 1, y: 2 }), W({ x: 3, y: 4 })])))\n", ""},
		{"a let-bound field reader used twice at W",
			"export func main() -> () ! {IO} { let gx = \\t. t.x; println(show(map(gx, [W({ x: 1, y: 2 })]))); println(show(gx(W({ x: 7, y: 0 })))) }\n", ""},
		{"a let-bound field reader stays polymorphic over records and newtypes",
			"export pure func main() -> int { let gx = \\t. t.x; gx(W({ x: 1, y: 2 })) + gx({ x: 3, z: true }) + gx(V({ x: 5, y: 0 })) }\n", ""},
		{"foldl picking the best W by a field",
			"export pure func main() -> int { let best = foldl(\\acc w. if w.x > acc.x then w else acc, W({ x: 0, y: 0 }), [W({ x: 2, y: 1 }), W({ x: 5, y: 1 })]); getw(best) }\n", ""},
		{"match W(r) and the record inside",
			"export pure func main() -> int = getw(W({ x: 5, y: 1 }))\n", ""},
		{"records and record aliases stay structural",
			"type P = { x: int, y: int }\npure func gp(p: P) -> int = p.x\n" +
				"export pure func main() -> int { let r = \\q. q.x + q.y; gp({ x: 1, y: 2 }) + r({ x: 3, y: 4 }) }\n", ""},
		{"a field reader at a top-level declaration is still a record reader",
			"pure func getx(r: { x: int, y: int }) -> int = r.x\nexport pure func main() -> int = getx({ x: 1, y: 2 })\n", ""},
		{"an unannotated field reader in its own declaration is settled as a record reader",
			"export pure func main() -> int { let k = 1; let f = \\r. r.y + k; f({ y: 2 }) }\n", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := checkModules(t, map[string]string{"m/main.ail": ntHdr + c.body})
			if c.want == "" {
				if err != nil {
					t.Fatalf("must type-check: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("accepted; want an error containing %q", c.want)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Fatalf("error must contain %q:\n%v", c.want, err)
			}
		})
	}
}
