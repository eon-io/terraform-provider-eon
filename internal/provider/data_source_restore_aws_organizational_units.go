package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RestoreAwsOrganizationalUnitsDataSource{}

func NewRestoreAwsOrganizationalUnitsDataSource() datasource.DataSource {
	return &RestoreAwsOrganizationalUnitsDataSource{}
}

type RestoreAwsOrganizationalUnitsDataSource struct {
	client *client.EonClient
}

type RestoreAwsOrganizationalUnitsDataSourceModel struct {
	OrganizationalUnits []RestoreAwsOrganizationalUnitModel `tfsdk:"organizational_units"`
}

type RestoreAwsOrganizationalUnitModel struct {
	Id                           types.String `tfsdk:"id"`
	Name                         types.String `tfsdk:"name"`
	RoleArn                      types.String `tfsdk:"role_arn"`
	ProviderOrganizationalUnitId types.String `tfsdk:"provider_organizational_unit_id"`
	ProviderManagementAccountId  types.String `tfsdk:"provider_management_account_id"`
	Status                       types.String `tfsdk:"status"`
}

func (d *RestoreAwsOrganizationalUnitsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_aws_organizational_units"
}

func (d *RestoreAwsOrganizationalUnitsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of restore AWS organizational units for the Eon project.",
		Attributes: map[string]schema.Attribute{
			"organizational_units": schema.ListNestedAttribute{
				MarkdownDescription: "List of restore AWS organizational units.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Eon-assigned organizational unit ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The organizational unit display name in Eon.",
							Computed:            true,
						},
						"role_arn": schema.StringAttribute{
							MarkdownDescription: "ARN of the role Eon assumes to access the organizational unit in AWS.",
							Computed:            true,
						},
						"provider_organizational_unit_id": schema.StringAttribute{
							MarkdownDescription: "AWS Organizational Unit ID.",
							Computed:            true,
						},
						"provider_management_account_id": schema.StringAttribute{
							MarkdownDescription: "AWS Organization management account ID.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "Connection status of the AWS organizational unit. Possible values: `CONNECTED`, `DISCONNECTED`, `INSUFFICIENT_PERMISSIONS`.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RestoreAwsOrganizationalUnitsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.EonClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *RestoreAwsOrganizationalUnitsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreAwsOrganizationalUnitsDataSourceModel

	ous, err := d.client.ListRestoreAwsOrganizationalUnits(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore AWS organizational units: %s", err))
		return
	}

	for _, ou := range ous {
		ouModel := RestoreAwsOrganizationalUnitModel{
			Id:                           types.StringValue(ou.GetId()),
			Name:                         types.StringValue(ou.GetName()),
			RoleArn:                      types.StringValue(ou.GetRoleArn()),
			ProviderOrganizationalUnitId: types.StringValue(ou.GetProviderOrganizationalUnitId()),
			ProviderManagementAccountId:  types.StringValue(ou.GetProviderManagementAccountId()),
			Status:                       types.StringValue(string(ou.GetStatus())),
		}

		data.OrganizationalUnits = append(data.OrganizationalUnits, ouModel)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
