# Sprint Plan — M-BUILTIN-CLASSIFICATION-SURFACE

**Design doc:** [m-builtin-classification-surface.md](m-builtin-classification-surface.md)  
**Sprint ID:** `M-BUILTIN-CLASSIFICATION-SURFACE`  
**Issue:** GitHub #901  
**Target:** next release after v0.42.0 · P1  
**Duration:** **2 engineering days / about 16 hours**  
**Estimated change:** **~760 LOC** (about 310 implementation, 390 tests, 60 generated docs/help)  
**Risk:** Medium (the current builtin truth is split across two registries and prelude setup)

## 0. Scope and binding decisions

This sprint implements the approved producer-visible inventory; it does not change AILANG language
semantics or add a second hand-maintained builtin list.

- `ailang iface --builtins --json` is the stable machine contract; `--builtins` without `--json`
  renders the same inventory as a deterministic human-readable table.
- The JSON envelope has an explicit schema/format version and the linked binary version. Entries are
  sorted by name and contain name, arity, class, capability where applicable, and stdlib target where
  applicable.
- The inventory reconciles the live lightweight `internal/builtins.Registry`, the richer
  `specRegistry`, and the prelude names installed by the evaluator. It must reject inconsistent
  duplicate metadata. Neither existing registry is assumed complete on its own.
- `show` and `toText` classify as `pure`; `println` classifies as `effect` with capability `IO`.
  Shared prelude metadata must live below `internal/eval` so both evaluator setup paths consume it;
  the inventory must not import evaluator internals or scrape `env.Set` source text.
- The checked-in reference page is generated from the inventory. Existing
  `internal/iface/builtin_freeze.go` is not expanded or used as the inventory source.

## 1. Baseline and capacity

- The planning worktree was clean at `fc0d4c90`; `std/VERSION` is `v0.42.0`.
- The last seven days contain one design-doc commit and no useful implementation LOC signal.
  Therefore this plan uses the approved two-day estimate, inspected file scope, and a 20% integration
  allowance rather than claiming an unsupported LOC/day velocity.
- Current CLI parsing is in `cmd/ailang/commands_language.go`; current end-to-end iface tests are in
  `cmd/ailang/internal_dump_iface_test.go`. CLI help is centralized and must be updated through the
  conventions enforced by `cmd/ailang/help.go` and its tests.
- Current metadata is split between `internal/builtins/registry.go`, `internal/builtins/spec.go`, and
  duplicated prelude installation in `internal/eval/eval_simple.go` and
  `internal/eval/eval_typed_helpers.go`. This split is the principal schedule risk.

## 2. Milestone map

| Milestone | Outcome | Estimate | Depends on |
|---|---|---:|---|
| M0 | Green baseline and exact registration-set audit | 1h / 0 LOC | — |
| M1 | Canonical, validated, deterministic inventory API | 6h / 360 LOC | M0 |
| M2 | Public `iface --builtins` JSON/table surface and help | 4h / 230 LOC | M1 |
| M3 | Generated reference page and full acceptance proof | 5h / 170 LOC | M1, M2 |

## 3. Milestones

### M0 — Baseline and registration-set audit

**Files:** none.

Tasks:

1. Record `go version`, `git status --short`, and clean results for
   `go test ./internal/builtins/... ./internal/eval/... ./internal/iface/... ./cmd/ailang/...` and
   `make check-boundaries`.
2. Programmatically compare names in the lightweight registry, `AllSpecs()`, and both evaluator
   prelude setup paths. Record overlaps, missing names, and metadata disagreements before editing.
3. Confirm the linked version source used by `ailang version`; do not read `std/VERSION` at runtime.

Acceptance criteria:

- [ ] Baseline failures, if any, are distinguished from sprint regressions.
- [ ] The audit records counts and set differences for all live registration sources.
- [ ] No unrelated working-tree change is overwritten or absorbed into the sprint.

### M1 — Canonical inventory and totality gate

**Example files to update/create:** `internal/builtins/inventory.go` (new),
`internal/builtins/inventory_test.go` (new), `internal/builtins/prelude.go` (new),
`internal/eval/eval_simple.go`, and `internal/eval/eval_typed_helpers.go`.

Build:

- Define the versioned inventory envelope and entry types in `internal/builtins`, including an
  explicit class enum (`pure`, `effect`, `stdlib`) and optional capability/stdlib target fields.
- Add one canonical prelude descriptor set for `show`, `toText`, and `println`; refactor both
  evaluator setup paths to use it while preserving their existing implementations and behavior.
- Build inventory entries from the union of live registry/spec/prelude metadata, sorted by name.
  Validate arity/purity/effect agreement for overlaps; unknown effects, missing classifications,
  duplicate conflicts, or an empty inventory return errors rather than defaults.
