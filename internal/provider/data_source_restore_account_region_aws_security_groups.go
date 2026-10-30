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

var _ datasource.DataSource = &RestoreAccountRegionAwsSecurityGroupsDataSource{}

func NewRestoreAccountRegionAwsSecurityGroupsDataSource() datasource.DataSource {
	return &RestoreAccountRegionAwsSecurityGroupsDataSource{}
}

type RestoreAccountRegionAwsSecurityGroupsDataSource struct {
	client *client.EonClient
}

type RestoreAccountRegionAwsSecurityGroupsDataSourceModel struct {
	RestoreAccountId types.String             `tfsdk:"restore_account_id"`
	Region           types.String             `tfsdk:"region"`
	VpcId            types.String             `tfsdk:"vpc_id"`
	SubnetId         types.String             `tfsdk:"subnet_id"`
	SecurityGroups   []SecurityGroupListModel `tfsdk:"security_groups"`
}

type SecurityGroupListModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Vpc  types.String `tfsdk:"vpc"`
}

func (d *RestoreAccountRegionAwsSecurityGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account_region_aws_security_groups"
}

func (d *RestoreAccountRegionAwsSecurityGroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the security groups available to Eon in a region of a restore account. Use this data source to select `security_group_ids` for `eon_restore_job` network settings.",
		Attributes: map[string]schema.Attribute{
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the restore account.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Region where the security groups are located.",
				Required:            true,
			},
			"vpc_id": schema.StringAttribute{
				MarkdownDescription: "ID of the VPC to filter security groups by.",
				Optional:            true,
			},
			"subnet_id": schema.StringAttribute{
				MarkdownDescription: "ID of the subnet to filter security groups by.",
				Optional:            true,
			},
			"security_groups": schema.ListNestedAttribute{
				MarkdownDescription: "List of security groups available in the region.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Cloud-provider-assigned ID of the security group.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the security group.",
							Computed:            true,
						},
						"vpc": schema.StringAttribute{
							MarkdownDescription: "Cloud-provider-assigned ID of the VPC the security group is in.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RestoreAccountRegionAwsSecurityGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RestoreAccountRegionAwsSecurityGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreAccountRegionAwsSecurityGroupsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var vpcId *string
	if !data.VpcId.IsNull() && !data.VpcId.IsUnknown() {
		v := data.VpcId.ValueString()
		vpcId = &v
	}

	var subnetId *string
	if !data.SubnetId.IsNull() && !data.SubnetId.IsUnknown() {
		v := data.SubnetId.ValueString()
		subnetId = &v
	}

	tflog.Debug(ctx, "Listing restore account AWS security groups", map[string]interface{}{
		"restore_account_id": data.RestoreAccountId.ValueString(),
		"region":             data.Region.ValueString(),
	})

	result, err := d.client.ListAwsSecurityGroups(ctx, data.RestoreAccountId.ValueString(), data.Region.ValueString(), vpcId, subnetId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list AWS security groups: %s", err))
		return
	}

	data.SecurityGroups = make([]SecurityGroupListModel, 0, len(result.GetSecurityGroups()))
	for _, sg := range result.GetSecurityGroups() {
		data.SecurityGroups = append(data.SecurityGroups, SecurityGroupListModel{
			Id:   types.StringValue(sg.GetId()),
			Name: types.StringValue(sg.GetName()),
			Vpc:  types.StringValue(sg.GetVpc()),
		})
	}

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
