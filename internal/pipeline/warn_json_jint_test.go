package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
)

// TC_MATCH_005: a match over std/json's Json with a JNumber arm and no JInt
// arm (M-V1-STDLIB-SURFACE-POLISH §7). Each case is a whole module checked
// through the module pipeline, with the compile cache on.

const jsonImports = "import std/json (Json, JNull, JNumber, JInt, JString, JArray, decode, get)\n" +
	"import std/option (Option, Some, None)\n" +
	"import std/list (flatMap)\n"

func jintWarnings(ws []elaborate.Warning) []*JSONIntArmWarning {
	var out []*JSONIntArmWarning
	for _, w := range ws {
		if jw, ok := w.(*JSONIntArmWarning); ok {
			out = append(out, jw)
		}
	}
	return out
}

func checkJIntModule(t *testing.T, name, src string) []*JSONIntArmWarning {
	t.Helper()
	if err := os.WriteFile(name+".ail", []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: name + ".ail"}, productionCacheDependencies())
	if err != nil {
		t.Fatalf("%s: warning must not block compilation: %v", name, err)
	}
	return jintWarnings(res.Warnings)
}

func TestJSONIntArmWarning(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))

	cases := []struct {
		name string
		body string
		want int
		line int // line of the warned match, when want == 1
		// noCatchAll: the warning must say an integer fails at run time (RT002)
		noCatchAll bool
	}{
		{"catchall", jsonImports +
			"export pure func f(j: Json) -> float = match j {\n  JNumber(n) => n,\n  _ => 0.0\n}\n", 1, 5, false},
		{"nested", jsonImports +
			"export pure func f(j: Json) -> Option[float] = match get(j, \"k\") {\n  Some(JNumber(n)) => Some(n),\n  _ => None\n}\n", 1, 5, false},
		{"inLambda", jsonImports +
			"export pure func f(xs: [Json]) -> [float] =\n  flatMap(\\x. match x { JNumber(n) => [n], _ => [] }, xs)\n", 1, 6, false},
		{"hasJInt", jsonImports +
			"export pure func f(j: Json) -> float = match j {\n  JNumber(n) => n,\n  JInt(i) => _int_to_float(i),\n  _ => 0.0\n}\n", 0, 0, false},
		{"noNumberArm", jsonImports +
			"export pure func f(j: Json) -> string = match j {\n  JString(s) => s,\n  _ => \"\"\n}\n", 0, 0, false},
		{"aliased", "import std/json (Json, JNumber as Num)\n" +
			"export pure func f(j: Json) -> float = match j {\n  Num(n) => n,\n  _ => 0.0\n}\n", 1, 3, false},
		{"localCtor", "type Reading = JNumber(float) | Missing\n" +
			"export pure func f(r: Reading) -> float = match r {\n  JNumber(n) => n,\n  _ => 0.0\n}\n", 0, 0, false},
		{"localJson", "type Json = JNumber(float) | JText(string)\n" +
			"export pure func f(j: Json) -> float = match j {\n  JNumber(n) => n,\n  _ => 0.0\n}\n", 0, 0, false},
		// A JInt elsewhere does not cover the top-level JNumber position.
		{"jintElsewhere", jsonImports +
			"export pure func f(j: Json) -> string = match j {\n  JNumber(n) => \"num\",\n  JArray([JInt(i)]) => \"one int\",\n  _ => \"other\"\n}\n", 1, 5, false},
		// Same position, nested: Some(JInt(i)) covers Some(JNumber(n)).
		{"nestedCovered", jsonImports +
			"export pure func f(j: Json) -> int = match get(j, \"k\") {\n  Some(JNumber(n)) => 1,\n  Some(JInt(i)) => 2,\n  _ => 0\n}\n", 0, 0, false},
		// No catch-all: an integer is a run-time RT002, not a fall-through.
		{"exhaustiveNoJInt", jsonImports +
			"export pure func f(j: Json) -> string = match j {\n  JNull => \"null\",\n  JBool(_) => \"bool\",\n  JNumber(_) => \"number\",\n  JString(_) => \"string\",\n  JArray(_) => \"array\",\n  JObject(_) => \"object\"\n}\n", 1, 5, true},
		{"nestedNoCatchAll", jsonImports +
			"export pure func f(o: Option[Json]) -> int = match o {\n  Some(JNumber(n)) => 1,\n  None => 0\n}\n", 1, 5, true},
		{"nestedVarCatchAll", jsonImports +
			"export pure func f(o: Option[Json]) -> int = match o {\n  Some(JNumber(n)) => 1,\n  Some(other) => 2,\n  None => 0\n}\n", 1, 5, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := "module " + c.name + "\n" + c.body
			for _, run := range []string{"cold", "warm"} {
				ws := checkJIntModule(t, c.name, src)
				if len(ws) != c.want {
					t.Fatalf("%s run: %d TC_MATCH_005 warnings, want %d: %v", run, len(ws), c.want, ws)
				}
				if c.want == 1 {
					w := ws[0]
					if line, _ := w.Position(); line != c.line {
						t.Errorf("%s run: warning at line %d, want %d", run, line, c.line)
					}
					if w.NoCatchAll != c.noCatchAll || strings.Contains(w.Message(), "RT002") != c.noCatchAll {
						t.Errorf("%s run: NoCatchAll=%v, want %v: %s", run, w.NoCatchAll, c.noCatchAll, w.Message())
					}
					if w.Code() != "TC_MATCH_005" || !strings.Contains(w.String(), "JInt") ||
						!strings.HasSuffix(filepath.ToSlash(w.WarningFile()), c.name+".ail") || w.Suggestion() == "" {
						t.Errorf("%s run: malformed warning: code=%q file=%q\n%s", run, w.Code(), w.WarningFile(), w.String())
					}
				}
			}
		})
	}
}

