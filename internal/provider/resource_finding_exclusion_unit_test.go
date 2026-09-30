package provider

import (
	"testing"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindingExclusionWriteFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		model   FindingExclusionResourceModel
		wantErr string
	}{
		{
			name: "resource scope",
			model: FindingExclusionResourceModel{
				Scope:              types.StringValue("RESOURCE"),
				ProviderResourceId: types.StringValue("i-abc"),
				Value:              types.StringValue("/var/cache"),
				Type:               types.StringValue("PATH"),
				Detector:           types.StringValue("MALWARE"),
			},
		},
		{
			name: "account scope",
			model: FindingExclusionResourceModel{
				Scope:              types.StringValue("ACCOUNT"),
				ProviderResourceId: types.StringNull(),
				Value:              types.StringValue("orders"),
				Type:               types.StringValue("TABLE"),
				Detector:           types.StringValue("DATA_ANOMALY"),
			},
		},
		{
			name: "invalid scope",
			model: FindingExclusionResourceModel{
				Scope:              types.StringValue("PROJECT"),
				ProviderResourceId: types.StringNull(),
				Value:              types.StringValue("/tmp"),
				Type:               types.StringValue("PATH"),
				Detector:           types.StringValue("MALWARE"),
			},
			wantErr: "scope",
		},
		{
			name: "invalid type",
			model: FindingExclusionResourceModel{
				Scope:              types.StringValue("ACCOUNT"),
				ProviderResourceId: types.StringNull(),
				Value:              types.StringValue("/tmp"),
				Type:               types.StringValue("FILE"),
				Detector:           types.StringValue("MALWARE"),
			},
			wantErr: "type",
		},
		{
			name: "invalid detector",
			model: FindingExclusionResourceModel{
				Scope:              types.StringValue("ACCOUNT"),
				ProviderResourceId: types.StringNull(),
				Value:              types.StringValue("/tmp"),
				Type:               types.StringValue("PATH"),
				Detector:           types.StringValue("VIRUS"),
			},
			wantErr: "detector",
		},
		{
			name: "resource scope missing provider id",
			model: FindingExclusionResourceModel{
				Scope:              types.StringValue("RESOURCE"),
				ProviderResourceId: types.StringNull(),
				Value:              types.StringValue("/tmp"),
				Type:               types.StringValue("PATH"),
				Detector:           types.StringValue("MALWARE"),
			},
			wantErr: "provider_resource_id is required",
		},
		{
			name: "account scope with provider id",
			model: FindingExclusionResourceModel{
				Scope:              types.StringValue("ACCOUNT"),
				ProviderResourceId: types.StringValue("i-abc"),
				Value:              types.StringValue("app"),
				Type:               types.StringValue("DATABASE"),
				Detector:           types.StringValue("RANSOMWARE_BEHAVIOR"),
			},
			wantErr: "provider_resource_id must be omitted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			createReq, createDiags := findingExclusionCreateRequest(&tt.model)
			updateReq, updateDiags := findingExclusionUpdateRequest(&tt.model)
			if tt.wantErr != "" {
				require.True(t, createDiags.HasError())
				require.True(t, updateDiags.HasError())
				assert.Contains(t, createDiags.Errors()[0].Detail(), tt.wantErr)
				assert.Contains(t, updateDiags.Errors()[0].Detail(), tt.wantErr)
				return
			}

			require.False(t, createDiags.HasError(), "unexpected diagnostics: %v", createDiags.Errors())
			require.False(t, updateDiags.HasError(), "unexpected diagnostics: %v", updateDiags.Errors())
			assert.Equal(t, tt.model.Scope.ValueString(), string(createReq.GetScope()))
			assert.Equal(t, tt.model.Value.ValueString(), createReq.GetValue())
			assert.Equal(t, tt.model.Type.ValueString(), string(createReq.GetType()))
			assert.Equal(t, tt.model.Detector.ValueString(), string(createReq.GetDetector()))
			assert.Equal(t, createReq.GetScope(), updateReq.GetScope())
			assert.Equal(t, createReq.GetValue(), updateReq.GetValue())
			assert.Equal(t, createReq.GetType(), updateReq.GetType())
			assert.Equal(t, createReq.GetDetector(), updateReq.GetDetector())
			assert.Equal(t, createReq.HasProviderResourceId(), updateReq.HasProviderResourceId())
			if tt.model.ProviderResourceId.IsNull() {
				assert.False(t, createReq.HasProviderResourceId())
			} else {
				assert.Equal(t, tt.model.ProviderResourceId.ValueString(), createReq.GetProviderResourceId())
				assert.Equal(t, tt.model.ProviderResourceId.ValueString(), updateReq.GetProviderResourceId())
			}
		})
	}
}

func TestFindingExclusionToState(t *testing.T) {
	t.Parallel()

	updated := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	t.Run("resource scope", func(t *testing.T) {
		t.Parallel()

		exclusion := externalEonSdkAPI.NewFindingExclusion(
			"excl-1",
			externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_RESOURCE,
			"/var/cache",
			externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH,
			externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE,
			updated,
		)
		exclusion.SetProviderResourceId("i-abc")

		var state FindingExclusionResourceModel
		findingExclusionToState(exclusion, &state)

		assert.Equal(t, "excl-1", state.Id.ValueString())
		assert.Equal(t, "RESOURCE", state.Scope.ValueString())
		assert.Equal(t, "i-abc", state.ProviderResourceId.ValueString())
		assert.Equal(t, "/var/cache", state.Value.ValueString())
		assert.Equal(t, "PATH", state.Type.ValueString())
		assert.Equal(t, "MALWARE", state.Detector.ValueString())
		assert.Equal(t, "2026-01-02T03:04:05Z", state.UpdatedAt.ValueString())

		createReq, diags := findingExclusionCreateRequest(&state)
		require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
		assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_RESOURCE, createReq.GetScope())
		assert.Equal(t, "i-abc", createReq.GetProviderResourceId())
		assert.Equal(t, "/var/cache", createReq.GetValue())
		assert.Equal(t, externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH, createReq.GetType())
		assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE, createReq.GetDetector())
	})

	t.Run("account scope", func(t *testing.T) {
		t.Parallel()

		exclusion := externalEonSdkAPI.NewFindingExclusion(
			"excl-2",
			externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_ACCOUNT,
			"orders",
			externalEonSdkAPI.FINDING_OBJECT_TYPE_DATABASE,
			externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_RANSOMWARE_BEHAVIOR,
			updated,
		)

		var state FindingExclusionResourceModel
		findingExclusionToState(exclusion, &state)

		assert.True(t, state.ProviderResourceId.IsNull())
		assert.Equal(t, "ACCOUNT", state.Scope.ValueString())
		assert.Equal(t, "DATABASE", state.Type.ValueString())
		assert.Equal(t, "RANSOMWARE_BEHAVIOR", state.Detector.ValueString())

		updateReq, diags := findingExclusionUpdateRequest(&state)
		require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
		assert.False(t, updateReq.HasProviderResourceId())
		assert.Equal(t, externalEonSdkAPI.FINDING_EXCLUSION_SCOPE_ACCOUNT, updateReq.GetScope())
	})
}
