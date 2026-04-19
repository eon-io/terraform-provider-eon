package provider

import (
	"context"
	"fmt"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// roleAccessConditionOperandsSchema returns the schema for group.operands (each operand is one condition type).
func roleAccessConditionOperandsSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"environment": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescEnvironment,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":     schema.StringAttribute{Required: true},
				"environments": schema.ListAttribute{ElementType: types.StringType, Required: true},
			},
		},
		"resource_type": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceType,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":       schema.StringAttribute{Required: true},
				"resource_types": schema.ListAttribute{ElementType: types.StringType, Required: true},
			},
		},
		"data_classes": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescDataClasses,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":     schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorContains, Required: true},
				"data_classes": schema.ListAttribute{ElementType: types.StringType, Required: true},
			},
		},
		"tag_keys": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescTagKeys,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Required: true},
				"tag_keys": schema.ListAttribute{ElementType: types.StringType, Required: true},
			},
		},
		"tag_key_values": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescTagKeyValues,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Required: true},
				"tag_key_values": schema.ListNestedAttribute{
					Required: true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"key":   schema.StringAttribute{Required: true},
							"value": schema.StringAttribute{Optional: true},
						},
					},
				},
			},
		},
		"apps": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescApps,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Required: true},
				"apps":     schema.ListAttribute{ElementType: types.StringType, Required: true},
			},
		},
		"cloud_provider": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescCloudProvider,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":        schema.StringAttribute{Required: true},
				"cloud_providers": schema.ListAttribute{ElementType: types.StringType, Required: true},
			},
		},
		"account_id": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescAccountID,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":    schema.StringAttribute{Required: true},
				"account_ids": schema.ListAttribute{ElementType: types.StringType, Required: true},
			},
		},
		"source_region": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescSourceRegion,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":       schema.StringAttribute{Required: true},
				"source_regions": schema.ListAttribute{ElementType: types.StringType, Required: true},
			},
		},
		"vpc": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescVPC,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Required: true},
				"vpcs":     schema.ListAttribute{ElementType: types.StringType, Required: true},
			},
		},
		"subnets": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescSubnets,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Required: true},
				"subnets":  schema.ListAttribute{ElementType: types.StringType, Required: true},
			},
		},
		"resource_group_name": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceGroupName,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":             schema.StringAttribute{Required: true},
				"resource_group_names": schema.ListAttribute{ElementType: types.StringType, Required: true},
			},
		},
		"resource_name": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceName,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":       schema.StringAttribute{Required: true},
				"resource_names": schema.ListAttribute{ElementType: types.StringType, Required: true},
			},
		},
		"resource_id": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceID,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":     schema.StringAttribute{Required: true},
				"resource_ids": schema.ListAttribute{ElementType: types.StringType, Required: true},
			},
		},
	}
}

// roleAccessConditionExpressionSchema returns the schema for access_conditions[].expression.
// Matches the same nested structure as backup policy resource_selector.expression
// (environment, resource_type, group, data_classes, tag_keys, tag_key_values, etc.).
func roleAccessConditionExpressionSchema() map[string]schema.Attribute {
	operandsAttr := roleAccessConditionOperandsSchema()
	return map[string]schema.Attribute{
		"environment": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescEnvironment,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":     schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorINorNOTIN, Required: true},
				"environments": schema.ListAttribute{MarkdownDescription: RoleExprDescListEnvironments, ElementType: types.StringType, Required: true},
			},
		},
		"resource_type": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceType,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":       schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorINorNOTIN, Required: true},
				"resource_types": schema.ListAttribute{MarkdownDescription: RoleExprDescListResourceTypes, ElementType: types.StringType, Required: true},
			},
		},
		"data_classes": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescDataClasses,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":     schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorContains, Required: true},
				"data_classes": schema.ListAttribute{MarkdownDescription: RoleExprDescListDataClasses, ElementType: types.StringType, Required: true},
			},
		},
		"tag_key_values": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescTagKeyValues,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorContains, Required: true},
				"tag_key_values": schema.ListNestedAttribute{
					MarkdownDescription: RoleExprDescListTagKeyValuesMatch,
					Required:            true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"key":   schema.StringAttribute{MarkdownDescription: RoleExprDescTagKey, Required: true},
							"value": schema.StringAttribute{MarkdownDescription: RoleExprDescTagValue, Optional: true},
						},
					},
				},
			},
		},
		"tag_keys": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescTagKeys,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorContains, Required: true},
				"tag_keys": schema.ListAttribute{MarkdownDescription: RoleExprDescListTagKeysMatch, ElementType: types.StringType, Required: true},
			},
		},
		"apps": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescApps,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{Required: true},
				"apps":     schema.ListAttribute{MarkdownDescription: RoleExprDescListApps, ElementType: types.StringType, Required: true},
			},
		},
		"cloud_provider": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescCloudProvider,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":        schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorINorNOTIN, Required: true},
				"cloud_providers": schema.ListAttribute{MarkdownDescription: RoleExprDescListCloudProviders, ElementType: types.StringType, Required: true},
			},
		},
		"account_id": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescAccountID,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":    schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorINorNOTIN, Required: true},
				"account_ids": schema.ListAttribute{MarkdownDescription: RoleExprDescListAccountIDs, ElementType: types.StringType, Required: true},
			},
		},
		"source_region": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescSourceRegion,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":       schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorINorNOTIN, Required: true},
				"source_regions": schema.ListAttribute{MarkdownDescription: RoleExprDescListSourceRegions, ElementType: types.StringType, Required: true},
			},
		},
		"vpc": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescVPC,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorINorNOTIN, Required: true},
				"vpcs":     schema.ListAttribute{MarkdownDescription: RoleExprDescListVPCs, ElementType: types.StringType, Required: true},
			},
		},
		"subnets": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescSubnets,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorContains, Required: true},
				"subnets":  schema.ListAttribute{MarkdownDescription: RoleExprDescListSubnets, ElementType: types.StringType, Required: true},
			},
		},
		"resource_group_name": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceGroupName,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":             schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorINorNOTIN, Required: true},
				"resource_group_names": schema.ListAttribute{MarkdownDescription: RoleExprDescListResourceGroupNames, ElementType: types.StringType, Required: true},
			},
		},
		"resource_name": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceName,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":       schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorINorNOTIN, Required: true},
				"resource_names": schema.ListAttribute{MarkdownDescription: RoleExprDescListResourceNames, ElementType: types.StringType, Required: true},
			},
		},
		"resource_id": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescResourceID,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator":     schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorINorNOTIN, Required: true},
				"resource_ids": schema.ListAttribute{MarkdownDescription: RoleExprDescListResourceIDs, ElementType: types.StringType, Required: true},
			},
		},
		"group": schema.SingleNestedAttribute{
			MarkdownDescription: RoleExprDescGroup,
			Optional:            true,
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{MarkdownDescription: RoleExprDescOperatorLogical, Required: true},
				"operands": schema.ListNestedAttribute{
					MarkdownDescription: RoleExprDescListOperands,
					Optional:            true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: operandsAttr,
					},
				},
			},
		},
	}
}

