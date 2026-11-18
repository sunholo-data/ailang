# M-AGENTS-FIXTURE-GPT62-SOL-MIGRATION: Record the 2026-11-17 planner/executor head migration in the agents-fixture ledger

**Status**: Planned
**Target**: v0.60.5
**Priority**: P0 (High — dev CI red on every push since 2026-11-17; all three test jobs fail)
**Estimated**: 1–2 hours (single test-file edit + one comment + CI verification)
**Dependencies**: None. The ratified registry change this doc records is already landed on `dev`.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This change touches one Go test file's ratification ledger and one YAML comment. It changes no
language semantics, no runtime path, and no shipped behavior.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No runtime behavior change; test-only |
| A2: Replayability | 0 | No trace surface touched |
| A3: Effect Legibility | 0 | No effects involved |
| A4: Explicit Authority | +1 | The ledger IS explicit authority: a role head may only move from its recorded predecessor with a cited ratification. This doc exercises that authority instead of bypassing it |
| A5: Bounded Verification | 0 | Test scope unchanged |
| A6: Safe Concurrency | 0 | No concurrency surface |
| A7: Machines First | +1 | The failing message already names agent, role, actual, table head, and expected — a machine (or agent) can act on it directly, as this doc does |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | No cost surface (the $2/$10 pricing parity of the migration is recorded in models.yml) |
| A10: Composability | 0 | Test-local change |
| A11: Structured Failure | +1 | The red CI is the drift alarm working as designed; the fix feeds the alarm its missing record rather than muting it |
| A12: System Boundary | 0 | No boundary crossing |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted — the opposite: a ratification is made explicit
- [x] A7 (Machines First): not optimizing for human convenience over machine analysis

## Problem Statement

