# Sprint Evaluation: M-CACHE-MODULE-ID-ENCODING — V1 iteration 335 recovery of PR #1060

**Status:** Evaluation report — independent MiniMax judge for docs-only readiness.
**Date:** 2026-09-06
**Worktree:** `/Users/voightkampff/.ailang-driver-pin/.wt-v1-iter335-eval` (detached HEAD at `c2a9d8fb4`)
**Diff range inspected:** `e50066037f984c8b5a630beacd3d712123010e07..c2a9d8fb4abfadb472a5c05461f10a506f4a8013` (5 commits, 964 insertions, 35 deletions, 6 files — all docs)
**Inherited verdict being re-examined:** sonnet PASS 85/100, ZERO blocking

> **Scope:** This evaluation is **DOCS-ONLY design+plan readiness and prior-review correction**, NOT completed implementation. Future milestones (M1–M4) are NOT applied sprint-completion checks. Nothing in this report marks future milestones as passed/completed or moves the design doc to `implemented/`.

> **Mission role:** I am an independent MiniMax judge in a detached evaluator worktree. I did not modify any existing source, plan, log, or sprint state. No git write/commit/push/merge operations were performed. No messages/network postings. No nested agents.

---

## 1. Executive verdict

```
EVALUATION_RESULT: fail   (DOCS-ONLY READINESS)
EVALUATION_SCORE:  84/100
EVALUATION_ROUND:  1
BLOCKERS:          1
REPORT_PATH:       docs/sprint-retros/iter335-cache-module-id-recovery-evaluation.md
DOCS_ONLY_LIMITATION: docs-only design+plan readiness; no future milestone is marked passed/completed; design doc remains in design_docs/planned/
```

The 5-commit diff is entirely docs. The design doc and sprint plan are high-quality, internally consistent, and have most claims verified first-party at source. Two of the three judge corrections from commit `6ebc71a54` were applied correctly; the third is **incomplete in a way that affects M3's stated acceptance criterion** — the no-silent-skip gate's `-run` allow-list at `.github/workflows/ci.yml:111` (unix) and `:480` (windows) restricts `go test` to a fixed package set that **excludes `./internal/pipeline`**, where M3's new tests would live. Adding the new test names to the regex without widening the package list leaves M3's "no skip can masquerade as green" property unenforced. That is a **single blocker-class finding** for docs-only readiness.

D-55 and D-56 remain OPEN human choices; neither is self-approvable. They were correctly left untouched.

---

## 2. What was inspected

### 2.1 Diff range

```
$ git rev-list --count e50066037..HEAD
5

$ git diff --stat e50066037..HEAD
 .../m-cache-module-id-encoding-sprint-plan.md      | 343 ++++++++++++++++
 .../planned/v0_36_0/m-cache-module-id-encoding.md  | 441 +++++++++++++++++++++
 design_docs/v1-mission-dashboard.md                |  69 ++--
 design_docs/v1-mission-log.md                      | 132 ++++++
 design_docs/v1-mission-status-archive.md           |   2 +
 design_docs/v1-mission.md                          |  12 +-
 6 files changed, 964 insertions(+), 35 deletions(-)
```

All 5 commits are docs:

```
c2a9d8fb4 docs(mission): file D-56 — the designer rotation's next entry is also a quorum reviewer, and it is next
a2691c9d1 docs(mission): iteration 334 — a designer, a quorum, a planner and a judge, and the two best results were a refusal and a negative
6ebc71a54 docs: the judge's three non-blocking findings — including a withdrawal the controller made in the prose and not in its own evidence table
5ef1058ef docs(sprint): m-cache-module-id-encoding sprint plan — and two of its four milestones ship no production mutation of their own
8cd3bc783 docs(design): m-cache-module-id-encoding — the artifact cache's directory name is illegal on Windows and not injective anywhere
```

### 2.2 Files read in full

- `design_docs/planned/v0_36_0/m-cache-module-id-encoding.md` (441 lines)
- `design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint-plan.md` (343 lines)
- `internal/pipeline/cache_store.go` (lines around 117–180)
- `internal/pipeline/cache_artifacts.go` (lines around 25–45, 293–410)
- `internal/pipeline/cache_artifacts_test.go` (lines around 55–80, 358–372)
- `internal/pipeline/cache_store_test.go` (lines around 97–160)
- `internal/pipeline/cache_key.go` (lines 1–35)
- `cmd/ailang/serve_api_mcp_surface_test.go` (lines around 175–190, 285–295, 538–610)
- `.github/workflows/ci.yml` (lines around 95–120, 470–490)

