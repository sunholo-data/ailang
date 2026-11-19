# M-FLEET-ZSH-SAFE-MISSION-LIBS

Fleet iteration 57 · 2026-10-22 · ticket `quorum-reader:zsh-nomatch` (stapledon iteration 46).
Status: planned. This plan is its own design document; controller measurements C1–C6 and decisions D1–D6 are the design.

## Goal and evidence

Prevent empty glob expansions from aborting mission helpers sourced into the controller's zsh shell, while preserving bash 3.2 behavior and caller options.
At HEAD `d25d9b8d0`, an export-only quorum match plus an existing empty local directory aborts zsh before the after-call echo; the mirror case also aborts. A missing local directory succeeds. Empty pi extensions similarly print `--no-extensions` then abort.
Controller baseline: quorum suite all PASS; pi suite 14 passed, 0 failed; `make test-launchd-drivers` green. These are premises, not planner-run test results.
Read-only inspection confirmed both glob sites, existing guards, extraction seam, suite wiring and identical Gate-2 skill copies; no other `for … in <glob>` exists in `tools/launchd/lib/*.sh`.

## One milestone: M1_ZSH_SAFE_MISSION_LIBS

Dependencies: none. Estimate: 0.5 day, approximately 220 added/changed LOC (8 helper/comment, 200 tests/controls, 12 changelog); allow a 60–90 minute executor run plus controller validation outside the sandbox.
This is a bounded shell regression fix; effort follows the measured two-site scope and existing test harnesses rather than a speculative repository-wide LOC velocity.
Registry reuse: `none`; repair the existing repository shell functions and tests, with no package-like capability or registry dependency. No `.ail` files or new program examples are needed; executable fixtures demonstrate the behavior.

Executor changes exactly these implementation files:

- `tools/launchd/lib/quorum-artifacts.sh`
- `tools/launchd/lib/pi-ext-args.sh`
- `tools/launchd/test_quorum_artifacts.sh`
- `tools/launchd/test_pi_ext_args.sh`
- `changelogs/unreleased/2026-10-22-zsh-safe-mission-libs.md` (new)

As the first statement of BOTH `mc_quorum_lookup` and `_mc_pi_ext_args`, insert exactly:

```sh
if [ -n "${ZSH_VERSION:-}" ]; then setopt local_options null_glob; fi
```

Update the quorum “Pure bash 3.2” header and pi header with one WHY line: skill snippets source these helpers into the controller's zsh tool shell, where NOMATCH aborts unmatched loops.
Keep every existing existence/file guard, return code and output byte under bash 3.2; the zsh guard is false there. Do not move initialization to source time or alter lookup policy.

## Behavioral arms and observable results

Extend existing harnesses, not grep-only tests. Quorum uses `run_arm` and `--arm`; pi uses its existing ok/bad counters. Register all new arms in default suite execution.
Each arm sources the REAL helper in `/bin/zsh -f -c`, with isolated scratch directories, scrubbed mission/policy exports, and explicit `setopt nomatch; unsetopt null_glob` before the call.
Capture function rc explicitly without `errexit` suppressing the absence case; echo `AFTER_CALL` immediately after capture. Assert child exit status, exact result fields/output, and the marker independently. Capture diagnostics separately.
Assert arrays by length and quoted contents without assuming bash's zero-based indexing in zsh. Fixtures use valid JSON and absolute paths, including a path with spaces.

| Arm | Required observation |
| --- | --- |
| `zsh_quorum_export_only` | Export has `doc-x-2026.json`; local exists empty. `mc_quorum_lookup doc-x`: rc 0, exactly one matching path, `AFTER_CALL`. |
| `zsh_quorum_local_only` | Export exists empty; local holds the artifact. rc 0, exactly one matching path, `AFTER_CALL`. |
| `zsh_quorum_absent` | Both directories exist empty. rc 1, zero matches, `AFTER_CALL`; child exits 0 after assertions. |
| `zsh_quorum_options` | Call in the same zsh process, then `[[ -o nomatch && ! -o null_glob ]]` succeeds; include early absence return. |
| `zsh_gate2_export_only` | Existing `extract` reads the real QUORUM READER block from each skill copy; active-controller fixture invokes `mc_quorum_gate2 doc-x`: rc 0, one match, `AFTER_CALL`. |
| `zsh_pi_empty` | Empty `.pi/extensions`: helper stdout exactly `--no-extensions`, rc 0, `AFTER_CALL`. |
| `zsh_pi_two_ts` | `a.ts`, `b.ts`, README: exact ordered five helper-output lines (`--no-extensions`, `-e`, a path, `-e`, b path), no README, rc 0, `AFTER_CALL`. |
| `zsh_pi_options` | Invoke directly in the caller process (no command substitution around the function); capture stdout via redirection. `[[ -o nomatch && ! -o null_glob ]]` succeeds afterwards. |

