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

var _ datasource.DataSource = &RestoreAccountRegionAwsSecurityGroupsDataSource{}

func NewRestoreAccountRegionAwsSecurityGroupsDataSource() datasource.DataSource {
	return &RestoreAccountRegionAwsSecurityGroupsDataSource{}
}

type RestoreAccountRegionAwsSecurityGroupsDataSource struct {
	client *client.EonClient
}

type RestoreAccountRegionAwsSecurityGroupsDataSourceModel struct {
	RestoreAccountId types.String                          `tfsdk:"restore_account_id"`
	Region           types.String                          `tfsdk:"region"`
	VpcId            types.String                          `tfsdk:"vpc_id"`
	SubnetId         types.String                          `tfsdk:"subnet_id"`
	SecurityGroups   []RestoreAccountAwsSecurityGroupModel `tfsdk:"security_groups"`
}

type RestoreAccountAwsSecurityGroupModel struct {
	Id    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	VpcId types.String `tfsdk:"vpc_id"`
}

func (d *RestoreAccountRegionAwsSecurityGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account_region_aws_security_groups"
}

func (d *RestoreAccountRegionAwsSecurityGroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the security groups available to Eon in a region of an AWS restore account. Use this data source to select security group IDs for an EC2 `eon_restore_job`.",
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
				MarkdownDescription: "Optional ID of the VPC to filter security groups by.",
				Optional:            true,
			},
			"subnet_id": schema.StringAttribute{
				MarkdownDescription: "Optional ID of the subnet to filter security groups by.",
				Optional:            true,
			},
			"security_groups": schema.ListNestedAttribute{
				MarkdownDescription: "List of security groups available to Eon in the region.",
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
						"vpc_id": schema.StringAttribute{
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

	tflog.Debug(ctx, "Listing restore account AWS security groups", map[string]interface{}{
		"restore_account_id": data.RestoreAccountId.ValueString(),
		"region":             data.Region.ValueString(),
	})

	securityGroups, err := d.client.ListRestoreAccountAwsSecurityGroups(
		ctx,
		data.RestoreAccountId.ValueString(),
		data.Region.ValueString(),
		optionalStringPointer(data.VpcId),
		optionalStringPointer(data.SubnetId),
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore account AWS security groups: %s", err))
		return
	}

	data.SecurityGroups = restoreAccountAwsSecurityGroupModels(securityGroups)

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// optionalStringPointer returns nil for a null, unknown or empty value so the filter is omitted from the request.
func optionalStringPointer(value types.String) *string {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}
	s := value.ValueString()
	return &s
}

func restoreAccountAwsSecurityGroupModels(securityGroups []externalEonSdkAPI.RestoreSecurityGroupsSecurityGroupsInner) []RestoreAccountAwsSecurityGroupModel {
	models := make([]RestoreAccountAwsSecurityGroupModel, 0, len(securityGroups))
	for i := range securityGroups {
		models = append(models, RestoreAccountAwsSecurityGroupModel{
			Id:    types.StringValue(securityGroups[i].GetId()),
			Name:  types.StringValue(securityGroups[i].GetName()),
			VpcId: types.StringValue(securityGroups[i].GetVpc()),
		})
	}
	return models
}
