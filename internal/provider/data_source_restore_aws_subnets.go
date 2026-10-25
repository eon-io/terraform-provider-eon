package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RestoreAwsSubnetsDataSource{}

func NewRestoreAwsSubnetsDataSource() datasource.DataSource {
	return &RestoreAwsSubnetsDataSource{}
}

type RestoreAwsSubnetsDataSource struct {
	client *client.EonClient
}

type RestoreAwsSubnetsDataSourceModel struct {
	RestoreAccountID types.String            `tfsdk:"restore_account_id"`
	Region           types.String            `tfsdk:"region"`
	Subnets          []RestoreAwsSubnetModel `tfsdk:"subnets"`
}

type RestoreAwsSubnetModel struct {
	Id  types.String `tfsdk:"id"`
	Vpc types.String `tfsdk:"vpc"`
}

func (d *RestoreAwsSubnetsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_aws_subnets"
}

func (d *RestoreAwsSubnetsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists AWS subnets and their VPCs in a restore account and region.",
		Attributes: map[string]schema.Attribute{
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon restore account ID.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS region to list subnets in.",
				Required:            true,
			},
			"subnets": schema.ListNestedAttribute{
				MarkdownDescription: "Subnets available in the region.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "AWS subnet ID.",
							Computed:            true,
						},
						"vpc": schema.StringAttribute{
							MarkdownDescription: "AWS VPC ID containing the subnet.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RestoreAwsSubnetsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RestoreAwsSubnetsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreAwsSubnetsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subnets, err := d.client.ListRestoreAwsSubnets(ctx, data.RestoreAccountID.ValueString(), data.Region.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore AWS subnets: %s", err))
		return
	}

	data.Subnets = make([]RestoreAwsSubnetModel, 0, len(subnets))
	for _, subnet := range subnets {
		data.Subnets = append(data.Subnets, RestoreAwsSubnetModel{
			Id:  types.StringValue(subnet.GetId()),
			Vpc: types.StringValue(subnet.GetVpc()),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
