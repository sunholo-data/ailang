package runner

import (
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/eval"
)

// The evaluator finds the contract checker by a type assertion on its effect
// context (eval.ContractChecker), and a context that does not satisfy the
// interface silently disables contract checking. Pin the production context
// to the interface at compile time, so a signature change on either side is a
// build error instead of `--verify-contracts` quietly checking nothing.
var _ eval.ContractChecker = (*effects.EffContext)(nil)

// ContractsOnVMRefusal returns the CTR004 refusal for a run that asks for
// runtime contract checking on the bytecode VM, or "" when the run may go
// ahead (M-V1-VERIFICATION-HONESTY M2a = M-RUNTIME-CONTRACT-PARITY M1, #1956).
//
// The VM never consults the effect context's contract checker, so
// `--bytecode --verify-contracts` (and `--strict-bytecode`, and either under
// `--batch`) used to execute with no contract checked and exit 0. The
// combination is refused before compiling, rather than silently falling back
// to the evaluator: the caller asked for the VM, and a fallback would hide the
// gap the VM work must close. Lifted when the VM enforces contracts.
func ContractsOnVMRefusal(opts Options) string {
	if !opts.VerifyContracts || !opts.BytecodeMode {
		return ""
	}
	flag := "--bytecode"
	if opts.StrictBytecode {
		flag = "--strict-bytecode"
	}
	return errors.CTR004 + ": --verify-contracts is not enforced on the bytecode VM yet (runtime contracts " +
		"run only on the evaluator). Drop " + flag + " to check contracts, or drop --verify-contracts " +
		"to run on the VM. Tracking: #1956."
}
