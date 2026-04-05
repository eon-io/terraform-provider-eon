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
  volume_id   = "2ff45ed6-1b8d-5f67-b931-a28482f16d9e" # EBS volume ID in Eon
}
