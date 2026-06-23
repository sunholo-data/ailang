# Secrets demo (M-SECRET-EFFECT)

AILANG resolves 1Password references behind a declared, capability-gated,
flow-controlled `Secret` effect. The resolved value is labelled `<secret>`
(M-TAINT-TYPES) and the type checker forbids it from reaching a `{not secret}`
sink without an explicit `! {Declassify}` step.

## Files

| File | What it shows | `ailang check` |
|------|---------------|----------------|
| [`leak_attempt.ail`](leak_attempt.ail) | A secret flowing straight into a `{not secret}` sink | **fails** (information-flow violation) |
| [`gated_secret.ail`](gated_secret.ail) | The safe pattern: declassify before the sink | passes |
| [`secret_demo.ail`](secret_demo.ail) | Runnable: resolve → declassify → print | passes; runs with `op` |

## Quick demo

```bash
make install                       # build ailang onto PATH
./examples/runnable/secrets/demo.sh
```

`demo.sh` runs three parts: static leak prevention (`ailang check`), the
capability gate (running without `--caps Secret` is denied), and runtime
resolution. By default it uses a **fake `op`** so no real 1Password is needed;
run with `USE_REAL_OP=1` (with the [`op` CLI](https://developer.1password.com/docs/cli/)
installed and signed in, or `OP_SERVICE_ACCOUNT_TOKEN` set) to hit a real vault.

## How it works

- `secret("op://Vault/Item/field", "why you need it")` returns `string<secret>`
  and requires the `Secret` capability (`--caps Secret`). The second argument is
  the human-readable **purpose** shown in the approval request on your phone.
  With no capability, the run is denied.
- Resolution shells `op read --no-newline <ref>`. If `op` is missing or fails,
  the call errors `E_SECRET_UNAVAILABLE` — there is **no silent fallback** to a
  blank/placeholder credential.
- A `! {Declassify}` function is the only way to lower the `<secret>` label so
  the value can reach an ordinary sink.

The human-in-the-loop **remote approval** flow (push-to-phone Approve/Deny) is a
coordinator/ntfy cloud feature; the local CLI runs with no approval gate.

See the [IFC labels guide](../../../docs/docs/guides/ifc-labels.mdx) and
[design doc](../../../design_docs/planned/v0_26_0/m-secret-effect-remote-approval.md).
