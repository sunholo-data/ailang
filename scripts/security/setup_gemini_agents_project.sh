#!/bin/bash
# Locked-down GCP project for KEYLESS Gemini Developer API access (2026-10-13).
#
# Why a separate project: the Developer API (generativelanguage.googleapis.com) has
# features Vertex lacks (Managed Agents repo sources, GitHub credentials, files API,
# newer antigravity agents). Enabling it in a project that holds a Firebase
# auto-created browser key is how a key gets silently upgraded to AI access — that
# cost $30k once. So: no Firebase, no API keys (org policy), no SA key files, only
# two services allowed, and access ONLY by impersonating a zero-role SA.
#   Never use a default compute SA for this — they usually hold project Editor.
#
# Billing account, folder and user are read from the reference project at run time
# so no account identifiers live in this (public) repo.
#
# Default is a DRY RUN that only prints. Apply with: APPLY=1 bash <this file>
# Use afterwards:
#   T=$(gcloud auth print-access-token --impersonate-service-account=$SA \
#        --scopes=https://www.googleapis.com/auth/generative-language,https://www.googleapis.com/auth/cloud-platform)
#   curl -H "Authorization: Bearer $T" -H "x-goog-user-project: $PROJECT" \
#        https://generativelanguage.googleapis.com/v1beta/models
set -euo pipefail

PROJECT=${PROJECT:-ailang-gemini-agents}
REF_PROJECT=${REF_PROJECT:-ailang-dev}   # billing account + parent folder are copied from here
FOLDER=$(gcloud projects describe "$REF_PROJECT" --format='value(parent.id)')
BILLING=$(gcloud billing projects describe "$REF_PROJECT" --format='value(billingAccountName)'); BILLING=${BILLING#billingAccounts/}
USER_MEMBER=user:$(gcloud config get-value account 2>/dev/null)
SA=gemini-agent@${PROJECT}.iam.gserviceaccount.com
BUDGET_USD=${BUDGET_USD:-50}
[ -n "$FOLDER" ] && [ -n "$BILLING" ] || { echo "could not read folder/billing from $REF_PROJECT" >&2; exit 1; }
run() { echo "+ $*"; if [ "${APPLY:-0}" = 1 ]; then "$@"; fi; }

# 1. Project under the ailang folder (inherits the org's no-API-keys policy). No Firebase, ever.
run gcloud projects create "$PROJECT" --folder="$FOLDER" --name="AILANG Gemini agents - locked"

# 2. Local policy hardening, on top of what the folder already inherits.
#    - no SA key files (the folder does NOT enforce this today)
#    - default SAs get no automatic Editor grant (belt-and-braces; compute stays disabled)
run gcloud resource-manager org-policies enable-enforce iam.disableServiceAccountKeyCreation --project="$PROJECT"
run gcloud resource-manager org-policies enable-enforce iam.automaticIamGrantsForDefaultServiceAccounts --project="$PROJECT"

# 3. Billing, then ONLY the two APIs the keyless path needs.
run gcloud billing projects link "$PROJECT" --billing-account="$BILLING"
run gcloud services enable generativelanguage.googleapis.com iamcredentials.googleapis.com --project="$PROJECT"

# 4. Allowlist services AFTER enabling, so nothing else (aiplatform, compute, firebase...) can be turned on.
run gcloud resource-manager org-policies allow gcp.restrictServiceUsage \
  generativelanguage.googleapis.com iamcredentials.googleapis.com --project="$PROJECT"

# 5. One keyless SA. Its only role: consume this project's services (needed for x-goog-user-project).
run gcloud iam service-accounts create gemini-agent --project="$PROJECT" \
  --display-name="Gemini agent (keyless, impersonation only)"
run gcloud projects add-iam-policy-binding "$PROJECT" \
  --member="serviceAccount:$SA" --role=roles/serviceusage.serviceUsageConsumer --condition=None
run gcloud iam service-accounts add-iam-policy-binding "$SA" --project="$PROJECT" \
  --member="$USER_MEMBER" --role=roles/iam.serviceAccountTokenCreator

# 6. Budget alerts at 50/90/100% (alerts only; the hard stop is the quota cap, set after enable — see below).
run gcloud billing budgets create --billing-account="$BILLING" \
  --display-name="$PROJECT cap" --budget-amount="${BUDGET_USD}USD" \
  --filter-projects="projects/$PROJECT" \
  --threshold-rule=percent=0.5 --threshold-rule=percent=0.9 --threshold-rule=percent=1.0


echo
echo "Next (read-only): list the real quota metric names so the daily cap is set on a measured name:"
echo "  gcloud alpha services quota list --service=generativelanguage.googleapis.com --consumer=projects/$PROJECT"
