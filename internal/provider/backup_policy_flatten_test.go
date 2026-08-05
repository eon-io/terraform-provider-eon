package provider

import (
	"context"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func policySchemaType(t *testing.T) types.ObjectType {
	t.Helper()

	policyResource := &BackupPolicyResource{}
	schemaType, ok := policyResource.schemaObjectType(context.Background())
	require.True(t, ok, "backup policy schema should be an object type")
	return schemaType
}

// nestedValue walks an object value along a path of attribute names, stepping into the element at
// index 0 whenever it meets a list, which is enough to reach any value these tests assert on.
func nestedValue(t *testing.T, object types.Object, path ...string) attr.Value {
	t.Helper()

	var current attr.Value = object
	for _, name := range path {
		if list, ok := current.(types.List); ok {
			elements := list.Elements()
			require.NotEmpty(t, elements, "list before %q is empty", name)
			current = elements[0]
		}

		parent, ok := current.(types.Object)
		require.True(t, ok, "value before %q is %T, expected an object", name, current)
		require.False(t, parent.IsNull(), "object before %q is null", name)

		value, ok := parent.Attributes()[name]
		require.True(t, ok, "no attribute %q", name)
		current = value
	}
	return current
}

func standardPolicy(scheduleConfig externalEonSdkAPI.StandardBackupScheduleConfig) *externalEonSdkAPI.BackupPolicy {
	selector := externalEonSdkAPI.NewBackupPolicyResourceSelector(externalEonSdkAPI.RESOURCE_SELECTOR_MODE_ALL)
	schedule := externalEonSdkAPI.NewStandardBackupSchedules("vault-1", scheduleConfig, 30)
	standardPlan := externalEonSdkAPI.NewStandardBackupPolicyPlan([]externalEonSdkAPI.StandardBackupSchedules{*schedule})
	plan := externalEonSdkAPI.NewBackupPolicyPlan(externalEonSdkAPI.BACKUP_POLICY_TYPE_STANDARD)
	plan.SetStandardPlan(*standardPlan)

	return externalEonSdkAPI.NewBackupPolicy("policy-1", "nightly", true, *selector, *plan)
}

func dailyScheduleConfig(hour, minute int32) externalEonSdkAPI.StandardBackupScheduleConfig {
	dailyConfig := externalEonSdkAPI.NewDailyConfig()
	dailyConfig.SetTimeOfDay(*externalEonSdkAPI.NewTimeOfDay(hour, minute))

	scheduleConfig := externalEonSdkAPI.NewStandardBackupScheduleConfig(externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_DAILY)
	scheduleConfig.SetDailyConfig(*dailyConfig)
	return *scheduleConfig
}

// TestFlattenBackupPolicy_RefreshesSelectorAndPlan covers the drift gap this file exists for: Read
// used to carry resource_selector and backup_plan over from prior state, so console-side edits never
// produced a plan diff (EON-16210).
func TestFlattenBackupPolicy_RefreshesSelectorAndPlan(t *testing.T) {
	t.Parallel()

	schemaType := policySchemaType(t)
	policy := standardPolicy(dailyScheduleConfig(3, 15))

	data := BackupPolicyResourceModel{}
	diags := flattenBackupPolicy(context.Background(), policy, schemaType, &data)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())

	assert.Equal(t, "policy-1", data.Id.ValueString())
	assert.Equal(t, "nightly", data.Name.ValueString())
	assert.True(t, data.Enabled.ValueBool())

	assert.Equal(t, types.StringValue("ALL"), nestedValue(t, data.ResourceSelector, "resource_selection_mode"))
	assert.Equal(t, types.StringValue("STANDARD"), nestedValue(t, data.BackupPlan, "backup_policy_type"))
	assert.Equal(t, types.StringValue("vault-1"), nestedValue(t, data.BackupPlan, "standard_plan", "backup_schedules", "vault_id"))
	assert.Equal(t, types.Int64Value(30), nestedValue(t, data.BackupPlan, "standard_plan", "backup_schedules", "retention_days"))
	assert.Equal(t, types.Int64Value(3), nestedValue(t, data.BackupPlan, "standard_plan", "backup_schedules", "schedule_config", "daily_config", "time_of_day_hour"))
	assert.Equal(t, types.Int64Value(15), nestedValue(t, data.BackupPlan, "standard_plan", "backup_schedules", "schedule_config", "daily_config", "time_of_day_minutes"))

	// Terraform-only bookkeeping: the API carries no timestamps, so refresh must not invent new ones.
	assert.True(t, data.CreatedAt.IsNull())
	assert.True(t, data.UpdatedAt.IsNull())
}

