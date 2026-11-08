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

func TestRestoreAccountRegionAwsSubnetsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewRestoreAccountRegionAwsSubnetsDataSource())
}

func TestRestoreAccountRegionAwsSubnetsDataSource_MetadataAndSchema(t *testing.T) {
	t.Parallel()

	d := NewRestoreAccountRegionAwsSubnetsDataSource()

	metaResp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "eon"}, metaResp)
	assert.Equal(t, "eon_restore_account_region_aws_subnets", metaResp.TypeName)

	schemaResp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())
	assert.False(t, schemaResp.Schema.ValidateImplementation(context.Background()).HasError())
	for _, attr := range []string{"account_id", "region", "subnets"} {
		assert.Contains(t, schemaResp.Schema.Attributes, attr)
	}
}

func TestRestoreAccountRegionAwsSubnetsDataSource_Configure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		providerData any
		expectError  bool
	}{
		{"nil provider data", nil, false},
		{"wrong provider data type", "not-a-client", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &RestoreAccountRegionAwsSubnetsDataSource{}
			resp := &datasource.ConfigureResponse{}
			d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: tt.providerData}, resp)
			assert.Equal(t, tt.expectError, resp.Diagnostics.HasError())
			assert.Nil(t, d.client)
		})
	}
}

func TestRestoreAccountRegionAwsSubnetsToModel(t *testing.T) {
	t.Parallel()

	named := externalEonSdkAPI.NewListSubnetsResponseV2SubnetsInner("subnet-1", "vpc-1", "us-east-1a")
	named.SetName("private-a")
	named.SetCidrBlock("10.0.0.0/24")
	unnamed := externalEonSdkAPI.NewListSubnetsResponseV2SubnetsInner("subnet-2", "vpc-2", "us-east-1b")

	tests := []struct {
		name  string
		input []externalEonSdkAPI.ListSubnetsResponseV2SubnetsInner
		want  []RestoreAccountRegionAwsSubnetModel
	}{
		{
			name:  "nil input yields empty list",
			input: nil,
			want:  []RestoreAccountRegionAwsSubnetModel{},
		},
		{
			name:  "all fields populated",
			input: []externalEonSdkAPI.ListSubnetsResponseV2SubnetsInner{*named},
			want: []RestoreAccountRegionAwsSubnetModel{{
				Id:               types.StringValue("subnet-1"),
				Vpc:              types.StringValue("vpc-1"),
				AvailabilityZone: types.StringValue("us-east-1a"),
				Name:             types.StringValue("private-a"),
				CidrBlock:        types.StringValue("10.0.0.0/24"),
			}},
		},
		{
			name:  "optional fields absent map to null",
			input: []externalEonSdkAPI.ListSubnetsResponseV2SubnetsInner{*unnamed},
			want: []RestoreAccountRegionAwsSubnetModel{{
				Id:               types.StringValue("subnet-2"),
				Vpc:              types.StringValue("vpc-2"),
				AvailabilityZone: types.StringValue("us-east-1b"),
				Name:             types.StringNull(),
				CidrBlock:        types.StringNull(),
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := restoreAccountRegionAwsSubnetsToModel(tt.input)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got)
		})
	}
}
