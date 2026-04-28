package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SourceGcpOrganizationsDataSource{}

func NewSourceGcpOrganizationsDataSource() datasource.DataSource {
	return &SourceGcpOrganizationsDataSource{}
}

type SourceGcpOrganizationsDataSource struct {
	client *client.EonClient
}

type SourceGcpOrganizationsDataSourceModel struct {
	Organizations []SourceGcpOrganizationDataModel `tfsdk:"organizations"`
}

type SourceGcpOrganizationDataModel struct {
	Id                         types.String `tfsdk:"id"`
	OrganizationId             types.String `tfsdk:"organization_id"`
	ManagementServiceAccountId types.String `tfsdk:"management_service_account_id"`
	ManagementProjectId        types.String `tfsdk:"management_project_id"`
	Name                       types.String `tfsdk:"name"`
	State                      types.String `tfsdk:"state"`
	ExcludeProjectPatterns     types.List   `tfsdk:"exclude_project_patterns"`
}

func (d *SourceGcpOrganizationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_gcp_organizations"
}

func (d *SourceGcpOrganizationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of source GCP organizations for the Eon project.",
		Attributes: map[string]schema.Attribute{
			"organizations": schema.ListNestedAttribute{
				MarkdownDescription: "List of source GCP organizations.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Eon-assigned organization ID.",
							Computed:            true,
						},
						"organization_id": schema.StringAttribute{
							MarkdownDescription: "GCP organization ID.",
							Computed:            true,
						},
						"management_service_account_id": schema.StringAttribute{
							MarkdownDescription: "Email of the GCP service account used to manage the organization.",
							Computed:            true,
						},
						"management_project_id": schema.StringAttribute{
							MarkdownDescription: "GCP project ID where the management service account resides.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Display name of the GCP organization in Eon.",
							Computed:            true,
						},
						"state": schema.StringAttribute{
							MarkdownDescription: "Connection state of the GCP organization.",
							Computed:            true,
						},
						"exclude_project_patterns": schema.ListAttribute{
							MarkdownDescription: "Glob patterns for GCP project IDs excluded from discovery.",
							Computed:            true,
							ElementType:         types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *SourceGcpOrganizationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SourceGcpOrganizationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SourceGcpOrganizationsDataSourceModel

	orgs, err := d.client.ListGcpOrganizations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read GCP organizations: %s", err))
		return
	}

	for _, org := range orgs {
		var patterns types.List
		if len(org.ExcludeProjectPatterns) > 0 {
			patternList, diags := types.ListValueFrom(ctx, types.StringType, org.ExcludeProjectPatterns)
			resp.Diagnostics.Append(diags...)
			patterns = patternList
		} else {
			patterns = types.ListNull(types.StringType)
		}

		orgModel := SourceGcpOrganizationDataModel{
			Id:                         types.StringValue(org.Id),
			OrganizationId:             types.StringValue(org.OrganizationId),
			ManagementServiceAccountId: types.StringValue(org.ManagementServiceAccountId),
			ManagementProjectId:        types.StringValue(org.ManagementProjectId),
			Name:                       types.StringValue(org.Name),
			State:                      types.StringValue(org.State),
			ExcludeProjectPatterns:     patterns,
		}

		data.Organizations = append(data.Organizations, orgModel)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
