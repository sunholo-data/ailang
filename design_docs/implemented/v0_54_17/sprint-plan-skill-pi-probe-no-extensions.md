# Sprint plan: mission-control skill pi extension discovery

**Sprint:** M-FLEET-SKILL-PI-PROBE-NO-EXTENSIONS · fleet iteration 50 · 2026-10-20  
**Ticket:** `probe:pi-duplicate-extension-discovery` (triage iterations 6 and 8, 2026-10-19).  
**Scope:** one small harness fix, one milestone; planning artifacts only in this stage. Controller commits artifacts; no planner git writes or implementation dispatch.  
**Estimate:** 120 changed/added LOC including mirrors: 16 per resource, 84 test, 4 changelog. One executor session, approximately 60–90 minutes; reserve one working day for controller verification. Estimates are task-based, not measured LOC/day throughput.

## Goal and reuse

Make prescribed skill launches load each extension once, matching existing driver behavior. Iterations 6 and 8 lost the exact no-tools evaluator probe to duplicate registration of `ailang_run`, `ailang_cli`, `ailang_check`, `builtins_search`, `freshness_report`, `examples_search`, `quota_report`, and `session_protocol_ack`; iteration 8's isolated `-ne/-ns/-nc/-np` probe returned rc 0.

Registry reuse for M1_SKILL_PI_PROBE_NO_EXTENSIONS: **none**. This is Bash/documentation wiring, with no package-like AILANG capability or `.ail` module; registry search and package inspection are inapplicable. Reuse `_mc_pi_ext_args`, existing Wiring arms, existing LAUNCHD_SUITE integration, and the skill mirror check. No new test file or package. Controller's cross-skill search (M4) supplies the systemic audit.

## Fixed design decisions D1–D6

**D1 — probe, both copies.** Prescribe `pi --mode json --no-session --no-tools --no-extensions --model "$MODEL" -p 'reply with exactly: ok'`. Add at most two explanatory sentences: this matches `_mc_probe_pi` in `tools/launchd/lib/lane-probe.sh`; without it a checkout carrying `.pi/extensions` discovers both the global `~/.pi/agent/extensions` copy installed by `ailang pi install` and the repo copy, exits rc 1 with `Tool ... conflicts`, and reports a working model as dead (ticket `probe:pi-duplicate-extension-discovery`).

**D2 — raw sandbox snippet, both copies.** Add this initialization, following `scripts/mission_pi_run.sh:288` exactly with `$WT` as the helper argument:

```bash
source "$REPO/tools/launchd/lib/pi-ext-args.sh"
PI_EXT_ARGS=(); while IFS= read -r _l; do PI_EXT_ARGS+=("$_l"); done < <(_mc_pi_ext_args "$WT")
cd "$WT" && PI_FENCE_ROOT="$WT" pi --mode json --no-session ${PI_EXT_ARGS[@]+"${PI_EXT_ARGS[@]}"} \
  -e "$REPO/tools/pi-extensions/sandbox/index.ts" \
  -e "$REPO/tools/pi-extensions/worktree-fence.ts" \
  --model "$MODEL" -p "$PROMPT" \
  > /tmp/pi_run_iter<N>.ndjson 2> /tmp/pi_run_iter<N>.stderr
```

Keep the existing `mkdir -p /tmp/claude`, surrounding guards, and “do not drop the two `-e` flags” instruction. Add one sentence that `scripts/mission_pi_run.sh` already supplies these arguments and remains the preferred route.

**D3 — regressions in existing Wiring section.** Preserve the original 8 arms. Add two arms per copy: probe line has `--no-tools --no-extensions`; raw fenced snippet has the source, `_mc_pi_ext_args "$WT"`, array initialization, and the literal `${PI_EXT_ARGS[@]+"${PI_EXT_ARGS[@]}"}` splice before the two sandbox flags. Mere mention of `PI_EXT_ARGS` elsewhere must not satisfy the raw arm. Add one sweep safety arm and one count/instrument arm, for **14 passing arms** at the two default roots.

