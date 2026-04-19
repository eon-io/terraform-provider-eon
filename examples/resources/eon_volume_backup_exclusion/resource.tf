terraform {
  required_providers {
    eon = {
      source = "eon-io/eon"
    }
  }
}

# Example: Exclude a specific EBS volume from EC2 instance backups
resource "eon_volume_backup_exclusion" "data_volume" {
  resource_id = "1ee34dc5-0a7c-4e56-a820-917371e05c8d" # EC2 instance resource ID in Eon
  volume_id   = "vol-049df61146c064d1c"                # AWS EBS volume ID
}
