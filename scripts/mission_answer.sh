#!/usr/bin/env bash
#
# mission_answer.sh — record an ATTENDED human ruling directly into a mission's
# decision ledger, without a round trip through the bookkeeping issue.
#
# WHY THIS EXISTS. Until now a decision could only be answered by commenting on a
# PUBLIC bookkeeping issue as an allowlisted author, because that comment was the
# only thing carrying human provenance the loop could check. That made the fast
# path — Mark is at the terminal, in the repo, looking at the ask — the slow one:
# he had to leave the session, find the week's issue, and wait a whole fire for
# the loop to read it back. This script is the attended channel, and it carries
# the SAME trust root the issue channel does: an identity the fleet bot does not
# have and is forbidden to author with.
#
# THE PROVENANCE CONTRACT. An attended resolution is trustworthy only if the loop
# could not have written it itself. The fleet account (`sunholo-voight-kampff`)
# authors every commit the loop makes; this script refuses to run under that
# identity and stamps the row with the attended one. The loop's skill forbids it
# from ever authoring with an attended identity, exactly as mission_directives.sh
# forbids the fleet account from being a directive principal. Both are conventions
# enforced in code at the point of writing and auditable afterwards in `git log`.
#
# WHAT IT DOES NOT DO. It does not push, and it does not touch anything but the
# one ledger row (and, with --commit, the commit that carries it). Code, gates and
# benchmark curation still route through the loop — see the charter guardrail on
# attended side-sessions.
#
# Usage:
#   scripts/mission_answer.sh --id D-51 --answer "TEXT" [--evidence "TEXT"]
#   scripts/mission_answer.sh --id D-51 --answer-file ruling.txt [--evidence-file ev.txt]
#                             [--file design_docs/v1-mission.md] [--commit] [--dry-run]
#
# USAGE TRAPS, all three measured 2026-09-02 on the D-53 ruling:
#   * THE ANSWER MUST BE ONE LINE. A ledger row is a single markdown table row, so a blank line
#     or newline in --answer-file shatters it into fragments with empty fields. The validator
#     catches this and refuses to commit ("empty decision/evidence field for D-nn") — believe it,
#     collapse the file to one line, and re-run. It is not a false alarm.
#   * DO NOT START THE ANSWER WITH "ANSWERED —". This script prepends its own, and you get
#     "ANSWERED — ANSWERED —" in the permanent record.
#   * WHO MAY RUN IT: an attended session, including one where Mark directs an agent and states
#     the ruling. What the contract forbids is the UNATTENDED loop resolving a row on its own
#     behalf. Do not refuse a ruling Mark has actually given — see CLAUDE.md principle 4.
#
# KNOWN WEAKNESS, recorded rather than papered over: ATT_NAME/ATT_EMAIL below are DEFAULTS, not
# derived from whoever invokes the script, so it stamps the attended identity for any caller and
# the identity guard only catches an explicit override to the fleet account. The real control is
# the charter rule, not this code. Tightening it (an explicit --as, or checking the invoking
# identity) is an open design question, not a lint.
#
# Exit codes: 0 = row rewritten (and validated), 1 = refused/error.
set -uo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RESET='\033[0m'
die() { echo -e "${RED}✗ mission_answer: $*${RESET}" >&2; exit 1; }
note() { echo -e "${YELLOW}• $*${RESET}"; }
ok() { echo -e "${GREEN}✓ $*${RESET}"; }

ID=""; ANSWER=""; EVIDENCE=""; DOC="${MISSION_DOC:-design_docs/v1-mission.md}"
COMMIT=0; DRYRUN=0
while [ $# -gt 0 ]; do
	case "$1" in
		--id)       ID="${2:-}";       shift 2 ;;
		--answer)   ANSWER="${2:-}";   shift 2 ;;
		--evidence) EVIDENCE="${2:-}"; shift 2 ;;
		# File forms: the ruling prose is long and full of quotes, backticks and
		# pipes, so passing it as a path beats quoting it through a shell.
		--answer-file)   ANSWER="$(cat "${2:-}")"   || die "unreadable --answer-file: ${2:-}"; shift 2 ;;
		--evidence-file) EVIDENCE="$(cat "${2:-}")" || die "unreadable --evidence-file: ${2:-}"; shift 2 ;;
		--file)     DOC="${2:-}";      shift 2 ;;
		--commit)   COMMIT=1;          shift ;;
		--dry-run)  DRYRUN=1;          shift ;;
		-h|--help)  sed -n '2,50p' "$0"; exit 0 ;;
		*) die "unknown argument: $1" ;;
	esac
done

[ -n "$ID" ]     || die "--id is required (e.g. --id D-51)"
[ -n "$ANSWER" ] || die "--answer is required — the ruling, in your words"
[ -w "$DOC" ]    || die "unwritable doc: $DOC"
case "$ID" in D-*) ;; *) die "--id must look like D-… , got: $ID" ;; esac

