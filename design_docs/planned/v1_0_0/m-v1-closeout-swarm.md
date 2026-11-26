# M-V1-CLOSEOUT-SWARM: v1.0.0 Pre-Soak Close-Out Swarm

**Status**: Planned
**Target**: v0.64.0 (last 0.x pre-soak release) → `v1.0.0-rc.1` → `v1.0.0`
**Priority**: P0 (Final engineering close-out before the 7-day soak and `v1.0.0-rc.1`)
**Estimated**: 1 swarm wave (~4–6 agent-hours wall time across 3 parallel slices + integrator + verifier)
**Dependencies**:
- [ROADMAP.md](ROADMAP.md) (Steps 15–16, Close-out)
- [m-v1-release-gates.md](m-v1-release-gates.md) (§3 Release candidates)
- [m-v1-docs-for-1-0.md](m-v1-docs-for-1-0.md) (§D New documents, §M2–M3)
- [m-v1-stdlib-surface-polish.md](m-v1-stdlib-surface-polish.md) (16 `@deprecated` aliases, `importhint.StdlibRenames`)
- [m-v1-simplification-program.md](../m-v1-simplification-program.md) (Phase 5 = Bar `G5`)
- [v1-mission-dashboard.md](../../v1-mission-dashboard.md) (Bar v3: `N = 1` open gate item = `G5`)
**Baseline Commit**: `cc58e9a8a` (`v0.63.0`, branch `release-v063`)
**Created**: 2026-11-26

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Numeric SemVer pre-release ordering (`rc.2 < rc.10`) in `internal/effects/pkg.go` and deterministic stable-tag filtering in docs/release scripts eliminate version-order nondeterminism. |
| A2: Replayability | 0 | Trace format unchanged. |
| A3: Effect Legibility | 0 | No effect-system semantics change; `security-model.md` documents every `--caps` capability honestly. |
| A4: Explicit Authority | +1 | RC tags are blocked from publishing to the prod MCP Registry and from overwriting `STABLE_RELEASE` / GitHub `latest`; `security-model.md` makes `--caps auto` and `[bin]` grant authority explicit. |
| A5: Bounded Verification | +1 | Every new/updated doc sample is verified by `make check-doc-samples` (0 new failures); alias removal is verified by `make verify-stdlib` and `make downstream-check` against `v0.63.0`. |
| A6: Safe Concurrency | 0 | No concurrency changes. |
| A7: Machines First | +1 | Removing 16 duplicate `@deprecated` stdlib names eliminates synonym ambiguity for AI code synthesis while preserving `importhint.StdlibRenames` so `IMP010` still emits machine-actionable fix hints. |
| A8: Minimal Syntax | 0 | No new syntax; `language-syntax.md` catches up to syntax already shipped in `v0.61`–`v0.63`. |
| A9: Cost Visibility | 0 | No cost-model changes. |
| A10: Composability | +1 | Pairwise-disjoint 3-slice swarm contract guarantees zero merge conflicts across stdlib, release CI/semver, and docs. |
| A11: Structured Failure | +1 | Importing any of the 16 removed stdlib aliases produces structured `IMP010` naming the exact replacement (and argument swap note for `std/list.member`). |
| A12: System Boundary | 0 | Core closure unchanged (verified by Bar `G5` simplicity metrics diff). |

**Net Score: +6** → **Decision: Proceed**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted (tightens RC release isolation)
- [x] A7 (Machines First): Reduces stdlib synonym surface and keeps `IMP010` structured hints

---

## Measured Verification Log

Every premise below was measured directly against `release-v063` (`cc58e9a8a`) on 2026-11-26 in `/tmp/v1-closeout-plan`.

