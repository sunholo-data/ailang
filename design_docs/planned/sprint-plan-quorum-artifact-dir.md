# Sprint Plan: M-FLEET-QUORUM-ARTIFACT-DIR

## Summary

Make future host quorum evidence durable across worktrees by registering an opt-in absolute directory, preserving policy confinement, and teaching both Gate 2 readers the same fixed-order lookup.

**Ticket:** `quorum:artifact-dir-cwd-relative` · **Mission:** fleet-mission iteration 38  
**Input:** `design_docs/planned/m-quorum-artifact-dir-stable.md`, revision 3, HEAD `58decf4f2f42bfc20eb267fc0b2368462cc32231`  
**Duration:** 1.5 days / 12 hours, including independent evaluation and buffer  
**Milestones:** exactly M1 (resolution/confinement, 4h) and M2 (driver/reader/integration, 8h)  
**Estimate:** M1 300 LOC + M2 500 LOC = 800 LOC; risk medium.

This is a planner deliverable. The controller owns approval, commits, dispatch, landing and activation. No implementation, importer, message handoff or git write is part of this planner run.

## Current status analysis

- Required `CLAUDE.md` and `.agents/skills/sprint-planner/SKILL.md` read. Initial and pre-write git status were clean. This is an unattended mission-role task; ambient inbox is outside this assignment.
- This tree includes PR #1824 salvage: `23467c484` added 268 lines; `22bad5519` added 222/deleted 9, including the current artifact Markdown salvage suffixes. Preserve those additions.
- The velocity script returned rc141 from its `git log | head` pipeline under pipefail. It did not establish LOC/day. Two recent local quorum milestones provide size evidence, not measured elapsed-time throughput. Budget 533 LOC/day over 1.5 days, with explicit integration/evaluation allowance.
- Neither directory resolver nor new shell helper/test exists. Current writer, constants, driver, source skill and mirror were inspected. Baseline gates below are measured on this stacked tree.
- No .ail changes or runnable language examples are involved. Operational examples are hermetic repo/HOME fixtures in the named tests; contracts/effects/inline .ail tests are not applicable.

## Premise check

These corrections supersede conflicting design prose for execution; the input design remains unedited because this assignment permits exactly two artifacts.

