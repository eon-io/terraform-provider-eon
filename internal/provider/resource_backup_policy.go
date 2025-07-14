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
	ScheduleMode     types.String `tfsdk:"schedule_mode"`
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
	TagValues      types.Object `tfsdk:"tag_values"`
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
	ResourceType types.Object `tfsdk:"resource_type"`
	Environment  types.Object `tfsdk:"environment"`
}

type ResourceTypeConditionModel struct {
	Operator      types.String `tfsdk:"operator"`
	ResourceTypes types.List   `tfsdk:"resource_types"`
}

type EnvironmentConditionModel struct {
	Operator     types.String `tfsdk:"operator"`
	Environments types.List   `tfsdk:"environments"`
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
			"schedule_mode": schema.StringAttribute{
				MarkdownDescription: "Schedule mode: 'STANDARD'",
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
														"interval_hours": schema.Int64Attribute{
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

	default:
		resp.Diagnostics.AddError(
			"Unsupported Backup Policy Type",
			fmt.Sprintf("Backup policy type '%s' is not supported. Only STANDARD and PITR are currently supported.",
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
		"name":          data.Name.ValueString(),
		"enabled":       data.Enabled.ValueBool(),
		"schedule_mode": data.ScheduleMode.ValueString(),
	})

	policy, err := r.client.CreateBackupPolicy(ctx, *createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create backup policy: %s", err))
		return
	}

	data.Id = types.StringValue(policy.Id)
	data.Name = types.StringValue(policy.Name)
	data.Enabled = types.BoolValue(policy.Enabled)
	data.ScheduleMode = types.StringValue("STANDARD")
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

	data.Name = types.StringValue(policy.Name)
	data.Enabled = types.BoolValue(policy.Enabled)
	data.ScheduleMode = types.StringValue("STANDARD")

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
		"name":          plan.Name.ValueString(),
		"enabled":       plan.Enabled.ValueBool(),
		"schedule_mode": plan.ScheduleMode.ValueString(),
	})

	policy, err := r.client.UpdateBackupPolicy(ctx, state.Id.ValueString(), *updateReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating backup policy",
			"Could not update backup policy, unexpected error: "+err.Error(),
		)
		return
	}

	plan.Id = types.StringValue(policy.Id)
	plan.Name = types.StringValue(policy.Name)
	plan.Enabled = types.BoolValue(policy.Enabled)
	plan.ScheduleMode = types.StringValue("STANDARD") // Default assumption
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

	return nil, fmt.Errorf("expression must have at least one condition (environment, resource_type, etc.)")
}
