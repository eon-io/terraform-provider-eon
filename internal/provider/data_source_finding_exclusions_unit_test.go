package provider

import (
	"testing"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/stretchr/testify/assert"
)

func TestFindingExclusionsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewFindingExclusionsDataSource())
}

func TestFindingExclusionListItem(t *testing.T) {
	t.Parallel()

	updated := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name               string
		exclusion          *externalEonSdkAPI.FindingExclusion
		wantProviderIDNull bool
		wantProviderID     string
	}{
		{
			name: "resource scope",
			exclusion: func() *externalEonSdkAPI.FindingExclusion {
				exclusion := externalEonSdkAPI.NewFindingExclusion(
					"excl-1",
					externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_RESOURCE,
					"/var/cache",
					externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH,
					externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE,
					updated,
				)
				exclusion.SetProviderResourceId("i-abc")
				return exclusion
			}(),
			wantProviderID: "i-abc",
		},
		{
			name: "account scope",
			exclusion: externalEonSdkAPI.NewFindingExclusion(
				"excl-2",
				externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_ACCOUNT,
				"orders",
				externalEonSdkAPI.FINDING_OBJECT_TYPE_TABLE,
				externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_DATA_ANOMALY,
				updated,
			),
			wantProviderIDNull: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			item := findingExclusionListItem(*tt.exclusion)
			assert.Equal(t, tt.exclusion.GetId(), item.Id.ValueString())
			assert.Equal(t, string(tt.exclusion.GetScope()), item.Scope.ValueString())
			assert.Equal(t, tt.exclusion.GetValue(), item.Value.ValueString())
			assert.Equal(t, string(tt.exclusion.GetType()), item.Type.ValueString())
			assert.Equal(t, string(tt.exclusion.GetDetector()), item.Detector.ValueString())
			assert.Equal(t, "2026-01-02T03:04:05Z", item.UpdatedAt.ValueString())
			if tt.wantProviderIDNull {
				assert.True(t, item.ProviderResourceId.IsNull())
				return
			}
			assert.Equal(t, tt.wantProviderID, item.ProviderResourceId.ValueString())
		})
	}
}
