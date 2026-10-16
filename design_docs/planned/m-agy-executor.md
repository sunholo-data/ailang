# M-AGY-EXECUTOR: Google Antigravity CLI (`agy`) as a local-editing executor

**Status**: Planned. Phase 0 (discovery spike) gates everything after it.
**Target**: v0.55.x
**Priority**: P1
**Estimated**: Phase 0 ½ day (needs Mark for sign-in); Phases 1–2 ~2 days; Phase 3 is a decision, not work
**Dependencies**: none in code. Phase 0 needs one interactive Google sign-in by Mark.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | An LLM executor like every other one; no change to language semantics |
| A2: Replayability | +1 | stream-json events are banked per run (fixture-replayable parser), like codex/pi |
| A3: Effect Legibility | 0 | Executor edits files in its workspace, same contract as claude/codex |
| A4: Explicit Authority | +1 | Credentials are granted by name in `executorCredentialGrants`, and the CLI gets an isolated `HOME` per auth route instead of ambient `~/.gemini` |
| A5: Bounded Verification | 0 | No effect |
| A6: Safe Concurrency | 0 | One subprocess per task, process-group kill on timeout |
| A7: Machines First | +1 | Adds a Google lane to the fleet that edits locally; today Google only reviews (Vertex) or edits via patch round trip (`gemini_agents`) |
| A8: Minimal Syntax | 0 | No language change |
| A9: Cost Visibility | +1 | Every run banks `cost_provenance` (subscription vs metered), the executor version, and `usage` from the json envelope |
| A10: Composability | +1 | Fits the existing executor contract; coordinator and eval harness pick it up with no dispatch changes |
| A11: Structured Failure | +1 | A soft-denied tool exits 0 with `status:"ERROR"`; the executor maps `status`, not the exit code, to failure |
| A12: System Boundary | 0 | One more vendor CLI, pinned and checksum-verified |

**Net Score: +6** → **Decision: proceed, gated on Phase 0**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism beyond the LLM itself, which every executor has
- [x] A3 (Effects): edits stay in the task workspace; `allowNonWorkspaceAccess` stays false
- [x] A4 (Authority): no ambient credentials. The API-key route gets exactly one granted variable, and the subscription route gets one isolated `HOME`
- [x] A7 (Machines First): the motivation is fleet capacity, not convenience

## Problem Statement

Google is our only frontier vendor without a local, flat-rate, file-editing lane:

| Vendor | Local editing lane | Billing |
|---|---|---|
| Anthropic | `claude` CLI | subscription (OAuth) |
| OpenAI | `codex` CLI | subscription (OAuth) |
| Google | none. Vertex `managed_agents` can't export edits; `gemini_agents` (ffef3402a) exports a patch from a remote sandbox | metered, API key or ADC |

The Gemini CLI was retired in v0.22.0. Google's replacement for Pro/Ultra plans is the Antigravity CLI
(`agy`, google-antigravity/antigravity-cli, v1.3.4 released 2026-10-13).

**Current state (verified 2026-10-13, see Verification Log):**
- `agy` runs headless, edits files in its working directory, and emits NDJSON (`--output-format stream-json`).
- It authenticates by Google sign-in, which uses Google AI Pro/Ultra subscription quota, or by a Gemini API key.
- Mark may or may not have Google AI Pro. Nobody has checked yet.

**Impact:** if the subscription route works, the fleet gains a third flat-rate lane. That's the same economics that made codex
the planner/executor head (`roles:` in models.yml). If it doesn't, an API-key `agy` lane is still a local-editing
Google lane, without `gemini_agents`' upload/download round trip.

## Goals

**Primary Goal:** an `agy` executor that passes the agent smoke tier and can serve as a mission/coordinator rung,
on subscription billing if Mark's account allows it.

