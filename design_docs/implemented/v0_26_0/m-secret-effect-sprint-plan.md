# M-SECRET-EFFECT — Sprint Plan (Phase 1 + 2)

**Design doc**: [m-secret-effect-remote-approval.md](m-secret-effect-remote-approval.md)
**Target**: v0.26.0
**Scope**: Phase 1 (gated effect + resolver + remote approval) + Phase 2 (taint + contracts). **Phase 3 (zero-standing-secret, model B) deferred.**
**Trust model**: A — service-account `op read` on the cloud coordinator.
**Risk**: Medium (touches `internal/effects`, `internal/types`; reuses shipped approval + taint machinery).
**Estimated**: ~7 working days, ~1300 LOC (impl + tests).

## Frozen decisions (from design doc Design Freeze)

- Trust model **A** (service account on server); B deferred.
- `<secret>` label **default-forbids** `{not secret}` sinks (static leak prevention).
- **Sync blocking** approval (`ApprovalCheckpoint.RequestApproval`).
- Resolution on **server** (cloud coordinator, `ailang-multivac*`).
- Push transport **ntfy** self-hosted on Cloud Run.

## Milestones

### M1 — 1Password resolver (`internal/secrets`)
- **Files**: `internal/secrets/onepassword.go` (+test). 
- **Work**: `Read(ref) (string, error)` shelling `op read`; `op://` ref-shape validation; structured `SecretUnavailable` on CLI failure (NO fallback per CLAUDE.md §2); value never logged. Backend-agnostic `Resolver` interface, 1Password the only impl.
- **LOC**: ~180 + ~150 test.
- **Deps**: none.
- **Accept**: valid ref → value; malformed ref → typed error; `op` non-zero exit → `SecretUnavailable`; test asserts value absent from any log output; `make test` green.

### M2 — `Secret` effect + `secret()` builtin
- **Files**: `internal/effects/` (effect registration), `internal/eval/` (builtin).
- **Work**: register `Secret` effect alongside IO/FS/Net; `secret(ref) -> string ! {Secret}` builtin whose handler calls the M1 resolver (label added in M5). Capability check.
- **LOC**: ~100 + ~70 test.
- **Deps**: M1.
- **Accept**: `secret("op://…")` parses, type-checks with `! {Secret}` row, runs against a fake `op`; calling without the `Secret` capability is a typed error; conflict-surface fixtures (`inbox_injection_v2.ail`, `basic.ail`) still pass.

### M3 — Approval bridge + audit + redaction
- **Files**: `internal/effects/secret_approval.go`, `internal/coordinator/approval_*.go`, tracing hook.
- **Work**: `ApprovalTypeSecret{Ref,Purpose}`; bridge the `Secret` effect to `ApprovalCheckpoint.RequestApproval()` (sync blocking); audit span `{ref,purpose,agent,chain,approver,decision,ts}`; redact resolved values in traces/logs as `«secret:op://…»`.
- **LOC**: ~150 + ~90 test.
- **Deps**: M2.
- **Accept**: cache miss blocks on approval; approve → resolves; reject → `SecretDenied`; timeout → typed error; audit span emitted; integration test with fake `op` shows the value redacted in the emitted trace.

### M4 — ntfy approvals channel + dedicated subscription (code + cloudbuild)
- **Files**: `internal/daemon/ntfy_channel.go`, `internal/daemon/approvals_subscriber.go`, `internal/server/handlers_approvals.go`, `cloudbuild-dev.yaml`, `docker/Dockerfile.ntfy`.
- **Work**: `NtfyChannel` (implements `notify.Channel` + `EventFilter`, only accepts `approval` events) posting title/body (ref+purpose+agent, never value) + Approve/Deny action buttons; per-request **signed single-use short-TTL token** minting + verification on `/api/approvals/{id}/approve|reject` (token auth, not IAM); publisher sets Pub/Sub attribute `kind=approval`; dedicated `approvals-push-sub` push subscription (filter `attributes.kind="approval"`); Cloud Build step to build/deploy `${PREFIX}-ntfy` (min-instances=1).
- **LOC**: ~250 + ~120 test.
- **Deps**: M3.
- **Accept**: `NtfyChannel.Send` posts correct payload to a fake ntfy HTTP server with two action buttons carrying valid signed tokens; reused/expired token rejected; channel skips non-approval events; cloudbuild step authored. **Note**: actual `gcloud run deploy ${PREFIX}-ntfy` + Pub/Sub subscription creation are a HUMAN/manual deploy step (flagged), not run by the executor.

