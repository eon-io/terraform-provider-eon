package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &IdpsDataSource{}

func NewIdpsDataSource() datasource.DataSource {
	return &IdpsDataSource{}
}

type IdpsDataSource struct {
	client *client.EonClient
}

type IdpsDataSourceModel struct {
	Idps []IdpModel `tfsdk:"idps"`
}

type IdpModel struct {
	Id           types.String `tfsdk:"id"`
	ProviderName types.String `tfsdk:"provider_name"`
}

func (d *IdpsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_idps"
}

func (d *IdpsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of identity providers configured in the Eon account. An IdP ID is required input for `eon_idp_group` role assignments.",
		Attributes: map[string]schema.Attribute{
			"idps": schema.ListNestedAttribute{
				MarkdownDescription: "List of identity providers.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Eon-assigned identity provider ID.",
							Computed:            true,
						},
						"provider_name": schema.StringAttribute{
							MarkdownDescription: "Identity provider display name.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *IdpsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IdpsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IdpsDataSourceModel
	data.Idps = []IdpModel{}

	idps, err := d.client.ListIdps(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list identity providers: %s", err))
		return
	}

	for _, idp := range idps {
		data.Idps = append(data.Idps, IdpModel{
			Id:           types.StringValue(idp.GetId()),
			ProviderName: types.StringValue(idp.GetProviderName()),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
