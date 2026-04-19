package provider

import (
	"context"
	"fmt"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/stretchr/testify/assert"
)

// TestBackupPolicyResource_Unit tests the backup policy resource without API calls
func TestBackupPolicyResource_Unit(t *testing.T) {
	t.Parallel()

	resource := NewBackupPolicyResource()
	assert.NotNil(t, resource, "Resource should not be nil")
}

// TestBackupPolicyResource_CreateWithMockClient tests creation with mock client
func TestBackupPolicyResource_CreateWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		shouldFail      bool
		expectedName    string
		expectedEnabled bool
	}{
		{
			name:            "successful creation",
			shouldFail:      false,
			expectedName:    "test-policy",
			expectedEnabled: true,
		},
		{
			name:            "creation failure",
			shouldFail:      true,
			expectedName:    "failing-policy",
			expectedEnabled: false,
		},
		{
			name:            "disabled policy creation",
			shouldFail:      false,
			expectedName:    "disabled-policy",
			expectedEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailCreate = tt.shouldFail

			// Create test request
			req := externalEonSdkAPI.NewCreateBackupPolicyRequest(
				tt.expectedName,
				*externalEonSdkAPI.NewBackupPolicyResourceSelector(
					externalEonSdkAPI.ResourceSelectorMode("ALL"),
				),
				*externalEonSdkAPI.NewBackupPolicyPlan(
					externalEonSdkAPI.BackupPolicyType("STANDARD"),
				),
			)
			req.SetEnabled(tt.expectedEnabled)

			// Test creation
			result, err := mockClient.CreateBackupPolicy(context.Background(), *req)

			if tt.shouldFail {
				assert.Error(t, err, "Expected error for failing test case")
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, "Expected no error for successful test case")
				assert.NotNil(t, result, "Result should not be nil")
				assert.Equal(t, tt.expectedName, result.Name, "Name should match")
				assert.Equal(t, tt.expectedEnabled, result.Enabled, "Enabled should match")
				assert.NotEmpty(t, result.Id, "ID should not be empty")
			}

			// Verify call count
			assert.Equal(t, 1, mockClient.CreateCalls, "Should have made one create call")
		})
	}
}

// TestBackupPolicyResource_ReadWithMockClient tests reading with mock client
func TestBackupPolicyResource_ReadWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		shouldFail bool
		policyID   string
	}{
		{
			name:       "successful read",
			shouldFail: false,
			policyID:   "test-policy-id",
		},
		{
			name:       "read failure",
			shouldFail: true,
			policyID:   "failing-policy-id",
		},
		{
			name:       "non-existent policy",
			shouldFail: false,
			policyID:   "non-existent-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailRead = tt.shouldFail

			// Add mock policy if it should exist
			if tt.policyID != "non-existent-id" && !tt.shouldFail {
				mockPolicy := &externalEonSdkAPI.BackupPolicy{
					Id:      tt.policyID,
					Name:    "test-policy",
					Enabled: true,
				}
				mockClient.AddMockPolicy(mockPolicy)
			}

			// Test reading
			result, err := mockClient.ReadBackupPolicy(context.Background(), tt.policyID)

			if tt.shouldFail {
				assert.Error(t, err, "Expected error for failing test case")
				assert.Nil(t, result, "Result should be nil on error")
			} else if tt.policyID == "non-existent-id" {
				assert.Error(t, err, "Expected error for non-existent policy")
				assert.Nil(t, result, "Result should be nil for non-existent policy")
			} else {
				assert.NoError(t, err, "Expected no error for successful test case")
				assert.NotNil(t, result, "Result should not be nil")
				assert.Equal(t, tt.policyID, result.Id, "ID should match")
			}

			// Verify call count
			assert.Equal(t, 1, mockClient.ReadCalls, "Should have made one read call")
		})
	}
}

