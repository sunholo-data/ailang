#!/bin/bash
# simplicity_external.sh — the external-service rows of the simplicity audit,
# printed as ONE JSON object of metric entries that tools/simplicity_metrics.sh
# merges into its snapshot. Run on its own to see just these rows.
#
# Every row here depends on something outside the repo (SonarCloud, the GitHub
# API, gcloud), so every row is OPTIONAL and OFFLINE-SAFE: when the service is
# unreachable or the credential is missing, the row records `value: null` with a
# `reason`. It never records 0 for "could not look" (0 is a real measurement:
# "the API answered and there are none"), and it never fails the run.
#
# Usage:
#   tools/simplicity_external.sh               # Sonar + GitHub rows
#   tools/simplicity_external.sh --offline     # every row null, reason "skipped: --offline"
#   tools/simplicity_external.sh --services    # also the Cloud Run used-or-not check
#                                              #   (gcloud, read-only, SLOW: minutes)
#
# bash 3.2 / BSD grep compatible. No `timeout` on macOS: with_deadline below.

set -euo pipefail

REPO_SLUG="sunholo-data/ailang"
SONAR_KEY="sunholo-data_ailang"
SONAR_API="https://sonarcloud.io/api"
GH_DEADLINE=90
# The Cloud Run planes (prod, dev, test). Read-only calls only: `run services
# list`, `logging read`. The identity is gcloud's own: its active account, or
# CLOUDSDK_CORE_ACCOUNT=<account> to pick one. No address is written here:
# this file is loop-written surface (make check-no-personal-email).
GCLOUD_PROJECTS="ailang-multivac ailang-multivac-dev ailang-multivac-test"
# Per-service ceiling on log entries read. A service at the ceiling is reported
# truncated:true, so a capped count is never mistaken for the whole month.
SERVICES_LOG_CAP=250000

OFFLINE=0
SERVICES=0
while [ $# -gt 0 ]; do
  case "$1" in
    --offline) OFFLINE=1 ;;
    --services) SERVICES=1 ;;
    -h|--help) sed -n '2,18p' "$0"; exit 0 ;;
    *) echo "unknown flag: $1" >&2; exit 64 ;;
  esac
  shift
done

command -v jq >/dev/null || { echo "jq is required" >&2; exit 69; }

TMPD="$(mktemp -d -t simplicity-ext.XXXXXX)"
trap 'rm -rf "$TMPD"' EXIT

METRICS='{}'
# put KEY VALUE_JSON DIR HOW [DETAIL] [REASON]
put() {
  if [ "$2" = "null" ] && [ -z "${6:-}" ]; then
    echo "simplicity_external: BUG: $1 recorded null without a reason" >&2
    exit 70
  fi
  METRICS="$(jq -c --arg k "$1" --argjson v "$2" --arg dir "$3" --arg how "$4" \
    --arg detail "${5:-}" --arg reason "${6:-}" \
    '. + {($k): ({value: $v, gate: null, dir: $dir, how: $how}
        + (if $detail != "" then {detail: $detail} else {} end)
        + (if $reason != "" then {reason: $reason} else {} end))}' <<<"$METRICS")"
}
unmeasured() { put "$1" null "$2" "$3" "" "$4"; }  # KEY DIR HOW REASON

# First line of a captured stderr, one line, bounded — goes into `reason`.
first_err() { { head -1 "$1" 2>/dev/null || true; } | tr -d '\r' | cut -c1-160; }

# with_deadline SECS CMD... — run CMD, TERM it after SECS. rc 143 on expiry.
with_deadline() {
  local secs="$1"; shift
  "$@" &
  local pid=$!
  ( sleep "$secs"; kill -TERM "$pid" 2>/dev/null ) </dev/null >/dev/null 2>&1 &
  local watcher=$!
  local rc=0
  wait "$pid" || rc=$?
  kill "$watcher" 2>/dev/null || true
  wait "$watcher" 2>/dev/null || true
  return "$rc"
}

# ---- SonarCloud (public API, no token) ------------------------------------
SONAR_GATE_HOW="SonarCloud quality gate: $SONAR_API/qualitygates/project_status?projectKey=$SONAR_KEY (1 = OK, 0 = ERROR/WARN)"
SONAR_RATING_HOW="SonarCloud overall rating from $SONAR_API/measures/component (1=A .. 5=E)"
sonar_rows_unmeasured() {
  unmeasured sonar_gate_passing ge "$SONAR_GATE_HOW" "$1"
  unmeasured sonar_security_rating le "$SONAR_RATING_HOW: security_rating" "$1"
  unmeasured sonar_reliability_rating le "$SONAR_RATING_HOW: reliability_rating" "$1"
  unmeasured sonar_maintainability_rating le "$SONAR_RATING_HOW: sqale_rating" "$1"
}