Use `PI_EXT_ARGS_SKILL_ROOTS` as a newline-delimited directory override for both the new per-root arms and sweep, defaulting to the repository's `.claude/skills/mission-control` and `.agents/skills/mission-control` absolute directories. Quote paths and read roots with `IFS= read -r`; use Bash 3.2, POSIX awk and BSD grep (`grep -E` for alternatives, never `grep -P` or BRE `\|`).

Simplest sweep predicate: scan each root's `SKILL.md` and `resources/*.md`; enumerate every line containing literal `pi --mode json`. A line is safe iff it contains `--no-extensions`, or belongs to the same fenced Bash block containing the literal array splice. A small awk block buffer can retain launch line numbers until the closing fence, validate each line, and clear context on every fence/file boundary; inspect unfenced lines immediately and flush at EOF. Do not authorize a whole file or fixed number of neighboring lines from a helper mention. Print every unsafe `file:line: text`; unreadable/missing roots or files fail explicitly.

Count launch lines independently of safety **per root** and print the count. Require at least 2 per root; fewer is `INSTRUMENT FAILURE: <root>: <n> launch lines (minimum 2)`, with nonzero exit. This known-positive control prevents an empty scan from passing. Fixtures use complete copied skill directories except the deliberate empty-root control.

**D4 — mirrors.** Both resource files remain byte-identical; run the existing mirror gate. Executor must attempt the `.agents` write. If refused (as in four prior iterations), report the exact refusal and pending equality criterion; controller mirrors only this resource byte-for-byte and reruns the gates. Do not bypass permissions or claim a green mirror gate before it passes.

**D5 — changelog.** Create `changelogs/unreleased/2026-10-20-skill-pi-probe-no-extensions.md`, 2–5 lines total, opening exactly `### Fixed — mission-control skill pi probe and raw pi invocation load each extension once`.

**D6 — excluded.** No changes to `.pi/extensions/**`, `tools/pi-extensions/**`, `~/.pi`, `tools/launchd/lib/*.sh`, `scripts/mission_pi_run.sh`, `tools/launchd/mission-control.sh`, `make/test.mk`, Go code, or SKILL.md content (unless the sweep finds a launch there; none today). No live provider probe is needed for this documentation/wiring fix.

## M1_SKILL_PI_PROBE_NO_EXTENSIONS

Dependencies: none; driver behavior is already shipped (M3). One milestone is intentional; do not add a dummy to satisfy validator policy.

| File to change | Responsibility |
| --- | --- |
| `.claude/skills/mission-control/resources/gate-3-route.md` | D1 probe, D2 raw snippet and concise explanation. |
| `.agents/skills/mission-control/resources/gate-3-route.md` | Byte-identical mirror; attempt write and report refusal if denied. |
| `tools/launchd/test_pi_ext_args.sh` | D3 per-copy arms, scoped sweep, count control and fixture override in Wiring section. |
| `changelogs/unreleased/2026-10-20-skill-pi-probe-no-extensions.md` | D5 fragment. |

Day 1 sequence: add focused arms and observe failure on the stale snippets; edit source resource, attempt mirror, add fragment; run positive gates and all disposable mutations below; record evidence and let controller finish denied mirrors and host integration. No `.ail` examples or language tests apply; the corrected snippets are the executable documentation examples.

### Arm → mutation evidence

Use `mktemp -d` and copied skill roots; select them with `PI_EXT_ARGS_SKILL_ROOTS`. Mutate only disposable files, clean up temporary fixtures, never edit checked-in files for mutations, never use git checkout. After each mutation, assert nonzero rc and the specific intended arm/diagnostic (unrelated red arms do not establish a kill).

| Arm | Disposable mutation/control that must turn it red |
| --- | --- |
| `.claude` probe | Delete only `--no-extensions` from that copy's probe line; its probe arm fails. |
| `.agents` probe | Same deletion in the other copied root; that root's probe arm fails. |
| `.claude` raw snippet | Delete only the `${PI_EXT_ARGS...}` launch splice, keeping initialization/helper mention; its raw arm fails. |
| `.agents` raw snippet | Same splice deletion in the other copied root; that root's raw arm fails. |
| Sweep safety | Append synthetic `pi --mode json --no-session --model x` outside any fence to a disposable resource copy; point override at copied roots; sweep fails naming that exact file:line. |
| Sweep instrument | Point override at an empty temporary directory; fail explicitly as instrument failure with zero launch lines, never PASS. |