Dev has been red since 2026-11-17. CI run 44158236421 (job 134304679657, step "Run tests with
timeout (M-DX11)") on commit `022f68f1a` fails `TestCloudAgents_RegistryMatchesTheDeletedRoutingTable`
(`internal/coordinator/agents_fixture_test.go:151`), and `test-windows` and the Build matrix fail
the same way on the same push. First red commits: `6df452df6` (feat(modelreg): add gpt6-2-astra,
gpt6-2-sol) or `8e04a29c5` (fleet defaults upgraded to gpt-6.2-sol); parent `5cf3088a0` is green.

**Root cause (verified by reading the code, not inferred):**

1. On 2026-11-17 the fleet's `planner` and `executor` role heads moved from `gpt-6.1-sol` to
   `gpt-6.2-sol`. The registry records the ratification (models.yml, `gpt6-2-sol` row):
   *"ROLES since 2026-11-17 (Mark, attended: 'upgrade our defaults to these'): planner + executor
   head, controller/designer codex rung, design-quorum OpenAI seat (codex:gpt-6.2-sol),
   benchmark_suite seat — all moved from gpt6-1-sol."* The `role_routing:` rows now read
   `planner: [gpt6-2-sol, …]` and `executor: [gpt6-2-sol, …]`.
2. `agents_fixture_test.go` is the M-MODEL-REGISTRY-SINGLE-SOURCE M5+M7 arm: it proves the
   registry resolves every cloud agent exactly as the (deleted) `model_routing` table did, with
   exactly one sanctioned escape hatch — the `ratifiedRoleMigrations` map, whose contract reads
   *"Add a row only with the commit that ratified it."*
3. That map was last updated 2026-09-30 (`gpt-6-sol → gpt-6.1-sol`). The 2026-11-17 migration
   landed in the registry but **no ledger row was added**, so `expectedRoleHead("gpt-5.6-sol")`
   still walks the chain to `gpt-6.1-sol` while the registry resolves the roles to `gpt-6.2-sol`.
4. All 34 fixture agents carry explicit pins, so the first loop's role path never fires
   (`checkedRoles == 0`) and the test exercises the role path in its second, pin-dropped block:
   `sprint-planner` (role `planner`) and `sprint-executor` (role `executor`) resolve to
   `gpt-6.2-sol`, expected `gpt-6.1-sol` → fail at line 151. `design-doc-creator` (designer) and
   `sprint-evaluator` (evaluator) pass because their heads deliberately kept the same model when
   the harness rungs moved (models.yml records this inertness decision next to the rows).

**Impact:** every push to `dev` fails all three CI test jobs, masking any NEW regression behind a
known-red. The fleet's actual model routing is unaffected (the cloud agents run their pins), but
the M7 "inert by construction" measurement is out of sync with the ratified state.

## Goals

**Primary Goal:** Make `TestCloudAgents_RegistryMatchesTheDeletedRoutingTable` green again by
recording the 2026-11-17 ratified `gpt-6.1-sol → gpt-6.2-sol` role-head migration in the
fixture's ratification ledger — restoring the ledger's invariant that every deliberate head
movement is recorded with its ratification.

**Success Metrics:**
1. `go test ./internal/coordinator/ -run 'TestCloudAgents' -count=1` passes.
2. All three CI jobs on the next `dev` push are green: "Run tests with timeout (M-DX11)",
   `test-windows`, and the Build matrix.
3. The ledger row cites the ratification (date, who, the ruling) in the same format as the two
   existing rows.
4. `TestCloudAgents_NoAgentReliesOnAProviderDefault` and `TestChainHeadMatchesResolveModel`
   remain green (no behavior change reaches them).
5. The point-of-change friction is reduced: a future ratifier changing a role head in
   `models.yml` is told, at the edit site, that the ledger row must accompany it.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Update the fixture ledger (add the row) rather than revert the registry change | The registry change is a ratified, measured fleet upgrade; reverting it would put the registry out of sync with the ruling the ledger exists to record | human (ratified by Mark 2026-11-17; this doc ratifies the recording) | design | low |
| D2: Keep the ledger hand-maintained (do NOT machine-derive it from models.yml) | A derived ledger makes the test compare the registry against a record the registry authors — the independence is the point of the gate | human (this doc) | design | high (if reversed later) |
| D3: Add a pointer comment at the `role_routing:` section in models.yml naming the ledger | The 2026-11-17 migration is the first to land without its row; the pointer puts the requirement where the ratifier edits | agent (this doc proposes; implementer may refine wording) | design | low |
| D4: Exact commit cited in the ledger row comment | The two existing rows cite the ratifying commit; `6df452df6`/`8e04a29c5` are the candidates named by CI triage | agent (verify against git history at implementation) | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D1 — update ledger, not revert (ratified here; see Solution Design → Options Considered)
- [x] D2 — ledger stays hand-maintained (see Non-Goals)
- [ ] D3 — pointer comment wording reviewed at implementation (agent latitude on phrasing, not on presence)
- [ ] D4 — exact ratifying commit confirmed from git history (`git log` on the red commits)

## Solution Design

### Overview

Add one entry to `ratifiedRoleMigrations` in `internal/coordinator/agents_fixture_test.go`:

```go
// 2026-11-17 (attended, Mark: "upgrade our defaults to these"): GPT-6.2 Sol
// replaces GPT-6.1 Sol on the planner and executor heads — same $2/$10 list,
// codex subscription lane verified on codex-cli 0.169.0. Ratified in the
// gpt6-2-sol registry row; landed with 6df452df6/8e04a29c5.
"gpt-6.1-sol": "gpt-6.2-sol",
```

`expectedRoleHead` already follows chains of unbounded length up to
`len(ratifiedRoleMigrations)` hops, so the new 4-entry map correctly walks
`gpt-5.6-sol → gpt-6-sol → gpt-6.1-sol → gpt-6.2-sol` (3 hops ≤ 4). No change to
`expectedRoleHead`, the fixture JSON, or the test logic is needed or wanted: **the red CI is the
drift alarm firing correctly; the fix is the missing ledger entry, not a test change.**

Second, add a short comment above the `role_routing:` rows in `internal/modelreg/models.yml` (the
point of change for every future role-head migration):

```yaml
  # ROLE-HEAD CHANGES ALSO REQUIRE A LEDGER ROW: after changing any head below,
  # add a hop to `ratifiedRoleMigrations` in internal/coordinator/agents_fixture_test.go
  # with the ratification citation — TestCloudAgents_RegistryMatchesTheDeletedRoutingTable
  # fails on dev CI until you do (it did, 2026-11-17 → 2026-11-18).
```

### Options Considered

**Option A — add the ledger row (CHOSEN).** The map's own comment is the specification of this
fix: *"this map is the only way a role may legitimately resolve somewhere else… Add a row only
with the commit that ratified it."* The migration is ratified, measured (placement 2026-11-17:
102/105, anchored standard ELO 2432.4, same $2/$10), and lane-verified (codex-cli 0.169.0,
rc=0; nonexistent-model control still 400s). Recording it is the designed path.

