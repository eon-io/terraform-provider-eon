package provider

import (
	"context"
	"testing"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceEnvironmentOverrideResource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewResourceEnvironmentOverrideResource())
}

func TestResourceEnvironmentOverride_OverrideAndRemoveWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		failOverride bool
		failRemove   bool
	}{
		{"success", false, false},
		{"override failure", true, false},
		{"remove failure", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailOverrideEnvironment = tt.failOverride
			mockClient.ShouldFailRemoveEnvironmentOverride = tt.failRemove
			mockClient.AddMockInventoryResource(externalEonSdkAPI.NewInventoryResource(
				"res-1",
				externalEonSdkAPI.PROTECTED,
				"i-1",
				"demo-1",
				"123456789012",
				externalEonSdkAPI.SnapshotStorage{},
				externalEonSdkAPI.SourceStorage{},
				map[string]string{},
				externalEonSdkAPI.AWS,
				externalEonSdkAPI.AWS_EC2,
				"us-east-1",
			))

			env, err := mockClient.OverrideEnvironment(context.Background(), "res-1", "PROD")
			if tt.failOverride {
				assert.Error(t, err)
				assert.Empty(t, env)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "PROD", env)
			}
			assert.Equal(t, 1, mockClient.OverrideEnvironmentCalls)

			err = mockClient.RemoveEnvironmentOverride(context.Background(), "res-1")
			if tt.failRemove {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, 1, mockClient.RemoveEnvironmentOverrideCalls)
		})
	}
}

func TestEnvironmentOverrideFromInventoryResource(t *testing.T) {
	t.Parallel()

	t.Run("overridden prod", func(t *testing.T) {
		t.Parallel()
		res := externalEonSdkAPI.NewInventoryResource(
			"res-1",
			externalEonSdkAPI.PROTECTED,
			"i-1",
			"demo-1",
			"123456789012",
			externalEonSdkAPI.SnapshotStorage{},
			externalEonSdkAPI.SourceStorage{},
			map[string]string{},
			externalEonSdkAPI.AWS,
			externalEonSdkAPI.AWS_EC2,
			"us-east-1",
		)
		details := externalEonSdkAPI.NewEnvironmentDetails()
		details.SetEnvironment(externalEonSdkAPI.PROD)
		details.SetIsOverridden(true)
		res.Classifications = externalEonSdkAPI.NewClassifications()
		res.Classifications.SetEnvironmentDetails(*details)

		env, ok := environmentOverrideFromInventoryResource(res)
		assert.True(t, ok)
		assert.Equal(t, "PROD", env)
	})

	t.Run("not overridden", func(t *testing.T) {
		t.Parallel()
		res := externalEonSdkAPI.NewInventoryResource(
			"res-1",
			externalEonSdkAPI.PROTECTED,
			"i-1",
			"demo-1",
			"123456789012",
			externalEonSdkAPI.SnapshotStorage{},
			externalEonSdkAPI.SourceStorage{},
			map[string]string{},
			externalEonSdkAPI.AWS,
			externalEonSdkAPI.AWS_EC2,
			"us-east-1",
		)
		details := externalEonSdkAPI.NewEnvironmentDetails()
		details.SetEnvironment(externalEonSdkAPI.STAGE)
		details.SetIsOverridden(false)
		res.Classifications = externalEonSdkAPI.NewClassifications()
		res.Classifications.SetEnvironmentDetails(*details)

		_, ok := environmentOverrideFromInventoryResource(res)
		assert.False(t, ok)
	})
}

func TestResourceSnapshotsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewResourceSnapshotsDataSource())
}

func TestResourceSnapshotsDataSource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shouldFail  bool
		expectError bool
	}{
		{"successful list", false, false},
		{"list failure", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailListResourceSnapshots = tt.shouldFail
			created := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
			snap := externalEonSdkAPI.NewSnapshot("snap-1", created, "res-1")
			mockClient.AddMockResourceSnapshot("res-1", snap)

			result, err := mockClient.ListResourceSnapshots(context.Background(), "res-1", nil)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result, 1)
				assert.Equal(t, "snap-1", result[0].GetId())
			}
			assert.Equal(t, 1, mockClient.ListResourceSnapshotsCalls)
		})
	}
}

func TestBuildListResourceSnapshotsFilters(t *testing.T) {
	t.Parallel()

	t.Run("no filters", func(t *testing.T) {
		t.Parallel()
		filters := buildListResourceSnapshotsFilters(ResourceSnapshotsDataSourceModel{})
		assert.Nil(t, filters)
	})

	t.Run("with point in time", func(t *testing.T) {
		t.Parallel()
		filters := buildListResourceSnapshotsFilters(ResourceSnapshotsDataSourceModel{
			PointInTimeStartDate: types.StringValue("2024-01-01"),
			PointInTimeEndDate:   types.StringValue("2024-12-31"),
		})
		require.NotNil(t, filters)
		require.NotNil(t, filters.PointInTime)
		assert.Equal(t, "2024-01-01", filters.PointInTime.GetStartDate())
		assert.Equal(t, "2024-12-31", filters.PointInTime.GetEndDate())
	})
}
