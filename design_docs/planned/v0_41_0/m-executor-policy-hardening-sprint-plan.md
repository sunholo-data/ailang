# Sprint Plan: M-EXECUTOR-POLICY-HARDENING

## Summary

Turn AILANG's execution policy from an admission check into an enforced boundary: root-anchored
filesystem operations (`os.Root`), one network destination authorizer applied on every round trip and
stream connection, an immutable typed policy with `security_mode`, an enforced `timeout_ms` and
operator budgets, and Go-mediated agent tools that cannot reach around the gate.

**Design doc:** [m-executor-policy-hardening.md](../m-executor-policy-hardening.md)
**Duration:** 5 milestones, planned as ~8–10 working days of agent time (design doc says 15–22
engineering days; recent velocity is far above one engineer, see below)
**Dependencies:** none external — Go 1.26.6 already in `go.mod`, `os.Root` has every method M1 needs
(`OpenFile/Stat/Lstat/Mkdir/MkdirAll/Remove/Rename/ReadFile/WriteFile`, verified with `go doc os.Root`)
**Risk Level:** High (touches every FS and Net operation; the confined-worker claim waits for M5)

## Current Status Analysis

### Premise verification at HEAD `e1a360666`

Confirmed by reading the code (not the design doc's summary):

- `internal/effects/fs.go:resolveSandboxPath` joins a relative path with `filepath.Join(sandbox, path)`
  and never checks the result is still under the sandbox → `../marker.txt` escapes (F1). Every FS op
  (23 registered ops across `fs.go`, `fs_write.go`, `fs_dir.go`) goes through it, then calls plain
  `os.*` on the resolved path, which follows symlinks (F2).
- `internal/effects/net_proxy.go:directRoundTrip` resolves + IP-validates `req.URL.Hostname()` on every
  round trip (so redirects DO get IP validation) but **never re-checks the domain allowlist**; the
  allowlist is checked once in `netHTTPGet`/`buildSecureRequest` before the first request, and
  `validateRedirect` checks only protocol + count (N1).
- `stream_sse.go` / `stream_ndjson.go` build a bare `http.Transport` + `http.Client` with **no
  CheckRedirect, no resolver validation, no dial pinning** — only `StreamContext.ValidateURL`
  (hostname-string checks) runs before connect. Same defect class as N1, wider.
- `internal/platform/streamws` dials `cfg.URL` with gorilla's `websocket.Dialer` and its own
  `NetDialContext`; the core hands it a URL and nothing else.
- `policy.Policy.TimeoutMs` has no consumer outside the policy package (`rg TimeoutMs --glob '*.go'`).
- `cmd/ailang/run_policy.go` refuses `--caps/--no-budgets/--allow-env/--ai/--ai-stub` but NOT
  `--entry`, `--env`, `--env-snapshot`, `--net-allow-localhost`, `--net-allow-metadata`,
  `--stream-*`, `--stdlib-path`, `--package-dir`; it exports the sandbox via `os.Setenv`.
- `.pi/extensions/ailang-exec.ts` parses the policy TOML with regexes (`fsSandboxOf`,
  `policySummary`) and filters `ailang_cli` argv by heuristics (`cliDecision`); the `ailang_only`
  profile hands the model pi's native `read`/`write`/`edit`.
- `policy.Check` returns "admitted" for `len(row.Labels)==0` BEFORE looking at `row.Tail`.

### Velocity

- Last 7 days: 432 commits, 80k insertions across the fleet (multiple agents). Attended single-agent
  throughput on comparable security work (M-NET-EFFECT-PROXY-BOUNDARY, M-PROCESS-SUBCMD-ALLOWLIST):
  ~600–900 LOC/day including tests.
- This sprint: ~4,500 LOC estimated (design doc's per-phase figures summed, +20% buffer on tests).

### Remaining from Design Doc

All five phases are unstarted. Nothing in the design doc is implemented; V8 confirms no
`os.OpenRoot` use anywhere in `internal/` or `cmd/`.

## Proposed Milestones

### Milestone 1: Root-anchored filesystem (`internal/fileguard`) + permanent F1/F2 denial tests

**Goal:** Every sandboxed FS operation goes through an `os.Root` handle; relative traversal, outside
symlinks (absolute or relative), and race-window link swaps cannot reach outside the sandbox.

**Estimated:** ~450 implementation + ~500 tests = ~950 LOC
**Duration:** 2 days

**Tasks:**
1. **Red first.** Write `internal/effects/fs_containment_test.go` with `TestFSContainment_*` cases
   that build `t.TempDir()/sandbox` + a sibling `marker.txt` carrying a random sentinel + `link.txt`
   inside the sandbox pointing at it, then call `effects.Call(ctx, "FS", op, …)` for `readFile` on
   `../marker.txt` and `link.txt`. Assert **denial** and that the sentinel never appears. Run them,
   record the failing transcript in the commit body (mutation-test rule: the test must fail on the
   audited baseline).
2. New leaf package `internal/fileguard` (stdlib only): `Open(dir) (*Root, error)`, `Root.Rel(path)`
   (absolute-inside-root → relative, absolute-outside → typed `ErrEscapes`, relative passes through),
   thin wrappers `Open/OpenFile/Stat/Lstat/Mkdir/MkdirAll/Remove/Rename/ReadDir`, `Close`. Every
   error from `os.Root` that is an escape becomes the typed `*fileguard.EscapeError` so callers can
   distinguish "outside the root" from "not found". No second path parser: containment is decided
   by the root handle, never by string comparison.
3. `EffContext` gains a shared `fsRoot *fsRootHandle` (pointer, opened once per sandbox path, copied
   by `WithBudget`/`Clone`, `CloseFSRoot()` for the owner). Owner call sites: `runFile` paths in
   `cmd/ailang`, `internal/embed`, `serve-api`. Context copies never close it.
4. Migrate all 23 FS ops to `fsOpen/fsStat/…` helpers that dispatch to the root when
   `ctx.Env.Sandbox != ""` and to `os.*` otherwise (unsandboxed CLI behaviour preserved). `readCapped`
   takes an opened `*os.File`. Rename resolves both operands through the same root. `removeDirResult`
   uses `Lstat` (so a symlink to a directory is "not a directory", the entry is never dereferenced).
5. Boolean probes (`exists/isDir/isFile`) return `false` + `logSandboxReject` on escape; Result ops
   return `Err`; error ops return an error — no outside side effect in any case.
6. Adversarial tests per AC2/AC3: traversal, sibling-prefix (`/tmp/sandbox2` vs `/tmp/sandbox`),
   intermediate-dir symlink, final symlink, valid in-root relative symlink (must keep working),
   absolute symlink back into the root (now denied — documented migration), rename both operands,
   write/append/mkdir side-effect absence, a concurrent symlink-swap loop under `-race`, and
   descriptor cleanup (`CloseFSRoot` idempotent; open count via `/dev/fd` on darwin/linux).
7. `GOOS=js` build tag: `fileguard.Open` returns `ErrUnsupportedPlatform`; the FS ops refuse when a
   sandbox is set on that platform instead of falling back.

**Acceptance Criteria:**
- [ ] `TestFSContainment_Traversal` and `TestFSContainment_Symlink` fail on baseline, pass after
- [ ] Every registered FS op has a containment test (table-driven over the registry names — a new
      op with no row fails the test)
- [ ] `TestFSSandbox_AbsolutePathWithinSandbox`, `TestFSRenameFile_*`, `TestFS_RemoveDirResult_EmptyOnly`,
      `TestSandboxReject_*` unchanged and green
- [ ] `go test -race ./internal/effects -run 'FSContainment|FSSandbox|Rename|RemoveDir'` green
- [ ] `make test-core`, `make lint`, `make check-boundaries` green

**Risks:**
- Programs relying on absolute symlinks inside the sandbox break — Mitigation: documented in the
  CHANGELOG as an intentional change; relative symlinks keep working.
- Root lifetime with `Clone` under serve-api concurrency — Mitigation: the handle is a pointer to a
  refcount-free "opened once, closed by owner" holder; `Clone` shares it and never closes.

### Milestone 2: One network destination authorizer, redirects, and Stream transports

**Goal:** A single `netAuthorizer` decides scheme + hostname + allowlist + resolved-IP for **every**
round trip and connection: HTTP GET/POST/request/requestBytes, SSE GET/POST, NDJSON POST, WebSocket.

**Estimated:** ~350 implementation + ~450 tests = ~800 LOC
**Duration:** 2 days

**Tasks:**
1. **Red first.** `internal/effects/net_redirect_containment_test.go`: `httptest` server whose
   `/start` 302s to `http://denied.example/final` and records `/final` hits; `ctx.Net.AllowedDomains =
   ["allowed.example"]`, injected `lookupIP` → `203.0.113.10`, injected `dialContext` → the listener,
   `proxySelector` → nil. Call `Net.httpGet("http://allowed.example/start")`. Assert `/final` was
   **never** hit and the error names the domain. Same shape for `httpRequest`, SSE connect and NDJSON.
2. New `internal/effects/net_authorize.go`: `type destinationPolicy` (scheme allow, domain list,
   localhost/metadata flags) built from `NetContext` or `StreamContext`; `authorizeURL(u)` (scheme,
   non-empty host, no userinfo, allowlist with consistent lowercase + trailing-dot normalisation,
   label-boundary wildcard) and `authorizeAndPin(ctx, host)` (resolve with the context, validate
   every candidate, return the pinned dial address). `isAllowedDomain`/`isStreamAllowedDomain`
   collapse into one implementation (`*.example.com` must NOT match `evil-example.com`; today's
   `HasSuffix(".example.com")` is label-safe but `isStreamAllowedDomain` and `matchDomain` are two
   copies — one remains).
3. `netProxyRoundTripper.RoundTrip` calls `authorizeURL` **before** selecting proxy/direct, so every
   redirect hop is authorized before any dial. `validateRedirect` also authorizes (belt and braces,
   and it is where the error message is legible). Cross-origin redirects strip `Authorization`,
   `Cookie`, `Proxy-Authorization` and any header the policy marks sensitive (`ctx.Net.SensitiveHeaders`,
   default the three above) in all four HTTP entrypoints, not only `buildSecureRequest`.
4. SSE/NDJSON: replace the bare transport with `netProxyRoundTripper{ctx}` semantics adapted for the
   Stream policy (a `streamRoundTripper` that shares the authorizer + pinned dial, keeps the connect
   timeouts, and installs a `CheckRedirect` that re-authorizes). Whole-response timeout stays 0.
5. `StreamDialConfig` gains `DialContext func(ctx, network, addr) (net.Conn, error)` and
   `Authorize func(*url.URL) error`; `StreamConnect` fills them from the authorizer; `streamws.Open`
   uses `cfg.DialContext` as `NetDialContext` (no independent re-resolution) and `cfg.Authorize` on
   the handshake's redirect (gorilla does not follow redirects; assert that in a test).
6. Restricted-mode hooks for M3: `NetContext.RefuseProxy bool` → `selectProxy` returns a named
   error when a proxy would be used.

**Acceptance Criteria:**
- [ ] Redirect containment tests fail on baseline (record transcript), pass after — for httpGet,
      httpRequest, sseConnect, ndjsonPost
- [ ] Table test over allowed/denied host, redirect to denied, resolved private IP, IPv4/IPv6/mapped
      IPv4 (`::ffff:10.0.0.1`), cancellation via `GoCtx`, and TLS SNI = original hostname (tls
      `httptest.NewTLSServer` + pinned dial) for every entrypoint
- [ ] Proxy tests in `net_proxy_test.go` unchanged and green; `RefuseProxy` test proves no direct fallback
- [ ] `go test -race ./internal/effects ./internal/platform/streamws` green

**Risks:**
- gorilla `Dialer` with a custom `NetDialContext` and TLS — Mitigation: keep `TLSClientConfig.ServerName`
  = original host; test against `httptest.NewTLSServer` with the test cert.

### Milestone 3: Immutable typed policy, `security_mode`, entrypoint pin, enforced `timeout_ms`, bounded supervisor

**Goal:** Operator authority is decoded once into an immutable `policy.Resolved`, every widening
argv route is refused, restricted mode admits only the effects with tested adapters, and wall-time
is enforced by a parent supervisor around a child worker.

**Estimated:** ~500 implementation + ~450 tests = ~950 LOC
**Duration:** 2 days

**Tasks:**
1. `internal/policy`: add `SecurityMode string` (`restricted` default, `trusted_host`), strict TOML
   decode (already refuses unknown keys), `Resolve(p) (*Resolved, error)` producing an immutable
   struct (mode, root, admitted effects, net grants, budgets, `Timeout time.Duration`,
   `MaxSourceBytes`, entry, `RestrictedEffects` list) and validating: `timeout_ms` > 0 and
   ≤ 24h; restricted mode + any of `Process/Env/Secret/AI/SharedCache/…` (anything not in the tested
   set `IO, FS, Net, Clock, Rand, Stream`) → named error; `ailang_only` agents with `trusted_host` →
   error at the executor (`internal/executor/agent_policy.go`).
2. `policy.Check`: check `row.Tail != nil` **before** the empty-labels shortcut; direct unit test with
   an open empty row.
3. `cmd/ailang/run_policy.go`: refuse `--entry` (entrypoint comes from the policy), `--env`,
   `--env-snapshot`, `--net-allow-localhost`, `--net-allow-metadata`, `--stream-allow-*`,
   `--stdlib-path`, `--package-dir` under `--policy`; derive Stream domains from `net_allow` when
   Stream is admitted; set `RefuseProxy` in restricted mode; read the source bytes ONCE, check
   `max_source_bytes`, admit and execute from the same bytes (`runFile` gets the pre-read source);
   bank the resolved config + module-graph digest in the `policy:` line.
4. Supervisor: `ailang run --policy` becomes parent + child. Parent re-execs the same binary with
   `--policy-worker` (internal flag) + a framed control channel (an extra pipe fd, JSON lines) where
   the child writes the admission decision; stdout/stderr stay data. Parent uses
   `exec.CommandContext` with `timeout_ms` from the resolved policy, `proctree.Configure` for group
   kill, bounded output capture (`8 MiB` combined; stop reading at the cap, kill the child), and
   returns a versioned result envelope `{version:1, stage, effect, policy_digest, reason}` on
   timeout/limit. Deadline starts before source load. Windows: refuse restricted mode with a named
   error (no descendant-termination claim).
5. Environment allowlist for the worker: `PATH, HOME, TMPDIR, TZ, LANG, AILANG_STDLIB_PATH` + the
   policy-derived `AILANG_FS_SANDBOX`; nothing else.

**Acceptance Criteria:**
- [ ] `TestRunPolicy_TimeoutMsEnforced`: a program that loops forever with `timeout_ms = 200` exits
      within 2s with the envelope naming `timeout`; the grandchild fixture (Process cap in
      trusted_host mode, `sleep 30`) is dead after
- [ ] `TestRunPolicy_RefusesWideningFlags` covers every flag above by name
- [ ] `TestCheck_OpenEmptyRowDenied` (policy) — fails on baseline
- [ ] `TestResolve_RestrictedRefusesProcess`, `…TrustedHostKeepsProcess`, `…AilangOnlyRejectsTrustedHost`
- [ ] Existing `TestRunPolicy_*` (33s suite) green
- [ ] Every restricted-refusal is a startup refusal with a named reason; no permissive fallback

**Risks:**
- Re-exec changes the CLI's process shape for `--policy` runs — Mitigation: only `--policy` takes the
  supervised path; everything else is untouched.

### Milestone 4: Operator budgets and Go-mediated agent tools

**Goal:** `[budgets]` in the policy become an enforced shared ceiling independent of source frames;
the `ailang_only` lane's file/CLI tools are served by the Go binary from the resolved policy, not by
pi's native tools or regex/heuristic argv filtering.

**Estimated:** ~600 implementation + ~500 tests = ~1,100 LOC
**Duration:** 2 days

**Tasks:**
1. `internal/effects/budget_operator.go`: `OperatorBudget` (mutex-guarded per-effect counters,
   explicit 0 = zero ops, absent = unlimited) charged in `RequireCapWithBudget` at the canonical
   charge scope (`budgetChargeDepth == 0`) before the frame charge; shared by pointer across
   `WithBudget`/`Clone`; `--no-budgets` does not touch it. `run_policy.go` stops refusing `[budgets]`
   and installs it.
2. `internal/policytool` (leaf, imports `policy` + `fileguard`): a dispatch table
   `{read, write, edit, check, fmt, iface, test, docs_search, examples, builtins}` with typed request
   structs (no argv); `read/write/edit` are root-anchored through `fileguard`; `edit` keeps pi's
   expected-content contract (old string must match exactly once); `check/fmt/iface/test` build the
   argv themselves from validated relative paths inside the root, never from caller strings.
3. `ailang policy-tool --policy <p> <op>` reads one JSON request on stdin, writes one JSON response —
   the local endpoint the pi adapter calls. Policy path comes from the launcher env, never from the
   request.
4. `.pi/extensions/ailang-exec.ts`: `fsSandboxOf`/`policySummary`/`cliDecision` replaced by
   `ailang policy-tool --policy … summary` (Go-produced JSON); `ailang_cli` forwards `{op, …}` to
   `policy-tool`; new `ailang_read/ailang_write/ailang_edit` tools registered by the extension with
   pi's canonical parameter shapes. `executor.ProfileTools("ailang_only")` → `AilangRead, AilangWrite,
   AilangEdit, …` (native `Read/Write/Edit` removed from the restricted profile); `canonicalToPi` maps
   them. `make pi-assets` regenerates embedded copies.
5. Direct tool tests (no LLM): `internal/policytool` tests read/write/edit against outside marker,
   policy file, extension dir; `check` with `../x.ail`, `--stdlib-path /etc`, `-o /tmp/x`; assert
   refusal. `internal/executor/pi` tests assert the exact registered tool list for `ailang_only`.

**Acceptance Criteria:**
- [ ] `TestOperatorBudget_ExactlyN` across nested calls, imports and `WithBudget` scopes; source
      `@limit` cannot raise it; `-race` green on an async Stream charge test
- [ ] `TestPolicyTool_*` deny outside marker / policy / extension paths and path-bearing options
- [ ] `TestToolArgs_AilangOnlyProfile` proves the registered surface has no native read/write/edit
- [ ] `make verify-pi-assets` green; extension unit tests (`npm test` in `.pi/`) green

**Risks:**
- pi tool registration shape for read/edit differs from native — Mitigation: keep parameter names
  identical (`path`, `content`, `oldText`, `newText`) so the model's habits transfer.

### Milestone 5: End-to-end proof, migration, docs

**Goal:** The restricted-worker claim is made only after the integrated suite passes; operators have
a migration path for Process-capable `ailang_only` agents.

**Estimated:** ~150 implementation + ~350 tests/docs = ~500 LOC
**Duration:** 1 day

**Tasks:**
1. `cmd/ailang/run_policy_containment_test.go`: builds the checkout binary, runs `.ail` fixtures
   under a generated policy: traversal read, symlink read, redirect (hermetic `httptest` + the
   `AILANG_NET_TEST_RESOLVER` hooks are package-private, so the CLI test uses a local listener on an
   allowed literal IP with `net_allow_http`), timeout, budget exhaustion, refused Process in
   restricted mode, trusted_host positive control.
2. Registry inventory: `ailang coordinator agents` output → list every `ailang_only` agent whose
   policy admits Process/AI/Env; write the migration note (`docs/docs/guides/agent-policy.md` or the
   existing tool-policy guide) with `security_mode = "trusted_host"` examples; update the example
   policies under `examples/` / `internal/executor/pi/testdata`.
3. CHANGELOG entry (Security), `docs/LIMITATIONS.md` residual trust assumptions (hard links, devices,
   mounts, CPU/memory, proxy, GOOS=js/windows), supported-platform matrix.
4. Final validation: `make test`, `make lint`, `make check-boundaries`, `make verify-pi-assets`,
   targeted `-race`, `make verify-examples`.

**Acceptance Criteria:**
- [ ] All AC1–AC12 rows in the design doc have a named test or doc artifact
- [ ] Four conflict fixtures (`effects_fs_io`, `fs_walk_glob`, `process_subcmd_allowlist`,
      `net_only_admitted`) still `ailang check` clean and run under trusted mode
- [ ] Design doc moved to `design_docs/implemented/v0_41_0/` with a verification log

## Success Metrics

- Every counterexample from the audit (F1, F2, N1) is a permanent, red-on-baseline test.
- `go test -race` on `internal/effects`, `internal/policy`, `internal/fileguard`, `internal/policytool`.
- Docs: CHANGELOG, `docs/LIMITATIONS.md`, agent-policy guide, example policies.
- `make test`, `make lint`, `make check-boundaries`, `make verify-pi-assets` green.

## Dependencies

- None external. `internal/proctree` is reused for group kill (M3).

## Open Questions (decided by the design doc's D1–D6; recorded here so execution does not re-litigate)

- D1/D2: `os.Root` + one authorizer — adopted.
- D3: `security_mode` absent = restricted; `ailang_only` requires restricted — adopted. Existing
  policies that admit Process will start failing at startup with a named migration message; M5
  inventories them.
- D5: proposed default caps (1 MiB entry, 16 MiB graph, 8 MiB output, 8 MiB per FS transfer) — adopted
  as `policy.Resolved` defaults, overridable by policy fields.
- D6: Linux/macOS first; GOOS=js/windows refuse restricted mode.

## Notes

- Commit per milestone on `dev` (attended session, main checkout). Red-first transcripts go in the
  commit body for M1, M2, M3 (mutation-test rule).
- The `CLI_DEFAULT_ALLOW` list in the pi extension is kept as the default `cli_allow` in Go so the
  operator-visible behaviour does not change when the parser moves.