**Option B — revert the registry to gpt6-1-sol (REJECTED).** This would make CI green by
desyncing the registry from the ratified ruling: the fleet would run GPT-6.1 Sol heads against
Mark's explicit 2026-11-17 decision ("upgrade our defaults to these"). It also re-opens the
benchmark_suite and design-quorum seats that moved in the same ruling. A green CI bought by
undoing a ratified upgrade is worse than the red it removes.

**Option C — machine-readable migration ledger in models.yml consumed by the test (REJECTED).**
See D2 / Non-Goals. It would turn the independent gate into a tautology and add modelreg parser
surface for negative independence value.

### Architecture

**Components:**
1. **The ledger** (`ratifiedRoleMigrations`, agents_fixture_test.go): the single sanctioned record
   of post-2026-08-27 role-head migrations. One new entry. Consumed only by
   `expectedRoleHead`, which is consumed only by this test file (grep-verified).
2. **The point-of-change pointer** (models.yml `role_routing:` comment): documentation at the
   site future ratifiers edit; no semantic effect.
3. **CI**: nothing new — the existing three jobs re-run the test and must go green.

### Implementation Plan

**Phase 1: The ledger row** (~15 min)
- [ ] Confirm the ratifying commit from git history (`git log --oneline --all | grep -E '6df452df|8e04a29c5'`, or `git show` each; if the squash history on `dev` names only one, cite that one)
- [ ] Add the `"gpt-6.1-sol": "gpt-6.2-sol"` row with the citation comment to `ratifiedRoleMigrations`

**Phase 2: The pointer comment** (~15 min)
- [ ] Add the role-routing → ledger pointer comment in `internal/modelreg/models.yml` above the `designer:`/`planner:` rows

**Phase 3: Verification** (~30 min)
- [ ] `go test ./internal/coordinator/ -run 'TestCloudAgents' -count=1` — green
- [ ] `go test ./internal/coordinator/ -count=1` — green (chain-head and retry-chain arms untouched)
- [ ] `go test ./internal/modelreg/... ./internal/mission/quorum/...` — green (comment-only YAML change)
- [ ] `make fmt` / `go vet` on touched files
- [ ] Push to `dev`; confirm run on `022f68f1a`'s successor is green on all three jobs

### Files to Modify/Create

**New files:** none.

**Modified files:**
- `internal/coordinator/agents_fixture_test.go` — one map entry + citation comment (~5 LOC)
- `internal/modelreg/models.yml` — pointer comment above `role_routing` rows (~4 comment LOC; zero semantic change)

## Examples

### Example 1: The failing assertion, before and after

**Before** (current CI red, line 151):
```
agent "sprint-planner" role "planner" WITHOUT its pin: registry says "gpt-6.2-sol",
the deleted table said "gpt-5.6-sol" (expected "gpt-6.1-sol" after ratified
migrations) — M7 is not inert on the role path
```
(`sprint-executor` fails identically for role `executor`.)

**After:**
```
34 agents checked; 4 role resolutions verified against the deleted table
```
`expectedRoleHead("gpt-5.6-sol")` now returns `gpt-6.2-sol`, matching the registry.

### Example 2: The ledger entry pattern (existing rows for reference)

```go
var ratifiedRoleMigrations = map[string]string{
	// 045bb3ed3 (2026-09-22, attended): fleet migration to GPT-6 Sol …
	"gpt-5.6-sol": "gpt-6-sol",
	// 2026-09-30 (attended): GPT-6.1 Sol replaces GPT-6 Sol on the same heads.
	"gpt-6-sol": "gpt-6.1-sol",
	// <NEW> 2026-11-17 (attended, Mark: "upgrade our defaults to these"): GPT-6.2 Sol …
	"gpt-6.1-sol": "gpt-6.2-sol",
}
```

## Success Criteria

