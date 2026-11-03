1. **Scope and disposition.** Plan against clean HEAD `b23cadced75c7bbf1328eebcc5ab011453fba672`. Prepare a **draft PR**, pending D-FLEET-25; do not merge or resolve the decision. Change comments/documentation and one changelog fragment only. No ledger mechanism, new doctor check, or TOML key changes.

2. **Doctor finding.** World does **not** receive `driver-fork` from the HEAD registry: [doctor.go:281](internal/mission/doctor.go:281) requires `m.Driver != ""`, and `missions/world.toml` has no `driver` assignment. [registry.go:66](internal/mission/registry.go:66) describes that conditional override, followed by past-tense history; it does not classify World as a fork. **Neither production Go file needs changing.** [golden_live_test.go:367](internal/mission/golden_live_test.go:367) records de-forking on 2026-09-06 and lines 379–383 reject a fork finding. `doctor_test.go:187–215` deliberately constructs an explicit fork fixture.
   
   Audit covered the requested allowlist, including both hidden skill trees and charter bodies. Additional current claims are listed below. Preserve dated history: V1’s August narrative (`v1-mission.md:3609–3613`), fleet’s dated iteration-79 narrative (`fleet-mission.md:175`), skill war stories, logs, archives, and implemented designs. Other allowed CLI/driver/docs hits are generic conditions, repository identities, or historical explanations.

3. **`missions/world.toml:3–8`**, exact replacement. Retain the existing title and blank comment line, matching the short descriptive headers in `stapledon.toml` and `motoko.toml`.

   Before:
   ```text
   # ⚠ WORLD IS A FORK. Its driver lives in a SEPARATE GitHub repo
   # (sunholo-data/ailang-world) with its own copy of mission-control.sh and no
   # lib/pin-root.sh, so it never re-execs from the driver pin and has silently missed
   # every routing fix landed in sunholo-data/ailang. `ailang mission doctor` reports this
   # as a note on every run. De-forking is Phase 3 of M-MISSION-LOOP-WORKBENCH and is
   # gated on HD-2, which is deliberately unratified.
   ```
   After:
   ```text
   # Works in sunholo-data/ailang-world and runs the shared driver from
   # sunholo-data/ailang, re-executed from the driver pin.
   # De-forked 2026-09-06; rechecked 2026-11-03. No driver override is declared,
   # so `ailang mission doctor` does not report a driver-fork note.
   ```

