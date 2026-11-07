# M-SEC-GO-2026-6860: golang.org/x/net v0.61.0 → v0.62.0 (vuln-gate remediation, design only)

**Status**: Planned — awaiting human design approval (coordinator PR is labeled `needs-design-approval`; quorum is the optional next step, see Routing & Approval)
**Target**: v0.54.20 (patch lane on `dev`; **v1 is paused** — not targeted)
**Priority**: P0 (High) — dev CI is red on a security gate for a reviewed, shipped upstream fix
**Estimated**: 0.5 day (one sprint-executor pass: bump + regression fixture + CI proof)
**Dependencies**: None

**Scope of this document**: design only, per D-TRIAGE-2 red-dev routing. This task changes **no code, no config, no allowlist**. What follows is the remediation design for human approval; implementation starts only after approval via sprint-planner → "execute sprint".

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms) (`docs/docs/references/axioms.mdx`)

No language semantics change: this is a dependency version bump plus one regression test. Scores reflect that.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Parse output is byte-identical pre/post bump (upstream fix changes internal text accumulation, not observable semantics); no new nondeterminism. |
| A2: Replayability | 0 | No impact on traces. |
| A3: Effect Legibility | 0 | `_html_parse`/`_html_parseFragment` remain pure builtins; no effect signatures change. |
| A4: Explicit Authority | 0 | No capability surface change. |
| A5: Bounded Verification | 0 | Type check unchanged; new fixture is a plain Go test. |
| A6: Safe Concurrency | 0 | No concurrency changes. |
| A7: Machines First | 0 | No machine-facing surface change. |
| A8: Minimal Syntax | 0 | No syntax change. |
| A9: Cost Visibility | +1 | Worst-case cost of `std/html.parse` on adversarial input drops from O(n²) to O(n) allocations/copies; the builtin's declared cost becomes honest for the inputs it already accepts (≤50MB). |
| A10: Composability | 0 | XmlNode ADT and std/xml helpers unchanged. |
| A11: Structured Failure | 0 | `Result[XmlNode, string]` error paths unchanged. |
| A12: System Boundary | +1 | `std/html.parse` is a boundary-input parser (fetched/scraped HTML); adversarial boundary input can no longer amplify CPU/memory quadratically inside the existing 50MB cap. |

**Net Score: +2** → **Decision: Move forward** (contingent on human approval below)

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation (after human design approval) |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

---

## Problem Statement

Dev CI is red on the `govulncheck (vuln gate)` job, and the red is a **real, reachable security regression in a production path**, not noise.

### The red

