terraform {
  required_version = ">= 1.6"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  # Local state. NOTE: state contains secret values - never commit it
  # (covered by the repo .gitignore). Keep it on your machine or move it to
  # an encrypted S3 backend you control across rotations.
}

provider "aws" {
  region = var.aws_region
}

data "aws_caller_identity" "current" {}
data "aws_vpc" "default" { default = true }

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

data "aws_ssm_parameter" "ubuntu_ami" {
  name = "/aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp3/ami-id"
}

locals {
  name = var.project
  tags = { Project = var.project, ManagedBy = "terraform" }
}

# ---------------------------------------------------------------- S3

resource "aws_s3_bucket" "data" {
  bucket = var.bucket_name
  tags   = local.tags
}

resource "aws_s3_bucket_public_access_block" "data" {
  bucket                  = aws_s3_bucket.data.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_versioning" "data" {
  bucket = aws_s3_bucket.data.id
  versioning_configuration { status = "Enabled" }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "data" {
  bucket = aws_s3_bucket.data.id
  rule {
    apply_server_side_encryption_by_default { sse_algorithm = "AES256" }
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "data" {
  bucket = aws_s3_bucket.data.id
  rule {
    id     = "abort-incomplete-multipart"
    status = "Enabled"
    filter {}
    abort_incomplete_multipart_upload { days_after_initiation = 3 }
  }
}

# ---------------------------------------------------------------- Secrets

resource "aws_secretsmanager_secret" "app" {
  name = "${local.name}-app"
  tags = local.tags
}

resource "aws_secretsmanager_secret_version" "app" {
  secret_id = aws_secretsmanager_secret.app.id
  secret_string = jsonencode({
    database_name          = var.database_name
    database_user          = var.database_user
    database_password      = var.database_password
    jwt_secret             = var.jwt_secret
    opensearch_master_user = var.opensearch_master_user
    opensearch_master_pass = var.opensearch_master_password
    vapid_public_key       = var.vapid_public_key
    vapid_private_key      = var.vapid_private_key
    vapid_subject          = var.vapid_subject
  })
}

# ---------------------------------------------------------------- IAM (EC2)

resource "aws_iam_role" "ec2" {
  name = "${local.name}-ec2"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
  tags = local.tags
}

resource "aws_iam_role_policy" "ec2" {
  name = "${local.name}-ec2"
  role = aws_iam_role.ec2.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["s3:ListBucket"]
        Resource = [aws_s3_bucket.data.arn]
      },
      {
        Effect   = "Allow"
        Action   = ["s3:GetObject"]
        Resource = ["${aws_s3_bucket.data.arn}/*"]
      },
      {
        # App uploads only ever go under this prefix (S3_UPLOAD_PREFIX).
        Effect   = "Allow"
        Action   = ["s3:PutObject"]
        Resource = ["${aws_s3_bucket.data.arn}/ingest/raw/*"]
      },
      {
        Effect   = "Allow"
        Action   = ["secretsmanager:GetSecretValue"]
        Resource = [aws_secretsmanager_secret.app.arn]
      },
    ]
  })
}

resource "aws_iam_instance_profile" "ec2" {
  name = "${local.name}-ec2"
  role = aws_iam_role.ec2.name
}

# ---------------------------------------------------------------- Network

