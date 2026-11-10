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

var _ datasource.DataSource = &RestoreAccountRegionEncryptionKeysDataSource{}

func NewRestoreAccountRegionEncryptionKeysDataSource() datasource.DataSource {
	return &RestoreAccountRegionEncryptionKeysDataSource{}
}

type RestoreAccountRegionEncryptionKeysDataSource struct {
	client *client.EonClient
}

type RestoreAccountRegionEncryptionKeysDataSourceModel struct {
	AccountId             types.String                             `tfsdk:"account_id"`
	Region                types.String                             `tfsdk:"region"`
	IncludeAwsManagedKeys types.Bool                               `tfsdk:"include_aws_managed_keys"`
	Keys                  []RestoreAccountRegionEncryptionKeyModel `tfsdk:"keys"`
}

type RestoreAccountRegionEncryptionKeyModel struct {
	Id        types.String `tfsdk:"id"`
	Arn       types.String `tfsdk:"arn"`
	AliasName types.String `tfsdk:"alias_name"`
}

func (d *RestoreAccountRegionEncryptionKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_account_region_encryption_keys"
}

func (d *RestoreAccountRegionEncryptionKeysDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the encryption keys available to Eon in a region of a restore account. Use this data source to select values for `eon_restore_job` encryption settings, such as `encryption_key_id`.",
		Attributes: map[string]schema.Attribute{
			"account_id": schema.StringAttribute{
				MarkdownDescription: "Eon-assigned ID of the restore account.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Region where the encryption keys are located.",
				Required:            true,
			},
			"include_aws_managed_keys": schema.BoolAttribute{
				MarkdownDescription: "Whether to include AWS-managed keys in the response.",
				Optional:            true,
			},
			"keys": schema.ListNestedAttribute{
				MarkdownDescription: "List of encryption keys available in the region.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Cloud-provider-assigned ID of the encryption key.",
							Computed:            true,
						},
						"arn": schema.StringAttribute{
							MarkdownDescription: "ARN of the encryption key.",
							Computed:            true,
						},
						"alias_name": schema.StringAttribute{
							MarkdownDescription: "Encryption key alias.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RestoreAccountRegionEncryptionKeysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RestoreAccountRegionEncryptionKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreAccountRegionEncryptionKeysDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	opts := client.ListEncryptionKeysOptions{}
	if !data.IncludeAwsManagedKeys.IsNull() && !data.IncludeAwsManagedKeys.IsUnknown() {
		includeAwsManagedKeys := data.IncludeAwsManagedKeys.ValueBool()
		opts.IncludeAwsManagedKeys = &includeAwsManagedKeys
	}

	tflog.Debug(ctx, "Listing restore account region encryption keys", map[string]interface{}{
		"account_id": data.AccountId.ValueString(),
		"region":     data.Region.ValueString(),
	})

	result, err := d.client.ListEncryptionKeys(ctx, data.AccountId.ValueString(), data.Region.ValueString(), opts)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read encryption keys: %s", err))
		return
	}

	data.Keys = restoreAccountRegionEncryptionKeysToModel(result.GetKeys())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func restoreAccountRegionEncryptionKeysToModel(keys []externalEonSdkAPI.EncryptionKey) []RestoreAccountRegionEncryptionKeyModel {
	models := make([]RestoreAccountRegionEncryptionKeyModel, 0, len(keys))
	for _, key := range keys {
		models = append(models, RestoreAccountRegionEncryptionKeyModel{
			Id:        types.StringValue(key.GetId()),
			Arn:       types.StringValue(key.GetArn()),
			AliasName: types.StringValue(key.GetAliasName()),
		})
	}
	return models
}
