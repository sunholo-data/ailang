# M-PLATFORM-SPLIT: put the Internal tier behind a boundary the build can see

**Status**: Planned (draft for Mark's decisions; see Design Freeze). D4 decided 2026-11-12 (post-1.0, v1.1); D1–D3 open until v1.1 starts
**Target**: v1.1 (post-1.0), all milestones including M0 (D4, decided by Mark 2026-11-12); see "Pre-1.0 or post-1.0"
**Priority**: P1
**Estimated**: 13–18 engineer-days for M0–M4, plus 2–3 days for the optional M5 (an estimate; see Timeline)
**Dependencies**: [m-v1-public-surface-hygiene](../v1_0_0/m-v1-public-surface-hygiene.md) M1 (a `Tier` field on the command table); dashboard UI removal (handover workstream 4) before M3; Phase 3b preconditions of [m-v1-simplification-program](../m-v1-simplification-program.md) for M4
**Planner-Lane**: opus-required. M3 rewrites about 950 import lines across about 690 files

**Implements:** [HANDOVER-v1-codebase-health-2026-11-09](../HANDOVER-v1-codebase-health-2026-11-09.md) §2a and the
last row of §4 ("Platform split: needs a design doc").
**Builds on:** [m-v1-simplification-program](../m-v1-simplification-program.md) Phase 1 (the closure test) and
Phase 3b (the gated `ailang` / `ailang-ops` split, ruled by D1) ·
[FOLLOWUPS-m-v1-simplification-s5](../FOLLOWUPS-m-v1-simplification-s5.md) F1/F2 ·
[v1-release-readiness-review-2026-11-09](../v1-release-readiness-review-2026-11-09.md) §8–§9 ("After, but say so in 1.0") ·
[m-v1-public-surface-hygiene](../v1_0_0/m-v1-public-surface-hygiene.md) §9 (the stability.md paragraph that declares the split)

**How this was measured (2026-11-12).** Worktree at `origin/dev` = `1e99b5b33`, `go1.26.11 darwin/arm64`,
default build tags. Every number below names the command that produced it. The classification lists (which
package is Internal tier) are written out in full in §11 so the numbers can be re-run, not trusted. No code was
changed for this doc. Two throwaway Go modules in a scratch directory tested the module-boundary rules (§11, V10–V11).

---

## 1. Problem statement

### 1.1 What the handover measured, and what it means by "platform"

Handover §2a splits tracked non-test `.go` under `internal/` by the 13-directory core list in
`.claude/rules/architecture.md` (`parser types eval core elaborate effects builtins lexer ast pipeline runtime
link iface`). Re-run today:

| Split by the 13-dir core list | prod LOC (11-09) | prod LOC (11-12) | test LOC (11-12) |
|---|---|---|---|
| core | 109,849 | 113,923 | 118,975 |
| everything else under `internal/` | 258,075 | 254,941 | 252,403 |
| "platform" share of prod | 70.1% | **69.1%** | |

Command (V1): `git ls-files 'internal/*.go'`, then `wc -l` per file, bucketed by the second path element against
the 13 names, `_test.go` counted separately.

**That 69% is not what would move.** The 13-dir list leaves out language packages that serve Stable and
Experimental commands: `smt` (`check --verify`), `gen/golang` and `vm` (`run --bytecode`, `compile`), `format`
(`fmt`), `repl`, `lsp`, `pkg` (the Stable `pkg` group), `loader`, `prompt`, `testing` and `runner` (`ailang
test`), the AI providers behind `run --caps AI`, and `apiserver` (`serve-api`, which hosts the Stable executor
MCP). None of those is in the D-64 Internal tier. The question this doc answers is narrower: **what does the
D-64 Internal tier weigh, and what holds it to the language?**

### 1.2 The import-graph split

The binary links 154 in-module packages (141 under `internal/`) (V2: `go list -deps ./cmd/ailang`). Classify each one
by the tier of the commands that need it (stability.md, ratified by `D-65`), with the lists in §11:

| Set | What it is | Packages | prod LOC | test LOC |
|---|---|---|---|---|
| **L**: language side | the closure of the 13 language roots (53 `internal/` + `std` + vendored goopus) plus language tooling outside it (AI providers, `apiserver`, `testing`, `runner`, `bytecode/compiler`, `gen/lower`, `modelreg`, `internal/platform/*` backends) | 105 | 248,317 (221,745 under `internal/`) | 236,867 |
| **T**: D-64 Internal tier | coordinator, observatory, executor/\*, eval_harness, eval_analysis/analyzer, mission/\*, server/\*, daemon, browser/\*, rig/credit/approval tooling, microrag, notify (§11 lists all 43) | 43 | **123,143** | 120,620 |
| **X**: message plane | `messaging`, `pubsub`, `storage`, `storage/firestore`, `feedback`. Experimental `messages` and the Experimental `pkg` coordination verbs use it, and it is entangled with T | 5 | **19,603** | 9,109 |
| `cmd/ailang` | one `main` package, 292 non-test files | 1 | 68,710 | 43,397 |
|  …of which Internal-tier command files | 184 files (`coordinator_*`, `eval_*`, `mission_*`, `chains_*`, `serve.go`, `server.go`, …) | | **43,165** | |
|  …of which public + shared files | 108 files (Stable/Experimental commands, the table, `main`) | | 25,567 | |

Commands: V2 for the package set, V3 for LOC per package, V4 for the `cmd/ailang` file split (all in §11).

So the Internal tier is **166,308 prod LOC (T + its command files), 36% of the 459,773 prod LOC linked into
the binary**, or **185,911 (40%) with the message plane**. Its tests add about 120k more. Moving it leaves a
language side of about 274k prod LOC. That side is still larger than the 13-dir core, because it carries the
tooling that users run.

### 1.3 What already holds, and what does not

**Holds:** the language *packages* are a leaf. `TestLanguageCoreIsALeaf` (`internal/diag/closure_test.go`) passes
with an empty `closure_expected_violations.txt` (V5: `go test ./internal/diag/ -run TestLanguageCoreIsALeaf
-count=1` → `ok`), and `scripts/check_boundaries.sh` reports no violations (V5).

**Does not hold:** the language *commands* are not a leaf. The closure test's roots are 13 `internal/` packages.
`cmd/ailang` is a single `main` package, so `go list` cannot tell `ailang run`'s files from `ailang
coordinator`'s. Reading the imports file by file (V4, V6), the public and shared command files reach the Internal
tier directly:

| Public-side file(s) | Imports | For | Serves |
|---|---|---|---|
| `ai_handlers.go`, `ai_prepare.go` | `internal/eval_harness` | `InitModelsConfig`, `ResolveModelName`, `ModelConfig` | `run --ai*` (Stable command, Experimental flags) |
| `prompt_freeze_core.go` | `internal/eval_harness` | `FrozenMarker`, `IsHexSHA256` | `prompt freeze` |
| `run_policy_supervise.go` | `internal/executor` | `PolicyCredentialVars` | `run --policy` |
| `commands.go` | `internal/observatory` | `CheckHealth` (the platform startup probe) | every command's dispatch |
| `pkg_publish.go`, `pkg_msg.go`, `pkg_commands.go` | `internal/messaging`, `internal/pubsub` | post-publish upgrade notices; `affected-by` / `notify-upgrade` / `cascade` | Stable `pkg publish`; Experimental `pkg` verbs |
| 16 `messages_*.go` (4,535 LOC) | `messaging`, `storage`, `storage/firestore`, `coordinator` (`AgentRegistry`, `SuggestInbox`) | the Experimental `messages` command | `messages` |

Language-side **packages** that import T or X (V7, package-level edges over `go list -deps -f '{{.ImportPath}}
{{join .Imports " "}}' ./cmd/ailang`), 4 edges:

| Edge | Why |
|---|---|
| `internal/ai/openai` → `internal/riglock` | `riglock.WithLease` wraps the HTTP client (rig GPU lease) |
| `internal/ai/ollama` → `internal/riglock` | same |
| `internal/docsearch` → `internal/messaging` | `messaging.NewOllamaEmbedder`, `LoadEmbedConfigFromEnv` |
| `internal/apiserver` → `internal/feedback` | `feedback_tool.go` (the hosted `submit_feedback` tool). Through `feedback` → `storage/firestore` → `coordinator`, **this one edge pulls the whole Internal tier into `serve-api`** |

Plus two test-only edges (V8): `internal/proctree` tests → `executor/{claude,codex,pi}`, and `internal/ai/openai` tests
→ `riglock`.

**What that costs, measured:** third-party modules linked (V9, `go list -deps -f '{{.Module.Path}}'`): the language
closure links **32**. The public and shared command files, as wired today, link **77**. The full binary links **91**.
`go.mod` has 98 `require` lines and `go list -m all` resolves 189. The difference between 32 and 77 is
Firestore, Pub/Sub, Cloud Run, Secret Manager, gRPC, the OTel SDK and exporters, sqlite and gorilla/websocket. The
full binary is **107,193,186 bytes** (V9: `go build -o … ./cmd/ailang`, darwin/arm64, no `-s -w`).

### 1.4 Who is affected

- **Every contributor and agent:** `rg`, `grep -r internal/`, gopls workspace symbols and Sonar all treat 166–186k
  lines of fleet tooling as if they were language. Handover §2b's 1,266 unreachable functions are spread across
  both sides.
- **Users:** none today. The Internal tier carries no promise (D-64/D-65), and every Stable and Experimental
  command keeps its spelling (D1).
- **Security triage:** Dependabot and govulncheck alerts on Firestore, gRPC or OTel exporters land on "ailang".
  No Stable command needs those modules once the seams in §5 are cut.

## 2. Goals

**Primary goal:** make the D-64 Internal tier a set the build can name and exclude, so that a language-only
`ailang` builds, is gated in CI, and is what the Phase 3b split ships. One default binary stays in place until
Phase 3b's preconditions hold.

| Metric | Today (V-row) | After M2 | After M3 |
|---|---|---|---|
| Package-level edges L → T/X | 4 (V7) | **0** | 0 |
| Public/shared `cmd/ailang` file imports of T (`eval_harness`, `observatory`, `executor`, `coordinator`) | 8 import lines in 8 files (V6) | **0** | 0 |
| Third-party modules linked by the language-only build | n/a (no such build). Public side as wired: 77 (V9) | measured at M2. **Gate ≤ 45** (closure is 32, plus AI-provider and serve-api deps) | same |
| `internal/` packages in the language-only build | n/a | ≤ 105 (set L, V2) | same |
| Prod LOC under one Internal-tier prefix (`internal/ops/`) | 0 | 0 | **≥ 166,000** (T + command files; ≥ 185,000 if D2 = b) |
| Closure tests | 1 (`TestLanguageCoreIsALeaf`, packages) | 2 (+ `TestLanguageBinaryIsALeaf`, the tagged binary) | 2; deny-list = one prefix |
| Default build's route list: `docs/docs/reference/cli.md`, generated from the dispatch table and byte-compared by `make check-cli-docs` | 88 tiered rows in stability.md (V12) | byte-identical | byte-identical |

## 3. Options weighed

Each option was tested against the six constraints the handover and ledger set: (a) **one binary** at v1.0.0
(D1/F2); (b) the import directions in ARCHITECTURE.md and the closure test; (c) the cloud images built from this
repo; (d) no Go API growth (D-67 decision 9: the Go API is outside the 1.x promise, and
m-v1-public-surface-hygiene §8 says to keep the importable surface small); (e) every Stable and Experimental
spelling keeps working (D1); (f) each step reversible.

### Option A: Go multi-module repo, language as its own module, one binary from both

Two measured facts decide this option:

1. **Internal packages are importable across a nested module boundary.** Go's `internal/` rule is path-based. A nested
   module `…/ailang/platform` can import `…/ailang/internal/parser`, and an unrelated module path cannot (V10:
   scratch modules, `go build` succeeds for the nested module and fails with `use of internal package … not allowed`
   for the outsider). So a nested platform module needs no Go API change.
2. **The one binary's `main` cannot stay in the language module without undoing the split.** If root
   `cmd/ailang` imports the platform module, then `go mod tidy` writes the platform's third-party requirements into
   the **root** `go.mod` as `// indirect` (V11: scratch module cycle with `github.com/google/uuid`). The language
   module's dependency list would be back to the full set. So "one binary built from both" means the full `main`
   lives in the *platform* module, at a new path such as `platform/cmd/ailang`. Then all 22 build sites that name
   `./cmd/ailang` (V13) change, the coordinator buildpack's `GOOGLE_BUILDABLE` changes, and dev builds need a
   `go.work` or `replace`. A `replace` in turn means `go install …@version` refuses the platform module.

**Verdict: not now.** The dependency-list gain is real (91 → ~45 modules visible to the language module). But it
costs a second `go.mod` to keep in step, a `go.work`, a split `go test ./...` in CI, Sonar and CodeQL path
changes, and a moved `main`. The same gain comes almost free once M3 has put the tier in one directory: M5 makes
that directory a module. Keep it as M5, decided on M2's measurements.

### Option B: build tags

A `//go:build !ailang_lang` line on the Internal-tier command files. The default build is unchanged: one binary,
every route. `go build -tags ailang_lang ./cmd/ailang` builds the language-only binary, and `go list -deps -tags
ailang_lang ./cmd/ailang` gives a closure test of the binary rather than of 13 packages. No tag of that kind exists today; the
only build constraints in `cmd/ailang` are OS tags on `mission_activation_*.go`. The one other mention is a
`release` tag in the text of a code-generation template in `compile_types.go`, which is not a constraint on the
binary (V14). The change is reversible (delete the tag lines), touches no import path and changes no deploy.

On its own it does not clean up search results, `go.mod`, or Sonar's view: the files stay where they are.

### Option C: a separate `ailang-ops` binary after v1.0 (Phase 3b)

This is already designed and ruled (D1, program Phase 3b): `ailang` delegates every ops spelling to `ailang-ops`
beside it, and five build paths ship both. It was ruled out for v1.0.0 only because precondition 2, the caller
sweep F1, could not be met in time. Precondition 1 (an empty violations list) **now holds** (V5). Option C is the
*delivery* of a split, not the boundary itself. Without Option B, `cmd/ailang` has no language-only build to put
in the `ailang` binary. Without M1's seams, the language-only build does not link (`run` needs `eval_harness`,
V6).

