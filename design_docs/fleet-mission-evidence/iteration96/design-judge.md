# Fleet Iteration 96 — Independent Final Design-Judge Report

## Classification

This was a **report-only independent design-proposal evaluation**. It was **not** a sprint-implementation evaluation: there is no approved design, no sprint plan, no implementation commit, and no acceptance evidence. The implementation score is therefore **N/A**.

The reviewed proposal is an uncommitted, frozen design artifact:

- Path: `design_docs/planned/m-fleet-role-artifact-contract.md`
- Reviewed HEAD: `2249a1923a02be2e4f3ce2f1dcb96239484a237b`
- Size: 25,214 bytes, 199 lines
- SHA-256 before review: `df93fee6c072bbdc19850126963049a50ff7df7cd3db13cc573a08e4fd859998`
- SHA-256 after review: `df93fee6c072bbdc19850126963049a50ff7df7cd3db13cc573a08e4fd859998`
- Frozen-artifact stability: **UNCHANGED**

## Session protocol

1. The first tool action was a complete read of `CLAUDE.md` (152 lines).
2. No `ailang messages` command was run: this was an unattended mission stage.
3. The truthful work classification was report-only independent design-proposal evaluation. No approval was inferred from the controller's supplied context.
4. `session_protocol_ack({})` was called after the required read/classification step. The tool returned: `Session protocol acknowledged — mutating tools unlocked for this session. Work routing per AGENTS.md still applies.` The tool rendered text only; the transport-visible structured acknowledgement is not reproduced or invented here.
5. Only after that acknowledgement was report work performed.

## Executive judgment

- **Technical decision:** **FAIL — NEEDS REVISION**.
- **Gated execution:** **BLOCKED**, independently of the technical defect.
- **Implementation score:** **N/A** (no implementation exists).
- **Technical soundness:** The P1 instruction-only direction is coherent, appropriately narrowed, and materially improved. It resolves every original quorum objection except that the frozen revision still makes an unsupported universal policy-compliance claim and does not fully document the applicable-policy conflict surface.
- **Gated execution:** A technical pass or quorum pass would not approve this design. The strict Duplicate/Coverage gate is unresolved, the terminal re-quorum is blocked, and explicit attended human approval has not occurred. No sprint plan or execution may follow from this report.

## Strongest objection

The proposal says the original directive was **“conformant with every standing rule”** and that the observed broad discovery was permitted by every standing rule, while its negative evidence (R10) only greps one recipe file for the word `identity`. That does not establish either the complete applicable instruction hierarchy or whether some other standing rule already restricted filesystem/session discovery. The full original directive is not among the named evidence; only excerpts are.

My bounded independent source review found **no generic read/discovery prohibition** in the surfaces inspected: full `CLAUDE.md`, full `gate-3-route.md`, full `codex-lane-false-greens.md`, targeted `AGENTS.md`, targeted `.claude/rules/**`, and the relevant planner/sync source. The false-greens document concerns destructive writes outside the worktree, not the observed read sweep. This supports the narrower claim that the inspected mission-control recipe lacks a preventive discovery clause. It does **not** prove the universal “every standing rule” claim.

Required correction: narrow the premise to the inspected policy surface, inventory the applicable policy sources, and add a conflict-surface statement saying whether the proposed block specializes or changes any existing rule and which rule has precedence. This is a prose/evidence correction, not a request for runtime enforcement or a high-impact implementation test.

## Original quorum objections — status against the frozen revision

### `gpt6-1-sol` (r1)

**Objection:** The claimed 100% real-directive conformance metric was not established by synthetic test fixtures.

**Status: RESOLVED in the frozen revision.** The new test suite, synthetic directive fixtures, population-level conformance metric, and prevention/compliance claims were removed. The success metrics now cover recipe-text presence, byte parity, preservation of the rc18 wording, and existing launchd-driver checks. The design explicitly says actual controller-composed directive conformance is unmeasured and that no obedience or prevention rate is claimed.

### `gemini-4-argon` (r1)

**Objection:** A test-local checker against a hard-coded synthetic directive is tautological and proves nothing about production composition.

