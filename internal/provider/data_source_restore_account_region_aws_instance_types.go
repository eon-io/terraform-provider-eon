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
		MarkdownDescription: "Retrieves the EC2 instance types available for restore in a restore account's region and subnet. Use this data source to select an instance type for `eon_restore_job`.",
		Attributes: map[string]schema.Attribute{
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the AWS restore account.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS region to list instance types for.",
				Required:            true,
			},
			"subnet_id": schema.StringAttribute{
				MarkdownDescription: "ID of the subnet to get instance types for.",
				Required:            true,
			},
			"volume_count": schema.Int64Attribute{
				MarkdownDescription: "Number of guest volumes planned to be restored. Used to size the host-memory reserve for instant-restore host selection. The server uses a default when omitted.",
				Optional:            true,
			},
			"total_volume_bytes": schema.Int64Attribute{
				MarkdownDescription: "Sum of guest-volume sizes in bytes. Used to size the host-memory reserve. The server uses a default when omitted.",
				Optional:            true,
			},
			"instance_types": schema.ListAttribute{
				MarkdownDescription: "List of EC2 instance types available for restore.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"instant_restore_mapping": schema.MapAttribute{
				MarkdownDescription: "Map from post-promotion instance type to instant-restore host instance type. Instance types present in `instance_types` but absent from this map have no instant-restore option in this availability zone.",
				Computed:            true,
				ElementType:         types.StringType,
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

	opts, diags := buildListAwsInstanceTypesOptions(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Listing restore account region AWS instance types", map[string]interface{}{
		"restore_account_id": data.RestoreAccountId.ValueString(),
		"region":             data.Region.ValueString(),
		"subnet_id":          data.SubnetId.ValueString(),
	})

	instanceTypes, err := d.client.ListAwsInstanceTypes(ctx, data.RestoreAccountId.ValueString(), data.Region.ValueString(), data.SubnetId.ValueString(), opts)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore account region AWS instance types: %s", err))
		return
	}

	resp.Diagnostics.Append(instanceTypesToModel(ctx, instanceTypes, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// buildListAwsInstanceTypesOptions translates the optional config attributes into the client's
// query-parameter options, leaving unset values nil so the server applies its own defaults.
func buildListAwsInstanceTypesOptions(data RestoreAccountRegionAwsInstanceTypesDataSourceModel) (client.ListAwsInstanceTypesOptions, diag.Diagnostics) {
	var diags diag.Diagnostics
	opts := client.ListAwsInstanceTypesOptions{}

	if !data.VolumeCount.IsNull() && !data.VolumeCount.IsUnknown() {
		volumeCount, err := SafeInt32Conversion(data.VolumeCount.ValueInt64())
		if err != nil {
			diags.AddError("Invalid volume_count", err.Error())
			return opts, diags
		}
		opts.VolumeCount = &volumeCount
	}

	if !data.TotalVolumeBytes.IsNull() && !data.TotalVolumeBytes.IsUnknown() {
		totalVolumeBytes := data.TotalVolumeBytes.ValueInt64()
		opts.TotalVolumeBytes = &totalVolumeBytes
	}

	return opts, diags
}

func instanceTypesToModel(ctx context.Context, apiResp *externalEonSdkAPI.ListInstanceTypesResponse, data *RestoreAccountRegionAwsInstanceTypesDataSourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	instanceTypes := apiResp.GetInstanceTypes()
	if instanceTypes == nil {
		instanceTypes = []string{}
	}
	instanceTypesList, d := types.ListValueFrom(ctx, types.StringType, instanceTypes)
	diags.Append(d...)
	data.InstanceTypes = instanceTypesList

	mapping := apiResp.GetInstantRestoreMapping()
	if mapping == nil {
		mapping = map[string]string{}
	}
	mappingVal, d := types.MapValueFrom(ctx, types.StringType, mapping)
	diags.Append(d...)
	data.InstantRestoreMapping = mappingVal

	return diags
}
