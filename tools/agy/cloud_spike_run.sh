#!/bin/bash
# Launch the M-AGY-CLOUD Phase 0 spike (tools/agy/cloud_spike.sh) as a one-off Cloud
# Run job in the DEV project, then print its log. Never prod.
#
# The job runs the stock Google Cloud SDK image (Debian) as the zero-role spike SA,
# which can only read and add versions of the one agy secret.
#
# Usage: CLOUDSDK_CORE_ACCOUNT=m@sunholo.com tools/agy/cloud_spike_run.sh
set -euo pipefail

PROJECT="${SPIKE_PROJECT:-ailang-multivac-dev}"
REGION="${SPIKE_REGION:-europe-west1}"
SECRET="${SPIKE_SECRET:-ailang-dev-agy-oauth-token}"
JOB="${SPIKE_JOB:-ailang-dev-agy-spike}"
SA="ailang-dev-agy-spike@${PROJECT}.iam.gserviceaccount.com"
IMAGE="gcr.io/google.com/cloudsdktool/google-cloud-cli:slim"
HERE="$(cd "$(dirname "$0")" && pwd)"

case "$PROJECT" in *-dev) ;; *) echo "refusing: the spike runs in a -dev project only (got $PROJECT)" >&2; exit 2 ;; esac

script="$(cat "$HERE/cloud_spike.sh")"
# gcloud splits list flags on commas; ^@@^ switches the delimiter (the script has commas).
case "$script" in *@@*) echo "cloud_spike.sh must not contain '@@'" >&2; exit 2 ;; esac
verb=create
gcloud run jobs describe "$JOB" --project "$PROJECT" --region "$REGION" --format='value(name)' >/dev/null 2>&1 && verb=update
gcloud run jobs "$verb" "$JOB" --project "$PROJECT" --region "$REGION" \
  --image "$IMAGE" --service-account "$SA" \
  --command bash --args="^@@^-c@@$script" \
  --set-env-vars "SPIKE_PROJECT=$PROJECT,SPIKE_SECRET=$SECRET,SPIKE_MODEL=${SPIKE_MODEL:-gemini-4-neon-low}" \
  --labels purpose=m-agy-cloud-phase0,managed-by=hand \
  --task-timeout 900s --max-retries 0 --cpu 2 --memory 4Gi --quiet >/dev/null
echo "job $JOB ${verb}d in $PROJECT/$REGION; executing (waits for completion)..."
exec_name=$(gcloud run jobs execute "$JOB" --project "$PROJECT" --region "$REGION" --wait --format='value(metadata.name)' 2>/dev/null) || true
echo "execution: ${exec_name:-<unknown — see console>}"
sleep 5
gcloud logging read "resource.type=cloud_run_job AND resource.labels.job_name=$JOB${exec_name:+ AND labels.\"run.googleapis.com/execution_name\"=$exec_name}" \
  --project "$PROJECT" --order asc --limit 400 --format='value(textPayload)' --freshness=1h