- Define `stdlib` as the strongest class when a qualified stdlib target exists, while retaining the
  invariant that it contributes no effects.

Acceptance criteria:

- [ ] Unit tests prove deterministic byte-equivalent ordering across repeated builds.
- [ ] Every lightweight registry key, every `AllSpecs()` key, and each prelude descriptor occurs
  exactly once in the inventory.
- [ ] `show`/`toText` are pure and `println` is effect/IO; `intToFloat` and `floatToInt` are present.
- [ ] Table-driven negative tests reject missing class, unknown capability, conflicting overlap,
  malformed stdlib target, and empty inventory.
- [ ] A compiling mutation that removes one prelude descriptor or registry entry makes the named
  totality test fail; restoring it makes the test pass.
- [ ] Both evaluator paths retain focused behavior tests for the three prelude builtins.

### M2 — Producer-visible CLI surface

**Example files to update/create:** `cmd/ailang/commands_language.go`,
`cmd/ailang/builtin_inventory_output.go` (new),
`cmd/ailang/builtin_inventory_output_test.go` (new), `cmd/ailang/help.go`, and relevant help tests.

Build:

- Extend `iface` parsing with `--builtins` and `--json`. `--json` without `--builtins`, a module
  argument combined with `--builtins`, and unexpected positional arguments fail with actionable
  usage errors.
- Render JSON from the canonical inventory with exactly one trailing newline. Stamp the linked
  binary version and explicit inventory schema version.
- Render a stable table from the same entry values; do not maintain a parallel display list.
- Update centralized CLI help and usage text using cli-doc-maintainer conventions.

Acceptance criteria:

- [ ] Built-binary tests prove `ailang iface --builtins --json` parses as JSON and includes the full
  sorted inventory, schema version, binary version, and representative pure/effect/stdlib entries.
- [ ] `ailang iface --builtins` contains the same names and classifications as JSON.
- [ ] Invalid flag/argument combinations are non-zero, explain the contract, and emit no partial
  inventory to stdout.
- [ ] Existing module `ailang iface [--compact] <module>` golden behavior remains byte-identical.
- [ ] `ailang iface --help` and top-level help advertise both flags consistently.

### M3 — Generated docs and end-to-end acceptance

**Example files to update/create:** `tools/generate-builtin-reference.go` or the repository-standard
docs generator target, `docs/docs/reference/builtin-inventory.md` (generated), generator tests or a
golden fixture, `Makefile`, and `changelogs/v0.42-current.md` if present at execution time.

Build and verify:

- Add a deterministic docs-sync generation path that consumes the same canonical inventory API and
  emits the human reference page. Add a check mode/Make target that fails on generated-file drift.
- Document schema fields, class semantics, capability behavior, version-skew handling, and the rule
  that consumers must fail rather than default unknown names to pure.
- Exercise the installed/built binary as a motoko-style consumer: parse JSON, classify
  `show`/`intToFloat` as effect-free and `println` as requiring IO, and reject an injected unknown.
- Run formatting, focused tests, full test gate, lint, boundaries, and docs drift checks.

Acceptance criteria:

- [ ] Regeneration is byte-identical and a deliberate edit to the generated reference is detected.
- [ ] No authored builtin list exists in the generator, CLI renderer, or docs source; names come from
  the inventory API.
- [ ] The reference page is reachable under `docs/docs/reference/` and describes the JSON contract.
- [ ] Consumer acceptance proves the three required classifications and hard failure on unknowns.
- [ ] `make fmt`, focused Go tests, `make test`, `make lint`, `make check-boundaries`, and the docs
  generation check pass.

## 4. Risks and controls

| Risk | Control |
|---|---|
| Registry union exposes conflicting legacy metadata | M0 audits the sets; M1 rejects conflicts explicitly and tests them. |
| Refactoring prelude setup changes evaluator semantics | Shared descriptors carry metadata only; focused tests pin both evaluator paths. |
| Capability strings drift from runtime effects | Validate effect entries against the repository's known capability/effect set. |
| JSON becomes an accidental unversioned API | Versioned envelope, deterministic fields/order, and golden built-binary tests. |
| Generated docs become another stale copy | One generator over the inventory plus a CI-visible drift check. |
| CLI flags regress module iface behavior | Existing iface byte-golden tests stay binding and get explicit coexistence cases. |

## 5. Definition of done and handoff

- All M0–M3 acceptance criteria pass and the progress JSON has no placeholders.
- GitHub #901 is referenced during milestone commits and closed only by the final implementation
  commit/PR (`Fixes #901`).
- The static classifier can obtain a closed, versioned inventory from an installed binary without
  importing or scraping AILANG source.
- This plan does **not** authorize implementation. The user must approve it and explicitly say
  **“execute sprint”** before the `sprint-executor` skill begins.

