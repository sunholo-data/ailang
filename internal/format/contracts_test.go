package format

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// contracts_test.go holds the M1 round-trip fixtures for the contract-clause
// printer fix (m-fmt-properties-printer-roundtrip): requires/ensures contract
// clauses must be emitted in signature position and re-parse to an identical
// AST, and a properties block must not clobber contract entries.
//
// Each case asserts parse -> print -> re-parse cmp.Diff AST identity AND
// idempotence via assertIdempotentAndRoundTrips.

// TestContractRoundTrips covers the seven M1 fixtures (a)-(g).
func TestContractRoundTrips(t *testing.T) {
	cases := []struct {
		name string
		src  string
		// mustContain is a set of substrings the canonical output MUST contain,
		// verifying the emitted surface form (e.g. contract survived, not dropped).
		mustContain []string
	}{
		{
			name: "a_requires_only",
			src: "module m\n" +
				"export func absolute(x: int) -> int ! {}\n" +
				"requires { x >= 0 }\n" +
				"{\n  x\n}\n",
			mustContain: []string{"requires {"},
		},
		{
			name: "b_ensures_only",
			src: "module m\n" +
				"export func absolute(x: int) -> int ! {}\n" +
				"ensures { result >= 0 }\n" +
				"{\n  x\n}\n",
			mustContain: []string{"ensures {"},
		},
		{
			name: "c_both",
			src: "module m\n" +
				"export func absolute(x: int) -> int ! {}\n" +
				"requires { x >= 0 }\n" +
				"ensures { result >= 0 }\n" +
				"{\n  x\n}\n",
			mustContain: []string{"requires {", "ensures {"},
		},
		{
			name: "d_multi_predicate_requires",
			src: "module m\n" +
				"export func safeDivide(a: int, b: int) -> int ! {}\n" +
				"requires { a >= 0, b > 0 }\n" +
				"{\n  a / b\n}\n",
			mustContain: []string{"requires {"},
		},
		{
			name: "e_ensures_forall_expr",
			src: "module m\n" +
				"export func fill(n: int) -> int ! {}\n" +
				"ensures { forall i: 0..n => i >= 0 }\n" +
				"{\n  n\n}\n",
			mustContain: []string{"ensures {", "forall"},
		},
		{
			name: "f_genuine_properties_block",
			src: "module m\n" +
				"export func f(x: int) -> int ! {}\n" +
				"  properties [\n" +
				"    forall(y: int) => f(y) >= 0\n" +
				"  ]\n" +
				"{\n  x\n}\n",
			mustContain: []string{"properties ["},
		},
		{
			name: "g_contracts_and_properties_combined",
			src: "module m\n" +
				"export func f(x: int) -> int ! {}\n" +
				"requires { x >= 0 }\n" +
				"ensures { result >= 0 }\n" +
				"  properties [\n" +
				"    forall(y: int) => f(y) >= 0\n" +
				"  ]\n" +
				"{\n  x\n}\n",
			// The combined case is the silent-deletion regression guard: the
			// requires clause MUST survive fmt output.
			mustContain: []string{"requires {", "ensures {", "properties ["},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := assertIdempotentAndRoundTrips(t, tc.src, "test://"+tc.name)
			for _, want := range tc.mustContain {
				if !strings.Contains(out, want) {
					t.Errorf("%s: output missing %q\n--- output ---\n%s", tc.name, want, out)
				}
			}
		})
	}
}

// TestContractKindPartitionInParse locks the parse-level source-of-truth
// ordering for the combined case: fn.Properties is EXACTLY
// [RequiresKind, EnsuresKind, PropertyKind] in that order, proving the parser
// append fix preserves contract entries when a properties block follows.
func TestContractKindPartitionInParse(t *testing.T) {
	src := "module m\n" +
		"export func f(x: int) -> int ! {}\n" +
		"requires { x >= 0 }\n" +
		"ensures { result >= 0 }\n" +
		"  properties [\n" +
		"    forall(y: int) => f(y) >= 0\n" +
		"  ]\n" +
		"{\n  x\n}\n"
	prog := parseProg(t, src, "test://partition")

	var fn *ast.FuncDecl
	for _, d := range prog.File.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name == "f" {
			fn = fd
			break
		}
	}
	if fn == nil {
		t.Fatalf("func f not found in parsed decls")
	}
	wantKinds := []ast.ContractKind{ast.RequiresKind, ast.EnsuresKind, ast.PropertyKind}
	if len(fn.Properties) != len(wantKinds) {
		t.Fatalf("fn.Properties length = %d, want %d (kinds: %v)", len(fn.Properties), len(wantKinds), propKinds(fn.Properties))
	}
	for i, want := range wantKinds {
		if fn.Properties[i].Kind != want {
			t.Errorf("fn.Properties[%d].Kind = %v, want %v", i, fn.Properties[i].Kind, want)
		}
	}
}

