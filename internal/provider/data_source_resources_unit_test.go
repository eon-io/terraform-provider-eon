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
		filters, diags := buildListResourcesFilters(context.Background(), ResourcesDataSourceModel{})
		assert.False(t, diags.HasError())
		assert.Nil(t, filters)
	})

	t.Run("with provider resource ids", func(t *testing.T) {
		t.Parallel()
		ids, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"i-abc", "i-def"})
		require.False(t, diags.HasError())

		filters, diags := buildListResourcesFilters(context.Background(), ResourcesDataSourceModel{
			ProviderResourceIds: ids,
		})
		require.False(t, diags.HasError())
		require.NotNil(t, filters)
		require.True(t, filters.HasProviderResourceId())
		providerFilters, ok := filters.GetProviderResourceIdOk()
		require.True(t, ok)
		require.NotNil(t, providerFilters)
		assert.Equal(t, []string{"i-abc", "i-def"}, providerFilters.GetIn())
	})
}

func TestResourceListModelFromInventoryResource(t *testing.T) {
	t.Parallel()

	res := externalEonSdkAPI.NewInventoryResource(
		"res-1",
		externalEonSdkAPI.PROTECTED,
		"i-1234567890abcdef0",
		"demo-resource",
		"123456789012",
		externalEonSdkAPI.SnapshotStorage{},
		externalEonSdkAPI.SourceStorage{},
		map[string]string{"env": "prod"},
		externalEonSdkAPI.AWS,
		externalEonSdkAPI.AWS_EC2,
		"us-east-1",
	)
	vpc := "vpc-123"
	res.SetVpc(vpc)
	res.SetSubnets([]string{"subnet-1"})

	details := externalEonSdkAPI.NewEnvironmentDetails()
	details.SetEnvironment(externalEonSdkAPI.PROD)
	res.Classifications = externalEonSdkAPI.NewClassifications()
	res.Classifications.SetEnvironmentDetails(*details)

	model, diags := resourceListModelFromInventoryResource(context.Background(), *res)
	require.False(t, diags.HasError())
	assert.Equal(t, "res-1", model.Id.ValueString())
	assert.Equal(t, "i-1234567890abcdef0", model.ProviderResourceId.ValueString())
	assert.Equal(t, "demo-resource", model.ResourceName.ValueString())
	assert.Equal(t, "PROTECTED", model.BackupStatus.ValueString())
	assert.Equal(t, "AWS", model.CloudProvider.ValueString())
	assert.Equal(t, "AWS_EC2", model.ResourceType.ValueString())
	assert.Equal(t, "us-east-1", model.Region.ValueString())
	assert.Equal(t, "vpc-123", model.Vpc.ValueString())
	assert.Equal(t, "PROD", model.Environment.ValueString())
}
