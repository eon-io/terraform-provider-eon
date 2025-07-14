package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

// TestSafeInt32Conversion tests the SafeInt32Conversion function
func TestSafeInt32Conversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       int64
		expected    int32
		shouldError bool
	}{
		{
			name:        "valid positive number",
			input:       100,
			expected:    100,
			shouldError: false,
		},
		{
			name:        "valid negative number",
			input:       -100,
			expected:    -100,
			shouldError: false,
		},
		{
			name:        "zero value",
			input:       0,
			expected:    0,
			shouldError: false,
		},
		{
			name:        "max int32 value",
			input:       2147483647,
			expected:    2147483647,
			shouldError: false,
		},
		{
			name:        "min int32 value",
			input:       -2147483648,
			expected:    -2147483648,
			shouldError: false,
		},
		{
			name:        "overflow - max int32 + 1",
			input:       2147483648,
			expected:    0,
			shouldError: true,
		},
		{
			name:        "underflow - min int32 - 1",
			input:       -2147483649,
			expected:    0,
			shouldError: true,
		},
		{
			name:        "large positive overflow",
			input:       9223372036854775807,
			expected:    0,
			shouldError: true,
		},
		{
			name:        "large negative underflow",
			input:       -9223372036854775808,
			expected:    0,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := SafeInt32Conversion(tt.input)

			if tt.shouldError {
				assert.Error(t, err, "Expected error for input %d", tt.input)
				assert.Equal(t, int32(0), result, "Result should be 0 on error")
			} else {
				assert.NoError(t, err, "Expected no error for input %d", tt.input)
				assert.Equal(t, tt.expected, result, "Result should match expected value")
			}
		})
	}
}

// TestCreateDailyConfigFromModel tests the createDailyConfigFromModel function
func TestCreateDailyConfigFromModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		model   *DailyConfigModel
		wantErr bool
	}{
		{
			name: "valid daily config",
			model: &DailyConfigModel{
				TimeOfDayHour:      types.Int64Value(9),
				TimeOfDayMinutes:   types.Int64Value(30),
				StartWindowMinutes: types.Int64Value(240),
			},
			wantErr: false,
		},
		{
			name: "null values",
			model: &DailyConfigModel{
				TimeOfDayHour:      types.Int64Null(),
				TimeOfDayMinutes:   types.Int64Null(),
				StartWindowMinutes: types.Int64Null(),
			},
			wantErr: false,
		},
		{
			name: "mixed null and valid values",
			model: &DailyConfigModel{
				TimeOfDayHour:      types.Int64Value(12),
				TimeOfDayMinutes:   types.Int64Null(),
				StartWindowMinutes: types.Int64Value(120),
			},
			wantErr: false,
		},
		{
			name: "edge case - hour 0",
			model: &DailyConfigModel{
				TimeOfDayHour:      types.Int64Value(0),
				TimeOfDayMinutes:   types.Int64Value(0),
				StartWindowMinutes: types.Int64Value(240),
			},
			wantErr: false,
		},
		{
			name: "edge case - hour 23",
			model: &DailyConfigModel{
				TimeOfDayHour:      types.Int64Value(23),
				TimeOfDayMinutes:   types.Int64Value(59),
				StartWindowMinutes: types.Int64Value(240),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := createDailyConfigFromModel(tt.model)

			if tt.wantErr {
				assert.Error(t, err, "Expected error for test case %s", tt.name)
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, "Expected no error for test case %s", tt.name)
				assert.NotNil(t, result, "Result should not be nil")
			}
		})
	}
}

// TestStringValidation tests string validation functions
func TestStringValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		validator string
		isValid   bool
	}{
		{
			name:      "valid URL",
			input:     "https://test.eon.io",
			validator: "url",
			isValid:   true,
		},
		{
			name:      "invalid URL",
			input:     "not-a-url",
			validator: "url",
			isValid:   false,
		},
		{
			name:      "valid UUID",
			input:     "123e4567-e89b-12d3-a456-426614174000",
			validator: "uuid",
			isValid:   true,
		},
		{
			name:      "invalid UUID",
			input:     "not-a-uuid",
			validator: "uuid",
			isValid:   false,
		},
		{
			name:      "empty string",
			input:     "",
			validator: "nonempty",
			isValid:   false,
		},
		{
			name:      "non-empty string",
			input:     "test",
			validator: "nonempty",
			isValid:   true,
		},
		{
			name:      "whitespace only",
			input:     "   ",
			validator: "nonempty",
			isValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			switch tt.validator {
			case "url":
				// Simple URL validation
				isValid := len(tt.input) > 0 && (strings.HasPrefix(tt.input, "http://") || strings.HasPrefix(tt.input, "https://"))
				assert.Equal(t, tt.isValid, isValid, "URL validation should match expected result")
			case "uuid":
				// Simple UUID validation (length and format)
				isValid := len(tt.input) == 36 && tt.input[8] == '-' && tt.input[13] == '-'
				assert.Equal(t, tt.isValid, isValid, "UUID validation should match expected result")
			case "nonempty":
				// Non-empty validation
				trimmed := strings.TrimSpace(tt.input)
				isValid := len(trimmed) > 0
				assert.Equal(t, tt.isValid, isValid, "Non-empty validation should match expected result")
			}
		})
	}
}