### Option D: a separate repository

**Rejected.** A repo with a different module path cannot import `…/ailang/internal/…` (V10), and the Internal
tier imports **47 distinct language-side packages** (V15: the union of T/X → L package edges and Internal-tier
command-file imports). Every one of them, including `config`, `ai`, `effects`, `pipeline`, `parser` and `embed`,
would have to leave `internal/` and become Go API. That contradicts D-67(9). It would also end atomic commits
across the boundary, and the eval harness and coordinator are how the language is tested and built day to day.

### Recommendation

**Keep one Go module and one default `ailang` binary. Cut the nine seams in §5, then put the D-64 Internal tier
behind one build tag and one directory (`internal/ops/`), so that `go build -tags ailang_lang ./cmd/ailang`
yields a language-only binary that CI gates.** That turns Phase 3b's two-binary split (Option C, already ruled)
into a build-flag change rather than a refactor. A nested module (Option A) stays an optional last step, taken
only if M2's measurements justify it. A separate repo (Option D) is rejected.

## 4. Pre-1.0 or post-1.0

**Post-1.0 (v1.1). Decided (D4): post-1.0 (v1.1) — Mark, 2026-11-12.** Four reasons, each checkable:

1. **Nothing in the 1.0 promise needs it.** Internal carries no promise (`D-65`; stability.md "Tiers"). The
   readiness review §8 ruled "Platform split: **After, but say so in 1.0**", and m-v1-public-surface-hygiene §9
   drafts the stability.md paragraph that says so. Moving Internal code later breaks nobody, which is the best
   argument against rushing it (review §9).
