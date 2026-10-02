package testing

// `ailang test --bytecode` (#1487): named-test bodies on the bytecode VM.
//
// Same contract as `ailang run --bytecode`: the VM runs what it can compile,
// and anything it cannot — a body the compiler cannot lower, an
// evaluator-only entry, a VM runtime error — falls back to the evaluator, so
// every outcome is the evaluator's outcome unless the VM produced a value.
// `--strict-bytecode` turns each fallback into a failure of that test instead.
//
// The VM path compiles the body as a nullary function in the test module,
//
//	pure func __namedtest_entry() -> bool { <body> }    (-> int for assert bodies)
//
// through the same pipeline as the evaluator path, then compiles every loaded
// module with runner.CompileBytecodeFromResult — the function `ailang run
// --bytecode` uses. Functions the compiler leaves evaluator-only are bridged
// to the harness evaluator (module-scoped, see module_scope.go). The result
// is the same bool / int sentinel the evaluator path produces, so pass/fail,
// error text and the assert decoding are shared.
//
// Inline `tests [...]` tables and property harnesses are Core built in Go
// with no surface source or type info to lower, so they stay on the
// evaluator under both flags. Property seeding is engine-independent.

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/runner"
	"github.com/sunholo-data/ailang/internal/vm"
)

// namedTestEntry is the synthesized entry function's name. The leading
// underscores keep it private and out of any user's namespace.
const namedTestEntry = "__namedtest_entry"

// defaultVMMaxStack matches the evaluator's default recursion limit so a body
// that recurses deeply fails (or passes) at the same depth on both engines.
const defaultVMMaxStack = 10000

// EngineStats counts where named-test bodies ran under --bytecode. It is not
// part of the test report (outcomes are engine-independent); the CLI prints
// it on stderr so a fallback is never silent.
type EngineStats struct {
	VMBodies       int    // bodies whose result came from the VM
	FallbackBodies int    // bodies that fell back to the evaluator
	StrictFailures int    // bodies failed by --strict-bytecode
	FirstFallback  string // reason for the first fallback, for the summary
}

// Merge adds other's counts into s.
func (s *EngineStats) Merge(other EngineStats) {
	s.VMBodies += other.VMBodies
	s.FallbackBodies += other.FallbackBodies
	s.StrictFailures += other.StrictFailures
	if s.FirstFallback == "" {
		s.FirstFallback = other.FirstFallback
	}
}

func (s *EngineStats) noteFallback(err error) {
	s.FallbackBodies++
	if s.FirstFallback == "" {
		s.FirstFallback = err.Error()
	}
}

// Summary is the one-line engine report for stderr.
func (s EngineStats) Summary() string {
	msg := fmt.Sprintf("bytecode: %d named-test bod%s ran on the VM, %d fell back to the evaluator",
		s.VMBodies, plural(s.VMBodies, "y", "ies"), s.FallbackBodies)
	if s.StrictFailures > 0 {
		msg += fmt.Sprintf(", %d failed under --strict-bytecode", s.StrictFailures)
	}
	if s.FirstFallback != "" {
		msg += " (first fallback: " + firstLine(s.FirstFallback) + ")"
	}
	return msg + "; inline tests and properties run on the evaluator"
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// evalNamedTestBodyOnVM runs a folded named-test body on the bytecode VM and
// returns its raw value (bool, or the int assert sentinel when sentinel is
// set). Any error means the VM produced no value; the caller decides between
// evaluator fallback and a strict failure.
func (e *Executor) evalNamedTestBodyOnVM(baseSource string, hasModule bool, folded ast.Expr, sentinel bool) (eval.Value, error) {
	resultType := "bool"
	if sentinel {
		resultType = "int"
	}
	var sb strings.Builder
	sb.WriteString(baseSource)
	sb.WriteString("\npure func " + namedTestEntry + "() -> " + resultType + " {\n  ")
	sb.WriteString(PrintAILANGSource(folded))
	sb.WriteString("\n}\n")

	res, err := e.runNamedTestPipeline(sb.String(), hasModule)
	if err != nil {
		return nil, fmt.Errorf("compile body: %w", err)
	}
	e.cacheModules(&res)

	// The lower pass still panics on some unbridged shapes; that is a compile
	// failure, exactly as `ailang run --bytecode` treats it.
	var img *bytecode.BytecodeImage
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("compile panic: %v", r)
			}
		}()
		img, err = runner.CompileBytecodeFromResult(res, "test")
	}()
	if err != nil {
		return nil, fmt.Errorf("bytecode compile: %w", err)
	}
	if err := img.Validate(); err != nil {
		return nil, fmt.Errorf("bytecode validate: %w", err)
	}
	proto := runner.FindEntryProto(img, namedTestEntry)
	if proto == nil {
		return nil, fmt.Errorf("test body entry not found in bytecode image")
	}
	if proto.EvalOnly {
		return nil, fmt.Errorf("test body is evaluator-only (%s)", proto.EvalReason)
	}

	machine := vm.NewVM(img)
	machine.MaxStack = defaultVMMaxStack
	if e.maxRecursionDepth > 0 {
		machine.MaxStack = e.maxRecursionDepth
	}
	if !e.strictBytecode {
		machine.Interop = &harnessBridge{evaluator: e.newHarnessEvaluator()}
	}

	var args []bytecode.Value
	if proto.NumParams == 1 { // the lower pass gives nullary functions a Unit parameter
		args = []bytecode.Value{bytecode.Unit()}
	}
	out, err := machine.Run(proto, args)
	if err != nil {
		return nil, err // a *vm.VMError already says "vm: ..."
	}
	val, err := vm.BytecodeToEval(out)
	if err != nil {
		return nil, fmt.Errorf("vm result: %w", err)
	}
	return val, nil
}

// harnessBridge implements vm.EvalInterop over the harness evaluator: an
// evaluator-only function is called by its canonical "<module>.<name>", which
// is exactly the key injectModuleBindings binds every module function under in
// the shared environment — so the bridge resolves through the same module
// scoping as the evaluator path and never by a bare name that another module
// could shadow (#1461).
type harnessBridge struct {
	evaluator *eval.CoreEvaluator
}

func (b *harnessBridge) CallEvalFunc(name string, args []bytecode.Value) (bytecode.Value, error) {
	fn, ok := b.evaluator.Env().Get(name)
	if !ok || fn == nil {
		return bytecode.Value{}, fmt.Errorf("evaluator-only function %q is not bound in the test harness", name)
	}
	evArgs := make([]eval.Value, len(args))
	for i, a := range args {
		v, err := vm.BytecodeToEval(a)
		if err != nil {
			return bytecode.Value{}, fmt.Errorf("arg %d of %s: %w", i, name, err)
		}
		evArgs[i] = v
	}
	result, err := b.evaluator.CallValueN(fn, evArgs)
	if err != nil {
		return bytecode.Value{}, err
	}
	return vm.EvalToBytecode(result)
}
