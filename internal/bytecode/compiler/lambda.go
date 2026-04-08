package compiler

import (
	"fmt"

	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/gen/stmt"
)

// compileLambda hoists a stmt.Lambda into a child FuncPrototype, registers it
// in the image, threads its index through NestedProtos, and emits OpClosure +
// pseudo-OpMove instructions to materialize a closure value at runtime.
//
// Capture model (matches Phase 2B VM):
//   - The lambda's prototype has NumParams = len(params) and
//     NumCaptures = len(free vars).
//   - In the callee's frame, params occupy r[0..NumParams-1] and captures
//     occupy r[NumParams..NumParams+NumCaptures-1] — the VM places them there
//     on CALL/TAIL_CALL.
//   - From the OUTER function: emit OpClosure r_dst, #nested_proto_idx,
//     followed by N pseudo-OpMove instructions whose B field is the source
//     register in the *outer* function for capture i.
func (fc *funcCompiler) compileLambda(lam stmt.Lambda) (uint8, error) {
	// 1. Free-variable analysis: which outer names does this lambda use?
	freeVars := freeVarsLambda(lam)
	// Filter to those that are actually bound in our outer scope (locals).
	// Names that are not locals must be top-level functions (handled via
	// closure-of-name in the body) or unbound (which is a compile error in
	// the inner pass).
	var captures []string
	captureSrc := make(map[string]uint8)
	for _, name := range freeVars {
		if r, ok := fc.locals.lookup(name); ok {
			captures = append(captures, name)
			captureSrc[name] = r
		}
	}
	if len(captures) > 255 {
		return 0, fmt.Errorf("lambda has %d captures, exceeds 255", len(captures))
	}

	// 2. Build & register the inner prototype.
	innerProto := &bytecode.FuncPrototype{
		Name:        fmt.Sprintf("%s$lambda%d", fc.proto.Name, len(fc.img.Prototypes)),
		NumParams:   uint8(len(lam.Params)),
		NumCaptures: uint8(len(captures)),
	}
	innerImageIdx := fc.img.AddPrototype(innerProto)

	// 3. Compile the inner body in a fresh funcCompiler.
	inner := newFuncCompiler(fc.img, innerProto, fc.funcIdx)
	inner.recordTypes = fc.recordTypes
	inner.adtTypes = fc.adtTypes
	// Pin parameters to r[0..NumParams-1].
	for _, p := range lam.Params {
		r, err := inner.regs.allocPinned()
		if err != nil {
			return 0, err
		}
		inner.locals.bind(p.Name, r)
	}
	// Pin captures to r[NumParams..NumParams+NumCaptures-1]. The VM writes
	// them there on call entry.
	for _, name := range captures {
		r, err := inner.regs.allocPinned()
		if err != nil {
			return 0, err
		}
		inner.locals.bind(name, r)
	}

	// Lower body + return.
	for _, s := range lam.Body {
		if err := inner.compileStmt(s); err != nil {
			return 0, fmt.Errorf("lambda body: %w", err)
		}
	}
	if lam.Return != nil {
		if err := inner.compileReturnExpr(lam.Return); err != nil {
			return 0, err
		}
	} else {
		r, err := inner.regs.allocTemp()
		if err != nil {
			return 0, err
		}
		inner.emit(bytecode.EncodeABC(bytecode.OpLoadNil, r, 0, 0))
		inner.emit(bytecode.EncodeABC(bytecode.OpReturn, r, 0, 0))
	}
	innerProto.NumRegs = inner.regs.highWater()

	// 4. Outer-side: emit OpClosure + N pseudo-MOVEs.
	dst, err := fc.regs.allocTemp()
	if err != nil {
		return 0, err
	}
	nestedIdx, err := fc.lookupNestedProto(innerImageIdx)
	if err != nil {
		return 0, err
	}
	fc.emit(bytecode.EncodeABx(bytecode.OpClosure, dst, nestedIdx))
	for _, name := range captures {
		// Pseudo-MOVE: A is unused, B is the source register in the outer frame.
		fc.emit(bytecode.EncodeABC(bytecode.OpMove, 0, captureSrc[name], 0))
	}
	return dst, nil
}

// --- Free variable analysis ------------------------------------------------

