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

var _ datasource.DataSource = &RestoreAccountRegionAwsSubnetsDataSource{}

func NewRestoreAccountRegionAwsSubnetsDataSource() datasource.DataSource {
	return &RestoreAccountRegionAwsSubnetsDataSource{}
}

type RestoreAccountRegionAwsSubnetsDataSource struct {
	client *client.EonClient
}

type RestoreAccountRegionAwsSubnetsDataSourceModel struct {
	AccountId types.String                         `tfsdk:"account_id"`
	Region    types.String                         `tfsdk:"region"`
	Subnets   []RestoreAccountRegionAwsSubnetModel `tfsdk:"subnets"`
}

type RestoreAccountRegionAwsSubnetModel struct {
	Id               types.String `tfsdk:"id"`
	Vpc              types.String `tfsdk:"vpc"`
	AvailabilityZone types.String `tfsdk:"availability_zone"`
	Name             types.String `tfsdk:"name"`
	CidrBlock        types.String `tfsdk:"cidr_block"`
}

func (d *RestoreAccountRegionAwsSubnetsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account_region_aws_subnets"
}

func (d *RestoreAccountRegionAwsSubnetsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the subnets available to Eon in a region of a restore account. Use this data source to select values for `eon_restore_job` network configuration, such as `subnet_id`.",
		Attributes: map[string]schema.Attribute{
			"account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the restore account.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Region where the subnets are located.",
				Required:            true,
			},
			"subnets": schema.ListNestedAttribute{
				MarkdownDescription: "List of subnets available in the region.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Cloud-provider-assigned ID of the subnet.",
							Computed:            true,
						},
						"vpc": schema.StringAttribute{
							MarkdownDescription: "Cloud-provider-assigned ID of the VPC the subnet is in.",
							Computed:            true,
						},
						"availability_zone": schema.StringAttribute{
							MarkdownDescription: "Availability zone the subnet is in.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the subnet. Null if the subnet has no name.",
							Computed:            true,
						},
						"cidr_block": schema.StringAttribute{
							MarkdownDescription: "Primary IPv4 CIDR range of the subnet.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RestoreAccountRegionAwsSubnetsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RestoreAccountRegionAwsSubnetsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreAccountRegionAwsSubnetsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Listing restore account region AWS subnets", map[string]interface{}{
		"account_id": data.AccountId.ValueString(),
		"region":     data.Region.ValueString(),
	})

	result, err := d.client.ListAwsSubnets(ctx, data.AccountId.ValueString(), data.Region.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read AWS subnets: %s", err))
		return
	}

	data.Subnets = restoreAccountRegionAwsSubnetsToModel(result.GetSubnets())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func restoreAccountRegionAwsSubnetsToModel(subnets []externalEonSdkAPI.ListSubnetsResponseV2SubnetsInner) []RestoreAccountRegionAwsSubnetModel {
	models := make([]RestoreAccountRegionAwsSubnetModel, 0, len(subnets))
	for _, subnet := range subnets {
		models = append(models, RestoreAccountRegionAwsSubnetModel{
			Id:               types.StringValue(subnet.GetId()),
			Vpc:              types.StringValue(subnet.GetVpc()),
			AvailabilityZone: types.StringValue(subnet.GetAvailabilityZone()),
			Name:             types.StringPointerValue(subnet.Name),
			CidrBlock:        types.StringPointerValue(subnet.CidrBlock),
		})
	}
	return models
}
