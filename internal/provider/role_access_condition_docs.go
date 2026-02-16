package provider

// Shared MarkdownDescription strings for access_conditions[].expression.
// Used by both eon_role (resource) and eon_roles (data source) so role.md and roles.md
// show identical descriptions for the expression nested schema.
var (
	// Condition block descriptions (top-level expression and group.operands)
	RoleExprDescEnvironment       = "Environment condition"
	RoleExprDescResourceType      = "Resource type condition"
	RoleExprDescDataClasses       = "Data classes condition"
	RoleExprDescTagKeys           = "Tag keys condition"
	RoleExprDescTagKeyValues      = "Tag key-value pairs condition"
	RoleExprDescApps              = "Apps condition"
	RoleExprDescCloudProvider     = "Cloud provider condition"
	RoleExprDescAccountID         = "Account ID condition"
	RoleExprDescSourceRegion      = "Source region condition"
	RoleExprDescVPC               = "VPC condition"
	RoleExprDescSubnets           = "Subnets condition"
	RoleExprDescResourceGroupName = "Resource group name condition"
	RoleExprDescResourceName      = "Resource name condition"
	RoleExprDescResourceID        = "Resource ID condition"
	RoleExprDescGroup             = "Group condition with logical operator and operands"

	// Operator / field descriptions (expression level)
	RoleExprDescOperatorINorNOTIN      = "Operator: IN or NOT_IN"
	RoleExprDescOperatorContains       = "Operator: CONTAINS_ANY_OF, CONTAINS_NONE_OF, or CONTAINS_ALL_OF"
	RoleExprDescOperatorLogical        = "Logical operator: AND or OR"
	RoleExprDescListEnvironments       = "List of environments"
	RoleExprDescListResourceTypes      = "List of resource types"
	RoleExprDescListDataClasses        = "List of data classes"
	RoleExprDescListTagKeysMatch       = "List of tag keys to match"
	RoleExprDescListTagKeyValuesMatch  = "List of tag key-value pairs to match"
	RoleExprDescTagKey                 = "Tag key"
	RoleExprDescTagValue               = "Tag value"
	RoleExprDescListApps               = "List of apps"
	RoleExprDescListCloudProviders     = "List of cloud providers"
	RoleExprDescListAccountIDs         = "List of account IDs"
	RoleExprDescListSourceRegions      = "List of source regions"
	RoleExprDescListVPCs               = "List of VPCs"
	RoleExprDescListSubnets            = "List of subnets"
	RoleExprDescListResourceGroupNames = "List of resource group names"
	RoleExprDescListResourceNames      = "List of resource names"
	RoleExprDescListResourceIDs        = "List of resource IDs"
	RoleExprDescListOperands           = "List of nested conditions"

	// access_conditions[].expression block (used by both eon_role and eon_roles)
	RoleExprDescExpression = "Conditional expression that defines which resources this condition applies to. Same structure as backup policy resource_selector.expression (environment, resource_type, group, data_classes, tag_keys, tag_key_values, etc.)."
)
