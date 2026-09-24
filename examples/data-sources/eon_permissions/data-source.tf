# Example: List available permissions for custom roles
data "eon_permissions" "all" {}

# Example: Use a permission when creating a custom role
resource "eon_role" "inventory_viewer" {
  name = "inventory-viewer"

  permission_grants = [
    {
      permission = [
        for p in data.eon_permissions.all.permissions : p.permission_type
        if p.permission_type == "inventory.view"
      ][0]
    }
  ]
}

output "permission_types" {
  description = "Available permission type identifiers"
  value       = [for p in data.eon_permissions.all.permissions : p.permission_type]
}