// roleAccessConditionsToSDK converts Terraform access_conditions list to SDK []AccessCondition.
func roleAccessConditionsToSDK(ctx context.Context, list types.List) ([]externalEonSdkAPI.AccessCondition, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var models []RoleAccessConditionModel
	diags := list.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]externalEonSdkAPI.AccessCondition, 0, len(models))
	for _, m := range models {
		var expr externalEonSdkAPI.NullableAccessConditionalExpression
		if !m.Expression.IsNull() && !m.Expression.IsUnknown() {
			sdkExpr, d := createRoleAccessConditionalExpression(ctx, m.Expression)
			if d.HasError() {
				return nil, d
			}
			if sdkExpr != nil {
				expr = *externalEonSdkAPI.NewNullableAccessConditionalExpression(sdkExpr)
			}
		}
		// API requires each access condition to have an expression; reject empty expression
		if !expr.IsSet() {
			return nil, diag.Diagnostics{diag.NewErrorDiagnostic(
				"InvalidExpression",
				fmt.Sprintf("access_conditions entry %q must have an expression (e.g. data_classes, environment, resource_type, tag_keys, or group). The API requires each data access rule to have a condition or expression.", m.Id.ValueString()),
			)}
		}
		ac := externalEonSdkAPI.NewAccessCondition(
			m.Id.ValueString(),
			externalEonSdkAPI.AccessConditionEffect(m.Effect.ValueString()),
			expr,
		)
		out = append(out, *ac)
	}
	return out, nil
}

