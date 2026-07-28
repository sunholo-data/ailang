#!/usr/bin/env bash
# triage_cascade.sh — one-shot diagnosis for an ailang publish cascade
#
# Usage:
#   triage_cascade.sh <vendor/name>@<version> [hours-window]
#   triage_cascade.sh sunholo/motoko_ext_abi@2.1.0 2
#
# Inspects:
#   1. registry-validator HTTP logs (was the publish accepted?)
#   2. cascade Pub/Sub topic existence + push subscription endpoint
#   3. coordinator logs around the publish window (cascade tasks created? dispatched? failed?)
#   4. Cloud Run Job execution status if any tasks ran
#
# Env:
#   AILANG_CLOUD_PROJECT (default: ailang-multivac-dev)
#   AILANG_TOPIC_PREFIX  (default: ailang-dev)
#   AILANG_REGION        (default: europe-west1)

set -euo pipefail

if [ $# -lt 1 ]; then
  echo "usage: $0 <vendor/name>@<version> [hours-window=1]" >&2
  exit 1
fi

PKG_VER="$1"
HOURS="${2:-1}"
PROJECT="${AILANG_CLOUD_PROJECT:-ailang-multivac-dev}"
PREFIX="${AILANG_TOPIC_PREFIX:-ailang-dev}"
REGION="${AILANG_REGION:-europe-west1}"

PKG="${PKG_VER%%@*}"
VER="${PKG_VER##*@}"

# Time window: ENDS now, STARTS $HOURS ago.
NOW_UTC="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
if date -u -v-${HOURS}H +%Y-%m-%dT%H:%M:%SZ >/dev/null 2>&1; then
  START_UTC="$(date -u -v-${HOURS}H +%Y-%m-%dT%H:%M:%SZ)"   # macOS / BSD
else
  START_UTC="$(date -u -d "${HOURS} hours ago" +%Y-%m-%dT%H:%M:%SZ)"   # GNU
fi

echo "═══════════════════════════════════════════════════════════════"
echo " Cascade triage: $PKG_VER"
echo " Project : $PROJECT"
echo " Prefix  : $PREFIX"
echo " Window  : $START_UTC to $NOW_UTC ($HOURS h)"
echo "═══════════════════════════════════════════════════════════════"
echo

# ────────────────────────────────────────────────────────────────
echo "─── 1/4 Registry validator (project: ailang-registry) ─────"
gcloud logging read \
  "resource.type=\"cloud_run_revision\" AND
   resource.labels.service_name=\"ailang-registry-validator\" AND
   httpRequest.requestMethod=\"POST\" AND
   timestamp>=\"${START_UTC}\" AND
   timestamp<=\"${NOW_UTC}\"" \
  --project ailang-registry --limit 20 \
  --format="value(timestamp,httpRequest.requestUrl,httpRequest.status)" 2>/dev/null \
  | sed -n '1,15p' \
  || echo "  (no validator logs in window — publish may have been outside this window)"
echo

# ────────────────────────────────────────────────────────────────
echo "─── 2/4 Cascade Pub/Sub topic + subscriptions ────────────"
TOPIC="$PREFIX-cascade"
echo "  Topic: projects/$PROJECT/topics/$TOPIC"
gcloud pubsub topics describe "$TOPIC" --project "$PROJECT" \
  --format="value(name)" 2>&1 | sed 's/^/    /' || true

echo "  Subscriptions:"
gcloud pubsub subscriptions list --project "$PROJECT" \
  --filter="topic~cascade" \
  --format="value(name,pushConfig.pushEndpoint)" 2>/dev/null \
  | sed 's/^/    /' \
  || echo "    (no cascade subscriptions found — cascade is silently no-op)"
echo

# ────────────────────────────────────────────────────────────────
echo "─── 3/4 Coordinator activity (cascade tasks) ─────────────"
# Best-effort: try the dev coordinator name first, then prod.
for SVC in "${PREFIX}-coordinator" "ailang-coordinator"; do
  echo "  Service: $SVC"
  COUNT="$(gcloud logging read \
    "resource.type=\"cloud_run_revision\" AND
     resource.labels.service_name=\"$SVC\" AND
     timestamp>=\"${START_UTC}\" AND
     timestamp<=\"${NOW_UTC}\" AND
     (textPayload=~\"cascade\" OR textPayload=~\"task-\" OR textPayload=~\"failed\" OR textPayload=~\"$PKG\")" \
    --project "$PROJECT" --limit 100 \
    --format="value(timestamp,textPayload)" 2>/dev/null | wc -l | tr -d ' ')"
  echo "    Matching log lines: $COUNT"
  if [ "$COUNT" != "0" ]; then
    gcloud logging read \
      "resource.type=\"cloud_run_revision\" AND
       resource.labels.service_name=\"$SVC\" AND
       timestamp>=\"${START_UTC}\" AND
       timestamp<=\"${NOW_UTC}\" AND
       (textPayload=~\"Created task\" OR textPayload=~\"Cloud dispatch\" OR textPayload=~\"failed.*error\" OR textPayload=~\"$PKG\")" \
      --project "$PROJECT" --limit 30 \
      --format="value(timestamp,textPayload)" 2>/dev/null \
      | sed 's/^/      /'
  fi
done
echo

# ────────────────────────────────────────────────────────────────
echo "─── 4/4 Cloud Run Job recent executions (last 10) ─────────"
gcloud run jobs executions list --project "$PROJECT" --region "$REGION" --limit 10 \
  --format="value(metadata.name,status.completionTime,status.conditions[0].message)" 2>/dev/null \
  | sed 's/^/  /' \
  || echo "  (no job executions found)"
echo

# ────────────────────────────────────────────────────────────────
echo "═══════════════════════════════════════════════════════════════"
echo " Quick interpretation guide:"
echo "  • No validator POST logs    → publish targeted a different project,"
echo "                                 or HOURS window is wrong"
echo "  • Topic missing             → AILANG_CLOUD_PROJECT/TOPIC_PREFIX wrong"
echo "  • Tasks created with"
echo "    'agent: ' (empty)         → no agent in coordinator config for"
echo "                                 'pkg:*' inbox routing"
echo "  • Tasks dispatched but"
echo "    'AILANG_AGENT_ID required'→ same as above; downstream symptom"
echo "  • No coordinator logs at all→ coordinator service down or revision broken"
echo "═══════════════════════════════════════════════════════════════"
