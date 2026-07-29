# Sprint Plan — M-PROPERTY-GENERATOR-COVERAGE **Lane A only**

**Design doc**: [m-property-generator-coverage.md](m-property-generator-coverage.md) (Lane A = A1–A6)
**Sprint ID**: `M-PROPERTY-GENERATOR-LANE-A`
**Upstream filing**: sunholo-data/ailang#517
**Branch / worktree**: `sprint/m-property-generator-lane-a` @ `.claude/worktrees/sprint-m-property-generator-lane-a`
**Base**: `6c70a6c36` (= `4ebd85873` + 2 doc-only commits; `git diff 4ebd85873 HEAD -- internal/testing cmd/ailang/test.go` is **empty**, so all controller measurements taken at `4ebd85873` transfer unchanged)
**Planner**: iteration 121, V1 mission. All facts below re-measured first-party in this worktree with `go build -o /tmp/ailang_planner121 ./cmd/ailang`.
**Estimate**: 1.0 day (doc says ~0.5d; +0.5d is measured triage the doc did not budget — see H1 / M1)
**Risk**: medium (semantic exit-code change; 3 in-repo example files change exit status at M1 alone)

**Scope**: Lane A only. Lane B (B1 structural derivation, B2 deferred) is **OUT**. Lane A ships
without any part of Lane B — confirmed: A1's list arm recurses into `createGeneratorForType` for the
element type and correctly returns `nil, nil` when the element is non-derivable (probe-verified on
`list[Tree]`), so no derivation machinery is required.

---

## 0. Executor operating constraints (read first)

1. **Flags BEFORE the positional path, always.** `ailang test f.ail --format json` silently ignores
   `--format` (measured: human text, rc=0). Root cause is Go `flag` stopping at the first non-flag
   arg (`cmd/ailang/main.go:120`, `_ = testFlags.Parse(flag.Args()[1:])`); filed as **#534**,
   systemic across subcommands, **explicitly out of scope**. Every command in this plan is written
   flags-first. If you write a command flags-last, your acceptance check tests nothing.
