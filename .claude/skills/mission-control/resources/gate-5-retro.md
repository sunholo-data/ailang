## Gate 5 — RETRO + REPORT

First action: `bash tools/launchd/mission-heartbeat.sh stamp gate-5`.
After every Gate-5 report has been sent and this iteration is fully complete, run
`bash tools/launchd/mission-heartbeat.sh stamp complete` as the final gate action.

1. Scan this iteration's friction (evaluator feedback, executor corrections, your own dead ends)
   plus unread `docs/sprint-retros/` material. Route each item to exactly ONE lane:
   - **skill fix** — edit the offending SKILL.md. Max ONE skill edit per iteration; requires ≥2
     recorded frictions pointing at the same gap; state both in the commit message.
   - **process fix** — edit the mission doc (guardrails/ordering/routing policy per its rules).
   - **backlog** — new design doc via design-doc-creator, or re-prioritize the queue.
2. Routing-policy change? Only with ≥3 evidence rows; stamp it in the mission doc.
3. Morning report, TWO channels (both required). **DIGEST FORMAT, HARD-CAPPED (Mark directive
   2026-07-31: "the github progress issues are very verbose … we could work on more conciseness").**
   The issue thread is a COMMUNICATION channel, not loop memory — the loop never re-reads its own
   reports (Gate 0 filters for Mark's comments only); the full record lives in the charter STATUS
   + mission log, which Gate 4 already wrote. Do NOT mirror the STATUS entry into the issue.
   - `ailang messages send controlplane "<digest>" --title "Mission iteration N: <headline>"
     --from "mission-${MISSION_NAME:-control}"`
     **⚠ THE BODY IS A POSITIONAL ARGUMENT, AND `ailang messages send` HAS NO `--body-file` FLAG —
     IT SILENTLY ACCEPTS THE LITERAL STRING `--body-file /path` AS THE MESSAGE BODY, EXITS 0, AND
     PUBLISHES A PUB/SUB NOTIFICATION FOR A MESSAGE WHOSE CONTENT IS GONE** (added 2026-08-22 V1
     iteration 252; instance 1 is `mission-world` iter-110, whose cross-mission message arrived at
     V1 with a body reading exactly `--body-file /tmp/w110_xmsg.txt` — recoverable only because
     the file happened to still exist on the shared rig; instance 2 is V1 iteration 252's own
     reply, which failed identically **in the same iteration that had just read World's**). The
     form above is CORRECT — the trap is that this very gate prescribes `--body-file` for the
     `gh` channel one line below, and the rotation step at 4.2 emphasises it hard ("`--body-file`,
     never an inline `--body`") for good reason. So the reader learns "always `--body-file`" from
     the loud rule and carries it to the adjacent command, where it is not a flag at all. Two
     missions made the identical substitution; that is a property of this file's layout, not of
     either controller.
     Note which half of the channel survives, because it is what conceals the loss: the **title**
     parses fine, so the message appears in `ailang messages list` looking entirely normal, and
     the sender sees `✓ Message sent` plus `✓ Pub/Sub notification published`. Gate 0's own
     mechanism-B rule already names this class — **a reporting command's exit code describes the
     request, not the delivery** — and it is worse here than for `gh issue close --comment`,
     because there is no `! already closed` warning of any kind. The recipient gets a title and a
     path that means nothing on their machine.
     **Rules. (a)** Pass the body POSITIONALLY, and pass flags with the single-dash Go form this
     CLI actually parses (`-title`, `-from`); read `ailang messages send --help` rather than
     assuming the `gh` flag vocabulary transfers. **(b)** For a long body, use a quoted command
     substitution — `ailang messages send controlplane "$(cat /tmp/body.txt)" -title "…" -from "…"`
     — which is safe for markdown: substituted output is not re-scanned, so backticks inside the
     file stay data. **(c)** ASSERT THE ARTIFACT after sending: `ailang messages read <id>` and
     confirm the body is your text, exactly as Gate 0 requires a comment count to grow. **(d)** A
     re-send of the same title is refused as a duplicate — change the title or pass `-force`, and
     say in the re-send why. **(e)** When a message ARRIVES looking like this, the sender's content
     may still be on the rig at the named path; read it before replying, and tell the sender their
     body was lost, or the same silent failure recurs in both directions forever.
     Mission-independent, and the generalisation is this file's own recurring shape: **a flag
     vocabulary is per-tool, and an emphatic rule about one tool becomes a bug in the tool beside
     it.** The tell: you are about to write `--body-file` on a command that is not `gh`.
   - `gh issue comment "$MISSION_GH_ISSUE" --repo "${MISSION_REPO:-sunholo-data/ailang}" --body "<digest>"`
     — the human-facing bookkeeping thread (Mark reads by email; number comes from the driver env /
     `~/.ailang/state/mission-${MISSION_NAME}-gh-issue`, NOT hardcoded and NOT the bare,
     fleet-shared `mission-gh-issue` — see the Repo Profile).
   **The digest — ≤26 lines / ≤2,200 chars, exactly these sections, nothing else (Mark directive
   2026-08-31: the report exists so Mark can PRIORITIZE — goal distance, the banked queue and
   complete decision asks go IN; narrative goes OUT):**
   ```
   **Iteration N — <one-line headline>**
   - **Pick**: <item> (<why in ≤1 clause, only if not the queue head>)
   - **Outcome**: LANDED/PARKED/none · [PRODUCT|HARNESS|ADMIN|REFUTATION] · evaluator <score> · <commit SHAs as links>
   - **Progress**: <distance to the charter's finish line, in the charter's own countable unit —
     e.g. "sweep 7/93 sites converted (M1 of 4 milestones)" — then what THIS iteration moved.
     A HARNESS/ADMIN iteration writes "goal unmoved" in those words.>
   - **Up next (banked)**: <top 2-3 READY queue items, each "<item> — <why it ranks>" on one line>
   - **Key find**: <≤2 sentences, ONLY if it should change Mark's priorities — else omit the row>
   - **Cost**: metered $<x> · quota buckets <list>
   - **DECISIONS FOR MARK**: "none", or one bullet per ask. A complete ask carries INLINE: the
     question in one sentence · each option with a one-line consequence · the loop's own
     recommendation · the default if unanswered (and when it triggers). Model on
     D-MOTOKO-WORKDIR-2, answered in 21 characters BECAUSE the ask was complete. An ask Mark
     must research before answering is not an ask; it is homework, and it will sit unanswered.
   Full record: <link to the charter STATUS entry / log>
   ```
   Work-class tags (tag honestly — the weekly report aggregates the mix, and "everything is
   HARNESS" is itself the signal Mark is watching for): PRODUCT = an AILANG user or eval model
   experiences the change (language, stdlib, prompts, examples, published docs, registry, eval
   benchmarks as a surface); HARNESS = loop/CI/routing/observability/git-hygiene machinery;
   ADMIN = bookkeeping only; REFUTATION = a premise died, nothing shipped.
   Mirror the class tag into the Gate-4 log headline (trailing `[CLASS]`) and repeat the same
   `**Progress**:` line inside the log entry — `tools/mission-weekly-report.py` parses both the
   way it already parses `**Next**` (the log keeps its `**Next**` field; only the digest replaces
   Next with the banked list). If the charter defines no measurable finish line, write
   "Progress: charter has no finish line" — and that sentence is a standing Gate-5 process-fix
   trigger to add one (a goal block with a countable unit), outranking other retro lanes.
   The DECISIONS row is first-class — it is the one section Mark acts on; never bury an ask in
   prose. No gate-by-gate narration, no routing-evidence dump, no war stories (those belong in
   Gate 4's charter/log record). End the body with:
     `🤖 Generated with [Claude Code](https://claude.com/claude-code)`

4. **WEEKLY THREAD ROTATION (Mark 2026-07-16 — do this BEFORE posting the report):** the
   bookkeeping thread rolls weekly so neither GitHub's UI nor Gate-0's comment fetch grows without
   bound (#329 hit 120KB/53 comments in 6 days). **Rotate when** (either): the current time is past
   the most recent **Monday 07:00 LOCAL time** (the quota-reset boundary, in the rig's own
   timezone — NOT UTC; compare with `date` output, not `date -u`) AND the current issue was created
   before that boundary; OR the current issue has >80 comments. **The timezone is load-bearing, not
   pedantry** (stamped iteration 114 after iterations 111/112/113 each flagged the omission, the
   third time verdict-relevantly): `#484` was created `2026-07-27T05:27:49Z`, so read as UTC (07:00Z)
   the boundary sits BEFORE creation and the issue spuriously rotates at one day old, while read as
   local CEST (= 05:00Z) it correctly does not. Issue `createdAt` comes back from `gh` in UTC, so
   convert one side explicitly rather than comparing the two strings as-is. To rotate:
   1. `gh issue create --repo "${MISSION_REPO:-sunholo-data/ailang}" --title "<mission> bookkeeping — week of <this
      Monday's date>" --body "<5-line state snapshot: queue head · fleet state · parked-for-human
      list · link to predecessor issue #N · directive convention: comments from
      @MarkEdmondson1234 on THIS issue steer the loop>"` — the mention auto-subscribes Mark.
   2. **Open the new thread with last week in one screen (Mark 2026-08-12) — FAIL-SOFT.**
      ```bash
      # CAPABILITY check, not an existence check — see (d). First copy that actually
      # supports --mission wins; checkouts lag independently.
      RPT=""
      for c in tools/mission-weekly-report.py \
               "$HOME/dev/sunholo-data/ailang-motoko/tools/mission-weekly-report.py" \
               "$HOME/dev/sunholo-data/ailang/tools/mission-weekly-report.py"; do
        [ -f "$c" ] && python3 "$c" --help 2>&1 | grep -q -- '--mission' && { RPT="$c"; break; }
      done
      if [ -n "$RPT" ]; then
        python3 "$RPT" --mission "$MISSION_NAME" > "/tmp/wk-$MISSION_NAME.md" 2>/dev/null \
          && gh issue comment "<new>" --repo "${MISSION_REPO:-sunholo-data/ailang}" \
               --body-file "/tmp/wk-$MISSION_NAME.md" \
          || echo "weekly report generated but did not post (non-fatal) — rotation continues"
      else
        echo "NO --mission-capable weekly report found in any checkout — skipping (non-fatal)"
      fi
      ```
      Four properties, each the fix for something that has already bitten:
      **(a) FAIL-SOFT.** Rotating the thread is essential; opening it with a summary is not. A
      formatting bug in the report must never wedge the bookkeeping of three loops, so every arm
      falls through to `echo` and rotation proceeds.
      **(b) The `$HOME` fallbacks cover Ailang World, which has no repo-local copy.** The script
      reads absolute paths for all three missions, so it need not be repo-local — and copying it
      into each repo would start a second drift surface like the driver's (233 lines adrift as of
      2026-08-12). One file, three lookup paths.
      **(d) CAPABILITY, NOT EXISTENCE — and this one was caught by testing the snippet before
      shipping it, not by reasoning.** The first draft did `[ -f "$RPT" ]` and fell back to the V1
      checkout. That file *existed* and was **two commits stale** (V1's clone only pulls when V1
      runs), so it predated `--mission` and died on `unrecognized arguments`. Combined with (a),
      World would have posted no report ever, and fail-soft would have swallowed the reason. A
      checkout being present says nothing about a checkout being current; ask the tool what it
      supports. The `else` branch is loud for the same reason.
      **(c) `--body-file`, never an inline `--body`.** A markdown body is *made* of backticks, and
      iteration 149 lost its evidence out of a `gh issue close --comment` when unescaped backticks
      triggered zsh command substitution — `gh` reported `✓ Closed` on a comment whose evidence had
      been surgically removed. Same class, same surface.
      **`--mission` scopes it deliberately:** each mission rotates independently, so an unscoped
      fleet report would land three times, two thirds of it off-topic for the thread it is in.
   3. Final comment on the OLD issue: "→ continues in #<new>" — then `gh issue close` it.
   4. Write the new number to `~/.ailang/state/mission-${MISSION_NAME}-gh-issue` and the old one
      to `~/.ailang/state/mission-${MISSION_NAME}-gh-issue-prev`. **NEVER the bare
      `mission-gh-issue`** — that path is fleet-shared and this step is the one that would clobber
      a sibling's live pointer (Repo Profile, iteration 246).
   5. Post this iteration's report to the NEW issue.
   **Rotation-week catch:** on the first iteration after a rotation (the `-prev` file is fresh),
   Gate-0's Mark-comment read must ALSO check the predecessor issue — Mark may have replied to the
   old thread over the boundary. Same allowlist + watermark.
