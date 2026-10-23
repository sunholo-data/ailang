# Maintenance containment: measured primitives, backend admission CLOSED

The 2026-10-23 macOS record is an actual no-model run on Darwin25.6.0,
Node26.12.1, installed pi SDK0.85.1 and the existing locked
`@anthropic-ai/sandbox-runtime`0.0.71. It is a scoped development qualification
record, not a universal host-security or protected-holdout claim.

Seven tests pass with no skips. One is deliberately a **negative admission test**:
the outer pi kernel read/write boundary rejects native SDK canonical-check/use
symlink races, but a second sandbox launched inside it fails with exit71 and
`sandbox-exec: sandbox_apply: Operation not permitted`. A green test therefore
asserts **CLOSED**, not successful nested execution. M8 and protected-backend M11
must remain incomplete. SDK `enableWeakerNestedSandbox` is passed only through its
Linux branch and is not a verified macOS solution. No model/provider was called.

Other measured primitives allow ordinary owned app writes/builds while refusing
cat/Python/compiled AILANG FS reads of the declared oracle root, static symlink
and hardlink escapes, protected writes, real pi SDK read/write races, and TCP/Unix
requests to independently live reference servers. Changed policy bytes fail before
child execution. The native-tool fence has its own actual pi SDK tests.

`EvalContainmentPrefix` is useful for direct **host-side** candidate grading. Its
returned policy digest is only an identity; callers still need a real admission
callback/receipt. The pi executor refuses every non-nil `Task.EvalContainment`
with typed `EvalContainmentAdmissionError` before model resolution, version probing,
CLI start or inference. Nil preserves the old payload. Unsupported backends also
refuse the requirement. `agent_env` cannot replace the trusted containment carrier.

The wrapper reuses the installed locked SDK, not a new container or inference
loop. Short per-invocation scratch directories replace the SDK's global
`/tmp/claude`; explicit denies remove its default global temp/log/TTY write grants.
Outer inference-mode sockets are scoped to that owned scratch path; tool/app
mode admits no network/Unix endpoints and removes provider credentials and loader
hooks. Current production inference mode remains unreachable through the executor.
Policy/read-only runtime exceptions are declarations and must be reviewed along
with every known source/reference copy, other trajectory root and Git object
store before any future admission. The temporary canary root does not establish
that inventory on a real rig.

Reproduction from a clean source tree (uses no model):

```sh
npm ci --prefix tools/pi-extensions/sandbox --ignore-scripts --no-audit --no-fund
node --test tools/pi-extensions/maintenance-sandbox.test.mjs
MAINTENANCE_PI_SDK=/absolute/pinned/pi/dist/index.js \
  node --experimental-strip-types --test tools/pi-extensions/maintenance-fence.test.ts
```

These actual OS qualification tests require the pinned SDKs, macOS and the AILANG
runtime. Missing support must be recorded as an admission gap, not silently skipped.
To bank a new create-only record, set `MAINTENANCE_CONTAINMENT_REPORT` to a new
absolute JSON path. The checked-in JSON records `status: blocked`, actual blocker
stderr, runtime-lock and wrapper digests. Primitive protections and blocked backend
readiness are distinct facts. A host-side tool bridge outside the outer sandbox is
a scoped follow-up design, requiring its own concrete approval and qualification.

AILANG prompt version loaded for the canary: v0.16.7. Its temporary probe declares
IO/FS, type-checks successfully, then fails the protected FS read at runtime.
Contracts/inline pure tests are skipped because this is an effectful kernel-denial
probe with no new pure rule; compiler success and denied execution are asserted
separately in the host test.
