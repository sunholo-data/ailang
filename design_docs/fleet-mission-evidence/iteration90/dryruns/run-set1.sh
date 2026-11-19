#!/bin/bash
WT=/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter90-secrets-m3; cd "$WT" || exit 2
O=/tmp/fleet_i90_dry; rm -rf "$O"; mkdir -p "$O"
S=$HOME/.config/ailang/secrets.env
pre="$(shasum -a 256 < "$S") $(wc -l < "$S")"
ls -lT ~/.ailang/state | awk '{print $6,$7,$8,$9,$10}' | sort > "$O/state_before.txt"
patched_sha=$(git rev-parse HEAD); dry_profile=triage
for controller_pin in claude:claude-opus-5-5 codex:gpt-6.1-sol pi:ollama/glm-5.3:cloud; do
  provider=${controller_pin%%:*}
  [ -f ~/.ailang/state/mission-triage.pid ] && { echo "triage pidfile appeared — abort" > "$O/ABORT"; break; }
  env -i HOME="$HOME" PATH="$PATH" USER="$USER" TMPDIR="${TMPDIR-/tmp}" \
    AILANG_WORKTREE_SWEEP=0 AILANG_DRIVER_PINNED="$patched_sha" \
    MISSION_PROFILE="$dry_profile" MISSION_DRY_RUN=1 MISSION_MODEL="$controller_pin" \
    /bin/bash tools/launchd/mission-control.sh > "$O/${provider}-healthy.log" 2>&1
  echo "$provider healthy_rc=$?" >> "$O/rc.txt"
  env -i HOME="$HOME" PATH="$PATH" USER="$USER" TMPDIR="${TMPDIR-/tmp}" \
    AILANG_WORKTREE_SWEEP=0 AILANG_DRIVER_PINNED="$patched_sha" \
    MISSION_PROFILE="$dry_profile" MISSION_DRY_RUN=1 MISSION_MODEL="$controller_pin" \
    MISSION_EXECUTOR_MODEL=codex:secrets-scope-nonexistent-model MISSION_PROBE_TIMEOUT=10 \
    /bin/bash tools/launchd/mission-control.sh > "$O/${provider}-degraded.log" 2>&1
  echo "$provider degraded_rc=$?" >> "$O/rc.txt"
done
post="$(shasum -a 256 < "$S") $(wc -l < "$S")"; [ "$pre" = "$post" ] && echo "secrets.env SAME" >> "$O/rc.txt" || echo "secrets.env CHANGED" >> "$O/rc.txt"
ls -lT ~/.ailang/state | awk '{print $6,$7,$8,$9,$10}' | sort > "$O/state_after.txt"
echo "DONE for $patched_sha" > "$O/done"
