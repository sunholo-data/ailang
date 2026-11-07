# Design quorum round 1 — BLOCKED

Draft SHA-256: `923125adf34a557e9c789eea872c859215c98bd5b1d3758d15976681ba9adcf6`.

Actual author `pi:ollama/glm-5.3:cloud`; its vendor was benched. CLI rc 3. Controller rejected independently. Three external reviewers responded and all rejected; Anthropic was absent (quota), replaced by GPT-6.1 Sol. Reported cost $0.219057000.

The first call used installed CLI commit `f20e6dea2fac8e92387a6ab5f612ed7749574422`. It warned on source mtimes. Controller checked the delta to `a3bb8629106a4b8459df5aa33f867b4ed4dd84a5`: quorum Go sources are identical; the registry differs only by two unused Gemma fine-tune model entries. The seated/replacement reviewer definitions are unchanged. Re-review uses a freshly stamped scratch binary built at the full base SHA; the shared installed binary is untouched.

## gemini-4-argon

Present: True; input 13438; output 537; reported cost $0.064492000.

**verdict**: reject

**strongest_objection**: The seed claim `D-FR-2 require-text "gpt-6-astra" scope=skills` is a guaranteed self-failing trap as specified. Under 'Bounded Checker Semantics', the doc defines `require-text "<lit>"` as: 'the literal must appear in EVERY in-scope file' and defines `scope=skills` as `SKILL.md` + `resources/*.md` across both `.claude` and `.agents` mirrors. `gpt-6-astra` appears only in `resources/gate-3-route.md:263`, not in `SKILL.md` or any other `resources/*.md` file. The moment D-FR-2 is RESOLVED, `ailang mission doctor` will emit `ruling-missing-link` / missing-precondition drift findings for every other skill file.

**catch**: Two contradictions in the specification: (1) `require-text` with `scope=skills` requires the literal in every file in `skills`, which breaks `D-FR-2` (either `D-FR-2` must use `scope=file:.claude/skills/mission-control/resources/gate-3-route.md` plus its `.agents` mirror, or `require-text` needs an `any-in-scope` vs `every-in-scope` distinction). (2) Under Solution Design -> Overview, the doc claims the decision ledger has '5 columns', whereas the example markdown table and Verification Log V23 both show a 4-column table (`| ID | Status | Decision / recorded answer | Evidence |`).

**proposed_fix**: In `design_docs/fleet-rulings.md` seed claims block, replace:
`D-FR-2 require-text "gpt-6-astra" scope=skills`
with:
`D-FR-2 require-text "gpt-6-astra" scope=file:.claude/skills/mission-control/resources/gate-3-route.md`
`D-FR-2 require-text "gpt-6-astra" scope=file:.agents/skills/mission-control/resources/gate-3-route.md`
And in Solution Design -> Overview, change '5 columns' to '4 columns' to match V23 and the markdown table.

## oc-kimi-k3

Present: True; input 12334; output 5975; reported cost $0.126627000.

**verdict**: reject