**Status: RESOLVED in the frozen revision.** The design no longer claims production-directive linting or synthetic fixtures as evidence of real conformance. Runtime checker wiring remains out of scope, and real-composition linting is explicitly deferred. The proposal no longer conflates recipe-text conformance with production composition.

### `oc-glm-5-3` (r1), three sub-objections

1. **Handshake identity/session-metadata adjacency was not read.**  
   **Status: RESOLVED.** The frozen design records the complete handshake content and its `0` identity/metadata mention count; the proposal places the new block after `PI_EVALUATOR_SESSION_HANDSHAKE_END`, preserving the exact preamble and protocol. I independently read the current handshake and confirmed that adjacency.

2. **Planner-lane derivation/path analysis was asserted rather than observed, and the prior `make/test.mk` path was also outside the default planner allowlist.**  
   **Status: RESOLVED.** The field is the bare `opus-required`; the recorded post-rewrite derive output is distinguished as a declared lane plus configured Anthropic-unavailable fallback. The revised scope removes `make/test.mk`, and both proposed recipe-copy paths remain outside the default planner allowlist without widening it.

3. **The sibling slot-kill plan overlaps `make/test.mk`.**  
   **Status: RESOLVED.** The revision acknowledges the sibling's actual `make/test.mk` row and removes its own `make/test.mk`/suite change, eliminating the merge-order surface.

### Controller-in-session note (r1)

**Objection:** Do not claim synthetic fixtures prove real conformance; distinguish configured identity from observed metadata; do not self-resolve human freeze choices; recognize that the original directive already named inputs/output and that the N=1 recovery is correlation, not causal proof.

**Status: RESOLVED.** All five corrections are explicit in the frozen revision: real conformance is not measured; identity has DECLARED/RESOLVED, OBSERVED, and unavailable provenance labels; every High-Impact and Design Freeze row remains proposed/unapproved and unchecked; the original named inputs/output are recorded; and the one changed-variable retry is described as N=1 correlation with no prevention proof.

## Terminal re-quorum (`quorum-r2.json`)

The named terminal file was checked **once, after the independent technical review**, and was present. It contains two present reviewer results, one quota-absent reviewer, and synthesis verdict `blocked`. I did not inherit its conclusion; the policy-evidence concern above had already been derived from the frozen log and source.

- **`gpt6-1-sol` r2 objection — standing-policy gap and conflict surface not verified.**  
  **Status: OPEN.** Independently upheld for the reason stated under “Strongest objection.” My targeted review found no contrary rule in the inspected surfaces, but that does not establish the frozen document's universal claim. The revision needs a bounded applicable-policy inventory/conflict-surface correction.
- **`gemini-4-argon` r2 objection — pointer placement at `~line 355` lacks an exact verification row.**  
  **Status: OPEN, MINOR / NON-BLOCKING BY ITSELF.** Current source lines 351–355 are the pi “Real executor run” heading and the start of its invocation block, so the anchor is plausible. The frozen document still lacks an exact-context verification row and should replace the approximation when corrected.
- **Controller r2 pass:** Not approval. It explicitly leaves the Duplicate/Coverage gate unresolved and requires attended approval.

## Independent technical findings

### 1. P1 scope and proportionality

The scope is proportionate to a P1 prose defect: one marked contract block, one pointer, two appended MUST sentences, two mirrored recipe files, and existing sync/parity checks. The estimated one phase and ~1.5–2 hours are plausible. The design does not inflate a reversible recipe edit into new runtime machinery.

### 2. Prevention instruction versus proved prevention

This boundary is handled correctly. The incident and retry facts are separated; the retry changed several variables at once; N=1 is called correlation rather than causal proof; report shape and rc 0 are not substantive acceptance; and no prevention rate or obedience guarantee is claimed. The proposed clause is correctly described as preventive **instruction**, not as proved prevention.

### 3. P2 exclusion

P2 is fully excluded from implementation, acceptance criteria, and changed paths. The proposal chooses no cap value, threshold, environment knob, extension load, runtime wiring, lane, retry count, quota, ration, or billing behavior. The `(cap, stall]` semantics and controller-versus-role wording are explicitly left to a separate attended decision.

### 4. Identity provenance

