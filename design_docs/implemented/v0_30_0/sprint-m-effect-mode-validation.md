# Sprint Plan: M-EFFECT-MODE-VALIDATION

**Design doc**: [m-effect-mode-validation.md](m-effect-mode-validation.md)
**Sprint JSON**: `.ailang/state/sprints/sprint_M-EFFECT-MODE-VALIDATION.json`
**Planner model**: claude-opus-4-8 (mission iteration 8 routing policy)
**Planned**: 2026-07-11 · branch `dev` · HEAD `b91338dab`
**Verification binary**: `~/go/bin/ailang` = `AILANG v0.29.0-2-g8dad7e80b` (matches `git describe`, fresh)
**Duration**: 1 day (~8h) · **Risk**: low · **Total LOC estimate**: ~360 (impl ~150 + tests ~180 + docs ~30)

---

## Summary

Phase 1 (v0.15.0) ratified a **closed mode set** and the public guide claims the typechecker
rejects unknown values — **that claim is false at HEAD** (verified live below). This sprint makes
it true: a per-effect parameter **schema** (allowed keys → allowed value sets) registered in
`internal/types/effects.go`, enforced at effect-row **elaboration** inside
`ElaborateEffectRowWithBudgets`, producing **3 structured, fix-carrying diagnostics**. Fixtures are
promoted to the existing diagnostic-fixture CI harness (`internal/diag/footgun_fixtures_test.go` +
`footguns.md`). The guide's interim accuracy note is removed; the teaching prompt gains the new
codes.

---

## Premise Verification (live, 2026-07-11, HEAD b91338dab)

All design-doc premises were re-verified against live code and `ailang check` transcripts before
planning. **Every core premise holds.** Verified facts:

| Premise | Status | Evidence |
|---------|--------|----------|
| Bug reproduces at v0.29.0-2 | ✅ CONFIRMED | `Clock[mode=banana]`, `Rand[mode=banana]`, `Rand[flavor=hot]` all `✓ No errors found!` (rc=0) via `~/go/bin/ailang check` |
| No validation code in-tree | ✅ CONFIRMED | `defaultEffectModes` (effects.go:32) is `map[string]struct{Key,Value string}` — defaults only, no value sets; no `unknown mode`/`validModes` in `internal/types` or `internal/parser` |
| `defaultEffectModes` at effects.go:32, `effectiveParamsOf` at :75, bridge comment at :72 | ✅ CONFIRMED | exact line numbers match |
| Parser accepts arbitrary key/value | ✅ CONFIRMED | `parseEffectParams` (parser_effect.go:263) accepts any bare-ident key + any bare-ident value; only reports empty-list/dup-key/malformed. **This is why validation cannot live in the parser** — it must be at elaboration |
| Elaboration chokepoint carrying params | ✅ CONFIRMED | `ElaborateEffectRowWithBudgets` (effects.go:225) captures `eff.Params` at :264-270. Called from `typechecker_functions.go:104`; its returned `error` already surfaces as a diagnostic `invalid effect annotation at <span>: <err>` |
| AI shipped surface = `{mode: fixed\|routeable\|replay-only, scope: byok}` | ✅ CONFIRMED against M-AI-EFFECT-MODES report | report §"AI mode table" lines 149-152 register exactly those; `replay-only` + `scope=byok` are "Reserved (parser accepts; runtime stub)". Present in tests (`effect_params_test.go:475,562`) and treated as routing-intent in `routing_flags.go:183`. **The doc's guessed schema table is CORRECT.** |
| Rand shipped surface = `{mode: os\|seeded\|crypto}` | ✅ CONFIRMED | `examples/modal_rand.ail` + `prompts/v0.16.x.md` register those three; effects.go default `mode=os` |
| Diagnostic-fixture harness exists | ✅ CONFIRMED | `internal/diag/footgun_fixtures_test.go` (table-driven, drives `pipeline.Run(ModeCheck)`, asserts `code` + `fix` substrings) + `internal/diag/footguns.md`. Sibling sprint M-DIAG-FIXTURE-PROMOTION used this exact harness |
| Guide interim note present | ✅ CONFIRMED | `docs/docs/guides/parameterised-effects.md`: "Mode set is closed" §150, Accuracy note §155-159 |
| Legal v0.15.0 forms pass today | ✅ CONFIRMED | `examples/modal_rand.ail`, `examples/ai_modes.ail`, and all of `{Rand[mode=seeded\|crypto]}`, `{AI[mode=routeable]}`, `{AI[scope=byok]}` → `✓ No errors found!` |

