# Fleet iteration 136 — independent judge, round 2

Same judge (sonnet, resumed via SendMessage, worktree `.fleet-i136-judge` moved to `66e0aae56`; 76,218 tokens cumulative, 7 tool uses, 47 s).

**Verdict: PASS 90/100. 0 BLOCKING.** Tests 20, lint 10, acceptance 28/30, quality 13/15, docs 14/15, fidelity 5/10.

Findings 1 (bz2/xz silent clean) and 2 (value in a path name) FIXED and re-verified:
| Input | Round 1 | Round 2 |
|---|---|---|
| `a.tar.bz2`, `a.tar.xz`, `e.bz2`, `e.xz` | clean rc 0 | rc 2 instrument-failure |
| xz content without suffix; `.tar` holding a `.bz2` member | not tested | rc 2 |
| file `name-<value>`, dir child `dn/f-<value>`, dir `d-<value>` | clean rc 0 | rc 1 LEAK, path redacted |
| plain leak / tgz leak / clean file / dir with symlink | 1 / 1 / 0 / 2 | unchanged |

Judge mutants: disable compressed-signature check → killed by `compressed-unsupported`; remove name-value record → killed by `name-value`; unmodified control passes.

NON-BLOCKING: (1) base64, split and zlib values still scan clean by design, and the rule now says so; (2) a plain text file beginning `BZh` is rejected (fails safe); (3) the SKILL.md re-wrap noise remains.

EVALUATION_VERDICT: PASS score=90