// TestParenStatementSeparatorRoundTrips locks the pre-existing Phase-1 printer bug
// surfaced by scoring.ail: a block statement whose RENDERED form begins with `(`
// (because a low-precedence left operand is parenthesised) must be `;`-separated
// from the preceding statement, or re-parse glues it as a call. startsWithStatementStarter
// walked BinaryOp.Left ignoring parenthesisation and wrongly reported "starter".
func TestParenStatementSeparatorRoundTrips(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{
			// (a + b) * c: left of `*` is a parenthesised `+`, so the statement
			// renders starting with `(`.
			"paren_left_binop",
			"module m\nfunc f(a: int, b: int, c: int) -> int { let x = a; (a + b) * c }",
		},
		{
			// match then a paren-leading bare expression (the scoring.ail shape).
			"match_then_paren_expr",
			"module m\nfunc f(a: int, b: int, w: int) -> int {\n" +
				"  let s = match a { 0 => 1, _ => 2 };\n" +
				"  (s + b) * w / 100 + s\n" +
				"}",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertIdempotentAndRoundTrips(t, tc.src, "test://"+tc.name)
		})
	}
}

// TestVerifyAnnotationRoundTrips locks the @verify(depth: N) annotation round-trip:
// the parser stores only the int literal (dropping the `depth:` key), so the printer
// must re-synthesise the key or re-parse fails with PAR_VERIFY_ATTR_KEY.
func TestVerifyAnnotationRoundTrips(t *testing.T) {
	src := "module m\n@verify(depth: 5)\nexport func f(x: int) -> int ! {}\nrequires { x >= 0 }\n{\n  x\n}\n"
	out := assertIdempotentAndRoundTrips(t, src, "test://verify_annotation")
	if !strings.Contains(out, "@verify(depth: 5)") {
		t.Errorf("output missing @verify(depth: 5):\n%s", out)
	}
}

func propKinds(props []*ast.Property) []ast.ContractKind {
	ks := make([]ast.ContractKind, len(props))
	for i, p := range props {
		ks[i] = p.Kind
	}
	return ks
}

// TestMCPAnnotationsRoundTrip: @mcp_hints() is a complete, empty hint list. The
// parser stores no args, and a bare `@mcp_hints` does not re-parse, so the
// printer must keep the parens. The other MCP annotations round-trip as usual.
func TestMCPAnnotationsRoundTrip(t *testing.T) {
	src := "module m\n@mcp_title(\"Log it\")\n@mcp_hints()\nexport func f(x: int) -> int ! {IO} {\n  x\n}\n\n@mcp_hints(\"readOnly\", \"openWorld\")\nexport func g(x: int) -> int ! {IO} {\n  x\n}\n"
	out := assertIdempotentAndRoundTrips(t, src, "test://mcp_annotations")
	for _, want := range []string{"@mcp_title(\"Log it\")", "@mcp_hints()\n", "@mcp_hints(\"readOnly\", \"openWorld\")"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

// M-SERVEAPI-DIRECTORY-READY M1: the listed-surface annotations round-trip
// (string args as written; the two parameterless ones stay bare).
func TestMCPDirectoryAnnotationsRoundTrip(t *testing.T) {
	src := "module m\n@mcp_auth(\"oauth2\")\n@mcp_secret(\"apiKey\")\n@mcp_agent_only\n@optional(\"apiKey\")\nexport func f(apiKey: string) -> string ! {IO} {\n  apiKey\n}\n\n@mcp_token_verifier\nexport func v(token: string) -> bool ! {IO} {\n  true\n}\n"
	out := assertIdempotentAndRoundTrips(t, src, "test://mcp_directory_annotations")
	for _, want := range []string{"@mcp_auth(\"oauth2\")", "@mcp_secret(\"apiKey\")", "@mcp_agent_only\n", "@mcp_token_verifier\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

// M-MCP-FILE-HANDOFF F1b: @mcp_file round-trips, repeated and multi-arg.
func TestMCPFileAnnotationRoundTrip(t *testing.T) {
	src := "module m\n@mcp_file(\"a\", \"b\")\n@mcp_file(\"c\")\nexport func f(a: string, b: string, c: string) -> string {\n  a\n}\n"
	out := assertIdempotentAndRoundTrips(t, src, "test://mcp_file_annotation")
	for _, want := range []string{"@mcp_file(\"a\", \"b\")\n", "@mcp_file(\"c\")\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}
