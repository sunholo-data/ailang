# Gate 2 — the verification protocol

The 18 rules the mission loop verifies against, in full. Each one is an epistemics
failure **this loop actually committed**, with the measurement that caught it, the
commands that discriminate, and the tell that says you are about to repeat it.

Read this file at Gate 2 before verifying anything. It lives outside `SKILL.md`
because it is 1,378 lines the other five gates do not need in context — not because
it is optional. The index in `SKILL.md` is a recall aid for rules you have already
read; the discriminating command is always here, never there.

Back to the procedure: [`../SKILL.md`](../SKILL.md).

---

**Verification protocol** (added iteration 1 after three same-class frictions). Steps 1–3 are the
`go-compiler` verify profile (V1); under `ailang-code` the shipped binary IS the gate — skip the
compile/staleness steps and run `ailang check`/`ailang test`/`ailang ai-check` instead (see
the Repo Profile above):
1. **Rebuild before any live check** (`go-compiler` only): `make quick-install && make build` — BOTH
   binaries. `~/go/bin/ailang` (PATH) and `bin/ailang` (preferred by test helpers when present) go
   stale independently; a stale one silently falsifies results (1a: stale installed binary showed
   pre-fix behavior; 1b-eval: Jun-26 `bin/ailang` v0.26.0 broke `make test` with a phantom
   `_io_flush` error). Confirm `--version` matches `git describe` before trusting output.
   **⚠ AND THE STALE BINARY REACHES YOU THROUGH *TESTS*, NOT ONLY THROUGH YOUR OWN COMMANDS —
   WHERE THERE IS NO `--version` TO CHECK AND THE STALENESS ARRIVES WEARING A PLAUSIBLE CODE
   DEFECT** (added 2026-08-20 V1 iteration 237; instance 1 is iteration 235's quorum, run on a
   binary **35** commits adrift, instance 2 is this iteration's, at **37**). Everything in step 1
   is written for a binary *you* invoke: rebuild it, then confirm `--version` against
   `git describe`. That remedy cannot reach a test which shells out to `ailang` **from PATH**
   inside its own body — you never type the command, so there is nothing to version-check, and
   the failure surfaces as an ordinary red with a specific, technical, entirely convincing cause.
   Measured here: `tests/golden/codegen` calls `exec.Command("ailang", "compile", …)`, so
   `TestGoldenCompile/string_charat` failed on `undefined: CharCode` — a real symbol, in a repo
   that has a test *about* that symbol, which is why the executor read it as a codegen gap and
   said so in its report. Two arms on the **identical pristine tree**, exit codes captured to
   file and printed side by side: stale PATH `rc=1`, fresh binary `rc=0`. Nothing about the diff
   was involved.
   **The trap is structural, not an accident, which is why it recurs.** Step 1's own remedy is
   `make quick-install`, and this loop must NOT run it — `~/go/bin/ailang` is shared with every
   concurrent agent on the rig, so installing mid-iteration is the shared-checkout hazard rule 4
   already forbids, aimed at a binary instead of a tree. So the correct behaviour (leave
   `~/go/bin` alone) *guarantees* the PATH copy drifts, without bound, forever. Iteration 235 got
   half of it right — it built into the worktree and used `bin/ailang` — and that half does
   nothing for a test's inner shell-out.
   **Rule. (a)** Build to a scratch directory and **prepend it to `PATH`** for any suite you are
   about to trust — `go build -o /tmp/<dir>/ailang ./cmd/ailang` then
   `PATH="/tmp/<dir>:$PATH" make test` — rather than installing. It leaves `~/go/bin` untouched,
   so concurrent agents are undisturbed, and it reaches shell-outs a `bin/ailang` build cannot.
   **⚠ CARRY THE LDFLAGS: A BARE `go build` IN A LINKED WORKTREE PRODUCES A BINARY THAT CANNOT SAY
   WHAT IT IS, AND NOTHING WARNS YOU** (added 2026-08-23 V1 iteration 256; instance 1 is iteration
   253, whose frozen cohort manifest recorded `ailang_version:"dev"`/`git_commit:"dev"` and said so
   in its own STATUS stamp because *"the artifact could not record it"*; instance 2 is this
   iteration, which measured the cause). This rule and Gate 3's worktree rule are each correct and
   their **intersection** is the defect: Gate 3 says never build in the shared main tree, this
   clause says never `make quick-install`, and Go's VCS stamping **does not work in a linked git
   worktree** — so every binary this loop builds is stamped `"dev"`, by construction, forever.
   Measured on V1, `go version -m`, dotted `vcs\.` pattern, control = total `build` settings:
   pin worktree (detached) **0** vcs lines / 10 settings; the main checkout, a real `.git`
   **directory**, **4** / 14 with a correct `-dirty` commit; a linked worktree **on a branch**
   **0** / 10 — so it is the worktree, not the detached HEAD. And the obvious fix does not work:
   `-buildvcs=true` in a worktree exits **rc=0**, produces the binary, emits **0** vcs lines and
   **does not error**, so there is no failure to notice. Note `Version` has no runtime fallback at
   all — only ldflags ever sets it — while `Commit` has a `debug.ReadBuildInfo()` fallback that the
   worktree is precisely what disables.
   **The remedy is one flag block and it is PROVEN, not argued** — executed in a linked worktree at
   iteration 256: `VERSION=$(git describe --tags --always --dirty); COMMIT=$(git rev-parse HEAD);
   go build -ldflags "-X <mod>/internal/version.Version=$VERSION -X <mod>/internal/version.Commit=$COMMIT"
   -o /tmp/<dir>/ailang ./cmd/ailang` → rc=0, and `--version` prints both fields correctly, because
   `git` is being asked rather than Go's build system. It does **not** restore `vcs.*` (still 0
   lines, control 11 vs 10), which is fine — the consumers read the package vars — but do not expect
   `go version -m` to show provenance on a remediated binary and conclude the remediation failed.
   **Why it matters beyond tidiness:** on V1 three consumers silently accept the `"dev"` — the frozen
   release-evidence manifest whose stated purpose is independent recomputation, the module cache's
   *compiler-identity* component (so a compiler bugfix does not invalidate cache), and the
   `--bank-by-version` output bucket (so results from different builds pool). Mission-independent:
   any mission whose driver or sprint work runs from a worktree builds unidentifiable binaries, and
   under `ailang-code` the same axis is a lockfile-pinned artifact. The tell: you are about to quote
   a binary's provenance, or bank an artifact that records one, and the build command you ran was a
   bare `go build`.
   **(b)** Before attributing ANY test red to the diff, run the same command in both arms —
   stale PATH and fresh — and require the exit codes to DIFFER (rule 3d, aimed at a red you did
   not predict). Identical codes mean it is the code; differing codes mean it was the
   instrument. **(c)** Read a red's *cause* with the same suspicion as its exit code: a failure
   naming a real symbol in a real file is exactly what a version skew produces, so "the message
   is specific" is not evidence that the message is about your change. **(d)** State the binary's
   provenance wherever a suite's green is quoted — "`make test` rc=0 under a freshly built
   binary" — because "the tests pass" from a stale PATH is a claim about a build nobody can
   identify. Mission-independent: under `ailang-code` the same axis is a lockfile-pinned release
   artifact that the repo's own tooling invokes by name. The tell: a test failed, you did not
   write the command it ran, and you are about to explain the failure with something in the diff.
2. **A parked test is a claim, not evidence**: `t.Skip`-ed / disabled tests say "nobody
   re-checked", not "still broken". Un-skip and RUN before treating the bug as open — the
   M-TYPEENV-SUB "open P0" was already fixed; only un-skipping revealed it.