// createRoleAccessConditionalExpression converts Terraform expression object to SDK AccessConditionalExpression.
// Handles environment, resource_type, and group (same pattern as backup policy createBackupPolicyExpression).
func createRoleAccessConditionalExpression(ctx context.Context, expr types.Object) (*externalEonSdkAPI.AccessConditionalExpression, diag.Diagnostics) {
	if expr.IsNull() || expr.IsUnknown() {
		return nil, nil
	}
	attrs := expr.Attributes()
	sdkExpr := externalEonSdkAPI.NewAccessConditionalExpression()

	if envObj, ok := attrs["environment"]; ok && !envObj.IsNull() {
		var envEnv []string
		envAttr := envObj.(types.Object)
		envAttrs := envAttr.Attributes()
		if o, ok := envAttrs["operator"]; ok && !o.IsNull() {
			if e, ok := envAttrs["environments"]; ok && !e.IsNull() {
				_ = e.(types.List).ElementsAs(ctx, &envEnv, false)
			}
		}
		var envEnums []externalEonSdkAPI.Environment
		for _, s := range envEnv {
			envEnums = append(envEnums, externalEonSdkAPI.Environment(s))
		}
		op := externalEonSdkAPI.ScalarOperators(envAttr.Attributes()["operator"].(types.String).ValueString())
		envCond := externalEonSdkAPI.NewEnvironmentCondition(op, envEnums)
		sdkExpr.SetEnvironment(*envCond)
		return sdkExpr, nil
	}

	if rtObj, ok := attrs["resource_type"]; ok && !rtObj.IsNull() {
		var rtList []string
		rtAttr := rtObj.(types.Object)
		rtAttrs := rtAttr.Attributes()
		if e, ok := rtAttrs["resource_types"]; ok && !e.IsNull() {
			_ = e.(types.List).ElementsAs(ctx, &rtList, false)
		}
		var rtEnums []externalEonSdkAPI.ResourceType
		for _, s := range rtList {
			rtEnums = append(rtEnums, externalEonSdkAPI.ResourceType(s))
		}
		op := externalEonSdkAPI.ScalarOperators(rtAttr.Attributes()["operator"].(types.String).ValueString())
		rtCond := externalEonSdkAPI.NewResourceTypeCondition(op, rtEnums)
		sdkExpr.SetResourceType(*rtCond)
		return sdkExpr, nil
	}

	if groupObj, ok := attrs["group"]; ok && !groupObj.IsNull() {
		groupCond, diags := roleGroupConditionToSDK(ctx, groupObj.(types.Object))
		if diags.HasError() {
			return nil, diags
		}
		sdkExpr.SetGroup(*groupCond)
		return sdkExpr, nil
	}

	if dcObj, ok := attrs["data_classes"]; ok && !dcObj.IsNull() {
		sdkCond, d := roleDataClassesConditionToSDK(ctx, dcObj.(types.Object))
		if d.HasError() {
			return nil, d
		}
		sdkExpr.SetDataClasses(*sdkCond)
		return sdkExpr, nil
	}
	if tkObj, ok := attrs["tag_keys"]; ok && !tkObj.IsNull() {
		sdkCond, d := roleTagKeysConditionToSDK(ctx, tkObj.(types.Object))
		if d.HasError() {
			return nil, d
		}
		sdkExpr.SetTagKeys(*sdkCond)
		return sdkExpr, nil
	}
	if tkvObj, ok := attrs["tag_key_values"]; ok && !tkvObj.IsNull() {
		sdkCond, d := roleTagKeyValuesConditionToSDK(ctx, tkvObj.(types.Object))
		if d.HasError() {
			return nil, d
		}
		sdkExpr.SetTagKeyValues(*sdkCond)
		return sdkExpr, nil
	}
	if appsObj, ok := attrs["apps"]; ok && !appsObj.IsNull() {
		sdkCond, d := roleAppsConditionToSDK(ctx, appsObj.(types.Object))
		if d.HasError() {
			return nil, d
		}
		sdkExpr.SetApps(*sdkCond)
		return sdkExpr, nil
	}
	if cpObj, ok := attrs["cloud_provider"]; ok && !cpObj.IsNull() {
		sdkCond, d := roleCloudProviderConditionToSDK(ctx, cpObj.(types.Object))
		if d.HasError() {
			return nil, d
		}
		sdkExpr.SetCloudProvider(*sdkCond)
		return sdkExpr, nil
	}
	if acctObj, ok := attrs["account_id"]; ok && !acctObj.IsNull() {
		sdkCond, d := roleAccountIdConditionToSDK(ctx, acctObj.(types.Object))
		if d.HasError() {
			return nil, d
		}
		sdkExpr.SetAccountId(*sdkCond)
		return sdkExpr, nil
	}
	if regionObj, ok := attrs["source_region"]; ok && !regionObj.IsNull() {
		sdkCond, d := roleSourceRegionConditionToSDK(ctx, regionObj.(types.Object))
		if d.HasError() {
			return nil, d
		}
		sdkExpr.SetSourceRegion(*sdkCond)
		return sdkExpr, nil
	}
	if vpcObj, ok := attrs["vpc"]; ok && !vpcObj.IsNull() {
		sdkCond, d := roleVpcConditionToSDK(ctx, vpcObj.(types.Object))
		if d.HasError() {
			return nil, d
		}
		sdkExpr.SetVpc(*sdkCond)
		return sdkExpr, nil
	}
	if subnetsObj, ok := attrs["subnets"]; ok && !subnetsObj.IsNull() {
		sdkCond, d := roleSubnetsConditionToSDK(ctx, subnetsObj.(types.Object))
		if d.HasError() {
			return nil, d
		}
		sdkExpr.SetSubnets(*sdkCond)
		return sdkExpr, nil
	}
	if rgnObj, ok := attrs["resource_group_name"]; ok && !rgnObj.IsNull() {
		sdkCond, d := roleResourceGroupNameConditionToSDK(ctx, rgnObj.(types.Object))
		if d.HasError() {
			return nil, d
		}
		sdkExpr.SetResourceGroupName(*sdkCond)
		return sdkExpr, nil
	}
	if rnObj, ok := attrs["resource_name"]; ok && !rnObj.IsNull() {
		sdkCond, d := roleResourceNameConditionToSDK(ctx, rnObj.(types.Object))
		if d.HasError() {
			return nil, d
		}
		sdkExpr.SetResourceName(*sdkCond)
		return sdkExpr, nil
	}
	if ridObj, ok := attrs["resource_id"]; ok && !ridObj.IsNull() {
		sdkCond, d := roleResourceIdConditionToSDK(ctx, ridObj.(types.Object))
		if d.HasError() {
			return nil, d
		}
		sdkExpr.SetResourceId(*sdkCond)
		return sdkExpr, nil
	}

	return nil, diag.Diagnostics{diag.NewErrorDiagnostic("InvalidExpression", "access_conditions[].expression must have exactly one of: environment, resource_type, group, data_classes, tag_keys, tag_key_values, apps, cloud_provider, account_id, source_region, vpc, subnets, resource_group_name, resource_name, resource_id")}
}

