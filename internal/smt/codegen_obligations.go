package smt

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// Call-site preconditions (M-VERIFY-SOUNDNESS-COVERAGE M3, decision D1(a);
// design: design_docs/implemented/v0_61_0/m-verify-call-site-requires.md).
//
// A call to g in the verified function's body adds the call fact
// (=> $req_ok_g <g's requires on the actual arguments>). Call facts are already
// scoped by the if/let/match that evaluates the call, which is exactly the path
// condition an obligation needs. $req_ok_g is never declared; every assertion
// that mentions it binds it with let:
//
//   - the postcondition query binds every $req_ok_* to true: callee
//     preconditions are assumed at their calls (assume-guarantee);
//   - one obligation query per callee g asserts the facts with every
//     $req_ok_* false (assumptions only) and NOT the facts with $req_ok_g true:
//     sat means a reachable call to g breaks g's requires.
//
// Obligations are emitted while runtime code is encoded (obligationsOn): the
// contracted body, and the helper bodies, unrolled recursion and HOF
// specialisations it calls, whose obligations travel as $ob_ define-funs
// (codegen_def_obligations.go). Contracts are not runtime code of the caller
// and are encoded with obligations off (withoutObligations).

const reqOkPrefix = "$req_ok_"

var (
	// obligationsOn is true while EncodeFunction encodes runtime code.
	obligationsOn bool
	// activeCalleeRequires maps an inline (define-fun) callee to the Bool
	// define-fun holding its requires over its parameters.
	activeCalleeRequires map[string]string
	// activeUncheckableRequires maps a callee whose requires could not be
	// encoded to the reason: a call to it cannot be checked.
	activeUncheckableRequires map[string]string
)

// CallObligation is the query that checks every call to Callee against its requires.
type CallObligation struct {
	Callee string
	SMTLib string
	// DependsOnUninterpreted: a fact carrying this obligation mentions an
	// uninterpreted builtin, so a model may rest on a value the real function
	// never returns. Otherwise the clauses involve only exact terms and a sat
	// is a genuine violation.
	DependsOnUninterpreted bool
}

func reqOkSymbol(callee string) string { return reqOkPrefix + callee }

// addObligation records that requires must hold at the call being encoded.
func addObligation(callee, requires string) {
	if !obligationsOn || requires == "" || requires == "true" {
		return
	}
	addCallFact(fmt.Sprintf("(=> %s %s)", reqOkSymbol(callee), requires))
}

// uncheckableCall is the error for a call whose callee's requires has no encoding.
func uncheckableCall(callee, why string) error {
	return fmt.Errorf("%w: the requires of %q cannot be encoded, so a call to it cannot be checked: %s", ErrUnsupportedConstruct, callee, why)
}

// obligationsOff turns obligations off and returns the func restoring them.
func obligationsOff() func() {
	previous := obligationsOn
	obligationsOn = false
	return func() { obligationsOn = previous }
}

// withoutObligations encodes with obligations off: a callee's own contract is
// not the caller's body.
func withoutObligations(encode func() (string, error)) (string, error) {
	previous := obligationsOn
	obligationsOn = false
	defer func() { obligationsOn = previous }()
	return encode()
}

// requiresContracts returns meta's requires clauses.
func requiresContracts(meta *core.DeclMeta) []*core.Contract {
	if meta == nil {
		return nil
	}
	var out []*core.Contract
	for _, c := range meta.Contracts {
		if c != nil && c.Kind == core.RequiresKind && c.Expr != nil {
			out = append(out, c)
		}
	}
	return out
}

// encodeRequiresBody encodes requires clauses over the callee's parameters
// (already renamed by the caller's withArgRenames). Facts a clause needs (an
// uninterpreted builtin's properties) are hypotheses of that clause.
func encodeRequiresBody(reqs []*core.Contract) (string, error) {
	parts := make([]string, 0, len(reqs))
	for _, c := range reqs {
		value, facts, err := captureCallFacts(func() (string, error) {
			return withoutObligations(func() (string, error) { return EncodeExpr(c.Expr) })
		})
		if err != nil {
			return "", err
		}
		if facts != "true" {
			value = fmt.Sprintf("(=> %s %s)", facts, value)
		}
		parts = append(parts, value)
	}
	return conjoinFacts(parts...), nil
}