### Discrepancies found (2)

**D1 — The `stringSliceToEffectRow` bridge cannot carry an illegal param; the doc over-scopes it.**
The design doc's Conflict Surface (§Solution Design) and Risks table both mandate validation "cover
the `stringSliceToEffectRow` bridge path." Live reading of `validate_effects.go:191-208` shows this
bridge builds rows from effect **name strings only** (`ast.EffectNames` → labels; `Params` field is
always nil). **Params never flow through this bridge**, so it can never present an illegal param to
validate. The only param-carrying elaboration path is `ElaborateEffectRowWithBudgets`. **Impact on
plan:** validation is placed in `ElaborateEffectRowWithBudgets` (the true chokepoint); the bridge is
covered by a cheap *regression* test proving it still round-trips (nil params → no validation
triggered), satisfying the doc's back-compat fixture (§Conflict Surface item 4 iface-cache
round-trip) without needing validation logic in the bridge itself. This is a **scope reduction**,
not a gap.

**D2 — Error codes are the planner's call; `PAR_EFF0NN` is a *parser* namespace, unusable here.**
The doc defers error-code naming to the planner "following existing conventions in
`internal/pipeline/validate_effects.go`" — but that file has **no error codes** (it emits prose
`Effect checking failed for function...`). The real code convention is the parser's `PAR_EFF0NN_*`
family (parser_effect.go), which is a **parse-phase** namespace and semantically wrong for an
elaboration/type-phase rejection. **Decision (this plan):** use the `EFF_`-prefixed semantic codes
the design doc's own Examples section already uses — `EFF_UNKNOWN_MODE`, `EFF_UNKNOWN_PARAM_KEY`,
`EFF_PARAMS_NOT_SUPPORTED` — which are (a) consistent with the doc's stated Success Criteria, (b)
outside the `PAR_` parse namespace, (c) greppable and stable for AI repair loops. The code string
must be embedded verbatim in the returned `error` text so the footgun harness (`strings.Contains`)
and `ailang check` both see it.

---

## High-Impact Decisions (resolved by planner)

| Decision | Resolution | Rationale |
|----------|-----------|-----------|
| Schema location | **Extend `defaultEffectModes`** into a richer `effectSchema` table in the SAME file (`internal/types/effects.go`), keeping `DefaultModeFor` working off it | One audit point; the guide already points readers at `effects.go`; Phase-5 ports edit the same table |
| Validation location | **`ElaborateEffectRowWithBudgets`** (effects.go:225), at the point `eff.Params` are captured (:264-270) | Sole param-carrying path; error already surfaces as a diagnostic with span; covers elaborator-built rows (bridge carries no params — see D1) |
| Error codes | **`EFF_UNKNOWN_MODE`** / **`EFF_UNKNOWN_PARAM_KEY`** / **`EFF_PARAMS_NOT_SUPPORTED`** | Matches doc Examples; semantic (not `PAR_`) namespace; embedded verbatim in error text |
| Params on schema-less effect (Clock/Net/FS) | **Hard error** `EFF_PARAMS_NOT_SUPPORTED` naming the tracking doc `m-effect-clock-net-fs-modes` | Design Freeze mandates hard error; makes Phase-5 ports *unlock* syntax |

---

## Schema Table (frozen)

```
Rand: { mode: {os, seeded, crypto} }
AI:   { mode: {fixed, routeable, replay-only}, scope: {byok} }
```

Verified against M-AI-EFFECT-MODES outcome report (lines 149-152) and `examples/modal_rand.ail`.
All other effects (IO, FS, Net, Clock, Env, DB, …): **no schema** → any explicit param is a hard
error (`EFF_PARAMS_NOT_SUPPORTED`). Bare forms of all effects are unaffected (no params to validate).

---

## Milestones

### M1 — Schema table + validation function (~4h, ~150 LOC)

Extend `internal/types/effects.go`:
- Add an `effectSchema` structure: `map[string]map[string]map[string]struct{}` (effect → key →
  set-of-values), or an equivalent typed struct. Keep `defaultEffectModes` / `DefaultModeFor`
  functioning (either derive defaults from the schema or keep both consistent with a compile-time
  cross-check test).
