#!/bin/bash
# tests:driver-env-leaks-into-launchd-suite: live mission exports made routing 86/87 red.
# Suites needing a variable must set it inside the test, never inherit it.
set -e

if [ "$#" -lt 1 ]; then
  echo "usage: suite-env.sh <script> [args...]" >&2
  exit 2
fi

clean_env=()
for name in HOME PATH TMPDIR USER LOGNAME SHELL LANG LC_ALL LC_CTYPE TERM; do
  if [ "${!name+x}" = x ]; then
    clean_env+=("$name=${!name}")
  fi
done

exec env -i "${clean_env[@]}" /bin/bash "$@"
