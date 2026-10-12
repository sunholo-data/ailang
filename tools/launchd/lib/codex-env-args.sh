# shellcheck shell=bash
# codex-env-args.sh — sourced by the mission driver (bash 3.2: no associative arrays, no ${v,,},
# no mapfile, no `. <(…)`). Pure functions; nothing runs at source time.
#
# Why: codex builds every shell tool call's environment from shell_environment_policy. The rig's
# ~/.codex/config.toml sets inherit = "core", so a codex CONTROLLER's shells lose every MISSION_*,
# AILANG_DRIVER_* pin and the HD-4 scope guard's GIT_CONFIG_* entry the driver exported:
# resolve-role-spawn.sh then fails closed for every role and the slot is lost (ticket
# agent-tool:mission-role-pins-unavailable). Per-variable `-c shell_environment_policy.set.NAME=…`
# adds exactly the named vars, merges with the rig's own .set table and works for any `inherit`.
# `inherit=all` is rejected: it would also hand the controller every API key in the driver env.

# mc_toml_basic_string VALUE → prints a TOML basic string ("…"), rc 0. A value that still holds a
# control character after \n \t \r are escaped prints nothing and returns 1 (caller skips loudly).
mc_toml_basic_string() {
  local v="$1" bs=$'\\'
  v=${v//"$bs"/"$bs$bs"}
  v=${v//\"/"$bs\""}
  v=${v//$'\n'/"${bs}n"}
  v=${v//$'\t'/"${bs}t"}
  v=${v//$'\r'/"${bs}r"}
  case "$v" in *[[:cntrl:]]*) return 1 ;; esac
  printf '"%s"' "$v"
}

# _mc_codex_env_add NAME VALUE — append one -c pair; rc 1 if the value cannot be encoded.
_mc_codex_env_add() {
  local enc
  enc=$(mc_toml_basic_string "$2") || return 1
  MC_CODEX_ENV_ARGS+=(-c "shell_environment_policy.set.$1=$enc")
}

# mc_codex_env_args → fills the GLOBAL array MC_CODEX_ENV_ARGS and writes ONE summary line to
# stderr (names only, never values). Returns 0 even when nothing is forwarded.
mc_codex_env_args() {
  MC_CODEX_ENV_ARGS=()
  local n fwd=0 denied="" skipped="" warn="" cnt i k hk="" hv="" have_hk=0 cfg
  while IFS= read -r n; do
    case "$n" in
      CODEX_HOME|AILANG_CODEX_RUNTIME|MISSION_*|AILANG_DRIVER_*|AILANG_STORAGE_MESSAGING|AILANG_MESSAGES_PROJECT|AILANG_MISSION_REGISTRY|AILANG_STATE_DIR|CONTROLLER_PROVIDER|CONTROLLER_ID) ;;
      *) continue ;;
    esac
    case "$n" in
      *KEY*|*TOKEN*|*SECRET*|*PASSWORD*|*PASSWD*|*CREDENTIAL*|*AUTH*|*COOKIE*|*PRIVATE*)
        denied="$denied $n"; continue ;;
    esac
    case "$n" in *[!A-Za-z0-9_]*) skipped="$skipped $n"; continue ;; esac
    if _mc_codex_env_add "$n" "${!n}"; then fwd=$((fwd + 1)); else skipped="$skipped $n"; fi
  done < <(compgen -e)

  # HD-4 scope guard: forward ONLY the core.hooksPath entry, renumbered to index 0. Other
  # GIT_CONFIG entries stay behind (a VALUE_n may be an http.extraHeader credential).
  cnt="${GIT_CONFIG_COUNT:-0}"
  case "$cnt" in ''|*[!0-9]*) cnt=0 ;; esac
  i=0
  while [ "$i" -lt "$cnt" ]; do
    eval "k=\${GIT_CONFIG_KEY_$i-}"
    if [ "$k" = "core.hooksPath" ]; then
      eval "hv=\${GIT_CONFIG_VALUE_$i-}"
      hk="$k"; have_hk=1
    fi
    i=$((i + 1))
  done
  if [ "$have_hk" = 1 ]; then
    if _mc_codex_env_add GIT_CONFIG_COUNT 1 \
      && _mc_codex_env_add GIT_CONFIG_KEY_0 "$hk" \
      && _mc_codex_env_add GIT_CONFIG_VALUE_0 "$hv"; then
      fwd=$((fwd + 3))
    else
      skipped="$skipped GIT_CONFIG_*"
    fi
  fi

  # include_only is applied AFTER .set, so it would strip everything forwarded. Warn, never rewrite.
  cfg="${CODEX_HOME:-$HOME/.codex}/config.toml"
  if [ -r "$cfg" ] && grep -q '^[[:space:]]*include_only' "$cfg" 2>/dev/null; then
    warn=" WARNING include_only-present: forwarded vars may be stripped"
  fi
  echo "codex-env: forwarded=$fwd denied=${denied# } skipped=${skipped# }$warn" >&2
  return 0
}