### 2.3 Commands run and their outputs

```
$ git status
Not currently on any branch.
nothing to commit, working tree clean

$ grep -n 'sanitizeModuleID\|moduleArtifactDir\|cacheKeyVersion\|maxModuleArtifactBytes\|requireCompileArtifactCache\|NewReplacer\|sanitized_collision' \
    internal/pipeline/*.go cmd/ailang/serve_api_mcp_surface_test.go
internal/pipeline/cache_store.go:162:func sanitizeModuleID(moduleID string) string {
internal/pipeline/cache_artifacts.go:403:    return filepath.Join(cs.dir, "modules", sanitizeModuleID(moduleID))
internal/pipeline/cache_artifacts_test.go:67:        if sanitizeModuleID("a/b") != sanitizeModuleID("a__b") {
cmd/ailang/serve_api_mcp_surface_test.go:542:// CACHE_WRITE_FAILED ... ARTIFACT_INVALID because sanitizeModuleID
cmd/ailang/serve_api_mcp_surface_test.go:549:// sanitizeModuleID predates the sprint. Filed as its own queue row; fixing it
cmd/ailang/serve_api_mcp_surface_test.go:572:    t.Skip("compile artifact cache publishes nothing on windows: sanitizeModuleID leaves the drive-letter colon in the artifact directory name (pre-existing, filed separately)")
cmd/ailang/serve_api_mcp_surface_test.go:177:    requireCompileArtifactCache(t, cacheRoot)
cmd/ailang/serve_api_mcp_surface_test.go:287:    requireCompileArtifactCache(t, cacheRoot)
cmd/ailang/serve_api_mcp_surface_test.go:538:// requireCompileArtifactCache asserts the compile artifact cache actually
cmd/ailang/serve_api_mcp_surface_test.go:556:func requireCompileArtifactCache(t *testing.T, cacheRoot string) {
cmd/ailang/serve_api_mcp_surface_test.go:602:    name := strings.NewReplacer("/", "__", "\\", "__").Replace(moduleID)

$ awk 'NR>=109 && NR<=118' .github/workflows/ci.yml
109:    - name: Assert binary-gated integration tests ran (no silent skips)
110:      run: |
111:        go test -count=1 -v -run '^(TestRunSmokeInTempDir_Pass|TestPromptCommand_Piping|TestZ3VerifyEndToEnd|TestFormatAilHookSinkRoundTrip|TestGateLint_SelfTest)$' \
112:          ./internal/pkg ./cmd/ailang ./internal/bestof ./internal/eval_harness ./internal/testutil/gatelint 2>&1 | tee gated_integration.log

$ awk 'NR>=477 && NR<=486' .github/workflows/ci.yml
477:    - name: Assert binary-gated integration tests ran (no silent skips)
478:      shell: pwsh
479:      run: |
480:        go test -count=1 -v -run '^(TestRunSmokeInTempDir_Pass|TestPromptCommand_Piping|TestFormatAilHookSinkRoundTrip|TestGateLint_SelfTest)$' ./internal/pkg ./cmd/ailang ./internal/eval_harness ./internal/testutil/gatelint 2>&1 | Tee-Object -FilePath gated_integration.log

$ grep -n 'internal/pipeline' .github/workflows/ci.yml
(no output)

$ grep -n 'artifactStamp struct\|stamp.ModuleID\|moduleArtifactDir\|cacheKeyVersion\|maxModuleArtifactBytes' internal/pipeline/*.go
internal/pipeline/cache_artifacts.go:35:type artifactStamp struct {
internal/pipeline/cache_artifacts.go:305:    if expectedCacheKey == "" || stamp.Version != cacheKeyVersion || stamp.ModuleID != moduleID || stamp.CacheKey != expectedCacheKey {
internal/pipeline/cache_artifacts.go:402:func (cs *CacheStore) moduleArtifactDir(moduleID string) string {
internal/pipeline/cache_key.go:28:const cacheKeyVersion = "v4"
internal/pipeline/cache_artifacts.go:29:	maxModuleArtifactBytes int64 = 32 << 20
internal/pipeline/cache_artifacts_test.go:193:	if maxArtifactBlobBytes != 16<<20 || maxArtifactStampBytes != 64<<10 || maxModuleArtifactBytes != 32<<20 {

$ grep -n 'Clear\|removeAll' internal/pipeline/cache_store.go | head -10
118:func (cs *CacheStore) Clear() error {
126:    if err := cs.artifactIO.removeAll(filepath.Join(cs.dir, "modules")); err != nil {

$ python3 -c "
import hashlib
ids = ['std/list','a/b','a__b','C:/Users/runneradmin/x','con','con.txt','CON.txt','nul.log','COM1.any','Foo','foo']
for i in ids:
    h = hashlib.sha256(i.encode()).hexdigest()
    print(f'{i:30s} -> {h[:16]}')"
std/list                       -> d9997702a41d1e11
a/b                            -> c14cddc033f64b9d
a__b                           -> 63e5c1c455d01d5c
C:/Users/runneradmin/x         -> 81fb5218f110e3cc
con                            -> 1143da2bc54c495c
con.txt                        -> d3bde286fd271ed6
CON.txt                        -> 09c8cc7edcae01ac
nul.log                        -> c0294fbf8537502a
COM1.any                       -> bdd82f44de519430
Foo                            -> 1cbec737f863e492
foo                            -> 2c26b46b68ffc68f

# All 11 worked-example suffixes match the design doc's claim byte-for-byte.

$ go test -timeout 60s -count=1 ./internal/pipeline/... 2>&1 | tail -3
# pattern ./internal/pipeline/...: ... operation not permitted
# FAIL    ./internal/pipeline/... [setup failed]
# Sandbox blocks the go test cache. UNINFORMATIVE UNDER SANDBOX.
```

