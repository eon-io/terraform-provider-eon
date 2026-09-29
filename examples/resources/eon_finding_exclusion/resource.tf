# Stop malware detection on a build cache, on every resource in the account
resource "eon_finding_exclusion" "build_cache" {
  scope    = "ACCOUNT"
  value    = "/var/cache/build/"
  type     = "PATH"
  detector = "MALWARE"
}

# Stop ransomware-behavior findings for an application's rotating logs on one EC2 instance
resource "eon_finding_exclusion" "app_logs" {
  scope                = "RESOURCE"
  provider_resource_id = "i-0abc123def4567890"
  value                = "/var/log/app/"
  type                 = "PATH"
  detector             = "RANSOMWARE_BEHAVIOR"
}
