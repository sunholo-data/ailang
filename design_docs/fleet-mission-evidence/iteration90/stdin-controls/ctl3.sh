#!/bin/bash
WT=/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter90-secrets-m3; O=/tmp/fleet_i90_pi_ctl3; rm -rf $O; mkdir -p $O
SCOPE='. "$WT/tools/launchd/lib/secrets-scope.sh"; export AILANG_SECRETS_LANE=pi-controller MISSION_DRIVER_ROOT="$WT"; unset BASH_ENV; mc_secrets_scope_apply pi-controller 2>/dev/null || exit 9;'
BOUNDED='log(){ :; }; . "$WT/tools/launchd/lib/lane-probe.sh"; s=$(date +%s); _mc_bounded 40 pi --mode json --no-session --no-tools --no-extensions --model ollama/glm-5.3:cloud -p "reply with exactly: ok"; echo "rc=$? secs=$(( $(date +%s) - s ))"'
DIRECT='s=$(date +%s); dl=$(( s + 40 )); pi --mode json --no-session --no-tools --no-extensions --model ollama/glm-5.3:cloud -p "reply with exactly: ok" < /dev/null > /dev/null 2>&1 & p=$!; while kill -0 $p 2>/dev/null; do [ $(date +%s) -ge $dl ] && { kill $p; break; }; sleep 1; done; wait $p; echo "rc=$? secs=$(( $(date +%s) - s ))"'
for cwd in /tmp "$WT"; do for launcher in BOUNDED DIRECT; do
  eval "body=\$$launcher"; tag="$(basename $cwd)-$launcher"
  ( cd "$cwd"; env -i HOME="$HOME" PATH="$PATH" USER="$USER" TMPDIR="${TMPDIR-/tmp}" WT="$WT" /bin/bash -c 'set -a; . "$HOME/.config/ailang/secrets.env"; set +a; unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN OPENAI_API_KEY; '"$SCOPE $body" 2>&1 | grep -E '^rc=' ) > "$O/$tag.rc"
done; done
echo DONE > $O/done