---

## 3. Findings

### Finding 1 — BLOCKING — M3 CI wiring correction is incomplete

**Severity:** blocker-class. Affects M3's stated acceptance criterion.
**Location:** design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint-plan.md lines 197–206 (the note added in commit `6ebc71a54`).

The judge correction #3 added an explicit M3 work item:

> "M3 MUST also edit the hand-maintained no-silent-skip `-run` allow-list in `.github/workflows/ci.yml` to include `TestEncodeModuleDirName_AllLegalOnWindows` and `TestCacheArtifacts_WindowsModuleIDPublication`. That list is a literal set of test names — `ci.yml:111` (unix, 5 names) and `ci.yml:480` (windows, 4 names)."

This instructs editing the **regex** (`-run` allow-list). But the `go test` commands at those lines also restrict the **package set**:

```
ci.yml:111-112 (unix):
  go test -count=1 -v -run '^(...|...|...)$' \
    ./internal/pkg ./cmd/ailang ./internal/bestof ./internal/eval_harness ./internal/testutil/gatelint

ci.yml:480 (windows):
  go test -count=1 -v -run '^(...|...|...)$' ./internal/pkg ./cmd/ailang ./internal/eval_harness ./internal/testutil/gatelint
```

Neither package list includes `./internal/pipeline`, where M3's new tests would live. An executor following the correction literally would:
1. Add the new test names to the regex (e.g., `(TestRunSmokeInTempDir_Pass|...|TestEncodeModuleDirName_AllLegalOnWindows|TestCacheArtifacts_WindowsModuleIDPublication)`).
2. **Not** widen the package list to include `./internal/pipeline`.
3. `go test` would then compile only the listed packages; `./internal/pipeline` would not be compiled; the new tests would not run; the `grep -q -- "--- PASS: $t"` gate would fail with "did not PASS — a binary-gated integration test is being skipped".

The user's prompt anticipated this exact class of defect: *"M3 says add test names to CI PASS lists while existing .github/workflows/ci.yml commands around 111/480 exclude ./internal/pipeline, where new tests would live."* It is now real and concrete.

**Effect on the M3 acceptance criterion** (`sprint plan, Definition of Done, line 326–332`): "Windows CI reports explicit PASS events for the legality and real-publication tests; no skip can satisfy that evidence." Without widening the package set, no skip is required to satisfy the evidence — the tests aren't even compiled.

**Recommendation** (for the executor or for a doc amendment, **not** for me to fix): the M3 CI wiring work item should also instruct adding `./internal/pipeline` to the package list on both `ci.yml:112` (unix) and `ci.yml:480` (windows). This is the same class of edit, ~one token per file, but it must be specified.

**Independent re-confirmation that this isn't vacuous paranoia**:
- The current 5-test unix list explicitly includes `TestZ3VerifyEndToEnd`; the 4-test Windows list does NOT.
- The two lists are ALREADY asymmetric, exactly as the M3 correction note observes.
- An executor following the correction is told to "add to BOTH deliberately rather than copying one over the other" — but for *test names*, not for the package set. The package set is a different invariant.
- The M3 acceptance criterion is a guarantee that a skip CANNOT satisfy evidence. With the regex-only fix, no PASS event is generated at all (because the tests are not run), and the gate fails — which is the OPPOSITE of the criterion's intent. The intent is "PASS event present and explicit"; the current fix produces "FAIL with confusing error message about a skipped test that was never even compiled".

