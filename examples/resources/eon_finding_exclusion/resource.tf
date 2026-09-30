# Suppress malware findings for a cache directory on one EC2 instance
resource "eon_finding_exclusion" "malware_cache" {
  scope                = "RESOURCE"
  provider_resource_id = "i-0abc123def4567890"
  value                = "/var/cache"
  type                 = "PATH"
  detector             = "MALWARE"
}