**strongest_objection**: The design's entire mechanism is exact-literal matching, yet none of the 27 Verification Log rows checks the match status of the three shipped seed claims against the corpus they target — and one seed claim is broken on its face under the doc's own verb semantics. The doc defines require-text as "the literal must appear in EVERY in-scope file," then ships `D-FR-2 require-text "gpt-6-astra" scope=skills`, where the skills surface is SKILL.md + resources/*.md + the .agents mirrors (~10+ files). The doc's own evidence places that literal at exactly one location (gate-3-route.md:263), and V15 verifies only that the mirrors are byte-identical, not that the literal occurs in every resources file. The moment Mark resolves D-FR-2 — the doc's designated activation path — the doctor fires missing-precondition findings across nearly every skill file: guaranteed self-inflicted noise from the shipped seed, contradicting both the Risk-table claim that activation noise equals real propagation debt and the success metric that consumers "see no behavior change." Compounding it, D-FR-2 is a behavior claim ("never spawned in any role") enforced by text presence, which violates the design's own stated boundary: "Behavior claims about code are for tests, not text matching." A doc whose core premise is verified exact-literal checking ships its only concrete claims unverified, and one unverifiable-by-construction.

**catch**: Add the missing verification: no V-row (and no test in the mutation matrix) dry-evaluates the seed claims against the live corpus under a hypothetical RESOLVED status, so the doc cannot promise activation is quiet-by-default. Also a smaller factual slip in the same section: The Ledger File overview says "5 columns," but the example table and V23 both show a 4-column header (ID, Status, Decision/recorded answer, Evidence) — an asserted premise contradicted by the doc's own verification row.

**proposed_fix**: (1) Replace or re-scope the D-FR-2 claim: either delete it (leaving 'gpt-6-astra never spawned' to a code/allowlist test, per the doc's own behavior-claims boundary) or, if documentation presence is the intent, narrow it to `require-text "gpt-6-astra" scope=file:.claude/skills/mission-control/resources/gate-3-route.md` plus the corresponding .agents mirror path. (2) Add a Verification Log row and a Success Criterion: "each seed claim, evaluated against the live corpus as if its row were RESOLVED, yields exactly zero findings" — implemented as a doctor test fixture seeded from a snapshot of the real corpus, complementing M2's OPEN-is-inert arm. (3) Do the same quiet-on-activation check for D-FR-3's `forbid-text "never re-execs from the driver pin" scope=registry` against missions/*.toml. (4) Correct "5 columns" to "4 columns" in The Ledger File overview to match the example table and V23.

## claude-sonnet-5@claude-p

Present: False; input 0; output 0; reported cost $0.000000000.

## gpt6-1-sol

Present: True; input 12419; output 310; reported cost $0.027938000.

**verdict**: reject

**strongest_objection**: Human resolution does not actually control claim activation: claims live outside the decision-ledger markers, and activation checks only the referenced row's RESOLVED status. After Mark resolves D-FR-3, an unattended edit can add or change claims referencing D-FR-3 and immediately activate new enforcement without another attended ruling. This violates explicit authority and contradicts the assertion that resolution is the only activation path.

**catch**: V8 establishes that the existing validator ignores the claims block; V24 verifies identity guards on row resolution, not authorization of subsequent claim edits. The document provides no verified mechanism binding the exact enforced claims to the human-approved ruling.

**proposed_fix**: Bind enforcement to an immutable, human-approved claim payload. For example, have each attended resolution record a digest of a precisely specified canonical claim set in the provenance-protected decision row. Doctor must recompute and verify that digest before evaluating any claim; additions, edits, or deletions must produce a typed authorization-mismatch finding rather than activate altered enforcement. Claim revisions require a new attended approval through a verified protected path. Add tests that resolve a row, then independently add, modify, and delete claims without changing the row, and verify that none of those altered claim sets is enforced. Verify that the existing provenance machinery protects the digest-bearing approval; if it does not, explicitly revise the claimed unchanged-machinery scope.

## Controller verification before revision

- Existing validator accepts original/modified/added/deleted fenced claims with the exact same synthetic RESOLVED row: all four rc 0, `decision ledger valid: 1 rows`. This is a format-validation proof, not a claim about an implemented checker. No real ruling or attended identity was created.
- Provenance helper lines 102–104 explicitly select the latest status transition and exclude resolved-to-resolved rewrites. A digest stored only in a mutable current row does not close the approval gap.
- Idle fingerprint reads `MISSION_WORKDIR` git history. Registry entries place World and Stapledon in other repos; a fleet-ledger commit cannot alone change their head fingerprint.
- `ref-drift.md` contains no banned model name while its known header exists; requiring that name in every skill file is a guaranteed false finding. It also fails to represent a behavioral ban.
- Twelve axiom rows sum to +6, while draft says +4. Example and existing validator use four columns, not five.
- Operator request makes independent evaluation non-negotiable in this run; a new global preference seed must not weaken it.
