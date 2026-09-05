# M-MISSION-LOOP-WORKBENCH: One Registry, Generated Artifacts, and a Command That Answers "Does This Reach Mission X?"

**Status**: **RATIFIED for Phase 1 — HD-1, HD-3, HD-4 and the role-config question decided by
Mark, attended 2026-09-05.** HD-1(a) one TOML per mission in `missions/`; HD-3(c) refuse on
un-registered edits; role config **passes through verbatim** and M8 stays parked; HD-4(b) decided
by Claude (refuse on a live pidfile, `--force` escape hatch). **HD-2 (de-fork ordering) is NOT
ratified and is not needed** — the ratified sprint is Phase 1 only. Quorum-blocked twice with all
six objections addressed in-doc and unreviewed; Mark ratified with that stated.
**Previously**: quorum-blocked twice, and SCOPE-CUT after review.
**HD-2(b) ratified 2026-09-06 — Phase 3 unblocked.**
**Scope cut 2026-09-05 (Mark: "is this in addition to the other mission design doc we have?"):**
the answer was five live mission-machinery docs, not two, and this one had duplicated a ratified
decision. The `[roles]` block is **removed** — role/model assignment belongs to the ratified
[M-MODEL-REGISTRY-SINGLE-SOURCE](../v0_35_0/m-model-registry-single-source.md) (V19). What
remains is the surface nothing else owns: which missions exist, where they run, on what schedule,
and whether the artifacts on disk match. The doc is smaller and better for it. Both rounds returned
3/3 reject and **every objection was correct and is now addressed in-doc** (see Quorum History).
Per the design-doc rule that the re-quorum guardrail is spent after one round, this doc is NOT
re-submitted a third time: it goes to Mark with its gaps labelled rather than grinding rounds,
which is what parks sound designs. The round-2 fixes are unreviewed by the quorum — that is the
known, stated gap.
**Target**: v0.36.0
**Priority**: P1
**Estimated**: 4 days (Phase 1: 1.5d, Phase 2: 1.5d, Phase 3: 1d)
**Dependencies**: None. Complementary to [M-MISSION-COMMS-INTO-THE-BINARY](m-mission-comms-into-the-binary.md), which extracts the *comms* seam; this extracts the *configuration and topology* seam. Neither blocks the other.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | A mission's runtime config becomes a function of one registry file. Today the same repo state yields different behaviour depending on what was hand-copied to `~/.config` and `~/Library/LaunchAgents` (V4, V5 — measured drift on 2 of 4 missions). |
| A2: Replayability | +1 | `ailang mission explain` reproduces a lane resolution from declared state, including a simulated provider drought, without firing an iteration. |
| A3: Effect Legibility | +1 | Installing a mission is currently an undeclared sequence of `cp`, `bootout`, `bootstrap` a human performs from memory. It becomes one declared, logged effect. |
| A4: Explicit Authority | 0 | The generator writes only the two paths the registry names. `install` does **not** reload: it renders, verifies and prints the reload command. Reloading — which kills a running iteration — requires `mission reload`, or `install --reload`. (Corrected after quorum round 1: the first draft claimed reload was explicit while the architecture had `install` performing it. `gemini-3-1-pro` was right; the architecture moved, not the justification.) |
| A5: Bounded Verification | +2 | The suites are bash-3.2, macOS-only, and hand-copied per repo (V6). Registry + generator logic moves to Go tests that run on every push on Linux, and the bash suites become parameterised over the registry instead of duplicated. |
| A6: Safe Concurrency | +1 | `install` refuses while the target mission holds a live pidfile — today nothing stops a `cp` over a driver that bash is mid-read of (V9). |
| A7: Machines First | +2 | The core of the doc. "Which missions exist, where do they run, on what schedule, with which lanes, and does change C reach mission M?" is answerable today only by a human reading six surfaces and one hand-maintained comment table (V1, V3). It becomes one query. |
| A8: Minimal Syntax | 0 | No language surface. One new CLI noun (`mission`) with four verbs. |
| A9: Cost Visibility | +1 | V5 is a *cost* bug: docs work that should ride the codex subscription has been silently routed to opus. Generated config makes that class visible at install time. |
| A10: Composability | +1 | The driver keeps sourcing `~/.config/ailang/mission-<name>.env` exactly as today. We stop hand-writing that file; we do not change who reads it. |
| A11: Structured Failure | +1 | `doctor` returns a typed drift report with a non-zero exit, replacing "notice the log line that says kern.boottime unreadable" (V8). |
| A12: System Boundary | +1 | launchd becomes one explicit crossing (`install`/`status`) instead of an oral procedure. |

**Net Score: +11** → **Decision: Proceed to planning.**

### Hard Violation Check

- [x] A1 (Determinism): Strictly removes nondeterminism — behaviour stops depending on undeployed hand-copies.
- [x] A3 (Effects): No hidden side effects; the point is to declare an effect performed by hand today.
- [x] A4 (Authority): No ambient access granted. The generator writes two well-known paths and nothing else.
- [x] A7 (Machines First): The doc exists to make a currently human-only question machine-answerable.

### Decision Thresholds

