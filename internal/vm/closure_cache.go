package vm

import "github.com/sunholo-data/ailang/internal/bytecode"

// zeroCapClosure returns this VM's shared closure for an image prototype that
// captures nothing, creating it on first use (M-VM-QUICK-WINS W1).
func (vm *VM) zeroCapClosure(imageIdx int, proto *bytecode.FuncPrototype) bytecode.Value {
	if vm.zeroCapClosures == nil {
		vm.zeroCapClosures = make([]bytecode.Value, len(vm.Image.Prototypes))
	}
	if vm.zeroCapClosures[imageIdx].Tag != bytecode.TagClosure {
		vm.zeroCapClosures[imageIdx] = bytecode.NewClosure(proto, nil)
	}
	return vm.zeroCapClosures[imageIdx]
}
