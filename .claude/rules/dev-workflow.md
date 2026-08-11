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
| `AILANG_OLLAMA_V1_STREAM=1` | Opt into the streaming ollama `/v1` path (v0.34.0, default **off**). Exactly `"1"` opts in — flag-off wire bytes are unchanged. Changes what `AILANG_OLLAMA_HTTP_TIMEOUT_SEC` means (see below) |
| `AILANG_OLLAMA_IDLE_TIMEOUT_SEC=120` | Streaming only. Max silence **between** bytes before the stream is failed with a typed `idle-timeout`. Default 120. Unparseable/`<= 0` falls back to the default |
| `AILANG_OLLAMA_TTFT_TIMEOUT_SEC=600` | Streaming only. Max silence **before the first byte** (cold 35B load under GPU contention). Default 600. Unparseable/`<= 0` falls back to the default |
| `AILANG_OLLAMA_HTTP_TIMEOUT_SEC` | **Semantics depend on the flag.** Flag-off (buffered `/v1`): HTTP client + whole-call cap, default **300**, `0` means *no timeout*. Flag-on (streaming): the **mandatory hard deadline** on the whole stream, default **3600**, and `0`/negative/unparseable is **REJECTED** at construction (typed error, no request sent) rather than meaning unbounded. **Two delivery sites on the rig** — see below |
| `--trace-tier off\|standard\|deep` | Tracing tier CLI flag (overrides env) |
| `--timeout 30s` | Compilation timeout with stack dump (CLI flag) |
| `--debug-compile` | Phase timing breakdown (CLI flag) |
| `-cpuprofile FILE` | Write Go CPU profile (CLI flag) |
| `-memprofile FILE` | Write memory allocation profile (CLI flag) |

**Rig gotcha — `AILANG_OLLAMA_*` reaches a launchd job by TWO paths.** The plist's
`EnvironmentVariables` block, *and* the launchd **user-domain global** set by `launchctl setenv`.
No plist edit touches the global, and no `grep` over the repo can see it. Audit the live value with
a control, never the plist alone:

```bash
launchctl getenv AILANG_OLLAMA_HTTP_TIMEOUT_SEC   # empty == not pinned
launchctl getenv AILANG_NOT_A_REAL_VAR            # control: also empty, so empty is a measurement
```

Measured 2026-08-11: both repo plists were cleaned and every grep read green while the domain
global still held `1800`, so streamed requests kept logging `effective_deadline_sec = 1800` instead
of the 3600s default. A grep-over-files criterion cannot see site 2 — that is a property of the
instrument, not of the config.

**Two more things that bite in this order.** The repo plists are *source*: the installed copies in
`~/Library/LaunchAgents/` are regular files (not symlinks), updated by a manual
`cp` + `launchctl load` (`tools/launchd/nightly-eval.sh:19-21`), so **editing the repo changes
nothing on the rig**. And clearing site 2 is **ordered** — while `AILANG_OLLAMA_V1_STREAM` is off,
the buffered path's `ollamaV1Timeout()` falls back to **300s** and the global is the only thing
raising it, so `launchctl unsetenv` *before* the flag-on plists are installed re-creates the
#618 defect (895 retries / ~74.6 GPU-hours, `b67d415cd`). Install the flag-on plists first,
`launchctl unsetenv` second.

**Full guide**: See `docs/docs/guides/debugging.md`

## Telemetry & Traces

Use the `trace-debugger` skill. Quick: `ailang trace status`, `ailang trace list --hours 1`.

## Release Workflow

**For releases**: Use the `release-manager` skill
**After release**: Use the `post-release` skill
