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

	"github.com/sunholo/ailang/internal/bytecode"
	"github.com/sunholo/ailang/internal/gen/stmt"
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
	funcIdx := make(map[string]int, len(prog.FuncDecls))
	for i := range prog.FuncDecls {
		fd := &prog.FuncDecls[i]
		proto := &bytecode.FuncPrototype{
			Name:      fd.Name,
			NumParams: uint8(len(fd.Params)),
		}
		idx := img.AddPrototype(proto)
		funcIdx[fd.Name] = idx
	}

	// Phase 2: compile each function body.
	for i := range prog.FuncDecls {
		fd := &prog.FuncDecls[i]
		proto := img.Prototypes[funcIdx[fd.Name]]
		fc := newFuncCompiler(img, proto, funcIdx)
		fc.recordTypes = recordTypes
		fc.adtTypes = adtTypes
		if err := fc.compile(fd); err != nil {
			return nil, fmt.Errorf("compiling %s: %w", fd.Name, err)
		}
	}

	// Phase 3: pick an entry point. Prefer the first exported function; fall
	// back to the first function. Tests that don't care can ignore EntryPoint.
	entry := -1
	for i := range prog.FuncDecls {
		if prog.FuncDecls[i].Exported {
			entry = funcIdx[prog.FuncDecls[i].Name]
			break
		}
	}
	if entry == -1 && len(prog.FuncDecls) > 0 {
		entry = funcIdx[prog.FuncDecls[0].Name]
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
	funcIdx     map[string]int // global function name → image prototype index
	regs        *regAlloc
	locals      *scopeStack // named local → register
	recordTypes map[string]recordTypeInfo
	adtTypes    map[string]adtTypeInfo
}

func newFuncCompiler(img *bytecode.BytecodeImage, proto *bytecode.FuncPrototype, funcIdx map[string]int) *funcCompiler {
	return &funcCompiler{
		img:     img,
		proto:   proto,
		funcIdx: funcIdx,
		regs:    newRegAlloc(),
		locals:  newScopeStack(),
	}
}

// compile lowers a single FuncDecl into the funcCompiler's prototype.
func (fc *funcCompiler) compile(fd *stmt.FuncDecl) error {
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

// emit appends an instruction to the current prototype.
func (fc *funcCompiler) emit(inst bytecode.Instruction) {
	fc.proto.Instructions = append(fc.proto.Instructions, inst)
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
