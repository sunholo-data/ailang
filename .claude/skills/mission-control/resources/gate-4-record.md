## Gate 4 — RECORD (append-only; the log is the mission's memory)

**ROTATE THE LOG WHEN IT PASSES ~40 ENTRIES.** After appending this iteration's entry:

```bash
ailang mission rotate-log ${MISSION_NAME} --keep 20
```

It keeps the newest 20 full entries live, appends the rest to
`<name>-mission-log-archive.md` with their FULL bodies, and REGENERATES
`<name>-mission-index.md` — one line per iteration across live + archive, which is what
Gate 2 greps before picking. Nothing is deleted; the command is verified lossless by test.

Regenerated, never appended: an append-only index drifts the moment an entry is edited,
and an index that answers "already tried?" confidently and wrongly is worse than none.

First action: `bash tools/launchd/mission-heartbeat.sh stamp gate-4`.

**FIRST: overwrite `design_docs/${MISSION_NAME}-mission-dashboard.md`** (Mark 2026-08-04: the
30-second control context for fresh sessions — his long-lived thread was burning 14%/week of quota
as cache-rebuild). Keep it ≤40 lines, OVERWRITE never append: latest release · in-flight/next
picks · loop cadence+routing · parked-on-Mark · quota posture. It is a snapshot, not a record —
history stays in the charter/log.

