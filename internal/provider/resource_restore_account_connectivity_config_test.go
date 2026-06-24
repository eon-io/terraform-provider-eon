package provider

import (
	"context"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreAccountConnectivityConfigResource_Metadata(t *testing.T) {
	t.Parallel()

	r := NewRestoreAccountConnectivityConfigResource()
	require.NotNil(t, r)
}

func TestModelToUpdateRequest_Aws(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	restoreServer, diags := types.ListValueFrom(ctx, types.StringType, []string{"sg-restore"})
	require.False(t, diags.HasError())
	restoredRds, diags := types.ListValueFrom(ctx, types.StringType, []string{"sg-rds"})
	require.False(t, diags.HasError())

	model := &RestoreAccountConnectivityConfigResourceModel{
		RestoreAccountId: types.StringValue("acct-1"),
		Aws: &awsConnectivityConfigModel{
			VpcConfigs: []awsVpcConnectivityConfigModel{
				{
					Region: types.StringValue("us-east-1"),
					Vpc:    types.StringValue("vpc-123"),
					SubnetsPerAvailabilityZone: []subnetPerAvailabilityZoneModel{
						{
							AvailabilityZone: types.StringValue("us-east-1a"),
							SubnetId:         types.StringValue("subnet-123"),
						},
					},
					SecurityGroups: &resourceTypeToSecurityGroupModel{
						RestoreServer:       restoreServer,
						RestoredRdsInstance: restoredRds,
					},
				},
			},
		},
	}

	req, diags := modelToUpdateRequest(ctx, model)
	require.False(t, diags.HasError(), "conversion should not error")
	require.True(t, req.HasAws())
	assert.False(t, req.HasGcp())

	aws := req.GetAws()
	require.Len(t, aws.VpcConfigs, 1)
	vpc := aws.VpcConfigs[0]
	assert.Equal(t, "us-east-1", vpc.Region)
	assert.Equal(t, "vpc-123", vpc.Vpc)
	require.Len(t, vpc.SubnetsPerAvailabilityZone, 1)
	assert.Equal(t, "us-east-1a", vpc.SubnetsPerAvailabilityZone[0].AvailabilityZone)
	assert.Equal(t, "subnet-123", vpc.SubnetsPerAvailabilityZone[0].SubnetId)
	require.NotNil(t, vpc.SecurityGroups)
	assert.Equal(t, []string{"sg-restore"}, vpc.SecurityGroups.RestoreServer)
	assert.Equal(t, []string{"sg-rds"}, vpc.SecurityGroups.RestoredRdsInstance)
}

func TestModelToUpdateRequest_Gcp(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	model := &RestoreAccountConnectivityConfigResourceModel{
		RestoreAccountId: types.StringValue("acct-2"),
		Gcp: &gcpConnectivityConfigModel{
			NetworkConfigs: []gcpNetworkConnectivityConfigModel{
				{
					Network:            types.StringValue("net-1"),
					IsSharedVpc:        types.BoolValue(true),
					NetworkHostProject: types.StringValue("host-project"),
					SubnetsPerRegion: []gcpSubnetPerRegionModel{
						{
							Region:     types.StringValue("us-east1"),
							SubnetName: types.StringValue("subnet-1"),
						},
					},
				},
			},
		},
	}

	req, diags := modelToUpdateRequest(ctx, model)
	require.False(t, diags.HasError())
	require.True(t, req.HasGcp())
	assert.False(t, req.HasAws())

	gcp := req.GetGcp()
	require.Len(t, gcp.NetworkConfigs, 1)
	net := gcp.NetworkConfigs[0]
	assert.Equal(t, "net-1", net.Network)
	assert.True(t, net.IsSharedVpc)
	assert.Equal(t, "host-project", net.NetworkHostProject)
	require.Len(t, net.SubnetsPerRegion, 1)
	assert.Equal(t, "us-east1", net.SubnetsPerRegion[0].Region)
	assert.Equal(t, "subnet-1", net.SubnetsPerRegion[0].SubnetName)
}

func TestConnectivityConfigToModel_RoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	awsCfg := externalEonSdkAPI.AwsRestoreAccountConnectivityConfig{
		VpcConfigs: []externalEonSdkAPI.AwsVpcConnectivityConfig{
			{
				Region: "eu-west-1",
				Vpc:    "vpc-999",
				SubnetsPerAvailabilityZone: []externalEonSdkAPI.SubnetPerAvailabilityZone{
					{AvailabilityZone: "eu-west-1a", SubnetId: "subnet-999"},
				},
				SecurityGroups: &externalEonSdkAPI.ResourceTypeToSecurityGroup{
					RestoreServer: []string{"sg-a"},
				},
			},
		},
	}
	provider := externalEonSdkAPI.ProviderRestoreAccountConnectivityConfig{}
	provider.SetAws(awsCfg)
	config := externalEonSdkAPI.RestoreAccountConnectivityConfig{
		RestoreAccountId: "acct-3",
		Config:           provider,
	}

	model := &RestoreAccountConnectivityConfigResourceModel{
		RestoreAccountId: types.StringValue("acct-3"),
	}
	diags := connectivityConfigToModel(ctx, &config, model)
	require.False(t, diags.HasError())

	require.NotNil(t, model.Aws)
	assert.Nil(t, model.Gcp)
	require.Len(t, model.Aws.VpcConfigs, 1)
	vpc := model.Aws.VpcConfigs[0]
	assert.Equal(t, "eu-west-1", vpc.Region.ValueString())
	assert.Equal(t, "vpc-999", vpc.Vpc.ValueString())
	require.Len(t, vpc.SubnetsPerAvailabilityZone, 1)
	assert.Equal(t, "eu-west-1a", vpc.SubnetsPerAvailabilityZone[0].AvailabilityZone.ValueString())
	require.NotNil(t, vpc.SecurityGroups)
	// restored_rds_instance was absent -> null list, restore_server -> populated.
	assert.True(t, vpc.SecurityGroups.RestoredRdsInstance.IsNull())
	var restoreServer []string
	vpc.SecurityGroups.RestoreServer.ElementsAs(ctx, &restoreServer, false)
	assert.Equal(t, []string{"sg-a"}, restoreServer)
}
