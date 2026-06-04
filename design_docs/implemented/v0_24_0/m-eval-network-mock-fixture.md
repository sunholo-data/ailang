# M-EVAL-NETWORK-MOCK-FIXTURE: Deterministic local HTTP mock for network benchmarks

**Status**: Implemented
**Target**: v0.24.0
**Priority**: P1 (High — a non-deterministic benchmark corrupts every eval run and the public leaderboard)
**Estimated**: 1 day
**Dependencies**: None (extends existing `internal/eval_harness/agent_prompt.go` templating)

> **📊 EMPIRICALLY VERIFIED (2026-06-04):** In the v0.23.0 standard run, `api_call_json`
> failed for 5 of 7 models with HTTP `503` and passed for 2 — with **identical correct code**.
> The benchmark hits the live `https://httpbin.org/post`, which returns `503` intermittently.
> This single flaky benchmark inflated GLM-5.1's leaderboard rank and depressed every frontier
> model's. Fixing it makes the benchmark deterministic, offline, and nightly-safe.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | The entire point — replaces a non-deterministic external network call with a deterministic local mock. Removes the only source of run-to-run variance in the core benchmark suite. |
| A2: Replayability | +1 | Eval runs become reproducible offline; the same benchmark produces the same result regardless of httpbin/network state. |
| A3: Effect Legibility | 0 | No change to AILANG effect semantics; the benchmark still declares `caps: [Net, IO]` and exercises the Net effect against a real (local) socket. |
| A4: Explicit Authority | 0 | No capability changes; `Net` is still required and exercised. |
| A5: Bounded Verification | 0 | No verification change. |
| A6: Safe Concurrency | +1 | Each parallel benchmark run gets its own ephemeral-port mock server, so `--parallel N` runs cannot interfere — a concurrency-correctness improvement over a shared external endpoint with rate limits. |
| A7: Machines First | +1 | A flaky benchmark teaches the model (and us) the wrong signal; a deterministic one gives a clean capability measurement. |
| A8: Minimal Syntax | 0 | No language syntax change. |
| A9: Cost Visibility | 0 | No resource-cost change. |
| A10: Composability | +1 | The `{{MOCK_HTTP_URL}}` fixture is a general mechanism reusable by any future network benchmark. |
| A11: Structured Failure | 0 | No error-handling change. |
| A12: System Boundary | +1 | Moves the eval's trust boundary inward: the harness no longer depends on a third-party service's uptime to produce a verdict. |

### Hard Violation Check

- A1 Determinism: **+2** (no violation — strongly positive)
- A3 Effect Legibility: **0** (no violation)
- A4 Explicit Authority: **0** (no violation)
- A7 Machines First: **+1** (no violation)

No hard violations.

### Decision Thresholds

**Net Score: +8** → **Decision: Proceed** (well above +2; no −1 on A1/A3/A4/A7).

---

## Problem Statement

`benchmarks/api_call_json.yml` is the **only** one of the 32 core benchmarks that makes a
real external network call. Its task prompt hardcodes:

```
1. Makes an HTTP POST request to https://httpbin.org/post
...
4. Prints ONLY the response status code (e.g., "200" or "201")
```

and it grades on `expected_stdout: "200"`.

**The failure mode (verified 2026-06-04, v0.23.0 standard run):**

| Model | `api_call_json` stdout | Verdict |
|-------|------------------------|---------|
| claude-opus-4-8 | `503` | ❌ FAIL |
| claude-sonnet-4-6 | `503` | ❌ FAIL |
| gemini-3-1-pro | `503` | ❌ FAIL |
| gpt5-5 | `503` | ❌ FAIL |
| or-minimax-m3 | `503` | ❌ FAIL |
| or-glm-5 | `200` | ✅ PASS |
| or-glm-5-1 | `200` | ✅ PASS |

The two passing models and the five failing models wrote **functionally identical** AILANG
(`httpRequest("POST", "https://httpbin.org/post", headers, body)` → print `resp.status`). The
only difference was whether `httpbin.org` happened to be healthy at request time. httpbin
returned `503` to five of the seven requests.

**Impact:**
1. **Leaderboard corruption.** This one benchmark handed GLM-5.1 a point it did not earn and
   docked every frontier model a point it did not deserve. The raw v0.23.0 AILANG result —
   "GLM-5.1 91% beats Opus 84%" — is an artifact. Excluding `api_call_json`, GLM-5.1 and
   gemini-3-1-pro tie at ~90% and Opus/GPT-5.5 sit at ~87% (a 3-point spread, not 7).
