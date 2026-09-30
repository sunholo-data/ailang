# M-OTLP-EXPORTER-MIGRATION: Retire the Deprecated Cloud Trace Exporter

**Status**: Planned — **M0 spike gates everything else**
**Target**: v0.50.0 (M0–M2); M3–M5 land across subsequent releases, all before 2027-01-01
**Priority**: P1 (a P0 deadline with 15 months of runway, plus present-day stderr damage)
**Estimated**: M0 spike ~1 day; M1 ~0.5 day; M2 ~0.5 day; M3 ~2 days; M4 ~0.5 day; M5 ~0.5 day
**Dependencies**: M0 decision gate. M2 needs a multivac terraform apply. Builds on commit `06cf98181` (typed-nil segfault fix) — that work is **done, not restated here**.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Removes a stderr line whose presence depends on ambient env (`GOOGLE_CLOUD_PROJECT`) and ADC reachability. Today identical commands produce different stderr on different boxes |
| A2: Replayability | 0 | Span content and trace ids unchanged; only the wire protocol to Google changes |
| A3: Effect Legibility | 0 | No effect-system surface |
| A4: Explicit Authority | +1 | Cloud export becomes an explicitly configured endpoint + credential instead of "a project happened to be resolvable" (`internal/config/cloud.go:191-196`) |
| A5: Bounded Verification | 0 | No type/check surface |
| A6: Safe Concurrency | +1 | Removes the motive for the rejected `log.SetOutput(io.Discard)` window (see Rejected Option R1), which would race the goroutine at `sdk.go:256` |
| A7: Machines First | +1 | Clean stderr is the machine-parseable contract; polluted stderr breaks anything parsing `ailang` output |
| A8: Minimal Syntax | 0 | No language syntax |
| A9: Cost Visibility | +1 | Preserves the self-span-loop guard whose absence cost a 24/7 Cloud Run instance (`sdk.go:213-222`), and extends it to the new RPC name |
| A10: Composability | 0 | Exporter lanes stay independent |
| A11: Structured Failure | +1 | Keeps fail-loud: exporter failure surfaces in `InitializationStatus` rather than being suppressed |
| A12: System Boundary | +1 | One exporter protocol (OTLP) at the GCP boundary instead of two SDK families |

**Net Score: +7** → **Decision: Move forward (M0 spike first)**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — removes some
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted — narrows it
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

Net +7 ≥ +2, no −1 on A1/A3/A4/A7 → proceed.

## Problem Statement

`github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace` (`go.mod:15`, v1.38.0, aliased `cloudtrace`) is **deprecated upstream and will be archived after 2027-01-01**. The module's own source says so:

```
cloudtrace.go:18-20  // Deprecated: Google Cloud OpenTelemetry Trace exporter for Go is deprecated
                     // and will be archived after January 1st, 2027.
                     // Please migrate to the OpenTelemetry OTLP exporters.
```

