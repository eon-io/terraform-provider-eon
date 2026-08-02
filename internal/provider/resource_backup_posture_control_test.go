package provider

import (
	"context"
	"encoding/json"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const backupPostureControlAPIResponse = `{
  "id": "control-1",
  "name": "Production 3-2-1",
  "severity": "HIGH",
  "resourceSelector": {
    "resourceSelectionMode": "CONDITIONAL",
    "expression": {
      "group": {
        "operator": "AND",
        "operands": [
          {"environment": {"operator": "IN", "environments": ["PROD"]}},
          {"tagKeyValues": {"operator": "CONTAINS_ANY_OF", "tagKeyValues": [{"key": "team", "value": "core"}]}}
        ]
      }
    }
  },
  "rules": {
    "minimumRetention": [{"frequency": "DAILY", "minimumRetention": 30}],
    "numberOfCopies": {"minCopies": 2},
    "crossRegion": true
  }
}`

// The generic conversion maps Terraform's snake_case attributes onto the API's camelCase fields, so
// a mistyped attribute name silently drops a whole condition. Round-tripping a response back into a
// request payload fails loudly when that happens.
func TestBackupPostureControlStateRoundTrip(t *testing.T) {
	var control externalEonSdkAPI.BackupPostureControl
	require.NoError(t, json.Unmarshal([]byte(backupPostureControlAPIResponse), &control))

	var state BackupPostureControlResourceModel
	diags := backupPostureControlToState(context.Background(), &control, &state)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())

	assert.Equal(t, "control-1", state.Id.ValueString())
	assert.Equal(t, "Production 3-2-1", state.Name.ValueString())
	assert.Equal(t, "HIGH", state.Severity.ValueString())

	payload, diags := backupPostureControlPayload(context.Background(), &state)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())

	rendered, err := json.Marshal(payload)
	require.NoError(t, err)

	assert.JSONEq(t, `{
	  "name": "Production 3-2-1",
	  "severity": "HIGH",
	  "resourceSelector": {
	    "resourceSelectionMode": "CONDITIONAL",
	    "expression": {
	      "group": {
	        "operator": "AND",
	        "operands": [
	          {"environment": {"operator": "IN", "environments": ["PROD"]}},
	          {"tagKeyValues": {"operator": "CONTAINS_ANY_OF", "tagKeyValues": [{"key": "team", "value": "core"}]}}
	        ]
	      }
	    }
	  },
	  "rules": {
	    "minimumRetention": [{"frequency": "DAILY", "minimumRetention": 30}],
	    "numberOfCopies": {"minCopies": 2},
	    "crossRegion": true,
	    "crossAccount": false,
	    "crossCloudProvider": false
	  }
	}`, string(rendered))
}

func TestBackupPostureControlPayloadDecodesIntoSDKRequest(t *testing.T) {
	var control externalEonSdkAPI.BackupPostureControl
	require.NoError(t, json.Unmarshal([]byte(backupPostureControlAPIResponse), &control))

	var state BackupPostureControlResourceModel
	diags := backupPostureControlToState(context.Background(), &control, &state)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())

	payload, diags := backupPostureControlPayload(context.Background(), &state)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())

	var createReq externalEonSdkAPI.CreateBackupPostureControlRequest
	require.False(t, decodePayload(payload, &createReq).HasError())

	assert.Equal(t, externalEonSdkAPI.HIGH, createReq.Severity)

	selector := createReq.ResourceSelector.Get()
	require.NotNil(t, selector)
	assert.Equal(t, externalEonSdkAPI.RESOURCE_SELECTOR_MODE_CONDITIONAL, selector.ResourceSelectionMode)

	group := selector.Expression.Get().Group.Get()
	require.NotNil(t, group)
	assert.Equal(t, externalEonSdkAPI.AND_OPERATOR, group.Operator)
	require.Len(t, group.Operands, 2)
	assert.Equal(t, []externalEonSdkAPI.Environment{"PROD"}, group.Operands[0].Environment.Get().Environments)
	assert.Equal(t, "core", *group.Operands[1].TagKeyValues.Get().TagKeyValues[0].Value)

	rules := createReq.Rules.Get()
	require.NotNil(t, rules)
	require.Len(t, rules.MinimumRetention, 1)
	assert.Equal(t, int32(30), rules.MinimumRetention[0].MinimumRetention)
	assert.Equal(t, int32(2), rules.NumberOfCopies.Get().MinCopies)
	assert.True(t, *rules.CrossRegion)
	assert.False(t, *rules.CrossCloudProvider)
	// A rule the config leaves out must stay out of the request, or the API starts enforcing it.
	assert.False(t, rules.MaximumRetention.IsSet())
}

// A control that applies to every resource carries no expression, and no rule beyond the flags.
func TestBackupPostureControlAllResourcesSelector(t *testing.T) {
	var control externalEonSdkAPI.BackupPostureControl
	require.NoError(t, json.Unmarshal([]byte(`{
	  "id": "control-2",
	  "name": "Any copy",
	  "severity": "LOW",
	  "resourceSelector": {"resourceSelectionMode": "ALL"},
	  "rules": {"crossRegion": true}
	}`), &control))

	var state BackupPostureControlResourceModel
	diags := backupPostureControlToState(context.Background(), &control, &state)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())

	assert.True(t, state.ResourceSelector.Attributes()["expression"].IsNull())
	assert.True(t, state.Rules.Attributes()["minimum_retention"].IsNull())

	payload, diags := backupPostureControlPayload(context.Background(), &state)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())

	rendered, err := json.Marshal(payload)
	require.NoError(t, err)
	assert.JSONEq(t, `{
	  "name": "Any copy",
	  "severity": "LOW",
	  "resourceSelector": {"resourceSelectionMode": "ALL"},
	  "rules": {"crossRegion": true, "crossAccount": false, "crossCloudProvider": false}
	}`, string(rendered))
}

// The expression schema is generated from a table and unrolled per nesting level, so a bad
// attribute shape only surfaces when the framework validates the schema.
func TestBackupPostureControlSchemaIsValid(t *testing.T) {
	diags := backupPostureControlSchema.ValidateImplementation(context.Background())
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())

	expression := backupPostureControlSchema.Attributes["resource_selector"].(schema.SingleNestedAttribute).Attributes["expression"].(schema.SingleNestedAttribute)
	group := expression.Attributes["group"].(schema.SingleNestedAttribute)
	operand := group.Attributes["operands"].(schema.ListNestedAttribute).NestedObject.Attributes
	assert.Len(t, expression.Attributes, len(backupPostureControlConditions)+1)
	assert.Contains(t, operand, "group", "operands must accept a nested group")
	assert.NotContains(t,
		operand["group"].(schema.SingleNestedAttribute).Attributes["operands"].(schema.ListNestedAttribute).NestedObject.Attributes,
		"group",
		"nesting must stop at the configured depth",
	)
}

func TestCamelCase(t *testing.T) {
	for attribute, expected := range map[string]string{
		"resource_selection_mode": "resourceSelectionMode",
		"tag_key_values":          "tagKeyValues",
		"minimum_retention":       "minimumRetention",
		"min_copies":              "minCopies",
		"subnets":                 "subnets",
	} {
		assert.Equal(t, expected, camelCase(attribute))
	}
}
