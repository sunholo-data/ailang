# M-CLI-HELP-FLAG-UNIFORMITY — one contract for `--help` and `-h`: exit 0, usage on stdout, at every dispatch level

**Status**: Planned
**Target**: v0.41.3
**Priority**: P2 (reporter: "not user-blocking"; it is a machine-first contract defect — see A7 below)
**Estimated**: ~2 days (~16 hours; see Timeline)
**Dependencies**: None hard. Builds on the M-V1-SIMPLIFY-S5 dispatch table (`cmd/ailang/commands.go`); consumes the F5 follow-up ("21 commands answer `--help` on **stderr**… wants its own pass" — [FOLLOWUPS-m-v1-simplification-s5.md](../FOLLOWUPS-m-v1-simplification-s5.md)) as its backlog entry.

**Filed from**: dev-plane report by Mark (attended), 2026-09-23, originally mistitled `--help…` and
misrouted to inbox `mark-attended-steering` by the companion defect (dash-prefixed flag VALUE,
[M-coordinator-execution-trust](../m-coordinator-execution-trust.md) V29). Resent with a word-leading
title and routed to design-doc-creator.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language-semantics change; CLI help output is already deterministic, this makes the *exit code and stream* deterministic too |
| A2: Replayability | 0 | No trace/runtime impact |
| A3: Effect Legibility | 0 | No effect-surface change |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | +1 | The help contract becomes machine-checkable: extended `help_exit0_rate` gate + hidden-route enumeration make every route's help locally verifiable |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Today a machine doing `cmd --help | head` gets **nothing** for 38/83 routes (usage on stderr) and a non-zero exit for `fmt -h`; the contract makes `--help` the reliable machine introspection path |
| A8: Minimal Syntax | +1 | Deletes two hand-maintained denylists (`helpFallbackCommands`, `pkgHelpFallback`) in favour of one dispatcher-level mechanism — net surface reduction |
| A9: Cost Visibility | 0 | No resource-cost change |
| A10: Composability | +1 | The `Command.Help` hook composes with the existing table: one definition per route, inherited by every spelling (bare, group-hop, alias) |
| A11: Structured Failure | +1 | `--help` is the recovery path from every usage mistake; today that path itself fails inconsistently (exit 0/1/2 across commands). Uniform exit 0 makes it a dependable recovery channel |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

### The report (measured on v0.34.0-dev by the reporter)

`--help` behaved three different ways across subcommands, same binary, same invocation shape:

```
ailang messages --help  -> exit 0, "Usage: ailang messages <subcommand> [options]"
ailang check    --help  -> exit 0, "Usage of check:"
ailang run      --help  -> exit 0, "Usage of run:"
ailang prompt   --help  -> exit 0, "Usage: ailang prompt [OPTIONS]"
ailang chains   --help  -> exit 1, "Unknown subcommand: --help"
ailang eval-elo --help  -> exit 1, "Error: unknown flag \"--help\""
```

The reporter asked for a systemic audit of every subcommand rather than a chains patch, and warned
that a scripted survey had already lied to him once (it reported no subcommand supported `--help`,
contradicting a direct measurement).

### Current State at HEAD (v0.41.1, commit `2eadd646`, re-audited 2026-09-23)

Two of the reporter's hard failures were fixed by M-V1-SIMPLIFY-S5 (v0.40.1): `chains --help` is
real help now ([changelog](../../../changelogs/v0.32-current.md), "M-V1-SIMPLIFY-S5 M3"), and
`eval-elo --help` is intercepted by the dispatcher's `helpFallbackCommands` denylist. But the bug
class — one flag, several behaviours — persists, in three measured forms:

**Full audit: 93 route names probed × 2 flags (209 probes incl. positive controls).** 83 are real
routes (the other 10 names — `quality`, `provenance`, `stats`, `versions`, `info`, `key`, `history`,
`notify-upgrade`, `affected-by`, `cascade` — are **not** top-level routes; `ailang quality --help`
correctly answers `Error: unknown command 'quality'`, exit 1).