| ID | Premise | Command / File Inspected | Measured Result |
|----|---------|--------------------------|-----------------|
| **V1** | Count & locations of `@deprecated` stdlib aliases | `grep -rn -A 3 "@deprecated" std/` | Exactly **16** `@deprecated` exports across **8** files: `std/math.ail` (`abs_Float`, `abs_Int`), `std/string.ail` (`charAt_or`), `std/bytes.ail` (`slice`), `std/array.ail` (`unsafeGet`), `std/io.ail` (`printErr`), `std/rand.ail` (`rand_int`, `rand_float`, `rand_bool`, `rand_seed`), `std/list.ail` (`nth_or`, `head_or`, `last_or`, `member`), `std/fs.ail` (`readFileBytes`, `glob`). |
| **V2** | Repo-internal `.ail` & Go test call sites that break on alias removal | `grep` across `examples/`, `tests/`, `benchmarks/`, `internal/`, `cmd/` | **0** runnable `examples/*.ail` import any of the 16 deprecated stdlib names (note: `std/map.member` and `std/set.member` are NOT deprecated). **3** Go/test files WILL fail unless updated: (1) `internal/pipeline/warn_deprecated_test.go` (`targetDeprecatedStd` at line 265 imports `std/list.head_or`; `TestStdlibRenamesMatchDeprecatedAnnotations` lines 403–427 asserts all 16 exist in `std/`; `TestDeprecatedAliasesForwardToTheirReplacement` lines 433–472 inspects the 16 forwarders), (2) `internal/runner/show_nominal_test.go:25–31` (imports & calls `std/list.head_or`), (3) `internal/smt/testdata/stdlib/list_defaulted.ail:2,14` (imports & calls `std/list.nth_or`). `internal/builtins/fs_test.go` (`effects.Call("FS", "readFileBytes")`) and `internal/runtime/builtins.go` (`_rand_int`) are Go effect/builtin names, not `std/*.ail` exports. |
| **V3** | Post-removal `IMP010` hints & `stability.md` test coupling | `internal/importhint/renames.go:1–66`, `internal/iface/tier_test.go:233–254` | `importhint.StdlibRenames` (16 entries) is designed to **outlive** the aliases so `IMP010` still names replacements. `TestStabilityDocListsEveryDeprecation` (`internal/iface/tier_test.go:233`) asserts `docs/docs/reference/stability.md` has `### Deprecated stdlib names` containing all 16 rows — so `stability.md` MUST keep that table and update prose to past tense ("removed before v1.0.0 soak"). |
| **V4** | Teaching prompt freeze state | `jq '.versions["v0.16.20"]' prompts/versions.json` | Active prompt `prompts/v0.16.20.md` is **hashed** (`3f8cfa17...`). Line 790 already instructs agents to use the camelCase names and says the old spellings "are being removed". Editing `v0.16.20.md` in place without a version bump would break `check-prompt-freeze`. |
| **V5** | Downstream compatibility gate interface | `cat make/downstream.mk` | `make downstream-check CANDIDATE=<bin> [BASELINE=latest\|vX.Y.Z\|<bin>] [TIER=release\|ci] [FETCH_DEPS=0\|1]` builds `bin/downstream` (`./tools/downstream`) and exits `0` (PASS), `2` (unacknowledged break), `3` (instrument error), `4` (NOT_MEASURED over budget). |
| **V6** | `.github/workflows/release.yml` RC hazards | Read `.github/workflows/release.yml` lines 5–7, 446, 495–501, 582–591 | **4 concrete bugs for `v1.0.0-rc.1`**: (1) line 7 has `- '!v*-*'` which blocks ALL RC tags from running `release.yml`; (2) line 500 sets `prerelease: ${{ contains(steps.get_version.outputs.VERSION, '-') }}` but does NOT set `make_latest: false` explicitly; (3) line 446 diffs changelog from `git describe --tags --abbrev=0 HEAD^` (would diff `v1.0.0` against `v1.0.0-rc.N` instead of last final tag `v0.64.0`); (4) lines 582–591 (`mcp-registry` job) has `if: github.event_name == 'push'` with NO pre-release guard, so an RC tag would publish to the official MCP Registry. *(Good news: `cloudbuild-registry-validator.yaml:15` already refuses non-`vX.Y.Z` tags.)* |
| **V7** | Docs version constants & Docusaurus deploy RC hazards | Read `docs/scripts/generate-version-constants.sh:18`, `.github/workflows/docusaurus-deploy.yml:242` | Both run `git tag ... --sort=-v:refname \| head -1` **without filtering `-`**, so tagging `v1.0.0-rc.1` would overwrite `STABLE_RELEASE` / `AILANG_VERSION` on the live docs site with `v1.0.0-rc.1` (and `sort -V` / git `v:refname` even orders `v1.0.0-rc.1` above `v1.0.0`). |
| **V8** | Go semver pre-release ordering | Read `internal/effects/pkg.go:159–177`, `internal/effects/pkg_semver_test.go`, `internal/pkg/version_compat.go:39–46`, `internal/pkg/version_compat_test.go` | `internal/effects/pkg.go:176` compares pre-release suffixes lexically (`return av.pre < bv.pre`), so `"1.0.0-rc.10" < "1.0.0-rc.2"` (WRONG per SemVer 2.0.0 §11). `internal/pkg/version_compat_test.go` has only 1 RC parse case (`0.9.5-rc1`) and 0 `SatisfiesAILANGVersion` cases for `v1.0.0-rc.1`. |
| **V9** | Website docs sync gaps (`v0.61`–`v0.63` features) | `grep` on `docs/docs/reference/language-syntax.md` & `stdlib.md` | **0 mentions** of `|>`, `_?`, `deriving (Json)`, `expr?`, `ailang inspect`, `check --fix`, `v0.62`, or `v0.63` in `language-syntax.md` or `stdlib.md`. `stdlib.md:18` still says the 16 renamed functions "still work but warn `TC_DEPRECATED_001`". |
| **V10** | Missing 1.0 docs & `SECURITY.md` gaps | `ls` on `docs/docs/guides/upgrading-to-1-0.md`, `docs/docs/reference/security-model.md`, `docs/docs/why-1-0.mdx`; read `SECURITY.md:1–60` | None of the 3 new 1.0 pages exist yet (`upgrading-to-1-0.md`, `security-model.md`, `why-1-0.mdx`). `SECURITY.md` still says "AILANG is pre-1.0" (line 5), lists personal email `mark@aitanalabs.com` (line 25), and says `stdlib/` instead of `std/` (line 55). `scripts/doc_samples_baseline.txt` has only **1** remaining entry (`examples/runnable/secrets/sink_function_value.ail`). |
| **V11** | Bar `G5` (Simplification Phase 5) state | `.ailang/state/simplicity/2026-09-14.json` vs `2026-11-09.json`, `design_docs/v1-mission-dashboard.md:29` | `G5` is the **only** open item on Bar v3 (`N = 1`). Last banked simplicity snapshot is `2026-11-09.json` (`eedcd9972`) — stale by 17 days (pre-Go-emitter removal `-29k LOC`!). Between `2026-09-14` and `2026-11-09`: `closure_leak_roots` `5→0` ✓, `closure_platform_packages` `4→0` ✓, `commands_top_level` `89→17` (≤20) ✓, `help_exit0_rate` `100%` ✓, `backend_switches` `6→1` ✓, `env_vars_documented_pct` `11%→98%` (gate 100%), `getenv_outside_config` `388→5` (gate 0), `packages_without_doc` `17→14` (gate 0), `skill_trees` `2→2` (gate 1), `instruction_surface_bytes` `26780→29394` (gate 25600), `tracked_files` `24829→30431` (gate 8000; `eval_results/` intentionally kept per `D6`). Fresh `make simplicity-metrics-fast` run + waiver table closes `G5`. |

