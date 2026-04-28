package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SourceGcpFoldersDataSource{}

func NewSourceGcpFoldersDataSource() datasource.DataSource {
	return &SourceGcpFoldersDataSource{}
}

type SourceGcpFoldersDataSource struct {
	client *client.EonClient
}

type SourceGcpFoldersDataSourceModel struct {
	Folders []SourceGcpFolderDataModel `tfsdk:"folders"`
}

type SourceGcpFolderDataModel struct {
	Id                         types.String `tfsdk:"id"`
	OrganizationId             types.String `tfsdk:"organization_id"`
	FolderId                   types.String `tfsdk:"folder_id"`
	ManagementServiceAccountId types.String `tfsdk:"management_service_account_id"`
	ManagementProjectId        types.String `tfsdk:"management_project_id"`
	Name                       types.String `tfsdk:"name"`
	State                      types.String `tfsdk:"state"`
	ExcludeProjectPatterns     types.List   `tfsdk:"exclude_project_patterns"`
}

func (d *SourceGcpFoldersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_gcp_folders"
}

func (d *SourceGcpFoldersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of source GCP folders for the Eon project.",
		Attributes: map[string]schema.Attribute{
			"folders": schema.ListNestedAttribute{
				MarkdownDescription: "List of source GCP folders.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Eon-assigned folder ID.",
							Computed:            true,
						},
						"organization_id": schema.StringAttribute{
							MarkdownDescription: "GCP organization ID that contains the folder.",
							Computed:            true,
						},
						"folder_id": schema.StringAttribute{
							MarkdownDescription: "GCP folder ID.",
							Computed:            true,
						},
						"management_service_account_id": schema.StringAttribute{
							MarkdownDescription: "Email of the GCP service account used to manage the folder.",
							Computed:            true,
						},
						"management_project_id": schema.StringAttribute{
							MarkdownDescription: "GCP project ID where the management service account resides.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Display name of the GCP folder in Eon.",
							Computed:            true,
						},
						"state": schema.StringAttribute{
							MarkdownDescription: "Connection state of the GCP folder.",
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

func (d *SourceGcpFoldersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SourceGcpFoldersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SourceGcpFoldersDataSourceModel

	folders, err := d.client.ListGcpFolders(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read GCP folders: %s", err))
		return
	}

	for _, folder := range folders {
		var patterns types.List
		if len(folder.ExcludeProjectPatterns) > 0 {
			patternList, diags := types.ListValueFrom(ctx, types.StringType, folder.ExcludeProjectPatterns)
			resp.Diagnostics.Append(diags...)
			patterns = patternList
		} else {
			patterns = types.ListNull(types.StringType)
		}

		folderModel := SourceGcpFolderDataModel{
			Id:                         types.StringValue(folder.Id),
			OrganizationId:             types.StringValue(folder.OrganizationId),
			FolderId:                   types.StringValue(folder.FolderId),
			ManagementServiceAccountId: types.StringValue(folder.ManagementServiceAccountId),
			ManagementProjectId:        types.StringValue(folder.ManagementProjectId),
			Name:                       types.StringValue(folder.Name),
			State:                      types.StringValue(folder.State),
			ExcludeProjectPatterns:     patterns,
		}

		data.Folders = append(data.Folders, folderModel)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
