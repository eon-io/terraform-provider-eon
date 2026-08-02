package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &BackupPostureControlResource{}
var _ resource.ResourceWithImportState = &BackupPostureControlResource{}

func NewBackupPostureControlResource() resource.Resource {
	return &BackupPostureControlResource{}
}

type BackupPostureControlResource struct {
	client *client.EonClient
}

type BackupPostureControlResourceModel struct {
	Id               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Severity         types.String `tfsdk:"severity"`
	ResourceSelector types.Object `tfsdk:"resource_selector"`
	Rules            types.Object `tfsdk:"rules"`
}

type PostureControlResourceSelectorModel struct {
	ResourceSelectionMode types.String `tfsdk:"resource_selection_mode"`
	Expression            types.Object `tfsdk:"expression"`
}

type PostureControlRulesModel struct {
	MinimumRetention     types.List  `tfsdk:"minimum_retention"`
	MaximumRetentionDays types.Int64 `tfsdk:"maximum_retention_days"`
	MinCopies            types.Int64 `tfsdk:"min_copies"`
	CrossRegion          types.Bool  `tfsdk:"cross_region"`
	CrossAccount         types.Bool  `tfsdk:"cross_account"`
	CrossCloudProvider   types.Bool  `tfsdk:"cross_cloud_provider"`
}

type MinimumRetentionRuleModel struct {
	Frequency            types.String `tfsdk:"frequency"`
	MinimumRetentionDays types.Int64  `tfsdk:"minimum_retention_days"`
}

func (r *BackupPostureControlResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backup_posture_control"
}

func (r *BackupPostureControlResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates and manages a backup posture control. Posture controls define the backup requirements resources must meet, such as minimum retention, number of copies, and cross-region or cross-account redundancy, and Eon reports resources that violate them.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Backup posture control identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name for the backup posture control.",
				Required:            true,
			},
			"severity": schema.StringAttribute{
				MarkdownDescription: "Severity reported when a resource violates the control. Possible values: `LOW`, `MEDIUM`, `HIGH`.",
				Required:            true,
			},
			"resource_selector": schema.SingleNestedAttribute{
				MarkdownDescription: "Selects the resources the control applies to.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"resource_selection_mode": schema.StringAttribute{
						MarkdownDescription: "Resource selection mode: 'ALL' or 'CONDITIONAL'",
						Required:            true,
					},
					"expression": conditionalExpressionSchema(),
				},
			},
			"rules": schema.SingleNestedAttribute{
				MarkdownDescription: "Backup requirements enforced by the control. At least one rule should be set.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"minimum_retention": schema.ListNestedAttribute{
						MarkdownDescription: "Minimum retention required per backup frequency.",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"frequency": schema.StringAttribute{
									MarkdownDescription: "The backup cadence this minimum retention applies to. Possible values: `DAILY`, `WEEKLY`, `MONTHLY`, `ANNUAL`.",
									Required:            true,
								},
								"minimum_retention_days": schema.Int64Attribute{
									MarkdownDescription: "The minimum backup retention period, in days.",
									Required:            true,
								},
							},
						},
					},
					"maximum_retention_days": schema.Int64Attribute{
						MarkdownDescription: "The maximum backup retention period, in days.",
						Optional:            true,
					},
					"min_copies": schema.Int64Attribute{
						MarkdownDescription: "The minimum number of backup copies that must exist.",
						Optional:            true,
					},
					"cross_region": schema.BoolAttribute{
						MarkdownDescription: "Whether at least one backup copy must be stored in a different region.",
						Optional:            true,
					},
					"cross_account": schema.BoolAttribute{
						MarkdownDescription: "Whether at least one backup copy must be stored in a different cloud account.",
						Optional:            true,
					},
					"cross_cloud_provider": schema.BoolAttribute{
						MarkdownDescription: "Whether at least one backup copy must be stored in a different cloud provider.",
						Optional:            true,
					},
				},
			},
		},
	}
}