func TestFlattenBackupPolicy_GroupExpression(t *testing.T) {
	t.Parallel()

	operand := externalEonSdkAPI.NewBackupPolicyExpression()
	operand.SetResourceName(*externalEonSdkAPI.NewResourceNameCondition(
		externalEonSdkAPI.StringOperators("STARTS_WITH"), []string{"prod-"}))

	group := externalEonSdkAPI.NewBackupPolicyGroupCondition(
		externalEonSdkAPI.AND_OPERATOR, []externalEonSdkAPI.BackupPolicyExpression{*operand})

	expression := externalEonSdkAPI.NewBackupPolicyExpression()
	expression.SetGroup(*group)

	policy := standardPolicy(dailyScheduleConfig(1, 0))
	policy.ResourceSelector.ResourceSelectionMode = externalEonSdkAPI.RESOURCE_SELECTOR_MODE_CONDITIONAL
	policy.ResourceSelector.SetExpression(*expression)

	data := BackupPolicyResourceModel{}
	diags := flattenBackupPolicy(context.Background(), policy, policySchemaType(t), &data)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())

	assert.Equal(t, types.StringValue("AND"), nestedValue(t, data.ResourceSelector, "expression", "group", "operator"))
	assert.Equal(t, types.StringValue("STARTS_WITH"),
		nestedValue(t, data.ResourceSelector, "expression", "group", "operands", "resource_name", "operator"))

	names, ok := nestedValue(t, data.ResourceSelector, "expression", "group", "operands", "resource_name", "resource_names").(types.List)
	require.True(t, ok)
	assert.Equal(t, []attr.Value{types.StringValue("prod-")}, names.Elements())
}

// TestFlattenBackupPolicy_RejectsUnrepresentableCondition guards the other half of drift detection:
// silently dropping a condition the schema cannot express would refresh the resource into a state no
// configuration can reproduce, which reads as an undiffable plan forever.
func TestFlattenBackupPolicy_RejectsUnrepresentableCondition(t *testing.T) {
	t.Parallel()

	expression := externalEonSdkAPI.NewBackupPolicyExpression()
	expression.SetResourceName(*externalEonSdkAPI.NewResourceNameCondition(
		externalEonSdkAPI.StringOperators("CONTAINS"), []string{"db"}))

	policy := standardPolicy(dailyScheduleConfig(1, 0))
	policy.ResourceSelector.SetExpression(*expression)

	data := BackupPolicyResourceModel{}
	diags := flattenBackupPolicy(context.Background(), policy, policySchemaType(t), &data)
	require.True(t, diags.HasError(), "top-level resource_name should be rejected")
	assert.Contains(t, diags.Errors()[0].Detail(), "resource_name")
}

func TestFlattenBackupPolicy_RejectsUnsupportedPolicyType(t *testing.T) {
	t.Parallel()

	policy := standardPolicy(dailyScheduleConfig(1, 0))
	policy.BackupPlan.BackupPolicyType = externalEonSdkAPI.BACKUP_POLICY_TYPE_AWS_NATIVE_STANDARD

	data := BackupPolicyResourceModel{}
	diags := flattenBackupPolicy(context.Background(), policy, policySchemaType(t), &data)
	require.True(t, diags.HasError(), "AWS_NATIVE_STANDARD should be rejected")
	assert.Contains(t, diags.Errors()[0].Detail(), "AWS_NATIVE_STANDARD")
}

