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

## Pushing dev — automatic, fast-forward only

Commit straight to `dev` in the main checkout (never branch there). You do **not** need to
remember to push: `scripts/hooks/push_dev_on_stop.sh` runs as a `Stop` hook and pushes when
local `dev` is ahead of origin **and not behind**.

It refuses when the branch is ahead *and* behind, because that needs a real merge and the
conflicts land in the mission charter and changelog, where a careless resolution silently
drops decision rows. Do that merge by hand, verify with
`scripts/mission_decisions.sh --check`, then push. Opt out for a session with
`AILANG_AUTOPUSH=0`.

Why it exists: nothing used to push the attended path, and mission-control Gate 1 forbids
the loop from touching the shared tree, so work stranded — 25 commits deep by 2026-09-02.
