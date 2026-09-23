# Sprint Plan — M-IFC-AUTHORITY-SCOPING

**Design doc:** [../m-ifc-authority-scoping.md](../m-ifc-authority-scoping.md)  
**Sprint ID:** `M-IFC-AUTHORITY-SCOPING`  
**Source:** GitHub issue #752 (`email-parse`)  
**Planned at:** `fc0d4c90` on `coordinator/task-3c9c1ce2`, 2026-09-23  
**Current version:** v0.42.0 (the approved design targeted v0.39.0)  
**Duration:** 3 engineering days, 3 milestones  
**Estimated change:** ~300 LOC (implementation ~150, tests ~150), plus focused docs  
**Risk:** High — security-sensitive type/effect semantics in `internal/types`

## 1. Outcome

Ship exact IFC authority semantics without breaking today's blanket form:

- `Declassify[label=email]` may lower `email`, but not `secret`.
- Effect-row validation uses a Declassify-only covering relation: a caller's
  authority must cover its callee's authority, while Rand/AI remain invariant.
- A positive parameter label is enforced at local call sites: bottom/unlabelled
  and covered labels pass; an incompatible label produces
  `ParamLabelCoverError`.
- Existing bare `Declassify`, secret examples, and IFC fixtures retain their
  current behavior.

Cross-module summaries and enforcement remain owned by
`m-ifc-cross-module-labels.md` and are not part of this sprint.

## 2. Planning Evidence and Constraints

### Current code still matches the design's implementation map

| Concern | Current locus | Confirmed state |
|---|---|---|
| Parameter schema/validation | `internal/types/effects.go` | Rand and AI have closed value sets; Declassify has no schema entry |
| Effect validation subsumption | `internal/types/effects.go` | `effectParamsCompatible` is invariant and feeds `DiffEffectRows`/`SubsumeEffectRows` |
| IFC signature authority | `internal/types/ifc_check.go` | `ifcSig.declassify` is a whole-body boolean |
| Positive-label call checks | `internal/types/ifc_check.go` | `labelOfCall` enforces `{not label}` only |
| Diagnostics | `internal/types/errors.go`, `internal/types/ifc_check.go` | `DeclassifyRequiredError` exists; no `ParamLabelCoverError` |
| User docs | `docs/docs/guides/ifc-labels.mdx`, `examples/runnable/secrets/README.md` | Describe blanket Declassify only |
| Teaching prompt | `docs/docs/prompts/current.md`, generated prompt sources | Describes blanket Declassify; changes require the prompt-manager gate |

### Velocity and capacity

The last 14 days contain only one design-doc commit and no usable LOC history,
so a repository-wide LOC/day figure would be invented. This plan instead uses
the approved design's three-day decomposition and the current code surface:
approximately 100 changed LOC/day, including tests. Re-estimate after M1 if
effect parameter representation cannot encode the documented comma-separated
label set without parser/elaborator work.

### Baseline status

The planning environment has neither `go` nor `make`; attempted baseline commands
returned `rc=127`. This is **UNAVAILABLE**, not green. Before editing production
code, the executor must run and record:

```bash
go test ./internal/types
make check-boundaries
make check-file-sizes
```

If any command is red on untouched HEAD, preserve its output and distinguish the
base failure from sprint regressions. Do not weaken a gate to accommodate an
unrelated base failure.

## 3. Dependency Order and Contract Mapping

| Milestone | Design contract closed | Depends on |
|---|---|---|
| M1 | Scoped syntax validation and Declassify covering subsumption | Green/informative baseline |
| M2 | Scoped IFC Check B, caller behavior, and positive-label Check C | M1 |
| M3 | Backward compatibility, examples/docs, full repository gates | M2 |

The companion cross-module sprint must consume `declassify: [labels]` with `*`
for blanket authority; it must not revert this sprint's representation to a bool.

## 4. Milestones

### M1: Scoped effect surface and authority covering (~100 LOC)

**Goal:** Make scoped Declassify legal and give validation subsumption the exact
security-monotone covering rule, without changing other parameterized effects.

**Files:**

- Update `internal/types/effects.go`.
- Extend the existing effect tests in `internal/types/effects_test.go` (or the
  current colocated schema/subsumption test file; do not create a redundant suite).

**Day 1 tasks:**

1. Write failing tests for accepted `Declassify[label=email]`, rejected unknown
   keys, rejected empty labels, and preservation of Rand/AI closed value sets.
2. Represent Declassify's `label` key as an explicitly open value set. Keep the
   exception local to this effect/key and keep deterministic fail-loud errors.
3. Add Declassify-only covering compatibility: bare covers all; a scoped set
   covers its subsets; `email` does not cover `secret`. Preserve invariant
   comparison for Rand and AI.
4. Verify formatting and deterministic rendering of scoped rows.

**Acceptance criteria:**

- [ ] `Declassify[label=email]` validates; bare `Declassify` remains valid.
- [ ] Unknown Declassify keys and empty/whitespace label values fail loudly.
- [ ] Caller `Declassify[label=email]` covers an email-scoped callee but not a
      secret-scoped callee; bare caller authority covers either.
- [ ] A narrower caller cannot cover a wider or blanket callee.
- [ ] Existing Rand/AI mismatch and default-normalization tests remain unchanged
      and green.
- [ ] `go test ./internal/types -run 'Effect|Subsume|Param'` is informative and green.

