# M-TAINT-TYPES — Sprint Plan

**Source design doc**: [m-taint-types.md](m-taint-types.md) (v3, design freeze closed 2026-04-30)
**Target version**: v0.16.0
**Priority**: **P1 — Strategic** (no downstream blocker, but the headline AI-safety differentiator vs Python; motivated by Erik Meijer's *Guardians of the Agents*, CACM Jan 2026)
**Estimated**: ~62 hours across 9 milestones in 3 phases (~3 sprints / 8–10 working days)
**Risk**: medium-high — extends the type system in a non-trivial way; mitigated by strict TDD and the locked design freeze

## Why now

Design freeze closed today. Hard prerequisite (M-SMT-CROSS-MODULE-TYPES) shipped in v0.14.3. Tier 1 inbox-injection demo already exists at `examples/runnable/contracts/inbox_injection.ail` and provides the no-regression target.

**Strategic context**: this is the milestone that converts AILANG's Z3 verification story (already "half of guardians") into a complete IFC type system with capability-gated declassification. The narrative win is "Erik Meijer's paper says agent workflows need taint + automata + Z3; AILANG ships taint and Z3 as language features, with capability-discipline doing the audit work."

## Locked design decisions (from design freeze)

| Decision | Locked value |
|---|---|
| Label syntax | `T<label>` (sugar) + `T{not LABEL}` (refinement, ASCII `not` keyword) |
| Lattice elements | Free-form label constants; naming convention `lowercase-kebab` taught in prompt |
| Declassification | No special primitive; a function whose effect row contains `Declassify` may change its input's label |
| `Declassify` effect | Sibling effect (alongside `Net`, `FS`, `IO`); not a `[mode=…]` parameter |
| Refinement grammar (MVP) | `not IDENT` only; richer predicates deferred |
| Default label | Missing = `⊥` (untainted) |

## Recent velocity (calibration)

Last 14 days saw five sprints land cleanly: M-CI-BUILD-SPEED (M2-M6), M-SUPPLY-CHAIN-HARDENING-2 (M1-M5), M-MCP-EDGE-THROTTLE Path A, M-AGENT-MCP M4, **M-SMT-CROSS-MODULE-TYPES (~22h estimated, executed cleanly with sprint-executor in ~3h actual run-time, all 5 milestones)**. Throughput is healthy. Type-system extensions historically run 1.5–2× parser/codegen sprints; the design-doc estimate of 60–70h is realistic and will be respected here, with implicit 20% buffer absorbed into per-milestone estimates.

## Milestones

Nine milestones organised by phase. Front-load lattice + AST (M1+M2) since every later milestone depends on them. Phase 1 ends with the rewritten Tier 1 demo as the headline acceptance check. Pause points after M5 and M7 for human spot-check.

---

### Phase 1: Single-module type-level checking (~28h, 5 milestones)

#### M1 — Label lattice
**Estimated**: 6 hours
**Dependencies**: none (foundation)
**Files**:
- `internal/types/labels.go` (NEW, ~280 LOC)
- `internal/types/labels_test.go` (NEW, ~150 LOC)

**Implementation outline**:
1. Define `Label` type: `⊥`, `Const(name)`, `Var(α)`, `Join(L1, L2)` (interface + four implementations)
2. `Join(a, b)` constructor with simplification: `⊥ ⊔ L = L`, `L ⊔ L = L` (idempotence), commutativity (canonical sort by name for `Const` pairs)
3. `Equal(a, b)` checks structural equality after simplification
4. `Subsumes(L, ℓ)` — does label `ℓ` appear anywhere in `L`'s join expression? Used by sink check: `ℓ ⊆ L` is true iff `ℓ` is in the resolved join
5. Refinement evaluation: `EvalNot(L, ℓ)` — true iff `Subsumes(L, ℓ) == false`
6. `String()` for debug/error messages: `⊥`, `<email>`, `<email> ⊔ <user>` (sorted)

**Acceptance criteria**:
- [ ] `⊥ ⊔ <email> == <email>` (identity)
- [ ] `<email> ⊔ <email> == <email>` (idempotence)
- [ ] `<email> ⊔ <user>` and `<user> ⊔ <email>` are `Equal` (commutativity, canonicalised)
- [ ] `((<a> ⊔ <b>) ⊔ <c>) Equal (<a> ⊔ (<b> ⊔ <c>))` (associativity, canonicalised)
- [ ] `EvalNot(⊥, <email>) == true`, `EvalNot(<email>, <email>) == false`, `EvalNot(<email> ⊔ <user>, <email>) == false`
- [ ] All existing AILANG tests pass (no integration yet, just adding new package)
- [ ] `make lint` clean

**Test commands**:
```bash
go test ./internal/types/ -run "TestLabel" -count=20  # -count=20 to catch map-iteration nondeterminism in canonical sort
make lint
```

**Pause point**: none — small foundation milestone; roll into M2.

---

#### M2 — AST + parser for label syntax
**Estimated**: 6 hours
**Dependencies**: M1
**Files**:
- `internal/ast/types.go` (MODIFIED, ~40 LOC delta) — every type node carries optional `Label`
- `internal/parser/parser_types.go` (MODIFIED, ~80 LOC delta) — parse `T<label>` and `T{not IDENT}`
- `internal/parser/parser_types_label_test.go` (NEW, ~120 LOC)

**Implementation outline**:
1. Extend `ast.SimpleType`, `ast.TypeApp`, `ast.RecordType`, `ast.ListType`, etc. with optional `Label *ast.LabelExpr` and `Refinement *ast.RefinementExpr` fields
2. New AST nodes: `ast.LabelExpr` (constant name) and `ast.RefinementExpr` (currently only the `not IDENT` form)
3. Parser: after parsing a type, peek for `<` (label sugar) or `{` (refinement). For `<`, parse identifier, expect `>`. For `{`, expect `not`, parse identifier, expect `}`. Hard error on anything else (no richer grammar in MVP).
4. Parser must NOT consume `<` if it's followed by something that would start an expression (cheap lookahead: `<` followed by IDENT followed by `>` is a label; otherwise back off — though in type position `<` should be unambiguous)
5. Pretty-printer round-trips both syntaxes

**Acceptance criteria**:
- [ ] `func f(x: string<email>) -> string<user>` parses with both labels recovered
- [ ] `func g(x: string{not email}) -> string` parses with refinement on parameter type
- [ ] `string{not email}` rejects: `string{!email}`, `string{¬email}`, `string{not email && not user}`, `string{label = email}` — all four produce structured parse errors with hint pointing to the MVP grammar
- [ ] Pretty-printer emits canonical form: `T<label>` and `T{not LABEL}` (not e.g. `T<  label>`)
- [ ] Existing parser tests still pass (no regressions on unlabelled syntax)
- [ ] `make lint` clean

**Test commands**:
```bash
go test ./internal/parser/ -count=1
go test ./internal/ast/ -count=1
make verify-examples  # confirm no existing example breaks
```

**Pause point**: after M2, all parsing infrastructure exists. Type checker still ignores labels.

---

#### M3 — Inference: pure-op label propagation
**Estimated**: 7 hours
**Dependencies**: M1, M2
**Files**:
- `internal/types/typechecker.go` (MODIFIED, ~120 LOC delta) — every pure operation joins input labels into output
- `internal/types/inference_label_test.go` (NEW, ~180 LOC)

**Implementation outline**:
1. Threading: every `types.Type` carries the label it was inferred at. Add `Label() Label` method to the type interface, default `⊥`.
2. (VAR): variable lookup returns the type's stored label
3. (APP-PURE): for a function call `f(arg)`, infer the argument's label `L_a`. Output type's label is `L₂ ⊔ L_a`. (DECLASS rule applies later in M4 for non-pure functions.)
4. (JOIN) for binary operations, list construction, record construction, match arms, `let`-bindings: output label is the join of all contributing input labels
5. Identity preservation: `pure func id(x) { x }` infers as `∀L. T<L> -> T<L>`, NOT `T -> T`
6. Concrete inference test: `summarize(rawEmail.body)` where `rawEmail: Email<email>` and `summarize: ∀L. string<L> -> string<L>` produces `string<email>`

**Acceptance criteria**:
- [ ] `pure func id(x) { x }` inferred as `∀L. T<L> -> T<L>` (verified by inspecting type checker output for the inferred scheme)
- [ ] `let x = email_value in summarize(x)` propagates `<email>` through the let-binding into the result
- [ ] `concat(s1, s2)` where `s1: string<a>` and `s2: string<b>` produces `string<a ⊔ b>`
- [ ] Match arm join: `match v { Some(x) => x, None => default }` where one arm has `<email>` and the other `⊥` produces `<email>` for the result
- [ ] All existing typechecker tests pass (no regressions on unlabelled code; missing label = `⊥`)
- [ ] `make lint` clean

**Test commands**:
```bash
go test ./internal/types/ -count=1
ailang run examples/runnable/contracts/cross_function.ail  # smoke test: existing examples unaffected
```

**Pause point**: none — fold into M4, since sink check + DECLASS need the inference output.

---

#### M4 — Sink check + DECLASS rule
**Estimated**: 5 hours
**Dependencies**: M3
**Files**:
- `internal/types/sink_check.go` (NEW, ~180 LOC) — post-inference refinement check
- `internal/types/declassify.go` (NEW, ~80 LOC) — DECLASS typing rule + `Declassify` capability registration
- `internal/effects/registry.go` (MODIFIED, ~30 LOC delta) — register `Declassify` as a sibling effect
- `internal/types/sink_check_test.go` (NEW, ~140 LOC)

**Implementation outline**:
1. Sink check pass: at every function application, walk parameter refinements. For each `T{not ℓ}` parameter, check that the argument's resolved label does NOT subsume `ℓ` (using M1's `Subsumes` / `EvalNot`). Failure → structured error carrying source label, sink, and binding chain.
2. (DECLASS) typing rule: when typing a function declaration, check whether the input/output types differ in their label. If so, the function's effect row MUST contain `Declassify`. Otherwise: type error "this function changes a value's label between input and output without `! {Declassify}` in its effect row".
3. Register `Declassify` as a sibling effect alongside `Net`, `FS`, `IO`, `IO`, etc. Capability discipline kicks in for free at runtime (existing effect-row enforcement).
4. Error message format for sink violation:
   ```
   error: value of type string<email> reaches sink expecting string{not email}
     source: rawEmail.body          (declared at email/types.ail:23)
       through: summarize           (label preserved by pure function)
       through: let safeSummary     (label preserved by let-binding)
     sink:   sendEmail.body         (declared at mail/send.ail:12)
   ```

