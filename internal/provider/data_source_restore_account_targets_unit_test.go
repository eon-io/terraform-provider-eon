package provider

import (
	"context"
	"math"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreAccountTargetDataSources_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewRestoreAccountAwsS3BucketsDataSource())
	assert.NotNil(t, NewRestoreAccountRegionAwsInstanceTypesDataSource())
	assert.NotNil(t, NewRestoreAccountRegionAwsSecurityGroupsDataSource())
}

func TestRestoreAccountAwsS3BucketsDataSource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shouldFail  bool
		numBuckets  int
		expectError bool
	}{
		{"successful list with multiple buckets", false, 2, false},
		{"successful list with no buckets", false, 0, false},
		{"list failure", true, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailListRestoreAccountS3Buckets = tt.shouldFail
			for i := 0; i < tt.numBuckets; i++ {
				mockClient.AddMockRestoreAccountS3Bucket("ra-1", *externalEonSdkAPI.NewS3Bucket("bucket", "us-east-1"))
			}

			result, err := mockClient.ListRestoreAccountS3Buckets(context.Background(), "ra-1")
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, restoreAccountS3BucketModels(result), tt.numBuckets)
			}
			assert.Equal(t, 1, mockClient.ListRestoreAccountS3BucketsCalls)
		})
	}
}

func TestRestoreAccountS3BucketModels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		buckets []externalEonSdkAPI.S3Bucket
		want    []RestoreAccountS3BucketModel
	}{
		{name: "nil input", buckets: nil, want: []RestoreAccountS3BucketModel{}},
		{
			name: "maps bucket name and region",
			buckets: []externalEonSdkAPI.S3Bucket{
				*externalEonSdkAPI.NewS3Bucket("restore-a", "us-east-1"),
				*externalEonSdkAPI.NewS3Bucket("restore-b", "eu-west-1"),
			},
			want: []RestoreAccountS3BucketModel{
				{BucketName: types.StringValue("restore-a"), Region: types.StringValue("us-east-1")},
				{BucketName: types.StringValue("restore-b"), Region: types.StringValue("eu-west-1")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, restoreAccountS3BucketModels(tt.buckets))
		})
	}
}

func TestRestoreAccountRegionAwsSecurityGroupsDataSource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shouldFail  bool
		vpcId       types.String
		wantIDs     []string
		expectError bool
	}{
		{name: "all security groups", vpcId: types.StringNull(), wantIDs: []string{"sg-1", "sg-2"}},
		{name: "filtered by vpc", vpcId: types.StringValue("vpc-2"), wantIDs: []string{"sg-2"}},
		{name: "list failure", shouldFail: true, vpcId: types.StringNull(), expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailListRestoreAccountSecurityGroups = tt.shouldFail
			mockClient.AddMockRestoreAccountSecurityGroup("ra-1", "us-east-1", *externalEonSdkAPI.NewRestoreSecurityGroupsSecurityGroupsInner("default", "sg-1", "vpc-1"))
			mockClient.AddMockRestoreAccountSecurityGroup("ra-1", "us-east-1", *externalEonSdkAPI.NewRestoreSecurityGroupsSecurityGroupsInner("web", "sg-2", "vpc-2"))

			result, err := mockClient.ListRestoreAccountAwsSecurityGroups(context.Background(), "ra-1", "us-east-1", optionalStringPointer(tt.vpcId), nil)
			assert.Equal(t, 1, mockClient.ListRestoreAccountSecurityGroupsCalls)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			models := restoreAccountAwsSecurityGroupModels(result)
			ids := make([]string, 0, len(models))
			for _, m := range models {
				ids = append(ids, m.Id.ValueString())
			}
			assert.Equal(t, tt.wantIDs, ids)
		})
	}
}

func TestRestoreAccountAwsSecurityGroupModels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		groups []externalEonSdkAPI.RestoreSecurityGroupsSecurityGroupsInner
		want   []RestoreAccountAwsSecurityGroupModel
	}{
		{name: "nil input", groups: nil, want: []RestoreAccountAwsSecurityGroupModel{}},
		{
			name: "maps id, name and vpc",
			groups: []externalEonSdkAPI.RestoreSecurityGroupsSecurityGroupsInner{
				*externalEonSdkAPI.NewRestoreSecurityGroupsSecurityGroupsInner("default", "sg-1", "vpc-1"),
			},
			want: []RestoreAccountAwsSecurityGroupModel{
				{Id: types.StringValue("sg-1"), Name: types.StringValue("default"), VpcId: types.StringValue("vpc-1")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, restoreAccountAwsSecurityGroupModels(tt.groups))
		})
	}
}

