# std/clock & std/datetime: prompt and one guide say "seconds", implementation is milliseconds

- **Date**: 2026-09-17
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `rg -il "datetime|clock|timestamp" design_docs/`; `ls design_docs/planned/`; `grep -rn "seconds since epoch|int seconds|in seconds" *.{md,go,ail}`; none found — no existing design doc rules on the time-unit contract of std/clock or std/datetime, and no prior triage row touches it
- **Estimate**: n/a (see below — this is why it is design-doc)

## Why

The report is **verified and correct**; the implementation is milliseconds everywhere and
consistent, but two doc surfaces say seconds:

- Implementation: `internal/builtins/clock.go` (`_clock_now`) uses `time.Now().UnixMilli()`;
  `std/datetime.ail:4` states "All timestamps are Unix milliseconds in UTC";
  `std/clock.ail:16` states "Get current Unix timestamp in milliseconds". The `.ail`
  stdlib source is already right.
- Wrong surface 1: the teaching prompt (v0.16.6, current) — `prompts/v0.16.6.md:981` says
  `now() ... (seconds since epoch)` and `:995` says "Unix timestamps (int seconds)". The
  same text is mirrored in the embedded served copies `cmd/ailang/prompts/v0.16.1.md` …
  `v0.16.6.md` and the published `docs/docs/prompts/current.md` (+ versioned copies).
- Wrong surface 2: `docs/docs/guides/ai-stdlib-discovery.md:24` — "Returns epoch time in
  seconds". (Minor correction to the report: the std module docs themselves do say
  milliseconds; the silent mismatch lives in the prompt and this guide.)

Not covered by an existing doc — none found for terms "datetime clock milliseconds",
"seconds since epoch", "Unix timestamp" in `design_docs/`.

## Why design-doc, not direct-fix

Rubric rows 3 and 5 both fire:

- **Row 3 — more than one acceptable way.** The reporter explicitly offers two remedies:
  (a) reword the prompt/guide to say "milliseconds" (no code change, doc contract fix), or
  (b) add unambiguously-named seconds/ms variants (e.g. `nowMs`/`nowSec`, `makeDateSec`)
  — that is new public stdlib surface and needs an API ruling. A third variant exists:
  change the implementation to seconds and fix the four call sites, which would break
  existing programs. Someone could legitimately disagree between (a) and (b).
- **Row 5 — spans more than one file.** A reword-only fix still touches the prompt source
  (`prompts/`), the embedded served copies (`cmd/ailang/prompts/*.md` + `versions.json`
  bump), the published prompt docs, and `docs/docs/guides/ai-stdlib-discovery.md`. There
  is also a process question the design doc must answer: v0.16.6 is a pinned released
  snapshot — does the correction land in-place in v0.16.6 (breaking "the prompt is
  versioned truth") or in the next prompt version? That is a prompt-manager / release
  lane decision, not a two-line fix.

The reporter's own conclusion is the right frame: the ms choice itself is fine; the
deliverable is a single unambiguous unit contract stated identically in the prompt,
the stdlib docs, and the reference guides.
