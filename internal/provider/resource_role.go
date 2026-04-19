package provider

import (
	"context"
	"fmt"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &RoleResource{}
var _ resource.ResourceWithImportState = &RoleResource{}

func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

type RoleResource struct {
	client *client.EonClient
}

type RoleResourceModel struct {
	Id                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	PermissionGrants         types.List   `tfsdk:"permission_grants"`
	AccessConditions         types.List   `tfsdk:"access_conditions"`
	RestoreDestinationLimits types.Object `tfsdk:"restore_destination_limits"`
}

type RolePermissionGrantModel struct {
	Permission        types.String `tfsdk:"permission"`
	AccessConditionId types.String `tfsdk:"access_condition_id"`
}

type RoleAccessConditionModel struct {
	Id         types.String `tfsdk:"id"`
	Effect     types.String `tfsdk:"effect"`
	Expression types.Object `tfsdk:"expression"`
}

func (r *RoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *RoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates and manages a custom role in Eon. Custom roles define a set of permissions and optional access conditions that restrict which resources the permissions apply to. Built-in roles cannot be created or modified.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "System-generated unique identifier for the role.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Display name of the role. Must be unique in your Eon account.",
				Required:            true,
			},
			"permission_grants": schema.ListNestedAttribute{
				MarkdownDescription: "List of permissions granted by the role.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"permission": schema.StringAttribute{
							MarkdownDescription: "The permission identifier (e.g. vaults.manage, dashboard.view).",
							Required:            true,
						},
						"access_condition_id": schema.StringAttribute{
							MarkdownDescription: "Optional ID of an access condition that restricts the resources this permission applies to. Must match the id of an entry in access_conditions.",
							Optional:            true,
						},
					},
				},
			},
			"access_conditions": schema.ListNestedAttribute{
				MarkdownDescription: "Optional list of access conditions that can be referenced by permission_grants to restrict the scope of permissions.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Unique identifier for this access condition, used in permission_grants.access_condition_id.",
							Required:            true,
						},
						"effect": schema.StringAttribute{
							MarkdownDescription: "Effect of the condition (e.g. ALLOW, DENY).",
							Required:            true,
						},
						"expression": schema.SingleNestedAttribute{
							MarkdownDescription: RoleExprDescExpression,
							Optional:            true,
							Attributes:          roleAccessConditionExpressionSchema(),
						},
					},
				},
			},
			"restore_destination_limits": schema.SingleNestedAttribute{
				MarkdownDescription: "Optional limits on which restore destination accounts are allowed or denied for this role.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"effect": schema.StringAttribute{
						MarkdownDescription: "Effect of the limit (e.g. ALLOW, DENY).",
						Required:            true,
					},
					"restore_account_provider_ids": schema.ListAttribute{
						MarkdownDescription: "List of restore account provider IDs to match against.",
						Required:            true,
						ElementType:         types.StringType,
					},
				},
			},
		},
	}
}

