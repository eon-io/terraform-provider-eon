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
		name                   string
		apiResponse            string
		wantScope              externalEonSdkAPI.FindingExclusionScope
		wantProviderResourceId types.String
	}{
		{
			name: "resource exclusion",
			apiResponse: `{"id": "exclusion-1", "scope": "RESOURCE", "providerResourceId": "i-0abc123def4567890",
				"value": "/var/cache", "type": "PATH", "detector": "MALWARE", "updatedAt": "2026-09-01T00:00:00Z"}`,
			wantScope:              externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_RESOURCE,
			wantProviderResourceId: types.StringValue("i-0abc123def4567890"),
		},
		{
			name: "account-wide exclusion has no resource",
			apiResponse: `{"id": "exclusion-1", "scope": "ACCOUNT", "value": "/var/cache", "type": "PATH", "detector": "MALWARE",
				"updatedAt": "2026-09-01T00:00:00Z"}`,
			wantScope:              externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_ACCOUNT,
			wantProviderResourceId: types.StringNull(),
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
			assert.Equal(t, string(tt.wantScope), state.Scope.ValueString())
			assert.Equal(t, tt.wantProviderResourceId, state.ProviderResourceId)
			assert.Equal(t, "/var/cache", state.Value.ValueString())
			assert.Equal(t, "PATH", state.Type.ValueString())
			assert.Equal(t, "MALWARE", state.Detector.ValueString())
			assert.Equal(t, "2026-09-01T00:00:00Z", state.UpdatedAt.ValueString())

			createReq, diags := findingExclusionCreateRequest(&state)
			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
			assert.Equal(t, tt.wantScope, createReq.GetScope())
			assert.Equal(t, tt.wantProviderResourceId.ValueString(), createReq.GetProviderResourceId())
			assert.Equal(t, externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH, createReq.GetType())
			assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE, createReq.GetDetector())

			updateReq, diags := findingExclusionUpdateRequest(&state)
			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
			assert.Equal(t, tt.wantScope, updateReq.GetScope())
			assert.Equal(t, tt.wantProviderResourceId.ValueString(), updateReq.GetProviderResourceId())
			assert.Equal(t, "/var/cache", updateReq.GetValue())
		})
	}
}

func TestFindingExclusionStateKeepsProviderIdOfResourceGoneFromInventory(t *testing.T) {
	t.Parallel()
	var exclusion externalEonSdkAPI.FindingExclusion
	require.NoError(t, json.Unmarshal([]byte(`{"id": "exclusion-1", "scope": "RESOURCE", "value": "/var/cache",
		"type": "PATH", "detector": "MALWARE", "updatedAt": "2026-09-01T00:00:00Z"}`), &exclusion))

	state := FindingExclusionResourceModel{ProviderResourceId: types.StringValue("i-0abc123def4567890")}
	findingExclusionToState(&exclusion, &state)

	assert.Equal(t, "RESOURCE", state.Scope.ValueString())
	assert.Equal(t, types.StringValue("i-0abc123def4567890"), state.ProviderResourceId)
}

func TestFindingExclusionRequestRejectsScopeMismatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		scope              string
		providerResourceId types.String
	}{
		{name: "resource scope without a resource", scope: "RESOURCE", providerResourceId: types.StringNull()},
		{name: "account scope with a resource", scope: "ACCOUNT", providerResourceId: types.StringValue("i-0abc123def4567890")},
		{name: "unknown scope", scope: "GLOBAL", providerResourceId: types.StringNull()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data := &FindingExclusionResourceModel{
				Scope:              types.StringValue(tt.scope),
				ProviderResourceId: tt.providerResourceId,
				Value:              types.StringValue("/var/cache"),
				Type:               types.StringValue("PATH"),
				Detector:           types.StringValue("MALWARE"),
			}

			_, diags := findingExclusionCreateRequest(data)
			assert.True(t, diags.HasError())
			_, diags = findingExclusionUpdateRequest(data)
			assert.True(t, diags.HasError())
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
				Scope:    types.StringValue("ACCOUNT"),
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