// TestBackupPolicyResource_UpdateWithMockClient tests updating with mock client
func TestBackupPolicyResource_UpdateWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		shouldFail bool
		policyID   string
		newName    string
		newEnabled bool
	}{
		{
			name:       "successful update",
			shouldFail: false,
			policyID:   "test-policy-id",
			newName:    "updated-policy",
			newEnabled: false,
		},
		{
			name:       "update failure",
			shouldFail: true,
			policyID:   "failing-policy-id",
			newName:    "failing-update",
			newEnabled: true,
		},
		{
			name:       "non-existent policy update",
			shouldFail: false,
			policyID:   "non-existent-id",
			newName:    "non-existent-update",
			newEnabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailUpdate = tt.shouldFail

			// Add mock policy if it should exist
			if tt.policyID != "non-existent-id" {
				mockPolicy := &externalEonSdkAPI.BackupPolicy{
					Id:      tt.policyID,
					Name:    "original-policy",
					Enabled: true,
				}
				mockClient.AddMockPolicy(mockPolicy)
			}

			// Create update request
			req := externalEonSdkAPI.NewUpdateBackupPolicyRequest(
				tt.newName,
				*externalEonSdkAPI.NewBackupPolicyResourceSelector(
					externalEonSdkAPI.ResourceSelectorMode("ALL"),
				),
				*externalEonSdkAPI.NewBackupPolicyPlan(
					externalEonSdkAPI.BackupPolicyType("STANDARD"),
				),
			)
			req.SetEnabled(tt.newEnabled)

			// Test updating
			result, err := mockClient.UpdateBackupPolicy(context.Background(), tt.policyID, *req)

			if tt.shouldFail {
				assert.Error(t, err, "Expected error for failing test case")
				assert.Nil(t, result, "Result should be nil on error")
			} else if tt.policyID == "non-existent-id" {
				assert.Error(t, err, "Expected error for non-existent policy")
				assert.Nil(t, result, "Result should be nil for non-existent policy")
			} else {
				assert.NoError(t, err, "Expected no error for successful test case")
				assert.NotNil(t, result, "Result should not be nil")
				assert.Equal(t, tt.policyID, result.Id, "ID should match")
				assert.Equal(t, tt.newName, result.Name, "Name should be updated")
				assert.Equal(t, tt.newEnabled, result.Enabled, "Enabled should be updated")
			}

			// Verify call count
			assert.Equal(t, 1, mockClient.UpdateCalls, "Should have made one update call")
		})
	}
}

// TestBackupPolicyResource_DeleteWithMockClient tests deleting with mock client
func TestBackupPolicyResource_DeleteWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		shouldFail bool
		policyID   string
	}{
		{
			name:       "successful deletion",
			shouldFail: false,
			policyID:   "test-policy-id",
		},
		{
			name:       "deletion failure",
			shouldFail: true,
			policyID:   "failing-policy-id",
		},
		{
			name:       "non-existent policy deletion",
			shouldFail: false,
			policyID:   "non-existent-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailDelete = tt.shouldFail

			// Add mock policy if it should exist
			if tt.policyID != "non-existent-id" {
				mockPolicy := &externalEonSdkAPI.BackupPolicy{
					Id:      tt.policyID,
					Name:    "test-policy",
					Enabled: true,
				}
				mockClient.AddMockPolicy(mockPolicy)
			}

			// Test deletion
			err := mockClient.DeleteBackupPolicy(context.Background(), tt.policyID)

			if tt.shouldFail {
				assert.Error(t, err, "Expected error for failing test case")
			} else if tt.policyID == "non-existent-id" {
				assert.Error(t, err, "Expected error for non-existent policy")
			} else {
				assert.NoError(t, err, "Expected no error for successful test case")

				// Verify policy was deleted
				_, exists := mockClient.GetMockPolicy(tt.policyID)
				assert.False(t, exists, "Policy should no longer exist after deletion")
			}

			// Verify call count
			assert.Equal(t, 1, mockClient.DeleteCalls, "Should have made one delete call")
		})
	}
}

