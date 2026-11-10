#!/bin/bash
# install_hooks.sh — link versioned git hooks into this clone's hooks dir.
#
#   install_hooks.sh           install / repair; quiet when already installed (SessionStart-safe)
#   install_hooks.sh --check   read-only: one warning line per managed hook that is not ours and
#                              resolving; exit 1 if any (git_health.sh surfaces this)
#
# Hooks always link to the MAIN checkout's copy (dirname of the common git dir), never to a
# worktree's: .git/hooks is shared by every worktree, and a link into a worktree dangles when that
# worktree is removed. If the main checkout has no copy yet, refuse rather than fall back to this
# worktree's — that fallback is how the 2026-09-27 link into .claude/worktrees/fleet-rulings
# was made (m-v1-release-gates §5). A dangling link, or one into a worktree, is replaced (old -> new
# printed). A regular file or a live link elsewhere is someone else's hook: refused, exit 1.
set -u
mode=install
case "${1-}" in
	'') ;;
	--check) mode=check ;;
	*) echo "usage: install_hooks.sh [--check]" >&2; exit 2 ;;
esac
hooks="pre-push"   # the managed hooks; each lives at scripts/hooks/<name> in the main checkout

root=$(git rev-parse --show-toplevel 2>/dev/null) || { echo "install_hooks: not in a git repo" >&2; exit 1; }
hooks_dir=$(git config --get core.hooksPath 2>/dev/null)
[ -n "$hooks_dir" ] || hooks_dir="$(git rev-parse --git-common-dir)/hooks"
case "$hooks_dir" in /*) ;; *) hooks_dir="$root/$hooks_dir" ;; esac
common=$(cd "$(git rev-parse --git-common-dir)" && pwd)
main_root=$(dirname "$common")

# into_worktree TARGET — the link target names a linked worktree (absolute or relative).
into_worktree() {
	case "$1" in
		*/.claude/worktrees/*|.claude/worktrees/*|*/.git/worktrees/*|.git/worktrees/*) return 0 ;;
	esac
	return 1
}

# state NAME — prints one of: ours, missing, dangling, worktree, foreign, nosource.
# nosource wins over every repairable state: with no main copy there is nothing safe to link.
state() {
	local src="$main_root/scripts/hooks/$1" dst="$hooks_dir/$1" st
	if [ -L "$dst" ]; then
		local tgt; tgt=$(readlink "$dst")
		if [ ! -e "$dst" ]; then st=dangling
		elif [ "$tgt" = "$src" ]; then st=ours
		elif into_worktree "$tgt"; then st=worktree
		else echo foreign; return
		fi
	elif [ -e "$dst" ]; then echo foreign; return
	else st=missing
	fi
	if [ ! -f "$src" ]; then echo nosource; return; fi
	echo "$st"
}

rc=0
for h in $hooks; do
	src="$main_root/scripts/hooks/$h"; dst="$hooks_dir/$h"; st=$(state "$h")
	if [ "$mode" = check ]; then
		case "$st" in
			ours) continue ;;
			missing)  echo "⚠️  GIT: $h hook is not installed — the stdlib and ci-quick gates are not running on push. make install-hooks" ;;
			dangling) echo "⚠️  GIT: $h hook does not resolve (-> $(readlink "$dst")) — the stdlib and ci-quick gates are not running on push. make install-hooks" ;;
			worktree) echo "⚠️  GIT: $h hook points into a worktree (-> $(readlink "$dst")); it stops running when that worktree is removed. make install-hooks" ;;
			foreign)  echo "⚠️  GIT: $dst is not ours — the stdlib and ci-quick gates are not running on push. Merge it with scripts/hooks/$h by hand." ;;
			nosource) echo "⚠️  GIT: the main checkout ($main_root) has no scripts/hooks/$h — the $h gate is not installed; bring the main checkout up to date, then make install-hooks." ;;
		esac
		rc=1; continue
	fi
	case "$st" in
		ours) ;;
		missing)
			mkdir -p "$hooks_dir" || exit 1
			ln -s "$src" "$dst" && echo "install_hooks: $h -> $src" || rc=1 ;;
		dangling|worktree)
			old=$(readlink "$dst")
			why="dangling"; [ "$st" = worktree ] && why="points into a worktree"
			# ln -sf replaces the link itself (dst is a symlink to a file, never a directory).
			ln -sf "$src" "$dst" && echo "install_hooks: replaced $h: $old ($why) -> $src" || rc=1 ;;
		foreign)
			echo "install_hooks: $dst exists and is not ours — leaving it; merge by hand." >&2; rc=1 ;;
		nosource)
			echo "install_hooks: the main checkout ($main_root) has no scripts/hooks/$h — refusing to link a worktree copy (it would dangle when the worktree is removed). Update the main checkout, then make install-hooks." >&2
			rc=1 ;;
	esac
done
exit $rc
