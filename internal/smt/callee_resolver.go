package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
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
	Name        string
	SMTLib      string // The (define-fun ...) or contract (declare-const + assert) declaration
	ADTDecl     string // Any ADT declarations needed (may be empty)
	IsContract  bool   // True when using contract-as-spec fallback (not define-fun)
	ResultConst string // Non-empty when IsContract: the constant to substitute for call sites
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
	if prog == nil {
		return nil, nil
	}

	var imported map[string]*core.Program
	if len(importedPrograms) > 0 {
		imported = importedPrograms[0]
	}

	// Find all user-defined function calls in the body (same-module + cross-module)
	callees := collectCalleeCalls(body, funcName, prog, imported, 0)
	if len(callees) == 0 {
		return nil, nil
	}

	// Build topological order with cycle detection
	order, err := topoSort(callees, funcName, prog, imported)
	if err != nil {
		return nil, err
	}

	// Encode each callee as a define-fun
	var defs []CalleeDef
	for _, calleeName := range order {
		// Look up body in current program first, then imported programs
		calleeBody, calleeProg := findFuncBodyInAnyProg(calleeName, prog, imported)
		if calleeBody == nil || calleeProg == nil {
			continue
		}

		// Unwrap lambda to get inner body
		_, innerBody := unwrapLambda(calleeBody)

		// Check if callee is SMT-encodable
		meta := calleeProg.Meta[calleeName]
		if meta == nil {
			continue
		}

		// Get params and return sort
		params := surfaceParams[calleeName]
		returnSort := surfaceReturnSorts[calleeName]
		if returnSort == "" {
			returnSort = "Int"
		}

		encodable, _ := IsSMTEncodableForCallee(calleeName, meta, calleeBody)
		if !encodable {
			// Contract-as-spec fallback: emit a declare-const for the callee result
			// and assert only the ensures clauses as axioms.
			// Requires clauses are skipped because they reference param names that
			// cannot be substituted without call-site arg tracking (deferred to M4).
			var ensuresOnly []*core.Contract
			for _, c := range meta.Contracts {
				if c.Kind == core.EnsuresKind {
					ensuresOnly = append(ensuresOnly, c)
				}
			}
			if len(ensuresOnly) > 0 {
				spec := ContractSpec{
					FuncName:   calleeName,
					Params:     params,
					ReturnSort: returnSort,
					Ensures:    ensuresOnly,
					// Requires omitted — cannot bind without call-site args
				}
				contractDecl, err := EncodeCalleeByContract(spec, nil, 0)
				if err == nil {
					defs = append(defs, CalleeDef{
						Name:        calleeName,
						SMTLib:      contractDecl.SMTLib,
						IsContract:  true,
						ResultConst: contractDecl.ResultConst,
					})
				}
			}
			continue
		}

		// Encode the callee body
		bodyExpr, err := EncodeExpr(innerBody)
		if err != nil {
			// Skip callees we can't encode rather than failing the whole verification
			continue
		}

		// Build define-fun
		smtDef := buildDefineFun(calleeName, params, returnSort, bodyExpr)
		defs = append(defs, CalleeDef{
			Name:   calleeName,
			SMTLib: smtDef,
		})
	}

	return defs, nil
}