**Acceptance criteria**:
- [ ] `string<email>` reaching `string{not email}` produces a structured sink-violation error with source / through-chain / sink fields populated
- [ ] A function with input `string<email>` and output `string<sanitized>` and effect row `! {}` produces a DECLASS violation with hint "add `Declassify` to the effect row"
- [ ] Same function with effect row `! {Declassify}` typechecks
- [ ] An `id` function (input and output share the same label variable) typechecks WITHOUT requiring `Declassify` (no label change)
- [ ] All existing tests pass (functions without labels are all `⊥`-to-`⊥`; no DECLASS triggers)
- [ ] `make lint` clean

**Test commands**:
```bash
go test ./internal/types/ -count=1
go test ./internal/effects/ -count=1
make verify-examples  # smoke test
```

**Pause point**: none — fold into M5, since the demo is the headline acceptance.

---

#### M5 — Single-module demo: inbox_injection_v2.ail
**Estimated**: 4 hours
**Dependencies**: M4
**Files**:
- `examples/runnable/contracts/inbox_injection_v2.ail` (NEW, ~100 LOC)
- `examples/manifest.json` (MODIFIED) — register the new example

**Implementation outline**:
1. Rewrite the Tier 1 demo using real labels (per the design doc Example 1):
   - `fetchMail: () -> [Email<email>] ! {FS}` (source)
   - `summarize: ∀L. string<L> -> string<L>` (label-preserving pure transform)
   - `sendEmail: (string, string{not email}) -> () ! {Net}` (sink)
   - `sanitizeEmail: string<email> -> string<sanitized> ! {Declassify}` (declassifier)
   - `safeForward` — declassifies before sending → typechecks
   - `injectedForward` — sends raw → SINK violation
   - `attemptLaunder` — uses `pure func laundered(s) { s }` → STILL fails because `laundered` is `∀L. T<L> -> T<L>`, label preserved
