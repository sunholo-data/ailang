# M-RUN-POLICY-RESULT-UNFORGEABLE: the confined program cannot forge the supervisor's `policy-result:` line (#1548)

**Status**: Planned
**Target**: v0.53.0 (triage doc carried no target; retargeted per maintainer scheduling, 2026-10-08)
**Priority**: P0 (maintainer-approved scheduling from the P0 issue triage, 2026-10-08; triage severity: bug, security, low)
**Estimated**: 2 days (M1 5h, M2 3h, M3 2h, plus buffer)
**Dependencies**: None. Builds on the supervisor and the fd-3 control pipe shipped by
`m-executor-policy-hardening` (implemented v0.41.0) and the 2026-10-03 policy fixes
(v0.52.1: `96b6e5174`, `c66f33e4d`, `1099e49f6`, `ed1df2d3f`), which closed #1551–#1554/#1558/#1559
but not this one.
**Issue**: Refs #1548 (sunholo-data/ailang#1548, bug / security, open). Do not open a new issue —
this doc is the existing issue's design deliverable. `Refs #1548` goes in the sprint plan and the PR;
`Closes #1548` goes only in the implementation PR.

---

## Measurement provenance

Every code claim below was re-read in this worktree at **`62ac2d09` (branch `dev`, 2026-10-08,
v0.52.5)**. Issue triage verified the defect live on origin/dev `658ff76a3` (2026-10-08 pin);
the triage doc additionally verified it at `2d7a9a174` (v0.52.1). This session re-reproduced it
**live** with the installed v0.52.5 binary (transcript in the Verification Log, rows V1–V2): the
mechanism is unchanged from the triage pin to HEAD.

This doc makes no claim about AILANG language semantics that `ailang check` could falsify — the
one language-level premise (a confined program can `eprintln` and `exit(3)` under an IO-admitting
policy) was verified live as part of the reproduction, not asserted. Every negative-existence
claim (the "nothing filters worker stderr" / "nothing reaches fd 3" family) carries its own
Verification Log row.

## Problem Statement

`ailang run --policy P prog.ail` is a supervised run: the parent resolves the policy, arms the
wall-clock deadline, and starts a worker (this same binary, `--policy-worker 3`) in its own
process group. The worker's admission decision travels on a dedicated control pipe (fd 3,
`cmd.ExtraFiles`, `run_policy_supervise.go:71`) and the parent re-emits it "from the control pipe
only" — a program that prints `policy: {...}` on **stdout** is not believed (pinned by
`TestRunPolicy_StdoutCannotSpoofDecision`).

The *limit* verdicts have no such protection. Three facts compose into the defect (all verified
at `62ac2d09`):

1. **The worker's stderr is relayed byte-for-byte onto the supervisor's stderr.**
   `copyCapped(os.Stderr, stderr)` (`run_policy_supervise.go:126`) is a raw 32 KiB chunk copier
   (`:105–124`) — no filtering, no line awareness.
2. **The supervisor writes the limit envelope on that same stderr stream.** Four
   `policy-result: {...}` writes (`run_policy_supervise.go:166` output_limit, `:170` timeout,
   `:180` worker_failed, `:185` no_decision; envelope built by `limitEnvelope`,
   `run_policy.go:299`), plus the admission line `policy: {...}` (`:159`).
3. **The worker's exit code passes through verbatim** (`return exitErr.ExitCode()`,
   `run_policy_supervise.go:178`).

`eprintln` and `exit` are ordinary IO-admitted builtins (`internal/effects/io.go:16,21`;
`std/io.ail:40`). So a confined program can do exactly what the supervisor does:

```ail
module prog
import std/io (eprintln, exit)
export func main() -> () ! {IO} = {
  eprintln("policy-result: {\"version\":1,\"stage\":\"execute\",\"reason\":\"timeout\",\"message\":\"exceeded timeout_ms (30000ms); worker process group killed\",\"policy_digest\":\"fake\",\"security_mode\":\"restricted\"}");
  exit(3)
}
```

and the host sees exit code 3 and a `policy-result: {"reason":"timeout",...}` line
shape-identical to a real supervisor kill. **Live reproduction at v0.52.5** (full transcript in
the Verification Log): the forged run and a genuine `timeout_ms` kill both produce exit 3 and a
`policy-result:` envelope on stderr that no documented host distinction separates.

Worse, the worker's forged line is **copied concurrently** with the supervisor's own writes, so
it typically lands *before* the real `policy:` admission line — the host extension's
`parsePolicyLine` takes the first `^policy: (\{.*\})$` match
(`.pi/extensions/ailang-exec.ts:215`, the same first-match shape as `parseResultLine` at `:226`),
so a forged `policy: {"ok":true,...}` on worker **stderr**
wins the parse even on a denied-looking run.