3. **Exit codes through pipes lie**: `cmd | tail; echo $?` reports tail's status. The portable
   remedy is to capture and read back — `cmd > /tmp/out 2>&1; echo "rc=$?"` — or invoke directly.
   **Do NOT use `${PIPESTATUS[0]}`: it is bash-only and SILENTLY EMPTY in zsh**, which is the shell
   both mission rigs actually run (verified 2026-07-29 iteration 120 inside the loop's own tool
   shell: `false | true; echo "[${PIPESTATUS[0]}]"` prints `[]` here and `[1]` under bash; zsh
   spells it `${pipestatus[1]}`, lower-case and 1-INDEXED). So the remedy this step prescribes for
   "exit codes lie" has itself been printing `rc=` — an empty reading that looks like a formatting
   quirk rather than a failed check, **voiding the very gate it was added to protect**. Reported by
   mission-world (iter-37, two instances) and reproduced first-party before adoption.
   **AND THE PIPE TRAP IS WORST INSIDE A TWO-ARM CONTROL, BECAUSE IT DOES NOT LOSE ONE READING — IT
   MAKES BOTH ARMS AGREE, AND AGREEMENT IS EXACTLY WHAT A CONTROL IS READ FOR** (added 2026-08-20 V1
   iteration 236; instance 1 is iteration 233's Gate-3b poll, instance 2 is this iteration's Gate-2
   measurement — two gates, two mechanisms, one shape, which is this loop's own
   *guard the helper, miss the call site* pattern aimed at its rulebook). Everything above is written
   about ONE command whose status you lose. The dangerous case is the **experiment**: rule 3d tells
   you to run the mechanism removed and require the outcomes to DIFFER, and rule 3f tells you to
   measure a reviewer's premise rather than forward it — so the loop's best instincts all point at
   two-arm comparisons, and a broken reader corrupts both arms identically. The result is not a
   missing number, it is a **false symmetry**: the discriminator collapses and the arms look like
   evidence that the variable does not matter. Rule 3e(iii) already names that inference ("identical
   results across arms is equally consistent with … 'both arms are already broken'") and tells you to
   ask what the arms SHARE — the half it does not say is that **the thing they share is often the
   READER, not the tree**, so no amount of care about the base will catch it.
   Measured here, in the middle of the very measurement meant to settle a quorum objection:
   `go build ./internal/spikeprobe_consumer/ 2>&1 | head -5; echo "rc=$?"` and the same shape for the
   positive control both printed **rc=0** — `head`'s status, twice — on arms whose true codes are
   **1** and **0**. The compiler's refusal text was visible in the negative arm, which is the only
   reason it was noticed; a quieter check would have banked a clean, symmetric, entirely false result
   and reported the reviewer's premise as unfalsifiable. Iteration 233's instance is the same shape
   one gate over: a `jq` parse error left BOTH poll counts empty, `[ "" = "" ]` is true, and the gate
   that decides LANDED vs parked printed `ALL COMPLETE` over three still-running workflows.
   **Rule. (a)** In any comparison — two arms, before/after, check-vs-control — capture each side's
   status WITHOUT a pipe (`cmd > /tmp/out 2>&1; rc=$?`) and print the codes **beside each other**,
   because two codes on one line is what makes a false symmetry visible. **(b)** Before reading a
   symmetric result as a finding, ask what would have to be true for the arms to differ, and confirm
   your reader could have SHOWN that difference — a control proves the mechanism fires, and this
   proves the instrument can report it. **(c)** Where the arms are expected to differ, assert the
   difference explicitly (`[ "$rc_neg" -ne "$rc_pos" ]`) rather than eyeballing two values you
   printed; an equality you did not intend is then a loud failure instead of a quiet conclusion.
   **(d)** Mission-independent and shell-independent: the mechanism is whatever stands between the
   work and your reading of it — a pipe, a `jq`, a truncation, an API that 200s on an error page.
   The tell: you are about to report that two arms behaved the same, and the same command shape
   produced both readings.
3a. **A SEARCH THAT FOUND NOTHING IS A CLAIM, NOT A FACT — and so is any probe that came back
   empty** (added 2026-07-29 iteration 119; widened to all instruments iteration 120; the
   cheapest vacuous pass in the toolbox, and the one this loop keeps buying). An empty `grep` is
   indistinguishable from a `grep` that could never have matched — same silent output, same exit
   path, and it *feels* like evidence of absence. Four recorded instances, two of them in
   iteration 119 alone: (a) iteration 119 told its own planner, under an explicit VERIFIED-BY-ME
   label, that a 603-line test suite "runs nowhere in CI", from a grep of the root `Makefile` and
   `.github/workflows/` — the root `Makefile` is a ten-line `include` shell, `make/test.mk:19`
   defines the target and `ci.yml:133-144` runs it *with an anti-vacuity floor*; the planner had
   to refute it and delete a fabricated milestone; (b) the same iteration, ten minutes later,
   grepped `PASSES -lt` against a file reading `"$PASSES" -lt 45` and briefly believed its
   executor had claimed a change it never made; (c) iteration 105's `RunAgentBenchmark`, where a
   `RunAICheck(` hit proved a call site existed but never that anything reached it; (d)
   iterations 55–58's `rev-parse --short`, which fataled to stderr and printed nothing to stdout,
   wearing the all-clear's clothes for four iterations.
   Two more, both from iteration 120 and both showing the class is NOT limited to `grep`:
   (e) mission-world's unquoted `--include=*.go`, which **zsh glob-expands before `grep` ever
   runs** (`zsh:1: no matches found`), so the caller reads 0 hits from a command that never
   executed — it nearly shipped a fabricated "zero callers anywhere" fact to a sprint executor;
   the real answer was 11 call sites, and only a known-positive control in the same call caught
   it. (f) iteration 120's own MCP `tools/list` probe returned an EMPTY TOOL LIST for all five
   flag combinations — not a search at all, but a live protocol handshake that had failed
   (`rc=1`, `server is closing: EOF`) and whose empty result was indistinguishable from a genuine
   "no tools registered". It was caught only because a known-present tool was expected in the same
   output. **A remedy is an instrument and inherits the same burden of proof as the thing it
   verifies** — which is exactly how step 3's own `PIPESTATUS` advice went four-plus iterations
   without anyone noticing it printed nothing.
   Before a negative or empty result from ANY instrument — a search, a probe, a handshake, an
   exit-code check — becomes a fact you act on or hand downstream:
   **(i) prove the instrument can see a positive, in the SAME call** — assert something you KNOW
   is there comes back alongside the absence you are claiming; a pattern or probe that finds
   nothing anywhere is broken, not informative. Pair the check and its known-positive control so
   a broken instrument cannot masquerade as a clean result; where this becomes a committed test,
   an empty result set must FAIL LOUDLY (`t.Fatal("instrument failure")`), never pass;
   **(i-b) quote anything glob-shaped** — `--include='*.go'`, not `--include=*.go`; under zsh an
   unquoted glob-shaped flag value aborts the whole command before it runs;
   **(i-c) the SHELL is an instrument too, and zsh silently rewrites THREE shapes** (added
   2026-07-30 iteration 123; instances 3 and 4 of the zsh class after (i-b)'s glob and step 3's
   `PIPESTATUS`, each corroborated first-party against a `bash` control on the identical string
   before adoption). **Brace any variable followed by a colon** — in zsh, `"$rev:path"` applies
   `:h`/`:t`/`:r`/`:e` as HISTORY MODIFIERS and rewrites the string: measured on the rig with
   `c=abc123`, `"$c:host/x"` → `.ost/x` (`:h`=dirname), `":tail/x"` → `abc123ail/x`,
   `":runtime/x"` → `abc123untime/x`, `":extra/x"` → `xtra/x`, while bash returns all four literal
   and `"${c}:host/x"` is literal in both. This one is worse than the glob because `git show
   "$rev:path"` is THE git-archaeology idiom and **Gate 1 PRESCRIBES that exact shape** — its
   literal form is safe only because the rev is a literal, so the natural generalisation (put the
   rev in a variable) breaks it, **on the first letter of the path**, silently, into a plausible
   number when piped to `grep -c` (mission-world read `total_tables=0` for the commit that CREATED
   the schema). Reported by `world-coordinator`, which shares this skill but cannot edit it; V1's
   committed shell was audited CLEAN for this shape (matcher control-verified, scope widened,
   worktrees excluded), so it is a CONTROLLER-instrument rule here, not a code defect.
   **And `echo` is not a byte-faithful reader** — zsh's builtin `echo` INTERPRETS backslash
   escapes, so a literal two-character `\n` prints as a real newline. Iteration 123 hit this
   *inside the verification of its own pick*: `#541`'s defect WAS a literal `\n`, and the
   controller's known-positive control appeared to emit real newlines until `od -c` showed the
   bytes `5c 6e` — the instrument hid precisely the bug under test. To read bytes use
   `printf '%s'`, `od -c`, or `cat -v`; never `echo`. (`cat -A` is GNU-only — BSD `cat` rejects
   it, earned the same hour.) **And zsh does NOT word-split an unquoted variable** (added
   2026-08-04 iteration 140; the 5th zsh instance, and the first to produce a vacuous pass in a
   MUTATION TEST — the mission's own headline discipline). `FILES=$(grep -l … | head -4)` then
   `sed -i '' … $FILES` passes ONE argument whose value is four newline-joined paths, so `sed`
   fails `No such file or directory` on a filename that does not exist and **nothing is
   mutated**. In bash the same two lines work, which is why the shape reads as correct. Iteration
   140 ran exactly this to prove two re-centered CI gates could still fail; both gates returned
   **rc=0**, and an unexamined rc=0 there says "the assertion is vacuous" in precisely the same
   voice as "the mutation never ran". Only a *did-the-mutation-apply* control
   (`git diff --name-only | wc -l` — expected 4, got 1) caught it. Use an ARRAY —
   `FILES=($(…))`, then `"${FILES[@]}"` — and assert `${#FILES[@]}` before use. The general rule
   this mission already knows, in its sharpest form: **a mutation test needs proof the mutation
   LANDED before its result means anything**, because "the mutation didn't red" and "the mutation
   never ran" are the same exit code.
   **AND "LANDED" IS NECESSARY, NOT SUFFICIENT — A `sed`/REGEX MUTANT CAN CHANGE THE FILE, BUILD
   CLEANLY, AND HAVE MUTATED SOMETHING OTHER THAN WHAT YOU NAMED; THE DRILL THEN REPORTS A RED FOR
   AN ARM THAT WAS NEVER EXERCISED** (added 2026-08-25 V1 iteration 274; two first-party frictions
   in ONE iteration, both in the controller's own verification of a landed gate). Every mutation
   rule in this file asks whether the mutation *happened*: sha256 differs, `go build` rc=0, the
   file changed. All of those pass when the edit lands **in the wrong place**, so the sufficient
   question is not *did bytes change* but **did the specific thing I named actually change state**.
   Note the failure is invisible in the direction that matters: the arm goes red, which is what you
   predicted, so rule 3d's negative control agrees with you for the wrong reason.
   Measured, both in one iteration. **(a)** `sed 's/^ci: \(.*\)check-protocol-closure /ci: \1/'`
   — the greedy `\(.*\)` matched to the LAST occurrence, so it stripped `test-check-protocol-closure`
   and `test-check-tmpfile-hygiene` instead of the target named; the gate then reported exactly those
   two, and **the gate's own error message is what revealed which targets had really moved**.
   **(b)** A mutant appending a prerequisite with `sed 's/^ci: \(.*\)$/ci: \1 <target>/'` put it
   AFTER the line's trailing `## help text`, i.e. inside a comment — so it was never a prerequisite,
   and the arm redded on a *different* assertion than the one under test, making a false-green
   reproduction look like a successful one.
   **Rules. (a)** After mutating, assert the mutant's INTENDED EFFECT with a query against the
   system's own view — `make -pn | grep -c '^ci:.*<target>'` must go 1→0, `grep -c 'run: make X'`
   must go 1→0 — never against the file's bytes. **(b)** Prefer a structural editor (a few lines of
   python over the parsed form) to a regex over a line whose tail you have not read; `^X: \(.*\)$`
   on a line with a trailing comment is the commonest instance. **(c)** When an arm reds, read WHICH
   assertion failed and confirm it is the one the mutant targets — rule 3j's corollary
   ("read WHICH TEST failed, never the exit code alone") aimed at your own drill rather than at CI.
   **(d)** Mission-independent, and the generalisation is this file's own recurring shape one level
   down: **a mutation is an instrument too, so "the mutation landed" needs the same known-positive
   discipline as "the search found nothing."**
   **AND THE SAME NON-SPLITTING BREAKS `set -- $var`, WHICH IS THE SHAPE THAT LANDS IN *POLL
   READERS* — SO THE FALSE READING ARRIVES AT THE GATE THAT DECIDES LANDED vs PARKED** (added
   2026-08-20 V1 iteration 239; instance 1 is iteration 107's `set -- $res` Gate-3b poll, which
   printed `TIMEOUT — PARK` while its own last line read `completed success`; instance 2 is
   iteration 236's `set -- $pair` containment check; instance 3 is this iteration's Gate-3b poll).
   Note where this rule was NOT: `set -- $res` appears twice in this file already — both times as a
   *war story* about a past defect, neither time in THIS list, which is the one place a reader looks
   for "what does zsh silently rewrite?". That is this loop's own **guard the helper, miss the call
   site** shape aimed at its rulebook, and it is why three iterations paid for one construct.
   The mechanism is the clause immediately above — zsh does not word-split an unquoted variable — but
   the *surface* is different and worse: `FILES=$(…)` fails loudly (`sed: No such file`), while
   `set -- $st` succeeds, assigning the WHOLE string to `$1` and leaving `$2`/`$3` **empty**. Measured
   here on `st="3 1 0"`: unquoted → `$1='3 1 0'`; `set -- ${=st}` → `$1='3'`. Empty positionals then
   feed exactly the two-empty-values comparison Gate 3b's numeric-floor rule was written for — so a
   poll can report completion over still-running workflows, or, as at iteration 107, a park over a
   green. **Use `set -- ${=var}` (zsh's explicit-split flag), or avoid positional splitting entirely
   and read each value with its own command** — the latter is what this iteration switched to, and it
   is the form that also survives being copied into `bash`. Assert each value is a NUMBER before
   comparing it (the floor caught this one: it printed `INSTRUMENT FAILURE — not a verdict` instead
   of a completion, which is the only reason the bug was visible rather than banked). Mission-
   independent, and the generalisation is the one this list keeps re-earning: **when a shape has
   burned this loop twice in war stories, it belongs in the remedy list, not in the anecdote.**
   **AND `|| echo <default>` INSIDE `$(...)` IS THE SAME CLASS ARRIVING FROM THE OPPOSITE
   DIRECTION — IT IS DEFENSIVE SHELL THAT FIRES ON THE *SUCCESS* PATH, AND FOR `grep -c` THE
   SUCCESS IT OVERRIDES IS A LEGITIMATE ZERO** (added 2026-08-21 V1 iteration 244; proposed by
   `mission-world` iter-105 with two first-party instances in ONE iteration across TWO gates, and
   corroborated first-party in V1's own tool shell before adoption — sibling-claim ghost
   discipline). Every entry above concerns a shape the shell silently *rewrites*. This one is a
   shape the AUTHOR adds on purpose, to be careful, and that is what makes it durable: `|| echo 0`
   reads as a safety net, so nobody re-examines it. `grep -c` exits **1** when the count is
   legitimately **zero**, so `||` fires on an ordinary result and command substitution
   concatenates BOTH outputs — the variable becomes the two-line string `0\n0`, not `0`. The
   intent ("default to 0 if the command fails") is the exact inverse of the effect. Same for any
   command whose exit code reports a RESULT rather than a failure: `grep -q`, `diff`, `cmp`,
   `test`.
   Two surfaces, and the quiet one lands on a poll reader. **LOUD:** World's Gate-0 sweep ran
   `nc=$((nc + $(grep -cE "#N\b" "$f" || echo 0)))` and died `zsh: bad math expression: operator
   expected at '0'` — visible, cheap. **SILENT, and the dangerous one:** World's executor poll ran
   `done=$(grep -c "codex rc=" "$log" || echo 0)` then `[ "$done" != "0" ]`, which is **TRUE on the
   first tick** — `0\n0` != `0` — so the loop printed WRAPPER FINISHED while the executor was six
   minutes from done. Believing it means reading an empty worktree diff as a failed run, or ending
   the turn over a live background task, which is standing rule 7's vacuous pass exactly.
   Reproduced first-party in V1: `printf 'x\n' > /tmp/t; n=$(grep -c zzz /tmp/t || echo 0)` gives
   `od -c` bytes `0 \n 0`, `[ "$n" != "0" ]` is TRUE, the arithmetic form dies loudly, and the
   correct form on a matching pattern returns a clean `1`. Note what does NOT catch the silent
   surface: Gate 3b's numeric floor tests values **compared as numbers**, and this one is compared
   as a **string**, where a multi-line value passes every existing guard in this file.
   **Rules. (a)** Never write `|| echo <default>` inside a command substitution whose command uses
   its exit code to report a result. **(b)** Read the code deliberately instead —
   `c=$(grep -c X f); rc=$?` — and note `rc=2` is *no such file*, which is (i-d)'s scope trap, not
   a zero. **(c)** Or strip with `| head -1`. **(d)** Assert the value is a single numeric token
   before ANY use, **including string comparisons**, not only before arithmetic. **(e)** The same
   caution applies to any "robustness" wrapper placed between the work and your reading of it —
   a `2>/dev/null` that hides *no such directory*, a `|| true` that erases a real failure: V1's
   own iteration 244 greened a worktree-creation poll early because its readiness test was
   `grep -q .` against a log `git` was still writing progress into. Mission- and shell-independent.
   The generalisation is this file's own recurring shape aimed one level down: **a default is an
   instrument too — when the fallback fires on the success path, the default is not a safety net,
   it is the bug.** The tell: you wrote `|| echo` inside `$(...)`, and the command before it can
   exit non-zero on a perfectly ordinary result. **And a mutation red counts only when the mutant BUILDS —
   assert `go build ./...` (or the verify profile's compile step; under `ailang-code`,
   `ailang check`) rc=0 on the mutated tree BEFORE reading the test result, and prefer a mutant
   **AND THE ARRAY THIS RULE JUST PRESCRIBED IS 1-INDEXED IN ZSH, SO `${arr[0]}` IS EMPTY AND
   EVERY LATER INDEX IS OFF BY ONE — WHICH IN A *REPORTING* INSTRUMENT SHIFTS EVERY COLUMN AND
   SILENTLY DROPS THE LAST ELEMENT, WHILE THE OUTPUT STILL LOOKS LIKE A TABLE** (added 2026-08-17
   motoko iteration 8; instance 1 is iteration 140's word-splitting above, instance 2 is this
   iteration's Gate-0 sweep — same zsh-array class, new surface, and the first to land on an
   instrument whose whole job is to be *read*). The remedy immediately above says "use an ARRAY —
   `FILES=($(…))`, then `"${FILES[@]}"`". That is correct and it is where the next trap lives: in
   bash `${FILES[0]}` is the first element, in zsh it is **nothing at all**, and `${FILES[1]}` is
   the first. Iterating with `"${FILES[@]}"` is safe (which is why the prescribed remedy works);
   **indexing is not**, and the two sit one line apart in the same idiom.
   Why this earns a rule rather than a caution: the failure is *silent and plausible*. Measured
   here on Gate 0's weekly external-issue sweep, whose rule (b) exists precisely to make per-issue
   zeros auditable — an 8-file `FILES` array printed with `${counts[0]}…${counts[7]}` rendered
   every count under the WRONG file's header and never printed the 8th file (`mission-dashboard.md`)
   on any row. Nothing was blank enough to notice: the first column merely looked narrow. The
   orphan *verdict* survived only by luck — the accumulator summed the loop variable (`for f in
   "${FILES[@]}"`) rather than the display array — so a broken table sat beside a correct total,
   which is the worst possible arrangement, because the total certifies the table. Note the
   collision with rule 3a(i-d): a same-path control cannot catch this, since every column really
   did run; only a control on the *last* element, or asserting `${#arr[@]}` against what you
   printed, separates them.
   Rules: **(a)** never index a zsh array with a literal `0` — iterate with `"${arr[@]}"`, or index
   from **1**; **(b)** where a loop builds a display row, build the row by appending inside the same
   loop that reads the element (as the corrected sweep does) rather than by indexing a parallel
   array afterwards — parallel arrays and hand-written indices are the whole defect; **(c)** assert
   `${#arr[@]}` equals the number of fields you emit, and print the array's own first and last
   element once as a control; **(d)** this is mission-independent and shell-level, so it applies to
   every gate in this file that formats a table — Gate 0's sweep, Gate 1's check enumeration, Gate
   3b's per-workflow poll. The tell: you wrote `${something[0]}` in a `.sh`/tool-shell snippet on
   this rig, or a table's first column is unexpectedly empty and you assumed it was a formatting
   quirk. General form, and the reason it outranks its two instances: **a remedy is an instrument
   too (step 3a(i) already says so) — when this skill prescribes a construct, the construct's own
   footguns become the skill's problem, not the reader's.**
   Prefer a mutant
   that keeps every import used (neuter the call — `_ = f(x)`) over one that deletes a block**
   (added 2026-08-07 iteration 160; proposed by `mission-world` iter-62, which shares this skill
   but cannot edit it, and corroborated first-party in V1's own checkout before adoption —
   sibling-claim ghost discipline: all 6 `compil*` lines in this file are verify-profile/toolchain
   prose, not one about mutants, while the control fires — the mutation-LANDED rule above is
   present). "The mutant does not compile" is a THIRD fact wearing that same exit code, and it is
   the one rule 3d cannot catch, because a build-failure red arrives in **exactly** the direction
   you predicted — so the negative control agrees with you for the wrong reason and the mechanism
   was never exercised at all. Three instances in one `mission-world` iteration: a deleted refusal
   block redding on `imported and not used`; a non-matching regex leaving sha256 **unchanged**
   (LANDED=NO) whose fallback edit then stripped an import — two reds, zero information; and an
   opus executor that hit the class, self-reported it, and re-ran with a compiling mutant before
   believing the RED. Generalises past Go to any compiled or typechecked language. These shapes are silent and all survive `set -euo pipefail`;
   **(i-d) SCOPE THE KNOWN-POSITIVE CONTROL TO THE SAME PATH AS THE CHECK — A CONTROL RUN
   SOMEWHERE ELSE PROVES THE PATTERN, AND THE THING THAT BREAKS IS ALMOST ALWAYS THE SCOPE**
   (added 2026-08-12 V1 iteration 181; two first-party frictions, and the older one put a false
   fact in the charter for eleven iterations). Clause (i) says pair the check with a known
   positive **in the same call**. It never says *in the same scope*, and that is the half that
   fails: `grep -r <pattern> <dir>` over a directory that DOES NOT EXIST prints nothing, and
   piped to `wc -l` it reports a confident **0** — indistinguishable from a real absence, while
   a control aimed at a *different* directory comes back large and certifies it. Measured on V1,
   all three in one call: `grep -ril 'flatmap' stdlib/ | wc -l` → **0**; the SAME-PATH control
   `grep -ril 'export' stdlib/` → **0** (the signal you want — instrument broken); the
   DIFFERENT-PATH control `grep -ril 'export' std/` → **46** (the signal that misleads). The
   real path has always been `std/`; `stdlib/` has never existed in this repo. Iteration 170's
   weekly sweep recorded exactly that pair as *"grep 0, control firing"* and wrote into the
   charter that *"stdlib has NO `flatMap` … so the class is user-written eager flatMaps"* —
   false, and `std/list.ail:202` exports `flatMap`, `:250` `flatMapE`, `:99` `take`. That
   sharpening then sat in the queue row for **eleven iterations**, and it pointed `#617` at a
   docs/lint lane when both halves of the trap are the stdlib's own exported, taught surface;
   iteration 181 hit the identical trap on its own first Gate-2 command. Rules: **(a)** run the
   control against the SAME directory/file-set as the check — a same-path control over a bad
   path returns zero too, and that zero is the instrument-broken signal (i)'s whole design
   depends on; **(b)** `grep` already distinguishes them **in its exit code** — `1` is "no
   match", **`2` is "no such file"** — and `| wc -l` throws it away, which is step 3's
   exit-codes-through-pipes class aimed at the control rather than at the result; **(c)** where
   the scope is load-bearing, assert it exists before reading its emptiness (`test -d`, or a
   `find <dir> -type f | wc -l` denominator quoted beside the zero); **(d)** when a charter or
   queue row quotes "control firing", the control's SCOPE travels with it, exactly as rule
   3b(ii) makes a `-run`/`--version` narrowing travel with a green — "control firing" without a
   named scope is not a citation. The tell: you are about to write "there is no X in `<dir>`"
   and you have never confirmed that `<dir>` is a directory. Mission-independent: under
   `ailang-code` the same trap is a module path that does not resolve.
   **(i-e) TO TEST AN *ENUMERATION*, ADD A MEMBER — EVERY MUTATION RULE IN THIS FILE IS
   REMOVAL-SHAPED, AND REMOVAL CANNOT DETECT A LIST THAT IS SHORT** (added 2026-08-21 V1
   iteration 242; instance 1 is iteration 170's weekly-sweep enumeration, instance 2 is this
   iteration's builtin enumeration, and in BOTH the pattern was correct and the enumeration was
   the hole). Clause (i) proves the instrument can see a positive; (i-d) proves it looks in the
   right place. Neither asks the question an enumeration actually turns on — **is the list of
   things being checked COMPLETE?** — and no control over an EXISTING member can answer it,
   because an existing member is in the list by assumption. Rule 3d ("remove the mechanism and
   require a red") and rule 3j ("a guard is not a gate until something reds when you remove it")
   both point the same way, so a gate can pass every drill this file prescribes and still be blind
   to the case it exists for: **a NEW thing that was never enumerated**.
   Measured here. A CI gate required every registered `_list_*` builtin to be delegated from `std/`
   or carry an explicit reason. It enumerated by AST-parsing `Name:` fields for string literals,
   and its commit message claimed *"names are derived, never hardcoded, so a new builtin cannot
   slip past"*. Five removal-direction mutants all red — revert the delegation, launder it behind a
   comment, two stale-exemption shapes — and every one of them passed the rule as written. The
   evaluator then **added** a builtin registered as `Name: someConstant`, an `*ast.Ident` rather
   than a `*ast.BasicLit`: the mutant compiled, the gate stayed **GREEN** at an unchanged
   "31 registered", and the new builtin needed neither a call site nor an exemption. Iteration
   170's instance is the same shape one gate over — the sweep's per-issue grep was fine and the
   *issue list* was short, so four orphaned issues were invisible while the known-tracked control
   fired correctly; that rule's remedy (assert the list's length) is a count you must already know,
   which is the very fact in dispute.
   **Rules. (a)** For any check that iterates a derived set — registered builtins, open issues,
   workflow files, exported symbols, config keys — run one mutant that **ADDS a member the
   enumerator might not see**, chosen to differ from existing members in the way the enumerator is
   most likely to key on (a constant instead of a literal, a different registration call, a file in
   an unscanned directory, a differently-named object). Require the count to MOVE, not merely the
   verdict to flip. **(b)** Prefer an enumeration that is complete BY CONSTRUCTION over one that is
   complete by inspection: a live registry, `go list`, an API's own listing — the thing the system
   itself uses. Here the fix was to read the two live registries instead of parsing source, and
   note the trap inside the fix — **neither registry was complete alone** (18 and 26 names, union
   31), so "use the live one" needed measuring too. **(c)** Where an enumerator must stay
   source-derived, say in the record what shape it can miss, rather than claiming it cannot be
   evaded. **(d)** Mission-independent, and it is this file's own *guard the helper, miss the call
   site* shape aimed at the mutation rules themselves: **a removal proves the check FIRES; only an
   addition proves it LOOKS.** The tell: your gate's evidence is a list of things that went red
   when you deleted them, and you have never made it go red by creating something.
   **(ii) widen once before concluding** — drop the quoting, the anchors, the file filter, and the
   directory scope (a root `Makefile` includes; a workflow calls a make target; a caller lives in
   a file type your `--include` excluded); **(iii) prefer the tool that cannot miss** — `make -pn`
   over grepping makefiles, a language server or `go list` over grepping for callers, `gh api
   .../check-runs` over listing runs; **(iv) label it honestly** — "grep found no X" is not
   "there is no X", and the difference is exactly the provenance distinction Gate 2 already
   demands. The tell that you are about to pay for this: you are about to write "there is no…",
   "it runs nowhere", or "nothing calls it" on the strength of one command that printed nothing.
3b. **A PASSING check is a claim too — match its SCOPE and its VERSION to the sentence you cite it
   for** (added 2026-07-31 iteration 124; the mirror of 3a, which only covers *empty* results).
   3a stops you trusting a check that found nothing. This one stops you over-reading a check that
   came back **green**: the command really ran, really passed, and still does not support the
   claim attached to it. Both instances below came from ONE quorum round, both were caught by the
   reviewer rather than the author, and one of them was the controller's own evidence:
   (a) **Scope.** A sandbox port-bind denial blocked `go test ./internal/effects`, so the
   controller isolated the new tests — `-run 'Recorded|StreamRecorded'` → 4/4 PASS — and cited
   that while routing the patch. `gemini-3-1-pro` correctly rejected it: running the patch's OWN
   tests proves the new code works, never that it **breaks nothing existing**. That claim needed
   the whole suite minus the denied test
   (`go test ./internal/effects -skip TestNetHTTPRequestBytes_RoundTripSHA` → rc=0, **658 PASS**)
   — a different command answering a different question.
   (b) **Version.** The designer verified an example with `ailang prompt --version v0.16.2` and
   cited it as evidence of correctness at the **v0.31.0** target. Green, honest, and worthless for
   that sentence — the instrument was fifteen minor versions stale. This is the stale-binary class
   step 1 already guards for *builds*, but nobody re-checks it for *tools invoked with an explicit
   `--version`*.
   Before a green result becomes evidence: **(i)** name the sentence it supports, then check the
   command's scope actually covers that sentence — "does X still work" and "did I break anything"
   are never the same command; **(ii)** when a `-run`/`-skip`/`--version`/single-package filter
   narrowed the run, the narrowing is PART of the finding and travels with it — never dropped when
   the result is quoted downstream; **(iii)** a denial, skip, or flake that forced the narrowing is
   UNINFORMATIVE, so re-run the widest form that excludes only the denied item rather than quietly
   citing the narrow one; **(iv)** use the negative framing as the acceptance test — "what would
   this command still pass under, if the thing I am claiming were false?" The tell: you are about
   to write "the tests pass" or "it checks clean" while the command you actually ran carried a
   `-run`, a `-skip`, a `--version`, or a single package.
   **(v) AN ENUMERATION YOU TRUNCATED IS NOT AN ENUMERATION, AND A VALUE YOU TRANSCRIBED IS NOT A
   MEASUREMENT** (added 2026-08-04 iteration 137; three instances in ONE spawn directive, all three
   caught by the DESIGNER rather than by the controller who wrote them). Everything above is aimed
   at commands narrowed by a *flag*; these two shapes narrow the result with no flag to notice, and
   both landed in a directive under an explicit VERIFIED-BY-ME heading — the exact laundering Gate 2
   forbids. **(a) `| head -N` / `| tail -N` silently turns a complete-looking list into an
   incomplete one.** Iteration 137 ran `go list ./... | grep -v /internal/ | head -20`, read those
   20 lines back as the whole answer, and told the designer there was exactly ONE importable library
   package. There are two — `testutil` sat past the cut. The command was right, the output was real,
   and the sentence built on it was false. If you are about to write "the only", "all of", "there
   are N", or "nothing else", the limiter comes OFF, or is replaced by a count (`| wc -l`) that
   cannot lie by omission — and you quote the count beside the list. **(b) A number or SHA copied
   out of a DOCUMENT is a claim about that document, not about the repo.** The same directive
   asserted Lane A's squash was `a81d66983`, transcribed from an adjacent charter row; that is
   `#517`'s Lane A, not `#498`'s (`aa02f0d9f`), and one `git log -1 --format=%s <sha>` catches it.
   A near-identical sibling literal is precisely what makes this shape easy. The same directive also
   said a struct had 16 fields where the listing it was quoting showed 15. Rule: anything a
   downstream role will treat as ground truth — especially a SHA, a count, a line number, or a file
   path — is re-derived by command at the moment you write it, never carried over from prose you
   read earlier. The tell for both: you are quoting a *quantity* or an *identifier* and cannot name
   the command that produced it **in this session**. (Corollary, cheap and repeatedly earned: when a
   sub-agent refutes one of these, that is the loop WORKING — record it in Ruled out and fix the
   provenance habit, per Gate 2's rule (d), rather than treating the refutation as noise.)
   **(vi) A DOCUMENT'S VERIFICATION LOG CAN REFUTE THAT DOCUMENT'S OWN ACCEPTANCE CRITERIA — DIFF
   THE TWO BEFORE ROUTING, BECAUSE NOTHING ELSE DOES** (added 2026-08-04 iteration 138; 2nd instance
   after iteration 135). Everything above polices a check at the moment you *run* it. This is the
   same error one step later, and it is now the likelier one: the measurement was taken correctly,
   written down honestly, and then a conclusion elsewhere in the SAME FILE was built on the version
   of reality that predates it. Nobody re-reads a 28-row Verification Log against a 27-item AC list,
   so the contradiction ships. Iteration 138's pick had row **V18** recording that the boundary gate
   iterates three fixed package sets none of which contain `apiserver` — and its M3 acceptance
   criterion was still "`make check-boundaries` passes", a gate that passes identically whether or
   not the new code violates the boundary it is cited to protect. **Two reviewers cleared that doc
   across two full quorum rounds and neither caught it**, which is the point: quorum reads for design
   soundness, not for internal consistency between a doc's evidence and its claims. Iteration 135 was
   the same shape — a planner evidence row measured at pre-split `HEAD`, then cited for an ordering
   claim at a position that row never covered. So the cross-check is a CONTROLLER duty at pick time,
   not something a reviewer or the designer will do for you: for each acceptance criterion that names
   a command, find the verification row covering that command and confirm the row's measured SCOPE
   actually reaches the thing the AC is about. Where it does not, the AC is **vacuous** — replace it
   with one that can fail, and say so in the routing evidence. The tell: an AC of the form "`make X`
   passes" or "the suite is green" for work that lives somewhere the doc has already measured `X` as
   not looking. Cheap generalisation, worth more than the two instances: **a long document is an
   instrument too, and its Verification Log is the control — if the log and the claims disagree, the
   claims are what's wrong.**
   **(vi-b) THE INSTRUMENT FOR (vi): SWEEP FROM THE *OLDEST* DECLARED MEASUREMENT BASE, BECAUSE THE
   NATURAL CHOICE GIVES A FALSE ALL-CLEAR** (added 2026-08-04 iteration 141, adopted from a
   `mission-world` proposal — World shares this skill but cannot edit it, so it proposes and V1
   applies). Rule (vi) tells you to diff a document's Verification Log against its claims. It never
   names a **base**, and the base *is* the whole instrument. A doc revised in place across several
   iterations accumulates rows measured at different commits, and its header may declare more than
   one — so the natural move (sweep from the newest base, or from the doc's last revision) silently
   exempts every row measured before it. Measured by mission-world on
   `design_docs/planned/w-bench-load-confound.md`, whose header declares two bases:
   `git diff --name-only <NEWER>..HEAD -- ':!design_docs'` returned **ZERO** files — a confident
   clean bill of health on a genuinely stale document — while `<OLDER>..HEAD` returned **8** and
   found every stale row. Three premise rows had gone false from a single commit; one iteration
   named two, the next repaired those two and declared the class closed, and the **planner** found
   the third. The sweep had checked the rows someone had named rather than the commit that caused
   them. Concretely: **(a)** parse EVERY base the Verification Log declares and sweep from the
   **earliest**; **(b)** treat a row as unverified whenever the diff touches any file that row
   cites — not merely when someone flagged it; **(c)** pair the diff with a known-changed file as a
   control, so an empty result proves the instrument ran rather than that nothing moved (rule 3a,
   applied to freshness); **(d)** re-measure rather than reason — a row's age is not evidence it is
   still true, and neither is its author's confidence. General form: **a document is only as fresh
   as its OLDEST measurement**, so it degrades precisely in the rows nobody has reason to re-read.
   Two recorded frictions, both V1's own — iterations 135 and 138 — and (vi) was authored at 138
   without an instrument. Reviewers will not close this gap for you: quorum reads for design
   soundness, not for freshness against HEAD (five rounds and two reviewers missed all three rows
   above).
   **(vii) A DESIGN DOC AND ITS SPRINT PLAN ARE TWO DOCUMENTS DESCRIBING ONE SPRINT, AND REVISING
   EITHER SILENTLY ROTS THE OTHER — DIFF THE PLAN'S MILESTONE SECTION AGAINST THE DOC'S ACCEPTANCE
   CRITERIA AT PICK TIME** (added 2026-08-05 iteration 146). Rules (vi) and (vi-b) police
   consistency *within* one document. This is the same error across the **file boundary**, and it
   is more likely, because the two files are written by different roles at different times: the
   designer writes the doc, quorum reviews the doc, the planner reads the doc **once** and emits a
   plan — and from that moment nothing re-diffs them. Every later revision lands in exactly one of
   the two. A mid-sprint human directive is the worst case, because it revises the doc by
   definition and no one thinks of the plan as affected. Iteration 145 applied Mark's `D5` ruling
   by editing the doc — AC3 → AC3′(a/b/c) plus a brand-new **AC10(d)**. Nothing touched the plan,
   whose M3 task list still read `AC10 (a) … (b) … (c)`. Routed as written, that milestone would
   have shipped **without the tripwire whose entire purpose is to red when the follow-up item
   lands** — i.e. the loop would have silently dropped the mechanism connecting two queue rows.
   Measured at iteration 146: the plan said `AC10 (a)` in **2** places while the doc carried
   `AC10(d)` in **4**; and the rot ran BOTH ways — the doc's Implementation-Plan section still
   bundled a milestone with workflow edits the *newer* plan had split out, and the doc still said
   "5 CI legs" in **6** places despite its own `V34` having measured **6**. So neither file
   dominates: whichever was edited last is fresher *in that spot only*. Concretely, at pick time:
   **(a)** for the milestone you are about to route, list the ACs the DOC says it closes and the
   ACs the PLAN's milestone section names, and diff those two lists — a one-minute read that the
   executor cannot do for you, because a cross-provider executor is handed the plan and has no
   reason to doubt it; **(b)** when they disagree, state explicitly in the executor directive
   which document wins (normally the doc, as the reviewed artifact) and quote the delta verbatim,
   rather than assuming the executor will notice; **(c)** treat a doc revision landed by any
   iteration OTHER than the one that wrote the plan as positive evidence of divergence — check,
   do not hope; **(d)** file the residue as explicit cleanup work rather than fixing it inline,
   so the sprint's own docs milestone owns it. The tell: you are routing milestone N of a
   multi-milestone sprint whose design doc was edited after its plan was written — which, in a
   loop that answers human directives by editing the doc, is most of them.
   **(viii) THE HOST PLATFORM IS A NARROWING YOU NEVER TYPED, SO THERE IS NO FLAG TO NOTICE — AND
   IT IS THE ONE NARROWING THAT SILENTLY CHANGES WHAT YOUR CODE *MEANS*, NOT JUST WHICH TESTS RAN**
   (added 2026-08-13 V1 iteration 195; three recorded frictions, one first-party and measured).
   Rule 3b(ii) makes a `-run`/`-skip`/`--version`/single-package narrowing travel with the finding,
   because you typed it and can therefore see it. Rule 3b(v) adds the shapes with no flag —
   `| head -N`, a transcribed value. **The platform is the purest member of that second family:**
   you never wrote `--os=darwin`, nothing in the output says `darwin`, and every command reads as
   unqualified. So "the tests pass" is uttered honestly about a matrix leg you cannot run, and rule
   3g does not catch it — 3g asks whether you ran the right *commands*, and here the command list
   was complete and every one of them was green.
   What makes it worse than an ordinary narrowing: on another platform the same source has
   **different semantics**, so the failure is not "a test I didn't run" but "a test whose input the
   code never received". Iteration 195's own instance, filed as BLOCKING by the evaluator against
   the controller's PR: two new negative arms set `t.Setenv("HOME", …)` to drive a guard through
   `os.UserHomeDir()` — which reads **`USERPROFILE`** on windows and `$home` on plan9. On Windows
   the runner's real profile resolved anyway, so the guard never saw the input the test believed it
   supplied, and both arms **failed for the PLATFORM rather than for the code** — inside a sprint
   whose entire subject was arms that do not pin what they claim. The controller's PR body had
   claimed *"Gates (all outside the sandbox) … rc=0"* with no Windows caveat, and every one of
   those commands really had returned rc=0. Two corroborating frictions in this mission's own
   charter, both Windows, both invisible locally and both caught only by Gate 3b: iteration 120's
   *"Windows `.exe` fix Gate 3b caught"*, and the recorded finding that *"Windows env vars are
   case-INSENSITIVE, so `http_proxy`/`HTTP_PROXY` are ONE variable"*.
   Rules: **(a)** before writing "the gates pass" anywhere a human or a downstream role will read
   it, name the platform — "green on darwin/arm64; windows and ubuntu legs unrun locally" — so the
   narrowing travels exactly as 3b(ii) requires of a `-run`; **(b)** when a diff touches anything
   whose meaning is per-GOOS — env-var *names* (`HOME`/`USERPROFILE`), path separators and drive
   letters, case-sensitivity of filesystems AND of env vars, line endings, symlinks, file
   permissions, executable suffixes, temp-dir shape, `os/user` and `os.UserHomeDir` — treat a
   single-platform green as **UNINFORMATIVE for that behaviour**, in the same voice the codex recipe
   uses for a sandbox denial; **(c)** prefer a helper that sets EVERY variable the stdlib consults
   over the one your machine happens to read, since the portable form costs a line and the
   non-portable one costs a CI cycle plus a merge block; **(d)** Gate 3b is the only instrument that
   sees the whole matrix, so a red there on a leg you cannot reproduce is **information, not noise**
   — read which leg and why before reaching for a re-run, and note that the required-contexts list
   may not include it, so a matrix leg can be genuinely broken while the merge button stays green.
   The tell: you are about to write "all gates green" and every command you ran executed on one
   machine, whose operating system you did not mention because it did not occur to you that it was
   a parameter. Mission-independent: under `ailang-code` the same axis is whatever `ailang check`
   resolves differently per host.
   **(ix) A COUNT IS ONLY TRUE INSIDE THE SCOPE IT WAS TAKEN IN, AND THE SCOPE IS THE PART NOBODY
   WRITES DOWN — SO THE NUMBER SURVIVES BEING COPIED INTO A WIDER SENTENCE, WHERE NOTHING ABOUT IT
   LOOKS WRONG** (added 2026-08-14 V1 iteration 202; proposed by `mission-world` iter-86 with three
   first-party instances in ONE iteration across THREE roles, and corroborated first-party in V1's
   own artifacts before adoption — sibling-claim ghost discipline). Rule 3b(ii) makes a
   `-run`/`--version`/single-package narrowing travel with a **green**. Rule 3a(i-d) makes a scope
   travel with an **empty** result. Nothing makes a scope travel with a **non-empty count** — and
   that is the shape that keeps shipping, because a cardinality reads as a fact about the world
   rather than as a fact about the command that produced it. Note the asymmetry that makes it
   durable: the count is usually **correct where it was taken**, so re-deriving it reproduces the
   number and confirms the error.
   World's three, one iteration: a queue row saying "four context-free read getters" that missed a
   fifth **on its own scope**; a controller directive headed VERIFIED BY ME that placed three
   functions in `store.go "~229-290"` where a definition grep returns **0** (they are in
   `writer_lock.go`; `store.go` holds CALL SITES, read as definitions — that one violates the
   EXISTING 3b(v)(b), so it is a rule broken rather than a gap found, and the DESIGNER refuted it,
   which is Gate 2 rule (d) working); and the designer's own correction becoming a false universal,
   "all five read getters are context-free" as a property of the STORE, which has **six**. Two of
   those are the SAME number corrected in the SAME iteration and wrong both times, in **opposite
   directions**.
   V1's corroborating instance is the purest form, because nothing was miscounted at all: iteration
   202's PR body carried a mutation table of **8** rows covering **9** test functions (7 + 2,
   measured at `e86ffc36f`), and the evaluator read the 8 as an arm count and filed the mismatch.
   The number was right in its scope — mutations — and wrong in the sentence a reader built from it.
   **Rule.** Before a count becomes a fact you act on or hand downstream, write the scope INTO the
   sentence — "five getters **on the daemon read path**", "eight **mutations** across nine arms" —
   never a bare "five getters" or "eight". Where the count will be quoted downstream, quote the
   enumerating command beside it, exactly as 3a(i-d) requires "control firing" to carry its scope.
   The tell: you are about to write "all N", "the only", "there are N", or "N of them" about a set
   whose boundary you chose and did not state. Mission-independent; under `ailang-code` the same
   trap is a module set.
3c. **"THE SERVICE" IS AN ASSUMPTION — a probe identifies the endpoint you REACHED, never the
   service you NAMED** (added 2026-08-01 iteration 130; 2nd instance of this gap after iteration
   129 recorded "ollama server is 0.31.2, up 11 days; client already 0.32.1" as a fact and built a
   remediation on it). Rules 3a/3b cover results that come back empty or green. This one covers
   results that are **specific, non-empty, confidently phrased, and about the wrong object**: the
   probe answered honestly for whatever it happened to connect to, while a second copy of the same
   service was live the whole time. That failure mode has no tell in the output — it looks exactly
   like a clean reading, which is why it survived a full iteration.
   Iteration 130 measured, stable 6/6: `127.0.0.1:11434/api/version` → **0.31.2**, while
   `[::1]:11434/api/version` → **0.32.1**. Two `ollama serve` processes — one launchd-managed, one
   app-managed — bound to the same port on different ADDRESS FAMILIES since a reboot, sharing a
   model store but holding separate GPU state. `ollama --version` and `ollama ps` talk to the IPv4
   one, so the CLI reported an idle GPU while a 37 GB model was resident on the other. Iteration
   129's single-instrument reading was not wrong about what it measured; it was wrong about **what
   it was measuring**, and the remediation it proposed ("restart to get onto 0.32.1") would have
   restarted the wrong server and left the rig on the older one.
   Before a probe's answer becomes a fact about a NAMED service: **(i)** ask what the client
   RESOLVED to, not what you typed — `localhost` is TWO addresses on a dual-stack host, and Go,
   node, curl and python order them differently, so two clients can reach two different servers
   from one identical string; prefer a literal address in anything load-bearing; **(ii)** probe
   each address family / socket / port explicitly and compare, instead of probing "the service"
   once; **(iii)** when two access paths disagree about version or identity, the default
   explanation is **TWO INSTANCES**, not one instance misreporting — enumerate processes AND their
   parents (`ps -o pid=,ppid=`) before theorising; **(iv)** a service under two process managers
   has no single owner, so "restart it" is underspecified — check what a watchdog PROBES against
   what it RESTARTS, because it may heal the rig back onto the very copy you were retiring. The
   tell: you are about to write "the server is version X", "nothing is loaded", or "the service is
   up" on the strength of one endpoint, one CLI, or one `ps` line.
3d. **A RESULT THAT CAME BACK RED IN THE DIRECTION YOU PREDICTED IS THE MOST SEDUCTIVE CLAIM OF
   ALL — IT NEEDS A NEGATIVE CONTROL EXACTLY AS MUCH AS AN EMPTY RESULT NEEDS A POSITIVE ONE**
   (added 2026-08-04 iteration 142; pre-registered by iteration 140 as "watch-item instance 1,
   bar is two", and this is instance 2). Rule 3a covers results that come back **empty**; 3b
   covers results that come back **green**. Neither covers the third shape: the check **failed,
   exactly as you expected it to**, and you bank that as proof your mechanism works. It arrives
   as confirmation, so nothing in you wants to test it — which is precisely why it survives
   longer than the other two. The failure mode is always the same: **co-occurrence read as
   causation.** Something else was also capable of producing that red, and no control separated
   them.
   Two instances, both this mission's own, both landing inside otherwise-careful iterations:
   **(a)** iteration 140 — a deterministic tier-gate regression was attributed to a known runner
   flake (`#587`) because both commits went red in the same window. Wrong platform *and* wrong
   failing test; the real regression sat on `dev` for ~2h and was reported to the human as a
   flake. The lesson recorded then was narrow ("two commits red in the same window is not
   evidence they failed for the same reason") because it had one instance.
   **(b)** iteration 141 — the controller *predicted* an acceptance criterion would be vacuous,
   ran its poisoned-proxy command once, observed `rc=1`, and recorded that as **refuting its own
   prediction**, crediting the poison for an HTTP error page. Iteration 142 measured it: the
   poison never touched the request. AILANG's `Net` effect builds its transports by hand with
   `Proxy == nil`, so the proxy is never consulted; the error page came from **`httpbin.org`
   itself** — the known-flaky third party that the very sprint under design exists to remove.
   The original prediction had been CORRECT. Poisoned `rc=0 ok 0.767s`, unpoisoned `rc=0
   ok 0.724s`: **outcome-identical**. A single unpoisoned run in the same breath would have shown
   the same red and exposed it instantly, and the AC would not have shipped into a sprint plan.
   Before "it failed, so the mechanism works" becomes a fact you act on or hand downstream:
   **(i) run it once with the mechanism REMOVED** — no poison, no flag, no patch, no gate — and
   require the outcomes to DIFFER. Same outcome means you measured the environment, not the
   mechanism, and the size of the difference is the size of your evidence;
   **(ii) name every other thing that could produce this exact failure** before crediting the one
   you were hoping for — a flaky third party, an outage, a cache, a runner, an unrelated
   concurrent change. If you cannot rule them out by command, say "consistent with" rather than
   "caused by";
   **(iii) attribution must match on MECHANISM, not on timing** — same failing test AND same
   platform AND same layer, never redness plus adjacency (that is (a)'s form of this rule);
   **(iv) a prediction you set out to test is not refuted by one observation that merely
   contradicts it** — it is refuted by an observation whose *cause* you established. Iteration
   141's error was not the measurement; it was concluding causation from a single arm.
   The tell: you are about to write "this proves the guard works", "the drill is non-vacuous",
   "confirmed — it fails as expected", or "same failure as `#NNN`", and every command you ran had
   the mechanism switched ON.
3e. **BASELINE EVERY ACCEPTANCE COMMAND ON A PRISTINE TREE — A GATE ALREADY RED AT BASE MEASURES
   THE REPO, NOT YOUR CHANGE, AND A CONTROL RUN AFTER AN EARLIER STEP HAS MUTATED SHARED STATE IS
   NOT A CONTROL** (added 2026-08-05 iteration 147). Rules 3a/3b/3d police a *result* — empty,
   green, or red. This one polices the **base you measured against**, which nothing above names,
   and it has two faces that look nothing alike until you see they are the same mistake.
   **(a) The gate was already red before you touched anything.** A sprint plan's acceptance list
   is written by someone reading the repo, not running it, so it routinely contains commands that
   do not pass on unmodified `dev`. Iteration 145's executor found `go build ./...` — a plan gate —
   **fails identically on untouched dev** (`cmd/wasm` and `gen/main` have no native `main`).
   Iteration 147's plan gate `actionlint <files>` → rc=0 is **rc=1 at base**, on 5 pre-existing
   shellcheck findings. Such a gate can only be waved through or blamed on the sprint, and both
   happen. So: before routing, run each acceptance command on the base and record the result *as
   part of the criterion*. If it is already red, the AC is broken — fix the AC and say so, rather
   than "fixing" the code or quietly dropping the gate.
   **(b) Your control was contaminated by a step of your own change.** This is the dangerous face,
   because it produces a confident, symmetric, entirely false all-clear. Iteration 147 saw three
   binary-gated tests SKIP locally, and ran the obvious control: the SAME assertion in its
   *pre-change* form. Both arms skipped identically, which reads as "pre-existing, not mine" — and
   it was recorded as a local environment artifact, with a `make quick-install` to move past it.
   It was in fact **the change's own defect**: an earlier step, `go mod download all`, had written
   to the tracked `go.sum`, and the binary-staleness detector compares binary mtime against the
   newest Go source. Both arms ran in a tree that step had **already** contaminated, so the control
   could not distinguish and the symmetry was an artifact of the shared mutation, not evidence of
   innocence. It shipped, and CI red-lighted the milestone's own acceptance step ~40 minutes later.
   Concretely: **(i)** a control is only a control if it runs from a tree in the state the
   *baseline* was in — re-clone, `git stash`, a fresh worktree, or at minimum restore the mutated
   file and its **mtime**; **(ii)** enumerate what your change WRITES, not just what it reads —
   tracked files, caches, mtimes, env, installed binaries — and ask which later assertion consumes
   each one; a step that mutates a tracked file mid-run is the tell; **(iii)** when two arms agree,
   ask what they SHARE before concluding the variable does not matter — identical results across
   arms is equally consistent with "the variable is irrelevant" and "both arms are already
   broken"; **(iv)** if you cannot obtain a pristine base, the control is UNINFORMATIVE — say so,
   exactly as the sandbox rule requires, rather than banking the symmetry.
   The generalisable point, and the reason this outranks its two instances: **an environmental
   explanation is always available for a symptom you caused**, and it is more comfortable than the
   alternative, so it wins by default unless the base is pinned down by command.
   **AND A BASELINE IS A CLAIM ABOUT THE ENVIRONMENT YOU RAN IT IN, NOT ABOUT THE COMMAND — SO A
   GATE LIST BASELINED IN YOUR OWN SHELL AND HANDED TO A SANDBOXED LANE CERTIFIES AN ENVIRONMENT
   THAT WILL NEVER EXECUTE IT** (added 2026-08-24 V1 iteration 270; proposed by `mission-world`
   iter-119 with a first-party instance, and corroborated first-party in V1's own iteration within
   the hour before adoption — sibling-claim ghost discipline). Clauses (a) and (b) above pin the
   *tree* a baseline runs against: pristine, uncontaminated, re-measured rather than assumed. Both
   are silent about the *lane*. So a controller can follow 3e to the letter, and the codex recipe's
   false-green #4 to the letter, and still hand an executor a gate that is **unsatisfiable by
   construction** in the sandbox it is about to run in — because the axis deciding satisfiability is
   one no rule asked about. The failure is not a misread verdict; it is a gate list that CANNOT be
   green, married to a directive asserting every entry was measured green.
   The mechanism is already in this file, filed one gate away: false-green #3 teaches that
   `workspace-write` denies loopback binds, so any suite touching `httptest`/servers fails with
   `bind: operation not permitted`, and that the CONTROLLER must re-run such gates OUTSIDE the
   sandbox. That rule is about *reading a verdict*; nothing points it at *composing a gate list*.
   Guard the helper, miss the call site — this file's own recurring shape, aimed at its own hands.
   World's instance: `go test ./host/workbench ./host/daemon ./host/boundary` is rc=0 in its shell
   and unsatisfiable inside the sandbox on two independent paths (`d.Listen()` and
   `httptest.NewServer`), inside a drill protocol requiring that arm rc=0 as a control after EVERY
   mutant — so the milestone could not be executed in the lane it was routed to, however correct the
   work. V1's instance, same day: a scoped
   `go test ./internal/gen/golang/... ./internal/eval_harness/...` baselined **rc=0** outside the
   sandbox and shipped as gate G4 in a directive stating "every one rc=0 there", returned **rc=1**
   inside on a denied `httptest.NewServer` bind. It cost nothing only because the directive
   independently told the executor to label such results `UNINFORMATIVE UNDER SANDBOX`, and it did —
   the label saved the verdict; it did not make the gate list correct.
   **Rules. (a)** Baseline a gate list in the LANE THAT WILL EXECUTE IT, or state in the directive
   AND the evidence row **which environment was certified** — "rc=0 on darwin/arm64 outside the
   sandbox; G4 not established inside `workspace-write`". **(b)** Before routing, ask of each gate
   whether it binds a socket, needs the network, writes outside the workspace, or reads a path the
   sandbox excludes; those are the entries that differ by lane, and they are enumerable in advance
   rather than discoverable at cost. **(c)** Prefer a gate satisfiable in the executing lane over one
   that is thorough and is not — and where the thorough one matters, keep it as a CONTROLLER gate run
   outside, never as an executor obligation. **(d)** Mission-independent, and it generalises past
   sandboxes to every lane boundary: a CI runner, a different GOOS (rule 3b(viii) is this same rule
   aimed at the host), a container, a read-only checkout. The tell: you are about to write "every one
   of these is rc=0 at base" in a directive, and the shell you measured in is not the shell that will
   run them.
3f. **A REVIEWER'S OBJECTION IS A CLAIM TOO — WHEN A QUORUM BLOCKS ON AN "UNVERIFIED PREMISE", THE
   CONTROLLER'S JOB IS TO *MEASURE* IT, NOT TO FORWARD IT** (added 2026-08-06 iteration 150). Every
   rule above polices claims flowing *downward* — from a sub-agent, a designer, a judge, a document.
   This one polices a claim flowing *upward*, from a reviewer, and it is the one shape the loop
   reflexively treats as authoritative: a quorum reject is a *verdict*, arrives with a
   `proposed_fix`, and costs money, so the natural move is to route it straight to the designer.
   But an objection of the form "the doc never established that X" is itself an **unverified
   premise** — the reviewer did not check either; it correctly noticed that *nobody had*. Forwarding
   it buys a revision round to answer a question one command can settle, and the answer frequently
   **refutes the objection outright** or, better, *shrinks* the work.
   Iteration 126 is instance 1: two quorum rounds lost to premise objections, and the fix recorded
   then was narrow ("hand the designer the measurement rather than the objection"). Iteration 150 is
   instance 2 and generalises it. `gpt5-6-sol` blocked a design on the grounds that the repo might
   already contain reusable HTTP transport/RoundTripper machinery the new mechanism would duplicate.
   The controller ran the audit itself: **0** custom `RoundTripper`s, **0** `DefaultTransport` uses,
   **0** `Transport.Clone`, no shared factory anywhere (control: **29** inline `http.Client{}` sites,
   so the zeros are measurements). The objection was answered, not litigated. The same pass then
   produced a fact the doc never had — Go's `DefaultTransport` sets `Proxy: ProxyFromEnvironment`,
   so bare clients are *already* inside the egress boundary and only hand-built nil-`Proxy`
   transports can escape — which converted a counted claim ("we found seven sites") into a
   **derivation** ("seven is all there can be"). No revision round could have produced that; only
   running the check could.
   And the same instrument works on an objection you *cannot* satisfy. R2's surviving objection
   asked for a `go/packages` AST analyzer because textual matching cannot see aliased imports,
   `new(http.Transport)`, post-construction assignment, factories, or custom `RoundTripper`s. Rather
   than park a vague "is the audit complete?", the controller **tested the reviewer's own
   hypothesis**: all five shapes are **zero at HEAD**, each with a firing control. That did not
   resolve the objection — the reviewer's point about *future* escapes still stands — but it
   converted an open-ended completeness dispute into a bounded, one-word human decision (cheap gate
   now vs durable gate in-sprint), which is the difference between a useful park and a stalled one.
   Concretely, on any quorum reject: **(a)** classify each objection as *premise* (asserts something
   about the codebase) or *design* (disputes a choice); **(b)** run every premise objection yourself
   before routing anything, with rule 3a's known-positive control, and hand the designer the
   measurement; **(c)** where an objection is not satisfiable in-loop, still measure whatever part of
   it *is* empirical, so the park carries numbers and the human decision is one word rather than an
   investigation; **(d)** record refuted objections in Ruled out — a reviewer refuted by measurement
   is the loop working, exactly as rule (d) already says for sub-agents. The tell: you are about to
   forward a `proposed_fix` whose first step is "verify that…", and you have not run it.
3g. **YOUR LOCAL GATE SWEEP IS A HAND-PICKED SUBSET; THE CI JOB'S OWN COMMAND LIST IS KNOWABLE, SO
   DERIVE IT INSTEAD OF REMEMBERING IT** (added 2026-08-06 iteration 152; 2nd instance after
   iteration 151). Rules 3a–3f police individual results. This one polices the *set* of checks you
   chose to run before pushing — and nothing above names it, because a hand-picked sweep never looks
   incomplete: every command in it passes, so the report reads "all gates green" right up until a
   REQUIRED remote context goes red. The subset is chosen from memory of what usually matters, and it
   drifts from CI silently, because CI gains steps and your habit does not.
   Iteration 151 caught a changelog entry misfiled into the root `CHANGELOG.md` **by hand**, noting
   that file is an INDEX and release-manager builds notes from `changelogs/*` — anything left in the
   index is *silently dropped from the release*. Iteration 152 made the identical mistake and did
   **not** catch it: seven local gates (`go test` on four package sets, `vet`, `build`,
   `check-file-sizes`, `check-boundaries`, `gofmt`) all rc=0, and `make check-changelog` — which was
   simply not in the habit — red-lighted the REQUIRED `test` context. Note the asymmetry that makes
   this worth a rule: the gate existed the whole time and was one command away.
   Concretely, before pushing: **(a)** derive the list rather than recall it — `make -pn`, the
   workflow file, or most reliably the previous run's own log (`gh api
   repos/<o>/<r>/actions/jobs/<id>/logs`, then extract the commands it echoed) — and run the ones
   your diff can plausibly break; **(b)** pair the extraction with a control, because an empty
   command list is rule 3a's trap wearing this gate's clothes (iteration 152's first two extraction
   attempts returned nothing and only a `grep -c` control revealed the pattern was wrong, not the
   log); **(c)** when a remote gate reds anyway, add that command to the local sweep in the same
   iteration rather than noting it — a lesson recorded but not wired in is what produced instance 2;
   **(d)** this is mission-independent: under `ailang-code` the same rule points at `ailang check` /
   `ailang test` / `ailang ai-check` plus whatever that repo's CI adds. The tell: you are about to
   write "all gates pass" and you assembled the gate list from memory.
3h. **AN EXECUTOR'S DEVIATION FROM THE PLAN IS A CLAIM IN *BOTH* DIRECTIONS — ADJUDICATE IT BY
   MEASUREMENT, AND NEVER BY A "DEVIATIONS ARE SUSPECT" PRIOR, WHICH GETS MOST OF THEM BACKWARDS**
   (added 2026-08-07 iteration 159; pre-registered by iteration 158 after `mission-world` delivered
   the third instance, and corroborated first-party in V1 before adoption — sibling-claim ghost
   discipline). Rules 3a–3g police claims you or a reviewer produced. This one polices the claim an
   executor hands you when it did something other than what the plan said, and it is the one shape
   with no safe default: **the executor's own report cannot distinguish the good case from the bad
   one, because in both the executor states a reason and the reason is usually TRUE.** Only running
   the check separates them.
   Three cross-mission instances, and note which way they point: World iter-58 **better than the
   plan** (the plan was wrong; the judge scored it −5 anyway), World iter-60 **vacuous** — and it
   was easy to wave through precisely *because* its stated reason was true — and World iter-61
   **better than the plan**, self-reported (the executor was told to route writes through
   `confinedWrite`, did so, then volunteered that this collides with a landed assertion the
   directive never mentioned, and raised an exact-count from 42 to 43 while keeping it an equality).
   Two of three came out in the executor's favour, so a "deviations are suspect" heuristic would
   have discarded the two best outcomes. V1's own record carries the same shape three times
   (`v1-mission-log.md`: the M2 direct-SQLite deviation, the cohort/baseline hash-mismatch
   deviation adjudicated ACCEPTABLE, and the `consec >= K` escalation deviation APPROVED) — each
   one adjudicated ad hoc, by a different argument, with no written rule; that absence is the gap
   this closes, not the deviations themselves.
   Concretely, on any deviation: **(a)** restate it as a checkable proposition — "this is strict,
   not a weakening", "the plan's step was impossible", "this is equivalent" — and find the command
   that would come out differently if it were false; a deviation you cannot phrase this way is not
   yet understood; **(b)** run that command **in both arms**, exactly as rule 3d requires for a red
   you predicted: World iter-61's "strict, not a weakening" was checkable because the count stayed
   an *equality*, so dropping either write still reds; **(c)** hand the deviation to the evaluator
   as a **named target to attack**, rather than hoping it notices — an independent judge that agrees
   after being pointed at it is evidence, one that never looked is not; **(d)** treat a self-reported
   deviation as *better* evidence than a silent one, never worse — an executor that names which of
   your instructions was under-specified has done Gate-2 work for you, and the plan is what needs
   fixing; **(e)** record the verdict in Gate 4's evidence row with the command, because "adjudicated
   acceptable" with no measurement is exactly the vacuous pass this mission keeps closing elsewhere.
   The tell: you are about to write "sound deviation", "adjudicated acceptable", or "the executor's
   reasoning is correct" and every word of your justification came from the executor's own report.
3i. **A TEST-PLAN ROW'S "KILLS WHICH MUTATION" COLUMN IS A CLAIM, AND IT IS THE ONE CLAIM IN THE
   WHOLE SPRINT NOBODY EVER CHECKS — RUN THE NAMED MUTATION AGAINST THE ROW THAT NAMES IT, NOT
   AGAINST THE SUITE** (added 2026-08-07 iteration 162). Rules 3a–3h police claims about the
   *codebase*. This one polices a claim about a *test*, and it survives every gate above because a
   test written to catch defect D and a test that merely passes are indistinguishable while D is
   absent — which is the entire time, until the day it isn't. The plan asserts the kill, a quorum
   reads the plan for design soundness, an executor implements the row faithfully, an evaluator
   confirms the row exists and is green, and at no point does anyone apply D. So the row ships as
   documentation of protection that was never present.
   Measured here, on the milestone whose *whole point* was a three-call-site sweep. §5.6's S11 row
   read "run the fixture twice; every `PropertyResult.Seed` is non-zero, the three differ,
   both runs agree — **kills guarding two sites and missing `contract_domain.go:89`**." It does not.
   `Seed` is stamped into each path's result *initializer*, alongside the RNG construction rather
   than by it, so replacing `newRNG(r.propertySeed(…))` with a constant at that exact site left the
   **entire seed suite green** (mutant asserted LANDED via sha256 and BUILDS via `go build` rc=0,
   per this skill's own mutation rule). Two of three swept sites were unguarded, in a repo whose
   named recurring failure shape is *guard the helper, miss a call site*. Note what did NOT catch
   it: the executor implemented S11 exactly as specified; the row's own author had reasoned
   correctly about *what* to observe and wrongly about *whether that observable moves*; and the
   sprint's grep-based sweep AC passed throughout, because the call sites really were swept — it
   was the behavioural pin that was hollow.
   The general shape, which is worth more than the instance: **an assertion that observes a value
   set ALONGSIDE the mechanism cannot fail for the reason it claims** — only one observing a value
   set BY it can. Ask, per row, "which write does this read?"; if the answer is a sibling statement
   rather than the mechanism, the row is decorative. Concretely: **(a)** at pick/route time, for
   each test-plan row carrying a named mutation, name the *observable* and check it is downstream
   of the mechanism, not adjacent to it; **(b)** run the named mutation, per row, with only that
   row's test selected (`-run`), because a suite-wide green hides which member did the killing and
   a suite-wide red hides which member did not; **(c)** when a mutant survives, the finding is that
   the ROW is wrong, not that the code is — repair the row, and say in the commit which mutant used
   to survive, since that sentence is the only durable evidence the pin is real; **(d)** where no
   observable exists (here the forall path had none — every such property errors out before
   sampling), record the residual explicitly in the code and the AC rather than letting arm-count
   arithmetic imply coverage; a named gap is cheap, an assumed one is not. The tell: a test-plan row
   says "kills X" and you are about to accept it because the test is green.
   **AND THE OBSERVABLE CAN BE DOWNSTREAM OF THE MECHANISM AND *STILL* NOT DISCRIMINATE IT — ASK NOT
   ONLY "WHICH WRITE DOES THIS READ?" BUT "WHAT ELSE WRITES THIS VALUE?"** (added 2026-08-14 V1
   iteration 200 at the ≥2-friction bar iteration 199 pre-registered; instance 1 was 199's
   absence-only assertion, instance 2 is this iteration's over-subscribed enum). The rule above
   catches an observable set *alongside* the mechanism. It does not catch one the mechanism really
   does write — where **other mechanisms write the same value**, so the assertion passes for any of
   them. Both instances shipped green, both sat inside otherwise-careful mutation drills, and in
   both the sibling rows redded convincingly enough that the drill looked like it had worked.
   Instance 1 was the pure form: an assertion that a below-threshold log line is **ABSENT**, which
   "the filter suppressed it" and "nothing was ever emitted" satisfy equally. Instance 2 is the form
   that will fool you *after* you have learned instance 1, because it asserts a **present, specific
   value**: `TestA2AExitNonzeroFails` required the A2A task state to be `"failed"`. It is a
   three-value enum, and **every** failure mode reaches `"failed"` — including one where the code
   under test never executed. Measured by neutering the test's own precondition (the IO capability
   grant, without which the effect layer refuses before `exit()` is ever entered): **5 of the 6
   exit arms correctly failed and that one PASSED**. The production fix was correct throughout; the
   test certifying it was hollow. Note how little the two surfaces resemble each other — an absence
   and a named string constant — and that the underlying defect is identical: **the observable's
   value set is larger than the mechanism's**.
   **The drill, and it is cheap enough to run on every arm you are unsure of: neuter the test's
   PRECONDITION, not the production code, and require the arm to die.** Every mutation rule in this
   skill mutates the thing under test; this one mutates the *setup* — the capability grant, the
   fixture load, the seeded row, the injected clock — i.e. whatever must hold for the mechanism to
   run at all. An arm that survives its own precondition being removed is not testing the mechanism.
   Run it over the whole arm set in one call, because the informative output is the **split**: the
   arms that die are the honest ones, and the survivors name themselves. Concretely: **(a)** for
   each assertion, enumerate what else in the system can produce that exact value — an error path, a
   default, a zero value, a shared enum branch, a timeout; **(b)** prefer an observable whose value
   is *unique* to the mechanism (the message text, not the status enum; the computed value, not the
   fact that something was written); **(c)** where the discriminating observable is unavoidably
   coarse, pair it with a second assertion that is fine-grained, exactly as the routes/MCP siblings
   here asserted the message and the a2a arm did not; **(d)** apply this to POSITIVE CONTROLS too,
   which is where iteration 200 also got caught — its control fixture (`main() -> int = 42`) needed
   no capability, so it passed with or without the grant and never proved what it was cited for. A
   control that cannot fail is not a control, and a green suite hides that fact perfectly. The tell:
   your assertion compares against an enum member, a boolean, a status class, or an absence — rather
   than against a value only this code path could have produced.
3j. **WHEN A MILESTONE'S DELIVERABLE IS A REFUSAL, THE UNIT OF MUTATION IS THE *BRANCH*, NOT THE
   MILESTONE — AND A ONE-SHOT ACCEPTANCE COMMAND IS NOT A GUARD** (added 2026-08-08 iteration 164;
   proposed by `mission-world` iter-63 with three first-party instances, corroborated here on V1's
   own freshly-landed milestone before adoption — sibling-claim ghost discipline). Every mutation
   discipline this skill has aims at a mutation someone NAMED: the plan's `named_mutations`, the
   doc's mutation table, rule 3i's "kills which mutation" column. None points at the refusal
   branches of a validator or a flag guard the executor writes *during* the sprint. Those ship with
   a green suite and no pin, and the green is what makes the gap invisible — a function whose
   contract is "refuse X" can have N distinct refusal branches and pins for none of them.
   **Rule:** for any function added or modified by a milestone whose contract is a refusal,
   enumerate its refusal branches and require **one neutering mutation per branch** before the
   milestone closes.
   **⚠ THE MECHANICAL FIRST CUT THIS RULE USED TO PRESCRIBE — `grep -c 'return .*fmt.Errorf(.*%w'`
   — IS BLIND TO EVERY REFUSAL THAT DOES NOT WRAP, AND IT RETURNS A LARGE CONFIDENT NUMBER WHILE
   DOING SO** (fixed 2026-08-11 V1 iteration 178; proposed by `mission-world` iter-72 with a
   first-party measurement, corroborated in V1's own checkout before adoption per the
   sibling-claim ghost discipline — and V1's numbers are WORSE than the ones that motivated it).
   `%w` is a *wrapping* convention, and a terminal refusal has nothing to wrap, so the qualifier
   silently excludes exactly the branches most likely to be the last word. This is rule 3a's trap
   in its most seductive form: the count is non-zero and large, so it is its own known-positive
   control and reads as a thorough enumeration. Measured on World's `transitionreg.go`: the
   prescribed cut saw **22** of ~**55** refusal returns, and **0 of the 2** `errors.New` branches
   that shipped with no coverage — neutered together, a tampered object with a broken revision
   chain read as sound, mutant landed and building, whole package rc=0. Measured repo-wide in V1
   (non-test Go): prescribed cut **1781**, all `return … fmt.Errorf(` **4273**, plus **20**
   `return … errors.New(` — so the qualifier alone hides **~2,500** refusal returns here, and the
   dominant blind class is *non-wrapping `fmt.Errorf`*, not `errors.New`. Use
   `grep -cE 'return .*(fmt\.Errorf|errors\.New|status\.Error)\('`, or better, phrase the task as
   **"every `return` on an error path"** and pair the count with a known-positive control (rule
   3a). Generalises to any language whose terminal and wrapping error constructors differ — under
   `ailang-code`, the same question is asked of whatever that repo's refusal form is.
   **And the more general half — ANCHOR THE ENUMERATION TO THE DIFF, NEVER TO THE DESIGN DOC'S
   DECISION LIST.** World's sprint implemented this rule as an audit of the branches the doc
   *froze*, so the rules a later decision added and the wrappers the milestone itself wrote were
   outside the enumeration **by construction**: the gap was not an oversight *within* the audit's
   scope, it **was** the audit's scope. A doc can only freeze the branches it knew about, and the
   ones a milestone writes during the sprint are precisely the ones no pre-sprint enumeration can
   contain — which is the same reason this rule exists at all. Neuter with `if false && <cond>` rather than deleting the block, so every import stays
   used and "the mutant does not build" cannot masquerade as "the guard fired" (the class the
   mutation-BUILDS rule above already names). A genuinely unreachable branch is an acceptable
   outcome **when declared in the code and in the AC**; an undeclared one is a guard nobody is
   protecting.
   **⚠ BUT "UNREACHABLE" IS A CLAIM ABOUT A PLATFORM, NOT ABOUT A BRANCH — AND THE CLAUSE ABOVE
   INVITES YOU TO DECLARE IT FROM THE ONE MACHINE THAT CANNOT TEST IT** (added 2026-09-02 V1
   iteration 318; instance 1 is iteration 195, whose two negative arms set `t.Setenv("HOME")` to
   drive a guard that on windows reads `USERPROFILE`, so the branch the test believed it was
   exercising was never entered; instance 2 is this iteration, the same error in mirror image —
   a branch declared UNREACHABLE that windows reached in minutes). Rule 3b(viii) already says the
   host platform is a narrowing you never typed. It is written about **greens** — "the tests
   pass" — and nothing points it at the opposite claim. That gap matters more, because a green is
   provisional by nature while *unreachable* is stated as a property of the code, gets written
   into a comment, and is then inherited by every later reader as settled.
   Measured here, and note the shape: **three independent parties certified the same branch
   unreachable and all three were wrong for the same reason.** The executor wrote "unreachable by
   construction"; the independent sonnet judge was *explicitly asked to break it* and reported "I
   could not devise a test-only way to trigger this"; the controller wrote "requires a
   monotonic-clock anomaly". All three reasoned on darwin/arm64, where `time.Since` has nanosecond
   granularity, so `schedulingLatency <= 0` looked impossible. On a coarse-clock platform a
   sub-tick interval reads back as exactly `0s`, and CI reddened **both** windows jobs
   deterministically — `--- FAIL: TestMessageWatcherStart (0.00s)` /
   `instrument failure: initial task scheduling latency 0s is outside (0, 1s)` — on the first push.
   **Adding reviewers did not help and could not have**: independence in the *reviewer* does not
   buy independence in the *platform*, and the platform was the shared premise.
   **Rules. (a)** Before declaring a branch unreachable, name WHY: **control flow** (a `t.Fatal`,
   an early `return`, a type invariant) is platform-independent and may be declared; a **value
   property** (a clock's granularity or monotonicity, a filesystem's case sensitivity, an env
   var's name, a path separator, an integer width, a locale) is platform-scoped and may NOT be
   declared from one host. Write the reason in the comment, because the reason is what a later
   reader must re-check. **(b)** Where the reason is a value property, the declaration is
   **PROVISIONAL until the matrix has run** — CI is the only instrument that sees it, so treat a
   red there as the measurement rather than as noise (3b(viii)(d)), and say in the record which
   legs certified it. **(c)** Prefer a design that makes the question moot: here the round-2 floor
   ALREADY made a zero latency safe (`max(20 × 0, 1s)` = the historical bound), so the guard was
   rejecting a value its own neighbour had handled — when a defensive branch and a fallback both
   cover the same input, the branch is not defence, it is a second opinion that can disagree.
   **(d)** A guard whose trigger condition is a property of the *measuring instrument* rather than
   of the *system under test* is the highest-risk member of this class; ask what the instrument
   reads on the least-precise platform in the matrix. Mission-independent; under `ailang-code` the
   same axis is whatever `ailang check` resolves differently per host. The tell: you are about to
   write "unreachable", "cannot happen", "by construction" or "defensive only" in a comment or an
   AC, and every machine you have run on is the one under your desk.
   **AND THE SYMMETRIC HALF, WHICH IS THE ONE THIS LOOP ACTUALLY BOUGHT NEXT: A PLATFORM-SCOPED
   PROPERTY MAY NOT BE DECLARED *UNNECESSARY* FROM ONE HOST EITHER — AND THAT DIRECTION IS WORSE,
   BECAUSE IT ARRIVES AS A DELETION THAT EVERY RULE IN THIS FILE APPLAUDS** (added 2026-09-02 V1
   iteration 319; instance 1 is iteration 318 immediately above, instance 2 is this iteration, which
   committed the mirror-image error **after reading 318's rule at Gate 1 the same iteration**). The
   rule above is written entirely about *asserting* a branch cannot be reached. Nothing points it at
   the opposite move — deciding a guard, a stabilizer, a sleep, a retry or a margin is surplus and
   removing it. That move is strictly more attractive, because rule 3n(b) rewards deleting an
   unpinned hunk, rule 3n(a) rewards removing nondeterminism rather than enlarging a margin, and a
   judge that measures the hunk unpinned hands you the evidence. All three fire correctly, and all
   three are silent about the axis that decides it.
   Measured here. An executor added two arm-scoped stabilizers to a shell test and **self-reported**
   that its directive was under-specified without them (rule 3h(d)). The independent judge then
   measured one of them **unpinned** — reverting that hunk alone left the suite 42/42 green — and its
   justifying comment demonstrably **false**. The controller measured the other unnecessary across
   **8 local runs**, quiet and under 8× CPU contention. Both were deleted; the clean suite even got
   *faster*, and the milestone's own killer mutant still killed. CI then reddened **deterministically
   on the first push**: the stub driver exited before the lane entered its sampling loop, so the code
   path under test was never entered at all. Fast darwin/arm64 always wins that race; the runner does
   not. The executor had been right, and two independent reviewers on the same platform had overruled
   it — **adding a reviewer cannot help, because the platform is the shared premise, not the
   reasoner.**
   **Rules. (a)** Before deleting a guard, margin, sleep, retry or ordering constraint as
   unnecessary, name the property that makes it so — **control flow** (a `return`, a type invariant)
   is host-independent and may be decided locally; a **value or timing property** (scheduling order,
   clock granularity, process lifetime, filesystem or env-var semantics) is platform-scoped and the
   deletion is **PROVISIONAL until the matrix has run**. **(b)** "N local runs came back green" is
   the *identical* evidence 318's rule already rejects, wearing the opposite sign — count it as
   evidence about your host, never about the code. **(c)** Weight a **self-reported** deviation
   accordingly: rule 3h(d) already says it is better evidence than a silent one, and this is the case
   that proves it — the executor knew something about the executing environment that neither reviewer
   could see. Before overruling one, ask what the deviator observed that you have not. **(d)** When
   you restore something CI forced back, write into the code *why local greenness was not enough*, or
   the next reader deletes it for your exact reason. Mission-independent. The tell: you are about to
   remove something as surplus, your evidence is that things stay green without it, and every one of
   those runs was on the machine under your desk.
   World's three instances, one iteration, three roles: a refusal term satisfiable by
   nothing an operator can mint (two quorum rounds read past it); its replacement left the ENTIRE
   `host/broker` package green under `if false && …`; and once the evaluator was handed that as a
   named target per rule 3h(c) it found six more, the executor's own audit twelve, and `AC9` ended
   at 20 negative arms.
   **V1's corroborating instance, measured on the milestone landed the same hour** — and note it is
   *not* the shape you would look for, because the branch was not unguarded by oversight, it was
   guarded by something that never runs again. M2C's `--seed`/`--random-seed` mutual exclusion
   (`cmd/ailang/main.go:148`) was covered only by the sprint plan's `AC6(d)` shell grep of
   `conflict.err` — wired into **no** make target and **no** CI job (control: `check-golden-drift`
   appears in both `make/test.mk` and `ci.yml`), and **zero** `*_test.go` mentioned either flag
   (control: `--seed` appears in ten test files). Measured: `if false && seedSet && randomSet`
   LANDED (sha256), BUILDS (`go build` rc=0), and the **entire rest of `cmd/ailang` is rc=0 with
   the defect present** (`-skip` the new test, `ok 19.000s`). So the generalisation worth more than
   either instance: **a guard is not a gate until something reds when you remove it** — an
   executor's one-shot acceptance command proves the branch worked *once*, on a tree that no longer
   exists, and reads in the plan exactly like coverage.
   **Corollary, which nearly cost World the finding: read WHICH TEST failed, never the exit code
   alone.** One controller probe returned rc=1 in exactly the predicted direction and its only FAIL
   was a pre-existing load flake (measured 2/5 by the evaluator) — banking the exit code would have
   recorded a pin that did not exist. This is rule 3d aimed at a mutation run rather than at a CI
   red, and it is why the drill above scopes with `-run` and quotes the assertion text. Pair it
   with the inverse arm this iteration used: run the suite `-skip`-ing your new test under the same
   mutant, and require rc=0 — that is what proves *your* test is the killer rather than a
   bystander. The tell: a milestone's headline verb is "reject", "refuse", "validate" or "exit
   non-zero", and your mutation list has one entry.
   **AND THAT `rc=0` INVERSE IS CORRECT ONLY FOR A MUTANT PROVEN TO BE SINGLE-TEST — FOR ANY OTHER
   IT IS UNSATISFIABLE BY CONSTRUCTION, AND FAILING IT READS EXACTLY LIKE "YOUR ARM IS A
   BYSTANDER"** (added 2026-08-19 V1 iteration 227; proposed by `mission-world` iter-94 at the
   ≥2-friction bar, corroborated first-party in V1's own log before adoption — sibling-claim ghost
   discipline — and then met a third time in the adopting iteration's own drill). The sentence
   immediately above prescribes the inverse arm **unconditionally**, and that is the half that
   fails: a mutant whose blast radius exceeds one test reds *other* arms too, so `-skip <your arm>`
   returns non-zero however honest your arm is. The criterion is then measuring the **mutant's
   reach**, not the arm's honesty — and it fails in the direction that reads as a confession, so
   the natural response is to weaken or delete a test that was doing its job. Note which mutants
   trigger it: the ones that reach furthest, i.e. the ones whose guards matter most.
   The symmetric error is worse and is what World hit. A doc or test-plan row that **states** an
   expected red set instead of **running** it will score a **correct** mutant as a failed arm:
   `MU-DEADLINE-DETACH` declared a two-test set plus "any red outside that set fails the arm", and
   the measured set is **four** — the extra two are the mutant's own phenotype. Implemented to the
   letter, the doc would have rejected a working mutation; reproduced by four roles independently.
   V1's own two: iteration 225 saw **4 of 12** mutants fail the `rc=0` criterion, read at first as
   "4 vacuous arms", until enumeration showed M1 killed **5** arms, M7 **6**, M4 four, M5 two —
   with the named arm among the killers every time; and iteration 227 found **5 of 10** mutants
   broad-blast (red sets of 3, 2, 4, 4 and 6), so the criterion was inapplicable to half the drill.
   **Rule. (a)** Classify each mutant by blast radius *before* choosing a criterion — that means
   running it once and reading the red set, not predicting it. **(b)** Single-test mutant →
   `-skip <arm>` rc=0 is correct, and it is the strongest evidence available; keep it. **(c)**
   Otherwise the expected result is an **enumerated set of failing test names, produced by running
   it**, and the check is "the named arm is IN the set, and every other member is explained" —
   never `rc=0`. **(d)** A red set written into a plan, doc or mutation table before anyone executed
   it is a claim, not a measurement (rule 3b(v)(a) aimed at a red SET rather than a count, and
   3b(ix)'s scope discipline aimed at the same); a document cannot enumerate a set it has not run.
   **(e)** Report *sole killer* separately from *set membership*: sole-killer is the finding a green
   suite can never give you, and collapsing the two is what made iteration 225's one genuine
   zero-killer arm illegible among four false alarms. Mission-independent, and the generalisation
   is this skill's own recurring shape: **a criterion is an instrument too** — when it fails, ask
   first whether it could have succeeded. The tell: you are about to write "vacuous arm",
   "bystander" or "the drill did not pin this", and the mutants that failed your criterion are the
   ones you would have predicted to reach furthest.
   **AND A GATE'S COVERAGE IS A PROPERTY OF ITS *ENUMERATOR*, ONE LEVEL BELOW ITS BRANCHES — SO
   EVERY BRANCH CAN BE PINNED AND THE GATE STILL SEE NOTHING** (added 2026-08-12 V1 iteration 187;
   proposed by `mission-world` iter-77 with a first-party instance, corroborated in V1's own
   checkout before adoption per the sibling-claim ghost discipline). Everything above asks *how
   many ways can this mechanism refuse*, and iter-75's dual asks *how many ways can the forbidden
   thing be spelled*. Neither asks **who decides what counts as an input at all**. An enumerator's
   blind spot is invisible to every downstream assertion **by construction**, which is exactly why
   a full set of arms, mutations and a high evaluator score all agree: the input never reached the
   branches. World's instance: a gate refusing any `.ail` module outside an allowlist, four
   refusal branches all mutation-killed, five committed arms, ten mutations, evaluator 93/100 —
   defeated by `SNEAKY.AIL`, because the enumerator is `find -name '*.ail'` and `-name` is
   case-sensitive. The gate exited **rc=0** and printed its own success line **byte-identical to
   the pristine baseline's**; same-call control, `-name` saw **4** files, `-iname` saw **5**.
   **V1's corroborating instance is a different mechanism — wrong SCOPE, not wrong case — and a
   live 46-file hole.** `make fmt-check-ail` (`make/code-health.mk:28-39`) advertises "examples/ +
   stdlib/" and enumerates `find examples stdlib -name '*.ail' 2>/dev/null`. **`stdlib/` has never
   existed in this repo** (`test -d stdlib` → NO; the real path is `std/`, → YES), `find` reports
   that only on stderr, and the `2>/dev/null` swallows it. Measured in one call: as-written
   **400** files, `find examples std` **446**. So 46 stdlib `.ail` files sit outside a gate that
   still prints `✓ All .ail files are canonical`. Worse, its empty-enumeration branch prints a
   **GREEN checkmark and `exit 0`** — the anti-vacuity floor iteration 183 added to
   `test-stdlib-ail` is absent here. Note this is the *same wrong path* rule 3a(i-d) already
   records from iteration 181, now inside a gate rather than inside a controller probe: a repo
   with one wrong-path habit will grow enumerators around it.
   **Rule.** Before trusting any set-compare, allowlist, manifest or sweep gate, ask what its
   enumerator **cannot see** — case, symlinks, extension variants, roots that do not exist,
   permissions, build tags, ignore files, `head`/`tail` limiters. Pair the enumeration with a
   deliberately **widened** control in the same call (`-iname` beside `-name`, `find` beside
   `go list`, the parent directory beside the named one) and require the two counts to agree, or
   record the delta as a declared limitation. Assert the roots exist (`test -d`) rather than
   reading their emptiness, since a missing root returns zero exactly like a clean one — that is
   rule 3a(i-d)'s scope trap, aimed at a committed gate instead of at your own probe. And any
   enumerator-fed gate needs an anti-vacuity floor: an empty set must FAIL LOUDLY, never print a
   checkmark. The tell: you are about to trust a gate whose branches you have all mutation-killed,
   and you have never asked what feeds it.
   **AND A GATE THAT CONSULTS MORE THAN ONE LIST, ENUMERATION OR CALL HAS MORE THAN ONE PLACE TO
   BE VACUOUS — FLOORING ONE OF THEM READS AS COVERING THE GATE** (added 2026-08-24 V1 iteration
   271; three instances across TWO files, one first-party). The clause immediately above asks what
   a gate's enumerator *cannot see*. This one asks a question one level to the side and cheaper to
   answer: **how many enumerations does this gate actually run, and does each carry its own
   floor?** Rule 3a(i-d) already states the principle — scope the known-positive to the same place
   as the check — but it is written for a controller's ad-hoc probe, whose remedy is `test -d` and
   a grep exit code. Here the control is a *permanent branch in committed code*, which is what
   makes it durable: a reviewer sees a named, deliberate known-positive check a few lines above
   and reads the gate as floored. Nobody asks *which* enumeration it floors.
   Three instances. **(1)** Iteration 269: `make lint` has a **scan** path list and a separate
   **verdict** path list, and the sprint plan's edit widened only the scan — golangci-lint would
   have LOOKED at the new package while the gate stayed unable to REFUSE anything found there.
   **(2)** Iteration 268: `check_protocol_closure.sh` floors arm 1 with four branches and arm 2
   with two. **(3)** Iteration 271, first-party and the sharpest: *within* arm 2, the deps
   enumeration was floored (`R6` rc/non-empty, `R7` known-positive) and the **module-root
   enumeration was not** — no rc check (its status was discarded as the head of a pipeline), no
   non-emptiness, no known positive. That second enumeration is the one the allowlist check
   actually consumes, so `R7` was a control on a *different call*. Measured with a stub `go`
   delegating every other call to the real toolchain: reducing the roots call alone from **10** to
   **0**, plain deps untouched at **224**, left the violator loop iterating zero times and the gate
   printing its green checkmark at rc=0.
   Note the gradient across the three: arm-vs-arm (visible to anyone reading the file), then
   list-vs-list inside one target, then call-vs-call inside one arm. **They get harder to see as
   they get closer together**, and the last one is invisible to a reader who has just satisfied
   themselves that "arm 2 has a known-positive".
   **Rules. (a)** Enumerate the gate's *inputs* before auditing its *branches*: grep every
   invocation that produces a list the gate later reads (`go list`, `find`, `git ls-files`, an API
   listing, a second `grep`), and pair the count with a known-positive control so a short
   enumeration cannot masquerade as a complete one. **(b)** Require each enumeration to carry its
   own three legs — the producing command's status captured **without a pipe**, a non-emptiness
   assertion, and a known positive **queried against the very file or variable the check
   consumes**. **(c)** When you fix one, say in the commit which enumerations you audited and
   which you floored; "the gate is floored" is a claim whose scope is exactly one list. **(d)**
   Mission-independent, and under `ailang-code` the same shape is a check that resolves one module
   set and asserts over another. The tell: a gate has a known-positive control you find
   reassuring, and you have not checked that the control and the check read the same list.
3k. **IF THE PRODUCT HANDS A HUMAN SOMETHING TO RUN, A TEST MUST RUN EXACTLY THAT — A TEST THAT
   REBUILDS THE SAME COMMAND BY A SECOND ROUTE VERIFIES YOUR ARITHMETIC, NEVER YOUR ARTIFACT**
   (added 2026-08-08 iteration 166). Rules 3a–3j police claims about the codebase, about a check's
   scope, and about a mutation. None of them points at the class of deliverable that is *itself* a
   string a user is told to execute — a replay command, a suggested fix, a `--help` example, a
   copy-pasteable line in a guide, a generated URL. Those ship with green suites by construction,
   because the natural test computes the correct value independently and asserts on *that*, so the
   emitted text is never touched by anything. The bug then lives exactly where it is most visible
   to users and least visible to CI.
   Two instances, both first-party. **(a)** Iteration 166: `ailang test` printed
   `replay: ailang test --seed 0 All Tests` on every failing property — `All Tests` is the
   *aggregate display label* (`NewSuiteResult("All Tests")`), not a path, so the command the tool
   told the user to run could not run. It had been broken since the milestone before, through a
   quorum, a sprint plan, an evaluator PASS and a Gate-3b green, because the acceptance criterion
   covering replay (`AC6-M2`) reconstructed `--seed=${seed} "$tmp/multi.ail"` from the JSON's
   `.seed` field instead of executing `.properties[0].replay`. Every arm of it passed on a product
   that was broken. **(b)** Iteration 111: the public guide taught a function name absent from the
   example file while handing the reader copy-pasteable commands against that very file — filed by
   the judge as a maintenance nit, and worse than filed for exactly this reason.
   Concretely: **(i)** enumerate, per milestone, every string the product EMITS for a human to run
   or paste, and require one test that takes that string *out of the output* and executes it
   verbatim — split it, drop the binary name, pass the remaining tokens through; **(ii)** an
   assertion built from a field the same output also contains is not that test, however equal the
   two values happen to be today; **(iii)** the mutation that proves it is "make the emitter use
   the wrong source" — if only a test that re-derives the command exists, that mutant survives, and
   it is the survival you should be predicting before you run it; **(iv)** when the emitted form
   cannot be executed in-process (it names a network resource, a paid API, a destructive action),
   assert its *shape* against a parser rather than against a reconstruction, and say in the AC that
   execution was not possible. The tell: an acceptance criterion mentions a user-facing command,
   snippet or link, and every command in it was written by the test author rather than read out of
   the product's own output.
   **Corollary on the mutation drill this rule will make you run more often — RESTORE FROM A COPY,
   NEVER `git checkout -- <file>`** (one instance, iteration 166, first-party, and recorded as a
   correction to an existing prescription rather than as a rule earning its way in on evidence).
   Every mutation rule above ends "restore byte-identical, verified by sha256". None says how, and
   in a sprint worktree the file you are mutating is *uncommitted by construction* — so
   `git checkout --` restores it to HEAD and silently deletes the executor's work. The sha256 check
   then fires correctly, which is the good news and also the whole problem: it reports MISMATCH
   *after* the loss. Iteration 166 did this to `internal/testing/reporter.go` and recovered only
   because the diff was still in-session and the pre-mutation hash was known, so the reconstruction
   could be *proved* byte-identical rather than hoped to be. `cp <file> <backup>` before the
   mutation, `cp <backup> <file>` after, and keep the sha256 assertion as the check on the restore
   rather than as the discovery of a disaster.
3l. **"ENVIRONMENTAL" IS A CLAIM, AND THE FLEET IS ITS CONTROL GROUP — THREE MISSIONS RUN ON THIS
   RIG, SO ANY "IT'S THE MACHINE, NOT US" DIAGNOSIS HAS A READY-MADE THIRD ARM, AND SKIPPING IT
   COSTS MONTHS** (added 2026-08-15 motoko iteration 5; two frictions, both about the same defect).
   Rules 3a–3k police claims about the repo, a check, a probe, a mutation. None points at the loop's
   diagnosis of **its own health**, and that is where the most expensive wrong verdict this fleet has
   recorded actually lived. The failure mode is specific and seductive: **two missions failing
   together reads as evidence of an ENVIRONMENT when it may be evidence of a shared REPO.** It is
   rule 3d's shape — co-occurrence read as causation — but the co-occurrence is across *missions*
   rather than across commits, so nothing in 3d prompts you to look for it.
   Friction 1: motoko iteration 4 measured the driver's empty-output probe refusals in two logs,
   found v1 refusing with an identical signature in an overlapping window from *"a separate checkout
   with separate config"*, and recorded — reasonably, and wrongly — *"Not motoko-specific … what
   makes this environmental rather than per-mission"*. Friction 2: iteration 5 opened on GPU
   contention for the same reason, and the filler's `rig.lock` window fit three refusals before the
   *fourth* data point (a **successful** fire inside the same window) killed it.
   The third arm was free and sitting in `/tmp` the whole time. Refusals per fire over one 24-day
   window: v1 **47/186**, motoko **6/11**, world **0/89** — and world is the one mission whose
   checkout has **no `.claude/settings.json`**, hence no SessionStart hooks. The cause was a hook in
   the *shared repo* (a backgrounded child holding stdout past the probe's cap), which is why exactly
   the two `sunholo-data/ailang` checkouts were affected and the AILANG-source one never was. A
   two-mission sample cannot distinguish "the rig" from "the repo"; the three-mission one does it in
   a single `grep -c`.
   Rules: **(a)** before writing "environmental", "fleet-wide", "transient" or "not <mission>-specific"
   anywhere, count the symptom in **all three** driver logs — `/tmp/ailang-mission-{control,world,motoko}.log`
   — and quote rates, not presence: two missions failing is not a rate, and 47/186 vs 0/89 is;
   **(b)** pair the count with a known-positive control per log, because a mission whose log spells
   the symptom differently greps to a clean zero (world's zero is a measurement only because its log
   carries **90** `probe ok` lines — rule 3a aimed at a sibling's log rather than at your own repo);
   **(c)** when the arms differ, ask what the *failing* ones SHARE that the healthy one does not —
   repo, hooks, config, checkout path, verify profile — rather than what the environment was doing;
   the missions are deliberately configured differently, which is what makes them a usable control;
   **(d)** a driver's own summary line is not a diagnosis: this one flattened every failure into
   `quota-limited, timed out, or errored`, and **the quota arm had never fired once** in either log,
   so four months of refusals were read as quota pressure that was never present. Check which arm of
   a disjunctive log line actually fired before inheriting its framing.
   Mission-independent by construction, and it generalises past this fleet: **whenever you are about
   to blame a shared environment, find the peer that is NOT failing and ask what it lacks.**
3m. **A STRESS OR LOAD CONTROL ONLY CERTIFIES THE AXIS YOU VARIED — AND WHERE A BOUND AND ITS
   STIMULUS BOTH SCALE WITH THE MACHINE, THE BOUND MUST BE *DERIVED* FROM THE MEASURED STIMULUS**
   (added 2026-08-22 V1 iteration 248; proposed by `mission-world` iter-107 with two first-party
   instances ninety minutes apart on one test, and corroborated first-party in V1's own checkout
   before adoption — sibling-claim ghost discipline). Rule 3a(i) makes an *empty* result prove its
   instrument can see a positive. Rule 3b(ii) makes a `-run`/`--version` narrowing travel with a
   *green*. Rule 3b(ix) makes a scope travel with a *count*. Rule 3e pins the *base*. None of them
   points at a **stress control**, where the parameter you vary is chosen by you and is invisible in
   the output — so `N/N green under load` reads as *"the timing is sound"* when it means *"the
   timing is sound on the one knob I turned"*. Note the asymmetry that makes it durable: **more
   effort does not help**, because a larger sample of the same shape grows the N and not the
   coverage.
   World's instance 1, found by its own evaluator: a wall-clock arm re-run **15× unloaded and 8×
   under eight CPU spinners — 23/23 green**, hold ratio 26.7–30.2× against a 20× floor, with the
   loaded arm moving the ratio the *safe* direction. The judge varied **parallelism** instead:
   `GOMAXPROCS=1` → **10/10 FAIL on unmutated, sha256-identical code**, failing with
   `blocked read returned after 10–33ms` — **indistinguishable from the mutant signature the arm
   exists to detect**. Instance 2, found by CI after instance 1 was fixed: a **docs-only** record
   commit reddened `dev` in the same arm, because the stimulus scales with the machine (53 ms on the
   laptop, **2.63 s** on the runner — 49×) while the bounds were absolute millisecond constants
   calibrated on the laptop. Three axes, three reds — CPU contention, parallelism, absolute speed —
   and after each fix the *surviving constant still encoded one machine*. Enumerating axes is
   unbounded; deriving the bound is not. World's fix (`a87c723`): `readTimeout := hold / 20` makes
   the doc's "hold > 20× timeout" floor true **by construction** on any machine, the watchdog becomes
   the hold itself, and a `minDecoyHold` floor keeps a too-fast decoy a loud instrument failure
   rather than a silent pass — verified after at `GOMAXPROCS=1` under 16 spinners, 0 FAILs, and both
   mutations the arm owns **still die**, so scaling cost no kill.
   **V1's corroboration says the exposure is not World-specific and is large.** Measured at
   `404226a48` across `internal/` and `cmd/`: **51** `_test.go` files contain a hardcoded
   `N * time.Millisecond` literal used as a bound, against a control of **52** files mentioning
   `time.Millisecond` at all (negative control, a fresh absent literal: **0**; scopes asserted with
   `test -d`). And **ZERO** test files anywhere in those trees vary `GOMAXPROCS` — so the axis that
   produced World's 10/10 red is one this repo has never turned.
   **Rules. (a)** Before a timing or load result becomes evidence, name the axes you held FIXED
   (parallelism, CPU contention, memory pressure, page cache, disk, clock granularity, machine
   class) and vary the one the mechanism under test actually depends on — a scheduling race makes
   *parallelism* load-bearing and CPU contention decorative. **(b)** Where the bound and the stimulus
   both scale with the machine, **derive the bound from the stimulus measured in-test** rather than
   hardcoding wall-clock, so the ratio the design specifies holds by construction. **(c)** A floor on
   the *stimulus* is not a calibration: keep it absolute and loud, so a degenerate stimulus reports
   instrument failure instead of passing quietly. **(d)** Generalises past timing to any bound
   calibrated against an environment — buffer sizes, retry counts, memory ceilings, token budgets.
   Mission-independent. The tell: you are about to write "N/N green under load" and every run varied
   the same knob — or your test contains a millisecond literal you chose on the machine you are
   typing on.
3n. **YOUR MUTATION SET IS DERIVED FROM WHAT THE MILESTONE *FIXES*, SO IT SYSTEMATICALLY MISSES WHAT
   THE MILESTONE *SHIPS* — ANCHOR THE ENUMERATION TO THE DIFF, WHICH IS COMPLETE BY CONSTRUCTION**
   (added 2026-08-22 V1 iteration 250; instance 1 is iteration 249, instance 2 is this iteration, and
   in BOTH the gap was found by the judge rather than by the controller who wrote the mutants).
   Rules 3d, 3i and 3j all sharpen a mutation you have already decided to run. None of them asks how
   you CHOSE the set — and the choice is made, every time, by reading the defect: you mutate the thing
   the milestone was about, because that is the thing you have been thinking about all iteration. A
   diff ships more than that. Supporting predicates, shared helpers, a registry entry, a case added to
   a switch three files away — each is a line you are now responsible for and none of them appears in
   a mutation list derived from the bug. Note the asymmetry that makes this durable: the mutants you
   DO run all behave, so the drill reads as thorough precisely where it is narrowest.
   Two instances, both V1, consecutive. **249:** the controller ran two mutants on the milestone's
   deliverables, both sole killers, and the judge then found that the milestone's own M1 unit test
   asserted unconditional runtime-preamble boilerplate — it passed for a program containing no array
   at all. **250:** the controller ran two mutants, both sole killers, both aimed at the behaviour the
   milestone fixed; the judge reverted the two *supporting* edits and found that `types.go`'s
   `TArray` case reds **only its own unit test** (the whole golden + differential suite stays rc=0),
   while `IsUserDefinedType`'s `"ArrayVal"` case reds **nothing at all** — unit, golden and
   `verify-examples` all rc=0. Two shipped lines pinned by nothing, in a green sprint.
   **The cheap instrument already exists and is free.** `git diff` enumerates what you shipped,
   completely, by construction — which is exactly what rule 3a(i-e) asks for and what a
   defect-derived list can never be. And on 250 a **second, independent** instrument found the same
   two lines: SonarCloud's *new-code coverage* condition. That is worth knowing because it is
   already running on every PR — a coverage-on-new-code red is not a style nag, it is a machine
   telling you which shipped lines nothing exercises, which is the same question this rule asks.
   **Rules. (a)** Before recording a Gate-4 verdict, walk the diff **hunk by hunk** and ask, per hunk,
   *what would go red if I reverted just this?* Where you cannot name a test, revert it and find out —
   mutant LANDED (sha256) and BUILDS asserted first, as always. **(b)** A hunk with **no** killer is a
   finding, not a failure: it is either genuinely unreachable, in which case say so **in the code**,
   or it is unpinned, in which case it is a queue row. Do not quietly widen the sprint to fix it.
   **(c)** Report *sole killer* separately from *some killer*, and expect supporting hunks to have
   weaker coverage than the headline one — that gradient is the signal. **(d)** Read a
   coverage-on-new-code gate as evidence about this rule rather than as a threshold to satisfy, and
   **re-read WHICH condition failed rather than inheriting a previous iteration's framing** — V1
   iterations 247 and 249 both met a *duplication* red on this same suite and correctly named it
   benign, and iteration 250 nearly inherited that reading for a red that was in fact *coverage*, on
   the very lines its own drill had missed. **(e)** Mission-independent: under `ailang-code` the diff
   is still the enumeration and `ailang test` is still the killer. The tell: every mutant you ran was
   a sole killer, and you chose all of them by thinking about the bug.
   **AND A MUTANT THAT REDS *SOMETIMES* IS NEITHER A KILLER NOR A MISS — IT IS A REPORT THAT YOUR
   SYSTEM IS NONDETERMINISTIC, AND THE MOVE IT INVITES (ENLARGE THE SAMPLE UNTIL THE KILL LOOKS
   RELIABLE) IS THE ONE MOVE THAT CANNOT WORK** (added 2026-09-01 V1 iteration 314; instance 1 is
   rule 3m's World sequence, three rounds of adjusting a stress control's parameters where the fix
   was to DERIVE the bound; instance 2 is this iteration, which made the mistake in a code comment
   after the judge had already handed it the measurement). Every mutation rule above is binary: the
   mutant reds or it does not, and 3n(b) offers exactly two dispositions for a hunk — declared
   unreachable, or a queue row. Neither covers the middle case, and the middle case has a default
   that feels like diligence: the arm *does* kill, just not every run, so you make the fixture
   bigger. That treats variance as a sampling problem when it is a property of the code, and it
   leaves the nondeterminism shipped.
   Measured here. The judge found the `sort.Strings` mutant on a signature set killing ~**1 in 15**
   runs, because the set was built by ranging a map. The controller widened the fixture from two
   exports to six and wrote into a code comment that this bounded survival at `1/6! = 1/720`. Re-
   measured: **4 kills in 8 runs** — no better than two elements. The probability model was simply
   false (Go rotates single-bucket map iteration rather than permuting it), and it was a claim about
   a mechanism nobody had measured, sitting in a comment no later reader would ever re-derive. The
   real fix was to remove the nondeterminism — collect in encounter order behind a seen-set — after
   which the arm is **0 failures in 10 runs**, the sort's mutant reds **nothing**, and the sort is an
   honest declared residual with its three invariants written in the code. Note what the widening
   would have bought if the arithmetic had been right: a *quieter* flake, i.e. the same defect with
   a longer mean time to discovery.
   **Rules. (a)** An intermittent kill is a finding about the SYSTEM, not about the drill — find the
   nondeterminism (map iteration, goroutine scheduling, wall-clock, filesystem order, a random seed)
   and remove it, then re-classify the hunk as sole-killer or declared residual. **(b)** Never quote
   a probability you have not measured; a survival rate is an experiment (`for i in $(seq 1 N)`),
   and a closed-form bound is a claim about a mechanism, which is rule 3f's discipline aimed at your
   own reasoning. **(c)** Never write an unmeasured probability into code or a record — it is
   unfalsifiable in place and will be inherited as fact. **(d)** Enlarging a sample is legitimate
   only to MEASURE a rate, never to make a flaky assertion pass; if the honest outcome is "no
   killer", say so, which is what 3n(b) already asks. **(e)** Mission-independent, and it
   generalises past mutation to every flaky gate this loop meets: a re-run that goes green is
   iteration 153's environment-divergence control, and a re-run that goes green *often enough* is
   not a fix. The tell: you are about to make a fixture, timeout, or retry count bigger, and your
   justification is a number you calculated rather than one you ran.

4. **The shared main checkout is mutable mid-iteration** (added 2026-07-10 iteration 4, TWO
   frictions: a sibling agent opened a conflicted merge in the main tree mid-iteration, turning
   the Gate-2 rebuild `-dirty` — binaries built from a half-merged tree; and a persisted `cd`
   into a worktree made a later "main-tree" check read the WORKTREE's `.git` and report the
   merge cleared when it wasn't). Rules: (a) Bash cwd persists across calls — before trusting
   any main-tree git check, re-confirm `pwd` or use absolute paths; (b) re-run `git status` at
   the moment of use, not from memory — a clean tree at preflight proves nothing an hour later;
   (c) if `MERGE_HEAD` exists (a sibling's in-progress merge), do NOT commit in the main tree —
   your commit would complete THEIR merge; integrate via a worktree branch + PR with
   `gh pr merge --auto` instead (worked cleanly: PR #336); (d) a `-dirty` version suffix on a
   rebuilt binary means the tree changed under you — rebuild inside the isolated worktree.