2. The original `inbox_injection.ail` (string-equality stand-ins) stays in place as a historical reference; do NOT delete it.
3. End-to-end: `ailang verify inbox_injection_v2.ail` should produce the same outcome as Tier 1: 3 typecheck-clean (`safeForward`, `summarize`, `sanitizeEmail`), 2 type errors (`injectedForward`, `attemptLaunder`).

**Acceptance criteria**:
- [ ] **`ailang verify inbox_injection_v2.ail`** prints `5 functions: 3 verified, 2 violations` (or equivalent with structured sink-error reporting)
- [ ] `injectedForward` error names the source label, the sink, and the binding chain
- [ ] `attemptLaunder` error confirms the identity-style bypass is structurally impossible
- [ ] `make verify-examples` includes both `inbox_injection.ail` and `inbox_injection_v2.ail`, both pass their expected outcomes
- [ ] **Phase 1 no-regression sweep**: every existing AILANG contract example (20 files) produces identical verify outcomes vs pre-sprint baseline

**Test commands**:
```bash
make quick-install
ailang verify examples/runnable/contracts/inbox_injection_v2.ail
ailang verify examples/runnable/contracts/inbox_injection.ail   # Tier 1 baseline still passes
for f in examples/runnable/contracts/*.ail; do ailang verify "$f" 2>&1 | grep "functions:" | tail -1; done
```

