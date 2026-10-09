package vm

import (
	"github.com/sunholo-data/ailang/internal/bytecode"
	"testing"
)

// The production default must allow evaluator-legal deep non-tail recursion.
func TestVMDefaultStackLimit(t *testing.T) {
	machine := NewVM(bytecode.NewImage())
	if machine.MaxStack != 10000 {
		t.Fatalf("default MaxStack = %d, want 10000", machine.MaxStack)
	}
}
