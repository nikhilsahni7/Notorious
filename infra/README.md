# Infrastructure (Terraform)

Provisions one full notorious stack in a fresh AWS account for the monthly
rotation. **To run a rotation, follow [../ROTATION.md](../ROTATION.md)** —
this file only describes what the code creates and why.

## What gets created

| Resource | Notes |
|---|---|
| S3 bucket | Transfer staging; versioned, SSE-S3, all public access blocked |
| OpenSearch domain | `OpenSearch_3.7`, 3× `r8g.2xlarge.search` data + 3× `m7g.large.search` masters, 1500GB gp3/node (matches production). VPC-only — reachable only from the EC2 security group; the old domain was public, this is a deliberate hardening |
| EC2 + Elastic IP | Ubuntu 24.04, `t3.large`, IAM instance profile (no access keys on the box, IMDSv2 required), 80GB root + 700GB gp3 data volume at `/data` |
| Secrets Manager secret | DB/JWT/OpenSearch/VAPID secrets, fetched by bootstrap via the instance role — never present in user-data |
| Service-linked role | `AWSServiceRoleForAmazonOpenSearchService`, required once per account before the first VPC domain |

## Design rules (keep it this simple)

- **Few files, boring resources.** `main.tf` (resources), `variables.tf`
  (knobs), `outputs.tf` (what `migrate.sh` consumes). No modules — one stack
  does not justify them.
- **Change sizing in `terraform.tfvars`, never in `main.tf`.** The tfvars
  file is gitignored (it also carries secrets).
- **Local state, archived per rotation.** `migrate.sh provision` detects a
  state file belonging to a different account and moves it to
  `state-archive/<account>-<timestamp>/` before applying. State contains
  secret values: gitignored, back it up encrypted.
- **Bootstrap fails loudly.** `bootstrap.sh.tftpl` exits non-zero if the
  data volume is missing rather than letting a 479GB download fill the root
  disk, and only then touches `/var/log/notorious-bootstrap.done`.
- **Ingestion is systemd, not tmux.** `notorious-ingest` /
  `notorious-ingest-csv` units with auto-resume; the one-shot
  `notorious-continue` unit (from `remote-continue.sh`) performs download,
  unzip, Postgres restore and starts ingestion without any laptop attached.

## Verify changes

```bash
terraform -chdir=infra fmt -check
terraform -chdir=infra init -backend=false && terraform -chdir=infra validate
```
