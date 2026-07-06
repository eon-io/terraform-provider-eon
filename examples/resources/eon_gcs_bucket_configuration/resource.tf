terraform {
  required_providers {
    eon = {
      source = "eon-io/eon"
    }
  }
}

# Example: Force GCS event notifications for scan detection and use auto full-scan selection.
resource "eon_gcs_bucket_configuration" "bucket" {
  resource_id = "1ee34dc5-0a7c-4e56-a820-917371e05c8d" # GCS bucket resource ID in Eon

  scan_detection_method = "FORCE_ENABLED"
  full_scan_method      = "AUTO"
}