// TestFlattenIntervalConfig_KeepsConfiguredUnit pins the ambiguity that makes a naive refresh diff
// forever: the API stores a standard interval in hours, so a configuration written in minutes must
// come back in minutes.
func TestFlattenIntervalConfig_KeepsConfiguredUnit(t *testing.T) {
	t.Parallel()

	schemaType := policySchemaType(t)
	standardPlanType, diags := nestedObjectType(schemaType, "backup_plan")
	require.False(t, diags.HasError())
	standardPlanType, diags = nestedObjectType(standardPlanType, "standard_plan")
	require.False(t, diags.HasError())
	scheduleType, diags := nestedListElementType(standardPlanType, "backup_schedules")
	require.False(t, diags.HasError())
	configType, diags := nestedObjectType(scheduleType, "schedule_config")
	require.False(t, diags.HasError())
	intervalType, diags := nestedObjectType(configType, "interval_config")
	require.False(t, diags.HasError())

	priorWith := func(attrs map[string]attr.Value) types.Object {
		object, objectDiags := objectValue(intervalType, attrs)
		require.False(t, objectDiags.HasError())
		return object
	}

	tests := []struct {
		name            string
		prior           types.Object
		expectedMinutes types.Int64
		expectedHours   types.Int64
		expectedWindow  types.Int64
	}{
		{
			name:            "configuration used minutes",
			prior:           priorWith(map[string]attr.Value{"interval_minutes": types.Int64Value(720)}),
			expectedMinutes: types.Int64Value(720),
			expectedHours:   types.Int64Null(),
			expectedWindow:  types.Int64Null(),
		},
		{
			name:            "configuration used hours",
			prior:           priorWith(map[string]attr.Value{"interval_hours": types.Int64Value(12)}),
			expectedMinutes: types.Int64Null(),
			expectedHours:   types.Int64Value(12),
			expectedWindow:  types.Int64Null(),
		},
		{
			name: "start window has no API field and is carried over",
			prior: priorWith(map[string]attr.Value{
				"interval_hours":       types.Int64Value(12),
				"start_window_minutes": types.Int64Value(45),
			}),
			expectedMinutes: types.Int64Null(),
			expectedHours:   types.Int64Value(12),
			expectedWindow:  types.Int64Value(45),
		},
		{
			name:            "no prior state falls back to hours",
			prior:           types.ObjectNull(intervalType.AttrTypes),
			expectedMinutes: types.Int64Null(),
			expectedHours:   types.Int64Value(12),
			expectedWindow:  types.Int64Null(),
		},
		{
			name:            "prior no longer matches the API interval",
			prior:           priorWith(map[string]attr.Value{"interval_hours": types.Int64Value(6)}),
			expectedMinutes: types.Int64Null(),
			expectedHours:   types.Int64Value(12),
			expectedWindow:  types.Int64Null(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			flattened, flattenDiags := flattenIntervalConfig(intervalType, 720, tt.prior)
			require.False(t, flattenDiags.HasError(), "unexpected diagnostics: %v", flattenDiags.Errors())

			assert.Equal(t, tt.expectedMinutes, flattened.Attributes()["interval_minutes"])
			assert.Equal(t, tt.expectedHours, flattened.Attributes()["interval_hours"])
			assert.Equal(t, tt.expectedWindow, flattened.Attributes()["start_window_minutes"])
		})
	}
}

func TestFlattenScheduleTimezone(t *testing.T) {
	t.Parallel()

	utc := externalEonSdkAPI.SCHEDULE_TIMEZONE_UTC
	resourceLocal := externalEonSdkAPI.SCHEDULE_TIMEZONE_RESOURCE

	tests := []struct {
		name     string
		timezone *externalEonSdkAPI.ScheduleTimezone
		prior    types.String
		expected types.String
	}{
		{
			name:     "API default stays unset when the configuration omitted it",
			timezone: &utc,
			prior:    types.StringNull(),
			expected: types.StringNull(),
		},
		{
			name:     "explicit UTC is kept",
			timezone: &utc,
			prior:    types.StringValue("UTC"),
			expected: types.StringValue("UTC"),
		},
		{
			name:     "drift away from the default is reported",
			timezone: &resourceLocal,
			prior:    types.StringNull(),
			expected: types.StringValue("RESOURCE"),
		},
		{
			name:     "absent timezone is null",
			timezone: nil,
			prior:    types.StringValue("UTC"),
			expected: types.StringNull(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, flattenScheduleTimezone(tt.timezone, tt.prior))
		})
	}
}

// TestFlattenBackupPolicy_MonthlyAndAnnually covers the two schedule kinds whose day and month were
// never sent to the API, so a refresh had nothing to compare against.
func TestFlattenBackupPolicy_MonthlyAndAnnually(t *testing.T) {
	t.Parallel()

	monthlyConfig := externalEonSdkAPI.NewMonthlyConfig()
	monthlyConfig.SetTimeOfDay(*externalEonSdkAPI.NewTimeOfDay(2, 30))
	monthlyConfig.SetDaysOfMonth([]int32{14})
	monthly := externalEonSdkAPI.NewStandardBackupScheduleConfig(externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_MONTHLY)
	monthly.SetMonthlyConfig(*monthlyConfig)

	annuallyConfig := externalEonSdkAPI.NewAnnuallyConfig()
	annuallyConfig.SetTimeOfDay(*externalEonSdkAPI.NewTimeOfDay(4, 0))
	annuallyConfig.SetTimeOfYear(*externalEonSdkAPI.NewTimeOfYear(9, 21))
	annually := externalEonSdkAPI.NewStandardBackupScheduleConfig(externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_ANNUALLY)
	annually.SetAnnuallyConfig(*annuallyConfig)

	schemaType := policySchemaType(t)

	monthlyData := BackupPolicyResourceModel{}
	diags := flattenBackupPolicy(context.Background(), standardPolicy(*monthly), schemaType, &monthlyData)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
	assert.Equal(t, types.Int64Value(14),
		nestedValue(t, monthlyData.BackupPlan, "standard_plan", "backup_schedules", "schedule_config", "monthly_config", "day_of_month"))

	annuallyData := BackupPolicyResourceModel{}
	diags = flattenBackupPolicy(context.Background(), standardPolicy(*annually), schemaType, &annuallyData)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
	assert.Equal(t, types.StringValue("SEPTEMBER"),
		nestedValue(t, annuallyData.BackupPlan, "standard_plan", "backup_schedules", "schedule_config", "annually_config", "month"))
	assert.Equal(t, types.Int64Value(21),
		nestedValue(t, annuallyData.BackupPlan, "standard_plan", "backup_schedules", "schedule_config", "annually_config", "day_of_month"))
}

