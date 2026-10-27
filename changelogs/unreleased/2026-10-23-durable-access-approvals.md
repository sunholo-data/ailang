### Added — durable local access approval canary

- Added `ailang access-approvals --state-dir ABS_PATH --source daneel-local` with bounded JSON request/get/list/decide/claim/execute/receipt/cancel/reconcile commands, reusing the coordinator SQLite store.
- Persisted immutable worker/task/attempt/operation/argument/policy bindings, idempotent decisions and claims, expiry, cancellation and crash-safe unknown outcomes; atomic pending cap and scoped SQL pagination. The access-only SQLite opener opts into FULL commit sync; legacy openers retain NORMAL.
- Enabled only the fixed dummy/canary operation. Real vault resolution, arbitrary references and shell execution remain unconfigured; existing synchronous secrets and legacy approval APIs are unchanged. No deployment or credentials are changed.