// findFuncBodyInAnyProg looks up a function body first in the current program,
// then in importedPrograms. Returns (body, sourceProg) or (nil, nil) if not found.
func findFuncBodyInAnyProg(funcName string, prog *core.Program, importedPrograms map[string]*core.Program) (core.CoreExpr, *core.Program) {
	if body := findFuncBody(prog, funcName); body != nil {
		return body, prog
	}
	for _, imp := range importedPrograms {
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

// collectCalleeCalls walks the body and collects names of user-defined
// functions that are called (VarGlobal references where module != "$builtin").
// xmodDepth tracks how many cross-module hops we've taken; stops at crossModuleInlineDepth.
func collectCalleeCalls(body core.CoreExpr, selfName string, prog *core.Program, imported map[string]*core.Program, xmodDepth int) []string {
	seen := make(map[string]bool)
	collectCalleeCallsInner(body, selfName, prog, imported, seen, xmodDepth)
	var result []string
	for name := range seen {
		result = append(result, name)
	}
	return result
}

func collectCalleeCallsInner(expr core.CoreExpr, selfName string, prog *core.Program, imported map[string]*core.Program, seen map[string]bool, xmodDepth int) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *core.App:
		// Check if function is a user-defined call via VarGlobal
		if vg, ok := e.Func.(*core.VarGlobal); ok {
			if vg.Ref.Module != "$builtin" && vg.Ref.Name != selfName {
				// Skip stdlib functions with known SMT mappings (handled as builtins by encoder)
				if _, mapped := ResolveStdlibToBuiltin(vg.Ref.Module, vg.Ref.Name); !mapped {
					calleeBody, _ := findFuncBodyInAnyProg(vg.Ref.Name, prog, imported)
					if calleeBody != nil && !seen[vg.Ref.Name] {
						seen[vg.Ref.Name] = true
						// Recurse into callee body; increment xmodDepth for cross-module hops
						nextDepth := xmodDepth
						if findFuncBody(prog, vg.Ref.Name) == nil {
							nextDepth++ // crossing a module boundary
						}
						if nextDepth <= crossModuleInlineDepth {
							_, innerBody := unwrapLambda(calleeBody)
							collectCalleeCallsInner(innerBody, selfName, prog, imported, seen, nextDepth)
						}
					}
				}
			}
		}
		// Check if function is a user-defined call via plain Var (same-module)
		if v, ok := e.Func.(*core.Var); ok {
			if v.Name != selfName {
				if calleeBody := findFuncBody(prog, v.Name); calleeBody != nil {
					if !seen[v.Name] {
						seen[v.Name] = true
						_, innerBody := unwrapLambda(calleeBody)
						collectCalleeCallsInner(innerBody, selfName, prog, imported, seen, xmodDepth)
					}
				}
			}
		}
		// Also check for curried calls: App(App(VarGlobal(...), args1), args2)
		if innerApp, ok := e.Func.(*core.App); ok {
			collectCalleeCallsInner(innerApp, selfName, prog, imported, seen, xmodDepth)
		}
		// Walk arguments
		for _, arg := range e.Args {
			collectCalleeCallsInner(arg, selfName, prog, imported, seen, xmodDepth)
		}
		collectCalleeCallsInner(e.Func, selfName, prog, imported, seen, xmodDepth)
	case *core.If:
		collectCalleeCallsInner(e.Cond, selfName, prog, imported, seen, xmodDepth)
		collectCalleeCallsInner(e.Then, selfName, prog, imported, seen, xmodDepth)
		collectCalleeCallsInner(e.Else, selfName, prog, imported, seen, xmodDepth)
	case *core.Let:
		collectCalleeCallsInner(e.Value, selfName, prog, imported, seen, xmodDepth)
		collectCalleeCallsInner(e.Body, selfName, prog, imported, seen, xmodDepth)
	case *core.LetRec:
		for _, b := range e.Bindings {
			collectCalleeCallsInner(b.Value, selfName, prog, imported, seen, xmodDepth)
		}
		collectCalleeCallsInner(e.Body, selfName, prog, imported, seen, xmodDepth)
	case *core.Match:
		collectCalleeCallsInner(e.Scrutinee, selfName, prog, imported, seen, xmodDepth)
		for _, arm := range e.Arms {
			collectCalleeCallsInner(arm.Body, selfName, prog, imported, seen, xmodDepth)
		}
	case *core.BinOp:
		collectCalleeCallsInner(e.Left, selfName, prog, imported, seen, xmodDepth)
		collectCalleeCallsInner(e.Right, selfName, prog, imported, seen, xmodDepth)
	case *core.UnOp:
		collectCalleeCallsInner(e.Operand, selfName, prog, imported, seen, xmodDepth)
	case *core.Intrinsic:
		for _, arg := range e.Args {
			collectCalleeCallsInner(arg, selfName, prog, imported, seen, xmodDepth)
		}
	case *core.DictApp:
		collectCalleeCallsInner(e.Dict, selfName, prog, imported, seen, xmodDepth)
		for _, arg := range e.Args {
			collectCalleeCallsInner(arg, selfName, prog, imported, seen, xmodDepth)
		}
	case *core.DictAbs:
		collectCalleeCallsInner(e.Body, selfName, prog, imported, seen, xmodDepth)
	case *core.Lambda:
		collectCalleeCallsInner(e.Body, selfName, prog, imported, seen, xmodDepth)
	case *core.Forall:
		collectCalleeCallsInner(e.Lo, selfName, prog, imported, seen, xmodDepth)
		collectCalleeCallsInner(e.Hi, selfName, prog, imported, seen, xmodDepth)
		collectCalleeCallsInner(e.Body, selfName, prog, imported, seen, xmodDepth)
	}
}

