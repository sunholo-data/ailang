package compiler

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// Local `letrec` (M-VM-COVERAGE G1).
//
// A local letrec lowers to consecutive stmt.VarDecls that share a non-empty
// RecGroup. Compiling each value as an ordinary lambda is wrong: the value is
// compiled before its own name is a local, so a self reference is not
// captured and falls through to a top-level function of the same name. That
// gave a silent wrong answer (G1b: shadow.ail printed 1000 instead of 10).
//
// The scheme here needs no new opcode. Every member of a group gets its own
// prototype, and all members share ONE capture list: the union of the
// members' free variables, minus the group names. Inside a member body, a
// reference to any member that no local shadows is rebuilt on the spot,
// with OpClosure on that member's prototype and pseudo-MOVEs from this
// frame's own capture registers. Since the capture lists are identical, the
// rebuilt closure is the same value the enclosing scope holds. A self call
// in tail position is therefore an OpTailCall on a closure, which reuses the
// frame like any other tail call.

// compileStmts compiles a statement list. It is the only loop over
// compileStmt, because a letrec group spans several consecutive VarDecls and
// must be compiled as a unit.
func (fc *funcCompiler) compileStmts(stmts []stmt.Stmt) error {
	for i := 0; i < len(stmts); i++ {
		d, ok := stmts[i].(stmt.VarDecl)
		if !ok || len(d.RecGroup) == 0 {
			if err := fc.compileStmt(stmts[i]); err != nil {
				return err
			}
			continue
		}
		// Collect the run of VarDecls that belong to this group.
		group := []stmt.VarDecl{d}
		for i+1 < len(stmts) {
			next, ok := stmts[i+1].(stmt.VarDecl)
			if !ok || !sameRecGroup(next.RecGroup, d.RecGroup) {
				break
			}
			group = append(group, next)
			i++
		}
		if len(group) != len(d.RecGroup) {
			return fmt.Errorf("compiler: letrec group %v has %d consecutive bindings, want %d", d.RecGroup, len(group), len(d.RecGroup))
		}
		if err := fc.compileRecGroup(group); err != nil {
			return err
		}
	}
	return nil
}

