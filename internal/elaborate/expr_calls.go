package elaborate

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
)

// Function call normalization

// normalizeFuncCall handles function application
func (e *Elaborator) normalizeFuncCall(app *ast.FuncCall) (core.CoreExpr, error) {
	// Check if this is the :: (cons) operator used as an expression.
	// The parser desugars x :: xs to FuncCall{"::", [x, xs]}.
	// If :: is registered in globalEnv (via AddBuiltinsToGlobalEnv), route
	// it to the builtin via VarGlobal. Otherwise fall through to general path.
	if ident, ok := app.Func.(*ast.Identifier); ok && ident.Name == "::" {
		if ref, found := e.globalEnv["::"]; found {
			var allBindings []binding
			var atomicArgs []core.CoreExpr

			for _, arg := range app.Args {
				atomic, binds, err := e.normalizeToAtomic(arg)
				if err != nil {
					return nil, err
				}
				atomicArgs = append(atomicArgs, atomic)
				allBindings = append(allBindings, binds...)
			}

			consRef := &core.VarGlobal{
				CoreNode: e.makeNode(app.Position()),
				Ref:      ref,
			}

			result := &core.App{
				CoreNode: e.makeNode(app.Position()),
				Func:     consRef,
				Args:     atomicArgs,
			}

			return e.wrapWithBindings(result, allBindings), nil
		}
	}

	// Check if this is a constructor call
	if ident, ok := app.Func.(*ast.Identifier); ok {
		if ctorInfo, isConstructor := e.constructors[ident.Name]; isConstructor {
			// This is a constructor! Emit $adt factory call
			// Transform Some(x) → $adt.make_Option_Some(x)
			factoryName := fmt.Sprintf("make_%s_%s", ctorInfo.TypeName, ctorInfo.CtorName)

			// Normalize arguments to atomic values
			var allBindings []binding
			var atomicArgs []core.CoreExpr

			for _, arg := range app.Args {
				atomic, binds, err := e.normalizeToAtomic(arg)
				if err != nil {
					return nil, err
				}
				atomicArgs = append(atomicArgs, atomic)
				allBindings = append(allBindings, binds...)
			}

			// Create factory function reference
			factoryRef := &core.VarGlobal{
				CoreNode: e.makeNode(app.Position()),
				Ref: core.GlobalRef{
					Module: "$adt",
					Name:   factoryName,
				},
			}

			result := &core.App{
				CoreNode: e.makeNode(app.Position()),
				Func:     factoryRef,
				Args:     atomicArgs,
			}

			return e.wrapWithBindings(result, allBindings), nil
		}
	}

	// Not a constructor - handle as normal function call
	fun, err := e.normalize(app.Func)
	if err != nil {
		return nil, err
	}

	var allBindings []binding
	var atomicArgs []core.CoreExpr

	for _, arg := range app.Args {
		atomic, binds, err := e.normalizeToAtomic(arg)
		if err != nil {
			return nil, err
		}
		atomicArgs = append(atomicArgs, atomic)
		allBindings = append(allBindings, binds...)
	}

	// If function is not atomic, bind it too
	if !core.IsAtomic(fun) {
		freshVar := e.freshVar()
		allBindings = append(allBindings, binding{Name: freshVar, Value: fun})
		fun = &core.Var{
			CoreNode: e.makeNode(app.Position()),
			Name:     freshVar,
		}
	}

	result := &core.App{
		CoreNode: e.makeNode(app.Position()),
		Func:     fun,
		Args:     atomicArgs,
	}

	return e.wrapWithBindings(result, allBindings), nil
}
