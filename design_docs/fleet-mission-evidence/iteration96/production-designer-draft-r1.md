# M-FLEET-ROLE-ARTIFACT-CONTRACT

**Status**: Planned — **Draft awaiting design-quorum (controller-run) and attended design approval**. Authored by the rotation designer at fleet iteration 96 (`pi:ollama/kimi-k3:cloud`, resolved `recipe … declared:provider-pin`); quorum NOT yet run; no approval asserted.
**Target**: Unversioned (loop-harness skill change; no language release coupling; stays in `design_docs/planned/` root)
**Priority**: P2
**Estimated**: ~0.5 day (M1 ~1.5–2h + M2 ~2h + buffer)
**Dependencies**: None blocking. Source incident ticket `tool-hang:triage-judge:home-metadata-search` (triage iteration 33, fleet iteration 95/96 evidence); independent disposition at `design_docs/fleet-mission-evidence/iteration96/evaluator-report.md` §§4–6.
**Planner-Lane**: opus-required — measured against `tools/launchd/derive-planner-lane.sh`: the fleet mission does not widen `MISSION_PLANNER_ALLOWLIST` (`grep` over `missions/fleet.toml` empty; default at `tools/launchd/mission-control.sh:1439` covers only `tools/launchd/*|.claude/skills/mission-control/SKILL.md|.claude/skills/design-doc-creator/*`), and this design's primary file, `.claude/skills/mission-control/resources/gate-3-route.md` (+ its `.agents` mirror), is outside that default, so the derivation would emit `opus fail-closed:path-not-in-codex-allowlist`. The docs-mission precedent for widening exists (`tools/launchd/mission-env/mission-docs.env:138`) but widening a mission's planner allowlist is a per-mission lane configuration decision, not something this design smuggles in. If an attended ruling later widens the allowlist, this surface is mechanical and codex-feasible.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms) (repo file: `docs/docs/references/axioms.mdx`)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Prose guidance in a skill recipe plus one prose-pinning test suite. No runner verdict, threshold, retry-count, or lane semantics change; `scripts/mission_pi_run.sh` untouched. |
| A2: Replayability | 0 | No trace/NDJSON format or capture change. |
| A3: Effect Legibility | +1 | Harness-level: the role's expected read/artifact surface becomes *declared* (named exact paths) instead of ambient post-hoc discovery. No language effect-row change. |
| A4: Explicit Authority | +1 | Narrows what an unattended role is invited to touch (explicit broad-discovery prohibition over `$HOME`, session stores, metadata trees). Grants no new access. |
| A5: Bounded Verification | +1 | Adds one bounded driver-side test suite (mktemp-rooted fixtures, explicit positive and negative controls) and pins the preserved retry wording against silent regressions. |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | Directives become self-describing to the executing model: named inputs, named outputs, worktree+SHA, transport-verified identity. Measured cost of the absence: one full slot loss (1119 s, rc 18, no report). |
| A8: Minimal Syntax | 0 | No syntax. |
| A9: Cost Visibility | 0 | No metering/ration/billing change; slot-loss reduction is an outcome, not a mechanism. |
| A10: Composability | 0 | Reuses existing sync machinery and the existing rc18 retry rule unmodified; neither is extended structurally. |
| A11: Structured Failure | +1 | Converts "input not found → sweep the disk" into a typed, reportable condition: *missing named INPUT is reported against its supplied path*, distinct from an empty-but-successful read and from a not-yet-created OUTPUT. |
| A12: System Boundary | 0 | No boundary change; sandbox policy and fences untouched. |

**Net Score: +5** → **Decision: Move forward** (≥ +2)

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced — no code path changes.
- [x] A3 (Effects): No hidden side effects — the test suite's fixtures live entirely under `mktemp -d`; acceptance commands write nothing outside a synthetic worktree root.
- [x] A4 (Authority): No ambient access granted; the change *removes* implicit license to sweep, grants nothing.
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis — the contract is machine-consumable directive structure.

## Problem Statement

An unattended mission **role** directive that does not name its artifacts and supplies no identity invites the role's model to *discover* them — and that discovery, measured once on the role lane, cost a whole slot with zero output.

**Current State (measured, historical provenance):**

