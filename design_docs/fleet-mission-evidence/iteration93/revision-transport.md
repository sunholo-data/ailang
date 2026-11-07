# Single GLM revision — artifact preserved, helper interrupted

Author: `pi:ollama/glm-5.3:cloud`, same author as round one (no new rotation turn). Artifact SHA-256: `bad5f6a2f40a1f6bdcca2495be6b4b92c0e31386d61822e4e2de7341375e1eac`, 698 lines. The production worktree is outside `/tmp`; its HEAD stayed at `a3bb8629106a4b8459df5aa33f867b4ed4dd84a5`, and only the owned design file changed.

The substantive revision added immutable flip-blob claim binding and its mutation matrix, removed the independence and model-behavior seeds, dry-evaluated the remaining World text claims (two real findings until #2082), and bounded paths/corpus/provenance reads. It corrected four-column and axiom-score facts. No policy, implementation, ruling, or runtime configuration changed.

The controller subsequently relaxed its initial line-cap preference, but print-mode had consumed its directive and could not receive that steering. The model repeatedly trimmed the otherwise written revision. The controller explicitly stopped that run rather than claiming the formatting churn was productive completion. SIGTERM at `2026-11-07T05:49:13Z`; process-group/helper/node/filter were absent on the post-exit `ps` check. The artifact is byte-identical to the saved stopped snapshot; the bridge did not edit content.

Canonical helper: raw `launch_failed`, rc 14, fenced true, elapsed 589 s, 67 tool executions, zero provider errors, zero `agent_end` events. This is an operator/transport interruption, not a dead-model fallback or successful helper verdict. Its own fresh protocol acknowledgment preceded writes. The same substantive revision is reviewed once; no third author round is authorized by this run.

Reported completed-call usage: probe 1,837 tokens; revision 7,518,102 tokens (221,813 input + 148,769 output + 7,147,520 cache read), 58 completed revision calls. Combined 7,519,939 tokens, 370,883 noncached input/output. Unfinished inference usage is unknown. Flat Ollama Cloud metered cost $0; unknown quota usage is not invented as zero. Initial authoring used another 6,812,677 provider-reported total tokens, so the known GLM total is 14,332,616 including cache reads.

## Controller findings retained for re-quorum

- The canonicalization specifies excluding line terminators and reconstructing LF, yet says newline style changes identity; M16/M20 also demand a CRLF mismatch. The byte contract needs one answer.
- The revision demands ORIGINAL flip binding and new-ID-only claim revisions, while the unchanged helper explicitly selects the LATEST verified non-RESOLVED→RESOLVED transition. A reopen/re-resolve needs an unambiguous contract; a current-row prose rule does not enforce one.
- The revision reads registry `workdir` strings and concludes only V1 shares this repository. `tools/launchd/lib/pin-root.sh` lines 131–143 compare repository origins and repoint same-repository missions to their pin worktree. Distinct clone paths do not establish distinct repositories. World/Stapledon remain truly external, but the stronger V1-only assertion is unsupported. No automatic external-repo wake is promised.
- M1 says deleting a claim yields no finding, while M15 says post-approval deletion yields authorization mismatch. The fixture-history preconditions must distinguish these arms.

These are controller review findings about a draft, not claims that an implemented checker is exploitable. Re-quorum and the final independent judge assess the frozen artifact; neither may be represented as author approval.

Runtime-root control: sourcing `tools/launchd/lib/pin-root.sh` in an isolated Bash and calling `_pin_same_repo <mission workdir> <source>` returned 0 for docs, motoko and V1; World returned 1. This is the helper’s own same-origin test, paired with both same/different controls. Separate clone paths therefore do not support V1-only repository identity. No helper state or live files were changed.
