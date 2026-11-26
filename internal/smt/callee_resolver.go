package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// crossModuleInlineDepth is the maximum number of hops through imported-module
// function bodies we will follow before refusing to inline (falling back to M3
// contract-based reasoning).
const crossModuleInlineDepth = 3

// CalleeInfo describes a resolved callee function for SMT encoding.
type CalleeInfo struct {
	Name       string
	Params     []FunctionParam
	Body       core.CoreExpr
	ReturnSort string
	ADTTypes   map[string][]ADTVariant
}

// CalleeDef holds a ready-to-emit SMT-LIB define-fun for a callee.
type CalleeDef struct {
	ConditionName string        // Paired Bool helper with exactly the value definition's parameters.
	Contract      *ContractSpec // Finite actual-call template; never a global axiom.
	Name          string
	SMTLib        string // Value/condition definitions or typed contract UF declaration
	ADTDecl       string // Any ADT declarations needed (may be empty)
	IsContract    bool   // True when using contract-as-spec fallback (not define-fun)
	ResultConst   string // Non-empty when IsContract: the UF symbol applied to actual arguments
	// HOFBoundedDepth > 0: the callee's body specialised HOF calls to this depth,
	// so any proof through it is bounded (see EncodeResult.HOFBoundedDepth).
	HOFBoundedDepth int
	// RequiresName is the Bool define-fun holding the callee's requires over
	// its parameters; UncheckableRequires says why it could not be encoded.
	// Each call in the contracted body must meet it (codegen_obligations.go).
	RequiresName        string
	UncheckableRequires string
	// Unrolled: a self-recursive callee encoded by unrollRecursiveCallee.
	Unrolled bool
	// Opaque, when set, says why this helper is an uninterpreted function
	// (declareOpaqueHelper): its body could not be encoded.
	Opaque string
}

// ResolveCallees finds all user-defined function calls in the body,
// resolves their Core bodies from the program (and optionally from
// importedPrograms for cross-module calls), and returns ordered
// define-fun declarations for SMT-LIB emission.
//
// The returned definitions are topologically ordered: if A calls B,
// B's definition appears before A's. Circular calls are detected
// and produce an error.
//
// importedPrograms may be nil — when nil, only same-module callees are resolved.
func ResolveCallees(
	funcName string,
	body core.CoreExpr,
	prog *core.Program,
	surfaceParams map[string][]FunctionParam,
	surfaceReturnSorts map[string]string,
	adtTypes map[string][]ADTVariant,
	importedPrograms ...map[string]*core.Program,
) ([]CalleeDef, error) {
	return ResolveCalleesFromRoots(funcName, []core.CoreExpr{body}, prog,
		surfaceParams, surfaceReturnSorts, adtTypes, importedPrograms...)
}

// ResolveCalleesFromRoots is ResolveCallees over several root expressions — the
// function body plus its requires/ensures predicates — so a user function called
// only from a contract (e.g. `ensures { result <= cap() }`) is also defined.
func ResolveCalleesFromRoots(
	funcName string,
	roots []core.CoreExpr,
	prog *core.Program,
	surfaceParams map[string][]FunctionParam,
	surfaceReturnSorts map[string]string,
	adtTypes map[string][]ADTVariant,
	importedPrograms ...map[string]*core.Program,
) ([]CalleeDef, error) {
	return resolveCalleesFromRoots(funcName, roots, prog, surfaceParams, surfaceReturnSorts, adtTypes, nil, importedPrograms...)
}

