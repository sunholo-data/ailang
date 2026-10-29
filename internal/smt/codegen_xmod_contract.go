package smt

import (
	"fmt"
	"sort"
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
	// ResultConst is the actual UF call term (legacy API field name).
	ResultConst string
	// SMTLib is the full SMT-LIB fragment to prepend before the caller body encoding.
	// It declares the UF and asserts this invocation's guarded fact.
	SMTLib string
}

// EncodeCalleeByContract instantiates a specification only at the supplied call.
// ResultConst is retained as an API field name, but is now a UF application:
// equal arguments share a result and different arguments are not conflated.
// The finite fact is guarded by requires; no global quantified axiom is emitted.
func EncodeCalleeByContract(spec ContractSpec, args []core.CoreExpr, _ int) (*ContractCalleeDecl, error) {
	declaration, err := declareContractFunction(spec)
	if err != nil {
		return nil, err
	}
	subs, err := buildParamSubstitution(spec.Params, args)
	if err != nil {
		return nil, err
	}
	result, err := encodeUserFunctionCall(contractSymbol(spec), args)
	if err != nil {
		return nil, err
	}
	fact := instantiateContract(spec, subs, result)
	if fact != "true" {
		declaration += "\n" + Assert(fact)
	}
	return &ContractCalleeDecl{Name: spec.FuncName, ResultConst: result, SMTLib: declaration}, nil
}

func contractSymbol(spec ContractSpec) string { return "$spec_" + spec.FuncName }

func declareContractFunction(spec ContractSpec) (string, error) {
	var sorts []string
	for _, p := range spec.Params {
		if isUnitFunctionParam(p) {
			continue
		}
		sort, err := MapType(p.Type)
		if err != nil {
			return "", fmt.Errorf("%w: contract parameter %q: %v", ErrUnresolvableTypes, p.Name, err)
		}
		sorts = append(sorts, sort)
	}
	return fmt.Sprintf("(declare-fun %s (%s) %s)", contractSymbol(spec), strings.Join(sorts, " "), spec.ReturnSort), nil
}

// Unknown requires invalidates the entire implication. Unknown ensures can
// safely contribute no clause. The caller retains an unconstrained UF value.
func instantiateContract(spec ContractSpec, subs map[string]string, result string) string {
	var requires, requiresFacts, ensures []string
	for _, c := range spec.Requires {
		encoded, facts, err := captureCallFacts(func() (string, error) { return encodeContractExprWithSubs(c.Expr, subs, result) })
		if err != nil {
			return "true"
		}
		requires = append(requires, encoded)
		requiresFacts = append(requiresFacts, facts)
	}
	for _, c := range spec.Ensures {
		encoded, facts, err := captureCallFacts(func() (string, error) { return encodeContractExprWithSubs(c.Expr, subs, result) })
		if err == nil {
			ensures = append(ensures, conjoinFacts(facts, encoded))
		}
	}
	if len(ensures) == 0 {
		return "true"
	}
	return conjoinFacts(conjoinFacts(requiresFacts...), fmt.Sprintf("(=> %s %s)", conjoinFacts(requires...), conjoinFacts(ensures...)))
}

