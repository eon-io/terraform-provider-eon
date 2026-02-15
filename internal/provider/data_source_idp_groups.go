package provider

import (
	"context"
	"fmt"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &IdpGroupsDataSource{}

func NewIdpGroupsDataSource() datasource.DataSource {
	return &IdpGroupsDataSource{}
}

type IdpGroupsDataSource struct {
	client *client.EonClient
}

type IdpGroupsDataSourceModel struct {
	Groups []IdpGroupModel `tfsdk:"groups"`
}

type IdpGroupModel struct {
	Id              types.String `tfsdk:"id"`
	IdpId           types.String `tfsdk:"idp_id"`
	ProviderGroupId types.String `tfsdk:"provider_group_id"`
	RoleIds         types.List   `tfsdk:"role_ids"`
}

func (d *IdpGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_idp_groups"
}

func (d *IdpGroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of IDP (Identity Provider) groups and their role assignments in the Eon account.",
		Attributes: map[string]schema.Attribute{
			"groups": schema.ListNestedAttribute{
				MarkdownDescription: "List of IDP groups.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "System-generated unique identifier for the IDP group.",
							Computed:            true,
						},
						"idp_id": schema.StringAttribute{
							MarkdownDescription: "The ID of the Identity Provider this group belongs to.",
							Computed:            true,
						},
						"provider_group_id": schema.StringAttribute{
							MarkdownDescription: "The group identifier from the Identity Provider.",
							Computed:            true,
						},
						"role_ids": schema.ListAttribute{
							ElementType:         types.StringType,
							MarkdownDescription: "List of Eon role IDs assigned to this IDP group.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *IdpGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *IdpGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IdpGroupsDataSourceModel

	groups, err := d.client.ListIdpGroups(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list IDP groups: %s", err))
		return
	}

	for _, g := range groups {
		roleIdsVal, diags := types.ListValueFrom(ctx, types.StringType, stringSliceToInterface(g.GetRoleIds()))
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		data.Groups = append(data.Groups, IdpGroupModel{
			Id:              types.StringValue(g.GetId()),
			IdpId:           types.StringValue(g.GetIdpId()),
			ProviderGroupId: types.StringValue(g.GetProviderGroupId()),
			RoleIds:         roleIdsVal,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// stringSliceToInterface converts []string to []interface{} for use with types.ListValueFrom.
func stringSliceToInterface(s []string) []interface{} {
	if s == nil {
		return nil
	}
	out := make([]interface{}, len(s))
	for i, v := range s {
		out[i] = v
	}
	return out
}
