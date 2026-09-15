package runner

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/bytecode/compiler"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/gen/lower"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runtime"
	"github.com/sunholo-data/ailang/internal/types"
	"github.com/sunholo-data/ailang/internal/vm"
)

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

// CompileBytecodeFromResult lowers an already-typechecked pipeline result and
// compiles it to a BytecodeImage. This is the shared back-half of the bytecode
// compile pipeline used by both `ailang disasm` (which runs the pipeline
// itself) and `ailang run --bytecode` (which reuses the pipeline result that
// the evaluator path already produced, avoiding a double compile).
//
// pkgName is the package label stamped onto the resulting stmt.Program; it
// affects only diagnostics and disassembly headers, not name resolution.
func CompileBytecodeFromResult(res pipeline.Result, pkgName string) (*bytecode.BytecodeImage, error) {
	if res.Artifacts.Core == nil || res.Artifacts.AST == nil {
		return nil, fmt.Errorf("compile from result: missing AST/Core artifacts")
	}
	prog := &stmt.Program{Package: pkgName}

	// M-BYTECODE-MULTIMODULE M1: if the pipeline loaded additional modules
	// (imports from the entry file), lower every reachable module and merge
	// the results into a single stmt.Program. Each module's FuncDecls are
	// tagged with its module path so the compiler can canonicalize funcIdx
	// keys and cross-module GlobalRefs resolve without bridging.
	//
	// Ordering: modules are lowered in sorted path order so the resulting
	// image is deterministic across runs. TypeDecls from every module are
	// deduplicated by name (later modules lose the race — single-source-of
	// -truth per name is the contract).
	seenTypes := map[string]bool{}

	if len(res.Modules) > 0 {
		// Multi-module mode. Each LoadedModule already carries Core, CoreTI,
		// and the surface AST. The entry module is included in res.Modules,
		// so we do NOT additionally lower res.Artifacts.Core here (that would
		// double-register its FuncDecls and fail funcIdx canonicalization).
		modIDs := make([]string, 0, len(res.Modules))
		for id := range res.Modules {
			modIDs = append(modIDs, id)
		}
		sort.Strings(modIDs)

		for _, modID := range modIDs {
			mod := res.Modules[modID]
			if mod == nil || mod.Core == nil || mod.File == nil {
				continue
			}
			// TypeDecls: walk the module's AST and register each unique type
			// declaration. Tag with the module name so cross-module ADT/record
			// lookups (M3) can find the right entry.
			for _, decl := range mod.File.Decls {
				td, ok := decl.(*ast.TypeDecl)
				if !ok {
					continue
				}
				canonicalType := modID + "." + td.Name
				if seenTypes[canonicalType] {
					continue
				}
				seenTypes[canonicalType] = true
				lowered := lower.LowerTypeDecl(td)
				// Preserve the bare name for in-module lookups; M3 will add
				// module-tagged lookups on top of this.
				prog.TypeDecls = append(prog.TypeDecls, lowered)
			}

			// CoreTI is stored as interface{} in LoadedModule to avoid an
			// import cycle; the pipeline always puts a types.CoreTypeInfo
			// there, so the assertion is safe.
			cti, _ := mod.CoreTI.(types.CoreTypeInfo)
			modProg, err := lower.LowerProgram(mod.Core, cti, mod.File, pkgName)
			if err != nil {
				return nil, fmt.Errorf("lower module %s: %w", modID, err)
			}
			// Tag every FuncDecl with its source module so the compiler
			// keys funcIdx by the canonical "module.name" form.
			for i := range modProg.FuncDecls {
				modProg.FuncDecls[i].Module = modID
			}
			prog.FuncDecls = append(prog.FuncDecls, modProg.FuncDecls...)
		}
		return compiler.Compile(prog)
	}

	// Single-file mode: no imported modules, lower the entry Core directly.
	for _, decl := range res.Artifacts.AST.Decls {
		td, ok := decl.(*ast.TypeDecl)
		if !ok || seenTypes[td.Name] {
			continue
		}
		seenTypes[td.Name] = true
		prog.TypeDecls = append(prog.TypeDecls, lower.LowerTypeDecl(td))
	}
	fileProg, err := lower.LowerProgram(res.Artifacts.Core, res.Artifacts.CoreTI, res.Artifacts.AST, pkgName)
	if err != nil {
		return nil, fmt.Errorf("lower: %w", err)
	}
	prog.FuncDecls = append(prog.FuncDecls, fileProg.FuncDecls...)
	return compiler.Compile(prog)
}

