# M-RUN-POLICY-RESULT-UNFORGEABLE: the confined program cannot forge the supervisor's `policy-result:` line (#1548)

**Status**: Planned
**Target**: v0.53.0 (triage doc carried no target; retargeted per maintainer scheduling, 2026-10-08)
**Priority**: P0 (maintainer-approved scheduling from the P0 issue triage, 2026-10-08; triage severity: bug, security, low)
**Estimated**: 2 days (M1 6h, M2 4h, plus buffer)
**Dependencies**: None. Builds on the supervisor and the fd-3 control pipe shipped by
`m-executor-policy-hardening` (implemented v0.41.0) and the 2026-10-03 policy fixes
(v0.52.1: `96b6e5174`, `c66f33e4d`, `1099e49f6`, `ed1df2d3f`), which closed #1551–#1554/#1558/#1559
but not this one.
**Issue**: Refs #1548 (sunholo-data/ailang#1548, bug / security, open). Do not open a new issue —
this doc is the existing issue's design deliverable. `Refs #1548` goes in the sprint plan and the PR;
`Closes #1548` goes only in the implementation PR.

---

## Revision 2026-10-08 (review)

A review against origin/dev found two remaining forge paths and one regression in the first
draft. This revision changes the following and keeps every other decision (in-place
authentication, the `worker: ` prefix, rc 3 reserved and remapped to 1, stdout untouched).

1. **The guard recognises every line start a host parser may see, not only `\n`.** The host
   regexes (`.pi/extensions/ailang-exec.ts:215,226`) use JavaScript's `m` flag, which also
   treats `\r`, U+2028 and U+2029 as line starts: node confirms
   `/^policy-result: (\{.*\})\s*$/m` matches `"junk\rpolicy-result: {…}"` (V17). Forged lines
   are relayed before the supervisor's lines and the first match wins, so a `\n`-only guard
   still let a program forge `limit` and, on an admitted run, `policy: {"ok":true,…}`. The
   guard now escapes after `\n`, `\r`, and the UTF-8 sequences of U+2028/U+2029 (plus the other
   `str.splitlines` boundaries, see Architecture §1).
2. **The supervisor's own lines always start a fresh line.** An unterminated last worker stderr
   line glued the supervisor's `policy: {…}` (`run_policy_supervise.go:159`) onto it, so the
   line-anchored regex missed it and `composeEnvelope` took the not-admitted branch, parsing
   stdout as the decision (`ailang-exec.ts:268–275`): a program-forged denial (V18). The
   supervisor now writes `\n` before its own line whenever the relay is not at a `\n` line start.
3. **The in-repo host is hardened as defence in depth.** `parsePolicyLine`/`parseResultLine`
   split stderr on `\n` only and take the **last** matching line (the supervisor's lines are
   written after the relay drains, so they are always last). The guard alone closes the hole
   for third-party hosts; the host change makes either layer sufficient on its own.
4. **The first draft's Phase 2 (keeping fd 3 open for a self-limit report) is dropped.** It is dead in supervised
   runs: the supervisor's deadline (`run_policy_supervise.go:61`, `res.Timeout`) always expires
   before the worker backstop (`run_policy.go:195`, `res.Timeout+1s`), so the
   `DeadlineExceeded` case at `:169` wins the verdict switch whatever the worker prints or
   returns (V19). Keeping fd 3 open would also be a regression: on Linux a `trusted_host`
   `Process` child could write to `/proc/<worker>/fd/3` (V20). The worker keeps today's
   close-at-admission; the backstop keeps its stderr print, which the guard now prefixes.
5. **Regression tests added** for the `\r`, U+2028/U+2029 and unterminated-line variants, beside
   the forge-vs-real-timeout charter test, plus host-side tests for the split-and-last parse.
