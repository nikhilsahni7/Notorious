#!/usr/bin/env bash
# Monthly AWS account rotation for notorious.
#
#   ./migrate.sh start    # provision new account, copy data S3->S3, then arm a
#                         # detached service ON the new EC2 that downloads,
#                         # unzips, restores Postgres and starts ingestion.
#                         # After `start` returns, the laptop can be switched off.
#   ./migrate.sh status   # check ingestion progress + OpenSearch doc count
#   ./migrate.sh finish   # after ingestion: deploy API, copy certs, DNS cutover
#   ./migrate.sh <phase>  # run a single phase (see usage)
#
# Copy migrate.env.example to migrate.env and fill it in first.
# Design note: ingestion reads the file from local disk (/data), exactly like
# the current manual flow. S3 is used only as AWS-CLI staging between accounts;
# the app's untested Go S3 code paths are not used anywhere here.
set -Eeuo pipefail

cd "$(dirname "$0")"
CONFIG_FILE="${MIGRATE_CONFIG:-migrate.env}"
[[ -f "$CONFIG_FILE" ]] || { echo "ERROR: $CONFIG_FILE not found (copy migrate.env.example)" >&2; exit 1; }
# shellcheck disable=SC1090
set -a; source "$CONFIG_FILE"; set +a

: "${OLD_PROFILE:?}" "${NEW_PROFILE:?}" "${OLD_BUCKET:?}" "${DATA_S3_KEY:?}"
: "${OLD_EC2_HOST:?}" "${NEW_EC2_SSH_KEY:?}" "${DOMAIN:?}"
OLD_EC2_USER="${OLD_EC2_USER:-ubuntu}"
OLD_EC2_SSH_KEY="${OLD_EC2_SSH_KEY:-$NEW_EC2_SSH_KEY}"
OLD_DUMP_CMD="${OLD_DUMP_CMD:-sudo -u postgres pg_dump -Fc notorious}"
NEW_REGION="${NEW_REGION:-us-east-2}"
JSON_FILE_NAME="${JSON_FILE_NAME:-users_data.json}"
CSV_S3_KEY="${CSV_S3_KEY:-}"
CSV_LOCAL_PATH="${CSV_LOCAL_PATH:-}"
CSV_FILE_NAME="${CSV_FILE_NAME:-dela.csv}"
CSV_INDEX="${CSV_INDEX:-people-delhi-ncr-0001}"
TF_BIN="${TF_BIN:-terraform}"
command -v "$TF_BIN" >/dev/null 2>&1 || TF_BIN="tofu"