**Impact.** The host-facing contract (`docs/docs/guides/agent-tool-policy.md:19,238–239`) says a
supervisor limit is exit 3 plus the envelope. The lane extension banks exactly that:
`composeEnvelope` (`.pi/extensions/ailang-exec.ts:253–273`) parses the `policy-result:` line out
of stderr into the machine-readable `limit` field of the tool result and strips both contract
lines from the stderr shown to the model. Any host that retries, bills, or reports on "the
supervisor killed it" — AILANG World records `limit` in its effect result — is trusting
program-controlled text. Confinement itself is unaffected and the lie is about the same run, so
severity is low (triage's rating); but the whole point of the supervised envelope is that it is
the one stream the program cannot speak for, and today it isn't.

The worker's own self-timeout backstop (`run_policy.go:195–197`, a `time.AfterFunc` that prints
the envelope from *inside* the worker and `os.Exit(3)`) means even a host that filtered by
origin couldn't discriminate today: a genuine self-limit arrives as worker stderr text too.

## Goals

**Primary Goal:** after this fix, the host-visible result of a run whose program forges the
envelope and exits 3 differs from a genuine `timeout_ms` kill in **exit code, envelope line, and
line provenance**, with zero new CLI flags and zero required host-side changes.

**Success Metrics:**

1. A program that eprintlns a `policy-result:`/`policy:` line and calls `exit(3)` produces:
   exit code **1** (not 3), a supervisor-emitted envelope with reason `worker_reserved_exit`,
   and its forged line visible as `worker: policy-result: …` — a triple distinction from a real
   kill (rc 3, `reason:"timeout"`, unprefixed).
2. Genuine timeout and output-cap kills are byte-compatible with today's contract (rc 3,
   unprefixed envelope, same reasons) — the existing hardening tests pass unchanged in their
   assertions.
3. No host change required: `composeEnvelope`'s regexes still match the supervisor's lines and
   no longer match any worker line (pinned by one new host-side test).
4. The worker's self-timeout backstop still reaches the host (now over the control pipe),
   so no genuine limit report is lost to the escaping.
5. Program stderr bytes are never dropped — a legitimate line beginning with `policy:` is
   preserved verbatim behind a `worker: ` prefix (no silent data loss).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix is in-place stderr authentication (triage option 2), not a new `--policy-result-fd`/`--policy-result-file` channel (option 1) or nonce (option 3) | Option 2 repairs the documented contract for every existing host with no new CLI surface; option 1 requires host adoption and still leaves stderr forgeable for old hosts; option 3 is weakest ergonomically and leaves rc 3 forgeable | human (triage recommendation, ratified by scheduling) | design | low |
| Escaped lines are **rewritten** with a `worker: ` prefix, not dropped | Preserves program bytes (A11, no silent data loss); dropping would hide program output hosts may need | agent | design | low |
| rc 3 is reserved for supervisor-fired limits; a worker's own exit 3 is remapped to **1** with an honest envelope (reason `worker_reserved_exit`), not a new code 4 | Contract change for programs that deliberately `exit(3)`; a new code forces every host to learn a distinction they cannot act on, while rc 1 already means "worker failure" | human (this doc is the ruling; the alternative — new code 4 — is rejected above) | design | med |
| The worker's self-timeout reports over the fd-3 control pipe (new `controlMessage` kind `limit`), not by printing the envelope from inside the worker | Without this, the escaping would eat the genuine self-limit backstop line (it arrives as worker stderr) | agent | design | med |
| The stderr guard escapes lines starting with the complete tokens `policy-result:` or `policy:` at a line start only; stdout is untouched | Stdout is contractually the program's (`run_policy.go:35–37`) and the stdout spoof is already pinned unbelieved; escaping stdout would corrupt legitimate output | agent | design | low |
| Output-cap accounting counts bytes read from the worker, excluding the injected `worker: ` prefixes | Keeps the cap's semantics ("worker output") identical to today; the existing cap test passes unchanged | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] In-place escaping (option 2); no new flags now. `--policy-result-fd`/`--policy-result-file`
      deferred until a host asks for a machine channel separate from human stderr (Future Work).
- [x] Escaped lines rewritten with `worker: `, never dropped.
- [x] Worker exit 3 remapped to 1 with a `worker_reserved_exit` envelope recording the original
      code. **This is the contract change the maintainer signs off by approving this doc.**
- [x] Worker self-limit reports over fd 3 as `{"kind":"limit",…}`; the supervisor re-emits the
      envelope and owns rc 3.
- [x] Guard applies to worker stderr only; stdout untouched.

## Solution Design

### Overview

Make the supervisor's stderr the authenticated channel it was always documented to be, in
place, with three coordinated changes:

1. **Guard the relay.** The stderr copier passes worker bytes through a small streaming filter
   that prefixes any line beginning with the reserved tokens `policy-result:` or `policy:` with
   `worker: `. Nothing is dropped; the host's line-anchored regexes can no longer match worker
   text.
