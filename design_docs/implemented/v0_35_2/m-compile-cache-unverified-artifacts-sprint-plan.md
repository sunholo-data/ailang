# Sprint Plan: M-COMPILE-CACHE-UNVERIFIED-ARTIFACTS

## Summary

Bind every executable compile-cache artifact set to the exact module ID and newly computed cache
key that authorize it, preserve the loader's exact source snapshot, make compile-cache clearing
complete, and fail visibly rather than serving a route surface inconsistent with the compiled
interface.

**Target:** v0.35.2  
**Planned at:** 137842bfd7f7767e5e8893c6238f62feb73dcdc1 (std/VERSION = v0.35.1)  
**Duration:** 4 working days  
**Milestones:** 4; one green, independently verifiable commit per milestone  
**Risk:** High correctness impact; medium implementation risk  
**Dependencies:** None  
**Design:** design_docs/implemented/v0_35_2/m-compile-cache-unverified-artifacts.md  
**Issue:** #1046

The design's four-milestone decomposition and estimates are preserved exactly: M1 2 days, M2
0.75 day, M3 0.5 day, and M4 0.75 day. The seven-day repository diff is 499 files, 61,378
insertions, and 6,057 deletions across unrelated parallel mission lanes, so it is not a credible
single-sprint velocity sample. The approved design's task-level estimates are the better planning
instrument. JSON LOC values are forecasting units for progress reporting, not scope or acceptance
caps.

## Approval and scope boundary

Decision `D-55` remains **OPEN** in the mission ledger. What has happened is narrower and must not be
read as a ruling: the ask carried a pre-registered *default* — option (a), to be applied at the next
iteration "as a controller routing call rather than as a ruling" — and iteration 329 is that next
iteration, so the default has been applied. The loop may not resolve a ledger row on its own behalf,
so `D-55` stays open and answerable: a later human ruling of (b) or (c) supersedes this sprint's scope
and this plan must then be revised. The tracked design's earlier PARKED banner
is superseded for this sprint: the approved threat model is **accidental corruption and concurrent
or interrupted cache publication**, not an adversary who can write the cache directory.

In scope:

- SHA-256 consistency binding for the four fixed payloads, exact module ID, caller-computed expected
  cache key, and v4 format stamp.
- Stamp-last publication, manifest-after-artifacts ordering, visible optional-persistence failures,
  source snapshot ownership, complete compile-cache clearing, and the narrow route/iface invariant.
- The design's bounded-read work in T13: 16 MiB per blob, 64 KiB stamp, 32 MiB aggregate module
  limit, regular-file checks, LimitReader(limit+1) sentinel behavior, and decode only after all
  hashes pass. These byte-limit tests are corruption robustness and must remain in M1.

Explicitly out of scope:

- Provenance signatures, MACs, authenticated artifacts, or any claim that a colocated SHA-256
  proves compiler provenance.
- Adversarial encoding/gob hardening, hostile-input allocation/work budgets beyond the approved
  byte ceilings, decoder sandboxing, or a new serialization format. That work is parked as
  m-cache-artifact-adversarial-decode and must not be pulled into this sprint.
- Hashed module-directory identities, cross-process locks, durable transactions/fsync, cache GC,
  cache doctor/compile-verify, a cache-bypass flag for serve-api, or unrelated key-input audits.

If implementation reveals that correctness requires excluded hardening, stop at the current green
boundary and return to design review. Do not silently widen M1.

## Baseline acceptance gate

The design's Success-Criteria Python gate was run verbatim on the clean planning tree before this
plan was written. It exited 1 after 6.461702084 seconds with exactly:

~~~text
Traceback (most recent call last):
  File "<stdin>", line 19, in <module>
AssertionError: ('M1', 'missing or skipped', {'TestCacheArtifacts_ByteLimits', 'TestCacheArtifacts_Authorization', 'TestCacheArtifacts_ReadSnapshot', 'TestCacheArtifacts_Migration', 'TestCacheArtifacts_PartialWrite', 'TestCachePipeline_WriteFailure'})
~~~

