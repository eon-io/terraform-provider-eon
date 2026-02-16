package provider

import (
	"context"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlattenRoleAccessConditions_Nil(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	list, diags := flattenRoleAccessConditions(ctx, nil)

	require.False(t, diags.HasError())
	assert.True(t, list.IsNull())
}

func TestFlattenRoleAccessConditions_EmptySlice(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	list, diags := flattenRoleAccessConditions(ctx, []externalEonSdkAPI.AccessCondition{})

	require.False(t, diags.HasError())
	require.False(t, list.IsNull())
	assert.Len(t, list.Elements(), 0)
}

func TestFlattenRoleAccessConditions_SingleConditionEnvironment(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	envCond := externalEonSdkAPI.NewEnvironmentCondition(
		externalEonSdkAPI.IN_OPERATOR,
		[]externalEonSdkAPI.Environment{externalEonSdkAPI.PROD},
	)
	expr := externalEonSdkAPI.NewAccessConditionalExpression()
	expr.SetEnvironment(*envCond)
	ac := externalEonSdkAPI.NewAccessCondition(
		"cond-env",
		externalEonSdkAPI.RULE_EFFECT_INCLUSIVE,
		*externalEonSdkAPI.NewNullableAccessConditionalExpression(expr),
	)

	list, diags := flattenRoleAccessConditions(ctx, []externalEonSdkAPI.AccessCondition{*ac})

	require.False(t, diags.HasError())
	require.False(t, list.IsNull())
	elems := list.Elements()
	require.Len(t, elems, 1)

	obj := elems[0].(types.Object)
	attrs := obj.Attributes()
	assert.Equal(t, "cond-env", attrs["id"].(types.String).ValueString())
	assert.Equal(t, "INCLUSIVE", attrs["effect"].(types.String).ValueString())

	exprObj := attrs["expression"].(types.Object)
	exprAttrs := exprObj.Attributes()
	assert.False(t, exprAttrs["environment"].IsNull())
	envObj := exprAttrs["environment"].(types.Object)
	envAttrs := envObj.Attributes()
	assert.Equal(t, "IN", envAttrs["operator"].(types.String).ValueString())
	envList := envAttrs["environments"].(types.List)
	require.Len(t, envList.Elements(), 1)
	assert.Equal(t, "PROD", envList.Elements()[0].(types.String).ValueString())
}

func TestFlattenRoleAccessConditions_MultipleConditions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// Condition 1: environment
	envCond := externalEonSdkAPI.NewEnvironmentCondition(
		externalEonSdkAPI.IN_OPERATOR,
		[]externalEonSdkAPI.Environment{externalEonSdkAPI.PROD},
	)
	expr1 := externalEonSdkAPI.NewAccessConditionalExpression()
	expr1.SetEnvironment(*envCond)
	ac1 := externalEonSdkAPI.NewAccessCondition(
		"cond-env",
		externalEonSdkAPI.RULE_EFFECT_INCLUSIVE,
		*externalEonSdkAPI.NewNullableAccessConditionalExpression(expr1),
	)
	// Condition 2: resource_type
	rtCond := externalEonSdkAPI.NewResourceTypeCondition(
		externalEonSdkAPI.IN_OPERATOR,
		[]externalEonSdkAPI.ResourceType{externalEonSdkAPI.AWS_EC2},
	)
	expr2 := externalEonSdkAPI.NewAccessConditionalExpression()
	expr2.SetResourceType(*rtCond)
	ac2 := externalEonSdkAPI.NewAccessCondition(
		"cond-rt",
		externalEonSdkAPI.RULE_EFFECT_EXCLUSIVE,
		*externalEonSdkAPI.NewNullableAccessConditionalExpression(expr2),
	)

	list, diags := flattenRoleAccessConditions(ctx, []externalEonSdkAPI.AccessCondition{*ac1, *ac2})

	require.False(t, diags.HasError())
	require.False(t, list.IsNull())
	elems := list.Elements()
	require.Len(t, elems, 2)

	// First element: environment condition
	obj0 := elems[0].(types.Object)
	attrs0 := obj0.Attributes()
	assert.Equal(t, "cond-env", attrs0["id"].(types.String).ValueString())
	assert.Equal(t, "INCLUSIVE", attrs0["effect"].(types.String).ValueString())
	expr0 := attrs0["expression"].(types.Object)
	assert.False(t, expr0.Attributes()["environment"].IsNull())
	assert.True(t, expr0.Attributes()["resource_type"].IsNull())

	// Second element: resource_type condition
	obj1 := elems[1].(types.Object)
	attrs1 := obj1.Attributes()
	assert.Equal(t, "cond-rt", attrs1["id"].(types.String).ValueString())
	assert.Equal(t, "EXCLUSIVE", attrs1["effect"].(types.String).ValueString())
	expr1Out := attrs1["expression"].(types.Object)
	assert.True(t, expr1Out.Attributes()["environment"].IsNull())
	assert.False(t, expr1Out.Attributes()["resource_type"].IsNull())
	rtObj := expr1Out.Attributes()["resource_type"].(types.Object)
	rtAttrs := rtObj.Attributes()
	assert.Equal(t, "IN", rtAttrs["operator"].(types.String).ValueString())
	rtList := rtAttrs["resource_types"].(types.List)
	require.Len(t, rtList.Elements(), 1)
	assert.Equal(t, "AWS_EC2", rtList.Elements()[0].(types.String).ValueString())
}

