# shellcheck shell=bash
# Mission credentials are independent of the operator's interactive login.
# Sourcing defines functions only; callers select the profile explicitly.
mc_select_codex_profile() {
  local selected="${CODEX_HOME:-$HOME/.codex-missions}" interactive="${CODEX_INTERACTIVE_HOME:-$HOME/.codex}" canonical parent
  case "$selected" in /*) ;; *) echo 'codex profile: CODEX_HOME must be absolute' >&2; return 1 ;; esac
  [ ! -L "$selected" ] || { echo 'codex profile: symlink home refused' >&2; return 1; }
  if [ -d "$selected" ]; then canonical=$(cd "$selected" && pwd -P) || return 1
  else
    parent=$(dirname "$selected")
    [ -d "$parent" ] || { echo 'codex profile: prepare the mission home first' >&2; return 1; }
    canonical="$(cd "$parent" && pwd -P)/$(basename "$selected")"
  fi
  if [ -d "$interactive" ]; then interactive=$(cd "$interactive" && pwd -P) || return 1; fi
  case "$canonical/" in "$interactive/"*) echo 'codex profile: interactive home refused' >&2; return 1 ;; esac
  case "$interactive/" in "$canonical/"*) echo 'codex profile: overlapping home refused' >&2; return 1 ;; esac
  case "${AILANG_CODEX_RUNTIME:-daemon}" in daemon) ;; *) echo 'codex profile: missions require the managed daemon' >&2; return 1 ;; esac
  export CODEX_HOME="$canonical" AILANG_CODEX_RUNTIME=daemon
}

mc_codex_command() {
  case "${AILANG_CODEX_RUNTIME:-}" in
    daemon) MC_CODEX_COMMAND=(ailang mission codex-exec) ;;
    "") MC_CODEX_COMMAND=(codex exec) ;; # legacy callers outside an initialized mission driver
    *) echo 'codex profile: unsupported runtime; no exec fallback' >&2; return 1 ;;
  esac
}

mc_codex_exec() {
  mc_codex_command || return
  "${MC_CODEX_COMMAND[@]}" "$@"
}