func resolveCalleesFromRoots(funcName string, roots []core.CoreExpr, prog *core.Program, surfaceParams map[string][]FunctionParam, surfaceReturnSorts map[string]string, adtTypes map[string][]ADTVariant, declarations map[*core.DeclMeta]*ast.FuncDecl, importedPrograms ...map[string]*core.Program) ([]CalleeDef, error) {
	if prog == nil {
		return nil, nil
	}

	var imported map[string]*core.Program
	if len(importedPrograms) > 0 {
		imported = importedPrograms[0]
	}

	// Find all user-defined function calls in the roots (same-module + cross-module)
	callees := collectCalleeCallsFromRoots(roots, funcName, prog, imported)
	if len(callees) == 0 {
		return nil, nil
	}

	// Callee bodies are encoded in dependency order; register each successfully
	// defined callee as it is emitted so a later callee's call to it encodes as a
	// define-fun application (dropping unit args of nullary callees) rather than
	// falling through to the constructor path. Restore the caller's view after.
	previousSpecs, previousContractCallees, previousConditions := activeContractSpecs, activeContractCallees, activeHelperConditions
	previousReqs, previousUncheckable := activeCalleeRequires, activeUncheckableRequires
	activeContractSpecs = make(map[string]*ContractSpec)
	activeContractCallees = make(map[string]string)
	activeHelperConditions = make(map[string]string)
	activeCalleeRequires = make(map[string]string)
	activeUncheckableRequires = make(map[string]string)
	defer func() {
		activeContractSpecs = previousSpecs
		activeContractCallees = previousContractCallees
		activeHelperConditions = previousConditions
		activeCalleeRequires, activeUncheckableRequires = previousReqs, previousUncheckable
	}()
	prevResolved := activeResolvedCallees
	resolvedSoFar := make(map[string]bool)
	activeResolvedCallees = resolvedSoFar
	defer func() { activeResolvedCallees = prevResolved }()

	// Build topological order; members of a call cycle are called opaquely
	// (or by their contract) instead of being defined.
	order, cyclic := topoSort(callees, funcName, prog, imported)
	noteCalleeOrder(order)

	// Encode each callee as a define-fun
	var defs []CalleeDef
	for _, calleeName := range order {
		// Look up body in current program first, then imported programs
		calleeBody, calleeProg := findFuncBodyInAnyProg(calleeName, prog, imported)
		if calleeBody == nil || calleeProg == nil {
			recordCalleeFailure(calleeName, "its definition was not found")
			continue
		}
		if bad := activeUnencodableSignatures[calleeName]; bad != "" {
			// No sort to define it, or to call it opaquely, at: a helper
			// calling it fails to encode and is called opaquely itself.
			recordCalleeFailure(calleeName, fmt.Sprintf("its signature uses an unencodable type %q", bad))
			continue
		}

		// Unwrap lambda to get inner body
		_, innerBody := unwrapLambda(calleeBody)

		// Check if callee is SMT-encodable
		meta := calleeProg.Meta[calleeName]
		if meta == nil {
			recordCalleeFailure(calleeName, "it has no declaration metadata")
			continue
		}

		// Get params and return sort
		params := surfaceParams[calleeName]

		// Contract-as-spec reasoning must not bypass the declaration row gate.
		if !smtAdmissible(meta, declarations[meta]) {
			return nil, inadmissibleCalleeError{calleeName}
		}
		returnSort := surfaceReturnSorts[calleeName]
		if returnSort == "" {
			// No annotation: the checked type says what it returns. An
			// invented Int made any other return a Z3 sort error.
			if returnSort = checkedReturnSort(calleeBody, calleeProg); returnSort == "" {
				recordCalleeFailure(calleeName, "it has no return annotation, and its checked return type has no SMT sort")
				continue
			}
		}
		encodable, rejections := IsSMTEncodableForCallee(calleeName, meta, calleeBody, declarations[meta])
		if cycle := cyclic[calleeName]; cycle != "" {
			// A define-fun cannot refer forward, so a cycle member is never
			// defined: its contract stands in for it, else it is opaque.
			encodable = false
			rejections = []SMTRejectionReason{{Code: RejectRecursive, Message: "it is mutually recursive (" + cycle + ")"}}
		}
		// A self-recursive callee with no ensures has no contract to stand in
		// for it: unroll it instead of dropping it (callers were skipped).
		unrollSelf := !encodable && cyclic[calleeName] == "" && onlyRecursive(rejections) && len(BuildContractSpec(calleeName, meta, params, returnSort).Ensures) == 0
		if !encodable && !unrollSelf {
			// Publish only a typed UF declaration and the original template.
			// Actual arguments and selected-path facts are instantiated by the caller.
			spec := BuildContractSpec(calleeName, meta, params, returnSort)
			if len(spec.Ensures) > 0 {
				declaration, err := declareContractFunction(spec)
				if err != nil {
					return nil, err
				}
				defs = append(defs, CalleeDef{Name: calleeName, SMTLib: declaration, IsContract: true, ResultConst: contractSymbol(spec), Contract: &spec})
				activeContractSpecs[calleeName] = &spec
				activeContractCallees[calleeName] = contractSymbol(spec)
			} else {
				defs = appendSpecOrOpaque(defs, calleeName, meta, params, returnSort, rejectionsReason(rejections))
			}
			continue
		}

		// Encode the callee body (under the callee program's type info)
		var previousTI types.CoreTypeInfo
		if activeADTContext != nil {
			previousTI = activeADTContext.ti
			activeADTContext.ti = activeADTContext.programTI[calleeProg]
		}
		// Literal-lambda HOF calls in the callee are specialised exactly as in a
		// contracted body; their chains are declared ahead of the callee.
		var hofDecls []string
		hofDepth := 0
		if r := InlineHOFCalls(calleeName, innerBody, defaultHOFInlineDepth); r != nil {
			if r.Unsupported != "" {
				if activeADTContext != nil {
					activeADTContext.ti = previousTI
				}
				// Like any body that does not encode: by contract, or opaque.
				defs = appendSpecOrOpaque(defs, calleeName, meta, params, surfaceReturnSorts[calleeName], r.Unsupported)
				continue
			}
			innerBody = r.NewBody
			for _, spec := range r.Specializations {
				registerSpecParamSorts(spec)
				hofDecls = append(hofDecls, spec.Declarations...)
				resolvedSoFar[spec.TopLevelName] = true
			}
			if len(r.Specializations) > 0 {
				hofDepth = defaultHOFInlineDepth
			}
		}
		if unrollSelf {
			smtDef, defParams, uerr := unrollRecursiveCallee(calleeName, params, innerBody, returnSort)
			if activeADTContext != nil {
				activeADTContext.ti = previousTI
			}
			if uerr != nil {
				defs = appendSpecOrOpaque(defs, calleeName, meta, params, surfaceReturnSorts[calleeName], failureReason(uerr))
				continue // as for any callee body we cannot encode
			}
			if len(hofDecls) > 0 {
				smtDef = strings.Join(hofDecls, "\n") + "\n" + smtDef
			}
			reqName, reqUncheckable := calleeRequiresDef(calleeName, meta, params, defParams, &smtDef)
			registerCalleeRequires(calleeName, reqName, reqUncheckable)
			defs = append(defs, CalleeDef{
				Name:                calleeName,
				SMTLib:              smtDef,
				HOFBoundedDepth:     defaultCalleeUnrollDepth,
				Unrolled:            true,
				RequiresName:        reqName,
				UncheckableRequires: reqUncheckable,
			})
			resolvedSoFar[calleeName] = true
			continue
		}
		var bodyExpr, bodyFacts string
		var bounded map[string]bool
		defParams, _, err := withArgRenames(params, func() (string, error) {
			var encErr error
			bodyExpr, bodyFacts, bounded, encErr = captureDefinition(func() (string, error) { return EncodeExpr(innerBody) })
			return bodyExpr, encErr
		})
		if activeADTContext != nil {
			activeADTContext.ti = previousTI
		}
		if err != nil {
			// A callee we cannot encode is called opaquely when its signature
			// allows; otherwise a caller that reaches it reports why.
			defs = appendSpecOrOpaque(defs, calleeName, meta, params, surfaceReturnSorts[calleeName], failureReason(err))
			continue
		}

		// Build define-fun (parameters bound under $a_ names: see argSymbol)
		smtDef := buildDefineFun(calleeName, defParams, returnSort, bodyExpr)
		if len(hofDecls) > 0 {
			smtDef = strings.Join(hofDecls, "\n") + "\n" + smtDef
		}
		// The body's obligations become $ob_ define-funs; the rest of its
		// facts, with every obligation off, is its condition.
		obDecls, bodyFacts := definitionObligations(calleeName, defParams, bodyFacts, bounded)
		for _, d := range obDecls {
			smtDef += "\n" + d
		}
		conditionName := ""
		if bodyFacts != "true" {
			conditionName = "$facts_" + calleeName
			smtDef += "\n" + buildDefineFun(conditionName, defParams, "Bool", bodyFacts)
			activeHelperConditions[calleeName] = conditionName
		}
		// The callee's requires, over the same renamed parameters: each call in
		// the contracted body must meet it (codegen_obligations.go).
		reqName, reqUncheckable := calleeRequiresDef(calleeName, meta, params, defParams, &smtDef)
		registerCalleeRequires(calleeName, reqName, reqUncheckable)
		defs = append(defs, CalleeDef{
			Name:                calleeName,
			SMTLib:              smtDef,
			ConditionName:       conditionName,
			RequiresName:        reqName,
			UncheckableRequires: reqUncheckable,
			HOFBoundedDepth:     hofDepth,
		})
		resolvedSoFar[calleeName] = true
	}

	return defs, nil
}

