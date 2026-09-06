# V1 Mission — iteration log (append-only)

One entry per mission-control iteration, newest LAST (append). Fixed template — keep every
section, write "none" rather than omitting:

```markdown
## N — YYYY-MM-DD — <headline>
**Picked**: <backlog item + why it was top>
**Reality check**: <what git/code verification of the doc's status found>
**Shipped**: <commits/branches/PRs, evaluator result + score, or "parked: reason">
**Routing evidence**: model=<m> task-class=<design|plan|execute|evaluate|mechanical>
  round1-score=<n> rounds=<n> corrections=<n>
  provider=<p> agent=<a> cost=<$<n>|quota-bucket:weekly-fable|quota-bucket:weekly-opus|unknown>
  <!-- provider/agent/cost appended 2026-07-16 (M2). Leading columns unchanged so historical rows
       still parse. provider = anthropic|codex|gemini|motoko|... ; agent = claude-code|codex|... ;
       cost = $ for metered providers (executResult.CostUSD), quota-bucket:<weekly-*> for Anthropic
       subscription calls, explicit "unknown" otherwise — NEVER silent 0 (Critical Principle 2). -->
**Ruled out**: <hypotheses/approaches refuted this iteration — the anti-re-chase ledger>
**Retro lane**: <skill-fix: file+change | process-fix: change | backlog: new doc | none>
**Next**: <what iteration N+1 should pick up>
```

---

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `v1-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `v1-mission-index.md`.

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `v1-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `v1-mission-index.md`.

## 318 — 2026-09-02 — My fix turned a 2.5% Windows flake into a 100% Windows failure, and CI refuted the executor, the judge and me at once [HARNESS]

**Pick**: queue head `m-message-watcher-windows-wallclock-flake` — `TestMessageWatcherStart` reds on the Windows runner from an absolute wall-clock bound. Confirmed unfixed first-party at a fresh `origin/dev`.

**Progress**: N = **10** design docs remaining before v1.0.0, **unmoved** — this is a HARNESS iteration and it moved the goal by 0, in those words. D-53's UNCLASSIFIED bucket of 4 (which would make it 14) is still named and unruled.

