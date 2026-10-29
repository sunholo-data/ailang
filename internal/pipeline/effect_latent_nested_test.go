package pipeline

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// M-EFFECT-LATENT-NESTED-CALLBACKS (#1718, #1826).
//
// #1718: a function value nested in a list/tuple/Option/record argument was paid
// by nobody — the callee body erased its implicit tail while the call site only
// charged top-level arrow parameters. #1826: a concrete annotation's L1-open
// compatibility tail escaped into an App row (`Unresolved effect tail`, or the
// ApplicationEffects invariant when no binder owned it).

const (
	nestedImports = "module main\nimport std/io (println)\nimport std/option (Option, Some, None)\n"
	nestedDecls   = "func logIt(x: int) -> int ! {IO} = { println(\"side effect\"); x }\npure func inc(x: int) -> int = x + 1\n"
)

// nestedSource builds a module from a case body, hoisting the body's import
// lines above the shared declarations.
func nestedSource(module, body string) string {
	var imports, decls []string
	for _, line := range strings.SplitAfter(body, "\n") {
		if strings.HasPrefix(line, "import ") {
			imports = append(imports, line)
		} else {
			decls = append(decls, line)
		}
	}
	header := strings.Replace(nestedImports, "module main", "module "+module, 1)
	return header + strings.Join(imports, "") + nestedDecls + strings.Join(decls, "")
}

const (
	nestedApplyFst  = "func applyFst(p: ((int) -> int, int)) -> int = match p { (f, x) => f(x) }\n"
	nestedApplyHead = "func applyHead(fs: [(int) -> int], x: int) -> int = match fs { [] => x, [f, ..._] => f(x) }\n"
	nestedApplyOpt  = "func applyOpt(o: Option[(int) -> int], x: int) -> int = match o { Some(f) => f(x), None => x }\n"
	nestedApplyBoth = "func applyBoth(p: ((int) -> int, (int) -> int), x: int) -> int = match p { (f, g) => f(g(x)) }\n"
)

type nestedCase struct {
	name  string
	body  string            // main.ail declarations (see nestedSource)
	extra map[string]string // other modules, relpath -> source
	blame string            // "" = must check clean
	label string            // missing label for a rejection (default IO)
}

func runNestedCases(t *testing.T, cases []nestedCase) {
	t.Helper()
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{"main.ail": nestedSource("main", tt.body)}
			for k, v := range tt.extra {
				files[k] = v
			}
			err := checkModules(t, files)
			if err != nil && (strings.Contains(err.Error(), "internal invariant") || strings.Contains(err.Error(), "Unresolved effect tail")) {
				t.Fatalf("checker leaked an internal tail/invariant: %v", err)
			}
			if tt.blame == "" {
				if err != nil {
					t.Fatalf("want clean check: %v", err)
				}
				return
			}
			label := tt.label
			if label == "" {
				label = "IO"
			}
			if err == nil {
				t.Fatalf("accepted undeclared %s", label)
			}
			for _, want := range []string{"Effect checking failed for function '" + tt.blame + "'", "Missing effects: " + label} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("want %q: %v", want, err)
				}
			}
		})
	}
}

// AC1 — #1718 leak shapes are charged to the pure caller.
func TestLatentNested_LeaksRejected(t *testing.T) {
	runNestedCases(t, []nestedCase{
		{name: "R1_tuple", body: nestedApplyFst + "pure func sneaky(x: int) -> int = applyFst((logIt, x))\n", blame: "sneaky"},
		{name: "R2_list", body: nestedApplyHead + "pure func sneaky(x: int) -> int = applyHead([logIt], x)\n", blame: "sneaky"},
		{name: "R3_option_unannotated", body: "func applyOpt(o, x: int) -> int = match o { Some(f) => f(x), None => x }\npure func sneaky(x: int) -> int = applyOpt(Some(logIt), x)\n", blame: "sneaky"},
		{name: "R4_record", body: "func applyRec(p: {f: (int) -> int, x: int}) -> int = p.f(p.x)\npure func sneaky(x: int) -> int = applyRec({f: logIt, x: x})\n", blame: "sneaky"},
		{name: "R5_record_alias_cross_module", body: "import x/librec (applyRec)\npure func sneaky(x: int) -> int = applyRec({f: logIt, x: x})\n",
			extra: map[string]string{"x/librec.ail": "module x/librec\nexport type R = {f: (int) -> int, x: int}\nexport func applyRec(p: R) -> int = p.f(p.x)\n"}, blame: "sneaky"},
		{name: "R6_let_bound_tuple", body: nestedApplyFst + "pure func sneaky(x: int) -> int = { let t = (logIt, x); applyFst(t) }\n", blame: "sneaky"},
		{name: "R7_returned_list", body: nestedApplyHead + "func mk() -> [(int) -> int] = [logIt]\npure func sneaky(x: int) -> int = applyHead(mk(), x)\n", blame: "sneaky"},
		{name: "R8_adt_holding_list", body: "type Reg = Reg([(int) -> int])\nfunc applyReg(r: Reg, x: int) -> int = match r { Reg(fs) => match fs { [] => x, [f, ..._] => f(x) } }\npure func sneaky(x: int) -> int = applyReg(Reg([logIt]), x)\n", blame: "sneaky"},
		{name: "R9_list_of_tuples", body: "func applyAll(fs: [((int) -> int, int)]) -> int = match fs { [] => 0, [(f, x), ..._] => f(x) }\npure func sneaky(x: int) -> int = applyAll([(logIt, x)])\n", blame: "sneaky"},
		{name: "R1_cross_module_tuple_list", body: "import x/lib (applyFst, applyHead)\npure func sneaky(x: int) -> int = applyFst((logIt, x)) + applyHead([logIt], x)\n",
			extra: map[string]string{"x/lib.ail": "module x/lib\nexport " + nestedApplyFst + "export " + nestedApplyHead}, blame: "sneaky"},
		{name: "R10_cross_module_option", body: "import x/libr (applyOpt)\npure func sneaky(x: int) -> int = applyOpt(Some(logIt), x)\n",
			extra: map[string]string{"x/libr.ail": "module x/libr\nimport std/option (Option, Some, None)\nexport " + nestedApplyOpt}, blame: "sneaky"},
	})
}

