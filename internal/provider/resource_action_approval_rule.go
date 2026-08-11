package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const actionApprovalRuleMaxExpressionDepth = 3

var _ resource.Resource = &ActionApprovalRuleResource{}
var _ resource.ResourceWithImportState = &ActionApprovalRuleResource{}

func NewActionApprovalRuleResource() resource.Resource {
	return &ActionApprovalRuleResource{}
}

type ActionApprovalRuleResource struct {
	client *client.EonClient
}

type ActionApprovalRuleResourceModel struct {
	Id                      types.String `tfsdk:"id"`
	Operation               types.String `tfsdk:"operation"`
	RequiredApprovals       types.Int64  `tfsdk:"required_approvals"`
	ApprovalWindowHours     types.Int64  `tfsdk:"approval_window_hours"`
	ExecutionWindowHours    types.Int64  `tfsdk:"execution_window_hours"`
	Description             types.String `tfsdk:"description"`
	ResourceSelector        types.Object `tfsdk:"resource_selector"`
	ApproverIdpId           types.String `tfsdk:"approver_idp_id"`
	ApproverProviderGroupId types.String `tfsdk:"approver_provider_group_id"`
	ExemptApiCredentials    types.Bool   `tfsdk:"exempt_api_credentials"`
}

type actionApprovalRuleCondition struct {
	attr      string
	values    string
	operators string
	desc      string
	tagKV     bool
}

var actionApprovalRuleConditions = []actionApprovalRuleCondition{
	{attr: "resource_type", values: "resource_types", operators: scalarOperators, desc: "Matches resources by resource type, for example `AWS_EC2` or `GCP_CLOUD_SQL`."},
	{attr: "data_classes", values: "data_classes", operators: listOperators, desc: "Matches resources by detected data class."},
	{attr: "environment", values: "environments", operators: scalarOperators, desc: "Matches resources by environment, for example `PROD` or `DEV`."},
	{attr: "apps", values: "apps", operators: listOperators, desc: "Matches resources by detected application."},
	{attr: "sensitivity_annotations", values: "sensitivity_annotations", operators: scalarOperators, desc: "Matches resources by sensitivity annotation."},
	{attr: "security_scan_conclusion", values: "security_scan_conclusions", operators: listOperators, desc: "Matches resources by security scan conclusion."},
	{attr: "cloud_provider", values: "cloud_providers", operators: scalarOperators, desc: "Matches resources by cloud provider, for example `AWS`, `AZURE` or `GCP`."},
	{attr: "account_id", values: "account_ids", operators: scalarOperators, desc: "Matches resources by the cloud account they live in."},
	{attr: "source_region", values: "regions", operators: scalarOperators, desc: "Matches resources by source region."},
	{attr: "vpc", values: "vpcs", operators: scalarOperators, desc: "Matches resources by VPC."},
	{attr: "subnets", values: "subnets", operators: listOperators, desc: "Matches resources by subnet."},
	{attr: "resource_group_name", values: "resource_group_names", operators: scalarOperators, desc: "Matches resources by resource group name (Azure)."},
	{attr: "encryption_type", values: "encryption_types", operators: listOperators, desc: "Matches resources by encryption type."},
	{attr: "resource_name", values: "resource_names", operators: stringOperators, desc: "Matches resources by resource name."},
	{attr: "resource_id", values: "resource_ids", operators: scalarOperators, desc: "Matches resources by Eon-assigned resource ID."},
	{attr: "tag_keys", values: "tag_keys", operators: listOperators, desc: "Matches resources by cloud tag key."},
	{attr: "tag_key_values", values: "tag_key_values", operators: listOperators, desc: "Matches resources by cloud tag key and value.", tagKV: true},
}

var actionApprovalRuleSchema = buildActionApprovalRuleSchema()

func buildActionApprovalRuleSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages an action approval rule: a durable multi-party approval (MPA) policy that requires one or more approvals before a sensitive operation can run. " +
			"`operation` selects the protected action; optional `resource_selector` narrows which resources the rule applies to.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the action approval rule.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"operation": schema.StringAttribute{
				MarkdownDescription: "Protected operation. Supported values: `ADD_RESTORE_ACCOUNT`, `CREATE_BACKUP_POLICY`, `RESTORE_RESOURCE`, `UPDATE_BACKUP_POLICY`, `DELETE_BACKUP_POLICY`, `REMOVE_SNAPSHOT_HOLD`. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"required_approvals": schema.Int64Attribute{
				MarkdownDescription: "Number of approvals required before the action can be executed. Defaults to `1`.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(1),
			},
			"approval_window_hours": schema.Int64Attribute{
				MarkdownDescription: "Hours the request stays open for approval before expiring.",
				Required:            true,
			},
			"execution_window_hours": schema.Int64Attribute{
				MarkdownDescription: "Hours after approval during which the approved action can be executed.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Optional description explaining the purpose of this rule.",
				Optional:            true,
			},
			"resource_selector": schema.SingleNestedAttribute{
				MarkdownDescription: "Selects the resources this rule applies to. Omit to leave resource scoping unset (operation-dependent).",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"resource_selection_mode": schema.StringAttribute{
						MarkdownDescription: "How the rule selects resources. Supported values: `ALL`, `NONE`, `CONDITIONAL`. `CONDITIONAL` requires `expression`.",
						Required:            true,
					},
					"resource_inclusion_override": schema.ListAttribute{
						MarkdownDescription: "Resource identifiers to include regardless of `resource_selection_mode`.",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"resource_exclusion_override": schema.ListAttribute{
						MarkdownDescription: "Resource identifiers to exclude regardless of `resource_selection_mode`.",
						ElementType:         types.StringType,
						Optional:            true,
					},
					"expression": schema.SingleNestedAttribute{
						MarkdownDescription: "Condition selecting the resources this rule applies to. Set exactly one attribute; use `group` to combine conditions with `AND`/`OR`. Required when `resource_selection_mode` is `CONDITIONAL`, ignored otherwise.",
						Optional:            true,
						Attributes:          actionApprovalRuleExpressionAttributes(actionApprovalRuleMaxExpressionDepth),
					},
				},
			},
			"approver_idp_id": schema.StringAttribute{
				MarkdownDescription: "UUID of the SAML identity provider connection the approver group belongs to.",
				Optional:            true,
			},
			"approver_provider_group_id": schema.StringAttribute{
				MarkdownDescription: "Provider group identifier from the IdP whose members may approve.",
				Optional:            true,
			},
			"exempt_api_credentials": schema.BoolAttribute{
				MarkdownDescription: "When true, API credential users bypass this approval rule.",
				Optional:            true,
			},
		},
	}
}

func actionApprovalRuleExpressionAttributes(depth int) map[string]schema.Attribute {
	attrs := make(map[string]schema.Attribute, len(actionApprovalRuleConditions)+1)

	for _, condition := range actionApprovalRuleConditions {
		var values schema.Attribute
		if condition.tagKV {
			values = schema.ListNestedAttribute{
				MarkdownDescription: "Tag key-value pairs to match. Omit `value` to match any value of the key.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key":   schema.StringAttribute{MarkdownDescription: "Tag key.", Required: true},
						"value": schema.StringAttribute{MarkdownDescription: "Tag value.", Optional: true},
					},
				},
			}
		} else {
			values = schema.ListAttribute{
				MarkdownDescription: "Values to match.",
				ElementType:         types.StringType,
				Required:            true,
			}
		}

		attrs[condition.attr] = schema.SingleNestedAttribute{
			MarkdownDescription: condition.desc,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":       schema.StringAttribute{MarkdownDescription: "Operator. Supported values: " + condition.operators + ".", Required: true},
				condition.values: values,
			},
		}
	}

	if depth > 1 {
		attrs["group"] = schema.SingleNestedAttribute{
			MarkdownDescription: fmt.Sprintf("Combines at least two conditions with a logical operator. Groups nest up to %d levels deep.", actionApprovalRuleMaxExpressionDepth),
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					MarkdownDescription: "Logical operator. Supported values: `AND`, `OR`.",
					Required:            true,
				},
				"operands": schema.ListNestedAttribute{
					MarkdownDescription: "The expressions to combine with `operator`. At least two.",
					Required:            true,
					NestedObject:        schema.NestedAttributeObject{Attributes: actionApprovalRuleExpressionAttributes(depth - 1)},
				},
			},
		}
	}

	return attrs
}

func (r *ActionApprovalRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_action_approval_rule"
}

func (r *ActionApprovalRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = actionApprovalRuleSchema
}

func (r *ActionApprovalRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	eonClient, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.EonClient, got: %T", req.ProviderData))
		return
	}

	r.client = eonClient
}

