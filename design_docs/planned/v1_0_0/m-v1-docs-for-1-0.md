# M-V1-DOCS-FOR-1-0: Everything Written That Must Be True on Announcement Day

**Status**: Planned
**Target**: v0.58.x (gate, rewrites, fixes) → v1.0.0 (new documents, version strings)
**Priority**: P0 (the Stability Promise is defined by reference to a page that is wrong)
**Estimated**: 9–12 working days in total, about half attended authoring (an estimate, not a plan)
**Dependencies**: [v1-release-readiness-review-2026-11-09.md](../v1-release-readiness-review-2026-11-09.md) §5, §6; sibling v1_0_0 docs (type identity, exhaustive match, `show`, stdlib polish, `check --verify`) fill the migration-guide placeholders
**Planner-Lane**: opus-required (M2, M3 are attended writing); M1 and M5 are codex-ok

## Problem Statement

The Stability Promise ([stability.md](../../../docs/docs/reference/stability.md):73-76) defines Stable
syntax **by reference to** `ailang prompt` and [language-syntax.md](../../../docs/docs/reference/language-syntax.md).
That page says it documents v0.6.0, and it is wrong about the language. Nothing checks the inline
code on the site, so any fix made by hand drifts back. 1.0 turns every wrong page into a broken
promise. All measurements below were taken on 2026-11-10 with the installed
`ailang v0.57.4-120-gb4e97cdbf` (dev, with the v0.58.0 changelog fold in progress in the main checkout).

**Verified blockers** (repro files are in the session scratchpad, `dd-docs/v/`):

| Page:line | Claim | Reality (command → output) |
|---|---|---|
| language-syntax.md:9 | "documents … AILANG v0.6.0" | binary is v0.57.4+; the page has 15 AILANG blocks fenced as `typescript`, of which 8 fail and 3 more need a `module` line |
| language-syntax.md:260-266 | record subsumption "✅ Works" | `getName({name:"Alice", age:30})` → `record field mismatch: expected 1 fields, got 2 … extra fields: age` |
| language-syntax.md:269-273 | row polymorphism needs `AILANG_RECORDS_V2=1` | `{name: string \| r}` checks and runs with no env var |
| language-syntax.md:375-387 | `loop`, `parallel`, `spawn` shown unlabelled | `parallel {` → `PAR_NO_PREFIX_PARSE … unexpected token in expression: parallel`; `loop {` → `PAR_UNEXPECTED_TOKEN` |
| arrays.md:94-144 (+ :10 "v0.5.6") | `arrayMap`, `arrayFold`, `arrayFilter`, `arraySlice`, … | `undefined variable: arrayMap`; `std/array` exports `get/set/fromList/toList/…` and float kernels, no map/filter/fold at all |
| modules.md:272-274 | `export import std/io (…)` | `PAR_EXPORT_REQUIRES_FUNC: export must be followed by 'func' or 'type', got 'import'` |
| modules.md:315 | `import std/list (… fold)` | `IMP010: symbol 'fold' not exported by 'std/list' — did you mean 'foldl'?` |
| module_execution.mdx:7, :511 | "v0.5.11" guide | 10 AILANG blocks: 2 pass, 3 pass once a `module` line is added, 5 fail (e.g. `_io_println` with no effect row → `Function uses effects not declared in signature`) |
| stability.md:11-17, :58, :262-354 | cites ledger `D-64`/`D-65`, "bar v3", a person's first name | internal governance vocabulary on the public promise page |
| stability.md:93, :102 | #1443 and #1134 listed as open P0s | both CLOSED (fixed by PRs #2184, #2185) |
| stability.md:77 | active prompt "v0.16.12" | true, but reads as a language version on a v0.57 binary |
| `ailang prompt` line 1 | `# AILANG v0.16.12 - AI Teaching Prompt` | binary is v0.57.4; the number is the prompt revision |
| `ailang prompt --help` | `--compact … (~15KB vs ~49KB)` | full prompt is 107,195 bytes; **`--compact` exits 1**: `"v0.16.12-compact" is not a known prompt version` (new finding; `prompt` is a Stable command, so a broken flag on it is a now-or-2.0 item) |
| prompt line 23 | `func f() = { a; b }` errors with `PAR017` | it checks and runs (prints `a`, `b`) |
| prompt line 1296 | `member(key, m)` on a map | `cannot unify Map[α6, α7] with string` (std/map is `member(m, key)`) |
| SECURITY.md:5, intro.mdx:20,:152, why-ailang.mdx:317 | "pre-1.0" | stale on the tag |
| SECURITY.md:27 | personal email as a reporting channel | needs a role address |
| SECURITY.md:54 | scope lists `stdlib/` | the tree is `std/` |
| README.md:193 | `/docs/guides/development` | page is `guides/development-workflow` (dead link) |
| README.md:103 | install pin example `VERSION=v0.33.1` | 24 minors stale |
| README.md:156-165, getting-started.mdx:290, quick-start-examples.mdx:326, repl-commands.md:69 | `:type \x. x + x` prints `∀α. Num α ⇒ α → α` at a `λ>` prompt | REPL prints `\x. x + x :: Int` at a `λ[Debug,IO]>` prompt |
| README.md:222 | "For AI agents: … Read AGENTS.md" | AGENTS.md is this repo's fleet/contributor instruction file (work routing, inbox, sprint gates), not a guide to writing AILANG |
| docs/LIMITATIONS.md:13,:21 | "live-verified at v0.33.1" | 24 minors old; missing cliffs: integer overflow wraps (`9223372036854775807 + 1` → `-9223372036854775808`), `show` elides (`show(range(0,50))` → `[0, 1, 2, 3, 4, 5, 6... 45, 46, 47, 48, 49]`), ≈5 KB/frame non-tail recursion (review §10) |
| CHANGELOG.md:3, :11 | "latest version: v0.32-current.md", row "v0.32.0+" | that file is 16,629 lines holding v0.32.0 → v0.58.0 (92 version headings) |

