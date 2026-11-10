# M-V1-DOWNSTREAM-COMPAT-GATE: Measure a Release's Blast Radius Before It Ships

**Status**: Planned
**Target**: v0.58.x (must exist before v1.0.0-rc.1; it is the instrument for the last breaking window)
**Priority**: P0 (release-readiness review §4.3 step 4: "the highest-value new instrument in this review")
**Estimated**: 7–8 working days across 5 milestones (estimate, not a plan; see §12)
**Dependencies**: none to start. M3 hooks into the `release-manager` skill and the sibling doc
[m-v1-release-gates.md](m-v1-release-gates.md); one optional AILANG-fix-lane item (§6.6).
**Planner-Lane**: opus-required for M1–M3 (isolation and freshness are where this goes wrong silently); codex-ok for M4–M5

## 1. Problem Statement

v0.58.x is the last window for breaking changes before the 1.0 promise. Several are queued
(module-qualified type identity, non-exhaustive match becomes an error, `show` output, stdlib
renames — review §3), on top of three unreleased BREAKING fragments (#2184, #2185, named tests).
Under `stability.md`, any break we *miss* now becomes a 2.0 problem later.

**Current state (measured 2026-11-10):**

- **The last two breaking changes were validated by hand.** The V1 dashboard's "#2184 release
  order" (measured old-vs-new binary across downstream repos; five ordered steps; "106 breaks → 0"
  in the motoko fork) and the #2185 IFC-site list were produced in-session with throwaway scripts.
  The 2026-11-03 corpus sweep scripts (`relcmp.sh`, `candcmp.sh`) lived in an ephemeral
  scratchpad and are gone. Nothing in `Makefile`, `make/`, `scripts/` or `tools/` re-runs any of it.
- **The one consumer gate in CI is one file wide and its local arm is dead.**
  `tools/ci/motoko_smoke.sh` (CI step "Consumer regression guardrail") checks a single vendored
  file, `internal/eval_harness/testdata/consumer_smoke.ail`, plus the real motoko core *if* it
  exists at `${MOTOKO_REPO:-$HOME/dev/mk-ast}`. `~/dev/mk-ast` no longer exists (the rig runs
  `~/dev/mk-20261027`, MOTOKO.md), so that arm prints "skipped" and passes.
- **Pins are scattered and inconsistent** (§3): v0.57.1, v0.54.18, v0.52.5, v0.41.0 + `latest`,
  "dev HEAD", and "whatever is on PATH" across seven repos. No report says who is on what.
- **Local checkouts are not a reliable picture of downstream.** 6 of 7 downstream checkouts on this
  machine are behind their remote default branch, and in four of them the missing commits are
  exactly the #2184/#2185 fixes (§3). A sweep over the working trees reports already-fixed breaks
  as new ones (§4 shows it doing so).

**Impact:** every breaking fragment in v0.58.x either costs an attended half-day of hand sweeping
or ships blind. The review's soak (step 5) requires "downstream pins moved and their CI green",
which is unverifiable without this instrument.

## 2. Goals

**Primary goal:** one command, `make downstream-check CANDIDATE=<path-to-ailang>`, answers *"what
breaks for real users if we ship this binary?"* — newly failing entry points grouped by error code
and by repo, with a suggested release order — in about 15 minutes for the release tier (eval corpus excluded).

**Success metrics:**

- Reproduces the #2184 hand measurement (motoko fork: 106 newly failing at the pre-fix fork commit,
  0 at `597a6fbf`) with no hand edits — the M2 acceptance control (the §4 prototype already did).
- Null control (candidate = baseline) reports 0 differences on every corpus, every run.
- Zero bytes written into any downstream checkout (verified by an mtime scan in the report).
- Every v0.58.x release carrying a BREAKING fragment has a committed report and an
  `expected-breaks.yaml` entry for each acknowledged (corpus, code) bucket.

## 3. Corpus Inventory (measured 2026-11-10, read-only; no git operations were run)

`.ail` counts exclude `.git`, `.ailang/`, `.claude/`, worktree copies (`.wt-*`), `node_modules`,
build outputs. HEADs were read from `.git` files; remote heads via `gh api` (read-only).

