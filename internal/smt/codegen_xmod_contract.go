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
	// ResultConst is the fresh constant name for the call result (e.g. $xmod_result_double_0).
	ResultConst string
	// SMTLib is the full SMT-LIB fragment to prepend before the caller body encoding.
	// It declares the result constant and asserts the callee's contracts.
	SMTLib string
}

// EncodeCalleeByContract emits a declare-const for the callee result and asserts
// the callee's requires/ensures contracts as axioms. This is the fallback path
// when the callee body cannot be inlined (too deep, recursive, or unencodable).
//
// callSiteIdx disambiguates multiple calls to the same callee within one function.
//
// The returned ContractCalleeDecl.SMTLib can be prepended to the caller's SMT-LIB
// program. The ResultConst field gives the name to substitute wherever the callee
// call appears in the encoded caller body.
func EncodeCalleeByContract(spec ContractSpec, args []core.CoreExpr, callSiteIdx int) (*ContractCalleeDecl, error) {
	resultConst := fmt.Sprintf("$xmod_result_%s_%d", spec.FuncName, callSiteIdx)

	var sb strings.Builder

	// Declare a fresh constant for the callee's return value.
	sb.WriteString(fmt.Sprintf("(declare-const %s %s)\n", resultConst, spec.ReturnSort))

	// Build param→arg substitution for encoding contracts.
	// contracts reference param names; we substitute actual call args.
	paramSubs, err := buildParamSubstitution(spec.Params, args)
	if err != nil {
		return nil, fmt.Errorf("callee contract param substitution: %w", err)
	}

	// Assert requires (preconditions) — these hold at the call site.
	for _, req := range spec.Requires {
		encoded, err := encodeContractExprWithSubs(req.Expr, paramSubs, resultConst)
		if err != nil {
			continue // skip unencodable preconditions rather than failing
		}
		sb.WriteString(fmt.Sprintf("(assert %s) ; %s requires\n", encoded, spec.FuncName))
	}

	// Assert ensures (postconditions) — these bind the result constant.
	for _, ens := range spec.Ensures {
		encoded, err := encodeContractExprWithSubs(ens.Expr, paramSubs, resultConst)
		if err != nil {
			continue // skip unencodable postconditions
		}
		sb.WriteString(fmt.Sprintf("(assert %s) ; %s ensures\n", encoded, spec.FuncName))
	}

	return &ContractCalleeDecl{
		Name:        spec.FuncName,
		ResultConst: resultConst,
		SMTLib:      sb.String(),
	}, nil
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
func buildParamSubstitution(params []FunctionParam, args []core.CoreExpr) (map[string]string, error) {
	subs := make(map[string]string)
	for i, p := range params {
		if i >= len(args) {
			break
		}
		encoded, err := EncodeExpr(args[i])
		if err != nil {
			return nil, fmt.Errorf("encoding arg %d for param %q: %w", i, p.Name, err)
		}
		subs[p.Name] = encoded
	}
	return subs, nil
}

// encodeContractExprWithSubs encodes a contract expression, substituting param
// names with their call-site encodings and "result" with the result constant.
func encodeContractExprWithSubs(expr core.CoreExpr, paramSubs map[string]string, resultConst string) (string, error) {
	substituted := substituteContractExpr(expr, paramSubs, resultConst)
	return EncodeExpr(substituted)
}

// substituteContractExpr walks a contract expression and replaces Var references
// to parameter names with their substituted values, and "result" with resultConst.
func substituteContractExpr(expr core.CoreExpr, subs map[string]string, resultConst string) core.CoreExpr {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *core.Var:
		if e.Name == "result" {
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
