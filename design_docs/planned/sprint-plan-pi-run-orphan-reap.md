# Sprint plan: reap what a pi role run leaves behind (fleet iteration 109)

Ticket `orphan:pi-runner:own-pgid-children-survive-role-exit` (stapledon, 2026-11-12T12:41Z).
Base `4b9fda4bf`, branch `fleet/i109-pi-orphan-reap`. Three milestones, each sized for one bounded
30-minute claude-haiku run: about +120 LOC new lib, +20 LOC runner, +230 LOC new test suite, +1 make line,
and one changelog fragment.

Lane: **motoko-free harness fix** (`scripts/mission_*`, `tools/launchd/**`). It changes no policy:
no stall, max-seconds, poll, routing, lane, quota or billing default moves, and no verdict name or exit
code changes. The verdict JSON gains two keys and nothing else.

## Premise check (read at 4b9fda4bf; evidence in the Verification Log, V-numbers)

- TRUE: supervision is `scripts/mission_pi_run.sh:316-415`: `set -m` (:316), a `( … pi | awk … ) &`
  subshell (:317-340), `RUNNER_PID=$!` (:341), the poll loop (:360-404), and a group kill **only when
  `OUTCOME != finished`** (:406-412). The verdict heredoc is at :493-521. (V1)
- TRUE, reproduced first-party: an own-pgid child survives both `ok` (rc 0) and `wall_timeout` (rc 13).
  The no-setpgrp control is killed on the wall arm. (V2)
- **INCOMPLETE: "own-pgid" is only half the story.** On the `ok`/`finished` path the runner sends no
  signal at all, so **any** background descendant survives, even one that stayed in the runner's
  group. My extra control (the same stub without setpgrp, ok arm) leaves a survivor with
  `pgid=12959 ≠ pid 12975`. (V3) So the fix must reap on every outcome, which the brief already
  requires, and the tests must not lean on setpgrp to make the ok arm red.
- **FALSE: "must also run on the Linux CI runners that execute `make test-launchd-drivers`".** The
  only CI invocation is the `launchd-drivers` job, `runs-on: macos-latest`
  (`.github/workflows/ci.yml:921`, `:934`). No workflow runs these suites on ubuntu. (V4) The code is
  still written portably (POSIX `ps -o` keywords, `LC_ALL=C`, BSD/GNU-neutral awk), but no Linux leg
  proves it. That is stated as residual R4, not claimed as a pass.
- **Mechanism (c) cannot work, for a reason the brief does not name.** pi's bash tool spawns every
  command with `detached: true` (`…/pi-coding-agent/dist/core/tools/bash.js:52`), and so does the
  mission sandbox's own bash ops (`tools/pi-extensions/sandbox/index.ts:154`). On POSIX that is
  `setsid()`. Measured: a node `detached:true` child has `sid == pid`, while the non-detached control
  inherits the parent's sid. (V5) Every tool command therefore already lives in its **own** session,
  so reaping pi's session would reach no tool descendant. Separately, macOS `ps -o sess` prints `0`
  (a kernel pointer, not the sid), so a shell cannot even list sessions without a per-pid
  `perl syscall(310)` that only works on darwin. (V6)
- TRUE: `_mc_kill_tree` (`tools/launchd/mission-control.sh:396-446`) finds processes with a
  kill-time `pgrep -P` walk (`_mc_descendants`, :388-392), which cannot see a process already
  reparented to pid 1. **Do not move it into `tools/launchd/lib/`:** three suites extract both
  functions from `mission-control.sh` by awk range (`test_mission_kill_tree.sh:42-43`,
  `test_pi_controller_note.sh:47-48`, `test_controller_capacity.sh:47-48`), so moving them breaks
  those suites. This is data coupling that a reference count cannot see. (V7) The new code reuses its
  **discipline**: PID plus `lstart` birth identity, a zombie skip, TERM, then grace, then KILL. It
  does not reuse the function.
- Verdict consumers: nothing in `tools/launchd` parses the verdict as a program. The readers are the
  runner suites (a per-line sed `field()` helper and `jq -e`) and the LLM controller via prose in
  `gate-3-route.md:404-414`, which reads `verdict`, `rc`, `hung_tool` and `stall_bound_seconds`. No
  reader rejects unknown keys. (V8) The new keys go on **compact single lines** (`jq -c`, so
  `"k":v` has no space after the colon). The sed `field()` pattern `"name": ` (colon-space) therefore
  cannot match inside them.
