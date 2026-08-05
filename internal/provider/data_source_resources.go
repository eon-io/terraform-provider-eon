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

var _ datasource.DataSource = &ResourcesDataSource{}

func NewResourcesDataSource() datasource.DataSource {
	return &ResourcesDataSource{}
}

type ResourcesDataSource struct {
	client *client.EonClient
}

type ResourcesDataSourceModel struct {
	Ids                 types.List          `tfsdk:"ids"`
	ProviderResourceIds types.List          `tfsdk:"provider_resource_ids"`
	ResourceNames       types.List          `tfsdk:"resource_names"`
	ResourceTypes       types.List          `tfsdk:"resource_types"`
	CloudProviders      types.List          `tfsdk:"cloud_providers"`
	ProviderAccountIds  types.List          `tfsdk:"provider_account_ids"`
	Environments        types.List          `tfsdk:"environments"`
	Resources           []ResourceListModel `tfsdk:"resources"`
}

type ResourceListModel struct {
	Id                 types.String `tfsdk:"id"`
	ProviderResourceId types.String `tfsdk:"provider_resource_id"`
	ResourceName       types.String `tfsdk:"resource_name"`
	ProviderAccountId  types.String `tfsdk:"provider_account_id"`
	CloudProvider      types.String `tfsdk:"cloud_provider"`
	ResourceType       types.String `tfsdk:"resource_type"`
	Region             types.String `tfsdk:"region"`
	BackupStatus       types.String `tfsdk:"backup_status"`
}

func (d *ResourcesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resources"
}

