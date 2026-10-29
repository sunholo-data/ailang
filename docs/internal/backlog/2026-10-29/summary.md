# Attended backlog sweep — 2026-10-29

Status at11:26UTC: **15 AILANG PRs merged,6 redundant PRs retired,9 issues closed**. Every issue in the initial66-item inventory was checked against source/history/designs;57 remain open. The initial issue evidence below is a historical snapshot, not a claim that subsequently repaired issues remain broken. All GitHub access used `gh`. The [initial per-issue verification](initial-issue-verification.md) preserves its original source baselines; the [operator ownership review](operator-pending-review.md) records the existing cards and cross-plane holds.

## Accepted PRs

| PR | Result and evidence |
|---|---|
| [1904](https://github.com/sunholo-data/ailang/pull/1904) | Consolidated triage history; preserved evidence from redundant carriers. |
| [2003](https://github.com/sunholo-data/ailang/pull/2003) | Node-forge dependency update. |
| [1951](https://github.com/sunholo-data/ailang/pull/1951) | Actions update. |
| [1992](https://github.com/sunholo-data/ailang/pull/1992) | User-approved mirrored evaluator wording preserves generated-Go checks explicitly required by approved codegen/--verify-contracts scope. |
| [1957](https://github.com/sunholo-data/ailang/pull/1957) | Blocker research evidence only; M1 remains failed0/7, no feature enablement. |
| [1809](https://github.com/sunholo-data/ailang/pull/1809) | Firebase dependency update. |
| [1962](https://github.com/sunholo-data/ailang/pull/1962) | Gate0 watermark crash safety; focused33/33 shell checks, parallel runner preserved; closed981. |
| [1889](https://github.com/sunholo-data/ailang/pull/1889) | User-approved triage recommendation correction: three-line std/list.all estimate exceeds two-line direct-fix rubric. Report only. |
| [2004](https://github.com/sunholo-data/ailang/pull/2004) | User-approved Vertex GA TTS default, retaining AIStudio preview/overrides, and Faber Romanus comparison; closed1949/1902. |
| [1964](https://github.com/sunholo-data/ailang/pull/1964) | Package authority confinement; actual new coverage82%; closed1720. Optional PATH-security warning remains visible/audited, no Sonar disposition or policy change. |
| [1963](https://github.com/sunholo-data/ailang/pull/1963) | Nested callback effect charging; independent93/100, full clean native gate, required remote gates; closed1718/1826. Separate alias ownership follow-up remains active below. |
| [2006](https://github.com/sunholo-data/ailang/pull/2006) | User-approved evaluator grep-zero JSON repair in both mirrors; four parsed real JSON-emission fixtures. Fixture make commands were stubs, not product-test evidence. |
| [2007](https://github.com/sunholo-data/ailang/pull/2007) | Approved1719 IFC function-value sinks/wrapper summaries; independent90/100, full native gate,26-case CLI/REPL matrix, frozen corpus. Required CI retry exposed only independently owned1926 PTY race. Closed1719. |
| [1849](https://github.com/sunholo-data/ailang/pull/1849) | Package asset snapshot residual repair, including real CallPreserveFloats malformed-lock bypass; independent round3 PASS95 after preserved FAIL51/68, full native gate, actual Sonar94.9%. No linked issue auto-closed. |
| [2005](https://github.com/sunholo-data/ailang/pull/2005) | Approved1958 row soundness plus separately approved explicit-empty callback amendment; independent95/95 (initial97/98 preserved with explicit rubric corrections), full native plus final test-only overlay and exact-head Linux/Windows/required CI, actual Sonar93.5%. Consumer fork083ef437:60core/9extensions, fresh no-cache60 after three recorded serial deadline retries. Closed1958. |

The six retired redundant carriers are [1571](https://github.com/sunholo-data/ailang/pull/1571), [1832](https://github.com/sunholo-data/ailang/pull/1832), [1833](https://github.com/sunholo-data/ailang/pull/1833), [1893](https://github.com/sunholo-data/ailang/pull/1893), [1881](https://github.com/sunholo-data/ailang/pull/1881), and [1868](https://github.com/sunholo-data/ailang/pull/1868). Source branches and useful historical evidence were preserved.

Closed issues: [981](https://github.com/sunholo-data/ailang/issues/981), [1711](https://github.com/sunholo-data/ailang/issues/1711), [1718](https://github.com/sunholo-data/ailang/issues/1718), [1719](https://github.com/sunholo-data/ailang/issues/1719), [1720](https://github.com/sunholo-data/ailang/issues/1720), [1826](https://github.com/sunholo-data/ailang/issues/1826), [1902](https://github.com/sunholo-data/ailang/issues/1902), [1949](https://github.com/sunholo-data/ailang/issues/1949), [1958](https://github.com/sunholo-data/ailang/issues/1958). #1711 had explicit closure approval and verified cloud implementation, merged Daneel migration and live site-rda grant; separate credential-isolation work remains tracked.

## Final repairs in progress

- Alias ownership follow-up to1963: transparent local/imported aliases hide callback tails from binder ownership.15 production lines reuse the existing alias resolver and re-anchor inferred tails without erasing declared obligations. Initial original social_dynamics reproduction is fixed; root reviewed source/tests. Final combined consumer proof/newPR remains pending.
- [1771](https://github.com/sunholo-data/ailang/pull/1771): original explicitly approved M1–M2 only. Preserve source history; canonical F4 forbidden lists/reserved applied labels within v2, deterministic producer facts and strict cache validation; cache epoch v8 retains current CoreTI DAG. Root review found and verified producer alias/eta and extern fixes. Actual final integration, artifacts/goldens, CI and independent acceptance remain pending. M3/M4 and1134/1132 closure remain excluded.
- Completion-audit branch archives accepted designs AND companion plans/evidence under implemented/v0_56_0 as UNRELEASED. No release, version bump, tag or deployment is part of this sweep.

## PRs deliberately retained

Seventeen PRs have existing pending operator approval ownership across prod/test/dev: **1789,1790,1862,1867,1883,1894,1916,1926,1928,1931,1934,1939,1978,1983,1984,1985,1998**. No controller identity was set; no card was approved/rejected, no orphan was cleared, and no PR was merged as a bypass. The prod inventory has15 records, including one without an open PR. Existing ownership was checked before assigning agents.

Important retained findings:

- 1984 and1985 compete for the same duplicate-import issue; merge/execution of both would duplicate work and disagree about diagnostics.
- 1983's x/net remediation already landed independently; reconcile the existing card instead of dispatching another fix.
- 1928/1931 appear superseded but require operator reconciliation;1978 must account for the existing grounding design;1998 needs corrected coverage evidence/scope.
- 1926 owns the shared empty-PID readiness race that surfaced in unrelated CI. No competing PTY fix was authored.
- 1939/1934 and failed1759 retain cascade evidence. The ACK-versus-NACK human ruling remains open; useful failed-task source is not disposable.
- 1894 belongs to an existing dev-plane experiment; scope review is still needed.

Three other held PRs: **1759** (cascade human ruling/source consolidation), **1726** (flagship design/quorum and human decisions), **1898** (ESLint10 incompatible with current React-plugin peer range; reproduced npm ERESOLVE, no peer-check bypass).

## Remaining issue work

The starting inventory contains unresolved semantic/design decisions and existing owners; it was not a list of safely closable tickets. Priority unowned findings include false-VERIFIED SMT1960/1961 (design requires quorum), confined docs fallback1965, protected publisher-key file1900, and staged path-dependency smoke1773. Runtime-contract1954/1955/1956 has open design decisions. Shared AI cancellation1259/578/231 is one root cause, not three parallel implementations. See the historical per-issue source verification for every item; newly accepted resolutions are recorded above.

## Evidence limits and preservation

Required gates are distinguished from optional queued/running Mac matrix jobs. The final #2005 follow-up at11:31:16UTC found all27 checks complete,20green/7skipped,0pending; the existing evaluator CI script returned PASS. Full native tests ran on isolated clean review worktrees outside TMPDIR; unrelated startup/deadline failures and passing bounded retries remain preserved. Original failed hypotheses, full fork provenance, corpus inherited errors/timeouts and prior FAIL evaluations remain available. No Sonar threshold/baseline/exclusion/disposition was changed. No generated-Go requirement was dropped when explicitly in approved scope.

The shared checkout's three dirty benchmark JSON files were preserved. No destructive Git operations, live-rig upgrade, upstream Motoko write, coordinator dispatch/approval or inbox acknowledgement occurred. Ambient20 unread messages remain untouched. New development used isolated worktrees and existing agents with distinct ownership.
