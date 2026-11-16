# M-V1-PUBLIC-SURFACE-HYGIENE: what a stranger sees, and what becomes a promise by accident

**Status**: Planned
**Target**: v0.58.x for the breaking items (M4, M5); any time before v1.0.0 for the rest
**Priority**: P0 (M4, M5, M7, M8), P1 otherwise
**Estimated**: about 16 engineer-days, or 10–14 calendar days with parallel milestones (an estimate; see §10)
**Dependencies**: none. Coordinates with the other v1.0.0 docs written from the same review
**Planner-Lane**: opus-required. M4/M5 change the capability grant; M9 moves 164 MB of tracked data

**Implements:** [v1 release-readiness review](../v1-release-readiness-review-2026-11-09.md) §3.5, the two
capability rows of §6, §7, §8 (workstream 1 and the before-tag hygiene row) and §9 (stating the split).
**Takes from:** [HANDOVER-v1-codebase-health](../HANDOVER-v1-codebase-health-2026-11-09.md) §2d. Dead code,
dashboard UI removal and the platform split itself are **after 1.0** and out of scope.
**Related:** [m-v1-simplification-program](../m-v1-simplification-program.md) (S5 regroup; D1 "every spelling
keeps working") · [HANDOVER-m-v1-simplification-phase3](../HANDOVER-m-v1-simplification-phase3.md) ·
[m-eval-data-hosting-decouple](../m-eval-data-hosting-decouple.md) ·
[m-builtin-classification-surface](../m-builtin-classification-surface.md) ·
[stability.md](../../../docs/docs/reference/stability.md)

**How this was measured (2026-11-10).** Working tree `dev` at `2ce73c3fa`, plus a scratch binary built from
`51f6636`. This session ran **no git commands**. Tracked status was read from the bytes of `.git/index`
(30,409 entries), and tracked totals come from the 2026-11-09 simplicity snapshot or from the review; each one
is labelled with its source. Every probe used files created in a scratch directory.

## 1. Problem statement

Under `stability.md`, whatever is a promise at v1.0.0 can only change in 2.0. Six surfaces either turn into
promises by accident or contradict what the promise says.

1. **Help.** `ailang --help` is **189 lines**. The 17 commands take 17 of them. The rest is hand-written
   `fmt.Println` text in `cmd/ailang/help.go:242-403`: a WebAssembly JS API (printed twice); coordinator,
   credit-gateway, browser-profile and eval-suite examples; `DEBUG_APPROVAL_WATCHER`, Browserbase variables and
   `ailang doctor memory`. Three Internal commands (`eval`, `serve`, `iface`) are visible at the top level, pinned
   there by `wantVisibleTopLevel` (`commands_groups_test.go:14`). `Command` has no tier field
   (`commands.go:26-70`), so `stability.md`'s CLI tables are hand-kept and nothing checks them.
2. **Grants are not disclosed.** `--caps auto` only prints its grant when `!opts.Quiet`
   (`internal/runner/run.go:277`), but `run` defaults `--quiet` to **true** (`cmd/ailang/main_run.go:31`), so in
   practice it never prints. `[bin]` shims hard-code `run --quiet … --caps auto` (`internal/pkg/bin.go:197`).
   `ailang install` prints the word `auto`, not the capability list (`cmd/ailang/pkg_bin.go:108`). Installing a
   new version re-shims with `auto`, so any capability the new version adds is granted silently.
3. **No pre-flight.** A program that declares `{IO, FS}` and is run with `--caps IO` performs its IO and then
   dies at the first FS call. The partial side effects have already happened.
4. **Go API.** Nine importable library packages sit outside `internal/`. Under Go semver they become v1 on the
   tag, and `stability.md` never mentions Go.
5. **Clutter.** Generated state churns in git. `eval_results/` holds 17,410 of the 30,432 tracked files (review
   figure). `make simplicity-audit` exits 2 on `tracked_files` (30,219 → 30,432, snapshot).
6. **Alerts.** Two Dependabot alerts on `github.com/ollama/ollama`, nine allow-listed govulncheck advisories, a
   high `zod` alert and 30 high CodeQL alerts. Secret scanning is disabled.

## 2. Goals and success metrics

**Primary goal:** at v1.0.0 the first help screen, the repository and the importable Go surface show only what
the promise covers, and every capability grant is visible before it takes effect.

| Metric | Now | Target |
|---|---|---|
| `ailang --help \| wc -l` | 189 | **≤ 50** (draft: 47) |
| Internal command names in top-level help | 3 listed + ~20 in prose | 0, checked by a test that reads the table |
| Stable/Experimental routes whose `--help` is Go's `Usage of x:` | 6 (§4.2) | 0 |
| Capabilities printed: `run --caps auto` / `[bin]` install | never / the word "auto" | always / the resolved list, pinned |
| `go list -deps ./cmd/ailang \| grep -c github.com/ollama` | 7 | 0 |
| High CodeQL alerts open without a fix or note | 30 | 0 |
| `tracked_files` (simplicity audit) | 30,432 | ≤ 13,000, re-banked |

## 3. High-impact decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| A `Tier` field on each dispatch row is the single source; help and a `stability.md` check read it | Help cannot drift from the promise | agent | design | med |
| `[bin]` grants pinned at install; widening needs `--grant-new-caps` | Changes the meaning of `[bin].caps = auto` (Stable package surface) | **human** | v0.58.x | high |
| `run` refuses before execution when the entry declares an effect that was not granted | Rejects runs that work today (§5.3) | **human** | v0.58.x | high |
| `auto` never grants `Process` | Narrows `--caps auto` | **human** | v0.58.x | med |
| Go API declared outside the promise; `runtime/` and `third_party/goopus` move under `internal/` | Go semver binds on the tag | **human** | before tag | low |
| `eval_results/` moves to a bucket with a tracked index; no history rewrite | 164 MB; one CI gate and the MCP image read it | **human** | before tag | med |

**Design freeze** (tick before M4/M5/M9 start):
- [ ] Pinned `[bin]` semantics (§5.2)
- [ ] Pre-flight is an error with no default escape hatch (§5.3)
- [ ] `auto` excludes `Process` (§5.2)
- [ ] `eval_results/` destination (§8.2)

**Deferred (implementer may choose):** help wording and column widths · the pre-flight error code name (a new
`E_*`) · file layout of the local ollama wire types · `.gitignore` grouping · whether `--all` is spelled
`--help --all`, `help --all`, or both.

## 4. Help text and per-command usage

### 4.1 Where help is built

- The command list is generated: `printHelp` (`help.go:232`) → `renderCommandList` (`commands.go:396`) →
  `visibleTopLevel()` (`commands.go:374`), which drops rows where `Hidden || Group != ""`. Group help is
  generated the same way.
- Everything after the list is hand-written: 160 lines in `help.go:242-403`. Only `runner.CapsList` is
  interpolated.
- `docs/docs/reference/cli.md` is generated from the table (`cli_reference.go`, `make check-cli-docs`), but it has
  no tier column (`grep -c 'Stable\|Experimental\|Internal'` → 0). `stability.md:214-315` is hand-written and
  unchecked.
- `trace` is a `Hidden` alias row (`commands_platform.go:143`) routed into `chains`. It works, and it appears
  only in `stability.md`'s "Also accepted" column.

### 4.2 Defects (probed with the scratch binary)

| Probe | Observed | Tier |
|---|---|---|
| `check`, `run`, `verify`, `ai-check` `--help` | Go default `Usage of check:` with single-dash flags; `run --help` is 155 lines | Stable, Stable, Exp, Stable route |
| `export-training`, `replay` `--help` (and Internal `iface`, `eval-suite`) | same | Exp |
| `pkg docs --help` | `Usage: ailang docs <vendor/name>`. **Wrong**: `ailang docs` is the *stdlib* docs command | Stable |
| `pkg add/install/lock/tree/search/publish/bin/unpublish --help` | shows the legacy bare spelling (`ailang install …`) | Stable |
| `version --help` / `sandbox-check --help` | prints the version / runs the check | Stable / Exp |
| `ailang test sq.ail \| grep -c $'\033'` | **10**; still 10 with `NO_COLOR=1`. Colour is on unless `--no-color` (`commands_language.go:236`, `internal/testing/reporter.go:320`) | Stable |
| `ailang --help \| grep -c $'\033'` | 0 (fatih/color already honours isatty and `NO_COLOR`) | — |

The brief named `verify`, `eval-suite` and `ai-check`. **`check` and `run` (both Stable) have the same defect.**

### 4.3 Design

1. **A `Tier` on every route.** `Tier` (`Stable | Experimental | Internal`) goes on `Command` and on the pkg
   subcommand spec. It has no zero-value default: a test fails if any row lacks one. Initial values are copied
   verbatim from `stability.md`.
2. **Top-level help shows Stable and Experimental rows that are filed at top level**, in two sections, followed by
   group footers. Each group's help tags every row `[Experimental]` or `[Internal]`. `ailang --help --all` lists
   every visible row in every group, with its tier.
3. **Re-file, don't rename.** `eval` leaves the visible list; its footer line `ailang eval --help` is how users
   reach it. `serve` moves to `ops` and `iface` to `dev`. `serve-api` (Experimental, user-facing, and how published
   packages deploy) moves to the top level. Every spelling keeps resolving (simplification D1), so **none of this is
   breaking**. Internal routes carry no promise, and moving a route between groups does not change its spelling.
   `wantVisibleTopLevel` becomes the 15 rows in the draft.
4. **Move the prose.** Run flags move to a real `ailang run --help` that keeps today's descriptions, with `--`
   spellings. Environment variables collapse to a link to `env-vars.md`, which is already generated from
   `internal/config.Registry`. The WASM, coordinator, eval, browser and credit text leaves the top-level help.
5. **Per-command usage.** Each Stable or Experimental route gets an `fs.Usage` that prints
   `Usage: ailang <canonical spelling> …` with double-dash flags. The pkg verbs print `ailang pkg <verb>`, and
   `pkg docs` is fixed. `version` and `sandbox-check` answer `--help`.
6. **Colour.** The test reporter colours only when stdout is a terminal, `NO_COLOR` is unset and `--no-color` was
   not given. `cmd/ailang` shares the same helper.
7. **`stability.md` is checked, not generated.** Its "Why Experimental" prose does not belong in Go.
   `TestStabilityDocMatchesTable` parses the first column of the three CLI tables and asserts, tier by tier, that
   the set matches the dispatch table. It runs under `make check-cli-docs`. `cli.md` gains a generated Tier column.

### 4.4 Proposed `ailang --help` (47 lines)

```text
AILANG - The AI-First Programming Language

Usage:
  ailang <command> [arguments]
  ailang help <command>               Full usage for one command

Commands:
  check (ai-check)    Type-check a file or directory (--verify adds contract verification)
  docs                Show stdlib module documentation (--list to enumerate)
  help                Show help for ailang, a group or a command
  init                Scaffold a new AILANG package or web app
  pkg                 Packages: add, install, lock, tree, search, publish (ailang pkg --help)
  prompt              Print the AILANG teaching prompt (for AI code generation)
  repl                Start the interactive REPL
  run                 Run an AILANG program (ailang run --help for all flags)
  test                Run inline and named tests (--package for package mode)
  version             Show version, commit hash and build time

Experimental (may change in a 1.x minor release, with a changelog entry):
  examples            Search and explore working code examples
  fmt                 Format AILANG source (stdout / --write / --check)
  lsp                 Language Server Protocol server (for AI agents and IDEs)
  serve-api           Serve a module's exports as REST and MCP endpoints
  verify              Verify contracts and proof obligations

Capabilities:
  A program gets no capability unless you grant it:  ailang run --caps IO,FS app.ail
  Known: IO,FS,Net,Env,Process,Clock,AI,Stream,SharedMem,SharedIndex,Secret,Trace,DOM,Msg,Cog
  --caps auto grants the entry function's declared effects (never Process) and prints them.
  Process runs programs on this machine; narrow it with --process-allowlist.

Examples:
  ailang run --caps IO hello.ail         Run a program with the IO capability
  ailang check src/                      Type-check without running
  ailang test src/                       Run tests
  ailang pkg install vendor/name         Install a package and its [bin] commands
  ailang prompt > ailang-prompt.md       Teach a model the language

More:
  ailang dev --help    Language-development and diagnostic commands (mostly Internal)
  ailang ops --help    Platform operations: messages, coordinator, missions (mostly Internal)
  ailang eval --help   Benchmark harness (Internal)
  ailang --help --all  Every command, with its stability tier
  Reference: https://ailang.sunholo.com/docs/reference/cli
  Stability: https://ailang.sunholo.com/docs/reference/stability

Flags go before the file name: ailang run --caps IO file.ail (not: ailang run file.ail --caps IO)
```

The capability lines assume §5 lands. If D3 is declined, drop "(never Process)".

### 4.5 Work table

| Item | Evidence | Action | Risk | Size |
|---|---|---|---|---|
| Tier field and tests | `commands.go:26` | add the field; copy values from stability.md | a row gets the wrong tier | S |
| Top-level help | `help.go:242-403` | generate it from the table plus three static blocks | skills that grep the old help text | S |
| Re-file `eval`/`serve`/`iface`; show `serve-api` | `commands_groups_test.go:14` | change `Group` only | `commands_top_level` goes 17→15 (an improvement) | S |
| Seven Go-default usages | §4.2 | an `fs.Usage` for each | `run`'s flag list is long | M |
| pkg spellings; `pkg docs` | §4.2 | fix the strings | — | S |
| `test` colour | `reporter.go:328` | isatty + `NO_COLOR` | CI logs lose colour, as intended | S |
| stability.md check; cli.md tier column | `stability.md:214-315` | a Go test | prose edits trip it, as intended | S |

## 5. Capability grant disclosure and pre-flight

### 5.1 What happens today

The probe `capsprobe.ail` declares `export func main() -> () ! {IO, FS}`, prints `step 1`, then reads a scratch
file it created.

| Invocation | stdout | stderr | rc |
|---|---|---|---|
| `run --caps auto` | step 1, step 2 | *(nothing)* | 0 |
| `run --verbose --caps auto` | step 1, step 2 | `auto-granted capabilities: FS,IO` | 0 |
| `run --caps IO` | **step 1** | `effect 'FS' requires capability … Hint: Run with --caps FS` | 1 |

A second probe, `branchy.ail`, declares `{IO, FS}` but takes its non-FS branch. Under `--caps IO` it exits 0.

`ResolveAutoCaps` (`internal/runner/caps.go:15`) returns the labels of the entry's declared row. The checker has
already forced every callee effect into that row, so `auto` grants everything the program could do. `[bin]`
`caps` defaults to `"auto"` (`internal/pkg/manifest.go:98`), and the shim re-resolves it on every invocation. For
`ailang install --path <dir>` (`pkg_bin.go:41`) the program root is the live checkout, so editing the source
changes the grant on the next run.

**Correction to the review:** `auto` does list its grant, but only under `--verbose`, because `--quiet` became
the default. The review's conclusion holds. **Downstream evidence:** both real `[bin]` users already pin by hand.
`ailang-parse/ailang.toml` uses `caps = "IO,FS,Env"` and `"IO,FS,Env,Process"`, and
`email-parse/packages/eparse/ailang.toml` says *"Caps are pinned rather than auto so a future import cannot widen
them silently."*

### 5.2 Design: the minimum honest behaviour for 1.0

1. **`run --caps auto` always prints its grant** to stderr (`ailang: granted (auto): FS,IO`), unless `--quiet`
   was passed **explicitly**. `fs.Visit` can tell an explicit `--quiet` from the default. Stdout is unchanged.
2. **`auto` never grants `Process`.** If the entry's row contains it, `auto` refuses with `auto will not grant
   Process (it can run any program); pass it in --caps explicitly`. `Secret` stays in `auto`, because it is
   already gated per use in cloud mode (`caps.go:68`).
3. **`[bin]` grants are pinned at install.** Install type-checks the bin's entry module (today it only does the
   text scan in `VerifyBinEntrypoints`) and resolves `auto` to an explicit list. The shim gets `--caps FS,IO` and
   a header `# caps: FS,IO (resolved from auto at install)`, and the install output names the list.
4. **Widening needs consent.** When the replacement shim has the same name, its list must be a subset of the old
   `# caps:` line. Otherwise install refuses and names what was added (`sunholo/x@2.0 adds Net; re-run with
   --grant-new-caps`). There is no interactive prompt, because agents install packages too. `--path` installs
   are pinned the same way. If the source later gains an effect, the run hits the pre-flight check (§5.3)
   instead of gaining the capability silently.
5. **Documentation.** Replace the run-time-grant row in `stability.md` with: "`[bin].caps` defaults to `auto`,
   which is resolved to the entry's declared effects once, at install, then printed and recorded in the shim. A
   later version that declares more is refused until it is re-granted." Update `guides/packages.md:180,191` to
   match. After this change, "Declaring an effect does not grant it" is true as written.

### 5.3 Pre-flight check

The entry's row is known statically. Before `runSingle`, `runBatch` or the policy worker execute anything,
compute `declared − granted − {Debug}`. If the result is non-empty, exit 1 with no side effects:
`E_CAPS_MISSING: main declares {IO, FS}; granted {IO}; missing FS. Re-run with --caps IO,FS`. An open row
variable counts only its concrete labels. Per-route pre-flight for `serve-api` is follow-up work.

**What breaks:** any run that grants less than the entry declares and relies on never reaching the missing
effect, as `branchy.ail` does. Known fixed-caps harnesses include `tools/verify_examples.sh:65`
(`IO,FS,Clock,Rand`), `make/test.mk:223,225,407` (`IO`) and the eval runners. In `examples/runnable/`, 37 of the
231 mains declare at least one of Net, Env, Process, AI, Secret or Stream. `scripts/verify_examples.go:381`
infers caps per file. M5 therefore starts with a census: run `make verify-examples` and the stdlib tests with the
check on, and record every refusal in the PR. **No default escape hatch.** An `--allow-undeclared-reach` flag
would bring back exactly the partial-run problem. If the census finds legitimate cases, the choice goes to Mark
(D2).

### 5.4 Work table

| Item | Evidence | Action | Breaking? | Size |
|---|---|---|---|---|
| Print the auto grant | `main_run.go:31`, `run.go:277` | check for an explicit `--quiet` | no (stderr only) | S |
| `auto` excludes Process | `caps.go:15` | refuse with a message | **yes**: v0.58.x | S |
| Pinned `[bin]` grants and widening refusal | `bin.go:197`, `pkg_bin.go:108` | resolve at install, write the header, compare | **yes**: v0.58.x | M |
| Pre-flight refusal | the two probes | check before execution | **yes**: v0.58.x | M |
| stability.md and packages guide wording | §5.1 | doc edit | no | S |

## 6. Go API

`go list -f '{{.ImportPath}} {{.Name}}' ./... | grep -v /internal/` returns 47 packages:
- 35 are `package main` and cannot be imported. One of them is a stray `cmd/zz_realtask_runner_tmp`.
- 2 are test-only (`tests/golden/*`).
- 1, `ui/node_modules/.../flatted`, shows up only because an ignored local `node_modules` exists.

That leaves **nine importable libraries**. Their importers were found with a read-only grep of sibling checkouts
under `~/dev` and `~/dev/sunholo-data`. Of every sibling `go.mod`, only **ailang-world** requires this module, and
it pins **v0.47.2**.

| Package | In-repo importers (non-test) | External importers | Recommendation |
|---|---|---|---|
| `runtime` (4 files) | **none**: one test, `internal/repl/cons_expression_test.go`; `internal/gen` does not emit it | none | **Move under `internal/` or delete** along with that test (emit-go is frozen) |
| `serveapi` | none (its own external test) | world: **tests only**. World's admission test **refuses** this facade (`host/daemon/daemon_test.go:972-980`) | Keep; add notice |
| `serveapi/protocol` | `internal/apiserver` (6 files) | world: **4 non-test files** (`host/daemon/daemon.go`, `host/projection/{mcp,mcpname,projection}.go`); 13 including tests | Keep; add notice |
| `serveapi/protocol/hostcall`, `…/mcphttp` | serveapi | world: `host/projection/mcp.go` | Keep; add notice |
| `std`, `examples` (one `embed.go` each) | 6 / 1 | none | Keep, because `go:embed` must sit beside the files; add notice |
| `third_party/goopus/{opus,oggopus}` | `internal/builtins/audio_ogg.go` | none | **Move to `internal/third_party/goopus/`**: a vendored fork must not become our v1 API |

**Contradiction with the review:** world does not import `serveapi` in 14 files. It imports
**`serveapi/protocol`** (plus `hostcall` and `mcphttp`) in 13 files, 4 of them non-test, and deliberately gates
against the `serveapi` facade.

**Statement for `stability.md`** (a new subsection under Tiers), with a matching sentence in a `doc.go` for each
package kept:

> **Go packages.** AILANG is a Go module so the binary can be built with `go install`. Its importable Go packages
> (`serveapi/...`, `std`, `examples`) are **Internal**. They are not covered by the 1.x promise and may change in
> any release, whatever Go's semantic-import-versioning convention suggests for a v1 tag. If you import them, pin
> an exact version.

**Module path semantics.** Going from v0.x to v1.0.0 needs no path change. A **v2.0.0 tag requires the module path
`github.com/sunholo-data/ailang/v2`**: `go.mod` changes, about 4,700 in-repo import lines are rewritten, and
`go install …/cmd/ailang@v2…` uses the new path. Without that change, the go command refuses a v2 tag on a module
that has a `go.mod`. A *language* 2.0 therefore forces an import-path migration on every Go consumer, which is the
best reason to keep the importable surface small now.

| Item | Action | Breaking? | Size |
|---|---|---|---|
| `runtime/`: move or delete | fix one test import | no (no external importers) | S |
| `third_party/goopus` → `internal/third_party/` | move; keep LICENSE and THIRD_PARTY | no | S |
| Notices and stability.md paragraph | text, plus a grep test | no | S |
| `cmd/zz_realtask_runner_tmp` | owner check; delete if it is a leftover | no | S |

## 7. Security alerts before the tag

### 7.1 Replace `github.com/ollama/ollama`

The module is used in five non-test files (`internal/ai/ollama/{client,step,tokens}.go`,
`internal/builtins/ollama_embed.go`, `internal/messaging/embedder.go`) and three tests. The test file `host_test.go`
uses `envconfig.Host`. The surface it uses:
- `Client`, `NewClient`, `ClientFromEnvironment`.
- `Generate` and `Chat`, which stream NDJSON callbacks, plus `Embed`.
- `GenerateRequest`/`Response`, `ChatRequest`/`Response`, `Message`, `ImageData`, `Tool(s)`, `ToolFunction`,
  `ToolCall`, `ToolCallFunction` (with `.String()` on Arguments), `Metrics`, `EmbedRequest`.

Seven ollama packages are linked: `api`, `auth`, `envconfig`, `format`, `internal/orderedmap`, `types/model` and
`version`.

**Replacement:** a new `internal/ai/ollama/wire` package. Sizing:
- about 250 lines of types and 150 lines of client (POST, an NDJSON reader with a large line buffer, status-error
  decoding);
- about 200 lines of tests.

Tool parameters and arguments become `json.RawMessage`, which keeps key order. `parseHost` already replicates
`envconfig.Host` (`host.go:19`). The rig lease (`riglock.WithLease`) stays the injected `http.Client`.

**One behaviour is dropped:** the SDK signs requests with the device key when the host is `ollama.com` or
`OLLAMA_AUTH` is set (`api/client.go:119`). Nothing uses that path: AILANG reaches Ollama Cloud through the local
daemon proxy (`internal/ai/factory/factory.go:254`, `models.yml:2976`). The new client refuses a direct
`ollama.com` host with a clear error.

**Size: M, 1.5–2 days.** Parity gate: an `httptest` server captures the SDK's request bytes for generate,
chat-with-tools, chat-with-images and embed before the swap, and the test asserts the new client sends the same
bytes. After the swap, delete the nine Ollama entries in `.govulncheck-allow.yml`.

**Contradiction with the review:** the Dependabot ranges (≤ 0.13.5 for the critical, ≤ 0.20.2 for the low) do not
cover the pinned **v0.40.15**, so both alerts look stale: they are server advisories matched by module name.
govulncheck does still report nine unfixed findings. Removing the module clears both.

### 7.2 `zod`

`tools/pi-extensions/sandbox` uses `"zod": "^3.24.1"`, locked at 3.25.76. The fix, ≥ 4.6.6, is a **major** bump
(the v3 → v4 API), so check `index.test.ts` and `mission.test.ts`. Do the bump off the rig, not with `npm i` there:
the pi pin lives in four places. **S–M.**

### 7.3 The 30 high CodeQL alerts

Pulled with `gh api …/code-scanning/alerts`, GET only, last analysed at `b23cadced75c`.
- **CodeQL:** 266 open (30 high, 236 medium).
- **Scorecard:** 134 open, 5 of them high: token permissions in `dependabot-automerge.yml` and `ci.yml`,
  vulnerabilities, code review, and branch protection.

| Rule (high) | Location | Verdict |
|---|---|---|
| go/zipslip (3) | `internal/pkg/tarball.go:148`, `internal/builtins/tar.go:442`, `cmd/ailang/examples.go:459` | **FP**, containment verified: `isTarPathTraversal` + `filepath.Rel`; the FS-sandbox root; `HasPrefix(Clean(dest)+sep)`. Optionally make `examples.go` fail loudly when it skips an entry |
| go/path-injection (12) | `internal/pkg/{tarball,loader,hasher,lockfile,manifest}.go`, `internal/loader/snapshot.go` | **FP**: the input comes from the local user, the lockfile or the CLI, or is contained. One hardening: reject `..` in the module part of a package import (`loader.go:124`) |
| go/path-injection (5) | one Internal-tier HTTP read path (one package) | **Likely real**, behind auth. Fix: accept only identifier-shaped IDs, check membership, verify with `Rel`. Details are in the private ops notes |
| go/clear-text-logging (4) | `internal/apiserver/{routes_dispatch,auth}.go`, `cmd/ailang/eval_parallel.go` | **FP**: these log env-var *names*, not values. Hardening: redact `Authorization`/`Cookie` from the header map passed to handlers |
| go/weak-sensitive-data-hashing (2) | `cmd/registry-validator/auth.go`, `internal/mission/ollama_quota.go` | **FP**: SHA-256 used as a lookup ID for random tokens |
| go/incorrect-integer-conversion (3) | `internal/parser/parser_literals.go` (2), `builtins/string_ops_parse.go` | **FP**: intentional two's complement (#1481); 64-bit targets only |
| go/allocation-size-overflow (1) | `internal/ai/openrouter/step.go:193` | **FP**: a sum of in-memory slice lengths |

**Result: 5 likely real (one cluster) and 25 false positives.**

**Policy:** every high alert is either fixed or dismissed with a note. A fix is a PR that names the alert. A
dismissal uses `false positive` or `won't fix`, plus a one-line note naming the containment (file:line) or the
trust boundary. The 236 medium `go/log-injection` alerts come after the tag, under the same policy.

**Owner actions:**
- Enable secret scanning and push protection. The API reports `secret_scanning: disabled`, and the alerts endpoint
  returns 404.
- **One tracked eval-result file is flagged for owner review; details are in the private ops notes.** That review
  must happen before §8 uploads anything.

| Item | Action | Size |
|---|---|---|
| ollama → local wire types | §7.1 | M |
| zod major bump | §7.2 | S–M |
| HTTP read-path containment (5 alerts) | fix | S |
| 25 dismissals with notes; 2 hardenings | triage | S |
| Secret scanning; review of the flagged file | owner | — |

## 8. Tracked clutter

Method note: use `/usr/bin/grep` for these sweeps. The interactive `grep` here is ugrep, which skips ignored
files: it reported 841 files containing `/Users/` where the true count is 6,550.

### 8.1 Small items

| Path | Writer | Reader | Action |
|---|---|---|---|
| `.claude/fmt_hook_events.jsonl` (109 lines, 97 contain `/Users/`) | `scripts/hooks/format_ail.sh:46-63`, on every attended `.ail` edit | only the per-workspace copies (`internal/eval_harness/fmt_hook_mode.go:125`) | **untrack + ignore** |
| `tests/codegen-harness/.ailang/cache/compile/manifest.json` | `internal/pipeline/cache_store.go:71,114` (`make test-codegen`) | the compiler cache | **untrack**; already ignored by `.gitignore:82` |
| `examples/manifest.json.bak`, `examples/manifest.tmp` (0 B) | no current writer | none (`make/examples.mk:24` uses `.manifest_rc.tmp`) | **untrack, delete and ignore** |
| `tools/finetune/pilot2.yaml` | hand-written **config**; it churns because of a documented `cp pilot2b.yaml pilot2.yaml` | `pilot2_run.sh:47,54` | **keep** |
| `dist/error_codes.json` | `make error-codes` | `release.yml:239-262` regenerates it and uploads it as an asset; nothing reads the tracked copy | **untrack + ignore `/dist/`** |
| `lane-reports/` (3 files) | cloud `ailang-only-executor` tasks | none | keep (does not churn) |
| `docs/static/benchmarks/*` (3.4 MB) | `eval-report --format=json` (`eval_tools.go:454`), dashboard publish, rig rotation | `benchmarkFetch.js` **falls back to static**; `AxiomScorecard.jsx:12` and `axioms.go:121` read static only; **`export_json.go:17-20` reads `latest.json` to preserve history** | **keep before 1.0**; removal belongs to m-eval-data-hosting-decouple |
| `.ailang/state/{sprints,simplicity,perf,decisions}` (318) | banked deliberately: re-admitted at `.gitignore:102-119`; read by `agent_registry.go:652`, `audit.sh:27`, `sweep.sh:29` | — | **keep** |
| `.ailang/state/evaluations/` (51), `implementation/` (1) | sprint-evaluator (`agent_registry.go:658` declares them as artifacts) | — | **a bug, not clutter**: ignored yet tracked, so new reports cannot be committed. **Re-admit** them |

### 8.2 The large items

`eval_results/` holds 41,247 files (388 MB) on disk. About 17,400 are tracked, all under the re-admitted
`baselines/` and `performance_tables/` (`.gitignore:128-130`); the review puts the tracked total at 164 MB.
`design_docs/fleet-mission-evidence/` holds 2,012 tracked files (9.7 MB). Mission archives are about 8 MB.

**Readers that rule out a plain untrack:**
1. **`check-prompt-freeze` is a CI gate.** It runs `git ls-files eval_results/baselines/*.json`
   (`cmd/ailang/prompt_freeze_core.go:196`). Without the files it **passes vacuously**.
2. **The executor image's `benchmarks_*` MCP tools** are built from the baselines in Cloud Build
   (`tools/build-snapshot/main.go:562`). If the directory is absent, the build **silently writes an empty index**
   (`:565`).
3. The dashboard history export reads `eval_results/baselines/<ver>`.
4. Three Go tests and about 30 `tools/` scripts skip quietly when the data is absent.
5. Mission archives **must stay on origin**, because mission-control rebuilds its index from them in worktrees
   branched from origin (`internal/mission/index.go:16-22`). Keep them tracked.

**Recommended destination: a bucket plus a tracked index.** Not a data repo, which would add a second clone to
every gate. Not a release asset either: assets are per tag, while baselines change between tags.
- The raw `baselines/` and `performance_tables/` go to a versioned, public-read bucket prefix next to the existing
  benchmark read-through. This data is already published on the docs site.
- `eval_results/index.json` stays tracked: version, model, benchmark, pass and run id per row, generated by
  `make eval-index`. `check-prompt-freeze` and `build-snapshot` read it, and both **fail loudly** when neither the
  index nor the tree exists. That turns both silent fallbacks into errors.
- `make fetch-eval-data` rsyncs the bucket into the ignored directory for anyone who wants the raw rows.
- `fleet-mission-evidence/` makes a one-time move to the same bucket, leaving a stub README behind.
- **Before any upload:** the owner reviews the flagged file (§7.3). This doc does not scrub home paths from the
  rows. The review counts 2,671 tracked files with home paths; I did not re-measure that, and an index-based lower
  bound found ≥ 840. Most of them sit in these two trees.

**Retention rule:**
- Drop the two `eval_results` re-admits and ignore `fleet-mission-evidence/`. The writers upload instead of
  committing.
- Add a simplicity metric, `tracked_evidence_files` (gate: no growth), counting tracked files under
  `eval_results/` and `design_docs/*evidence*/`.
- `mission rotate-log` caps archives at one file per mission per quarter.

**History.** Untracking does **not** shrink the repository: every clone still downloads about 318 MB of history. A
`git filter-repo` rewrite would change every SHA on `dev`. It would break the 61 open PRs and the ledger's SHA
citations, and force every checkout, worktree and the rig to re-clone. **Recommendation: no rewrite before 1.0.**
Reconsider only if clone size becomes a measured problem.

**Re-bank.** After the untrack lands, run `make simplicity-audit-fast` and bank the snapshot. A regression means
"higher than the last snapshot" (`audit.sh:63-68`), so the drop re-baselines the audit and clears today's exit 2.

| Item | Action | Breaking? | Size |
|---|---|---|---|
| Four small untracks + ignores; re-admit `evaluations/` | `.gitignore` | no | S |
| Tracked eval index; loud failure in prompt-freeze and build-snapshot | code + make target | no | M |
| Upload and untrack `eval_results/` and the fleet evidence | after the owner review | no | M |
| Retention metric; re-bank | `tools/simplicity_metrics.sh` | no | S |

## 9. Declaring the platform split

**Paragraph for `stability.md` (after Tiers) and the README:**

> **What ships in the binary.** The `ailang` binary also contains the Internal tier: the coordinator, mission
> loops, eval harness, dashboards and the fleet tooling we use to build AILANG. It ships in the same binary today
> for convenience. During 1.x we intend to move it behind a separate Go module or binary. That is not a breaking
> change. Internal surface carries no promise, and every Stable and Experimental command, stdlib module and
> package format stays in `ailang`. If you depend on an Internal command, expect it to move.

**Tracking.** The simplicity metrics already record `binary_internal_packages` = **140** and the core closure
`closure_internal_packages` = **52**. Add `binary_bytes`, which is **110,013,586** for the darwin/arm64 scratch
build. **Contradiction:** the review's "253 modules" is the module graph (`go list -m all`, 253). Only **97**
modules are linked into the binary (`go version -m`). Track both.

## 10. Milestones

| M | Scope | Window | Est. | Acceptance (machine-checkable) | Mutation / control |
|---|---|---|---|---|---|
| M1 | Tier field, help rewrite, re-filing, `--help --all`, stability.md check, cli.md tier column | any | 2d | `TestHelp_TopLevelHasNoInternal` renders help and asserts every command token maps to a row with Tier ≠ Internal; `ailang --help \| wc -l` ≤ 50; `TestStabilityDocMatchesTable`; `make check-cli-docs` | Set `check`'s tier to Internal: help test fails. Delete a stability.md row: doc test fails. Control: `--help --all` lists `coordinator` |
| M2 | Per-command usages, pkg spellings, `pkg docs`, `version`/`sandbox-check` | any | 1.5d | For each Stable/Exp route, the first line of `--help` matches `^Usage: ailang ` and no line matches `^\s+-[a-z]` | Restore the default `fs.Usage` on `check`: test fails |
| M3 | `test` colour | any | 0.5d | Piped output has 0 ESC bytes; under a pty, `NO_COLOR=1` gives 0 | Under a pty without `NO_COLOR`, output has > 0 (control) |
| M4 | auto print, no Process in auto, pinned `[bin]`, widening refusal | **v0.58.x** | 2.5d | The shim has `--caps FS,IO` and `# caps:`; reinstalling a fixture v2 that adds Net exits ≠ 0 naming `Net`; `--grant-new-caps` passes; `auto` stderr has `granted (auto)`; `auto` with Process exits ≠ 0 | A v2 with no new cap installs cleanly (control). Remove the subset check: widening test fails |
| M5 | Pre-flight refusal | **v0.58.x** | 2d | `capsprobe` under `--caps IO` exits ≠ 0 with **empty stdout**; `branchy` is refused; census in the PR | Disable the check: stdout contains `step 1` |
| M6 | Go API moves, notices | before tag | 0.5d | `go list ./... \| grep -v /internal/` shows neither `/runtime` nor `third_party`; every kept package's doc contains "not covered by the 1.x" | Delete one notice: test fails |
| M7 | Ollama wire types | before tag | 2d | `go list -deps ./cmd/ailang \| grep -c github.com/ollama` = 0; no ollama in `go.mod`; parity tests pass | Recorded SDK bytes are the control; change a field tag: parity fails |
| M8 | zod, CodeQL triage, hardenings, read-path fix | before tag | 1.5d | `gh api` reports 0 open high CodeQL alerts without a `dismissed_comment`; zod ≥ 4.6.6 in the lock | — |
| M9 | Untracks, eval index, relocation, retention, re-bank | before tag | 3–4d | The index matches no newly ignored path; `tracked_files` ≤ 13,000 in the banked snapshot; with `eval_results/` deleted, prompt-freeze and build-snapshot exit ≠ 0 and pass when only `index.json` is present; docs build green | Restore the empty-index fallback: the new test fails |
| M10 | Split paragraph, `binary_bytes` | any | 0.5d | stability.md and README contain the paragraph; the snapshot has `binary_bytes` | — |

M4 and M5 are **breaking** and must ship in v0.58.x. Everything else is non-breaking. Re-filing changes only
listings, and the Go packages that move have no external importers. Single-threaded, the total is about
16 engineer-days; M1–M3 and M6–M8 can run in parallel (an estimate, not calibrated against velocity).

## 11. Changelog fragment drafts

`changelogs/unreleased/<date>-help-shows-the-promise.md`
```markdown
### Changed — `ailang --help` lists only Stable and Experimental commands

Top-level help shrinks from 189 to 47 lines. `eval`, `serve` and `iface` (Internal) move to their group listings,
and every spelling still works. `ailang --help --all` shows everything with its tier. `check`, `run`, `verify` and
`ai-check` print real usage. `ailang test` honours `NO_COLOR` and stops colouring piped output.
```
`changelogs/unreleased/<date>-caps-pinned-and-preflight.md`
```markdown
### Changed — capability grants are printed, pinned and checked before a run (BREAKING)

`run --caps auto` prints its grant and never grants `Process`. `ailang install` resolves a `[bin]`'s `auto` caps
once, prints them and pins them in the shim; a version that needs more is refused until re-run with
`--grant-new-caps`. `run` refuses before any side effect when the entry declares an effect that was not granted
(`E_CAPS_MISSING`). Migration: pass the full `--caps` list your entry declares.
```
`changelogs/unreleased/<date>-drop-ollama-module.md`
```markdown
### Changed — the Ollama client uses local wire types; `github.com/ollama/ollama` is no longer a dependency

This clears the two Dependabot alerts and nine allow-listed govulncheck advisories. Parity tests show the requests
are byte-identical. A direct `OLLAMA_HOST=ollama.com` is refused; use the local daemon proxy.
```

## 12. Risks

| Risk | Mitigation |
|---|---|
| Pre-flight breaks harness runs that pass fixed caps | Run the M5 census first and fix those lists in the same PR |
| Skills or tools grep the old help text | No route changes; grep `.claude/skills` and `tools/` for help strings in M1 |
| `[bin]` install now type-checks, so it is slower and can refuse | Publish already compiles; a refusal is the correct outcome |
| The wire types drift from the Ollama server | Parity tests plus one live smoke run on the rig, outside CI |
| Relocation breaks a reader | The §8.2 reader census is a precondition; loud failures replace silent ones |
| Untracking leaves history unchanged | Stated in §8.2; no rewrite |

## 13. Non-goals

Dead code; dashboard UI; the platform split itself; Sonar; the 236 medium log-injection alerts; `serve-api`
per-route pre-flight; publisher signatures; renaming a Stable command; a history rewrite.

## 14. Decisions for Mark

1. **D1: pin `[bin]` grants, with `--grant-new-caps` to widen.** Recommend **yes**. The two real users already pin
   by hand.
2. **D2: make pre-flight an error with no escape hatch.** Recommend **yes**. Revisit only if the M5 census finds
   legitimate cases.
3. **D3: `auto` never grants `Process`.** Recommend **yes**: `Process` escapes the FS sandbox (review §6).
4. **D4: state that the Go API is Internal; move `runtime/` and `goopus` under `internal/`.** Recommend **yes**.
   World is unaffected.
5. **D5: move `eval_results/` to a bucket plus a tracked index, with no history rewrite before 1.0.** Recommend
   **yes**.
6. **D6: enable secret scanning and push protection.** Recommend **yes, now**. It is a repo setting that takes
   minutes.
7. **D7: zod v3 → v4 in the pi sandbox, or drop the dependency.** Recommend the bump, tested off the rig.
8. **D8: is `cmd/zz_realtask_runner_tmp` a leftover?** If so, delete it.

## 15. Axiom compliance

A3 effect legibility **+1** (grants printed and recorded) · A4 explicit authority **+1** (no silent widening, no
partial runs) · A7 machines first **+1** (one-screen help; tier is data) · A11 structured failure **+1**
(`E_CAPS_MISSING` before side effects) · A12 system boundary **+1** (Go API and platform boundary stated) · all
others 0. **Net +5.** Hard-violation check: no nondeterminism (A1), no hidden effects (A3), no ambient authority
(A4; this removes one), nothing traded away for human convenience (A7).
