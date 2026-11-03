package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	AccountId             types.String      `tfsdk:"account_id"`
	Region                types.String      `tfsdk:"region"`
	SubnetId              types.String      `tfsdk:"subnet_id"`
	VolumeCount           types.Int64       `tfsdk:"volume_count"`
	TotalVolumeBytes      types.Int64       `tfsdk:"total_volume_bytes"`
	InstanceTypes         []string          `tfsdk:"instance_types"`
	InstantRestoreMapping map[string]string `tfsdk:"instant_restore_mapping"`
}

func (d *RestoreAccountRegionAwsInstanceTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account_region_aws_instance_types"
}

func (d *RestoreAccountRegionAwsInstanceTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the EC2 instance types available for restore in a subnet and region of a restore account. Use this data source to select a value for `eon_restore_job` `ec2_config.instance_type`.",
		Attributes: map[string]schema.Attribute{
			"account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the restore account.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Region where the subnet is located.",
				Required:            true,
			},
			"subnet_id": schema.StringAttribute{
				MarkdownDescription: "ID of the subnet whose instance types to list.",
				Required:            true,
			},
			"volume_count": schema.Int64Attribute{
				MarkdownDescription: "Number of guest volumes you plan to restore. Used to size the host-memory reserve for instant-restore host selection. If omitted, the server uses a default.",
				Optional:            true,
			},
			"total_volume_bytes": schema.Int64Attribute{
				MarkdownDescription: "Total size of the guest volumes you plan to restore, in bytes. Used to size the host-memory reserve. If omitted, the server uses a default.",
				Optional:            true,
			},
			"instance_types": schema.ListAttribute{
				MarkdownDescription: "List of EC2 instance types available for restore.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"instant_restore_mapping": schema.MapAttribute{
				MarkdownDescription: "Map from post-promotion to instant-restore host instance type. Options that are present in `instance_types` but absent from this map are not available for instant restore in the availability zone.",
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

	opts := client.ListAwsInstanceTypesOptions{}
	if !data.VolumeCount.IsNull() && !data.VolumeCount.IsUnknown() {
		volumeCount, err := SafeInt32Conversion(data.VolumeCount.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("volume_count"),
				"Invalid Attribute Value",
				fmt.Sprintf("volume_count could not be converted: %s", err),
			)
			return
		}
		opts.VolumeCount = &volumeCount
	}
	if !data.TotalVolumeBytes.IsNull() && !data.TotalVolumeBytes.IsUnknown() {
		totalVolumeBytes := data.TotalVolumeBytes.ValueInt64()
		opts.TotalVolumeBytes = &totalVolumeBytes
	}

	tflog.Debug(ctx, "Listing restore account region AWS instance types", map[string]interface{}{
		"account_id": data.AccountId.ValueString(),
		"region":     data.Region.ValueString(),
		"subnet_id":  data.SubnetId.ValueString(),
	})

	result, err := d.client.ListAwsInstanceTypes(ctx, data.AccountId.ValueString(), data.Region.ValueString(), data.SubnetId.ValueString(), opts)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read AWS instance types: %s", err))
		return
	}

	data.InstanceTypes = append([]string{}, result.GetInstanceTypes()...)
	mapping := result.GetInstantRestoreMapping()
	data.InstantRestoreMapping = make(map[string]string, len(mapping))
	for k, v := range mapping {
		data.InstantRestoreMapping[k] = v
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