// findFuncBodyInAnyProg looks up a function body by BARE name: first in the
// current program, then in importedPrograms in sorted path order. Returns
// (body, sourceProg) or (nil, nil) if not found. The order is fixed so the
// result never depends on map iteration; Verify refuses any call whose module
// this lookup would not pick (firstMisresolvedCallee, callee_identity.go).
func findFuncBodyInAnyProg(funcName string, prog *core.Program, importedPrograms map[string]*core.Program) (core.CoreExpr, *core.Program) {
	if prog != nil {
		if body := findFuncBody(prog, funcName); body != nil {
			return body, prog
		}
	}
	for _, path := range sortedProgramPaths(importedPrograms) {
		imp := importedPrograms[path]
		if imp == nil {
			continue
		}
		if body := findFuncBody(imp, funcName); body != nil {
			return body, imp
		}
	}
	return nil, nil
}

// buildDefineFun generates an SMT-LIB define-fun declaration.
//
//	(define-fun name ((p1 Sort1) (p2 Sort2)) RetSort body)
func buildDefineFun(name string, params []FunctionParam, returnSort string, bodyExpr string) string {
	var paramParts []string
	for _, p := range params {
		sort, err := MapType(p.Type)
		if err != nil {
			sort = "Int" // fallback
		}
		paramParts = append(paramParts, fmt.Sprintf("(%s %s)", p.Name, sort))
	}
	return fmt.Sprintf("(define-fun %s (%s) %s %s)",
		name, strings.Join(paramParts, " "), returnSort, bodyExpr)
}

