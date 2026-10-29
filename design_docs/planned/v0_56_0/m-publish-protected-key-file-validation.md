# M-PUBLISH-PROTECTED-KEY-FILE implementation validation

Implementation/local gates complete, unreleased. Independent evaluation and
actual exact-head CI are pending. This is executor evidence, not an evaluation
score. The design and companion remain planned until independent acceptance.

## Approval and integrated source

The attended user approved D1–D7 at design commit 7d000322b and explicitly
said "exeute all three sprints" with the presented dependency sequencing.
Original approved artifacts were banked at c9698d88e. No approval was inferred
from a plan-only merge. Root independently accepted #2013 PASS95/all 27 CI,
then normally merged 96d69f05 at 20:53:24UTC. M0 normally integrated that source
at a9aa13313 before product edits; its Go/AIL trees were identical to the accepted
baseline. The actual integration satisfied the conditional execution approval.

Milestones were banked separately: M1 be066a2b1, M2 87fa35f90, M3 f7ff92f20,
M4 f9539a602. Final M5 changes help/dispatch/reference/guide/fragment plus this
handoff metadata; all final Go bytes were included in the terminal local gates.
No compiler/effects/cache/authority-policy or other credential-consumer changes.
Root confirmed no unrelated newer source merge is required for this accepted
base. Other agents' work and main checkout edits remain untouched.

## Acceptance map

| AC | Committed evidence | Qualification |
|---|---|---|
| AC1 source selection | `TestPublishCredential_Selection`, `NoDefaultOrUnusedFile`, `TestPublishAuthCLI_HelpAndSelection`, `ZeroUploadAndUnusedFile` | No default; explicit-empty, ambiguity, missing and unsafe errors; zero counted HTTP before package work |
| AC2 protection/format | `internal/pkg/publish_keyfile*_test.go` | Real0400/0600/mode/symlink/nonregular/FIFO/limit/handle-identity checks; foreign owner is synthetic metadata, no chown/root; endless reader measures16385byte bound |
| AC3 authority | `TestPublishKeyFile_AuthorityBeforePath`, `TestPublishCredential_FrozenLegacyAndAuthority`, `TestPublishAuthCLI_EnvWarningAndAuthority`, `TestPublishUpload_NoAnonymousOrConfinedRequest` | Missing paths and competing env source still return typed/canonical refusal; source gates precede open/read/request; authority table unchanged |
| AC4 child secrecy | `TestPublishChild_RawEnvironmentAndFrozenUpload`, `testdata/publish_env_probe/main.go` | Actual smoke executes compiled Go `os.LookupEnv`; env arm literal PRESENT vs file arm literal ABSENT. No Env allowlist simulation. Counted validator receives header, not multipart secret; output has no selected key |
| AC5 lifetime/no unused access | Same child test plus file/selector/CLI tests | Smoke replaces dummy key path before upload; original header remains. Same-handle rename test proves read identity; source closes the handle before checks. Actual dry-run succeeds with missing file and unused env; no credential warning/read |
| AC6 legacy/errors | Child env arm, CLI warning test, upload reflection test | Legacy header and exactly one warning; controlled400 preserves cause and redacts exact selected-key echo; missing/authority cases issue zero request; unpublish/key management unchanged |
| AC7 platforms | Supported/unsupported build companions and platform-aware tests | Native macOS full positive reader proof. Windows actually runs named unsupported-file refusal plus env/help/dry-run, awaiting real CI. No mode-as-ACL claim or whole-test Windows skip |
| AC8 integration/docs | Retained `TestPkgSmokeStaging_QualityAndDryRun`, full affected packages/core and structures; publishing guide/help/generated reference/fragment | Accepted registry-source+asset/D2/Notes/author-restoration code retained; no full local suite; broad exact-head CI and independent evaluator required |

The child proof deliberately receives HTTP400 from loopback. It reaches the real
upload boundary but never succeeds, so no package-message/cascade path runs.
Private author content and lock absence are checked, as are private collaboration
and coordinator database absence. All keys/files/services are synthetic; no real
credentials, provider requests, live registry publication or coordinator actions.

## Terminal local gates

Logs are retained under session `/tmp/ailang-backlog-20261029/1900-execution/`.
Native Darwin arm64, cached Go 1.26.10; an allowlisted outer environment, private
HOME/USERPROFILE/home, offline toolchain/modules, GOFLAGS=-p=1 and no metadata
prevents inherited registry/cloud/trace credentials even during package TestMain.
The temporary runner is an evidence harness, not a new product script.

| Log | Result |
|---|---|
| `M1-repaired-harness-green.log`, `M1-lint-retry.log`, `M1-structure.log` | Reader focused PASS25.555s, lint0, boundary/home PASS |
| `M2-green.log`, `M2-lint.log` | Reader/format/selector PASS pkg17.804s/cmd0.768s, lint0 |
| `M3-native-green.log`, `M3-lint.log` | CLI/header/error/retained staging PASS27.296s, lint0 |
| `M4-raw-presence.log`, `M4-green.log`, `M4-lint.log` | Actual raw-child/path-replacement/header plus zero-request/dry-run controls PASS; final combined12.179s, lint0 |
| `final-test-changed.log` | PASS: cmd/ailang 478.983s + internal/pkg 26.049s, then every configured core package |
| `final-build.log`, `final-lint.log` | Build PASS; whole configured lint0 |
| `final-structure.log` | Boundaries, home isolation,800line sizes, changelog, CLI reference, fmt PASS |
| `docs-cli.log` | Reference regenerated from actual dispatch source |

The final build is stamped f9539a602-dirty because M5 help/docs were being
verified before committing; its tested product source is identical to the final
commit. A new PR's exact-head CI will supply clean-commit broad/platform proof.
No full make test was run and no existing timeout or gate policy was changed.

## Preserved failures and controls

- `M1-api-red.log` and `M2-api-red.log` are missing-API compile RED, not exploit
  demonstrations. `M3-cli-red.log` is actual built-CLI functional RED: absent
  flag/help and missing legacy inheritance warning before wiring.
- `M1-green.log` records the initial fixture's3second child timeout: the package
  TestMain rebuilt the CLI before m.Run. The fixture was corrected to directly
  bound the FIFO reader with cleanup. No product timeout was relaxed.
- `M1-lint.log` was an external linter lock; the normal retry after release is
  separately retained. `M3-green.log` failed only sandbox cache/listen permission;
  the same isolated test with native permission passed in M3-native-green.log.
- `M4-real-cli.log` found a wrong new expected local-error prefix with zero HTTP;
  changing that test expectation required no production fix. Active env-present
  control, actual file-variable absence and original-header-after-replacement
  controls are retained; none rely only on smoke.Passed or hidden env policy.

## Retrospective and handoff

All milestones ran sequentially in one isolated execution worktree. The estimate
was 650 changed product/test/user-doc lines; actual is 1017 (240 production,
717 test/helper,60 user-doc), counting additions+deletions against accepted 96d
and excluding approval/design/plan/state/validation metadata. The variance is
bounded security/platform/process proof, not additional product semantics.
The actual affected CLI package took478.983seconds; final configured core also
ran. CI waiting and independent review are separate from implementation work.

No executor score, archive, release, merge or issue closure is claimed. Root owns
independent evaluation and normal merge once actual exact-head CI is terminal.
