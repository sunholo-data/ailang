# Explicit-empty callback contract execution evidence

Mark authorized “Execute explicit-empty sprint” on 2026-10-29. Root is the final independent evaluator. This report records execution, not a self-evaluation or release.

## M1 — closed-empty conversion

Checkpoint `e4bee03c9c88210360dbe2b7967d57c650708e0e`.
Failing-first tests observed an open/implicit Core row for explicit emptiness, missing legacy concrete metadata, and acceptance of an IO value in an empty callback slot from an IO caller. After conversion changes, all three reject/identity discriminators pass; matching pure, omitted open, concrete nonempty and row-polymorphic controls retain behavior.

Required `make test-changed`: five branch-touched packages plus full core loop pass. CLI 206.498s, builtins 16.267s, elaborate 5.018s, pipeline 21.236s, types 5.153s. Lint 0, size gate green. Independent read-only M1 review found no blocker (`explicit-empty-M1-independent-review.md` in the session evidence directory).

## M2 — presence, identity and cache

Failing-first AST String, iface conversion/restoration, normalized interface identity, internal type string and alias-digest tests failed for the intended lost-contract reasons. The fixed focused checks pass across AST/iface/types/pipeline. Additional tests cover tuple/list/Option/record aliases, multi-argument callback arrows, pure controls and IO rejection, valid closed-empty substitution/JSON and CoreTI-DAG persistence.

The change preserves canonical formatter behavior and distinguishes FuncType empty presence in AST JSON. Existing iface conversion moved to a companion file to keep builder below 800 lines; other conversion cases remain intact. Normalized exported callback/return, constructor and alias signatures retain `!{}`. Isolated constructor-only and export-only edits change public HashProjection/SignatureSet. Alias-only edits change normalized JSON and the existing internal alias dependency digest; public HashProjection/SignatureSet intentionally remain unchanged because their existing policy excludes alias bodies. Direct/transitive warm alias tests separately cover cache invalidation. No schema, cache format, hash policy or epoch changes. #1771 owns reserved epoch v8; this sprint reserves none.

### Same-binary dependency edits

Automated pipeline fixtures seed an accepted omitted-arrow program, prove a warm artifact was decoded, edit only a direct or transitive dependency's callback alias to explicit empty, and require IO rejection. Reversing the dependency edit restores acceptance. Exported callback signatures have the same forward/reverse verdict check.

The existing row-guard baseline already rejects IO in an omitted constructor callback slot, so that fixture cannot honestly seed an IO acceptance cache. Its cache discriminator instead uses a pure argument accepted on both versions, verifies the dependency iface digest and unchanged importer's cache key BOTH change after the annotation edit, then separately requires the cached explicitly-empty constructor to reject an IO value. This preserves existing omitted-constructor semantics while proving signature identity is covered.

### Real old-compiler acceptance cache

Old compiler source `752cebf3cf09202c01ec74d6ce1172a9c39b2294`, built from a fresh archive with that exact commit identity, accepts `empty_slot_io.ail`. It populated 16 cache files; a second old run accepted and changed only manifest bookkeeping, leaving artifact files unchanged. The actual fixed binary then checked identical source against the SAME cache and rejected IO, exactly as its no-cache run did. The old cached acceptance was not reused.

The companion cache-proof JSON preserves source-base identity, old main key, artifact count and all four command outcomes. Fixed binary was built from M1 plus this approved M2 delta; its dirty-build fingerprint participates in the existing compiler identity. Reproduce with the archived old source and current tree; never pretend fixed-only cold/warm runs prove cross-compiler invalidation.

### Gates

- Focused AST/iface/type identity tests: pass.
- Same-binary alias/export/constructor warm-edit and valid CoreTI-DAG tests: pass.
- Nested container/multi-argument acceptance/rejection matrix: pass.
- Actual four-pure-field scratch motoko `ext_fixture.ail`: passes with the fixed compiler. This early check is NOT the required final 60/9 proof or delivery.
- Lint 0, sizes and boundaries: pass.
- `make freeze-stdlib` and all 49-interface verification: pass, zero golden diff.
- Full changed/core gate: seven branch packages plus core pass. Later milestones are recorded as they complete.

Raw logs under `/tmp/ailang-backlog-20261029/`: `explicit-empty-M1-{red,green,test-changed,lint}.log`, `explicit-empty-M2-{red,green,warm,containers,cross-compiler-cache,actual-fixture,test-changed,lint,freeze,stdlib}.log`. Original premise matrices and failed four-line hypothesis remain unchanged in the design's evidence JSON. No additional consumer patch has been delivered at this checkpoint.

## M3 — engine behavior, example and immutable teaching

The actual old compiler executes `EMPTY_CALLBACK_SIDE_EFFECT` on evaluator and
bytecode for the IO-in-empty-slot program (explicit-empty-M3-old-engine.log).
Current CLI fixtures reject direct, record and imported variants before that
marker even with IO caps and an IO main; a pure factory storing a mixed fully
concrete callback record executes 42 without invoking the stored IO renderer.
The imported CLI fixture runs in its real module directory with matching stdlib,
using the existing bounded subprocess helper; the initial wrong-CWD setup error
was a test harness issue and was corrected before acceptance.

Runnable example checks, prints 42 and passes its answer() inline test. Contracts
are deliberately skipped for higher-order storage/field invocation outside SMT;
pure callback/function and main IO annotations remain explicit.