**Risk control:** The approved design describes comma-separated label lists but
the current AST parameter map exposes one string value. Lock tests for parsing,
normalization, duplicate labels, ordering, and whitespace before implementing
set comparison. Escalate rather than introducing parser syntax beyond the
already-approved parameter value surface.

### M2: Scoped IFC enforcement and positive-label Check C (~160 LOC)

**Goal:** Replace whole-body boolean authority with exact label authority and
enforce positive parameter labels at every known local call.

**Files:**

- Update `internal/types/ifc_check.go` and `internal/types/errors.go`.
- Extend `internal/types/ifc_check_test.go` and
  `internal/types/ifc_closure_test.go` where closure behavior is exercised.
- Fold or remove the unused whole-row `CheckDeclassify` in
  `internal/types/sink_check.go` only if tests prove it is superseded; record the
  choice in the sprint notes.

**Day 2 tasks:**

1. Add failing fixtures for design verification shapes V2–V7 plus pass controls.
2. Replace `ifcSig.declassify bool` with an authority value supporting none,
   a label set, and blanket/top.
3. Always execute Check B. Permit only leaked constituent labels covered by the
   function's authority; retain declared-return relabeling for authorized callees.
4. Extend `DeclassifyRequiredError` to name leaked and authorized labels and give
   a deterministic remedy.
5. Add `ParamLabelCoverError` and Check C beside Check A in `labelOfCall`.
   Bottom/unlabelled input passes; matching/covered input passes; incompatible
   constituent labels fail.
6. Verify caller propagation and the closure-laundering regression explicitly.

**Acceptance criteria:**

- [ ] Email-scoped authority may lower email and may not lower secret.
- [ ] A caller of an email-scoped declassifier cannot lower an unrelated secret
      in its own body.
- [ ] Unauthorized diagnostics name the hidden label and authorized set.
- [ ] A secret argument passed to `string<email>` produces
      `ParamLabelCoverError`; bottom and email arguments pass.
- [ ] Multi-label joins are rejected if any constituent is not covered.
- [ ] Closure values carrying secret cannot be laundered by email-only authority.
- [ ] Existing `{not label}` Check A diagnostics remain unchanged.
- [ ] `go test ./internal/types -run 'IFC|Declass|Label|Sink'` is informative and green.

**Risk control:** `calleeResultLabel` may still return the declared output only
after Check B establishes that the callee's body used no unauthorized authority.
Do not add a permissive fallback for malformed or missing effect parameters.

### M3: Compatibility, examples, documentation, and repository gates (~40 LOC docs)

**Goal:** Prove backward compatibility, document the opt-in narrowing form, and
run all release-quality gates.

**Files:**

- Verify unchanged: `examples/runnable/secrets/gated_secret.ail`,
  `leak_attempt.ail`, and `secret_demo.ail`.
- Update `examples/runnable/secrets/README.md` and
  `docs/docs/guides/ifc-labels.mdx` with scoped and blanket authority behavior.
- Do not directly edit generated/current teaching prompt artifacts in this
  sprint. Route any prompt update through `prompt-manager` as its own approved
  follow-up because the design explicitly calls out that gate.

**Day 3 tasks:**

1. Run all existing IFC tests and the three secret examples without modifying
   their AILANG source.
2. Add concise docs showing scoped syntax, propagation, positive-label call-site
   enforcement, and the blanket-form hazard.
3. Run targeted tests, full tests, lint, boundary, and file-size checks.
4. Record whether the current teaching prompt needs a separately gated update
   (static inspection says yes) and hand that finding to `prompt-manager`.

**Acceptance criteria:**

- [ ] The three existing secret examples preserve their documented pass/fail behavior.
- [ ] All pre-existing IFC fixtures pass unchanged.
- [ ] Docs distinguish `Declassify[label=...]` from blanket `Declassify` and
      explain that positive labels constrain local call arguments.
- [ ] No cross-module IFC summary/enforcement is added.
- [ ] `go test ./internal/types` passes.
- [ ] `make test`, `make lint`, `make check-boundaries`, and
      `make check-file-sizes` pass, or an independently recorded pre-existing
      failure is reported without being hidden.

## 5. Refusal-Branch Verification

The executor must demonstrate that tests fail when each security check is
temporarily neutralized, then restore the implementation:

| Branch | Temporary mutation | Expected killer |
|---|---|---|
| Open schema exception is Declassify-only | Treat another effect's value set as open | schema rejection test |
| Scoped covering rejects unrelated labels | Make all Declassify params compatible | effect subsumption test |
| Scoped Check B rejects unauthorized leaks | Skip Check B when any authority exists | secret-under-email fixture |
| Check C rejects cross-label calls | Disable positive-label branch | secret-to-email fixture |
| Check C permits bottom | Reject every nonmatching representation | unlabelled-to-email pass control |

These mutations are verification only and must not remain in the final diff.

## 6. Success Metrics

- All six design success criteria are mapped to M1/M2 tests.
- At least one pass and one reject test for each of: schema validation,
  authority covering, scoped Check B, caller propagation, and Check C.
- Existing IFC and closure tests stay green without weakening assertions.
- Existing AILANG secret example files remain unchanged and verified.
- Documentation is updated; teaching-prompt work is explicitly handed through
  its separate gate rather than silently editing generated artifacts.
- No new parser grammar, runtime IFC, cross-module summaries, or bare-form lint.

## 7. Executor Handoff

This plan is ready for user review. Implementation remains gated: the user must
explicitly say **"execute sprint"**, after which `sprint-executor` owns TDD,
progress updates in the JSON file, and final `sprint-evaluator` handoff.

