package types_test

import (
	"os"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
	"github.com/sunholo-data/ailang/internal/types"
)

// parseIFC parses an AILANG source string to an *ast.File, failing the test on
// any parse error so IFC assertions run against a well-formed module.
func parseIFC(t *testing.T, src string) *ast.File {
	t.Helper()
	p := parser.New(lexer.New(src, "ifc_test.ail"))
	file := p.ParseFile()
	if errs := p.Errors(); len(errs) > 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if file == nil {
		t.Fatal("ParseFile returned nil")
	}
	return file
}

// TestIFC_DirectLeak_Caught: a <secret> source flowing through a let-binding
// into a {not secret} sink is a compile-time violation. This is the headline
// M5 acceptance criterion.
func TestIFC_DirectLeak_Caught(t *testing.T) {
	src := `module test/leak
func getSecret() -> string<secret> ! {} { "sk-xxx" }
func logIt(msg: string{not secret}) -> string ! {} { msg }
func leak() -> string ! {} {
  let s = getSecret() in logIt(s)
}`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 1 {
		t.Fatalf("expected 1 IFC violation, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Error(), "secret") {
		t.Errorf("error should mention the secret label: %v", errs[0])
	}
	if errs[0].Kind != types.SinkRefinementError {
		t.Errorf("expected SinkRefinementError, got %q", errs[0].Kind)
	}
}

// TestIFC_DirectArgLeak_Caught: same leak without an intervening let — the
// source feeds the sink directly as a call argument.
func TestIFC_DirectArgLeak_Caught(t *testing.T) {
	src := `module test/direct
func getSecret() -> string<secret> ! {} { "x" }
func logIt(msg: string{not secret}) -> string ! {} { msg }
func leak() -> string ! {} { logIt(getSecret()) }`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(errs), errs)
	}
}

// TestIFC_BlockStatementLet_Caught: a block-statement let (`{ let s = ...; sink(s) }`,
// no `in`) scopes the binding over the rest of the block — taint must survive it.
func TestIFC_BlockStatementLet_Caught(t *testing.T) {
	src := `module test/block
func getSecret() -> string<secret> ! {} { "x" }
func logIt(msg: string{not secret}) -> string ! {} { msg }
func leak() -> string ! {} {
  let s = getSecret();
  logIt(s)
}`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 1 {
		t.Fatalf("expected 1 violation through a block-statement let, got %d: %v", len(errs), errs)
	}
}

// TestIFC_DeclassifyGate_Passes: routing the secret through a ! {Declassify}
// function lowers its label, so the downstream {not secret} sink is satisfied.
func TestIFC_DeclassifyGate_Passes(t *testing.T) {
	src := `module test/gate
func getSecret() -> string<secret> ! {} { "sk-xxx" }
func reveal(s: string<secret>) -> string ! {Declassify} { s }
func logIt(msg: string{not secret}) -> string ! {} { msg }
func ok() -> string ! {} {
  let s = getSecret() in
  let clean = reveal(s) in
  logIt(clean)
}`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 0 {
		t.Fatalf("expected 0 violations (declassify gate), got %d: %v", len(errs), errs)
	}
}

// TestIFC_TransparentPassthrough_NoError: plain, unlabelled data forwarded
// through an unannotated function into a sink raises no violation — ordinary
// code needs no annotations.
func TestIFC_TransparentPassthrough_NoError(t *testing.T) {
	src := `module test/plain
func wrap(x: string) -> string ! {} { x }
func sink(msg: string{not secret}) -> string ! {} { msg }
func go() -> string ! {} {
  let a = wrap("hi") in sink(a)
}`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 0 {
		t.Fatalf("expected 0 violations for plain data, got %d: %v", len(errs), errs)
	}
}

// TestIFC_LaunderViaExplicitReturnLabel_Caught: a function that takes a
// <secret> and declares a *different* explicit return label (<clean>) while
// returning the secret unchanged, without ! {Declassify}, is laundering (Check B).
func TestIFC_LaunderViaExplicitReturnLabel_Caught(t *testing.T) {
	src := `module test/launder
func launder(s: string<secret>) -> string<clean> ! {} { s }`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 1 {
		t.Fatalf("expected 1 declassify violation, got %d: %v", len(errs), errs)
	}
	if errs[0].Kind != types.DeclassifyRequiredError {
		t.Errorf("expected DeclassifyRequiredError, got %q", errs[0].Kind)
	}
}

