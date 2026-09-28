package provider

import (
	"encoding/json"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindingExclusionStateRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		apiResponse    string
		wantResourceId types.String
	}{
		{
			name: "resource exclusion",
			apiResponse: `{"id": "exclusion-1", "resourceId": "1ee34dc5-0a7c-4e56-a820-917371e05c8d", "value": "/var/cache",
				"type": "PATH", "detector": "MALWARE", "updatedAt": "2026-09-01T00:00:00Z"}`,
			wantResourceId: types.StringValue("1ee34dc5-0a7c-4e56-a820-917371e05c8d"),
		},
		{
			name: "account-wide exclusion has no resource",
			apiResponse: `{"id": "exclusion-1", "value": "/var/cache", "type": "PATH", "detector": "MALWARE",
				"updatedAt": "2026-09-01T00:00:00Z"}`,
			wantResourceId: types.StringNull(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var exclusion externalEonSdkAPI.FindingExclusion
			require.NoError(t, json.Unmarshal([]byte(tt.apiResponse), &exclusion))

			var state FindingExclusionResourceModel
			findingExclusionToState(&exclusion, &state)

			assert.Equal(t, "exclusion-1", state.Id.ValueString())
			assert.Equal(t, tt.wantResourceId, state.ResourceId)
			assert.Equal(t, "/var/cache", state.Value.ValueString())
			assert.Equal(t, "PATH", state.Type.ValueString())
			assert.Equal(t, "MALWARE", state.Detector.ValueString())
			assert.Equal(t, "2026-09-01T00:00:00Z", state.UpdatedAt.ValueString())

			createReq, diags := findingExclusionCreateRequest(&state)
			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
			assert.Equal(t, externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH, createReq.GetType())
			assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE, createReq.GetDetector())
			assert.Equal(t, !tt.wantResourceId.IsNull(), createReq.HasResourceId())

			updateReq, diags := findingExclusionUpdateRequest(&state)
			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
			assert.Equal(t, "/var/cache", updateReq.GetValue())
			assert.Equal(t, !tt.wantResourceId.IsNull(), updateReq.HasResourceId())
		})
	}
}

func TestFindingExclusionRequestRejectsUnknownValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typ      string
		detector string
	}{
		{name: "unknown type", typ: "BUCKET", detector: "MALWARE"},
		{name: "detector with no exclusions", typ: "PATH", detector: "AI_DATA_RISK"},
		{name: "lowercase value", typ: "path", detector: "MALWARE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data := &FindingExclusionResourceModel{
				Value:    types.StringValue("/var/cache"),
				Type:     types.StringValue(tt.typ),
				Detector: types.StringValue(tt.detector),
			}

			_, diags := findingExclusionCreateRequest(data)
			assert.True(t, diags.HasError())
			_, diags = findingExclusionUpdateRequest(data)
			assert.True(t, diags.HasError())
		})
	}
}