func TestFlattenBackupPolicy_HighFrequencyPlan(t *testing.T) {
	t.Parallel()

	intervalConfig := externalEonSdkAPI.NewHighFrequencyIntervalConfig(15)
	scheduleConfig := externalEonSdkAPI.NewHighFrequencyBackupScheduleConfig()
	scheduleConfig.SetFrequency(externalEonSdkAPI.HIGH_FREQUENCY_BACKUP_SCHEDULE_INTERVAL)
	scheduleConfig.SetIntervalConfig(*intervalConfig)

	resourceType := externalEonSdkAPI.NewHighFrequencyBackupResourceType()
	resourceType.SetResourceType(externalEonSdkAPI.ResourceType("AWS_EBS"))

	schedule := externalEonSdkAPI.NewHighFrequencyBackupSchedules("vault-hf", *scheduleConfig, 7)
	highFrequencyPlan := externalEonSdkAPI.NewHighFrequencyBackupPolicyPlan(
		[]externalEonSdkAPI.HighFrequencyBackupResourceType{*resourceType},
		[]externalEonSdkAPI.HighFrequencyBackupSchedules{*schedule},
	)

	plan := externalEonSdkAPI.NewBackupPolicyPlan(externalEonSdkAPI.BACKUP_POLICY_TYPE_HIGH_FREQUENCY)
	plan.SetHighFrequencyPlan(*highFrequencyPlan)

	selector := externalEonSdkAPI.NewBackupPolicyResourceSelector(externalEonSdkAPI.RESOURCE_SELECTOR_MODE_ALL)
	policy := externalEonSdkAPI.NewBackupPolicy("policy-hf", "every-15", true, *selector, *plan)

	data := BackupPolicyResourceModel{}
	diags := flattenBackupPolicy(context.Background(), policy, policySchemaType(t), &data)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())

	assert.Equal(t, types.Int64Value(15),
		nestedValue(t, data.BackupPlan, "high_frequency_plan", "backup_schedules", "schedule_config", "interval_config", "interval_minutes"))

	resourceTypes, ok := nestedValue(t, data.BackupPlan, "high_frequency_plan", "resource_types").(types.List)
	require.True(t, ok)
	assert.Equal(t, []attr.Value{types.StringValue("AWS_EBS")}, resourceTypes.Elements())
}

func TestMonthNameAndNumberRoundTrip(t *testing.T) {
	t.Parallel()

	for month := int32(1); month <= 12; month++ {
		name, err := monthName(month)
		require.NoError(t, err)

		number, err := monthNumber(name)
		require.NoError(t, err)
		assert.Equal(t, month, number, "%s should map back to %d", name, month)
	}

	_, err := monthName(0)
	assert.Error(t, err)
	_, err = monthName(13)
	assert.Error(t, err)
	_, err = monthNumber("SMARCH")
	assert.Error(t, err)
}

// TestObjectAttributes_RejectsNullObject pins the crash the customer hit: a null object answers
// Attributes() with an empty map, and the old code asserted the concrete type of the resulting nil.
func TestObjectAttributes_RejectsNullObject(t *testing.T) {
	t.Parallel()

	attrs, err := objectAttributes(types.ObjectNull(map[string]attr.Type{"resource_selection_mode": types.StringType}), "resource_selector")
	require.Error(t, err, "a null object must be reported, not indexed")
	assert.Nil(t, attrs)
	assert.Contains(t, err.Error(), "ignore_changes")

	populated, objectDiags := types.ObjectValue(
		map[string]attr.Type{"resource_selection_mode": types.StringType},
		map[string]attr.Value{"resource_selection_mode": types.StringValue("ALL")},
	)
	require.False(t, objectDiags.HasError())

	attrs, objectErr := objectAttributes(populated, "resource_selector")
	require.NoError(t, objectErr)

	mode, modeErr := stringAttr(attrs, "resource_selector", "resource_selection_mode")
	require.NoError(t, modeErr)
	assert.Equal(t, "ALL", mode.ValueString())

	_, missingErr := stringAttr(attrs, "resource_selector", "nope")
	assert.Error(t, missingErr)
}