| Premise | Verified current anchor and correction |
|---|---|
| Registry must use an AILANG-prefixed env name | False. `internal/config/mission.go:13,42` declares/registers `EnvMissionDesignAuthor = "MISSION_DESIGN_AUTHOR"`. `registry_test.go:60–101` checks Env* constants, exact-once registration, area and documentation; it imposes no AILANG_ prefix. `.golangci.yml` forbidigo prohibits os.Getenv/os.LookupEnv outside exempt config/test paths, not MISSION_ names. `EnvMissionQuorumArtifactDir` is acceptable. |
| Getter pattern exists | Yes: `mission.go:72` uses raw `get(...)`; `registry.go:118` implements it. `MissionDesignAuthor():65` trims. More critically, existing `job.go:109` AgentPolicy() trims: do NOT reuse it for confinement. Add `MissionQuorumArtifactDir() string { return get(EnvMissionQuorumArtifactDir) }` and `AgentPolicyPresentRaw() bool { return get(EnvAgentPolicy) != "" }` in allowed mission.go only. |
| Writer anchored before salvage | Current `artifact.go:14` sentinel, `:40–44` default/MkdirAll, `:60` O_EXCL, `:97–105` salvage rendering. Modify the resolver/write preamble; preserve salvage output, JSON schema, slug and exclusive-create loop. |
| Constant is unpinned | False: `internal/policytool/policytool_deny_test.go:147` TestPolicyTool_QuorumArtifactDirInSync compares it with policytool's private constant. Run unchanged; add effective-destination chain coverage outside policytool. |
| Profiles/secrets/controller placement | Profile sources are `mission-control.sh:71–72,81–82`; secrets source `:119`, log defined `:123`. Insert checked helper source and initialization immediately AFTER log() and before credential/probe work. Source from MC_DRIVER_ROOT, which was captured at `:47`; do not depend on MISSION_DRIVER_ROOT, exported only at `:2246`. |
| There is only one controller launch path | Binary path `:2048–2056` execs `ailang mission iterate`; legacy Codex `:2337–2340`, Pi `:2381`, Claude `:2383`. Early initialization covers BOTH binary and legacy paths, before any reviewers/controllers. The driver uses set -uo pipefail (`:38`), not set -e: check source and initializer rc explicitly. |
| Codex needs a new forwarding rule | No: `lib/codex-env-args.sh:40` already forwards MISSION_*, and the name is not credential-shaped (`:44`). Do not edit the helper. Its argv policy does not forward AILANG_AGENT_POLICY; the CLI process still inherits it. Capture process marker presence separately from per-shell .set overrides; never claim all controller shell policies retain it. |
| Recipe environments guarantee empty markers | False: `dispatch/run.go:199–259` taskFor sets no PolicyPath; `executor/envpolicy.go:89–95` permits AILANG_* and MISSION_*. BuildEnvironment `environment.go:89,105,163` filters inheritance and overrides the marker only for a nonempty task policy. Test these real builders, without changing them. |
| Reader can stop after an exported-directory match | Incorrect under rev3: error in EITHER directory is an error. Consult and validate both, export first then reader repo root, aggregate results before REVIEWED/UNREVIEWED. Use indexed arrays/quoted arguments, never whitespace-split paths. |
| AC11 reverse direction works with no-export attended reader | Too broad. The frozen decision table makes no-export attended readers local-only. Reverse direction means the attended reader shares the same explicit export as the mission writer. AC9 separately proves no-export local writer/reader agreement. No implicit host-default discovery is added. |
| “Do not automatically ... read both trees” | Interpret the older rollout sentence as forbidding legacy/main-clone scavenging; rev3 explicitly REQUIRES export + current-reader-repo fallback. It does not authorize reading a different checkout or importing evidence. |
| Existing Gate 2 already enforces freshness | It currently gives a presence lookup at `gate-2-pick.md:139`, invocation `:140,199`, nested JSON examples `:224–228` and historical measurement `:215`. Preserve existing verdict/remedy policy, schema paths and doc slug pattern; do not invent a freshness or conflict-selection policy in this path fix. |
| CLI can validate before paid reviews within scope | No: `cmd/ailang/design_quorum.go:89` reviews precede writer `:97` and successful path diagnostic `:102`. CLI edits are forbidden. Driver preflight is in scope; standalone late IO validation and sentinel-equal explicit flags remain documented limitations. |

