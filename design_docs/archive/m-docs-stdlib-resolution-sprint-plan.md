# Sprint Plan: Docs stdlib resolution for release installs

> **Superseded (2026-09-26)** by [M-STDLIB-ROOT-RESOLUTION](../planned/v0_44_0/m-stdlib-root-resolution.md),
> which unifies every stdlib resolver (loader, `ailang docs`, import hints) behind one
> `internal/stdlibroot` package ending at the embedded `std.FS`. PR #1197, which implemented
> this doc, was closed in favour of it. Kept for history; see "Relation to the earlier doc"
> in the new doc for what this one got right and what had gone stale.

## Summary

Make `ailang docs` use the same ordered filesystem locations as the compiler, fall back to the existing version-matched embedded `std.FS`, and package the full stdlib with release archives. This implements the approved deployment/UX ruling for GitHub issue #1131 without changing AILANG semantics, package-registry behavior, or docs-search embeddings.

**Duration:** 2 days (about 12 engineering hours)  
**Target:** v0.38.6  
**Dependencies:** Approved `design_docs/planned/m-docs-stdlib-resolution.md`; existing `std.FS` embed  
**Risk Level:** Medium

## Current Status Analysis

### Existing capability

- `cmd/ailang/docs.go` has a private three-location resolver and requires a filesystem directory.
- `internal/loader/stdlib_resolver.go` already implements the richer filesystem search, but exposes module lookup rather than stdlib-root discovery.
- `std/embed.go` already embeds all `.ail` sources as `std.FS`; the loader already uses it as a final module fallback. No second embedded copy is needed.
- `.github/workflows/release.yml` currently archives only the executable. The design's referenced `RELEASING.md` is not present, so the workflow and its tests are the release source of truth.

### Velocity and scope

Only one repository commit is visible in the last seven days, so recent history does not provide a meaningful LOC/day baseline. The estimate instead uses the approved two-day design estimate plus a 25% integration buffer. The work is localized Go/CI deployment code: about 410 changed lines over two days, including tests.

### Required outcomes

- One explicit root-resolution order shared by docs and loader: CLI override where supported, `AILANG_STDLIB_PATH`, development/repo paths, binary-relative, user-data, and system locations.
- Docs can read full sources through either an on-disk root or the existing embedded FS.
- Unix and Windows release archives contain `bin/ailang` (or `bin/ailang.exe`) and `std/` in the loader-compatible relative layout.
- Genuine absence remains a non-zero, search-trace-bearing error; no cache, network, registry, signature-only, or empty fallback is introduced.

## Proposed Milestones

### M1: Shared filesystem root resolution

**Goal:** Remove the private docs lookup policy and expose one loader-owned filesystem-root resolution API with an inspectable search trace.  
**Estimated:** 80 LOC implementation + 90 LOC tests = 170 LOC  
**Duration:** Day 1 morning

**Files to update:**

- `internal/loader/stdlib_resolver.go`
- `internal/loader/stdlib_resolver_test.go`
- `cmd/ailang/docs.go`

**Tasks:**

- Add a loader API that resolves a valid stdlib root, rather than making docs infer a root from a resolved module filename.
- Make search-order precedence explicit and testable, including multi-path `AILANG_STDLIB_PATH`, binary-relative, user-data, and system candidates.
- Return the attempted roots in the terminal error and retain the environment-variable remedy.
- Replace `findStdlibDir`'s duplicated candidate list with the shared API.

**Acceptance Criteria:**

- [ ] Docs and compiler filesystem resolution are driven by the same loader-owned candidate builder.
- [ ] Table tests pin precedence and path-list handling across supported OS conventions.
- [ ] Invalid overrides are reported in the search trace instead of being silently treated as success.
- [ ] Existing loader resolution/version tests pass.

**Risk:** Changing precedence can affect users with both repo-local and environment stdlibs. **Mitigation:** Pin the approved override-first order in tests and document the behavior in code.

### M2: Full-source embedded docs fallback

**Goal:** Let every stdlib docs mode consume full `.ail` sources from the existing `std.FS` when no filesystem root exists.  
**Estimated:** 90 LOC implementation + 110 LOC tests = 200 LOC  
**Duration:** Day 1 afternoon through Day 2 morning

**Files to update:**

- `cmd/ailang/docs.go`
- `cmd/ailang/docs_all_functions.go` and/or its helpers, only where required by the source abstraction
- `cmd/ailang/docs_all_functions_test.go`
- `cmd/ailang/docs_examples_test.go`
- New focused test file such as `cmd/ailang/docs_resolution_test.go`

