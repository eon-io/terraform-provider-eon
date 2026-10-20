package provider

import (
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// labelsFromSet reads a configured labels set into the slice the SDK takes. A null or unknown set
// yields nil, which the callers treat as "no labels supplied".
func labelsFromSet(set types.Set) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}

	labels := make([]string, 0, len(set.Elements()))
	for _, element := range set.Elements() {
		if value, ok := element.(types.String); ok && !value.IsNull() && !value.IsUnknown() {
			labels = append(labels, value.ValueString())
		}
	}

	return labels
}

// labelsToSet turns the API's labels into state. Empty becomes null rather than an empty set: the
// field is omitempty on the wire, so a vault or policy with no labels is indistinguishable from one
// that never had them, and null is the only representation that round-trips without a perpetual diff.
func labelsToSet(labels []string) types.Set {
	if len(labels) == 0 {
		return types.SetNull(types.StringType)
	}

	elements := make([]attr.Value, 0, len(labels))
	for _, label := range labels {
		elements = append(elements, types.StringValue(label))
	}

	return types.SetValueMust(types.StringType, elements)
}

// labelsEqual compares two label sets irrespective of order, which the API does not preserve.
func labelsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	sortedA := slices.Clone(a)
	sortedB := slices.Clone(b)
	slices.Sort(sortedA)
	slices.Sort(sortedB)

	return slices.Equal(sortedA, sortedB)
}
