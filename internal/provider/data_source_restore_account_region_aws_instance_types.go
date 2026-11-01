package provider

import (
	"context"
	"fmt"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &RestoreAccountRegionAwsInstanceTypesDataSource{}

func NewRestoreAccountRegionAwsInstanceTypesDataSource() datasource.DataSource {
	return &RestoreAccountRegionAwsInstanceTypesDataSource{}
}

type RestoreAccountRegionAwsInstanceTypesDataSource struct {
	client *client.EonClient
}

type RestoreAccountRegionAwsInstanceTypesDataSourceModel struct {
	RestoreAccountId      types.String `tfsdk:"restore_account_id"`
	Region                types.String `tfsdk:"region"`
	SubnetId              types.String `tfsdk:"subnet_id"`
	VolumeCount           types.Int64  `tfsdk:"volume_count"`
	TotalVolumeBytes      types.Int64  `tfsdk:"total_volume_bytes"`
	InstanceTypes         types.List   `tfsdk:"instance_types"`
	InstantRestoreMapping types.Map    `tfsdk:"instant_restore_mapping"`
}

func (d *RestoreAccountRegionAwsInstanceTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account_region_aws_instance_types"
}

func (d *RestoreAccountRegionAwsInstanceTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the EC2 instance types available for restore in a subnet of an AWS restore account region. Use this data source to select an instance type for an EC2 `eon_restore_job`.",
		Attributes: map[string]schema.Attribute{
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the restore account.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Region where the subnet is located.",
				Required:            true,
			},
			"subnet_id": schema.StringAttribute{
				MarkdownDescription: "ID of the subnet whose instance types are listed.",
				Required:            true,
			},
			"volume_count": schema.Int64Attribute{
				MarkdownDescription: "Optional number of guest volumes you plan to restore. Used to size the host-memory reserve for instant-restore host selection. If omitted, the server uses a default.",
				Optional:            true,
			},
			"total_volume_bytes": schema.Int64Attribute{
				MarkdownDescription: "Optional total size of the guest volumes you plan to restore, in bytes. Used to size the host-memory reserve. If omitted, the server uses a default.",
				Optional:            true,
			},
			"instance_types": schema.ListAttribute{
				MarkdownDescription: "List of EC2 instance types available for restore.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"instant_restore_mapping": schema.MapAttribute{
				MarkdownDescription: "Map from post-promotion instance type to instant-restore host instance type. Instance types in `instance_types` that are absent from this map are not available for instant restore in the availability zone.",
				ElementType:         types.StringType,
				Computed:            true,
			},
		},
	}
}

func (d *RestoreAccountRegionAwsInstanceTypesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.EonClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *RestoreAccountRegionAwsInstanceTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreAccountRegionAwsInstanceTypesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts, err := buildListAwsInstanceTypesOptions(data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Configuration", err.Error())
		return
	}

	tflog.Debug(ctx, "Listing restore account AWS instance types", map[string]interface{}{
		"restore_account_id": data.RestoreAccountId.ValueString(),
		"region":             data.Region.ValueString(),
		"subnet_id":          data.SubnetId.ValueString(),
	})

	result, err := d.client.ListRestoreAccountAwsInstanceTypes(
		ctx,
		data.RestoreAccountId.ValueString(),
		data.Region.ValueString(),
		data.SubnetId.ValueString(),
		opts,
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore account AWS instance types: %s", err))
		return
	}

	resp.Diagnostics.Append(setInstanceTypesFromResponse(ctx, &data, result)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func buildListAwsInstanceTypesOptions(data RestoreAccountRegionAwsInstanceTypesDataSourceModel) (client.ListAwsInstanceTypesOptions, error) {
	var opts client.ListAwsInstanceTypesOptions
	if !data.VolumeCount.IsNull() && !data.VolumeCount.IsUnknown() {
		volumeCount, err := SafeInt32Conversion(data.VolumeCount.ValueInt64())
		if err != nil {
			return opts, fmt.Errorf("invalid volume_count: %w", err)
		}
		opts.VolumeCount = &volumeCount
	}
	if !data.TotalVolumeBytes.IsNull() && !data.TotalVolumeBytes.IsUnknown() {
		totalVolumeBytes := data.TotalVolumeBytes.ValueInt64()
		opts.TotalVolumeBytes = &totalVolumeBytes
	}
	return opts, nil
}

func setInstanceTypesFromResponse(ctx context.Context, data *RestoreAccountRegionAwsInstanceTypesDataSourceModel, result *externalEonSdkAPI.ListInstanceTypesResponse) diag.Diagnostics {
	var diags diag.Diagnostics

	instanceTypes := result.GetInstanceTypes()
	if instanceTypes == nil {
		instanceTypes = []string{}
	}
	listValue, d := types.ListValueFrom(ctx, types.StringType, instanceTypes)
	diags.Append(d...)
	data.InstanceTypes = listValue

	mapping := result.GetInstantRestoreMapping()
	if mapping == nil {
		mapping = map[string]string{}
	}
	mapValue, d := types.MapValueFrom(ctx, types.StringType, mapping)
	diags.Append(d...)
	data.InstantRestoreMapping = mapValue

	return diags
}