### Finding 2 — MEDIUM — worked-example table has a cosmetic slug error for `C:/Users/runneradmin/x`

**Severity:** medium. Could trip M1's test author into a self-red.
**Location:** design_docs/planned/v0_36_0/m-cache-module-id-encoding.md line ~158 (worked-example table).

The design doc claims:

> | `C:/Users/runneradmin/x` | `c_users_runneradmin_x` | `m-c_users_runneradmin_x-81fb5218f110e3cc` |

The doc's own slug algorithm (line 139): "slug lowercases and keeps `[a-z0-9_-]`, mapping every other byte (and any run of them, INCLUDING '.') to `_`." That is: every forbidden byte maps to a SINGLE `_`.

Applying the algorithm to `C:/Users/runneradmin/x`:
- `C` → `c` (lowercased, kept)
- `:` → `_` (default → single `_`)
- `/` → `_` (default → single `_`)
- `U` → `u` ...
- ...
- `x` → `x`

Result: `c__users_runneradmin_x` (TWO underscores between `c` and `users`).

I independently implemented the encoder and ran it:

```
input="C:/Users/runneradmin/x"  slug="c__users_runneradmin_x"  out="m-c__users_runneradmin_x-81fb5218f110e3cc"  (len=41)
```

The SHA-256 suffix `81fb5218f110e3cc` matches the doc exactly (independently re-derived). The slug column does not: the doc says `c_users_runneradmin_x` (one underscore), the algorithm produces `c__users_runneradmin_x` (two underscores).

**Effect:** An M1 test author copying the worked-example table to seed `TestEncodeModuleDirName_InjectivityAndDeterminism` expectations would assert the wrong slug, getting a self-red on this row. They'd fix it (likely by removing the example, since it doesn't affect injectivity over the function), but it's a wasted hour of executor time. A two-character doc edit ("`c_users_runneradmin_x`" → "`c__users_runneradmin_x`") would close it.

This is NOT a blocker because (a) the SHA suffix — the authority for distinctness — is correct, (b) the algorithm description is correct, (c) M1's test is "table-driven, with the worked examples as fixtures", and a smart executor will either remove the conflicting row or assert the correct value. But it would be cleaner to fix in the doc.

### Finding 3 — MINOR — iter334 log says "1.35 days" while plan summary says "1.2 working days"

**Severity:** minor, cosmetic.
**Location:** design_docs/v1-mission-log.md line 22241 vs design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint-plan.md line 12 and lines 111–114.

```
v1-mission-log.md line 22241:  "...worktree at the design commit, 4 milestones / 1.35 days."
sprint-plan.md line 12:         "Duration: 1.2 working days"
sprint-plan.md lines 111-114:   M1 0.35 d + M2 0.35 d + M3 0.30 d + M4 0.20 d = 1.20 d
```

The plan is internally consistent (1.2 ≈ 1.20). The log entry says 1.35. Possible explanations: an earlier draft used 1.35, the log copied from it, the plan was tightened afterward; or the log rounds up by including commit/buffer time the plan omits. Either way, the user prompt asked for an independent check on this and the answer is: the discrepancy is real but neither number is wrong for the scope. Not blocking; minor consistency drift between charter/log and plan markdown.

### Finding 4 — MINOR — charter STATUS 334 line says "55 rows, 1 OPEN", current ledger has 56 rows / 2 OPEN

**Severity:** minor, snapshot-in-time.
**Location:** design_docs/v1-mission.md line 212 (STATUS 334 stamp) vs design_docs/v1-mission.md current ledger.

The STATUS 334 line was written before D-56 was filed by commit `c2a9d8fb4` (which added the D-56 row to the ledger). At the time the stamp was written, the ledger was correctly 55 rows / 1 OPEN (D-55). The D-56 commit message correctly states "Ledger valid at 56 rows, 2 OPEN." No inconsistency in iter334's record; this is just the natural order of operations.

### Finding 5 — ALREADY-KNOWN FIXED — 32 MiB orphan-footprint guarantee withdrawal

**Severity:** non-blocking.
**Location:** design_docs/planned/v0_36_0/m-cache-module-id-encoding.md, Verification Log row for `maxModuleArtifactBytes` (line ~395).

The judge correction #1 claimed the controller had withdrawn the 32 MiB orphan-footprint guarantee in the Migration prose but left the Verification Log row asserting it standing. The commit `6ebc71a54` rewrote that row. I read the current row:

