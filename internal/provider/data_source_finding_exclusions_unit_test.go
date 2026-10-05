package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindingExclusionsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewFindingExclusionsDataSource())
}

func TestFindingExclusionsDataSource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		shouldFail    bool
		numExclusions int
		expectError   bool
	}{
		{"successful list with multiple exclusions", false, 2, false},
		{"successful list with no exclusions", false, 0, false},
		{"list failure", true, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailFindingExclusionList = tt.shouldFail
			for i := 0; i < tt.numExclusions; i++ {
				mockClient.AddMockFindingExclusion(externalEonSdkAPI.NewFindingExclusion(
					fmt.Sprintf("fe-%d", i+1),
					externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_ACCOUNT,
					fmt.Sprintf("table-%d", i+1),
					externalEonSdkAPI.FINDING_OBJECT_TYPE_TABLE,
					externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE,
					time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				))
			}

			result, err := mockClient.ListFindingExclusions(context.Background())
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result, tt.numExclusions)
			}
			assert.Equal(t, 1, mockClient.FindingExclusionListCalls)
		})
	}
}

func TestFindingExclusionListItem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		providerResourceId *string
		wantNull           bool
	}{
		{name: "resource scoped", providerResourceId: externalEonSdkAPI.PtrString("i-1")},
		{name: "account scoped", providerResourceId: nil, wantNull: true},
		{name: "empty provider resource id", providerResourceId: externalEonSdkAPI.PtrString(""), wantNull: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			exclusion := externalEonSdkAPI.NewFindingExclusion(
				"fe-1",
				externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_RESOURCE,
				"/var/cache/app",
				externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH,
				externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_RANSOMWARE_BEHAVIOR,
				time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC),
			)
			exclusion.ProviderResourceId = tt.providerResourceId

			item := findingExclusionListItem(exclusion)

			assert.Equal(t, "fe-1", item.Id.ValueString())
			assert.Equal(t, "RESOURCE", item.Scope.ValueString())
			assert.Equal(t, "/var/cache/app", item.Value.ValueString())
			assert.Equal(t, "PATH", item.Type.ValueString())
			assert.Equal(t, "RANSOMWARE_BEHAVIOR", item.Detector.ValueString())
			assert.Equal(t, "2024-05-06T07:08:09Z", item.UpdatedAt.ValueString())
			if tt.wantNull {
				assert.True(t, item.ProviderResourceId.IsNull())
			} else {
				assert.Equal(t, *tt.providerResourceId, item.ProviderResourceId.ValueString())
			}
		})
	}
}
