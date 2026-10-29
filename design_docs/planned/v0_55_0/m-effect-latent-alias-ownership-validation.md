# Alias ownership review repair validation

This is a repair of M-EFFECT-LATENT-NESTED-CALLBACKS AC6 and its accepted
`fwd_two` controls. It does not introduce a new effect handoff model. Root owns
independent evaluation; no evaluator verdict is claimed here.

## Reproduction and scope

Base: `9dc6890919aa001deebee4361719456ee087440f` (actual origin/dev).
The unmodified registry module `sunholo/social_dynamics/0.4.0/tasks.ail`
failed at accept→acceptReady (application1070, argument1). Its captured source
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
R2 combined integration and independent evaluation are pending.
