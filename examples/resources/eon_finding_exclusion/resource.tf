# Stop the malware detector from reporting findings under a scratch directory on one resource
resource "eon_finding_exclusion" "scratch_dir" {
  resource_id = "12345678-1234-1234-1234-123456789012"
  value       = "/data/tmp"
  type        = "PATH"
  detector    = "MALWARE"
}

# Stop the data anomaly detector from reporting findings for a staging table on every resource
resource "eon_finding_exclusion" "staging_table" {
  value    = "staging_events"
  type     = "TABLE"
  detector = "DATA_ANOMALY"
}