**Success Metrics:**
- `ailang eval-suite --agent --models <agy row> --tier smoke` passes the gate in model-manager §5.
- Every banked run carries `executor_version`, `cost_provenance` and the model `agy` reports in its `init` event.
- No run can hang past `Task.Timeout`. The process group is killed and the run is banked as a timeout.
- Installing it never edits shell rc files and never runs an unverified binary.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1. Default auth route: subscription OAuth vs API key | Subscription = flat rate and the fleet use case. API key = metered against the $100 cap. They need different HOME layouts and env grants | **human (Mark)**, after Phase 0 shows what his account has | design | med |
| D2. Subscription credentials under launchd | `agy` caches the sign-in in the OS keyring. Mission loops run under launchd, where we have already lost a day to keychain shadowing with claude | **human** | before Phase 3 | high |
| D3. Permission posture: `--dangerously-skip-permissions` vs `--mode accept-edits` plus settings `toolPermission` | Decides whether shell tools run unprompted. Too strict = 0-tool-call runs (the claude cloud lesson); too loose = whatever the CLI allows | agent proposes from Phase 0 data, **human ratifies** | design | med |
| D4. Version pinning and auto-update | `agy update` exists. An unpinned CLI silently changes the harness under a longitudinal series (the pi pin lesson) | agent | compile | low |
| D5. Default model per row | `gemini-4-argon` is rejected on the API-key route. The subscription route may list different models | agent, from `agy models` output in Phase 0 | compile | low |
| D6. Fleet placement (role-chain rung) | Changes who runs mission work | **human** | after smoke and core gate | med |

### Design Freeze

- [ ] D1: Mark signs in once (Phase 0) and says which route is the default
- [ ] D2: launchd credential strategy chosen (or "agy is attended/coordinator-only, never a launchd loop rung")
- [ ] D3: permission posture ratified from the Phase 0 tool-denial measurements

## Solution Design

### Overview

A standard CLI-subprocess executor (`docs/internal/EXECUTOR_SHAPE.md` Pillar 1), modelled on `codex`.
It spawns `agy -p` with stream-json output in the task workspace, parses the NDJSON, maps `status` to success,
and kills the process group at `Task.Timeout`. The install path is a repo script that reproduces Google's installer
checksum check without the installer's side effects.

### Architecture

**Components:**
1. **Installer, `tools/agy/install.sh`.** It replays the official installer's verify chain (manifest → `url` + `sha512` →
   download → `shasum -a 512` must match → extract), pinned to a version, into `~/.local/bin/agy`. It **skips** the
   installer's final `agy install` step, which is the one that appends PATH lines to `~/.zshrc` and `~/.zprofile`.
   This is exactly what was done by hand on 2026-10-13 (V3).
2. **Executor, `internal/executor/agy/`.** It builds the arguments:
   `agy --model <m> --mode <D3> --output-format stream-json --print-timeout <Task.Timeout> -p <directive>`, with
   stdin from `/dev/null`, `cmd.Dir = Task.Workspace`, and `Setpgid` plus a group-kill cancel (the motoko precedent, V7).
   A hard external deadline applies even though `--print-timeout` exists, because of the hang history (V5).
3. **Auth isolation.** Each route gets its own `HOME`:
   - API key: `HOME=$AILANG_STATE/agy-home-apikey`, with `settings.json` `{"modelProvider":"gemini"}`, and
     `GEMINI_API_KEY` set **in the child only** from `GEMINI_AGENTS_API_KEY`. That needs an `agy` case in
     `executorCredentialGrants`; today an unknown executor gets no credentials at all (V6).
   - Subscription: `HOME` = the operator's real home, or a dedicated one Mark signs into, per D2. No env credentials
     are granted.
4. **Stream parser.** It parses the `init` event (records the model the client reports), the tool and edit events,
   and the final envelope (`status`, `usage`). Unknown events go to `ProviderData`, per the contract.
5. **Registry rows.** `agy-<model>` rows with `agent_cli: "agy"`. Pricing is `cost_provenance: subscription` with $0
   marginal on the OAuth route, or Gemini list rates on the API-key route.

### Implementation Plan

**Phase 0: discovery spike** (~½ day; **needs Mark for one step**) — every row marked PENDING below is closed here
- [ ] Mark runs `agy` once in a terminal and signs in. Then `agy models` answers "does Mark have Pro?": a
      subscription-only model list or quota banner means yes.