| Decision | Threshold | Owner |
|---|---|---|
| Registry format and location | HD-1, needs ratification | Mark |
| De-fork world now vs after the registry | HD-2, needs ratification | Mark |
| Generated env file clobbers vs merges | HD-3, needs ratification | Mark |
| Subcommand naming, TOML field names | Agent latitude | Loop |

## Problem Statement

Changing a mission loop is currently hard in a way that is not obvious from the code, and the
difficulty is not shell-vs-Go — it is that **the same facts are duplicated across six
unsynchronised surfaces, and nothing detects when they disagree.**

Measured 2026-09-05 while making one routing change plus one scheduling change (all values in
the Verification Log below):

**V1 — Six surfaces declare one mission.** Installed plist, versioned plist copy, installed env
file, versioned env file copy, a `case` arm for the boot-stagger offset inside the driver, and a
hand-maintained reach truth-table in a code comment (`mission-control.sh:752`).

**V2 — One of the four missions runs a forked driver.** `world` is a separate GitHub repo
carrying its own copy: 1029 lines against the shared driver's 1576. It has no `lib/pin-root.sh`,
so it never re-execs from the driver pin and silently missed every routing fix landed in
`sunholo-data/ailang`. Today's work had to be written twice and diffed byte-for-byte to prove
equivalence.

**V3 — "Does this change reach mission X?" has no instrument.** The answer depends on whether
the plist's `ProgramArguments` path lands in a clone that sources `pin-root.sh`, and which repo
that clone is of. It is recorded as a prose table in a comment, maintained by hand.

**V4 — The versioned config copies have already drifted from the installed ones.** `docs`
differs by 4 lines, `world` by 65. The repo copy is the reviewable artifact; the `~/.config`
copy is what the driver reads.

