package testing

// forall properties run as compiled functions (#624; M-TEST-RUNNER-COMPILE-ONCE
// Phase 2, design_docs/planned/v0_53_0/m-test-runner-compile-once.md).
//
// A forall property used to be evaluated by splicing each generated value
// into the predicate as a literal and compiling the result as a module-less
// program: one compile per generated case (100 per property, plus shrinking).
// That program had no imports and no module scope, so every forall property
// failed on its first case ("empty program", or a parse error once the body
// called a module function: #624). The property is now one entry of the
// file's batch,
//
//	pure func __namedtest_prop_<k>(<binder>: <type>, ...) { <predicate> }
//
// and each generated case is a call with the generated values as arguments:
// no splice, no per-case compile. Binder types are annotated because the
// generated value's type must drive operator resolution (`x + 1.0` on a float
// binder must not default to int).

import (
	"fmt"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/loader"
)

// propertyCall evaluates a forall predicate on one generated case.
type propertyCall func(args []eval.Value) (eval.Value, error)

// isForall reports whether a property runs on the forall path (runProperty
// routes ensures and requires to their own harnesses).
func isForall(p *ast.Property) bool {
	return p.Kind != ast.EnsuresKind && p.Kind != ast.RequiresKind
}

// binderParams renders the binders as a parameter list, `n: int, xs: [int]`.
func binderParams(p *ast.Property) string {
	params := make([]string, len(p.Binders))
	for i, b := range p.Binders {
		params[i] = b.Name + ": " + b.Type.String()
	}
	return strings.Join(params, ", ")
}

// forallCaller returns the call for a forall property: its entry in the
// file's batch, or, when the batch is unavailable (D1), a compile of this
// property alone so a property that does not type-check fails by itself.
// One evaluator serves every case of the property: entries are pure.
func (e *Executor) forallCaller(p *ast.Property) (propertyCall, error) {
	if b := e.batch; b != nil {
		if ent, ok := b.props[p]; ok {
			return e.callerFor(ent, b.lineMap, b.modules, b.root), nil
		}
	}
	if e.sourceFile == nil {
		return nil, fmt.Errorf("source file not set on executor")
	}
	src, err := os.ReadFile(e.modulePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source file: %w", err)
	}
	base, lineMap := e.stripTestBlocks(string(src), e.sourceFile)
	es := newEntrySource(base)
	ent := es.add(namedBatchPrefix+"prop_0", binderParams(p), PrintAILANGSource(p.Expr), p.Pos.Line)
	ent.label = "property"
	for _, d := range e.sourceFile.Decls {
		if pd, ok := d.(*ast.PropertyDecl); ok && pd.Property == p {
			ent.title = pd.Name
			break
		}
	}
	res, err := e.runNamedTestPipeline(es.String(), e.sourceFile.Module != nil)
	if err != nil {
		if e.batchFailure != nil {
			e.perBodyCompileFailures++
		}
		// The compile error names the private temp file; point it at the
		// user's file and the property's own line instead (D5).
		return nil, &mappedError{msg: e.mapEntryCompileError(err.Error(), ent, lineMap), err: err}
	}
	e.cacheModules(&res)
	return e.callerFor(ent, lineMap, e.modules, e.rootModule), nil
}

// callerFor binds the entry on a fresh harness evaluator over the modules of
// the compile that produced it.
func (e *Executor) callerFor(ent namedEntry, lineMap []int, modules map[string]*loader.LoadedModule, root string) propertyCall {
	e.modules, e.rootModule = modules, root
	ev := e.newHarnessEvaluator()
	fn, ok := ev.Env().Get(e.rootModule + "." + ent.name)
	return func(args []eval.Value) (eval.Value, error) {
		if !ok {
			return nil, fmt.Errorf("property entry %s is not bound in the test harness", ent.name)
		}
		v, err := ev.CallValueN(fn, args)
		if err != nil {
			return nil, &mappedError{msg: e.mapEntryCompileError(err.Error(), ent, lineMap), err: err}
		}
		return v, nil
	}
}