- CI run [40655808264](https://github.com/sunholo-data/ailang/actions/runs/40655808264) on reviewed `origin/dev` @ `51b00832f7c4aec09ff5695a32bd27181b12c8e1` (local HEAD, merge of PR #1979) fails the `govulncheck (vuln gate)` job, step **"Run govulncheck against allowlist"**, with **exactly 1 unallowlisted gating finding: `GO-2026-6860`** (directive-sourced; consistent with the fleet log's identical observation, below).
- First observed red: run 40607442090 @ `8f8294c4eb91fcd9c11dbb322a4227ff661246dd`. Preceding CI 40551944965 @ `9646caebd0f7f43d1e3d6198b897b795ce6ae1ee` was green.
- Latest complete check-set observation at routing time: **17 checks; the failing vulnerability step is the last substantive step; 0 substantive gates masked** (2 post-setup cleanup steps skipped). Note: per the fleet log, this job is **non-required** on PRs (merges are MERGEABLE/UNSTABLE while it is red), but it reddens dev CI and is tracked; leaving it red is not an option.

### The advisory (primary source, fetched 2026-10-28)

[GO-2026-6860](https://vuln.go.dev/ID/GO-2026-6860.json), alias **CVE-2026-78661**, review status REVIEWED, **published 2026-10-27T21:16:18Z**:

> *"Avoid quadratic complexity when coalescing text nodes in html in golang.org/x/net"* — when coalescing adjacent text nodes in `addText` and `fosterParent`, appending to an existing TextNode's `Data` via string concatenation (`n.Data += text`) reallocated and copied the accumulated string on every token. Inputs containing **many short text runs separated by ignored end tags (such as `a</x>`)** caused **O(n²) allocations and byte copies**. Fix: accumulate coalesced text in per-node byte slices during parsing and materialize `Data` strings once at the end of parse.

- Affected package: `golang.org/x/net` (SEMVER `introduced: 0`), affected import `golang.org/x/net/html` with symbols **`Parse`, `ParseFragment`, `ParseFragmentWithOptions`, `ParseWithOptions`, `parser.addText`, `parser.fosterParent`, `parser.parse`**.
- **Fixed in `0.62.0`** (`events[].fixed`). Fix refs: [go.dev/cl/849365](https://go.dev/cl/849365), [go.dev/issue/82224](https://go.dev/issue/82224), [golang-announce](https://groups.google.com/g/golang-announce/c/sqrs2vTx_Xs).

### Time-dependence: this is NOT merge-caused

Per the routing directive ("Do not infer recent merge caused this time-dependent advisory") and independently corroborated by the fleet mission log ([fleet-mission-log.md:1944](../fleet-mission-log.md), iteration 64): dev HEAD `9646caebd`'s govulncheck was green in CI at 17:37Z on 2026-10-27; the advisory was published at **21:16:18Z** that day; the fix tag `v0.62.0` was created at **19:00:16Z** — before publication. A local negative control on the untouched base `9646caebd` (go 1.26.10) already gives `govulncheck ./...` rc 3 with Vulnerability #1 = GO-2026-6860. **The reviewed merge is innocent; the vuln DB moved.** The commits `8f8294c4`/`9646caebd` are not in this workspace clone (see Verification Log) — the timeline above is directive-sourced, corroborated by the fleet log.

### The affected production path (why it *gates*)

`golang.org/x/net` is a **direct** require at `go.mod:43` (`v0.61.0`). A full-repo census finds exactly **one first-party importer**: `internal/builtins/html.go`, which imports `golang.org/x/net/html` + `html/atom` and calls:

- `html.Parse` (`html.go:90`) — registered builtin **`_html_parse`** → `std/html.ail:31` `export pure func parse(html: string) -> Result[XmlNode, string]`
- `html.ParseFragment` (`html.go:170`) — registered builtin **`_html_parseFragment`** → `std/html.ail:36` `export pure func parseFragment(html: string) -> Result[[XmlNode], string]`

Both call targets are in the advisory's symbol list, so govulncheck reports the finding **function-reaching**, and the filter (`tools/govulncheck-filter`) exits 1 on unallowlisted function-reaching findings (module-level findings are reported but never gate — #703 invariant). That is the red.

**Impact (measured, not just quoted)**: any AILANG program that calls `std/html.parse` / `parseFragment` on untrusted HTML (fetch/scrape workflows — a core use for a deterministic AI-code substrate) can be driven into quadratic CPU/allocation blowup by an adversary who controls the document. The existing input caps (`htmlMaxInputSize` = 50MB, `htmlMaxDepth` = 256, reused from std/xml at `xml.go:19-20` / `html.go:25-26`) bound input size but **do not prevent the O(n²) amplification**. Measured on this rig, today, on the shipped `ailang` binary (v0.54.19, x/net v0.61.0), with the advisory's own input shape `"a</x>"` repeated n times (all text coalesces correctly — `chars == n`):

| n | input | wall (v0.61.0, this rig) |
|---|---|---|
| 40,000 | 200KB | ~1s |
| 80,000 | 400KB | ~2s |
| 160,000 | 800KB | ~3s |
| 320,000 | 1.6MB | ~10s |
| **640,000** | **3.2MB** | **~40s** |

Doubling 320k→640k gives **4× wall time — the quadratic signature, measured** (small-n rows are startup-noise-dominated). Extrapolating to the 50MB cap: **~2.7 hours of CPU** (and equivalent allocation churn) for a ~5–10 second download — a denial-of-service regression on a reachable production path. Post-fix (v0.62.0) the same work is linear: the 3.2MB case should drop from 40s to well under 1s.

### Why the gate goes red (mechanics)

- `.github/workflows/ci.yml` job `govulncheck` ("govulncheck (vuln gate)", lines 919–949): `actions/setup-go` with `go-version: '1.26.10'`; `go install golang.org/x/vuln/cmd/govulncheck@v1.3.0` (pinned deliberately — do not bump in this sprint); builds `./tools/govulncheck-filter`; runs `govulncheck -format json ./... | ./bin/govulncheck-filter` with `pipefail`.
- The filter reads `.govulncheck-allow.yml`, exits **0** only when every finding is allowlisted and unexpired; exits **1** on any unallowlisted function-reaching finding **or any expired allowlist entry**.
- `.govulncheck-allow.yml` currently holds **8 entries, all Ollama** (reason: upstream unpatched, re-verified 2026-07-31), all `expires: 2026-10-29`. **No entry for GO-2026-6860** — correctly so: the file's own policy says a fixed-upstream advisory gets the dependency **upgrade**, not an allowlist extension ("if any entry gains a fixed version, upgrade the dependency instead of extending this date").

---

## Goals

**Primary Goal:** Return the `govulncheck (vuln gate)` job to green on `dev` by upgrading `golang.org/x/net` v0.61.0 → **v0.62.0** (the reviewed upstream fix), with a regression fixture pinning the fix — **without adding any allowlist entry for GO-2026-6860**.

**Success Metrics:**
- `govulncheck -format json ./... | govulncheck-filter` exits 0 on the remediation PR: 0 unallowlisted findings; GO-2026-6860 gone from the finding stream; allowlist byte-identical (8 Ollama entries, untouched).
- `go.mod` diff is exactly one line (`golang.org/x/net v0.61.0` → `v0.62.0`) plus `go.sum` hashes; **no other module version moves** (MVS floors pre-verified as satisfied, see Solution Design).
- All 12 existing `internal/builtins/html_test.go` tests pass unchanged, `examples/runnable/html_parser.ail` type-checks and runs unchanged, goldens unchanged — parse output byte-identical.
- New coalescing regression fixture green post-bump and failing pre-bump (measured reference: ~10s pre-bump vs ≪1s expected post-bump at 1.6MB — see Problem Statement).
- Full CI matrix green on the remediation PR (17+ checks, 0 substantive gates masked; the `changes` classifier routes go.mod/go.sum PRs to `code=true` → full matrix, including this gate).

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Remediation path = dependency bump to v0.62.0 (vs. allowlist entry vs. vendored backport) | Security posture: taking the upstream fix vs. suppressing a *fixed* advisory; determines whether dev CI stays red for weeks | **human** (security policy) | design (this doc's approval) | low |
| D2: **No** `.govulncheck-allow.yml` entry for GO-2026-6860 — now or as fallback unless human rules otherwise | A silent/expiring suppression would mask a shipped fix and contradict the allowlist's own policy; must be an explicit, visible choice | **human** | design | low |
| D3: The 8 expiring Ollama entries (`expires: 2026-10-29`) are **not** re-armed in this sprint | Same PR would conflate this advisory's fix with an unrelated re-verification (8 re-checks per the file's header); mixing them hides both from review | **human** | design | low |
| D4: Release vehicle = patch **v0.54.20** on `dev` (v1 paused, not targeted) | Chooses which release carries the fix and keeps the paused v1 lanes untouched | **human** (release policy) | design | low |
| D5: govulncheck stays pinned at **v1.3.0** and gate mechanics stay untouched | The pin is deliberate (ci.yml comment: v1.4.0 crashes on `./...`); a version bump inside a security fix PR would make the green unprovable | agent (implementer) | design (recorded here) | low |
| D6: Regression fixture = functional coalescing test + generous wall-clock bound (size/tuning below) | Pins the fix against future downgrades/regressions without adding a flaky CI timing assertion | agent (implementer) | sprint | low |

### Design Freeze

Before implementation begins, these must be resolved (all are D1–D4 above — the human checkboxes):

- [ ] D1 approved: bump `golang.org/x/net` to v0.62.0 as the remediation path
- [ ] D2 approved: no allowlist entry for GO-2026-6860 in this sprint
- [ ] D3 approved: Ollama re-review (expiry 2026-10-29) routed separately, not swept into this PR
- [ ] D4 approved: ship as v0.54.20 patch on dev

---

## Solution Design

### Overview

One direct-dependency version bump (the reviewed upstream fix), one regression fixture, one changelog entry at release time, and CI proof that the gate is green. **No AILANG-language code changes** — `internal/builtins/html.go`, `std/html.ail`, the caps (50MB / 256 depth), and all public behavior stay byte-for-byte identical.

### Architecture

**Compatibility analysis (why a clean single-module bump):**

| Constraint | Required by x/net v0.62.0 | Repo state | Verdict |
|---|---|---|---|
| Go toolchain | `go 1.26.0` (from `v0.62.0.mod`, proxy) | `go.mod: go 1.26.10`; CI `go-version: '1.26.10'` | ✅ satisfied |
| `golang.org/x/crypto` | `v0.58.0` | `go.mod:105 v0.58.0 // indirect` | ✅ at floor |
| `golang.org/x/sys` | `v0.49.0` | `go.mod:45 v0.49.0` | ✅ at floor |
| `golang.org/x/term` | `v0.47.0` | `go.mod:46 v0.47.0` | ✅ at floor |
| `golang.org/x/text` | `v0.43.0` | `go.mod:47 v0.43.0` | ✅ at floor |

Because every floor is already met, `go get golang.org/x/net@v0.62.0` moves **exactly one module**. MVS will re-select x/net at v0.62.0 for all transitive consumers (Ollama, cloud SDKs, otel — x/net appears in `go.sum` only transitively beyond the direct pin), which is safe: they consume it as a library at ≥ their own required version.

**API surface:** `html.Parse(io.Reader) (*Node, error)` / `html.ParseFragment(io.Reader, *Node, ...)` signatures are unchanged — the advisory describes an internal accumulation change (`n.Data += text` → per-node byte slices, materialized once at parse end), i.e. same WHATWG parse semantics, same output tree, non-quadratic cost. `html/atom` usage (`atom.Body`) is untouched.

**Why NOT the alternatives (recorded for the approver):**

- *Allowlist entry:* contradicts the allowlist policy (fixed upstream → upgrade), leaves a known DoS-class path open for 90 days, and D-TRIAGE-2 explicitly forbids silent allowlisting. Rejected (D2).
- *Vendored backport:* forks upstream security maintenance for zero benefit when the upstream release exists and is compatible. Rejected.
- *Do nothing (gate is non-required on PRs):* leaves dev CI red on a *reachable security regression* in a stdlib parser and erodes the gate's meaning for the 8 legitimate Ollama suppressions. Rejected.

**Components:**
1. **The bump** — `go.mod`/`go.sum` only.
2. **The regression fixture** — new test in `internal/builtins/html_test.go` pinning linear-coalescing behavior on the advisory's own input shape (`a</x>` repetitions).
3. **The gate proof** — CI re-run; the finding stream must lose GO-2026-6860; allowlist must NOT grow.
4. **Bookkeeping** — this doc → implemented via `move_to_implemented.sh` after the release; triage iteration bookkeeping links the OpenAI/Anthropic evidence per the routing directive.

### Implementation Plan

**Phase 0: Pre-flight reproduction** (~0.5h)
- [ ] On a clean checkout of `dev` @ `51b00832`: `go install golang.org/x/vuln/cmd/govulncheck@v1.3.0`; `govulncheck -format json ./... | go run ./tools/govulncheck-filter` → expect exit 1, exactly `GO-2026-6860`, function-reaching frame through `internal/builtins/html.go` (recorded as the "before" evidence).
- [ ] `go list -m -versions golang.org/x/net` → confirm `v0.62.0` is the newest tag (proxy list verified at design time: …, v0.60.0, v0.61.0, v0.62.0).

**Phase 1: The bump** (~0.5h)
- [ ] `go get golang.org/x/net@v0.62.0 && go mod tidy`
- [ ] Diff audit: `go.mod` moves only line 43 (`v0.61.0` → `v0.62.0`); `go.sum` adds/removes only x/net hashes; `git diff --stat` shows exactly `go.mod`, `go.sum` (+ the fixture/changelog from Phase 2/4). If any other module moves, STOP and report — that contradicts this design's compat analysis.
- [ ] `make build` / `make fmt` / `make lint` / `make test` (or `make test-core` for the fast loop) — all green.

**Phase 2: Regression proof** (~2h)
- [ ] New fixture `TestHtmlParse_AdvisoryGo20266860TextCoalescing` (final name at implementer's discretion): build input of N repetitions of `a</x>` (the advisory's own trigger shape — short text runs separated by ignored end tags), calibrated from the measured curve (40k→~1s … 640k→~40s on this rig; the check-verified fixture in Example 1 is the reference): N ≈ 320,000 (1.6MB) is a good candidate — ~10s pre-bump, expected ≪1s post-bump;
  assert (a) parse returns `Ok`, (b) coalesced text length/content is exactly the n expected chars (verified semantics: `chars == n`), (c) wall-clock bound generous enough never to flake post-fix but fail pre-fix (exact N and bound are D6 tuning parameters, see Risks);
- [ ] Run the full 12 existing html tests + `internal/vm/builtin_coverage_test.go` + goldens: byte-identical behavior, no golden regeneration expected. If a golden moves, STOP and report — that would mean the bump changed observable output, contradicting the advisory.
- [ ] `ailang run --caps IO --entry main examples/runnable/html_parser.ail` → identical output (verified at design time: `ailang check` exit 0 on this file).

**Phase 3: CI proof** (~1h)
- [ ] Open PR. The `changes` job classifies go.mod/go.sum as code (`code=true`; docs-only lane regex is `^(design_docs/.*\.md|changelogs/.*\.md)$` only) → full matrix runs, including `govulncheck (vuln gate)`.
- [ ] Gate step green: `govulncheck-filter` prints "… reachable finding(s), all allowlisted and unexpired." with **0 unallowlisted**; GO-2026-6860 absent from the stream; allowlist diff shows no new entry (D2) and untouched Ollama expiries (D3).
- [ ] Confirm the rest of the 17+ check set is green and 0 substantive gates masked; record run URL in the implementation report.

**Phase 4: Release & bookkeeping** (~1h, post-approval of merge)
- [ ] Changelog entry under `[Unreleased]` → released as **v0.54.20** (D4) in `changelogs/v0.32-current.md`; `std/VERSION` bump per release flow.
- [ ] Move this doc `planned/` → `implemented/v0_54_20/` via `move_to_implemented.sh` + implementation report.
- [ ] Triage bookkeeping: link the CI run, the OpenAI verification and Anthropic Sonnet judgment evidence already attached to this iteration, and the predicted-next-red note below (Ollama expiry).

### Files to Modify/Create

**New files:**
- none (fixture goes into the existing test file)

**Modified files:**
- `go.mod` — 1 line: `golang.org/x/net v0.61.0` → `v0.62.0` (line 43)
- `go.sum` — x/net hash entries for v0.62.0 (drop v0.61.0 pair, add v0.62.0 pair)
- `internal/builtins/html_test.go` — add ~40 LOC fixture (D6 shape)
- `changelogs/v0.32-current.md` — `[Unreleased]` security entry (at release time)
- `std/VERSION` — `v0.54.19` → `v0.54.20` (release step, D4)

**Explicitly untouched (regression surface):**
- `internal/builtins/html.go`, `std/html.ail`, `.govulncheck-allow.yml`, `.github/workflows/ci.yml`, `tools/govulncheck-filter/`

---

## Examples

### Example 1: The adversarial input shape (verified live, 2026-10-28)

This program type-checks (`ailang check` → exit 0) and runs on the current binary; its cost profile is what the bump changes, not its output:

```ailang
module myproject/adv_fixture

import std/html (parse)
import std/string (repeat, length)
import std/xml (getText)
import std/result (Result)

-- GO-2026-6860 trigger shape: many short text runs separated by
-- ignored end tags. Pre-v0.62.0 this costs O(n^2) upstream; post-fix O(n).
func adversarialDoc(n: int) -> string {
  repeat("a</x>", n)
}

func coalescedChars(doc: string) -> Result[int, string] {
  match parse(doc) {
    Ok(root) => Ok(length(getText(root))),
    Err(msg) => Err(msg)
  }
}

export func main() -> () ! {IO} {
  match coalescedChars(adversarialDoc(100000)) {
    Ok(n) => println("coalesced text chars: ${show(n)}"),
    Err(msg) => println("parse error: ${msg}")
  }
}
```

Measured today on this binary (v0.61.0): `n=320000` → ~10s, `n=640000` → ~40s (**4× per doubling**). The Go fixture (Phase 2) pins exactly this shape; post-bump the same inputs must run ~linearly. (Existing canonical example: `examples/runnable/html_parser.ail` — `ailang check` exit 0, live.)

### Example 2: Before/after at the gate (what the sprint changes)

**Before (today, dev @ `51b00832`):**
```
govulncheck -format json ./... | ./bin/govulncheck-filter
→ exit 1: 1 unallowlisted finding(s): GO-2026-6860   [CI run 40655808264]
```

**After (v0.62.0):**
```
govulncheck -format json ./... | ./bin/govulncheck-filter
→ exit 0: N reachable finding(s), all allowlisted and unexpired.  [allowlist byte-identical]
```

---

## Success Criteria

- [ ] `govulncheck (vuln gate)` green on the remediation PR run URL recorded in the implementation report
- [ ] `go.mod`/`go.sum` are the only dependency changes; x/net at v0.62.0; `git diff` audit clean
- [ ] All 12 existing html tests + builtin coverage + goldens pass with **no golden changes**
- [ ] New coalescing fixture present and green, sized/tuned per D6
- [ ] `.govulncheck-allow.yml` byte-identical (no GO-2026-6860 entry — D2; no Ollama re-arm — D3)
- [ ] All tests passing; full CI matrix green; 0 substantive gates masked
- [ ] Documentation updated (changelog `[Unreleased]` → v0.54.20; this doc moved to implemented)
- [ ] Triage iteration bookkeeping links CI proof + the iteration's OpenAI/Anthropic evidence

---

## Testing Strategy

**Unit tests:**
- The 12 existing `internal/builtins/html_test.go` tests (simple paragraph, unclosed paragraphs, boolean attribute, lower-cased tag, comment preserved, script passthrough, namespaced tag, oversized input ×2, empty input, wrong-arg-type, fragment multiple roots) — unchanged, all must pass.
- New fixture (D6): functional assertion (correct coalesced text) + wall-clock bound, calibrated from the measured curve so it *fails* pre-bump and has ≥10× headroom post-bump on CI runners.

**Integration tests:**
- `internal/vm/builtin_coverage_test.go` (html builtins covered); `internal/pipeline/testdata/builtin_types.golden` must not move.
- `ailang check` + `ailang run` on `examples/runnable/html_parser.ail`.

**Manual / CI proof:**
- Phase 0 "before" reproduction (exit 1, single finding) vs Phase 3 "after" (exit 0) — the pair is the regression/CI proof.
- `make check-boundaries` if anything unexpected appears in the diff (should be a no-op for this change set).

---

## Verification Log (evidence for every load-bearing claim)

| # | Claim | Evidence (command + result, 2026-10-28 unless noted) |
|---|---|---|
| V1 | Advisory facts: id, alias, published, summary, symbols, fixed version | `curl -s https://vuln.go.dev/ID/GO-2026-6860.json` → exit 0; JSON fields: `id: GO-2026-6860`, `aliases: [CVE-2026-78661]`, `published: 2026-10-27T21:16:18Z`, affected import `golang.org/x/net/html` symbols `Parse, ParseFragment, ParseFragmentWithOptions, ParseWithOptions, parser.addText, parser.fosterParent, parser.parse`, `fixed: 0.62.0`, `review_status: REVIEWED`; refs CL 849365, issue 82224, golang-announce |
| V2 | Fix version v0.62.0 exists and is the newest | `curl proxy.golang.org/golang.org/x/net/@v/v0.62.0.info` → `Version: v0.62.0, Time: 2026-10-27T19:00:16Z`; `@v/list` sorted -V tail: `v0.60.0 v0.61.0 v0.62.0` |
| V3 | x/net v0.62.0 toolchain requirement ≤ repo pin | `v0.62.0.mod` → `go 1.26.0`; repo `go.mod:3 go 1.26.10`; CI `go-version: '1.26.10'` (ci.yml:934) |
| V4 | All four x/net v0.62.0 floors already met (no collateral module moves) | `v0.62.0.mod` requires x/crypto v0.58.0 / x/sys v0.49.0 / x/term v0.47.0 / x/text v0.43.0 vs repo `go.mod` lines 105/45/46/47 → identical |
| V5 | x/net is a **direct** require, pinned v0.61.0 | `go.mod:43` `golang.org/x/net v0.61.0`, no `// indirect` marker |
| V6 | Exactly ONE first-party importer of x/net (negative-existence) | `grep -rln "golang.org/x/net" --include="*.go" .` (full repo, excl. third_party/vendor/testdata) → single hit `internal/builtins/html.go` |
| V7 | Call path is function-reaching (advisory symbols) | `html.go` imports `golang.org/x/net/html` + `html/atom`; `html.Parse` at html.go:90, `html.ParseFragment` at html.go:170 — both in advisory symbol list (V1) |
| V8 | AILANG surface: std/html exports | `std/html.ail:31` `parse → _html_parse`, `:36` `parseFragment → _html_parseFragment`, both `pure func` |
| V9 | Input caps exist but don't prevent quadratic blowup | `xml.go:19-20` `xmlMaxDepth = 256`, `xmlMaxInputSize = 50 * 1024 * 1024`; `html.go:25-26` reuses both (`htmlMaxDepth = xmlMaxDepth`, `htmlMaxInputSize = xmlMaxInputSize`); advisory details confirm the trigger (`a</x>` runs) is *within* any size cap |
| V10 | Gate mechanics: which findings gate | `tools/govulncheck-filter/main.go` header: exit 1 on unallowlisted; `decide()` — `classifyReaching` gates function-reaching, module-only "reported but never gate" (#703 invariant); `pipefail` in ci.yml:948 |
| V11 | Allowlist has no GO-2026-6860 entry; 8 Ollama entries, expiry 2026-10-29 | `cat .govulncheck-allow.yml` → 8 entries (GO-2026-5750, GO-2025-4251/3824/3695/3689/3582/3559/3558/3557), all `expires: 2026-10-29`; grep for `6860` → absent |
| V12 | A go.mod-bump PR runs the full gate (no gate masking) | ci.yml `changes` job: docs-only lane regex `^(design_docs/.*\.md\|changelogs/.*\.md)$`, default-is-code; go.mod/go.sum → `code=true` → `govulncheck` job `if: needs.changes.outputs.code == 'true'` (ci.yml:919–922) |
| V13 | Regression fixtures exist and are green at design time | `grep -c "func Test" internal/builtins/html_test.go` → 12; `ailang check examples/runnable/html_parser.ail` → **exit 0** (live, 2026-10-28); `examples/runnable/html_parser.ail` exists (manifest + ls) |
| V14 | govulncheck is deliberately pinned v1.3.0 | ci.yml:936–941 comment: "Pinned (was @latest): govulncheck v1.4.0 started crashing on ./...; bump deliberately after confirming a newer release is healthy" |
| V15 | Timeline is time-dependent, not merge-caused (negative control) | Fleet mission log (independent lane): base `9646caebd` green at 17:37Z 2026-10-27; advisory published 21:16:18Z; local `govulncheck ./...` on untouched `9646caebd` → rc 3, Vulnerability #1 GO-2026-6860 ([fleet-mission-log.md:1943-1945](../../fleet-mission-log.md)); directive: "Do not infer recent merge caused this" |
| V16 | The two timeline SHAs are directive-sourced (not in this clone) | `git cat-file -t 8f8294c4…` → `fatal: could not get object info`; same for `9646caebd…`; local HEAD is `51b00832` (= the red CI commit) |
| V17 | No existing GitHub issue/PR names the advisory (routing-time) | Directive: "No existing issue/PR search hit for advisory ID at routing time"; corroborated by fleet log: "No open PR or issue names it" ([fleet-mission-log.md:1945](../../fleet-mission-log.md)); repo-wide grep for `GO-2026-6860` hits only fleet bookkeeping (dashboard line 9; log 1943/1944/1991/2019) |
| V18 | No prior design doc covers this topic (duplicate gate) | `create_planned_doc.sh m-sec-go-2026-6860-xnet-bump v0_54_20` dual search (SimHash + neural) → "(none found)" in all four buckets |
| V19 | Quadratic impact is **measured**, not inferred: 4× wall per 2× input on the shipped binary | `ailang run` (this rig, v0.54.19 / x/net v0.61.0) on the check-verified fixture above: n=320,000 → ~10s; n=640,000 → ~40s (rc 0, correct output `chars == n`); extrapolates to ~2.7h at the 50MB cap. Post-fix, the Phase-0→Phase-3 pair must show the same fixture ≪ its bound |

---

## Deferred Decisions

The following are intentionally left open for the implementer (agent latitude within the frozen decisions above):

- Fixture input size and wall-clock bound (D6) — agent may choose, given the constraints: fails pre-bump, ≥10× headroom post-bump, ≤ CI timeout budget (calibrate from the measured curve in Problem Statement).
- Exact fixture name and whether the timing assertion lives in the same test or a `-short`-gated sibling — agent may choose (avoid `testing.Short()` skip in CI default mode).
- Whether the changelog entry lands in the bump PR or the release commit — agent may choose per repo release convention (post-release skill owns the release flow).
- Whether to run `ailang design-quorum` on this doc before sprint-planning — optional documented step (skill); controller/coordinator may schedule it during approval.

## Non-Goals

**Not attempted in this feature:**
- **No allowlist entry for GO-2026-6860** — D2; the fix exists upstream, so suppression is wrong (and D-TRIAGE-2 forbids silent allowlisting).
- **No re-arm/review of the 8 Ollama entries** (they expire **2026-10-29**; that is a separate re-verification cycle per the allowlist header, routed below) — D3.
- **No changes to AILANG language semantics, stdlib API, builtins, caps, or effect signatures** — `internal/builtins/html.go` and `std/html.ail` are untouched.
- **No gate/tooling changes** — govulncheck stays pinned v1.3.0 (D5); filter and workflow untouched.
- **No v1 work** — v1 is paused; this ships as a v0.54.20 patch (D4).
- **No backports to other branches** — none exist for this line; out of scope.

## Timeline

**Day 1** (~5h total, single sprint pass):
- Phase 0 pre-flight reproduction (~0.5h)
- Phase 1 the bump (~0.5h)
- Phase 2 regression proof (~2h)
- Phase 3 CI proof (~1h)
- Phase 4 release & bookkeeping (~1h, after merge approval)

**Total: ~0.5–1 day.** (Estimate is 2× the mechanical work: the audit/diff-verification steps dominate.)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Transitive consumers (Ollama, cloud SDKs, otel) misbehave at x/net v0.62.0 | Med | Floors pre-verified identical (V4); `make test` full suite gates the PR; MVS means consumers get ≥ their required versions — same minor release family |
| Fix changes observable parse output (text materialized "once at end") | Med | Advisory + fix CL describe internal accumulation only; 12 existing tests + goldens + example run must show byte-identical output — any golden move is a STOP-and-report (Phase 2) |
| Timing-bound fixture flakes in CI | Med | D6: calibrate N and bound from the measured curve (1.6MB ≈ 10s pre-fix on this rig; CI runners differ) — pick ≥10× headroom post-fix that still fails pre-fix, and keep the fixture's total CI cost ≪ the gate's 15-min timeout |
| A new advisory lands mid-sprint (vuln DB is time-dependent — this red proved it) | Low | Gate re-runs on the PR; if a new finding appears, it routes through triage again — do not fold unrelated advisories into this PR (same rule as D3) |
| **Predicted next red:** the 8 Ollama allowlist entries expire **2026-10-29** (tomorrow); gate "fails on any allowlist entry past its expiry" | **High (certain date, unrelated cause)** | **Routed, not scope:** flag to the allowlist's owning re-review cycle (file header: post-release skill checks expiries; re-verify per its 3-check procedure — do not just bump dates). Recorded here so it is not silently discovered as a *new* mystery red |
| This design doc's own PR skips the gate (docs-only lane) | None | Intentional and safe: the doc changes no code; the remediation PR (go.mod) is the one that must and does re-run the full gate (V12) |

## Related Documents

<!-- Auto-populated by dual search on "6860 xnet bump": (none found). Manual additions below. -->

**Implemented (may inform design):**
- [m-stdlib-html.md](../../implemented/v0_19_1/m-stdlib-html.md) — the std/html builtins this advisory lands on (peer of std/xml, same XmlNode ADT and caps)

**Planned (check for overlap):**
- (none found — dual SimHash + neural search returned no matches in either bucket)

**Fleet bookkeeping (independent corroboration):**
- [fleet-mission-log.md:1943-1945](../../fleet-mission-log.md) — iteration 64: the red, the attribution negative control, and "No open PR or issue names it"
- [fleet-mission-log.md:1991,2019](../../fleet-mission-log.md) — iterations 64/65: inherited red observations on later bases
- [fleet-mission-dashboard.md:9](../../fleet-mission-dashboard.md) — "It needs a `go.mod` bump; that is Mark's or V1's" (fleet hands the fix to the human; this doc is that hand-off)

**Gate surface:**
- `.github/workflows/ci.yml` (govulncheck job, lines 919–949) · `tools/govulncheck-filter/` · `.govulncheck-allow.yml`

## References

- Advisory: https://vuln.go.dev/ID/GO-2026-6860.json · https://pkg.go.dev/vuln/GO-2026-6860 (alias CVE-2026-78661)
- Upstream fix: [go.dev/cl/849365](https://go.dev/cl/849365) · [go.dev/issue/82224](https://go.dev/issue/82224) · [golang-announce](https://groups.google.com/g/golang-announce/c/sqrs2vTx_Xs)
- Module proxy: https://proxy.golang.org/golang.org/x/net/@v/v0.62.0.info (and `.mod`, `@v/list`)
- CI run: https://github.com/sunholo-data/ailang/actions/runs/40655808264 (red, dev @ `51b00832`)
- [Design Axioms](/docs/references/axioms) (`docs/docs/references/axioms.mdx`) · [GOVERNANCE.md](../../../GOVERNANCE.md) · [SECURITY.md](../../../SECURITY.md)

## Future Work

- The **Ollama re-review cycle** (8 entries, expiry 2026-10-29): re-run the allowlist header's three-check procedure (complete-finding census, fixed-version check, newest-release check) before extending any date — that work owns the next predicted red, not this design.
- Consider (separately, P2): a scheduled job or make target that diffs `govulncheck` finding streams between dev HEAD and the vuln DB *at run time*, so time-dependent advisories surface in hours, not at the next merge — this red took a triage iteration to attribute.
- The eval-harness Ollama path (the allowlist's reason for existing) remains the long-term path to zero Ollama suppressions; unchanged by this design.

---

## Routing & Approval (D-TRIAGE-2 bookkeeping)

- **This task**: design only — this document is the only artifact (`design_docs/planned/v0_54_20/`); **no code, no config, no allowlist change** by the triage mission itself.
- **Next gates (per Work Routing)**: human approval of D1–D4 (this doc carries `needs-design-approval`) → optional `ailang design-quorum` → `sprint-planner` ("execute sprint") → `sprint-executor` → `sprint-evaluator`. Merge starts at sprint-planner per coordinator config; **nothing here is implemented, approved, merged, or deployed by this task.**
- **Evidence linked in this iteration's bookkeeping** (to follow): independent OpenAI verification and Anthropic Sonnet judgment, per the routing directive; CI observations: 17-check set, failing vuln step last substantive, 0 substantive gates masked.

---

**Document created**: 2026-10-28
**Last updated**: 2026-10-28