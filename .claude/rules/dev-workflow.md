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
