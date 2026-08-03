# M-PLANNER-CODEX-LANE: Route the Mission Planner Through the Hardened codex Spawn Recipe

**Status**: Planned — **revision 3** (round-2 reviewers' verbatim fixes applied under the mission's narrow-refinement carve-out — NOT a re-quorum; see Revision History)
**Target**: v1.0 mission infra (control plane only — does NOT gate v1.0; protects its timeline)
**Priority**: P1
**Estimated**: 1 day, split M1/M2 — the default flip lands ONLY at the end of M2 (see Timeline).
Re-sized at rev 3: the allowlist (+~20 LOC) and five extra fixtures (tiny text docs, zero-token
checks) fit within M1's existing half-day — still ≤1 day, no milestone split needed
**Dependencies**: M1b codex executor lane (implemented — `design_docs/implemented/v0_30_0/m-mission-agentic-provider-routing-sprint-plan.md`)
**Origin**: Mark directive 2026-07-30 (attended), queued in `design_docs/v1-mission.md:1380-1383`
**Planner-Lane**: codex-ok

> **Standing constraint (new in rev 2): the driver shell is Bash 3.2.** `/bin/bash` is GNU bash
> 3.2.57 and **no bash 4.x exists anywhere on this rig** (L19), so `#!/usr/bin/env bash` resolves
> to 3.2. NO associative arrays (`declare -A`), NO `${var,,}`. This is the SECOND
> associative-array defect in this loop (iter-107, `v1-mission-log.md:3997`); AC7 now proves 3.2
> compatibility non-vacuously.

> **Scope note (explicit, not omitted):** this is mission-control infrastructure. No AILANG
> language code is involved, so the design-doc-creator skill's live `ailang check` verification
> gate and the parser/types/codegen **Conflict Surface** section are **N/A** — there are no
> "AILANG supports X" claims in this doc and no language-surface files are touched. The claims
> this doc DOES make are about the control plane, and each carries its verification below.

---

## Verification Log

Every load-bearing claim, with how it was verified. "Session" = verified first-party by this
doc's author on 2026-07-31 in this authoring session. Rows the controller labelled VERIFIED-BY-ME
were independently re-verified here where the design depends on them.

