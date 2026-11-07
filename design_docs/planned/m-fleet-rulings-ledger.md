# Fleet-Wide Rulings Ledger and Doctor Contradiction Checker

**Status**: Planned (DRAFT — revised once under quorum R1; nothing here is approved policy until Mark rules)
**Target**: v0.57.5 (next patch; doc deliberately unversioned in `planned/`)
**Priority**: P1 (Medium)
**Estimated**: 4 days (mechanism+binding ~2d, tests ~1.25d, docs/help ~0.75d, at 2x raw estimate)
**Dependencies**: None. Draft PR #2082 (`a58e08e68` + `399cf799b`) corrects stale World-fork text in
`missions/world.toml`; it is **not rebuilt here** — and quiet activation of the seed ruling's claims
additionally requires it to land: both seed literals are still present on the unchanged base (V27/V28).
**Planner-Lane**: codex-ok

> **Provenance.** Fleet iteration 93, D-FLEET-25 option A's undrafted mechanism step, ruled BUILD-eligible
> by the independent pick-judge (`design_docs/fleet-mission-evidence/iteration93/pick-judge.md`, MiniMax,
> SHA `a3bb86291`). Draft only: nothing here resolves D-FLEET-25, ratifies policy, or activates
> precedence — those are Mark's (§ Design Freeze, § Ratification Map). **Scaffold note (raw
> observations only):** the prescribed scaffolder could not run — its neural docs-search step requires
> the rig GPU lock, and `rig_lock_acquire wait` did not return within the session's 180s command bound;
> no cause is claimed (not measured), no retry attempted. This doc follows the scaffolder's embedded
> template; related documents come from documented **non-neural** searches (SimHash `ailang docs
> search`, plus scoped `rg` with positive controls) — **no neural similarity score is claimed.**
> Revision R1: quorum round 1 blocked (all three present reviewers reject; Anthropic absent — quota;
> reserve `gpt6-1-sol` served); objections and fixes are itemized in § Quorum Revision Log.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Byte-exact substring matching over a fixed corpus; sorted findings; the approved-claimset comparison is SHA-256 over exactly specified bytes of two fixed blobs — same inputs → same report, every run. |
| A2: Replayability | 0 | No traces, no execution paths touched. |
| A3: Effect Legibility | 0 | Read-only checker; no new effects (git reads only). |
| A4: Explicit Authority | +1 | Enforcement is bound to the exact claimset bytes of the human-approved flip commit; an unattended claim edit yields a typed mismatch and enforces nothing; resolution stays in the two existing human channels. |
| A5: Bounded Verification | +1 | Every check is local and size-capped: literal ≤ 256 B, ≤ 256 claim lines, ≤ 32 resolved-with-claims rows, file ≤ 8 MiB, total corpus ≤ 64 MiB / ≤ 512 files, helper and blob reads timeout- and size-capped. |
| A6: Safe Concurrency | 0 | No concurrency. |
| A7: Machines First | +1 | Machine-checkable claim lines instead of prose-only policy; structured `Finding` rows a loop can parse. |
| A8: Minimal Syntax | 0 | No language syntax; a two-verb line format deliberately too small to grow syntax obligations. |
| A9: Cost Visibility | 0 | No resource semantics. |
| A10: Composability | +1 | Reuses `Registry`/`Mission`/`Paths`/`Finding`/`Severity`, all three ledger scripts unchanged, `internal/gitexec` for git reads (existing precedent), the existing provenance helper as authority oracle. |
| A11: Structured Failure | +1 | Missing/malformed ledger, unknown IDs, oversize inputs, contradictions, authorization mismatch, unavailable/undetermined/unauthorized provenance, unreadable blobs — distinct typed `Finding`s, always fail-closed on the affected ruling. |
| A12: System Boundary | 0 | Reads only the fleet's own repos/clones plus this repo's git history; writes nothing anywhere. |

**Net Score: +6** (A1+A4+A5+A7+A10+A11) → **Decision: Proceed to quorum** (this doc does not self-approve).

### Hard Violation Check

- [x] A1/A3: no implicit nondeterminism; no writes, no network, no ambient access
- [x] A4/A7: no new write authority (enforcement bound to attended approval commits, human-only resolution); claims/findings are machine-structured, human prose stays human

## Problem Statement

**Ticket**: `policy:fleet-rulings-not-propagated-to-charters-and-skill` — fleet decision row **D-FLEET-25**
(`design_docs/fleet-mission.md:174`), option A (recommended). "A ruling still reaches only the layers
someone edits (Go, driver, shared skill + `.agents` mirror, each charter)."

**Current State:**

- A ruling Mark has made exists in **whichever layers someone happened to edit**. Historical example —
  cited ONLY as the propagation case in point, not as proposed policy: the independence preference ruling
  (V1 `D-62`/`D-63`, 2026-09-08, gate-3-route.md:257) had to be re-imported **by hand** into World's
  charter (`D-WORLD-48`, world-mission.md:967). Nothing checks that layers stay consistent. (The
  independence question itself is NOT seeded here — § Independence Conflict.) The ticket's two cited
  contradictions were fixed by hand and are gone, but the **class** is unfixed.
- Concretely: `missions/world.toml:3` still says "WORLD IS A FORK" while `doctor.go:281` reports no
  fork (`row.Fork = m.Driver != ""` — World sets no `driver` key); PR #2082 removes it, but the
  recurrence risk remains. Each charter's decision ledger is an **ask/provenance record for that
  mission** — correctly so — but no fleet-wide statement of policy outranks charter prose, so "which
  text wins" is undefined.

**Impact:** all 7 mission loops re-read their charter and the shared skill every fire; contradictory
guidance is consumed by an unattended loop as authoritative text — silently steering loops against
ratified policy until a human notices.

## Goals

**Primary Goal:** One fleet-wide rulings ledger, linked from every active charter, whose RESOLVED rows a
bounded `ailang mission doctor` check can contradict-check charters, the shared skill, and the mission
registry against — deterministically, read-only — with enforcement bound to the exact claimset bytes
Mark's attended resolution approved, so no unattended edit can activate, alter or extend enforcement.

**Success Metrics:**
- 1 new file owns fleet-wide policy; 7 charters (5 in-repo, 2 cross-repo flagged for Mark) carry the
  precedence link; 0 changes to any of the four `scripts/mission_*.sh` helpers or any
  provenance/allowlist guard; the #2082 regression class is detected by a doctor finding in a fixture
  test seeded from a snapshot of the real corpus (non-vacuous killer, explicit expected findings).
- Post-approval claim mutations (add/modify/delete/reorder) are detected as typed
  `rulings-authorization-mismatch` findings and enforce NOTHING (arms M13-M16).
- Doctor stays read-only and its zero-finding output line (`N mission(s): no drift`) is byte-unchanged,
  so `mission-lane-check.sh` consumers see no behavior change while no claims are active.
- Automatic idle wake on a ruling commit is claimed ONLY for missions whose `MISSION_WORKDIR` is this
  repository (v1 today — V30); cross-repo missions activate only via attended refresh in their own
  repos (existing directive/inbox/charter-commit inputs — V31). No automatic cross-repo wake is promised.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Contradiction checking is exact-literal claims only (2 verbs), not NLP | Decides what the fleet can mechanically promise forever; unbounded prose checking is undecidable and would over-claim | human | design | high |
