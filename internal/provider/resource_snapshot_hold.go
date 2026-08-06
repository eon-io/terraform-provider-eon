package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
)

var _ resource.Resource = &SnapshotHoldResource{}
var _ resource.ResourceWithImportState = &SnapshotHoldResource{}

func NewSnapshotHoldResource() resource.Resource {
	return &SnapshotHoldResource{}
}

type SnapshotHoldResource struct {
	client *client.EonClient
}

type SnapshotHoldResourceModel struct {
	Id          types.String `tfsdk:"id"`
	SnapshotId  types.String `tfsdk:"snapshot_id"`
	Description types.String `tfsdk:"description"`
}

func (r *SnapshotHoldResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot_hold"
}

func (r *SnapshotHoldResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Places a retention hold on an Eon snapshot so retention policy cannot delete it.\n\n" +
			"When this resource is created, the snapshot is held. " +
			"When destroyed, the hold is removed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the snapshot. Mirrors `snapshot_id`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"snapshot_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the snapshot to hold.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Note explaining why the snapshot is being held. Cleared when the hold is removed.",
				Optional:            true,
			},
		},
	}
}

func (r *SnapshotHoldResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SnapshotHoldResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SnapshotHoldResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.hold(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to hold snapshot: %s", err))
		return
	}

	data.Id = types.StringValue(data.SnapshotId.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SnapshotHoldResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SnapshotHoldResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshotId := data.SnapshotId.ValueString()
	snapshot, err := r.client.GetSnapshot(ctx, snapshotId)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read snapshot hold: %s", err))
		return
	}

	if !snapshot.GetOnHold() {
		tflog.Debug(ctx, "Snapshot is no longer on hold; removing from state", map[string]interface{}{
			"snapshot_id": snapshotId,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	data.Id = types.StringValue(snapshot.GetId())
	data.SnapshotId = types.StringValue(snapshot.GetId())
	if snapshot.HasHoldDescription() {
		data.Description = types.StringValue(snapshot.GetHoldDescription())
	} else {
		data.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SnapshotHoldResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SnapshotHoldResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.hold(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update snapshot hold: %s", err))
		return
	}

	data.Id = types.StringValue(data.SnapshotId.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SnapshotHoldResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SnapshotHoldResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshotId := data.SnapshotId.ValueString()

	tflog.Debug(ctx, "Removing snapshot hold", map[string]interface{}{
		"snapshot_id": snapshotId,
	})

	err := r.client.RemoveSnapshotHold(ctx, snapshotId)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to remove snapshot hold: %s", err))
		return
	}

	tflog.Debug(ctx, "Snapshot hold removed", map[string]interface{}{
		"snapshot_id": snapshotId,
	})
}

func (r *SnapshotHoldResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("snapshot_id"), req.ID)...)

	tflog.Info(ctx, "Successfully imported snapshot hold", map[string]interface{}{
		"snapshot_id": req.ID,
	})
}

func (r *SnapshotHoldResource) hold(ctx context.Context, data *SnapshotHoldResourceModel) error {
	snapshotId := data.SnapshotId.ValueString()
	req := externalEonSdkAPI.NewHoldSnapshotRequest()
	if !data.Description.IsNull() && !data.Description.IsUnknown() && data.Description.ValueString() != "" {
		req.SetDescription(data.Description.ValueString())
	}

	tflog.Debug(ctx, "Holding snapshot", map[string]interface{}{
		"snapshot_id": snapshotId,
	})

	if err := r.client.HoldSnapshot(ctx, snapshotId, *req); err != nil {
		return err
	}

	tflog.Debug(ctx, "Snapshot held", map[string]interface{}{
		"snapshot_id": snapshotId,
	})
	return nil
}
