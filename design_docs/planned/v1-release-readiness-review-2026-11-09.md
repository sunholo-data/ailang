# v1.0.0 release-readiness review (2026-11-09)

**Status:** recommendation for Mark. Nothing in the code was changed by this review.
**Measured at:** `dev` = `51f6636ae` (120 commits past `v0.57.4`).
**Companion:** [HANDOVER-v1-codebase-health-2026-11-09.md](HANDOVER-v1-codebase-health-2026-11-09.md)
(simplicity audit, dead code, Sonar, GitHub alerts, dashboard usage). This document cites its
numbers rather than re-measuring them, and sorts its workstreams into before-tag and after-tag (§8).

## 1. Verdict

**Do not tag v1.0.0 as the next release. Ship the current `dev` as v0.58.0, fix a short list of
things that can only be fixed cheaply before the promise, soak, then cut v1.0.0 from that line
with docs and version changes only.**

Bar v3 ("1.0 = a promise we can keep") is nearly closed, and it measured the right things for
its question: is the promise written, ratified, and free of P0s. It did not ask the other
question a 1.0 raises: **is this the surface we want to be unable to change?** Under
`stability.md`, any change that makes a previously accepted program fail needs 2.0, and there is
no compile-time deprecation mechanism yet. So every item in §3 is cheap this month and a major
version bump afterwards.

What is solid, and should be said plainly in the announcement:

- Effect checking is real and deny-by-default, statically and at run time; no I/O builtin is
  reachable from pure code (405 builtins through one wrapper).
- Output is deterministic (20/20 byte-identical runs; evaluator and VM agree).
- Releases are signed (cosign, SHA256SUMS, SLSA provenance) for five platforms.
- 336 of 346 examples pass, 0 fail; `make test-core` and `go vet ./...` are clean.
- Version handling has no hard-coded "major is 0" assumptions.
- The Stable stdlib is a sensible size: 17 modules, 280 exports, 94% documented.

## 2. How this was produced

Seven independent Opus reviewers, each read-only and given one dimension, asked for a sceptical
read rather than a restatement of bar v3. The controller (Fable 5.2) then reproduced the
highest-severity claims directly.

| Dimension | Reviewer verdict |
|---|---|
| Language core | **NOT READY** |
| Release engineering | **NOT READY** as the next tag; ready after one soaked 0.x |
| Standard library | Ready with conditions |
| Docs and onboarding | Ready with conditions (conditions are blockers, all doc work) |
| Security and trust claims | Ready with conditions |
| Evidence (evals, perf, tests) | Ready with conditions |
| Repo and CLI cleanliness | Ready with conditions |

**Reproduced by the controller** (commands in §11): the cross-module ADT unsoundness; the missing
ADT exhaustiveness check; `show` truncation; `check --verify` exiting 0 without Z3; the dangling
`pre-push` hook; 120 unreleased commits with three BREAKING fragments; no test gate in
`release.yml`; no `-race` anywhere; PR #2195 open; the tracked 29 MB machine dump.

Everything else below is a reviewer finding that the reviewer verified but the controller did
not. Items a reviewer only inferred are marked *(inferred)*.

## 3. Decide before the promise: changes that become 2.0 afterwards

These are the core of the recommendation. Each either rejects programs that pass today, or
changes observable behaviour of a Stable item.

### 3.1 Type soundness (blockers)

