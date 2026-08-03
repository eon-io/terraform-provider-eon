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

var _ datasource.DataSource = &PermissionsDataSource{}

func NewPermissionsDataSource() datasource.DataSource {
	return &PermissionsDataSource{}
}

type PermissionsDataSource struct {
	client *client.EonClient
}

type PermissionsDataSourceModel struct {
	Permissions []PermissionModel `tfsdk:"permissions"`
}

type PermissionModel struct {
	PermissionType  types.String `tfsdk:"permission_type"`
	Description     types.String `tfsdk:"description"`
	AllowConditions types.Bool   `tfsdk:"allow_conditions"`
}

func (d *PermissionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permissions"
}

func (d *PermissionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves the catalog of permissions available for composing custom role permission lists. Useful as input when creating `eon_role` resources.",
		Attributes: map[string]schema.Attribute{
			"permissions": schema.ListNestedAttribute{
				MarkdownDescription: "List of available permissions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"permission_type": schema.StringAttribute{
							MarkdownDescription: "Permission identifier (for example `inventory.view`).",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Description of the actions the permission allows.",
							Computed:            true,
						},
						"allow_conditions": schema.BoolAttribute{
							MarkdownDescription: "Whether the permission can be restricted with access conditions when used in a custom role.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *PermissionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PermissionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PermissionsDataSourceModel

	tflog.Debug(ctx, "Listing permissions")

	permissions, err := d.client.ListPermissions(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list permissions: %s", err))
		return
	}

	for _, permission := range permissions {
		data.Permissions = append(data.Permissions, PermissionModel{
			PermissionType:  types.StringValue(string(permission.GetPermissionType())),
			Description:     types.StringValue(permission.GetDescription()),
			AllowConditions: types.BoolValue(permission.GetAllowConditions()),
		})
	}

	tflog.Debug(ctx, "Listed permissions", map[string]interface{}{
		"count": len(data.Permissions),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
