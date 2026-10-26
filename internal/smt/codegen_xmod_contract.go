package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
)

// ContractSpec holds the SMT-encodable contracts for a callee function.
type ContractSpec struct {
	// FuncName is the callee function name.
	FuncName string
	// Params are the callee's surface parameters (for substitution).
	Params []FunctionParam
	// ReturnSort is the SMT sort of the callee's return value.
	ReturnSort string
	// Requires are the pre-condition contracts (may be empty).
	Requires []*core.Contract
	// Ensures are the post-condition contracts (must be non-empty to be useful).
	Ensures []*core.Contract
}

// ContractCalleeDecl holds the SMT-LIB declarations for a contract-based callee.
type ContractCalleeDecl struct {
	// Name is the callee function name.
	Name string
	// SpecFun is the uninterpreted function standing for the callee (`<f>__spec`).
	// Each call site encodes as `(<f>__spec args…)`.
	SpecFun string
	// Arity is the number of SpecFun arguments (the callee's surface parameters).
	Arity int
	// SMTLib declares SpecFun and asserts the callee's contract axiom. It is
	// prepended to the caller's SMT-LIB program.
	SMTLib string
}

// contractCalleeRef is what encodeApp needs to encode a call to a contract callee.
type contractCalleeRef struct {
	SpecFun string
	Arity   int
}

// contractSpecFunName names the uninterpreted function for a contract callee.
func contractSpecFunName(funcName string) string { return funcName + "__spec" }

// EncodeCalleeContract encodes a callee that is seen only through its contract
// (too deep, recursive, or otherwise not inlinable) as one uninterpreted function
// plus one axiom:
//
//	(declare-fun f__spec (S1 … Sn) R)
//	(assert (forall ((q1 S1) … (qn Sn))
//	  (! (=> requires[q] ensures[q, result := (f__spec q…)]) :pattern ((f__spec q…)))))
//
// The pattern makes Z3 instantiate the axiom once per distinct call term, with
// that call's arguments. So each call has its own result, and two calls with the
// same arguments share one (functional consistency). It replaces a single shared
// result constant that made `f(a) - f(a + 1)` provably 0 (#1961).
//
// requires GUARDS the ensures: the callee only guarantees its ensures when its
// requires holds, so a call outside the requires gets an unconstrained result.
// If a requires clause cannot be encoded, the guard cannot be formed and no
// ensures is assumed (weaker, sound). Discharging requires as a call-site proof
// obligation is a separate decision (verify-call-site-precondition-not-checked.md).
// An ensures clause that cannot be encoded is dropped, which is also weaker.
func EncodeCalleeContract(spec ContractSpec) (*ContractCalleeDecl, error) {
	specFun := contractSpecFunName(spec.FuncName)
	bound := make([]string, len(spec.Params))
	binders := make([]string, len(spec.Params))
	sorts := make([]string, len(spec.Params))
	renames := make(map[string]string, len(spec.Params)+1)
	subs := make(map[string]string, len(spec.Params))
	for i, p := range spec.Params {
		sort, err := MapType(p.Type)
		if err != nil {
			return nil, fmt.Errorf("%w: contract callee %q parameter %q: %v", ErrUnresolvableTypes, spec.FuncName, p.Name, err)
		}
		bound[i] = fmt.Sprintf("$q_%s_%s", spec.FuncName, p.Name)
		binders[i] = fmt.Sprintf("(%s %s)", bound[i], sort)
		sorts[i] = sort
		renames[p.Name] = bound[i]
		subs[p.Name] = bound[i]
	}
	specTerm := specFun
	if len(bound) > 0 {
		specTerm = fmt.Sprintf("(%s %s)", specFun, strings.Join(bound, " "))
	}
	renames["result"] = specTerm

	var sb strings.Builder
	fmt.Fprintf(&sb, "(declare-fun %s (%s) %s)\n", specFun, strings.Join(sorts, " "), spec.ReturnSort)

	axiom, ok := encodeContractAxiom(spec, renames, subs, specTerm)
	if ok {
		if len(bound) == 0 {
			fmt.Fprintf(&sb, "(assert %s) ; %s contract\n", axiom, spec.FuncName)
		} else {
			fmt.Fprintf(&sb, "(assert (forall (%s) (! %s :pattern (%s)))) ; %s contract\n",
				strings.Join(binders, " "), axiom, specTerm, spec.FuncName)
		}
	}
	return &ContractCalleeDecl{Name: spec.FuncName, SpecFun: specFun, Arity: len(bound), SMTLib: sb.String()}, nil
}

// encodeContractAxiom encodes `requires => ensures` over the quantifier's bound
// names. The callee's parameters and `result` are bound through the parameter
// rename table, which the scoped let/match encoders already keep lexical, so the
// caller's own `$p_` renames can neither capture nor leak into the axiom. ok is
// false when there is nothing sound to assert.
func encodeContractAxiom(spec ContractSpec, renames, subs map[string]string, specTerm string) (string, bool) {
	previous := activeParamRenames
	activeParamRenames = renames
	defer func() { activeParamRenames = previous }()

	encodeAll := func(contracts []*core.Contract, dropOnError bool) ([]string, bool) {
		var out []string
		for _, c := range contracts {
			if c == nil || c.Expr == nil {
				continue
			}
			enc, err := EncodeExpr(substituteContractExpr(c.Expr, subs, specTerm))
			if err != nil {
				if dropOnError {
					continue
				}
				return nil, false
			}
			out = append(out, enc)
		}
		return out, true
	}
	reqs, ok := encodeAll(spec.Requires, false)
	if !ok {
		return "", false // no guard → no ensures may be assumed
	}
	ens, _ := encodeAll(spec.Ensures, true)
	if len(ens) == 0 {
		return "", false
	}
	body := conjoin(ens)
	if len(reqs) > 0 {
		body = fmt.Sprintf("(=> %s %s)", conjoin(reqs), body)
	}
	return body, true
}