- Side observation (an orphan holding the stdout pipe turns a finished run into `wall_timeout`):
  **out of scope, deliberately.** Real pi tool descendants cannot inherit the runner's pipe. Both
  spawners give the child fresh stdio (`bash.js:54` `stdio: [… "pipe", "pipe"]`,
  `index.ts:155` `["ignore","pipe","pipe"]`), and libuv marks the other fds CLOEXEC. Only the
  test stub, which is the pi process itself, can hold it. That is a stub artifact. Every fixture
  below detaches its stdio (`</dev/null >/dev/null 2>&1`) so the artifact cannot mask a result.
  Not measured: a process spawned by a `.pi/extensions/*` extension with `stdio: "inherit"`. None
  was found (both `spawn(` sites use pipes, V9), but that is a grep, not a proof.
- Repro hygiene: `/tmp/fleet109/repro/*.sh` count and **kill every `/bin/sleep 300` on the host**.
  On the shared rig that can hit a stranger's process. The new suite identifies fixtures by pid file
  plus birth, never by command line. Those command-line negative controls (N1, N2) exist precisely to
  prove that the reaper does not do it either.
- Write scope: `tools/launchd/*`, `scripts/mission_*`, `scripts/test_mission_*`, `make/test.mk`,
  `design_docs/*`, `changelogs/*` are all allowlisted (`tools/launchd/githooks/pre-push:16,19,31,32`).
  (V10) **The sandbox tool half is scoped out** (F2 below), so no `.pi/extensions/**` and no
  `tools/pi-extensions/**` edit is needed.

## Mechanism decision

**Chosen: (a), an ancestry ledger, with one refinement.** Each poll takes a single
`LC_ALL=C ps -A -o pid=,ppid=,pgid=,stat=,lstart=` snapshot. It computes, in awk, the transitive
`ppid` closure seeded by RUNNER_PID (if its birth still matches) **and by every live ledger entry
whose birth still matches**. New members are appended to the ledger as `pid<TAB>birth`. Seeding from
live ledger members is the refinement. It is how a child that a process spawns **after** that process
was reparented to pid 1 is still found (instance 1: Godot was orphaned, and its AILANG sim child was in
yet another group). At role exit, on **every** outcome, the same closure gives the candidate set. The
reaper TERMs each candidate whose birth still matches, waits a 2 s grace (only if it sent a TERM),
refreshes the ledger once more to catch processes born during the grace, KILLs the survivors whose
identity still matches, and reports the result in the verdict.

Why: the ledger is the only candidate that **proves** membership without trusting a name or a
location. Membership is "this exact (pid, birth) was a descendant of this run's subshell, or a
descendant of such a process". All three field instances are covered. In each one the escaping
process lived for minutes under a parent that was still attached (Godot under the 600 s
`script_gate.sh` perl supervisor), so it was seen on many polls.

| | double-fork whose intermediate dies **between two polls** | ownership proof | verdict |
|---|---|---|---|
| (a) ledger | **missed**: the grandchild is reparented to pid 1 before any snapshot sees it attached. With `MISSION_PI_POLL_SECONDS=10`, a pi tool call `cmd >/dev/null 2>&1 &` escapes, because the tool's bash exits in milliseconds. Residual R1. | strong (pid + birth + ancestry) | **chosen** |
| (b) `lsof -d cwd` under `--workdir` | unaffected, because cwd is inherited | **none**. The controller's own Bash-tool shells, a concurrent attended session, or a Monitor with its cwd in the worktree would all be killed. A deleted clone (instance 3) has no resolvable path. | rejected as a kill rule. A report-only variant is follow-up F1. |
| (c) perl `POSIX::setsid` + reap by session | unaffected unless the process calls `setsid` | n/a | **rejected**: pi already `setsid`s every tool (V5), so the run's session excludes all tool descendants, and macOS `ps` cannot report a sid (V6) |
| (e) env token in `ps -E` (evaluated, not in the brief) | unaffected (env survives fork, setpgrp and setsid) | strong (a random token) | **deferred to F1**. macOS hides the env of platform binaries: `/bin/sleep`, a **copy** of it, and `/usr/bin/perl` show no token, while `node` does (V11). It would cover Godot and `ailang`, but not shell utilities. |