func roleGroupConditionToSDK(ctx context.Context, group types.Object) (*externalEonSdkAPI.RoleAccessGroupCondition, diag.Diagnostics) {
	attrs := group.Attributes()
	opStr := attrs["operator"].(types.String).ValueString()
	operandsAttr, ok := attrs["operands"]
	if !ok || operandsAttr.IsNull() {
		return nil, diag.Diagnostics{diag.NewErrorDiagnostic("InvalidExpression", "group.operands is required")}
	}
	operandsList := operandsAttr.(types.List)
	operands := make([]externalEonSdkAPI.AccessConditionalExpression, 0, len(operandsList.Elements()))
	for _, elem := range operandsList.Elements() {
		operandObj := elem.(types.Object)
		sdkExpr, d := createRoleAccessConditionalExpression(ctx, operandObj)
		if d.HasError() {
			return nil, d
		}
		if sdkExpr != nil {
			operands = append(operands, *sdkExpr)
		}
	}
	groupCond := externalEonSdkAPI.NewRoleAccessGroupCondition(externalEonSdkAPI.LogicalOperator(opStr), operands)
	return groupCond, nil
}

func roleDataClassesConditionToSDK(ctx context.Context, obj types.Object) (*externalEonSdkAPI.DataClassesCondition, diag.Diagnostics) {
	attrs := obj.Attributes()
	op := externalEonSdkAPI.ListOperators(attrs["operator"].(types.String).ValueString())
	var list []string
	_ = attrs["data_classes"].(types.List).ElementsAs(ctx, &list, false)
	dcs := make([]externalEonSdkAPI.DataClass, len(list))
	for i, s := range list {
		dcs[i] = externalEonSdkAPI.DataClass(s)
	}
	c := externalEonSdkAPI.NewDataClassesCondition(op, dcs)
	return c, nil
}

func roleTagKeysConditionToSDK(ctx context.Context, obj types.Object) (*externalEonSdkAPI.TagKeysCondition, diag.Diagnostics) {
	attrs := obj.Attributes()
	op := externalEonSdkAPI.ListOperators(attrs["operator"].(types.String).ValueString())
	var list []string
	_ = attrs["tag_keys"].(types.List).ElementsAs(ctx, &list, false)
	return externalEonSdkAPI.NewTagKeysCondition(op, list), nil
}

func roleTagKeyValuesConditionToSDK(ctx context.Context, obj types.Object) (*externalEonSdkAPI.TagKeyValuesCondition, diag.Diagnostics) {
	attrs := obj.Attributes()
	op := externalEonSdkAPI.ListOperators(attrs["operator"].(types.String).ValueString())
	var kvs []struct {
		Key   types.String `tfsdk:"key"`
		Value types.String `tfsdk:"value"`
	}
	_ = attrs["tag_key_values"].(types.List).ElementsAs(ctx, &kvs, false)
	out := make([]externalEonSdkAPI.TagKeyValue, 0, len(kvs))
	for _, kv := range kvs {
		t := externalEonSdkAPI.NewTagKeyValue(kv.Key.ValueString())
		if !kv.Value.IsNull() && !kv.Value.IsUnknown() {
			v := kv.Value.ValueString()
			t.SetValue(v)
		}
		out = append(out, *t)
	}
	return externalEonSdkAPI.NewTagKeyValuesCondition(op, out), nil
}

func roleAppsConditionToSDK(ctx context.Context, obj types.Object) (*externalEonSdkAPI.AppsCondition, diag.Diagnostics) {
	attrs := obj.Attributes()
	op := externalEonSdkAPI.ListOperators(attrs["operator"].(types.String).ValueString())
	var list []string
	_ = attrs["apps"].(types.List).ElementsAs(ctx, &list, false)
	return externalEonSdkAPI.NewAppsCondition(op, list), nil
}

func roleCloudProviderConditionToSDK(ctx context.Context, obj types.Object) (*externalEonSdkAPI.CloudProviderCondition, diag.Diagnostics) {
	attrs := obj.Attributes()
	op := externalEonSdkAPI.ScalarOperators(attrs["operator"].(types.String).ValueString())
	var list []string
	_ = attrs["cloud_providers"].(types.List).ElementsAs(ctx, &list, false)
	providers := make([]externalEonSdkAPI.Provider, len(list))
	for i, s := range list {
		providers[i] = externalEonSdkAPI.Provider(s)
	}
	return externalEonSdkAPI.NewCloudProviderCondition(op, providers), nil
}

func roleAccountIdConditionToSDK(ctx context.Context, obj types.Object) (*externalEonSdkAPI.AccountIdCondition, diag.Diagnostics) {
	attrs := obj.Attributes()
	op := externalEonSdkAPI.ScalarOperators(attrs["operator"].(types.String).ValueString())
	var list []string
	_ = attrs["account_ids"].(types.List).ElementsAs(ctx, &list, false)
	return externalEonSdkAPI.NewAccountIdCondition(op, list), nil
}

func roleSourceRegionConditionToSDK(ctx context.Context, obj types.Object) (*externalEonSdkAPI.RegionCondition, diag.Diagnostics) {
	attrs := obj.Attributes()
	op := externalEonSdkAPI.ScalarOperators(attrs["operator"].(types.String).ValueString())
	var list []string
	_ = attrs["source_regions"].(types.List).ElementsAs(ctx, &list, false)
	return externalEonSdkAPI.NewRegionCondition(op, list), nil
}

func roleVpcConditionToSDK(ctx context.Context, obj types.Object) (*externalEonSdkAPI.VpcCondition, diag.Diagnostics) {
	attrs := obj.Attributes()
	op := externalEonSdkAPI.ScalarOperators(attrs["operator"].(types.String).ValueString())
	var list []string
	_ = attrs["vpcs"].(types.List).ElementsAs(ctx, &list, false)
	return externalEonSdkAPI.NewVpcCondition(op, list), nil
}