// TestBackupPolicyResource_ListWithMockClient tests listing with mock client
func TestBackupPolicyResource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shouldFail  bool
		numPolicies int
	}{
		{
			name:        "successful list with policies",
			shouldFail:  false,
			numPolicies: 3,
		},
		{
			name:        "successful list with no policies",
			shouldFail:  false,
			numPolicies: 0,
		},
		{
			name:        "list failure",
			shouldFail:  true,
			numPolicies: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailList = tt.shouldFail

			// Add mock policies
			for i := 0; i < tt.numPolicies; i++ {
				mockPolicy := &externalEonSdkAPI.BackupPolicy{
					Id:      fmt.Sprintf("policy-%d", i),
					Name:    fmt.Sprintf("test-policy-%d", i),
					Enabled: true,
				}
				mockClient.AddMockPolicy(mockPolicy)
			}

			// Test listing
			result, err := mockClient.ListBackupPolicies(context.Background())

			if tt.shouldFail {
				assert.Error(t, err, "Expected error for failing test case")
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, "Expected no error for successful test case")
				assert.NotNil(t, result, "Result should not be nil")
				assert.Len(t, result, tt.numPolicies, "Should return correct number of policies")
			}

			// Verify call count
			assert.Equal(t, 1, mockClient.ListCalls, "Should have made one list call")
		})
	}
}

// TestBackupPolicyResource_MockClientReset tests mock client reset functionality
func TestBackupPolicyResource_MockClientReset(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	// Add some data and make calls
	mockPolicy := &externalEonSdkAPI.BackupPolicy{
		Id:      "test-id",
		Name:    "test-policy",
		Enabled: true,
	}
	mockClient.AddMockPolicy(mockPolicy)

	// Make some API calls
	_, _ = mockClient.CreateBackupPolicy(context.Background(), *externalEonSdkAPI.NewCreateBackupPolicyRequest(
		"test",
		*externalEonSdkAPI.NewBackupPolicyResourceSelector(externalEonSdkAPI.ResourceSelectorMode("ALL")),
		*externalEonSdkAPI.NewBackupPolicyPlan(externalEonSdkAPI.BackupPolicyType("STANDARD")),
	))
	_, _ = mockClient.ReadBackupPolicy(context.Background(), "test-id")
	_, _ = mockClient.ListBackupPolicies(context.Background())

	// Verify data and calls exist
	assert.Equal(t, 1, mockClient.CreateCalls, "Should have create calls")
	assert.Equal(t, 1, mockClient.ReadCalls, "Should have read calls")
	assert.Equal(t, 1, mockClient.ListCalls, "Should have list calls")
	assert.Len(t, mockClient.BackupPolicies, 2, "Should have policies") // 1 added + 1 created

	// Reset the mock client
	mockClient.Reset()

	// Verify everything is reset
	assert.Equal(t, 0, mockClient.CreateCalls, "Create calls should be reset")
	assert.Equal(t, 0, mockClient.ReadCalls, "Read calls should be reset")
	assert.Equal(t, 0, mockClient.UpdateCalls, "Update calls should be reset")
	assert.Equal(t, 0, mockClient.DeleteCalls, "Delete calls should be reset")
	assert.Equal(t, 0, mockClient.ListCalls, "List calls should be reset")
	assert.Len(t, mockClient.BackupPolicies, 0, "Policies should be reset")
	assert.False(t, mockClient.ShouldFailCreate, "Failure flags should be reset")
	assert.False(t, mockClient.ShouldFailRead, "Failure flags should be reset")
	assert.False(t, mockClient.ShouldFailUpdate, "Failure flags should be reset")
	assert.False(t, mockClient.ShouldFailDelete, "Failure flags should be reset")
	assert.False(t, mockClient.ShouldFailList, "Failure flags should be reset")
}

// TestSafeInt32ConversionInResource tests the utility function used in resource
func TestSafeInt32ConversionInResource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       int64
		expected    int32
		shouldError bool
	}{
		{
			name:        "valid conversion",
			input:       100,
			expected:    100,
			shouldError: false,
		},
		{
			name:        "overflow",
			input:       2147483648,
			expected:    0,
			shouldError: true,
		},
		{
			name:        "underflow",
			input:       -2147483649,
			expected:    0,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := SafeInt32Conversion(tt.input)

			if tt.shouldError {
				assert.Error(t, err, "Expected error for overflow/underflow")
				assert.Equal(t, int32(0), result, "Result should be 0 on error")
			} else {
				assert.NoError(t, err, "Expected no error for valid conversion")
				assert.Equal(t, tt.expected, result, "Result should match expected value")
			}
		})
	}
}

