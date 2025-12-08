You’re right to ask for a single, objective stack-rank that isn’t swayed by the last thing we discussed. Here’s the one true priority list that balances user value, AI-first goals, technical risk, and dependencies—grounded in what’s actually shipped vs. aspirational across your docs.

⸻

AILANG — Unified Priority Roadmap (Realistic, Dependency-Aware)

Tier 0 — Trust & Ground Truth (must do first)
	1.	Doc/Reality Sync + Example Gate
	•	Remove/flag non-working examples; add CI that fails on any example not runnable by ailang run.
	•	README “What works / What fails” table stays and is CI-generated.
	•	Exit: make verify-examples green; README regenerated in CI.
	2.	Schema & Compact Mode Baseline (already largely done)
	•	Freeze ailang.error/v1, ailang.test/v1, ailang.effects/v1 with Accepts() prefix matching.
	•	Deterministic JSON (sorted keys) + --compact respected everywhere.
	•	Exit: Golden fixtures pass; all CLIs produce valid schema.

Why Tier 0 first: Without truth and deterministic IO, everything else (including AI agents) is brittle.

⸻

Tier 1 — Core Language Usability (unlock “serious work”)
	3.	File Parser + Minimal Module Loader + func Declarations (v3.3)
	•	Parse .ail modules; func desugars to lambda + let/letrec.
	•	Imports: std/prelude and ./relative only; topological load; cycle error LDR001/2.
	•	CLI: ailang run <file.ail>, ailang check <file.ail>.
	•	Exit:
	•	examples/v3_3/hello.ail → “Hello AILANG!”
	•	examples/v3_3/math.ail (recursive fact) → 120
	•	examples/v3_3/imports.ail → 720
	•	Any parse/module/elab/type error emits ailang.error/v1 with SID.
	4.	Unify REPL + File Pipelines
	•	The same elaboration/typecheck/eval path for both (no “REPL special case”).
	•	Exit: :dump-core / :effects identical for a snippet in REPL vs. the same snippet in a file.

Why Tier 1 now: This turns AILANG from a REPL toy into a usable language. Also removes split-brain between REPL/file.

⸻

Tier 2 — AI-First Loop (close the reactive loop end-to-end)
	5.	Error-as-Values Everywhere
	•	Replace ad-hoc Go errors across parse → module → typecheck → elaborate → link → eval with the JSON encoder (TC###/ELB###/LNK###/RT###).
	•	Always include fix.suggestion + confidence and best-effort sid.
	•	Exit: Grep shows all user-visible errors created via encoder; 3 golden fixtures per phase.
	6.	Test Runner + :test --json (files)
	•	Discover inline tests [...] blocks in functions (or, if not parsed yet, support file-level test "name" { … } minimal runner).
	•	Deterministic order; full counts; run_id; platform info.
	•	Exit: Empty suite still yields valid ailang.test/v1. CI runs ailang :test --json on examples/tests/.
	7.	Effects Inspector (file + REPL, real types)
	•	Parse + infer only (no eval); display inferred type and projected effect row (even if effects engine is partial).
	•	Include a decisions slice when defaulting occurs (ties into ledger later).
	•	Exit: :effects <expr> returns {type, effects, decisions}; golden fixtures for numeric defaulting messages.

Why Tier 2 now: This gives AI agents a reliable reactive loop: precise errors, test outcomes, and pre-execution inspection.

⸻

Tier 3 — Context-Drift Protection (persistent “why”)
	8.	Stable Node IDs (SIDs) across files
	•	Derive from hash(path | span byte offsets | ast_kind | child_path).
	•	Maintain Surface→Core mapping through desugar.
	•	Exit: Same code → same SID on reruns; :trace-slice <sid> shows surface→core journey for top-level decls.
	9.	Decision Ledger (MVP) + :why
	•	Log defaulting, instance selection, and normalization decisions with inputs/outputs.
	•	:why shows last N decisions; output schema ailang.decisions/v1.
	•	Exit: Golden decisions file stable under CI; privacy redaction for large payloads.

Why Tier 3 here: With files working and errors/tests structured, the ledger keeps AI and humans anchored across sessions.

⸻

Tier 4 — Proactive Planning (the “next-level” loop)
	10.	Planning & Scaffolding Protocol

	•	:propose plan.json (schema plan/v1) validates names, signatures, effect plausibility, dependency reachability.
	•	:scaffold --from-plan generates idempotent stubs with plan SID in header.
	•	Exit: Plan validation report golden; scaffolding overwrites only marked regions or warns cleanly.

Why Tier 4 after ledger: The plan validator benefits from SIDs, schemas, and consistent pipelines.

⸻

Tier 5 — Language Power-ups (after foundation is solid)
	11.	Pattern Matching (phased)

	•	Phase 1: literals/tuples; Phase 2: lists; Phase 3: records/variants; Phase 4: guards + basic exhaustiveness.
	•	Exit: Deterministic case order; warning JSON hints for non-exhaustive matches.

	12.	Effect System Runtime (phased)

	•	Admit effect rows in types now; gradually add capability tokens and a small IO/FS kernel.
	•	Exit: :effects agrees with runtime capability checks for first effects.

	13.	Lockfiles + Snapshots

	•	Lockfile to pin instance sets and stdlib version; :snapshot / :replay wired to loader + ledger.
	•	Exit: Replaying a build reproduces identical outputs and SIDs.

⸻

Cutline (if time is tight)

If you can only do a subset, ship in this exact order:
	1.	Tier 1.3–1.4 (parser + func + imports + pipeline unification)
	2.	Tier 2.5 (error JSON everywhere)
	3.	Tier 2.6 (test runner --json)
	4.	Tier 3.8 (SIDs + trace-slice)

Everything else can follow.

⸻

What NOT to do right now
	•	Don’t start CSP/session types, quasiquotes, or a package manager before Tier 1–3 are done.
	•	Don’t add more examples that rely on unsupported features; every new example must be CI-validated.
	•	Don’t maintain two evaluators; unify and delete special-case code.

⸻

“Why this order?” in one paragraph

This order makes AILANG useful (files + func), trustworthy (one pipeline + structured, deterministic IO), and AI-friendly (introspection, tests, SIDs, ledger) before we expand language surface area. It prevents context drift and rework, while giving you quick, visible wins for users and agents.

⸻

Near-term sprint (2–3 weeks of focused work)
	•	Week 1: Parser for top-level func/let/type alias, desugar to let/letrec, minimal loader (std/prelude, ./rel), new golden examples; wire file runner; delete REPL-only shortcuts.
	•	Week 2: Error-as-values throughout; :test --json; effects inspector hooked to the real type inference (no eval).
	•	Week 3: SIDs + surface→core map; :trace-slice + :why MVP; CI: example gate + JSON schema validation + compact mode diffs.

⸻

Acceptance gates per tier (succinct)
	•	T1: run/check on files; recursion works; identical REPL/file behavior.
	•	T2: all user-visible errors use ailang.error/v1; :test --json deterministic; :effects returns {type,effects,decisions}.
	•	T3: SIDs stable; :trace-slice and :why operational with goldens.
	•	T4: Plans validate; scaffolds idempotent with plan SID header.