func (d *ResourcesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a filtered list of inventory resources in the Eon project. Use this data source to resolve Eon resource IDs for other resources that require them.",
		Attributes: map[string]schema.Attribute{
			"ids": schema.ListAttribute{
				MarkdownDescription: "Filter to resources whose Eon-assigned IDs are in this list.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"provider_resource_ids": schema.ListAttribute{
				MarkdownDescription: "Filter to resources whose cloud-provider-assigned IDs are in this list.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"resource_names": schema.ListAttribute{
				MarkdownDescription: "Filter to resources whose display names are in this list.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"resource_types": schema.ListAttribute{
				MarkdownDescription: "Filter to resources whose types are in this list. Possible values include `AWS_EC2`, `AWS_S3`, `AWS_RDS`, `AWS_EBS`, `AZURE_VM`, `GCP_COMPUTE_INSTANCE`, and other Eon resource types.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"cloud_providers": schema.ListAttribute{
				MarkdownDescription: "Filter to resources whose cloud providers are in this list. Possible values: `AWS`, `AZURE`, `GCP`.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"provider_account_ids": schema.ListAttribute{
				MarkdownDescription: "Filter to resources whose cloud-provider-assigned account IDs are in this list.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"environments": schema.ListAttribute{
				MarkdownDescription: "Filter to resources whose environments are in this list. Possible values: `PROD`, `PROD_INTERNAL`, `STAGE`, `ENVIRONMENT_UNSPECIFIED`.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"resources": schema.ListNestedAttribute{
				MarkdownDescription: "List of matching inventory resources.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Eon-assigned resource ID.",
							Computed:            true,
						},
						"provider_resource_id": schema.StringAttribute{
							MarkdownDescription: "Cloud-provider-assigned resource ID.",
							Computed:            true,
						},
						"resource_name": schema.StringAttribute{
							MarkdownDescription: "Resource display name.",
							Computed:            true,
						},
						"provider_account_id": schema.StringAttribute{
							MarkdownDescription: "Cloud-provider-assigned account ID.",
							Computed:            true,
						},
						"cloud_provider": schema.StringAttribute{
							MarkdownDescription: "Cloud provider. Possible values: `AWS`, `AZURE`, `GCP`.",
							Computed:            true,
						},
						"resource_type": schema.StringAttribute{
							MarkdownDescription: "Resource type, for example `AWS_EC2` or `AWS_S3`.",
							Computed:            true,
						},
						"region": schema.StringAttribute{
							MarkdownDescription: "Region the resource is hosted in.",
							Computed:            true,
						},
						"backup_status": schema.StringAttribute{
							MarkdownDescription: "Backup status of the resource.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *ResourcesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ResourcesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ResourcesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filters, err := buildListResourcesFilters(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Configuration Error", fmt.Sprintf("Unable to build resource filters: %s", err))
		return
	}

	tflog.Debug(ctx, "Listing inventory resources")

	resources, err := d.client.ListResources(ctx, filters)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read resources: %s", err))
		return
	}

	data.Resources = make([]ResourceListModel, 0, len(resources))
	for _, resource := range resources {
		data.Resources = append(data.Resources, ResourceListModel{
			Id:                 types.StringValue(resource.GetId()),
			ProviderResourceId: types.StringValue(resource.GetProviderResourceId()),
			ResourceName:       types.StringValue(resource.GetResourceName()),
			ProviderAccountId:  types.StringValue(resource.GetProviderAccountId()),
			CloudProvider:      types.StringValue(string(resource.GetCloudProvider())),
			ResourceType:       types.StringValue(string(resource.GetResourceType())),
			Region:             types.StringValue(resource.GetRegion()),
			BackupStatus:       types.StringValue(string(resource.GetBackupStatus())),
		})
	}

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func buildListResourcesFilters(ctx context.Context, data ResourcesDataSourceModel) (*externalEonSdkAPI.InventoryFilterConditions, error) {
	filters := externalEonSdkAPI.NewInventoryFilterConditions()
	hasFilters := false

	ids, err := optionalStringList(ctx, data.Ids)
	if err != nil {
		return nil, fmt.Errorf("ids: %w", err)
	}
	if len(ids) > 0 {
		idFilters := externalEonSdkAPI.NewIdFilters()
		idFilters.SetIn(ids)
		filters.SetId(*idFilters)
		hasFilters = true
	}

	providerResourceIds, err := optionalStringList(ctx, data.ProviderResourceIds)
	if err != nil {
		return nil, fmt.Errorf("provider_resource_ids: %w", err)
	}
	if len(providerResourceIds) > 0 {
		providerIdFilters := externalEonSdkAPI.NewResourceIdFilters()
		providerIdFilters.SetIn(providerResourceIds)
		filters.SetProviderResourceId(*providerIdFilters)
		hasFilters = true
	}

	resourceNames, err := optionalStringList(ctx, data.ResourceNames)
	if err != nil {
		return nil, fmt.Errorf("resource_names: %w", err)
	}
	if len(resourceNames) > 0 {
		nameFilters := externalEonSdkAPI.NewResourceNameFilters()
		nameFilters.SetIn(resourceNames)
		filters.SetResourceName(*nameFilters)
		hasFilters = true
	}

	resourceTypes, err := optionalStringList(ctx, data.ResourceTypes)
	if err != nil {
		return nil, fmt.Errorf("resource_types: %w", err)
	}
	if len(resourceTypes) > 0 {
		typeEnums := make([]externalEonSdkAPI.ResourceType, 0, len(resourceTypes))
		for _, value := range resourceTypes {
			typeEnums = append(typeEnums, externalEonSdkAPI.ResourceType(value))
		}
		typeFilters := externalEonSdkAPI.NewResourceTypeFilters()
		typeFilters.SetIn(typeEnums)
		filters.SetResourceType(*typeFilters)
		hasFilters = true
	}

	cloudProviders, err := optionalStringList(ctx, data.CloudProviders)
	if err != nil {
		return nil, fmt.Errorf("cloud_providers: %w", err)
	}
	if len(cloudProviders) > 0 {
		providerEnums := make([]externalEonSdkAPI.Provider, 0, len(cloudProviders))
		for _, value := range cloudProviders {
			providerEnums = append(providerEnums, externalEonSdkAPI.Provider(value))
		}
		providerFilters := externalEonSdkAPI.NewCloudProviderFilters()
		providerFilters.SetIn(providerEnums)
		filters.SetCloudProvider(*providerFilters)
		hasFilters = true
	}

	providerAccountIds, err := optionalStringList(ctx, data.ProviderAccountIds)
	if err != nil {
		return nil, fmt.Errorf("provider_account_ids: %w", err)
	}
	if len(providerAccountIds) > 0 {
		accountFilters := externalEonSdkAPI.NewAccountIdFilters()
		accountFilters.SetIn(providerAccountIds)
		filters.SetAccountId(*accountFilters)
		hasFilters = true
	}

	environments, err := optionalStringList(ctx, data.Environments)
	if err != nil {
		return nil, fmt.Errorf("environments: %w", err)
	}
	if len(environments) > 0 {
		envEnums := make([]externalEonSdkAPI.Environment, 0, len(environments))
		for _, value := range environments {
			envEnums = append(envEnums, externalEonSdkAPI.Environment(value))
		}
		envFilters := externalEonSdkAPI.NewEnvironmentFilters()
		envFilters.SetIn(envEnums)
		filters.SetEnvironment(*envFilters)
		hasFilters = true
	}

	if !hasFilters {
		return nil, nil
	}
	return filters, nil
}

func optionalStringList(ctx context.Context, list types.List) ([]string, error) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}

	var values []types.String
	diags := list.ElementsAs(ctx, &values, false)
	if diags.HasError() {
		return nil, fmt.Errorf("%s", diags.Errors())
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.ValueString())
	}
	return out, nil
}