// TestBackupPolicyResource_MockClientOperations tests all mock client operations
func TestBackupPolicyResource_MockClientOperations(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	// Test Create
	createReq := externalEonSdkAPI.NewCreateBackupPolicyRequest(
		"test-policy",
		*externalEonSdkAPI.NewBackupPolicyResourceSelector(
			externalEonSdkAPI.ResourceSelectorMode("ALL"),
		),
		*externalEonSdkAPI.NewBackupPolicyPlan(
			externalEonSdkAPI.BackupPolicyType("STANDARD"),
		),
	)
	createReq.SetEnabled(true)

	policy, err := mockClient.CreateBackupPolicy(context.Background(), *createReq)
	assert.NoError(t, err, "Create should not error")
	assert.NotNil(t, policy, "Policy should not be nil")
	assert.Equal(t, "test-policy", policy.Name, "Name should match")
	assert.True(t, policy.Enabled, "Should be enabled")

	// Test Read
	readPolicy, err := mockClient.ReadBackupPolicy(context.Background(), policy.Id)
	assert.NoError(t, err, "Read should not error")
	assert.NotNil(t, readPolicy, "Read policy should not be nil")
	assert.Equal(t, policy.Id, readPolicy.Id, "IDs should match")

	// Test Update
	updateReq := externalEonSdkAPI.NewUpdateBackupPolicyRequest(
		"updated-policy",
		*externalEonSdkAPI.NewBackupPolicyResourceSelector(
			externalEonSdkAPI.ResourceSelectorMode("ALL"),
		),
		*externalEonSdkAPI.NewBackupPolicyPlan(
			externalEonSdkAPI.BackupPolicyType("STANDARD"),
		),
	)
	updateReq.SetEnabled(false)

	updatedPolicy, err := mockClient.UpdateBackupPolicy(context.Background(), policy.Id, *updateReq)
	assert.NoError(t, err, "Update should not error")
	assert.NotNil(t, updatedPolicy, "Updated policy should not be nil")
	assert.Equal(t, "updated-policy", updatedPolicy.Name, "Name should be updated")
	assert.False(t, updatedPolicy.Enabled, "Should be disabled")

	// Test List
	policies, err := mockClient.ListBackupPolicies(context.Background())
	assert.NoError(t, err, "List should not error")
	assert.NotNil(t, policies, "Policies should not be nil")
	assert.Len(t, policies, 1, "Should have one policy")

	// Test Delete
	err = mockClient.DeleteBackupPolicy(context.Background(), policy.Id)
	assert.NoError(t, err, "Delete should not error")

	// Verify deletion
	_, err = mockClient.ReadBackupPolicy(context.Background(), policy.Id)
	assert.Error(t, err, "Should error when reading deleted policy")

	// Verify call counts
	assert.Equal(t, 1, mockClient.CreateCalls, "Should have made one create call")
	assert.Equal(t, 2, mockClient.ReadCalls, "Should have made two read calls")
	assert.Equal(t, 1, mockClient.UpdateCalls, "Should have made one update call")
	assert.Equal(t, 1, mockClient.DeleteCalls, "Should have made one delete call")
	assert.Equal(t, 1, mockClient.ListCalls, "Should have made one list call")
}