| # | Claim | How verified |
|---|-------|--------------|
| L1 | The codex lane is live; a real `codex exec --model gpt-5.6-sol --sandbox workspace-write` run completes rc=0 | Session: full capability-test run (L5) exited rc=0 — stronger than the controller's 1-token probe (V1), which it supersedes |
| L2 | The driver's codex probe/fallback is **executor-only**; `MISSION_PLANNER_MODEL` is exported with no probe and no fallback | Session: read `tools/launchd/mission-control.sh:280` (`export MISSION_PLANNER_MODEL="${MISSION_PLANNER_MODEL:-opus}"`) and the `case "$MISSION_EXECUTOR_MODEL" in codex:*)` block at lines 295-308 |
| L3 | codex authenticates via ChatGPT subscription → quota bucket, not metered $ | Session: `python3 -c "…json.load(open('~/.codex/auth.json'))…"` → `auth_mode: chatgpt` (field-only read, no token exposure); corroborates driver comment `mission-control.sh:258-259` |
| L4 | sprint-planner skill has **zero** `gh`/`curl` references (the iteration-122 "gh auth fails in sandbox" blocker does not apply) | Session: `grep -cE "gh api\|gh \|curl" .claude/skills/sprint-planner/SKILL.md` → **0**, with known-positive control in the same call: `grep -c "sprint"` → **94** |
| L5 | **U1 RESOLVED — the two planner scripts run under the codex `workspace-write` sandbox.** `analyze_velocity.sh` rc=0; `create_sprint_json.sh` rc=0 producing valid JSON (`jq -e` outside the sandbox: sprint_id correct, 2 features parsed from a 2-milestone plan); linked-worktree `git log` rc=0 | Session: real bounded `codex exec --model gpt-5.6-sol --sandbox workspace-write -C <throwaway repo>` (7-min cap, `< /dev/null`, directive-delivery asserted at 713 B), transcript in `/tmp/codex_u1_out.log`, final message in `/tmp/codex_u1_last.txt` |
| L6 | **A linked worktree's `git commit` is DENIED under the sandbox** — `fatal: Unable to create '….git/worktrees/wt/index.lock': Operation not permitted` | Session: same L5 run, command 5, rc=128 — first-party corroboration of the iter-32 live observation cited in `mission-control/SKILL.md:556-559` |
| L7 | `create_sprint_json.sh` has an **interactive prompt**: `read -p "Overwrite? (y/N)"` when the output JSON already exists | Session: read `.claude/skills/sprint-planner/scripts/create_sprint_json.sh:85-94`. With stdin `< /dev/null` this fail-louds (read hits EOF → rc≠0 → `set -e` aborts); **without** the stdin redirect under a backgrounded launch it would BLOCK — the same class as false-green #1 in the executor recipe. The `< /dev/null` guard is load-bearing for the planner too |
| L8 | `create_sprint_json.sh` contains a **built-in vacuous-pass generator**: if milestone parsing finds nothing it writes a placeholder (`"id": "MILESTONE_ID"`, "auto-parse failed - fill manually") and exits 0 | Session: read `create_sprint_json.sh:160-172,199-203`. Acceptance MUST grep for placeholder absence (AC3) |
| L9 | `create_sprint_json.sh:77` calls `ailang messages import-github` (network) — **non-fatal by construction** (`\|\| true`) | Session: read the line. Whether it succeeds or silently skips inside the sandbox is **UNVERIFIED** (the L5 test captured only output tails); treated as silently-degraded in-sandbox, with a controller-side compensation (see D4 step 5) |
| L10 | Planner artifacts land in the **main checkout** paths: `design_docs/planned/vX_Y/<name>-sprint-plan.md` + `.ailang/state/sprints/sprint_<id>.json`, relative to CWD | Session: `grep -n` on `sprint-planner/SKILL.md:48,62-63,163` + read `create_sprint_json.sh:39-40` (`PROGRESS_DIR=".ailang/state/sprints"` — relative, so artifacts land wherever `-C` points) |
| L11 | Current planner spawn = `$MISSION_PLANNER_MODEL`-pinned **Agent sub-agent in the main checkout** | Session: read `mission-control/SKILL.md:641` |
| L12 | Cross-role independence has measured value: iter-121 planner (opus) refuted **five** controller premises; iter-122 planner refuted 4; iter-123 codex **executor** refuted the controller's own fix directive ("My own fix directive was WRONG, and the executor refuted it") | Session: read `design_docs/v1-mission-log.md:6336` (iter-121), `:6411` (iter-122 routing evidence), and the iter-123 entry under `:6419` |
| L13 | The `generator≠judge` guard is executor-vs-**evaluator** only (evaluator stays `sonnet`) — it does NOT cover planner/executor collision | Session: read `mission-control/SKILL.md:563-567` and driver comment `mission-control.sh:262` |
| L14 | Driver is **bash** — but **Bash 3.2**, see L19 (`${!var}` indirection and `printf -v` available for D3, L22; `declare -A` and `${var,,}` are NOT, L20/L21); `_mc_bounded` at line 145, `QUOTA_SIG` at 133, `PROBE_TIMEOUT` at 134; **`MISSION_DRY_RUN=1` exists** and prints all four exported roles (lines 351-352), AFTER the codex probe block in file order — the line-350 "no probes fired" comment refers to the *claude* model-selection probes below it, not the codex block | Session: `head -3` (shebang), `grep -n`, and `sed -n '345,356p'` on `tools/launchd/mission-control.sh` (rev-2 re-read) |
| L15 | **Blast-radius topology (refines the directive's premise):** the Ailang World mission runs its **own byte-identical copy** of the driver (`ailang-world/tools/launchd/mission-control.sh`, `diff -q` = identical; its line 280 has the same opus planner default), launched by `dev.ailang.mission-world.plist`. The skill exists as a second copy at `~/.claude/skills/mission-control/SKILL.md` (`diff -q` = identical today). So a driver edit here does NOT immediately execute in World — propagation is an explicit sync step (see D6) | Session: `grep -rn mission-control.sh ~/Library/LaunchAgents/*.plist`, `diff -q` both driver copies, `diff -q` both skill copies |
| L16 | The #486 lesson is recorded in driver comments: probe MUST carry `--model`; fall back on ANY non-zero rc, not just quota signatures | Session: read `mission-control.sh:285-294` |
| L17 | `codex sandbox` (token-free seatbelt runner) is **unavailable without config**: requires `--permission-profile` and errors `default_permissions requires a [permissions] table` on this rig | Session: ran it; noted as a nice-to-have for future zero-token sandbox tests, not used by this design |
| L18 | Negative claim: `create_sprint_json.sh`'s GitHub-issue discovery success **inside** the sandbox is not established | UNVERIFIED (see L9) — explicitly compensated, not assumed |
| L19 | **The driver executes under GNU bash 3.2.57, with no escape hatch on this rig**: `head -1 tools/launchd/mission-control.sh` → `#!/usr/bin/env bash`; `command -v bash` → `/bin/bash`; `env bash --version` → 3.2.57(1)-release; `/opt/homebrew/bin/bash` and `/usr/local/bin/bash` are BOTH "No such file or directory", so `env bash` cannot resolve to 4.x — the launchd plist PATH contains those two dirs, which is exactly why the rev-1 sketch looked safe, but neither holds a bash | Controller-verified (quorum round 1, first-party) + re-verified this session: `/bin/bash --version` → 3.2.57(1)-release, both `ls` probes → ENOENT |
| L20 | Rev-1's D3 sketch **crashes** under 3.2: `/bin/bash -c 'declare -A t; t[gpt-5.6-sol]=1'` emits BOTH `declare: -A: invalid option` AND `gpt-5.6-sol: syntax error: invalid arithmetic operator (error token is ".6-sol")`, rc=1 — the string index is evaluated arithmetically. Controller's decisive test under the driver's own shebang form printed `ASSOC UNSUPPORTED — driver would crash`: the next launchd fire would have taken down the driver before any role spawned | Controller-verified + session repro (identical output, rc=1) |
| L21 | **Second Bash-4.0-ism in the rev-1 sketch, not caught by either reviewer**: `${role,,}` → `/bin/bash -c 'role=PLANNER; echo "${role,,}"'` → `bad substitution`, rc=1. Fixed alongside L20 (a `tr` pipeline) | Session |
| L22 | The 3.2-compatible replacements all work under `/bin/bash`: `printf -v "$var" opus` + `${!var}` round-trip rc=0; the ':'-delimited-string dedupe pattern (gemini's proposed fix) correctly reports `probe` on first sight and `dup` on second sight of the same model | Session: live `/bin/bash -c` runs, dedupe test includes its own positive control (second pass MUST report dup) |
| L23 | **Partial downstream backstop for the L8 placeholder path**: `.claude/skills/sprint-executor/scripts/validate_sprint_json.sh:109` rejects milestones with `estimated_loc == 0` or `description == "Milestone description"`, and the L8 placeholder writes exactly `estimated_loc: 0` (`create_sprint_json.sh:165`). Scope of the credit, stated precisely: it fires at **executor** time (iter-121 observed it live, `v1-mission-log.md:6369`), NOT at planning time, and covers only the placeholder-field form — not a plan file that names no design doc or a greeting-message final output. AC3's own greps remain load-bearing. Known overload watch-item: `estimated_loc == 0` also means "docs-only" (`v1-mission-log.md:6379`) | Session: read both scripts + both log lines |

---

## Problem Statement

The opus quota bucket dried **Thursday at ~55% duty cycle** in the week of 2026-07-27 (Mark,
directive 2026-07-30). The mission's per-role routing (`mission-control.sh:280-319`) moved the
**executor** to the ChatGPT-subscription codex lane on 2026-07-27, but the **planner still rides
opus every iteration** (L2, L11). The planner is one of the two remaining Anthropic-heavy
sub-agent roles; moving it off-bucket stretches the weekly quota toward Monday-to-Monday, which
protects the v1.0 timeline (capacity multipliers before capacity consumers).

**Current state:**
- `MISSION_PLANNER_MODEL` defaults to `opus`, exported with **no probe and no fallback** (L2).
- The hardened `PROVIDER=codex` recipe (`mission-control/SKILL.md:477-570`) is written for the
  executor only: it targets a sprint **worktree**, and its roles table names codex only in the
  executor row.
- The planner's artifacts belong in the **main checkout** (L10), which the current Agent-spawned
  planner writes directly (L11) — a codex planner cannot safely do the same (D1).

**Impact:** every mission iteration of both missions; the opus bucket funds the controller, which
is the one role that cannot move.

## Goals

**Primary goal:** `MISSION_PLANNER_MODEL=codex:gpt-5.6-sol` produces real sprint plans through
the hardened codex spawn recipe, with the same fail-loud guarantees the executor lane has, at $0
metered.

**Success metrics:**
- ≥1 full E2E rehearsal of the codex planner path within the sprint (AC3a), then ≥1 real sprint
  planned end-to-end by codex with non-placeholder artifacts in the main checkout (AC3b).
- A failed planner probe demonstrably falls back to opus and says so in the driver log (AC2).
- Opus-bucket consumption per iteration drops by the planner's share (observable in the weekly
  routing-evidence rows; the falsifiable form: on infra-class iterations, `planner=codex…` appears
  in the evidence row instead of `planner=opus`).

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: planner writes in an **ephemeral detached worktree from local HEAD**; controller copies artifacts back and commits | Wrong choice either exposes a sibling's uncommitted work to a sandboxed agent or produces plans against stale bases | agent (this doc) | design | med |
| D2: same-model planner+executor allowed **only for non-language-surface sprints**, enforced by a **deterministic fail-closed derivation** (`derive-planner-lane.sh` + `**Planner-Lane**` doc field), NOT controller judgement | Cross-role refutation has measured value (L12); losing it silently correlates plan and implementation blind spots; rev-1's prose-only rule was the exported-env-vs-actual-spawn divergence | quorum round 2 (mechanism per gpt5-6-sol objection + controller steer option (i)); rev 3: allowlist + env-pin step 0, both R2 reviewer fixes verbatim | design | low-med (one new ~60 LOC script) |
| D3: driver probe/fallback becomes **role-generic** (loop over PLANNER, EXECUTOR) with per-model probe dedupe | The exported env is what the routing-evidence row reports; a silent fallback is a routing lie | agent (this doc) | design | low |
| D6: rollback = **one env var** (`MISSION_PLANNER_MODEL=opus`); World propagation is an explicit sync step | This edits the control plane that runs every V1 iteration; a bad landing wedges the loop | agent (this doc) | design | low |

### Design Freeze

- [x] D1 resolved: option (b), ephemeral detached worktree + controller copy-back (below)
- [x] D2 resolved (rev 2): conditional independence rule with **deterministic fail-closed
      enforcement** — the quorum DID disagree with rev 1's prose-only enforcement, and was
      right; the mechanism below is executable, missing/invalid classification fails closed
      to opus, and the default flip is sequenced behind its acceptance fixtures
- [x] D3 resolved (rev 2): role-generic probe loop rewritten **Bash 3.2-compatible**
      (delimited-string dedupe per gemini-3-1-pro's confirmed objection; `${role,,}` also
      removed — L21), planner fallback target `opus`
- [x] Quorum round 2 (2026-07-31T16:32Z): BLOCKED with two NEW reviewer-authored fixes, neither
      disputing direction — gpt5-6-sol (D2 step-4 denylist→allowlist) and gemini-3-1-pro
      (contract step 0: `$MISSION_PLANNER_MODEL` env-pin check). Rev 3 applies BOTH fixes
      verbatim under the mission's narrow-refinement carve-out — **NOT a re-quorum**, and no
      controller-invented resolution was substituted for either reviewer's text

---

## Solution Design

### D1 — Where does the codex planner write? **Decision: (b) ephemeral detached worktree, controller copy-back.**

The constraint triangle: (i) planner artifacts belong in the main checkout (L10); (ii) a
sandboxed agent must not roam the main checkout while a sibling session may hold uncommitted
work there (Critical Principle 0 + concurrent-agent discipline); (iii) `analyze_velocity.sh`
needs real git history (`git log --since`), so a bare scratch dir won't run it (`set -euo
pipefail` aborts on the failed `git log`).

| Option | Mechanics | Failure mode | Verdict |
|---|---|---|---|
| (a) `-C` the main checkout | codex gets `workspace-write` over the whole main tree | A misbehaving or confused codex run can edit/clobber a **sibling agent's uncommitted work** — the exact class Critical Principle 0 exists to prevent. Also `read -p` aborts on a stale same-id sprint JSON (L7) — fail-loud but avoidable | **REJECTED** — the one option that is unsafe under concurrency, which is the stated bar |
| (b) ephemeral **detached** worktree from **local HEAD**; controller copies the 2 artifacts back and commits | `git worktree add --detach /tmp/planner-wt-iter<N> HEAD` → codex runs `-C` there → controller verifies + copies `…-sprint-plan.md` and `sprint_<id>.json` into the main checkout → `git worktree remove` | Worktree lacks **uncommitted** main-tree context → mitigated by a hard precondition: the design doc must be **committed** before the planner fires (assert `git status --porcelain -- <doc>` is empty; the mission flow already commits the doc at quorum time). Copy-back could clobber an existing file → controller checks existence first; per-iteration sprint IDs make collisions rare. Leftover worktrees → AC4 asserts none remain | **CHOSEN** |
| (c) controller-placed scratch dir | plain directory outside any repo | `analyze_velocity.sh` aborts (no git history); fixing that means teaching the scripts a `--repo` flag — a bigger, shared-script edit for zero safety gain over (b). A full clone would work but is slower and re-introduces the stale-base problem via origin | REJECTED |

Notes on (b), all first-party:
- **`--detach` from local HEAD**, not `-b` from `origin/dev`: the executor-worktree machinery
  branches from origin/dev (memory: `project_agent_worktree_isolation_quirks`), which would make
  a just-committed-but-unpushed design doc invisible to the planner. Detached-HEAD sidesteps
  both that and the "branch already checked out" error.
- The sandbox denies worktree commits anyway (L6), so codex **cannot** commit even by accident;
  the controller finalizes the commit in the main checkout, crediting the planner
  (`Co-Authored-By: codex <model>`) — the same discipline the executor lane already uses
  (`mission-control/SKILL.md:559-562`).
- Both planner scripts and JSON-writing verified working under exactly this shape (L5).

### D2 — Planner/executor independence. **Decision: same-model planner+executor is NOT acceptable unconditionally; it is acceptable for non-language-surface sprints, with the opus planner retained for Conflict-Surface sprints.**

The risk, stated honestly: `MISSION_EXECUTOR_MODEL` already defaults to `codex:gpt-5.6-sol`
(`mission-control.sh:281`). Routing the planner there too means plan and implementation share
one provider AND one model. The evaluator guard does not cover this (L13). What we would lose is
measured, not hypothetical: the planner refuted 5 controller premises in iter-121 and 4 in
iter-122; the executor refuted the controller's directive in iter-123 (L12). Those refutations
worked **because** the refuting role had different blind spots than the role being refuted. A
same-model pair can still catch *context* errors (different inputs, different working dirs) but
loses *model-correlated* blind-spot coverage — and premise errors in language-surface work are
precisely where this mission has repeatedly been saved (iter-121's wall-clock-seeded `ailang
test` find; the m-arity-style fictional-test case in the skill's own case studies).

**Enforcement (rev 2 — deterministic, fail-closed; replaces rev 1's "controller interprets the
doc", which the quorum correctly rejected as the exported-env-vs-actual-spawn divergence the
driver's own comments call out).** One honest constraint shapes the mechanism: the driver runs
ONCE PER FIRE and exports roles BEFORE the controller has picked a queue item, so at driver time
the sprint's design doc is **not yet known** — a purely driver-side classifier cannot exist.
The resolution is **two-stage** (controller-steer option (i)):

- **Stage 1 (driver):** exports the CONFIGURED planner lane, probed for *reachability* only.
  Reachability ≠ authorization — the driver makes no independence decision.
- **Stage 2 (skill, at Gate-3 planner spawn time, when the design doc IS known):** the
  controller MUST run `tools/launchd/derive-planner-lane.sh <design-doc-path>` (NEW file, Bash
  3.2-compatible, pure text derivation — no network, no codex invocation) and use its one-line
  output verbatim as the EFFECTIVE planner lane. The script's contract:
  0. **(rev 3, gemini-3-1-pro R2 fix, verbatim):** If `$MISSION_PLANNER_MODEL` does not start
     with `codex:`, output `opus fail-closed:env-pin` and exit. This ensures the script
     respects both driver probe fallbacks and the D6 rollback — the exported env ALWAYS wins;
     a `codex-ok` doc can never re-route the planner to codex against a driver probe fallback
     or the `MISSION_PLANNER_MODEL=opus` rollback pin.
  1. Doc missing/unreadable → `opus fail-closed:no-doc`.
  2. Read the machine-readable header field `**Planner-Lane**: <value>` (design docs use
     `**Key**: value` header lines, not YAML — greppable with
     `grep -m1 -E '^\*\*Planner-Lane\*\*:'`). Field absent →
     `opus fail-closed:planner-lane-field-missing`. Value not exactly one of
     `codex-ok` | `opus-required` → `opus fail-closed:planner-lane-field-invalid`.
     **Missing/invalid NEVER defaults to codex.**
  3. `opus-required` → `opus declared:opus-required`.
  4. `codex-ok` is a *declaration, not trusted alone* — **allowlist cross-check (rev 3,
     gpt5-6-sol R2 fix, verbatim — replaces the rev-2 conflict-surface denylist regex):**
     parse every declared path in the doc's **Files to Modify/Create** section (awk
     heading-range; section absent → `opus fail-closed:no-files-section`). `codex-ok` may
     produce `codex` only when EVERY declared modified/created path belongs to explicitly
     approved mission-control infrastructure prefixes. Initial allowlist (the reviewer's own
     example set; extensible ONLY by quorum-reviewed edit to this list):
     - `tools/launchd/` (includes `tools/launchd/testdata/planner-lane/` fixtures)
     - `.claude/skills/mission-control/SKILL.md`
     - `.claude/skills/design-doc-creator/SKILL.md`
     Any missing, unparsable, absolute, traversal-containing, unlisted, or newly introduced
     path must return `opus fail-closed:path-not-in-codex-allowlist`. Malformed or duplicate
     Files-to-Modify sections → the same fail-closed token. There is NO denylist to outrun:
     a path the allowlist has never heard of (a new `internal/...` dir, builtins, runtime,
     stdlib, module loading, optimization, any future top-level dir) fails closed by
     construction — this is the fail-closed guarantee D2 claims, actually delivered.
  5. Only `codex-ok` + every path allowlisted → `codex declared:codex-ok`.

  > **⚠ CONTRADICTION FLAG for the controller (rev 3, not resolved here per the carve-out):**
  > applied verbatim, the allowlist rule fail-closes THIS doc itself — its own Files-to-Modify
  > lists `~/.claude/skills/mission-control/SKILL.md` and
  > `~/dev/sunholo-data/ailang-world/tools/launchd/mission-control.sh` sync copies
  > (absolute/`~` paths → `fail-closed:path-not-in-codex-allowlist`), contradicting its
  > `codex-ok` header and AC3a's "(this doc itself qualifies)" parenthetical. Controller must
  > decide: rehearse AC3a against the clean-infra fixture instead, or rule how sync-copy paths
  > are declared. Not silently resolved.
- The skill spawns the planner ONLY on the derived lane, does NOT probe or spawn codex for the
  planner role unless the derivation returned `codex`, and records the script's full output line
  in the routing-evidence row. Script missing on disk → fail closed to opus, loudly.
- **Field supply:** the design-doc-creator skill template gains the `**Planner-Lane**` header
  line (one-line edit). Docs written before this field exists fail closed to opus — a safe
  degraded state that converges as new docs carry the field (this doc self-hosts it: see its own
  header, `codex-ok`, consistent with its N/A Conflict Surface). The declared value is reviewed
  by the same quorum that reviews the doc, so the classification decision is made at review
  time by reviewers, not at spawn time by controller judgement.
- The controller's existing Gate-2 premise handoff discipline (VERIFIED-BY-ME labels, treat-as-
  established-fact rules, `mission-control/SKILL.md:277-279,335-338`) stays unchanged as the
  backstop on every sprint.
- **Sequencing guard:** the `mission-control.sh:280` default flip to codex lands ONLY after
  AC-D2 and AC3a are green (Timeline, M2). If M2 slips, the flip does not land and the lane
  stays explicit-opt-in via env pin — gpt5-6-sol's own escape hatch (option (iii)) as the
  stated degraded outcome, not a silent one.

**One stated refinement of gpt5-6-sol's proposed AC wording** ("proves codex is not probed or
spawned"): when the EXECUTOR is on its codex default, the driver legitimately probes
`gpt-5.6-sol` for the executor's sake (deduped, D3) — that probe is not planner routing. The
standing guarantee is therefore scoped to the **planner role**: no planner-role codex spawn, and
the effective planner lane is opus + recorded. AC-D2's fixture pins `MISSION_EXECUTOR_MODEL=opus`
so the strong form ("codex neither probed nor spawned, at all") holds within the test itself.

**Why not the alternatives:** a *different codex model* (e.g. gpt-5.5) keeps provider-correlated
blind spots and adds a second un-probed model pin (#486 class) for marginal independence; an
*unconditional opus planner* is the status quo the directive exists to end; a *controller-side
premise re-check* alone re-serializes the exact opus work we are trying to offload.

**What would falsify this decision (either direction):**
- *Loosen* (drop the opus carve-out): if over ≥3 codex-planned infra sprints the codex planner
  refutes controller premises at a rate comparable to the opus planner's logged rate, the
  independence concern is over-weighted and the carve-out can go.
- *Tighten* (revert to opus planner): if any codex-planned + codex-executed sprint passes the
  sonnet evaluator but later fails on a **premise** error the opus planner's pattern would have
  caught (the L12 shape), the lane narrows or closes. Record either outcome in the mission log
  with the iteration number.

### D3 — Driver generalization (role-generic probe/fallback)

Replace the executor-only `case` block (`mission-control.sh:295-308`) with a loop.

**Rev-2 rewrite (objection 1, CONFIRMED first-party — L19/L20):** the rev-1 sketch used
`declare -A` and string indices, which under the driver's actual shell (Bash 3.2, L19) is an
invalid option AND an arithmetic-evaluation crash on `gpt-5.6-sol` (L20) — the next launchd fire
would have wedged the driver before any role spawned. gemini-3-1-pro's delimited-string fix is
adopted verbatim in shape; additionally, `${role,,}` (a second 4.0-ism the review did not flag)
is replaced with a `tr` pipeline (L21). Every construct below is live-verified under `/bin/bash`
3.2 (L22). Sketch (~30 LOC):

```bash
# Codex-lane pre-flight, ROLE-GENERIC (m-planner-codex-lane): probe once per DISTINCT
# codex model, fall back per-role on ANY non-zero rc (#486: probe MUST carry --model;
# an unusable pin is exactly as fatal as spent quota). Export AFTER fallback so the
# EXPORTED env — what the routing-evidence row reports — stays honest.
# BASH 3.2 (L19): ':'-delimited string sets, NOT associative arrays; no ${var,,}.
_cx_probed=":"   # models probed this fire (dedupe: planner+executor share the default model)
_cx_failed=":"   # models whose probe failed
for role in PLANNER EXECUTOR; do
  var="MISSION_${role}_MODEL"; val="${!var}"
  case "$val" in codex:*)
    cx_model="${val#codex:}"
    case "$_cx_probed" in *":${cx_model}:"*) : ;; *)   # not yet probed
      _cx_probed="${_cx_probed}${cx_model}:"
      _mc_bounded "$PROBE_TIMEOUT" codex exec --skip-git-repo-check --model "$cx_model" 'reply with exactly: ok'
      cx_rc=$?; cx_out="$MC_BOUNDED_OUT"
      if [ "$cx_rc" -ne 0 ]; then
        _cx_failed="${_cx_failed}${cx_model}:"
        # why-classification happens ONCE, at probe time (timeout / quota-sig / other)
        if [ "$cx_rc" -eq 124 ]; then cx_why="probe timed out after ${PROBE_TIMEOUT}s"
        elif printf '%s' "$cx_out" | grep -qiE "$QUOTA_SIG"; then cx_why="quota-limited"
        else cx_why="probe failed (rc=$cx_rc)"; fi
        log "codex model '$cx_model' unusable: $cx_why"
        log "codex probe output: $(printf '%s' "$cx_out" | tail -3 | tr '\n' ' ')"
      fi
    ;; esac
    case "$_cx_failed" in *":${cx_model}:"*)
      role_lc=$(printf '%s' "$role" | tr 'A-Z' 'a-z')   # ${role,,} is bash-4.0-only (L21)
      log "codex ${role_lc} lane -> falling back to opus for this fire (model '$cx_model')"
      printf -v "$var" 'opus'; export "$var"
    ;; esac
  ;; esac
