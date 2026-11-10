# tools/downstream — `make downstream-check`

Measures a release's blast radius before it ships: runs `ailang check` over every corpus in
`corpora.yaml` with a **baseline** and a **candidate** binary and reports only the difference.
Design: `design_docs/planned/v1_0_0/m-v1-downstream-compat-gate.md`.

## Run the release tier

```bash
make build                                   # or any candidate binary
make downstream-check CANDIDATE=./bin/ailang TIER=release FETCH_DEPS=1   # BASELINE defaults to the latest release
```

Other knobs: `BASELINE=vX.Y.Z|<binary>`, `CORPORA=a,b`, `REF=default|pinned|worktree`,
`OUT=<dir>` (new or empty), `DOWNSTREAM_STATE=<dir>` (snapshot and baseline cache, default
`~/.ailang/state/downstream`), `OFFLINE=0|1`, `FETCH_DEPS=0|1` (below), `JOBS=n` (default 3;
checks run under `nice -n 15`), `TIMEOUT=s` (per check, default **600**, retried once at 2×; at
240 s motoko's root package timed out in the baseline on a loaded machine and read as 22 false
breaks). The private corpora are fetched with the operator's own `gh` auth.

**Package root.** Dependencies resolve from a per-run copy of `--package-root` (default
`~/.ailang/cache/registry`, about 60 MiB without compile caches) at `<OUT>/pkgroot`. The operator's
directory is only read: it is shared with every other `ailang` process on the machine, and on the
first real run a scheduled job compiled a registry package into it mid-run. The copy leaves out
`.ailang` compile caches, both arms read it, and the write audit covers it.

## Dependency provisioning

Before the arms run, each corpus gets the dependencies its own manifests and locks declare, by
the **baseline** binary (the released `ailang lock` / `ailang install`) in its own scratch HOME
(`<OUT>/arms/provision`), so both arms read one provisioned input:

| package dir (an `ailang.toml`, pruned and excluded as discovery is) | what the gate does |
|---|---|
| committed `ailang.lock` that parses | honoured as committed; each registry version it pins must be in the package root copy |
| no lock, `[dependencies]` not empty | `ailang lock` in `<OUT>/provisioned/<corpus>`, a copy of the snapshot (path deps resolve from the snapshot's content) |
| committed lock that does not parse (e.g. a names-only fixture stub) | removed in the copy and regenerated there, and reported as a **FINDING** for the corpus owner at the top of `report.md` |
| no lock, no dependencies | nothing: the check needs no lock |

**The candidate's own lock.** A corpus that commits no lock (daneel gitignores `ailang.lock`; its
CI runs `ailang lock` with the deployed binary) depends on the candidate's resolver too. So
wherever the gate generates a lock, the candidate also runs `ailang lock` in its own copy
(`<OUT>/provisioned-candidate/<corpus>`), and the two resolved sets (name, version, source) are
compared as an entry of kind `lock` (`<dir>/ailang.lock`):

| baseline lock | candidate lock | state |
|---|---|---|
| ok | same set | `pass_both` |
| ok | fails | **NEWLY_FAILING** `LOCK/failed` — a break |
| ok | a different set | **NEWLY_FAILING** `LOCK/resolution-differs` — a break: both arms' checks read the **baseline's** lock, so a different candidate tree is a change the checks never saw (the message names both sets) |
| fails | fails | NOT_MEASURED (`DEP/lock`) |
| fails | ok | `newly_passing` |

**What the write audit covers.** The run's own inputs: each snapshot, each provisioned tree and the
package root copy. A write into any of those is exit 3. The operator's sibling checkouts are *not*
inputs — every arm reads a snapshot (in `REF=worktree` mode, a copy taken before the run) with a
scratch HOME and cwd — so a path that changes there while the gate runs is listed in the report
under "Local checkouts that changed during the run" and **never gated**: on a machine where several
sessions edit those trees it is somebody else's work, not a write by the gate.

The snapshot itself is never written (the write audit's snapshot stamp is taken before
provisioning); the provisioned copy and the package root copy are audited from the moment
provisioning ends. Entries are discovered in the provisioned copy when there is one.
`ailang lock` never gets the network (its HTTP(S) proxy is a closed port, as in CI), every
provisioning command has the `--timeout` deadline, and provisioning as a whole has a 30-minute
budget; running out is exit 3.

**`FETCH_DEPS=1`** (off by default; needs the network, refused with `OFFLINE=1`): a registry
version missing from the package root copy — pinned by a committed lock, named by a manifest
being locked (including through its path deps inside the snapshot), or a registry dependency of a
package fetched this way — is fetched with `ailang install --no-bin <name>@<version>` into the
provision arm's HOME and copied into `<OUT>/pkgroot`; the operator's package root is not touched.
Nothing is published and no credential is used: the public registry is read. Only the registry's
`HTTP 404` for that version leaves it **UNPROVISIONED** and named; any other fetch failure (a 5xx,
no network, a timeout) is an instrument failure, exit 3. Without `FETCH_DEPS`, a missing version
is named with the hint to rerun with it. Git dependencies are read from the package root only;
the gate never clones.

