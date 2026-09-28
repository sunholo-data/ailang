# M-RIG-GPU-ADMISSION-GATEWAY: Enforce the rig lock at ollama, not by convention

**Status**: Phase 2 CUT OVER (2026-09-28): ollama on :11435, gateway on :11434, pi/opencode/ailang clients send the lease, cloud models pass. Remaining: Daneel minting (cross-repo), bypass detector, Phase 3 soak (see "Phase 2 as built")
**Target**: v0_44_2+
**Priority**: P1 (rig throughput; the rig produced zero usable eval rows on 2026-09-23 and 2026-09-27)
**Estimated**: 4–5 days (Phase 0 spike DONE 2026-09-27; the children.go decision is still open)
**Dependencies**: M-RIG-LOCK-ENFORCE (v0.25), M-RIG-LOCK-YIELD (v0.38); the stranded
`internal/riglock/children.go` (see Phase 0)
**Scope**: Rig tooling (Go shell + one AILANG policy function + launchd). No language, compiler or stdlib surface.

**Quorum round 1 (2026-09-27)**: BLOCKED. gpt6-astra and gemini-3-1-pro answered; glm and kimi were unreachable.
Both objections were accepted and closed in this revision without a second round (Mark: one round):
(1) ":11435 is not private": answered with an explicit **Threat model**, a **bypass detector**, and optional pf
hardening (D5). (2) "V10/V11 unmeasured": both are now **measured** (V10, V11 below). Artifact:
`.ailang/state/mission-quorum/m-rig-gpu-admission-gateway-2026-09-27T17-20-17Z.json`.

**Quorum trigger**: #1 fired (the doc has design-freeze items D1–D4), and #4 partially fired (the premise
about client templating in opencode and pi is external; it is now measured, see V9). Run `ailang design-quorum` before planning.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Admission order is still arrival order within a class. No new nondeterminism in eval results |
| A2: Replayability | +1 | Every GPU request gets one log line (lease, class, decision, wait, prompt tokens), so a rig day can be replayed from one ledger instead of reconstructed from a 190 MB ollama log |
| A3: Effect Legibility | +1 | Who is using the GPU becomes an explicit, logged fact. Today it is inferred from `lsof` and log archaeology |
| A4: Explicit Authority | +1 | GPU access becomes a lease you hold, not ambient access to a port. Unowned long requests are refused while someone else holds the lease |
| A5: Bounded Verification | 0 | — |
| A6: Safe Concurrency | +1 | Turns the advisory mkdir lock into enforcement at the one shared resource |
| A7: Machines First | 0 | Rejections are structured (holder, lease age, class). No human-convenience trade-off |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Per-lease GPU-seconds become measurable. Wasted prefill (cancelled requests) becomes countable |
| A10: Composability | 0 | Clients are unchanged except for their base URL |
| A11: Structured Failure | +1 | A contended request fails at once with the holder's name, instead of a 4m59s timeout that pi retries 3× |
| A12: System Boundary | +1 | The ollama boundary gets one gatekeeper instead of N callers with N conventions |

**Net Score: +8**, so proceed. There are no −1 scores on A1, A3, A4 or A7.

## Problem Statement

The rig is one Mac Studio GPU running ollama with `NUM_PARALLEL=1` and `MAX_LOADED_MODELS=2`. Three
consumers (nightly-eval, os-rotation-filler, ad-hoc sessions) and now a second OS user (Daneel) share it.
Mutual exclusion is `rig-lock.sh` / `internal/riglock` / `daneel_rig.ail`: an **advisory** mkdir lock
that gates **starting a job**. Nothing gates **sending a request to ollama**. Measured in the 2026-09-27 audit:

| Symptom | Measurement (2026-09-27) | Why the lock didn't prevent it |
|---|---|---|
| Orphaned agents | 2 `opencode run` processes, PPID 1, 15h and 18h old, still streaming prompts to `qwen3.8:27b` | Born at the 00:26 and 03:11 watchdog wedge-kills. opencode runs in its own pgroup, so `kill -TERM -$pgid` missed it. It holds no lock and nothing checks |
| Queue collapse | 92 `/v1/chat/completions` requests took more than 15 minutes (the daily norm is 5–16). The nightly scored 0/10 and was marked INVALID (`infra_outage`). The filler had 0 passes | Every lock-holding eval queued behind the orphans at `NUM_PARALLEL=1` |
| Wasted prefill | 500 at exactly 4m59s, then `Request terminated: context canceled`. pi retries 3× | pi 0.85's `DEFAULT_HTTP_IDLE_TIMEOUT_MS = 300_000` cancels a queued 34k–138k-token prefill. Each retry re-queues and re-prefills |
| Unlocked traffic | 40,581 `/api/embed` calls in two days (`ailang docs embed-warmup`, run by `scripts/hooks/session_start.sh` on every session). Email triage `qwen3:8b` generations until today | These paths never call riglock. It is not in `cmd/ailang/docs.go` |
| Same shape before | 09-23: 78% prefill-busy, 106 requests over 15 minutes, nightly 0% | — |

The small fixes landed in 41f2d7640 cover part of this:
- The watchdog now kills the whole process tree.
- An opt-in orphan reaper exists.
- `~/.pi/agent/settings.json` now sets `httpIdleTimeoutMs=1800000`.

Those treat symptoms. Any process that can reach `127.0.0.1:11434` can still use the GPU with no lease. That covers:
- a future orphan from a crash the watchdog never saw;
- a new tool that forgets the lock;
- another OS user's process.

The fix is to enforce the lock at the only shared resource.

## Goals

**Primary**: while a lease is held, only the holder can send long requests (chat/generate) to the GPU. Every
other long request fails at once, naming the holder.

Success metrics:
1. **Orphaned or unleased chat traffic during a held lease: 0 admitted.** It is refused and logged (gateway ledger).
2. **Requests that fail at the client's timeout because of queueing: 0.** A 4m59s 500 no longer occurs.
3. **Nightly INVALID nights from `infra_outage`: 0 over 14 nights.** Baseline: 2 in the last 7.
4. A single command answers "who used the GPU, for how long, and how much prefill was wasted" for any day
   (the ledger). Today's audit needed about 30 ad-hoc commands against the raw ollama log.

## High-Impact Decisions

| # | Decision | Options | Recommendation | Who | Change cost |
|---|---|---|---|---|---|
| D1 | How the gateway identifies a caller | (a) lease token in the base URL path; (b) TCP socket → PID lookup; (c) custom header `X-Rig-Lease`; (d) **lease token as the API key** (`Authorization: Bearer <token>`) | **(d)**. Measured in the Phase 0 spike (V9): **pi does not template `baseUrl`**, so it sent the literal `${SPIKE_LEASE}`, which rules out (a). Both clients template `apiKey` and `headers` from the environment. (b) is refuted on a multi-user rig (V6). Every OpenAI-compatible client already sends an API key, and ollama ignores it (pi's configured key is literally `ollama`). Keep (c) as a fallback for native-API callers with no key field (Daneel's `/api/generate`) | Mark | Medium |
| D2 | Policy for a long request with no valid lease while a lease is held | reject with 423 and the holder; queue behind; admit | **Reject.** Queueing is exactly what collapsed 09-27. When no lease is held, admit it and mark it `unleased` in the ledger, so ad-hoc use keeps working | Mark | Low |
| D3 | Where the gateway runs | standalone `dev.ailang.rig-gate` launchd job; inside `ailang serve` (dev.ailang.server) | **Standalone.** dev.ailang.server shows last-exit −9 in `launchctl list`, and restarting it must not drop GPU access | Agent | Low |
| D5 | Hard enforcement of the private port | none (cooperative model); **pf anchor** allowing TCP to 127.0.0.1:11435 only from the gateway's uid (root, operator-installed) | **Start without pf, with the bypass detector on.** Add pf only if the detector ever fires. A root firewall rule on the rig is Mark's call and not needed for the observed failures | Mark | Low |
| D4 | Gateway down | clients fail; fall back to direct ollama | **Clients fail** (no silent fallback, per CLAUDE.md §2). rig-watchdog kickstarts it like ollama | Mark | Low |

