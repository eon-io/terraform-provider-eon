package provider

import (
	"encoding/json"
	"testing"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindingExclusionRequests(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		resourceID     types.String
		objectType     string
		detector       string
		wantErr        bool
		wantResourceID bool
	}{
		{
			name:           "with resource",
			resourceID:     types.StringValue("res-1"),
			objectType:     "PATH",
			detector:       "MALWARE",
			wantResourceID: true,
		},
		{
			name:       "account wide",
			resourceID: types.StringNull(),
			objectType: "TABLE",
			detector:   "DATA_ANOMALY",
		},
		{
			name:       "invalid detector",
			resourceID: types.StringNull(),
			objectType: "PATH",
			detector:   "NOT_A_DETECTOR",
			wantErr:    true,
		},
		{
			name:       "invalid type",
			resourceID: types.StringNull(),
			objectType: "NOT_A_TYPE",
			detector:   "MALWARE",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := &FindingExclusionResourceModel{
				ResourceId: tt.resourceID,
				Value:      types.StringValue("/var/cache"),
				Type:       types.StringValue(tt.objectType),
				Detector:   types.StringValue(tt.detector),
			}

			createReq, diags := findingExclusionCreateRequest(data)
			if tt.wantErr {
				require.True(t, diags.HasError())
				return
			}
			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
			assert.Equal(t, "/var/cache", createReq.GetValue())
			assert.Equal(t, tt.objectType, string(createReq.GetType()))
			assert.Equal(t, tt.detector, string(createReq.GetDetector()))
			assert.Equal(t, tt.wantResourceID, createReq.HasResourceId())
			if tt.wantResourceID {
				assert.Equal(t, "res-1", createReq.GetResourceId())
			}

			updateReq, diags := findingExclusionUpdateRequest(data)
			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
			encoded, err := json.Marshal(updateReq)
			require.NoError(t, err)
			if tt.wantResourceID {
				assert.Contains(t, string(encoded), `"resourceId":"res-1"`)
			} else {
				assert.NotContains(t, string(encoded), "resourceId")
			}
		})
	}
}

func TestFindingExclusionStateFromAPI(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name           string
		resourceID     string
		wantResourceID string
	}{
		{name: "with resource", resourceID: "res-1", wantResourceID: "res-1"},
		{name: "account wide"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			exclusion := externalEonSdkAPI.NewFindingExclusion(
				"fe-1",
				"/var/cache",
				externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH,
				externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE,
				updatedAt,
			)
			if tt.resourceID != "" {
				exclusion.SetResourceId(tt.resourceID)
			}

			var state FindingExclusionResourceModel
			findingExclusionToState(exclusion, &state)
			assert.Equal(t, "fe-1", state.Id.ValueString())
			assert.Equal(t, "/var/cache", state.Value.ValueString())
			assert.Equal(t, "PATH", state.Type.ValueString())
			assert.Equal(t, "MALWARE", state.Detector.ValueString())
			assert.Equal(t, "2026-01-02T03:04:05Z", state.UpdatedAt.ValueString())
			if tt.wantResourceID == "" {
				assert.True(t, state.ResourceId.IsNull())
			} else {
				assert.Equal(t, tt.wantResourceID, state.ResourceId.ValueString())
			}

			item := findingExclusionListItem(*exclusion)
			assert.Equal(t, state.Id, item.Id)
			assert.Equal(t, state.ResourceId, item.ResourceId)
			assert.Equal(t, state.Value, item.Value)
			assert.Equal(t, state.Type, item.Type)
			assert.Equal(t, state.Detector, item.Detector)
			assert.Equal(t, state.UpdatedAt, item.UpdatedAt)
		})
	}
}
