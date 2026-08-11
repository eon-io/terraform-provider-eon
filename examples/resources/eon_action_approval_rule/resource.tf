# Require approval before adding a restore account
resource "eon_action_approval_rule" "add_restore_account" {
  operation              = "ADD_RESTORE_ACCOUNT"
  required_approvals     = 1
  approval_window_hours  = 24
  execution_window_hours = 4
  description            = "Require approval before connecting a restore account"
}