done
```

Preserved invariants, each traceable to a live incident: `--model` on the probe and fallback on
ANY rc≠0 (L16, #486); bounded probe via `_mc_bounded`/`PROBE_TIMEOUT` (L14); fallback target
`opus` for both roles; the log line names the ROLE so a planner fallback is distinguishable from
an executor fallback; probe dedupe so the default same-model config costs ONE probe, not two
(iter-122 recorded ~20.7k tokens for a probe — doubling that every fire for an identical answer
is pure waste). The `MISSION_DRY_RUN=1` path (L14, lines 351-352) already prints all four
exported roles AFTER this block runs, so AC2's fallback demo costs a probe and nothing else.

**Default flip (GATED — rev 2):** `mission-control.sh:280` becomes
`export MISSION_PLANNER_MODEL="${MISSION_PLANNER_MODEL:-codex:gpt-5.6-sol}"`, with a comment
citing this doc and the D2 derivation. This one-line change is the LAST landing of the sprint
(M2), permitted only after AC-D2 and AC3a are green — the configured default may not outrun its
enforcement. Rollback stays one env var (D6).

### D4 — Skill recipe (minimal edit, planner-parameterized)

Two edits to `.claude/skills/mission-control/SKILL.md` (and its `~/.claude/skills` copy — L15):

1. **Roles table** (`SKILL.md:451`): Sprint-planner default becomes `codex:gpt-5.6-sol` with the
   D2 derivation stated in the cell ("effective lane = `derive-planner-lane.sh` output, verbatim;
   fail-closed to opus"). This also supersedes the stale "down-tier A/B = M3" note in that cell.
1b. **Derivation step (D2 stage 2, MANDATORY, runs FIRST):** before any planner probe or spawn,
   the controller runs `tools/launchd/derive-planner-lane.sh <design-doc>` and takes its output
   as the effective planner lane. **Rev 3 (gemini-3-1-pro R2 fix):** the script's step 0 checks
   `$MISSION_PLANNER_MODEL` FIRST — if it does not start with `codex:` the script outputs
   `opus fail-closed:env-pin` and exits, so the driver's reachability fallback and the D6
   rollback pin (`MISSION_PLANNER_MODEL=opus`) always win over a `codex-ok` doc; the skill must
   invoke the script with the driver-exported env intact (no unset/re-export). Output `opus …`
   → spawn the L11 opus Agent path directly (no codex probe/spawn for the planner role occurs);
   the reason token goes in the evidence row verbatim. Only `codex declared:codex-ok` proceeds
   to the codex recipe below.
2. **`PROVIDER=codex` section**: add a "planner role" sub-bullet that *parameterizes* the
   existing executor recipe rather than duplicating it. It differs in exactly four ways;
   everything else (probe form, bounded `date +%s` deadline, background launch via
   `run_in_background: true` because the harness foreground Bash cap is 10 min, `exec` in the
   subshell so the cap's kill reaches codex, `-o` final-message capture, the
   secrets-probe hygiene rule) is carried **verbatim by reference**:
   - **Working dir:** `-C` the ephemeral detached planner worktree (D1), created by the
     controller, removed by the controller after copy-back.
   - **Directive file:** per-iteration name `/tmp/codex_planner_directive_iter<N>.txt`; the
     delivery assertion is IDENTICAL (exists / ≥200 B / non-empty prompt / exit 64 otherwise)
     and stdin is `< /dev/null` on probe and run — for the planner this guard is doubly
     load-bearing because `create_sprint_json.sh` contains an interactive `read -p` that would
     BLOCK a backgrounded run with open stdin (L7).
   - **Sandbox dirs:** keep `--add-dir "$GOCACHE" --add-dir "$GOMODCACHE"` — the planner's
     premise-verification work may legitimately run `go build`/`go test`; and the executor
     lane's caveat applies verbatim: **a gate/premise verdict from inside the sandbox is NOT
     evidence** — socket-touching checks are UNINFORMATIVE UNDER SANDBOX and the controller
     re-verifies load-bearing premises outside before handing the plan to the executor.
   - **Post-run controller steps (replaces the executor's diff-read/commit step):**
     (1) assert both artifacts exist **in the worktree** and are well-formed
     (`jq -e . sprint_<id>.json`; plan file non-empty and names the design doc);
     (2) assert NO placeholder vacuous-pass (`grep -L "MILESTONE_ID"` and no
     "auto-parse failed" — L8);
     (3) copy both to the main checkout paths, refusing to overwrite unexpected existing files;
     (4) `git worktree remove` the planner worktree;
     (5) run `ailang messages import-github --labels bug,feature,ailang-message` **outside** the
     sandbox once, compensating for the possibly-skipped in-sandbox sync (L9/L18);
     (6) commit with `Co-Authored-By: codex <model>`.
   - **Fallback:** probe-fail / cap / assertion-fail → spawn the planner as an `opus` Agent
     sub-agent in the main checkout (today's L11 path, unchanged) + FLAG in Gate-5.

The skill stays ONE skill — the planner bullet references the executor recipe's guards instead
of forking them, so a future guard fix lands once.

### Files to Modify/Create

**Modified:**
- `tools/launchd/mission-control.sh` — role-generic probe loop (Bash 3.2 form) + gated default flip (~35 LOC net)
- `.claude/skills/mission-control/SKILL.md` — roles-table cell + derivation step + planner sub-bullet (~55 lines)
- `.claude/skills/design-doc-creator/SKILL.md` — one-line template addition: the
  `**Planner-Lane**: codex-ok | opus-required` header field (D2 field supply)

**New:**
- `tools/launchd/derive-planner-lane.sh` — D2 stage-2 derivation (~60 LOC, Bash 3.2-compatible,
  pure text — contains no codex invocation, provable by AC-D2's grep)
- `tools/launchd/testdata/planner-lane/` — twelve tiny fixture docs for AC-D2 (rev 3, per
  gpt5-6-sol's R2 fixture list): (a) unlisted language path (declares `codex-ok`,
  Files-to-Modify touches internal/parser/… — the lying-declaration case), (b) field-missing,
  (c) clean infra `codex-ok`, (f) unknown FUTURE internal/... path (a dir that does not exist
  today), (g) mixed infra + language paths, (h) malformed Files-to-Modify section, (i) duplicate
  Files-to-Modify sections, (j) prose-first bullet, (k) alternate `## Files to Modify`
  heading, (l) invalid Planner-Lane value, (m) `opus-required`, (n) a Files bullet carrying no
  backticked token at all — the sentinel arm (j) does not reach