// A module that imports a JNumber from some other module than std/json is not
// matching std/json's constructor.
func TestJSONIntArmWarning_ForeignJNumber(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
	if err := os.WriteFile("units.ail", []byte("module units\nexport type Reading = JNumber(float) | Missing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws := checkJIntModule(t, "user", "module user\nimport units (Reading, JNumber, Missing)\n"+
		"export pure func f(r: Reading) -> float = match r {\n  JNumber(n) => n,\n  _ => 0.0\n}\n")
	if len(ws) != 0 {
		t.Fatalf("foreign JNumber warned: %v", ws)
	}
}

// std/json itself, std/jwt and std/sem carry JInt arms; checking a module that
// imports all three must not warn.
func TestJSONIntArmWarning_StdlibIsClean(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
	ws := checkJIntModule(t, "usesstd", "module usesstd\nimport std/json (Json, decode)\nimport std/jwt (decodeJWT)\nimport std/sem (decode_frame)\n"+
		"export pure func f(s: string) -> bool = true\n")
	if len(ws) != 0 {
		for _, w := range ws {
			t.Errorf("stdlib warns: %s", w)
		}
	}
}

// The walker reaches matches inside tuples, arrays and record updates, which
// walkCore does not descend into.
func TestDetectJSONIntArm_ReachesTupleArrayRecordUpdate(t *testing.T) {
	m := func() core.CoreExpr {
		return &core.Match{
			Scrutinee: &core.Var{Name: "j"},
			Arms: []core.MatchArm{
				{Pattern: &core.ConstructorPattern{Name: "JNumber", TypeName: "Json", Args: []core.CorePattern{&core.VarPattern{Name: "n"}}}, Body: &core.Var{Name: "n"}},
				{Pattern: &core.WildcardPattern{}, Body: &core.Var{Name: "z"}},
			},
		}
	}
	lam := func() core.CoreExpr { return &core.Lambda{Params: []string{"j"}, Body: m()} }
	prog := &core.Program{Decls: []core.CoreExpr{
		&core.Tuple{Elements: []core.CoreExpr{lam()}},
		&core.Array{Elements: []core.CoreExpr{lam()}},
		&core.RecordUpdate{Base: lam(), Updates: map[string]core.CoreExpr{"f": lam()}},
	}}
	ws := DetectJSONIntArm(&CompileUnit{ID: "m", Core: prog})
	if len(ws) != 4 {
		t.Fatalf("got %d warnings, want 4", len(ws))
	}
	if DetectJSONIntArm(nil) != nil || DetectJSONIntArm(&CompileUnit{ID: "m"}) != nil {
		t.Fatal("nil unit/core must yield no warnings")
	}
}