**⚠ THE PATH IS NAMESPACED, AND THE UNNAMESPACED `design_docs/mission-dashboard.md` THIS GATE USED
TO PRESCRIBE IS ONE FILE THAT EVERY MISSION OVERWRITES — SO THE INSTRUCTION "OVERWRITE, NEVER
APPEND" MADE EACH LOOP DESTROY ITS SIBLING'S SNAPSHOT ON EVERY ITERATION** (fixed 2026-08-17 V1
iteration 216; four recorded frictions, all first-party). This is the same class the roles table
already fixed for the designer-rotation state key — a shared skill naming a bare literal that reads
as per-mission and is in fact fleet-global — and it is worse here, because the gate's own emphasis
is on clobbering. The failure is silent and self-concealing in both directions: a controller that
obeys the rule destroys a sibling's dashboard and reports success, while a controller that notices
the collision can only skip its own record, so the dashboard is *never* right for both missions.
Measured on V1: `design_docs/mission-dashboard.md` held **Motoko's** snapshot (`# Mission
Dashboard — Motoko`, iteration 7) while a **separate, hand-created** `motoko-mission-dashboard.md`
also existed — i.e. a careful sibling controller had already worked around this by hand, exactly as
the sibling controllers did for the rotation key, and the shared skill never read the namespaced
path. V1 iterations 212 ("the cross-mission single-dashboard collision is a new process-gap
instance"), 213, 214 and 215 each recorded the friction and each responded by **omitting V1's
dashboard refresh** rather than clobbering — so V1 had no current dashboard for four consecutive
iterations, which is precisely the 30-second context this gate exists to guarantee.
**Rule.** Write the mission-namespaced path. Never write the bare `mission-dashboard.md`, and if
you find a sibling's content there, leave it alone and say so in the report rather than "fixing"
it. **And audit the whole literal-path list rather than one key at a time** — the roles table's
migration note says the same thing about `~/.ailang/state/`, and this gate is the instance that
proves the audit was never done: one namespacing fix landed, the neighbouring literal did not.
Whenever this skill names a path a mission writes to, ask what the sibling writes there first. The
tell: you are about to overwrite a file whose name contains no mission identifier.

Append an entry to `design_docs/v1-mission-log.md` using its fixed template — every section,
"none" over omission. The **Routing evidence** row and **Ruled out** ledger are the two highest-
value fields: evidence drives routing-policy changes; ruled-out stops re-chasing. Update the
mission doc's queue tags ([LANDED], [PARKED], etc.) and STATUS stamp.

**⚠ AND THE COMMIT THAT LANDS THIS RECORD CAN *CLOSE AN ISSUE IT ONLY TALKED ABOUT* — GITHUB'S
AUTO-CLOSE PARSER DOES NOT READ ENGLISH, AND GATE 4 MANDATES EXACTLY THE DISCURSIVE PROSE THAT
FEEDS IT** (added 2026-08-21 V1 iteration 240; two measured instances plus a recorded near-miss).
Every rule in this gate polices the record's *content*. None polices what merging it *does to
other objects*. `fix`/`fixes`/`fixed`/`close`/`closes`/`closed`/`resolve`/`resolves`/`resolved`
followed by `#N` closes `#N` at merge, from a **commit message**, a **PR title**, or a **PR body**
— with no regard for the surrounding sentence. So *"the arena **fixes #676** completely"*, written
inside a paragraph *arguing about a candidate design option*, closes the issue exactly as hard as a
deliberate `Fixes #676` would.
**The loop is uniquely exposed, and by its own instructions.** This gate requires long records that
discuss issues by number, and a mission record's whole job is to reason about what *would* fix
what. The practice manufactures the keyword. Note the failure is silent and inverted: the more
carefully you reason in prose about a fix you have **not** shipped, the likelier you are to close
the issue tracking it.
Measured on V1. **(a)** `#676`, a live user-reported OOM this mission had itself triaged **REAL at
HEAD**, was closed `COMPLETED` by `dedf3b91f` — a **docs-only** record, 7 files, **zero code** —
1h46m after our own comment said it was real and unfixed. The repo is public; an external reporter
saw their live bug marked done. **(b)** `#612` was closed by `7c7e5e58a`, which shipped **one
636-line sprint plan**; its deliverable never landed (`go/packages` importers **0**, `x/tools` in
`go.mod` **0**, controls firing at **2** and **99**). **(c)** The near-miss that makes this a
recurrence rather than an accident: the charter records that a planner, *in that same sprint*,
*"stripped an auto-linked `Fixes #612` that would have wrongly auto-closed the out-of-scope
follow-up."* **The hazard was known and the guard was applied to the DOCUMENT, never to the COMMIT
MESSAGE** — *guard the helper, miss the call site*, aimed at this loop's own record-keeping.
**Rule. (a)** Before committing, scan the commit message AND the PR title/body for
`(clos(e|es|ed)|fix(e|es|ed)?|resolv(e|es|ed))\s*:?\s*#[0-9]+`, and pair the scan with a
known-bad control string (`printf 'this fixes #1\n' | grep -E …`) so a clean result proves the
matcher fires rather than that you typed the pattern wrong — rule 3a, aimed at your own commit.
**(b)** Reserve a closing keyword for the commit that **actually ships the thing**. Everywhere else
— records, plans, triage, reasoning about options — reference the issue with a non-triggering form:
*"reported at #676"*, *"the defect in #676"*, *"filed as #612"*. **(c)** After any merge that
mentions an issue, **assert the issue is still in the state you expect**; the exit code of `gh pr
merge` says nothing about what the merge closed, and a PR body's `Fixes #N` closes **before** any
Gate-0 close step of yours runs (the mechanism-B hazard that gate already names, arriving from the
other side). **(d)** A file's *contents* are safe — GitHub parses commit messages, PR titles and PR
bodies only — so a charter, log or changelog may quote the offending phrase when describing this
defect. Verified first-party: an **issue comment** containing `fixes #676` twice left the issue
`OPEN`. Do not over-apply the guard and mangle your own record.
Mission-independent, and the generalisation outranks the two instances: **a record is not inert —
writing about a system can mutate it.** Wherever this loop's prose is parsed by a machine that does
not share its intent (issue keywords, `@mentions`, CI directives like `[skip ci]`, changelog
scrapers), the record is an *actuator*, and Gate 4 has been treating it as a notebook. The tell:
your commit message or PR body names an issue number, and you have not run the scan.

