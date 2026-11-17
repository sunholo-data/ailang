# Fleet-Wide Rulings Ledger + `mission doctor` Contradiction Report

**Status**: Planned — revision R3 (full rewrite of R2 `985aa0161`; R1 and R2 both BLOCKED by quorum, R2 judged NEEDS-HUMAN-REVIEW)
**Target**: next patch (doc unversioned in `planned/`)
**Priority**: P2 (Medium)
**Estimated**: 1.5 days at 2x (parser + check ~0.5d raw, tests ~0.25d raw)
**Dependencies**: none blocking. Sibling: draft PR #2082 (`399cf799b`, OPEN draft — V18; refreshed onto dev 2026-11-17 as PR #2396, same diff) corrects the
World-fork text in `missions/world.toml` and the doctor note. It is **already handled separately and is not
re-planned here**; it only decides whether seed ruling D-FR-1 reports 1 finding or 0 (V11).
**Planner-Lane**: codex-ok

> **The ruling this implements** — D-FLEET-25, answered by Mark (attended, 2026-11-16;
> `design_docs/fleet-mission.md:173`, V21): **"A (as recommended): a design doc (quorum, plan, judge) for one
> fleet-wide rulings ledger every charter links to and cannot override, plus an `ailang mission doctor` check
> reporting charter/skill text that contradicts a RESOLVED fleet-wide ruling; the fleet also corrects the
> World fork claim in world.toml and the doctor note."**
>
> Three nouns, three deliverables: **one ledger**, **a link in every charter** stating it cannot be
> overridden, **a doctor check that REPORTS**. Nothing in the ruling asks the doctor to enforce, gate, or
> authenticate who resolved a ruling. R1/R2 built that anyway; every quorum objection and judge blocker
> landed on it (§ Quorum Revision Log). R3 deletes it.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1 Determinism | +1 | Byte-exact substring match over a fixed, sorted file list; findings sorted. Same tree → same report. |
| A2 Replayability | 0 | No traces touched. |
| A3 Effect Legibility | 0 | Read-only file reads; no new effects, no git, no network. |
| A4 Explicit Authority | +1 | Authority is stated as what it is: reviewed PRs + git history. The doctor never resolves, activates, edits or blocks anything. |
| A5 Bounded Verification | +1 | Literal 8–200 bytes, ≤ 64 detector lines, files ≤ 4 MiB, corpus = enumerated files only. |
| A6 Safe Concurrency | 0 | None. |
| A7 Machines First | +1 | One-line detector grammar a loop can read; typed `Finding` kinds. |
| A8 Minimal Syntax | 0 | Two line forms (`forbid`, `supersedes`); no language change. |
| A9 Cost Visibility | 0 | — |
| A10 Composability | +1 | Reuses the decision-ledger table format, so `mission_decisions.sh` and `mission_answer.sh` work on it unchanged (V10); reuses `Finding`/`Note`. |
| A11 Structured Failure | +1 | Malformed ledger → one named finding listing every bad line; nothing is silently skipped. |
| A12 System Boundary | 0 | Reads this checkout plus the two registered cross-repo charters at their registered workdirs. |

**Net +6 → proceed to quorum.** Hard-violation check: no writes, no network, no new authority.

## Problem

A ruling Mark makes reaches only the layers someone edits by hand: Go, the driver, the shared skill and
its `.agents` mirror, each charter. The ticket's own case: V1 D-62/D-63 (2026-09-08, `v1-mission.md:129-130`,
V19) made generator≠judge a preference, but `gate-3-route.md` still said `generator≠judge guard (HARD`
until `349e311b9` (2026-10-01), and World parked work on the stale text. Both cited contradictions were
fixed by hand; the **class** is not. Two gaps:

1. **No single place** holds fleet-wide rulings. Each charter's decision ledger records that mission's
   asks; no text says which wins when a charter and a fleet ruling disagree.
2. **Nothing notices** when charter, skill or registry text reverts to wording a ruling retired.

## Design

### 1. The ledger: `design_docs/fleet-rulings.md`

One file, two marked blocks. The **row table is the existing decision-ledger format** (4 columns, 6
`|`-fields), so the existing tools work on it unchanged — measured, not assumed (V10):
`mission_decisions.sh --check --file design_docs/fleet-rulings.md` validates it, and
`mission_answer.sh --id D-FR-n --file design_docs/fleet-rulings.md` resolves an OPEN row.