// AC2 — annotated generic containers are valid definitions and charge at use.
func TestLatentNested_AnnotatedGenericContainers(t *testing.T) {
	const holder = "type Holder[a] = Hold(a)\nfunc applyHold(h: Holder[(int) -> int], x: int) -> int = match h { Hold(f) => f(x) }\n"
	runNestedCases(t, []nestedCase{
		{name: "R10_definition", body: nestedApplyOpt},
		{name: "R11_definition", body: holder},
		{name: "R10_pure_use", body: nestedApplyOpt + "pure func sneaky(x: int) -> int = applyOpt(Some(logIt), x)\n", blame: "sneaky"},
		{name: "R11_pure_use", body: holder + "pure func sneaky(x: int) -> int = applyHold(Hold(logIt), x)\n", blame: "sneaky"},
		{name: "ok_declared", body: nestedApplyFst + nestedApplyHead + nestedApplyOpt + "func ok(x: int) -> int ! {IO} = applyFst((logIt, x)) + applyHead([logIt], x) + applyOpt(Some(logIt), x)\n"},
		{name: "R11_declared_use", body: holder + "func ok(x: int) -> int ! {IO} = applyHold(Hold(logIt), x)\n"},
	})
}

// AC3 — #1826: a concrete annotation's compatibility tail never escapes.
func TestLatentNested_ConcreteTailsClosed(t *testing.T) {
	runNestedCases(t, []nestedCase{
		{name: "S1_via_field", body: "import std/list (mapE)\ntype H = { act: (string) -> bool ! {IO} }\nfunc viaField(h: H, xs: [string]) -> [bool] ! {IO} = mapE(\\d. h.act(d), xs)\n"},
		{name: "S2_via_param", body: "import std/list (mapE)\nfunc viaParam(g: (string) -> bool ! {IO}, xs: [string]) -> [bool] ! {IO} = mapE(\\d. g(d), xs)\n"},
		{name: "S3_local_mapE", body: "func myMapE[a, b](f: (a) -> b ! {e}, xs: [a]) -> [b] ! {e} = match xs { [] => [], [x, ...rest] => f(x) :: myMapE(f, rest) }\nfunc viaOwn(g: (string) -> bool ! {IO}, xs: [string]) -> [bool] ! {IO} = myMapE(\\d. g(d), xs)\n"},
		{name: "S4_param_value_hof", body: "func applyTo(f: (string) -> bool, x: string) -> bool = f(x)\nfunc viaParamValue(g: (string) -> bool ! {IO}, x: string) -> bool ! {IO} = applyTo(g, x)\n"},
		{name: "S4_pure_variant_rejected", body: "func applyTo(f: (string) -> bool, x: string) -> bool = f(x)\npure func viaParamValue(g: (string) -> bool ! {IO}, x: string) -> bool = applyTo(g, x)\n", blame: "viaParamValue"},
		{name: "S2_handwritten_ap", body: "import std/list (mapE)\nfunc ap(k: (string) -> bool ! {IO}, x: string) -> bool ! {IO} = k(x)\nfunc viaAp(g: (string) -> bool ! {IO}, xs: [string]) -> [bool] ! {IO} = mapE(\\d. ap(g, d), xs)\n"},
		// sunholo/mcp_oauth 0.1.0/0.1.1 revokeFamily, reduced: a record-of-hooks
		// field whose concrete row is reached only through a type declaration.
		{name: "S5_mcp_oauth_revokeFamily", body: "import std/list (mapE)\ntype Hooks = { get: (string) -> Option[string] ! {IO}, revoke: (string) -> bool ! {IO, FS} }\n" +
			"func revokeFamily(h: Hooks, family: string) -> bool ! {IO, FS} = match h.get(family) { None => false, Some(raw) => { let _ = mapE(\\d. h.revoke(d), [raw]); true } }\n"},
	})
}

