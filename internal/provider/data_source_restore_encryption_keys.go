package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RestoreEncryptionKeysDataSource{}

func NewRestoreEncryptionKeysDataSource() datasource.DataSource {
	return &RestoreEncryptionKeysDataSource{}
}

type RestoreEncryptionKeysDataSource struct {
	client *client.EonClient
}

type RestoreEncryptionKeysDataSourceModel struct {
	RestoreAccountID      types.String                `tfsdk:"restore_account_id"`
	Region                types.String                `tfsdk:"region"`
	IncludeAwsManagedKeys types.Bool                  `tfsdk:"include_aws_managed_keys"`
	Keys                  []RestoreEncryptionKeyModel `tfsdk:"keys"`
}

type RestoreEncryptionKeyModel struct {
	Id        types.String `tfsdk:"id"`
	Arn       types.String `tfsdk:"arn"`
	AliasName types.String `tfsdk:"alias_name"`
}

func (d *RestoreEncryptionKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_encryption_keys"
}

func (d *RestoreEncryptionKeysDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists encryption keys available to a restore account in a region.",
		Attributes: map[string]schema.Attribute{
			"restore_account_id": schema.StringAttribute{
				MarkdownDescription: "Eon restore account ID.",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS region to list encryption keys in.",
				Required:            true,
			},
			"include_aws_managed_keys": schema.BoolAttribute{
				MarkdownDescription: "Whether to include AWS-managed encryption keys.",
				Optional:            true,
			},
			"keys": schema.ListNestedAttribute{
				MarkdownDescription: "Encryption keys available in the region.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Encryption key ID.",
							Computed:            true,
						},
						"arn": schema.StringAttribute{
							MarkdownDescription: "Encryption key ARN.",
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

func (d *RestoreEncryptionKeysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RestoreEncryptionKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreEncryptionKeysDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var includeAwsManagedKeys *bool
	if !data.IncludeAwsManagedKeys.IsNull() && !data.IncludeAwsManagedKeys.IsUnknown() {
		value := data.IncludeAwsManagedKeys.ValueBool()
		includeAwsManagedKeys = &value
	}

	keys, err := d.client.ListRestoreEncryptionKeys(
		ctx,
		data.RestoreAccountID.ValueString(),
		data.Region.ValueString(),
		includeAwsManagedKeys,
	)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore encryption keys: %s", err))
		return
	}

	data.Keys = make([]RestoreEncryptionKeyModel, 0, len(keys))
	for _, key := range keys {
		data.Keys = append(data.Keys, RestoreEncryptionKeyModel{
			Id:        types.StringValue(key.GetId()),
			Arn:       types.StringValue(key.GetArn()),
			AliasName: types.StringValue(key.GetAliasName()),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