2. **Every run is contaminated**, including the nightly regression guard and any published
   baseline. A nightly can show a spurious "regression" purely because httpbin had a bad minute.
3. **It violates AILANG's foundational determinism principle.** An eval harness for a language
   whose thesis is reproducibility must not produce non-reproducible verdicts.

This is a **harness-infrastructure** problem, not a language gap. The benchmark's *intent* —
"can the model write a correct HTTP POST with custom headers and a JSON body, and read back the
status?" — is sound. Only its *grading dependency on a live third party* is broken.

---

## Goals

**Primary goal:** Make `api_call_json` (and any future network benchmark) deterministic and
offline by routing its HTTP call to a harness-managed local mock server.

**Success metrics:**
- `api_call_json` produces the **same verdict on every run**, with no network access, for all
  seven models above (and for python/js/go).
- No benchmark prompt references an external URL after this ships (`grep -r httpbin benchmarks/`
  returns nothing).
- The mock works under `--parallel N` with zero cross-run interference.
- The mock survives a benchmark hard-timeout without leaking a goroutine/socket.

---

## High-Impact Decisions

| Decision | Choice | Who decides | Change cost if revisited |
|----------|--------|-------------|--------------------------|
| Mock vs reliable external endpoint (incl. our own Cloud Run) | **Local mock** | Owner (decided 2026-06-04) | Low — URL substitution is the only coupling |
| Per-run server vs shared server | **Per-run, ephemeral `:0` port** | Author | Medium — shared server reintroduces concurrency/port coupling |
| URL injection mechanism | **`{{MOCK_HTTP_URL}}` prompt token** (reuses existing `strings.ReplaceAll` templating) | Author | Low |
| Mock response fidelity | **httpbin-compatible**: `200`, echo posted headers + JSON body in a `{"headers":..., "json":...}` envelope | Author | Low — only `status` is graded today |

### Design Freeze

- [x] Local mock (not external endpoint) — owner-approved 2026-06-04
- [x] Per-run ephemeral-port server (concurrency-safe)
- [x] `{{MOCK_HTTP_URL}}` token reuses existing templating in `agent_prompt.go`
- [ ] Mock-server module location confirmed (`internal/eval_harness/httpmock.go` proposed)
- [ ] httpbin URL fully removed from `api_call_json.yml`

---

## Solution Design

### Overview

When a benchmark's task prompt contains the token `{{MOCK_HTTP_URL}}`, the harness, for **each
run of that benchmark**, starts a local HTTP server on an ephemeral port (`127.0.0.1:0`),
substitutes the live server URL into the prompt in place of the token, runs the benchmark
(generation → execution → repair → execution), and tears the server down when the run finishes
(success, failure, or timeout). The model's generated code calls `http://127.0.0.1:PORT/...`,
which always returns a deterministic `200`.

### Architecture

```
RunBenchmark(spec, config)
  ├─ if spec.TaskPrompt contains "{{MOCK_HTTP_URL}}":
  │     mock := httpmock.Start()         // net/http/httptest.Server on 127.0.0.1:0
  │     defer mock.Close()               // teardown on ALL exit paths
  │     prompt = ReplaceAll(prompt, "{{MOCK_HTTP_URL}}", mock.URL)   // e.g. http://127.0.0.1:53122
  ├─ generate code from prompt
  ├─ execute (may run multiple times across repair attempts — mock stays up)
  └─ score against expected_stdout
```

The mock handler (httpbin-compatible subset):

```go
// POST <any path> -> 200 with an httpbin-shaped JSON echo.
func mockHandler(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    var parsed any
    _ = json.Unmarshal(body, &parsed)
    resp := map[string]any{
        "headers": flattenHeaders(r.Header), // includes X-Test-Header
        "json":    parsed,                   // echoes posted JSON
        "url":     r.URL.String(),
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK) // deterministic 200
    _ = json.NewEncoder(w).Encode(resp)
}
```

### Implementation Plan

**Phase 1 — mock server module (~0.3 day)**
- [ ] Add `internal/eval_harness/httpmock.go`: `StartHTTPMock() *httptest.Server` with the
      httpbin-compatible handler above. Bind `127.0.0.1:0` (ephemeral).
- [ ] Unit test: POST returns 200, echoes `X-Test-Header` and the JSON body.

