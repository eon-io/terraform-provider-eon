package provider

import (
	"context"
	"fmt"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewPermissionsDataSource())
}

func TestPermissionsDataSource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shouldFail  bool
		numPerms    int
		expectError bool
	}{
		{"successful list with multiple permissions", false, 2, false},
		{"successful list with no permissions", false, 0, false},
		{"list failure", true, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailPermissionsList = tt.shouldFail
			for i := 0; i < tt.numPerms; i++ {
				mockClient.AddMockPermission(externalEonSdkAPI.NewPermission(
					externalEonSdkAPI.INVENTORY_VIEW,
					fmt.Sprintf("Permission %d", i+1),
					i%2 == 0,
				))
			}

			result, err := mockClient.ListPermissions(context.Background())
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result, tt.numPerms)
			}
			assert.Equal(t, 1, mockClient.PermissionsListCalls)
		})
	}
}

func TestResourceBackupExclusionResource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewResourceBackupExclusionResource())
}

func TestResourceBackupExclusion_ExcludeAndCancelWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		failExclude   bool
		failCancel    bool
		expectExclude bool
		expectCancel  bool
	}{
		{"successful exclude and cancel", false, false, false, false},
		{"exclude failure", true, false, true, false},
		{"cancel failure", false, true, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailExcludeResource = tt.failExclude
			mockClient.ShouldFailCancelResourceExclusion = tt.failCancel
			mockClient.AddMockInventoryResource(externalEonSdkAPI.NewInventoryResource(
				"res-1",
				externalEonSdkAPI.PROTECTED,
				"i-123",
				"demo",
				"123456789012",
				externalEonSdkAPI.SnapshotStorage{},
				externalEonSdkAPI.SourceStorage{},
				map[string]string{},
				externalEonSdkAPI.AWS,
				externalEonSdkAPI.AWS_EC2,
				"us-east-1",
			))

			err := mockClient.ExcludeResourceFromBackup(context.Background(), "res-1")
			if tt.expectExclude {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				res, getErr := mockClient.GetResourceById(context.Background(), "res-1")
				require.NoError(t, getErr)
				assert.Equal(t, externalEonSdkAPI.EXCLUDED_FROM_BACKUP, res.GetBackupStatus())
			}

			err = mockClient.CancelResourceBackupExclusion(context.Background(), "res-1")
			if tt.expectCancel {
				assert.Error(t, err)
			} else if !tt.expectExclude {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResourceDataClassesOverrideResource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewResourceDataClassesOverrideResource())
}

func TestResourceDataClassesOverride_OverrideAndRemoveWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		failOverride bool
		failRemove   bool
		expectError  bool
	}{
		{"successful override and remove", false, false, false},
		{"override failure", true, false, true},
		{"remove failure", false, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailOverrideDataClasses = tt.failOverride
			mockClient.ShouldFailRemoveDataClassesOverride = tt.failRemove
			mockClient.AddMockInventoryResource(externalEonSdkAPI.NewInventoryResource(
				"res-1",
				externalEonSdkAPI.PROTECTED,
				"i-123",
				"demo",
				"123456789012",
				externalEonSdkAPI.SnapshotStorage{},
				externalEonSdkAPI.SourceStorage{},
				map[string]string{},
				externalEonSdkAPI.AWS,
				externalEonSdkAPI.AWS_EC2,
				"us-east-1",
			))

			result, err := mockClient.OverrideDataClasses(context.Background(), "res-1", []string{"PII", "PCI"})
			if tt.failOverride {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []string{"PII", "PCI"}, result)

			res, getErr := mockClient.GetResourceById(context.Background(), "res-1")
			require.NoError(t, getErr)
			dataClasses, ok := dataClassesOverrideFromInventoryResource(res)
			require.True(t, ok)
			assert.Equal(t, []string{"PII", "PCI"}, dataClasses)

			err = mockClient.RemoveDataClassesOverride(context.Background(), "res-1")
			if tt.failRemove {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestDataClassesOverrideFromInventoryResource(t *testing.T) {
	t.Parallel()

	t.Run("nil resource", func(t *testing.T) {
		t.Parallel()
		_, ok := dataClassesOverrideFromInventoryResource(nil)
		assert.False(t, ok)
	})

	t.Run("not overridden", func(t *testing.T) {
		t.Parallel()
		res := externalEonSdkAPI.NewInventoryResource(
			"res-1",
			externalEonSdkAPI.PROTECTED,
			"i-123",
			"demo",
			"123456789012",
			externalEonSdkAPI.SnapshotStorage{},
			externalEonSdkAPI.SourceStorage{},
			map[string]string{},
			externalEonSdkAPI.AWS,
			externalEonSdkAPI.AWS_EC2,
			"us-east-1",
		)
		details := externalEonSdkAPI.NewDataClassesDetails()
		details.SetDataClasses([]string{"PII"})
		details.SetIsOverridden(false)
		res.Classifications = externalEonSdkAPI.NewClassifications()
		res.Classifications.SetDataClassesDetails(*details)

		_, ok := dataClassesOverrideFromInventoryResource(res)
		assert.False(t, ok)
	})

	t.Run("overridden", func(t *testing.T) {
		t.Parallel()
		res := externalEonSdkAPI.NewInventoryResource(
			"res-1",
			externalEonSdkAPI.PROTECTED,
			"i-123",
			"demo",
			"123456789012",
			externalEonSdkAPI.SnapshotStorage{},
			externalEonSdkAPI.SourceStorage{},
			map[string]string{},
			externalEonSdkAPI.AWS,
			externalEonSdkAPI.AWS_EC2,
			"us-east-1",
		)
		details := externalEonSdkAPI.NewDataClassesDetails()
		details.SetDataClasses([]string{"PII", "PHI"})
		details.SetIsOverridden(true)
		res.Classifications = externalEonSdkAPI.NewClassifications()
		res.Classifications.SetDataClassesDetails(*details)

		dataClasses, ok := dataClassesOverrideFromInventoryResource(res)
		require.True(t, ok)
		assert.Equal(t, []string{"PII", "PHI"}, dataClasses)
	})
}

func TestSetToStringSliceAndBack(t *testing.T) {
	t.Parallel()

	set := stringSliceToSet([]string{"PCI", "PII"})
	values, err := setToStringSlice(context.Background(), set)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"PCI", "PII"}, values)

	empty, err := setToStringSlice(context.Background(), types.SetNull(types.StringType))
	require.NoError(t, err)
	assert.Empty(t, empty)
}
