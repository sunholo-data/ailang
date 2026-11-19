# Sprint plan: pi controller session-protocol note

Fleet iteration 69 · 2026-10-29 · ticket `gate0:session-protocol-deadlock:agent-policy-unset`
(stapledon, filed 2026-10-07). Base: `origin/dev` 65253e86e. Status: planned. This plan is
its own design document. The controller's measurements and the planner's re-checks below are
the design.

## Goal and evidence

Stop pi-driven mission controllers from deadlocking at Gate 0. The deadlock happens when the
model tries to satisfy the session-protocol inbox step with a tool that can never satisfy it.

What was measured, and re-checked by the planner against the worktree and the session store:

- **The deadlock.** Stapledon fire `2026-09-29T12-53-33` ran `pi:ollama/glm-5.3:cloud` for
  487 s and died before Gate 0. Its transcript holds **2** bash calls. Neither one contains
  `ailang messages` (both are compound: `ls … | head -50; echo ---; …` and `wc -l …; sed -n …`).
  It also holds **5** `ailang_cli` calls, including `{"argv":["messages","list","--unread","--json"]}`.
  All 5 refused with "AILANG_AGENT_POLICY is unset", and the ack refused 4 times.
- **How the gate counts.** `.pi/extensions/session-protocol-gate.ts` `prerequisitesMet()`
  (lines 217–260) counts the inbox step only from a `toolCall` named `bash` whose command
  contains `ailang messages`. `BASH_ALLOW` (lines 29–33) allowlists `^ailang (check|messages|doctor|builtins|docs|prompt)`,
  but `bashAllowed()` splits on `&&`, `||`, `;` and `|` and requires every segment to match.
  The ack refusal text says "run `ailang messages list --unread`" and does not name the bash tool.
- **Why the model reaches for `ailang_cli`.** `.pi/extensions/ailang-exec.ts` registers
  `ailang_cli`/`ailang_run` even with no policy (default-deny, D4). The lane prompt that would
  explain this is injected only when a policy is attached (lines 322–337).
- **How often.** 11 stapledon pi sessions contain `session_protocol_ack`. 5 of them got at
  least one `REFUSED: session protocol`. One deadlocked (above). The other four (10-02, 10-09 ×2,
  10-19) recovered later through bash.
- **Reach.** Both `dev.ailang.mission-stapledon.plist` and `dev.ailang.mission-world.plist` run
  `~/dev/sunholo-data/ailang/tools/launchd/mission-control.sh`. One driver edit therefore covers
  every mission with a pi controller rung, and no world-fork port is needed. The
  mission-loop-change skill's "Gate 2 — World is a fork" text is out of date on this point.

The fix is a short, driver-side, **pi-controller-only** note. It tells the model how the inbox
step is actually satisfied. It changes no extension, no gate and no policy.

## Decision: where the note goes (system prompt AND user prompt)

The note goes in **both** places: `--append-system-prompt "$note"` and a prefix on the `-p`
prompt. This holds on the fresh path and on the resume path.

- **Why not system-only:** `ailang-exec.ts:322-324` records a measured case (2026-09-16) where
  the `ollama/glm-5.3-flash:cloud` route dropped the system role entirely. The deadlocked fire ran
  `ollama/glm-5.3:cloud`, a sibling route whose system-role handling has not been measured.
  pi controllers are the last-resort GLM/ollama rungs, so a system-only note could vanish on
  exactly the routes that need it. The same extension delivers its own lane text both ways for
  this reason.
- **Why the user-prompt copy is not undone by compaction:** "teaching in a user message dies
  in compaction" is true for teaching that must last a long session. This note governs Gate 0,
  the first few turns, before any compaction can occur. On a resume after compaction, two copies
  cover it. The system copy is re-sent on every pi launch, because pi rebuilds the system prompt
  per process. The resume prompt also carries its own prefixed copy.
- **Why prefix, not suffix:** the protocol is the controller's first action. Nothing parses the
  pi `-p` value: `_mc_pi_prompt` is read only by the pi spawn, and `test_driver_notify.sh` greps
  driver source, not the value. `PROMPT` itself stays byte-identical, and
  `test_mission_prompt_role.sh` enforces that with a golden for every mission. claude and codex
  never see the note.

## Decision: `scripts/mission_pi_run.sh` (pi role spawns) does NOT get the note