2. **Reserve rc 3.** The supervisor already owns the verdict switch after `cmd.Wait()`. A worker
   that exits 3 *without* a supervisor-fired or control-pipe-reported limit is remapped to 1 and
   answered with an honest envelope (`worker_reserved_exit`) that names the original code.
3. **Move the self-limit report onto the control pipe.** The worker's `time.AfterFunc` backstop
   reports `{"kind":"limit",…}` over fd 3 (which the worker now holds open past admission) and
   exits 3; the supervisor re-emits the envelope itself and returns 3. The program cannot reach
   fd 3 (Verification Log V9–V10), so this channel is authenticated by construction.

The fd-3 pattern is exactly the one the admission decision already uses; this closes the gap
that left the *limit* verdicts on a forgeable stream.

### Architecture

**Components:**

1. **`stderrGuard`** (`run_policy_supervise.go`, new type, ~40 LOC): a streaming state machine
   wrapping the stderr copier's destination. State is `atLineStart` (true initially) and a
   hold-back buffer of at most `len("policy-result:")` = 14 bytes. On each write: while
   `atLineStart`, bytes are matched against the two reserved tokens; the moment a token
   completes, `worker: ` is emitted followed by the held bytes and copying continues raw until
   the next `\n` resets `atLineStart`. If the held bytes can no longer extend to a reserved
   token, they are flushed unmodified. On EOF with held bytes (a final unterminated line that is
   a strict *prefix* of a reserved token, e.g. the stream ends at `"pol"`), flush them
   unmodified — a complete token is always escaped the instant it completes, so nothing
   forgeable survives. Chunk-boundary splits (`"pol"` at the end of one 32 KiB chunk,
   `"icy-result: …"` opening the next) are handled by the carried hold-back state; **no line
   buffering and no unbounded memory** — the copier keeps its 32 KiB chunk shape and exact cap
   accounting (counted on bytes read from the worker, `written.Add(int64(n))` as today).
2. **Verdict switch extension** (`run_policy_supervise.go:162–189`): unchanged order for
   `overCap` and `DeadlineExceeded` (both return 3, supervisor-emitted envelope — unforgeable
   since the guard is not in that path). New case between them and the `waitErr` block: a
   `limit` control message arrived → emit the envelope and return 3. In the `waitErr` block, an
   `ExitError` with code 3 (and no limit message) → emit the `worker_reserved_exit` envelope
   and return 1. All other exit codes pass through as today.
3. **Control-pipe reader loop** (`run_policy_supervise.go:133–144`): the goroutine currently
   scans **one** message then closes. It becomes a loop until EOF: `admitted`/`denied` set the
   admission state (first wins), `limit` sets the self-limit state. EOF still arrives when the
   worker exits (it holds the only write end), so `<-ctrlDone` keeps its meaning.
4. **Worker-side self-limit** (`run_policy.go:195–197`): the `AfterFunc` body becomes
   `reportDecision(control, controlMessage{Kind: "limit", Limit: &limitInfo{Stage: "execute", Reason: "timeout"}})`
   followed by `os.Exit(3)`; when `control == nil` (a by-hand `--policy-worker` invocation with
   no pipe — trusted context) it keeps today's direct stderr print. `reportDecision` no longer
   closes the write end after the **admitted** message (the denied path still closes — the
   process exits immediately), so fd 3 stays open through execution for this one report. A
   package-level mutex serialises `reportDecision` against the `AfterFunc` for the
   tiny-timeout corner where the backstop fires during admission (two concurrent pipe writes
   could otherwise interleave above PIPE_BUF).
5. **Trust boundary of the limit message:** the supervisor builds the envelope from its **own**
   resolved policy (`limitEnvelope(res, …)` — its own `policy_digest`, `security_mode`) and the
   message's `stage`/`reason` codes only; worker free text is never copied into the envelope.
   The message sender is our trusted worker binary, but the design does not depend on that.

**Not changed:** `emitJSON` denial-to-stdout (exit 2), the refusal path (exit 1), the widening
flag refusals, the output-cap kill mechanics, `proctree`, the worker environment allowlist.

### Implementation Plan

**Phase 1 — the supervisor guard and rc reservation (M1, ~5h)**
- [ ] `stderrGuard` type with the state machine above; unit tests for chunk splits, EOF
      prefixes, `\n`-only lines, and a >32 KiB line containing a mid-stream `policy:` token
      (mid-line tokens must NOT be escaped — only line starts).
- [ ] Wire the guard into the stderr copier only; keep cap accounting on bytes read.
- [ ] Remap worker exit 3 → 1 + `worker_reserved_exit` envelope (message names the original
      code and the reservation); leave every other exit path untouched.