- [ ] `TestCloudAgents_RegistryMatchesTheDeletedRoutingTable` green locally (`go test ./internal/coordinator/ -run 'TestCloudAgents' -count=1`) — acceptance: exit 0, "4 role resolutions verified"
- [ ] All three CI jobs green on the fix push (M-DX11 tests, test-windows, Build matrix) — acceptance: no red jobs on the push
- [ ] Ledger row cites date + ruling + commit, matching the two existing rows' format — acceptance: row comment contains "2026-11-17" and the ratifying commit hash
- [ ] `TestCloudAgents_NoAgentReliesOnAProviderDefault` still green — acceptance: same test run, exit 0
- [ ] No change to `models.yml` semantics, fixture JSON, `expectedRoleHead`, or any non-test Go code — acceptance: `git diff` touches only the two files listed, and the models.yml diff is comment-only
- [ ] All tests passing (`make test`)
- [ ] No registry defaults reverted (gpt6-2-sol stays the planner/executor head) — acceptance: `planner: [gpt6-2-sol, …]` unchanged in models.yml

## Testing Strategy

**Unit tests:** the fix IS a unit-test expectation update; the strategy is to run the full
coordinator package (not just the one test) to prove no neighbor arm moved:
`TestCloudAgents_NoAgentReliesOnAProviderDefault`, `TestChainHeadMatchesResolveModel`, and the
retry-chain tests all read the same registry state.

**Integration tests:** none new; CI matrix (linux/windows/build) is the integration surface.

**Manual testing:** none required. Note for the implementer: this workspace had no Go toolchain
at authoring time — the green run must come from the implementer's environment or CI; do not
mark the success criteria on inference.

## Verification Log

The design-doc-creator skill's hard gates applied (this doc makes no AILANG-language claims, so
the `ailang check` gate does not apply; every code/behavior claim below was verified by reading
the artifact named):