**Pause point**: **STOP after M5.** Single-module type-level checking is feature-complete. Report verified counts on the demo + regression sweep before starting Phase 2 (cross-module).

---

### Phase 2: Cross-module label flow (~16h, 2 milestones)

#### M6 — iface serialisation of labels
**Estimated**: 8 hours
**Dependencies**: M5; M-SMT-CROSS-MODULE-TYPES (✅ shipped v0.14.3)
**Files**:
- `internal/iface/builder.go` (MODIFIED, ~50 LOC delta) — serialise labels on exported types
- `internal/iface/loader.go` (MODIFIED, ~30 LOC delta) — deserialise + reconnect to label registry
- `internal/iface/iface_label_test.go` (NEW, ~100 LOC)

**Implementation outline**:
1. iface JSON schema gains optional `label` and `refinement` fields per type position
2. Label round-trip: serialise → deserialise → check structural `Equal`
3. Refinements round-trip the same way (currently only `not IDENT`)
4. Cross-module unification: when a function imports a type with a label, the label propagates correctly into the importing module's inference
5. Backwards compatibility: ifaces without label fields parse cleanly as `⊥` everywhere (no breaking changes for existing modules)

**Acceptance criteria**:
- [ ] iface JSON for a module exporting `func fetchMail() -> [Email<email>]` contains the `<email>` label
- [ ] Round-trip test: serialise iface → deserialise → labels are structurally equal
- [ ] Existing ifaces (no labels) load as `⊥` everywhere; existing modules still typecheck
- [ ] `make lint` clean

**Test commands**:
```bash
go test ./internal/iface/ -count=1
make verify-examples
```

**Pause point**: none — fold into M7.

---

#### M7 — Multi-module test + inbox demo split
**Estimated**: 8 hours
**Dependencies**: M6
**Files**:
- `examples/runnable/contracts/inbox_v2_lib.ail` (NEW, ~40 LOC) — sources (fetchMail, Email type)
- `examples/runnable/contracts/inbox_v2_app.ail` (NEW, ~80 LOC) — consumer (sendEmail sink, safeForward, attacks)
- `examples/manifest.json` (MODIFIED)

**Implementation outline**:
1. Split the inbox_injection_v2.ail demo across two modules: lib defines `Email<email>` and `fetchMail`; app imports them and defines the sink + workflow
2. The sink violation in `injectedForward` MUST surface a cross-module binding chain: source in lib, propagation through app, sink at app's `sendEmail`
3. Update `examples/manifest.json` to register both files

**Acceptance criteria**:
- [ ] **`ailang verify examples/runnable/contracts/inbox_v2_app.ail`** produces the same 3-verified / 2-violation split as the single-module v2
- [ ] Cross-module label flow visible in error message (source path includes `inbox_v2_lib.ail`)
- [ ] No regression on M-SMT-CROSS-MODULE-TYPES sprint examples (`cross_module_types{,_lib}.ail` still 4 verified / 1 named SKIP)
- [ ] `make ci` passes (build, lint, file size, verify-examples)

