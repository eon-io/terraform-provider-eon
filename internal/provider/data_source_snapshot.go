package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &SnapshotDataSource{}

func NewSnapshotDataSource() datasource.DataSource {
	return &SnapshotDataSource{}
}

// SnapshotDataSource defines the data source implementation.
type SnapshotDataSource struct {
	client *client.EonClient
}

// SnapshotDataSourceModel describes the data source data model.
type SnapshotDataSourceModel struct {
	Id             types.String `tfsdk:"id"`
	ProjectId      types.String `tfsdk:"project_id"`
	ResourceId     types.String `tfsdk:"resource_id"`
	VaultId        types.String `tfsdk:"vault_id"`
	CreatedAt      types.String `tfsdk:"created_at"`
	ExpirationDate types.String `tfsdk:"expiration_date"`
	PointInTime    types.String `tfsdk:"point_in_time"`
}

func (d *SnapshotDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot"
}

func (d *SnapshotDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Eon snapshot data source",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Snapshot identifier",
				Required:            true,
			},
			"project_id": schema.StringAttribute{
				MarkdownDescription: "ID of the project that the snapshot belongs to",
				Computed:            true,
			},
			"vault_id": schema.StringAttribute{
				MarkdownDescription: "ID of the vault where the snapshot is stored",
				Computed:            true,
			},
			"resource_id": schema.StringAttribute{
				MarkdownDescription: "ID of the resource that was snapshotted",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Snapshot creation timestamp",
				Computed:            true,
			},
			"expiration_date": schema.StringAttribute{
				MarkdownDescription: "Snapshot expiration timestamp",
				Computed:            true,
			},
			"point_in_time": schema.StringAttribute{
				MarkdownDescription: "The point in time the resource was backed up from",
				Computed:            true,
			},
		},
	}
}

func (d *SnapshotDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SnapshotDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SnapshotDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get snapshot from API
	tflog.Debug(ctx, "Getting snapshot", map[string]interface{}{
		"snapshot_id": data.Id.ValueString(),
	})

	snapshot, err := d.client.GetSnapshot(ctx, data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read snapshot, got error: %s", err))
		return
	}

	data.Id = types.StringValue(snapshot.Id)
	data.ResourceId = types.StringValue(snapshot.ResourceId)
	data.CreatedAt = types.StringValue(snapshot.GetCreatedTime().String())
	data.VaultId = types.StringValue(snapshot.VaultId)
	data.ExpirationDate = types.StringValue(snapshot.GetExpirationTime().String())
	data.PointInTime = types.StringValue(snapshot.GetPointInTime().String())
	if snapshot.ProjectId != nil {
		data.ProjectId = types.StringValue(*snapshot.ProjectId)
	}

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