func (r *BackupPostureControlResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.EonClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *BackupPostureControlResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BackupPostureControlResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	selector, err := createPostureControlResourceSelector(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Configuration", err.Error())
		return
	}

	rules, err := createPostureControlRules(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Configuration", err.Error())
		return
	}

	tflog.Debug(ctx, "Creating backup posture control", map[string]interface{}{
		"name":     data.Name.ValueString(),
		"severity": data.Severity.ValueString(),
	})

	createReq := externalEonSdkAPI.NewCreateBackupPostureControlRequest(
		data.Name.ValueString(),
		externalEonSdkAPI.Severity(data.Severity.ValueString()),
		*externalEonSdkAPI.NewNullableBackupPostureControlResourceSelector(selector),
		*externalEonSdkAPI.NewNullableBackupPostureControlRules(rules),
	)

	control, err := r.client.CreateBackupPostureControl(ctx, *createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create backup posture control: %s", err))
		return
	}

	data.Id = types.StringValue(control.Id)
	data.Name = types.StringValue(control.Name)
	data.Severity = types.StringValue(string(control.Severity))

	tflog.Info(ctx, "Created backup posture control", map[string]interface{}{
		"id": control.Id,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BackupPostureControlResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BackupPostureControlResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	control, err := r.client.GetBackupPostureControl(ctx, data.Id.ValueString())
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			tflog.Warn(ctx, "Backup posture control not found, removing from state", map[string]interface{}{
				"id": data.Id.ValueString(),
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read backup posture control: %s", err))
		return
	}

	data.Id = types.StringValue(control.Id)
	data.Name = types.StringValue(control.Name)
	data.Severity = types.StringValue(string(control.Severity))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BackupPostureControlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BackupPostureControlResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	selector, err := createPostureControlResourceSelector(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Configuration", err.Error())
		return
	}

	rules, err := createPostureControlRules(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Configuration", err.Error())
		return
	}

	tflog.Debug(ctx, "Updating backup posture control", map[string]interface{}{
		"id": data.Id.ValueString(),
	})

	updateReq := externalEonSdkAPI.NewUpdateBackupPostureControlRequest(
		data.Name.ValueString(),
		externalEonSdkAPI.Severity(data.Severity.ValueString()),
		*externalEonSdkAPI.NewNullableBackupPostureControlResourceSelector(selector),
		*externalEonSdkAPI.NewNullableBackupPostureControlRules(rules),
	)

	control, err := r.client.UpdateBackupPostureControl(ctx, data.Id.ValueString(), *updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update backup posture control: %s", err))
		return
	}

	data.Id = types.StringValue(control.Id)
	data.Name = types.StringValue(control.Name)
	data.Severity = types.StringValue(string(control.Severity))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BackupPostureControlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BackupPostureControlResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteBackupPostureControl(ctx, data.Id.ValueString())
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			// Already gone — nothing to do.
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete backup posture control: %s", err))
		return
	}
}

func (r *BackupPostureControlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// createPostureControlResourceSelector converts the resource_selector object
// into the SDK selector. The conditional expression reuses the backup policy
// expression builder and converts the result, since both APIs share the same
// condition types.
func createPostureControlResourceSelector(ctx context.Context, data *BackupPostureControlResourceModel) (*externalEonSdkAPI.BackupPostureControlResourceSelector, error) {
	var selectorModel PostureControlResourceSelectorModel
	diags := data.ResourceSelector.As(ctx, &selectorModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("failed to parse resource selector")
	}

	mode := externalEonSdkAPI.ResourceSelectorMode(selectorModel.ResourceSelectionMode.ValueString())
	selector := externalEonSdkAPI.NewBackupPostureControlResourceSelector(mode)

	if !selectorModel.Expression.IsNull() {
		policyExpr, err := createBackupPolicyExpression(ctx, &ResourceSelectorModel{Expression: selectorModel.Expression})
		if err != nil {
			return nil, fmt.Errorf("failed to create conditional expression: %w", err)
		}
		selector.SetExpression(*policyExpressionToPostureExpression(policyExpr))
	} else if mode == externalEonSdkAPI.ResourceSelectorMode("CONDITIONAL") {
		return nil, fmt.Errorf("expression is required for CONDITIONAL resource selection mode")
	}

	return selector, nil
}

// policyExpressionToPostureExpression maps a backup policy expression onto the
// posture control expression. Both SDK types share the same condition types;
// only the group wrapper differs, so groups are converted recursively.
func policyExpressionToPostureExpression(src *externalEonSdkAPI.BackupPolicyExpression) *externalEonSdkAPI.BackupPostureControlExpression {
	if src == nil {
		return nil
	}
	dst := externalEonSdkAPI.NewBackupPostureControlExpression()

	if v := src.ResourceType.Get(); v != nil {
		dst.SetResourceType(*v)
	}
	if v := src.DataClasses.Get(); v != nil {
		dst.SetDataClasses(*v)
	}
	if v := src.Environment.Get(); v != nil {
		dst.SetEnvironment(*v)
	}
	if v := src.Apps.Get(); v != nil {
		dst.SetApps(*v)
	}
	if v := src.CloudProvider.Get(); v != nil {
		dst.SetCloudProvider(*v)
	}
	if v := src.AccountId.Get(); v != nil {
		dst.SetAccountId(*v)
	}
	if v := src.SourceRegion.Get(); v != nil {
		dst.SetSourceRegion(*v)
	}
	if v := src.Vpc.Get(); v != nil {
		dst.SetVpc(*v)
	}
	if v := src.Subnets.Get(); v != nil {
		dst.SetSubnets(*v)
	}
	if v := src.ResourceGroupName.Get(); v != nil {
		dst.SetResourceGroupName(*v)
	}
	if v := src.ResourceName.Get(); v != nil {
		dst.SetResourceName(*v)
	}
	if v := src.ResourceId.Get(); v != nil {
		dst.SetResourceId(*v)
	}
	if v := src.TagKeys.Get(); v != nil {
		dst.SetTagKeys(*v)
	}
	if v := src.TagKeyValues.Get(); v != nil {
		dst.SetTagKeyValues(*v)
	}
	if g := src.Group.Get(); g != nil {
		operands := make([]externalEonSdkAPI.BackupPostureControlExpression, 0, len(g.Operands))
		for i := range g.Operands {
			operands = append(operands, *policyExpressionToPostureExpression(&g.Operands[i]))
		}
		dst.SetGroup(*externalEonSdkAPI.NewBackupPostureControlGroupCondition(g.Operator, operands))
	}

	return dst
}

// createPostureControlRules converts the rules object into the SDK rules model.
func createPostureControlRules(ctx context.Context, data *BackupPostureControlResourceModel) (*externalEonSdkAPI.BackupPostureControlRules, error) {
	var rulesModel PostureControlRulesModel
	diags := data.Rules.As(ctx, &rulesModel, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("failed to parse rules")
	}

	rules := externalEonSdkAPI.NewBackupPostureControlRules()

	if !rulesModel.MinimumRetention.IsNull() {
		var minRetentionModels []MinimumRetentionRuleModel
		diags = rulesModel.MinimumRetention.ElementsAs(ctx, &minRetentionModels, false)
		if diags.HasError() {
			return nil, fmt.Errorf("failed to parse minimum retention rules")
		}

		minRetentionRules := make([]externalEonSdkAPI.MinRetentionRule, 0, len(minRetentionModels))
		for _, m := range minRetentionModels {
			days, err := SafeInt32Conversion(m.MinimumRetentionDays.ValueInt64())
			if err != nil {
				return nil, fmt.Errorf("invalid minimum_retention_days: %w", err)
			}
			minRetentionRules = append(minRetentionRules, *externalEonSdkAPI.NewMinRetentionRule(days, m.Frequency.ValueString()))
		}
		rules.SetMinimumRetention(minRetentionRules)
	}

	if !rulesModel.MaximumRetentionDays.IsNull() {
		days, err := SafeInt32Conversion(rulesModel.MaximumRetentionDays.ValueInt64())
		if err != nil {
			return nil, fmt.Errorf("invalid maximum_retention_days: %w", err)
		}
		rules.SetMaximumRetention(*externalEonSdkAPI.NewMaxRetentionRule(days))
	}

	if !rulesModel.MinCopies.IsNull() {
		copies, err := SafeInt32Conversion(rulesModel.MinCopies.ValueInt64())
		if err != nil {
			return nil, fmt.Errorf("invalid min_copies: %w", err)
		}
		rules.SetNumberOfCopies(*externalEonSdkAPI.NewNumberOfCopiesRule(copies))
	}

	if !rulesModel.CrossRegion.IsNull() {
		rules.SetCrossRegion(rulesModel.CrossRegion.ValueBool())
	}

	if !rulesModel.CrossAccount.IsNull() {
		rules.SetCrossAccount(rulesModel.CrossAccount.ValueBool())
	}

	if !rulesModel.CrossCloudProvider.IsNull() {
		rules.SetCrossCloudProvider(rulesModel.CrossCloudProvider.ValueBool())
	}

	return rules, nil
}