// conjoin renders a conjunction, without a wrapper for a single term.
func conjoin(terms []string) string {
	if len(terms) == 1 {
		return terms[0]
	}
	return fmt.Sprintf("(and %s)", strings.Join(terms, " "))
}

// encodeContractCalleeCall encodes one call to a contract callee as an
// application of its spec function to this call's arguments.
func encodeContractCalleeCall(name string, ref contractCalleeRef, args []core.CoreExpr) (string, error) {
	nonUnit := 0
	for _, a := range args {
		if !isUnitLiteral(a) {
			nonUnit++
		}
	}
	if nonUnit != ref.Arity {
		return "", fmt.Errorf("%w: call to contract callee %q has %d arguments, its contract has %d parameters",
			ErrUnresolvableTypes, name, nonUnit, ref.Arity)
	}
	return encodeUserFunctionCall(ref.SpecFun, args)
}

// BuildContractSpec builds a ContractSpec for a callee from its core.DeclMeta
// and surface parameter information.
func BuildContractSpec(
	funcName string,
	meta *core.DeclMeta,
	params []FunctionParam,
	returnSort string,
) ContractSpec {
	var requires, ensures []*core.Contract
	for _, c := range meta.Contracts {
		switch c.Kind {
		case core.RequiresKind:
			requires = append(requires, c)
		case core.EnsuresKind:
			ensures = append(ensures, c)
		}
	}
	if returnSort == "" {
		returnSort = "Int"
	}
	return ContractSpec{
		FuncName:   funcName,
		Params:     params,
		ReturnSort: returnSort,
		Requires:   requires,
		Ensures:    ensures,
	}
}

// substituteContractExpr walks a contract expression and replaces "result" with
// resultConst, and VarGlobal references to parameter names with their substituted
// values. A plain Var parameter is NOT substituted here: EncodeCalleeContract binds
// those through the parameter rename table, whose let/match handling is lexical,
// so a let that shadows a parameter name is not captured.
func substituteContractExpr(expr core.CoreExpr, subs map[string]string, resultConst string) core.CoreExpr {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *core.Var:
		if e.Name == "result" {
			return &core.VarGlobal{Ref: core.GlobalRef{Module: "$const", Name: resultConst}}
		}
		return e
	case *core.VarGlobal:
		// The AILANG elaborator may produce VarGlobal for "result" when it finds
		// the name in globalEnv (e.g., if a function named "result" is in scope).
		// Treat any non-sentinel VarGlobal named "result" as the return-value pseudo-var.
		if e.Ref.Name == "result" && e.Ref.Module != "$builtin" && e.Ref.Module != "$encoded" && e.Ref.Module != "$const" {
			return &core.VarGlobal{Ref: core.GlobalRef{Module: "$const", Name: resultConst}}
		}
		// Substitute param names in VarGlobal references too (covers edge cases
		// where elaboration uses VarGlobal for function parameters).
		if replacement, ok := subs[e.Ref.Name]; ok && e.Ref.Module != "$builtin" {
			return &core.VarGlobal{Ref: core.GlobalRef{Module: "$encoded", Name: replacement}}
		}
		return e
	case *core.App:
		newFunc := substituteContractExpr(e.Func, subs, resultConst)
		newArgs := make([]core.CoreExpr, len(e.Args))
		for i, arg := range e.Args {
			newArgs[i] = substituteContractExpr(arg, subs, resultConst)
		}
		return &core.App{Func: newFunc, Args: newArgs}
	case *core.If:
		return &core.If{
			Cond: substituteContractExpr(e.Cond, subs, resultConst),
			Then: substituteContractExpr(e.Then, subs, resultConst),
			Else: substituteContractExpr(e.Else, subs, resultConst),
		}
	case *core.Let:
		return &core.Let{
			Name:  e.Name,
			Value: substituteContractExpr(e.Value, subs, resultConst),
			Body:  substituteContractExpr(e.Body, subs, resultConst),
		}
	case *core.BinOp:
		return &core.BinOp{
			Op:    e.Op,
			Left:  substituteContractExpr(e.Left, subs, resultConst),
			Right: substituteContractExpr(e.Right, subs, resultConst),
		}
	case *core.UnOp:
		return &core.UnOp{
			Op:      e.Op,
			Operand: substituteContractExpr(e.Operand, subs, resultConst),
		}
	case *core.DictApp:
		// Type-class dispatch: substitute into arguments (dict itself has no free vars).
		newArgs := make([]core.CoreExpr, len(e.Args))
		for i, arg := range e.Args {
			newArgs[i] = substituteContractExpr(arg, subs, resultConst)
		}
		return &core.DictApp{Dict: e.Dict, Method: e.Method, Args: newArgs}
	case *core.DictAbs:
		return &core.DictAbs{Body: substituteContractExpr(e.Body, subs, resultConst)}
	default:
		return e
	}
}