**Impact:** agents read the reference pages and `llms.txt` (generated from them by
`tools/generate-llms-txt.sh`), so a wrong page becomes wrong generated code. Humans arriving from the
announcement land on the same pages.

## Goals

**Primary goal:** on the day v1.0.0 is announced, every AILANG block on the published site either
passes `ailang check` or is explicitly tagged as a fragment or anti-example, and a CI gate keeps it that way.

**Success metrics** (each machine-checkable, see Milestones):
- Unclassified failing AILANG blocks on site pages: 226 → 0 (baseline file empty at the tag).
- Pages carrying a stale version stamp: 18 → 0 (`grep` in M4).
- "pre-1.0" outside changelogs: 4 → 0.
- Five new documents exist and pass the gate: migration guide, why-1.0, security model, support
  policy (SECURITY.md + stability.md), 1.0 changelog + fixed index.
- Every page documenting Internal-tier surface is either unpublished or carries `tier: internal`.

## Measured baseline: the doc-sample prototype

Prototype extractor (session scratchpad `dd-docs/docblocks.py`, Python stdlib only): walks
`docs/docs/**/*.md(x)`, takes every fence tagged `ailang`, plus every fence tagged `typescript` or
untagged that a heuristic recognises as AILANG (`func`/`module x/`/`import std/`/`! {Eff}`), stages
all `module`-declaring blocks of a page at their module path (so cross-block imports resolve), then
runs `ailang check --json` in three tiers: as written → with `module docblk/bN` prepended → with
imports hoisted and statements wrapped in a function body. Runtime: **13 s wall for all 2,002 blocks
(8 workers)**; the site-only set is 342 blocks.

| Scope | Blocks | Pass as written | +module line | +auto-wrap | Fail |
|---|---:|---:|---:|---:|---:|
| Site pages (188, excluding `prompts/`), fenced `ailang` | 242 | 43 | 38 | 1 | 160 |
| Site pages, AILANG fenced as `typescript`/none (15 pages) | 100 | 17 | 16 | 1 | 66 |
| **Site total** (59 pages with AILANG; 48 with ≥1 fail; 11 all-green) | **342** | **60** | **54** | **2** | **226** |
| `docs/docs/prompts/*.md` (18 archived prompt pages) | 1,660 | 505 | 196 | — | 959 |
| Active prompt (`ailang prompt`, one document) | 98 | 35 | 13 | 4 | 46 |

Failure mix on site pages: 100 parse (`PAR_UNEXPECTED_TOKEN` 42, `PAR_NO_PREFIX_PARSE` 30, `PAR017` 9,
`PAR015` 6), 100 type (85 `undefined variable`, mostly fragments that use names defined elsewhere),
15 module/import, 8 effect, 5 other. Auto-wrapping rescues only 2 blocks, so **fragments must be
classified by a person (or the docs loop), not by heuristics.** Samples: `no-loops.md`'s `PAR015`
blocks are deliberate anti-examples; `ifc-labels.mdx`'s are body-less signatures; `serve-api.md`'s
`@route` blocks (`PAR_UNEXPECTED_TOKEN` ×10) look genuinely stale.

**15 worst pages** (site; "+hdr" includes auto-wrap; "ts" = blocks fenced as non-AILANG):

