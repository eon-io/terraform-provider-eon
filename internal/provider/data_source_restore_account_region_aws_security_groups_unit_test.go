package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestRestoreAccountRegionAwsSecurityGroupsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewRestoreAccountRegionAwsSecurityGroupsDataSource())
}

func TestBuildListAwsSecurityGroupsOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		vpcId        types.String
		subnetId     types.String
		wantVpcId    *string
		wantSubnetId *string
	}{
		{
			name:     "both unset",
			vpcId:    types.StringNull(),
			subnetId: types.StringNull(),
		},
		{
			name:         "both set",
			vpcId:        types.StringValue("vpc-1"),
			subnetId:     types.StringValue("subnet-1"),
			wantVpcId:    stringPtr("vpc-1"),
			wantSubnetId: stringPtr("subnet-1"),
		},
		{
			name:     "empty strings treated as unset",
			vpcId:    types.StringValue(""),
			subnetId: types.StringValue(""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data := RestoreAccountRegionAwsSecurityGroupsDataSourceModel{
				VpcId:    tt.vpcId,
				SubnetId: tt.subnetId,
			}

			opts := buildListAwsSecurityGroupsOptions(data)
			assert.Equal(t, tt.wantVpcId, opts.VpcId)
			assert.Equal(t, tt.wantSubnetId, opts.SubnetId)
		})
	}
}
