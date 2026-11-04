#!/bin/bash
# Post-fire detector. Never source assignments or put secret values in child argv.
set +x
umask 077
secrets_file="$HOME/.config/ailang/secrets.env"
marker='' min_len=20 scratch='' active_child=''
files=0 keys=0 leaks=0 failure=0
roots=() names=() values=() key_names=() key_patterns=() seen=()

# Paths stay recognizable, but no pathname can forge a second output record.
escape_path() {
  escaped=${1//\\/\\\\}
  local code char replacement
  for ((code=1; code<32; code++)); do
    printf -v char '%b' "\\$(printf '%03o' "$code")"
    printf -v replacement '\\u%04x' "$code"
    escaped=${escaped//"$char"/"$replacement"}
  done
  escaped=${escaped//$'\177'/\\u007f}
}
diag() {
  escape_path "$2"
  printf 'transcript-secret-scan: %s: %s\n' "$1" "$escaped" >&2
}
# shellcheck disable=SC2329 # Called by the EXIT handler.
cleanup() {
  if [ -n "$active_child" ]; then
    kill "$active_child" 2>/dev/null || true
    kill -KILL "$active_child" 2>/dev/null || true
    wait "$active_child" 2>/dev/null || true
  fi
  if [ -n "$scratch" ] && ! rm -rf "$scratch" 2>/dev/null; then
    failure=1
    diag 'INSTRUMENT FAILURE removing scratch directory' "$scratch"
  fi
}
# shellcheck disable=SC2329 # Invoked by trap.
terminal() {
  local status=$? verdict
  trap - EXIT TERM INT
  [ "$status" -le 1 ] || failure=1
  cleanup
  if [ "$failure" -ne 0 ]; then verdict=instrument-failure; status=2
  elif [ "$leaks" -gt 0 ]; then verdict=LEAK; status=1
  else verdict=clean
  fi
  escape_path "$marker"
  printf 'transcript-secret-scan: verdict=%s files=%s keys=%s leaks=%s since=%s\n' \
    "$verdict" "$files" "$keys" "$leaks" "$escaped"
  exit "$status"
}
trap terminal EXIT
trap 'failure=1; diag "INSTRUMENT FAILURE interrupted" "$marker"; exit 2' TERM INT
argument_error() { diag 'INSTRUMENT FAILURE invalid arguments' ''; exit 2; }
while [ "$#" -gt 0 ]; do
  case "$1" in
    --secrets-file|--since|--root|--min-len)
      [ "$#" -ge 2 ] || argument_error
      case "$1" in
        --secrets-file) secrets_file=$2 ;;
        --since) marker=$2 ;;
        --root) roots+=("$2") ;;
        --min-len) min_len=$2 ;;
      esac
      shift 2 ;;
    *) argument_error ;;
  esac
