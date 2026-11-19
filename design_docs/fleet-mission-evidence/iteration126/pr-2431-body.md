Fleet mission iteration 126. Ticket `driver:idle-fingerprint-head-ignores-design-docs-verification` (filed by `world`, 2 slots lost).

**Defect.** `tools/launchd/lib/idle-fingerprint.sh` excludes a mission's per-iteration records from `head=` only under the repo-root `verification/<name>-iter*`. World has written them under `design_docs/verification/<name>-iter*` since about iteration 358, so every record-only commit moved `head=` and the idle pre-check ran a full fire on unchanged inputs (World iter 414 `4299cdb6` → iter 415).

**Fix.** Add `:(exclude)design_docs/verification/${name}-iter*` and keep the root exclusion. Non-iter `design_docs/verification/` dirs (`w-*`, `world-attended-*`) still wake the loop. A negative control pins that.

**Tests.** A new record-only arm, a non-iter negative control, and three single-line mutants: `exclude-root-verification`, `exclude-dd-verification` and `exclude-dd-broad`, each with a sole killer. `--mutations` kills 16/16. `make test-launchd-drivers` rc 0, shellcheck rc 0, `make check-changelog` rc 0.

**Real repro (world clone, read-only).** The old pathspecs give `a15551c2 record: iteration 415`. The new ones give `0e9e8d35 test(row 243 …)`, a product commit.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
