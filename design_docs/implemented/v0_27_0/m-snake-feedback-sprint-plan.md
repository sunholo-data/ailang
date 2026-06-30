# Sprint Plan: M-SNAKE-FEEDBACK — Terminal I/O + Onboarding fixes

**Sprint ID**: M-SNAKE-FEEDBACK
**Target version**: v0.27.0
**Created**: 2026-06-30
**Source**: External 0→1 user feedback (Snake Showdown build team), `ailang-feedback-for-mark.md`
**Design docs**: [m-terminal-io.md](../m-terminal-io.md) (P1), [m-onboarding-ergonomics.md](../m-onboarding-ergonomics.md) (P2)
**Risk**: Medium (M2 touches TTY buffering; everything else low)
**Estimated**: ~3 days / ~450 LOC + docs

All claims verified empirically against v0.26.2 (ran each repro). Sequence: P1 runtime (M1–M2) before P2 (M3–M5).

---

## Goal

Unblock the entire class of real-time terminal programs (games, progress bars, dashboards)
that v0.26.1 made impossible without `/dev/tty` hacks, and remove the documentation/example
rot that tripled a first build's time. Two genuine runtime bugs; everything else is examples,
one stdlib helper, and docs-truth.

---

## Milestones

### M1 — Lexer: C-style escape sequences (P1, ~120 LOC + 80 test)
**Bug**: `"\x1b[92m"` prints literal `x1b[92m`; unknown escapes silently pass through
([internal/lexer/lexer.go:357](../../../internal/lexer/lexer.go#L357)).

- Factor escape handling into a shared `readEscape()` used by `readStringOrInterp()` and
  `readCharLiteral()` (kills string/char drift).
- Support `\xHH` (2 hex → `U+00HH`), `\u{H…H}` (1–6 hex), `\e` (ESC, 0x1B), `\0` (NUL), as
  **Unicode code points** (always valid UTF-8). Keep existing `\n \t \r \\ \" \$`.
- **Malformed escapes → positioned lexer error** (replaces silent passthrough; aligns with
  No-Silent-Fallbacks). Decision: unknown escape *letters* also error.
- Tests: hex/unicode/octal-rejection/`\e`/`\0` + malformed-error cases.

**Acceptance**: `println("\x1b[92mok\x1b[0m")` emits ESC bytes (green); `"\u{1F40D}"` = 🐍;
`"\x"`/`"\u{}"` produce a lexer error.

### M2 — IO: `flush()` + line-buffered TTY (P1, ~100 LOC + tests + 1 example)
**Bug**: TTY stdout is 64KB full-buffered, flushed only at exit; no `flush()`
([cmd/ailang/main_run_exec.go:326](../../../cmd/ailang/main_run_exec.go#L326)).

- Add `_io_flush` builtin; export `flush() -> () ! {IO}` from `std/io`.
- Change the **TTY** writer from 64KB full-buffer to **line-buffered** (flush on `\n`).
  **Leave the piped branch unbuffered** — that is deliberate (M-PERF6B: downstream
  line-readers see events live). *(Corrects the design doc, which wrongly suggested
  buffering the piped branch.)*
- Add `printErr`/`eprintln` (unbuffered stderr) to `std/io`.
- New `examples/progress_bar.ail` (uses `\r` + `flush()`), runnable under `--caps IO,Clock`.

**Acceptance**: a real-time example renders incrementally under `--caps IO,Clock` (no FS, no
`/dev/tty`); `ailang run … | cat` still streams line-by-line (no M-PERF6B regression).

### M3 — Fix 8 broken examples + add a real verification gate (P2, ~migrate + ~60 LOC script)
**Bug**: 8 examples use `++` on strings (list-only since v0.13.0) and don't run; none are in
`examples_report.json`/`STATUS.md`, and there is **no `make verify-examples` CI gate**.

Examples: `bounded_xml_fold`, `datetime_demo`, `deriving_eq`, `effect_budget_demo`,
`first_non_repeat`, `short_circuit_and`, `short_circuit_or`, `string_replace`.

- Migrate each `++`-on-strings to `${…}` interpolation / `concat([...])` / `join`.
- Add `make verify-examples` that runs/type-checks every non-archive `examples/*.ail` and
  **fails on any broken example**; wire into `make ci`.
- Audit the existing `tools/audit-examples.sh` flag order (`ailang --caps X run file` puts
  flags before the subcommand — verify caps aren't silently dropped).

**Acceptance**: all 8 run clean; `make verify-examples` is green and fails if any example breaks.

### M4 — stdlib: `nth_or` / `head_or` (P2, ~30 LOC + tests)
- Add to `std/list`:
  `nth_or[a](xs,idx,default) -> a`, `head_or[a](xs,default) -> a` (match-based, safe).
- Test + document `nth`'s `Option` return under "list indexing".

**Acceptance**: `nth_or([1,2,3], 5, 0) == 0`; exported and in `ailang docs std/list`.

### M5 — Docs-truth fixes (P2, docs-only)
- `.ail` is canonical: purge `.ailang` from docs/tutorials; upgrade the non-`.ail` warning to
  a precise error.
- Getting-started: add "Effects & purity" (`pure func` vs `func`).
- Language reference: document module-level `let name : T = expr` (works since v0.5.0 —
  [archive doc](../../archive/v0_4_9_m-bug-module-let-scope.md)).
- Effects guide: capability-per-stdlib-module table; surface required caps in MCP
  `stdlib_module`/`effects_catalog`.
- Limitations: precise stdin status — `readLine()` exists (blocking); raw/non-blocking
  keyboard input does not (#231).

**Acceptance**: zero `.ailang` in docs; the four doc sections exist; MCP output names caps.

---

## Day-by-day

| Day | Work |
|-----|------|
| 1 | M1 lexer escapes (impl + tests, `make test`) |
| 2 | M2 flush + line-buffered TTY + progress_bar example + regression check on piped output |
| 3 AM | M3 migrate 8 examples + `make verify-examples` gate; M4 `nth_or`/`head_or` |
| 3 PM | M5 docs sweep; `make ci`; CHANGELOG |

## Success metrics
- `make test` + `make ci` green; new lexer + io + stdlib tests pass.
- All 8 examples run; `make verify-examples` gates future rot.
- `examples/progress_bar.ail` renders real-time under `--caps IO,Clock`.
- CHANGELOG updated; both design docs moved to `implemented/v0_27_0/` on completion.

## Out of scope
- Raw-mode / non-blocking keyboard input (#231) — separate, larger design.
- `std/term` curses-style layer — `flush()` + escapes are enough to unblock.
- Re-enabling `++` on strings — would revert M-CONCAT-DISAMBIG; explicitly rejected.
