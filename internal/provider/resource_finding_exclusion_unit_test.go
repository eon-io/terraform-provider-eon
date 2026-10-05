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

const findingExclusionAPIResponse = `{
  "id": "fe-1",
  "scope": "RESOURCE",
  "providerResourceId": "i-0123456789abcdef0",
  "value": "/var/cache/app",
  "type": "PATH",
  "detector": "MALWARE",
  "updatedAt": "2024-01-02T03:04:05Z"
}`

func TestFindingExclusionResource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewFindingExclusionResource())
}

func TestFindingExclusionStateRoundTrip(t *testing.T) {
	t.Parallel()

	var exclusion externalEonSdkAPI.FindingExclusion
	require.NoError(t, json.Unmarshal([]byte(findingExclusionAPIResponse), &exclusion))

	var state FindingExclusionResourceModel
	findingExclusionToState(&exclusion, &state)

	assert.Equal(t, "fe-1", state.Id.ValueString())
	assert.Equal(t, "RESOURCE", state.Scope.ValueString())
	assert.Equal(t, "i-0123456789abcdef0", state.ProviderResourceId.ValueString())
	assert.Equal(t, "/var/cache/app", state.Value.ValueString())
	assert.Equal(t, "PATH", state.Type.ValueString())
	assert.Equal(t, "MALWARE", state.Detector.ValueString())
	assert.Equal(t, "2024-01-02T03:04:05Z", state.UpdatedAt.ValueString())

	createReq, diags := findingExclusionCreateRequest(&state)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
	assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_RESOURCE, createReq.GetScope())
	assert.Equal(t, "i-0123456789abcdef0", createReq.GetProviderResourceId())
	assert.Equal(t, "/var/cache/app", createReq.GetValue())
	assert.Equal(t, externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH, createReq.GetType())
	assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE, createReq.GetDetector())

	updateReq, diags := findingExclusionUpdateRequest(&state)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
	assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_RESOURCE, updateReq.GetScope())
	assert.Equal(t, "i-0123456789abcdef0", updateReq.GetProviderResourceId())
	assert.Equal(t, "/var/cache/app", updateReq.GetValue())
	assert.Equal(t, externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH, updateReq.GetType())
	assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE, updateReq.GetDetector())
}

func TestFindingExclusionToStateProviderResourceId(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		providerResourceId *string
		wantNull           bool
		want               string
	}{
		{name: "absent", providerResourceId: nil, wantNull: true},
		{name: "empty string", providerResourceId: externalEonSdkAPI.PtrString(""), wantNull: true},
		{name: "set", providerResourceId: externalEonSdkAPI.PtrString("i-1"), want: "i-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			exclusion := externalEonSdkAPI.NewFindingExclusion(
				"fe-2",
				externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_ACCOUNT,
				"staging_events",
				externalEonSdkAPI.FINDING_OBJECT_TYPE_TABLE,
				externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_DATA_ANOMALY,
				time.Date(2024, 1, 1, 12, 0, 0, 0, time.FixedZone("plus-two", 2*60*60)),
			)
			exclusion.ProviderResourceId = tt.providerResourceId

			var state FindingExclusionResourceModel
			findingExclusionToState(exclusion, &state)

			assert.Equal(t, "ACCOUNT", state.Scope.ValueString())
			assert.Equal(t, "2024-01-01T10:00:00Z", state.UpdatedAt.ValueString(), "updated_at is normalised to UTC")
			if tt.wantNull {
				assert.True(t, state.ProviderResourceId.IsNull())
				return
			}
			assert.Equal(t, tt.want, state.ProviderResourceId.ValueString())
		})
	}
}

func TestFindingExclusionRequestsOmitUnsetProviderResourceId(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		providerResourceId types.String
	}{
		{name: "null", providerResourceId: types.StringNull()},
		{name: "unknown", providerResourceId: types.StringUnknown()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := &FindingExclusionResourceModel{
				Scope:              types.StringValue("ACCOUNT"),
				ProviderResourceId: tt.providerResourceId,
				Value:              types.StringValue("staging_events"),
				Type:               types.StringValue("TABLE"),
				Detector:           types.StringValue("DATA_ANOMALY"),
			}

			createReq, diags := findingExclusionCreateRequest(data)
			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
			assert.False(t, createReq.HasProviderResourceId())
			assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_ACCOUNT, createReq.GetScope())

			updateReq, diags := findingExclusionUpdateRequest(data)
			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
			assert.False(t, updateReq.HasProviderResourceId())
			assert.Equal(t, externalEonSdkAPI.FINDING_OBJECT_TYPE_TABLE, updateReq.GetType())
		})
	}
}

func TestFindingExclusionRequestsRejectInvalidEnums(t *testing.T) {
	t.Parallel()

	valid := FindingExclusionResourceModel{
		Scope:    types.StringValue("ACCOUNT"),
		Value:    types.StringValue("staging_events"),
		Type:     types.StringValue("TABLE"),
		Detector: types.StringValue("DATA_ANOMALY"),
	}

	tests := []struct {
		name    string
		mutate  func(*FindingExclusionResourceModel)
		wantAll bool
	}{
		{name: "invalid scope", mutate: func(m *FindingExclusionResourceModel) { m.Scope = types.StringValue("GLOBAL") }},
		{name: "invalid type", mutate: func(m *FindingExclusionResourceModel) { m.Type = types.StringValue("FILE") }},
		{name: "invalid detector", mutate: func(m *FindingExclusionResourceModel) { m.Detector = types.StringValue("VIRUS") }},
		{
			name: "every enum invalid reports every attribute",
			mutate: func(m *FindingExclusionResourceModel) {
				m.Scope = types.StringValue("GLOBAL")
				m.Type = types.StringValue("FILE")
				m.Detector = types.StringValue("VIRUS")
			},
			wantAll: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := valid
			tt.mutate(&data)

			_, diags := findingExclusionCreateRequest(&data)
			require.True(t, diags.HasError())
			if tt.wantAll {
				assert.Len(t, diags.Errors(), 3)
			} else {
				assert.Len(t, diags.Errors(), 1)
			}

			_, diags = findingExclusionUpdateRequest(&data)
			require.True(t, diags.HasError())
		})
	}
}
