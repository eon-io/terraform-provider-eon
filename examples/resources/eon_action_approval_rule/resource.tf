# Require approval before adding restore accounts
resource "eon_action_approval_rule" "add_restore_account" {
  operation              = "ADD_RESTORE_ACCOUNT"
  required_approvals     = 1
  approval_window_hours  = 24
  execution_window_hours = 12
  description            = "Require approval to connect restore accounts"
  exempt_api_credentials = false
}
