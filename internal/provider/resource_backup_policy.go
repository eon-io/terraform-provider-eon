package provider

import (
	"context"
	"fmt"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	Id                         types.String `tfsdk:"id"`
	Name                       types.String `tfsdk:"name"`
	Enabled                    types.Bool   `tfsdk:"enabled"`
	ResourceSelectionMode      types.String `tfsdk:"resource_selection_mode"`
	ResourceInclusionOverride  types.List   `tfsdk:"resource_inclusion_override"`
	ResourceExclusionOverride  types.List   `tfsdk:"resource_exclusion_override"`
	ConditionalExpression      types.Object `tfsdk:"conditional_expression"`
	BackupPolicyType           types.String `tfsdk:"backup_policy_type"`
	BackupSchedules            types.List   `tfsdk:"backup_schedules"`
	ScheduleFrequency          types.String `tfsdk:"schedule_frequency"`
	TimeOfDayHour              types.Int64  `tfsdk:"time_of_day_hour"`
	TimeOfDayMinutes           types.Int64  `tfsdk:"time_of_day_minutes"`
	IntervalMinutes            types.Int64  `tfsdk:"interval_minutes"`
	StartWindowMinutes         types.Int64  `tfsdk:"start_window_minutes"`
	HighFrequencyResourceTypes types.List   `tfsdk:"high_frequency_resource_types"`
	CreatedAt                  types.String `tfsdk:"created_at"`
	UpdatedAt                  types.String `tfsdk:"updated_at"`
}