| Class | `--help` | `-h` | Example |
|---|---|---|---|
| exit 0, usage on **stdout** (custom or table-rendered help) | 62 routes | 44 routes | `messages`, `chains`, `coordinator`, `daemon`, `pkg`, `eval-elo` (denylist block) |
| exit 0, usage on **stderr only** (Go flagset `ExitOnError` default, `"Usage of X:"`) | 21 routes | 38 routes | `check`, `run`, `verify`, `iface`, `lsp`, `exec`, `eval-suite`, … |
| **exit 2**, help text on stdout | — | **`fmt`** | `ailang fmt -h` prints full custom help, exits **2** |
| exit 0 on `--help` (stdout) but stderr on `-h` | — | 17 routes | `docs`, `prompt`, `test`, `init`, `budget`, `serve-api`, bare pkg verbs `add`/`bin`/`install`/`lock`/`publish`/`search`/`tree`/`unpublish`, `agent-prompt`, `axioms`, `devtools-prompt` |

The exact stderr-only set on `--help` (21 routes, cross-validating F5's 2026-09-18 measurement of
the same class): `ai-check, check, debug, design-quorum, design-review, eval-analyze,
eval-censored-pairs, eval-paired, eval-publish, eval-suite, exec, export-training,
generate-extension-registry, iface, lsp, policy-check, policy-tool, replay, run, select-best,
verify`.

Concrete residual defects, all at HEAD:

1. **`ailang fmt -h` exits 2.** `runFmtCommand` uses `flag.ContinueOnError`; `-h` is an undefined
   flag, so `fs.Parse` returns `ErrHelp` after invoking `fs.Usage` (which prints the custom help to
   stdout), and the error branch calls `os.Exit(2)` (`cmd/ailang/fmt.go`). `--help` is a defined
   bool flag there and exits 0. This is the only real route where a help flag exits non-zero.
2. **38/83 routes (46%) answer `-h` on stderr**, 21/83 answer `--help` on stderr — a consumer
   piping stdout (`ailang check --help | head`) gets nothing. Notably `ailang help check` already
   prints the same flagset usage on **stdout**, so the CLI already disagrees with itself about
   which stream help belongs on.
3. **17 routes disagree with themselves between the two spellings**: `--help` is hand-parsed to a
   custom stdout block, `-h` is not a defined flagset flag and falls through to the stderr default.
4. **Denylist architecture**: `helpFallbackCommands` (17 routes, `commands.go`) and `pkgHelpFallback`
   (2 routes, `commands_pkg.go`) are hand-maintained maps "measured against `bin/ailang-before`, not
   assumed" — a new command with a custom parser silently regresses unless someone remembers to add
   it. The groups entry points keep a third copy of the same idea (`commands_groups.go:255`).
5. **The gate is blind to all of the above.** `help_exit0_rate` (`tools/simlicity_metrics.sh`) probes
   `--help` only (never `-h`), checks only the exit code (a route answering on stderr passes), and
   enumerates routes **from the help output** — hidden rows (bare pkg verbs, `internal-dump-iface`,
   folded `trace`/`observatory`/`dashboard` rows) are invisible to it. `fmt -h` cannot fail this gate.
6. **Nested verb level is mixed too** (sampled): `coordinator status --help` → stdout; `chains chat
   --help` → stderr flagset (`Usage of chains chat:`); `messages send --help` → stderr; `daemon run
   --help` → stderr. Group-hop spellings are consistent (`dev builtins --help` ≡ `builtins --help`),
   which the audit confirmed.

**Impact:** agents and scripts are the primary CLI audience (A7). A machine that has learned
"`--help` works" gets, elsewhere, empty output, the wrong stream, or a non-zero exit that its error
handling treats as a command failure. The reporter's own survey failure illustrates the cost: a
stdout-only reader concluded "no subcommand supports `--help`".

## Survey methodology (and why it is in this doc)

The reporter warned: *"[my] first attempt to script the survey produced a WRONG answer… Measure
each subcommand individually and keep a known-good positive control in the output, or the survey
will lie to you."* The audit above followed exactly that, and the warning earned itself twice more
during this session:

- The survey script captured stdout and stderr **separately** (the 21-route stderr class is exactly
  what a stdout-only reader misses), ran each route individually under `timeout 15` with stdin
  closed (daemon/exec/repl can block), and printed **positive controls** (`check --help`,
  `messages --help`) at the start AND end of the output file. Both control pairs were valid
  (`check`: exit 0, stderr-only; `messages`: exit 0, stdout) — the survey is certified.