2. **You cannot commit in this worktree** (sandbox blocks the linked worktree's git metadata). Leave a
   clean, reviewable working diff and report, per milestone, exactly which files changed and why. The
   controller commits each milestone separately.
3. **Sandbox-uninformative gates.** `go test ./internal/testing/...` binds no sockets — **informative**.
   `go test ./cmd/ailang/...` and `make test` (= `go test ./...`) do bind loopback sockets in 6 test
   files (`configdriven_callstream_test.go`, `configdriven_dispatch_test.go`,
   `configdriven_harvest_test.go`, `configdriven_streaming_span_snapshot_test.go`,
   `configdriven_streaming_test.go`, `messages_send_test.go`) and MUST be reported as
   **`UNINFORMATIVE UNDER SANDBOX`**, never pass or fail. The controller re-runs them outside the sandbox.
4. **`make check-file-sizes` is a gate on every milestone.** `internal/testing/runner.go` is **774
   lines**; the gate fails at >800 (`make/code-health.mk:123`). You have **26 lines of headroom in that
   one file**. If the TypeApp arm plus SkipKind assignments would exceed it, put the `SkipKind`
   constants and any classification helper in `internal/testing/result.go` (122 lines) instead.
5. **Do not add files under `examples/runnable/`.** That trips the manifest-drift gate
   (`scripts/validate_manifest.go --ci`, `make/examples.mk:24`) unless `examples/manifest.json` gains a
   matching entry. Lane A fixtures go in Go tests via `t.TempDir()` + `os.WriteFile`, the convention
   already used at `internal/testing/runner_ensures_test.go:19-21`. Coding-standards' "every language
   feature needs an example" does not apply: Lane A adds **zero syntax** (the doc's own A8 score is 0).
   If you add an examples/ file anyway you MUST add the manifest entry and run `make verify-examples`.
6. **Do not modify any existing test assertion.** Measured: **zero** existing Go tests need updating
   (see A-10). `git diff -U0 -- '*_test.go'` must show no removed lines in pre-existing test files.
7. **For the controller**: `.ailang/` is gitignored (`.gitignore:77`) but sprint JSONs are tracked, so
   the progress file needs `git add -f .ailang/state/sprints/sprint_M-PROPERTY-GENERATOR-LANE-A.json`.
   Without `-f` it will be silently dropped from the commit.
8. **Bounded waits when sweeping examples.** `timeout` is **not on PATH** on this rig (rc 127). Five
   contract examples are pathologically slow — see R-9.

---

## 1. Ground truth measured for this plan

### 1.1 The six `StatusSkip` sites are the *entire* skip surface

`grep -n StatusSkip internal/testing/runner.go` → **249, 331, 373, 454, 490, 536**. All six are
`PropertyResult` sites. `TestResult` is **never** assigned `StatusSkip` anywhere in non-test Go under
`internal/` or `cmd/` — the only hits are those six lines. (`reporter.go:148`'s test-skip branch is
therefore dead code today; it is still the correct model for the A4 property fix.)

| Site | Message text | Class | `TestsRun` at return |
|---|---|---|---|
| 249 | `no generator for type %v` (forall binder) | `no_generator` | 0 |
| 331 | `ensures property has no function context (top-level ensures not supported)` | `unsupported` | 0 |
| 373 | `no generator for parameter %s: %v` (ensures) | `no_generator` | 0 |
| 454 | `requires property has no function context (top-level requires not supported)` | `unsupported` | 0 |
| 490 | `no generator for parameter %s: %v` (requires) | `no_generator` | 0 |
| 536 | `requires not satisfied by random input (consider tighter generators): %s` | `out_of_contract` | **≥1** |

Because the six sites are exhaustive, the taxonomy can be made **total** and that totality is a
checkable invariant (A-8) rather than a best-effort convention.

### 1.2 Two distinct no-generator texts (resolves the doc's ambiguous "keep the existing message text")

Site 249 emits `no generator for type %v`; sites 373/490 emit `no generator for parameter %s: %v`.
**Resolution: both texts stay byte-unchanged.** Lane A adds no diagnostic text. The machine-readable
signal is `skip_kind`, not the prose. No test may assert a generic `"no generator"` substring as if
there were one message.

### 1.3 In-repo exit-status table (all 33 files in `examples/runnable/contracts/`, measured today)

`succ` = JSON `success`; `s` classified by originating site. **`rc` today = 1 whenever `succ=false`.**

| File | today `succ` / p / f / s | skip classes today | after **M1** (A1) | after **M3** (A3) |
|---|---|---|---|---|
| access_control | false 0/0/4 | nogen×4 (Role) | unchanged (rc 1) | rc 1 |
| basic | false 1/4/3 | ooc×3 | unchanged (rc 1, 4 real fails) | rc 1 |
| cross_function | false 0/0/5 | nogen×5 (Priority, Region) | unchanged (rc 1) | rc 1 |
| cross_module_functions | false 2/1/2 | ooc×2 | unchanged (rc 1) | rc 1 |
| **cross_module_functions_lib** | **true 2/0/1** | **ooc×1** | **unchanged (rc 0)** | **rc 0 — A-4 fixture, MUST stay green** |
| cross_module_types | false 1/1/6 | nogen×5 (`()`, Cell, `list[Tree]`), ooc×1 | `list[Tree]` **stays vacuous** (A-9) | rc 1 |
| cross_module_types_lib | false 0/0/0 | — (no tests) | unchanged (rc 1) | rc 1 |
| ensures_violation_demo | false 1/1/0 | — | unchanged (rc 1) | rc 1 |
| finance | false 0/1/5 | nogen×4 (TaxBracket), ooc×1 | unchanged (rc 1) | rc 1 |
| **hof_verify** | false 0/0/2 | nogen×2 (`list[int]`) | **rc 1 → 0** (2 pass ×100) | rc 0 |
| inbox_injection | false 4/4/2 | ooc×2 | unchanged (rc 1) | rc 1 |
| inbox_injection_v2 | **true 1/0/10** | nogen×10 (`string<email>`) | unchanged (rc 0) | **rc 0 → 1** |
| inbox_v2_app | **true 1/0/10** | nogen×10 (Mail, `string<email>`) | unchanged (rc 0) | **rc 0 → 1** |
| inbox_v2_lib | false 0/0/0 | — (no tests) | unchanged (rc 1) | rc 1 |
| insurance | false 0/0/6 | nogen×6 (AgeBand, Coverage, RiskTier) | unchanged (rc 1) | rc 1 |
| invoice | false 2/1/18 | nogen×15, ooc×3 | 2 `list[int]` now run; 13 vacuous remain (rc 1) | rc 1 |
| **list_recursive_verify** | false 0/0/6 | nogen×6 (`list[int]`) | **rc 1 → 0 — see H1** | **rc 0 (stays green)** |
| **list_verify** | **true 1/0/6** | nogen×6 (`list[int]`) | **rc 1** — 5 pass + **1 real fail** (`brokenAllPositive_property_2`) + 1 ooc | rc 1 |
| nested_record_verify | false 0/0/5 | nogen×5 (nested anon record) | unchanged (rc 1) | rc 1 |
| park | **true 1/0/7** | nogen×6 (Season), ooc×1 | unchanged (rc 0) | **rc 0 → 1** |
| per_function_depth_verify | false 0/4/3 | ooc×3 | unchanged (rc 1) | rc 1 |
| quantifier_verify | false 0/3/4 | **ooc×4** | unchanged (rc 1) | rc 1 |
| record_adt_cycle_verify | false 0/0/1 | nogen×1 (Doc) | unchanged (rc 1) | rc 1 |
| record_adt_sort_verify | false 0/0/1 | nogen×1 (Proposal) | unchanged (rc 1) | rc 1 |
| record_discovery_verify | **true 2/0/8** | nogen×6 (anon records), ooc×2 | unchanged (rc 0) | **rc 0 → 1** |
| record_pattern_verify | false 0/0/5 | nogen×5 (`{x,y}`) | unchanged (rc 1) | rc 1 |
| record_verify | false 0/0/15 | nogen×15 (`{x,y}`) | unchanged (rc 1) | rc 1 |
| unencodable_callee_skip | false 0/1/1 | **ooc×1** (not a Z3 class — see R-3) | unchanged (rc 1) | rc 1 |
| recursive_verify, scoring, showcase, string_verify, temperature | **UNMEASURED — see R-9** | | | |

**R-9 — these 5 files are pathologically slow under `ailang test`.** `recursive_verify.ail` emitted only
the `→ Running tests in …` preamble and produced **no JSON after several minutes** (measured; the run
was killed). The other four behave the same. Use a **bounded wait** when sweeping (`gtimeout`/`perl
alarm` — note plain `timeout` is **not** on PATH on this rig, it returns rc 127) and **exclude these
five from the per-milestone gate sweep**, reporting them as `UNMEASURED (exceeds bounded wait)` rather
than pass/fail. They contain no `list[…]` params in the shapes A1 touches as far as could be
determined, so they are not expected to change; if the executor can measure them under a bound, record
the result. Do **not** let a slow example block a milestone gate.

**Net semantic change at M3: exactly 4 files flip rc 0 → 1** — `inbox_injection_v2`, `inbox_v2_app`,
`park`, `record_discovery_verify`. That is the honesty fix landing, and it is the whole point.
**`cross_module_functions_lib` must NOT flip** (out_of_contract forgiven) — that is the false-red guard.

### 1.4 Blast-radius reality check on `ailang test`

Zero invocations of `ailang test` across `make/`, `Makefile`, `.github/workflows/`, `tools/`,
`scripts/` (positive control in the same search: 8 `ailang check` hits). Sole programmatic consumer is
`cmd/ailang/coordinator_cloud.go:586`, itself guarded by `len(testFiles) > 0` over `*_test.ail`
(`:583-584`) — so the escalate-to-AI flip only reaches packages that actually ship test files.
**Lane A's exit-code flip reds no CI gate.**

---

## 2. Resolution of the six-skip-sites taxonomy hole (REQUIRED decision, stated)

**Class name: `unsupported`.** Sites 331 and 454. **Decision: it FAILS the suite** — counted into
`VacuousSkips` alongside `no_generator`.

**Reasons:**

1. **Semantics.** A `requires`/`ensures` clause the user wrote that executed **zero** cases is vacuous
   by definition, identically to `no_generator`. `out_of_contract` is categorically different: the
   contract *did* execute (`TestsRun ≥ 1`) and a random input was discarded.
2. **Zero live blast radius**, because these two sites are **currently unreachable from any parsed
   AILANG program** (proof in §6, Refutation R-1). Choosing "fail" therefore costs nothing today and
   installs the honest default for the day the parser grows top-level contracts.
3. **Default-deny.** The defect this sprint closes *is* an "unclassified ⇒ forgiven" default. Leaving a
   fourth silent bucket would reproduce it. Hence `AddPropertyResult` treats an **empty `SkipKind` on a
   `StatusSkip` property as vacuous (fail-closed)**, and A-8 asserts the taxonomy is total so the
   fail-closed arm is never actually reached.

**Not deferred.** Cost is two `SkipKind` assignments (~2 LOC) plus one unit test that constructs
`PropertyCase{Property: &ast.Property{Kind: ast.EnsuresKind}, Function: nil}` directly — the only way
to reach the sites, and a legitimate white-box test of a defensive guard. No `.ail` fixture and no
triage budget is spent, because there is no live bug there (R-1).

---

## 3. Milestones

Each milestone is independently committable and has its own gate. **Every** gate additionally requires
`make check-file-sizes` green and `gofmt`/`make lint` clean on touched files.

### M1 — A1: reachable list generator (`*ast.TypeApp{"list"}` arm)

**Why first**: it shrinks the vacuous surface *before* M3 flips exit codes, so M3's measured blast
radius is the final one, not an intermediate.

**Change** — `internal/testing/createGeneratorForType` (runner.go:629): add, before the existing
`*ast.ListType` arm, a `*ast.TypeApp` arm matching `Constructor == "list" && len(Args) == 1`, recursing
into `createGeneratorForType(app.Args[0])` and returning `nil, nil` when the element generator is nil.
`[T]` / `list[T]` has parsed to `ast.TypeApp{Constructor: "list"}` since DX-17 Phase 2
(`internal/parser/parser_type.go:56`, `:112`).

**Keep the `*ast.ListType` arm.** It has **zero** non-test constructors under `internal/`, `cmd/`,
`scripts/` (positive control: **21** constructions in `_test.go` files), so it is dead from parsed
programs but live from Go tests. Deleting it would break those tests for no benefit —
coding-standards.md's "never delete because the linter says unused" applies.

**Probe-verified outcome** (planner applied this exact patch to a scratch copy, built, measured, then
restored `runner.go` byte-identically — worktree left clean):
- `headOr(xs: [int], d: int)` ensures-property: `skip / 0 cases` → **`pass / 100 cases`**.
- `list[Tree]`: **stays** a `no_generator` vacuous skip (element ADT non-derivable) — correct, pinned by A-9.
- In-repo: `hof_verify` rc 1→0; `list_verify` rc stays 1 but `success` true→**false** with a **genuine
  failure** surfaced (`brokenAllPositive_property_2`, 2 cases); `list_recursive_verify` rc **1→0** (H1);
  `invoice` gains 2 running properties.

**Triage budget: 0.5 day** — the doc attributes the latent-bug-detector risk to Lane B only, but A1
starts executing **16 previously-never-executed properties across 5 files**. `list_verify`'s new
failure is deliberate (`brokenAllPositive*`); `hof_verify`'s and `list_recursive_verify`'s outcomes
are measured above; the rest must be triaged, not treated as regressions.

**Gate M1**
```bash
go test ./internal/testing/... -count=1                 # informative under sandbox
make check-file-sizes                                    # runner.go must stay <=800
# behavioral, flags BEFORE path:
ailang test --format json <tmp mixed fixture>            # headOr: tests_run == 100, status pass
ailang test --format json examples/runnable/contracts/hof_verify.ail        # rc 0
ailang test --format json examples/runnable/contracts/list_verify.ail       # rc 1, 1 real fail
ailang test --format json examples/runnable/contracts/cross_module_types.ail # list[Tree] still nogen skip
```
Report `go test ./cmd/ailang/...` as `UNINFORMATIVE UNDER SANDBOX`.

**Est. LOC**: 12 impl / 60 test.

---

### M2 — A2: total skip taxonomy (behaviour-inert)

**Change**
- `PropertyResult.SkipKind string` (result.go:27-35).
- Named constants: `SkipKindNoGenerator = "no_generator"`, `SkipKindUnsupported = "unsupported"`,
  `SkipKindOutOfContract = "out_of_contract"` (in `result.go` — keeps runner.go under the size gate).
- Assign at **all six** sites: 249/373/490 → `no_generator`; **331/454 → `unsupported`**; 536 → `out_of_contract`.
- `SuiteResult.VacuousSkips int`, incremented in `AddPropertyResult` when
  `Status == StatusSkip && SkipKind != SkipKindOutOfContract` — i.e. `no_generator`, `unsupported`, **and
  fail-closed for an unexpected empty kind**.
- Merge `VacuousSkips` in **both** aggregation loops: `cmd/ailang/test.go:52-58` (single-file) and
  `:181-187` (package mode). Missing either silently zeroes the counter for that mode.

**`Success()` is NOT touched in this milestone.** M2 must be provably inert.

**Gate M2**
```bash
go test ./internal/testing/... -count=1
make check-file-sizes
# INERTNESS PROOF: re-run the §1.3 sweep and diff against the M1 table — must be byte-identical
```
Plus the new unit tests: A-8 (totality), A-4 classification, and the `Function == nil` white-box test
reaching sites 331/454.

**Est. LOC**: 35 impl / 90 test.

---

### M3 — A3 + A4: exit/`success` semantics + reporter honesty

**Change**
- `result.go:97` — `Success()` becomes `ran > 0 && sr.FailedTests == 0 && sr.VacuousSkips == 0`.
  `AllSkipped()` (`:104`) and `SuccessAllowingSkips()` (`:110`) stay **unchanged**, so
  `--allow-skips` forgives vacuous skips for free through the existing
  `succeeded := Success() || (allowSkips && SuccessAllowingSkips())` at `test.go:80` and `:208`. No new flag.
- `reporter.go:47-64` (JSON) — add top-level `"vacuous_skips"` and per-property `"skip_kind"`.
  `"success"` (`:56`) tracks A3 automatically.
- `reporter.go:181` (human) — widen `if prop.Status == StatusFail` to
  `if prop.Status == StatusFail || prop.Status == StatusSkip`, mirroring the existing test-side
  condition at `:148`. **Verify this surfaces the reason for `out_of_contract` skips too**, not only
  `no_generator` (H1 disclosure depends on it).
- `reporter.go:205-219` — add a summary branch **between** `case result.AllSkipped():` and
  `case result.Success():` that names the count and `--allow-skips`, e.g.
  `✗ N properties never ran (no generator) — use --allow-skips to permit`.

**Gate M3**: acceptance criteria A-1, A-2, A-4, A-6, A-7, A-10 (§4) plus the full §1.3 sweep matching
the "after M3" column — in particular **exactly 4** files flip rc 0→1 and
`cross_module_functions_lib.ail` stays rc 0.

**Est. LOC**: 45 impl / 110 test.

---

### M4 — A5: `--format json` writes only JSON to stdout

**Change** — route preamble lines to **stderr when `formatStr == "json"`**, human mode unchanged:
- `cmd/ailang/test.go:18` — `→ Running tests in %s` (single-file mode).
- `cmd/ailang/test.go:163`, `:164`, `:173` — package-mode preamble (`→ Package %s`, the
  `%s source modules, %s test files` line, and the blank `fmt.Println()`). **Three writes, not one** —
  the doc's `:163-173` range is right but it is easy to move only the first.

**Note**: the assertion is on **stdout only**. stderr is *not* guaranteed empty — a legitimate
`Warning: stdlib version mismatch: …` (166 bytes, `internal/loader/stdlib_resolver.go:310`) already
goes to stderr on some files. Do not assert `stderr == ""`.

**Gate M4**
```bash
ailang test --format json <fixture> 2>/dev/null | jq .           # must parse, zero preprocessing
ailang test --package --format json . 2>/dev/null | jq .          # package mode too
ailang test <fixture>                                             # human mode preamble UNCHANGED on stdout
go test ./internal/testing/... -count=1 && make check-file-sizes
```

**Est. LOC**: 15 impl / 30 test.

---

### M5 — closeout: CHANGELOG, doc status, follow-ups

- `changelogs/v0.18-current.md` `[Unreleased]` — **BREAKING**: mixed suites with `no_generator` /
  `unsupported` property skips now exit 1; escape is `--allow-skips`. Additive JSON fields
  `vacuous_skips`, `skip_kind`. JSON-mode stdout is now pure JSON. `[T]`/`list[T]` parameters now get
  generators. Name the 4 in-repo files that flip.
- `design_docs/planned/v0_31_0/m-property-generator-coverage.md` — mark Lane A landed; record the
  corrections in §6 (R-2 in particular, so the stale before/after table does not mislead Lane B's planner).
- File follow-ups (**do not implement**):
  - **F1** — out-of-contract discard rate is a generator-quality vacuous pass (H1). A property that
    `out_of_contract`-skips after 1–2 of 100 cases is near-unvalidated yet forgiven; `list_recursive_verify.ail`
    goes rc 1 → 0 under it. Needs a threshold policy, which is a human-owned semantic decision.
  - **F2** — #534: flags after the positional path are silently ignored across `flag.NewFlagSet` subcommands.
  - **F3** — Lane B1 (structural derivation) remains the fix for the 6 files still vacuous after Lane A.

**Gate M5**: `make check-file-sizes`, `make lint`, `go test ./internal/testing/... -count=1`, and
`make verify-examples` (should be unaffected — it invokes `ailang run`, never `ailang test`).

**Est. LOC**: 0 impl / 0 test (docs ~80 lines).

---

## 4. Acceptance criteria — validated against the code, with red-turning mutations

Doc rows A-1…A-7 checked one by one; corrections marked. A-8…A-10 are **new**, added because the code
disagreed with the doc.

| # | Criterion (corrected) | Red-turning production mutation | Status vs doc |
|---|---|---|---|
| **A-1** | Mixed fixture — `anchor(x: int)` [pass] + `shiftX(p: Point, dx: int)` [vacuous] + `headOr(xs: [int], d: int)` [pass after A1] — gives `ailang test --format json F` → **rc 1**, `success=false`, `vacuous_skips=1`, 2 passing siblings | Revert `Success()` to `ran > 0 && FailedTests == 0` (drop the `VacuousSkips` term) | **CORRECTED** — the doc's fixture is unusable, see R-2 |
| **A-2** | Same fixture, `ailang test --allow-skips --format json F` → **rc 0** | Make the allow-skips branch also require `VacuousSkips == 0` | OK — but note this is **non-discriminating before M3** (the fixture already exits 0 today); assert it only post-M3 |
| **A-3** | `headOr(xs: [int], d: int)` ensures-property runs **100 cases** and passes | Delete the new `*ast.TypeApp` "list" arm | **VERIFIED by probe** (0 → 100 cases, pass) |
| **A-4** | `examples/runnable/contracts/cross_module_functions_lib.ail` (2 pass + 1 out-of-contract skip): **rc 0**, `vacuous_skips=0`, that property's `skip_kind == "out_of_contract"` | Set `SkipKind = SkipKindNoGenerator` at runner.go:536 | OK — **fixture already exists in repo**, no new file needed |
| **A-5** | `ailang test --format json F 2>/dev/null \| jq .` parses on raw **stdout** (single-file **and** `--package` mode) | Restore any preamble `fmt.Printf` to stdout in JSON mode | **VERIFIED** (`jq` fails today) — scope corrected: 4 write sites, and **do not** assert `stderr == ""` |
| **A-6** | Human output for the mixed fixture contains `no generator for parameter` and does **not** contain `All tests passed!` | Revert reporter.go:181 to `StatusFail`-only (drops the reason) / remove the new summary branch (restores the headline) | OK — but two distinct texts exist (§1.2); assert the *parameter* form for this fixture, never a generic `"no generator"` substring |
| **A-7** | All-skipped suite whose only skip is **`out_of_contract`** still prints `NO TESTS RAN` + rc 1 | **Drop the `ran > 0` term** from `Success()` (leaving `FailedTests == 0 && VacuousSkips == 0`) | **MUTATION CORRECTED** — the doc's mutation (`Success()` returns `FailedTests == 0`) also turns A-1 red, so it does not isolate A-7; and the fixture must be out-of-contract-only, because a `no_generator` all-skipped suite stays red via `VacuousSkips` regardless of `ran > 0`. Unit-level `SuiteResult` fixture. |
| **A-8** | **NEW — taxonomy totality**: every `StatusSkip` `PropertyResult` the runner can produce carries a non-empty `SkipKind`; a table-driven test covers all six sites, reaching 331/454 by constructing `PropertyCase{Property: &ast.Property{Kind: ast.EnsuresKind/RequiresKind}, Function: nil}` | Remove the `SkipKind` assignment at **any one** of the six sites | **NEW** — this is what permanently closes the six-sites hole |
| **A-9** | **NEW — no silent element fallback**: `list[Tree]` (list of a non-derivable element, live in `cross_module_types.ail`) remains a `no_generator` vacuous skip after A1 | Make the new TypeApp arm substitute a default (int/unit) generator when the element generator is nil, instead of returning `nil, nil` | **NEW** — CLAUDE.md principle 2; without it A1 would silently test wrong values |
| **A-10** | **NEW — no test was quietly widened**: `git diff -U0 -- '*_test.go'` shows **no removed lines** in pre-existing test files | Any deletion/edit of an existing assertion | **NEW, and it REFUTES the doc** — see R-5: the doc's "deliberately updated tests" list is **empty** |

---

## 5. Regression fixtures — corrected

| # | Doc entry | Verdict |
|---|---|---|
| 1 | `basic.ail` — "passing properties must keep passing with unchanged case counts" | Valid but weak: measured **1 pass / 4 fail / 3 out_of_contract skips**, already rc 1. Assert the 1 pass and the 3 `skip_kind=="out_of_contract"`; do **not** assert rc changes. |
| 2 | `quantifier_verify.ail` — "4 skips with vacuous=0 (non-generator skip class)" | **Class misnamed**: its 4 skips are all `out_of_contract` (site 536). And with **3 failures** it is already rc 1, so "must still exit as today" is non-discriminating. Keep it only as a `vacuous_skips == 0` assertion. |
| 3 | `unencodable_callee_skip.ail` — "the Z3-unencodable skip class" | **WRONG — see R-3.** The file exists, but its one skip is `out_of_contract`, and **there is no Z3-unencodable skip class in `ailang test` at all**. Replace with: assert its skip classifies `out_of_contract` and it stays rc 1 (it has 1 real failure). |
| 4 | `ensures_violation_demo.ail` — failing suite keeps failing for the failure reason | Valid, measured 1 pass / 1 fail / 0 skips → assert rc 1 and `vacuous_skips == 0`. |
| 5 | New mixed-shape fixture | Valid — built per A-1 above, as an in-test `t.TempDir()` source (see §0.5). |
| 6 | A requires-out-of-contract case ⇒ exit 0 preserved | Valid, and **`cross_module_functions_lib.ail` already is it** (2 pass + 1 ooc skip, rc 0). This is the single most important false-red guard in the sprint. |

---

## 6. Refutations and new findings (read before executing)

### R-1 — REFUTES the controller's impact claim on sites 331/454 (measurement upheld, inference refuted)

The controller's **measurement** is correct: six sites at 249/331/373/454/490/536 with the texts as
quoted. The **inference** — "That is very likely the same bug one layer over: the user wrote a
contract, it never ran, and the suite reports success" — is **refuted**. Sites 331 and 454 are
**unreachable from any parsed AILANG program**:

1. `runEnsuresProperty` / `runRequiresProperty` are entered only from `runProperty`'s dispatch on
   `propCase.Property.Kind ∈ {EnsuresKind, RequiresKind}` (runner.go:225-228).
2. The only producer of those Kinds is `parseContractPredicate` (parser_contracts.go:129-148) — the sole
   `ast.Property{}` construction in the parser that sets `Kind` — reached only via `parseContractBlocks`
   (parser_contracts.go:22), called only from `parseFunctionDeclaration` (parser_func.go:121), whose
   results are appended to `fn.Properties` (parser_func.go:124-125). Always inside a `FuncDecl`.
3. The collector has exactly three `PropertyCase{…}` sites (collector.go:48 / 84 / 138; **zero** in
   `_test.go`). The only one leaving `Function` nil is `collectPropertyDecl` (:84), fed by
   `*ast.PropertyDecl`; and `parsePropertyDecl` (parser_test_decl.go:75-118) builds its `Property` via
   `parseProperty` (parser_testing.go:235-301), which sets **no** `Kind` ⇒ `PropertyKind`, the zero
   value (ast_decl.go:93). Inline `properties [...]` (parser_func.go:173) likewise.
4. **Behavioral proof**: a module with a passing `ensures` sibling plus a top-level
   `property "commutes" { forall(a: int, b: int) => a + b == b + a }` — the only construct producing
   `Function == nil` — routes to the **forall** path and reports
   `status: fail, error: "test 0: evaluation failed: empty program"`, rc=1. It does **not** emit
   "top-level ensures not supported".

So no user-writable program silently passes through 331/454 today. They are defensive nil-guards,
structurally the same species as the dead `ListType` arm A1 fixes. They are still classified
(`unsupported`, fail-closed) for the future-proofing reason in §2 — but **no fixture and no triage
budget is spent on a live bug that does not exist**, and the evaluator must not expect one.

### R-2 — REFUTES the design doc's Lane A before/after table (survived BOTH quorum rounds)

Doc lines 330-341 use the `mixed` module (`dbl(x: int)` + `headOr(xs: [int], d: int)`) as the single
Lane A example and claim, in the same "After" column, `vacuous_skips: 1` / exit 1 **and**
`headOr … pass, 100 cases (list-arm fix)`. **Probe-measured with the A1 patch applied, that module
yields 2 pass, 0 skip, `success=true`, rc=0** — A1 removes the file's only vacuous skip, so it cannot
also demonstrate a vacuous-pass exit-1. The A-1 fixture must carry a shape Lane A does **not** derive
(record / ADT / tuple). Lane A was certified "byte-identical and unobjected across both quorum
rounds"; this defect survived both, and it would have made A-1 unimplementable as specified.

### R-3 — REFUTES the doc's "Z3-unencodable skip class"

All six `StatusSkip` assignments in the entire non-test Go tree under `internal/` + `cmd/` are
`PropertyResult` sites in runner.go; **`TestResult` is never assigned `StatusSkip` anywhere**
(`reporter.go:148`'s test-skip branch is dead code today). `unencodable_callee_skip.ail`'s single skip
is measured as `requires not satisfied by random input … x=-65.42` — site 536, `out_of_contract`.
There is no Z3-unencodable skip class in `ailang test`. The upside: the six-site taxonomy is
**exhaustive**, which is exactly what makes A-8's totality invariant checkable.

### R-4 — CORRECTS the controller's and the doc's blast-radius counts

Measured across all **33** files in `examples/runnable/contracts/` (the doc/controller worked from 17):
- The genuinely silent set (`success=true` with ≥1 skip) is **6 files, not 5** — the doc's five plus
  **`cross_module_functions_lib.ail`**, which the doc missed and which is the *out_of_contract* shape
  that must stay green. Counts also differ: `park` 7 skips (doc: 6), `record_discovery_verify` 8 (doc: 6).
- Files the controller listed as "have skipping properties", implying silent-green, that are **already
  rc 1 today**: `basic` (4 fails), `invoice` (1), `inbox_injection` (4), `cross_module_functions` (1),
  `finance` (1), `quantifier_verify` (3), `per_function_depth_verify` (4). Their exit codes cannot flip.
- Only **4** files actually flip rc 0 → 1 at M3 (§1.3).

### R-5 — REFUTES the doc's "tests deliberately updated" requirement (it is a no-op)

The doc requires the executor to enumerate every test it updates because "tests asserting
`Success()==true` for pass+skip mixes get deliberately updated". Measured: **that set is empty.**
Every existing test touching `Success()` / `AllSkipped()` / `SuccessAllowingSkips()`
(`result_test.go:126,137,145`; `named_test_test.go:161,164,175,185,299,309`) builds its suite with
`AddTestResult`, and `TestResult` has no `SkipKind`, so `VacuousSkips` stays 0 and every one of them
keeps its current verdict. The nearest candidate, `TestSuiteResult_AllSkipped_False_HasPassed`
(named_test_test.go:170-178), asserts only `AllSkipped() == false`, never `Success()`.
Additive JSON fields are also safe: `reporter_test.go` reads `output["success"].(bool)` by key, and
`integration_test.go:113-125` uses `strings.Contains` — neither asserts a closed key set.
**So the requirement is upgraded from a soft "enumerate what you touched" to the hard, checkable
invariant A-10: no removed lines in pre-existing `_test.go` files.**

### H1 — NEW honesty defect that Lane A itself introduces (live, unlike 331/454)

With the A1 patch applied, `examples/runnable/contracts/list_recursive_verify.ail` goes
**rc 1 → rc 0, `success=true`** — from honestly-red ("NO TESTS RAN", all 6 properties vacuous) to
green — while **4 of its 6 properties are `out_of_contract` skips that executed only 1–2 of 100 cases**
(`containsImpliesNonEmpty_property_1` skip/1, `extractBounded_property_1` skip/2, `_2` skip/1,
`_3` skip/2). It **stays** green after M3, because `out_of_contract` is deliberately forgiven. So this
sprint converts an honest red into a green suite in which two-thirds of the contracts are essentially
unvalidated — the #517 class, one layer over, **live and reachable**.

**Resolution inside Lane A (bounded, deliberate):**
(a) Do **not** change `out_of_contract`'s exit semantics — that is a human-owned High-Impact Decision
in the doc, and flipping it would red 8+ in-repo files. (b) A4's reporter widening already surfaces the
reason **and** the `(N cases)` count for these skips — the plan requires this be *verified for
`out_of_contract`*, not just `no_generator`. (c) `skip_kind` + the existing `tests_run` make the class
machine-countable, so no new counter is needed. (d) The rc 1 → 0 transition is recorded explicitly in
§1.3 so the evaluator reads it as **disclosed, not accidental**. (e) Filed as follow-up **F1**;
a discard-rate threshold is a semantic decision outside Lane A's remit.

### Additional risks not in the doc

- **R-6 file-size gate**: `internal/testing/runner.go` = **774** lines; `make check-file-sizes` fails
  >800 (`make/code-health.mk:123`). The doc's runner.go budget (+25/-5 ⇒ 794) leaves **6 lines**.
  Mitigation in §0.4 — constants and helpers go to `result.go`.
- **R-7 sandbox**: `internal/testing` binds no sockets (informative); `cmd/ailang` has 6
  socket-binding test files, so `go test ./cmd/ailang/...` and `make test` are
  **UNINFORMATIVE UNDER SANDBOX**. §0.3.
- **R-8 manifest drift**: any new file under `examples/runnable/` reds `make verify-examples` via
  `validate_manifest.go --ci` unless `examples/manifest.json` is updated. §0.5.

---

## 7. Velocity & sizing

| Milestone | impl LOC | test LOC | est. |
|---|---|---|---|
| M1 A1 list arm (+ triage) | 12 | 60 | 0.5 d (0.1 code + 0.4 triage) |
| M2 total taxonomy | 35 | 90 | 0.2 d |
| M3 exit/success + reporter | 45 | 110 | 0.2 d |
| M4 JSON-is-JSON | 15 | 30 | 0.05 d |
| M5 closeout + follow-ups | 0 | 0 (+80 docs) | 0.05 d |
| **Total** | **107** | **290** | **1.0 d** |

The doc's ~0.5 d covers M2–M5 accurately. The extra 0.5 d is M1's measured triage of 16
newly-executing properties — real work the doc assigned to Lane B but which A1 triggers.