The DECLARED/RESOLVED versus OBSERVED distinction is sound. A configured lane is not backend identity attestation; OBSERVED requires transport-captured response metadata with a known NDJSON path; and even OBSERVED is not weights attestation. The contract forbids deriving identity from session stores, `$HOME`, metadata trees, or environment values. This evaluator report makes no backend-attestation claim.

### 5. Named INPUT versus OUTPUT and missing inputs

The proposal correctly distinguishes exact INPUT paths, exact OUTPUT paths, and expected-absent OUTPUT paths. A missing required INPUT must be reported against its supplied path and cannot authorize a search elsewhere. This is a specific, usable prose rule and does not depend on runtime enforcement.

### 6. Adjacency and settled rc18 wording

The proposed placement is after the handshake END marker, so the exact evaluator preamble and protocol remain intact. The rc18 extension says to retain every existing sentence verbatim and append requirements to name the forbidden `hung_tool` command and restate identity disposition when relevant. Current source confirms the settled one-retry wording is present before any future edit. No settled retry count or fallback rule is changed.

### 7. Paths, allowlists, and extensions

The two proposed recipe paths are inside the fleet Authority allowlist. No extension path is changed, including `.pi/extensions/**`; no `tools/pi-extensions/**` path is changed; and no new runtime hook is introduced. The mirror files are currently byte-identical, and the existing sync script supports `--check` and is already wired into `test-launchd-drivers`.

### 8. Authoring provenance

The document distinguishes the quota-interrupted Kimi partial from the GLM fallback continuation, does not convert the interrupted session into a completed verdict, and labels configured/probe metadata rather than backend attestation. The Verification Log also distinguishes Kimi-session observations from controller-supplied proof.

### 9. Remaining technical defect

The universal standing-policy conformity claim is not established by R10 and is not fully closable from the named original evidence. This is the one material technical defect in the frozen proposal. It is localized and correctable in prose/evidence; it does not invalidate the instruction-only direction.

### 10. Duplicate/Coverage gate

The strict threshold was not self-resolved. The captured planned-doc semantic matches are 1.00, 0.95, and 0.90, all at or above 0.75. The proposal documents their apparent scopes but does not grant itself an “irrelevant match” exception. The correct state is **UNRESOLVED / BLOCKED**, requiring an operator/human ruling.

## Human and workflow gate status

- **Design status:** Planned; not approved.
- **High-Impact Decisions:** all explicitly PROPOSED / UNAPPROVED; human owns the choice.
- **Design Freeze:** every row unchecked; no self-approved human freeze.
- **Duplicate/Coverage gate:** UNRESOLVED / BLOCKED; strict `>= 0.75` result remains unexcepted.
- **Terminal re-quorum:** `quorum-r2.json` present; synthesis `blocked`.
- **Explicit attended human final-design approval:** absent.
- **Sprint plan:** absent.
- **“execute sprint” authorization:** absent.
- **Implementation / acceptance evidence:** absent.
- **Consequence:** No planning, implementation, or acceptance may proceed from this review. A corrected design would still require the duplicate ruling, a passing quorum, explicit human approval, a sprint plan, and the normal execution gates.

## Source observation limits and side-effect statement

- Reads were bounded to the named worktree files plus directly relevant worktree source. No `HOME`, session-store, session-metadata, credential, or environment-value research was performed; no environment dump was printed.
- No source code, decision, ticket, Git state, or earlier evaluator report was modified.
- No test, lint, build, mutation drill, or implementation acceptance command was run. The existing sync/parity wiring was inspected, not executed.
- The original directive was available only through the named excerpts, not as a complete directive; therefore this review could not attest the universal “every standing rule” claim.
- The terminal quorum was not used to bootstrap this review; it was opened only after the independent technical review.
- The frozen design remained byte-identical throughout review. The only writes made by this evaluator are these two report paths.

EVALUATION_VERDICT: FAIL reasons=the frozen proposal retains an unsupported universal policy-compliance claim and lacks a complete applicable-policy conflict-surface proof; independently, the strict Duplicate/Coverage gate remains unresolved, the terminal quorum is blocked, and explicit human design approval is absent