sonar_get() {  # URL -> stdout body; nonzero + stderr on failure
  curl -sS --fail --connect-timeout 5 --max-time 20 "$1"
}

probe_sonar() {
  if [ "$OFFLINE" -eq 1 ]; then sonar_rows_unmeasured "skipped: --offline"; return; fi
  if ! command -v curl >/dev/null; then sonar_rows_unmeasured "curl not installed"; return; fi
  local gate measures status
  if ! gate="$(sonar_get "$SONAR_API/qualitygates/project_status?projectKey=$SONAR_KEY" 2>"$TMPD/sonar.err")"; then
    sonar_rows_unmeasured "sonarcloud unreachable: $(first_err "$TMPD/sonar.err")"
    return
  fi
  status="$(jq -r '.projectStatus.status // empty' <<<"$gate" 2>/dev/null || true)"
  case "$status" in
    OK) put sonar_gate_passing 1 ge "$SONAR_GATE_HOW" "OK" ;;
    ERROR|WARN)
      put sonar_gate_passing 0 ge "$SONAR_GATE_HOW" \
        "$status: $(jq -r '[.projectStatus.conditions[]? | select(.status != "OK") | "\(.metricKey) \(.actualValue) (\(.comparator) \(.errorThreshold))"] | join("; ")' <<<"$gate")" ;;
    *) unmeasured sonar_gate_passing ge "$SONAR_GATE_HOW" "unexpected response: no projectStatus.status" ;;
  esac

  if ! measures="$(sonar_get "$SONAR_API/measures/component?component=$SONAR_KEY&metricKeys=security_rating,reliability_rating,sqale_rating,vulnerabilities,bugs,code_smells" 2>"$TMPD/sonar.err")"; then
    local r
    r="sonarcloud unreachable: $(first_err "$TMPD/sonar.err")"
    unmeasured sonar_security_rating le "$SONAR_RATING_HOW: security_rating" "$r"
    unmeasured sonar_reliability_rating le "$SONAR_RATING_HOW: reliability_rating" "$r"
    unmeasured sonar_maintainability_rating le "$SONAR_RATING_HOW: sqale_rating" "$r"
    return
  fi
  local pair key metric count v n
  # key:sonar-metric:the count that explains the rating
  for pair in sonar_security_rating:security_rating:vulnerabilities \
              sonar_reliability_rating:reliability_rating:bugs \
              sonar_maintainability_rating:sqale_rating:code_smells; do
    key="${pair%%:*}"; metric="${pair#*:}"; count="${metric#*:}"; metric="${metric%%:*}"
    v="$(jq -r --arg m "$metric" '.component.measures[]? | select(.metric == $m) | .value' <<<"$measures" 2>/dev/null || true)"
    n="$(jq -r --arg m "$count" '.component.measures[]? | select(.metric == $m) | .value' <<<"$measures" 2>/dev/null || true)"
    case "$v" in
      [1-5]|[1-5].0)
        v="${v%.0}"
        put "$key" "$v" le "$SONAR_RATING_HOW: $metric" \
          "$(printf '%s' "$v" | tr 12345 ABCDE) ($count=${n:-?})" ;;
      *) unmeasured "$key" le "$SONAR_RATING_HOW: $metric" "metric $metric absent from api/measures/component" ;;
    esac
  done
}

# ---- GitHub security (gh api; needs repo security-read) -------------------
CODEQL_HOW="open CodeQL code-scanning alerts with security_severity_level high or critical (gh api repos/$REPO_SLUG/code-scanning/alerts?tool_name=CodeQL)"
DEPENDABOT_HOW="open Dependabot alerts of this severity (gh api repos/$REPO_SLUG/dependabot/alerts)"
SECRET_HOW="secret scanning enabled on the repo, 1/0 (gh api repos/$REPO_SLUG .security_and_analysis.secret_scanning)"
dependabot_rows_unmeasured() {
  local sev
  for sev in critical high medium low; do
    unmeasured "dependabot_open_$sev" le "$DEPENDABOT_HOW" "$1"
  done
}
gh_rows_unmeasured() {
  unmeasured codeql_high_open le "$CODEQL_HOW" "$1"
  dependabot_rows_unmeasured "$1"
  unmeasured secret_scanning_enabled ge "$SECRET_HOW" "$1"
}