resource "aws_security_group" "ec2" {
  name   = "${local.name}-ec2"
  vpc_id = data.aws_vpc.default.id
  tags   = local.tags

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.ssh_cidr]
  }
  ingress {
    description = "HTTP"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  ingress {
    description = "HTTPS"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "opensearch" {
  name   = "${local.name}-opensearch"
  vpc_id = data.aws_vpc.default.id
  tags   = local.tags

  ingress {
    description     = "HTTPS from the EC2 instance only"
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = [aws_security_group.ec2.id]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# ---------------------------------------------------------------- OpenSearch

# Required once per account before the first VPC domain can be created.
resource "aws_iam_service_linked_role" "opensearch" {
  aws_service_name = "opensearchservice.amazonaws.com"
}

resource "aws_opensearch_domain" "main" {
  depends_on     = [aws_iam_service_linked_role.opensearch]
  domain_name    = "${local.name}-search"
  engine_version = var.opensearch_engine_version
  tags           = local.tags

  cluster_config {
    instance_type            = var.opensearch_instance_type
    instance_count           = var.opensearch_node_count
    dedicated_master_enabled = true
    dedicated_master_type    = var.opensearch_master_instance_type
    dedicated_master_count   = var.opensearch_master_count
    zone_awareness_enabled   = var.opensearch_node_count > 1

    dynamic "zone_awareness_config" {
      for_each = var.opensearch_node_count > 1 ? [1] : []
      content {
        availability_zone_count = min(var.opensearch_node_count, 3)
      }
    }
  }

  ebs_options {
    ebs_enabled = true
    volume_type = "gp3"
    volume_size = var.opensearch_volume_gb
  }

  encrypt_at_rest { enabled = true }
  node_to_node_encryption { enabled = true }

  domain_endpoint_options {
    enforce_https       = true
    tls_security_policy = "Policy-Min-TLS-1-2-2019-07"
  }

  advanced_security_options {
    enabled                        = true
    internal_user_database_enabled = true
    master_user_options {
      master_user_name     = var.opensearch_master_user
      master_user_password = var.opensearch_master_password
    }
  }

  vpc_options {
    subnet_ids         = slice(data.aws_subnets.default.ids, 0, var.opensearch_node_count > 1 ? min(var.opensearch_node_count, 3) : 1)
    security_group_ids = [aws_security_group.opensearch.id]
  }

  # Fine-grained access control (master user) is the auth layer; the app does
  # not SigV4-sign requests, so this policy must stay permissive at the
  # resource level. Network access is restricted by the VPC security group
  # above (443 from the EC2 security group only). Do not "tighten" the
  # Principal here without first adding es:* to the EC2 role + SigV4 signing.
  access_policies = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { AWS = "*" }
      Action    = "es:*"
      Resource  = "arn:aws:es:${var.aws_region}:${data.aws_caller_identity.current.account_id}:domain/${local.name}-search/*"
    }]
  })
}

# ---------------------------------------------------------------- EC2

resource "aws_key_pair" "main" {
  key_name   = "${local.name}-key"
  public_key = file(pathexpand(var.ssh_public_key_path))
  tags       = local.tags
}

resource "aws_instance" "app" {
  ami                    = data.aws_ssm_parameter.ubuntu_ami.value
  instance_type          = var.ec2_instance_type
  key_name               = aws_key_pair.main.key_name
  subnet_id              = data.aws_subnets.default.ids[0]
  vpc_security_group_ids = [aws_security_group.ec2.id]
  iam_instance_profile   = aws_iam_instance_profile.ec2.name
  tags                   = merge(local.tags, { Name = "${local.name}-app" })

  # IMDSv2 only: the instance role can read every app secret, so tokenless
  # metadata access (SSRF/RCE -> credential theft) must not work.
  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 1
  }

  root_block_device {
    volume_type = "gp3"
    volume_size = var.ec2_root_volume_gb
    encrypted   = true
  }

  user_data = templatefile("${path.module}/bootstrap.sh.tftpl", {
    aws_region       = var.aws_region
    domain           = var.domain
    bucket           = aws_s3_bucket.data.bucket
    opensearch_host  = aws_opensearch_domain.main.endpoint
    opensearch_index = var.opensearch_index
    secret_id        = aws_secretsmanager_secret.app.name
  })

  # Bootstrap reads the OpenSearch endpoint and the secret, so both must exist first.
  depends_on = [aws_secretsmanager_secret_version.app]
}

resource "aws_ebs_volume" "data" {
  availability_zone = aws_instance.app.availability_zone
  size              = var.ec2_data_volume_gb
  type              = "gp3"
  encrypted         = true
  tags              = merge(local.tags, { Name = "${local.name}-data" })
}

resource "aws_volume_attachment" "data" {
  device_name = "/dev/sdf"
  volume_id   = aws_ebs_volume.data.id
  instance_id = aws_instance.app.id
}

resource "aws_eip" "app" {
  instance = aws_instance.app.id
  domain   = "vpc"
  tags     = local.tags
}
