## M-TERMINAL-IO: Real-time terminal output — C-style string escapes + `flush()`

**Status**: IMPLEMENTED (on `dev`, pending release) — escapes + `flush()` + line-buffered TTY + `examples/progress_bar.ail`
**Target**: v0.27.0 (tentative)
**Priority**: P1 (High — blocks an entire class of programs: games, progress bars, streaming dashboards, TUIs)
**Estimated**: 1–1.5 days
**Dependencies**: None. Two additive runtime changes (lexer + IO effect/builtin) plus stdlib surface in `std/io`.

**Reported from**: Snake Showdown build team (external 0→1 user), AILANG v0.26.1 — see `ailang-feedback-for-mark.md`, Friction #2 and #5.
**Verified against**: v0.26.2 (current `dev`).

---

## Verdict: VALID — both are confirmed runtime bugs, tightly coupled by the same use case

A 0→1 user built a self-playing Snake game that renders live in the terminal at ~10 FPS.
They hit two independent runtime limitations that **together** made real-time terminal
rendering impossible without obscure workarounds (`std/bytes.fromInts([27])` for ESC, and
`writeFileBytes("/dev/tty", ...)` under a `FS` capability for flushing). Both are real,
both are small, and both are in the runtime lane (not docs, not motoko). They are bundled
here because they share one root cause class — "you cannot drive a terminal in real time" —
and an acceptance test that exercises both at once.

Neither change is a silent fallback; both make the runtime do the obviously-correct thing
that every other language already does.

---

## Problem A — Lexer does not process `\xHH` / `\uHHHH` / `\0` / `\e` escapes

`"\x1b[92m"` outputs the literal text `x1b[92m`, not an ESC byte (0x1B) followed by `[92m`.
The lexer's escape switch handles only `\n \t \r \\ \" \$`; everything else falls through a
`default:` case that writes the character after the backslash **literally**.