Upstream guidance is the standard OTel OTLP exporters ([MIGRATION.md](https://github.com/GoogleCloudPlatform/opentelemetry-operations-go/blob/main/MIGRATION.md)).

**Current State — the damage is present-day, not only 2027:**

1. **`cloudtrace.New` writes a deprecation notice to stderr on first construction in every process.** `cloudtrace.go:198-199` calls `logDeprecated()`; `cloudtrace.go:40-46` is a `sync.Once`-guarded `log.Println(...)`, and the standard logger's default destination is stderr. Our single call site is `internal/platform/otel/sdk.go:236-239`.

2. **Every `ailang` command on a machine where a trace project resolves has polluted stderr.** The switch is `OTLP_GOOGLE_CLOUD_PROJECT` or `GOOGLE_CLOUD_PROJECT` (`internal/config/cloud.go:179`, `:191-196`) — both are set on this developer box, and `OTLP_GOOGLE_CLOUD_PROJECT` is set on **every** deployed Cloud Run service and job (`terraform/cloud_run.tf:183-185`, `:304-306`, and the 17 job blocks in `terraform/cloud_run_jobs.tf`).

3. **`TestInstallPath_ShimRunsFromAnyCwd` fails.** `cmd/ailang/pkg_bin_test.go:100` asserts the shim's stderr is exactly empty. The assertion is deliberate and load-bearing — the comment at `:86-94` records that a weaker assertion let the 2026-09-18 cwd bug through (the shim worked from `/` and nowhere else; email-parse found it). **Weakening the test is the wrong fix.**

   Measured 2026-09-30 on `b9cd1bd20`:

   ```
   --- FAIL: TestInstallPath_ShimRunsFromAnyCwd (16.60s)
   pkg_bin_test.go:100: shim stderr must be empty: "2026/09/30 21:00:54 Google Cloud
   OpenTelemetry Trace exporter for Go is deprecated and will be archived after January
   1st, 2027. ...\nWarning: stdlib version mismatch: expected dev, found v0.49.0 at
   /Users/mark/dev/sunholo/ailang/std\n"
   ```

   Pre-existing, not introduced by `06cf98181`: `go.mod` pinned the same v1.38.0 at `17db86424`, and that commit's copy of the assertion is byte-identical (`git show 17db86424:cmd/ailang/pkg_bin_test.go`).

4. **⚠️ Correction to the premise: there are TWO independent stderr sources in that test, and Cloud Trace is only one.** Control run, same commit, same test, with the trace switch unset:

   ```
   $ env -u GOOGLE_CLOUD_PROJECT -u OTLP_GOOGLE_CLOUD_PROJECT \
       go test ./cmd/ailang/ -run TestInstallPath_ShimRunsFromAnyCwd -count=1
   --- FAIL: TestInstallPath_ShimRunsFromAnyCwd (10.72s)
   pkg_bin_test.go:100: ... "Warning: stdlib version mismatch: expected dev, found
   v0.49.0 at /Users/mark/dev/sunholo/ailang/std\n"
   ```

   There are indeed two independent stderr sources, but the control run above did
   not isolate them: it unset `GOOGLE_CLOUD_PROJECT` and `OTLP_GOOGLE_CLOUD_PROJECT`
   while leaving `AILANG_STDLIB_PATH` set. Isolating both (2026-09-30, same commit):

   | `AILANG_STDLIB_PATH` | cloud project resolvable | result |
   |---|---|---|
   | unset | unset | **PASS** — stderr clean |
   | set | unset | FAIL — `stdlib version mismatch` only |
   | unset | set | FAIL — deprecation notice only |
   | set | set | FAIL — both lines |

   So **removing Cloud Trace IS sufficient to turn this test green** on any machine
   that does not export `AILANG_STDLIB_PATH` — which includes CI and a plain
   checkout. The `stdlib version mismatch` line is **latent**, not active: it fires
   only when `AILANG_STDLIB_PATH` points at a `std/` whose `VERSION` differs from the
   binary's injected version, and `buildAilang` (`cmd/ailang/main_test.go:386`) builds
   without the ldflags `Makefile:42` injects, so its version is always `"dev"`. That
   is a real fragility — the test fails on a developer box for a reason unrelated to
   what it asserts — but it is a developer-environment sensitivity, not a defect that
   makes the headline criterion unreachable.

5. **Polluted stderr is a correctness hazard**, not cosmetics — anything parsing `ailang` output inherits the noise. Precedent in this repo: the same class of stray warning polluted 291 eval runs' BashExec context in the 2026-06-20 rotation (`internal/loader/stdlib_resolver.go:192-195`).

**Impact if nothing is done:** after 2027-01-01 the module is archived — unmaintained, no security patches, and a dependency the build cannot responsibly carry. The stderr pollution and the red test persist for 15 months in the meantime.

## Goals

**Primary goal:** Remove the deprecated `cloudtrace` dependency, keeping (a) spans reaching Google, (b) `ailang trace list/view` working, (c) the self-span loop dead, and (d) stderr clean.

**Success metrics:**
1. `grep -rn opentelemetry-operations-go/exporter/trace go.mod internal/ cmd/` returns nothing (the three *indirect* `operations-go` modules at `go.mod:64-66` stay — see Non-Goals).
2. `TestInstallPath_ShimRunsFromAnyCwd` passes on a box with a resolvable cloud project (needs M3/M4 alone; it already passes when neither `AILANG_STDLIB_PATH` nor a cloud project is set). M1 additionally makes it insensitive to a developer's `AILANG_STDLIB_PATH`.
3. A span emitted by `ailang` is retrievable by `ailang trace view` after the flip — or, if M0 proves it cannot be, that loss is an explicit, ratified decision rather than a discovery.
4. Self-span rate stays 0/min; no scale-to-zero service is held up.
5. `ailang storage status`-style startup reporting (`InitializationStatus`) still distinguishes "registered" from "delivered".

## What already exists (this is a consolidation, not a greenfield build)

The OTLP path is **live today** in the same file, and the observatory is already OTLP-only.

| Concern | Cloud Trace lane | OTLP lane |
|---|---|---|
| Constructor | `sdk.go:236-239` `cloudtrace.New` | `sdk.go:108` `otlptracehttp.New`, `sdk.go:114` `otlpmetrichttp.New` |
| Enabled by | `OTLP_GOOGLE_CLOUD_PROJECT` / `GOOGLE_CLOUD_PROJECT` (`internal/telemetry/config.go:22`) | `OTEL_EXPORTER_OTLP_ENDPOINT` non-empty (`internal/telemetry/config.go:14`) |
| Endpoint config | project id only; host is implicit in the SDK | `OTEL_EXPORTER_OTLP_ENDPOINT` + per-signal overrides, validated at `sdk.go:164-174` |
| Credentials | ADC, discovered inside a 3s bound (`sdk.go:207-210`, `:249`) | none — plain HTTP to the dashboard |
| Metrics | not used | yes (`sdk.go:114-118`, `:140-144`) |
| Sampler | `AlwaysSample` (`sdk.go:129`) | `ParentBased(AlwaysSample)` (`sdk.go:131`) |
| Status field | `InitializationStatus.CloudTrace` (`sdk.go:37`) | `.OTLPTraces` / `.OTLPMetrics` (`sdk.go:38-39`) |
| Self-span guard | client-side `option.WithTelemetryDisabled()` (`sdk.go:158-159`) **plus** sampler deny-list (`export_self_spans.go:19-22`) | sampler deny-list only |

**Only one production import site exists** — `sdk.go:13`. Verified: `grep -rn "opentelemetry-operations-go\|cloudtrace\." --include=*.go internal/ cmd/` hits only `sdk.go:13,223,231,236,237,238` and the span *name* string at `export_self_spans.go:20`. The blast radius of the swap is one file.

## The decisive downstream finding

**Nothing downstream consumes spans via the Cloud Trace API — with exactly one exception, and it is a reader, not a writer.**

- The dashboard, `/api/observatory/spans`, chains, cost rollups and the OpenRouter **Broadcast** traces are **all OTLP-only**. `OTEL_EXPORTER_OTLP_ENDPOINT` points at the dashboard itself (`terraform/main.tf:157`, `environments/prod/terraform.tfvars:76` → `https://dashboard.ailang.sunholo.com`); `internal/observatory/otlp_receiver.go:171-174` serves `POST /v1/traces`; spans land in Firestore `obs_spans` (`internal/storage/firestore/observatory.go:24`) and `internal/observatory/api.go:47` reads them straight back out. Broadcast posts to the same receiver (`internal/observatory/otlp_receiver.go:730-731`). Cloud Trace is not in that path at any point.
- **There is no OTel Collector anywhere in the infra.** Verified: `grep -rln "otelcol\|opentelemetry-collector"` over all of `ailang-multivac` returns nothing. The AILANG dashboard *is* the collector.
- **The one Cloud Trace reader is `ailang trace list` and `ailang trace view`.** `cmd/ailang/trace.go:12-13` are the only imports of `cloud.google.com/go/trace/apiv1` in the repo; `:119`/`:141` call `ListTraces`, `:258`/`:270` call `GetTrace`. `trace status` reads only env (`:67-93`) and `trace hierarchy` reads the local DB — both unaffected.
- **Those two subcommands only ever worked from a developer's own ADC.** Every Cloud Trace grant in terraform is write-only `roles/cloudtrace.agent` (`terraform/iam.tf:78-80`, `:244`, `:338-340`, `terraform/billing.tf:161-163`, `terraform/docparse.tf:448`); `grep -rn "cloudtrace.user\|cloudtrace.viewer\|cloudtrace.admin"` over all of `ailang-multivac` returns **nothing**.
- Reference load on those two commands: `.claude/skills/trace-debugger/` holds **17** occurrences of `ailang trace list`, **6** of `trace view`, **4** of `trace status` (measured 2026-09-30); repo-wide across `*.md`/`*.sh` there are 69 occurrences of `ailang trace list`. (`cmd/ailang/trace.go:20-25` claims 37/12/9 — either stale or counted over a wider tree; the migration's risk is real either way.)

**Therefore the single question M0 must answer is: do spans ingested via OTLP still appear to the Cloud Trace `ListTraces`/`GetTrace` API?** If yes, the migration is invisible. If no, `ailang trace list/view` must either be repointed at the observatory or retired — a user-visible change needing ratification.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Cloud lane target — `telemetry.googleapis.com` OTLP vs. drop the Google lane entirely and keep only the observatory | Determines whether `ailang trace list/view` survive, and whether M2 terraform work is needed at all | human | design | high |
| D2: OTLP/gRPC vs OTLP/HTTP for the Google lane | MIGRATION.md documents **only** `otlptracegrpc`; gRPC is a new direct dependency (`otlptracegrpc` is absent from `go.mod`/`go.sum` today) and re-opens the instrumented-transport hazard that caused the self-span loop | human | design | high |
| D3: How the Google lane is configured, given `OTEL_EXPORTER_OTLP_ENDPOINT` is **already claimed** by the local/dashboard observatory | A single global endpoint var cannot address both the dashboard and `telemetry.googleapis.com`; the Google lane needs its own explicit setting | human | design | high |
| D4: Whether `InitializationStatus`'s `cloud_trace` JSON key is renamed | It is read by `cmd/ailang/server.go:158` and `cmd/ailang/coordinator_lifecycle.go:71`; the key is in a `json` tag (`sdk.go:37`) so a rename is an output-contract change | human | design | med |
| D5: Fate of `ailang trace list`/`view` if D1/M0 shows OTLP-ingested spans are not `ListTraces`-visible | 69 doc/script references; the trace-debugger skill leans on them | human | design | med |
| D6: Whether the sampler keeps the Cloud-lane `AlwaysSample` vs `ParentBased` asymmetry (`sdk.go:128-132`) | Changes sampled volume and therefore cost; the asymmetry is currently undocumented | agent | compile | low |

### Design Freeze

Every "high" row above must be resolved before M3 starts. **M0 produces the evidence; a human decides.**

- [ ] D1 — cloud lane target (`telemetry.googleapis.com` OTLP, or drop the Google lane)
- [ ] D2 — gRPC vs HTTP transport for the Google lane
- [ ] D3 — configuration surface for the Google lane (new env var name and precedence)
- [ ] D4 — `InitializationStatus` JSON key stability
- [ ] D5 — disposition of `ailang trace list` / `trace view`

**M1 and M2 are deliberately unblocked by the freeze** — M1 is a test-robustness fix and M2 is additive terraform, neither depends on D1–D5. Note M1 is not a prerequisite for anything (rescoped 2026-09-30).

## Solution Design

### Overview

Replace the one `cloudtrace.New` call site with a **second, explicitly configured OTLP exporter** — a separate lane from the existing observatory OTLP lane, because the two have different endpoints, different auth and different credentials. Keep both self-span guards. Keep `InitializationStatus`'s three-way registration report. Delete the dependency last.

### Architecture

**Components:**

1. **Google OTLP lane (new)** — an OTLP trace exporter constructed with an *explicit* endpoint and ADC per-RPC credentials, not from `OTEL_EXPORTER_OTLP_ENDPOINT`. It replaces `cloudExporter` (`sdk.go:205-291`) while keeping that function's two hard-won properties: a bounded credential-discovery deadline, and the unbuffered handoff that disposes a late-arriving exporter instead of leaking it.
2. **Observatory OTLP lane (unchanged)** — `sdk.go:104-119`, env-driven, points at the dashboard.
3. **Self-span guard (extended)** — `exportSelfSpanNames` (`export_self_spans.go:19-22`) gains the OTLP export RPC name (see below). `cloudTraceClientOptions` (`sdk.go:154-160`) is deleted with the dependency in M5, and its intent — *refuse instrumentation at the client* — is re-expressed for whatever transport D2 picks.
4. **Status reporting (shape preserved)** — three registration states, whatever D4 decides about the key name.

### The self-span loop must not come back — and the current guard would not catch it

The loop is a **measured, fixed bug**, recorded at `sdk.go:213-222`: the google-api gRPC transport attaches `otelgrpc.NewClientHandler()` by default, so every `BatchWriteSpans` RPC emitted a span, which the next export shipped, which emitted another. **Measured 2026-09-22: a steady 12 spans/min (the 5s default batch delay), which held a scale-to-zero Cloud Run instance up 24/7 and made dashboards discard 173 of 174 batches as internal noise.**

The trap for this migration is that **the guard is an exact-name match, and the name changes.** `export_self_spans.go:19-22` matches exactly:

- `google.devtools.cloudtrace.v2.TraceService/BatchWriteSpans`
- `google.monitoring.v3.MetricService/CreateTimeSeries`

An OTLP exporter's RPC is `opentelemetry.proto.collector.trace.v1.TraceService/Export`. **Neither entry matches it.** So:

- **In-process sampler**: must gain the OTLP export RPC name (and the metrics equivalent). Without this line, an instrumented OTLP transport restarts the loop with the guard still nominally "in place".
- **Observatory receiver**: already safe — `internal/observatory/otlp_receiver.go:82` denies prefix `opentelemetry.`, which covers the new name. (`:80` denies `google.devtools.cloudtrace`, which becomes dead weight after M5.)
- **Client-side refusal (the actual cure, per `export_self_spans.go:14-18`)**: `otlptracehttp` uses a plain `net/http` client and attaches no OTel instrumentation, so OTLP/HTTP is loop-free by construction. **`otlptracegrpc` with `grpc.WithPerRPCCredentials` (the MIGRATION.md shape) needs explicit verification that no stats handler is attached** — this is the strongest argument for HTTP in D2 and is a named M0 spike item.

### Why the obvious interim mitigation is rejected

**R1 — Rejected: wrap the constructor in `log.SetOutput(io.Discard)`.** Because `logDeprecated` is `sync.Once`-guarded (`cloudtrace.go:40-46`) it only needs suppressing once, so this is a ~4-line fix that would turn the test green today.

**Rejected on CLAUDE.md Critical Principle 2 ("No silent fallbacks — fail loudly").** `log.SetOutput` mutates the **global** standard logger, so every log line from any other goroutine during that window is swallowed. The concurrency is real, not hypothetical: the constructor runs in a goroutine (`sdk.go:256`), and the same package logs from the dispose path at `sdk.go:275` and from `init.go:49` — lines that exist precisely to report degradation. Silencing the logger to hide a deprecation notice would trade a cosmetic problem for a diagnostic blind spot on the exact code path most likely to fail.

Recorded as considered-and-rejected. If the 2027 deadline is ever at risk of being missed, R1 is the emergency lever — but it is a lever, not a plan.

## Conflict Surface

Not strictly required (no parser/typechecker/codegen files), written anyway per the skill's 2026-08-26 lesson — and it surfaced D3 and the sampler-name gap, both of which would otherwise have shipped.

| Position this change extends | What else already lives there | Resolution |
|---|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` as "the OTLP destination" | **Already claimed** by the dashboard/observatory in prod (`terraform/cloud_run.tf:179-181`) and by `http://localhost:1957` on dev boxes | **Override**: the Google lane must get its own explicit setting (D3). Reusing the standard var would silently redirect all observatory traffic to Google and break the dashboard |
| `exportSelfSpanNames` exact-match deny-list | Two GCP RPC names | **Extend** with the OTLP export RPC name — not extending it is a silent regression of a measured bug |
| `InitializationStatus` JSON (`sdk.go:36-41`) | Read by `cmd/ailang/server.go:158`, `cmd/ailang/coordinator_lifecycle.go:71`, `internal/observatory/telemetry_roundtrip_test.go:52` | **Reuse** the shape; D4 decides the key |
| The `Init*` API family (`init.go:19-44`) | 10 `cmd/ailang` call sites, incl. `InitGoogleCloudTrace` used only by `gcp_integration_test.go:36,91` | **Reuse**; `InitGoogleCloudTrace` keeps its name or is renamed in M5 with its two callers |
| `cloudtrace.googleapis.com` enabled in `terraform/main.tf:66` | — | **Extend**: `telemetry.googleapis.com` is required and is **absent from all of `ailang-multivac`** (verified empty grep) |
| GCP IAM for tracing | write-only `roles/cloudtrace.agent` on 5+ service accounts | **Verify**: whether OTLP ingest to `telemetry.googleapis.com` is authorized by that same role is an M0 spike item |

**Programs/commands that MUST still work post-change:**
1. `ailang repl --help` on a box with **no** ADC — must print help and exit 0 (this is the `06cf98181` regression; guarded by `internal/platform/otel/sdk_nilexporter_test.go`).
2. `ailang trace status` — env-only, must be unchanged (`cmd/ailang/trace.go:67-93`).
3. `ailang trace hierarchy` — local DB, must be unchanged.
4. `ailang dashboard spans` / `/api/observatory/spans` — OTLP path, must be unchanged.
5. `ailang server` startup telemetry status line (`cmd/ailang/server.go:158`).

**Deliberate changes:** stderr loses the deprecation line; the cloud lane's configuration surface gains an explicit endpoint setting; `ailang trace list`/`view` behaviour is subject to D5.

## Implementation Plan

### M0 — Discovery spike (**no production code**; gates D1–D5) (~1 day)

- [ ] Send a span to `telemetry.googleapis.com` with ADC and attempt to read it back via `ListTraces`/`GetTrace` — the one question that decides D1 and D5.
- [ ] Determine whether OTLP/**HTTP** works against `telemetry.googleapis.com`, or whether gRPC is mandatory (MIGRATION.md shows only `otlptracegrpc`). Decides D2.
- [ ] Determine required API enablement (`telemetry.googleapis.com`), IAM role, and whether `x-goog-user-project` is needed for quota. Feeds M2.
- [ ] Confirm whether `otlptracegrpc` attaches any OTel stats handler by default (self-span hazard).
- [ ] Confirm what "archived" means for availability — does a 2027-archived module keep resolving from the proxy, or does the build break?
- [ ] Deliverable: a spike note with the five answers and a D1–D5 recommendation. **No sprint proceeds past M2 without it.**

### M1 — Make the stderr assertion hermetic, and assert it directly (independent; NOT a blocker) (~0.5 day)

Rescoped 2026-09-30 after isolating the two stderr sources (see the table in Problem
Statement item 4). M1 is a robustness fix, not a prerequisite: the migration turns
`TestInstallPath_ShimRunsFromAnyCwd` green on its own anywhere `AILANG_STDLIB_PATH`
is unset, which is CI and any plain checkout. What M1 buys is that the test stops
failing on a developer box for a reason unrelated to what it asserts.

- [ ] Fix `buildAilang` (`cmd/ailang/main_test.go:386`) to inject the version ldflags the real build uses (`Makefile:42`), so `version.Version` is not `"dev"` and a set `AILANG_STDLIB_PATH` no longer trips `internal/loader/stdlib_resolver.go:196-199`.
- [ ] Add a direct regression test: a trivial `ailang` invocation with a resolvable cloud project writes **nothing** to stderr. This is the assertion that actually guards the migration; `TestInstallPath_ShimRunsFromAnyCwd` guards cwd behaviour and merely happens to catch this.
- [ ] Verify the 2×2 above holds green in all four cells after M1 + M4.

### M2 — Infra prerequisites (additive, no behaviour change) (~0.5 day)

- [ ] Add `telemetry.googleapis.com` to the enabled services in `terraform/main.tf:55-71` (currently absent from the whole repo).
- [ ] Add whatever IAM M0 identifies; keep `roles/cloudtrace.agent` until M5.
- [ ] Do **not** touch `OTEL_EXPORTER_OTLP_ENDPOINT` — it belongs to the observatory.
- [ ] Verify: `terraform plan` is additive only; dashboard span ingest unaffected.

### M3 — Add the Google OTLP lane alongside Cloud Trace, opt-in (~2 days)

- [ ] Add the new lane in `sdk.go` per D1–D3, reusing `cloudExporter`'s bounded-deadline + dispose-on-late-arrival structure.
- [ ] Extend `exportSelfSpanNames` with the OTLP export RPC name(s).
- [ ] Keep `cloudtrace` registered by default; the new lane is opt-in for one release so both can be compared on live traffic.
- [ ] Verify: dual registration reported; spans arrive by both routes; self-span rate 0/min.

### M4 — Flip the default (~0.5 day)

- [ ] New lane on by default; `cloudtrace` becomes explicitly opt-in.
- [ ] Verify: stderr clean by default; `TestInstallPath_ShimRunsFromAnyCwd` **green** without needing M1 on a box with no `AILANG_STDLIB_PATH`; spans still arrive; self-span rate 0/min.

### M5 — Delete the deprecated dependency (~0.5 day)

- [ ] Remove `sdk.go:13`, `sdk.go:236-239`, `cloudTraceClientOptions` (`sdk.go:154-160`); `go mod tidy` to drop `go.mod:15`.
- [ ] Retire the now-dead `google.devtools.cloudtrace` entries in `export_self_spans.go:20` and `internal/observatory/otlp_receiver.go:80` **only if** nothing else can emit them.
- [ ] Per `.claude/rules/coding-standards.md` ("ALWAYS remove out-of-date tests"): delete `TestCloudTraceClientOptions_DisablesDefaultClientTelemetry` (`export_self_spans_test.go:78`) and rewrite the typed-nil tests against the new constructor — **do not** keep them as compatibility shims.
- [ ] Verify: the grep in Success Metric 1 is empty; `make test-core` and the full `internal/platform/otel` suite pass.

### Files to Modify/Create

**New files:**
- `internal/platform/otel/google_otlp.go` — the Google OTLP lane: endpoint, ADC credentials, bounded construction, ~120 LOC
- `internal/platform/otel/google_otlp_test.go` — construction, timeout, typed-nil and dispose paths, ~150 LOC

**Modified files:**
- `internal/platform/otel/sdk.go` — swap the lane; extend status; delete `cloudtrace` import and `cloudTraceClientOptions`, ~−60/+40 LOC
- `internal/platform/otel/export_self_spans.go` — add the OTLP export RPC name(s), ~+4 LOC
- `internal/platform/otel/init.go` — rename/retarget `InitGoogleCloudTrace` per D4, ~+5 LOC
- `internal/config/telemetry.go` — register the new Google-lane env var (D3), ~+4 LOC
- `cmd/ailang/main_test.go` — inject version ldflags in `buildAilang` (M1), ~+6 LOC
- `cmd/ailang/trace.go` — only if D5 says so
- `go.mod` — drop line 15
- `/Users/mark/dev/sunholo/ailang-multivac/terraform/main.tf` — enable `telemetry.googleapis.com`, ~+1 LOC
- `CHANGELOG.md`, `docs/docs/guides/telemetry.md`, `docs/docs/guides/debugging.md` — per `.claude/rules/coding-standards.md`

## Success Criteria

- [ ] `grep -rn "opentelemetry-operations-go/exporter/trace" go.mod internal/ cmd/` is empty
- [ ] `ailang repl --help` with `OTLP_GOOGLE_CLOUD_PROJECT` set and no ADC: exit 0, **stderr empty**
- [ ] `TestInstallPath_ShimRunsFromAnyCwd` passes (requires M4; M1 only for `AILANG_STDLIB_PATH` boxes)
- [ ] New M1 regression test asserts empty stderr for a cloud-configured invocation
- [ ] A span emitted by `ailang` is visible in its intended destination after the flip, demonstrated not assumed
- [ ] Self-span rate measured at 0/min for ≥1h post-flip; no scale-to-zero service held up
- [ ] `exportSelfSpanNames` contains the OTLP export RPC name
- [ ] All tests passing · Documentation updated · CHANGELOG entry

## Testing Strategy

**Safety net — these must keep passing unchanged, and are the reason this migration is reviewable:**
- `internal/platform/otel/sdk_status_test.go` — 9 tests over registration/degradation/dual/cancellation; `TestInitializationReportsRegisteredNotDelivered` (`:22`) pins the registered≠delivered distinction
- `internal/platform/otel/sdk_recovery_test.go` — 4 tests; `TestOTLPFailedBatchIsBoundedAndLaterBatchRecovers` (`:155`) pins bounded failure
- `internal/platform/otel/export_self_spans_test.go:20,47,61` — the loop regression guard
- `internal/platform/otel/sdk_nilexporter_test.go` — the `06cf98181` segfault guard
- `internal/observatory/telemetry_roundtrip_test.go:52` — end-to-end OTLP round trip

**Must change:**
- `export_self_spans_test.go:78` `TestCloudTraceClientOptions_DisablesDefaultClientTelemetry` — deleted in M5 with the function it pins
- `gcp_integration_test.go:36,91` — the only `InitGoogleCloudTrace` callers; retargeted in M3/M5
- `sdk_nilexporter_test.go` — rewritten against the new constructor (the *hazard* persists: any constructor returning `(*T, error)` can hand back a typed nil)
- `export_self_spans_test.go` — new case for the OTLP export RPC name
- `cmd/ailang/main_test.go` — M1

**Manual / measured:**
- Self-span rate before and after, by the same method as the 2026-09-22 measurement
- Cloud Run instance count on a scale-to-zero service across the flip

## Deferred Decisions

- D6, the sampler `AlwaysSample` vs `ParentBased` asymmetry (`sdk.go:128-132`) — **agent may resolve**, but must document why the asymmetry exists or remove it deliberately
- New env var *name* for D3 (the *existence* of a separate setting is frozen, the spelling is not) — agent may choose, subject to the `cli-doc-maintainer` conventions
- Batch/timeout tuning for the new lane — agent may keep the existing 3s (`sdk.go:108`, `:134`)
- Whether `internal/observatory/otlp_receiver.go:80`'s dead prefix is removed in M5 or left — agent may choose

## Non-Goals

- **Removing the three indirect `operations-go` modules** (`go.mod:64-66`) — `go mod why` shows `exporter/metric` arrives via `cloud.google.com/go/storage`'s client-side metrics, not via our tracing. Not ours to remove and unaffected by the deadline.
- **Deploying an OTel Collector** — there is none today (verified), the dashboard fills that role, and adding one is a much larger infra change than the deadline requires.
- **Changing observatory/Firestore span schema or the Broadcast ingest path** — entirely OTLP already.
- **Reworking `ailang trace hierarchy` or `internal/trace/`'s JSONL store** — no Cloud Trace dependency.
- **Redoing the typed-nil segfault fix** — shipped in `06cf98181`.
- **Weakening `cmd/ailang/pkg_bin_test.go:100`** — the assertion is the point.

## Timeline

**Week 1**: M0 spike + D1–D5 freeze decisions; M1 and M2 land in parallel (neither is blocked by the freeze).
**Week 2**: M3 — new lane alongside, opt-in; live comparison.
**Week 3**: M4 flip; one release of soak.
**Week 4**: M5 removal, docs, CHANGELOG.

**Total: ~5 days of work across ~4 weeks**, with ~14 months of slack before 2027-01-01. The slack is the point: M3's opt-in release and M4's soak are affordable, so neither needs to be skipped.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| OTLP-ingested spans are not `ListTraces`-visible → `ailang trace list/view` break (69 doc/script references) | High | M0 answers this **before** any code; D5 ratifies the outcome; fallback is repointing both subcommands at the observatory, which already holds the spans |
| Self-span loop returns under a gRPC transport with a stats handler | High — cost is invisible in the thing it holds up | Extend the exact-match deny-list in M3; prefer HTTP in D2; measure self-span rate at M3 and M4 |
| The Google lane accidentally reuses `OTEL_EXPORTER_OTLP_ENDPOINT` and redirects all observatory traffic to Google | High — silently breaks the dashboard | D3 freezes a separate setting; Conflict Surface row makes the collision explicit |
| `telemetry.googleapis.com` not enabled / IAM insufficient in prod | Med | M2 lands the terraform additively before the flip; `roles/cloudtrace.agent` retained until M5 |
| M4 declared done on a still-red `TestInstallPath_ShimRunsFromAnyCwd` because the developer's box exports `AILANG_STDLIB_PATH` | Med — looks like the migration failed when it did not | Read the 2×2 in Problem Statement item 4 before concluding; re-run with `env -u AILANG_STDLIB_PATH`. M1 removes the sensitivity |
| `InitializationStatus` JSON key rename breaks a consumer | Low | Three known consumers, all in-repo; D4 decides deliberately |

## Open Questions / Unverified Premises

**These are labelled PENDING because they could not be verified in this session — `gcloud` is not available on this machine. Per the design-doc-creator skill's guidance, phases are gated behind the M0 spike rather than a confident guess being written in.**

| # | Premise | Status |
|---|---|---|
| P1 | Spans ingested via `telemetry.googleapis.com` OTLP are readable by `ListTraces`/`GetTrace` | **PENDING — M0.** Decides whether `ailang trace list/view` survive |
| P2 | OTLP/**HTTP** works against `telemetry.googleapis.com` | **PENDING — M0.** MIGRATION.md documents only `otlptracegrpc` |
| P3 | `telemetry.googleapis.com` must be enabled per project | **PENDING — M0.** Confirmed *absent* from all of `ailang-multivac` today |
| P4 | `roles/cloudtrace.agent` authorizes OTLP ingest, or a different role is needed | **PENDING — M0** |
| P5 | `otlptracegrpc` attaches no OTel stats handler by default | **PENDING — M0.** Self-span hazard |
| P6 | An archived module still resolves from the Go proxy after 2027-01-01 | **PENDING — M0.** Affects urgency, not direction |
| P7 | Whether CI is currently red on `TestInstallPath_ShimRunsFromAnyCwd` | **RESOLVED — no.** The 2×2 in Problem Statement item 4 shows the test PASSES when neither `AILANG_STDLIB_PATH` nor a cloud project is set. CI exports neither: the only two references in `.github/workflows/ci.yml` (lines 148, 152) *unset* it with `env -u AILANG_STDLIB_PATH`. So this is a developer-box-only failure and no CI job is red on it. |

### Quorum disposition

Triggers **1** (design-freeze items) and **4** (load-bearing premises about external systems we do not control) fire, so `ailang design-quorum` would normally run before planning. **Deliberately not run.** P1–P6 can only be closed by a live measurement against a third-party API from a machine with `gcloud`/ADC — the exact case the skill says not to spend a round on ("when the remaining objections can only be closed by an action OUTSIDE the session … stop and hand over. Do not spend the round."). The doc is instead handed over with its gaps labelled and every phase past M2 gated behind M0. **Run the quorum after M0 fills in P1–P6.**

## Verification Log

| # | Claim | How verified (2026-09-30, `b9cd1bd20`) | Result |
|---|---|---|---|
| V1 | Exporter is deprecated, archived after 2027-01-01 | Read `cloudtrace.go:18-20` in the module cache | Confirmed, verbatim |
| V2 | `cloudtrace.New` logs to stderr via `sync.Once` | Read `cloudtrace.go:40-46`, `:198-199` | Confirmed: `log.Println` inside `logDeprecatedOnce.Do` |
| V3 | v1.38.0 is what we use, at one site | `go.mod:15`; `grep -rn opentelemetry-operations-go --include=*.go` | Confirmed: `sdk.go:13` is the only production import |
| V4 | `TestInstallPath_ShimRunsFromAnyCwd` fails on the deprecation line | Ran it | Confirmed, 16.60s, output quoted above |
| V5 | Failure is pre-existing at `17db86424` | `git show 17db86424:go.mod`, `git show 17db86424:cmd/ailang/pkg_bin_test.go` | Confirmed: same v1.38.0, byte-identical assertion |
| V6 | **A second, unrelated stderr source exists** | Re-ran with `env -u GOOGLE_CLOUD_PROJECT -u OTLP_GOOGLE_CLOUD_PROJECT` | **Confirmed — corrects the brief.** Only the stdlib-mismatch line remains |
| V7 | That second source is the harness's missing ldflags | Read `cmd/ailang/main_test.go:386`, `Makefile:42`, `internal/loader/stdlib_resolver.go:196-199` | Confirmed: bare `go build` → `Version="dev"` vs `std/VERSION=v0.49.0` |
| V8 | Cloud Trace export is switched on by `OTLP_GOOGLE_CLOUD_PROJECT`/`GOOGLE_CLOUD_PROJECT` | Read `internal/config/cloud.go:179`, `:191-196`; `internal/telemetry/config.go:22` | Confirmed; both set on this box |
| V9 | The OTLP lane already exists in the same file | Read `sdk.go:104-119`, `:164-174`; `internal/config/telemetry.go:11-14` | Confirmed |
| V10 | **Negative:** `otlptracegrpc` is NOT a dependency | `grep otlptrace go.mod go.sum`; listed the module cache | Confirmed absent — D2 means a new direct dep |
| V11 | Self-span loop numbers (12 spans/min; 173/174 batches; 24/7 instance) | Read the comment at `sdk.go:213-222` | Cited from the code comment's 2026-09-22 measurement; **not independently re-measured** |
| V12 | The sampler guard is exact-match and would miss the OTLP RPC name | Read `export_self_spans.go:19-22`, `:32-40` | Confirmed — map lookup on `p.Name`, no prefix logic |
| V13 | The observatory receiver *would* catch it | Read `internal/observatory/otlp_receiver.go:80-82` | Confirmed: prefix `opentelemetry.` already denied |
| V14 | Only `cmd/ailang/trace.go` reads the Cloud Trace API | Read `cmd/ailang/trace.go:12-13`, `:119`, `:141`, `:258`, `:270` | Confirmed: sole importer of `cloud.google.com/go/trace/apiv1` |
| V15 | **Negative:** no Cloud Trace *reader* IAM in infra | `grep -rn "cloudtrace.user\|cloudtrace.viewer\|cloudtrace.admin"` over `ailang-multivac` | Confirmed empty (exit 1) |
| V16 | **Negative:** no OTel Collector in infra | `grep -rln "otelcol\|opentelemetry-collector"` over `ailang-multivac` | Confirmed empty (exit 1) |
| V17 | **Negative:** `telemetry.googleapis.com` appears nowhere in infra | `grep -rn telemetry.googleapis.com` over `ailang-multivac`; read `terraform/main.tf:55-71` | Confirmed empty; `cloudtrace.googleapis.com` enabled at `:66` |
| V18 | Prod OTLP endpoint is the dashboard, so the var is already claimed | Read `terraform/main.tf:157`, `environments/prod/terraform.tfvars:76`, `terraform/cloud_run.tf:179-185` | Confirmed |
| V19 | All infra Cloud Trace grants are write-only | Read `terraform/iam.tf:78-80`, `:244`, `:338-340`; `billing.tf:161-163`; `docparse.tf:448` | Confirmed: `roles/cloudtrace.agent` only |
| V20 | trace-debugger reference load | `grep -ro` over `.claude/skills/trace-debugger/` | **17** `trace list`, **6** `trace view`, **4** `trace status` — the `trace.go:20-25` comment's 37/12/9 is not reproducible against that path |
| V21 | `exporter/metric` is indirect, via GCS | `go mod why .../exporter/metric` | Confirmed: via `cloud.google.com/go/storage` |
| V22 | Upstream guidance is `otlptracegrpc` + `telemetry.googleapis.com:443` + `grpc.WithPerRPCCredentials` | Fetched MIGRATION.md | Confirmed; `WithAttributeMapping` has no equivalent |
| V23 | `InitializationStatus` has three in-repo consumers | `grep -rn InitWithStatus` | Confirmed: `cmd/ailang/server.go:158`, `cmd/ailang/coordinator_lifecycle.go:71`, `internal/observatory/telemetry_roundtrip_test.go:52` |
| V24 | Duplicate/coverage gate | Neural search at doc creation: top planned 0.40, top implemented 0.36 | Both < 0.45 → proceed normally |

## Related Documents

Neural search returned nothing above 0.40 — no existing doc covers exporter migration. Closest in subject matter, read for context:

- [design_docs/implemented/v0_8_0/m-trace-export-phase2-otel-replay.md](../../implemented/v0_8_0/m-trace-export-phase2-otel-replay.md) (0.36) — the original OTel bring-up; establishes the dual-destination shape this doc collapses
- [design_docs/implemented/v0_29_0/m-cloud-observatory.md](../../implemented/v0_29_0/m-cloud-observatory.md) — the observatory ingest path that makes Cloud Trace non-load-bearing
- [design_docs/implemented/v0_33_1/m-openrouter-broadcast-ingest.md](../../implemented/v0_33_1/m-openrouter-broadcast-ingest.md) — Broadcast traces arrive by OTLP, not Cloud Trace; distinct topic, cited because CLAUDE.md's instrument table points at it
- [design_docs/planned/m-v1-simplification-program.md](../m-v1-simplification-program.md) (0.33) — dependency and env-var surface reduction; this migration removes one direct dependency and one SDK family. Distinct: that program does not address the exporter

## References

- [Design Axioms](/docs/references/axioms) — the 12 non-negotiable principles
- [Upstream MIGRATION.md](https://github.com/GoogleCloudPlatform/opentelemetry-operations-go/blob/main/MIGRATION.md)
- `06cf98181` — typed-nil interface segfault fix (prerequisite, already landed)
- `docs/docs/guides/telemetry.md`, `docs/docs/guides/debugging.md`
- CLAUDE.md Critical Principle 2 (no silent fallbacks) — the grounds for rejecting R1
- `.claude/rules/coding-standards.md` — "ALWAYS remove out-of-date tests"; governs M5

## Future Work

- Retire `internal/observatory/otlp_receiver.go:80`'s `google.devtools.cloudtrace` deny prefix once nothing can emit it
- Reconsider whether the Google lane is needed at all: everything user-facing already reads the observatory, so the cloud lane may be pure cost (a question for after D5)
- Revisit `AILANG_OTLP_INGEST_TOKEN`, which is unset in prod — the OTLP receiver runs with ingest auth disabled. **Out of scope here; worth its own doc**

---

**Document created**: 2026-09-30
**Last updated**: 2026-09-30
