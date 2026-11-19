#!/bin/bash
# Provider-pinned role entry point. Root comes from this wrapper, never from the caller.
# shellcheck shell=bash
set -u
if [ "$#" -lt 3 ] || [ "$2" != -- ]; then
  echo 'usage: mission_scoped_run.sh <codex-role|claude-role> -- <command...>' >&2
  exit 64
fi
lane=$1
shift 2
RUNNER_ROOT=$(cd "$(dirname "$0")/.." && pwd -P) || exit 78
# shellcheck source=/dev/null
. "$RUNNER_ROOT/tools/launchd/lib/secrets-scope.sh" || exit 78
export AILANG_SECRETS_LANE="$lane" MISSION_DRIVER_ROOT="$RUNNER_ROOT"
mc_secrets_scope_apply "$AILANG_SECRETS_LANE" || exit $?
case "$lane" in
  codex-role)
    # Append per-variable controls without changing the command's existing arguments or stdin.
    # shellcheck source=/dev/null
    . "$RUNNER_ROOT/tools/launchd/lib/codex-env-args.sh" || exit 78
    mc_codex_env_args || exit $?
    exec "$@" ${MC_CODEX_ENV_ARGS[@]+"${MC_CODEX_ENV_ARGS[@]}"}
    ;;
  claude-role) exec "$@" ;;
  *) echo 'mission-scoped-run: unsupported role lane; launch refused' >&2; exit 64 ;;
esac
