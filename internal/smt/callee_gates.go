package smt

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

// calleeGateRejections runs the gates on the callees a contracted function's
// body and contracts reach, before any of them is encoded. Each refuses a call
// the encoder would get wrong, as a skip instead of a wrong verdict or a Z3
// error:
//
//   - callee identity (S7): lookups are by bare name, so a call meaning another
//     module's function than that lookup binds would prove a contract about the
//     wrong code;
//   - unannotated parameters (S5): a callee parameter without a type has no sort
//     and would reach Z3 as an unknown constant;
//   - callee sorts (M-SMT-CALLEE-SORT-GATE): an unencodable signature type (a
//     type variable, tuple, function type or bytes) would leak an undeclared
//     sort into the script. Contract predicates are gated too: their callees are
//     resolved as define-funs (resolveCalleesFromRoots).
func calleeGateRejections(
	funcName, owner string,
	body core.CoreExpr,
	contracts []*core.Contract,
	prog *core.Program,
	imported map[string]*core.Program,
	calleeASTFuncs map[string]*ast.FuncDecl,
	declarable map[string]int,
) []SMTRejectionReason {
	var out []SMTRejectionReason
	if m := firstMisresolvedCallee(append([]core.CoreExpr{body}, contractExprs(contracts)...), funcName, owner, prog, imported); m != nil {
		resolved := m.Resolved
		if resolved == "" {
			resolved = owner
		}
		out = append(out, SMTRejectionReason{
			Code:    RejectUnencodable,
			Message: fmt.Sprintf("Function %q calls %s.%s, but another loaded module (%s) also defines %q and the verifier resolves callees by bare name", funcName, m.Wanted, m.Name, resolved, m.Name),
			Hint:    "Rename one of the two functions so the name is unique among the modules this function's calls reach.",
		})
	}
	if callee, param := firstUnannotatedCalleeParam(funcName, body, contracts, prog, imported, calleeASTFuncs); callee != "" {
		out = append(out, SMTRejectionReason{
			Code:    RejectUnencodable,
			Message: fmt.Sprintf("Function %q calls %q, whose parameter %q has no type annotation and so no SMT sort", funcName, callee, param),
			Hint:    fmt.Sprintf("Annotate the parameter in %q, e.g. `%s: int`.", callee, param),
		})
	}
	if callee, badType := firstUnencodableCalleeInFunc(funcName, body, contracts, prog, imported, calleeASTFuncs, declarable); callee != "" {
		out = append(out, SMTRejectionReason{
			Code:    RejectUnencodable,
			Message: fmt.Sprintf("Function %q calls %q whose signature uses an unencodable type %q", funcName, callee, badType),
			Hint:    "Cross-function verification cannot encode type variables, tuples, function types or bytes in a callee signature (Option/Result and other ADTs at a concrete type are fine). Give the callee a concrete signature, or inline its logic into the caller.",
		})
	}
	return out
}