---

## High-Impact Decisions (For Mark to Ratify)

| ID | Decision | Options & Recommendation | Chosen By | Deadline | Change Cost |
|----|----------|--------------------------|-----------|----------|-------------|
| **D1** | **Downstream alias-removal policy if `make downstream-check` finds a remaining call site** | **(Recommended)** Fix any remaining downstream call site in its repo (using `tools/migrate-stdlib-renames.sed` + hand-edits) before tagging `v0.64.0`; do NOT re-introduce aliases. | human (Mark) | before `v0.64.0` tag | med |
| **D2** | **Teaching prompt `v0.16.20` during Step 16** | **(Recommended)** Keep `prompts/v0.16.20.md` frozen (avoiding eval-baseline churn before the paired baseline; line 790 already teaches the camelCase names and states the old ones are being removed). Defer updating line 790's tense ("were removed") to `v0.16.21` / `v1.0.0`. | human (Mark) | before Slice 1 merge | low |
| **D3** | **Role email in `SECURITY.md:25`** | **(Recommended)** Replace `mark@aitanalabs.com` with `security@sunholo.com` (or keep GitHub Security Advisories as sole channel if no role alias exists yet). | human (Mark) | before `v1.0.0-rc.1` | low |
| **D4** | **Bar `G5` Simplification Phase 5 waivers** | Several Goals-table rows (`tracked_files` due to `D6` keeping `eval_results/baselines/`, Phase 4 optional items `packages_without_doc`, `skill_trees`, `instruction_surface_bytes`, and the 5 remaining `getenv_outside_config` sites) are above original Sept-14 targets. **(Recommended)** Record exact Phase 0 → `v0.64.0` metrics diff in `g5-simplicity-metrics-fragment.md` and ratify explicit waivers for the Phase 4 / `D6` rows so Bar `G5` closes cleanly (`N = 0`). | human (Mark) | before `v1.0.0` tag | low |
| **D5** | **`pre-1.0` strings in `intro.mdx` & `why-ailang.mdx`** | **(Recommended)** Leave `intro.mdx` and `why-ailang.mdx` "pre-1.0" banners intact on `v0.64.0` (since `v0.64.0` IS still pre-1.0!) and flip them only on the `v1.0.0` release commit (tracked in the `v1.0.0` release checklist). `SECURITY.md` gets the 1.x support-window table now (covering both `1.x` and final `0.x`). | human (Mark) | before Slice 3 merge | low |

