# Example: List all identity providers
data "eon_idps" "all" {}

# Example: Use an IdP ID when creating an IDP group mapping
# resource "eon_idp_group" "admins" {
#   idp_id            = data.eon_idps.all.idps[0].id
#   provider_group_id = "okta-admins-group-id"
#   role_ids          = [eon_role.admin.id]
# }

output "idps_count" {
  description = "Total number of identity providers"
  value       = length(data.eon_idps.all.idps)
}

output "idp_ids" {
  description = "Identity provider IDs keyed by display name"
  value = {
    for idp in data.eon_idps.all.idps :
    idp.provider_name => idp.id
  }
}
