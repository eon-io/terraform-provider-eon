# Require approval before restoring resources
resource "eon_action_approval_rule" "restore" {
  operation              = "RESTORE_RESOURCE"
  required_approvals     = 1
  approval_window_hours  = 24
  execution_window_hours = 12
  description            = "Require approval before restoring production resources"
  exempt_api_credentials = false

  resource_selector = {
    resource_selection_mode = "ALL"
  }
}