| Page | Blocks | Pass | +hdr | Fail | ts |
|---|---:|---:|---:|---:|---:|
| guides/serve-api.md | 24 | 4 | 3 | 17 | 0 |
| reference/effects.md | 20 | 0 | 4 | 16 | 18 |
| reference/modules.md | 23 | 6 | 5 | 12 | 22 |
| reference/no-loops.md | 21 | 0 | 9 | 12 | 0 |
| guides/semantic-caching-how-to.mdx | 12 | 0 | 1 | 11 | 0 |
| reference/limitations.md | 11 | 1 | 1 | 9 | 0 |
| reference/arrays.md | 10 | 1 | 0 | 9 | 10 |
| guides/ifc-labels.mdx | 10 | 0 | 1 | 9 | 0 |
| recipes/ai-token-streaming.md | 9 | 0 | 0 | 9 | 0 |
| reference/language-syntax.md | 15 | 4 | 3 | 8 | 15 |
| examples/ai-api-integration.mdx | 14 | 4 | 2 | 8 | 0 |
| guides/semantic-search.md | 10 | 1 | 1 | 8 | 0 |
| reference/std-xml.md | 8 | 0 | 1 | 7 | 0 |
| guides/mcp-connectors.md | 8 | 0 | 2 | 6 | 0 |
| reference/capability-budgets.mdx | 6 | 0 | 0 | 6 | 6 |

The review's "about 1,900 inline blocks" is the whole tree: **1,660 of them are on the 18 archived
prompt pages**. The real site backlog is 342 blocks on 59 pages — about a fifth of the stated size.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| Which document is normative for Stable syntax | The 1.x promise is defined by reference to it | human | design | high |
| Gate inline blocks instead of converting to raw-loader | Reverses docs-mission clause 2 and the docs-sync "never inline" rule | human | design | med |
| 1.x support / patch window | Public commitment in SECURITY.md and stability.md for the life of 1.x | human | before tag | high |
| Internal pages: label in place vs unpublish | Sets what a stranger sees; changes URLs | human | before tag | med |
| How the prompt version is shown next to the language version | Prompt keys are referenced by banked eval rows | human | v0.58.x | med |
| Info-string grammar for block classes | Every docs author and the docs loop writes it from then on | agent (within this doc) | M1 | low |

### Design Freeze

- [ ] Normative syntax source (Decision 2 below)
- [ ] Gate replaces "inline is a defect" in `docs-mission.md` clause 2 and `docs-sync` SKILL.md rule 2
- [ ] Support window wording (Decision 1)
- [ ] Internal-docs mechanism (Decision 3)

## Deferred Decisions

- Gate implementation language (Python stdlib via `uv run --no-project`, or a Go `tools/` program) — agent may choose.
- Exact baseline key format (content hash length, normalisation) — agent may choose.
- Whether `ailang repl` transcripts are replayed in v1 of the gate or only skipped and counted — agent may choose; replay is a 1.x follow-up.
- Prism grammar for `ailang` (alias of an existing grammar or a small custom one) — agent may choose.
- Final prose of the five new documents — attended author, within the outlines below.

## Solution Design

### A. The doc-sample gate (`make check-doc-samples`)

**Info-string grammar.** Docusaurus ignores unknown metastring words, so tags cost nothing in rendering.

