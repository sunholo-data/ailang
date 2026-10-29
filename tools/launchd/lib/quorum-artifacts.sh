# shellcheck shell=bash
# Bash 3.2 and zsh functions. Nothing runs at source time. rc 1 is absence;
# rc 2 is a configuration/read error, never permission to skip quorum.
# Skill snippets source this into the controller's zsh tool shell, where NOMATCH aborts unmatched loops.
mc_quorum_error() { printf 'quorum ERROR: %s\n' "$*" >&2; return 2; }

mc_quorum_absolute() {
  case "$1" in /*) return 0 ;; esac
  mc_quorum_error 'artifact directory must be absolute and nonempty'
}

mc_quorum_repo_valid() {
  local owner repo
  case "$1" in */*) owner=${1%%/*}; repo=${1#*/} ;; *) return 2 ;; esac
  case "$owner" in ''|.|..|*[!a-zA-Z0-9_.-]*) return 2 ;; esac
  case "$repo" in ''|.|..|*[!a-zA-Z0-9_.-]*) return 2 ;; esac
}

mc_quorum_init() {
  local repo
  if [ -n "${AILANG_AGENT_POLICY:-}" ]; then
    mc_quorum_error "policy marker present; local confinement: $(pwd -P)/.ailang/state/mission-quorum"
    return 2
  fi
  if [ -n "${MISSION_QUORUM_ARTIFACT_DIR:-}" ]; then
    mc_quorum_absolute "$MISSION_QUORUM_ARTIFACT_DIR" || return 2
  else
    repo=${MISSION_REPO:-}
    mc_quorum_repo_valid "$repo" || { mc_quorum_error 'invalid owner/repo namespace'; return 2; }
    mc_quorum_absolute "${HOME:-}" || return 2
    case "$repo" in
      sunholo-data/ailang) MISSION_QUORUM_ARTIFACT_DIR="$HOME/.ailang/state/mission-quorum" ;;
      sunholo-data/ailang-world) MISSION_QUORUM_ARTIFACT_DIR="$HOME/.ailang/state/mission-quorum-world" ;;
      *) MISSION_QUORUM_ARTIFACT_DIR="$HOME/.ailang/state/mission-quorum-repos/$repo" ;;
    esac
  fi
  export MISSION_QUORUM_ARTIFACT_DIR
}

# Fill a quoted indexed array. Marked standalone readers stay in sandbox CWD.
mc_quorum_reader_dirs() {
  local root="$1"
  mc_quorum_absolute "$root" || return 2
  MC_QUORUM_DIRS=()
  if [ -n "${AILANG_AGENT_POLICY:-}" ]; then
    if [ -n "${MISSION_QUORUM_ARTIFACT_DIR:-}" ]; then
      printf 'quorum artifact override suppressed: policy marker present; effective local directory: %s/.ailang/state/mission-quorum\n' "$root" >&2
    fi
  elif [ -n "${MISSION_QUORUM_ARTIFACT_DIR:-}" ]; then
    mc_quorum_absolute "$MISSION_QUORUM_ARTIFACT_DIR" || return 2
    MC_QUORUM_DIRS=("$MISSION_QUORUM_ARTIFACT_DIR")
  fi
  MC_QUORUM_DIRS+=("$root/.ailang/state/mission-quorum")
}

# Inspect every directory before returning any matches. Do not select a winner
# or change freshness/verdict policy: the caller reads the existing nested schema.
mc_quorum_lookup() {
  if [ -n "${ZSH_VERSION:-}" ]; then setopt local_options null_glob; fi
  local slug="$1" dir file bad=0
  MC_QUORUM_MATCHES=()
  case "$slug" in ''|*[!a-z0-9-]*) mc_quorum_error 'invalid doc-id slug'; return 2 ;; esac
  for dir in "${MC_QUORUM_DIRS[@]}"; do
    printf 'quorum consult: %s\n' "$dir" >&2
    if [ ! -e "$dir" ] && [ ! -L "$dir" ]; then continue; fi
    if [ ! -d "$dir" ] || [ ! -r "$dir" ] || [ ! -x "$dir" ]; then
      mc_quorum_error "cannot read artifact directory: $dir"
      bad=1; continue
    fi
    for file in "$dir/$slug-"*.json; do
      if [ ! -e "$file" ] && [ ! -L "$file" ]; then continue; fi
      if [ ! -f "$file" ] || [ ! -r "$file" ] || ! jq empty "$file" >/dev/null 2>&1; then
        mc_quorum_error "cannot read/parse artifact: $file"
        bad=1; continue
      fi
      MC_QUORUM_MATCHES+=("$file")
    done
  done
  if [ "$bad" = 1 ]; then return 2; fi
  [ "${#MC_QUORUM_MATCHES[@]}" -gt 0 ] && return 0
  return 1
}
