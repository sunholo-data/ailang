# M-SMT-WELL-SORTED-PREFLIGHT: resolve operators and symbols by elaborated type, and check the SMT before Z3 sees it

**Status**: M0 design/evidence human-ratified for planning only; sprint draft and execution unapproved; M1–M4 deferred.
**Target**: next unreleased release after v0.55.0; original v0.54.19 target is historical, not a release request.
**Priority**: P0 for M0 (#1960/#1961 false-VERIFIED); #1950/#1953 and the wider preflight roadmap remain later scope.
**Estimated**: historical full roadmap about 6 engineering days; M0 estimate must be refreshed only after design approval.
**Dependencies**: existing SMT callee/parametric gates; preserve actual dev effect/empty/IFC and #602 test fixes.
**Tracks now**: #1960/#1961 only. **Future:** #1953/#1950; #1959 is fixed (ef7bdc8e0) and regression-only.
**Created**: 2026-10-26. Original audit at 7d29031ea / Z3 4.16.0; current refresh base aca5a6ead, 2026-10-29.
**Quorum**: two scoped runs remain **BLOCKED** (GLM/Kimi reject, Gemini pass in both). The later actual human
ruling below ratifies corrected evidence and M0-F1/F4/Z for planning only; it does not rewrite either tool verdict
or authorize further disclosure/review/execution. Full-roadmap M3 banking/F2/F5 remain deferred.

---

**Human disposition (2026-10-29, observed 14:48:58 UTC):** user said “approve the three designs”, explicitly ratifying
this M0 evidence amendment and M0-F1/F4/Z for **planning only**. Approval record: local
`three-design-approvals.json`, design commit 88ca9e1e974150fa783d19b15b8a93be1682e2f7. Observation time is the
assistant's receipt record, not an independently supplied user send time. Both external BLOCKED records remain
unchanged. No third provider review, plan approval or execution is inferred. The current sprint plan/JSON are
uncommitted drafts pending review and explicit execute instruction.

<!-- M0-REVIEW-BEGIN -->
## Current decision packet: M0 only (2026-10-29)

**State:** two external scoped quorum runs remain BLOCKED. The user has now ratified the corrected evidence and
M0-F1/F4/Z for planning only at the observed time above; no further disclosure/review, implementation, parked-work
recovery or merge is authorized. A new plan/JSON draft is not an execution instruction. This packet is the authoritative current M0 proposal;
§§1–16 retain the original wider audit/roadmap and its historical measurements. M1–M4 are **not** part of the request.

**Scope:** #1960 and #1961 only. Correct the Int operator and contract-only callee encodings that can report false
VERIFIED. Reuse only the sound S1 expansion and selected S2 plumbing after current-head validation; **replace** the parked
universal-axiom implementation. Nothing is recovered “as-is”; do not duplicate the topic under a new design. #1950/#1953,
`encoder_gap`, new ERROR/SKIP rules, ai-check exit policy, caller-precondition proof obligations, width/BitVec and
preflight belong to later decisions. Operator-held #1790's purity/admission card is untouched.

**Source anchors:** proposal base `aca5a6eadeedb5b7b59feab6f2e0137ccf3dc239` (dev when this proposal was refreshed, #2011 merged; not a latest-head claim);
parked `p0/smt-unsound-encodings` at `a5f81711904452b432f850ba37f686fde79c895a`,
merge-base `3996302744503d2a9ed06198c3190dc9d61911df` (literal command/output in evidence), plan `74cc3dba4`,
S1 `981a7aee9`, S2 `5410d0b2f`. Recovery, if later approved, uses a **new isolated branch**. The parked worktree,
its dirty sprint JSON/plan/fmt log and its completion claims must remain intact and are not accepted gate evidence.

### Historical authority and present decisions

The original primary human message, 2026-10-26T14:59:31.188Z, says:
“can we fix the p0 issues via subagents with design doccs, sprint plans and excutes?”
The selected primary record is retained only in the local audit artifact
`smt-m0-readonly/historical-provenance.json`; private session records/identifiers are not published here. This is
historical context only: this proposal grants no current authority from that private record, and human inspection
of the present semantics/gates does not require the private session. Fresh scoped human choices were subsequently ruled above for planning only.
It authorizes a broad P0 workflow historically. The more specific “required sound choice (F1)” wording appears in
agent delegation and then the parked plan, **not** in a separately named human F1 ruling found in that session.
The parked plan's attribution must not be reused as proof of an explicit named human decision. This proposal asks
for fresh, bounded choices honestly; it does not amend the historical plan or its unvalidated completion metadata.

| Current M0 choice | Proposed contract | Authority / state |
|---|---|---|
| M0-F1 (parent F1) | Match runtime truncating Int `/` and `%`, including negative operands; preserve the existing math-Int width disclaimer | human-approved for planning only, 2026-10-29 |
| M0-F4 (bounded parent F4) | Same callee and same arguments share a UF value; only actual call instances contribute requires-guarded postcondition facts within their lexical/path scope; no global contract `forall` | human-approved scoped proof contract for planning only; private representation resolved during separately approved execution |
| M0-Z | Preserve existing zero-domain/partial-correctness policy below; no new nonzero assumptions, error-freedom promise or special skip policy | human-ratified unchanged zero policy for planning only; different semantics still outside scope |
| M0-Q | Scoped independent quorum on this packet before planning; then fresh design approval → sprint plan → explicit execute → independent evaluation | two external BLOCKED records preserved; human evidence ratification authorizes planning, not a fabricated external PASS |

- [x] M0-F1 human-approved for the current M0 design/planning scope.
- [x] M0-F4 human-approved, including partial-callee and lexical/path controls.
- [x] M0-Z human-ratified with no zero-domain policy widening.
- [ ] M0-Q external PROCEED is not achieved: both tool records stay BLOCKED. Separate explicit human evidence ratification is the planning disposition; this is not an unresolved request to re-ask M0-F1/F4/Z.

Parent F2/F5 remain unruled and out of scope; parent F3 remains a later architecture decision. No M0 approval checks
those later freeze boxes. There is no new diagnostic/reason code or schema/cache epoch in this packet.

### M0 evidence and the rejected universal shortcut

[Compact reproducible evidence](m-smt-well-sorted-preflight-m0-evidence.json) stores checked fixture source, raw
compiler/runtime outputs, Z3 scripts/results, source anchors and primary-message provenance. The executable used was
an archived 80793 compiler with its pinned stdlib. Its actual executions remain **807-only execution evidence**:
no aca5, parked or fixed compiler was built/run for this design. `verify_source_equivalence` now captures the exact
80793→aca5 named-path command, returncode 0 and empty stdout/stderr over `internal/smt`, `internal/verification`,
`std`, `cmd/ailang/verify.go`, `cmd/ailang/ai_check.go`. `control_front_half_equivalence` and the CLI-prefix comparison
also capture equality over elaborate/eval/VM/bytecode and the specified run/verify front halves. These are pinned
source comparisons, **not whole-compiler equivalence or a fresh current-head run**. Version/temp-module warnings
are retained. The broader `internal/pipeline` diff is nonempty and qualified below.

On the executed 807 archive the #1960 and #1961 sources both `ailang check` successfully. Its `verify` reports VERIFIED `parity` (exit 0) and
VERIFIED `gap` (aggregate exit 1 because the distinct known bounded-unrolling `total` row is a counterexample).
Always inspect rows: aggregate nonzero does not mean the false gap proof is fixed.

The parked S1 formula passed small direct-Z3 div/rem controls for 12 nonzero-divisor sign pairs and a nested
parallel-let control. These are formula checks, not execution of every compiler emitter. Actual `/0` and `%0`
controls type-check and then terminate with runtime RT001 / exit 1.

The round-two local repair **executes runtime arithmetic**, independently of the SMT transcription: two pure
parameterized fixtures return the same 12 sign-pair rows for `/` and `%` on the interpreter and strict VM
(`run --bytecode --strict-bytecode`), all equal to the truncating table. Pairs include `(±7,±2)`, zero dividends,
small quotients/remainders and `(9223372036854775807,-1)` within the existing width model. For example `-7/2=-3`,
`7/-2=-3`, `-7%2=-1`, `7%-2=1`. Each source also checks successfully. Runtime source at aca5 uses Go integer `/`
and `%` after zero checks: `internal/builtins/math_arithmetic.go:22–33`,
`internal/eval/builtins_arithmetic.go:39–60`, `internal/vm/vm_arith.go:29–38` and
`internal/vm/builtins_math.go:86–97`. The literal `runtime_source_equivalence` record captures equality of these files between 807 and aca5.
The binary SHA256, actual version output, 807 source tree, matching std tree and all 54 actual std-file hashes are
banked; every std file matches the pinned 807 source. Trace/export environment variables were removed for these
pure local calls, avoiding the separately held ambient-export issue. Raw stdout/stderr and commands are retained.

The original `int_divmod` fixture from the parked commit is now embedded whole, including `halfDown`, with its
source hash, check 0 and full verbose JSON verify output. Old rows are parity/halfDown/halfNonNeg/parityBounded/
remSign VERIFIED, halfTowardZero COUNTEREXAMPLE; aggregate exit 1 comes from that last row and is not a fix for
halfDown. Fixed-branch CLI proof remains an execution acceptance gate; none is claimed by this design repair.


The parked S2's global axiom is **not equivalent** to per-call instantiation. Consider this actual checked source:

```ailang
export pure func partial(n: int) -> int
ensures { n != 0 && result == 0 } {
  if n == 0 then partial(n) else 0
}

export pure func wrong(a: int) -> int
requires { a != 0 }
ensures { result == 1 } {
  partial(a)
}
```

`partial` satisfies its postcondition on all terminating executions; zero diverges. `main = wrong(1)` actually
returns 0 and runtime contract checking fails `wrong`'s ensures. A global
`forall n. n != 0 && f__spec(n) == 0` is nevertheless UNSAT at the **uncalled n=0**, so it could prove the false caller.
At the actual nonzero call, instantiating only `a != 0 && f__spec(a) == 0` leaves the false ensures refutation SAT.
A second raw solver control shows that adding `requires =>` does not cure an inconsistent/uninhabited spec at an
uncalled input satisfying requires. E-matching patterns guide search; they do not restrict a universal axiom's
logical scope. A false-requires guard control remains SAT, and same/distinct-call identity controls behave correctly.

**Evidence limit:** this is checked AILANG/runtime plus a direct translation of inspected parked SMT, **not a parked
branch CLI reproduction**. Execution acceptance below requires old/new compiler CLI and mutation evidence. Assuming
every recursive contract admits a total return witness would be a new proof/trust semantics, rejected by this
partial-correctness proposal. A bad spec may make the actually selected divergent path vacuous; it must not poison
other input values or unselected branches.

### Two terminal quorum runs and this UNSENT evidence amendment

| Frozen run | Packet bytes | GLM | Kimi | Gemini | Synthesis | Registry-imputed USD |
|---|---:|---|---|---|---|---:|
| round1 |37,638|reject|reject|pass|BLOCKED|0.2283714|
| round2 |111,807|reject|reject|pass|BLOCKED|0.536629|

All 3 reviewers were present in each run; no absences, reserves or outside-pool fallback. Root's technical draft PASS
was recorded separately and did not override either synthesis. Both exact packets, complete reviewer objections,
proposed fixes, token/cost records and raw CLI diagnostics are preserved in evidence. These are registry-price
calculations; Ollama Cloud values are flat-rate imputations, not independently measured invoices. Round 2 **was sent**
under fresh exact-packet user consent and remains BLOCKED. This later amendment has **not** been sent.

Round 1's missing runtime sign matrix, parked merge-base/diff, halfDown source, help and packet identity were repaired
before round2. GLM's “reuse as-is” reading did not match the explicit original rejection of universal axioms; the
clarification preserves selective S1/plumbing reuse and replaces `forall`, without changing the proposed semantics.

Round 2's GLM/Kimi objections identified remaining evidence-binding gaps: the executed 807→aca5 named-path comparison
and installed e863 identity/diff were asserted rather than included as literal records, and normalizer/front-half
provenance was incomplete. The author ran some earlier checks but failed to bank those exact rows in the reviewed
packet. The new local supplement now supplies them verbatim, plus the **nonempty** cache diff and source excerpts.
This is an evidence-only amendment, not a claim that the rejected packet already contained that proof.

Gemini's round2 concern remains an **unvalidated eventual acceptance obligation**, quoted verbatim:

> During sprint planning and execution, verify that the paired helper-condition define-fun declarations in S2 properly thread all lexically captured variables (or restrict eligibility to closed top-level helpers) so that deduplicating declarations per resolved helper symbol does not drop or misbind outer scope variables across diamond call graphs.

Its exact proposed fix is also retained:

> In the sprint plan for S2, specify that paired helper-condition define-fun declarations take the exact same parameter list (including any lifted closure/environment parameters) as their corresponding value define-fun declarations.

Under the existing same-callee/same-arguments contract, the proposed acceptance below must establish parameter and
environment preservation. **No new closed-helper restriction is selected**, and no implementation validation is
claimed. The complete raw Gemini review remains alongside the GLM/Kimi rejections, not replaced by this response.

The attended two-run limit is exhausted. Design-doc-creator's “re-quorum-ONCE” reference is a case study, not a
standalone universal cap; mission-control's pick-time rule and its conditional narrow-refinement carve-out are
quoted in the supplement. This is not an invoked mission iteration and no carve-out is self-applied. A human must
explicitly decide the evidence/gate disposition. After independent evidence-completeness checking, root may ask
for direct human ratification of these evidence-only corrections and approval of scoped M0-F1/F4/Z, with both
BLOCKED verdicts and limitations visible. **No third external review is presumed**. Any further disclosure/review
would need its own explicit decision despite the attended cap. M0-Q is not marked satisfied; no design approval,
planning or execution follows automatically from these added records.

### Proposed S1: one truncating Int encoding

Reuse the centralized S1 expansion and all-emitter inventory in `981a7aee9`:

```smt2
; n and d are hygienic, parallel-let-bound encoded operands.
q = (ite (= (>= n 0) (>= d 0))
         (div (abs n) (abs d)) (- (div (abs n) (abs d))))
r = (- n (* d q))
```

All Int builtin/curried/intrinsic/binop/dictionary division and remainder producers must reach this semantic helper.
Current aca5 inventory: builtin `div_Int`/`mod_Int` at `internal/smt/types.go:269,271`, direct and curried App
routes at `codegen_apps.go:15,46,146`, Intrinsic at `codegen_operators.go:11–28`, BinOp at `:32–54`, DictApp `div`
at `:75–95`. The DictApp method map has no `mod` entry; this inventory does not authorize adding unsupported methods.
The non-test operator search/source snippets are recorded in evidence; re-enumerate against the actual execution base.

Inline parallel lets are acceptable in place of the original preamble functions: both operands are encoded once and
nested divisions cannot capture the outer operands. Do not add a second competing operator table. Type-directed
Float/unknown-dictionary work remains M1; do not claim this makes Float division supported. Fixed-width overflow,
including MinInt/-1, remains the original math-Int model limitation.

**Zero policy (M0-Z):** runtime Int division and modulo by zero report RT001 and return no value. Retain the existing
verification scope: undefined zero cases are represented by the emitted SMT totalization and do not establish a
runtime return value, termination or absence of RT001. No new `d != 0` assumption is injected globally or into a
caller's requires, and no zero-specific VERIFIED/SKIP/ERROR policy is added. In particular, do not change the accepted
input domain silently to get a desired model. The existing `%0` totalization can reduce to the numerator; that is
not a claim about a successful runtime return. Changing the totalization can change solver models on those
non-returning inputs; retain and explain those rows rather than labeling them concrete runtime violations. Tests
must distinguish zero-only paths, nonzero returning paths and
unselected zero branches. If review demands nonzero-domain obligations, a runtime-error proof, a skip, or fresh
arbitrary zero results instead, **freeze and request that different contract explicitly**.

### Proposed S2: functional identity, finite lexical call facts

Reuse S2's UF call-term/arity plumbing, requires fail-weak behavior and counterexample model cleanup, but replace
`EncodeCalleeContract`'s global universal assertion. One UF represents each already-resolved contract-only callee;
a call at arguments `a` yields `f__spec(a)`. Existing name-resolution eligibility is retained, not widened to new
builtins or a module-identity overhaul. Repeated identical calls share that term; distinct arguments do not collapse.

The encoder must carry **both a value term and its finite call-condition expression** through the relevant Core
structure. For one actual invocation the fact is `requires(a) => ensures(a, result := f__spec(a))`. It is an assumed
callee specification under the existing contract-as-spec trust boundary, **not** proof that the callee body or its
requires has been verified. Caller requires remains an independent assumption and is not discharged as an obligation.

- Encode/evaluate argument conditions first. Substitute actual arguments capture-free into the callee predicates.
  Unknown/unencodable requires means no postcondition fact for that invocation; unencodable ensures contributes no
  such ensures fact. Unexpected verifier faults retain existing loud behavior. Arity/type failures use existing
  sentinels and their current classifications, without `encoder_gap` or ai-check policy changes.
- For `let x = v in b`, the value is `let x = v in value(b)` and the condition is
  `condition(v) && (let x = value(v) in condition(b))`, using hygienic binders. No raw `x` is asserted at top level.
- For `if c then t else e`, require `condition(c)` and guard each branch condition by its selected Boolean value.
  The unselected branch cannot contribute contradictory facts. Match/scrutinee/guard bindings must follow the
  existing encoder's first-match and lexical rules, including nested patterns/shadowing.
  **Short-circuit control:** `normalizeBinaryOp` → `normalizeShortCircuit`
  (`internal/elaborate/expr_simple.go:120,205`) already lowers `&&`/`||` to Core `If`, keeping RHS lets lazy;
  `TestShortCircuit_AndDesugarsToIf`/`OrDesugarsToIf` assert that shape. `elaborateContracts` in `file_funcs.go`
  reaches the same normalizer; `patterns.go:normalizeArm` normalizes guards inside pattern scope, and
  `OpLowerer.Lower`/`lowerExpr` preserve/lower contract predicates and match guards. Therefore call facts must never
  come from an unselected short-circuit RHS, an unmatched arm's guard, or a guard after an earlier selected arm.
  Add paired selected/unselected callee controls for `&&`/`||` in body and contract predicates, plus sequential
  match guards, to M0-A5; false claims remain refutable and selected true controls verify. No new Boolean semantics.
- A resolved non-recursive helper containing contract-callee calls must propagate the same condition to **its actual
  caller**. A paired helper-condition `define-fun` over the helper parameters is acceptable, with that predicate
  instantiated only at real calls; a definition alone is not an asserted universal contract. Never assert its body
  for every parameter value, leave its formal names free, or silently drop those nested facts.
  Emit each value/condition helper declaration once per already-resolved helper symbol and refer to it at calls;
  do not recursively clone its condition body along every path through a diamond dependency graph. The condition
  declaration must preserve the value helper's exact parameter/environment list, including any already-lifted
  closure variables; declaration reuse cannot merge distinct captured environments. This is a future acceptance
  obligation under the same-callee/same-arguments semantics, not an adopted closed-helper restriction or validated
  implementation. A diamond
  control must prove repeated calls reuse declarations and retain their selected/path facts. This is a declaration-
  sharing invariant, not a new asymptotic runtime or solver-performance guarantee.
- Calls in body and caller contract predicates use the same machinery and current admission/sort gates. Keep their
  positive/negative controls separate so refutation is not weakened by an unconstrained or globally inconsistent
  postcondition-only path. This is not authorization to implement the adjacent proof-vs-refutation design.

The exact private data structure can be settled by the agent after these invariants are approved. A minimal scoped
condition-carrying adapter is preferred to rewriting all SMT syntax producers or changing public evidence schemas.
If a required existing Core form cannot preserve these invariants, use its **existing** unsupported path and document
the regression for review; do not guess at bindings, invent constraints or quietly change global outcome policy.

### M0 acceptance and compatibility boundary

The following must be demonstrated with actual old/current and fixed compiler binaries plus matching stdlib in a
new isolated execution branch, not only raw Z3 or unvalidated parked logs:

- [ ] **M0-A1:** both #1960/#1961 sources check on both builds; old parity/halfDown/gap rows falsely VERIFY, new rows
  are counterexamples (negative x for parity/halfDown, `a >= 0` for gap). Preserve the independent `total` bounded-
  unrolling row and aggregate exit evidence rather than hiding it.
- [ ] **M0-A2:** all Int producer shapes and sign combinations agree with truncation for nonzero divisors, **and
  actual interpreter/strict-VM CLI results for the same sign-pair fixtures equal the truncating table row-for-row**.
  Any executed runtime mismatch freezes M0-F1 for an amended contract, rather than silently choosing new semantics.
  Nested divisions and source/generated-name shadowing retain bindings. Nonnegative half, bounded parity, toward-zero and
  remainder-sign true controls remain VERIFIED. Z3-availability skips use existing platform conventions.
- [ ] **M0-A3:** same-argument UF identity verifies; distinct-call false equality is refutable; parameter-dependent
  postconditions use the actual arguments; actual satisfying/violating requires and unencodable-requires controls
  demonstrate guarded/no-fact behavior without asserting a missing requires as true.
- [ ] **M0-A4:** the checked `partial`/`wrong` fixture above is refutable through the fixed **CLI** with actual
  returning nonzero witness; normal runtime returns 0 and runtime contracts fail. A scratch mutation restoring the
  parked universal axiom must falsely verify that caller; retain the raw SAT/UNSAT discriminator separately.
- [ ] **M0-A5:** nested let/shadowed/match-bound arguments, conditional/uninvoked divergent callees and a non-recursive
  helper forwarding into the contract-only callee preserve lexical facts. False contracts stay refutable; true
  selected-path controls verify. Include calls in body/requires/ensures, not only top-level direct calls, plus a
  diamond-helper control asserting unique declarations and preserved call facts at each selected invocation. Verify
  exact value/condition-helper parameter lists, including lifted environment variables, with distinct captured-
  environment controls. Gemini's concern is not considered implemented or proven by this design amendment.
- [ ] **M0-A6:** `/0` and `%0` still check and fail runtime RT001; fixed verification follows M0-Z, with no nonzero
  assumption or new skip/error policy. Nonzero and unselected-zero controls stay consistent. No claim of totality,
  no proof of runtime-error freedom, no new width guarantee.
- [ ] **M0-A7:** per-fix mutations on disposable copies fail the meaningful row/control tests. Before/after examples,
  std and the same pinned package corpus explain every changed verdict. No new false VERIFIED or encoder ERROR is
  accepted; inherited ERROR rows are recorded without implementing M1–M4 or changing reason/exit policy.
- [ ] **M0-A8:** focused changed/core tests, lint, structure/boundaries, changelog and verify examples pass; required
  exact-head CI and independent sprint evaluation pass. Preserve current #602 fake-solver warm-up and existing
  effect/explicit-empty/IFCv2-v8 changes. No full executor `make test` rerun is mandated by this design refresh.

**M0 axiom check:** A1/A2/A3/A4/A6/A8/A9/A12 =0 (runtime, authority, schemas and syntax unchanged); A5 +1 (removes
false proofs), A7 +1 (honest per-function result), A10 +1 (one call-fact mechanism), A11 0 (existing outcome policy).
Net +3, no claimed hard-axiom violation; this is the author's proposal, **not** an independent verdict.

**Conflict surface:** same expression positions contain nested Int arithmetic, locals/pattern binders, conditionals,
helper calls and user functions in contract predicates. Parse/typechecking syntax does not change. Verification
changes intentionally for proofs resting on wrong Int operators or collapsed callee values; runtime programs and
admission/purity rules do not. Existing `examples/runnable/contracts/{finance,cross_module_functions,
nullary_constant_verify,result_error_guard,empty_list_sort_verify,string_verify}.ail` remain controls (existence checked
at aca5; their historical statuses are not fresh execution evidence). The literal named-path comparison is
`git diff 3996302744503d2a9ed06198c3190dc9d61911df aca5a6eadeedb5b7b59feab6f2e0137ccf3dc239 -- internal/smt internal/verification std cmd/ailang/verify.go cmd/ailang/ai_check.go`.
Over **SMT/verification/verify CLI**, only #602's test warm-up changes; SMT production remains byte-identical from
that base. The broader named-path diff additionally includes `std/VERSION`, `std/ai.ail`, `std/array.ail`,
`std/map.ail`, `std/package.ail` and `std/set.ail` (129 insertions/6 deletions across std). These landed changes
must be preserved; “only #602” is **not** a claim that std stayed unchanged from the parked base. The complete
literal 12,529-byte diff is in evidence. Separately, 80793→aca5 is empty over all those named paths (including std),
which records named-path equality only; actual executions remain 807-only and do not validate an unbuilt aca5 or
parked/fixed compiler. The broader `internal/pipeline` comparison is **nonempty**: four cache production files
(`cache_artifacts.go`, `cache_dep_closure.go`, `cache_key.go`, `cache_store.go`) plus seven tests. The complete
28,663-byte patch, name list and returncode 1 are embedded in the supplement. The actual harness set
`AILANG_NO_CACHE=1`; the config/phase source handling is captured and identical 807→aca5. This qualifies the
recorded controls but does not establish whole-compiler behavioral equivalence. #1790 is an operator-held scope
instruction from root, not a freshly queried live status: leave its callee/codegen/admission semantics untouched
and coordinate before any later execution branch integration.

### Scoped quorum gate before planning

**Current disposition:** the actual human ruling above permits this planning draft despite retained external
BLOCKED records. The following packet/history describes the pre-ruling inspection; it supplies no new execution
authority. Do not reopen the already-ratified F1/F4/Z or require another provider call at executor handoff.

CLI/help and source were inspected at aca5: `cmd/ailang/design_quorum.go` declares no milestone/section/scope flag;
`readDoc` in `cmd/ailang/design_review.go` accepts a file path or stdin (`-`/no positional path).
`--seats` defaults 3. `--author` records author/vendor attribution for seating; it is not a controller identity.

The UNSENT human-decision packet includes **the complete canonical Markdown and evidence JSON**, both complete
reviewer results and needed literal supplement commands/outputs/source excerpts. Review instructions remain strictly
**M0 current decision section only** (between the canonical M0-REVIEW markers); the full historical M1–M4 material
is context, not a request to approve/review those deferred policies or supply evidence for their old audit counts.
No required M0 evidence may depend on an unavailable external scratch file: provenance paths are navigation only.

An in-packet identity record names the committed canonical/evidence blobs/SHA256, literal marker byte offsets and
current-M0 section SHA256. The complete payload is concatenated once; JSON marker-like strings are never reparsed.
The final packet's own SHA256 belongs only in the external manifest to avoid circular hashing. Installed CLI full
version/hash/help and the literal e863→aca5 quorum-source diff are now included. **Its -dirty version stamp is not
an independently attested clean source-to-binary build**; source reads, actual help and two recorded runs establish
only their stated observations. No new binary build or actual aca5/fixed execution is claimed.

For this author use **`--author codex`**, an honest product/vendor attribution: this runtime identifies the author
as Codex/GPT-6-based but does not expose an exact API model alias. `fs.String` accepts the raw author label;
`VendorOf` in `internal/mission/quorum/seating.go:50` matches the substring `codex` to `openai`, and
`SelectReviewers` benches that vendor without looking the author up in the model registry. No model ID is invented.
Bare `--author openai` is **not** recognized by the current matcher; it would warn and bench nobody. The existing
reviewers still use configured real model IDs. A human decision is required before **any further invocation**.
If separately authorized, it must retain the explicit three-destination pool and cap below; prior exact-packet
consents do not authorize this UNSENT amendment or override the exhausted attended limit:

```text
ailang design-quorum /tmp/m-smt-well-sorted-preflight-M0-review.md \
  --reviewers gemini-4-argon,oc-glm-5-3,oc-kimi-k3 \
  --author codex --seats 3 --max-cost-usd 0.30 --json
```

File input is preferable to stdin for a stable named artifact; stdin is supported but is labelled `(stdin)` by the
CLI. There is no native `--scope M0`. Reviewer prompt/input must explicitly say: **judge only this M0 contract;
F2/F5 and M1–M4 remain unapproved, not omissions to fill during execution**. Any objection to M0 must be resolved;
a scoped pass must never be reported as full-design approval. Record absent reviewers and any degraded seating
honestly. Later controller flags may only record the actual reviewing controller's in-session judgment; this author
does not set them, impersonate that identity or pre-fill a pass. A reviewer synthesis never supplies human approval.

Both external rounds remain terminal BLOCKED and are preserved unchanged. The later human ruling above now
authorizes this planning draft and ratifies the corrected M0 evidence/choices. It authorizes no provider call or
implementation. The concrete sprint plan/state JSON remain uncommitted drafts, and a separate explicit execute
instruction is still required. An unchecked external-PROCEED status is not a request to re-ask ratified M0 choices.
<!-- M0-REVIEW-END -->

---

## Historical full roadmap (M1–M4 remain unapproved)

The remainder retains the 2026-10-26 audit and wider proposal. “Today”, corpus counts and source line numbers below
refer to that original snapshot unless a current packet/verification-log entry states otherwise. The current packet
supersedes the old M0 wording and approval implications only; it does not ratify the wider roadmap.

## 0. Summary

1. The SMT encoder translates Core to SMT-LIB text without consulting the elaborated type at three choice points:
   operators, literals and unresolved names. Ill-sorted or undeclared SMT is only discovered when Z3 rejects it.
2. Every one of the **32 verifier ERRORs** across 57 packages, and all three issue repros, comes from one of those
   three points. examples/ and std/ have **0 ERRORs**, and no CI gate checks that this stays true.
3. **No Z3 error is ever reported as VERIFIED or as a counterexample.** `Solve` classifies on the first output line,
   and Z3 prints `(error …)` before the verdict. Of the 31 Z3 errors in the corpus, 25 were followed by a bogus `sat`
   and 3 by `unsat`, and all 31 were reported as ERROR. No test pins this guard.
4. Two well-sorted encodings are **unsound** (they report VERIFIED for a false contract), and a pre-flight cannot see
   them: Int `/` and `%` use SMT-LIB Euclidean `div`/`mod`, and a contract-as-spec callee collapses every call site
   into one constant.
5. The fix:
   - M0 fixes S1 and S2.
   - M1 resolves each operator and literal from its elaborated type.
   - M2 routes every name through one resolver.
   - M3 adds a pre-flight that sort-checks the emitted script and turns any residual failure into
     SKIPPED `encoder_gap`.
   - M4 adds a CI gate (0 ERRORs in examples/ and std/, plus pinned fixtures).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Verifier-only. The emitted script for a function is already deterministic (6 runs of relativity/blackbody.ail gave byte-identical per-function SMT; only report order varies, see Non-Goals) |
| A2: Replayability | 0 | No trace surface |
| A3: Effect Legibility | 0 | No effect change |
| A4: Explicit Authority | 0 | No authority change |
| A5: Bounded Verification | +1 | VERIFIED regains its meaning. M0 removes two false proofs. M3 turns every residual encoder gap into a bounded, named SKIP instead of a solver crash |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | +1 | An agent gets `encoder_gap: float division …` instead of a raw Z3 error that includes a bogus `sat` model (the #1959 reporter read that model as a possible false proof) |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | The pre-flight is a linear pass over the script; Z3 runs no longer than before |
| A10: Composability | +1 | One operator table and one symbol resolver, used by body, contract, callee and HOF-specialised encodings alike, instead of the current three operator paths and two name paths |
| A11: Structured Failure | +1 | `verifier_error` comes to mean "verifier bug". Unencodable programs get a typed reason code |
| A12: System Boundary | 0 | — |

**Historical author assessment: net +4.** This is not a quorum/controller verdict or current approval.
Hard violations claimed by the original proposal (−1 on A1/A3/A4/A7): none.

---

## 1. Problem Statement

`ailang verify` and `ai-check` promise three honest outcomes:

- VERIFIED: the contract is proven.
- COUNTEREXAMPLE: the contract is refuted, with a model.
- SKIPPED: the verifier cannot encode the function, and the reason is named.

ERROR is meant to mean the verifier itself failed. In practice an *encodable-looking* program often produces an ERROR
whose text is raw Z3 output, including a `sat` model that refutes nothing. `ai-check` exits 1 on any ERROR
(`cmd/ailang/ai_check.go:208`), so **correct code fails the agent's check**:
`ai-check cmd/ailang/testdata/verify_preflight/float_div.ail` gives `rc=1`, `errors: 2` for `x / 2.0`.

### 1.1 The three tracked issues, reproduced on origin/dev 7d29031ea

**#1953: string ordering in a body is encoded as Int comparison.** Reproduced, and it is wider than reported: `<` and
contract-side `>=` fail the same way.

```
$ ailang verify --verbose strcmp.ail          # body: "a" <= c && c <= "z"
  ! ERROR lower
    Z3 error (exit exit status 1): (error "line 5 column 51: Sort mismatch at argument #1 for function
      (declare-fun <= (Int Int) Bool) supplied sort is String")
(error "line 7 column 31: unknown constant result")
sat
(  (define-fun $p_c () String "") )
      (define-const result Bool (let (($tmp2 (<= "a" $p_c))) (ite $tmp2 (<= $p_c "z") false)))
```

`strlt.ail` (`"a" < c`) gives the same error. `pick` in the fixture (`requires { c >= "a" } ensures { result >= "a" }`)
gives the same error from the contract side.

**#1950: a builtin behind a helper is a Z3 ERROR, while a direct call is SKIPPED.** Reproduced exactly.

```
  ! ERROR takeFile
    Z3 error: (error "line 4 column 81: unknown constant sha256Hex (String) ")
(error "line 6 column 60: unknown constant shortHash (String Int) ")
(error "line 8 column 35: unknown constant result")
sat
      (define-fun shortHash ((text String) (n Int)) String (let (($tmp1 (sha256Hex text))) (str.substr $tmp1 0 (- n 0))))
  ⚠ SKIPPED direct
    Reason: … calls user function "sha256Hex" that is not SMT-encodable in this context [unsupported_encoding]
```

**#1959: an imported ADT constructor is undeclared. It reproduces on v0.54.17 and is already fixed on v0.54.18 and
dev.** The reporter's exact package (sunholo/audio 0.2.0, ailang-packages merge 5e35e62, extracted read-only with
`git archive`) on a v0.54.17 build from `git archive v0.54.17`:

```
9 verified, 0 counterexample, 39 skipped, 1 errors, 33 uncontracted
  ! ERROR checkSize   (error "line 13 column 435: unknown constant ResourceLimit (String) ") … unknown constant result … sat
```

The same package on v0.54.18 (cd33d05) and on dev 7d29031ea gives `10 verified, 0 counterexample, 39 skipped,
0 errors`, with `checkSize` VERIFIED. The commit that fixed it is ef7bdc8e0 *fix(smt): preserve typed parametric ADTs
and lexical bindings* (M-SMT-PARAMETRIC-ADT-DECLARATIONS, written from the same inbox message
`inbox_1792745981583_2823c9cc`).

The minimal package repro (`import ./safety (AudioError, ResourceLimit)`) on dev gives:

- `checked`: VERIFIED.
- `wrong` (`Ok(x) => x > 0`): VIOLATION with `$p_n = 0`. This is the negative control.
- `unitR` (`Result[(), AudioError]`): SKIPPED, "unit type cannot be encoded". It was an encoding ERROR on v0.54.17.

The **mechanism** that produced #1959 is still in the tree, though. An unresolved name falls through to "treat it as a
constructor" and is emitted raw (§3, RC2). ef7bdc8e0 closed that one leak site by declaring the type. This doc closes
the class.

### 1.2 Why this keeps coming back

Each of these commits closed one way for an undeclared symbol or an ill-sorted term to reach Z3:

| Commit | What it closed |
|--------|----------------|
| 94e2a5d27 / efd251f16 (M-SMT-CALLEE-SORT-GATE, v0.30) | Parametric callee signatures, and unresolved user-function calls in the *caller* body |
| 32ee90ed9 (#689) | Empty-list sort in *body* cons position |
| 9b6584612 (M-SMT-NULLARY-CALLEE) | Zero-argument callees |
| ef7bdc8e0 (M-SMT-PARAMETRIC-ADT) | Imported ADT declarations and Err payload sort |

Each fix was correct, and each left the adjacent site open: callee bodies, contracts, literals, float operators.
That is the incremental special-casing pattern. Nothing checks the *output* of the encoder, so every new producer
(HOF specialiser, unroller, xmod contract, interpolation) can leak in its own way.

---

## 2. Audit: every ERROR reachable today

### 2.1 How ERROR is produced (`internal/smt`)

`Solve` is called from exactly one place, `verify.go:467`. There are four ways to reach `Status: "error"`:

| Site | Trigger |
|------|---------|
| `verify.go:453-462` | Go-side encoder error that is neither `ErrUnresolvableTypes` nor `ErrUnsupportedConstruct` (or any `ErrUnsupportedConstruct` when `UnsupportedConstructIsError`, which is how ai-check runs) |
| `verify.go:468-476` | `Solve` returned a Go error (temp file I/O) |
| `solver.go:213-218` → `verify.go:495` | Z3 exit ≠ 0 with output not starting `unsat`/`sat`/`unknown` and not containing `timeout`: the **"Z3 error (exit …)"** path |
| `solver.go:221-223` | Unexpected Z3 output with exit 0 |

The ai-check exit contract is `cmd/ailang/ai_check.go:208` (`verify.Errors > 0 → 1`). Eval banking reads the same
count: `internal/eval_harness/verify.go:143,156`.

### 2.2 Corpora and method

The build was origin/dev 7d29031ea (`ailang-od`).

- **examples/ + std/** of the worktree: every `.ail` containing `ensures` or `requires` (71 files). Each was run
  through `ailang verify --json --timeout 5s <file>` from the repo root.
- **ailang-packages**: `origin/main` 8e6bbaf, exported read-only with `git archive`, 58 package dirs. Each was run
  through `ailang verify --package . --json --timeout 5s`. 57 were verified. `motoko-ext-typefix-agent` has an invalid
  manifest (`exported module "main" must start with package name`).
- Each ERROR was bucketed by its **first** Z3/encoder error. The location was mapped back to the emitted SMT through
  `verify --verbose --json` per file.

Totals:

| Corpus | verified | counterexample | skipped | **error** | uncontracted |
|--------|---------:|---------------:|--------:|----------:|-------------:|
| examples/ + std/ (68 compiled; 3 are deliberately non-compiling) | 110 | 23 | 8 | **0** | 282 |
| ailang-packages (57 packages) | 170 | 0 | 454 | **32** | 855 |

The 23 example counterexamples are intended:

- `broken*` and `*_demo` functions, plus the inbox-injection attack demos.
- `basic.ail` `safeDivide`/`increment` are genuine contract bugs in the example.
- Six are the bounded-unrolling false counterexamples already ruled on in
  `m-verify-bounded-unrolling-false-counterexample.md`.

### 2.3 The ERROR bucket table

| # | First failure (bucket) | Root cause | examples+std | packages | Package functions (first error) | Minimal repro (dev status) |
|---|---|---|---:|---:|---|---|
| A1 | String `<`/`<=`/`>`/`>=` emitted as Int `<`/`<=` | RC1 | 0 | 0 | — | #1953 `strcmp.ail`, `strlt.ail`; fixture `string_order.ail` (4× ERROR) |
| A2 | Float `/` emitted as Int `div` (`invalid constant definition, sort mismatch`) | RC1 | 0 | **7** | relativity `cosSeen`, `weakDeflection`, `weakDeflection2`, `tidalRadial`, `tidalTransverse`, `tidalRadialOrbit`; celestial `starIlluminanceAt` | `float_div.ail` (2× ERROR) |
| A3 | Float literal rendered with `%g` exponent (`1e+06`, `1.416784e+32`) → `unknown constant e+06` / `')' expected` | RC1 | 0 | **5** | relativity `surfaceBrightnessRatio`, `luminousEfficacy`, `pointFluxRatio`; celestial `satelliteElementsAt`; content-library `nextRoll` | `float_literal.ail` (2× ERROR) |
| A4 | `[]` in a *contract* defaults to `(Seq Int)` | RC1 | 0 | **1** | crew-core `empty` | `empty_list_contract.ail` (ERROR) |
| B1 | Unencodable builtin inside an inlined **callee** body emitted raw (`exp`, `sqrt`, `floor`, `pow`; `sha256Hex`) | RC2 | 0 | **14** | relativity `photopicRadiance`, `tanh`, `sinhc`, `coastAt`, `betaOf`, `gammaOf`, `accelerate`, `doppler`, `dopplerApparent`, `cmbSeenTemperature`, `cmbSeenTemperatureApparent`; celestial `period`, `wrapPi`, `discIlluminance` | #1950 `helper_builtin.ail` (`takeFile` ERROR, `direct` SKIPPED) |
| B2 | Stdlib name in scope without an import (`length`, `startsWith`) emitted raw | RC2 | 0 | **3** | billing-proposals `statusLabel`, `createProposal`, `markAwaitingPayment` | `unimported_stdlib.ail` (ERROR); the same body with `import std/string (length)` VERIFIES |
| B3 | Contract-as-spec callee: `ensures` mentions the callee's own parameter, unsubstituted (`unknown constant e` / `n`) | RC2 | 0 | **1** | discord `validEmbed` | `callee_param.ail` (`atLeast` ERROR) |
| B4 | Imported ADT constructor undeclared (#1959) | RC2 | 0 | 0 on dev (1 on v0.54.17) | — | package repro above (VERIFIED / VIOLATION / SKIPPED on dev) |
| C1 | Go encoder error outside the two skip sentinels (`record construction: unknown record type`) | RC3 | 0 | **1** | crew-core `policies` (record with function-valued fields) | `fn_field_record.ail` (ERROR) |
| | **Total** | | **0** | **32** | relativity 20, celestial 5, billing-proposals 3, crew-core 2, content-library 1, discord 1 | |

Notes on the table:

- Each function is counted once, by its first error, and many carry several defects. `luminousEfficacy` hits A3, B1
  (`exp`) and an unencodable `pow`. Fixing one bucket does not make every function in it green; some will move to an
  honest SKIP.
- An adjacent misclassification, M1: a Z3 error whose *text* contains `timeout` is reported as
  `unknown`/`solver_timeout`. The cause is the substring test at `solver.go:206`. Fixture `timeout_name.ail` (a helper
  named `timeoutScale` wrapping `exp`) currently gives `backoff unknown solver_timeout "solver timeout"`. It is not
  unsound, but a verifier bug gets blamed on solver capacity.

---

## 3. Root causes (file:line at 7d29031ea)

### RC1: Operators and literals are chosen by syntax, not by elaborated type

Operators reach the encoder in three shapes, and only one of them is type-indexed:

| Shape | Encoder | Type-aware? |
|-------|---------|-------------|
| `App($builtin.lt_String, …)` (after op lowering) | `BuiltinToSMTOp` / `StringBuiltinSpecial`, `types.go:261-312` (`lt_String→str.<` :302, `le_String→str.<=` :303, `gt/ge_String` flipped :310/:312, `div_Float→/` :270) | Yes |
| `DictApp{Dict: DictRef{Ord,String}, Method:"lte"}` (dictionary elaboration) | `encodeDictApp`, `codegen_operators.go:75-95`: `methodToSMT` (:77-87) maps `lt→<`, `lte→<=`, `div→div` and **never reads `da.Dict`** | **No** |
| `Intrinsic` / `BinOp` (fallbacks) | `codegen_operators.go:11-29` (`OpDiv:"div"` :14) and `:32-54` (`"/":"div"` :34) | **No** |

What the verifier actually receives was confirmed with a throwaway Go probe (written and then deleted in the
worktree) that ran `pipeline.Run` on `"a" <= c` and `x / 2.0`. It printed
`DictApp method=lte dict=*core.DictRef dict_Ord_String` and `DictApp method=div dict=*core.DictRef dict_Fractional_Float`.

So the type is *on the node*, and #1953's premise holds: `types.go:302-312` has the right mapping, but `encodeDictApp`
never consults it. Op lowering passes `DictApp` through unchanged (`internal/pipeline/op_lowering.go:293`).

Literal and constant rendering has the same problem, picking a form without the sort:

- Float literals use `fmt.Sprintf("%g")` (`codegen_expr.go:138,140`). With shortest precision, Go's `%g` switches to
  exponent form when the decimal exponent is below −4 or at least 6. `1000000.0` gives `1e+06` and `0.00001` gives
  `1e-05`, which Z3 reads as `1` followed by the symbol `e+06`. That is bucket A3. Shortest `%g` is exact, so precision
  is not lost: the defect is the form, not the value.
- `encodeList` returns `(as seq.empty (Seq Int))` for `[]` (`codegen_records.go:403-406`). Its comment says "the SMT
  solver will unify sorts as needed", which Z3 does not. #689 fixed the body/cons site (`codegen_seq_hint.go`) but not
  contract equality. That is bucket A4.
- `ResolveCalleesFromRoots` defaults a callee's return sort to `Int` when it is unknown (`callee_resolver.go:119-121`).
  It was not observed in the corpus, but it is the same guess, and it breaks the "no silent fallbacks" rule.

### RC2: Unresolved names fall through to "it's a constructor" and are emitted raw

- **The fallback.** `encodeApp` ends with "ADT constructor application": `name := stripConstructorPrefix(...)` and
  then `encodeCheckedConstructor` (`codegen_apps.go:115-116`; the `*core.Var` twin is :139). For a name that no ADT
  owns, `constructorSymbol` returns it unchanged (`parametric_context.go:44-45` and `:59-60`, where
  `needsType == false` when `owners` is empty). Any name the encoder does not recognise is emitted as a bare SMT
  symbol: `sha256Hex`, `exp`, `length`, `ResourceLimit` before v0.54.18.
- **The leak guard is not active where callee bodies are encoded.** `activeUserFunctions` turns an unresolved
  user-function call into `ErrUnresolvableTypes`, which becomes an honest SKIP (`codegen_apps.go:111`). It is only
  populated at `codegen.go:194-205`, *after* `ResolveCalleesFromRoots` has already encoded every callee body
  (`codegen.go:169`, inside it `callee_resolver.go:163`). During callee encoding the map is nil.
- **The callee gate is asymmetric.** The top-level gate rejects unencodable builtins (`IsSMTEncodable` →
  `firstUnencodableBuiltin`, `encodable.go:93`, :580). `IsSMTEncodableForCallee` (`callee_resolver.go:526-563`)
  checks purity, recursion, HOFs and types, **but not builtins**. So `shortHash`'s body
  `substring(sha256Hex(text),…)` is judged encodable and emitted as a `define-fun` containing a raw `sha256Hex`.
  That is #1950 and bucket B1.
- **Silent callee failure.** When `EncodeExpr(innerBody)` fails, the callee is skipped with `continue`
  (`callee_resolver.go:166-169`). The caller then emits a call to an undefined function.
- **Stdlib names without an import.** `ResolveStdlibToBuiltin` (`types.go:431-443`) maps only references whose module
  is `std/string`/`std/list`. A name that type-checks without an import reaches the encoder under a reference the
  table does not list. It then falls through to the constructor fallback: bucket B2 (`length`; `startsWith` in the
  `strens` probe).
- **Contract-as-spec without arguments.** `EncodeCalleeByContract(spec, nil, 0)` (`callee_resolver.go:144`) passes no
  call-site arguments, so `buildParamSubstitution` (`codegen_xmod_contract.go:114-127`) substitutes nothing. The
  callee's `ensures` is asserted with its parameter names free: bucket B3, and also S2 below.

### RC3: There is no pre-flight. Z3 is the only well-formedness check, and its verdict is read by prefix.

- Nothing checks the emitted script before `Solve`. `codegen_decl_validation.go` checks only the sorts *inside*
  datatype and constant declarations. No checker walks applications against a symbol table: `grep -rn "unknown
  constant" internal/smt/*.go` (non-test) finds only comments.
- Any Go encoder error outside the two sentinels is ERROR (`verify.go:453-462`). That is bucket C1.
- `Solve` decides the verdict with `strings.HasPrefix(outputStr, "unsat")` / `"sat"` (`solver.go:188,193`), and
  decides "timeout" with `strings.Contains` over the whole output (`solver.go:206`). That last test is misclassification
  M1.

---

## 4. Soundness

### 4.1 Can a Z3 error become VERIFIED or a counterexample? **No. The evidence follows, along with why that is fragile.**

Z3 (4.16.0, default `continued-execution`) reports each failing command as an `(error …)` line *in command order*,
**drops that command**, and keeps going. It then prints a verdict for the weakened script. Every command we emit that
can fail comes before `(check-sat)`, so the first output line is `(error`. `Solve` then falls past the `unsat`/`sat`
prefixes to `StatusError` (`solver.go:213-218`). Evidence:

- All 31 Z3-error rows in the package corpus, plus all issue and fixture repros, start with `(error`. The verdict that
  follows is `sat` in 25, `unsat` in 3 (billing `createProposal`, `markAwaitingPayment`; discord `validEmbed`) and
  `unknown` in 3 (relativity `coastAt`, `betaOf`, `gammaOf`). Every one is reported as `status: error`.
- Direct Z3 check: a script with `(assert (> (length $p_s) 0))` followed by a valid goal prints
  `(error "… unknown constant length (String) ")` then `unsat`.

What the post-error verdict means:

- Dropping commands only *removes* constraints. An `unsat` after errors is therefore still unsat for the intended
  script, so it is not a false proof *of that encoding*.
- A `sat` after errors is a model of a **weakened** script, and that is meaningless. In the #1953 and #1950 repros,
  `define-const result` was dropped, then every assertion on `result`, so Z3 checked an empty problem.
- That bogus `sat` and its model are **printed to the user inside the ERROR reason**. That is what the #1959 reporter
  read as a possible unsound proof.

The guard is fragile because it is a single, incidental, untested property. `solver_test.go` has no case with an
`(error` line before a verdict. Any refactor that looks for the verdict line instead of the first line (to tolerate
warnings, say, or a `print-success` option) would turn all 25 bogus `sat`s into published COUNTEREXAMPLES.
M3 makes the guard explicit and pins it with a test.

### 4.2 Two false-VERIFIED paths in the same layer (found by this audit; well-sorted, so invisible to Z3 and to a pre-flight)

**S1: Int `/` and `%` are Euclidean in SMT but truncating at runtime.**

- `div_Int→div` and `mod_Int→mod` (`types.go:269,271`), `OpDiv/OpMod` (`codegen_operators.go:14`), `"/"→div` /
  `"%"→mod` (:34) and DictApp `div→div` (:79).
- SMT-LIB `div`/`mod` are Euclidean (`-7 div 2 = -4`, `-7 mod 2 = 1`). AILANG follows Go and truncates toward zero.

Fixture `int_divmod.ail`:

```
$ ailang verify int_divmod.ail
  ✓ VERIFIED parity     -- ensures { result >= 0 }  body x % 2      (define-const result Int (mod $p_x 2))
  ✓ VERIFIED halfDown   -- ensures { result * 2 <= x } body x / 2
$ ailang run --caps IO --entry main divmod.ail
parity(-7) = -1
halfDown(-7) = -3
$ ailang run --caps IO --verify-contracts --entry main divmod.ail
Error: execution failed: contract violation: ensures failed … (result >= 0)
```

The `Scope:` banner says "int=Int; runtime equivalence not_established (not an IEEE/machine-arithmetic proof)". That
disclaims *width* (int64 wrap, which is
`ailang-core-triage/verify-int-width-mismatch-z3-int-vs-int64.md`) and IEEE floats. It does not disclaim a different
division operator: AILANG's `/` truncates over unbounded integers too. This is a wrong operator, not a model limit.

**S2: A contract-as-spec callee collapses every call site into one constant.**

`EncodeCalleeByContract(spec, nil, 0)` (`callee_resolver.go:144`) always uses call-site index 0, and `encodeApp`
substitutes `$xmod_result_<f>_0` for *every* call of `f`, whatever its arguments (`codegen_apps.go:101-105`,
`:123-127`). So `f(a) - f(a+1)` encodes as `c - c`. Fixture `call_site_collapse.ail`:

```
  ✓ VERIFIED gap                -- ensures { result == 0 }  body total(a) - total(a + 1)
      (declare-const $xmod_result_total_0 Int)
      (assert (>= $xmod_result_total_0 0)) ; total ensures
      (define-const result Int (let (($tmp4 $xmod_result_total_0)) … (- $tmp4 $tmp6)))   ; $tmp6 is the same constant
$ ailang run --caps IO --entry main xspec.ail        → gap(1) = -2
$ ailang run --caps IO --verify-contracts …          → contract violation: ensures failed … (result == 0)
```

Neither path is reachable from a Z3 error, so the caller's P1 trigger ("ERROR→false-VERIFIED") is not literally met.
But M1 and M2 make float division, helpers and contract-callees *encodable*, which sends more functions into exactly
these two paths. **M0 must land before or with M1/M2.**

The target encodings were checked directly in Z3:

| Script | Result |
|--------|--------|
| Truncating `mod`, `(- a (* b (ite (= (>= a 0) (>= b 0)) (div (abs a) (abs b)) (- (div (abs a) (abs b))))))`, on `parity` | `sat`, `$p_x = -1` |
| Per-call-site instantiated UF `sumTo_spec` on `gap` | `sat` |
| `str.<=` on `lower` | `unsat` |
| The false variant `lowerWrong` | `sat`, `c = "z"` |
| `cosSeen` with `/` and Real literals | `unsat` |

---

## 5. Reconciliation with existing planned docs

| Doc | Relationship |
|-----|-------------|
| `ailang-core-triage/smt-encode-dt-builtins.md` | **Not subsumed.** It adds *encodings* for `_dt_*` builtins. This doc guarantees that a builtin *without* an encoding becomes a named SKIP, including inside callee bodies (B1). Both are needed; they do not conflict |
| `ailang-core-triage/verify-hof-inline-behind-callee-sort-gate.md` (#1375, #215) | **Partially subsumed.** Its "#215 masked" half has two mechanisms. The hard-coded `(Seq Int)`/`Int` element sort in `hof_inline.go:331-347` is RC1, and a wrong element sort is a sort error the M3 pre-flight catches (SKIP, never ERROR). Its accessor-shadowing `let` hygiene is **not** covered here. `let` names are still emitted under their source names (`codegen_control.go:21`); ef7bdc8e0's `codegen_scope.go` only stops a `$p_` rename from capturing a let. Shadowing produces well-sorted but wrongly-bound SMT, so it stays with that doc. The #1375 decision (exempt inlinable HOFs from the gate) stays with that doc, and M3 is the safety net that makes doing it safe |
| `ailang-core-triage/verify-ensures-verified-without-encodable-refutation.md` | **Not subsumed; adjacent.** That is proof-vs-refutation asymmetry when `ensures` calls a user function. S2 here is a different false proof (call-site collapse). Both live in `callee_resolver.go`/`codegen_xmod_contract.go`, so the M0 S2 fix should be reviewed against that doc's option (a) "uninterpreted function + conjoined callee contract". M0 adopts that shape |
| `ailang-core-triage/verify-call-site-precondition-not-checked.md` | **Not subsumed; shares S2's root.** Both stem from the missing call-site argument tracking ("deferred to M4", `callee_resolver.go:129`). The current M0 packet carries actual-call arguments and guards postcondition facts; discharging `requires` as an obligation remains that doc's decision |
| `ailang-core-triage/verify-int-width-mismatch-z3-int-vs-int64.md` | **Not subsumed.** That doc is about width/overflow. S1 is the *division operator*, which is wrong even over unbounded ℤ. If that doc chooses BitVec 64, S1's `bvsdiv`/`bvsrem` are truncating by definition, so the two fixes agree |
| `m-contract-verification-coverage.md` (skipped vs not_applicable) | **Not subsumed; must compose.** That doc splits the *skip counter* by cause and is parked under D-29/D-30. M3 adds a new skip cause, `encoder_gap`, so that it stays separable from `unsupported_encoding` in any future split, and it does **not** change `isVerifiedSuccess` (`internal/observatory/cost_per_verified_success.go`). See Design Freeze F2 |
| `v0_38_0/m-smt-string-casefold-charfold.md` | **Not subsumed.** It adds `toUpper`/`toLower`/`foldChars` encodings. This doc makes the existing string-ordering encodings reachable (A1). No overlap in files beyond `types.go` tables |
| `implemented/v0_54_18/m-smt-parametric-adt-declarations.md` | **#1959's primary symptom is done there.** This doc generalises its point fix (B4) into the M2 resolver and M3 pre-flight, and pins the #1959 package repro as a regression fixture |

**Subsumed outright:** #1953, #1950 and #1959 (the residual). The `%g` literal defect and the contract-side empty-list
defect had no doc or issue. The soundness findings S1 and S2 were subsequently filed as #1960/#1961; the current packet controls M0.

---

## 6. Goals

**Primary goal:** every contracted function ends as VERIFIED (sound), COUNTEREXAMPLE (a model of the *intended*
script), or SKIPPED with a named reason. ERROR is reserved for verifier faults, and CI keeps it at zero on the
in-repo corpus.

**Success metrics:**

1. The 11 fixtures and the #1959 package fixture in §9 reach their pinned statuses, the 3 issue repros included.
2. Package corpus re-run (ailang-packages 8e6bbaf): **0 ERROR**, down from 32. Each former ERROR becomes VERIFIED,
   COUNTEREXAMPLE or SKIPPED with a reason code.
3. examples/ + std/ stay at 0 ERROR, with the CI gate in place. The previously-VERIFIED set (110) and the intended
   COUNTEREXAMPLE set (23) do not change.
4. S1/S2 fixtures are no longer VERIFIED.

---

## 7. High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| F1: Fix S1 by **encoding truncating `/`/`%`** (recommended) instead of documenting Euclidean semantics in the scope banner | Changes which existing proofs hold. A contract that only holds under Euclidean division stops verifying | human | design | med |
| F2: Pre-flight failures report **SKIPPED with new reason code `encoder_gap`**, in *both* `verify` and `ai-check` | ai-check exit and eval `verify_errors` change for programs that used to ERROR (bank-forward boundary). `UnsupportedConstructIsError` keeps its existing meaning only for `ErrUnsupportedConstruct` | human | design | med |
| F3: Pre-flight is a **post-hoc sort checker over the emitted SMT-LIB** (recommended) rather than a typed SMT AST built in Go | Typed AST = rewrite of about 11k lines of string emitters. A post-hoc checker covers every producer (body, contract, callee, HOF specialiser, unroller, xmod) with no emitter changes | agent (recommended: post-hoc) | design | high |
| F4: Fix S2 with an **uninterpreted function per contract-callee, with ensures instantiated per call site under argument substitution** (recommended), not fresh constants per call | Fresh constants are sound but lose functional consistency (`f(a) - f(a)` stops proving 0). UF keeps it | agent | design | med |
| F5: Residual Z3 errors *after* a passing pre-flight stay **ERROR** (a verifier bug) and are CI-gated to 0 | Keeps ERROR meaningful instead of turning every surprise into a SKIP | human | design | low |

### Design Freeze — full roadmap (not approved by an M0-only decision)

Current M0-F1/F4/Z are separately ratified above for planning only. Parent F1 is checked only for that M0 scope;
F2/F3/F5 stay deferred. The authority is the actual 2026-10-29 user ruling, not the parked agent plan.

- [x] F1: truncating Int `/`/`%` ratified for M0 planning only on 2026-10-29; later milestones remain deferred.
- [ ] F2: `encoder_gap` SKIP in both verify and ai-check, plus a changelog bank-forward note
- [ ] F3: post-hoc sort checker
- [ ] F5: residual Z3 error stays ERROR

---

## 8. Solution Design

### M0: Soundness first (S1, S2), current proposal above

The authoritative current M0 algorithm, partial-correctness/zero policy, approval freezes and actual-CLI acceptance
criteria are in the **Current decision packet** above. S1 may use centralized inline parallel lets instead of the
original preamble helpers. S2 retains UF identity but requires finite, requires-guarded lexical/path facts; the
parked universal-axiom alternative is rejected by the partial-callee discriminator. No M0 implementation is accepted
or newly authorized by this document update.

### M1: Type-directed operator and literal resolution (RC1). About 1 day.

- One function, `resolveOperator(method string, operandType types.Type) (smtOp, error)`, keyed by `(method, head type)`
  over the existing `BuiltinToSMTOp`/`StringBuiltinSpecial` tables, which become the single source of truth.
  - `encodeDictApp` takes the head from `da.Dict.(*core.DictRef).TypeName`. That is already `"String"`/`"Float"`/`"Int"`;
    see the probe in §3.
  - `encodeBinOp` and `encodeIntrinsic` take it from `CoreTI` on the operand node, via the existing
    `activeADTContext.ti`.
  - An operand whose type is unknown or unmapped returns `ErrUnsupportedConstruct`-class `encoder_gap`, never a default
    to Int.
  - Fixes A1 and A2, and with M0 the S1 mapping.
- Float literal rendering: an exact decimal with no exponent. Use `strconv.FormatFloat(v, 'f', -1, 64)`, appending `.0`
  when the value is integral, and keep the `(- …)` form for negatives. Fixes A3. Non-finite literals (NaN/Inf) become
  `encoder_gap`.
- `encodeList([])` takes the element sort from the node's elaborated type (`CoreTI`) wherever it appears; the contract
  site is the missing one. Fixes A4. The `callee_resolver.go:119-121` `Int` default becomes `encoder_gap`.

### M2: One name resolver (RC2). About 1.5 days.

- **Order fix.** Populate `activeUserFunctions` (and the resolver state below) *before* `ResolveCalleesFromRoots`, so
  callee bodies are encoded under the same leak guard as caller bodies (`codegen.go:169` vs `:194`).
- **Symmetric gate.** `IsSMTEncodableForCallee` calls `firstUnencodableBuiltin`. An unencodable callee then takes the
  existing contract-as-spec path if it has `ensures` (now sound after M0), or skips the *caller* with
  `"calls shortHash, which uses an unencodable builtin: std/crypto.sha256Hex"`. That is #1950's expected message.
  The reporter's second suggestion (treat `sha256Hex` as an uninterpreted `String→String` function) is a sound
  *improvement* that would make `takeFile` VERIFY. It is listed under Future Work and not taken here, because it
  widens what VERIFIED rests on.
- **No silent callee drop.** When a callee body fails to encode, `callee_resolver.go:166-169` records the failure. A
  call to that callee then returns `ErrUnresolvableTypes` (SKIP) instead of emitting an undefined symbol.
- **Constructor fallback only for known constructors.** `constructorSymbol` with no owner returns
  `ErrUnresolvableTypes` naming the symbol, instead of the raw name (`parametric_context.go:44-45, 59-60`). Fixes the
  generic form of B4.
- **Stdlib names by resolved identity.** Resolve a stdlib call by the function's resolved module and name (the
  `VarGlobal.Ref` after import/prelude resolution), not by the import spelling, so unimported `length`/`startsWith`
  map like imported ones. Fixes B2.

### M3: Well-sortedness and declaration pre-flight (RC3). About 2 days.

New file `internal/smt/preflight.go`, about 350 LOC. It runs on `encResult.SMTLib` between `EncodeFunction` and `Solve`
(`verify.go:465`):

1. A minimal s-expression reader. The emitted subset is closed: `set-logic`, `declare-datatype(s)`, `declare-const`,
   `declare-fun`, `define-fun`, `define-const`, `assert`, `check-sat`, `get-model`, with terms built from `let`, `ite`,
   `match`, `as`, `forall`/`exists`, application and literals.
2. A symbol table built in command order:
   - sorts: `Int Real Bool String`, `(Seq s)`, declared datatypes including `par` binders;
   - functions/constants: declared, defined, plus constructors, accessors and testers generated from each datatype;
   - a **theory signature table** for every operator the encoder can emit (`+ - * / div mod abs < <= > >= = distinct
     and or not ite str.* seq.* to_real to_int bv*` …), with Int/Real overloading as Z3 accepts it.
3. Check every command. A symbol must be declared before use, and every application must match a signature. A
   `define-*` body must have the declared sort. An `assert` must be Bool.
4. On the first failure, return `PreflightError{Symbol, Want, Got, Command}`. The driver reports
   `status: skipped, reason_code: encoder_gap` with the reason naming the symbol and sort, e.g.
   `"encoder_gap: '<=' applied to (String String); expected (Int Int)"`. Z3 is not run.

Also, in `Solve`:

- Any `(error` line anywhere in the output means `StatusError`. Never parse a verdict or model from errored output,
  and do not include the post-error verdict and model in `Reason`.
- Replace `strings.Contains(outputStr, "timeout")` (`solver.go:206`) with an exact match on Z3's own `timeout` verdict
  line. Fixes M1.
- Pin both with unit tests (§10).

Generic Go encoder errors (`verify.go:453-462`, bucket C1) become `encoder_gap` SKIPs too. They are the Go-side
analogue of a pre-flight failure. `ErrUnsupportedConstruct` keeps its current ai-check behaviour behind
`UnsupportedConstructIsError`.

Add `EncoderGap ReasonCode = "encoder_gap"` to `internal/verification/evidence.go` (unallocated:
`grep -rn encoder_gap internal cmd` is empty).

### M4: CI gate. About 0.5 day.

- `cmd/ailang/verify_preflight_e2e_test.go` (skips without Z3, like the existing verify e2e tests; CI installs z3 at
  `.github/workflows/ci.yml:171-173`). It does two things:
  1. A table test over `cmd/ailang/testdata/verify_preflight/*.ail` with pinned per-function statuses (§9).
  2. A **corpus walk** over every `examples/**/*.ail` and `std/*.ail` with a contract, failing on any `status: error`
     and on any function whose status leaves the pinned VERIFIED set. That set is generated once into
     `cmd/ailang/testdata/verify_preflight/corpus_baseline.json` (110 verified / 23 counterexample at 7d29031ea).
- Why not `make verify-examples`: that target runs only `ailang check`/`ailang run` (`scripts/verify_examples.go:216-226,
  375-390`), never `ailang verify`. That is why `hof_verify.ail` could regress to SKIP unnoticed (#1375).
- Optional, off by default: `make verify-corpus-packages PKGS=~/dev/sunholo-data/ailang-packages` reruns §2.2 and
  prints the bucket table. It is not a CI dependency, because that repo is external.

### Files

**New**

- `internal/smt/preflight.go`: s-expr reader, symbol table, theory signatures, checker (~350 LOC)
- `internal/smt/preflight_test.go`: one case per bucket, at the SMT-text level (~250 LOC)
- `internal/smt/codegen_operator_resolve.go`: `resolveOperator`, float literal rendering, `ail_idiv`/`ail_imod` (~150 LOC)
- `cmd/ailang/verify_preflight_e2e_test.go`: fixture table plus corpus walk (~200 LOC)
- `cmd/ailang/testdata/verify_preflight/`: the 11 fixtures in §9, the #1959 package dir (`pkg_adt_import/`), `corpus_baseline.json`

**Modified**

- `internal/smt/codegen_operators.go`: DictApp/BinOp/Intrinsic through `resolveOperator` (−60/+30)
- `internal/smt/codegen_expr.go`: float literal rendering (±10)
- `internal/smt/codegen_records.go`: typed empty list (±10)
- `internal/smt/codegen.go`: populate resolver state before `ResolveCalleesFromRoots` (±15)
- `internal/smt/callee_resolver.go`: builtin gate, no silent drop, no `Int` default, call-site args (+60)
- `internal/smt/codegen_xmod_contract.go`: UF-per-callee with per-call ensures (+50)
- `internal/smt/codegen_apps.go`: contract callee to UF application; stdlib by resolved identity (±30)
- `internal/smt/parametric_context.go`: no raw fallback (±10)
- `internal/smt/solver.go`: error-anywhere means ERROR; exact `timeout` line (±20)
- `internal/smt/verify.go`: pre-flight call; `encoder_gap` mapping (+40)
- `internal/verification/evidence.go`: `EncoderGap` (+1)
- `docs/docs/guides/contracts.mdx`: string ordering, float division, and Int division semantics (truncating) stated
  with examples
- `changelogs/unreleased/2026-10-2x-smt-well-sorted-preflight.md`: Fixed (#1953, #1950, #1959) and Changed (bank-forward
  note for `verify_errors` → `verify_skipped:encoder_gap`)

`verify.go` is 554 lines; the additions keep it under 800.

---

## 9. Historical full-roadmap acceptance (M0 packet controls current acceptance)

All fixtures live under `cmd/ailang/testdata/verify_preflight/` with `module cmd/ailang/testdata/verify_preflight/<name>`,
and run from the repo root. **Every fixture below was written and `ailang check`ed in a scratch mirror of that path at
7d29031ea (all "No errors found"), and its current dev status recorded.**

| Fixture · function | Contract truth | Dev today | **Required after** |
|---|---|---|---|
| `string_order` · `lower` (#1953) | true | ERROR (Int `<=`) | **VERIFIED** |
| `string_order` · `after` (#1953, `<`) | true | ERROR | **VERIFIED** |
| `string_order` · `pick` (ordering in requires/ensures) | true | ERROR | **VERIFIED** |
| `string_order` · `lowerWrong` (negative control) | **false** | ERROR | **COUNTEREXAMPLE** |
| `float_div` · `half` | true | ERROR (`div` on Real) | **VERIFIED** |
| `float_div` · `halfWrong` (negative control) | **false** | ERROR | **COUNTEREXAMPLE** |
| `float_literal` · `scaleUp` (`1000000.0`) | true | ERROR (`e+06`) | **VERIFIED** |
| `float_literal` · `scaleUpWrong` (`0.00001`, negative control) | **false** | ERROR (`e-05`) | **COUNTEREXAMPLE** |
| `helper_builtin` · `takeFile` (#1950) | n/a | ERROR (`unknown constant sha256Hex`) | **SKIPPED**, reason names `std/crypto.sha256Hex` |
| `helper_builtin` · `direct` (#1950 control) | n/a | SKIPPED | SKIPPED (unchanged) |
| `unimported_stdlib` · `label` | true | ERROR (`unknown constant length`) | **VERIFIED** |
| `callee_param` · `atLeast` (B3) | true | ERROR (`unknown constant n`) | **VERIFIED** |
| `empty_list_contract` · `empty` (A4) | true | ERROR (`(Seq Issue)` vs `(Seq Int)`) | **VERIFIED** |
| `fn_field_record` · `policies` (C1) | n/a | ERROR (encoding error) | **SKIPPED** `encoder_gap` |
| `int_divmod` · `parity` (S1) | **false** | **VERIFIED (unsound)** | **COUNTEREXAMPLE** (e.g. `x = -1`) |
| `int_divmod` · `halfDown` (S1) | **false** | **VERIFIED (unsound)** | **COUNTEREXAMPLE** |
| `int_divmod` · `halfNonNeg` (S1 control) | true | VERIFIED | VERIFIED |
| `call_site_collapse` · `gap` (S2) | **false** | **VERIFIED (unsound)** | **COUNTEREXAMPLE** |
| `timeout_name` · `backoff` (M1) | n/a | unknown / `solver_timeout` | **SKIPPED**, reason names `std/math.exp` (never `solver_timeout`) |
| `pkg_adt_import/probe` · `checked` (#1959) | true | VERIFIED | VERIFIED (pinned) |
| `pkg_adt_import/probe` · `wrong` (#1959 negative control) | **false** | VIOLATION | COUNTEREXAMPLE (pinned) |
| `pkg_adt_import/probe` · `unitR` (`Result[(), E]`) | n/a | SKIPPED | SKIPPED (pinned; never ERROR) |

The `sumTo`/`total` functions inside `callee_param`/`call_site_collapse` currently report the known bounded-unrolling
false counterexample (`m-verify-bounded-unrolling-false-counterexample.md`). The test does not pin them.

Further criteria:

- [ ] **Package corpus:** ailang-packages 8e6bbaf, `verify --package .` over all 57 valid packages gives **0 error**.
      The bucket table in §2.3 is re-run and attached to the PR.
- [ ] **In-repo corpus gate:** the corpus walk in M4 passes. Any `status: error` in examples/ or std/ fails CI. The 110
      VERIFIED and 23 COUNTEREXAMPLE outcomes at 7d29031ea are unchanged.
- [ ] **Solver guard pinned:** `TestSolve_ErrorBeforeVerdictIsError` feeds `Solve` a script with an undeclared symbol
      in an `assert` and a satisfiable remainder (Z3 prints `(error …)` then `sat`), and a second one that prints
      `(error …)` then `unsat`. Both must be `StatusError`, with no model and no verdict in `Reason`. Mutation-test it:
      revert the guard and the test must fail.
- [ ] **Pre-flight unit tests:** one SMT-text case per bucket (A1–A4, B1–B3, C1), each producing `PreflightError` with
      the expected symbol. A well-sorted control script (`cosseen_real.smt2` from §4.2) must pass.
- [ ] `ai-check` on `float_div.ail` reports `verified: 1, counterexample: 1, errors: 0`. Today it is
      `rc=1, errors: 2`. It still exits 1, but now for the genuine counterexample (`ai_check.go:208`). A copy holding
      only `half` exits 0. The ai-check exit contract test (`cmd/ailang/ai_check_exit_test.go`) is updated where it
      pinned an encoder-error case.
- [ ] `make test`, `make lint`, `make check-file-sizes` and `make verify-examples` pass. A changelog fragment is added
      and `docs/docs/guides/contracts.mdx` is updated.

---

## 10. Historical full-roadmap testing strategy

- **Unit (internal/smt).** Pre-flight per bucket; `resolveOperator` table (each method × Int/Float/String/Bool, and
  unknown → `encoder_gap`); float rendering round-trip (`strconv.ParseFloat(render(v)) == v` for 10k random finite
  values, with no `e`); `ail_idiv`/`ail_imod` agree with Go `/`/`%` on the grid −9…9 × {−3,−2,−1,1,2,3}; Solve error
  ordering.
- **E2E (cmd/ailang).** The fixture table and corpus walk (M4).
- **Mutation testing** (project rule, done via scratchpad copies). Revert each fix and its fixture row must flip. In
  particular, re-adding the `codegen.go` ordering must make `takeFile` ERROR again.
- **Manual.** Re-run §2.2 against ailang-packages and diff the bucket table.

---

## 11. Conflict Surface

This touches `internal/smt` and `internal/verification`, not the parser or typechecker. These are the shared consumers
whose behaviour changes:

| Surface | Change | Decision |
|---|---|---|
| `ai-check` exit code (`cmd/ailang/ai_check.go:208`) | Programs that ERROR today exit 0 once they SKIP | F2 (human) |
| Eval banking `verify_errors` / `verify_skipped` (`internal/eval_harness/verify.go:143,156`) | Errors move into skipped (`encoder_gap`). Historical rows are not comparable across the boundary | F2, plus a changelog bank-forward note |
| `isVerifiedSuccess` (`internal/observatory/cost_per_verified_success.go`) requires `VerifySkipped == 0` | A run that was ERROR (already not a success) becomes SKIPPED (still not a success). **No KPI flips from fail to pass** | Reuse; no override |
| `UnsupportedConstructIsError` (verify.go `VerifyOptions`) | Keeps its meaning for `ErrUnsupportedConstruct` only | Reuse |
| Previously-VERIFIED proofs relying on Euclidean `/`/`%` | May become COUNTEREXAMPLE (true at runtime? then the contract was wrong) | F1 (human). None found in examples/std: the corpus walk is the check |
| `verify --strict` | Unchanged: SKIP already fails `--strict` | Reuse |

Programs that must still work: `examples/runnable/contracts/result_error_guard.ail`, `empty_list_sort_verify.ail`,
`nullary_constant_verify.ail`, `cross_module_functions.ail`, `string_verify.ail`, `finance.ail` (uses Int `/` on
non-negative values; must stay VERIFIED under truncating division). All exist at 7d29031ea.

**Historical issue-filing note resolved:** S1 is tracked by #1960 and S2 by #1961; do not create duplicates.

---

## 12. Alternatives considered

| Option | Verdict |
|---|---|
| **A. Point fixes only**: map DictApp by type (#1953), add the builtin check to the callee gate (#1950), close #1959 as done | Fixes 3 issues and ~21 of the 32 package ERRORs. Leaves `%g`, contract `[]`, unimported names and C1, and the next producer leaks again. This is the pattern §1.2 documents. Rejected as the whole fix; it is M1+M2's subset |
| **B. Reclassify by Z3 message**: map `unknown constant` / `Sort mismatch` in Z3 output to SKIPPED | One afternoon of work. But it is still discovery-by-Z3, it string-matches a third-party message format, and it gives no Go-side origin. It would also have SKIPPED S1/S2's siblings silently instead of fixing them. Rejected; M3's checker is the structural form |
| **C. Typed SMT AST in Go**: emitters build typed terms and a printer renders them | The strongest guarantee, but it means rewriting every string emitter (the codegen/HOF/unroll/callee/parametric files total about 11k non-test lines) before any user-visible fix. Deferred: M3's checker can later be replaced by C without changing the outcome contract |
| **D. `(set-option :print-success true)` + incremental Z3, stop at first error** | Still uses Z3 as the checker, needs a stateful process, and gives no symbol-level reason we control. Rejected |
| **E. Treat unencodable builtins as uninterpreted functions** | Sound and more complete (`takeFile` would VERIFY). It changes what a VERIFIED rests on, and needs the M0 UF machinery first. Future Work |

---

## 13. Deferred Decisions

- The exact shape of the theory signature table (generated from the emitters' operator tables vs hand-written): the
  agent may choose, provided every operator the encoder can emit is listed. A test enumerates `BuiltinToSMTOp`,
  `StringBuiltinSpecial`, `ListBuiltinSpecial` and `NumericBuiltinSpecial` and fails if any operator is missing.
- Whether `encoder_gap` reasons also carry the Core node position: the agent may choose.
- Whether to sort `report.Results` by function name (human output order currently follows Go map iteration in
  `verify.go`): the agent may choose. It is cosmetic, but it would make diffs of the bucket table stable.

## 14. Non-Goals

- New encodings for unencodable builtins (`exp`, `sqrt`, `pow`, `sha256Hex`, `_dt_*`). Each becomes a named SKIP here.
  Encoding them belongs to `smt-encode-dt-builtins.md` and future encoding docs.
- Discharging callee `requires` at call sites (`verify-call-site-precondition-not-checked.md`).
- Int width / BitVec (`verify-int-width-mismatch-z3-int-vs-int64.md`).
- The bounded-unrolling false counterexample (`m-verify-bounded-unrolling-false-counterexample.md`).
- The skipped vs not_applicable counter split (`m-contract-verification-coverage.md`).

## 15. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Truncating `ail_idiv`/`ail_imod` (abs/ite, non-linear) slows Z3 on arithmetic-heavy contracts | Med | Emit the helpers only when Int `/`/`%` occurs. When both operands are provably non-negative (from `requires`), the agent may emit plain `div`/`mod`; that is an optimisation, not a semantics change. Measure on `finance.ail`, `invoice.ail` and relativity |
| The pre-flight's theory table lags a new emitter and false-rejects a valid script | Med (a would-be VERIFIED becomes SKIPPED) | The table-completeness test in Deferred Decisions; corpus baseline pins the 110 VERIFIED |
| `encoder_gap` hides a real verifier bug as a quiet SKIP | Med | Reason names the symbol; the package-corpus re-run and the per-reason counts in `verify --package` output keep them visible; F5 keeps residual Z3 errors loud |
| Bank-forward discontinuity in `verify_errors` | Low | Changelog note with the cut-over version; `reason_code` is already banked per function so old/new are separable |

## 16. Timeline

M0 1d → M1 1d → M2 1.5d → M3 2d → M4 0.5d. That is about 6 days. M0 is independently shippable and should go first.

## Related Documents

- `design_docs/implemented/v0_30_0/m-smt-callee-sort-gate.md`: the gate and the leak-site guard this doc generalises
- `design_docs/implemented/v0_54_18/m-smt-parametric-adt-declarations.md`: #1959's point fix
- `design_docs/planned/ailang-core-triage/verify-hof-inline-behind-callee-sort-gate.md`
- `design_docs/planned/ailang-core-triage/verify-ensures-verified-without-encodable-refutation.md`
- `design_docs/planned/ailang-core-triage/verify-call-site-precondition-not-checked.md`
- `design_docs/planned/ailang-core-triage/verify-int-width-mismatch-z3-int-vs-int64.md`
- `design_docs/planned/ailang-core-triage/smt-encode-dt-builtins.md`
- `design_docs/planned/m-contract-verification-coverage.md`
- `design_docs/planned/v0_38_0/m-smt-string-casefold-charfold.md`
- Neural related-doc search on "smt well sorted preflight" returned nothing at or above 0.45. The highest were
  `implemented/v0_4_3/M-SOUNDNESS-COMPLETION-REPORT.md` (0.31) and `planned/v0_38_6/m-smt-datetime-sprint-plan.md`
  (0.30). Duplicate gate: proceed.

## Verification Log

| # | Claim | How verified |
|---|---|---|
| V1 | #1953 reproduces for `<=`, `<`, and in contracts | `ailang-od verify` on `strcmp.ail`, `strlt.ail`, fixture `string_order.ail` (4 ERRORs) |
| V2 | Operators reach the encoder as `DictApp` with a typed `DictRef` | Throwaway `pipeline.Run` probe printed `dict_Ord_String` / `dict_Fractional_Float`; probe file deleted, worktree clean |
| V3 | `encodeDictApp` ignores `da.Dict` | Read `codegen_operators.go:75-95` |
| V4 | #1950: callee bodies are encoded before `activeUserFunctions` is set | Read `codegen.go:169` vs `:194-205`, `callee_resolver.go:163`; repro shows raw `sha256Hex` inside `define-fun shortHash` |
| V5 | `IsSMTEncodableForCallee` has no builtin check | Read `callee_resolver.go:526-563` (purity, recursion, HOF, types only) vs `encodable.go:93` |
| V6 | #1959 reproduces on v0.54.17, fixed on v0.54.18/dev | v0.54.17 built from `git archive v0.54.17`: audio 0.2.0 → 1 error (`checkSize`); `ailang` v0.54.18 cd33d05 and dev → 0 errors; `git tag --contains ef7bdc8e0` → v0.54.18 |
| V7 | examples/ + std/ have 0 ERRORs | 71 files run, aggregated JSON (§2.2) |
| V8 | Packages have 32 ERRORs, bucketed as in §2.3 | 58 package dirs from ailang-packages origin/main 8e6bbaf via `git archive` (read-only); per-file `--verbose --json` to map each first error to its SMT token |
| V9 | No Z3 error is reported as VERIFIED/COUNTEREXAMPLE | `solver.go:186-224` prefix logic; all 31 Z3-error outputs start `(error`; post-error verdicts 25 sat / 3 unsat / 3 unknown, all `status: error` |
| V10 | No test pins error-before-verdict | `grep -n "func Test" internal/smt/solver_test.go`: no such case |
| V11 | S1 false VERIFIED | `int_divmod` / `divmod.ail`: VERIFIED statically; `run` prints `-1`/`-3`; `--verify-contracts` raises the ensures violation |
| V12 | S2 false VERIFIED | `call_site_collapse` / `xspec.ail`: VERIFIED; `gap(1) = -2` at runtime; `--verify-contracts` raises the violation; SMT shows one shared `$xmod_result_*_0` |
| V13 | Target encodings are right | Direct `z3` runs: `str_true` unsat, `str_false` sat, `cosseen_real` unsat, `parity_trunc` sat (`x=-1`), `gap_percall` sat |
| V14 | `timeout` substring misclassification | `timeout_name.ail` → `unknown` / `solver_timeout` for a Z3 "unknown constant timeoutScale" error |
| V15 | `make verify-examples` never runs `ailang verify` | Read `make/examples.mk:18-27` and `scripts/verify_examples.go:216-226, 375-390` (only `check` and `run`) |
| V16 | `encoder_gap` is unallocated | `grep -rn "encoder_gap\|EncoderGap" internal cmd` empty; existing codes listed at `internal/verification/evidence.go:15-26` |
| V17 | No pre-flight checker exists | `codegen_decl_validation.go` only validates declaration headers; no application/symbol check in `internal/smt` (non-test grep for `unknown constant` finds only comments) |
| V18 | ai-check exits 1 on these ERRORs | `ailang-od ai-check float_div.ail` → `rc=1`, `errors: 2` |
| V19 | All §9 fixtures are valid AILANG | `ailang-od check` on each in a scratch mirror of `cmd/ailang/testdata/verify_preflight/`: "No errors found" ×11; the #1959 package probe was checked in its own package dir |
| V20 | Archived 807 M0 rows falsely VERIFY; no actual aca5 run claimed | Actual 807 check/verify outputs plus new literal named-path equivalence records; broader cache changes and unbuilt-current qualification retained |
| V21 | Partial-callee fixture is valid source and false caller actually returns | Actual check 0; wrong(1) runtime 0; runtime contract exit 1; exact source/std warnings retained in companion JSON |
| V22 | Universal UF differs from finite call facts | Local Z3 4.16.0: universal partial-spec refutation UNSAT, actual-call control SAT; requires/identity controls and exact scripts in companion; not a parked branch CLI run |
| V23 | Zero arithmetic does not return a runtime value | Archived 807 compiler check 0, run 1 / RT001 for both `/0` and `%0`; companion JSON |
| V24 | Historical named-human-F1 claim is unproven | Primary user transcript message at 2026-10-26T14:59:31.188Z is a broad P0 workflow request; named “required sound choice” occurs in agent delegation/plan, not an explicit human F1 message |
| V25 | No native scoped-quorum flag | Read aca5 `design_quorum.go` flags and actual CLI help; `design_review.go:111–127` accepts path/stdin; DefaultSeats 3; actual two authorized runs/CLI records retained separately |
| V26 | Pinned integration and operator-held scope | Literal base/named-path diffs and real cache delta are embedded; no whole-compiler or fresh-live-ownership claim. Root instruction keeps #1790 held; no card/state change |
| V27 | Short-circuit/contract/guard fact selection and author label | Read `expr_simple.go:normalizeShortCircuit`, shape tests, `file_funcs.go:elaborateContracts`, `patterns.go:normalizeArm`, op-lowering Meta/Guard traversal; `design_quorum.go` author string passes directly to `VendorOf`/`SelectReviewers`; `codex` recognized, bare `openai` unrecognized; no new tests/provider call |
| V28 | Runtime signed Int premise, source/stdlib identity and full halfDown rows | Archived 807 CLI checks + tree/strict-VM 12-pair/operator matrix all match truncation; full original int_divmod check/verbose JSON verify, all 54 std hashes match 807; raw outputs and runtime snippets in companion |
| V29 | Named-path base comparison and quorum-input identity are explicit | Actual merge-base 399630; full named-path diff includes #602 test plus 6 std paths; actual help 0; source excerpts and in-packet committed-source identity protocol; both terminal BLOCKED records retained |
| V30 | Missing literal records now captured locally, without changing the reviewed inputs | Executed 807→aca5 verify/front-half diffs 0; stamped e863→aca5 quorum diff 0; full versions, blobs/source excerpts, real cache diff, cache-disable handling and frozen-packet hashes in local supplement now embedded |
| V31 | Installed dirty build and scope remain limited | Actual installed version reports v0.55.0-dirty/e863 fullstamp; hash/source comparison do not attest clean built bytes. Archived 807 is not a new aca5/fixed build; no whole-compiler equivalence claim |
| V32 | Gemini helper parameter/environment concern remains future acceptance | Round 2 exact catch/proposed_fix retained verbatim; M0-A5 must prove value/condition parameter/environment parity and distinct captures; no closed-helper restriction or validation assumed |




---

**Document created**: 2026-10-26
**Last updated**: 2026-10-29 — UNSENT M0 evidence amendment after two BLOCKED rounds; no planning/execution approval