6. **Compatibility with sibling PR #1664** (#1547, worker `AILANG_CACHE_DIR`): its new supervisor
   warning line must not start with `policy`, and must use the same fresh-line rule (Non-Goals).

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
`parsePolicyLine` takes the first `/^policy: (\{.*\})\s*$/m` match
(`.pi/extensions/ailang-exec.ts:215`, the same first-match shape as `parseResultLine` at `:226`),
so a forged `policy: {"ok":true,...}` on worker **stderr**
wins the parse even on a denied-looking run. Two details of that parse shape the fix:
JavaScript's `m` flag makes `^` match after `\r`, U+2028 and U+2029 as well as `\n` (V17), and
a worker's unterminated last stderr line glues the supervisor's own `policy:` line onto it, so
the anchored regex misses the real line entirely (V18).

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
the envelope from *inside* the worker and `os.Exit(3)`) prints on worker stderr too. In a
supervised run it never decides the verdict: the supervisor's deadline (`res.Timeout`, armed
before the worker starts, `run_policy_supervise.go:61`) always expires first, and the
`DeadlineExceeded` case (`:169`) answers rc 3 with the supervisor's own envelope regardless of
what the worker printed or returned (V19). The backstop only speaks for a by-hand
`--policy-worker` invocation, which is a trusted context.

## Goals

**Primary Goal:** after this fix, the host-visible result of a run whose program forges the
envelope and exits 3 differs from a genuine `timeout_ms` kill in **exit code, envelope line, and
line provenance**, with zero new CLI flags and no change required of third-party hosts (the
in-repo extension is hardened as defence in depth).

**Success Metrics:**

1. A program that eprintlns a `policy-result:`/`policy:` line and calls `exit(3)` produces:
   exit code **1** (not 3), a supervisor-emitted envelope with reason `worker_reserved_exit`,
   and its forged line visible as `worker: policy-result: …` — a triple distinction from a real
   kill (rc 3, `reason:"timeout"`, unprefixed).
2. Genuine timeout and output-cap kills are byte-compatible with today's contract (rc 3,
   unprefixed envelope, same reasons) — the existing hardening tests pass unchanged in their
   assertions.
3. No host change required: today's `m`-flag regexes in `composeEnvelope` still match the
   supervisor's lines and no longer match any worker line, whatever line terminator the
   program uses (`\n`, `\r`, U+2028, U+2029) and whether or not its last line is terminated.
   The in-repo extension additionally splits on `\n` and takes the last match (pinned by
   host-side tests).
4. Genuine limit reports are unaffected: supervised timeouts are decided by the supervisor's
   own deadline (V19), so prefixing the worker backstop's stderr line loses nothing.
5. Program stderr bytes are never dropped — a legitimate line beginning with `policy:` is
   preserved verbatim behind a `worker: ` prefix (no silent data loss).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Fix is in-place stderr authentication (triage option 2), not a new `--policy-result-fd`/`--policy-result-file` channel (option 1) or nonce (option 3) | Option 2 repairs the documented contract for every existing host with no new CLI surface; option 1 requires host adoption and still leaves stderr forgeable for old hosts; option 3 is weakest ergonomically and leaves rc 3 forgeable | human (triage recommendation, ratified by scheduling) | design | low |
