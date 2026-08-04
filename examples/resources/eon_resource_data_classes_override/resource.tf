terraform {
  required_providers {
    eon = {
      source = "eon-io/eon"
    }
  }
}

# Example: Override auto-classified data classes for an inventory resource
resource "eon_resource_data_classes_override" "demo" {
  resource_id  = "1ee34dc5-0a7c-4e56-a820-917371e05c8d"
  data_classes = ["PII", "PCI"]
}
