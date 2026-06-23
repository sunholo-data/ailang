#!/usr/bin/env bash
#
# M-SECRET-EFFECT — turnkey demo of AILANG's gated, flow-controlled secrets.
#
# Runs three parts:
#   A. Static leak prevention — `ailang check` rejects a secret leak, accepts the
#      declassified version. (No 1Password needed.)
#   B. Explicit authority — running without --caps Secret is denied.
#   C. Runtime resolution — resolve a secret, declassify it, print the safe form.
#      Uses a FAKE `op` on PATH by default so no real 1Password is required; set
#      USE_REAL_OP=1 (with `op` installed + signed in) to hit a real vault.
#
# Usage:
#   ./examples/runnable/secrets/demo.sh            # fake op (laptop-friendly)
#   USE_REAL_OP=1 ./examples/runnable/secrets/demo.sh   # real `op`
#
# Requires: a built `ailang` on PATH (make install), run from the repo root.

set -euo pipefail

DIR="examples/runnable/secrets"
say() { printf '\n\033[1;36m=== %s ===\033[0m\n' "$1"; }

command -v ailang >/dev/null || { echo "ailang not on PATH — run 'make install'"; exit 1; }

say "A. Static leak prevention (compile-time, no 1Password needed)"
echo "\$ ailang check $DIR/leak_attempt.ail   # expect: REJECTED"
ailang check "$DIR/leak_attempt.ail" && echo "UNEXPECTED: leak_attempt passed!" || true
echo
echo "\$ ailang check $DIR/gated_secret.ail   # expect: clean"
ailang check "$DIR/gated_secret.ail"

say "B. Explicit authority (capability gate)"
echo "\$ ailang run --caps IO --entry main $DIR/secret_demo.ail   # expect: denied"
ailang run --caps IO --entry main "$DIR/secret_demo.ail" && echo "UNEXPECTED: ran without Secret cap!" || true

say "C. Runtime resolution + declassify"
if [ "${USE_REAL_OP:-0}" = "1" ]; then
  echo "(using real op — needs op installed + signed in, and a valid op:// ref)"
  ailang run --caps Secret,IO --entry main "$DIR/secret_demo.ail"
else
  echo "(using a fake op on PATH; set USE_REAL_OP=1 for a real vault)"
  FAKE="$(mktemp -d)"
  printf '#!/bin/sh\n[ "$1" = read ] && printf "sk-live-DEMO-abc123" && exit 0\nexit 1\n' > "$FAKE/op"
  chmod +x "$FAKE/op"
  PATH="$FAKE:$PATH" ailang run --caps Secret,IO --entry main "$DIR/secret_demo.ail"
  rm -rf "$FAKE"
fi

say "Done"
