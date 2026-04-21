// Package compiler lowers Statement IR (internal/gen/stmt) into bytecode
// images runnable by the register VM (internal/vm).
//
// Per M-BYTECODE-VM §10 Phase 2C, this package:
//   - reads from internal/gen/stmt (read-only)
//   - writes to internal/bytecode (mutates the image)
//   - does NOT import internal/vm, internal/eval, internal/core, internal/lower
//
// The compiler is intentionally dumb: a single linear pass per function with
// a bump-style register allocator. No SSA, no graph coloring, no liveness
// intervals, no spilling. The corpus targeted by Phase 2C (12 golden tests)
// is well within the 256-register limit.
//
// Milestone scope:
//
//	M1 (this file)  : skeleton, literals, arithmetic, locals
//	M2              : control flow, comparisons, short-circuit
//	M3              : calls, recursion, TCO
//	M4              : lambdas, closures, free-variable analysis
//	M5              : collections, ADTs, pattern matching
//	M6              : builtins + golden parity gate
package compiler

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// Compile lowers a Statement IR program into a bytecode image. The first
// exported function in prog.FuncDecls is set as the entry point if any
// function is exported; otherwise the first function is used.
//
// On success, the returned image has been validated via BytecodeImage.Validate().
// A validation failure is treated as a compiler bug and surfaces as an error.
func Compile(prog *stmt.Program) (*bytecode.BytecodeImage, error) {
	if prog == nil {
		return nil, fmt.Errorf("compiler: nil program")
	}

	img := bytecode.NewImage()

	// Phase 0: register type declarations. ADT tags get source-order ordinals.
	// Record types record their alphabetically-sorted field list.
	recordTypes := make(map[string]recordTypeInfo)
	adtTypes := make(map[string]adtTypeInfo)
	for _, td := range prog.TypeDecls {
		switch k := td.Kind.(type) {
		case stmt.ADTDecl:
			info := adtTypeInfo{
				tagOrdinal: make(map[string]int, len(k.Variants)),
				tagFields:  make(map[string]int, len(k.Variants)),
			}
			for i, v := range k.Variants {
				info.tagOrdinal[v.Tag] = i
				info.tagFields[v.Tag] = len(v.Fields)
			}
			adtTypes[td.Name] = info
		case stmt.RecordDecl:
			names := make([]string, len(k.Fields))
			for i, f := range k.Fields {
				names[i] = f.Name
			}
			sortedNames := append([]string(nil), names...)
			sort.Strings(sortedNames)
			recordTypes[td.Name] = recordTypeInfo{sortedFields: sortedNames}
		}
	}

	// Phase 1: register every function so calls (M3) can resolve forward refs.
	//
	// funcIdx is keyed by the function's *canonical* name:
	//   - bare `Name` for single-module programs (fd.Module == "")
	//   - `Module + "." + Name` for multi-module programs
	//
	// This keying is what allows M-BYTECODE-MULTIMODULE to lower every
	// reachable module's functions into a single bytecode image without
	// bare-name collisions (e.g. two modules that both define `helper`).
	funcIdx := make(map[string]int, len(prog.FuncDecls))
	for i := range prog.FuncDecls {
		fd := &prog.FuncDecls[i]
		canonical := canonicalFuncName(fd.Module, fd.Name)
		proto := &bytecode.FuncPrototype{
			Name:      canonical,
			NumParams: uint8(len(fd.Params)),
		}
		idx := img.AddPrototype(proto)
		funcIdx[canonical] = idx
	}

	// Phase 2: compile each function body. Per M-BYTECODE-2D M3, a per-function
	// compile failure does NOT abort the whole image — instead the prototype is
	// marked EvalOnly so the VM dispatches it through the evaluator at call
	// time. Other functions can keep referencing it via OpClosure/NestedProtos.
	//
	// M-BYTECODE-BATCH extends this: if the lower pass caught a panic while
	// producing this FuncDecl (LowerError set), skip compilation entirely and
	// tag the proto EvalOnly with the lower-pass reason, so the bridge
	// dispatches the call to the evaluator transparently.
	for i := range prog.FuncDecls {
		fd := &prog.FuncDecls[i]
		proto := img.Prototypes[funcIdx[canonicalFuncName(fd.Module, fd.Name)]]

		if fd.LowerError != "" {
			proto.EvalOnly = true
			proto.EvalReason = fd.LowerError
			continue
		}

		// Snapshot the prototype-table length so we can roll back any partial
		// child lambda prototypes appended by a failed compile. (Constants are
		// dedup-shared and harmless to leave behind as orphans.)
		protoCheckpoint := len(img.Prototypes)

		fc := newFuncCompiler(img, proto, funcIdx)
		fc.currentModule = fd.Module
		fc.recordTypes = recordTypes
		fc.adtTypes = adtTypes
		compileErr := fc.compile(fd)

		// After a successful compile, run per-proto structural validation on
		// every prototype registered during this FuncDecl's compile — both
		// the top-level proto AND any child lambdas appended to img.Prototypes
		// between protoCheckpoint and the current end.
		//
		// M-BYTECODE-MULTIMODULE M1 uncovered that whole-image Validate() was
		// masking register-allocation bugs in specific stdlib/docparse protos:
		// a single bad proto would fail the entire image. By validating here,
		// we can roll back just this FuncDecl's protos and tag the top-level
		// one EvalOnly, matching the compile-error path.
		//
		// Note: lifted lambdas often live as child NestedProtos, so a parent
		// function with a clean body can still have a buggy child — we must
		// walk the appended range, not just the top-level proto.
		if compileErr == nil {
			topIdx := funcIdx[canonicalFuncName(fd.Module, fd.Name)]
			if vErr := img.ValidatePrototype(topIdx); vErr != nil {
				compileErr = vErr
			} else {
				for childIdx := protoCheckpoint; childIdx < len(img.Prototypes); childIdx++ {
					if vErr := img.ValidatePrototype(childIdx); vErr != nil {
						compileErr = vErr
						break
					}
				}
			}
		}

		if compileErr != nil {
			// Roll back any orphan child lambdas the failed compile registered.
			img.Prototypes = img.Prototypes[:protoCheckpoint]
			// Reset any partial state on the failed prototype itself and tag
			// it as an evaluator-only stub. Name, NumParams and File stay
			// intact so the bridge can still resolve and call it.
			proto.Instructions = nil
			proto.LineInfo = nil
			proto.Constants = nil
			proto.NestedProtos = nil
			proto.NumRegs = 0
			proto.NumCaptures = 0
			proto.EvalOnly = true
			proto.EvalReason = compileErr.Error()
		}
	}

	// Phase 3: pick an entry point. Prefer the first exported function; fall
	// back to the first function. Tests that don't care can ignore EntryPoint.
	entry := -1
	for i := range prog.FuncDecls {
		fd := &prog.FuncDecls[i]
		if fd.Exported {
			entry = funcIdx[canonicalFuncName(fd.Module, fd.Name)]
			break
		}
	}
	if entry == -1 && len(prog.FuncDecls) > 0 {
		fd := &prog.FuncDecls[0]
		entry = funcIdx[canonicalFuncName(fd.Module, fd.Name)]
	}
	if entry >= 0 {
		if err := img.SetEntryPoint(entry); err != nil {
			return nil, fmt.Errorf("compiler: %w", err)
		}
	}

	if err := img.Validate(); err != nil {
		return nil, fmt.Errorf("compiler produced invalid image: %w", err)
	}

	return img, nil
}

