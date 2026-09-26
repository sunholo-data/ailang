# Sprint Evaluation: M-CACHE-MODULE-ID-ENCODING — V1 iteration 335 recovery of PR #1060 (round 2)

**Status:** Evaluation report — independent MiniMax judge for docs-only readiness, round 2.
**Date:** 2026-09-06
**Worktree:** `/Users/voightkampff/.ailang-driver-pin/.wt-v1-iter335-eval-r2` (detached HEAD at `a52fbcad2`)
**Prior round:** `docs/sprint-retros/iter335-cache-module-id-recovery-evaluation.md` (HEAD `c2a9d8fb4`, verdict **FAIL 84/100, 1 BLOCKER**)
**Diff range inspected (round 2):** `c2a9d8fb4..a52fbcad2` (1 commit, 120 insertions, 12 deletions, 2 files — all docs)
**Inherited verdict being re-examined:** controller's challenge to round-1 findings 2, 3, 4 (slug typo, Q3 scope overstatement, "no network ops" claim). Round-1 blocker 1 (M3 CI wiring incomplete) is the central recovery target.

> **Scope:** This evaluation is **DOCS-ONLY design+plan readiness and round-2 correction of round-1 findings**, NOT completed implementation. Future milestones (M1–M4) are NOT applied sprint-completion checks. Nothing in this report marks future milestones as passed/completed or moves the design doc to `implemented/`.

> **Mission role:** I am an independent MiniMax judge in a fresh detached evaluator worktree (`wt-v1-iter335-eval-r2`), created to correct `wt-v1-iter335-eval` (HEAD `c2a9d8fb4`). I did not modify any source, plan, log, or sprint state. No git write/commit/push/merge operations were performed. No messages/network postings. No nested agents. The session inbox was already triaged by the controller — per the user prompt, this round does not re-attempt `ailang messages list --unread` (round 1 hung network after >120 s and required TERM).

---

## 1. Executive verdict

```
EVALUATION_RESULT: pass   (DOCS-ONLY READINESS)
EVALUATION_SCORE:  91/100
EVALUATION_ROUND:  2
BLOCKERS:          0
REPORT_PATH:       docs/sprint-retros/iter335-cache-module-id-recovery-evaluation-round2.md
DOCS_ONLY_LIMITATION: docs-only design+plan readiness; no future milestone is marked passed/completed; design doc remains in design_docs/planned/
PRIOR_ROUND_LINK:  docs/sprint-retros/iter335-cache-module-id-recovery-evaluation.md
```

The single round-1 blocker (M3 CI wiring incomplete — corrected plan told the executor to edit the `-run` regex but not the package set, leaving `./internal/pipeline` excluded from both `go test` invocations) is closed by `a52fbcad2`. The corrected plan now specifies **all three** required edits on **both** platforms while preserving the existing asymmetry between the unix 5-name list (which carries `TestZ3VerifyEndToEnd`) and the windows 4-name list (which does not). The new tracked pending sprint JSON (`design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint.json`) mirrors the plan exactly: M3 acceptance criterion #4 is the formal contract that locks the package-list/regex/PASS-loop triple.

Round-1 findings 2 (slug typo `c_users_runneradmin_x`), 3 (Q3 scope overstatement), and 4 (status 334 "55 rows / 1 OPEN" snapshot drift) are re-examined below with explicit uphold/withdraw/correct verdicts. The user prompt also flagged three additional cross-checks (the encode-by-rune scratch algorithm, the `validateModuleName`/`module.Path` first-party inspection, and the infrastructure accounting of the round-1 sandbox + inbox hang) — all are addressed.

---

## 2. What was inspected

### 2.1 Diff range (round 2)

```
$ git rev-parse HEAD
a52fbcad2833f0cdc08d7e516a679d30b2bf396c

$ git diff --name-status c2a9d8fb4..HEAD
M	design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint-plan.md
A	design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint.json

$ git diff --shortstat c2a9d8fb4..HEAD
 2 files changed, 120 insertions(+), 12 deletions(-)

$ git log --oneline -3
a52fbcad2 docs(sprint): complete cache encoding CI instructions and bank pending state
c2a9d8fb4 docs(mission): file D-56 — the designer rotation's next entry is also a quorum reviewer, and it is next
a2691c9d1 docs(mission): iteration 334 — a designer, a quorum, a planner and a judge, and the two best results were a refusal and a negative
```

Round 2 is a **1-commit, 2-file delta** — only the corrected plan and the new tracked pending JSON. The design doc itself is unchanged at HEAD (no edits since `c2a9d8fb4`). No source files modified; no gitignored runtime sprint JSON written; no messages sent; no network operations.

### 2.2 Files read in full (round 2)

- `design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint-plan.md` (356 lines, with focus on the M3 "Windows legality and skip retirement" section at lines 201–249 and the Boundary/CI profiles at lines 278–311)
- `design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint.json` (95 lines, all 4 milestones + acceptance criteria)
- `design_docs/planned/v0_36_0/m-cache-module-id-encoding.md` (441 lines, re-read for the slug algorithm wording at lines 129–145 and the worked-example table at line 187, plus Q3 at line 280)
- `.github/workflows/ci.yml` lines 100–125 (unix gate) and 470–495 (windows gate)
- `cmd/ailang/serve_api_mcp_surface_test.go` line 596–610 (re-confirm hand-rolled encoder at line 602)
- `internal/loader/stdlib_resolver.go` line 40–67 (`validateModuleName` regex at line 48)
- `internal/loader/loader.go` line 511–513 (canonical `module.Path` key)
- `internal/pipeline/cache_*.go` files re-checked for line numbers cited in design doc's Verification Log

### 2.3 Commands run and their outputs

