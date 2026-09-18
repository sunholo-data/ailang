#!/bin/bash
# check_prompt_commands.sh — every `ailang <cmd> [<sub>]` the devtools prompt
# TEACHES must be a route the binary ACCEPTS.
#
# Why this exists (M-V1-SIMPLIFY-S5, 2026-09-18): `make check-prompt-freeze`
# pins each prompt's SHA256, so it proves the file has not changed. It cannot
# notice that the BINARY changed underneath it. S5 M3 deleted eight
# `observatory` subcommands and the freeze gate stayed green while
# `ailang devtools-prompt` went on teaching all eight to every agent that read
# it — the same defect class as `ailang fmt` telling eval models their correct
# code was non-canonical for two weeks. The freeze gate pins the prompt's
# BYTES; this one pins its TRUTH.
#
# SAFETY — read before editing. The first version of this script probed each
# opener by running `"$BIN" $opener --help`. Two things went wrong at once:
#   1. the token class [a-z0-9-]+ matches a FLAG, because `-` inside a bracket
#      expression is literal — so `ailang test --format json` yielded the
#      opener `ailang test --format`;
#   2. `ailang test --format --help` does not print help, it RUNS THE TEST
#      SUITE against the working directory.
# Several `ailang test` processes were left grinding for half an hour.
# So: this script NEVER passes a captured token to the binary. It invokes
# `"$BIN" <name> --help` for bare command names only — which M1 made
# side-effect-free and exit-0 at every level — and does all matching as text.
#
# Usage: check_prompt_commands.sh [path-to-ailang-binary]
# bash 3.2 compatible (the rig runs 3.2.57). BSD grep: no \s, no BRE \|.

set -u

BIN="${1:-bin/ailang}"
# The SOURCE OF TRUTH, not cmd/ailang/prompts/devtools — that is a generated
# mirror. `make prepare-embed` (make/build.mk) does `rm -rf cmd/ailang/prompts;
# cp -r prompts cmd/ailang/prompts` whenever the two differ, so a fix applied
# only to the mirror is silently reverted by the next build. That happened to
# the first version of this very fix: the gate passed on a clean checkout and
# went red again after `make lint`. Reading the source means a mirror-only fix
# can never look green.
PROMPTS="prompts/devtools"

if [ ! -x "$BIN" ]; then
  echo "check-prompt-commands: no binary at $BIN — run 'make build' first" >&2
  exit 1
fi

work=$(mktemp -d "${TMPDIR:-/tmp}/promptcmd.XXXXXX") || exit 1
trap 'rm -rf "$work"' EXIT

# Openers: `ailang <name>` optionally followed by a second word. Both tokens
# must START WITH A LETTER, which is what keeps flags (--format), placeholders
# (<id>) and paths ($FILE) out. Anything else on the line is ignored.
grep -ohE '^ailang [a-z][a-z0-9-]*( [a-z][a-z0-9-]*)?' "$PROMPTS"/*.md \
  | sed 's/^ailang //' | sort -u > "$work/openers"

# Commands whose subcommands may be probed by RUNNING `<name> <sub> --help`.
# Every one is inspection-only: it reads state and prints. A command that can
# execute, build, serve, watch or spend is NOT here, because `--help` after an
# unrecognised token does not reliably mean "print help" — `ailang test <x>
# --help` runs the suite. Nor is a command whose second word in the prompt is
# an ARGUMENT rather than a subcommand (`check <file>`, `verify <file>`,
# `iface <module>`): probing those would report a placeholder as a dead route.
SUBCOMMAND_PROBE_SAFE="observatory dashboard trace chains eval-chains coordinator \
messages models workspaces budget cache mission storage examples docs builtins pkg"

dead=0
checked=0
unprobed=0

while read -r opener; do
  [ -n "$opener" ] || continue
  name=${opener%% *}
  sub=""
  case "$opener" in *" "*) sub=${opener#* };; esac
  checked=$((checked + 1))

  # Is the top-level name a route? One safe `<name> --help`, cached per name.
  helpfile="$work/help.$name"
  if [ ! -f "$helpfile" ]; then
    "$BIN" "$name" --help </dev/null >"$helpfile" 2>&1
  fi
  if grep -qE 'nknown command' "$helpfile"; then
    echo "  DEAD: ailang $opener   (no such command)" >&2
    dead=$((dead + 1))
    continue
  fi

  [ -n "$sub" ] || continue

  case " $SUBCOMMAND_PROBE_SAFE " in
    *" $name "*) ;;
    *) unprobed=$((unprobed + 1)); continue ;;
  esac

  out=$("$BIN" "$name" "$sub" --help </dev/null 2>&1)
  case "$out" in
    *"nknown "*"subcommand"*|*"nknown command"*)
      echo "  DEAD: ailang $opener" >&2
      dead=$((dead + 1))
      ;;
  esac
done < "$work/openers"

if [ "$dead" -gt 0 ]; then
  echo "check-prompt-commands: $dead of $checked taught routes are not accepted by the binary" >&2
  echo "  The devtools prompt is what agents believe. Fix the prompt AND bump its" >&2
  echo "  hash in $PROMPTS/versions.json, or restore the command." >&2
  exit 1
fi

echo "check-prompt-commands: $checked taught routes accepted by the binary ($unprobed subcommands of execute-capable commands not probed, by design)"
