package compiler

import "fmt"

// regAlloc is a bump-style register allocator with a free list for temps.
//
// Allocation strategy:
//   - Pinned registers (parameters, named locals) are allocated from the bottom
//     and never freed during a function's lifetime.
//   - Temporary registers (intermediate expression results) are allocated from
//     the same address space but recorded on a free list when released.
//   - The high-water mark is what gets written to FuncPrototype.NumRegs.
//
// This is intentionally simple. Phase 2C makes no attempt to minimize register
// pressure beyond reusing temps released in the same expression. The corpus
// is small enough that the 256-register ceiling is not a constraint.
type regAlloc struct {
	next uint16  // next register to hand out (uint16 to detect 256-reg overflow)
	high uint16  // high-water mark
	free []uint8 // freed temp registers, LIFO
}

func newRegAlloc() *regAlloc {
	return &regAlloc{}
}

// allocPinned allocates a register that will not be reclaimed. Used for
// parameters and named locals.
func (r *regAlloc) allocPinned() (uint8, error) {
	return r.bump()
}

// allocTemp allocates a temporary register. Prefer reusing a freed slot.
func (r *regAlloc) allocTemp() (uint8, error) {
	if n := len(r.free); n > 0 {
		reg := r.free[n-1]
		r.free = r.free[:n-1]
		return reg, nil
	}
	return r.bump()
}

// freeTemp returns a previously-allocated temp to the free list. Pinned
// registers should not be passed here.
func (r *regAlloc) freeTemp(reg uint8) {
	r.free = append(r.free, reg)
}

// allocContig allocates n contiguous fresh registers and returns the base.
// Used for call frames, where the callee + args must occupy a contiguous
// register range. The free list is intentionally bypassed.
func (r *regAlloc) allocContig(n int) (uint8, error) {
	if n <= 0 {
		return 0, fmt.Errorf("regAlloc: allocContig n must be > 0")
	}
	if r.next+uint16(n) > 256 {
		return 0, fmt.Errorf("register allocator: contiguous block of %d would exceed 256", n)
	}
	base := uint8(r.next)
	r.next += uint16(n)
	if r.next > r.high {
		r.high = r.next
	}
	return base, nil
}

// freeContig releases a contiguous block back to the free list, individually.
// (Order is LIFO so subsequent allocTemp calls reuse high regs first.)
func (r *regAlloc) freeContig(base uint8, n int) {
	for i := n - 1; i >= 0; i-- {
		r.free = append(r.free, base+uint8(i))
	}
}

// bump advances the next pointer and returns the freshly allocated register.
func (r *regAlloc) bump() (uint8, error) {
	if r.next >= 256 {
		return 0, fmt.Errorf("register allocator: function exceeds 256-register limit")
	}
	reg := uint8(r.next)
	r.next++
	if r.next > r.high {
		r.high = r.next
	}
	return reg, nil
}

// highWater returns the largest register count this allocator has needed.
// This is what gets written to FuncPrototype.NumRegs.
func (r *regAlloc) highWater() uint8 {
	if r.high == 0 {
		// Every function needs at least one register for the return value.
		return 1
	}
	if r.high > 255 {
		return 255
	}
	return uint8(r.high)
}

// scopeStack maps named local variables to registers, supporting nested scopes
// (let bindings, match bindings). Phase 2C M1 only uses a flat scope; nested
// scopes arrive in M2 (control flow) and M5 (match).
type scopeStack struct {
	frames []map[string]uint8
}

func newScopeStack() *scopeStack {
	return &scopeStack{
		frames: []map[string]uint8{{}},
	}
}

// bind associates name with register reg in the current (innermost) scope.
// Shadowing is allowed: an inner scope may rebind an outer name.
func (s *scopeStack) bind(name string, reg uint8) {
	s.frames[len(s.frames)-1][name] = reg
}

// lookup walks scopes from innermost to outermost, returning the register
// bound to name, or (0, false) if not found.
func (s *scopeStack) lookup(name string) (uint8, bool) {
	for i := len(s.frames) - 1; i >= 0; i-- {
		if r, ok := s.frames[i][name]; ok {
			return r, true
		}
	}
	return 0, false
}

// push opens a new scope. Used by control-flow constructs that introduce
// short-lived bindings.
func (s *scopeStack) push() {
	s.frames = append(s.frames, map[string]uint8{})
}

// pop closes the innermost scope.
func (s *scopeStack) pop() {
	if len(s.frames) <= 1 {
		panic("scopeStack: pop on root scope")
	}
	s.frames = s.frames[:len(s.frames)-1]
}