- Two of the author's own follow-up probes initially produced wrong answers and were caught by
  mechanism checks: a piped exit-code read (`cmd | head; echo $?`) reported `head`'s status, and a
  route list built from a name grep contained 10 pkg-verb names that are not top-level routes,
  which briefly looked like "10 commands reject `--help`" until each was shown to be an unknown-command
  error. Every classification in this doc was re-derived from the mechanism (source read), not from
  output shape alone.

The design makes this methodology a standing gate (see P4) so the audit does not have to be redone
by hand.

## Goals

**Primary Goal:** One contract at every dispatch level (top-level, group, `pkg` verb, nested verb):
`--help` and `-h` exit 0 and print that command's usage to stdout.

**Success Metrics:**
- 83/83 real top-level routes (+ all nested verbs audited in the sprint): `--help` and `-h` both
  exit 0, both print non-empty usage to stdout (measured by the extended gate).
- `help_exit0_rate` extended to probe both flags and require non-empty stdout; route enumeration
  includes hidden routes (via a table-generated route dump).
- `helpFallbackCommands` and `pkgHelpFallback` deleted; help-flag handling has exactly one
  mechanism (the dispatcher + per-command help hooks).
- `tools/cli_surface_snapshot.sh` regenerated once, deliberately, with the stream moves recorded.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Usage moves to **stdout** for all routes (the F5 pass) — stderr reserved for errors | Moves observable output for ~38 routes and ~38+17 snapshot records; consumers piping stderr today change behaviour | human | design | med |
| D2: `--help`/`-h` interception becomes the dispatcher's **default** (denylist inverted to a per-command help hook) | Removes the regression vector for every future command; deletes two maps | agent (mechanism) | design | med |
| D3: Per-command help **content** stays as-is (flagset defaults, custom blocks, table block) — only exit code and stream are unified | Rewriting 83 help texts would churn every doc/snapshot for no machine benefit | agent | design | low |
| D4: Unknown commands + `--help` keep exit 1 (interception applies to real routes only) | `ailang typo --help` exiting 0 would mask typos; must be stated or an implementer will "fix" it | design (this doc decides) | design | low |
| D5: Gate shape — extend `help_exit0_rate` + add hidden-route enumeration source (`internal-dump-routes`, patterned on `internal-dump-iface`) | A gate that cannot see `-h`, streams, or hidden routes certifies none of the contract | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **D1 (stdout move)** — human sign-off required: this is the deliberate, one-time snapshot
  churn F5 deferred to "its own pass". This doc IS that pass; approving the doc approves the churn.
- [ ] D2 mechanism confirmed against the S5 table doctrine (help becomes a row attribute, like
  `Summary`; one definition inherited by every spelling).

## Solution Design

### Overview

Make the dispatch table the single authority for help-flag handling, exactly as S5 M1 made it the
single authority for routing and help listing. Every `Command` row gains a `Help` hook; the
dispatcher guarantees the contract (exit 0, stdout) for every route at every level; the flagset
class is normalized by pointing flagset output at stdout; `fmt` is fixed; the gates are extended so
the contract cannot silently regress.

### Architecture

**Components:**

1. **`Command.Help` hook + universal interception (D2).**
   `dispatchCommand` (and the group/pkg dispatchers) intercept a bare `--help`/`-h` tail for
   **every** route, before `Run`:
   - If the row has a `Help func(w io.Writer)` — call it, exit 0. Commands with rich help today
     (`messages`, `chains`, `coordinator`, `daemon`, `pkg`, `mission`, `models`, `storage`, `fmt`,
     group entry points, …) register their existing help printer here; nothing about their text
     changes.
   - Otherwise the dispatcher renders the table's generic block (`printCommandHelp`) — what the
     denylist does today, now the default.
   `helpFallbackCommands`, `pkgHelpFallback` and the group-entry copy are deleted; the three places
   that implement the same idea become one. Because the interception happens before `Run`, it also
   guarantees `--help` can never start a daemon, a server, or a spend (a property only the
   denylist routes have today).

