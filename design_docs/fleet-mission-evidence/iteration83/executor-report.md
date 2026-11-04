Applied the exact four-file patch. D-FLEET-18 remains OPEN; no `tools/eval/` files changed. All six mutations were reverted to the intended text.

`git status --short` confirms exactly four files. `git diff --stat` shows three tracked files, 13 insertions; the untracked fragment adds 3 lines, shown separately with `git diff --no-index`. No git write commands were issued directly; the requested suite performed its temporary fixture operations.

| Gate | Base rc | Patched rc | Final restored rc | Label |
|---|---:|---:|---:|---|
| `bash -n tools/launchd/githooks/pre-push` | 0 | 0 | 0 | PASS |
| `bash -n tools/launchd/test_mission_scope_guard.sh` | 0 | 0 | 0 | PASS |
| `/bin/bash tools/launchd/test_mission_scope_guard.sh` | 0 | 0 | 0 | **UNINFORMATIVE UNDER SANDBOX** |
| `scripts/mission_decisions.sh --check --file design_docs/fleet-mission.md` | 0 | 0 | 0 | PASS — 40 rows |
| `git diff --check` | 0 | 0 | 0 | PASS |

All mutation suite runs are labeled **UNINFORMATIVE UNDER SANDBOX**. Each returned nonzero at the named verdict check, before temporary fixture writes.

| Mutation | Suite rc | Failing arm |
|---|---:|---|
| M1: drop path 1 | 1 | `fleet tools/eval/motoko_connection_probe.sh`: got refuse, want allow |
| M2: drop path 2 | 1 | `fleet tools/eval/test_motoko_connection_probe.sh`: got refuse, want allow |
| M3: widen to `tools/eval/*` | 1 | `fleet tools/eval/qwen38-motoko-rerun.sh`: got allow, want refuse |
| M4: move pair into harness | 1 | `motoko tools/eval/motoko_connection_probe.sh`: got refuse, want allow |
| M5: path 1 suffix glob | 1 | `fleet tools/eval/motoko_connection_probe.sh.bak`: got allow, want refuse |
| M6: path 1 basename-prefix glob | 1 | `fleet tools/eval/xmotoko_connection_probe.sh`: got allow, want refuse |

Full final diff, including the new fragment’s complete content:

```diff
diff --git a/design_docs/fleet-mission.md b/design_docs/fleet-mission.md
index c38e1a066..d27498476 100644
--- a/design_docs/fleet-mission.md
+++ b/design_docs/fleet-mission.md
@@ -84,6 +84,7 @@ Clause map: **1** UNMEASURED, goal unmoved; **2** UNMET (nothing resolved); **3*
   `scripts/test_mission_*` (D-FLEET-5), its own
   `design_docs/fleet-mission*`), plus support paths (`design_docs/**`, `changelogs/**`,
   `CHANGELOG.md`, `docs/**`, `.ailang/state/sprints/**`, `internal/config/mission.go`, `make/test.mk`,
+  `tools/eval/motoko_connection_probe.sh`, `tools/eval/test_motoko_connection_probe.sh` (D-FLEET-18),
   `tools/pi-extensions/sandbox/**` (D-FLEET-6)).
   **Anything else is refused**, including CI workflows, `go.mod`, the server, the UI and non-mission
   commands. A fix that genuinely needs one of those parks for Mark.
diff --git a/tools/launchd/githooks/pre-push b/tools/launchd/githooks/pre-push
index 3c086ea44..806d056fe 100755
--- a/tools/launchd/githooks/pre-push
+++ b/tools/launchd/githooks/pre-push
@@ -30,6 +30,7 @@ _scope_is_fleet_support() {
   case "$1" in
     design_docs/*|changelogs/*|CHANGELOG.md|docs/*|.ailang/state/sprints/*) return 0 ;;
     internal/config/mission.go|make/test.mk) return 0 ;;
+    tools/eval/motoko_connection_probe.sh|tools/eval/test_motoko_connection_probe.sh) return 0 ;;   # support, not harness (D-FLEET-18)
     tools/pi-extensions/sandbox/*) return 0 ;;   # the pi sandbox fail-closed fix (D-FLEET-6); the eval harness shares the rest of tools/pi-extensions
   esac
   return 1
diff --git a/tools/launchd/test_mission_scope_guard.sh b/tools/launchd/test_mission_scope_guard.sh
index e8685ebc8..9364eb1d4 100755
--- a/tools/launchd/test_mission_scope_guard.sh
+++ b/tools/launchd/test_mission_scope_guard.sh
@@ -50,6 +50,17 @@ check v1 scripts/test_mission_pi_run.sh refuse
 check fleet tools/pi-extensions/sandbox/index.ts allow
 check fleet tools/pi-extensions/ailang-exec/index.ts refuse
 check v1 tools/pi-extensions/sandbox/index.ts allow
+# D-FLEET-18: exact support paths; product loops retain access.
+check fleet tools/eval/motoko_connection_probe.sh allow # kills: drop path 1
+check fleet tools/eval/test_motoko_connection_probe.sh allow # kills: drop path 2
+check motoko tools/eval/motoko_connection_probe.sh allow # kills: move path 1 to _scope_is_harness
+check motoko tools/eval/test_motoko_connection_probe.sh allow # kills: move path 2 to _scope_is_harness
+check v1 tools/eval/motoko_connection_probe.sh allow # kills: move path 1 to _scope_is_harness
+check v1 tools/eval/test_motoko_connection_probe.sh allow # kills: move path 2 to _scope_is_harness
+check fleet tools/eval/qwen38-motoko-rerun.sh refuse # kills: widen to tools/eval/*
+check fleet tools/eval/motoko_connection_probe.sh.bak refuse # kills: path 1 suffix glob
+check fleet tools/eval/test_motoko_connection_probe.sh-x refuse # kills: path 2 suffix glob
+check fleet tools/eval/xmotoko_connection_probe.sh refuse # kills: path 1 basename-prefix glob
 echo 'PASS scope verdicts: product loops refused on harness paths (incl. the fleet charter), fleet held to its allowlist, attended unaffected'
 
 # End to end: git must actually run the hook when core.hooksPath comes ONLY from env.
diff --git a/changelogs/unreleased/2026-11-04-fleet-scope-arm-dfleet18.md b/changelogs/unreleased/2026-11-04-fleet-scope-arm-dfleet18.md
new file mode 100644
index 000000000..d125ad061
--- /dev/null
+++ b/changelogs/unreleased/2026-11-04-fleet-scope-arm-dfleet18.md
@@ -0,0 +1,3 @@
+### Changed — Fleet scope guard: two Motoko probe support paths (PROPOSED, D-FLEET-18 A)
+
+PROPOSED, pending Mark's ruling on D-FLEET-18. The fleet support allowlist admits exactly `tools/eval/motoko_connection_probe.sh` and `tools/eval/test_motoko_connection_probe.sh`; product missions retain access, and other `tools/eval/` paths remain refused. Guard arm only, with Authority documentation and regression tests. No probe changes or flake fixes are included; those follow only after the ruling and this arm lands on the pinned `origin/dev` copy (iteration-16 rule). See `design_docs/fleet-mission.md`, D-FLEET-18.
```