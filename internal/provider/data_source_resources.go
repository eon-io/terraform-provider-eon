package provider

import (
	"context"
	"fmt"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	AccountIds          types.List          `tfsdk:"account_ids"`
	BackupStatuses      types.List          `tfsdk:"backup_statuses"`
	Environments        types.List          `tfsdk:"environments"`
	TotalCount          types.Int64         `tfsdk:"total_count"`
	Resources           []ResourceListModel `tfsdk:"resources"`
}

type ResourceListModel struct {
	Id                 types.String `tfsdk:"id"`
	ProviderResourceId types.String `tfsdk:"provider_resource_id"`
	ResourceName       types.String `tfsdk:"resource_name"`
	BackupStatus       types.String `tfsdk:"backup_status"`
	ProviderAccountId  types.String `tfsdk:"provider_account_id"`
	CloudProvider      types.String `tfsdk:"cloud_provider"`
	ResourceType       types.String `tfsdk:"resource_type"`
	Region             types.String `tfsdk:"region"`
	Vpc                types.String `tfsdk:"vpc"`
	Subnets            types.List   `tfsdk:"subnets"`
	Tags               types.Map    `tfsdk:"tags"`
	Environment        types.String `tfsdk:"environment"`
	DataClasses        types.List   `tfsdk:"data_classes"`
	CreatedTime        types.String `tfsdk:"created_time"`
	DiscoveredTime     types.String `tfsdk:"discovered_time"`
	LatestSnapshotTime types.String `tfsdk:"latest_snapshot_time"`
	OldestSnapshotTime types.String `tfsdk:"oldest_snapshot_time"`
}

func (d *ResourcesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resources"
}

