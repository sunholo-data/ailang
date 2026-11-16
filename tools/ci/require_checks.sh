#!/bin/bash
# require_checks.sh — refuse a release unless the commit's required CI check-runs are green.
#
#   tools/ci/require_checks.sh [--tag TAG] [--repo OWNER/NAME] <sha> [check-name...]
#
# Names default to tools/ci/release_required_checks.txt. Polls
#   GET /repos/{repo}/commits/{sha}/check-runs?filter=latest&per_page=100&page=N  (all pages)
# and requires conclusion == success for EVERY GitHub Actions check-run carrying each name
# (build.yml emits two `Build macos-latest` runs; taking the first one would miss a red second).
# A name with no run yet, or a run still in progress, is pending: poll again. Any completed run
# that is not `success` (failure, cancelled, timed_out, skipped, ...) is red: stop at once.
# Pending past the last poll is red too (fail closed: a renamed job never turns green).
#
# Override (§1.3 of m-v1-release-gates): an ANNOTATED tag whose message has a line
#   release-gate-override: https://github.com/<owner>/<repo>/issues/<n> <reason>
# waives the checks that are red. It does not waive pending or missing ones. A lightweight tag is
# never read for the marker (its "message" is the commit message, which anyone landing a commit
# controls). A malformed marker is an error, not a no-op. Under GitHub Actions the marker and the
# waived names are written to $GITHUB_OUTPUT as `override` and `waived` for the release notes.
#
# A known flake is handled by re-running the failed job (the API then reports the latest attempt),
# not by the override.
#
# Env: REQUIRE_CHECKS_INTERVAL (s, default 15), REQUIRE_CHECKS_ATTEMPTS (default 360 = 90 min;
#      build.yml's per-OS jobs, which the list requires, have taken 45-77 min on a busy queue).
# Exit: 0 releasable, 1 not releasable / bad marker / timeout, 2 usage.
# bash 3.2 (runs on the rig for the runbook): no assoc arrays, no ${v,,}.
set -u

here=$(cd "$(dirname "$0")" && pwd)
list="$here/release_required_checks.txt"
tag=""; repo="${GITHUB_REPOSITORY:-sunholo-data/ailang}"
interval="${REQUIRE_CHECKS_INTERVAL:-15}"; attempts="${REQUIRE_CHECKS_ATTEMPTS:-360}"
usage() { echo "usage: require_checks.sh [--tag TAG] [--repo OWNER/NAME] <sha> [check-name...]" >&2; exit 2; }
while [ $# -gt 0 ]; do
	case "$1" in
		--tag) [ $# -ge 2 ] || usage; tag=$2; shift 2 ;;
		--repo) [ $# -ge 2 ] || usage; repo=$2; shift 2 ;;
		--) shift; break ;;
		-*) usage ;;
		*) break ;;
	esac
done
[ $# -ge 1 ] || usage
sha=$1; shift
printf '%s' "$sha" | grep -Eq '^[0-9a-f]{7,40}$' || { echo "require_checks: '$sha' is not a commit SHA" >&2; exit 2; }
case "$interval$attempts" in *[!0-9]*) usage ;; esac
[ "$attempts" -ge 1 ] || usage