done
case "$min_len" in ''|*[!0-9]*) argument_error ;; esac
while [ "${min_len#0}" != "$min_len" ]; do min_len=${min_len#0}; done
[ -n "$min_len" ] || argument_error
if [ ! -f "$marker" ] || [ ! -r "$marker" ]; then
  diag 'INSTRUMENT FAILURE missing or unreadable marker' "$marker"; exit 2
fi
if ! { exec 4< "$secrets_file"; } 2>/dev/null; then
  diag 'INSTRUMENT FAILURE opening secrets file' "$secrets_file"; exit 2
fi
if [ ! -f "$secrets_file" ] || [ ! -r "$secrets_file" ]; then
  diag 'INSTRUMENT FAILURE unreadable secrets file' "$secrets_file"; exit 2
fi
# Last assignment wins, even when it replaces a long value with a short one.
while IFS= read -r line <&4 || [ -n "$line" ]; do
  line="${line#"${line%%[![:space:]]*}"}"
  case "$line" in ''|\#*) continue ;; esac
  if [[ "$line" = export[[:space:]]* ]]; then
    line=${line#export}
    line="${line#"${line%%[![:space:]]*}"}"
  fi
  case "$line" in *=*) ;; *) continue ;; esac
  name=${line%%=*}
  [[ "$name" =~ ^[a-zA-Z_][a-zA-Z0-9_]*$ ]] || continue
  value=${line#*=}
  case "$value" in
    \"*\") value=${value:1:${#value}-2} ;;
    \'*\') value=${value:1:${#value}-2} ;;
  esac
  index=${#names[@]}
  for ((i=0; i<${#names[@]}; i++)); do
    if [ "${names[i]}" = "$name" ]; then index=$i; break; fi
  done
  names[index]=$name; values[index]=$value
done
exec 4<&-
for ((i=0; i<${#names[@]}; i++)); do
  value=${values[i]}
  if [ "${#min_len}" -gt 9 ] || [ "${#value}" -lt "$min_len" ]; then continue; fi
  json=${value//\\/\\\\}
  json=${json//\"/\\\"}
  json=${json//$'\b'/\\b}; json=${json//$'\f'/\\f}
  json=${json//$'\n'/\\n}; json=${json//$'\r'/\\r}; json=${json//$'\t'/\\t}
  slashed=${json//\//\\/}
  patterns=$value
  [ "$json" = "$value" ] || patterns="$patterns"$'\n'"$json"
  if [ "$slashed" != "$json" ] && [ "$slashed" != "$value" ]; then
    patterns="$patterns"$'\n'"$slashed"
  fi
  key_names+=("${names[i]}"); key_patterns+=("$patterns")
done
keys=${#key_names[@]}
if [ "$keys" -eq 0 ]; then
  diag 'INSTRUMENT FAILURE zero eligible keys' "$secrets_file"; exit 2
fi
if [ "${#roots[@]}" -eq 0 ]; then
  roots=("$HOME/.codex-missions/sessions" "$HOME/.codex/sessions"
    "$HOME/.pi/agent/sessions" "$HOME/.claude/projects")
fi
scratch=$(mktemp -d "${TMPDIR:-/tmp}/transcript-secret-scan.XXXXXX" 2>/dev/null) || {
  diag 'INSTRUMENT FAILURE creating scratch directory' "${TMPDIR:-/tmp}"; exit 2;
}
for root in "${roots[@]}"; do
  if [ ! -e "$root" ]; then diag 'skipped missing root' "$root"; continue; fi
  if [ ! -d "$root" ]; then
    failure=1; diag 'INSTRUMENT FAILURE accessing root' "$root"; continue
  fi
  if ! { : > "$scratch/candidates"; } 2>/dev/null; then
    failure=1; diag 'INSTRUMENT FAILURE creating candidate list' "$scratch"; continue
  fi
  find "$root" -type f -newer "$marker" -print0 > "$scratch/candidates" 2>/dev/null &
  active_child=$!
  wait "$active_child"; find_rc=$?; active_child=
  if [ "$find_rc" -ne 0 ]; then
    failure=1; diag 'INSTRUMENT FAILURE enumerating root' "$root"
  fi
  if ! { exec 5< "$scratch/candidates"; } 2>/dev/null; then
    failure=1; diag 'INSTRUMENT FAILURE opening candidate list' "$scratch"; continue
  fi
  while IFS= read -r -d '' file <&5; do
    duplicate=0
    for previous in "${seen[@]}"; do
      if [ "$previous" = "$file" ]; then duplicate=1; break; fi
    done
    [ "$duplicate" -eq 0 ] || continue
    seen+=("$file"); files=$((files + 1))
    for ((i=0; i<keys; i++)); do
      patterns=${key_patterns[i]}
      grep -F -c -f /dev/fd/3 -- "$file" 3<<< "$patterns" >/dev/null 2>/dev/null &
      active_child=$!
      wait "$active_child"; match_rc=$?; active_child=
      case "$match_rc" in
        0)
          leaks=$((leaks + 1)); escape_path "$file"
          printf 'LEAK key=%s file=%s\n' "${key_names[i]}" "$escaped"
          ;;
        1) ;;
        *) failure=1; diag 'INSTRUMENT FAILURE matching file' "$file" ;;
      esac
    done
  done
  exec 5<&-
done
if [ "$failure" -ne 0 ]; then exit 2
elif [ "$leaks" -gt 0 ]; then exit 1
else exit 0
fi