func (d *ResourcesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a filtered list of inventory resources in the Eon project. Use this data source to resolve Eon resource IDs for `eon_restore_job`, `eon_resource_snapshots`, backup exclusions, and classification overrides.",
		Attributes: map[string]schema.Attribute{
			"ids": schema.ListAttribute{
				MarkdownDescription: "Optional filter: match resources whose Eon-assigned ID is in this list.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"provider_resource_ids": schema.ListAttribute{
				MarkdownDescription: "Optional filter: match resources whose cloud-provider-assigned ID is in this list.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"resource_names": schema.ListAttribute{
				MarkdownDescription: "Optional filter: match resources whose display name is in this list.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"resource_types": schema.ListAttribute{
				MarkdownDescription: "Optional filter: match resources whose type is in this list (for example `AWS_EC2`, `AWS_RDS`).",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"cloud_providers": schema.ListAttribute{
				MarkdownDescription: "Optional filter: match resources whose cloud provider is in this list. Possible values: `AWS`, `AZURE`, `GCP`.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"account_ids": schema.ListAttribute{
				MarkdownDescription: "Optional filter: match resources whose cloud-provider-assigned source account ID is in this list.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"backup_statuses": schema.ListAttribute{
				MarkdownDescription: "Optional filter: match resources whose backup status is in this list (for example `PROTECTED`, `EXCLUDED_FROM_BACKUP`).",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"environments": schema.ListAttribute{
				MarkdownDescription: "Optional filter: match resources whose environment classification is in this list (for example `PROD`, `STAGE`, `DEV`).",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"total_count": schema.Int64Attribute{
				MarkdownDescription: "Number of resources returned.",
				Computed:            true,
			},
			"resources": schema.ListNestedAttribute{
				MarkdownDescription: "List of inventory resources matching the filters.",
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
						"backup_status": schema.StringAttribute{
							MarkdownDescription: "Backup protection status of the resource.",
							Computed:            true,
						},
						"provider_account_id": schema.StringAttribute{
							MarkdownDescription: "Cloud-provider-assigned source account ID.",
							Computed:            true,
						},
						"cloud_provider": schema.StringAttribute{
							MarkdownDescription: "Cloud provider. Possible values: `AWS`, `AZURE`, `GCP`.",
							Computed:            true,
						},
						"resource_type": schema.StringAttribute{
							MarkdownDescription: "Resource type (for example `AWS_EC2`, `AWS_RDS`).",
							Computed:            true,
						},
						"region": schema.StringAttribute{
							MarkdownDescription: "Region the resource is hosted in.",
							Computed:            true,
						},
						"vpc": schema.StringAttribute{
							MarkdownDescription: "VPC the resource is in, when applicable.",
							Computed:            true,
						},
						"subnets": schema.ListAttribute{
							MarkdownDescription: "Subnets the resource belongs to, when applicable.",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"tags": schema.MapAttribute{
							MarkdownDescription: "Resource tags as key-value pairs.",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"environment": schema.StringAttribute{
							MarkdownDescription: "Effective environment classification, when present.",
							Computed:            true,
						},
						"data_classes": schema.ListAttribute{
							MarkdownDescription: "Effective data-classification labels, when present.",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"created_time": schema.StringAttribute{
							MarkdownDescription: "Date and time the resource record was created in Eon.",
							Computed:            true,
						},
						"discovered_time": schema.StringAttribute{
							MarkdownDescription: "Date and time the resource was first discovered.",
							Computed:            true,
						},
						"latest_snapshot_time": schema.StringAttribute{
							MarkdownDescription: "Date and time of the resource's latest Eon snapshot.",
							Computed:            true,
						},
						"oldest_snapshot_time": schema.StringAttribute{
							MarkdownDescription: "Date and time of the resource's first Eon snapshot.",
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

	filters, diags := buildListResourcesFilters(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
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
		model, modelDiags := resourceListModelFromInventoryResource(ctx, resource)
		resp.Diagnostics.Append(modelDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Resources = append(data.Resources, model)
	}

	totalCount, err := SafeInt32Conversion(int64(len(resources)))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to convert resource count: %s", err))
		return
	}
	data.TotalCount = types.Int64Value(int64(totalCount))

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func buildListResourcesFilters(ctx context.Context, data ResourcesDataSourceModel) (*externalEonSdkAPI.InventoryFilterConditions, diag.Diagnostics) {
	var diags diag.Diagnostics

	ids, d := listFilterStrings(ctx, data.Ids)
	diags.Append(d...)
	providerResourceIds, d := listFilterStrings(ctx, data.ProviderResourceIds)
	diags.Append(d...)
	resourceNames, d := listFilterStrings(ctx, data.ResourceNames)
	diags.Append(d...)
	resourceTypes, d := listFilterStrings(ctx, data.ResourceTypes)
	diags.Append(d...)
	cloudProviders, d := listFilterStrings(ctx, data.CloudProviders)
	diags.Append(d...)
	accountIds, d := listFilterStrings(ctx, data.AccountIds)
	diags.Append(d...)
	backupStatuses, d := listFilterStrings(ctx, data.BackupStatuses)
	diags.Append(d...)
	environments, d := listFilterStrings(ctx, data.Environments)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	hasFilter := len(ids) > 0 || len(providerResourceIds) > 0 || len(resourceNames) > 0 ||
		len(resourceTypes) > 0 || len(cloudProviders) > 0 || len(accountIds) > 0 ||
		len(backupStatuses) > 0 || len(environments) > 0
	if !hasFilter {
		return nil, diags
	}

	filters := externalEonSdkAPI.NewInventoryFilterConditions()
	if len(ids) > 0 {
		idFilters := externalEonSdkAPI.NewIdFilters()
		idFilters.SetIn(ids)
		filters.SetId(*idFilters)
	}
	if len(providerResourceIds) > 0 {
		providerFilters := externalEonSdkAPI.NewResourceIdFilters()
		providerFilters.SetIn(providerResourceIds)
		filters.SetProviderResourceId(*providerFilters)
	}
	if len(resourceNames) > 0 {
		nameFilters := externalEonSdkAPI.NewResourceNameFilters()
		nameFilters.SetIn(resourceNames)
		filters.SetResourceName(*nameFilters)
	}
	if len(resourceTypes) > 0 {
		typeFilters := externalEonSdkAPI.NewResourceTypeFilters()
		typeValues := make([]externalEonSdkAPI.ResourceType, len(resourceTypes))
		for i, value := range resourceTypes {
			typeValues[i] = externalEonSdkAPI.ResourceType(value)
		}
		typeFilters.SetIn(typeValues)
		filters.SetResourceType(*typeFilters)
	}
	if len(cloudProviders) > 0 {
		providerFilters := externalEonSdkAPI.NewCloudProviderFilters()
		providerValues := make([]externalEonSdkAPI.Provider, len(cloudProviders))
		for i, value := range cloudProviders {
			providerValues[i] = externalEonSdkAPI.Provider(value)
		}
		providerFilters.SetIn(providerValues)
		filters.SetCloudProvider(*providerFilters)
	}
	if len(accountIds) > 0 {
		accountFilters := externalEonSdkAPI.NewAccountIdFilters()
		accountFilters.SetIn(accountIds)
		filters.SetAccountId(*accountFilters)
	}
	if len(backupStatuses) > 0 {
		statusFilters := externalEonSdkAPI.NewBackupStatusFilters()
		statusValues := make([]externalEonSdkAPI.BackupStatus, len(backupStatuses))
		for i, value := range backupStatuses {
			statusValues[i] = externalEonSdkAPI.BackupStatus(value)
		}
		statusFilters.SetIn(statusValues)
		filters.SetBackupStatus(*statusFilters)
	}
	if len(environments) > 0 {
		environmentFilters := externalEonSdkAPI.NewEnvironmentFilters()
		environmentValues := make([]externalEonSdkAPI.Environment, len(environments))
		for i, value := range environments {
			environmentValues[i] = externalEonSdkAPI.Environment(value)
		}
		environmentFilters.SetIn(environmentValues)
		filters.SetEnvironment(*environmentFilters)
	}

	return filters, diags
}

func listFilterStrings(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}

	var values []string
	d := list.ElementsAs(ctx, &values, false)
	if d.HasError() {
		return nil, d
	}
	return values, nil
}

func resourceListModelFromInventoryResource(ctx context.Context, resource externalEonSdkAPI.InventoryResource) (ResourceListModel, diag.Diagnostics) {
	model := ResourceListModel{
		Id:                 types.StringValue(resource.GetId()),
		ProviderResourceId: types.StringValue(resource.GetProviderResourceId()),
		ResourceName:       types.StringValue(resource.GetResourceName()),
		BackupStatus:       types.StringValue(string(resource.GetBackupStatus())),
		ProviderAccountId:  types.StringValue(resource.GetProviderAccountId()),
		CloudProvider:      types.StringValue(string(resource.GetCloudProvider())),
		ResourceType:       types.StringValue(string(resource.GetResourceType())),
		Region:             types.StringValue(resource.GetRegion()),
	}

	if resource.Vpc != nil {
		model.Vpc = types.StringValue(*resource.Vpc)
	} else {
		model.Vpc = types.StringNull()
	}

	if len(resource.Subnets) > 0 {
		subnets, d := types.ListValueFrom(ctx, types.StringType, resource.Subnets)
		if d.HasError() {
			return ResourceListModel{}, d
		}
		model.Subnets = subnets
	} else {
		model.Subnets = types.ListNull(types.StringType)
	}

	if len(resource.Tags) > 0 {
		tagValues := make(map[string]string, len(resource.Tags))
		for key, value := range resource.Tags {
			tagValues[key] = value
		}
		tags, d := types.MapValueFrom(ctx, types.StringType, tagValues)
		if d.HasError() {
			return ResourceListModel{}, d
		}
		model.Tags = tags
	} else {
		model.Tags = types.MapNull(types.StringType)
	}

	model.Environment, model.DataClasses = classificationsFromInventoryResource(ctx, resource)

	if resource.HasCreatedTime() {
		model.CreatedTime = types.StringValue(resource.GetCreatedTime().String())
	} else {
		model.CreatedTime = types.StringNull()
	}
	if resource.HasDiscoveredTime() {
		model.DiscoveredTime = types.StringValue(resource.GetDiscoveredTime().String())
	} else {
		model.DiscoveredTime = types.StringNull()
	}
	if resource.HasLatestSnapshotTime() {
		model.LatestSnapshotTime = types.StringValue(resource.GetLatestSnapshotTime().String())
	} else {
		model.LatestSnapshotTime = types.StringNull()
	}
	if resource.HasOldestSnapshotTime() {
		model.OldestSnapshotTime = types.StringValue(resource.GetOldestSnapshotTime().String())
	} else {
		model.OldestSnapshotTime = types.StringNull()
	}

	return model, nil
}

func classificationsFromInventoryResource(ctx context.Context, resource externalEonSdkAPI.InventoryResource) (types.String, types.List) {
	if resource.Classifications == nil {
		return types.StringNull(), types.ListNull(types.StringType)
	}

	var environment types.String
	if details, ok := resource.Classifications.GetEnvironmentDetailsOk(); ok && details != nil {
		if env, ok := details.GetEnvironmentOk(); ok && env != nil {
			environment = types.StringValue(string(*env))
		} else {
			environment = types.StringNull()
		}
	} else {
		environment = types.StringNull()
	}

	var dataClasses types.List
	if details, ok := resource.Classifications.GetDataClassesDetailsOk(); ok && details != nil {
		classes := details.GetDataClasses()
		if len(classes) > 0 {
			list, diags := types.ListValueFrom(ctx, types.StringType, classes)
			if diags.HasError() {
				return environment, types.ListNull(types.StringType)
			}
			dataClasses = list
		} else {
			dataClasses = types.ListNull(types.StringType)
		}
	} else {
		dataClasses = types.ListNull(types.StringType)
	}

	return environment, dataClasses
}