// topoSort returns a topological ordering of callee functions: if A calls B,
// B appears before A. A call cycle cannot be ordered (a define-fun cannot refer
// forward), so its members are returned in cyclic, each mapped to the cycle as
// text; the resolver calls them by contract or opaquely, never defines them,
// so their place in the order does not matter.
func topoSort(callees []string, rootFunc string, prog *core.Program, imported map[string]*core.Program) ([]string, map[string]string) {
	// Build adjacency: for each callee, find what it calls
	adj := make(map[string][]string)
	for _, name := range callees {
		body, sourceProg := findFuncBodyInAnyProg(name, prog, imported)
		if body == nil {
			continue
		}
		_, innerBody := unwrapLambda(body)
		deps := collectDirectCalls(innerBody, name, sourceProg, imported)
		adj[name] = deps
	}

	// DFS-based topological sort; a back edge marks the cycle on the stack.
	var order []string
	state := make(map[string]int) // 0=unvisited, 1=visiting, 2=visited
	cyclic := make(map[string]string)
	var stack []string

	var visit func(string)
	visit = func(name string) {
		if state[name] == 2 {
			return
		}
		if state[name] == 1 {
			i := len(stack) - 1
			for i > 0 && stack[i] != name {
				i--
			}
			members := append(append([]string(nil), stack[i:]...), name)
			text := strings.Join(members, " → ")
			for _, m := range stack[i:] {
				if cyclic[m] == "" {
					cyclic[m] = text
				}
			}
			return
		}
		state[name] = 1
		stack = append(stack, name)
		for _, dep := range adj[name] {
			visit(dep)
		}
		state[name] = 2
		stack = stack[:len(stack)-1]
		order = append(order, name)
	}

	for _, name := range callees {
		if state[name] == 0 {
			visit(name)
		}
	}
	return order, cyclic
}