Evidence — [internal/lexer/lexer.go:340-361](../../internal/lexer/lexer.go#L340-L361):

```go
if l.ch == '\\' {
    l.readChar()
    switch l.ch {
    case 'n': segment.WriteRune('\n')
    case 't': segment.WriteRune('\t')
    case 'r': segment.WriteRune('\r')
    case '\\': segment.WriteRune('\\')
    case '"': segment.WriteRune('"')
    case '$': segment.WriteRune('$')
    default:
        segment.WriteRune(l.ch)   // <-- \x1b becomes "x", \033 becomes "0", etc.
    }
    l.readChar()
    continue
}
```

Char literals have the same gap — [internal/lexer/lexer.go:474-486](../../internal/lexer/lexer.go#L474-L486).
No tests cover hex/unicode/octal escapes ([internal/lexer/lexer_test.go:196-224](../../internal/lexer/lexer_test.go#L196-L224)
only tests `\n \t \"`).

This is doubly bad against the **No Silent Fallbacks** principle: an unrecognized escape
is silently demoted to "drop the backslash, keep the next char" — exactly the kind of quiet
data corruption the project forbids.

### Design

AILANG strings are UTF-8 Go strings, so the fix must never emit invalid UTF-8.
Interpret hex/unicode escapes as **Unicode code points**, not raw bytes:

| Escape | Meaning | Notes |
|---|---|---|
| `\xHH` | code point `U+00HH` (exactly 2 hex digits) | covers `\x1b` = ESC perfectly; latin-1 range |
| `\u{H…H}` | code point `U+H…H` (1–6 hex digits) | the explicit, unambiguous form |
| `\e` | `U+001B` (ESC) | convenience alias for the dominant terminal use case |
| `\0` | `U+0000` (NUL) | |

- **Malformed escapes are a lexer ERROR, not a silent passthrough.** `\x` with fewer than
  two hex digits, `\xZZ`, `\u{}`, `\u{110000}` (> max code point) → a precise lexer
  diagnostic with position. This is the No-Silent-Fallbacks-correct behavior and replaces
  the current `default:` corruption.
- **Decision point (back-compat):** today `"\d"` silently yields `d`. After this change,
  an *unknown* escape letter is an error. Per the project's "no backward compatibility,
  delete old tests" testing policy this is acceptable, but it is a breaking change for any
  program that leaned on the old passthrough. Recommended: error, with the message naming
  the offending escape and listing the supported set. (Alternative if we want a softer
  landing: keep passthrough for unknown *letters* but still parse `\x \u \e \0` — pick one,
  do not do both silently.)
- **Raw bytes ≥ 0x80** remain the job of `std/bytes.fromInts([...])`. Document that `\xHH`
  is a code point, not a raw byte, so nobody expects `"\xff"` to be a single 0xFF byte.

### Implementation sketch

- Factor the escape switch into a shared `readEscape()` helper used by both
  `readStringOrInterp()` and `readCharLiteral()` (kills the duplicate-and-drift between
  string and char literals).
- Add hex/unicode lookahead (consume N hex digits, validate, `WriteRune(rune(v))`).
- Emit lexer errors via the existing diagnostic path with start/end positions.

---

## Problem B — No `flush()`; stdout is 64 KB full-buffered on a TTY

On a TTY, stdout is wrapped in a 64 KB `bufio.Writer` that is flushed **only at program
exit**. `sleep()` does not flush. There is no `flush()` in `std/io`. For a ~1 KB/frame
game board this means the screen updates roughly every 63 frames instead of every frame.

Evidence — [cmd/ailang/main_run_exec.go:326](../../cmd/ailang/main_run_exec.go#L326):

```go
stdoutBuf := bufio.NewWriterSize(os.Stdout, 64*1024) // 64KB buffer
if isStdoutTTY() {
    effCtx.IOWriter = stdoutBuf      // interactive terminal: fully buffered (!)
} else {
    effCtx.IOWriter = os.Stdout      // piped: unbuffered, line-readers see events live
}
```

`print` / `println` / `writeBytes` all route through `ctx.GetIOWriter()`
([internal/effects/io.go](../../internal/effects/io.go)), so all three hit the same buffer.
The only `Flush()` calls are at normal exit and on error
([cmd/ailang/main_run_exec.go:513,520](../../cmd/ailang/main_run_exec.go#L513-L520)).

The irony: the buffer was added for throughput (M-PERF6B) and is applied **precisely when
it hurts most** — the interactive TTY case — while the piped case it could safely buffer is
left unbuffered. The current TTY workaround (`writeFileBytes("/dev/tty", ...)`) forces the
user to add the `FS` capability to every render function, which is semantically wrong:
`FS` advertises filesystem access for what is actually terminal output.

### Design — three layered fixes (do 1+2; 3 is optional polish)

1. **`flush()` builtin** — primary fix. Register `_io_flush` and export
   `flush() -> () ! {IO}` from `std/io`. It flushes the active `IOWriter` if it is a
   `*bufio.Writer` (no-op otherwise). This is the universal, explicit escape hatch and the
   thing the user expected to exist. Required for `\r`-style progress bars / partial-line
   renders that never emit a newline.

2. **Line-buffered default on a TTY** — make the common case "just work". Replace the 64 KB
   full-buffer on a TTY with line buffering (flush on `\n`). Most renders end in a newline,
   so a per-frame `println` flushes per frame with no `flush()` call needed, while batched
   no-newline output still coalesces. Keep (or even enlarge) full buffering for the **piped**
   branch, where there is no human waiting and throughput is what matters — the opposite of
   today's split.

3. **(Optional) auto-flush before `Clock.sleep`** — a game-loop nicety. The render→sleep→
   repeat loop is the canonical real-time pattern; flushing pending output before a real
   sleep means even a no-newline frame appears before the wait. Cheap, and matches user
   intuition. Gate behind "is the IOWriter buffered & TTY" so it is free for piped runs.

Also add **`printErr` / `eprintln`** (`std/io`, unbuffered stderr) — the conventional
real-time channel and what several other runtimes use for progress output. Low cost, fills
a gap the user explicitly called out.

### Implementation sketch

- `internal/builtins/io.go` + `internal/effects/io.go`: add `_io_flush`; type-erase to the
  writer and `Flush()` if it implements `interface{ Flush() error }`.
- `cmd/ailang/main_run_exec.go`: swap the TTY branch to a line-flushing writer (e.g. a thin
  wrapper that flushes on `\n`, or `bufio` + flush-on-newline in `_io_println`). Swap the
  piped branch to the 64 KB buffer (flush at exit — already wired).
- `std/io.ail`: export `flush`, `printErr`/`eprintln`.

---

## Acceptance Criteria

- [ ] `println("\x1b[92mgreen\x1b[0m")` prints green text (ESC bytes emitted), no `std/bytes` import.
- [ ] `"\e[2J"`, `"\u{1F40D}"` (🐍), `"\x07"` all lex to the correct code points.
- [ ] Malformed escapes (`\x`, `\xZ`, `\u{}`) produce a positioned lexer error, not silent corruption.
- [ ] A real-time example (`examples/progress_bar.ail` or a self-playing snake) renders
      incrementally under `ailang run --caps IO,Clock` — **no `FS`, no `/dev/tty`**.
- [ ] `flush()` is exported from `std/io` and documented; `ailang docs std/io` shows it.
- [ ] Piped output (`ailang run ... | cat`) still streams line-by-line (no regression for
      log/line-reader consumers — the original M-PERF6B intent).
- [ ] New lexer tests for hex/unicode/octal/`\e`/`\0` and malformed-escape errors.

## Out of Scope

- Raw-mode / non-blocking keyboard input (that is the stdin story — see #231 and
  M-ONBOARDING-ERGONOMICS §stdin).
- Full terminal/ANSI stdlib (`std/term`). A `flush()` + escapes are enough to unblock; a
  curses-style layer is a separate, larger design.

## Files Touched (estimate)

- `internal/lexer/lexer.go`, `internal/lexer/lexer_test.go`
- `internal/builtins/io.go`, `internal/effects/io.go`
- `cmd/ailang/main_run_exec.go`
- `std/io.ail`
- `examples/progress_bar.ail` (new), `CHANGELOG.md`, prompt/onboarding surfaces
