#!/usr/bin/env bash
set -Eeuo pipefail

# Robust ingestion runner for huge files.
# - Auto loads .env safely (handles ! in passwords)
# - Auto retries with exponential backoff
# - Auto resumes using last known checkpoint
#
# Usage examples:
#   ./scripts/run_ingest_with_auto_resume.sh --mode=json --file=/home/ubuntu/users_data.json
#   ./scripts/run_ingest_with_auto_resume.sh --mode=csv --file=/home/ubuntu/dela.csv --region=delhi-ncr --batch=25000

MODE=""
INPUT_FILE=""
REGION="delhi-ncr"
BATCH="25000"
INITIAL_RESUME="0"
MAX_RETRIES="0" # 0 = infinite retries
SLEEP_BASE="5"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode=*) MODE="${1#*=}" ;;
    --file=*) INPUT_FILE="${1#*=}" ;;
    --region=*) REGION="${1#*=}" ;;
    --batch=*) BATCH="${1#*=}" ;;
    --resume=*) INITIAL_RESUME="${1#*=}" ;;
    --max-retries=*) MAX_RETRIES="${1#*=}" ;;
    --sleep-base=*) SLEEP_BASE="${1#*=}" ;;
    -h|--help)
      cat <<'EOF'
Usage:
  run_ingest_with_auto_resume.sh --mode=json --file=/path/to/users_data.json [--resume=0] [--max-retries=0]
  run_ingest_with_auto_resume.sh --mode=csv --file=/path/to/dela.csv [--region=delhi-ncr] [--batch=25000] [--resume=0] [--max-retries=0]

Options:
  --mode           json | csv (required)
  --file           input file path (required)
  --region         csv mode only (default: delhi-ncr)
  --batch          csv mode only (default: 25000)
  --resume         start resume offset (default: 0)
  --max-retries    max retries before exit; 0 means retry forever (default: 0)
  --sleep-base     base retry sleep seconds for backoff (default: 5)
EOF
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
  shift
done

if [[ -z "$MODE" || -z "$INPUT_FILE" ]]; then
  echo "ERROR: --mode and --file are required" >&2
  exit 1
fi

if [[ "$MODE" != "json" && "$MODE" != "csv" ]]; then
  echo "ERROR: --mode must be json or csv" >&2
  exit 1
fi

if [[ ! -f "$INPUT_FILE" ]]; then
  echo "ERROR: file not found: $INPUT_FILE" >&2
  exit 1
fi

if [[ ! -f "./go.mod" || ! -d "./cmd" ]]; then
  echo "ERROR: run this script from backend directory (where go.mod exists)" >&2
  exit 1
fi

set +H
set -a
source ./.env
set +a

timestamp="$(date +%Y%m%d_%H%M%S)"
log_file="ingest_${MODE}_${timestamp}.log"
state_file=".ingest_${MODE}_state"

resume="$INITIAL_RESUME"
if [[ -f "$state_file" ]]; then
  saved_resume="$(cat "$state_file" 2>/dev/null || true)"
  if [[ "$saved_resume" =~ ^[0-9]+$ && "$saved_resume" -gt "$resume" ]]; then
    resume="$saved_resume"
  fi
fi

echo "=== Ingestion runner started ===" | tee -a "$log_file"
echo "mode=$MODE file=$INPUT_FILE region=$REGION batch=$BATCH resume=$resume" | tee -a "$log_file"
echo "log_file=$log_file state_file=$state_file" | tee -a "$log_file"

extract_resume_checkpoint() {
  local f="$1"
  local candidates
  candidates="$({
    grep -Eo 'resume[= ]+[0-9]+' "$f" || true
    grep -Eo 'Processed [0-9]+' "$f" || true
    grep -Eo 'Total documents processed: [0-9]+' "$f" || true
  } | grep -Eo '[0-9]+' || true)"

  if [[ -n "$candidates" ]]; then
    echo "$candidates" | sort -n | tail -1
  else
    echo ""
  fi
}

attempt=0
while true; do
  attempt=$((attempt + 1))
  echo "[attempt $attempt] starting with resume=$resume" | tee -a "$log_file"

  if [[ "$MODE" == "json" ]]; then
    set +e
    go run ./cmd/ingest/main.go --resume="$resume" "$INPUT_FILE" 2>&1 | tee -a "$log_file"
    exit_code=${PIPESTATUS[0]}
    set -e
  else
    set +e
    go run ./cmd/ingest_csv/main.go -file="$INPUT_FILE" -region="$REGION" -batch="$BATCH" -resume="$resume" 2>&1 | tee -a "$log_file"
    exit_code=${PIPESTATUS[0]}
    set -e
  fi

  if [[ "$exit_code" -eq 0 ]]; then
    echo "[attempt $attempt] ingestion completed successfully" | tee -a "$log_file"
    echo "$resume" > "$state_file"
    exit 0
  fi

  checkpoint="$(extract_resume_checkpoint "$log_file")"
  if [[ "$checkpoint" =~ ^[0-9]+$ && "$checkpoint" -gt "$resume" ]]; then
    resume="$checkpoint"
    echo "$resume" > "$state_file"
    echo "[attempt $attempt] checkpoint detected, resuming from $resume" | tee -a "$log_file"
  else
    echo "[attempt $attempt] no higher checkpoint detected, retrying from resume=$resume" | tee -a "$log_file"
  fi

  if [[ "$MAX_RETRIES" -gt 0 && "$attempt" -ge "$MAX_RETRIES" ]]; then
    echo "[attempt $attempt] reached max retries ($MAX_RETRIES), exiting with failure" | tee -a "$log_file"
    exit 1
  fi

  sleep_seconds=$((SLEEP_BASE * attempt))
  if [[ "$sleep_seconds" -gt 300 ]]; then
    sleep_seconds=300
  fi
  echo "[attempt $attempt] sleeping ${sleep_seconds}s before retry" | tee -a "$log_file"
  sleep "$sleep_seconds"
done
