#!/usr/bin/env bash
set -Eeuo pipefail

# Updates year_of_registration for already-ingested OpenSearch documents.
# Distribution is deterministic by document _id hash across years: 2022..2026.
#
# Usage:
#   ./scripts/update_existing_years.sh
#   ./scripts/update_existing_years.sh --index=delhi-ncr-0001
#   ./scripts/update_existing_years.sh --old-only
#
# Requires environment variables (from .env or shell):
#   OPENSEARCH_ENDPOINT, OPENSEARCH_MASTER_USER, OPENSEARCH_MASTER_PASSWORD
#   OPENSEARCH_INDEX (optional if --index passed)

INDEX=""
OLD_ONLY="false"
POLL_INTERVAL=8
LOG_FILE=""
START_TS="$(date +%s)"

ts() {
  date '+%Y-%m-%d %H:%M:%S'
}

log() {
  echo "[$(ts)] $*"
}

for arg in "$@"; do
  case "$arg" in
    --index=*) INDEX="${arg#*=}" ;;
    --old-only) OLD_ONLY="true" ;;
    --poll-interval=*) POLL_INTERVAL="${arg#*=}" ;;
    --log-file=*) LOG_FILE="${arg#*=}" ;;
    --help|-h)
      cat <<'EOF'
Usage:
  ./scripts/update_existing_years.sh [--index=<index-name>] [--old-only] [--poll-interval=8] [--log-file=path]

Options:
  --index=<name>  OpenSearch index to update (defaults to OPENSEARCH_INDEX)
  --old-only      Update only docs with year_of_registration in [2022,2023,2024] or missing
  --poll-interval Poll interval seconds for task status (default: 8)
  --log-file      Write all output to this file and stdout (default: logs/update_existing_years_<timestamp>.log)
EOF
      exit 0
      ;;
    *)
      echo "Unknown argument: $arg" >&2
      exit 1
      ;;
  esac
done

# Prevent history expansion issues with passwords containing '!'.
set +H

if [[ -f ./.env ]]; then
  set -a
  # shellcheck disable=SC1091
  source ./.env
  set +a
fi

ENDPOINT="${OPENSEARCH_ENDPOINT:-}"
USER_NAME="${OPENSEARCH_MASTER_USER:-}"
PASSWORD="${OPENSEARCH_MASTER_PASSWORD:-}"

if [[ -z "$INDEX" ]]; then
  INDEX="${OPENSEARCH_INDEX:-}"
fi

if [[ -z "$ENDPOINT" || -z "$USER_NAME" || -z "$PASSWORD" || -z "$INDEX" ]]; then
  echo "Missing required OpenSearch configuration." >&2
  echo "Need: OPENSEARCH_ENDPOINT, OPENSEARCH_MASTER_USER, OPENSEARCH_MASTER_PASSWORD, OPENSEARCH_INDEX (or --index)." >&2
  exit 1
fi

if [[ ! "$POLL_INTERVAL" =~ ^[0-9]+$ || "$POLL_INTERVAL" -lt 1 ]]; then
  echo "--poll-interval must be a positive integer" >&2
  exit 1
fi

if [[ -z "$LOG_FILE" ]]; then
  mkdir -p logs
  LOG_FILE="logs/update_existing_years_$(date +%Y%m%d_%H%M%S).log"
fi

exec > >(tee -a "$LOG_FILE") 2>&1

log "Starting update_existing_years.sh"
log "Log file: $LOG_FILE"

AUTH=(-u "${USER_NAME}:${PASSWORD}")
COMMON_HEADERS=(-H 'Content-Type: application/json')

if [[ "$OLD_ONLY" == "true" ]]; then
  read -r -d '' QUERY_JSON <<'JSON' || true
{
  "bool": {
    "should": [
      { "terms": { "year_of_registration": [2022, 2023, 2024] } },
      {
        "bool": {
          "must_not": {
            "exists": { "field": "year_of_registration" }
          }
        }
      }
    ],
    "minimum_should_match": 1
  }
}
JSON
else
  QUERY_JSON='{"match_all":{}}'
fi

log "Index: $INDEX"
log "Old-only mode: $OLD_ONLY"

COUNT_PAYLOAD="{\"query\":$QUERY_JSON}"
COUNT_RESPONSE="$(curl -sS "${AUTH[@]}" "${COMMON_HEADERS[@]}" -X POST "${ENDPOINT}/${INDEX}/_count" -d "$COUNT_PAYLOAD")"
TARGET_COUNT="$(python3 -c 'import json,sys; print(json.loads(sys.stdin.read()).get("count","unknown"))' <<< "$COUNT_RESPONSE" 2>/dev/null || echo unknown)"
log "Candidate docs to update: $TARGET_COUNT"