| Fence | Meaning | Gate action |
|---|---|---|
| ` ```ailang ` | complete program (default) | `ailang check`; if there is no `module` line, synthesise `module docs/<page-slug>/b<N>` (54 blocks pass only this way; the header changes no semantics) |
| ` ```ailang file=pkg/mod.ail ` | one file of a multi-file example | staged at that path in the page's temp root **before** any block is checked, so siblings can `import pkg/mod`; then checked like a default block |
| ` ```ailang parse ` | fragment that is syntactically whole but uses outside names | must parse: `ailang fmt <file>` exits 0 (measured: exit 0 on undefined names, exit 3 on `parallel {` and on body-less signatures) |
| ` ```ailang sketch ` | pseudo-code, body-less signatures, elisions (`...`) | not run; counted per page and reported. A page with more sketch than checked blocks is flagged in the report |
| ` ```ailang error=PAR015 ` | deliberate anti-example | `ailang check --json` must fail **and** the diagnostics must contain that code. A fixed bug (the block starts passing) fails the gate, which is how PAR017-style stale claims get caught |
| ` ```ailang repl ` | REPL transcript (`λ…>` lines) | skipped and counted in v1; replay via `ailang repl` stdin is a 1.x follow-up |
| AILANG in a `typescript` / untagged fence | mis-fenced | **gate failure** once M1 registers a Prism `ailang` grammar (100 blocks on 15 pages today use `typescript` only for highlighting) |

**Scope.** All of `docs/docs/**` except `docs/docs/prompts/v*.md` (archived prompt revisions are
historical records; docs-15 in the docs-mission queue decides whether they stay published). The
**active prompt** is in scope by checking `ailang prompt` output (equivalently the synced
`prompts/current.md`), with its own baseline section. README.md and `docs/LIMITATIONS.md` are in scope.

**Baseline.** `scripts/doc_samples_baseline.txt`, same contract as
`scripts/context_docs_baseline.txt` / `context_docs_links_baseline.txt`: one line per known-failing
block, `<page> <sha1-12 of the normalised block> <first error code>`. Rules: a block not in the baseline
must pass; a baselined block that now passes **must be removed** (gate fails otherwise); editing a
baselined block changes its hash, so touching a block forces fixing or tagging it. The file may only
shrink; a CI step compares line count with `origin/dev`. Seeded once in M1 with the 226 site
failures and the active prompt's 46.

**Where and how long.** A `ci.yml` step beside `check-doc-review` (ci.yml:509) using the binary CI
already builds, plus `make test-check-doc-samples` (fixture pages: one per tag, plus a mutation
fixture). Expected runtime: under 10 s for 342 + 98 blocks with 8 workers (13 s measured for 2,002
locally); budget 60 s. It does **not** run inside the Docusaurus build.

### B. Normative source for syntax (structural answer)

Options considered:

1. **Prompt normative, reference generated from it.** Rejected: the prompt is pedagogy tuned by eval
   evidence; it contains anti-examples and repetition by design, it is 107 KB, and revisions are
   versioned/frozen for eval reproducibility (`ailang prompt freeze --check`), so correcting a
   normative error would need a new prompt revision and fresh eval evidence.
2. **Reference normative, prompt generated from it.** Rejected: destroys what makes the prompt work.
3. **Recommended: the reference page is normative; both documents draw their examples from one
   checked corpus.** `examples/runnable/syntax/<construct>.ail` (one file per Stable construct in the
   stability.md list), verified by the existing `make verify-examples` (which covers
   `examples/runnable/` only — put the corpus there, or extend `verify-examples-toplevel`). language-syntax.md is
   rewritten as a thin page whose examples are raw-loader imports of that corpus (the "13 pages use
   raw-loader" pattern, used where it matters most), and the prompt cites the same snippets when it
   is next revised. A drift check (`make check-syntax-coverage`) asserts that every Stable construct
   named in stability.md has a corpus file and a reference section. stability.md then defines Stable
   syntax by reference to the **reference page only** and names the prompt as its teaching rendering.

This mirrors existing generated pages (`reference/cli.md` via `make docs-cli`, byte-compared by
`check-cli-docs`; `reference/env-vars.md` via `make docs-env`), which have not drifted.

### C. Prompt version vs language version

Keep the `prompts/versions.json` keys (banked eval rows and frozen hashes reference them). Change
what is **rendered**: `ailang prompt` replaces the file's first line at output time with
`# AILANG Teaching Prompt (prompt revision v0.16.12) — for ailang <binary major.minor>`, so the
frozen file and its hash do not change. `--info` prints both. Fix `--help` to the measured size or
drop the number, and either repair `--compact` or remove the flag in v0.58.x (it is a flag of a
Stable command). The two false prompt claims (PAR017, `member` argument order) go through the
`prompt-manager` skill as the next prompt revision.

### D. New documents — outlines a writer can execute

**D1. `docs/docs/guides/upgrading-to-1-0.md` — "Upgrading from 0.x to 1.0".** Opens with: who must
act (anyone on ≤ v0.57.x), the one-command check (`ailang check` + `ailang test` over the package,
then `ailang lock`), and that 1.0.0 = the last v0.58.x plus docs (no new breaks). One section per
change: *what changed · the diagnostic you will see · the mechanical fix · since version*.
Source the v0.58.0 entries from the "Breaking changes and upgrade notes" block already in the
changelog (one home per fact: link it, do not paraphrase it twice).
1. `pure` means `! {}` (#1443, PR #2184, v0.58.0) — drop `pure` or the row; `iface` purity now derived.
2. IFC contracts cross module boundaries (#1134, PR #2185, v0.58.0) — route through a declared declassifier; limits in #2196.
3. Named tests: a non-final bare expression must be `bool` (v0.58.0) — `let _ignored = …;`.
4. Operators only: `ailang serve` requires auth on non-loopback binds (PR #2197, Internal tier).
5. `<!-- SIBLING:type-identity -->` Same-named ADTs in different modules become distinct types (review §3.1a).
6. `<!-- SIBLING:exhaustive-match -->` Non-exhaustive `match` on ADTs/lists/Option is an error; code TBD (§3.1b, #2146).
7. `<!-- SIBLING:checker-strictness -->` Undeclared type names (#1485), duplicate top-level functions (#1278), ambiguous type imports (#1980) (§3.1d).
8. `<!-- SIBLING:show -->` `show` is total and quotes strings in containers; truncation moves to a named display helper (§3.2).
9. `<!-- SIBLING:stdlib-polish -->` Renames (old spellings are aliases on 0.58.x only), one `member` argument order, `json.decode` trailing input, `JNumber` integer decision (§3.3).
10. `<!-- SIBLING:verify-exit -->` `check --verify` non-zero when Z3 is missing/skipped/inconclusive; `run --bytecode --verify-contracts` (#1956) (§3.4).
11. `<!-- SIBLING:int-overflow -->` Integer overflow: trap or documented wrap (§3.4).
12. `<!-- SIBLING:callback-rows -->` Unannotated callback effect rows and `map`/`mapE` (§3.4).
13. What is **not** covered by the promise: Go packages (`runtime`, `serveapi`, `std`), Internal commands.
Closing checklist for package authors (bump `ailang` constraint, republish, cascade note).

**D2. `docs/docs/why-1-0.mdx` — "Why 1.0 / What's new".** Sections: *What 1.0 means* (a promise,
link stability.md) · *What you can rely on* · *What changed since 0.57* (links D1) · *Honest limits*
· *What comes in 1.x* (platform split declared, review §9; additive stdlib). Draft claims paragraph,
restricted to the review §6 "Safe" rows:

> AILANG 1.0 is a promise about what will not break. From 1.0.0, a program written against the
> Stable surface — the syntax, the 17 Stable standard-library modules, the Stable CLI commands and
> the package format — keeps checking and keeps behaving the same across every 1.x release.
> Effects are part of every function's type and are checked before anything runs; a capability you
> do not grant with `--caps` cannot be used. Network access is https-only, with private and
> cloud-metadata addresses blocked unless you widen it. Program output is deterministic: the same
> program and inputs give byte-identical output, on the interpreter and the bytecode VM alike.
> Releases are signed with Sigstore and carry SLSA provenance, and nothing phones home by default.
> On our core-tier benchmarks, frontier models wrote passing AILANG for 368 of 373 tasks (98.7%),
> on runs made between v0.34 and v0.52 `<!-- DATE RANGE + rc baseline if Mark funds it (review §12.7) -->`.

*Honest limits* paragraph must say: capabilities are coarse (`FS` is your filesystem, `Process` is
your machine, `Env` is every variable); for untrusted code use `--policy` (Experimental) or an OS
sandbox; IFC is Experimental and does not track implicit flows; on the same models and benchmarks
AILANG trails Python (85.5% vs 91.8% over 363 cells) and costs more per run. **Never** write:
"safe to run", "as good as Python", "cheaper than Python", "Z3 proves", any per-release trend over
v0.54–v0.57, or "declaring an effect does not grant it" without the `--caps auto` caveat.

**D3. `docs/docs/reference/security-model.md` — "Security Model"** (documentation level only).
1. *Threat model in one paragraph*: AILANG confines what a **program** can do; it is not an OS sandbox.
2. *Capabilities and what each grants*, one row per `--caps` value from `ailang run --help`
   (gate: a script diffs the table against that list): `IO` console; `FS` every file the user can
   reach unless `AILANG_FS_SANDBOX` or `--policy fs_sandbox` narrows it; `Net` https-only, localhost /
   private / metadata refused unless `--net-allow-*`, 30 s timeout, 5 MB body cap; `Env` every
   environment variable, including secrets kept there; `Process` runs host commands — treat as full
   host access, it is not confined by the FS sandbox; `Clock`, `Rand`; `AI` spends provider credit;
   `Secret`, `Stream`, `Trace`, `SharedMem`, `SharedIndex`, `DOM`, `Msg`, `Cog` one line each with tier.
3. *Granting*: `--caps` lists; **`--caps auto` and installed `[bin]` shims (`caps = auto`, the default)
   grant whatever the entry declares** — read `ailang check`/`iface` output before running code you
   did not write. Fix stability.md:336 to say the same.
4. *Confinement with `--policy`* (Experimental): operator file sets caps, net allowlist, FS sandbox,
   limits; widening flags refused; supervised run; link agent-tool-policy.md. Packages under
   `AILANG_AGENT_POLICY` (summarise SECURITY.md's section, link it).
5. *Information-flow labels* (Experimental): tracks explicit flows through typed values, across modules
   since v0.58.0; does **not** track implicit flows through branches; labels do not survive an FS
   round-trip; inferred parameter rules do not cross modules (#2187). Fix why-ailang.mdx wording to match.
6. *Contracts and Z3*: `requires`/`ensures` semantics; verification is bounded; state the exit-code
   behaviour as fixed by the sibling doc `<!-- SIBLING:verify-exit -->`.
7. *Determinism is not isolation*. 8. *Supply chain*: signatures, provenance, lockfile `content_hash`;
   registry packages are not publisher-signed yet. 9. *Reporting*: link SECURITY.md.

**D4. Support policy** (SECURITY.md "Supported Versions" + a new stability.md section "Support
window"). Replace "pre-1.0" with the table from Decision 1; replace the personal email with a role
address (Mark to create); keep GitHub Security Advisories as preferred; fix `stdlib/` → `std/`.

**D5. Changelog for 1.0.** At the tag: rename `changelogs/v0.32-current.md` →
`changelogs/v0.32-v0.58-<theme>.md` (archive), create `changelogs/v1.0-current.md` with
`## [Unreleased]` and `## [v1.0.0]`; `scripts/changelog_fold.sh` already picks the single `*current*`
file. The v1.0.0 section is short: what 1.0 means, links to D1/D2, and "no code changes since
v0.58.N". Update the hard-coded `v0.32-current.md` in release-manager SKILL.md (lines 26, 32, 254,
265) **and its `.agents/` mirror** (`tools/launchd/sync-agents-skills.sh`). **Now (v0.58.x):** fix the
CHANGELOG.md index so the row reads "v0.32.0–v0.58.x" and the first line names the range; add to
`make check-changelog` that the index's newest row covers the latest tag.

### E. Internal vs public

**Recommendation: hybrid.** (a) Fleet runbooks that document no user surface **leave the published
site** for `docs/internal/` (which already exists, unpublished, and is where CLAUDE.md points
operators), with a stub or client redirect at the old URL. (b) Pages documenting Internal-tier
commands a user *can* run stay, in one collapsed sidebar category **"Internal & contributor"**,
each with frontmatter `tier: internal`; a swizzled `DocItem` wrapper renders an `<InternalBanner/>`
from the frontmatter ("No stability promise; may change in any release"), so the banner cannot be
forgotten and a script can check it. Experimental pages get `tier: experimental` and the same
mechanism. This satisfies docs-mission clause 5 (navigation only shrinks) and review §9 (split
declared and visible). Measured: 36 non-prompt pages match `multivac|motoko|\brig\b` (50 with
`coordinator|mission`); the review's 53 used a different pattern.

Also: README's agent line points external agents to `ailang prompt` and
`/docs/start-here/for-ai-agents`; AGENTS.md gets a first line "for agents contributing to this
repository" and README's "Development" list labels AGENTS.md / CLAUDE.md as contributor files.

## Page-by-page work table

Actions: **rewrite** · **fix** · **gen** (generate/single-source) · **label-I** (Internal) · **label-X**
(Experimental) · **move** (to `docs/internal/`) · **new**. Size S < 1 h, M < half a day, L ≥ half a day.

| Page | Problem | Action | Size |
|---|---|---|---|
| reference/language-syntax.md | v0.6.0; false subsumption/RECORDS_V2; unlabelled `loop`/`parallel`; 8/15 fail | rewrite + gen (§B) | L |
| reference/arrays.md | 7 nonexistent functions; v0.5.6; 9/10 fail | rewrite against `std/array` | M |
| reference/modules.md | `export import`, `std/list (fold)`; 12/23 fail | rewrite | M |
| guides/module_execution.mdx | v0.5.11/v0.2.0; `_io_*` internals; 5 fail, 3 need header | rewrite (or merge into getting-started; clause 5) | M |
| reference/stability.md | ledger IDs, "bar v3", first name; #1443/#1134 "open"; prompt version; `[bin]` grant wording; no support window; no soundness clause (review §3.1) | fix | M |
| `ailang prompt` / prompt-manager | header version; `--help` size; `--compact` broken; PAR017 claim; `member` order; 46 failing blocks | fix (new revision) | M |
| reference/effects.md | 16/20 fail, 18 mis-fenced | fix + tag | M |
| guides/serve-api.md | 17/24 fail (`@route` blocks do not parse) | fix | M |
| reference/no-loops.md | v0.3.14 stamp; 12 fail (mostly anti-examples) | tag `error=` | S |
| reference/limitations.md + docs/LIMITATIONS.md | v0.33.1 stamp; missing overflow / `show` / recursion cliffs; 9 fail | fix | M |
| guides/ifc-labels.mdx, why-ailang.mdx | IFC overclaim; v0.16.0/v0.26.0 stamps; "pre-1.0"; "Safe sandboxing" bullet | fix | M |
| intro.mdx | "pre-1.0" ×2 | fix | S |
| getting-started.mdx, quick-start-examples.mdx, repl-commands.md | REPL `:type` output and prompt shape | fix | S |
| 18 version-stamped pages (ai-effect, ai-routing, custom-ai-providers, debugging, telemetry, three-camps-comparison, model-capability-threshold, ai-api-integration, ai-token-streaming, ai-tool-loop, finetuning/baseline, motoko-extension-development, …) | "AILANG v0.x"/"as of v0.x" narration | fix (delete stamps; clause 5) | S each |
| About 40 further pages with failing blocks (semantic-caching-how-to, semantic-search, std-xml, mcp-connectors, capability-budgets, ai-api-integration, std-extension, extension-packages, go-interop, …) | 1–11 failing blocks each | fix/tag via baseline burn-down | S–M |
| README.md | dead link; v0.33.1 pin; REPL transcript; AGENTS.md pointer | fix | S |
| SECURITY.md | "pre-1.0"; personal email; `stdlib/` | fix + D4 | S |
| AGENTS.md | read as external-agent guide | fix (one-line audience) | S |
| CHANGELOG.md | index range | fix now, D5 at tag | S |
| guides/mission-bootstrap, mission-model-fleet, claude-api-credits, coordinator-workers, notify-daemon, database-architecture, evaluation/local-ollama | fleet runbooks | move (7) | S each |
| guides/coordinator, coordinator-setup, collaboration-hub, workspaces, secret-approvals, agent-workflows, state-system-workflow, notification-channels, development-workflow, telemetry, traces, benchmarking, hooks-setup, evaluation/* (10) | Internal-tier surface, unlabelled | label-I (23) | S each |
| finetuning/* (7) | research log in the product docs | label-I ("research log") | S |
| guides/agent-messaging, cloud-messaging-integration, extension-packages, build-a-motoko-extension, motoko-extension-development, reference/std-extension | Experimental surface; fleet paths in examples | label-X + scrub (6) | S–M |
| guides/upgrading-to-1-0.md, why-1-0.mdx, reference/security-model.md, changelogs/v1.0-current.md | missing | new (D1–D3, D5) | L, M, L, S |

**Totals:** 4 rewrites; 52 pages to fix (48 with failing blocks or stamps or "pre-1.0", net of the
rewrites) plus 5 root files; 37 Internal (7 move, 30 label) and 6 Experimental labels; 4 new pages.

## Milestones (ordered for sprint-planner)

**M1 — Gate first (codex-ok, docs loop can execute; ~1.5 d).** `make check-doc-samples` per §A, the
baseline seeded, Prism `ailang` grammar registered, `ci.yml` step, self-test with fixtures.
*Acceptance:* `make check-doc-samples` exits 0 on dev; adding an untagged failing block to a fixture
exits 1; a baselined block made to pass without removing its entry exits 1; `wc -l
scripts/doc_samples_baseline.txt` ≤ 226 + 46 + header; the CI step finishes in < 60 s; mutation test
per memory rule (revert the gate logic, watch the self-test fail).

**M2 — Blockers (attended author; ~2.5 d).** Section B decision applied; language-syntax, arrays,
modules, module_execution rewritten; stability.md fixed; prompt fixes through prompt-manager.
*Acceptance:* `git grep -n -E "v0\.6\.0|AILANG_RECORDS_V2|arrayMap|arrayFold|export import|std/list \(.*\bfold\b" docs/docs ':!docs/docs/prompts'` is empty;
the four pages have zero baseline entries and zero mis-fenced blocks; `git grep -n -E "D-6[0-9]|bar v3" docs/docs/reference/stability.md` empty;
`ailang prompt | head -1` contains the binary's major.minor; `ailang prompt --compact >/dev/null; echo $?` is 0 or the flag is gone from `--help`;
`make check-syntax-coverage` passes.

**M3 — New documents (attended author; ~3 d).** D1–D5 per the outlines; sidebar entries for D1–D3.
*Acceptance:* files exist and pass the gate with no baseline entries; `git grep -c "SIBLING:"` on D1/D3
is 0 before the v1.0.0 tag (the release-manager checklist runs it); the security page's capability
table matches `ailang run --help`'s `-caps` list (script); `git grep -n "pre-1.0" -- SECURITY.md
README.md docs/docs` empty; SECURITY.md has no personal address (extend the existing
`make check-no-personal-email`, today scoped to the loop-written surface, to cover it);
`make check-changelog` validates the index range.

**M4 — Labelling and stale stamps (docs loop after Decision 3; ~1.5 d).** Banner component,
frontmatter, the 7 moves with redirects, category, README/AGENTS lines, version-stamp removal.
*Acceptance:* every page in the "Internal & contributor" category has `tier: internal` (script); no
page outside it and outside an allowlist matches `multivac|motoko|\brig\b`; the sidebar measure from
docs-22 shows no growth in top-level sections; the stamp grep (this doc's §Problem pattern) returns
0 pages; `make docs-build` green.

**M5 — Baseline burn-down (docs loop, continuous from M1 until the tag).** One page per iteration,
worst first (table above). *Acceptance at the tag:* baseline file has zero site entries; the
release-manager pre-tag checklist refuses a non-empty baseline.

**Loop vs attended.** The docs mission (`design_docs/docs-mission.md`) suits M1, M4 and M5: they are
mechanical, gated, and measurable per iteration. M2's normative rewrite, D1–D3 and Decision-bearing
wording in stability.md and SECURITY.md need an attended author, because they are promises. Note
the loop was last recorded parked on judge transport (STATUS 2026-10-01); confirm it is running
before routing M5 to it, or M5 becomes attended work.

## Risks

- **The gate turns into a tagging exercise.** Authors may mark broken blocks `sketch` to go green.
  Mitigation: the per-page sketch count is reported, and sketch > checked flags the page.
- **Sibling docs change the language under the rewrite.** M2 rewrites against v0.58.x; anything
  §3 changes later must update the corpus files, which `verify-examples` catches.
- **Moving pages breaks inbound links.** The site has no redirect plugin today; add
  `@docusaurus/plugin-client-redirects` or leave stub pages. Only one inbound reference to the seven
  moved pages exists inside the repo.
- **Prompt revision churn.** A new prompt revision shifts the eval baseline; coordinate with the
  rc baseline run (review §12.7) so one revision is measured, not two.
- **Concurrent docs work.** Nine sibling docs and the docs loop touch the same pages; M2 pages
  should be claimed (docs-mission queue entry) before work starts.

## Non-Goals

- Converting inline blocks to raw-loader imports wholesale (only the syntax reference does).
- Deleting the archived prompt pages (docs-15 owns that; the gate just excludes them).
- Replaying REPL transcripts, running samples (`ailang run`) or checking expected output (1.x).
- Docusaurus versioned docs for 0.x vs 1.x.
- The platform split itself; sidebar redesign beyond the one new category (docs-22 owns it).

## Decisions for Mark

1. **Support window.** *Recommended:* the latest 1.x minor receives all fixes; security fixes ship
   as a patch of the latest minor (1.x minors are non-breaking by promise, so upgrading is the
   supported path); v0.58.x receives security fixes for 90 days after 1.0.0. An N−1 window needs
   release branches and backports, which the release pipeline (cut from `dev`) does not have —
   adopt it only with that work funded. Mirror the table in stability.md.
2. **Normative syntax source.** *Recommended:* the reference page, backed by a checked
   `examples/runnable/syntax/` corpus shared with the prompt (§B option 3); stability.md cites the reference only.
3. **Internal docs.** *Recommended:* hybrid — move 7 fleet runbooks to `docs/internal/`, label 30
   Internal-tier pages through frontmatter + banner in one collapsed category (§E).
4. **Inline-block policy.** Amend docs-mission clause 2 and docs-sync rule 2 from "an inline block is
   a defect" to "an inline block must pass the gate or be tagged". *Recommended:* yes.
5. **Prompt header.** Render "prompt revision vX — for ailang Y" at output time, keep registry keys.
   *Recommended:* yes; and fix or remove `--compact` in v0.58.x.

## Open questions

- Is the role reporting address to be on sunholo.com, and who receives it?
- Does `serve-api` (`@route` annotations, Experimental) stay in the public docs for 1.0, given 17 of
  its 24 blocks fail?
- Should finetuning research stay on the product site at all (labelled), or move to a blog?

## Success Criteria

- [ ] `make check-doc-samples` in CI; baseline empty for site pages at the tag
- [ ] Four blocker pages rewritten; Stable syntax defined by a checked reference
- [ ] stability.md free of internal ledger vocabulary and stale P0s, with support window and soundness clause
- [ ] Five new documents published; zero `SIBLING:` placeholders at the tag
- [ ] No "pre-1.0", no stale version stamps, no unlabelled Internal page in the published site
