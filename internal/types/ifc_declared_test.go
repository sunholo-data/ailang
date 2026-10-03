package types_test

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// M-IFC-DECLARED-RECORD-LABELS (#1523, #1527): a label written INSIDE a type
// annotation — a declared record's field, an alias, an ADT constructor field, a
// list/tuple element, a let or lambda annotation — is a source label for every
// value of that type. Before the fix only a top-level <label> on a parameter or
// return type was read, so all of the shapes below were accepted.

// ifcDeclPrelude is shared by every fixture: declared types carrying a label in
// a nested position, constructors for them, and a {not authcode} sink.
const ifcDeclPrelude = `module test/decl
type RawCode = { raw: string<authcode> }
type Session = { user: string, code: RawCode }
type Tok = Tok(string<authcode>) | NoTok
type Pair = (string<authcode>, int)
type Code = string<authcode>
pure func logLine(s: string{not authcode}) -> string = "log: ${s}"
pure func mk(s: string) -> RawCode = { raw: s }
pure func mkS(u: string) -> Session = { user: u, code: mk(u) }
pure func getRaw(c: RawCode) -> string = c.raw
pure func getUser(s: Session) -> string = s.user
pure func plain(s: string) -> string = s
`

func checkDecl(t *testing.T, body string) []*types.TypeCheckError {
	t.Helper()
	return types.CheckModuleIFC(parseIFC(t, ifcDeclPrelude+body))
}

// TestIFCDeclared_Rejected: every shape a declared/nested label can reach a sink
// through must be an information-flow violation (17 measured shapes, design doc
// table p1–p13, r1–r4).
func TestIFCDeclared_Rejected(t *testing.T) {
	cases := []struct{ name, body string }{
		{"p1_param_field", `export pure func f(c: RawCode) -> string = logLine(c.raw)`},
		{"p2_call_result_field", `export pure func f(x: string) -> string { let c = mk(x); logLine(c.raw) }`},
		{"p3_nested_param", `export pure func f(s: Session) -> string = logLine(s.code.raw)`},
		{"p4_nested_call", `export pure func f(u: string) -> string = logLine(mkS(u).code.raw)`},
		{"p5_destructure", `export pure func f(c: RawCode) -> string = match c { {raw} => logLine(raw) }`},
		{"p6_nested_destructure", `export pure func f(s: Session) -> string = match s { {code: {raw: r}, ...} => logLine(r) }`},
		{"p7_inline_record_annotation", `export pure func f(c: {raw: string<authcode>}) -> string = logLine(c.raw)`},
		{"p8_adt_param", `export pure func f(t: Tok) -> string = match t { Tok(s) => logLine(s), NoTok => "" }`},
		{"p9_adt_constructor_app", `export pure func f(x: string) -> string = match Tok(x) { Tok(s) => logLine(s), NoTok => "" }`},
		{"p10_tuple_alias", `export pure func f(p: Pair) -> string = match p { (s, _) => logLine(s) }`},
		{"p11_list_element", `export pure func f(xs: [string<authcode>]) -> string = match xs { h :: _ => logLine(h), [] => "" }`},
		{"p12_let_annotation", `export pure func f(x: string) -> string { let c: RawCode = { raw: x }; logLine(c.raw) }`},
		{"p13_lambda_param", `export pure func f(x: string) -> string { let g = func(c: RawCode) -> string { logLine(c.raw) }; g(mk(x)) }`},
		{"r1_alias_of_labelled", `export pure func f(c: Code) -> string = logLine(c)`},
		{"r2_let_top_level_label", `export pure func f(x: string) -> string { let t: string<authcode> = plain(x); logLine(t) }`},
		{"r3_getter", `export pure func f(x: string) -> string = logLine(getRaw(mk(x)))`},
		{"r4_list_of_declared", `export pure func f(cs: [RawCode]) -> string = match cs { c :: _ => logLine(c.raw), [] => "" }`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			errs := checkDecl(t, tc.body)
			if len(errs) == 0 {
				t.Fatalf("expected an information-flow violation, got none")
			}
			if errs[0].Kind != types.SinkRefinementError {
				t.Errorf("expected SinkRefinementError, got %q: %v", errs[0].Kind, errs[0])
			}
			if !strings.Contains(errs[0].Error(), "authcode") {
				t.Errorf("error should name the authcode label: %v", errs[0])
			}
		})
	}
}