- [ ] Capture stream-json fixtures for: a trivial edit, a shell-tool call, a soft-denied tool, an unknown model,
      and a `--print-timeout` expiry. They go into `internal/executor/agy/testdata/`.
- [ ] Measure the non-TTY hang under launchd (P3): one `launchctl`-spawned run against one terminal-spawned run.
- [ ] Find how subscription quota is observable (P4): a usage field, an error string at exhaustion, or nothing. This
      decides whether the mission admission gate can see an empty bucket. The codex 1-token-probe trap (memory)
      says assume nothing.
- [ ] Find whether `agy` auto-updates on start (P5), and how to disable it.

**Phase 1: local executor** (~1½ days)
- [ ] `tools/agy/install.sh` (pinned, verified, no rc edits) and a `make install-agy` target
- [ ] `internal/executor/agy/{agy.go,args.go,events.go,README.md}` plus fixture-replay, mock-binary, gated-live and
      HealthCheck tests (EXECUTOR_SHAPE testing checklist)
- [ ] `HealthCheck` uses `executor.VersionProbe` and asserts the pinned version (V8)
- [ ] An `envExecAgy` case in `executorCredentialGrants`
- [ ] A blank import in `internal/coordinator/provider_executor.go`, and `"agy"` in `exec.go` validProviders

**Phase 2: registry and gate** (~½ day)
- [ ] One opt-in row per route that Phase 0 shows works. In no suite.
- [ ] Agent smoke tier, then core+frontier for placement (model-manager §5/§5.5)

**Phase 3: fleet placement** — a D6 decision for Mark, with the gate numbers in hand. Not pre-decided here.

### Files to Modify/Create

**New files:**
- `tools/agy/install.sh` - pinned, checksum-verified install with no shell-rc edits, ~80 LOC
- `internal/executor/agy/agy.go` - executor, Register/init, process-group kill, ~300 LOC
- `internal/executor/agy/args.go` - argument and HOME construction per auth route, ~120 LOC
- `internal/executor/agy/events.go` - stream-json parser, ~200 LOC
- `internal/executor/agy/agy_test.go` - fixture, mock-binary, HealthCheck and gated live tests, ~350 LOC
- `internal/executor/agy/README.md` - flags, auth routes, event schema, limits

**Modified files:**
- `internal/executor/envpolicy.go` - `envExecAgy` plus its credential grant, ~10 LOC
- `internal/coordinator/provider_executor.go` - one blank import
- `cmd/ailang/exec.go` - `"agy"` in validProviders, 1 LOC
- `internal/modelreg/models.yml` - opt-in rows after Phase 0
- `Makefile` - `install-agy` target

## Examples

### Example 1: attended one-off

```bash
make install-agy                                  # verified, pinned, no rc edits
ailang exec agy --workspace . "fix the failing test in internal/foo"
```

### Example 2: eval gate (after Phase 2)

```bash
SMOKE=$(grep -l 'tier: smoke' benchmarks/*.yml | xargs -n1 basename | sed 's/\.yml$//' | paste -sd, -)
ailang eval-suite --agent --models agy-gemini-3-1-pro --benchmarks "$SMOKE" --langs ailang --output /tmp/smoke_agy
```

## Success Criteria

- [ ] `make install-agy` on a clean machine installs the pinned version, leaves `~/.zshrc` and `~/.zprofile`
      byte-identical (checked with a diff in the test), and refuses on a checksum mismatch (mutation test with a corrupted payload)
- [ ] A fixture replay of all five Phase 0 fixtures produces the right Result: success, tool count, usage, model, `status` mapping
- [ ] A mock binary that never exits is killed at `Task.Timeout`, along with its children (process-group test)
- [ ] A soft-denied tool, with `status:"ERROR"` and exit 0, banks as **failure**
- [ ] The API-key route: the child env contains `GEMINI_API_KEY` and no other credential; the parent env never gains it
- [ ] The agent smoke tier passes the model-manager gate on at least one row
- [ ] All tests passing, `make lint` clean, README and changelog fragment written

