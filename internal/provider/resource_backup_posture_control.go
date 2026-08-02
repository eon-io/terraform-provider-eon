package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

func (r *BackupPostureControlResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backup_posture_control"
}

func (r *BackupPostureControlResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates and manages a backup posture control in an Eon project. A backup posture control selects resources via `resource_selector` and defines backup requirements those resources must meet via `rules`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "System-generated unique identifier for the backup posture control.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name of the backup posture control.",
				Required:            true,
			},
			"severity": schema.StringAttribute{
				MarkdownDescription: "Severity assigned to the control and to the violations it raises. Possible values: `HIGH`, `MEDIUM`, `LOW`.",
				Required:            true,
			},
			"resource_selector": schema.SingleNestedAttribute{
				MarkdownDescription: "Selects which resources the control applies to.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"resource_selection_mode": schema.StringAttribute{
						MarkdownDescription: "Mode that determines how resources are selected. Possible values: `ALL`, `NONE`, `CONDITIONAL`.",
						Required:            true,
					},
					"expression": schema.SingleNestedAttribute{
						MarkdownDescription: "Resource selector expression used when `resource_selection_mode` is `CONDITIONAL`. Set exactly one condition type (or a `group`).",
						Optional:            true,
						Attributes:          roleAccessConditionExpressionSchema(),
					},
				},
			},
			"rules": schema.SingleNestedAttribute{
				MarkdownDescription: "Backup requirements a resource must satisfy to pass the control. Every rule is optional: a rule you set is evaluated, and one you omit is not.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"minimum_retention": schema.ListNestedAttribute{
						MarkdownDescription: "Minimum retention required per backup frequency (daily, weekly, monthly, annual).",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"minimum_retention": schema.Int64Attribute{
									MarkdownDescription: "The minimum backup retention period, in days.",
									Required:            true,
								},
								"frequency": schema.StringAttribute{
									MarkdownDescription: "The backup cadence this minimum retention applies to.",
									Required:            true,
								},
							},
						},
					},
					"maximum_retention": schema.SingleNestedAttribute{
						MarkdownDescription: "Maximum allowed retention for backups covered by the control.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"maximum_retention": schema.Int64Attribute{
								MarkdownDescription: "The maximum backup retention period, in days.",
								Required:            true,
							},
						},
					},
					"number_of_copies": schema.SingleNestedAttribute{
						MarkdownDescription: "Minimum number of backup copies required.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"min_copies": schema.Int64Attribute{
								MarkdownDescription: "Minimum number of backup copies.",
								Required:            true,
							},
						},
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

	createReq, diags := backupPostureControlToCreateRequest(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating backup posture control", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	control, err := r.client.CreateBackupPostureControl(ctx, *createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create backup posture control: %s", err))
		return
	}

	resp.Diagnostics.Append(setBackupPostureControlModelFromAPI(ctx, &data, control)...)
	if resp.Diagnostics.HasError() {
		return
	}

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
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read backup posture control: %s", err))
		return
	}

	resp.Diagnostics.Append(setBackupPostureControlModelFromAPI(ctx, &data, control)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BackupPostureControlResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BackupPostureControlResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq, diags := backupPostureControlToUpdateRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating backup posture control", map[string]interface{}{
		"id": plan.Id.ValueString(),
	})

	control, err := r.client.UpdateBackupPostureControl(ctx, plan.Id.ValueString(), *updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update backup posture control: %s", err))
		return
	}

	resp.Diagnostics.Append(setBackupPostureControlModelFromAPI(ctx, &plan, control)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BackupPostureControlResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BackupPostureControlResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting backup posture control", map[string]interface{}{
		"id": data.Id.ValueString(),
	})

	if err := r.client.DeleteBackupPostureControl(ctx, data.Id.ValueString()); err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete backup posture control: %s", err))
		return
	}
}

