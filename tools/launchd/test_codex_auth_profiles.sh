#!/bin/bash
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
T=$(mktemp -d); trap 'rm -rf "$T"' EXIT
mkdir -p "$T/home/.codex" "$T/bin"
cat > "$T/bin/ailang" <<'EOF'
#!/bin/bash
printf '%s\n' "$@" > "$ARGV_OUT"
printf '%s\n' "$CODEX_HOME:$AILANG_CODEX_RUNTIME" > "$ENV_OUT"
EOF
chmod +x "$T/bin/ailang"
export ARGV_OUT="$T/argv" ENV_OUT="$T/env" PATH="$T/bin:$PATH"
export HOME="$T/home"
unset CODEX_HOME AILANG_CODEX_RUNTIME
. "$HERE/lib/codex-auth-profile.sh"
mc_select_codex_profile
[[ $CODEX_HOME == "$HOME/.codex-missions" && $AILANG_CODEX_RUNTIME == daemon ]]
mc_codex_exec --model m --json prompt
[[ $(head -2 "$T/argv" | tr '\n' ' ') == 'mission codex-exec ' ]]
[[ $(cat "$T/env") == "$HOME/.codex-missions:daemon" ]]
export CODEX_HOME="$HOME/custom"
mc_select_codex_profile
[[ $CODEX_HOME == "$HOME/custom" ]]
if ( export CODEX_HOME="$HOME/.codex"; mc_select_codex_profile ); then echo 'FAIL shared interactive home';exit 1;fi
if ( export CODEX_HOME=relative; mc_select_codex_profile ); then echo 'FAIL relative home';exit 1;fi
mkdir -p "$HOME/.codex-missions"
ln -s "$HOME/.codex" "$HOME/alias"
if ( export CODEX_HOME="$HOME/alias"; mc_select_codex_profile ); then echo 'FAIL aliased home';exit 1;fi
if ( export AILANG_CODEX_RUNTIME=cli; mc_select_codex_profile ); then echo 'FAIL competing runtime';exit 1;fi
if ( export AILANG_CODEX_RUNTIME=cli; mc_codex_exec --model m prompt ); then echo 'FAIL silent fallback';exit 1;fi
# Wiring assertions cover probes and controllers; the existing env-forward suite
# exercises the controller's actual extracted block with a daemon stub.
grep -q 'mc_codex_exec.*MC_CODEX_ENV_ARGS' "$HERE/mission-control.sh"
grep -q '_mc_bounded.*MC_CODEX_COMMAND' "$HERE/lib/lane-probe.sh"
grep -q 'CODEX_HOME|AILANG_CODEX_RUNTIME' "$HERE/lib/codex-env-args.sh"
# Exercise the real bounded probe, including graceful cancellation. Wiring
# assertions alone cannot catch Bash exec refusing a shell function.
export CODEX_HOME="$HOME/custom" AILANG_CODEX_RUNTIME=daemon
. "$HERE/lib/lane-probe.sh"
log() { :; }
_mc_is_over_ration() { return 1; }
_mc_probe_codex test-model
[[ $(head -2 "$T/argv" | tr '\n' ' ') == 'mission codex-exec ' ]]
cat > "$T/bin/ailang" <<'EOF'
#!/bin/bash
trap 'sleep 3; echo interrupted > "$TERM_OUT"; exit 0' TERM
while :; do sleep 1; done
EOF
export TERM_OUT="$T/terminated" PROBE_TIMEOUT=1
set +e
_mc_probe_codex test-model
rc=$?
set -e
[[ $rc == 124 && $(cat "$TERM_OUT") == interrupted ]]
echo 'PASS mission OAuth home, daemon routing, and fail-loud isolation'