**Phase 2 — prompt injection + lifecycle (~0.4 day)**
- [ ] In the benchmark-run path, detect `{{MOCK_HTTP_URL}}` in the (already-templated) prompt.
- [ ] Start a mock per run, `ReplaceAll` the token with `mock.URL`, `defer mock.Close()`.
- [ ] Ensure the server is created **after** the existing `{{CAPS}}`/`{{TIMEOUT}}`/`{{PYTHON_VERSION}}`
      substitutions and lives for the whole run (including agent-mode repair iterations).
- [ ] Verify teardown on the hard-timeout path (the run's context cancel must not skip `Close()`).

**Phase 3 — benchmark + verification (~0.3 day)**
- [ ] Edit `benchmarks/api_call_json.yml`: replace `https://httpbin.org/post` with
      `{{MOCK_HTTP_URL}}` in the task prompt. Keep `caps: [Net, IO]` and `expected_stdout: "200"`.
- [ ] Run the 7-model standard set; confirm `api_call_json` now passes deterministically for all
      (modulo genuine codegen errors) and that no run touches the public internet.
- [ ] `grep -r "httpbin\|https://" benchmarks/` returns nothing.

### Files to Modify/Create

| File | Change | LOC |
|------|--------|-----|
| `internal/eval_harness/httpmock.go` (new) | Local httptest mock + handler | +70 |
| `internal/eval_harness/httpmock_test.go` (new) | Mock unit tests | +60 |
| `internal/eval_harness/agent_prompt.go` (or run path) | `{{MOCK_HTTP_URL}}` detection + per-run start/substitute/teardown | +40 |
| `benchmarks/api_call_json.yml` | httpbin URL → `{{MOCK_HTTP_URL}}` | ~1 |

---

## Examples

### Example 1: api_call_json after the fix

Benchmark prompt (excerpt):
```
1. Makes an HTTP POST request to {{MOCK_HTTP_URL}}
2. Includes custom headers: "X-Test-Header: value123" and "Content-Type: application/json"
3. Sends a JSON body: {"message":"Hello from <LANG>","count":42}
4. Prints ONLY the response status code
```

At runtime the harness substitutes `{{MOCK_HTTP_URL}}` → `http://127.0.0.1:53122`. The model's
generated code POSTs there; the mock returns `200`; stdout is `200`; verdict is deterministic.

### Example 2: concurrency safety

Under `--parallel 8`, eight `api_call_json` trials (different models) each get their **own**
ephemeral-port mock. No shared state, no port collision, no rate limit — unlike a single shared
external endpoint.

---

## Success Criteria

- [ ] `api_call_json` passes deterministically for all 7 v0.23.0 models, offline (airplane mode)
- [ ] `grep -rE "httpbin|https?://" benchmarks/` returns no matches
- [ ] `--parallel 8` run shows no port/connection errors on `api_call_json`
- [ ] Mock server is torn down on success, failure, AND hard-timeout (verified by a leak test)
- [ ] All harness tests pass (`go test ./internal/eval_harness/`)
- [ ] CHANGELOG updated; this doc moved to `implemented/v0_24_0/`

## Testing Strategy

- **Unit**: `httpmock_test.go` — POST returns 200, echoes `X-Test-Header` + JSON body; GET also 200.
- **Integration**: run `api_call_json` standard mode with network disabled (e.g. block egress)
  and confirm it still passes — proving zero external dependency.
- **Concurrency**: `--parallel 8` × `api_call_json` × 8 trials, assert all green, no port errors.
- **Leak**: force a benchmark timeout mid-request, assert the mock goroutine/socket is released
  (`-count=20` to catch nondeterministic teardown races).

## Deferred Decisions

- Whether to generalize to non-HTTP fixtures (mock TCP, mock filesystem) — out of scope; revisit
  only if a second network benchmark needs a different protocol.
- Whether the mock should support configurable status codes per-benchmark (`{{MOCK_HTTP_URL_500}}`
  to test error handling) — a clean extension, but not needed for api_call_json. Agent has
  latitude to add a status-code suffix convention if a future benchmark needs it.

## Non-Goals

- Not removing or weakening the `Net` capability requirement — the benchmark still exercises a
  real socket (just a local one).
- Not changing the stdout exact-match grading model (separate concern; see Future Work).
- Not adding a general benchmark "setup/teardown script" mechanism — this is a focused HTTP-mock
  fixture, not a generic shell-hook system (which would be a larger, separate design).

## Timeline

| Day | Work |
|-----|------|
| 0.3 | Phase 1: mock module + unit tests |
| 0.4 | Phase 2: injection + lifecycle + teardown |
| 0.3 | Phase 3: benchmark edit + 7-model verification + grep guard |

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Server leak on hard-timeout | `defer mock.Close()` on the run path; explicit leak test with `-count=20` |
| Port collision under parallelism | Ephemeral `127.0.0.1:0`, one server per run (never shared) |
| Agent-mode repair re-runs code after server closed | Server lifetime spans the **entire** run (all repair iterations), closed only at run end |
| A future benchmark uses a real URL again | CI grep guard: `grep -rE "https?://" benchmarks/` fails the build |

## Conflict Surface

**This touches `internal/eval_harness` (prompt templating + benchmark execution lifecycle).**

1. **What positions does this change extend?** (a) The prompt-templating pass in
   `agent_prompt.go` gains one more token, `{{MOCK_HTTP_URL}}`; (b) the per-benchmark run path
   gains an optional server start/stop around generation+execution.
2. **What else lives there?** Existing tokens `{{CAPS}}`, `{{TIMEOUT}}`, `{{PYTHON_VERSION}}`
   (must remain substituted, in order, before the mock URL is injected). The run path already
   handles standard mode (one execution) and agent mode (generate + repair + re-execute).
3. **Disambiguation:** `{{MOCK_HTTP_URL}}` is a unique literal token; substitution is a plain
   `strings.ReplaceAll` like the others — no parser/grammar interaction. Server lifecycle is
   gated on the token's presence, so benchmarks without it are completely unaffected.
4. **Programs/flows that MUST still work:**
   - Every non-network benchmark (no `{{MOCK_HTTP_URL}}`) — runs unchanged, no server started.
   - Standard mode `api_call_json` — single execution against the live mock URL.
   - Agent mode `api_call_json` — generate → execute → (repair → execute)\* with the mock URL
     stable across all repair iterations (server outlives the iterations).
   - `--parallel N` — N independent mock servers, no shared port/state.
   - Hard-timeout path — server torn down, no leak.
   - All other `{{...}}` tokens still substitute correctly.
5. **Intentional change:** `api_call_json` no longer reaches the public internet; its URL is a
   local mock. Verdicts become deterministic (this is the goal, not a regression).

**Key risk:** the server must outlive agent-mode **repair** (the model may regenerate and re-run
the HTTP code on a second attempt). Lifetime is the whole run, not a single execution.

---

## Leaderboard-Integrity Note (interim)

Until this ships, **`api_call_json` must be excluded from scoring and from any published
leaderboard.** The v0.23.0 standard result must be reported on **31 benchmarks, not 32**. This is
specifically why the raw "GLM-5.1 91% beats Opus 84%" headline from the v0.23.0 standard run was
**not published** — the corrected, de-flaked table (GLM-5.1 ≈ gemini-3-1-pro ≈ 90%, Opus ≈
GPT-5.5 ≈ 87%) is the honest result. Once the mock lands, re-run and re-include the benchmark.

---

## Related Documents

- `design_docs/planned/v0_24_0/m-eval-openrouter-baseline-rotation.md` — the baseline rotation
  this de-flaked benchmark feeds; rotations inherit the determinism fix.
- `design_docs/planned/v0_24_0/m-eval-local-ollama.md` — nightly local-Ollama eval; was emitting
  spurious `api_error`/regression alerts partly traceable to this kind of network flakiness.
- `internal/eval_harness/agent_prompt.go` — existing `{{...}}` templating this extends.
- `benchmarks/api_call_json.yml` — the benchmark being fixed.

## References

- v0.23.0 standard run raw data: `/tmp/std_v23_clean/standard/api_call_json_*` (the 503/200 split)
- httpbin response shape (the mock mimics the `{headers, json, url}` envelope subset)

## Future Work

- Generalize to a small fixture catalog (`{{MOCK_HTTP_URL}}`, `{{MOCK_HTTP_URL_500}}`,
  mock TCP) if more network/error-path benchmarks are added.
- Separate, larger design: revisit stdout **exact-match strictness** so that "correct answer +
  extra debug lines" (e.g. Opus on `list_comprehension`: right `220`, plus two extra printed
  lines) is scored distinctly from a wrong answer. That is a grading-policy change, out of scope
  here.
