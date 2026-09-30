#!/usr/bin/env bash
# Resolve, compile, execute native tests and report package quality evidence.
set -euo pipefail
cd "${1:-.}"
AILANG_BIN="${AILANG_BIN:-ailang}"
"$AILANG_BIN" lock
"$AILANG_BIN" check --package .
"$AILANG_BIN" test --package .
# Fails explicitly on binaries without `pkg quality`; never silently downgrade to
# compilation only. --strict promotes warn-level badges to failures.
"$AILANG_BIN" pkg quality --strict .
printf '%s\n' 'Compilation, *_test.ail tests and package quality completed. Inline test blocks in source modules are NOT run in package mode: run `ailang test <module.ail>` for each. Review test totals/skips; proof and publication are separate.'