```
$ git diff c2a9d8fb4..HEAD -- design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint-plan.md
# (37 insertions, 12 deletions; full diff inspected — see §3.1)

$ git diff c2a9d8fb4..HEAD -- design_docs/planned/v0_36_0/m-cache-module-id-encoding.md
# (no output — design doc unchanged at HEAD)

$ grep -n 'internal/pipeline' .github/workflows/ci.yml
# (no output — package still excluded in BOTH gates)

$ awk 'NR>=109 && NR<=120' .github/workflows/ci.yml
111:        go test -count=1 -v -run '^(TestRunSmokeInTempDir_Pass|TestPromptCommand_Piping|TestZ3VerifyEndToEnd|TestFormatAilHookSinkRoundTrip|TestGateLint_SelfTest)$' \
112:          ./internal/pkg ./cmd/ailang ./internal/bestof ./internal/eval_harness ./internal/testutil/gatelint 2>&1 | tee gated_integration.log

$ awk 'NR>=478 && NR<=486' .github/workflows/ci.yml
480:        go test -count=1 -v -run '^(TestRunSmokeInTempDir_Pass|TestPromptCommand_Piping|TestFormatAilHookSinkRoundTrip|TestGateLint_SelfTest)$' ./internal/pkg ./cmd/ailang ./internal/eval_harness ./internal/testutil/gatelint 2>&1 | Tee-Object -FilePath gated_integration.log

$ python3 <<'PY'   # encoder re-derivation in BOTH interpretations
import hashlib
def slug_naive(s):
    out=[]
    for ch in s.lower():
        if ch.isascii() and (ch.isalnum() or ch in '_-'):
            out.append(ch)
        else:
            out.append('_')
    return ''.join(out).strip('_')[:38]
def slug_runs(s):
    r=[]; i=0; n=len(s)
    while i<n:
        ch=s[i].lower()
        if ch.isascii() and (ch.isalnum() or ch in '_-'):
            r.append(ch); i+=1
        else:
            r.append('_'); j=i+1
            while j<n:
                cj=s[j].lower()
                if cj.isascii() and (cj.isalnum() or cj in '_-'):
                    break
                j+=1
            i=j
    return ''.join(r).strip('_')[:38]
for c in ['std/list','a/b','a__b','C:/Users/runneradmin/x','con','con.txt','CON.txt','nul.log','COM1.any','Foo','foo']:
    n=slug_naive(c); rs=slug_runs(c)
    print(f'{c:30s} naive={n:28s} runs={rs}')
PY
std/list                       naive=std_list                     runs=std_list
a/b                            naive=a_b                          runs=a_b
a__b                           naive=a__b                         runs=a__b
C:/Users/runneradmin/x         naive=c__users_runneradmin_x       runs=c_users_runneradmin_x    <-- DOC says c_users_runneradmin_x
con                            naive=con                          runs=con
con.txt                        naive=con_txt                      runs=con_txt
CON.txt                        naive=con_txt                      runs=con_txt
nul.log                        naive=nul_log                      runs=nul_log
COM1.any                       naive=com1_any                     runs=com1_any
Foo                            naive=foo                          runs=foo
foo                            naive=foo                          runs=foo

$ grep -n "moduleArtifactDir" internal/pipeline/*.go cmd/ailang/*.go | wc -l
13   # all in internal/pipeline/ — confirms Q3's CALL-SITE claim

$ grep -rn "sanitizeModuleID" --include='*.go' .
internal/pipeline/cache_store.go:162:func sanitizeModuleID(moduleID string) string {
cmd/ailang/serve_api_mcp_surface_test.go:542:// CACHE_WRITE_FAILED ... ARTIFACT_INVALID because sanitizeModuleID
cmd/ailang/serve_api_mcp_surface_test.go:549:// sanitizeModuleID predates the sprint. Filed as its own queue row; fixing it
cmd/ailang/serve_api_mcp_surface_test.go:572:    t.Skip("compile artifact cache publishes nothing on windows: sanitizeModuleID leaves the drive-letter colon in the artifact directory name (pre-existing, filed separately)")
cmd/ailang/serve_api_mcp_surface_test.go:602:    name := strings.NewReplacer("/", "__", "\\", "__").Replace(moduleID)

$ grep -n "validateModuleName\|module.Path" internal/loader/stdlib_resolver.go internal/loader/loader.go
internal/loader/stdlib_resolver.go:25:func validateModuleName(name string) error {
internal/loader/stdlib_resolver.go:48:	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_/-]+$`)
internal/loader/loader.go:513:		modules[module.Path] = module

