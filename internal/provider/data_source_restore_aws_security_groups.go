package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RestoreAwsSecurityGroupsDataSource{}

func NewRestoreAwsSecurityGroupsDataSource() datasource.DataSource {
	return &RestoreAwsSecurityGroupsDataSource{}
}

type RestoreAwsSecurityGroupsDataSource struct {
	client *client.EonClient
}

type RestoreAwsSecurityGroupsDataSourceModel struct {
	RestoreAccountID types.String                   `tfsdk:"restore_account_id"`
	Region           types.String                   `tfsdk:"region"`
	VpcID            types.String                   `tfsdk:"vpc_id"`
	SubnetID         types.String                   `tfsdk:"subnet_id"`
	SecurityGroups   []RestoreAwsSecurityGroupModel `tfsdk:"security_groups"`
}

type RestoreAwsSecurityGroupModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Vpc  types.String `tfsdk:"vpc"`
}

func (d *RestoreAwsSecurityGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_aws_security_groups"
}

func (d *RestoreAwsSecurityGroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists AWS security groups available to a restore account in a region.",
		Attributes: map[string]schema.Attribute{
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon restore account ID.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS region to list security groups in.",
				Required:            true,
			},
			"vpc_id": schema.StringAttribute{
				MarkdownDescription: "Optionally filter security groups by VPC ID.",
				Optional:            true,
			},
			"subnet_id": schema.StringAttribute{
				MarkdownDescription: "Optionally filter security groups by subnet ID.",
				Optional:            true,
			},
			"security_groups": schema.ListNestedAttribute{
				MarkdownDescription: "Security groups available in the region.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "AWS security group ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "AWS security group name.",
							Computed:            true,
						},
						"vpc": schema.StringAttribute{
							MarkdownDescription: "AWS VPC ID containing the security group.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RestoreAwsSecurityGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RestoreAwsSecurityGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreAwsSecurityGroupsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var vpcID *string
	if !data.VpcID.IsNull() && !data.VpcID.IsUnknown() {
		value := data.VpcID.ValueString()
		vpcID = &value
	}
	var subnetID *string
	if !data.SubnetID.IsNull() && !data.SubnetID.IsUnknown() {
		value := data.SubnetID.ValueString()
		subnetID = &value
	}

	securityGroups, err := d.client.ListRestoreAwsSecurityGroups(
		ctx,
		data.RestoreAccountID.ValueString(),
		data.Region.ValueString(),
		vpcID,
		subnetID,
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore AWS security groups: %s", err))
		return
	}

	data.SecurityGroups = make([]RestoreAwsSecurityGroupModel, 0, len(securityGroups))
	for _, securityGroup := range securityGroups {
		data.SecurityGroups = append(data.SecurityGroups, RestoreAwsSecurityGroupModel{
			Id:   types.StringValue(securityGroup.GetId()),
			Name: types.StringValue(securityGroup.GetName()),
			Vpc:  types.StringValue(securityGroup.GetVpc()),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
