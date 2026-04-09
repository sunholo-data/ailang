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

// allocContig allocates n contiguous registers and returns the base.
// Used for call frames, where the callee + args must occupy a contiguous
// register range.
//
// Strategy: first scan the free list for n adjacent slots. If found, remove
// them and return the base. Otherwise, bump from r.next as before.
func (r *regAlloc) allocContig(n int) (uint8, error) {
	if n <= 0 {
		return 0, fmt.Errorf("regAlloc: allocContig n must be > 0")
	}

	// Try to find n contiguous registers in the free list.
	if base, ok := r.findContigInFreeList(n); ok {
		return base, nil
	}

	// Fall back to bumping from r.next.
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

// findContigInFreeList searches for n adjacent free registers. If found,
// removes them from the free list and returns (base, true).
func (r *regAlloc) findContigInFreeList(n int) (uint8, bool) {
	if len(r.free) < n {
		return 0, false
	}

	// Build a sorted copy and look for a run of n.
	sorted := make([]uint8, len(r.free))
	copy(sorted, r.free)
	sortUint8(sorted)

	for i := 0; i <= len(sorted)-n; i++ {
		if sorted[i+n-1]-sorted[i] == uint8(n-1) {
			// Found a contiguous run starting at sorted[i].
			base := sorted[i]
			// Remove these n registers from the free list.
			removeSet := make(map[uint8]bool, n)
			for j := 0; j < n; j++ {
				removeSet[base+uint8(j)] = true
			}
			filtered := r.free[:0]
			for _, reg := range r.free {
				if !removeSet[reg] {
					filtered = append(filtered, reg)
				}
			}
			r.free = filtered
			return base, true
		}
	}
	return 0, false
}

// sortUint8 sorts a small slice of uint8 values using insertion sort
// (faster than sort.Slice for the small sizes typical of free lists).
func sortUint8(s []uint8) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
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
//
// M-BYTECODE-REGALLOC-FIX adds scope-aware register recycling: each scope
// frame tracks which registers were pinned in it, and pop() releases them
// back to the allocator's free list. This prevents switch cases with many
// bindings from permanently consuming registers.
type scopeStack struct {
	frames []scopeFrame
	alloc  *regAlloc // back-pointer for recycling on pop
}

// scopeFrame holds the name→register bindings AND the list of registers
// pinned (allocated) in this scope. When the scope is popped, pinnedRegs
// are returned to the allocator's free list.
type scopeFrame struct {
	names      map[string]uint8
	pinnedRegs []uint8
}

func newScopeStack(alloc *regAlloc) *scopeStack {
	return &scopeStack{
		frames: []scopeFrame{{names: map[string]uint8{}}},
		alloc:  alloc,
	}
}

// bind associates name with register reg in the current (innermost) scope.
// Shadowing is allowed: an inner scope may rebind an outer name.
func (s *scopeStack) bind(name string, reg uint8) {
	s.frames[len(s.frames)-1].names[name] = reg
}

// bindScoped associates name with a freshly allocated pinned register and
// records it for release when this scope is popped. Used for switch case
// bindings and other short-lived local variables in nested scopes.
func (s *scopeStack) bindScoped(name string, reg uint8) {
	f := &s.frames[len(s.frames)-1]
	f.names[name] = reg
	f.pinnedRegs = append(f.pinnedRegs, reg)
}

// lookup walks scopes from innermost to outermost, returning the register
// bound to name, or (0, false) if not found.
func (s *scopeStack) lookup(name string) (uint8, bool) {
	for i := len(s.frames) - 1; i >= 0; i-- {
		if r, ok := s.frames[i].names[name]; ok {
			return r, true
		}
	}
	return 0, false
}

// push opens a new scope. Used by control-flow constructs that introduce
// short-lived bindings.
func (s *scopeStack) push() {
	s.frames = append(s.frames, scopeFrame{names: map[string]uint8{}})
}

// pop closes the innermost scope and releases any scoped registers back to
// the allocator's free list for reuse.
func (s *scopeStack) pop() {
	if len(s.frames) <= 1 {
		panic("scopeStack: pop on root scope")
	}
	top := s.frames[len(s.frames)-1]
	// Release scoped registers in reverse order (LIFO) for deterministic reuse.
	for i := len(top.pinnedRegs) - 1; i >= 0; i-- {
		s.alloc.freeTemp(top.pinnedRegs[i])
	}
	s.frames = s.frames[:len(s.frames)-1]
}