This is the intended non-vacuous baseline failure. The first go test -json
./internal/pipeline subprocess returned 0 and emitted valid JSON; Python parsed it and failed only
because none of the six selected M1 test names produced a pass event. There was no package compile
error, wrong package path, Python/Go invocation error, or timeout. The gate stops at M1 by design,
so later missing names were confirmed separately with a controlled search below.

## HEAD verification

All evidence here was re-run in this worktree on 2026-09-05. Empty/negative searches are paired
with known-positive controls in the same command. Proposed behavior and estimates are not presented
as HEAD facts.

### H1 — planning identity and clean tree

~~~sh
git rev-parse HEAD
cat std/VERSION
git status --short
~~~

Observed:

~~~text
137842bfd7f7767e5e8893c6238f62feb73dcdc1
v0.35.1
~~~

git status --short printed no lines. The design's recorded baseline
087fbea631a0b80556baa034b499fbdae33e76d2 is historical rather than current; behavior was
therefore re-measured.

### H2 — artifact loads have neither expected-key authorization nor bounded reads

~~~sh
rg -n 'func \(cs \*CacheStore\) (LoadArtifacts|StoreArtifacts)|os\.ReadFile|cacheKeyVersion' \
  internal/pipeline/cache_store.go internal/pipeline/cache_key.go
sed -n '137,235p' internal/pipeline/cache_store.go
~~~

Observed controls and defect predicates:

