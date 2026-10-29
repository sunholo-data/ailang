package vm

import (
	"math/rand"
	"testing"

	"github.com/sunholo-data/ailang/internal/bytecode"
)

// TestResetForTailCallMatchesZeroAllThenWrite: resetting a frame for a tail
// call must leave every register, up to the slab's capacity, exactly as the
// old sequence did: stash the arguments, zero the whole new slab, write the
// arguments, then the captures (M-VM-QUICK-WINS W4). Random slab sizes cover
// shrinking, growing within capacity, reallocating, and arguments that
// overlap their destination.
func TestResetForTailCallMatchesZeroAllThenWrite(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for iter := 0; iter < 5000; iter++ {
		oldLen := 1 + rng.Intn(20)
		oldCap := oldLen + rng.Intn(6)
		argc := rng.Intn(5)
		if argc > oldLen-1 {
			argc = oldLen - 1
		}
		argStart := 0
		if oldLen-argc > 0 {
			argStart = rng.Intn(oldLen - argc + 1)
		}
		ncaps := rng.Intn(3)
		numRegs := argc + ncaps + rng.Intn(12)
		if numRegs == 0 {
			numRegs = 1
		}

		backing := make([]bytecode.Value, oldCap)
		for i := range backing {
			backing[i] = bytecode.NewInt(int64(1000*iter + i + 1))
		}
		caps := make([]bytecode.Value, ncaps)
		for i := range caps {
			caps[i] = bytecode.NewInt(int64(-1 - i))
		}
		proto := &bytecode.FuncPrototype{Name: "callee", NumParams: uint8(argc), NumRegs: uint8(numRegs)}

		// Reference: the pre-W4 sequence on a copy of the same slab.
		refBacking := make([]bytecode.Value, oldCap) // exact capacity, unlike append
		copy(refBacking, backing)
		ref := &Frame{Regs: refBacking[:oldLen]}
		args := append([]bytecode.Value(nil), ref.Regs[argStart:argStart+argc]...)
		if numRegs > cap(ref.Regs) {
			ref.Regs = make([]bytecode.Value, numRegs)
		} else {
			ref.Regs = ref.Regs[:numRegs]
			for i := range ref.Regs {
				ref.Regs[i] = bytecode.Value{}
			}
		}
		copy(ref.Regs, args)
		for i, c := range caps {
			ref.Regs[argc+i] = c
		}

		got := &Frame{Regs: backing[:oldLen]}
		got.resetForTailCall(proto, argStart, argc, caps)

		if len(got.Regs) != len(ref.Regs) || cap(got.Regs) != cap(ref.Regs) {
			t.Fatalf("iter %d: len/cap %d/%d, want %d/%d", iter, len(got.Regs), cap(got.Regs), len(ref.Regs), cap(ref.Regs))
		}
		g, r := got.Regs[:cap(got.Regs)], ref.Regs[:cap(ref.Regs)]
		for i := range r {
			if g[i] != r[i] {
				t.Fatalf("iter %d (oldLen %d cap %d argStart %d argc %d caps %d numRegs %d): reg %d = %v, want %v",
					iter, oldLen, oldCap, argStart, argc, ncaps, numRegs, i, g[i], r[i])
			}
		}
		if got.Proto != proto || got.IP != 0 {
			t.Fatalf("iter %d: proto/IP not reset", iter)
		}
	}
}
