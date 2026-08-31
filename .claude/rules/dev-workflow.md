# Development Workflow

## Building and Testing

```bash
make build          # Build the interpreter
make install        # Install ailang to system
make quick-install  # Fast reinstall after changes (recommended)
make test           # Run all tests
make run FILE=...   # Run an AILANG file
make repl           # Start interactive REPL
```

**Important**: `ailang` in PATH points to `/Users/mark/go/bin/ailang` (system). Always reinstall after building.

## Code Quality

```bash
make test-coverage-badge  # Quick coverage check
make lint                 # Run golangci-lint
make fmt                  # Format all Go code
make ci                   # Full CI verification locally
make verify-examples      # Check example files
make check-file-sizes     # Fails if >800 lines
```

## Debug Flags

| Flag | Purpose |
|------|---------|
| `DEBUG_STRICT=1` | Fail loudly on unhandled cases |
| `DEBUG_MONO_VERBOSE=1` | Monomorphization tracing |
| `DEBUG_OPERATOR_LOWERING=1` | Operator resolution |
| `DEBUG_PARSER=1` | Token position tracing |
| `DEBUG_CODEGEN=1` | Record type fallback warnings |
| `DEBUG_APPROVAL_WATCHER=1` | ApprovalWatcher polling |
| `DEBUG_CONCURRENCY=1` | Per-request evaluator Fork/Call/Done tracing with goroutine IDs |
| `AILANG_TRACE=off\|standard\|deep` | Tracing tier (v0.12.0+). Default: `standard`. `deep` = per-call spans (~2x overhead) |
| `AILANG_TRACE_MAX_SPANS=N` | Per-trace span budget (default 500). Overflow emits `trace.truncated` rollup |
| `AILANG_NO_TRACE=1` | Back-compat alias for `AILANG_TRACE=off` |
| `AILANG_EVAL_MAX_RSS=8G` | Eval memory cap per generated-code run (process-group RSS, default 8G, `off` disables). Breach → tree killed, banked as `resource_limit` |
| `AILANG_OLLAMA_NUM_CTX=N` | Pin ollama's `num_ctx`. **Unset (default) sends none** — ollama sizes from the model. Non-tool paths only; raise/lower only for VRAM. Details: debugging guide |
| `AILANG_OLLAMA_V1_STREAM=1` | Streaming ollama `/v1` path (v0.34.0, default **off**; exactly `"1"` opts in). Flips the meaning of `HTTP_TIMEOUT_SEC` |
| `AILANG_OLLAMA_IDLE_TIMEOUT_SEC=120` | Streaming only: max silence between bytes (typed `idle-timeout`) |
| `AILANG_OLLAMA_TTFT_TIMEOUT_SEC=600` | Streaming only: max silence before first byte (cold-load allowance) |
| `AILANG_OLLAMA_HTTP_TIMEOUT_SEC` | Whole-call cap — **semantics flip with the streaming flag** (300 buffered / 3600 streaming, where `0` is rejected). Read the guide before touching it on the rig |
| `--trace-tier off\|standard\|deep` | Tracing tier CLI flag (overrides env) |
| `--timeout 30s` | Compilation timeout with stack dump (CLI flag) |
| `--debug-compile` | Phase timing breakdown (CLI flag) |
| `-cpuprofile FILE` | Write Go CPU profile (CLI flag) |
| `-memprofile FILE` | Write memory allocation profile (CLI flag) |

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
