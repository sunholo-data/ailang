<!-- always-on: how to build, test, and land work — needed before the session
     knows which paths it will touch, so path-scoping cannot fire in time.
     Detail lives behind pointers (debugging.md, local-models.md, the release
     skills); only the operationally-biting lines belong in the file itself. -->

# Development Workflow

## Building and Testing

`make help` lists all targets (build, test, lint, fmt, ci, repl, verify-examples,
check-file-sizes…). The one that isn't guessable: `make quick-install` for a fast reinstall
after changes.

**Important**: `ailang` in PATH points to `/Users/mark/go/bin/ailang` (system). Always reinstall
after building.

## Debug Flags

Full flags table (DEBUG_*, tracing tiers, CLI profiling flags, all `AILANG_OLLAMA_*` semantics):
`docs/docs/guides/debugging.md`. The ones that bite operationally:

| Flag | Purpose |
|------|---------|
| `AILANG_EVAL_MAX_RSS=8G` | Eval memory cap per generated-code run (breach → tree killed, banked as `resource_limit`) |
| `AILANG_TRACE=off\|standard\|deep` | Tracing tier (default `standard`; `deep` ~2x overhead) |
| `AILANG_OLLAMA_V1_STREAM=1` | Streaming ollama `/v1` (default off; flips the meaning of `HTTP_TIMEOUT_SEC` — 300 buffered / 3600 streaming) |
| `AILANG_OLLAMA_NUM_CTX=N` | Pin ollama `num_ctx`; unset (default) sends none — ollama sizes from the model |
| `OLLAMA_GPU_OVERHEAD` / `OLLAMA_CONTEXT_LENGTH` | Rig memory bound — unset, ollama takes 84% of RAM and panicked the box (2026-09-03). Details load with `.claude/rules/local-models.md` |
| `MISSION_MIN_AVAIL_GB` / `MISSION_BOOT_WINDOW` | Fleet memory admission — four missions all carry `RunAtLoad=true` and stampede a boot; capping ollama did not stop the box filling (3 OOM events, 09-04/09-05). Same rule file |

**Rig gotcha:** `AILANG_OLLAMA_*` reaches a launchd job by TWO paths — the plist AND a
`launchctl setenv` domain global that no plist edit or repo grep can see. Installed plists are
copies, not symlinks, and clearing the global is order-sensitive (wrong order re-creates #618).
Before touching any of it on the rig, read "Ollama Streaming Timeouts" in
`docs/docs/guides/debugging.md`; audit live values with
`launchctl getenv <var>` paired with a known-unset control.

## Telemetry & Traces

Use the `trace-debugger` skill. Quick: `ailang trace status`, `ailang trace list --hours 1`.

## Release Workflow

**For releases**: Use the `release-manager` skill
**After release**: Use the `post-release` skill

## Pushing dev — automatic; attended work goes straight to dev

Attended work: commit straight to `dev` in the main checkout, small and often. Worktrees are for
unattended loops, coordinator tasks, and long/risky changes — they delay integration, and delay
is what conflicts. The `Stop` hook (`push_dev_on_stop.sh`) pushes, and when `dev` is ahead *and*
behind it rebases first — only if no uncommitted edit touches a rebased file, **aborting on any
conflict** so the checkout is never left mid-rebase. A conflict it reports is yours to resolve
(charter: verify with `scripts/mission_decisions.sh --check`). Opt-outs: `AILANG_AUTOREBASE=0`,
`AILANG_AUTOPUSH=0`.

**Changelog = fragments** in `changelogs/unreleased/`; `make check-changelog` refuses entries
under `## [Unreleased]`. **Worktrees self-clean:** SessionStart (`git_health.sh`) flags a stuck
rebase or diverged `dev`, and ≤6-hourly runs `worktree_sweep.sh --apply` (landed + clean + idle
≥2h + unused only; log with SHAs in `~/.ailang/state/worktree-sweep.log`; dry run: `make worktree-sweep`).

**Stdlib freeze gate on push:** a `pre-push` hook (installed by SessionStart / `make install-hooks`)
refuses a dev push that changes `std/` interfaces without `make freeze-stdlib`. Direct pushes skip
PR CI, which is how #1274 and #1318 turned dev red.

Why it exists: nothing used to push the attended path, and mission-control Gate 1 forbids
the loop from touching the shared tree, so work stranded — 25 commits deep by 2026-09-02.