// funcCompiler holds per-function state during a single function's lowering.
type funcCompiler struct {
	img         *bytecode.BytecodeImage
	proto       *bytecode.FuncPrototype
	funcIdx     map[string]int // canonical function name → image prototype index
	regs        *regAlloc
	locals      *scopeStack // named local → register
	recordTypes map[string]recordTypeInfo
	adtTypes    map[string]adtTypeInfo

	// currentModule is the module name of the function currently being
	// compiled. Used to canonicalize bare VarRef lookups in funcIdx so that
	// intra-module calls resolve to the right prototype in a multi-module
	// image. Empty string means "single-module program" — bare names are
	// the canonical keys.
	currentModule string

	// currentLine is the source line of the statement currently being
	// compiled. emit() snapshots this into proto.LineInfo for every
	// instruction it appends, so VM runtime errors can report a source
	// location. Zero means "no line info available".
	currentLine int
}

// canonicalFuncName returns the funcIdx key for a function. Single-module
// programs (Module == "") use the bare name; multi-module programs use
// "module/path.name". This must stay in lockstep with Phase 1 registration,
// classifyCallee (call.go), and the expr.go funcIdx lookups.
func canonicalFuncName(module, name string) string {
	if module == "" {
		return name
	}
	return module + "." + name
}

func newFuncCompiler(img *bytecode.BytecodeImage, proto *bytecode.FuncPrototype, funcIdx map[string]int) *funcCompiler {
	alloc := newRegAlloc()
	return &funcCompiler{
		img:     img,
		proto:   proto,
		funcIdx: funcIdx,
		regs:    alloc,
		locals:  newScopeStack(alloc),
	}
}