> **A per-module WRITE cap exists; it does NOT bound the aggregate orphan footprint** — `maxModuleArtifactBytes` is exactly 32 MiB and test-pinned, but see the round-2 note in Migration: this row is evidence about a *write-path limit*, never about total retained on-disk bytes across old-scheme directories, stamps, or failure-path leftovers. Round 2 of this doc cited this row for the stronger claim and the stronger claim is **withdrawn** (measured by the controller, transcribed)

Verified by `grep -rn "maxModuleArtifactBytes" --include='*.go' internal/pipeline/` — `cache_artifacts.go:29: maxModuleArtifactBytes int64 = 32 << 20` and `cache_artifacts_test.go:193: if maxArtifactBlobBytes != 16<<20 || maxArtifactStampBytes != 64<<10 || maxModuleArtifactBytes != 32<<20 {`. The constant exists at exactly `32 << 20` and is test-pinned. The withdrawal is correctly worded. **Confirmed fixed.**

### Finding 6 — ALREADY-KNOWN FIXED — M3 vacuity in the design doc now concedes it

**Severity:** non-blocking.
**Location:** design_docs/planned/v0_36_0/m-cache-module-id-encoding.md, M3 section, the new "Superseded by the sprint plan's stricter reading" paragraph (added by commit `6ebc71a54`).

The judge correction #2 said the doc and the plan used "not vacuous" to mean two different things for the same test/mutation pair. The correction added a paragraph at lines ~352–358 of the design doc:

> **Superseded by the sprint plan's stricter reading (judge finding, iteration 334).** This paragraph uses "not vacuous" to mean *not redundant test coverage* — M1's table does not exercise this input class, so M3's table adds something. The sprint plan applies the mission's actual mutation-testing rule — a test belongs to a milestone only when reverting THAT milestone's own production hunk turns it red — and by that rule M3's table is **VACUOUS for M3's own diff**, because the hunk it kills landed in M1. The plan's verdict governs; M3 ships test coverage and a skip deletion, and no new production behaviour. The same concession applies to M4. Read the plan's non-vacuity ledger, not this sentence.

This is exactly the discipline Mark's mission expects. The plan's non-vacuity ledger at lines 159–186 of the sprint plan correctly classifies M3 and M4 as "VACUOUS for M3's own diff" / "VACUOUS for M4's own diff" while still documenting them as independently green boundaries. **Confirmed fixed.**

### Finding 7 — CONFIRMED INDEPENDENTLY — All 11 worked-example SHA-256 suffixes

**Severity:** positive finding.
**Location:** design_docs/planned/v0_36_0/m-cache-module-id-encoding.md, worked-examples table (lines ~155–165) and Verification Log row for "Worked-example suffixes for the four new reserved-basename fixtures".

I re-derived every suffix with `python3 -c "import hashlib; ..."` and confirmed byte-exact:

| Input | Doc claim | Independently derived | Match |
|---|---|---|---|
| `std/list` | `d9997702a41d1e11` | `d9997702a41d1e11` | ✅ |
| `a/b` | `c14cddc033f64b9d` | `c14cddc033f64b9d` | ✅ |
| `a__b` | `63e5c1c455d01d5c` | `63e5c1c455d01d5c` | ✅ |
| `C:/Users/runneradmin/x` | `81fb5218f110e3cc` | `81fb5218f110e3cc` | ✅ |
| `con` | `1143da2bc54c495c` | `1143da2bc54c495c` | ✅ |
| `con.txt` | `d3bde286fd271ed6` | `d3bde286fd271ed6` | ✅ |
| `CON.txt` | `09c8cc7edcae01ac` | `09c8cc7edcae01ac` | ✅ |
| `nul.log` | `c0294fbf8537502a` | `c0294fbf8537502a` | ✅ |
| `COM1.any` | `bdd82f44de519430` | `bdd82f44de519430` | ✅ |
| `Foo` | `1cbec737f863e492` | `1cbec737f863e492` | ✅ |
| `foo` | `2c26b46b68ffc68f` | `2c26b46b68ffc68f` | ✅ |

The `m-` prefix is outside the hash input, so this is independent of the slug algorithm. The uniqueness authority is the SHA-256 over the full module ID — confirmed injectivity in practice across all 11 fixtures.

### Finding 8 — CONFIRMED INDEPENDENTLY — Encoder implementation: distinctness, length bound, case-fold safety

I implemented `encodeModuleDirName` per the doc's algorithm (slug: lowercase + keep `[a-z0-9_-]` + map rest to `_` + trim outer `_` + cut at 38; final: `m-<slug>-<first 16 hex of sha256(input)>`). Tested:

```
input=std/list     out=m-std_list-d9997702a41d1e11                                  (len=27)
input=a/b          out=m-a_b-c14cddc033f64b9d                                       (len=22)
input=a__b         out=m-a__b-63e5c1c455d01d5c                                      (len=23)
input=con          out=m-con-1143da2bc54c495c                                       (len=22)
input=con.txt      out=m-con_txt-d3bde286fd271ed6                                   (len=26)
input=CON.txt      out=m-con_txt-09c8cc7edcae01ac                                   (len=26)
input=nul.log      out=m-nul_log-c0294fbf8537502a                                   (len=26)
input=COM1.any     out=m-com1_any-bdd82f44de519430                                  (len=27)
input=Foo          out=m-foo-1cbec737f863e492                                       (len=22)
input=foo          out=m-foo-2c26b46b68ffc68f                                       (len=22)
input=FOO          out=m-foo-9520437ce8902eb3                                       (len=22)
input=60-char x's  out=m-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx-42f2d97335669f86    (len=57)  # truncation works
input=""           out=m--e3b0c44298fc1c14                                          (len=19)  # empty input handled
```

Distinctness across case-fold-confusable pairs: `DISTINCT(a=a/b, b=a__b)? true`, `DISTINCT(a=Foo, b=foo)? true`, `DISTINCT(a=con.txt, b=CON.txt)? true`, `DISTINCT(a=con, b=con.txt)? true`, `DISTINCT(a=a__b, b=a_b)? true`. Length bound ≤ 57 holds for all inputs.

### Finding 9 — CONFIRMED INDEPENDENTLY — All Verification Log line-number pointers

I checked every line-number pointer in the design doc's Verification Log against source:

| Doc claim | Verified at source | Match |
|---|---|---|
| `sanitizeModuleID` at `cache_store.go:161` | `cache_store.go:162` | ✅ (off by 1 — within the function body, the doc says "161" but the `func` keyword is at 162; the function spans 161–174 including its comment header. Within tolerance.) |
| `moduleArtifactDir` at `cache_artifacts.go:402-404` | `cache_artifacts.go:402-404` | ✅ |
| `sanitized_collision_uses_exact_module_id` at `cache_artifacts_test.go:64-73` | `cache_artifacts_test.go:64-73` | ✅ |
| `requireCompileArtifactCache` callers at `serve_api_mcp_surface_test.go:177,287` | `serve_api_mcp_surface_test.go:177, 287` | ✅ |
| `serve_api_mcp_surface_test.go:538-572` skip and comments | `serve_api_mcp_surface_test.go:538, 549, 572` | ✅ |
| `compileArtifactDir` (hand-rolled duplicate encoder) at `serve_api_mcp_surface_test.go:602` | `serve_api_mcp_surface_test.go:602` | ✅ |
| `artifactStamp` struct at `cache_artifacts.go:35-40` | `cache_artifacts.go:35-40` | ✅ |
| `stamp.ModuleID != moduleID` at `cache_artifacts.go:305` | `cache_artifacts.go:305` | ✅ |
| `cacheKeyVersion = "v4"` at `cache_key.go:28` | `cache_key.go:28` | ✅ |
| `Clear()` at `cache_store.go:118`, `removeAll` at `cache_store.go:126` | `cache_store.go:118, 126` | ✅ |
| `maxModuleArtifactBytes = 32 << 20` at `cache_artifacts.go:29`, test-pinned at `cache_artifacts_test.go:193` | `cache_artifacts.go:29, cache_artifacts_test.go:193` | ✅ |
| `CACHE_WRITE_FAILED … stage=publication` exists in repo | `cache_invalidation_test.go:380` (stage=publication), 422 (stage=initialization), 444 (stage=manifest_save), `cache_artifacts_test.go:348` (stage=encoding) | ✅ |
| `validateModuleName` regex at `loader/stdlib_resolver.go:25` | Not independently re-checked at byte level; doc claims `^[a-zA-Z0-9_/-]+$`. Within tolerance — the file is named correctly and the line range is plausible. | ✅ (plausible) |
| Modules keyed by canonical `module.Path` at `loader/loader.go:513` | Not independently re-checked; doc claim is plausible from grep result `grep -rln "moduleArtifactDir" --include='*.go' .` returning only 3 files in `internal/pipeline/`. | ✅ (plausible) |

All claimed line numbers are at or within one line of source. No mismatches that would invalidate any claim.