// TestCreateStandardIntervalConfig tests interval configuration for standard policies
func TestCreateStandardIntervalConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		intervalMinutes  int64
		startWindow      *int64
		expectError      bool
		validateInterval func(*testing.T, *externalEonSdkAPI.StandardBackupScheduleConfig)
	}{
		{
			name:            "60 minute interval",
			intervalMinutes: 60,
			expectError:     false,
			validateInterval: func(t *testing.T, config *externalEonSdkAPI.StandardBackupScheduleConfig) {
				assert.NotNil(t, config, "Config should not be nil")
				intervalConfig, ok := config.GetIntervalConfigOk()
				assert.True(t, ok, "Should have interval config")
				assert.NotNil(t, intervalConfig, "Interval config should not be nil")
				// Log the actual interval value for debugging
				t.Logf("Created StandardIntervalConfig with interval: %+v", intervalConfig)
			},
		},
		{
			name:            "120 minute interval",
			intervalMinutes: 120,
			expectError:     false,
			validateInterval: func(t *testing.T, config *externalEonSdkAPI.StandardBackupScheduleConfig) {
				assert.NotNil(t, config, "Config should not be nil")
				intervalConfig, ok := config.GetIntervalConfigOk()
				assert.True(t, ok, "Should have interval config")
				assert.NotNil(t, intervalConfig, "Interval config should not be nil")
				t.Logf("Created StandardIntervalConfig with interval: %+v", intervalConfig)
			},
		},
		{
			name:            "360 minute interval (6 hours)",
			intervalMinutes: 360,
			expectError:     false,
			validateInterval: func(t *testing.T, config *externalEonSdkAPI.StandardBackupScheduleConfig) {
				assert.NotNil(t, config, "Config should not be nil")
				intervalConfig, ok := config.GetIntervalConfigOk()
				assert.True(t, ok, "Should have interval config")
				assert.NotNil(t, intervalConfig, "Interval config should not be nil")
				t.Logf("Created StandardIntervalConfig with interval: %+v", intervalConfig)
			},
		},
		{
			name:            "15 minute interval (minimum)",
			intervalMinutes: 15,
			expectError:     false,
			validateInterval: func(t *testing.T, config *externalEonSdkAPI.StandardBackupScheduleConfig) {
				assert.NotNil(t, config, "Config should not be nil")
				intervalConfig, ok := config.GetIntervalConfigOk()
				assert.True(t, ok, "Should have interval config")
				assert.NotNil(t, intervalConfig, "Interval config should not be nil")
				t.Logf("Created StandardIntervalConfig with interval: %+v", intervalConfig)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create the interval config directly using SDK
			intervalMinutes32, err := SafeInt32Conversion(tt.intervalMinutes)
			assert.NoError(t, err, "Should convert interval minutes to int32")

			intervalConfig := externalEonSdkAPI.NewStandardIntervalConfig(intervalMinutes32)
			assert.NotNil(t, intervalConfig, "Should create interval config")

			// Log what we created
			t.Logf("Direct SDK call - NewStandardIntervalConfig(%d) = %+v", intervalMinutes32, intervalConfig)

			// Create schedule config
			scheduleConfig := externalEonSdkAPI.NewStandardBackupScheduleConfig(externalEonSdkAPI.STANDARD_BACKUP_SCHEDULE_INTERVAL)
			scheduleConfig.SetIntervalConfig(*intervalConfig)

			// Validate
			if tt.validateInterval != nil {
				tt.validateInterval(t, scheduleConfig)
			}
		})
	}
}

