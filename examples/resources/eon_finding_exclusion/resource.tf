# Stop malware detection on a build cache, on every resource in the account
resource "eon_finding_exclusion" "build_cache" {
  path     = "/var/cache/build/"
  type     = "PATH"
  detector = "MALWARE"
}

# Stop data-anomaly findings for one table on one database resource
resource "eon_finding_exclusion" "audit_log_table" {
  resource_id = "1ee34dc5-0a7c-4e56-a820-917371e05c8d"
  path        = "audit_log"
  type        = "TABLE"
  detector    = "DATA_ANOMALY"
}