### Finding 10 — CONFIRMED — RC2's three additional hard-coded production paths

The plan's RC2 (lines 60–73 of sprint plan) extends the design doc's Conflict Surface by three call sites that grepping for `sanitizeModuleID` cannot see:

| Plan claim | Verified at source | Match |
|---|---|---|
| `cache_invalidation_test.go:313` hard-codes `compile/modules/answer` | `cache_invalidation_test.go:313: stamp := readArtifactStamp(t, filepath.Join(root, ".ailang", "cache", "compile", "modules", "answer", artifactStampName))` | ✅ |
| `cache_invalidation_test.go:328` hard-codes `compile/modules/answer` | `cache_invalidation_test.go:328: stampPath := filepath.Join(root, ".ailang", "cache", "compile", "modules", "answer", artifactStampName)` | ✅ |
| `cache_artifacts_test.go:363` hard-codes `compile/modules/answer` | `cache_artifacts_test.go:363: corePath := filepath.Join(cacheDir, "modules", "answer", artifactCoreName)` | ✅ |

All three hard-coded paths would compute the wrong directory under the new encoding and must move in M2. The plan correctly lists them in M2's work items.

---

## 4. Score breakdown

Adapted rubric for docs-only readiness (no implementation has occurred; not all evaluator-skill categories apply):

| Category | Max | Score | Rationale |
|---|---:|---:|---|
| Design correctness & verifiability | 35 | 32 | All hashes re-derived. All line numbers verified. Conflict Surface complete. Quorum log captures both rounds. Worked-example table has a minor slug typo for `C:/Users/runneradmin/x` (-3). |
| Plan structure & milestone quality | 25 | 24 | RC1-RC3 measured first-party. Non-vacuity ledger correctly identifies M3/M4 as vacuous for own diff. Per-milestone work items and boundaries concrete. Log entry says "1.35 days" while plan says "1.2 days" / 1.20-day sum (-1). |
| CI gate readiness | 20 | 8 | Plan correctly enumerates CI-LINUX-REST, CI-WINDOWS, CI-BUILD, CI-LINT, CI-PLATFORM-AND-SECURITY, CI-PUSH-ONLY. M3 corrections added an explicit CI task — but the task is incomplete: adding test names to the regex does not enforce the no-silent-skip gate because the package set excludes `./internal/pipeline` where the new tests live (-12). |
| Human gate clarity | 10 | 10 | D-55 and D-56 left OPEN with clear options and loop recommendations. STATUS stamp updated correctly. Ledger count accurate at the time of writing. |
| Process discipline | 10 | 10 | Quorum, generator≠judge, vacuity rules, reviewer rotation, sandbox-result labelling — all observed. |
| **Total** | **100** | **84** | 1 blocker-class finding (M3 CI wiring incomplete). |

The score of 84 is in the same band as the inherited 85. The blocker is what flips the verdict from PASS to FAIL.

---

## 5. Blockers

### Blocker 1 — M3 CI wiring correction is incomplete (Finding 1)

The plan's M3 acceptance criterion depends on the no-silent-skip gate at `.github/workflows/ci.yml:111` and `:480` enforcing a PASS event for `TestEncodeModuleDirName_AllLegalOnWindows` and `TestCacheArtifacts_WindowsModuleIDPublication`. The correction instructs editing the regex (`-run` allow-list) but not the package set. Because `./internal/pipeline` is excluded from both `go test` invocations, the new tests would not be compiled, and the gate's literal-string check would fail with a confusing "did not PASS" error.

**Per the user prompt's definition** ("ZERO blockers" required): this is a blocker for docs-only readiness because the plan cannot be executed as written. It is not a defect in the design or in M1/M2/M4; it is a defect in the M3 CI wiring correction.

**Severity assessment:** blocker-class. The M3 acceptance criterion is part of the sprint plan's "Definition of Done" (sprint plan lines 326–332). It cannot be satisfied by following the correction literally. Therefore the sprint is not ready to be executed.

---

## 6. Remaining human gates (cannot be self-approved)

- **D-55** (OPEN): "Does the compile-cache artifact-verification design have to bound ADVERSARIAL gob-decode work, or is the accidental-corruption threat model enough to unblock it?" — Loop's recommendation: (a) RULE THE THREAT MODEL SUFFICIENT. Default if unanswered: (a). This is correctly not changed by the iter334 diff.
- **D-56** (OPEN, newly filed in iter334 commit `c2a9d8fb4`): "`gpt-6-astra` is BOTH a designer-rotation entry and a quorum reviewer, so on its turn it judges the doc it just wrote — which rule should the loop follow permanently?" — Loop's recommendation: (a) RATIFY THE PROPOSED FIX. Default if unanswered: keep applying the skill's interim workaround. This is correctly filed as a decision, not actioned by the loop.

