# Require approval before restoring production resources
resource "eon_action_approval_rule" "restore_production" {
  operation              = "RESTORE_RESOURCE"
  required_approvals     = 1
  approval_window_hours  = 24
  execution_window_hours = 48
  description            = "Require approval for production restores"

  resource_selector = {
    resource_selection_mode = "CONDITIONAL"

    expression = {
      environment = {
        operator     = "IN"
        environments = ["PROD"]
      }
    }
  }
}