// tryRunEntryViaVM compiles the entry module to bytecode (reusing the
// pipeline result that the evaluator path already produced) and runs the
// named entry function on the VM. Functions the compiler couldn't lower are
// dispatched back to the evaluator through bytecodeBridge — that bridge
// requires the module instance to already be evaluated, which is why this
// helper runs *after* rt.LoadAndEvaluate.
//
// Returns (true, nil) on a successful VM run (result already printed).
// Returns (false, err) when the bytecode path could not be used at all
// (compile failure for the entry, missing prototype, unsupported arity).
// Runtime VM errors that occur after dispatch starts are returned as
// (false, err) too — the caller decides strict vs. fallback.
func tryRunEntryViaVM(rt *runtime.ModuleRuntime, inst *runtime.ModuleInstance, params ModuleExecParams, entry string, args []eval.Value) (bool, error) {
	if params.PipelineResult == nil {
		return false, fmt.Errorf("internal: bytecodeMode set but pipelineResult is nil")
	}

	// Recover from lower/compile panics so they degrade to a normal compile
	// error and let the caller fall back to the evaluator in non-strict mode.
	// The lower pass still contains a few `panic(...)` fast-fails for shapes
	// that haven't been bridged yet (e.g. non-tail-position match lowering);
	// treating them as compile errors is the M3 contract.
	var img *bytecode.BytecodeImage
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("compile panic: %v", r)
			}
		}()
		img, err = CompileBytecodeFromResult(*params.PipelineResult, params.Iface.Module)
	}()
	if err != nil {
		return false, fmt.Errorf("compile: %w", err)
	}
	if err := img.Validate(); err != nil {
		return false, fmt.Errorf("validate: %w", err)
	}

	proto := findEntryProto(img, entry)
	if proto == nil {
		return false, fmt.Errorf("entry function %q not found in bytecode image", entry)
	}

	// Convert decoded args from eval.Value to bytecode.Value via the bridge.
	bcArgs := make([]bytecode.Value, 0, len(args))
	for i, a := range args {
		bv, err := evalValueToBytecode(a)
		if err != nil {
			return false, fmt.Errorf("entry arg %d: %w", i, err)
		}
		bcArgs = append(bcArgs, bv)
	}

	// The lower pass synthesizes a Unit parameter for nullary functions so
	// they round-trip through the call ABI. Pad if needed.
	if proto.NumParams == 1 && len(bcArgs) == 0 {
		bcArgs = []bytecode.Value{bytecode.Unit()}
	}
	if int(proto.NumParams) != len(bcArgs) {
		return false, fmt.Errorf("entry %q expects %d args, got %d", entry, proto.NumParams, len(bcArgs))
	}

	if !params.Quiet {
		fmt.Fprintf(os.Stderr, "%s Running %s via bytecode VM\n", green("✓"), params.Filename)
	}

	machine := vm.NewVM(img)
	// Wire the eval bridge in non-strict mode so EvalOnly stubs trap back
	// to the evaluator. In strict mode, leaving Interop nil makes any trap
	// a loud VM error.
	if !params.StrictBytecode {
		machine.Interop = newBytecodeBridge(rt, inst, rt.GetEvaluator())
	}

	// If the entry itself is EvalOnly the VM adds nothing on top of the
	// evaluator — dispatch through runtime.CallEntrypoint directly so we
	// get the normal per-request evaluator fork, global resolver setup and
	// goroutine-evaluator registration. Going through bridge.CallEvalFunc
	// here (which was the original fallback) skips those steps and
	// subtly breaks effects that depend on resolver/fork state — e.g.
	// Process subprocess stdout ordering.
	if proto.EvalOnly {
		if params.StrictBytecode {
			return false, fmt.Errorf("entry %q is evaluator-only (%s); strict mode disables bridge dispatch", entry, proto.EvalReason)
		}
		result, err := runtime.CallEntrypoint(rt, inst, entry, args)
		if err != nil {
			return false, fmt.Errorf("eval-only entry %q: %w", entry, err)
		}
		if !params.NoPrint && params.Print && result != nil && result.Type() != "unit" {
			fmt.Println(result.String())
		}
		return true, nil
	}

	result, err := machine.Run(proto, bcArgs)
	if err != nil {
		return false, fmt.Errorf("vm: %w", err)
	}
	printVMResult(result, params)
	return true, nil
}

// printVMResult mirrors the evaluator path's print policy for VM-side
// results: skip Unit, honor --no-print, and render the return value in
// evaluator-compatible spelling. We route the VM value through the bridge
// back to an eval.Value so that String() matches the tree-walking path
// byte-for-byte — bytecode.Value.String() is debug-oriented (it prints
// strings with Go %q quoting and would break parity tests).
func printVMResult(v bytecode.Value, params ModuleExecParams) {
	if params.NoPrint {
		return
	}
	if v.Tag == bytecode.TagUnit {
		return
	}
	if !params.Print {
		return
	}
	ev, err := bytecodeValueToEval(v)
	if err != nil || ev == nil {
		// Fall back to the bytecode value's own formatting for shapes
		// the bridge doesn't know how to convert (currently none on
		// the success path — all Tier-1 shapes round-trip).
		fmt.Println(v.String())
		return
	}
	if ev.Type() == "unit" {
		return
	}
	fmt.Println(ev.String())
}
