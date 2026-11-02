package types

// Effect-row union with operand resolution (#1958 re-land, F-A1 and F-A2).
//
// Effect rows of sibling sub-expressions are combined bottom-up while their
// operands can still be bare tails: inferApp leaves a callee's row unsubstituted,
// so `{ println(x); println(x) }` meets as `{...ε9} ∪ {...ε11}`. combineEffects
// cannot merge two distinct tails and closes the row, so the lambda published
// `{}` with no labels. That made an IO lambda look pure to a callback slot
// whenever its body made more than one call.
//
// unionEffects resolves such operands first, against the equality constraints
// added since the enclosing lambda began. It follows the same rules as #386's
// application-local solve in inferApp: bindings of variables owned by an
// enclosing scope are dropped, and an operand is replaced only when it resolves
// to a closed row. One exception is fixpoint absorption: an operand whose tail
// is the effect row of a let-rec binding being inferred is its own recursive
// call, and contributes only its labels (rho = L ∪ rho has least solution
// rho = L). A callback parameter's row is not a self row and is never absorbed.

// unionFrame is the resolution scope of one lambda body.
type unionFrame struct {
	checkpoint int             // first constraint added inside the lambda
	outer      map[string]bool // type/row vars owned by the enclosing scope
}

// pushUnionFrame opens a lambda's resolution scope; call the result to close it.
func (ctx *InferenceContext) pushUnionFrame() func() {
	outer := make(map[string]bool)
	for v := range ctx.env.FreeTypeVars() {
		outer[v] = true
	}
	for v := range ctx.env.FreeRowVars() {
		outer[v] = true
	}
	for v := range ctx.env.FreeEffectRowVars() {
		outer[v] = true
	}
	ctx.unionFrames = append(ctx.unionFrames, unionFrame{checkpoint: len(ctx.constraints), outer: outer})
	depth := len(ctx.unionFrames) - 1
	return func() { ctx.unionFrames = ctx.unionFrames[:depth] }
}

// pushSelfTypes records the binding types of a let-rec group being inferred.
func (ctx *InferenceContext) pushSelfTypes(types []Type) func() {
	depth := len(ctx.selfTypes)
	ctx.selfTypes = append(ctx.selfTypes, types...)
	return func() { ctx.selfTypes = ctx.selfTypes[:depth] }
}

// unionEffectList combines effect rows left to right with unionEffects.
func (ctx *InferenceContext) unionEffectList(effects []*Row) *Row {
	result := EmptyEffectRow()
	for _, e := range effects {
		result = ctx.unionEffects(result, e)
	}
	return result
}

// unionEffects is combineEffects with distinct-tail operands resolved first.
func (ctx *InferenceContext) unionEffects(e1, e2 *Row) *Row {
	if !hasDistinctEffectTails(e1, e2) || len(ctx.unionFrames) == 0 {
		return combineEffects(e1, e2)
	}
	frame := ctx.unionFrames[len(ctx.unionFrames)-1]
	sub, err := ctx.solveLocalEqualities(frame.checkpoint)
	if err != nil {
		// The enclosing SolveConstraints reports the error with its full path.
		return combineEffects(e1, e2)
	}
	local := make(Substitution, len(sub))
	for v, t := range sub {
		if !frame.outer[v] {
			local[v] = t
		}
	}
	return combineEffects(ctx.resolveUnionOperand(sub, local, e1), ctx.resolveUnionOperand(sub, local, e2))
}

func hasDistinctEffectTails(e1, e2 *Row) bool {
	if e1 == nil || e2 == nil {
		return false
	}
	t1, t2 := effectTailOnly(e1.Tail), effectTailOnly(e2.Tail)
	return t1 != nil && t2 != nil && t1.Name != t2.Name
}

// resolveUnionOperand applies fixpoint absorption against the full local
// solution, then closeIfResolved against the solution without outer bindings.
func (ctx *InferenceContext) resolveUnionOperand(full, local Substitution, row *Row) *Row {
	if row == nil || row.Tail == nil {
		return row
	}
	resolved := applySubToEffectRow(full, row)
	if resolved != nil && resolved.Tail != nil && ctx.isSelfEffectTail(full, resolved.Tail.Name) {
		absorbed := *resolved
		absorbed.Tail = nil
		return &absorbed
	}
	return closeIfResolved(row, applySubToEffectRow(local, row))
}

// isSelfEffectTail reports whether tail is the (resolved) effect-row tail of a
// let-rec binding currently being inferred.
func (ctx *InferenceContext) isSelfEffectTail(sub Substitution, tail string) bool {
	for _, self := range ctx.selfTypes {
		fn, ok := ApplySubstitution(sub, self).(*TFunc2)
		if !ok || fn.EffectRow == nil {
			continue
		}
		if row := applySubToEffectRow(sub, fn.EffectRow); row != nil && row.Tail != nil && row.Tail.Name == tail {
			return true
		}
	}
	return false
}
