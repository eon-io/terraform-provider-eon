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

var _ resource.Resource = &ResourceBackupExclusionResource{}
var _ resource.ResourceWithImportState = &ResourceBackupExclusionResource{}

func NewResourceBackupExclusionResource() resource.Resource {
	return &ResourceBackupExclusionResource{}
}

type ResourceBackupExclusionResource struct {
	client *client.EonClient
}

type ResourceBackupExclusionResourceModel struct {
	Id         types.String `tfsdk:"id"`
	ResourceId types.String `tfsdk:"resource_id"`
}

func (r *ResourceBackupExclusionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_backup_exclusion"
}

func (r *ResourceBackupExclusionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Excludes an inventory resource from future Eon backups and suppresses scanning and violations.\n\n" +
			"When this resource is created, the inventory resource is excluded from backup. " +
			"When destroyed, the exclusion is cancelled and the resource is included in future backups.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the inventory resource. Mirrors `resource_id`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the inventory resource to exclude from backup.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *ResourceBackupExclusionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ResourceBackupExclusionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceBackupExclusionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceId := data.ResourceId.ValueString()

	tflog.Debug(ctx, "Excluding resource from backup", map[string]interface{}{
		"resource_id": resourceId,
	})

	err := r.client.ExcludeResourceFromBackup(ctx, resourceId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to exclude resource from backup: %s", err))
		return
	}

	tflog.Debug(ctx, "Resource excluded from backup", map[string]interface{}{
		"resource_id": resourceId,
	})

	data.Id = types.StringValue(resourceId)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceBackupExclusionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceBackupExclusionResourceModel

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
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read resource backup exclusion: %s", err))
		return
	}

	if inventoryResource.GetBackupStatus() != externalEonSdkAPI.EXCLUDED_FROM_BACKUP {
		tflog.Debug(ctx, "Resource is no longer excluded from backup; removing from state", map[string]interface{}{
			"resource_id":   resourceId,
			"backup_status": string(inventoryResource.GetBackupStatus()),
		})
		resp.State.RemoveResource(ctx)
		return
	}

	data.Id = types.StringValue(inventoryResource.GetId())
	data.ResourceId = types.StringValue(inventoryResource.GetId())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ResourceBackupExclusionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Not Supported", "Resource backup exclusions cannot be updated. Changing resource_id requires replacement.")
}

func (r *ResourceBackupExclusionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceBackupExclusionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceId := data.ResourceId.ValueString()

	tflog.Debug(ctx, "Cancelling resource backup exclusion", map[string]interface{}{
		"resource_id": resourceId,
	})

	err := r.client.CancelResourceBackupExclusion(ctx, resourceId)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to cancel resource backup exclusion: %s", err))
		return
	}

	tflog.Debug(ctx, "Resource backup exclusion cancelled", map[string]interface{}{
		"resource_id": resourceId,
	})
}

func (r *ResourceBackupExclusionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("resource_id"), req.ID)...)

	tflog.Info(ctx, "Successfully imported resource backup exclusion", map[string]interface{}{
		"resource_id": req.ID,
	})
}
