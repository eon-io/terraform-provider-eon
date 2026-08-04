package provider

import (
	"fmt"
	"math"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SafeInt32Conversion performs bounds checking for int64 to int32 conversion
func SafeInt32Conversion(value int64) (int32, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, fmt.Errorf("value %d out of int32 bounds", value)
	}
	return int32(value), nil
}

// objectAttributes returns obj's attributes, refusing a null or unknown object. A null
// types.Object still answers Attributes() with an empty map, so callers that index the result and
// assert the element's concrete type crash the provider on a nil attr.Value instead of reporting
// something a practitioner can act on. Terraform hands a null object whenever the attribute is
// absent from the plan, which lifecycle.ignore_changes on that attribute is enough to cause.
func objectAttributes(obj types.Object, path string) (map[string]attr.Value, error) {
	if obj.IsNull() || obj.IsUnknown() {
		return nil, fmt.Errorf(
			"%s has no value in the plan. This happens when the attribute is excluded from the plan "+
				"(for example by lifecycle.ignore_changes) while other attributes change, or when it was "+
				"never written to state by an import. Set %s explicitly, or drop it from ignore_changes",
			path, path)
	}
	return obj.Attributes(), nil
}

func attrValue(attrs map[string]attr.Value, parent, name string) (attr.Value, error) {
	value, ok := attrs[name]
	if !ok || value == nil {
		return nil, fmt.Errorf("%s.%s is missing from the plan", parent, name)
	}
	return value, nil
}

func stringAttr(attrs map[string]attr.Value, parent, name string) (types.String, error) {
	value, err := attrValue(attrs, parent, name)
	if err != nil {
		return types.StringNull(), err
	}
	typed, ok := value.(types.String)
	if !ok {
		return types.StringNull(), fmt.Errorf("%s.%s is %T, expected a string", parent, name, value)
	}
	return typed, nil
}

// requiredPlanObject fetches the plan variant that backupPolicyType selects, rejecting the null
// object Terraform supplies for the other, unconfigured variants.
func requiredPlanObject(backupPlanAttrs map[string]attr.Value, name, backupPolicyType string) (types.Object, error) {
	planObj, err := objectAttr(backupPlanAttrs, "backup_plan", name)
	if err != nil {
		return types.ObjectNull(nil), err
	}
	if planObj.IsNull() || planObj.IsUnknown() {
		return types.ObjectNull(nil), fmt.Errorf(
			"backup_plan.%s is required when backup_policy_type is '%s'", name, backupPolicyType)
	}
	return planObj, nil
}

func objectAttr(attrs map[string]attr.Value, parent, name string) (types.Object, error) {
	value, err := attrValue(attrs, parent, name)
	if err != nil {
		return types.ObjectNull(nil), err
	}
	typed, ok := value.(types.Object)
	if !ok {
		return types.ObjectNull(nil), fmt.Errorf("%s.%s is %T, expected an object", parent, name, value)
	}
	return typed, nil
}
