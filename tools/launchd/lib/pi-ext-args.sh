# shellcheck shell=bash
# pi-ext-args.sh — sourced by the mission driver and scripts/mission_pi_run.sh.
#
# _mc_pi_ext_args DIR → prints pi extension flags (one per line) for a pi run whose CWD is
# DIR, so a caller can read them into an array:
#   _pa=(); while IFS= read -r _l; do _pa+=("$_l"); done < <(_mc_pi_ext_args "$dir")
#
# Why: `ailang pi install` copies the repo's .pi/extensions into the GLOBAL
# ~/.pi/agent/extensions. pi discovers BOTH, so in a checkout that carries its own
# .pi/extensions every tool is registered twice and pi exits rc=1 ("Tool \"quota_report\"
# conflicts with …"). Measured 2026-10-01: 12 fleet fires since 2026-09-30 lost every pi
# lane to it, and the docs evaluator chain fell through to opus.
#
# So when DIR has .pi/extensions, discovery is switched off and the repo's own
# extensions are named explicitly — each loads exactly once, the repo copy winning
# (it is the one versioned with the code). Settings packages (npm:…) are not loaded
# in that case; they never were in a fire, because the fire crashed first.
# When DIR has none (world, stapledon), nothing is printed and discovery is unchanged.
_mc_pi_ext_args() {
  local dir="$1" f
  [ -d "$dir/.pi/extensions" ] || return 0
  printf '%s\n' --no-extensions
  for f in "$dir"/.pi/extensions/*.ts; do
    [ -f "$f" ] || continue
    printf '%s\n%s\n' -e "$f"
  done
}
