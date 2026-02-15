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

var _ resource.Resource = &IdpGroupResource{}
var _ resource.ResourceWithImportState = &IdpGroupResource{}

func NewIdpGroupResource() resource.Resource {
	return &IdpGroupResource{}
}

type IdpGroupResource struct {
	client *client.EonClient
}

type IdpGroupResourceModel struct {
	Id              types.String `tfsdk:"id"`
	IdpId           types.String `tfsdk:"idp_id"`
	ProviderGroupId types.String `tfsdk:"provider_group_id"`
	RoleIds         types.List   `tfsdk:"role_ids"`
}

func (r *IdpGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_idp_group"
}

func (r *IdpGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Creates and manages an IDP (Identity Provider) group mapping. An IDP group maps a group from your Identity Provider (e.g. Okta, SAML) to one or more Eon roles.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "System-generated unique identifier for the IDP group.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"idp_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the Identity Provider this group belongs to.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"provider_group_id": schema.StringAttribute{
				MarkdownDescription: "The group identifier from the Identity Provider (e.g. Okta group ID, SAML group name).",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"role_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of Eon role IDs assigned to this IDP group.",
				Required:            true,
			},
		},
	}
}

func (r *IdpGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IdpGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IdpGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleIds, diags := listOfStringFromModel(ctx, data.RoleIds)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := externalEonSdkAPI.NewCreateIdpGroupRequest(
		data.IdpId.ValueString(),
		data.ProviderGroupId.ValueString(),
		roleIds,
	)

	tflog.Debug(ctx, "Creating IDP group", map[string]interface{}{
		"idp_id":            data.IdpId.ValueString(),
		"provider_group_id": data.ProviderGroupId.ValueString(),
	})

	group, err := r.client.CreateIdpGroup(ctx, *createReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create IDP group: %s", err))
		return
	}

	data.Id = types.StringValue(group.Id)
	data.IdpId = types.StringValue(group.IdpId)
	data.ProviderGroupId = types.StringValue(group.ProviderGroupId)
	roleIdsVal, diags := types.ListValueFrom(ctx, types.StringType, stringSliceToInterface(group.GetRoleIds()))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.RoleIds = roleIdsVal

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IdpGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IdpGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.client.GetIdpGroup(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read IDP group: %s", err))
		return
	}

	data.Id = types.StringValue(group.Id)
	data.IdpId = types.StringValue(group.IdpId)
	data.ProviderGroupId = types.StringValue(group.ProviderGroupId)
	roleIdsVal, diags := types.ListValueFrom(ctx, types.StringType, stringSliceToInterface(group.GetRoleIds()))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.RoleIds = roleIdsVal

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IdpGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan IdpGroupResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleIds, diags := listOfStringFromModel(ctx, plan.RoleIds)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := externalEonSdkAPI.NewUpdateIdpGroupRequest(roleIds)

	tflog.Debug(ctx, "Updating IDP group roles", map[string]interface{}{
		"id": plan.Id.ValueString(),
	})

	group, err := r.client.UpdateIdpGroup(ctx, plan.Id.ValueString(), *updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update IDP group: %s", err))
		return
	}

	plan.Id = types.StringValue(group.Id)
	plan.IdpId = types.StringValue(group.IdpId)
	plan.ProviderGroupId = types.StringValue(group.ProviderGroupId)
	roleIdsVal, diags := types.ListValueFrom(ctx, types.StringType, stringSliceToInterface(group.GetRoleIds()))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.RoleIds = roleIdsVal

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IdpGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IdpGroupResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting IDP group", map[string]interface{}{
		"id": data.Id.ValueString(),
	})

	if err := r.client.DeleteIdpGroup(ctx, data.Id.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete IDP group: %s", err))
		return
	}
}

func (r *IdpGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)

	group, err := r.client.GetIdpGroup(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read IDP group during import: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("idp_id"), group.IdpId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("provider_group_id"), group.ProviderGroupId)...)
	roleIdsVal, diags := types.ListValueFrom(ctx, types.StringType, stringSliceToInterface(group.GetRoleIds()))
	if !diags.HasError() {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_ids"), roleIdsVal)...)
	} else {
		resp.Diagnostics.Append(diags...)
	}
}

// listOfStringFromModel converts a types.List of String to []string.
func listOfStringFromModel(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var out []string
	diags := list.ElementsAs(ctx, &out, false)
	return out, diags
}