- **Triage iteration 33, first judge attempt** (`design_docs/fleet-mission-evidence/iteration95/new-ticket-first-attempt.json`): fenced run, verdict `tool_hang`, **rc 18** after **1119 s**, **148 completed tool executions**, zero provider errors, zero commits, worktree unchanged since start. The one open tool was a literal bash command: `printf 'HOME=%s\n' "$HOME"; ls -la "$HOME/.pi" … ; find "$HOME" -maxdepth 4 -type f \( -name '*.jsonl' -o -name '*.json' \) -mmin -120 -print …` — a broad `$HOME` metadata sweep. The runner's typed verdict records the literal command with `$HOME` unexpanded; no expanded HOME value was read into evidence.
- **Trigger, from the directive text itself** (`design_docs/fleet-mission-evidence/iteration96/original-directive-excerpts.json`, sanitized; originals at `/tmp/triage-i33/judge/directive.txt`, sha256 `90536f…6dd7` as recorded by the controller): the original directive **named no input artifacts before that paragraph** and asked the judge to "*Return actual provider/model from your session metadata if available, not a guessed identity.*" The model went looking for its own identity in session/HOME metadata trees — exactly the trees the directive failed to rule out.
- **The prescribed remedy worked once** (`design_docs/fleet-mission-evidence/iteration95/new-ticket-retry.json`; retry directive `/tmp/triage-i33/judge-retry/directive.txt`, sha256 `07f702…bbd4`): the ONE same-lane retry whose directive (a) supplied the identity verbatim ("The transport has independently verified … provider=ollama and model=minimax-m3:cloud … do not research your identity") and (b) explicitly banned the hung command and "absolutely no HOME scans, hidden-store/session lookup, environment inspection, global file discovery" returned **rc 0 in 274 s** (75 tool executions, 1 `agent_end`, 2 files changed — a report was written). Substance caveat held explicitly: `/tmp/triage-i33/judge-retry-worktree/JUDGE_REPORT.json` has the required shape (5 items, 6 auxiliary — verified structurally), but **the retry's rc 0 does not prove every item was substantively judged**, and this design claims no such thing.
- **The instruction gap is standing, not one-off.** The *reactive* retry clause exists (`gate-3-route.md:476-480`); the *preventive* clause does not: today nothing in the pi role recipe requires a directive to name its inputs/outputs, to supply transport-verified identity, or to forbid broad discovery. The identity-research invitation in the original directive was conformant with every standing rule.
- **Class history with the lane caveat attached** (evaluator report §4.3, adopted here as scoping discipline): the same *class* — a long/unreturning command eating a slot — has three occurrences across three missions (stapledon 2026-09-28 `find ~ -maxdepth 6`; fleet 2026-10-04 no progress across 600 s of samples; triage 2026-11-07, this incident, recorded under D-FLEET-34's ticket family). The first two were **controller**-lane events, which already carry a per-command cap (`mission-control.sh:2578`); the **role**-lane recurrence count is N=1. This doc cites the class as motivation for the role-runner gap's existence, not as a measured frequency for it.

**Impact:**
- Who: every mission's pi executor/planner/evaluator/designer run through `scripts/mission_pi_run.sh`; the controller that must recover the slot; the iteration budget.
- How significant: one whole slot lost (1119 s → rc 18, no report) per occurrence, plus the fallback/flag bookkeeping. Annoyance-to-blocker depending on role (an evaluator slot loss = no independent verdict that fire).

## Goals

**Primary Goal:** Make the instruction that recovered the incident (named artifacts, supplied identity, no broad discovery) a **standing, proactive clause in the shared pi role directive recipe** — prevention via the instruction channel that is measured working — while precisely preserving what legitimate scoped work needs.

**Success Metrics:**
- 100 % of pi role directives composed under the recipe contain the four contract elements (named inputs, named output, worktree+SHA, identity disposition) — *checked by generated-directive arms in the new test suite*, not by hoping.
- rc18 retry wording preserved verbatim AND extended with a MUST-forbid-literal-command sentence — pinned by string assertions.
- Mirror parity (`.claude` ↔ `.agents`) holds at merge and is gated — `make test-launchd-drivers` green including `tools/launchd/test_agents_skills_sync.sh`.
- Zero tolerance retained for the prohibited shapes: the contract's prohibition list is pinned as strings, and a synthetic counter-example directive containing an identity-research invitation is *detected* by the suite.
- No acceptance command writes outside a synthetic (`mktemp -d`) root; no seeded writes to real shared state (codex-lane false-greens V327 lesson applied at authoring time).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Contract is **instruction-level standing guidance**, not a tool-boundary enforcement | An enforcement gate (block/refusal) has false-positive risk against the legitimate scoped searches this design explicitly preserves (`eval-shell-guard.ts`'s refusal regex is the measured example of how blunt that instrument is — it would NOT have refused the hung command); the instruction channel is measured working (retry rc 0 in 274 s) | human (this doc) | design | high |
| Contract block lives in `gate-3-route.md` as a marked block beside `PI_EVALUATOR_SESSION_HANDSHAKE`, referenced from the pi recipe and the rc18 clause | Single composition point for all pi role directives; marker convention already exists (lines 494/522); `.agents` parity rides the existing sync with zero new machinery | human (this doc) | design | med |
| Scope is the **pi role recipe only** (the `mission_pi_run.sh` lane that hung); extending the identical clause to the codex recipe is deferred | The incident and its evidence are pi-lane; the codex recipe composes directives differently and its extension is cheap but unproven — minimal blast radius beats speculative uniformity | human (this doc) | design | med |
| rc18 retry rule **preserved**: exactly ONE same-lane retry, forbid the command, FLAG a harness ticket; the retry directive MUST name the literal hung command (sanitized shape permitted) | The retry rule was exercised exactly as written and worked; changing it re-opens settled policy | human (this doc) | design | low |
| P2 (per-command bash cap on role runs via unmodified `controller-bash-cap.ts`) is **excluded — deferred as a separate optional proposal requiring attended choice** | Behavior band, policy-class question and protected-surface risk are NOT robustly bounded (see Risks + Deferred); keeping P1 unbundled prevents the cheap half being held hostage by the unjustified half | human (this doc) | design | high |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Contract block is a new marked block in `.claude/skills/mission-control/resources/gate-3-route.md` immediately after the `PI_EVALUATOR_SESSION_HANDSHAKE_END` marker (line 522), with one-pointer sentences added in the pi recipe's real-run section (~line 355 area) and the rc18 clause (lines 476-480) — mirrors via the existing `tools/launchd/sync-agents-skills.sh`.
- [x] Scope: pi role directives only; codex-recipe adoption is a Deferred Decision (human at review).
- [x] The worked example uses ONLY the sanitized shape from the iteration-95 evidence artifacts — literal `$HOME` preserved exactly as recorded; no expanded HOME value exists in evidence and none may be introduced.
- [x] P2 is not in this doc's milestones, acceptance criteria, or changed paths; it appears only as Deferred/Non-Goal + Risks.
- [x] No `.pi/extensions/**` change (D-FLEET-3 parks that for Mark) and no `tools/pi-extensions/**` change outside `sandbox/**` (D-FLEET-6): this design touches neither.

## Solution Design

### Overview

Add one standing marked block — the **Role Artifact Contract** — to the shared mission-control routing recipe, right where pi role directives are composed, and one MUST-sentence to the existing rc18 retry clause. The contract tells the directive-writing controller (and is itself visible to any role agent that can read the skill) what every pi role directive MUST contain and MUST forbid; the retry MUST-sentence turns "telling it not to run that command" from description into requirement, with a worked sanitized example.

This is standing proactive guidance and a concrete artifact contract in the shared recipe — **not sandbox enforcement, not a prompt-obedience guarantee.** It does not claim to deterministically eliminate every tool hang. Its mechanism is the measured one: the incident's trigger was a directive defect (identity not supplied, discovery not ruled out), and the one controlled replay with the defect repaired succeeded.

### Architecture

**Components:**

1. **The Role Artifact Contract block (new, marked).** Inserted after line 522 of `gate-3-route.md`. Normative content:
   - **Named INPUT artifacts**: every artifact the role is expected to read is named by exact path in the directive. 
   - **Named OUTPUT artifacts**: every artifact the role must produce is named by exact path; an absent output path is *expected* (the author creates it) and is NOT a missing input.
   - **Working-tree context**: the worktree root path and the immutable base SHA the run reviews.
   - **Identity disposition**: when a role must record provider/model, the directive supplies the **transport-resolved** identity verbatim ("transport-provided metadata — do not research your identity"); if it is unavailable to the controller, the directive instructs the role to record `unavailable` against the supplied telemetry paths, never to derive it from session/HOME/metadata trees.
   - **Broad-discovery prohibition, precisely defined.** *Prohibited:* broad or unbounded filesystem sweeps — `find`/`ls -R`/`grep -r`/glob-walks rooted at `$HOME`(in any quoting form), `/`, session stores (`~/.pi`, `~/.claude`, `~/.ailang`, equivalent model-session stores), or metadata trees; environment dumps (`env`/`printenv`) aimed at identity or credential discovery; reading the previous attempt's artifacts unless explicitly named. *Remains permitted (explicitly not suppressed):* exact-path reads of supplied artifacts; **bounded repository-scoped searches** (`rg`, `git grep`, `git log/show`, `ls` of named directories inside the worktree); reads of explicitly named sibling evidence paths.
   - **Missing-input rule**: a required INPUT that is absent/unreadable at its supplied path is **reported against that supplied path** (name it, state what was checked, proceed only on what remains or stop per the role's own gate) — it is never license to search for the artifact elsewhere, and a **missing report is not an empty-but-successful result**.
2. **rc18 clause extension (edit in place, lines 476-480).** Keep every existing sentence verbatim — "**`tool_hang` (rc 18) is NOT a lane failure at all** … Retry ONCE on the same lane … and FLAG the command as a harness ticket" — and append: the retry directive **MUST name the literal `hung_tool` command it forbids** (verbatim where shareable; the *sanitized shape* where it is not — worked example: `find "$HOME" -maxdepth 4 -type f \( -name '*.jsonl' -o -name '*.json' \) -mmin <N>` "a HOME metadata sweep of this shape"; never expand `$HOME`), **MUST restate the artifact contract**, and MUST supply the transport-verified identity when the hang's context shows identity research was the trigger.
3. **Driver-side conformance suite (new test).** `tools/launchd/test_gate3_artifact_contract.sh`, Bash 3.2, all fixtures under `SCRATCH="$(mktemp -d)"` with `trap 'rm -rf "$SCRATCH"' EXIT`, in the pattern of `tools/launchd/test_agents_skills_sync.sh` (markers, string pins, explicit negative controls). Wired into `LAUNCHD_SUITES` in `make/test.mk`, i.e. run by `make test-launchd-drivers`.

What this design deliberately does NOT build: any hook, guard, sandbox rule, timeout, or extension load. The contract is recipe prose plus a prose-pinning suite. That is the minimality the ticket's evidence justifies (N=1 on this lane), and the boundary that keeps it safely inside the fleet allowlist.

### Implementation Plan

**M1: Contract clause + rc18 MUST wording** (~1.5–2h)
- [ ] Draft the marked Role Artifact Contract block (start/end markers, mirroring the `PI_EVALUATOR_SESSION_HANDSHAKE` marker convention at lines 494/522) into `.claude/skills/mission-control/resources/gate-3-route.md` after line 522 (~45–60 lines).
- [ ] Add the one-sentence pointer from the pi recipe real-run section (~line 355 area, ~2–4 lines): role directives MUST satisfy the Role Artifact Contract below.
- [ ] Extend the rc18 clause (lines 476-480) with the MUST-name-literal-command / MUST-restate-contract sentences and the sanitized worked example (~6–10 lines); preserve every existing sentence verbatim, including the ONE-retry rule.

**M2: Sync, conformance suite, wiring** (~2h)
- [ ] Run `tools/launchd/sync-agents-skills.sh` (verbatim rsync, existing machinery — no edits to it); verify `--check` exits 0.
- [ ] Write `tools/launchd/test_gate3_artifact_contract.sh` (~150 LOC) per the Testing Strategy arms.
- [ ] Add the suite to `LAUNCHD_SUITES` in `make/test.mk` (+1 line).
- [ ] Run the Acceptance Criteria; all fixtures synthetic-rooted; record evidence in the Gate-4 row.

### Files to Modify/Create

**New files:**
- `tools/launchd/test_gate3_artifact_contract.sh` - (~150 LOC) Bash 3.2 conformance suite: contract-block presence, generated-directive conformance + counter-example detection, rc18 wording pins, missing-input/output distinction pins, negative controls. All fixtures under `mktemp -d`.

**Modified files:**
- `.claude/skills/mission-control/resources/gate-3-route.md` - (+45–75 lines est.) new marked Role Artifact Contract block; one pointer sentence in the pi recipe; rc18 clause extended (existing sentences verbatim).
- `.agents/skills/mission-control/resources/gate-3-route.md` - (content-identical to the above) produced by the EXISTING `tools/launchd/sync-agents-skills.sh`; parity enforced by the existing `tools/launchd/test_agents_skills_sync.sh`.
- `make/test.mk` - (+1 line) add the new suite to `LAUNCHD_SUITES`.

All four paths are inside the fleet Authority allowlist (`design_docs/fleet-mission.md` § Authority: `.claude/skills/mission-*/**`, `.agents/skills/mission-*/**`, `tools/launchd/**`, `make/test.mk`).

## Examples

### Example 1: What the original directive said vs. what the contract requires

**Before (triage 33 original directive, sha256 `90536f…6dd7` — verbatim excerpt):**
```
Return actual provider/model from your session metadata if available, not a guessed identity.
```
Read-path paragraph named the four judge-input files only; nothing ruled out discovery. **Result:** 148 tool calls later, one broad `$HOME` `find` still open at the stall boundary → rc 18 `tool_hang`, 1119 s, no report.

**After (what the contract clause makes standing — the shape the retry directive already carried, sha256 `07f702…bbd4`):**
```
The transport has independently verified your actual message metadata as provider=ollama and
model=minimax-m3:cloud. Record those identity values as transport-provided metadata and do not
research your identity. Absolutely no HOME scans, hidden-store/session lookup, environment
inspection, global file discovery, or access outside this isolated worktree.
```
**Result (same lane, one retry):** rc 0, 274 s, 75 tool executions, report written. Under this design, a directive composed WITHOUT the identity disposition is a contract violation the conformance suite can be pointed at — prevention instead of recovery.

### Example 2: The contract's precision — what stays permitted

The clause names its prohibition list explicitly so it cannot be read as "stop searching":
- **Permitted:** `rg "derive-planner-lane" tools/launchd/` inside the named worktree; `read` on each supplied artifact path; `ls` the supplied `judge-input/` directory; `git show <sha>:<path>` for immutable source.
- **Prohibited:** the incident shape — `find "$HOME" -maxdepth 4 -type f \( -name '*.jsonl' -o -name '*.json' \) -mmin -120` — quoted as the sanitized worked example in the rc18 clause (literal `$HOME`, exactly as the typed verdict preserved it).
- **Missing input:** "manifest.json absent at the supplied path /tmp/…/judge-input/; checked that exact path only; proceeding on remaining named items / stopping" — a typed gap report. Not: sweep `/tmp`, `$HOME`, or prior-attempt directories for a copy.
- **Not-yet-created output:** "JUDGE_REPORT.json does not exist yet" is the expected pre-write state and MUST NOT be reported as a missing input nor treated as evidence a report exists.

## Success Criteria

- [ ] `/bin/bash tools/launchd/test_gate3_artifact_contract.sh` exits 0 with all arms PASS and its negative controls firing (arm details under Testing Strategy).
- [ ] `/bin/bash tools/launchd/sync-agents-skills.sh --check` exits 0 (mirror parity), and `tools/launchd/test_agents_skills_sync.sh` stays green.
- [ ] `make test-launchd-drivers` green (includes the new suite and the sync suite).
- [ ] String pins hold on the changed recipe: the pre-existing phrase "Retry ONCE on the same lane" survives verbatim; the new MUST-forbid-literal-command sentence and the sanitized `$HOME`-literal example are present; the contract block's four element names (named inputs, named output, worktree+SHA, identity disposition) are present.
- [ ] `git diff --name-only` for the implementation limited to the four paths above (plus this doc's Status flip if landed together at the controller's discretion; controller owns final commit either way).
- [ ] No acceptance command, test arm, or fixture writes outside a synthetic root: verified by reading the diff for `>`/`>>`/`tee`/`rm`/`mv` against `$HOME`, `~/.ailang`, or any shared path (codex-lane false-greens rule 5a) AND by the suite itself using only `mktemp -d` scratch with cleanup trap. Seeded writes to real shared state: none — all guards are observation-only.
- [ ] All tests passing; documentation updated = this doc + the two recipe copies (the sync target is documentation parity by construction).

## Testing Strategy

**Unit/driver-side tests (the whole plan — there is no runtime code):**
`tools/launchd/test_gate3_artifact_contract.sh`, `/bin/bash` 3.2, fixtures only under `SCRATCH=$(mktemp -d)`:
- **Arm 1 — block presence and markers.** `.claude/skills/mission-control/resources/gate-3-route.md` contains exactly one `ROLE_ARTIFACT_CONTRACT` start/end marker pair (convention copied from the `PI_EVALUATOR_SESSION_HANDSHAKE_START/END` markers verified at lines 494/522); `.agents` copy byte-identical (delegates to `sync-agents-skills.sh --check`).
- **Arm 2 — generated-directive conformance (meaningful example check).** Generate a synthetic evaluator-style directive inside `$SCRATCH` per the recipe (handshake preamble + contract elements) and assert it contains: ≥1 named input path, the named output path, a worktree+SHA line, the identity-disposition sentence, and the prohibition sentence. Then generate a **counter-example** (a directive omitting the identity disposition and carrying the incident's "session metadata if available" invitation) and assert the suite's checker **flags it** — a checker that passes everything is vacuous; the counter-example is the discriminating control.
- **Arm 3 — rc18 regression pins.** The preserved "Retry ONCE on the same lane" string still present; the new MUST-name-literal-command sentence present; the worked example present with literal `$HOME` (assert the raw string `find "$HOME" -maxdepth`; assert NO line in the recipe expands a real home path into it).
- **Arm 4 — missing-input vs. output distinction pins.** Contract text contains both distinguishing sentences (missing INPUT reported against supplied path; absent required OUTPUT is expected). The synthetic conformant directive from Arm 2 names an output path that does not exist in `$SCRATCH`; the checker does NOT flag it — while a synthetic directive whose named INPUT path is absent IS handled per the contract's report rule (pin the rule's text, not a model's behavior).
- **Arm 5 — negative controls.** Removing the block (drifted copy in `$SCRATCH`) is detected; a recipe copy missing the rc18 MUST sentence is detected. Mirrors the negative-control discipline of `tools/launchd/test_agents_skills_sync.sh` (drifted-copy arm).

**Integration tests:**
- `make test-launchd-drivers` (runs the new suite, `test_agents_skills_sync.sh`, and the sibling harness suites in one pass).

**Manual testing (controller, at review):**
- Compose one synthetic pi role directive under the updated recipe and read it end-to-end: the four elements present, nothing broad authorized. Read-only exercise; no spawn required for the design gate.

## Deferred Decisions

The following are intentionally left open:

- **Applying the identical contract clause to the codex executor recipe** — human at review; the design deliberately scopes to the pi recipe that owns the measured incident.
- **Marker names and exact sentence wording of the contract block** — agent may choose during M1 within the pinned string set Arms 1–4 require.
- **Whether the block also gets a pointer from the handshake preamble for other roles (planner/executor) beyond the pi recipe pointer** — agent may choose where the recipe already composes pi role directives; keep additions pointer-sized.

## Deferred proposal — P2 (OPTIONAL, SEPARATE, attended decision required; NOT part of this design)

> The independent evaluator (`iteration96/evaluator-report.md` §5.2–5.3) proposed, as its item P2, wiring the **existing, unmodified** `tools/pi-extensions/controller-bash-cap.ts` into role runs of `scripts/mission_pi_run.sh` (exporting `MISSION_PI_BASH_CAP_S`), cap strictly below the 600 s stall bound. This design **defers that proposal in full**. It is listed in Non-Goals and appears nowhere in milestones, acceptance criteria, or changed paths. **No cap value, threshold, env var, extension activation, runtime wiring, lane, or billing change is chosen or introduced here.** If robustly justified later, it needs its own design doc and an attended choice. The risks that deferral is protecting:

1. **The changed behavior band is real and unmeasured for role runs.** With cap < stall, a *silent* command that today dies fatally at the stall boundary (rc 18, whole run lost) would instead return `timeout:<s>` at the cap and let the model continue — a strict improvement for that shape. But the band `(cap, stall]` also contains *chatty* long commands: pi's stream carries `tool_execution_update` events (measured first-party: 232 such events in the incident run's filtered stream), and the runner's awk filter routes everything that is not `message_update` — including tool updates — into the progress file, refreshing the mtime progress clock (`scripts/mission_pi_run.sh` filter + poll loop). **A command emitting periodic updates can therefore remain live past the stall window today; "every command longer than stall already dies" is NOT mechanically true — duration is not what kills, silence is.** A cap below stall would truncate such commands that previously completed. Real existence proof that long legitimate role-adjacent commands exist: D-FLEET-17's evidence measured a `go test` package at 395 s. Choosing a cap is a measurement problem this design does not solve.
2. **Controller-vs-role differences.** The controller spawn carries the cap plus `--session-id` resume semantics, an `--append-system-prompt` controller note, and the driver's own watchdog rhythm (`mission-control.sh:2560-2578`); the role runner runs `--no-session` with typed verdicts and the mtime progress clock. Guidance text rendered by the unchanged extension reads "## Command time limit (mission **controller**)" — a role agent would see controller wording, and the file is outside the fleet's scope (D-FLEET-6 admits only `tools/pi-extensions/sandbox/**`), so the fleet cannot reword it. **A mechanically unchanged file loaded into a new lane is not behavior-neutral**: both its hooks (`tool_call` clamp, `before_agent_start` system-prompt append) fire the moment it is loaded.
3. **Protected-policy/approval risk.** The evaluator report asserts P2 moves "nothing policy-class"; **that is not established.** Any per-command ceiling adjacent to `--stall-seconds` abuts the settled threshold rulings D-FLEET-4/-7 ("sample counts and the 600 s budget stay unchanged"), and the Authority section parks any routing/threshold-adjacent or billing-touching change for Mark. Whether a role-run cap is policy-class is itself an attended question.
4. **Enforcement instruments have measured blind spots here.** `eval-shell-guard.ts`'s refusal rule (`isUnboundedFind`: rooted AND lacking `-maxdepth`) would NOT have refused the incident command, which carried `-maxdepth 4`; and extending `sandbox.mission.json`'s `denyRead` into the home tree changes what every role in every mission may read — an attended decision, explicitly out of scope of the incident ticket.

## Non-Goals

**Not attempted in this feature:**
- **P2 (per-command cap on role runs)** — deferred per the section above; separate optional proposal, attended choice, own design doc.
- Any change to `tools/pi-extensions/**` outside `sandbox/**` (D-FLEET-6: out of fleet scope), to `.pi/extensions/**` (D-FLEET-3: parks for Mark), to `internal/executor/**` or `cmd/ailang/**` (outside the allowlist).
- Any change to `sandbox.mission.json` `denyRead`/`allowWrite` — attended decision affecting every mission's roles.
- Any threshold, stall/wall bound, retry-count, lane-order, fallback, quota/ration, or billing change. The ONE-same-lane-retry rule and the 600 s/1800 s bounds stand untouched.
- Sandbox-level enforcement of the prohibition (no hook, no refusal rule) — enforcement is the unjustified half; see P2 deferral.
- Any claim that directive prose deterministically eliminates every tool hang or guarantees prompt obedience — this is standing guidance plus conformance pinning, both of which are honest about only binding what is checkable.
- Resolving or closing `tool-hang:triage-judge:home-metadata-search` — ticket close is the controller's `ailang mission ticket resolve … --sha <commit on origin/dev>` after the fix lands and the done-gate passes. A recovered command never authorized ticket resolution (iteration 95's note stands).

## Timeline

**Iteration N (single mission iteration, ~4.5h across the loop):**
- M1: contract block + rc18 MUST wording (1.5–2h)
- M2: sync + conformance suite + make wiring (2h)
- Acceptance run + Gate-4 evidence (0.5h)

**Total: ~4–5h, one iteration; 2x buffer already included over the raw draft estimate (~2h).**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Prose-only fix can be ignored by a future model; the prevention is probabilistic, not enforced | Med | Accepted and stated (no obedience guarantee claimed). The measured trigger was a directive *defect*, and the one controlled replay with the defect repaired succeeded; enforcement (P2) is deferred rather than bundled, so this doc is not hostage to the unmeasured band. |
| The prohibition is read as "stop searching" and suppresses legitimate scoped repository searches a role genuinely needs | Med | The clause enumerates what REMAINS permitted (exact supplied paths, bounded in-worktree `rg`/`git`, named sibling evidence) and the missing-input rule channels needs into a typed report instead of a sweep. Arm 4 pins this distinction. |
| rc18 clause edit accidentally weakens the settled ONE-retry rule | Low | Preserve-existing-sentences constraint in M1; Arm 3 string pins the retry wording; diff review shows the clause grew without sentence removals. |
| Mirror drift between `.claude` and `.agents` copies (pi/codex read `.agents`) | Med | Zero new machinery: existing verbatim sync + existing parity suite + new block-presence arm; `make test-launchd-drivers` gates merge. |
| Over-broad worked example leaks a real home path into the skill (credential-ish surface) | Med | Worked example is the EVIDENCE file's literal string with `$HOME` unexpanded — the only form that exists in evidence; Arm 3 asserts no expanded path appears. |
| Suite's own fixtures touch real shared state (the V327 class) | Med | All fixtures `mktemp -d` + cleanup trap; acceptance criterion requires reading the diff for out-of-worktree writes; observation-only guards, no seeded writes. |
| Scope creep: someone "while here" adds the cap or a sandbox rule into M1/M2 | Med | Design Freeze + Non-Goals name P2 and sandbox changes as out of scope; acceptance `git diff --name-only` limited to four paths. |

## Related Documents

Duplicate/coverage gate (run at authoring time via `create_planned_doc.sh`, top neural matches read by the author): top planned matches were `m-coordinator-child-env-opencode-retry-storm-recovery-plan.md` (a sprint-recovery bookkeeping note), `m-entropy-budgets.md` (AI-call entropy budgets), `m-ifc-cross-module-labels-sprint-plan.md` (IFC label sprint) — **all unrelated topics; the high similarity scores are keyword noise** ("contract/recovery/plan" stems). No existing doc covers what a role directive must declare. Distinct from:

- **[m-fleet-slot-kill-reaps-descendants-sprint-plan.md](m-fleet-slot-kill-reaps-descendants-sprint-plan.md)** — driver-side watchdog kill-tree reaping (process supervision). Sibling harness fix in the same broad "slots lost to hangs" family, different mechanism (post-kill reaping vs. pre-hang directive contract), no overlap in files or gates.
- **D-FLEET-34 / [fleet-mission.md](../fleet-mission.md) row (OPEN)** — `stall:watchdog:slow-on-hung-tool`: ask 1 (does the controller bash cap reach pi controllers) answered yes (#1391); ask 2 (shorten pi thresholds) recommended WAIVED. Neither ask addresses a **role-run** command bound or directive content: the row itself was about *controller* thresholds. This doc changes no threshold.
- **[evaluator-report.md §§4–6](../fleet-mission-evidence/iteration96/evaluator-report.md)** — the independent disposition this design implements (P1) and defers (P2). Two of its claims are deliberately NOT inherited: (i) that P2 is policy-neutral — here marked not-established; (ii) any implication that every over-stall command already dies — refuted by code inspection (progress-clock semantics, see Deferred P2 risk 1). Its strongest objection (N=1, remedy already works, don't churn a shared harness) is answered by bundling only the cheapest mechanism-matched half and pinning it with tests.
- **[m-harness-mission-loop.md](m-harness-mission-loop.md)** — the fleet mission's own charter doc; this design is one of its "every loop-harness defect comes back as a fix" items.
- **Not an AGENTS.md "friction twice" fast-track** — the evaluator's report invoked that doctrine; this doc explicitly does not. The carve-out covers `.pi/extensions/` fast-track fixes and does not replace design approval; this proposal proceeds through quorum → attended design approval → sprint-planner → executor gates.

## References

- **Incident evidence (first-party):** `design_docs/fleet-mission-evidence/iteration95/new-ticket-first-attempt.json` · `…/new-ticket-open-tool.json` · `…/new-ticket-retry.json` · `…/new-ticket-triage.md`
- **Sanitized directives + hashes:** `design_docs/fleet-mission-evidence/iteration96/original-directive-excerpts.json`; originals `/tmp/triage-i33/judge/directive.txt`, `/tmp/triage-i33/judge-retry/directive.txt` (read directly, contents not executed); `/tmp/triage-i33/judge-retry-worktree/JUDGE_REPORT.json` (existence + shape verified only).
- **Independent disposition:** `design_docs/fleet-mission-evidence/iteration96/evaluator-report.md` §§4–6
- **Fleet charter:** `design_docs/fleet-mission.md` § Authority; ledger row D-FLEET-34; queue rows 48 and 61
- **Design Axioms:** [Design Axioms](/docs/references/axioms) — repo file verified at `docs/docs/references/axioms.mdx`

## Verification Log

Every load-bearing codebase claim in this doc carries one row: claim → command → observed result → control. Negative-existence claims carry their own rows with known-positive controls. All commands read-only, bounded (≤30 s), run from the worktree root; no HOME/session/metadata sweep was performed (the two `/tmp` directive files and the three `run.ndjson` streams are artifacts *named by the evidence itself*).

| # | Claim | Command (bounded) | Observed | Control |
|---|-------|-------------------|----------|---------|
| V1 | First attempt: `tool_hang` rc 18, 1119 s, 148 completed tools, no commits/worktree change; hung tool = literal `$HOME` find | read `design_docs/fleet-mission-evidence/iteration95/new-ticket-first-attempt.json` | JSON exactly as cited | `new-ticket-triage.md` restates the same numbers independently |
| V2 | Retry: `ok` rc 0, 274 s, 75 tools, 1 `agent_end`, 2 files changed | read `…/iteration95/new-ticket-retry.json` | JSON exactly as cited | RC/elapsed cross-cited in `fleet-mission.md` queue row 61 |
| V3 | Retry produced a *shaped* report (5 items, 6 aux) — shape only, substance NOT re-verified | `python3 json.load` on `/tmp/triage-i33/judge-retry-worktree/JUDGE_REPORT.json` | keys = provider/model/dev_sha/handshake/items/aux/checks/limitations; items=5, aux=6; provider=ollama, model=minimax-m3:cloud | matches controller's stated count (19,800 bytes, 5/6) |
| V4 | Original directive invited identity research; retry supplied identity + HOME ban | read `/tmp/triage-i33/judge/directive.txt`, `/tmp/triage-i33/judge-retry/directive.txt` | original: "Return actual provider/model from your session metadata if available"; retry: "transport has independently verified … do not research your identity … Absolutely no HOME scans…" | sha256 pair quoted from controller-sanitized `original-directive-excerpts.json` (not recomputed) |
| V5 | Role runner loads NO per-command cap (negative) | `grep -n "bash-cap" scripts/mission_pi_run.sh` → **empty** | no hit | control: `grep -c "bash-cap" tools/launchd/mission-control.sh` → 2 |
| V6 | Runner has no cap env knob (negative) | `grep "MISSION_PI_BASH_CAP_S\|AILANG_EVAL_BASH_CAP_S" scripts/mission_pi_run.sh` → **empty** | no hit | control: `grep -c STALL_SECONDS` same file → 6 |
| V7 | Stall/wall defaults 600/1800 | `grep -n 'STALL_SECONDS=\|MAX_SECONDS=' scripts/mission_pi_run.sh` | `:88` MAX=1800, `:89` STALL=600 | full read of poll loop + usage text |
| V8 | Progress clock = filtered-file mtime; `message_update`→snapshot, all else (incl. tool events)→filtered file | full read of `mission_pi_run.sh` (awk filter + poll loop) | as stated | — |
| V9 | pi streams carry `tool_execution_update` events (so a chatty long command refreshes the clock; duration ≥ stall is NOT mechanically fatal) | `grep -c '"type":"tool_execution_update"'` on the two evidence-named `run.ndjson` files | 232 (attempt), 144 (retry) | control: `tool_execution_start/end` counts 149/148 (attempt; the open tool) and 75/75 (retry; none open) |
| V10 | `tool_hang` verdict = stream_dead reclassified when starts>ends | full read of `mission_pi_run.sh` verdict block | as stated | consistent with V9's 149>148 |
| V11 | `_mc_pi_ext_args` emits only `--no-extensions` + workdir `.pi/extensions/*.ts` | full read of `tools/launchd/lib/pi-ext-args.sh` (29 lines) | as stated | — |
| V12 | Controller lane DOES load the cap; wiring pinned | `grep -n "controller-bash-cap" tools/launchd/mission-control.sh tools/launchd/test_driver_notify.sh` | spawn `mission-control.sh:2578` (comment :2560); pin `test_driver_notify.sh:1003-1005` | bounded read of `mission-control.sh:2548-2582` |
| V13 | `controller-bash-cap.ts`: default cap 540, knob `MISSION_PI_BASH_CAP_S`, clamps `tool_call` timeout + appends "## Command time limit (mission controller)" guidance; fail-loud on malformed env | full read (70 lines) | as stated | own unit file `controller-bash-cap.test.ts` exists, run by `test-pi-extensions` (make/test.mk:39) |
| V14 | `eval-shell-guard.ts` refusal would NOT have refused the incident command (it carries `-maxdepth 4`); eval cap knob `AILANG_EVAL_BASH_CAP_S` default 120 | full read (82 lines) | `isUnboundedFind` = rooted `&& !/-maxdepth \d/` | control: `REFUSAL` string present |
| V15 | Mission sandbox denies reads only for credential-shaped paths; home tree readable | full read of `tools/pi-extensions/sandbox/sandbox.mission.json` | denyRead = `.ssh/.aws/.gnupg/.config/ailang/secrets.env/.pi/agent/models.json` | — |
| V16 | `controller-bash-cap.ts` is NOT a mirrored asset (negative) | `ls .pi/extensions/` (no match) + `internal/executor/pi/profile_assets.go:88-90` (`evalShellGuardFile` only) + `internal/executor/pi/profile_assets/` contains `eval-shell-guard.ts` | cap absent from both mirror surfaces; eval guard present | positive control: eval-shell-guard IS mirrored |
| V17 | rc18 retry clause current wording | read `.claude/skills/mission-control/resources/gate-3-route.md:476-480` | "…not a lane failure at all… Retry ONCE on the same lane with the directive telling it not to run that command, and FLAG…" | — |
| V18 | Handshake marker convention exists (494/522) | `grep -n PI_EVALUATOR_SESSION_HANDSHAKE_ gate-3-route.md` | START 494, END 522 | — |
| V19 | `.claude` and `.agents` mission-control copies currently identical | `diff -q` both `gate-3-route.md` copies; `/bin/bash tools/launchd/sync-agents-skills.sh --check` | `MIRROR_EQUAL`; `agents skills in sync`, rc=0 | negative-control arm already inside `test_agents_skills_sync.sh` |
| V20 | The directive-named path `scripts/test_agents_skills_sync.sh` does NOT exist; actual suite = `tools/launchd/test_agents_skills_sync.sh`, in `LAUNCHD_SUITES` | `ls scripts/test_agents_skills_sync.sh` → **No such file**; `grep` make/test.mk | actual path wired in `LAUNCHD_SUITES` (~line 103), target `test-launchd-drivers` (:132), part of `ci` (make/ci.mk:11) | positive control: sibling scripts `scripts/test_mission_pi_run.sh` etc. exist |
| V21 | Fleet Authority allowlist covers all four changed paths; exclusions as stated | read `design_docs/fleet-mission.md:75-94` | allowlist: `tools/launchd/**`, `.claude/skills/mission-*/**` + `.agents` mirrors, `scripts/mission_*`, `make/test.mk`, `tools/pi-extensions/sandbox/**` only; `.pi/extensions/**` parks for Mark | — |
| V22 | D-FLEET-34 content and posture | read `fleet-mission.md:178` (ledger) + `:319`/`:333` (queue rows 48/61) | OPEN; ask1 = ghost of #1391 (controller cap, 540 s, `MISSION_PI_BASH_CAP_S`); ask2 = shorter thresholds, recommended waived; no role-run command-bound ruling | ≤15 ledger rows read per call |
| V23 | Planner-Lane derivation mechanics and fleet posture | read `tools/launchd/derive-planner-lane.sh` (full flow); `grep -n PLANNER_ALLOWLIST missions/fleet.toml` → **empty** | `**Planner-Lane**:` field = codex-ok|opus-required; Files-section paths must match default allowlist (mission-control.sh:1439), which excludes this doc's primary file | positive control: `missions/docs.toml` comment + `tools/launchd/mission-env/mission-docs.env:138` widening precedent |
| V24 | Duplicate/coverage gate | `create_planned_doc.sh m-fleet-role-artifact-contract` neural+simhash results; headers of top-3 planned matches read; `grep -rli "role-artifact\|artifact contract"` over design_docs | top matches unrelated (V1 row in Related Documents); single implemented hit = browser-session artifact contract (unrelated) | existing `m-fleet-slot-kill…` sprint plan read in full, distinct |
| V25 | Acceptance-safety constraints (seeded-write ban, observation-only guards) | read `.claude/skills/mission-control/resources/codex-lane-false-greens.md` (full) | rules 5a-5d adopted verbatim into Success Criteria/Arms | — |
| V26 | Scaffold + version context; target folder = planned root (unversioned) | `create_planned_doc.sh m-fleet-role-artifact-contract` | file created at `design_docs/planned/m-fleet-role-artifact-contract.md`; current AILANG v0.57.4 | `ls` confirms path post-run |
| V27 | Runner + sync tooling parse clean at base | `/bin/bash -n` on `mission_pi_run.sh`, `sync-agents-skills.sh`, `test_agents_skills_sync.sh` | all "syntax OK" | — |

**Named-artifact availability report (per this doc's own contract doctrine):** exactly one artifact named to the author was absent: `scripts/test_agents_skills_sync.sh` (directive's path list). Reported here against its supplied path; the actual suite was located in-repo via a bounded grep of make/test.mk (V20) rather than any broader discovery. All other named artifacts existed and were read directly.

## Future Work

- **P2 design doc** (if an attended ruling wants the role-run cap): must own the `(cap, stall]` band measurement across role-run workloads, the controller-wording question for the unmodifiably-scoped extension file, and the policy-class question. This doc deliberately leaves it unstarted.
- Applying the contract clause to codex-recipe role directives (deferred decision above).
- A checker that lints *composed* role directives at Gate 3 against the contract before spawn (this design lints the recipe text and synthetic directives only; linting real compositions is a controller-workflow change, attended).

---

**Document created**: 2026-11-08 (fleet iteration 96, unattended mission stage — rotation designer `pi:ollama/kimi-k3:cloud`; scaffold via `create_planned_doc.sh`)
**Last updated**: 2026-11-08