// freeVarsLambda returns the deterministic ordered list of free variables in
// a lambda. A free variable is any VarRef whose name is not bound by the
// lambda's parameters or by an inner let/match. Order is "first encountered
// in a left-to-right body walk", which is stable for a given AST.
func freeVarsLambda(lam stmt.Lambda) []string {
	bound := make(map[string]bool, len(lam.Params))
	for _, p := range lam.Params {
		bound[p.Name] = true
	}
	v := &freeVarVisitor{bound: bound, seen: map[string]bool{}}
	for _, s := range lam.Body {
		v.visitStmt(s)
	}
	if lam.Return != nil {
		v.visitExpr(lam.Return)
	}
	return v.order
}

type freeVarVisitor struct {
	bound map[string]bool
	seen  map[string]bool
	order []string
}

func (v *freeVarVisitor) recordFree(name string) {
	if v.bound[name] || v.seen[name] {
		return
	}
	v.seen[name] = true
	v.order = append(v.order, name)
}

func (v *freeVarVisitor) withBound(name string, body func()) {
	if v.bound[name] {
		body()
		return
	}
	v.bound[name] = true
	body()
	delete(v.bound, name)
}

func (v *freeVarVisitor) visitStmt(s stmt.Stmt) {
	switch s := s.(type) {
	case stmt.VarDecl:
		v.visitExpr(s.Value)
		v.bound[s.Name] = true
	case stmt.AssignStmt:
		v.visitExpr(s.Value)
	case stmt.ReturnStmt:
		v.visitExpr(s.Value)
	case stmt.ExprStmt:
		v.visitExpr(s.Value)
	case stmt.IfStmt:
		v.visitExpr(s.Cond)
		for _, ts := range s.Then {
			v.visitStmt(ts)
		}
		for _, es := range s.Else {
			v.visitStmt(es)
		}
	case stmt.SwitchStmt:
		v.visitExpr(s.Scrutinee)
		for _, c := range s.Cases {
			// Bindings are scoped to the case body — track and untrack them.
			added := make([]string, 0, len(c.Bindings))
			for _, b := range c.Bindings {
				if !v.bound[b.Name] {
					v.bound[b.Name] = true
					added = append(added, b.Name)
				}
			}
			for _, bs := range c.Body {
				v.visitStmt(bs)
			}
			for _, n := range added {
				delete(v.bound, n)
			}
		}
		for _, ds := range s.Default {
			v.visitStmt(ds)
		}
	}
}

func (v *freeVarVisitor) visitExpr(e stmt.Expr) {
	if e == nil {
		return
	}
	switch e := e.(type) {
	case stmt.VarRef:
		v.recordFree(e.Name)
	case stmt.BinOp:
		v.visitExpr(e.Left)
		v.visitExpr(e.Right)
	case stmt.UnOp:
		v.visitExpr(e.Operand)
	case stmt.Call:
		v.visitExpr(e.Func)
		for _, a := range e.Args {
			v.visitExpr(a)
		}
	case stmt.FieldAccess:
		v.visitExpr(e.Record)
	case stmt.RecordLit:
		for _, f := range e.Fields {
			v.visitExpr(f.Value)
		}
	case stmt.RecordUpdate:
		v.visitExpr(e.Base)
		for _, f := range e.Fields {
			v.visitExpr(f.Value)
		}
	case stmt.ListLit:
		for _, el := range e.Elems {
			v.visitExpr(el)
		}
	case stmt.ArrayLit:
		for _, el := range e.Elems {
			v.visitExpr(el)
		}
	case stmt.TupleLit:
		for _, el := range e.Elems {
			v.visitExpr(el)
		}
	case stmt.Cons:
		v.visitExpr(e.Head)
		v.visitExpr(e.Tail)
	case stmt.ADTConstructor:
		for _, a := range e.Args {
			v.visitExpr(a)
		}
	case stmt.Lambda:
		// Inner lambda: anything it references that ISN'T its own params and
		// ISN'T already in our bound set is free *to us*.
		inner := &freeVarVisitor{bound: copyMap(v.bound), seen: map[string]bool{}}
		for _, p := range e.Params {
			inner.bound[p.Name] = true
		}
		for _, s := range e.Body {
			inner.visitStmt(s)
		}
		if e.Return != nil {
			inner.visitExpr(e.Return)
		}
		for _, name := range inner.order {
			v.recordFree(name)
		}
	case stmt.TypeAssert:
		v.visitExpr(e.Value)
	case stmt.IfExpr:
		v.visitExpr(e.Cond)
		v.visitExpr(e.Then)
		v.visitExpr(e.Else)
	case stmt.BuiltinCall:
		for _, a := range e.Args {
			v.visitExpr(a)
		}
	}
}

func copyMap(m map[string]bool) map[string]bool {
	out := make(map[string]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
