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
| `--timeout 30s` | Compilation timeout with stack dump (CLI flag) |
| `--debug-compile` | Phase timing breakdown (CLI flag) |

**Full guide**: See `docs/docs/guides/debugging.md`

## Telemetry & Traces

Use the `trace-debugger` skill. Quick: `ailang trace status`, `ailang trace list --hours 1`.

## Release Workflow

**For releases**: Use the `release-manager` skill
**After release**: Use the `post-release` skill