4. **Both `.claude/skills/mission-loop-change/SKILL.md` and `.agents/skills/mission-loop-change/SKILL.md`**, identical original line numbers. These contain current operational claims, not dated war stories. Apply these exact substring/block replacements to both:
   
   - **:3:** `porting a fix to the world fork` → `checking the shared driver's reach to World`.
   - **:27:** replace the complete row with `| Driver changes for ` + `` `world` `` + ` | the shared driver in the **ailang repo** | driver pin; World de-forked 2026-09-06, rechecked 2026-11-03 — see Gate 2 |`.
   - **:43–77**, replace this entire block:

   ```text
   A mission runs the driver its plist's `ProgramArguments` points at. Three of four re-exec from
   the driver pin (committed `origin/dev` of *their own clone's* repo); world does not.

   ```bash
   for m in v1:ailang docs:ailang-docs motoko:ailang-motoko world:ailang-world; do
     n=${m%%:*}; P=~/dev/sunholo-data/${m##*:}/tools/launchd/mission-control.sh
     printf "%-7s re-execs-from-pin=%s\n" "$n" \
       "$(grep -c '^\s*\. "\$REPO/tools/launchd/lib/pin-root.sh"' $P 2>/dev/null | sed 's/^0$/NO/;s/^[1-9].*/YES/')"
   done
   ```

   - `YES` → the clone's own contents are irrelevant; it runs committed `origin/dev`. **Push or it
     does not ship.** A "N behind" line about the source clone is expected and harmless.
   - `NO` (world) → it runs the working tree directly. Committing is not enough and not required;
     the file on disk is what executes.

   ## Gate 2 — World is a fork

   `world` lives in a separate GitHub repo with its own ~1000-line copy of the driver and no
   `pin-root.sh`. It has silently missed every routing fix landed in `sunholo-data/ailang`.

   - Any driver change you make for the fleet **must be made twice** until the de-fork lands.
   - Port **verbatim**, then prove it:

   ```bash
   for fn in _mc_set_controller select_model; do
     diff <(sed -n "/^${fn}()/,/^}/p" ~/dev/sunholo-data/ailang/tools/launchd/mission-control.sh) \
          <(sed -n "/^${fn}()/,/^}/p" ~/dev/sunholo-data/ailang-world/tools/launchd/mission-control.sh) \
       >/dev/null && echo "$fn: identical" || echo "$fn: DIFFERS"
   done
   ```

   Byte-identical is what lets you claim the upstream suite covers the port. Anything else and you
   owe world its own tests.
   ```
   After:
   ```text
   A mission runs the driver its plist's `ProgramArguments` points at. Check the computed
   reach with `ailang mission list` and `ailang mission doctor`; inspect the installed plist
   when verifying deployment. Shared-driver changes ship through the committed driver pin.

   ## Gate 2 — World uses the shared driver (rechecked 2026-11-03)

   World works in `sunholo-data/ailang-world` and uses the shared AILANG driver.
   `internal/mission/golden_live_test.go:367` records de-forking on 2026-09-06.
   `missions/world.toml` declares no `driver` override; `doctor.go:281` therefore marks no fork.
   Shared-driver changes require no separate World port.
   ```
   - **:81–82:** `World executes its driver straight from the working tree, and **bash reads a script` / `incrementally** — truncating it under a live interpreter feeds it garbage mid-iteration.` → `For any driver executing from a working tree, **bash reads a script incrementally** —` / `truncating it under a live interpreter feeds it garbage mid-iteration. World uses the pin (rechecked 2026-11-03).`
   - **:148–149:** `The suites are grep-based structural assertions over `$HERE/mission-control.sh`, so **copying a` / `suite into another repo tests that repo's driver** — that is how world gets coverage today.` → `The suites include structural assertions over `$HERE/mission-control.sh`.` / `World uses the shared driver and its upstream suites (rechecked 2026-11-03).`
   - **:236:** `; ported to world if fleet-wide (Gate 2)` → `; World uses the shared driver (Gate 2, rechecked 2026-11-03)`.

5. **Remaining exact current-text substitutions**; `/` between quoted lines below denotes an existing newline.
   
   - **`scripts/mission_directives.sh:34`:** `#     read: exclusive (>), no ids, output unchanged since before #981 (the world fork still uses it)` → `#     read: exclusive (>), no ids, output unchanged since before #981 (compatibility retained; rechecked 2026-11-03)`.
   - **Same file :115:** `# watermark (the world fork's Gate 0 runs this script by absolute path with --since) rely on` → `# watermark rely on` followed by `# this compatibility contract (rechecked 2026-11-03), including`.
   - **`internal/mission/driver_agreement_test.go:177`:** `// while world's fork is still on disk (measured: it is).` → `// if a separate driver exists outside this checkout (rechecked 2026-11-03).` Preserve indentation.
   - **`internal/mission/doctor_test.go:191`:** `// unpinned, exactly like the real world mission` → `// deliberately unpinned fixture; real World uses the pin (rechecked 2026-11-03)`.
   - **`design_docs/fleet-mission.md:174`:** replace the complete sentence beginning `One factual sub-item is still wrong:` and ending with the installed `mission-control.sh` path with: `Rechecked 2026-11-03: World was de-forked on 2026-09-06 (golden_live_test.go:367); world.toml declares no driver override, so doctor.go:281 reports no fork. Current documentation still contains stale fork claims; the mechanical correction is proposed separately, pending this ruling.` Preserve A/B/C, OPEN, and the default.
   - **Same charter :289:** `` `missions/world.toml` still calls World a fork. `` → `A mechanical correction to the stale World fork descriptions is proposed separately (2026-11-03); D-FLEET-25 stays OPEN.`

6. **Comment parsing and existing checks.** No test, golden file, or `make check-*` gate found parses or asserts `world.toml:3–8`. `registry.go:238–246` uses TOML decoding. Relevant consumers are `internal/mission/registry_test.go`, `golden_live_test.go`, `render_test.go`, and `tools/launchd/test_mission_routing.sh:800–819` (assignment/section checks). `driver_agreement_test.go` checks driver source/copies, not this header. Skill parity is checked by `tools/launchd/test_agents_skills_sync.sh` and `sync-agents-skills.sh --check`.
   
   Zero-result searches for the old header literals across tests/goldens were paired with the positive `WORLD IS DE-FORKED` hit at `golden_live_test.go:367`; the no-`driver` search was paired with `repo` at `world.toml:10`. The zero check-script search was paired with `changelog_fold.sh --check` at `scripts/check_changelog.sh:105`.

7. **`changelogs/unreleased/2026-11-03-world-shared-driver-comments.md`**, exact content, following the fragment README:
   
   ```markdown
   ### Fixed — stale World mission fork descriptions (2026-11-03)

   Correct World mission comments and operational guidance to describe the shared, pinned AILANG driver, following its 2026-09-06 de-fork. The doctor already classifies World correctly; no runtime behavior or TOML keys change. This is the mechanical sub-item of [D-FLEET-25](../../design_docs/fleet-mission.md), whose policy decision remains open.
   ```

8. **Gates at both base and proposed head, narrowest first.** Record baseline results separately; old-text matches at base and new-text absence are expected measurements.
   
   ```bash
   git diff --check
   /bin/bash -n scripts/mission_directives.sh
   /bin/bash tools/launchd/sync-agents-skills.sh --check
   go test ./internal/mission/...
   /bin/bash tools/launchd/test_agents_skills_sync.sh
   make check-skills
   make check-context-docs
   make check-changelog
   ```
   Audit commands at both revisions:
   ```bash
   scope=(missions internal/mission cmd/ailang/mission* tools/launchd .claude/skills/mission-* .agents/skills/mission-* scripts/mission_* docs design_docs/*mission.md design_docs/mission-charter-TEMPLATE.md)
   rg -n -i 'fork|ailang-world' "${scope[@]}"
   rg -n 'WORLD IS DE-FORKED' internal/mission/golden_live_test.go
   rg -n 'WORLD IS A FORK|World is a fork|world runs a FORK|world fork still|world fork.s Gate|world.s fork is still|porting a fix to the world fork|ported to world if fleet-wide|world does not[.]|that is how world gets coverage today' "${scope[@]}"
   rg -n 'De-forked 2026-09-06; rechecked 2026-11-03' missions/world.toml
   ```
   At head require zero residual matches from the targeted third command, paired with the positive second/fourth commands. Review the broad first command against the history exclusions in step 2. Confirm unchanged TOML assignments with `git diff -- missions/world.toml`, and inspect `git diff --stat` for scope.
   
   Planner ran no write-requiring test gates. Any execution requiring sockets or writes outside this tree is **UNINFORMATIVE UNDER SANDBOX**, never pass/fail; live golden skips do not prove installed deployment.

9. **Mechanical assessment and risks.** Yes: factual comments, existing guidance, and changelog only; no routing, lane order, quota, ration threshold, billing guard, assertions, or configuration values change. Risks are accidentally rewriting dated evidence, drifting the skill mirror, or implying a fresh live deployment measurement. Keep historical literals, preserve all guards, and hold the PR as draft until Mark rules.