#!/bin/bash
# Review-date gate for the published docs. Every page in scope carries two
# frontmatter keys:
#
#   reviewed: 2026-10-05     # when someone last read the page against the code
#   reviewBy: 2027-01-05     # when it must be read again, or `never` for a
#                            # point-in-time record (a talk, a retro)
#
# `reviewed` is optional: a page dated by the 2026-10-05 rollout has only reviewBy,
# because nobody had read it against the code yet and a date there would claim so.
#
# Two classes of finding, and they are treated differently on purpose:
#
#   WARN  (rc=0) — the page is OVERDUE, or carries no reviewBy at all. Both arrive by
#                  the calendar or by an old page, never by the PR under test, so they
#                  must not turn an unrelated PR red. They surface as ::warning::
#                  annotations under --github and in the docs mission's backlog.
#   ERROR (rc=1) — a date that does not parse, or reviewed after reviewBy. Those are
#                  typos made by the PR that wrote them, so the PR is the place to fail.
#
# rc=2 is an instrument failure: fewer pages enumerated than the floor, which means the
# scope globs have stopped matching (a renamed docs tree), not that the docs are clean.
#
# SCOPE: tracked .md/.mdx under docs/docs/ plus the top-level docs/ pages listed in
# EXTRA, minus the pages a sync script regenerates (a date added by hand there is wiped
# on the next sync). design_docs/ is out of scope: those are decision records.
#
#   bash scripts/check_doc_review.sh                 # report
#   bash scripts/check_doc_review.sh --github        # + ::warning:: / ::error:: lines
#   bash scripts/check_doc_review.sh --list-overdue  # paths only, oldest due first
#
# DOC_REVIEW_TODAY=YYYY-MM-DD overrides today (the self-test uses it).
set -u

REPO_ROOT="${DOC_REVIEW_REPO_ROOT:-$(pwd)}"
TODAY="${DOC_REVIEW_TODAY:-$(date +%Y-%m-%d)}"
MINIMUM="${DOC_REVIEW_MINIMUM:-50}"
GITHUB=0
LIST_OVERDUE=0

EXTRA="docs/LIMITATIONS.md docs/TESTING.md docs/VISION.md"

for arg in "$@"; do
	case "$arg" in
		--github) GITHUB=1 ;;
		--list-overdue) LIST_OVERDUE=1 ;;
		*) echo "usage: $0 [--github] [--list-overdue]" >&2; exit 2 ;;
	esac
done

cd "$REPO_ROOT" || exit 2

# Pages a script under docs/scripts/ or the Makefile regenerates.
generated() {
	case "$1" in
		docs/docs/design-docs.md) return 0 ;;            # sync-design-docs.sh
		docs/docs/reference/env-vars.md) return 0 ;;     # make docs-env
		docs/docs/reference/cli.md) return 0 ;;          # make docs-cli (byte-compared by check-cli-docs)
		docs/docs/prompts/index.md) return 1 ;;          # hand-written landing page
		docs/docs/prompts/*) return 0 ;;                 # sync-prompts.sh, sync-active-prompt.sh
		docs/docs/packages/*/*) return 0 ;;              # sync-registry.sh (per-vendor dirs)
	esac
	return 1
}

valid_date() {
	echo "$1" | grep -Eq '^[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])$'
}

# Value of a top-level frontmatter key, quotes stripped; empty if absent or no frontmatter.
fm_value() {
	awk -v key="$2" '
		NR == 1 { if ($0 != "---") exit; next }
		$0 == "---" { exit }
		index($0, key ":") == 1 {
			v = substr($0, length(key) + 2)
			sub(/^[ \t]+/, "", v); sub(/[ \t]+$/, "", v)
			sub(/^["\x27]/, "", v); sub(/["\x27]$/, "", v)
			print v; exit
		}
	' "$1"
}

pages=$( { git ls-files -- 'docs/docs/*.md' 'docs/docs/*.mdx'; for f in $EXTRA; do git ls-files -- "$f"; done; } | sort -u)

scanned=0; overdue=0; missing=0; errors=0
overdue_list=""

warn() { [ "$GITHUB" -eq 1 ] && echo "::warning file=$1::$2"; echo "  WARN  $1: $2"; }
err()  { [ "$LIST_OVERDUE" -eq 1 ] && return; [ "$GITHUB" -eq 1 ] && echo "::error file=$1::$2";   echo "  ERROR $1: $2"; }

for page in $pages; do
	generated "$page" && continue
	scanned=$((scanned + 1))
	reviewed=$(fm_value "$page" reviewed)
	review_by=$(fm_value "$page" reviewBy)

	if [ -z "$review_by" ]; then
		missing=$((missing + 1))
		[ "$LIST_OVERDUE" -eq 0 ] && warn "$page" "no reviewBy in frontmatter"
		continue
	fi
	if [ -n "$reviewed" ] && ! valid_date "$reviewed"; then
		errors=$((errors + 1)); err "$page" "reviewed '$reviewed' is not YYYY-MM-DD"
	fi
	[ "$review_by" = "never" ] && continue
	if ! valid_date "$review_by"; then
		errors=$((errors + 1)); err "$page" "reviewBy '$review_by' is not YYYY-MM-DD or never"
		continue
	fi
	if [ -n "$reviewed" ] && valid_date "$reviewed" && [[ "$reviewed" > "$review_by" ]]; then
		errors=$((errors + 1)); err "$page" "reviewed $reviewed is after reviewBy $review_by"
		continue
	fi
	if [[ "$review_by" < "$TODAY" ]]; then
		overdue=$((overdue + 1))
		overdue_list="${overdue_list}${review_by} ${page}
"
		[ "$LIST_OVERDUE" -eq 0 ] && warn "$page" "review was due $review_by (last reviewed ${reviewed:-never})"
	fi
done

if [ "$scanned" -lt "$MINIMUM" ]; then
	echo "instrument failure: enumerated $scanned pages (floor $MINIMUM); the scope globs no longer match the docs tree" >&2
	exit 2
fi

if [ "$LIST_OVERDUE" -eq 1 ]; then
	printf '%s' "$overdue_list" | sort | awk '{print $2}'
	exit 0
fi

echo "doc review: $scanned pages, $overdue overdue, $missing without reviewBy, $errors malformed (today $TODAY)"
[ "$errors" -eq 0 ] || exit 1
exit 0