| Escaped lines are **rewritten** with a `worker: ` prefix, not dropped | Preserves program bytes (A11, no silent data loss); dropping would hide program output hosts may need | agent | design | low |
| rc 3 is reserved for supervisor-fired limits; a worker's own exit 3 is remapped to **1** with an honest envelope (reason `worker_reserved_exit`), not a new code 4 | Contract change for programs that deliberately `exit(3)`; a new code forces every host to learn a distinction they cannot act on, while rc 1 already means "worker failure" | human (this doc is the ruling; the alternative — new code 4 — is rejected above) | design | med |
| fd 3 stays closed at admission (today's behaviour); the worker backstop keeps its stderr print, which the guard prefixes. *(Revised: the first draft moved the backstop onto a held-open fd 3.)* | The backstop never decides a supervised verdict (V19), and an open fd 3 during execution is reachable via `/proc/<worker>/fd/3` from a `trusted_host` `Process` child on Linux (V20) | review | design | low |
| The stderr guard escapes lines starting with the complete tokens `policy-result:` or `policy:` at a line start only, where a line start is the stream start or the byte after **any** line terminator a host parser may honour (`\n`, `\r`, U+2028, U+2029, and the other `str.splitlines` boundaries); stdout is untouched | Stdout is contractually the program's (`run_policy.go:35–37`) and the stdout spoof is already pinned unbelieved; a `\n`-only notion of line start is forgeable through JS `m`-flag regexes (V17) | review | design | low |
| The supervisor writes `\n` before each of its own stderr lines unless the relay's last byte was `\n` | Otherwise an unterminated last worker line swallows the supervisor's `policy:` line and the host falls back to parsing stdout as the decision (V18) | review | design | low |
| The in-repo host parsers split on `\n` only and take the last match | Defence in depth: either the guard or the host parse alone defeats the forge | review | design | low |
| Output-cap accounting counts bytes read from the worker, excluding the injected `worker: ` prefixes | Keeps the cap's semantics ("worker output") identical to today; the existing cap test passes unchanged | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] In-place escaping (option 2); no new flags now. `--policy-result-fd`/`--policy-result-file`
      deferred until a host asks for a machine channel separate from human stderr (Future Work).
- [x] Escaped lines rewritten with `worker: `, never dropped.
- [x] Worker exit 3 remapped to 1 with a `worker_reserved_exit` envelope recording the original
      code. **This is the contract change the maintainer signs off by approving this doc.**
- [x] fd 3 closed at admission, as today; no `limit` control message (revision item 4).
- [x] Guard applies to worker stderr only; stdout untouched. Line starts include `\r`, U+2028,
      U+2029 and the other `splitlines` boundaries (revision item 1).
- [x] Supervisor lines always begin on a fresh `\n` line (revision item 2).

## Solution Design

### Overview

Make the supervisor's stderr the authenticated channel it was always documented to be, in
place, with three supervisor changes and one host hardening:

1. **Guard the relay.** The stderr copier passes worker bytes through a small streaming filter
   that prefixes any line beginning with the reserved tokens `policy-result:` or `policy:` with
   `worker: `, where "line" begins after any line terminator a host parser may honour, not
   only `\n`. Nothing is dropped; the host's line-anchored regexes can no longer match worker
   text.
2. **Reserve rc 3.** The supervisor already owns the verdict switch after `cmd.Wait()`. A worker
   that exits 3 *without* a supervisor-fired limit is remapped to 1 and answered with an honest
   envelope (`worker_reserved_exit`) that names the original code.
3. **Start supervisor lines on a fresh line.** Before writing `policy:` or `policy-result:`, the
   supervisor writes `\n` if the relayed stderr did not end in `\n`, so a program cannot glue
   the real line onto its own unterminated text.
4. **Host defence in depth.** The in-repo extension's parsers split stderr on `\n` only and take
   the last matching line, which is always the supervisor's.

The control pipe is unchanged: one admission message, closed at admission. The admission
decision already travels there; the limit verdicts are decided by the supervisor itself
(`overCap`, its own deadline) and only needed their stderr lines authenticated.

### Architecture

**Components:**

1. **`stderrGuard`** (`run_policy_supervise.go`, new type, ~50 LOC): a streaming state machine
   wrapping the stderr copier's destination. State is `atLineStart` (true initially), a
   partial-terminator state for multi-byte terminators, and a hold-back buffer of at most
   `len("policy-result:")` = 14 bytes. **Line starts** are the stream start and the byte after
   any of: `\n`, `\r`, `\v`, `\f`, `\x1c`, `\x1d`, `\x1e`, U+0085 (`C2 85`), U+2028
   (`E2 80 A8`), U+2029 (`E2 80 A9`). The JS `LineTerminator` set (`\n`, `\r`, U+2028, U+2029)
   is the required minimum (it is what the in-repo host honours, V17); the remainder is
   Python's `str.splitlines` set, included because it costs one table entry each and a
   Python host is the likeliest third-party parser. Terminator bytes themselves pass through
   unmodified the moment they are read; only the partial-sequence state (`E2`, `E2 80`, `C2`)
   is carried across writes, so a terminator split across chunks still sets `atLineStart`.
   While `atLineStart`, bytes are matched against the two reserved tokens; the moment a token
   completes, `worker: ` is emitted followed by the held bytes and copying continues raw until
   the next terminator resets `atLineStart`. If the held bytes can no longer extend to a
   reserved token, they are flushed unmodified. On EOF with held bytes (a final unterminated
   line that is a strict *prefix* of a reserved token, e.g. the stream ends at `"pol"`), flush
   them unmodified — a complete token is always escaped the instant it completes, so nothing
   forgeable survives. Chunk-boundary splits (`"pol"` at the end of one 32 KiB chunk,
   `"icy-result: …"` opening the next) are handled by the carried hold-back state; **no line
   buffering and no unbounded memory** — the copier keeps its 32 KiB chunk shape and exact cap
   accounting (counted on bytes read from the worker, `written.Add(int64(n))` as today). The
   guard also records `endsWithNewline` (last byte written was `\n`; true initially) for §3.
2. **Verdict switch extension** (`run_policy_supervise.go:162–189`): unchanged order for
   `overCap` and `DeadlineExceeded` (both return 3, supervisor-emitted envelope — unforgeable
   since the guard is not in that path). In the `waitErr` block, an `ExitError` with code 3 →
   emit the `worker_reserved_exit` envelope and return 1. All other exit codes pass through as
   today. Because the deadline case precedes the `waitErr` block, a worker backstop exit 3 that
   races a genuine timeout still reports rc 3 / `reason:"timeout"` from the supervisor (V19).
3. **Fresh-line rule for supervisor output.** After `drainOutput` returns (both copiers have
   finished, so the guard's state is final), every supervisor write to stderr goes through one
   helper, `supervisorLine(format, args…)`, which writes `\n` first when
   `!guard.endsWithNewline`, then the line, and sets `endsWithNewline`. The test is `\n`
   specifically, not "any terminator": a trailing `\r` or U+2028 is a line start for a JS
   regex but not for a `\n`-splitting host, so the supervisor starts its line after a real
   `\n`. This covers `policy:` (`:159`) and all four `policy-result:` writes.
4. **Control pipe: unchanged.** One message, read once (`:133–144`); the worker closes its end
   in `reportDecision` at admission (`run_policy.go:288–294`). The backstop `AfterFunc`
   (`run_policy.go:195–197`) is unchanged: its stderr line is now prefixed `worker: ` by the
   guard, and its exit 3 can never be the reported verdict in a supervised run (§2).
5. **Host hardening** (`.pi/extensions/ailang-exec.ts:213–235`): `parsePolicyLine` and
   `parseResultLine` split `stderr` on `"\n"`, match each line with the non-multiline regex
   `^policy: (\{.*\})\s*$` / `^policy-result: (\{.*\})\s*$`, and return the **last** match.
   `composeEnvelope`'s `cleanErr` removes exactly the lines it parsed (by index) instead of the
   first `m`-flag match. Run `make pi-assets` to resync the embedded copies.

**Not changed:** `emitJSON` denial-to-stdout (exit 2), the refusal path (exit 1), the widening
flag refusals, the output-cap kill mechanics, `proctree`, the worker environment allowlist.

### Implementation Plan

**Phase 1 — the supervisor guard, fresh-line rule and rc reservation (M1, ~6h)**
- [ ] `stderrGuard` type with the state machine above; unit tests for chunk splits, EOF
      prefixes, `\n`-only lines, each line terminator (`\r`, U+2028, U+2029, and a sample of
      the `splitlines` extras) including a multi-byte terminator split across writes, and a
      >32 KiB line containing a mid-stream `policy:` token (mid-line tokens must NOT be escaped
      — only line starts).
- [ ] Wire the guard into the stderr copier only; keep cap accounting on bytes read.
- [ ] `supervisorLine` helper; route the `policy:` line and all four `policy-result:` writes
      through it (fresh-line rule).
- [ ] Remap worker exit 3 → 1 + `worker_reserved_exit` envelope (message names the original
      code and the reservation); leave every other exit path untouched.
- [ ] End-to-end tests (Testing Strategy): the charter forge-vs-real-timeout test, and its
      `\r`, U+2028/U+2029 and unterminated-line variants.

**Phase 2 — host hardening, docs, changelog (M2, ~4h)**
- [ ] `.pi/extensions/ailang-exec.ts`: split-on-`\n`, last-match parsers; `cleanErr` removes
      the parsed lines. `make pi-assets` to resync the embedded copies.
- [ ] Host tests in `.pi/extensions/.ailang-exec.test.ts`: `worker: `-prefixed lines → `null`;
      `"junk\rpolicy-result: {…}"` and the U+2028/U+2029 forms → `null`; two `policy:` lines →
      the last one; a glued-then-fresh supervisor line parses.
- [ ] `docs/docs/guides/agent-tool-policy.md`: document the reserved line tokens, the
      `worker: ` prefix and the terminators that trigger it, the fresh-line guarantee for
      supervisor lines, the rc-3 reservation and the `worker_reserved_exit` remap, and that
      hosts should split on `\n` and take the last contract line.
- [ ] Changelog entry under `[Unreleased]` in `changelogs/v0.32-current.md`.

### Files to Modify/Create

**Modified files:**
- `cmd/ailang/run_policy_supervise.go` (+~80/−10 LOC) — the guard, the `supervisorLine`
  helper, the rc-3 remap.
- `cmd/ailang/run_policy.go` — **no change** (fd 3 lifetime and the backstop stay as today).
- `cmd/ailang/run_policy_hardening_test.go` (+~180 LOC) — the new tests enumerated in the
  Testing Strategy.
- `docs/docs/guides/agent-tool-policy.md` (+~20 LOC) — the contract paragraph.
- `.pi/extensions/ailang-exec.ts` (+~20/−8 LOC) — split-on-`\n`, last-match parsers.
- `.pi/extensions/.ailang-exec.test.ts` (+~30 LOC) — the host-side tests (then `make pi-assets`).
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

### Example 4: the line-terminator and unterminated-line forges (revision)

A program that eprints `"junk\rpolicy-result: {…\"reason\":\"timeout\"…}"` and exits 3: the
guard treats the byte after `\r` as a line start and emits `junk\rworker: policy-result: {…}`;
rc is remapped to 1. Same for U+2028/U+2029 in place of `\r`.

A program that writes `"partial"` to stderr with no trailing newline and then exits 0 on an
admitted run:
```
partial
policy: {"ok":true,…}
```
The supervisor saw `endsWithNewline == false` and wrote `\n` first, so the admission line is
its own line and the host reports `admitted: true` — not a not-admitted envelope with stdout
parsed as the decision.

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
- [ ] The `\r`, U+2028 and U+2029 variants of the forge come back rc 1 with the forged line
      prefixed after the terminator, and no `m`-flag host regex matches any worker text.
- [ ] A worker whose last stderr line is unterminated cannot glue the supervisor's `policy:` or
      `policy-result:` line: both appear on their own `\n`-delimited line and parse.
- [ ] fd 3 is still closed by the worker at admission (unchanged; `run_policy.go` has no diff).
- [ ] Host: `parseResultLine`/`parsePolicyLine` return null for `worker: `-prefixed lines and for
      `\r`/U+2028/U+2029-introduced lines, and return the last of several matching lines.
- [ ] All tests passing (`make test`); `make check-boundaries` green (no new imports).
- [ ] Documentation updated (agent-tool-policy guide, changelog Unreleased entry).

## Testing Strategy

**Unit tests (`cmd/ailang/run_policy_supervise_test.go` or the hardening file):**
- `stderrGuard` table test: line-start token escaped; mid-line token not escaped; token split
  across two writes; `"\n"`-only chunks; EOF with a partial token prefix; EOF with a complete
  token and no trailing newline (must be escaped); empty input; 32 KiB+ single line; token
  after `\r`, after U+2028, after U+2029, after `\f`/U+0085; U+2028 split `E2 | 80 A8 | policy…`
  across three writes; `endsWithNewline` after `\n` (true), after `\r` (false), after no
  output (true).

**Integration tests (`cmd/ailang/run_policy_hardening_test.go`, following the existing
`buildAilang`/`runAilangBin` helpers):**
- `TestRunPolicy_ForgedResultLineEscapedAndExitRemapped` — the charter case (Example 1's
  before/after), including the side-by-side contrast with a real timeout run.
- `TestRunPolicy_ForgedAdmissionLineOnStderrIsEscaped` — the `policy:` variant.
- `TestRunPolicy_WorkerExit3WithoutLimitRemapped` — `exit(3)` with no forged line: rc 1 +
  `worker_reserved_exit`.
- `TestRunPolicy_EscapeAcrossChunkBoundary` — the forge line emitted as >32 KiB of padding
  followed by `\npolicy-result: …`, and a second variant splitting the token itself.
- `TestRunPolicy_ForgedResultLineAfterCR` / `…AfterLineSeparator` / `…AfterParagraphSeparator`
  — the charter forge with the line introduced by `junk\r`, `junk\u2028`, `junk\u2029`; assert
  rc 1, the forged token preceded by `worker: `, and (splitting supervisor stderr on
  `[\n\r\u2028\u2029]`, the JS-equivalent line split) that no line starting with
  `policy-result:` carries the forged reason. Same table for the `policy: {"ok":true}` forge on
  a denied-by-policy run.
- `TestRunPolicy_UnterminatedStderrDoesNotSwallowSupervisorLine` — an admitted program whose
  last stderr write has no newline; assert stderr contains `\npolicy: {` and that the line
  parses; a second case does the same with a genuine `timeout_ms` kill and `policy-result:`.
- These sit beside the charter test (`TestRunPolicy_ForgedResultLineEscapedAndExitRemapped`)
  and share its side-by-side contrast with a real timeout run.
- Mutation checks: removing the guard makes the charter test fail; making the guard `\n`-only
  makes the `\r`/U+2028 tests fail; removing the fresh-line rule makes the unterminated test
  fail.

**Host-side (`.pi/extensions/.ailang-exec.test.ts`):**
- Parse-null for `worker: `-prefixed lines and for `\r`/U+2028/U+2029-introduced lines;
  last-match when two contract lines are present; `composeEnvelope` on the unterminated-then-
  fresh shape reports `admitted: true`.

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
| V6 | The control pipe is read exactly once and the worker closes its end after the admitted report (kept as-is by this design) | Read `run_policy_supervise.go:133–144` (single `sc.Scan()`), `run_policy.go:288–294` (`reportDecision` closes), `:245` (admitted report site) |
| V7 | The host extension parses and banks the line from stderr, first-match-wins | Read `.pi/extensions/ailang-exec.ts:221–273` (`parseResultLine`, `parsePolicyLine`, `composeEnvelope`); the three embedded copies are identical (`diff -q` × 2) and synced by `make pi-assets` (`Makefile:367`) |
| V8 | No filtering of worker stderr exists anywhere (negative existence) | `grep -rn "policy-result" cmd/ internal/` → only `run_policy.go`, `run_policy_supervise.go`, the hardening test, and the TS assets; no other writer or filter touches the relay |
| V9 | The AILANG program cannot write to raw fds (negative existence) | `grep -n "RegisterOp(\"IO\"" internal/effects/io.go` → print/println/readLine/writeBytes/exit/flush/printErr/eprintln only — every sink is fixed stdout/stderr; `grep -rn "os.NewFile\|Fd()" internal/effects/` → no hits (non-test) |
| V10 | Grandchildren do not inherit fd 3 (negative existence) | `grep -rn "ExtraFiles" internal/ cmd/` → the only non-test hit is `run_policy_supervise.go:71`; `internal/effects/process.go:161` spawns with `exec.CommandContext` and no `ExtraFiles`, so fd 3 is closed in every `Process` child |
| V11 | `eprintln`/`exit` are IO-admitted builtins | `internal/effects/io.go:16,21`; `std/io.ail:40` (`export func exit(code: int) -> () ! {IO}`); the live forge (V1) exercised both under `allowed_caps = ["IO"]`, and `ailang check` passed on the exact program |
| V12 | The stdout spoof is already covered; the existing corpus pins the regression surface | Read `run_policy_hardening_test.go`: `TestRunPolicy_StdoutCannotSpoofDecision`, `TestRunPolicy_TimeoutMsEnforced`, `TestRunPolicy_OutputCapStopsProduction`, `TestRunPolicy_TimeoutKillsDescendants`, `TestRunPolicy_EntryComesFromPolicy` — all exist and assert what the doc claims |
| V13 | The documented host contract is exit 3 + envelope on stderr | Read `docs/docs/guides/agent-tool-policy.md:19` and `:238–239` |
| V14 | rc 3 is not otherwise used by `run` | `grep -rn "Exit(3)\|exit 3" cmd/ailang/*.go` (non-test) → only the policy-limit sites; `ailang fmt`'s parse-error exit 3 (`fmt.go:38,119`) is a different subcommand with its own documented meaning |
| V15 | Version/target retarget | `std/VERSION` → v0.52.5; `changelogs/v0.32-current.md` has an `[Unreleased]` head; the triage doc carried no target, so this doc targets v0.53.0 per the maintainer directive |
| V17 | JS `m`-flag regexes treat `\r`, U+2028, U+2029 as line starts | `node -e` with the exact `parseResultLine` regex `/^policy-result: (\{.*\})\s*$/m`: `"junk\rpolicy-result: {…}"`, `"junk\u2028policy-result: {…}"`, `"junk\u2029policy-result: {…}"` all match (review, re-run 2026-10-08) |
| V18 | An unterminated last worker line hides the supervisor's `policy:` line | `/^policy: (\{.*\})\s*$/m` on `"partial-linepolicy: {\"ok\":true}\n"` → no match (node); `composeEnvelope` (`ailang-exec.ts:268–275`) then parses stdout as the decision; the supervisor writes `policy:` with no leading newline (`run_policy_supervise.go:159`) |
| V19 | The worker backstop never decides a supervised verdict | `run_policy_supervise.go:61` arms `context.WithTimeout(…, res.Timeout)` before `cmd.Start`; the worker's `AfterFunc` fires `res.Timeout+time.Second` after the worker starts (`run_policy.go:195`); the `DeadlineExceeded` case (`run_policy_supervise.go:169`) precedes the `waitErr` block, so even a backstop exit 3 reports the supervisor's own timeout envelope |
| V20 | Holding fd 3 open past admission would be reachable by a child | On Linux, a same-uid process can `open("/proc/<pid>/fd/3", O_WRONLY)` on another process's pipe write end; a `trusted_host` `Process` child (`internal/effects/process.go:161`) runs as the worker's uid. Today the worker closes fd 3 at admission (`run_policy.go:288–294`), before any program code runs, so the path does not exist; this design keeps it that way |
| V16 | No existing design doc covers this (duplicate gate) | `ailang docs search` (SimHash, implemented + planned): no match above 0.45 on the topic; the nearest real coverage is `implemented/v0_41_0/m-executor-policy-hardening.md` (built the supervisor, treats the envelope as trusted output) and the triage doc itself, which routes here |

## Conflict Surface

This design touches no parser/typechecker/codegen package — the surface it extends is the
**host-facing output contract** of a supervised run. Enumerated in the same spirit:

**Positions touched:**
1. The worker stderr relay (`run_policy_supervise.go:105–126`) — bytes now pass a guard.
2. The worker exit-code passthrough (`:178`) — code 3 is intercepted.
3. The documented stderr lines (`policy:`, `policy-result:`) — worker-origin lines with those
   tokens after any line terminator are now prefixed; supervisor lines always begin after `\n`.
4. The in-repo host parsers (`ailang-exec.ts:213–235`) — split on `\n`, last match.

**What else lives in those positions:**

| Position | Existing valid form | Post-change behavior |
|----------|--------------------|----------------------|
| Worker stderr line starting with `policy:` / `policy-result:` | A legitimate program printing such a line (a linter, a policy report) — passes verbatim today | Byte-preserved behind `worker: ` — visible, unambiguous, nothing dropped |
| Worker stdout line starting with those tokens | The program's output, contractually (`run_policy.go:35–37`: "stdout stays the program's") | **Untouched** — the stdout spoof is pinned unbelieved (`TestRunPolicy_StdoutCannotSpoofDecision`) |
| Worker exit code 3 | Passed through; indistinguishable from a limit | Remapped to 1 + honest envelope; every other code (0,1,2,4…) passes through as today |
| fd 3 after admission | Closed by the worker's `reportDecision` | **Unchanged** — still closed at admission (revision item 4) |
| Worker stderr ending without `\n` | Supervisor's next line is glued onto it | An extra `\n` precedes the supervisor's line; the worker's bytes are unchanged |
| Worker stderr using `\r` (progress bars) | Passes verbatim | Passes verbatim, except a reserved token right after the `\r` gains `worker: ` |
| `ailang fmt` exit 3 | A different subcommand's documented parse-error code (`fmt.go:38`) | Unaffected — the reservation is scoped to `run --policy`'s supervisor |

**Disambiguation strategy:** the guard matches the *complete* token (`policy-result:` /
`policy:`) at a line start only; `policies:`, `my-policy:`, or a mid-line token never match.
"Line start" is a superset of what any host regex anchors on (`m`-flag JS: `\n`, `\r`, U+2028,
U+2029; `\n`-splitting hosts: `\n`), so worker text can no longer satisfy them.

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
gain a `worker: ` prefix; (3) a supervisor line may be preceded by an extra `\n` when the
worker's stderr ended mid-line. Anything else that breaks is a regression, not an intentional
change.

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
  neighbour; a direct fix, tracked separately (sibling PR #1664), not bundled here.
  **Compatibility constraint:** #1664 adds a one-line supervisor warning on stderr. That line
  must not start with `policy` (no `policy-warning:` or similar), so the reserved prefix stays
  exclusively the two contract lines; and if it is written after the relay starts, it must go
  through the `supervisorLine` fresh-line helper. Whichever PR lands second adapts.
- An out-of-band result channel (`--policy-result-fd` / `--policy-result-file`) — only if a
  host asks for a machine channel separate from human stderr (triage's recommendation; see
  Future Work).
- Per-run nonce authentication (triage option 3) — dominated by the in-place fix.
- Escaping worker **stdout** — contractually the program's, and the spoof is already pinned
  unbelieved.
- Changing the denial (rc 2) or refusal (rc 1) contracts.
- Guarding the stdout relay against an unterminated-line glue: on a denial the worker runs no
  program code, so stdout carries only the decision JSON.
- Guarantees about write interleaving order between the supervisor's own stderr lines and relayed
  worker lines within the same stream position (the supervisor emits its verdicts after
  `drainOutput`, so the *lines* are ordered; byte-level interleaving of concurrent writes was
  never guaranteed and is not made so).
- Auditing whether a *crashing* worker's exit status (Go panic, signal deaths) can collide with
  the reserved codes — noted for a future audit (a signal death returns −1, not 3, from
  `ExitCode()`; a Go panic conventionally exits 2, which shares denial's code) — see Future Work.

## Timeline

**Day 1** (~6h): M1 — the guard (all terminators), the fresh-line rule, the rc reservation,
unit + charter and variant integration tests.
**Day 2** (~4h): M2 — host hardening and tests, docs, changelog.

**Total: ~2 days** (estimate doubled from the naive ~1 day per house rules).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| The guard introduces a copy bug (dropped/duplicated bytes) on chunk boundaries | High — corrupts all worker stderr | State-machine unit table incl. split-token and >32 KiB lines; the existing stderr-asserting corpus (`TimeoutMsEnforced`, `OutputCapStopsProduction`, `TimeoutKillsDescendants`) must pass unchanged; hold-back is ≤14 bytes, structurally incapable of eating a line |
| A multi-byte terminator split across chunks is missed | High — reopens the U+2028 forge | Partial-sequence state carried across writes; unit test splits `E2 80 A8` across three writes |
| The fresh-line `\n` changes byte-exact stderr for programs that end mid-line | Low | Only an added `\n` before a supervisor line; worker bytes untouched; documented in the guide |
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
| A6: Safe Concurrency | 0 | No new concurrency; the guard runs in the existing stderr copier goroutine and the supervisor reads its state after the copiers finish |
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
**Last updated**: 2026-10-08 (revision after review: terminator-aware guard, fresh-line rule, host last-match, fd-3 self-limit phase dropped)
