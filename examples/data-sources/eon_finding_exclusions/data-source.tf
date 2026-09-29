# All finding exclusions in the project
data "eon_finding_exclusions" "all" {}

# Only account-wide malware exclusions
data "eon_finding_exclusions" "account_wide_malware" {
  account_wide = true
  detectors    = ["MALWARE"]
}

output "finding_exclusion_ids" {
  value = [for exclusion in data.eon_finding_exclusions.all.finding_exclusions : exclusion.id]
}