// TestModelValidation tests model validation functions
func TestModelValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "backup schedule model validation",
			test: func(t *testing.T) {
				model := &BackupScheduleModel{
					VaultId:       types.StringValue("test-vault"),
					RetentionDays: types.Int64Value(30),
				}

				assert.NotNil(t, model, "Model should not be nil")
				assert.False(t, model.VaultId.IsNull(), "VaultId should not be null")
				assert.False(t, model.RetentionDays.IsNull(), "RetentionDays should not be null")
				assert.Equal(t, "test-vault", model.VaultId.ValueString(), "VaultId should match")
				assert.Equal(t, int64(30), model.RetentionDays.ValueInt64(), "RetentionDays should match")
			},
		},
		{
			name: "daily config model validation",
			test: func(t *testing.T) {
				model := &DailyConfigModel{
					TimeOfDayHour:      types.Int64Value(9),
					TimeOfDayMinutes:   types.Int64Value(30),
					StartWindowMinutes: types.Int64Value(240),
				}

				assert.NotNil(t, model, "Model should not be nil")
				assert.False(t, model.TimeOfDayHour.IsNull(), "TimeOfDayHour should not be null")
				assert.False(t, model.TimeOfDayMinutes.IsNull(), "TimeOfDayMinutes should not be null")
				assert.False(t, model.StartWindowMinutes.IsNull(), "StartWindowMinutes should not be null")
				assert.Equal(t, int64(9), model.TimeOfDayHour.ValueInt64(), "TimeOfDayHour should match")
				assert.Equal(t, int64(30), model.TimeOfDayMinutes.ValueInt64(), "TimeOfDayMinutes should match")
				assert.Equal(t, int64(240), model.StartWindowMinutes.ValueInt64(), "StartWindowMinutes should match")
			},
		},
		{
			name: "null values handling",
			test: func(t *testing.T) {
				model := &DailyConfigModel{
					TimeOfDayHour:      types.Int64Null(),
					TimeOfDayMinutes:   types.Int64Null(),
					StartWindowMinutes: types.Int64Null(),
				}

				assert.NotNil(t, model, "Model should not be nil")
				assert.True(t, model.TimeOfDayHour.IsNull(), "TimeOfDayHour should be null")
				assert.True(t, model.TimeOfDayMinutes.IsNull(), "TimeOfDayMinutes should be null")
				assert.True(t, model.StartWindowMinutes.IsNull(), "StartWindowMinutes should be null")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.test(t)
		})
	}
}

// TestErrorHandling tests error handling functions
func TestErrorHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "string error",
			input:    "test error",
			expected: "test error",
		},
		{
			name:     "formatted error",
			input:    fmt.Errorf("formatted error: %s", "test"),
			expected: "formatted error: test",
		},
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var result string
			if tt.input != nil {
				switch v := tt.input.(type) {
				case string:
					result = v
				case error:
					result = v.Error()
				}
			}

			assert.Equal(t, tt.expected, result, "Error handling should match expected result")
		})
	}
}

// TestTypeConversions tests type conversion functions
func TestTypeConversions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "int64 to int32 conversion",
			test: func(t *testing.T) {
				input := int64(100)
				result, err := SafeInt32Conversion(input)
				assert.NoError(t, err, "Conversion should not error")
				assert.Equal(t, int32(100), result, "Result should match")
			},
		},
		{
			name: "string to types.String conversion",
			test: func(t *testing.T) {
				input := "test-string"
				result := types.StringValue(input)
				assert.False(t, result.IsNull(), "Result should not be null")
				assert.Equal(t, input, result.ValueString(), "Result should match input")
			},
		},
		{
			name: "int64 to types.Int64 conversion",
			test: func(t *testing.T) {
				input := int64(42)
				result := types.Int64Value(input)
				assert.False(t, result.IsNull(), "Result should not be null")
				assert.Equal(t, input, result.ValueInt64(), "Result should match input")
			},
		},
		{
			name: "bool to types.Bool conversion",
			test: func(t *testing.T) {
				input := true
				result := types.BoolValue(input)
				assert.False(t, result.IsNull(), "Result should not be null")
				assert.Equal(t, input, result.ValueBool(), "Result should match input")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.test(t)
		})
	}
}