**⚠ AND `git add` SILENTLY SKIPS AN IGNORED PATH, SO ANY STEP WHOSE DELIVERABLE IS *"BANK IT"* /
*"ARCHIVE IT"* / *"RECORD IT"* CAN REPORT SUCCESS HAVING WRITTEN NOTHING TO THE REPO — ASSERT THE
DESTINATION IS TRACKED BEFORE YOU CLAIM THE ARTIFACT EXISTS** (added 2026-08-22 V1 iteration 253;
instance 1 is iteration 195, instance 2 is this iteration). Every rule in this gate polices the
record's *content* and *base*. None asks whether the path you are writing to is one git will
accept. That question has no tell: `git add <ignored>` prints nothing and exits **0**,
`git status` shows nothing, and the commit succeeds — so the artifact is absent in exactly the
voice of an artifact that landed. It is the vacuous-pass class aimed at the *archiving* step.
Note who is most exposed: an **acceptance criterion** that says "archive the full output" is
written by a designer or planner reasoning about deliverables, never about `.gitignore`, so the
criterion passes as long as a file appears **on disk**.
Measured. Instance 1: `.gitignore:77` ignores `.ailang/` with no negation, so a NEW sprint JSON
was skipped by `git add -A` silently — 0 staged, empty output — and one milestone's state
artifact was orphaned. Instance 2: `eval_results/` is ignored (`.gitignore:91`), and M4b's own
acceptance criterion is *"archive its full output"*; the cohort artifacts were copied there,
looked correct in `ls`, and would have been committed as **nothing**. Caught by running
`git check-ignore` *before* staging rather than by noticing an empty diff afterwards.
**Rules. (a)** Before recording that an artifact is banked, run
`git check-ignore -v <path>` and pair it with a control on a path you KNOW is ignored, so a
clean answer proves the instrument fires (rule 3a, aimed at your own write). **(b)** When the
destination is ignored, do NOT reach for `git add -f` by reflex — ask first whether the repo is
right that this class does not belong in git, and route the *decision-bearing* subset (a
manifest, a KPI record, a summary) to a tracked path instead, leaving the bulk where the
ignore rule intends. **(c)** Where the artifact's real home is a database or an external store,
say so explicitly in the record, so a later reader does not search the tree for it. **(d)** An
AC of the form "archive/bank/record X" is **vacuous** unless it names a path a reviewer can
open; treat a criterion that only requires a file to exist on disk the same way rule 3b(vi)
treats an AC whose gate cannot see the code. Mission-independent — every mission on this rig
ignores build and result directories. The tell: your deliverable is *a file that must persist*,
and the only thing you have checked is that you wrote it.

**WRITE THE RECORD WHERE YOU READ THE STATE — NEVER INTO A WORKING-TREE COPY YOU HAVE NOT
RE-CONFIRMED AGAINST ORIGIN** (added 2026-08-01 iteration 129; instance 2 of the diverged-checkout
class after iter-128's stale *skill* — this one is the stale *charter*, and its failure mode is a
silent mass deletion). Gate 1 already tells you to READ mission state from origin when local `dev`
is behind. Nothing said the same about WRITING it, and the two halves are not symmetric: Gate 1's
remedy leaves you reading origin's charter while your editor still points at the working tree's.
Measured at iteration 129: local `dev` was 1 ahead / 8 behind, and the working-tree charter carried
STATUS stamps **123/125/126** while origin carried **126/127/128** — so an in-place "add your stamp,
rotate the 4th out", the literal instruction above, would have committed a charter with **iterations
127 and 128 deleted**, and the line-count assertion below would have *passed*, because that
arithmetic is self-consistent against the wrong base. Same shape as the STATUS-rotation bug (a
destructive edit reports success exactly like a correct one), but the corruption arrives from the
BASE rather than from the edit, so no amount of care inside the edit can catch it. Before the first
Gate-4 write, re-confirm the base:

```bash
git fetch origin
git rev-parse dev origin/dev                     # differ at all? the working tree is NOT the base
git diff --stat origin/dev -- "$MISSION_DOC" design_docs/*-mission-log.md
```

If charter/log differ from `origin/dev`, do **not** edit them in the shared checkout: write the
record in a worktree branched from `origin/dev` (`git worktree add -b … <path> origin/dev`) and land
it by PR. **The cheap tell, and the one to actually use:** grep the file you are about to edit for
the PREVIOUS iteration's stamp — if the last iteration's own record is missing, you are holding a
stale copy, not a charter awaiting your entry. One command, and it is the difference between
appending history and erasing it.

**SPELL THE TELL IN THE CHARTER'S OWN CASE, AND PAIR IT WITH A CONTROL** (added 2026-08-03
iteration 134). Stamps are written `ITERATION 133` — **UPPERCASE** — while the sentence above says
"the previous iteration's stamp", so the natural transcription is `grep -c "Iteration 133"`, and
that returns **0** on a perfectly healthy charter. Iteration 134 ran exactly that and read `0` for
a charter that was byte-identical to origin. This is rule 3a's trap wearing THIS gate's clothes,
and it is the worst place for it: a broken tell and a genuinely stale charter produce the
identical output, so the failure routes a healthy iteration down the stale-copy path — or, in the
other direction, teaches you to distrust a tell you will need for real. Run it
**case-INSENSITIVELY** — `grep -ci "ITERATION <N-1>"` — **alongside a known-present control in the
same breath** (`ITERATION <N-2>`, must be ≥1). A `0` on the control means your instrument is
broken, not that the charter is stale — that is the failure mode here, and the known-present
control is the one that catches it.