1. Role spawns are stages. `mission_pi_run.sh:298-301` launches pi with `AILANG_MISSION_STAGE=1`,
   and `unattendedFromEnv()` (gate line 200) drops the inbox prerequisite for them (#1393).
   The deadlocking step does not exist there.
2. Part of the note would be **false** in a role spawn. A role lane can attach an
   `AILANG_AGENT_POLICY` (the ailang_only lane), and then `ailang_cli`/`ailang_run` ARE granted.
   Telling such a session "ailang_cli is not granted" would recreate the 2026-09-19 failure in
   reverse: text that talks a session out of a tool it has.
3. Role spawns run `--no-session`, so no transcript shows a role deadlock. There is no evidence
   to act on.

The test pins this decision (row T7).

## Files to change during execution

- `tools/launchd/lib/pi-controller-note.sh` (new, ~25 lines). `# shellcheck shell=bash`.
  Defines one function, `_mc_pi_controller_note`, which prints the note with `printf '%s'` and
  no trailing newline. It is pure ASCII (so byte and rune counts agree on the rig), uses no
  globs, and works in bash 3.2 and zsh. Its header comment cites this ticket and the two
  decisions above.
- `tools/launchd/mission-control.sh` (~6 changed lines, inside the `elif [ "$CONTROLLER_PROVIDER" = "pi" ]`
  branch of `_mc_run_once` only):
  1. Leave the line `_mc_pi_session_exists "$MC_PI_SESSION_ID" && _mc_pi_prompt="$MC_PI_RESUME_PROMPT"`
     byte-identical. `test_driver_notify.sh:1012` greps it verbatim.
  2. **After** that line, source the lib from `"$MC_DRIVER_ROOT/tools/launchd/lib/pi-controller-note.sh"`
     (the same pattern as `pi-ext-args.sh` next to it). Then set `_mc_pi_note=$(_mc_pi_controller_note)`
     and prefix the prompt: `_mc_pi_prompt="$_mc_pi_note"$'\n\n'"$_mc_pi_prompt"`. If `$'…'`
     trips shellcheck under `#!/bin/bash`, a literal two-newline string is fine.
  3. In the pi spawn, add `--append-system-prompt "$_mc_pi_note"` between the `-e …controller-bash-cap.ts`
     argument and `--session-id`. The spawn must still match `test_driver_notify.sh:966`'s
     `pi --model "\$MODEL" .* -p "\$_mc_pi_prompt"`.
  4. Add a two-line comment above it citing the ticket and `lib/pi-controller-note.sh`. Do not
     touch the claude or codex branches, `PROMPT`, or `MC_PI_RESUME_PROMPT`.
- `tools/launchd/test_pi_controller_note.sh` (new, ~180 lines, `#!/bin/bash`, bash 3.2).
- `make/test.mk`: add `tools/launchd/test_pi_controller_note.sh \` to `LAUNCHD_SUITES`. It is
  fast, so it goes near `test_pi_ext_args.sh`, not at the head of the list.
- `changelogs/unreleased/2026-10-29-pi-controller-protocol-note.md` (new).

Nothing under `.pi/extensions/**` or `tools/pi-extensions/**` changes (see Ruled out). Every
path above is inside the fleet charter: `tools/launchd/**`, `make/test.mk`, `changelogs/**`,
`design_docs/**`.

### The note text (the executor uses this verbatim; one paragraph, ~700 bytes)

```
PI CONTROLLER SESSION PROTOCOL (from the mission driver). This session runs behind a session-protocol gate: until session_protocol_ack succeeds, bash runs only single read-only commands, and a chained command (any `;`, `&&`, `||` or `|`) is refused if any part of it is not on the read-only list. Do the protocol first, in this order: (1) read CLAUDE.md with the read tool; (2) run `ailang messages list --unread` as ONE plain bash call on its own, with nothing chained to it; (3) call session_protocol_ack. The ailang_cli and ailang_run tools are NOT granted in a controller session (AILANG_AGENT_POLICY is unset on purpose): every call refuses, and none of them counts toward the protocol. Run every ailang command through bash.
```

If shell quoting forces a change (the backticks inside `printf '%s' '…'` are literal in single
quotes, so none is expected), keep every phrase that T5 asserts.

## M1 — driver note, lib, regression suite (one milestone, ~0.5 day, ~230 LOC)

Dependencies: none. Most of the LOC is in the test. Registry reuse: none, since this is a
driver string and its harness.

### Steps

1. **Base run first.** Before any edit, run the done-gate's base arm (below) and record its rc.
2. Write the lib, then the driver edit, then the test. Wire the test into `make/test.mk`.
3. Run every mutation in the test-plan table on **scratch copies**. Never mutate a tracked
   file and then restore it with `git checkout <file>`.
4. Write the changelog fragment, then run the done-gate's head arm.

### Test design: `tools/launchd/test_pi_controller_note.sh`

Overrides that make the mutation drill possible:

- `PI_NOTE_DRIVER` (default `$HERE/mission-control.sh`): the driver whose `_mc_run_once` is extracted.
- `PI_NOTE_ROOT` (default the repo root): used as `MC_DRIVER_ROOT`, and it also locates the
  lib, `.pi/extensions/*.ts` and `scripts/mission_pi_run.sh`.

**Rule 3k (the test runs exactly what the product hands over).** The expected note is read by
sourcing `"$PI_NOTE_ROOT/tools/launchd/lib/pi-controller-note.sh"` and calling
`_mc_pi_controller_note`. That is the same file and function the driver calls. The test never
holds its own copy of the text. The driver side is the **real** `_mc_run_once`, awk-extracted
(`awk '/^_mc_run_once\(\) \{/,/^\}$/'`) together with `_mc_pi_session_exists`,
`_mc_claude_session_prepare`, `_mc_uuid`, `_mc_descendants` and `_mc_kill_tree`, as
`test_controller_capacity.sh:31-47` does. The `log() {` line is grep'd out of the driver.

Harness rules:

- Real `tools/launchd/lib/codex-env-args.sh` and `codex-auth-profile.sh` are sourced for the
  codex arm, as `test_codex_controller_env.sh` does.
- Fake `pi`, `claude` and `codex` go first on `PATH`. Each writes its argv NUL-separated
  (`printf '%s\0' "$@"`) to a per-case file and exits 0.
- `_mc_stalled() { return 1; }`. Use `HARD_TIMEOUT=60`, `STALL_GRACE=60` and `STALL_INTERVAL=5`.
  The fakes exit at once, so `_mc_run_once` cancels both watchdogs. Bound the whole case with
  a `date +%s` deadline and fail loudly if it is exceeded.
- `PI_CODING_AGENT_DIR` points into `mktemp -d`. All files live under one `mktemp -d` with a
  `trap` cleanup. Nothing writes to `$HOME` or `.ailang/state`.
- An **extraction guard** fails the suite if any extract is empty, or if the extracted
  `_mc_run_once` does not contain `_mc_pi_controller_note`. Without it every case could pass
  for the wrong reason.

| Row | Arm | Assertion | Mutation it kills (executor must run it) |
|---|---|---|---|
| T1 | pi, fresh (no session file) | argv has `--append-system-prompt` and the **next** element is byte-equal to `_mc_pi_controller_note` | M1: delete `--append-system-prompt "$_mc_pi_note"` from the spawn |
| T1b | same | the extracted `_mc_run_once` calls `_mc_pi_controller_note`; no copy of the note's first 40 bytes appears in the driver file | M2: the driver rebuilds the string inline (paste the lib text with one word changed **and** with no change) instead of calling the lib |
| T2 | pi, fresh | the `-p` value starts with the note, then a blank line, and ends with `$PROMPT` | M3: system-only. Drop the prompt prefix but keep `--append-system-prompt` |
| T3 | pi, **resume** (a `<ts>_<id>.jsonl` exists under `$PI_CODING_AGENT_DIR/sessions/--<slug>--/`) | `-p` = note + blank line + `$MC_PI_RESUME_PROMPT`, does not contain `$PROMPT`, and `--append-system-prompt <note>` is still present | M4: the resume path loses the note. Move the prefix line **above** the `_mc_pi_session_exists … && _mc_pi_prompt=…` line, so the resume assignment overwrites it |
| T4 | claude (`CONTROLLER_PROVIDER=claude`) | the claude argv has no `--append-system-prompt`, no note substring, and the value after `-p` equals `$PROMPT` byte-for-byte | M5: send the note on the claude path (add `--append-system-prompt "$_mc_pi_note"`, or prefix the claude `-p`) |
| T5 | codex | the last argv element equals `$PROMPT` byte-for-byte and no element contains the note | M6: prefix the codex prompt with the note |
| T6 | note content (lib output) | contains `` `ailang messages list --unread` ``, `bash`, `ailang_cli`, `ailang_run`, `AILANG_AGENT_POLICY`, `CLAUDE.md`, `session_protocol_ack`; the backticked inbox command has no `;`, `&`, or `\|`; total ≤ 900 bytes; pure ASCII (`LC_ALL=C grep -q '[^ -~]'` finds nothing) | M7: (a) remove the inbox command from the lib; (b) change it to `ailang messages list --unread && echo ok`; (c) pad the note past 900 bytes |
| T6b | positive control on the scratch root | after a **benign** rewording of the lib on a scratch `PI_NOTE_ROOT` copy, T1–T3 still PASS | M8: the test hard-codes the note instead of sourcing the lib. This control fails if the test keeps its own copy |
| T7 | role runner | `scripts/mission_pi_run.sh` does not reference `pi-controller-note`, and still sets `AILANG_MISSION_STAGE=1` on its pi spawn | M9: source the lib, and pass the note, in `mission_pi_run.sh` |
| T8 | drift tripwire on what the note claims (read-only grep of the gate and exec extension) | `session-protocol-gate.ts` still contains `part.name === "bash"`, `cmd.includes("ailang messages")` and the `^ailang (check\|messages` allow entry; `ailang-exec.ts` still contains `AILANG_AGENT_POLICY is unset`. On failure, name the file and say the note must be revisited | M10: on a scratch `PI_NOTE_ROOT` copy, rename `cmd.includes("ailang messages")` in the gate copy |

Running each mutation: copy the driver to `$SCRATCH/mission-control.sh`, or copy the needed
tree (`tools/launchd`, `.pi/extensions`, `scripts/mission_pi_run.sh`) to `$SCRATCH/root`.
Apply the mutation with `sed` or an editor on the copy. Then run
`PI_NOTE_DRIVER=… PI_NOTE_ROOT=… /bin/bash tools/launchd/test_pi_controller_note.sh` and read
its rc directly. Every row except T6b must turn red. T6b must stay green. Record each mutation
with its observed rc in the sprint report. Also run M4 and M1 against
`tools/launchd/test_controller_capacity.sh` with `MC_CAPACITY_DRIVER=<copy>`. That suite must
**stay green**, which shows the extra pi argument does not disturb its fake pi. Note this in
the report, not as a kill.

### Acceptance criteria

- On the fresh and resume paths, the pi controller launch passes the lib's exact note as
  `--append-system-prompt` and as the prefix of `-p`. claude and codex argv are byte-unchanged
  (T1–T5).
- `PROMPT` stays byte-identical for every mission (`test_mission_prompt_role.sh` green). The
  existing wiring greps (`test_driver_notify.sh` rows at lines 966/1003/1010) stay green.
- `scripts/mission_pi_run.sh` and `.pi/extensions/**` are unchanged:
  `git diff --name-only origin/dev` lists only the five execution paths above plus this plan.
- All ten mutations are run and recorded. M1–M7, M9 and M10 turn the suite red. M8 is
  detected by the T6b control.

## Done-gate (executor runs all of it; controller runs the dry-runs separately)

Run each gate unpiped and read its rc directly. A piped make reports the pipe's rc. Run the
base and head arms one after the other, never in parallel.

```bash
WT=/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter69
# base (before the first edit)
make -C "$WT" test-launchd-drivers >"$TMPDIR/i69-base.log" 2>&1; echo "base rc=$?"
# head (after all edits)
make -C "$WT" test-launchd-drivers >"$TMPDIR/i69-head.log" 2>&1; echo "head rc=$?"
/bin/bash "$WT/tools/launchd/test_pi_controller_note.sh"; echo "suite rc=$?"
make -C "$WT" -n test-launchd-drivers | grep -c test_pi_controller_note   # must be >= 1
shellcheck "$WT/tools/launchd/lib/pi-controller-note.sh" "$WT/tools/launchd/test_pi_controller_note.sh"; echo "sc-new rc=$?"
# driver: no NEW findings (base has 30 "In …" findings; compare counts and the diff)
git -C "$WT" show origin/dev:tools/launchd/mission-control.sh > "$TMPDIR/i69-drv-base.sh"
shellcheck "$TMPDIR/i69-drv-base.sh" >"$TMPDIR/i69-sc-base.txt" 2>&1
shellcheck "$WT/tools/launchd/mission-control.sh" >"$TMPDIR/i69-sc-head.txt" 2>&1
grep -c '^In ' "$TMPDIR/i69-sc-base.txt" "$TMPDIR/i69-sc-head.txt"
make -C "$WT" check-changelog; echo "changelog rc=$?"
make -C "$WT" check-file-sizes; echo "sizes rc=$?"
git -C "$WT" diff --check origin/dev
```

The base and head rc must both be 0. If base is red, stop and report the pre-existing failure
with its log. Do not "fix" it in this sprint. In the new files, any `# shellcheck disable=` needs
a reason comment, following `test_codex_controller_env.sh`'s header. The driver's head count must
not exceed base.

## Changelog fragment

`changelogs/unreleased/2026-10-29-pi-controller-protocol-note.md`:

```markdown
### Fixed — pi mission controllers no longer deadlock on the session-protocol inbox step (2026-10-29)

A pi-driven mission controller is now told, in both its system prompt and its first user message, that the session-protocol inbox step is satisfied only by one plain bash call `ailang messages list --unread`, and that `ailang_cli`/`ailang_run` are not granted in a controller session. Stapledon's 2026-09-29 fire died before Gate 0 after five `ailang_cli` attempts at the inbox. claude/codex controllers and pi role stages are unchanged. Ticket `gate0:session-protocol-deadlock:agent-policy-unset`.
```

## Ruled out

- **Ask (1): have the driver always export `AILANG_AGENT_POLICY` for pi controllers. Refuted.**
  With a policy attached, `ailang-exec.ts` injects the ailang_only lane prompt ("You have NO shell…",
  line 175) into both the system prompt and the conversation (lines 322–337). The extension's
  own comment records what that did, measured 2026-09-19: on a plain `pi` with bash, "You have
  NO shell" talked sessions out of a shell they had. The controller's whole job is shell work,
  and the inbox step needs bash. A policy would also grant program execution to a controller
  that has no need of it.
- **Exporting a stage marker (`AILANG_MISSION_STAGE`/`AILANG_TASK_ID`) for the controller. Ruled out.**
  It would silence the deadlock by skipping the inbox prerequisite (#1393). But the controller
  is deliberately not a stage: it triages the inbox at Gate 0.
- **Ask (2): let the ack pass when the policy is absent, and any change to the gate's or exec
  extension's text (BLOCK_REASON naming the bash tool, the ack refusal naming bash, `ailang_cli`
  refusing `messages` with a pointer to bash). Out of scope.** These are `.pi/extensions/**`
  edits, which park for Mark under fleet ruling D-FLEET-3, because the pi eval harness shares
  those files. They are probably the better long-term fix, since they would cover attended pi
  sessions as well. The controller may file them as a decision row. This plan does not open a
  gate on that row.
- **Putting the note in `tools/pi-extensions/controller-bash-cap.ts`. Out of fleet scope.**
  Only `tools/pi-extensions/sandbox/**` is in charter.
- **Adding the note to `scripts/mission_pi_run.sh` role spawns.** Covered under "Decision"
  above: stages skip the step, and the note would be false in a policy-bearing role lane.

## Risks and what this does not prove

- **The fake-pi suite proves the argv, not pi's handling of it.** pi 0.85.1's `--help` lists
  `--append-system-prompt <text>` ("text or file contents", repeatable). Whether that flag
  combines cleanly with `-p` and `--session-id` resume on a real pi is for the controller's
  dry-run or the next real pi controller fire. The signal to read is the session JSONL: the
  first `ailang messages` bash call should come before the first `ailang_cli` call.
- **Effect size is unproven.** The baseline is 1 deadlock and 4 slow recoveries in 11 sessions.
  The note targets the observed tool-choice error. It cannot force a model that ignores both the
  system and the user copy.
- **Live-edit hazard.** The stapledon and world plists execute `~/dev/sunholo-data/ailang/tools/launchd/mission-control.sh`,
  and bash reads scripts incrementally. Landing (push, then the main checkout and driver pin
  pick it up) belongs to the controller, under mission-loop-change Gate 3 (no driver edit while
  a mission is executing). The executor works only in this worktree and does not push.
