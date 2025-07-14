package provider

import (
	"context"
	"fmt"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/stretchr/testify/assert"
)

// TestBackupPoliciesDataSource_Unit tests the data source creation without API calls
func TestBackupPoliciesDataSource_Unit(t *testing.T) {
	t.Parallel()

	dataSource := NewBackupPoliciesDataSource()
	assert.NotNil(t, dataSource, "Data source should not be nil")
}

// TestBackupPoliciesDataSource_ListWithMockClient tests listing with mock client
func TestBackupPoliciesDataSource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		shouldFail    bool
		numPolicies   int
		expectedNames []string
	}{
		{
			name:          "successful list with multiple policies",
			shouldFail:    false,
			numPolicies:   3,
			expectedNames: []string{"policy-1", "policy-2", "policy-3"},
		},
		{
			name:          "successful list with single policy",
			shouldFail:    false,
			numPolicies:   1,
			expectedNames: []string{"single-policy"},
		},
		{
			name:          "successful list with no policies",
			shouldFail:    false,
			numPolicies:   0,
			expectedNames: []string{},
		},
		{
			name:          "list failure",
			shouldFail:    true,
			numPolicies:   0,
			expectedNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailList = tt.shouldFail

			// Add mock policies
			for i := 0; i < tt.numPolicies; i++ {
				name := "default-policy"
				if i < len(tt.expectedNames) {
					name = tt.expectedNames[i]
				}

				mockPolicy := &externalEonSdkAPI.BackupPolicy{
					Id:      fmt.Sprintf("policy-%d", i+1),
					Name:    name,
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

				// Verify policy names if any
				for i, policy := range result {
					if i < len(tt.expectedNames) {
						// Check if the policy name is in the expected names (order doesn't matter)
						found := false
						for _, expectedName := range tt.expectedNames {
							if policy.Name == expectedName {
								found = true
								break
							}
						}
						assert.True(t, found, "Policy name %s should be in expected names %v", policy.Name, tt.expectedNames)
					}
				}
			}

			// Verify call count
			assert.Equal(t, 1, mockClient.ListCalls, "Should have made one list call")
		})
	}
}

// TestBackupPoliciesDataSource_EmptyList tests empty list handling
func TestBackupPoliciesDataSource_EmptyList(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	// Test listing with no policies
	result, err := mockClient.ListBackupPolicies(context.Background())

	assert.NoError(t, err, "Expected no error for empty list")
	assert.NotNil(t, result, "Result should not be nil")
	assert.Len(t, result, 0, "Should return empty list")
	assert.Equal(t, 1, mockClient.ListCalls, "Should have made one list call")
}

// TestBackupPoliciesDataSource_LargeList tests handling of large lists
func TestBackupPoliciesDataSource_LargeList(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()
	numPolicies := 100

	// Add many mock policies
	for i := 0; i < numPolicies; i++ {
		mockPolicy := &externalEonSdkAPI.BackupPolicy{
			Id:      fmt.Sprintf("policy-%d", i),
			Name:    fmt.Sprintf("large-test-policy-%d", i),
			Enabled: i%2 == 0, // Alternate enabled/disabled
		}
		mockClient.AddMockPolicy(mockPolicy)
	}

	// Test listing
	result, err := mockClient.ListBackupPolicies(context.Background())

	assert.NoError(t, err, "Expected no error for large list")
	assert.NotNil(t, result, "Result should not be nil")
	assert.Len(t, result, numPolicies, "Should return all policies")

	// Verify some properties
	enabledCount := 0
	for _, policy := range result {
		assert.NotEmpty(t, policy.Id, "Policy ID should not be empty")
		assert.NotEmpty(t, policy.Name, "Policy name should not be empty")
		assert.Contains(t, policy.Name, "large-test-policy-", "Policy name should contain prefix")
		if policy.Enabled {
			enabledCount++
		}
	}

	// Should have roughly half enabled (50 out of 100)
	assert.Equal(t, 50, enabledCount, "Should have 50 enabled policies")
}

// TestBackupPoliciesDataSource_PolicyTypes tests different policy types
func TestBackupPoliciesDataSource_PolicyTypes(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	// Add policies with different properties
	policies := []struct {
		name    string
		enabled bool
	}{
		{"standard-enabled", true},
		{"standard-disabled", false},
		{"pitr-enabled", true},
		{"pitr-disabled", false},
	}

	for i, p := range policies {
		mockPolicy := &externalEonSdkAPI.BackupPolicy{
			Id:      fmt.Sprintf("policy-%d", i),
			Name:    p.name,
			Enabled: p.enabled,
		}
		mockClient.AddMockPolicy(mockPolicy)
	}

	// Test listing
	result, err := mockClient.ListBackupPolicies(context.Background())

	assert.NoError(t, err, "Expected no error")
	assert.NotNil(t, result, "Result should not be nil")
	assert.Len(t, result, len(policies), "Should return all policies")

	// Verify properties
	for i, policy := range result {
		expected := policies[i]
		assert.Equal(t, expected.name, policy.Name, "Policy name should match")
		assert.Equal(t, expected.enabled, policy.Enabled, "Policy enabled should match")
	}
}

// TestBackupPoliciesDataSource_ErrorHandling tests error scenarios
func TestBackupPoliciesDataSource_ErrorHandling(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupError func(*client.MockEonClient)
		expectErr  bool
	}{
		{
			name: "list failure",
			setupError: func(client *client.MockEonClient) {
				client.ShouldFailList = true
			},
			expectErr: true,
		},
		{
			name: "normal operation",
			setupError: func(client *client.MockEonClient) {
				// No error setup
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			tt.setupError(mockClient)

			// Test listing
			result, err := mockClient.ListBackupPolicies(context.Background())

			if tt.expectErr {
				assert.Error(t, err, "Expected error")
				assert.Nil(t, result, "Result should be nil on error")
			} else {
				assert.NoError(t, err, "Expected no error")
				assert.NotNil(t, result, "Result should not be nil")
			}
		})
	}
}

// TestBackupPoliciesDataSource_ConcurrentAccess tests concurrent access
func TestBackupPoliciesDataSource_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	// Add some mock policies
	for i := 0; i < 10; i++ {
		mockPolicy := &externalEonSdkAPI.BackupPolicy{
			Id:      fmt.Sprintf("concurrent-policy-%d", i),
			Name:    fmt.Sprintf("concurrent-test-policy-%d", i),
			Enabled: true,
		}
		mockClient.AddMockPolicy(mockPolicy)
	}

	// Test concurrent access
	const numGoroutines = 5
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() {
				if r := recover(); r != nil {
					results <- fmt.Errorf("panic in goroutine %d: %v", goroutineID, r)
					return
				}
			}()

			// Test listing
			result, err := mockClient.ListBackupPolicies(context.Background())
			if err != nil {
				results <- fmt.Errorf("error in goroutine %d: %v", goroutineID, err)
				return
			}

			if len(result) != 10 {
				results <- fmt.Errorf("unexpected result count in goroutine %d: got %d, want 10", goroutineID, len(result))
				return
			}

			results <- nil
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-results:
			assert.NoError(t, err, "Goroutine should complete without error")
		}
	}

	// Verify total call count
	assert.Equal(t, numGoroutines, mockClient.ListCalls, "Should have made calls from all goroutines")
}
