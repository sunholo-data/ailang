package types

import "github.com/sunholo-data/ailang/internal/core"

// applicationDiagnosticPath annotates the application equality itself, so both
// local and deferred constraint solving preserve callee names without changing
// raw unification errors or their arity codes. Anonymous heads stay neutral.
func applicationDiagnosticPath(app *core.App) []string {
	name, from := coreCalleeName(app.Func)
	context := "function application"
	if name != "" {
		context += " of " + name
	}
	path := []string{context + " at " + app.Span().String()}
	if name != "" {
		if hint := collisionHint(name, from); hint != "" {
			path = append(path, hint)
		}
	}
	return path
}

// coreCalleeName returns the bare name of an applied callee and the module it
// resolved from ("" if unknown), or ("","") when the callee is not a variable
// reference. Used to drive the stdlib name-collision hint on a failing application.
func coreCalleeName(fn core.CoreExpr) (name, from string) {
	switch v := fn.(type) {
	case *core.Var:
		return v.Name, ""
	case *core.VarGlobal:
		return v.Ref.Name, v.Ref.Module
	}
	return "", ""
}
