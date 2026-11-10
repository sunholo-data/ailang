# tools/downstream — `make downstream-check`

Measures a release's blast radius before it ships: runs `ailang check` over every corpus in
`corpora.yaml` with a **baseline** and a **candidate** binary and reports only the difference.
Design: `design_docs/planned/v1_0_0/m-v1-downstream-compat-gate.md`.

## Run the release tier

```bash
make build                                   # or any candidate binary
make downstream-check CANDIDATE=./bin/ailang TIER=release   # BASELINE defaults to the latest release
```

Other knobs: `BASELINE=vX.Y.Z|<binary>`, `CORPORA=a,b`, `REF=default|pinned|worktree`,
`OUT=<dir>` (new or empty), `DOWNSTREAM_STATE=<dir>` (snapshot and baseline cache, default
`~/.ailang/state/downstream`), `OFFLINE=0|1`, `JOBS=n` (default 3; checks run under `nice -n 15`),
`TIMEOUT=s` (per check, default 120, retried once at 2×). The private corpora are fetched with
the operator's own `gh` auth.

**Package root.** Dependencies resolve from a per-run copy of `--package-root` (default
`~/.ailang/cache/registry`, about 60 MiB without compile caches) at `<OUT>/pkgroot`. The operator's
directory is only read: it is shared with every other `ailang` process on the machine, and on the
first real run a scheduled job compiled a registry package into it mid-run. The copy leaves out
`.ailang` compile caches, both arms read it, and the write audit covers it.

The report lands in `OUT` (default `<state>/runs/<timestamp>/`): `report.md`, `report.json`,
`rows.jsonl` (every entry) and `pins.md`.

## Exit codes

| code | verdict | meaning |
|---|---|---|
| 0 | PASS | every entry of every blocking corpus was measured, and no unacknowledged break |
| 2 | FAIL | an unacknowledged or over-`max` bucket in a blocking corpus, **or** an `expected-breaks.yaml` entry that matched nothing (a stale acknowledgement — remove it) |
| 3 | INSTRUMENT FAILURE | could not measure: a corpus not fetched, a binary that does not start, the built-in canary failed, bad configuration (e.g. an invalid `error_pattern`), an entry with a result from one arm only, or a write where nothing may be written (snapshots, read local checkouts, the run's package-root copy) |
| 4 | INCOMPLETE | a blocking corpus has `NOT_MEASURED` entries beyond its `allow_not_measured` (or an allowance that no longer excuses anything) |
| 64 | usage | a flag that does not parse (`--offline=maybe`) |

Precedence is 3 > 2 > 4: an instrument failure voids the run; a confirmed break is reported
even when coverage is also short (the verdict line names both).

**Through make**, the recipe's failure makes `make` itself exit 2 and print
`make: *** [downstream-check] Error N`, where N is the tool's code. For a script, run the
built tool directly (`go build -o bin/downstream ./tools/downstream && bin/downstream --help`)
or read `exit_code` from `report.json`. (The target used to `go run` the tool, which reports
every non-zero exit as 1.)

## Entry states

| state | gated? | meaning |
|---|---|---|
| `pass_both` | — | passes in both arms |
| **`NEWLY_FAILING`** | yes (its codes) | baseline passes, candidate fails |
| `fail_changed` | yes (new codes only) | both fail, different code sets |
| `timeout_cand` | yes (`TIMEOUT`) | the candidate timed out (after one retry at 2×) where the baseline reached a verdict |
| **`NOT_MEASURED`** | yes: counted against the allowance (exit 4); any real candidate error on it is a new code | the baseline reported only an infrastructure outcome — `deps_unresolved` (every code `DEP/*`), `timeout`, or `crash` (no parsable JSON, or "failed" with no diagnostic) — so "unchanged" would be a guess |
| `fail_diag_changed` | no: reported and counted | same code set, but the errors differ (message, location or count) after volatile text (each arm's scratch dir, the run dir, the package root, temp dirs) is normalised away. Messages change wording between versions, so this does not fail the gate by itself; read the list in `report.md` |
| `fail_same` | no | both fail with the same errors |
| `newly_passing` | no | the candidate passes where the baseline failed (or was unmeasured) |
| `expected_fail` | no | an intended failure: an `expected_fail` glob, or an examples-manifest entry with a non-zero `exit_code` **and** an `error_pattern` that the candidate's message matches. `exit_code` alone is a *run-time* exit code and excuses nothing |
| `MISSING_ARM` | instrument failure | a result from one arm only |

## NOT_MEASURED and `allow_not_measured`

A blocking corpus tolerates **zero** unmeasured entries by default. When some cannot be measured
yet, the operator has two choices, and the exit-4 message says which applies:

1. **Fix the cause.** `deps_unresolved`: provision the corpus's dependencies into the package root
   the gate copies (`--package-root`, default `~/.ailang/cache/registry`; the gate never writes
   it); `timeout`: raise `TIMEOUT=` or fix the hang; `crash`: read the rows.
2. **Acknowledge it, reviewed.** Add to the corpus in `corpora.yaml`:

   ```yaml
   allow_not_measured: {max: 41, reason: "daneel's private deps are not in the rig's package root (#NNNN)"}
   ```

   `max` and `reason` are both required. A count above `max` fails (exit 4), and an allowance
   with nothing left to excuse is stale and fails too, so it cannot silently cover a later loss.

Advisory corpora report their unmeasured entries and never gate on them.

## Process control

Each check runs in its own process group under a Go-side deadline (`--timeout`, default 120 s,
retried once at 2×). On the deadline the whole group is SIGKILLed — the compiler and anything it
started — and the wait for its output pipes is bounded. (A Go binary ignores `SIGALRM`, so the
earlier `perl alarm` wrapper never fired on the real compiler.) Windows builds but kills only the
compiler.

## Tests

`make test-downstream` — null control, planted break, expected-breaks accounting, not-measured
gating, a Go stand-in compiler that ignores SIGALRM and leaves a grandchild holding stdout, cache
healing, extraction caps, and the loud failures.