**No Go code. No AILANG code.**

### Landing Checklist

- Sync `.claude/skills/mission-control/SKILL.md` through its `~/.claude/skills/mission-control/SKILL.md` symlink (a no-op when the verified symlink remains intact).
- Deploy `tools/launchd/mission-control.sh` to `~/dev/sunholo-data/ailang-world/tools/launchd/mission-control.sh` when the World rollout is authorized.

---

## Acceptance Criteria (D5 — non-vacuous by construction)

The recurring failure class here is the **vacuous pass** — rc=0 for work never requested. Each
criterion below names the false green it kills.

- [ ] **AC1 (dry-run roles):** `MISSION_DRY_RUN=1 bash tools/launchd/mission-control.sh` prints
      `planner=codex:gpt-5.6-sol` in the roles line (driver line 352 mechanism, L14).
      *Kills:* "the default flip never actually exported."
- [ ] **AC2 (fallback is real and loud):** `MISSION_DRY_RUN=1 MISSION_PLANNER_MODEL=codex:no-such-model bash tools/launchd/mission-control.sh`
      → driver log contains `codex planner lane … falling back to opus` AND the dry-run roles
      line prints `planner=opus`. *Kills:* #486-class silent false-green on an unusable pin —
      note the probe MUST fail here precisely BECAUSE it carries `--model` (L16); if this test
      passes probe on the default model instead, the #486 regression has returned.