| Corpus | Vis. | `.ail` | Manifests | AILANG pin as declared (file) | What "passing" means in its own CI | Local checkout vs remote |
|---|---|---|---|---|---|---|
| **ailang (in-repo)** | public | 902: examples 556, internal testdata 86, tests 77, std 51, benchmarks 41, cmd 30 | — | *is* the candidate | `make verify-examples` (runnable subset), stdlib freeze gate, `motoko_smoke.sh` | — |
| **docparse** | private | 44 | 1 (6 registry deps) | v0.57.1: `Dockerfile:51` `ARG AILANG_VERSION`, `cloudbuild-ci.yaml:14`, `cloudbuild-deploy.yaml:14` | `scripts/ci/checks.sh`: `ailang lock`, `ailang verify`, `ailang run` of `tests/verify_*.ail` | `main` @3213277 vs origin @7950010 (**#263 missing locally**) |
| **ailang-parse** | public | 93 | 2 | **none for CI**: `ci.yml:31` clones and builds `ailang` **dev HEAD**; a second job builds the `ailang.toml` floor (`>=0.50.0`). `Dockerfile:4` `AILANG_REF=v0.24.0` (stale) | `ailang lock`, `./bin/docparse --check` / `--prove`, inline tests, python checks | branch `bench/extend-adapter` vs origin `main` @ee4deb7 |
| **email-parse** (eparse) | private | 72 (49 outside `scripts/wip`) | 3 (root + `packages/email`, `packages/eparse`) | **none**: no CI; runs the PATH binary; floors `>=0.14.0` / `>=0.40.0` | manual `scripts/test_*.sh` | `main` @719b225 (2026-10-07) vs origin @ddfb700 (**#40/#41/#43/#44 missing**) |
| **daneel** | private | 80 local / 92 origin | 14 / 16 | v0.54.18 + sha256 in `cloudbuild-ci.yaml:25` **on origin `main`**; the local branch predates that file and still carries `guard.yml:22` v0.53.3 | `tools/ci.sh`: `ailang check` + verify on every module, manifest lint | branch `docs/site-input-rollout` (10-20) vs origin @0d6e670 (**#417 missing**) |
| **world** (ailang-world) | public | 39 | 1 (`packages/world-core`) | `ci.yml` installs `releases/latest` **and** a sha-verified v0.41.0 "PINNED" leg; `go.mod` requires the ailang Go module v0.47.2; attended `se_smoke.sh` wants v0.41.0 + v0.54.15 | `go test` incl. `host/verifygate` (gated on the compiler version string) | `dev` @3a96cf7 vs origin @b7d2e62 |
| **motoko fork** | public | 575 (incl. `.agent/` evidence probes) | 25 | `scripts/install-prerequisites.sh:40` `AILANG_REF="v0.52.5"` on our rig branch (Makefile enforces it equals the `ailang.toml` floor); upstream `main` says v0.54.14; fork `main` v0.19.1. **The rig actually runs the PATH binary** (MOTOKO.md §2) | `make check_core && make verify_extensions` | `mk-20261027` @597a6fb = origin (fresh) |
| **ailang-packages** | public | 171 local / **600 origin** | 45 local / **61 origin** package dirs (review said 42) | per-package `ailang = ">=…"` floors only; no CI workflow in the repo | the registry validator runs `ailang check --package .` at publish | branch `build/v0.4.0/…` @e998928 (**2026-09-19**) vs origin @6854907 (**#183 missing**) |
| **registry (published)** | public | n/m | 66 package names cached in `~/.ailang/cache/registry` | each package's floor | validator: `check --package .` | the cache holds only versions someone installed |
| **eval corpus** | public (tracked) | **16,162** unique `compile_ok` AILANG programs (19,363 rows in 41,070 JSON files under `eval_results/`) | — | — | it compiled once | — |
| *others found* | mixed | stapledons-godot 199 source (+4.6k cached/generated), decision-budget 27 (private), ailang_bootstrap 28, stapledons_voyage 23, cphu-aipla-app 20, aipla 18 | — | stapledons-godot `Makefile:17` `AILANG_RELEASE ?= v0.56.1` | — | advisory tier only |

Notes. `benchmarks/*.yml` (99 specs) carry no AILANG reference solutions (2 of 99 have a
solution-like key); the 41 `.ail` under `benchmarks/` are runtime perf workloads. The eval corpus
is the stand-in for "what models write". World's pinned leg is a deliberate publish fence for its
`world/core` packet, not drift (a 2026-09-21 bump attempt failed 17 `verifygate` tests by design;
whether the packet has since published was not re-checked).

## 4. Prototype Evidence (2026-11-10, check-only, by hand)

**Arms.** Baseline = the v0.57.4 release asset (`darwin.arm64`, sha256 verified); candidate =
`dev` @ `51f6636` (the review's measurement point; includes #2184 and #2185). Each binary copied
into its own empty dir. Per arm: scratch `HOME`, `AILANG_CACHE_DIR`, `AILANG_STATE_DIR`; every
`AILANG_*` scrubbed; `AILANG_PACKAGE_ROOT` = the existing local registry cache (read-only);
`AILANG_TRACE=off`; `check --json` per file, cwd from the module-line rule (§6.4), `perl` alarm
120 s, `nice -n 15`, 3 parallel. Studio load average 13–23 from other work throughout. About 150
lines of Python; scripts and raw JSONL are in the session scratchpad, not committed.

**Run A — local working checkouts as they sit** (the "obvious" design):

| Corpus | Files | pass both | **newly failing** | fail both |
|---|---|---|---|---|
| examples (copied out of the repo) | 556 | 474 | **2** | 80 |
| docparse | 44 | 22 | **22** | 0 |
| email-parse | 49 | 15 | **34** | 0 |
| daneel | 80 | 64 | **1** (+1 changed) | 14 |
| motoko fork (`mk-20261027`) | 575 | 422 | 0 | 153 |
| ailang-packages, ailang-parse, world | 303 | 291 | 0 | 12 |
| **Total** | **1,607** | 1,288 | **59** | 260 |

Wall time 178 s (baseline) + 162 s (candidate). All 57 downstream breaks are #2184 ("The pure
keyword contradicts the declared effect row") — and **all 57 are already fixed on the remote default
branches** (docparse#263, email-parse#40/#41, daneel#417); the local checkouts simply predate them.
The two examples are the changes' own expected-fail demonstrations
(`examples/expected_fail/pure_with_effect_row.ail`; `runnable/secrets/ifc_xmod_leak.ail`, whose
manifest entry says `exit_code: 1`): correct reports, which the gate should read from the
manifest's `expected` field rather than flag. Write audit: nothing written in any checkout (one
`.git` mtime change in world came from a concurrent process, not the sweep).

**Run B — remote heads fetched as tarballs into scratch, plus controls:**

| Corpus (ref) | Files | pass both | **newly failing** | fail both | note |
|---|---|---|---|---|---|
| **plant** (positive control) | 3 | 1 | **1** | 1 | exactly the planted `pure func … ! {IO}`; the planted type error is `fail_same`; a planted non-exhaustive match passes both (51f6636 has no exhaustiveness check yet) |
| **motoko fork @083ef43** (before fork#6) | 575 | 316 | **106** | 135 (+18 changed) | **matches the dashboard's hand count, "106 breaks → 0"** — all `EFFECT/pure-with-row` |
| **motoko fork @597a6fb** (after fork#6) | 575 | 422 | **0** | 153 | |
| docparse / email-parse / daneel (origin `main`) | 44 / 49 / 92 | 44 / 49 / 52 | **0** | 0 / 0 / 40 | daneel's 40 are unresolved `daneel_ext_*` deps (no lock entry in the snapshot): unchecked, not passing |
| ailang-packages (origin `main`) | 600 | 588 | **0** | 12 | |
| ailang-parse, world | 132 | 125 | 0 | 7 | |
| **Total** | **2,070** | 1,597 | **107** | 348 | wall 280 s + 213 s |
| null control (baseline vs baseline; plant, motoko, docparse) | 622 | — | **0** differences | — | 127 s |
| package mode, `check --package .` on all 61 ailang-packages dirs | 61 | 57 | 0 | 4 | 56 s + 55 s |

**What this proves.** (1) The mechanism reproduces the #2184 hand measurement exactly, unattended,
in under 9 minutes for both arms. (2) The null control is clean, so `check` is deterministic enough
to diff. (3) **Freshness dominates correctness:** the same candidate reports 59 breaks against
local checkouts and 0 against remote heads. A gate over working trees would have blocked a release
whose downstream was already fixed. (4) Cost is about 0.37 CPU-s per file; a `check` process per
file is affordable. **What it does not prove:** dependency provisioning (daneel's 40 unchecked files),
`ailang test`, Linux CI runners, the registry beyond `ailang-packages`, or the eval corpus (sized
only: 16,162 files ≈ 27 min per arm at the same rate, an extrapolation).

## 5. High-Impact Decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| Diff two arms, never gate on absolute pass/fail | Downstream corpora contain intentionally-broken fixtures and stale probes (§4: 260 of 1,607 files already fail on the baseline); only the *difference* is signal | human | design | high |
| Snapshot corpora from refs via tool-owned mirrors, not working checkouts | §3/§4: working trees are stale and report fixed breaks as new | human | design | high |
| Two tiers: public corpora in PR CI; full sweep (private repos, registry, eval corpus) attended before release | Private repos need cross-repo tokens in CI | human | design | med |
| Acceptance via committed `expected-breaks.yaml` (corpus × code × max count) | Makes "we know, and here is the fix order" reviewable in the breaking PR | human | design | med |
| Group by a normalised error code | `check --json` reports `"code": "ERROR"` for most failures (§6.6) | agent | compile | low |
| No new `ailang` subcommand; a `tools/downstream` program behind a make target | CLI surface is being cut for 1.0 (S5: 89 → 17); this is Internal tooling | human | design | low |

### Design Freeze

- [ ] Diff-only gating with an expected-breaks file (§6.5)
- [ ] Snapshot source = remote default branch via tool-owned mirrors; working-tree mode is debug-only and labelled DIRTY (§6.2)
- [ ] Tier membership: which corpora block a release (Decision 3, §11)
- [ ] Isolation contract (§6.3) is enforced by the runner, not by convention

### Deferred Decisions

- Implementation language (Go under `tools/downstream/` like `tools/eval-elo`, or shell + Python via `uv`) — agent may choose
- Snapshot transport: tool-owned bare mirrors + `git archive`, or the GitHub tarball API used in §4 — agent may choose
- Report layout beyond the required tables — agent may choose
- Parallelism and timeout defaults within the limits in §6.4 — agent may choose
- Baseline-arm caching key and storage location (under `~/.ailang/state/downstream/`) — agent may choose
- Whether `ailang test` runs per file or per package, for corpora that declare hermetic tests — human at review (M5)

## 6. Solution Design

### 6.1 Shape

```
make downstream-check CANDIDATE=./bin/ailang [BASELINE=v0.57.4|<path>] [TIER=ci|release] [CORPORA=a,b] [REF=default|pinned|worktree]
  -> tools/downstream  (reads tools/downstream/corpora.yaml + expected-breaks.yaml)
     1. resolve arms      baseline = latest release asset, sha256-verified, cached by tag
     2. materialise       per corpus: mirror fetch -> `git archive <ref>` into scratch
     3. provision deps    `ailang lock` inside the scratch snapshot, then each lock -> scratch AILANG_PACKAGE_ROOT (offline after this)
     4. discover entries  per corpus rules (6.4)
     5. run both arms     isolated env (6.3), Go-side deadline killing the check's process group, nice 15, -P3
     6. classify + diff   normalised codes; states per entry
     7. gate + report     report.md + report.json + pins.md; exit 0 / 2 (unexpected breaks) / 3 (instrument failure) / 4 (NOT_MEASURED over allowance)
```

**Step 3 as built (2026-11-10, after Mark's ruling to provision rather than allow):** the first
implementation only copied the local registry cache, so 66 entries stayed NOT_MEASURED. Step 3 now
runs the baseline binary's `ailang lock` in a per-run copy of the snapshot for every package dir
with dependencies and no parseable lock (the snapshot itself stays unwritten), honours committed
locks as committed, and fetches registry versions missing from the package root copy with
`ailang install --no-bin` only under the explicit `FETCH_DEPS=1` (a non-404 fetch failure is
exit 3). Where it generates a lock, the candidate's own `ailang lock` runs too and a failure or a
different resolved set is a break (review of #2220: corpora that gitignore their lock re-lock with
the deployed binary in CI). Usage and the measured diagnosis: `tools/downstream/README.md`.

### 6.2 Freshness

Working checkouts are the wrong source: they are on feature branches, behind origin, and dirty
(§3). The tool keeps its **own** bare mirrors under `~/.ailang/state/downstream/mirrors/<repo>`
(`git clone --mirror` once, `git fetch` per run) and materialises each corpus with
`git archive <ref> | tar -x` into the run's scratch dir. The user's checkouts are never touched.
Three ref modes: `default` (remote default branch head — "what breaks if we ship today"; the
release default), `pinned` (a ref listed in the manifest — reproducible CI), and `worktree`
(the local checkout as-is, for attended debugging; every row is labelled DIRTY and the report
prints HEAD, branch and `git status --porcelain` count). Network is needed only for the mirror
fetch and for the baseline asset; a run with `--offline` uses the last fetch and says so.
The §4 prototype took the simpler route — `gh api repos/<r>/tarball/<ref>`, which is GitHub's
`git archive` — and needed no local git state at all; it is a valid M2 implementation if caching
by SHA proves cheap enough (deferred, §5).

### 6.3 Isolation contract (enforced by the runner)

Measured traps, each of which silently corrupts a comparison:

| Trap | Evidence | Rule |
|---|---|---|
| Compile cache defaults to `<project>/.ailang/cache` — **inside the downstream checkout** | `internal/config/paths.go` (`AILANG_CACHE_DIR` doc); email-parse and stapledons-godot already hold `.ailang/` dirs | per-arm `AILANG_CACHE_DIR` in scratch |
| Stdlib resolution prefers `./std` in the cwd, then `<binary>/../std` | `AILANG_STDLIB_PATH` doc in `paths.go`; the candidate given to this design sat next to a stray `scratchpad/std` | copy each binary into its own empty dir; never run with cwd = the ailang repo; scrub all `AILANG_*` from the env |
| Missing dependencies are downloaded and written to `~/.ailang/cache/registry` | `internal/pkg/resolver.go` ("Not cached — download from registry") | set `AILANG_PACKAGE_ROOT` (read-only, "never written or created"; a missing dep is a loud error, not a download) |
| Other `~/.ailang` writes (state stores, traces) | `internal/pkg/authority.go` `UserDir` uses `$HOME` | per-arm scratch `HOME`, `AILANG_STATE_DIR`, `AILANG_TRACE=off` |
| Module paths resolve against the cwd | `check --json` from the wrong cwd gives MOD010 on every file | entry discovery computes the root (6.4) |
| macOS has no `timeout`; a missing one exits 127 in both arms and reports "0 flips" | corpus-sweep memory, 2026-11-03 | a Go-side deadline that SIGKILLs the check's process group (the first build used `perl -e 'alarm N; exec …'`, which never fires on a Go binary: Go ignores SIGALRM — PR #2208 review), plus a planted positive control every run |

The report ends with a **write audit**: files under each materialised corpus *and* each named
local checkout newer than the run's start stamp. Non-empty for a local checkout = exit 3.

### 6.4 Entry discovery and commands

`ailang check` checks only its first file argument, so the unit is one process per entry
(§4: ~0.37 CPU-s per file, ~0.1–0.14 s wall per file at `-P3`). Per file, the root is inferred from the `module` line: if the file path ends
with the module path, cwd = the prefix and the argument is `<module>.ail`; otherwise cwd = the
nearest ancestor holding `ailang.toml`. Per package (every `ailang.toml`), `ailang check --package
--json .` from the package dir — the same command the registry validator runs
(`cmd/registry-validator/bounded.go`). Both modes are on by default; each corpus can narrow them.

`tools/downstream/corpora.yaml` (one entry per corpus):

```yaml
- name: motoko-fork
  repo: sunholo-voight-kampff/motoko_agent      # public
  ref: { default: sunholo/main-dst-20261027, pinned: 597a6fbf9 }
  include: ["src/**/*.ail", "packages/**/*.ail", "scripts/**/*.ail"]
  exclude: [".agent/**", "**/fixtures/reject_*.ail"]   # evidence probes, intended failures
  modes: [files, package]
  test: { cmd: "make check_core", hermetic: false }      # informational; not run by the gate
  pin_files: ["scripts/install-prerequisites.sh:^AILANG_REF=", "ailang.toml:^ailang ="]
  owner: motoko
  tier: { ci: true, release: blocking }
```

Repos with no `include` default to every `git ls-files '*.ail'` in the snapshot. The eval corpus is
a generated corpus: the `code` field of `eval_results/**/*.json` rows with `lang=ailang` and
`compile_ok`, de-duplicated by sha256, each written as `<sha>.ail` into scratch and checked with
`--relax-modules` (models rarely write matching module paths).

### 6.5 Diff, acceptance and release order

Each entry gets one state: `pass_both`, **`NEWLY_FAILING`**, `newly_passing`, `fail_same`,
**`fail_changed`** (both fail, different code set), `timeout_*`. The gate counts `NEWLY_FAILING`
and `fail_changed` in blocking corpora, grouped by (corpus, normalised code), and subtracts
`tools/downstream/expected-breaks.yaml`, which the breaking PR edits:

```yaml
- change: "#2184 pure implies empty effect row"
  code: EFFECT/pure-with-row    # the normaliser's bucket for this break in §4
  corpus: motoko-fork
  max: 106
  fix: "motoko fork#6"          # PR/issue that removes the breaks
  republish: []                 # registry packages that must be re-published first
  expires: v0.58.1              # entry is reported stale after this release
```

Exit 2 if any bucket is unacknowledged or over its `max`. *Implemented (PR #2208 review):* an entry
whose **baseline** result is an infrastructure outcome (unresolved dependencies, timeout, crash) is
`NOT_MEASURED`, not `fail_same`; a blocking corpus tolerates none unless `corpora.yaml` carries a
reviewed `allow_not_measured: {max, reason}`, and exceeding it exits 4. Same code with a different
error is `fail_diag_changed` (reported, not gated). Full state and exit-code tables:
`tools/downstream/README.md`. `newly_passing` is reported, never gated
(a fixture that was meant to fail and now passes deserves a look). For in-repo examples the
expectation already exists: an entry whose `examples/manifest.json` `expected.exit_code` is non-zero
and that fails with its `error_pattern` is `expected_fail`, not a break (`exit_code` alone is a
run-time exit code — `exit_code.ail` exits 42 and type-checks — so it excuses nothing) (§4 run A's two examples). The **release order** is derived:
for each acknowledged bucket, the `republish` packages come first (ordered by the lock graph read
from every corpus's `ailang.lock`), then the apps that depend on them, then pin moves, then the
compiler release — the dashboard's hand-written "#2184 release order", generated, with each `fix`
PR's merge state looked up via `gh` (read-only).

### 6.6 Error-code normalisation (and one AILANG-fix-lane item)

`check --json` emits `"code": "ERROR"` for every error that is not an `ailerrors.Report` —
`cmd/ailang/check_output.go:129`, the directory path at `check.go:682`, and `check_package.go:143`.
Type errors, MOD010 and effect failures all arrive that way; the real code, when there is one, is
inside the message (`PAR_UNEXPECTED_TOKEN`, `TC_ARITY_001`, `IMP012_UNSUPPORTED_NAMESPACE`,
`MOD010`, `LDR001`). The normaliser tries, in order: the structured code; an embedded code;
a family keyed on stable message phrases (`TYPE_ERROR/unify|unbound|other`,
`EFFECT/pure-with-row|undeclared|ceiling`, `IFC/flow-violation`, `DEP/unresolved`); else
`UNCODED:<fingerprint>` with module names stripped (a first prototype pass that kept them split
one #2184 cause into five buckets). Telling detail: #2184's own diagnostic carries no code at all. The report prints the UNCODED share as an instrument-quality number.
**Routing:** giving every `check --json` diagnostic its real code is an AILANG fix (A11) and should
land in v0.58.x, because a code taxonomy is itself a Stable surface after 1.0. The gate does not
wait for it.

### 6.7 Pin report and the 1.0 floor

Every run writes `pins.md`: per corpus, each `pin_files` hit with file:line, the declared floors,
what its CI actually installs, local HEAD/branch/dirty vs remote head, and whether the pin is a
release, `latest`, `dev`, or PATH. **Proposed floor for 1.0:** every blocking corpus pins the final
v0.58.x (release asset + sha256) and its own CI is green on it before v1.0.0-rc.1 is tagged; world's
v0.41.0 fence is the documented exception.

### 6.8 The reverse direction

ailang-parse builds `ailang` dev HEAD as a *required* CI step and also compiles against its declared
floor. The floor job is the pattern to spread: it keeps `ailang = ">=X"` honest. Required CI on
dev HEAD is the hazard: a red dev turns their PRs red for reasons that are not theirs, builds are not
reproducible, and the Dockerfile ships something else (`v0.24.0` ref). **Recommendation:** each
downstream runs required CI on its pinned release plus the floor job, and a scheduled, non-required
`ailang@dev` job that calls `tools/downstream` from this repo with `CORPORA=<self> REF=worktree`
— the same instrument, pointed the other way. That is a template PR per repo, not work in this doc.

### 6.9 Tiers

| Tier | When | Corpora | Arms | Expected runtime |
|---|---|---|---|---|
| **ci** | PRs touching `internal/{lexer,parser,types,elaborate,effects,pipeline,module,loader}`, `std/`, `cmd/ailang/check*` | in-repo examples, motoko fork, ailang-packages, ailang-parse, world — public, `ref: pinned`, no token | base = latest release, cand = PR build | ~1,860 files + 61 packages: ~10 min both arms on the studio (estimate from §4); GitHub runners unmeasured |
| **release** | attended, before any tag carrying a BREAKING fragment; required for v1.0.0-rc.1 | all of the above + docparse, email-parse, daneel (private, via the operator's own `gh` auth), registry (latest of every published package), eval corpus | same; baseline cached per release | ~12 min without the eval corpus (measured shape: 2,070 files in 493 s for both arms); eval corpus ~27 min more for the candidate arm only (extrapolated) |

The ci tier starts advisory (comment + artifact) and becomes required after two weeks without a false
positive. It replaces `tools/ci/motoko_smoke.sh`, whose local arm is dead (§1).

## 7. Implementation Plan

| Milestone | Scope | Acceptance criteria (each with a control) |
|---|---|---|
| **M1** runner + isolation (2 d) | arms, isolated env, discovery, per-file + package modes, normaliser, diff, report.md/json; `REF=worktree` only | (a) **planted break**: a scratch corpus with one construct the baseline accepts and the candidate rejects (today: `pure func … ! {IO}`) plus one file broken in both — reports exactly the first as `NEWLY_FAILING`, once, with its code, and the second as `fail_same` (§4 run B did this by hand); (b) **null control**: CANDIDATE = baseline → 0 non-`pass_both`/`fail_same` rows; (c) write audit empty across every local checkout; (d) a deliberately missing `perl` alarm or binary yields exit 3, not "0 flips" |
| **M2** freshness + pins (1.5 d) | mirrors, `git archive` snapshots, `default`/`pinned` modes, `pins.md` | (a) **#2184 replay**: motoko fork at the commit before fork#6 vs `597a6fbf`, v0.57.4 vs a #2184 build — 106 newly failing at the former, 0 at the latter (§4 measured exactly this by hand); (b) pins.md reproduces the §3 pin column |
| **M3** acceptance + release order (1.5 d) | `expected-breaks.yaml`, exit codes, derived order, `release-manager` step | (a) an unacknowledged bucket → exit 2, acknowledged → 0, over-count → 2; (b) a stale entry past `expires` is reported; (c) the generated order for the #2184 replay matches the dashboard's five steps |
| **M4** CI tier (1 d) | workflow job on front-end paths, public corpora at pinned refs, baseline asset cache | (a) a PR that plants a parser break in a throwaway branch turns the job red; (b) a docs-only PR does not run it |
| **M5** breadth (1–2 d) | registry (latest of each published package, offline after fetch), eval corpus, hermetic `ailang test` where declared | (a) eval-corpus run inside the release-tier budget; (b) a registry package pinned to a version known to fail its floor shows as `fail_same`, not a new break |

**Files to create:** `tools/downstream/` (runner + tests), `tools/downstream/corpora.yaml`,
`tools/downstream/expected-breaks.yaml`, a `make/downstream.mk` target, one CI job in
`.github/workflows/ci.yml`, one step in `.claude/skills/release-manager/SKILL.md`, a changelog
fragment. **Retire:** `tools/ci/motoko_smoke.sh` once M4 is required.

## 8. Hooks

- **release-manager:** a new step between "1. Pre-Release Verification" and "1.5 Fold": if any
  unreleased fragment has a `BREAKING` section, `make downstream-check TIER=release
  CANDIDATE=./bin/ailang` must exit 0, and its `report.md` is linked from the release notes.
- **m-v1-release-gates.md:** the downstream gate is one of the gates that doc enumerates; this doc
  owns the instrument, that doc owns where it sits in `release.yml`/the tag pipeline. The CI tier is
  a PR check, not a `release.yml` dependency (it needs the PR build, not the tag build).
- **Soak (review §4.3 step 5):** "downstream CI green on the final v0.58.x" is read from `pins.md`
  plus each repo's own CI status, not asserted.

## 9. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Private repos unreachable from CI | release tier runs attended with the operator's `gh` auth; the report lists any corpus it could not fetch as **NOT CHECKED**, never as passing |
| Stale or dirty checkouts give false breaks | mirrors + `git archive` (6.2); `worktree` mode labels every row |
| Flaky network during dep install | deps provisioned once into a scratch `AILANG_PACKAGE_ROOT`; checks run offline; a missing dep is its own state (`DEP_MISSING`), not a code |
| Baseline failures hide real breaks (a file that already fails cannot newly fail) | report baseline-fail count per corpus as coverage; manifest `exclude` drops known probes; target < 5% baseline-fail in blocking corpora |
| Load on the shared studio | `nice -n 15`, `-P3` (`-P10` pushed load to 155 once); baseline arm cached by binary sha + snapshot sha |
| Timeout asymmetry looks like a break | timeouts are their own state, retried once at 2× before counting |
| Error codes shift between arms, so `fail_changed` is noisy | normaliser output is part of the report; `fail_changed` is gated only in blocking corpora |
| The gate gets bypassed under release pressure | exit 2 is overridable only by editing `expected-breaks.yaml`, which leaves a reviewed diff |

## 10. Non-Goals

- Running downstream apps or their effectful test suites (no `--caps`, no network, no Docker boots;
  the serve-api boot smoke from the 2026-11-03 sweep stays a separate, manual check).
- Fixing downstream code, opening downstream PRs, publishing packages, or moving pins.
- The Go API (world imports `serveapi` via `go.mod`); that is review §3.5's decision.
- Performance regressions (`perf-sweep`) and eval pass rates (`eval-suite`).
- A new `ailang` CLI command.

## 11. Decisions for Mark

1. **Does the release tier block?** *Recommendation:* yes, for any release with a BREAKING
   fragment and for v1.0.0-rc.1; the ci tier is advisory for two weeks, then required.
2. **Snapshot source for the release tier.** *Recommendation:* remote default branch heads via
   tool-owned mirrors; pinned refs in CI. Never local working trees, given §3.
3. **Blocking corpora.** *Recommendation:* blocking = in-repo examples, docparse, ailang-parse,
   email-parse, daneel, motoko fork, ailang-packages, registry; advisory = world (deliberate v0.41.0
   fence), eval corpus, stapledons-godot and the small repos.
4. **Cross-repo tokens in CI for the private repos.** *Recommendation:* no, not before 1.0; the
   attended release tier covers them at no standing secret cost.
5. **ailang-parse on dev HEAD.** *Recommendation:* move its required CI to a pinned release plus its
   floor job, and add a scheduled advisory `ailang@dev` job (6.8).
6. **The 1.0 pin floor.** *Recommendation:* every blocking corpus on the final v0.58.x, sha-pinned,
   CI green, before v1.0.0-rc.1; world exempt by its fence.
7. **Structured codes in `check --json`** (6.6) as a v0.58.x AILANG fix. *Recommendation:* yes —
   after 1.0 the code taxonomy is part of what users script against.

## 12. Effort Estimate (an estimate, not a plan)

M1 2 d · M2 1.5 d · M3 1.5 d · M4 1 d · M5 1–2 d = **7–8 working days**, with M1–M3 (5 days) needed
before the first v0.58.x breaking release. The prototype in §4 is roughly the M1 core minus the
package mode, mirrors and acceptance file.

## 13. Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | +1 | the null control makes run-to-run nondeterminism in `check` visible |
| A5 Bounded Verification | +1 | turns a release's compatibility claim into a bounded, re-runnable measurement |
| A7 Machines First | +1 | JSON report, normalised codes; pushes `check --json` toward real codes |
| A11 Structured Failure | +1 | breaks are grouped by code, not by prose |
| A2–A4, A6, A8–A10, A12 | 0 | tooling only; no language or capability change |

**Net +4.** No hard violations: the tool runs `check` only, grants no capabilities, and writes only to scratch.

## 14. Verification Log and Limits

- **Verified:** every pin in §3 against the file named (local checkout, or `gh api` read of the
  remote default branch where the local branch differs); `.ail` counts by `find`; cache, stdlib,
  package-root and `$HOME` behaviour by reading `internal/config/paths.go`,
  `internal/pkg/resolver.go`, `internal/pkg/authority.go`; `check --json` code fallback by reading
  `cmd/ailang/check_output.go`; the release asset's sha256.
- **Contradicts the review / the brief:** motoko's v0.54.14 is *upstream's* installer pin — our rig
  branch says v0.52.5 and the rig runs PATH; daneel's v0.54.18 is true only on origin `main` (the
  local branch says v0.53.3); world's v0.54.15 is an attended script, not CI, and its CI also takes
  `releases/latest`; ailang-packages has 61 package manifests on origin `main` (45 in the stale local checkout), not 42; "nothing re-checks" is true in substance but
  `tools/ci/motoko_smoke.sh` exists (one file, dead local arm); the installed `~/go/bin/ailang` is
  now v0.57.4-120, not -26; `benchmarks/` has no AILANG reference solutions to sweep.
- **Not verified:** private repos beyond their default-branch tarballs (no other branches, no CI
  history); dependency provisioning by `ailang lock` in a snapshot (daneel's 40 unchecked files); `ailang test` hermeticity anywhere; CI wall time on GitHub runners (§4 numbers are
  the studio under load ~20); the registry's full published list (only the local cache was read).

## References

- [v1-release-readiness-review-2026-11-09.md](../v1-release-readiness-review-2026-11-09.md) §4.1, §4.3, §9
- [v1-mission-dashboard.md](../../v1-mission-dashboard.md) — the #2184 release order and #2185 site list
- [m-v1-release-gates.md](m-v1-release-gates.md) — sibling: where gates sit in the release pipeline
- [m-v1-simplification-program.md](../m-v1-simplification-program.md) — why this is not a CLI command
- `docs/docs/reference/stability.md` — what counts as breaking
- `tools/ci/motoko_smoke.sh`, `make verify-examples`, `cmd/registry-validator` — prior art reused
- [MOTOKO.md](../../../MOTOKO.md) — which motoko checkout is real
