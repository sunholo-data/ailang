package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/vm"
)

// tryRunBytecode attempts to compile filename through the bytecode pipeline
// and execute the named entry function on the VM. It is the M2 wiring for
// `ailang run --bytecode`.
//
// Returns (true, nil) on successful execution. Returns (false, err) when
// the bytecode path cannot be used (compile failure, missing entry, entry
// requires non-Unit args). The caller decides whether to fall back to the
// evaluator or fail hard based on --strict-bytecode.
//
// Phase 2D scope: only nullary entry points (e.g. `func main() -> int`).
// Anything that needs program arguments or argsJSON falls back. M3 (eval
// fallback bridge) and M6 (full parity) widen this.
func tryRunBytecode(filename, entry string, relaxModules, quiet bool) (bool, error) {
	img, err := compileBytecodeFromFile(filename, relaxModules)
	if err != nil {
		return false, err
	}

	proto := findEntryProto(img, entry)
	if proto == nil {
		return false, fmt.Errorf("entry function %q not found in bytecode image", entry)
	}

	// The lower pass synthesizes a Unit parameter for nullary functions so
	// they round-trip through the runtime call ABI. Pad accordingly.
	var args []bytecode.Value
	if proto.NumParams == 1 {
		args = []bytecode.Value{bytecode.Unit()}
	} else if proto.NumParams != 0 {
		return false, fmt.Errorf(
			"entry function %q expects %d args; --bytecode only supports nullary entries in Phase 2D",
			entry, proto.NumParams)
	}

	if !quiet {
		fmt.Fprintf(os.Stderr, "%s Running %s via bytecode VM\n", green("✓"), filename)
	}

	v := vm.NewVM(img)
	result, err := v.Run(proto, args)
	if err != nil {
		// Compile succeeded but execution failed — this IS a real failure,
		// not a "fall back" case. Return the error so --strict-bytecode
		// reports it; non-strict will still fall back via the caller.
		return false, fmt.Errorf("vm: %w", err)
	}

	// Print the return value (matching the evaluator's `--print` default).
	fmt.Println(formatBytecodeResult(result))
	return true, nil
}

// findEntryProto resolves an entry name (e.g. "main") to a prototype in the
// image. The lower pass prefixes function names with the module package, so
// we accept several spellings.
func findEntryProto(img *bytecode.BytecodeImage, name string) *bytecode.FuncPrototype {
	for _, p := range img.Prototypes {
		if p.Name == name {
			return p
		}
		// Tolerate "<pkg>.<name>" and "<pkg>_<name>" qualifications.
		if strings.HasSuffix(p.Name, "."+name) || strings.HasSuffix(p.Name, "_"+name) {
			return p
		}
	}
	return nil
}

// formatBytecodeResult is a minimal printer for VM return values. Matches
// the evaluator's REPL-style default formatting closely enough that golden
// tests across both backends can use the same expected outputs.
func formatBytecodeResult(v bytecode.Value) string {
	return v.String()
}