gh_api() {  # OUTFILE ERRFILE gh-api-args... ; bounded by GH_DEADLINE
  local out="$1" err="$2"; shift 2
  with_deadline "$GH_DEADLINE" gh api "$@" >"$out" 2>"$err"
}

probe_github() {
  if [ "$OFFLINE" -eq 1 ]; then gh_rows_unmeasured "skipped: --offline"; return; fi
  if ! command -v gh >/dev/null; then gh_rows_unmeasured "gh not installed"; return; fi
  # `gh auth token` is local (no network), so "no credential" and "no network"
  # stay distinct reasons; `gh auth status` validates online and conflates them.
  if ! gh auth token >/dev/null 2>&1; then gh_rows_unmeasured "gh has no credential (gh auth token fails; run gh auth login)"; return; fi

  local high crit med
  if gh_api "$TMPD/codeql" "$TMPD/codeql.err" --paginate \
       "repos/$REPO_SLUG/code-scanning/alerts?state=open&tool_name=CodeQL&per_page=100" \
       --jq '.[].rule.security_severity_level // "none"'; then
    high="$(grep -cx high "$TMPD/codeql" || true)"
    crit="$(grep -cx critical "$TMPD/codeql" || true)"
    med="$(grep -cx medium "$TMPD/codeql" || true)"
    put codeql_high_open "$((high + crit))" le "$CODEQL_HOW" \
      "critical=$crit high=$high medium=$med (of $(grep -c . "$TMPD/codeql" || true) open CodeQL)"
  else
    unmeasured codeql_high_open le "$CODEQL_HOW" "gh api code-scanning failed: $(first_err "$TMPD/codeql.err")"
  fi

  if gh_api "$TMPD/dependabot" "$TMPD/dependabot.err" --paginate \
       "repos/$REPO_SLUG/dependabot/alerts?state=open&per_page=100" \
       --jq '.[] | "\(.security_advisory.severity) \(.dependency.package.name)"'; then
    local sev c pkgs
    for sev in critical high medium low; do
      c="$(grep -c "^$sev " "$TMPD/dependabot" || true)"
      pkgs="$({ grep "^$sev " "$TMPD/dependabot" || true; } | cut -d' ' -f2 | sort -u | tr '\n' ' ' | sed 's/ $//')"
      put "dependabot_open_$sev" "$c" le "$DEPENDABOT_HOW" "$pkgs"
    done
  else
    dependabot_rows_unmeasured "gh api dependabot failed: $(first_err "$TMPD/dependabot.err")"
  fi

  local ss pp
  if gh_api "$TMPD/repo" "$TMPD/repo.err" "repos/$REPO_SLUG" \
       --jq '"\(.security_and_analysis.secret_scanning.status // "unknown") \(.security_and_analysis.secret_scanning_push_protection.status // "unknown")"'; then
    ss="$(cut -d' ' -f1 "$TMPD/repo")"; pp="$(cut -d' ' -f2 "$TMPD/repo")"
    case "$ss" in
      enabled) put secret_scanning_enabled 1 ge "$SECRET_HOW" "push protection: $pp" ;;
      disabled) put secret_scanning_enabled 0 ge "$SECRET_HOW" "push protection: $pp" ;;
      *) unmeasured secret_scanning_enabled ge "$SECRET_HOW" "security_and_analysis not returned (needs admin on $REPO_SLUG)" ;;
    esac
  else
    unmeasured secret_scanning_enabled ge "$SECRET_HOW" "gh api repo failed: $(first_err "$TMPD/repo.err")"
  fi
}

# ---- Cloud Run used-or-not (--services; gcloud; slow) ---------------------
SERVICES_HOW="Cloud Run services with >=1 2xx request in 30 days, across $GCLOUD_PROJECTS (gcloud logging read, request logs, gcloud active account or CLOUDSDK_CORE_ACCOUNT); per-service request classes in .services"