**V5 — That drift is a LIVE production bug, and CI is green over it.** The versioned
`mission-docs.env` widened `MISSION_PLANNER_ALLOWLIST` to admit `scripts/*` (PR #1010) and its
own comment says *"deploy the live ~/.config copy separately. This versioned profile is the
sprint's only deployment surface."* The deploy never happened. Same input, same script, today:

```
repo copy      (what CI arm 12 asserts against) -> codex:gpt-5.6-luna declared:codex-ok
installed copy (what the docs mission reads)    -> opus fail-closed:path-not-in-codex-allowlist
```

So docs work that should ride the codex subscription has been routed to opus, silently, with a
passing test. The file's comment names the exact hazard that then occurred — documentation of a
manual step is not a control.

**V6 — Tests are hand-copied between repos and two were orphaned.** `test_mission_memgate.sh`
and `test_cron_kicker.sh` existed but were absent from `make test-launchd-drivers`, so CI never
ran them — including the memory gate guarding against the 09-04/09-05 OOM events, which landed
the same day with a suite nothing executed. (Fixed in passing; the class is the point.)

**V7 — Config can be declared and never walked.** `MISSION_EVALUATOR_FALLBACK` was added
2026-08-26 under an explicit directive and was dead for ten days: the role pre-flights iterated
`PLANNER EXECUTOR` only, and the skill reads zero `MISSION_*_FALLBACK` vars. Nothing could have
told you that except reading all three loops.

**V8 — Environment differences between missions are invisible until something logs a miss.**
`sysctl` lives in `/usr/sbin`; the v1 and docs plists set a PATH omitting it, so the boot stagger
shipped inert on two of four missions and announced it only as a log line.

**V9 — Editing a live driver can corrupt the running iteration.** `world` executes its driver
straight from the working tree, and bash reads a script incrementally. This is knowable only by
having been bitten.

Adding a *new* mission is the same problem in the other direction. The template
(`tools/launchd/mission-template.plist`) documents a four-step manual procedure — `sed` a plist,
hand-write an env file, seed a state file, pick a non-colliding `StartInterval` — with no
generator, no installer, and no check that you did it right.

## Goals

**Primary goal:** Make "change a mission loop" and "add a mission loop" single declarative edits
whose reach and effect are machine-checkable before they run.

**Success metrics:**

1. A new mission (e.g. `ailang-parse`) is added by one registry entry plus two commands
   (`install`, `apply`), with no hand-edited plist and no hand-written env file. Its models come
   from `models.yml`, not from the registry entry.
2. `ailang mission doctor` exits non-zero on any drift between registry and installed artifacts —
   and, run today against `docs`, reproduces V5.
3. The reach truth-table at `mission-control.sh:752` is deleted, because `doctor` computes it.
4. Exactly one driver file exists across all missions; a test fails if a second appears.
5. The bash suites are parameterised over the registry — adding a mission adds its coverage with
   no new test file.
6. `ailang mission explain <name> --drought anthropic` prints the resolved lane per role without
   firing an iteration or spending a token.

## High-Impact Decisions

| # | Decision | Options | Recommendation | Change cost if wrong |
|---|---|---|---|---|
| **HD-1** | Registry format + home | (a) one TOML per mission in `missions/` in the ailang repo; (b) one combined `missions.toml`; (c) a file in each mission's own repo | **(a)** — per-file keeps diffs and PR review local to one mission; one directory keeps the fleet enumerable. (c) reintroduces exactly the distribution problem V2 describes. | Low. Format is internal; a migration is mechanical. |
| **HD-2** | De-fork `world` before or after the registry | (a) before; (b) after; (c) never — keep the fork and generate into both | **(b) after.** The registry makes the fork *visible and testable* first; de-forking is then a mechanical change with a test that keeps it de-forked. Doing it first repeats today's byte-for-byte hand-diffing with no net. | Medium. Reordering costs a sprint, not a rewrite. |
| **HD-3** | Generated env file: clobber or merge | (a) clobber, registry is the only source; (b) merge, preserving hand edits; (c) clobber but refuse if the target has un-registered edits | **(c).** (a) is the correct end state but would silently discard the 65 lines of live `world` config on first run; (b) preserves the drift the doc exists to kill. (c) converges: it forces one deliberate reconciliation per mission, then behaves as (a). | Low, and (c) is the reversible option. |
| **HD-4** | Should `reload` be allowed on a busy mission | (a) refuse while a pidfile is live; (b) allow with `--force`; (c) always allow | **(b).** Refusing by default addresses V9; `--force` keeps an escape hatch for a wedged loop. Liveness comes from `internal/riglock`'s existing `pidAlive`/stale-window logic (V13), not a new check. | Low. |

### Design Freeze

Sprint planning starts only when these are checked:

- [x] **HD-1(a)** — one TOML per mission in `missions/` (Mark, attended 2026-09-05)
- [x] **HD-2(b)** — de-fork world AFTER the registry, i.e. now (Mark, attended 2026-09-06: *"go with your recommendations for the loop decisions"* — delegated to the doc's recommended option, not chosen option-by-option). **Phase 3 is unblocked.**
- [x] **HD-3(c)** — refuse on un-registered edits; converges without discarding world's 65 divergent lines (Mark, attended 2026-09-05)
- [x] **HD-4(b)** — refuse on a live pidfile, `--force` escape hatch (decided by Claude, low stakes, per the deferred-decisions latitude)
- [x] **Role config passes through verbatim; M8 stays parked** (Mark, attended 2026-09-05)
- [x] Confirmed: the driver's *runtime* behaviour is out of scope (see Non-Goals)

## Solution Design

### Overview

**Generate, do not rewrite.** The driver keeps sourcing
`~/.config/ailang/mission-<name>.env` and launchd keeps reading
`~/Library/LaunchAgents/dev.ailang.mission-<name>.plist`, byte-for-byte the same contract as
today. The change is that a human stops *authoring* those two files. That keeps the blast radius
at config generation and leaves gate sequencing, probing, the stall watchdog and the pin logic
untouched — the same seam discipline
[M-MISSION-COMMS-INTO-THE-BINARY](m-mission-comms-into-the-binary.md) sets for the comms
extraction, applied to configuration.

### Architecture

```
missions/<name>.toml          ← THE source of truth (reviewed, versioned, one per mission)
        │
        ├── ailang mission install <name>          [renders to STAGING — applies nothing]
        │      ├── renders  ~/.config/ailang/mission-<name>.env.staged
        │      ├── renders  ~/Library/LaunchAgents/dev.ailang.mission-<name>.plist.staged
        │      ├── refuses if the LIVE target has un-registered edits (HD-3)
        │      └── diffs staged vs live; prints the apply command
        │
        ├── ailang mission apply <name>            [the only verb that changes behaviour]
        │      ├── refuses while the mission holds a live pidfile     (HD-4, via internal/riglock)
        │      ├── promotes both staged files by atomic rename
        │      └── bootout + bootstrap + verify, every wait BOUNDED (see below)
        │
        ├── ailang mission doctor [<name>]
        │      registry vs installed artifacts; which driver each plist resolves to;
        │      whether that clone re-execs from the pin; PATH reachability of sysctl/vm_stat.
        │      Non-zero on drift. This replaces the comment table at mission-control.sh:752.
        │
        ├── ailang mission explain <name> [--drought anthropic|codex|pi]
        │      resolved model per role, and the chain each would walk, without firing anything.
        │
        └── ailang mission list
               name, repo, workdir, schedule, driver, pin status, last slot verdict.
```

A registry entry is the six surfaces collapsed into one reviewable file:

```toml
name    = "parse"
repo    = "sunholo-data/ailang-parse"
workdir = "~/dev/sunholo-data/ailang-parse"
doc     = "design_docs/parse-mission.md"

[schedule]
mode              = "keepalive"   # or "interval"
throttle_seconds  = 10800
boot_offset       = 1680          # today: a case arm inside the driver

# NO [roles] BLOCK — DELIBERATELY. See below.
```

**⚠ The registry does NOT own model or role assignment, and an earlier draft of this doc wrongly
gave it a `[roles]` block.** [M-MODEL-REGISTRY-SINGLE-SOURCE](../v0_35_0/m-model-registry-single-source.md)
is **ratified** (D1(a)/D2(a)/D3(a), Mark 2026-08-27) and its problem statement already names
`tools/launchd/mission-control.sh`'s *"7 role/fallback exports / 3 unattended mission loops"* as
one of the four places model assignment lives. **D3(a) puts them in `models.yml`, read via
`ailang models role <role>` — described there as "the mission driver's read path".** A `[roles]`
block here would have been a competing fifth source: exactly the failure this doc is written
against, committed by this doc (V19).

So the boundary is: **that doc owns WHICH MODEL runs a role. This one owns WHICH MISSIONS EXIST,
where they run, on what schedule, and whether the artifacts on disk match.** The registry names
the mission; `models.yml` answers what it runs. Where a mission needs a genuine per-mission
override, it uses the env-override escape hatch D3(a) already preserves — it does not restate the
chain.

**V7 (declared-but-unwalked config) therefore moves too.** It stays as evidence that config can
be dead for ten days without anyone noticing, but the *fix* belongs with role assignment, not
here. What `doctor` can still check without owning the data is the cheap structural half: that
every role the driver's pre-flight loops iterate is a role `ailang models role` can answer for,
and vice versa — a mismatch between the two lists is reportable without this doc deciding either.

### Existing Machinery — reuse, extend, or build new

Added after quorum round 1. `gpt5-6-sol` blocked on the absence of this inventory, correctly:
the first draft proposed a registry, renderer, installer, doctor and CLI without establishing
what the repo already has. Measured (V11–V15):

| Need | What already exists | Decision |
|---|---|---|
| Config format + parser | `github.com/BurntSushi/toml v1.6.0` is already a direct dependency, used by `internal/pkg/manifest.go` and `internal/policy/policy.go` | **REUSE.** TOML is established repo practice, which is the substantive argument for HD-1 over inventing a format. |
| Loader shape | YAML loaders in `internal/coordinator/agent_config.go`, `internal/microrag/config.go` | **FOLLOW** their validate-on-load shape; no new pattern. |
| launchd **lifecycle** (install path, force guard, uninstall, status) | `internal/daemon/install.go` (118 LOC) — `Install(InstallOpts)`, `Uninstall`, `Status`, `plistInstallPath` | **EXTEND.** Generalise the hardcoded label to a parameter and reuse the lifecycle. |
| launchd plist **rendering** | `internal/daemon/plist.tmpl` — **cannot render a mission plist** (V17): it hardcodes `Label`, the daemon's `ProgramArguments`, `KeepAlive`, log paths and daemon-only env vars, exposing just `{{.BinaryPath}}`, `{{.Env}}`, `{{.Project}}` — no label, schedule, profile or boot offset | **NEW.** `internal/mission/render.go` stays ~250 LOC. **Round-2 correction:** the first revision claimed "the renderer and installer shrink to an extension", which `oc-glm-5-2` correctly called overclaimed — V12 established that `install.go` exists, not that its template generalises. Reuse is the lifecycle only. |
| launchd lifecycle from the CLI | `cmd/ailang/daemon.go`, `cmd/ailang/coordinator_lifecycle.go` | **FOLLOW** for `mission reload`; same bootout/bootstrap conventions. |
| Liveness / pidfile semantics | **`internal/riglock`** — `pidAlive`, `holderAlive`, stale-window and dead-holder stealing, with tests | **REUSE** for HD-4's busy check. Writing a fresh `kill -0` check beside this would be the parallel machinery the objection warns about. |
| CLI subcommand registration | `cmd/ailang/messages_send.go` and siblings | **FOLLOW.** |
| Atomic file write | **None.** No named helper exists; the temp+rename pattern is ad-hoc across **11** `os.Rename` sites (V15) | **BUILD ONE**, in `internal/mission`, and use it for both rendered artifacts. Deliberately not a repo-wide refactor — that is its own doc. |

**Net effect on the plan:** the *installer lifecycle* is an extension of
`internal/daemon/install.go`; the *renderer*, registry, doctor and `explain` are genuinely new.
The inventory's value was not that it shrank the plan — it barely did — but that it stopped a
second launchd lifecycle being written beside a working one, and it supplied `internal/riglock`
for HD-4.

### Implementation Plan

**Phase 1 — Registry + generator + `doctor` (1.5d).** Define the schema; render env and plist;
implement `doctor` with drift, driver-resolution, pin and PATH checks. Go tests throughout. Ship
`doctor` first and run it against the live fleet — it should reproduce V4, V5 and V8 before
anything is changed. A doctor that reports a clean fleet on day one is a broken doctor.

**Phase 2 — Adopt the four existing missions (1.5d).** Write `missions/{v1,docs,motoko,world}.toml`
from what is *installed* (not from the repo copies — the installed ones are what runs), reconcile
each divergence deliberately under HD-3(c), then `install` each. Deletes the reach comment table.
This phase is where V5 gets fixed on purpose rather than in passing.

**Phase 3 — De-fork world + parameterise the suites (1d).** Point world's plist at the shared
driver, settle the pinned-workdir question the dry run surfaced (the pin resolved
`workdir=~/.ailang-driver-pin/world`, an ailang worktree, while world's mission repo is
`ailang-world`), delete the fork, and add the invariant test. Rewrite the bash suites to iterate
the registry.

### Files to Modify/Create

- `missions/*.toml` — new, ~40 lines each × 4
- `internal/mission/registry.go` — new, ~200 LOC: schema, load, validate
- `internal/mission/render.go` — new, ~250 LOC: env + plist rendering
- `internal/mission/doctor.go` — new, ~300 LOC: drift, driver resolution, pin, PATH
- `cmd/ailang/mission.go` — new, ~200 LOC: `list`/`install`/`doctor`/`explain`
- `internal/mission/*_test.go` — new, ~500 LOC
- `tools/launchd/mission-control.sh` — modify, ~−60 LOC: delete the boot-offset `case` arm and the reach comment table; both move to the registry
- `tools/launchd/test_mission_routing.sh` — modify: iterate the registry instead of hardcoding `docsenv`
- `docs/docs/guides/mission-bootstrap.md` — modify: replace the 4-step manual procedure

## Examples

### Example 1: Adding the ailang-parse mission

**Today** — four manual steps from a plist comment, no verification, and the outcome depends on
whether you remembered the PATH, the boot offset, the state seed, and a non-colliding interval:

```bash
sed s/__NAME__/parse/g < tools/launchd/mission-template.plist > ~/Library/LaunchAgents/dev.ailang.mission-parse.plist
$EDITOR ~/.config/ailang/mission-parse.env          # hand-written, unversioned
echo 1234 > ~/.ailang/state/mission-parse-gh-issue
launchctl bootstrap gui/$UID ~/Library/LaunchAgents/dev.ailang.mission-parse.plist
```

**After:**

```bash
$EDITOR missions/parse.toml      # the only authored artifact; reviewed in a PR
ailang mission install parse     # renders to *.staged; changes NOTHING that runs
ailang mission doctor parse      # exit 0 = live artifacts match registry
ailang mission apply parse       # promotes staged -> live, then reloads (refuses if busy)
```

**Why staging, and not just "render then reload":** round 2 of the quorum caught that rendering
the env file *in place* applies it immediately — the driver re-sources
`~/.config/ailang/mission-<name>.env` on **every fire**, so the next interval would pick up new
config without anyone reloading anything, silently bypassing the safety step. `gemini-3-1-pro`
was right and the fix is structural, not wording: `install` never writes a path the driver
reads. Only `apply` promotes, by atomic rename, immediately before the reload it is paired with.

### Example 2: Answering "does this change reach mission X?"

**Today** — read `mission-control.sh:752`, a comment table maintained by hand, then verify it by
dry-running each mission's real plist path.

**After:**

```
$ ailang mission doctor
NAME    DRIVER            PIN    SCHEDULE              DRIFT
v1      ailang (shared)   yes    keepalive/5400        -
docs    ailang (shared)   yes    interval/21600        ** env: MISSION_PLANNER_ALLOWLIST
motoko  ailang (shared)   yes    interval/46800        -
world   ailang-world FORK no     keepalive/14400       ** driver is a fork (1029 vs 1576 lines)
exit 1
```

## Success Criteria

- [ ] `ailang mission doctor` run against the fleet *before* any migration reproduces V4, V5 and V8
- [ ] `missions/*.toml` exists for all four missions; installed artifacts are generated
- [ ] V5 fixed: the docs planner allowlist admits `scripts/*` in the file production reads
- [ ] The reach comment table at `mission-control.sh:752` is deleted
- [ ] Exactly one `mission-control.sh` across all mission repos, with a test asserting it
- [ ] Bash suites iterate the registry; adding a mission adds coverage with no new test file
- [ ] A declared-but-unwalked role chain is a `doctor` failure (V7 cannot recur silently)
- [ ] `mission-bootstrap.md` documents the two-command path
- [ ] All tests passing; documentation updated

## Testing Strategy

Go tests for schema, rendering and drift detection — these run on Linux on every push, which the
bash-3.2 macOS-only suites do not (A5). Golden-file tests render each of the four current
missions and assert byte-equality with what is *installed today*, so Phase 2 is provably
behaviour-preserving except where a divergence is reconciled on purpose.

**The doctor needs a non-vacuous check of its own.** A drift detector that reports a clean fleet
is indistinguishable from one that is not looking. Its test fixtures include the three real
divergences measured here (V4/V5/V8), and it must fail on each.

## Deferred Decisions

Agent latitude — do not escalate: exact TOML field names; subcommand help text; whether `list`
renders a table or JSON by default; where the golden files live; whether `doctor` checks state
seeds in the same pass.

## Non-Goals

- **Porting the driver's runtime to Go.** Gate sequencing, model probing, the stall watchdog and
  the pin logic stay in shell. Same boundary the comms doc sets.
- **Changing scheduling or routing policy.** The registry describes what is already true. Any
  cadence change is a separate, measured decision — the 09-02 measurement (three loops tightened
  together, fleet stall rate 6% → 33%, no extra starts) stands.
- **Owning role or model assignment.** Ratified to `models.yml` via `ailang models role` (D3(a), Mark 2026-08-27). The registry names missions; it does not name their models.
- **Replacing launchd.**
- **Managing missions on hosts other than the rig.**

## Timeline

| Day | Work |
|---|---|
| 1 | Phase 1: schema, loader, renderer, Go tests |
| 1.5 | Phase 1: `doctor`; run against the live fleet and confirm it reproduces V4/V5/V8 |
| 2.5 | Phase 2: author the four registry entries from installed state; reconcile divergences |
| 3 | Phase 2: `install` each mission; delete the comment table |
| 4 | Phase 3: de-fork world; parameterise the bash suites; invariant test |

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Generating over live config breaks a running fleet | HD-3(c): refuse on un-registered edits. Golden tests assert byte-equality with what is installed today, so the first `install` is a no-op by construction. |
| World's pinned-workdir question is unresolved | It is Phase 3, explicitly scoped, and surfaced by a dry run already recorded here. If it proves deep, Phases 1–2 still deliver — de-forking is the only part that depends on it. |
| The registry becomes a seventh surface rather than a replacement | Success criterion: the comment table is *deleted* and the boot-offset `case` arm *moves*. Adding without removing is the failure mode. |
| Doctor is vacuous | Its fixtures are the three real divergences; it must fail on each before it is trusted. |

## Related Documents

- [M-MISSION-COMMS-INTO-THE-BINARY](m-mission-comms-into-the-binary.md) — the comms seam. Complementary in *purpose*: it leaves "the rest of the driver" alone and establishes the extraction pattern; this applies that pattern to configuration and topology. **The first draft claimed "no file overlap". That was false and unverified — `oc-glm-5-2` blocked on it, correctly.** The measured overlap and its ordering are in the Conflict Surface below.
- [M-MODEL-REGISTRY-SINGLE-SOURCE](../v0_35_0/m-model-registry-single-source.md) — **RATIFIED, and it owns role/model assignment.** Found only after this doc's first draft: the create-script's neural search scored it 0.41, below the 0.45 review threshold, because the two docs share almost no vocabulary while sharing a surface. The `[roles]` block was cut in response (V19). **Ordering CORRECTED 2026-09-05, and it is not what an earlier draft said.** That draft claimed
"the model registry lands first". False in practice: M1–M7 have **landed** (`internal/modelreg`
exists, `ailang models role executor` returns a working chain), but **M8 — mission driver
adoption, the only milestone that would move role config — is PARKED** by Mark 2026-08-27 ("if it
ain't broken won't fix"). The park note is right on the merits: the driver's routing is *richer*
than the registry's, and a naive wiring would have silently downgraded the fable designer to opus
on three live loops.

So this doc cannot defer to a parked path. **Ratified 2026-09-05: the generated env file passes
role/model lines through VERBATIM.** The registry owns schedule and topology; role config is
copied, not authored, and carries a marker naming M8 as its future owner. Nothing here blocks on
the model registry, and nothing here claims ownership of roles.
- [M-DRIVER-PIN-ROLLOUT](../m-driver-pin-rollout.md) — PARKED at the quorum gate after two blocked rounds. Adjacent to Phase 3, but **not overlapping**: its V18 audit explicitly scopes world OUT — *"`mission-world`: … explicitly not ours (world fork — Non-goals)"*. Phase 3 fills precisely the gap that doc declines. Worth reading before Phase 3 anyway: it is the prior art on pin semantics, and it is parked on a design choice a controller may not invent.
- [M-MISSION-ELO-ROUTING](../m-mission-elo-routing.md) — PROPOSED. Consumes role assignment to choose lanes by rating; owns no config surface. No overlap.
- [M-MISSION-SLOT-HEARTBEAT](../v1_0_0/m-mission-slot-heartbeat.md) — slot death attribution. Consumes the slot-verdict log this doc's world port started writing. No overlap.
- [M-MISSION-PORTABILITY](../../implemented/v0_30_0/m-mission-portability.md) — shipped `MISSION_PROFILE` and the env-file convention this builds on. It made a second mission *possible*; it did not make a fifth *easy*.
- [mission-bootstrap guide](../../../docs/docs/guides/mission-bootstrap.md) — the current 330-line manual procedure this replaces.

## Conflict Surface

Added after quorum round 1. Two of three reviewers blocked on its absence.

**1. `tools/launchd/mission-control.sh` — both docs modify it.** Measured, and the regions are
disjoint:

| Doc | Regions touched | Nature |
|---|---|---|
| Comms | 6 `gh issue comment` sites at lines **126, 886, 899, 1206, 1224, 1241** (−120/+40) | replaces call sites |
| This doc | `_mc_boot_offset` case arm (~line 290) and the reach comment table (~line 752) (−60) | deletes two blocks |

No shared line region, but git merges by hunk context, not intent, so **concurrent PRs on one
1576-line file will still conflict on context.** Declared ordering: **comms lands first.** It is
the larger edit, it is already written, and this doc's deletions rebase onto it trivially,
whereas the reverse shifts every line number in the comms doc's enumerated call-site list — the
premise its V7 row rests on. If this doc must land first, the comms doc re-derives those six line
numbers before planning.

**2. The `ailang mission` CLI noun is shared.** Comms adds `cmd/ailang/mission_comms.go` for
`ailang mission report`; this adds `cmd/ailang/mission.go` for `list`/`install`/`reload`/
`doctor`/`explain`. Whichever lands first **creates the `mission` command group**; the second
registers into it. Same ordering as above. The verbs do not collide.

**3. `internal/mission/` package tree is shared.** Comms creates `internal/mission/comms/`; this
creates `internal/mission/` top-level. Nested, not colliding — but the second to land must not
assume an empty tree.

**0. `apply` must not be able to hang (A5).** `bootout` is asynchronous and `bootstrap` can
block on a busy domain, so every wait carries a deadline, mirroring the driver's own standing
rule that every wait is bounded (`_mc_bounded`):

| Step | Bound | On expiry |
|---|---|---|
| `bootout` settle | poll `launchctl print` at 250ms, deadline **10s** | typed `ErrBootoutTimeout`; live files already promoted, so report state and exit non-zero — never retry blindly |
| `bootstrap` | deadline **10s** | typed `ErrBootstrapTimeout`; the staged copies are retained for a re-`apply` |
| readiness verify | poll for `state = running`\|`waiting`, deadline **15s** | typed `ErrVerifyTimeout`, exit non-zero with the last `launchctl print` |

Total worst case is bounded at 35s. `apply` is never a background operation and never retries on
its own; a wedged domain is reported, not worked around.

**4. Extending `internal/daemon/install.go` touches a shipped path.** Generalising the hardcoded
`com.sunholo.ailang.daemon` label to a parameter changes code the coordinator daemon depends on.
Existing programs that MUST still work: `ailang daemon install`, `daemon uninstall`,
`daemon status`, and `make coord-install` / `coord-status` / `coord-uninstall`. Mitigation: keep
the existing exported signatures as thin wrappers that pass the daemon's label;
`internal/daemon/install_test.go` is the regression gate and must stay green unmodified.

**5. Generating over live config is the sharpest surface.** The rendered env file is read by a
running fleet. HD-3(c) plus golden tests asserting byte-equality with what is installed *today*
make the first `install` a no-op by construction; without both, this section would be the reason
to reject the doc.

**6. Role/model assignment is NOT this doc's surface.** `m-model-registry-single-source` is
ratified and owns it. If both land, the generated env file must contain no `MISSION_*_MODEL` or
`MISSION_*_FALLBACK` value that `models.yml` also declares. Ordering: model registry first.

**What deliberately changes:** the docs planner allowlist (V5) starts routing `scripts/*` to
codex instead of opus. That is the bug fix, and it is an intentional behaviour change on a live
mission, not a regression.

## Quorum History

Two rounds, `gpt5-6-sol` + `gemini-3-1-pro` + `oc-glm-5-2`, reject-by-default, $0.17 total.
Six blocking objections, all accepted, none argued. Recorded because the *pattern* is the
evidence for this doc's premise: the failures were all "a claim I did not verify".

| Round | Reviewer | Objection | Resolution |
|---|---|---|---|
| 1 | gpt5-6-sol | No inventory of existing machinery; parallel machinery unjustified | Added Existing Machinery (V11–V15). **Found `internal/daemon/install.go`, a working launchd installer the draft would have duplicated.** |
| 1 | gemini-3-1-pro | A4 justification said reload was explicit; architecture had `install` reloading | Split into separate verbs |
| 1 | oc-glm-5-2 | "No file overlap" with the comms doc was false | Measured it (V14, V16); added Conflict Surface with declared ordering |
| 2 | gemini-3-1-pro | **"install renders only" was still false** — the driver re-sources the env file every fire, so writing it in place applies immediately, bypassing the safety step | Structural fix: `install` writes only `*.staged`; `apply` promotes by atomic rename. `install` now writes no path the driver reads |
| 2 | gpt5-6-sol | `apply` had no bounded waits (A5) | Deadlines + typed timeouts, Conflict Surface §0; worst case 35s |
| 2 | oc-glm-5-2 | "Renderer shrinks to an extension" overclaimed — V12 proved `install.go` exists, not that its template generalises | Verified (V17): the template cannot render a mission plist. Claim narrowed to lifecycle-only reuse; renderer stays new |

**The objection that mattered most** is round 2's gemini: two rounds of "make reload explicit"
were cosmetic while the env file was still written in place. Wording changes cannot fix an
apply-semantics bug — a reviewer reading the architecture against the driver's actual behaviour
found it, and no amount of author-side re-reading had.

## Verification Log

Every claim below was measured on 2026-09-05 on the rig, not inferred.

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | Six surfaces declare one mission | Enumerated and counted each | 4 installed plists, 4 repo plist copies, 4 repo env files, 4 installed env files, 4 boot-offset case arms, 1 comment table |
| V2 | World runs a forked driver | `wc -l` both; `grep -c pin-root` | 1029 vs 1576 lines; world sources no `pin-root.sh` |
| V3 | Reach is a hand-maintained comment | Read `mission-control.sh:752` | Prose table, v1 ✅ motoko ✅ docs ➖ world ❌ |
| V4 | Repo config copies have drifted from installed | `cmp` each pair | docs differs 4 lines, world 65; v1 and motoko identical |
| V5 | That drift changes live routing | Ran `derive-planner-lane.sh` on one fixture against both env files | repo → `codex:gpt-5.6-luna declared:codex-ok`; installed → `opus fail-closed:path-not-in-codex-allowlist` |
| V6 | Two suites existed but CI never ran them | Cross-checked `tools/launchd/test_*.sh` against `make/test.mk` | `test_mission_memgate.sh`, `test_cron_kicker.sh` orphaned (since wired) |
| V7 | A declared fallback need not be walked | Read both pre-flight loops; grepped the skill | Loops iterated `PLANNER EXECUTOR`; skill matches 0 `MISSION_*_FALLBACK` |
| V8 | v1/docs plists omit `/usr/sbin`, disabling the stagger | `PlistBuddy` each PATH; `env -i` repro | `sysctl` unreachable → `kern.boottime unreadable`; motoko/world set no PATH key and inherit it |
| V9 | World executes its driver from the working tree | `ProgramArguments` + live `ps` during an iteration | Confirmed; edits require atomic replace |
| V10 | `StartInterval` re-arms from exit, not start | 3 consecutive v1 gaps + 1 world gap from slot-verdict logs | 5403/5372/5379s vs 5400s; 14390s vs 14400s |
| V11 | TOML is already established repo practice | `grep BurntSushi/toml go.mod`; grep users | `v1.6.0` direct dependency; used by `internal/pkg/manifest.go`, `internal/policy/policy.go` |
| V12 | A launchd plist installer already exists in Go | Read `internal/daemon/install.go` | 118 LOC: `//go:embed plist.tmpl`, `Install`/`Uninstall`/`Status`/`plistInstallPath`, force guard, with `install_test.go` |
| V13 | Process-liveness semantics already exist | Read `internal/riglock` | `pidAlive`, `holderAlive`, stale window, dead-holder stealing, tested |
| V14 | Both docs modify `mission-control.sh` | Read the comms doc's Files section and line-cited V7 | Comms −120/+40 at lines 126/886/899/1206/1224/1241; this doc −60 at ~290 and ~752. Disjoint regions, same file |
| V15 | **No** named atomic-write helper exists | `grep -rn 'os.Rename'`; `grep 'func .*Atomic\|func .*SafeWrite'` | 11 ad-hoc `os.Rename` sites, zero named helpers. Negative-existence claim, logged per the design-doc rule |
| V16 | Both docs add an `ailang mission` subcommand | Read both Files sections | Comms `cmd/ailang/mission_comms.go`; this `cmd/ailang/mission.go`. Shared command group, disjoint verbs |
| V17 | The daemon plist template **cannot** render a mission plist | Read `internal/daemon/plist.tmpl` in full | Hardcodes `Label`, daemon `ProgramArguments`, `KeepAlive`, log paths, daemon-only env. Exposes only `{{.BinaryPath}}`, `{{.Env}}`, `{{.Project}}` — no label, schedule, profile or boot offset |
| V18 | The driver re-sources the env file on **every** fire, so an in-place write applies without a reload | Read `mission-control.sh:63-64` and `:73-74` | Sourced TWICE per fire — once by `MISSION_PROFILE` (:63-64) and again by `MISSION_NAME` (:73-74). An in-place write is live on the next fire with no reload. This is why `install` must stage |

| V19 | A **ratified** doc already owns mission role/model assignment | Read `m-model-registry-single-source.md` — status line, problem table row 2, D3(a) | "Ratified … D1(a)/D2(a)/D3(a) decided by Mark 2026-08-27"; table names `mission-control.sh` "7 role/fallback exports"; D3(a) makes `ailang models role` "the mission driver's read path". The `[roles]` block was cut |
| V20 | The pin-rollout doc does **not** cover world | Read its V18 audit row | "`mission-world`: … explicitly not ours (world fork — Non-goals)" — Phase 3 fills that gap rather than duplicating it |

## References

- `tools/launchd/mission-control.sh` — shared driver (1576 lines)
- `tools/launchd/mission-template.plist` — the 4-step manual procedure this replaces
- `~/.ailang/state/mission-<name>-slot-verdicts.log` — durable per-slot duration record

## Future Work

- **Render passthrough from the REVIEWED copy, not the installed one.** Phase 2 renders
  passthrough content out of `~/.config/ailang/mission-<name>.env` (what runs) and
  promotes the result, which leaves `tools/launchd/mission-env/*.env` trailing until it
  is synced by hand — the very drift class V5 belongs to, merely narrowed. Reading
  passthrough from the reviewed copy instead makes reviewed-vs-installed drift
  impossible after an apply and restores the repo copy's meaning as the thing you edit.
  ~10 lines in `RenderStaged`; deliberately NOT done at the end of a live adoption.

- Fold the `~/.ailang/state` seeds (gh-issue, designer-rotation pointer) into the registry so a
  mission's full identity is one file.
- A `mission new <name>` scaffold that writes the registry entry, the charter skeleton and the
  bookkeeping issue in one step.
- Once the registry exists, cadence experiments become declarative and reversible — which is the
  precondition for re-testing the 09-02 tightening safely.
