package provider

import (
	"context"
	"fmt"
	"strings"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// monthNames indexes the annually_config month names by their API value, which is a 1-based month
// number rather than a string enum.
var monthNames = []string{
	"JANUARY", "FEBRUARY", "MARCH", "APRIL", "MAY", "JUNE",
	"JULY", "AUGUST", "SEPTEMBER", "OCTOBER", "NOVEMBER", "DECEMBER",
}

func monthName(month int32) (string, error) {
	if month < 1 || int(month) > len(monthNames) {
		return "", fmt.Errorf("month %d is out of range 1-12", month)
	}
	return monthNames[month-1], nil
}

func monthNumber(name string) (int32, error) {
	for i, candidate := range monthNames {
		if candidate == name {
			return int32(i + 1), nil
		}
	}
	return 0, fmt.Errorf("month must be one of %s, got %q", strings.Join(monthNames, ", "), name)
}

func nullOf(t attr.Type) (attr.Value, error) {
	switch typed := t.(type) {
	case basetypes.StringType:
		return types.StringNull(), nil
	case basetypes.BoolType:
		return types.BoolNull(), nil
	case basetypes.Int64Type:
		return types.Int64Null(), nil
	case basetypes.ObjectType:
		return types.ObjectNull(typed.AttrTypes), nil
	case basetypes.ListType:
		return types.ListNull(typed.ElemType), nil
	default:
		return nil, fmt.Errorf("no null value defined for attribute type %T", t)
	}
}

// objectValue builds an object of type t from the attributes the caller resolved, filling the rest
// with typed nulls. The framework rejects an object whose attribute map does not cover its type
// exactly, and every condition and schedule variant in this schema leaves most siblings unset.
func objectValue(t types.ObjectType, present map[string]attr.Value) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	values := make(map[string]attr.Value, len(t.AttrTypes))
	for name, attrType := range t.AttrTypes {
		if value, ok := present[name]; ok {
			values[name] = value
			continue
		}
		nullValue, err := nullOf(attrType)
		if err != nil {
			diags.AddError("Backup Policy Refresh Failed", fmt.Sprintf("Cannot build attribute %q: %s", name, err))
			return types.ObjectNull(t.AttrTypes), diags
		}
		values[name] = nullValue
	}

	for name := range present {
		if _, ok := t.AttrTypes[name]; !ok {
			diags.AddError("Backup Policy Refresh Failed", fmt.Sprintf("Attribute %q is not part of the schema", name))
			return types.ObjectNull(t.AttrTypes), diags
		}
	}

	object, objectDiags := types.ObjectValue(t.AttrTypes, values)
	diags.Append(objectDiags...)
	return object, diags
}

// nestedObjectType and nestedListElementType read the shape of a nested attribute off the resource
// schema instead of restating it here, so the flatten path cannot drift away from the schema.
func nestedObjectType(t types.ObjectType, name string) (types.ObjectType, diag.Diagnostics) {
	var diags diag.Diagnostics

	attrType, ok := t.AttrTypes[name]
	if !ok {
		diags.AddError("Backup Policy Refresh Failed", fmt.Sprintf("Schema has no attribute %q", name))
		return types.ObjectType{}, diags
	}
	objectType, ok := attrType.(types.ObjectType)
	if !ok {
		diags.AddError("Backup Policy Refresh Failed", fmt.Sprintf("Attribute %q is %T, expected an object", name, attrType))
		return types.ObjectType{}, diags
	}
	return objectType, diags
}

func nestedListElementType(t types.ObjectType, name string) (types.ObjectType, diag.Diagnostics) {
	var diags diag.Diagnostics

	attrType, ok := t.AttrTypes[name]
	if !ok {
		diags.AddError("Backup Policy Refresh Failed", fmt.Sprintf("Schema has no attribute %q", name))
		return types.ObjectType{}, diags
	}
	listType, ok := attrType.(types.ListType)
	if !ok {
		diags.AddError("Backup Policy Refresh Failed", fmt.Sprintf("Attribute %q is %T, expected a list", name, attrType))
		return types.ObjectType{}, diags
	}
	elementType, ok := listType.ElemType.(types.ObjectType)
	if !ok {
		diags.AddError("Backup Policy Refresh Failed", fmt.Sprintf("Attribute %q holds %T elements, expected objects", name, listType.ElemType))
		return types.ObjectType{}, diags
	}
	return elementType, diags
}

// priorObject digs an object out of state Terraform already holds. Everything is null after an
// import, and the flatten path only consults prior values where the API cannot round-trip what the
// configuration wrote, so a null here is expected rather than an error.
func priorObject(prior types.Object, name string) types.Object {
	if prior.IsNull() || prior.IsUnknown() {
		return types.ObjectNull(nil)
	}
	value, ok := prior.Attributes()[name]
	if !ok {
		return types.ObjectNull(nil)
	}
	object, ok := value.(types.Object)
	if !ok {
		return types.ObjectNull(nil)
	}
	return object
}

func priorInt64(prior types.Object, name string) types.Int64 {
	if prior.IsNull() || prior.IsUnknown() {
		return types.Int64Null()
	}
	value, ok := prior.Attributes()[name]
	if !ok {
		return types.Int64Null()
	}
	number, ok := value.(types.Int64)
	if !ok {
		return types.Int64Null()
	}
	return number
}