- [ ] **AC3a (in-sprint REHEARSAL — rev 2, satisfiable WITHIN this sprint):** run the full D4
      planner recipe against an EXISTING design doc (this doc itself qualifies: infra,
      `codex-ok`) with a throwaway sprint id (`rehearsal-iter<N>`): derivation returns
      `codex declared:codex-ok`; codex runs in the detached worktree; **in the worktree**,
      `jq -e` validates the JSON, `features | length ≥ 1` with **zero** occurrences of
      `MILESTONE_ID` or `auto-parse failed` (L8), the plan file names the input doc, and the
      codex final message (`-o` capture) is a plan summary, not "What would you like me to work
      on?". Copy-back and commit are SKIPPED by design — rehearsal artifacts are inspected then
      discarded with the worktree, so the main checkout is untouched. Cheap by construction: L5
      already shows both planner scripts rc=0 under this exact sandbox shape. *Kills:* the
      placeholder vacuous pass, the empty-directive chat-greeting pass, AND rev-1's structural
      defect that the sprint shipping this lane could not verify its own E2E path — the first
      real use was the test. (Note the downstream validator backstop is only PARTIAL — L23 —
      so these greps stay load-bearing here.)
- [ ] **AC3b (first real codex-planned sprint — follow-up criterion, explicitly NOT this
      sprint's gate):** on the first infra-class sprint after landing, the same checks as AC3a
      but at the **main-checkout** paths with copy-back and commit. *Kills:* "rehearsal green,
      copy-back path never exercised for real."
- [ ] **AC4 (hygiene):** after AC3a (and again after AC3b), `git worktree list` shows no `planner-wt-*` leftovers and
      `git status --porcelain` in the main checkout shows ONLY the two intended artifacts (plus
      their commit). *Kills:* "plan written to the worktree, never copied back, success reported
      anyway."
- [ ] **AC5 (delivery assertion demonstrably wired):** invoking the planner wrapper with the
      directive file absent exits **64** without spawning codex (grep the wrapper log for the
      FATAL line). *Kills:* false-green #2 (absent/truncated directive).
- [ ] **AC6 (evidence-row honesty):** the iteration's routing-evidence row records the
      derivation output verbatim — e.g. `planner=opus (fail-closed:path-not-in-codex-allowlist)`,
      `planner=opus (fail-closed:planner-lane-field-missing)`, `planner=opus (fail-closed:env-pin)`,
      `planner=codex:gpt-5.6-sol (declared:codex-ok)` —
      and `planner=opus (probe fallback, FLAGGED)` on AC2-class days. *Kills:* exported-env
      vs actual-spawn divergence — the routing lie the driver comments call out.
- [ ] **AC7 (Bash 3.2 compatibility, non-vacuous — rev 2):** in ONE check:
      `/bin/bash -c '/bin/bash --version | head -1; MISSION_DRY_RUN=1 /bin/bash tools/launchd/mission-control.sh'`
      — the output MUST show `version 3.2` AND the dry-run roles line, rc=0. The version print
      in the same invocation is the point: it *kills the false green where the loop is exercised
      on a 4.x shell* (e.g. a dev machine with homebrew bash) *and the green is credited as
      3.2-compat proof* — exactly how rev-1's `declare -A` sketch would have passed review while
      wedging the rig (L19/L20). **AC7b (rev 3, controller R2 residual-risk note — the driver
      was covered, the NEW skill-side script was not):** `derive-planner-lane.sh` itself MUST
      be proven Bash 3.2-compatible with the version print in the SAME invocation:
      `/bin/bash -c '/bin/bash --version | head -1; MISSION_PLANNER_MODEL=codex:gpt-5.6-sol /bin/bash tools/launchd/derive-planner-lane.sh tools/launchd/testdata/planner-lane/clean-infra.md'`
      — output MUST show `version 3.2` AND `codex declared:codex-ok`, rc=0 (zero tokens: pure
      text, no probe). This is new shell surface on a rig where `env bash` resolves to 3.2.57
      and no 4.x bash exists anywhere (L19). Known cost: with the flip landed AC7's driver form fires one real probe (the dry-run exit is
      AFTER the probe block — L14); run AC7 pre-flip (planner default still opus, executor
      pinned `MISSION_EXECUTOR_MODEL=opus`) for a zero-probe pure-shell pass.
