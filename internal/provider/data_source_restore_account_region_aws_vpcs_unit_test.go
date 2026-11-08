package provider

import (
	"context"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreAccountRegionAwsVpcsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewRestoreAccountRegionAwsVpcsDataSource())
}

func TestRestoreAccountRegionAwsVpcsDataSource_MetadataAndSchema(t *testing.T) {
	t.Parallel()

	d := NewRestoreAccountRegionAwsVpcsDataSource()

	metaResp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "eon"}, metaResp)
	assert.Equal(t, "eon_restore_account_region_aws_vpcs", metaResp.TypeName)

	schemaResp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())
	assert.False(t, schemaResp.Schema.ValidateImplementation(context.Background()).HasError())
	for _, attr := range []string{"account_id", "region", "vpcs"} {
		assert.Contains(t, schemaResp.Schema.Attributes, attr)
	}
}

func TestRestoreAccountRegionAwsVpcsDataSource_Configure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		providerData any
		expectError  bool
	}{
		{"nil provider data", nil, false},
		{"wrong provider data type", 42, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &RestoreAccountRegionAwsVpcsDataSource{}
			resp := &datasource.ConfigureResponse{}
			d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: tt.providerData}, resp)
			assert.Equal(t, tt.expectError, resp.Diagnostics.HasError())
			assert.Nil(t, d.client)
		})
	}
}

func TestRestoreAccountRegionAwsVpcsToModel(t *testing.T) {
	t.Parallel()

	named := externalEonSdkAPI.NewListAwsVpcsResponseVpcsInner("vpc-1")
	named.SetName("main")
	unnamed := externalEonSdkAPI.NewListAwsVpcsResponseVpcsInner("vpc-2")

	tests := []struct {
		name  string
		input []externalEonSdkAPI.ListAwsVpcsResponseVpcsInner
		want  []RestoreAccountRegionAwsVpcModel
	}{
		{
			name:  "nil input yields empty list",
			input: nil,
			want:  []RestoreAccountRegionAwsVpcModel{},
		},
		{
			name:  "named and unnamed VPCs",
			input: []externalEonSdkAPI.ListAwsVpcsResponseVpcsInner{*named, *unnamed},
			want: []RestoreAccountRegionAwsVpcModel{
				{Id: types.StringValue("vpc-1"), Name: types.StringValue("main")},
				{Id: types.StringValue("vpc-2"), Name: types.StringNull()},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := restoreAccountRegionAwsVpcsToModel(tt.input)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got)
		})
	}
}
