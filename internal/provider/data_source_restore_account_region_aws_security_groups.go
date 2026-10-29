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
	RestoreAccountId types.String                                `tfsdk:"restore_account_id"`
	Region           types.String                                `tfsdk:"region"`
	VpcId            types.String                                `tfsdk:"vpc_id"`
	SubnetId         types.String                                `tfsdk:"subnet_id"`
	SecurityGroups   []RestoreAccountRegionAwsSecurityGroupModel `tfsdk:"security_groups"`
}

type RestoreAccountRegionAwsSecurityGroupModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Vpc  types.String `tfsdk:"vpc"`
}

func (d *RestoreAccountRegionAwsSecurityGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account_region_aws_security_groups"
}

func (d *RestoreAccountRegionAwsSecurityGroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the security groups available to a restore account in a region. Use this data source to select a security group for `eon_restore_job` network settings.",
		Attributes: map[string]schema.Attribute{
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the AWS restore account.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS region to list security groups for.",
				Required:            true,
			},
			"vpc_id": schema.StringAttribute{
				MarkdownDescription: "Optional VPC ID to filter security groups by.",
				Optional:            true,
			},
			"subnet_id": schema.StringAttribute{
				MarkdownDescription: "Optional subnet ID to filter security groups by.",
				Optional:            true,
			},
			"security_groups": schema.ListNestedAttribute{
				MarkdownDescription: "List of security groups available in the region.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "ID of the security group.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Name of the security group.",
							Computed:            true,
						},
						"vpc": schema.StringAttribute{
							MarkdownDescription: "ID of the VPC the security group belongs to.",
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

	opts := buildListAwsSecurityGroupsOptions(data)

	tflog.Debug(ctx, "Listing restore account region AWS security groups", map[string]interface{}{
		"restore_account_id": data.RestoreAccountId.ValueString(),
		"region":             data.Region.ValueString(),
	})

	securityGroups, err := d.client.ListAwsSecurityGroups(ctx, data.RestoreAccountId.ValueString(), data.Region.ValueString(), opts)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore account region AWS security groups: %s", err))
		return
	}

	groups := securityGroups.GetSecurityGroups()
	data.SecurityGroups = make([]RestoreAccountRegionAwsSecurityGroupModel, 0, len(groups))
	for _, sg := range groups {
		data.SecurityGroups = append(data.SecurityGroups, RestoreAccountRegionAwsSecurityGroupModel{
			Id:   types.StringValue(sg.GetId()),
			Name: types.StringValue(sg.GetName()),
			Vpc:  types.StringValue(sg.GetVpc()),
		})
	}

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// buildListAwsSecurityGroupsOptions translates the optional filter attributes into the client's
// query-parameter options, omitting unset values so the server returns the unfiltered list.
func buildListAwsSecurityGroupsOptions(data RestoreAccountRegionAwsSecurityGroupsDataSourceModel) client.ListAwsSecurityGroupsOptions {
	opts := client.ListAwsSecurityGroupsOptions{}

	if !data.VpcId.IsNull() && !data.VpcId.IsUnknown() && data.VpcId.ValueString() != "" {
		vpcId := data.VpcId.ValueString()
		opts.VpcId = &vpcId
	}
	if !data.SubnetId.IsNull() && !data.SubnetId.IsUnknown() && data.SubnetId.ValueString() != "" {
		subnetId := data.SubnetId.ValueString()
		opts.SubnetId = &subnetId
	}

	return opts
}