| # | Claim | Verification |
|---|-------|--------------|
| V1 | The 2026-11-17 planner/executor head move to gpt-6.2-sol was ratified, attended | `sed -n '783,786p' internal/modelreg/models.yml`: "ROLES since 2026-11-17 (Mark, attended: 'upgrade our defaults to these'): planner + executor head … — all moved from gpt6-1-sol"; corroborated by `internal/mission/quorum/codex_reviewer.go:39` ("gpt-6.1-sol -> gpt-6.2-sol 2026-11-17 (Mark, attended)") |
| V2 | The registry role heads are gpt6-2-sol for planner and executor | `grep -n 'planner:\|executor:' internal/modelreg/models.yml` → lines 6814/6834: `planner: [gpt6-2-sol, …]`, `executor: [gpt6-2-sol, …]`; independently confirmed by CI's own failure message ("registry says gpt-6.2-sol") |
| V3 | The ledger chain currently ends at gpt-6.1-sol (the bug) | Read `agents_fixture_test.go:42-49`: two entries, `gpt-5.6-sol→gpt-6-sol`, `gpt-6-sol→gpt-6.1-sol` |
| V4 | All 34 fixture agents carry pins, so the role path fires only in the pin-dropped block, failing at line 151 | Fixture read programmatically: 34 agents, `without pin: []`; line 151 is the second block's `t.Errorf` (sed-verified line numbers 140–170); matches CI's report of sprint-planner + sprint-executor only |
| V5 | designer/evaluator heads resolve to the same model the deleted table named (only planner/executor fail) | models.yml lines 6802–6806 record the deliberate model-identical harness move "keeps the CLOUD side inert"; CI reports only the two agents failing |
| V6 | `expectedRoleHead` follows chains long enough after the new row (negative: no loop-bound change needed) | Read `expectedRoleHead`: `for hops := 0; hops < len(ratifiedRoleMigrations); hops++` — bound grows with the map; 3 hops needed ≤ 4 entries |
| V7 | `ratifiedRoleMigrations`/`expectedRoleHead` are consumed ONLY by agents_fixture_test.go (negative-existence: no other consumer to update) | `grep -rn "ratifiedRoleMigrations\|expectedRoleHead" --include="*.go" .` → hits only in agents_fixture_test.go |
| V8 | No other test encodes planner/executor role-head model expectations (negative-existence: this is the only red) | `grep -rn "gpt-6.1-sol\|gpt-6-sol\|gpt-5.6-sol" --include="*_test.go" internal/ cmd/` → all other hits are chatgpt-lane/cost/preflight tests about the model itself, not role heads; CI corroborates: exactly one test red |
| V9 | The fixture is a faithful frozen snapshot with a 34-agent guard | Read `loadCloudFixture`: `len(f.Agents) != 34` fatals with "re-snapshot and say so in the commit" — so the fix must NOT touch the JSON (it doesn't) |
| V10 | Priors landed WITH their ledger rows (this is the first miss) | The two existing rows each cite their ratifying commit/date (045bb3ed3, 2026-09-30); the 2026-11-17 migration has no row (V3). Shallow clone: prior episodes of red-CI-before-row could not be checked from history — not claimed |

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Pointer comment wording (D3)** — agent may refine the phrasing; the requirement that it name
  the test file and the ledger map is fixed.
- **Exact ratifying commit cited in the row (D4)** — agent resolves from git history at
  implementation; CI triage named `6df452df6` and `8e04a29c5` as the red commits, and the row
  should cite the one that actually carried the role-head change (or both, with one line each
  saying which).
- **Whether to also run `make check-boundaries`** — no boundary is crossed (test + comment), but
  the implementer may run it for free as part of the verification pass.

## Non-Goals

**Not attempted in this feature:**
- **Reverting the registry to gpt-6.1-sol** — would desync the fleet from the 2026-11-17 ruling
  (Option B, rejected above).
- **Machine-deriving the ledger from models.yml** (Option C) — the ledger's independence from the
  registry is the entire point of the gate; a registry-authored ratification record is
  self-certification. Also: `expectedRoleHead`'s hop bound and the map's "add a row only with the
  commit that ratified it" contract are deliberately human-paced.
- **Migrating the mission-quorum model allowlist** — already done in
  `internal/mission/quorum/rolemodels.go` (gpt-6.2-sol allowed, gpt-6.1-sol retained while
  codex-cli 0.159.2 cloud images 400 it; tracked there, out of scope here).
- **Re-snapshotting the fixture** — the 34-agent snapshot is a faithful frozen record with an
  explicit re-snapshot guard (V9); this migration is exactly what the ledger exists for.
- **Fixing the codex-cli image lag (0.159.2 vs 0.169.0)** — operational, tracked in
  rolemodels.go; the test asserts model names, not lane availability.

## Timeline

**Total: ~1–2 hours, single sitting** (this is a red-CI hotfix-scale change; the phase breakdown
above is the whole plan — no multi-week schedule applies).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Next role-head migration again lands without its ledger row, re-redding CI fleet-wide | Med | The models.yml pointer comment (D3) puts the requirement at the edit site. If this bites again after the pointer lands, that is the second friction → per the friction-twice doctrine it graduates to an extension (e.g. a pre-push check), to be designed then, not now |
| Ledger row cites the wrong commit (D4) | Low | Row is comment-only; a wrong hash is a one-line fix, and the date + ruling text are unambiguous |
| gpt-6.2-sol unusable on codex-cli 0.159.2 cloud images (lane not test-asserted) | Med (operational, not this test) | Already tracked in rolemodels.go (gpt-6.1-sol stays allowed until 0.169.0 deploys); this doc changes no execution routing |
| Hidden consumer of the fixture JSON expecting gpt-6.1-sol | Low | V7/V8: fixture consumed only by this test file; no other test encodes role-head expectations; CI shows exactly one red test |

## Related Documents

**Implemented (may inform design):**
- [m-model-registry-single-source.md](../../implemented/v0_35_0/m-model-registry-single-source.md) — the M5+M7 decisions this fixture test guards (D2(a)'s precondition, the deleted-table inertness measurement); its M7 acceptance line "model_routing deleted with no behavior change (fixture test: pre/post resolution identical for all 34 agents)" is the invariant the ledger serves
- [m-pipeline-reconciliation.md](../../implemented/v0_35_0/m-pipeline-reconciliation.md) — M5 built the `model_routing` table that became the frozen snapshot

**Planned (check for overlap):**
- (none found — doc search on "sol migration", "role migration fixture registry routing table" returned no planned overlap; the nearest planned docs are unrelated)

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- CI evidence: run 44158236421, job 134304679657, step "Run tests with timeout (M-DX11)", commit `022f68f1a`; first red `6df452df6`/`8e04a29c5`, parent `5cf3088a0` green
- Ratification record: `internal/modelreg/models.yml`, `gpt6-2-sol` row, "ROLES since 2026-11-17"

## Future Work

- **Friction-twice graduation:** if another role-head migration lands without its ledger row
  AFTER the pointer comment ships, design a mechanical pre-merge guard (extension or CI step)
  rather than a third reminder comment.
- **M8 parked work:** re-deciding WHICH model runs a role is explicitly parked per
  models.yml's role_routing block; this doc does not reopen it.

---

**Document created**: 2026-11-18
**Last updated**: 2026-11-18
