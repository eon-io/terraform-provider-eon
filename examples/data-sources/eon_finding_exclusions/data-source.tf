data "eon_finding_exclusions" "all" {}

output "malware_exclusion_ids" {
  value = [for exclusion in data.eon_finding_exclusions.all.exclusions : exclusion.id if exclusion.detector == "MALWARE"]
}