**Kept as is (measured, not reopened)**: `OLLAMA_MAX_LOADED_MODELS=2`. Mark 2026-09-27: `1` caused a
large lag whenever a model swap was actually wanted. The gateway does not try to control residency.
The one exception is refusing an **unleased** request for a model that is *not loaded* while a lease is
held, because that load could evict the holder's model.

### Threat model (quorum objection 1)

The gateway defends against **accidents, not adversaries**. Every failure measured on 2026-09-27 was a
client using its *configured* endpoint without a lease: an orphaned agent, a tool that never took the lock,
or a second user's job. All of these reach :11434 and so hit the gateway. It does **not** stop a process that
deliberately targets the private port. Loopback TCP cannot be restricted per uid without a root pf rule (D5).
Two things keep that gap visible rather than silent:

- **Bypass detector.** ollama logs one `[GIN]` line per request. The gateway ledger logs one line per request
  it proxied. A reconciler run by rig-watchdog every 60 s compares the two counts for `/v1/chat/completions`,
  `/api/chat` and `/api/generate` over the last complete minute. An excess on ollama's side means a bypass: log it
  and send one controlplane message per hour. It needs no identity, only counts, so it works across users.
- **The private port is not published**: not in `models.yml`, the opencode or pi configs, or any doc except this
  one and the plist.

### Design Freeze

- [ ] D1 lease token as the API key (Bearer), with header fallback. The spike measured it, so ratify
- [ ] D2 reject-while-held
- [ ] D4 no fallback
- [ ] D5 no pf at first; bypass detector on

## Solution Design

### Overview

```
  opencode / pi / eparse / ailang builtins      (any OS user)
          │  http://127.0.0.1:11434/…   Authorization: Bearer <lease-token>
          ▼
  rig-gate (Go shell, 127.0.0.1:11434) ── admit(req, lease, loaded) ── policy.ail (pure)
          │  admitted → reverse-proxy, streaming
          ▼
  ollama serve (127.0.0.1:11435, private)
```

- **Lease = rig lock + token (sent as the API key, D1d).** Acquiring the lock (shell, Go or `daneel_rig.ail`) also writes a random
  token into the lock directory (`/Users/Shared/ailang/rig.lock.d/token`, group `rig`, mode 0640). The
  holder exports `AILANG_RIG_LEASE=<tok>`. Release removes the directory, so the token dies with it.
  **This is what kills orphans structurally**: an orphan's token is revoked when its job's lock is
  released or stolen, and its next request gets a 423. No process tracking is needed.
- **Classes**: `short` is `/api/embed`, `/api/tags`, `/api/ps`, `/api/version`, `/api/show`, always admitted.
  The embedder lives in the second loaded-model slot and is cheap (median 78 ms). `long` is
  `/v1/chat/completions`, `/api/chat`, `/api/generate` and `/v1/completions`, admitted by policy.
- **Policy in AILANG** (project rule: policy decisions belong in AILANG, Go is the shell).
  `admit(class, leaseState, tokenMatches, modelLoaded) -> Admit | Reject(reason)` is a pure function
  called through the existing embed path (about 13 µs per call). Go owns sockets, streaming and the ledger.
- **Ledger**: one JSONL line per request in `/Users/Shared/ailang/rig-gate.jsonl`, with these fields:
  `ts, lease_holder, token_ok, class, model, decision, queue_ms, duration_ms, prompt_tokens, status, cancelled`.
  `ailang rig ledger --day D` summarises it.

### Implementation Plan