# Required names, newline-separated (names contain spaces and parentheses).
if [ $# -gt 0 ]; then
	names=$(printf '%s\n' "$@")
else
	[ -f "$list" ] || { echo "require_checks: no names given and $list is missing" >&2; exit 2; }
	names=$(sed -e 's/[[:space:]]*#.*$//' -e 's/[[:space:]]*$//' "$list" | grep -v '^$')
fi
[ -n "$names" ] || { echo "require_checks: the required-check list is empty — refusing to pass vacuously" >&2; exit 2; }

annotate() { [ "${GITHUB_ACTIONS:-}" = true ] && echo "::$1::$2"; return 0; }
label="$sha"; [ -n "$tag" ] && label="$tag"
short=$(printf '%s' "$sha" | cut -c1-9)

# ---- override marker (annotated tags only) ----
override=""
if [ -n "$tag" ]; then
	ttype=$(git cat-file -t "refs/tags/$tag" 2>/dev/null)
	tcommit=$(git rev-parse -q --verify "refs/tags/$tag^{commit}" 2>/dev/null)
	if [ -z "$tcommit" ]; then
		echo "release gate: tag $tag is not in this clone — fetch it (git fetch origin +refs/tags/$tag:refs/tags/$tag)" >&2; exit 1
	fi
	case "$tcommit" in "$sha"*) ;; *)
		echo "release gate: tag $tag points at $tcommit, not $sha — refusing to judge one commit by another's checks" >&2; exit 1 ;;
	esac
	if [ "$ttype" = tag ]; then
		marker=$(git for-each-ref "refs/tags/$tag" --format='%(contents)' | grep -E '^release-gate-override:' | head -1)
		if [ -n "$marker" ]; then
			if ! printf '%s\n' "$marker" | grep -Eq '^release-gate-override: https://github\.com/[^/ ]+/[^/ ]+/(issues|pull)/[0-9]+ +[^ ].*$'; then
				echo "release gate: malformed override in tag $tag: '$marker'" >&2
				echo "  expected: release-gate-override: https://github.com/<owner>/<repo>/issues/<n> <reason>" >&2
				exit 1
			fi
			override=${marker#release-gate-override: }
			echo "release gate: tag $tag carries an override: $override"
		fi
	elif git for-each-ref "refs/tags/$tag" --format='%(contents)' | grep -Eq '^release-gate-override:'; then
		echo "release gate: $tag is a lightweight tag; its commit message's override line is ignored (overrides are read from annotated tags only)"
	fi
fi

# ---- one poll: fetch every page, print "<state>\t<name>\t<detail>" per required name ----
poll() {
	local page=1 pages="" body total got=0 n
	while :; do
		body=$(gh api "repos/$repo/commits/$sha/check-runs?filter=latest&per_page=100&page=$page" 2>/dev/null) || return 1
		pages="$pages$body"
		n=$(printf '%s' "$body" | jq '.check_runs | length') || return 1
		total=$(printf '%s' "$body" | jq '.total_count // 0') || return 1
		got=$((got + n))
		[ "$n" -lt 100 ] && break
		[ "$got" -ge "$total" ] && break
		page=$((page + 1))
		[ "$page" -le 20 ] || break
	done
	printf '%s\n' "$names" | while IFS= read -r name; do
		printf '%s' "$pages" | jq -rs --arg n "$name" '
			[.[].check_runs[] | select(.name == $n and (.app.slug // "") == "github-actions")] as $rs
			| ($rs | map(select(.status == "completed" and .conclusion != "success"))) as $bad
			| if ($rs | length) == 0 then "missing\t\($n)\tno check-run with this name"
			  elif ($bad | length) > 0 then
			    "red\t\($n)\t" + ($bad | map("\(.conclusion) (\(.html_url // "no url"))") | join("; "))
			  elif any($rs[]; .status != "completed") then
			    "pending\t\($n)\t" + ($rs | map(.status) | join(","))
			  else "green\t\($n)\t\($rs | length) run(s)"
			  end'
	done
}

i=0; table=""
while [ "$i" -lt "$attempts" ]; do
	i=$((i + 1))
	if ! table=$(poll) || [ -z "$table" ]; then
		echo "  [$i/$attempts] check-runs API unreadable for $sha; retrying"
		table=""
		[ "$i" -lt "$attempts" ] && sleep "$interval"
		continue
	fi
	red=$(printf '%s\n' "$table" | awk -F'\t' '$1=="red"')
	waiting=$(printf '%s\n' "$table" | awk -F'\t' '$1=="pending"||$1=="missing"')
	if [ -n "$red" ] && [ -z "$override" ]; then break; fi
	[ -z "$waiting" ] && break
	echo "  [$i/$attempts] waiting on: $(printf '%s\n' "$waiting" | awk -F'\t' '{printf "%s%s (%s)", sep, $2, $1; sep=", "}')"
	[ "$i" -lt "$attempts" ] && sleep "$interval"
done

echo "release gate: $label ($short) — required checks:"
printf '%s\n' "$table" | awk -F'\t' 'NF { printf "  %-8s %s — %s\n", $1, $2, $3 }'
red=$(printf '%s\n' "$table" | awk -F'\t' '$1=="red"')
waiting=$(printf '%s\n' "$table" | awk -F'\t' '$1=="pending"||$1=="missing"')

if [ -z "$table" ]; then
	echo "release gate: $label ($short) is not releasable — the check-runs API was unreadable for $attempts poll(s)." >&2
	annotate error "release gate: check-runs API unreadable for $sha"
	exit 1
fi
if [ -n "$red" ] && [ -z "$override" ]; then
	why=$(printf '%s\n' "$red" | awk -F'\t' '{printf "%s%s: %s", sep, $2, $3; sep="; "}')
	msg="release gate: $label ($short) is not releasable — $why. Re-run the failed job if it is a quarantined flake; otherwise fix and re-tag. Override: release-gate-override marker in an annotated tag (m-v1-release-gates §1.3)."
	echo "$msg" >&2; annotate error "$msg"
	exit 1
fi
if [ -n "$waiting" ]; then
	why=$(printf '%s\n' "$waiting" | awk -F'\t' '{printf "%s%s (%s)", sep, $2, $1; sep=", "}')
	msg="release gate: $label ($short) is not releasable — still not green after $attempts poll(s): $why. A missing name may be a renamed job: check tools/ci/release_required_checks.txt."
	echo "$msg" >&2; annotate error "$msg"
	exit 1
fi
if [ -n "$red" ]; then
	waived=$(printf '%s\n' "$red" | awk -F'\t' '{printf "%s%s", sep, $2; sep=", "}')
	msg="release gate: $label ($short) released with a CI-gate OVERRIDE ($override) — waived red checks: $waived"
	echo "$msg"; annotate warning "$msg"
	if [ -n "${GITHUB_OUTPUT:-}" ]; then
		{ echo "override=$override"; echo "waived=$waived"; } >> "$GITHUB_OUTPUT"
	fi
	exit 0
fi
echo "release gate: $label ($short) is releasable — every required check is green."
exit 0
