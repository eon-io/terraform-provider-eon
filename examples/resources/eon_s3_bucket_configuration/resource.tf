terraform {
  required_providers {
    eon = {
      source = "eon-io/eon"
    }
  }
}

# Example: Read the bucket from S3 Inventory reports and use S3 change events between scans.
resource "eon_s3_bucket_configuration" "bucket" {
  resource_id = "1ee34dc5-0a7c-4e56-a820-917371e05c8d" # S3 bucket resource ID in Eon

  scan_mode             = "INVENTORY"
  change_detection_mode = "SCAN_AND_CHANGE_EVENTS"
}