func (r *BackupPostureControlResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func backupPostureControlToCreateRequest(ctx context.Context, data BackupPostureControlResourceModel) (*externalEonSdkAPI.CreateBackupPostureControlRequest, diag.Diagnostics) {
	selector, diags := backupPostureControlResourceSelectorToSDK(ctx, data.ResourceSelector)
	if diags.HasError() {
		return nil, diags
	}
	rules, diags := backupPostureControlRulesToSDK(ctx, data.Rules)
	if diags.HasError() {
		return nil, diags
	}

	req := externalEonSdkAPI.NewCreateBackupPostureControlRequest(
		data.Name.ValueString(),
		externalEonSdkAPI.Severity(data.Severity.ValueString()),
		*externalEonSdkAPI.NewNullableBackupPostureControlResourceSelector(selector),
		*externalEonSdkAPI.NewNullableBackupPostureControlRules(rules),
	)
	return req, nil
}

func backupPostureControlToUpdateRequest(ctx context.Context, data BackupPostureControlResourceModel) (*externalEonSdkAPI.UpdateBackupPostureControlRequest, diag.Diagnostics) {
	selector, diags := backupPostureControlResourceSelectorToSDK(ctx, data.ResourceSelector)
	if diags.HasError() {
		return nil, diags
	}
	rules, diags := backupPostureControlRulesToSDK(ctx, data.Rules)
	if diags.HasError() {
		return nil, diags
	}

	req := externalEonSdkAPI.NewUpdateBackupPostureControlRequest(
		data.Name.ValueString(),
		externalEonSdkAPI.Severity(data.Severity.ValueString()),
		*externalEonSdkAPI.NewNullableBackupPostureControlResourceSelector(selector),
		*externalEonSdkAPI.NewNullableBackupPostureControlRules(rules),
	)
	return req, nil
}

func backupPostureControlResourceSelectorToSDK(ctx context.Context, obj types.Object) (*externalEonSdkAPI.BackupPostureControlResourceSelector, diag.Diagnostics) {
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Invalid Resource Selector", "resource_selector is required")}
	}

	attrs := obj.Attributes()
	mode := attrs["resource_selection_mode"].(types.String).ValueString()
	selector := externalEonSdkAPI.NewBackupPostureControlResourceSelector(externalEonSdkAPI.ResourceSelectorMode(mode))

	if exprObj, ok := attrs["expression"]; ok && !exprObj.IsNull() && !exprObj.IsUnknown() {
		accessExpr, diags := createRoleAccessConditionalExpression(ctx, exprObj.(types.Object))
		if diags.HasError() {
			return nil, diags
		}
		bpcExpr := accessConditionalExpressionToBackupPosture(accessExpr)
		if bpcExpr != nil {
			selector.SetExpression(*bpcExpr)
		}
	}

	return selector, nil
}

func backupPostureControlRulesToSDK(ctx context.Context, obj types.Object) (*externalEonSdkAPI.BackupPostureControlRules, diag.Diagnostics) {
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("Invalid Rules", "rules is required")}
	}

	attrs := obj.Attributes()
	rules := externalEonSdkAPI.NewBackupPostureControlRules()

	if minRet, ok := attrs["minimum_retention"]; ok && !minRet.IsNull() && !minRet.IsUnknown() {
		var items []struct {
			MinimumRetention types.Int64  `tfsdk:"minimum_retention"`
			Frequency        types.String `tfsdk:"frequency"`
		}
		diags := minRet.(types.List).ElementsAs(ctx, &items, false)
		if diags.HasError() {
			return nil, diags
		}
		out := make([]externalEonSdkAPI.MinRetentionRule, 0, len(items))
		for _, item := range items {
			out = append(out, *externalEonSdkAPI.NewMinRetentionRule(int32(item.MinimumRetention.ValueInt64()), item.Frequency.ValueString()))
		}
		rules.SetMinimumRetention(out)
	}

	if maxRet, ok := attrs["maximum_retention"]; ok && !maxRet.IsNull() && !maxRet.IsUnknown() {
		maxAttrs := maxRet.(types.Object).Attributes()
		rules.SetMaximumRetention(*externalEonSdkAPI.NewMaxRetentionRule(int32(maxAttrs["maximum_retention"].(types.Int64).ValueInt64())))
	}

	if copies, ok := attrs["number_of_copies"]; ok && !copies.IsNull() && !copies.IsUnknown() {
		copyAttrs := copies.(types.Object).Attributes()
		rules.SetNumberOfCopies(*externalEonSdkAPI.NewNumberOfCopiesRule(int32(copyAttrs["min_copies"].(types.Int64).ValueInt64())))
	}

	if v, ok := attrs["cross_region"]; ok && !v.IsNull() && !v.IsUnknown() {
		rules.SetCrossRegion(v.(types.Bool).ValueBool())
	}
	if v, ok := attrs["cross_account"]; ok && !v.IsNull() && !v.IsUnknown() {
		rules.SetCrossAccount(v.(types.Bool).ValueBool())
	}
	if v, ok := attrs["cross_cloud_provider"]; ok && !v.IsNull() && !v.IsUnknown() {
		rules.SetCrossCloudProvider(v.(types.Bool).ValueBool())
	}

	return rules, nil
}