func TestOptionalStringPointer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value types.String
		want  *string
	}{
		{name: "null", value: types.StringNull(), want: nil},
		{name: "unknown", value: types.StringUnknown(), want: nil},
		{name: "empty", value: types.StringValue(""), want: nil},
		{name: "set", value: types.StringValue("vpc-1"), want: externalEonSdkAPI.PtrString("vpc-1")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, optionalStringPointer(tt.value))
		})
	}
}

func TestRestoreAccountRegionAwsInstanceTypesDataSource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shouldFail  bool
		expectError bool
	}{
		{name: "successful list", shouldFail: false},
		{name: "list failure", shouldFail: true, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailListRestoreAccountInstanceTypes = tt.shouldFail
			resp := externalEonSdkAPI.NewListInstanceTypesResponse()
			resp.SetInstanceTypes([]string{"t3.micro", "m5.large"})
			mockClient.SetMockRestoreAccountInstanceTypes("ra-1", "us-east-1", resp)

			result, err := mockClient.ListRestoreAccountAwsInstanceTypes(context.Background(), "ra-1", "us-east-1", "subnet-1", client.ListAwsInstanceTypesOptions{})
			assert.Equal(t, 1, mockClient.ListRestoreAccountInstanceTypesCalls)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []string{"t3.micro", "m5.large"}, result.GetInstanceTypes())
		})
	}
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
		{name: "both omitted", volumeCount: types.Int64Null(), totalVolumeBytes: types.Int64Null()},
		{name: "both unknown", volumeCount: types.Int64Unknown(), totalVolumeBytes: types.Int64Unknown()},
		{
			name:                 "both set",
			volumeCount:          types.Int64Value(4),
			totalVolumeBytes:     types.Int64Value(5368709120),
			wantVolumeCount:      externalEonSdkAPI.PtrInt32(4),
			wantTotalVolumeBytes: externalEonSdkAPI.PtrInt64(5368709120),
		},
		{name: "volume count overflows int32", volumeCount: types.Int64Value(math.MaxInt32 + 1), totalVolumeBytes: types.Int64Null(), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts, err := buildListAwsInstanceTypesOptions(RestoreAccountRegionAwsInstanceTypesDataSourceModel{
				VolumeCount:      tt.volumeCount,
				TotalVolumeBytes: tt.totalVolumeBytes,
			})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "volume_count")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantVolumeCount, opts.VolumeCount)
			assert.Equal(t, tt.wantTotalVolumeBytes, opts.TotalVolumeBytes)
		})
	}
}

func TestSetInstanceTypesFromResponse(t *testing.T) {
	t.Parallel()

	withMapping := externalEonSdkAPI.NewListInstanceTypesResponse()
	withMapping.SetInstanceTypes([]string{"m5.large", "t3.micro"})
	withMapping.SetInstantRestoreMapping(map[string]string{"m5.large": "m5.xlarge"})

	tests := []struct {
		name        string
		resp        *externalEonSdkAPI.ListInstanceTypesResponse
		wantTypes   []string
		wantMapping map[string]string
	}{
		{name: "empty response", resp: externalEonSdkAPI.NewListInstanceTypesResponse(), wantTypes: []string{}, wantMapping: map[string]string{}},
		{name: "types and mapping", resp: withMapping, wantTypes: []string{"m5.large", "t3.micro"}, wantMapping: map[string]string{"m5.large": "m5.xlarge"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var data RestoreAccountRegionAwsInstanceTypesDataSourceModel
			diags := setInstanceTypesFromResponse(context.Background(), &data, tt.resp)
			require.False(t, diags.HasError(), "unexpected diagnostics: %v", diags)

			var gotTypes []string
			require.False(t, data.InstanceTypes.ElementsAs(context.Background(), &gotTypes, false).HasError())
			assert.Equal(t, tt.wantTypes, gotTypes)

			var gotMapping map[string]string
			require.False(t, data.InstantRestoreMapping.ElementsAs(context.Background(), &gotMapping, false).HasError())
			assert.Equal(t, tt.wantMapping, gotMapping)
		})
	}
}
