package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// conditionalExpressionSchema is the shared schema for the conditional
// resource-selector expression used by eon_backup_policy and
// eon_backup_posture_control. Both APIs accept the same condition set;
// keep this block in sync with the ExpressionModel/OperandModel structs
// and the createBackupPolicyExpression builder in resource_backup_policy.go.
func conditionalExpressionSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Conditional expression for CONDITIONAL resource selection mode",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
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
			"group": schema.SingleNestedAttribute{
				MarkdownDescription: "Group condition with logical operator and operands",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"operator": schema.StringAttribute{
						MarkdownDescription: "Logical operator: 'AND' or 'OR'",
						Required:            true,
					},
					"operands": schema.ListNestedAttribute{
						MarkdownDescription: "List of conditions",
						Required:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
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
							},
						},
					},
				},
			},
		},
	}
}
