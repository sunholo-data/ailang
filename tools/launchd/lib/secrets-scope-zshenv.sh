# shellcheck shell=bash
# No marker means no file access and no shell changes.
[ -n "${AILANG_SECRETS_LANE-}" ] || return 0
[ -n "${MISSION_DRIVER_ROOT-}" ] &&
  [ -r "$MISSION_DRIVER_ROOT/tools/launchd/lib/secrets-scope.sh" ] || return 78
# shellcheck disable=SC1091 # Authoritative pinned root supplied by the mission driver.
. "$MISSION_DRIVER_ROOT/tools/launchd/lib/secrets-scope.sh" || return 78
mc_secrets_scope_apply "$AILANG_SECRETS_LANE"
return $?
