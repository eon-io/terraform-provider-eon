variable "eon_endpoint" {
  description = "Eon API endpoint URL"
  type        = string
  default     = "https://your-eon-endpoint.com"
}

variable "eon_client_id" {
  description = "Eon API client ID"
  type        = string
  sensitive   = true
}

variable "eon_client_secret" {
  description = "Eon API client secret"
  type        = string
  sensitive   = true
}

variable "eon_project_id" {
  description = "Eon project ID"
  type        = string
} 