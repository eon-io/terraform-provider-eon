package provider

import (
	"context"
	"fmt"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &FindingExclusionsDataSource{}

func NewFindingExclusionsDataSource() datasource.DataSource {
	return &FindingExclusionsDataSource{}
}

type FindingExclusionsDataSource struct {
	client *client.EonClient
}

type FindingExclusionsDataSourceModel struct {
	ResourceIds       types.List                      `tfsdk:"resource_ids"`
	AccountWide       types.Bool                      `tfsdk:"account_wide"`
	Types             types.List                      `tfsdk:"types"`
	Detectors         types.List                      `tfsdk:"detectors"`
	ValueContains     types.List                      `tfsdk:"value_contains"`
	FindingExclusions []FindingExclusionListItemModel `tfsdk:"finding_exclusions"`
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
		MarkdownDescription: "Retrieves a list of finding exclusions in the Eon project. Every filter is optional; filters you set are combined with `AND`.",
		Attributes: map[string]schema.Attribute{
			"resource_ids": schema.ListAttribute{
				MarkdownDescription: "Only return exclusions scoped to one of these Eon-assigned resource IDs.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"account_wide": schema.BoolAttribute{
				MarkdownDescription: "When `true`, only return exclusions that apply to every resource in the account. When `false`, only return exclusions scoped to a single resource.",
				Optional:            true,
			},
			"types": schema.ListAttribute{
				MarkdownDescription: "Only return exclusions of these types. Supported values: `PATH`, `TABLE`, `DATABASE`.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"detectors": schema.ListAttribute{
				MarkdownDescription: "Only return exclusions for these detectors. Supported values: `RANSOMWARE_BEHAVIOR`, `DATA_ANOMALY`, `MALWARE`.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"value_contains": schema.ListAttribute{
				MarkdownDescription: "Only return exclusions whose `value` contains one of these substrings.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"finding_exclusions": schema.ListNestedAttribute{
				MarkdownDescription: "List of finding exclusions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Eon-assigned ID of the finding exclusion.",
							Computed:            true,
						},
						"resource_id": schema.StringAttribute{
							MarkdownDescription: "Eon-assigned ID of the resource the exclusion applies to. Null when the exclusion applies to every resource in the account.",
							Computed:            true,
						},
						"value": schema.StringAttribute{
							MarkdownDescription: "What the exclusion matches: a path prefix for `PATH`, or the exact name for `TABLE` and `DATABASE`.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "What the exclusion refers to. Possible values: `PATH`, `TABLE`, `DATABASE`.",
							Computed:            true,
						},
						"detector": schema.StringAttribute{
							MarkdownDescription: "Detector whose findings the exclusion suppresses. Possible values: `RANSOMWARE_BEHAVIOR`, `DATA_ANOMALY`, `MALWARE`.",
							Computed:            true,
						},
						"updated_at": schema.StringAttribute{
							MarkdownDescription: "When the exclusion was created or last updated, in RFC 3339 format.",
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

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filters, diags := buildFindingExclusionFilters(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Listing finding exclusions", map[string]interface{}{"filtered": filters != nil})

	exclusions, err := d.client.ListFindingExclusions(ctx, filters)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read finding exclusions: %s", err))
		return
	}

	data.FindingExclusions = make([]FindingExclusionListItemModel, 0, len(exclusions))
	for _, exclusion := range exclusions {
		data.FindingExclusions = append(data.FindingExclusions, findingExclusionListItem(exclusion))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func findingExclusionListItem(exclusion externalEonSdkAPI.FindingExclusion) FindingExclusionListItemModel {
	item := FindingExclusionListItemModel{
		Id:         types.StringValue(exclusion.GetId()),
		ResourceId: types.StringNull(),
		Value:      types.StringValue(exclusion.GetValue()),
		Type:       types.StringValue(string(exclusion.GetType())),
		Detector:   types.StringValue(string(exclusion.GetDetector())),
		UpdatedAt:  types.StringValue(exclusion.GetUpdatedAt().UTC().Format(time.RFC3339)),
	}
	if resourceId, ok := exclusion.GetResourceIdOk(); ok && resourceId != nil {
		item.ResourceId = types.StringValue(*resourceId)
	}
	return item
}

// buildFindingExclusionFilters renders the data source's optional filter attributes as the SDK
// filter conditions. It returns nil when no filter is set so the list request carries no body filter.
func buildFindingExclusionFilters(ctx context.Context, data *FindingExclusionsDataSourceModel) (*externalEonSdkAPI.FindingExclusionFilterConditions, diag.Diagnostics) {
	var diags diag.Diagnostics
	filters := externalEonSdkAPI.NewFindingExclusionFilterConditions()
	filtered := false

	resourceIds, d := stringListElements(ctx, data.ResourceIds)
	diags.Append(d...)
	hasAccountWide := !data.AccountWide.IsNull() && !data.AccountWide.IsUnknown()
	if len(resourceIds) > 0 || hasAccountWide {
		resourceFilter := externalEonSdkAPI.NewFindingExclusionResourceFilter()
		if len(resourceIds) > 0 {
			resourceFilter.SetIn(resourceIds)
		}
		if hasAccountWide {
			resourceFilter.SetIsAccountWide(data.AccountWide.ValueBool())
		}
		filters.SetResource(*resourceFilter)
		filtered = true
	}

	typeValues, d := stringListElements(ctx, data.Types)
	diags.Append(d...)
	if len(typeValues) > 0 {
		objectTypes := make([]externalEonSdkAPI.FindingObjectType, 0, len(typeValues))
		for _, value := range typeValues {
			objectType, err := externalEonSdkAPI.NewFindingObjectTypeFromValue(value)
			if err != nil {
				diags.AddError("Invalid Attribute", fmt.Sprintf("types: %s", err))
				continue
			}
			objectTypes = append(objectTypes, *objectType)
		}
		typeFilter := externalEonSdkAPI.NewFindingExclusionTypeFilters()
		typeFilter.SetIn(objectTypes)
		filters.SetType(*typeFilter)
		filtered = true
	}

	detectorValues, d := stringListElements(ctx, data.Detectors)
	diags.Append(d...)
	if len(detectorValues) > 0 {
		detectors := make([]externalEonSdkAPI.FindingExclusionDetectorType, 0, len(detectorValues))
		for _, value := range detectorValues {
			detector, err := externalEonSdkAPI.NewFindingExclusionDetectorTypeFromValue(value)
			if err != nil {
				diags.AddError("Invalid Attribute", fmt.Sprintf("detectors: %s", err))
				continue
			}
			detectors = append(detectors, *detector)
		}
		detectorFilter := externalEonSdkAPI.NewFindingExclusionDetectorFilters()
		detectorFilter.SetIn(detectors)
		filters.SetDetector(*detectorFilter)
		filtered = true
	}

	valueContains, d := stringListElements(ctx, data.ValueContains)
	diags.Append(d...)
	if len(valueContains) > 0 {
		valueFilter := externalEonSdkAPI.NewFindingExclusionValueFilters()
		valueFilter.SetContains(valueContains)
		filters.SetValue(*valueFilter)
		filtered = true
	}

	if diags.HasError() || !filtered {
		return nil, diags
	}
	return filters, diags
}

func stringListElements(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	var elements []string
	diags.Append(list.ElementsAs(ctx, &elements, false)...)
	return elements, diags
}