func setBackupPostureControlModelFromAPI(ctx context.Context, data *BackupPostureControlResourceModel, control *externalEonSdkAPI.BackupPostureControl) diag.Diagnostics {
	var diags diag.Diagnostics

	data.Id = types.StringValue(control.GetId())
	data.Name = types.StringValue(control.GetName())
	data.Severity = types.StringValue(string(control.GetSeverity()))

	selectorVal, d := flattenBackupPostureControlResourceSelector(ctx, control.GetResourceSelector())
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	data.ResourceSelector = selectorVal

	rulesVal, d := flattenBackupPostureControlRules(ctx, control.GetRules())
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	data.Rules = rulesVal

	return diags
}

func flattenBackupPostureControlResourceSelector(ctx context.Context, selector externalEonSdkAPI.BackupPostureControlResourceSelector) (types.Object, diag.Diagnostics) {
	attrTypes := map[string]attr.Type{
		"resource_selection_mode": types.StringType,
		"expression":              types.ObjectType{AttrTypes: roleExpressionAttrTypes},
	}

	attrs := map[string]attr.Value{
		"resource_selection_mode": types.StringValue(string(selector.GetResourceSelectionMode())),
		"expression":              types.ObjectNull(roleExpressionAttrTypes),
	}

	if selector.HasExpression() {
		accessExpr := backupPostureExpressionToAccessConditional(selector.GetExpression())
		exprVal, diags := flattenAccessConditionalExpression(ctx, accessExpr)
		if diags.HasError() {
			return types.ObjectNull(attrTypes), diags
		}
		attrs["expression"] = exprVal
	}

	return types.ObjectValue(attrTypes, attrs)
}

func flattenBackupPostureControlRules(ctx context.Context, rules externalEonSdkAPI.BackupPostureControlRules) (types.Object, diag.Diagnostics) {
	minRetAttrTypes := map[string]attr.Type{
		"minimum_retention": types.Int64Type,
		"frequency":         types.StringType,
	}
	maxRetAttrTypes := map[string]attr.Type{
		"maximum_retention": types.Int64Type,
	}
	copiesAttrTypes := map[string]attr.Type{
		"min_copies": types.Int64Type,
	}
	attrTypes := map[string]attr.Type{
		"minimum_retention":    types.ListType{ElemType: types.ObjectType{AttrTypes: minRetAttrTypes}},
		"maximum_retention":    types.ObjectType{AttrTypes: maxRetAttrTypes},
		"number_of_copies":     types.ObjectType{AttrTypes: copiesAttrTypes},
		"cross_region":         types.BoolType,
		"cross_account":        types.BoolType,
		"cross_cloud_provider": types.BoolType,
	}

	attrs := map[string]attr.Value{
		"minimum_retention":    types.ListNull(types.ObjectType{AttrTypes: minRetAttrTypes}),
		"maximum_retention":    types.ObjectNull(maxRetAttrTypes),
		"number_of_copies":     types.ObjectNull(copiesAttrTypes),
		"cross_region":         types.BoolNull(),
		"cross_account":        types.BoolNull(),
		"cross_cloud_provider": types.BoolNull(),
	}

	if rules.HasMinimumRetention() {
		items := rules.GetMinimumRetention()
		elems := make([]attr.Value, 0, len(items))
		for _, item := range items {
			obj, d := types.ObjectValue(minRetAttrTypes, map[string]attr.Value{
				"minimum_retention": types.Int64Value(int64(item.GetMinimumRetention())),
				"frequency":         types.StringValue(item.GetFrequency()),
			})
			if d.HasError() {
				return types.ObjectNull(attrTypes), d
			}
			elems = append(elems, obj)
		}
		list, d := types.ListValue(types.ObjectType{AttrTypes: minRetAttrTypes}, elems)
		if d.HasError() {
			return types.ObjectNull(attrTypes), d
		}
		attrs["minimum_retention"] = list
	}

	if rules.HasMaximumRetention() {
		max := rules.GetMaximumRetention()
		obj, d := types.ObjectValue(maxRetAttrTypes, map[string]attr.Value{
			"maximum_retention": types.Int64Value(int64(max.GetMaximumRetention())),
		})
		if d.HasError() {
			return types.ObjectNull(attrTypes), d
		}
		attrs["maximum_retention"] = obj
	}

	if rules.HasNumberOfCopies() {
		copies := rules.GetNumberOfCopies()
		obj, d := types.ObjectValue(copiesAttrTypes, map[string]attr.Value{
			"min_copies": types.Int64Value(int64(copies.GetMinCopies())),
		})
		if d.HasError() {
			return types.ObjectNull(attrTypes), d
		}
		attrs["number_of_copies"] = obj
	}

	if rules.HasCrossRegion() {
		attrs["cross_region"] = types.BoolValue(rules.GetCrossRegion())
	}
	if rules.HasCrossAccount() {
		attrs["cross_account"] = types.BoolValue(rules.GetCrossAccount())
	}
	if rules.HasCrossCloudProvider() {
		attrs["cross_cloud_provider"] = types.BoolValue(rules.GetCrossCloudProvider())
	}

	return types.ObjectValue(attrTypes, attrs)
}

