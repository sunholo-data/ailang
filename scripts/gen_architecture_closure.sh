#!/bin/bash
# gen_architecture_closure.sh — write the "Language closure" section of
# ARCHITECTURE.md from the real import graph, so the map is derived, not drawn.
#
# The lists (language roots, platform deny-list, third-party leak roots) come
# from tools/simplicity_metrics.sh; the enforced state comes from
# internal/diag/closure_expected_violations.txt; the closure itself from
# `go list -deps`. The section sits between two HTML-comment markers and is the
# only part of ARCHITECTURE.md this script touches.
#
# Usage:
#   scripts/gen_architecture_closure.sh          # rewrite the section in place
#   scripts/gen_architecture_closure.sh --check  # exit 1 if it would change (CI)
#
# bash 3.2 / BSD grep compatible.
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

MODULE="github.com/sunholo-data/ailang"
DOC=ARCHITECTURE.md
BEGIN='<!-- BEGIN GENERATED: language closure (scripts/gen_architecture_closure.sh) -->'
END='<!-- END GENERATED: language closure -->'
CHECK=0
[ "${1:-}" = "--check" ] && CHECK=1

list_from_script() {
  sed -nE "s/^$1=\"([^\"]+)\"/\1/p" tools/simplicity_metrics.sh
}
roots="$(list_from_script LANGUAGE_ROOTS)"
platform="$(list_from_script PLATFORM_PKGS)"
leaks="$(list_from_script LEAK_ROOTS)"

args=""
for r in $roots; do args="$args ./$r"; done
# shellcheck disable=SC2086
closure="$(go list -deps $args | grep "^$MODULE/internal/" | sed "s#^$MODULE/##" | sort)"
count="$(printf '%s\n' "$closure" | grep -c . || true)"
binary="$(go list -deps ./cmd/ailang | grep -c "^$MODULE/internal/" || true)"
violations="$(grep -v '^#' internal/diag/closure_expected_violations.txt | grep -v '^$' || true)"
vcount="$(printf '%s\n' "$violations" | grep -c . || true)"

section="$(
cat <<EOF
$BEGIN
### Language closure (generated $(date +%Y-%m-%d) @ $(git rev-parse --short HEAD))

What \`ailang run / check / fmt / prompt / repl\` link, measured with \`go list -deps\`
over the language roots. **$count** of the internal packages are in the closure; the
full binary links **$binary**. The gate is \`internal/diag/closure_test.go\`; the numbers
are banked by \`make simplicity-metrics\`. Regenerate this section with
\`scripts/gen_architecture_closure.sh\` (CI runs it with \`--check\`).

**Language roots** (\`tools/simplicity_metrics.sh\` LANGUAGE_ROOTS):
$(for r in $roots; do printf '%s' "\`$r\` "; done)

**In the closure** ($count packages):

$(printf '%s\n' "$closure" | sed 's/^/- `/; s/$/`/')

**Must never be reached from the roots** — platform packages:
$(for p in $platform; do printf '%s' "\`$p\` "; done)

— and third-party roots:
$(for l in $leaks; do printf '%s' "\`$l\` "; done)

**Known violations still listed** ($vcount; the list can only shrink — the test fails
if an entry appears that is not listed, or a listed entry is no longer reached):

$(if [ "$vcount" -eq 0 ]; then echo "- none — the language core is a leaf of the platform"; else printf '%s\n' "$violations" | sed 's/^/- `/; s/$/`/'; fi)
$END
EOF
)"

if ! grep -qF "$BEGIN" "$DOC"; then
  # First run: insert before "## Capability-effect system".
  python3 - "$DOC" "$section" <<'PY'
import sys
doc, section = sys.argv[1], sys.argv[2]
s = open(doc).read()
anchor = "## Capability-effect system"
assert anchor in s, "anchor heading not found"
s = s.replace(anchor, section + "\n\n" + anchor, 1)
open(doc, "w").write(s)
PY
  [ "$CHECK" -eq 1 ] && { echo "ARCHITECTURE.md had no generated section; inserted — commit it" >&2; exit 1; }
  echo "inserted generated section into $DOC"
  exit 0
fi

tmp="$(mktemp -t archgen.XXXXXX)"
trap 'rm -f "$tmp"' EXIT
python3 - "$DOC" "$section" "$BEGIN" "$END" "$tmp" <<'PY'
import sys
doc, section, begin, end, out = sys.argv[1:6]
s = open(doc).read()
i, j = s.index(begin), s.index(end) + len(end)
open(out, "w").write(s[:i] + section + s[j:])
PY

# The date/commit line changes on every run; compare everything else.
strip() { grep -v '^### Language closure (generated' "$1"; }
if diff -q <(strip "$DOC") <(strip "$tmp") >/dev/null; then
  echo "ARCHITECTURE.md language-closure section is current"
  exit 0
fi
if [ "$CHECK" -eq 1 ]; then
  echo "ARCHITECTURE.md language-closure section is STALE — run scripts/gen_architecture_closure.sh and commit" >&2
  diff <(strip "$DOC") <(strip "$tmp") | head -40 >&2 || true
  exit 1
fi
cp "$tmp" "$DOC"
echo "rewrote the language-closure section of $DOC"