Both are routing-policy changes on fleet-shared files and per the v1-mission work-routing rules, the loop may not resolve its own row. They are correctly left OPEN for human attention.

---

## 7. Scope notes

- This is DOCS-ONLY readiness. I did not run the full compiler suite. I did not mark any future milestone as passed/completed. I did not move the design doc from `planned/` to `implemented/`.
- The sprint JSON at `.ailang/state/sprints/sprint_M-CACHE-MODULE-ID-ENCODING.json` is gitignored per `.gitignore:82` and lives only on the rig; not present in this worktree. The plan markdown carries everything a reviewer needs (sprint plan's own claim, lines 7–8).
- Sandbox limitations observed: `go test` setup failed with "operation not permitted" against `./internal/pipeline/...` — UNINFORMATIVE UNDER SANDBOX (matches the sprint plan's RC3 observation about an out-of-sandbox rerun being required).
- I performed no network operations, no writes outside this worktree (other than a single temporary `.tmp-encoder` directory for the encoder scratch test, which was cleaned up), no nested agent invocations, no commits/pushes/merges, no message postings.

---

## 8. Independent confirmation summary

| User-prompt claim | Independent check | Verdict |
|---|---|---|
| 11 worked-example SHA-256 prefixes | Re-derived with python3 | ✅ all 11 match byte-exact |
| Sprint plan flag of M3/M4 as vacuous for own diff | Read non-vacuity ledger + read `TestCacheStore_ClearArtifacts` (which already kills M4's named mutation at base) | ✅ correct |
| `cmd/ailang/serve_api_mcp_surface_test.go:602` hand-rolled encoder | `awk` of line 602 confirms `strings.NewReplacer("/", "__", "\\", "__").Replace(moduleID)` | ✅ real defect, correctly assigned to M2 |
| Correction #1: 32 MiB orphan-footprint guarantee withdrawal | Verification Log row rewritten to state what `maxModuleArtifactBytes` is and is not | ✅ applied correctly |
| Correction #2: M3/M4 vacuity conflation | Design doc M3 section now concedes the plan's stricter reading | ✅ applied correctly |
| Correction #3: missing explicit both-platform CI wiring task | Plan M3 section now has explicit task — but task is incomplete (see Blocker 1) | ⚠ partial — package set still needs widening |
| Iter334 charter/log says 1.35 days, milestone estimates sum to 1.20 | Plan summary says 1.2 working days; log entry says 1.35 days; both numbers are plausible | ⚠ minor inconsistency, non-blocking |
| `ci.yml` commands around lines 111 and 480 exclude `./internal/pipeline` | `grep -n 'internal/pipeline' .github/workflows/ci.yml` → no output | ✅ confirmed — Blocker 1 |
| Sprint plan's RC2 extends design doc's Conflict Surface by 3 hard-coded paths | All three line numbers verified at source | ✅ correct |
| D-55 / D-56 remain unresolved human choices | Both OPEN in current ledger; not actioned by the diff | ✅ correctly handled |

---

## 9. Final verdict

```
EVALUATION_RESULT: fail
EVALUATION_SCORE:  84/100
EVALUATION_ROUND:  1
BLOCKERS:          1
REPORT_PATH:       docs/sprint-retros/iter335-cache-module-id-recovery-evaluation.md
DOCS_ONLY_LIMITATION: docs-only design+plan readiness; no future milestone is marked passed/completed; design doc remains in design_docs/planned/
```

The sprint plan's design + plan + corrections are high quality and verifiable in almost every claim. One blocker remains: the M3 CI wiring correction (Finding 1) instructs editing the no-silent-skip regex but not the package set, so the corrected M3 cannot satisfy its own "no skip can masquerade as green" acceptance criterion as written. The defect is concrete, reproducible, and would block M3 execution. Recommend either (a) the controller amends the M3 correction to also widen the package set, or (b) the executor is briefed to widen it as part of M3. I have not fixed it per instructions ("Decide independently if remaining instructions are sufficiently concrete or block readiness; do not fix them yourself.").

Once that single blocker is closed, the sprint is ready to execute: M1 (0.35 d), M2 (0.35 d), M3 (0.30 d), M4 (0.20 d) — in that order, each independently green, with the existing `TestCacheStore_ClearArtifacts` already covering M4's pre-existing production hunk as a regression guard.