#!/bin/bash
# Post-fire detector. Never source assignments or put secret values in child argv.
set +x
umask 077
secrets_file="$HOME/.config/ailang/secrets.env"
marker='' min_len=20 scratch='' active_child=''
snapshot_out='' snapshot_tmp='' baseline_file='' baseline_given=0 baseline=none
baseline_paths=() baseline_sizes=()
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
  if [ -n "$snapshot_tmp" ]; then rm -f "$snapshot_tmp" 2>/dev/null || true; fi
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
  if [ -n "$snapshot_out" ]; then
    [ "$status" -eq 0 ] && [ "$failure" -eq 0 ] && exit 0
    exit 2
  fi
  if [ "$failure" -ne 0 ]; then verdict=instrument-failure; status=2
  elif [ "$leaks" -gt 0 ]; then verdict=LEAK; status=1
  else verdict=clean
  fi
  escape_path "$marker"
  printf 'transcript-secret-scan: verdict=%s files=%s keys=%s leaks=%s baseline=%s since=%s\n' \
    "$verdict" "$files" "$keys" "$leaks" "$baseline" "$escaped"
  exit "$status"
}
trap terminal EXIT
trap 'failure=1; diag "INSTRUMENT FAILURE interrupted" "$marker"; exit 2' TERM INT
argument_error() { diag 'INSTRUMENT FAILURE invalid arguments' ''; exit 2; }
while [ "$#" -gt 0 ]; do
  case "$1" in
    --secrets-file|--since|--root|--min-len|--snapshot-out|--baseline)
      [ "$#" -ge 2 ] || argument_error
      case "$1" in
        --secrets-file) secrets_file=$2 ;;
        --since) marker=$2 ;;
        --root) roots+=("$2") ;;
        --min-len) min_len=$2 ;;
        --snapshot-out) snapshot_out=$2; [ -n "$snapshot_out" ] || argument_error ;;
        --baseline) baseline_file=$2; baseline_given=1 ;;
      esac
      shift 2 ;;
    *) argument_error ;;
  esac
done
# Both modes use the same inventory roots. Snapshot mode reads metadata only,
# before marker validation or opening any secrets file.
if [ "${#roots[@]}" -eq 0 ]; then
  roots=("$HOME/.codex-missions/sessions" "$HOME/.codex/sessions"
    "$HOME/.pi/agent/sessions" "$HOME/.claude/projects")
fi
if [ -n "$snapshot_out" ]; then
  snapshot_tmp="${snapshot_out}.tmp"
  python3 - "$snapshot_tmp" "$snapshot_out" "${roots[@]}" 2>/dev/null <<'SNAPSHOT' &
import os
import stat
import sys

try:
    pending, destination = sys.argv[1:3]
    seen = set()
    def walk_error(error):
        raise error
    fd = os.open(pending, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(fd, "wb") as output:
        os.fchmod(output.fileno(), 0o600)
        for root in map(os.fsencode, sys.argv[3:]):
            try:
                mode = os.lstat(root).st_mode
            except FileNotFoundError:
                continue
            if stat.S_ISLNK(mode):
                continue
            if not stat.S_ISDIR(mode):
                raise ValueError("invalid root")
            for directory, _, files in os.walk(root, followlinks=False, onerror=walk_error):
                for name in files:
                    path = os.path.join(directory, name)
                    if path in seen:
                        continue
                    metadata = os.lstat(path)
                    if stat.S_ISREG(metadata.st_mode):
                        seen.add(path)
                        output.write(str(metadata.st_size).encode("ascii") + b"\0" + path + b"\0")
    os.replace(pending, destination)
except Exception:
    sys.exit(2)
SNAPSHOT
  active_child=$!
  wait "$active_child"; snapshot_rc=$?; active_child=''
  if [ "$snapshot_rc" -ne 0 ]; then
    diag 'INSTRUMENT FAILURE creating metadata snapshot' "$snapshot_out"; exit 2
  fi
  exit 0
fi
# Normalize before arithmetic; reserve one byte for tail's one-based offset.
valid_size() {
  size=$1
  case "$size" in ''|*[!0-9]*) return 1 ;; esac
  while [ "${#size}" -gt 1 ] && [ "${size#0}" != "$size" ]; do size=${size#0}; done
  [ "${#size}" -lt 19 ] || {
    # Compare decimal strings before allowing Bash signed arithmetic.
    # shellcheck disable=SC2071
    [ "${#size}" -eq 19 ] && [[ "$size" < 9223372036854775807 ]]
  }
}
if [ "$baseline_given" -eq 1 ]; then
  baseline=failed
  baseline_valid=1
  if [ -f "$baseline_file" ] && [ -r "$baseline_file" ] && { exec 6< "$baseline_file"; } 2>/dev/null; then
    while :; do
      saved_size=''
      if ! IFS= read -r -d '' saved_size <&6; then
        [ -z "$saved_size" ] || baseline_valid=0
        break
      fi
      saved_path=''
      if ! IFS= read -r -d '' saved_path <&6 || [ -z "$saved_path" ] || ! valid_size "$saved_size"; then
        baseline_valid=0; break
      fi
      baseline_paths+=("$saved_path"); baseline_sizes+=("$size")
    done
    exec 6<&-
    if [ "$baseline_valid" -eq 1 ]; then baseline=${#baseline_paths[@]}; fi
  fi
  if [ "$baseline" = failed ]; then
    baseline_paths=(); baseline_sizes=()
    diag 'baseline unavailable; using whole-file fallback' "$baseline_file"
  fi
fi
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
    match_file=$file
    offset=0
    for ((b=0; b<${#baseline_paths[@]}; b++)); do
      if [ "${baseline_paths[b]}" = "$file" ]; then
        if ! current_size=$(stat -f '%z' "$file" 2>/dev/null) || ! valid_size "$current_size"; then
          failure=1; diag 'INSTRUMENT FAILURE reading file size' "$file"; continue 2
        fi
        if [ "$size" -ge "${baseline_sizes[b]}" ]; then
          offset=${baseline_sizes[b]}
        else
          diag 'whole-file fallback for truncated file' "$file"
        fi
        break
      fi
    done
    # Historical exposures remain parked under D-FLEET-42. Do not overlap the
    # saved boundary or re-read historical content on a successful baseline.
    if [ "$offset" -gt 0 ]; then
      tail -c "+$((offset + 1))" 2>/dev/null < "$file" > "$scratch/suffix" &
      active_child=$!
      wait "$active_child"; read_rc=$?; active_child=''
      if [ "$read_rc" -ne 0 ]; then
        failure=1; diag 'INSTRUMENT FAILURE reading file suffix' "$file"; continue
      fi
      match_file=$scratch/suffix
    fi
    for ((i=0; i<keys; i++)); do
      patterns=${key_patterns[i]}
      grep -F -c -f /dev/fd/3 -- "$match_file" 3<<< "$patterns" >/dev/null 2>/dev/null &
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