// topoSort returns a topological ordering of callee functions.
// If A calls B, B appears before A in the result.
// Returns error if circular dependencies are detected.
func topoSort(callees []string, rootFunc string, prog *core.Program, imported map[string]*core.Program) ([]string, error) {
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

	// DFS-based topological sort with cycle detection
	var order []string
	state := make(map[string]int) // 0=unvisited, 1=visiting, 2=visited
	var cycle []string

	var visit func(string) error
	visit = func(name string) error {
		if state[name] == 2 {
			return nil
		}
		if state[name] == 1 {
			cycle = append(cycle, name)
			return fmt.Errorf("circular function call detected: %s", strings.Join(cycle, " → "))
		}
		state[name] = 1
		cycle = append(cycle, name)
		for _, dep := range adj[name] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[name] = 2
		cycle = cycle[:len(cycle)-1]
		order = append(order, name)
		return nil
	}

	for _, name := range callees {
		if state[name] == 0 {
			if err := visit(name); err != nil {
				return nil, err
			}
		}
	}

	return order, nil
}

// collectDirectCalls collects only the direct (non-transitive) callee references
// in a function body. Used for building the dependency graph.
func collectDirectCalls(body core.CoreExpr, selfName string, prog *core.Program, imported map[string]*core.Program) []string {
	seen := make(map[string]bool)
	collectDirectCallsInner(body, selfName, prog, imported, seen)
	var result []string
	for name := range seen {
		result = append(result, name)
	}
	return result
}

func collectDirectCallsInner(expr core.CoreExpr, selfName string, prog *core.Program, imported map[string]*core.Program, seen map[string]bool) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *core.App:
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
func IsSMTEncodableForCallee(funcName string, meta *core.DeclMeta, body core.CoreExpr) (bool, []SMTRejectionReason) {
	var reasons []SMTRejectionReason

	// Must be pure
	if !isPure(meta) {
		reasons = append(reasons, SMTRejectionReason{
			Code:    RejectNotPure,
			Message: fmt.Sprintf("Callee %q has effects", funcName),
		})
	}

	// Must be non-recursive
	if isRecursive(body, funcName) {
		reasons = append(reasons, SMTRejectionReason{
			Code:    RejectRecursive,
			Message: fmt.Sprintf("Callee %q is recursive", funcName),
		})
	}

	// No higher-order functions
	if hasHigherOrder(body) {
		reasons = append(reasons, SMTRejectionReason{
			Code:    RejectHigherOrder,
			Message: fmt.Sprintf("Callee %q uses higher-order functions", funcName),
		})
	}

	// Encodable types only
	if hasUnencodableTypes(body) {
		reasons = append(reasons, SMTRejectionReason{
			Code:    RejectUnencodable,
			Message: fmt.Sprintf("Callee %q uses unencodable types", funcName),
		})
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
