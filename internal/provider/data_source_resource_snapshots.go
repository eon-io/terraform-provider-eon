package provider

import (
	"context"
	"fmt"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &ResourceSnapshotsDataSource{}

func NewResourceSnapshotsDataSource() datasource.DataSource {
	return &ResourceSnapshotsDataSource{}
}

type ResourceSnapshotsDataSource struct {
	client *client.EonClient
}

type ResourceSnapshotsDataSourceModel struct {
	ResourceId           types.String                `tfsdk:"resource_id"`
	PointInTimeStartDate types.String                `tfsdk:"point_in_time_start_date"`
	PointInTimeEndDate   types.String                `tfsdk:"point_in_time_end_date"`
	Snapshots            []ResourceSnapshotListModel `tfsdk:"snapshots"`
}

type ResourceSnapshotListModel struct {
	Id             types.String `tfsdk:"id"`
	ProjectId      types.String `tfsdk:"project_id"`
	ResourceId     types.String `tfsdk:"resource_id"`
	VaultId        types.String `tfsdk:"vault_id"`
	CreatedAt      types.String `tfsdk:"created_at"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
	PointInTime    types.String `tfsdk:"point_in_time"`
	OnHold         types.Bool   `tfsdk:"on_hold"`
}

func (d *ResourceSnapshotsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_snapshots"
}

func (d *ResourceSnapshotsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the list of restorable snapshots for an inventory resource. Use this data source to select a snapshot ID for `eon_restore_job`.",
		Attributes: map[string]schema.Attribute{
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the inventory resource whose snapshots are listed.",
				Required:            true,
			},
			"point_in_time_start_date": schema.StringAttribute{
				MarkdownDescription: "Optional lower bound for snapshot `point_in_time`, in ISO 8601 `YYYY-MM-DD` format.",
				Optional:            true,
			},
			"point_in_time_end_date": schema.StringAttribute{
				MarkdownDescription: "Optional upper bound for snapshot `point_in_time`, in ISO 8601 `YYYY-MM-DD` format.",
				Optional:            true,
			},
			"snapshots": schema.ListNestedAttribute{
				MarkdownDescription: "List of snapshots for the resource.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Eon snapshot ID.",
							Computed:            true,
						},
						"project_id": schema.StringAttribute{
							MarkdownDescription: "ID of the snapshot's parent project.",
							Computed:            true,
						},
						"vault_id": schema.StringAttribute{
							MarkdownDescription: "ID of the vault the snapshot is stored in.",
							Computed:            true,
						},
						"resource_id": schema.StringAttribute{
							MarkdownDescription: "Eon-assigned ID of the resource the snapshot is backing up.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "Date and time the snapshot creation was started.",
							Computed:            true,
						},
						"expiration_date": schema.StringAttribute{
							MarkdownDescription: "Date and time the snapshot's retention is expected to expire.",
							Computed:            true,
						},
						"point_in_time": schema.StringAttribute{
							MarkdownDescription: "Date and time of the resource that's preserved by the snapshot.",
							Computed:            true,
						},
						"on_hold": schema.BoolAttribute{
							MarkdownDescription: "Whether the snapshot is on user hold.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *ResourceSnapshotsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	eonClient, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.EonClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = eonClient
}

func (d *ResourceSnapshotsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ResourceSnapshotsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filters := buildListResourceSnapshotsFilters(data)

	tflog.Debug(ctx, "Listing resource snapshots", map[string]interface{}{
		"resource_id": data.ResourceId.ValueString(),
	})

	snapshots, err := d.client.ListResourceSnapshots(ctx, data.ResourceId.ValueString(), filters)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read resource snapshots: %s", err))
		return
	}

	data.Snapshots = make([]ResourceSnapshotListModel, 0, len(snapshots))
	for _, snapshot := range snapshots {
		model := ResourceSnapshotListModel{
			Id:         types.StringValue(snapshot.GetId()),
			ResourceId: types.StringValue(snapshot.GetResourceId()),
			CreatedAt:  types.StringValue(snapshot.GetCreatedTime().String()),
			OnHold:     types.BoolNull(),
		}

		if snapshot.ProjectId != nil {
			model.ProjectId = types.StringValue(*snapshot.ProjectId)
		} else {
			model.ProjectId = types.StringNull()
		}
		if snapshot.VaultId != nil {
			model.VaultId = types.StringValue(*snapshot.VaultId)
		} else {
			model.VaultId = types.StringNull()
		}
		if snapshot.ExpirationTime != nil {
			model.ExpirationDate = types.StringValue(snapshot.GetExpirationTime().String())
		} else {
			model.ExpirationDate = types.StringNull()
		}
		if snapshot.PointInTime != nil {
			model.PointInTime = types.StringValue(snapshot.GetPointInTime().String())
		} else {
			model.PointInTime = types.StringNull()
		}
		if snapshot.OnHold != nil {
			model.OnHold = types.BoolValue(*snapshot.OnHold)
		}

		data.Snapshots = append(data.Snapshots, model)
	}

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func buildListResourceSnapshotsFilters(data ResourceSnapshotsDataSourceModel) *externalEonSdkAPI.SnapshotFilterConditions {
	hasStart := !data.PointInTimeStartDate.IsNull() && !data.PointInTimeStartDate.IsUnknown() && data.PointInTimeStartDate.ValueString() != ""
	hasEnd := !data.PointInTimeEndDate.IsNull() && !data.PointInTimeEndDate.IsUnknown() && data.PointInTimeEndDate.ValueString() != ""
	if !hasStart && !hasEnd {
		return nil
	}

	dateFilters := externalEonSdkAPI.NewSnapshotDateFilters()
	if hasStart {
		dateFilters.SetStartDate(data.PointInTimeStartDate.ValueString())
	}
	if hasEnd {
		dateFilters.SetEndDate(data.PointInTimeEndDate.ValueString())
	}

	filters := externalEonSdkAPI.NewSnapshotFilterConditions()
	filters.SetPointInTime(*dateFilters)
	return filters
}
