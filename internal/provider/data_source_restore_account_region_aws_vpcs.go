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

var _ datasource.DataSource = &RestoreAccountRegionAwsVpcsDataSource{}

func NewRestoreAccountRegionAwsVpcsDataSource() datasource.DataSource {
	return &RestoreAccountRegionAwsVpcsDataSource{}
}

type RestoreAccountRegionAwsVpcsDataSource struct {
	client *client.EonClient
}

type RestoreAccountRegionAwsVpcsDataSourceModel struct {
	AccountId types.String                      `tfsdk:"account_id"`
	Region    types.String                      `tfsdk:"region"`
	Vpcs      []RestoreAccountRegionAwsVpcModel `tfsdk:"vpcs"`
}

type RestoreAccountRegionAwsVpcModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (d *RestoreAccountRegionAwsVpcsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account_region_aws_vpcs"
}

func (d *RestoreAccountRegionAwsVpcsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the VPCs available to Eon in a region of a restore account. Use this data source to pick a VPC before choosing subnets and security groups for `eon_restore_job`, for example as the `vpc_id` filter of `eon_restore_account_region_aws_security_groups`.",
		Attributes: map[string]schema.Attribute{
			"account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the restore account.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Region where the VPCs are located.",
				Required:            true,
			},
			"vpcs": schema.ListNestedAttribute{
				MarkdownDescription: "List of VPCs available in the region.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Cloud-provider-assigned ID of the VPC.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the VPC. Null if the VPC has no name.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RestoreAccountRegionAwsVpcsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RestoreAccountRegionAwsVpcsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreAccountRegionAwsVpcsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Listing restore account region AWS VPCs", map[string]interface{}{
		"account_id": data.AccountId.ValueString(),
		"region":     data.Region.ValueString(),
	})

	result, err := d.client.ListAwsVpcs(ctx, data.AccountId.ValueString(), data.Region.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read AWS VPCs: %s", err))
		return
	}

	data.Vpcs = restoreAccountRegionAwsVpcsToModel(result.GetVpcs())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func restoreAccountRegionAwsVpcsToModel(vpcs []externalEonSdkAPI.ListAwsVpcsResponseVpcsInner) []RestoreAccountRegionAwsVpcModel {
	models := make([]RestoreAccountRegionAwsVpcModel, 0, len(vpcs))
	for _, vpc := range vpcs {
		models = append(models, RestoreAccountRegionAwsVpcModel{
			Id:   types.StringValue(vpc.GetId()),
			Name: types.StringPointerValue(vpc.Name),
		})
	}
	return models
}
