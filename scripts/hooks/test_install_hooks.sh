#!/usr/bin/env bash
# Arms for scripts/hooks/install_hooks.sh (bash 3.2). Throwaway repos only: a main checkout,
# a linked worktree under .claude/worktrees/, and a second main checkout whose committed tree
# has no hook. Never touches the real clone's hooks dir.
#
# WHY: on 2026-09-27 the main checkout's pre-push was linked into a worktree
# (.claude/worktrees/fleet-rulings) that was later removed. The link dangled, git skipped the
# hook silently, and the installer refused to replace it because it was "not ours", so every
# session's SessionStart re-ran the installer and changed nothing (m-v1-release-gates §5).
set -u
INSTALL="$(cd "$(dirname "$0")" && pwd)/install_hooks.sh"
T=$(mktemp -d "${TMPDIR:-/tmp}/install-hooks-test.XXXXXX") || exit 1
T=$(cd "$T" && pwd -P) || exit 1
trap 'rm -rf "$T"' EXIT
pass=0; fail=0
ck() { if [ "$2" = "$3" ]; then echo "  ok   — $1"; pass=$((pass+1)); else echo "  FAIL — $1 (got '$2' want '$3')"; fail=$((fail+1)); fi; }
inst() { local d=$1; shift; OUT=$(cd "$d" && bash "$INSTALL" "$@" 2>&1); RC=$?; }

mkrepo() { # mkrepo DIR WITH_HOOK(yes|no)
	git init -q "$1"; ( cd "$1" && git config user.email t@t && git config user.name t
	  git checkout -q -b dev; mkdir -p scripts/hooks; echo a > a
	  if [ "$2" = yes ]; then printf '#!/bin/sh\nexit 0\n' > scripts/hooks/pre-push; chmod +x scripts/hooks/pre-push; fi
	  git add -A; git commit -qm init )
}
mkrepo "$T/main" yes
M="$T/main"; H="$M/.git/hooks/pre-push"; SRC="$M/scripts/hooks/pre-push"
mkdir -p "$M/.claude/worktrees"
git -C "$M" worktree add -q "$M/.claude/worktrees/wt" -b wt 2>/dev/null
W="$M/.claude/worktrees/wt"

echo "install_hooks:"
rm -f "$H"
inst "$M"
ck "fresh: installs, exit 0" "$RC" 0
ck "fresh: links to the main checkout's copy" "$(readlink "$H")" "$SRC"
inst "$M"
ck "already ours: silent, exit 0" "$RC:$OUT" "0:"

rm -f "$H"
inst "$W"
ck "run from a worktree: exit 0" "$RC" 0
ck "run from a worktree: links to MAIN copy, never the worktree" "$(readlink "$H")" "$SRC"

rm -f "$H"; ln -s "$M/.claude/worktrees/gone/scripts/hooks/pre-push" "$H"
inst "$M"
ck "dangling symlink: replaced, exit 0" "$RC:$(readlink "$H")" "0:$SRC"
ck "dangling symlink: says old -> new" "$(printf '%s' "$OUT" | grep -c 'replaced .*gone/scripts/hooks/pre-push (dangling) -> ')" 1

rm -f "$H"; ln -s "$W/scripts/hooks/pre-push" "$H"
inst "$M"
ck "symlink into a live worktree: replaced, exit 0" "$RC:$(readlink "$H")" "0:$SRC"
ck "symlink into a live worktree: reason named" "$(printf '%s' "$OUT" | grep -c 'points into a worktree')" 1

rm -f "$H"; ln -s "../../.claude/worktrees/wt/scripts/hooks/pre-push" "$H"
inst "$M"
ck "relative symlink into a worktree: replaced" "$RC:$(readlink "$H")" "0:$SRC"

rm -f "$H"; printf '#!/bin/sh\necho mine\n' > "$H"; chmod +x "$H"
inst "$M"
ck "foreign regular file: refused, exit 1" "$RC" 1
ck "foreign regular file: left untouched" "$(cat "$H" | tail -1)" "echo mine"

rm -f "$H"; printf '#!/bin/sh\n' > "$T/elsewhere-hook"; ln -s "$T/elsewhere-hook" "$H"
inst "$M"
ck "foreign live symlink: refused, exit 1" "$RC" 1
ck "foreign live symlink: left untouched" "$(readlink "$H")" "$T/elsewhere-hook"

# The 09-27 shape: the main checkout's tree has no hook yet, a worktree does. The old fallback
# linked the worktree copy; now the installer refuses and creates nothing.
mkrepo "$T/main2" no
M2="$T/main2"; H2="$M2/.git/hooks/pre-push"
mkdir -p "$M2/.claude/worktrees"
git -C "$M2" worktree add -q "$M2/.claude/worktrees/wt" -b wt 2>/dev/null
W2="$M2/.claude/worktrees/wt"; mkdir -p "$W2/scripts/hooks"; printf '#!/bin/sh\n' > "$W2/scripts/hooks/pre-push"
rm -f "$H2"
inst "$W2"
ck "main copy missing, run from worktree: refused, exit 1" "$RC" 1
ck "main copy missing: no hook created" "$( [ -e "$H2" ] || [ -L "$H2" ] && echo created || echo none)" none
ck "main copy missing: refusal names the main checkout" "$(printf '%s' "$OUT" | grep -c "main checkout ($M2) has no scripts/hooks/pre-push")" 1
inst "$M2"
ck "main copy missing, run from main: refused (no dangling link)" "$RC:$( [ -L "$H2" ] && echo created || echo none)" "1:none"

echo "install_hooks --check:"
rm -f "$H"; ln -s "$SRC" "$H"
inst "$M" --check
ck "healthy: silent, exit 0" "$RC:$OUT" "0:"
rm -f "$H"
inst "$M" --check
ck "missing: exit 1, says missing" "$RC:$(printf '%s' "$OUT" | grep -c 'pre-push hook is not installed')" "1:1"
ck "missing: --check changes nothing" "$( [ -L "$H" ] && echo created || echo none)" none
ln -s "$M/.claude/worktrees/gone/scripts/hooks/pre-push" "$H"
inst "$M" --check
ck "dangling: exit 1, says does not resolve" "$RC:$(printf '%s' "$OUT" | grep -c 'pre-push hook does not resolve')" "1:1"
ck "dangling: --check leaves the link alone" "$(readlink "$H")" "$M/.claude/worktrees/gone/scripts/hooks/pre-push"
rm -f "$H"; ln -s "$W/scripts/hooks/pre-push" "$H"
inst "$M" --check
ck "worktree target: exit 1, says points into a worktree" "$RC:$(printf '%s' "$OUT" | grep -c 'points into a worktree')" "1:1"
rm -f "$H"; printf '#!/bin/sh\n' > "$H"
inst "$M" --check
ck "foreign file: exit 1, says not ours" "$RC:$(printf '%s' "$OUT" | grep -c 'is not ours')" "1:1"
inst "$M2" --check
ck "main copy missing: exit 1, says so" "$RC:$(printf '%s' "$OUT" | grep -c 'has no scripts/hooks/pre-push')" "1:1"
inst "$M" --bogus
ck "unknown flag: exit 2" "$RC" 2

echo "  ---- $pass passed, $fail failed"
[ "$fail" -eq 0 ] && [ "$pass" -eq 27 ]
