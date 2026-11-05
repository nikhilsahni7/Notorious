#!/usr/bin/env bash
# Runs ON the new EC2 as a detached systemd unit (notorious-continue),
# launched by migrate.sh. Owns every long-running step after provisioning so
# the operator's laptop can be switched off:
#   1. download + unzip the data archive   3. Postgres restore (EC2 -> EC2)
#   2. download the CSV dataset            4. start JSON ingestion service
# Idempotent: safe to re-launch after a failure; completed steps are skipped.
# Config comes from /opt/notorious/rotation.env (written by migrate.sh).
set -Eeuo pipefail

# shellcheck disable=SC1091
source /opt/notorious/rotation.env

# The old EC2's SSH key is only needed for the dump in step 3. Remove it on
# any exit - success, failure, or kill.
trap 'rm -f /opt/notorious/aditya.pem' EXIT

log() { echo "[continue] $*"; }

s3_size() { # bucket key -> bytes, empty if missing
  aws s3api head-object --bucket "$1" --key "$2" --region "$REGION" \
    --query ContentLength --output text 2>/dev/null || true
}

file_size() { stat -c%s "$1" 2>/dev/null || echo 0; }

# --- 1. Data archive -> local JSON -----------------------------------------
# Completion is judged by the size marker written after a verified unzip,
# never by the zip (which is deleted), so re-runs skip this step entirely.
JSON_MARKER=/opt/notorious/json.size
if [[ -f "$JSON_MARKER" && "$(file_size "/data/$JSON_NAME")" == "$(cat "$JSON_MARKER")" ]]; then
  log "JSON already complete ($(cat "$JSON_MARKER") bytes), skipping download/unzip"
else
  ZIP_SIZE="$(s3_size "$BUCKET" "$DATA_KEY")"
  if [[ "$(file_size /data/data.zip)" != "$ZIP_SIZE" ]]; then
    log "downloading s3://$BUCKET/$DATA_KEY ..."
    aws s3 cp "s3://$BUCKET/$DATA_KEY" /data/data.zip --region "$REGION" --no-progress
  fi
  JSON_SIZE="$(unzip -l /data/data.zip | awk 'END {print $1}')"
  log "unzipping to /data/$JSON_NAME ($JSON_SIZE bytes)..."
  unzip -o /data/data.zip -d /data
  [[ "$(file_size "/data/$JSON_NAME")" == "$JSON_SIZE" ]] || { echo "ERROR: JSON size mismatch after unzip" >&2; exit 1; }
  echo "$JSON_SIZE" > "$JSON_MARKER"
  rm -f /data/data.zip
fi
log "JSON ready: /data/$JSON_NAME"

# --- 2. CSV dataset ----------------------------------------------------------
if [[ -n "${CSV_KEY:-}" ]]; then
  CSV_SIZE="$(s3_size "$BUCKET" "$CSV_KEY")"
  if [[ "$(file_size "/data/$CSV_NAME")" == "$CSV_SIZE" ]]; then
    log "CSV already present, skipping download"
  else
    log "downloading s3://$BUCKET/$CSV_KEY ..."
    aws s3 cp "s3://$BUCKET/$CSV_KEY" "/data/$CSV_NAME" --region "$REGION" --no-progress
  fi
  log "CSV ready: /data/$CSV_NAME"
fi

# --- 3. Postgres restore (old EC2 -> this EC2, direct SSH pipe) --------------
# Marker-guarded: pg_restore --clean is destructive, so a re-run must never
# repeat this over a database the API may already have written to.
if [[ -f /opt/notorious/pg-restored ]]; then
  log "Postgres already restored (marker present), skipping"
else
  log "restoring Postgres from $OLD_HOST..."
  ssh -i /opt/notorious/aditya.pem -o StrictHostKeyChecking=accept-new -o ServerAliveInterval=30 \
    "$OLD_USER@$OLD_HOST" "$DUMP_CMD" \
    | sudo -u postgres pg_restore --clean --if-exists --no-owner -d notorious
  touch /opt/notorious/pg-restored
  log "Postgres restore complete"
fi
rm -f /opt/notorious/aditya.pem

# --- 4. Start JSON ingestion --------------------------------------------------
sed -i -E "s|--file=/data/[^[:space:]]*|--file=/data/$JSON_NAME|" /etc/systemd/system/notorious-ingest.service
systemctl daemon-reload
systemctl enable notorious-ingest
systemctl restart notorious-ingest
sleep 5
systemctl is-active notorious-ingest
touch /opt/notorious/start-complete
log "ingestion running; continuation complete"