func roleSubnetsConditionToSDK(ctx context.Context, obj types.Object) (*externalEonSdkAPI.SubnetsCondition, diag.Diagnostics) {
	attrs := obj.Attributes()
	op := externalEonSdkAPI.ListOperators(attrs["operator"].(types.String).ValueString())
	var list []string
	_ = attrs["subnets"].(types.List).ElementsAs(ctx, &list, false)
	return externalEonSdkAPI.NewSubnetsCondition(op, list), nil
}

func roleResourceGroupNameConditionToSDK(ctx context.Context, obj types.Object) (*externalEonSdkAPI.ResourceGroupNameCondition, diag.Diagnostics) {
	attrs := obj.Attributes()
	op := externalEonSdkAPI.ScalarOperators(attrs["operator"].(types.String).ValueString())
	var list []string
	_ = attrs["resource_group_names"].(types.List).ElementsAs(ctx, &list, false)
	return externalEonSdkAPI.NewResourceGroupNameCondition(op, list), nil
}

func roleResourceNameConditionToSDK(ctx context.Context, obj types.Object) (*externalEonSdkAPI.ResourceNameCondition, diag.Diagnostics) {
	attrs := obj.Attributes()
	op := externalEonSdkAPI.ScalarOperators(attrs["operator"].(types.String).ValueString())
	var list []string
	_ = attrs["resource_names"].(types.List).ElementsAs(ctx, &list, false)
	return externalEonSdkAPI.NewResourceNameCondition(op, list), nil
}

func roleResourceIdConditionToSDK(ctx context.Context, obj types.Object) (*externalEonSdkAPI.ResourceIdCondition, diag.Diagnostics) {
	attrs := obj.Attributes()
	op := externalEonSdkAPI.ScalarOperators(attrs["operator"].(types.String).ValueString())
	var list []string
	_ = attrs["resource_ids"].(types.List).ElementsAs(ctx, &list, false)
	return externalEonSdkAPI.NewResourceIdCondition(op, list), nil
}

// restoreDestinationLimitsAttrTypes is the attr type map for restore_destination_limits.
var restoreDestinationLimitsAttrTypes = map[string]attr.Type{
	"effect":                       types.StringType,
	"restore_account_provider_ids": types.ListType{ElemType: types.StringType},
}

// restoreDestinationLimitsToSDK converts a Terraform restore_destination_limits object to the SDK type.
func restoreDestinationLimitsToSDK(ctx context.Context, obj types.Object) (*externalEonSdkAPI.RestoreDestinationLimits, diag.Diagnostics) {
	if obj.IsNull() || obj.IsUnknown() {
		return nil, nil
	}
	attrs := obj.Attributes()
	effect := externalEonSdkAPI.AccessConditionEffect(attrs["effect"].(types.String).ValueString())
	var ids []string
	if d := attrs["restore_account_provider_ids"].(types.List).ElementsAs(ctx, &ids, false); d.HasError() {
		return nil, d
	}
	return externalEonSdkAPI.NewRestoreDestinationLimits(effect, ids), nil
}

// flattenRestoreDestinationLimits converts an SDK RestoreDestinationLimits to a Terraform types.Object.
func flattenRestoreDestinationLimits(rdl externalEonSdkAPI.RestoreDestinationLimits) (types.Object, diag.Diagnostics) {
	ids := rdl.GetRestoreAccountProviderIds()
	idVals := make([]attr.Value, len(ids))
	for i, id := range ids {
		idVals[i] = types.StringValue(id)
	}
	idList, d := types.ListValue(types.StringType, idVals)
	if d.HasError() {
		return types.ObjectNull(restoreDestinationLimitsAttrTypes), d
	}
	return types.ObjectValue(restoreDestinationLimitsAttrTypes, map[string]attr.Value{
		"effect":                       types.StringValue(string(rdl.GetEffect())),
		"restore_account_provider_ids": idList,
	})
}

// roleAccessConditionAttrTypes is the attr type map for one access condition (id, effect, expression).
var roleAccessConditionAttrTypes = map[string]attr.Type{
	"id":         types.StringType,
	"effect":     types.StringType,
	"expression": types.ObjectType{AttrTypes: roleExpressionAttrTypes},
}

// flattenRoleAccessConditions converts SDK []AccessCondition to Terraform types.List.
func flattenRoleAccessConditions(ctx context.Context, conds []externalEonSdkAPI.AccessCondition) (types.List, diag.Diagnostics) {
	if conds == nil {
		return types.ListNull(types.ObjectType{AttrTypes: roleAccessConditionAttrTypes}), nil
	}
	elems := make([]attr.Value, 0, len(conds))
	for _, c := range conds {
		exprVal, diags := flattenAccessConditionalExpression(ctx, c.GetExpression())
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: roleAccessConditionAttrTypes}), diags
		}
		obj, d := types.ObjectValue(roleAccessConditionAttrTypes, map[string]attr.Value{
			"id":         types.StringValue(c.GetId()),
			"effect":     types.StringValue(string(c.GetEffect())),
			"expression": exprVal,
		})
		if d.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: roleAccessConditionAttrTypes}), d
		}
		elems = append(elems, obj)
	}
	return types.ListValue(types.ObjectType{AttrTypes: roleAccessConditionAttrTypes}, elems)
}