## Testing Strategy

**Unit tests:** argument building per route, the parser over the Phase 0 fixtures, status mapping, the credential grant.
**Integration tests:** a POSIX mock `agy` covering streaming, hangs (process-group kill) and soft denial.
**Manual / gated live:** `AILANG_AGY_LIVE=1` runs a one-line edit in a temp git repo; the launchd run is the P3 measurement.

## Verification Log

| # | Claim | Evidence | Status |
|---|---|---|---|
| V1 | `agy` v1.3.4 has `-p/--print`, `--output-format text\|json\|stream-json`, `--mode accept-edits\|plan`, `--dangerously-skip-permissions`, `--print-timeout` (default 0s), `--sandbox`, `--model`, `--effort`, and subcommands `models`, `update`, `install` | `agy --help` output, 2026-10-13, local binary | Confirmed |
| V2 | Headless runs edit local files and do not hang on macOS (stdin `/dev/null`, output redirected) | subagent live run, 2026-10-13: `write_to_file` created `hello.txt` in accept-edits mode, 2–3 s per run | Confirmed, macOS terminal only |
| V3 | The official installer verifies SHA512 against a manifest, then runs `agy install`, which edits shell rc files | read `install.sh` (242 lines): manifest at `…run.app/manifests/darwin_arm64.json`, check at L191-197, `agy install` at L236-239. Payload `storage.googleapis.com/antigravity-public/…/cli_mac_arm64.tar.gz` re-verified by hand: SHA512 matches the manifest | Confirmed |
| V4 | The API-key route needs settings `{"modelProvider":"gemini"}` plus `GEMINI_API_KEY`; the env var alone does nothing. That route lists only Gemini 3.8/3.7/3.6 Flash and 3.1 Pro, and rejects `gemini-4-argon` | subagent live run with an isolated HOME, 2026-10-13 | Confirmed |
| V5 | Non-TTY hang #318 was closed as obsolete on 2026-09-03, then re-reported on v1.2.0/1.2.3 (Linux, piped) | github.com/google-antigravity/antigravity-cli/issues/318 | Confirmed (history); 1.3.4 on Linux not measured |
| V6 | An executor name with no `executorCredentialGrants` case gets **no** credential variables | `internal/executor/envpolicy.go` `executorCredentialGrants`: switch falls through to `return nil` | Confirmed |
| V7 | A process-group kill precedent exists | `internal/executor/motoko/motoko.go:367` (Setpgid plus group-kill Cancel) | Confirmed |
| V8 | A shared version probe exists for asserting the pinned CLI version | `internal/executor/version.go` `VersionProbe.Raw/Identity` | Confirmed |
| V9 | Without sign-in, `agy models` prints "Please sign in…" and exits 0 | local run 2026-10-13 under a 20 s alarm | Confirmed. Also an exit-0-on-error case the parser must not trust |
| P1 | Mark has Google AI Pro/Ultra | No consumer AI Pro: agy refuses m@sunholo.com on the consumer route ("doesn't use Workspace, needs Gmail"). Billing export (2026-10-14) shows two live one-seat subscriptions on the shared billing account: Gemini Enterprise Standard `sunholo-dev` (~$35/mo) and Gemini Code Assist `mark-gemini` (~$22.80/mo), both since late July. agy's Cloud-project route against `sunholo-gemini-enterprise` reports "no license available" | **PENDING** (Phase 0; see log below) |
| P2 | Subscription quota is 5-hour rolling plus weekly, with AI-credit overage | forum reports only (discuss.ai.google.dev/t/146716; antigravity.google/docs/cli/credits) | **PENDING** (Phase 0) |
| P3 | Headless runs do not hang under launchd | not measured | **PENDING** (Phase 0) |
| P4 | Quota exhaustion is observable (a field or error) before or at the failing run | not measured | **PENDING** (Phase 0) |
| P5 | Whether `agy` auto-updates itself on start | `agy update` exists; startup behaviour not measured | **PENDING** (Phase 0) |
| P6 | Under launchd the subscription keyring credential is readable without a GUI login | not measured. Prior: claude's keychain shadowing (memory: blanked-keychain incident) | **PENDING** (D2) |

