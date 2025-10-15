variable "client_id" {
  type      = string
  default   = "Eon API client ID"
  sensitive = true
  ephemeral = true
}

variable "client_secret" {
  type      = string
  default   = "Eon API client secret"
  sensitive = true
  ephemeral = true
}

provider "eon" {
  endpoint      = "https://your-eon-endpoint.co"
  client_id     = var.client_id
  client_secret = var.client_secret
  project_id    = "Eon project ID"
}