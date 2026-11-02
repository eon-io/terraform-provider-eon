package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource = &RestoreAccountAwsS3BucketsDataSource{}
	_ datasource.DataSource = &RestoreAccountRegionAwsInstanceTypesDataSource{}
	_ datasource.DataSource = &RestoreAccountRegionAwsSecurityGroupsDataSource{}
)

type RestoreAccountAwsS3BucketsDataSource struct {
	client *client.EonClient
}

type RestoreAccountAwsS3BucketsDataSourceModel struct {
	RestoreAccountID types.String                     `tfsdk:"restore_account_id"`
	Buckets          []RestoreAccountAwsS3BucketModel `tfsdk:"buckets"`
}

type RestoreAccountAwsS3BucketModel struct {
	BucketName types.String `tfsdk:"bucket_name"`
	RegionName types.String `tfsdk:"region_name"`
}

func NewRestoreAccountAwsS3BucketsDataSource() datasource.DataSource {
	return &RestoreAccountAwsS3BucketsDataSource{}
}

func (d *RestoreAccountAwsS3BucketsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account_aws_s3_buckets"
}

func (d *RestoreAccountAwsS3BucketsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the S3 buckets available in an AWS restore account. Use this data source to select a destination bucket for `eon_restore_job`.",
		Attributes: map[string]schema.Attribute{
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the AWS restore account.",
				Required:            true,
			},
			"buckets": schema.ListNestedAttribute{
				MarkdownDescription: "List of S3 buckets available in the restore account.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"bucket_name": schema.StringAttribute{
							MarkdownDescription: "Name of the S3 bucket.",
							Computed:            true,
						},
						"region_name": schema.StringAttribute{
							MarkdownDescription: "AWS Region where the S3 bucket is located.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RestoreAccountAwsS3BucketsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureRestoreAccountAwsDataSource(ctx, req, resp)
}

func (d *RestoreAccountAwsS3BucketsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreAccountAwsS3BucketsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Listing S3 buckets for restore account", map[string]interface{}{
		"restore_account_id": data.RestoreAccountID.ValueString(),
	})

	buckets, err := d.client.ListS3Buckets(ctx, data.RestoreAccountID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read S3 buckets: %s", err))
		return
	}

	data.Buckets = make([]RestoreAccountAwsS3BucketModel, 0, len(buckets))
	for _, bucket := range buckets {
		data.Buckets = append(data.Buckets, RestoreAccountAwsS3BucketModel{
			BucketName: types.StringValue(bucket.GetBucketName()),
			RegionName: types.StringValue(bucket.GetRegionName()),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type RestoreAccountRegionAwsInstanceTypesDataSource struct {
	client *client.EonClient
}

type RestoreAccountRegionAwsInstanceTypesDataSourceModel struct {
	RestoreAccountID      types.String `tfsdk:"restore_account_id"`
	Region                types.String `tfsdk:"region"`
	SubnetID              types.String `tfsdk:"subnet_id"`
	VolumeCount           types.Int64  `tfsdk:"volume_count"`
	TotalVolumeBytes      types.Int64  `tfsdk:"total_volume_bytes"`
	InstanceTypes         types.List   `tfsdk:"instance_types"`
	InstantRestoreMapping types.Map    `tfsdk:"instant_restore_mapping"`
}

func NewRestoreAccountRegionAwsInstanceTypesDataSource() datasource.DataSource {
	return &RestoreAccountRegionAwsInstanceTypesDataSource{}
}

func (d *RestoreAccountRegionAwsInstanceTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account_region_aws_instance_types"
}

func (d *RestoreAccountRegionAwsInstanceTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves EC2 instance types available for restore in an AWS restore-account subnet. Use this data source to select an `instance_type` for `eon_restore_job`.",
		Attributes: map[string]schema.Attribute{
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the AWS restore account.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS Region where the subnet is located.",
				Required:            true,
			},
			"subnet_id": schema.StringAttribute{
				MarkdownDescription: "Cloud-provider-assigned ID of the subnet whose available instance types are listed.",
				Required:            true,
			},
			"volume_count": schema.Int64Attribute{
				MarkdownDescription: "Number of guest volumes planned for restore. When omitted, Eon uses its default host-memory reserve.",
				Optional:            true,
			},
			"total_volume_bytes": schema.Int64Attribute{
				MarkdownDescription: "Total size in bytes of guest volumes planned for restore. When omitted, Eon uses its default host-memory reserve.",
				Optional:            true,
			},
			"instance_types": schema.ListAttribute{
				MarkdownDescription: "List of EC2 instance types available for restore.",
				Computed:            true,
				ElementType:         types.StringType,
			},
			"instant_restore_mapping": schema.MapAttribute{
				MarkdownDescription: "Map from post-promotion to instant-restore host instance type. Instance types absent from this map are unavailable for instant restore.",
				Computed:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *RestoreAccountRegionAwsInstanceTypesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureRestoreAccountAwsDataSource(ctx, req, resp)
}

func (d *RestoreAccountRegionAwsInstanceTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreAccountRegionAwsInstanceTypesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var volumeCount *int32
	if !data.VolumeCount.IsNull() && !data.VolumeCount.IsUnknown() {
		value, err := SafeInt32Conversion(data.VolumeCount.ValueInt64())
		if err != nil {
			resp.Diagnostics.AddError("Invalid Attribute", fmt.Sprintf("volume_count: %s", err))
			return
		}
		volumeCount = &value
	}

	var totalVolumeBytes *int64
	if !data.TotalVolumeBytes.IsNull() && !data.TotalVolumeBytes.IsUnknown() {
		value := data.TotalVolumeBytes.ValueInt64()
		totalVolumeBytes = &value
	}

	tflog.Debug(ctx, "Listing AWS instance types for restore account", map[string]interface{}{
		"restore_account_id": data.RestoreAccountID.ValueString(),
		"region":             data.Region.ValueString(),
		"subnet_id":          data.SubnetID.ValueString(),
	})

	instanceTypes, err := d.client.ListAwsInstanceTypes(
		ctx,
		data.RestoreAccountID.ValueString(),
		data.Region.ValueString(),
		data.SubnetID.ValueString(),
		volumeCount,
		totalVolumeBytes,
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read AWS instance types: %s", err))
		return
	}

	data.InstanceTypes, resp.Diagnostics = types.ListValueFrom(ctx, types.StringType, instanceTypes.GetInstanceTypes())
	if resp.Diagnostics.HasError() {
		return
	}
	data.InstantRestoreMapping, resp.Diagnostics = types.MapValueFrom(ctx, types.StringType, instanceTypes.GetInstantRestoreMapping())
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

type RestoreAccountRegionAwsSecurityGroupsDataSource struct {
	client *client.EonClient
}

type RestoreAccountRegionAwsSecurityGroupsDataSourceModel struct {
	RestoreAccountID types.String                                        `tfsdk:"restore_account_id"`
	Region           types.String                                        `tfsdk:"region"`
	VpcID            types.String                                        `tfsdk:"vpc_id"`
	SubnetID         types.String                                        `tfsdk:"subnet_id"`
	SecurityGroups   []RestoreAccountRegionAwsSecurityGroupListItemModel `tfsdk:"security_groups"`
}

type RestoreAccountRegionAwsSecurityGroupListItemModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Vpc  types.String `tfsdk:"vpc"`
}

func NewRestoreAccountRegionAwsSecurityGroupsDataSource() datasource.DataSource {
	return &RestoreAccountRegionAwsSecurityGroupsDataSource{}
}

func (d *RestoreAccountRegionAwsSecurityGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account_region_aws_security_groups"
}

func (d *RestoreAccountRegionAwsSecurityGroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves security groups available in an AWS restore-account region. Use this data source to select `security_group_ids` for `eon_restore_job`.",
		Attributes: map[string]schema.Attribute{
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the AWS restore account.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS Region where the security groups are located.",
				Required:            true,
			},
			"vpc_id": schema.StringAttribute{
				MarkdownDescription: "Optional cloud-provider-assigned VPC ID to filter the security groups.",
				Optional:            true,
			},
			"subnet_id": schema.StringAttribute{
				MarkdownDescription: "Optional cloud-provider-assigned subnet ID to filter the security groups.",
				Optional:            true,
			},
			"security_groups": schema.ListNestedAttribute{
				MarkdownDescription: "List of security groups available in the restore account region.",
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
						"vpc": schema.StringAttribute{
							MarkdownDescription: "Cloud-provider-assigned ID of the VPC containing the security group.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RestoreAccountRegionAwsSecurityGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureRestoreAccountAwsDataSource(ctx, req, resp)
}

func (d *RestoreAccountRegionAwsSecurityGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreAccountRegionAwsSecurityGroupsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vpcID := optionalString(data.VpcID)
	subnetID := optionalString(data.SubnetID)
	tflog.Debug(ctx, "Listing AWS security groups for restore account", map[string]interface{}{
		"restore_account_id": data.RestoreAccountID.ValueString(),
		"region":             data.Region.ValueString(),
	})

	securityGroups, err := d.client.ListAwsSecurityGroups(
		ctx,
		data.RestoreAccountID.ValueString(),
		data.Region.ValueString(),
		vpcID,
		subnetID,
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read AWS security groups: %s", err))
		return
	}

	data.SecurityGroups = make([]RestoreAccountRegionAwsSecurityGroupListItemModel, 0, len(securityGroups.GetSecurityGroups()))
	for _, securityGroup := range securityGroups.GetSecurityGroups() {
		data.SecurityGroups = append(data.SecurityGroups, RestoreAccountRegionAwsSecurityGroupListItemModel{
			ID:   types.StringValue(securityGroup.GetId()),
			Name: types.StringValue(securityGroup.GetName()),
			Vpc:  types.StringValue(securityGroup.GetVpc()),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func configureRestoreAccountAwsDataSource(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *client.EonClient {
	if req.ProviderData == nil {
		return nil
	}

	eonClient, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.EonClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return nil
	}

	return eonClient
}

func optionalString(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	stringValue := value.ValueString()
	return &stringValue
}
