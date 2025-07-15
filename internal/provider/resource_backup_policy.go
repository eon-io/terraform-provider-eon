package provider

import (
	"context"
	"fmt"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &BackupPolicyResource{}
var _ resource.ResourceWithImportState = &BackupPolicyResource{}

func NewBackupPolicyResource() resource.Resource {
	return &BackupPolicyResource{}
}

type BackupPolicyResource struct {
	client *client.EonClient
}

type BackupPolicyResourceModel struct {
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	ResourceSelector types.Object `tfsdk:"resource_selector"`
	BackupPlan       types.Object `tfsdk:"backup_plan"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

type ResourceSelectorModel struct {
	ResourceSelectionMode     types.String `tfsdk:"resource_selection_mode"`
	ResourceInclusionOverride types.List   `tfsdk:"resource_inclusion_override"`
	ResourceExclusionOverride types.List   `tfsdk:"resource_exclusion_override"`
	Expression                types.Object `tfsdk:"expression"`
}

type StandardPlanModel struct {
	BackupSchedules types.List `tfsdk:"backup_schedules"`
}

type HighFrequencyPlanModel struct {
	ResourceTypes   types.List `tfsdk:"resource_types"`
	BackupSchedules types.List `tfsdk:"backup_schedules"`
}

type BackupScheduleModel struct {
	VaultId        types.String `tfsdk:"vault_id"`
	RetentionDays  types.Int64  `tfsdk:"retention_days"`
	ScheduleConfig types.Object `tfsdk:"schedule_config"`
}

type DailyConfigModel struct {
	TimeOfDayHour      types.Int64 `tfsdk:"time_of_day_hour"`
	TimeOfDayMinutes   types.Int64 `tfsdk:"time_of_day_minutes"`
	StartWindowMinutes types.Int64 `tfsdk:"start_window_minutes"`
}

type ExpressionModel struct {
	// Direct condition types
	Environment    types.Object `tfsdk:"environment"`
	ResourceType   types.Object `tfsdk:"resource_type"`
	DataClasses    types.Object `tfsdk:"data_classes"`
	TagKeyValues   types.Object `tfsdk:"tag_key_values"`
	TagKeys        types.Object `tfsdk:"tag_keys"`
	ResourceNames  types.Object `tfsdk:"resource_names"`
	ResourceIds    types.Object `tfsdk:"resource_ids"`
	ResourceLabels types.Object `tfsdk:"resource_labels"`
	Apps           types.Object `tfsdk:"apps"`
	Regions        types.Object `tfsdk:"regions"`
	Vpc            types.Object `tfsdk:"vpc"`
	Subnets        types.Object `tfsdk:"subnets"`

	Group types.Object `tfsdk:"group"`
}

type ConditionalExpressionModel struct {
	Group types.Object `tfsdk:"group"`
}

type GroupConditionModel struct {
	Operator types.String `tfsdk:"operator"`
	Operands types.List   `tfsdk:"operands"`
}

type OperandModel struct {
	ResourceType      types.Object `tfsdk:"resource_type"`
	Environment       types.Object `tfsdk:"environment"`
	TagKeys           types.Object `tfsdk:"tag_keys"`
	TagKeyValues      types.Object `tfsdk:"tag_key_values"`
	DataClasses       types.Object `tfsdk:"data_classes"`
	Apps              types.Object `tfsdk:"apps"`
	CloudProvider     types.Object `tfsdk:"cloud_provider"`
	AccountId         types.Object `tfsdk:"account_id"`
	SourceRegion      types.Object `tfsdk:"source_region"`
	Vpc               types.Object `tfsdk:"vpc"`
	Subnets           types.Object `tfsdk:"subnets"`
	ResourceGroupName types.Object `tfsdk:"resource_group_name"`
	ResourceName      types.Object `tfsdk:"resource_name"`
	ResourceId        types.Object `tfsdk:"resource_id"`
}

type ResourceTypeConditionModel struct {
	Operator      types.String `tfsdk:"operator"`
	ResourceTypes types.List   `tfsdk:"resource_types"`
}

type EnvironmentConditionModel struct {
	Operator     types.String `tfsdk:"operator"`
	Environments types.List   `tfsdk:"environments"`
}

type TagKeyValuesConditionModel struct {
	Operator     types.String `tfsdk:"operator"`
	TagKeyValues types.List   `tfsdk:"tag_key_values"`
}

type TagKeyValueModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

type DataClassesConditionModel struct {
	Operator    types.String `tfsdk:"operator"`
	DataClasses types.List   `tfsdk:"data_classes"`
}

type AppsConditionModel struct {
	Operator types.String `tfsdk:"operator"`
	Apps     types.List   `tfsdk:"apps"`
}

type CloudProviderConditionModel struct {
	Operator       types.String `tfsdk:"operator"`
	CloudProviders types.List   `tfsdk:"cloud_providers"`
}

type AccountIdConditionModel struct {
	Operator   types.String `tfsdk:"operator"`
	AccountIds types.List   `tfsdk:"account_ids"`
}

type SourceRegionConditionModel struct {
	Operator      types.String `tfsdk:"operator"`
	SourceRegions types.List   `tfsdk:"source_regions"`
}

type VpcConditionModel struct {
	Operator types.String `tfsdk:"operator"`
	Vpcs     types.List   `tfsdk:"vpcs"`
}

type SubnetsConditionModel struct {
	Operator types.String `tfsdk:"operator"`
	Subnets  types.List   `tfsdk:"subnets"`
}

type ResourceGroupNameConditionModel struct {
	Operator           types.String `tfsdk:"operator"`
	ResourceGroupNames types.List   `tfsdk:"resource_group_names"`
}

type ResourceNameConditionModel struct {
	Operator      types.String `tfsdk:"operator"`
	ResourceNames types.List   `tfsdk:"resource_names"`
}

type ResourceIdConditionModel struct {
	Operator    types.String `tfsdk:"operator"`
	ResourceIds types.List   `tfsdk:"resource_ids"`
}

type TagKeysConditionModel struct {
	Operator types.String `tfsdk:"operator"`
	TagKeys  types.List   `tfsdk:"tag_keys"`
}

func (r *BackupPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backup_policy"
}

func (r *BackupPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Eon backup policy resource for managing backup policies",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Backup policy identifier",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name for the backup policy",
				Required:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the backup policy is enabled",
				Required:            true,
			},
			"resource_selector": schema.SingleNestedAttribute{
				MarkdownDescription: "Resource selector configuration",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"resource_selection_mode": schema.StringAttribute{
						MarkdownDescription: "Resource selection mode: 'ALL', 'NONE', or 'CONDITIONAL'",
						Required:            true,
					},
					"resource_inclusion_override": schema.ListAttribute{
						MarkdownDescription: "List of resource IDs to include regardless of selection mode",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"resource_exclusion_override": schema.ListAttribute{
						MarkdownDescription: "List of resource IDs to exclude regardless of selection mode",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"expression": schema.SingleNestedAttribute{
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
															MarkdownDescription: "Operator: 'CONTAINS' or 'NOT_CONTAINS'",
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
					},
				},
			},
			"backup_plan": schema.SingleNestedAttribute{
				MarkdownDescription: "Backup plan configuration",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"backup_policy_type": schema.StringAttribute{
						MarkdownDescription: "Backup policy type: 'STANDARD', 'HIGH_FREQUENCY', or 'PITR'",
						Required:            true,
					},
					"standard_plan": schema.SingleNestedAttribute{
						MarkdownDescription: "Standard backup plan configuration",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"backup_schedules": schema.ListNestedAttribute{
								MarkdownDescription: "List of backup schedules",
								Required:            true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"vault_id": schema.StringAttribute{
											MarkdownDescription: "Vault ID",
											Required:            true,
										},
										"retention_days": schema.Int64Attribute{
											MarkdownDescription: "Retention days",
											Required:            true,
										},
										"schedule_config": schema.SingleNestedAttribute{
											MarkdownDescription: "Schedule configuration",
											Required:            true,
											Attributes: map[string]schema.Attribute{
												"frequency": schema.StringAttribute{
													MarkdownDescription: "Frequency: 'DAILY', 'WEEKLY', 'MONTHLY', 'ANNUALLY', 'INTERVAL'",
													Required:            true,
												},
												"daily_config": schema.SingleNestedAttribute{
													MarkdownDescription: "Daily configuration",
													Optional:            true,
													Attributes: map[string]schema.Attribute{
														"time_of_day_hour": schema.Int64Attribute{
															MarkdownDescription: "Hour of day (0-23)",
															Optional:            true,
														},
														"time_of_day_minutes": schema.Int64Attribute{
															MarkdownDescription: "Minutes of hour (0-59)",
															Optional:            true,
														},
														"start_window_minutes": schema.Int64Attribute{
															MarkdownDescription: "Start window in minutes",
															Optional:            true,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					"high_frequency_plan": schema.SingleNestedAttribute{
						MarkdownDescription: "High frequency backup plan configuration",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"resource_types": schema.ListAttribute{
								MarkdownDescription: "List of resource types for high frequency backups",
								ElementType:         types.StringType,
								Required:            true,
							},
							"backup_schedules": schema.ListNestedAttribute{
								MarkdownDescription: "List of backup schedules",
								Required:            true,
								NestedObject: schema.NestedAttributeObject{
									Attributes: map[string]schema.Attribute{
										"vault_id": schema.StringAttribute{
											MarkdownDescription: "Vault ID",
											Required:            true,
										},
										"retention_days": schema.Int64Attribute{
											MarkdownDescription: "Retention days",
											Required:            true,
										},
										"schedule_config": schema.SingleNestedAttribute{
											MarkdownDescription: "Schedule configuration",
											Required:            true,
											Attributes: map[string]schema.Attribute{
												"frequency": schema.StringAttribute{
													MarkdownDescription: "Frequency: 'INTERVAL'",
													Required:            true,
												},
												"interval_config": schema.SingleNestedAttribute{
													MarkdownDescription: "Interval configuration",
													Required:            true,
													Attributes: map[string]schema.Attribute{
														"interval_minutes": schema.Int64Attribute{
															MarkdownDescription: "Interval in hours",
															Required:            true,
														},
														"start_window_minutes": schema.Int64Attribute{
															MarkdownDescription: "Start window in minutes",
															Optional:            true,
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Creation timestamp",
				Computed:            true,
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "Last update timestamp",
				Computed:            true,
			},
		},
	}
}

func (r *BackupPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.EonClient, got: %T", req.ProviderData))
		return
	}

	r.client = client
}

func (r *BackupPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BackupPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceSelectorAttrs := data.ResourceSelector.Attributes()
	resourceSelectionMode := resourceSelectorAttrs["resource_selection_mode"].(types.String)

	resourceSelector := externalEonSdkAPI.NewBackupPolicyResourceSelector(
		externalEonSdkAPI.ResourceSelectorMode(resourceSelectionMode.ValueString()),
	)

	if expressionObj, exists := resourceSelectorAttrs["expression"]; exists && !expressionObj.IsNull() {
		var resourceSelectorModel ResourceSelectorModel
		diags := data.ResourceSelector.As(ctx, &resourceSelectorModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		expression, err := createBackupPolicyExpression(ctx, &resourceSelectorModel)
		if err != nil {
			resp.Diagnostics.AddError("Invalid Conditional Expression", fmt.Sprintf("Failed to create conditional expression: %s", err))
			return
		}
		resourceSelector.SetExpression(*expression)
	}

	if inclusionOverrideObj, exists := resourceSelectorAttrs["resource_inclusion_override"]; exists && !inclusionOverrideObj.IsNull() {
		var inclusionOverride []string
		diags := inclusionOverrideObj.(types.List).ElementsAs(ctx, &inclusionOverride, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		resourceSelector.SetResourceInclusionOverride(inclusionOverride)
	}

	if exclusionOverrideObj, exists := resourceSelectorAttrs["resource_exclusion_override"]; exists && !exclusionOverrideObj.IsNull() {
		var exclusionOverride []string
		diags := exclusionOverrideObj.(types.List).ElementsAs(ctx, &exclusionOverride, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		resourceSelector.SetResourceExclusionOverride(exclusionOverride)
	}

	backupPlanAttrs := data.BackupPlan.Attributes()
	backupPolicyType := backupPlanAttrs["backup_policy_type"].(types.String)

	backupPlan := externalEonSdkAPI.NewBackupPolicyPlan(
		externalEonSdkAPI.BackupPolicyType(backupPolicyType.ValueString()),
	)

	var diags diag.Diagnostics
	switch backupPolicyType.ValueString() {
	case "STANDARD", "PITR":
		standardPlanObj := backupPlanAttrs["standard_plan"].(types.Object)
		var standardPlanModel StandardPlanModel
		diags = standardPlanObj.As(ctx, &standardPlanModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		var backupSchedules []externalEonSdkAPI.StandardBackupSchedules
		var schedules []BackupScheduleModel
		diags = standardPlanModel.BackupSchedules.ElementsAs(ctx, &schedules, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		for _, schedule := range schedules {
			scheduleConfig, err := createStandardScheduleConfig(&schedule)
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Schedule Configuration",
					fmt.Sprintf("Failed to create schedule configuration for %s policy: %s", backupPolicyType.ValueString(), err),
				)
				return
			}

			retentionDays, err := SafeInt32Conversion(schedule.RetentionDays.ValueInt64())
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Retention Days",
					fmt.Sprintf("Failed to validate retention days: %s", err),
				)
				return
			}

			backupSchedule := externalEonSdkAPI.NewStandardBackupSchedules(
				schedule.VaultId.ValueString(),
				*scheduleConfig,
				retentionDays,
			)
			backupSchedules = append(backupSchedules, *backupSchedule)
		}

		standardPlan := externalEonSdkAPI.NewStandardBackupPolicyPlan(backupSchedules)
		backupPlan.SetStandardPlan(*standardPlan)

	case "HIGH_FREQUENCY":
		highFrequencyPlanObj := backupPlanAttrs["high_frequency_plan"].(types.Object)
		var highFrequencyPlanModel HighFrequencyPlanModel
		diags = highFrequencyPlanObj.As(ctx, &highFrequencyPlanModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		var resourceTypeStrings []string
		diags = highFrequencyPlanModel.ResourceTypes.ElementsAs(ctx, &resourceTypeStrings, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		var resourceTypes []externalEonSdkAPI.HighFrequencyBackupResourceType
		for _, resourceTypeStr := range resourceTypeStrings {
			resourceType := externalEonSdkAPI.NewHighFrequencyBackupResourceType()
			sdkResourceType := externalEonSdkAPI.ResourceType(resourceTypeStr)
			resourceType.SetResourceType(sdkResourceType)
			resourceTypes = append(resourceTypes, *resourceType)
		}

		var backupSchedules []externalEonSdkAPI.HighFrequencyBackupSchedules
		var schedules []BackupScheduleModel
		diags = highFrequencyPlanModel.BackupSchedules.ElementsAs(ctx, &schedules, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		for _, schedule := range schedules {
			scheduleConfig, err := createHighFrequencyScheduleConfig(&schedule)
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Schedule Configuration",
					fmt.Sprintf("Failed to create high frequency schedule configuration: %s", err),
				)
				return
			}

			retentionDays, err := SafeInt32Conversion(schedule.RetentionDays.ValueInt64())
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Retention Days",
					fmt.Sprintf("Failed to validate retention days: %s", err),
				)
				return
			}

			backupSchedule := externalEonSdkAPI.NewHighFrequencyBackupSchedules(
				schedule.VaultId.ValueString(),
				*scheduleConfig,
				retentionDays,
			)
			backupSchedules = append(backupSchedules, *backupSchedule)
		}

		highFrequencyPlan := externalEonSdkAPI.NewHighFrequencyBackupPolicyPlan(
			resourceTypes,
			backupSchedules,
		)
		backupPlan.SetHighFrequencyPlan(*highFrequencyPlan)

	default:
		resp.Diagnostics.AddError(
			"Unsupported Backup Policy Type",
			fmt.Sprintf("Backup policy type '%s' is not supported. Only STANDARD, HIGH_FREQUENCY, and PITR are currently supported.",
				backupPolicyType.ValueString()),
		)
		return
	}

	createReq := externalEonSdkAPI.NewCreateBackupPolicyRequest(
		data.Name.ValueString(),
		*resourceSelector,
		*backupPlan,
	)

	if !data.Enabled.IsNull() {
		enabled := data.Enabled.ValueBool()
		createReq.SetEnabled(enabled)
	}

	tflog.Debug(ctx, "Creating backup policy", map[string]interface{}{
		"name":    data.Name.ValueString(),
		"enabled": data.Enabled.ValueBool(),
	})

	policy, err := r.client.CreateBackupPolicy(ctx, *createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create backup policy: %s", err))
		return
	}

	data.Id = types.StringValue(policy.Id)
	data.Name = types.StringValue(policy.Name)
	data.Enabled = types.BoolValue(policy.Enabled)
	data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

	tflog.Debug(ctx, "Backup policy created", map[string]interface{}{
		"id":   data.Id.ValueString(),
		"name": data.Name.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BackupPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BackupPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.GetBackupPolicy(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read backup policy: %s", err))
		return
	}

	data.Id = types.StringValue(policy.Id)
	data.Name = types.StringValue(policy.Name)
	data.Enabled = types.BoolValue(policy.Enabled)
	data.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
	data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BackupPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BackupPolicyResourceModel
	var state BackupPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceSelectorAttrs := plan.ResourceSelector.Attributes()
	resourceSelectionMode := resourceSelectorAttrs["resource_selection_mode"].(types.String)

	resourceSelector := externalEonSdkAPI.NewBackupPolicyResourceSelector(
		externalEonSdkAPI.ResourceSelectorMode(resourceSelectionMode.ValueString()),
	)

	if expressionObj, exists := resourceSelectorAttrs["expression"]; exists && !expressionObj.IsNull() {
		var resourceSelectorModel ResourceSelectorModel
		diags := plan.ResourceSelector.As(ctx, &resourceSelectorModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		expression, err := createBackupPolicyExpression(ctx, &resourceSelectorModel)
		if err != nil {
			resp.Diagnostics.AddError("Invalid Conditional Expression", fmt.Sprintf("Failed to create conditional expression: %s", err))
			return
		}
		resourceSelector.SetExpression(*expression)
	}

	if inclusionOverrideObj, exists := resourceSelectorAttrs["resource_inclusion_override"]; exists && !inclusionOverrideObj.IsNull() {
		var inclusionOverride []string
		diags := inclusionOverrideObj.(types.List).ElementsAs(ctx, &inclusionOverride, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		resourceSelector.SetResourceInclusionOverride(inclusionOverride)
	}

	if exclusionOverrideObj, exists := resourceSelectorAttrs["resource_exclusion_override"]; exists && !exclusionOverrideObj.IsNull() {
		var exclusionOverride []string
		diags := exclusionOverrideObj.(types.List).ElementsAs(ctx, &exclusionOverride, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		resourceSelector.SetResourceExclusionOverride(exclusionOverride)
	}

	backupPlanAttrs := plan.BackupPlan.Attributes()
	backupPolicyType := backupPlanAttrs["backup_policy_type"].(types.String)

	backupPlan := externalEonSdkAPI.NewBackupPolicyPlan(
		externalEonSdkAPI.BackupPolicyType(backupPolicyType.ValueString()),
	)

	switch backupPolicyType.ValueString() {
	case "STANDARD", "PITR":
		standardPlanObj := backupPlanAttrs["standard_plan"].(types.Object)
		var standardPlanModel StandardPlanModel
		diags := standardPlanObj.As(ctx, &standardPlanModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		var backupSchedules []externalEonSdkAPI.StandardBackupSchedules
		var schedules []BackupScheduleModel
		diags = standardPlanModel.BackupSchedules.ElementsAs(ctx, &schedules, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		for _, schedule := range schedules {
			scheduleConfig, err := createStandardScheduleConfig(&schedule)
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Schedule Configuration",
					fmt.Sprintf("Failed to create schedule configuration: %s", err),
				)
				return
			}

			retentionDays, err := SafeInt32Conversion(schedule.RetentionDays.ValueInt64())
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Retention Days",
					fmt.Sprintf("Failed to validate retention days: %s", err),
				)
				return
			}

			backupSchedule := externalEonSdkAPI.NewStandardBackupSchedules(
				schedule.VaultId.ValueString(),
				*scheduleConfig,
				retentionDays,
			)
			backupSchedules = append(backupSchedules, *backupSchedule)
		}

		standardPlan := externalEonSdkAPI.NewStandardBackupPolicyPlan(backupSchedules)
		backupPlan.SetStandardPlan(*standardPlan)

	case "HIGH_FREQUENCY":
		highFrequencyPlanObj := backupPlanAttrs["high_frequency_plan"].(types.Object)
		var highFrequencyPlanModel HighFrequencyPlanModel
		diags := highFrequencyPlanObj.As(ctx, &highFrequencyPlanModel, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		var resourceTypeStrings []string
		diags = highFrequencyPlanModel.ResourceTypes.ElementsAs(ctx, &resourceTypeStrings, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		var resourceTypes []externalEonSdkAPI.HighFrequencyBackupResourceType
		for _, resourceTypeStr := range resourceTypeStrings {
			resourceType := externalEonSdkAPI.NewHighFrequencyBackupResourceType()
			sdkResourceType := externalEonSdkAPI.ResourceType(resourceTypeStr)
			resourceType.SetResourceType(sdkResourceType)
			resourceTypes = append(resourceTypes, *resourceType)
		}

		var backupSchedules []externalEonSdkAPI.HighFrequencyBackupSchedules
		var schedules []BackupScheduleModel
		diags = highFrequencyPlanModel.BackupSchedules.ElementsAs(ctx, &schedules, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		for _, schedule := range schedules {
			scheduleConfig, err := createHighFrequencyScheduleConfig(&schedule)
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Schedule Configuration",
					fmt.Sprintf("Failed to create high frequency schedule configuration: %s", err),
				)
				return
			}

			retentionDays, err := SafeInt32Conversion(schedule.RetentionDays.ValueInt64())
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Retention Days",
					fmt.Sprintf("Failed to validate retention days: %s", err),
				)
				return
			}

			backupSchedule := externalEonSdkAPI.NewHighFrequencyBackupSchedules(
				schedule.VaultId.ValueString(),
				*scheduleConfig,
				retentionDays,
			)
			backupSchedules = append(backupSchedules, *backupSchedule)
		}

		highFrequencyPlan := externalEonSdkAPI.NewHighFrequencyBackupPolicyPlan(
			resourceTypes,
			backupSchedules,
		)
		backupPlan.SetHighFrequencyPlan(*highFrequencyPlan)

	default:
		resp.Diagnostics.AddError(
			"Unsupported Backup Policy Type",
			fmt.Sprintf("Backup policy type '%s' is not supported. Only STANDARD and PITR are currently supported.",
				backupPolicyType.ValueString()),
		)
		return
	}

	updateReq := externalEonSdkAPI.NewUpdateBackupPolicyRequest(
		plan.Name.ValueString(),
		*resourceSelector,
		*backupPlan,
	)

	if !plan.Enabled.IsNull() {
		enabled := plan.Enabled.ValueBool()
		updateReq.SetEnabled(enabled)
	}

	tflog.Debug(ctx, "Updating backup policy", map[string]interface{}{
		"name":    plan.Name.ValueString(),
		"enabled": plan.Enabled.ValueBool(),
	})

	updatedPolicy, err := r.client.UpdateBackupPolicy(ctx, state.Id.ValueString(), *updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating backup policy",
			"Could not update backup policy, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(updatedPolicy.Id)
	plan.Name = types.StringValue(updatedPolicy.Name)
	plan.Enabled = types.BoolValue(updatedPolicy.Enabled)
	plan.CreatedAt = types.StringValue(time.Now().Format(time.RFC3339))
	plan.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BackupPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BackupPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteBackupPolicy(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete backup policy: %s", err))
		return
	}
}

func (r *BackupPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func createDailyConfigFromModel(data *DailyConfigModel) (*externalEonSdkAPI.DailyConfig, error) {
	dailyConfig := externalEonSdkAPI.NewDailyConfigWithDefaults()

	if !data.TimeOfDayHour.IsNull() && !data.TimeOfDayMinutes.IsNull() {
		hour, err := SafeInt32Conversion(data.TimeOfDayHour.ValueInt64())
		if err != nil {
			return nil, err
		}

		minutes, err := SafeInt32Conversion(data.TimeOfDayMinutes.ValueInt64())
		if err != nil {
			return nil, err
		}

		timeOfDay := externalEonSdkAPI.NewTimeOfDay(hour, minutes)
		dailyConfig.SetTimeOfDay(*timeOfDay)
	}

	if !data.StartWindowMinutes.IsNull() {
		value, err := SafeInt32Conversion(data.StartWindowMinutes.ValueInt64())
		if err != nil {
			return nil, err
		}
		dailyConfig.SetStartWindowMinutes(value)
	}

	return dailyConfig, nil
}

// createStandardScheduleConfig creates a StandardBackupScheduleConfig based on the policy type and frequency
func createStandardScheduleConfig(schedule *BackupScheduleModel) (*externalEonSdkAPI.StandardBackupScheduleConfig, error) {
	scheduleConfigAttrs := schedule.ScheduleConfig.Attributes()
	frequencyObj := scheduleConfigAttrs["frequency"]
	if frequencyObj == nil {
		return nil, fmt.Errorf("frequency field is required in schedule config")
	}

	frequency := frequencyObj.(types.String).ValueString()

	switch frequency {
	case "DAILY":
		scheduleConfig := externalEonSdkAPI.NewStandardBackupScheduleConfig(externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_DAILY)

		if dailyConfigObj, exists := scheduleConfigAttrs["daily_config"]; exists && !dailyConfigObj.IsNull() {
			dailyConfigAttrs := dailyConfigObj.(types.Object).Attributes()

			timeOfDayHour, err := SafeInt32Conversion(dailyConfigAttrs["time_of_day_hour"].(types.Int64).ValueInt64())
			if err != nil {
				return nil, fmt.Errorf("invalid time of day hour: %s", err)
			}
			timeOfDayMinutes, err := SafeInt32Conversion(dailyConfigAttrs["time_of_day_minutes"].(types.Int64).ValueInt64())
			if err != nil {
				return nil, fmt.Errorf("invalid time of day minutes: %s", err)
			}

			timeOfDay := externalEonSdkAPI.NewTimeOfDay(
				timeOfDayHour,
				timeOfDayMinutes,
			)

			dailyConfig := externalEonSdkAPI.NewDailyConfig()
			dailyConfig.SetTimeOfDay(*timeOfDay)

			if startWindowObj, exists := dailyConfigAttrs["start_window_minutes"]; exists && !startWindowObj.IsNull() {
				startWindow, err := SafeInt32Conversion(startWindowObj.(types.Int64).ValueInt64())
				if err != nil {
					return nil, fmt.Errorf("invalid start window minutes: %s", err)
				}
				dailyConfig.SetStartWindowMinutes(startWindow)
			}

			scheduleConfig.SetDailyConfig(*dailyConfig)
		}

		return scheduleConfig, nil

	default:
		return nil, fmt.Errorf("unsupported schedule frequency: %s", frequency)
	}
}

func createHighFrequencyScheduleConfig(schedule *BackupScheduleModel) (*externalEonSdkAPI.HighFrequencyBackupScheduleConfig, error) {
	scheduleConfigAttrs := schedule.ScheduleConfig.Attributes()
	frequencyObj := scheduleConfigAttrs["frequency"]
	if frequencyObj == nil {
		return nil, fmt.Errorf("frequency field is required in schedule config")
	}

	frequency := frequencyObj.(types.String).ValueString()

	highFreqScheduleConfig := externalEonSdkAPI.NewHighFrequencyBackupScheduleConfig()

	switch frequency {
	case "INTERVAL":
		highFreqScheduleConfig.SetFrequency(externalEonSdkAPI.HIGH_FREQUENCY_BACKUP_SCHEDULE_INTERVAL)

		intervalConfigObj := scheduleConfigAttrs["interval_config"]
		if intervalConfigObj == nil {
			return nil, fmt.Errorf("interval_config field is required for INTERVAL frequency")
		}

		intervalConfigAttrs := intervalConfigObj.(types.Object).Attributes()

		intervalHours, err := SafeInt32Conversion(intervalConfigAttrs["interval_minutes"].(types.Int64).ValueInt64())
		if err != nil {
			return nil, fmt.Errorf("invalid interval hours: %s", err)
		}

		intervalConfig := externalEonSdkAPI.NewHighFrequencyIntervalConfig(intervalHours)
		highFreqScheduleConfig.SetIntervalConfig(*intervalConfig)

		return highFreqScheduleConfig, nil

	default:
		return nil, fmt.Errorf("unsupported high frequency schedule frequency: %s", frequency)
	}
}

func createBackupPolicyExpression(ctx context.Context, data *ResourceSelectorModel) (*externalEonSdkAPI.BackupPolicyExpression, error) {
	if data.Expression.IsNull() {
		return nil, fmt.Errorf("expression is required for CONDITIONAL resource selection mode")
	}

	expressionAttrs := data.Expression.Attributes()
	expr := externalEonSdkAPI.NewBackupPolicyExpression()

	if environmentObj, exists := expressionAttrs["environment"]; exists && !environmentObj.IsNull() {
		var envCondition EnvironmentConditionModel
		diags := environmentObj.(types.Object).As(ctx, &envCondition, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			tflog.Error(ctx, "Failed to parse environment condition", map[string]interface{}{
				"error": diags.Errors(),
			})
			return nil, fmt.Errorf("failed to parse environment condition")
		}

		var environments []string
		diags = envCondition.Environments.ElementsAs(ctx, &environments, false)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to parse environments list")
		}

		var environmentEnums []externalEonSdkAPI.Environment
		for _, env := range environments {
			environmentEnums = append(environmentEnums, externalEonSdkAPI.Environment(env))
		}

		operator := externalEonSdkAPI.ScalarOperators(envCondition.Operator.ValueString())
		envConditionApi := externalEonSdkAPI.NewEnvironmentCondition(operator, environmentEnums)
		expr.SetEnvironment(*envConditionApi)

		tflog.Debug(ctx, "Successfully created environment condition", map[string]interface{}{
			"operator":     envCondition.Operator.ValueString(),
			"environments": environments,
		})

		return expr, nil
	}

	if resourceTypeObj, exists := expressionAttrs["resource_type"]; exists && !resourceTypeObj.IsNull() {
		var resourceTypeCondition ResourceTypeConditionModel
		diags := resourceTypeObj.(types.Object).As(ctx, &resourceTypeCondition, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			return nil, fmt.Errorf("failed to parse resource type condition")
		}

		var resourceTypes []string
		diags = resourceTypeCondition.ResourceTypes.ElementsAs(ctx, &resourceTypes, false)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to parse resource types list")
		}

		var resourceTypeEnums []externalEonSdkAPI.ResourceType
		for _, rt := range resourceTypes {
			resourceTypeEnums = append(resourceTypeEnums, externalEonSdkAPI.ResourceType(rt))
		}

		operator := externalEonSdkAPI.ScalarOperators(resourceTypeCondition.Operator.ValueString())
		resourceTypeConditionApi := externalEonSdkAPI.NewResourceTypeCondition(operator, resourceTypeEnums)
		expr.SetResourceType(*resourceTypeConditionApi)

		return expr, nil
	}

	if tagKeyValuesObj, exists := expressionAttrs["tag_key_values"]; exists && !tagKeyValuesObj.IsNull() {
		var tagKeyValuesCondition TagKeyValuesConditionModel
		diags := tagKeyValuesObj.(types.Object).As(ctx, &tagKeyValuesCondition, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			tflog.Error(ctx, "Failed to parse tag key-value condition", map[string]interface{}{
				"error": diags.Errors(),
			})
			return nil, fmt.Errorf("failed to parse tag key-value condition")
		}

		var tagKeyValues []TagKeyValueModel
		diags = tagKeyValuesCondition.TagKeyValues.ElementsAs(ctx, &tagKeyValues, false)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to parse tag key-value list")
		}

		var tagKeyValueEnums []externalEonSdkAPI.TagKeyValue
		for _, kv := range tagKeyValues {
			tagKeyValue := externalEonSdkAPI.NewTagKeyValue(kv.Key.ValueString())
			tagKeyValue.SetValue(kv.Value.ValueString())
			tagKeyValueEnums = append(tagKeyValueEnums, *tagKeyValue)
		}

		operator := externalEonSdkAPI.ListOperators(tagKeyValuesCondition.Operator.ValueString())
		tagKeyValuesConditionApi := externalEonSdkAPI.NewTagKeyValuesCondition(operator, tagKeyValueEnums)
		expr.SetTagKeyValues(*tagKeyValuesConditionApi)

		tflog.Debug(ctx, "Successfully created tag key-value condition", map[string]interface{}{
			"operator":       tagKeyValuesCondition.Operator.ValueString(),
			"tag_key_values": tagKeyValues,
		})

		return expr, nil
	}

	if tagKeysObj, exists := expressionAttrs["tag_keys"]; exists && !tagKeysObj.IsNull() {
		var tagKeysCondition TagKeysConditionModel
		diags := tagKeysObj.(types.Object).As(ctx, &tagKeysCondition, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			tflog.Error(ctx, "Failed to parse tag keys condition", map[string]interface{}{
				"error": diags.Errors(),
			})
			return nil, fmt.Errorf("failed to parse tag keys condition")
		}

		var tagKeys []string
		diags = tagKeysCondition.TagKeys.ElementsAs(ctx, &tagKeys, false)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to parse tag keys list")
		}

		operator := externalEonSdkAPI.ListOperators(tagKeysCondition.Operator.ValueString())
		tagKeysConditionApi := externalEonSdkAPI.NewTagKeysCondition(operator, tagKeys)
		expr.SetTagKeys(*tagKeysConditionApi)

		tflog.Debug(ctx, "Successfully created tag keys condition", map[string]interface{}{
			"operator": tagKeysCondition.Operator.ValueString(),
			"tag_keys": tagKeys,
		})

		return expr, nil
	}

	if groupObj, exists := expressionAttrs["group"]; exists && !groupObj.IsNull() {
		var groupCondition GroupConditionModel
		diags := groupObj.(types.Object).As(ctx, &groupCondition, basetypes.ObjectAsOptions{})
		if diags.HasError() {
			tflog.Error(ctx, "Failed to parse group condition", map[string]interface{}{
				"error": diags.Errors(),
			})
			return nil, fmt.Errorf("failed to parse group condition")
		}

		var operands []OperandModel
		diags = groupCondition.Operands.ElementsAs(ctx, &operands, false)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to parse operands")
		}

		var expressions []externalEonSdkAPI.BackupPolicyExpression
		for _, operand := range operands {
			operandExpr := externalEonSdkAPI.NewBackupPolicyExpression()

			if !operand.ResourceType.IsNull() {
				var resourceTypeCondition ResourceTypeConditionModel
				diags := operand.ResourceType.As(ctx, &resourceTypeCondition, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse resource type condition in operand")
				}

				var resourceTypes []string
				diags = resourceTypeCondition.ResourceTypes.ElementsAs(ctx, &resourceTypes, false)
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse resource types in operand")
				}

				var resourceTypeEnums []externalEonSdkAPI.ResourceType
				for _, rt := range resourceTypes {
					resourceTypeEnums = append(resourceTypeEnums, externalEonSdkAPI.ResourceType(rt))
				}

				operator := externalEonSdkAPI.ScalarOperators(resourceTypeCondition.Operator.ValueString())
				resourceTypeConditionApi := externalEonSdkAPI.NewResourceTypeCondition(operator, resourceTypeEnums)
				operandExpr.SetResourceType(*resourceTypeConditionApi)
			}

			if !operand.Environment.IsNull() {
				var envCondition EnvironmentConditionModel
				diags := operand.Environment.As(ctx, &envCondition, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse environment condition in operand")
				}

				var environments []string
				diags = envCondition.Environments.ElementsAs(ctx, &environments, false)
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse environments in operand")
				}

				var environmentEnums []externalEonSdkAPI.Environment
				for _, env := range environments {
					environmentEnums = append(environmentEnums, externalEonSdkAPI.Environment(env))
				}

				operator := externalEonSdkAPI.ScalarOperators(envCondition.Operator.ValueString())
				envConditionApi := externalEonSdkAPI.NewEnvironmentCondition(operator, environmentEnums)
				operandExpr.SetEnvironment(*envConditionApi)
			}

			if !operand.TagKeys.IsNull() {
				var tagKeysCondition TagKeysConditionModel
				diags := operand.TagKeys.As(ctx, &tagKeysCondition, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse tag keys condition in operand")
				}

				var tagKeys []string
				diags = tagKeysCondition.TagKeys.ElementsAs(ctx, &tagKeys, false)
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse tag keys in operand")
				}

				operator := externalEonSdkAPI.ListOperators(tagKeysCondition.Operator.ValueString())
				tagKeysConditionApi := externalEonSdkAPI.NewTagKeysCondition(operator, tagKeys)
				operandExpr.SetTagKeys(*tagKeysConditionApi)
			}

			if !operand.TagKeyValues.IsNull() {
				var tagKeyValuesCondition TagKeyValuesConditionModel
				diags := operand.TagKeyValues.As(ctx, &tagKeyValuesCondition, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse tag key-values condition in operand")
				}

				var tagKeyValues []TagKeyValueModel
				diags = tagKeyValuesCondition.TagKeyValues.ElementsAs(ctx, &tagKeyValues, false)
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse tag key-values in operand")
				}

				var tagKeyValueEnums []externalEonSdkAPI.TagKeyValue
				for _, kv := range tagKeyValues {
					tagKeyValue := externalEonSdkAPI.NewTagKeyValue(kv.Key.ValueString())
					tagKeyValue.SetValue(kv.Value.ValueString())
					tagKeyValueEnums = append(tagKeyValueEnums, *tagKeyValue)
				}

				operator := externalEonSdkAPI.ListOperators(tagKeyValuesCondition.Operator.ValueString())
				tagKeyValuesConditionApi := externalEonSdkAPI.NewTagKeyValuesCondition(operator, tagKeyValueEnums)
				operandExpr.SetTagKeyValues(*tagKeyValuesConditionApi)
			}

			if !operand.DataClasses.IsNull() {
				var dataClassesCondition DataClassesConditionModel
				diags := operand.DataClasses.As(ctx, &dataClassesCondition, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse data_classes condition in operand")
				}

				var dataClasses []string
				diags = dataClassesCondition.DataClasses.ElementsAs(ctx, &dataClasses, false)
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse data_classes list in operand")
				}

				var dataClassEnums []externalEonSdkAPI.DataClass
				for _, dc := range dataClasses {
					dataClassEnums = append(dataClassEnums, externalEonSdkAPI.DataClass(dc))
				}

				operator := externalEonSdkAPI.ListOperators(dataClassesCondition.Operator.ValueString())
				dataClassesConditionApi := externalEonSdkAPI.NewDataClassesCondition(operator, dataClassEnums)
				operandExpr.SetDataClasses(*dataClassesConditionApi)
			}

			if !operand.Apps.IsNull() {
				var appsCondition AppsConditionModel
				diags := operand.Apps.As(ctx, &appsCondition, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse apps condition in operand")
				}

				var apps []string
				diags = appsCondition.Apps.ElementsAs(ctx, &apps, false)
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse apps list in operand")
				}

				operator := externalEonSdkAPI.ListOperators(appsCondition.Operator.ValueString())
				appsConditionApi := externalEonSdkAPI.NewAppsCondition(operator, apps)
				operandExpr.SetApps(*appsConditionApi)
			}

			if !operand.CloudProvider.IsNull() {
				var cloudProviderCondition CloudProviderConditionModel
				diags := operand.CloudProvider.As(ctx, &cloudProviderCondition, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse cloud_provider condition in operand")
				}

				var cloudProviders []string
				diags = cloudProviderCondition.CloudProviders.ElementsAs(ctx, &cloudProviders, false)
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse cloud_providers list in operand")
				}

				var providerEnums []externalEonSdkAPI.Provider
				for _, cp := range cloudProviders {
					providerEnums = append(providerEnums, externalEonSdkAPI.Provider(cp))
				}

				operator := externalEonSdkAPI.ScalarOperators(cloudProviderCondition.Operator.ValueString())
				cloudProviderConditionApi := externalEonSdkAPI.NewCloudProviderCondition(operator, providerEnums)
				operandExpr.SetCloudProvider(*cloudProviderConditionApi)
			}

			if !operand.AccountId.IsNull() {
				var accountIdCondition AccountIdConditionModel
				diags := operand.AccountId.As(ctx, &accountIdCondition, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse account_id condition in operand")
				}

				var accountIds []string
				diags = accountIdCondition.AccountIds.ElementsAs(ctx, &accountIds, false)
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse account_ids list in operand")
				}

				operator := externalEonSdkAPI.ScalarOperators(accountIdCondition.Operator.ValueString())
				accountIdConditionApi := externalEonSdkAPI.NewAccountIdCondition(operator, accountIds)
				operandExpr.SetAccountId(*accountIdConditionApi)
			}

			if !operand.SourceRegion.IsNull() {
				var sourceRegionCondition SourceRegionConditionModel
				diags := operand.SourceRegion.As(ctx, &sourceRegionCondition, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse source_region condition in operand")
				}

				var sourceRegions []string
				diags = sourceRegionCondition.SourceRegions.ElementsAs(ctx, &sourceRegions, false)
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse source_regions list in operand")
				}

				operator := externalEonSdkAPI.ScalarOperators(sourceRegionCondition.Operator.ValueString())
				sourceRegionConditionApi := externalEonSdkAPI.NewRegionCondition(operator, sourceRegions)
				operandExpr.SetSourceRegion(*sourceRegionConditionApi)
			}

			if !operand.Vpc.IsNull() {
				var vpcCondition VpcConditionModel
				diags := operand.Vpc.As(ctx, &vpcCondition, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse vpc condition in operand")
				}

				var vpcs []string
				diags = vpcCondition.Vpcs.ElementsAs(ctx, &vpcs, false)
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse vpcs list in operand")
				}

				operator := externalEonSdkAPI.ScalarOperators(vpcCondition.Operator.ValueString())
				vpcConditionApi := externalEonSdkAPI.NewVpcCondition(operator, vpcs)
				operandExpr.SetVpc(*vpcConditionApi)
			}

			if !operand.Subnets.IsNull() {
				var subnetsCondition SubnetsConditionModel
				diags := operand.Subnets.As(ctx, &subnetsCondition, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse subnets condition in operand")
				}

				var subnets []string
				diags = subnetsCondition.Subnets.ElementsAs(ctx, &subnets, false)
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse subnets list in operand")
				}

				operator := externalEonSdkAPI.ListOperators(subnetsCondition.Operator.ValueString())
				subnetsConditionApi := externalEonSdkAPI.NewSubnetsCondition(operator, subnets)
				operandExpr.SetSubnets(*subnetsConditionApi)
			}

			if !operand.ResourceGroupName.IsNull() {
				var resourceGroupNameCondition ResourceGroupNameConditionModel
				diags := operand.ResourceGroupName.As(ctx, &resourceGroupNameCondition, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse resource_group_name condition in operand")
				}

				var resourceGroupNames []string
				diags = resourceGroupNameCondition.ResourceGroupNames.ElementsAs(ctx, &resourceGroupNames, false)
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse resource_group_names list in operand")
				}

				operator := externalEonSdkAPI.ScalarOperators(resourceGroupNameCondition.Operator.ValueString())
				resourceGroupNameConditionApi := externalEonSdkAPI.NewResourceGroupNameCondition(operator, resourceGroupNames)
				operandExpr.SetResourceGroupName(*resourceGroupNameConditionApi)
			}

			if !operand.ResourceName.IsNull() {
				var resourceNameCondition ResourceNameConditionModel
				diags := operand.ResourceName.As(ctx, &resourceNameCondition, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse resource_name condition in operand")
				}

				var resourceNames []string
				diags = resourceNameCondition.ResourceNames.ElementsAs(ctx, &resourceNames, false)
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse resource_names list in operand")
				}

				operator := externalEonSdkAPI.ScalarOperators(resourceNameCondition.Operator.ValueString())
				resourceNameConditionApi := externalEonSdkAPI.NewResourceNameCondition(operator, resourceNames)
				operandExpr.SetResourceName(*resourceNameConditionApi)
			}

			if !operand.ResourceId.IsNull() {
				var resourceIdCondition ResourceIdConditionModel
				diags := operand.ResourceId.As(ctx, &resourceIdCondition, basetypes.ObjectAsOptions{})
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse resource_id condition in operand")
				}

				var resourceIds []string
				diags = resourceIdCondition.ResourceIds.ElementsAs(ctx, &resourceIds, false)
				if diags.HasError() {
					return nil, fmt.Errorf("failed to parse resource_ids list in operand")
				}

				operator := externalEonSdkAPI.ScalarOperators(resourceIdCondition.Operator.ValueString())
				resourceIdConditionApi := externalEonSdkAPI.NewResourceIdCondition(operator, resourceIds)
				operandExpr.SetResourceId(*resourceIdConditionApi)
			}

			expressions = append(expressions, *operandExpr)
		}

		logicalOperator := externalEonSdkAPI.LogicalOperator(groupCondition.Operator.ValueString())
		groupConditionApi := externalEonSdkAPI.NewBackupPolicyGroupCondition(logicalOperator, expressions)
		expr.SetGroup(*groupConditionApi)

		tflog.Debug(ctx, "Successfully created group condition", map[string]interface{}{
			"operator":       groupCondition.Operator.ValueString(),
			"operands_count": len(operands),
		})

		return expr, nil
	}

	return nil, fmt.Errorf("expression must have at least one condition (environment, resource_type, tag_key_values, tag_keys, group, etc.)")
}
