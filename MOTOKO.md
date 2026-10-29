# motoko_agent on this machine — the map

**Rig answer (since 2026-10-27):** evals run `/Users/voightkampff/dev/mk-20261027`, branch
`sunholo/main-dst-20261027` @ `835c738b` (§13) — upstream `main` 97244396 + our commits. The Docker source pin
is now `af1d5651` with the approved #1958 callback migration (§14); the live rig was not switched. 2026-10-19..10-27: `mk-20261019` / `sunholo/main-dst-20261019` @ `907dd78e` (§12), now
the rollback. 2026-10-15..10-19: `mk-20261015` / `sunholo/main-dst-20261015` @
`965f2ec5` (§11). 2026-10-08..10-15: `mk-20261002` (§10). Before that (2026-09-28..10-07):
`/Users/voightkampff/dev/mk-main`, branch `sunholo/main-dst` = Arni's `main` (DST core, extension ABI 8.0) plus our local commits
(`cloud` and `ollama_microrag` profiles, `motoko_ext_ailang_tools`, rig-lease forwarding).
Its extensions are the in-repo `packages/` copies. The ABI 2.2 fork (`sunholo/eval-canonical`)
and its worktrees were removed on 2026-09-28; the branch survives on our fork. See §9.

This file exists because "which motoko are we actually running?" cost a full
audit once. Read it before touching any `~/dev/mk-*` directory.

---

## 1. There is only ONE clone

```
/Users/voightkampff/dev/arniwesth/motoko_agent     <- the clone
  remotes:  origin = arniwesth/motoko_agent            (upstream, Arni's)
            fork   = sunholo-voight-kampff/motoko_agent (ours)
```

Every `~/dev/mk-*` directory is a **git worktree** of that one clone — not a
separate checkout, not a separate fork. `git worktree list` is the truth.

