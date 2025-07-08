terraform {
  required_providers {
    eon = {
      source = "eon-io/eon"
    }
  }
}

provider "eon" {
  endpoint       = var.eon_endpoint
  client_id      = var.eon_client_id
  client_secret  = var.eon_client_secret
  project_id     = var.eon_project_id
  eon_account_id = var.eon_account_id
}