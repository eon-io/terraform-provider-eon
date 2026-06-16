output "discovery_role_arn" {
  description = "ARN of the management-account role Eon assumes to enumerate the OU."
  value       = local.discovery_role_arn
}

output "stack_set_name" {
  description = "Name of the CloudFormation StackSet that deploys the source role into member accounts."
  value       = aws_cloudformation_stack_set.source_members.name
}

output "stack_set_id" {
  description = "ID of the CloudFormation StackSet."
  value       = aws_cloudformation_stack_set.source_members.stack_set_id
}

output "organizational_units" {
  description = "Map of OU ID => Eon registration details."
  value = {
    for ou_id, ou in eon_source_aws_organizational_unit.this : ou_id => {
      id                             = ou.id
      name                           = ou.name
      status                         = ou.status
      provider_management_account_id = ou.provider_management_account_id
    }
  }
}
