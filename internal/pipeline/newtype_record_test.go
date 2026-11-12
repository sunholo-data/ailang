package pipeline

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// #2239: a newtype-record ADT is nominal — never its record — across modules
// too, and with its dependencies served from the compile cache. The
// one-module rows (the issue's repros, the laundering holes, and the field
// access controls) are internal/types/field_access_deferred_pipeline_test.go.
func newtypeRecordCases() []nominalCase {
	return []nominalCase{
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

func TestNewtypeRecord_RowsWarm(t *testing.T) { runNominalRowsWarm(t, newtypeRecordCases()) }
