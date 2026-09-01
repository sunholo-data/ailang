# docs-10 Sprint Plan

## Sprint summary

Close two independent vacuous-pass defects in the docs mission's example-verification
tooling: `scripts/verify_examples.go` never compares `examples/manifest.json`'s
per-entry `expected.stdout` (issue #670), and `scripts/validate_manifest.go` prints a
`checked` count it never asserts (issue #654). Two small, independent milestones —
M1 (stdout truth) and M2 (enumeration floor) — each landing with a negative control
that is *measured red on the pristine tree before the fix and green only because of
the fix*, not merely "the gate passes".

Both defects are confirmed present at `227d1e370`; both negative controls were run on
the unmodified tree and both currently exit 0 (see "Baseline measurements" below).

## Baseline measurements (run on the pristine worktree at `227d1e370`)

| Command | Result on the UNMODIFIED tree |
|---|---|
| `go build ./scripts/...` | rc=0, no output |
| `go run ./scripts/verify_examples.go --parallel 8 --json` | rc=0 — 217 total, 211 passed, **0 failed**, 6 skipped |
| `go run ./scripts/validate_manifest.go --ci` | rc=0 — `193 modules checked, 0 drift, 1 missing-on-disk` |
| `go run ./scripts/validate_manifest.go --dir /nonexistent/xyz --ci` | **rc=0** — `0 modules checked, 0 drift, 199 missing-on-disk` followed by the green `✓ manifest \`modules\` field is in sync` line. **This is the #654 vacuity, reproduced.** |

Manifest population (measured, `examples/manifest.json`, 199 entries):

- 20 entries carry a non-empty `expected.stdout`; 179 do not.
- 198 of 199 resolve on disk under `importextract.ResolvePath` semantics
  (`examples/<path>`, else `examples/runnable/<path>`); no two entries resolve to the
  same file, so a resolved-path keyed index is unambiguous.
- Of the 20 pinned entries, **16** resolve under `examples/runnable/` (the default
  sweep scope) and one of those 16 — `exit_code.ail` — is on `skippedExamples`.
  **The default gate will therefore perform exactly 15 stdout comparisons.** Under
  `--all` it performs 19.

Three facts that a naive implementation of #670 would trip over — all measured, not
assumed:

1. **`ailang run` prints a status preamble to STDOUT**, not stderr:
   `→ Type checking...\n→ Effect checking...\n✓ Running <path>\n`. Every one of the
   15 pinned entries therefore mismatches on a raw `stdout == expected` compare.
   A naive fix turns the gate red 15/15 for the wrong reason.
2. **`hello.ail` emits no trailing newline** (`Hello, AILANG!`) while its pin is
   `Hello, AILANG!\n`. The comparison must be trailing-newline-insensitive.
3. After preamble-stripping + trailing-newline-insensitive comparison, **17 of 19
   pinned entries match and 2 genuinely do not**:
   - `runnable/arithmetic.ail` — pin is `42\n3.14159\ntrue\nHello, AILANG!\n`; the
     example actually prints `x = 11\ny = 2\nz = 13\n`, which is what the example's
     own inline comments say it should print. The pin is a stale copy-paste from a
     different example. This is exactly the class of rot #670 exists to catch.
   - `runnable/process_stdin_write.ail` — pin includes three lines echoed by the
     `cat` subprocess (`hello from AILANG` / `streaming to subprocess` / `done!`)
     that the harness never captures, because `ailang` exits before `cat` flushes.
     Measured deterministic: 10/10 serial runs produced byte-identical stdout
     containing only `wrote line 1..3`, and both the default and `--all` harness
     runs agreed.

Consequently M1 **cannot land green without repairing those two pinned values** — see
the Scope note.

`make verify-examples-all` (`--all --threshold 60`) is **already red on the pristine
tree** (405 total / 305 passed / 94 failed; it exits non-zero on `failed > 0`
regardless of the threshold). Do not use it as an acceptance gate for either
milestone; it measures a pre-existing condition.

## Scope note — one file beyond the brief's Files list, tightly bounded

The brief lists only `scripts/verify_examples.go` and `scripts/validate_manifest.go`.
The baseline measurement above shows that is insufficient for M1 to land green: two
committed `expected.stdout` values are genuinely stale, so the new check correctly
turns `make verify-examples` red until they are corrected. M1 therefore also edits
**`examples/manifest.json`**, under a hard cap:

- ONLY the `expected.stdout` string of `runnable/arithmetic.ail` and
  `runnable/process_stdin_write.ail` may change.
- No other field, no other entry, no reformat of the file, and **no backfilling of
  the 179 entries with an empty `expected.stdout`** — that remains a non-goal.
- `git diff examples/manifest.json` at the end of M1 must show exactly two changed
  string values and nothing else.

`examples/*` is inside the mission's planner allowlist, so this is a bounded widening
of the brief's Files list on measured evidence — not a scope drift.

---

## M1 — Enforce `expected.stdout` in verify_examples.go (#670)

Estimated effort: 0.5–1 day; medium risk (the comparison must not go vacuously red).

Files: `scripts/verify_examples.go`, `examples/manifest.json` (two pinned values only).

### Implementation

**Task 1a — build a resolved-path → pinned-stdout index, once, before any goroutine.**

In `scripts/verify_examples.go` (which already carries `//go:build ignore` and is run
via `go run`), add imports of `github.com/sunholo-data/ailang/internal/manifest` and
`github.com/sunholo-data/ailang/scripts/internal/importextract` — the same two
packages `scripts/validate_manifest.go` already uses, so no new dependency enters the
tree.

Add package-level state next to the existing `useAllExamples` / `useTrace` block:

- `var pinnedStdout map[string]string` — key: `filepath.ToSlash(filepath.Clean(resolvedPath))`, value: the non-empty `Expected.Stdout`.
- `var pinnedTotal int` — count of manifest entries with a non-empty `Expected.Stdout`, resolvable or not (used only in the floor's diagnostic message).
- `var stdoutChecked atomic.Int64` (`sync/atomic`) — comparisons actually performed.

Populate it from `main()`, immediately after flag parsing and before any of the
`verifyExamples*` / `runUpdateBaselines` dispatch, via a `loadPinnedStdout()` helper:

- `m, err := manifest.Load("examples/manifest.json")`. On error, print
  `INSTRUMENT FAILURE: verify_examples could not load examples/manifest.json: <err>`
  to stderr and `os.Exit(1)`. **No silent fallback** — a verifier that cannot read
  its own oracle must not report a clean run (CLAUDE.md Principle 2).
- For each entry: skip if `ex.Expected == nil || ex.Expected.Stdout == ""`; else
  `pinnedTotal++`, then `full, ok := importextract.ResolvePath("examples", ex.Path)`
  and, if `ok`, record `pinnedStdout[filepath.ToSlash(filepath.Clean(full))] = ex.Expected.Stdout`.
  Do **not** hand-roll the path resolution — reuse `ResolvePath`, so this tool and
  `validate_manifest.go` cannot disagree about which file an entry names. (Measured:
  106 of 199 entries use a flat `hello.ail`-style path that only resolves under
  `examples/runnable/`, so string-munging the walked path would silently miss them.)

Populating from `main()` (single-goroutine) and only *reading* the map inside
`runExample` keeps the parallel path (`runExamplesParallel`, default `parallelism=8`)
race-free without a mutex; only the counter is atomic.

**Task 1b — compare in `runExample`, after the existing pass/fail determination.**

Insert the comparison after the `err != nil` / stderr-scan block that currently sets
`result.Status` (the block ending at the `result.Status = "passed"` assignment), and
*before* the `useTrace` trace-verification block. Guard it with:

- `if useTrace { skip }` — under `--trace` the run is invoked with `--emit-trace jsonl`
  and trace JSONL is interleaved with program output on stdout, so a stdout comparison
  there would be measuring the trace, not the program. Skipping is correct; the floor
  in Task 1c is skipped for the same reason.
- `if result.Status != "passed" { skip }` — a failed or skipped example already has a
  verdict; do not double-report, and do not count it toward the floor.
- Look up `pinnedStdout[filepath.ToSlash(filepath.Clean(filename))]`; if absent, skip
  (this is the 179-entry / unpinned majority, which must remain unaffected).

When a pin is present:

1. `stdoutChecked.Add(1)` (count the comparison, whatever its outcome).
2. Strip the runner preamble from the captured `stdout.String()` with a small helper
   `stripRunPreamble(out, filename string) string`: build the anchor
   `"✓ Running " + filepath.ToSlash(filename) + "\n"`; if `strings.Index(out, anchor)`
   is `>= 0`, return everything AFTER that anchor; otherwise return `out` unchanged.
   Anchoring on the example's own path makes this deterministic and unable to eat
   program output unless the program prints that exact line. **Do not** add `--quiet`
   to the run arguments instead: that would change `result.Output` (and hence
   `examples_report.json`) for every example and make the pinned examples run under
   different flags from the gated set.
3. Compare `strings.TrimRight(got, "\n") == strings.TrimRight(want, "\n")` — byte-exact
   apart from trailing newlines (required: `hello.ail` emits no trailing newline; see
   Baseline fact 2). Do **not** trim leading whitespace, do not normalise interior
   whitespace, and do not fall back to a substring/`HasSuffix` test — those would
   re-introduce vacuity.
4. On mismatch: `result.Status = "failed"` and set `result.Error` to a message that
   shows both sides, e.g.
   `fmt.Sprintf("stdout mismatch vs manifest expected.stdout\n  expected: %q\n  actual:   %q", want, got)`.
   Use `%q` so trailing-newline and invisible-character differences are legible.

**Task 1c — the anti-vacuity floor.**

After the sweep completes and before the existing exit decisions, in **all three** of
`verifyExamplesPlain`, `verifyExamplesJSON` and `verifyExamplesMarkdown` (a shared
`enforceStdoutFloor()` helper called from each is preferred), assert:

```
if !useTrace && stdoutChecked.Load() == 0 {
    fmt.Fprintf(os.Stderr, "INSTRUMENT FAILURE: verify_examples compared 0 pinned expected.stdout values (manifest has %d non-empty pins)\n", pinnedTotal)
    os.Exit(1)
}
```

Do not apply the floor in `runUpdateBaselines` (it is a generator, not a gate) and do
not apply it under `--trace` (comparison is deliberately disabled there).

Ordering in JSON mode: emit the floor message **after** the report has been encoded to
stdout, so the report is still written. Note that `make verify-examples` redirects
`2>&1` into `examples_report.json`, so a tripped floor will append a non-JSON line to
that artifact — this matches the existing behaviour of the threshold failure path and
is acceptable because the gate is failing anyway. Do not restructure the makefile to
avoid it (out of scope).

In `verifyExamplesPlain` only, also print the count in the summary block, e.g.
`fmt.Printf("  Stdout pins checked: %d\n", stdoutChecked.Load())`. Plain mode is not
consumed by the gate, so this is free, and it is what makes the positive control in
AC-3 observable. Do not add a field to `reporttypes.ExampleResult` /
`VerificationReport` — that file is out of scope.

**Task 1d — repair the two genuinely stale pins (`examples/manifest.json`).**

- `runnable/arithmetic.ail`: set `expected.stdout` to `"x = 11\ny = 2\nz = 13\n"`.
- `runnable/process_stdin_write.ail`: set `expected.stdout` to
  `"wrote line 1\nwrote line 2\nwrote line 3\n"`.

Edit these two string values in place with a text edit; do not round-trip the file
through a JSON dumper (that reformats all 199 entries). Do not run
`go run ./scripts/backfill_manifest_modules.go` as part of this task — **measured: it
rewrites `examples/manifest.json` byte-for-byte even when drift is 0.**

If, on the executor's machine, the new check flags any pinned entry *other* than these
two, that is a new finding: report it with the `%q` diff and stop. Do not loosen the
comparator to make it green, and do not clear a pin to silence it. (If
`process_stdin_write.ail` proves flaky rather than deterministic — run it 10× and
compare hashes; it was 10/10 identical here — the correct response is to clear that
one pin to `""` and record why in the entry's `description`, not to weaken the
comparator.)

### Non-goals (binding)

Do not backfill the 179 empty `expected.stdout` entries; do not add flag-gated staging
or a report-only mode; do not touch `scripts/internal/reporttypes`, `make/examples.mk`,
`internal/manifest`, `cmd/**` or `internal/**`; do not move `modules`-drift
responsibility into this file.

### Acceptance criteria

1. **Negative control — a corrupted pin must FLIP the result.** With the fixed tree:

   ```bash
   cp examples/manifest.json /tmp/docs10_manifest.bak
   shasum /tmp/docs10_manifest.bak                       # record
   python3 - <<'PY'
   p = 'examples/manifest.json'
   s = open(p).read()
   # hello.ail's pin, replaced TEXTUALLY so the other 198 entries are untouched.
   old = '"stdout": "Hello, AILANG!\\n"'
   assert s.count(old) == 1, 'expected exactly one occurrence, got %d' % s.count(old)
   open(p, 'w').write(s.replace(old, '"stdout": "DELIBERATELY WRONG\\n"'))
   PY
   go run ./scripts/verify_examples.go --parallel 8 --json > /tmp/docs10_neg.json 2>&1; echo "rc=$?"
   cp /tmp/docs10_manifest.bak examples/manifest.json
   shasum examples/manifest.json                          # must equal the recorded hash
   ```

   Required: rc **1** (it is rc 0 today — measured), `/tmp/docs10_neg.json` shows
   `runnable/hello.ail` with `"status": "failed"` and an error naming both the
   expected and actual strings, and the restore is byte-identical (hashes match).
2. **Negative control — the floor must FLIP too.** Temporarily point the index at a
   manifest with no pins and confirm the floor fires rather than a clean run:

   ```bash
   python3 - <<'PY'
   import json
   d = json.load(open('examples/manifest.json'))
   n = 0
   for e in d['examples']:
       if e.get('expected', {}).get('stdout'):
           e['expected']['stdout'] = ''
           n += 1
   assert n == 20, n
   json.dump(d, open('/tmp/docs10_nopins.json', 'w'), indent=2)
   PY
   ```

   (A JSON round-trip is safe here because the output is a throwaway temp file — do
   NOT round-trip the real `examples/manifest.json`, which would reformat all 199
   entries. A regex over the raw text is not safe: several pinned values contain
   escaped quotes, e.g. `json_jint.ail`.)

   Run the verifier with `loadPinnedStdout()` reading `/tmp/docs10_nopins.json` (a
   one-line temporary edit, reverted immediately — do NOT ship a `--manifest` flag).
   Required: rc **1** and stderr containing
   `INSTRUMENT FAILURE: verify_examples compared 0 pinned expected.stdout values`.
   Revert the temporary edit and confirm `git diff scripts/verify_examples.go` no
   longer mentions `/tmp/`.
3. **Positive control — the check is actually running and is not vacuous.**
   `go run ./scripts/verify_examples.go --parallel 8` prints
   `Stdout pins checked: 15` (exactly 15 — 16 pinned entries resolve under
   `examples/runnable/`, minus `exit_code.ail`, which is on `skippedExamples`) and
   exits 0.
4. **No collateral reds.** `go run ./scripts/verify_examples.go --parallel 8 --json`
   exits **0** with 217 total / 211 passed / **0 failed** / 6 skipped — the same
   triple as the pristine baseline. An example with no pinned `expected.stdout` must
   be unaffected.
5. **Manifest diff is exactly two strings.** `git diff examples/manifest.json` shows
   only the `expected.stdout` values of `runnable/arithmetic.ail` and
   `runnable/process_stdin_write.ail`; `git diff --stat` shows 2 files changed in
   total for M1 (`scripts/verify_examples.go`, `examples/manifest.json`).
6. `go build ./scripts/...` exits 0, and `gofmt -l scripts/verify_examples.go` prints
   nothing.

### Verification (what the planner actually ran, pristine tree, `227d1e370`)

- `go build ./scripts/...` → rc 0.
- `go run ./scripts/verify_examples.go --parallel 8 --json` → rc 0; 217/211/0/6.
- Compared every pinned `expected.stdout` against the report's captured `output`:
  0/15 match raw (all differ by the stdout preamble) → the preamble finding.
- Re-compared with anchor-stripping + trailing-newline trimming across `--all`:
  **17 match, 2 mismatch** (`arithmetic.ail`, `process_stdin_write.ail`), 1 skipped.
- `./bin/ailang run --caps Process,IO examples/runnable/process_stdin_write.ail` ×10 →
  10/10 identical stdout hashes (`a3a0f98d0c7b`).
- `scripts/verify_examples.go` has **no `--help`**: `main()` hand-parses `os.Args` and
  silently ignores unknown flags, so `--help` would run the full sweep. Use
  `--json` / `--markdown` / plain as above.

---

## M2 — Assert the `checked` floor in validate_manifest.go (#654)

Estimated effort: <0.5 day; low risk. Independent of M1 — no shared code path.

Files: `scripts/validate_manifest.go`.

### Implementation

In `main()`, the drift loop increments `checked` (currently line ~83) and the summary
line prints it (~line 93), but the only non-zero exit is `driftCount > 0` (~103–108).
Insert the floor **after** the summary `fmt.Printf` and **before** the
`if driftCount > 0 { … } else { … }` block:

```go
if checked == 0 {
    fmt.Fprintf(os.Stderr, "\n%s validate_manifest enumerated 0 modules (%d entries in manifest, %d missing on disk)\n",
        red("INSTRUMENT FAILURE:"), len(m.Examples), missingCount)
    fmt.Fprintf(os.Stderr, "A zero-enumeration means the instrument is broken, not that the manifest is clean.\n")
    os.Exit(1)
}
```

Requirements:

- The exit must be unconditional on `*ciMode` — a zero enumeration is a broken
  instrument in both modes, not a drift-tolerance question (brief, §2).
- It must run **before** the green `✓ manifest \`modules\` field is in sync with actual
  imports` line, so that line is never printed on a zero enumeration.
- Use the honest floor `checked == 0`. Do **not** introduce a fixed threshold such as
  `>= 150`; that needs a maintenance story that is out of scope.
- Do not change `missingCount` handling (still a warning, per the file's header
  comment), do not change the schema-load failure path, and do not alter the
  `driftCount` exit semantics.
- Keep the header comment accurate: add one sentence to the existing block comment
  noting that a zero enumeration is a hard failure.

### Acceptance criteria

1. **Negative control — the exit code must FLIP.**
   `go run ./scripts/validate_manifest.go --dir /nonexistent/xyz --ci; echo $?` →
   **1** after the fix (**measured 0 on the pristine tree**, printing
   `0 modules checked, 0 drift, 199 missing-on-disk` and then the green line).
   Its stderr must contain `INSTRUMENT FAILURE:` and `enumerated 0 modules`, and its
   stdout must **not** contain `field is in sync with actual imports`.
2. **The same control flips in non-CI mode.**
   `go run ./scripts/validate_manifest.go --dir /nonexistent/xyz; echo $?` → **1**
   (also 0 on the pristine tree).
3. **Positive control — a real run is unaffected.**
   `go run ./scripts/validate_manifest.go --ci; echo $?` → **0**, printing
   `193 modules checked, 0 drift, 1 missing-on-disk` and the green in-sync line —
   identical to the pristine baseline.
4. **The pre-existing drift arm still works.** `make validate-manifest-selftest`
   passes. ⚠ Measured: that target runs `backfill_manifest_modules.go`, which
   **rewrites `examples/manifest.json` byte-for-byte even at 0 drift.** Back the file
   up first (`cp examples/manifest.json /tmp/docs10_m2.bak`) and restore it after,
   then confirm `shasum` matches and `git status --porcelain examples/manifest.json`
   is clean (or, after M1, shows only M1's two-string diff).
5. `go build ./scripts/...` exits 0 and `gofmt -l scripts/validate_manifest.go` prints
   nothing. `git diff --stat` for M2 shows exactly one file changed.

### Verification (what the planner actually ran, pristine tree, `227d1e370`)

- `go run ./scripts/validate_manifest.go --ci` → rc 0,
  `193 modules checked, 0 drift, 1 missing-on-disk`, green line printed.
- `go run ./scripts/validate_manifest.go --dir /nonexistent/xyz --ci` → **rc 0**,
  `0 modules checked, 0 drift, 199 missing-on-disk`, green line printed. The #654
  vacuity is reproduced, and this exact command is a working negative control that
  reaches the `checked == 0` branch **without** tripping the schema-load path (an
  empty-`examples` temp manifest would fail `manifest.Load`'s statistics check first
  and would therefore be a vacuous control).
- `go run ./scripts/backfill_manifest_modules.go` → rc 0 but changed
  `examples/manifest.json`'s hash (`600e9016…` → `3574d75e…`); restored from backup.
- `go build ./scripts/...` → rc 0.

---

## Sequencing and reporting

M1 and M2 touch disjoint files and can be done in either order or in parallel. The
executor should report, for each milestone, the negative control's **before** and
**after** exit codes side by side — a milestone whose negative control was never
observed red on the pristine tree has not been verified, only asserted.