### Design Freeze Checklist

- [ ] **D1** ratified (fix any remaining downstream alias callers before `v0.64.0` tag)
- [ ] **D2** ratified (keep `prompts/v0.16.20.md` frozen during Step 16)
- [ ] **D3** ratified (`SECURITY.md` role email vs Advisories-only)
- [ ] **D4** ratified (Bar `G5` waiver policy for Phase 4 / `D6` rows)
- [ ] **D5** ratified (`intro.mdx` / `why-ailang.mdx` `pre-1.0` flip deferred to `v1.0.0` tag commit)

---

## Deferred Decisions (Agent Latitude)

- Exact structure of the synthetic local deprecated module in `internal/pipeline/warn_deprecated_test.go` (`targetDeprecatedStd`).
- Exact helper function signature for numeric SemVer pre-release identifier comparison in `internal/effects/pkg.go`.
- Whether to add a small shell unit test `tools/ci/test_rc_version_filter.sh` alongside `docs/scripts/generate-version-constants.sh` changes.
- Exact prose layout and example snippets inside `upgrading-to-1-0.md`, `security-model.md`, and `why-1-0.mdx`, subject to `make check-doc-samples` passing with 0 new failures and zero prohibited claims from `m-v1-docs-for-1-0.md` §D2.

---

## Non-Goals (Explicitly Out of Scope for This Swarm)

1. **`ailang-multivac` repo edits** (`scripts/release.sh promote` refusing RC unless `ALLOW_PRERELEASE=1`, validator candidate traffic shift M3b): lives in a separate repository; tracked in the `v1.0.0` release runbook.
2. **Stable-surface golden diff tool (`tools/check-stable-surface`, `m-v1-release-gates.md` M6) and `-race` CI jobs (M7)**: separate follow-on work; `make verify-stdlib` already enforces exact `.stdlib-golden/` parity on every PR and pre-push.
3. **Internal-tier docs labelling / 7 fleet-runbook moves (`m-v1-docs-for-1-0.md` M4)**: non-blocking polish; does not gate `v0.64.0` soak or `v1.0.0-rc.1`.
4. **Paired AILANG/Python baseline (ROADMAP Step 17) and ≥7-day soak (Step 18)**: attended post-merge operations run on the `v0.64.0` binary produced by this swarm.
5. **Removing `internal/importhint/renames.go`**: MUST be preserved so `IMP010` continues naming replacements for 0.x programs after the aliases are deleted.
6. **Renaming Go-internal effect/builtin identifiers** (`effects.Call("FS", "readFileBytes")`, `_rand_int`, etc.): internal runtime plumbing, not `std/*.ail` exports.