**Why `-ci` and not the charter's own casing: STAMP CASING IS MISSION-SPECIFIC, AND THIS SKILL IS
SHARED, SO HARDCODING ONE MISSION'S CASING BREAKS THE TELL FOR EVERY OTHER MISSION** (added
2026-08-07 iteration 157; proposed by `mission-world` iter-60, which shares this skill but cannot
edit it, and corroborated first-party in BOTH repos before adoption — sibling-claim ghost
discipline). Iteration 134 correctly diagnosed the casing trap but fixed it by pinning the literal
to **V1's** format, `## STATUS 2026-08-07 — ITERATION 156:`. World stamps
`## STATUS 2026-08-07 (iteration 60)` — lower-case and parenthesised. Measured against World's
**healthy** charter, the prescribed form returns `grep -c "ITERATION 60"` → **0**, and — the part
that matters — its known-present controls return **0 too** (`ITERATION 59` → 0, `ITERATION 58` → 0),
while `grep -ci` returns **1 / 4 / 4**. So the remedy iteration 134 wrote to stop a healthy charter
reading as stale did exactly that on the sibling mission. It at least fails LOUDLY rather than
silently — a zeroed control is the documented "instrument broken" signal, which is how World caught
it and ran `-ci` as a workaround — but a tell that cannot run unmodified outside the mission that
authored it is not a shared tell. **Read the result as PRESENCE (≥1), never as an exact count:**
`-ci` also matches ordinary prose ("…added 2026-08-06 V1 iteration 154…"), measured in V1 as
`ITERATION 154` → **2** case-sensitive vs **3** case-insensitive. The tell asks "is the previous
iteration's record here at all", so presence is the whole question and the extra prose hits are
harmless; the structural count you actually assert against is the rotation invariant below. General
form, and the reason this outranks its two instances: **anything this skill tells you to grep for is
a claim about ONE mission's file format** — when a shared gate hardcodes a literal, ask what the
sibling writes there before trusting a zero.

