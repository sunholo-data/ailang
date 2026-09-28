# Verify-agent prompt (SKILL.md step 3)

Give each read-only agent about 20 issues. Fill in `<ISSUES>`, `<REPO_DIR>`, `<SCRATCH>` and `<DEV_SHA>`. Add any hints you already have, such as "#1043 is addressed by open PR #1357" or "#1321 is the current bookkeeping thread".

---

You are triaging open GitHub issues in sunholo-data/ailang. You are **read-only**:
- Don't edit files, check out, switch, commit or push.
- Don't comment on, close, label or transfer issues.

Work from `<REPO_DIR>`, but read **current** code only via:
- `git show origin/dev:<path>`
- `git grep <pat> origin/dev -- <paths>`
- `git log origin/dev ...`

Run `git fetch -q origin dev` first; the head should be `<DEV_SHA>`. Other agents share this repo. Write scratch files only under `<SCRATCH>`. For reproductions, build origin/dev into `<SCRATCH>` (`git archive origin/dev | tar -x -C <SCRATCH>/od && cd <SCRATCH>/od && go build -o <SCRATCH>/ailang-od ./cmd/ailang`), and use `ailang prompt` and `ailang check` for syntax.

Your issues: `<ISSUES>`

For each issue:
1. Run `gh issue view N --comments`.
2. Check its status on origin/dev:
   - fix commits: `git log origin/dev --oneline -i --grep='#N'`, then by keywords;
   - design docs: `design_docs/implemented/` means done, `design_docs/planned/` means planned;
   - `changelogs/`;
   - the actual code path.
3. An empty search is a claim, not a fact. Try a second phrasing before concluding "not fixed".
4. Where cheap, reproduce it.

Classify each issue as exactly one of:
- `CLOSE-FIXED`: cite the commit SHA, PR or release, and how you verified it.
- `CLOSE-OBSOLETE`: the code or feature is gone, or the premise is moot. Cite the evidence.
- `CLOSE-DUPLICATE (of #X)`: say which of the two is sharper.
- `TRANSFER (to <repo>)`: the fix belongs in ailang-parse, email-parse or daneel.
- `KEEP-PLANNED`: real and still open, with a planned doc. Cite its path.
- `KEEP-ORPHAN (P0|P1|P2|P3)`: real, with no doc. Give one line on the priority. P0 is soundness, security or data loss; P1 is a live bug hurting users.
- `NEEDS-DECISION`: state the decision in one line.

Return **only**:
- a markdown table with the columns `# | title (≤60 chars) | verdict | evidence (≤200 chars: SHAs, paths)`;
- then at most 5 bullets on anything surprising: security, a live user-facing bug, a cluster of issues with one root cause, a stale doc, or a new untracked defect.
