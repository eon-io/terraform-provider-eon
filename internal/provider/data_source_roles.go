package provider

import (
	"context"
	"fmt"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RolesDataSource{}

func NewRolesDataSource() datasource.DataSource {
	return &RolesDataSource{}
}

type RolesDataSource struct {
	client *client.EonClient
}

type RolesDataSourceModel struct {
	Roles []RoleModel `tfsdk:"roles"`
}

type RoleModel struct {
	Id                       types.String `tfsdk:"id"`
	Name                     types.String `tfsdk:"name"`
	IsBuiltInRole            types.Bool   `tfsdk:"is_built_in_role"`
	PermissionGrants         types.List   `tfsdk:"permission_grants"`
	AccessConditions         types.List   `tfsdk:"access_conditions"`
	RestoreDestinationLimits types.Object `tfsdk:"restore_destination_limits"`
}

type PermissionGrantModel struct {
	Permission        types.String `tfsdk:"permission"`
	AccessConditionId types.String `tfsdk:"access_condition_id"`
}

var permissionGrantAttrTypes = map[string]attr.Type{
	"permission":          types.StringType,
	"access_condition_id": types.StringType,
}

func (d *RolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

func (d *RolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a list of roles in the Eon account, including built-in and custom roles.",
		Attributes: map[string]schema.Attribute{
			"roles": schema.ListNestedAttribute{
				MarkdownDescription: "List of roles.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "System-generated unique identifier for the role.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Display name of the role.",
							Computed:            true,
						},
						"is_built_in_role": schema.BoolAttribute{
							MarkdownDescription: "Whether the role is a built-in role provided by Eon (true) or a custom role (false). Built-in roles cannot be modified or deleted.",
							Computed:            true,
						},
						"permission_grants": schema.ListNestedAttribute{
							MarkdownDescription: "List of permissions granted by the role.",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"permission": schema.StringAttribute{
										MarkdownDescription: "The permission identifier (e.g. vaults.manage, dashboard.view).",
										Computed:            true,
									},
									"access_condition_id": schema.StringAttribute{
										MarkdownDescription: "Optional ID of the access condition that restricts the resources this permission applies to.",
										Computed:            true,
										Optional:            true,
									},
								},
							},
						},
						"access_conditions": schema.ListNestedAttribute{
							MarkdownDescription: "Optional list of access conditions that can be referenced by permission_grants to restrict the scope of permissions.",
							Computed:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										MarkdownDescription: "Unique identifier for this access condition, used in permission_grants.access_condition_id.",
										Computed:            true,
									},
									"effect": schema.StringAttribute{
										MarkdownDescription: "Effect of the condition (e.g. ALLOW, DENY).",
										Computed:            true,
									},
									"expression": schema.SingleNestedAttribute{
										MarkdownDescription: RoleExprDescExpression,
										Computed:            true,
										Optional:            true,
										Attributes:          roleAccessConditionExpressionSchemaForDataSource(),
									},
								},
							},
						},
						"restore_destination_limits": schema.SingleNestedAttribute{
							MarkdownDescription: "Optional limits on which restore destination accounts are allowed or denied for this role.",
							Computed:            true,
							Optional:            true,
							Attributes: map[string]schema.Attribute{
								"effect": schema.StringAttribute{
									MarkdownDescription: "Effect of the limit (e.g. ALLOW, DENY).",
									Computed:            true,
								},
								"restore_account_provider_ids": schema.ListAttribute{
									MarkdownDescription: "List of restore account provider IDs to match against.",
									Computed:            true,
									ElementType:         types.StringType,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *RolesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RolesDataSourceModel

	roles, err := d.client.ListRoles(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list roles: %s", err))
		return
	}

	for _, r := range roles {
		permGrantsVal, diags := permissionGrantsFromSDK(ctx, r.GetPermissionGrants())
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		accessCondsVal, diags := flattenRoleAccessConditions(ctx, r.GetAccessConditions())
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		// When the list API returns no access_conditions but permission_grants reference condition IDs,
		// use placeholders so the data source output shows those IDs (e.g. "No PII") in access_conditions.
		grants := r.GetPermissionGrants()
		acLen := 0
		if !accessCondsVal.IsNull() {
			acLen = len(accessCondsVal.Elements())
		}
		if acLen == 0 && len(grants) > 0 {
			placeholders, d2 := accessConditionPlaceholdersFromGrants(ctx, grants)
			if !d2.HasError() && !placeholders.IsNull() {
				accessCondsVal = placeholders
			}
		}

		rdlVal := types.ObjectNull(restoreDestinationLimitsAttrTypes)
		if r.HasRestoreDestinationLimits() {
			rdl := r.GetRestoreDestinationLimits()
			v, d := flattenRestoreDestinationLimits(rdl)
			if d.HasError() {
				resp.Diagnostics.Append(d...)
				return
			}
			rdlVal = v
		}

		data.Roles = append(data.Roles, RoleModel{
			Id:                       types.StringValue(r.GetId()),
			Name:                     types.StringValue(r.GetName()),
			IsBuiltInRole:            types.BoolValue(r.GetIsBuiltInRole()),
			PermissionGrants:         permGrantsVal,
			AccessConditions:         accessCondsVal,
			RestoreDestinationLimits: rdlVal,
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// permissionGrantsFromSDK converts SDK PermissionGrant slice to a types.List of objects.
func permissionGrantsFromSDK(ctx context.Context, grants []externalEonSdkAPI.PermissionGrant) (types.List, diag.Diagnostics) {
	objectType := types.ObjectType{AttrTypes: permissionGrantAttrTypes}
	if grants == nil {
		return types.ListValueFrom(ctx, objectType, []attr.Value{})
	}
	elems := make([]attr.Value, 0, len(grants))
	for _, g := range grants {
		acId := types.StringNull()
		if g.AccessConditionId != nil && *g.AccessConditionId != "" {
			acId = types.StringValue(*g.AccessConditionId)
		}
		obj, d := types.ObjectValue(permissionGrantAttrTypes, map[string]attr.Value{
			"permission":          types.StringValue(string(g.GetPermission())),
			"access_condition_id": acId,
		})
		if d.HasError() {
			return types.ListNull(objectType), d
		}
		elems = append(elems, obj)
	}
	l, d := types.ListValue(objectType, elems)
	return l, d
}