// accessConditionPlaceholdersFromGrants builds a list of placeholder access conditions (id only, effect and expression null)
// for each access_condition_id referenced in permission grants. Use when the API returns no access_conditions so the
// data source output at least shows which condition IDs are referenced (e.g. "No PII").
func accessConditionPlaceholdersFromGrants(ctx context.Context, grants []externalEonSdkAPI.PermissionGrant) (types.List, diag.Diagnostics) {
	seen := make(map[string]struct{})
	var ids []string
	for _, g := range grants {
		if g.AccessConditionId != nil && *g.AccessConditionId != "" {
			if _, ok := seen[*g.AccessConditionId]; !ok {
				seen[*g.AccessConditionId] = struct{}{}
				ids = append(ids, *g.AccessConditionId)
			}
		}
	}
	if len(ids) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: roleAccessConditionAttrTypes}), nil
	}
	elems := make([]attr.Value, 0, len(ids))
	nullExpr := types.ObjectNull(roleExpressionAttrTypes)
	for _, id := range ids {
		obj, d := types.ObjectValue(roleAccessConditionAttrTypes, map[string]attr.Value{
			"id":         types.StringValue(id),
			"effect":     types.StringNull(),
			"expression": nullExpr,
		})
		if d.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: roleAccessConditionAttrTypes}), d
		}
		elems = append(elems, obj)
	}
	return types.ListValue(types.ObjectType{AttrTypes: roleAccessConditionAttrTypes}, elems)
}

// objectValueWithAllAttrs ensures the Object has a value for every key in attrTypes (null if not set).
// Terraform plugin framework requires Object values to include all attributes.
func objectValueWithAllAttrs(attrTypes map[string]attr.Type, partial map[string]attr.Value) (types.Object, diag.Diagnostics) {
	full := make(map[string]attr.Value, len(attrTypes))
	for k, t := range attrTypes {
		if v, ok := partial[k]; ok {
			full[k] = v
		} else {
			ot, ok := t.(types.ObjectType)
			if !ok {
				full[k] = types.StringNull() // fallback for non-object (e.g. group.operator)
				continue
			}
			full[k] = types.ObjectNull(ot.AttrTypes)
		}
	}
	return types.ObjectValue(attrTypes, full)
}

// roleOperandAttrTypes is the attr type map for a single operand (no nested group).
var roleOperandAttrTypes = map[string]attr.Type{
	"environment":         types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "environments": types.ListType{ElemType: types.StringType}}},
	"resource_type":       types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "resource_types": types.ListType{ElemType: types.StringType}}},
	"data_classes":        types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "data_classes": types.ListType{ElemType: types.StringType}}},
	"tag_keys":            types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "tag_keys": types.ListType{ElemType: types.StringType}}},
	"tag_key_values":      types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "tag_key_values": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"key": types.StringType, "value": types.StringType}}}}},
	"apps":                types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "apps": types.ListType{ElemType: types.StringType}}},
	"cloud_provider":      types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "cloud_providers": types.ListType{ElemType: types.StringType}}},
	"account_id":          types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "account_ids": types.ListType{ElemType: types.StringType}}},
	"source_region":       types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "source_regions": types.ListType{ElemType: types.StringType}}},
	"vpc":                 types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "vpcs": types.ListType{ElemType: types.StringType}}},
	"subnets":             types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "subnets": types.ListType{ElemType: types.StringType}}},
	"resource_group_name": types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "resource_group_names": types.ListType{ElemType: types.StringType}}},
	"resource_name":       types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "resource_names": types.ListType{ElemType: types.StringType}}},
	"resource_id":         types.ObjectType{AttrTypes: map[string]attr.Type{"operator": types.StringType, "resource_ids": types.ListType{ElemType: types.StringType}}},
}

// roleExpressionAttrTypes includes group (operands use roleOperandAttrTypes).
var roleExpressionAttrTypes = func() map[string]attr.Type {
	out := make(map[string]attr.Type)
	for k, v := range roleOperandAttrTypes {
		out[k] = v
	}
	out["group"] = types.ObjectType{AttrTypes: map[string]attr.Type{
		"operator": types.StringType,
		"operands": types.ListType{ElemType: types.ObjectType{AttrTypes: roleOperandAttrTypes}},
	}}
	return out
}()