# stdin: "url<TAB>remoteIp" lines -> "T<TAB>total<TAB>distinct_ips" + "C<TAB>n<TAB>class" lines
classify_requests() {
  awk -F'\t' '
    { url = $1; ip = $2; n++
      if (ip != "" && !(ip in seen)) { seen[ip] = 1; nip++ }
      sub(/^[a-z]+:\/\/[^\/]*/, "", url); sub(/[?#].*$/, "", url); if (url == "") url = "/"
      if (url ~ /^\/v1\/(traces|metrics|logs)/) c = "otlp"
      else if (url ~ /^\/(health|healthz|readyz|livez)(\/|$)/) c = "health"
      else if (url ~ /\.(js|css|map|png|jpe?g|gif|svg|ico|woff2?|ttf|html|txt|webmanifest)$/) c = "static"
      else { split(url, s, "/"); c = "/" s[2]; if (s[2] == "api" && s[3] != "") c = c "/" s[3] }
      cls[c]++ }
    END { printf "T\t%d\t%d\n", n, nip; for (c in cls) printf "C\t%d\t%s\n", cls[c], c }'
}

probe_services() {
  if [ "$SERVICES" -eq 0 ]; then
    unmeasured cloud_run_services_used info "$SERVICES_HOW" "not run: pass --services (slow; needs gcloud with read access to the request logs)"
    return
  fi
  if [ "$OFFLINE" -eq 1 ]; then unmeasured cloud_run_services_used info "$SERVICES_HOW" "skipped: --offline"; return; fi
  if ! command -v gcloud >/dev/null; then unmeasured cloud_run_services_used info "$SERVICES_HOW" "gcloud not installed"; return; fi

  local rows="$TMPD/services.jsonl" errors="" project svc region filter
  : >"$rows"
  for project in $GCLOUD_PROJECTS; do
    if ! gcloud run services list --project "$project" --quiet \
         --format='value(metadata.name,metadata.labels."cloud.googleapis.com/location")' \
         >"$TMPD/svcs" 2>"$TMPD/svcs.err"; then
      errors="$errors $project: $(first_err "$TMPD/svcs.err");"
      continue
    fi
    while IFS="$(printf '\t')" read -r svc region; do
      [ -n "$svc" ] || continue
      echo "simplicity_external: services: $project/$svc ..." >&2
      filter="resource.type=\"cloud_run_revision\" AND resource.labels.service_name=\"$svc\" AND logName=\"projects/$project/logs/run.googleapis.com%2Frequests\" AND httpRequest.status>=200 AND httpRequest.status<300"
      if ! gcloud logging read "$filter" --project "$project" --quiet \
           --freshness=30d --limit "$SERVICES_LOG_CAP" \
           --format='value(httpRequest.requestUrl,httpRequest.remoteIp)' \
           >"$TMPD/log" 2>"$TMPD/log.err" </dev/null; then
        jq -cn --arg p "$project" --arg s "$svc" --arg r "$region" --arg e "$(first_err "$TMPD/log.err")" \
          '{project: $p, service: $s, region: $r, error: $e}' >>"$rows"
        continue
      fi
      classify_requests <"$TMPD/log" >"$TMPD/cls"
      jq -cn --arg p "$project" --arg s "$svc" --arg r "$region" --argjson cap "$SERVICES_LOG_CAP" \
        --rawfile cls "$TMPD/cls" '
        ($cls | split("\n") | map(select(. != "") | split("\t"))) as $l
        | ($l | map(select(.[0] == "T")) | .[0]) as $t
        | {project: $p, service: $s, region: $r,
           requests_2xx: ($t[1] | tonumber), distinct_ips: ($t[2] | tonumber),
           truncated: (($t[1] | tonumber) >= $cap),
           classes: ($l | map(select(.[0] == "C") | {class: .[2], n: (.[1] | tonumber)}) | sort_by(-.n) | .[:8])}' >>"$rows"
    done <"$TMPD/svcs"
  done

  local measured
  measured="$(jq -s '[.[] | select(.error == null)] | length' "$rows")"
  if [ "$measured" -eq 0 ]; then
    unmeasured cloud_run_services_used info "$SERVICES_HOW" "no service measured:${errors:- gcloud listed no services}"
    return
  fi
  METRICS="$(jq -c --slurpfile svc "$rows" --arg how "$SERVICES_HOW" --arg errors "${errors# }" '
    ($svc | map(select(.error == null))) as $ok
    | . + {cloud_run_services_used: {
        value: ($ok | map(select(.requests_2xx > 0)) | length), gate: null, dir: "info", how: $how,
        detail: ((($ok | length) | tostring) + " measured; unused (0 2xx in 30d): "
                 + ([$ok[] | select(.requests_2xx == 0) | "\(.project)/\(.service)"] | if length == 0 then "none" else join(" ") end)
                 + ([$svc[] | select(.error != null) | " ERROR \(.project)/\(.service): \(.error)"] | join(""))
                 + (if $errors == "" then "" else " LIST-ERRORS " + $errors end)),
        services: $svc}}' <<<"$METRICS")"
}

probe_sonar
probe_github
probe_services
printf '%s\n' "$METRICS"
