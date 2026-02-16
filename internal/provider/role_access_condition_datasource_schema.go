package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// roleAccessConditionExpressionSchemaForDataSource returns the schema for
// access_conditions[].expression when used in the roles data source.
// Mirrors roleAccessConditionExpressionSchema() but uses datasource/schema.
func roleAccessConditionExpressionSchemaForDataSource() map[string]schema.Attribute {
	operandsAttr := roleAccessConditionOperandsSchemaForDataSource()
	return map[string]schema.Attribute{
		"environment": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescEnvironment,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":     schema.StringAttribute{Computed: true},
				"environments": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"resource_type": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceType,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":       schema.StringAttribute{Computed: true},
				"resource_types": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"data_classes": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescDataClasses,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":     schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorContains, Computed: true},
				"data_classes": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"tag_keys": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescTagKeys,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Computed: true},
				"tag_keys": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"tag_key_values": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescTagKeyValues,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Computed: true},
				"tag_key_values": schema.ListNestedAttribute{
					Computed: true,
					Optional: true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"key":   schema.StringAttribute{Computed: true},
							"value": schema.StringAttribute{Optional: true, Computed: true},
						},
					},
				},
			},
		},
		"apps": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescApps,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Computed: true},
				"apps":     schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"cloud_provider": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescCloudProvider,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":        schema.StringAttribute{Computed: true},
				"cloud_providers": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"account_id": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescAccountID,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":    schema.StringAttribute{Computed: true},
				"account_ids": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"source_region": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescSourceRegion,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":       schema.StringAttribute{Computed: true},
				"source_regions": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"vpc": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescVPC,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Computed: true},
				"vpcs":     schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"subnets": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescSubnets,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Computed: true},
				"subnets":  schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"resource_group_name": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceGroupName,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":             schema.StringAttribute{Computed: true},
				"resource_group_names": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"resource_name": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceName,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":       schema.StringAttribute{Computed: true},
				"resource_names": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"resource_id": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceID,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":     schema.StringAttribute{Computed: true},
				"resource_ids": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"group": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescGroup,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: RoleExprDescOperatorLogical,
					Computed:            true,
				},
				"operands": schema.ListNestedAttribute{
					MarkdownDescription: RoleExprDescListOperands,
					Optional:            true,
					Computed:            true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: operandsAttr,
					},
				},
			},
		},
	}
}

func roleAccessConditionOperandsSchemaForDataSource() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"environment": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescEnvironment,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":     schema.StringAttribute{Computed: true},
				"environments": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"resource_type": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceType,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":       schema.StringAttribute{Computed: true},
				"resource_types": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"data_classes": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescDataClasses,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":     schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorContains, Computed: true},
				"data_classes": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"tag_keys": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescTagKeys,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Computed: true},
				"tag_keys": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"tag_key_values": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescTagKeyValues,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Computed: true},
				"tag_key_values": schema.ListNestedAttribute{
					Optional: true,
					Computed: true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"key":   schema.StringAttribute{Computed: true},
							"value": schema.StringAttribute{Optional: true, Computed: true},
						},
					},
				},
			},
		},
		"apps": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescApps,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Computed: true},
				"apps":     schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"cloud_provider": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescCloudProvider,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":        schema.StringAttribute{Computed: true},
				"cloud_providers": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"account_id": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescAccountID,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":    schema.StringAttribute{Computed: true},
				"account_ids": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"source_region": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescSourceRegion,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":       schema.StringAttribute{Computed: true},
				"source_regions": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"vpc": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescVPC,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Computed: true},
				"vpcs":     schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"subnets": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescSubnets,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Computed: true},
				"subnets":  schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"resource_group_name": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceGroupName,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":             schema.StringAttribute{Computed: true},
				"resource_group_names": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"resource_name": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceName,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":       schema.StringAttribute{Computed: true},
				"resource_names": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
		"resource_id": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceID,
			Optional:            true,
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"operator":     schema.StringAttribute{Computed: true},
				"resource_ids": schema.ListAttribute{ElementType: types.StringType, Computed: true},
			},
		},
	}
}