- Register `Rand` and `AI` per the frozen schema table above.
- Add `validateEffectParams(effectName string, params map[string]string, span) error` returning
  one of the three structured errors (or nil). The error **text must embed the code verbatim** plus
  the fix hint + legal-value list, e.g.:
  ```
  EFF_UNKNOWN_MODE: effect 'Rand' has no mode 'banana'. Allowed modes: os, seeded, crypto.
    Fix: use one of the allowed modes, or drop the parameter for the default (mode=os).
  ```
- Wire the call into `ElaborateEffectRowWithBudgets` at the point `eff.Params` are captured
  (effects.go:264-270), before defaults are applied. Return the error (propagates via
  typechecker_functions.go:104 as a diagnostic).

**Acceptance criteria:**
- `EFF_UNKNOWN_MODE`, `EFF_UNKNOWN_PARAM_KEY`, `EFF_PARAMS_NOT_SUPPORTED` are distinct and each
  carries a fix hint + (for the first two) the legal-value/key list.
- `EFF_PARAMS_NOT_SUPPORTED` names the tracking doc `m-effect-clock-net-fs-modes`.
- Schema/defaults consistency test: every `defaultEffectModes` entry's (key,value) is a member of
  the schema.
- `go test ./internal/types/ -count=1` green (existing effect_params/effects tests unchanged).

**Test commands:**
```bash
go test ./internal/types/ -count=1
go build -o /tmp/ailang-plan ./cmd/ailang   # or make build
```

### M2 — Unit tests: schema lookup + error shapes + legal matrix (~2h, ~120 LOC)

Add to `internal/types/effect_params_test.go` (or a new `effect_schema_test.go`):
- **Legal matrix**: every registered `(effect,key,value)` triple elaborates clean — `Rand[mode=os|
  seeded|crypto]`, `AI[mode=fixed|routeable|replay-only]`, `AI[scope=byok]`, and bare `Rand`/`AI`.
- **Error shapes**: `Rand[mode=banana]`→`EFF_UNKNOWN_MODE`; `Rand[flavor=hot]`→
  `EFF_UNKNOWN_PARAM_KEY`; `Clock[mode=pinned]`→`EFF_PARAMS_NOT_SUPPORTED`. Assert code + legal-list
  substring present.
- **Bridge regression** (satisfies doc §Conflict-Surface item 4): a row built via
  `stringSliceToEffectRow(["Rand"])` (nil Params) round-trips and triggers NO validation — proving
  the bridge/iface-cache back-compat path is untouched.

**Acceptance criteria:**
- All three error-class assertions pass; full legal matrix passes.
- Bridge/back-compat regression passes.
- `go test ./internal/types/ ./internal/pipeline/ -count=1` green.

**Test commands:**
```bash
go test ./internal/types/ ./internal/pipeline/ -count=1
```

### M3 — Diagnostic-fixture CI promotion (~1h, ~40 LOC)

