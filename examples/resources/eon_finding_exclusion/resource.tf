# Suppress malware findings for files under /var/cache on every resource.
resource "eon_finding_exclusion" "cache" {
  value    = "/var/cache"
  type     = "PATH"
  detector = "MALWARE"
}

# Suppress data-anomaly findings for one table on a single resource.
resource "eon_finding_exclusion" "orders" {
  resource_id = "1ee34dc5-0a7c-4e56-a820-917371e05c8d"
  value       = "orders"
  type        = "TABLE"
  detector    = "DATA_ANOMALY"
}
