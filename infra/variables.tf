variable "aws_region" {
  type    = string
  default = "us-east-2"
}

variable "project" {
  type    = string
  default = "notorious"
}

variable "domain" {
  description = "Public hostname served by nginx on the EC2 instance."
  type        = string
  default     = "api.knotorious.us"
}

# --- EC2 ---

variable "ec2_instance_type" {
  # Matches the current production instance.
  type    = string
  default = "t3.large"
}

variable "ec2_root_volume_gb" {
  type    = number
  default = 80
}

variable "ec2_data_volume_gb" {
  # Holds the ~500GB JSON file locally for ingestion (known-working local path).
  type    = number
  default = 700
}

variable "ssh_public_key_path" {
  description = "Path to the SSH public key allowed on the new EC2 instance."
  type        = string
}

variable "ssh_cidr" {
  description = "CIDR allowed to SSH to the EC2 instance, e.g. \"203.0.113.10/32\"."
  type        = string

  validation {
    condition     = !contains(["0.0.0.0/0", "::/0"], var.ssh_cidr)
    error_message = "ssh_cidr must not be world-open; use your IP as a /32."
  }
}

# --- S3 (transfer staging + app bucket; accessed via AWS CLI / IAM role only) ---

variable "bucket_name" {
  description = "Globally unique bucket name in the NEW account, e.g. notorious-data-202611."
  type        = string
}

# --- OpenSearch ---

variable "opensearch_engine_version" {
  type    = string
  default = "OpenSearch_3.7"
}

variable "opensearch_instance_type" {
  type    = string
  default = "r8g.2xlarge.search"
}

variable "opensearch_master_instance_type" {
  type    = string
  default = "m7g.large.search"
}

variable "opensearch_master_count" {
  type    = number
  default = 3
}

variable "opensearch_node_count" {
  # Index uses 6 shards / 2 replicas (backend/internal/services/opensearch.go),
  # so 3 data nodes is the minimum sensible default.
  type    = number
  default = 3
}

variable "opensearch_volume_gb" {
  description = "EBS size per OpenSearch data node."
  type        = number
  default     = 1500
}

variable "opensearch_index" {
  type    = string
  default = "people-dev-0001"
}

variable "opensearch_master_user" {
  type    = string
  default = "notorious-admin"
}

# --- Secrets (supplied via terraform.tfvars, stored in Secrets Manager) ---

variable "opensearch_master_password" {
  type      = string
  sensitive = true
}

variable "database_name" {
  type    = string
  default = "notorious"
}

variable "database_user" {
  type    = string
  default = "notorious_user"
}

variable "database_password" {
  type      = string
  sensitive = true
}

variable "jwt_secret" {
  description = "Carry over the OLD value so existing logins/sessions keep working."
  type        = string
  sensitive   = true
}

variable "vapid_public_key" {
  type    = string
  default = ""
}

variable "vapid_private_key" {
  type      = string
  sensitive = true
  default   = ""
}

variable "vapid_subject" {
  type    = string
  default = ""
}