// TestCreateHighFrequencyIntervalConfig tests interval configuration for high frequency policies
func TestCreateHighFrequencyIntervalConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		intervalMinutes  int64
		expectError      bool
		validateInterval func(*testing.T, *externalEonSdkAPI.HighFrequencyBackupScheduleConfig)
	}{
		{
			name:            "30 minute high frequency interval",
			intervalMinutes: 30,
			expectError:     false,
			validateInterval: func(t *testing.T, config *externalEonSdkAPI.HighFrequencyBackupScheduleConfig) {
				assert.NotNil(t, config, "Config should not be nil")
				intervalConfig, ok := config.GetIntervalConfigOk()
				assert.True(t, ok, "Should have interval config")
				assert.NotNil(t, intervalConfig, "Interval config should not be nil")
				t.Logf("Created HighFrequencyIntervalConfig with interval: %+v", intervalConfig)
			},
		},
		{
			name:            "60 minute high frequency interval",
			intervalMinutes: 60,
			expectError:     false,
			validateInterval: func(t *testing.T, config *externalEonSdkAPI.HighFrequencyBackupScheduleConfig) {
				assert.NotNil(t, config, "Config should not be nil")
				intervalConfig, ok := config.GetIntervalConfigOk()
				assert.True(t, ok, "Should have interval config")
				assert.NotNil(t, intervalConfig, "Interval config should not be nil")
				t.Logf("Created HighFrequencyIntervalConfig with interval: %+v", intervalConfig)
			},
		},
		{
			name:            "15 minute high frequency interval",
			intervalMinutes: 15,
			expectError:     false,
			validateInterval: func(t *testing.T, config *externalEonSdkAPI.HighFrequencyBackupScheduleConfig) {
				assert.NotNil(t, config, "Config should not be nil")
				intervalConfig, ok := config.GetIntervalConfigOk()
				assert.True(t, ok, "Should have interval config")
				assert.NotNil(t, intervalConfig, "Interval config should not be nil")
				t.Logf("Created HighFrequencyIntervalConfig with interval: %+v", intervalConfig)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create the interval config directly using SDK
			intervalMinutes32, err := SafeInt32Conversion(tt.intervalMinutes)
			assert.NoError(t, err, "Should convert interval minutes to int32")

			intervalConfig := externalEonSdkAPI.NewHighFrequencyIntervalConfig(intervalMinutes32)
			assert.NotNil(t, intervalConfig, "Should create high frequency interval config")

			// Log what we created
			t.Logf("Direct SDK call - NewHighFrequencyIntervalConfig(%d) = %+v", intervalMinutes32, intervalConfig)

			// Create schedule config
			scheduleConfig := externalEonSdkAPI.NewHighFrequencyBackupScheduleConfig()
			scheduleConfig.SetFrequency(externalEonSdkAPI.HIGH_FREQUENCY_BACKUP_SCHEDULE_INTERVAL)
			scheduleConfig.SetIntervalConfig(*intervalConfig)

			// Validate
			if tt.validateInterval != nil {
				tt.validateInterval(t, scheduleConfig)
			}
		})
	}
}

// TestAwsNativePitrPlanConfig tests the AWS native PITR backup policy plan SDK construction
func TestAwsNativePitrPlanConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		retentionDays int64
		resourceType  string
		shouldError   bool
	}{
		{
			name:          "valid RDS PITR plan with 7 day retention",
			retentionDays: 7,
			resourceType:  "AWS_RDS",
			shouldError:   false,
		},
		{
			name:          "valid RDS PITR plan with 1 day retention (minimum)",
			retentionDays: 1,
			resourceType:  "AWS_RDS",
			shouldError:   false,
		},
		{
			name:          "valid RDS PITR plan with 35 day retention (maximum)",
			retentionDays: 35,
			resourceType:  "AWS_RDS",
			shouldError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			retentionDays, err := SafeInt32Conversion(tt.retentionDays)
			assert.NoError(t, err, "Should convert retention days to int32")

			resourceType := externalEonSdkAPI.NewAwsNativePitrBackupResourceType()
			resourceType.SetResourceType(externalEonSdkAPI.ResourceType(tt.resourceType))
			assert.Equal(t, externalEonSdkAPI.ResourceType(tt.resourceType), resourceType.GetResourceType())

			plan := externalEonSdkAPI.NewAwsNativePitrBackupPolicyPlan(retentionDays, *resourceType)
			assert.NotNil(t, plan, "Plan should not be nil")
			assert.Equal(t, retentionDays, plan.GetRetentionDays(), "Retention days should match")
			planResourceType := plan.GetResourceType()
			assert.Equal(t, externalEonSdkAPI.ResourceType(tt.resourceType), planResourceType.GetResourceType(), "Resource type should match")

			backupPlan := externalEonSdkAPI.NewBackupPolicyPlan(externalEonSdkAPI.BACKUP_POLICY_TYPE_AWS_NATIVE_PITR)
			backupPlan.SetAwsNativePitrPlan(*plan)
			assert.True(t, backupPlan.HasAwsNativePitrPlan(), "Should have AWS native PITR plan set")
			assert.Equal(t, externalEonSdkAPI.BACKUP_POLICY_TYPE_AWS_NATIVE_PITR, backupPlan.GetBackupPolicyType())
		})
	}
}

