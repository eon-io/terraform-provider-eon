package provider

import (
	"context"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreAccountRegionAwsInstanceTypesDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewRestoreAccountRegionAwsInstanceTypesDataSource())
}

func TestBuildListAwsInstanceTypesOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		volumeCount          types.Int64
		totalVolumeBytes     types.Int64
		wantVolumeCount      *int32
		wantTotalVolumeBytes *int64
		wantErr              bool
	}{
		{
			name:             "both unset",
			volumeCount:      types.Int64Null(),
			totalVolumeBytes: types.Int64Null(),
		},
		{
			name:                 "both set",
			volumeCount:          types.Int64Value(3),
			totalVolumeBytes:     types.Int64Value(2048),
			wantVolumeCount:      int32Ptr(3),
			wantTotalVolumeBytes: int64Ptr(2048),
		},
		{
			name:        "volume count out of int32 bounds",
			volumeCount: types.Int64Value(1 << 40),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := RestoreAccountRegionAwsInstanceTypesDataSourceModel{
				VolumeCount:      tt.volumeCount,
				TotalVolumeBytes: tt.totalVolumeBytes,
			}

			opts, diags := buildListAwsInstanceTypesOptions(data)
			if tt.wantErr {
				assert.True(t, diags.HasError())
				return
			}
			require.False(t, diags.HasError())
			assert.Equal(t, tt.wantVolumeCount, opts.VolumeCount)
			assert.Equal(t, tt.wantTotalVolumeBytes, opts.TotalVolumeBytes)
		})
	}
}

func TestInstanceTypesToModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		instanceTypes  []string
		mapping        map[string]string
		wantTypesLen   int
		wantMappingLen int
	}{
		{
			name:           "with instance types and mapping",
			instanceTypes:  []string{"m5.large", "m5.xlarge"},
			mapping:        map[string]string{"m5.large": "i3.large"},
			wantTypesLen:   2,
			wantMappingLen: 1,
		},
		{
			name:           "no instance types or mapping",
			instanceTypes:  nil,
			mapping:        nil,
			wantTypesLen:   0,
			wantMappingLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			apiResp := externalEonSdkAPI.NewListInstanceTypesResponse()
			if tt.instanceTypes != nil {
				apiResp.SetInstanceTypes(tt.instanceTypes)
			}
			if tt.mapping != nil {
				apiResp.SetInstantRestoreMapping(tt.mapping)
			}

			var data RestoreAccountRegionAwsInstanceTypesDataSourceModel
			diags := instanceTypesToModel(context.Background(), apiResp, &data)
			require.False(t, diags.HasError())

			assert.False(t, data.InstanceTypes.IsNull())
			assert.Len(t, data.InstanceTypes.Elements(), tt.wantTypesLen)
			assert.False(t, data.InstantRestoreMapping.IsNull())
			assert.Len(t, data.InstantRestoreMapping.Elements(), tt.wantMappingLen)
		})
	}
}

func int32Ptr(v int32) *int32 { return &v }
func int64Ptr(v int64) *int64 { return &v }
