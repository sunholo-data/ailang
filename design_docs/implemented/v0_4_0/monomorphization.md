🧠 What Monomorphization is — in human-AI terms

Monomorphization = turning polymorphic functions into specialized, concrete versions when their argument types are known.

In AILANG today:

let max = \x. \y. if x > y then x else y

has the type
Ord a => a -> a -> a.

That’s polymorphic — it can work on Int, Float, String, etc.
But your compiler currently produces a single Core function with unresolved type variables (α1, α2), and lowering can’t decide which builtin (gt_Int, gt_Float, …) to call.

Monomorphization solves this by:
	1.	Detecting that max is called with concrete types (e.g. Int).
	2.	Cloning its Core body.
	3.	Substituting a = Int.
	4.	Re-running operator lowering on the clone.
	5.	Caching that specialized function so repeated calls reuse it.

The result:
max(3)(4) calls _max_Int,
max(3.14)(2.71) calls _max_Float.
No runtime dispatch, no Intrinsics left.
It’s what Rust, MLton, and OCaml’s flambda all do internally.

⸻

Now here’s the draft design document for AILANG:

⸻

M-POLY-A: Monomorphization Pass (v0.4.0)

Status: Planned
Priority: P0 (architectural)
Estimated: 2–3 working days
Depends on: M-DX4 (CoreTypeInfo), M-DX2 (Type-Guided Lowering)

⸻

1. Problem Statement

AILANG’s Hindley-Milner type inference produces polymorphic Core functions.
However, the evaluator only executes monomorphic builtins.
Currently, operator lowering runs before functions are specialized, leaving polymorphic Intrinsics unlowered inside lambda bodies.

Example:

let max = \x. \y. if x > y then x else y
max(10)(5)     -- runtime panic: Intrinsic not lowered

The body x > y cannot be lowered, because x and y have type variables (α1, α2) during compilation.

⸻

2. Goals
	•	Primary: Specialize polymorphic functions at call-sites with known concrete types.
	•	Secondary: Re-run operator lowering on specialized bodies so Intrinsics always disappear before evaluation.
	•	Tertiary: Cache and reuse specializations to avoid code bloat.

Success metric:
✔ \x. \y. x > y works automatically for Int, Float, and String arguments.
✔ No Intrinsics remain post-lowering.
✔ No runtime “binop shim” needed.

⸻

3. Design Overview

3.1 Conceptual Flow
	1.	Inference produces polymorphic Core with type schemes and a CoreTypeInfo map.
	2.	Specialization Pass runs after type checking, before lowering:
	•	Detect each call site (f args) where f refers to a polymorphic definition and all argument types have concrete heads.
	•	Clone f’s Core body, substituting type variables with concrete types.
	•	Run lowering on the clone (using CoreTypeInfo + substitution).
	•	Register the new symbol _f$Int$Int in the current module scope.
	•	Rewrite the call to point to the specialized function.
	3.	Lowering & Evaluation then proceed as normal.

3.2 Architecture Diagram

Parse → Elaborate → TypeCheck → [Specialize] → Lower → Eval

3.3 Naming Convention

Polymorphic	Specialized	Description
max	_max$Int$Int	Two Int arguments
max	_max$Float$Float	Two Float arguments
id	_id$String	Single String argument


⸻

4. Implementation Plan

Phase 1: Infrastructure (4–5h)
	•	File: internal/pipeline/specialize.go
	•	Create Specializer struct with:

type Specializer struct {
    CoreTI *types.CoreTypeInfo
    Cache  map[string]*core.Func // (fnID + typeHeads) → specialized Core
}


	•	Implement typeHeads([]types.Type) []types.Head
	•	Implement substituteTypevars(fn *core.Func, subst map[types.TypeVar]types.Type) *core.Func

Phase 2: Specialization Pass (4h)
	•	Walk Core AST:
	•	For each call node (App f args):
	•	If f is polymorphic and args all have concrete heads:
	•	Generate key (f.ID, heads)
	•	Lookup or create specialized version
	•	Replace (App f args) with (App f_spec args)
	•	Apply lowerer.Run() on new bodies

Phase 3: Caching & Hygiene (2h)
	•	Maintain a per-module specialization cache
	•	Ensure unique names (_fn$Int$Float, etc.)
	•	Avoid recursive infinite specialization (detect by in-progress set)

Phase 4: Testing (3–4h)
	•	examples/snippets/showcase/lambdas_polymorphic.ail
	•	max, min, abs, cmp, etc. work for Int, Float, String
	•	Unit tests for specialization cache
	•	Test mixed literals (Int+Float) defaulting

Phase 5: Docs (2h)
	•	docs/architecture/monomorphization.md
	•	Update CLAUDE.md “Critical Principles”:
	•	Add “All Intrinsics lowered before eval” invariant
	•	Update LIMITATIONS.md: remove “polymorphic lambda” limitation

⸻

5. Example Walkthrough

Input:

let max = \x. \y. if x > y then x else y
let result = max(3.14)(2.71)

1️⃣ Type inference:

max : Ord a => a -> a -> a
result : Float

2️⃣ Specialization detects call max(3.14)(2.71) → a = Float.
Generates _max$Float$Float:

let _max$Float$Float = \x: Float. \y: Float. if x > y then x else y

3️⃣ Runs lowering:

Intrinsic(>) → App(Var "_gt_Float", [x, y])

4️⃣ Rewrites call:

let result = _max$Float$Float(3.14)(2.71)

5️⃣ Evaluator executes normally. ✅

⸻

6. Risks & Mitigations

Risk	Impact	Mitigation
Infinite specialization recursion	Medium	Track active (fn, types) pairs
Code bloat for many instantiations	Medium	Cache and deduplicate
Complex substitution chains	Low	Use existing ApplySubstitution()
Interaction with effects	Low	Effects preserved; no change
Cross-module specialization	Low	For v0.4.0, specialize only within module


⸻

7. Non-Goals (v0.4.0)
	•	No user-visible typeclass or instance syntax.
	•	No higher-rank or impredicative polymorphism.
	•	No dictionary passing (planned for v0.5.x).
	•	No partial specialization across recursive groups.

⸻

8. Deliverables

Deliverable	Description
internal/pipeline/specialize.go	Core specialization pass
internal/pipeline/specialize_test.go	100% coverage
docs/architecture/monomorphization.md	Architecture doc
examples/snippets/showcase/lambdas_polymorphic.ail	Demonstration file
CHANGELOG.md	v0.4.0 “Monomorphization Pass” entry


⸻

9. Timeline

Day	Task
1	Implement Specializer, clone/subst helpers
2	Integrate pass + cache, initial tests
3	Full regression suite, docs, examples
4 (optional)	Cross-module cache + metrics


⸻

10. Expected Impact

✅ 100 % of polymorphic lambda examples compile and run
✅ All Intrinsics eliminated before eval
✅ Zero runtime type errors from operator polymorphism
✅ Demonstrably faster AI DX (no manual annotations)
✅ Paves the way for typeclass dictionaries (v0.5.0)

