package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RestoreAccountsDataSource{}

func NewRestoreAccountsDataSource() datasource.DataSource {
	return &RestoreAccountsDataSource{}
}

type RestoreAccountsDataSource struct {
	client *client.EonClient
}

type RestoreAccountsDataSourceModel struct {
	Accounts []RestoreAccountModel `tfsdk:"accounts"`
}

type RestoreAccountModel struct {
	Id                types.String `tfsdk:"id"`
	Provider          types.String `tfsdk:"provider"`
	ProviderAccountId types.String `tfsdk:"provider_account_id"`
	Status            types.String `tfsdk:"status"`
	Regions           types.List   `tfsdk:"regions"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func (d *RestoreAccountsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_restore_accounts"
}

func (d *RestoreAccountsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of restore accounts for the Eon project.",
		Attributes: map[string]schema.Attribute{
			"accounts": schema.ListNestedAttribute{
				MarkdownDescription: "List of restore accounts.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Eon-assigned restore account ID.",
							Computed:            true,
						},
						"provider": schema.StringAttribute{
							MarkdownDescription: "Cloud provider. Possible values: `AWS`, `AZURE`, `GCP`.",
							Computed:            true,
						},
						"provider_account_id": schema.StringAttribute{
							MarkdownDescription: "Cloud-provider-assigned account ID.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "Connection status of the AWS account, Azure subscription, or GCP project. Only `CONNECTED` restore accounts can be restored to. For an explanation of statuses, see [Restore Account Statuses](/docs/user-guide/restoring/connect-restore-accounts/restore-account-statuses). Possible values: `CONNECTED`, `DISCONNECTED`, `INSUFFICIENT_PERMISSIONS`.",
							Computed:            true,
						},
						"regions": schema.ListAttribute{
							MarkdownDescription: "List of regions associated with the restore account.",
							Computed:            true,
							ElementType:         types.StringType,
						},
						"created_at": schema.StringAttribute{
							MarkdownDescription: "Date and time the restore account was connected to the Eon project.",
							Computed:            true,
						},
						"updated_at": schema.StringAttribute{
							MarkdownDescription: "Date and time the restore account was last updated.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *RestoreAccountsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RestoreAccountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RestoreAccountsDataSourceModel

	accounts, err := d.client.ListRestoreAccounts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read restore accounts: %s", err))
		return
	}

	for _, account := range accounts {
		accountModel := RestoreAccountModel{
			Id:                types.StringValue(account.Id),
			ProviderAccountId: types.StringValue(account.ProviderAccountId),
			Status:            types.StringValue(string(account.Status)),
			Regions:           types.ListNull(types.StringType),
			CreatedAt:         types.StringNull(),
			UpdatedAt:         types.StringNull(),
		}

		if account.RestoreAccountAttributes.HasCloudProvider() {
			accountModel.Provider = types.StringValue(string(account.RestoreAccountAttributes.GetCloudProvider()))
		} else {
			accountModel.Provider = types.StringNull()
		}

		data.Accounts = append(data.Accounts, accountModel)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
