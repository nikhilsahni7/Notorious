output "new_account_id" {
  value = data.aws_caller_identity.current.account_id
}

output "bucket" {
  value = aws_s3_bucket.data.bucket
}

output "ec2_public_ip" {
  value = aws_eip.app.public_ip
}

output "aws_region" {
  value = var.aws_region
}

output "domain" {
  value = var.domain
}

output "ec2_instance_id" {
  description = "For console/debugging lookups; not consumed by migrate.sh."
  value       = aws_instance.app.id
}

output "opensearch_endpoint" {
  description = "Full HTTPS endpoint for OPENSEARCH_ENDPOINT."
  value       = "https://${aws_opensearch_domain.main.endpoint}"
}

output "opensearch_index" {
  value = var.opensearch_index
}

output "opensearch_master_user" {
  value = var.opensearch_master_user
}

output "opensearch_master_password" {
  sensitive = true
  value     = var.opensearch_master_password
}

output "jwt_secret" {
  sensitive = true
  value     = var.jwt_secret
}

output "database_url" {
  description = "Used by migrate.sh to update the GitHub Actions secrets."
  sensitive   = true
  value       = "postgresql://${var.database_user}:${urlencode(var.database_password)}@localhost:5432/${var.database_name}?sslmode=disable"
}