// compile lowers a single FuncDecl into the funcCompiler's prototype.
func (fc *funcCompiler) compile(fd *stmt.FuncDecl) error {
	// Seed source-location info from the FuncDecl. Statements override the
	// line on entry; if a statement has no line of its own, the function's
	// declaration line is the best fallback.
	fc.proto.File = fd.File
	fc.currentLine = fd.Line

	// Parameters occupy r0..r(N-1). They are pinned for the function's lifetime.
	for _, p := range fd.Params {
		r, err := fc.regs.allocPinned()
		if err != nil {
			return err
		}
		fc.locals.bind(p.Name, r)
	}

	// Body statements.
	for _, s := range fd.Body {
		if err := fc.compileStmt(s); err != nil {
			return err
		}
	}

	// Trailing return expression. The Statement IR convention is that every
	// FuncDecl has a Return expression (which may be nil only for void/Unit
	// returning functions).
	if fd.Return != nil {
		if err := fc.compileReturnExpr(fd.Return); err != nil {
			return err
		}
	} else {
		// Implicit Unit return.
		r, err := fc.regs.allocTemp()
		if err != nil {
			return err
		}
		fc.emit(bytecode.EncodeABC(bytecode.OpLoadNil, r, 0, 0))
		fc.emit(bytecode.EncodeABC(bytecode.OpReturn, r, 0, 0))
	}

	fc.proto.NumRegs = fc.regs.highWater()
	return nil
}

// emit appends an instruction to the current prototype. It also pushes the
// current source line into proto.LineInfo so the slot stays in lockstep with
// Instructions (Image.Validate enforces equal lengths).
func (fc *funcCompiler) emit(inst bytecode.Instruction) {
	fc.proto.Instructions = append(fc.proto.Instructions, inst)
	fc.proto.LineInfo = append(fc.proto.LineInfo, fc.currentLine)
}

// addLocalConst registers a Value in the image constant pool and returns the
// LOCAL constant index (suitable for an OpLoadConst Bx). The local table is
// deduplicated against the pool index, so two LitInt(42) in one function
// share a single local entry.
func (fc *funcCompiler) addLocalConst(v bytecode.Value) (uint16, error) {
	poolIdx := fc.img.AddConstant(v)
	for i, existing := range fc.proto.Constants {
		if existing == poolIdx {
			if i > 0xFFFF {
				return 0, fmt.Errorf("constant table overflow (%d entries)", i)
			}
			return uint16(i), nil
		}
	}
	fc.proto.Constants = append(fc.proto.Constants, poolIdx)
	idx := len(fc.proto.Constants) - 1
	if idx > 0xFFFF {
		return 0, fmt.Errorf("constant table overflow (%d entries)", idx)
	}
	return uint16(idx), nil
}