New prompt v0.16.11 inherits v0.16.10 and adds only the callback contract distinction.
Frozen source hashes remain byte-identical: .9=6197fe5f4bf1c112737c550e08d557eea5c92db9b85839efaac8d46accd28be1;
.10=a9435c79d78002c3c0b51e36bff06b5f918f7e3a90198dd2301e493c0ff9e001.
Prompt source/embedded/docs mirrors and source/embedded registries match. Registry
freeze and 120 taught command routes pass. Feature code blocks are the actual
checked/run example; inherited prompt blocks were not rewritten or optimized.

Additional imported alias control requested by root: pinned social_dynamics 0.4.0
/tasks.ail source SHA256 261faeb15fa083aed19e7f94dbe5af5e9b21273134c41ef2f90147fdaf92797a reproduces the same accept→acceptReady app 1070 argument 1
missing-callback-row invariant on actual dev 9dc and current fixed M2 compiler,
using no cache and matching stdlib. It is recorded as a known baseline failure,
not repaired or counted as a pass (explicit-empty-M2-social-dynamics-control.log).

M3 initial full changed-package run passed every test except the registry's
one-mutable-version invariant: the version-creation helper left both .10 and .11
mutable. The existing `ailang prompt freeze --migrate` froze only the superseded
version, preserving its bytes/hash; no test relaxation. Failed/affected prompt
tests pass 4.907s and the full core loop that initial failure aborted completed
successfully. Lint 0, file sizes, all examples/manifest, registry freeze and taught
command routes pass. The original red log remains; focused repair/core logs are
`explicit-empty-M3-prompt-recheck.log` / `explicit-empty-M3-core-recheck.log`.

## M4 — compatibility and delivery complete; independent acceptance pending

Actual final compiler source checkpoint `9ab45c98e071c9c90f1c365ff82cca00d74ef151`
is normally rebased onto actual dev `9dc6890919aa001deebee4361719456ee087440f`,
including the merged nested-callback and IFC changes. An immutable compiler copy
and matching stdlib drive every consumer check below.

AST-aware inventory visits function-type nodes, distinguishing absent effect
annotations from explicit emptiness and excluding function declarations. It scans
826 repository/package/parser files and finds one explicit-empty function type,
the new runnable example; seven parse failures are recorded, so absence is not
asserted for those files. A second 210-file world/consumer inventory finds exactly
the four proposed fixture fields, with one pre-existing invalid fixture parse.

Paired no-cache comparison uses matched pre-amendment compiler
`61290d130b437d615d350e79ee2e268f8320e113` and current compiler, each with matching
stdlib. The same 529 in-repository files produce 445 passes on both arms, zero
status flips and zero first-diagnostic changes. The new example separately changes
from the expected pure-factory IO rejection to acceptance. External pins and root
counts: packages `e998928e3dbab4661b47ae94e3e74c593f37e121` 43/45 on both;
parser `0931946850e23a6897df783731ab74b0b1d43b88` 0/2 on both;
world `df5a0671dadd567ece879fe89afcc018c2deaa23` 1/1 on both;
pristine motoko `835c738b8ca50ed38f36c117058847c816db32b8` 21/25 on both.
All have zero status/diagnostic changes. Existing parser broad-root and motoko
root timeouts, plus existing invalid/error roots, remain failures, not passes.

The exact four-line scratch consumer patch passes actual `gmake check_core`:
60 core files, 9 extension boots and all runtime prerequisites, using a new isolated
cache. The separate no-cache 60-file sweep records 57 passes and three 180-second
deadlines under two workers; only those three were retried serially and all passed (127.07s, 79.44s, 67.97s).
Fresh no-cache result is therefore 60/60 with initial deadlines retained.
Final `make test-changed` passes all seven changed packages and the full core loop
(CLI 580.695s, pipeline 125.760s). Lint 0, sizes, boundaries, architecture closure,
changelog, prompt registry/120 taught routes and examples (247 modules) pass. Final
freeze/regeneration plus verification passes all 49 interfaces with zero golden diff.

Raw evidence: `explicit-empty-M4-{inventory,consumer-inventory,corpus-inrepo-summary,
external-corpus-summary,new-example-before-after}.json`,
`explicit-empty-M4-consumer-{check-core.log,core.json,core-summary.json}`;
original source matrices and failed four-line-before-compiler hypothesis are retained.


The conditional F4 migration was delivered only after that fresh proof. Bot account,
pinned base and absence of competing fork PRs were rechecked. Fork PR
[#5](https://github.com/sunholo-voight-kampff/motoko_agent/pull/5), exact head
`8dc0ad84f9a9dc029d624468278def981f807e07`, passed root's independent four-line
review and merged normally at `083ef43726981099a0bad4fedda18fc72105d6fa`.
Merged tree `067e8990d698af8385431647f6c5cfff99a7e63a` equals the tested head.
Docker pins the full merge SHA and MOTOKO.md §15 records exact provenance; upstream
and live rig remain unchanged. No image build/deployment or release is inferred.

Fresh paired checks under fork CI's old v0.52.5 reproduce the identical skills
`HeaderEntry.boot.cost_rates` 2-versus-4-field error on base af1d and migrated head.
This inherited failure is disclosed in the PR; broader/pending fork CI is not
claimed green. Relevant actual-compiler gates and independent diff review passed,
with no admin override or check-policy change.

The companion compatibility JSON banks compact source/corpus/inventory/deadline,
retry/delivery and gate results. Original premise/cache reports remain intact.
M4 implementation is complete; its final acceptance remains pending root's
independent sprint-evaluator and exact-head CI. No self-evaluation is asserted.
