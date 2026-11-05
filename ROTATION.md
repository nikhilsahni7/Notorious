# AWS Account Rotation Runbook

The only document you need for the monthly account change. Everything else
(`infra/`, `migrate.sh`) is implementation that this file tells you how to run.

## How it works (30 seconds)

```
OLD ACCOUNT                              NEW ACCOUNT
S3 (zip 110GB + CSV 17GB) ──S3→S3 copy──▶ S3 ──▶ EC2 /data (unzip → 479GB JSON)
EC2 Postgres ──SSH dump pipe (EC2→EC2)──▶ EC2 Postgres
                                          EC2 systemd: notorious-ingest      → people-dev-0001        (1,782,434,387 docs)
                                          EC2 systemd: notorious-ingest-csv  → people-delhi-ncr-0001  (98,090,524 docs)
api.knotorious.us DNS A record ── flipped LAST, after health checks pass
```

- Your laptop orchestrates only the short steps (Terraform, S3 copy). All
  long work (unzip, restore, ~24h ingestion) runs as systemd services **on
  the new EC2**. After `./migrate.sh start` returns, close the laptop.
- The old EC2 keeps serving live traffic until the DNS flip. Rollback at any
  point = point the A record back to the old IP.
- Ingestion reads the JSON from local disk, like the manual flow. The app's
  untested Go S3 code paths are deliberately not used; S3 is AWS-CLI staging.

## Files

| File | What it is | Committed? |
|---|---|---|
| `migrate.sh` | The orchestrator (`start` / `status` / `start-csv` / `finish`) | Yes |
| `infra/*.tf` | Terraform: EC2 + EIP, OpenSearch, S3, IAM role, Secrets Manager | Yes |
| `infra/bootstrap.sh.tftpl` | EC2 first-boot: Docker, Postgres, nginx, `.env`, ingest units | Yes |
| `infra/remote-continue.sh` | On-EC2 detached continuation (download→unzip→restore→ingest) | Yes |
| `migrate.env` | Your values: profiles, buckets, hosts, expected doc counts | **No — secret-adjacent, gitignored** |
| `infra/terraform.tfvars` | Sizing + carried-over secrets (JWT, DB/OpenSearch passwords) | **No — gitignored** |
| `infra/terraform.tfstate*` | Terraform state (contains secret values) | **No — gitignored, back it up encrypted** |

## One-time prerequisites

- AWS CLI profiles `old-account` and `new-account` (must be different
  accounts; the script refuses otherwise). Root keys work but are
  discouraged — an IAM admin user is better. Root MFA does **not** protect
  access keys.
- Tools: `terraform` (or tofu), `aws`, `jq`, `gh` (authenticated), `go`, `dig`.
- An SSH keypair for new EC2 instances (`~/.ssh/id_ed25519` by default).
- `gh auth login` done, so `finish` can set repo secrets and trigger deploys.

## Monthly procedure

### 1. Prepare (15 min)

1. Create the new AWS account; create access keys for both profiles.
2. A day ahead, set the `api.knotorious.us` A-record TTL to **60** at the
   registrar and note the current IP (= rollback target).
3. Edit `migrate.env` for this rotation:
   - `OLD_PROFILE` / `NEW_PROFILE`
   - `OLD_BUCKET` = the bucket created last rotation
   - `OLD_EC2_HOST` = last rotation's Elastic IP
   - `EXPECTED_DOC_COUNT` / `CSV_EXPECTED_DOC_COUNT` — refresh from the
     current OpenSearch (`_cat/indices`) **only if the source data changed**
4. Edit `infra/terraform.tfvars`:
   - `bucket_name` = new unique name (e.g. `notorious-data-<account-id>`)
   - `ssh_cidr` = your current IPv4 `/32` (`curl -4 ifconfig.me`)
   - Carry the secrets over **unchanged** (same `jwt_secret`, DB and
     OpenSearch passwords) or all users are logged out.
5. Sanity check:

```bash
./migrate.sh preflight
```

Expect: two different account IDs and `preflight OK`.

### 2. Start (laptop attended ~30–60 min, then free)

```bash
./migrate.sh start
```

This provisions the new account (OpenSearch is the slow part), uploads the
CSV to the old bucket, copies both objects cross-account (temporary,
auto-revoked bucket grant, byte-verified), waits for EC2 bootstrap, then arms
the detached `notorious-continue` service on the new EC2 and exits.

**After this returns, close the laptop.** The EC2 does the rest.

### 3. Ingestion (~24h+, unattended)

Check in anytime:

```bash
./migrate.sh status                 # services + live doc counts
./migrate.sh start-csv               # start Delhi-NCR CSV ingest once JSON
                                     # ingestion is done or well under way
```

Targets: `people-dev-0001` = 1,782,434,387 · `people-delhi-ncr-0001` = 98,090,524.

### 4. Cutover (~15 min, laptop attended)

```bash
./migrate.sh finish
```