Promote the 3 error classes to the shared harness (as sibling M-DIAG-FIXTURE-PROMOTION did):
- Add 3 rows to `footgunFixtures` in `internal/diag/footgun_fixtures_test.go` with
  status `"shipped-this-sprint"`: each `{name, code, fix, src, status}` where `code` is the
  `EFF_*` string and `fix` is a stable substring of the fix hint. `src` is a minimal
  `module benchmark/solution` snippet (use `pipeline.Run(ModeCheck, RelaxModules:true)` — no
  `Filename`, per the harness's in-memory contract).
- Add the corresponding rows to `internal/diag/footguns.md`.

**Acceptance criteria:**
- `go test ./internal/diag/ -count=1` green (3 new fixtures + pre-existing rows).
- Each fixture asserts both the `EFF_*` code and the fix substring appear in `ailang check` output.

**Test commands:**
```bash
go test ./internal/diag/ -count=1
```

### M4 — Docs truth-up + teaching prompt + example (~1h, ~30 LOC)

- **Guide** `docs/docs/guides/parameterised-effects.md`: remove the interim "Accuracy note"
  block (§155-159); verify the "Mode set is closed" section (§150) now reads as *shipped* fact
  (adjust tense/wording so it no longer implies future work).
- **Teaching prompt**: mention the three error codes (`EFF_UNKNOWN_MODE`, `EFF_UNKNOWN_PARAM_KEY`,
  `EFF_PARAMS_NOT_SUPPORTED`) so AI repair loops key on them. Coordinate via the `prompt-manager`
  skill; edit the current prompt (do NOT retro-edit frozen `prompts/v0.16.x.md`).
- **Negative example** (optional but recommended per CLAUDE.md "every feature needs an example"):
  the existing `examples/modal_rand.ail` already covers the legal Rand surface; add a short comment
  block or a `docs/`-hosted snippet documenting the three rejected forms (do NOT add a
  `.ail` file that fails `make verify-examples`).
- **CHANGELOG.md**: add an entry under the current changelog (`changelogs/v0.18-current.md`) noting
  the three new diagnostics + the closed-set enforcement, flagged as a (pre-1.0, Experimental-surface)
  breaking narrowing.

**Acceptance criteria:**
- Guide interim note removed; "Mode set is closed" verified accurate post-sprint.
- Teaching prompt references the three codes.
- CHANGELOG entry present.
- `make verify-examples` green (no in-tree program uses an illegal param — confirmed by sweep below).

**Test commands:**
```bash
make verify-examples
grep -rn "mode=\|scope=\|flavor=" std/ examples/ benchmarks/ prompts/    # sweep re-run
```

---

## In-tree Sweep (verified 2026-07-11)

`grep -rn "mode=\|scope=\|flavor="` over `std/ examples/ benchmarks/ prompts/`. Every occurrence of
an effect **param** (as opposed to prose) uses a **legal** value under the frozen schema:

| File | Params used | Legal? |
|------|-------------|--------|
| `examples/modal_rand.ail` | `Rand[mode=os]`, `Rand[mode=seeded]` (crypto only in comments) | ✅ all legal |
| `examples/ai_modes.ail` | `AI[mode=fixed]`, `AI[mode=routeable]` | ✅ all legal |
| `benchmarks/openrouter_cost_compare/scenario.md` | `AI[mode=routeable]` (prose only) | ✅ n/a (doc) |
| `prompts/v0.16.{0,1,2}.md` | `Rand[mode=os\|seeded]`, `AI[mode=fixed\|routeable]` (prose) | ✅ all legal (frozen prompts) |

**No in-tree program uses an illegal param.** `make verify-examples` is expected green pre- and
post-sprint. No fixes required. (Note: `std/` has no `.ail` param usages; the grep `std/` hits were
comment/prose only.)

---

## Success Metrics (from design doc §Success Criteria)

- [x] Schema registers Rand + AI surfaces exactly as shipped in v0.15.0 (frozen table above)
- [x] Unknown value / unknown key / schema-less effect param → 3 distinct structured errors w/ fixes
- [x] All legal v0.15.0 forms type-check unchanged; `make verify-examples` green
- [x] Bridge/iface-cache round-trip regression (nil Params → no validation) passes
- [x] Guide "Mode set is closed" verified accurate; interim note removed
- [x] Teaching prompt mentions the three error codes
- [x] `make test && make lint` green

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `EFF_*` code not embedded in error text → footgun harness `strings.Contains` red | Med | M1 acceptance requires code embedded verbatim; M3 fixture asserts it end-to-end |
| Out-of-tree code (motoko fork, packages) uses speculative params | Med | Stability-page Experimental; error names tracking doc; CHANGELOG breaking-narrowing entry |
| Schema/defaults drift | Low | M1 consistency test asserts every default is a schema member; Phase-5 ports edit the same table |
| Editing a frozen prompt by mistake | Low | M4 explicitly edits the *current* prompt only, via prompt-manager |

---

## Executor Notes

- Concurrent agents share this worktree; `git status` before any commit, never checkout/stash/reset.
- The param-carrying chokepoint is `ElaborateEffectRowWithBudgets` (effects.go:225) — validate there,
  NOT in the parser (parser accepts any bare-ident key/value by design) and NOT in
  `stringSliceToEffectRow` (carries no params — see discrepancy D1).
- Footgun fixtures drive `pipeline.Run(Config{Mode: ModeCheck, RelaxModules: true}, Source{Code:...})`
  with no `Filename`; assert on `err.Error()` substrings.
- Do NOT add a `.ail` example that fails `make verify-examples`; document rejected forms in prose.