// collectDirectCalls collects only the direct (non-transitive) callee references
// in a function body. Used for building the dependency graph.
func collectDirectCalls(body core.CoreExpr, selfName string, prog *core.Program, imported map[string]*core.Program) []string {
	seen := make(map[string]bool)
	collectDirectCallsInner(body, selfName, prog, imported, seen)
	return sortedKeys(seen)
}

func collectDirectCallsInner(expr core.CoreExpr, selfName string, prog *core.Program, imported map[string]*core.Program, seen map[string]bool) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *core.App:
		if call, ok := matchHOFCall(e); ok { // specialised in place, not a callee
			for _, part := range hofCallParts(call) {
				collectDirectCallsInner(part, selfName, prog, imported, seen)
			}
			return
		}
		if vg, ok := e.Func.(*core.VarGlobal); ok {
			if vg.Ref.Module != "$builtin" && vg.Ref.Name != selfName {
				// Skip stdlib functions with known SMT mappings
				if _, mapped := ResolveStdlibToBuiltin(vg.Ref.Module, vg.Ref.Name); !mapped {
					if body, _ := findFuncBodyInAnyProg(vg.Ref.Name, prog, imported); body != nil {
						seen[vg.Ref.Name] = true
					}
				}
			}
		}
		// Also check plain Var (same-module function references)
		if v, ok := e.Func.(*core.Var); ok {
			if v.Name != selfName {
				if findFuncBody(prog, v.Name) != nil {
					seen[v.Name] = true
				}
			}
		}
		for _, arg := range e.Args {
			collectDirectCallsInner(arg, selfName, prog, imported, seen)
		}
		collectDirectCallsInner(e.Func, selfName, prog, imported, seen)
	case *core.If:
		collectDirectCallsInner(e.Cond, selfName, prog, imported, seen)
		collectDirectCallsInner(e.Then, selfName, prog, imported, seen)
		collectDirectCallsInner(e.Else, selfName, prog, imported, seen)
	case *core.Let:
		collectDirectCallsInner(e.Value, selfName, prog, imported, seen)
		collectDirectCallsInner(e.Body, selfName, prog, imported, seen)
	case *core.LetRec:
		for _, b := range e.Bindings {
			collectDirectCallsInner(b.Value, selfName, prog, imported, seen)
		}
		collectDirectCallsInner(e.Body, selfName, prog, imported, seen)
	case *core.Match:
		collectDirectCallsInner(e.Scrutinee, selfName, prog, imported, seen)
		for _, arm := range e.Arms {
			collectDirectCallsInner(arm.Body, selfName, prog, imported, seen)
			collectDirectCallsInner(arm.Guard, selfName, prog, imported, seen)
		}
	case *core.BinOp:
		collectDirectCallsInner(e.Left, selfName, prog, imported, seen)
		collectDirectCallsInner(e.Right, selfName, prog, imported, seen)
	case *core.UnOp:
		collectDirectCallsInner(e.Operand, selfName, prog, imported, seen)
	case *core.Intrinsic:
		for _, arg := range e.Args {
			collectDirectCallsInner(arg, selfName, prog, imported, seen)
		}
	case *core.DictApp:
		collectDirectCallsInner(e.Dict, selfName, prog, imported, seen)
		for _, arg := range e.Args {
			collectDirectCallsInner(arg, selfName, prog, imported, seen)
		}
	case *core.DictAbs:
		collectDirectCallsInner(e.Body, selfName, prog, imported, seen)
	case *core.Lambda:
		collectDirectCallsInner(e.Body, selfName, prog, imported, seen)
	case *core.Forall:
		collectDirectCallsInner(e.Lo, selfName, prog, imported, seen)
		collectDirectCallsInner(e.Hi, selfName, prog, imported, seen)
		collectDirectCallsInner(e.Body, selfName, prog, imported, seen)
	}
}

