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
  "id": "fx-1",
  "resourceId": "res-1",
  "value": "/data/tmp",
  "type": "PATH",
  "detector": "MALWARE",
  "updatedAt": "2026-01-02T03:04:05Z"
}`

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return parsed
}

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

	assert.Equal(t, "fx-1", state.Id.ValueString())
	assert.Equal(t, "res-1", state.ResourceId.ValueString())
	assert.Equal(t, "/data/tmp", state.Value.ValueString())
	assert.Equal(t, "PATH", state.Type.ValueString())
	assert.Equal(t, "MALWARE", state.Detector.ValueString())
	assert.Equal(t, "2026-01-02T03:04:05Z", state.UpdatedAt.ValueString())

	createReq, diags := findingExclusionCreateRequest(&state)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
	assert.Equal(t, "/data/tmp", createReq.GetValue())
	assert.Equal(t, externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH, createReq.GetType())
	assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE, createReq.GetDetector())
	assert.Equal(t, "res-1", createReq.GetResourceId())

	updateReq, diags := findingExclusionUpdateRequest(&state)
	require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
	assert.Equal(t, "/data/tmp", updateReq.GetValue())
	assert.Equal(t, externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH, updateReq.GetType())
	assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE, updateReq.GetDetector())
	assert.Equal(t, "res-1", updateReq.GetResourceId())
}

func TestFindingExclusionToStateAccountWide(t *testing.T) {
	t.Parallel()

	exclusion := externalEonSdkAPI.NewFindingExclusion(
		"fx-2",
		"customers",
		externalEonSdkAPI.FINDING_OBJECT_TYPE_TABLE,
		externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_DATA_ANOMALY,
		mustParseTime(t, "2026-03-04T05:06:07+02:00"),
	)

	var state FindingExclusionResourceModel
	findingExclusionToState(exclusion, &state)

	assert.Equal(t, "fx-2", state.Id.ValueString())
	assert.True(t, state.ResourceId.IsNull())
	assert.Equal(t, "customers", state.Value.ValueString())
	assert.Equal(t, "TABLE", state.Type.ValueString())
	assert.Equal(t, "DATA_ANOMALY", state.Detector.ValueString())
	assert.Equal(t, "2026-03-04T03:06:07Z", state.UpdatedAt.ValueString(), "updated_at is normalized to UTC")
}

func TestFindingExclusionRequestsOmitNullResourceId(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		resourceId types.String
	}{
		{name: "null", resourceId: types.StringNull()},
		{name: "unknown", resourceId: types.StringUnknown()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := &FindingExclusionResourceModel{
				ResourceId: tt.resourceId,
				Value:      types.StringValue("customers"),
				Type:       types.StringValue("DATABASE"),
				Detector:   types.StringValue("RANSOMWARE_BEHAVIOR"),
			}

			createReq, diags := findingExclusionCreateRequest(data)
			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
			assert.False(t, createReq.HasResourceId())
			assert.Equal(t, externalEonSdkAPI.FINDING_OBJECT_TYPE_DATABASE, createReq.GetType())
			assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_RANSOMWARE_BEHAVIOR, createReq.GetDetector())

			updateReq, diags := findingExclusionUpdateRequest(data)
			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
			assert.False(t, updateReq.HasResourceId())
		})
	}
}

func TestFindingExclusionRequestsRejectInvalidEnums(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		typ      string
		detector string
		wantErrs int
	}{
		{name: "invalid type", typ: "FOLDER", detector: "MALWARE", wantErrs: 1},
		{name: "invalid detector", typ: "PATH", detector: "VIRUS", wantErrs: 1},
		{name: "both invalid", typ: "FOLDER", detector: "VIRUS", wantErrs: 2},
		{name: "lowercase is rejected", typ: "path", detector: "malware", wantErrs: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := &FindingExclusionResourceModel{
				Value:    types.StringValue("/data/tmp"),
				Type:     types.StringValue(tt.typ),
				Detector: types.StringValue(tt.detector),
			}

			_, diags := findingExclusionCreateRequest(data)
			require.True(t, diags.HasError())
			assert.Len(t, diags.Errors(), tt.wantErrs)

			_, diags = findingExclusionUpdateRequest(data)
			require.True(t, diags.HasError())
			assert.Len(t, diags.Errors(), tt.wantErrs)
		})
	}
}