**A CONTROL YOU RECORD IS A CONTROL YOU SPEND — in a file the loop WRITES ABOUT ITSELF, the
absent token does not stay absent** (generalised 2026-08-19 V1 iteration 232 from a Gate-4-only
note; proposed by `mission-world` iter-97 at the two-gates bar, and corroborated first-party in
V1's own artifacts before adoption — sibling-claim ghost discipline). Any known-ABSENT literal used
as a negative control must never be written into a corpus the loop later greps: the charter, the
log, the STATUS archive, the dashboard, or an issue body a sweep reads. The loop writes *about* its
own measurements, so **naming the control in a record is what makes it stop being absent** — and the
next controller to reuse it reads a `1` and concludes its *matcher* is over-matching rather than
that the corpus was poisoned by a record.
**This is not a Gate-4 quirk; it fires in every gate that both requires a control and publishes the
measurement.** Instance 1, Gate 4: iteration 134 shipped `ITERATION 999` as its known-absent control
and measured it coming back **1** within the same iteration, because the STATUS stamp it had just
written documents the control. Instance 2, Gate 0's weekly external-issue sweep, whose rule (b)
*mandates* printing per-issue counts so a zero is auditable — `mission-world` iter-96 recorded
`#9999 → 0 negative` in its stamp and iter-97 measured `#9999` → **1**. Nothing in Gate 0 said the
control's identifier must not be one of the things you print, and the warning lived two gates away
attached to a different artifact and a different literal shape: **guard the helper, miss the call
site**, this loop's own named recurring shape, aimed at its rulebook.
V1's first-party corroboration is stronger than the proposal's, in two ways. Its Gate-0 sweep
control is spent in **two** files at once — `grep -cE '#99999\b'` returns **1** in
`design_docs/v1-mission.md` (line 2102, iteration 216's own sweep verdict) and **1** in
`v1-mission-log.md`, controls firing at `#613` = **7** and **50**. And `ITERATION 999` reads **0**
in the charter but **1** in the log (control `ITERATION 231` = 2 / 1), which is the sharpening
neither instance had: **rotation does not un-spend a control.** The STATUS block's own three-entry
rotation eventually carries the stamp out of the charter, so a controller re-measuring only the
charter sees the literal go absent again — while the log and the archive keep it forever. A control
that is spent is spent across *every* corpus the loop greps, not the one it was written in.
Rules: **(a)** choose a FRESH absent literal each time and treat any reuse across iterations as
suspect — including a literal that currently reads zero, since a rotated-out stamp is not a
retracted one; **(b)** where a gate requires the measurement to be published (Gate 0's per-issue
table, Gate 4's rotation assertion), publish the control's RESULT and not its IDENTIFIER —
"negative control fired" rather than "`#9999` → 0"; **(c)** prefer a structural check that cannot be
poisoned at all — Gate 4's `grep -c "^## STATUS 2026"` must equal **3** is the model: line-anchored,
format-bound, immune to prose. Mission-independent: every mission on this rig writes records it
later greps. The tell: you are about to write a known-absent literal into a file this loop reads.

**THE STATUS ROTATION IS THE MOST DANGEROUS EDIT THIS LOOP MAKES — SCRIPT IT WITH A LINE-COUNT
ASSERTION, NEVER A BARE `## `-HEADER SCAN** (added 2026-08-01 iteration 127; third failure of this
same step — iter-83 hand-corrected an already-drifted N>4, iter-123 found the block drifted to 4
and applied the self-heal, and iter-127 *deleted the entire queue*). The charter is ~1,600 lines of
which the STATUS block is ~4, so a rotation bug is a **mass-deletion bug**, and it lands in the one
file every future iteration reads as ground truth. Iteration 127's script computed each stamp's
extent by scanning forward to the next `## STATUS` header; for the NEWEST-3 boundary that works,
but for the **last** stamp there is no following header, so the scan ran to EOF and moved the whole
1,571-line queue into the archive. It exited 0 and printed a plausible `archived: [...]` line.
Rules: **(a)** a STATUS stamp is a **single line** followed by one blank — do not model it as a
block delimited by the next header; **(b)** assert the arithmetic *before* writing, and fail loudly
if it does not hold — `after == before + 2 - 2*len(moved)` is the whole invariant, and it is what
caught this on the re-run; **(c)** after any charter edit, grep for a **queue** row you know exists
(not a STATUS row) — the damage here was invisible in the STATUS block itself and surfaced only
because a later `grep` for a queue item came back empty and was treated as a claim (rule 3a) rather
than as "the row must have moved"; **(d)** never `git add` the charter in the same breath as
writing it — `git diff --stat` first, and a charter diff whose net line delta is not roughly
`+stamp -archived` is a bug, not a formatting artifact. The generalisable point, which is rule 3a
pointed at your own edits: **you are an instrument too, and a destructive edit reports success
exactly like a correct one.**

**AND EVERY ASSERTION ABOVE IS CHARTER-SIDE, SO ALL OF THEM PASS IDENTICALLY WHETHER THE ROTATED
STAMP WAS *MOVED* TO THE ARCHIVE OR SIMPLY *DELETED* — ASSERT THE ARCHIVE END TOO** (added
2026-08-13 V1 iteration 190; two first-party instances, iterations **171** and **186**). The
rotation is a two-file operation described by one file's arithmetic. `after == before + 2 −
2×len(moved)` is a statement about the CHARTER's line count; it is satisfied exactly as well by a
correct move as by a deletion, because both remove the same two lines. Rule (c)'s queue-row grep
looks at the charter, and rule (d)'s `git diff --stat` shows `-archived` in the charter without
ever asking where those lines went — indeed a diff **stat** over both files still nets out
plausibly if the archive gained a *different* number of lines. So the one thing the rotation exists
to guarantee — that the stamp survives somewhere — is the one thing nothing checks. This is the
vacuous-pass shape the mission keeps closing, aimed at Gate 4's own edit: the assertion passes for
the wrong reason.
Measured on V1: of the six iteration numbers absent from charter+archive in the range 150–190,
**159/160/165 were never written at all** (reaped slots — Standing rule 7, so a gap is NOT by
itself a rotation defect, and attributing it as one would be rule 3d), while **171** and **186**
were each *added to the charter and later removed with no archive commit ever touching them*
(`git log -S "ITERATION <n>" -- <charter>` shows 2 commits, `-- <archive>` shows **0**; control
`185` shows 2 and **1**). Iteration 186's stamp was recovered from `8ecebc0e1` at iteration 190;
171's is still recoverable and was not.
**Rule:** after any rotation, grep the ARCHIVE for the stamp you just moved and require **≥1**, in
the same breath as the charter arithmetic — `grep -c "ITERATION <moved>" <archive>`. Pair it with
a known-present control (`ITERATION <moved-1>`) so a zero means "the move failed" rather than "my
pattern is wrong". **And do NOT enumerate stamps with a header-shaped pattern to audit this**:
V1's own audit of exactly this used `^## STATUS … — ITERATION N` and reported `163` and `184`
missing when a raw `grep -c "ITERATION N"` finds both — the header format drifted, so the strict
pattern manufactured two false gaps beside the two real ones. Count the bare token, not the
header. Mission-independent: the casing/format of the stamp is per-mission (see the `-ci` rule
above), but "the destination gained what the source lost" is a property of every rotation.
