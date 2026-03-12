package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/core"
)

// encodeApp handles function application.
// After op_lowering, most operators appear as App(VarGlobal($builtin.XXX), args).
func encodeApp(app *core.App) (string, error) {
	// Check for builtin operator pattern: App(VarGlobal($builtin.XXX), args)
	if vg, ok := app.Func.(*core.VarGlobal); ok && vg.Ref.Module == "$builtin" {
		// Standard builtins (direct op mapping)
		smtOp, isBuiltin := BuiltinToSMTOp[vg.Ref.Name]
		if isBuiltin {
			return encodeBuiltinOp(smtOp, app.Args)
		}
		// String builtins with special encoding
		if spec, ok := StringBuiltinSpecial[vg.Ref.Name]; ok {
			return encodeStringBuiltin(spec, app.Args)
		}
		// List builtins with special encoding
		if spec, ok := ListBuiltinSpecial[vg.Ref.Name]; ok {
			return encodeListBuiltin(spec, app.Args)
		}
		// Numeric conversion builtins (intToFloat → to_real, floatToInt → to_int)
		if spec, ok := NumericBuiltinSpecial[vg.Ref.Name]; ok {
			return encodeNumericBuiltin(spec, app.Args)
		}
	}

	// Check for stdlib function with SMT mapping (std/string, std/list wrappers)
	if vg, ok := app.Func.(*core.VarGlobal); ok {
		if builtinName, mapped := ResolveStdlibToBuiltin(vg.Ref.Module, vg.Ref.Name); mapped {
			if spec, ok := StringBuiltinSpecial[builtinName]; ok {
				return encodeStringBuiltin(spec, app.Args)
			}
			if spec, ok := ListBuiltinSpecial[builtinName]; ok {
				return encodeListBuiltin(spec, app.Args)
			}
		}
	}

	// Check for curried builtin: App(App(VarGlobal($builtin.XXX), [arg1]), [arg2])
	if innerApp, ok := app.Func.(*core.App); ok {
		if vg, ok := innerApp.Func.(*core.VarGlobal); ok && vg.Ref.Module == "$builtin" {
			// Combine args: inner args + outer args
			allArgs := make([]core.CoreExpr, 0, len(innerApp.Args)+len(app.Args))
			allArgs = append(allArgs, innerApp.Args...)
			allArgs = append(allArgs, app.Args...)

			smtOp, isBuiltin := BuiltinToSMTOp[vg.Ref.Name]
			if isBuiltin {
				return encodeBuiltinOp(smtOp, allArgs)
			}
			// String builtins with special encoding (curried)
			if spec, ok := StringBuiltinSpecial[vg.Ref.Name]; ok {
				return encodeStringBuiltin(spec, allArgs)
			}
			// List builtins with special encoding (curried)
			if spec, ok := ListBuiltinSpecial[vg.Ref.Name]; ok {
				return encodeListBuiltin(spec, allArgs)
			}
			// Numeric conversion builtins (curried)
			if spec, ok := NumericBuiltinSpecial[vg.Ref.Name]; ok {
				return encodeNumericBuiltin(spec, allArgs)
			}
		}
		// Curried stdlib call: App(App(VarGlobal(std/string.XXX), [arg1]), [arg2])
		if vg, ok := innerApp.Func.(*core.VarGlobal); ok {
			if builtinName, mapped := ResolveStdlibToBuiltin(vg.Ref.Module, vg.Ref.Name); mapped {
				allArgs := make([]core.CoreExpr, 0, len(innerApp.Args)+len(app.Args))
				allArgs = append(allArgs, innerApp.Args...)
				allArgs = append(allArgs, app.Args...)
				if spec, ok := StringBuiltinSpecial[builtinName]; ok {
					return encodeStringBuiltin(spec, allArgs)
				}
				if spec, ok := ListBuiltinSpecial[builtinName]; ok {
					return encodeListBuiltin(spec, allArgs)
				}
			}
		}
	}

	// Check for std/list builtins (:: and concat_List registered under std/list module)
	if vg, ok := app.Func.(*core.VarGlobal); ok && vg.Ref.Module == "std/list" {
		if spec, ok := ListBuiltinSpecial[vg.Ref.Name]; ok {
			return encodeListBuiltin(spec, app.Args)
		}
	}

	// Cross-function call: App(VarGlobal(module.funcName), args)
	// where funcName has been resolved as a define-fun callee
	if vg, ok := app.Func.(*core.VarGlobal); ok && vg.Ref.Module != "$builtin" && vg.Ref.Module != "std/list" && vg.Ref.Module != "std/string" {
		if activeResolvedCallees != nil && activeResolvedCallees[vg.Ref.Name] {
			return encodeUserFunctionCall(vg.Ref.Name, app.Args)
		}
		// ADT constructor application
		name := stripConstructorPrefix(vg.Ref.Name)
		return encodeConstructorApp(name, app.Args)
	}

	// Plain variable application — check if it's a resolved callee (same-module function call)
	if v, ok := app.Func.(*core.Var); ok {
		if activeResolvedCallees != nil && activeResolvedCallees[v.Name] {
			return encodeUserFunctionCall(v.Name, app.Args)
		}
		// Otherwise treat as constructor reference
		name := stripConstructorPrefix(v.Name)
		if len(app.Args) == 0 {
			return name, nil
		}
		return encodeConstructorApp(name, app.Args)
	}

	return "", fmt.Errorf("unsupported application: %s", app.String())
}

