#!/bin/bash
# cli_surface_snapshot.sh — record the observable CLI surface of an ailang
# binary into one deterministic file, so two binaries can be byte-diffed.
#
#   tools/cli_surface_snapshot.sh <binary> <outfile>
#
# For every top-level command name and alias in COMMANDS below it runs
#   <bin> <cmd> --help
#   <bin> <cmd>            (unless the command is in HELP_ONLY)
# and records exit code, stdout and stderr.
#
# HELP_ONLY holds everything that would start a server, open a network
# connection, spend money or mutate state when run bare: daemon, server/serve,
# serve-api, lsp, repl, watch, mcp, eval, eval-suite (no safe no-op scope —
# omitting --models/--tier runs dev_models x every tier, real spend), the
# registry mutators, and the installers.
#
# Determinism: every invocation gets a FRESH empty $HOME and a FRESH empty
# working directory, stdin is /dev/null, and the output is scrubbed of
# timestamps, durations, pids and machine paths. Build both binaries with
# IDENTICAL ldflags — a version stamp once produced 196 false diffs.
#
#   go build -ldflags "-X github.com/sunholo-data/ailang/internal/version.Version=vSNAP \
#     -X github.com/sunholo-data/ailang/internal/version.Commit=SNAPSHOT \
#     -X github.com/sunholo-data/ailang/internal/version.BuildTime=SNAPSHOT" \
#     -o bin/ailang-before ./cmd/ailang
#
# Written for bash 3.2 (the rig's shell): no associative arrays, no ${v,,}.
# Run it under /bin/bash — the interactive shell shadows grep with ugrep.

set -u

BIN="${1:-}"
OUT="${2:-}"

if [ -z "$BIN" ] || [ -z "$OUT" ]; then
  echo "usage: $0 <binary> <outfile>" >&2
  exit 2
fi
if [ ! -x "$BIN" ]; then
  echo "$0: no executable at $BIN" >&2
  exit 2
fi