It refuses unless both ingestions completed and doc counts match exactly.
It updates GitHub secrets from Terraform outputs, runs the deploy workflow,
health-checks the API on the new IP, copies the TLS certificates, then
**pauses**: go to your registrar, point the `api.knotorious.us` A record to
the printed Elastic IP, press Enter. It waits for DNS, installs SSL, and
prints `CUTOVER COMPLETE` when `https://api.knotorious.us/health` answers.

Verify manually: log in (should still work), run a normal search and a
Delhi-NCR search, open the admin panel.

### 5. Cleanup (after 24–48h)

1. Rollback window: keep the old EC2/OpenSearch/S3 running. Rollback = flip
   the A record back.
2. Tear down the old account. If it was Terraform-created (true from this
   rotation on), its state was auto-archived by the next `provision`:

   ```bash
   cp infra/state-archive/<old-account>-*/terraform.tfstate infra/terraform.tfstate
   AWS_PROFILE=old-account terraform -chdir=infra destroy
   ```

   (The pre-Terraform hand-built account must be deleted in the console:
   terminate EC2, delete the OpenSearch domain, empty + delete the bucket.)
3. Delete/deactivate the access keys you minted in step 1.
4. Back up `infra/terraform.tfstate` + `infra/terraform.tfvars` somewhere
   encrypted. Lose them and the next rotation becomes a fresh install
   (new JWT secret = everyone logged out).

## Re-running a single phase

Every phase is idempotent. If anything stops mid-way, re-run just that
phase instead of restarting:

```bash
./migrate.sh            # lists phases
./migrate.sh provision  # or: upload-csv | copy-data | setup | arm | deploy | certs | cutover
```

`arm` relaunches the on-EC2 continuation; completed steps (download, unzip,
restore) detect their outputs and skip.

## Troubleshooting (all hit during the first run, all fixed)

| Symptom | Cause / fix |
|---|---|
| OpenSearch create fails: "enable a service-linked role" | First VPC domain in a fresh account. Terraform creates `AWSServiceRoleForAmazonOpenSearchService`; if it already exists in the account (created outside Terraform), import it once: `terraform -chdir=infra import aws_iam_service_linked_role.opensearch <role-ARN>` |
| EC2 bootstrap failed, `/data` errors | Check `sudo tail /var/log/cloud-init-output.log` on the EC2. Bootstrap waits for the data volume; if it failed, re-run `sudo bash /var/lib/cloud/instance/scripts/part-001` |
| SSH to new EC2 times out | Your ISP IP changed. Update `ssh_cidr` in `terraform.tfvars` to the new `/32`, then `./migrate.sh provision` |
| `finish` refuses: ingestion not finished / doc count mismatch | Check `./migrate.sh status`. If the data file changed, update `EXPECTED_DOC_COUNT` in `migrate.env`; `FORCE=1 ./migrate.sh finish` exists but defeats the safety check |
| Copy fails with KMS error | Old object is SSE-KMS encrypted with an unshareable key. Re-encrypt it to SSE-S3 in the old account, or copy via the old EC2 |
| Deploy workflow watches the wrong run | Fixed: `finish` only watches runs created after it dispatched. If `gh` errors, update secrets by re-running `./migrate.sh deploy` |
| Laptop slept during `start` | Nothing corrupts. Re-run the interrupted phase (see above). After `arm` succeeds, laptop state no longer matters |
| New EC2 rebooted during unzip | `notorious-continue` is one-shot; relaunch with `./migrate.sh arm`. Ingestion services are boot-enabled and auto-resume from checkpoints |

## Security notes

- Never commit: `migrate.env`, `*.tfvars`, `*.tfstate*`, `*.pem`/`*.key`.
  Covered by the repo-root `.gitignore`; verified with `git check-ignore`.
- Secrets live in Secrets Manager in the new account and reach the EC2 via
  its IAM instance role (IMDSv2 enforced). They are never in user-data,
  Terraform outputs shown on screen, or GitHub except as Actions secrets
  (set via stdin, never command-line arguments).
- The EC2 gets no AWS access keys. Its role can read the data bucket and
  its own secret, and write only under `ingest/raw/`.
- The cross-account S3 grant exists only for the copy, is scoped to the
  operator identity and the exact object keys, self-expires after 6 hours,
  and is revoked (and verified revoked) on success, failure, or Ctrl-C.
- First SSH connections pin host keys on first use. Verify the new EC2's
  fingerprint (EC2 console → instance → console output shows SSH
  fingerprints) after the first connection, then set `MIGRATE_STRICT_SSH=1`
  in `migrate.env` so later runs refuse unknown keys.
- The old EC2's SSH key exists on the new EC2 only during `arm`, root-owned
  (`600`), and is deleted by the continuation script right after the
  Postgres dump (and on any failure path out of `arm`).
- OpenSearch is VPC-only, reachable from the EC2 security group only —
  an intentional hardening over the old public endpoint.