---

## Swarm Execution Contract (Pairwise-Disjoint Ownership)

All three author slices execute in **parallel** in isolated working copies (`workspace: "share"` or worktrees off `v0.63.0` / `dev`). Their write sets are **strictly pairwise-disjoint** — zero shared files.

| Role | Slice / Responsibility | Exclusive Write Set | Must NOT Touch | Acceptance Gate |
|------|------------------------|---------------------|----------------|-----------------|
| **Slice 1** (`stdlib-drop-deprecated`) | Remove all 16 `@deprecated` exports from `std/*.ail`; update the 3 affected Go/SMT tests; verify `IMP010` rename hints fire for all 16 | `std/math.ail`, `std/string.ail`, `std/bytes.ail`, `std/array.ail`, `std/io.ail`, `std/rand.ail`, `std/list.ail`, `std/fs.ail`, `internal/pipeline/warn_deprecated_test.go`, `internal/runner/show_nominal_test.go`, `internal/smt/testdata/stdlib/list_defaulted.ail` | `.stdlib-golden/*` (integrator), `internal/importhint/renames.go`, `prompts/*`, `docs/*`, `.github/*`, `internal/effects/*`, `internal/pkg/*` | `grep -rn "@deprecated" std/` empty; `go test ./internal/pipeline/... ./internal/runner/... ./internal/smt/... ./internal/importhint/... ./internal/iface/...` green; `make test-core` green; `make verify-examples` green |
| **Slice 2** (`rc-pipeline`) | Make release workflow, docs version scripts, and Go semver 100% `v1.0.0-rc.1`-safe | `.github/workflows/release.yml`, `.github/workflows/docusaurus-deploy.yml`, `docs/scripts/generate-version-constants.sh`, `internal/effects/pkg.go`, `internal/effects/pkg_semver_test.go`, `internal/pkg/version_compat_test.go`, `tools/ci/test_rc_version_filter.sh` (new) | `std/*`, `.stdlib-golden/*`, `docs/docs/*`, `docs/sidebars.js`, `SECURITY.md`, `internal/pipeline/*`, `internal/runner/*`, `internal/smt/*` | `go test -v ./internal/effects -run TestAssetVersionOrdering\|TestSemver` (`rc.2 < rc.10` passes); `go test -v ./internal/pkg -run TestSatisfiesAILANGVersion\|TestParseSemver` green; `bash tools/ci/test_rc_version_filter.sh` green; YAML syntax valid |
| **Slice 3** (`docs-1-0-sync`) | Sync `language-syntax.md` & `stdlib.md` with `v0.61`–`v0.63`; author `upgrading-to-1-0.md`, `security-model.md`, `why-1-0.mdx`; update `stability.md` & `SECURITY.md` & `docs/sidebars.js`; draft Bar `G5` simplicity metrics fragment | `docs/docs/reference/language-syntax.md`, `docs/docs/reference/stdlib.md`, `docs/docs/reference/stability.md`, `docs/docs/guides/upgrading-to-1-0.md` (new), `docs/docs/reference/security-model.md` (new), `docs/docs/why-1-0.mdx` (new), `docs/sidebars.js`, `SECURITY.md`, `design_docs/planned/v1_0_0/g5-simplicity-metrics-fragment.md` (new) | `std/*`, `.stdlib-golden/*`, `.github/*`, `docs/scripts/*`, `internal/*`, `prompts/*`, `scripts/doc_samples_baseline.txt` (must not grow) | `make check-doc-samples` exits 0 (0 new baseline entries); `go test ./internal/iface -run TestStabilityDocListsEveryDeprecation` green; `git grep -n "SIBLING:" docs/docs/` empty |
| **Integrator** (`integrator`) | Merge Slices 1–3; re-freeze `.stdlib-golden/*.json` via `make freeze-stdlib`; run `make simplicity-metrics-fast` and update G5 fragment with live numbers; run `make downstream-check` | `.stdlib-golden/{array,bytes,fs,io,list,math,rand,string}.json`, `.ailang/state/simplicity/<date>.json`, `design_docs/planned/v1_0_0/g5-simplicity-metrics-fragment.md` (numbers refresh) | No unreviewed feature edits | `make verify-stdlib` green; `make test-core` green; `make check-doc-samples` green; `make downstream-check CANDIDATE=bin/ailang BASELINE=v0.63.0` exits 0 |
| **Verifier** (`verifier`) | Independent read-only review of merged diff (per `ROADMAP.md` §7: 8/11 Wave 1–2 PRs had blocking defects caught only in review); drive REPL + WASM build check + negative controls | Read-only (no edits; reports defects to `@integrator` / slice owner) | All files (read-only) | Clean review sign-off; REPL smoke + `make build-wasm` (or wasm test) + `make verify-stdlib` + `make check-doc-samples` + RC filter control all pass |

