# Sprint Plan — M-MISSION-AGENTIC-ROUTING (remaining slice: M1b · M2 · M3)

**Design doc**: [m-mission-agentic-provider-routing.md](m-mission-agentic-provider-routing.md)
**Sprint JSON**: `.ailang/state/sprints/sprint_M-MISSION-AGENTIC-ROUTING.json`
**Planned by**: sprint-planner (Opus-pinned), mission iteration 31
**Branch base**: `dev` @ `d545d4a9e` · **Version**: v0.29.2
**Risk level**: medium (touches the loop's own driver + skills; deployment reality = launchd reads on-disk main checkout)

> **Scope**: M1a is LANDED (commit `8ee07ef23`, amended `d545d4a9e`) — enforcement + Anthropic
> per-role model pinning, driver-side verified live 2026-07-16 06:23. This plan covers the
> **remaining** milestones only: M1b (one non-Claude cross-provider executor), M2 (right-sizing
> table + evidence-row schema), M3 (planner down-tier A/B — parked with a concrete protocol).

---

## Velocity note

7-day repo velocity is dominated by data/eval-rotation churn (547 files, ~88k insertions — almost
all `eval_results/`, dashboard JSON, and OS-rotation data), so LOC/day is **not** a meaningful
signal for this sprint. This is **skill/driver text + a small Go-adjacent smoke path**, not feature
code. Sizing below is by *task shape* (skill edits, a probe test, a doc-schema change), not LOC.
Estimated executable slice: **~1.5 days**; M3 is parked as a protocol (see below), not executed
inline, because it needs 3 real quorum-reviewed docs across ≥2 loop iterations.

---

## Repo-verified anchors (all live-checked at `d545d4a9e`)

| Claim in design doc | Verification | Result |
|---|---|---|
| `provider_executor.go` registry has claude/codex/motoko/opencode/pi + managed_agents | `internal/coordinator/provider_executor.go:10-15` | ✅ confirmed |
| DryRun path exists (token-free smoke) | `provider_executor.go:72-77` (`if opts.DryRun { … "DRY RUN: Would execute %s …" }`) | ✅ confirmed |
| codex executor registers `"codex"` | `internal/executor/codex/codex.go:777-783` (`Register()` → `GlobalFactory().Register("codex", …)`, `init()`) | ✅ confirmed |
| codex drives `codex exec --json --model <m>` | `internal/executor/codex/codex.go:135-146` | ✅ confirmed |
| codex CLI installed | `which codex` → `/opt/homebrew/bin/codex`; `codex exec` subcommand present | ✅ confirmed |
| `OPENAI_API_KEY` set | env probe → present | ✅ confirmed |
| codex models exist in harness | `internal/eval_harness/models.yml` — gpt-5.x families, `agent_cli: "codex"` (e.g. gpt-5.6-sol `models.yml:201`, gpt-5.2-codex `:344`, gpt-5.1-codex-max `:385`) | ✅ confirmed |
| Gate 3 already *documents* the `provider:model` branch | `.claude/skills/mission-control/SKILL.md:163-164` (`a provider:model value (e.g. codex:gpt-5.6) … signals cross-provider routing via provider_executor`) | ⚠️ documented as a **concept only — no concrete spawn recipe**. This is the M1b gap. |
| Executor sub-agent pins model from env | `.claude/skills/sprint-executor/SKILL.md:343-348` (`Task(subagent_type="general-purpose", model=os.environ.get("MISSION_EXECUTOR_MODEL","opus"))`) | ✅ confirmed — this is the **Agent-tool alias** path (M1a); it does NOT understand `provider:model`. |
| Driver exports role env vars | `tools/launchd/mission-control.sh:186-189` (`MISSION_DESIGNER/PLANNER/EXECUTOR/EVALUATOR_MODEL`) | ✅ confirmed |
| Routing-evidence row schema (current) | `design_docs/v1-mission-log.md:11-12` (`model=<m> task-class=… round1-score=<n> rounds=<n> corrections=<n>`) | ✅ confirmed — **no provider/agent/cost columns yet** (M2 gap) |
| Charter routing table | `design_docs/v1-mission.md:153-181` ("Model routing policy") | ✅ confirmed — has the model table + evidence rule; **lacks the (provider, agent, tier) right-sizing table** (M2 gap) |

### Discrepancies / premises the plan re-scoped

1. **The doc frames M1b as "wire a `provider:model` executor through `provider_executor.go`" — but
   `provider_executor.go` needs NO code change.** The registry, DryRun, and codex executor already
   exist (v0.22.0). M1b is a **skill-orchestration** change: teach mission-control Gate 3 to *branch*
   a `provider:model` env value to a `codex exec`-driven executor run instead of the Agent tool. The
   design doc's own Conflict Surface already says `provider_executor.go` is a "read-only consumer —
   additive." Plan treats it as **zero Go plumbing** (constraint #2 satisfied by reuse).
2. **There is no CLI verb that runs a *single* agentic executor task headlessly** (`ailang eval` runs
   the benchmark harness, not an arbitrary mission directive). So the Gate-3 cross-provider branch
   must shell out to `codex exec` directly (the same binary the executor package drives) OR run a
   one-benchmark `ailang eval` probe. For the **acceptance smoke** we use the token-free / trivial
   path; for a real iteration the branch calls `codex exec` against the sprint worktree. See M1b tasks.
3. **codex is the preferred M1b provider** (per Gate-2 environment facts: codex CLI + OPENAI_API_KEY
   present; motoko needs the GPU `rig.lock`, which is OUT of scope per constraint #4). Plan pins M1b
   to codex and leaves motoko as a documented future lane.
4. **M3 cannot execute inside one iteration** — it requires the *same 3 quorum-reviewed docs* planned
   by a mid-tier planner vs Opus, then executor round-1 scores compared. That is ≥3 real sprints of
   evidence (the charter's ≥3-datapoint rule). Forcing it into one iteration would violate constraint
   #5 (sprint-sized) and the evidence rule. **M3 is planned as a PARKED milestone with a concrete
   A/B protocol** (below), to be executed as evidence accrues.

---

## Deployment reality (HARD CONSTRAINT #1 — encoded as an explicit milestone step)

launchd runs `tools/launchd/mission-control.sh` from the **on-disk main checkout**
(`/Users/voightkampff/dev/sunholo-data/ailang`). It reads the SKILL + driver **as they are on
disk**, not from a PR branch. Therefore any M1b/M2 edit to `.claude/skills/mission-control/SKILL.md`,
`.claude/skills/sprint-executor/SKILL.md`, `tools/launchd/mission-control.sh`, or the charter that
is merged only to `origin/dev` via PR **does NOT reach the loop until the main checkout is synced**.

**Deploy step (every skill/driver/charter milestone), Critical-Principle-0-safe:**

```bash
# From the MAIN checkout (not a worktree). Verify no uncommitted work would be clobbered.
git -C /Users/voightkampff/dev/sunholo-data/ailang status --porcelain    # MUST be clean-or-known
# If clean and on dev: fast-forward only. NEVER reset --hard / checkout over dirty tree.
git -C /Users/voightkampff/dev/sunholo-data/ailang pull --ff-only origin dev
# Confirm the on-disk skill now contains the change:
grep -n "provider:model" /Users/voightkampff/dev/sunholo-data/ailang/.claude/skills/mission-control/SKILL.md
```

If the main checkout has uncommitted changes at deploy time → **STOP and ask the human** (do not
stash-then-switch). Because this sprint edits the loop's own skills, the executor SHOULD do these
edits directly on `dev` in the main checkout (as M1a was) rather than a worktree that never lands
on disk — a worktree-only merge is invisible to launchd until the pull above runs.

---

## Milestone M1b — one NON-CLAUDE (codex) cross-provider executor via Gate 3 (~1 day)

**Goal**: mission-control Gate 3 branches a `provider:model` env value (e.g.
`MISSION_EXECUTOR_MODEL="codex:gpt-5.6-sol"`) to a `codex exec`-driven executor run against the
sprint worktree, instead of the in-session Agent tool. Acceptance: an iteration where **controller
and executor run on different PROVIDERS**.

**No Go plumbing** — reuse `provider_executor.go` + the codex executor. Skill-orchestration + driver
env only.

### Tasks

1. **Gate 3 spawn recipe for the `provider:model` branch** (edit
   `.claude/skills/mission-control/SKILL.md`, at the existing lines 160-166 where the concept is
   already noted). Add a concrete, bounded recipe:
   - Parse the role env: if value matches `^([a-z_]+):(.+)$` → `PROVIDER=$1`, `MODEL=$2` → cross-provider.
   - For `PROVIDER=codex`: run `codex exec` from inside the sprint worktree with the sprint
     directive (invoke sprint-executor's skill contract as the prompt), a **bounded** wall-clock cap
     (≤30 min, `date +%s` deadline per Standing rule 6), model `$MODEL`. Capture the worktree branch
     + report exactly as the claude executor path does (the executor commits to its branch; the
     controller reads the diff via `git -C <worktree> diff`).
   - **generator≠judge guard (constraint #3)**: assert `PROVIDER` of the evaluator env ≠ `PROVIDER`
     of the executor env before spawning the evaluator; if equal, re-route the evaluator to a
     distinct provider (fable/gemini) and FLAG it.
   - Fallback rule (already present at SKILL.md:165): if codex is unavailable / errors, fall back to
     `$MODEL` via the Agent tool for that role and FLAG in the Gate-5 report — never wedge the loop.
2. **Token-free smoke test (constraint #6)** — a `codex exec` **probe** that proves the branch wires
   end-to-end without burning real coding tokens:
   - Trivial probe: `codex exec --model gpt-5.6-sol 'reply with exactly: ok'` (mirrors the driver's
     own Anthropic probe at `mission-control.sh:102`) with a short timeout — proves auth + model +
     CLI reachability for ~1 token.
   - Document the DryRun option: `provider_executor.go:72-77` returns `"DRY RUN: Would execute codex
     with directive: …"` with **zero** executor calls — usable by any Go-level test/harness invoking
     the provider. (The skill path shells `codex exec` directly, so the probe above is its analogue.)
   - Add the probe as the pre-flight the Gate-3 codex branch runs BEFORE the real executor call: if
     the probe fails, fall back immediately (don't spend the full directive against a dead endpoint).
3. **Driver default note** (edit `tools/launchd/mission-control.sh` comment block near :186-189):
   document that `MISSION_EXECUTOR_MODEL` accepts either an Agent alias (`opus`) OR a `provider:model`
   value (`codex:gpt-5.6-sol`), and that the latter routes cross-provider. Do NOT change the default
   (`opus`) — codex is opt-in per iteration via env override, so no accidental spend.
4. **Deploy to main checkout** (see Deployment reality above) — `git pull --ff-only` + grep-verify
   the on-disk SKILL contains the `provider:model` recipe.

### Acceptance criteria (M1b)
- [x] Gate 3 in the on-disk SKILL has a concrete `provider:model` → `codex exec` spawn recipe with a bounded (≤30 min) cap.
- [x] A `codex exec` trivial probe succeeds token-cheaply (`reply with exactly: ok`) proving auth+model+CLI. *(live-verified 2026-07-16: `codex exec --model gpt-5.6-sol 'reply with exactly: ok'` → exit 0, replied `ok`, ~13.7k tokens, codex-cli 0.137.0)*
- [x] generator≠judge is asserted in-recipe: evaluator provider ≠ executor provider (re-route+flag on collision).
- [x] Fallback-to-`$MODEL`-and-flag on codex failure is documented in the recipe.
- [ ] One iteration (or a dry harness run) demonstrates controller (claude) and executor (codex) on **different providers**; Gate-4 row records `provider=codex agent=codex model=<gpt-5.x>`. *(recipe executable; deferred — acceptance smoke uses the token-cheap probe per constraint #6; first real cross-provider run happens when the controller opts in via env)*
- [x] Change is deployed to the main checkout (grep-verified on disk), not merely merged to a PR branch. *(edited directly on `dev` in the main checkout — controller-authorized deviation; launchd reads on-disk)*

### Risks
- **codex worktree contract** — the codex executor commits to its own branch; verify the skill reads
  the branch/diff the same way the claude path does (mitigate: reuse the existing worktree-read step,
  don't invent a new one).
- **Real-token spend** — the *acceptance* uses the trivial probe + DryRun; a full real codex run bills
  metered OpenAI $ (the intended relief per the doc's Cost note). Keep the smoke path token-free.

---

## Milestone M2 — right-sizing table + evidence-row schema in the charter (~0.5 day)

**Goal**: land the (provider, agent, tier) right-sizing table into the charter and extend the
routing-evidence row schema. Acceptance: one iteration writes a full new-schema row.

### Tasks

1. **Right-sizing table → charter** (edit `design_docs/v1-mission.md`, in the "Model routing policy"
   section at :153-181). Add the doc's Design table (Role | Agentic? | Needs | Tier hypothesis |
   Agent candidates) as the canonical right-sizing hypothesis, cross-linked to this design doc.
2. **Evidence-row schema extension** (edit `design_docs/v1-mission-log.md:11-12` template block).
   Extend from `model=<m> task-class=… round1-score=<n> rounds=<n> corrections=<n>` to:
   `provider=<p> agent=<a> model=<m> task-class=… round1-score=<n> rounds=<n> corrections=<n> cost=<$/quota-bucket>`.
   Keep the old columns (append the three new ones) so historical rows stay parseable.
3. **Cost source** — `cost` records `$` for metered providers (codex/gemini, from executor
   `execResult.CostUSD` surfaced at `provider_executor.go:146`) or the **quota bucket** for Anthropic
   (`weekly-fable` / `weekly-opus`) since subscription calls don't have per-call $. No silent
   fallback (Critical Principle 2): if cost is genuinely unknown, write `cost=unknown` explicitly,
   never `0`.
4. **Deploy to main checkout** (charter + log are read by the controller session on disk).

### Acceptance criteria (M2)
- [x] Charter `design_docs/v1-mission.md` contains the (provider, agent, tier) right-sizing table, cross-linked to this doc.
- [x] `v1-mission-log.md` template row schema has `provider`, `agent`, and `cost` columns appended (old columns retained).
- [x] One new log entry is written in the new schema with real (provider, agent, model, task-class, round-1 score, rounds, corrections, cost). *(entry 34; round-1 score marked `<pending-evaluator>` — the row is written by this executor run, the evaluator fills scores next)*
- [x] `cost` uses `$` for metered providers, quota-bucket for Anthropic, explicit `unknown` otherwise (never silent 0). *(this row: `cost=quota-bucket:weekly-opus` — Anthropic subscription, no per-call $)*
- [x] Deployed to main checkout on disk. *(edited directly on `dev` in the main checkout)*

### Risks
- **Log parser drift** — if any tooling parses the fixed row template, appending columns could break
  it. Mitigate: grep for consumers of the routing-evidence line before editing; append (don't
  reorder) so positional parsers still find the leading `model=`.

---

## Milestone M3 — planner down-tier A/B (PARKED with a concrete protocol)

**Status**: **PARKED — cannot execute inside one loop iteration** (needs 3 real quorum-reviewed docs;
charter ≥3-datapoint rule). Recorded here as an executable protocol so the controller can run it as
evidence accrues, NOT forced into this sprint (constraint #5).

### A/B protocol (execute over ≥3 future iterations)

1. **Population**: the next 3 design docs that have passed the live quorum (adversarial 3-provider
   review). Each is a *solid spec* — the condition the doc argues moves synthesis upstream of the planner.
2. **Arms**:
   - **Arm A (control)**: plan with `MISSION_PLANNER_MODEL=opus` (current default).
   - **Arm B (down-tier)**: plan the *same doc* with a mid-tier planner — codex-mid or
     `sonnet`/`fable` alias (a `provider:model` or Agent-alias value via the M1b branch).
   - Same doc, same executor model, same evaluator — only the **planner** varies.
3. **Metric**: executor **round-1 evaluator score** + **corrections count** per arm (the doc's stated
   comparison). A down-tier is justified only if Arm B's round-1 scores are within noise of Arm A
   across all 3 (charter evidence rule: ≥3 data points, both directions).
4. **Record**: one routing-evidence row per arm per doc (new M2 schema), and a verdict line in the
   charter right-sizing table (`sprint-planner: keep Opus | down-tier to <tier>` + the 3 data points).
5. **Bounded**: each arm's planning run inherits the sprint's existing bounded caps; no new waits.

### Acceptance criteria (M3 — deferred)
- [ ] Protocol recorded in this plan + referenced from the charter table (this milestone = the *plan*, not the run).
- [ ] (Deferred) 3 quorum-reviewed docs planned in both arms; round-1 scores + corrections recorded per arm.
- [ ] (Deferred) A data-backed keep/down-tier verdict for sprint-planner written to the charter (≥3 data points).

### Why parked, not descoped
The design doc's own Non-goals keep "Full Phase E" out; M3 is the *first* Phase-E evidence probe.
Its value is real but its cadence is multi-iteration. Parking with a concrete protocol (rather than a
vague "do the A/B later") means the controller can execute it deterministically the moment M1b + M2
give it the cross-provider planner branch + the evidence schema to record into.

---

## Sequencing & dependencies

```
M2 (evidence schema) ──┐
                       ├──> M3 (A/B protocol needs both the schema AND the provider:model branch)
M1b (provider:model) ──┘
```

- **M1b and M2 are independent** — can be done in either order (M2 is pure doc-schema; M1b is skill
  orchestration + probe). Do M1b first (it's the acceptance the doc leads with).
- **M3 depends on both** — it needs M1b's `provider:model` planner branch to run Arm B, and M2's
  extended schema to record the A/B rows. Hence parked until both land + 3 quorum docs exist.

## Success metrics (executable slice)
- M1b: one demonstrated cross-provider (claude controller ≠ codex executor) run, token-cheap probe green, deployed on disk.
- M2: charter table + extended schema on disk; one new-schema row written.
- No GPU (`rig.lock`) touched (constraint #4). All waits bounded ≤30 min (constraint #6). No new Go plumbing (constraint #2).

## Out of scope (explicit)
- motoko/ollama executor (GPU lane, `rig.lock`) — constraint #4.
- Agentic *review* verification — sibling doc `m-mission-quorum-agentic-verify`.
- Retiring the `std/ai` single-call path.
- Full Phase E across all task classes (only the planner A/B is seeded, and it's parked).

---
**Plan created**: 2026-07-16 · sprint-planner (Opus) · mission iteration 31
