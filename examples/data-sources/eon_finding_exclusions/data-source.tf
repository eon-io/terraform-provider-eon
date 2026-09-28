data "eon_finding_exclusions" "all" {}

output "finding_exclusion_ids" {
  value = [for exclusion in data.eon_finding_exclusions.all.exclusions : exclusion.id]
}