**Tasks:**

- Introduce a small docs source abstraction based on `fs.FS`; adapt an on-disk root with `os.DirFS` and use `std.FS` as the final fallback.
- Route module listing, module rendering, type/signature parsing, `--examples`, and `--all-functions` through that abstraction.
- Preserve live filesystem sources ahead of embedded sources for repository development.
- Add isolated-process tests that clear resolution inputs and prove embedded fallback works without a repository or home stdlib.
- Add an injectable no-embedded-source test seam to prove genuine absence still exits non-zero with searched locations and the tarball/env remedy.

**Acceptance Criteria:**

- [ ] `ailang docs --list`, `ailang docs std/io`, `--examples`, and `--all-functions` work using only embedded full sources.
- [ ] A filesystem override wins over embedded content.
- [ ] Docs output from filesystem and embedded copies is equivalent for the tagged tree.
- [ ] Genuine absence fails loudly and reports every filesystem tier tried.
- [ ] `docs search` and `docs embed-warmup` behavior is unchanged.

**Risk:** Existing parsing helpers accept file paths and may be coupled to `os.*`. **Mitigation:** Keep the abstraction narrow (`ReadDir`/`ReadFile`) and add parity tests before mechanical conversion.

### M3: Release archive capability gate

**Goal:** Ship full stdlib sources beside `bin/` and prove an extracted archive works without repository state or environment configuration.  
**Estimated:** 20 LOC workflow + 20 LOC validation/docs = 40 LOC  
**Duration:** Day 2 afternoon

**Files to update:**

- `.github/workflows/release.yml`
- A repository-native release verification script/test under `scripts/` if workflow inline checks are insufficient
- Release/install documentation that actually exists, if extraction commands need adjustment

**Tasks:**

- Stage Unix and Windows archives with `bin/<executable>` plus the complete `std/` directory, preserving `std/VERSION`.
- Verify archive contents before upload and run extracted `docs --list` plus a stdlib-importing `check` smoke test with `AILANG_STDLIB_PATH` unset and outside the repository.
- Update published extraction examples if the new directory layout changes the executable path.

**Acceptance Criteria:**

- [ ] Unix tarballs and Windows zip files contain the executable and all 45 current `.ail` files plus `std/VERSION`.
- [ ] The archive layout satisfies the loader's `<bindir>/../std` lookup.
- [ ] An extracted archive passes zero-config docs and compiler stdlib smoke tests.
- [ ] Archive verification fails if `std/` is omitted or incomplete.

**Risk:** Moving the executable under `bin/` changes current one-line install examples. **Mitigation:** update those examples atomically and test the exact documented extraction invocation.

## Day-by-Day Plan

### Day 1

1. Write failing resolver precedence/root tests, then implement the shared loader API.
2. Write embedded-vs-filesystem docs source tests.
3. Convert `--list` and module detail rendering to the FS abstraction, then extend it to all docs modes.
4. Run focused loader and docs tests plus `make fmt`.

### Day 2

1. Complete parity, examples, all-functions, and honest-failure tests.
2. Stage full stdlib sources in Unix and Windows release archives.
3. Add archive-content and extracted-install smoke gates.
4. Run `go test ./cmd/ailang ./internal/loader`, `make test`, `make lint`, and `make check-boundaries`.

## Success Metrics

- Fresh darwin/arm64 and linux/amd64-style archive extractions run `ailang docs --list` without an environment variable or repository checkout.
- A stdlib-importing program checks successfully from the extracted archive.
- All four docs reading modes are covered against the embedded source.
- Search precedence and honest terminal errors are pinned by tests.
- No new download/cache/registry path and no change to docs-search embeddings.
- `make test`, `make lint`, and `make check-boundaries` pass.

## Dependencies and Assumptions

- The approved ruling authorizes the release layout change and full-source distribution.
- `std.FS` remains the canonical embedded source; this sprint must not add a duplicate embed under `cmd/ailang`.
- GitHub issue #1131 is the tracked external report; milestone commits should use `refs #1131`, with the final implementation commit using `Fixes #1131` when appropriate.
- No `.ail` source is authored or edited, so `ailang prompt` is not required for this sprint plan.

## Deferred / Non-Goals

- Cached downloads or network resolution.
- Registry distribution of the stdlib.
- Signature-only generated artifacts.
- Changes to `docs search`, neural embeddings, or package docs.
- New language semantics.

## Approval Gate

After this plan is approved, the user must explicitly say **execute sprint** before `sprint-executor` implements it.