2. **Flagset stream normalization (D1, D3).**
   The flagset class (21 routes on `--help`, 38 on `-h`) keeps `flag.PrintDefaults` content verbatim
   but lands it on stdout: `fs.SetOutput(os.Stdout)` (or a package-level helper so the choice is not
   re-made 38 times). The 17 self-inconsistent routes (`docs`, `test`, `prompt`, …) additionally get
   `-h` treated like their hand-parsed `--help` (define an `-h` bool alias or handle `ErrHelp`
   explicitly) — or simply drop their hand-parsing in favour of the dispatcher hook + `Help` printer.

3. **`fmt` fix.**
   `runFmtCommand`: `if err := fs.Parse(args); err == flag.ErrHelp { printFmtHelp(); os.Exit(0) }`
   (and route flagset output to stdout with the rest of the class). Exit 2 remains for genuine
   usage errors — the doc's contract is about the help *flags*, and `fmt --help`/`-h` must not be
   conflated with `fmt`'s documented operational-error exit code.

4. **Nested verb level.**
   The sprint audits every nested verb of the multi-verb commands (`chains`, `messages`,
   `coordinator`, `mission`, `models`, `storage`, `daemon`, `budget`, `mcp`, `access-control`,
   `cache`, `micro-rag`, `pi`, `editor`, `dashboard`, `observatory`, `trace`, `pkg`) with the same
   probe script and positive controls, and normalizes the flagset-based ones with the same
   `SetOutput` helper. Custom dispatchers that already handle help (`coordinator`, `access-control`,
   `cache`, `browser-profile` — they case on `"help", "-h", "--help"` already) keep their behaviour.

5. **Gates (D5).**
   - `tools/simplicity_metrics.sh`: `help_exit0_rate` probes **both** flags, requires **non-empty
     stdout**, and enumerates routes from `ailang internal-dump-routes` (new hidden command, same
     pattern as `internal-dump-iface`: prints every table row incl. hidden/alias routes, one per
     line) instead of parsing the human help listing. The metric's existing anti-vacuity floor
     ("enumerated only N routes") carries over to the dump.
   - `tools/cli_surface_snapshot.sh`: regenerate `bin/ailang-before` comparison once; the stream
     moves show up as a single deliberate diff.
   - A standing test (extending `commands_table_test.go`) asserts: for every row in `allCommands`,
     `wantsHelp` interception applies or a `Help` hook exists — i.e. the denylist cannot be
     reintroduced without failing a test.

### Implementation Plan

**Phase 1: The contract at the top level** (~4 hours)
- [ ] Add `Help func(w io.Writer)` to `Command`; wire universal interception in `dispatchCommand`;
      migrate denylist members and the group/pkg entry points; delete `helpFallbackCommands` and
      `pkgHelpFallback`
- [ ] Fix `fmt -h` (ErrHelp → exit 0)
- [ ] `commands_table_test.go`: no-route-left-behind assertion

**Phase 2: Stream normalization** (~4 hours)
- [ ] Shared `newStdoutFlagSet(name)` helper; sweep the 21+38 flagset sites (mechanical, one line
      each)
- [ ] Give the 17 self-inconsistent routes `-h` parity with their `--help`
- [ ] Regenerate the CLI surface snapshot; record the deliberate diff

**Phase 3: Nested verbs** (~4 hours)
- [ ] Enumerate nested verbs (from the dispatchers, not by hand); probe both flags each with
      positive controls; normalize the flagset-based ones

**Phase 4: Gates and docs** (~4 hours)
- [ ] `internal-dump-routes` hidden command; `help_exit0_rate` extension (both flags, stdout
      non-empty, dump-sourced enumeration, anti-vacuity floor)
- [ ] Regenerate `docs/docs/reference/cli.md`; changelog entry; update this doc's status

### Files to Modify/Create

**New files:**
- `cmd/ailang/internal_dump_routes.go` — hidden route dump from the table, ~40 LOC
- `cmd/ailang/helpflags_test.go` (or extension of `commands_table_test.go`) — contract tests, ~80 LOC

**Modified files:**
- `cmd/ailang/commands.go` — `Command.Help` field, universal interception, delete denylist (~60 LOC)
- `cmd/ailang/commands_groups.go`, `cmd/ailang/commands_pkg.go` — group/pkg entry points use the hook; delete `pkgHelpFallback` (~30 LOC)
- `cmd/ailang/fmt.go` — ErrHelp branch (~5 LOC)
- ~30–40 command files — `fs.SetOutput(os.Stdout)` / `-h` parity (1–3 LOC each, mechanical)
- `tools/simplicity_metrics.sh` — metric extension (~30 LOC)
- `tools/cli_surface_snapshot.sh` — COMMANDS list sourced from the dump (~10 LOC)
- `docs/docs/reference/cli.md` — regenerated