func (r *ActionApprovalRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ActionApprovalRuleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, diags := actionApprovalRuleCreateRequest(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating action approval rule", map[string]interface{}{"operation": data.Operation.ValueString()})

	rule, err := r.client.CreateActionApprovalRule(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create action approval rule: %s", err))
		return
	}

	resp.Diagnostics.Append(actionApprovalRuleToState(ctx, rule, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ActionApprovalRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ActionApprovalRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.GetActionApprovalRule(ctx, data.Id.ValueString())
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			tflog.Warn(ctx, "Action approval rule not found, removing from state", map[string]interface{}{"id": data.Id.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read action approval rule: %s", err))
		return
	}

	resp.Diagnostics.Append(actionApprovalRuleToState(ctx, rule, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ActionApprovalRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ActionApprovalRuleResourceModel
	var state ActionApprovalRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq, diags := actionApprovalRuleUpdateRequest(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating action approval rule", map[string]interface{}{"id": state.Id.ValueString()})

	rule, err := r.client.UpdateActionApprovalRule(ctx, state.Id.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update action approval rule: %s", err))
		return
	}

	resp.Diagnostics.Append(actionApprovalRuleToState(ctx, rule, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ActionApprovalRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ActionApprovalRuleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteActionApprovalRule(ctx, data.Id.ValueString()); err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete action approval rule: %s", err))
		return
	}
}

func (r *ActionApprovalRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func actionApprovalRuleCreateRequest(ctx context.Context, data *ActionApprovalRuleResourceModel) (externalEonSdkAPI.CreateActionApprovalRuleRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	approvalWindow, err := SafeInt32Conversion(data.ApprovalWindowHours.ValueInt64())
	if err != nil {
		diags.AddError("Invalid Attribute", fmt.Sprintf("approval_window_hours: %s", err))
		return externalEonSdkAPI.CreateActionApprovalRuleRequest{}, diags
	}
	executionWindow, err := SafeInt32Conversion(data.ExecutionWindowHours.ValueInt64())
	if err != nil {
		diags.AddError("Invalid Attribute", fmt.Sprintf("execution_window_hours: %s", err))
		return externalEonSdkAPI.CreateActionApprovalRuleRequest{}, diags
	}

	req := *externalEonSdkAPI.NewCreateActionApprovalRuleRequest(
		externalEonSdkAPI.ActionApprovalOperationType(data.Operation.ValueString()),
		approvalWindow,
		executionWindow,
	)

	if !data.RequiredApprovals.IsNull() && !data.RequiredApprovals.IsUnknown() {
		required, err := SafeInt32Conversion(data.RequiredApprovals.ValueInt64())
		if err != nil {
			diags.AddError("Invalid Attribute", fmt.Sprintf("required_approvals: %s", err))
			return externalEonSdkAPI.CreateActionApprovalRuleRequest{}, diags
		}
		req.SetRequiredApprovals(required)
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		req.SetDescription(data.Description.ValueString())
	}
	if !data.ApproverIdpId.IsNull() && !data.ApproverIdpId.IsUnknown() {
		req.SetApproverIdpId(data.ApproverIdpId.ValueString())
	}
	if !data.ApproverProviderGroupId.IsNull() && !data.ApproverProviderGroupId.IsUnknown() {
		req.SetApproverProviderGroupId(data.ApproverProviderGroupId.ValueString())
	}
	if !data.ExemptApiCredentials.IsNull() && !data.ExemptApiCredentials.IsUnknown() {
		req.SetExemptApiCredentials(data.ExemptApiCredentials.ValueBool())
	}

	selector, d := actionApprovalRuleSelectorFromTF(ctx, data.ResourceSelector)
	diags.Append(d...)
	if diags.HasError() {
		return externalEonSdkAPI.CreateActionApprovalRuleRequest{}, diags
	}
	if selector != nil {
		req.SetResourceSelector(*selector)
	}

	return req, diags
}

func actionApprovalRuleUpdateRequest(ctx context.Context, data *ActionApprovalRuleResourceModel) (externalEonSdkAPI.UpdateActionApprovalRuleRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := *externalEonSdkAPI.NewUpdateActionApprovalRuleRequest()

	if !data.RequiredApprovals.IsNull() && !data.RequiredApprovals.IsUnknown() {
		required, err := SafeInt32Conversion(data.RequiredApprovals.ValueInt64())
		if err != nil {
			diags.AddError("Invalid Attribute", fmt.Sprintf("required_approvals: %s", err))
			return req, diags
		}
		req.SetRequiredApprovals(required)
	}

	approvalWindow, err := SafeInt32Conversion(data.ApprovalWindowHours.ValueInt64())
	if err != nil {
		diags.AddError("Invalid Attribute", fmt.Sprintf("approval_window_hours: %s", err))
		return req, diags
	}
	req.SetApprovalWindowHours(approvalWindow)

	executionWindow, err := SafeInt32Conversion(data.ExecutionWindowHours.ValueInt64())
	if err != nil {
		diags.AddError("Invalid Attribute", fmt.Sprintf("execution_window_hours: %s", err))
		return req, diags
	}
	req.SetExecutionWindowHours(executionWindow)

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		req.SetDescription(data.Description.ValueString())
	}
	if data.ApproverIdpId.IsNull() {
		req.SetApproverIdpIdNil()
	} else if !data.ApproverIdpId.IsUnknown() {
		req.SetApproverIdpId(data.ApproverIdpId.ValueString())
	}
	if data.ApproverProviderGroupId.IsNull() {
		req.SetApproverProviderGroupIdNil()
	} else if !data.ApproverProviderGroupId.IsUnknown() {
		req.SetApproverProviderGroupId(data.ApproverProviderGroupId.ValueString())
	}
	if !data.ExemptApiCredentials.IsNull() && !data.ExemptApiCredentials.IsUnknown() {
		req.SetExemptApiCredentials(data.ExemptApiCredentials.ValueBool())
	}

	selector, d := actionApprovalRuleSelectorFromTF(ctx, data.ResourceSelector)
	diags.Append(d...)
	if diags.HasError() {
		return req, diags
	}
	if selector == nil {
		req.SetResourceSelectorNil()
	} else {
		req.SetResourceSelector(*selector)
	}

	return req, diags
}

func actionApprovalRuleSelectorFromTF(ctx context.Context, value types.Object) (*externalEonSdkAPI.ActionApprovalRuleResourceSelector, diag.Diagnostics) {
	var diags diag.Diagnostics

	if value.IsNull() || value.IsUnknown() {
		return nil, diags
	}

	rendered, d := tfValueToAPI(ctx, value)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	encoded, err := json.Marshal(rendered)
	if err != nil {
		diags.AddError("Invalid Request", fmt.Sprintf("Unable to encode resource_selector: %s", err))
		return nil, diags
	}

	var selector externalEonSdkAPI.ActionApprovalRuleResourceSelector
	if err := json.Unmarshal(encoded, &selector); err != nil {
		diags.AddError("Invalid Request", fmt.Sprintf("Unable to build resource_selector: %s", err))
		return nil, diags
	}

	return &selector, diags
}

func actionApprovalRuleToState(_ context.Context, rule *externalEonSdkAPI.ActionApprovalRule, data *ActionApprovalRuleResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.Id = types.StringValue(rule.GetId())
	data.Operation = types.StringValue(string(rule.GetOperation()))
	data.RequiredApprovals = types.Int64Value(int64(rule.GetRequiredApprovals()))
	data.ApprovalWindowHours = types.Int64Value(int64(rule.GetApprovalWindowHours()))
	data.ExecutionWindowHours = types.Int64Value(int64(rule.GetExecutionWindowHours()))

	if rule.HasDescription() {
		data.Description = types.StringValue(rule.GetDescription())
	} else {
		data.Description = types.StringNull()
	}

	if rule.HasApproverIdpId() && rule.ApproverIdpId.Get() != nil {
		data.ApproverIdpId = types.StringValue(rule.GetApproverIdpId())
	} else {
		data.ApproverIdpId = types.StringNull()
	}

	if rule.HasApproverProviderGroupId() && rule.ApproverProviderGroupId.Get() != nil {
		data.ApproverProviderGroupId = types.StringValue(rule.GetApproverProviderGroupId())
	} else {
		data.ApproverProviderGroupId = types.StringNull()
	}

	if rule.HasExemptApiCredentials() {
		data.ExemptApiCredentials = types.BoolValue(rule.GetExemptApiCredentials())
	} else {
		data.ExemptApiCredentials = types.BoolNull()
	}

	selectorType := actionApprovalRuleSchema.Attributes["resource_selector"].GetType()
	if rule.HasResourceSelector() && rule.ResourceSelector.Get() != nil {
		raw, err := json.Marshal(rule.GetResourceSelector())
		if err != nil {
			diags.AddError("Invalid API Response", fmt.Sprintf("Unable to encode resource_selector: %s", err))
			return diags
		}
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			diags.AddError("Invalid API Response", fmt.Sprintf("Unable to decode resource_selector: %s", err))
			return diags
		}
		selector, d := apiValueToTF(decoded, selectorType)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		data.ResourceSelector = selector.(types.Object)
	} else {
		nullSelector, d := apiValueToTF(nil, selectorType)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		data.ResourceSelector = nullSelector.(types.Object)
	}

	return diags
}
