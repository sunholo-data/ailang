#!/bin/bash
WT=/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter90-secrets-m3; O=/tmp/fleet_i90_pi_ctl; rm -rf $O; mkdir -p $O
probe='deadline=$(( $(date +%s) + 150 )); start=$(date +%s); pi --mode json --no-session --no-tools --no-extensions --model ollama/glm-5.3:cloud -p "reply with exactly: ok" < /dev/null > "$OUTF" 2>&1 & p=$!; while kill -0 $p 2>/dev/null; do [ $(date +%s) -ge $deadline ] && { kill $p; echo "cap" >> "$OUTF"; break; }; sleep 2; done; wait $p; rc=$?; echo "rc=$rc secs=$(( $(date +%s) - start ))"'
for arm in unscoped scoped unscoped2; do
  if [ "$arm" = scoped ]; then pre='. "$WT/tools/launchd/lib/secrets-scope.sh"; export AILANG_SECRETS_LANE=pi-controller MISSION_DRIVER_ROOT="$WT"; unset BASH_ENV; mc_secrets_scope_apply pi-controller 2>/dev/null || { echo scope-failed; exit 9; };'; else pre=''; fi
  env -i HOME="$HOME" PATH="$PATH" USER="$USER" TMPDIR="${TMPDIR-/tmp}" WT="$WT" OUTF="$O/$arm.out" /bin/bash -c 'set -a; . "$HOME/.config/ailang/secrets.env"; set +a; unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN OPENAI_API_KEY; '"$pre $probe" > "$O/$arm.rc" 2>&1
done
echo "DONE pi control for b216285db" > $O/done