| # | Problem | Evidence |
|---|---|---|
| a | **Two ADTs with the same name in different modules are one type.** `type T = A(int)` and `type T = A(string)` unify; the program passes `check` and dies at run time with `_str_upper: arg 0 - expected string, got int`. `LIMITATIONS.md:145` mentions unqualified names but not that it breaks soundness. No open issue. | reproduced |
| b | **Non-exhaustive matches on ADTs, lists and Option pass `check` with no diagnostic.** `inferScrutineeType` only recognises literal patterns (`internal/elaborate/patterns.go:248`). The run-time error carries no location. #2146, currently P2. | reproduced |
| c | A valid program is rejected depending on unrelated code — type-variable collision with an imported polymorphic signature. #2157, P1. | reviewer |
| d | Undeclared type names (#1485), duplicate top-level functions (#1278, last one wins) and ambiguous type imports (#1980) are all accepted silently. | reviewer |

For a language whose pitch is that the checker makes AI-written code trustworthy, (a) and (b)
are the two findings I would not ship 1.0 with. **Recommendation:** fix (a), (b) and (d) in
v0.58.x; make non-exhaustive match an **error**, not a warning, since AI authors do not read
warnings. Raise #2146 to P0 and file (a).

**Also add a soundness clause to `stability.md`.** Today the promise has no carve-out for a fix
that starts rejecting an unsound program, so even after this round the next soundness bug would
formally need 2.0. Rust-style wording works: accepting a program the type system should reject
is a bug, and fixing it is not a breaking change.

### 3.2 `show`

`show` on any container wider than 80 columns returns an elided string:
`show(range(0, 50))` is 43 characters, `[0, 1, 2, 3, 4, 5, 6... 45, 46, 47, 48, 49]`
(`internal/eval/show.go:53`, reproduced). The elision is in the returned value, so anything that
serialises, compares or hashes via `show` is silently wrong. Separately, `show` is not injective
for strings in containers: `show(["a", "b, c"])` prints `[a, b, c]` and `show(Some("x"))` prints
`Some(x)`.

**Recommendation:** make `show` total and unambiguous (no elision, strings quoted inside
containers), and put truncation in an explicitly named display helper. This is a behaviour change
to the most-used function in the language, so it must land before the freeze or never.

### 3.3 Stdlib surface

There is no deprecation mechanism, so renames are now-or-2.0.

- **Naming:** `head_or`, `last_or`, `nth_or`, `charAt_or`, `abs_Int`, `abs_Float`, `rand_int`,
  `rand_float`, `rand_bool`, `rand_seed`, `PI()`, `E()` break the camelCase convention.
- **Argument order:** `list.member(x, xs)` against `list.contains(xs, x)`, `map.member(m, k)`,
  `set.member(s, x)`. The teaching prompt (around line 1295) teaches `member(key, m)` on a map,
  which fails to type-check.
- **Misleading names:** `fs.readFileBytes` returns base64 text; `fs.glob(root, suffix)` is a
  suffix filter.
- **`json.decode` accepts trailing garbage** (`decode("1 2")` is `Ok`), while `decodeFloatArray`
  in the same module rejects it.
- **`JNumber(float)` loses integers above 2^53.** Adding a `JInt` constructor later breaks every
  exhaustive match on `Json` — and §3.1(b) is about to make those matches matter.
- **Tiering is per module.** Niche helpers get frozen with the core: `json.repair`,
  `asBoolLoose`, the `*Strict`/`*OrEmpty` getters, `string.decodeQuotedPrintable`,
  `mapSlicesJoin`, `foldSlices`, the `array` numeric kernels, `list.takeMap`.

**Recommendation:** one stdlib-polish sprint in v0.58.x — rename with the old spelling kept as an
alias for the 0.58 line only, pick one argument order, fix `json.decode`, decide `JNumber`
(add an int constructor now, or document the limit and accept it), and add per-function
Experimental marking so the niche helpers stay movable. The reviewer's full symbol list is in §11.

### 3.4 Smaller semantic decisions

- **Integer overflow wraps silently** (`9223372036854775807 + 1` is negative). Choose trap or
  documented wrap; either is defensible, undocumented is not.
- **Unannotated callback types.** The prompt says `T -> U` is an implicit open row, but
  closed-pure `std/list.map` rejects an `! {IO}` callback. One rule, stated, before freezing —
  together with the `map`/`mapE` split.
- **`check --verify` exits 0 when Z3 is missing, skipped or inconclusive** (reproduced; exit 1
  with Z3 present on the same file). `check` is Stable and this is a silent pass on a
  verification command — it contradicts the project's own "no silent fallbacks" rule.
- **`run --bytecode --verify-contracts` silently skips contracts** (#1956).
- **`::` does not parse inside a `test` body** (#1914), and `test` is Stable.
- **IFC export analysis rejects a valid unit-pattern parameter** (#2156, a regression).

### 3.5 Surfaces that become promises by accident

- **`eval`, `serve` and `iface` are top-level commands** but tiered Internal. Either move them
  under a group or mark them in help.
- **Public Go packages** (`runtime`, `serveapi`, `serveapi/protocol/*`, `std`) become Go-semver
  v1 on the tag. World imports `serveapi` in 14 files. State in `stability.md` that the Go API is
  not covered, or move what can move under `internal/`.
- **Interface digests are non-deterministic** (#2076: six cold checks of `std/sem.ail` gave four
  digests). Whether this reaches the Stable `interface_hash` is unchecked *(inferred)* — it needs
  checking, because a nondeterministic hash in a determinism-first language is an awkward 1.0 bug.

## 4. Release mechanics

### 4.1 Why not tag now

- **Three BREAKING changes have never been released**: #2184 (`pure` implies empty row), #2185
  (cross-module IFC) and the named-test check. v1.0.0 would be their first exposure. 29 unreleased
  fragments in total.
- **No release candidate has been used since v0.3.0-alpha1.** Cadence is 78 tags in 60 days;
  v0.57.1 to v0.57.4 shipped inside about 42 hours; v0.56.0 was tagged and never published.
- **Downstream validation was by hand.** Pins are scattered — docparse v0.57.1, daneel v0.54.18,
  motoko v0.54.14, world v0.41.0/v0.54.15 with v0.47.2 in `go.mod` — and nothing in `Makefile`,
  `scripts/` or `tools/` re-checks them against a candidate.

### 4.2 Gates to fix first

| Gap | Fix |
|---|---|
| `release.yml` publishes GitHub binaries with no test dependency. v0.57.4's tag commit had a failed macOS build and still shipped. | Add `needs:` on test, test-windows and the macOS build. |
| A `v*` tag (rc tags included) redeploys the single prod registry validator with no CI gate (multivac `triggers.tf:208`). Every `ailang publish` gets the new checks the moment a tag lands. | Exclude pre-release tags; gate on the same check-run as core. |
| `.git/hooks/pre-push` on the main checkout is a symlink into a deleted worktree, so the stdlib freeze gate has not run on push since 27 September. `install_hooks.sh` reportedly refuses to replace it. | Repair the hook; make the installer replace a dangling symlink. |
| The stdlib freeze gate proves code matches goldens, not that the promise was kept — re-freezing accepts any change in any tier. Goldens also render one type two ways (`[a]`, `list[a]`), and record callback effect rows as `[]`. | Add a check that diffs **Stable-tier** goldens against the v1.0.0 tag and refuses removals or changed signatures. Normalise the rendering. |
| No `-race` run anywhere. | One CI job. |
| Dev CI about 81% green since 28 October, mostly macOS timing flakes. | Quarantine or fix before the soak, or the soak signal is noise. |

### 4.3 Recommended sequence

1. **Now:** close duplicate PR #2195; repair the pre-push hook; land the gate fixes in §4.2.
2. **v0.58.0:** ship current `dev` — the three BREAKING changes — through the normal pipeline.
3. **v0.58.x (the last breaking window):** §3.1, §3.2, §3.3, §3.4. Each is a breaking fragment.
4. **Build the downstream gate as a script:** check and test docparse, email-parse, daneel, world,
   the motoko fork and every `ailang-packages` manifest against a candidate binary. This is the
   highest-value new instrument in this review; it turns "many downstream applications" from a
   risk into the release test.
5. **Soak at least 7 days** on the final v0.58.x with downstream pins moved and their CI green.
   No new breaking fragments during the soak; one resets the clock.
6. **v1.0.0-rc.1**, once the validator no longer fires on pre-release tags.
7. **v1.0.0** = last v0.58.x plus docs, version strings and the changelog. Then verify:
   `install.sh` resolves 1.0.0, `cosign verify`, validator `/version`, test-env smoke. Promote.
   Rollback is `promote core v0.58.x` plus a manual validator redeploy; the GitHub "latest"
   binary can only be rolled back by re-marking an earlier release.

Rough size, as an estimate rather than a plan: two to three weeks, dominated by §3.1 and the soak.

### 4.4 The dashboard API auth hotfix

The handover's open question 1 interacts with this. A patch tag cut from `dev` is not a v0.57.5 —
it carries 120 commits and three breaking changes. So the hotfix either rides v0.58.0 if that is
cut within a day or two, or goes out as a manual prod rollout of the dashboard image. It should
not wait on the soak. I have not looked at the incident itself; severity is Mark's call.

## 5. Documentation (all doc work, about a week)

**Blockers**

- `reference/language-syntax.md` says it documents **v0.6.0**, and the Stability Promise defines
  Stable syntax by reference to it. It claims record subsumption works (it does not), says row
  polymorphism needs `AILANG_RECORDS_V2=1` (it does not), and shows `parallel`/`spawn`/`loop`
  with no "not implemented" label. Rewrite against the teaching prompt.
- Reference pages document functions that do not exist: `arrayMap`, `arrayFilter`, `arraySlice`,
  `arrayFind`, `arrayAll`, `arrayFold`, `arrayForEach`; `export import`; `std/list (fold)`.
- `SECURITY.md`, `intro.mdx` and `why-ailang.mdx` say "pre-1.0". No 1.x support window is stated
  anywhere.

**Should**

- The second onboarding page (`guides/module_execution.mdx`, "v0.5.11") has 10 of 12 samples
  failing.
- The teaching prompt header says "AILANG v0.16.12" on a v0.57 binary; `prompt --help` says
  about 49 KB when it is 107 KB.
- 13 pages import examples via raw-loader; 66 embed about 1,900 inline blocks. Do not convert
  them all — add a CI job that extracts and checks fenced AILANG blocks, which is the same
  guarantee for a fraction of the work.
- `stability.md` cites internal ledger entries and still lists #1443/#1134 as open P0s.
- Internal guides (coordinator, mission, motoko, model fleet) sit in the public sidebar unlabelled.
- `AGENTS.md`, which the README tells external agents to read, is the fleet's own instruction file.
- Missing entirely: a 0.x to 1.0 migration guide, a "why 1.0" narrative, a 1.0 changelog file, and
  a security-model page (§6).

## 6. What the announcement may and may not claim

| Claim | Verdict |
|---|---|
| Effects are declared in types and checked at compile time | **Safe** |
| A capability not granted with `--caps` cannot be used | **Safe** |
| Net defaults: https only, private and metadata addresses blocked | **Safe** |
| Signed releases with provenance; nothing phones home by default | **Safe** |
| Program output is deterministic | **Safe** |
| Frontier models pass about 98% of core-tier benchmarks | **Safe, with the date** — 368/373, but the frontier rows were last run between v0.34 and v0.52 |
| "AI-written code is safe to run" | **Not without qualification.** `FS` grants the whole user filesystem, `Process` runs any command and escapes the FS sandbox, `Env` exposes every variable. The real sandbox is `--policy`, which is Experimental. |
| "Declaring an effect does not grant it" (`stability.md`) | **Not as written.** `--caps auto` and installed `[bin]` shims grant whatever the program declares, without listing it. |
| IFC as a guarantee | **No.** Implicit flows through branches are untracked; FS round-trips strip labels; #2187 reproduced. Keep Experimental and fix the wording in `why-ailang.mdx`. |
| "Z3 proves it" via `check --verify` | **Not until §3.4 is fixed.** |
| AI writes AILANG as well as Python | **No.** Same model and benchmark: 85.5% vs 91.8% over 363 cells; wider on local models and hard tiers. |
| Cheaper than Python | **No.** About 2x per standard run; about 10x tokens in agent mode. |
| Any per-release trend over v0.54–v0.57 | **No.** Those history points are the same rolling bank re-stamped (803 runs, 715 passes each). |

Two things follow. First, **write a security-model page**: what each capability grants, that
`Process` is host access, and that `--policy` is the confinement story. Second, **run one fresh
paired AILANG/Python baseline on the release candidate** with current frontier models — it is
the only number the announcement can honestly lead with, and it should be quoted with its cost.

Further security findings, including two in `serve-api` and one about tracked data, are held out
of this public file; see §11.

## 7. Cleanliness: what a stranger sees

- **`ailang --help` is 189 lines and mostly advertises Internal machinery** — coordinator,
  eval-suite, credit gateway, browser profiles, a WASM JS API section. Cut it to Stable and
  Experimental. This is the cheapest visible improvement in the review.
- **"89 → 17 commands" describes the help listing.** `stability.md` counts 88 routes and every
  `dev`/`ops` command keeps its top-level spelling. That is fine, but describe it accurately in
  the G5 release note rather than as a reduction of the surface.
- **The repo is mostly data:** `eval_results/` is 17,410 of 30,432 tracked files and 164 MB of
  about 318 MB; 2,671 tracked files contain home-directory paths. Churning state is tracked
  (`.claude/fmt_hook_events.jsonl`, a compile-cache manifest, `examples/manifest.json.bak`).
- **The binary is 110 MB** because it links the fleet: 140 internal packages against 52 for the
  language core, 253 modules.
- `make simplicity-audit` exits 2 on `tracked_files`.

## 8. The handover's workstreams, sorted

| Workstream | When | Why |
|---|---|---|
| S — dashboard API auth hotfix | **Now** | Independent of 1.0; see §4.4. |
| 1 — security alerts | **Before the tag** | Drop the `ollama` Go module for local types (clears the critical with no patch, and 8 govulncheck advisories); bump `zod`; triage the 30 high CodeQL, fixing or dismissing each with a note. Enable secret scanning — owner setting, minutes. |
| 2 — Sonar gate | **Do not gate 1.0 on it** | New-code coverage 78.3 vs 80 is not a user-facing risk. Triage the 7 bugs before the tag; the 203 "vulnerabilities" after. |
| 3 — dead code (1,614 functions) | **After** | Deleting across `types`, `eval` and `core` during the last breaking window adds risk exactly where §3.1 is working. The caveat in the handover is right. |
| 4 — dashboard UI removal | **After**, independently | The usage data is convincing; it is not a language-release matter. |
| 5 — audit gains new metrics | **After** | Add the Stable-golden diff (§4.2) first; it matters more. |
| Platform split | **After, but say so in 1.0** | See §9. |
| Help text, untracking state files, the 29 MB dump | **Before the tag** | Cheap, and it is what a stranger sees. |

## 9. Strategy

**What 1.0 is.** 70% of production Go here is platform, not language (handover §2a). The
tiering in D-64 is the right answer on paper; the binary, the help text, the docs sidebar and the
repo contents do not yet reflect it. 1.0 does not need the platform split done. It needs the
split **declared and visible**: help and docs that show only the promised surface, and a stated
intent that the Internal tier moves behind a module or binary boundary during 1.x. Moving it is
not breaking, because Internal carries no promise — which is the best argument for not rushing it.

**What the downstream applications are for.** They are the strongest evidence the language is
real, and currently they are also the largest unmanaged release risk. The downstream gate in
§4.3 converts one into the other, and it keeps paying after 1.0: it is the instrument that makes
the stability promise checkable rather than asserted.

**What to promise about AI-writability.** The honest position is narrower than the pitch and
still good: frontier models write core AILANG almost perfectly, and the checker catches effect
errors before anything runs. The measured gap to Python is on parsing and text-heavy tasks and on
local models, and timeouts dominate the failures, not compile errors. That reads as a stdlib and
teaching gap *(inferred, not transcript-verified)* — additive work, well suited to 1.x.

## 10. Explicitly not needed for 1.0

- Persistent collections. `Map`/`Set`/`Array` updates are O(n) and `::` accumulation is
  quadratic (#1947); both are documented and the fix is non-breaking. State the cliff in
  `LIMITATIONS.md` along with the roughly 5 KB per frame recursion cost.
- Additive stdlib holes: `list.sort`, int `min`/`max`, `padLeft`, path helpers, `fs.stat`,
  `result.getOrElse`.
- Type-checker fuzzing and property tests.
- Sonar green, dead-code removal, dashboard UI removal, the platform split itself.
- Publisher signatures for the package registry.
- IFC and `--policy` leaving Experimental.
- The 48 pending approvals and 61 open PRs (60 from the bot, several duplicates) — worth a
  cleanup pass, not a gate.

## 11. Evidence and limits

**Reproductions** (binary built from `51f6636`; same result on the installed v0.57.4 build):

```
# 3.1a — passes check, fails at run time
module q/a:    export type T = A(int);  export pure func mk() -> T = A(42)
module q/b:    export type T = A(string)
module q/main: import q/a (mk); import q/b (T, A)
               pure func use(t: T) -> string = match t { A(s) => toUpper(s) }
ailang check q/main.ail   -> No errors found
ailang run ...            -> _str_upper: arg 0 - expected string, got int

# 3.1b
type Color = Red | Green | Blue
pure func name(c: Color) -> string = match c { Red => "r", Green => "g" }
ailang check p1.ail       -> No errors found
ailang run ... name(Blue) -> no pattern matched in match expression

# 3.4
AILANG_Z3_PATH=/nonexistent ailang check --verify c1.ail ; echo $?   -> 0
ailang check --verify c1.ail ; echo $?                               -> 1
```

**Held out of this file.** Three security findings are specific enough that they should not sit
in a public repository before they are fixed. They are in the session scratchpad as
`v1-security-private-notes.md` and belong in the private ops repo beside the dashboard incident
note.

**Not checked by anyone in this review:** whether #2076 reaches `interface_hash`; REPL and WASM
paths; type classes beyond deriving; Z3 soundness depth (#1953, #1954, #1375); Experimental
stdlib modules in depth; the `--policy` supervisor in practice; the registry validator service;
byte-identical output across operating systems beyond the float tests; a full `make test` or
`make ci`; the docs site build; sibling repos' remote state (local refs may be stale).

**Known instrument notes:** `make test-core` took 44–48 s against the documented "<15 s". The
installed `~/go/bin/ailang` is v0.57.4-26, 93 commits behind `dev`.

## 12. Decisions for Mark

1. **Accept the v0.58.x-then-1.0 sequence**, or tag sooner and carry §3 as known limitations?
   My recommendation is the sequence; §3.1 alone justifies it.
2. **Non-exhaustive match: error or warning?** Recommended: error.
3. **`show`:** total and unambiguous, with a separate display helper?
4. **Stdlib renames and `JNumber`:** do the polish sprint, and which way on integers?
5. **Soundness clause** in `stability.md`?
6. **Go packages:** excluded from the promise in writing, or moved?
7. **A fresh paired baseline on the rc** — it costs real money (AILANG standard runs are about
   5x Python); scope and budget are yours.
8. **Bar v4:** add gates for the §3 list, the downstream gate, the release-gate fixes and the
   syntax-reference rewrite, so the dashboard counts down the real remaining work instead of N = 1.
