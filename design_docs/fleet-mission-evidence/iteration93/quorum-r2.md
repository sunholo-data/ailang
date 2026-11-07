# Design quorum round 2 — BLOCKED / needs human review

Frozen draft SHA-256: `bad5f6a2f40a1f6bdcca2495be6b4b92c0e31386d61822e4e2de7341375e1eac` (698 lines).

Fresh scratch CLI built and stamped at `a3bb8629106a4b8459df5aa33f867b4ed4dd84a5`; shared installed binary unchanged. Same GLM author vendor benched. One substantive revision, one re-quorum; no third author round or implementation authorized.

CLI rc 3; synthesis **blocked**. All three present external reviewers rejected; Anthropic absent/quota, OpenAI reserve served. Cost $0.256186000; tokens in 51524, out 6593.

## gemini-4-argon

Present: True; input 18016; output 614; reported cost $0.084344000.

**verdict**: reject

**strongest_objection**: The canonical claimset extraction is all-or-nothing per ruling across unrelated rulings in the same ```fleet-rulings fence, creating a cross-ruling denial-of-enforcement failure mode. The spec states: 'locate the single ```fleet-rulings fenced block (zero fences = empty set for every R; >1 fences or an unparseable claim line for R -> rulings-approved-blob-unavailable for R, fail-closed); take the lines that parse as valid claim lines (comments/blanks excluded) whose ruling ID equals R'. Because a syntax error on a line might damage or omit the ruling ID token itself (e.g., a typo'd ID `D-FR1` or a missing ID at the start of a line), the parser cannot attribute an arbitrary unparseable line inside the shared fence to a specific ruling R. Either an unparseable line with a mangled ID is silently ignored for all rulings (violating A11 / fail-closed and allowing malformed edits to slip by), or any unparseable line in the shared fence invalidates the entire block and disables enforcement for ALL previously resolved rulings R1..Rn simultaneously.

**catch**: In 'Canonical claimset for ruling R in blob B', specify how line-level attribution to R works BEFORE full claim-line syntax parsing, or isolate each ruling's claims so a syntax error in a newly drafted OPEN ruling's claim line cannot either be silently ignored or break enforcement of already-RESOLVED rulings.

**proposed_fix**: Replace the canonical claimset extraction rule in '### Claim-Set Authorization Binding' with a two-stage prefix attribution rule: 'Every non-blank, non-`#`-comment line inside the single ```fleet-rulings fence MUST begin with a whitespace-delimited token matching `^D-FR-[0-9]+$`. If ANY line in the fence fails this prefix check (or if >1 ```fleet-rulings fences exist), the fence is structurally malformed: emit `rulings-malformed` at the ledger level and fail-closed on the entire file in HEAD (whilst in an approved blob B, treat as `rulings-approved-blob-unavailable`). Once prefix-attributed to ruling ID R, only lines whose first token equals R are inspected when evaluating R: if any line attributed to R fails full claim grammar parsing (`<verb> "<literal>" [except "<literal>"] scope=<surface>`) or exceeds byte bounds, fail closed for R (`rulings-malformed` at HEAD, `rulings-approved-blob-unavailable` in blob B) without invalidating other rulings R' != R. The canonical claimset for R is computed over R's valid attributed lines in file order.'

## oc-kimi-k3

Present: True; input 16681; output 5671; reported cost $0.135108000.

**verdict**: reject

**strongest_objection**: The authority core — the section R1 demanded and Freeze item 2 hinges on — rests on an unverified CLI contract: the design invokes `mission_ledger_provenance.sh --id R --file design_docs/fleet-rulings.md` and promises the helper is 'reused verbatim, byte-unchanged' (with '0 changes to any of the four scripts/mission_*.sh helpers' as a success metric), yet V33 only *read* the helper (the Observed column cites its stdout format and :102-104, not a run) and never demonstrates it accepts `--file` pointing at a non-charter ledger or that it emits a flip for a file outside the per-mission charter path it was built for. The one live run attempted in the activation chain, `mission_answer.sh --dry-run` (V9), was blocked by the sandbox and is nonetheless cited by a Success Criterion as evidence that the attended channel 'accepts a D-FR-* ID.' So Freeze item 3's activation step and the binding's enforcement precondition have never been executed end-to-end, even in dry-run; if either helper contract fails, the binding cannot be built as specified without violating the doc's own zero-helper-change metric.

