package types_test

import (
	"fmt"
	"github.com/sunholo-data/ailang/internal/types"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sinkFieldPrelude = `module test/sink_fields
type Box = { payload: string{not secret} }
type Nested = { box: Box }
type Handler = { run: (string{not secret}) -> string }
type Alias = Box
pure func source() -> string<secret> = "secret"
pure func consume(b: Box) -> string = b.payload
`

func TestIFCSinkFieldsReject(t *testing.T) {
	cases := []struct{ name, body string }{
		{"plain_let", `pure func f() -> string { let b: Box = {payload: source()}; b.payload }`},
		{"return", `pure func f() -> Box = {payload: source()}`},
		{"argument", `pure func f() -> string = consume({payload: source()})`},
		{"alias", `pure func f() -> Alias = {payload: source()}`},
		{"nested", `pure func f() -> Nested = {box: {payload: source()}}`},
		{"update", `pure func f(b: Box) -> Box = {b | payload: source()}`},
		{"unannotated_update", `pure func f(b: Box) -> string { let changed = {b | payload: source()}; changed.payload }`},
		{"fn_domain", `pure func f(h: Handler) -> string = h.run(source())`},
		{"fn_alias", `pure func f(h: Handler) -> string { let run = h.run; run(source()) }`},
		{"fn_higher_order", `pure func apply(run: (string{not secret}) -> string) -> string = run(source())`},
		{"nested_domain", `type H = {run: (Box) -> string}
pure func f(h: H) -> string = h.run({payload: source()})`},
		{"whole_record_transfer", `type Raw = {payload: string<secret>}
pure func f(b: Raw) -> Box = b`},
		{"destructure", `pure func f(h: Handler) -> string = match h { {run} => run(source()) }`},
		{"conditional", `pure func f(h: Handler) -> string { let run = if true then h.run else h.run; run(source()) }`},
		{"inferred_record", `pure func f() -> string { let h = {run: func(s: string{not secret}) -> string { s }}; h.run(source()) }`},
		{"erased_domain", `pure func apply(run: (string) -> string) -> string = run(source())
pure func f(h: Handler) -> string = apply(h.run)`},
		{"erased_record", `type OpenHandler = {run: (string) -> string}
pure func f(h: Handler) -> OpenHandler = h`},
		{"list_erasure", `pure func f(h: Handler) -> string { let runs = [h.run]; match runs { run :: _ => run(source()), [] => "" } }`},
		{"tuple_erasure", `pure func f(h: Handler) -> string { let pair = (h.run, 1); match pair { (run, _) => run(source()) } }`},
		{"conditional_erasure", `pure func identity(s: string) -> string = s
pure func f(h: Handler) -> string { let run = if true then h.run else identity; run(source()) }`},
		{"unrelated_label", `type Other = {payload: string{not authcode}}
pure func raw() -> string<authcode> = "x"
pure func f() -> Other = {payload: raw()}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := types.CheckModuleIFC(parseIFC(t, sinkFieldPrelude+tc.body))
			if len(errs) == 0 {
				t.Fatal("secret sink silently accepted")
			}
			if errs[0].Kind != types.SinkRefinementError {
				t.Fatalf("wrong diagnostic: %v", errs)
			}
		})
	}
}

func TestIFCSinkFieldsSafe(t *testing.T) {
	for _, body := range []string{
		`pure func f() -> Box = {payload: "safe"}`,
		`pure func f(h: Handler) -> string = h.run("safe")`,
		`pure func f(b: Box) -> Box = {b | payload: "safe"}`,
		`pure func f() -> Nested = {box: {payload: "safe"}}`,
		`type Mixed = {safe: string{not secret}, raw: string<secret>}
pure func f() -> Mixed = {safe: "public", raw: source()}`,
		`pure func apply(run: (string{not secret}) -> string) -> string = run("safe")
pure func f(h: Handler) -> string = apply(h.run)`,
		`pure func f() -> string { let b: Box = {payload: "safe"}; consume(b) }`,
		`type Public = {payload: string{not authcode}}
pure func f() -> Public = {payload: source()}`,
	} {
		errs := types.CheckModuleIFC(parseIFC(t, sinkFieldPrelude+body))
		if len(errs) != 0 {
			t.Fatalf("safe value rejected: %v", errs)
		}
	}
}

func TestIFCSinkUnsupportedPositions(t *testing.T) {
	for _, body := range []string{
		`type Maker = {mk: () -> string{not secret}}`,
		`type Items = [string{not secret}]`,
		`type Pair = (string{not secret}, int)`,
		`type Tok = Tok(string{not secret})`,
		`type Items = [Box]`,
		`type Recursive = {next: Recursive, payload: string{not secret}}`,
		`type Recursive = {next: Recursive, run: (string{not secret}) -> string}`,
		`type Higher = {run: ((string{not secret}) -> string) -> string}`,
		`type Generic[a] = {payload: a{not secret}}`,
		`pure func f(xs: [string{not secret}]) -> int = 0`,
		`let h: Handler = {run: func(s: string) -> string {s}}`,
		`let h = {run: func(s: string{not secret}) -> string {s}}`,
		`pure func f() -> int { let xs: [string{not secret}] = []; 0 }`,
	} {
		errs := types.CheckModuleIFC(parseIFC(t, sinkFieldPrelude+body))
		if len(errs) == 0 || !strings.Contains(errs[0].Message, "not enforceable") {
			t.Fatalf("unsupported annotation silently accepted: %s: %v", body, errs)
		}
	}
}

func TestIFCSinkLongAliasesAndCycles(t *testing.T) {
	var chain strings.Builder
	for i := 0; i < 32; i++ {
		fmt.Fprintf(&chain, "type Alias%d = Alias%d\n", i, i+1)
	}
	chain.WriteString("type Alias32 = Box\npure func f() -> Alias0 = {payload: source()}\n")
	if errs := types.CheckModuleIFC(parseIFC(t, sinkFieldPrelude+chain.String())); len(errs) == 0 {
		t.Fatal("alias depth hides sink")
	}
	// Parse-only robustness: the HM checker rejects aliases with no finite body,
	// but the IFC surface walk must still terminate if handed one.
	types.CheckModuleIFC(parseIFC(t, `module cycle
type A = B<secret>
type B = A
pure func f(x: A) -> A = x`))
}

func TestIFCFieldSinkExample(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "expected_fail", "mcp_friction_field_sinks.ail"))
	if err != nil {
		t.Fatal(err)
	}
	errs := types.CheckModuleIFC(parseIFC(t, string(data)))
	if len(errs) != 2 {
		t.Fatalf("want exactly two leaks and no safe-path errors: %v", errs)
	}
	for _, err := range errs {
		if err.Kind != types.SinkRefinementError {
			t.Fatalf("wrong fixture diagnostic: %v", err)
		}
	}
}
