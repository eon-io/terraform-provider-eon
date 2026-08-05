terraform {
  required_providers {
    eon = {
      source = "eon-io/eon"
    }
  }
}

# Example: Exclude an inventory resource from future backups
resource "eon_resource_backup_exclusion" "demo" {
  resource_id = "1ee34dc5-0a7c-4e56-a820-917371e05c8d"
}