```markdown
<!-- decision-ledger:start -->
| ID | Status | Ruling | Evidence |
|---|---|---|---|
| D-FR-1 | RESOLVED | World runs the shared driver; it is not a fork. Text calling it a fork is stale. | Transcribed from D-FLEET-25 (fleet-mission.md, Mark attended 2026-11-16). |
| D-FR-2 | RESOLVED | generator≠judge is a PREFERENCE at vendor level, never a blocker (model-level self-review stays refused). | Transcribed from V1 D-62/D-63 (v1-mission.md, Mark attended 2026-09-08); skill fixed in 349e311b9. |
<!-- decision-ledger:end -->

<!-- fleet-rulings-detectors:start -->
D-FR-1 forbid registry "WORLD IS A FORK"
D-FR-1 forbid charters "WORLD IS A FORK"
D-FR-2 forbid skills "generator≠judge guard (HARD"
<!-- fleet-rulings-detectors:end -->
```

IDs are `D-FR-<n>`. Status is OPEN or RESOLVED only (what `mission_decisions.sh` accepts, V9).
Supersession is derived, not a status (§ 4). The detector block is outside the ledger markers, so
`mission_answer.sh`'s 6-field row rewrite never sees it (V8, V10).

**Detector grammar** (one per line; blank lines and lines starting `#` ignored):

```
<ID> forbid <scope> "<literal>"
<ID> supersedes <ID>
```

- `<scope>` ∈ `registry` | `skills` | `charters` (§ 2). One scope per line; repeat the line for more.
- `<literal>`: between the first and last `"` on the line; 8–200 bytes; no `"` inside; matched
  byte-exact, case-sensitive, within one line. The 8-byte floor stops a literal that matches everything.
- There is **no `require` verb**. A require verb is what self-failed in R1 (it fires on every file of a
  scope that lacks the literal). Forbid-only also means two detectors can never contradict each other:
  absence satisfies every forbid at once.
- No `except` clause. A document that must quote a retired phrase verbatim paraphrases it, or the
  detector is narrowed by a reviewed PR. Decision-ledger blocks are already excluded (§ 2).

### 2. Scopes (the corpus is enumerated, never globbed from user input)

`root` = `Mission.Root()` of the loaded registry (the checkout holding `missions/`; `registry.go:90`, set
in `LoadFile` at `:251`, which `Load` calls at `:271` — V5).

| Scope | Files | Exclusions |
|---|---|---|
| `registry` | each loaded mission's `m.Path` (`missions/*.toml`) | none |
| `skills` | `root/.claude/skills/mission-*/**/*.md` and `root/.agents/skills/mission-*/**/*.md` (32 files today, V16) | none |
| `charters` | each loaded mission's charter: `root/m.Doc` when `m.Repo == sunholo-data/ailang` (`doctor.go:136`), else `m.Workdir/m.Doc` (world, stapledon — V15) | lines inside `<!-- decision-ledger:start/end -->` |

The ledger-block exclusion is load-bearing: charter decision rows quote the very wording a ruling retires.
Measured: `WORLD IS A FORK` appears once in `fleet-mission.md`, inside the D-FLEET-25 row, and 0 times
outside the block (V11). `ailang mission doctor <name>` restricts `registry` and `charters` to that mission;
`skills` is always the whole set.

### 3. The doctor check (report only)

New `internal/mission/rulings.go`: `CheckRulings(root string, missions []*Mission) []Finding`, called once at
the end of `DoctorWith` (`doctor.go:154`) when `root != ""`. A hand-built `Mission` has no root (the existing
doctor fixtures build them that way, `doctor_test.go:59`, V6), so existing tests are untouched.

| Kind | When | Mission column |
|---|---|---|
| `ruling-contradiction` | an active detector's literal is on a line in its scope; detail = `D-FR-n` + `file:line` + literal; ≤ 20 per detector, then one `… N more` | owning mission (charters/registry) or `rulings` for skills (not `fleet`: that is a mission name) |
| `ruling-link-missing` | a charter has no `fleet-rulings.md` outside its decision-ledger block | that mission |
| `rulings-malformed` | any detector line fails the grammar, names an unknown ID, a scope not in the table, a literal out of bounds, or a `supersedes` cycle/forward reference; or the file lacks exactly one of each block; detail lists **every** bad line number | `rulings` |
| `rulings-ledger-absent` | `root/design_docs/fleet-rulings.md` does not exist | `rulings` |
| `ruling-file-unreadable` | a scoped file cannot be read or is > 4 MiB; detail names it | owning mission |

