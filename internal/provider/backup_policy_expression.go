package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Nested group conditions are unrolled into the schema, so nesting depth is fixed up front.
// Depth 3 covers the console pattern: top AND → nested OR → nested AND with leaf conditions.
const backupPolicyMaxNestedGroupDepth = 3

// backupPolicyOperandLeafAttributes is every leaf condition an operand may carry.
func backupPolicyOperandLeafAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"resource_type": schema.SingleNestedAttribute{
			MarkdownDescription: "Resource type condition",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Operator: 'IN' or 'NOT_IN'",
					Required:            true,
				},
				"resource_types": schema.ListAttribute{
					MarkdownDescription: "List of resource types",
					ElementType:         types.StringType,
					Required:            true,
				},
			},
		},
		"environment": schema.SingleNestedAttribute{
			MarkdownDescription: "Environment condition",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Operator: 'IN' or 'NOT_IN'",
					Required:            true,
				},
				"environments": schema.ListAttribute{
					MarkdownDescription: "List of environments",
					ElementType:         types.StringType,
					Required:            true,
				},
			},
		},
		"tag_keys": schema.SingleNestedAttribute{
			MarkdownDescription: "Tag keys condition",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Operator: 'IN' or 'NOT_IN'",
					Required:            true,
				},
				"tag_keys": schema.ListAttribute{
					MarkdownDescription: "List of tag keys to match",
					ElementType:         types.StringType,
					Required:            true,
				},
			},
		},
		"tag_key_values": schema.SingleNestedAttribute{
			MarkdownDescription: "Tag key-value pairs condition",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Operator: 'IN' or 'NOT_IN'",
					Required:            true,
				},
				"tag_key_values": schema.ListNestedAttribute{
					MarkdownDescription: "List of tag key-value pairs to match",
					Required:            true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"key": schema.StringAttribute{
								MarkdownDescription: "Tag key",
								Required:            true,
							},
							"value": schema.StringAttribute{
								MarkdownDescription: "Tag value",
								Required:            true,
							},
						},
					},
				},
			},
		},
		"data_classes": schema.SingleNestedAttribute{
			MarkdownDescription: "Data classes condition",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Operator: 'CONTAINS' or 'NOT_CONTAINS'",
					Required:            true,
				},
				"data_classes": schema.ListAttribute{
					MarkdownDescription: "List of data classes",
					ElementType:         types.StringType,
					Required:            true,
				},
			},
		},
		"apps": schema.SingleNestedAttribute{
			MarkdownDescription: "Apps condition",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Operator: 'CONTAINS' or 'NOT_CONTAINS'",
					Required:            true,
				},
				"apps": schema.ListAttribute{
					MarkdownDescription: "List of apps",
					ElementType:         types.StringType,
					Required:            true,
				},
			},
		},
		"cloud_provider": schema.SingleNestedAttribute{
			MarkdownDescription: "Cloud provider condition",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Operator: 'IN' or 'NOT_IN'",
					Required:            true,
				},
				"cloud_providers": schema.ListAttribute{
					MarkdownDescription: "List of cloud providers",
					ElementType:         types.StringType,
					Required:            true,
				},
			},
		},
		"account_id": schema.SingleNestedAttribute{
			MarkdownDescription: "Account ID condition",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Operator: 'IN' or 'NOT_IN'",
					Required:            true,
				},
				"account_ids": schema.ListAttribute{
					MarkdownDescription: "List of account IDs",
					ElementType:         types.StringType,
					Required:            true,
				},
			},
		},
		"source_region": schema.SingleNestedAttribute{
			MarkdownDescription: "Source region condition",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Operator: 'IN' or 'NOT_IN'",
					Required:            true,
				},
				"source_regions": schema.ListAttribute{
					MarkdownDescription: "List of source regions",
					ElementType:         types.StringType,
					Required:            true,
				},
			},
		},
		"vpc": schema.SingleNestedAttribute{
			MarkdownDescription: "VPC condition",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Operator: 'IN' or 'NOT_IN'",
					Required:            true,
				},
				"vpcs": schema.ListAttribute{
					MarkdownDescription: "List of VPCs",
					ElementType:         types.StringType,
					Required:            true,
				},
			},
		},
		"subnets": schema.SingleNestedAttribute{
			MarkdownDescription: "Subnets condition",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Operator: 'CONTAINS' or 'NOT_CONTAINS'",
					Required:            true,
				},
				"subnets": schema.ListAttribute{
					MarkdownDescription: "List of subnets",
					ElementType:         types.StringType,
					Required:            true,
				},
			},
		},
		"resource_group_name": schema.SingleNestedAttribute{
			MarkdownDescription: "Resource group name condition",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Operator: 'CONTAINS' or 'NOT_CONTAINS'",
					Required:            true,
				},
				"resource_group_names": schema.ListAttribute{
					MarkdownDescription: "List of resource group names",
					ElementType:         types.StringType,
					Required:            true,
				},
			},
		},
		"resource_name": schema.SingleNestedAttribute{
			MarkdownDescription: "Resource name condition",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Operator: 'IN', 'NOT_IN', 'CONTAINS', 'NOT_CONTAINS', 'STARTS_WITH', 'NOT_STARTS_WITH', 'ENDS_WITH', or 'NOT_ENDS_WITH'",
					Required:            true,
				},
				"resource_names": schema.ListAttribute{
					MarkdownDescription: "List of resource names",
					ElementType:         types.StringType,
					Required:            true,
				},
			},
		},
		"resource_id": schema.SingleNestedAttribute{
			MarkdownDescription: "Resource ID condition",
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Operator: 'IN' or 'NOT_IN'",
					Required:            true,
				},
				"resource_ids": schema.ListAttribute{
					MarkdownDescription: "List of resource IDs",
					ElementType:         types.StringType,
					Required:            true,
				},
			},
		},
	}
}

// backupPolicyOperandAttributes returns leaf conditions plus nested group when depth allows it.
func backupPolicyOperandAttributes(depth int) map[string]schema.Attribute {
	attrs := backupPolicyOperandLeafAttributes()
	if depth > 1 {
		attrs["group"] = schema.SingleNestedAttribute{
			MarkdownDescription: fmt.Sprintf(
				"Nested group condition with logical operator and operands. Groups nest up to %d levels deep.",
				backupPolicyMaxNestedGroupDepth,
			),
			Optional: true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Logical operator: 'AND' or 'OR'",
					Required:            true,
				},
				"operands": schema.ListNestedAttribute{
					MarkdownDescription: "List of conditions",
					Required:            true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: backupPolicyOperandAttributes(depth - 1),
					},
				},
			},
		}
	}
	return attrs
}
