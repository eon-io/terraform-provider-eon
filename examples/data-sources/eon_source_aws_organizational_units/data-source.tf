# Example: Retrieve all source AWS organizational units
data "eon_source_aws_organizational_units" "all" {}

# Example: Output all source AWS organizational units information
output "all_source_aws_organizational_units" {
  description = "Information about all connected source AWS organizational units"
  value = {
    total_ous            = length(data.eon_source_aws_organizational_units.all.organizational_units)
    organizational_units = data.eon_source_aws_organizational_units.all.organizational_units
  }
}

# Example: Filter connected organizational units using locals
locals {
  connected_ous = [
    for ou in data.eon_source_aws_organizational_units.all.organizational_units :
    ou if ou.status == "CONNECTED"
  ]
}

# Output filtered results
output "connected_organizational_units" {
  description = "List of connected source AWS organizational units only"
  value       = local.connected_ous
}