### Pairwise-Disjoint Verification

- `Slice 1 ∩ Slice 2 = ∅` (std + pipeline/runner/smt tests vs `.github/` + `docs/scripts/` + `internal/effects/` + `internal/pkg/`)
- `Slice 1 ∩ Slice 3 = ∅` (std + Go tests vs `docs/docs/` + `docs/sidebars.js` + `SECURITY.md` + `design_docs/`)
- `Slice 2 ∩ Slice 3 = ∅` (`.github/` + `docs/scripts/` + Go semver vs `docs/docs/` + `docs/sidebars.js` + `SECURITY.md` + `design_docs/`)
- Shared artifacts (`.stdlib-golden/*.json`, `.ailang/state/simplicity/<date>.json`) have **one** owner: `integrator`.

---

## Detailed Slice Specifications

### Slice 1 — `stdlib-drop-deprecated` (ROADMAP Step 16)

1. **Remove all 16 `@deprecated` exports from `std/*.ail`**:
   - `std/math.ail`: remove `abs_Float` (L88–89), `abs_Int` (L91–92)
   - `std/string.ail`: remove `charAt_or` (L191–192)
   - `std/bytes.ail`: remove `slice` (L78–79)
   - `std/array.ail`: remove `unsafeGet` (L71–74)
   - `std/io.ail`: remove `printErr` (L42–43)
   - `std/rand.ail`: remove `rand_int` (L27–28), `rand_float` (L30–31), `rand_bool` (L33–34), `rand_seed` (L36–37)
   - `std/list.ail`: remove `nth_or` (L167–168), `head_or` (L170–171), `last_or` (L173–174), `member` (L447–448)
   - `std/fs.ail`: remove `readFileBytes` (L31–32), `glob` (L220–221)
2. **Update affected tests**:
   - `internal/pipeline/warn_deprecated_test.go`:
     - Update `targetDeprecatedStd` (L265) to import a synthetic local deprecated module instead of `std/list (head_or)`.
     - Invert `TestStdlibRenamesMatchDeprecatedAnnotations` (L403–427) → `TestStdlibHasNoDeprecatedAliases`: assert `len(stdDeprecations(t)) == 0`, assert `len(importhint.StdlibRenames) == 16`, and assert none of the 16 old names are exported by `std/*.ail`.
     - Replace `TestDeprecatedAliasesForwardToTheirReplacement` (L433–472) with `TestRemovedStdlibAliasesEmitIMP010Hint`: verify that importing each of the 16 removed names fails with `IMP010` mentioning its replacement (and `Note` where non-empty).
     - Keep `TestMigrateSedCoversRenames` (L476–495) intact so `tools/migrate-stdlib-renames.sed` remains verified for upgrading 0.x users.
   - `internal/runner/show_nominal_test.go`: replace `head_or` with `headOr` (L11, L25, L30–31).
   - `internal/smt/testdata/stdlib/list_defaulted.ail`: remove `nth_or` from import (L2) and call site (L14, use `nthOr`).