## Examples

### Example 1: The machine introspection path

**Before (HEAD, measured):**
```
$ ailang check --help | head -1
$                      # nothing — usage went to stderr
$ ailang fmt -h >/dev/null; echo $?
2
```

**After:**
```
$ ailang check --help | head -1
Usage of check:        # (content unchanged; now on stdout, exit 0)
$ ailang fmt -h >/dev/null; echo $?
0
```

### Example 2: A new command cannot regress the contract

**Before:** a new command with a custom parser forgets to add itself to `helpFallbackCommands`;
`ailang newcmd --help` exits 1; `help_exit0_rate` stays green because it probes only `--help` exit
codes of routes visible in help output.

**After:** the dispatcher intercepts `--help`/`-h` before `Run` for every row — the only way to get
a non-zero exit or empty stdout is to explicitly register a `Help` hook that does so, which the
table test flags for review.

## Success Criteria

- [ ] All 83 real top-level routes: `--help` and `-h` exit 0 with non-empty stdout (probe script,
      positive controls included in its output, checked into the sprint)
- [ ] All nested verbs audited in Phase 3 pass the same probe
- [ ] `ailang fmt -h` exits 0; `fmt --help` unchanged
- [ ] `helpFallbackCommands` and `pkgHelpFallback` no longer exist in the source (grep)
- [ ] Extended `help_exit0_rate` = 100% over dump-enumerated routes, both flags, stdout checked
- [ ] CLI surface snapshot regenerated once, diff reviewed and attached to the sprint
- [ ] Unknown-command behaviour unchanged: `ailang quality --help` still exit 1 "unknown command"
- [ ] All tests passing (`make test`); docs regenerated (`make check-cli-docs`)
- [ ] Documentation updated

## Testing Strategy

**Unit tests:**
- Table test: every `allCommands` row is either intercepted by the dispatcher or has a `Help` hook
- `fmt`: `-h` and `--help` both exit 0 (the current `fmt_test.go` covers `--help` only)
- Dispatcher: unknown name + `--help` still routes to `unknownCommand` (exit 1)

**Integration tests:**
- Probe script from this audit, checked in (with its positive controls), run against the built
  binary; extended `help_exit0_rate` in `make simplicity-audit`

**Manual testing:**
- One smoke pass of the reporter's original six commands, before/after, recorded in the sprint log

## Conflict Surface (CLI positions, not parser — included because dispatch is semantics-sensitive)

**Syntactic positions touched:** `args[0]` of a command's argument tail, at three dispatch levels
(top-level, group entry, `pkg` verb). The existing `wantsHelp` already defines the match: exact
string `--help` or `-h` in position 0 **only**.

**What else lives in those positions:** subcommand names (`ailang docs search`, `ailang chains
chat`), positional file/package/`<pkg>@<ver>` arguments. No legitimate positional in this CLI is
spelled `--help` or `-h`; interception is positional, so **flag VALUES are untouched** — the
companion defect (a `--title "--help"` value misrouting `messages send`,
[M-coordinator-execution-trust](../m-coordinator-execution-trust.md) V29) is a different mechanism
and is NOT fixed or affected by this doc.

**Disambiguation:** exact match before subcommand dispatch, exactly as `wantsHelp` does today; no
new lookahead.

**Programs that MUST still work (fixtures):**
- `ailang chains chat --help` → chains' own subcommand help (content unchanged)
- `ailang dev builtins --help` ≡ `ailang builtins --help` (group hop, measured consistent today)
- `ailang pkg provenance --help` → generic block (today via `pkgHelpFallback`, after via the hook)
- `ailang messages send … --title "--help"` value positions (companion-report repro) — unaffected
- `ailang quality --help` → exit 1 unknown command (D4)

**What deliberately changes:**
- ~38 routes: usage moves stderr → stdout (exit 0 both before and after)
- 17 routes: `-h` gains `--help`'s behaviour instead of the flagset stderr default
- `fmt -h`: exit 2 → 0
- Two denylists deleted; snapshot records move once

## Deferred Decisions

The following are intentionally left open for the implementer:

- Whether a flagset route's stdout help keeps the verbatim `flag.PrintDefaults` rendering (assumed
  yes, D3) or gets a light header — agent may choose, must not lose flag documentation.
- The exact output format of `internal-dump-routes` (one route per line with group/hidden/alias
  columns is the working assumption) — agent may choose; gates are its only consumer.
- Whether the 17 self-inconsistent routes keep hand-parsed help or migrate wholesale to the hook —
  agent may choose per route, as long as the contract holds.

## Non-Goals

**Not attempted in this feature:**
- The `messages send` dash-prefixed flag-VALUE misrouting (companion report; separate doc,
  [M-coordinator-execution-trust](../m-coordinator-execution-trust.md) V29)
- Rewriting help *content* or unifying usage-line grammar beyond exit code and stream (D3)
- Bare-run exit codes (F4's `budget` question — separate decision, other milestone)
- REPL `:help` commands; `ailang help` subcommand behaviour beyond the stream it already gets right
- Making unknown commands exit 0 on `--help` (D4 explicitly keeps exit 1)

## Timeline

**Days 1** (4h): Phase 1 — dispatcher hook, denylist deletion, `fmt` fix, table test
**Days 1–2** (4h): Phase 2 — flagset stream sweep, `-h` parity, snapshot regen
**Day 2** (4h): Phase 3 — nested-verb audit and normalization
**Day 2–3** (4h): Phase 4 — gates, route dump, docs, changelog

**Total: ~16 hours across ~2–3 working days** (estimate doubled from the ~8h gut number, per skill
guidance)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fleet scripts read help from stderr today (e.g. `2>&1` merges fine; `2>` captures change) | Med | Sprint task: grep `.agents/skills/`, `.claude/`, `Makefile`, `make/`, `tools/`, `scripts/` for `--help` consumers reading stderr; the S5 M3 audit's zero/low-caller methodology applies |
| Snapshot churn masks an unintended behaviour change | Med | One deliberate regen with reviewed diff (same procedure as S5 M3's `bin/ailang-before` diff) |
| Gate extension adds false failures (a command intentionally silent on stdout) | Low | `Help` hook is the explicit escape; the anti-vacuity floor keeps the metric honest |
| Nested-verb audit scope balloons (100+ verbs) | Med | Probe script is mechanical; normalization is the same one-line `SetOutput` sweep; custom dispatchers already case on help tokens (verified: `coordinator.go:79`, `access_control.go:27`, `cache.go:58`, `browser_profile.go:96`) |

## Related Documents

**Planned (check for overlap):**
- [FOLLOWUPS-m-v1-simplification-s5.md](../FOLLOWUPS-m-v1-simplification-s5.md) — F5 ("21 commands
  answer `--help` on stderr… wants its own pass") is this doc's backlog entry; F4 (bare-run exit
  codes) is explicitly a non-goal here
- [m-v1-simplification-program.md](../m-v1-simplification-program.md) — parent program; `help_exit0_rate`
  gate named at its Phase 3
- [HANDOVER-m-v1-simplification-phase3.md](../HANDOVER-m-v1-simplification-phase3.md) — the dispatch-table
  rework this builds on ("`--help` honoured at every level" was its design intent; this doc finishes it)
- [m-coordinator-execution-trust.md](../m-coordinator-execution-trust.md) — companion report: the
  dash-prefixed flag VALUE misroute that silenced this very report's first filing (V29)

**Implemented (may inform design):**
- [m-pkg-bin-entrypoints.md](../../implemented/v0_40_1/m-pkg-bin-entrypoints.md) and the v0.40.1
  changelog entries for M-V1-SIMPLIFY-S5 M2/M3/M5/M6 — the dispatch table, groups, the `chains`
  help fix and the metrics this doc extends

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [tools/simplicity_metrics.sh](../../../tools/simplicity_metrics.sh) — `help_exit0_rate` (line ~166)
- [tools/cli_surface_snapshot.sh](../../../tools/cli_surface_snapshot.sh) — the before/after surface diff instrument
- Reporter's message: dev-plane filing, resent 2026-09-23 (originally mistitled `--help…`, misrouted — see header)

## Future Work

- `ailang help <route>` resolving hidden routes and group members (`ailang help provenance` →
  today "unknown command"; could route through the table and print the `Help` hook output)
- Teaching-prompt coverage of the now-reliable `--help` introspection path (a machine can learn
  flags per command from stdout alone)

## Verification Log

Every row measured 2026-09-23 at commit `2eadd646` (binary v0.41.1, built 2026-09-23T14:51:30Z,
`ailang version` reports `2eadd64…-dirty`). Survey artifact: 209-probe script with positive
controls (method in this doc; script at `/tmp/help_survey.sh`, output `/tmp/help_survey.txt` —
ephemeral, key rows reproduced below).

| # | Claim | Command | Observed |
|---|---|---|---|
| VL-1 | Survey validity: positive controls valid at start AND end | `grep -A2 "PROBE=control" /tmp/help_survey.txt` | 4 rows: `control-check` exit 0 STDERR_ONLY (stdout 0 B, stderr 1010 B, `Usage of check:`); `control-messages` exit 0 OK_STDOUT (`Usage: ailang messages <subcommand> [options]`); identical at start and end |
| VL-2 | Reporter's `chains --help` failure (v0.34.0-dev) is fixed at HEAD | `timeout 15 ailang chains --help`; cite changelog `changelogs/v0.32-current.md` M3 entry | exit 0, stdout 2491 B `Usage: ailang chains <subcommand> [options]`; changelog: "`ailang chains --help` is real help now" (v0.40.1, 2026-09-18) |
| VL-3 | Reporter's `eval-elo --help` failure is fixed at HEAD (denylist) | `timeout 15 ailang eval-elo --help` | exit 0, stdout 236 B: `ailang eval-elo - Per-language (AILANG vs Python) ELO leaderboard…` (the `helpFallbackCommands` generic block) |
| VL-4 | 83 real routes; class breakdown 62/21 (`--help`), 44/38/1 (`-h`) | survey script + Python tabulation over `/tmp/help_survey.txt` | 93 probed names; 10 exit 1 on `--help` are unknown commands; of 83 real: `--help` 62 OK_STDOUT + 21 STDERR_ONLY, all exit 0; `-h` 44 OK_STDOUT + 38 STDERR_ONLY + 1 BROKEN (`fmt`, exit 2) |
| VL-5 | The 10 "broken" names are NOT top-level routes (correct exit 1) — a grep-built route list lied to the author too | `timeout 15 ailang quality --help` (stderr: `Error: unknown command 'quality'`); `grep -n 'pkgLegacyTopLevel' cmd/ailang/commands_groups.go` | map has exactly 9 verbs (add, lock, tree, install, bin, search, publish, unpublish, docs→pkg-docs); `quality`/`provenance`/`stats`/… appear only in `pkgSubcommands()`, reachable solely as `ailang pkg <verb>` — all of which exit 0 (19/19 probed) |
| VL-6 | `fmt -h` exits 2 while printing help to stdout; mechanism | `ailang fmt -h`; `sed -n 55,80p cmd/ailang/fmt.go` | exit 2, full custom help on stdout (1188 B, stderr 0 B); source: `flag.NewFlagSet("fmt", flag.ContinueOnError)`, `fs.Usage = printFmtHelp`, error branch `os.Exit(2)`; `-h` undefined → `ErrHelp` → usage printed → exit 2. `--help` is a defined bool → exit 0 |
| VL-7 | The stderr class is flagset `ExitOnError` default behaviour | `ailang check --help`; `grep -n 'flag.NewFlagSet' cmd/ailang/check.go` | exit 0, `Usage of check:` + PrintDefaults on stderr (1010 B), stdout empty |
| VL-8 | 17 routes: `--help` hand-parsed to stdout, `-h` falls to flagset stderr; mechanism | survey tabulation; `sed -n 55,75p cmd/ailang/docs.go` | e.g. `docs`: `helpFlag := docsFlags.Bool("help", …)` → `--help` prints `printDocsHelp()` to stdout exit 0; `-h` undefined → ErrHelp → default usage stderr exit 0. Full list: add, agent-prompt, axioms, bin, budget, devtools-prompt, docs, init, install, lock, prompt, publish, search, serve-api, test, tree, unpublish |
| VL-9 | Dispatcher interception is a denylist, not a default | `sed -n 128,146p cmd/ailang/commands.go`; `sed -n 120,135p cmd/ailang/commands_pkg.go` | `helpFallbackCommands` (17 names) gates `wantsHelp`; `pkgHelpFallback` (2 names: provenance, history) gates the pkg route; non-members rely on their own parsers |
| VL-10 | `help_exit0_rate` probes `--help` only, exit code only, and enumerates from help output (hidden routes invisible) | `sed -n 166,195p tools/simplicity_metrics.sh` | `probe_help` runs `"$@" --help >/dev/null 2>&1` (no `-h` probe, no stream check); `help_routes_of` parses `ailang --help` / group help listings — hidden rows never enumerated; anti-vacuity floor exists (`< 20 routes` aborts) but only for the visible set |
| VL-11 | `ailang help check` prints the same usage on stdout while `check --help` uses stderr | `ailang help check >/dev/null`; `ailang check --help >/dev/null` | `help check`: stdout `Usage of check:` exit 0; `check --help`: stdout empty (stderr 1010 B) exit 0 — same content, different stream by trigger |
| VL-12 | No route-dump facility exists (negative-existence for D5) | `grep -rn 'Name: *"internal-' cmd/ailang/*.go \| grep -v _test` | exactly one: `internal-dump-iface` (iface JSON subprocess helper) — nothing enumerates CLI routes incl. hidden |
| VL-13 | Group-hop spellings are help-consistent | `ailang dev builtins --help`, `ailang ops coordinator --help`, both `-h` variants | all exit 0, identical stdout to the bare spellings; `commands_groups.go:255` shows the group entry re-dispatches with `--help` appended |
| VL-14 | Nested verb level is mixed (sampled, exit 0 everywhere) | `coordinator status --help` (stdout), `chains chat --help` (stderr flagset), `messages send --help` (stderr flagset), `daemon run --help` (stderr flagset), `eval run --help` (stderr flagset) | all exit 0; stream varies — the contract is needed at this level too |
| VL-15 | F5's "21 commands answer `--help` on stderr" cross-validates the audit | F5 list vs survey's 21-route STDERR_ONLY set | 19/21 names identical (`run check iface select-best export-training eval-analyze eval-paired eval-censored-pairs eval-suite eval-publish debug lsp replay exec design-review design-quorum verify generate-extension-registry ai-check policy-check`); drift: F5's bare `eval` now prints the group's custom help on stdout (S5 M2/M3), `policy-tool` joined the class since |
| VL-16 | Denylist machinery landed in the current tree via commit `2eadd646` | `git log --oneline -S helpFallbackCommands -- cmd/ailang/commands.go`; `git log -S "pkgHelpFallback"` | single commit `2eadd646` (2026-09-23, the handover merge); changelog attributes the underlying S5 work to v0.40.1 (2026-09-18) |
| VL-17 | Unknown commands exit 1 today and must keep doing so (D4 premise) | `ailang quality --help` | `Error: unknown command 'quality'`, exit 1 — correct; the design must not intercept unknown names |

## Adjacent defects found while verifying (not fixed here, recorded for routing)

1. **`create_planned_doc.sh` cannot complete on this machine.** Two independent bugs: (a) its
   result-greps use `grep -E "^\d+\."` — GNU grep 3.8 does not interpret `\d`, so the doc-search
   results NEVER match and the coverage gate silently reports "(none found)" on every query;
   (b) with empty results, `merge_results`' pipeline ends in a no-match `grep` that exits 1, and
   `set -euo pipefail` aborts the script before the template is written. Net effect: the skill's
   duplicate/coverage gate is inert and doc creation fails whenever no related doc matches —
   this doc had to be scaffolded by hand. Fix: `[0-9]` instead of `\d`, and `|| true` on the
   result-capture pipelines. (Also: `ailang docs search --neural` reports
   `Embeddings: 0 computed… (model: fallback-simhash)` — the "neural" search silently degrades to
   SimHash here; per the no-silent-fallbacks principle this deserves its own surfacing.)
2. **The stale-binary warning fires on every platform invocation from a clean checkout** (`⚠ Binary
   may be stale (source files modified after build)` despite `git status` clean — mtime pre-filter
   vs. worktree mtimes, the ailang#687 class). It polluted stderr in ~97 of the survey's probes and
   is the reason the survey separates streams.

---

**Document created**: 2026-09-23
**Last updated**: 2026-09-23