// TestIFCDeclared_InlineRecordStillRejected: the inline-record case that already
// worked keeps working.
func TestIFCDeclared_InlineRecordStillRejected(t *testing.T) {
	errs := checkDecl(t, `export pure func f(x: string<authcode>) -> string { let r = { raw: x }; logLine(r.raw) }`)
	if len(errs) != 1 || errs[0].Kind != types.SinkRefinementError {
		t.Fatalf("expected 1 SinkRefinementError, got %v", errs)
	}
}

// TestIFCDeclared_FieldPrecision: an UNLABELLED field next to a labelled one
// stays clean — projection is field-precise, not the record's whole label.
func TestIFCDeclared_FieldPrecision(t *testing.T) {
	cases := []struct{ name, body string }{
		{"param", `export pure func f(s: Session) -> string = logLine(s.user)`},
		{"call_result", `export pure func f(u: string) -> string = logLine(mkS(u).user)`},
		{"let", `export pure func f(u: string) -> string { let s = mkS(u); logLine(s.user) }`},
		{"annotated_let", `export pure func f(u: string) -> string { let s: Session = mkS(u); logLine(s.user) }`},
		{"getter", `export pure func f(s: Session) -> string = logLine(getUser(s))`},
		{"getter_of_call", `export pure func f(u: string) -> string = logLine(getUser(mkS(u)))`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if errs := checkDecl(t, tc.body); len(errs) != 0 {
				t.Fatalf("expected no violation for an unlabelled sibling field, got %v", errs)
			}
		})
	}
}

// TestIFCDeclared_AnnotationCannotLower: HM strips labels in unification, so an
// annotation can put the label on a DIFFERENT field than the value's type does
// (V12). The binding must keep the value's labels, not adopt the annotation's.
func TestIFCDeclared_AnnotationCannotLower(t *testing.T) {
	src := `module test/swap
type A = { user: string<authcode>, code: string }
pure func logLine(s: string{not authcode}) -> string = "log: ${s}"
pure func mkA(u: string) -> A = { user: u, code: u }
export pure func f(u: string) -> string {
  let s: { user: string, code: string<authcode> } = mkA(u);
  logLine(s.user)
}`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 1 || errs[0].Kind != types.SinkRefinementError {
		t.Fatalf("an annotation must not lower the value's label: got %v", errs)
	}
}

// TestIFCDeclared_McpOauthIdioms: the three IFC idioms sunholo/mcp_oauth's
// flow.ail relies on keep their meaning — a pure classifier raises a label, a
// plain-string return propagates it, and a Declassify digest lowers it.
func TestIFCDeclared_McpOauthIdioms(t *testing.T) {
	src := `module test/mcp
pure func asSecret(raw: string) -> string<secret> = raw
pure func body(t: string<secret>) -> string = "{\"access_token\":\"${t}\"}"
pure func digestOf(s: string<secret>) -> string ! {Declassify} = "digest"
pure func audit(msg: string{not secret}) -> string = msg
export pure func okDigest(raw: string) -> string = audit(digestOf(asSecret(raw)))
export pure func okInline(raw: string) -> string { let r = { d: digestOf(asSecret(raw)) }; audit(r.d) }
`
	if errs := types.CheckModuleIFC(parseIFC(t, src)); len(errs) != 0 {
		t.Fatalf("clean mcp_oauth idioms must check: %v", errs)
	}
	leaks := []string{
		`export pure func leak(raw: string) -> string = audit(asSecret(raw))`,
		`export pure func leak(raw: string) -> string = audit(body(asSecret(raw)))`,
		`export pure func leak(raw: string) -> string { let r = { t: asSecret(raw) }; audit(r.t) }`,
	}
	for _, l := range leaks {
		if errs := types.CheckModuleIFC(parseIFC(t, src+l)); len(errs) != 1 {
			t.Errorf("expected exactly 1 violation for %q, got %v", l, errs)
		}
	}
}
