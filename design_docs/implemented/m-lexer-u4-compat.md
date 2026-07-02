# M-LEXER-U4-COMPAT — restore `\uXXXX` string escapes (v0.27.0 regression fix)

**Status:** Implemented (2026-07-02)
**Type:** Bug fix — undo a shipped breaking change
**Severity:** High (silently bricked the motoko eval harness for ~35h)

## Summary

`v0.27.0` (M-TERMINAL-IO, commit `37b1de765`) reworked the lexer's string-escape handling.
The rework accepted **only** the braced Unicode form `\u{H..H}` and made the classic fixed
4-hex JS/JSON form **`\uXXXX` a hard parse error** (`PAR_ILLEGAL_TOKEN`). That turned
previously-parsing source into a compile failure — a breaking language change.

## Impact (how it surfaced)

- The motoko harness core `src/core/tool_runtime.ail:1037` contains `"… — …"` (an em-dash).
  Under the new lexer that file **no longer parsed** → `ailang run src/core/supervisor.ail`
  failed at module-load → **every `motoko-local` agent run produced 0 events** (it died before
  emitting `session_start`).
- This ran for **~35 hours** (last working motoko session was 35h before discovery). It
  masqueraded as "flaky infra": a 9h-wedged rotation chunk (the filler stuck retrying dead
  runs), `api_error`s, and stalled A/B experiments were **all downstream of this one break**.
- The leaderboard's `motoko 93.4%` was **stale** — real runs were 0-event failures, but
  `--skip-existing` froze the pre-regression passing results.
- **It escaped CI** because the affected code (the `motoko_agent` fork) is not in ailang's test
  suite, and no ailang example/stdlib used `\uXXXX`.

## Root cause

`internal/lexer/lexer_strings.go`, `case 'u'` unconditionally called `readBracedUnicodeEscape()`,
which errors unless the next char is `{`.

## Fix

Accept **both** forms, matching JS/JSON/Rust:
- `\u{H..H}` — 1–6 hex (unchanged).
- `\uXXXX` — exactly 4 hex (restored) via the existing `readHexEscape(4)`.

`readHexEscape`'s error message was parameterised by the actual escape char (it hardcoded `\x`)
so a malformed `\u` reports `\u requires exactly 4 hex digits`, not `\x`.

## Tests

- `TestHexAndUnicodeEscapes`: added `\uXXXX` cases, incl. the exact regression `"—"` → `—`,
  `A` → `A`, in-context, and both forms in one string.
- `TestMalformedEscapesAreIllegal`: `\u41` (too-short 4-hex) now expects `4 hex digits`.
- Validated end-to-end: `cd mk-ast && ailang check src/core/supervisor.ail` → `✓ No errors found!`
  (was un-parseable for 35h).

## Prevention (why days were lost, and how to stop it recurring)

1. **CI smoke test on the motoko core** — `ailang check` a representative external consumer
   (the motoko `.ail` core) in ailang CI, so a language breaking-change fails the release, not
   the rig 35h later. This is the single highest-value follow-up.
2. **Treat lexer/escape changes as breaking by default** — when tightening escape grammar,
   keep legacy forms (or ship a deprecation warning window), don't hard-error.
3. **The rig can't self-diagnose a compile break** — 0-event motoko sessions must trip an
   alarm (a run that emits no `session_start` within N seconds is a harness/lang break, not a
   model failure). Ties into the rig-watchdog work (M-RIG-WATCHDOG-WEDGE).
