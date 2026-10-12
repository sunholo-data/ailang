# Typed MCP JSON-RPC errors from host Invoke

Closes #1602

Hosts can return an error implementing `protocol.JSONRPCError` from `Invoke`.
Valid nonzero codes and nonempty messages produce per-message HTTP 200 SSE
errors, including through wrapped errors and the public facade. Supported
batches preserve successful siblings. Untyped and malformed hooks retain the
frozen whole-POST envelope. The adapter, A2A and gate behavior is unchanged.

## Validation

- `go test ./serveapi/... -count=1`: PASS, including SDK parity and unchanged
  frozen callback regressions.
- `make check-protocol-closure`: PASS on Linux, macOS and Windows.
- `make check-boundaries`: PASS.
- `make check-file-sizes`: PASS.
- `make test-core`: FAIL only in eight existing `internal/effects` brain tests:
  `CGO_ENABLED=0`, `go-sqlite3 requires cgo to work`. All other core packages pass.
  No C compiler was installed. This is an environment limitation separate from
  the typed MCP change.
- `make lint`: PASS (0 issues).
- Full `make test` intentionally deferred to CI per executor dispatch.

M1 wire tests were observed failing before implementation, then passing across
all supported versions. Exact wire goldens cover special characters, direct and
wrapped hooks, malformed/untyped fallbacks, mixed batches and facade adapters.
Independent sprint review found no implementation defects.

## Delivery

M1 and M2 are separate local commits; M3 records documentation and final checks.
The dated Unreleased fragment follows the explicit dispatch rather than the
plan's old `v0.32-current.md` destination. No push or PR creation was attempted;
the coordinator can use this file as the implementation PR body.

The design describes the untyped batch fallback as first-request-id; the actual
frozen behavior has `id: null`, which the added regression preserves. World
still needs a published release and dependency pin update; this sprint does not
publish a release. Evaluation by the coordinator remains a separate gate.
