#!/bin/bash
# test_mission_worktree.sh — mission-worktree.sh never hands back a half-built or dirty worktree.
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
MW="$HERE/mission-worktree.sh"
PASS=0; FAIL=0
ok(){ PASS=$((PASS+1)); echo "  PASS: $1"; }
bad(){ FAIL=$((FAIL+1)); echo "  FAIL: $1 — got: $2"; }

T=$(cd "$(mktemp -d)" && pwd -P); trap 'rm -rf "$T"' EXIT
case "$T" in /tmp/*|/private/tmp/*) echo "SKIP: TMPDIR is under /tmp; this suite needs a non-/tmp scratch dir"; exit 0 ;; esac
R="$T/repo"; mkdir -p "$R"; export MISSION_WORKTREE_STATE="$T/state"
git -C "$R" init -q && echo a > "$R/a" && git -C "$R" add a && git -C "$R" -c user.name=t -c user.email=t@t commit -qm i
SHA=$(git -C "$R" rev-parse HEAD)
waitfor() { local o; for _ in 1 2 3 4 5 6 7 8 9 10; do o=$(cd "$R" && bash "$MW" wait "$1" 5); [ "$o" != pending ] && break; done; echo "$o"; }

echo "== happy path: add returns at once, wait reports ready, tree is clean"
(cd "$R" && bash "$MW" add br1 "$SHA" "$T/wt1") >/dev/null
o=$(waitfor "$T/wt1"); [ "$o" = ready ] && ok "ready" || bad "ready" "$o"
[ -f "$T/wt1/a" ] && [ -z "$(git -C "$T/wt1" status --porcelain)" ] && ok "checked out and clean" || bad "tree" "$(ls "$T/wt1")"

echo "== /tmp is refused"
o=$(cd "$R" && bash "$MW" add br2 "$SHA" /tmp/mw-test-refuse 2>&1); rc=$?
[ "$rc" = 1 ] && case "$o" in *path_under_tmp*) true;; *) false;; esac && ok "/tmp refused" || bad "/tmp" "$o rc=$rc"

echo "== timeout: a checkout that outlives the bound is removed, branch included"
mkdir -p "$T/slowbin"
printf '#!/bin/bash\nif [ "$3" = "worktree" ] && [ "$4" = "add" ]; then sleep 30; fi\nexec %s "$@"\n' "$(command -v git)" > "$T/slowbin/git"
chmod +x "$T/slowbin/git"
(cd "$R" && PATH="$T/slowbin:$PATH" MISSION_WORKTREE_TIMEOUT=3 bash "$MW" add br3 "$SHA" "$T/wt3") >/dev/null
o=$(waitfor "$T/wt3"); case "$o" in failed:timeout_3s) ok "timeout reported";; *) bad "timeout" "$o";; esac
[ ! -e "$T/wt3" ] && ok "partial worktree removed" || bad "partial worktree left" "$(ls "$T/wt3")"
git -C "$R" rev-parse --verify -q br3 >/dev/null && bad "branch left behind" "br3" || ok "branch removed"

echo "== a tree that is not clean after checkout is rejected"
mkdir -p "$T/hooks"; printf '#!/bin/bash\necho junk > stray.txt\n' > "$T/hooks/post-checkout"; chmod +x "$T/hooks/post-checkout"
git -C "$R" config core.hooksPath "$T/hooks"
(cd "$R" && bash "$MW" add br4 "$SHA" "$T/wt4") >/dev/null
o=$(waitfor "$T/wt4"); case "$o" in failed:not_clean_after_add*) ok "dirty tree rejected";; *) bad "dirty" "$o";; esac
[ ! -e "$T/wt4" ] && ok "dirty worktree removed" || bad "dirty worktree left" ""
git -C "$R" config --unset core.hooksPath

echo "== wait on an unknown path fails rather than hanging"
o=$(cd "$R" && bash "$MW" wait "$T/nope" 1); [ "$o" = failed:unknown_worktree ] && ok "unknown worktree" || bad "unknown" "$o"

echo "==== $PASS passed, $FAIL failed ===="
[ "$FAIL" -eq 0 ]
