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
export CLOUDSDK_CORE_DISABLE_PROMPTS=1   # a gcloud y/N prompt must fail, never hang

PROJECT=${PROJECT:-ailang-gemini-agents}
REF_PROJECT=${REF_PROJECT:-ailang-dev}   # billing account + parent folder are copied from here
FOLDER=$(gcloud projects describe "$REF_PROJECT" --format='value(parent.id)')
BILLING=$(gcloud billing projects describe "$REF_PROJECT" --format='value(billingAccountName)'); BILLING=${BILLING#billingAccounts/}
USER_MEMBER=user:$(gcloud config get-value account 2>/dev/null)
SA=gemini-agent@${PROJECT}.iam.gserviceaccount.com
BUDGET_USD=${BUDGET_USD:-50}
[ -n "$FOLDER" ] && [ -n "$BILLING" ] || { echo "could not read folder/billing from $REF_PROJECT" >&2; exit 1; }
run() { echo "+ $*"; if [ "${APPLY:-0}" = 1 ]; then "$@"; fi; }

# Prerequisite: roles/orgpolicy.policyAdmin at the ORG (org Owner/Admin do NOT include it):
#   gcloud organizations add-iam-policy-binding <org-id> --member=user:<you> --role=roles/orgpolicy.policyAdmin
# Re-runnable: steps that already happened are skipped or are idempotent upserts.

# 1. Project under the ailang folder (inherits the org's no-API-keys policy). No Firebase, ever.
if gcloud projects describe "$PROJECT" >/dev/null 2>&1; then echo "= project $PROJECT exists"
else run gcloud projects create "$PROJECT" --folder="$FOLDER" --name="AILANG Gemini agents - locked"; fi

# 2. Local policy hardening, on top of what the folder already inherits.
#    - no SA key files (the folder does NOT enforce this today)
#    - default SAs get no automatic Editor grant (belt-and-braces; compute stays disabled)
run gcloud resource-manager org-policies enable-enforce iam.disableServiceAccountKeyCreation --project="$PROJECT"
run gcloud resource-manager org-policies enable-enforce iam.automaticIamGrantsForDefaultServiceAccounts --project="$PROJECT"

# 3. Billing; turn OFF the ~20 services a new project gets by default (BigQuery, Storage,
#    Datastore, Dataplex...), keeping only service plumbing + audit logging; then enable
#    ONLY what the keyless path needs (iam = SA management, iamcredentials = token minting).
run gcloud billing projects link "$PROJECT" --billing-account="$BILLING"
KEEP='^(serviceusage|servicemanagement|logging|monitoring|iam|iamcredentials|generativelanguage|apikeys)\.googleapis\.com$'
EXTRA=$(gcloud services list --enabled --project="$PROJECT" --format='value(config.name)' | grep -Ev "$KEEP" || true)
if [ -n "$EXTRA" ]; then run gcloud services disable $EXTRA --project="$PROJECT" --force; fi
run gcloud services enable generativelanguage.googleapis.com iam.googleapis.com iamcredentials.googleapis.com --project="$PROJECT"

# 4. One keyless SA. Its only role: consume this project's services (needed for x-goog-user-project).
#    Created BEFORE the allowlist below so the allowlist cannot block SA management.
if gcloud iam service-accounts describe "$SA" --project="$PROJECT" >/dev/null 2>&1; then echo "= $SA exists"
else run gcloud iam service-accounts create gemini-agent --project="$PROJECT" \
  --display-name="Gemini agent (keyless, impersonation only)"
  # A new SA is not yet visible to IAM bindings ("does not exist") for a few seconds.
  if [ "${APPLY:-0}" = 1 ]; then
    for _ in 1 2 3 4 5 6 7 8 9 10 11 12; do
      gcloud iam service-accounts describe "$SA" --project="$PROJECT" >/dev/null 2>&1 && break; sleep 5
    done
    sleep 10
  fi
fi
run gcloud projects add-iam-policy-binding "$PROJECT" \
  --member="serviceAccount:$SA" --role=roles/serviceusage.serviceUsageConsumer --condition=None
run gcloud iam service-accounts add-iam-policy-binding "$SA" --project="$PROJECT" \
  --member="$USER_MEMBER" --role=roles/iam.serviceAccountTokenCreator

# 5. Allowlist services LAST, so nothing else (aiplatform, compute, firebase...) can be turned on.
run gcloud resource-manager org-policies allow gcp.restrictServiceUsage \
  generativelanguage.googleapis.com iam.googleapis.com iamcredentials.googleapis.com \
  apikeys.googleapis.com --project="$PROJECT"

# 6. Budget alerts at 50/90/100% (alerts only — the HARD stop is the AI Studio Project Spend Cap, step 8).
if gcloud billing budgets list --billing-account="$BILLING" --format='value(displayName)' 2>/dev/null | grep -qx "$PROJECT cap"; then
  echo "= budget '$PROJECT cap' exists"
else run gcloud billing budgets create --billing-account="$BILLING" \
  --display-name="$PROJECT cap" --budget-amount="${BUDGET_USD}USD" \
  --filter-projects="projects/$PROJECT" \
  --threshold-rule=percent=0.5 --threshold-rule=percent=0.9 --threshold-rule=percent=1.0 \
  || echo "! budget not created (needs billingbudgets API on your gcloud quota project) — create it in the Billing console"; fi


# 7. API key — ONLY because the Gemini Developer API refuses SA OAuth tokens (403 "Access to Gemini
#    API is restricted with service accounts. Use authorization keys instead", 2026-10-13).
#    Blast radius is minimised: bound to the zero-role SA, restricted to generativelanguage only, in a
#    project where nothing else can be enabled, behind a spend cap. The key string is written straight
#    into SECRETS_FILE and never printed. Opt in with CREATE_KEY=1.
SECRETS_FILE=${SECRETS_FILE:-$HOME/.config/ailang/secrets.env}
KEY_VAR=GEMINI_AGENTS_API_KEY
if [ "${CREATE_KEY:-0}" = 1 ]; then
  run gcloud services enable apikeys.googleapis.com --project="$PROJECT"
  KEY_NAME=$(gcloud services api-keys list --project="$PROJECT" --filter="displayName=gemini-agent-key" --format='value(name)' 2>/dev/null | head -1 || true)
  if [ -n "$KEY_NAME" ]; then echo "= key gemini-agent-key exists ($KEY_NAME)"
  else
    # gcloud echoes the full Key resource INCLUDING keyString on create — never let it reach a terminal.
    echo "+ gcloud services api-keys create --project=$PROJECT --display-name=gemini-agent-key (output suppressed)"
    if [ "${APPLY:-0}" = 1 ]; then
      ERR=$(mktemp)
      gcloud services api-keys create --project="$PROJECT" --display-name=gemini-agent-key \
        --api-target=service=generativelanguage.googleapis.com --service-account="$SA" \
        >/dev/null 2>"$ERR" || { grep -v -i keyString "$ERR" >&2; rm -f "$ERR"; exit 1; }
      rm -f "$ERR"
    fi
    [ "${APPLY:-0}" = 1 ] && KEY_NAME=$(gcloud services api-keys list --project="$PROJECT" --filter="displayName=gemini-agent-key" --format='value(name)' | head -1)
  fi
  if [ "${APPLY:-0}" = 1 ] && [ -n "$KEY_NAME" ]; then
    KEY=$(gcloud services api-keys get-key-string "$KEY_NAME" --format='value(keyString)')
    touch "$SECRETS_FILE"; chmod 600 "$SECRETS_FILE"
    grep -v "^export $KEY_VAR=" "$SECRETS_FILE" > "$SECRETS_FILE.tmp" || true
    printf 'export %s=%s\n' "$KEY_VAR" "$KEY" >> "$SECRETS_FILE.tmp" && mv "$SECRETS_FILE.tmp" "$SECRETS_FILE"
    unset KEY; echo "= wrote $KEY_VAR to $SECRETS_FILE (value not shown)"
  fi
fi

# 8. HARD STOP (manual, UI only): AI Studio → import project $PROJECT → set a Project Spend Cap.
#    Enforced with ~10 min lag. The billing-account tier cap also applies on top.
echo
echo "MANUAL: https://aistudio.google.com → import project $PROJECT → Spend → set a monthly cap."
