package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RestoreAwsInstanceTypesDataSource{}

func NewRestoreAwsInstanceTypesDataSource() datasource.DataSource {
	return &RestoreAwsInstanceTypesDataSource{}
}

type RestoreAwsInstanceTypesDataSource struct {
	client *client.EonClient
}

type RestoreAwsInstanceTypesDataSourceModel struct {
	RestoreAccountID types.String `tfsdk:"restore_account_id"`
	Region           types.String `tfsdk:"region"`
	SubnetID         types.String `tfsdk:"subnet_id"`
	VolumeCount      types.Int32  `tfsdk:"volume_count"`
	TotalVolumeBytes types.Int64  `tfsdk:"total_volume_bytes"`
	InstanceTypes    types.List   `tfsdk:"instance_types"`
}

func (d *RestoreAwsInstanceTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_aws_instance_types"
}

func (d *RestoreAwsInstanceTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists AWS instance types available to restore into a subnet.",
		Attributes: map[string]schema.Attribute{
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon restore account ID.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS region to list instance types in.",
				Required:            true,
			},
			"subnet_id": schema.StringAttribute{
				MarkdownDescription: "AWS subnet ID to use for the restore.",
				Required:            true,
			},
			"volume_count": schema.Int32Attribute{
				MarkdownDescription: "Number of guest volumes to restore.",
				Optional:            true,
			},
			"total_volume_bytes": schema.Int64Attribute{
				MarkdownDescription: "Total size of guest volumes to restore, in bytes.",
				Optional:            true,
			},
			"instance_types": schema.ListAttribute{
				MarkdownDescription: "Instance types available for the subnet.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *RestoreAwsInstanceTypesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RestoreAwsInstanceTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreAwsInstanceTypesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var volumeCount *int32
	if !data.VolumeCount.IsNull() && !data.VolumeCount.IsUnknown() {
		value := data.VolumeCount.ValueInt32()
		volumeCount = &value
	}
	var totalVolumeBytes *int64
	if !data.TotalVolumeBytes.IsNull() && !data.TotalVolumeBytes.IsUnknown() {
		value := data.TotalVolumeBytes.ValueInt64()
		totalVolumeBytes = &value
	}

	instanceTypes, err := d.client.ListRestoreAwsInstanceTypes(
		ctx,
		data.RestoreAccountID.ValueString(),
		data.Region.ValueString(),
		data.SubnetID.ValueString(),
		volumeCount,
		totalVolumeBytes,
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore AWS instance types: %s", err))
		return
	}

	list, diags := types.ListValueFrom(ctx, types.StringType, instanceTypes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.InstanceTypes = list
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