2. **The 1.0 sequence wants a quiet tree.** `D-66(1)`: the last breaking window, then a soak of at least 7 days, then `v1.0.0-rc.1`,
   then v1.0.0 as that build. M3 moves 1,467 `.go` files and rewrites 951 import lines in 686 files (V16). Doing that
   during the soak would conflict with every in-flight PR and invalidate the soaked build.
3. **G5 reports a settled state.** The one open bar-v3 gate, G5, puts the `make simplicity-metrics` diff against the
   Phase 0 baseline into the v1.0.0 notes (`v1-mission.md`, bar v3). Moving 166k lines during that measurement
   changes what G5 reports.
4. **Phase 3b's own precondition 2 is still open.** The caller sweep F1 has no landing commit (V17: `git log
   --grep="caller sweep"` shows only the 2026-09-18 Phase 3 close-out).

**What may land before the tag:** nothing. The option was M0 (tools and tests, no binary byte changes), and only if
Mark wanted the boundary numbers in the 1.0 notes; D4 chose post-1.0 (v1.1) for every milestone (Mark, 2026-11-12).

## 5. The seams (M1)

Each seam is one small PR and moves a symbol **down** (the closure test's own instruction) or adds a
registration hook (the pattern `registerPlatform` already uses: `cmd/ailang/platform_init.go`).

| # | Edge cut | Mechanism | Control for the test |
|---|---|---|---|
| S1 | `commands.go` → `observatory.CheckHealth` | `probeObservatoryHealth` is already a func var (`commands.go:115`). Set it from an ops init instead of the table file. If the hook is absent, nothing probes. That is correct, not a fallback: only platform commands run the probe (the `Language` contract in `commands.go`), and the language-only build has none | `go list -deps -tags ailang_lang ./cmd/ailang` lacks `observatory`; in the default build, a platform command still probes (existing startup test) |
| S2 | `ai_handlers.go`, `ai_prepare.go` → `eval_harness.InitModelsConfig` / `ResolveModelName` / `ModelConfig` | Move into `internal/modelreg`, which is already a deliberate leaf (ARCHITECTURE.md) | `run --ai` model resolution test passes unchanged; revert the move and the M2 gate fails |
| S3 | `prompt_freeze_core.go` → `eval_harness.FrozenMarker`, `IsHexSHA256` | Move to `internal/prompt` (a language root) | `make check-prompt-freeze` unchanged |
| S4 | `run_policy_supervise.go` → `executor.PolicyCredentialVars` | Move the list down (it names env vars; executor keeps a re-export) | `run --policy` supervision tests unchanged |
| S5 | `ai/openai`, `ai/ollama` → `riglock.WithLease` | **Move down, no registration.** `WithLease` and `LeaseHeader` live in `internal/riglock/lease_transport.go` (58 lines, imports only `net`, `net/http`, `internal/config`; V20). It is the *client* half of the lease: it stamps `X-Rig-Lease` from `AILANG_RIG_LEASE` on loopback requests. It moves to a language-side leaf (for example `internal/riglease`), and the current importers of `LeaseHeader` take the constant from there. The language-only binary therefore stamps the lease exactly as today: no fallback, and no behaviour difference between builds | The existing lease-transport tests move with the file; mutation: drop the header set → they fail. Control: with `AILANG_RIG_LEASE` unset, no header (unchanged) |
| S6 | `docsearch` → `messaging.NewOllamaEmbedder` | Move the embedder and its env config to a leaf (`internal/embedder`); `messaging` imports it | `docs search` neural path test unchanged |
| S7 | `apiserver` → `feedback` | `apiserver.RegisterFeedbackTool(handler)`; ops registers it. A language-only `serve-api` lists no `submit_feedback` tool and **says so on stderr at startup** (no silent drop) | Tool-list golden with and without registration |
| S8 | `pkg_publish.go`, `pkg_msg.go`, `pkg_commands.go` → `messaging`, `pubsub` | Envelope and inbox-address types (`PackageVersionInfo`, `FormatPackageInbox`, `PackageMessageEnvelope`) move to a leaf (`internal/pkgmsg`). Sending goes through a `PublishNotifier` hook that ops registers. A language-only `pkg publish` prints one line that notifications were not sent | Publish test with a recording notifier; with none, the stderr line is asserted |
| S9 | **test-only** edges (V8): `internal/proctree` tests → `executor/{claude,codex,pi}`; `internal/ai/openai` tests → `riglock` | The proctree tests that exercise real executors move to the executor side (for example `internal/executor/proctree_integration_test.go`), and proctree keeps its own fake-child tests. The openai → riglock test edge goes away with S5, because the test then imports the language-side leaf | `go list -test -deps` over every L package reaches no T/X package (M2 gate, below) |

The 16 `messages_*.go` files and the `storage` → `coordinator`/`observatory` edges are **not** seams under the
recommended D2 (b): they move with the platform. Under D2 (a) they would need two more seams,
`coordinator.AgentRegistry` and the storage store constructors. Those are larger, which is one reason for (b).

## 6. Solution design

### 6.1 End state for v1.x (before Phase 3b)

```
repo root = ONE Go module  github.com/sunholo-data/ailang   (unchanged path, unchanged go.mod)
│
├── cmd/ailang/                     language + shared command files (~25.6k LOC)
│     ops_link.go                   //go:build !ailang_lang   → imports internal/ops/cli, registers
│                                     Internal-tier commands + seams S1,S5,S7,S8 (+ messages if D2=b)
│     ops_absent.go                 //go:build ailang_lang    → ops spellings print one line:
│                                     "ailang <cmd>: not in this build (ailang-ops); see …"   (exit 2)
├── internal/<language side>        set L, unchanged paths (parser … lsp, ai/*, apiserver, pkg, testing …)
├── internal/platform/              UNCHANGED: backends the LANGUAGE binary links (otel, sharedmem, streamws,
│                                     streamcred, originpolicy), registered by registerPlatform()
└── internal/ops/                   NEW: the D-64 Internal tier (set T, + set X if D2=b)
      cli/                          the 184 Internal-tier command files, as package `cli`
      coordinator/ observatory/ executor/… eval_harness/ mission/… server/… daemon/ …
```

**Why `internal/ops/` and not `internal/platform/`.** `internal/platform/` already exists and holds the opposite
thing: five backends the *language* binary links through registration seams (V18: `ls internal/platform`;
`cmd/ailang/platform_init.go`). Reusing the name would make the closure deny-list (which names
`internal/platform`) mean two things. `ops` matches the CLI group (`ailang ops …`) and the Phase 3b binary name
(`ailang-ops`), and no `internal/ops` exists (V18).

**The tag follows the tier, not the help group.** `serve-api` and `messages` are filed under the `ops` help group
(`cmd/ailang/commands_platform.go`, `Group: groupOps`) but are Experimental. `serve-api` is language-side (set L,
seam S7), so it stays in-process in the language-only build and is never delegated. `messages` follows D2. Only
Internal-tier routes are tagged out and, after M4, delegated.

**Why the command files move to a package.** Tagging 184 files in place (M2) gives the instrument. Moving them to
`internal/ops/cli` (M3) is what shrinks `cmd/ailang` to about 25.6k LOC and makes `rg --glob '!internal/ops'`
exclude them. That needs the `Command` struct (`cmd/ailang/commands.go`) to live in a leaf both sides import, for
example `internal/cli/command`. It is a small type with no dependencies beyond `internal/pkg.Authority`.

### 6.2 Import directions after the change

| Direction | Allowed? | Enforced by |
|---|---|---|
| language roots → `internal/ops/…` or X | **No** | `TestLanguageCoreIsALeaf`, with PLATFORM_PKGS gaining `internal/ops` |
| `cmd/ailang` built with `-tags ailang_lang` → `internal/ops/…` or X | **No** | **new** `TestLanguageBinaryIsALeaf` (same shrink-only list mechanism, root = `./cmd/ailang` with the tag) |
| `internal/ops/…` → language side | Yes (it is the platform's job to use the language) | by construction. ARCHITECTURE.md's apps → core rule (only via `internal/embed`) stays as written for `server`/`coordinator`/`observatory`/`messaging` |
| `internal/platform/*` | unchanged: linked by the language binary, never by language roots | existing |

`scripts/check_boundaries.sh` keeps its two rules. Its "dashboard" list changes from `internal/{server,coordinator,observatory,messaging}` to `internal/ops/{server,coordinator,observatory}` (+ `messaging` per D2).

### 6.3 Deploy impact (cloud images built from this repo)

`cloudbuild-dev.yaml` and `cloudbuild-release.yaml` build the same image set (V19: `grep -n '^  - id:'
cloudbuild-release.yaml`). The table shows what each runs (`ENTRYPOINT`/`CMD`, V19) and what changes:

| Image | Built by | Runs | Tier | M0–M3 (one binary) | M4 (Phase 3b, two binaries) |
|---|---|---|---|---|---|
| `coordinator` | buildpack, `GOOGLE_BUILDABLE=./cmd/ailang/` (dev l.97, release l.118) | `ailang coordinator …` | Internal | **none**: default build is unchanged | `GOOGLE_BUILDABLE=./cmd/ailang-ops/`, or build both. Program Phase 3b says to decide by what the service invokes |
| `dashboard` | `docker/Dockerfile.dashboard` (`go build ./cmd/ailang/`) | `ailang serve --port 1957` | Internal | none | builds `ailang-ops` |
| `agent-base` and 12 derived `agent-*` images | `docker/Dockerfile.agent-base` | `ailang coordinator execute-job` | Internal | none | ships **both**: the job is ops, and the agents it runs call `ailang run/check` |
| `mcp` (docs MCP) | `docker/Dockerfile.mcp` | `ailang serve-api --mcp-http …` | Experimental | none | S7 registers `submit_feedback` **in-process**, so the tool exists only in a process built with ops. Either the image builds the full default build (`ailang-ops`, which contains every language command, and runs `ailang-ops serve-api`), or it builds the language-only `ailang` and the tool is not offered (D6). Installing `ailang-ops` *beside* a language-only `ailang` does **not** bring the tool back: delegation is per command and `serve-api` is not delegated |
| `executor-mcp` | `docker/Dockerfile.executor-mcp` | `ailang serve-api` + child `ailang run --policy` | hosts **Stable** MCP tools | none | same choice as `mcp` (D6). The child `ailang run --policy` works with either binary |
| `registry-validator` | `docker/Dockerfile.registry-validator` | `registry-validator`, `ailang check/verify` | Stable | none | can switch to the language-only build (a smaller image; optional) |
| `resident` | `docker/resident/Dockerfile`, `FROM agent-pi` | `boot.sh` → `ailang run --policy` | Stable | none | inherits agent-pi |

The release gate's `/health` probes (`cloudbuild-release.yaml` around l.526–530) hit services whose binaries do
not change before M4. **M0–M3 change no image, no Terraform and no `ailang-multivac` config.** M3 changes only
Go import paths. The `-X github.com/sunholo-data/ailang/internal/version.Version` ldflag in four Dockerfiles stays
valid, because `internal/version` is on the language side.

### 6.4 The `ailang serve` dashboard and the coordinator

- **Dashboard (`internal/server`, 5,688 prod LOC, plus `server/auth`):** moves to `internal/ops/server` in M3. Handover
  §3 recommends deleting the UI (`ui/`, the committed `internal/server/dist`, and ≈7.7k lines of UI-only handlers).
  **Do that first** (handover workstream 4). Otherwise M3 moves code that is about to be deleted and doubles the
  conflict surface. The service and its load-bearing routes (`/v1/*` OTLP, hook ingest, `/api/observatory/*`,
  `/benchmarks/`, approvals intake, `/health`) are unaffected: same binary, same routes, only the package path
  changes.
- **Coordinator (`internal/coordinator`, 29,953 prod LOC, the largest Internal package):** moves to
  `internal/ops/coordinator`. It reaches the language only through `internal/embed` and leaf packages (config,
  telemetry, statedir), which the closure tests already constrain. Its `execute-job` entry point is an ops command,
  so after M4 the agent images need `ailang-ops`, and keep `ailang` for the programs agents run.

## 7. Milestones

Each milestone lands on its own, can be reverted on its own, and has a machine-checkable exit. LOC figures are
prod, from V3/V4.

| M | Scope | Est. | Exit criteria (measurable) | Reversal |
|---|---|---|---|---|
| **M0** Instrument | Add `INTERNAL_TIER_PKGS` and `MESSAGE_PLANE_PKGS` lists to `tools/simplicity_metrics.sh` (beside `PLATFORM_PKGS`). New metrics: `internal_tier_loc`, `message_plane_loc`, `lang_to_internal_edges` (package level, V7), `lang_cmd_files_importing_internal` (public/shared `cmd/ailang` files that import a **T** package; file level, V6; classified by `Tier` once public-surface-hygiene M1 lands, else by the §11 prefix list) | 1d | `make simplicity-metrics` prints 123,143 / 19,603 / 4 / 8 (±1% for LOC drift); snapshot banked | delete the lines |
| **M1** Seams S1–S9 | Nine PRs, §5 | 4.5–5.5d | `lang_to_internal_edges` **4 → 0**; test-only L → T/X edges **2 → 0** (V8); `lang_cmd_files_importing_internal` **8 → 0**: S1–S4 cut 5 files, and with D2 = b the 3 `messages_*` files are reclassified as Internal; each PR has its mutation control; `make test` green | revert one PR; each is independent |
| **M2** Build tag | `//go:build !ailang_lang` on the Internal-tier command files (and on `messages_*` if D2 = b); Internal rows of `commands_platform.go`/`commands_eval.go` move into tagged files; `ops_absent.go` for the tagged-out spellings; new `TestLanguageBinaryIsALeaf`; CI job `build-lang-only` that builds with the tag and runs `run`/`check`/`test`/`pkg lock`/`fmt --check` on `examples/` | 2–3d | (1) `make check-cli-docs` passes with `docs/docs/reference/cli.md` unchanged, so the default build's routes are byte-identical; (2) `go list -deps -tags ailang_lang ./cmd/ailang` reaches **no** T package (and no X if D2 = b), **and** `go list -test -deps` over every L package reaches none either. Test imports count, because M5 would make them a module boundary; (3) third-party modules in that build **≤ 45** (from 91); (4) binary size recorded beside 107,193,186 | delete the tag lines and the test |
| **M3** Directory | M3a: `git mv` the 43 T packages (+5 X if D2 = b) under `internal/ops/`, using `gopls`/`gofmt -r`, with `make test-imports` and `make verify-examples` after each batch (coding-standards rule). M3b: move the 184 tagged command files into `internal/ops/cli`, with `Command` in a leaf. Update the 65 non-Go path references (V16), `check_boundaries.sh`, ARCHITECTURE.md (regenerate), `.claude/rules/architecture.md`, `sonar-project.properties` | 3–4d | (1) `INTERNAL_TIER_PKGS` collapses to the single prefix `internal/ops`; (2) ≥ 166,000 prod LOC under `internal/ops/` (≥ 185,000 if D2 = b); (3) `cmd/ailang` ≤ 26,000 prod LOC (from 68,710); (4) both closure tests green and unchanged in meaning; (5) `make check-referenced-paths` green | `git mv` back. It is mechanical, but schedule it in a quiet window with the mission loops paused, because it conflicts with every open PR touching those packages |
| **M4** Phase 3b | `cmd/ailang-ops` = the default build; `cmd/ailang` release builds use `-tags ailang_lang` plus an `ops_absent.go` that **delegates** (exec `ailang-ops` beside itself, then on `PATH`) instead of failing. Five build paths and §6.3's image changes, exactly as the program's Phase 3b states | 3–5d | Phase 3b's five preconditions, unchanged; `binary_internal_packages` for `cmd/ailang` ≤ set L | ship one binary again (drop the tag from the release build) |
| **M5** (optional) Nested module | `internal/ops/go.mod` (`github.com/sunholo-data/ailang/internal/ops`), with `cmd/ailang-ops` inside it; `go.work` for development | 2–3d | root `go.mod` `require` count drops from 98 to ≤ the M2 language-only count + test-only deps; CI runs `go test` in both modules | delete `internal/ops/go.mod` and `go.work` |

**Order and independence.** M0 needs nothing. Each M1 seam needs nothing. M2 needs M1 (otherwise the tagged
build does not link: `run` imports `eval_harness`, V6). M3 needs M2 (the tag tells the move which files go). M4
needs M3 plus Phase 3b's preconditions. M5 needs M3, and is taken only if D5 says yes.

## 8. High-impact decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| D1. Mechanism: one module + directory + build tag (recommended) **vs** a nested module now **vs** a separate repo | Fixes how every later step works; a separate repo would turn 47 language packages into Go API (V15) | human | design | high |
| D2. Where the message plane goes (`messaging`, `pubsub`, `storage`, `storage/firestore`, `feedback`: 19,603 LOC). **(a)** language side: the language binary keeps Firestore/Pub/Sub. **(b)** ops side (recommended): `ailang messages` stays in the default binary, and after M4 it delegates (D1 spelling rule). **(c)** split it further | `messages` is **Experimental** (D-64), not Internal, and it imports `coordinator.AgentRegistry` (V6). The Stable `pkg publish` notifies through it (S8) | human | design | high |
| D3. Directory name: `internal/ops/` (recommended) **vs** a top-level `platform/` | A top-level non-`internal` directory would make 43 packages importable by any module, which is Go API growth (D-67(9)) | human | before M3 | med |
| D4. Timing: all milestones after v1.0.0 (recommended) **vs** M0 before the tag so the numbers appear in the 1.0 notes. **Decided: post-1.0 (v1.1) — Mark, 2026-11-12.** | M0 changes no binary byte, but it adds rows to the metric set G5 reports | human | before M0 | low |
| D5. Whether M5 (nested module) is ever done | Decided on M2's measured module count and on the Dependabot share attributable to ops-only modules | human | after M2 | med |
| D6. Hosted `submit_feedback` after M4: the `mcp`/`executor-mcp` images run the full default build as `ailang-ops serve-api` (the tool stays; the image keeps the platform dependencies), **or** they run the language-only `ailang serve-api` and the tool is dropped. Shipping `ailang-ops` beside a language-only `ailang` is **not** an option, because S7's registration is in-process (§6.3) | The tool is Experimental (D-65 call 3). Dropping it changes the hosted tool list; keeping it keeps Firestore in the hosted Stable-MCP image | human | before M4 | low |

### Design Freeze

Before implementation begins, these must be resolved (unchecked = pause for Mark):

- [ ] D1: mechanism (recommendation: one module, `internal/ops/`, build tag `ailang_lang`)
- [ ] D2: message plane placement (recommendation: (b), with S8 cutting the Stable `pkg publish` dependency)
- [ ] D3: directory name (recommendation: `internal/ops/`)
- [x] D4: post-1.0 for everything, or M0 before the tag (recommendation: post-1.0; M0 is allowed earlier). **Decided: post-1.0 (v1.1) — Mark, 2026-11-12.**

D1–D3 stay open; they are decided when v1.1 starts.

D5 and D6 are not freeze items. They are decided later, on measurements this doc's milestones produce.

## 9. Deferred decisions (agent latitude)

- Tag name (`ailang_lang` vs `langonly`): the agent may choose, but must use one spelling everywhere.
- Name of the `Command` leaf package and of the `cli` package under `internal/ops/`: the agent may choose.
- M3a batch order (by package size or by in-flight-PR conflict): the agent may choose, with `make test-imports`
  run between batches.
- Whether S3's frozen-marker type lives in `internal/prompt` or a new leaf: the agent decides by which keeps the
  closure smaller.
- The wording of the `ops_absent.go` one-liner: the agent may choose, but it must name `ailang-ops` and exit non-zero.

## 10. Non-goals

- **Renaming any CLI spelling.** D1 holds: every route keeps working in the default build.
- **Changing what the Internal tier does.** Dead-code removal (handover workstream 3) and UI removal (workstream 4)
  are separate. UI removal should go *before* M3.
- **Deleting `internal/platform/`** or moving its five backends: they are language-binary dependencies.
- **Language, stdlib, or Go API changes.** `serveapi/protocol` stays where it is (13 world imports, D-67).
- **Phase 3b's own design.** M4 executes it as written in m-v1-simplification-program, with the preconditions
  unchanged.

## 11. Verification log (every number, with its command)

All run at `1e99b5b33` from the repo root, except V10/V11, which ran in scratch modules. `M=github.com/sunholo-data/ailang`.

**Classification lists** (the definition behind every T/X number):

```bash
T='coordinator|observatory|executor|eval_harness|eval_analysis|eval_analyzer|bestof|mission|server|daemon|claudegateway|claudehistory|creditbudget|credentiallease|riggate|riglock|accessapproval|dispatch/cloudrun|microrag|policytool|notify|astedit|browser|websocket|approvaltoken|feedbackgate'
X='messaging|pubsub|storage|feedback'        # anchored: internal/(X)(/|$), so feedbackgate is T
# Internal-tier cmd/ailang files, by name prefix (stability.md CLI tiers); serve_api*.go is NOT Internal:
CMD_I='^(coordinator|eval|mission|chains|observatory|dashboard|trace|server|serve\.go|serve_|ast_|builtins|cache|debug|devtools|disasm|doctor|ext_registry|microrag|pi_|pi\.go|policy|select|access|budget|credit|daemon|design|exec|models|rig|storage|workspaces|browser|plane_label|memory_)'
```

`modelreg`, `apiserver`, `mcpcheck` and `auth/gcp` are deliberately **L**. `modelreg` is a documented leaf that
`run --ai` needs (S2). `apiserver` hosts `serve-api` (S7). `mcpcheck` (`ailang mcp`, Experimental) and `auth/gcp`
(Gemini ADC) import no T/X package.

| # | Claim | Command | Result |
|---|---|---|---|
| V1 | 13-dir split, tracked files | `git ls-files 'internal/*.go'` → `wc -l` per file, bucket by `$2` of the path against the 13 names, `_test.go` separate | core 113,923 / 118,975; rest 254,941 / 252,403; 69.1% |
| V2 | packages in the binary and the closure | `go list -deps ./cmd/ailang \| grep "^$M/"` (154; 141 `internal/`); same over `LANGUAGE_ROOTS` from `tools/simplicity_metrics.sh` (63; 53 `internal/`) | matches ARCHITECTURE.md's generated "53 … full binary links 141" |
| V3 | LOC per package | `go list -f '{{.ImportPath}}\|{{.Dir}}\|{{join .GoFiles " "}} {{join .CgoFiles " "}}\|{{join .TestGoFiles " "}} {{join .XTestGoFiles " "}}' ./...`, then `wc -l` per file | L 105 pkgs 248,317 / 236,867; T 43 pkgs 123,143 / 120,620; X 5 pkgs 19,603 / 9,109; `cmd/ailang` 68,710 / 43,397 |
| V4 | `cmd/ailang` file split | `go/parser` `ImportsOnly` over each non-test file (a 40-line throwaway program), then bucket by `CMD_I` | 184 files / 43,165 LOC Internal; 108 / 25,567 public + shared |
| V5 | the language packages are a leaf | `go test ./internal/diag/ -run TestLanguageCoreIsALeaf -count=1`; `bash scripts/check_boundaries.sh` | `ok`; "no architecture boundary violations" |
| V6 | public cmd files → T/X | V4's per-file imports, filtered to `internal/($T\|$X)` | eval_harness ×3 files, observatory ×1, executor ×1, coordinator ×3 (`messages_*`), messaging ×15, storage ×6, pubsub ×2, storage/firestore ×1. Symbols: `grep -n 'eval_harness\.\|executor\.\|observatory\.' cmd/ailang/{ai_handlers,ai_prepare,prompt_freeze_core,run_policy_supervise,commands}.go` |
| V7 | package-level L → T/X edges | `go list -deps -f '{{.ImportPath}} {{join .Imports " "}}' ./cmd/ailang`, classify both ends | 4 edges (table in §1.3); X → T: 7 (`storage`, `storage/firestore` → coordinator, observatory, creditbudget, credentiallease, feedbackgate); T → L: 116 |
| V8 | test-only L → T/X edges | `go list -f '{{join .TestImports " "}} {{join .XTestImports " "}}' ./<each L pkg>` | `internal/proctree` → executor/{claude,codex,pi}; `internal/ai/openai` → riglock |
| V9 | third-party modules, binary size | `go list -deps -f '{{if .Module}}{{.Module.Path}}{{end}}' <roots> \| sort -u` for the binary / the 13 roots / the public-file imports; `go version -m <bin> \| grep -c dep`; `grep -cE '^\s+[a-z].* v[0-9]' go.mod`; `go list -m all \| wc -l`; `go build -o bin ./cmd/ailang; wc -c` | 91 / 32 / 77; 91 linked; 98 requires; 189; 107,193,186 bytes |
| V10 | nested module may import the parent's `internal/`; a foreign module may not | scratch: module `github.com/example/lang` with `internal/lang`; nested `…/lang/platform` (`replace => ../`) importing it; foreign `github.com/other/thing` importing it; `go build` each | nested OK; foreign: `use of internal package github.com/example/lang/internal/lang not allowed` |
| V11 | root `main` importing a nested platform module pulls the platform's deps into the root `go.mod` | scratch: platform requires `github.com/google/uuid v1.6.0`; root `cmd/ailang` imports platform; `GOFLAGS=-mod=mod GOPROXY=off go mod tidy` in both | root `go.mod` gains `require github.com/google/uuid v1.6.0 // indirect`; one binary builds |
| V12 | default route count | stability.md CLI tables: Stable 19 + Experimental 23 + Internal 46 | 88. M2's check is the generated `cli.md` instead, because `--help --all` does not exist yet: the installed binary answers `flag provided but not defined: -all`. It is proposed in m-v1-public-surface-hygiene M1 |
| V13 | build sites naming `./cmd/ailang` | `grep -rn '\./cmd/ailang\b\|\./cmd/ailang/\|cmd/ailang@' Makefile make .github/workflows docker cloudbuild-*.yaml` | 22: build.yml 2, ci.yml 3, release.yml 1, cloudbuild-dev 1, cloudbuild-release 1, 5 Dockerfiles, make/build.mk 5, make/code-health.mk 4 |
| V14 | no `ailang_lang`-style tag exists | `grep -rn '//go:build' cmd/ailang/*.go \| grep -v _test` | only `mission_activation_{unix,other}.go` (OS). `compile_types.go` matches only inside a template's text |
| V15 | language packages the Internal tier imports | union of V7's T/X → L targets and the in-module imports of the `CMD_I` files, minus T/X | 47 distinct packages |
| V16 | M3's rewrite size | `grep -rhE --include='*.go' '"github.com/sunholo-data/ailang/internal/($T)(/[a-z_/]+)?"' internal cmd tools serveapi scripts \| wc -l` (and `$X`); `find internal/<T\|X> -name '*.go' \| wc -l`; `grep -rlE 'internal/($T\|$X)\b' Makefile make scripts .github docker .claude/skills tools/launchd \| wc -l` | 708 + 243 = 951 import lines in 686 files; 1,467 `.go` files; 65 non-Go files |
| V17 | the caller sweep F1 has not landed | `git log --oneline -i --grep="caller sweep"` | only `9659396bd` (the Phase 3 close-out, 2026-09-18) |
| V18 | `internal/platform` holds language backends; `internal/ops` is free | `ls internal/platform`; `ls -d internal/ops` | `originpolicy otel sharedmem streamcred streamws`; no such directory |
| V20 | the lease transport is a 58-line client-side file with only a `config` dependency | `sed -n 1,23p internal/riglock/lease_transport.go`; `wc -l` | imports `net`, `net/http`, `internal/config`; `RoundTrip` reads `config.RigLease()` per request and stamps loopback hosts only |
| V19 | image set and entrypoints | `grep -n '^  - id:' cloudbuild-release.yaml`; `grep -nE '^(ENTRYPOINT\|CMD)' docker/Dockerfile.{dashboard,mcp,executor-mcp,agent-base,registry-validator}` | as §6.3 |

**Negative-existence claims, each with a row:** no language-only build tag (V14); no `internal/ops` (V18); the
closure test does not cover `cmd/ailang` (read: `LANGUAGE_ROOTS` in `tools/simplicity_metrics.sh` l.47 has no `cmd/`
entry); no `Tier` field on `Command` yet (`grep -n Tier cmd/ailang/commands.go` → empty); no caller-sweep commit
(V17).

## 12. Risks and mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| The tagged build silently drops something a Stable command needs (for example, `pkg publish` notifications) | High: a user-visible regression in the language-only `ailang` after M4 | S7 and S8 print a stderr line when the hook is absent. M2's CI job runs Stable commands on `examples/`, and the downstream gate (`make downstream-check`) runs on the language-only binary before M4 |
| M3 collides with in-flight PRs and mission-loop worktrees | High (stalled PRs) | One quiet window, loops paused (`launchd` HOLD protocol), M3a in batches. `git mv` keeps history (`git log --follow`) |
| A ninth seam appears that this doc missed | Med | M2's gate is the instrument: it fails on any T package reached from the tagged build, whatever this doc listed |
| The classification is wrong for a package (for example, `notify` or `astedit`) | Low | The lists live in one place (`tools/simplicity_metrics.sh`, M0), and moving a package between L and T is one line plus M2's gate |
| Two closure tests drift apart | Low | Both read their lists from `tools/simplicity_metrics.sh`, as the existing test does |
| The import-path change in M3 breaks `make test-imports` (the Sept-2025 class of failure) | Med | `gopls rename`/`gofmt -r` only, `make test-imports` + `make verify-examples` between batches (coding-standards rule) |

## 13. Axiom compliance

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | 0 | no runtime behaviour change |
| A2 Replayability | 0 | traces unchanged |
| A3 Effect legibility | 0 | effects unchanged; S1/S7/S8 are Go-side registration seams, not language effects. S2–S6 move code down and change no behaviour |
| A4 Explicit authority | 0 | capability grants unchanged |
| A5 Bounded verification | 0 | none |
| A6 Safe concurrency | 0 | none |
| A7 Machines first | +1 | agents searching the language see about 274k prod LOC instead of about 460k; one prefix excludes the rest |
| A8 Minimal syntax | 0 | none |
| A9 Cost visibility | 0 | none |
| A10 Composability | +1 | the language binary composes with ops through registration seams instead of direct imports |
| A11 Structured failure | +1 | the hooks S7/S8 and the absent-command path fail with a named line and a non-zero exit instead of a missing feature |
| A12 System boundary | +1 | the boundary between the language and the fleet becomes a build-checked fact |

**Net +4 → proceed.** No −1 on A1/A3/A4/A7.

## 14. Quorum

Trigger 1 fires: this doc has design-freeze items D1–D4. Run on 2026-11-12 with `ailang design-quorum
design_docs/planned/v1_1_0/m-platform-split.md --author claude:claude-opus-5-5 --max-cost-usd 0.30
--controller-verdict pass --controller-note …`. Seated: `gemini-4-argon`, `oc-glm-5-3`, `oc-kimi-k3`, with
reserve `gpt6-1-sol`. `oc-glm-5-3` and `oc-kimi-k3` were **unreachable** in both rounds, so the quorum ran
degraded to 2 reviewers (gemini-4-argon, gpt6-1-sol) plus the controller. Artifacts are in
`.ailang/state/mission-quorum/m-platform-split-*.json` (local, untracked).

| Round | Verdict | Cost | Objections, all accepted and fixed in this text |
|---|---|---|---|
| 1 | **blocked** (exit 3) | $0.094 | (gemini) the two test-only edges (V8) had no plan, and a nested module (M5) would turn them into a module-boundary break → **S9 added**, and the M2 gate now covers `go list -test -deps`. (gpt6) D6's "ship `ailang-ops` beside `ailang`" cannot bring back an in-process registration → **D6 and §6.3 corrected** |
| 2 | **blocked** (exit 3) | $0.095 | (both) S5 as written fell back silently to an unleased HTTP client in the language-only build, which breaks the no-silent-fallback rule → **S5 rewritten as a move-down** of the 58-line client-side lease transport (V20). Both builds now behave identically |

**State: not re-reviewed after the round-2 fix.** The design-doc-creator guardrail is to re-run the quorum once,
not to grind further rounds, so a third round was not spent. Total spent: $0.189. The round-2 objection was
narrow and its fix removes the registration rather than arguing for it. The next reviewer, human or quorum,
should look first at whether any other seam (S1, S7, S8) degrades silently. Each states its absent-hook
behaviour in §5.

## Related documents

- [m-v1-simplification-program](../m-v1-simplification-program.md): Phase 1 closure test, Phase 3b split (D1)
- [FOLLOWUPS-m-v1-simplification-s5](../FOLLOWUPS-m-v1-simplification-s5.md): F1 caller sweep, F2 split → v1.1
- [m-v1-public-surface-hygiene](../v1_0_0/m-v1-public-surface-hygiene.md): M1 `Tier` field; §8 Go API; §9 declaring the split
- [v1-release-readiness-review-2026-11-09](../v1-release-readiness-review-2026-11-09.md): §8 ("After, but say so in 1.0"), §9
- [HANDOVER-v1-codebase-health-2026-11-09](../HANDOVER-v1-codebase-health-2026-11-09.md): §2a, §3 (dashboard), §4
- [ARCHITECTURE.md](../../../ARCHITECTURE.md): layers, enforced directions, generated closure section
- [stability.md](../../../docs/docs/reference/stability.md): CLI tiers (19 / 23 / 46)

---

**Document created**: 2026-11-12
**Last updated**: 2026-11-12