func TestFlattenRoleAccessConditions_ConditionWithGroup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	// Operand 1: environment IN [PROD]
	envCond := externalEonSdkAPI.NewEnvironmentCondition(
		externalEonSdkAPI.IN_OPERATOR,
		[]externalEonSdkAPI.Environment{externalEonSdkAPI.PROD},
	)
	expr1 := externalEonSdkAPI.NewAccessConditionalExpression()
	expr1.SetEnvironment(*envCond)
	// Operand 2: resource_type IN [AWS_EC2]
	rtCond := externalEonSdkAPI.NewResourceTypeCondition(
		externalEonSdkAPI.IN_OPERATOR,
		[]externalEonSdkAPI.ResourceType{externalEonSdkAPI.AWS_EC2},
	)
	expr2 := externalEonSdkAPI.NewAccessConditionalExpression()
	expr2.SetResourceType(*rtCond)
	// Group: AND(environment, resource_type)
	groupCond := externalEonSdkAPI.NewRoleAccessGroupCondition(
		externalEonSdkAPI.LogicalOperator("AND"),
		[]externalEonSdkAPI.AccessConditionalExpression{*expr1, *expr2},
	)
	mainExpr := externalEonSdkAPI.NewAccessConditionalExpression()
	mainExpr.SetGroup(*groupCond)
	ac := externalEonSdkAPI.NewAccessCondition(
		"cond-group",
		externalEonSdkAPI.RULE_EFFECT_INCLUSIVE,
		*externalEonSdkAPI.NewNullableAccessConditionalExpression(mainExpr),
	)

	list, diags := flattenRoleAccessConditions(ctx, []externalEonSdkAPI.AccessCondition{*ac})

	require.False(t, diags.HasError())
	require.False(t, list.IsNull())
	elems := list.Elements()
	require.Len(t, elems, 1)

	obj := elems[0].(types.Object)
	attrs := obj.Attributes()
	assert.Equal(t, "cond-group", attrs["id"].(types.String).ValueString())
	assert.Equal(t, "INCLUSIVE", attrs["effect"].(types.String).ValueString())

	exprObj := attrs["expression"].(types.Object)
	exprAttrs := exprObj.Attributes()
	assert.False(t, exprAttrs["group"].IsNull())

	groupObj := exprAttrs["group"].(types.Object)
	groupAttrs := groupObj.Attributes()
	assert.Equal(t, "AND", groupAttrs["operator"].(types.String).ValueString())
	operandsList := groupAttrs["operands"].(types.List)
	operands := operandsList.Elements()
	require.Len(t, operands, 2)

	// First operand: environment
	op0 := operands[0].(types.Object)
	op0Attrs := op0.Attributes()
	assert.False(t, op0Attrs["environment"].IsNull())
	op0Env := op0Attrs["environment"].(types.Object).Attributes()
	assert.Equal(t, "IN", op0Env["operator"].(types.String).ValueString())
	op0EnvList := op0Env["environments"].(types.List).Elements()
	require.Len(t, op0EnvList, 1)
	assert.Equal(t, "PROD", op0EnvList[0].(types.String).ValueString())

	// Second operand: resource_type
	op1 := operands[1].(types.Object)
	op1Attrs := op1.Attributes()
	assert.False(t, op1Attrs["resource_type"].IsNull())
	op1Rt := op1Attrs["resource_type"].(types.Object).Attributes()
	assert.Equal(t, "IN", op1Rt["operator"].(types.String).ValueString())
	op1RtList := op1Rt["resource_types"].(types.List).Elements()
	require.Len(t, op1RtList, 1)
	assert.Equal(t, "AWS_EC2", op1RtList[0].(types.String).ValueString())
}