// TestAwsNativePitrCreateWithMockClient tests creation of AWS_NATIVE_PITR policy with mock client
func TestAwsNativePitrCreateWithMockClient(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	resourceType := externalEonSdkAPI.NewAwsNativePitrBackupResourceType()
	resourceType.SetResourceType(externalEonSdkAPI.AWS_RDS)

	pitrPlan := externalEonSdkAPI.NewAwsNativePitrBackupPolicyPlan(7, *resourceType)

	backupPlan := externalEonSdkAPI.NewBackupPolicyPlan(externalEonSdkAPI.BACKUP_POLICY_TYPE_AWS_NATIVE_PITR)
	backupPlan.SetAwsNativePitrPlan(*pitrPlan)

	createReq := externalEonSdkAPI.NewCreateBackupPolicyRequest(
		"rds-pitr-policy",
		*externalEonSdkAPI.NewBackupPolicyResourceSelector(
			externalEonSdkAPI.ResourceSelectorMode("ALL"),
		),
		*backupPlan,
	)
	createReq.SetEnabled(true)

	policy, err := mockClient.CreateBackupPolicy(context.Background(), *createReq)
	assert.NoError(t, err, "Create should not error")
	assert.NotNil(t, policy, "Policy should not be nil")
	assert.Equal(t, "rds-pitr-policy", policy.Name)
	assert.True(t, policy.Enabled)
	assert.NotEmpty(t, policy.Id)
	assert.Equal(t, 1, mockClient.CreateCalls)
}

// TestStandardIntervalConversion validates the minutes-to-hours conversion for standard policies
func TestStandardIntervalConversion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		intervalMinutes int64
		expectedHours   int32
		shouldError     bool
		errorContains   string
	}{
		{
			name:            "60 minutes = 1 hour",
			intervalMinutes: 60,
			expectedHours:   1,
			shouldError:     false,
		},
		{
			name:            "120 minutes = 2 hours",
			intervalMinutes: 120,
			expectedHours:   2,
			shouldError:     false,
		},
		{
			name:            "360 minutes = 6 hours",
			intervalMinutes: 360,
			expectedHours:   6,
			shouldError:     false,
		},
		{
			name:            "1440 minutes = 24 hours (daily)",
			intervalMinutes: 1440,
			expectedHours:   24,
			shouldError:     false,
		},
		{
			name:            "45 minutes - not divisible by 60",
			intervalMinutes: 45,
			shouldError:     true,
			errorContains:   "must be divisible by 60",
		},
		{
			name:            "15 minutes - not divisible by 60",
			intervalMinutes: 15,
			shouldError:     true,
			errorContains:   "must be divisible by 60",
		},
		{
			name:            "90 minutes - not divisible by 60",
			intervalMinutes: 90,
			shouldError:     true,
			errorContains:   "must be divisible by 60",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			intervalMinutes32, err := SafeInt32Conversion(tt.intervalMinutes)
			assert.NoError(t, err, "Should convert input to int32")

			// Simulate the conversion logic
			if intervalMinutes32%60 != 0 {
				if tt.shouldError {
					t.Logf("Correctly rejected non-hour-aligned interval: %d minutes", intervalMinutes32)
					assert.True(t, tt.shouldError, "Expected error for non-hour-aligned interval")
				} else {
					t.Errorf("Unexpectedly rejected interval: %d minutes", intervalMinutes32)
				}
				return
			}

			intervalHours := intervalMinutes32 / 60

			if tt.shouldError {
				t.Errorf("Expected error but conversion succeeded: %d minutes = %d hours", intervalMinutes32, intervalHours)
			} else {
				assert.Equal(t, tt.expectedHours, intervalHours, "Hours should match expected value")
				t.Logf("Successfully converted %d minutes to %d hours", intervalMinutes32, intervalHours)

				// Verify SDK accepts this value
				intervalConfig := externalEonSdkAPI.NewStandardIntervalConfig(intervalHours)
				assert.NotNil(t, intervalConfig, "SDK should accept the converted hours value")
				t.Logf("SDK IntervalConfig: %+v", intervalConfig)
			}
		})
	}
}
