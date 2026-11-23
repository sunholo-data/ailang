package smt

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/core"
)

// Callee identity (M-VERIFY-SOUNDNESS-COVERAGE M1, S7).
//
// The encoder keys callees by their BARE name everywhere: the define-fun symbol,
// SurfaceParams/SurfaceReturnSorts, the callee-sort gate's AST lookup, and the
// body lookup findFuncBodyInAnyProg (root program first, then imported programs
// in sorted path order). A call names its callee by (module, name). When two
// loaded modules define the same name, the bare lookup can return the OTHER
// module's body: `import a/good (clamp0 as g)` plus `import a/bad (clamp0)`
// verified a contract that a/bad.clamp0 violates (VERIFIED in 5 of 6 runs while
// the lookup still iterated a Go map).
//
// Rather than qualify every symbol, the verifier refuses the cases where the
// bare-name pipeline would pick a different program than the call means.
// firstMisresolvedCallee finds them. Everywhere else the bare name is
// unambiguous, and lookups are deterministic.

// sortedProgramPaths returns the keys of imported in sorted order, so every
// bare-name lookup over imported programs is deterministic.
func sortedProgramPaths(imported map[string]*core.Program) []string {
	paths := make([]string, 0, len(imported))
	for path := range imported {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

// misresolvedCallee names a call whose bare-name lookup would bind a different
// definition than the call refers to.
type misresolvedCallee struct {
	Name     string // bare function name
	Wanted   string // module the call refers to
	Resolved string // module the bare-name lookup would use ("" = the checked module)
}

// firstMisresolvedCallee walks every call reachable from roots — through callee
// bodies, across modules — and returns the first one whose bare name resolves,
// under findFuncBodyInAnyProg, to a different program than the one the call
// refers to. rootModule is the checked module's path (VarGlobal refs to the
// checked module's own functions carry it). selfName is the contracted function.
// The walk visits callees in a deterministic order.
func firstMisresolvedCallee(roots []core.CoreExpr, selfName, rootModule string, root *core.Program, imported map[string]*core.Program) *misresolvedCallee {
	pathOf := make(map[*core.Program]string, len(imported)+1)
	pathOf[root] = rootModule
	for path, p := range imported {
		pathOf[p] = path
	}
	w := &identityWalker{
		selfName:   selfName,
		rootModule: rootModule,
		root:       root,
		imported:   imported,
		pathOf:     pathOf,
		visited:    make(map[string]bool),
	}
	for _, r := range roots {
		if m := w.walk(r, root); m != nil {
			return m
		}
	}
	return nil
}

type identityWalker struct {
	selfName   string
	rootModule string
	root       *core.Program
	imported   map[string]*core.Program
	pathOf     map[*core.Program]string
	visited    map[string]bool
}

// programForModule returns the loaded program for a module path, or nil.
func (w *identityWalker) programForModule(module string) *core.Program {
	if module == w.rootModule {
		return w.root
	}
	return w.imported[module]
}

// check compares the program a call means with the program the bare-name
// pipeline would use, then descends into the meant body.
func (w *identityWalker) check(name string, wanted *core.Program, wantedModule string) *misresolvedCallee {
	if wanted != nil && findFuncBody(wanted, name) == nil {
		wanted = nil // the referenced module does not define it as a function (constructor, value)
	}
	_, got := findFuncBodyInAnyProg(name, w.root, w.imported)
	// Calls the bare pipeline would not inline at all are not its concern.
	if got == nil {
		return nil
	}
	// A reference to the contracted function's own name from another module is
	// treated as recursion by the bare-name pipeline (collectCalleeCalls skips
	// selfName); flag it unless it really is the checked module's function.
	if name == w.selfName && wanted != w.root {
		return &misresolvedCallee{Name: name, Wanted: wantedModule, Resolved: w.rootModule}
	}
	if wanted != got {
		return &misresolvedCallee{Name: name, Wanted: wantedModule, Resolved: w.pathOf[got]}
	}
	key := wantedModule + "\x00" + name
	if w.visited[key] {
		return nil
	}
	w.visited[key] = true
	if name == w.selfName {
		return nil
	}
	_, inner := unwrapLambda(findFuncBody(wanted, name))
	return w.walk(inner, wanted)
}

// checkCallHead checks the head of a call: a global function reference, or a
// plain Var naming a top-level function of the program whose body we are in.
func (w *identityWalker) checkCallHead(head core.CoreExpr, current *core.Program) *misresolvedCallee {
	switch f := head.(type) {
	case *core.VarGlobal:
		if f.Ref.Module == "$builtin" {
			return nil
		}
		if _, mapped := ResolveStdlibToBuiltin(f.Ref.Module, f.Ref.Name); mapped {
			return nil // encoded natively, never looked up as a body
		}
		return w.check(f.Ref.Name, w.programForModule(f.Ref.Module), f.Ref.Module)
	case *core.Var:
		if findFuncBody(current, f.Name) == nil {
			return nil
		}
		return w.check(f.Name, current, w.pathOf[current])
	}
	return nil
}

// walk visits expr, whose plain Var references belong to program current.
func (w *identityWalker) walk(expr core.CoreExpr, current *core.Program) *misresolvedCallee {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *core.App:
		// A specialised HOF's head is never looked up; its (eta-expanded)
		// function argument is a call head inside the lambda body.
		if call, isHOF := matchHOFCall(e); isHOF {
			return w.walkAll(hofCallParts(call), current)
		}
		// Only a name in CALL position is looked up as a callee body by the
		// bare-name pipeline (collectCalleeCallsInner inspects App.Func), so only
		// those can be misbound. A bare Var elsewhere is a value — often a
		// parameter that happens to share a top-level function's name.
		if m := w.checkCallHead(e.Func, current); m != nil {
			return m
		}
		if m := w.walk(e.Func, current); m != nil {
			return m
		}
		return w.walkAll(e.Args, current)
	case *core.Lambda:
		return w.walk(e.Body, current)
	case *core.Let:
		if m := w.walk(e.Value, current); m != nil {
			return m
		}
		return w.walk(e.Body, current)
	case *core.LetRec:
		for _, b := range e.Bindings {
			if m := w.walk(b.Value, current); m != nil {
				return m
			}
		}
		return w.walk(e.Body, current)
	case *core.If:
		return w.walkAll([]core.CoreExpr{e.Cond, e.Then, e.Else}, current)
	case *core.Match:
		if m := w.walk(e.Scrutinee, current); m != nil {
			return m
		}
		for _, arm := range e.Arms {
			if m := w.walkAll([]core.CoreExpr{arm.Guard, arm.Body}, current); m != nil {
				return m
			}
		}
	case *core.BinOp:
		return w.walkAll([]core.CoreExpr{e.Left, e.Right}, current)
	case *core.UnOp:
		return w.walk(e.Operand, current)
	case *core.Record:
		return w.walkFields(e.Fields, current)
	case *core.RecordAccess:
		return w.walk(e.Record, current)
	case *core.RecordUpdate:
		if m := w.walk(e.Base, current); m != nil {
			return m
		}
		return w.walkFields(e.Updates, current)
	case *core.List:
		return w.walkAll(e.Elements, current)
	case *core.Array:
		return w.walkAll(e.Elements, current)
	case *core.Tuple:
		return w.walkAll(e.Elements, current)
	case *core.Intrinsic:
		return w.walkAll(e.Args, current)
	case *core.DictApp:
		if m := w.walk(e.Dict, current); m != nil {
			return m
		}
		return w.walkAll(e.Args, current)
	case *core.DictAbs:
		return w.walk(e.Body, current)
	case *core.Forall:
		return w.walkAll([]core.CoreExpr{e.Lo, e.Hi, e.Body}, current)
	}
	return nil
}

func (w *identityWalker) walkAll(exprs []core.CoreExpr, current *core.Program) *misresolvedCallee {
	for _, x := range exprs {
		if m := w.walk(x, current); m != nil {
			return m
		}
	}
	return nil
}

// walkFields visits a field map in sorted key order (deterministic first hit).
func (w *identityWalker) walkFields(fields map[string]core.CoreExpr, current *core.Program) *misresolvedCallee {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if m := w.walk(fields[k], current); m != nil {
			return m
		}
	}
	return nil
}

// contractExprs returns the non-nil predicate expressions of contracts.
func contractExprs(contracts []*core.Contract) []core.CoreExpr {
	out := make([]core.CoreExpr, 0, len(contracts))
	for _, c := range contracts {
		if c != nil && c.Expr != nil {
			out = append(out, c.Expr)
		}
	}
	return out
}
