# Example: List configured identity providers
data "eon_idps" "all" {}

# Example: Use an IdP ID when creating an IDP group role assignment
resource "eon_idp_group" "admins" {
  idp_id            = data.eon_idps.all.idps[0].id
  provider_group_id = "okta-admins"
  role_ids          = [data.eon_builtin_roles.builtin.global_admin]
}

data "eon_builtin_roles" "builtin" {}

output "idp_ids" {
  description = "Configured identity provider IDs"
  value       = [for i in data.eon_idps.all.idps : i.id]
}

output "idps_by_name" {
  description = "Identity providers keyed by display name"
  value = {
    for i in data.eon_idps.all.idps :
    i.provider_name => i.id
  }
}