| Path | Branch | What it is |
|---|---|---|
| `dev/arniwesth/motoko_agent` | `feat/local-eval-profiles` | the clone; stale branch, do not eval from it |
| **`dev/mk-main`** | **`sunholo/main-dst`** | **CANONICAL — what `motoko` on PATH runs** (Arni's `main`, ABI 8.0; see §9) |

Branches `integration/sync-clean-20260624` (was mk-sync) and
`integration/editdecl-timeout` (was mk-integration) still exist but their
worktrees were removed on 2026-07-28: both are fully superseded by mk-ast.

"Fully superseded" was verified by commit-subject comparison, and for the one
commit whose subject was absent (`register ollama/qwen3 context limit =
262144`) by confirming the same content is present in mk-ast under a different
lineage. Removing those worktrees deletes no unique work; the branches survive
regardless.

`dev/sunholo-data/motoko_explore` is an unrelated repo. Ignore it.

## 2. How the eval harness finds motoko

`~/go/bin/motoko` is a shim, not a binary:

```bash
exec /Users/voightkampff/dev/mk-main/scripts/run-agent.sh "$@"
```

`run-agent.sh` prefers a repo-local `ailang/bin/ailang` if one exists — it
currently does **not**, so motoko uses the `ailang` on PATH. If motoko ever
behaves like an old compiler, check for that file first.

## 3. Extensions: the in-repo `packages/` are the source of truth

Upstream `main` resolves every extension by path (`mk-main/ailang.toml`:
`{ path = "packages/motoko-ext-*" }`). The ABI 2.2 registry packages
(`sunholo/motoko_ext_*`) were unpublished on 2026-09-28 — see
`ailang-packages/packages/MOTOKO_EXTENSIONS_RETIRED.md`. Edit an extension in
`mk-main/packages/`, then `make check_core && make verify_extensions`. Ours
(`motoko-ext-ailang-tools`) goes upstream as a PR (arniwesth/motoko_agent#200).

## 4. Profiles decide which extensions actually load

Compiled-in ≠ enabled. A profile's `extensions.order` is what loads at runtime.

| Profile | Extensions | Verify gate | Used by |
|---|---|---|---|
| `cloud` | empty_stop_guard, compaction_ai, context_mode, ailang_docs, ailang_tools, microrag | `ailang check benchmark/solution.ail` | all cloud `motoko-*` models |
| `ollama_microrag` | the `cloud` set, compaction on the local model | — | `motoko-local-qwen3-8-27b-microrag` (full local stack) |
| `ollama` | compaction_ai, context_mode (upstream's, 50 steps) | — | `motoko-local-*` lean arms |
| `dogfood` | compaction_ai, context_mode, exa_search | `make check_core` | motoko's own self-hosting work |

`dogfood` is motoko's **development** profile — its `make check_core` gate only
makes sense inside the motoko repo. It is not for benchmarks. Cloud eval models
previously fell through to it by default, which is why their runs had neither
the AILANG-knowledge extensions nor a meaningful verify gate.

Profile selection lives in `internal/eval_harness/models.yml` as
`motoko_profile:`. **Every** motoko model now sets it explicitly — no implicit
defaulting.

## 5. Known deferral: motoko_ext_a2a is not compiled in

`a2a`'s delegate path calls `uuid4()` (Rand), but `motoko_ext_abi` 2.2.0
declares **closed** effect rows on `ExtensionHooks` that exclude Rand, so it
cannot type-check under the post-`1282767ca` effect checker. Re-enabling it
needs an ABI bump (add Rand to the hook rows), which forces a re-annotation
through every extension. No profile references a2a, so nothing is lost today.

## 6. When motoko "won't start"

Startup crashes are silent in the eval output — check the stderr log first:

```bash
ls -t $TMPDIR/motoko-stderr-*.log | head -1 | xargs tail -20
```

Historic causes, in order of likelihood:

1. **Effect-checking failure after an AILANG upgrade.** AILANG's checker gets
   stricter over time; motoko's declared rows must widen to match. This killed
   motoko for six days in July 2026 (`1282767ca`) and 72 runs were banked as
   failures before anyone noticed.
2. **stdlib drift** — the stderr log reads "stdlib version mismatch". The executor
   now exports `AILANG_STDLIB_PATH` only for a std/ identical to the binary's own.
3. **423 from the rig gateway** — a local model called without the rig lease
   (`/Users/Shared/ailang/rig-gate.jsonl`, decision `refused-none`).

Port 8080 is no longer a cause: the executor gives every run its own `ENV_PORT`.

A green boot is:

```bash
cd /Users/voightkampff/dev/mk-main && make check_core && make verify_extensions
```

## 7. Staying mergeable with upstream

Arni is actively refactoring `origin/main` (~18 open PRs). Our branch carries
~45 commits they don't have; we are behind on theirs. Deliberate choices that
keep the merge cheap:

- **Never run `ailang fmt` across motoko sources.** It reflows whole
  expressions and inserts blank lines between imports, producing hundreds of
  lines of conflict surface for no benefit. Keep diffs signature-sized.
- Query our delta with `git log origin/main..sunholo/main-dst`.
- Rebase deliberately, not reflexively — check what Arni has in flight first
  (`gh pr list --repo arniwesth/motoko_agent`).

## 8. Housekeeping

2026-07-28: worktrees for two superseded branches removed; the eval branch
renamed `integration/sync-ast-20260624` -> `sunholo/eval-canonical`.

2026-09-28: worktrees `mk-ast`, `mk-prwork`, `mk-ast-upstream-fix` removed. Every
branch tip is on our fork; `mk-prwork`'s uncommitted snapshot of older upstream
files was committed to `backup/mk-prwork-staged-20260928` first.

Still outstanding: the clone sits on the stale `feat/local-eval-profiles`.

## 9. Migration to upstream `main` (ABI 8.0) — in progress since 2026-09-26

Plan (completed 2026-09-28/30): `design_docs/implemented/v0_48_0/m-motoko-dst-refactor-migration.md` (per-commit verdicts:
`m-motoko-fork-disposition.md`). **The shim was switched to `mk-main` on 2026-09-28 (Mark):**
we are adopting it, so it is measured on its own (before/after per change), not A/B'd
against the old fork.

- `~/dev/mk-main` builds on AILANG **≥ v0.44.1** (v0.42.0–v0.44.0 fail on a `deriving (Eq)`
  regression; v0.44.1 fixed it). `make check_core`: 60/60 core, 9/9 extensions.
- Our delta on top of upstream: `git log origin/main..sunholo/main-dst` (cloud and
  `ollama_microrag` profiles, `motoko_ext_ailang_tools` = PR #200, rig-lease forwarding, and
  one local-only commit that drops `src/core/ailang.toml`). #191, #193 and #198 are merged.
- `make dst` needs **GNU make 4** (`gmake`, via `brew install make`). Stock macOS make 3.81
  skips the parallel targets and still prints "all targets passed".
- Headless runs need `MODEL` set explicitly: without it motoko ignores the profile's model and
  falls back to `anthropic/claude-sonnet-4-6`. The eval executor always sets it.
- **The rig lease.** Since 2026-09-28 the rig's ollama sits behind a gateway that, under a held
  rig lock, refuses long work without the holder's token (M-RIG-GPU-ADMISSION-GATEWAY). The token
  reaches motoko's AILANG child as `AILANG_RIG_LEASE`; mk-main's `buildChildEnv` allowlist dropped
  it, so every local-model step got 423. Fixed on `sunholo/main-dst` (3e09f986, with a test). It
  also needs an ailang whose clients attach the lease (dev after adc59d45b; not in v0.47.2).
- Open questions to Arni (package source of truth, where AILANG extensions live, a post-tool
  hook, versioning): arniwesth/motoko_agent#192. Arni confirms ABI 8.0 is stable.
- Ported: compact-interface auto-read (opt-in `autoread`, OFF in every profile we run),
  per-edit `ailang check` and whitespace-tolerant `EditFile` — all in `motoko_ext_ailang_tools`,
  merged upstream as #208. Not yet ported: fmt, `MOTOKO_MAX_STEPS`.
  ABI 8.0 CAN wrap native tools: a `ToolProvider` advertising `"WriteFile"`/`"EditFile"`/
  `"ReadFile"` receives those calls and `Delegate` falls through to the next provider, then
  native (see `motoko_ext_microrag`). It cannot APPEND to a native result, so the extension
  performs the write/edit itself. Not blocked on Arni.
- Local motoko is IN the GPU rotation (`motoko-local-qwen3-8-27b-microrag`, since 2026-09-29).
  09-29..10-02 on the same benchmarks: motoko 44/53, pi 31/93, opencode 28/86 (pi/opencode
  mostly config faults and budget kills, not the model).

### To do (agreed 2026-09-28)
- **Cloud executors:** once motoko main is stable locally, update the AILANG cloud executor images
  to motoko main + `motoko_ext_ailang_tools` (they still carry the old fork). Mark: "once you are
  happy with motoko locally".
- **Registry packages: RETIRED 2026-09-28 (Mark).** All 14 ABI 2.2 `sunholo/motoko_ext_*`
  packages (96 versions) were unpublished; see `ailang-packages/packages/MOTOKO_EXTENSIONS_RETIRED.md`.
  Still to do: port `fmt` to ABI 8.0 (likely inside `motoko_ext_ailang_tools`).
- ~~Local motoko in the rotation~~ — done (see above).

## 10. Canonical since 2026-10-08: `sunholo/main-dst-20261002` (rig and cloud on `de68fddf`)

Worktree `~/dev/mk-20261002`, fork branch `sunholo/main-dst-20261002`. It is upstream `main`
`4023bf08` (strict extensions #205/#206, `ailang_tools` #208 with Arni's review fixes) plus:

- our profiles and lane (`cloud`, `ollama_microrag`, `ailang_only`, `motoko_ext_ailang_policy`);
- `extensions.strict = true` on `cloud`, `ollama_microrag` and `ollama`: a missing extension now
  refuses to start instead of silently running defaults (registries build 6/6/2 under strict);
- PR #209 cherry-picked. **Upstream `main` without #209 fails every `EditFile` on macOS**
  (`chmod --reference`), so never point the rig at plain upstream `main`;
- a tool-schema dedupe (`tool_catalog.ail`, half of upstream #204). `ailang_tools` and `microrag`
  both wrap ReadFile/WriteFile/EditFile, and each wrap appended a second schema; OpenRouter
  providers Relace and Sail Research reject duplicate tool names with 400 ("Provider returned
  error"). `sunholo/main-dst` (mk-main) still sends the duplicates.
- a portable path guard: motoko's native guard and `ailang_tools`' `inside_workdir` both
  needed GNU `realpath -m`. On macOS the native guard let symlink escapes through and
  `ailang_tools` delegated every `.ail` call (inert on the rig). Both now resolve the deepest
  existing ancestor with plain `realpath`. `sunholo/main-dst` (mk-main) does not have #208's
  guard, so its `ailang_tools` works today — switching to plain upstream would break it.

Verified on it (macOS): `make check_core` passes in full (10/10 path-guard checks, symlink
escapes included), `ailang_tools` unit tests, every profile's registry under strict, and 6
concurrent cloud motoko trials (`motoko-or-deepseek-v4-flash`, 6/6, $0.06, no port collision).

- `agent.context_limit: 262144` on our profiles (e3fa68ec, 2026-10-08): the catalogue lacks every fleet model, so compaction never fired (context limit Unknown); the profile override makes it compact at ~196k.
- the ailang_only lane teaches AILANG (b3347a96, 2026-10-12): the lane's system role was 1,783
  chars against pi's canonical `ailang prompt` (~23k tokens) plus scratch-dir / package-ceiling
  guidance. `motoko_ext_ailang_policy` now reads `ailang prompt` at registration and prepends it
  (AILANG_LANE_TEACHING=0 turns it off, as on pi). On Anthropic/Gemini it only reaches the model
  with ailang ≥ v0.54.6 (those providers dropped system-role messages). Cloud image pins it.
- `repetition_guard` loads in the ailang_only profile (a1ece42e, 2026-10-12): GLM-5.3-Flash sent
  one malformed `ailang_cli` call 153 times on a real task; the guard refuses a call repeated with
  byte-identical output and tells the model to change course. It could never fire on extension
  tools until 6de44ada (upstream #260): each result envelope carries its own `tool_call_id`, so no
  two results were identical (measured: 215 repeats, guard loaded, silent).
- the in-run cost cap prices prompt-cache tokens (d668e630, 2026-10-12): `CostRates` gains cache
  read/write rates (the executor forwards them from the budget), and cache reads come out of input
  on OpenAI-shaped routes, where providers count them inside it. Before, GLM lane runs stopped at a
  self-reported $2.00 that OpenRouter billed at about $0.45.

**Rig switched 2026-10-08:** the `~/go/bin/motoko` shim now execs `~/dev/mk-20261002` (this branch is
checked out in that worktree, so `mk-main` cannot take it; `mk-main` stays on `sunholo/main-dst` as
the rollback — repoint the shim's exec line to go back). The cloud image (`docker/Dockerfile.agent-motoko`) pins `de68fddf` on this
branch since 2026-10-02: the dev plane builds it from `dev`; test/prod get it only via a release +
promote. Until the rig switches, rig and cloud run different motoko commits.

## 11. Canonical since 2026-10-15: `sunholo/main-dst-20261015` (rig and cloud on `965f2ec5`)

Worktree `~/dev/mk-20261015`, fork branch `sunholo/main-dst-20261015`: §10's branch rebased onto
upstream `main` `340fae14` (2026-10-14), 346 upstream commits newer than `4023bf08` (about 300
non-merge; most are chore/docs, ~40 touch behaviour: `refactor(core)`, `fix(tui)`, `fix(tools)`,
spec 037, strict context-limit refusal #237).

What the rebase did to our commits:

- dropped as already upstream: repetition_guard tool_call_id (#260), byte-stable compaction summary
  (#262), and our tool-schema dedupe (40d94780) — upstream landed the same fix as #220 and closed #235;
  the `tool_catalog.ail` conflict was resolved to upstream's version;
- the `verify_native_path_guard` Makefile fix (0992d765) applied empty — its content is in the tree;
- `scripts/smoke_v2_dp7_gate.ail` (touched by the cost-cap commit) was deleted upstream; kept deleted;
- `ailang.lock` taken from upstream and relocked (`d5aa3e31`), which adds `motoko_ext_ailang_policy`;
  no absolute paths.

Upstream's strict context-limit refusal (#237: strict + compactor + unknown limit exits 2) does not
bite our profiles — `cloud`, `ollama`, `ollama_microrag` and `ailang_only` all set
`agent.context_limit: 262144`.

Verified: `make check_core` passes in full on macOS (59/59 core files, 192 `OK` checks, no FAIL).
Not yet verified at switch time: a live eval trial on the new commit (local or cloud).

2026-10-16: fast-forwarded to `965f2ec5` — the TUI's child-env allowlist now forwards
`AILANG_OLLAMA_V1_STREAM`, `AILANG_OLLAMA_IDLE_TIMEOUT_SEC` and `AILANG_OLLAMA_TTFT_TIMEOUT_SEC`
to the runtime (upstream as arniwesth/motoko_agent#266). Before it, the executor's streaming flag
for `ollama/*` models never reached the runtime, every local turn ran buffered, and the idle
watchdog killed long generations (9 of 39 v0.54.9 qwen3.8 rows). Only the env differs; requests
without the flag are byte-identical. Smokes on `d5aa3e31` (the same tree minus this commit):
local qwen3.8 fizzbuzz / json_parse / glob_match_spec, cloud Sol 6.1 and OpenRouter deepseek-v4-flash
fizzbuzz / json_parse, all passed (2026-10-15/16).

**Rollback:** repoint `~/go/bin/motoko`'s exec line to `~/dev/mk-20261002` and the Dockerfile's
`MOTOKO_COMMIT` to `b1ffa339`. Results banked before 2026-10-15 13:00 (including the v0.54.9 OS
rotation) ran `b1ffa339`; do not pool them with later motoko rows as one build.

## 12. Canonical since 2026-10-19: `sunholo/main-dst-20261019` (rig and cloud on `907dd78e`)

Worktree `~/dev/mk-20261019`, fork branch `sunholo/main-dst-20261019`: §11's 20 commits rebased onto
upstream `main` `a94204aa` (2026-10-18), 93 upstream commits newer than `340fae14`. Most of them are
ADR 034 Phase 0 (#277: a tool fault carries a `ToolFault` saying whether its effect may have
happened; `Search` is read-only), the Phase 0b deadline-wrapper probe and its withdrawal (#279),
herdr `DelegateCheck` fixes (#275, #278: park instead of polling, bounded waits) and
`progress_contract_guard` taking its contract from the operator, not a tool result (#270).

The only conflict was `ailang.lock`; it was taken from upstream and relocked (`a241b15d`, whose
message still says "340fae14": it is the same relock step, re-run on the new base). No absolute paths.

Verified 2026-10-19: `make check_core` exit 0 (60/60 core files, 9/9 extensions boot); eval smoke
through a scratch shim, Sol 6.1 on the ChatGPT subscription, fizzbuzz and json_parse both pass, with
the sessions in `mk-20261019/.motoko/logfile/`. Not yet verified: a cloud image build on this commit,
and a local ollama or OpenRouter run.

**Rollback:** repoint `~/go/bin/motoko`'s exec line to `~/dev/mk-20261015` and the Dockerfile's
`MOTOKO_COMMIT` to `965f2ec5`. Results banked before 2026-10-19 ran `965f2ec5` or earlier; do not pool
them with later motoko rows as one build.

## 13. Canonical since 2026-10-27: `sunholo/main-dst-20261027` (rig on `835c738b`; cloud source update in §14)

Worktree `~/dev/mk-20261027`: §12's commits rebased onto upstream `main` `97244396` (2026-10-25),
31 upstream commits newer than `a94204aa`. Most are ADR 034 Phases 1–3 (DST world/fault catalogue,
properties, run report: test infrastructure). The one that changes runs is **#286** (closes #263, raised
from our 2026-10-13 measurement): extension `ai_step` calls go out with no `tools`, so the compaction
summariser can no longer answer with a tool call and fall into its 75 s retry.

Two carried commits were replaced by their reviewed upstream-PR versions:

- **Path guard** → #234 as revised 2026-10-27 (`b127204a` + `60fedfde`). The old carried guard
  (`7349b452`) walked with `$(dirname …)`, which drops trailing newlines: a symlink named `nl\n` was
  walked past and `WriteFile nl\n/x` landed outside the workdir. **That hole was live in `907dd78e`
  and every build before it that carried the guard.** The walk now uses parameter expansion; the guard
  lives in `tool_runtime` and `ailang_tools` (ABI stays 8.0), kept equal by `verify_native_path_guard`.
- **Ollama stream env** → #266 as revised (comment only: deltas reach the session log on AILANG ≥ v0.54.13).

Conflicts: `ailang.lock` (relocked, no absolute paths) and the #266 hunk. 21 carried commits.

Verified 2026-10-27: `make check_core` exit 0 (60/60 core files, 9/9 extensions boot, path guard 13
OK incl. the newline link, each new check mutation-tested); `tsc` clean; TUI jest failures identical
to pristine upstream `main` (20 suites / 1 test, environmental); `ailang_only` resolves through
motoko's resolver and passes every `checkLaneProfile` property; eval smoke via a scratch shim, Sol 6.1
on the ChatGPT subscription, fizzbuzz + json_parse pass (sessions in `mk-20261027/.motoko/logfile/`).
Local ollama smoke 2026-10-27 12:39 (`motoko-local-qwen3-8-27b-microrag`, fizzbuzz) passes in 6 turns,
230 s, with 272 reasoning and 421 thinking deltas journalled during the calls. The rig shim was switched to
`mk-20261027` at 12:43 (atomic `mv`). **Not yet verified:** a cloud run on the rebuilt image.

The rig lock was orphaned 09:18–12:38 that day (`rig.lock.d` holding only a stale `children` file, no
`holder`): `holderAlive` treats a missing holder as alive, so nothing could take it until the 6 h
staleness window, and the rotation yielded every slot. Cleared by hand; cause not found.

**Rollback (rig):** repoint `~/go/bin/motoko`'s exec line to `~/dev/mk-20261019`. Results banked before
2026-10-27 12:43 ran `907dd78e` or earlier; do not pool them with later motoko rows as one build.
**Rollback (cloud):** `MOTOKO_COMMIT` back to `907dd78e`.


## 14. Approved #1958 compiler compatibility migration (2026-10-29)

The first #1958 source-pin migration selected **`af1d56513d3ae314bd831fd149c7b6f2503a361c`** from
our fork `sunholo-voight-kampff/motoko_agent`, PR [#4](https://github.com/sunholo-voight-kampff/motoko_agent/pull/4).
It is exactly one annotation line after `835c738b8ca50ed38f36c117058847c816db32b8`,
targeting our `sunholo/main-dst-20261027` branch. `src/core/test/stub_step.ail`'s
`play_chunks` callback slot narrows `IO, Trace` to `IO`, matching the Ports callback.
The helper's outer `IO, Trace` row and Ports ABI stay unchanged. The stricter compiler
correctly rejects implicit callback widening until width subsumption is implemented.

Before migration, the same 60-file core selection passed **60/60** with the base
compiler and **55/60** with #1958. After migration it passed **60/60 on both arms**.
The real delivery worktree also passed fresh no-cache checks with #1958 compiler
`9eb1ebaed78a0ecf6b232b5b329633ed11438358` and its matching stdlib: **60/60**.
Full `gmake check_core` passed **60 core files, 9 extension boots**, and all runtime
prerequisites. Seven supplemental checks initially hit a 90-second deadline, then
passed at lower concurrency with a 180-second deadline (25–122 seconds); timeouts
remain recorded in the validation artifact. Use GNU make on this Mac.

This updates the cloud **source pin**, not a built/deployed image. The live rig
checkout and shim remain on §13's `835c738b`; neither they nor upstream
`arniwesth/motoko_agent` were modified. Do not infer cloud deployment or rig migration
from this pin. #1958's breaking row guard targets **AILANG v0.56.0**, not a v0.55.x patch.

Fork PR #4 merged normally on 2026-10-29, without an admin override. Its merged
source tree is byte-identical to tested migration head `c3f4c85eec681d8d5d4787da4d9e56c22c0a2457`.
CI's core/extensions gate passed, then its skills fixture failed on an inherited
v0.52.5 pricing-record mismatch. The same error reproduces unchanged on the pinned
base and migrated head; [the PR records the evidence](https://github.com/sunholo-voight-kampff/motoko_agent/pull/4#issuecomment-6491761805).
The broader CI is not claimed green. #1958's final integrated independent gate remains pending. See `design_docs/planned/v0_55_0/m-effect-row-open-closed-soundness-validation.md`
for preserved original failures and final delivery/integration status.


## 15. Approved explicit-empty callback source migration (2026-10-29)

The current Docker source pin is **`083ef43726981099a0bad4fedda18fc72105d6fa`**,
normal merge of fork PR [#5](https://github.com/sunholo-voight-kampff/motoko_agent/pull/5)
onto `sunholo/main-dst-20261027` after §14's `af1d5651`. It changes exactly four
annotation lines in `src/core/test/ext_fixture.ail`: `describe`, `prompt`, `budget`
and `policy` callback fields gain `! {}`. Effectful siblings, outer factory rows,
runtime behavior, lockfile and Ports ABI are unchanged. The merged source tree is
identical to tested head `8dc0ad84f9a9dc029d624468278def981f807e07`.

With #1958 plus merged #1963, the original fixture hand-off fails 55/60 core checks.
The four annotations also fail before the separately approved explicit-empty
compiler amendment. Actual amended compiler `9ab45c98e071c9c90f1c365ff82cca00d74ef151`
and matching stdlib restore **fresh no-cache 60/60** and actual **`gmake check_core`
60 core files / 9 extension boots**, with all runtime prerequisites. Three initial
180-second no-cache deadlines under two workers pass on serial retries in
127.07s, 79.44s and 67.97s; the initial results remain recorded. This preserves the
original width 60→55, width-migrated 60→60 and combined nested 55/60 measurements.

The fork's old v0.52.5 CI compiler reproduces the same inherited skills pricing
record 2-versus-4-field failure on both `af1d5651` and the four-field candidate.
Relevant actual-compiler gates and root's independent exact-diff review passed;
normal merge used no admin override. Pending/baseline fork CI is not claimed green.
See `design_docs/planned/v0_56_0/m-effect-explicit-empty-callback-contract-validation.md`
and its compatibility JSON for exact pins, corpus/control results and final gates.

This is the v0.56.0 compiler-compatible **source pin**, not a built/deployed image.
The live rig/shim remain at §13's `835c738b`; upstream `arniwesth/motoko_agent` was
not modified. Final AILANG independent evaluation and exact-head CI remain pending.