// flattenAccessConditionalExpression converts SDK AccessConditionalExpression to Terraform types.Object.
func flattenAccessConditionalExpression(ctx context.Context, expr externalEonSdkAPI.AccessConditionalExpression) (types.Object, diag.Diagnostics) {
	attrs := make(map[string]attr.Value)
	if expr.HasEnvironment() {
		env := expr.GetEnvironment()
		envStrs := make([]attr.Value, 0, len(env.GetEnvironments()))
		for _, e := range env.GetEnvironments() {
			envStrs = append(envStrs, types.StringValue(string(e)))
		}
		envList, _ := types.ListValue(types.StringType, envStrs)
		obj, _ := types.ObjectValue(map[string]attr.Type{"operator": types.StringType, "environments": types.ListType{ElemType: types.StringType}}, map[string]attr.Value{
			"operator": types.StringValue(string(env.GetOperator())), "environments": envList,
		})
		attrs["environment"] = obj
	}
	if expr.HasResourceType() {
		rt := expr.GetResourceType()
		rtStrs := make([]attr.Value, 0, len(rt.GetResourceTypes()))
		for _, r := range rt.GetResourceTypes() {
			rtStrs = append(rtStrs, types.StringValue(string(r)))
		}
		rtList, _ := types.ListValue(types.StringType, rtStrs)
		obj, _ := types.ObjectValue(map[string]attr.Type{"operator": types.StringType, "resource_types": types.ListType{ElemType: types.StringType}}, map[string]attr.Value{
			"operator": types.StringValue(string(rt.GetOperator())), "resource_types": rtList,
		})
		attrs["resource_type"] = obj
	}
	if expr.HasDataClasses() {
		dc := expr.GetDataClasses()
		dcStrs := make([]attr.Value, 0, len(dc.GetDataClasses()))
		for _, d := range dc.GetDataClasses() {
			dcStrs = append(dcStrs, types.StringValue(string(d)))
		}
		dcList, _ := types.ListValue(types.StringType, dcStrs)
		obj, _ := types.ObjectValue(map[string]attr.Type{"operator": types.StringType, "data_classes": types.ListType{ElemType: types.StringType}}, map[string]attr.Value{
			"operator": types.StringValue(string(dc.GetOperator())), "data_classes": dcList,
		})
		attrs["data_classes"] = obj
	}
	if expr.HasTagKeys() {
		tk := expr.GetTagKeys()
		tkStrs := make([]attr.Value, 0, len(tk.GetTagKeys()))
		for _, t := range tk.GetTagKeys() {
			tkStrs = append(tkStrs, types.StringValue(t))
		}
		tkList, _ := types.ListValue(types.StringType, tkStrs)
		obj, _ := types.ObjectValue(map[string]attr.Type{"operator": types.StringType, "tag_keys": types.ListType{ElemType: types.StringType}}, map[string]attr.Value{
			"operator": types.StringValue(string(tk.GetOperator())), "tag_keys": tkList,
		})
		attrs["tag_keys"] = obj
	}
	if expr.HasTagKeyValues() {
		tkv := expr.GetTagKeyValues()
		kvObjs := make([]attr.Value, 0, len(tkv.GetTagKeyValues()))
		for _, kv := range tkv.GetTagKeyValues() {
			val := types.StringNull()
			if kv.Value != nil {
				val = types.StringValue(*kv.Value)
			}
			o, _ := types.ObjectValue(map[string]attr.Type{"key": types.StringType, "value": types.StringType}, map[string]attr.Value{"key": types.StringValue(kv.Key), "value": val})
			kvObjs = append(kvObjs, o)
		}
		kvList, _ := types.ListValue(types.ObjectType{AttrTypes: map[string]attr.Type{"key": types.StringType, "value": types.StringType}}, kvObjs)
		obj, _ := types.ObjectValue(map[string]attr.Type{"operator": types.StringType, "tag_key_values": types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"key": types.StringType, "value": types.StringType}}}}, map[string]attr.Value{
			"operator": types.StringValue(string(tkv.GetOperator())), "tag_key_values": kvList,
		})
		attrs["tag_key_values"] = obj
	}
	if expr.HasApps() {
		apps := expr.GetApps()
		appStrs := make([]attr.Value, 0, len(apps.GetApps()))
		for _, a := range apps.GetApps() {
			appStrs = append(appStrs, types.StringValue(a))
		}
		appList, _ := types.ListValue(types.StringType, appStrs)
		obj, _ := types.ObjectValue(map[string]attr.Type{"operator": types.StringType, "apps": types.ListType{ElemType: types.StringType}}, map[string]attr.Value{
			"operator": types.StringValue(string(apps.GetOperator())), "apps": appList,
		})
		attrs["apps"] = obj
	}
	if expr.HasCloudProvider() {
		cp := expr.GetCloudProvider()
		cpStrs := make([]attr.Value, 0, len(cp.GetCloudProviders()))
		for _, p := range cp.GetCloudProviders() {
			cpStrs = append(cpStrs, types.StringValue(string(p)))
		}
		cpList, _ := types.ListValue(types.StringType, cpStrs)
		obj, _ := types.ObjectValue(map[string]attr.Type{"operator": types.StringType, "cloud_providers": types.ListType{ElemType: types.StringType}}, map[string]attr.Value{
			"operator": types.StringValue(string(cp.GetOperator())), "cloud_providers": cpList,
		})
		attrs["cloud_provider"] = obj
	}
	if expr.HasAccountId() {
		acct := expr.GetAccountId()
		acctStrs := make([]attr.Value, 0, len(acct.GetAccountIds()))
		for _, a := range acct.GetAccountIds() {
			acctStrs = append(acctStrs, types.StringValue(a))
		}
		acctList, _ := types.ListValue(types.StringType, acctStrs)
		obj, _ := types.ObjectValue(map[string]attr.Type{"operator": types.StringType, "account_ids": types.ListType{ElemType: types.StringType}}, map[string]attr.Value{
			"operator": types.StringValue(string(acct.GetOperator())), "account_ids": acctList,
		})
		attrs["account_id"] = obj
	}
	if expr.HasSourceRegion() {
		reg := expr.GetSourceRegion()
		regStrs := make([]attr.Value, 0, len(reg.GetRegions()))
		for _, r := range reg.GetRegions() {
			regStrs = append(regStrs, types.StringValue(r))
		}
		regList, _ := types.ListValue(types.StringType, regStrs)
		obj, _ := types.ObjectValue(map[string]attr.Type{"operator": types.StringType, "source_regions": types.ListType{ElemType: types.StringType}}, map[string]attr.Value{
			"operator": types.StringValue(string(reg.GetOperator())), "source_regions": regList,
		})
		attrs["source_region"] = obj
	}
	if expr.HasVpc() {
		vpc := expr.GetVpc()
		vpcStrs := make([]attr.Value, 0, len(vpc.GetVpcs()))
		for _, v := range vpc.GetVpcs() {
			vpcStrs = append(vpcStrs, types.StringValue(v))
		}
		vpcList, _ := types.ListValue(types.StringType, vpcStrs)
		obj, _ := types.ObjectValue(map[string]attr.Type{"operator": types.StringType, "vpcs": types.ListType{ElemType: types.StringType}}, map[string]attr.Value{
			"operator": types.StringValue(string(vpc.GetOperator())), "vpcs": vpcList,
		})
		attrs["vpc"] = obj
	}
	if expr.HasSubnets() {
		sub := expr.GetSubnets()
		subStrs := make([]attr.Value, 0, len(sub.GetSubnets()))
		for _, s := range sub.GetSubnets() {
			subStrs = append(subStrs, types.StringValue(s))
		}
		subList, _ := types.ListValue(types.StringType, subStrs)
		obj, _ := types.ObjectValue(map[string]attr.Type{"operator": types.StringType, "subnets": types.ListType{ElemType: types.StringType}}, map[string]attr.Value{
			"operator": types.StringValue(string(sub.GetOperator())), "subnets": subList,
		})
		attrs["subnets"] = obj
	}
	if expr.HasResourceGroupName() {
		rgn := expr.GetResourceGroupName()
		rgnStrs := make([]attr.Value, 0, len(rgn.GetResourceGroupNames()))
		for _, r := range rgn.GetResourceGroupNames() {
			rgnStrs = append(rgnStrs, types.StringValue(r))
		}
		rgnList, _ := types.ListValue(types.StringType, rgnStrs)
		obj, _ := types.ObjectValue(map[string]attr.Type{"operator": types.StringType, "resource_group_names": types.ListType{ElemType: types.StringType}}, map[string]attr.Value{
			"operator": types.StringValue(string(rgn.GetOperator())), "resource_group_names": rgnList,
		})
		attrs["resource_group_name"] = obj
	}
	if expr.HasResourceName() {
		rn := expr.GetResourceName()
		rnStrs := make([]attr.Value, 0, len(rn.GetResourceNames()))
		for _, r := range rn.GetResourceNames() {
			rnStrs = append(rnStrs, types.StringValue(r))
		}
		rnList, _ := types.ListValue(types.StringType, rnStrs)
		obj, _ := types.ObjectValue(map[string]attr.Type{"operator": types.StringType, "resource_names": types.ListType{ElemType: types.StringType}}, map[string]attr.Value{
			"operator": types.StringValue(string(rn.GetOperator())), "resource_names": rnList,
		})
		attrs["resource_name"] = obj
	}
	if expr.HasResourceId() {
		rid := expr.GetResourceId()
		ridStrs := make([]attr.Value, 0, len(rid.GetResourceIds()))
		for _, r := range rid.GetResourceIds() {
			ridStrs = append(ridStrs, types.StringValue(r))
		}
		ridList, _ := types.ListValue(types.StringType, ridStrs)
		obj, _ := types.ObjectValue(map[string]attr.Type{"operator": types.StringType, "resource_ids": types.ListType{ElemType: types.StringType}}, map[string]attr.Value{
			"operator": types.StringValue(string(rid.GetOperator())), "resource_ids": ridList,
		})
		attrs["resource_id"] = obj
	}
	if expr.HasGroup() {
		group := expr.GetGroup()
		operands := group.GetOperands()
		operandVals := make([]attr.Value, 0, len(operands))
		for _, o := range operands {
			flat, d := flattenAccessConditionalExpression(ctx, o)
			if d.HasError() {
				return types.ObjectNull(roleExpressionAttrTypes), d
			}
			// Operands list expects objects with roleOperandAttrTypes (no "group"); copy set keys from flat.
			partial := make(map[string]attr.Value)
			for k := range roleOperandAttrTypes {
				if v, ok := flat.Attributes()[k]; ok && v != nil && !v.IsNull() {
					partial[k] = v
				}
			}
			operandObj, d := objectValueWithAllAttrs(roleOperandAttrTypes, partial)
			if d.HasError() {
				return types.ObjectNull(roleExpressionAttrTypes), d
			}
			operandVals = append(operandVals, operandObj)
		}
		operandsList, _ := types.ListValue(types.ObjectType{AttrTypes: roleOperandAttrTypes}, operandVals)
		groupObj, _ := types.ObjectValue(map[string]attr.Type{"operator": types.StringType, "operands": types.ListType{ElemType: types.ObjectType{AttrTypes: roleOperandAttrTypes}}}, map[string]attr.Value{
			"operator": types.StringValue(string(group.GetOperator())), "operands": operandsList,
		})
		attrs["group"] = groupObj
	}
	if len(attrs) == 0 {
		return types.ObjectNull(roleExpressionAttrTypes), nil
	}
	return objectValueWithAllAttrs(roleExpressionAttrTypes, attrs)
}
