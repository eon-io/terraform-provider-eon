package provider

import (
	"context"
	"errors"
	"fmt"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &ResourceEnvironmentOverrideResource{}
var _ resource.ResourceWithImportState = &ResourceEnvironmentOverrideResource{}

func NewResourceEnvironmentOverrideResource() resource.Resource {
	return &ResourceEnvironmentOverrideResource{}
}

type ResourceEnvironmentOverrideResource struct {
	client *client.EonClient
}

type ResourceEnvironmentOverrideResourceModel struct {
	Id          types.String `tfsdk:"id"`
	ResourceId  types.String `tfsdk:"resource_id"`
	Environment types.String `tfsdk:"environment"`
}

func (r *ResourceEnvironmentOverrideResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_environment_override"
}

func (r *ResourceEnvironmentOverrideResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manually overrides the environment for an inventory resource, replacing auto-classification.\n\n" +
			"When this resource is created or updated, the listed environment is applied as a manual override. " +
			"When destroyed, the override is removed and auto-classification resumes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the inventory resource. Mirrors `resource_id`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the inventory resource whose environment is overridden.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"environment": schema.StringAttribute{
				MarkdownDescription: "Environment to set as the manual override. Possible values: `PROD`, `PROD_INTERNAL`, `STAGE`.",
				Required:            true,
			},
		},
	}
}

func (r *ResourceEnvironmentOverrideResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ResourceEnvironmentOverrideResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceEnvironmentOverrideResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyOverride(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to override environment: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceEnvironmentOverrideResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceEnvironmentOverrideResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceId := data.ResourceId.ValueString()

	inventoryResource, err := r.client.GetResourceById(ctx, resourceId)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read environment override: %s", err))
		return
	}

	environment, ok := environmentOverrideFromInventoryResource(inventoryResource)
	if !ok {
		tflog.Debug(ctx, "Environment override is no longer present; removing from state", map[string]interface{}{
			"resource_id": resourceId,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	data.Id = types.StringValue(inventoryResource.GetId())
	data.ResourceId = types.StringValue(inventoryResource.GetId())
	data.Environment = types.StringValue(environment)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceEnvironmentOverrideResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ResourceEnvironmentOverrideResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.applyOverride(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update environment override: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceEnvironmentOverrideResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceEnvironmentOverrideResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceId := data.ResourceId.ValueString()

	tflog.Debug(ctx, "Removing environment override", map[string]interface{}{
		"resource_id": resourceId,
	})

	err := r.client.RemoveEnvironmentOverride(ctx, resourceId)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove environment override: %s", err))
		return
	}

	tflog.Debug(ctx, "Environment override removed", map[string]interface{}{
		"resource_id": resourceId,
	})
}

func (r *ResourceEnvironmentOverrideResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), req.ID)...)

	tflog.Info(ctx, "Successfully imported environment override", map[string]interface{}{
		"resource_id": req.ID,
	})
}

func (r *ResourceEnvironmentOverrideResource) applyOverride(ctx context.Context, data *ResourceEnvironmentOverrideResourceModel) error {
	resourceId := data.ResourceId.ValueString()
	environment := data.Environment.ValueString()

	tflog.Debug(ctx, "Overriding environment", map[string]interface{}{
		"resource_id": resourceId,
		"environment": environment,
	})

	overridden, err := r.client.OverrideEnvironment(ctx, resourceId, environment)
	if err != nil {
		return err
	}

	data.Id = types.StringValue(resourceId)
	data.Environment = types.StringValue(overridden)

	tflog.Debug(ctx, "Environment overridden", map[string]interface{}{
		"resource_id": resourceId,
		"environment": overridden,
	})

	return nil
}

// environmentOverrideFromInventoryResource returns the overridden environment when a manual override is present.
func environmentOverrideFromInventoryResource(inventoryResource *externalEonSdkAPI.InventoryResource) (string, bool) {
	if inventoryResource == nil || inventoryResource.Classifications == nil {
		return "", false
	}

	details, ok := inventoryResource.Classifications.GetEnvironmentDetailsOk()
	if !ok || details == nil {
		return "", false
	}
	if !details.GetIsOverridden() {
		return "", false
	}

	env, ok := details.GetEnvironmentOk()
	if !ok || env == nil {
		return "", false
	}
	return string(*env), true
}