| Enforcement binds to the exact canonical claimset bytes of the attended flip commit; current-row text is never trusted | Without it, an unattended claim-block edit activates enforcement the moment a row reads RESOLVED (proven: V32) — authority would rest on nothing | human | design | high |
| Ledger lives in `design_docs/fleet-rulings.md` and reuses the decision-ledger row format + `D-FR-*` IDs | Makes all three existing scripts work on it unchanged (verified live); cross-repo missions get GitHub links, the doctor gets registry-relative reads | human | design | high |
| Only RESOLVED rows' claims run; rows resolve only via the existing human channels | Preserves no-auto-resolve and the provenance contract verbatim | human | design | high |
| One seed row (D-FR-1, world-fork, two forbid-text claims on `registry`); behavior rulings and the independence question are NOT seeded | A model-name's text presence cannot prove a behavior ban; the independence question conflicts with the current operator instruction and is Mark's to rule | human | design | med |
| Charter-link lines for world and stapledon are attended follow-ups, not written by the fleet | Outside fleet authority (other repos); the doctor reports their absence instead of writing | human | design | med |
| Doctor findings use `Drift` severity (exit 1 once claims are active) | Aligns with existing drift semantics and lane-check consumers (already `dead` on today's live drift) | agent | implementation | low |
| Finding wording, internal names, fixture layout, exact cap values within stated bounds | Cosmetic; bounded by this doc's semantics | agent | implementation | low |

### Design Freeze

Each item is a SEPARATE attended act by Mark (§ Ratification Map):

- [ ] **Mark ratifies the bounded-claims narrowing**: the checker decides only the enumerated
      exact-literal claims; paraphrases and semantic conflicts stay human (§ Bounded Checker
      Semantics). Unratified ⇒ the checker half must not ship with active claims.
- [ ] **Mark ratifies the authorization-binding semantics** (§ Claim-Set Authorization Binding):
      enforcement requires the attended flip commit's provenance verdict AND byte-identical canonical
      claimsets between HEAD and that commit's blob; every divergence fails closed.
- [ ] **Mark resolves the seed row** (D-FR-1) via `scripts/mission_answer.sh --file
      design_docs/fleet-rulings.md` — this, not any code step, activates enforcement. Quiet activation
      on the unchanged base additionally requires PR #2082 to land (V27/V28).
- [ ] **Mark accepts the charter-link precedence lines** (5 in-repo charters; 2 cross-repo as attended
      follow-ups he owns).
- [ ] Quorum passes (controller-run, reject-by-default; one requorum, then an independent final review).

### Ratification Map — what a one-letter D-FLEET-25 "A" ruling approves, and what it does not

An "A" approves **building the mechanism only**: the ledger file with its OPEN seed row, the bounded
checker with the authorization binding, the tests, the doctor wiring, the changelog, and the five
in-repo charter link lines. It does NOT, by itself: (1) ratify the bounded-literal narrowing (Freeze
item 1); (2) activate any seed claim (the D-FR-1 resolution, item 3 — resolving before #2082 lands
simply reports the two live findings honestly); (3) accept the cross-repo charter links (attended edits
in world/stapledon repos, Mark-owned); (4) resolve the independence question or any other policy
conflict; (5) approve any future claim-DSL widening. Draft-build authority is not design approval, and
"A" is not approval of items 1-5.

### Independence Conflict (deliberately unresolved — Mark's, not this design's)

The V1 preference ruling ("generator≠judge — a PREFERENCE, never a blocker", gate-3-route.md:257) and the
CURRENT operator instruction for this loop (independent judge required, non-negotiable) disagree. This
design neither propagates the old preference as a new canonical seed nor weakens the current instruction:
**no independence row is seeded**, the historical chain (V1 `D-62`/`D-63` → hand-import into
world-mission.md:967 `D-WORLD-48`) is cited only as the propagation example motivating the mechanism,
and any fleet-wide independence ruling is a NEW OPEN row Mark authors and resolves.

## Solution Design

### Overview

A single file, `design_docs/fleet-rulings.md`, holds fleet-wide policy rulings in the **same marked
decision-ledger row format every charter already uses** (`<!-- decision-ledger:start/end -->`, **4
columns** — `| ID | Status | Decision / recorded answer | Evidence |`, matching
`scripts/mission_decisions.sh`'s awk fields and the charter template, V23 — `D-FR-<n>` IDs,
append-only), so the existing machinery works on it unchanged (verified live, V8/V9). Charter
decision ledgers remain the mission-local ask/provenance record; no auto-resolve, no second decision
parser: the Go side reads only a row's **ID and Status** — the authorization binding is what makes
that safe. Each RESOLVED ruling **may** carry machine-checkable claims in a separate visible fenced
block outside the ledger markers; the doctor evaluates only those claims, byte-exactly, and ONLY when
the claimset is byte-identical to the one in the human-approved flip commit (§ Claim-Set Authorization
Binding), so contradiction detection stays **bounded and honest**. Every active charter gains one
precedence line linking the ledger (§ Cross-Repo Handling for the two foreign charters).

### The Ledger File

`design_docs/fleet-rulings.md` (new, in sunholo-data/ailang):

````markdown
# Fleet-wide policy rulings

Every active charter links to this file and **cannot override a RESOLVED ruling here**; a fleet ruling
outranks charter prose and charter rows alike. Rows are append-only; IDs are never reused; rows resolve
only through the two human channels (allowlisted bookkeeping-issue directive, or an attended ruling via
`scripts/mission_answer.sh --file design_docs/fleet-rulings.md`) — never the unattended loop.
Machine-checked claims bind to the exact bytes approved at the ruling's attended resolution commit;
claims edited after approval are never enforced.

<!-- decision-ledger:start -->
| ID | Status | Decision / recorded answer | Evidence |
|---|---|---|---|
| D-FR-1 | OPEN | World is not a fork: de-forked 2026-09-06; `missions/world.toml` sets no `driver` key and `doctor.go:281` reports no fork. Stale current-tense fork claims in the registry are text defects, not facts. | PR #2082 (a58e08e68); doctor.go:281; golden_live_test.go de-fork record. |
<!-- decision-ledger:end -->

## Enforced claims (machine-checked by `ailang mission doctor`)

Only claims whose ruling ID is RESOLVED above AND whose bytes match the approved flip commit are
evaluated. Claims are exact byte literals; see the design doc for the coverage boundary, the
authorization binding, and the residual human judgments.

```fleet-rulings
# <ruling-id> <verb> "<literal>" [except "<literal>"] scope=<surface>
# verbs: forbid-text | require-text
# surfaces: charters | skills | registry | file:<repo-relative-path>
D-FR-1 forbid-text "WORLD IS A FORK" scope=registry
D-FR-1 forbid-text "never re-execs from the driver pin" scope=registry
```
````

Schema notes:
- **IDs**: `D-FR-<n>`, append-only, never reused — the `D-` prefix is required by `mission_answer.sh`'s
  guard (`case "$ID" in D-*)`) and `mission_decisions.sh`'s row regex; `FR` avoids every existing
  namespace (fleet rows are `D-FLEET-*`; docs/stapledon charters use bare `D-1`, `D-2`… — V17);
  verified unallocated (V10). The seed set is ONE row (world-fork); independence and behavior bans are
  deliberately not seeded (§ Independence Conflict; § Non-Goals).
- **Statuses**: exactly `OPEN | RESOLVED` (existing validator vocabulary).
- **Claims block**: one claim per line, `#` comments allowed, one double-quoted literal per verb; the
  block is outside the decision-ledger markers, so the awk validator ignores it (V8) — and that
  ignorance is exactly why the binding exists (V32). Visible on purpose: what Mark reads at
  resolution is what runs forever after.

### Bounded Checker Semantics

**The honest boundary:** arbitrary natural-language contradiction is not mechanically decidable. This
checker decides ONLY the claims a RESOLVED ruling explicitly lists — exact byte literals only. It
**misses paraphrases, semantic conflicts, and cross-file logic by construction**; those remain
human-review judgments (gate quorums, sprint evaluators, Mark). This narrowing is Design Freeze item 1
and must be ratified before any claim activates; the claim author chooses literals specific enough to
mean the ruling, and `except` handles legitimately quoted history.

- **Verbs**: `forbid-text "<lit>"` — the literal must not appear in any in-scope file (a hit is a
  contradiction finding); `require-text "<lit>"` — the literal must appear in every in-scope file (an
  absence is a missing-precondition finding), for genuinely-related surfaces only (e.g. a precedence
  link every charter must carry) — never across unrelated resources.
- **Optional `except "<lit>"`**: line-scoped — a hit is waived if the same line also contains the except
  literal (mechanism for historical quotation). Deterministic, visible in the claim line. **Matching**:
  byte-exact, case-sensitive substring on the whole file, one pass; no regex, no stemming, no case
  folding — casing variants are separate claim lines.
- **Surfaces and the checker's two path notions**:
  - **Registry root** (= `Mission.Root()`, parent of the registry dir, registry.go:251) owns the ledger
    path, the `skills` surface (`.claude/skills/mission-control/SKILL.md` + `resources/*.md` and the
    `.agents` mirrors — byte-identical today, V15), the `registry` surface (`missions/*.toml` beside the
    registry), and ALL `file:` scopes.
  - **Per-mission charter reads** use each mission's absolute `Workdir` + `Doc` from the registry — the
    existing `DriverPath()` pattern — and are NOT under the registry root. The checker never treats a
    foreign workdir as under root; a repointed workdir is honored as the registry states (missing clone
    → stable unreadable finding, M9).
  - **`file:<path>` containment**: the path must be repo-relative and resolve to a clean, rooted path
    strictly under the registry root. Absolute paths, any `..` component, and symlink escapes yield a
    typed `rulings-scope-escape` finding and the claim is skipped (fail-closed).
- **Not scanned (deliberate)**: Go code, `tools/launchd/mission-control.sh`, other skills, mission logs,
  `*-archive.md` files, GitHub issues, anything outside the fixed corpus. Behavior claims about code or
  routing are for behavior tests, not text matching (§ Non-Goals: the astra ban's real channel).
- **Input bounds (fail loud, never hang)**: literal ≤ 256 B; ≤ 256 claim lines; ≤ 32 distinct RESOLVED
  ruling IDs referenced by claims (each needs a provenance run); in-scope file ≤ 8 MiB (largest charter
  1,402,731 B, V16); **total corpus ≤ 64 MiB and ≤ 512 files per doctor run**; helper exec timeout
  15 s, stdout ≤ 4 KiB; blob read timeout 15 s, ≤ 8 MiB. A breach is a typed
  `rulings-malformed`/`rulings-oversize` finding; the affected claim/row is skipped and reported —
  never a silent pass, never a hang.
- **Determinism + stability**: findings collected per (claim, file), sorted by (mission, kind, path,
  literal); mission order is the registry's already-sorted order; ledger absent → one
  `rulings-ledger-missing` Drift finding, check ends; markers malformed → `rulings-malformed`, claims
  skipped; a charter unreadable → `rulings-charter-unreadable` naming the path, others still checked.

### Claim-Set Authorization Binding (the authority core)

**The hole, proven:** claims live outside the decision-ledger markers; the existing validator ignores
them (V8), and the controller's four-variant proof shows claim lines modified/added/deleted after a row
is RESOLVED all pass `mission_decisions.sh --check` unchanged (`rc 0`, `decision ledger valid: 1 rows`,
row untouched — V32) — **format-validation evidence about existing tooling, not an exploit in a runtime:
the checker does not exist yet.** Without a binding, an unattended edit could add or change claims
referencing a RESOLVED row and immediately alter enforcement: human resolution would control nothing. A
digest in the CURRENT row alone is equally insufficient: the row is mutable, and the provenance
helper never treats a RESOLVED→RESOLVED reword as a transition (mission_ledger_provenance.sh:102-104,
V33), so nothing guards a row-resident digest. **No claim is made that the existing helpers protect any
mutable digest — they do not, and this design does not ask them to.** The helpers stay byte-unchanged.

**The binding:** the human-approved immutable payload is the ledger file **blob at the ruling's attended
status-flip commit** (the commit `mission_answer.sh --commit` makes — the approval commit; the attended
channel is identity-guarded and does not push). Enforcement of ruling R's claims requires ALL of:

1. R's row at HEAD reads RESOLVED (activation precondition, unchanged);
2. provenance verdict `attended` for R, with the flip commit, from the EXISTING helper
   (`mission_ledger_provenance.sh --id R --file <ledger>`, cwd = registry root; it emits
   `<id> verdict=<v> flip=<rev|none> creation=<rev|none>`, V33);
3. the approved blob retrieved via `internal/gitexec` —
   `gitexec.CommandContext(ctx, "-C", root, "show", flip+":"+relpath)`, the exact pattern
   `internal/mission/iteration/review_packet.go:180` already uses; **no new raw git execution outside
   `internal/gitexec`** (the helper's own internal git is the helper's, unchanged); and
4. the canonical claimset for R at HEAD being **byte-identical** (SHA-256) to the canonical claimset
   for R in that approved blob.

**Canonical claimset for ruling R in blob B (exact byte encoding):** locate the single
```` ```fleet-rulings ```` fenced block (zero fences = empty set for every R; >1 fences or an unparseable
claim line for R → `rulings-approved-blob-unavailable` for R, fail-closed); take the lines that parse as
valid claim lines (comments/blanks excluded) whose ruling ID equals R, **in file order, verbatim bytes
excluding line terminators** — no trimming, no re-escaping, no reordering, duplicates preserved; payload
= ASCII `fleet-rulings/claims/v1\n` followed by each such line + `\n` (empty set = version line + `\n`);
digest = SHA-256 over those bytes. Ordering, duplicates, quoting/escaping variations, and newline style
are all part of the identity — any of them changing changes the digest.

**Verdict handling (every case typed, deterministic, fail-closed):**

| Provenance result | Finding | Enforcement of R's claims |
|---|---|---|
| `attended`, flip present, digests equal | — | **enforced** (exactly the bound set) |
| `attended`, digests differ (modified/added/deleted/reordered/re-quoted claim) | `rulings-authorization-mismatch` (Drift; names R, expected + observed digests) | **nothing enforced** — neither HEAD's set nor the approved set |
| `undetermined:squash-merge` | `rulings-provenance-undetermined` | nothing enforced |
| `fleet` | `rulings-unauthorized-resolution` | nothing enforced |
| `never-resolved` (e.g. row born RESOLVED) or `attended` with `flip=none` | `rulings-no-approval-flip` | nothing enforced |
| `instrument-failure` (helper exit 2: dirty/uncommitted ledger, shallow history, duplicate rows, renamed/ambiguous history, git unavailable) | `rulings-provenance-unavailable` | nothing enforced |
| missing/oversized/unreadable approved blob | `rulings-approved-blob-unavailable` | nothing enforced |

An uncommitted (dirty) ledger therefore reports `rulings-provenance-unavailable` — fail-closed during
development, deterministic, documented.

**Revision path:** a claim revision requires a NEW attended approval through a NEW OPEN ID
(`D-FR-<n+1>`) whose own flip binds its own claimset. The old row cannot serve: rewriting a RESOLVED row
cannot mint a new flip — the helper counts only non-RESOLVED→RESOLVED transitions, so a reopen to OPEN
first makes claims inert at HEAD, and a re-resolve lands on the LATEST flip, which the helper attributes
(V33) — and any claimset diff vs the ORIGINAL flip blob is a mismatch that enforces nothing. Rewording
the row's prose or its optional `claims-sha256:` note changes nothing enforced — enforcement reads ID,
status, and bound claim bytes only (arm M17: the current row is never trusted).

**Honest limits (stated, not papered over):** authority here is **convention-level, not cryptographic**
(the helper's own header: "Git names are conventions, not authentication") — exactly as strong as the
existing provenance convention (GitHub/web-flow committers conservatively fail closed as undetermined;
a non-fleet-named author reads as attended by convention). The helper's RESOLVED→RESOLVED reword
blind spot (102-104) is not relied on and not claimed closed — the design never stores an
authoritative digest in the mutable row; tightening that trust is a NEW design touching the frozen
helper, flagged for Mark, not disguised as solved.

### Doctor Integration (reuse, not rebuild)

- `internal/mission/doctor.go` — `Paths` gains one injected field, `FleetRulings string` (the ledger's
  absolute path). Empty disables the check — same contract as the existing `ReviewedEnvDir` field
  (doctor.go:27-28), keeping every existing `fakeFleet` test unchanged (they build `Paths{Home: root}`,
  doctor_test.go:32). `DoctorWith` runs the rulings check once per invocation after the per-mission
  loop, appending findings to the SAME `Report.Findings` (`Finding{Mission, Kind, Severity, Detail}`
  reused verbatim). With `ailang mission doctor <name>`, charter-scope claims check that mission's
  charter only; `skills` and `registry` surfaces are always checked.
- `internal/mission/rulings.go` (new) — pure, table-testable core: claim-block parser; RESOLVED-ID
  lookup between markers; canonical-claimset digest; scope→corpus mapping via `Registry`; matcher;
  findings; git reads via `internal/gitexec`; provenance helper via an injected `ProvenanceFunc`
  (implemented with `os/exec`, precedented in `internal/mission`, with the stated timeout/output caps;
  faked in table tests).
- `cmd/ailang/mission_cmd.go` — `missionDoctor` (line 236) resolves the ledger path as
  `filepath.Join(<registry repo root>, "design_docs", "fleet-rulings.md")` via `Mission.Root()`
  (registry.go:251), so a foreign `AILANG_MISSION_REGISTRY` gets the ledger beside its own registry —
  no new config knob, no home-path hardcoding.
- **No new command, no new flags.** The zero-finding summary line stays byte-identical (findings print,
  then `N mission(s): no drift` only when there are none, mission_cmd.go:255-256) — what
  `mission-lane-check.sh:49` consumes via `tail -1` and `test_mission_lane_check.sh:80` asserts.

### Source-of-Truth Precedence and Migration Rules

1. A **RESOLVED fleet ruling** (D-FR-*) outranks charter prose, charter decision rows, and skill text.
2. A charter's decision ledger remains the authoritative ask/provenance record for that mission;
   **existing charter decision ledgers stay asks/provenance — no auto-resolve, ever.**
3. Provenance is unchanged and untouched: the same two human channels, the same allowlist
   (`mission_directives.sh` refuses the authenticated account), the same attended-identity guard
   (`mission_answer.sh` refuses the fleet bot), the same Gate-0 provenance read; no resolution path
   added, no guard weakened, all script write-guards preserved verbatim.
4. **Migration INTO the fleet ledger is never automatic.** Charter rows are not moved, copied or
   re-interpreted; a ruling becomes fleet-wide only when a D-FR row is proposed and Mark resolves it.
5. **IDs are append-only, never reused** — the charter contract (gate-0-preflight.md:178-180) restated
   for the fleet namespace.

### Cross-Repo Handling (read, never write; no automatic wake)

World's charter lives in sunholo-data/ailang-world, stapledon's in sunholo-data/stapledons-godot (V22).
The doctor reads them (read-only) via each mission's absolute `Workdir` from the registry — the same way
`DriverPath()` reaches cross-repo files today — and, once claims are active, reports their missing
precedence link as a `ruling-missing-link` finding. Fixing them is an attended edit in their own repos,
owned by Mark; the fleet writes nothing outside this repository.

**Idle wake is bounded to same-repo missions (V30/V31).** `idle-fingerprint.sh` computes `head_sha`
with `git -C "$MISSION_WORKDIR" log …` — each mission's git history is its OWN workdir's repo. Only
`v1` points at this repository; the other six workdirs are separate repos, so a fleet-rulings commit
here never appears in their `head_sha` (and idle-fingerprint refuses fleet/triage outright,
idle-fingerprint.sh:20-21). Therefore:

- **Automatic wake on a ruling commit is promised ONLY for missions whose workdir is the ledger's own
  repo (v1 today); everything else gets NO automatic wake.** Cross-repo activation is an attended act
  in that repo, through inputs the fingerprint already hashes (V31): a directive comment on that
  mission's bookkeeping issue (allowlisted authors), an unread message in its mission inbox, or a
  commit in its own repo. These are Mark's existing levers — this design adds no mechanism, writes
  nothing external, and a separately gated cross-repo wake mechanism is Future Work, not promised here.

### Implementation Plan

**Phase 1: Checker core + authorization binding** (~8 hours)
- [ ] `internal/mission/rulings.go`: claim-block parser (verbs, quoting, except, scopes, bounds,
      `file:` containment), RESOLVED-ID lookup, canonical-claimset digest (version-prefixed, exact
      bytes), scope→corpus mapping via `Registry`, matcher, findings; `ProvenanceFunc` invoking the
      unchanged helper (cwd = registry root, 15 s / 4 KiB caps), the verdict table above, approved-blob
      read via `gitexec` (15 s / 8 MiB caps), fail-closed enforcement rules.
- [ ] `internal/mission/rulings_test.go`: parser/digest/binding table tests (§ Testing Strategy),
      including a real non-shallow fixture git repo with an attended flip history.

**Phase 2: Doctor + CLI wiring** (~5 hours)
- [ ] `Paths.FleetRulings` field; `DoctorWith` fleet-rulings pass; scoped-`<name>` behavior;
      `cmd/ailang/mission_cmd.go` ledger-path injection and help line; `design_docs/fleet-rulings.md`
      with the one OPEN seed row + claim block; `TestDoctor_IsReadOnly`-pattern coverage for the new
      read path (git reads included).

**Phase 3: Charter links + docs** (~5 hours)
- [ ] One precedence line in the 5 in-repo charters (`v1`, `docs`, `fleet`, `motoko`, `triage`), outside
      their decision-ledger markers so `mission_decisions.sh` and `idle-fingerprint.sh` are unaffected
      (both parse only inside the markers, V8).
- [ ] Changelog fragment; run `ailang mission doctor` and `go test ./internal/mission/...
      ./cmd/ailang/...` (touched packages) + `make test-core` AT MOST; record the #2082 non-overlap in
      the PR description.

### Files to Modify/Create

**New files:**
- `design_docs/fleet-rulings.md` (~55 LOC) — the ledger: precedence header, 1 OPEN seed row, claim block.
- `internal/mission/rulings.go` (~320 LOC) — claim parsing, canonical digest, provenance+blob binding, RESOLVED lookup, scope mapping, matching.
- `internal/mission/rulings_test.go` (~430 LOC) — table tests, binding matrix, bounds, non-vacuity killers, fixture git repo (committed, non-shallow).
- `changelogs/unreleased/<date>-fleet-rulings-ledger-doctor.md` (~20 LOC) — repo-convention fragment.

**Modified files:**
- `internal/mission/doctor.go` (+~45/-2 LOC) — `Paths.FleetRulings`; one `DoctorWith` block.
- `cmd/ailang/mission_cmd.go` (+~15/-1 LOC) — ledger-path injection in `missionDoctor`; help line.

**Draft-PR text edits, Mark-flagged (one precedence-link line each, outside ledger markers):**
`design_docs/v1-mission.md`, `design_docs/docs-mission.md`, `design_docs/fleet-mission.md`,
`design_docs/triage-mission.md`, `design_docs/motoko-mission.md`.

**Explicitly NOT touched:** `missions/world.toml` (stale text remains until #2082 lands — expected
findings on the unchanged base recorded in V27/V28), `mission-loop-change` mirrors, all four
`scripts/mission_*.sh` helpers (reused verbatim, guards preserved), `tools/launchd/lib/idle-fingerprint.sh`,
the world/stapledon charters (cross-repo; findings only). Post-sprint queue bookkeeping in
`design_docs/fleet-mission.md` is the controller's, not the sprint's.

## Examples

### Example 1: The #2082 class becomes detectable (forbid-text)

Today `missions/world.toml:3` says "WORLD IS A FORK" and only a human reading `doctor.go:281` notices.
After D-FR-1 is RESOLVED (claim bound and active), re-introducing that header in any `missions/*.toml`
yields `[DRIFT] fleet ruling-contradiction D-FR-1: missions/world.toml contradicts a RESOLVED fleet
ruling — forbidden text "WORLD IS A FORK" present`, exit 1. Expected findings on the CURRENT unchanged
base are recorded exactly (V27/V28): 2 findings, both in `missions/world.toml` — activation is NOT
promised quiet until #2082 lands.

### Example 2: The binding fires — an unattended claim edit enforces nothing

Mark activates D-FR-1 attended (`scripts/mission_answer.sh --id D-FR-1 --file
design_docs/fleet-rulings.md --commit` — attended identity, refusing the fleet bot; this commit IS the
approval commit the claimset binds to). Later, an unattended commit adds a claim (`D-FR-1 forbid-text
"gpt-6.1-sol" scope=skills`, author `sunholo-voight-kampff`). The doctor reports `[DRIFT] fleet
rulings-authorization-mismatch D-FR-1: enforced claimset does not match the attended approval (expected
sha256:9f2c…, observed sha256:41aa…); NO claims for D-FR-1 enforced` — the added claim is not enforced,
nor are the previously approved ones (fail-closed). Restoring the approved bytes, or a NEW OPEN row
resolved attended, are the only paths back to enforcement. On the next fire, the one same-repo mission
(v1) wakes via its idle fingerprint; cross-repo missions need their own attended refresh (§ Cross-Repo
Handling). (Exact-literal matching does not understand quotation — the claim author scopes
history-heavy surfaces with `file:` or `except`.)

## Success Criteria

- [ ] `ailang mission doctor` reports `ruling-contradiction` on a fixture registry seeded from a
      snapshot of the current corpus (V27/V28 literals), and reports nothing when the phrase is absent
      (non-vacuous, both arms, explicit expected findings).
- [ ] A claim whose row is OPEN produces no findings, and flipping the row to RESOLVED (attended, in
      the fixture repo's history) turns it on — the no-auto-resolve premise is asserted by a test.
- [ ] Post-approval claim mutations (add/modify/delete/reorder/duplicate/requote) each yield
      `rulings-authorization-mismatch` and enforce NOTHING (M13-M16); a row-prose/`claims-sha256:`
      reword alone changes nothing enforced (M17). Provenance verdict arms (fleet /
      squash-undetermined / never-resolved / instrument-failure) each yield their typed finding and
      enforce nothing (M19); approved-blob retrieval and its bounded failures are tested against a
      real fixture git repo (M21).
- [ ] `scripts/mission_decisions.sh --check --file design_docs/fleet-rulings.md` passes on the shipped
      ledger (V8), and `mission_answer.sh --dry-run` accepts a `D-FR-*` ID (V9).
- [ ] Touched packages pass (`go test ./internal/mission/... ./cmd/ailang/...`) plus `make test-core`;
      no other packages' behavior changes (empty `Paths.FleetRulings` skips the check — existing
      `fakeFleet` fixtures unchanged, doctor_test.go:32); `TestDoctor_IsReadOnly`-style assertion
      extended: the rulings pass (including git reads) opens no file for writing.
- [ ] Zero-findings doctor output line unchanged: `N mission(s): no drift`
      (`test_mission_lane_check.sh:80` still passes — touched-package test run only).
- [ ] Charter link lines lie outside every `decision-ledger` marker (each charter still validates under
      `scripts/mission_decisions.sh --check`); the seed row remains OPEN in the merged PR — activation
      is a later, attended, Mark-owned step.

## Testing Strategy

**Unit tests** (`internal/mission/rulings_test.go`): parser table (both verbs, quoting, except,
scopes, comments, malformed arms); canonical-digest table (ordering, duplicates, quoting/escaping,
CRLF-vs-LF, version prefix); RESOLVED-lookup arms; bounds arms (257-byte literal, 257th claim, 33rd
resolved row, >8 MiB file, >64 MiB/>512-file corpus, `file:` containment escapes); matcher arms
(case sensitivity, exact substring, sorted findings).

**Doctor integration tests**: fixture registry + ledger; per-scope findings; scoped-`<name>` run
checks only that charter plus shared surfaces; missing ledger / unreadable charter / malformed markers
arms. **Hypothetical-RESOLVED fixtures are seeded from a checked-in snapshot of the CURRENT corpus**
(the `missions/world.toml` header at pin `a3bb86291`, lines 1-5, both literals present) with explicit
expected findings recorded in the test (exactly 2, both naming `missions/world.toml`), matching the
live dry-eval (V27/V28), plus a known-positive control asserting the snapshot contains the literals.
**Binding fixture repo**: a real, committed, non-shallow git repo with an OPEN→RESOLVED flip history
authored by a non-fleet identity (the helper refuses dirty/shallow/renamed history — V33), so arms
M13-M21 exercise the actual helper, the actual gitexec blob read, and the actual digest comparison.

**Mutation matrix (non-vacuity killers; each mutation must produce the expected verdict change):**

| # | Mutation | Expected outcome | What it kills |
|---|----------|------------------|---------------|
| M1 | Delete the claim line from the ledger | no finding | "checker fires unconditionally" |
| M2 | Flip the ruling row RESOLVED→OPEN | no finding | auto-resolve/enforcement-without-ratification |
| M3 | Change the forbidden literal in the fixture file so it no longer matches | no finding | "matcher matches anything" |
| M4 | Re-seed the fixture with the #2082 phrase `WORLD IS A FORK` | `ruling-contradiction` fires, names D-FR-1, file and literal | vacuous pass — the ticket's real defect is the fixture |
| M5 | Claim references a nonexistent `D-FR-99` | `rulings-malformed`, claim skipped | typo'd ID silently disabling a check |
| M6 | Malformed verb (`forbid`), unbalanced quote | `rulings-malformed` | half-written claims passing as valid |
| M7 | Add `except "RETIRED"` and put the hit on a line containing `RETIRED` | no finding | except semantics (history quoting) |
| M8 | Ledger file removed | `rulings-ledger-missing` | vacuous absence (wrong-root greps reading as clean) |
| M9 | Fixture charter missing (workdir absent) | `rulings-charter-unreadable`, other charters still checked | crash-on-missing, one bad clone blinding the fleet |
| M10 | `require-text` claim (charter link) with/without the link line | finding absent/present | the precedence-link enforcement, both arms |
| M11 | Literal at 257 bytes / 257th claim / 33rd resolved row / 9 MiB file / corpus past file+byte totals | `rulings-malformed`/`rulings-oversize` | unbounded input hangs |
| M12 | Duplicate `D-FR-1` rows in the ledger | `rulings-malformed` (validator) / `rulings-provenance-unavailable` (helper refuses duplicates) | ambiguous activation |
| M13 | After attended resolution, MODIFY one claim line (unattended commit) | `rulings-authorization-mismatch`, zero enforcement for D-FR-1 | claim edits activating altered enforcement |
| M14 | After resolution, ADD a claim line for the resolved ruling | `rulings-authorization-mismatch`, zero enforcement (incl. the new claim) | post-approval claim injection |
| M15 | After resolution, DELETE a claim line | `rulings-authorization-mismatch`, zero enforcement | silent enforcement narrowing |
| M16 | After resolution, REORDER lines / duplicate a line / change quoting or trailing whitespace / CRLF-ify | `rulings-authorization-mismatch` | canonical-encoding drift passing as identical |
| M17 | After resolution, reword ONLY the row prose or its `claims-sha256:` evidence note | NO mismatch; bound claims still enforce | the mutable row being mistaken for the binding |
| M18 | Resolve with an empty/absent claim block, then add claims at HEAD | `rulings-authorization-mismatch` | claims minted after an approval that never covered them |
| M19 | Provenance verdicts in the fixture: fleet-authored flip / squash-style committer / born-RESOLVED row / dirty (uncommitted) ledger | `rulings-unauthorized-resolution` / `rulings-provenance-undetermined` / `rulings-no-approval-flip` / `rulings-provenance-unavailable` — all zero enforcement | unauthorized or unattributable flips activating claims |
| M20 | Digest unit arms: different version constant never matches; newline/escaping/quoting byte changes always change the digest | mismatch/verdict change per arm | sloppy canonicalization comparing across encodings |
| M21 | Approved-blob retrieval: flip blob read via gitexec returns exact approved bytes; missing blob / >8 MiB blob / read error | exact bytes enforce; `rulings-approved-blob-unavailable`, zero enforcement | unverified blob reads; hangs/crashes on bad history |

**Manual testing:** run `ailang mission doctor` on the rig before/after landing with the row OPEN
(expect findings identical to today's baseline — 6 `env-source-drift`, exit 1, V7); after Mark resolves
the seed row, expect its claims to report exactly the dry-eval findings (V27/V28) until #2082 lands.

## Deferred Decisions

- Exact finding `Detail` wording; internal function/struct names in `rulings.go` — agent may choose
  (kind names fixed by this doc).
- Test fixture layout under `internal/mission/testdata/` (including the binding fixture repo's exact
  shape) — agent may choose, provided it is a real committed non-shallow repo.
- Whether the bound constants live in `rulings.go` consts or a small table (values fixed by this doc);
  whether the scoped-`<name>` mode also re-checks that mission's charter against `file:`-scoped claims
  (documented in the test) — agent may choose.
- Whether the optional human-readable `claims-sha256:` note is recorded in the row's Evidence column at
  resolution — agent may choose; the checker never reads it either way.

## Non-Goals

- **No rebuild of #2082** — the fix is drafted and complete for its scope (V2); the seed's expected
  findings on the unchanged base are recorded instead of promised away.
- **No behavior claims enforced by text** — e.g. the `gpt-6-astra` ban (attended 2026-10-02) is a
  behavior ruling whose real channel is routing config and behavior tests (rotation table,
  gate-3-route.md:18; fallback mapping, mission-control.sh:902-923; behavior assertion,
  test_mission_routing.sh:310). No text claim seeds it; a model-name's text presence proves nothing
  about spawning.
- **No auto-propagation of rulings into charters; no general/NLP contradiction detection** —
  propagation is links + findings, prose adoption is human; bounded claims only (Freeze item 1).
- **No change to any provenance, allowlist, ratification, or directive mechanism** — reused verbatim,
  guards preserved; no claim that they protect mutable row digests.
- **No new CLI subcommand or flag; no scanning of Go code, the driver, logs, or archives** — the doctor
  gains a bounded check over the fixed corpus only.
- **No writes to external repositories, no automatic cross-repo wake, no resolution of the independence
  conflict** — cross-repo charters get findings and attended activation (§ Cross-Repo Handling); the
  independence conflict is Mark's (§ Independence Conflict).

## Timeline

**Week 1** (~18 hours): Phase 1 checker core + binding + tests (8h), Phase 2 doctor/CLI wiring +
ledger (5h), Phase 3 charter links + changelog + scoped test runs (5h) — one sprint-sized item.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Exact-literal claims false-positive on legitimately quoted history | Med | `except` literals + `file:` scope, both visible in the claim line; claim review is part of row resolution (human) |
| Activation findings on the unchanged base surprise consumers | Med | Expected findings recorded exactly (V27/V28: 2, both world.toml); #2082 landing is the stated quiet-activation precondition; consumers already treat drift as `dead` |
| The binding fails closed more than expected (squash merges, dirty worktrees) | Med | Every fail-closed case is a typed finding naming the row, not a silent skip; attended resolutions here create conventional commits (`mission_answer.sh --commit`), so the normal path is unaffected |
| Authority is convention-level (git names, not authentication) | Med | Stated honestly (§ Claim-Set Authorization Binding); exactly as strong as the existing provenance contract the fleet already relies on; tightening it is a separate design, flagged for Mark |
| Doctor exit 1 on new drift surprises an automation | Low | Consumers already treat drift as `dead` (mission-lane-check.sh:49) and today's live exit is already 1 (V7); landing ships with zero active claims |
| Ledger becomes a second opinion forum (charter rows duplicated there) | Med | Migration rule 4: no auto-migration; D-FR rows must cite their original; the fleet row is the *cross-mission* statement only |
| Scope creep toward NLP checking | High | The two-verb DSL is the freeze item; new verbs require a new design doc |
| The checker silently stops running (path regression) | Med | M8's missing-ledger finding; `TestDoctor_IsReadOnly`-pattern; touched-package test gate |

## Conflict Surface

No language surface is touched (`internal/mission`, `cmd/ailang`, docs — harness paths per the fleet
charter's Authority); the conflicts are with surfaces other tooling reads:

| Surface | Other user | Disambiguation / must-still-work |
|---------|-----------|-----------------------------------|
| Charter files (`decision-ledger` markers) | `mission_decisions.sh`, `mission_answer.sh`, `idle-fingerprint.sh`, Gate 0 | Link lines go OUTSIDE the markers; all three scripts parse only inside them (V8 live; idle-fingerprint.sh:85-93). Fixtures: `design_docs/v1-mission.md`, `docs-mission.md`, `triage-mission.md`, `motoko-mission.md`, `fleet-mission.md` must still pass `--check` after the one-line edit |
| `scripts/mission_ledger_provenance.sh` | Gate 0 (per-charter provenance) | Invoked read-only, byte-unchanged, per resolved-with-claims row; its verdict vocabulary is consumed, never widened; its 102-104 reword blind spot is not relied on |
| Doctor output contract | `mission-lane-check.sh:49` (`tail -1`), `test_mission_lane_check.sh:80` (summary line) | Findings print before the unchanged zero-findings summary; new kinds reuse `Finding.String()` format |
| Git reads (blob lookup, provenance) | repo tooling conventions | All checker git execution via `internal/gitexec` (review_packet.go precedent); helper exec is the existing script with context/size caps; no new raw git anywhere |
| Idle fingerprint semantics | driver idle-skip | Only same-repo missions (v1) see a ruling commit in `head_sha`; no change to `idle-fingerprint.sh`; charter link lines change no ledger rows |
| `.claude` vs `.agents` skill mirrors | every loop's skill load | Read-only scanning; mirrors byte-identical today (V15) |
| `missions/*.toml` | registry `Load` | Read-only; doctor never edits registry files |

**Deliberately changes:** doctor gains finding kinds (including the binding's typed findings); that is
the feature. Nothing else.

## Related Documents

**Implemented (may inform design):**
- (none topical — SimHash over 2328 docs on "fleet rulings ledger" returned no implemented doc proposing
  or containing this mechanism; nearest machinery docs are below under References)

**Planned (check for overlap):**
- `design_docs/planned/sprint-plan-gate0-ledger-provenance.md` — Gate-0 ledger-provenance validation
  machinery (related, distinct: it hardens the EXISTING per-charter ledger channel; this doc adds the
  fleet-wide layer on top of that same channel, consuming its helper as-is).
- `design_docs/planned/m-harness-mission-loop.md` — the fleet mission's ratified design (HD-1..HD-6);
  D-FLEET-25 sits under its decision-ledger policy class.
- `design_docs/fleet-mission.md` (charter) — D-FLEET-25 row `:174`, Authority `:79`, Guardrails `:99`.
- `design_docs/fleet-mission-evidence/iteration93/pick-judge.md` — the BUILD ruling scoping this draft.

Duplicate/coverage gate: recorded with its positive controls in V14 (scoped `rg` + SimHash; corroborated
by the pick-judge's 8-draft-head inspection); no planned or implemented doc designs this mechanism.

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- `design_docs/mission-charter-TEMPLATE.md` — the charter format every active mission follows; the
  decision-ledger block this design reuses verbatim (template lines 101-107)
- `.claude/skills/mission-control/resources/gate-0-preflight.md:170-196` — DECISION RECORDING CONTRACT,
  ATTENDED LEDGER EDITS, and the provenance rules this design must not perturb
- `.claude/skills/mission-control/resources/gate-3-route.md:257` — the hand-fixed independence ruling
  cited ONLY as the historical propagation example (§ Independence Conflict)
- `scripts/mission_decisions.sh`, `scripts/mission_answer.sh`, `scripts/mission_ledger_provenance.sh` —
  the reused decision/provenance helpers (byte-unchanged); `tools/launchd/lib/idle-fingerprint.sh` —
  the idle fingerprint whose wake scope § Cross-Repo Handling bounds (same-repo only)
- PR #2082 (`a58e08e68`, judge edit `399cf799b`) — the shared mechanical World-fork correction, not
  rebuilt here; landing it is the seed's quiet-activation precondition
- Quorum round-1 artifact (controller-banked; mirrored at `/tmp/fleet-i93-quorum-r1.json`) and the
  claim-binding proof (`/tmp/fleet-i93-claim-binding-proof/results.json`, V32) — the objections this
  revision answers
- Ticket: `policy:fleet-rulings-not-propagated-to-charters-and-skill` (fleet inbox, 2026-10-01), D-FLEET-25

## Quorum Revision Log (R1 → this revision)

Round 1 blocked: all three present reviewers reject; Anthropic reviewer absent (quota — reserve
`gpt6-1-sol` served). Each objection, the actual fix, its evidence, and the remaining human freeze:

| Reviewer | Objection (strongest) | Fix in this revision | Evidence | Remaining freeze |
|---|---|---|---|---|
| `gpt6-1-sol` (reserve) | Human resolution does not control claim activation: claims live outside the markers; the validator ignores them; post-resolution edits would activate altered enforcement; a current-row digest is unguarded | Added § Claim-Set Authorization Binding: enforcement requires attended flip provenance + SHA-256-identical canonical claimsets between HEAD and the approved flip-commit blob (exact byte encoding specified); all divergences fail closed with typed findings; revisions require a NEW OPEN ID; helpers unchanged, never claimed to protect row digests | V32 (4-variant proof), V33 (helper emits flip; 102-104 reword blind spot), V34 (gitexec precedent); arms M13-M21 | Mark ratifies binding semantics (Freeze item 2) |
| `oc-kimi-k3` | No V-row dry-evaluates the seed claims against their corpus; the astra seed is broken on its face (require-text over every skill file) and is a behavior claim enforced by text | Astra seed row + claim DELETED (behavior channel cited honestly: routing + behavior tests, § Non-Goals); every remaining seed claim dry-evaluated with exact commands + observations + known-positive controls; hypothetical-RESOLVED fixtures must snapshot the current corpus with explicit expected findings | V27/V28 (2 findings on unchanged base), V29 (astra in only 2 of the skill files — the old seed would have fired fleet-wide), Testing Strategy snapshot requirement | Mark resolves D-FR-1 knowing activation reports 2 findings until #2082 lands (Freeze item 3) |
| `gemini-4-argon` | `require-text "gpt-6-astra" scope=skills` self-fails on the doc's own EVERY-file semantics; "5 columns" contradicts the example table and V23 | Astra claim deleted (above); "5 columns" corrected to **4** (ID/Status/Decision/Evidence, matching the awk fields and V23) | mission_decisions.sh fields; V23; V29 | none (factual fix) |
| controller (in-session) | (a) D-FR-1 seed proposed a generator-equals-judge fallback contrary to the run operator's non-negotiable independence instruction; (b) the all-product-missions idle-wake claim is false for world/stapledon; (c) D-FR-2 scope wrong | (a) Independence row REMOVED — cited only as historical propagation example; conflict left unresolved for Mark (§ Independence Conflict); (b) wake bounded to same-repo missions (v1 only — stronger than the objection: ALL other workdirs are separate repos, V30); cross-repo activation is attended-only via existing fingerprint inputs (V31); (c) astra seed removed entirely | § Independence Conflict; § Cross-Repo Handling; V30/V31; V29 | Independence policy conflict remains Mark's, deliberately unresolved |

Seed numbering note: R1's draft seeded D-FR-1 (independence), D-FR-2 (astra), D-FR-3 (world-fork); the
first two are removed and the world-fork ruling is renumbered **D-FR-1** as the single seed. The ledger
file was never created (V13) and the `D-FR-` namespace is unallocated (V10) — no ID was ever used or
reused.

## Verification Log

Every current-code claim and every negative-existence premise below was checked first-party at pin
`a3bb86291` (branch `fleet/i93-rulings-ledger`, clean). Negative searches carry known-positive controls.

| # | Claim | Command | Observed |
|---|-------|---------|----------|
| V1 | D-FLEET-25 row and queue entry | `grep -n "D-FLEET-25" design_docs/fleet-mission.md` | `:174` full option-A row (A: ledger + doctor check; B: World-fork text only; C default), `:291` PARKED queue entry |
| V2 | #2082 scope is the shared sub-item only | `git show a58e08e68 --stat; git show 399cf799b --stat` | 8 files: `missions/world.toml`, both `mission-loop-change` mirrors, 2 test + 2 `mission_directives.sh` comments, changelog; judge edit 1 comment file. No ledger, no doctor check |
| V3 | Doctor fork report is driver-key-based | `grep -n "row.Fork" internal/mission/doctor.go` | `:281 row.Fork = m.Driver != ""` (fork only when `driver` set; World sets none) |
| V4 | Doctor is read-only by design | `grep -n "func Test" internal/mission/doctor_test.go` | `TestDoctor_IsReadOnly` at `:239`; `DoctorWith` doc comment "It is READ-ONLY: it opens no file for writing" (doctor.go:147-148) |
| V5 | Registry API to reuse | `grep -n "func " internal/mission/registry.go` | `Mission.Root() :90`, `Validate :110`, `Registry.Get :194`, `Names :204`, `LoadFile :238`, `Load :261` (sorted); `LoadFile` sets `root` = parent of registry dir (:251) |
| V6 | Doctor command surface + help line | `grep -n "doctor" cmd/ailang/mission_cmd.go` | dispatch `:49`; help line `:125` (`ailang mission doctor [<name>]`); `func missionDoctor :236` |
| V7 | Live doctor baseline | `ailang mission doctor; echo $?` | 6 `[DRIFT] … env-source-drift` findings (all missions, env-source), exit 1 — the baseline any new check must not disturb while claims are inactive |
| V8 | Existing validator accepts the new ledger format unchanged (REUSE premise, live) | scratch ledger with `\| D-FR-1 \| OPEN \| … \|` between markers + a fenced ```fleet-rulings block; `scripts/mission_decisions.sh --check --file <scratch>` then `--open` | `decision ledger valid: 1 rows`, exit 0; `--open` lists `D-FR-1 OPEN …`; the fenced claim block is ignored. Scratch deleted; worktree clean |
| V9 | Attended channel accepts `D-FR-*` | read `scripts/mission_answer.sh` | `case "$ID" in D-*)` accepts the prefix; awk row regex `^\|[[:space:]]*<id>[[:space:]]*\|`. **Live `--dry-run` blocked by sandbox** (`mktemp: mkstemp failed … Operation not permitted`, TMPDIR override also refused) — sandbox limitation reported, code-read evidence stands; `mission_ledger_provenance.sh` ID regex `^D-[A-Za-z0-9_-]+$` also accepts it |
| V10 | `D-FR-` namespace unallocated | `grep -rn "D-FR-" --include="*.md" --include="*.go" --include="*.sh" . \| grep -v fleet-mission-evidence` | empty (control: same grep for `D-FLEET-` hits the charter) |
| V11 | No Go code parses decision-ledger markers today | `grep -rn "decision-ledger" internal/ cmd/` → 0 hits | control: `grep -rln "decision-ledger" scripts/ tools/` → 5 files (roots correct) |
| V12 | Claim verbs unallocated | `grep -rn "forbid-text\|require-text" internal/ cmd/ tools/ scripts/ design_docs/planned/` | empty (control: `grep -rln "generator≠judge" .claude/skills/mission-control/` → gate-3-route.md, gate-2-pick.md) |
| V13 | Ledger file does not exist yet | `ls design_docs/fleet-rulings.md` | `No such file or directory` |
| V14 | No existing doc proposes this mechanism (duplicate gate) | `ailang docs search "fleet-wide rulings ledger charter contradiction"` (SimHash, 2328 docs); scoped `rg "rulings ledger\|fleet-rulings"` | SimHash top hits topically unrelated; rg hits only the D-FLEET-25 row + iteration-93/81 evidence. No neural score measured (scaffolder blocked, header note). Corroborated by the independent pick-judge's 8-draft-head inspection |
| V15 | Skill mirrors' current state | `cmp -s .claude/skills/mission-control/SKILL.md .agents/skills/mission-control/SKILL.md; diff -rq …/resources/ …/resources/` | identical (both 54,999 B; resources identical) — the skill's own 44,067-vs-72,254 note is stale; this doc cites the current state |
| V16 | Corpus sizes (bound basis) | `wc -c design_docs/*-mission.md <world charter> <stapledon charter>; du -sh .claude/skills/mission-control/` | world 1,402,731 B (largest), v1 930,233 B, total 3,051,771 B; skill dir 552 KB → 8 MiB per-file and 64 MiB/512-file total caps have ≥5.7x headroom |
| V17 | Charter ledger ID namespaces are not globally unique | `grep -o "^\| D-[A-Z]*-[0-9]" <charters>` | v1: `D-COV-*`, `D-ROUTE-*`; fleet: `D-FLEET-*`; world: `D-WORLD-*`; triage: `D-TRIAGE-*`; motoko: `D-MOTOKO-*`; **docs and stapledon: bare `D-1`, `D-2`** → fleet namespace needs its own prefix |
| V18 | Doctor exit-code consumer behavior | `grep -rn "mission doctor" tools/ launchd tests` | `mission-lane-check.sh:49` if/else on exit, `tail -1` of output; `test_mission_lane_check.sh:80` asserts the `config ok 1 mission(s): no drift` line |
| V19 | Idle fingerprint wake semantics (same-repo only) | read `tools/launchd/lib/idle-fingerprint.sh:54-93` | `head_sha` = `git -C "$MISSION_WORKDIR" log …` — each mission's OWN repo; exclusions cover only `design_docs/<name>-mission*`/`verification/<name>-iter*` → a commit in THIS repo appears only in missions whose workdir is this repo |
| V20 | Top-level help does not document `mission doctor` | `grep -n "doctor" cmd/ailang/help.go` | only `:326` (`ailang doctor memory`, the separate top-level doctor command) |
| V21 | Independence ruling provenance chain (historical example ONLY) | `grep -n "D-WORLD-48" <world charter>`; `grep -n "generator≠judge — a PREFERENCE" gate-3-route.md` | world-mission.md:967 (D-WORLD-48 RESOLVED, cites V1 D-62/D-63); gate-3-route.md:257 (hand-imported into the shared skill) |
| V22 | Registry's cross-repo entries | `grep -H "^name\|^repo\|^workdir\|^doc" missions/*.toml` | 7 missions; world → sunholo-data/ailang-world, stapledon → sunholo-data/stapledons-godot (charters in other repos; workdirs absolute) |
| V23 | Charter-template decision-ledger block is the canonical format (4 columns) | read `design_docs/mission-charter-TEMPLATE.md:101-107`; `scripts/mission_decisions.sh` awk fields | markers + 4-column header (`ID/Status/Decision/Evidence`) + D-1 row; template instructs keeping the markers; the validator reads exactly those 4 fields |
| V24 | Attended/provenance guards are code, not prose | read `scripts/mission_answer.sh` (identity guard, fleet-bot refusal), `gate-0-preflight.md:170-196` | guards live in the scripts and are reused untouched — no bypass, no relaxation; mission_answer.sh records its own convention-level identity limits honestly |
| V25 | Target version | `cat std/VERSION` | `v0.57.4` → next patch v0.57.5 |
| V26 | Lane-check tail-line convention must survive | read `cmd/ailang/mission_cmd.go:255-256` | findings print first; `N mission(s): no drift` printed only when findings are empty — new findings cannot displace the summary, only its trigger condition |
| V27 | Seed claim 1 dry-eval on the ACTUAL corpus (explicit expected findings) | `grep -rn "WORLD IS A FORK" missions/` | exactly 1 hit: `missions/world.toml:3`. Expected at D-FR-1 activation on the unchanged base: exactly 1 `ruling-contradiction` finding (NOT zero — quiet activation requires #2082). Control: `grep -rln "WORLD IS A FORK"` repo-wide hits world.toml + 5 other files → the root/glob is live |
| V28 | Seed claim 2 dry-eval on the ACTUAL corpus | `grep -rn "never re-execs from the driver pin" missions/` | exactly 1 hit: `missions/world.toml:5`. Same precondition story; combined expected findings on the unchanged base: exactly 2, both in `missions/world.toml` |
| V29 | Why the R1 astra seed was broken; where the ban really lives | `grep -rln "gpt-6-astra" .claude/skills/mission-control/` (and `.agents` mirror); `sed -n '18p;263p' gate-3-route.md`; `grep -n "gpt-6-astra" tools/launchd/mission-control.sh tools/launchd/test_mission_routing.sh` | literal in ONLY gate-3-route.md + gate-4-record.md (both mirrors) — NOT SKILL.md or other resources files → the old EVERY-file claim would have fired missing-precondition fleet-wide. The ban (gate-3-route.md:263, attended 2026-10-02) is behavior: rotation table (:18), fallback mapping (mission-control.sh:902-923), behavior test (test_mission_routing.sh:310) — text presence proves nothing, so the seed is deleted, not re-scoped |
| V30 | Idle-wake reality: which missions' workdirs are THIS repo | `grep -H "^name\|^workdir" missions/*.toml` | only `v1` → `/Users/voightkampff/dev/sunholo-data/ailang`; docs/fleet/motoko/triage/world/stapledon each point at SEPARATE repos → their `head_sha` never sees this repo's commits (idle-fingerprint.sh:20-21 additionally refuses fleet/triage) — automatic wake is bounded to v1 |
| V31 | Existing attended wake inputs for cross-repo missions | read `idle-fingerprint.sh:106-127` | directive comments (allowlisted gh authors) and unread mission-inbox counts are hashed into the fingerprint → an attended directive/inbox message wakes any mission — the existing, non-automatic refresh channel § Cross-Repo Handling cites; no new mechanism is built |
| V32 | Claim-block mutations pass the existing validator (the hole the binding closes) | controller proof: scratch ledger, RESOLVED row + claim lines mutated (original/modified/added/deleted) → `scripts/mission_decisions.sh --check`; `/tmp/fleet-i93-claim-binding-proof/results.json` | all 4 variants: rc 0, `decision ledger valid: 1 rows`, `row_unchanged: true` — the validator authorizes nothing about claims. **Format-validation evidence about existing tooling, not a runtime exploit — the checker does not exist yet**; this design closes the gap |
| V33 | Provenance helper's usable outputs and its reword blind spot | read `scripts/mission_ledger_provenance.sh` (emit, verdicts, :102-104) | stdout `<id> verdict=<attended\|fleet\|never-resolved\|undetermined:squash-merge\|instrument-failure> flip=<rev\|none> creation=<rev\|none>` — the flip commit IS emitted; :102-104: "Removal and resolved-to-resolved reword are never transitions" → a row-resident digest is mutable and unguarded (why the binding uses the flip blob, never the current row); the helper refuses dirty/shallow/renamed/duplicate-row history (all fail closed); "Git names are conventions, not authentication" |
| V34 | gitexec precedent for the checker's blob read | `grep -rn "gitexec" internal/mission/` | `internal/mission/iteration/review_packet.go:8,180` already runs `git -C <repo> …` via `gitexec.CommandContext` → the approved-blob lookup follows existing precedent; no new raw git outside `internal/gitexec`; exec of repo scripts is precedented in `internal/mission` |

## Future Work

- Cross-repo charter links (world, stapledon) — attended, Mark-owned, after the mechanism lands.
- A separately-gated cross-repo wake/refresh mechanism (e.g. a directive bot for ruling commits) — its
  own design with its own authority analysis; NOT promised here.
- A Gate-0 step that prints the fleet ledger's OPEN rows in every mission's `DECISIONS FOR MARK`
  section — separate small design, not this one.
- Widening the claim DSL (new verbs/scopes) — requires a new design doc per the freeze.

---

**Document created**: 2026-11-07
**Last updated**: 2026-11-07 (quorum R1 revision — same designer, one revision under the quorum protocol)
