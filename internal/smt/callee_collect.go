package smt

import "github.com/sunholo-data/ailang/internal/core"

// collectCalleeCalls walks the body and collects names of user-defined
// functions that are called (VarGlobal references where module != "$builtin").
// xmodDepth tracks how many cross-module hops we've taken; stops at crossModuleInlineDepth.
func collectCalleeCalls(body core.CoreExpr, selfName string, prog *core.Program, imported map[string]*core.Program, xmodDepth int) []string {
	seen := make(map[string]bool)
	collectCalleeCallsInner(body, selfName, calleeWalk{prog, prog, imported}, seen, xmodDepth)
	var result []string
	for name := range seen {
		result = append(result, name)
	}
	return result
}

// collectCalleeCallsFromRoots is collectCalleeCalls over several root expressions,
// de-duplicated. Order is unspecified (topoSort orders the result).
func collectCalleeCallsFromRoots(roots []core.CoreExpr, selfName string, prog *core.Program, imported map[string]*core.Program) []string {
	seen := make(map[string]bool)
	for _, root := range roots {
		collectCalleeCallsInner(root, selfName, calleeWalk{prog, prog, imported}, seen, 0)
	}
	// Sorted, not map order: the callee order decides which cycle a skip
	// names and the order define-funs are emitted (A1 determinism).
	return sortedKeys(seen)
}

func collectCalleeCallsInner(expr core.CoreExpr, selfName string, w calleeWalk, seen map[string]bool, xmodDepth int) {
	prog, imported := w.root, w.imported
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *core.App:
		// A literal-lambda map/filter/foldl/any/flatMap is specialised in place
		// (InlineHOFCalls), never resolved as a callee body: walk its arguments
		// (the lambda body's own calls still count) but not its head.
		if call, ok := matchHOFCall(e); ok {
			for _, part := range hofCallParts(call) {
				collectCalleeCallsInner(part, selfName, w, seen, xmodDepth)
			}
			return
		}
		// Check if function is a user-defined call via VarGlobal
		if vg, ok := e.Func.(*core.VarGlobal); ok {
			if vg.Ref.Module != "$builtin" && vg.Ref.Name != selfName {
				// Skip stdlib functions with known SMT mappings (handled as builtins by encoder)
				if _, mapped := ResolveStdlibToBuiltin(vg.Ref.Module, vg.Ref.Name); !mapped {
					calleeBody, source := findFuncBodyInAnyProg(vg.Ref.Name, prog, imported)
					if calleeBody != nil && !seen[vg.Ref.Name] {
						seen[vg.Ref.Name] = true
						// Recurse into callee body; increment xmodDepth for cross-module hops
						nextDepth := xmodDepth
						if findFuncBody(prog, vg.Ref.Name) == nil {
							nextDepth++ // crossing a module boundary
						}
						if nextDepth <= crossModuleInlineDepth {
							_, innerBody := unwrapLambda(calleeBody)
							collectCalleeCallsInner(innerBody, selfName, calleeWalk{prog, source, imported}, seen, nextDepth)
						}
					}
				}
			}
		}
		// A plain Var call is to a function of the module whose body this is:
		// std/json.get's findInList is in std/json, not in the caller's
		// program, where it was looked up before and never collected.
		if v, ok := e.Func.(*core.Var); ok {
			if v.Name != selfName {
				if calleeBody := findFuncBody(w.current, v.Name); calleeBody != nil {
					if !seen[v.Name] {
						seen[v.Name] = true
						_, innerBody := unwrapLambda(calleeBody)
						collectCalleeCallsInner(innerBody, selfName, w, seen, xmodDepth)
					}
				}
			}
		}
		// Also check for curried calls: App(App(VarGlobal(...), args1), args2)
		if innerApp, ok := e.Func.(*core.App); ok {
			collectCalleeCallsInner(innerApp, selfName, w, seen, xmodDepth)
		}
		// Walk arguments
		for _, arg := range e.Args {
			collectCalleeCallsInner(arg, selfName, w, seen, xmodDepth)
		}
		collectCalleeCallsInner(e.Func, selfName, w, seen, xmodDepth)
	case *core.If:
		collectCalleeCallsInner(e.Cond, selfName, w, seen, xmodDepth)
		collectCalleeCallsInner(e.Then, selfName, w, seen, xmodDepth)
		collectCalleeCallsInner(e.Else, selfName, w, seen, xmodDepth)
	case *core.Let:
		collectCalleeCallsInner(e.Value, selfName, w, seen, xmodDepth)
		collectCalleeCallsInner(e.Body, selfName, w, seen, xmodDepth)
	case *core.LetRec:
		for _, b := range e.Bindings {
			collectCalleeCallsInner(b.Value, selfName, w, seen, xmodDepth)
		}
		collectCalleeCallsInner(e.Body, selfName, w, seen, xmodDepth)
	case *core.Match:
		collectCalleeCallsInner(e.Scrutinee, selfName, w, seen, xmodDepth)
		for _, arm := range e.Arms {
			collectCalleeCallsInner(arm.Body, selfName, w, seen, xmodDepth)
			collectCalleeCallsInner(arm.Guard, selfName, w, seen, xmodDepth)
		}
	case *core.BinOp:
		collectCalleeCallsInner(e.Left, selfName, w, seen, xmodDepth)
		collectCalleeCallsInner(e.Right, selfName, w, seen, xmodDepth)
	case *core.UnOp:
		collectCalleeCallsInner(e.Operand, selfName, w, seen, xmodDepth)
	case *core.Intrinsic:
		for _, arg := range e.Args {
			collectCalleeCallsInner(arg, selfName, w, seen, xmodDepth)
		}
	case *core.DictApp:
		collectCalleeCallsInner(e.Dict, selfName, w, seen, xmodDepth)
		for _, arg := range e.Args {
			collectCalleeCallsInner(arg, selfName, w, seen, xmodDepth)
		}
	case *core.DictAbs:
		collectCalleeCallsInner(e.Body, selfName, w, seen, xmodDepth)
	case *core.Lambda:
		collectCalleeCallsInner(e.Body, selfName, w, seen, xmodDepth)
	case *core.Forall:
		collectCalleeCallsInner(e.Lo, selfName, w, seen, xmodDepth)
		collectCalleeCallsInner(e.Hi, selfName, w, seen, xmodDepth)
		collectCalleeCallsInner(e.Body, selfName, w, seen, xmodDepth)
	}
}

// calleeWalk is where collectCalleeCallsInner is: the root program and the
// imported ones (bare-name lookups of global calls; firstMisresolvedCallee
// refuses any call they would bind to the wrong module), and the program
// whose body is being walked (its plain Var calls).
type calleeWalk struct {
	root, current *core.Program
	imported      map[string]*core.Program
}
