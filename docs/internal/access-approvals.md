# Durable access approval operator protocol

The local canary slice exposes one bounded JSON stdin/stdout command protocol:

```sh
/absolute/path/to/ailang access-approvals \
  --state-dir /absolute/private/operator-state --source daneel-local request <<'JSON'
{"owner":"mark","worker":"machine:daneel","task":"task-123","attempt":"1","operation":"dummy","resource_alias":"canary","purpose":"Run the remote approval canary","arguments":{"message":"canary complete"},"policy_version":"1","version":1}
JSON
```

Flags precede the command. `ailang ops access-approvals` is the same route.
Use a new private state directory (created as 0700); an existing directory must
be private. The database is `coordinator.db`, created as 0600. The operator must
own and protect the directory and its ancestors. Local symlink directory/file
endpoints and public modes are refused. The source and absolute path are required;
there is no implicit default, Firestore or cloud durable mode. Non-local
`AILANG_STORAGE_COORDINATOR` (or inherited `AILANG_STORAGE`) is refused. A trusted
adapter may explicitly set `AILANG_STORAGE_COORDINATOR=local` for this subprocess.
The messaging storage override does not affect it.

The process accepts exactly one JSON object, up to 16 KiB, with known protocol
fields. Success writes one JSON record to stdout. Errors write
`{"error":{"code":"...","message":"..."}}` to stderr and exit nonzero; stdout
stays empty. Startup does not probe unrelated stores. This is an internal machine
protocol; transport authentication and HTTP authorization are the host's job.

## Commands

| Command | JSON stdin | JSON stdout |
|---|---|---|
| `request` | Request identity and immutable content as in the example; optional `expires_at` | Canonical request |
| `get` | `{"id":"access-..."}` | Canonical request |
| `list` | `{"owner":"mark","worker":"machine:daneel","limit":50,"offset":0,"status":"all"}` | `{requests,total,limit,offset}` |
| `decide` | `{"id":"access-...","actor":"mark","version":1,"decision":"approved","idempotency_key":"decision-unique-id"}` | Canonical request |
| `claim` | Binding object below plus `idempotency_key` | Canonical request including opaque `execution.claim_id` |
| `execute` | Binding object below plus `claim_id` | Canonical request with durable execution/receipt |
| `receipt` | `{"id":"access-..."}` | Persisted canonical request; no receipt submission |
| `cancel` | `{"id":"access-...","actor":"mark","version":1,"reason":"task ended"}` | Canonical request |
| `reconcile` | Owner, optional worker, limit and offset | Same paginated wrapper as `list` |

`list` requires owner. Worker is optional. Limit defaults to 50 and must be 1..50;
offset is nonnegative. Status is `all` (default), `pending`, or `history`. Pending
means awaiting the human decision; history includes all other decisions. Pending
requests are oldest first to prevent queue starvation; history/all are newest first. Scope,
count and pagination happen in SQL; effective expiry is included in pending/history
counts. Reconcile and reads durably normalize expiry on the addressed page/record.
An adapter that wants to sweep all history must paginate; ordinary use never needs
a background process to make an expired authorization unusable.

`decide` accepts `approved` or `declined`, only for the exact stored owner and
version. The adapter must derive actor from an authenticated direct human action,
never model-supplied text. `cancel` accepts the bound owner or worker; it binds the
version and records a bounded reason. Receipt is read-only and cannot forge success.

Claim/execute binding object:

```json
{"id":"access-...","worker":"machine:daneel","task":"task-123","attempt":"1","operation":"dummy","arguments_digest":"COPY_FROM_REQUEST","policy_version":"1","version":1}
```

All binding fields must exactly match the immutable request. Claims must also
provide a stable, unique `idempotency_key` (1..256 bytes). Execute requires the
returned opaque `claim_id`. Neither command accepts a new operation payload; the
protected execution seam uses the persisted arguments. The only current handler
returns a bounded dummy receipt, with the optional message and operation id. It
never reads a vault, invokes `op`, launches a shell or calls an external service.

## Record and lifecycle

Canonical fields are `id,source,owner,worker,task,attempt,operation,resource_alias,
purpose,arguments,arguments_digest,policy_version,version,revision,created_at,
expires_at,decision,execution`. Optional decision fields are `decision_by,
decision_at,decision_key,claim_until,cancel_reason`. Optional execution fields are
`claim_id,claim_key,claim_until,started_at,completed_at,result,error`.
Times are UTC Unix seconds. Version is the positive immutable request version;
revision increments only on a persisted lifecycle transition. No record field is
a raw credential. The host should expose a narrower public envelope and redact
operator-only descriptions/arguments/keys when they are unnecessary.

Owner, worker, task and attempt are each 1..256 bytes; purpose is 1..1024 bytes.
Only `operation=dummy`, `resource_alias=canary` and `policy_version=1` are enabled.
Arguments may contain only an optional `message` string up to 1024 bytes. Argument
JSON is normalized by Go's JSON map serialization before SHA-256 digesting.

Request ids derive from the encoded `(source,owner,worker,task,attempt,operation,
version)` tuple. Retrying that identity returns the existing record, including
after expiry, and rejects changed arguments/purpose/resource/policy or an explicit
changed expiry. New work uses a fresh attempt or version and needs a fresh decision.
Creation atomically caps unexpired pending requests at 100 per owner and worker;
an identical delivery still succeeds at capacity. This cap is enforced by SQLite,
not a worker-supplied count.

Default pending TTL is seven days; explicit expiry must be within seven days at
creation. Approval permits claiming for at most 24 hours and never beyond the
request expiry. Each claim has a five-minute execution lease. Decisions and claims
use compare-and-swap in SQLite; one concurrent decision/claim wins. A matching
idempotency key replays the durable result; a different claim cannot take it over.

Decisions are `pending,approved,declined,expired,cancelled`. Execution states are
`unclaimed,claimed,started,completed,failed,unknown`. Execution persists `started`
before the effect, then its receipt. This dedicated opener uses WAL synchronous=FULL
on every connection, so the start commit is synced before an effect; legacy
coordinator openers keep their existing NORMAL default. Replay returns the persisted state and never
repeats a started effect. A claimed lease that expires before execution becomes
failed. A started lease that expires without a receipt becomes unknown. Cancellation
after start also becomes unknown; a late completion cannot overwrite that evidence.
Unknown outcomes require operator investigation and a fresh request if appropriate;
there is deliberately no blind retry or claim takeover command.

Error codes are `invalid_input,invalid_command,unsupported_operation,
unsupported_source,unsafe_state_dir,not_found,forbidden,conflict,capacity,
invalid_record,internal_error`. Expiry is represented by the canonical lifecycle;
a mutation against an unusable decision generally returns conflict.

This slice does not configure a protected service account, migrate credentials,
deploy a worker/host, enable real secrets, or provide remote/cloud storage. Keep
worker state separate from this trusted operator state directory. Existing
synchronous secret and coordinator approval commands keep their behavior.
