package pipeline

// Bodies that are not part of the program but must still type-check
// (#2232, #2248): contract clauses, and the test-only bodies — named test
// blocks, properties and inline test rows — that the elaborator builds for
// checking (elaborate.CheckBody). Before these passes a contract's inference
// error was swallowed and test bodies were never inferred at all, so `check`
// passed code that `ailang test`, `ailang verify` or a reader would reject.

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/elaborate"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/link"
	"github.com/sunholo-data/ailang/internal/types"
)

// CodeContract (TC_CONTRACT_001): a requires/ensures clause does not
// type-check, or is not a bool (#2248).
const CodeContract = "TC_CONTRACT_001"

// typeCheckContracts infers every contract clause of the module as a bool
// predicate over its function's parameters (and `result`, in ensures), in
// function-name order so the first error reported is deterministic. It
// returns the checked clauses; any inference error is a TC_CONTRACT_001.
func typeCheckContracts(prog *core.Program, tc *types.CoreTypeChecker, moduleTypeEnv *types.TypeEnv) ([]core.CoreExpr, error) {
	if prog.Meta == nil {
		return nil, nil
	}
	names := make([]string, 0, len(prog.Meta))
	for name, meta := range prog.Meta {
		if meta != nil && len(meta.Contracts) > 0 {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	funcParams := extractFuncParams(prog)
	var checked []core.CoreExpr
	for _, funcName := range names {
		for _, contract := range prog.Meta[funcName].Contracts {
			if contract.Expr == nil {
				continue
			}
			// #2273: the parameters are bound at the declaration's rigid types,
			// so a clause cannot specialise its type variables either.
			sig := tc.RigidSignatureOf(funcName)
			contractEnv := contractEnvFor(funcName, funcParams, moduleTypeEnv, contract.Kind == core.EnsuresKind, sig)
			// Populates CoreTI for op-lowering (M-CONTRACT-OPLOWERING-FIX) and
			// is the contract's type check: an error fails the module.
			if _, err := tc.InferPredicateRigid(contract.Expr, contractEnv, sig); err != nil {
				return nil, contractError(funcName, contract, err)
			}
			tc.FillOperatorMethods(contract.Expr)
			checked = append(checked, contract.Expr)
		}
	}
	return checked, nil
}

// contractEnvFor is the module env plus the function's parameters (names from
// its Core lambda, types from its inferred signature) and, for an ensures
// clause only, `result`: a precondition is checked before there is one.
func contractEnvFor(funcName string, funcParams map[string][]string, moduleTypeEnv *types.TypeEnv, withResult bool, sig *types.RigidSig) *types.TypeEnv {
	env := moduleTypeEnv
	binding, err := moduleTypeEnv.Lookup(funcName)
	if err != nil {
		return env
	}
	if scheme, ok := binding.(*types.Scheme); ok && sig != nil {
		binding = sig.Instantiate(scheme) // the signature's variables at their rigid names
	}
	paramTypes, retType := extractFuncSignature(binding)
	for i, paramName := range funcParams[funcName] {
		if i < len(paramTypes) {
			env = env.Extend(paramName, paramTypes[i])
		}
	}
	if withResult && retType != nil {
		env = env.Extend("result", retType)
	}
	return env
}

func contractError(funcName string, c *core.Contract, cause error) error {
	pos := c.Expr.Span()
	return ailerrors.WrapReport(&ailerrors.Report{
		Schema: "ailang.error/v1",
		Code:   CodeContract,
		Phase:  "typecheck",
		Message: fmt.Sprintf("%s clause of %s at %s does not type-check (a contract is a bool predicate over the parameters%s): %v%s",
			c.Kind, funcName, c.Location, resultNote(c.Kind), cause, eqNote(cause)),
		Span: &ast.Span{Start: pos, End: pos},
		Data: map[string]any{"function": funcName, "kind": c.Kind.String()},
		Fix:  &ailerrors.Fix{Suggestion: "fix the contract expression; it is type-checked like a function body"},
	})
}

// eqNote: a contract needs Eq for == exactly as a body does (the ruling of
// 2026-11-12), which authors who wrote enum == for the verifier do not expect.
func eqNote(cause error) string {
	if strings.Contains(cause.Error(), "No instance for Eq[") {
		return " (== needs Eq in a requires/ensures clause exactly as in a function body)"
	}
	return ""
}

func resultNote(k core.ContractKind) string {
	if k == core.EnsuresKind {
		return " and `result`"
	}
	return ""
}

// typeCheckTestBodies infers each test-only body against the module's
// declarations, as the harness's entry for it would be inferred.
func typeCheckTestBodies(bodies []elaborate.CheckBody, tc *types.CoreTypeChecker, moduleTypeEnv *types.TypeEnv) error {
	for _, b := range bodies {
		if _, _, _, _, err := tc.InferWithConstraints(b.Expr, moduleTypeEnv); err != nil {
			return elaborate.TestBodyError(b, err)
		}
	}
	return nil
}

// checkTestBodyEffects requires each test and property body to be pure, as
// the harness compiles it (`pure func __namedtest_<k>`), using the effect
// checker's own collector. Runs after FinalizeSubstitutions.
func checkTestBodyEffects(surface *ast.File, bodies []elaborate.CheckBody, tc *types.CoreTypeChecker) error {
	var collector *effectCollector
	for _, b := range bodies {
		if !b.Pure {
			continue
		}
		if collector == nil {
			declared, err := declaredFuncEffects(surface)
			if err != nil {
				return err
			}
			collector = &effectCollector{typeInfo: tc.CoreTI, declared: declared,
				maskLookup: tc.ApplicationEffects, valueLookup: tc.EffectValueType, invariant: new(error)}
		}
		required := eraseGhostEffects(collector.collect(b.Expr))
		if *collector.invariant != nil {
			return elaborate.TestBodyError(b, *collector.invariant)
		}
		if !types.SubsumeEffectRows(required, nil) {
			return elaborate.TestBodyError(b, fmt.Errorf("the body requires effects %s, but test and property bodies are checked pure; effectful checks belong in an exported entry run with `ailang run --caps ...`",
				types.FormatEffectRow(required)))
		}
	}
	return nil
}

// declaredFuncEffects is each module function's declared effect row, as the
// effect validator reads it.
func declaredFuncEffects(surface *ast.File) (map[string]*types.Row, error) {
	declared := make(map[string]*types.Row)
	if surface == nil {
		return declared, nil
	}
	for _, fn := range surface.Funcs {
		row, err := types.ElaborateEffectRowWithBudgets(fn.Effects)
		if err != nil {
			return nil, fmt.Errorf("elaborate effects for function %q: %w", fn.Name, err)
		}
		declared[fn.Name] = row
	}
	return declared, nil
}

// checkBodyExprs is the Core of each test-only body, for the passes that
// walk them like declarations (match exhaustiveness).
func checkBodyExprs(bodies []elaborate.CheckBody) []core.CoreExpr {
	out := make([]core.CoreExpr, 0, len(bodies))
	for _, b := range bodies {
		out = append(out, b.Expr)
	}
	return out
}

// hasTestOnlyBodies reports whether file has any body elaborate.CheckBodies
// would return: a test or property block, an inline row or forall property.
func hasTestOnlyBodies(file *ast.File) bool {
	if file == nil {
		return false
	}
	for _, d := range file.Decls {
		switch d.(type) {
		case *ast.TestDecl, *ast.PropertyDecl:
			return true
		}
	}
	for _, fn := range file.Funcs {
		if len(fn.Tests) > 0 {
			return true
		}
		for _, p := range fn.Properties {
			if p.Kind == ast.PropertyKind {
				return true
			}
		}
	}
	return false
}

// recordTypeDeclSites tells the instance environment where modID declares
// each type, so a missing-Eq error can say "add `deriving (Eq)` to type X
// (declared at file:line)". Diagnostics only; runs on a cache hit too.
func recordTypeDeclSites(env *types.InstanceEnv, modID link.ModuleID, file *ast.File) {
	if env == nil || file == nil {
		return
	}
	for _, d := range file.Decls {
		if td, ok := d.(*ast.TypeDecl); ok && td.Pos.Line > 0 {
			_, record := td.Definition.(*ast.RecordType)
			// Forward slashes on every OS and for every load path: a root's
			// site keeps the CLI argument, a dependency's went through
			// filepath.Join, and the message must read the same either way.
			site := fmt.Sprintf("%s:%d", filepath.ToSlash(td.Pos.File), td.Pos.Line)
			env.RecordTypeDecl(string(modID), td.Name, site, record)
		}
	}
}
