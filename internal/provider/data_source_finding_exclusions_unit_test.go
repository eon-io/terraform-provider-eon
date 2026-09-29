package provider

import (
	"context"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindingExclusionsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewFindingExclusionsDataSource())
}

func stringList(values ...string) types.List {
	elements := make([]attr.Value, 0, len(values))
	for _, value := range values {
		elements = append(elements, types.StringValue(value))
	}
	return types.ListValueMust(types.StringType, elements)
}

func TestFindingExclusionListItem(t *testing.T) {
	t.Parallel()

	scoped := externalEonSdkAPI.NewFindingExclusion(
		"fx-1",
		"/data/tmp",
		externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH,
		externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE,
		mustParseTime(t, "2026-01-02T03:04:05Z"),
	)
	scoped.SetResourceId("res-1")

	item := findingExclusionListItem(*scoped)
	assert.Equal(t, "fx-1", item.Id.ValueString())
	assert.Equal(t, "res-1", item.ResourceId.ValueString())
	assert.Equal(t, "/data/tmp", item.Value.ValueString())
	assert.Equal(t, "PATH", item.Type.ValueString())
	assert.Equal(t, "MALWARE", item.Detector.ValueString())
	assert.Equal(t, "2026-01-02T03:04:05Z", item.UpdatedAt.ValueString())

	accountWide := externalEonSdkAPI.NewFindingExclusion(
		"fx-2",
		"customers",
		externalEonSdkAPI.FINDING_OBJECT_TYPE_TABLE,
		externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_DATA_ANOMALY,
		mustParseTime(t, "2026-01-02T03:04:05Z"),
	)

	item = findingExclusionListItem(*accountWide)
	assert.Equal(t, "fx-2", item.Id.ValueString())
	assert.True(t, item.ResourceId.IsNull())
	assert.Equal(t, "TABLE", item.Type.ValueString())
	assert.Equal(t, "DATA_ANOMALY", item.Detector.ValueString())
}

func TestBuildFindingExclusionFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    FindingExclusionsDataSourceModel
		wantNil bool
		wantErr bool
		check   func(t *testing.T, filters *externalEonSdkAPI.FindingExclusionFilterConditions)
	}{
		{
			name: "no filters",
			data: FindingExclusionsDataSourceModel{
				ResourceIds:   types.ListNull(types.StringType),
				AccountWide:   types.BoolNull(),
				Types:         types.ListNull(types.StringType),
				Detectors:     types.ListNull(types.StringType),
				ValueContains: types.ListNull(types.StringType),
			},
			wantNil: true,
		},
		{
			name: "empty lists are no filters",
			data: FindingExclusionsDataSourceModel{
				ResourceIds:   stringList(),
				AccountWide:   types.BoolNull(),
				Types:         stringList(),
				Detectors:     stringList(),
				ValueContains: stringList(),
			},
			wantNil: true,
		},
		{
			name: "resource ids",
			data: FindingExclusionsDataSourceModel{
				ResourceIds:   stringList("res-1", "res-2"),
				AccountWide:   types.BoolNull(),
				Types:         types.ListNull(types.StringType),
				Detectors:     types.ListNull(types.StringType),
				ValueContains: types.ListNull(types.StringType),
			},
			check: func(t *testing.T, filters *externalEonSdkAPI.FindingExclusionFilterConditions) {
				require.True(t, filters.HasResource())
				resourceFilter := filters.GetResource()
				assert.Equal(t, []string{"res-1", "res-2"}, resourceFilter.GetIn())
				assert.False(t, resourceFilter.HasIsAccountWide())
				assert.False(t, filters.HasType())
				assert.False(t, filters.HasDetector())
				assert.False(t, filters.HasValue())
			},
		},
		{
			name: "account wide false",
			data: FindingExclusionsDataSourceModel{
				ResourceIds:   types.ListNull(types.StringType),
				AccountWide:   types.BoolValue(false),
				Types:         types.ListNull(types.StringType),
				Detectors:     types.ListNull(types.StringType),
				ValueContains: types.ListNull(types.StringType),
			},
			check: func(t *testing.T, filters *externalEonSdkAPI.FindingExclusionFilterConditions) {
				require.True(t, filters.HasResource())
				resourceFilter := filters.GetResource()
				require.True(t, resourceFilter.HasIsAccountWide())
				assert.False(t, resourceFilter.GetIsAccountWide())
				assert.Empty(t, resourceFilter.GetIn())
			},
		},
		{
			name: "all filters",
			data: FindingExclusionsDataSourceModel{
				ResourceIds:   stringList("res-1"),
				AccountWide:   types.BoolValue(true),
				Types:         stringList("PATH", "TABLE"),
				Detectors:     stringList("MALWARE"),
				ValueContains: stringList("tmp", "cache"),
			},
			check: func(t *testing.T, filters *externalEonSdkAPI.FindingExclusionFilterConditions) {
				resourceFilter := filters.GetResource()
				assert.Equal(t, []string{"res-1"}, resourceFilter.GetIn())
				assert.True(t, resourceFilter.GetIsAccountWide())

				typeFilter := filters.GetType()
				assert.Equal(t, []externalEonSdkAPI.FindingObjectType{
					externalEonSdkAPI.FINDING_OBJECT_TYPE_PATH,
					externalEonSdkAPI.FINDING_OBJECT_TYPE_TABLE,
				}, typeFilter.GetIn())
				assert.Empty(t, typeFilter.GetNotIn())

				detectorFilter := filters.GetDetector()
				assert.Equal(t, []externalEonSdkAPI.FindingExclusionDetectorType{
					externalEonSdkAPI.FINDING_EXCLUSION_DETECTOR_TYPE_MALWARE,
				}, detectorFilter.GetIn())

				valueFilter := filters.GetValue()
				assert.Equal(t, []string{"tmp", "cache"}, valueFilter.GetContains())
			},
		},
		{
			name: "invalid type",
			data: FindingExclusionsDataSourceModel{
				ResourceIds:   types.ListNull(types.StringType),
				AccountWide:   types.BoolNull(),
				Types:         stringList("FOLDER"),
				Detectors:     types.ListNull(types.StringType),
				ValueContains: types.ListNull(types.StringType),
			},
			wantErr: true,
		},
		{
			name: "invalid detector",
			data: FindingExclusionsDataSourceModel{
				ResourceIds:   types.ListNull(types.StringType),
				AccountWide:   types.BoolNull(),
				Types:         types.ListNull(types.StringType),
				Detectors:     stringList("VIRUS"),
				ValueContains: types.ListNull(types.StringType),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filters, diags := buildFindingExclusionFilters(context.Background(), &tt.data)
			if tt.wantErr {
				require.True(t, diags.HasError())
				assert.Nil(t, filters)
				return
			}
			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags.Errors())
			if tt.wantNil {
				assert.Nil(t, filters)
				return
			}
			require.NotNil(t, filters)
			tt.check(t, filters)
		})
	}
}