~~~text
internal/pipeline/cache_key.go:24:const cacheKeyVersion = "v3"
internal/pipeline/cache_store.go:137:func (cs *CacheStore) StoreArtifacts(moduleID string, cm *CachedModule) error {
internal/pipeline/cache_store.go:185:func (cs *CacheStore) LoadArtifacts(moduleID string) (*CachedModule, error) {
internal/pipeline/cache_store.go:189: coreData, err := os.ReadFile(...)
internal/pipeline/cache_store.go:200: ctiData, err := os.ReadFile(...)
internal/pipeline/cache_store.go:211: ifaceData, err := os.ReadFile(...)
internal/pipeline/cache_store.go:221: ctorData, err := os.ReadFile(...)
~~~

The complete method body decodes each returned slice immediately. It has no expected cache-key
parameter, artifact stamp/hash check, regular-file check, or bounded reader.

### H3 — manifest authorization precedes artifacts and errors are discarded

~~~sh
rg -n 'cacheStore\.(Store|StoreArtifacts|Save|LoadArtifacts)|NewCacheStore' \
  internal/pipeline/pipeline_module.go
sed -n '245,385p' internal/pipeline/pipeline_module.go
sed -n '415,427p' internal/pipeline/pipeline_module.go
~~~

Observed:

~~~text
280: if cached, loadErr := cacheStore.LoadArtifacts(string(modID)); loadErr == nil {
369: cacheStore.Store(string(modID), &CacheEntry{
377: _ = cacheStore.StoreArtifacts(string(modID), &CachedModule{
423: _ = cacheStore.Save()
~~~

The complete block confirms the ordering and ignored errors. A lookup with the correct new key can
therefore load stale, independently stored bytes and skip compilation.

### H4 — source identity is reread opportunistically and embedded bytes are not retained

~~~sh
rg -n 'os\.ReadFile|sourceContent|SourceContent|std\.FS\.ReadFile|LoadedModule\{' \
  internal/pipeline/pipeline_module.go internal/loader/loader.go
sed -n '258,277p' internal/pipeline/pipeline_module.go
sed -n '194,225p' internal/loader/loader.go
sed -n '309,345p' internal/loader/loader.go
~~~

Observed controls include the disk loader read at loader.go:276, embedded fallback via
std.FS.ReadFile, and the final LoadedModule literal. The pipeline initializes sourceContent to an
empty string and only replaces it when os.ReadFile(srcPath) succeeds. No SourceContent field appears
in the paired search or returned literal. A failed second read therefore hashes empty text rather
than the bytes the lexer parsed.

### H5 — compile-clear only clears the manifest

~~~sh
rg -n 'func \(cs \*CacheStore\) Clear|compile-clear|runCompileCacheClear' \
  internal/pipeline/cache_store.go cmd/ailang/cache.go cmd/ailang/cache_compile.go
sed -n '109,117p' internal/pipeline/cache_store.go
~~~

Observed controls locate compile-clear in the dispatcher and runCompileCacheClear. The complete
Clear method only replaces the manifest and returns cs.Save(); it never removes modules/.

### H6 — route/iface mismatch is silently skipped

~~~sh
rg -n 'registerModule|extractRouteAnnotations|CACHE_ROUTE_IFACE_MISMATCH|loaded\.Iface' \
  internal/apiserver/module_entry.go internal/apiserver/routes.go
sed -n '31,125p' internal/apiserver/module_entry.go
sed -n '83,126p' internal/apiserver/routes.go
~~~

Observed controls find registerModule, extractRouteAnnotations, and loaded.Iface. The stable
diagnostic search is empty. registerModule returns silently for nil iface and performs its
idempotent return before AST/iface reconciliation. extractRouteAnnotations has no unmatched-name
branch.

### H7 — required new names are absent; controls are discoverable

~~~sh
rg -n 'TestCacheArtifacts_|TestCachePipeline_WriteFailure|TestCacheSource_ExactSnapshot|TestCachePipeline_EmbeddedKeys|TestCachePipeline_SourceEditBehavior|TestCacheStore_ClearArtifacts|TestCompileCacheClear_Artifacts|TestRegisterModule_RouteIfaceMismatch|TestServeAPI_DivergentCacheTools|TestCacheStore_ArtifactRoundTrip|TestServeAPI_MCPToolSurface' \
  internal cmd --glob '*_test.go'
~~~

Observed output contains only the positive controls:

~~~text
cmd/ailang/serve_api_mcp_surface_test.go:16:func TestServeAPI_MCPToolSurface(t *testing.T) {
internal/pipeline/cache_store_test.go:135:func TestCacheStore_ArtifactRoundTrip(t *testing.T) {
internal/pipeline/cache_store_test.go:303:func TestCacheStore_ArtifactRoundTrip_DiverseExprTypes(t *testing.T) {
~~~

No T1-T13 name exists at HEAD.

### H8 — relevant suites are green while the defect remains

~~~sh
go test ./internal/pipeline \
  -run '^(TestCacheKey_InvalidatesOnSourceEdit|TestCacheStore_.*|TestNewCacheStore_.*|TestModuleCacheKey_.*)$' \
  -count=1 -timeout=90s
go test ./internal/loader ./internal/apiserver -count=1 -timeout=150s
make check-boundaries
~~~

Observed:

~~~text
ok github.com/sunholo-data/ailang/internal/pipeline 0.331s
ok github.com/sunholo-data/ailang/internal/loader 0.326s
ok github.com/sunholo-data/ailang/internal/apiserver 0.933s
Checking architecture boundaries (logical layers over internal/)...
OK: no architecture boundary violations.
~~~

All exited 0 under explicit outer deadlines.

### H9 — stale execution is still reproducible at this HEAD

The design's reviewed build and AC-E2E body were run with expected baseline count 6 against a fresh
binary from this tree. Observed:

~~~text
build_stdout=''
build_stderr=''
build_exit=0
divergent tools: ['f1', 'f2', 'f3', 'f4', 'f5', 'f6']
manifest stayed fresh; deleting one artifact directory yields seven tools
repro_exit=0
~~~

The newer HEAD therefore preserves the design's core behavioral premise.

### H10 — M4 sandbox viability and durable paths

~~~sh
go test ./cmd/ailang -run '^TestServeAPI_MCPToolSurface$' -count=1 -timeout=150s
git ls-files cmd/ailang/serve_api_mcp_surface_test.go \
  internal/pipeline/cache_store_test.go internal/pipeline/cache_invalidation_test.go \
  internal/apiserver/module_entry_test.go
git check-ignore -v .ailang/state/sprints/sprint_M-COMPILE-CACHE-UNVERIFIED-ARTIFACTS.json
~~~

Observed:

~~~text
ok github.com/sunholo-data/ailang/cmd/ailang 13.050s
cmd/ailang/serve_api_mcp_surface_test.go
internal/apiserver/module_entry_test.go
internal/pipeline/cache_invalidation_test.go
internal/pipeline/cache_store_test.go
.gitignore:82:.ailang/ .ailang/state/sprints/sprint_M-COMPILE-CACHE-UNVERIFIED-ARTIFACTS.json
~~~

The existing test's buildAilang helper creates a fresh binary under os.MkdirTemp, bounds go build
to 120 seconds, and uses exec.CommandContext with a 30-second MCP deadline over stdin/stdout pipes.
It opens no listening socket and uses no network or PATH-installed ailang. It passed in the
workspace-write sandbox, so M4 and the reviewed AC-E2E probe are **not sandbox-blind**. The JSON is
intentionally ignored; all decisions and evidence are duplicated here.

## Corrections to approved-design metadata

- The design records verification at 087fbea63; HEAD is now 137842bfd. No behavioral claim used by
  this plan was refuted after remeasurement.
- The design's Target: v0.35.1 is stale: HEAD already reports v0.35.1, and the approved document and
  plan live under planned/v0_35_2. This plan targets v0.35.2.
- The design's PARKED/D-55 banner is superseded for this sprint by the applied pre-registered default,
  NOT by a human ruling: `D-55` is still OPEN. This changes authorization and scope status, not the
  technical design, and it is reversible if the human answers (b) or (c).

## Schedule and commit boundaries

| Day | Milestone | Estimate | Dependency | Boundary |
|---|---|---:|---|---|
| 1-2 | M1 — verified artifacts and visible publication failure | 2.0 d | none | Commit 1; T1-T5 and T13 green |
| 2-3 | M2 — loader-owned source identity | 0.75 d | M1 | Commit 2; M1 plus T6-T8 green |
| 3 | M3 — complete compile-cache clearing | 0.5 d | M2 | Commit 3; M1-M2 plus T9-T10 green |
| 3-4 | M4 — route invariant and MCP regression | 0.75 d | M3 | Commit 4; all 14 named passes plus compatibility gates green |

There is no intentionally-red boundary. If a milestone cannot meet its boundary, leave its work
uncommitted for the controller and do not begin the next milestone.

### Cumulative named-test runner

At each milestone run this self-contained command with only the final boundary argument changed.
It is the design's non-vacuous event check filtered to that milestone and every earlier milestone.

~~~sh
python3 - M1 <<'PY'
import json, re, subprocess, sys

boundary = sys.argv[1]
order = {"M1": 1, "M2": 2, "M3": 3, "M4": 4}
assert boundary in order, boundary
checks = [
 ("M1", "./internal/pipeline", ["TestCacheArtifacts_Authorization", "TestCacheArtifacts_PartialWrite", "TestCacheArtifacts_ReadSnapshot", "TestCacheArtifacts_Migration", "TestCachePipeline_WriteFailure", "TestCacheArtifacts_ByteLimits"]),
 ("M2", "./internal/loader", ["TestCacheSource_ExactSnapshot"]),
 ("M2", "./internal/pipeline", ["TestCacheSource_ExactSnapshot", "TestCachePipeline_EmbeddedKeys", "TestCachePipeline_SourceEditBehavior"]),
 ("M3", "./internal/pipeline", ["TestCacheStore_ClearArtifacts"]),
 ("M3", "./cmd/ailang", ["TestCompileCacheClear_Artifacts"]),
 ("M4", "./internal/apiserver", ["TestRegisterModule_RouteIfaceMismatch"]),
 ("M4", "./cmd/ailang", ["TestServeAPI_DivergentCacheTools"]),
]
for milestone, package, names in checks:
    if order[milestone] > order[boundary]:
        continue
    pattern = "^(" + "|".join(re.escape(n) for n in names) + ")$"
    p = subprocess.run(["go", "test", "-json", package, "-run", pattern,
                        "-count=1", "-timeout=150s"],
                       capture_output=True, text=True, timeout=180)
    events = [json.loads(line) for line in p.stdout.splitlines() if line.startswith("{")]
    passed = {e.get("Test") for e in events if e.get("Action") == "pass"}
    assert p.returncode == 0, p.stdout + p.stderr
    assert set(names) <= passed, (milestone, "missing or skipped", set(names) - passed)
    for name in names:
        print(milestone, "PASS", name)
PY
~~~

The exact final invocation line per boundary is python3 - M1, M2, M3, or M4 respectively; the
heredoc body is otherwise identical. Every row has a 150-second Go deadline and 180-second outer
subprocess deadline.

## M1 — verified artifact loads and visible publication failure

**Estimate:** 2 days; approximately 780 LOC including tests.  
**Dependency:** none.  
**Commit:** one M1 commit only after all six M1 names pass.

### Files touched

- Create internal/pipeline/cache_artifacts.go for stamp schema, bounded single-open reads, hashes,
  serialization, stamp-last publication, and private instance-local I/O seams.
- Create internal/pipeline/cache_artifacts_test.go for T1-T3 and storage-level T13.
- Create internal/pipeline/cache_runtime.go for private lookup/publication/warning coordination.
- Modify internal/pipeline/cache_store.go to delegate artifacts and accept cache-key signatures.
- Modify internal/pipeline/cache_key.go for v3 to v4 only.
- Modify internal/pipeline/pipeline_module.go to pass the computed key, count verified hits, order
  publication correctly, and surface init/save/write diagnostics on stderr.
- Modify internal/pipeline/cache_store_test.go for existing round-trip signature updates.
- Modify internal/pipeline/cache_invalidation_test.go for T4-T5 and pipeline-level T13.

This file placement is measured, not another milestone: cache_store.go is already 506 lines and
pipeline_module.go is 772 lines, only 28 below the 800-line CI ceiling. M1 extracts cache
responsibility rather than growing either large file. No exported test-only or mutable global hook.

### Day breakdown

- Day 1: constants/stamp; serialize before writes; bounded fixed-order snapshots; identity, key,
  version, digest checks; hash-before-decode; T1, T3, and read-side T13.
- Day 2: unique stamp publication; failure seams; manifest-after-artifacts; diagnostics; migration;
  T2, T4, T5, and publication-side T13; close a green boundary.

### Named tests and mutation obligations

| ID / name | Mutation this test must kill |
|---|---|
| T1 TestCacheArtifacts_Authorization | Remove key/version/module checks or trust the stored key. |
| T2 TestCacheArtifacts_PartialWrite | Accept a key-only stamp, omit a blob hash, or decode unverified bytes; untouched A remains the control. |
| T3 TestCacheArtifacts_ReadSnapshot | Reopen blobs after hash checks. |
| T4 TestCacheArtifacts_Migration | Omit version bump or accept legacy artifact fallback. |
| T5 TestCachePipeline_WriteFailure | Discard persistence errors, publish after failed artifacts, or make optional persistence fatal. |
| T13 TestCacheArtifacts_ByteLimits | Remove blob/stamp/module ceilings, keep only stat, omit the extra-byte sentinel, or decode before every hash passes. Reason/scope/read/decode assertions must fail even if a later hash mismatch recompiles. |

These are the design's claimed kills; no adversarial-decode mutation is added.

### Exact boundary acceptance

Run the cumulative runner with final invocation python3 - M1. It must print six M1 PASS lines. Then:

~~~sh
go test ./internal/pipeline \
  -run '^(TestCacheStore_ArtifactRoundTrip|TestCacheStore_ArtifactRoundTrip_DiverseExprTypes|TestCacheStore_RoundTrip)$' \
  -count=1 -timeout=150s
~~~

Both commands must exit 0. They are sandbox-viable.

## M2 — loader-owned source identity

**Estimate:** 0.75 day; approximately 260 LOC including tests.  
**Dependency:** green M1 commit.  
**Commit:** one M2 commit after cumulative M1-M2 acceptance.

### Files touched

- Modify internal/loader/loader.go to retain immutable SourceContent *string from the exact read.
- Modify internal/loader/loader_test.go for loader-side T6.
- Modify internal/pipeline/pipeline_module.go to remove the reread and visibly bypass nil snapshots.
- Modify internal/pipeline/cache_invalidation_test.go for pipeline-side T6 and T7-T8.
- Modify internal/pipeline/cache_runtime.go only for the nil-source diagnostic path; do not serialize
  or copy source snapshots into runtime result assembly.

### Day breakdown

- Pin disk/embedded bytes and known-empty versus unavailable source; remove opportunistic rereads.
- Prove embedded keys, observable 3 to 41 execution, current Core, subsequent verified warm hit, and
  NoCache parity before committing.

### Named tests and mutation obligations

| ID / name | Mutation this test must kill |
|---|---|
| T6 TestCacheSource_ExactSnapshot in loader and pipeline | Retain only disk text, use empty embedded text, reread disk at key time, or treat nil as empty. |
| T7 TestCachePipeline_EmbeddedKeys | Reintroduce the silent empty-source branch at the pipeline key site. |
| T8 TestCachePipeline_SourceEditBehavior | Keep old blobs while advancing the manifest, or disable all hits to fake correctness. |

### Exact boundary acceptance

Run the cumulative runner with final invocation python3 - M2, then:

~~~sh
go test ./internal/loader -run '^TestLoad_EmbeddedStdlibFallback$' -count=1 -timeout=150s
go test ./internal/pipeline -run '^TestCacheKey_InvalidatesOnSourceEdit$' -count=1 -timeout=150s
~~~

All commands must exit 0. They are sandbox-viable.

## M3 — complete compilation-cache clearing

**Estimate:** 0.5 day; approximately 180 LOC including tests.  
**Dependency:** green M2 commit.  
**Commit:** one M3 commit after cumulative M1-M3 acceptance.

### Files touched

- Modify internal/pipeline/cache_store.go so Clear saves empty v4 state and removes only modules/.
- Modify internal/pipeline/cache_store_test.go for T9.
- Modify tracked cmd/ailang/serve_api_mcp_surface_test.go for T10 and isolated CLI fixtures.

**File-size contingency for `cmd/ailang/serve_api_mcp_surface_test.go` (round-1 evaluator finding,
reproduced first-party by the controller).** M1 gives the two production files a measured size
justification; this test file had none, and it is the only file receiving fixtures from *two*
milestones — T10 here (old-stamp restoration, single-stale-blob poisoning) and T12 in M4
(fresh-binary MCP stdio, f7 execution). `make check-file-sizes` gates test files too: its glob is
`find internal cmd -name "*.go"` at `make/code-health.mk:167`, with the 800-line ceiling at `:169`,
and nothing excludes `_test.go`. Measured at `137842bfd`: the file is **140** lines, so the two
additions have ~660 lines of headroom and a breach is unlikely. The contingency, so the executor
does not have to invent one at a commit boundary: if the file exceeds **650** lines after either
addition, split the M4 fixtures into a sibling `cmd/ailang/serve_api_cache_divergence_test.go` in
the same package rather than trimming a fixture — the fixtures are the evidence, and the durable
tracked-path requirement is satisfied by any tracked file in `cmd/ailang`. Re-run
`make check-file-sizes` at each of the M3 and M4 boundaries, not only at the end.

This is also why M3 stays independent of the API diagnostic work, exactly as the design doc's M3
section states: the shared test file is a *location*, not a dependency, and M3's commit must be
green without any M4 symbol.

cmd/ailang/cache.go and cache_compile.go are exercised but need no planned production edit: HEAD
already dispatches compile-clear, returns Clear errors nonzero, and prints success afterward. Change
them only if T10 refutes that observed contract, and record the discrepancy first.

### Day breakdown

- Implement narrow subtree removal, missing-directory idempotence, errors, root variants, sentinels.
- Add the fresh-binary CLI success/failure test and close the cumulative green boundary.

### Named tests and mutation obligations

| ID / name | Mutation this test must kill |
|---|---|
| T9 TestCacheStore_ClearArtifacts | Restore manifest-only clear, swallow deletion failure, or delete wider than the compile-cache root. |
| T10 TestCompileCacheClear_Artifacts | Use a different clear path or print success on failure. |

### Exact boundary acceptance

Run the cumulative runner with final invocation python3 - M3. It must print every M1-M3 pass.
T10 must reuse the fresh test binary, not a PATH-installed binary. This is sandbox-viable.

## M4 — route integrity diagnostic and MCP regression

**Estimate:** 0.75 day; approximately 280 LOC including tests.  
**Dependency:** green M3 commit.  
**Commit:** one M4 commit after full named and compatibility gates.

### Files touched

- Modify internal/apiserver/module_entry.go to check local exported routes against iface exports
  before idempotent return and map publication.
- Modify internal/apiserver/module_entry_test.go for T11 and all specified controls.
- Modify tracked cmd/ailang/serve_api_mcp_surface_test.go for T12: fresh binary, isolated cache,
  bounded MCP stdio initialize/list/call, old-stamp restoration, one stale blob under fresh stamp,
  exact seven names, f7 result, and a subsequent verified hit.

### Day breakdown

- Add CACHE_ROUTE_IFACE_MISMATCH without changing drop or exposure-filter behavior; implement T11.
- Extend the bounded MCP fixture for T12, run all gates, and accept seven-tool success—not an
  error-only substitute.

### Named tests and mutation obligations

| ID / name | Mutation this test must kill |
|---|---|
| T11 TestRegisterModule_RouteIfaceMismatch | Retain silent missing-name skip or bypass validation on repeat registration. |
| T12 TestServeAPI_DivergentCacheTools | Trust manifest alone or omit one blob hash; error-only behavior instead of successful recompilation also fails. |

### Exact boundary acceptance

Run the cumulative runner with final invocation python3 - M4. It must print all 14 package/name
passes (T6 passes in two packages). Then run, with a 10-minute outer controller deadline per command:

~~~sh
go test ./internal/pipeline ./internal/loader ./internal/apiserver ./cmd/ailang \
  -count=1 -timeout=5m
make check-boundaries
make check-file-sizes
~~~

Finally run the design's AC-E2E block against a fresh /tmp/ailang-cache-reviewed with expected
count 7. T12 remains stronger because it also calls f7, poisons one blob under a fresh stamp, checks
the diagnostic, and proves a verified later hit.

M4 is **not sandbox-blind**: H10 ran the same build and bounded MCP-stdio machinery in the workspace
sandbox. It needs no network, socket, or PATH-installed binary; /tmp and test temporary directories
are writable. No milestone is marked sandbox-blind. Controller re-verification remains normal
release evidence, not compensation for missing sandbox coverage.

## Final success criteria

- All T1-T13 mutations have their named pass events: 14 package/name passes.
- Divergent current manifest plus old artifacts yields seven exact tools, working f7, and a later hit.
- Invalid/oversized artifacts are explicit misses; correct source still compiles; diagnostics stay
  on stderr and never contaminate program or MCP stdout.
- Exact parsed bytes determine disk and embedded keys; nil is not empty source.
- compile-clear removes manifest authorization and modules/ while preserving siblings and errors.
- Route/iface mismatch cannot silently publish, but error-only behavior does not replace repair.
- One commit per milestone; earlier named tests stay green at every boundary; final package,
  architecture, and file-size checks are green.
- No adversarial gob hardening, provenance scheme, or D-55 follow-up enters the diff.

## Handoff constraints

- Executor updates only progress fields in the ignored JSON; scope decisions stay in this Markdown.
- The controller owns git writes. The planner performs none.
- Any discrepancy against H1-H10 is a finding: record exact command/output and revise the plan
  rather than making acceptance vacuous.
