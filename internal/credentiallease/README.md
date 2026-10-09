# Rotating credential ownership

Cloud Codex subscription executions acquire `credential_leases` in the
credential secret's Firestore project before reading Secret Manager. The key is
the full, unversioned secret resource using a project **ID**. Numeric project
aliases and version paths are refused, so two spellings cannot establish
independent ownership. Both Codex job images use the same secret identity.

An owner is `CLOUD_RUN_EXECUTION` plus a random process generation. Transactions
check this owner for heartbeats, write-back authorization, quarantine and release.
A competing execution fails explicitly without fetching or installing tokens.
Dispatchers should serialize jobs sharing the credential and retry contention.

The owner holds the credential through preflight, execution and durable write-back.
Every changed valid credential file is persisted, including failed/cancelled tasks
and rotations with equal timestamp precision. Write-back errors fail the task and
quarantine the lease. A failed quarantine or release RPC leaves the existing owner
blocking reuse. Heartbeat errors cancel the executor context, which kills the
Codex process group on the Linux cloud runtime. An executor error reporting
unconfirmed process termination quarantines without reading or releasing the
credential, including deferred cleanup. Panics quarantine rather than
release because termination cannot be proven on that path.

**No TTL or automatic takeover is permitted.** OAuth servers do not enforce our
Firestore generation: expiring a lease cannot fence a still-running old process.
A crashed owner remains blocked regardless of its heartbeat age.

Attended recovery must:

1. Identify the recorded Cloud Run execution and verify it is fully terminated,
   including every task attempt. Cancel it and wait for termination if necessary.
   If termination cannot be verified, keep the lease blocked.
2. Recover its final credential only if durable write-back can be established.
   Otherwise perform a fresh cloud-only OAuth login and publish that independent
   authorization. Do not restore an old seed or the interactive/mission login.
3. After the new credential is durable and all old owners are terminated, an
   operator with Firestore administrative authority may remove the blocked lease
   document. Its ID is the SHA-256 hex of the full unversioned secret resource;
   the `credential_resource` field identifies it without credential data.

The runtime service account needs transactional Firestore read/write access in
the secret's project and Secret Manager access/add-version permissions. It must
receive an explicit isolated `CODEX_HOME`; the runtime refuses the default home.
Tests use fake stores and fake credential bytes, without OAuth or live secrets.