3. **Preserve `internal/importhint/renames.go`**: untouched (required for `IMP010` migration hints).

### Slice 2 — `rc-pipeline` (ROADMAP Step 15)

1. **`.github/workflows/release.yml`**:
   - Line 5–7: replace `- '!v*-*'` with explicit allow of `v*.*.*` and `v*.*.*-rc.*` (e.g. `tags: ['v[0-9]+.[0-9]+.[0-9]+', 'v[0-9]+.[0-9]+.[0-9]+-rc.[0-9]+']` or `'v*'` with `'!v*-*'` replaced by `'!v*-alpha*'`, `'!v*-beta*'`, keeping `v*-rc.*` enabled — verify GitHub Actions glob syntax!). Note: GitHub Actions tag filters use glob patterns, so:
     ```yaml
     tags:
       - 'v*.*.*'
       - '!v*-*'
       - 'v*.*.*-rc.*'
     ```
     (Check Actions negation order: in GitHub Actions `tags:`, a later positive pattern re-includes what an earlier `!` excluded; verify or use explicit `'v[0-9]*.[0-9]*.[0-9]*'` with a job-level tag-shape guard in `ci-gate` that accepts `^v[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?$` and rejects any other hyphenated tag!)
   - Line 446: diff changelog from previous **final** tag:
     ```bash
     PREV_FINAL=$(git describe --tags --abbrev=0 --exclude='*-*' HEAD^ 2>/dev/null || echo "")
     ```
   - Line 500: add `make_latest: ${{ !contains(steps.get_version.outputs.VERSION, '-') && 'true' || 'false' }}` alongside `prerelease: ${{ contains(steps.get_version.outputs.VERSION, '-') }}`.
   - Line 585 (`mcp-registry` job): guard `if: github.event_name == 'push' && !contains(github.ref_name, '-')` so pre-releases never publish to the official MCP Registry.
2. **`docs/scripts/generate-version-constants.sh` (L18)** & **`.github/workflows/docusaurus-deploy.yml` (L242)**:
   - Filter out hyphenated tags when computing `STABLE_RELEASE` / `AILANG_VERSION`:
     ```bash
     STABLE_RELEASE=$(git tag -l 'v*.*.*' --sort=-v:refname | grep -v '-' | head -1)
     ```
   - Add `tools/ci/test_rc_version_filter.sh` testing that in a temp git repo with tags `v0.63.0`, `v0.64.0`, `v1.0.0-rc.1`, `v1.0.0-rc.2`, `STABLE_RELEASE` resolves to `v0.64.0` (never `v1.0.0-rc.*`), and once `v1.0.0` is tagged, resolves to `v1.0.0`.
3. **`internal/effects/pkg.go` (`semverLess`)** & **`internal/effects/pkg_semver_test.go`**:
   - Implement SemVer 2.0.0 §11.4 dot-separated pre-release identifier comparison (numeric identifiers compared as integers, numeric < alphanumeric, shorter prefix < longer).
   - Add table tests covering `1.0.0-rc.1 < 1.0.0-rc.2 < 1.0.0-rc.10 < 1.0.0`, `1.0.0-alpha.1 < 1.0.0-rc.1`, `1.0.0-rc.1 < 1.0.0-rc.1.1`.
4. **`internal/pkg/version_compat_test.go`**:
   - Add tests for `ParseSemver("v1.0.0-rc.1")`, `ParseSemver("v1.0.0-rc.10")`, `SatisfiesAILANGVersion(">=0.40.0", "v1.0.0-rc.1") == true`, `SatisfiesAILANGVersion(">=1.0.0", "v1.0.0-rc.1") == true` (documenting current strip-suffix behavior per `m-v1-release-gates.md` §3).

### Slice 3 — `docs-1-0-sync` (1.0 Docs + Bar `G5`)