(d), a combination: (a) now, with the mechanisms in (e) and (b, report-only) as follow-up F1. One
combination was considered and dropped: a "group-mate" rule (admit any process whose pgid equals a
live member's pgid). It only helps when another member of the group is already in the ledger, and it
does not rescue the common `cmd &` escape, because the escaped process is the only member of its group.
It would add mutation surface for little gain.

Safety rules (each one has a test arm):
1. Identity is `pid` plus a **normalized** `lstart`, made by joining awk fields `$1..$5`. Raw `ps -o lstart=` carries
   4 trailing spaces on darwin (V12), and `%e` day-padding differs. **Every** birth string, from the
   snapshot or from `ps -p`, goes through the one normalizer. A raw string comparison silently matched
   nothing in the planner's prototype. (V13)
2. Never admit pid 1, pid 0, `$$` (the runner's main shell, which is where the reaper is called), or
   any ancestor of `$$`. The ancestor chain is walked through `ppid` in the same snapshot. This holds by
   construction too, since a descendant of RUNNER_PID cannot be an ancestor of `$$`, but it is enforced
   explicitly and tested. (Bash 3.2 has no `BASHPID`. Call the reaper from the main shell, never from a
   `$( )`.)
3. Skip `stat` containing `Z`. Re-verify identity immediately before each signal with a fresh `ps -p`,
   normalized.
4. A snapshot that does not list `$$` counts as a failed snapshot. Its absence is the known-positive
   control. In that case the ledger is left untouched rather than replaced. The ledger is
   **append-only**, so one bad `ps` cannot erase what earlier polls proved.
5. The reaper never matches on command line, cwd, pgid or session.

## M1: `tools/launchd/lib/pi-run-reap.sh` plus its unit arms plus make wiring (≈30 min)

**Files:** `tools/launchd/lib/pi-run-reap.sh` (new), `tools/launchd/test_mission_pi_run_orphan_reap.sh`
(new, unit section only), `make/test.mk` (one line).

The lib is bash 3.2 and `set -u`-safe: no `declare -A`, no `mapfile`, no `${v,,}`, no `BASHPID`, and
every hostile string goes through files or `jq -R`, never `awk -v`. The `bash -n` check is automatic,
through `make/test.mk:145`'s `tools/launchd/lib/*.sh` glob.

- `_pr_birth PID` prints the normalized birth, or nothing:
  `LC_ALL=C ps -p "$1" -o lstart= 2>/dev/null | awk 'NF>=5{print $1" "$2" "$3" "$4" "$5}'`.
- `_pr_members LEDGER ROOT ROOT_BIRTH` prints `pid<TAB>pgid<TAB>stat<TAB>birth` for every live run
  member. It writes the snapshot to a temp file. On a failed snapshot (see rule 4) it returns 1 and
  prints nothing. The awk core, which the planner proved feasible on bash 3.2 and BSD awk (V13):
  ```awk
  BEGIN { while ((getline l < led) > 0) { split(l, f, "\t"); want[f[1]] = f[2] } close(led) }
  NF >= 9 { p=$1; pp[p]=$2; pg[p]=$3; st[p]=$4; b[p]=$5" "$6" "$7" "$8" "$9; ord[++n]=p }
  END { x=self; while (x != "" && x+0 > 0 && !(x in bad)) { bad[x]=1; x=pp[x] }; bad[1]=1; bad[0]=1
        if ((root in b) && b[root]==rootb && !(root in bad)) m[root]=1
        for (p in want) if ((p in b) && b[p]==want[p] && !(p in bad)) m[p]=1
        do { ch=0; for (i=1;i<=n;i++) { p=ord[i]
               if (!(p in m) && !(p in bad) && (pp[p] in m)) { m[p]=1; ch=1 } } } while (ch)
        for (i=1;i<=n;i++) { p=ord[i]; if (p in m) printf "%s\t%s\t%s\t%s\n", p, pg[p], st[p], b[p] } }
  ```
  (`-v self="$$" -v root=… -v rootb=… -v led=…`. Those values are numbers, a normalized date and a
  mktemp path, so they are safe in `-v`.)
- `_pr_ledger_update LEDGER ROOT ROOT_BIRTH` appends the `pid<TAB>birth` of every member not already
  in the ledger.
- `_pr_reap LEDGER ROOT ROOT_BIRTH GRACE REPORT_TSV`:
  1. Run `_pr_ledger_update`.
  2. For each `_pr_members` line, skip it if `stat` has Z or `_pr_birth` does not equal the recorded
     birth. Otherwise capture `ps -p pid -o command=`, truncated to 200 chars, then `kill -TERM`. On
     success append `pid\tpgid\tTERM\tcommand`.
  3. If at least one TERM was sent, `sleep GRACE`.
  4. Run `_pr_ledger_update` again. For each live member with matching identity and no Z,
     `kill -KILL`. Mark the existing row `TERM+KILL`, or append a `KILL` row for a process born
     during the grace.
- `_pr_report_json REPORT_TSV` prints `{"count":N,"list":[…first 50…]}` in compact form. It is built with
  `jq -R -s` (split on `\n`, then `\t`; `pid`/`pgid` with `tonumber`; `command` is the remaining fields
  joined). Never use printf interpolation.

**Make wiring:** add `tools/launchd/test_mission_pi_run_orphan_reap.sh \` to `LAUNCHD_SUITES`
immediately after `scripts/test_mission_pi_run.sh \` (`make/test.mk:79`). The list is ordered
slowest-first, and this suite runs about 60 s.

**Unit arms (U1–U5).** Every arm is red at base because the lib does not exist (source fails, and the
arm prints FAIL). The fixture helper `$TMP/fix.pl` is written by the test. Its modes are `own`
(setpgrp, write pid file atomically via tmp+rename, `exec /bin/sleep 3127`), `super` (`own`, plus a
forked child that setpgrps into a third group and execs
`{"/usr/bin/perl"} "perl","-e","sleep 3127",'hostile"arg\back'`), `stubborn` (setpgrp, `$SIG{TERM}`
ignored, a child as in `super`, loop on `select`), and `termfork` (on TERM, fork a child that writes its
pid file and sleeps, then keep running). `alive()` is `kill -0` plus `stat` not Z, copied from
`test_mission_kill_tree.sh:11-17`. `gone_within PID 10` is a `date +%s`-deadline poll. Cleanup is a trap
that KILLs every pid in `$TMP/fix/*.pid`.
- **U1 identity:** fixture U (live, unrelated) goes into the ledger with the wrong birth
  `Mon Jan 1 00:00:00 2001`. Positive control V goes in with its real `_pr_birth`. Then
  `_pr_members LEDGER 999999 x` must list V and not U. `_pr_reap … 10 R.tsv` must leave V gone within
  10 s and U alive, and `R.tsv` must name V only.
- **U2 never-admit:** the ledger holds `$$` (the test shell), `$PPID` and `1`, each with its real
  birth, plus positive control W. `_pr_members` must list W and none of `$$`, `$PPID` or `1`. **Call
  only `_pr_members` here, never `_pr_reap`.** Under the mutant, `_pr_reap` would TERM the suite itself.
- **U3 closure through a reparented member:** `super` fixture Y has its child Z. Put **only Y** in the
  ledger and pass ROOT `999999`, which is dead. Z must be listed. This is the instance-1 shape,
  deterministic, with no runner timing involved.
- **U4 report JSON:** write a TSV whose command holds `"`, `\` and a non-ASCII byte. Then
  `_pr_report_json | jq -e '.count==1 and .list[0].pid==123 and .list[0].signal=="TERM" and (.list[0].command|test("hostile\""))'`.
- **U5 born during grace:** a `termfork` fixture T goes into the ledger. Run `_pr_reap … 10 R.tsv`.
  T's TERM-child C must have a pid file (fixture fired) and be gone within 10 s. T's row must read
  `TERM+KILL` if T survived the TERM, or `TERM` otherwise, and C's row must read `KILL`. The 10 s grace
  passed **here** is what makes this arm load-tolerant. The runner itself uses 2.

M1 done: `/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_mission_pi_run_orphan_reap.sh`
runs unpiped, rc 0, with U1–U5 PASS. `/bin/bash -n tools/launchd/lib/pi-run-reap.sh` succeeds.

## M2: wire the reaper into the runner plus the runner arms (≈30 min)

**Files:** `scripts/mission_pi_run.sh`, `tools/launchd/test_mission_pi_run_orphan_reap.sh` (runner
section).

Runner edits (keep the file at or under about 555 lines; today it is 528, V1):
1. Next to the `pi-ext-args.sh` source at :314, add `. "$RUNNER_ROOT/tools/launchd/lib/pi-run-reap.sh"`.
2. After `RUNNER_PID=$!` (:341), add `REAP_LEDGER="$STAGE/reap.ledger"; : > "$REAP_LEDGER"; RUNNER_BIRTH=$(_pr_birth "$RUNNER_PID")`.
3. In the loop, as the first statement after `sleep "$POLL_SECONDS"` (:361) and **before** the
   liveness check, add `_pr_ledger_update "$REAP_LEDGER" "$RUNNER_PID" "$RUNNER_BIRTH"`. The poll that
   discovers `finished` then still records the last attached generation.
4. After `wait "$RUNNER_PID"` (:413), unconditionally (the existing group kill stays as it is), add
   `_pr_reap "$REAP_LEDGER" "$RUNNER_PID" "$RUNNER_BIRTH" 2 "$STAGE/reaped.tsv"`. Then compute
   `REAPED_JSON=$(_pr_report_json "$STAGE/reaped.tsv")`. If jq fails, fall back loudly: set
   `REAPED_JSON='{"count":<wc -l>,"list":[]}'` and print a stderr WARNING, so the count is still true.
   Derive `REAPED_COUNT` and `REAPED_LIST` from that value.
5. In the verdict heredoc, after `"agent_end_events"` (:515), add the two compact lines
   `"reaped_count": ${REAPED_COUNT:-0},` and `"reaped": $REAPED_LIST,`.
6. After the summary echo (:524), add
   `[ "${REAPED_COUNT:-0}" -gt 0 ] && echo "pi lane verdict: reaped $REAPED_COUNT process(es) the role left behind — see reaped in the verdict JSON" >&2`.
7. In the header "WHAT THIS SCRIPT DOES", add item 6: one paragraph on reaping by ancestry ledger, why
   not by pgid or session (V5), and residual R1.

The verdict and the rc are **not** changed by reaping.

**Runner arms (R0–R3, N1, N2).** These copy the scaffold of `test_mission_pi_run_tool_stall.sh:12-58`:
sandbox stub, `stub()`, `repo()`, `MISSION_PI_POLL_SECONDS=1`, and `unset` of the stall and max env.
Every arm also asserts `jq -e . "$VF"` (JSON validity) and that the existing `verdict` and `rc` keys are
unchanged. Fixtures are backgrounded from the stub with `</dev/null >/dev/null 2>&1 &`.

| arm | stub | runner flags | expect after (green) | red at base |
|---|---|---|---|---|
| R0 clean | change `f.txt`, `agent_end`, nothing backgrounded | `--max-seconds 30` | rc 0 `ok`; `jq -e '.reaped_count==0 and .reaped==[]'` | keys missing, so `jq -e` fails |
| R1 ok + Godot shape | `perl fix.pl super r1 &`; `sleep 12`; change; `agent_end` | `--max-seconds 40` | rc 0 `ok`; r1 and its child gone within 10 s; `.reaped_count==2`; the `.reaped[].pid` set equals {r1, r1.child}; the child's `command` contains `hostile"arg\back`; N1 and N2 alive | both alive, no keys |
| R2 wall + TERM-proof | `perl fix.pl stubborn r2 &`; `sleep 60` | `--max-seconds 12` | rc 13 `wall_timeout`; r2 and its child gone; r2's row has `signal=="TERM+KILL"`; N1 alive | both alive |
| R3 stall | `perl fix.pl own r3 &`; `sleep 60` (silent) | `--stall-seconds 10 --max-seconds 40` | rc 12 `stream_dead`; r3 gone | alive |
| N1 unrelated, pre-existing | the **test** starts `perl fix.pl own n1` before R1, with its cwd in `r1.wt`, giving the **same command line** `/bin/sleep 3127` | none | alive after R1, R2 and R3 | alive (green at base too; it is a control) |
| N2 unrelated, born during the run, orphaned, cwd in workdir | before R1 the test runs `( sleep 3; cd "$TMP/r1.wt" && perl fix.pl own n2 </dev/null >/dev/null 2>&1 & ) &` | none | alive after R1, with its pid file present (fixture fired) | alive (control) |

Timing margins: each stub stays attached for at least 12 s, so even at load 30–80 a poll sees it many
times. Every "gone" check uses a 10 s deadline. R2 depends on the existing `--max-seconds` wall, and no
arm depends on a bound under 10 s. The suite takes about 55–65 s, which is under the 72 s of
`test_mission_pi_run_tool_stall.sh` in the base run (V14).

M2 done: the new suite rc 0 unpiped. The four existing runner suites still pass, unpiped:
`scripts/test_mission_pi_run.sh`, `tools/launchd/test_mission_pi_run_{commits,sandbox,provider_quota,tool_stall}.sh`.
No existing stub backgrounds anything (V15), so `reaped_count` is 0 there and nothing else moves.
Finally re-run `/tmp/fleet109/repro/repro.sh scripts/mission_pi_run.sh` **only if** it reports
`pre-existing sleep300: 0` (its cleanup is host-wide). The expected result is
`surviving_sleep300=0` on both arms.

## M3: changelog fragment plus the done gate (≈15 min)

`changelogs/unreleased/2026-11-12-pi-run-orphan-reap.md` should be a single
`### Fixed — pi role runs no longer leave processes behind (2026-11-12)` section covering: the three
field instances; the ancestry ledger; that the reap runs on every outcome including `ok`; the new
`reaped_count` and `reaped` verdict keys; that verdict names and exit codes are unchanged; and
residual R1 in one sentence. Format per `changelogs/unreleased/README.md`: it opens with `### `, and
has no `#`/`##` heading.

### Done gate (run each one unpiped; a piped make reports the pipe's rc)
```bash
make test-launchd-drivers         # rc 0. Known inherited load-sensitive flakes: test_skill_sync.sh,
                                  # test_motoko_connection_probe.sh. If one fails, re-run THAT suite alone
                                  # via suite-env.sh and record both runs. Never re-run the whole job to launder it.
git diff --check                  # no output, rc 0
make check-changelog              # rc 0
make check-file-sizes             # rc 0. It gates only Go files >800 lines (make/code-health.mk:164-167),
                                  # so it proves nothing about the shell files. Also record
                                  # `wc -l scripts/mission_pi_run.sh tools/launchd/lib/pi-run-reap.sh` (≤555 / ≤160)
```

## Mutation drill (scratch copy, never `git checkout <file>`)

The recipe is proven at base (V16). Make `MUT=$(mktemp -d)`. Copy `scripts/mission_pi_run.sh`,
`tools/launchd/lib/*.sh`, `tools/pi-extensions/sandbox/{index.ts,package.json,mission.ts,sandbox.mission.json}`,
`tools/pi-extensions/worktree-fence.ts` and the new suite into the same relative paths under `$MUT`.
Apply one mutant with `perl -pi`. Run
`/bin/bash tools/launchd/lib/suite-env.sh "$MUT/tools/launchd/test_mission_pi_run_orphan_reap.sh"`.
Expect rc ≠ 0 **and** the named arm FAIL. Then restore by re-copying.

| # | mutant | killed by |
|---|---|---|
| 1 | delete the per-poll `_pr_ledger_update` in the runner loop | R1, R3 (the orphan is already reparented, so the dead root cannot reach it) |
| 2 | wrap the reap in `if [ "$OUTCOME" != finished ]` | R1 (R0 stays green: control) |
| 3 | drop `b[p]==want[p]` from the ledger seed rule | U1 |
| 4 | drop the `bad[]` never-admit set | U2 |
| 5 | seed the closure from ROOT only (skip `want`) | U3, R1 |
| 6 | delete the KILL pass | R2, U5 |
| 7 | delete the post-grace `_pr_ledger_update` | U5 |
| 8 | replace membership with a command-line match (`$0 ~ /sleep 3127/`) | R1 via N1/N2 |
| 9 | build the report with `printf '{"command":"%s"}'` | U4, R1 (`jq -e .`) |
| 10 | `REAPED_COUNT=$(wc -l < "$REAP_LEDGER")` | R0 |

At least 6 of these (1, 2, 3, 4, 6, 9) must be run and recorded with the arm that went red.
A mutant that no arm kills is a test bug: fix the arm, don't drop the mutant.

## Residuals and follow-ups (explicitly out of scope)

- **R1:** a double-fork whose intermediate dies between two polls escapes. The typical case is
  `cmd >/dev/null 2>&1 &` inside one pi tool call. None of the three field instances has this shape.
  **F1** (ticket): an env-token sweep (`MISSION_PI_RUN_TOKEN=<random>` exported to pi, matched in
  `ps -A -E -ww`). It covers non-platform binaries such as Godot, `ailang` and node, but not
  platform binaries (V11). Pair it with a **report-only** `lsof -d cwd` list of unproven survivors
  under `--workdir`.
- **R2:** a shared daemon that a run boots (for example `ollama serve`) belongs to the run by ancestry
  and **is** reaped. That is the ticket's intent, and the `reaped` list makes it visible.
- **R3:** if the runner itself is killed from outside (mission-control's `_mc_kill_tree`), no reap
  runs. **F3:** a TERM trap that does a TERM-only, no-grace reap of the ledger.
- **R4:** no CI leg runs this suite on Linux (V4).
- **F2 (parked, tool semantics):** `tools/pi-extensions/sandbox/index.ts` kills the tool's group only
  on timeout or abort (:166, :185). A kill on normal `close` as well would reap `cmd &` per call, but
  it would change behaviour for a model that deliberately starts a server in one tool call and uses it
  in the next. That is a behaviour change, not a bug fix, so it is not taken here.
- The pre-existing unescaped `"model": "$MODEL"` in the heredoc (:498) is untouched.

## Verification Log

Commands were run from the worktree at `4b9fda4bf` unless they show a path. Outputs are abridged
only where marked "…".

- **V1** `grep -n -E 'OUTCOME" != "finished"|^set -m|kill -TERM -|^cat > "\$VERDICT" <<EOF' scripts/mission_pi_run.sh`
  gave `316:set -m`, `406:if [ "$OUTCOME" != "finished" ]; then`,
  `409:  kill -TERM -"$RUNNER_PID" …`, `493:cat > "$VERDICT" <<EOF`. `wc -l` gave `528`.
- **V2** `pre-existing sleep 300: 0` (a known-zero baseline), then
  `repro.sh scripts/mission_pi_run.sh` gave `ok_exit: runner rc=0 verdict=ok surviving_sleep300=1 / survivor pid=11353 ppid=1 pgid=11353`
  and `wall: runner rc=13 verdict=wall_timeout surviving_sleep300=1 / survivor pid=11803 ppid=1 pgid=11803`.
  `control.sh` gave `wall: runner rc=13 verdict=wall_timeout surviving_sleep300=0`.
- **V3** `/tmp/fleet109/planner/control_ok.sh` is control.sh with its arm swapped to the ok body.
  `grep -n setpgrp` showed only the comment line 2, so the stub has no setpgrp. The run gave
  `ok_exit_nosetpgrp: runner rc=0 verdict=ok surviving_sleep300=1 / survivor pid=12975 ppid=1 pgid=12959`.
- **V4** `grep -rn "test-launchd-drivers" .github/workflows/` gave only `ci.yml:934: run: make test-launchd-drivers`.
  The job header is `ci.yml:921: runs-on: macos-latest`. `grep -rn "mission_pi_run\|test_mission_pi\|launchd/test_\|run-suites" .github/`
  gave rc=1 (no hits), with the positive control `grep -rln launchd-drivers .github/` returning `ci.yml`.
- **V5** `grep -n detached …/pi-coding-agent/dist/core/tools/bash.js` gave `52: detached: process.platform !== "win32",`
  (pi 0.85.1). `grep -n detached tools/pi-extensions/sandbox/index.ts` gave `154: detached: true,`.
  The node probe printed `detached child: pgid sid pid = 38961 38961 38961` and
  `non-detached control: pgid sid pid = 38958 38958 38962`, with shell sid `38958`.
- **V6** `ps -o pid=,ppid=,pgid=,sess=,lstart= -p $$` gave `6279 81932 6279 0 Thu Nov 12 14:43:38 2026`
  (sess printed `0`), while `perl -e 'print syscall(310,0)'` gave `6279`, the real sid.
  `perl -MPOSIX -e 'defined(&POSIX::getsid)'` gave `getsid no`.
- **V7** `grep -n "_mc_kill_tree\|_mc_descendants" -r tools/ scripts/ .claude/skills` found the
  awk-range extraction in `test_pi_controller_note.sh:47-48` and `test_controller_capacity.sh:47-48`.
  `test_mission_kill_tree.sh:42-43` does `awk '/^_mc_kill_tree\(\) \{/,/^\}$/' "$DRIVER"`.
- **V8** `git grep -n -E "worktree_changed_since_start|stall_bound_seconds|ndjson_bytes_filtered" -- . ':!design_docs' ':!changelogs' ':!.ailang'`
  hit only the runner, its suites, `gate-3-route.md:487` (prose) and two banked retro JSONs. It found
  no program reader in `tools/launchd`. `field()` is
  `sed -n "s/.*\"$2\": \"\{0,1\}\([^\",}]*\)\"\{0,1\}.*/\1/p"` (`test_mission_pi_run_tool_stall.sh:28`).
- **V9** `git grep -n -E "spawn\(|detached" -- .pi/extensions/*.ts` gave two hits, `ailang-exec.ts:91` and
  `ailang-lsp-lite.ts:111`, both `stdio: ["pipe","pipe","pipe"]`.
- **V10** `grep -n … tools/launchd/githooks/pre-push` gave `16: tools/launchd/*|…`, `19: … scripts/mission_*|scripts/test_mission_*`,
  `31: design_docs/*|changelogs/*|…`, `32: internal/config/mission.go|make/test.mk`, `33: tools/pi-extensions/sandbox/*`.
- **V11** The env-token probe gave `copied-sleep visible=0`, `usr-bin-perl visible=0`,
  `node(control) visible=1`. `codesign -dv` reported `Platform identifier=26` for both `/bin/sleep`
  and its copy. An earlier probe on the original `/bin/sleep` also gave 0, while the token was visible
  in `ps`'s own env, which shows that the instrument can see a positive.
- **V12** `LC_ALL=C ps -p $$ -o lstart= | od -c` ended `2 0 2 6` followed by 4 spaces and then `\n`.
- **V13** `/tmp/fleet109/planner/proto.sh` is the prototype of the closure above (not the
  implementation). On its first run, comparing raw `ps -p` output against awk-joined births, the
  ledger was **empty** and the orphan was not reaped. After normalizing, it gave
  `ledger: 19752 … 19755 … 19756 …`, `neg control ok: unrelated 20146 not ledgered`, `TERM 19755`,
  and `survivors sleep 61: 1 (expect 1 = the unrelated one)`.
- **V14** `run-suites.sh 5 <the 5 pi-run suites + test_mission_kill_tree.sh>` at load 2.66 gave rc=0,
  `wall 72s`, with `72s tool_stall`, `66s test_mission_pi_run`, `12s provider_quota`, `11s commits`,
  `7s sandbox`, `5s kill_tree`, and `all 6 suites passed`.
- **V15** `grep -n -E "[^&]&[[:space:]]*($|;|\))|setpgrp|nohup|disown"` over the five runner suites
  (excluding `&&`) gave rc=1 (no hits). The positive control on the runner, `grep -c -E "^\) &$"`,
  gave `1`.
- **V16** The scratch-copy recipe ran on `test_mission_pi_run_sandbox.sh` copied with the runner, lib,
  sandbox and fence files into a non-git temp dir. It gave `scratch-copy suite rc=0`,
  `passed=13 failed=0`, and `git -C "$MUT" rev-parse` reported `not a git repository`.
- **V17** Name collision: `ls tools/launchd/lib/pi-run-reap.sh tools/launchd/test_mission_pi_run_orphan_reap.sh changelogs/unreleased/2026-11-12-pi-run-orphan-reap.md`
  reported `No such file` for all three, with the positive control `tools/launchd/lib/pi-ext-args.sh`
  listed. `git grep -E "reaped|_pr_" -- scripts/mission_pi_run.sh tools/launchd/lib` hit only an
  unrelated docstring in `proc_rusage.py:2`.