// encodeBuiltinOp encodes a builtin operator application.
func encodeBuiltinOp(smtOp string, args []core.CoreExpr) (string, error) {
	if len(args) == 1 {
		// Unary: (- x) or (not x)
		arg, err := EncodeExpr(args[0])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s %s)", smtOp, arg), nil
	}
	if len(args) == 2 {
		// Binary: (>= x 0)
		left, err := EncodeExpr(args[0])
		if err != nil {
			return "", err
		}
		right, err := EncodeExpr(args[1])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s %s %s)", smtOp, left, right), nil
	}
	return "", fmt.Errorf("builtin %q with unexpected arity %d", smtOp, len(args))
}

// encodeStringBuiltin encodes a string builtin with special handling.
func encodeStringBuiltin(spec StringBuiltinSpec, args []core.CoreExpr) (string, error) {
	if spec.Unary {
		if len(args) != 1 {
			return "", fmt.Errorf("string builtin %q expects 1 arg, got %d", spec.Op, len(args))
		}
		arg, err := EncodeExpr(args[0])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s %s)", spec.Op, arg), nil
	}

	// SubstrMode: _str_slice(s, start, end) → (str.substr s start (- end start))
	if spec.SubstrMode {
		if len(args) != 3 {
			return "", fmt.Errorf("string builtin %q expects 3 args, got %d", spec.Op, len(args))
		}
		s, err := EncodeExpr(args[0])
		if err != nil {
			return "", err
		}
		start, err := EncodeExpr(args[1])
		if err != nil {
			return "", err
		}
		end, err := EncodeExpr(args[2])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s %s %s (- %s %s))", spec.Op, s, start, end, start), nil
	}

	if len(args) < 2 {
		return "", fmt.Errorf("string builtin %q expects at least 2 args, got %d", spec.Op, len(args))
	}

	left, err := EncodeExpr(args[0])
	if err != nil {
		return "", err
	}
	right, err := EncodeExpr(args[1])
	if err != nil {
		return "", err
	}

	if spec.FlipArgs {
		left, right = right, left
	}

	if spec.AppendZero {
		return fmt.Sprintf("(%s %s %s 0)", spec.Op, left, right), nil
	}

	return fmt.Sprintf("(%s %s %s)", spec.Op, left, right), nil
}

// encodeUserFunctionCall encodes a call to a user-defined function
// that has been resolved as a define-fun in the SMT-LIB context.
func encodeUserFunctionCall(funcName string, args []core.CoreExpr) (string, error) {
	if len(args) == 0 {
		return fmt.Sprintf("(%s)", funcName), nil
	}
	encodedArgs := make([]string, len(args))
	for i, arg := range args {
		encoded, err := EncodeExpr(arg)
		if err != nil {
			return "", fmt.Errorf("function call %s arg %d: %w", funcName, i, err)
		}
		encodedArgs[i] = encoded
	}
	return fmt.Sprintf("(%s %s)", funcName, strings.Join(encodedArgs, " ")), nil
}

// encodeConstructorApp encodes an ADT constructor application.
func encodeConstructorApp(ctorName string, args []core.CoreExpr) (string, error) {
	if len(args) == 0 {
		return ctorName, nil
	}
	encodedArgs := make([]string, len(args))
	for i, arg := range args {
		encoded, err := EncodeExpr(arg)
		if err != nil {
			return "", fmt.Errorf("constructor arg %d: %w", i, err)
		}
		encodedArgs[i] = encoded
	}
	return fmt.Sprintf("(%s %s)", ctorName, strings.Join(encodedArgs, " ")), nil
}

// encodeListBuiltin encodes a list builtin with special handling.
func encodeListBuiltin(spec ListBuiltinSpec, args []core.CoreExpr) (string, error) {
	if spec.ConsMode {
		// :: (cons): (seq.++ (seq.unit head) tail)
		if len(args) != 2 {
			return "", fmt.Errorf("cons (::) expects 2 args, got %d", len(args))
		}
		head, err := EncodeExpr(args[0])
		if err != nil {
			return "", err
		}
		tail, err := EncodeExpr(args[1])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(seq.++ (seq.unit %s) %s)", head, tail), nil
	}

	if spec.TailMode {
		// _list_tail(xs) → (seq.extract xs 1 (- (seq.len xs) 1))
		if len(args) != 1 {
			return "", fmt.Errorf("list builtin %q expects 1 arg, got %d", spec.Op, len(args))
		}
		arg, err := EncodeExpr(args[0])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(seq.extract %s 1 (- (seq.len %s) 1))", arg, arg), nil
	}

	if spec.Unary {
		if len(args) != 1 {
			return "", fmt.Errorf("list builtin %q expects 1 arg, got %d", spec.Op, len(args))
		}
		arg, err := EncodeExpr(args[0])
		if err != nil {
			return "", err
		}
		if spec.AppendZero {
			// _list_head: (seq.nth xs 0)
			return fmt.Sprintf("(%s %s 0)", spec.Op, arg), nil
		}
		return fmt.Sprintf("(%s %s)", spec.Op, arg), nil
	}

	// Binary: concat_List(xs, ys) → (seq.++ xs ys), _list_nth(xs, i) → (seq.nth xs i)
	if len(args) < 2 {
		return "", fmt.Errorf("list builtin %q expects at least 2 args, got %d", spec.Op, len(args))
	}

	left, err := EncodeExpr(args[0])
	if err != nil {
		return "", err
	}
	right, err := EncodeExpr(args[1])
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("(%s %s %s)", spec.Op, left, right), nil
}

// encodeNumericBuiltin encodes numeric conversion builtins (intToFloat → to_real, floatToInt → to_int).
func encodeNumericBuiltin(spec NumericBuiltinSpec, args []core.CoreExpr) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("numeric builtin %q expects 1 arg, got %d", spec.Op, len(args))
	}
	arg, err := EncodeExpr(args[0])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("(%s %s)", spec.Op, arg), nil
}
