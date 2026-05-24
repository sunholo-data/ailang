# M-EVAL-LOCAL-OLLAMA: Production-Grade Local Ollama Eval Rig

**Status**: Planned
**Target**: v0.24.0
**Priority**: P1 (Medium — quality-of-life + reliability for the eval loop the team uses daily)
**Estimated**: 3-4 days (sprint of ~5 small PRs)
**Dependencies**: M-AI-OLLAMA (v0.7.0, implemented). Builds on the operational learnings from the 2026-05-22..24 weekend (commits `5cf6287b..1b7e06d4`).

## Problem Statement

The weekend autonomous run validated the *strategic* loop ("error-message improvements unblock benchmarks") but exposed a *long tail of operational paper cuts* in the local Ollama rig:

- **`-agent-parallel` was dead decoration** for unknown how-many weeks before discovery — same bug had been "fixed" once before and silently re-introduced. The harness advertised a knob with no semantics, and `eval-smoke` couldn't pass a missing flag. ([Commit 98c0c408](commits/98c0c408), [Commit 07cdcbbd](commits/07cdcbbd))
- **`make install` did not symlink `ailang` into opencode's child-shell PATH** until [commit 8f0f415d](commits/8f0f415d). Pre-fix, every fresh rig setup spent 5-30 min in `find /` and `ls -R /` thrashing before the model gave up.
- **opencode silently drops per-model options unless `options.name` is set** (sst/opencode#971). No warning, no error — just the agent runs with default temperature on a model known to collapse at default temperature. Verified by determinism check, but only after spending hours assuming the Modelfile params propagated.
- **ollama's OpenAI-compat surface silently drops `top_k`, `repeat_penalty`, `min_p`, `num_predict`** even when sent as JSON. The fix (Modelfile + `ollama create`) is real, but it lives in shell history and the runbook, not in code or CI.
- **MoE / single-GPU constraint discovered empirically** after a 15-min TTFT timeout. `OLLAMA_NUM_PARALLEL > 1` looks like free parallelism but oversubscribes the single GPU's serial inference loop. No code currently warns about this.
- **Pathological bash exploration** — pre-denylist, model would spawn `find / -name ailang`, `ls -R /`, `grep -r /usr` — chewing wall-clock for tens of minutes per session. The denylist now lives in `~/.config/opencode/opencode.jsonc` (per-user config), not in repo, so every fresh rig setup is one un-onboarded user away from re-hitting it.
- **The warmup step (MCP initialize + opencode hello)** is a manual script (`warmup_rig.sh`), not invoked automatically by `make eval-*` targets. First benchmark of a cold rotation pays a 60-90s prefill that subsequent trials don't.

Each of these is individually small. Cumulatively they are 6-8 hours of weekend debugging that should never need to happen again.

**Current State** (after weekend's fixes):
- Pass rate ceiling lifted from 80% to repeatable 82-100% on specific benchmarks via compiler error work
- 14 hours of autonomous rotations completed, 5 numbered iterations, 7 commits
- Runbook at `.claude/skills/local-ollama-eval/resources/rig_operations_runbook.md` (~400 lines) captures lessons
- `~/.config/opencode/opencode.jsonc` is correct on this machine but is NOT versioned and NOT validated by any check

**Impact:**
- Anyone setting up a fresh rig (CI/secondary developer machine/contributor) will silently re-hit ≥3 of these traps. The current safeguard is "Mark or the agent reads the runbook" — not durable.
- The eval loop is now central to language development (M-AILANG-ERROR-QUALITY drives error message work directly from eval signal). Every paper cut is a tax on the iteration cycle.

## Goals

**Primary Goal:** Make the local Ollama eval rig reliable enough that a new model can be onboarded in <30 min with no out-of-band knowledge, and so that the four configuration surfaces (Modelfile, opencode.jsonc, models.yml, ollama plist) cannot silently drift out of sync.

**Success Metrics:**
- New model onboarding follows a single `make rig-onboard MODEL=qwen3-coder:30b` style command that drives the whole TL;DR checklist
- `make rig-precheck` (or equivalent) fails loudly if any of: `~/.local/bin/ailang` symlink missing, `options.name` missing from opencode.jsonc, `OLLAMA_NUM_PARALLEL != 1`, MCP server unreachable
- Eval-suite startup banner reports the active sampling params (so a silent option drop is visible at trial start, not 3 hours later)
- Dead flags are caught by a unit test that asserts every flag advertised in `--help` has a non-trivial handler
- The repo-tracked example `~/.config/opencode/opencode.jsonc` (sanitized) is verified by CI to match the running machine via a sample-comparison test

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Where canonical opencode.jsonc lives | If it lives in `~/.config/...` only, every new machine re-hits the traps. If versioned in repo, must keep it sanitized (no API keys). | human | design | high |
| Whether `make rig-precheck` is opt-in or wired into `eval-smoke`/`eval-suite` targets | Wired-in catches drift always; opt-in respects users who know what they're doing | human | design | med |
| Should `make rig-onboard` be a make target, a bash script, or an `ailang` CLI subcommand | CLI subcommand integrates with `ailang doctor`-style ecosystem; bash is simplest | human | design | med |
| Whether to detect MoE/dense and auto-set `OLLAMA_NUM_PARALLEL` or just warn | Auto-set is invasive (modifies system launchd plist); warn is safe | human | design | low |
| Sanitization strategy for the tracked opencode.jsonc | Need to strip API keys, prompt-injection-test the example, decide what is "real default" | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Location of canonical opencode.jsonc**: in-repo (`tools/opencode/opencode.example.jsonc`) versus dotfiles-style external (`scripts/install-rig.sh` writes it). Recommendation: in-repo, sanitized, with a `make rig-install-config` target that diffs/copies.
- [ ] **`make rig-precheck` invocation**: pre-step of `eval-smoke`/`eval-suite` (opinionated) vs separate target (consenting). Recommendation: opinionated, with `SKIP_PRECHECK=1` escape hatch.
- [ ] **`make rig-onboard` shape**: ailang CLI subcommand for tighter integration, or bash script for fewer code-paths. Recommendation: start as `tools/rig/onboard.sh` bash, promote to CLI in v0.25 if it earns it.

## Solution Design

### Overview

Five buckets of work, each a small PR. The runbook stays as the human-readable knowledge artifact; this design doc drives the *code* that prevents the runbook from being re-discovered:

1. **Configuration in repo** — sanitized `opencode.example.jsonc`, copyable Modelfile templates, with versioning of what they should look like
2. **Pre-flight checks** — `make rig-precheck` + invocation from `eval-smoke`/`eval-suite`
3. **Onboarding script** — `tools/rig/onboard.sh MODEL_TAG` driving the TL;DR checklist
4. **Trial-time visibility** — eval-suite banner prints active sampling params (so silently-dropped options are visible)
5. **Dead-flag prevention** — unit test that every `--help` flag has a non-trivial handler

### Architecture

```
Repo                            Machine                          Per-trial
────                            ───────                          ─────────

tools/opencode/                 ~/.config/opencode/              eval-suite startup:
  opencode.example.jsonc  ─┐    opencode.jsonc                     ↓ precheck
                           │     ↑                                 ↓ banner with params
                           │     │ make rig-install-config         ↓ warmup_rig invoked
                           │     │ (diffs then copies)             ↓ run trials
tools/ollama/              │     │
  gemma4-26b-ailang        │     ollama create + ollama serve
  qwen3-coder-30b-ailang   │
  <new>-ailang.modelfile.template ─→ tools/rig/onboard.sh

tools/rig/
  onboard.sh MODEL=...    ──┐
                            │ Runs: pull → create variant → write opencode block → models.yml entry → symlink → warmup
                            │ Each step has a verify-and-skip-if-already-done check
                            └ Output: ready-for-eval-smoke summary

cmd/ailang/cmd/rig.go (or tools/rig/precheck.sh)
  rig-precheck
   ├ ailang symlink exists (~/.local/bin/ailang → ~/go/bin/ailang)
   ├ OLLAMA_NUM_PARALLEL==1 in launchd plist
   ├ opencode.jsonc has options.name="Ollama"
   ├ opencode.jsonc has bash denylist with trailing wildcards
   ├ MCP server reachable (mcp.ailang.sunholo.com /health)
   ├ Target model variant exists in ollama
   └ Modelfile sampling params present in variant
```

The repo gains 3 new touchpoints (`tools/opencode/`, `tools/rig/`, banner enhancement). No core compiler/parser changes. Each PR is independently revertable.

### Implementation Plan

**Phase 1: Track configuration in repo** (~3 hours)
- [ ] Create `tools/opencode/opencode.example.jsonc` — sanitized version of working machine config (strip API keys, comment-document each block)
- [ ] Create `tools/ollama/<model>-ailang.modelfile.template` — parameterized template with the sampling tune baked in
- [ ] Add `make rig-install-config` — diffs `~/.config/opencode/opencode.jsonc` against repo example, offers to copy
- [ ] Add CI check `make verify-opencode-example` — lints the example for required fields (`options.name`, bash denylist, MCP wiring)

**Phase 2: Pre-flight checks** (~3 hours)
- [ ] Write `tools/rig/precheck.sh` (or `cmd/ailang/cmd/rig.go` if we go CLI route — decide in design freeze)
- [ ] Checks: symlink, NUM_PARALLEL, options.name, denylist wildcards, MCP reach, model variant exists, Modelfile params present
- [ ] Wire `rig-precheck` into `eval-smoke` and `eval-suite` make targets as a `pre:` dependency
- [ ] Add `SKIP_PRECHECK=1` escape hatch
- [ ] Each failed check prints one-line remediation pointing at the runbook section

**Phase 3: Onboarding driver** (~4 hours)
- [ ] Write `tools/rig/onboard.sh MODEL=qwen3-coder:30b` — drives TL;DR checklist
- [ ] Each of the 9 steps in the runbook becomes a function with verify-and-skip
- [ ] Idempotent: re-running on already-onboarded model is a no-op that prints "ready"
- [ ] On failure: prints exactly which step failed and which env/config to inspect
- [ ] `make rig-onboard MODEL=...` thin wrapper

**Phase 4: Trial-time visibility** (~2 hours)
- [ ] Modify `cmd/ailang/eval_suite.go` startup banner to fetch and print active sampling params for each model
- [ ] Query opencode model info or ollama API at suite start
- [ ] Compare against expected (from models.yml or opencode.jsonc) and warn on mismatch
- [ ] Useful side benefit: caught-on-startup if `options.name` was lost

**Phase 5: Dead-flag prevention** (~2 hours)
- [ ] Unit test `cmd/ailang/eval_suite_test.go::TestNoFlagsAreUnimplemented`
- [ ] Walk all flags exposed by `eval-suite --help`
- [ ] For each flag, set a sentinel non-default value, run a 1-trial smoke, verify the sentinel appears in the run record OR the flag changed observable behavior
- [ ] Flag must either visibly affect behavior or be deleted
- [ ] Catches `-agent-parallel`-style decorations at PR time

### Files to Modify/Create

**New files:**
- `tools/opencode/opencode.example.jsonc` — sanitized canonical config, ~200 lines
- `tools/ollama/gemma4-26b-ailang.modelfile.template` — already exists; promote/document
- `tools/rig/precheck.sh` — pre-flight checks, ~120 lines
- `tools/rig/onboard.sh` — model onboarding driver, ~200 lines
- `make/rig.mk` — `rig-precheck`, `rig-onboard`, `rig-install-config` targets, ~30 lines

**Modified files:**
- `make/eval.mk` — wire `rig-precheck` into `eval-smoke`/`eval-suite` (~10 lines added)
- `cmd/ailang/eval_suite.go` — sampling-params startup banner, ~40 lines added
- `cmd/ailang/eval_suite_test.go` — `TestNoFlagsAreUnimplemented`, ~80 lines added
- `Makefile` — include `make/rig.mk`, ~1 line
- `.claude/skills/local-ollama-eval/resources/rig_operations_runbook.md` — point at the new tooling at top, keep the long-form lessons (~20 lines edit)
- `Makefile.help` (if exists) or `make help` entries for new targets (~5 lines)

## Examples

### Example 1: Onboarding qwen3-coder:30b on a fresh machine

**Before** (current state — what the weekend taught us):
```bash
# Open the runbook, read TL;DR section, copy-paste commands one at a time
# Hit the symlink trap → fix → continue
# Hit the options.name trap → discover via behavior, fix opencode.jsonc → continue
# Forget to set OLLAMA_NUM_PARALLEL=1 → discover via 15-min TTFT timeout → fix plist, restart, continue
# Total elapsed: 2-4 hours of new-rig pain
```

**After** (this design):
```bash
# Single command
make rig-onboard MODEL=qwen3-coder:30b

# Output:
# [1/9] Pulling qwen3-coder:30b... ✓ (already present)
# [2/9] Creating ailang variant... ✓ (created qwen3-coder:30b-ailang)
# [3/9] Updating opencode.jsonc... ✓ (added provider.ollama.models entry)
# [4/9] Updating models.yml... ✓ (added opencode-qwen3-coder-30b-ailang)
# [5/9] Ensuring ~/.local/bin/ailang symlink... ✓
# [6/9] Verifying OLLAMA_NUM_PARALLEL=1... ✓
# [7/9] MCP server reachable... ✓
# [8/9] Sanity check (1 fizzbuzz trial)... ✓ (12s)
# [9/9] Warmup... ✓
#
# Rig ready. To run smoke tier:
#   make eval-smoke MODELS=opencode-qwen3-coder-30b-ailang
```

### Example 2: Catching a silently-dropped option

**Before**:
```bash
# Edit opencode.jsonc, accidentally delete `options.name` line
make eval-smoke MODELS=opencode-gemma4-26b-ailang
# Trial runs at temperature 1.0 (default), gemma4 collapses
# Discovered 30 minutes later when reading individual trial outputs
```

**After**:
```bash
make eval-smoke MODELS=opencode-gemma4-26b-ailang
# [rig-precheck] ✗ ~/.config/opencode/opencode.jsonc missing required field: provider.ollama.options.name
#   Fix: see .claude/skills/local-ollama-eval/resources/rig_operations_runbook.md#2-opencode-silently-drops-model-options
#   Or run: make rig-install-config
# Make target exits non-zero; user fixes; reruns; no wasted trials
```

### Example 3: Sampling params visible at trial start

**Before**: banner shows `Dispatch parallelism: 1`. No info about sampling.

**After**:
```
Eval Suite — 2026-05-25 14:32
  Models:
    opencode-gemma4-26b-ailang
      sampling: temp=0.5 max_tokens=4096 repeat_penalty=1.1 num_predict=4096
      modelfile sha: 7121486771cb...
      opencode options: name=Ollama freq_penalty=0.3
      ⚠ options.name confirmed (sst/opencode#971 guard)
  Benchmarks: 17 (smoke tier)
  Trials per benchmark: 3
  Dispatch parallelism: 1
```

## Success Criteria

- [ ] `tools/opencode/opencode.example.jsonc` exists and is verified by CI to contain `options.name`, bash denylist with wildcards, MCP wiring
- [ ] `make rig-precheck` exits non-zero on a deliberately-broken config (test: rename symlink, run, expect failure)
- [ ] `make eval-smoke` calls `rig-precheck` automatically unless `SKIP_PRECHECK=1`
- [ ] `make rig-onboard MODEL=qwen3-coder:30b` produces a runnable rig from a known-clean machine in <10 min wall time, idempotent on re-run
- [ ] Eval-suite startup banner shows sampling params, sha of Modelfile, and `options.name` confirmation per model
- [ ] `TestNoFlagsAreUnimplemented` passes; if a flag is added with no implementation, this test fails
- [ ] All tests passing
- [ ] CHANGELOG updated under v0.24.0 section
- [ ] Runbook updated to point at the new tooling at top
- [ ] One end-to-end demo: wipe `~/.config/opencode/opencode.jsonc`, run `make rig-install-config && make rig-onboard MODEL=gemma4:26b && make eval-smoke ...`, confirm green from cold

## Testing Strategy

**Unit tests:**
- `TestNoFlagsAreUnimplemented` — walks `eval-suite --help` flags, asserts each one observably changes behavior
- `TestOpencodeExampleHasRequiredFields` — parses `tools/opencode/opencode.example.jsonc`, asserts `provider.ollama.options.name` present, bash denylist contains trailing wildcards
- `TestSamplingParamsBanner` — golden-file test for the new banner output

**Integration tests:**
- `TestRigPrecheckCatchesMissingSymlink` — uses tempdir + faked HOME, removes symlink, asserts precheck exits non-zero with the right error
- `TestRigPrecheckCatchesMissingOptionsName` — writes a broken jsonc to tempdir, asserts precheck catches it

**Manual testing:**
- End-to-end: blank machine setup via `rig-install-config` + `rig-onboard` + `eval-smoke` should pass
- Drift detection: deliberately break each of 5 prerequisites in turn, confirm each precheck remediation message points at the right runbook section

## Deferred Decisions

The following are intentionally left open for the implementer:

- **CI for the example config**: whether the in-repo `opencode.example.jsonc` is just lint-checked or also `opencode mcp list`-roundtripped in CI. Agent may choose — the lint check is the floor, roundtrip is nice-to-have.
- **Whether to auto-set `OLLAMA_NUM_PARALLEL=1`** if precheck finds it unset — agent may warn or auto-fix; default to warn unless the launchd plist is detected as "vanilla" (no user customization).
- **Shape of the per-model Modelfile template variables** — agent may template via envsubst, sed, or a small Go templating helper. Pick the simplest.
- **Whether `ailang-rig` becomes a long-term subcommand of the `ailang` CLI** in v0.25 — out of scope for this milestone, but the bash scripts here should be structured so that promotion is mechanical.

## Non-Goals

**Not attempted in this feature:**
- **New CI to spin up an ollama server on GitHub Actions runners** — out of scope (GPU runners absent, would need a self-hosted runner; that's a separate project)
- **Coverage of cloud-API rig paths (OpenRouter, OpenAI, Anthropic via opencode)** — they don't have the same drift surface; their failures are typically observable at the API response level, not weeks later
- **Refactoring the underlying `internal/ai/ollama/` provider package** — that's v0.7.0 work and already implemented; this milestone is purely the rig/operations layer
- **Solving model-capability ceiling issues** (e.g., gemma4:26b on fizzbuzz) — that's a model + prompt + compiler-error problem, tracked elsewhere

## Timeline

**Day 1** (~4 hours):
- Phase 1 (in-repo config)
- Phase 4 (banner enhancement — quick win, visible at every trial)

**Day 2** (~3 hours):
- Phase 2 (precheck script + wire into eval-smoke)

**Day 3** (~4 hours):
- Phase 3 (onboarding driver)

**Day 4** (~2 hours):
- Phase 5 (dead-flag test) + integration testing + docs

**Total: ~13 hours over 3-4 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Sanitizing `opencode.jsonc` accidentally leaks an API key in repo history | High | Pre-commit hook scans for known key prefixes (`sk-`, `xai-`, etc.); CI lint repeats; first commit reviewed manually before push |
| `rig-precheck` becomes annoying friction for fast iteration | Med | `SKIP_PRECHECK=1` escape hatch; pre-check should be <1s to run |
| Auto-fixes (e.g. symlink creation, plist edits) silently change user environment | Med | Default to warn-only; explicit `--fix` flag required for mutations |
| Per-model Modelfile template diverges from per-model needs (different models want different sampling) | Low | Template is a starting point; per-model overrides in `tools/ollama/<model>.modelfile` are kept as the source of truth, template generates only when no file exists |
| Tracked example config drifts from real machine config | Med | CI test diffs them; if user accepts drift, they update example or the test fails next CI run |
| Onboarding script too rigid for "advanced" users with custom setups | Low | Onboarding is opt-in (not required to use rig); advanced users skip it; the precheck still validates their custom setup |

## Related Documents

**Implemented (the foundation this builds on):**
- [design_docs/implemented/v0_7_0/m-eval-ollama-local-models.md](../../implemented/v0_7_0/m-eval-ollama-local-models.md) — Unified Ollama provider in `internal/ai/`. Touched the provider layer; this milestone covers the operational/rig layer above it.
- [design_docs/implemented/v0_23_x/weekend-iteration-report-2026-05-23.md](../../implemented/v0_23_x/weekend-iteration-report-2026-05-23.md) — The longitudinal weekend report this design is the operational follow-up to.

**Planned (companion work):**
- [m-eval-rating-efficiency.md](m-eval-rating-efficiency.md) — ELO ratings + selective reruns + tier saturation. Complementary: that doc makes evals *informative*; this doc makes them *operationally clean*.
- [m-eval-metrics-taxonomy.md](m-eval-metrics-taxonomy.md) — 25-metric vocabulary. The new banner output should emit these tagged correctly.
- [m-eval-finetuning-data-pipeline.md](m-eval-finetuning-data-pipeline.md) — Uses the eval rotations as a fine-tuning corpus source; depends on the rig being reproducible (this doc enforces that).
- [m-ailang-error-quality-for-llm-iteration.md](m-ailang-error-quality-for-llm-iteration.md) — Error-quality work driven by eval signal. Depends on the eval signal being trustworthy (this doc protects that).

**Operational reference:**
- `.claude/skills/local-ollama-eval/resources/rig_operations_runbook.md` — The 400+ line human-readable runbook. This design doc proposes the *code* that operationalizes its lessons.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Locks down sampling param visibility at trial start — silently-dropped options were the source of cross-run non-determinism |
| A2: Replayability | +1 | Modelfile sha + recorded sampling params in trial output enables exact replay |
| A3: Effect Legibility | 0 | No change to AILANG-level effect surface |
| A4: Explicit Authority | +1 | Bash denylist promoted to in-repo example; can't be silently widened without a PR |
| A5: Bounded Verification | 0 | No change to type/verification surface |
| A6: Safe Concurrency | +1 | Codifies the NUM_PARALLEL=1 rule with auto-detection |
| A7: Machines First | +1 | Banner output is machine-parseable; precheck output is structured |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Sampling-param banner makes per-trial cost-determining settings visible at run start |
| A10: Composability | 0 | Tooling sits at integration layer; doesn't change language composition |
| A11: Structured Failure | +1 | Precheck failures point at exact runbook section; structured remediation rather than "something's wrong" |
| A12: System Boundary | +1 | Makes the rig boundary surfaces (4 config files) explicit; precheck catches drift across them |

**Net Score: +7** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — actually *removes* a source (silent option drop)
- [x] A3 (Effects): No hidden side effects in AILANG code
- [x] A4 (Authority): No ambient access granted — explicit `--fix` flag required for environment mutations
- [x] A7 (Machines First): All output structured/parseable

### Conflict Surface

**Not applicable for this milestone.** This work touches `cmd/ailang/eval_suite.go` (banner addition only, no semantic change), `make/eval.mk`, and new `tools/` files. No changes to `internal/parser/`, `internal/types/`, `internal/elaborate/`, `internal/codegen/`, `internal/eval/`, or any other language-semantic surface. The conflict surface is the rig operations layer, where the existing fragility is the very thing being fixed.

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [sst/opencode#971](https://github.com/sst/opencode/issues/971) — The trap that motivates Phase 4
- [Commit 98c0c408](commits/98c0c408) — `-agent-parallel` removal, motivates Phase 5
- [Commit 8f0f415d](commits/8f0f415d) — `make install` symlink, motivates Phase 3 step 5
- [Commit 07cdcbbd](commits/07cdcbbd) — `make/eval.mk` grep wildcard fix; motivates having a test
- `.claude/skills/local-ollama-eval/resources/rig_operations_runbook.md` — The runbook this milestone makes (mostly) obsolete

## Future Work

- Promote `tools/rig/` bash scripts to `cmd/ailang/cmd/rig.go` subcommands in v0.25.0 if they earn their keep
- CI for ollama-server-backed integration tests using a GPU-equipped self-hosted runner (separate project)
- Multi-model precheck (catch case where multiple variants drift from each other)
- Per-model "expected pass rate" floor — if a smoke run on gemma4:26b suddenly drops to 0%, the rig is broken before the model is

---

**Document created**: 2026-05-24
**Last updated**: 2026-05-24