**Phase 0 sign-in log (attended, 2026-10-14):**
- agy consumer route: rejected for m@sunholo.com (Workspace account; requires a Gmail account).
- agy Cloud-project route: license auto-discovery found nothing; manual project `sunholo-gemini-enterprise` → "no license available".
- The agy binary's project route fetches Gemini Code Assist licences (`aicode`/`businessaicode` FetchLicenses; `UNLICENSED_TIER` state) and calls `aiplatform` heavily.
- Mark (manually, in the console) added IAM on `sunholo-gemini-enterprise` for m@sunholo.com: `roles/cloudaicompanion.user` (Gemini for Google Cloud User) and `roles/serviceusage.serviceUsageConsumer`. Verified by `get-iam-policy`; m@sunholo.com also holds `roles/owner` and `roles/discoveryengine.user` there. Subscription still not working at time of writing.
- ROOT CAUSE FOUND: `mark-gemini` used **automatic** licence assignment, which reclaims a seat from a user idle for ~a month — m@sunholo.com had been silently unassigned. Mark switched the subscription to **manual** assignment and assigned m@sunholo.com. Trap for the fleet: under automatic assignment an idle month silently de-licenses the lane; keep it manual.
- Gemini Enterprise licence check (discoveryengine v1alpha, read-only): `sunholo-gemini-enterprise` global licenseConfig = 1 seat, SUBSCRIPTION_TIER_ENTERPRISE, geminiBundle, ACTIVE 2026-10-07→11-07 autoRenew; userLicenses: m@sunholo.com ASSIGNED (last login 2026-10-14). The eu config is an EXPIRED free trial.
- Mark enabled `businessaicode.googleapis.com` in sunholo-gemini-enterprise. agy still reports "No licenses found" (2 more attempts, 16:05 and 16:07). agy's FetchLicenses goes to `aicode.googleapis.com` (proto google.gca.aicode.v1main), a Google-managed endpoint that is not enableable per project, so the licence decision is server-side. agy logs "admin controls not applicable" and `quotaProject=` empty.
- With `aiplatform.googleapis.com` enabled, agy completes "GCP setup" but logs `Selecting license (Tier: , …)` — an EMPTY tier, i.e. no licence attached; inference would bill as metered Vertex in sunholo-gemini-enterprise (no hard cap). Conclusion (Mark, 2026-10-14): agy does not see the paid seat; the subscription route is dead for now.
- Later the same day agy DID answer: it calls `aiplatform.googleapis.com/v1/projects/sunholo-gemini-enterprise/locations/global/publishers/google/models/gemini-4-argon:streamGenerateContent` (default "Gemini 4 Argon (High)", plus gemini-3.5-flash-lite for side calls) with the user's OAuth token, still with `Tier: ` empty — i.e. METERED Vertex, not the licence. Billing export for 2026-10-14 should confirm (Argon SKUs under sunholo-gemini-enterprise). Also: from the agent's own shell, `agy models` reports "Please sign in" — the interactive sign-in is in a store a non-interactive process can't read (evidence for P6/D2). Mark logged out; licence lookup still empty; will retry the licence route 2026-10-15.
- OUTCOME: Mark is cancelling auto-renew on the `mark-gemini` Code Assist subscription (unusable by agy). Gemini Enterprise `sunholo-dev` is kept (the GE executor MCP uses it). Open question for Google support: does GE Standard include Antigravity CLI? If agy is wanted before then, the only sanctioned route is the API key on locked `ailang-gemini-agents` ($100 cap).
- 2026-10-15: billing export confirms the 10-14 agy runs were METERED Vertex in sunholo-gemini-enterprise, not a licence: Gemini 4 Argon input 11,907 tokens + output 34, Gemini 3.8 Flash input 10,904 + output 18, Flash Lite trivial = $0.065 gross, about half offset by credits (~$0.03 net). No licence SKU moved. The Code Assist subscription still accrues ~$0.74/day until its term ends (auto-renew off). Mark's 10-15 sign-in retry still found no licence. Stop retrying without a Google support answer on whether agy honours a Gemini Enterprise Standard or Code Assist seat for a Workspace account.
- Previously unresolved: whether the `mark-gemini` Code Assist seat is assigned to m@sunholo.com, and whether agy accepts a legacy Code Assist seat or only a Gemini Enterprise entitlement. Fallback: API-key route on the locked `ailang-gemini-agents` key (metered, $100 cap).