log()  { echo "[migrate] $*"; }
die()  { echo "ERROR: $*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "missing tool: $1"; }

tf()  { AWS_PROFILE="$NEW_PROFILE" "$TF_BIN" -chdir=infra "$@"; }
tf_out() { tf output -raw "$1" 2>/dev/null; }

NEW_IP=""; NEW_BUCKET_NAME=""
load_outputs() {
  NEW_IP="$(tf_out ec2_public_ip)";        [[ -n "$NEW_IP" ]] || die "no terraform outputs - run provision first"
  NEW_BUCKET_NAME="$(tf_out bucket)"
  # Region/domain live in BOTH migrate.env and terraform.tfvars; drift
  # between them fails late and confusingly, so refuse early.
  local tf_region tf_domain
  tf_region="$(tf_out aws_region)"; tf_domain="$(tf_out domain)"
  [[ -z "$tf_region" || "$tf_region" == "$NEW_REGION" ]] || die "region drift: migrate.env NEW_REGION=$NEW_REGION but terraform aws_region=$tf_region"
  [[ -z "$tf_domain" || "$tf_domain" == "$DOMAIN" ]]     || die "domain drift: migrate.env DOMAIN=$DOMAIN but terraform domain=$tf_domain"
}

# First connections pin the host key (TOFU). Set MIGRATE_STRICT_SSH=1 after
# verifying fingerprints out-of-band (EC2 console output shows them) to make
# later runs refuse unknown keys.
if [[ "${MIGRATE_STRICT_SSH:-}" == "1" ]]; then
  SSH_OPTS=(-o StrictHostKeyChecking=yes -o ServerAliveInterval=30)
else
  SSH_OPTS=(-o StrictHostKeyChecking=accept-new -o ServerAliveInterval=30)
fi
ssh_new() { ssh "${SSH_OPTS[@]}" -i "$NEW_EC2_SSH_KEY" "ubuntu@$NEW_IP" "$@"; }
ssh_old() { ssh "${SSH_OPTS[@]}" -i "$OLD_EC2_SSH_KEY" "$OLD_EC2_USER@$OLD_EC2_HOST" "$@"; }
scp_new() { scp "${SSH_OPTS[@]}" -i "$NEW_EC2_SSH_KEY" "$1" "ubuntu@$NEW_IP:$2"; }

# Doc count for an index, queried from the new EC2 (the only host the
# OpenSearch security group allows). Creds are read from the remote .env.
os_count() {
  ssh_new "OSU=\$(grep '^OPENSEARCH_MASTER_USER=' ~/notorious/backend/.env | cut -d= -f2-); \
    OSP=\$(grep '^OPENSEARCH_MASTER_PASSWORD=' ~/notorious/backend/.env | cut -d= -f2-); \
    OSE=\$(grep '^OPENSEARCH_ENDPOINT=' ~/notorious/backend/.env | cut -d= -f2-); \
    curl -s -u \"\$OSU:\$OSP\" \"\$OSE/$1/_count\"" | jq .count
}

# ------------------------------------------------------------------ phases

preflight() {
  need aws; need jq; need ssh; need "$TF_BIN"
  local old_acct new_acct
  old_acct="$(aws sts get-caller-identity --profile "$OLD_PROFILE" --query Account --output text)"
  new_acct="$(aws sts get-caller-identity --profile "$NEW_PROFILE" --query Account --output text)"
  [[ "$old_acct" != "$new_acct" ]] || die "OLD_PROFILE and NEW_PROFILE resolve to the SAME account ($old_acct)"
  log "old account=$old_acct  new account=$new_acct"
  aws s3api head-bucket --bucket "$OLD_BUCKET" --profile "$OLD_PROFILE" >/dev/null \
    || die "cannot access old bucket s3://$OLD_BUCKET with profile $OLD_PROFILE"
  log "preflight OK"
}

provision() {
  preflight
  # Archive a previous rotation's state so we never apply it against the new account.
  if [[ -f infra/terraform.tfstate ]]; then
    local state_acct new_acct archive
    # Read the account straight from the state file - `terraform output`
    # silently fails before `terraform init` has run on this machine.
    state_acct="$(jq -r '.outputs.new_account_id.value // empty' infra/terraform.tfstate 2>/dev/null || true)"
    new_acct="$(aws sts get-caller-identity --profile "$NEW_PROFILE" --query Account --output text)"
    if [[ -n "$state_acct" && "$state_acct" != "$new_acct" ]]; then
      archive="infra/state-archive/$state_acct-$(date +%Y%m%d%H%M%S)"
      mkdir -p "$archive"
      mv infra/terraform.tfstate* "$archive"/
      log "archived previous state (account $state_acct) to $archive"
    fi
  fi
  log "terraform apply (OpenSearch takes ~15-30 min)..."
  tf init -input=false
  tf apply -input=false -auto-approve
  load_outputs
  log "new EC2=$NEW_IP bucket=$NEW_BUCKET_NAME opensearch=$(tf_out opensearch_endpoint)"
}

# Temporarily let the NEW account's operator identity read the one data
# object in the OLD bucket, copy S3->S3 server-side, then revoke. The revoke
# is trap-guarded: it runs on success, failure, and Ctrl-C alike.
revoke_old_bucket() {
  local policy
  policy="$(aws s3api get-bucket-policy --bucket "$OLD_BUCKET" --profile "$OLD_PROFILE" --query Policy --output text 2>/dev/null || true)"
  [[ -n "$policy" ]] || return 0
  policy="$(echo "$policy" | jq '.Statement |= map(select(.Sid != "NotoriousRotationRead"))')"
  if [[ "$(echo "$policy" | jq '.Statement | length')" == "0" ]]; then
    aws s3api delete-bucket-policy --bucket "$OLD_BUCKET" --profile "$OLD_PROFILE"
  else
    aws s3api put-bucket-policy --bucket "$OLD_BUCKET" --profile "$OLD_PROFILE" --policy "$policy"
  fi
  # Verify the grant is actually gone before claiming so.
  policy="$(aws s3api get-bucket-policy --bucket "$OLD_BUCKET" --profile "$OLD_PROFILE" --query Policy --output text 2>/dev/null || echo '{}')"
  echo "$policy" | jq -e '.Statement[]? | select(.Sid == "NotoriousRotationRead")' >/dev/null \
    && die "failed to revoke cross-account grant on old bucket - remove Sid NotoriousRotationRead manually" \
    || log "revoked temporary cross-account access on old bucket"
}

grant_old_bucket() {
  local principal policy
  # Narrow the grant to the exact identity running the copy (convert an
  # assumed-role session ARN to its role ARN, which is a valid principal).
  principal="$(aws sts get-caller-identity --profile "$NEW_PROFILE" --query Arn --output text)"
  principal="$(echo "$principal" | sed -E 's|arn:aws:sts::([0-9]+):assumed-role/([^/]+)/.*|arn:aws:iam::\1:role/\2|')"
  policy="$(aws s3api get-bucket-policy --bucket "$OLD_BUCKET" --profile "$OLD_PROFILE" --query Policy --output text 2>/dev/null || echo '{"Version":"2012-10-17","Statement":[]}')"
  local expires
  expires="$(date -u -d '+6 hours' +%Y-%m-%dT%H:%M:%SZ)"
  policy="$(echo "$policy" | jq --arg principal "$principal" --arg bucket "$OLD_BUCKET" \
      --arg key "$DATA_S3_KEY" --arg csvkey "$CSV_S3_KEY" --arg expires "$expires" '
    .Statement |= map(select(.Sid != "NotoriousRotationRead")) +
    [{Sid:"NotoriousRotationRead", Effect:"Allow",
      Principal:{AWS:$principal},
      Action:["s3:GetObject","s3:ListBucket"],
      # Self-expires: even an orphaned grant (laptop killed mid-copy) dies in 6h.
      Condition:{DateLessThan:{"aws:CurrentTime":$expires}},
      Resource:(["arn:aws:s3:::"+$bucket, "arn:aws:s3:::"+$bucket+"/"+$key]
        + (if $csvkey == "" then [] else ["arn:aws:s3:::"+$bucket+"/"+$csvkey] end))}]')"
  aws s3api put-bucket-policy --bucket "$OLD_BUCKET" --profile "$OLD_PROFILE" --policy "$policy"
  trap revoke_old_bucket EXIT
  log "granted $principal temporary read on s3://$OLD_BUCKET/$DATA_S3_KEY (auto-revoked on exit)"
}

upload_csv() {
  [[ -n "$CSV_S3_KEY" && -n "$CSV_LOCAL_PATH" ]] || { log "no CSV dataset configured, skipping upload"; return 0; }
  log "uploading $CSV_LOCAL_PATH from old EC2 to s3://$OLD_BUCKET/$CSV_S3_KEY..."
  ssh_old "aws s3 cp '$CSV_LOCAL_PATH' 's3://$OLD_BUCKET/$CSV_S3_KEY' --no-progress"
}

copy_one() {
  local key="$1" old_size new_size
  log "copying s3://$OLD_BUCKET/$key -> s3://$NEW_BUCKET_NAME/$key (server-side)..."
  aws s3 cp "s3://$OLD_BUCKET/$key" "s3://$NEW_BUCKET_NAME/$key" \
    --profile "$NEW_PROFILE" --region "$NEW_REGION" --no-progress
  old_size="$(aws s3api head-object --bucket "$OLD_BUCKET" --key "$key" --profile "$OLD_PROFILE" --query ContentLength --output text)"
  new_size="$(aws s3api head-object --bucket "$NEW_BUCKET_NAME" --key "$key" --profile "$NEW_PROFILE" --region "$NEW_REGION" --query ContentLength --output text)"
  [[ "$old_size" == "$new_size" ]] || die "size mismatch after copy of $key: old=$old_size new=$new_size"
  log "copy verified for $key ($new_size bytes)"
}

copy_data() {
  load_outputs
  grant_old_bucket
  copy_one "$DATA_S3_KEY"
  [[ -z "$CSV_S3_KEY" ]] || copy_one "$CSV_S3_KEY"
  revoke_old_bucket
  trap - EXIT
}

wait_bootstrap() {
  load_outputs
  log "waiting for EC2 bootstrap to finish..."
  for _ in $(seq 1 90); do
    if ssh_new "test -f /var/log/notorious-bootstrap.done" 2>/dev/null; then
      log "bootstrap done"; return 0
    fi
    sleep 20
  done
  die "bootstrap did not finish in 30 min - ssh in and check: sudo cloud-init status --long"
}

# Everything long-running after provisioning happens ON the new EC2 as a
# detached systemd unit (infra/remote-continue.sh), so this laptop only has
# to stay on for Terraform + the S3 copies, not the hours of unzip/restore.
write_rotation_env() {
  local tmp; tmp="$(mktemp)"
  {
    printf 'BUCKET=%q\n'        "$NEW_BUCKET_NAME"
    printf 'REGION=%q\n'        "$NEW_REGION"
    printf 'DATA_KEY=%q\n'      "$DATA_S3_KEY"
    printf 'JSON_NAME=%q\n'     "$JSON_FILE_NAME"
    printf 'CSV_KEY=%q\n'       "$CSV_S3_KEY"
    printf 'CSV_NAME=%q\n'      "$CSV_FILE_NAME"
    printf 'OLD_HOST=%q\n'      "$OLD_EC2_HOST"
    printf 'OLD_USER=%q\n'      "$OLD_EC2_USER"
    printf 'DUMP_CMD=%q\n'      "$OLD_DUMP_CMD"
  } > "$tmp"
  scp_new "$tmp" /opt/notorious/rotation.env
  rm -f "$tmp"
}

arm_remote() {
  load_outputs
  if ssh_new "sudo systemctl is-active notorious-continue" 2>/dev/null | grep -qx active; then
    log "notorious-continue is already running on the new EC2 - nothing to arm"
    return 0
  fi
  need go
  log "building ingest binaries..."
  (cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/notorious-ingest ./cmd/ingest)
  (cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/notorious-ingest-csv ./cmd/ingest_csv)
  scp_new /tmp/notorious-ingest /opt/notorious/ingest
  scp_new /tmp/notorious-ingest-csv /opt/notorious/ingest_csv
  scp_new backend/scripts/run_ingest_with_auto_resume.sh /opt/notorious/run_ingest_with_auto_resume.sh
  scp_new infra/remote-continue.sh /opt/notorious/remote-continue.sh
  scp_new "$OLD_EC2_SSH_KEY" /opt/notorious/aditya.pem
  write_rotation_env
  # The old EC2's key becomes root-owned immediately: the ubuntu user (which
  # runs the ingest binaries over third-party data) must not be able to read
  # a live production key. It is deleted by remote-continue.sh after the dump.
  ssh_new "chmod 600 /opt/notorious/rotation.env; \
    sudo chown root:root /opt/notorious/aditya.pem; sudo chmod 600 /opt/notorious/aditya.pem; \
    chmod +x /opt/notorious/ingest /opt/notorious/ingest_csv \
             /opt/notorious/run_ingest_with_auto_resume.sh /opt/notorious/remote-continue.sh"
  # Fail here, while the laptop is still present, if EC2->EC2 SSH is broken.
  if ! ssh_new "sudo ssh -i /opt/notorious/aditya.pem -o StrictHostKeyChecking=accept-new -o ConnectTimeout=10 \
      $OLD_EC2_USER@$OLD_EC2_HOST hostname" >/dev/null; then
    ssh_new "sudo rm -f /opt/notorious/aditya.pem" || true
    die "new EC2 cannot SSH to old EC2 - check the old security group allows SSH from the new EIP"
  fi
  log "launching detached continuation service on the new EC2..."
  if ! ssh_new "sudo systemctl reset-failed notorious-continue 2>/dev/null || true; \
      sudo systemd-run --unit=notorious-continue --collect /opt/notorious/remote-continue.sh; \
      sleep 5; sudo systemctl is-active notorious-continue"; then
    ssh_new "sudo rm -f /opt/notorious/aditya.pem" || true
    die "continuation service failed to start - check: journalctl -u notorious-continue on the new EC2"
  fi
  log "armed. The new EC2 now owns: download, unzip, CSV, Postgres restore, ingestion start."
  log "You can close this laptop. Check in later with: ./migrate.sh status"
}

start_csv() {
  load_outputs
  # The CSV binary is normally uploaded by arm_remote; build it only if missing.
  if ! ssh_new "test -x /opt/notorious/ingest_csv" 2>/dev/null; then
    need go
    log "building CSV ingest binary..."
    (cd backend && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/notorious-ingest-csv ./cmd/ingest_csv)
    ssh_new "sudo systemctl stop notorious-ingest-csv 2>/dev/null || true"
    scp_new /tmp/notorious-ingest-csv /opt/notorious/ingest_csv
  fi
  # Always refresh the runner so script fixes propagate on re-runs.
  scp_new backend/scripts/run_ingest_with_auto_resume.sh /opt/notorious/run_ingest_with_auto_resume.sh
  ssh_new "test -s '/data/$CSV_FILE_NAME'" \
    || die "CSV file /data/$CSV_FILE_NAME not on the new EC2 yet - let notorious-continue finish first (./migrate.sh status)"
  ssh_new "chmod +x /opt/notorious/ingest_csv /opt/notorious/run_ingest_with_auto_resume.sh \
    && sudo sed -i -E 's|--file=/data/[^[:space:]]*|--file=/data/$CSV_FILE_NAME|' /etc/systemd/system/notorious-ingest-csv.service \
    && sudo systemctl daemon-reload && sudo systemctl enable notorious-ingest-csv \
    && sudo systemctl restart notorious-ingest-csv && touch /opt/notorious/csv-ingest.started"
  log "CSV ingestion started. Check with: ./migrate.sh status"
}

status() {
  load_outputs
  ssh_new "for u in notorious-continue notorious-ingest notorious-ingest-csv; do echo \"-- \$u\"; \
      sudo systemctl show \$u -p ActiveState,SubState,Result --no-pager; done; \
    echo --- last log lines ---; \
    ls -t ~/notorious/backend/ingest_*.log 2>/dev/null | head -1 | xargs -r tail -5; \
    echo --- opensearch doc counts ---; \
    OSU=\$(grep '^OPENSEARCH_MASTER_USER=' ~/notorious/backend/.env | cut -d= -f2-); \
    OSP=\$(grep '^OPENSEARCH_MASTER_PASSWORD=' ~/notorious/backend/.env | cut -d= -f2-); \
    OSE=\$(grep '^OPENSEARCH_ENDPOINT=' ~/notorious/backend/.env | cut -d= -f2-); \
    curl -s -u \"\$OSU:\$OSP\" \"\$OSE/_cat/indices/people-*?h=index,docs.count,store.size\" || true"
}

deploy_api() {
  load_outputs
  need gh
  # A still-running (or never-started) unit also reports Result=success, so
  # require inactive + success + at least one completion line in the logs.
  check_ingest_done() { # unit log-glob label
    local state logs
    state="$(ssh_new "sudo systemctl show $1 -p ActiveState,Result --value | paste -sd' '")"
    logs="$(ssh_new "grep -l 'ingestion completed successfully' ~/notorious/backend/$2 2>/dev/null | wc -l")"
    if [[ "$state" != "inactive success" || "$logs" -lt 1 ]]; then
      [[ "${FORCE:-}" == "1" ]] || die "$3 not finished (state='$state', success logs=$logs). Re-run finish later, or FORCE=1 to deploy anyway"
      log "WARN: FORCE=1 - deploying despite $3 state='$state', success logs=$logs"
    fi
  }
  check_ingest_done notorious-ingest 'ingest_json_*.log' "JSON ingestion"
  # If a CSV dataset is configured, its ingestion MUST have been run - an
  # absent marker means the operator forgot start-csv, not that it finished.
  if [[ -n "$CSV_S3_KEY" ]]; then
    if ! ssh_new "test -f /opt/notorious/csv-ingest.started" 2>/dev/null && [[ "${FORCE:-}" != "1" ]]; then
      die "CSV dataset is configured but start-csv was never run - run ./migrate.sh start-csv first (or FORCE=1 to cut over without the $CSV_INDEX index)"
    fi
    check_ingest_done notorious-ingest-csv 'ingest_csv_*.log' "CSV ingestion"
  fi
  if [[ "${EXPECTED_DOC_COUNT:-}" != "" ]]; then
    local count
    count="$(os_count "$(tf_out opensearch_index)")"
    [[ "$count" == "$EXPECTED_DOC_COUNT" ]] || die "doc count $count != expected $EXPECTED_DOC_COUNT"
    log "doc count verified: $count"
  fi
  if [[ "${CSV_EXPECTED_DOC_COUNT:-}" != "" ]] && ssh_new "test -f /opt/notorious/csv-ingest.started" 2>/dev/null; then
    local csv_count
    csv_count="$(os_count "$CSV_INDEX")"
    [[ "$csv_count" == "$CSV_EXPECTED_DOC_COUNT" ]] || die "CSV doc count $csv_count != expected $CSV_EXPECTED_DOC_COUNT"
    log "CSV doc count verified: $csv_count"
  fi
  log "updating GitHub Actions secrets for the new account..."
  # Secrets go via stdin, never -b argv (visible in the process list).
  printf %s "$NEW_IP" | gh secret set EC2_HOST
  gh secret set EC2_SSH_KEY < "$NEW_EC2_SSH_KEY"
  printf %s "$(tf_out opensearch_endpoint)" | gh secret set OPENSEARCH_ENDPOINT
  printf %s "$(tf_out opensearch_index)" | gh secret set OPENSEARCH_INDEX
  printf %s "$(tf_out opensearch_index),$CSV_INDEX" | gh secret set OPENSEARCH_INDICES
  printf %s "$(tf_out opensearch_master_user)" | gh secret set OPENSEARCH_MASTER_USER
  tf_out opensearch_master_password | gh secret set OPENSEARCH_MASTER_PASSWORD
  tf_out database_url | gh secret set DATABASE_URL
  tf_out jwt_secret | gh secret set JWT_SECRET
  printf %s "$NEW_REGION" | gh secret set AWS_REGION
  printf %s "$NEW_BUCKET_NAME" | gh secret set S3_UPLOAD_BUCKET
  log "triggering deploy workflow..."
  local dispatched_at run_id=""
  dispatched_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  gh workflow run deploy-backend.yml
  for _ in $(seq 1 30); do
    run_id="$(gh run list --workflow deploy-backend.yml --limit 5 \
      --json databaseId,createdAt \
      --jq "[.[] | select(.createdAt >= \"$dispatched_at\")] | .[0].databaseId // empty")"
    [[ -n "$run_id" ]] && break
    sleep 10
  done
  [[ -n "$run_id" ]] || die "deploy workflow run never appeared in gh run list"
  gh run watch "$run_id" --exit-status
  log "waiting for API health on the new EC2..."
  for _ in $(seq 1 30); do
    if curl -fsS -H "Host: $DOMAIN" "http://$NEW_IP/health" >/dev/null 2>&1; then
      log "API healthy at http://$NEW_IP/health (Host: $DOMAIN)"; return 0
    fi
    sleep 10
  done
  die "API did not become healthy - check: ssh in, docker logs notorious-backend"
}

copy_certs() {
  load_outputs
  log "copying /etc/letsencrypt from old EC2 (avoids re-issuance / rate limits)..."
  # Extract into a staging dir and verify the contents before touching / -
  # never untar a stream from another host straight into the root filesystem.
  ssh_old "sudo tar -czf - -C / etc/letsencrypt" \
    | ssh_new "rm -rf /tmp/le-stage && mkdir -p /tmp/le-stage && tar -xzf - -C /tmp/le-stage \
        && test -d /tmp/le-stage/etc/letsencrypt \
        && test -z \"\$(find /tmp/le-stage -not -path '/tmp/le-stage/etc/letsencrypt*' -not -name le-stage -not -path '/tmp/le-stage/etc' -print -quit)\" \
        && sudo cp -a /tmp/le-stage/etc/letsencrypt /etc/ && rm -rf /tmp/le-stage" \
    || log "WARN: cert copy failed - certbot will issue fresh in cutover (fine after DNS flip)"
}

cutover() {
  load_outputs
  if [[ "${YES:-}" != "1" ]]; then
    echo
    echo ">>> At your registrar, set the A record for $DOMAIN to:  $NEW_IP  (TTL 60s if possible)"
    echo ">>> The old EC2 keeps serving until DNS propagates. Rollback = point the record back."
    read -r -p ">>> Press Enter once the A record is updated..."
  fi
  need dig
  log "waiting for $DOMAIN to resolve to $NEW_IP..."
  for _ in $(seq 1 60); do
    if dig +short @1.1.1.1 "$DOMAIN" A | grep -qx "$NEW_IP"; then break; fi
    sleep 20
  done
  dig +short @1.1.1.1 "$DOMAIN" A | grep -qx "$NEW_IP" || die "DNS still not pointing at $NEW_IP after 20 min"
  log "installing SSL cert and enabling HTTPS..."
  ssh_new "sudo certbot --nginx -d $DOMAIN --reinstall --redirect --non-interactive --agree-tos || \
           sudo certbot --nginx -d $DOMAIN --redirect --non-interactive --agree-tos"
  ssh_new "sudo nginx -t && sudo systemctl reload nginx"
  for _ in $(seq 1 30); do
    if curl -fsS "https://$DOMAIN/health" >/dev/null 2>&1; then
      log "CUTOVER COMPLETE: https://$DOMAIN/health is live on the new account"
      log "Keep the old account running 24-48h, then tear it down (see infra/README.md)."
      return 0
    fi
    sleep 10
  done
  die "HTTPS health check failed after cutover - check nginx/certbot on the new EC2"
}

# ------------------------------------------------------------------ main

usage() {
  cat <<'EOF'
Usage: ./migrate.sh <command>

  start     preflight + provision + upload-csv + copy-data + setup + arm.
            After arm, ALL long work runs on the new EC2 - close the laptop.
  start-csv start the Delhi-NCR CSV ingestion service (after JSON ingest)
  status    continuation + ingestion progress + OpenSearch doc counts
  finish    deploy-api + copy-certs + cutover (run after ingestion completes)
  <phase>   preflight | provision | upload-csv | copy-data | setup | arm |
            start-csv | deploy | certs | cutover
EOF
}

cmd="${1:-}"
case "$cmd" in
  start)        preflight; provision; upload_csv; copy_data; wait_bootstrap; arm_remote ;;
  status)       status ;;
  finish)       deploy_api; copy_certs; cutover ;;
  preflight)    preflight ;;
  provision)    provision ;;
  upload-csv)   upload_csv ;;
  copy-data)    copy_data ;;
  setup)        wait_bootstrap ;;
  arm)          wait_bootstrap; arm_remote ;;
  start-csv)    start_csv ;;
  deploy)       deploy_api ;;
  certs)        copy_certs ;;
  cutover)      cutover ;;
  *)            usage; exit 1 ;;
esac