### M5 — Taint `<secret>` label integration (Phase 2)
- **Files**: `internal/types/labels.go`, `internal/types/sink_check.go`, `secret()` return type, sink refinements.
- **Work**: register `secret` source label in the lattice; `secret()` returns `string<secret>`; honor `string{not secret}` sinks; ensure `CheckDeclassify` covers `<secret>`→`{not secret}` requiring `! {Declassify}`; std log/trace/Net sinks refined `{not secret}` where appropriate.
- **LOC**: ~90 + ~80 test (extend `sink_check_test.go`).
- **Deps**: M2.
- **Accept**: a `<secret>` value reaching a `{not secret}` sink without `Declassify` is a compile-time error; with `Declassify` it passes; `inbox_injection_v2.ail` unchanged behavior; **re-test the `{not secret}` parser-lookahead with the literal `secret` label** (conflict surface).

### M6 — Contracts predicate + examples + docs (Phase 2)
- **Files**: `internal/smt/` (predicate), `examples/runnable/secrets/gated_secret.ail`, `examples/runnable/secrets/leak_attempt.ail`, `std/secret` module, CHANGELOG, docs guide, prompt/reference.
- **Work**: `approved(ref)` predicate usable in `requires {}` with Z3 codegen; happy + leak examples; `std/secret` stdlib surface; docs.
- **LOC**: ~100 + examples.
- **Deps**: M5, M3.
- **Accept**: `ailang check examples/runnable/secrets/leak_attempt.ail` **fails** with a clear `<secret>`→`{not secret}` diagnostic; `gated_secret.ail` passes `ailang check`; `ailang verify` proves a `requires { approved(ref) }` contract; `make verify-examples` green; CHANGELOG + docs updated.

## Day-by-day

| Day | Milestone | Deliverable |
|-----|-----------|-------------|
| 1 | M1 | Resolver + tests green |
| 2 | M2 | `Secret` effect + builtin, fixtures pass |
| 3 | M3 | Approval bridge + redaction + audit, integration test |
| 4 | M4 | ntfy channel + signed token + cloudbuild step (deploy flagged manual) |
| 5 | M5 | `<secret>` label + sink enforcement, leak is compile error |
| 6 | M6 | contracts predicate + examples + docs |
| 7 | buffer | `make ci`, `make verify-examples`, conflict-surface re-test, polish |

## Success metrics

- [ ] `make test` + `make verify-examples` + `make ci` green
- [ ] `leak_attempt.ail` fails `ailang check`; `gated_secret.ail` passes
- [ ] End-to-end (fake `op` + fake ntfy): secret blocks → approval payload built → approve resolves → value redacted in trace
- [ ] No regression in `inbox_injection_v2.ail` / `basic.ail`
- [ ] CHANGELOG, docs guide, `std/secret` reference updated
- [ ] Conflict-surface `{not secret}` lookahead re-tested with `secret` label

## Risks

- **Parser lookahead** on `{not secret}` (historical M-PARSER-REFINEMENT-LOOKAHEAD): re-test with `secret` before M5 lands.
- **Cloud deploy not executable by agent**: M4 ships code + cloudbuild step; `gcloud run deploy` + subscription creation are manual. Flagged.
- **Trace redaction completeness**: defense-in-depth (static `{not secret}` + runtime redaction); negative example gated in CI.

---
**Created**: 2026-06-19