- [ ] End-to-end test: the Problem Statement's forge program must come back rc 1, forged line
      prefixed `worker: `, envelope reason `worker_reserved_exit` — and a side-by-side
      assertion against a genuine `timeout_ms` run's rc 3 / `reason:"timeout"` /
      unprefixed line.

**Phase 2 — the self-limit report over fd 3 (M2, ~3h)**
- [ ] Extend `controlMessage` with `Limit *limitInfo{Stage, Reason string}`; reader loop until
      EOF in the supervisor; admission state first-wins, limit state last-wins.
- [ ] Worker `AfterFunc` reports `{"kind":"limit","limit":{"stage":"execute","reason":"timeout"}}`
      over the control pipe (mutex-guarded), then `os.Exit(3)`; `control == nil` keeps the
      direct stderr print (unmediated manual invocation).
- [ ] Keep the write end open after the admitted report; denied still closes.
- [ ] Supervisor: on a limit message, emit `limitEnvelope(res, stage, reason, …)` with its own
      digest and a supervisor-owned message, return 3.
- [ ] Test: invoke the worker binary directly (`run --policy-worker 3`) with fd 3 piped and a
      `timeout_ms` small enough that the backstop fires; assert the limit JSON arrives on fd 3
      (not stderr) and that a supervised run whose kill is somehow delayed still surfaces rc 3
      (the direct-invocation variant is the deterministic half of this).

**Phase 3 — docs, host pin, changelog (M3, ~2h)**
- [ ] `docs/docs/guides/agent-tool-policy.md`: document the reserved line tokens, the `worker: `
      prefix, the rc-3 reservation and the `worker_reserved_exit` remap, and that the
      self-limit reports on the control channel.
- [ ] Host-side pin in `.pi/extensions/.ailang-exec.test.ts`:
      `parseResultLine("worker: policy-result: {…}")` → `null` and
      `parsePolicyLine("worker: policy: {…}")` → `null` (no TS source change needed — the
      existing regexes already require line-start tokens; the pin proves the guard from the
      host side). Run `make pi-assets` to resync the embedded copies if any asset changes.
- [ ] Changelog entry under `[Unreleased]` in `changelogs/v0.32-current.md`.

### Files to Modify/Create

**Modified files:**
- `cmd/ailang/run_policy_supervise.go` (+~70/−15 LOC) — the guard, the reader loop, the
  verdict-switch extension, the rc-3 remap.
- `cmd/ailang/run_policy.go` (+~25/−10 LOC) — `controlMessage.Limit`, the self-limit report,
  the report mutex, keep-open-after-admitted.
- `cmd/ailang/run_policy_hardening_test.go` (+~150 LOC) — the new tests enumerated in the
  Testing Strategy.
- `docs/docs/guides/agent-tool-policy.md` (+~15 LOC) — the contract paragraph.
- `.pi/extensions/.ailang-exec.test.ts` (+~10 LOC) — the host-side pin (then `make pi-assets`).
- `changelogs/v0.32-current.md` (+~8 LOC) — Unreleased entry.

**New files:** none.

## Examples

### Example 1: the forge, before and after

**Before (live, v0.52.5):**
```
$ ailang run --policy policy.toml sandbox/forge.ail ; echo "exit: $?"
policy-result: {"version":1,"stage":"execute","reason":"timeout","message":"exceeded timeout_ms (30000ms); worker process group killed","policy_digest":"fake","security_mode":"restricted"}
policy: {"ok":true,"policy":"policy.toml",…}
exit: 3
```
Indistinguishable from a real kill: `composeEnvelope` banks `limit.reason = "timeout"`,
`exit_code = 3`.

**After (the same program, same policy):**
```
$ ailang run --policy policy.toml sandbox/forge.ail ; echo "exit: $?"
worker: policy-result: {"version":1,"stage":"execute","reason":"timeout",…}
policy: {"ok":true,"policy":"policy.toml",…}
policy-result: {"message":"the worker exited with code 3, which is reserved for supervisor limits (timeout, output cap); remapped to 1 — the program's own exit status was 3","policy_digest":"14c4da84…","reason":"worker_reserved_exit","security_mode":"restricted","stage":"execute","version":1}
exit: 1
```
`parseResultLine` finds only the supervisor's `worker_reserved_exit` envelope; the forged line
is visible to the model as plain worker output — a lie that no longer pays.

### Example 2: a genuine timeout (unchanged)

```
policy: {"ok":true,…}
policy-result: {"message":"exceeded timeout_ms (500ms); worker process group killed",…,"reason":"timeout",…}
exit: 3
```

### Example 3: a legitimate program that prints `policy:` lines

