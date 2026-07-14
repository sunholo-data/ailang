# M-DX-EXAMPLES-COVERAGE: Close the Examples Gap — Coverage, a Real CI Gate, and a Working `--examples` Flag

**Status**: Implemented (2026-07-14, mission iteration 29 — PR #392 → squash `3d451947c`, dev
CI green per-workflow observed; re-scoped same day from the fully-stale v0.10.1-era version and
reviewed through the FIRST LIVE 5-round design quorum before planning)

## Implementation Report (2026-07-14)

Executed headless by the Opus sprint-executor in worktree `sprint/m-dx-examples-coverage`
(5 milestone commits + 1 hardening), independent evaluation round 1 FAIL 81/100 (single
sprint-introduced Windows path-separator defect) → hardening `881711325` → round-2 PASS, all
PR checks green including both Windows jobs.

- **M1 (`3b6ab098f`)**: bisect INCONCLUSIVE (first-bad landed on a docs-only commit —
  corrupted by go-build binary caching under `2>/dev/null`); per decision rule → quarantine.
  Real trigger additionally root-caused by in-file bisection: `show()` inside a string
  interpolation inside an effectful lambda collapses the combinator effect row to closed-empty
  (4 files); mcp_tools is a separate `getString`→`Option[string]` API change. All 5 quarantined
  (skippedExamples reason = issue URL, manifest `status: broken`) under **issue #386** (owner +
  5-file closing checklist). No `internal/types`/`internal/effects` changes (forbidden zone
  verified empty by the evaluator).
- **M2 (`d8b43e7c0`)**: 6 new deterministic examples — all six zero-importer modules covered.
- **M3 (`a283caafd`)**: all three `|| true` defeat layers fixed; artifacts written
  unconditionally then exit re-raised; `if: always()` on the trace step; `validate_manifest
  --ci` wired; gate self-test. Non-vacuity proven BOTH directions by the evaluator (sprint:
  broken example → exit 2 + artifacts written; base: same breakage → exit 0).
- **M4 (`6352c6d35`)**: additive manifest `modules` field, parser-backed extraction
  (`scripts/internal/importextract`, ast.File.Imports) shared by backfill + drift lint;
  `docs --examples` Try-it section; installed-binary integration test (AILANG_EXAMPLES temp
  layout, no network). 108 entries backfilled.
- **M5 (`b40d976f8`)**: CHANGELOG + issue #341 comment. Hardening (`881711325`): slash-form
  Try-it paths (Windows), mcp_tools quarantine manifest entry + stats recalc.
- **Declared deviations**: `validate_manifest.go` REWRITTEN (legacy version could not even
  load the real manifest — wrong path prefixes, nil-deref, no --caps; evaluator judged the
  relaxation justified: old strict Validate would have rejected the real manifest, and the
  old validator was wired into nothing); `aspirational` status added to the schema;
  SharedIndex cap spec added to the verifier.
- **Known follow-ups**: #386 (effect-row regression under the 5 quarantined examples);
  examples_report.json invalid-JSON-on-failure (pre-existing, now the gating case — backlog);
  examples_status.md doesn't show skip reasons; manifest `expected` unenforced (pre-existing);
  self-tests manual-only.
**Target**: v0.30.x
**Priority**: P2 (DX / clause-3 accessibility — mid-tier models and humans both learn from
runnable examples; a silent example-rot gate is a standing trust leak)
**Estimated**: 2 days (4 phases, each independently landable)
**Dependencies**: None (issue #341 is absorbed as Phase 1, not a dependency)
**Bug Report**: originally M-EVAL-XLANG benchmark analysis; re-grounded on GitHub issue #341 +
live HEAD verification 2026-07-14

## Related Documents

- GitHub issue #341 — "5 runnable examples fail type-check on dev (pre-existing;
  verify-examples not a CI gate)" (absorbed as Phase 1)
- Neural search top hits all < 0.45 (tier0_completion_summary 0.46 historical status doc —
  distinct; m-codegen-stdlib-math 0.44 — codegen, distinct). No planned doc overlaps.

## Problem Statement

