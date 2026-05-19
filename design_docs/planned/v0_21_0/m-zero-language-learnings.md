# M-ZERO-LANGUAGE-LEARNINGS: Borrowed Ideas from Vercel Labs' Zero

**Status**: Planned
**Target**: v0.21.0 (Phase 1: JSON envelope + Phase 1.5: modular skills) → v0.22.0 (Phase 2: fix-plan + Phase 3: eval-suite inclusion, gated)
**Priority**: P2 — Medium (tactical improvements, plus a high-signal test of the shape-thesis in Phase 3)
**Estimated**: ~80–100 hours across 2 sprints (Phase 1: ~25h, Phase 1.5: ~10h, Phase 2: ~25h, Phase 3: ~20h when unblocked)
**Dependencies**:
  - Existing diagnostic infrastructure (`internal/errors/`, `internal/iface/`)
  - `ailang explain` / `ailang docs` CLI surface
  - Eval harness language registry (`internal/eval_harness/langreg/`)

**Commissioning context**: [Vercel Labs](https://github.com/vercel-labs/zero) published **Zero** on 15 May 2026 — a systems language with the same tagline AILANG uses, _"The programming language for agents."_ Zero is C-family, manual-memory, native-compiled — fundamentally a different design philosophy. But the convergence on **explicit effects, capability objects, and machine-readable diagnostics** is striking, and a few of Zero's CLI / diagnostic patterns are clearly worth borrowing.

See [docs/docs/guides/zero-comparison.md](../../../docs/docs/guides/zero-comparison.md) for the public-facing comparison.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Diagnostic/CLI work; no semantic change |
| A2: Replayability | +1 | JSON fix-plan artifacts are replayable; current diagnostics are advisory text |
| A3: Effect Legibility | 0 | No change to effect surface |
| A4: Explicit Authority | 0 | No new authority surface |
| A5: Bounded Verification | +1 | Structured `expected/actual/repair` fields make diagnostic correctness testable |
| A6: Safe Concurrency | 0 | N/A |
| A7: Machines First | +2 | Direct upgrade to the agent diagnostic-consumption surface; the entire point |
| A8: Minimal Syntax | 0 | No language-surface changes |
| A9: Cost Visibility | 0 | N/A |
| A10: Composability | 0 | N/A |
| A11: Structured Failure | +2 | Fix-plan is literally "make failures structured enough to act on" |
| A12: System Boundary | 0 | N/A |

**Net Score: +6** → **Decision: Proceed to Phase 1**

### Hard Violation Check

- ❌ Does this require non-determinism? **No.**
- ❌ Does this hide effects? **No.**
- ❌ Does this require unsafe memory? **No.**
- ❌ Does this introduce ambient authority? **No.**

No violations. Proceed.

---

## Motivation

Comparing AILANG to Zero reveals three concrete CLI/diagnostic patterns Zero ships that AILANG either lacks or implements inconsistently. None of them are speculative — they're already proven in Zero's first release.

### What Zero does that AILANG should consider

1. **`zero fix --plan --json`** — Compiler emits a structured patch plan an agent can apply directly. AILANG currently emits human-readable suggestions only.
2. **`zero explain <CODE> [--json]`** — Every diagnostic has a stable error code and a dedicated `explain` subcommand with both human and JSON output. AILANG has `ailang docs` for stdlib but no equivalent for errors.
3. **Unified `--json` across all CLI commands** — `zero graph`, `zero size`, `zero routes`, `zero doctor` all support `--json`. AILANG has `--json` on _some_ commands; not all.
4. **Diagnostic schema with `expected / actual / help / repair` fields** — Zero's JSON diagnostics are designed for tool consumption, not pretty-printing. AILANG's diagnostics have these fields conceptually but aren't always exposed in machine form.
5. **Modular bundled skills via `zero skills`** — Zero ships seven focused, version-matched prompts inside the compiler binary (`zero-language`, `zero-agent`, `zero-diagnostics`, `zero-builds`, `zero-packages`, `zero-stdlib`, `zero-testing`). External skill managers (Claude Code marketplace, etc.) get a thin bootstrap skill that defers to `zero skills` for the rest. AILANG's `ailang prompt` ships a single consolidated file per version — agents pay the full token cost even when they only need diagnostic guidance. Splitting the prompt into focused skills is a real cost win for the eval loop and for production agent contexts. **Worth adding as Phase 1.5.**

### What Zero does that AILANG should _not_ borrow

For the record (and to prevent scope creep):
- Manual memory management (`owned<T>`, `defer`, borrow checker) — antithetical to AILANG's GC + pure-functional design.
- Width-explicit primitives (`i8`/`u32`/`usize`) — wrong abstraction level for an applied-FP language.
- Native binary compilation — already deliberately deferred, see [project_codegen_strategic_decision](../../implemented/v0_18_5/) (evaluator-first + bytecode VM).
- C FFI via `extern c` — Go interop already covers our use case.
- Imperative statements — would gut the "everything is an expression" axiom.

---

## Proposal

### Phase 1 — JSON pipeline envelope (v0.21.0, ~25h)

**Scope expanded after smoke-testing Zero v0.1.1.** Zero's `zero check --json` output is best-in-class — schema versioning, cache observability, interface fingerprints, capability-sandbox declarations, and phase timings all embedded in a single envelope per invocation. Most of the substrate already exists in AILANG; the work is wiring + surfacing, not new design.

#### Existing AILANG substrate to leverage

- [internal/schema/](../../../internal/schema/) — already has schema versioning, `ErrorV1` schema, golden tests
- [internal/manifest/manifest.go:25](../../../internal/manifest/manifest.go) — `SchemaVersion = "ailang.manifest/v1"` pattern is the convention to follow
- [internal/errors/](../../../internal/errors/) — already has `NewTypecheck("TC#123", TC001, ...)` style stable codes (e.g. `TC001`)
- v0.11.2 incremental typecheck — interface fingerprints exist conceptually
- `ailang iface <module>` — already produces JSON interface output
- `--debug-compile` — already collects phase timings (just not as JSON yet)
- Capability runtime — already tracks which capabilities a program declares/uses

#### Target envelope (modeled on Zero, adapted to AILANG)

```json
{
  "schemaVersion": "ailang.check/v1",
  "ok": true,
  "sourceFile": "examples/hello.ail",
  "diagnostics": [
    {
      "code": "TC001",
      "schema": "ailang.error/v1",
      "phase": "typecheck",
      "severity": "error",
      "span": {"file": "...", "line": 3, "col": 5},
      "message": "...",
      "expected": "Int",
      "actual": "String",
      "help": "...",
      "fix": {"suggestion": "...", "confidence": 0.85}
    }
  ],
  "compilerPhases": [
    {"name": "parse", "elapsedMs": 2, "cacheable": true},
    {"name": "elaborate", "elapsedMs": 5, "cacheable": true},
    {"name": "typecheck", "elapsedMs": 12, "cacheable": true},
    {"name": "linker", "elapsedMs": 3, "cacheable": false}
  ],
  "compilerCaches": [
    {"name": "parseTree", "key": "...", "hit": true, "stored": true, "invalidatesOn": "source"},
    {"name": "interface", "key": "...", "hit": true, "stored": true, "invalidatesOn": "public symbols"},
    {"name": "checkedBody", "key": "...", "hit": false, "stored": true, "invalidatesOn": "source or target"}
  ],
  "interfaceFingerprints": {
    "schemaVersion": "ailang.iface/v1",
    "modules": [
      {"name": "examples/hello", "sourceHash": "...", "publicInterfaceHash": "...", "publicSymbolCount": 1}
    ]
  },
  "compileTime": {
    "sandbox": {"filesystem": "denied", "network": "denied", "process": "denied", "ambientEnv": "denied"}
  },
  "incrementalInvalidation": {
    "affectedModules": 1,
    "cacheHits": 4,
    "cacheMisses": 1,
    "changedInputs": {"sourceFiles": ["examples/hello.ail"]}
  }
}
```

#### Work items

1. **Schema-version every JSON output** (~3h)
   - `ailang check --json`, `ailang ai-check`, `ailang iface`, `ailang axioms --json`, `ailang budget --json`, etc.
   - Add `schemaVersion` field per command (e.g. `ailang.check/v1`, `ailang.iface/v1`)
   - Follow `internal/manifest/manifest.go` precedent

2. **Stable error codes** (~5h)
   - Audit `internal/errors/` — make sure every category has a stable `TC001`-style code (some do, audit gaps)
   - New: `internal/errors/codes.go` — central registry mapping code → human description
   - Wire `ailang explain <CODE>` subcommand (e.g. `ailang explain TC001`) that reads from the registry — closes the loop Zero's own `zero explain` failed to deliver in v0.1.1
   - Add `fix.suggestion` and `fix.confidence` fields to every diagnostic that has a known repair pattern (substrate already exists per `golden_test.go`)

3. **`compilerPhases[]` in check JSON** (~3h)
   - Lift `--debug-compile` phase-timing collection into the always-on path
   - Emit array of `{name, elapsedMs, cacheable}` in check output
   - Touches: `cmd/ailang/check.go`, pipeline phase-timer

4. **`compilerCaches[]` observability** (~5h)
   - Survey existing cache hit/miss tracking (incremental typecheck v0.11.2 has some)
   - Add per-cache structured entries `{name, key, hit, stored, invalidatesOn}`
   - Bonus: surfaces "why was this cached" reasoning that helps agents understand build behavior

5. **`interfaceFingerprints` embedded in check output** (~3h)
   - Currently available via separate `ailang iface` call
   - Inline a compact form in `check --json` so a single call gives both diagnostics and module-state fingerprints
   - Same data, same hashes — just stop forcing two invocations

6. **`compileTime.sandbox` declaration** (~2h)
   - From the capability runtime, emit which capabilities the program _required_ vs were _denied_
   - This is a powerful agent-facing signal: "this program declared `! {IO, FS}` but didn't actually use FS" → capability tightening hint

7. **`ailang explain <CODE>` CLI** (~4h)
   - Reads from `internal/errors/codes.go` registry
   - Supports `--json` for tool consumption
   - Documents one row from the diagnostic registry per invocation
   - Note this is a Phase 1 deliverable even though Zero deferred to source-checkout-only: AILANG ships it in the binary or doesn't ship it at all

**Files to touch:**
- `cmd/ailang/check.go`, `cmd/ailang/ai_check.go` — wire envelope
- `internal/schema/` — extend with `CheckV1`, `PhasesV1` schema constants
- `internal/errors/codes.go` (new) — code registry + `ailang explain` source
- `internal/pipeline/` — phase-timer always-on, surface to JSON
- `internal/iface/` — expose compact fingerprint form for inline embedding
- `internal/loader/` or wherever capability runtime lives — surface sandbox decisions
- `cmd/ailang/explain.go` (new) — `ailang explain <CODE>` subcommand
- New: `docs/docs/reference/json-schema.md` — full envelope spec, published JSON Schema for round-trip validation
- Golden tests under `internal/schema/golden_test.go` — extend with check-envelope cases

**Acceptance criteria:**
- `ailang check --json file.ail` emits the full envelope above and round-trips through a Go decoder in tests.
- `ailang explain TC001` returns a human-readable description; `ailang explain TC001 --json` returns the registry row.
- Every JSON-emitting CLI command has a top-level `schemaVersion` field.
- Schema documented at `docs/docs/reference/json-schema.md` with a published JSON Schema file the eval harness validates against in CI.
- Existing `--json` consumers (eval harness, MCP server, dashboard) still parse correctly — additive only, no breaking changes to existing fields.

**What we explicitly do NOT borrow from Zero's schema:**
- `packageCache` with manifest-hash invalidation reasons — AILANG's package model is different (toml-based, no lockfile equivalent today)
- `selfHostRouting` metadata — AILANG isn't self-hosted, irrelevant
- `direct backend` / `wasm` emit metadata — AILANG is interpreted

### Phase 1.5 — Modular skill split (v0.21.0, ~10h)

Refactor `prompts/v0.X.X.md` from a single consolidated file into focused topic prompts modeled on Zero's seven skills:

| Skill | Loaded when agent is… |
|---|---|
| `ailang-language` | Writing or reviewing `.ail` code |
| `ailang-effects` | Working with effect rows / capabilities |
| `ailang-stdlib` | Calling stdlib (covered partly by existing `ailang docs`) |
| `ailang-diagnostics` | Interpreting compiler errors (depends on Phase 1 codes) |
| `ailang-testing` | Writing tests or property checks |
| `ailang-agent-workflow` | The edit/check/fix loop itself |
| `ailang-packages` | Authoring or consuming packages |

Surface via `ailang prompt <skill-name>` plus `ailang prompt list`. Keep `ailang prompt` (no arg) returning the consolidated bundle for back-compat.

**Files to touch:**
- `prompts/v0.X.X/` — directory of split skills (replaces single `.md`)
- `prompts/versions.json` — track per-skill versioning
- `cmd/ailang/prompt.go` — new subcommands
- `internal/prompt/loader.go` — modular loading

**Acceptance criteria:**
- `ailang prompt ailang-language` returns the language-syntax slice only.
- Eval harness can load skill-specific prompts and measures pass-rate parity vs. monolithic prompt on `core` tier benchmarks (no regression).
- Token count for `ailang-language` alone is ≤ 40% of the consolidated prompt.

### Phase 2 — `ailang fix --plan --json` (v0.22.0, ~25h)

Add a `fix` subcommand modeled on `zero fix --plan --json`:

```bash
ailang fix --plan --json file.ail
```

Emits a JSON document with:
- The list of diagnostics found
- For each diagnostic with confidence ≥ threshold, a `patch` object: `{ file, range, before, after, rationale }`
- A summary `{ total, fixable, unfixable }`

The agent (or coordinator) applies the patches; the language doesn't auto-apply. This is _strictly_ a structured proposal — no implicit mutation.

**Files to touch:**
- New: `cmd/ailang/fix.go`
- New: `internal/fixplan/` package — patch generation
- Extend categorizer with `Patch` candidates per error category

**Acceptance criteria:**
- `ailang fix --plan --json examples/wrong_constructor.ail` emits a patch that, when applied, makes the file type-check.
- Eval harness uses the fix-plan in its agent-mode retry loop (replaces current best-effort text repair).
- At least the 10 most common error categories have patch generators.

### Phase 3 — Eval suite inclusion (gated on Zero runtime maturity; revisit v0.22.0+)

**Decision: YES in principle, BLOCKED in practice as of Zero v0.1.1 (May 2026).**

Adding Zero to the eval suite remains the cleanest possible test of AILANG's own shape-over-training thesis — that argument hasn't changed. But a hands-on smoke test of the released Zero v0.1.1 binary (17 May 2026, in `/tmp/zero-smoke/`) revealed concrete runtime blockers that make benchmark execution impossible today:

**Smoke-test findings (Zero v0.1.1):**

1. **Direct emit "MVP subset" is severely limited.** Compiler diagnostic verbatim: _"restrict this program to exported no-parameter functions returning small integer literals."_ Any program with a `String` local or parameter fails to lower with `CGEN004`. `balanced_parens` cannot be expressed in a runnable form.
2. **C bridge deliberately removed.** The compiler's own JSON output reports `"cBridge":{"policy":"removed","explicitDirectFallback":"never-c-bridge"}`. Even with `--cc /usr/bin/clang` (which links trivial programs), the lowerer rejects `[4]String` locals before the link step.
3. **No float / math stdlib.** `adt_option` (calls `safeSqrt`) literally cannot be expressed in Zero v0.1.1. Full stdlib is integer/byte/IO oriented per `std.mem`, `std.io`, `std.fs`, `std.parse`, `std.codec`, `std.json`, `std.time`, `std.rand`, `std.crypto`, `std.net`, `std.http`.
4. **`zero skills` not in released binary.** The "version-matched bundled skills" pitch from CHANGELOG is currently served only by the `bin/zero` shell wrapper in the source checkout. The released binary returns: _"served by bin/zero wrapper; run `bin/zero skills` from the checkout."_ Agents would need to fetch skills from GitHub directly.
5. **`zero explain <code>` doesn't recognize its own diagnostic codes.** Tried `PAR100`, `TYP002`, `CGEN004` — all return `NAM003: unknown diagnostic code`. The fix-plan workflow we wanted to borrow from Zero is partially aspirational at this version.

**Syntax friction observed (would hurt agent pass rate):**

- No `else if` chains — must nest `else { if ... }`
- No `'(' as u8` cast — must use `40_u8` numeric literals
- `Maybe<T>` is `.has`/`.value` struct, not Some/None match — uncommon shape an LLM trained on Rust/Haskell wouldn't reach for first

**Genuine positives confirmed:**

- Install is clean: 656KB single binary from official GitHub releases, no telemetry, Apache 2.0.
- Type-checker handles real programs: `zero check balanced.0` passed on a Zero-translated `balanced_parens` solution.
- **JSON diagnostic schema is genuinely best-in-class** (schema versioning, cache-key inputs, sandbox declarations, phase timings, interface fingerprints, incremental invalidation metadata). This validates Phase 1.

**Re-gating Phase 3:**

| Gate | Trigger |
|---|---|
| Runtime works | `zero run` or `zero build` produces an executable for a program with `String` parameters, on the released binary, without source-checkout bootstrap |
| Skills accessible | `zero skills get zero-language` works in the released binary (not just the source-checkout wrapper) |
| Float math available | `std.math` ships with at least `sqrt`, basic arithmetic over `f64` |
| One of the above | We can do a **check-only** eval pass measuring "fraction of LLM-generated Zero programs that type-check" — this is a real signal of language fluency from prompt, and could be wired with the existing infra in ~3-4h once we accept the limitation. Optional standalone gate. |

**Trigger conditions to revisit:**
- Zero releases v0.2.0+ with working `zero run` on non-trivial programs, OR
- Zero releases ship `compiler-zero` (the self-hosted compiler) as a binary, OR
- A 6-month timeline check (Nov 2026) regardless

**Implementation when unblocked** (preserved from earlier draft):

- New `internal/eval_harness/langreg/zero.go` modeled on `javascript.go`.
- `LoadSyntaxRef` shells out to `zero skills get zero-language` (or bundles the skill text at AILANG-build time, version-pinned to a Zero release that actually supports `zero skills` in the binary).
- Runner invokes `zero check --json <file>` for check-only benchmarks, `zero run <file>` once runtime works.
- Add Zero to `languages:` in a curated subset of existing benchmarks where Zero's stdlib actually covers the requirements (NOT `adt_option`, NOT anything float-heavy, NOT anything needing concurrency).
- CI: Zero compiler installed via direct GitHub release URL (`curl -fsSL -o ~/.zero/bin/zero https://github.com/vercel-labs/zero/releases/latest/download/zero-<platform>-<arch>` + `chmod +x`). The pipe-to-sh installer is unnecessary.

**Acceptance criteria (unchanged):**

- `make eval-suite LANGS=zero BENCHMARKS=<curated-set>` runs end-to-end against at least three models.
- Results land in the public dashboard with a clear "Zero is alpha (v0.x, May 2026)" caveat — same epistemic posture we use for early AILANG versions.
- We publish a short post on the pass-rate matrix and what it implies about the shape thesis.

**Concrete artifacts to study from this smoke test** (these are wins regardless of eval inclusion):

- Zero's `zero check --json` schema → influence Phase 1 work. Particularly: `schemaVersion`, `compilerCaches[]`, `interfaceFingerprints`, `incrementalInvalidation`, `compileTime.sandbox`. AILANG's `--json` outputs should consider similar structure.
- Zero's `--cc /path/to/clang` flag pattern → useful precedent if AILANG ever wires a Go-emit backend.
- Zero's `compileTime.sandbox: { filesystem:denied, network:denied, process:denied }` declarations in diagnostics → could surface in AILANG eval traces for capability auditing.

**Original full-runtime decision (preserved):**

The original draft of this doc deferred Zero inclusion on "no training data" grounds. That was a category error. AILANG's entire methodology already assumes models have minimal training exposure to AILANG — that's what `ailang prompt` exists for, that's what every benchmark in the suite measures. If "language shape + good in-context prompt is enough" works for AILANG, it must be testable for Zero too. Refusing to test Zero under the same conditions we test ourselves is incoherent.

**What this tests:**

1. **The shape thesis directly.** AILANG and Zero are two independent "designed for agents" languages. Comparing both against Python (memorized by every model) and against each other tells us whether language-for-agents design actually moves pass-rate, or whether it's just nice ergonomics for humans.
2. **Prompt-loading isomorphism.** Zero ships `zero skills get zero-language` — exactly the same prompt-injection surface as AILANG's `ailang prompt`. The eval harness already knows how to load a language-specific syntax reference into context (per `langreg.LoadSyntaxRef`); adding Zero is mechanically symmetric to adding any other language.
3. **Negative-result protection.** If Zero out-performs Python on ADT/Option benchmarks from prompt alone, that's a major validation of shape-over-training. If Zero badly under-performs AILANG, that suggests AILANG's specific design choices (HM inference, row-polymorphism, ADT pattern matching) matter beyond just "explicit effects + good prompt."

**Implementation:**

- New `internal/eval_harness/langreg/zero.go` modeled on `javascript.go`.
- `LoadSyntaxRef` shells out to `zero skills get zero-language` (or bundles the skill text at AILANG-build time, version-pinned).
- Runner invokes `zero check --json <file>` for compile-only benchmarks, `zero run <file>` for runtime benchmarks.
- Add Zero to `languages:` in a curated subset (~10) of existing benchmarks: `adt_option`, `binary_tree_sum`, `balanced_parens`, `cli_args`, `effect_pure_separation`, etc. Start with benchmarks where AILANG's shape advantage should be most visible.
- New `prompts/zero/` directory mirroring `prompts/` for AILANG, with versioned skill content.
- CI: Zero compiler installed via `curl -fsSL https://zerolang.ai/install.sh`.

**Acceptance criteria:**

- `make eval-suite LANGS=zero BENCHMARKS=adt_option` runs end-to-end against at least three models.
- Results land in the public dashboard with a clear "Zero is alpha (v0.x, May 2026)" caveat — same epistemic posture we use for early AILANG versions.
- We publish a short post on the pass-rate matrix and what it implies about the shape thesis.

**Caveats we should document, not use as excuses:**

- Zero is unstable. Pin to a specific Zero release and re-run when it changes. Same as we do for AILANG.
- The first run will likely have lower pass rates than AILANG because Zero's prompt is newer and less tuned. That's data, not a reason to delay.
- Some benchmarks may not be expressible in Zero (no concurrency, no LLM primitives). Mark them `not_supported` and report the coverage gap honestly.

---

## Out of Scope

- Adopting any Zero syntax. AILANG's syntax is settled per the [Design Axioms](/docs/references/axioms).
- Manual memory management of any kind.
- Native binary compilation (see [project_codegen_strategic_decision](../../implemented/)).
- Migrating AILANG's error type from `Result[T, E]` to algebraic-effect-style `raises`.

---

## Open Questions

1. **Error code scheme.** Zero uses `NAM003`-style. Should AILANG use `AIL-TC-001` (category-prefixed) or stay numeric? Lean toward category-prefixed for grep-ability.
2. **Fix-plan confidence threshold.** Zero's docs don't expose this; we'll need to tune empirically. Start at 0.7, gather eval data, adjust.
3. **Patch idempotency.** If an agent applies a fix-plan and re-runs, the second `fix --plan` should be empty. Need a regression test.
4. **Eval harness integration.** Should agent-mode use `ailang fix --plan` as the canonical retry mechanism, replacing the current free-form "ask the model to fix it" loop? Likely yes — measure on a small benchmark first.

---

## Success Metrics

**Phase 1:**
- 100% of emitted diagnostics have stable codes (audit via grep + test).
- `ailang check --json` schema validates against published JSON Schema in CI.

**Phase 2:**
- `ailang fix --plan` produces apply-able patches for ≥ 80% of fixable diagnostics in a curated test suite.
- Eval harness agent-mode retry pass rate improves by ≥ 5 percentage points when using `--plan` vs. free-form retry (measured on `core` tier benchmarks).

**Phase 3 (if triggered):**
- Zero benchmarks have ≥ 30% pass rate on at least one model before promotion from `--experimental`.

---

## References

- [Vercel Labs Zero](https://github.com/vercel-labs/zero) — source
- [docs/docs/guides/zero-comparison.md](../../../docs/docs/guides/zero-comparison.md) — public comparison
- [M-EFFECT-HANDLERS](./m-effect-handlers.md) — prior Koka-comparison precedent for "borrow ideas, don't borrow syntax"
- [project_codegen_strategic_decision](../../implemented/) — why we already decided _not_ to chase native compilation
- AILANG eval harness language registry: [internal/eval_harness/langreg/](../../../internal/eval_harness/langreg/)