// AC4/AC5 — negative controls keep their base answer.
func TestLatentNested_Controls(t *testing.T) {
	const rowpoly = "func applyFstE[e](p: ((int) -> int ! {e}, int)) -> int ! {e} = match p { (f, x) => f(x) }\n"
	const fwd = nestedApplyBoth + "func fwd(f: (int) -> int, g: (int) -> int, x: int) -> int = applyBoth((f, g), x)\n"
	runNestedCases(t, []nestedCase{
		// accepted
		{name: "ok_pure_cb", body: nestedApplyFst + nestedApplyHead + "pure func ok(x: int) -> int = applyFst((inc, x)) + applyHead([inc, \\n. n * 2], x)\n"},
		{name: "ok_storage", body: "import std/list (length)\ntype H = { act: (string) -> bool ! {IO} }\nfunc logS(s: string) -> bool ! {IO} = { println(s); true }\n" +
			"pure func build() -> H = { act: logS }\npure func mkList() -> [(int) -> int] = [logIt]\npure func count() -> int = length([logIt, logIt])\npure func pair(x: int) -> ((int) -> int, int) = (logIt, x)\n"},
		{name: "ok_rowpoly", body: rowpoly + "pure func okPure(x: int) -> int = applyFstE((inc, x))\nfunc okIO(x: int) -> int ! {IO} = applyFstE((logIt, x))\n"},
		{name: "fwd_two", body: fwd + "pure func okPure(x: int) -> int = fwd(inc, inc, x)\nfunc okIO(x: int) -> int ! {IO} = fwd(logIt, inc, x)\n"},
		{name: "alias_two_callers", body: "type R = {f: (int) -> int, x: int}\nfunc applyRec(p: R) -> int = p.f(p.x)\npure func okPure(x: int) -> int = applyRec({f: inc, x: x})\nfunc okIO(x: int) -> int ! {IO} = applyRec({f: logIt, x: x})\n"},
		{name: "x_main_ok", body: "import x/lib (applyFst, applyHead)\nfunc fine(x: int) -> int ! {IO} = applyFst((logIt, x)) + applyHead([logIt], x)\n",
			extra: map[string]string{"x/lib.ail": "module x/lib\nexport " + nestedApplyFst + "export " + nestedApplyHead}},
		// rejected
		{name: "control_direct", body: "func applyTo(f: (int) -> int, x: int) -> int = f(x)\npure func sneaky(x: int) -> int = applyTo(logIt, x)\n", blame: "sneaky"},
		{name: "neg_rowpoly", body: rowpoly + "pure func sneaky(x: int) -> int = applyFstE((logIt, x))\n", blame: "sneaky"},
		{name: "neg_concrete_nested", body: "func applyFstIO(p: ((int) -> int ! {IO}, int)) -> int ! {IO} = match p { (f, x) => f(x) }\npure func sneaky(x: int) -> int = applyFstIO((logIt, x))\n", blame: "sneaky"},
		{name: "fwd_two_neg", body: fwd + "pure func sneaky(x: int) -> int = fwd(inc, logIt, x)\n", blame: "sneaky"},
		{name: "leak_adt_top_level", body: "type Box = Box((int) -> int)\nfunc applyBox(b: Box, x: int) -> int = match b { Box(f) => f(x) }\npure func sneaky(x: int) -> int = applyBox(Box(logIt), x)\n", blame: "sneaky"},
		{name: "ret_closure", body: "func mk() -> (int) -> int = logIt\nfunc applyTo(f: (int) -> int, x: int) -> int = f(x)\npure func sneaky(x: int) -> int = applyTo(mk(), x)\n", blame: "sneaky"},
		{name: "leak_lambda_in_list", body: nestedApplyHead + "pure func sneaky(x: int) -> int = applyHead([\\n. { println(\"side effect\"); n }], x)\n", blame: "sneaky"},
		{name: "mixed_rows_names_FS", body: "import std/fs (readFile)\nfunc readIt(x: int) -> int ! {FS} = { let _ = readFile(\"/dev/null\"); x }\n" + nestedApplyBoth +
			"func onlyIO(x: int) -> int ! {IO} = applyBoth((logIt, readIt), x)\n", blame: "onlyIO", label: "FS"},
		{name: "mixed_rows_names_IO", body: "import std/fs (readFile)\nfunc readIt(x: int) -> int ! {FS} = { let _ = readFile(\"/dev/null\"); x }\n" + nestedApplyBoth +
			"func onlyFS(x: int) -> int ! {FS} = applyBoth((logIt, readIt), x)\n", blame: "onlyFS", label: "IO"},
	})
}