func accessConditionalExpressionToBackupPosture(expr *externalEonSdkAPI.AccessConditionalExpression) *externalEonSdkAPI.BackupPostureControlExpression {
	if expr == nil {
		return nil
	}

	out := externalEonSdkAPI.NewBackupPostureControlExpression()
	if expr.HasEnvironment() {
		out.SetEnvironment(expr.GetEnvironment())
	}
	if expr.HasResourceType() {
		out.SetResourceType(expr.GetResourceType())
	}
	if expr.HasDataClasses() {
		out.SetDataClasses(expr.GetDataClasses())
	}
	if expr.HasTagKeys() {
		out.SetTagKeys(expr.GetTagKeys())
	}
	if expr.HasTagKeyValues() {
		out.SetTagKeyValues(expr.GetTagKeyValues())
	}
	if expr.HasApps() {
		out.SetApps(expr.GetApps())
	}
	if expr.HasCloudProvider() {
		out.SetCloudProvider(expr.GetCloudProvider())
	}
	if expr.HasAccountId() {
		out.SetAccountId(expr.GetAccountId())
	}
	if expr.HasSourceRegion() {
		out.SetSourceRegion(expr.GetSourceRegion())
	}
	if expr.HasVpc() {
		out.SetVpc(expr.GetVpc())
	}
	if expr.HasSubnets() {
		out.SetSubnets(expr.GetSubnets())
	}
	if expr.HasResourceGroupName() {
		out.SetResourceGroupName(expr.GetResourceGroupName())
	}
	if expr.HasResourceName() {
		out.SetResourceName(expr.GetResourceName())
	}
	if expr.HasResourceId() {
		out.SetResourceId(expr.GetResourceId())
	}
	if expr.HasGroup() {
		group := expr.GetGroup()
		operands := make([]externalEonSdkAPI.BackupPostureControlExpression, 0, len(group.GetOperands()))
		for _, operand := range group.GetOperands() {
			converted := accessConditionalExpressionToBackupPosture(&operand)
			if converted != nil {
				operands = append(operands, *converted)
			}
		}
		out.SetGroup(*externalEonSdkAPI.NewBackupPostureControlGroupCondition(group.GetOperator(), operands))
	}
	return out
}

func backupPostureExpressionToAccessConditional(expr externalEonSdkAPI.BackupPostureControlExpression) externalEonSdkAPI.AccessConditionalExpression {
	out := externalEonSdkAPI.NewAccessConditionalExpression()
	if expr.HasEnvironment() {
		out.SetEnvironment(expr.GetEnvironment())
	}
	if expr.HasResourceType() {
		out.SetResourceType(expr.GetResourceType())
	}
	if expr.HasDataClasses() {
		out.SetDataClasses(expr.GetDataClasses())
	}
	if expr.HasTagKeys() {
		out.SetTagKeys(expr.GetTagKeys())
	}
	if expr.HasTagKeyValues() {
		out.SetTagKeyValues(expr.GetTagKeyValues())
	}
	if expr.HasApps() {
		out.SetApps(expr.GetApps())
	}
	if expr.HasCloudProvider() {
		out.SetCloudProvider(expr.GetCloudProvider())
	}
	if expr.HasAccountId() {
		out.SetAccountId(expr.GetAccountId())
	}
	if expr.HasSourceRegion() {
		out.SetSourceRegion(expr.GetSourceRegion())
	}
	if expr.HasVpc() {
		out.SetVpc(expr.GetVpc())
	}
	if expr.HasSubnets() {
		out.SetSubnets(expr.GetSubnets())
	}
	if expr.HasResourceGroupName() {
		out.SetResourceGroupName(expr.GetResourceGroupName())
	}
	if expr.HasResourceName() {
		out.SetResourceName(expr.GetResourceName())
	}
	if expr.HasResourceId() {
		out.SetResourceId(expr.GetResourceId())
	}
	if expr.HasGroup() {
		group := expr.GetGroup()
		operands := make([]externalEonSdkAPI.AccessConditionalExpression, 0, len(group.GetOperands()))
		for _, operand := range group.GetOperands() {
			operands = append(operands, backupPostureExpressionToAccessConditional(operand))
		}
		out.SetGroup(*externalEonSdkAPI.NewRoleAccessGroupCondition(group.GetOperator(), operands))
	}
	return *out
}