Verified scope implementation: `tools/launchd/githooks/pre-push:16–18,31–32` permits internal/mission, mission skills/mirrors, docs/changelogs and ONLY internal/config/mission.go in config. No edits or mutation patches are planned to cmd/ailang/design_quorum.go, internal/policytool/**, internal/config/*_test.go, .claude/skills/design-doc-creator/** or its mirror.

## Registry reuse audit

`ailang pkg search quorum` and `ailang pkg search filesystem` both returned rc1: storage.googleapis.com DNS unavailable. **UNINFORMATIVE UNDER SANDBOX**, not an empty registry result; there were no candidates available to inspect with pkg info/docs.

| Milestone | Decision | Reuse |
|---|---|---|
| M1 | none | Private Go filesystem/config integration cannot depend on an AILANG package. Reuse filepath, config Registry/get, encoding/json, existing exclusive-create writer and public policytool Open/SetBinary/Dispatch. |
| M2 | none | Driver/skill integration is not a registry-package capability. Reuse mc_codex_env_args, taskFor/BuildEnvironment, suite-env.sh, git rev-parse, gen-env-docs and the six-skill sync tool; no new dependency. |

## Proposed milestones

### M1 — Resolution and confinement in Go

**Goal:** implement the four-row writer decision table, fail malformed unsuppressed overrides without writes, and prove real policytool children remain confined.  
**Estimate:** 70 implementation + 230 tests = **300 LOC**, **4 hours**; no dependencies.

| Exact file | Current verified anchor | Planned work |
|---|---|---|
| `internal/config/mission.go` | :13, :35–49, :65, :72 (93 lines) | Env constant, empty-default AreaMission row, raw override getter and raw marker-presence accessor with clear docs. |
| `internal/mission/quorum/artifact.go` | :14, :40–44, :60, :97–105 (152 lines) | Add ResolveArtifactDir and pure validateArtifactDir; resolve before MkdirAll; suppression stderr reports absolute CWD/sentinel and reason without marker contents. |
| `internal/mission/quorum/artifact_dir_test.go` | NEW, beside `quorum_test.go:206,242` | Unit/table arms, raw getter/Registry checks, deterministic IO failures and schema/collision regression. |
| `internal/mission/quorum/artifact_policy_test.go` | NEW; public host seam `policytool.go:99,118,140`, unchanged deny test :147 | External `package quorum_test` avoids import cycle. Real host launches a wrapper around this test executable's child mode; child calls real writer. Count launches before writing and capture stderr. |

Resolve explicit nonempty non-sentinel argument FIRST. For defaults, raw nonempty policy marker suppresses any override before validation. For unmarked defaults, validate the raw override: absolute, non-whitespace, no NUL; literal tilde/relative values fail; spaces in absolute paths survive. Unset/empty override returns the frozen relative sentinel. No directory creation on resolution alone. Default + marker + nonempty override prints the required diagnostic before writing, even for invalid overrides; CWD resolution failure errors. No broad home default, flag-presence or reviewer-result changes.

All new Go tests set HOME=t.TempDir() and isolate CWD in subprocesses; no t.Parallel with process env/CWD/stderr. Tests use fixed stub QuorumResult values and never invoke providers. For unwritable destinations use deterministic ENOTDIR (a file as parent), plus real permission refusal where supported; do not count a skipped permission arm as evidence.

#### M1 test arms

Names below are concrete new tests/subtests; each row identifies an asserted arm; unchanged out-of-scope regression controls are identified explicitly. MUTATION specifies a single-line change to allowed production code in a disposable evaluator copy; restore and prove green after every mutant.

| AC | Concrete test / file | Required assertion | MUTATION (one line) |
|---|---|---|---|
| AC1 | TestResolveArtifactDir_Default/unset and /empty — artifact_dir_test.go | Empty/sentinel args return EXACT relative sentinel; resolve creates nothing. | Replace final `return ArtifactDir, nil` with `return filepath.Join(os.TempDir(), ArtifactDir), nil`. |
| AC1 | TestResolveArtifactDir_ConstantPinned — artifact_dir_test.go; existing TestPolicyTool_QuorumArtifactDirInSync | Literal constant unchanged; existing comparison still runs. | Change ArtifactDir constant to `".ailang/state/quorum"`. |
| AC2 | TestWriteJSONArtifact_OverrideSpaces — artifact_dir_test.go | Actual absolute destination with spaces contains exact serialized bytes; local directory absent. | Replace validated override return with `return ArtifactDir, nil`. |
| AC2 | TestResolveArtifactDir_ExplicitWins/absolute and /relative — artifact_dir_test.go | Distinct explicit paths win even with invalid override and marker set. | Replace explicit-argument guard with `if false {`. |
| AC3 | TestWriteJSONArtifact_InvalidOverride/relative,/tilde,/whitespace — artifact_dir_test.go | Non-nil error, returned path empty, no local/host creation. | Replace validation-error return with `return ArtifactDir, nil`. |
| AC3 | TestValidateArtifactDir_NUL — artifact_dir_test.go | Pure validator rejects an absolute string containing NUL (never put NUL in env). | Delete/disable the NUL-check condition on its one line. |
| AC3 | TestWriteJSONArtifact_DestinationError — artifact_dir_test.go | File-as-parent yields error; no local fallback or artifact. | Replace MkdirAll error return with `return "", nil`. |
| AC4 | TestResolveArtifactDir_RawPolicyMarker/whitespace — artifact_dir_test.go | Marker `" "` still suppresses valid AND invalid override; getter returns raw override. | Change raw accessor to `return strings.TrimSpace(get(EnvAgentPolicy)) != ""`. |
| AC4 | TestPolicyHostQuorumArtifacts/absolute and /invalid — artifact_policy_test.go | Real Open → Dispatch(design_quorum) → SetBinary childEnv/runAilang → real writer; parent override inherited, host policy pinned, writes ONLY sandbox/sentinel, external fixture remains absent. | Replace resolver suppression guard with `if false {`. |
| AC4 | TestPolicyHostQuorumArtifacts/suppression_diagnostic — artifact_policy_test.go | Stderr names absolute sandbox/sentinel and suppression for absolute/invalid overrides; no policy path/content leaked. | Replace suppression fmt.Fprintf call with a comment. |
| AC4 | TestPolicyHostQuorumArtifacts/deny_no_child — artifact_policy_test.go | Policy denies .ailang/**: refusal contains fs_deny_write, child count zero, no local/external artifact. Pair with allowed positive arm above. | Same one-line removal of writer suppression; combined host-chain test must fail its positive confinement assertion. Existing deny implementation is unchanged and already covered; do not mutate forbidden policytool code. |
| AC4 | TestResolveArtifactDir_CWDUnavailable — artifact_dir_test.go | Child deletes only its disposable current directory; suppression path resolution errors before any write/diagnostic with fabricated path. | Replace filepath.Abs error return with `return ArtifactDir, nil`. |
| AC10 | TestQuorumArtifactRegistry/raw_and_default — artifact_dir_test.go | New Registry row has default "", AreaMission and docs; getter preserves leading/trailing spaces. Existing Registry scanner runs. | Remove the single new missionVars row (or trim the raw getter for the raw-value arm). |
| AC10 | TestWriteJSONArtifact_OverrideNoClobber — artifact_dir_test.go + unchanged TestWriteJSONArtifact_NoClobberOnSameSecond | Two same-timestamp writes in override dir have distinct names/bytes; old JSON/salvage rendering preserved. | Replace OpenFile's `O_EXCL` with `O_TRUNC`. |

The deny-only subarm is a regression control over an unchanged out-of-scope gate, not a claim to independently mutation-test policytool's guard. The new confinement proof kills a writer-destination escape while the constant comparison alone cannot.

**M1 executor gates:** G1, G2, G3, G6, G10, G12 below (all base rc0).  
**Risks:** import cycle, raw-marker trimming, accidental salvage overwrite. Mitigate with external host tests, whitespace arm and unchanged stacked salvage tests.

### M2 — Driver export, bash 3.2 helper, both readers and integration

**Goal:** export a validated repo namespace before either launch branch; implement fixed-order fail-closed readers in BOTH skills, shell suite wiring and generated env docs.  
**Estimate:** 160 implementation + 310 tests + 30 docs/generated mirror/changelog = **500 LOC**, **8 hours**, depends on M1.

| Exact file | Current verified anchor | Planned work |
|---|---|---|
| `tools/launchd/lib/quorum-artifacts.sh` | NEW, beside codex-env-args.sh:35–49 | Pure bash 3.2 initialization, path validation, reader-directory and JSON lookup functions; nothing happens on source. |
| `tools/launchd/mission-control.sh` | :71–72, :81–82, :119, :123, :2048–2056, :2246, :2337, :2381, :2383 (2597 lines) | Add small marked source/init block after log(); explicit rc checks, export/log result; marked startup errors before probes or launches. Existing oversized driver gets only wiring; put logic in new companion helper. |
| `tools/launchd/test_quorum_artifacts.sh` | NEW; extraction precedents test_mission_registry_env.sh:20 and test_codex_controller_env.sh:19 | Hermetic named shell arms; extract actual driver block and actual Gate 2 executable block with nonempty/count guards. Shell-only default for bash CI; explicit --writer mode for real Go writer. |
| `internal/mission/quorum/artifact_shell_test.go` | NEW, package quorum_test beside new artifact_policy_test.go | TestQuorumArtifactShellSuite invokes /bin/bash suite --writer with the real writer child wrapper from M1, temp HOME and bounded execution. Missing shell suite/helper/block is fatal; never skip. |
| `internal/mission/dispatch/quorum_env_test.go` | NEW; run.go:199, run_test.go:350 and environment.go:89 | taskFor → actual executor.BuildEnvironment → fake CLI, clean/inherited/policy-injected markers for designer/planner/executor/evaluator and supported provider identities. No production executor edit. |
| `.claude/skills/mission-control/resources/gate-2-pick.md` | :139–145, :199, :215, :224–228 (589 lines) | Executable marked reader block, checked helper source, cwd convention, quoted lookup/reads, same env/cwd quorum invocation; historical local-path clarification. |
| `.agents/skills/mission-control/resources/gate-2-pick.md` | Same verified anchors; cmp base rc0 | Verbatim generated mirror using existing sync tool; test extracted block from BOTH copies. |
| `make/test.mk` | :68, :70–75, :105 | Add one `@$(LAUNCHD_SUITE) tools/launchd/test_quorum_artifacts.sh` recipe; shell-only, no Go toolchain added to bash CI. |
| `docs/docs/reference/env-vars.md` | :304, :316; generated by make/docs.mk:25–26 | Regenerate using existing docs-env generator after Registry row lands; do not hand-edit or alter generator. |
| `changelogs/unreleased/2026-10-14-quorum-artifact-dir.md` | NEW, existing unreleased-fragment convention | Start with ### Fixed; describe opt-in durable repo evidence, marker confinement, fixed-order reader and legacy-import limitation. |

Helper contract: proposed functions `mc_quorum_init`, `mc_quorum_reader_dirs`, `mc_quorum_lookup`. Namespace construction validates safe two-component owner/repo (ASCII letters/digits/underscore/dot/hyphen, each not "." or ".."; no extra slash/traversal); absolute HOME required only for constructing defaults. Deliberate absolute override wins. AILANG maps to HOME/.ailang/state/mission-quorum, World to mission-quorum-world, others to mission-quorum-repos/owner/repo. Missions of one repo share the namespace. Initialization creates no artifact directory.

Active startup with any raw nonempty marker aborts and names effective local confinement; never unset marker. Gate 2 active mode requires export and absolute readable MISSION_DRIVER_ROOT/helper, repeats marker refusal. Attended mode locates git repo root, sources its helper if driver root absent, enters that root before lookup/invocation. Standalone policy reader uses sandbox CWD, never git ascent, and reports suppression. A supplied driver root must be valid in every mode.

With valid export, reader fills a quoted indexed array [export, current repo root/sentinel], logs each consulted absolute directory in order, checks BOTH directories including JSON readability/parse failures, and only then returns match/absence. Missing directory or zero matches is UNREVIEWED; not-a-directory, source/read/permission/parse/config failure is ERROR. Do not change slug, nested result/absence paths or existing verdict remedy. No alternate checkout discovery. Quorum invocation keeps --artifact-dir omitted with the same cwd/env.

Bash 3.2 only: indexed arrays, local variables, case, quoted expansions and read -r. No declare -A, ${v,,}, mapfile or source-from-process-substitution. Tests identify the interpreter as 3.2; /bin/bash on this tree is 3.2.57.

#### M2 test arms

All shell names are functions/labels in tools/launchd/test_quorum_artifacts.sh. TestQuorumArtifactShellSuite in artifact_shell_test.go executes ALL shell arms plus --writer arms with real writer; CI's shell-only mode exercises helper/documented-reader/driver mechanics without fabricating a Go writer proof. The dispatch tests use fake CLIs with no provider/quota/model probes.

| AC | Concrete test arm / file | Required assertion | MUTATION (one line) |
|---|---|---|---|
| AC5 | cross_tree_durable (--writer) | Real writer in fixture A writes exported dir; remove only fixture A, reader B and independent fixture clone C find same path/bytes. | Replace helper exported-reader first entry with local repo/sentinel only. |
| AC6 | repo_namespace_missions | v1/docs/motoko/fleet share AILANG namespace without mkdir at init. | Namespace using MISSION_NAME instead of MISSION_REPO on the selection line. |
| AC6 | repo_namespace_isolation | World and other owner/repo cannot match an AILANG same-slug fixture. | Return AILANG's directory for every repo on one line. |
| AC6 | invalid_repo_and_home | Reject empty components, "."/"..", extra slashes/traversal and nonabsolute HOME when constructing defaults. | Replace repo validation call with `:` (separate HOME mutant removes its absolute check). |
| AC6 | invalid_and_explicit_override | Relative/tilde/whitespace overrides error; absolute-with-spaces wins; no init mkdir. | Replace absolute override validation call with `:`. |
| AC7 | gate2_extracted_active_matrix | BOTH actual marked skill blocks: export absent or empty, driver root absent/relative/unreadable, missing helper, marked active (including whitespace) all ERROR; repo identity may be unset with valid export. | Delete executable-block source failure return/exit on its one line. |
| AC7 | lookup_error_matrix | Directory absence/zero matches → UNREVIEWED; malformed JSON, non-directory, unreadable directory/file → ERROR, including broken git-root resolution. | Replace lookup's error propagation line with `return 1` (reserved absence status). |
| AC7 | extraction_is_nonvacuous | Exactly one executable block from EACH skill, actual helper required, source/mirror hashes equal. | Delete one opening block marker from source skill. |
| AC8 | driver_export_before_both_launches | Extract actual after-profiles/secrets block under fixture HOME; inherited/profile/secrets overrides honored in order, child printenv sees exported path; corrupt config stops BEFORE fake binary/legacy launches; init logs no policy contents. | Delete `export MISSION_QUORUM_ARTIFACT_DIR` line in initializer. |
| AC8 | codex_actual_forwarding | Real mc_codex_env_args + extracted actual legacy call site → fake codex captures .set.MISSION_QUORUM_ARTIFACT_DIR path including spaces. Parent raw marker presence captured separately; no new forwarding rule. | In disposable driver call site replace mc_codex_env_args call with `MC_CODEX_ENV_ARGS=()`. |
| AC8 | contaminated_host_startup | Valid/invalid override + path/whitespace marker: startup and active reader fail, launch count zero, local confinement named. | Disable driver initializer's marker refusal condition. |
| AC8 | standalone_policy_reader | sandbox CWD + whitespace/path marker suppresses absolute/invalid override with required stderr; never ascent or host read. | Disable standalone reader marker branch. |
| AC8 | TestQuorumEnvironmentHandoff/clean,/inherited_marker,/task_policy — internal/mission/dispatch/quorum_env_test.go | Actual taskFor/BuildEnvironment result delivered to fake CLI: override raw bytes retained, inherited marker retained, task policy replaces marker. Test all used roles; print presence only for marker. | Add `config.EnvMissionQuorumArtifactDir: "",` to taskFor ExtraEnv on one line in the disposable copy; it shadows inherited override and must kill the test. |
| AC8 | bash32_and_suite_reach | /bin/bash runs helper/blocks; make dry-run includes EXACT new suite recipe once through LAUNCHD_SUITE. | Delete new make/test.mk recipe line. Bash4-syntax mutant: replace namespace assignment with `repo=${MISSION_REPO,,}`. |
| AC9 | attended_no_exports (--writer) | All mission exports + marker unset, nested fixture invocation enters repo root; real writer there is found byte-exact; unset MISSION_REPO/DRIVER_ROOT permitted. | Replace attended reader repo-root directory with HOME default. |
| AC9 | attended_absent_and_errors | Same extracted blocks: missing directory/zero matches UNREVIEWED; unreadable config/files ERROR; missing repo root ERROR, explicit supplied bad driver root ERROR. | Replace attended helper fallback with unconditional missing-driver-root error. |
| AC11 | cross_mode_fixed_order (--writer) | Attended local writer then active reader with empty exported namespace finds identical local bytes; mission host writer then attended reader WITH SAME EXPORT finds identical host bytes; both absent UNREVIEWED; diagnostics export then local. | Delete local fallback array append line. Separate order mutant swaps first/second array assignment in one line. |
| AC11 | cross_mode_errors_dominate | Both matches fixture plus malformed/unreadable other directory still ERROR; export absence + local match passes; failures in first or second directory never swallowed. | Return REVIEWED immediately on first match instead of after inspecting both. |
| AC10 | generated_docs_and_mirror | Registry row rendered by generator, BOTH blocks equal, no unrelated skill changes; existing six-skill --check passes. | Delete new missionVars Registry row; source-only reader mutation makes sync fail. |
| AC10 | fixture_home_and_observations | Fixture operations restricted to disposable HOME; real-directory manifests are read-only observations before/after. | Replace helper HOME prefix with fixed `"/nonexistent/quorum-mutation-home"`; fixture expected-path assertion must fail without touching real home. |

For environment-handoff mutations, only the taskFor ExtraEnv insertion is authorized in the disposable evaluator copy; do not change internal/executor or codex-env-args.sh.

**M2 executor gates:** G1, G2, G4, G5, G6, G9, G10, G11, G13, G14 below, all base rc0. G1 includes the new no-skip Go shell adapter after implementation, which ensures the new shell suite is actually executed rather than only syntax-checked.

**Risks:** error priority after an early match, bash path splitting, wrong source tree, downstream environment loss, protected mirror writes. Test actual skill/driver extraction and builders. Run existing sync copy on an authority-permitting executor/controller, then inspect that only intended mirror file changed; never widen its six-skill scope.

## Executor gate list and measured baseline

All commands were RUN on current HEAD before this plan. Reproduce each command below with the same disposable-HOME wrapper. This setup is mandatory, not a fixture seeded in real ~/.ailang:

```bash
task_home=$(mktemp -d)
env HOME="$task_home" \
  GOCACHE=/Users/voightkampff/Library/Caches/go-build \
  GOMODCACHE=/Users/voightkampff/go/pkg/mod \
  GOPATH=/Users/voightkampff/go /bin/bash -c '<command from table>'
```

On another runner substitute its already-established cache/module paths; never select real HOME as fixture HOME. Temporary directories may be left for system cleanup under sandbox; no destructive git command is needed. An initial cleanup-trap wrapper was rejected before execution by automatic command review because it contained rm -rf. The planner switched to this non-destructive wrapper; that rejection is not a gate rc or a test result. The planner made no git writes.

| Gate | Exact command inside wrapper | Base rc / evidence |
|---|---|---|
| G1 | `go test ./internal/mission/quorum/ ./internal/policytool/ -count=1` | 0; quorum 0.511s, policytool 0.839s |
| G2 | `go test ./internal/config -run 'TestEveryGetenvNameIsRegistered\|TestNoLiteralGetenvInConfig\|TestGetOrUsesRegistryDefault' -count=1` | 0; 0.659s |
| G3 | `go vet ./internal/mission/quorum/ ./internal/config/` | 0 |
| G4 | `go test ./internal/mission/dispatch/ -count=1` | 0; 1.159s |
| G5 | `/bin/bash tools/launchd/sync-agents-skills.sh --check` | 0; all six skills synchronized |
| G6 | `make check-boundaries` | 0; zero violations |
| G9 | `go run ./tools/gen-env-docs "$HOME/env-vars.md" && cmp docs/docs/reference/env-vars.md "$HOME/env-vars.md"` | 0; 258 variables / 24 areas, byte-identical |
| G10 | `golangci-lint run ./internal/mission/quorum/... ./internal/mission/dispatch/... ./internal/config/...` | 0; 0 issues |
| G11 | `for f in tools/launchd/*.sh tools/launchd/lib/*.sh; do /bin/bash -n "$f" || exit 1; done` | 0; includes newly added files when rerun |
| G12 | `go test -race ./internal/mission/quorum/ -count=1` | 0; 2.355s |
| G13 | `cmp .claude/skills/mission-control/resources/gate-2-pick.md .agents/skills/mission-control/resources/gate-2-pick.md` | 0 |
| G14 | `make -n test-launchd-drivers > "$HOME/launchd-recipe.txt" && test -s "$HOME/launchd-recipe.txt"` | 0; shell suite's own bash32_and_suite_reach arm asserts new recipe inclusion |

Additional measured green controls: G7 `/bin/bash -n tools/launchd/mission-control.sh` rc0; G8 `/bin/bash tools/launchd/test_mission_registry_env.sh` rc0. They are subsumed by G11 and the new extraction arms, not extra milestone acceptance gates.

### Red/unavailable at base — excluded from acceptance gate lists

| Probe actually run with disposable HOME | Base result | Handling |
|---|---|---|
| `/bin/bash tools/launchd/test_quorum_artifacts.sh` | rc127; file does not exist | New test development requirement, NOT a claimed green-baseline gate. After creation, executor runs its /bin/bash default and --writer modes through TestQuorumArtifactShellSuite (G1). Never make absence a passing skip. |
| `go test ./internal/config/ -count=1` | rc1; TestCloudProjectPrecedence panics at cloud_test.go:29, httptest IPv6 bind denied | **UNINFORMATIVE UNDER SANDBOX**; not a code-regression finding. Controller reruns on socket-permitting runner; G2 is the local Registry gate. |
| `ailang pkg search quorum`; `ailang pkg search filesystem` | rc1; storage.googleapis.com DNS unavailable | Reuse audit limitation; not an empty registry or acceptance gate. |
| velocity script (7 days) | rc141; head/pipefail | Planning evidence limitation, not product/test red. |

Do not list full make test, full launchd suite, make test-changed or full make lint as executor gates that this planner has not demonstrated green. The launchd aggregate is also inappropriate for this no-git-write planner run: some existing scripts create/commit scratch git fixtures and probe installed Codex. The scoped Go gates, named adapter and shell-only CI recipe provide focused coverage. Narrow lint was measured and covers changed Go packages. Known fleet ledger D-FLEET-17 (env-vars MDX and unrelated git-exec reds) remains controller-owned: generated-doc byte equality does not establish a docs-build pass. Do not fix those out-of-scope owners here.

## Task list and checkpoints

**Day 1, first 4h (M1):** write unit/raw-marker tests; add Registry/getters/resolver; exercise exported public policytool host with real child writer; run M1 gates and retain actual failures for suppression/validation/diagnostic/collision mutants.

**Day 1, next 4h (M2):** implement pure helper and early checked driver block; extract and test both Gate 2 blocks; prove attended, exported and standalone-policy branches including error priority; wire shell-only recipe plus mandatory Go adapter.

**Day 2, 4h (M2 finish):** test real task/environment handoff and Codex fake launch; regenerate env docs via existing generator; sync intended mirror; write changelog; run M2 gates and independent evaluator mutations; controller collects lane presence observations before activation.

The user requires exactly two milestones. Each can be resumed across bounded executor sessions with test/JSON checkpoints; do not introduce a third milestone to disguise the integration/evaluation effort. Record implementation/gate/mutation evidence in this plan's execution-evidence section and progress in the sprint JSON; never mark passes based on planning baseline results.

## Success metrics and independent evaluation

- Every AC1–11 arm above passes with positive controls, no provider calls and no skipped load-bearing permissions/source/propagation checks.
- Evaluator uses disposable copies and one-line mutants ONLY within allowed files. At minimum remove writer suppression, remove reader fallback and short-circuit on first match; named assertions must go red for the stated reason and return green after restoration. Preserve actual stderr/test failure and rc, not just a claim.
- New helper and suite are bash 3.2; real Go writer used by integration adapter. Both skill blocks tested and byte-identical. Shell-only CI stays independent of Go.
- Real paths are OBSERVATION ONLY: capture sorted basename + NUL + exact bytes SHA256, file count and line count before/after hermetic checks. Any concurrent change is reported as inconclusive with attribution, never “restored” or overwritten. No real-home fixtures, import, mkdir, copy or deletion.
- Planner's read-only observation at 2026-10-14: pin 5 files/377 lines, main 164/8190, host 15/1106, World 2/154. This differs from design's older host count; concurrent operational additions are possible. No recovery or ownership inference.
- Controller observes marker PRESENCE ONLY in actual controller and every used role shell, binary reach and matching effective writer/reader paths. Synthetic tests do not prove live lanes. Marked lane stops; never unset marker to activate. Activation waits for fresh observations and independent evaluation.
- Rollback removes opt-in/export and retains all evidence. Keep explicit host flag workaround until synchronized writer/reader deployment. World adoption and deleted-source recovery remain unverified follow-ups.


### Planner real-directory observations

Read-only snapshots taken twice during planning were identical (no fixture writes here). Manifest is SHA256(sorted basename + NUL + exact bytes).

| Observation-only path | Files / lines, before = after | SHA256, before = after |
|---|---|---|
| `~/.ailang-driver-pin/fleet/.ailang/state/mission-quorum` | 5 / 377 | `7b09a996ce8b90221bace1ab092aa23e6de20667a18f806669b03b0a8f57f787` |
| `~/dev/sunholo-data/ailang/.ailang/state/mission-quorum` | 164 / 8190 | `079f080899fbfc58fb9a1731a80a8a1c3f2105290ac130960ee70f96e3d33d60` |
| `~/.ailang/state/mission-quorum` | 15 / 1106 | `86c70d2f6ce8886e53df80bcbd071ed648fc897694266aaa027db53229187bbb` |
| `~/.ailang/state/mission-quorum-world` | 2 / 154 | `7da76b3a307c6d221d9fec386599891da5c2932980f797beef9b9025f941dc04` |

## Dependencies and open questions

No new product decision is needed for this plan. M2 depends on M1. Controller approval/execution routing and stacked PR #1824 landing are separate from these artifacts. Legacy importer (including report script), CLI preflight/help/flag-presence changes, policytool changes, generic skill rewrites, global environment-policy edits and deployment are OUT.

Nonblocking limitation: explicitly passing the sentinel is indistinguishable from default; exporting the override redirects it. A distinct absolute local path retains precedence. Existing post-write CLI path signal remains.

## Execution evidence (executor/evaluator to populate)

Record milestone, command/arm, actual rc and output; mutation file/one-line delta, failure assertion and restored control; interpreter version; generated-doc/mirror results; fixture HOME and observation hashes; controller live-lane observations and evaluation verdict. Planning baseline rc0 is not implementation acceptance.

