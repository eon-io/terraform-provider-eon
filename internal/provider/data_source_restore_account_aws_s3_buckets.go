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

var _ datasource.DataSource = &RestoreAccountAwsS3BucketsDataSource{}

func NewRestoreAccountAwsS3BucketsDataSource() datasource.DataSource {
	return &RestoreAccountAwsS3BucketsDataSource{}
}

type RestoreAccountAwsS3BucketsDataSource struct {
	client *client.EonClient
}

type RestoreAccountAwsS3BucketsDataSourceModel struct {
	RestoreAccountId types.String                     `tfsdk:"restore_account_id"`
	Buckets          []RestoreAccountAwsS3BucketModel `tfsdk:"buckets"`
}

type RestoreAccountAwsS3BucketModel struct {
	BucketName types.String `tfsdk:"bucket_name"`
	RegionName types.String `tfsdk:"region_name"`
}

func (d *RestoreAccountAwsS3BucketsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account_aws_s3_buckets"
}

func (d *RestoreAccountAwsS3BucketsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the list of S3 buckets available to an AWS restore account. Use this data source to select a destination bucket for `eon_restore_job` S3 restores.",
		Attributes: map[string]schema.Attribute{
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the AWS restore account whose S3 buckets are listed.",
				Required:            true,
			},
			"buckets": schema.ListNestedAttribute{
				MarkdownDescription: "List of S3 buckets available to the restore account.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"bucket_name": schema.StringAttribute{
							MarkdownDescription: "Name of the S3 bucket.",
							Computed:            true,
						},
						"region_name": schema.StringAttribute{
							MarkdownDescription: "Region the S3 bucket is located in.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RestoreAccountAwsS3BucketsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RestoreAccountAwsS3BucketsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreAccountAwsS3BucketsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Listing restore account S3 buckets", map[string]interface{}{
		"restore_account_id": data.RestoreAccountId.ValueString(),
	})

	buckets, err := d.client.ListS3Buckets(ctx, data.RestoreAccountId.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore account S3 buckets: %s", err))
		return
	}

	data.Buckets = make([]RestoreAccountAwsS3BucketModel, 0, len(buckets))
	for _, bucket := range buckets {
		data.Buckets = append(data.Buckets, RestoreAccountAwsS3BucketModel{
			BucketName: types.StringValue(bucket.GetBucketName()),
			RegionName: types.StringValue(bucket.GetRegionName()),
		})
	}

	tflog.Trace(ctx, "read a data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
