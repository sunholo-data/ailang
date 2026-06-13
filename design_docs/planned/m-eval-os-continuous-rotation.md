# M-EVAL-OS-CONTINUOUS-ROTATION: Background, incremental OS cross-language/harness evals between releases

**Status**: Planned (proposal)
**Target**: v0.26.0
**Priority**: P1 (fills the OS/Local leaderboard — cross-language + harness + longitudinal — continuously, at zero cloud cost)
**Dependencies**: `eval-publish --os-json` (done — emits the dashboard JSON), `eval-suite --benchmarks-by-confidence` + `ratings.db` (done — ELO selective rerun), the local rig (`dev.ollama.serve`, `dev.ailang.rig-watchdog`, `nightly-eval.sh`, `nightly-lang-eval.sh`), [[ailang-eval-language-split]], M-EVAL-STREAM-HEALTH-RETRY (idle/stream-death retry).

## Goal

Use the now-stable local rig to keep the **OS/Local leaderboard** (cross-language incl. JS/Go,
cross-harness, longitudinal) fresh **between releases**, running **incrementally almost all the
time in the background** — without ever colliding with the other scheduled local jobs (the 04:44
nightly, `nightly-lang-eval`, ollama model loads).

## Constraints (from the rig)

- **Single-GPU, bandwidth-bound**: only one model loaded at a time; `NUM_PARALLEL>1` thrashes (confirmed earlier). So the filler runs **strictly serial**.
- **Mutual exclusion**: the nightly + lang-eval + any ollama reload must never overlap a filler chunk (a mid-stream model reload silently killed a run on 2026-06-07 — the original finding).
- **Priority**: `nightly-eval` > `nightly-lang-eval` > continuous filler. The filler always yields.

## Design

### 1. One rig lock for all local jobs
A shared advisory lock `~/.ailang/state/rig.lock` (`flock`). **Every** rig job wraps its work in it:
`nightly-eval.sh`, `nightly-lang-eval.sh`, and the new filler. The two nightlies take the lock
blocking (they're scheduled, they win); the filler takes it **non-blocking** and exits immediately
if held. This is the core "don't run at the same time" guarantee — add the `flock` wrapper to the
two existing scripts (small change) so the lock is authoritative.

### 2. The filler: `tools/launchd/os-rotation-filler.sh` (new)
A launchd job, `StartInterval` ~2700s (45 min), that on each fire:
1. `flock -n ~/.ailang/state/rig.lock` — exit if a nightly/lang-eval holds it.
2. **Blackout check** — skip if within a configured window (default `04:00–07:00`, covering the
   nightly + lang-eval + their model reloads), or if a nightly is due within `lead_minutes`.
3. **Health check** — ollama reachable (reuse rig-watchdog's check); else exit.
4. **One bounded chunk** (`--timeout`-boxed, ~25 min): run `eval-suite` **serial** (`--parallel 1`)
   over a small set of cells chosen by **`--benchmarks-by-confidence ratings.db --max-benchmarks K`**
   for the OS models × harnesses, `--langs ailang,python,javascript,go`. ELO selection means each
   chunk runs the cells that most move belief, so coverage converges fast instead of brute-forcing
   the full matrix.
5. **Append** results to the current rotation dir `eval_results/rotation/<cycle>/`.
6. **Publish (batched)**: when a coverage pass completes (or every N chunks), run
   `ailang eval-publish <cycle-tag> --rotation … --os-json docs/static/benchmarks/os/latest.json`
   and commit+push the JSON (bot creds) → docs-deploy fills the OS/Local page. Batching avoids a
   commit per 25-min chunk.

### 3. Coverage + ratings state
- `ratings.db` (ELO) drives selection and is updated per trial (already supported).
- A small coverage cursor (`~/.ailang/state/os-rotation-cursor.json`) tracks which
  (model, benchmark, lang, harness) cells are done **at the current dev SHA**; a new release/SHA
  resets the cursor so the matrix re-fills against the new compiler.

### 4. Reliability
The filler uses the idle/stream-death retry (M-EVAL-STREAM-HEALTH-RETRY): an ollama reload mid-chunk
is detected in seconds and the cell retried, instead of burning the generation cap. `cost_killed`
/ infra categories are excluded from ratings updates (same policy as the nightly detector).

## Scheduling summary

| Job | Schedule | Lock | Priority |
|---|---|---|---|
| `dev.ollama.serve` | always (keepalive) | — | infra |
| `nightly-eval` | 04:44 calendar | blocking | highest |
| `nightly-lang-eval` | (its calendar) | blocking | high |
| **os-rotation-filler** | every ~45 min | **non-blocking (yields)** | background |
| `rig-watchdog` | StartInterval poll | — | infra |

Net effect: the rig runs the cloud-free cross-language/harness sweep in ~25-min background slices
whenever it's idle, never stepping on the nightlies, and the OS/Local page fills in over hours/days
and refreshes each release.

## Phases
1. **Lock**: add the shared `flock` wrapper to `nightly-eval.sh` + `nightly-lang-eval.sh`. (Small, safe.)
2. **Filler script + plist**: `os-rotation-filler.sh` + `dev.ailang.os-rotation-filler.plist` (blackout + non-blocking lock + bounded ELO-selected chunk + batched publish).
3. **Coverage cursor** + ratings wiring.
4. **First fill**: let it run a few cycles; verify the OS/Local page populates with JS/Go + harness.

## Open questions
- Blackout window exact bounds (need the `nightly-lang-eval` schedule — it's in its plist).
- Which OS models × harnesses are in scope (opencode-* set; add `pi`? `motoko` is still hangy).
- Publish cadence: per coverage-pass vs daily (avoid commit noise vs freshness).

## Axiom Compliance
| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | +1 | Seeded, serial, ratings-driven selection; reproducible per SHA. |
| A6 Safe Concurrency | +2 | The one-rig-lock + non-blocking-yield is the core safety property (no overlap, no thrash). |
| A9 Cost Visibility | +2 | Cross-language/harness longitudinal data at zero cloud cost, using otherwise-idle rig time. |
| A11 Structured Failure | +1 | Stream-death retry + infra-category exclusion keep flakes out of the ratings. |

**Hard violation check:** none.