case "$BIN" in
  /*) ;;
  *) BIN="$PWD/$BIN" ;;
esac
case "$OUT" in
  /*) ;;
  *) OUT="$PWD/$OUT" ;;
esac

# Per-command wall-clock budget. A command that outlives it is killed and
# recorded as TIMEOUT rather than wedging the snapshot.
TIMEOUT_SECS="${CLI_SNAPSHOT_TIMEOUT:-20}"

# Every top-level name the pre-S5 switch in cmd/ailang/main.go accepts,
# aliases included (msg, brain, microrag, urag, serve). Keep this list stable:
# it is the fixture both the before- and after-binary are measured against.
COMMANDS="
version
run
repl
test
watch
check
fmt
iface
internal-dump-iface
select-best
ast-edit
export-training
eval
eval-analyze
eval-compare
eval-paired
eval-censored-pairs
eval-matrix
eval-sweet-spot
eval-summary
eval-report
eval-suite
browser-profile
eval-elo
eval-trend
eval-publish
eval-chains
doctor
builtins
docs
debug
messages
msg
cache
brain
micro-rag
microrag
urag
prompt
devtools-prompt
agent-prompt
mcp
daemon
server
serve
serve-api
lsp
init
access-control
compile
disasm
editor
pi
axioms
replay
trace
observatory
chains
dashboard
budget
models
mission
coordinator
storage
workspaces
exec
design-review
design-quorum
verify
add
lock
tree
install
search
publish
unpublish
generate-extension-registry
pkg-docs
pkg
ai-check
policy-check
examples
sandbox-check
"

# Bare (no-argument) runs are suppressed for these: they start a server, open a
# network connection, spend money, install something or mutate a registry.
HELP_ONLY="
daemon
server
serve
serve-api
lsp
repl
watch
mcp
eval
eval-suite
exec
publish
unpublish
install
add
lock
generate-extension-registry
pi
editor
storage
design-review
design-quorum
doctor
browser-profile
"

# `pkg` is the one nested group in the pre-S5 switch; probe each verb's --help.
PKG_SUBS="quality info versions stats notify-upgrade affected-by provenance history cascade key"

is_help_only() {
  for _n in $HELP_ONLY; do
    if [ "$_n" = "$1" ]; then return 0; fi
  done
  return 1
}

SCRATCH="$(mktemp -d "${TMPDIR:-/tmp}/ailang-cli-snapshot.XXXXXX")"
trap 'rm -rf "$SCRATCH"' EXIT INT TERM

# scrub reads a captured stream on stdin and writes the stable form. Machine
# paths, timestamps, durations, pids and byte/size counters are replaced by
# placeholders so two runs on two machines agree.
scrub() {
  LC_ALL=C sed -E \
    -e "s|$SCRATCH|<SCRATCH>|g" \
    -e "s|$HOME|<HOME>|g" \
    -e 's|/private/var/folders/[A-Za-z0-9_/.+-]*|<TMP>|g' \
    -e 's|/var/folders/[A-Za-z0-9_/.+-]*|<TMP>|g' \
    -e 's|/tmp/[A-Za-z0-9_/.+-]*|<TMP>|g' \
    -e 's/[0-9]{4}-[0-9]{2}-[0-9]{2}[T ][0-9]{2}:[0-9]{2}:[0-9]{2}([.,][0-9]+)?(Z|[+-][0-9]{2}:?[0-9]{2})?/<TIMESTAMP>/g' \
    -e 's/[0-9]{4}\/[0-9]{2}\/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}/<TIMESTAMP>/g' \
    -e 's/[0-9]{4}-[0-9]{2}-[0-9]{2}/<DATE>/g' \
    -e 's/[0-9]{2}:[0-9]{2}:[0-9]{2}/<TIME>/g' \
    -e 's/[0-9]+(\.[0-9]+)?(ns|us|µs|ms)\b/<DUR>/g' \
    -e 's/\b[0-9]+(\.[0-9]+)?s\b/<DUR>/g' \
    -e 's/\b(pid|PID)[ =:]+[0-9]+/\1 <PID>/g'
}

# record runs one invocation in a pristine HOME and CWD and appends its result.
record() {
  label="$1"
  shift
  runhome="$SCRATCH/home.$$"
  runcwd="$SCRATCH/cwd.$$"
  rm -rf "$runhome" "$runcwd"
  mkdir -p "$runhome" "$runcwd"

  outf="$SCRATCH/out"
  errf="$SCRATCH/err"
  : >"$outf"
  : >"$errf"

  (
    cd "$runcwd" || exit 127
    HOME="$runhome" \
      NO_COLOR=1 \
      "$BIN" "$@" >"$outf" 2>"$errf" </dev/null
  ) &
  pid=$!

  # Watchdog instead of a poll loop: bash may not reap the child before
  # `kill -0` is next asked about it, and a spin on a zombie would report
  # every fast command as a timeout.
  ( sleep "$TIMEOUT_SECS"; kill -9 "$pid" 2>/dev/null ) >/dev/null 2>&1 &
  watchdog=$!

  wait "$pid"
  rc=$?
  kill "$watchdog" 2>/dev/null
  wait "$watchdog" 2>/dev/null
  if [ "$rc" = "137" ]; then
    rc="TIMEOUT(or SIGKILL)"
  fi

  {
    echo "=== $label"
    echo "--- exit: $rc"
    echo "--- stdout:"
    scrub <"$outf"
    echo "--- stderr:"
    scrub <"$errf"
    echo
  } >>"$OUT"

  rm -rf "$runhome" "$runcwd"
}

: >"$OUT"
{
  echo "# ailang CLI surface snapshot"
  echo "# generated by tools/cli_surface_snapshot.sh — do not hand-edit"
  echo
} >>"$OUT"

# Global entry points first.
record "ailang"
record "ailang --help" --help
record "ailang --version" --version
record "ailang this-command-does-not-exist" this-command-does-not-exist

for cmd in $COMMANDS; do
  record "ailang $cmd --help" "$cmd" --help
  if is_help_only "$cmd"; then
    echo "=== ailang $cmd  [bare run suppressed: HELP_ONLY]" >>"$OUT"
    echo >>"$OUT"
  else
    record "ailang $cmd" "$cmd"
  fi
done

for sub in $PKG_SUBS; do
  record "ailang pkg $sub --help" pkg "$sub" --help
done

echo "# end of snapshot" >>"$OUT"
