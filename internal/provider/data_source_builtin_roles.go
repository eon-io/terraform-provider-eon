package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Built-in role UUIDs from eon-service shared/roles/roles.go (customer-facing only).
// Internal roles (eon_admin, eon_pulumi_config) are not exposed.
var builtinRoleIDs = map[string]string{
	"global_admin":  "379a1104-838a-4bf3-af96-da3af27c5712",
	"global_viewer": "543bad56-e9b2-421f-8456-b43c53fcebfe",
	"viewer":        "d6afa067-d3a0-457e-923d-27cd26c9e5cb",
	"admin":         "a675e456-8602-4550-9c65-66583404e0d6",
	"operator":      "21d0ae2b-9bbc-4a41-bd5e-98011e9f10a5",
}

var _ datasource.DataSource = &BuiltinRolesDataSource{}

func NewBuiltinRolesDataSource() datasource.DataSource {
	return &BuiltinRolesDataSource{}
}

type BuiltinRolesDataSource struct{}

type BuiltinRolesDataSourceModel struct {
	GlobalAdmin  types.String `tfsdk:"global_admin"`
	GlobalViewer types.String `tfsdk:"global_viewer"`
	Viewer       types.String `tfsdk:"viewer"`
	Admin        types.String `tfsdk:"admin"`
	Operator     types.String `tfsdk:"operator"`
}

func (d *BuiltinRolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_builtin_roles"
}

func (d *BuiltinRolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides EON built-in role UUIDs as flat attributes. Use in `eon_idp_group.role_ids` instead of hardcoding UUIDs (e.g. `data.eon_builtin_roles.builtin.global_admin`). Only customer-facing built-in roles are included; internal roles are not exposed.",
		Attributes: map[string]schema.Attribute{
			"global_admin": schema.StringAttribute{
				MarkdownDescription: "Built-in Global Admin role UUID.",
				Computed:            true,
			},
			"global_viewer": schema.StringAttribute{
				MarkdownDescription: "Built-in Global Viewer role UUID.",
				Computed:            true,
			},
			"viewer": schema.StringAttribute{
				MarkdownDescription: "Built-in Viewer role UUID.",
				Computed:            true,
			},
			"admin": schema.StringAttribute{
				MarkdownDescription: "Built-in Admin role UUID.",
				Computed:            true,
			},
			"operator": schema.StringAttribute{
				MarkdownDescription: "Built-in Operator role UUID.",
				Computed:            true,
			},
		},
	}
}

func (d *BuiltinRolesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// No client needed; data is static.
}

func (d *BuiltinRolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	data := BuiltinRolesDataSourceModel{
		GlobalAdmin:  types.StringValue(builtinRoleIDs["global_admin"]),
		GlobalViewer: types.StringValue(builtinRoleIDs["global_viewer"]),
		Viewer:       types.StringValue(builtinRoleIDs["viewer"]),
		Admin:        types.StringValue(builtinRoleIDs["admin"]),
		Operator:     types.StringValue(builtinRoleIDs["operator"]),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
