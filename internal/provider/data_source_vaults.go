package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &VaultsDataSource{}

func NewVaultsDataSource() datasource.DataSource {
	return &VaultsDataSource{}
}

type VaultsDataSource struct {
	client *client.EonClient
}

type VaultsDataSourceModel struct {
	Vaults []VaultModel `tfsdk:"vaults"`
}

type VaultModel struct {
	Id                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Region            types.String `tfsdk:"region"`
	CloudProvider     types.String `tfsdk:"cloud_provider"`
	VaultAccountId    types.String `tfsdk:"vault_account_id"`
	ProviderAccountId types.String `tfsdk:"provider_account_id"`
	IsManagedByEon    types.Bool   `tfsdk:"is_managed_by_eon"`
	AwsKmsKeyArn      types.String `tfsdk:"aws_kms_key_arn"`
}

func (d *VaultsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vaults"
}

func (d *VaultsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of vaults in the Eon project.",
		Attributes: map[string]schema.Attribute{
			"vaults": schema.ListNestedAttribute{
				MarkdownDescription: "List of vaults.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Vault identifier (UUID).",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Vault display name.",
							Computed:            true,
						},
						"region": schema.StringAttribute{
							MarkdownDescription: "Region where the vault is hosted.",
							Computed:            true,
						},
						"cloud_provider": schema.StringAttribute{
							MarkdownDescription: "Cloud provider (AWS, AZURE, GCP).",
							Computed:            true,
						},
						"vault_account_id": schema.StringAttribute{
							MarkdownDescription: "Eon-assigned ID of the vault account.",
							Computed:            true,
						},
						"provider_account_id": schema.StringAttribute{
							MarkdownDescription: "Cloud provider-assigned ID of the vault account.",
							Computed:            true,
						},
						"is_managed_by_eon": schema.BoolAttribute{
							MarkdownDescription: "Whether the vault is in an Eon-managed vault account.",
							Computed:            true,
						},
						"aws_kms_key_arn": schema.StringAttribute{
							MarkdownDescription: "ARN of the AWS customer-managed KMS key for encryption. Empty if using Eon-managed encryption.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *VaultsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.EonClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.EonClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *VaultsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data VaultsDataSourceModel

	vaults, err := d.client.ListVaults(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read vaults: %s", err))
		return
	}

	for _, vault := range vaults {
		// Extract AWS KMS key if present
		var awsKmsKeyArn types.String
		if vault.VaultAttributes.Aws.IsSet() {
			awsConfig := vault.VaultAttributes.Aws.Get()
			if awsConfig.EncryptionKey != nil {
				awsKmsKeyArn = types.StringValue(*awsConfig.EncryptionKey)
			} else {
				awsKmsKeyArn = types.StringNull()
			}
		} else {
			awsKmsKeyArn = types.StringNull()
		}

		vaultModel := VaultModel{
			Id:                types.StringValue(vault.Id),
			Name:              types.StringValue(vault.Name),
			Region:            types.StringValue(vault.Region),
			CloudProvider:     types.StringValue(string(vault.VaultAttributes.CloudProvider)),
			VaultAccountId:    types.StringValue(vault.VaultAccountId),
			ProviderAccountId: types.StringValue(vault.ProviderAccountId),
			IsManagedByEon:    types.BoolValue(vault.IsManagedByEon),
			AwsKmsKeyArn:      awsKmsKeyArn,
		}

		data.Vaults = append(data.Vaults, vaultModel)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