`report.md` has a **Dependency provisioning** section (per corpus: locks committed / generated /
regenerated / FAILED; candidate lock same / DIFFERS / FAILED; packages by source; every fetched
package; every UNPROVISIONED dependency), and `report.json` lists every package each corpus's
locks name with its version, content hash and source (`package root copy`, `registry fetch`,
`path <rel>`, or `MISSING`).

The report lands in `OUT` (default `<state>/runs/<timestamp>/`): `report.md`, `report.json`,
`rows.jsonl` (every entry) and `pins.md`.

## Exit codes

| code | verdict | meaning |
|---|---|---|
| 0 | PASS | every entry of every blocking corpus was measured, and no unacknowledged break |
| 2 | FAIL | an unacknowledged or over-`max` bucket in a blocking corpus, **or** an `expected-breaks.yaml` entry that matched nothing (a stale acknowledgement — remove it) |
| 3 | INSTRUMENT FAILURE | could not measure: a corpus not fetched, a binary that does not start, the built-in canary failed, bad configuration (e.g. an invalid `error_pattern`), an entry with a result from one arm only, a `FETCH_DEPS=1` fetch that failed for any reason but HTTP 404 (or `FETCH_DEPS=1` with `OFFLINE=1`), or a write where nothing may be written (snapshots, read local checkouts, the run's package-root copy and provisioned trees once provisioning is done) |
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
| **`NOT_MEASURED`** | yes: counted against the allowance (exit 4); any real candidate error on it is a new code | the baseline reported only an infrastructure outcome — `deps_unresolved` (every code `DEP/*`), `timeout`, or `crash` (no parsable JSON, or "failed" with no diagnostic) — so "unchanged" would be a guess — or the loader stopped before type checking anything (measured: a type error planted in such a file is not reported): `invalid_manifest` (the entry's own `ailang.toml` was rejected, `UNCHECKED/invalid-manifest`; a dependency's bad manifest is `DEP/setup`) and `unexported_import` (it imports a module a package does not export, `UNCHECKED/not-exported`). These need a fix in the corpus or a reviewed allowance |
| `fail_diag_changed` | no: reported and counted | same code set, but the errors differ (message, location or count) after volatile text (each arm's scratch dir, the run dir, the package root, temp dirs) is normalised away. Messages change wording between versions, so this does not fail the gate by itself; read the list in `report.md` |
| `fail_same` | no | both fail with the same errors |
| `newly_passing` | no | the candidate passes where the baseline failed (or was unmeasured) |
| `expected_fail` | no | an intended failure: an `expected_fail` glob, or an examples-manifest entry with a non-zero `exit_code` **and** an `error_pattern` that the candidate's message matches. `exit_code` alone is a *run-time* exit code and excuses nothing |
| `MISSING_ARM` | instrument failure | a result from one arm only |

## NOT_MEASURED and `allow_not_measured`

A blocking corpus tolerates **zero** unmeasured entries by default. When some cannot be measured
yet, the operator has two choices, and the exit-4 message says which applies:

1. **Fix the cause.** `deps_unresolved`: read the UNPROVISIONED lines of the report's
   provisioning section — rerun with `FETCH_DEPS=1` for a registry version the package root
   lacks; a version the registry does not have, or a git dependency, has to reach the package
   root the gate copies (`--package-root`, default `~/.ailang/cache/registry`; the gate never
   writes it) some other way; `timeout`: raise `TIMEOUT=` or fix the hang; `crash`: read the rows;
   `invalid_manifest` / `unexported_import`: the corpus owner fixes the manifest or the import.
2. **Acknowledge it, reviewed.** Add to the corpus in `corpora.yaml`:

   ```yaml
   allow_not_measured: {max: 41, reason: "daneel's private deps are not in the rig's package root (#NNNN)"}
   ```

   `max` and `reason` are both required. A count above `max` fails (exit 4), and an allowance
   with nothing left to excuse is stale and fails too, so it cannot silently cover a later loss.

Advisory corpora report their unmeasured entries and never gate on them.

## Process control

Each check runs in its own process group under a Go-side deadline (`--timeout`, default 600 s,
retried once at 2×, so one hung entry costs at most 30 minutes; provisioning commands get the
same deadline, without the retry). On the deadline the whole group is SIGKILLed — the compiler and anything it
started — and the wait for its output pipes is bounded. (A Go binary ignores `SIGALRM`, so the
earlier `perl alarm` wrapper never fired on the real compiler.) Windows builds but kills only the
compiler.

## Tests

`make test-downstream` — null control, planted break, expected-breaks accounting, not-measured
gating, a Go stand-in compiler that ignores SIGALRM and leaves a grandchild holding stdout, cache
healing, extraction caps, the loud failures, and dependency provisioning (a fixture corpus with a
path dependency, a cached one, a missing one, a lock pinning a version only the registry has, and
an unreadable committed lock; offline, fetch, unreachable registry).