// TestIFC_DeclassifyAllowsRelabel_NoError: the same relabel is permitted when
// the function declares ! {Declassify}.
func TestIFC_DeclassifyAllowsRelabel_NoError(t *testing.T) {
	src := `module test/declass
func sanitize(s: string<email>) -> string<sanitized> ! {Declassify} { s }`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 0 {
		t.Fatalf("expected 0 violations (Declassify authorises relabel), got %d: %v", len(errs), errs)
	}
}

// TestIFC_StructuralRecordReturn_NoError mirrors inbox_injection_v2.ail: a
// labelled value flows into a structural record returned with no explicit
// return label and no {not ℓ} typed sink. The pass must leave this untouched
// (behaviour unchanged — Z3 contracts remain the enforcement there).
func TestIFC_StructuralRecordReturn_NoError(t *testing.T) {
	src := `module test/inbox
type SendAction = { to: string, body: string }
func injectedForward(rawEmail: string<email>, recipient: string) -> SendAction ! {} {
  { to: recipient, body: rawEmail }
}`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 0 {
		t.Fatalf("expected 0 violations (no typed sink, no explicit return label), got %d: %v", len(errs), errs)
	}
}

// TestIFC_SecretBuiltinSource_Caught: a call to the secret() builtin is a
// <secret> source even though the stdlib signature is plain `string`.
func TestIFC_SecretBuiltinSource_Caught(t *testing.T) {
	src := `module test/builtin
import std/secret (secret)
func logIt(msg: string{not secret}) -> string ! {} { msg }
func leak() -> string ! {Secret} { logIt(secret("op://Prod/k/v", "leak it")) }`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 1 {
		t.Fatalf("expected 1 violation from the secret() builtin source, got %d: %v", len(errs), errs)
	}
}

// TestIFC_LocalWrapperSource_Caught: an unannotated local function that calls
// secret() internally still propagates <secret> to its callers (effective body
// label), so leaking its result is caught.
func TestIFC_LocalWrapperSource_Caught(t *testing.T) {
	src := `module test/wrap
import std/secret (secret)
func fetch() -> string ! {Secret} { secret("op://k", "fetch it") }
func logIt(msg: string{not secret}) -> string ! {} { msg }
func leak() -> string ! {Secret} { logIt(fetch()) }`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 1 {
		t.Fatalf("expected 1 violation (local wrapper preserves secret), got %d: %v", len(errs), errs)
	}
}

// TestIFC_CleanModule_NoError: a module with no IFC annotations at all is clean.
func TestIFC_CleanModule_NoError(t *testing.T) {
	src := `module test/clean
func add(a: int, b: int) -> int ! {} { a + b }
func main() -> int ! {} { add(1, 2) }`
	errs := types.CheckModuleIFC(parseIFC(t, src))
	if len(errs) != 0 {
		t.Fatalf("expected 0 violations for an unlabelled module, got %d: %v", len(errs), errs)
	}
}

// TestSecretExamples_IFC validates the shipped M-SECRET-EFFECT example files:
// gated_secret.ail must be IFC-clean and leak_attempt.ail must be rejected.
// These examples are skipped by verify-examples (they need op/approval to run),
// so this is their CI coverage.
func TestSecretExamples_IFC(t *testing.T) {
	cases := []struct {
		path     string
		wantErrs int
	}{
		{"../../examples/runnable/secrets/gated_secret.ail", 0},
		{"../../examples/runnable/secrets/leak_attempt.ail", 1},
		{"../../examples/runnable/secrets/secret_demo.ail", 0},
	}
	for _, tc := range cases {
		src, err := os.ReadFile(tc.path)
		if err != nil {
			t.Fatalf("read %s: %v", tc.path, err)
		}
		errs := types.CheckModuleIFC(parseIFC(t, string(src)))
		if len(errs) != tc.wantErrs {
			t.Errorf("%s: expected %d IFC violation(s), got %d: %v", tc.path, tc.wantErrs, len(errs), errs)
		}
		if tc.wantErrs == 1 && len(errs) == 1 && errs[0].Kind != types.SinkRefinementError {
			t.Errorf("%s: expected SinkRefinementError, got %q", tc.path, errs[0].Kind)
		}
	}
}

// TestIFC_NilFile_NoPanic guards the entry point against a nil module.
func TestIFC_NilFile_NoPanic(t *testing.T) {
	if errs := types.CheckModuleIFC(nil); errs != nil {
		t.Fatalf("expected nil for nil file, got %v", errs)
	}
}
