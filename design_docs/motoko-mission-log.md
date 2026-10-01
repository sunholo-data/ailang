# Motoko Mission — iteration log (append-only)

One entry per mission-control iteration, newest LAST (append). Fixed template — keep every
section, write "none" rather than omitting. Same template as
[v1-mission-log.md](v1-mission-log.md); do not diverge it, so cross-mission comparisons parse.

```markdown
## N — YYYY-MM-DD — <headline>
**Picked**: <backlog item + why it was top>
**Reality check**: <what git/code verification of the doc's status found>
**Shipped**: <commits/branches/PRs, evaluator result + score, or "parked: reason">
**Routing evidence**: model=<m> task-class=<design|plan|execute|evaluate|mechanical>
  round1-score=<n> rounds=<n> corrections=<n>
  provider=<p> agent=<a> cost=<$<n>|quota-bucket:weekly-fable|quota-bucket:weekly-opus|unknown>
**Ruled out**: <hypotheses/approaches refuted this iteration — the anti-re-chase ledger>
**Retro lane**: <skill-fix: file+change | process-fix: change | backlog: new doc | none>
**Next**: <what iteration N+1 should pick up>
```

**Mission-specific note on the "Ruled out" ledger.** This mission's history is dense with
hypotheses that felt right and were wrong — "switch to qwen3.6", "it's a model wall", "the docx
loop is one bug". Every motoko conclusion so far has bottomed out in a harness bug. Write the
refutation down with its evidence; the ledger is the point.

**An idle iteration is a valid entry.** If every queue row is parked or waiting (a release, an
upstream review, rig time), exhausting the unblocked queue is a correct outcome. Record it as a real
entry with `**Shipped**: parked: <what each row waits on>` rather than pulling blocked work forward
to look productive. (The fork-era Phase-0 gate this note used to cite was discharged at the
2026-09-30 reset — `D-MOTOKO-RESET-3`.)

---

> **RESET 2026-09-30 — this live log starts empty.** The charter was reset for motoko main
> (`D-MOTOKO-RESET-1`) and every fork-era entry, **0–39**, now lives in full in
> `motoko-mission-log-archive.md`; the one-line index of ALL of them — the thing to grep before
> picking work, so the loop never repeats itself — is `motoko-mission-index.md`. The next entry is
> **40**, appended below this note. Row numbers in the archived entries are the fork-era queue's;
> the live queue starts at row 20.