func priorList(prior types.Object, name string) types.List {
	if prior.IsNull() || prior.IsUnknown() {
		return types.ListNull(types.StringType)
	}
	value, ok := prior.Attributes()[name]
	if !ok {
		return types.ListNull(types.StringType)
	}
	list, ok := value.(types.List)
	if !ok {
		return types.ListNull(types.StringType)
	}
	return list
}

// flattenOverrideList maps an override list the API omits when empty. A configuration that wrote an
// explicit empty list keeps it, since the API cannot distinguish that from an absent list and
// rewriting it as null would diff on every plan; anything else follows the API.
func flattenOverrideList(ctx context.Context, values []string, prior types.List) (types.List, diag.Diagnostics) {
	if len(values) == 0 {
		if !prior.IsNull() && !prior.IsUnknown() && len(prior.Elements()) == 0 {
			return prior, nil
		}
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(ctx, types.StringType, values)
}

func priorString(prior types.Object, name string) types.String {
	if prior.IsNull() || prior.IsUnknown() {
		return types.StringNull()
	}
	value, ok := prior.Attributes()[name]
	if !ok {
		return types.StringNull()
	}
	text, ok := value.(types.String)
	if !ok {
		return types.StringNull()
	}
	return text
}

// priorListElement returns the element that held position index in a prior list attribute, matching
// schedules to their previous representation by position because they carry no identifier.
func priorListElement(prior types.Object, name string, index int) types.Object {
	if prior.IsNull() || prior.IsUnknown() {
		return types.ObjectNull(nil)
	}
	value, ok := prior.Attributes()[name]
	if !ok {
		return types.ObjectNull(nil)
	}
	list, ok := value.(types.List)
	if !ok || list.IsNull() || list.IsUnknown() {
		return types.ObjectNull(nil)
	}
	elements := list.Elements()
	if index >= len(elements) {
		return types.ObjectNull(nil)
	}
	element, ok := elements[index].(types.Object)
	if !ok {
		return types.ObjectNull(nil)
	}
	return element
}

func enumStrings[T ~string](values []T) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

// flattenBackupPolicy refreshes resource_selector and backup_plan from the API so that changes made
// outside Terraform surface as drift and an imported policy compares against what Eon holds.
// created_at and updated_at are deliberately left alone: the API does not carry them, so anything
// written here would differ on every refresh and show as a permanent diff.
func flattenBackupPolicy(
	ctx context.Context,
	policy *externalEonSdkAPI.BackupPolicy,
	schemaType types.ObjectType,
	data *BackupPolicyResourceModel,
) diag.Diagnostics {
	var diags diag.Diagnostics

	selectorType, typeDiags := nestedObjectType(schemaType, "resource_selector")
	diags.Append(typeDiags...)
	planType, typeDiags := nestedObjectType(schemaType, "backup_plan")
	diags.Append(typeDiags...)
	if diags.HasError() {
		return diags
	}

	selector, selectorDiags := flattenResourceSelector(ctx, selectorType, policy.ResourceSelector, data.ResourceSelector)
	diags.Append(selectorDiags...)

	plan, planDiags := flattenBackupPlan(ctx, planType, policy.BackupPlan, data.BackupPlan)
	diags.Append(planDiags...)
	if diags.HasError() {
		return diags
	}

	data.Id = types.StringValue(policy.Id)
	data.Name = types.StringValue(policy.Name)
	data.Enabled = types.BoolValue(policy.Enabled)
	data.ResourceSelector = selector
	data.BackupPlan = plan

	return diags
}

func flattenResourceSelector(
	ctx context.Context,
	t types.ObjectType,
	selector externalEonSdkAPI.BackupPolicyResourceSelector,
	prior types.Object,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	present := map[string]attr.Value{
		"resource_selection_mode": types.StringValue(string(selector.ResourceSelectionMode)),
	}

	for name, values := range map[string][]string{
		"resource_inclusion_override": selector.ResourceInclusionOverride,
		"resource_exclusion_override": selector.ResourceExclusionOverride,
	} {
		override, overrideDiags := flattenOverrideList(ctx, values, priorList(prior, name))
		diags.Append(overrideDiags...)
		if !override.IsNull() {
			present[name] = override
		}
	}

	if expression := selector.Expression.Get(); expression != nil {
		expressionType, typeDiags := nestedObjectType(t, "expression")
		diags.Append(typeDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}

		flattened, expressionDiags := flattenExpression(ctx, expressionType, *expression, priorObject(prior, "expression"))
		diags.Append(expressionDiags...)
		present["expression"] = flattened
	}

	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	object, objectDiags := objectValue(t, present)
	diags.Append(objectDiags...)
	return object, diags
}

// unrepresentableConditions reports the conditions Eon returned that this provider's schema has no
// attribute for. Silently dropping them would refresh the resource into a state the configuration
// can never reproduce, which reads as an unfixable diff on every plan.
func unrepresentableConditions(expression externalEonSdkAPI.BackupPolicyExpression, supported map[string]bool) []string {
	candidates := map[string]bool{
		"group":                         expression.Group.Get() != nil,
		"resource_type":                 expression.ResourceType.Get() != nil,
		"data_classes":                  expression.DataClasses.Get() != nil,
		"environment":                   expression.Environment.Get() != nil,
		"apps":                          expression.Apps.Get() != nil,
		"cloud_provider":                expression.CloudProvider.Get() != nil,
		"account_id":                    expression.AccountId.Get() != nil,
		"source_region":                 expression.SourceRegion.Get() != nil,
		"vpc":                           expression.Vpc.Get() != nil,
		"subnets":                       expression.Subnets.Get() != nil,
		"resource_group_name":           expression.ResourceGroupName.Get() != nil,
		"resource_name":                 expression.ResourceName.Get() != nil,
		"resource_id":                   expression.ResourceId.Get() != nil,
		"tag_keys":                      expression.TagKeys.Get() != nil,
		"tag_key_values":                expression.TagKeyValues.Get() != nil,
		"source_account_tag_keys":       expression.SourceAccountTagKeys.Get() != nil,
		"source_account_tag_key_values": expression.SourceAccountTagKeyValues.Get() != nil,
		"global_cluster_identifier":     expression.GlobalClusterIdentifier.Get() != nil,
	}

	var unrepresentable []string
	for name, set := range candidates {
		if set && !supported[name] {
			unrepresentable = append(unrepresentable, name)
		}
	}
	return unrepresentable
}

func flattenExpression(
	ctx context.Context,
	t types.ObjectType,
	expression externalEonSdkAPI.BackupPolicyExpression,
	prior types.Object,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	supported := map[string]bool{"environment": true, "resource_type": true, "tag_keys": true, "tag_key_values": true, "group": true}
	if unrepresentable := unrepresentableConditions(expression, supported); len(unrepresentable) > 0 {
		diags.AddError(
			"Unsupported Backup Policy Condition",
			fmt.Sprintf("Eon returned the condition(s) %s at the top level of resource_selector.expression. "+
				"This provider version can only represent %s there; nest the others inside a group condition, "+
				"or upgrade the provider.",
				strings.Join(unrepresentable, ", "), "environment, resource_type, tag_keys, tag_key_values, group"),
		)
		return types.ObjectNull(t.AttrTypes), diags
	}

	present := map[string]attr.Value{}

	if condition := expression.Environment.Get(); condition != nil {
		flattened, conditionDiags := flattenValuesCondition(ctx, t, "environment", string(condition.Operator), enumStrings(condition.Environments))
		diags.Append(conditionDiags...)
		present["environment"] = flattened
	}

	if condition := expression.ResourceType.Get(); condition != nil {
		flattened, conditionDiags := flattenValuesCondition(ctx, t, "resource_type", string(condition.Operator), enumStrings(condition.ResourceTypes))
		diags.Append(conditionDiags...)
		present["resource_type"] = flattened
	}

	if condition := expression.TagKeys.Get(); condition != nil {
		flattened, conditionDiags := flattenValuesCondition(ctx, t, "tag_keys", string(condition.Operator), condition.TagKeys)
		diags.Append(conditionDiags...)
		present["tag_keys"] = flattened
	}

	if condition := expression.TagKeyValues.Get(); condition != nil {
		flattened, conditionDiags := flattenTagKeyValuesCondition(t, "tag_key_values", *condition)
		diags.Append(conditionDiags...)
		present["tag_key_values"] = flattened
	}

	if condition := expression.Group.Get(); condition != nil {
		groupType, typeDiags := nestedObjectType(t, "group")
		diags.Append(typeDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}

		flattened, groupDiags := flattenGroupCondition(ctx, groupType, *condition, priorObject(prior, "group"))
		diags.Append(groupDiags...)
		present["group"] = flattened
	}

	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	object, objectDiags := objectValue(t, present)
	diags.Append(objectDiags...)
	return object, diags
}

func flattenGroupCondition(
	ctx context.Context,
	t types.ObjectType,
	group externalEonSdkAPI.BackupPolicyGroupCondition,
	prior types.Object,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	operandType, typeDiags := nestedListElementType(t, "operands")
	diags.Append(typeDiags...)
	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	operands := make([]attr.Value, 0, len(group.Operands))
	for _, operand := range group.Operands {
		flattened, operandDiags := flattenOperand(ctx, operandType, operand)
		diags.Append(operandDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}
		operands = append(operands, flattened)
	}

	operandList, listDiags := types.ListValue(operandType, operands)
	diags.Append(listDiags...)
	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	object, objectDiags := objectValue(t, map[string]attr.Value{
		"operator": types.StringValue(string(group.Operator)),
		"operands": operandList,
	})
	diags.Append(objectDiags...)
	return object, diags
}

func flattenOperand(
	ctx context.Context,
	t types.ObjectType,
	operand externalEonSdkAPI.BackupPolicyExpression,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	supported := map[string]bool{
		"resource_type": true, "environment": true, "tag_keys": true, "tag_key_values": true,
		"data_classes": true, "apps": true, "cloud_provider": true, "account_id": true,
		"source_region": true, "vpc": true, "subnets": true, "resource_group_name": true,
		"resource_name": true, "resource_id": true,
	}
	if unrepresentable := unrepresentableConditions(operand, supported); len(unrepresentable) > 0 {
		diags.AddError(
			"Unsupported Backup Policy Condition",
			fmt.Sprintf("Eon returned the condition(s) %s inside a group operand. This provider version "+
				"cannot represent them; upgrade the provider to manage this policy with Terraform.",
				strings.Join(unrepresentable, ", ")),
		)
		return types.ObjectNull(t.AttrTypes), diags
	}

	present := map[string]attr.Value{}

	// Every operand condition is an {operator, <values>} pair, so they only differ in where the
	// values come from on the API type.
	type valuesCondition struct {
		name     string
		operator string
		values   []string
	}
	var conditions []valuesCondition

	if condition := operand.ResourceType.Get(); condition != nil {
		conditions = append(conditions, valuesCondition{"resource_type", string(condition.Operator), enumStrings(condition.ResourceTypes)})
	}
	if condition := operand.Environment.Get(); condition != nil {
		conditions = append(conditions, valuesCondition{"environment", string(condition.Operator), enumStrings(condition.Environments)})
	}
	if condition := operand.TagKeys.Get(); condition != nil {
		conditions = append(conditions, valuesCondition{"tag_keys", string(condition.Operator), condition.TagKeys})
	}
	if condition := operand.DataClasses.Get(); condition != nil {
		conditions = append(conditions, valuesCondition{"data_classes", string(condition.Operator), condition.DataClasses})
	}
	if condition := operand.Apps.Get(); condition != nil {
		conditions = append(conditions, valuesCondition{"apps", string(condition.Operator), condition.Apps})
	}
	if condition := operand.CloudProvider.Get(); condition != nil {
		conditions = append(conditions, valuesCondition{"cloud_provider", string(condition.Operator), enumStrings(condition.CloudProviders)})
	}
	if condition := operand.AccountId.Get(); condition != nil {
		conditions = append(conditions, valuesCondition{"account_id", string(condition.Operator), condition.AccountIds})
	}
	if condition := operand.SourceRegion.Get(); condition != nil {
		conditions = append(conditions, valuesCondition{"source_region", string(condition.Operator), condition.Regions})
	}
	if condition := operand.Vpc.Get(); condition != nil {
		conditions = append(conditions, valuesCondition{"vpc", string(condition.Operator), condition.Vpcs})
	}
	if condition := operand.Subnets.Get(); condition != nil {
		conditions = append(conditions, valuesCondition{"subnets", string(condition.Operator), condition.Subnets})
	}
	if condition := operand.ResourceGroupName.Get(); condition != nil {
		conditions = append(conditions, valuesCondition{"resource_group_name", string(condition.Operator), condition.ResourceGroupNames})
	}
	if condition := operand.ResourceName.Get(); condition != nil {
		conditions = append(conditions, valuesCondition{"resource_name", string(condition.Operator), condition.ResourceNames})
	}
	if condition := operand.ResourceId.Get(); condition != nil {
		conditions = append(conditions, valuesCondition{"resource_id", string(condition.Operator), condition.ResourceIds})
	}

	for _, condition := range conditions {
		flattened, conditionDiags := flattenValuesCondition(ctx, t, condition.name, condition.operator, condition.values)
		diags.Append(conditionDiags...)
		present[condition.name] = flattened
	}

	if condition := operand.TagKeyValues.Get(); condition != nil {
		flattened, conditionDiags := flattenTagKeyValuesCondition(t, "tag_key_values", *condition)
		diags.Append(conditionDiags...)
		present["tag_key_values"] = flattened
	}

	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	object, objectDiags := objectValue(t, present)
	diags.Append(objectDiags...)
	return object, diags
}

// flattenValuesCondition builds a condition object from its operator and values. Each condition in
// the schema holds exactly an operator plus one list whose name varies per condition
// (environments, resource_types, vpcs, ...), so the list attribute is the non-operator one.
func flattenValuesCondition(
	ctx context.Context,
	parent types.ObjectType,
	name string,
	operator string,
	values []string,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	conditionType, typeDiags := nestedObjectType(parent, name)
	diags.Append(typeDiags...)
	if diags.HasError() {
		return types.ObjectNull(nil), diags
	}

	valuesField := ""
	for attrName := range conditionType.AttrTypes {
		if attrName != "operator" {
			valuesField = attrName
		}
	}
	if valuesField == "" {
		diags.AddError("Backup Policy Refresh Failed", fmt.Sprintf("Condition %q has no values attribute", name))
		return types.ObjectNull(conditionType.AttrTypes), diags
	}

	list, listDiags := types.ListValueFrom(ctx, types.StringType, values)
	diags.Append(listDiags...)
	if diags.HasError() {
		return types.ObjectNull(conditionType.AttrTypes), diags
	}

	object, objectDiags := objectValue(conditionType, map[string]attr.Value{
		"operator":  types.StringValue(operator),
		valuesField: list,
	})
	diags.Append(objectDiags...)
	return object, diags
}

func flattenTagKeyValuesCondition(
	parent types.ObjectType,
	name string,
	condition externalEonSdkAPI.TagKeyValuesCondition,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	conditionType, typeDiags := nestedObjectType(parent, name)
	diags.Append(typeDiags...)
	if diags.HasError() {
		return types.ObjectNull(nil), diags
	}

	pairType, typeDiags := nestedListElementType(conditionType, "tag_key_values")
	diags.Append(typeDiags...)
	if diags.HasError() {
		return types.ObjectNull(conditionType.AttrTypes), diags
	}

	pairs := make([]attr.Value, 0, len(condition.TagKeyValues))
	for _, tagKeyValue := range condition.TagKeyValues {
		pair, pairDiags := objectValue(pairType, map[string]attr.Value{
			"key":   types.StringValue(tagKeyValue.Key),
			"value": types.StringValue(tagKeyValue.GetValue()),
		})
		diags.Append(pairDiags...)
		if diags.HasError() {
			return types.ObjectNull(conditionType.AttrTypes), diags
		}
		pairs = append(pairs, pair)
	}

	pairList, listDiags := types.ListValue(pairType, pairs)
	diags.Append(listDiags...)
	if diags.HasError() {
		return types.ObjectNull(conditionType.AttrTypes), diags
	}

	object, objectDiags := objectValue(conditionType, map[string]attr.Value{
		"operator":       types.StringValue(string(condition.Operator)),
		"tag_key_values": pairList,
	})
	diags.Append(objectDiags...)
	return object, diags
}

func flattenBackupPlan(
	ctx context.Context,
	t types.ObjectType,
	plan externalEonSdkAPI.BackupPolicyPlan,
	prior types.Object,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	policyType := string(plan.BackupPolicyType)
	present := map[string]attr.Value{"backup_policy_type": types.StringValue(policyType)}

	switch policyType {
	case "STANDARD", "PITR":
		standard := plan.StandardPlan.Get()
		if standard == nil {
			diags.AddError("Backup Policy Refresh Failed", fmt.Sprintf("Eon reported policy type %s without a standard plan", policyType))
			return types.ObjectNull(t.AttrTypes), diags
		}

		standardType, typeDiags := nestedObjectType(t, "standard_plan")
		diags.Append(typeDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}

		flattened, standardDiags := flattenStandardPlan(ctx, standardType, *standard, priorObject(prior, "standard_plan"))
		diags.Append(standardDiags...)
		present["standard_plan"] = flattened

	case "HIGH_FREQUENCY":
		highFrequency := plan.HighFrequencyPlan.Get()
		if highFrequency == nil {
			diags.AddError("Backup Policy Refresh Failed", "Eon reported policy type HIGH_FREQUENCY without a high frequency plan")
			return types.ObjectNull(t.AttrTypes), diags
		}

		highFrequencyType, typeDiags := nestedObjectType(t, "high_frequency_plan")
		diags.Append(typeDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}

		flattened, highFrequencyDiags := flattenHighFrequencyPlan(ctx, highFrequencyType, *highFrequency, priorObject(prior, "high_frequency_plan"))
		diags.Append(highFrequencyDiags...)
		present["high_frequency_plan"] = flattened

	case "AWS_NATIVE_PITR":
		awsNativePitr := plan.AwsNativePitrPlan.Get()
		if awsNativePitr == nil {
			diags.AddError("Backup Policy Refresh Failed", "Eon reported policy type AWS_NATIVE_PITR without an AWS native PITR plan")
			return types.ObjectNull(t.AttrTypes), diags
		}

		awsNativePitrType, typeDiags := nestedObjectType(t, "aws_native_pitr_plan")
		diags.Append(typeDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}

		flattened, awsNativePitrDiags := objectValue(awsNativePitrType, map[string]attr.Value{
			"retention_days": types.Int64Value(int64(awsNativePitr.RetentionDays)),
			"resource_type":  types.StringValue(string(awsNativePitr.ResourceType.GetResourceType())),
		})
		diags.Append(awsNativePitrDiags...)
		present["aws_native_pitr_plan"] = flattened

	case "AWS_NATIVE_STANDARD":
		awsNativeStandard := plan.AwsNativeStandardPlan.Get()
		if awsNativeStandard == nil {
			diags.AddError("Backup Policy Refresh Failed", "Eon reported policy type AWS_NATIVE_STANDARD without an AWS native standard plan")
			return types.ObjectNull(t.AttrTypes), diags
		}

		awsNativeStandardType, typeDiags := nestedObjectType(t, "aws_native_standard_plan")
		diags.Append(typeDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}

		flattened, awsNativeStandardDiags := flattenAwsNativeStandardPlan(
			awsNativeStandardType, *awsNativeStandard, priorObject(prior, "aws_native_standard_plan"))
		diags.Append(awsNativeStandardDiags...)
		present["aws_native_standard_plan"] = flattened

	default:
		diags.AddError(
			"Unsupported Backup Policy Type",
			fmt.Sprintf("Eon reports this policy as type '%s', which this provider version cannot represent. "+
				"Supported types: %s.", policyType, supportedBackupPolicyTypes),
		)
		return types.ObjectNull(t.AttrTypes), diags
	}

	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	object, objectDiags := objectValue(t, present)
	diags.Append(objectDiags...)
	return object, diags
}

func flattenStandardPlan(
	ctx context.Context,
	t types.ObjectType,
	plan externalEonSdkAPI.StandardBackupPolicyPlan,
	prior types.Object,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	scheduleType, typeDiags := nestedListElementType(t, "backup_schedules")
	diags.Append(typeDiags...)
	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	configType, typeDiags := nestedObjectType(scheduleType, "schedule_config")
	diags.Append(typeDiags...)
	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	schedules := make([]attr.Value, 0, len(plan.BackupSchedules))
	for index, schedule := range plan.BackupSchedules {
		priorSchedule := priorListElement(prior, "backup_schedules", index)

		config, configDiags := flattenStandardScheduleConfig(configType, schedule.ScheduleConfig, priorObject(priorSchedule, "schedule_config"))
		diags.Append(configDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}

		flattened, scheduleDiags := objectValue(scheduleType, map[string]attr.Value{
			"vault_id":        types.StringValue(schedule.VaultId),
			"retention_days":  types.Int64Value(int64(schedule.BackupRetentionDays)),
			"schedule_config": config,
		})
		diags.Append(scheduleDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}
		schedules = append(schedules, flattened)
	}

	scheduleList, listDiags := types.ListValue(scheduleType, schedules)
	diags.Append(listDiags...)
	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	present := map[string]attr.Value{"backup_schedules": scheduleList}
	if timezone := flattenScheduleTimezone(plan.ScheduleTimezone, priorString(prior, "schedule_timezone")); !timezone.IsNull() {
		present["schedule_timezone"] = timezone
	}

	object, objectDiags := objectValue(t, present)
	diags.Append(objectDiags...)
	return object, diags
}

// flattenAwsNativeStandardPlan mirrors flattenStandardPlan for source-account schedules, which are
// keyed by target region and carry no vault or plan-level timezone.
func flattenAwsNativeStandardPlan(
	t types.ObjectType,
	plan externalEonSdkAPI.AwsNativeStandardBackupPolicyPlan,
	prior types.Object,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	scheduleType, typeDiags := nestedListElementType(t, "backup_schedules")
	diags.Append(typeDiags...)
	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	configType, typeDiags := nestedObjectType(scheduleType, "schedule_config")
	diags.Append(typeDiags...)
	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	schedules := make([]attr.Value, 0, len(plan.BackupSchedules))
	for index, schedule := range plan.BackupSchedules {
		priorSchedule := priorListElement(prior, "backup_schedules", index)

		config, configDiags := flattenStandardScheduleConfig(
			configType,
			awsNativeStandardScheduleConfigAsStandard(schedule.ScheduleConfig),
			priorObject(priorSchedule, "schedule_config"),
		)
		diags.Append(configDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}

		// An empty region means "back up in the resource's own region", which the configuration
		// expresses by omitting target_region, so writing "" back would read as drift.
		targetRegion := types.StringNull()
		if schedule.TargetRegion != "" {
			targetRegion = types.StringValue(schedule.TargetRegion)
		}

		flattened, scheduleDiags := objectValue(scheduleType, map[string]attr.Value{
			"target_region":   targetRegion,
			"retention_days":  types.Int64Value(int64(schedule.BackupRetentionDays)),
			"schedule_config": config,
		})
		diags.Append(scheduleDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}
		schedules = append(schedules, flattened)
	}

	scheduleList, listDiags := types.ListValue(scheduleType, schedules)
	diags.Append(listDiags...)
	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	object, objectDiags := objectValue(t, map[string]attr.Value{"backup_schedules": scheduleList})
	diags.Append(objectDiags...)
	return object, diags
}

// awsNativeStandardScheduleConfigAsStandard re-keys the native schedule config onto its standard
// twin, which shares every frequency config and differs only in the interval type.
func awsNativeStandardScheduleConfigAsStandard(
	config externalEonSdkAPI.AwsNativeStandardBackupScheduleConfig,
) externalEonSdkAPI.StandardBackupScheduleConfig {
	standard := externalEonSdkAPI.NewStandardBackupScheduleConfig(config.Frequency)
	if interval := config.IntervalConfig.Get(); interval != nil {
		standard.SetIntervalConfig(*externalEonSdkAPI.NewStandardIntervalConfig(interval.IntervalHours))
	}
	if daily := config.DailyConfig.Get(); daily != nil {
		standard.SetDailyConfig(*daily)
	}
	if weekly := config.WeeklyConfig.Get(); weekly != nil {
		standard.SetWeeklyConfig(*weekly)
	}
	if monthly := config.MonthlyConfig.Get(); monthly != nil {
		standard.SetMonthlyConfig(*monthly)
	}
	if annually := config.AnnuallyConfig.Get(); annually != nil {
		standard.SetAnnuallyConfig(*annually)
	}
	return *standard
}

// flattenScheduleTimezone keeps schedule_timezone null when the configuration omitted it and Eon
// reports its UTC default, so an unset optional attribute does not read as drift on every plan.
func flattenScheduleTimezone(timezone *externalEonSdkAPI.ScheduleTimezone, prior types.String) types.String {
	if timezone == nil {
		return types.StringNull()
	}
	if prior.IsNull() && *timezone == externalEonSdkAPI.SCHEDULE_TIMEZONE_UTC {
		return types.StringNull()
	}
	return types.StringValue(string(*timezone))
}

func flattenStandardScheduleConfig(
	t types.ObjectType,
	config externalEonSdkAPI.StandardBackupScheduleConfig,
	prior types.Object,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	frequency := string(config.Frequency)
	present := map[string]attr.Value{"frequency": types.StringValue(frequency)}

	switch config.Frequency {
	case externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_DAILY:
		daily := config.DailyConfig.Get()
		if daily == nil {
			break
		}
		dailyType, typeDiags := nestedObjectType(t, "daily_config")
		diags.Append(typeDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}
		flattened, dailyDiags := objectValue(dailyType, timeOfDayAttrs(daily.TimeOfDay, daily.StartWindowMinutes))
		diags.Append(dailyDiags...)
		present["daily_config"] = flattened

	case externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_WEEKLY:
		weekly := config.WeeklyConfig.Get()
		if weekly == nil {
			break
		}
		if len(weekly.DaysOfWeek) > 1 {
			diags.AddError(
				"Unsupported Backup Schedule",
				fmt.Sprintf("Eon reports this weekly schedule on %d days of the week. This provider version "+
					"models a single day_of_week; upgrade the provider to manage this policy with Terraform.",
					len(weekly.DaysOfWeek)),
			)
			return types.ObjectNull(t.AttrTypes), diags
		}
		weeklyType, typeDiags := nestedObjectType(t, "weekly_config")
		diags.Append(typeDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}
		attrs := timeOfDayAttrs(&weekly.TimeOfDay, weekly.StartWindowMinutes)
		if len(weekly.DaysOfWeek) == 1 {
			attrs["day_of_week"] = types.StringValue(string(weekly.DaysOfWeek[0]))
		}
		flattened, weeklyDiags := objectValue(weeklyType, attrs)
		diags.Append(weeklyDiags...)
		present["weekly_config"] = flattened

	case externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_MONTHLY:
		monthly := config.MonthlyConfig.Get()
		if monthly == nil {
			break
		}
		if len(monthly.DaysOfMonth) > 1 {
			diags.AddError(
				"Unsupported Backup Schedule",
				fmt.Sprintf("Eon reports this monthly schedule on %d days of the month. This provider version "+
					"models a single day_of_month; upgrade the provider to manage this policy with Terraform.",
					len(monthly.DaysOfMonth)),
			)
			return types.ObjectNull(t.AttrTypes), diags
		}
		monthlyType, typeDiags := nestedObjectType(t, "monthly_config")
		diags.Append(typeDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}
		priorMonthly := priorObject(prior, "monthly_config")
		dayOfMonth := priorInt64(priorMonthly, "day_of_month")
		if len(monthly.DaysOfMonth) == 1 {
			dayOfMonth = types.Int64Value(int64(monthly.DaysOfMonth[0]))
		}
		attrs := timeOfDayAttrs(monthly.TimeOfDay, monthly.StartWindowMinutes)
		attrs["day_of_month"] = dayOfMonth
		flattened, monthlyDiags := objectValue(monthlyType, attrs)
		diags.Append(monthlyDiags...)
		present["monthly_config"] = flattened

	case externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_ANNUALLY:
		annually := config.AnnuallyConfig.Get()
		if annually == nil {
			break
		}
		annuallyType, typeDiags := nestedObjectType(t, "annually_config")
		diags.Append(typeDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}
		priorAnnually := priorObject(prior, "annually_config")
		attrs := timeOfDayAttrs(annually.TimeOfDay, annually.StartWindowMinutes)
		attrs["month"] = priorString(priorAnnually, "month")
		attrs["day_of_month"] = priorInt64(priorAnnually, "day_of_month")
		if timeOfYear := annually.TimeOfYear; timeOfYear != nil {
			name, err := monthName(timeOfYear.Month)
			if err != nil {
				diags.AddError("Backup Policy Refresh Failed", fmt.Sprintf("Eon returned an unusable annual schedule: %s", err))
				return types.ObjectNull(t.AttrTypes), diags
			}
			attrs["month"] = types.StringValue(name)
			attrs["day_of_month"] = types.Int64Value(int64(timeOfYear.DayOfMonth))
		}
		flattened, annuallyDiags := objectValue(annuallyType, attrs)
		diags.Append(annuallyDiags...)
		present["annually_config"] = flattened

	case externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_INTERVAL:
		interval := config.IntervalConfig.Get()
		if interval == nil {
			break
		}
		intervalType, typeDiags := nestedObjectType(t, "interval_config")
		diags.Append(typeDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}
		flattened, intervalDiags := flattenIntervalConfig(intervalType, int64(interval.IntervalHours)*60, priorObject(prior, "interval_config"))
		diags.Append(intervalDiags...)
		present["interval_config"] = flattened

	default:
		diags.AddError(
			"Unsupported Backup Schedule",
			fmt.Sprintf("Eon reports this schedule with frequency '%s', which this provider version cannot "+
				"represent. Supported frequencies: DAILY, WEEKLY, MONTHLY, ANNUALLY, INTERVAL.", frequency),
		)
		return types.ObjectNull(t.AttrTypes), diags
	}

	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	object, objectDiags := objectValue(t, present)
	diags.Append(objectDiags...)
	return object, diags
}

// timeOfDayAttrs maps the API's optional time-of-day and start window onto the attribute names every
// schedule config variant shares.
func timeOfDayAttrs(timeOfDay *externalEonSdkAPI.TimeOfDay, startWindowMinutes *int32) map[string]attr.Value {
	attrs := map[string]attr.Value{}
	if timeOfDay != nil {
		attrs["time_of_day_hour"] = types.Int64Value(int64(timeOfDay.Hour))
		attrs["time_of_day_minutes"] = types.Int64Value(int64(timeOfDay.Minute))
	}
	if startWindowMinutes != nil {
		attrs["start_window_minutes"] = types.Int64Value(int64(*startWindowMinutes))
	}
	return attrs
}

// flattenIntervalConfig reports the interval in whichever unit the configuration used, because
// interval_minutes and interval_hours describe the same interval and rewriting one as the other
// would show as permanent drift. start_window_minutes has no field on either interval config in the
// API, so the prior value is carried over rather than being cleared on every refresh.
func flattenIntervalConfig(t types.ObjectType, intervalMinutes int64, prior types.Object) (types.Object, diag.Diagnostics) {
	present := map[string]attr.Value{}

	if startWindow := priorInt64(prior, "start_window_minutes"); !startWindow.IsNull() {
		present["start_window_minutes"] = startWindow
	}

	priorHours := priorInt64(prior, "interval_hours")
	if !priorHours.IsNull() && priorHours.ValueInt64()*60 == intervalMinutes {
		present["interval_hours"] = priorHours
		return objectValue(t, present)
	}

	priorMinutes := priorInt64(prior, "interval_minutes")
	if !priorMinutes.IsNull() && priorMinutes.ValueInt64() == intervalMinutes {
		present["interval_minutes"] = priorMinutes
		return objectValue(t, present)
	}

	// Nothing in state to match: report hours when the interval divides evenly, since that is the
	// unit the API itself stores for standard schedules.
	if intervalMinutes%60 == 0 {
		present["interval_hours"] = types.Int64Value(intervalMinutes / 60)
	} else {
		present["interval_minutes"] = types.Int64Value(intervalMinutes)
	}
	return objectValue(t, present)
}

func flattenHighFrequencyPlan(
	ctx context.Context,
	t types.ObjectType,
	plan externalEonSdkAPI.HighFrequencyBackupPolicyPlan,
	prior types.Object,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	resourceTypes := make([]string, 0, len(plan.ResourceTypes))
	for _, resourceType := range plan.ResourceTypes {
		resourceTypes = append(resourceTypes, string(resourceType.GetResourceType()))
	}
	resourceTypeList, listDiags := types.ListValueFrom(ctx, types.StringType, resourceTypes)
	diags.Append(listDiags...)

	scheduleType, typeDiags := nestedListElementType(t, "backup_schedules")
	diags.Append(typeDiags...)
	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	configType, typeDiags := nestedObjectType(scheduleType, "schedule_config")
	diags.Append(typeDiags...)
	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	schedules := make([]attr.Value, 0, len(plan.BackupSchedules))
	for index, schedule := range plan.BackupSchedules {
		priorSchedule := priorListElement(prior, "backup_schedules", index)

		config, configDiags := flattenHighFrequencyScheduleConfig(configType, schedule.ScheduleConfig, priorObject(priorSchedule, "schedule_config"))
		diags.Append(configDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}

		flattened, scheduleDiags := objectValue(scheduleType, map[string]attr.Value{
			"vault_id":        types.StringValue(schedule.VaultId),
			"retention_days":  types.Int64Value(int64(schedule.BackupRetentionDays)),
			"schedule_config": config,
		})
		diags.Append(scheduleDiags...)
		if diags.HasError() {
			return types.ObjectNull(t.AttrTypes), diags
		}
		schedules = append(schedules, flattened)
	}

	scheduleList, listDiags := types.ListValue(scheduleType, schedules)
	diags.Append(listDiags...)
	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	object, objectDiags := objectValue(t, map[string]attr.Value{
		"resource_types":   resourceTypeList,
		"backup_schedules": scheduleList,
	})
	diags.Append(objectDiags...)
	return object, diags
}

func flattenHighFrequencyScheduleConfig(
	t types.ObjectType,
	config externalEonSdkAPI.HighFrequencyBackupScheduleConfig,
	prior types.Object,
) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	frequency := config.GetFrequency()
	if frequency != externalEonSdkAPI.HIGH_FREQUENCY_BACKUP_SCHEDULE_INTERVAL {
		diags.AddError(
			"Unsupported Backup Schedule",
			fmt.Sprintf("Eon reports this high frequency schedule with frequency '%s'. This provider version "+
				"only models INTERVAL high frequency schedules.", string(frequency)),
		)
		return types.ObjectNull(t.AttrTypes), diags
	}

	interval := config.IntervalConfig.Get()
	if interval == nil {
		diags.AddError("Backup Policy Refresh Failed", "Eon reported an INTERVAL high frequency schedule without an interval config")
		return types.ObjectNull(t.AttrTypes), diags
	}

	intervalType, typeDiags := nestedObjectType(t, "interval_config")
	diags.Append(typeDiags...)
	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	intervalConfig, intervalDiags := flattenIntervalConfig(intervalType, int64(interval.IntervalMinutes), priorObject(prior, "interval_config"))
	diags.Append(intervalDiags...)
	if diags.HasError() {
		return types.ObjectNull(t.AttrTypes), diags
	}

	object, objectDiags := objectValue(t, map[string]attr.Value{
		"frequency":       types.StringValue(string(frequency)),
		"interval_config": intervalConfig,
	})
	diags.Append(objectDiags...)
	return object, diags
}
