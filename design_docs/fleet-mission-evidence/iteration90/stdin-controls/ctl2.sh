#!/bin/bash
WT=/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter90-secrets-m3; O=/tmp/fleet_i90_pi_ctl2; rm -rf $O; mkdir -p $O
for arm in bounded-unscoped bounded-scoped; do
  if [ "$arm" = bounded-scoped ]; then pre='. "$WT/tools/launchd/lib/secrets-scope.sh"; export AILANG_SECRETS_LANE=pi-controller MISSION_DRIVER_ROOT="$WT"; unset BASH_ENV; mc_secrets_scope_apply pi-controller 2>/dev/null || exit 9;'; else pre=''; fi
  ( cd "$WT"; env -i HOME="$HOME" PATH="$PATH" USER="$USER" TMPDIR="${TMPDIR-/tmp}" WT="$WT" /bin/bash -c 'set -a; . "$HOME/.config/ailang/secrets.env"; set +a; unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN OPENAI_API_KEY; log(){ :; }; . "$WT/tools/launchd/lib/lane-probe.sh"; '"$pre"' s=$(date +%s); _mc_bounded 150 pi --mode json --no-session --no-tools --no-extensions --model ollama/glm-5.3:cloud -p "reply with exactly: ok"; rc=$?; echo "rc=$rc secs=$(( $(date +%s) - s )) outbytes=$(printf %s "$MC_BOUNDED_OUT" | wc -c)"; printf %s "$MC_BOUNDED_OUT" | tail -c 400 > "'$O'/'$arm'.tail"' ) > $O/$arm.rc 2>&1
done
echo "DONE ctl2" > $O/done
