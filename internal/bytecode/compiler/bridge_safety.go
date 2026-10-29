package compiler

import (
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// Bridge safety net (M-VM-COVERAGE G4, safety net only).
//
// A native function may call an evaluator-only one through the interop
// bridge, which converts each argument VM→evaluator and the result back.
// The bridge cannot carry a closure in either direction, nor any ADT from the
// VM (a VM ADT has no type identity, internal/vm/convert.go), nor a user ADT
// back from the evaluator. Hitting one of those is a run-time error, often
// after effects have run, so the runner cannot replay the program on the
// evaluator (#1545) and a program that is correct on the evaluator fails.
//
// Argument and result types are static, so the failure can be predicted at
// compile time. A native function that references an evaluator-only function
// whose signature may carry such a value is made evaluator-only too, to a
// fixed point. If that reaches `main`, the runner hands the whole program to
// the evaluator before anything runs.
//
// The check is deliberately conservative: an unknown or erased type counts
// as unsafe. A false positive costs speed (a function runs on the evaluator);
// a false negative costs a crash. The type projection erases ADT type
// arguments, so `Option[UserADT]` reads as `Option` and is not caught.

// markBridgeUnsafeCallers runs after every FuncDecl is compiled.
func markBridgeUnsafeCallers(img *bytecode.BytecodeImage, prog *stmt.Program, funcIdx map[string]int, edges map[int]map[int]bool, adtTypes map[string]adtTypeInfo) {
	// Declaration order, so the EvalReason is deterministic when several
	// edges are unsafe.
	declOf := make(map[int]*stmt.FuncDecl, len(prog.FuncDecls))
	callers := make([]int, 0, len(prog.FuncDecls))
	for i := range prog.FuncDecls {
		fd := &prog.FuncDecls[i]
		idx := funcIdx[canonicalFuncName(fd.Module, fd.Name)]
		declOf[idx] = fd
		if edges[idx] != nil {
			callers = append(callers, idx)
		}
	}

	for changed := true; changed; {
		changed = false
		for _, caller := range callers {
			proto := img.Prototypes[caller]
			if proto.EvalOnly {
				continue
			}
			for _, callee := range sortedKeys(edges[caller]) {
				target := img.Prototypes[callee]
				if !target.EvalOnly {
					continue
				}
				why := unbridgeableSignature(declOf[callee], adtTypes)
				if why == "" {
					continue
				}
				// Keep the root cause: the callee's own reason, which is
				// usually the thing to fix.
				markEvalOnly(proto, fmt.Sprintf("references evaluator-only %s, whose %s cannot cross the VM/evaluator bridge; %s is evaluator-only because: %s",
					target.Name, why, target.Name, target.EvalReason))
				changed = true
				break
			}
		}
	}
}

func markEvalOnly(p *bytecode.FuncPrototype, reason string) {
	p.Instructions = nil
	p.LineInfo = nil
	p.Constants = nil
	p.NestedProtos = nil
	p.NumRegs = 0
	p.NumCaptures = 0
	p.EvalOnly = true
	p.EvalReason = reason
}

func sortedKeys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// unbridgeableSignature names the part of fd's signature the bridge may not
// convert, or returns "" if every argument and the result are safe.
func unbridgeableSignature(fd *stmt.FuncDecl, adtTypes map[string]adtTypeInfo) string {
	if fd == nil || fd.StubSignature {
		return "signature (unknown: its lowering failed)"
	}
	for i, p := range fd.Params {
		if why := unsafeType(p.Type, toEvaluator, adtTypes); why != "" {
			return fmt.Sprintf("parameter %d (%s)", i+1, why)
		}
	}
	if why := unsafeType(fd.ReturnType, toVM, adtTypes); why != "" {
		return fmt.Sprintf("result (%s)", why)
	}
	return ""
}

type bridgeDir int

const (
	toEvaluator bridgeDir = iota // an argument: bytecodeToEval
	toVM                         // a result: EvalToBytecode
)

// bridgeStdADTs are the ADTs EvalToBytecode can name (bytecode.StdADTTag
// plus every std ADT it finds by ordinal). A user type of the same name is
// indistinguishable here; the projection carries no module.
var bridgeStdADTs = map[string]bool{"Option": true, "Result": true}

func unsafeType(t stmt.ResolvedType, dir bridgeDir, adtTypes map[string]adtTypeInfo) string {
	switch t := t.(type) {
	case stmt.PrimitiveType:
		return ""
	case stmt.SliceType:
		return unsafeType(t.Elem, dir, adtTypes)
	case stmt.TupleType:
		for _, e := range t.Elems {
			if why := unsafeType(e, dir, adtTypes); why != "" {
				return why
			}
		}
		return ""
	case stmt.FuncType:
		return "a function value"
	case stmt.NamedType:
		if !t.Pointer {
			// A nominal record: its field types are not projected.
			return "record " + t.Name + " (field types unknown)"
		}
		if dir == toVM && bridgeStdADTs[t.Name] {
			return ""
		}
		if _, isADT := adtTypes[t.Name]; isADT || dir == toEvaluator {
			return "ADT " + t.Name
		}
		return "opaque type " + t.Name
	case stmt.MapType:
		return "a map (element types unknown)"
	case stmt.InterfaceType, nil:
		return "an erased or polymorphic type"
	}
	return fmt.Sprintf("type %T", t)
}