## Deferred Decisions

- Parser struct shapes, file split, and how unknown stream-json events are grouped in ProviderData — agent may choose
- The isolated-HOME location under `~/.ailang/state` — agent may choose
- Whether `--effort` maps from the registry's `reasoning_effort` (see memory: only evals may restrict thinking) — agent, following the existing pi pattern

## Non-Goals

- **Cloud deployment (Pillar 2).** A subscription OAuth credential in Cloud Run repeats the claude "3 holders, 1 token" problem. The API-key route can go cloud later through the `gemini_agents` key path. Separate doc.
- **Replacing `gemini_agents`.** It runs in a remote sandbox with no local footprint, which `agy` cannot.
- **Using the undocumented `customModels` setting to reach Argon.** Not attempted until Google documents it.
- **Running Google's installer.** We replay its checks and never pipe it to a shell.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Mark has no Pro subscription, so the API-key route only | Med | That still delivers a local Google editing lane, metered under the $100 cap. Phase 3 then compares cost per solved benchmark, not flat-rate value |
| Hang recurs (#318) | High | Hard external deadline plus process-group kill. `--print-timeout` is only belt and braces |
| Exit 0 on failure (soft denial, sign-in missing) | High | Status comes from the envelope `status` field. The exit code is never trusted (V9, A11) |
| Silent CLI self-update changes the harness | Med | Pinned version asserted in HealthCheck. P5 decides how auto-update gets disabled |
| Subscription quota invisible until exhausted | Med | P4 measurement. Until then, no launchd loop rung (D2/D6) |
| Proprietary CLI, no license file | Low | Local install only, nothing vendored into the repo. Re-check before any image build |

## Quorum

Triggers fired: #1 (design-freeze items D1–D3) and #4 (load-bearing premises about an external CLI we do not control).
Per the skill's guidance, the remaining unknowns (P1–P6) can only be closed by actions outside an authoring session
(a sign-in and live measurements). So this doc is recast as **Phase-0-gated**, with those premises listed as PENDING rows
rather than argued. Re-run quorum after Phase 0 closes them.

## Related Documents

- [design_docs/implemented/v0_6_1/m-exec-multi-executor-support.md](../implemented/v0_6_1/m-exec-multi-executor-support.md) — the multi-executor contract this extends
- [design_docs/planned/v0_49_1/m-executor-env-hardening.md](v0_49_1/m-executor-env-hardening.md) — the credential-grant model `envExecAgy` must join
- [design_docs/planned/m-openai-agents-api-executor.md](m-openai-agents-api-executor.md) — a parallel "vendor agent as executor" design
- `docs/internal/EXECUTOR_SHAPE.md` — Pillar 1 recipe followed here
- `internal/executor/managed_agents/README.md` — the `gemini_agents` sibling (remote sandbox, patch round trip)

## References

- [Design Axioms](/docs/references/axioms)
- https://antigravity.google/docs/cli/headless · https://antigravity.google/docs/cli/install · https://antigravity.google/docs/settings?tab=cli
- https://github.com/google-antigravity/antigravity-cli/issues/318 · https://github.com/google-antigravity/antigravity-cli/releases/tag/1.3.4
- https://www.antigravity.google/docs/cli/credits

## Future Work

- Cloud Pillar 2 for the API-key route (Secret Manager key, `agent-agy` image)
- Model pinning to Argon once the CLI lists it
- A quota observer for the subscription route, feeding mission admission

---

**Document created**: 2026-10-13
**Last updated**: 2026-10-13