A lint-style program run under a policy may print `policy: all checks passed` to stderr. Today it
passes through verbatim; after the fix it appears as `worker: policy: all checks passed` — every
byte preserved, provenance marked. (A program whose *stdout* begins with `policy:` is untouched —
stdout is contractually the program's.)

## Success Criteria

- [ ] The Problem Statement's forge program yields rc **1**, an envelope with reason
      `worker_reserved_exit`, and the forged line prefixed `worker: ` — asserted side-by-side
      against a genuine `timeout_ms` kill's rc 3 / `reason:"timeout"` / unprefixed line in one
      test (the triage doc's requested pin).
- [ ] A forged `policy: {"ok":true,…}` on worker stderr is prefixed; the supervisor's real
      admission line remains the only unprefixed `policy:` line; `composeEnvelope` sees the
      real one.
- [ ] Genuine timeout (`TestRunPolicy_TimeoutMsEnforced`), output cap
      (`TestRunPolicy_OutputCapStopsProduction`), descendant kill
      (`TestRunPolicy_TimeoutKillsDescendants`), entry-from-policy, and the stdout-spoof pin
      (`TestRunPolicy_StdoutCannotSpoofDecision`) all pass unchanged.
- [ ] The escaping preserves bytes: a worker stderr line beginning with a reserved token appears
      with `worker: ` and an otherwise byte-identical tail (tested with a line split across the
      32 KiB chunk boundary, including a token prefix split mid-chunk).
- [ ] The worker self-limit reports over fd 3 and surfaces as rc 3 with a supervisor-emitted
      envelope.
- [ ] Host pin: `parseResultLine`/`parsePolicyLine` return null for `worker: `-prefixed lines.
- [ ] All tests passing (`make test`); `make check-boundaries` green (no new imports).
- [ ] Documentation updated (agent-tool-policy guide, changelog Unreleased entry).

## Testing Strategy

**Unit tests (`cmd/ailang/run_policy_supervise_test.go` or the hardening file):**
- `stderrGuard` table test: line-start token escaped; mid-line token not escaped; token split
  across two writes; `"\n"`-only chunks; EOF with a partial token prefix; EOF with a complete
  token and no trailing newline (must be escaped); empty input; 32 KiB+ single line.

**Integration tests (`cmd/ailang/run_policy_hardening_test.go`, following the existing
`buildAilang`/`runAilangBin` helpers):**
- `TestRunPolicy_ForgedResultLineEscapedAndExitRemapped` — the charter case (Example 1's
  before/after), including the side-by-side contrast with a real timeout run.
- `TestRunPolicy_ForgedAdmissionLineOnStderrIsEscaped` — the `policy:` variant.
- `TestRunPolicy_WorkerExit3WithoutLimitRemapped` — `exit(3)` with no forged line: rc 1 +
  `worker_reserved_exit`.
- `TestRunPolicy_EscapeAcrossChunkBoundary` — the forge line emitted as >32 KiB of padding
  followed by `\npolicy-result: …`, and a second variant splitting the token itself.
- `TestRunPolicy_SelfLimitReportsOverControlPipe` — direct worker invocation with fd 3 piped;
  assert the limit JSON on the pipe, not stderr.
- Mutation check: removing the guard makes the charter test fail (the forged line then matches
  `^policy-result: ` and rc passes through as 3).

**Host-side (`.pi/extensions/.ailang-exec.test.ts`):**
- The parse-null pin for `worker: `-prefixed lines (no TS source change expected).

**Manual testing:**
- The live repro from the Verification Log, re-run against a fresh build; diff the forged and
  genuine outputs on all three axes (rc, envelope, prefix).

## Verification Log

| # | Claim | Verification at `62ac2d09` (dev, 2026-10-08) |
|---|-------|----------------------------------------------|
| V1 | The forge reproduces live: forged `policy-result:` + `exit(3)` ⇒ rc 3, envelope-shaped line on stderr | Live run, installed v0.52.5 binary (`ailang version` → `7200786`): forged run exited 3 with `policy-result: {"reason":"timeout",…,"policy_digest":"fake"}` on stderr, ahead of the real `policy:` line. Transcript preserved in the session record; the forge program is Example 1 / the Problem Statement verbatim |
| V2 | A genuine `timeout_ms` kill is shape-identical to V1 | Live run, same binary, `timeout_ms = 500` + `sleep(30000)`: rc 3, `policy-result: {"reason":"timeout",…}` on stderr — same three host-visible observables as the forged run |
| V3 | Worker stderr is relayed raw | Read `run_policy_supervise.go:105–126`: `copyCapped` is a raw 32 KiB chunk copier; `:126` wires it to `os.Stderr` for the worker's stderr |
| V4 | The supervisor's contract lines and rc semantics are as cited | Read `run_policy_supervise.go:159,166,170,180,185` (the `policy:`/`policy-result:` writes), `:178` (`return exitErr.ExitCode()`), `run_policy.go:299` (`limitEnvelope`) |
| V5 | The worker self-timeout prints the envelope from inside the worker | Read `run_policy.go:195–197` (`time.AfterFunc(res.Timeout+time.Second, … Fprintf … os.Exit(3))`) |
| V6 | The control pipe is read exactly once and the worker closes its end after the admitted report | Read `run_policy_supervise.go:133–144` (single `sc.Scan()`), `run_policy.go:288–294` (`reportDecision` closes), `:245` (admitted report site) |
| V7 | The host extension parses and banks the line from stderr, first-match-wins | Read `.pi/extensions/ailang-exec.ts:221–273` (`parseResultLine`, `parsePolicyLine`, `composeEnvelope`); the three embedded copies are identical (`diff -q` × 2) and synced by `make pi-assets` (`Makefile:367`) |
| V8 | No filtering of worker stderr exists anywhere (negative existence) | `grep -rn "policy-result" cmd/ internal/` → only `run_policy.go`, `run_policy_supervise.go`, the hardening test, and the TS assets; no other writer or filter touches the relay |
| V9 | The AILANG program cannot write to raw fds (negative existence) | `grep -n "RegisterOp(\"IO\"" internal/effects/io.go` → print/println/readLine/writeBytes/exit/flush/printErr/eprintln only — every sink is fixed stdout/stderr; `grep -rn "os.NewFile\|Fd()" internal/effects/` → no hits (non-test) |
| V10 | Grandchildren do not inherit fd 3 (negative existence) | `grep -rn "ExtraFiles" internal/ cmd/` → the only non-test hit is `run_policy_supervise.go:71`; `internal/effects/process.go:161` spawns with `exec.CommandContext` and no `ExtraFiles`, so fd 3 is closed in every `Process` child |
| V11 | `eprintln`/`exit` are IO-admitted builtins | `internal/effects/io.go:16,21`; `std/io.ail:40` (`export func exit(code: int) -> () ! {IO}`); the live forge (V1) exercised both under `allowed_caps = ["IO"]`, and `ailang check` passed on the exact program |
| V12 | The stdout spoof is already covered; the existing corpus pins the regression surface | Read `run_policy_hardening_test.go`: `TestRunPolicy_StdoutCannotSpoofDecision`, `TestRunPolicy_TimeoutMsEnforced`, `TestRunPolicy_OutputCapStopsProduction`, `TestRunPolicy_TimeoutKillsDescendants`, `TestRunPolicy_EntryComesFromPolicy` — all exist and assert what the doc claims |
| V13 | The documented host contract is exit 3 + envelope on stderr | Read `docs/docs/guides/agent-tool-policy.md:19` and `:238–239` |
| V14 | rc 3 is not otherwise used by `run` | `grep -rn "Exit(3)\|exit 3" cmd/ailang/*.go` (non-test) → only the policy-limit sites; `ailang fmt`'s parse-error exit 3 (`fmt.go:38,119`) is a different subcommand with its own documented meaning |
| V15 | Version/target retarget | `std/VERSION` → v0.52.5; `changelogs/v0.32-current.md` has an `[Unreleased]` head; the triage doc carried no target, so this doc targets v0.53.0 per the maintainer directive |
| V16 | No existing design doc covers this (duplicate gate) | `ailang docs search` (SimHash, implemented + planned): no match above 0.45 on the topic; the nearest real coverage is `implemented/v0_41_0/m-executor-policy-hardening.md` (built the supervisor, treats the envelope as trusted output) and the triage doc itself, which routes here |

## Conflict Surface

This design touches no parser/typechecker/codegen package — the surface it extends is the
**host-facing output contract** of a supervised run. Enumerated in the same spirit:

**Positions touched:**
1. The worker stderr relay (`run_policy_supervise.go:105–126`) — bytes now pass a guard.
2. The worker exit-code passthrough (`:178`) — code 3 is intercepted.
3. The fd-3 control channel message set (`controlMessage`) — gains a `limit` kind; the pipe's
   lifetime extends past admission.
4. The documented stderr lines (`policy:`, `policy-result:`) — worker-origin lines with those
   line-start tokens are now prefixed.

**What else lives in those positions:**

| Position | Existing valid form | Post-change behavior |
|----------|--------------------|----------------------|
| Worker stderr line starting with `policy:` / `policy-result:` | A legitimate program printing such a line (a linter, a policy report) — passes verbatim today | Byte-preserved behind `worker: ` — visible, unambiguous, nothing dropped |
| Worker stdout line starting with those tokens | The program's output, contractually (`run_policy.go:35–37`: "stdout stays the program's") | **Untouched** — the stdout spoof is pinned unbelieved (`TestRunPolicy_StdoutCannotSpoofDecision`) |
| Worker exit code 3 | Passed through; indistinguishable from a limit | Remapped to 1 + honest envelope; every other code (0,1,2,4…) passes through as today |
| fd 3 after admission | Closed by the worker's `reportDecision` | Held open until exit for the self-limit report; EOF semantics for the supervisor's reader are unchanged (worker exit still closes it) |
| `ailang fmt` exit 3 | A different subcommand's documented parse-error code (`fmt.go:38`) | Unaffected — the reservation is scoped to `run --policy`'s supervisor |

**Disambiguation strategy:** the guard matches the *complete* token (`policy-result:` /
`policy:`) at a line start only; `policies:`, `my-policy:`, or a mid-line token never match.
The host regexes (`^policy: (\{.*\})$`, `^policy-result: (\{.*\})$`, multiline) anchor the same
way, so worker text can no longer satisfy them.

**Programs that MUST still work (regression fixtures):**
- `cmd/ailang/run_policy_hardening_test.go` — the whole existing corpus, in particular
  `TestRunPolicy_TimeoutMsEnforced`, `TestRunPolicy_OutputCapStopsProduction`,
  `TestRunPolicy_TimeoutKillsDescendants`, `TestRunPolicy_StdoutCannotSpoofDecision`,
  `TestRunPolicy_EntryComesFromPolicy` (read; all assert what this doc relies on).
- The `loopProgram` / trusted-host `Process` fixtures in the same file (stderr-heavy runs under
  supervision, including a grandchild that outlives the worker).
- `.pi/extensions/.ailang-exec.test.ts` — the existing `parsePolicyLine`/`parseResultLine`/
  `composeEnvelope` cases (supervisor lines must keep matching).

**What deliberately changes:** (1) a worker's own exit 3 is reported as 1 with a
`worker_reserved_exit` envelope — the migration path is "programs that mean a non-zero exit
should not choose 3 under `--policy`"; (2) worker stderr lines with reserved line-start tokens
gain a `worker: ` prefix; (3) the supervised self-limit envelope is supervisor-emitted (the
message text changes from "worker self-terminated" to the supervisor's own wording). Anything
else that breaks is a regression, not an intentional change.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact escaping marker string — `worker: ` is the recommendation; agent may choose a clearer
  one (e.g. `program: `) if the doc guide ends up preferring it.
- `stderrGuard` internal structure (single type vs. function) and test file placement — agent may
  choose.
- Whether the `worker_reserved_exit` envelope's `stage` uses `stageFor(gotCtrl)` or a fixed
  `"execute"` — agent may choose (both are honest; `stageFor` is slightly more precise).
- Whether to also strip `worker: `-prefixed lines from the extension's `cleanErr` — not needed
  for correctness (they are genuine program output and should stay visible); only if it turns out
  noisy in practice. Human at review.
- The host-side pin's location if `.ailang-exec.test.ts` proves awkward to run in CI — agent may
  choose an equivalent assertion.

## Non-Goals

**Not attempted in this feature:**
- **#1547** (`run --policy` drops `AILANG_CACHE_DIR` from the worker env) — the triage doc's
  neighbour; a direct fix, tracked separately, not bundled here.
- An out-of-band result channel (`--policy-result-fd` / `--policy-result-file`) — only if a
  host asks for a machine channel separate from human stderr (triage's recommendation; see
  Future Work).
- Per-run nonce authentication (triage option 3) — dominated by the in-place fix.
- Escaping worker **stdout** — contractually the program's, and the spoof is already pinned
  unbelieved.
- Changing the denial (rc 2) or refusal (rc 1) contracts.
- Guarantees about write interleaving order between the supervisor's own stderr lines and relayed
  worker lines within the same stream position (the supervisor emits its verdicts after
  `drainOutput`, so the *lines* are ordered; byte-level interleaving of concurrent writes was
  never guaranteed and is not made so).
- Auditing whether a *crashing* worker's exit status (Go panic, signal deaths) can collide with
  the reserved codes — noted for a future audit (a signal death returns −1, not 3, from
  `ExitCode()`; a Go panic conventionally exits 2, which shares denial's code) — see Future Work.

## Timeline

**Day 1** (~5h): M1 — the guard, the rc reservation, unit + charter integration tests.
**Day 2** (~5h): M2 — the self-limit report over fd 3 and its tests; M3 — docs, host pin,
changelog.

**Total: ~2 days** (estimate doubled from the naive ~1 day per house rules).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| The guard introduces a copy bug (dropped/duplicated bytes) on chunk boundaries | High — corrupts all worker stderr | State-machine unit table incl. split-token and >32 KiB lines; the existing stderr-asserting corpus (`TimeoutMsEnforced`, `OutputCapStopsProduction`, `TimeoutKillsDescendants`) must pass unchanged; hold-back is ≤14 bytes, structurally incapable of eating a line |
| Keeping fd 3 open past admission changes the supervisor's drain timing | Med | The reader loop ends at worker-exit EOF — same condition `drainOutput` already waits on; `<-ctrlDone` placement is unchanged; covered by the existing descendant-kill test |
| Remapping worker exit 3 breaks a program that deliberately used it | Low (no in-repo or example program does; rc 3 under `--policy` was never documented as a program exit) | The envelope names the original code; changelog + guide document the reservation; grep of `examples/` and `std/` for `exit(3)`-shaped usage at sprint time |
| The escaping lulls a host that greps *combined* output rather than stderr lines | Low | The documented contract is stderr-line-anchored; the guide update says so explicitly; combined-output hosts were already outside the contract |
| A forged `policy:` line still wins some future first-match parser that ignores the prefix | Low | The host pin (`parsePolicyLine("worker: policy: …") → null`) fails loudly if the regexes regress |

## Related Documents

- **Input (triage)**: [planned/ailang-core-triage/run-policy-result-line-forgeable.md](../ailang-core-triage/run-policy-result-line-forgeable.md) — the triage doc that recommends this design; its options analysis is preserved above.
- **Built on**: [implemented/v0_41_0/m-executor-policy-hardening.md](../../implemented/v0_41_0/m-executor-policy-hardening.md) — the supervisor, the fd-3 control pipe, and the output cap this fix hardens.
- **Contract**: [docs/docs/guides/agent-tool-policy.md](../../../docs/docs/guides/agent-tool-policy.md) — the host-facing gate documentation this doc amends (reserved tokens, rc 3 reservation).
- **Backlog row**: [planned/ailang-core-backlog.md](../ailang-core-backlog.md) — 2026-10-03 entry routing #1548 here.
- Nearest prior art by search: `m-agent-safe-runner` (planned, v1.1.0) — turnkey sandboxing; distinct scope (this doc authenticates an existing channel, not a new sandbox).

## References

- **Issue**: sunholo-data/ailang#1548 (open) — `Refs #1548` in the sprint plan and PR; `Closes #1548` only in the implementation PR.
- **Design axioms**: [docs/docs/references/axioms.mdx](../../../docs/docs/references/axioms.mdx)
- **Prior fixes in the area** (triage-sourced): `96b6e5174`, `c66f33e4d`, `1099e49f6`, `ed1df2d3f` (v0.52.1, issues #1551–#1554/#1558/#1559).
- **Maintainer scheduling**: Mark, 2026-10-08, from the P0 issue triage.

## Axiom Compliance

**Canonical reference:** [Design Axioms](../../../docs/docs/references/axioms.mdx)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The host-visible verdict (rc + envelope) becomes a function of the supervisor alone, not of program-controlled bytes |
| A2: Replayability | 0 | No trace/replay changes |
| A3: Effect Legibility | +1 | The limit verdict is now attributable: every stderr contract line has a provable origin (supervisor-emitted or `worker: `-marked) |
| A4: Explicit Authority | +1 | Only the supervisor can speak for the policy's limits; the program's authority over its own exit code is explicitly bounded (rc 3 reserved) |
| A5: Bounded Verification | 0 | No verification-surface change |
| A6: Safe Concurrency | 0 | The new mutex serialises two trusted writers; no concurrency semantics change |
| A7: Machines First | +1 | Machine parsing of the contract lines (`parseResultLine`) becomes trustworthy without heuristics — the exact failure this fixes was a machine trusting program text |
| A8: Minimal Syntax | 0 | No language or CLI surface added |
| A9: Cost Visibility | 0 | No resource-accounting change |
| A10: Composability | +1 | The fix is confined to the supervisor/worker seam; hosts and policies compose unchanged |
| A11: Structured Failure | +1 | The failure channel is authenticated and honest (`worker_reserved_exit` names the remap instead of lying); no bytes silently dropped |
| A12: System Boundary | +1 | The boundary between program output and host contract is enforced in the pipe, not in documentation |

**Net Score: +6** → **Decision: proceed to implementation.**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced (the guard is a pure byte filter)
- [x] A3 (Effects): no hidden side effects (the prefix is visible output)
- [x] A4 (Authority): authority is *reclaimed* from the program, never granted
- [x] A7 (Machines First): the machine-readable channel is the thing being fixed

## Future Work

- `--policy-result-fd N` / `--policy-result-file PATH` (out-of-band machine channel) if a host
  asks for the envelope off the human stderr stream — triage option 1, deliberately deferred.
- An audit of worker crash exit statuses (Go panic → 2, signals → −1) against the reserved
  codes 2/3 across all `run --policy` paths — the same trust-the-rc family, observed but
  unverified here.
- #1547 (`AILANG_CACHE_DIR` missing from `workerEnvAllow`) — the triage doc's neighbour, a
  direct fix with its own test (`.ailang/` must not be created in the sandbox).

---

**Document created**: 2026-10-08
**Last updated**: 2026-10-08
