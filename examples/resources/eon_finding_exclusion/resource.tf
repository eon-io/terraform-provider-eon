# Stop the malware detector from reporting findings under a cache directory on one EC2 instance
resource "eon_finding_exclusion" "instance_cache" {
  scope                = "RESOURCE"
  provider_resource_id = "i-0123456789abcdef0"
  type                 = "PATH"
  value                = "/var/cache/app"
  detector             = "MALWARE"
}

# Stop the data anomaly detector from reporting findings for a staging table on every resource in the account
resource "eon_finding_exclusion" "staging_table" {
  scope    = "ACCOUNT"
  type     = "TABLE"
  value    = "staging_events"
  detector = "DATA_ANOMALY"
}
