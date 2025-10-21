Great question. Short version: you switch to writing parts in .ail when correctness, velocity, and AI-first introspection matter more than raw speed—and you’ve got effects + patterns + a small stdlib. Keep the hot loops in Go.

Here’s a crisp way to think about it.

When .ail beats Go (even if Go is faster)
	1.	Correctness & guarantees
	•	ADTs + pattern matching make illegal states unrepresentable.
	•	Effect types prevent “accidental IO” and make refactors safer.
	•	Easier to prove purity and enable memoization/parallelism later.
	2.	Developer velocity & community
	•	One language to extend: contributors don’t need Go to add rules, transforms, or lints.
	•	Golden tests + deterministic IR printing + SIDs/ledger give great dev UX in .ail.
	3.	AI-first ergonomics
	•	Your structured errors, SIDs, decision ledger, and effects inspector work on .ail code itself, making agent loops richer (explanations, auto-fixes).
	4.	Portability & embedding
	•	.ail logic runs anywhere your runtime runs—no CGO, fewer cross-compilation headaches.

When to start (milestones)
	•	After M-P3 (pattern matching) + M-P4 (effect types) + basic stdlib (M-P5).
	•	That’s roughly your v0.2.x window. At v0.1.0 you’re still stabilizing the core.

What to move first (safe wins)

Think 3-tier split:
	•	Tier A (stay in Go – performance kernels):
	•	Lexer, parser, tight evaluator primitives, hashing/digests, BLAKE3/crypto, tight graph/SCC, persistent data structure kernels.
	•	Expose as $builtin/FFI to .ail.
	•	Tier B (move to .ail – high leverage, not hot):
	•	Rule engines & passes: operator/lowering tables, rewrite rules, pretty-printers, code formatters.
	•	Error suggesters (IMP/PAR/ELB fix hints), diagnostics shaping.
	•	Test runner orchestration, snapshot reporters, effect inspection renderers.
	•	Import resolution policies (not the filesystem calls—just the policy logic).
	•	Planning/scaffolding validators (:propose plan.json checks).
	•	Tier C (mixed – .ail orchestrates, Go provides intrinsics):
	•	Module linking orchestration, interface filtering, name resolution strategies.
	•	Constant folding, simple dead-code elimination (in .ail), but keep number/string ops as Go intrinsics.

A practical decision checklist

For each candidate module, switch to .ail if ≥3 of these are true:
	•	Not on the critical path for wall-time (≤1–5% of runtime budget).
	•	Benefits from ADTs/patterns or effect typing (i.e., rich invariants).
	•	Has lots of rules/tables/config you want contributors to edit.
	•	You want superb explainability (SIDs, :why, decisions).
	•	You expect frequent iteration by non-Go contributors.

Keep in Go if:
	•	It’s O(n) over large ASTs/run-time data with tight inner loops.
	•	It’s heavy I/O/CPU or depends on existing high-perf libs.
	•	It’s security-sensitive low-level code (hashing, fs watchers, symlink resolution).

Performance escape hatches
	•	FFI $builtin: expose hot Go functions to .ail.
	•	Specialization: keep high-level orchestration in .ail, call monomorphic builtins for tight work.
	•	Later: add an AOT or LLVM/Go-gen backend to compile .ail hot paths—then you can move more over without losing perf.

Suggested roadmap to “start using .ail”
	1.	v0.1.0 – all Go (what you’re doing now). Stabilize pipeline + parsing + imports + lowering.
	2.	v0.2.x – introduce .ail sidecar packages:
	•	Diagnostic suggesters, pretty-printer, simple rewriters, test runner orchestration.
	•	Keep FFI for $builtin operations and file/OS access.
	3.	v0.3.x – move more passes:
	•	Import/link policy logic, constant folding, dead-code elimination, formatting rules.
	4.	v0.4.x – selective self-hosting:
	•	Parts of elaboration/type-driven rewrites in .ail, with perf-critical unification still in Go.
	5.	v1.0 – optimize:
	•	Consider AOT compilation for .ail passes; shrink Go surface to a fast runtime + intrinsics.

Concrete examples to try first in .ail
	•	Fix suggester rules (PAR/IMP/ELB) with golden tests.
	•	Operator/rewrite tables (your OpLowering mapping as data + pattern rewrites).
	•	Test runner orchestration + reporting JSON.
	•	Pretty-printer/formatter (deterministic, rule-driven).

These are high impact, low risk, and showcase the language.

⸻

Bottom line:
Use Go for speed-critical kernels and as the stable substrate; use .ail for correctness-heavy, rule-driven, and user-visible logic where your type/effect system and AI-first tooling shine. Start right after v0.1.0—once M-P3/4/5 land—by migrating policy/diagnostic layers, not the hot kernels.