**Severity: every ruling finding is `Note`.** `HasDrift` counts only `Drift` (`doctor.go:118`), so these
never change the exit code. Why not `Drift`: the doctor's exit code is a gate — `mission-lane-check.sh:49`
turns a non-zero doctor into a `config dead` row, and `mission-arm.sh:30-36` then refuses to arm the mission
(V3). A `Drift` ruling finding would let any detector edit stop a loop from arming: enforcement the ruling
did not ask for, resting on authority that is only convention (§ 5). As `Note`s they are seen and
blocking nothing. This also settles judge B4: an absent ledger (any registry other than this repo's) is
one note, never red.

**Output change (one line).** Today `missionDoctor` prints `N mission(s): no drift` only when there are zero
findings (`mission_cmd.go:256`). It will print that line whenever `!HasDrift()`, after any notes. Zero-finding
output is byte-identical; the lane-check `config ok` row keeps its text when only notes exist. (The
lane-check test stubs `ailang`, `test_mission_lane_check.sh:70`, so it is unaffected — V4.)

**Malformed means the whole ledger, named.** On any `rulings-malformed` condition no detector runs, and the
single finding lists every offending line. This is the answer to the R2 objection: a mangled ID cannot be
attributed, so we do not try. We fail the whole ledger loudly instead of silently ignoring a line or
quietly disabling other rulings. It is made loud before merge by a Go test that parses the real
`design_docs/fleet-rulings.md` (§ Tests T7a), so a malformed ledger fails the PR's CI, not the rig. Hits are never asserted against the live tree.

### 4. Supersession (judge B3)

`D-FR-5 supersedes D-FR-2` means: **while D-FR-5 is RESOLVED**, D-FR-2's detectors are inactive. The target
must have a lower number than the superseder (a later ruling replaces an earlier one), so cycles cannot
occur; a forward or self reference is `rulings-malformed`. A ruling is **active** iff it is RESOLVED and no
RESOLVED ruling supersedes it. OPEN rulings' detectors never run. Re-answering a resolved row is already
refused by `mission_answer.sh` ("supersede it with a new ID instead", `:136`, V8), so revising a ruling
always means a new ID plus a `supersedes` line, in one reviewed PR.

### 5. Authority — convention-level, stated honestly

- Rulings, their status and their detectors change **only by reviewed PR**; git history is the audit.
- A **new** fleet ruling arrives OPEN and is resolved by Mark through the existing attended channel
  (`mission_answer.sh --file design_docs/fleet-rulings.md`). The unattended loop may not resolve a row on
  its own behalf. That rule is the charter rule that already governs every mission ledger.
- A **transcription** row (like the two seeds) restates a ruling Mark already resolved elsewhere and must
  cite the source row in its Evidence cell. Review checks the citation.
- **This is not enforcement, and the doc does not claim it is.** `mission_answer.sh` stamps its attended
  identity as a default for any caller (`:107-108`) and records that as a "KNOWN WEAKNESS … The real control
  is the charter rule, not this code" (`:44-48`, V7). On this rig the bot and Mark's sessions commit as the
  same identity, so no git-level check can tell them apart. R3 therefore binds nothing to commit
  provenance. What makes that safe is §3: the worst a rogue detector can do is print a wrong `Note` that a
  human reads and reverts.
- The doctor never resolves, activates, edits or blocks anything.

### 6. Charters link to it ("every charter links to and cannot override")

Each charter gets one line directly under its title block:

```markdown
> **Fleet-wide rulings:** [fleet-rulings.md](fleet-rulings.md). Every RESOLVED, unsuperseded ruling there binds this mission, and nothing in this charter overrides one. If this charter disagrees with it, the ledger wins and the disagreement is a bug to report.
```

The five in-repo charters (`design_docs/{v1,fleet,docs,motoko,triage}-mission.md`) are edited in this sprint.
They are in the fleet's write scope (`design_docs/**`). The world and stapledon charters live in other repos
(V15), outside the write scope. Their link (with an absolute GitHub URL) is an **attended follow-up**; until
it lands the doctor reports `ruling-link-missing` for them as a note, which is true. The mission-control
skill's Gate 0 gets the same sentence (both mirrors). It is the shared text every loop reads, including
the cross-repo loops whose charters are not yet linked.

### 7. Seed content: no self-failure, every count measured

| Detector | Today on this base (V11–V12) | Known-positive control | Expected after |
|---|---|---|---|
| `D-FR-1 forbid registry "WORLD IS A FORK"` | **1** (`missions/world.toml:3`) | the hit itself | **0** once sibling #2082 merges; until then the note is a true report |
| `D-FR-1 forbid charters "WORLD IS A FORK"` | **0** outside ledger blocks (the 1 occurrence is inside the D-FLEET-25 row) | inside-block count = 1 in the same command | 0 |
| `D-FR-2 forbid skills "generator≠judge guard (HARD"` | **0** | the same literal = 1 in each `349e311b9^` mirror blob; sibling literal `generator≠judge — a PREFERENCE` = 2 in skills (proves the `≠` bytes match) | 0 |

**Not seeded, with measured reasons.** Model-name literals (`codex:gpt-6-astra` 4 skill / 6 charter
hits, `codex:gpt-5.6-` 16 / 78, V13) appear in legitimate "stays refused" prose. A text forbid cannot
express a behaviour ban; that belongs to the code allowlist (`AllowedOpenAIModels`). The V1/fleet
judge-independence question beyond D-62's literal retired wording is not encoded either.

## Implementation plan

| # | Step | Files |
|---|---|---|
| 1 | Parser + `CheckRulings` (+ supersession, bounds, scopes) | `internal/mission/rulings.go` (~220 lines) |
| 2 | Wire into `DoctorWith` when `root != ""` | `internal/mission/doctor.go` (~5 lines) |
| 3 | Summary line printed whenever `!HasDrift()`; help text names the ruling check | `cmd/ailang/mission_cmd.go` |
| 4 | Ledger with the two seed rows + three detectors | `design_docs/fleet-rulings.md` |
| 5 | Link line in five in-repo charters | `design_docs/{v1,fleet,docs,motoko,triage}-mission.md` |
| 6 | Gate 0 sentence, both mirrors byte-identical | `.claude/skills/mission-control/resources/gate-0-preflight.md`, `.agents/skills/mission-control/resources/gate-0-preflight.md` |
| 7 | Tests | `internal/mission/rulings_test.go`, `internal/mission/testdata/rulings/*` |
| 8 | Changelog fragment | `changelogs/unreleased/2026-11-<dd>-fleet-rulings-ledger.md` |
| 9 | Move this doc on completion | `design_docs/planned/m-fleet-rulings-ledger.md` → `design_docs/implemented/<version>/` |

### Files touched (complete; all inside the fleet write scope)

`internal/mission/rulings.go` · `internal/mission/rulings_test.go` · `internal/mission/testdata/rulings/*` · `internal/mission/doctor.go` ·
`cmd/ailang/mission_cmd.go` · `design_docs/fleet-rulings.md` · `design_docs/v1-mission.md` ·
`design_docs/fleet-mission.md` · `design_docs/docs-mission.md` · `design_docs/motoko-mission.md` ·
`design_docs/triage-mission.md` · `.claude/skills/mission-control/resources/gate-0-preflight.md` ·
`.agents/skills/mission-control/resources/gate-0-preflight.md` · `changelogs/unreleased/<fragment>.md` ·
`design_docs/planned/m-fleet-rulings-ledger.md` (moved to `design_docs/implemented/…`).
Not touched: `missions/*.toml` (sibling #2082 owns `world.toml`), every `scripts/mission_*.sh`,
`tools/launchd/**`, world/stapledon repos.

## Tests (`rulings_test.go`, fixtures built via `LoadFile` in `t.TempDir()` so `root` is set)

| # | Test | Kills the mutant |
|---|---|---|
| T1 | Active forbid literal on a skill line → exactly one `ruling-contradiction` naming `D-FR-n`, file and line | matcher always-false |
| T2 | Same literal, ruling OPEN → zero findings; ruling RESOLVED but superseded by a RESOLVED later ruling → zero; superseder OPEN → finding returns | supersession ignored / inverted |
| T3 | Literal inside a charter's decision-ledger block → zero; same line moved outside → one | exclusion missing / over-broad |
| T4 | Malformed line (bad ID, unknown scope, 7-byte literal, forward `supersedes`) → one `rulings-malformed` listing every bad line number, and **zero** contradictions even where a valid detector would hit | silent skip / partial evaluation |
| T5 | No ledger file → exactly one `rulings-ledger-absent` `Note`; `ExitCode()==0` | absent ledger as Drift (B4) |
| T6 | Every ruling finding has `Severity == Note`; a fleet with only ruling findings exits 0 | severity regressed to Drift |
| T7a | **Live ledger parses**: read `../../design_docs/fleet-rulings.md`; assert zero `rulings-malformed` and that every detector names an existing row. Only the ledger's own shape is asserted. No live-corpus count is asserted: that would make CI enforce detector hits, and would break when sibling #2082 lands | malformed ledger merged |
| T7b | **Seed quiet-on-activation**: run the seed detectors over `testdata/rulings/` snapshots: current `gate-3-route.md` → 0 hits for D-FR-2; its `349e311b9^` version → 1; a post-#2082 `world.toml` → 0; today's → 1 | seed detector that self-fails or never fires |
| T8 | Charter without `fleet-rulings.md` → `ruling-link-missing`; with it → none | link check vacuous |
| T9 | Per-mission run restricts `charters`/`registry` to that mission | scope leak |

Each test is mutation-checked once by hand (revert the guarded branch in a scratchpad copy, watch it fail).

## Success criteria

- [ ] `scripts/mission_decisions.sh --check --file design_docs/fleet-rulings.md` → `decision ledger valid: 2 rows`.
- [ ] `ailang mission doctor` on the rig: exit code unchanged from before the change (ruling findings are notes); with #2082 unmerged it prints exactly one `ruling-contradiction` (`world.toml:3`) and two `ruling-link-missing` (world, stapledon); with #2082 merged, one fewer.
- [ ] T1–T9 (incl. T7a/T7b) green; `go test ./internal/mission/` green; zero-finding doctor output byte-identical.
- [ ] Five in-repo charters carry the link line; both Gate 0 mirrors are byte-identical.

## Non-goals (removed vs R2, and why)

| R2 machinery | R3 | Why |
|---|---|---|
| Attended-provenance binding (approved-blob snapshots, SHA-256 claimset digests, `mission_ledger_provenance.sh` verdicts, 7 verdict arms) | **removed** | The ruling asks for a check "**reporting** charter/skill text that contradicts a RESOLVED fleet-wide ruling", not one that authenticates resolvers. Commit identity cannot carry that proof on this rig (V7). |
| `require-text` verb | **removed** | Self-fails across a scope (R1); forbid-only cannot conflict. |
| `Drift` severity | **replaced by `Note`** | Doctor exit gates arming (V3). |
| Claimset extraction across a shared fence with per-ruling fail-closed | **replaced** | Whole-ledger fail with a named finding plus a CI test. |
| 32-row / 512-file / 64 MiB caps, helper timeouts | **removed** | No helper subprocesses; the corpus is ~40 enumerated files. |
| Design Freeze ratification of "narrowing" and "binding semantics" | **removed** | Nothing to ratify beyond the ruling; new rulings are ordinary OPEN rows. |
| World-fork text fix | **not here** | Sibling PR #2082. |
| Writing world/stapledon charters | **not here** | Other repos; attended follow-up; the doctor reports the gap. |
| NLP / paraphrase detection, auto-fix, auto-resolve | **never** | Undecidable / outside authority. |

## Quorum Revision Log

| Objection | Surface it lands on | R3 disposition |
|---|---|---|
| R1 gemini-4-argon: `require-text "gpt-6-astra" scope=skills` self-fails; "5 columns" vs 4 | require verb; seed | **Removed** (no require verb; no model-name seed, V13). Row format is the existing 4-column table, verified by running the tools (V10). |
| R1 oc-kimi-k3: no V-row measures the seed claims; behaviour claim enforced by text | seed verification | **Fixed**: every seed detector has a measured count + known-positive control (§ 7, V11–V12), and T7b re-runs them on snapshots in CI. Behaviour/model seeds removed. |
| R1 gpt6-1-sol: after resolution an unattended edit can add claims and "activate enforcement" | provenance binding | **Removed — out of scope**: the ruling asks for "reporting". Nothing is enforced (all `Note`, V3). Detector edits are reviewed diffs, audited in git, the same convention every charter ledger runs on (V7). |
| R2 gemini-4-argon: malformed line in a shared fence → silent ignore or cross-ruling denial | parser | **Fixed**: whole-ledger fail with one named finding listing every bad line (T4), plus a live-ledger parse test (T7a). |
| R2 oc-kimi-k3: provenance helper `--file` contract never run; `Root()` citation | helper contract; registry | **Removed** (no helper consumed). The two scripts R3 does reuse were **run** on a scratch D-FR ledger (V10). `Root()` = `registry.go:90`, set at `:251`, `Load`→`LoadFile` at `:271` (V5). |
| R2 gpt6-1-sol: binding disappears when the last claim is deleted | provenance binding | **Removed — out of scope.** Deleting a detector is a visible reviewed diff; the doctor claims no binding. |
| Judge B1: "attended" root satisfiable by the unattended loop; doc overclaims | authority claim | **Rebutted by not claiming it**: § 5 states convention-level authority and cites the KNOWN WEAKNESS (V7). |
| Judge B2: cited pick-judge artifact does not exist | header provenance | **Removed**: R3 cites only the D-FLEET-25 row (V21), which was opened and read. |
| Judge B3: no supersession between rulings | ruling semantics | **Fixed**: § 4 `supersedes`, lower-number targets only (acyclic), T2. Forbid-only removes detector conflicts. |
| Judge B4: missing ledger = Drift → foreign registries permanently red | severity | **Fixed**: all ruling findings `Note`; absent ledger = one note (T5, T6). |

Judge non-blocking refinements: `review_packet.go` precedent and the `sprint-plan-gate0-ledger-provenance.md`
compatibility condition no longer apply (no git reads, no helper). The in-repo charter links sit in
`design_docs/**`, so the fleet adds them (§ 6). `except` is dropped (§ 1).

## Verification Log (worktree `fleet/i118-rulings-ledger` @ `663429dba`, 2026-11-17)

| # | Claim | Command | Observed |
|---|---|---|---|
| V1 | `Note` never fails the run | read `internal/mission/doctor.go:81-84,118-133` | `Drift`→exit 1; `HasDrift` loops for `Severity == Drift` only; `ExitCode` 1 iff `HasDrift` |
| V2 | "no drift" line printed only on zero findings | read `cmd/ailang/mission_cmd.go:236-258` | `if len(rep.Findings) == 0 { … "%d mission(s): no drift" }` (:256); `os.Exit(rep.ExitCode())` (:258) |
| V3 | Doctor exit gates arming | read `tools/launchd/mission-lane-check.sh:49`, `tools/launchd/mission-arm.sh:30-36`, `mission-lane-check.sh:30,156-157` | non-zero doctor → `row config dead`; `row` increments `DEAD`; `DEAD>0` → `NOT READY` exit 1; `rc -ne 0` without `--force` → "NOT arming" exit 1 |
| V4 | Lane-check test does not run the real doctor | read `tools/launchd/test_mission_lane_check.sh:70,80` | stub `$B/ailang` echoes `1 mission(s): no drift`; :80 asserts that string |
| V5 | `Root()` wiring | read `internal/mission/registry.go:83-90,238-256,261-282` | accessor :90; `m.root = abs` (parent of registry dir) :251; `Load` calls `LoadFile` :271 |
| V6 | Existing doctor fixtures have no root | read `internal/mission/doctor_test.go:59`; `render.go:179-187` | `&Mission{Name: …, Workdir: …}` literal; `DriverPath` comments "No registry root (a hand-built Mission in a test)" |
| V7 | Attended identity is a convention | read `scripts/mission_answer.sh:44-48,107-108` | "KNOWN WEAKNESS … ATT_NAME/ATT_EMAIL below are DEFAULTS … The real control is the charter rule, not this code"; defaults `Mark Edmondson` / noreply |
| V8 | Answer tool rewrites 6-field rows only, refuses re-answer | read `scripts/mission_answer.sh:122-140` | `n != 6` → refuse (:131); `status != "OPEN"` → "supersede it with a new ID instead" (:136) |
| V9 | Validator semantics | read `scripts/mission_decisions.sh` (all 39 lines) | one start/end block; rows `^\|[[:space:]]*D-`; status must be OPEN/RESOLVED |
| V10 | Both scripts work on a D-FR ledger with a detector block, unchanged | scratch `/tmp/fleet_i118/scratch-fleet-rulings.md` (D-FR-1 RESOLVED, D-FR-9 OPEN, one detector line); `mission_decisions.sh --check/--open --file`; `mission_answer.sh --dry-run` with a scratch identity | `decision ledger valid: 2 rows` rc 0; `--open` lists only `D-FR-9`; dry-run diff rewrites D-FR-9 → RESOLVED, detector line untouched; `--id D-FR-1` → "row D-FR-1 is RESOLVED, not OPEN — refusing"; `git status` clean |
| V11 | `WORLD IS A FORK` counts | `grep -F -c` over `missions/*.toml` / 32 skill files / 7 charters; `grep -nF` for locations; awk inside/outside ledger block on `fleet-mission.md` | registry **1** (`missions/world.toml:3`), skills **0**, charters **1** = `fleet-mission.md:173` (D-FLEET-25 row); `inside-ledger=1 outside=0` |
| V12 | `generator≠judge guard (HARD` is 0 today, and the matcher sees it | same loop; `git show 349e311b9^:{.claude,.agents}/skills/mission-control/resources/gate-3-route.md \| grep -cF` | **0** in all 3 scopes; control: **1** in each pre-fix mirror blob; `generator≠judge — a PREFERENCE` = **2** in skills (`gate-3-route.md:257`, both mirrors) |
| V13 | Model-name literals are not seedable | same loop | `codex:gpt-6-astra` skills 4 / charters 6; `codex:gpt-5.6-` skills 16 / charters 78 |
| V14 | No charter links the ledger today | same loop, literal `fleet-rulings.md` | 0 / 0 / 0 (same command returned non-zero counts for V11–V13 literals) |
| V15 | Charter locations | `grep -E '^(name\|repo\|workdir\|doc)' missions/*.toml`; `ls` both cross-repo charters; `grep -c decision-ledger:start` | 5 missions `repo = sunholo-data/ailang`; world → `sunholo-data/ailang-world`, stapledon → `sunholo-data/stapledons-godot`; both charter files exist at workdir; all 7 charters have exactly 1 ledger block |
| V16 | Skills corpus | `ls -d .claude/skills/mission-* .agents/skills/mission-*`; `find … -name '*.md' \| wc -l`; `find … -type f \| grep -v '\.md$'` | 3 dirs × 2 mirrors; **32** `.md` files; no non-md files |
| V17 | No mission declares a fork driver | `grep -rn 'driver *=' missions/`; control `grep -c 'doc *='` | no output; control 1 per file |
| V18 | Sibling #2082 state | `gh pr view 2082 --json state,isDraft,headRefOid` | `OPEN`, `isDraft: true`, head `399cf799b…` |
| V19 | D-62/D-63 are RESOLVED attended rulings; `349e311b9` applied them to the skill | `grep -n '^\| D-62 \|^\| D-63 ' design_docs/v1-mission.md`; `git show 349e311b9` | :129-130 `RESOLVED`, "RULED — attended 2026-09-08 (Mark)"; commit: "generator≠judge is a preference, never a blocker … Mark, attended 2026-10-01"; diff removes `generator≠judge guard (HARD, constraint #3)` |
| V20 | Doctor test baseline green | `go test ./internal/mission/ -run TestDoctor -count=1` | `ok … 0.328s` |
| V21 | The ruling text | `/tmp/fleet_i118/d-fleet-25-row.md`; `design_docs/fleet-mission.md:173` | `RESOLVED`; "ANSWERED — A (as recommended): …" (Mark Edmondson, attended 2026-11-16) |

**Not verified:** the live rig `ailang mission doctor` output was not run (it reads launchd state). The
success-criterion counts are predicted from V11–V15 and need a re-check at build time.