- [ ] **AC8 (D2 enforcement fixtures — rev 3, per gpt5-6-sol's R2 fixture list):** with
      `MISSION_EXECUTOR_MODEL=opus` pinned (so ANY codex activity would be planner-attributable)
      AND `MISSION_PLANNER_MODEL=codex:gpt-5.6-sol` pinned for the fixture invocations — a
      direct consequence of contract step 0 (env-pin check): without the planner pin every
      fixture would short-circuit to `opus fail-closed:env-pin` and the path checks would never
      be exercised:
      (a) unlisted language path fixture (declares `codex-ok`, Files-to-Modify touches
      `internal/parser/…`) → `opus fail-closed:path-not-in-codex-allowlist`;
      (b) field-missing fixture → `opus fail-closed:planner-lane-field-missing`;
      (c) clean infra fixture → `codex declared:codex-ok` (the known-positive control proving
      the instrument can say codex at all — an all-opus result is a claim, not a fact);
      (d) `grep -cE 'codex (exec|sandbox)' tools/launchd/derive-planner-lane.sh` → **0**, with
      known-positive control `grep -cE 'codex (exec|sandbox)' tools/launchd/mission-control.sh`
      → **≥1** in the same command — proving the derivation path cannot probe or spawn codex;
      (e) for fixture (a) run through the skill's Gate-3 planner step: the spawned planner is
      the opus Agent path and the evidence row records the up-route token;
      (f) unknown FUTURE `internal/...` path fixture (a directory that does not exist today,
      declares `codex-ok`) → `opus fail-closed:path-not-in-codex-allowlist` — THE fixture that
      establishes fail-closed for paths the classifier has never seen;
      (g) mixed infra + language paths fixture (`tools/launchd/…` AND `internal/parser/…`) →
      `opus fail-closed:path-not-in-codex-allowlist` (one bad path poisons the whole doc);
      (h) malformed Files-to-Modify section fixture →
      `opus fail-closed:path-not-in-codex-allowlist`;
      (i) duplicate Files-to-Modify sections fixture →
      `opus fail-closed:path-not-in-codex-allowlist`.
      All of (a),(b),(f),(g),(h),(i) MUST up-route to opus. *Kills:* "the carve-out exists only
      as prose a controller may or may not apply" (the R1 objection-2 class) AND "fail-closed
      was claimed but the classifier was a denylist new directories can outrun" (the R2
      gpt5-6-sol class — AC8 previously tested only one recognized parser path).