Four verified gaps, one theme: **the examples surface looks maintained but is silently rotting,
and the tooling that should surface examples doesn't.**

1. **Coverage holes**: 6 std modules have ZERO example importers (`std/embedding`, `std/game`,
   `std/gzip`, `std/sharedindex`, `std/simhash`, `std/smoke`) and 17 more have exactly one.
   All six are real user-facing modules (headers read live: vector embeddings, frame timing,
   gzip, semantic index, LSH, package smoke-probe helpers). An AI agent looking for "how do I
   use std/gzip" finds nothing runnable and burns turns on trial-and-error (the original doc's
   benchmark evidence: 3–5 wasted turns/trial writing throwaway `/tmp` probes).
2. **5 examples are RED at HEAD and CI is green** (issue #341): `effectful_list.ail`,
   `effectful_list_t7_chain_combinators.ail`, `mcp_tools.ail`, `stream_multi_source.ail`,
   `stream_process_source.ail` all fail `ailang check` with effect-row unification errors
   (representative: `incompatible closed rows: r1 has extra labels [], r2 has extra labels
   [IO]` at effectful_list.ail:25:21). Untouched since ~v0.13.0; something under them changed.
3. **The CI gate is triple-defeated** (verified in code, all three layers):
   - `.github/workflows/ci.yml:205` runs `make verify-examples … || true`;
   - its `grep -q "Failed: 0"` success-detection can NEVER match — the actual summary format is
     `**Summary:** 186 passed, 5 failed, 5 skipped`, so `status=failure` is emitted every run
     and `steps.verify.outputs.status` is consumed NOWHERE (repo-wide grep: zero references);
   - `make/examples.mk:12-13` itself wraps both `go run ./scripts/verify_examples.go` calls in
     `|| true`, so even local `make ci` (which lists verify-examples, `make/ci.mk:11`) cannot
     fail on a rotted example — the "exit status 1" text seen in terminal output is `go run`'s
     stderr echo, swallowed before make sees it.
4. **`ailang docs --examples` is inert**: output is byte-identical with and without the flag
   for `std/list` (51 example importers exist). Mechanism (cmd/ailang/docs.go): the flag only
   prints `mod.Examples` (line 328), populated exclusively from stdlib source comments
   (lines 185–209) — empty for every module today. It never consults `examples/manifest.json`
   or the 163 files in `examples/runnable/`.

## Verification Log (live at HEAD v0.29.2-177-g4f486f0c2, 2026-07-14)

| Claim | Check | Result |
|---|---|---|
| 6 modules zero example importers | `for m in std/*.ail: grep -rl "import std/$m" examples/` | embedding, game, gzip, sharedindex, simhash, smoke = 0; 17 modules = 1 (delta vs iter-28 probe: trace now has 1; smoke added) |
| 5 examples red at HEAD | `make verify-examples` fresh binary | `186 passed, 5 failed, 5 skipped`; direct `ailang check examples/runnable/effectful_list.ail` → effect-row unification error at 25:21 |
| red examples pre-date this work | `git log` on effectful_list.ail | last touched 99f76ec7a (v0.13.0); issue #341 proved identical failure at v0.29.0-2 |
| CI step cannot fail | read ci.yml:203-212 + repo grep `steps.verify` | `\|\| true` + unmatchable grep + status output unused (0 refs) |
| make recipe swallows exit | read make/examples.mk:10-14 | `\|\| true` on both go-run lines |
| `--examples` inert | `diff <(ailang docs list) <(ailang docs --examples list)` | byte-identical (exit 0, no diff) |
| `--examples` mechanism | read cmd/ailang/docs.go:185-209,274,327-334 | comment-parsed only; no manifest/examples linkage |
| manifest structure | `examples/manifest.json` | 176 entries, keys: path, status, tags, description, expected |
| report artifacts have consumers | repo grep examples_report.json/examples_status.md | tools/build-snapshot, scripts/update_readme.go, scripts/flag_broken_examples.go, scripts/update_docs_examples.go, docusaurus-deploy.yml — generation must be preserved |
| installed-binary examples resolution exists | read cmd/ailang/examples.go:669-710 | `findExamplesDir`: AILANG_EXAMPLES → exe-relative → ~/.ailang/examples (via `ailang examples download`, GitHub releases) → CWD; explicit actionable error on exhaustion (line 709) |
| downloader ships manifest + layout matches (quorum r2 objection) | read release.yml:135-148 + examples.go:470-556 | release packages `zip -r examples.zip . -i '*.ail' '*.json'` from examples/ (manifest.json INCLUDED); downloader extracts to `~/.ailang/examples/` stripping one top-level dir and explicitly verifies `manifest.json` exists post-extract (examples.go:553) — layout matches what `loadExamplesManifest` reads (`<dir>/manifest.json`, `<dir>/runnable/`) |
| CI steps after "Verify examples" (quorum r2 objection) | read ci.yml:203-218 + docusaurus-deploy.yml:124-134 | ONLY "Verify example determinism (traces)" follows in the job; docusaurus-deploy REGENERATES its own examples_report.json (line 130) and local tools run the verifier themselves — none consume CI's in-job artifact |
| all examples/manifest.json consumers (quorum r3 objection) | repo grep manifest.json --include=*.go | cmd/ailang/examples.go, internal/manifest/manifest.go, scripts/validate_manifest.go; pipeline/cache_store.go is a DIFFERENT manifest (compile cache) |
| no existing module→example lookup to reuse (quorum r3 objection) | read scripts/update_docs_examples.go:15-40 + examples.go:156-330 | update_docs_examples reads examples_report.json (verifier output, rendering only); examples search = fuzzy text scoring, wrong semantics for exact module match |
| validate_manifest.go exists but unwired | repo grep validate_manifest in make/ + .github/workflows/ | zero references — has --ci flag, never gates; Phase 3/4 wire + extend it |
| quarantine mechanism (quorum r5 objection) | read internal/manifest/manifest.go:23-29,208-226 + scripts/verify_examples.go:83-136 | valid statuses = working/broken/experimental ONLY (invalid → validation error); verifier skips via its own hardcoded `skippedExamples` map with reasons, NOT the manifest — quarantine = map entry (issue URL as reason) + manifest `broken` + filed follow-up issue |

No new error codes proposed. No parser/type/codegen changes in scope (see Conflict Surface).

## Goals

1. **Zero red runnable examples at HEAD** — the 5 failures triaged per testing policy.
2. **Every user-facing std module has ≥1 runnable example**, registered in
   `examples/manifest.json` with expected output.
3. **`verify-examples` actually gates**: a rotted example turns dev CI red (all three defeat
   layers fixed), while report artifacts keep being generated for their 5 consumers.
4. **`ailang docs --examples <module>` shows real examples** — a "Try it" section listing
   registered runnable examples that import the module, with run commands.

## Solution Design (4 phases, each independently landable)

### Phase 1 — Triage the 5 red examples (timeboxed: half-day HARD)

Bisect ONE representative (`effectful_list.ail`) to the breaking commit (`git bisect run
sh -c 'go build -o /tmp/bisect-ailang ./cmd/ailang && /tmp/bisect-ailang check
examples/runnable/effectful_list.ail'`). Decision rule (testing policy: no backward compat,
delete out-of-date tests):

- **Deliberate semantics change** (likely: effect-row strictness landed after v0.13.0) →
  UPDATE the 5 examples to current semantics (or delete + replace with a modern equivalent
  demonstrating the same pattern). Record the breaking commit in the commit message.
- **Genuine regression** → do NOT fix the type system in this sprint (out of scope, Conflict
  Surface). Quarantine mechanism (VERIFIED, quorum r5 — the earlier `status: "known-broken"`
  draft was wrong twice over: valid manifest statuses are exactly `working`/`broken`/
  `experimental` (internal/manifest/manifest.go:27-29; anything else fails validation at
  :226), AND the verifier never consults the manifest for skipping — it uses its own
  hardcoded `skippedExamples map[string]string` exclusion list with per-file reasons
  (scripts/verify_examples.go:87,126)). So the quarantine is: (a) add the 5 files to
  `skippedExamples` with reason = the follow-up GitHub issue URL, (b) set their manifest
  status to `broken` (the valid enum), (c) FILE that follow-up issue carrying the bisect
  result, an owner (the mission queue), and the quarantine list as its closing checklist —
  no indefinite unowned skip. Skips print visibly in the summary (`SKIP (reason)`), never
  silently.

Timebox expiry (bisect inconclusive in half a day) → treat as genuine-regression branch.
Either branch unblocks Phase 3.

### Phase 2 — Coverage for zero-importer modules (~half-day)

Six new example files in `examples/runnable/` (names: `stdlib_embedding.ail`, `stdlib_game.ail`,
`stdlib_gzip.ail`, `stdlib_sharedindex.ail`, `stdlib_simhash.ail`, `stdlib_smoke.ail`), each
~30–60 LOC showing the module's headline patterns (not bare function calls), deterministic
output, registered in `examples/manifest.json` with `expected`. Capability notes per module
header (game needs Clock; sharedindex needs SharedIndex; gzip/simhash/embedding pure or IO-only;
smoke demonstrates the `_smoke.ail` one-liner pattern for package authors).

The 17 one-importer modules are OUT of scope (diminishing returns for a 1–2d sprint); listed
here so the next coverage pass starts from data.

### Phase 3 — Make the gate real (~2h, mechanical, guard-the-call-site)

- `make/examples.mk`: capture `go run ./scripts/verify_examples.go --json`'s exit code, still
  write both artifacts, exit non-zero AFTER printing the status markdown. (Keep artifact
  generation unconditional — 5 consumers.)
- `.github/workflows/ci.yml` "Verify examples" step: drop `|| true` and the dead
  `Failed: 0` grep + unused `status` output; let the make target's exit code gate.
- **Step-lifecycle (quorum r2, gemini-3-1-pro — addressed)**: a non-zero step aborts the
  job's remaining steps by default. Verified blast radius: the ONLY subsequent step in the
  job is "Verify example determinism (traces)" (ci.yml:216) — add `if: always()` to it so
  the trace signal still prints on a red gate. No other consumer is affected:
  docusaurus-deploy.yml regenerates its own `examples_report.json`
  (docusaurus-deploy.yml:130) and the local script consumers run the verifier themselves —
  none read CI's in-job artifact.
- Add a regression guard ON THE GATE ITSELF: a CI-adjacent test (or a `make` self-test target)
  that runs the verifier against a deliberately-broken fixture and asserts non-zero exit —
  the lesson from the env-forward class: guard the call-site, not the helper.
- `skipped` stays non-fatal (AI/network examples skip WITH logged reason, as today).

### Phase 4 — Wire `--examples` to reality (~2–3h)

**Path resolution outside a source checkout (quorum objection, 2026-07-14, gpt5-6-sol —
addressed):** Phase 4 does NOT invent path discovery. It reuses `loadExamplesManifest` →
`findExamplesDir` (cmd/ailang/examples.go:649-710, read live), which already resolves
deterministically in fixed order: (1) `AILANG_EXAMPLES` env var, (2) executable-relative
`../examples`/`examples` (local builds), (3) `~/.ailang/examples/` — populated by the existing
`ailang examples download` command that ships examples+manifest via GitHub releases (the
installed-binary path), (4) CWD-relative (inside the repo). On exhaustion it returns an explicit
actionable error ("examples not found … 1. ailang examples download …", examples.go:709). So
the installed-binary case is a SOLVED, shipped mechanism; `--examples` inherits it. When
resolution fails, the flag prints that same explicit error text as a note under the module doc
(doc output still shown) — never a silent fallback, never byte-identical-to-flagless output.

**Lookup mechanism (DECIDED, quorum r2/r3 — no implementer's choice left open):** the
manifest gains an optional `modules: ["std/gzip", …]` array per entry. Phase 4 backfills it
for all 176 existing entries ONCE, mechanically (a small Go script scanning each registered
file's `import std/…` lines — run at development time, committed to the manifest; NOT a
runtime scan). `docs --examples <module>` matches ONLY on the committed `modules` field — a
single linear pass over the ~176 in-memory entries (trivial at this scale; "O(1)" claimed in
an earlier revision was wrong, corrected per quorum r3), deterministic, works identically
installed and in-repo.

**Schema-addition blast radius (verified, quorum r3):** consumers of
`examples/manifest.json` are exactly: `cmd/ailang/examples.go` (its own `ExampleEntry`
struct), `internal/manifest/manifest.go` (`manifest.Load`, the canonical schema — the
`modules` field is added HERE and mirrored in `ExampleEntry`), and
`scripts/validate_manifest.go` (loads via `internal/manifest`). `internal/pipeline/
cache_store.go`'s manifest.json is a DIFFERENT file (compile cache under `.ailang/cache/`),
not a consumer. All parsers are `json.Unmarshal`-based → unknown/optional fields are ignored;
the addition is additive.

**Drift enforcement (quorum r3, gemini-3-1-pro — addressed; this doc's own silent-rot
standard applied to itself):** the committed `modules` field MUST NOT be maintainable by
hand-memory. `scripts/validate_manifest.go` already exists for exactly this purpose
("ensures documentation stays in sync with reality", has a `--ci` flag) but is wired into
NEITHER make nor CI (repo grep: zero references). Phase 4 extends it with a modules-vs-actual-
imports assertion (fail on drift with the exact regeneration command in the error) and Phase 3
wires `validate_manifest --ci` into the same gating CI step. A new example whose manifest
entry is missing/stale therefore turns CI red — same gate, no new silent surface.

**Import extraction is PARSER-BACKED, one canonical implementation (quorum r4, gpt5-6-sol —
addressed):** no line/regex scanning anywhere in this design. Both the one-time backfill
script and the validate_manifest drift assertion call ONE shared function that parses each
example with the real parser (`internal/parser` `Parser.ParseFile()` → `ast.File.Imports
[]*ImportDecl`, both verified present — ast.go:140) and reads the `std/…` module paths from
the AST. Aliases, selective imports, formatting, comments, and duplicates are therefore
handled by the same grammar the compiler uses — the CI authority cannot disagree with the
language. (An example that fails to PARSE is already red via the verify gate itself, so the
extractor never guesses at malformed input.)

**Reuse-vs-rebuild justification (quorum r3, gpt5-6-sol — verified):**
`scripts/update_docs_examples.go` reads `examples_report.json` (the verifier's OUTPUT) and
renders a status table for the docs site — it performs no module→example discovery; nothing
to reuse for lookup. `ailang examples search` is fuzzy text scoring over
descriptions/tags/content (examples.go:156-330) — wrong semantics for exact module matching
(a search for "gzip" also hits prose mentions). Phase 4 reuses the SHARED pieces: the
`internal/manifest` schema, `loadExamplesManifest`/`findExamplesDir` resolution, and
`validate_manifest.go` for enforcement; only the ~30-line filter+render in `docs.go` is new.

**Installed-binary premise (quorum r2, gpt5-6-sol — verified end-to-end, see Verification
Log):** the release workflow packages `manifest.json` inside `examples.zip`
(release.yml:138, `-i '*.ail' '*.json'`), and the downloader extracts to
`~/.ailang/examples/` stripping one top-level dir and explicitly checks `manifest.json`
exists post-extract (examples.go:553). Required test: an integration test running the built
binary with `AILANG_EXAMPLES` pointed at a temp directory laid out like a download
(manifest.json + runnable/*.ail copied there) and CWD outside any source checkout, asserting
`docs --examples` prints the Try-it section — proving the no-source-checkout path without
network.

`cmd/ailang/docs.go`: when `--examples <module>` is passed, load the manifest via
`loadExamplesManifest` (above), filter entries whose `modules` field contains
`std/<module>`, and print a "Try it" section:

```
## Try it

  examples/runnable/stdlib_gzip.ail — compress/decompress round-trip
    ailang run examples/runnable/stdlib_gzip.ail

  (2 more: ailang examples search gzip)
```

Comment-derived `mod.Examples` printing stays (additive). If no examples match, say so
explicitly ("No registered examples import std/<module> yet") — never silently identical
output (that's the current bug).

## Conflict Surface

This work touches `cmd/ailang/docs.go`, `make/examples.mk`, `.github/workflows/ci.yml`,
`examples/`, `examples/manifest.json` — NOT parser/lexer/types/elaborate/codegen/eval.

1. **Explicitly forbidden in this sprint**: any change under `internal/types/` or
   `internal/effects/` to make the 5 red examples pass. If the bisect shows a genuine
   regression, that fix is its own item (Conflict Surface mandatory there, not here).
2. **Programs that must still work**: the CI "Verify examples" step must keep producing
   `examples_output.txt` content for log readers; `examples_report.json` +
   `examples_status.md` must keep being generated on BOTH pass and fail (consumers:
   tools/build-snapshot/main.go:1104, scripts/update_readme.go, scripts/flag_broken_examples.go,
   scripts/update_docs_examples.go, docusaurus-deploy.yml).
3. **CI blast radius**: after Phase 3, ANY example rot turns dev CI red. That is the point —
   but Phase 3 must land in the SAME PR as (or after) Phase 1, never before, or we ship a
   permanently-red gate. Sequencing constraint: 1 → (2, 3 in either order) → 4.
4. **`docs --examples` disambiguation**: flag semantics change from "print comment examples"
   (never fires) to "print comment examples + Try-it section". Additive; `ailang docs
   <module>` without the flag is byte-identical to today.
5. **Windows CI** (test-windows job) does not run verify-examples — unaffected.

## Axiom Compliance

Net: **+6** (A2 determinism +1: examples must have deterministic expected output to gate;
A5 explicitness +2: a gate that can fail loudly replaces three silent `|| true` layers —
this is the NO-SILENT-FALLBACKS principle applied to CI; A8 AI-friendliness +2: runnable
examples are the highest-leverage training/prompt surface per the original benchmark evidence;
A11 tooling honesty +1: a documented flag that does nothing is a lie, fixed. No axiom scored
negative; no A1/A3/A4/A7 violations — no language semantics touched.)

## Success Criteria

- [ ] `make verify-examples` → 0 failed at HEAD (5 reds triaged per decision rule)
- [ ] Bisect result for effectful_list.ail recorded (breaking commit SHA or timebox-expired note)
- [ ] If quarantine branch taken: follow-up issue filed (bisect result + owner + the
      skippedExamples entries as its closing checklist) — no unowned indefinite skip
- [ ] 6 new `stdlib_*.ail` examples pass `ailang check` + run with expected output, registered
      in manifest
- [ ] A deliberately-broken example turns `make verify-examples` exit non-zero (self-test
      proves the gate) AND artifacts are still written
- [ ] CI "Verify examples" step fails on a red example (observed on the PR branch, e.g. by a
      temporary broken fixture commit that is then reverted, or the self-test)
- [ ] `ailang docs --examples gzip` prints a Try-it section; `ailang docs gzip` unchanged
- [ ] `ailang docs --examples <module-with-no-examples>` says so explicitly
- [ ] Integration test: built binary + `AILANG_EXAMPLES`→temp download-layout dir + CWD
      outside the repo → Try-it section prints (installed-binary path proven, no network)
- [ ] `modules` field backfilled for all manifest entries (mechanical script, committed)
- [ ] `validate_manifest --ci` wired into the gating CI step; a deliberately-drifted
      `modules` entry turns it red (drift-lint proven, same self-test pattern as the gate)
- [ ] CHANGELOG.md updated; issue #341 closeable (link the PR)

## Timeline

- Day 1 AM: Phase 1 (timeboxed bisect + example updates or quarantine)
- Day 1 PM: Phase 2 (6 examples + manifest)
- Day 2 AM: Phase 3 (gate + self-test) — same PR or after Phase 1
- Day 2 PM: Phase 4 (docs --examples) + CHANGELOG + issue #341 comment

## Non-goals

- Fixing whatever broke effect-row unification for the 5 examples (separate item if real)
- Examples for the 17 one-importer modules (next pass; data recorded above)
- `verify-examples-all` threshold changes, trace-determinism gating, or Windows CI expansion
- Embedding examples into the docs site (existing raw-loader flow already does this)
