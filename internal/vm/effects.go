package vm

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/eval"
)

// effectCallbackValue carries an arbitrary VM result through the host scope.
// The host only embeds it in Ok: it never inspects or executes the value.
// This preserves package ADTs and closures without guessing their identities.
type effectCallbackValue struct{ value bytecode.Value }

func (v *effectCallbackValue) Type() string { return "vm_callback_result" }

// This diagnostic enters the host effect trace. Keep it bounded even when
// the callback returns a large model: the real result travels separately.
func (v *effectCallbackValue) String() string {
	return fmt.Sprintf("<VM callback result: %s>", v.value.Tag)
}

func (vm *VM) callEffectBuiltin(index int, args []bytecode.Value) (bytecode.Value, error) {
	if index < 0 || index >= len(bytecode.EffectBuiltinNames) {
		return bytecode.Value{}, fmt.Errorf("unknown effect builtin index %d", index)
	}
	if vm.Effects == nil {
		return bytecode.Value{}, fmt.Errorf("effect context is not configured")
	}
	name := strings.TrimPrefix(bytecode.EffectBuiltinNames[index], "_")
	spec, ok := builtins.GetSpec(name)
	if !ok {
		return bytecode.Value{}, fmt.Errorf("missing registered effect builtin %s", name)
	}
	if len(args) != spec.NumArgs {
		return bytecode.Value{}, fmt.Errorf("%s: expected %d args, got %d", name, spec.NumArgs, len(args))
	}
	converted := make([]eval.Value, len(args))
	for i, a := range args {
		if name == "_terminal_withTerminal" && i == 1 {
			closure := a
			converted[i] = &eval.BuiltinFunction{Name: "vm_terminal_body", Fn: func(argv []eval.Value) (eval.Value, error) {
				if len(argv) != 1 {
					return nil, fmt.Errorf("terminal callback expected session")
				}
				session, err := EvalToBytecode(argv[0])
				if err != nil {
					return nil, err
				}
				previousCharge := vm.Effects.SaveAndResetBudgetChargeScope()
				defer vm.Effects.RestoreBudgetChargeScope(previousCharge)
				result, err := vm.CallClosure(closure, []bytecode.Value{session})
				if err != nil {
					return nil, err
				}
				return &effectCallbackValue{value: result}, nil
			}}
		} else {
			var err error
			converted[i], err = effectArgumentToEval(a)
			if err != nil {
				return bytecode.Value{}, fmt.Errorf("%s argument %d: %w", name, i, err)
			}
		}
	}
	if name == "_terminal_withTerminal" {
		previous := vm.Effects.FnCaller
		vm.Effects.FnCaller = func(fn eval.Value, arg eval.Value) (eval.Value, error) {
			if fn == converted[1] {
				return fn.(*eval.BuiltinFunction).Fn([]eval.Value{arg})
			}
			if previous == nil {
				return nil, fmt.Errorf("unconfigured callback")
			}
			return previous(fn, arg)
		}
		defer func() { vm.Effects.FnCaller = previous }()
	}
	vm.EffectCalls++
	result, err := spec.Impl(vm.Effects, converted)
	if err != nil {
		return bytecode.Value{}, err
	}
	return EvalToBytecode(result)
}

func effectArgumentToEval(v bytecode.Value) (eval.Value, error) {
	// Only TerminalSession crosses VM -> host as an ADT. Its constructor is
	// representational; the host checks lease/context/token authority separately.
	if v.Tag == bytecode.TagADT && v.AsADT().Ctor == "TerminalSession" {
		a := v.AsADT()
		if a.Tag != 0 || len(a.Fields) != 1 || a.Fields[0].Tag != bytecode.TagInt {
			return nil, fmt.Errorf("invalid terminal handle representation")
		}
		return &eval.TaggedValue{ModulePath: "std/terminal", TypeName: "TerminalSession", CtorName: a.Ctor, Fields: []eval.Value{&eval.IntValue{Value: int(a.Fields[0].Int)}}}, nil
	}
	return BytecodeToEval(v)
}
