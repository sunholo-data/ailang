# Alias ownership review repair validation

**Archive status:** R1/R2 accepted at95/100, [PR2008](https://github.com/sunholo-data/ailang/pull/2008) merged749045717c484299b3103008023bf27a826df844. Implemented/unreleased, target0.56.0. The executor checkpoint statements below are retained as history; [independent evaluation](m-effect-latent-alias-ownership-evaluation.md) is authoritative for acceptance.

This is a repair of M-EFFECT-LATENT-NESTED-CALLBACKS AC6 and its accepted
`fwd_two` controls. It does not introduce a new effect handoff model. Root owns
independent evaluation; no evaluator verdict is claimed here.

## Reproduction and scope

Base: `9dc6890919aa001deebee4361719456ee087440f` (actual origin/dev).
The unmodified registry module `sunholo/social_dynamics/0.4.0/tasks.ail`
failed at accept→acceptReady (application 1070, argument 1). Its captured source
SHA256 is `261faeb15fa083aed19e7f94dbe5af5e9b21273134c41ef2f90147fdaf92797a`.

Reduced local/imported aliases with one callback failed with an unresolved
annotation tail. Two callbacks produced a missing callback-row invariant even
when the receiving function returned 0. Inline records passed. The new
publication unit test also failed: both ownership sets were empty for a named
alias, and solver tail renaming left a nil callback row.

The environment retains annotations as named aliases. Caller charging expanded
these aliases through the existing checker resolver, while implicit ownership
passed a nil expander to the same data walker. This repair passes the existing
resolver into ownership discovery and includes every implicit-owned tail in
general publication ownership, which re-anchors solver-renamed tails.

Only `internal/types/typechecker_app_effects.go` changes production behavior.
Declared rows, unknown tails, checked union conflicts, callback masks, conservative
parameter-wide charging and the explicit-empty contract remain unchanged.

## Evidence

Raw artifacts live under `/tmp/ailang-backlog-20261029/`:

- `alias-ownership-red.log`: unit and local/imported one/two-field failures.
- `alias-ownership-green-first.log`: repaired original failing matrix passes.
- `alias-ownership-focused.log`: alias and latent control suite, including generic
  applied aliases, passes; existing distinct declared-tail conflict is retained.
- `alias-ownership-cli.log`: initial real CLI tree/strict VM controls pass.
- `alias-ownership-test-changed.log`: full pipeline and types packages pass.
  The full CLI package fails only the two new invoked-FS strict positive cases:
  `std/fs.readFile` is evaluator-only (Phase 2E), with no strict interop bridge.
  This preserved failure is a verification-contract mismatch; all other CLI
  tests pass. No VM implementation change is part of this repair.
- `alias-ownership-cli-final.log`: all final local/imported, both-field IO/FS
  cases pass (23.766s). Strict mode is required for missing-effect rejection,
  executed IO and unused-FS declared acceptance. Invoked FS passes tree and
  normal bytecode mode, asserting VM dispatch and no top-level evaluator
  fallback. Strict executed-FS parity is explicitly not claimed.
- `alias-ownership-original-social.log`: original unmodified cached tasks module
  checks clean with no cache and the repair binary's matching stdlib.

Paired examples/std corpus check: 528 files, 444 accepted and 84 rejected in each
arm; zero status flips, zero diagnostic changes, zero internal invariants. The
source set is the base9dc corpus; no files were added to or removed from it.
The aborted core gate is run separately; the already-green full pipeline/types
and unrelated CLI tests are not repeated after this test-only correction. The first structural pass
had clean lint, boundaries and file sizes, then caught an invalid changelog
heading (`##` instead of `###`); the fragment was corrected and that gate rerun.

## Required combined follow-up

Source acceptance is separate from combined release evidence. Normal-integrate
the actual merged #2005 explicit-empty implementation and delivered consumer
pin, then recheck the original social_dynamics module and actual consumer
core 60/60 plus extensions 9/9, with corpus comparison. No scratch consumer edit,
previous standalone run, or self-scored evaluation replaces this final gate.

Final R1 gates: separately invoked `make test-core` PASS, final `make lint` PASS
with 0 issues, boundaries/file sizes PASS, corrected changelog PASS, sprint JSON
validation PASS, and `git diff --check` PASS. R1 source/test milestone is complete;
R2 completed below; independent follow-up evaluation remains pending.

## R2 actual integration and consumer acceptance

R1 commit `34009a084a75d0dc13657672f5d1e17582211ee4` normally merges actual
#2005 dev `80793dbcf42423027be4bab1a0088da54e2323d8` at clean source checkpoint
`15afcc12cec28900afe3e312a9356884f8b07b06`. Repository `make build` stamps that
full commit. Later handoff commits change alias status/evidence metadata only.

Exact delivered consumer merge `083ef43726981099a0bad4fedda18fc72105d6fa`, tree
`067e8990d698af8385431647f6c5cfff99a7e63a`, was archived into a scratch directory
without source edits. With this compiler, matching stdlib and a new empty isolated
cache, actual `gmake check_core` exits 0: 60 core modules pass, nine extensions boot
and 204 default runtime guard assertions print OK. The in-pane Herdr leg retains
its normal conditional skip because this is outside a Herdr pane. No live rig,
upstream source, gate policy, model/compile flags or consumer annotations changed.

Combined focused types/pipeline/CLI controls pass (0.416s/8.796s/28.258s), including
alias ownership and generic forwarding, original nested/declared-tail controls,
explicit-empty callback/cache contracts, row guard, effect ceilings and both
engines. Lint 0, boundaries, file sizes and changelog gates pass. Full local suites
are not repeated; exact new PR CI supplies the broad gate for independent review.

The original unchanged social_dynamics tasks source (SHA above) checks clean
NoCache with the combined compiler. An immutable build of actual dev 80793 still
fails at the same application 1070, argument 1, proving this is the alias repair.
Paired actual-dev 80793 versus combined examples/std corpus: 530 files, 446 pass and
84 reject in each arm, zero status flips, zero diagnostic changes, zero invariants.

Raw `alias-ownership-R2-*` logs preserve build, focused, structure, consumer-check-core,
original-social, original-social-base and corpus evidence in the session directory.
Compact results/provenance are committed in m-effect-latent-alias-ownership-evidence.json.
R1/R2 implementation is complete. Independent evaluation remains pending; the
original #1963 PASS 93 on 86d3a5f8 is historical, unchanged and not a score for this fix.

## Independent acceptance

All9 follow-up criteria met. Exactd5f final26 checks completed19success/7skipped, no retries; Sonar100% at unchanged80 threshold. Independent95/100 includes the85line test and stale integration-field deductions. Original source statuses, strict-FS failed attempts and qualified normal-VM interop remain preserved. Original parent PASS93 is unchanged. No deployment or version bump.