# THE ATTENDED IDENTITY. Overridable for another human, but never the fleet bot:
# Uses a GitHub NOREPLY address on purpose (Mark, 2026-09-02): it is attributable to the
# account, and it keeps a personal address out of public commits, charters, and the issue
# comments the loop pastes ledger rows into verbatim. Never put a real address here.
# a row resolved under the loop's own identity is self-direction, which is the one
# failure this whole channel has to make impossible.
ATT_NAME="${MISSION_ATTENDED_NAME:-Mark Edmondson}"
ATT_EMAIL="${MISSION_ATTENDED_EMAIL:-3155884+MarkEdmondson1234@users.noreply.github.com}"
FLEET_PATTERN="${MISSION_FLEET_ACCOUNT:-sunholo-voight-kampff}"
case "$ATT_EMAIL" in *"$FLEET_PATTERN"*) die "attended identity is the fleet bot ($ATT_EMAIL) — refusing; an attended ruling must not be authored by the loop" ;; esac
case "$ATT_NAME" in *[Bb]ot*) die "attended identity looks like a bot ($ATT_NAME) — refusing" ;; esac

TODAY="$(date '+%Y-%m-%d')"
TMP="$(mktemp)" || die "mktemp failed"
trap 'rm -f "$TMP"' EXIT

# Rewrite exactly one row. Escaped pipes (\|) appear inside cell prose, so they are
# protected before the split and restored after — splitting naively on "|" would
# silently corrupt any row containing one.
ID="$ID" ANSWER="$ANSWER" EVIDENCE="$EVIDENCE" TODAY="$TODAY" \
ATT_NAME="$ATT_NAME" ATT_EMAIL="$ATT_EMAIL" \
awk '
  BEGIN { id=ENVIRON["ID"]; hits=0; inside=0 }
  /<!-- decision-ledger:start -->/ { inside=1 }
  /<!-- decision-ledger:end -->/   { inside=0 }
  {
    if (inside && $0 ~ ("^\\|[[:space:]]*" id "[[:space:]]*\\|")) {
      line=$0
      gsub(/\\\|/, "\001", line)
      n=split(line, c, "|")
      if (n != 6) {
        printf "mission_answer: row %s has %d fields, expected 6 — refusing to edit a row I cannot parse\n", id, n > "/dev/stderr"
        exit 3
      }
      status=c[3]; gsub(/^[[:space:]]+|[[:space:]]+$/, "", status)
      if (status != "OPEN") {
        printf "mission_answer: row %s is %s, not OPEN — refusing (a resolved row is never re-answered; supersede it with a new ID instead)\n", id, status > "/dev/stderr"
        exit 4
      }
      c[3]=" RESOLVED "
      c[4]=c[4] " **ANSWERED — " ENVIRON["ANSWER"] "** (" ENVIRON["ATT_NAME"] ", attended " ENVIRON["TODAY"] ", recorded directly in this ledger.)"
      c[5]=c[5] " **Attended ruling " ENVIRON["TODAY"] "** — recorded in-session under the ATTENDED LEDGER EDITS contract, not via the bookkeeping issue; provenance is the commit author `" ENVIRON["ATT_EMAIL"] "`, which the fleet bot does not hold and the loop may not author with."
      if (ENVIRON["EVIDENCE"] != "") c[5]=c[5] " " ENVIRON["EVIDENCE"]
      out=c[1]
      for (i=2; i<=n; i++) out=out "|" c[i]
      gsub(/\001/, "\\|", out)
      print out
      hits++
      next
    }
    print
  }
  END {
    if (hits == 0) { printf "mission_answer: no OPEN row %s inside the decision-ledger block\n", id > "/dev/stderr"; exit 5 }
    if (hits > 1)  { printf "mission_answer: %d rows match %s — the ledger has a duplicate ID\n", hits, id > "/dev/stderr"; exit 6 }
  }
' "$DOC" > "$TMP"
rc=$?
[ "$rc" -eq 0 ] || exit 1

if [ "$DRYRUN" -eq 1 ]; then
	diff -u "$DOC" "$TMP" | head -40
	note "--dry-run: $DOC not modified"
	exit 0
fi

cat "$TMP" > "$DOC" || die "write failed: $DOC"
ok "$ID → RESOLVED in $DOC"

# The ledger must still validate as a whole — a rewrite that breaks the table is
# worse than an unanswered row, because --open would stop listing real asks.
if [ -x scripts/mission_decisions.sh ]; then
	scripts/mission_decisions.sh --check --file "$DOC" || die "ledger no longer validates after the edit — revert with: git checkout -- $DOC"
fi

if [ "$COMMIT" -eq 1 ]; then
	git -c user.name="$ATT_NAME" -c user.email="$ATT_EMAIL" \
		commit -q -m "docs(mission): attended ruling — $ID resolved

Recorded directly in the decision ledger from an attended session under the
ATTENDED LEDGER EDITS contract. Authored with the attended identity, which the
fleet bot does not hold: the loop may not author a resolution on its own behalf." \
		-- "$DOC" || die "commit failed"
	ok "committed as $ATT_NAME <$ATT_EMAIL>"
fi
