# shellcheck shell=bash
# Pure definitions only: safe to source in bash 3.2 and zsh 5.9.

mc_secrets_scope_lane_valid() {
  case "${1-}" in
    claude-controller|codex-controller|pi-controller|claude-role|codex-role|pi-role) return 0 ;;
    *) return 64 ;;
  esac
}

mc_secrets_scope_allow() {
  case "${1-}" in
    claude-controller|codex-controller|pi-controller)
      # C: subscription inference/quorum: tools/launchd/mission-control.sh:157; internal/mission/quorum/claude_reviewer.go:69.
      printf '%s\n' CLAUDE_CODE_OAUTH_TOKEN
      # O: controller quota gauge: cmd/ailang/mission_quota_cmd.go:58.
      printf '%s\n' OLLAMA_API_KEY
      # R: controller quota gauge: cmd/ailang/mission_quota_cmd.go:75.
      printf '%s\n' OPENROUTER_API_KEY
      # Q: GPT quorum seat: internal/mission/quorum/call.go:89,120-128; internal/modelreg/models.yml:577-582.
      printf '%s\n' OPENAI_API_KEY
      ;;
    claude-role)
      # C: subscription inference: tools/launchd/mission-control.sh:157.
      printf '%s\n' CLAUDE_CODE_OAUTH_TOKEN
      ;;
    codex-role) : ;;
    pi-role)
      # P: pi quota gauges: cmd/ailang/pi_assets/provider-quota.ts:35,69,119-120.
      printf '%s\n' OLLAMA_API_KEY
      # P: pi quota gauges: cmd/ailang/pi_assets/provider-quota.ts:35,69,119-120.
      printf '%s\n' OPENROUTER_API_KEY
      ;;
    *) return 64 ;;
  esac
}

mc_secrets_scope_names() {
  [ -f "${1-}" ] && [ -r "$1" ] || return 78
  /usr/bin/awk '/^[ \t]*(export[ \t]+)?[A-Za-z_][A-Za-z0-9_]*=/ {
    n=$0; sub(/^[ \t]*(export[ \t]+)?/, "", n); sub(/=.*/, "", n)
    if (!seen[n]++) print n
  }' "$1" 2>/dev/null || return 78
}

mc_secrets_scope_unset() {
  # zsh aborts on readonly unsets even inside `if`; probe in a subshell first.
  # Successful unsets still run in the calling shell.
  (unset "$1") 2>/dev/null && unset "$1" 2>/dev/null
}

mc_secrets_scope_floor() {
  # Attempt every measured name, even if another is readonly. Failure still refuses launch.
  local _mc_floor_name
  for _mc_floor_name in ANTHROPIC_API_KEY CLAUDE_CODE_OAUTH_TOKEN GEMINI_AGENTS_API_KEY GOOGLE_API_KEY OLLAMA_API_KEY OPENAI_API_KEY OPENROUTER_API_KEY; do
    mc_secrets_scope_unset "$_mc_floor_name" || :
  done
}

mc_secrets_scope_helper_name() {
  case "$1" in
    _mc_scope_lane|_mc_scope_file|_mc_scope_names|_mc_scope_name|_mc_scope_allow|_mc_scope_rc|_mc_scope_kept|_mc_scope_stripped|_mc_scope_nkept|_mc_scope_nstripped) return 0 ;;
    *) return 1 ;;
  esac
}

mc_secrets_scope_prepare() {
  # Positional parameters avoid shadowing inventory variables with local helpers.
  # $1 is the parsed inventory; remaining arguments are our reserved helper names.
  [ "$#" -gt 1 ] || return 0
  case "
$1
" in
    *"
$2
"*) mc_secrets_scope_unset "$2" || return 78 ;;
  esac
  if [ "$#" -gt 2 ]; then
    mc_secrets_scope_prepare "$1" "${@:3}" || return 78
  fi
}

mc_secrets_scope_apply() {
  set -- "${1-}" "${2-${HOME-}/.config/ailang/secrets.env}"
  # Append parser status to its names-only output; substitution alone loses that status.
  set -- "$1" "$2" "$(mc_secrets_scope_names "$2"; printf '\n%s' "$?")"
  if [ "${3##*$'\n'}" != 0 ]; then
    mc_secrets_scope_floor
    printf '%s\n' 'secrets-scope: FAIL-CLOSED; inventory unreadable; emergency floor attempted' >&2
    return 78
  fi
  set -- "$1" "$2" "${3%$'\n'*}"
  # Clear matching globals BEFORE declaring locals of the same name. No values are read.
  if ! mc_secrets_scope_prepare "$3" _mc_scope_lane _mc_scope_file _mc_scope_names _mc_scope_name _mc_scope_allow _mc_scope_rc _mc_scope_kept _mc_scope_stripped _mc_scope_nkept _mc_scope_nstripped; then
    mc_secrets_scope_floor
    printf '%s\n' 'secrets-scope: FAIL-CLOSED; unset failed; emergency floor attempted' >&2
    return 78
  fi
  mc_secrets_scope_apply_inventory "$@"
}

mc_secrets_scope_apply_inventory() {
  local _mc_scope_lane="$1" _mc_scope_file="$2"
  local _mc_scope_names="$3" _mc_scope_name _mc_scope_allow _mc_scope_rc=0
  local _mc_scope_kept='' _mc_scope_stripped='' _mc_scope_nkept=0 _mc_scope_nstripped=0
  # Apply the already parsed names; values are never loaded.
  _mc_scope_allow=$(mc_secrets_scope_allow "$_mc_scope_lane") || _mc_scope_rc=64
  while IFS= read -r _mc_scope_name; do
    [ -n "$_mc_scope_name" ] || continue
    case "
$_mc_scope_allow
" in
      *"
$_mc_scope_name
"*)
        _mc_scope_kept="${_mc_scope_kept}${_mc_scope_kept:+,}$_mc_scope_name"
        _mc_scope_nkept=$((_mc_scope_nkept+1))
        ;;
      *)
        if mc_secrets_scope_helper_name "$_mc_scope_name"; then
          : # Its global was cleared before these locals existed.
        elif ! mc_secrets_scope_unset "$_mc_scope_name"; then
          _mc_scope_rc=78
        fi
        _mc_scope_stripped="${_mc_scope_stripped}${_mc_scope_stripped:+,}$_mc_scope_name"
        _mc_scope_nstripped=$((_mc_scope_nstripped+1))
        ;;
    esac
  done < <(printf '%s\n' "$_mc_scope_names")
  if [ "$_mc_scope_rc" -eq 78 ]; then
    mc_secrets_scope_floor
    printf '%s\n' 'secrets-scope: FAIL-CLOSED; unset failed; emergency floor attempted' >&2
  elif [ "$_mc_scope_rc" -eq 64 ]; then
    printf 'secrets-scope: FAIL-CLOSED; unknown lane; kept=0[] stripped=%s[%s]\n' "$_mc_scope_nstripped" "$_mc_scope_stripped" >&2
  else
    printf 'secrets-scope: lane=%s kept=%s[%s] stripped=%s[%s]\n' "$_mc_scope_lane" "$_mc_scope_nkept" "$_mc_scope_kept" "$_mc_scope_nstripped" "$_mc_scope_stripped" >&2
  fi
  return "$_mc_scope_rc"
}
