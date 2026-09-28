package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &FindingExclusionsDataSource{}

func NewFindingExclusionsDataSource() datasource.DataSource {
	return &FindingExclusionsDataSource{}
}

type FindingExclusionsDataSource struct {
	client *client.EonClient
}

type FindingExclusionsDataSourceModel struct {
	Exclusions []FindingExclusionListItemModel `tfsdk:"exclusions"`
}

type FindingExclusionListItemModel struct {
	Id         types.String `tfsdk:"id"`
	ResourceId types.String `tfsdk:"resource_id"`
	Value      types.String `tfsdk:"value"`
	Type       types.String `tfsdk:"type"`
	Detector   types.String `tfsdk:"detector"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

func (d *FindingExclusionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_finding_exclusions"
}

func (d *FindingExclusionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves finding exclusions in the Eon project.",
		Attributes: map[string]schema.Attribute{
			"exclusions": schema.ListNestedAttribute{
				MarkdownDescription: "Finding exclusions in the project.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Eon-assigned ID of the finding exclusion.",
							Computed:            true,
						},
						"resource_id": schema.StringAttribute{
							MarkdownDescription: "Eon-assigned ID of the resource the exclusion applies to. Unset when the exclusion applies to every resource in the account.",
							Computed:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "What the exclusion matches. For `PATH`, a file path prefix. For `TABLE` or `DATABASE`, the exact table or database name.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "What the exclusion refers to. Possible values: `PATH`, `TABLE`, `DATABASE`.",
							Computed:            true,
						},
						"detector": schema.StringAttribute{
							MarkdownDescription: "Detector whose findings this exclusion suppresses. Possible values: `RANSOMWARE_BEHAVIOR`, `DATA_ANOMALY`, `MALWARE`.",
							Computed:            true,
						},
						"updated_at": schema.StringAttribute{
							MarkdownDescription: "When the exclusion was created or last updated.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *FindingExclusionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *FindingExclusionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FindingExclusionsDataSourceModel

	exclusions, err := d.client.ListFindingExclusions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read finding exclusions: %s", err))
		return
	}

	for _, exclusion := range exclusions {
		data.Exclusions = append(data.Exclusions, findingExclusionListItem(exclusion))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
