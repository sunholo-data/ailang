package compiler

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/bytecode"
	"github.com/sunholo-data/ailang/internal/gen/stmt"
)

// Implicit prelude (M-VM-COVERAGE G2).
//
// An entry module may call `println` with no import: the type checker
// injects its type (pipeline.InjectPrelude), and the evaluator has a
// builtin of that name. The lowering emits a bare VarRef for it, which the
// compiler used to report as an unbound name. That made `main`
// evaluator-only, so the runner handed the whole program to the evaluator.
//
// A bare name that no local, letrec member or top-level function binds, and
// that bytecode.PreludeFuncs lists, now resolves to a synthetic prototype
// wrapping the prelude function's effect builtin. It is registered once per
// image, under a key no source identifier can spell. Only a type-checked
// program reaches the compiler, and the type checker gives the prelude to
// entry modules alone, so a library module's bare `println` is still a type
// error.

// preludeKey is the funcIdx key of the synthetic prototype for name.
func preludeKey(name string) string { return "$prelude." + name }

// preludeFunc returns the funcIdx key of the synthetic prototype for a
// prelude function, compiling it into the image on first use. The caller
// must already have ruled out every other binding of name.
func (fc *funcCompiler) preludeFunc(name string) (string, bool, error) {
	pf, ok := bytecode.PreludeFuncs[name]
	if !ok {
		return "", false, nil
	}
	key := preludeKey(name)
	if _, done := fc.funcIdx[key]; done {
		return key, true, nil
	}
	params := make([]stmt.Param, pf.Arity)
	args := make([]stmt.Expr, pf.Arity)
	for i := range params {
		params[i] = stmt.Param{Name: fmt.Sprintf("$a%d", i)}
		args[i] = stmt.VarRef{Name: params[i].Name}
	}
	proto := &bytecode.FuncPrototype{Name: key, NumParams: uint8(pf.Arity)}
	fc.funcIdx[key] = fc.img.AddPrototype(proto)
	inner := fc.child(proto)
	inner.edges = map[int]bool{} // its own body; nothing to check
	fd := &stmt.FuncDecl{
		Name:   key,
		Params: params,
		Return: stmt.BuiltinCall{Name: pf.Effect, Args: args},
		File:   fc.proto.File,
	}
	if err := inner.compile(fd); err != nil {
		return "", false, fmt.Errorf("prelude %s: %w", name, err)
	}
	return key, true, nil
}