**Phase 0: spike and stranded work (1 day, gates the rest)**
1. ~~Prove per-run token injection.~~ **DONE 2026-09-27 (V9).** The API key and a header are templated in
   both harnesses, and `baseUrl` is templated only in opencode, so D1 moves to (d). Remaining for Daneel:
   `daneel_model.ail:48` calls native `/api/generate` with no key field, so it sends `X-Rig-Lease` (D1c).
2. Decide the fate of `internal/riglock/children.go` + `children_test.go`: uncommitted since
   2026-09-22, same diagnosis (a dead holder's orphaned child keeps streaming while the lock reads free).
   Under D1 the token makes child registration redundant for *admission*, but it is still useful for
   *reaping* (kill_tree cannot see a tree whose root is already gone). Either land it as the reaper's
   input or delete it. Do not leave it stranded.

**Phase 1: gateway (2 days)**
- `cmd/ailang/rig_gate.go` + `internal/riggate/`: a streaming reverse proxy. It must pass SSE/NDJSON through
  unbuffered and propagate client cancellation to ollama (so a cancelled request frees the slot).
- `internal/riggate/policy.ail` + Go binding. Lease read from the lock directory, with no caching beyond 1 s.
- Ledger writer.

**Phase 2: cutover (1 day)**
- `tools/launchd/dev.ollama.serve.plist`: `OLLAMA_HOST=127.0.0.1:11435`. New `dev.ailang.rig-gate.plist`
  on :11434. rig-watchdog probes and kickstarts both (the 127.0.0.1 pin stays, see #557).
- Lease minting in `rig-lock.sh`, `internal/riglock`, and Daneel's `daneel_rig.ail` (the wire format is shared,
  and all three must mint and read identically; add it to `yield_shell_test.go`'s cross-implementation checks).
- Executors inject the leased base URL (per the Phase 0 result).

**Phase 3: observe and tighten (1 day)**
- A 7-night soak. Ledger report compared with the baseline table above.
- Then retire the pi 30-minute idle-timeout workaround to a sane value (a request is either admitted and
  served, or rejected at once) and move it from rig-only config into something the executor sets or checks.

### Files to Modify/Create

- `cmd/ailang/rig_gate.go` (new, ~150 LOC): the command, flags and launchd entry
- `internal/riggate/proxy.go` (new, ~250 LOC): streaming proxy, cancel propagation, ledger
- `internal/riggate/policy.ail` (new, ~40 LOC): the pure admission decision
- `internal/riglock/riglock.go` (~+40): mint and read the lease token
- `tools/launchd/rig-lock.sh` (~+25): shell half of token minting
- `tools/launchd/dev.ollama.serve.plist`: port 11435
- `tools/launchd/dev.ailang.rig-gate.plist` (new)
- `tools/launchd/rig-watchdog.sh` (~+10): probe the gateway too
- `internal/executor/pi/pi.go`, `internal/executor/opencode/*` (~+20 each): leased base URL
- `~/dev/daneel/tools/daneel_rig.ail`, `daneel_model.ail`: mint and read the token, parameterise the endpoint (Daneel repo, cross-repo PR)

### Phase 1 as built (2026-09-27)

- `internal/dashboard_transforms/rig_admission.ail`: the decision (`label` with 7 inline tests, `decide`, `isLong` with 11). Labels: `short | unleased | legacy-holder | lease | refused-foreign | refused-none`.
- **`legacy-holder` was added while building.** A holder that minted no token (an old binary, or Daneel's lock code until it mints) would otherwise have its own work refused, because nobody could match. It is admitted and labelled, which is the pre-gateway behaviour, so the ledger shows how often it happens. It disappears once every holder mints.
- **The `modelLoaded` fact was dropped.** D2 already refuses every unleased long request while a lease is held, so a separate eviction rule adds nothing.
- `internal/riggate/`: policy bridge (no Go fallback, so a policy error returns 500), proxy and ledger. Refusal is 423 and upstream failure is 502, never 503 (V11). Token precedence is `X-Rig-Lease`, then `Authorization: Bearer`. The values `none` and `ollama` count as no token.
- `internal/riglock/lease.go` + `tools/launchd/rig-lock.sh`: both mint a 32-hex `token` (0640) into the lock dir on acquire and export `AILANG_RIG_LEASE`. Release resets it to `none`.
- `internal/executor/environment.go`: every agent environment defines `AILANG_RIG_LEASE` (**V12**, measured: pi refuses to start when a templated `apiKey` or header variable is unset, and its template syntax has no default).
- `ailang rig-gate`: refuses to start when `--listen` equals `--upstream` or `--listen` is not loopback. `tools/launchd/dev.ailang.rig-gate.plist` is **not installed** and its header lists the cutover order.
- Tests: an admission table (8 cases), refusal under 100 ms, revocation cuts off an orphan, byte-identical and incremental streaming for SSE and NDJSON, client cancel reaching upstream, and the ledger. Mutation-checked: accepting a foreign token fails 2 tests. `FlushInterval=0` is an *equivalent* mutant, because Go 1.26 flushes any response of unknown length.
- End-to-end on the rig: the real binary on :18998 in front of the real ollama, with a fake held lease. `/api/tags` returned 200 in 4 ms. Unleased and foreign long requests got 423 in 0.5 ms and never reached the GPU. There are 3 ledger lines.

**Remaining (Phase 2 + detector):** Daneel mints and sends the token (cross-repo); an `ollama-rig` pi provider plus the opencode apiKey template; the ollama port move; installing the plist; the bypass reconciler in rig-watchdog. Each needs the step before it. The order is in the plist header.

### Phase 2 as built (2026-09-28)

- **Found while cutting over: Ollama Cloud models.** `…:cloud` / `…-cloud` models go through the local daemon to ollama.com. The doc never mentioned them, but under D2 they would be refused while a lease is held, and the mission fleet runs on them. The gateway now peeks at the body's `model` (at most 64 MB, restored for the proxy). `isCloudModel` in `rig_admission.ail` admits them with the label `cloud`.
- **Found: motoko and `--ai ollama:` never sent a token.** motoko is an AILANG program whose model calls go through ailang's OpenAI-compatible client. `riglock.WithLease` (loopback only, read per request) is used by `internal/ai/openai` and `internal/ai/ollama`. Seam test: a real `Generate` against a loopback server carries the lease, and removing the wrap fails it.
- **pi: a `!` command value, not an `ollama-rig` provider.** pi's `resolve-config-value.js` runs a `!`-prefixed value through the shell (cached per process). `"X-Rig-Lease": "!printf %s \"${AILANG_RIG_LEASE:-none}\""` sends `none` when the variable is unset, so V12's startup refusal does not apply and model ids stay `ollama/…`. Measured with a capture server: unset → `none`, set → the token, and 423 is final after 1 request.
- **opencode**: `"apiKey": "{env:AILANG_RIG_LEASE}"`. Measured: set → `Authorization: Bearer <tok>`, unset → no Authorization header, the run proceeds.
- **Daneel deferred.** `tools/daneel` runs eparse under Daneel's lock, and eparse calls ollama. If `daneel_rig.ail` minted a token before eparse and `daneel_model.ail` sent it, Daneel's own indexing would be refused. Until then Daneel is `legacy-holder`. Latent bug for that PR: `removeLockDir` is non-recursive and does not delete `token`, so Daneel cannot reclaim a dead Go/shell holder's lock (it reads "busy" until someone else clears it).

## Examples

**Orphan after the lock was stolen**:
```
$ curl -s 127.0.0.1:11434/lease/9f3c…/v1/chat/completions -d @req.json
HTTP/1.1 423 Locked      (V11: final for opencode and pi; never 503, which opencode retries)
{"error":"rig lease revoked","holder":"nightly-eval pid=81448","held_for_s":5412,"your_lease":"9f3c… (released 2026-09-27T00:26:10Z)"}
```

**Ad-hoc session while nothing holds the lock**: admitted and logged as `decision=admit class=long token_ok=false lease_holder=none`.

## Success Criteria

- [ ] Phase 0 spike result is recorded in this doc, and D1 is confirmed or revised
- [ ] Integration test: two clients, one leased and one not. The unleased request gets a 423 in under 100 ms while the lease is held
- [ ] Integration test: a leased client's lock is stolen, and its next request gets a 423 (the revocation path)
- [ ] Streaming parity: a proxied SSE chat gives byte-identical output to a direct call, and client cancel reaches ollama (the slot frees)
- [ ] Cross-implementation token test (shell ↔ Go ↔ `daneel_rig.ail`)
- [ ] 7-night soak meets metrics 1–3. `ailang rig ledger` produces metric 4
- [ ] CHANGELOG, `docs/docs/guides/evaluation/local-ollama.md`, and the local-ollama-eval skill runbook are updated

## Testing Strategy

Unit: the policy truth table (class × lease state × token × model loaded). Integration: a fake ollama on a
random port behind a real gateway (the existing fake-server pattern in the executor tests). The shell half is
run under `/bin/bash` (3.2), because the rig has no bash 4. Mutation-check each test against its fix.

## Verification Log

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | pi 0.85 default HTTP idle timeout is 300 s | Read `…/pi-coding-agent/dist/core/http-dispatcher` source map: `DEFAULT_HTTP_IDLE_TIMEOUT_MS = 300_000`. Setting resolved via `SettingsManager.create().getHttpIdleTimeoutMs()` → 1800000 after the change | Confirmed |
| V2 | opencode runs in its own process group | `ps -o pid,ppid,pgid` on 13208 and 41013: PGID equals PID, PPID is 1 | Confirmed |
| V3 | The old watchdog killed only the chunk's pgroup | `rig-watchdog.sh` before 41f2d7640: `kill -TERM "-$pgid"` | Confirmed; fixed in 41f2d7640 |
| V4 | Orphan birth times match the wedge-kills | etime at 18:11 gives starts of ≈00:21 and ≈02:53. The watchdog log has kills at 00:26:10 and 03:11:27 | Confirmed (timing correlation) |
| V5 | `embed-warmup` does not take the rig lock | `git grep riglock cmd/ailang/docs.go`: no hit. Started from `scripts/hooks/session_start.sh` | Confirmed |
| V6 | Socket → PID cannot see another OS user's sockets without root | `lsof -a -nP -iTCP -u daneel` as voightkampff returns 0 rows, while daneel runs `ailang serve-api --bind 127.0.0.1 --port 8945` (a listening TCP socket exists) | Confirmed, so D1(b) is refuted |
| V7 | Daneel's endpoint is a literal | `~/dev/daneel/tools/daneel_model.ail:48` `endpoint() -> "http://127.0.0.1:11434/api/generate"` | Confirmed |
| V8 | Daneel already honours the rig lock | `~/dev/daneel/tools/daneel:2828` `rig_lock_try`. The q27 measurement script takes the shared lock | Confirmed |
| V9 | Per-run token injection from the environment | Spike 2026-09-27: a capture server on :18999 recorded each client's request. **opencode 1.15.7** (`{env:SPIKE_LEASE}` in `baseURL`, `apiKey`, `headers`): path `/lease/tokOC123/v1/chat/completions`, `Authorization: Bearer tokOC123`, `X-Rig-Lease: tokOC123`, so all three are templated. **pi 0.85.1** (`${SPIKE_LEASE}` in `models.json`): path `/lease/$%7BSPIKE_LEASE%7D/v1/...` (NOT templated), `Authorization: Bearer tokPI456`, `X-Rig-Lease: tokPI456`. Source agrees: `provider-composer.js` runs `resolveConfigValueOrThrow` on the key and `resolveHeadersOrThrow` on headers; `baseUrl` is used verbatim | **Confirmed**: API key and header work in both; URL path works in opencode only, so D1 = (d) |
| V12 | pi's behaviour when a templated variable is unset | Capture server, `AILANG_RIG_LEASE` unset. With the variable in `apiKey`, pi printed "No API key found" and exited 1. With it in `headers`, pi returned `Failed to resolve provider header "X-Rig-Lease" from environment variable`. `resolve-config-value.js` has no `${VAR:-default}` form. opencode sends an empty value | **Confirmed**: executors must always define the variable; a separate `ollama-rig` provider keeps interactive and mission pi use unaffected |
| V11 | Which rejection status clients treat as final | Capture server returning a fixed status, both clients at their **real retry defaults** (pi `retry.maxRetries` unset = 3), a 40 s window. **503**: opencode retried until killed. **403**: opencode sent 2 requests (title call + main) and exited in 2 s; pi sent 1 and exited at once. **423**: opencode 2 requests, 1 s; pi 1 request, 0 s | **Confirmed**: 423 and 403 are final in both, 503 is not. The gateway rejects with **423** |
| V10 | ollama has no per-request auth, priority or admission hook we could use instead | `ollama serve --help` on the rig (ollama **0.33.2**): the environment knobs are HOST, CONTEXT_LENGTH, KEEP_ALIVE, MAX_LOADED_MODELS, MAX_QUEUE, NUM_PARALLEL, ORIGINS (browser CORS only, not caller identity), GPU_OVERHEAD, KV_CACHE_TYPE, LOAD_TIMEOUT and so on. None identifies or admits a caller | **Confirmed** (re-check when ollama is upgraded) |

## Deferred Decisions (agent latitude)

- Ledger rotation and retention; `ailang rig ledger` output format.
- Whether `short` includes small generations (for example a prompt under 2k tokens) as a separate interleaved lane. Only decide this once the ledger shows real contention there.

## Non-Goals

- Changing `OLLAMA_MAX_LOADED_MODELS` or `NUM_PARALLEL` (both are measured decisions).
- Fairness or scheduling between lease holders. M-RIG-LOCK-YIELD already owns that.
- Remote (non-loopback) access to the rig GPU.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| The proxy buffers streams and breaks tool-calling | Streaming-parity test in Success Criteria. Flush per chunk |
| The gateway becomes a single point of failure | rig-watchdog kickstart (same pattern as ollama). Fail loud, no fallback (D4) |
| A harness can't inject a per-run base URL | Phase 0 gate. Header fallback per harness (D1c) |
| Token file readable by the wrong users | Group `rig`, mode 0640, in the existing setgid `/Users/Shared/ailang` |
| Cross-repo drift (Daneel) | The token wire format goes in the shared cross-implementation test |

## Related Documents

- [M-RIG-LOCK-YIELD](../implemented/v0_38_0/m-rig-lock-yield.md): cooperative handoff between holders; this doc adds enforcement underneath it
- [M-EVAL-RIG-RELIABILITY](../implemented/v0_29_0/m-eval-rig-reliability.md)
- [M-EVAL-OS-CONTINUOUS-ROTATION](../implemented/v0_26_0/m-eval-os-continuous-rotation.md)
- [M-NIGHTLY-FLAKE-GUARD](../implemented/v1_0_0/m-nightly-flake-guard.md): its INVALID/`infra_outage` classification is metric 3's instrument
- Commit 41f2d7640: the symptom-level fixes this doc supersedes in part

## References

- 2026-09-27 rig audit (this session): `/tmp/ollama-serve-launchd.log`, `/tmp/ailang-rig-watchdog.log`,
  `/tmp/ailang-nightly-eval.log`, `/tmp/ailang-os-filler.log`, `~/.ailang/state/nightly-eval-history.jsonl`
