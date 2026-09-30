package provider

import (
	"context"
	"fmt"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
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
	FindingExclusions []FindingExclusionListItemModel `tfsdk:"finding_exclusions"`
}

type FindingExclusionListItemModel struct {
	Id                 types.String `tfsdk:"id"`
	Scope              types.String `tfsdk:"scope"`
	ProviderResourceId types.String `tfsdk:"provider_resource_id"`
	Value              types.String `tfsdk:"value"`
	Type               types.String `tfsdk:"type"`
	Detector           types.String `tfsdk:"detector"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func (d *FindingExclusionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_finding_exclusions"
}

func (d *FindingExclusionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of finding exclusions in the Eon project.",
		Attributes: map[string]schema.Attribute{
			"finding_exclusions": schema.ListNestedAttribute{
				MarkdownDescription: "List of finding exclusions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Finding exclusion ID.",
							Computed:            true,
						},
						"scope": schema.StringAttribute{
							MarkdownDescription: "Where the exclusion applies. Possible values: `RESOURCE`, `ACCOUNT`.",
							Computed:            true,
						},
						"provider_resource_id": schema.StringAttribute{
							MarkdownDescription: "Cloud provider ID of the resource the exclusion applies to. Present when `scope` is `RESOURCE`.",
							Computed:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "Path prefix, table name, or database name the exclusion matches.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "What the exclusion matches. Possible values: `PATH`, `TABLE`, `DATABASE`.",
							Computed:            true,
						},
						"detector": schema.StringAttribute{
							MarkdownDescription: "Detector whose findings the exclusion suppresses. Possible values: `RANSOMWARE_BEHAVIOR`, `DATA_ANOMALY`, `MALWARE`.",
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
		data.FindingExclusions = append(data.FindingExclusions, findingExclusionListItem(exclusion))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func findingExclusionListItem(exclusion externalEonSdkAPI.FindingExclusion) FindingExclusionListItemModel {
	item := FindingExclusionListItemModel{
		Id:                 types.StringValue(exclusion.GetId()),
		Scope:              types.StringValue(string(exclusion.GetScope())),
		Value:              types.StringValue(exclusion.GetValue()),
		Type:               types.StringValue(string(exclusion.GetType())),
		Detector:           types.StringValue(string(exclusion.GetDetector())),
		UpdatedAt:          types.StringValue(exclusion.GetUpdatedAt().UTC().Format(time.RFC3339)),
		ProviderResourceId: types.StringNull(),
	}
	if id, ok := exclusion.GetProviderResourceIdOk(); ok && id != nil && *id != "" {
		item.ProviderResourceId = types.StringValue(*id)
	}
	return item
}