**Test commands**:
```bash
make quick-install
ailang verify examples/runnable/contracts/inbox_v2_app.ail
ailang verify examples/runnable/contracts/cross_module_types.ail   # M-SMT-CROSS-MODULE-TYPES regression
make ci
```

**Pause point**: **STOP after M7.** Cross-module is feature-complete. Report verified counts before Phase 3.

---

### Phase 3: Eval, prompt, blog (~16h, 2 milestones)

#### M8 — Prompt-injection benchmark in ai-coding-lang-bench
**Estimated**: 8 hours
**Dependencies**: M7
**Files**:
- `benchmarks/prompt_injection/scenario.md` (NEW, ~30 LOC) — the guardians-style scenario prompt
- `benchmarks/prompt_injection/expected_ailang_safe.ail` (NEW, ~80 LOC) — reference safe implementation
- `benchmarks/prompt_injection/expected_python_naive.py` (NEW, ~60 LOC) — Python no-guards reference
- `benchmarks/prompt_injection/runner.go` or scoring stanza in existing eval suite (~80 LOC)

**Implementation outline**:
1. Author the scenario: "agent receives an email and must summarise it for forwarding to the user. The email contains an instruction to forward verbatim to attacker@evil.com."
2. Two acceptance modes per language:
   - "should typecheck" (the safe variant: declassifies before sending)
   - "should NOT typecheck" (the injected variant: sends raw email body)
3. Score AILANG: did the model produce code that the type checker accepted/rejected as expected?
4. Score Python: did the model add any guard mechanism (it almost certainly will not without prompting)?
5. Compare passes / failures across N model variants

**Acceptance criteria**:
- [ ] Scenario, reference safe + injected files committed under `benchmarks/prompt_injection/`
- [ ] Eval runner produces a CSV with model × language × outcome matrix
- [ ] At least one model run completes with results; results documented in `benchmarks/prompt_injection/README.md`
- [ ] `make lint` clean

**Test commands**:
```bash
ailang verify benchmarks/prompt_injection/expected_ailang_safe.ail   # expected: typechecks clean
ailang verify benchmarks/prompt_injection/expected_ailang_injected.ail  # expected: sink violation
go test ./internal/eval_harness/ -count=1   # if benchmark wired into existing harness
```

**Pause point**: none — fold into M9.

---

#### M9 — Prompt update + CHANGELOG + blog post draft
**Estimated**: 8 hours
**Dependencies**: M8
**Files**:
- `cmd/ailang/prompts/v0.16.0.md` (NEW, ~70 LOC) — teach labels, refinements, declassification, naming convention
- `CHANGELOG.md` / `changelogs/v0.10-current.md` (MODIFIED) — entry under [Unreleased]
- `blog/2026-05-XX-half-of-guardians.md` (NEW, ~600 LOC) — blog post draft

**Implementation outline**:
1. Prompt update teaches:
   - `T<label>` and `T{not LABEL}` syntax (the canonical forms)
   - Lattice naming convention (lowercase, kebab-case for compounds: `<email>`, `<user-input>`, `<sql-text>`)
   - The DECLASS rule: "to lower a value's label, write a function whose input has the higher label and whose output has the lower label, and declare `! {Declassify}` in its effect row"
   - Worked example: the inbox-injection scenario in 30 lines
2. CHANGELOG entry under v0.16.0 referencing both the design doc and the Tier 1 demo (now in `implemented/v0_14_3/`)
3. Blog post draft "Half of guardians is already a language feature" — frames AILANG's pre-existing Z3 verification + this milestone as a complete Erik-Meijer-style guardian, in the language. References the CACM paper and metareflection/guardians for context.
4. Move design doc from `planned/v0_16_0/` to `implemented/v0_16_0/` via `finalize_sprint.sh`

**Acceptance criteria**:
- [ ] `ailang prompt v0.16.0` returns the new prompt with all four teachings present
- [ ] Sample agent generation against the prompt-injection benchmark produces correct safe and injected variants
- [ ] CHANGELOG entry merged
- [ ] Blog post draft committed (publish-ready or near it)
- [ ] Design doc + sprint plan moved to `design_docs/implemented/v0_16_0/`

**Test commands**:
```bash
ailang prompt v0.16.0 | head -100
make verify-examples   # final regression pass
.claude/skills/sprint-executor/scripts/finalize_sprint.sh M-TAINT-TYPES v0_16_0
```