func (r *RoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permGrants, diags := rolePermissionGrantsToSDK(ctx, data.PermissionGrants)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := externalEonSdkAPI.NewCreateRoleRequest(data.Name.ValueString(), permGrants)

	if !data.AccessConditions.IsNull() && !data.AccessConditions.IsUnknown() {
		accessConds, diags := roleAccessConditionsToSDK(ctx, data.AccessConditions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.AccessConditions = accessConds
	}

	if !data.RestoreDestinationLimits.IsNull() && !data.RestoreDestinationLimits.IsUnknown() {
		rdl, diags := restoreDestinationLimitsToSDK(ctx, data.RestoreDestinationLimits)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if rdl != nil {
			createReq.SetRestoreDestinationLimits(*rdl)
		}
	}

	tflog.Debug(ctx, "Creating role", map[string]interface{}{
		"name": data.Name.ValueString(),
	})

	role, err := r.client.CreateRole(ctx, *createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role: %s", err))
		return
	}

	r.setModelFromRole(ctx, &data, role)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	role, err := r.client.GetRole(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role: %s", err))
		return
	}

	r.setModelFromRole(ctx, &data, role)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	permGrants, diags := rolePermissionGrantsToSDK(ctx, plan.PermissionGrants)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := externalEonSdkAPI.NewUpdateRoleRequest(plan.Name.ValueString(), permGrants)

	if !plan.AccessConditions.IsNull() && !plan.AccessConditions.IsUnknown() {
		accessConds, diags := roleAccessConditionsToSDK(ctx, plan.AccessConditions)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.AccessConditions = accessConds
	}

	if !plan.RestoreDestinationLimits.IsNull() && !plan.RestoreDestinationLimits.IsUnknown() {
		rdl, diags := restoreDestinationLimitsToSDK(ctx, plan.RestoreDestinationLimits)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		if rdl != nil {
			updateReq.SetRestoreDestinationLimits(*rdl)
		}
	} else {
		updateReq.SetRestoreDestinationLimitsNil()
	}

	tflog.Debug(ctx, "Updating role", map[string]interface{}{
		"id": plan.Id.ValueString(),
	})

	role, err := r.client.UpdateRole(ctx, plan.Id.ValueString(), *updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update role: %s", err))
		return
	}

	r.setModelFromRole(ctx, &plan, role)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting role", map[string]interface{}{
		"id": data.Id.ValueString(),
	})

	if err := r.client.DeleteRole(ctx, data.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete role: %s", err))
		return
	}
}

func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)

	role, err := r.client.GetRole(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role during import: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), role.GetName())...)
	permGrantsVal, diags := permissionGrantsFromSDK(ctx, role.GetPermissionGrants())
	if !diags.HasError() {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("permission_grants"), permGrantsVal)...)
	} else {
		resp.Diagnostics.Append(diags...)
	}

	accessCondsVal, diags := flattenRoleAccessConditions(ctx, role.GetAccessConditions())
	if !diags.HasError() && !accessCondsVal.IsNull() {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("access_conditions"), accessCondsVal)...)
	} else if diags.HasError() {
		resp.Diagnostics.Append(diags...)
	}

	if role.HasRestoreDestinationLimits() {
		rdl := role.GetRestoreDestinationLimits()
		rdlVal, d := flattenRestoreDestinationLimits(rdl)
		if d.HasError() {
			resp.Diagnostics.Append(d...)
		} else {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("restore_destination_limits"), rdlVal)...)
		}
	}
}

func (r *RoleResource) setModelFromRole(ctx context.Context, data *RoleResourceModel, role *externalEonSdkAPI.Role) {
	data.Id = types.StringValue(role.GetId())
	data.Name = types.StringValue(role.GetName())
	permGrantsVal, diags := permissionGrantsFromSDK(ctx, role.GetPermissionGrants())
	if diags.HasError() {
		return
	}
	data.PermissionGrants = permGrantsVal
	accessCondsVal, diags := flattenRoleAccessConditions(ctx, role.GetAccessConditions())
	if !diags.HasError() {
		data.AccessConditions = accessCondsVal
	}
	if role.HasRestoreDestinationLimits() {
		rdl := role.GetRestoreDestinationLimits()
		rdlVal, d := flattenRestoreDestinationLimits(rdl)
		if !d.HasError() {
			data.RestoreDestinationLimits = rdlVal
		}
	} else {
		data.RestoreDestinationLimits = types.ObjectNull(restoreDestinationLimitsAttrTypes)
	}
}

// rolePermissionGrantsToSDK converts Terraform permission_grants to SDK []PermissionGrantInput.
func rolePermissionGrantsToSDK(ctx context.Context, list types.List) ([]externalEonSdkAPI.PermissionGrantInput, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var models []RolePermissionGrantModel
	diags := list.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]externalEonSdkAPI.PermissionGrantInput, 0, len(models))
	for _, m := range models {
		p := externalEonSdkAPI.NewPermissionGrantInput(externalEonSdkAPI.PermissionType(m.Permission.ValueString()))
		if !m.AccessConditionId.IsNull() && m.AccessConditionId.ValueString() != "" {
			acId := m.AccessConditionId.ValueString()
			p.AccessConditionId = &acId
		}
		out = append(out, *p)
	}
	return out, nil
}
