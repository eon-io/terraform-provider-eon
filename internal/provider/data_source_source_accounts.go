package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SourceAccountsDataSource{}

func NewSourceAccountsDataSource() datasource.DataSource {
	return &SourceAccountsDataSource{}
}

type SourceAccountsDataSource struct {
	client *client.EonClient
}

type SourceAccountsDataSourceModel struct {
	Accounts []SourceAccountModel `tfsdk:"accounts"`
}

type SourceAccountModel struct {
	Id                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Provider          types.String `tfsdk:"provider"`
	ProviderAccountId types.String `tfsdk:"provider_account_id"`
	Status            types.String `tfsdk:"status"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func (d *SourceAccountsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_accounts"
}

func (d *SourceAccountsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of source accounts for the Eon project.",
		Attributes: map[string]schema.Attribute{
			"accounts": schema.ListNestedAttribute{
				MarkdownDescription: "List of source accounts.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Eon-assigned account ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Account display name in Eon.",
							Computed:            true,
						},
						"provider_account_id": schema.StringAttribute{
							MarkdownDescription: "Cloud-provider-assigned account ID.",
							Computed:            true,
						},
						"provider": schema.StringAttribute{
							MarkdownDescription: "Cloud provider. Possible values: `AWS`, `AZURE`, `GCP`.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "Connection status of the AWS account, Azure subscription, or GCP project. Only `CONNECTED` source accounts can be backed up. Possible values: `CONNECTED`, `DISCONNECTED`, `INSUFFICIENT_PERMISSIONS`.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "Date and time the source account was connected to the Eon project.",
							Computed:            true,
						},
						"updated_at": schema.StringAttribute{
							MarkdownDescription: "Date and time the source account was last updated.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *SourceAccountsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SourceAccountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SourceAccountsDataSourceModel

	accounts, err := d.client.ListSourceAccounts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read source accounts: %s", err))
		return
	}

	for _, account := range accounts {
		accountModel := SourceAccountModel{
			Id:                types.StringValue(account.Id),
			Name:              types.StringValue(account.Name),
			ProviderAccountId: types.StringValue(account.ProviderAccountId),
			Status:            types.StringValue(string(account.Status)),
			CreatedAt:         types.StringNull(),
			UpdatedAt:         types.StringNull(),
		}

		if account.SourceAccountAttributes.HasCloudProvider() {
			accountModel.Provider = types.StringValue(string(account.SourceAccountAttributes.GetCloudProvider()))
		} else {
			accountModel.Provider = types.StringNull()
		}

		data.Accounts = append(data.Accounts, accountModel)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