// contractCallObligation is the requires of a contract-path callee on the
// actual arguments of one call.
func contractCallObligation(spec *ContractSpec, subs map[string]string, value string) error {
	if !obligationsOn || len(spec.Requires) == 0 {
		return nil
	}
	parts := make([]string, 0, len(spec.Requires))
	for _, c := range spec.Requires {
		encoded, err := withoutObligations(func() (string, error) { return encodeContractExprWithSubs(c.Expr, subs, value) })
		if err != nil {
			return uncheckableCall(spec.FuncName, err.Error())
		}
		parts = append(parts, encoded)
	}
	addObligation(spec.FuncName, conjoinFacts(parts...))
	return nil
}

// bindRequiresOk binds every $req_ok_ symbol in fact: callee `on` to true and
// the others to `rest` ("true"/"false"). on == "*" binds all to rest.
func bindRequiresOk(fact string, callees []string, on, rest string) string {
	if !strings.Contains(fact, reqOkPrefix) {
		return fact
	}
	binds := make([]string, len(callees))
	for i, c := range callees {
		v := rest
		if c == on {
			v = "true"
		}
		binds[i] = fmt.Sprintf("(%s %s)", reqOkSymbol(c), v)
	}
	return fmt.Sprintf("(let (%s) %s)", strings.Join(binds, " "), fact)
}

// obligationCallees returns, sorted, the obligation names the facts carry.
func obligationCallees(facts []string) []string {
	return reqOkNames(facts...)
}

// obligationAssertions builds, for callee g, the assertions after the
// caller's requires: every fact as an assumption, then the negated
// obligations of g.
func obligationAssertions(facts, callees []string, g string) []string {
	out := make([]string, 0, len(facts)+1)
	for _, f := range facts {
		out = append(out, Assert(bindRequiresOk(f, callees, "", "false")))
	}
	withG := make([]string, len(facts))
	for i, f := range facts {
		withG[i] = bindRequiresOk(f, callees, g, "false")
	}
	return append(out, AssertNot(conjoinFacts(withG...)))
}

// leakedObligation reports a declaration that mentions an obligation symbol,
// which only the top-level assertions may bind.
func leakedObligation(decls []string) error {
	for _, d := range decls {
		if strings.Contains(d, reqOkPrefix) {
			return fmt.Errorf("internal: call-site obligation escaped into a declaration: %.120s", d)
		}
	}
	return nil
}

// obligationDependsOn reports whether any fact carrying callee's obligation
// mentions one of the uninterpreted symbols.
func obligationDependsOn(facts []string, callee string, uninterpreted []string) bool {
	ok := regexp.MustCompile(regexp.QuoteMeta(reqOkSymbol(callee)) + `[\s)]`)
	for _, f := range facts {
		if !ok.MatchString(f) {
			continue
		}
		f = expandObligationDefs(f)
		for _, u := range uninterpreted {
			if strings.Contains(f, u+" ") || strings.Contains(f, u+")") {
				return true
			}
		}
	}
	return false
}

// bindPostconditionAssumptions binds the obligation symbols of fact for the
// postcondition query: callee requires and NaN obligations are assumed (true);
// int64 range obligations are not (false).
func bindPostconditionAssumptions(fact string, callees []string) string {
	if !strings.Contains(fact, reqOkPrefix) {
		return fact
	}
	binds := make([]string, len(callees))
	for i, c := range callees {
		v := "true"
		if c == obligationInt64 {
			v = "false"
		}
		binds[i] = fmt.Sprintf("(%s %s)", reqOkSymbol(c), v)
	}
	return fmt.Sprintf("(let (%s) %s)", strings.Join(binds, " "), fact)
}