1. **`docs/docs/reference/language-syntax.md`**:
   - Add checked `ailang` sections/examples for all `v0.61`–`v0.63` language features:
     - Pipe operator (`|>`)
     - Typed holes (`_?`, `?name`)
     - Unbraced multi-statement match arms & `if` branches
     - `deriving (Json)` for monomorphic records & sum ADTs
     - `let x = expr?` spine error/option propagation
     - Lambda & record destructuring
     - Tooling cross-links (`ailang inspect`, `ailang check --fix`)
   - Every code block verified by `make check-doc-samples`.
2. **`docs/docs/reference/stdlib.md`**:
   - Document `v0.62`–`v0.63` additions (`std/list`, `std/map`, `std/string`, `std/fs` combinators & scripting batteries).
   - Update line 18 prose to reflect that the 16 renamed functions were removed before the v1.0.0 soak (`IMP010` names the replacement).
3. **`docs/docs/reference/stability.md`**:
   - Update `### Deprecated stdlib names` prose to past tense (removed before v1.0.0 soak; `IMP010` names replacement) while **preserving the 16-row markdown table** (`TestStabilityDocListsEveryDeprecation` invariant).
   - Add `## Support Window` section per `D-67`: latest 1.x minor receives bug & security fixes; final 0.x release receives 90 days of security fixes after 1.0.0.
4. **Three new 1.0 pages + `docs/sidebars.js`**:
   - `docs/docs/guides/upgrading-to-1-0.md`: full 0.x → 1.0 migration guide (from `m-v1-docs-for-1-0.md` §D1, with every `SIBLING:` placeholder filled from shipped `v0.58.0`–`v0.64.0` reality).
   - `docs/docs/reference/security-model.md`: full security model (from §D3, capability table verified against `ailang run --help`, `--caps auto`/`[bin]` disclosure, `--policy`, IFC explicit-flow scope, Z3 verification honesty, Sigstore/SLSA supply chain).
   - `docs/docs/why-1-0.mdx`: 1.0 promise, safe claims & honest limits (strictly respecting §D2 banned-claims list).
   - Wire all three into `docs/sidebars.js` under existing categories (zero new top-level categories).
5. **`SECURITY.md`**:
   - Replace `pre-1.0` table with `D-67` 1.x support window; fix `stdlib/` → `std/`; update reporting channel per `D3`.
6. **Bar `G5` Metrics Fragment (`design_docs/planned/v1_0_0/g5-simplicity-metrics-fragment.md`)**:
   - Draft the Phase 0 (`2026-09-14.json`) → `v0.63.0`/`v0.64.0` comparison table for every metric in `m-v1-simplification-program.md` Goals table, marking each row `PASS` or `WAIVER REQUESTED (D4)` with rationale, ready for `changelogs/v1.0-current.md`.

---

## Success Criteria

- [ ] `grep -rn "@deprecated" std/` returns 0 matches; `make verify-stdlib` passes on re-frozen `.stdlib-golden/`.
- [ ] `make downstream-check CANDIDATE=bin/ailang BASELINE=v0.63.0` exits `0` (or every break is fixed downstream before tagging `v0.64.0`).
- [ ] `go test ./internal/effects -run TestAssetVersionOrdering\|TestSemver` proves `1.0.0-rc.2 < 1.0.0-rc.10 < 1.0.0`.
- [ ] `tools/ci/test_rc_version_filter.sh` proves `v1.0.0-rc.1` never overwrites `STABLE_RELEASE`; `release.yml` sets `prerelease: true`, `make_latest: false`, skips `mcp-registry`, and diffs release notes from the previous final tag.
- [ ] `make check-doc-samples` exits `0` with 0 new baseline entries; `docs/docs/guides/upgrading-to-1-0.md`, `docs/docs/reference/security-model.md`, `docs/docs/why-1-0.mdx`, and `g5-simplicity-metrics-fragment.md` exist and are linked.
- [ ] Independent `verifier` signs off after checking REPL + WASM build + `make test-core`.