Also run the unmodified copied-root fixture through the override first (rc 0), so mutation failures demonstrate detection rather than a broken fixture. Record all six mutation/control outcomes with exit status and diagnostic in sprint JSON notes.

## Verification

Established **controller measurements**, made outside any sandbox at `origin/dev` **4c0ce261f**; cite these as baseline evidence, not planner/executor reruns:

| ID | Measurement |
| --- | --- |
| M1 | From checkout CWD, `pi --mode json --no-session --no-tools --model "ollama/kimi-k3:cloud" -p 'reply with exactly: ok'` returns rc 1. Stderr names `/Users/voightkampff/.pi/agent/extensions/ailang-exec.ts` conflicting with `/Users/voightkampff/.ailang-driver-pin/fleet/.pi/extensions/ailang-exec.ts` for `ailang_run`, plus other listed tools. |
| M2 | Identical command from `/tmp/fleet_i50_pi`, which has no `.pi/extensions`, returns rc 0; only CWD differs. |
| M3 | Driver is fixed: `tools/launchd/lib/lane-probe.sh:153–156` explains duplicate discovery and `_mc_probe_pi` passes `--no-extensions`. `_mc_pi_ext_args DIR` emits discovery-off plus explicit repo `.ts` extensions when the directory exists, otherwise nothing. Role runner `scripts/mission_pi_run.sh:286–302` and controller `tools/launchd/mission-control.sh:2442–2443` use it. |
| M4 | Cross-skill launch search finds only two invocations, in each byte-identical `gate-3-route.md`: probe at 334 lacks `--no-extensions`; raw sandbox launch at 347–351 lacks array splice. The adjacent guidance already prefers `scripts/mission_pi_run.sh`. |
| M5 | `tools/launchd/test_pi_ext_args.sh` checks driver/controller/role-runner Wiring, omitting skills; baseline 8 passed, 0 failed, rc 0. It is already wired via LAUNCHD_SUITE at `make/test.mk:86`. |
| M6 | Baseline `/bin/bash tools/launchd/test_pi_ext_args.sh` rc 0 (8/0); `/bin/bash tools/launchd/test_agents_skills_sync.sh` rc 0; `make check-changelog` rc 0. |

Acceptance gates (record actual outcomes, not assumed results):

- `/bin/bash tools/launchd/test_pi_ext_args.sh`: rc 0, 14/0; both roots count at least two launches; copied-root baseline rc 0 and all six mutation/control checks red for the intended reason.
- `/bin/bash tools/launchd/test_agents_skills_sync.sh`: rc 0, including byte-identical resource copies; controller completes this gate if the mirror write is refused.
- `make check-changelog`: rc 0; independently check exact fragment first line and 2–5 line count.
- **Controller only, outside sandbox:** `make test-launchd-drivers`. Existing Make integration is unchanged. Any socket-touching command's result must be labeled **UNINFORMATIVE UNDER SANDBOX** if run in a sandbox; it cannot establish a gate verdict.
- Planner validates JSON with `jq -e .`; run planner validator if present, otherwise the available sprint-executor validator, and record rc plus summary in JSON notes. A sole minimum-2-milestones error is documented, with one milestone retained.

## Risks and limits

Fence-context leakage or a vacuous scan could hide new unsafe launches; explicit block boundaries, offending file:line output, per-root counts and disposable controls cover these risks. Bash 3.2 empty-array expansion requires the exact guarded splice. The resource is already 863 lines; make only the prescribed local edits and reuse the existing test file, without a resource split or companion module. Permission refusal can leave a temporary mirror mismatch, so completion waits for controller mirroring and green gates. This fix repairs skill guidance and regression coverage; controller M1–M6 remain the established live-provider evidence.