func conjoinFacts(facts ...string) string {
	var nontrivial []string
	for _, fact := range facts {
		if fact != "" && fact != "true" {
			nontrivial = append(nontrivial, fact)
		}
	}
	if len(nontrivial) == 0 {
		return "true"
	}
	if len(nontrivial) == 1 {
		return nontrivial[0]
	}
	return "(and " + strings.Join(nontrivial, " ") + ")"
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

// buildParamSubstitution maps parameter names to their encoded call-site arguments.
func isUnitFunctionParam(p FunctionParam) bool {
	return p.Type != nil && (p.Type.String() == "()" || p.Type.String() == "unit")
}

// Nullary surface functions erase their synthesized unit parameter/argument.
// Arity is checked after the same erasure used by value helper applications.
func buildParamSubstitution(params []FunctionParam, args []core.CoreExpr) (map[string]string, error) {
	var nonunitParams []FunctionParam
	var nonunitArgs []core.CoreExpr
	for _, p := range params {
		if !isUnitFunctionParam(p) {
			nonunitParams = append(nonunitParams, p)
		}
	}
	for _, arg := range args {
		if !isUnitLiteral(arg) {
			nonunitArgs = append(nonunitArgs, arg)
		}
	}
	if len(nonunitParams) != len(nonunitArgs) {
		return nil, fmt.Errorf("contract call expects %d arguments, got %d", len(nonunitParams), len(nonunitArgs))
	}
	subs := make(map[string]string)
	for i, p := range nonunitParams {
		value, err := EncodeExpr(nonunitArgs[i])
		if err != nil {
			return nil, fmt.Errorf("encoding argument %d for %q: %w", i, p.Name, err)
		}
		subs[p.Name] = value
	}
	return subs, nil
}

// encodeContractExprWithSubs encodes a contract expression, substituting param
// names with their call-site encodings and "result" with the result constant.
func encodeContractExprWithSubs(expr core.CoreExpr, paramSubs map[string]string, resultConst string) (string, error) {
	if expr == nil {
		return "", fmt.Errorf("nil contract predicate")
	}
	// Bind caller expressions OUTSIDE all callee-local binders. Parameter-map
	// shadowing alone cannot stop textual SMT arguments from being captured.
	occupied := expr.String() + resultConst
	for _, reserved := range activeParamRenames {
		occupied += reserved
	}
	for _, v := range paramSubs {
		occupied += v
	}
	prefix := "$spec_sub_0_"
	for i := 1; strings.Contains(occupied, prefix); i++ {
		prefix = fmt.Sprintf("$spec_sub_%d_", i)
	}
	var keys []string
	for k := range paramSubs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	aliases := make(map[string]string, len(keys))
	var bindings []string
	for i, k := range keys {
		alias := fmt.Sprintf("%s%d", prefix, i)
		aliases[k] = alias
		bindings = append(bindings, fmt.Sprintf("(%s %s)", alias, paramSubs[k]))
	}
	resultAlias := prefix + "result"
	bindings = append(bindings, fmt.Sprintf("(%s %s)", resultAlias, resultConst))
	substituted := substituteContractExpr(expr, aliases, resultAlias)
	value, facts, err := captureCallFacts(func() (string, error) { return EncodeExpr(substituted) })
	if err != nil {
		return "", err
	}
	wrap := func(text string) string { return fmt.Sprintf("(let (%s) %s)", strings.Join(bindings, " "), text) }
	if facts != "true" {
		addCallFact(wrap(facts))
	}
	return wrap(value), nil
}

// substituteContractExpr walks a contract expression and replaces Var references
// to parameter names with their substituted values, and "result" with resultConst.
func substituteContractExpr(expr core.CoreExpr, subs map[string]string, resultConst string) core.CoreExpr {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *core.Var:
		if e.Name == "result" && resultConst != "" {
			return &core.VarGlobal{Ref: core.GlobalRef{Module: "$const", Name: resultConst}}
		}
		if replacement, ok := subs[e.Name]; ok {
			// Return a pre-encoded sentinel that EncodeExpr will pass through.
			// We use a VarGlobal with Module="$encoded" to signal pre-encoded text.
			return &core.VarGlobal{Ref: core.GlobalRef{Module: "$encoded", Name: replacement}}
		}
		return e
	case *core.VarGlobal:
		// The AILANG elaborator may produce VarGlobal for "result" when it finds
		// the name in globalEnv (e.g., if a function named "result" is in scope).
		// Treat any non-sentinel VarGlobal named "result" as the return-value pseudo-var.
		if e.Ref.Name == "result" && resultConst != "" && e.Ref.Module != "$builtin" && e.Ref.Module != "$encoded" && e.Ref.Module != "$const" {
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
		local := withoutContractBindings(subs, []string{e.Name})
		localResult := resultConst
		if e.Name == "result" {
			localResult = ""
		}
		return &core.Let{
			Name:  e.Name,
			Value: substituteContractExpr(e.Value, subs, resultConst),
			Body:  substituteContractExpr(e.Body, local, localResult),
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
	case *core.Intrinsic:
		out := *e
		out.Args = make([]core.CoreExpr, len(e.Args))
		for i, arg := range e.Args {
			out.Args[i] = substituteContractExpr(arg, subs, resultConst)
		}
		return &out
	case *core.Match:
		out := *e
		out.Scrutinee = substituteContractExpr(e.Scrutinee, subs, resultConst)
		out.Arms = append([]core.MatchArm(nil), e.Arms...)
		for i, arm := range e.Arms {
			names := patternBinders(arm.Pattern)
			local := withoutContractBindings(subs, names)
			localResult := resultConst
			for _, n := range names {
				if n == "result" {
					localResult = ""
				}
			}
			out.Arms[i].Body = substituteContractExpr(arm.Body, local, localResult)
			out.Arms[i].Guard = substituteContractExpr(arm.Guard, local, localResult)
		}
		return &out
	case *core.Record:
		out := *e
		out.Fields = make(map[string]core.CoreExpr, len(e.Fields))
		for k, v := range e.Fields {
			out.Fields[k] = substituteContractExpr(v, subs, resultConst)
		}
		return &out
	case *core.RecordAccess:
		out := *e
		out.Record = substituteContractExpr(e.Record, subs, resultConst)
		return &out
	case *core.RecordUpdate:
		out := *e
		out.Base = substituteContractExpr(e.Base, subs, resultConst)
		out.Updates = make(map[string]core.CoreExpr, len(e.Updates))
		for k, v := range e.Updates {
			out.Updates[k] = substituteContractExpr(v, subs, resultConst)
		}
		return &out
	case *core.List:
		out := *e
		out.Elements = make([]core.CoreExpr, len(e.Elements))
		for i, v := range e.Elements {
			out.Elements[i] = substituteContractExpr(v, subs, resultConst)
		}
		return &out
	case *core.Forall:
		out := *e
		out.Lo = substituteContractExpr(e.Lo, subs, resultConst)
		out.Hi = substituteContractExpr(e.Hi, subs, resultConst)
		localResult := resultConst
		if e.Var == "result" {
			localResult = ""
		}
		out.Body = substituteContractExpr(e.Body, withoutContractBindings(subs, []string{e.Var}), localResult)
		return &out
	case *core.DictAbs:
		return &core.DictAbs{Body: substituteContractExpr(e.Body, subs, resultConst)}
	default:
		return e
	}
}

func withoutContractBindings(subs map[string]string, names []string) map[string]string {
	local := make(map[string]string, len(subs))
	for k, v := range subs {
		local[k] = v
	}
	for _, name := range names {
		delete(local, name)
	}
	return local
}
