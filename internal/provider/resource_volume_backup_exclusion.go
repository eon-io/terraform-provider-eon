package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &VolumeBackupExclusionResource{}
var _ resource.ResourceWithImportState = &VolumeBackupExclusionResource{}

func NewVolumeBackupExclusionResource() resource.Resource {
	return &VolumeBackupExclusionResource{}
}

type VolumeBackupExclusionResource struct {
	client *client.EonClient
}

type VolumeBackupExclusionResourceModel struct {
	ResourceId types.String `tfsdk:"resource_id"`
	VolumeId   types.String `tfsdk:"volume_id"`
}

func (r *VolumeBackupExclusionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_volume_backup_exclusion"
}

func (r *VolumeBackupExclusionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Excludes a specific EBS volume from future EC2 instance backups. " +
			"Root volumes cannot be excluded.\n\n" +
			"When this resource is created, the volume is excluded from backup. " +
			"When destroyed, the volume is included back in future backups.",
		Attributes: map[string]schema.Attribute{
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the EC2 instance resource.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"volume_id": schema.StringAttribute{
				MarkdownDescription: "AWS EBS volume ID (`vol-…`) of the volume to exclude from backup. The volume must be attached to the EC2 instance and must not be the root volume.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *VolumeBackupExclusionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *VolumeBackupExclusionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VolumeBackupExclusionResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceId := data.ResourceId.ValueString()
	volumeId := data.VolumeId.ValueString()

	tflog.Debug(ctx, "Excluding volume from backup", map[string]interface{}{
		"resource_id": resourceId,
		"volume_id":   volumeId,
	})

	err := r.client.ExcludeVolumeFromBackup(ctx, resourceId, volumeId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to exclude volume from backup: %s", err))
		return
	}

	tflog.Debug(ctx, "Volume excluded from backup", map[string]interface{}{
		"resource_id": resourceId,
		"volume_id":   volumeId,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeBackupExclusionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VolumeBackupExclusionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VolumeBackupExclusionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Not Supported", "Volume backup exclusions cannot be updated. Changing resource_id or volume_id requires replacement.")
}

func (r *VolumeBackupExclusionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VolumeBackupExclusionResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resourceId := data.ResourceId.ValueString()
	volumeId := data.VolumeId.ValueString()

	tflog.Debug(ctx, "Cancelling volume backup exclusion", map[string]interface{}{
		"resource_id": resourceId,
		"volume_id":   volumeId,
	})

	err := r.client.CancelVolumeBackupExclusion(ctx, resourceId, volumeId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to cancel volume backup exclusion: %s", err))
		return
	}

	tflog.Debug(ctx, "Volume backup exclusion cancelled", map[string]interface{}{
		"resource_id": resourceId,
		"volume_id":   volumeId,
	})
}

func (r *VolumeBackupExclusionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in the format 'resource_id/volume_id', got: %s", req.ID),
		)
		return
	}

	data := VolumeBackupExclusionResourceModel{
		ResourceId: types.StringValue(parts[0]),
		VolumeId:   types.StringValue(parts[1]),
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Info(ctx, "Successfully imported volume backup exclusion", map[string]interface{}{
		"resource_id": parts[0],
		"volume_id":   parts[1],
	})
}