UPDATE_PAYLOAD="$(cat <<JSON
{
  "script": {
    "lang": "painless",
    "source": "int[] years = new int[]{2022,2023,2024,2025,2026}; int bucket = Math.floorMod(ctx._id.hashCode(), years.length); ctx._source.year_of_registration = years[bucket];"
  },
  "query": $QUERY_JSON
}
JSON
)"

log "Starting _update_by_query task..."
START_RESPONSE="$(curl -sS "${AUTH[@]}" "${COMMON_HEADERS[@]}" -X POST "${ENDPOINT}/${INDEX}/_update_by_query?conflicts=proceed&slices=auto&wait_for_completion=false" -d "$UPDATE_PAYLOAD")"

log "Start response: $START_RESPONSE"

TASK_ID="$(python3 -c 'import json,sys; print(json.loads(sys.stdin.read()).get("task",""))' <<< "$START_RESPONSE" 2>/dev/null || true)"
if [[ -z "$TASK_ID" ]]; then
  log "Could not extract task id. Check response above."
  exit 1
fi

log "Task ID: $TASK_ID"
log "Polling every ${POLL_INTERVAL}s..."

LAST_UPDATED=0
LAST_TOTAL=0

while true; do
  TASK_RESPONSE="$(curl -sS "${AUTH[@]}" "${ENDPOINT}/_tasks/${TASK_ID}")"

  COMPLETED="$(python3 -c 'import json,sys; print(str(json.loads(sys.stdin.read()).get("completed", False)).lower())' <<< "$TASK_RESPONSE" 2>/dev/null || echo false)"

  read -r TOTAL UPDATED CREATED DELETED CONFLICTS BATCHES FAILURES <<< "$(python3 - <<'PY' <<< "$TASK_RESPONSE"
import json,sys
j=json.loads(sys.stdin.read())
t=(j.get('task') or {}).get('status') or {}
f=(j.get('response') or {}).get('failures') or []
print(t.get('total',0), t.get('updated',0), t.get('created',0), t.get('deleted',0), t.get('version_conflicts',0), t.get('batches',0), len(f))
PY
)"

  ELAPSED=$(( $(date +%s) - START_TS ))
  RATE="0.00"
  if [[ "$ELAPSED" -gt 0 ]]; then
    RATE="$(awk -v u="$UPDATED" -v e="$ELAPSED" 'BEGIN { printf "%.2f", u/e }')"
  fi

  PCT="n/a"
  if [[ "$TARGET_COUNT" =~ ^[0-9]+$ && "$TARGET_COUNT" -gt 0 ]]; then
    PCT="$(awk -v u="$UPDATED" -v t="$TARGET_COUNT" 'BEGIN { printf "%.2f", (u/t)*100 }')%"
  elif [[ "$TOTAL" =~ ^[0-9]+$ && "$TOTAL" -gt 0 ]]; then
    PCT="$(awk -v u="$UPDATED" -v t="$TOTAL" 'BEGIN { printf "%.2f", (u/t)*100 }')%"
  fi

  LAST_UPDATED="$UPDATED"
  LAST_TOTAL="$TOTAL"

  log "progress updated=${UPDATED} total=${TOTAL} percent=${PCT} rate=${RATE}/s elapsed=${ELAPSED}s created=${CREATED} deleted=${DELETED} conflicts=${CONFLICTS} batches=${BATCHES} failures=${FAILURES}"

  if [[ "$COMPLETED" == "true" ]]; then
    log "Task completed."
    break
  fi

  sleep "$POLL_INTERVAL"
done

log "Final summary: updated=${LAST_UPDATED} total=${LAST_TOTAL}"

log "Verifying year distribution..."
AGG_RESPONSE="$(curl -sS "${AUTH[@]}" "${COMMON_HEADERS[@]}" -X POST "${ENDPOINT}/${INDEX}/_search?size=0" -d '{"aggs":{"years":{"terms":{"field":"year_of_registration","size":10,"order":{"_key":"asc"}}}}}')"
python3 - <<'PY' <<< "$AGG_RESPONSE"
import json,sys
j=json.loads(sys.stdin.read())
buckets = (((j.get('aggregations') or {}).get('years') or {}).get('buckets') or [])
print('year distribution:')
for b in buckets:
    print(f"  {b.get('key')}: {b.get('doc_count')}")
PY

log "Done."