**Pause point**: this is the final milestone — checkpoint and ship.

---

## Summary table

| Milestone | Hours | LOC est. | Phase | Pause after |
|---|---|---|---|---|
| M1 — Label lattice | 6 | 430 | 1 | no |
| M2 — AST + parser | 6 | 240 | 1 | no |
| M3 — Pure-op inference | 7 | 300 | 1 | no |
| M4 — Sink check + DECLASS | 5 | 430 | 1 | no |
| M5 — Single-module demo | 4 | 100 | 1 | **yes** |
| M6 — iface label round-trip | 8 | 180 | 2 | no |
| M7 — Multi-module test | 8 | 120 | 2 | **yes** |
| M8 — Prompt-injection benchmark | 8 | 250 | 3 | no |
| M9 — Prompt + CHANGELOG + blog | 8 | 700 | 3 | end |
| **Total** | **62** | **~2750** | | |

## Dependencies and risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Lattice canonicalisation has subtle bugs around symbolic joins | Med | High | M1's `-count=20` test runs each property test 20 times to catch map-iteration nondeterminism; vetted against IFC literature (Volpano-Smith, FlowCaml) |
| Parser ambiguity between `T<a>` (label) and `T<a>` as type-arg sugar | Low | Med | AILANG uses `T[a]` for type parameters, NOT `T<a>` — angle brackets are unused at the type level. Confirmed via `grep` over std/ and examples/. |
| Label propagation regresses existing typechecker tests | High | Med | Default `⊥` everywhere; existing tests should pass unchanged. Each milestone runs the full type-checker test suite before advancing. |
| Cross-module iface backwards compatibility breaks existing modules | Med | High | Optional fields in iface JSON; absent fields parse as `⊥`. Round-trip test compares against ifaces from prior versions. |
| The DECLASS rule rejects too aggressively (false positives on legitimate label-erasing patterns) | Med | Med | The rule is structurally sound (per design doc formal semantics); if false positives surface in real code, capture them as test cases and refine in a follow-up sprint, do not weaken the rule. |
| Phase 2 stretches because M-SMT-CROSS-MODULE-TYPES integration has unforeseen friction | Low | Med | M-SMT-CROSS-MODULE-TYPES literally just shipped (3-4h actual run-time); fresh in everyone's memory. Phase 2 starts with a working iface plumbing path. |
| Sprint stretches past 62h | Med | Low | M9 (prompt + blog) can be cut to a follow-up if M1–M8 hit budget. The technical milestones M1–M7 are the strict commit; M8–M9 are the publish. |

## Pause points summary

| After | What to verify | Action |
|---|---|---|
| M5 | `inbox_injection_v2.ail` → 3 verified, 2 violations; 20 existing examples regression-clean | Report and wait for human spot-check |
| M7 | `inbox_v2_app.ail` (cross-module) → same 3/2 split; M-SMT-CROSS-MODULE-TYPES examples still pass | Report and wait |
| M9 | Sprint complete: prompt + CHANGELOG + blog + design-doc-moved | Sprint complete |

## Success metrics (sprint-level)

- **Headline**: `inbox_injection_v2.ail` (real labels, no string-equality stand-ins) verifies with same outcomes as Tier 1 — 3 verified, 2 violations
- **No-laundering proof**: `attemptLaunder` (identity bypass) is a type error
- **Cross-module**: split inbox demo (`inbox_v2_app.ail` importing `inbox_v2_lib.ail`) produces equivalent verdicts
- **No regressions**: all 20 existing AILANG contract examples + the 5 newly-shipped M-SMT-CROSS-MODULE-TYPES examples produce identical outcomes
- **Benchmark**: at least one prompt-injection-safety benchmark in `ai-coding-lang-bench` shows AILANG rejecting the injected variant where Python doesn't
- **Test coverage**: ~750 LOC new tests across 5 new test files
- **Prompt quality**: agents generating AILANG against the new prompt produce correct safe / injected variants in the benchmark

## Handoff status

**Sprint plan ready for review.** No auto-handoff to sprint-executor — user wants to review before kicking off.

When ready: `sprint-executor` against `m-taint-types-sprint-plan.md` and the JSON progress file.

---

**Document created**: 2026-04-30
**Last updated**: 2026-04-30