func sameRecGroup(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// compileRecGroup compiles one local letrec group and binds each member name
// to a closure in the current scope.
func (fc *funcCompiler) compileRecGroup(group []stmt.VarDecl) error {
	prevLine := fc.currentLine
	defer func() { fc.currentLine = prevLine }()
	if group[0].Line > 0 {
		fc.currentLine = group[0].Line
	}

	names := make(map[string]bool, len(group))
	lams := make([]stmt.Lambda, len(group))
	for i, d := range group {
		lam, ok := d.Value.(stmt.Lambda)
		if !ok {
			// A non-function letrec binding has no closure to rebuild. Refuse
			// rather than guess; the enclosing function falls back cleanly.
			return fmt.Errorf("compiler: letrec binding %q is not a lambda (%T)", d.Name, d.Value)
		}
		lams[i] = lam
		names[d.Name] = true
	}

	// Union of the members' free variables, minus the group names, in a
	// deterministic first-seen order.
	var free []string
	seen := map[string]bool{}
	for _, lam := range lams {
		for _, n := range freeVarsLambda(lam) {
			if !names[n] && !seen[n] {
				seen[n] = true
				free = append(free, n)
			}
		}
	}
	captures, captureSrc, captureTemps, err := fc.resolveCaptures(free)
	if err != nil {
		return err
	}

	// Register every member prototype first, so each body can rebuild any
	// member's closure (mutual recursion).
	members := make(map[string]int, len(group))
	protos := make([]*bytecode.FuncPrototype, len(group))
	for i, d := range group {
		p := &bytecode.FuncPrototype{
			Name:        fmt.Sprintf("%s$letrec$%s$%d", fc.proto.Name, d.Name, len(fc.img.Prototypes)),
			NumParams:   uint8(len(lams[i].Params)),
			NumCaptures: uint8(len(captures)),
		}
		protos[i] = p
		members[d.Name] = fc.img.AddPrototype(p)
	}

	for i := range group {
		if err := fc.compileRecMember(protos[i], lams[i], captures, members); err != nil {
			return fmt.Errorf("letrec %q: %w", group[i].Name, err)
		}
	}

	// Materialise each member's closure in the enclosing scope. Members do
	// not read each other's closures at run time (they rebuild them), so the
	// order does not matter.
	dsts := make([]uint8, len(group))
	for i, d := range group {
		dst, err := fc.regs.allocTemp()
		if err != nil {
			return err
		}
		if err := fc.emitClosure(dst, members[d.Name], captures, captureSrc); err != nil {
			return err
		}
		dsts[i] = dst
	}
	fc.freeTemps(captureTemps)
	for i, d := range group {
		fc.locals.bindScoped(d.Name, dsts[i])
	}
	return nil
}

// compileRecMember compiles one letrec member body into proto.
func (fc *funcCompiler) compileRecMember(proto *bytecode.FuncPrototype, lam stmt.Lambda, captures []string, members map[string]int) error {
	inner := fc.child(proto)
	inner.proto.File = fc.proto.File
	inner.recGroup = members
	for _, p := range lam.Params {
		r, err := inner.regs.allocPinned()
		if err != nil {
			return err
		}
		inner.locals.bind(p.Name, r)
	}
	for _, name := range captures {
		r, err := inner.regs.allocPinned()
		if err != nil {
			return err
		}
		inner.locals.bind(name, r)
		// Rebuilding a member closure reads the capture REGISTERS, not the
		// names: a body-local `let` may shadow a captured name.
		inner.recCapRegs = append(inner.recCapRegs, r)
	}
	body, ret := inlineBodyCtorTemps(lam.Body, lam.Return)
	if err := inner.compileStmts(body); err != nil {
		return err
	}
	if ret != nil {
		if err := inner.compileReturnExpr(ret); err != nil {
			return err
		}
	} else {
		r, err := inner.regs.allocTemp()
		if err != nil {
			return err
		}
		inner.emit(bytecode.EncodeABC(bytecode.OpLoadNil, r, 0, 0))
		inner.emit(bytecode.EncodeABC(bytecode.OpReturn, r, 0, 0))
	}
	proto.NumRegs = inner.regs.highWater()
	return nil
}

// recMember reports whether name refers to a member of the letrec group this
// function belongs to. A local of the same name shadows the member, so the
// caller must check locals first.
func (fc *funcCompiler) recMember(name string) (int, bool) {
	if fc.recGroup == nil {
		return 0, false
	}
	idx, ok := fc.recGroup[name]
	return idx, ok
}

// emitRecClosure writes the closure of group member imageIdx into dst,
// rebuilt from this frame's capture registers.
func (fc *funcCompiler) emitRecClosure(dst uint8, imageIdx int) error {
	nestedIdx, err := fc.lookupNestedProto(imageIdx)
	if err != nil {
		return err
	}
	fc.emit(bytecode.EncodeABx(bytecode.OpClosure, dst, nestedIdx))
	for _, r := range fc.recCapRegs {
		fc.emit(bytecode.EncodeABC(bytecode.OpMove, 0, r, 0))
	}
	return nil
}

// emitClosure emits OpClosure for imageIdx plus one pseudo-MOVE per capture.
func (fc *funcCompiler) emitClosure(dst uint8, imageIdx int, captures []string, src map[string]uint8) error {
	nestedIdx, err := fc.lookupNestedProto(imageIdx)
	if err != nil {
		return err
	}
	fc.emit(bytecode.EncodeABx(bytecode.OpClosure, dst, nestedIdx))
	for _, name := range captures {
		fc.emit(bytecode.EncodeABC(bytecode.OpMove, 0, src[name], 0))
	}
	return nil
}

// resolveCaptures picks, from a list of free variable names, those the
// current scope can supply as captures, and the register each comes from.
// A local supplies its own register. A letrec group member that no local
// shadows is materialised into a temp first; the temps are returned so the
// caller frees them after emitting the closure. Any other name is left to
// the inner body to resolve (a top-level function, or a compile error).
func (fc *funcCompiler) resolveCaptures(free []string) ([]string, map[string]uint8, []uint8, error) {
	var captures []string
	src := make(map[string]uint8)
	var temps []uint8
	for _, name := range free {
		if r, ok := fc.locals.lookup(name); ok {
			captures = append(captures, name)
			src[name] = r
			continue
		}
		if idx, ok := fc.recMember(name); ok {
			r, err := fc.regs.allocTemp()
			if err != nil {
				return nil, nil, nil, err
			}
			if err := fc.emitRecClosure(r, idx); err != nil {
				return nil, nil, nil, err
			}
			captures = append(captures, name)
			src[name] = r
			temps = append(temps, r)
		}
	}
	if len(captures) > 255 {
		return nil, nil, nil, fmt.Errorf("closure has %d captures, exceeds 255", len(captures))
	}
	return captures, src, temps, nil
}

func (fc *funcCompiler) freeTemps(regs []uint8) {
	for _, r := range regs {
		fc.regs.freeTemp(r)
	}
}