// findFuncBody finds a function's body expression in the Core program.
// collectFunctionNames returns every top-level function name bound in a Core
// program (LetRec bindings and Let declarations). Used to distinguish user
// functions from ADT constructors at the encoding leak site.
func collectFunctionNames(prog *core.Program) []string {
	if prog == nil {
		return nil
	}
	var names []string
	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *core.LetRec:
			for _, binding := range d.Bindings {
				names = append(names, binding.Name)
			}
		case *core.Let:
			names = append(names, d.Name)
		}
	}
	return names
}

func findFuncBody(prog *core.Program, funcName string) core.CoreExpr {
	for _, decl := range prog.Decls {
		switch d := decl.(type) {
		case *core.LetRec:
			for _, binding := range d.Bindings {
				if binding.Name == funcName {
					return binding.Value
				}
			}
		case *core.Let:
			if d.Name == funcName {
				return d.Value
			}
		}
	}
	return nil
}

// unwrapLambda peels Lambda nodes from a Core expression,
// returning param names and the inner body.
func unwrapLambda(body core.CoreExpr) ([]string, core.CoreExpr) {
	var params []string
	inner := body
	for {
		lam, ok := inner.(*core.Lambda)
		if !ok {
			break
		}
		params = append(params, lam.Params...)
		inner = lam.Body
	}
	return params, inner
}

// IsSMTEncodableForCallee checks if a function can be used as a callee
// in cross-function verification. Similar to IsSMTEncodable but doesn't
// require contracts (callees don't need their own contracts to be inlined).
func IsSMTEncodableForCallee(funcName string, meta *core.DeclMeta, body core.CoreExpr, decl ...*ast.FuncDecl) (bool, []SMTRejectionReason) {
	var reasons []SMTRejectionReason

	// Must have an SMT-admissible declaration
	if !smtAdmissible(meta, decl...) {
		reasons = append(reasons, SMTRejectionReason{
			Code:    RejectNotPure,
			Message: fmt.Sprintf("Callee %q has effects", funcName),
		})
	}

	// Must be non-recursive to inline. resolveCalleesFromRoots then uses the
	// callee's contract when it has an ensures, and otherwise unrolls a
	// self-recursive callee to a bounded depth (unrollRecursiveCallee).
	if isRecursive(body, funcName) {
		reasons = append(reasons, SMTRejectionReason{
			Code:    RejectRecursive,
			Message: fmt.Sprintf("Callee %q is recursive", funcName),
		})
	}

	// No higher-order functions, except literal-lambda HOF calls, which are
	// specialised inside the callee's own define-fun (resolveCalleesFromRoots).
	if hasHigherOrder(body) && !AllHigherOrderIsInlinable(body) {
		reasons = append(reasons, SMTRejectionReason{
			Code:    RejectHigherOrder,
			Message: fmt.Sprintf("Callee %q uses higher-order functions", funcName),
		})
	}

	// Encodable builtins and shapes only. Name the builtin, as the
	// function-level check does: "uses unencodable types" was the largest
	// skip bucket in the gap map and said nothing about what to change.
	if blocker := firstUnencodableBuiltin(body); blocker != "" {
		reasons = append(reasons, SMTRejectionReason{
			Code:    RejectUnencodable,
			Message: fmt.Sprintf("Callee %q uses an unencodable builtin: %s", funcName, blocker),
		})
	} else if hasUnencodableTypes(body) {
		reasons = append(reasons, SMTRejectionReason{
			Code:    RejectUnencodable,
			Message: fmt.Sprintf("Callee %q uses an array literal, which has no SMT sort", funcName),
		})
	}

	if name := firstUnannotatedParam(decl...); name != "" {
		reasons = append(reasons, unannotatedParamRejection("Callee", funcName, name))
	}

	return len(reasons) == 0, reasons
}