// Design freeze (iii)/(ii): the deliberate flips.
func TestLatentNested_IntendedFlips(t *testing.T) {
	runNestedCases(t, []nestedCase{
		{name: "store_only_nested_handoff", body: "func keepAll(fs: [(int) -> int]) -> [(int) -> int] = fs\npure func storeNested(x: int) -> int = { let _ = keepAll([logIt]); x }\n", blame: "storeNested"},
		{name: "store_only_concrete_annotation_ok", body: "func keepAll(fs: [(int) -> int ! {IO}]) -> [(int) -> int ! {IO}] = fs\npure func storeNested(x: int) -> int = { let _ = keepAll([logIt]); x }\n"},
		{name: "reg_construct", body: "type Reg = Reg([(int) -> int])\npure func mkReg() -> Reg = Reg([logIt])\n", blame: "mkReg"},
	})
	// (i) a concrete `! {IO}` callback through closed-pure std/list.map was
	// accepted only via the row-unifier label hole (U1); it is now the same type
	// error as `map(\d. println(d), xs)`. Migration: mapE.
	t.Run("map_concrete_callback_is_type_error", func(t *testing.T) {
		err := checkModules(t, map[string]string{"main.ail": nestedSource("main", "import std/list (map)\nfunc viaMap(g: (string) -> bool ! {IO}, xs: [string]) -> [bool] ! {IO} = map(\\d. g(d), xs)\n")})
		if err == nil || !strings.Contains(err.Error(), "incompatible closed rows") {
			t.Fatalf("want closed-row type error: %v", err)
		}
	})
}

// Two distinct declared tails nested in one argument cannot be one callback
// row: the conflict is explicit (#616), never a silent first-wins acceptance.
// (On base this program checked clean while g's effects ran unpaid.)
func TestLatentNested_DistinctDeclaredTailsFailLoud(t *testing.T) {
	t.Setenv("AILANG_NO_CACHE", "1")
	err := checkModules(t, map[string]string{"main.ail": nestedSource("main", nestedApplyBoth+
		"func fwdE[e1, e2](f: (int) -> int ! {e1}, g: (int) -> int ! {e2}, x: int) -> int ! {e1} = applyBoth((f, g), x)\n")})
	if err == nil || !strings.Contains(err.Error(), "missing callback row") {
		t.Fatalf("want explicit callback-row conflict: %v", err)
	}
}

// performedLine matches the side effect actually printed, not a source excerpt.
var performedLine = regexp.MustCompile(`(?m)^side effect\r?$`)

// Rejected programs must fail before execution on both engines.
func TestLatentNested_RunRejectsBeforeExecution(t *testing.T) {
	binary := testutil.FindAilangBinary(t)
	t.Setenv("AILANG_NO_CACHE", "1")
	for _, tt := range []struct{ name, src string }{
		{"tuple", nestedApplyFst + "pure func sneaky(x: int) -> int = applyFst((logIt, x))\n"},
		{"list", nestedApplyHead + "pure func sneaky(x: int) -> int = applyHead([logIt], x)\n"},
		{"record", "func applyRec(p: {f: (int) -> int, x: int}) -> int = p.f(p.x)\npure func sneaky(x: int) -> int = applyRec({f: logIt, x: x})\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			src := nestedSource(tt.name, tt.src+"export func main() -> () ! {IO} = println(show(sneaky(1)))\n")
			path := filepath.Join(t.TempDir(), tt.name+".ail")
			if err := os.WriteFile(path, []byte(src), 0600); err != nil {
				t.Fatal(err)
			}
			for _, bytecode := range []bool{false, true} {
				args := []string{"run", "--caps", "IO", "--entry", "main"}
				if bytecode {
					args = append(args, "--bytecode")
				}
				output, err := exec.Command(binary, append(args, path)...).CombinedOutput()
				text := string(output)
				if err == nil || performedLine.MatchString(text) {
					t.Fatalf("executed unsound program (bytecode=%v): %s", bytecode, output)
				}
				if !strings.Contains(text, "Missing effects: IO") || !strings.Contains(text, "function 'sneaky'") {
					t.Fatalf("wrong rejection (bytecode=%v): %s", bytecode, output)
				}
			}
		})
	}
}
