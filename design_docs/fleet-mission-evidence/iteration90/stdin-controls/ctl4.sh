#!/bin/bash
WT=/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter90-secrets-m3; O=/tmp/fleet_i90_pi_ctl4; rm -rf $O; mkdir -p $O
SCOPE='. "$WT/tools/launchd/lib/secrets-scope.sh"; export AILANG_SECRETS_LANE=pi-controller MISSION_DRIVER_ROOT="$WT"; unset BASH_ENV; mc_secrets_scope_apply pi-controller 2>/dev/null || exit 9;'
B='log(){ :; }; . "$WT/tools/launchd/lib/lane-probe.sh"; s=$(date +%s);'
A1="$B"' _mc_bounded 30 pi --mode json --no-session --no-tools --no-extensions --model ollama/glm-5.3:cloud -p "reply with exactly: ok" < /dev/null; echo "rc=$? secs=$(( $(date +%s) - s ))"'
A2='s=$(date +%s); dl=$(( s + 30 )); pi --mode json --no-session --no-tools --no-extensions --model ollama/glm-5.3:cloud -p "reply with exactly: ok" > /dev/null 2>&1 & p=$!; while kill -0 $p 2>/dev/null; do [ $(date +%s) -ge $dl ] && { kill $p; break; }; sleep 1; done; wait $p; echo "rc=$? secs=$(( $(date +%s) - s ))"'
A3="$B"' ls -l /proc/self/fd/0 2>/dev/null; _mc_bounded 30 /bin/bash -c "ls -l /dev/fd/0; env | cut -d= -f1 | sort | tr \"\\n\" \" \"" ; printf %s "$MC_BOUNDED_OUT"'
for a in A1 A2 A3; do eval "body=\$$a"
  ( cd /tmp; env -i HOME="$HOME" PATH="$PATH" USER="$USER" TMPDIR="${TMPDIR-/tmp}" WT="$WT" /bin/bash -c 'set -a; . "$HOME/.config/ailang/secrets.env"; set +a; unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN OPENAI_API_KEY; '"$SCOPE $body" 2>&1 ) > "$O/$a.out"
done; echo DONE > $O/done