**What a false green looks like here, named:** rc=0 + placeholder JSON (L8); rc=0 + greeting
message (empty directive); probe green on the wrong model (#486); artifacts stranded in the
worktree; an in-sandbox "premises verified" claim banked without an outside re-check; a shell
test green on bash 4.x credited as 3.2 proof (AC7); a lane derivation that never fires because
the field is missing and the code silently defaults to codex (AC8b kills — missing MUST mean
opus); a "fail-closed" classifier that is actually a denylist a new directory outruns (AC8f
kills — rev 3); a skill derivation that reads `codex-ok` from the doc and routes to codex
AGAINST the driver's fallback or the D6 rollback pin (contract step 0 kills — rev 3).
AC3a/AC5, AC2, AC4, D4-step-(1)'s outside re-verify, AC7, and AC8 catch them respectively.

## Testing Strategy

**Unit-level:** none (no Go code). The driver loop is exercised by AC1/AC2/AC7 directly; the
derivation script by the AC8 fixtures.
**Integration:** AC2 (fallback), AC3a (E2E rehearsal), AC5 (assertion), AC7 (3.2 shell), AC8
(D2 fixtures) — all runnable in-sprint in under ~1.5h total; AC2 costs one probe, AC3a costs one
real planner run on the subscription bucket; AC7 (pre-flip form) and AC8 cost zero tokens.
**Manual:** read the AC3a rehearsal plan for quality (is it a real day-by-day plan?) — the
controller does this anyway at Gate-3 handoff.

## Rollback & Blast Radius (D6)

**Rollback = one env var:** set `MISSION_PLANNER_MODEL=opus` in
`~/.config/ailang/mission-v1.env` (driver sources it, `mission-control.sh:4`) — no code revert
needed; the role-generic probe loop is a no-op for non-`codex:*` values by construction (the
`case` falls through exactly as today's executor block does for alias values).

**Blast radius, stated per artifact (L15 — this REFINES the directive's "runs EVERY iteration of
BOTH missions" premise):**
- **V1 mission:** full — every iteration's planner spawn changes lane. Worst credible failure:
  probe passes, real run produces garbage plans → caught at AC3-class checks each iteration
  (placeholder grep + controller Gate-3 read); the fallback path is the L11 status quo, so a
  wedge requires BOTH the codex run and the opus fallback to fail, which is today's existing
  failure surface.
- **Ailang World mission:** **zero until synced.** World runs its own byte-identical driver copy
  and its planner default is opus at its line 280 (L15). The sync is an explicit step in this
  sprint (or an explicit decision NOT to sync yet, messaged to the world-coordinator via
  cross-mission messages per the sibling-claim discipline). Until then World is insulated —
  and conversely, forgetting the sync means World silently lacks the planner probe if someone
  pins World's planner to codex. Say which in the landing commit.
- **The skill:** shared as two identical copies (repo + `~/.claude`); both must be edited in the
  same commit window or the loaded copy drifts from the reviewed one.

## Deferred Decisions

- Exact planner worktree path/naming (`/tmp/planner-wt-iter<N>` vs under `$STATE_DIR`) — implementer.
- Planner wall-clock cap: reuse the executor's 30-min or shorten (planning is lighter) — implementer;
  must remain a bounded `date +%s` deadline either way (Standing rule 6).
- ~~Whether AC3's first run counts as the sprint's own acceptance~~ — RESOLVED rev 2: AC3a
  rehearsal is the in-sprint proof; AC3b is an explicit follow-up criterion.
- gemini as a third planner rotation entry — out of scope until G4 (same gate as the designer rotation).

## Non-Goals

- **No change to the evaluator** (`sonnet`, generator≠judge intact — L13) or the designer rotation.
- **No change to the sprint-planner skill or its scripts** — L5 proves they run as-is; the
  interactive `read -p` (L7) is guarded by `< /dev/null`, not edited (shared-script blast radius
  not worth it for a guard we already carry).
- **Not fixing the metered-ledger "cost UNKNOWN" honesty defect** (iter-121/123 known issue) —
  the planner run inherits token-count-only recording; that defect has its own queue slot.
- **No new wire protocols / providers** — codex only, per the standing aitana-platform constraint.

## Timeline

**Total: 1 day, split into two gated milestones (rev 2):**

- **M1 (~half-day):** Bash-3.2 role-generic probe loop (NO default flip yet) +
  `derive-planner-lane.sh` + fixtures + design-doc-creator field line. Gate: AC1 (with an env
  pin, since the default has not flipped), AC2, AC5, AC7, AC8 green.
- **M2 (~half-day):** skill planner recipe (D4) + AC3a rehearsal + copy syncs. Gate: AC3a and
  AC4 green — and ONLY THEN the one-line default flip lands, as the sprint's final commit.
- **Slip rule, stated in advance:** if M2 slips, M1 lands alone and the flip does NOT land —
  the lane remains explicit-opt-in (`MISSION_PLANNER_MODEL=codex:gpt-5.6-sol` env pin), which is
  gpt5-6-sol's option (iii) as a declared degraded outcome. A smaller lane that is provably safe
  beats a larger one gated on controller judgement.

AC3b/AC6 mature on the first real infra-class sprint after landing — a follow-up criterion, not
this sprint's gate (the E2E proof this sprint owns is AC3a). If that first sprint is
language-surface, AC3b waits for the next infra sprint rather than forcing one — say so in the
landing report instead of manufacturing a vacuous test sprint (though note AC8's fixture (a)
means the up-route path itself is already proven in-sprint, not deferred).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Same-model plan+implementation correlated blind spots slip through | High (silent) | D2 deterministic carve-out + Gate-2 backstop + named falsifiers; tighten on first L12-shape post-merge premise failure |
| A doc self-declares `codex-ok` while touching language surface | Med | Allowlist cross-check (rev 3): ANY path outside the approved infra prefixes — including new/future dirs — fails closed (`path-not-in-codex-allowlist`), proven by AC8 fixtures (a),(f),(g); quorum reviews the declared field with the doc |
| A future driver edit re-introduces a bash-4.0-ism | Med (wedges the loop) | Standing constraint in the doc header + driver comment + AC7 as a repeatable one-liner; two prior instances (iter-107, this rev-1) justify keeping AC7 in the post-edit checklist |
| codex produces placeholder/garbage plans at rc=0 | Med | AC3 placeholder grep is a standing per-iteration check, not one-time; opus fallback on assertion-fail |
| Probe cost doubles (planner+executor) | Low | Probe dedupe by model (D3) — default config costs ONE probe |
| World driver copy drifts | Med | Explicit sync step + cross-mission message; World's own default stays opus until its copy moves |
| Planner worktree leaks disk / stale worktrees accumulate | Low | Controller `git worktree remove` + AC4; worktrees are ephemeral per-iteration |
| In-sandbox GitHub issue sync silently skipped (L9/L18) | Low | Controller-side `import-github` compensation (D4 step 5) |

## Related Documents

- [m-mission-agentic-provider-routing-sprint-plan.md](../../implemented/v0_30_0/m-mission-agentic-provider-routing-sprint-plan.md) (0.45) —
  the M1/M1b machinery this doc EXTENDS to the planner role; its M3 "planner down-tier A/B" is
  superseded by this doc's D2 rule.
- [m-coord-codex-executor.md](../../implemented/v0_22_0/m-coord-codex-executor.md) (0.49) —
  DISTINCT: the *coordinator's* codex executor lane, not mission-control's planner role; informs
  sandbox behavior only.
- [m-exec-expand-codex-opencode.md](../../implemented/v0_15_0/m-exec-expand-codex-opencode.md) (0.43) —
  historical `ailang exec` codex wiring; not the spawn path used here.
- [m-mission-adaptive-multiprovider-routing.md](../v0_30_0/m-mission-adaptive-multiprovider-routing.md) (0.39) —
  planned adaptive routing; this doc is a static-default change and does not preempt it.

## References

- `design_docs/v1-mission.md:1380-1383` — the queued directive (Mark 2026-07-30, attended)
- `tools/launchd/mission-control.sh:280-308` — planner export + executor-only probe block (L2)
- `.claude/skills/mission-control/SKILL.md:477-570` — the hardened codex recipe + three false-greens
- `design_docs/v1-mission-log.md:6336,6411,6419+` — the L12 refutation evidence
- `/tmp/codex_u1_out.log`, `/tmp/codex_u1_last.txt` — L5/L6 live sandbox test transcript (2026-07-31)
- `.ailang/state/mission-quorum/m-planner-codex-lane-2026-07-31T16-21-29Z.json` — round-1 quorum
  artifact (BLOCKED, both objections resolved by rev 2)
- `.ailang/state/mission-quorum/m-planner-codex-lane-2026-07-31T16-32-39Z.json` — round-2 quorum
  artifact (BLOCKED with two reviewer-authored fixes; applied verbatim by rev 3 under the
  narrow-refinement carve-out, NOT a re-quorum)
- `design_docs/v1-mission-log.md:3997` — iter-107: the FIRST bash-3.2 `declare -A` defect in this loop
- `.claude/skills/sprint-executor/scripts/validate_sprint_json.sh:109` — the partial placeholder backstop (L23)

## Axiom Compliance

Mission infra, not a language change — scored for completeness; no language semantics touched.

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language impact; driver behavior stays deterministic given env |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No language effects involved |
| A4: Explicit Authority | +1 | Sandbox confinement + explicit controller-only commit authority; no ambient main-tree write access for the sandboxed planner |
| A5: Bounded Verification | +1 | Every spawn bounded (probe timeout, 30-min cap); acceptance checks are local and mechanical |
| A6: Safe Concurrency | +1 | D1 removes the one option that races a sibling's uncommitted work |
| A7: Machines First | +1 | Keeps the autonomous loop funded and running through the week — the loop IS the machine consumer |
| A8: Minimal Syntax | 0 | N/A |
| A9: Cost Visibility | +1 | Exported-env honesty + evidence-row lane recording + probe dedupe |
| A10: Composability | 0 | Reuses the existing recipe by reference; one skill, no fork |
| A11: Structured Failure | +1 | exit 64 assertion, ANY-rc fallback, FLAGGED evidence rows — every failure mode is loud and classified |
| A12: System Boundary | 0 | No boundary-crossing semantics change |

**Net Score: +6** → **Decision: Move forward** (no −1 anywhere; A1/A3/A4/A7 clean).

## Future Work

- gemini planner rotation entry (post-G4).
- Zero-token sandbox capability tests via `codex sandbox` once a `[permissions]` profile is
  configured (L17).
- Adaptive per-sprint lane selection (the 0.39-related planned doc) using the evidence rows this
  doc starts producing.

---

## Revision History

**Rev 3 (2026-07-31, narrow-refinement carve-out — NOT a re-quorum)** — round-2 quorum BLOCKED
(artifact: `.ailang/state/mission-quorum/m-planner-codex-lane-2026-07-31T16-32-39Z.json`) with
two NEW objections, both carrying concrete reviewer-authored fixes and NEITHER disputing the
design direction, so the mission's narrow-refinement carve-out applies: the reviewers' own
verbatim fixes were applied, and **no controller-invented resolution was substituted for either
reviewer's text**:
1. **gpt5-6-sol (D2 was denylist, not fail-closed):** D2 step 4's conflict-surface regex
   replaced with the reviewer's allowlist — `codex-ok` may produce `codex` only when EVERY
   declared path belongs to explicitly approved mission-control infrastructure prefixes
   (`tools/launchd/`, the named mission/design-doc skill files, planner-lane test fixtures);
   any missing, unparsable, absolute, traversal-containing, unlisted, or newly introduced path
   → `opus fail-closed:path-not-in-codex-allowlist`. AC8 fixtures added per the reviewer's
   list: unlisted language path, unknown future `internal/...` path, mixed infra/language
   paths, malformed and duplicate Files-to-Modify sections — all up-route to opus.
2. **gemini-3-1-pro (skill derivation ignored `$MISSION_PLANNER_MODEL`, breaking the driver
   probe fallback and the D6 rollback):** `derive-planner-lane.sh` contract gains the
   reviewer's step 0 — if `$MISSION_PLANNER_MODEL` does not start with `codex:`, output
   `opus fail-closed:env-pin` and exit. D4 step 1b updated to match.
3. **Controller residual-risk note (applied):** AC7b added — `derive-planner-lane.sh` itself
   proven Bash 3.2-compatible under `/bin/bash` with `--version` showing 3.2 in the SAME
   invocation (the driver was AC7-covered; the new skill-side script was not).
4. **Flagged, not resolved:** the verbatim allowlist fail-closes this doc's OWN `~`-path sync
   copies (see the D2 contradiction flag) — surfaced for the controller per the carve-out's
   no-silent-resolution rule.

**Rev 2 (2026-07-31, post-quorum)** — round-1 quorum BLOCKED (artifact:
`.ailang/state/mission-quorum/m-planner-codex-lane-2026-07-31T16-21-29Z.json`); direction
uncontested, both objections resolved:
1. **gemini-3-1-pro (Bash 3.2, CONFIRMED first-party — L19/L20):** D3 rewritten with
   ':'-delimited-string dedupe per the reviewer's proposed fix; additionally removed `${role,,}`,
   a second 4.0-ism neither reviewer flagged (L21); all replacement constructs live-verified
   under `/bin/bash` 3.2 (L22). Standing constraint added to the header (second instance of this
   defect class — iter-107, log:3997); AC7 added to prove 3.2 compat non-vacuously.
2. **gpt5-6-sol (D2 enforcement):** adopted the fail-closed derivation via controller-steer
   option (i) — two-stage because the driver runs before the queue item is known, so only the
   skill (at Gate-3, doc in hand) can classify. New `**Planner-Lane**` doc field +
   `derive-planner-lane.sh` (pure text, no codex invocation); missing/invalid field fails
   closed to opus; self-declared `codex-ok` cross-checked against Files-to-Modify paths; default
   flip gated behind AC8+AC3a (M2), with option (iii) as the declared slip outcome. AC8 added.
   One stated refinement (not a disagreement): "codex is not probed" is scoped to the planner
   role in the standing guarantee, since a codex EXECUTOR legitimately shares the deduped
   driver probe; the AC8 fixture pins the executor to opus so the strong form holds in-test.
3. **Controller note (AC3 self-verification):** AC3 split into AC3a (in-sprint rehearsal on an
   existing doc, throwaway sprint id, no copy-back) and AC3b (first real sprint, follow-up
   criterion). The `validate_sprint_json.sh:109` `estimated_loc == 0` backstop is credited
   PARTIALLY and precisely (L23).

**Rev 1 (2026-07-31)** — initial version; sent to multi-provider quorum.

**Document created**: 2026-07-31
**Last updated**: 2026-07-31 (revision 3)