type BackupScheduleModel struct {
	VaultId       types.String `tfsdk:"vault_id"`
	RetentionDays types.Int64  `tfsdk:"retention_days"`
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
			"conditional_expression": schema.SingleNestedAttribute{
				MarkdownDescription: "Conditional expression for CONDITIONAL resource selection mode",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"group": schema.SingleNestedAttribute{
						MarkdownDescription: "Group condition with logical operator and operands",
						Required:            true,
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
													MarkdownDescription: "List of resource types (e.g., 'AWS_EC2', 'AWS_S3')",
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
													MarkdownDescription: "List of environments (e.g., 'PROD', 'DEV', 'STAGING')",
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
			"backup_policy_type": schema.StringAttribute{
				MarkdownDescription: "Backup policy type: 'STANDARD', 'HIGH_FREQUENCY', or 'PITR'",
				Required:            true,
			},
			"schedule_frequency": schema.StringAttribute{
				MarkdownDescription: "Frequency for the backup schedule. For STANDARD and PITR: 'DAILY', 'INTERVAL', 'WEEKLY', 'MONTHLY', 'ANNUALLY'. For HIGH_FREQUENCY: 'INTERVAL', 'DAILY', 'WEEKLY', 'MONTHLY', 'ANNUALLY'",
				Required:            true,
			},
			"time_of_day_hour": schema.Int64Attribute{
				MarkdownDescription: "Hour of the day for the backup schedule (0-23). Used for DAILY, WEEKLY, MONTHLY, and ANNUALLY frequencies",
				Optional:            true,
			},
			"time_of_day_minutes": schema.Int64Attribute{
				MarkdownDescription: "Minutes of the hour for the backup schedule (0-59). Used for DAILY, WEEKLY, MONTHLY, and ANNUALLY frequencies",
				Optional:            true,
			},
			"interval_minutes": schema.Int64Attribute{
				MarkdownDescription: "Interval in minutes for backup schedule. For HIGH_FREQUENCY INTERVAL: any minute value. For STANDARD/PITR INTERVAL: will be converted to hours and must result in 6, 8, or 12 hours (360, 480, or 720 minutes).",
				Optional:            true,
			},
			"start_window_minutes": schema.Int64Attribute{
				MarkdownDescription: "Start window in minutes after the scheduled time (240-1320). Minimum 240 minutes (4 hours), maximum 1320 minutes (22 hours). Defaults to 240.",
				Optional:            true,
			},
			"high_frequency_resource_types": schema.ListAttribute{
				MarkdownDescription: "List of resource types for HIGH_FREQUENCY backup policies. Supported values: 'AWS_S3', 'AWS_DYNAMO_DB'. Required for HIGH_FREQUENCY policies.",
				ElementType:         types.StringType,
				Optional:            true,
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
		Blocks: map[string]schema.Block{
			"backup_schedules": schema.ListNestedBlock{
				MarkdownDescription: "List of backup schedules, each containing a vault_id and retention_days",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"vault_id": schema.StringAttribute{
							MarkdownDescription: "Vault ID to associate with the backup schedule",
							Required:            true,
						},
						"retention_days": schema.Int64Attribute{
							MarkdownDescription: "Number of days to retain backups for this schedule",
							Required:            true,
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
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

	resourceSelector := externalEonSdkAPI.NewBackupPolicyResourceSelector(
		externalEonSdkAPI.ResourceSelectorMode(data.ResourceSelectionMode.ValueString()),
	)

	if data.ResourceSelectionMode.ValueString() == "CONDITIONAL" && !data.ConditionalExpression.IsNull() {
		expression, err := createBackupPolicyExpression(ctx, &data)
		if err != nil {
			resp.Diagnostics.AddError("Invalid Conditional Expression", fmt.Sprintf("Failed to create conditional expression: %s", err))
			return
		}
		resourceSelector.SetExpression(*expression)
	}

	if !data.ResourceInclusionOverride.IsNull() {
		var inclusionOverride []string
		diags := data.ResourceInclusionOverride.ElementsAs(ctx, &inclusionOverride, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		resourceSelector.SetResourceInclusionOverride(inclusionOverride)
	}

	if !data.ResourceExclusionOverride.IsNull() {
		var exclusionOverride []string
		diags := data.ResourceExclusionOverride.ElementsAs(ctx, &exclusionOverride, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		resourceSelector.SetResourceExclusionOverride(exclusionOverride)
	}

	backupPlan := externalEonSdkAPI.NewBackupPolicyPlan(
		externalEonSdkAPI.BackupPolicyType(data.BackupPolicyType.ValueString()),
	)

	switch data.BackupPolicyType.ValueString() {
	case "STANDARD", "PITR":
		scheduleConfig, err := createStandardScheduleConfig(&data)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Schedule Configuration",
				fmt.Sprintf("Failed to create schedule configuration for %s policy: %s", data.BackupPolicyType.ValueString(), err),
			)
			return
		}

		var backupSchedules []externalEonSdkAPI.StandardBackupSchedules
		var schedules []BackupScheduleModel
		diags := data.BackupSchedules.ElementsAs(ctx, &schedules, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		for _, schedule := range schedules {
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
		highFreqScheduleConfig, err := createHighFrequencyScheduleConfig(&data)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Schedule Configuration",
				fmt.Sprintf("Failed to create high frequency schedule configuration: %s", err),
			)
			return
		}

		var backupSchedules []externalEonSdkAPI.HighFrequencyBackupSchedules
		var schedules []BackupScheduleModel
		diags := data.BackupSchedules.ElementsAs(ctx, &schedules, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		for _, schedule := range schedules {
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
				*highFreqScheduleConfig,
				retentionDays,
			)
			backupSchedules = append(backupSchedules, *backupSchedule)
		}

		resourceTypes, err := createHighFrequencyResourceTypes(ctx, &data)
		if err != nil {
			resp.Diagnostics.AddError("Invalid Resource Types", fmt.Sprintf("Failed to create high frequency resource types: %s", err))
			return
		}
		highFreqPlan := externalEonSdkAPI.NewHighFrequencyBackupPolicyPlan(
			resourceTypes,
			backupSchedules,
		)

		backupPlan.SetHighFrequencyPlan(*highFreqPlan)

	default:
		resp.Diagnostics.AddError(
			"Unsupported Backup Policy Type",
			fmt.Sprintf("Backup policy type '%s' is not supported. Supported types: STANDARD, HIGH_FREQUENCY, PITR",
				data.BackupPolicyType.ValueString()),
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
		"name":                    data.Name.ValueString(),
		"enabled":                 data.Enabled.ValueBool(),
		"resource_selection_mode": data.ResourceSelectionMode.ValueString(),
		"backup_policy_type":      data.BackupPolicyType.ValueString(),
	})

	policy, err := r.client.CreateBackupPolicy(ctx, *createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create backup policy: %s", err))
		return
	}

	data.Id = types.StringValue(policy.Id)
	data.Name = types.StringValue(policy.Name)
	data.Enabled = types.BoolValue(policy.Enabled)
	data.ResourceSelectionMode = types.StringValue(string(policy.ResourceSelector.ResourceSelectionMode))
	data.BackupPolicyType = types.StringValue(string(policy.BackupPlan.BackupPolicyType))

	if policy.ResourceSelector.ResourceInclusionOverride != nil {
		inclusionList, diags := types.ListValueFrom(ctx, types.StringType, policy.ResourceSelector.ResourceInclusionOverride)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.ResourceInclusionOverride = inclusionList
	} else if data.ResourceInclusionOverride.IsNull() {
		data.ResourceInclusionOverride = types.ListNull(types.StringType)
	}

	if policy.ResourceSelector.ResourceExclusionOverride != nil {
		exclusionList, diags := types.ListValueFrom(ctx, types.StringType, policy.ResourceSelector.ResourceExclusionOverride)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.ResourceExclusionOverride = exclusionList
	} else if data.ResourceExclusionOverride.IsNull() {
		data.ResourceExclusionOverride = types.ListNull(types.StringType)
	}

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
	data.ResourceSelectionMode = types.StringValue(string(policy.ResourceSelector.ResourceSelectionMode))
	data.BackupPolicyType = types.StringValue(string(policy.BackupPlan.BackupPolicyType))

	if policy.ResourceSelector.ResourceInclusionOverride != nil {
		inclusionList, diags := types.ListValueFrom(ctx, types.StringType, policy.ResourceSelector.ResourceInclusionOverride)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.ResourceInclusionOverride = inclusionList
	} else if data.ResourceInclusionOverride.IsNull() {
		data.ResourceInclusionOverride = types.ListNull(types.StringType)
	}

	if policy.ResourceSelector.ResourceExclusionOverride != nil {
		exclusionList, diags := types.ListValueFrom(ctx, types.StringType, policy.ResourceSelector.ResourceExclusionOverride)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.ResourceExclusionOverride = exclusionList
	} else if data.ResourceExclusionOverride.IsNull() {
		data.ResourceExclusionOverride = types.ListNull(types.StringType)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BackupPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BackupPolicyResourceModel
	var priorState BackupPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceSelector := externalEonSdkAPI.NewBackupPolicyResourceSelector(
		externalEonSdkAPI.ResourceSelectorMode(data.ResourceSelectionMode.ValueString()),
	)

	if data.ResourceSelectionMode.ValueString() == "CONDITIONAL" && !data.ConditionalExpression.IsNull() {
		expression, err := createBackupPolicyExpression(ctx, &data)
		if err != nil {
			resp.Diagnostics.AddError("Invalid Conditional Expression", fmt.Sprintf("Failed to create conditional expression: %s", err))
			return
		}
		resourceSelector.SetExpression(*expression)
	}

	if !data.ResourceInclusionOverride.IsNull() {
		var inclusionOverride []string
		diags := data.ResourceInclusionOverride.ElementsAs(ctx, &inclusionOverride, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		resourceSelector.SetResourceInclusionOverride(inclusionOverride)
	}

	if !data.ResourceExclusionOverride.IsNull() {
		var exclusionOverride []string
		diags := data.ResourceExclusionOverride.ElementsAs(ctx, &exclusionOverride, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		resourceSelector.SetResourceExclusionOverride(exclusionOverride)
	}

	backupPlan := externalEonSdkAPI.NewBackupPolicyPlan(
		externalEonSdkAPI.BackupPolicyType(data.BackupPolicyType.ValueString()),
	)

	switch data.BackupPolicyType.ValueString() {
	case "STANDARD", "PITR":
		scheduleConfig, err := createStandardScheduleConfig(&data)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Schedule Configuration",
				fmt.Sprintf("Failed to create schedule configuration for %s policy: %s", data.BackupPolicyType.ValueString(), err),
			)
			return
		}

		var backupSchedules []externalEonSdkAPI.StandardBackupSchedules
		var schedules []BackupScheduleModel
		diags := data.BackupSchedules.ElementsAs(ctx, &schedules, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		for _, schedule := range schedules {
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
		highFreqScheduleConfig, err := createHighFrequencyScheduleConfig(&data)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Schedule Configuration",
				fmt.Sprintf("Failed to create high frequency schedule configuration: %s", err),
			)
			return
		}

		var backupSchedules []externalEonSdkAPI.HighFrequencyBackupSchedules
		var schedules []BackupScheduleModel
		diags := data.BackupSchedules.ElementsAs(ctx, &schedules, false)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		for _, schedule := range schedules {
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
				*highFreqScheduleConfig,
				retentionDays,
			)
			backupSchedules = append(backupSchedules, *backupSchedule)
		}

		resourceTypes, err := createHighFrequencyResourceTypes(ctx, &data)
		if err != nil {
			resp.Diagnostics.AddError("Invalid Resource Types", fmt.Sprintf("Failed to create high frequency resource types: %s", err))
			return
		}
		highFreqPlan := externalEonSdkAPI.NewHighFrequencyBackupPolicyPlan(
			resourceTypes,
			backupSchedules,
		)

		backupPlan.SetHighFrequencyPlan(*highFreqPlan)

	default:
		resp.Diagnostics.AddError(
			"Unsupported Backup Policy Type",
			fmt.Sprintf("Backup policy type '%s' is not supported. Supported types: STANDARD, HIGH_FREQUENCY, PITR",
				data.BackupPolicyType.ValueString()),
		)
		return
	}

	updateReq := externalEonSdkAPI.NewUpdateBackupPolicyRequest(
		data.Name.ValueString(),
		*resourceSelector,
		*backupPlan,
	)

	if !data.Enabled.IsNull() {
		enabled := data.Enabled.ValueBool()
		updateReq.SetEnabled(enabled)
	}

	policy, err := r.client.UpdateBackupPolicy(ctx, data.Id.ValueString(), *updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update backup policy: %s", err))
		return
	}

	data.Name = types.StringValue(policy.Name)
	data.Enabled = types.BoolValue(policy.Enabled)
	data.ResourceSelectionMode = types.StringValue(string(policy.ResourceSelector.ResourceSelectionMode))
	data.BackupPolicyType = types.StringValue(string(policy.BackupPlan.BackupPolicyType))

	if policy.ResourceSelector.ResourceInclusionOverride != nil {
		inclusionList, diags := types.ListValueFrom(ctx, types.StringType, policy.ResourceSelector.ResourceInclusionOverride)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.ResourceInclusionOverride = inclusionList
	} else if data.ResourceInclusionOverride.IsNull() {
		data.ResourceInclusionOverride = types.ListNull(types.StringType)
	}

	if policy.ResourceSelector.ResourceExclusionOverride != nil {
		exclusionList, diags := types.ListValueFrom(ctx, types.StringType, policy.ResourceSelector.ResourceExclusionOverride)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		data.ResourceExclusionOverride = exclusionList
	} else if data.ResourceExclusionOverride.IsNull() {
		data.ResourceExclusionOverride = types.ListNull(types.StringType)
	}

	data.UpdatedAt = types.StringValue(time.Now().Format(time.RFC3339))
	data.CreatedAt = priorState.CreatedAt

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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

func createTimeOfDay(data *BackupPolicyResourceModel) (*externalEonSdkAPI.TimeOfDay, error) {
	if data.TimeOfDayHour.IsNull() || data.TimeOfDayMinutes.IsNull() {
		return nil, nil
	}

	hour, err := SafeInt32Conversion(data.TimeOfDayHour.ValueInt64())
	if err != nil {
		return nil, err
	}

	minutes, err := SafeInt32Conversion(data.TimeOfDayMinutes.ValueInt64())
	if err != nil {
		return nil, err
	}

	return externalEonSdkAPI.NewTimeOfDay(hour, minutes), nil
}

func getStartWindowMinutes(data *BackupPolicyResourceModel) (*int32, error) {
	if data.StartWindowMinutes.IsNull() {
		return nil, nil
	}
	value, err := SafeInt32Conversion(data.StartWindowMinutes.ValueInt64())
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func createDailyConfig(data *BackupPolicyResourceModel) (*externalEonSdkAPI.DailyConfig, error) {
	dailyConfig := externalEonSdkAPI.NewDailyConfigWithDefaults()

	timeOfDay, err := createTimeOfDay(data)
	if err != nil {
		return nil, err
	}
	if timeOfDay != nil {
		dailyConfig.SetTimeOfDay(*timeOfDay)
	}

	startWindow, err := getStartWindowMinutes(data)
	if err != nil {
		return nil, err
	}
	if startWindow != nil {
		dailyConfig.SetStartWindowMinutes(*startWindow)
	}

	return dailyConfig, nil
}

func createWeeklyConfig(data *BackupPolicyResourceModel) (*externalEonSdkAPI.WeeklyConfig, error) {
	weeklyConfig := externalEonSdkAPI.NewWeeklyConfigWithDefaults()

	timeOfDay, err := createTimeOfDay(data)
	if err != nil {
		return nil, err
	}
	if timeOfDay != nil {
		weeklyConfig.SetTimeOfDay(*timeOfDay)
	}

	startWindow, err := getStartWindowMinutes(data)
	if err != nil {
		return nil, err
	}
	if startWindow != nil {
		weeklyConfig.SetStartWindowMinutes(*startWindow)
	}

	daysOfWeek := []externalEonSdkAPI.DayOfWeek{
		externalEonSdkAPI.MON,
		externalEonSdkAPI.TUE,
		externalEonSdkAPI.WED,
		externalEonSdkAPI.THU,
		externalEonSdkAPI.FRI,
	}
	weeklyConfig.SetDaysOfWeek(daysOfWeek)

	return weeklyConfig, nil
}

func createMonthlyConfig(data *BackupPolicyResourceModel) (*externalEonSdkAPI.MonthlyConfig, error) {
	monthlyConfig := externalEonSdkAPI.NewMonthlyConfigWithDefaults()

	timeOfDay, err := createTimeOfDay(data)
	if err != nil {
		return nil, err
	}
	if timeOfDay != nil {
		monthlyConfig.SetTimeOfDay(*timeOfDay)
	}

	startWindow, err := getStartWindowMinutes(data)
	if err != nil {
		return nil, err
	}
	if startWindow != nil {
		monthlyConfig.SetStartWindowMinutes(*startWindow)
	}

	daysOfMonth := []int32{1}
	monthlyConfig.SetDaysOfMonth(daysOfMonth)

	return monthlyConfig, nil
}

func createAnnuallyConfig(data *BackupPolicyResourceModel) (*externalEonSdkAPI.AnnuallyConfig, error) {
	annuallyConfig := externalEonSdkAPI.NewAnnuallyConfigWithDefaults()

	timeOfDay, err := createTimeOfDay(data)
	if err != nil {
		return nil, err
	}
	if timeOfDay != nil {
		annuallyConfig.SetTimeOfDay(*timeOfDay)
	}

	startWindow, err := getStartWindowMinutes(data)
	if err != nil {
		return nil, err
	}
	if startWindow != nil {
		annuallyConfig.SetStartWindowMinutes(*startWindow)
	}

	// Set default time of year - January 1st
	timeOfYear := externalEonSdkAPI.NewTimeOfYear(1, 1) // January 1st
	annuallyConfig.SetTimeOfYear(*timeOfYear)

	return annuallyConfig, nil
}

func createIntervalConfig(data *BackupPolicyResourceModel) (*externalEonSdkAPI.StandardIntervalConfig, error) {
	// API only allows 6, 8, or 12 hours for STANDARD/PITR interval configs
	intervalHours := int32(6) // Default to 6 hours
	if !data.IntervalMinutes.IsNull() {
		intervalMinutes, err := SafeInt32Conversion(data.IntervalMinutes.ValueInt64())
		if err != nil {
			return nil, err
		}
		requestedHours := intervalMinutes / 60
		switch {
		case requestedHours <= 6:
			intervalHours = 6
		case requestedHours <= 8:
			intervalHours = 8
		default:
			intervalHours = 12
		}
	}

	return externalEonSdkAPI.NewStandardIntervalConfig(intervalHours), nil
}

func createHighFrequencyIntervalConfig(data *BackupPolicyResourceModel) (*externalEonSdkAPI.HighFrequencyIntervalConfig, error) {
	intervalMinutes := int32(30)
	if !data.IntervalMinutes.IsNull() {
		var err error
		intervalMinutes, err = SafeInt32Conversion(data.IntervalMinutes.ValueInt64())
		if err != nil {
			return nil, err
		}
	}

	return externalEonSdkAPI.NewHighFrequencyIntervalConfig(intervalMinutes), nil
}

// createHighFrequencyResourceTypes creates the required resource types for HIGH_FREQUENCY policies
func createHighFrequencyResourceTypes(ctx context.Context, data *BackupPolicyResourceModel) ([]externalEonSdkAPI.HighFrequencyBackupResourceType, error) {
	var resourceTypes []externalEonSdkAPI.HighFrequencyBackupResourceType

	if data.HighFrequencyResourceTypes.IsNull() || data.HighFrequencyResourceTypes.IsUnknown() {
		return nil, fmt.Errorf("high_frequency_resource_types is required for HIGH_FREQUENCY backup policies")
	}

	var userResourceTypes []string
	diags := data.HighFrequencyResourceTypes.ElementsAs(ctx, &userResourceTypes, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to parse high_frequency_resource_types")
	}

	if len(userResourceTypes) == 0 {
		return nil, fmt.Errorf("high_frequency_resource_types cannot be empty for HIGH_FREQUENCY backup policies")
	}

	for _, resourceTypeStr := range userResourceTypes {
		resourceType := externalEonSdkAPI.NewHighFrequencyBackupResourceType()

		switch resourceTypeStr {
		case "AWS_S3":
			resourceType.SetResourceType(externalEonSdkAPI.AWS_S3)
		case "AWS_DYNAMO_DB":
			resourceType.SetResourceType(externalEonSdkAPI.AWS_DYNAMO_DB)
		default:
			return nil, fmt.Errorf("unsupported resource type '%s'. Supported types: AWS_S3, AWS_DYNAMO_DB", resourceTypeStr)
		}

		resourceTypes = append(resourceTypes, *resourceType)
	}

	return resourceTypes, nil
}

// createStandardScheduleConfig creates a StandardBackupScheduleConfig based on the policy type and frequency
func createStandardScheduleConfig(data *BackupPolicyResourceModel) (*externalEonSdkAPI.StandardBackupScheduleConfig, error) {
	var scheduleConfig *externalEonSdkAPI.StandardBackupScheduleConfig

	switch data.ScheduleFrequency.ValueString() {
	case string(externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_DAILY):
		scheduleConfig = externalEonSdkAPI.NewStandardBackupScheduleConfig(
			externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_DAILY,
		)
		dailyConfig, err := createDailyConfig(data)
		if err != nil {
			return nil, err
		}
		scheduleConfig.SetDailyConfig(*dailyConfig)

	case "WEEKLY":
		scheduleConfig = externalEonSdkAPI.NewStandardBackupScheduleConfig(
			externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_WEEKLY,
		)
		weeklyConfig, err := createWeeklyConfig(data)
		if err != nil {
			return nil, err
		}
		scheduleConfig.SetWeeklyConfig(*weeklyConfig)

	case "MONTHLY":
		scheduleConfig = externalEonSdkAPI.NewStandardBackupScheduleConfig(
			externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_MONTHLY,
		)
		monthlyConfig, err := createMonthlyConfig(data)
		if err != nil {
			return nil, err
		}
		scheduleConfig.SetMonthlyConfig(*monthlyConfig)

	case "ANNUALLY":
		scheduleConfig = externalEonSdkAPI.NewStandardBackupScheduleConfig(
			externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_ANNUALLY,
		)
		annuallyConfig, err := createAnnuallyConfig(data)
		if err != nil {
			return nil, err
		}
		scheduleConfig.SetAnnuallyConfig(*annuallyConfig)

	case "INTERVAL":
		scheduleConfig = externalEonSdkAPI.NewStandardBackupScheduleConfig(
			externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_INTERVAL,
		)
		intervalConfig, err := createIntervalConfig(data)
		if err != nil {
			return nil, err
		}
		scheduleConfig.SetIntervalConfig(*intervalConfig)

	default:
		return nil, fmt.Errorf("unsupported schedule frequency: %s", data.ScheduleFrequency.ValueString())
	}

	return scheduleConfig, nil
}

// createHighFrequencyScheduleConfig creates a HighFrequencyBackupScheduleConfig based on the policy type and frequency
func createHighFrequencyScheduleConfig(data *BackupPolicyResourceModel) (*externalEonSdkAPI.HighFrequencyBackupScheduleConfig, error) {
	highFreqScheduleConfig := externalEonSdkAPI.NewHighFrequencyBackupScheduleConfig()

	switch data.ScheduleFrequency.ValueString() {
	case "INTERVAL":
		highFreqScheduleConfig.SetFrequency(externalEonSdkAPI.HIGH_FREQUENCY_BACKUP_SCHEDULE_INTERVAL)
		intervalConfig, err := createHighFrequencyIntervalConfig(data)
		if err != nil {
			return nil, err
		}
		highFreqScheduleConfig.SetIntervalConfig(*intervalConfig)

	case "DAILY":
		highFreqScheduleConfig.SetFrequency(externalEonSdkAPI.HIGH_FREQUENCY_BACKUP_SCHEDULE_DAILY)
		dailyConfig, err := createDailyConfig(data)
		if err != nil {
			return nil, err
		}
		highFreqScheduleConfig.SetDailyConfig(*dailyConfig)

	case "WEEKLY":
		highFreqScheduleConfig.SetFrequency(externalEonSdkAPI.HIGH_FREQUENCY_BACKUP_SCHEDULE_WEEKLY)
		weeklyConfig, err := createWeeklyConfig(data)
		if err != nil {
			return nil, err
		}
		highFreqScheduleConfig.SetWeeklyConfig(*weeklyConfig)

	case "MONTHLY":
		highFreqScheduleConfig.SetFrequency(externalEonSdkAPI.HIGH_FREQUENCY_BACKUP_SCHEDULE_MONTHLY)
		monthlyConfig, err := createMonthlyConfig(data)
		if err != nil {
			return nil, err
		}
		highFreqScheduleConfig.SetMonthlyConfig(*monthlyConfig)

	case "ANNUALLY":
		highFreqScheduleConfig.SetFrequency(externalEonSdkAPI.HIGH_FREQUENCY_BACKUP_SCHEDULE_ANNUALLY)
		annuallyConfig, err := createAnnuallyConfig(data)
		if err != nil {
			return nil, err
		}
		highFreqScheduleConfig.SetAnnuallyConfig(*annuallyConfig)

	default:
		return nil, fmt.Errorf("unsupported schedule frequency: %s", data.ScheduleFrequency.ValueString())
	}

	return highFreqScheduleConfig, nil
}

func createBackupPolicyExpression(ctx context.Context, data *BackupPolicyResourceModel) (*externalEonSdkAPI.BackupPolicyExpression, error) {
	if data.ConditionalExpression.IsNull() {
		return nil, fmt.Errorf("conditional_expression is required for CONDITIONAL resource selection mode")
	}

	var conditionalExpr ConditionalExpressionModel
	diags := data.ConditionalExpression.As(ctx, &conditionalExpr, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("failed to parse conditional_expression")
	}

	if conditionalExpr.Group.IsNull() {
		return nil, fmt.Errorf("group is required in conditional_expression")
	}

	var groupCondition GroupConditionModel
	diags = conditionalExpr.Group.As(ctx, &groupCondition, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("failed to parse group condition")
	}

	logicalOperator := externalEonSdkAPI.LogicalOperator(groupCondition.Operator.ValueString())

	var operands []OperandModel
	diags = groupCondition.Operands.ElementsAs(ctx, &operands, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to parse operands")
	}

	var expressions []externalEonSdkAPI.BackupPolicyExpression
	for _, operand := range operands {
		expr := externalEonSdkAPI.NewBackupPolicyExpression()

		if !operand.ResourceType.IsNull() {
			var resourceTypeCond ResourceTypeConditionModel
			diags = operand.ResourceType.As(ctx, &resourceTypeCond, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, fmt.Errorf("failed to parse resource_type condition")
			}

			var resourceTypes []string
			diags = resourceTypeCond.ResourceTypes.ElementsAs(ctx, &resourceTypes, false)
			if diags.HasError() {
				return nil, fmt.Errorf("failed to parse resource_types")
			}

			var resourceTypeEnums []externalEonSdkAPI.ResourceType
			for _, rt := range resourceTypes {
				resourceTypeEnums = append(resourceTypeEnums, externalEonSdkAPI.ResourceType(rt))
			}

			resourceTypeCondition := externalEonSdkAPI.NewResourceTypeCondition(
				externalEonSdkAPI.ScalarOperators(resourceTypeCond.Operator.ValueString()),
				resourceTypeEnums,
			)
			expr.SetResourceType(*resourceTypeCondition)
		}

		if !operand.Environment.IsNull() {
			var environmentCond EnvironmentConditionModel
			diags = operand.Environment.As(ctx, &environmentCond, basetypes.ObjectAsOptions{})
			if diags.HasError() {
				return nil, fmt.Errorf("failed to parse environment condition")
			}

			var environments []string
			diags = environmentCond.Environments.ElementsAs(ctx, &environments, false)
			if diags.HasError() {
				return nil, fmt.Errorf("failed to parse environments")
			}

			var environmentEnums []externalEonSdkAPI.Environment
			for _, env := range environments {
				environmentEnums = append(environmentEnums, externalEonSdkAPI.Environment(env))
			}

			environmentCondition := externalEonSdkAPI.NewEnvironmentCondition(
				externalEonSdkAPI.ScalarOperators(environmentCond.Operator.ValueString()),
				environmentEnums,
			)
			expr.SetEnvironment(*environmentCondition)
		}

		expressions = append(expressions, *expr)
	}

	groupConditionApi := externalEonSdkAPI.NewBackupPolicyGroupCondition(logicalOperator, expressions)

	mainExpression := externalEonSdkAPI.NewBackupPolicyExpression()
	mainExpression.SetGroup(*groupConditionApi)

	return mainExpression, nil
}