Quorum emits `PASS <arm>` only after assertions; pi emits `PASS: <arm>` through `ok`. A missing `/bin/zsh` prints `SKIP (no /bin/zsh): <arm>` on non-Darwin; on Darwin it is a FAIL with nonzero suite exit. No silent skipping.

## Mutation table and evidence

Apply each mutant separately to disposable scratch copies of the real libraries/tests, run the relevant suite/arm, and record mutant, arm, command, rc, diagnostic and killed/survived in the sprint execution notes. Never leave a mutant in the worktree.
Exercise each library independently: a healthy second library must not mask a mutant. Restore the clean candidate between runs and rerun the focused suites afterwards.

| Mutant | Arms/oracle that must detect it |
| --- | --- |
| Drop the entire setopt statement | Quorum export/local/absent and Gate-2 arms; pi empty: unmatched glob abort, missing marker and/or wrong rc. |
| Drop only `local_options` | Quorum/pi options arms must detect leaked NULL_GLOB after a same-process call. |
| Replace `null_glob` with `nullglob` | All relevant arms run; expected equivalent survivor on this rig (see Risks), explicitly record survived rather than inventing a kill. |
| Guard on BASH_VERSION instead of ZSH_VERSION, or invert the ZSH_VERSION condition | Quorum empty-directory and pi empty arms must fail under zsh; bash gates also catch inappropriate setopt execution. |
| Remove the marker check | Oracle mutation: feed the assertion path otherwise-correct captured output lacking `AFTER_CALL`; an intact checker rejects it, removing its check must make this negative control fail. |

Add behavioral negative controls to each existing test file for marker validation: capture the actual successful child result, remove only its marker, assert rejection. Also remove the child's marker echo as a producer mutant; positive marker arms must reject it.
Require every non-equivalent mutant killed and a recorded result for every row; do not make source-text grep assertions the oracle.

## Execution and validation

1. Before edits run the three gates below at base. Any red AT BASE is a finding to report, not something to work around or repair outside this scope.
2. Write the zsh regression arms and observe the empty-directory failures; implement the exact two guards and header explanations, then run focused gates.
3. Run mutation controls, restore clean sources, add the changelog fragment, and run final gates in this order:

```sh
/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_quorum_artifacts.sh
/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_pi_ext_args.sh
make test-launchd-drivers
make check-changelog
git diff --check
```

Expected: all commands exit 0, quorum named arms PASS, pi named arms PASS with 0 failed, no NOMATCH abort. Existing bash arms remain green on macOS bash 3.2.57 and Linux bash 5; existing make wiring reaches zsh arms on macos-latest without CI edits.
Socket-touching arms yielding in-sandbox results are **UNINFORMATIVE UNDER SANDBOX**; record the limitation. The controller reruns everything outside the sandbox; sandbox output cannot replace that gate or justify a workaround.
The changelog opens with `### Fixed — …`, explains both failures and scoped option restoration, and links this plan per `changelogs/unreleased/README.md`.

## Risks and measured corrections

- First-party planner probe `/bin/zsh -f -c 'setopt nullglob; [[ -o null_glob ]] && print NULLGLOB_ALIAS_ACCEPTED'` printed `NULLGLOB_ALIAS_ACCEPTED`: D3's spelling mutant is equivalent, so expecting its death would be false evidence. Keep D1's requested spelling.
- `/bin/zsh -f -c 'setopt nomatch; unsetopt null_glob; f() { setopt null_glob; }; f; [[ -o nomatch ]] && print NOMATCH_STILL_ON_WITHOUT_LOCAL_OPTIONS; [[ -o null_glob ]] && print NULL_GLOB_LEAKED'` printed both markers. D2's NOMATCH-only check cannot kill the local_options mutant; additionally assert NULL_GLOB is off.
- A deleted assertion cannot fail an otherwise correct positive run. The missing-marker negative controls above make D3's test-oracle mutation observable.
- Keep permission/error arms intact and report base failures; do not expand scope to compensate for sandbox restrictions.

## Out of scope

Changing Gate-2 snippet text, adding `bash -c` wrappers in skills, any other library, driver behavior, quorum policy, `.github/workflows`, and any files outside fleet authority. Planning creates only this plan and its sprint JSON; the controller commits and dispatches execution.