**catch**: Two unverified load-bearing premises to close before quorum: (1) run `scripts/mission_ledger_provenance.sh --id D-FR-1 --file <scratch-ledger>` and `scripts/mission_answer.sh --dry-run --id D-FR-1 --file <scratch-ledger>` against a scratch fleet-rulings.md in a throwaway repo (outside the blocked sandbox, with a recorded non-fleet identity), and add the observed stdout/exit codes as V-rows; also fix the V5 inconsistency — `Mission.Root()` is at registry.go:90 while :251 is where `LoadFile` sets `root`, and it is unverified whether the `missionDoctor` path loads via `Load` (:261) in a way that populates `root`, which matters because an empty root makes the ledger path cwd-relative and silently disables the check (the M8 failure mode the doc claims to kill). (2) The Success Criterion citing V9 currently asserts acceptance that was never observed — either re-run it or downgrade the criterion's evidence.

**proposed_fix**: Add a verification-log row and amend V9: 'V9b: `mission_ledger_provenance.sh --id D-FR-1 --file /tmp/scratch-fleet-rulings.md` run in a scratch git repo (committed, non-shallow, non-fleet author, OPEN→RESOLVED flip): observed `<id> verdict=attended flip=<rev> creation=<rev>`, rc 0 — the helper accepts an arbitrary `--file` ledger unchanged. V9c: `mission_answer.sh --dry-run --id D-FR-1 --file <scratch>` observed rc 0 with row mutated in dry-run output (run outside the sandbox that blocked V9).' Amend V5 with: '`Load` (:261) delegates to `LoadFile` (verified by call-graph read at :2XX), so `Mission.Root()` is populated for every registry-loaded mission in the `missionDoctor` path; the wiring citation is registry.go:90 for the accessor and :251 for the field assignment.' If the helper turns out not to accept `--file`, replace the '0 changes to helpers' success metric with an explicit, minimal helper-touch scope and a justification for why that is preferable to plumbing a wrapper — do not leave the binding specified against an unrun contract. Also change the Success Criterion wording from 'mission_answer.sh --dry-run accepts a D-FR-* ID (V9)' to reference the new live-run row.

## claude-sonnet-5@claude-p

Present: False; input 0; output 0; reported cost $0.000000000.

## gpt6-1-sol

Present: True; input 16827; output 308; reported cost $0.036734000.

**verdict**: reject

**strongest_objection**: The authorization binding can silently disappear when the last claim for a resolved ruling is deleted. The design schedules provenance checks only for 'resolved-with-claims' rows and caps IDs 'referenced by claims.' After deleting all claims for D-FR-1, that ID no longer enters the binding check, so no approved blob is loaded and no rulings-authorization-mismatch is reported. This contradicts M15 and the no-silent-fallback guarantee.

**catch**: M15 must explicitly delete the entire claimset, including removal of the whole fleet-rulings block. Checking deletion of one claim while another remains does not exercise the candidate-discovery hole.

**proposed_fix**: Replace claim-driven authorization discovery with: 'Enumerate every RESOLVED ledger row independently of the current claims block. For each row, obtain provenance and compare its HEAD canonical claimset, including an empty set, against its approved flip-commit claimset before deciding whether there are claims to execute. Apply the 32-row bound to all RESOLVED rows; exceeding it emits a typed oversize finding rather than silently truncating checks.' Add integration tests deleting the sole claim, deleting every claim for one ruling while retaining another ruling's claims, and removing the entire claims block after approval; each must report rulings-authorization-mismatch for every affected ruling and enforce none of its claims.

## Controller verdict

reject: Controller independently rejects remaining contract ambiguities in this revised draft. Canonicalization excludes line terminators yet says newline style changes its digest; specify one byte-identity contract. It says original flip binding and new-ID-only revisions, but the unchanged provenance helper returns the latest non-RESOLVED-to-RESOLVED flip; make the acceptance and revision rules consistent. The cross-repo section infers runtime workdirs from registry paths and says only v1 shares this repo; pin-root.sh _set_pin_workdir repoints same-origin missions including docs/motoko to their committed pin, so validate runtime claims against that path. These are draft review findings, not authorization for a third author revision or code. Review independently.

The claim-discovery deletion hole is a concrete unresolved blocker. Human freeze/direction review follows; this artifact is not an approved sprint input. No other ruling was resolved.
