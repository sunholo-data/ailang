# M-MISSION-HEARTBEAT-DRIVER-ROOT

**Status:** Planned mechanical correction; fleet iteration 12, 2026-10-02. **Revision 4** (D-FLEET-8 scope: both mirrors + rc-19 + guard allowlist; quorum round 4 exit-status objection closed by characterizing both measured invocation shapes of the `:?` expansion failure).
**Priority:** P0 queue ticket `skill:heartbeat-relative-path-absent-in-world`.
**Target:** next fleet deployment. **Effort:** one short milestone.
**Planner-Lane:** codex-ok

Revision history: Revision 1 (fleet iteration 7, 2026-09-29) covered the seven authoritative
`.claude` resources only and deferred the `.agents` mirror to a follow-up. Quorum rounds 1 and 2
(2026-09-29) blocked it; the human ruling **D-FLEET-8 (2026-10-01, Mark attended, "go with all
your recommendations")** resolved the scope question: *"YES: the next heartbeat design fixes BOTH
the `.claude/skills/mission-control/resources/**` and `.agents/skills/mission-control/resources/**`
call sites and removes the divergence; the gate-3-route.md rc-19 line is in scope for both
copies."* Revision 2 removes the deferral, adds the rc-19 line, and adds the pre-push guard arm the
mirror half needs.

## Problem and scope

Eighteen heartbeat commands in fourteen mission-control gate resources — nine in the seven
authoritative `.claude/skills/mission-control/resources/` gate files and nine in the byte-mirrored
`.agents/skills/mission-control/resources/` copies — resolve their helper against CWD. Both checked
World and Stapledon repositories contain a README but lack that relative helper. The existing
driver already exports `MISSION_DRIVER_ROOT` from its own resolved tree.

Two further mechanical gaps fall inside the same D-FLEET-8 scope:

1. **rc-19 line.** `scripts/mission_pi_run.sh` landed verdict `provider_quota` rc 19 in PR #1424
   (commit 94524a6fc, 2026-10-01). The skill's rc list at
   `.claude/skills/mission-control/resources/gate-3-route.md:386` (and the identical `.agents`
   copy) still reads "rc 0=ok · 10=empty_worktree · 11=reasoning_stall · 12=stream_dead ·
   13=wall_timeout · 14=launch_failed · 18=tool_hang" and does not name 19. A route step that
   reads this list cannot classify a quota refusal.
2. **Scope-guard seam.** The fleet pre-push guard `tools/launchd/githooks/pre-push`
   `_scope_is_harness` (line 17) allows `.claude/skills/mission-*` and `.claude/skills/sprint-*`
   but no `.agents` paths, so the mirror half of this fix would be refused at push. The fix's
   mirror edits are ordinary mission-harness mirror maintenance — the guard's own comment says
   "Keep in step with design_docs/fleet-mission.md 'Authority'" — so the arm and the Authority
   allowlist row change together.

This is a mechanical correction, not a new architecture. A separate full design is not warranted;
this concise artifact records the requested scope and verification for the mandated
designer/planner/executor/evaluator handoff. Ruling D-FLEET-2 (2026-09-26) allows walking a role's
declared fallback chain when the pinned model is dead; the rc-19 line makes quota refusals — the
trigger of exactly such walks — legible to the route step.

## Decision

Replace each `bash tools/launchd/mission-heartbeat.sh stamp ...` with:

```bash
bash "${MISSION_DRIVER_ROOT:?MISSION_DRIVER_ROOT must name the pinned driver tree}/tools/launchd/mission-heartbeat.sh" stamp ...
```

Keep labels and arguments identical, including `abort <reason>` and `complete`. Apply the same
nine substitutions to the authoritative `.claude` resources and to the `.agents` mirror: edit the
`.claude` sources, then run `tools/launchd/sync-agents-skills.sh` (which rsyncs
`.claude/skills/<skill>/` verbatim over `.agents/skills/<skill>/` for the six mission-loop skills);
`tools/launchd/test_agents_skills_sync.sh` enforces the copies stay identical. The two
gate-3-route.md copies are byte-identical today (cmp silent, re-measured 2026-10-02) and must
remain identical after the fix. The mirror edits are the same mechanical correction applied to the
copy pi and codex actually read — they remove the divergence D-FLEET-8 named, not a new feature.

The shell parameter check makes a missing/empty root an explicit error; a wrong root fails through
bash's file-open error. Do not fall back to CWD or `AILANG_DRIVER_SRC`; the existing driver comment
identifies that source clone as potentially behind the executing tree. Rely on the verified driver
contract to export an absolute root; the parameter guard enforces nonempty, not absoluteness.
Attended invocations must explicitly set it to the intended checkout using
`export MISSION_DRIVER_ROOT="$(cd /absolute/path/to/driver-checkout && pwd)"`. The unattended driver
exports this variable unconditionally for every scheduled mission, including those whose CWD is the
driver tree. An attended invocation with no export now fails loudly even in that formerly-working
CWD; that is an intentional contract change. Add a short attended setup sentence to each of the
fourteen modified resources so this behavior is actionable. A caller-supplied relative root
violates this documented contract; these snippets do not validate arbitrary caller configuration.

For the expansion-failure exit status: it is **invocation-shape dependent on the same shell**. Measured 2026-10-02 on the rig's only bash, `/bin/bash` 3.2.57: a `bash -c` command string — `env -u MISSION_DRIVER_ROOT /bin/bash -c 'bash "${MISSION_DRIVER_ROOT:?msg}" --version'` — prints `bash: MISSION_DRIVER_ROOT: msg` and exits **rc=127** (same for an empty value); a script *file* containing the identical `bash "${MISSION_DRIVER_ROOT:?msg}" --version` line, run as `/bin/bash <script>` with the variable unset, exits **rc=1** (likewise `: ${X:?msg}` in a script file). Round-2 quorum objected that bash expansion errors exit with 1; round 4 objected that the doc's round-2/3 measurement of 127 was a fabricated or flawed basis for dismissing that claim. Both objections are now resolved by measurement: rc=1 is true of the script-file shape and false of the `-c` shape; rc=127 is true of the `-c` shape and the earlier "could not be reproduced on any shell installed here" framing over-generalized it — the script-file shape reproduces rc=1 on this very machine. This shape dependence is precisely why the acceptance contract asserts **nonzero** failure and pins no exit code: the design's contract is "explicit nonzero failure before the helper runs", never a specific code.

For the rc-19 line: add `19=provider_quota` to the rc list in BOTH gate-3-route.md copies (append
to the `18=tool_hang` comment line, keeping the three-line comment shape), keeping the copies
identical. The existing sentence "Anything non-zero except 18 is a LANE FAILURE, not a result:
fall back and FLAG, never re-prompt in place" — verified in the Verification Log row "rc-list
catch-all sentence present" rather than asserted — then covers rc 19 without further edit: a
quota refusal is a lane failure, and falling back follows the role's declared chain per D-FLEET-2.

For the guard seam: add `.agents/skills/mission-*` and `.agents/skills/sprint-*` arms to
`_scope_is_harness` in `tools/launchd/githooks/pre-push` (line 17), and add
`.agents/skills/mission-*/**` and `.agents/skills/sprint-*/**` to the "May change" allowlist in
`design_docs/fleet-mission.md` Authority, per that section's own "Keep in step" contract with the
guard. This is a mission-harness-path change inside the fleet's existing write scope: the guard
still refuses product loops on every other `.agents/**` path, and nothing outside the six
mission-loop skills gains a `.agents` arm. No new authority is granted to product loops.

No new helper, export, state format, label, or heartbeat semantics is proposed.

## Files to modify

Heartbeat call sites — authoritative `.claude` sources (nine commands):
- `.claude/skills/mission-control/resources/gate-0-preflight.md` — two commands.
- `.claude/skills/mission-control/resources/gate-1-observe.md` — one command.
- `.claude/skills/mission-control/resources/gate-2-pick.md` — one command.
- `.claude/skills/mission-control/resources/gate-3-route.md` — one command.
- `.claude/skills/mission-control/resources/gate-3b-ci-green.md` — one command.
- `.claude/skills/mission-control/resources/gate-4-record.md` — one command.
- `.claude/skills/mission-control/resources/gate-5-retro.md` — two commands.

Heartbeat call sites — `.agents` mirror copies (nine commands; produced by running
`tools/launchd/sync-agents-skills.sh` after the `.claude` edits, not hand-edited):
- `.agents/skills/mission-control/resources/gate-0-preflight.md` — two commands.
- `.agents/skills/mission-control/resources/gate-1-observe.md` — one command.
- `.agents/skills/mission-control/resources/gate-2-pick.md` — one command.
- `.agents/skills/mission-control/resources/gate-3-route.md` — one command.
- `.agents/skills/mission-control/resources/gate-3b-ci-green.md` — one command.
- `.agents/skills/mission-control/resources/gate-4-record.md` — one command.
- `.agents/skills/mission-control/resources/gate-5-retro.md` — two commands.

rc-19 line (same two gate-3-route.md files as above):
- `.claude/skills/mission-control/resources/gate-3-route.md:386-388` and the identical
  `.agents/skills/mission-control/resources/gate-3-route.md:386-388` — add `19=provider_quota`.

Scope-guard seam:
- `tools/launchd/githooks/pre-push` — `_scope_is_harness` line 17: add `.agents/skills/mission-*`
  and `.agents/skills/sprint-*` arms.
- `design_docs/fleet-mission.md` — Authority "May change" allowlist: add the two matching
  `.agents/skills/*` entries.

## Acceptance and implementation plan

One milestone: apply the nine `.claude` substitutions, run
`tools/launchd/sync-agents-skills.sh`, add the attended setup sentence to each resource, add
`19=provider_quota` to the rc list in the `.claude` gate-3-route.md, re-sync, and verify the
complete call-site inventory across BOTH mirrors.

- **Call-site inventory.** Exact-literal `rg -F 'bash tools/launchd/mission-heartbeat.sh stamp'`
  over both resource trees returns 9 + 9 = 18 sites before the fix (re-measured 2026-10-02) and 0
  across the fourteen tracked gate files after it; the guarded form appears exactly 9 + 9 times.
  Historical retro references outside the fourteen files remain out of operational scope and are
  not claimed.
- **Mirror equality.** `tools/launchd/sync-agents-skills.sh --check` passes;
  `cmp` of the two gate-3-route.md copies is silent; `make -f make/test.mk` launchd suite row
  `test_agents_skills_sync.sh` (make/test.mk:71) passes, including its negative drift control.
- **rc-19 line.** Both gate-3-route.md copies name `19=provider_quota` in the rc list at line 386;
  `cmp` of the two copies is silent. Positive control: `rg -n '19=provider_quota' scripts/mission_pi_run.sh`
  still matches line 407 (`VERDICT_NAME="provider_quota"; RC=19`) and the line-71 comment.
- **Guard.** The revised `_scope_is_harness` accepts `.agents/skills/mission-control/resources/gate-0-preflight.md`
  (positive control) and still refuses a product path under `.agents/` such as
  `.agents/skills/model-manager/SKILL.md` (negative control); the Authority allowlist in
  `design_docs/fleet-mission.md` lists the same two `.agents` arms.
- **Drills.** Extract/run the revised commands with a temporary `AILANG_STATE_DIR` from World and
  Stapledon CWDs, checking each label, mission namespace, attempt, and abort note. Test missing and
  empty `MISSION_DRIVER_ROOT` fail **nonzero** without writing a heartbeat (measured rc=127 on rig
  bash 3.2.57; do not pin the code in the test).
- **Spaces-root drill.** Run the guarded command with a `MISSION_DRIVER_ROOT` whose path contains
  a space (a space-named directory holding `tools/launchd/mission-heartbeat.sh`), a temporary
  `AILANG_STATE_DIR`, and a drill `MISSION_NAME`: assert rc=0 and exactly one row in
  `mission-<name>-heartbeat` with the drilled label and attempt 1. Measured this revision (row
  below); the executor re-runs it as an acceptance step.
- **Scheduled-context extraction.** Run a scheduled-context driver-tree CWD test, extracting the
  actual root derivation and export from the driver and supplying relative
  `$0=tools/launchd/mission-control.sh`; assert the resulting root is absolute and the heartbeat
  command succeeds. Also test missing and empty exports from that same formerly-working CWD and
  assert nonzero failure without adding a row. This extraction tests the production derivation
  without launching a second live mission.
- **Suites.** Run the existing `tools/launchd/test_mission_heartbeat.sh` suite and the launchd
  suite row `test_agents_skills_sync.sh`. Independent evaluator must inspect the diff (fourteen
  gate files, two gate-3-route.md rc lines inside them, the guard, fleet-mission.md) and reject
  any broadened production scope. No live mission state is used for drills. Fleet controller owns
  approval, quorum policy, landing, and deployment reporting.

## Verification log (2026-10-02, this tree)

| Claim | Command / observed output |
|---|---|
| Nine `.claude` call sites, seven files | `rg -n -F 'bash tools/launchd/mission-heartbeat.sh stamp' .claude/skills/mission-control/resources` → exactly 9 matches: gate-0:3, gate-0:4, gate-2:89, gate-5:3, gate-5:5, gate-3:3, gate-3b:3, gate-4:61, gate-1:3. |
| Nine `.agents` mirror call sites, seven files | Same command over `.agents/skills/mission-control/resources` → exactly 9 matches at the identical lines (gate-0:3, gate-0:4, gate-2:89, gate-4:61, gate-3b:3, gate-5:3, gate-5:5, gate-3:3, gate-1:3). 18 sites, 14 files total. |
| rc list at :386 lacks 19, both copies identical | `rg -n '10=empty_worktree' .claude/... .agents/...` → line 386 in both: `# rc 0=ok · 10=empty_worktree · 11=reasoning_stall · 12=stream_dead` then `13=wall_timeout · 14=launch_failed · 18=tool_hang` — no `19`. `cmp` of the two files: silent (byte-identical). |
| rc 19 landed in mission_pi_run.sh | `rg -n 'provider_quota' scripts/mission_pi_run.sh` → `71:# 19 provider_quota — pi finished, but its LAST assistant message is a provider refusal…` and `407: elif [ "$PROVIDER_QUOTA" = true ]; then VERDICT_NAME="provider_quota"; RC=19`. `git log --oneline -3 scripts/mission_pi_run.sh` → `94524a6fc fix(mission): pi runner types a provider quota refusal as provider_quota rc 19 (#1424)`. |
| rc-list catch-all sentence present | The rc-list comment carries the catch-all sentence "Anything non-zero except 18 is a LANE FAILURE, not a result: fall back and FLAG, never re-prompt in place." (gate-3-route.md:387-388, both copies identical): `grep -n "Anything non-zero except 18" .claude/skills/mission-control/resources/gate-3-route.md .agents/skills/mission-control/resources/gate-3-route.md` → both files line 387: `#    13=wall_timeout · 14=launch_failed · 18=tool_hang.  Anything non-zero except 18`, rc=0; `sed -n '386,389p'` confirms the continuation on line 388. |
| Guard allowlist has no `.agents` arms (negative, with positive control) | `rg -n '\.agents' tools/launchd/githooks/pre-push` → no match (rc=1); control `rg -n '\.claude/skills/mission-' tools/launchd/githooks/pre-push` → `17: .claude/skills/mission-*|.claude/skills/sprint-*|.pi/extensions/*) return 0 ;;`. The file's own comment at lines 12-13: "Keep in step with design_docs/fleet-mission.md 'Authority'". |
| Authority allowlist in fleet-mission.md lacks `.agents` | `sed -n '73,83p' design_docs/fleet-mission.md` → "May change" lists `.claude/skills/mission-*/**`, `.claude/skills/sprint-*/**` — no `.agents` entry. |
| Sync test exists and is wired in | `ls tools/launchd/test_agents_skills_sync.sh` → present; `sed -n '68,74p' make/test.mk` → `@$(LAUNCHD_SUITE) tools/launchd/test_agents_skills_sync.sh` at make/test.mk:71. The test checks `.agents` copies equal `.claude` for the six mission-loop skills AND that a drifted copy is detected (negative control in the test itself). |
| `:?` expansion failure exit status (rounds 2 and 4) | Invocation-shape dependent on the same shell — `/bin/bash` 3.2.57, the only bash on this machine. **`-c` shape:** `env -u MISSION_DRIVER_ROOT /bin/bash -c 'bash "${MISSION_DRIVER_ROOT:?msg}" --version'` → stderr `bash: MISSION_DRIVER_ROOT: msg`, **rc=127**; same rc=127 with `MISSION_DRIVER_ROOT=` (empty). **Script-file shape:** a script containing `#!/bin/bash` then the identical `bash "${MISSION_DRIVER_ROOT:?msg}" --version` line, run as `/bin/bash <script>` with the variable unset → stderr `<script>: line 2: MISSION_DRIVER_ROOT: msg`, **rc=1**; same rc=1 for `: ${X:?msg}` and `true "${X:?msg}"` in script files. Both shapes re-measured 2026-10-02 in this tree. Inner probe (`set +e; bash "${MISSION_DRIVER_ROOT:?msg}" ...; echo`) never reaches the echo in either shape — bash exits on the expansion error. So the round-2 reviewer's exit-1 claim is true of the script-file shape and false of the `-c` shape, and the doc's earlier "unreproducible here" framing was itself an over-generalization of the `-c` measurement; the acceptance contract therefore asserts **nonzero** and pins no code. |
| Spaces-root drill (round-2 objection 4b) | Built a space-containing root `<worktree>/sp ace drill <pid>/tools/launchd/mission-heartbeat.sh`, `AILANG_STATE_DIR=<tmp>`, `MISSION_NAME=iter12-drill`; ran the guarded command with `stamp gate-3 "spaces drill"` → rc=0; `cat $STATE/mission-iter12-drill-heartbeat` → `1790905392\t2026-10-02T01:43:12Z\tgate-3\t1\tspaces drill`; namespace `mission-iter12-drill-heartbeat` verified. This session's sandbox refuses `/tmp` and `$HOME` writes, so the drill root lived inside the worktree and was removed afterward; the design's contract is space-safe quoting of `$MISSION_DRIVER_ROOT`, which this drill exercises. |
| Root is existing executing-driver contract | Read `tools/launchd/mission-control.sh:47` (exact derivation quoted below) and 2270–2285 (unconditional export for every mission, with World/Stapledon as motivating examples). |
| Existing skill uses contract | `rg -n MISSION_DRIVER_ROOT .claude/skills/mission-control/resources` returns `gate-3-route.md:72` and `role-spawn-routing.md:85`, invoking mission-lane-dead.sh. |
| Relative helper absent in both checked repositories | Python `Path.is_file()` positive controls: `/Users/voightkampff/dev/sunholo-data/{ailang-world,stapledons-godot}/README.md` both true; `tools/launchd/mission-heartbeat.sh` existence both false (revision-1 measurement, re-listed for context). |
| Proposed command works from each actual CWD | Revision-1 measurement (2026-09-29): guarded command with root `/Users/voightkampff/.ailang-driver-pin/fleet`, temporary state dirs, `stamp gate-0`; rc=0, namespaced row with `gate-0`, attempt `1`. |
| Existing state behavior | Read full `tools/launchd/mission-heartbeat.sh`: state root is `AILANG_STATE_DIR` or HOME default; file is `mission-${MISSION_NAME}-heartbeat`; label whitelist `fired|gate-0…gate-5|gate-3b|complete|abort`; attempt/note preserved; unset `MISSION_NAME` writes nothing and exits 0. |

No AILANG language claims are made. Parser/typechecker conflict surface is inapplicable;
shared operational surface is all missions reading these fourteen resources. Runtime rollout must
respect the existing running-skill/pinned-tree deployment rules; saving a pin alone is not proof
of fleet deployment.

## Related documents and duplicate gate

Scaffold script's neural search returned top planned match
[`m-mission-slot-heartbeat`](v1_0_0/m-mission-slot-heartbeat.md) at 0.42 and top implemented match
[`m-spawn-pin-enforcement-sprint-plan`](../implemented/v0_35_0/m-spawn-pin-enforcement-sprint-plan.md)
at 0.38; neither reaches the duplicate threshold. The heartbeat design owns the original
instrument; this artifact corrects its call sites. Read also
[`m-mission-portability`](../implemented/v0_30_0/m-mission-portability.md), which established
shared cross-repository mission profiles, and
[`m-spawn-pin-enforcement`](../implemented/v0_35_0/m-spawn-pin-enforcement.md), the existing
resolved driver contract context. [`fleet-mission.md`](../fleet-mission.md) owns the Authority
allowlist this revision extends and records the D-FLEET-8 ruling. SimHash results were unrelated
sprint docs and not treated as neural similarity evidence.

## Authority and axioms

Human rulings anchoring this revision: **D-FLEET-8** (2026-10-01) puts both mirrors and the rc-19
line in scope; **D-FLEET-2** (2026-09-26) governs the fallback behavior the rc-19 line documents.
Agent-resolvable choices remain: reuse the existing root export with explicit missing-root failure;
apply the mirror via the existing sync mechanism rather than a new one; extend the guard's
existing mission-harness arms rather than redefining its scope. The guard change is inside the
fleet's existing write scope (`tools/launchd/**`, `design_docs/fleet-mission*`) and adds
`.agents` mission-harness mirror paths only — the guard still refuses product loops on every
other `.agents/**` path. Unattended quorum requirement remains for the controller to satisfy or
explicitly apply the mechanical-fix exemption from mission routing.

Axiom scores: A1 +1 (same helper selected regardless of CWD, in both mirrors); A2 0; A3 0; A4 0;
A5 +1 (bounded isolated drills); A6 0; A7 +1 (machine-detectable missing root); A8 0; A9 0;
A10 0; A11 +1 (explicit root failure); A12 +1 (guard and Authority allowlist change in step, per
the guard's own comment). Net +5; no hard violations.

## Revision 1 evidence and conflict surface (retained where still true)

Read `sed -n '38,55p' tools/launchd/mission-control.sh` and
`sed -n '2270,2285p' tools/launchd/mission-control.sh`; operative lines and the
load-bearing source-clone comment are verbatim:

```bash
MC_DRIVER_ROOT=$(cd "$(dirname "$0")/../.." 2>/dev/null && pwd)
REPO="${MISSION_WORKDIR:-$MC_DRIVER_ROOT}"
cd "$REPO" || exit 1
# The pinned driver's own tree, for skill steps that call driver tools from a mission
# repo that has none (world, stapledon). NOT AILANG_DRIVER_SRC: that is the source clone,
# which can be far behind what actually runs.
MISSION_DRIVER_ROOT="${MC_DRIVER_ROOT:-}"; export MISSION_DRIVER_ROOT
```

The export is top-level, outside a mission-name branch. `pwd` produces an absolute path after
`cd` even when `$0` is relative; the derivation precedes the mission work-directory change.

| Revision check | Fresh result |
|---|---|
| Driver-tree positive control and relative invocation | Revision-1 probe: extracted the two assignment lines above, ran them with `bash -c ... tools/launchd/mission-control.sh` from the fleet checkout, then the proposed heartbeat command. rc=0; root printed `/Users/voightkampff/.ailang-driver-pin/fleet`; isolated row `1790678203 2026-09-29T10:36:43Z gate-0 7`. This is an extracted-code probe, not a claim to have launched the full live driver. |
| Attended missing and empty export in driver CWD | Revision 1 recorded rc=127 with `MISSION_DRIVER_ROOT must name the pinned driver tree`; the heartbeat row count stayed at the preceding successful call's single row. Re-measured 2026-10-02 in a clean env: rc=127 on `/bin/bash` 3.2.57 (Verification Log above). The doc now asserts **nonzero**, not a specific code, because the value is shell-version dependent. |
| Repository-wide inventory | `rg --hidden -n 'mission-heartbeat\.sh' --glob '!.git/**' --glob '!tools/launchd/test_*' --glob '!design_docs/**' --glob '!.ailang/**'` found authoritative `.claude` calls, `.agents` mirror calls, helper usage text, and historical retro references. `git ls-files .agents/skills/mission-control` confirmed the mirror resources are tracked files. Revision 2 fixes all 18 calls in the 14 tracked gate files; the remaining references are helper usage text and historical retros, outside operational scope. |

Preserved working path: scheduled driver-tree mission, verified above. Repaired paths: scheduled
World and Stapledon, plus the mirror copies pi and codex actually read. Intentional
incompatibility: attended calls without an explicit absolute root fail before invoking the helper,
with setup instructions added at each edited gate. Arbitrary externally supplied relative roots
remain outside the contract; production derives its own absolute value. Existing helper labels,
notes, state namespaces and unscheduled MISSION_NAME behavior stay owned by the unchanged helper.