**Outcome**: LANDED · [HARNESS] · evaluator PASS 94/100 then PASS 93/100, zero blocking in both · squash [`bd28f845c`](https://github.com/sunholo-data/ailang/commit/bd28f845c) via PR [#1015](https://github.com/sunholo-data/ailang/pull/1015), 21 checks, zero not-green.

**What happened.** The arm carried two unrelated absolute wall-clock constants — a 500 ms `WithTimeout` as the stop stimulus and a 1 s `time.After` as the bound. Fixed per rule 3m: explicit `cancel()` so the bound measures the property under test, and a budget of `max(20 × measured scheduling latency, 10 × pollInterval)`. **Three executor rounds, and the first two were both wrong in ways nothing local could see.**

**Round 1 — I sent it back on my own measurement.** It derived the budget with only a 1 µs floor, giving **2.3–4.2 ms** where the old bound was **1000 ms**: a ~400× *tightening* on the only machine where the arm has ever flaked. Locally it looked fine (25 runs under 8× contention: budget 2.31–4.18 ms against an actual stop duration of **1.46–3.88 µs**). The argument that killed it is structural — the old bound failed on a runner that must therefore have stalled ~**680 ms**, and a machine capable of that blows 2.5 ms far more often. R1 also shipped a ratio assertion that was a **tautology by inspection** (the floor can only raise the value it compares against) added purely so a mutant would red.

**Round 2 — correct direction, judged PASS 94/100, and CI destroyed it.** `9cf2765c8` reddened **both** Windows jobs deterministically: `--- FAIL: TestMessageWatcherStart (0.00s)` / `watcher_test.go:152: instrument failure: initial task scheduling latency 0s is outside (0, 1s)`. **0.00 s, not the 1.18 s timeout the milestone targets** — instant, and ours (`Build windows-latest` is `success` on the 3 most recent dev commits). The branch that fired is the degeneracy guard that **three independent parties certified unreachable**: the executor ("unreachable by construction"), the round-1 judge, which was *explicitly asked to break it* and reported *"I could not devise a test-only way to trigger this"*, and me ("requires a monotonic-clock anomaly"). All three reasoned on darwin/arm64. On a coarse-clock platform a sub-tick interval reads back as exactly `0s`, so zero is the **normal** reading there — and round 2's own floor already made it safe (`max(20×0, 1s)` = 1 s). The guard rejected a measurement its own floor had handled.

**Round 3 — the first genuine local killer this milestone ever had.** Predicate narrowed to `< 0`; the `>= maximumStimulus` arm's unreachability re-cited to CONTROL FLOW rather than to a clock property that had just been falsified. Two arms on the identical tree, latency forced to zero, exit codes captured **without a pipe**: round-2 code **rc=1** (message byte-identical to CI), round-3 code **rc=0**. Arms DIFFER, so the predicate is the variable. Independently reproduced by the round-2 judge.

**Ruled out / corrected**
- **The queue row's own framing.** It says the flake "taxes every PR". Measured across the last **40** CI runs: `test-windows` failed **1** time (**2.5%**); the other 3 reds were the `launchd drivers (bash 3.2)` race, a different and now-fixed defect. Real defect, low frequency.
- **Local reproduction: refuted, and recorded so nobody re-buys it.** 0 failures in 40 runs across quiet, `GOMAXPROCS=1`, 8× contention, and contention+`GOMAXPROCS=1`. I put this in the executor directive so it would not spend a slot hunting a repro I had already failed to find.
- **`gen/main` does not exist** (judge finding, confirmed first-party). `go build ./...` is rc=1 on `cmd/wasm` **alone**; `ls gen` → No such file or directory. I transcribed that parenthetical from charter prose instead of measuring it — rule 3b(v)(b), committed inside the artifact I was most careful about, and the same stale phrase iterations 145 and 316 passed forward. It is now measured, and the charter's copy should be corrected the next time anyone touches it.
- **The `testing.T` count is 1106, not the 1107 I asserted twice** (judge finding, confirmed). My pattern `'testing.T'` left the `.` unescaped, so it matched `testing T` — with a space — in `internal/coordinator/mock_store_test.go`, where the literal count is **0**. Proven by `comm` against `git grep -l 'testing\.T'`. **I defended 1107 once on "three independent methods" that all shared the same unescaped pattern** — three readings of one broken instrument, which is a control that cannot see the defect it controls for.
- **The executor's claim that `gofmt -l <directory>` does not recurse: FALSE.** Control fired — `gofmt -l` on a directory printed a misformatted file living in a *sub*directory. Its glob was an equivalent, not a correction. Reporting it was still right.

**Routing evidence**
| role | pinned | actual | notes |
|---|---|---|---|
| controller | `$CONTROLLER_ID` | `claude:claude-opus-5` | session |
| designer | rotation | **did not run** | queue row was a fully-specified fix with first-party CI evidence and a judge-ranked remedy; a design doc for a ~30-line test change would spend the Fable diet for nothing. Rotation pointer untouched at `claude:claude-fable-5`. |
| planner | `codex:gpt-5.6-sol` | **did not run** | same reason — single-file, single-function, no milestones to sequence |
| executor | `codex:gpt-5.6-sol` | `codex:gpt-5.6-sol` | probe rc=0; three sandboxed runs; no git writes; containment verified byte-identical (exactly the 4 pre-existing dirty files at start and end) |
| evaluator | `sonnet` | `sonnet` | own worktree, both rounds; generator≠judge holds (OpenAI vs Anthropic) |

metered=$0.00 of the $5 ceiling — every lane a quota bucket; no quorum round.

**Scope held.** Single arm, all three rounds. The standing exposure stays a queue row and was NOT swept: **54** `_test.go` files under `internal/`+`cmd/` carry a hardcoded `N * time.Millisecond` bound (control — 56 mention `time.Millisecond` at all; fresh negative literal 0); **0** test files repo-wide vary `GOMAXPROCS` (control — 1106 mention `testing.T`).

**Next**: the queue head is now `m-probe-derace-has-no-killer` (iter-317, judge-found: a full revert of the process-tree de-race passes all 42 arms, so CI cannot see that fix disappear), then `m-probe-discovery-default-30s-unpinned`, then the VERIFY-then-route `m-docparse-v0340-reports-2026-09-01` whose iface-cache half has already failed to reproduce in two shapes.

## 319 — 2026-09-02 — CI corrected me and the judge together, and the executor's self-reported deviation was the thing we both overruled [HARNESS]

**Pick**: queue head `m-probe-derace-has-no-killer` — a full revert of iteration 317's process-tree de-race passes the whole suite, so CI cannot see that fix disappear. **The claim was CONFIRMED first-party before any routing**, not inherited from the judge who filed it: the full revert, asserted LANDED (sha256 `f0b5e024`→`e3cc8148`), BUILDS (`bash -n` rc=0) and intended-effect (`discovery_deadline` refs 1→0, with a control on the surviving validation line), passed **all 42 arms rc=0 in 112s against a 50s baseline**.

**Progress**: N = **10** design docs remaining before v1.0.0, **unmoved** — this is a HARNESS iteration and it moved the goal by 0, in those words. D-53's UNCLASSIFIED bucket of 4 (which would make it 14) is still named and unruled.

**Outcome**: LANDED · [HARNESS] · evaluator PASS 80/100, zero blocking · squash [`f5d031161`](https://github.com/sunholo-data/ailang/commit/f5d031161) via PR [#1020](https://github.com/sunholo-data/ailang/pull/1020), 21 checks, zero not-green, CLEAN. One file, +51/−7: the wall-clock arm's lane deadline derived as `ARM_CAP_SECS + 30` (from the knob, not hardcoded), and the missing suite-scope leak guard for `PROBE_TREE_DISCOVERY_SECS`.

**The central finding — CI overruled the independent judge and the controller, and vindicated the executor.**
The executor added two arm-scoped stabilizers (`PROBE_TEST_PGREP_LOOP_DELAY=1`, `PROBE_TEST_DRIVER_SLEEP=$discovery_killer_lane_secs`) and **said so**, reporting that its directive was under-specified without them. The judge then measured one of them **unpinned** — reverting that hunk alone left the suite 42/42 green — and identified its justifying comment as **false** (`date +%s` has 1s granularity, so the delay cannot make the discovery check fire sooner). I measured the other unnecessary across **8 local runs**, quiet and under 8× CPU contention. Removing both looked like textbook rule 3n(b): a deletion, not scope growth, and the clean suite even returned to its 51s baseline while MUT-1 stayed a sole killer.
`launchd drivers (bash 3.2)` then reddened **deterministically on the first push**, step 4, read by arm NAME rather than exit code: `not ok - descendant discovery refuses on the real wall-clock deadline lacked expected message`, with `driver_rc=0` and an empty peer set. The mechanism is precisely the one the executor had named: `run_lane` calls `sample_tree` only from **inside its sampling loop**, so a stub driver that exits before the lane enters that loop means the discovery walk is never reached at all. Fast darwin/arm64 always wins that race; the GitHub macOS runner does not. Both stabilizers restored; the comment now records that *local greenness is exactly the evidence that failed*, so the next reader does not delete them for my reason.
**This is iteration 318's rule in mirror image, and I had read that rule at Gate 1 this same iteration.** 318 established that a *value property* (a clock's granularity, a scheduler's timing) is platform-scoped and may not be declared **unreachable** from one host. The symmetric half was unwritten: it may not be declared **unnecessary** from one host either. Adding a reviewer did not help and could not have — the judge and I shared the platform, which was the premise.

**Ruled out / corrected**
- *"This change raises the CI flake rate and must not land"* — **my own finding, REFUTED by the judge.** I measured N=4 under 8× spinners: pristine 0/4, this commit 1/4, wall time 55–66s → 82–95s, and flagged it potentially blocking. At N=8 the judge measured pristine **0/8** and the commit **0/8** at comparable times, with the box's own load average swinging **66 → 47 → 42** across my three sequential blocks. On a shared rig a *blocked* A/B/C design cannot separate the diff from ambient load; the correct design is **interleaved**. My N=4 was flagged weak at the time and that caution was warranted.
- *"The 150s driver sleep leaks an orphaned process"* — **CONFIRMED by the judge via `ps`**, not merely hypothesised: orphaned `ailang-stub`+`sleep 150` trees reparented to pid 1 and alive after the whole suite, a 75× widening of a ~2s default, because `run_lane`'s backgrounded driver is never killed on the `instrument_failure` path. Real, and **not** shown to move failure rates. Left as a queue row rather than fixed, since closing it means changing the probe's cleanup path, which this milestone deliberately did not touch.
- *"MUT-3 reds, so the leak guard works"* — nearly banked, and **wrong**: rc=1 exactly as predicted, but the `not ok` was `hermetic live success path completes`, a different arm. Proven an ambient flake because MUT-5 applied the **identical** leak with the guard neutered and passed 42/42. Re-run, the guard kills **3/3 by name**. Rule 3j's corollary paid for itself: read which arm failed, never the exit code.
- *"SonarCloud is a pick"* — `failure` on **7 of 7** commits walked back; inherited, not required, already tracked as `sonarcloud-new-code-gate-red`.
- *"An open PR on my own account is mine"* — **#1016 is the DOCS mission's**, attributed via `git worktree list` before acting (`.ailang-driver-pin/docs`). `--author` is a fleet filter.

**Routing evidence**: controller `claude:claude-opus-5` (session). **Designer not spawned** and the rotation pointer untouched at `claude:claude-fable-5` — the queue row was a fully-specified fix with a judge-ranked remedy in ONE test file, so authoring a design doc would have spent the one-doc Fable diet for nothing; recorded as a capability judgement, not a probe failure. **Planner not spawned** — no design doc exists for this row, so the "doc but no plan" condition never arose and `derive-planner-lane.sh` was not consulted. **Executor `codex:gpt-5.6-sol`** via the cross-provider recipe (probe rc=0; the `provider:model` form must NOT use the Agent tool), one sandboxed 30-min-capped run, delivery asserted ≥200B, stdin closed, no git writes, rc=0, one file. **Evaluator `sonnet`** via the Agent tool, in its **OWN** worktree (`.wt-iter319-eval`, detached at the sprint commit, 0 dirty files at handoff), one round, **PASS 80/100 zero blocking**, five named targets to attack. generator≠judge holds on provider: generator OpenAI, judge Anthropic, and the judge is distinct from the controller's model. Every spawn directive carried standing rule 7's operative half in its own words, per the rule the running skill was missing. Metered **$0.00** of the $5 ceiling; every lane a quota bucket; no quorum round.

**Friction / process**
- The **running skill was 7 commits / 186 lines behind origin** (3,929 vs 4,115), measured by `cmp` against the **resolved** symlink target (inode `60291442`; the pin's own copy is `63919825`, a different file). It lacks D-52's per-gate heartbeat contract **entirely** (`mission-heartbeat`: origin 10, running **0**) though the script exists — stamped every gate by hand. The drift is **not** a clean subset: 64 uncommitted lines in the main checkout duplicate work already on origin.
- **D-54 escalated materially.** The main checkout went **9 ahead / 27 behind at Gate 1** to **22 ahead / 31 behind by Gate 4** — 13 new unpushed commits *within this one iteration*, because an attended session is working in that tree right now (M-COORDINATOR-EXECUTION-TRUST through three quorum rounds, standard-mode Anthropic OAuth, Fable 5.1). The row was written about 9 stranded bookkeeping commits; it is now 22 commits of substantive **feature** work invisible to every unattended pick. Containment held: exactly the 4 pre-existing dirty files, byte-identical at start and end.
- A bare `ailang messages list --unread --json` returns only **20** of **48** unread; `--limit 200` is required for a complete enumeration. My first count used `grep -c '^ID:'` and read **0**, because the list view has no `ID:` lines — rule 3a fired on my own instrument.

**Next**: the queue head is now `m-probe-discovery-default-30s-unpinned` (the 30s default is a production-path tightening nobody chose; a mutant 30→5 passes 42/42 — and note rule 3n's warning that enlarging it is not a fix), then the VERIFY-then-route `m-docparse-v0340-reports-2026-09-01` whose iface-cache half has already failed to reproduce in two shapes, then `m-changeclass-unknown-consumers` as a precondition for Sprint 2. Three residuals filed here rather than absorbed: both suite-scope leak guards fire only *retrospectively* after arm 41; the suite has a genuine ambient-contention flakiness independent of this diff (judge saw 3/8 under load at three unrelated arms); and `run_lane`'s backgrounded driver is never killed on the `instrument_failure` path.

## 320 — 2026-09-02 — dev was red on Windows for a defect this repo had already fixed three times, each fix private [HARNESS]

**Pick**: **dev RED, which outranks the queue** — `CI` and `Build and Release` both `failure` on `origin/dev` `28002af1e`, and V1 owns `sunholo-data/ailang`. Attribution measured, not assumed: `test-windows` is `failure` on all four commits walked back to the merge `40f0554c1` that introduced it, and the four commits before that merge carry **NO-RUN** (only a push tip gets one — the wrong unit here would have read as a 9-commit outage). SonarCloud was red too and is NOT the pick: inherited, already tracked as `sonarcloud-new-code-gate-red`.

**Progress**: N = **10** design docs remaining before v1.0.0, **unmoved** — this is a HARNESS iteration and it moved the goal by 0, in those words. D-53's UNCLASSIFIED bucket of 4 (which would make it 14) is still named and unruled. What DID move: `m-spawn-pin-enforcement` is on origin for the first time, so the queue an unattended pick sees is finally the queue Mark filed into.

**Outcome**: LANDED · [HARNESS] · evaluator **PASS 95/100 round 2, zero blocking, "ship it"** (round 1: 86/100, one BLOCKING) · PR [#1025](https://github.com/sunholo-data/ailang/pull/1025), four commits. **`test-windows` and `Build windows-latest` are both `success` on the PR head — the only instrument that can verify this fix, and the reason the claim is not "it should work".**

**The defect, and why it was a sweep.**
`t.Setenv("HOME", dir)` does not redirect `os.UserHomeDir()` on windows. Verified against GOROOT's own source rather than from memory: `UserHomeDir` consults **exactly three** variables — `USERPROFILE` on windows, `$home` on plan9, `HOME` elsewhere — and errors when the chosen one is empty (android/ios return constants and are not in this matrix). So a test that sets only HOME resolves the runner's real profile, never sees its own fixture, and fails **for the platform rather than for the code**. Four arms, two packages: `TestResolveAnthropicCredential_FallsBackToClaudeCredentialsFile`'s two subtests in `internal/ai`, and `TestStandardModeCostProvenance_CredentialFileIsSubscription` in `internal/eval_harness`. The production resolver was correct throughout.
**The same private helper had already been written three times** — `setHomeDir` (cmd/ailang), `setHomeDirForTest` (internal/effects), and an inline GOOS-branched pair covering 2 of internal/executor's 6 sites. Each was a correct local fix and the next call site went red anyway: *guard the helper, miss the call site*, this loop's own named shape, arriving in the repo it keeps naming it about. So the deliverable is one `testutil.SetHomeDir` (16 bare sites → 1) plus **`make check-home-isolation`**, wired into `make/code-health.mk`, `make/ci.mk`'s `ci:` **and** `.github/workflows/ci.yml` — the third because ci.yml itself records that `make ci` is a local aggregate CI never invokes, so a gate added only to the makefile would not run at all.

**The judge earned its slot twice, and both findings were things I could not have found by re-reading my own work.**
- **It broke the gate.** The line-oriented first draft missed `t.Setenv(\n\t"HOME",\n\tdir,\n)` — and `gofmt -l` leaves that form alone, so it is reachable **by accident**, not by evasion. The matcher is now whitespace-normalised.
- **It found a live instance the sweep had missed.** Four `os.Setenv("HOME", …)` sites in `cmd/ailang/pkg_lock_ratchet_test.go`, whose code under test reaches `os.UserHomeDir()` via `internal/messaging/config.go:158,235`. Confirmed first-party before acting; converted; and now pinned — reverting that file alone reds the gate with 8 lines reported, rc=0 after restore. The matcher covers `os.Setenv` as a result, which is what makes the class closed rather than the instance.
- **It refuted my rule-3n disposition and I reproduced it.** I told it the `ci.yml` hunk had no local killer. It does: `internal/cihygiene/gate_wiring_test.go`'s `TestGateTargetsAreWiredIntoAWorkflow`, a meta-gate I did not know existed. Reverting that hunk alone (sha256 asserted moved, restored byte-identical) gives *"gate-shaped make targets are not wired into any workflow: check-home-isolation, test-check-home-isolation"*.

**Ruled out / corrected**
- *"The judge's 317 contradicts my 334"* — **neither was wrong, and the disagreement was point-in-time, not scope.** `git grep -c -F 't.Setenv(' -- '*.go'` summed is 334 at base, 317 at the mid-sweep commit the judge measured, 318 at the final head; the delta base→head is exactly 16, the converted sites. The judge found its own instrument bug in the same breath (summing `$2`, the path, instead of `$NF` when a revision is given, which coerces to 0) — the identical class I had just caught in myself: my first re-measure used a mis-quoted `-F` pattern and returned a confident **0** for a pattern the `-n` arm simultaneously showed 3 hits for. Two arms disagreeing is the only reason either of us noticed.
- *"SonarCloud is a pick"* — inherited red, present on the parent commits, already a queue row.
- *"The PR's missing CI runs are a dropped webhook"* — **no**, and the rule's own ordering saved a wrong diagnosis: `gh pr view --json mergeable` read `CONFLICTING`/`DIRTY` on the first look. origin/dev had advanced 9 commits; the only shared file was `changelogs/v0.32-current.md`, where both sides had inserted under `## [Unreleased]`. Rebased, resolved keeping both sections, force-pushed my own branch, and all five runs appeared.
- *"The 4 `os.Setenv` sites in `internal/loader/stdlib_resolver_test.go` are the same defect"* — **no**, and this is why the gate has an exemption rather than a blanket ban: `stdlib_resolver.go:88,94,100` read `os.Getenv("HOME")`/`os.Getenv("APPDATA")` **directly, per GOOS**, never through `os.UserHomeDir()`, so the three-variable helper would be wrong there rather than merely unnecessary. The exemption is asserted **live** — renaming that file's calls away makes the gate exit 2 `INSTRUMENT BROKEN … remove it`, because a stale exemption is how an allowlist quietly becomes the rule.

**Mutation drills** (each asserted LANDED by sha256, BUILDING, intended-effect against the system's own view, and restored byte-identically from a `cp` backup): drop `SetHomeDir`'s USERPROFILE line → `TestSetHomeDir` FAILS; drop its plan9 line → FAILS; reintroduce one bare site → gate rc=1, rc=0 after restore; revert the ratchet fix → gate rc=1 (8 lines); plant a gofmt-canonical multi-line call → rc=1, reported by file; delete the gate's allowlist line → rc=1 flagging `home.go` itself; rename the exempt file's calls → rc=2 INSTRUMENT BROKEN; empty fixture and one-shape fixture → rc=2 both; empty and missing scan root → rc=2 both. **And the one that matters most, rule 3n: reverting the whole `internal/ai` hunk leaves `go test ./internal/ai` rc=0 on darwin — NO local killer, by construction — while the gate reds.** That is the sprint's thesis in one measurement: the gate is the only thing on this machine that can see the defect the sprint exists to fix.

**Routing evidence**: controller `claude:claude-opus-5` (session). **Designer not spawned**, rotation pointer untouched at `claude:claude-fable-5` — a dev-red fix-forward is not a queued design item, and authoring a doc for a test-side sweep would have spent the one-doc Fable diet for nothing. Recorded as a routing judgement, not a probe failure. **Planner not spawned** — no design doc exists for a red, so the "doc but no plan" condition never arose and `derive-planner-lane.sh` was not consulted. **Executor `codex:gpt-5.6-sol`** via the cross-provider recipe (probe rc=0; a `provider:model` value must NOT use the Agent tool), one sandboxed 30-min-capped run, directive delivery asserted ≥200B, stdin closed, no git writes, rc=0, per-milestone `.snap/` snapshots; the controller reconstructed both commits and proved the reconstruction faithful by `shasum -c` over a 17-file manifest, **17/17 OK**. It correctly labelled its own aggregate `go test` UNINFORMATIVE UNDER SANDBOX (denied loopback binds) and I re-ran every gate outside the sandbox, including `internal/daemon`, `cmd/ailang` and `internal/cihygiene`, all rc=0. **Evaluator `sonnet`** via the Agent tool in its **OWN** worktree, two rounds, 86/100 then 95/100; generator≠judge holds on provider (generator OpenAI, judge Anthropic, both distinct from the controller). Round 2's directive re-measured every number on the moved tree per the staleness rule, listed the changed hunks exhaustively from `git diff`, and carried round-1 findings forward by name for adjudication. Every spawn directive carried standing rule 7's operative half in its own words. Metered **$0.00** of the $5 ceiling; every lane a quota bucket; no quorum round.

**Human channel**: **D-54 ANSWERED and RESOLVED in this iteration.** Mark, `#972` `2026-09-02T07:17:34Z`, verbatim *"D-54 b"* — the loop may branch the main checkout's unpushed `dev`, push, and open a PR, leaving the merge to CI. Twenty-one minutes later he cleared the divergence himself, attended, with merge `40f0554c1`, so the grant is standing rather than pending: main checkout `dev` is **0 ahead / 0 behind**, iteration 319's own Gate-5 skill edit `7292ec780` is an ancestor of origin, and **the running skill is byte-identical to `origin/dev` for the first time in at least four iterations** (`cmp` against the RESOLVED `readlink -f` target, not the pin's own copy — different inodes, and the relative-path form reads green from the wrong file).

**Friction / process**
- **A gate's anti-vacuity floor can itself be vacuous, one level down.** The judge showed the fixture floor (`>= 3` matches) is a COUNT, not a SHAPE check: a fixture with three copies of the same bare shape passes, and the gate's `os.Setenv` coverage would then be silently unprotected. Filed as a queue row with its fix (three named per-shape counts), not fixed inline — a finding is a queue row, not scope growth.
- The gate costs **22.55s** over 2467 `.go` files, subprocess-per-file (`tr | grep | wc` ×3 each). Acceptable inside `make ci`; annoying for a contributor iterating locally. Architectural, not algorithmic. Same queue row.
- The self-test's five new arms have **no killer** — nothing asserts a minimum arm count, so losing them would be invisible. Same queue row, and it is the counterpart of the floor finding: the shape-blind floor and the shape-testing arms are currently each other's only backstop.
- The line-number reporter hardcodes `(t|os)` receivers, so a call on any other receiver is correctly CAUGHT but mis-described as "whitespace-spanning". Diagnostic quality only; same queue row.

**Next**: `m-spawn-pin-enforcement` — the queue head, and newly visible to unattended picks now that Mark's merge put it and its design doc on origin. Design approved attended 2026-09-01; the sprint is enforcement code, not a design round.

## 321 & 322 — 2026-09-02 — NO ENTRY: both slots died mid-flight holding this fix, and neither left a charter row [ADMIN]

Recorded by iteration 323 so the log does not silently skip two numbers. Neither iteration
wrote a STATUS stamp or a log entry (`grep -ci "ITERATION 321"` and `322` in the charter =
**0**; control `ITERATION 320` = 2). Both did real work and both died before landing it:
iteration 321 opened PR [#1030](https://github.com/sunholo-data/ailang/pull/1030) with the fmt A/B removal and the new
`check-referenced-paths` gate; iteration 322 rebased that PR onto `701f86e5b`, **corrected
its predecessor's stated root cause** (the original body blamed an `Error 127` from a
*missing* script; by then the script had been restored and the real mechanism was the
marker-extraction floor firing), and pushed `e5b62347f`. Then nothing. Their traces were
exactly the ones Gate 2 names: an open PR on the fleet account, and the worktrees
`.wt-iter321`, `.wt-v1-iter321-{eval,record,verify}`, `.wt-v1-iter321b-{before,eval}`,
`.wt-v1-iter322-{eval,fmt-dangling}`, `.wt-iter322-record`.

**With iteration 317, that is three slots in seven that died holding finished work.** The
loop cannot diagnose why its own slots are dying; it can make the frequency visible.

## 323 — 2026-09-03 — dev was red for 24h on five defects stacked in one job, and only the first was visible [HARNESS]

**Pick**: **NOT the queue head.** A cross-mission handoff from `mission-docs` reported dev
RED; V1 owns `sunholo-data/ailang`, so per Gate 1 the red outranks the queue and docs
correctly kept its own pick and handed it over. I corrected their attribution back to them
on the cross-mission channel: not `55891002f`/`08da6ceea` (resident/A2A, docker-pi) but
**`327db37cd`, 2026-09-02 13:19**, against the last green `7668ed9df` at 13:18 — ~24 hours
and ~50 commits, on `test`, `launchd drivers (bash 3.2)` and `Build ubuntu-latest`.

**Progress**: N = **12** design docs remaining before v1.0.0, **unmoved** — this is a
HARNESS iteration and it moved the goal by 0, in those words.

**Outcome**: LANDED · [HARNESS] · evaluator **PASS 93/100, one BLOCKING finding, reproduced
first-party and closed** · PR [#1030](https://github.com/sunholo-data/ailang/pull/1030), 10 commits, squash-merged
[`b51e53f78`](https://github.com/sunholo-data/ailang/commit/b51e53f78).

**Three quarters of this was verifying a dead predecessor's work, not writing my own.**
Gate 2's died-mid-flight trace found PR #1030 already carrying iterations 321 and 322's
work. The instruction there is VERIFY AND LAND, not redo — and verify exactly as any other
inherited claim, because nobody has reviewed that work since the agent that wrote it stopped
existing. So: baseline arm on a pristine tree first (`go test ./internal/eval_analysis/...`
**rc=1** with CI's exact text, `make test-launchd-drivers` **rc=2**, same text), then the
branch (**rc=0**, **0** `FMT_AB` occurrences), then a rebase onto a `dev` that had moved 9
commits, then an independent judge.

**The cause chain, verified first-party rather than inherited from the PR body.**
`c8c841e24` deliberately removed the Wednesday fmt A/B from `tools/launchd/nightly-eval.sh`,
taking the `# BEGIN/END FMT_AB_TESTABLE_FUNCTIONS` markers with it, and its message says
*"its schedule test is deleted"*. That was true only because a **concurrent docs-mission
commit** (`327db37cd`, described and intended as docs-only) had deleted
`tools/launchd/test_fmt_ab_schedule.sh` **by accident** — a staged deletion that rode along
with a `git add <one path>`. `ce05af862` then correctly reverted the accident, restoring a
test whose subject had legitimately gone. Two callers went red. Nobody did anything wrong in
isolation; the three commits compose into a defect.

**The five, each revealed only by fixing the one in front of it.**
1. the dangling fixture reference (`test`, `launchd drivers`, `Build ubuntu-latest`);
2. `lint` **step 4** `fmt-check` — 7 files unformatted, arriving with the coordinator merges;
3. `check-file-sizes` — `backend_gcp.go` 811 and `inbox.go` 850;
4. `lint` **step 6** `golangci-lint unused` — two findings, only reachable once (2) passed;
5. `Build windows-latest` — 21 `TestFinalize_*` tests, only reachable once (1) passed.

**The shape is the finding, not any one defect.** `check-file-sizes` is **step 15** of the
`test` job and the job was dying at **step 11**, so steps 12–60 — **45 gates** — read
`skipped` for a day. Measured across the boundary: at the last green those two files were
**788** and **773** lines and step 15 read `success`. They crossed 800 inside the window
where nothing could see them. The same mechanism operates twice more in different clothes:
inside the `lint` job's step list (defect 4 behind defect 2), and across the build matrix via
fail-fast (defect 5 behind defect 1 — `Build windows-latest` reads `cancelled` on **every**
recent dev commit, so this branch is the first place that leg has COMPLETED in a day). And a
sixth: `SonarCloud` reads `none` on every dev commit in the window because it is step 58.
**A red that fails EARLY in a long ordered job silently suspends every gate behind it, and
the check set then reports ONE failure where there are six.** Filed as `m-ci-serial-gate-masking`.

**What I did NOT do, and why.** Defect 4's `diffResultFromEvidence` was **not** deleted. Its
consumer is **M1b of M-COMPLETION-PATH-PARITY**, which that sprint's plan records as
deliberately outstanding, so deleting it would remove half a contract whose other half is
already on `dev` — the Import System Disaster rule in this repo's own coding standards,
applied rather than quoted. Annotated `//nolint:unused` with the reason, following the 41
existing such sites. That leaves a debt with no gate to retire it, filed as
`m1b-nolint-suppression-owed`. None of defects 2–5 is my work; they arrived with a concurrent
workstream's merges, and are flagged in the commit messages so that workstream can object.

**Defects 4 and 5 share one root**, which is why the fix is two lines rather than two fixes:
`newFinalizeHarness` registers a `t.Cleanup` closer for two of its three stores and none for
the observatory backend, so `observatory.db` stays open. POSIX unlinks open files; Windows
refuses. The dead `cancel` field is the vestige of the cleanup that was never written.
**Audited rather than patched** (Principle 3): `observatory_sync_test.go` opens the same
backend three times and already `defer backend.Close()`s every time — one isolated omission,
not a package-wide pattern.

**The judge earned its slot, and its blocking finding was real.**
Directed to attack the new `check-referenced-paths` gate, it made it return **rc=0 on a
fixture carrying FOUR dangling references** in forms the matcher did not recognise (`.bash`,
`.pl`, uppercase `.SH`, and a make-variable-composed path). I reproduced it first-party WITH
a control before acting — the same fixture carrying a `.sh` reference reds at rc=1, so the
gate was *firing*, just not *looking*. Three forms are now matched (a captured extension
tested case-insensitively against a set, rather than a hardcoded alternation); the fourth
cannot be resolved without evaluating make variables, so the script's header now discloses
the scope in full and ends *"a green here means no LITERAL `tools/`/`scripts/` script
reference dangles, never no reference dangles"*.
Its NON-BLOCKING finding was sharper than its label: arm **A2 did not pin the branch it
named**. It asserted `rc!=0` plus a path substring, and the untracked `elif` catches the same
fixture and prints the same path — so neutering the missing-path branch left A2 **green**.
Self-test **6 arms → 10**; four mutants, each asserted LANDED (sha256 differs) and PARSING
(`bash -n`) before its result was read, restored byte-identical: MUT-1 now kills A2 **by
name**, MUT-2 kills exactly the `.bash`/`.pl` arms, MUT-3 exactly the uppercase arm, MUT-4
exactly the disclosure arm.

**Ruled out / corrected**
- *"My gofmt commit is a semantic no-op because `git diff -w` is empty"* — **I asserted that
  before measuring it and the measurement refuted me.** `git diff -w` is **2 lines**, both
  trailing blank lines at the end of one test file; every other hunk is struct-literal key
  alignment. Amended, with the wrong claim named in the message rather than quietly replaced.
- *"91 of the removed lines are missing from the new file"* — **an instrument failure of
  mine**, not a finding. My purity checker's header-skip heuristic consumed the whole file;
  its own `body_nonblank=0` reading is what exposed it. Re-measured against the whole file
  with positive and negative controls: **0** missing, **0** added, for both pairs. The judge
  re-derived it independently by a multiset method and agreed.
- *"The per-form gate arms all fail (rc=127)"* — **a zsh trap, not a finding.** Assigning
  `path=` rewrites `PATH`; they are linked. Re-run with a renamed variable, all six arms
  behave as designed.
- *"SonarCloud is a pick"* — non-required, `0.0% Coverage on New Code` on a diff that is
  shell, make, deletions and a pure move; already the queue row `sonarcloud-new-code-gate-red`.
  It IS, however, a sixth thing the red was hiding, and that is recorded above.
- *"`go build ./...` rc=1 is ours"* — no, red at base on `cmd/wasm` (no native `main`); the
  judge reproduced it on an ephemeral `origin/dev` worktree.
- *"`make test-launchd-drivers` fails on my branch"* — no, it needs `/usr/sbin` on `PATH` for
  `lsof`. Two arms differing only in `PATH`: rc=1 without, **rc=0 with, 43 probe arms**. This
  is iteration 317's finding, still true and still costing a measurement each time.

**Routing evidence**: controller `claude:claude-opus-5` (session). **Designer not spawned**,
rotation pointer untouched at `claude:claude-fable-5` — a dev-red fix-forward is not a queued
design item and there was no doc to author; recorded as a routing judgement, not a probe
failure. **Planner not spawned** — no design doc exists for a red, so the "doc but no plan"
condition never arose and `derive-planner-lane.sh` was not consulted. **Executor
`codex:gpt-5.6-sol`** via the cross-provider recipe (a `provider:model` value must NOT use the
Agent tool), probe rc=0, one sandboxed 30-min-capped run, directive delivery asserted at
3,548B, stdin closed, no git writes; it returned rc=0, touched exactly the four authorised
files, and **correctly self-labelled its own `go test` rc=1 `UNINFORMATIVE UNDER SANDBOX`** (a
loopback bind denial in `TestHub_WebSocketIntegration`) — so every gate was re-run by the
controller outside the sandbox before any verdict was banked. **Evaluator `sonnet`** via the
Agent tool in its **OWN** worktree; generator≠judge holds against the codex executor.
**FLAGGED**: the gofmt, split and changelog commits are Anthropic-authored and the judge is
Anthropic — same provider, different model. The judge named that exposure itself, said which
claims it was least confident it had escaped (the changelog's causal framing), and that is
precisely the finding I acted on by rewriting the entry from three defects to five. Metered
**$0.00** of the $5 ceiling; every lane a quota bucket; no quorum round.

**Human channel**: **0 directives** on `#972` since watermark `2026-09-02T07:17:34Z` (22
comments). Ledger valid at **54 rows, 0 OPEN** — nothing is waiting on Mark. No rotation owed
(#972 created `05:56:11Z` = 07:56 CEST Monday, after the 07:00-local boundary; 22 < 80); no
weekly sweep owed. Cross-mission: replied to `mission-docs` with the corrected attribution and
the masking finding, body read back and confirmed intact; their handoff acked.

**Friction / process**
- **`changelogs/v0.32-current.md` is the cross-mission collision surface for the fifth
  consecutive iteration.** Two conflicts this time, one of them against *my own* earlier
  commit after a union resolution shifted its context. Both resolved as unions with every
  section heading asserted present and a fresh negative control absent.
- **`mergeable` read FIRST at every push** (the iteration-198 rule) and caught a real
  `CONFLICTING` immediately, so no dropped-event lever was reached for. That rule keeps paying.
- **A `git commit --amend --only` folded a changelog edit into a commit whose message did not
  mention it.** Caught by reading `git show --name-only` afterwards and split back out. Same
  class as `ce05af862`'s own lesson, one layer up: check the resulting file list, not the
  paths you passed.

- **The Gate-5 skill-edit contract now collides with a CI ratchet, and every future iteration
  meets this wall.** My first attempt APPENDED the new rule to `SKILL.md`;
  `make check-context-docs` refused it — *"grew to 2905 lines (baseline 2854) — baselined docs
  may shrink, never grow. Split before you append."* Gate 5 says "edit the offending SKILL.md",
  one edit per iteration, and says nothing about where the lines go. The convention's own answer
  is *"write the pointer, not the payload"*, so this iteration followed it: a new on-demand
  `resources/ci-health.md` carries the new rule **and** the existing CI-provider-outage war
  story, with a 7-line pointer left in Gate 1, and `SKILL.md` went **2854 → 2819** — it shrank.
  Proven a MOVE and not a rewrite (block present verbatim in the new file, absent from
  `SKILL.md`, negative control not matching). **Worth saying plainly: the ratchet is right and
  Gate 5 is the one that is now under-specified.** A skill whose gate forbids growth needs its
  Gate-5 instruction to say "relocate a block of equal or greater size, or write to
  `resources/`" — otherwise the next controller spends a slot rediscovering this, or worse,
  baselines its way around the gate.

**Next**: `m-ci-serial-gate-masking` — the job *shape* that hid five defects behind one, and
the only item here that prevents a recurrence rather than cleaning one up. It wants a design
doc: the trade-off is CI minutes against observability, and the answer changes which gates are
"required". `m-spawn-pin-enforcement` remains the queue head and is design-approved.

## 324 — 2026-09-03 — The loop gated its own spawn path, and the judge blocked on the branch I had only asked about [HARNESS]

**Pick**: the queue head, `m-spawn-pin-enforcement` — after first landing iteration 323's orphaned
record (PR #1035, green and MERGEABLE, no report ever posted; the fourth of eight slots to die holding
finished work).

**Progress**: N = **12** design docs remaining before v1.0.0, **unmoved** — HARNESS iteration, goal
moved by 0 in those words. (N went 10 → 12 by Mark's attended D-53 ruling, acknowledged this iteration.)

**Outcome**: LANDED (M1+M2 of 4) · [HARNESS] · evaluator **round 1 FAIL 66/100 (one BLOCKING) →
round 2 PASS 92/100, zero blocking** · PR [#1038](https://github.com/sunholo-data/ailang/pull/1038)
→ squash [`70e453060`](https://github.com/sunholo-data/ailang/commit/70e453060) · Gate 3b `test`
and CI `success` SHA-addressed; SonarCloud inherited (walked back to the parent).

**What landed.** `tools/launchd/resolve-role-spawn.sh` (one line per role in the
`derive-planner-lane.sh` convention; the planner role CONSUMES that script and copies its reason
token through — measured 0/0/0/5 references, so the two compose rather than overlap) and
`tools/launchd/spawn-pin-hook.sh`, a PreToolUse hook wired as a SECOND `Agent|Task` entry. While
`MISSION_CONTROL_ACTIVE=1`: role by explicit first-line `MISSION-ROLE:` token only (prose
inference measured to false-positive), `subagent_type: Explore` the one read-only exception, a
`provider:model` pin denied on ANY alias, unset pin / unknown role / unparsable payload / evaluator
alias == executor's resolved model all denied, every decision logged (7 tab fields). Marker absent:
the hook prints NOTHING. Routing suite 36→45 arms; hook suite 17; `launchd drivers (bash 3.2)`
green; end-to-end through the repo's real settings file in a nested session.

**Two quorum rounds, both 3/3 reject, and the loop measured rather than forwarded.** Round 1 blocked
on an unverified premise (hook→env inheritance), a Conflict Surface ask and stale line cites; I ran
the spike myself (env inherited, `SPIKE_MARKER` control, deny honoured under `bypassPermissions`),
counted the overlap, re-cited by text, then routed the revision to the rotation designer (pi
deepseek, 61 s). Round 2 localised on ONE surface — evasion by omitting the skill name — with a
platform premise beside it. Two more measurements (both overlapping hooks fire, deny wins, second
alias; the real docs-9/10 prompts are NOT on disk) and the reviewers' own fixes applied under the
ratified narrow-refinement carve-out. Note the design ended at round 3 text without a third quorum:
that is the carve-out working as ratified, and the round count is data about scoping.

**The judge earned its slot on my own question.** I handed it F1 as an open question ("does an
explicit `allow` on the marker-absent branch change attended behaviour?"). It upgraded it to
BLOCKING with the right frame: nothing exports the marker yet, so that branch is the ONLY one that
fires today, on 100 % of Agent/Task calls in every attended session loading this repo's settings.
Fixed by printing no decision; drilled (re-adding the allow kills 7ctl alone). It also found arm L
checking 2 of 7 log fields — a role/pin argument swap survived — and a plan table row that
under-counted a kill set. Both fixed and drilled.

**Ruled out / corrected**
- "Layer-3 exports go beside the role exports" (design) — WRONG: the codex/pi loops rewrite
  `MISSION_<ROLE>_MODEL` in place at driver lines 722/770/779; the export anchor is before the
  `roles:` log line at 1003. Planner finding, verified by me.
- "line 904 is per-fire degradation" (design) — WRONG: it is the one-shot executor override.
- "`make test-launchd-drivers` picks up a sibling suite" — WRONG: each script is named; arm W guards
  the wiring.
- "deleting the Explore exception kills 7a alone" (plan) — WRONG: 7a AND L. "dropping `-e` from the
  payload gate kills 8n alone" (my own row) — WRONG: 8e AND 8n, `printf '' | jq .` exits 0.
- "the docs-9/docs-10 Agent prompts can be replayed" (reviewer ask) — NOT POSSIBLE: 0 captured in
  three mission logs and 0 skill names in either PR body; representative replay used instead.
- The eval-suite "0/23 passed" inbox message — 23 benchmarks in 0.02 s is a lane failure, not a
  regression; not the pick.

**Routing evidence**: controller `claude:claude-fable-5-1` (driver: opus probe timed out ×2, then
fable ok). Designer **`pi:ollama/deepseek-v4-flash:0731-cloud`** (rotation next after
`claude:claude-fable-5`; pointer advanced; `mission_pi_run.sh` verdict `ok`, 61 s, 12 tool calls,
1 file). Planner **opus** via Agent tool — `derive-planner-lane.sh` emitted
`opus fail-closed:path-not-in-codex-allowlist` (used verbatim; 592 s, 26 tool calls). Executor
**`pi:ollama/deepseek-v4-flash:0731-cloud`** (driver fallback, codex probe rc=1 `404` at the
chatgpt backend; pi probe rc=0; two runs `ok` 488 s / 712 s, no git writes, snapshots faithful) —
the DeepSeek promotion count is now **2 consecutive `ok` with non-empty diffs in one iteration**;
the promotion rule says "two consecutive real sprint executions", which these are. Evaluator
**sonnet** (Agent tool, own worktree; generator≠judge holds against pi/DeepSeek), rounds 1 and 2,
1052 s + 360 s. Quorum reviewers `gpt5-6-sol`/`gemini-3-1-pro`/`oc-glm-5-2` all present both rounds.
**metered=$0.10** (round 1 $0.0405, round 2 $0.0605); pi Ollama Cloud flat-rate; Anthropic quota
buckets fable/opus/sonnet. The new resolver, run on this iteration's real env, reads
`planner: agent-tool opus fail-closed:path-not-in-codex-allowlist` · `evaluator: agent-tool sonnet`
· `executor: recipe pi:ollama/deepseek-v4-flash:0731-cloud` · `designer: recipe
claude:claude-fable-5-1` — which is what actually ran, except the designer, which the rotation
file (not the env seed) sent to deepseek.

**Friction (one instance, recorded for the ≥2 bar)**: `scripts/mission_pi_run.sh` invokes pi
WITHOUT the `-e sandbox/index.ts -e worktree-fence.ts` extensions the pi recipe calls mandatory;
`~/.pi/extensions/` holds only the policy JSON, so designer and executor ran unfenced. The post-hoc
main-checkout check held (7 dirty files before and after, none mine). Two rules disagree; the next
instance is the skill-edit trigger.

**Next**: `m-spawn-pin-enforcement` **M3** (driver exports `MISSION_CONTROL_ACTIVE=1`,
`MISSION_<ROLE>_RESOLVED/PATH` immediately before the `roles:` log line; `scripts/*` appended to
the versioned docs allowlist) and **M4** (the Gate-3 spawn-pattern paragraph) — M3 ARMS the hook
fleet-wide, so its landing note must say that a controller running the stale main-checkout skill
will be denied until it adds the token, with the denial reason naming the fix.

## 325 — 2026-09-03 — The hook is ARMED, and the acceptance criterion meant to prove it was dead on arrival [HARNESS]

**Pick**: the queue head, `m-spawn-pin-enforcement` — M3 + M4, the two milestones that ARM the
Layer-1/Layer-2 machinery iteration 324 landed inert.

**Progress**: N = **12** design docs remaining before v1.0.0, **goal unmoved** — HARNESS iteration.
The sprint itself is now 4/4 milestones complete.

**Outcome**: LANDED (M3+M4 of 4 — sprint complete) · [HARNESS] · evaluator **PASS 94/100, zero
blocking, round 1** · commits `11aff5819` (M3) + `e21c3f1bd` (M4).

**What landed.** M3: the driver now publishes the plan it VERIFIED rather than the one it declared —
`export MISSION_CONTROL_ACTIVE=1` plus `MISSION_<ROLE>_RESOLVED` / `MISSION_<ROLE>_PATH` for all four
roles, inserted immediately before the `=== mission iteration starting` log line, i.e. AFTER the
codex lane loop (`:722`), the pi loop (`:770`/`:779`) and the one-shot override (`:904`) have all
finished rewriting `MISSION_<ROLE>_MODEL` in place. Plus `|scripts/*` appended to the versioned
docs-mission planner allowlist (the docs-10 / PR #1010 cost). M4: the Gate-3 spawn paragraph now
names `resolve-role-spawn.sh` and requires every role prompt to open with `MISSION-ROLE: <role>`,
with `subagent_type: Explore` as the one machine-readable read-only exception — +13/−3, one
paragraph, the fable capability paragraph beside it untouched. Routing suite 45 → **51** arms
(D1, D2, 12, 12-control, S1, S2); hook suite 17 unchanged; `launchd drivers (bash 3.2)` green.

**THE FINDING: A3.2 IS A DEAD ACCEPTANCE CRITERION, AND IT IS DEAD UNCONDITIONALLY.** The plan's
A3.2 is `MISSION_PROFILE=v1 MISSION_DRY_RUN=1 bash tools/launchd/mission-control.sh` → rc=0, and it
is the only criterion aimed at M3's production code running. I noticed it returned rc=0 having taken
the overlap-guard yield path, and handed that to the judge as my sharpest open question. The judge
came back with a stronger answer than I had: the `MISSION_DRY_RUN=1` branch `exit 0`s at **line 858**
and the Layer-3 block starts at **line 1008**, so the dry run can never reach it — with or without
the overlap guard, on an idle machine or a busy one. It proved it by running the criterion against
the BASE copy of the script: byte-identical output, rc=0, pre-M3. A criterion that passes identically
whether or not the milestone exists measures nothing. Reproduced first-party (`grep -n 'DRY RUN ok:'`
→ 858 with its `exit 0` on the same line; `grep -n 'export MISSION_CONTROL_ACTIVE=1'` → 1008). **Arm
D2 is the real coverage** and is not vacuous: it extracts the live block by `awk` and asserts the
four variables across a `/usr/bin/env` process boundary — mutation-drilled as a sole killer.

**The plan's own control could not fire, which is why the executor got a corrected one.** A4.4 and
test arm S2 both asserted a grep for ``the Agent tool's `model` enum in this build lists``. That
string is LINE-WRAPPED in SKILL.md (1109 ends `…tool's `model``, 1110 begins `enum in this build
lists`), so a line-oriented grep returns **0 on a healthy file** — the plan records "At base: 1",
which is simply wrong. Measured at base before routing, and the executor was given
`grep -c 'enum in this build lists'` (base 1, unique) instead, with a comment in the arm saying why.
The judge confirmed the base value is 0 against the base commit rather than taking my word for it.

**And the corrected control is narrower than it looks (non-blocking, recorded for the next reader).**
The judge probed S2 with a *different* realistic damage — deleting the fable paragraph's evidentiary
sentence (`model="fable"` was ACCEPTED and ran to completion) rather than the grepped line — and the
suite stayed **51/0**. So S2 pins one line, not the paragraph's substance. That is inherent to every
line-grep arm in this suite rather than a defect of this one, but it means "the fable paragraph
survives" is a weaker guarantee than its arm name implies.

**Ruled out / corrected**
- "A3.2's rc=0 was an overlap-guard artifact" (my own framing to the judge) — WRONG, and the judge
  said so: it is structural, `exit 0` at 858 precedes the block at 1008 on every path.
- "the plan's A4.4 reads 1 at base" (plan) — WRONG: 0, the literal is line-wrapped.
- "the plan's hook suite has 13 arms" (plan §7) — STALE: 17, the extra four (`NS`, `8p`, `8e`, `8n`)
  are iteration 324's judge findings. Reported as the real number, no arm adjusted to match.
- "`scripts/*` widens the allowlist into a traversal hole" (my adversarial ask) — REFUTED by the
  judge: `derive-planner-lane.sh` applies a prefix-independent `/*|~*|*..*` deny before the allowlist
  check; `scripts/../internal/foo.go` reads `opus fail-closed:path-not-in-codex-allowlist`.
- "the M4 skill text promises enforcement the hook may not implement" (the worst defect available
  here) — REFUTED: the judge read `spawn-pin-hook.sh` in full; the unlabelled-spawn deny
  (`fail-closed:role-missing`) and the `Explore` allow are both present as shipped.
- 3 bash-3.2 forbidden-construct grep hits in `mission-control.sh` — PRE-EXISTING at base (all three
  are comments naming the constructs), not introduced.
- SonarCloud red on dev HEAD — INHERITED: `failure` on every walked-back commit that has a Sonar run
  at all; already the tracked queue row, not this iteration's pick.
- `mission-world` iter-152's claim that `make check-no-personal-email` "does not exist in the V1
  Makefile" — **REFUTED for this repo**: it exists at `make/code-health.mk:192` and is wired into the
  `ci:` aggregate at `make/ci.mk:11`. World's grep was scoped to `Makefile` and missed the `make/*.mk`
  includes. The skill's ATTENDED-LEDGER sentence citing it is TRUE here. World's own repo genuinely
  lacks the gate; that is World's row, not ours.
- `mission-world` iter-152's claim that `ailang messages send --type` is MISFILED —
  **CONFIRMED first-party**: `cmd/ailang/messages_send.go:42` binds `--type` to `Category`
  ("Message category") while `:132` hardcodes `MessageType: InboxTypeNotification`. Entered as a
  queue row tagged [WORLD-DEMAND]; it does NOT outrank the queue (a sibling cannot set our
  priorities) and it does not break the approvals channel — Discord routes on `ToInbox`.

**Routing evidence**: controller `claude:claude-opus-5` (session). **Designer: NOT SPAWNED** — the
design doc and its sprint plan both existed on disk (`design_docs/m-spawn-pin-enforcement.md`,
`…-sprint-plan.md`), and Gate 3 routes by artifact state; spawning one would have spent the Fable
diet on a document that was already written and already quorum'd (two rounds, iteration 324).
Rotation pointer untouched. **Planner: NOT SPAWNED** — same reason: the plan covers all four
milestones and M3/M4's sections are fully specified with frozen contracts. **Executor
`codex:gpt-5.6-sol`** via the cross-provider recipe (the resolver's own output for this env is
`recipe codex:gpt-5.6-sol declared:provider-pin`, so the Agent tool is not a valid path for it):
probe rc=0 — the codex lane is UP this fire, against iteration 324's 404 — one sandboxed 30-min-capped
run, zero git writes, `.snap/M3` + `.snap/M4` cumulative and `shasum -c` **4/4 OK** against the
reconstructed commits. **Evaluator `sonnet`** via the Agent tool in its OWN worktree
(`.wt-v1-iter325-eval`, detached at `e21c3f1bd`) — generator≠judge holds (OpenAI executor vs
Anthropic judge), one round, PASS 94/100. Its directive carried standing rule 7's operative half in
its own words, and it returned a complete report rather than an intention to wait. No quorum at pick
(the doc carries iteration 324's artifacts and Mark's attended approval). **metered=$0.00** of the $5
ceiling — codex and sonnet are both quota buckets, and no quorum ran. No GPU, no `rig.lock`.

**Gate 1 health**: running skill byte-identical to origin (`cmp` rc=0 against the RESOLVED
`readlink -f` target, main checkout inode `9992963`; the pin's own copy is `35318787`, a different
file). Pin worktree, main checkout `dev` and `origin/dev` all at `5e860afeb` — zero divergence, the
first clean three-way reading in some time. Main checkout dirty with 7 files, none mine, left alone
(Principle 0).

**Friction (one instance, recorded for the ≥2 bar) — I silenced stderr on my own `git add` and it swallowed a fatal, producing a record commit with no record in it.** The Gate-4 staging command listed the pre-move paths of two files `git mv` had already staged, so `git add` aborted the WHOLE add on `pathspec ... did not match any files` — and I had written `2>/dev/null`, so nothing printed. The commit then succeeded, with the right message, containing only the two renames: charter, log and archive were all still unstaged. Caught by reading `git show --stat` afterwards rather than by any error. This is verification-protocol rule 3 (exit codes through pipes lie) aimed at the commit step, and the tell is exact: **a `2>/dev/null` on a command whose failure mode is "did nothing".** Amended rather than re-committed. Gate 4 already mandates `git diff --stat` before `git add` on the charter; the gap is that it says nothing about reading `git show --stat` *after* the commit, which is the only thing that catches a staging failure.

**Next**: the sprint is complete, so the deliverable is bookkeeping plus the first ARMED fire —
`m-spawn-pin-enforcement` moves to `design_docs/implemented/` WITH its sprint plan, and the next
iteration is the first to run with `MISSION_CONTROL_ACTIVE=1` exported, i.e. the first whose own
Agent spawns are denied without a `MISSION-ROLE:` line. Watch for a controller running the stale
main-checkout skill being denied with a reason that names the fix — that is the design working, not
a regression. Then `m-ci-serial-gate-masking` (one early red hid 45 gates for a day) and
`m1b-nolint-suppression-owed`.

---

## 326 — 2026-09-04 — The gofmt hook only ever guarded one of the ways we write Go, and the publish step guarded none [HARNESS]

**Pick**: NOT the queue head. dev HEAD `646bda1e1` was `lint: failure` with `lint: success` on all
three parents — a new red, and V1 owns `sunholo-data/ailang`, so per Gate 1 it outranks
`m-release-manager-skill-split`.

**Progress**: N = **12** design docs remaining before v1.0.0, **goal unmoved** — HARNESS iteration.

**Outcome**: LANDED · [HARNESS] · evaluator **PASS 86/100, zero blocking, round 1** · commits
[`fdba98d32`](https://github.com/sunholo-data/ailang/commit/fdba98d32) (M1),
[`d2ef77e09`](https://github.com/sunholo-data/ailang/commit/d2ef77e09) (M2),
[`c5227e6d7`](https://github.com/sunholo-data/ailang/commit/c5227e6d7) (M1b).

**What was actually wrong.** Not a careless commit — a hole with a shape. `scripts/hooks/format_go.sh`
runs `gofmt -w`, and `.claude/settings.json:136` wires it as a **PostToolUse hook with matcher
`Edit|Write`**. It therefore fires for the Claude Edit/Write tools and for nothing else: not for a
bash/sed/heredoc edit (which this repo's own bypass-permissions guidance actively prefers), and not
at all for a cross-provider executor, because codex and pi writes are not Claude tool calls.
Downstream, `push_dev_on_stop.sh` — the Stop hook that publishes local dev to origin/dev — had zero
correctness gates. So the formatting guarantee was **tool-shaped**, and the publish step, which is
the one chokepoint every executor passes through, had nothing on it. Measured end to end:
`646bda1e1` committed `00:14:14`, `autopush.log` records `[00:14:29] pushed 1 commit(s)`, `lint`
failed `00:15:07`. Fifteen seconds from commit to a red on the branch four loops build from.

**The second instance arrived mid-iteration, and it is the argument for the fix.** After the executor
finished, origin/dev had moved five commits; rebasing turned `make fmt-check` red again on a
different file — `cmd/ailang/coordinator_approvals_remote.go` from `17a363ca6` (`00:22:14`, unsorted
imports), auto-pushed `[00:27:06]`. Different author, different violation class, thirteen minutes
apart, identical mechanism. Fixed as its own commit (M1b) so the two causes stay separable.

**The gate.** It judges the committed blobs in `origin/dev..dev` — never the working tree, because
the shared checkout routinely holds other sessions' edits (8 of them during this iteration).
Deletions and renames-away are excluded, so removing an unformatted file still pushes. A missing
`gofmt`, or any check that cannot run, refuses **loudly** rather than passing silently (Principle 2).
The hook still always exits 0 and still honours `AILANG_AUTOPUSH=0`. Arms H–M, 18 passed / 0 failed;
I and J are the controls that stop it being a gate that refuses everything.

**Ruled out / corrected.**
- *"The first implementation was fine because its four arms passed."* **REFUTED by probing rather
  than reading.** It compared two command substitutions, and command substitution strips all
  trailing newlines, so violations living only in the trailing bytes were normalised away on both
  sides. Probe with controls in both directions: no-trailing-newline and trailing-blank-lines were
  both `UNFORMATTED` to `gofmt -l` and both **PUSHED**. Fixed with temp files + `cmp -s`; re-probed
  7/7. The judge's drill reverting to the old comparison kills L and M and nothing else.
- *"The judge's zero-byte finding is blocking."* **NARROWED, then ruled non-blocking.** The
  gate/`gofmt -l` disagreement is specific to a **zero-byte** `.go` file; a non-empty unparseable
  file returns rc=2 both ways and is correctly refused. And `make/code-health.mk:19` tests
  `[ -n "$(gofmt -l .)" ]` — stdout only, never gofmt's rc — so **CI's own fmt gate has the same
  blind spot**. The new gate is exactly as strong as the gate it mirrors; it cannot produce a false
  LANDED. Queued, not patched here.
- *"The harness returns rc=0 on 9 failures."* **My own instrument.** That reading came through a
  `| tail` pipe; the harness ends `[ "$fail" -eq 0 ]` and is correct.
- *"SonarCloud went red on my change."* **Inherited** — `failure` on `850f04189` and `ea6e0fbb6`
  before my commits existed, and my diff is five whitespace/import lines plus two shell scripts.

**Routing evidence**: controller `claude:claude-opus-5`. `resolve-role-spawn.sh` run for all four
roles and used verbatim — executor `recipe codex:gpt-5.6-sol declared:provider-pin`, evaluator
`agent-tool sonnet declared:alias-pin`, planner `agent-tool opus fail-closed:no-doc`, designer
`recipe claude:claude-fable-5-1 declared:provider-pin`. **Designer and planner not spawned — routing
call, not omission**: a Gate-1 red fix-forward has no doc to author and no plan to write, so neither
role's condition fires, and a designer run would have spent the Fable diet on a four-blank-line
deletion. Executor codex, probe rc=0, two bounded sandboxed 30-min-capped runs, zero git writes,
snapshots `.snap/M1|M2|M2b`; it self-labelled two results `UNINFORMATIVE UNDER SANDBOX` and I re-ran
every gate outside the sandbox. Evaluator sonnet in its **own** worktree at the landing commit
(generator≠judge holds: codex vs Anthropic). Rotation pointer untouched. metered **$0.00** of $5.

**Gate 1 health**: `lint` NEW-red at `646bda1e1` (control: `success` on ~1/~2/~3); after landing,
`c5227e6d7` reads `total=20 completed=20`, **`lint: success`**, 19/20 green, one inherited SonarCloud.

**FLAGGED**: the executor's sandboxed harness runs wrote four synthetic `[local] pushed …` /
`[local] REFUSED …` rows into the **real** shared `~/.ailang/state/autopush.log`, because the harness
does not override `$HOME` and codex ran with the real one. Every run of mine used a temp HOME. A
later iteration could read those rows as fleet evidence. Queued.

**Next**: `m-autopush-gate-followups` (the five findings below), then back to the queue head
`m-release-manager-skill-split`. The standing SonarCloud red on dev remains unowned.

## 327 — 2026-09-04 — The test written to prove the harness never touches the fleet log is the thing that destroyed it, and the sandbox is why nobody saw [HARNESS]

**Pick**: queue head `m-autopush-gate-followups` — the five non-blocking findings iteration 326's judge
raised about the committed-Go auto-push gate. All five re-confirmed first-party at HEAD before routing
(rule 3b(v)); none was a ghost.

**Progress**: N = **12** design docs remaining before v1.0.0, **goal unmoved** — HARNESS iteration.

**Outcome**: LANDED · [HARNESS] · evaluator **PASS 85/100, round 1, one blocking finding fixed
in-iteration** · PR [#1044](https://github.com/sunholo-data/ailang/pull/1044) → squash
[`da2b6689b`](https://github.com/sunholo-data/ailang/commit/da2b6689b), **20 checks, 0 pending,
0 failures**, all four required contexts green and SonarCloud green on the PR.

**What landed.** Five milestones, one commit each, bisectable: M1 `c717ee8a6` gives the test harness
its own `HOME` (it had **zero** `HOME` references, so every run appended `[local] …` rows to the real
shared `~/.ailang/state/autopush.log`) and repairs all eight `SC2164` sites. M2 `9bf9c5db2` makes both
formatting gates read gofmt's **exit code** rather than only its stdout — `gofmt -l empty.go` is rc=2
with the diagnostic on stderr and stdout empty, `gofmt < empty.go` is rc=0, and `make/code-health.mk`
tested `[ -n "$(gofmt -l .)" ]`. M3 `d08352176` moves the file list to `--name-only -z` +
`read -r -d ''`, because default `core.quotepath` returns `"caf\303\251.go"` for a committed
`café.go` and the hook then classed correctly-formatted code as unformatted. M4 `62d7aad4b` makes the
two earliest guards log `SKIP_ROOT` / `SKIP_NOT_GIT` instead of exiting silently. M5 `2eb6e25e4` wires
a **scoped** ShellCheck gate — an explicit two-file list, because repo-wide shellcheck is rc=1 across
**187** other tracked shell files — with `scripts/test_shellcheck_autopush.sh` as its anti-vacuity
control. Harness: **18 → 26** arms.

**The finding, and it cost real evidence.** M1's test row was specified in the sprint plan as a
*"caller sentinel"*: seed a known line into the caller's real `$HOME/.ailang/state/autopush.log`, then
assert it is unchanged. The executor implemented that literally, `printf … > "$CALLER_LOG"`. **Under
`--sandbox workspace-write` that write is DENIED**, so the executor's own run reported the arm passing.
My first re-run outside the sandbox truncated the real shared fleet log from **92 lines to 1**.
Unrecoverable — `find` over `$HOME` returned no second copy, and the only local snapshots are
`com.apple.os.update-*`. A test written to prove the harness never touches the caller's log was the
thing that touched it. The arm is now read-only (sha + line count on both sides, `absent` a legitimate
reading on each), propagated into **all five** snapshots so no commit on the branch carries the
landmine, and a marker row was written into the truncated log so a later reader cannot mistake one
line for a quiet fleet. The mutation it exists for still fires: dropping `export HOME="$W/home"` takes
it from 3 lines to 17 with a different sha, plus four other arms.

**The generalisation outranks the incident**: a sandboxed executor cannot distinguish *"my destructive
step was denied"* from *"my step was harmless"*, so **any acceptance step that writes outside the
worktree is unverifiable on that lane by construction** — and a guard for a shared artifact must be
specified as an **observation**, never as a seeded write. That is a defect in how the sprint plan
worded an acceptance criterion, not in the executor that implemented it faithfully.

**CI caught what my own sweep did not — rule 3g, on my hands.** `test` and `test-windows` went red on
`TestPackageInstallStepsAreBounded` (`internal/cihygiene`): my ShellCheck install step shells out to a
package mirror with no step-level `timeout-minutes:`. `lint` — including the new shellcheck gate
itself — passed throughout. Fixed at `8547d0dab` with `timeout-minutes: 5`, the value the three other
`apt-get` steps already use, mutation-verified both directions. My pre-push sweep was hand-picked
(fmt-check, shellcheck, the harness, check-referenced-paths, actionlint); the CI job's own command list
was knowable and I did not derive it.

**The judge's blocking finding, reproduced before acting.** M2's and M5's self-tests were added to the
`ci:` aggregate **and nowhere else** — and `ci.yml:228` states the rule in the repo's own words:
*"`make ci` is a LOCAL aggregate: CI never invokes it … Adding a target to `ci:` is necessary and NOT
sufficient."* Measured: **0** invocations of `make ci` across all workflows (control `make fmt-check`
= 1); **9 of 9** pre-existing `test-check-*` self-tests ARE wired as their own steps, the two new ones
the only exceptions. So the controls that prove the empty-Go rejection and the scoped file list would
never have run on a PR — a gate without its control, found inside the sprint that adds the control.
Fixed at `cacd35103`, plus two new rows in `test_shellcheck_autopush.sh` asserting each self-test
reaches CI (mutation-verified: 5/5 → 4/5 with the named row red).

**Ruled out / corrected.**
- **My "3 SC2164 findings at base" was a TRUNCATION, not an error** — the planner measured **8**. My
  baseline command ended in `head -20`. Rule 3a aimed at an instrument's *display width* rather than at
  its emptiness; the fix repairs all eight.
- **My directive's summary table said "base was 20 arms, +6"; the base is 18 (+8)** — the judge caught
  it, and the same table's other row said 18, so I contradicted myself inside one table. The M1
  *boundary* is 20; I conflated boundary with base.
- **My `8547d0dab` commit message says three Go packages read `.github/workflows/`; only
  `internal/cihygiene` does** — my grep was a union over `workflows|make/|Makefile|scripts/hooks` and I
  described it as if it were the first term alone. The fix is unaffected; the stated audit scope was
  overclaimed. Left in history rather than rewritten, recorded here.
- **PR #1041's `lint` red is entirely base-inherited, and I did NOT touch it.** Its head tree has 5
  unformatted files — all `internal/observatory/*` + `coordinator_approvals_remote.go`, exactly what
  iteration 326 fixed — while `origin/dev` is gofmt-clean (control fired: 5 vs 0). A rebase clears it.
  Left alone on attribution grounds: its worktree is `ailang-worktrees/mission-comms-p1`, not the
  `.wt-v1-iterN` mission convention, and `mission-comms-into-the-binary` appears **0** times in V1's,
  motoko's or docs' charters. The motoko-iter-17 rule says an unattributable PR is not mine to rebase.
- **SonarCloud on dev is inherited, again** — `failure` on `c5227e6d7`, `850f04189` and `ea6e0fbb6`
  before this iteration existed. It PASSED on PR #1044.

**Routing evidence**: controller `claude:claude-opus-5`. `resolve-role-spawn.sh` run for all four
roles, output used verbatim: designer `recipe claude:claude-fable-5-1`, planner `agent-tool opus
fail-closed:no-doc`, executor `recipe codex:gpt-5.6-sol declared:provider-pin`, evaluator `agent-tool
sonnet declared:alias-pin`. **Designer NOT spawned and not owed** — a five-item measured punch list
with named files is not a design doc, the Fable diet is for authoring, and the rotation pointer
`pi:ollama/deepseek-v4-flash:0731-cloud` is untouched. **The spawn-pin hook FIRED on the planner and it
was RIGHT**: I spawned the resolver's `opus` answer and got
`deny:provider-pin — planner is pinned to codex:gpt-5.6-sol`. That is `m-spawn-pin-enforcement` working
two iterations after landing; I followed the hook and routed the planner through the codex recipe.
Planner + executor both `codex:gpt-5.6-sol`, probe rc=0, one bounded sandboxed 30-min-capped run each,
zero git writes, `.snap/M1`–`.snap/M5` cumulative; reconstruction faithful by sha256 manifest, **7 of 8
files byte-identical**, `ci.yml` differing only by one documented controller edit. Evaluator `sonnet` in
its OWN worktree at the landing commit — generator≠judge holds (OpenAI executor, Anthropic judge). It
drilled **8 of 8** named mutations as exact sole killers with `shasum` restore each time, proved the
read-only arm non-destructive across five caller-log states (present / absent / unreadable / a
directory / a symlink), and confirmed by `git log -p` over all six commits that the destructive form
appears in **no** commit's code. `metered=$0.00` of the `$5` ceiling — codex and sonnet are quota
buckets, no quorum ran, no GPU, no `rig.lock`.

**Gate 1 health**: dev HEAD `2b5750ad9` checks=16, one NOT-GREEN = SonarCloud, inherited. Skill
byte-identical to origin (`cmp` rc=0 against the RESOLVED `readlink -f` target; pin inode `67083825` vs
main `67058129`, a different file). Ledger valid, 54 rows, **0 OPEN**. 0 directives on #972. No
rotation and no weekly sweep owed. 16 unread inbox rows triaged by sender, none directive-class, none
acked — including `mission-world`'s **D-WORLD-31 approval row, open for Mark since 2026-09-03T15:12Z**,
which is not V1's to answer or to ack.

**Next**: `m-release-manager-skill-split` (standing queue head — the 18-image walkthrough out of
`release-manager/SKILL.md`, and the `check-context-docs` ratchet back down 625 → 596), then
`m-gate-wiring-classifier-prefix-blind` (the systemic half of this iteration's blocking finding), then
`m-acceptance-criterion-green-at-base`.

## 328 — 2026-09-05 — The compile cache verifies the receipt and never the goods, so a correct source file does not describe the program that runs [PRODUCT]

**Pick.** NOT the queue head. Two downstream reports from `email-parse` sat unread in the canonical
cloud inbox; one was a correctness bug, ghost-disciplined at HEAD, confirmed, and a confirmed
correctness defect in a shipped surface outranks the queue. `m-gate-wiring-classifier-prefix-blind`
stays [NEXT].

**Progress.** N = **13** design docs remaining before v1.0.0 (was 12, **+1**): a new clause-2
soundness doc entered the count. The unit going up is the unit working.

**What was confirmed, all first-party at HEAD and all reproduced independently by the judge.**
The compile cache has TWO keyspaces and verifies only one.

- The **manifest** is content-keyed and IS checked (`pipeline_module.go:276-277` →
  `cache_store.go:85-91`). That half is correct and was never the bug.
- The **artifact blobs that actually execute** live in a path-derived directory and are loaded with
  **no verification of any kind**: `LoadArtifacts(moduleID string)` (`cache_store.go:185`) receives
  no key, and `grep -n "CacheKey" cache_store.go` returns exactly two lines — the struct field and
  the manifest comparison — neither in the artifact path (known-positive control `modDir` = 11).
- The two writes are neither ordered nor checked: `pipeline_module.go:369` writes the manifest
  entry unconditionally, then `:377` does `_ = cacheStore.StoreArtifacts(...)` with the error
  **discarded**. Any artifact-write failure advances the manifest to the new source's key while the
  blobs stay from the previous compile; every later run then passes the lookup, loads the old blobs
  and skips compilation.

**The end-to-end reproduction, which is the whole iteration.** Source on disk says `99`; the manifest
`cache_key` is the CORRECT key for the `99` source; the blobs are from the `42` compile;
`ailang run` prints **42**, with no diagnostic. Negative control: ordinary edit-and-re-run
invalidation works correctly (42 → 99), so this is not "invalidation is broken" — it is
"the artifacts that execute are never checked against the key that authorises them". Script banked at
`/tmp/iter328_repro_defectA.sh`; the judge re-ran it AND re-derived the mechanism from the call site
rather than trusting the script, explicitly checking it was not rigged.

This explains every one of the reporter's four elimination results, including the two that look like
exculpation: a byte-identical copy at a NEW path works because it is a new module id with no entry,
and `rm -rf` of that one directory works because it forces `loadErr != nil`.

**Second defect, independent: the source read is a silent fallback.** `pipeline_module.go:269`
swallows its `os.ReadFile` error, so a failed read leaves `sourceContent = ""` and the key collapses
to f(cacheKeyVersion, commit, depDigests) — the module's own source contributes nothing and later
edits are invisible. Measured: `ModuleCacheKey(commit, "", {})` =
`b5149f5d2d7eac93707cf159b94ccdcc9f97b8d2960fe843a7eeb20c3e6f8136`, and **both** `std/option` and
`std/result` carry that byte-identical key in a manifest generated at HEAD while their
`iface_digest`s differ. Cause: `loader.go:203` gives embedded stdlib modules the synthetic path
`<embedded>/std/<file>`. The judge reproduced this from scratch with an isolated binary that forces
the embedded-fallback branch — a stronger construction than the controller's.
**The comment three lines above the swallow records that this class was already found and fixed once**
(`mod.Path` vs `mod.File.Path`). The path bug was fixed; the silent fallback that let it become a
correctness bug was left in place. That is this loop's own *guard the helper, miss the call site*,
in the compiler.

**Third: `Clear()` never deletes `modules/`** (`cache_store.go:109-115`), so the documented remedy
for a suspected cache problem leaves every stale artifact on disk.

**Routing.** Resolver run for all four roles, output used verbatim.
Designer `codex:gpt-6-astra` (probe rc=0) — the **first real astra designer run**. Two bounded
sandboxed runs, zero git writes: the initial authoring run, then the ONE protocol-mandated revision.
**The first authoring run was killed and restarted by the controller after ~4 minutes**, because a
read-only reality-check agent returned a second, more severe mechanism (the unverified artifact load)
than the directive had described — a doc scoped only to the swallowed read would not have closed the
user's report. Restarting cost 4 minutes; a revision round would have cost a full run.
Planner and executor **NOT spawned, and that is the routing call rather than an omission**: the doc
is BLOCKED, so there is no approved design to plan or execute, and planning a blocked doc is work on
an unapproved design. Evaluator `sonnet` in its OWN worktree — generator≠judge holds (OpenAI author,
Anthropic judge).

**Quorum: BLOCKED after one revision and one re-quorum, so the doc PARKS.** Round 1 PROCEED — but
with `gpt5-6-sol` **ABSENT on budget**, the self-selecting degrade this skill warns about. Re-running
that reviewer alone at a raised cap returned **reject**, and the objection was real: the doc claimed
axiom A5 "Bounded Verification" on a fixed file COUNT (line 20) while deferring size limits to
out-of-scope (line 457) — a measured internal contradiction, handed to the designer as a measurement
rather than as an objection. The revision answered it well (16 MiB/64 KiB/32 MiB ceilings justified
against a 27-module survey, TOCTOU-safe `Stat` + `io.ReadAll(io.LimitReader(f, limit+1))` with the
extra byte as an overflow sentinel, over-limit always a MISS never a compile error).
Round 2 still BLOCKED: `gpt5-6-sol` rejected a second time on the SAME A5 surface (hashes prove
consistency, not compiler provenance; gob decode work unbounded), and `oc-glm-5-2` rejected on the
doc's self-admitted V54 row. `gemini-3-1-pro` passed both rounds. **PARK is correct and the judge
independently agreed**: objection A needs controller judgement about the threat model and carries no
verbatim fix, which forecloses the narrow-refinement carve-out regardless of objection B.

**Reviewer independence, and a policy that changed underneath this iteration.** `c4e692918` landed
**four minutes after this iteration's Gate-0 stamp** and moved the quorum's OpenAI seat to astra —
the model the rotation had just handed this iteration as its DESIGNER. The controller had already
pinned `--reviewers gpt5-6-sol,gemini-3-1-pro,oc-glm-5-2` on round 2 for exactly that reason,
independently, before reading the commit; both quorum artifacts confirm astra reviewed nothing.
**The rulebook changed mid-flight** — the running skill was byte-identical to origin at Gate 1 and is
byte-identical to a DIFFERENT origin now — and the same commit also restored fable as a rotation
entry, so under the new list this iteration's next-entry would have been fable rather than astra.
The pointer records what actually ran (`codex:gpt-6-astra`); fable simply comes round next cycle.

**The judge corrected the controller twice, and both stand (Ruled out).**
1. The controller wrote that `LoadArtifacts` performs "**5** unbounded `os.ReadFile` calls and then
   `gob.Decode`". It performs **4**, only **2** of them gob-decoded; the 5th grep hit is an unrelated
   JSON manifest read in `load()`. A whole-file grep count quoted as per-function behaviour — rule
   3b's scope error, in the controller's own evidence for a routing decision.
2. The controller's rebuttal of objection B restated the doc's own self-admittedly incomplete V54
   row (flag NAMES present in source text) as though it settled CLI acceptance. The judge actually
   invoked the binary — `serve-api --help` plus a live MCP `initialize` RPC — and settled it. The
   flags are real and work; the controller's check was weaker than the claim it supported, and
   closing it would have taken under a minute.
The judge also **weakened objection A** on its own initiative by reading `encoding/gob`'s decoder:
it does cap preallocation against remaining input, so a 16 MiB-capped blob cannot be amplified
arbitrarily. And it sharpened the other side: a poisoned cache blob is decoded on every routine
`ailang run` **without the victim knowingly compiling anything**, unlike a malicious `.ail` file.
That asymmetry is the strongest argument for the objection and neither the controller nor the doc
had made it.

**Ruled out.** "Invalidation is broken" — REFUTED, ordinary edit-and-re-run works (42 → 99).
"The degenerate stdlib key is itself the user's bug" — it is a real second defect but does NOT
produce the reporter's frozen-artifact-mtime signature; defect A does.
"Objection A is a defect this doc introduces" — REFUTED and measured: at HEAD `LoadArtifacts`
already reads unbounded and gob-decodes with **0** byte ceilings and **0** hash checks anywhere in
the file, so the doc strictly reduces that surface.

**Landed.** Issue [#1046](https://github.com/sunholo-data/ailang/issues/1046) — the confirmed defect,
public, with the workaround, filed independently of the doc's fate; body asserted present after
posting. Reply to the reporter on the canonical cloud inbox, body asserted after sending. The design
doc itself lands under `design_docs/planned/v0_35_2/` tagged PARKED with both quorum artifacts, so
the work is not lost and Mark's decision has something to act on. Retargeted from `v0_35_1` because
v0.35.1 shipped mid-iteration.

**Next.** `D-55` is the only thing between this doc and a sprint. If Mark rules the
accidental-corruption threat model sufficient, the doc goes straight to the planner next iteration
with no further design work. If not, the doc needs an adversarial-decode section and a third round.
Then `m-gate-wiring-classifier-prefix-blind`.

## 329 — 2026-09-05 — The parked compile-cache fix unparked itself on its own pre-registered default, and the acceptance gate it will be judged by was proven non-vacuous before a line of it was planned [ADMIN]

**Pick.** The queue head, `m-compile-cache-unverified-artifacts` — [PARKED] since iteration 328 on
ledger row `D-55`. The row's resume predicate is not a human answer but a **pre-registered default**:
*"(a), applied at the next iteration, and recorded as a controller routing call rather than as a
ruling."* This is that next iteration and `D-55` is still unanswered, so the default fired. The item
unparked and routed straight to the planner with no further design work, exactly as the row predicted.

**`D-55` REMAINS OPEN, and that is deliberate, not an oversight.** The loop may not resolve a ledger
row on its own behalf (Gate 0's decision-recording contract, rule (c)), and a default is not a
ruling. So the scope decision was applied and the question stays live and answerable: a later ruling
of (b) or (c) supersedes this sprint's scope and the plan must then be revised. Both the plan and the
report say so in those words. The first draft of the plan said *"the mission has resolved D-55 with
default option (a)"* — the controller corrected that wording before the judge saw it, because a plan
asserting a ruling that was never made would launder a controller routing call into a human decision,
which is precisely what the contract exists to prevent.

**Progress.** N = **13** design docs remaining before v1.0.0 (was 13, **±0**). **Goal unmoved.** By
the unit's own definition sprint plans are not design docs and never count, so an iteration that
plans rather than lands cannot move N — the doc leaves the count when it LANDS, is RULED OUT, or is
re-scored off the bar. What moved is the doc's *state*: from PARKED-on-a-human-decision to
[IN-SPRINT] with a verified 4-day plan, which is the step immediately before the count can fall.

**What was verified first-party, and why it is the substance of this iteration.**

The planner's most load-bearing claim was that the design doc's executable acceptance gate fails at
baseline *for the right reason*. That claim was made from inside a `workspace-write` sandbox, and an
in-sandbox gate verdict is not evidence. The controller re-ran it **outside the sandbox**, in a clean
worktree at `137842bfd`:

- M1 returns `go returncode=0` with **zero** selected passes. That zero-with-success is Go's vacuous
  zero-selected-tests result — the failure mode the gate exists to reject — and the gate's
  `assert set(names) <= passed` rejects it correctly. It fails on **missing test names**, not on a
  compile error, a wrong package path, a python fault or a timeout, any one of which would have made
  every milestone's acceptance vacuous.
- Positive control on the same mechanism: run against an existing test
  (`TestAliasPolyE2E_RecordSingleModule`, `./internal/pipeline`) it reports
  `passed=['TestAliasPolyE2E_RecordSingleModule']`. The instrument can see a pass, so the zero above
  is a fact about the tests rather than about the harness.

This is the check that decides whether the next four days of work can be graded at all, and it is the
one thing a plan cannot be trusted to self-report.

**The judge's two findings, both reproduced before being acted on.**

Evaluator `sonnet` returned **PASS 93/100, zero blocking**, having independently re-run the baseline
gate, rebuilt a binary at `137842bfd` and reproduced the stale-execution defect end to end, and run
**two live mutation drills** (T1 module-ID/key authorization; T9 `Clear()` error propagation). The T9
drill is the more interesting result: with the mutant applied, the *entire existing suite* including
`TestCacheStore_Clear` stayed green, and only a T9-shaped drill test caught it — so the plan's claim
that the named test, not the existing suite, is the killer holds.

- **F1 (non-blocking, CONFIRMED, fixed).** `cmd/ailang/serve_api_mcp_surface_test.go` receives
  fixtures from **two** milestones (T10 in M3, T12 in M4) and had no size contingency, while the two
  production files got a measured one. Reproduced: `make check-file-sizes` globs
  `find internal cmd -name "*.go"` (`make/code-health.mk:167`) with the ceiling at `:169` and no
  `_test.go` exclusion, and the file is **140** lines today. Real, and cheap: the plan now carries an
  explicit contingency (split the M4 fixtures into a sibling test file in the same package past 650
  lines; re-run the gate at the M3 *and* M4 boundaries, not only at the end).
- **F2 (non-blocking, fixed).** The doc's M3 section says that commit is independent of the API
  diagnostic work; the plan did not restate it. Now restated, with the reason the omission was
  harmless — the shared test file is a *location*, not a dependency.

**Ruled out.**

- *"The planner's sandbox-viability claim can be banked."* Not banked. The planner reports nothing is
  sandbox-blind, specifically that M4's fresh-binary bounded MCP stdio probe ran fine under
  `workspace-write`. The judge read `TestServeAPI_MCPToolSurface` and `buildAilang(t)` in full and
  found no network, socket or PATH-binary dependency, which is corroboration from a second reader —
  but it remains an executor-time confirmation, not a controller measurement, and is recorded as such.
- *"A quorum verdict of `proceed` in round 1 could have been banked."* It could not, and iteration 328
  was right not to. Read with the corrected paths, round 1 reads `verdict: proceed` with
  `absent_reviewers: [{model: gpt5-6-sol, reason: budget}]` — a pass with a named hole, and the absent
  reviewer is the one that rejected twice when restored in round 2. The re-run is what produced `D-55`
  at all.
- *"The resolver's routing line can be followed."* It could not. See below.

**Routing evidence.**

| Role | Configured | Resolver said | ACTUAL | Note |
|---|---|---|---|---|
| Controller | `claude:claude-opus-5` | — | opus (session) | quota bucket |
| Designer | rotation (`claude:claude-fable-5-1` next) | `recipe claude:claude-fable-5-1` | **not spawned** | doc already exists; Fable budget unspent, rotation pointer NOT advanced |
| Planner | `codex:gpt-5.6-sol` | `agent-tool opus fail-closed:planner-lane-field-missing` | **`codex:gpt-5.6-sol`** | resolver/hook DISAGREED — see below |
| Executor | `codex:gpt-5.6-sol` | `recipe codex:gpt-5.6-sol` | **not spawned** | deliverable is a plan; execution is the next iteration |
| Evaluator | `sonnet` | `agent-tool sonnet declared:alias-pin` | **`sonnet`** | generator≠judge holds: codex/OpenAI wrote it, Anthropic judged it |

`metered=$0.00` — every lane this iteration ran on a subscription or quota bucket (opus controller,
codex `gpt-5.6-sol` planner, sonnet evaluator). No metered call was made; no quorum round was run,
because the doc's quorum was already complete at two rounds and `D-55` is the disposition of its
outcome, not a third round.

**The routing defect, and it is a second instance.** `tools/launchd/resolve-role-spawn.sh planner`
returned `agent-tool opus fail-closed:planner-lane-field-missing`, and this skill says an `opus`
lane is spawned directly through the Agent tool. The spawn-pin **hook** refused it at the tool
boundary: `deny:provider-pin — planner is pinned to codex:gpt-5.6-sol; Agent-tool alias spawn
refused — use the cross-provider recipe (resolve-role-spawn.sh planner)`. Note the refusal message
names, as its remedy, the very script whose answer it is rejecting. The hook reads
`$MISSION_PLANNER_MODEL` directly; the resolver applies the lane-derivation fail-closed logic on top
of it, and the two disagree whenever that derivation fires. The hook wins because it is the
enforcement boundary, so the planner ran on codex as configured — the right outcome, reached by
being denied rather than by routing. Queue row `m-resolver-hook-disagree-on-docless-pick` (iteration
327) records instance 1 as a disagreement *on a doc-less pick*, and iteration **328 already recorded
instance 2** in that same row, with a doc in hand and the root cause named: `derive-planner-lane.sh`
requires a `planner_lane` field that only **2** design docs in the whole repo carry, so it returns
`opus fail-closed` for essentially every real pick and the hook then denies opus. This iteration is
**instance 3**, reproducing that mechanism exactly (`fail-closed:planner-lane-field-missing`). Three
instances clears both Gate-5 bars — the ≥2 for a skill fix and the ≥3 for a routing-policy change —
so the skill edit was spent here; the durable fix still belongs in the TOOL, as iteration 328 said,
and stays queued.

**The skill edit had to pay for its own space, and that is the more useful half.** The first form of
it appended 38 lines to `SKILL.md` and turned CI **red** — step 40, `Check context docs respect
progressive disclosure`, a **ratchet**: *"Baselined docs may shrink, never grow."* Attribution was
unambiguous rather than assumed: `test` is `completed/success` on the base commit and `failure` on
the PR head, and the diff is docs plus one markdown file. The escape valve is to bump the baseline,
and the baseline file itself names that as the wrong answer — it carries a standing note that
iteration 325 loosened the `release-manager` ratchet 596 → 625 without writing the growth, a debt
still owed under `m-release-manager-skill-split`. So the edit was restructured instead: the new note
AND the 2026-08-20 fable-pin correction both moved into a new linked
`resources/role-spawn-routing.md`, with the operative rules kept inline. `SKILL.md` went **2790 →
2781**, the baseline was ratcheted DOWN to match rather than up, and the gate is green. An iteration
that wanted the space did the burn-down instead of deferring it.

**And the layering was caught overreaching by a guard another sprint had written two days earlier.**
The first restructure moved the whole 2026-08-20 fable-pin correction out, and
`tools/launchd/test_mission_routing.sh` arm **S2** — *"fable capability paragraph survives the
spawn-pattern edit"*, added by `M-SPAWN-PIN-ENFORCEMENT` on 2026-09-03 — went red, because it greps
`SKILL.md` for the literal `enum in this build lists`. The guard is right and its intent is that the
capability MEASUREMENT stays discoverable inline, so the measurement sentence was restored to
`SKILL.md` rather than the guard being widened to accept the resources file: an unattended iteration
does not relax a two-day-old guard to fit its own edit. Reproduced locally on `/bin/bash 3.2.57`
before and after. The suite's one remaining local red (`run_lane fixture arm requires real lsof on
Darwin CI target`) is **pre-existing and environmental** — identical on a pristine tree at
`137842bfd`, and green on the real Darwin runner, which is why the previous head's `launchd drivers`
check reads `success`.

**Next.** Execute the sprint: M1 (2d) → M2 (0.75d) → M3 (0.5d) → M4 (0.75d), one commit per
milestone, each boundary green on that milestone's named tests. The executor lane is
`codex:gpt-5.6-sol` with the pi chain behind it; the judge must be non-codex. Two things to carry
forward: the M4 sandbox-viability claim needs an out-of-sandbox confirmation at execution time, and
if Mark answers `D-55` with (b) or (c) the plan's scope section is the thing to revise first.

## 330 — 2026-09-05 — The compile cache now verifies the artifacts it executes, and the judge found the two guards the plan's own mutation table could not see [PRODUCT]

**Pick.** The queue head, `m-compile-cache-unverified-artifacts`, [IN-SPRINT] since iteration 329
with the resume predicate *"execute M1"*. No design work and no planning work was owed: the doc
landed at iteration 328 and the verified 4-milestone plan at iteration 329, so this iteration is the
first that could put code on disk. `D-55` is still `OPEN` and its default (a) was already applied as
a controller routing call at 329; nothing about that changed here, and M1 has now shipped under it.

**Outcome. LANDED — M1 of 4.** PR [#1051](https://github.com/sunholo-data/ailang/pull/1051) →
squash [`3d7bbfad8`](https://github.com/sunholo-data/ailang/commit/3d7bbfad8), from two commits:
[`9cb3e711b`](https://github.com/sunholo-data/ailang/commit/9cb3e711b) (the milestone) and
[`726dd1866`](https://github.com/sunholo-data/ailang/commit/726dd1866) (the judge's two findings).
Artifact loads now verify a v4 stamp binding the exact module ID, the caller-computed **expected**
cache key and SHA-256 digests for all four payloads; blobs are read once under the design's byte
ceilings and decoded only after every hash passes; publication writes the stamp last and the manifest
entry after the artifacts, and optional-persistence failures are reported on stderr rather than
discarded. Issue [#1046](https://github.com/sunholo-data/ailang/issues/1046) deliberately stays
**OPEN** — M1 is one of four milestones, and the reporter's defect is not fully closed until M4.

**Progress.** N = **13** design docs remaining before v1.0.0 (was 13, **±0**). **Goal unmoved.** A
doc leaves the count when it LANDS, is ruled out, or is re-scored off the bar; this one is one of
four milestones in. What moved is that the count can now fall in three more milestones rather than
four, and that a confirmed, public, user-facing correctness defect is one quarter fixed in `dev`.

**THE SUBSTANCE: THE JUDGE ANCHORED ITS MUTATION SET TO THE DIFF AND FOUND WHAT THE PLAN COULD NOT.**
The plan's M1 table names six mutations, one per named test, and the executor reported all six
killing. The judge independently reproduced all six — including one it had to re-aim, because the
table's stated kill-mapping for T2 is imprecise (deleting the whole `validateArtifactDigests` call is
caught by T1's `extra_digest` subtest, not by T2, since T2's missing-digest subtests are independently
defended by the per-blob hash loop below it). That is a doc nit, not a coverage hole: both paths are
defended and both are tested. But the six mutations are derived from what M1 **fixes**, and rule 3n
says that systematically misses what M1 **ships** — so the directive required the judge to enumerate
from `git show` instead, and that is where the two real findings came from:

- **The write-side aggregate module ceiling had zero coverage.** The read path's aggregate ceiling is
  covered by `stamp_and_aggregate_scopes`; the store path's is not. Replacing
  `remaining := cs.artifactLimits.module - *accepted` in `checkArtifactSize` with an effectively
  infinite budget left the **whole `internal/pipeline` package green**.
- **The module-directory creation guard had zero coverage.** Replacing the `mkdirAll` error check
  with `_ = cs.artifactIO.mkdirAll(...)` — the swallowed-error class this loop keeps finding, and the
  exact class of the *original* bug one layer down — also left the whole package green.

Both were reproduced first-party by the controller before being acted on (rule 3b: a judge's finding
is a claim, and reproducing it before DISMISSING it matters as much as before acting). Both were then
fixed with one arm each, and **each new arm was proven to be the sole killer of the mutation it was
written for**: with the mutation applied the named arm fails and the package goes red; with the arm's
own assertion the only new failure; source restored byte-identical by `shasum` both times. Note the
shape — a milestone whose entire purpose is *"stop swallowing the artifact-write error"* shipped a new
swallowed-error path of its own, and nothing but a diff-anchored enumeration would have seen it.

**Ruled out / corrected.**

- **`go build ./...` is NOT a valid acceptance gate on this host, and it was in my directive's first
  draft.** Baselining it on the pristine tree (rule 3e(a)) returned rc=1: `cmd/wasm` is
  `//go:build js && wasm`, so under darwin it has no `main` and the build fails with
  *"runtime.main_main·f: function main is undeclared in the main package"*. CI builds it correctly
  with `GOOS=js GOARCH=wasm` (`make/build.mk:103`). The narrowed `go build ./internal/... ./cmd/ailang`
  is rc=0 at base and is the form that measures the change. Corrected in the directive **before** the
  executor ran, so no time was spent chasing it — which is the whole point of baselining first.
- **A coordinator codex 401 is NOT this loop's codex lane.** Gate-0 triage surfaced a
  `sprint-planner` task failed at `2026-09-05T11:28Z` with `401 Unauthorized` on
  `wss://api.openai.com/v1/responses`, repeated seven times. That is alarming for an iteration whose
  executor is pinned to `codex:gpt-5.6-sol` — and this loop's own `codex exec` probe returned **rc=0**
  twenty minutes later, and the real 30-minute executor run completed rc=0. Two different auth paths
  (the coordinator's websocket API-key path vs. `codex exec`'s OAuth). Queued as its own row rather
  than allowed to divert the pick.

**Gate 3b: the poll's own numeric floor fired, and it was right.** The first CI poll extracted three
counts with `set -- $res` and reported `INSTRUMENT FAILURE — not a verdict (raw=5 2 0)` on every
round. The raw payload was fine; **zsh does not word-split unquoted parameter expansions**, so `$1`
held the whole string `"5 2 0"` and no count was a number. This is instance 3 of the class iteration
233 recorded (instance 1: a `jq` parse error on a control character; instance 2: iteration 154's
vacuously-green aggregate) and the **first with a shell-semantics mechanism rather than a payload
one**. The remedy iteration 233 landed — *assert every count is a NUMBER before comparing, and print
`INSTRUMENT FAILURE` rather than a verdict* — is exactly what caught it, so the rule is not owed an
edit; it is owed a note that the `set -- $res` shape this file's own iteration-107 war story warns
about fails **silently and differently under zsh**, where it produces one argument instead of an
error. Re-polled with each value read in its own `gh --jq` call: 5 runs, 5 completed, 0 failures,
all four required contexts (`build` · `docs-gate` · `lint` · `test`) `pass`, `mergeStateStatus=CLEAN`.
Merged on an **observed** green, never a predicted one, and never behind auto-merge.

**Verification the controller did NOT delegate.** Every gate was re-run outside the executor's
sandbox: the design's cumulative acceptance gate prints six `M1 PASS` lines here and fails at the
parent with all six names missing and `go returncode=0` — Go's vacuous zero-selected-tests success,
which the gate's `assert set(names) <= passed` correctly rejects, so the gate is non-vacuous in both
directions. Whole-package `go test ./internal/pipeline` `ok` (and `ok` at the parent, so a red would
have been attributable). `internal/loader` and `cmd/ailang` also `ok`. `make fmt-check`,
`make check-file-sizes`, `go vet`, `gofmt -l` clean; `pipeline_module.go` is **776** lines against
the 800 ceiling, which is why the plan created two new files rather than growing it. End-to-end smoke
with a binary built from the branch: cold `42`, warm `42` from cache, `99` after editing the source —
invalidation still correct, and the new verification does not break normal operation.

**Routing evidence.** `resolve-role-spawn.sh` run for all four roles, output used verbatim.
Designer `recipe claude:claude-fable-5-1` and planner both **NOT SPAWNED** — the doc and the plan
both already exist, so there was nothing to author or to plan. That is a routing call, not an
omission: the Fable budget is **unspent** and the rotation pointer stays at `codex:gpt-6-astra`.
Executor `recipe codex:gpt-5.6-sol declared:provider-pin` (probe rc=0; one bounded sandboxed
30-minute-capped run, zero git writes, `.snap/M1/` cumulative snapshot verified **byte-identical to
the worktree for all eight files** before the controller built the commit). Evaluator
`agent-tool sonnet declared:alias-pin`, in its **own** worktree at the exact sprint commit —
generator≠judge holds by vendor as well as by model: OpenAI wrote the code, Anthropic judged it.
`metered=$0.00` of the $5 ceiling; every lane rode a subscription or quota bucket and no quorum round
ran. No GPU, no `rig.lock`. **The planner resolver disagreement reproduces for a fourth time** —
`resolve-role-spawn.sh planner <doc>` still returns `agent-tool opus fail-closed:planner-lane-field-missing`
— but it was measured and not routed on, because no planner was owed; the row stays queued and the
durable fix is still in the TOOL.

**Next.** M2 — loader-owned source identity (0.75 d), then M3 and M4. The plan's cumulative runner is
already written to accept `M2`, `M3` and `M4` as boundary arguments with an otherwise identical body,
so each milestone inherits every earlier milestone's named tests by construction. Two things to carry
forward: M4's sandbox-viability claim still wants an out-of-sandbox confirmation at execution time,
and if `D-55` is answered (b) or (c) the plan's scope section is revised before any further code.

#### Design-quorum review — `design_docs/planned/v0_36_0/m-mission-loop-workbench.md` (2026-09-05T20:07:58Z)

- **Synthesis: BLOCKED** (total $0.0761, 20044 in / 3224 out tok)
- `gpt5-6-sol` → **reject** ($0.0419, 6559/303 tok) — The document lacks a verified conflict-surface analysis for the new registry, renderer, installer, doctor, and CLI. It only discusses the complementary comms mission and asserts “No file overlap”; it does not inventory existing configuration loaders, TOML schemas, launchd install/status tooling, atomic-write helpers, pidfile/locking mechanisms, drift checks, or CLI registration patterns, nor justify creating parallel machinery instead of extending them.
- `gemini-3-1-pro` → **reject** ($0.0163, 6980/198 tok) — The document fatally contradicts itself regarding Axiom A4 (Explicit Authority). The A4 justification explicitly claims that 'launchd reload stays an explicit, confirmed step' and 'The generator writes two well-known paths and nothing else'. However, the Architecture diagram and Example 1 explicitly dictate that the 'ailang mission install' command automatically executes 'bootout + bootstrap + verify' (reloading launchd).
- `oc-glm-5-2` → **reject** ($0.0179, 6505/2723 tok) — The doc asserts 'No file overlap' with M-MISSION-COMMS-INTO-THE-BINARY in Related Documents, but both documents modify `tools/launchd/mission-control.sh` — this one deletes ~60 LOC (boot-offset case arm + reach comment table), and the comms extraction necessarily edits the same file to extract its seam. The claim is unverified and the actual conflict surface on the one shared file is unanalyzed. If both PRs land concurrently, they will collide on the same file with no declared ordering or section partitioning.
- controller (in-session, not an API call) → **pass** — Author-controller. Every premise V1-V10 measured first-party on the rig today, not inferred; V5 is a reproduced live production bug (same fixture, two env files, divergent routing) and the doc's first success criterion is that the proposed doctor reproduces it before anything changes. Strongest objection I hold against my own doc: Phase 3 depends on the world pinned-workdir question, which I surfaced by dry run but did not resolve, so the de-fork estimate is the softest number here.
- Blocking objections (return to author before planning):
  - gpt5-6-sol: The document lacks a verified conflict-surface analysis for the new registry, renderer, installer, doctor, and CLI. It only discusses the complementary comms mission and asserts “No file overlap”; it does not inventory existing configuration loaders, TOML schemas, launchd install/status tooling, atomic-write helpers, pidfile/locking mechanisms, drift checks, or CLI registration patterns, nor justify creating parallel machinery instead of extending them.
  - gemini-3-1-pro: The document fatally contradicts itself regarding Axiom A4 (Explicit Authority). The A4 justification explicitly claims that 'launchd reload stays an explicit, confirmed step' and 'The generator writes two well-known paths and nothing else'. However, the Architecture diagram and Example 1 explicitly dictate that the 'ailang mission install' command automatically executes 'bootout + bootstrap + verify' (reloading launchd).
  - oc-glm-5-2: The doc asserts 'No file overlap' with M-MISSION-COMMS-INTO-THE-BINARY in Related Documents, but both documents modify `tools/launchd/mission-control.sh` — this one deletes ~60 LOC (boot-offset case arm + reach comment table), and the comms extraction necessarily edits the same file to extract its seam. The claim is unverified and the actual conflict surface on the one shared file is unanalyzed. If both PRs land concurrently, they will collide on the same file with no declared ordering or section partitioning.

#### Design-quorum review — `design_docs/planned/v0_36_0/m-mission-loop-workbench.md` (2026-09-05T20:11:58Z)

- **Synthesis: BLOCKED** (total $0.0943, 25672 in / 3782 out tok)
- `gpt5-6-sol` → **reject** ($0.0512, 8400/307 tok) — The `mission reload` design violates the bounded-waits axiom: it promises `bootout + bootstrap + verify` but specifies no deadlines, polling bounds, or timeout failure semantics for launchd lifecycle and readiness verification. A lifecycle command that can wait indefinitely cannot proceed under A5.
- `gemini-3-1-pro` → **reject** ($0.0211, 8924/274 tok) — Architectural flaw violating A4 (Explicit Authority) and falsifying the core safety claim. The doc asserts that `install` 'renders only' and 'the fleet keeps running the OLD config until it runs reload'. However, because the bash driver dynamically sources `~/.config/ailang/mission-<name>.env` on every execution (as affirmed in 'Composability'), overwriting this file in-place during `install` causes the live fleet to pick up the new environment variables on its very next loop interval, silently bypassing `reload`. Config application is therefore not deferred.
- `oc-glm-5-2` → **reject** ($0.0219, 8348/3201 tok) — The doc calls extending `internal/daemon/install.go` 'the single most important finding of the inventory' and claims 'the renderer and installer shrink to an extension of internal/daemon/install.go,' but V12 only verifies that install.go exists with certain exported functions — it does NOT verify that the embedded `plist.tmpl` can render a mission plist. A coordinator-daemon plist and a mission plist have structurally different requirements (StartInterval vs keepalive, mission-specific EnvironmentVariables, ProgramArguments pointing at mission-control.sh with per-mission args, boot-offset). The doc simultaneously lists `internal/mission/render.go` as 250 LOC of new code, contradicting the claim that 'the renderer ... shrinks to an extension.' The actual reuse is limited to the install/uninstall/status lifecycle, not the rendering path, which means the 'extend do not duplicate' decision is overclaimed and the premise driving it is unverified.
- controller (in-session, not an API call) → **pass** — Round 2. All three round-1 objections accepted, none argued. (1) gpt5-6-sol: added an Existing Machinery inventory (V11-V15) — it found internal/daemon/install.go, a working 118-LOC launchd installer the first draft would have duplicated, plus internal/riglock for liveness and TOML already in-tree; renderer/installer are now an extension, not new machinery. (2) gemini-3-1-pro: the A4 contradiction was real; install and reload are now separate verbs and install never touches launchd. (3) oc-glm-5-2: 'no file overlap' was false; measured overlap on mission-control.sh (disjoint line regions), on the ailang mission CLI noun and on internal/mission/, with comms declared to land first. Strongest objection I still hold: Phase 3 depends on the world pinned-workdir question, surfaced by dry run and unresolved, so the de-fork estimate remains the softest number.
- Blocking objections (return to author before planning):
  - gpt5-6-sol: The `mission reload` design violates the bounded-waits axiom: it promises `bootout + bootstrap + verify` but specifies no deadlines, polling bounds, or timeout failure semantics for launchd lifecycle and readiness verification. A lifecycle command that can wait indefinitely cannot proceed under A5.
  - gemini-3-1-pro: Architectural flaw violating A4 (Explicit Authority) and falsifying the core safety claim. The doc asserts that `install` 'renders only' and 'the fleet keeps running the OLD config until it runs reload'. However, because the bash driver dynamically sources `~/.config/ailang/mission-<name>.env` on every execution (as affirmed in 'Composability'), overwriting this file in-place during `install` causes the live fleet to pick up the new environment variables on its very next loop interval, silently bypassing `reload`. Config application is therefore not deferred.
  - oc-glm-5-2: The doc calls extending `internal/daemon/install.go` 'the single most important finding of the inventory' and claims 'the renderer and installer shrink to an extension of internal/daemon/install.go,' but V12 only verifies that install.go exists with certain exported functions — it does NOT verify that the embedded `plist.tmpl` can render a mission plist. A coordinator-daemon plist and a mission plist have structurally different requirements (StartInterval vs keepalive, mission-specific EnvironmentVariables, ProgramArguments pointing at mission-control.sh with per-mission args, boot-offset). The doc simultaneously lists `internal/mission/render.go` as 250 LOC of new code, contradicting the claim that 'the renderer ... shrinks to an extension.' The actual reuse is limited to the install/uninstall/status lifecycle, not the rendering path, which means the 'extend do not duplicate' decision is overclaimed and the premise driving it is unverified.

## 331 — 2026-09-05 — The cache key now describes the bytes the lexer actually parsed, and the one red that stopped it was a CI gate my own gate list never contained [PRODUCT]

**Pick.** The queue head, `m-compile-cache-unverified-artifacts`, [IN-SPRINT] since iteration 329
with the resume predicate *"execute M2"*. No design and no planning work was owed — the doc landed
at 328 and the verified 4-milestone plan at 329 — so the designer and the planner were **not
spawned**. That is a routing call, not an omission: the Fable budget is unspent and the rotation
pointer stays at `codex:gpt-6-astra`. `D-55` is still `OPEN` and its default (a) remains applied as a
controller routing call, exactly as at 329 and 330.

**Outcome. LANDED — M2 of 4.** PR [#1053](https://github.com/sunholo-data/ailang/pull/1053) → squash
[`f5edd569a`](https://github.com/sunholo-data/ailang/commit/f5edd569a), from three commits:
[`41320c8ff`](https://github.com/sunholo-data/ailang/commit/41320c8ff) (the milestone),
[`089ae50f2`](https://github.com/sunholo-data/ailang/commit/089ae50f2) (the judge's finding) and
[`ca3b63085`](https://github.com/sunholo-data/ailang/commit/ca3b63085) (the CI red). The loader
retains the exact bytes it hands the lexer as an immutable `SourceContent *string`; the pipeline
hashes `*mod.SourceContent` and the opportunistic second `os.ReadFile` with its empty-string default
is gone. A module with no snapshot bypasses both cache lookup and publication, emits
`CACHE_SOURCE_UNAVAILABLE`, and is never hashed as `""`. Issue
[#1046](https://github.com/sunholo-data/ailang/issues/1046) deliberately stays **OPEN** — M2 is two
of four.

**Progress.** N = **13** design docs remaining before v1.0.0 (was 13, **±0**). **Goal unmoved.** A
doc leaves the count when it LANDS, is ruled out, or is re-scored off the bar; this one is halfway
through its milestones. What moved is that the confirmed, public, user-facing correctness defect is
now half fixed in `dev`, and that the count can fall in two more milestones rather than three.

**THE SUBSTANCE: BASELINING PROTECTS A GATE LIST FROM BEING RED; IT SAYS NOTHING ABOUT A GATE THE
LIST DOES NOT CONTAIN.** I did the thing false-green (4) asks for — every command I wrote into the
executor directive was run on the pristine base first, and I recorded which were expected-red
(the cumulative M2 gate, failing on the missing test name) and which were expected-green
(both single-test acceptance commands, i.e. regression controls rather than new proof). All of them
passed on the milestone. CI then went red on **`make check-home-isolation`**, which was nowhere on
my list: T7's fixture set `HOME` directly, and `os.UserHomeDir` reads `USERPROFILE` on Windows and
`$home` on plan9, so a bare `HOME` override silently leaves those platforms pointed at the runner's
real profile. The repo has a gate for exactly that and a helper (`testutil.SetHomeDir`) that does it
properly.

That is rule 3g — *your local gate sweep is a hand-picked subset* — and the pairing with 3e(a) is
what is worth recording, because the two rules protect against opposite failures and I had only run
one of them. 3e(a) asks *"is this command already red before I start?"*; 3g asks *"is this the set of
commands that will judge me?"* A perfectly baselined list can still be a short list, and a short
list is invisible: it produces an all-green sweep. The remedy is the one 3g already names and I had
not applied — DERIVE the job's command list rather than remember it. Parsing `.github/workflows/ci.yml`
for the `test` job's `run:` steps yields **57** entries; I then ran the eight the diff could
plausibly break (`check-boundaries`, `check-referenced-paths`, `check-git-exec`,
`check-no-personal-email`, `check-changelog`, `check-context-docs`, `test-coverage-gate`,
`check-golden-drift`) — all green, and I had run none of them either. One CI cycle, ~24 minutes,
bought for a `python3` one-liner I could have run before the executor started.

**THE JUDGE'S FINDING: "BYPASSES PUBLICATION" MEANT *FAILS TO PUBLISH*, NOT *NEVER ATTEMPTS*.**
Evaluator `sonnet`, in its own worktree at the exact sprint commit, **PASS 96/100 round 1, zero
blocking**. It reproduced all four of the plan's named mutations, and then — anchoring its
enumeration to `git show` rather than to the plan's table (rule 3n) — found that deleting
`&& moduleCacheKey != ""` from the publication guard left the **whole `internal/pipeline` package
green**. I reproduced that first-party before acting on it (rule 3b applies to a judge as much as to
any other sub-agent): with the clause removed, the cumulative gate still prints all ten `PASS` lines.
The system was safe anyway, but one layer below where the design describes the invariant — a
nil-source module DOES reach `cacheRuntime.publish`, and only M1's `StoreArtifacts` empty-key
rejection stops it, emitting `CACHE_WRITE_FAILED` on the way out. The design says such a module
"bypasses BOTH cache lookup and publication", which means *never attempts*. One arm now pins that
reading, and it was proven the **sole** killer of the mutation: with the clause removed the whole
package fails on exactly one subtest, with it restored the package is green and `pipeline_module.go`
is byte-identical (`sha256 62693b2f…`). Note the shape — the judge found the gap by asking what the
milestone *ships*, and the four planned mutations all ask what it *fixes*. That is now two
consecutive iterations where rule 3n produced the iteration's only real finding.

**Ruled out / corrected.**

- **A red that is non-required is not thereby inherited.** SonarCloud is `fail` on #1053:
  `new_maintainability_rating` **4** against a threshold of 1, six `go:S3776` cognitive-complexity
  smells, with zero bugs, zero vulnerabilities, zero duplication and **100.0%** coverage on new code.
  My first instinct was the standing-red reading — Gate 1 records a period when Sonar was `failure`
  for six consecutive commits — and the control refutes it: M1's PR `#1051` reads `pass` on the same
  check and `dev`'s own branch gate reads `OK`. So this red is NEW and it is mine. It did not gate
  the merge (`UNSTABLE` is not `BLOCKED`, and all four required contexts passed), and I filed it as
  `m-cachesrc-cognitive-complexity` rather than fixing it, because five of the six smells are on test
  functions and the sixth is `runModuleWithCacheDependencies` at **112** — a function already near
  that ceiling before M2 touched it, which Sonar counts only because its lines changed. Extracting
  its control flow after the judge has passed the code, in the exact function M3 and M4 both re-enter,
  is a worse trade than a row carrying the measurement. The row says so explicitly, because the next
  two milestones will inherit this red and cannot otherwise tell it from one of their own.
- **The `.claude/skills/mission-control/resources/codex-lane-false-greens.md` resource carries
  false-green (5) TWICE**, near-verbatim, ~35 lines apart (`grep -c 'FALSE-GREEN (3) SAYS A GATE
  VERDICT FROM INSIDE THE SANDBOX IS NOT EVIDENCE'` = 2). Almost certainly an artifact of the
  2026-09-04 split out of `SKILL.md`. Not spent as this iteration's skill edit: it is one friction,
  not two, and the Gate-5 bar is ≥2 recorded frictions pointing at the same gap. Recorded here so a
  second sighting clears the bar rather than being rediscovered.
- **A concurrent attended session landed a four-milestone `m-mission-loop-workbench` sprint to `dev`
  during this iteration** (`a9de67fe6`…`6536cfb98`, plus two blocked quorum records appended to this
  very log). Not mine, not a sibling loop's, and no conflict — my record is an append and my code
  touches no file it touches. Noted because a reader of this log will find two quorum blocks between
  iteration 330's entry and this one that belong to neither.

**Routing evidence.** `resolve-role-spawn.sh` run for all four roles and its output used verbatim.
Designer `recipe claude:claude-fable-5-1` and planner both **NOT SPAWNED** — nothing to author, nothing
to plan. Executor `recipe codex:gpt-5.6-sol declared:provider-pin`: probe rc=0, one bounded sandboxed
run under the 30-minute cap, zero git writes, directive delivered with the ≥200-byte assertion and
closed stdin, `.snap/M2/` verified **byte-identical to the worktree for all five files** before I
built the commit. Evaluator `agent-tool sonnet declared:alias-pin`, in its **own** worktree —
generator≠judge holds by vendor as well as by model: OpenAI wrote the code, Anthropic judged it.
`metered=$0.00` of the $5 ceiling; every lane rode a subscription or quota bucket and no quorum round
ran. No GPU, no `rig.lock`. **The planner resolver disagreement reproduces for a fifth time** —
`resolve-role-spawn.sh planner <doc>` returns `agent-tool opus fail-closed:planner-lane-field-missing`
and `derive-planner-lane.sh` agrees — measured but not routed on, since no planner was owed. The row
stays queued and the durable fix is still in the TOOL.

**AND `dev` IS RED — SIX JOBS, NONE OF THEM MINE, FOUND ONLY BECAUSE IT BLOCKED MY OWN RECORD.**
Gate 1 read dev's HEAD as `checks=13, ZERO failures, CI mid-flight` and that was true at `5b73f8dcc`.
While this iteration ran, a **concurrent attended session** landed a four-milestone
`m-mission-loop-workbench` sprint (`a9de67fe6`…`6536cfb98`), and it landed red: `lint`, `docs-gate`,
`docs-build`, `launchd drivers (bash 3.2)`, `test-windows` and `Build windows-latest` all fail. The
attribution is unambiguous and the control is the commit before mine — the identical set is already
red at `6536cfb98`, and workbench M1 was already red on both Windows jobs — while iteration 331's own
PR was green on all five runs and all four required contexts at `ca3b63085`.

I found it only because `lint` is a REQUIRED context, so an inherited red **stranded my own Gate-4
record**. That is worth naming as a property of this loop rather than as an incident: Gate 1's health
check runs ONCE, at the start, and an iteration that does real work is exactly the one during which
the tree can change underneath it. Gate 3b's base-inherited-red rule is written about a PR's own
staleness; nothing points it at a base that goes red *mid-iteration*. The tell here was benign —
a required check failing on a docs-only PR — and it is the only reason the six were seen at all.

Disposition: I fixed the **two required-context** reds inside the record PR, because without them the
record could not land, and said so plainly in the PR and in the commit messages rather than smuggling
production fixes into a docs commit. **`lint`**: one `ineffectual assignment to snap` at
`internal/mission/doctor_test.go:229` — `make lint` 1 issue → **0 issues**. **`docs-gate`**, which
adjudicates `docs-build`: a single broken link at `docs/docs/guides/mission-bootstrap.md:18`, a
relative path escaping the docs tree, the only one of its shape in `docs/docs` (grep 1 → 0), now an
absolute `blob/dev/` URL matching the repo's own precedent. That second one is the member of the set
that actually mattered, and it is worth saying why: `docs-gate` is REQUIRED and fires for any diff
touching a docs-relevant path, so one broken link had blocked **every pull request in the repo**. Its
local verification is partial and I labelled it partial — `make docs-build` clears the design-doc sync
and the stdlib-index check and then dies on `docusaurus: command not found`, because a fresh worktree
has no `node_modules`; the remote gate is the verifier, and the failure it must clear named this exact
link. I did NOT fix the remaining four: `go test ./internal/mission` fails
`TestLive_DoctorReproducesTheMeasuredDivergences` with the negative control firing (identical failure
at unmodified HEAD, so not a side effect of my one line), and the launchd-driver and both Windows reds
are uncharacterised. All six are filed as `ci-red-mission-loop-workbench`, positioned at
the TOP of the queue, since a red `dev` outranks the queue for the mission that owns the repo and V1
owns this one. The row carries the failing STEP per job — read from `actions/jobs/<id>`, because
`check-runs` reports the job and never its steps — and a handoff note that the authoring session may
still be live, so the set must be re-measured before anyone starts.

**Next.** The queue head is now `ci-red-mission-loop-workbench` — a red dev outranks the sprint. After
it: M3 — complete compilation-cache clearing (0.5 d), then M4. The cumulative runner already
accepts `M3` and `M4` as boundary arguments, so each milestone inherits every earlier one's named
tests by construction. Three things to carry forward: M4's sandbox-viability claim still wants an
out-of-sandbox confirmation at execution time; the Sonar row above will still be red when M3 lands
unless it is picked up; and if `D-55` is answered (b) or (c) the plan's scope section is revised
before any further code.

## 332 — 2026-09-05 — Clearing the compile cache now removes the artifacts it authorised, and the one red that blocked the whole repo was an instrument measuring its own checkout [PRODUCT]

**Pick.** NOT the queue head, and the reason is the finding. The head is
`ci-red-mission-loop-workbench` [NEXT, OUTRANKS THE QUEUE], and Gate 2's died-mid-flight sweep found
its entire fix already in flight: PR [#1055](https://github.com/sunholo-data/ailang/pull/1055),
opened **six minutes before this fire**, `MERGEABLE`, five milestones covering every remaining red.
`gh pr list --author sunholo-voight-kampff` returned it under the heading "your own account" —
because every mission on this rig pushes as the same bot — so it was ATTRIBUTED before anything was
done with it, exactly as the shared-repo rule requires. `git worktree list` in **motoko's** clone
carries `.wt-iter36-ci-red` on `sprint/m-mission-workbench-ci-red`; V1's own clone lists **69**
worktrees with **zero** matches; the two sets are disjoint by construction, and the PR body says
"Generated by motoko-mission iteration 36". It is motoko's. Duplicating it is the `#758`/`#759`
collision this rule exists to stop, so the row was HANDED OVER with its resume predicate and the
pick fell through to the [IN-SPRINT] item, `m-compile-cache-unverified-artifacts`, resume predicate
*"execute M3"*.

**Outcome. LANDED — M3 of 4.** PR [#1056](https://github.com/sunholo-data/ailang/pull/1056) → squash
[`d14bd42cc`](https://github.com/sunholo-data/ailang/commit/d14bd42cc), from three commits:
`a92fa666b` (the milestone), `672eb1c0f` (the judge's finding), `275a72c80` (an inherited
required-context red that belongs to nobody's sprint). `CacheStore.Clear()` reset the manifest and
saved it and never touched the blobs — measured at HEAD, `ailang cache compile-clear` reported
`Cleared 4 cached compilation entries` while the whole `modules/` subtree stayed on disk (V11, V25).
It now saves an empty v4 manifest **and** removes `<cs.dir>/modules`, returning a contextual error if
either half fails, so the CLI cannot print its success line over a failed deletion. A missing or
empty subtree is success. Removal rides a new injectable `removeAll` seam on `cacheArtifactIO`
beside the existing `writeManifest` seam, so both failure paths are testable without a global. The
deletion is deliberately narrow: `manifest.json`, the root's siblings, package caches and another
session's `AILANG_CACHE_DIR` override all survive, under both root variants. End-to-end with a
binary built from the branch: 20 blobs after a compile → `Cleared 4 cached compilation entries` →
subtree **removed**, compile-root sentinel **preserved**, manifest entries `0`, re-run still correct.

**Progress.** N = **13** design docs remaining before v1.0.0 (was 13, **±0**). **Goal unmoved.** A
doc leaves the count when it LANDS; this one is three of four milestones in, and issue
[#1046](https://github.com/sunholo-data/ailang/issues/1046) deliberately stays OPEN until M4.

**THE SUBSTANCE: A RATCHET THAT WAS MEASURING ITS OWN CHECKOUT, AND WHOSE OWN ERROR MESSAGE
PRESCRIBED THE WRONG FIX.** `test` — a REQUIRED context — was red on `dev`, which means it was
blocking every pull request in the repository, motoko's `#1055` fix included. Attribution first,
with a negative control on the base's own CI job rather than on the check name: same failing STEP
(11, *Run tests with timeout*), same single failing test, both sides — inherited, not from this
diff. The test is `TestDriverCopiesDoNotMultiply`, landed roughly forty minutes earlier by the
concurrent attended session, and it tells you what to do: *"driver copies FELL to 1 — good, but
lower knownDriverCopies to 1 so the ratchet keeps holding at the new level."* **That prescription is
wrong twice over, and only measurement shows it.** The probe reads SIBLING directories of the
checkout (`../../../ailang-world/tools/launchd/mission-control.sh`, and the same for docs and
motoko), so what it can see is a property of WHERE it runs, not of the fleet. Measured both ways:
from the main checkout the siblings are present, `distinct=2`, and the test passes; from any
worktree — and from every CI runner, which clones one repo — none of them exists, `distinct` is 1
**by construction**, and the FELL arm fires. It could never have passed in CI. Lowering the constant
would have greened CI, turned the main checkout red at "grew to 2", and **silently retired the
invariant while world's fork is still on disk** — measured, it is. So the fix is the branch the test
already reaches for elsewhere: it skips when the shared driver is unreadable, and now also when NO
sibling is observable at all, with `knownDriverCopies` untouched. **A skip is a claim, not evidence
(rule 2), so the guard was proven non-vacuous in three directions** using synthetic siblings checked
absent beforehand and removed afterwards with the removal verified: one observable fork → the test
**evaluates and passes**, it does not skip; a third copy → the "grew" arm fires **RED** naming both
forks; none → skip. This is the `/tmp`-worktree lesson at a different address, and worth stating in
its general form: **the location you run a check from is part of the instrument**, so a red that
moves when you move the tree is a fact about the tree. The failure message was the instrument
reporting its own blindness in the voice of a finding.

**THE JUDGE'S FINDING: THE ORDER INSIDE `Clear()` IS LOAD-BEARING AND NOTHING PINNED IT.** Evaluator
`sonnet`, in its own worktree at the milestone commit: **PASS 96/100, ZERO blocking**. Enumerating
mutations from the DIFF rather than from the plan's table (rule 3n — the third milestone in a row
where that technique produced the best defect of the iteration), it found that swapping the two
halves of `Clear()`, removing the artifacts before saving the empty manifest, leaves the **whole
`internal/pipeline` package green** and T10 green as well. Reproduced first-party before acting on
it, per the judge-claims rule. The shipped order is the correct one so this changes no behaviour,
but it is load-bearing rather than incidental: the manifest is what AUTHORISES the artifacts, so a
clear that cannot *record* itself must not already have destroyed them; remove-first leaves a saved
manifest naming blobs that are gone. The `save failure` subtest asserted the contextual error and
stopped there, so nothing observed what happened to the artifacts on that path; it now also requires
the subtree to survive a failed manifest save. **Proven the SOLE killer** — under the reordering
mutation exactly one subtest goes red (`save_failure`) while `default`, `override` and
`remove_failure` all stay green — with `cache_store.go` restored byte-identical (`sha256
21cdaa1b…`). The judge also re-ran all five of the plan's mutations and confirmed each was killed by
the test the plan's kill-mapping *claims* kills it, which is the one column nobody usually checks.

**Ruled out / corrected.**
- **Not "a stale ratchet constant".** The obvious reading of the red — a fork was removed, lower the
  number — was refuted by measuring the same test from two checkouts and getting opposite verdicts.
- **Not a dropped-event or infrastructure problem.** `mergeable` was read FIRST every round (the
  boring cause), and runs existed for every SHA polled; no `workflow_dispatch` lever was reached for.
- **Not the queue head.** A red dev outranks the queue for the mission that owns the repo, and V1
  does own it — but the owner of the *defect* had already opened the fix, and "outranks" does not
  mean "duplicate". The distinction is the whole content of the shared-repo attribution rule.
- **The executor's `cmd/ailang` red was not a regression.** It is false-green (3) verbatim:
  `bind: operation not permitted` under `workspace-write`. Outside the sandbox: rc=0, zero FAILs.
- **`gh api .../logs` is not an instrument here.** It returned **0 bytes** for both jobs with its
  positive control also at 0 — an instrument failure, not a clean reading. `gh run view --job
  <id> --log-failed` returned 23,549 bytes and the actual failing test. Recorded because the first
  form is the one that looks canonical.

**Routing evidence.** `resolve-role-spawn.sh` run for all four roles, output used verbatim.
Designer `recipe claude:claude-fable-5-1` and planner `agent-tool opus
fail-closed:planner-lane-field-missing` — **NEITHER SPAWNED**, and that is a routing call rather than
an omission: the design doc landed at iteration 328 and the verified four-milestone plan at 329, so
there was nothing to author and nothing to plan. **Fable budget UNSPENT; the rotation pointer stays
at `codex:gpt-6-astra`.** Executor `codex:gpt-5.6-sol` (probe rc=0), one bounded sandboxed run under
the 30-minute cap, zero git write operations, `.snap/M3/` verified byte-identical to its final tree
(4/4) before the commit was built — and 3/4 afterwards, the single difference being exactly the file
the judge's fix touches. Evaluator `sonnet` in its OWN worktree at the sprint commit;
generator≠judge holds by vendor as well as by model (OpenAI wrote it, Anthropic judged it).
**The planner resolver/hook disagreement reproduces a 5th time** — measured, not routed on; the row
stays queued and the fix still belongs in the TOOL, not in an iteration. `metered=$0.00` of the $5
ceiling: every lane rode a subscription or quota bucket and no quorum round ran. No GPU, no
`rig.lock`.

**Verification the controller did not delegate.** Rule 3e(a) applied to my OWN directive's gate list
before the executor ran: all seven baselined on the pristine tree, six green and the cumulative
runner correctly RED at boundary M3 for the right reason — `('M3', 'missing or skipped',
{'TestCacheStore_ClearArtifacts'})` at `go returncode=0`, i.e. Go's vacuous zero-selected success,
which the gate rejects. After the milestone: twelve PASS lines, `internal/pipeline` ·
`cmd/ailang` · `internal/loader` · `internal/mission` all `ok`, build/gofmt/vet/`lint`
(0 issues)/`check-file-sizes` clean, and **nine further gates DERIVED from `ci.yml` rather than
remembered** after last iteration's `check-home-isolation` miss (`check-home-isolation`,
`check-no-personal-email`, `check-boundaries`, `check-referenced-paths`, `check-git-exec`,
`check-changelog`, `check-context-docs`, `check-golden-drift`, `check-tmpfile-hygiene`) — all rc=0.
Before the mandatory out-of-sandbox re-run I read the diff for destructive writes (false-green 5),
which mattered unusually here because the code under test DELETES DIRECTORIES: every test root is
`t.TempDir()`-derived with `AILANG_CACHE_DIR` set through `t.Setenv`, and the negative control for
introduced absolute/`$HOME` paths returned **zero** hits against many TempDir hits.

**Gate 3b on an observed green, never a predicted one.** All four required contexts
(`build` · `docs-gate` · `lint` · `test`) pass; `mergeable` read first and `MERGEABLE/UNSTABLE` —
the three surviving reds (`test-windows`, `Build windows-latest`, `launchd drivers (bash 3.2)`) are
base-inherited, non-required and all covered by motoko's `#1055`, and `UNSTABLE` is not `BLOCKED`.
Autoclose scan over all three commit messages and the PR title and body: **0** hits with the
known-bad control matching **1**; `#1046` verified still OPEN after the merge, which is correct.

**Bookkeeping defect corrected.** The design doc's own banner still read *"STATUS: PARKED … MUST NOT
be executed until `D-55` is answered"* while three milestones had shipped under iteration 329's
pre-registered default — a tracked document contradicting the tree, flagged by the M3 executor and
walked past by three iterations. Rewritten to state what actually unparked it and that **`D-55`
REMAINS OPEN by design**, since the loop may not resolve a ledger row on its own behalf.

**`origin/dev` moved four commits mid-iteration** (`fe9c08ffc`…`19d6b03c7`, the attended session's
workbench Phase 2/3). Caught only because the worktree created from `origin/dev` came up at a SHA
the Gate-1 fetch had never seen: the clone is shared, so a sibling's fetch advances my refs under
me. Not a defect, but worth the line — three writers were active in this repo tonight.

**Next.** M4, the last milestone of this sprint (route integrity diagnostic and MCP regression,
0.75 d); `#1046` closes with it. Then `m-cachesrc-cognitive-complexity`, which M3 and M4 both
inherit. `ci-red-mission-loop-workbench` is handed over and resumes only as a close: check
`#1055` merged, re-read the check set, close the row.

## 333 — 2026-09-06 — The sprint is complete, and the judge proved the milestone's own headline test was measuring the milestone before it [PRODUCT]

**Pick.** `m-compile-cache-unverified-artifacts` **M4 of 4** — the [IN-SPRINT] item, resume
predicate *"execute M4"*. The queue head, `ci-red-mission-loop-workbench`, was handed to motoko at
iteration 332 and its resume predicate was *"#1055 merged"*; that predicate was RUN as a command
rather than transcribed, and at Gate 1 it was still `OPEN`/`BLOCKED`. It merged on its own at
`00:03:40Z` as [`45bbcf625`](https://github.com/sunholo-data/ailang/commit/45bbcf625) while this
iteration worked, so the row is now RESOLVED.

**dev was RED at Gate 1, and it was not this mission's to take.** HEAD `353082b1c` carried three
non-required reds, all read first-party from `actions/jobs/<id>` rather than from the check name:
`launchd drivers (bash 3.2)` at step *Run launchd driver tests* with
`make[1]: go: No such file or directory` — `test-launchd-drivers` had acquired a
`test-mission-registry` dependency that runs `go test ./internal/mission/...` inside a job whose own
comment says *"No Go, no cache: these are shell + git tests and nothing else"*; and `test-windows`
plus `Build windows-latest` on `internal\mission\kill_unix.go:6:51: undefined: syscall.Kill`, because
`unix` is a build TAG and has never been a recognised filename suffix, so the file compiles on
Windows. Negative control: the identical set is red at `dev~1` and `dev~2`, and `dev~3`/`dev~4` have
`total=0` runs. All three were already fixed in motoko's open `#1055`, whose file list
(`kill_unix.go`, a new `kill_windows.go`, `make/test.mk`) covers both causes exactly, and whose
branch `git worktree list` attributes to motoko's clone at the exact head `5f7a9c476`. Recorded,
diagnosed, left alone.

**Outcome. LANDED — M4 of 4; the sprint is COMPLETE.** PR
[#1058](https://github.com/sunholo-data/ailang/pull/1058) → squash
[`761b37e64`](https://github.com/sunholo-data/ailang/commit/761b37e64), from five commits:
`3554a7fd1` (the milestone), `0b4c4fe13` (round-1 judge, two blocking), `801201828` (round-2 judge,
one non-blocking), `79a63c4de` and `944b3a5ef` (two Windows-only reds).

`registerModule` walked the AST, found an exported `@route` function, looked for a matching entry in
`loaded.Iface.Exports`, found none, and fell out of the inner loop with **no branch to take** — so
the route was dropped silently and serve-api published a tool surface missing a function the source
plainly declares. Validation now runs after the under-basePath filter and **before** both the
idempotent-return path and map publication, so a repeat registration cannot bypass it. Every
exported `@route` function must have a **non-nil** `loaded.Iface.Exports[fn.Name]`; a nil iface with
an exported annotated route reports the same inconsistency rather than returning silently; the error
names source path, module and function, carries `CACHE_ROUTE_IFACE_MISMATCH` and states the
compile-clear remedy. Nothing is retried, no name is added to the export list, `@nomcp`/`@noexpose`
do not excuse a missing export, and private annotated functions stay outside the invariant.

**The substance is the round-1 judge, and it is the sharpest finding this sprint produced.**
Evaluator `sonnet`, own worktree, at the milestone commit: **FAIL 68/100, two blocking**, both
reproduced first-party before anything was done about them.

*(1) T12 — the milestone's own headline acceptance test — is VACUOUS with respect to the entire M4
diff.* With `module_entry.go` reverted to the parent, `TestServeAPI_DivergentCacheTools` stays green
(`ok cmd/ailang 8.062s`) while T11 correctly reddens on three subtests; the control fires, so this is
not a broken instrument. The cause is structural rather than sloppy, which is what makes it worth
recording: **every divergence T12 constructs — old artifacts under a fresh stamp, one stale blob — is
caught by M1's per-blob SHA-256 verification before `registerModule` is ever reached**, so the M4
invariant cannot be exercised that way at all. The sprint plan's T12 row ("trust manifest alone or
omit one blob hash") is an M1 mutation wearing M4's clothes, and nobody noticed because the test does
pass and does exercise real machinery. `TestServeAPI_RouteIfaceMismatchFromCache` is the arm T12
could not be: it hand-writes an artifact set that is hash-**VALID** and logically incomplete —
delete `f7` from the cached `iface.json`, re-stamp `artifacts.json` to the patched bytes so M1
accepts them — and requires serve-api to REFUSE to start, naming the token, `f7` and the remedy, and
**not** reporting `reason=ARTIFACT_INVALID`. Its non-vacuity proof is the reported bug itself: with
the M4 hunk reverted it fails at *"serve-api started on a route/interface mismatch"* and its captured
log reads `Registered: entry (6 exports)` then `Starting MCP server on stdio transport...` — six
tools published for a source declaring seven, which is `#1046` live.

*(2) The `item != nil` half of the export lookup had NO killer.* Dropping it left the whole
`internal/apiserver` package and T12 green. It is load-bearing: `extractModuleInfo`
(`internal/apiserver/server.go:479`) ranges `Exports` and dereferences `item.Purity` with no nil
check, and a key present with a nil item is reachable from a corrupted `iface.json` inside this
design's own accidental-corruption threat model. One subtest pins it and is the **SOLE killer** —
under the mutation exactly one subtest reddens across the package, `module_entry.go` restored
byte-identical (`sha256 3a7711e5…`).

**Round 2: PASS 93/100, zero blocking**, all three round-1 findings adjudicated CLOSED by name. The
round-2 judge then attacked the NEW fixture four ways — wrong key deleted, `f6` control also
deleted, hash forced unchanged, re-stamp skipped — and every instrument-failure guard fired. The
skipped-re-stamp attack is the one to keep: the test fails with a *different, correctly attributed*
message (M1's hash gate recompiling from source) rather than passing for the wrong reason, which is
what proves the re-stamp step load-bearing. Its one non-blocking finding became the third commit:
the helper's comment claimed *"a healthy server blocks on stdin"*, and that is false — a healthy
`serve-api --mcp` exits rc=0 at EOF in **0.9s**, measured. The assertion was sound for a reason
nobody had written down (the mismatch is fatal *before* the transport is announced), so the comment
now names the real discriminator and the test asserts it explicitly.

**Two Windows reds, and the first is a real product defect this milestone only surfaced.**
`test-windows` reddened at `manifest entries = 0, want 1`. On the runner **every** artifact
publication fails `CACHE_WRITE_FAILED … ARTIFACT_INVALID`, and the diagnostic's own path says why:
`…\compile\modules\C:__Users__runneradmin__…`. `sanitizeModuleID`
(`internal/pipeline/cache_store.go:162`) maps only `/` and `\`, so a module ID beginning `C:/Users/…`
yields a directory component containing a **colon**, which Windows forbids. **The compile artifact
cache is non-functional on Windows.** Pre-existing: `internal/pipeline` is untouched by this PR and
`sanitizeModuleID` predates the sprint at `d96de92f5`; M1 only made the failure audible. Every
existing Windows test tolerates a cache miss, so M4's two MCP fixtures — the first that require a
PUBLISHED artifact — are the first to see it. Filed as its own queue row rather than absorbed
(iteration 257's rule), because adding `:` changes artifact-directory identity on every platform and
collides with the already-filed `a/b` vs `a__b` row. `requireCompileArtifactCache` gates both
fixtures on the OBSERVED empty manifest **AND** Windows, so on any platform where publication works
an empty manifest is still a hard failure — a skip that cannot hide a regression. The second red is
ordinary: T11 asserted `loaded.File.Path` while `registerModule` emits the path it RESOLVED, and
`t.TempDir()` returns the 8.3 short form (`C:\Users\RUNNER~1\…`) that `EvalSymlinks` expands. Fixed
by resolving the expected path the same two ways, not by weakening the assertion to a basename.

**Verification the controller did not delegate.** Every gate baselined on the pristine parent FIRST
(rule 3e(a)) — all rc=0 there, and both named tests returned `no tests to run`, which is the
non-vacuity control — then re-run outside the sandbox at each commit: scoped build
(`./internal/... ./cmd/ailang`, never `./...`, since `cmd/wasm` is `js && wasm`), `go vet`,
`gofmt -l` empty, the four-package suite (`internal/pipeline`, `internal/loader`,
`internal/apiserver`, `cmd/ailang`) all `ok`, and **eleven** further gates DERIVED from `ci.yml`
rather than remembered: `check-boundaries`, `check-file-sizes`, `check-referenced-paths`,
`check-git-exec`, `check-home-isolation`, `check-no-personal-email`, `check-context-docs`,
`check-changelog`, `check-golden-drift`, `check-tmpfile-hygiene`, `fmt-check`. `.snap/M4/`
byte-identical 3/3 before the milestone commit.

**Gate 3b on an observed green, never a predicted one.** `mergeable` read FIRST each round (the
boring cause). Final head `944b3a5ef`: **5 runs, 5 completed, 0 not-green**, `total=21` checks with
`pending=0`, all four required contexts (`build` · `docs-gate` · `lint` · `test`) pass,
`MERGEABLE/CLEAN`. Worth naming: `SonarCloud`, `test-windows`, `Build windows-latest` and
`launchd drivers (bash 3.2)` are **all success** on that head — the branch merged GREENER than the
base it started from, because `pull_request` checks build on the updated base and motoko's `#1055`
had landed in between. Autoclose scan over all five commit messages and the PR title/body: **0**
hits, known-bad control matching **1**. `#1046` was then CLOSED with the verdict posted as its own
`--body-file` comment first and the comment count asserted to have grown, per the mechanism-B rule.

**Ruled out / corrected.**
- *"The three dev reds are this mission's to fix."* Refuted. V1 owns the repo, so a red does outrank
  the queue — but the entire fix was already in flight on a sibling's PR whose file list covers both
  causes. Duplicating it is the `#758`/`#759` collision; the correct action was to record and wait,
  and it merged unaided.
- *"T12 verifies M4."* Refuted by mutation, and it had passed two rounds of human-free review before
  the judge asked the question. A test that exercises real machinery and passes is not thereby
  testing the thing it is filed under.
- *"The Windows red is our tests being sloppy."* Half true. One of the two was (the short-path
  assertion). The other is a product defect that makes the compile cache unusable on an entire
  platform, and it was found only because a test finally demanded that publication SUCCEED.

**Routing evidence.** `resolve-role-spawn.sh` run for all four roles, output used verbatim.
Designer → `recipe claude:claude-fable-5-1`, **NOT SPAWNED** (doc landed at 328; nothing to author),
so the **Fable budget is UNSPENT for a fourth consecutive iteration** and the rotation pointer stays
at `codex:gpt-6-astra`. Planner → `agent-tool opus fail-closed:planner-lane-field-missing`, **NOT
SPAWNED** (the verified 4-milestone plan landed at 329); `derive-planner-lane.sh` returns the same
`opus fail-closed:planner-lane-field-missing` for the **sixth** consecutive iteration while the
spawn-pin hook would deny that spawn — measured, not routed on, and the fix still belongs in the
TOOL. Executor → `recipe codex:gpt-5.6-sol`, probe rc=0, one bounded sandboxed 30-minute-capped run,
zero git writes, `.snap/M4/` verified byte-identical before the commit. Evaluator →
`agent-tool sonnet`, in its OWN worktree, both rounds; generator≠judge holds by vendor as well as
model, and round 2 judged code the CONTROLLER had written. `metered=$0.00` of the $5 ceiling; no
quorum spend (the doc was already through the gate), no GPU, no `rig.lock`.

**Progress.** N = **12** design docs remaining before v1.0.0 (was 13, **−1**):
`m-compile-cache-unverified-artifacts` LANDED complete and left the count, its doc and sprint plan
moved to `design_docs/implemented/v0_35_2/`, and `#1046` is closed.

**Next.** `m-cachesrc-cognitive-complexity` — the SonarCloud new-code maintainability red iteration
331 filed against itself, which M3 and M4 have both now inherited; it is NEW and ours, and the gate
is green on this PR only because the diff is test-heavy. Then the new
`m-cache-sanitize-module-id-windows-colon` row, which should be designed together with the existing
`m-cache-sanitize-module-id-collision` row rather than patched twice.

## 334 — 2026-09-06 — A designer, a quorum, a planner and a judge all ran, and the two most valuable things any of them produced were a refusal and a negative result [PRODUCT]

**Picked.** The queue head, `m-cache-sanitize-module-id-windows-colon` (filed by iteration 333
from the Windows runner) — and the first routing call was to **merge it with
`m-cache-sanitize-module-id-collision`** (iter-328) into ONE design. They are two symptoms of one
function. `sanitizeModuleID` (`internal/pipeline/cache_store.go:161`) maps only `/` and `\` to
`__`, so a Windows drive-letter module ID yields a directory component containing a **colon** and
the compile artifact cache publishes nothing at all on that platform, while on every platform
`a/b` and `a__b` collide onto one directory and evict each other. Iteration 333's own row said the
two "want one design, not two one-liners"; this is that design.

**Reality check.** Both defects confirmed first-party at HEAD before any routing: the function
body read at `cache_store.go:161-174`; the sole production call site
`moduleArtifactDir` (`cache_artifacts.go:403`) established by grep with a fabricated-symbol
negative control at 0 and a 4-file positive control; the repo's own
`sanitized_collision_uses_exact_module_id` subtest read at `cache_artifacts_test.go:64-73`, where
it asserts the collision and then proves the stamp backstop holds. `grep -ri` across
`design_docs/` found no existing doc, so the NEW-DOC tag was a fact rather than a claim. Base gate
GREEN out-of-sandbox at the design commit: `ok internal/pipeline 5.300s`, `ok cmd/ailang 29.801s`,
rc=0.

**Shipped.** PR #1060 → three commits: `8cd3bc783` (design), `5ef1058ef` (plan), `6ebc71a54`
(judge findings). Chosen encoding `m-<slug>-<16hex>` — the `m-` prefix is the Windows
reserved-device-name guard, the 16-hex SHA-256 prefix over the full module ID is the uniqueness
authority, the slug is a truncated readability aid carrying no correctness weight; max component
57 chars, independent of input length. Judge `sonnet`, own worktree: **PASS 85/100, zero
blocking, three non-blocking**, all reproduced first-party before being acted on.

**The quorum earned its money twice, and the second time was a refusal.** Round 1: `gpt6-astra`
was **ABSENT on `budget`** — a pre-flight refusal, `estimated cost $0.1170 … exceeds cap
$0.1000`, zero spend. The absent-reviewer rule says re-run it alone at a raised cap before acting
on any synthesis, and **it was the reviewer whose objection mattered**: the doc's legality
argument — "the output always ends in `-<hex>` so it is never a reserved device name" — is
**FALSE**, because Windows reserves the basename *before the first dot*, so `con.txt-<hex>` still
has basename `con`. A two-reviewer reading would have shipped that hole. `gemini-3-1-pro` and
`oc-glm-5-2` both rejected on the SAME unverified premise, and rule 3f says the controller
measures an objection rather than forwarding it: both were measured and **both held** —
`pipeline_module.go:269-296`'s only early exit is the `continue` inside
`verified && cached != nil`, so a `LoadArtifacts` error falls through to the ordinary compile path
with no error, panic or silent fallback; `maxModuleArtifactBytes` is exactly `32 << 20` at
`cache_artifacts.go:29`, test-pinned at `cache_artifacts_test.go:193` (control `cacheKeyVersion` =
14 hits). Round 2: all three PRESENT, `absent_reviewers` empty, **3/3 reject on three different
surfaces**, each with a concrete reviewer-authored `proposed_fix` and none disputing the
direction — the Gate-2 narrow-refinement carve-out exactly, so the reviewers' own fixes were
applied instead of parking. gemini's premise was measured and **REFUTED in the doc's favour**
(`Clear()` resets the manifest and `Save()`s it before `removeAll`, so M4's assertion needs no
weakening). **`oc-glm-5-2`'s objection found the defect nobody had**: it noticed the Conflict
Surface conflated *"no other package"* with *"no other call site"*, and measuring that turned up
`cmd/ailang/serve_api_mcp_surface_test.go:601` — `compileArtifactDir`, a **named helper in another
package that reimplements the old mapping**, invisible to every `grep sanitizeModuleID` the doc
ran.

**The designer rotation fell through, and the runner's own load-bearing assertion is what hid
it.** The pointer read `codex:gpt-6-astra`, so the next entry was
`pi:ollama/deepseek-v4-flash:0731-cloud`. It authored the doc well (verdict `ok`, 214s, 34 tool
executions, a real Verification Log with a firing negative control, and eleven worked-example
SHA-256 prefixes that the controller re-derived byte-exact). Its REVISION pass then returned
verdict **`ok` with `worktree_changed_files: 1` and wrote nothing** — 9 tool executions, all reads
of a file the directive had already quoted verbatim, and the doc's mtime predated the run.
`mission_pi_run.sh`'s worktree-diff assertion — the one its own header calls load-bearing,
*because* `stopReason` is evadable — is **vacuous on a revision**: the file was already untracked
from round 1, so `git status --porcelain` counts the PREVIOUS run's output as this run's proof.
Filed as its own queue row. The lane then fell to the NEXT rotation entry,
`claude:claude-fable-5-1` via `claude-sub` (probe rc=0, subscription-only by construction), which
applied astra's fix verbatim, dropped `.` from the slug alphabet as a second independent guard,
and computed four new suffixes with a real `shasum`.

**The planner's best output is a negative result.** `codex:gpt-5.6-sol`, ephemeral detached
worktree at the design commit, 4 milestones / 1.35 days. Applying rule 3o it flags **M3 and M4 as
having NO non-vacuous mutation of their own diff** — M3 ships a legality table plus a skip
deletion, M4 a regression guard over `Clear()` behaviour that already exists at base, and both
production hunks land in M1/M2 — and says so plainly rather than naming a green test. The judge
independently confirmed it by reading the pre-existing `TestCacheStore_ClearArtifacts`, which
already kills M4's named mutation. The planner also extended the Conflict Surface by three more
sites, each verified first-party before the commit: `cache_invalidation_test.go:313,328` and
`cache_artifacts_test.go:363` hard-code `filepath.Join(…, "compile", "modules", "answer", …)`.

**The judge's finding (1) is mine, and it is the same shape as the defect it was about.** Round 2
made me WITHDRAW the 32 MiB orphan-footprint guarantee from the Migration prose — and I left the
Verification Log row asserting it standing, still bolded *"Orphan disk bound is a real
constant"*. The doc withdrew a claim in one section and cited it as settled fact in another. I had
applied the reviewer's fix exactly where the reviewer pointed and never swept the document for the
same claim's echo, which is precisely what the reviewers were complaining about in the first
place. (2) The doc and the plan used *"not vacuous"* to mean two different things for the same
test/mutation pair and neither flagged it; the plan's stricter reading is right and the doc now
concedes it. (3) The plan required M3 to "show explicit PASS events" on Windows without
instructing anyone to wire it — the no-silent-skip gate is a hand-maintained literal list at
`ci.yml:111` (unix, 5 names) and `:480` (windows, 4), already asymmetric — so M3 now carries an
explicit work item to edit both.

**Routing evidence.** model=`pi:ollama/deepseek-v4-flash:0731-cloud` task-class=design (authoring,
verdict `ok`); model=`claude:claude-fable-5-1` task-class=design (revision, after the pi lane
returned a false green; **ONE** Fable run — the diet's unit is one DOC and the authoring half was
not Fable, so this is inside the ceiling; rotation pointer advanced past deepseek);
model=`codex:gpt-5.6-sol` task-class=plan (probe rc=0, sandboxed, ephemeral detached worktree,
zero git writes); model=`sonnet` task-class=evaluate (Agent tool, its OWN worktree, killed at 48
tool calls by a transient API/DNS drop and **resumed by name** per standing rule 7's sub-agent
amendment rather than re-run); model=`claude-opus-5` task-class=controller. Quorum reviewers
`gpt6-astra` / `gemini-3-1-pro` / `oc-glm-5-2`. generator≠judge holds by model on every pair.
**Executor NOT spawned** — a deliberate routing call: the plan came into existence in this
iteration and was materially amended by its own judge minutes before Gate 3b, so M1 executes next
iteration against a plan that has settled. **Planner resolver disagreement reproduces a 7th time**
(`agent-tool opus fail-closed:planner-lane-field-missing`, for a pick WITH a complete design doc);
routed straight to the configured `codex:gpt-5.6-sol` pin without burning a spawn on a guaranteed
hook denial, as the amendment requires. `metered=$0.28014` of the $5 ceiling (quorum R1
`$0.03115`, astra solo re-run `$0.08279`, quorum R2 `$0.16619`); pi is flat-rate and fable is
subscription, both $0. No GPU, no `rig.lock`.

**Ruled out.**
- *"The pi designer lane failed a probe"* — REFUTED. Probe rc=0, run rc=0, verdict `ok`. It is a
  **deliverable** failure the runner cannot see, not a lane outage, which is why re-probing would
  have cleared nothing and why it is a queue row rather than a `PARKED-ON-LANE`.
- *"astra's reserved-device-name objection is a ghost"* — NOT refuted, and deliberately not
  overclaimed either. `validateModuleName` permits no `.` in stdlib module names, but module IDs
  for user files are resolved paths and the Windows CI evidence shows absolute paths becoming
  module IDs; the controller did not establish first-party that a dot reaches the encoder. The
  `m-` prefix costs nothing either way, so the fix was applied without inflating the claim.
- *"`Clear()` might not reset the manifest, so M4's test would fail on its own assertions"*
  (gemini R2) — REFUTED by measurement at `cache_store.go:117-132`.
- *"A wider replacement set is the small fix"* — refuted in the design: it leaves the reserved-name
  hazard, makes collisions strictly worse, and does nothing about the component-length limit.

**Retro lane.** backlog — one new queue row,
`m-pi-runner-worktree-assertion-vacuous-on-revision`, on first-party evidence. No skill edit
spent: the two frictions this iteration produced (the vacuous runner assertion; the resolver
disagreement) both have their fix in TOOLS rather than in the skill, and the resolver one is
already a standing row at 7 instances.

**Next.** Execute **M1** of `m-cache-module-id-encoding` (the pure `encodeModuleDirName` plus its
unit table, 0.35 d) against the settled plan — and read the plan's non-vacuity ledger before
touching M3 or M4, both of which it flags as shipping no production mutation of their own.

## 335 — 2026-09-06 — Recover the stranded cache-encoding design and make its CI instructions executable [ADMIN]

**Picked.** Recover iteration334's own open PR #1060 before starting M1. Its clean worktree and
five commits existed, but the PR was still OPEN at `c2a9d8fb4`, despite dashboard prose saying
landed. This is delivery recovery, not another design or implementation iteration.

**Reality check.** Pin `e50066037` and original iteration334 worktree were clean. Main checkout
had 14 pre-existing dirty paths; left untouched. Origin advanced to `294885901` through three
sibling docs-mission files, disjoint from this recovery. Kill switch armed, billing tripwire clean,
GitHub account `sunholo-voight-kampff`. Canonical inbox went from 12 to 13 unread when the sibling
report arrived; package messages and D-55 left unacknowledged. No new Mark directives on issue972.
Decision ledger validated at 56 rows/two OPEN. No current iteration334 worker remained.

**Shipped.** PR #1060 recovered and merged at `d7eb07deb8eeeaa25e8ce08e652b1529b52a6366` from exact reviewed head `a52fbcad2833f0cdc08d7e516a679d30b2bf396c`.
All five expected PR workflows completed (four success, Dependabot automation skipped), and
merge CI plus Build and Release were observed success; documentation deployment was path-filtered
on the dev push. This corrects the prior iteration's premature landing claim. Only docs changed.
`a52fbcad2` adds the missing BOTH-platform package/regex/required-PASS-loop instructions and banks
an initial four-milestone JSON. Runtime JSON was absent in all three checked worktrees; the tracked
snapshot is recoverable, while runtime state stays ignored and must never overwrite an active sprint.
No release, production code, milestone execution or human approval was manufactured.

**Routing evidence.** Controller `codex:gpt-6-astra`; advisory explorer inspected inherited fixes.
Planner resolver returned `recipe codex:gpt-5.6-sol anthropic-fallback:fail-closed:planner-lane-field-missing`.
The explicitly requested Agent tool spawned native `gpt-5.6-sol` for the bounded plan/state correction;
this is an Agent transport adapter rather than the CLI recipe, with the requested model pin retained.
Designer NOT spawned: inherited design/quorum unchanged; no authoring phase on this recovery pick.
Executor NOT spawned: no production milestone selected; its next work remains gated on M1 wording.
Evaluator REQUIRED and spawned via Agent tool: Astra was only the orchestration wrapper; the actual
independent judge was `pi:ollama/minimax-m3:cloud` in separate clean worktrees at each exact SHA.
Generator≠judge holds across models and vendors. No judge fallback, no self-score, no new quorum.
Round1 FAIL84/one blocker; round2 PASS91/zero blockers. Runner verdicts ok, rc0/pi_rc0, respectively
486s/157 tools and 242s/58 tools. The evaluator's first probe failed rc1: missing `@anthropic-ai/sandbox-runtime`, repaired by
loading the byte-identical canonical extension with dependencies through an invocation-only shim;
subsequent probes rc0. The stuck round1 judge inbox-read child was terminated after >120s after
PID lineage verification; judge continued. Sandbox-denied Go testing is uninformative, not product red.
Fresh report files were the only changed files in each judge worktree. Raw reports and controller
qualifications are retained under `docs/sprint-retros/iter335-cache-module-id-recovery-*`.
MiniMax usage: R1 10,653,918 input/34,894 output; R2 2,334,785/20,886; probes 4,298/50.
All are reported flat-rate usage, not metered bills. Codex/Agent token totals unavailable here;
not invented. `metered=$0.00`; quota buckets Codex and Ollama Cloud; no GPU or rig lock.

**Ruled out.**
- Prior green checks did not prove PR1060 landed: GitHub OPEN plus no merge commit disproved it.
- The prior CI plan fix was incomplete: both commands omitted `./internal/pipeline`. Adding only
  regex names can omit the tests; adding required-PASS entries without the package fails loudly.
  The corrected plan and JSON require all three edits on both platforms, preserving Z3 asymmetry.
- The Windows slug finding is a design/plan wording ambiguity, not an implemented encoder failure.
  Design run-collapse/table and plan per-byte/runs-allowed need designer clarification before M1.
  The raw judge's no-example-fixture claim contradicts the plan's worked-example requirement.
- Q3's outside-pipeline claim is too broad; M2 already owns the cmd/ailang fixture migration.
- A substring OPEN search is not a ledger count: status-column validation gives two, not five.
- The 1.35-day narrative estimate is stale: milestone sums and JSON equal 1.20 days/220 LOC.
- Independent reports are evidence, not unquestionable truth: controller addendum preserves the
  real findings and corrects unsupported validator, regex-only and harness-attribution claims.

**Retro lane.** backlog — retain the existing runner revision-assertion and shared-ref-drift
queue entries; add the nonblocking slug/Q3 wording clarification to M1's resume condition. No
skill edit or routing-policy change. D-56's missing approval-spine notice was recovered as an
operational action (message `inbox_1788668401999_50e12652`), and delivered body verified byte-exact.
D-55 and D-56 remain OPEN; standing defaults are not human answers.

**Progress**: N=12 design docs before v1.0; goal unmoved. Recovered design/plan; 0/4 milestones executed.

**Next.** Designer clarification of slug run handling/example and Q3 wording, then M1 of
`m-cache-module-id-encoding` against the corrected plan and pending snapshot. After this banked
sprint: shared-clone-ref-drift and pi runner revision assertion, in current queue order.

## 336 — 2026-09-06 — Correct the cache encoding specification and park the disputed naming direction [REFUTATION]

**Picked.** Resume `m-cache-module-id-encoding` at the iteration335 designer-clarification gate,
then intended M1 only. The pure encoder does not exist at base; the old separator-only mapping
and sole production consumer still do. The design gate remained blocked, so no milestone ran.

**Reality check.** Pin/origin base `e30904f71d59b8a6b93f10c3b8d77bc28bce4f48`, all 20 check runs
settled without a failure; CI, Build and Release, and Docs workflows present and successful.
Kill switch armed, Anthropic billing tripwire clean, gh account `sunholo-voight-kampff`.
Canonical inbox 15 unread at start; relevant prior report/approval notices triaged, no ack or
human answer inferred. CLAIM sent as `inbox_1788673087485_008bda3e`. Issue 972 had 40 comments,
zero new Mark directives; watermark unchanged. No weekly rotation/sweep owed. Ledger 56 / two OPEN
before D-57. Main dev0 ahead / 19 behind with 14 pre-existing dirty paths, incoming overlap; left parked.
Running skill resolved-target bytes matched origin. Fleet PRs1041/1033/945 unrelated; left alone.
Worktree `.wt-v1-iter336`, branch `sprint/v1-iter336-cache-encoder`, isolated from shared main.

**Shipped.** PR [#1061](https://github.com/sunholo-data/ailang/pull/1061) merged at `de529cb5f852d38cf625d9f91e90aa7e54fa1491` after observed complete green checks: 21 checks, zero pending/failures; all 5 expected workflows settled (4 success, Dependabot skipped), including Windows and SonarCloud, MERGEABLE/CLEAN. It banks a docs-only correction
and explicit park, not an implemented encoder. `ca2aff468` clarifies byte mapping, trimming,
38-byte truncation, full-original-ID hash and 16 hex suffix; all 16 reference rows reproduce exactly
(output SHA256 `de01bb14d1e5c6008b8b3f4c3a2d1637e4767eb8a6531fcd92e10757c1056b0d`).
It withdraws universal injectivity and captured-Windows-outage claims. `b2f31b154` marks the plan
and initial snapshot blocked, labels criteria historical, and forbids copying runtime state until
D-57 + design gate + planner resynchronization. All 4 features false/pending with null timestamps;
runtime snapshot absent. Independent review: **PASS 14/15**, exact covered record `b2f31b154f2024fa1494092b1f37ff018030fcc8`.
Evaluator reports retained; this verdict evaluates correction/parking, not the disputed direction.
Baseline make build, make test (8,242 top-level Go PASS events across 123 packages), pipeline tests
(9.557s) and make lint (0 issues) passed. No production diff, mutation claim, Windows execution
claim, completed milestone or release manufactured. The final digest records delivery of this bookkeeping commit separately.

**Routing evidence.** Controller `codex:gpt-6-astra`. User explicitly required Agent roles;
all four dispatched. Designer resolver `recipe codex:gpt-6-astra declared:provider-pin`; native
Agent Astra authored/revised design. Rotation advanced to Astra after actual run. D-56 interim
workaround substituted Sol for Astra in quorum; OpenAI/Google/Z-AI external reviewers all present.
R1 BLOCKED 2 rejects / 1 pass ($0.14160532 reported), bounded revision, R2 BLOCKED 2 rejects / 1 pass
($0.13907585 reported). Raw JSON preserved. Sol’s R1 injectivity objection corrected; R2 exact
source-provenance request satisfied with zero diff over its 8 named files between prior worktree
`c2a9d8fb4abfadb472a5c05461f10a506f4a8013` and base. GLM’s remaining choice of naming scheme
is a direction dispute; narrow-refinement carve-out does not apply and no third quorum ran.
Planner resolver `recipe codex:gpt-5.6-sol anthropic-fallback:fail-closed:planner-lane-field-missing`;
native Sol Agent audited option impact, then fixed blocked metadata only. Executor resolver
`recipe codex:gpt-5.6-sol declared:provider-pin`; native Sol Agent performed read-only execution-
gate audit: encoder 0 code hits, old production call present, four pending milestones. Neither role
advanced the blocked plan into implementation. These were gated readiness tasks, not a new sprint.
Evaluator resolver `recipe pi:ollama/minimax-m3:cloud declared:provider-pin`; requested Agent tool
spawned an Astra transport wrapper that invoked actual independent pi MiniMax in isolated exact-
commit worktrees. Wrapper never scored. Judge probe rc0 with requested model identity; byte-identical
canonical sandbox extension supplied dependencies via invocation-only shim. No model fallback.
First judge attempt could not write: repeated session-protocol denials, then verified-process
termination; runner rc10 empty_worktree/pi143,446 s, 226 tools, zero changed files and no report.
The same model was restarted at b2f31 after reading the guard source; bounded inbox list plus
session_protocol_ack returned acked:true, without disabling safeguards. Successful runner rc0,
pi 0, 182 s, 37 tools,exactly one fresh report. Reported flat-rate input/output: failed attempt
12,436,856/22,899; successful review 1,270,501/10,635. Judge verifies 11 explicit rows; the heading
claiming 16 is overbroad. Controller’s separate 16/16 check is not attributed to the judge.
Raw score 14/15 is retained without manufacturing a normalized 100-point score. Historical judge
prose mislabels PR1060 as iteration336; it was recovered in 335, as the controller addendum notes.
Native model pins were retained via Agent adapters to resolver CLI recipes. Workdir changes dropped
MISSION env during preliminary resolver calls; corrected by resolving in the pinned controller cwd,
not misclassified as provider failure. No role spawn failed. Quota lanes: Codex, Ollama Cloud;
no Anthropic, GPU or rig lock. API-priced quorum cost **$0.224037**; raw total **$0.28068117** also
includes **$0.05664417 imputed GLM flat-rate value** from registry pricing, not metered billing.
Actual per-reviewer tokens retained in raw artifacts; quota usage is not invented as metered cost.

**Ruled out.**
- Universal injectivity from a bounded 64-bit hash suffix: impossible; exact-ID stamp validation
  provides rejection of wrong-module artifacts, not freedom from all possible directory contention.
- GLM’s single-separator-to-two-underscores claim: false under one-byte-to-one-underscore mapping.
  Lossiness does not imply zero readable prefix value; neither observation overrides its direction rejection.
- Earlier worktree evidence automatically current: now proved by exact 8-file source diff, not assumed.
- Windows diagnostic captured on CI: not substantiated by fetched job 101410083473 log; reconstructed
  illustration now labeled, actual Windows publication proof remains M3’s obligation.
- Snapshot `not_started` safe while prose says parked: planner flagged the automation hazard and
  changed it to blocked. Historical acceptance text remains explicitly unapproved until resynchronized.
- Failing quorum means no independent judge needed: rejected. Actual MiniMax reviewed the park;
  no generator verdict substitutes for that review.

**Retro lane.** backlog — D-57 is the single new human decision and updates the existing encoding
row rather than creating duplicate work. The evaluator session-protocol denial gets one new
pre-registered handshake row; first recorded instance, so no skill edit spent. Keep banked
pi-runner revision assertion and shared-ref drift next. Workdir environment loss was repaired
within the invocation by resolving in the controller cwd, not by changing routing policy. D-55/D-56 remain OPEN with standing defaults;
D-57 defaults HOLD immediately until answered. Cost provenance separated from flat-rate imputation.

**Progress**: N=12 design docs before v1.0.0 (was 12, change 0); no doc left the bar inventory.

**Next.** `m-pi-runner-worktree-assertion-vacuous-on-revision`, then
`m-gate1-shared-clone-ref-drift`; encoding resumes only after D-57/design/planner gates.

**Record verification.** Dashboard 30 lines; ledger 57 rows / 3 OPEN; STATUS arithmetic 4777→4777 before ledger/queue edits, moved 333 present in archive with 332 control. All 3 decision-bearing artifacts copied byte-identically and not ignored (ignored-path control fired). Context-docs/file-size gates passed; referenced-paths 47 enumerated / 47 checked; diff whitespace clean.

## 337 — 2026-09-06 — Bank the pi runner false-green evidence and park its unresolved snapshot contract [HARNESS]

**Picked.** `m-pi-runner-worktree-assertion-vacuous-on-revision`, the ready queue head after
D-57 parked cache encoding. The requested outcome was one unattended mission iteration.

**Reality check.** Base `374d8a4217358721b2bb2ad7fe52b7be5c93377d`; existing main checkout14dirty
paths left untouched. Isolated sibling worktree used; charter/log re-confirmed byte-identical to
origin immediately before Gate4. Kill switch armed, billing tripwire CLEAN, GitHub account
sunholo-voight-kampff; current dev20checks/zero failure, no red preemption. Canonical inbox20unread
triaged without acknowledgement; own reports/approvals and sibling/package items were not human
directives. Issue972 no new allowlisted directives, no rotation or weekly sweep due. D-55–D-57
remain OPEN. A sibling World credential owner request was not V1 implementation authority.
CLAIM inbox_1788678433169_c8d970e2 sent before routing.

The writeless stub returns clean rc10/empty_worktree but dirty and preexisting-untracked rc0/ok.
A second edit to an already-dirty file preserves its porcelain status/name while changing bytes;
before/after porcelain comparison is therefore insufficient. Baseline make build, make test
(8,242 top-level Go PASS records,123packages), and make lint passed. Existing runner suite passed
9/9 with MISSION_PI_POLL_SECONDS=1; planner default8/9 exposed an inherited T3 timing flake.
No production regression is claimed from that flake.

**Shipped.** PARKED needs-human-review D-58. Design and two complete quorum artifacts are banked
at `a46e4c015cecb26ad06a3b44d5b6f8895a2d111e`; rejected draft retained with authoritative
controller qualifications, not presented as an approved contract. No sprint plan or production
implementation; no milestone claimed. Independent MiniMax docs-only evaluation PASS82/100; zero blocking evidence/park defects.
No third quorum, no narrow-refinement invocation, no self-approval. D-58 approval spine message
inbox_1788680529734_00877707 delivered and body verified via canonical list without acking others.

**Routing evidence.** Controller codex:gpt-6-astra; all FOUR explicitly requested Agent roles
spawned. Designer native Astra wrapper was transport only, actual author
pi:ollama/deepseek-v4-flash:0731-cloud, next after Astra in the namespaced rotation. Initial
pre-model launch failed from missing sandbox-runtime dependency, rc10/zero model records; repaired
with an invocation-only shim loading byte-identical canonical extension files with dependencies.
Fable probe rc0 was only a probe; no Fable author fallback. Actual designer rc0,538seconds,39tools,
1,433,715 reported total tokens; one revision rc0,667seconds,39tools,1,717,067tokens. Both quota
runs billed reported0; this is flat-rate usage, not invented API spend. Revision content SHA was
checked separately because this iteration's own bug makes the runner's dirty-tree assertion vacuous.

Planner and executor were native Agent gpt-5.6-sol, matching resolver pins via the user-requested
Agent transport adapter. Both performed read-only readiness audits; neither planning nor execution
could pass the blocked design gate. This is a gated non-execution, not a spawn failure or fallback.
Evaluator native Astra wrapper invokes actual pi:ollama/minimax-m3:cloud in a separate completed
worktree at the exact evidence SHA; generator≠judge by model and vendor. Judge results recorded
below when complete; wrapper never supplies a score. No role silently omitted.

Quorum R1 and R2 each rejected3/3, absent_reviewers empty. R1 API-priced Astra$0.083740
(6614in/352out), Gemini$0.017694(7179/278); GLM$0.00934762 flat-rate imputation(6554/414).
R2 API Astra$0.091570(7707/290), Gemini$0.020202(8445/276); GLM$0.01778778 imputation(7628/2329).
Metered API total$0.213206; flat-rate imputation$0.0271354 separate. Quota buckets Codex,
Ollama Cloud, plus a Fable probe; Codex tool token totals unavailable, not invented. No GPU lock.

**Ruled out.**
- A clean runner exit does not prove work: dirty no-op repro is a firing counterexample.
- A content manifest must validate ancestors: replacing tracked d with an external-directory
  symlink makes leaf-only checks read external d/f bytes. Astra's R2 objection reproduced.
- Gemini's R2 distinct-count objection is a prose clarification: prototype already cuts paths
  before comm, and one edited dirty file reports1. It is not evidence of a prototype count3 bug.
- `git hash-object` writes the object DB only with `-w`; negative/positive synthetic controls
  disproved the designer's blanket assertion. Alternative design remains open, not dismissed.
- The initial revised cost trial was invalid (WD not exported, killed after177seconds, later
  syntax error); all children ended. Independent controller prototype: rc0,5.442seconds,
  24,381records. Tracked bytes268,361,724; doc's3.7MB claim false and explicitly corrected.
- Narrow-refinement allows ONLY reviewer-authored verbatim fixes. All three external R2 fixes
  are concrete, but remaining controller objections still need design judgment: before-snapshot
  failure vs finished-only precedence, comparison outside shown watchdog, final-component output
  aliases, newline-preserving link text. Declined carve-out; did not buy a third round.
- Consumer census106hits/14files has runner/test positive controls; no other executable pi-verdict
  consumer was found in that census. No broad claim that every error parser supports rc15.
- Readiness audits and baseline green do not make an unimplemented design an evaluated sprint.

**Retro lane.** backlog — existing pi runner row parked D-58; add the separately measured inherited
shell-suite coverage/timing row, below shared-clone-ref-drift. No skill edit or routing-policy change.
D-58 asks A fresh designer revision/quorum (recommended) vs B commission Git-native alternative;
default HOLD immediately. Existing three decisions retained without fabricated answers.

**Progress**: N=12 design docs before v1.0.0 (was12, change0); goal unmoved. No bar doc landed.

**Next.** `m-gate1-shared-clone-ref-drift`, then `m-pi-runner-shell-suite-coverage`, then
`m-pi-evaluator-session-handshake`. Pi runner implementation waits for D-58/design/planner gates;
cache encoding waits for D-57/design/planner gates.


**Independent evaluation and controller qualifications.** Actual MiniMax PASS82/100, docs-only
park, at exact `a46e4c015`; raw report preserved byte-identically. Judge inspected runner/test
source, quorum, census and corrections independently; no implementation acceptance was claimed.
Sandbox inbox attempt stalled despite its45s alarm. Wrapper verified lineage and terminated only
the inbox child at100s; judge resumed, with no fallback or self-score. Controller addendum corrects
the raw Method's bound claim and distinguishes banked controller repros from judge-owned checks.
Review applies to evidence at the named SHA; later mission bookkeeping is controller-authored,
mechanically checked and does not change the design or raw verdict.

**Record verification.** Dashboard30lines; ledger58rows/four OPEN; STATUS rotation preserved line
arithmetic and all3blocks, moved334 confirmed in archive with prior333 control. Decision-bearing
artifacts tracked (ignored bin control fired). Context-docs, file-size and referenced-path gates
passed (47enumerated/47checked), whitespace clean. Shared main untouched. Remote gate pending.

Judge telemetry: runner rc0/pi_rc0,223seconds,41tools,one fresh report. Provider-reported
1,340,910input/8,795output tokens; cost0 on Ollama Cloud quota. No wrapper inference billed.

## 338 — 2026-09-06 — Pin shared-ref observations, recover the stranded sprint, and fix inherited CI red [HARNESS]

**Picked.** `m-gate1-shared-clone-ref-drift`, the ready queue head. Gate 2 found the prior
iteration338 attempt dead mid-flight in `.wt-v1-iter338`; its committed design, plan and M1 plus
stale uncommitted monolithic M2 were preserved. A clean recovery worktree was built from the
commits instead of continuing or discarding the stale tree. Claim
`inbox_1788691105959_885c997f` was sent before routing. The unattended standing request explicitly
required Agent-tool designer, planner, executor and evaluator roles; all four ran.

**Reality check.** Gate 1 began at `927d0dec086fc506173784c16d01b2a3373256ce` and the shared
remote-tracking ref advanced during setup, reproducing the queue row's premise. Kill switch armed;
billing tripwire clean; GitHub account `sunholo-voight-kampff`. Canonical inbox was triaged without
acknowledging any row; no new Mark directive was inferred. Ledger remained 58 rows/four OPEN
D-55–D-58. Main checkout ended 0 ahead/1 behind with exactly 11 pre-existing dirty paths, all
untouched. Separate worktrees held implementation and every judge round.

**Shipped.** PR [#1063](https://github.com/sunholo-data/ailang/pull/1063) landed as
[`0b7f3e3af`](https://github.com/sunholo-data/ailang/commit/0b7f3e3af). `mission-base.sh` records
one full-SHA/UTC snapshot and has eight non-vacuous arms; Gate 1 records/re-reads the base before
worktree creation; Gates 3, 3b and 4 distinguish missing evidence from measured drift and carry
the observation time into routing evidence. The root mission-control skill stayed within its
split-resource ratchet. R1 independent MiniMax: PASS92, no hard failures.

The first PR check set exposed three Windows failures already present on the pinned parent:
unescaped `t.TempDir()` backslashes in TOML and a POSIX-only explicit-driver fixture. The same
Sol executor changed only `internal/mission/render_test.go`; R2 MiniMax PASS93. PR #1063 then
passed its complete check set, including both Windows jobs, before merge.

After merge, exact-SHA Gate 3b observed `Build and Release` and `CI` red on one inherited test,
while docs deploy was green. Attribution was first-party: base `aebf8bb73` was already red before
#1063 on `TestMissionDocHeadingsStayCanonical`; Linux/macOS counted 12, Windows 22, and the local
checkout counted 30 when a sibling world repository existed. The ratchet mixed three ambient
inputs: sibling checkout presence, CRLF materialisation and repo-local headings. Sol made the
corpus repository-local, trims the CR delimiter, lowers the stable count 30→12, fails reads loudly,
and adds sibling/line-ending regression arms. R3 MiniMax PASS95. PR
[#1064](https://github.com/sunholo-data/ailang/pull/1064) passed all 20 checks and landed as
[`b50bb366e`](https://github.com/sunholo-data/ailang/commit/b50bb366e).

**Routing evidence.** Controller `codex:gpt-5.6-sol`. Resolver outputs were recorded verbatim:
designer `recipe codex:gpt-6-astra declared:provider-pin`; planner
`recipe codex:gpt-5.6-sol anthropic-fallback:fail-closed:planner-lane-field-missing`; executor
`recipe codex:gpt-5.6-sol declared:provider-pin`; evaluator
`recipe pi:ollama/minimax-m3:cloud declared:provider-pin`. The first recovered designer/planner
Agent launches inherited an ambiguous adapter model, so they were not silently accepted: designer
was re-spawned explicitly as Astra and planner explicitly as Sol. Astra designer audited the
recovered doc and found the Gate-3b exit-2 omission. Both quorum rounds were BLOCKED with Astra
ABSENT (`OPENAI_API_KEY` unavailable), Gemini and GLM present/rejecting; every applied change was
reviewer-authored concrete text under the already-ratified narrow-refinement carve-out, followed
by the explicit Astra designer audit. No force-pass or controller approval.

Planner and executor ran as explicitly pinned Sol Agent roles. Executor delivered M1/M2/M3 and
both inherited-CI corrections; controller committed each bounded change and re-ran gates outside
the executor. Evaluator was an explicitly pinned Astra Agent transport wrapper only. Actual R1
judge was `pi:ollama/minimax-m3:cloud`. In R2 the same primary probe succeeded but launch failed
before judging: typed rc10 `empty_worktree`, pi_rc0, zero tools, sandbox extension could not resolve
`@anthropic-ai/sandbox-runtime`. The locked dependency was restored with `npm ci`; per chain rule
the failed lane was not retried, and configured fallback `pi:openrouter/minimax/minimax-m3` judged
R2. R3 went directly to that successful fallback. Its first report emission ended provider
`stopReason:error` with response `gen-1788700729-R3LTzVPDDPf1TueU8Hgq`; runner rc0 was rejected as
a scratch-file false-ok. Same-round MiniMax continuation wrote PASS95 and MiniMax itself corrected
transport metadata. Wrapper supplied no score or prose. Generator OpenAI Sol != judge MiniMax in
all accepted rounds. No role omitted; evaluator failures and fallbacks are explicit.

**Verification.** Controller: `make test-launchd-drivers` 54+27+82+17 plus heartbeat/base/stall/
memgate/kicker/probe arms green; `make check-context-docs`; `make build`; `make lint`; focused and
full `internal/mission` tests. First local `make test` returned a truncated, non-reproducing red;
immediate cached rerun was rc0 and remote `test` was green. `.snap/M2` six files and `.snap/M3`
eight files were byte-identical to their worktree mutations, retained untracked. PR #1063 final
head `725aa5b74` was all green; PR #1064 final head `322a9f05a` was 20/20 green. Autoclose scans
over both PR title/body/commits found zero matches; known-bad controls matched.

**Ruled out.**
- The post-merge red was not caused by #1063: both workflows were already red on its immediate
  base `aebf8bb73`; #1063 removed the three pre-existing Windows fixture failures.
- A single lowered constant is not portable: the old count depended on sibling checkout presence
  and Windows line endings. The regression fix removes both ambient inputs.
- GOOS Windows compilation and string simulation are not Windows runtime proof; actual PR Windows
  jobs are the banked runtime evidence.
- Quorum `blocked` was not erased. Only its concrete proposed fixes were used under the narrow
  carve-out; missing-reviewer and design-audit facts remain in the record.
- A typed pi rc0 was not treated as judgment when the provider errored and only scratch changed.

**Retro lane.** skill — the landed Gate 1/3/3b/4 shared-ref protocol makes the observation time
and drift explicit. Backlog remains `m-pi-runner-shell-suite-coverage`; evaluator soft gaps remain
the missing dedicated double-snap race arm and a combined sibling+CRLF production fixture. No
extra feature scope was absorbed.

**Progress.** N=12 design docs before v1.0.0 (was12, change0); this HARNESS item moved the goal by
0 while making every later mission iteration's ref provenance auditable.

**Cost.** Actual metered API $1.48796278: Gemini quorum $0.057238; R2 MiniMax/OpenRouter
$0.69360648; R3 MiniMax/OpenRouter $0.73711830. GLM quorum $0.06206547 is flat-rate imputation,
reported separately. Astra/Sol/Ollama were quota lanes; no invented token cost. No GPU/rig lock.

**Next.** `m-pi-runner-shell-suite-coverage`, then `m-pi-evaluator-session-handshake`. Cache
encoding remains parked on D-57; pi runner revision remains parked on D-58. No unattended answer
was fabricated for D-55–D-58.

**Independent evaluation.** Actual MiniMax reports are tracked at
`docs/sprint-retros/iter338-gate1-ref-drift-eval-r{1,2,3}.md`: PASS92, PASS93 and PASS95, all with
zero hard failures. R2/R3 name sandbox/runtime limitations and carry-forward scope; each names its
exact evaluated SHA. The Agent wrappers were transport and containment only.

**Record verification.** Exact merge SHA `b50bb366e02e82a971bada39acf080b26b192161`
settled GREEN: CI run 34037385262, Build and Release 34037385268, and docs deploy 34037385203.
Gate 4 recorded `base=578e8c3008c576826baaf2198f9b25afa61abf4a@2026-09-06T14:16:01Z`;
the record branch was reconciled onto that observation before review. Charter remains 4783 lines
with exactly three structural STATUS rows; moved335 is present in the bounded 20-entry archive,
and moved313 is preserved in the old archive. Queue row survived and is LANDED. Dashboard31lines;
log rotation kept20 full entries and regenerated the index. Ledger, tracked-path, context-doc,
file-size, reference, changelog, personal-email, tmpfile, skill and whitespace checks passed.