// IsUserDefinedCall checks whether an App expression represents a call to
// a user-defined function (not a builtin or ADT constructor).
// Handles both VarGlobal (cross-module) and plain Var (same-module) references.
func IsUserDefinedCall(app *core.App, prog *core.Program) (string, bool) {
	// Check VarGlobal (cross-module references)
	if vg, ok := app.Func.(*core.VarGlobal); ok {
		if vg.Ref.Module == "$builtin" {
			return "", false
		}
		name := vg.Ref.Name
		if findFuncBody(prog, name) != nil {
			return name, true
		}
		return "", false
	}
	// Check plain Var (same-module references)
	if v, ok := app.Func.(*core.Var); ok {
		if findFuncBody(prog, v.Name) != nil {
			return v.Name, true
		}
	}
	return "", false
}

// defaultCalleeUnrollDepth bounds a self-recursive callee's unrolling, as
// defaultHOFInlineDepth bounds a HOF specialisation.
const defaultCalleeUnrollDepth = defaultHOFInlineDepth

// unrollRecursiveCallee encodes a self-recursive callee as a bounded unrolling
// (name_0 … name_N, name_0 an uninterpreted stub) plus a define-fun `name` over
// name_N, so call sites encode exactly as for any inline callee. Any proof
// through it is bounded: the caller reports HOFBoundedDepth, and a sat is
// inconclusive until witness replay decides it. Facts inside the unrolled
// levels are kept only as each level's obligations ($ob_ define-funs); the
// rest would escape their define-funs, and dropping them only weakens what the
// solver knows.
func unrollRecursiveCallee(name string, params []FunctionParam, body core.CoreExpr, returnSort string) (string, []FunctionParam, error) {
	savedFacts := activeCallFacts
	activeCallFacts = nil
	defer func() { activeCallFacts = savedFacts }()
	unrolled, err := UnrollRecursiveFunction(UnrollConfig{FuncName: name, Params: params, Body: body, ReturnSort: returnSort, Depth: defaultCalleeUnrollDepth})
	if err != nil {
		return "", nil, err
	}
	defParams, _, _ := withArgRenames(params, func() (string, error) { return "", nil })
	args := make([]string, len(defParams))
	for i, p := range defParams {
		args[i] = p.Name
	}
	top := unrolled.TopLevelName
	if len(args) > 0 {
		top = fmt.Sprintf("(%s %s)", top, strings.Join(args, " "))
	}
	decls := append(unrolled.Declarations, buildDefineFun(name, defParams, returnSort, top))
	decls = append(decls, aliasDefinitionObligations(name, unrolled.TopLevelName, defParams)...)
	return strings.Join(decls, "\n"), defParams, nil
}

// calleeRequiresDef appends to *smtDef the Bool define-fun holding the
// callee's requires (M3), returning its name, or why it could not be encoded.
func calleeRequiresDef(name string, meta *core.DeclMeta, params, defParams []FunctionParam, smtDef *string) (string, string) {
	reqs := requiresContracts(meta)
	if len(reqs) == 0 {
		return "", ""
	}
	_, reqBody, err := withArgRenames(params, func() (string, error) { return encodeRequiresBody(reqs) })
	if err != nil {
		return "", err.Error()
	}
	reqName := "$req_" + name
	*smtDef += "\n" + buildDefineFun(reqName, defParams, "Bool", reqBody)
	return reqName, ""
}

// onlyRecursive reports whether recursion is the callee's only rejection.
func onlyRecursive(reasons []SMTRejectionReason) bool {
	return len(reasons) == 1 && reasons[0].Code == RejectRecursive
}

// registerCalleeRequires makes a resolved callee's requires checkable at calls
// encoded after it: later callees' bodies as well as the contracted body.
func registerCalleeRequires(name, reqName, uncheckable string) {
	if reqName != "" {
		activeCalleeRequires[name] = reqName
	}
	if uncheckable != "" {
		activeUncheckableRequires[name] = uncheckable
	}
}
