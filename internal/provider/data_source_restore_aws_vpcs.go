package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RestoreAwsVpcsDataSource{}

func NewRestoreAwsVpcsDataSource() datasource.DataSource {
	return &RestoreAwsVpcsDataSource{}
}

type RestoreAwsVpcsDataSource struct {
	client *client.EonClient
}

type RestoreAwsVpcsDataSourceModel struct {
	RestoreAccountID types.String         `tfsdk:"restore_account_id"`
	Region           types.String         `tfsdk:"region"`
	Vpcs             []RestoreAwsVpcModel `tfsdk:"vpcs"`
}

type RestoreAwsVpcModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (d *RestoreAwsVpcsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_aws_vpcs"
}

func (d *RestoreAwsVpcsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists AWS VPCs available to a restore account in a region.",
		Attributes: map[string]schema.Attribute{
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon restore account ID.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS region to list VPCs in.",
				Required:            true,
			},
			"vpcs": schema.ListNestedAttribute{
				MarkdownDescription: "VPCs available in the region.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "AWS VPC ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "VPC name, when available.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RestoreAwsVpcsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RestoreAwsVpcsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreAwsVpcsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpcs, err := d.client.ListRestoreAwsVpcs(ctx, data.RestoreAccountID.ValueString(), data.Region.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore AWS VPCs: %s", err))
		return
	}

	data.Vpcs = make([]RestoreAwsVpcModel, 0, len(vpcs))
	for _, vpc := range vpcs {
		name := types.StringNull()
		if value, ok := vpc.GetNameOk(); ok {
			name = types.StringValue(*value)
		}

		data.Vpcs = append(data.Vpcs, RestoreAwsVpcModel{
			Id:   types.StringValue(vpc.GetId()),
			Name: name,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
