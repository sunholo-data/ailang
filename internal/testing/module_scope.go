package testing

// Module-scoped environments for the test harness
// (design: m-test-harness-module-scoped-envs; #1461, #1516).
//
// `ailang run` evaluates each module in its own environment (link.Resolver),
// so a module's private helpers are visible only inside that module. The test
// harness evaluates test bodies, inline/cluster harnesses and contract
// harnesses by injecting every loaded module's functions into one evaluator.
// It used to bind every function under its BARE name in that one shared
// environment, so the module that sorted last silently replaced every
// same-named function of every other module — a private `helper` in another
// imported module (#1461), or a private `isErr` in the module under test
// replaced by std/result's export (#1516).
//
// Scoping rule (one rule for all four harness paths):
//
//   - Each loaded module gets its own child environment of the shared env.
//     Its functions are bound there under their bare names, and their closures
//     capture it, so intra-module references (bare core.Var) resolve within
//     the module.
//   - Cross-module references are module-qualified (core.VarGlobal) and
//     resolve through "<module>.<name>" keys in the shared env, via
//     CombinedResolver.
//   - Only the module under test ("root") also exposes its bare names in the
//     shared env, because test bodies and harness expressions are elaborated
//     in the root module's scope and reference its functions by bare name.

import (
	"sort"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/runtime"
)

// cacheModules records the modules of a pipeline run for later harness
// evaluation, and identifies the root module (the module under test) by
// identity: the pipeline's root Core program is the same pointer as the root
// module's LoadedModule.Core. Comparing paths would be fragile — the root may
// be a harness temp file, a synthetic `_test/<base>` module or a relaxed
// module whose declared path differs from its canonical one.
func (e *Executor) cacheModules(res *pipeline.Result) {
	e.modules = res.Modules
	e.rootModule = ""
	if res.Artifacts.Core == nil {
		return
	}
	for path, mod := range res.Modules {
		if mod != nil && mod.Core == res.Artifacts.Core {
			e.rootModule = path
			return
		}
	}
}

// newHarnessEvaluator builds the evaluator every harness path uses: module
// bindings injected with module scoping, a CombinedResolver over them, and the
// source file's ADT constructors.
func (e *Executor) newHarnessEvaluator() *eval.CoreEvaluator {
	evaluator := e.newEvaluator()
	builtins := runtime.NewBuiltinRegistry(evaluator)
	env := evaluator.Env()
	own := e.injectModuleBindings(env, builtins)
	evaluator.SetGlobalResolver(&CombinedResolver{
		Builtins:       builtins,
		Env:            env,
		Modules:        e.modules,
		ModuleBindings: own,
	})
	e.injectADTConstructors(evaluator)
	return evaluator
}

// moduleScope is one module's environment during injection.
type moduleScope struct {
	path   string
	env    *eval.Environment     // child of the shared env; bare names live here
	own    map[string]eval.Value // the module's own bindings, without parents
	isRoot bool
}

// bind binds name in the module's own scope, under its qualified key in the
// shared env, and — for the root module only — under its bare name in the
// shared env.
func (s *moduleScope) bind(shared *eval.Environment, name string, v eval.Value) {
	s.env.Set(name, v)
	s.own[name] = v
	if s.path != "" {
		shared.Set(s.path+"."+name, v)
	}
	if s.isRoot {
		shared.Set(name, v)
	}
}

// injectModuleBindings binds every loaded module's functions for harness
// evaluation, scoped per module (see the file comment). It returns each
// module's own bindings (module path → bare name → value) for
// CombinedResolver's module-scoped fallback.
//
// Non-recursive lambdas (core.Let) close over their module's environment.
// Recursive groups (core.LetRec) get a group environment of IndirectValue
// cells under the module environment, as evalCoreLetRec builds, so self and
// mutual recursion resolve to the group's own functions.
// Re-exports (`let concat = VarGlobal{std/list, concat}`) are resolved after
// every module's functions are bound, because module order is lexical, not
// dependency order.
func (e *Executor) injectModuleBindings(env *eval.Environment, builtins *runtime.BuiltinRegistry) map[string]map[string]eval.Value {
	own := make(map[string]map[string]eval.Value, len(e.modules))
	if len(e.modules) == 0 {
		return own
	}

	type reexport struct {
		scope *moduleScope
		name  string
		ref   core.GlobalRef
	}
	var reexports []reexport

	// Deterministic module order regardless of Go's map iteration order.
	sortedPaths := make([]string, 0, len(e.modules))
	for modPath := range e.modules {
		sortedPaths = append(sortedPaths, modPath)
	}
	sort.Strings(sortedPaths)

	for _, modulePath := range sortedPaths {
		mod := e.modules[modulePath]
		if mod == nil || mod.Core == nil {
			continue
		}
		scope := &moduleScope{
			path:   modulePath,
			env:    env.NewChildEnvironment(),
			own:    make(map[string]eval.Value),
			isRoot: modulePath == e.rootModule,
		}
		own[modulePath] = scope.own

		for _, decl := range mod.Core.Decls {
			switch d := decl.(type) {
			case *core.Let:
				if lambda, ok := d.Value.(*core.Lambda); ok {
					scope.bind(env, d.Name, &eval.FunctionValue{
						Params: lambda.Params,
						Body:   lambda.Body,
						Env:    scope.env,
						Typed:  true,
					})
				} else if vg, ok := d.Value.(*core.VarGlobal); ok {
					reexports = append(reexports, reexport{scope: scope, name: d.Name, ref: vg.Ref})
				}

			case *core.LetRec:
				recEnv := scope.env.NewChildEnvironment()
				cells := make(map[string]*eval.RefCell, len(d.Bindings))
				for _, binding := range d.Bindings {
					cell := &eval.RefCell{}
					cells[binding.Name] = cell
					recEnv.Set(binding.Name, &eval.IndirectValue{Cell: cell})
				}
				for _, binding := range d.Bindings {
					lambda, ok := binding.Value.(*core.Lambda)
					if !ok {
						continue
					}
					funcVal := &eval.FunctionValue{
						Params: lambda.Params,
						Body:   lambda.Body,
						Env:    recEnv, // the group's own env, for self/mutual recursion
						Typed:  true,
					}
					cells[binding.Name].Val = funcVal
					cells[binding.Name].Init = true
					scope.bind(env, binding.Name, funcVal)
				}
			}
		}
	}

	// Resolve re-exports to a fixpoint: a re-export of a re-export becomes
	// resolvable once the inner one is bound. A target that never resolves
	// stays unbound, and a use of it fails loudly in CombinedResolver.
	lookup := func(ref core.GlobalRef) (eval.Value, bool) {
		if ref.Module == "$builtin" {
			return builtins.Get(ref.Name)
		}
		if v, ok := env.Get(ref.Module + "." + ref.Name); ok {
			return v, true
		}
		v, ok := own[ref.Module][ref.Name]
		return v, ok
	}
	for len(reexports) > 0 {
		var pending []reexport
		for _, re := range reexports {
			if v, ok := lookup(re.ref); ok && v != nil {
				re.scope.bind(env, re.name, v)
			} else {
				pending = append(pending, re)
			}
		}
		if len(pending) == len(reexports) {
			break
		}
		reexports = pending
	}
	return own
}
