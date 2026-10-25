package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RestoreS3BucketsDataSource{}

func NewRestoreS3BucketsDataSource() datasource.DataSource {
	return &RestoreS3BucketsDataSource{}
}

type RestoreS3BucketsDataSource struct {
	client *client.EonClient
}

type RestoreS3BucketsDataSourceModel struct {
	RestoreAccountID types.String           `tfsdk:"restore_account_id"`
	Buckets          []RestoreS3BucketModel `tfsdk:"buckets"`
}

type RestoreS3BucketModel struct {
	BucketName types.String `tfsdk:"bucket_name"`
	RegionName types.String `tfsdk:"region_name"`
}

func (d *RestoreS3BucketsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_s3_buckets"
}

func (d *RestoreS3BucketsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists S3 buckets available to a restore account.",
		Attributes: map[string]schema.Attribute{
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon restore account ID.",
				Required:            true,
			},
			"buckets": schema.ListNestedAttribute{
				MarkdownDescription: "S3 buckets available to the restore account.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"bucket_name": schema.StringAttribute{
							MarkdownDescription: "S3 bucket name.",
							Computed:            true,
						},
						"region_name": schema.StringAttribute{
							MarkdownDescription: "AWS region containing the bucket.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RestoreS3BucketsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RestoreS3BucketsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreS3BucketsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	buckets, err := d.client.ListRestoreS3Buckets(ctx, data.RestoreAccountID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore S3 buckets: %s", err))
		return
	}

	data.Buckets = make([]RestoreS3BucketModel, 0, len(buckets))
	for _, bucket := range buckets {
		data.Buckets = append(data.Buckets, RestoreS3BucketModel{
			BucketName: types.StringValue(bucket.GetBucketName()),
			RegionName: types.StringValue(bucket.GetRegionName()),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