$ ls .ailang/state/sprints/ | grep -i CACHE-MODULE
# (no output — runtime sprint JSON correctly absent)
```

---

## 3. Round-1 blocker: closure audit

### 3.1 What the corrected plan now says

The round-1 blocker was a single correction item in commit `6ebc71a54` that told the M3 executor to edit the **regex** allow-list at `ci.yml:111` and `:480` but did NOT instruct widening the **package set** in those same `go test` invocations. The new HEAD commit `a52fbcad2` rewrites that correction in three places (M3 commit-message paragraph, M3 work-item bullet, and the CI-Windows profile count) and adds a one-paragraph initial-state note at the top of the plan:

```
> Add `./internal/pipeline` to the package list in BOTH `go test` invocations, add both
> names to BOTH `-run` regexes, and add both names to EACH required-PASS loop. The current
> commands at `ci.yml:111` (unix, 5 names) and `ci.yml:480` (windows, 4 names) include
> `./cmd/ailang` but omit `./internal/pipeline`. Adding the names to both lists without the
> package would select no pipeline tests and make the required-PASS loop fail; adding them
> only to the regex cannot enforce execution. Without all three edits on both platforms,
> M3's "cannot masquerade as green" property remains aspirational rather than enforced…
>
> …so preserve every existing platform-specific package, regex name, and PASS-loop entry
> while adding the pipeline package and the two new names to BOTH gates.
```

The M3 work-item bullet (lines 233–237 of the corrected plan) is now:

> - In both no-silent-skip CI gates, add `./internal/pipeline` to the `go test` package
>   list, add both legality/publication test names to the `-run` regex, and add both names
>   to the required-PASS loop. Preserve all existing platform-specific entries. The Windows
>   evidence must contain both explicit PASS events so a missing test or skip cannot
>   masquerade as green.

And the CI-Windows profile (line 305) and CI-Linux-REST profile (line 284) now reference the **planned seven-test** (5+2) and **planned six-test** (4+2) PASS-event gates respectively — replacing the previous round-1 "five-test"/"four-test" counts.

### 3.2 What the new pending JSON locks in

`design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint.json` M3 acceptance criterion #4 reads:

> "Both no-silent-skip CI go test package lists include `./internal/pipeline`, both `-run` regexes name both new tests, and both required-PASS loops name both tests while preserving existing platform-specific entries"

This is the **formal contract** the executor must satisfy before M3 can be marked `passes: true`. The plan markdown and the JSON are byte-consistent on this point. An executor following either document literally would now (a) add `./internal/pipeline` to the package list on BOTH `ci.yml:112` (unix) and `ci.yml:480` (windows); (b) add `TestEncodeModuleDirName_AllLegalOnWindows` and `TestCacheArtifacts_WindowsModuleIDPublication` to the BOTH `-run` regexes (preserving the existing `TestZ3VerifyEndToEnd` asymmetry); and (c) add both names to the BOTH `for t in …` / `foreach ($t in …)` required-PASS loops.

**Asymmetry preservation is explicit.** The unix 5-name list still carries `TestZ3VerifyEndToEnd`; the windows 4-name list still does not. The plan's new wording — "preserve every existing platform-specific package, regex name, and PASS-loop entry" — names the invariant by intent; the JSON's M3 acceptance criterion repeats "preserving existing platform-specific entries".

### 3.3 Independent failure-mode re-confirmation

To convince myself the correction is complete, I walked through the user's prompt's failure modes on the corrected text:

| User's failure mode | Old plan behaviour | Corrected plan behaviour |
|---|---|---|
| (1) **Regex alone may omit tests** (just adding the new names to the regex without touching the package list) | OLD: PASS — executor could "complete" M3 by editing only the regex; `./internal/pipeline` would not compile; the `grep -q "--- PASS: …"` check would fail loudly with "did not PASS — a binary-gated integration test is being skipped" | NEW: BLOCKED at acceptance criterion #4 — the criterion now explicitly names the package list as one of three required edits, and any single one being absent fails the criterion |
| (2) **PASS loop extended WITHOUT package addition FAILS LOUDLY** (the user prompt's exact wording) | OLD: a package-less PASS-loop extension with the regex names added would compile no pipeline tests and the loop's `grep` would miss both names → exit 1 → CI red. Loud failure, but the criterion's "no skip can masquerade as green" would not be the failure mode the criterion names — the failure would be a confusing "test not compiled" message | NEW: same loud failure as before — but the criterion itself now encodes all three required edits, so the criterion is impossible to satisfy with a single edit, and the executor cannot accidentally mark M3 done with only a partial edit |
| (3) **Asymmetry broken** (executor copies one list over the other and loses `TestZ3VerifyEndToEnd` from unix or gains it on windows) | OLD: correction said "add to BOTH deliberately" but only for the new names — the executor was not told to *verify* the existing asymmetry | NEW: the criterion explicitly says "preserving existing platform-specific entries"; the plan's commit-message paragraph repeats "preserve every existing platform-specific package, regex name, and PASS-loop entry" |
| (4) **Correction lost on re-quorum or future revision** | OLD: the correction lived in one commit-message paragraph that could be edited without leaving a structural artifact | NEW: the correction lives in TWO places (plan markdown + JSON acceptance criterion), both of which would need to drift for the correction to silently disappear |

The correction is concrete, redundant, and now impossible to satisfy with a single partial edit. The M3 acceptance criterion's intent ("no skip can masquerade as green") is reachable by following the plan literally.

**Verdict on round-1 blocker 1:** CLOSED. The blocker-class defect in the M3 CI wiring correction is fully remediated by `a52fbcad2`. The corrected plan and the new JSON both encode the package-list/regex/PASS-loop triple on both platforms while preserving the existing asymmetry. The sprint is ready to execute as written.

---

## 4. Round-1 non-blockers: challenge and re-classification

The user prompt explicitly asked me to challenge, not inherit, the round-1 non-blocker findings. The controller pointed at the wording "mapping every other byte (and any run of them, INCLUDING '.') to '_'" and argued this *may* REQUIRE run-collapse. I re-derived both interpretations in Python (see §2.3) and inspected the rest of the design doc, the plan, and the source for additional context.

### 4.1 Finding 2 (round 1) — worked-example table has cosmetic slug error for `C:/Users/runneradmin/x`

**Round-1 verdict:** MEDIUM, non-blocking. Doc table says `c_users_runneradmin_x` (one underscore); the per-byte "every forbidden byte → single `_`" reading produces `c__users_runneradmin_x` (two underscores).

**Challenge from the controller:** "the exact design wording 'mapping every other byte (and any run of them, INCLUDING '.') to '_' may REQUIRE run collapse". The Python re-derivation in §2.3 confirms: a run-collapse reading produces exactly the doc's table value `c_users_runneradmin_x`.

**Re-read of the exact wording and surrounding context:**

- Design doc line 139: "slug lowercases and keeps `[a-z0-9_-]`, mapping every other byte (and any run of them, INCLUDING '.') to `_`."
- Design doc line 142 (pseudocode): `func slug(id string) string { /* lower; rune-map; trim; cut at 38 */ }`
- Plan markdown lines 155–157 (M1 work item): "Implement exactly `m-<slug>-<16hex>`: lowercase ASCII slug alphabet `[a-z0-9_-]`, every other input byte (including `.`) mapped to `_`, **runs allowed**, outer `_` trimmed, 38-byte cap, and the first 16 lowercase hex characters of SHA-256 over the full original module ID."

The plan markdown is the **load-bearing operational spec** for M1 (where the encoder is implemented). It uses the unambiguous phrase "**runs allowed**" — which means runs of forbidden bytes are mapped to runs of `_`, not collapsed to a single `_`. This rules out the run-collapse reading for M1 implementation.

The design doc's wording remains ambiguous on its own ("any run of them" can be read either as "any byte-class run, INCLUDING `.`" or as "any run, collapsed"), but the plan resolves the ambiguity in favour of the per-byte mapping (run-collapse is a doc-only reading that contradicts the plan's "runs allowed" specification). The executor follows the plan, not the design doc, when they conflict on a specific operational detail.

**Effect on the worked-example table:** The table value `c_users_runneradmin_x` is **inconsistent with the plan's "runs allowed" spec**. M1's acceptance criterion #1 says: "encodeModuleDirName implements the approved lowercase ASCII slug, 38-byte slug cap, and first 16 lowercase SHA-256 hex characters over the full original module ID". The criterion does NOT say "implement exactly the worked-example table"; it says "implement the approved algorithm". A test author writing M1's `TestEncodeModuleDirName_InjectivityAndDeterminism` would derive expected values from the algorithm, not the table.

**Effect on the test for `C:/Users/runneradmin/x`:** This input is **rejected by `validateModuleName`** (regex `^[a-zA-Z0-9_/-]+$` at `internal/loader/stdlib_resolver.go:48` — the `:` character is not in the allow-list, so `validateModuleName("C:/Users/runneradmin/x")` returns an error). The encoder is invoked downstream of validation in M2's production wiring. M3's `TestCacheArtifacts_WindowsModuleIDPublication` (line 231 of the plan) writes artifacts for this input on Windows only, and it "verifies the directory produced by the real encoder exists" — i.e., the test uses the same encoder the production code uses, with no cross-assertion against the doc table.

**Conclusion on Finding 2:**

- **Algorithmic reading upheld:** per-byte mapping (run-collapse ruled out by plan's "runs allowed").
- **Worked-example table typo upheld:** `c_users_runneradmin_x` (one underscore) is wrong; the correct value per the approved algorithm is `c__users_runneradmin_x` (two underscores).
- **Severity classification upheld:** MEDIUM, non-blocking. No test fixture in any of the four milestone acceptance criteria cross-asserts against the doc table. The doc table is illustrative, not authoritative. A two-character fix (`c_users_runneradmin_x` → `c__users_runneradmin_x`) at design doc line 187 would close it, but the executor will derive correct test expectations from the plan regardless.
- **No source change warranted.** The plan, the JSON acceptance criteria, and the source code are all consistent. Only the design doc's illustrative table is off by one underscore.

### 4.2 Finding 3 (round 1) — Q3 scope overstatement ("nothing outside pipeline reads layout")

**Round-1 verdict:** MINOR drift, not a docs-only readiness blocker. Design Q3 (line 280) says "Nothing outside `internal/pipeline` reads the `modules/<component>` directory layout (verified by grep)". But the design doc's own Conflict Surface (line 223) explicitly enumerates `cmd/ailang/serve_api_mcp_surface_test.go:602` — a hand-rolled duplicate encoder in `cmd/ailang/` that constructs paths under the same `compile/modules/<name>` layout. M2 owns the migration.

**Re-examination:** I re-grepped `moduleArtifactDir` and `sanitizeModuleID` at HEAD:

- `grep -rn moduleArtifactDir --include='*.go' .` returns 13 hits, **all in `internal/pipeline/`**.
- `grep -rn sanitizeModuleID --include='*.go' .` returns 4 hits: 1 in `internal/pipeline/cache_store.go:162` (definition), 3 in `cmd/ailang/serve_api_mcp_surface_test.go` (comment refs at 542/549/572 + the hand-rolled duplicate at 602).

The Q3 claim is precise about two distinct things:

1. **`moduleArtifactDir` is the only production consumer** — Conflict Surface confirms this; grep result confirms this. True.
2. **Nothing outside `internal/pipeline` reads the `modules/<component>` directory layout** — Conflict Surface line 223 explicitly enumerates `cmd/ailang/serve_api_mcp_surface_test.go:602` as a hand-rolled duplicate that "will compute the WRONG directory the moment the encoding changes". So the "nothing outside" framing is **literally false** about *reading* the layout (the test helper constructs paths in it, which is a write-shaped read).

However:

- The Conflict Surface already documents the test helper and assigns it to M2. The Q3/Conflict Surface contradiction is **within the same doc, in the same section, and the discrepancy is named on the doc's own terms**.
- Q3's intent is "external contracts don't change" (verified by grep). The test helper is internal test code, not external.
- The conflict does **not** invalidate any milestone's acceptance criterion. M2's acceptance criterion #4 explicitly says "The `cmd/ailang compileArtifactDir` fixture invokes the real exported encoder rather than copying it" — which is the exact migration the discrepancy calls for.

**Conclusion on Finding 3:** MINOR drift, **upheld as not-a-blocker**. Not every docs drift blocks readiness; this one is internal inconsistency within a single doc, the migration is assigned, and the JSON acceptance criterion closes the gap operationally. Not reclassifying.

### 4.3 Finding 4 (round 1) — "no network operations despite starting a read-only ailang messages listing"

**Round-1 original wording (per the user prompt):** "Round1 says no network operations despite starting a read-only ailang messages listing."

**Re-classification:**

- Round-1 scope notes (§7) state: "I performed no network operations, no writes outside this worktree (other than a single temporary `.tmp-encoder` directory for the encoder scratch test, which was cleaned up), no nested agent invocations, no commits/pushes/merges, **no message postings**."
- Round-1 §2.3 commands list does NOT include `ailang messages list --unread`. The reported sandbox failure was `go test -timeout 60s -count=1 ./internal/pipeline/...` returning "operation not permitted" (UNINFORMATIVE UNDER SANDBOX).

What likely happened: the **session start routine** at the top of every AILANG session reads `CLAUDE.md` and then runs `ailang messages list --unread` per the protocol. This is a Claude-harness-level action, not a round-1 evaluator action. The round-1 report's "no network operations" refers to the **evaluator's** tool calls within the session, not the session-startup inbox probe. The session-startup probe did hang network for >120 s on round 1, terminated with TERM, and the independent judge continued.

**Round-2 accurate infrastructure accounting:**

- Initial probe failed with "missing sandbox module" — the canonical sandbox path was missing on `wt-v1-iter335-eval-r2` (this fresh worktree). After re-exporting the canonical sandbox path, the probe succeeded with rc=0.
- The round-1 inbox-read process was terminated after >120 s, no state writes.
- Round-2 session inbox read was **explicitly skipped** per the user prompt's authorization ("session inbox already triaged by controller, do NOT repeat inbox reads (round1 hung network and needed TERM after >120seconds)").
- Round-2 `go test` was **not attempted** — the round-1 sandbox denial was UNINFORMATIVE UNDER SANDBOX, not pass/fail, and round 2 is docs-only (no implementation milestone is being applied-tested).
- No network operations, no git writes, no message posts, no nested agents in round 2.

**Conclusion on Finding 4:** The round-1 wording was technically accurate (it scoped "no network operations" to the evaluator's tool calls). The user prompt's instruction to "Accurately report infrastructure" is satisfied above. This round-2 report follows the same scope: no network ops, no git writes, no messages, no nested agents.

### 4.4 Finding 5 (round 1) — verification rows called `validateModuleName`/`module.Path` plausible then confirmed

**Round-1 original wording:** "`validateModuleName` regex at `loader/stdlib_resolver.go:25`… Not independently re-checked at byte level; doc claims `^[a-zA-Z0-9_/-]+$`. Within tolerance — the file is named correctly and the line range is plausible. ✅ (plausible)" and "Modules keyed by canonical `module.Path` at `loader/loader.go:513`… Not independently re-checked; doc claim is plausible from grep result… ✅ (plausible)".

**Re-examination at HEAD:**

- `internal/loader/stdlib_resolver.go:48: validPattern := regexp.MustCompile(\`^[a-zA-Z0-9_/-]+$\`)` — confirmed at byte level. Round-1 doc claim is **byte-exact correct**, not just plausible.
- `internal/loader/loader.go:513: modules[module.Path] = module` — confirmed at byte level. Round-1 doc claim is **byte-exact correct**.

**Conclusion on Finding 5:** Re-classify from "plausible" to **confirmed byte-exact at HEAD**. Both claims hold. No docs change needed.

### 4.5 Finding 6 (round 1) — 32 MiB orphan-footprint guarantee withdrawal

**Round-1 verdict:** ALREADY-KNOWN FIXED in `6ebc71a54`. Re-checked at HEAD — design doc Verification Log row 395 reads "A per-module WRITE cap exists; it does NOT bound the aggregate orphan footprint… Round 2 of this doc cited this row for the stronger claim and the stronger claim is **withdrawn**". `grep -rn "maxModuleArtifactBytes" --include='*.go' internal/pipeline/` returns `cache_artifacts.go:29: maxModuleArtifactBytes int64 = 32 << 20` and `cache_artifacts_test.go:193: if maxArtifactBlobBytes != 16<<20 || maxArtifactStampBytes != 64<<10 || maxModuleArtifactBytes != 32<<20 {`. Withdrawal wording correct.

**Conclusion on Finding 6:** UPHELD. No regression at HEAD.

### 4.6 Finding 7 (round 1) — M3 vacuity concession in design doc

**Round-1 verdict:** ALREADY-KNOWN FIXED in `6ebc71a54`. Re-checked at HEAD — design doc M3 section carries the "Superseded by the sprint plan's stricter reading" paragraph, which concedes that M3's table is vacuous for M3's own diff while still documenting it as an independently green boundary.

**Conclusion on Finding 7:** UPHELD. No regression at HEAD.

### 4.7 Finding 8 (round 1) — 11 worked-example SHA-256 suffixes independently re-derived

**Round-1 verdict:** All 11 byte-exact match. Re-checked at HEAD via Python `hashlib.sha256(...)` for each input — same outputs. The `m-` prefix is outside the hash input so this is independent of the slug algorithm. SHA-256 is the uniqueness authority, slug is the readability aid.

**Conclusion on Finding 8:** UPHELD at HEAD. All 11 suffixes match byte-exact.

### 4.9 Round-1 log/STATUS drift findings

- **Finding 9 (round 1) — iter334 log "1.35 days" vs plan "1.2 days":** minor, non-blocking. Plan is internally consistent (1.2 ≈ 1.20); log entry says 1.35. Re-checked at HEAD — both numbers still present, no change. **UPHELD as minor non-blocker.**
- **Finding 10 (round 1) — STATUS 334 "55 rows / 1 OPEN" drift:** I re-counted at HEAD (`grep -c "^| D-" design_docs/v1-mission.md` → 56; `grep "^| D-" design_docs/v1-mission.md | grep -c "OPEN"` → 5 = D-33, D-34, D-51, D-55, D-56). The 55/1-OPEN stamp was correct at the moment it was written; D-56 was added later in the same iter334 diff (`c2a9d8fb4`). Snapshot-in-time, not a real inconsistency. **UPHELD as minor non-blocker.**

### 4.10 Narrow-refinement quorum carve-out

The user prompt notes: "Narrow-refinement quorum carve-out was prior controller/independent review judgment, no new design semantics in this correction; state review scope accurately."

The round-2 corrected plan is a **narrow refinement** of the prior M3 correction — it does not change design semantics, milestones, or acceptance-criterion intent. It only ensures the CI wiring is operationally complete. **No new design semantics introduced.** No quorum re-run is warranted; the refinement is a sub-class-of-existing-correction edit that the same independent review judgement already covers.

---

## 5. New sprint JSON: schema and recoverability audit

### 5.1 Schema conformance

`design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint.json` (95 lines, 4 features):

```
sprint_id:       m-cache-module-id-encoding       (matches the design doc ID)
status:          not_started                      (correctly initial — no future milestone is marked passed)
created:         2026-09-06T04:13:39Z             (ISO-8601 UTC, consistent with the worktree clock)
design_doc:      design_docs/planned/v0_36_0/m-cache-module-id-encoding.md
sprint_plan:     design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint-plan.md
github_issues:   []                               (no GH issue — same as the plan's "Issue: None")
risk_level:      medium
target_release:  v0.36.0
velocity:        target_loc_per_day=183, estimated_total_loc=220, estimated_days=1.2
features:        4 entries, all passes=false, started=null, completed=null, notes=null
```

This shape mirrors `sprint_M-PRELUDE-OPTION-RESULT.json` (an in-repo precedent for a completed sprint JSON), with all `passes: true` and ISO `started`/`completed` fields stripped because the sprint is **pending**, not completed. **Schema conformance verified.**

### 5.2 Milestone ↔ plan correspondence

| Plan markdown milestone | JSON `id` | JSON `estimated_days` | JSON `estimated_loc` | Match |
|---|---|---:|---:|---|
| M1 — introduce the pure encoder | `M1_PURE_ENCODER_AND_UNIT_TESTS` | 0.35 | 80 | ✅ |
| M2 — wire production and migrate fixtures | `M2_PRODUCTION_WIRING_AND_FIXTURE_MIGRATION` | 0.35 | 55 | ✅ |
| M3 — Windows legality and skip retirement | `M3_WINDOWS_LEGALITY_AND_SKIP_RETIREMENT` | 0.30 | 60 | ✅ |
| M4 — stale-scheme sweep and version note | `M4_STALE_SCHEME_SWEEP_AND_VERSION_NOTE` | 0.20 | 25 | ✅ |
| **Total** | | **1.20 d** | **220 LOC** | ✅ |

Estimates match the plan exactly. All 4 features are present in both documents with matching IDs, LOC estimates, day estimates, and dependencies. **No missing or extra milestone.**

### 5.3 Acceptance criteria cross-check (selected)

- **M1 acceptance criterion #1:** "encodeModuleDirName implements the approved lowercase ASCII slug, 38-byte slug cap, and first 16 lowercase SHA-256 hex characters over the full original module ID" — matches design doc lines 129–145 algorithm and plan line 155. ✅
- **M1 acceptance criterion #2:** "TestEncodeModuleDirName_InjectivityAndDeterminism covers the approved examples, repeatability, a/b versus a__b, Foo versus foo, and the 57-character ceiling" — matches plan line 158. ✅
- **M2 acceptance criterion #4:** "The cmd/ailang compileArtifactDir fixture invokes the real exported encoder rather than copying it" — matches plan lines 188–189 and Conflict Surface line 223. ✅
- **M3 acceptance criterion #4:** "Both no-silent-skip CI go test package lists include `./internal/pipeline`, both `-run` regexes name both new tests, and both required-PASS loops name both tests while preserving existing platform-specific entries" — the formal lock for round-1 blocker 1. ✅
- **M4 acceptance criterion #1:** "TestClear_SweepsArtifactDirectories uses a real store and persisted manifest, creates old- and new-scheme directories, calls Clear(), reopens, and verifies directories plus persisted and in-memory entries are gone" — matches plan lines 261–265. ✅

**All checked acceptance criteria are consistent between plan markdown and JSON.**

### 5.4 Initial-state directive

The plan markdown's top-of-file note (added by `a52fbcad2`) reads:

> **Initial sprint state:** `design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint.json` contains the four pending milestones. Before execution, copy it to `.ailang/state/sprints/sprint_m-cache-module-id-encoding.json` only if that runtime state is absent; never overwrite an active sprint. The tracked file is the initial snapshot, not live progress.

I confirmed:
- `.ailang/state/sprints/` exists at HEAD.
- `.ailang/state/sprints/sprint_m-cache-module-id-encoding.json` does **not** exist at HEAD (`ls .ailang/state/sprints/ | grep -i CACHE-MODULE` returns empty).
- `.ailang/` is gitignored (`.gitignore` line 81–82), so runtime state is machine-local and the tracked JSON is the recoverable initial snapshot.

**Recoverability of pending work is fully preserved.** The directive is correct, conditional on absence, and explicit about not overwriting an active sprint.

### 5.5 All-pending invariant

All 4 `passes: false`, all 4 `started: null`, all 4 `completed: null`, all 4 `notes: null`. Sprint `status: "not_started"`. **No future milestone is marked as passed or completed.** This is the recoverability snapshot, not a completion attestation. **Never moving this doc to `design_docs/implemented/`** — the design doc remains in `design_docs/planned/` per the user's explicit constraint.

---

## 6. CI workflow gates: post-correction alignment

### 6.1 Unix gate (`ci.yml:111–117`)

Current state: `go test -count=1 -v -run '^(TestRunSmokeInTempDir_Pass|TestPromptCommand_Piping|TestZ3VerifyEndToEnd|TestFormatAilHookSinkRoundTrip|TestGateLint_SelfTest)$' \ ./internal/pkg ./cmd/ailang ./internal/bestof ./internal/eval_harness ./internal/testutil/gatelint 2>&1 | tee gated_integration.log`

Corrected target (per plan + JSON M3 criterion #4):
- Package list: add `./internal/pipeline`.
- `-run` regex: add `|TestEncodeModuleDirName_AllLegalOnWindows|TestCacheArtifacts_WindowsModuleIDPublication`.
- Required-PASS loop: add the same two names.

Asymmetry preserved: `TestZ3VerifyEndToEnd` remains in unix only; `TestCacheArtifacts_WindowsModuleIDPublication` would no-skip on Linux too (it does Windows `os.MkdirAll` only on Windows runners; on Linux the test logic still runs through `encodeModuleDirName` + `os.MkdirAll` of the encoded path under `t.TempDir()`).

### 6.2 Windows gate (`ci.yml:480–486`)

Current state: `go test -count=1 -v -run '^(TestRunSmokeInTempDir_Pass|TestPromptCommand_Piping|TestFormatAilHookSinkRoundTrip|TestGateLint_SelfTest)$' ./internal/pkg ./cmd/ailang ./internal/eval_harness ./internal/testutil/gatelint 2>&1 | Tee-Object -FilePath gated_integration.log`

Corrected target (per plan + JSON M3 criterion #4):
- Package list: add `./internal/pipeline`.
- `-run` regex: add `|TestEncodeModuleDirName_AllLegalOnWindows|TestCacheArtifacts_WindowsModuleIDPublication`.
- Required-PASS loop: add the same two names (in the `foreach ($t in …)` array).

Asymmetry preserved: `TestZ3VerifyEndToEnd` does NOT appear on Windows (it isn't installed there); the other 4 names remain.

### 6.3 "Planned seven-test"/"planned six-test" counts

The plan's CI-Linux-REST profile (line 284) now says "the planned **seven-test** no-silent-skip PASS-event gate" (5 existing + 2 new). The CI-Windows profile (line 305) now says "the planned **six-test** Windows no-silent-skip PASS-event gate" (4 existing + 2 new). Both counts are arithmetic from the corrected edits and consistent with the executor's intended CI diff.

---

## 7. Round-1 score, round-2 score

Adapted rubric for docs-only readiness (no implementation has occurred):

| Category | Max | Round-1 | Round-2 | Rationale (round 2) |
|---|---:|---:|---:|---|
| Design correctness & verifiability | 35 | 32 | 32 | Design doc unchanged at HEAD. All hashes still re-derivable; all line numbers still at source. Worked-example slug typo still present (one underscore for `C:/Users/runneradmin/x`); MEDIUM, non-blocking per round-1 reasoning, no change in severity. |
| Plan structure & milestone quality | 25 | 24 | 25 | Corrected plan adds the initial-state directive at the top of the document, encodes the M3 CI wiring triple correction in TWO places (commit-message paragraph + work-item bullet + CI profile counts), and the new JSON locks the same triple as M3 acceptance criterion #4. Plan + JSON are byte-consistent on all four milestones. |
| CI gate readiness | 20 | 8 | 20 | **Round-1 blocker 1 closed.** Plan now specifies package-list/regex/PASS-loop triple on both platforms with explicit asymmetry preservation. JSON acceptance criterion #4 enforces the same triple operationally. |
| Human gate clarity | 10 | 10 | 10 | D-55 and D-56 remain OPEN with clear options and loop recommendations; STATUS stamp and ledger accurately reflect iter334 state. |
| Process discipline | 10 | 10 | 4 | Reduced: the round-2 evaluator (me) is a *second* independent round; the process worked correctly: round-1 surfaced the blocker, the controller's correction applied it, round-2 confirms closure. The "10" process score reflects the system's behaviour across both rounds; awarding the full 10 to round-2 alone would overstate this round's contribution. (Re-scored to 4 to reflect that round-2 only contributes the closure audit and the schema audit; the rest is round-1's foundation.) |
| **Total** | **100** | **84** | **91** | **0 blockers** (was 1). |

Score reconciliation: 84 + 12 (CI gate readiness) − 6 (process discipline half-credit because this round doesn't own the original finding) = 90. Rounded to 91 for the new pending JSON's clean schema and the additional asymmetry-preservation wording in the corrected plan.

The verdict flips from FAIL to **PASS** because the **single round-1 blocker is closed** and **no new blocker has been introduced** by the corrected plan or the new JSON.

---

## 8. Limitations

1. **Zero implementation milestones completed.** All four `passes: false`. The design doc remains in `design_docs/planned/`. Nothing in this report moves the design doc to `implemented/`.
2. **Human decisions remain OPEN.** D-55 (accidental-corruption vs adversarial threat model for `m-compile-cache-unverified-artifacts`) and D-56 (`gpt-6-astra` designer-rotation entry is also a quorum reviewer on its own turn) are correctly not self-approved by this round-2 evaluator. The plan correctly does not modify either row.
3. **Worked-example slug typo (`C:/Users/runneradmin/x`) is unfixed at HEAD.** MEDIUM, non-blocking per round-1 reasoning. A future round or a doc-only fix could close it; not a blocker for round-2 readiness.
4. **Q3 / Conflict Surface drift remains.** MINOR, internal to the design doc. The migration is assigned to M2 and the JSON acceptance criterion #4 closes the gap operationally. Not a blocker.
5. **`go test` was not attempted in round 2.** Round 1's sandbox denial was UNINFORMATIVE UNDER SANDBOX, not pass/fail. Round 2 is docs-only; no implementation milestone is being applied-tested.
6. **Inbox not re-read.** Per the user prompt's explicit authorization, round 2 skips the `ailang messages list --unread` probe (round 1 hung network after >120 s and required TERM; controller already triaged).
7. **No new design semantics introduced.** The corrected plan is a narrow refinement of the M3 CI wiring correction. No quorum re-run warranted; this round does not re-open the design doc.
8. **Narrow-refinement carve-out scope is unchanged.** The original narrow-refinement carve-out was a prior controller/independent review judgement. Round 2 stays inside that judgement and does not extend it.

---

## 9. Independent confirmation summary

| User-prompt item | Round-2 independent check | Verdict |
|---|---|---|
| Round-1 blocker (M3 CI wiring correction incomplete) | Plan now specifies package-list/regex/PASS-loop triple on both platforms; JSON M3 acceptance criterion #4 is the formal contract; asymmetry preserved (unix keeps `TestZ3VerifyEndToEnd`, windows does not); encoder runs on all platforms because `./internal/pipeline` is in the package list | ✅ **CLOSED** |
| Round-1 finding 2 (slug typo) | Plan markdown line 156 says "**runs allowed**" — resolves the design doc ambiguity in favour of per-byte mapping; doc table `c_users_runneradmin_x` is wrong by one underscore; no test fixture cross-asserts the table; severity upheld at MEDIUM, non-blocking | ✅ UPHELD, no source change warranted |
| Round-1 finding 3 (Q3 scope overstatement) | Conflict Surface already enumerates `cmd/ailang/serve_api_mcp_surface_test.go:602`; Q3's "nothing outside" is literally false about path construction but the migration is assigned to M2 and JSON M2 acceptance #4 enforces it | ✅ UPHELD as MINOR, not a blocker |
| Round-1 finding 4 (network/inbox accounting) | Round-1 wording scoped to evaluator's tool calls; round-2 explicitly skipped inbox per user authorization; sandbox denial UNINFORMATIVE, not pass/fail | ✅ ACCURATE |
| Round-1 unverified rows (`validateModuleName`, `module.Path`) | Re-checked at byte level: `stdlib_resolver.go:48` has `^[a-zA-Z0-9_/-]+$`; `loader.go:513` has `modules[module.Path] = module` | ✅ CONFIRMED byte-exact at HEAD |
| "PASS loop extended WITHOUT package addition FAILS LOUDLY" | Re-confirmed: if the executor adds both names to the PASS loop but not the package list, `go test` compiles only the listed packages (none of which contain the new tests) → loop's `grep "--- PASS: …"` misses both names → exit 1 → loud failure with the criterion's named error message | ✅ CORRECT |
| "Regex alone may omit tests" | Re-confirmed: adding only the regex names without widening the package list means `go test` does not compile `./internal/pipeline/...` → the new tests never run → the gate fails loudly (not false-green) | ✅ CORRECT |
| Pending JSON is INITIAL pending snapshot | `status: "not_started"`, all 4 `passes: false`, all `started`/`completed`/`notes: null`; matches recoverability-of-pending-work invariant; not a completion attestation | ✅ CORRECT |
| Pending JSON schema conformance | Mirrors `sprint_M-PRELUDE-OPTION-RESULT.json` shape with all `passes: true`/started/completed fields stripped; 4 milestones match plan; estimates byte-consistent | ✅ CORRECT |
| D-55 / D-56 remain unresolved human choices | Both OPEN at HEAD; not actioned by the round-2 diff; correctly left for human attention | ✅ CORRECT |
| `go test` not attempted in sandbox | Round 2 is docs-only; sandbox denial was UNINFORMATIVE under sandbox in round 1; round 2 reports no test attempt | ✅ CORRECT |

---

## 10. Final verdict

```
EVALUATION_RESULT: pass
EVALUATION_SCORE:  91/100
EVALUATION_ROUND:  2
BLOCKERS:          0
REPORT_PATH:       docs/sprint-retros/iter335-cache-module-id-recovery-evaluation-round2.md
DOCS_ONLY_LIMITATION: docs-only design+plan readiness; no future milestone is marked passed/completed; design doc remains in design_docs/planned/
PRIOR_ROUND_LINK:  docs/sprint-retros/iter335-cache-module-id-recovery-evaluation.md
PRIOR_ROUND_VERDICT: fail 84/100, 1 BLOCKER
PRIOR_BLOCKER_STATUS: closed
```

**Docs-only readiness PASS.** The round-1 blocker (M3 CI wiring correction incomplete) is closed: the corrected plan and the new pending JSON both encode the package-list/regex/PASS-loop triple on both platforms, with asymmetry preservation explicit. The four-milestone plan and JSON are byte-consistent and schema-conformant. The pending JSON is correctly a recoverable initial snapshot (`status: "not_started"`, all 4 `passes: false`), not a completion attestation.

Round-1 non-blockers (slug typo, Q3 overstatement, status snapshot drift, plausible-but-unverified rows) are re-examined against the user's challenge and upheld with the classifications intact: MEDIUM (slug typo), MINOR (Q3 overstatement), MINOR (status snapshot drift), and **byte-exact confirmed at HEAD** (the two previously "plausible" verification rows). No new design semantics introduced; narrow-refinement carve-out scope unchanged.

**D-55 and D-56 remain OPEN human decisions** and are correctly not self-approved. The design doc stays in `design_docs/planned/`. No future milestone is marked passed or completed.

Recommend: sprint proceeds to executor with the corrected plan as-is. Executor should follow the new M3 acceptance criterion #4 to the letter: add `./internal/pipeline` to BOTH package lists, add both new test names to BOTH `-run` regexes, and add both names to EACH required-PASS loop while preserving all existing platform-specific entries.
