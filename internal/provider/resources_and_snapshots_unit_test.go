package provider

import (
	"context"
	"fmt"
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
		{"successful override and remove", false, false},
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

			result, err := mockClient.OverrideEnvironment(context.Background(), "res-1", "PROD")
			if tt.failOverride {
				assert.Error(t, err)
				assert.Empty(t, result)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "PROD", result)

			res, getErr := mockClient.GetResourceById(context.Background(), "res-1")
			require.NoError(t, getErr)
			environment, ok := environmentOverrideFromInventoryResource(res)
			require.True(t, ok)
			assert.Equal(t, "PROD", environment)

			err = mockClient.RemoveEnvironmentOverride(context.Background(), "res-1")
			if tt.failRemove {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestEnvironmentOverrideFromInventoryResource(t *testing.T) {
	t.Parallel()

	t.Run("nil resource", func(t *testing.T) {
		t.Parallel()
		_, ok := environmentOverrideFromInventoryResource(nil)
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
		details := externalEonSdkAPI.NewEnvironmentDetails()
		details.SetEnvironment(externalEonSdkAPI.PROD)
		details.SetIsOverridden(false)
		res.Classifications = externalEonSdkAPI.NewClassifications()
		res.Classifications.SetEnvironmentDetails(*details)

		_, ok := environmentOverrideFromInventoryResource(res)
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
		details := externalEonSdkAPI.NewEnvironmentDetails()
		details.SetEnvironment(externalEonSdkAPI.STAGE)
		details.SetIsOverridden(true)
		res.Classifications = externalEonSdkAPI.NewClassifications()
		res.Classifications.SetEnvironmentDetails(*details)

		environment, ok := environmentOverrideFromInventoryResource(res)
		require.True(t, ok)
		assert.Equal(t, "STAGE", environment)
	})
}

func TestResourcesDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewResourcesDataSource())
}

func TestResourcesDataSource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		shouldFail   bool
		numResources int
		expectError  bool
	}{
		{"successful list with multiple resources", false, 2, false},
		{"successful list with no resources", false, 0, false},
		{"list failure", true, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailListResources = tt.shouldFail
			for i := 0; i < tt.numResources; i++ {
				mockClient.AddMockInventoryResource(externalEonSdkAPI.NewInventoryResource(
					fmt.Sprintf("res-%d", i+1),
					externalEonSdkAPI.PROTECTED,
					fmt.Sprintf("i-%d", i+1),
					fmt.Sprintf("demo-%d", i+1),
					"123456789012",
					externalEonSdkAPI.SnapshotStorage{},
					externalEonSdkAPI.SourceStorage{},
					map[string]string{},
					externalEonSdkAPI.AWS,
					externalEonSdkAPI.AWS_EC2,
					"us-east-1",
				))
			}

			result, err := mockClient.ListResources(context.Background(), nil)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result, tt.numResources)
			}
			assert.Equal(t, 1, mockClient.ListResourcesCalls)
		})
	}
}

func TestBuildListResourcesFilters(t *testing.T) {
	t.Parallel()

	t.Run("no filters", func(t *testing.T) {
		t.Parallel()
		filters, err := buildListResourcesFilters(context.Background(), ResourcesDataSourceModel{})
		require.NoError(t, err)
		assert.Nil(t, filters)
	})

	t.Run("with filters", func(t *testing.T) {
		t.Parallel()
		ids, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"res-1"})
		require.False(t, diags.HasError())
		providers, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"AWS"})
		require.False(t, diags.HasError())
		resourceTypes, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"AWS_EC2"})
		require.False(t, diags.HasError())
		environments, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"PROD"})
		require.False(t, diags.HasError())

		filters, err := buildListResourcesFilters(context.Background(), ResourcesDataSourceModel{
			Ids:            ids,
			CloudProviders: providers,
			ResourceTypes:  resourceTypes,
			Environments:   environments,
		})
		require.NoError(t, err)
		require.NotNil(t, filters)
		require.NotNil(t, filters.Id)
		require.NotNil(t, filters.CloudProvider)
		require.NotNil(t, filters.ResourceType)
		require.NotNil(t, filters.Environment)
		assert.Equal(t, []string{"res-1"}, filters.Id.GetIn())
		assert.Equal(t, []externalEonSdkAPI.Provider{externalEonSdkAPI.AWS}, filters.CloudProvider.GetIn())
		assert.Equal(t, []externalEonSdkAPI.ResourceType{externalEonSdkAPI.AWS_EC2}, filters.ResourceType.GetIn())
		assert.Equal(t, []externalEonSdkAPI.Environment{externalEonSdkAPI.PROD}, filters.Environment.GetIn())
	})
}

func TestOptionalStringList(t *testing.T) {
	t.Parallel()

	empty, err := optionalStringList(context.Background(), types.ListNull(types.StringType))
	require.NoError(t, err)
	assert.Nil(t, empty)

	list, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"a", "b"})
	require.False(t, diags.HasError())
	values, err := optionalStringList(context.Background(), list)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, values)
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
		numSnaps    int
		expectError bool
	}{
		{"successful list with snapshots", false, 2, false},
		{"successful list with no snapshots", false, 0, false},
		{"list failure", true, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailListResourceSnapshots = tt.shouldFail
			for i := 0; i < tt.numSnaps; i++ {
				snap := externalEonSdkAPI.NewSnapshot(
					fmt.Sprintf("snap-%d", i+1),
					time.Date(2024, 1, i+1, 0, 0, 0, 0, time.UTC),
					"res-1",
				)
				mockClient.AddMockResourceSnapshot("res-1", snap)
			}

			result, err := mockClient.ListResourceSnapshots(context.Background(), "res-1", nil)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result, tt.numSnaps)
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

	t.Run("with date window", func(t *testing.T) {
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
