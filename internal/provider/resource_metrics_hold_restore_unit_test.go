package provider

import (
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelToEnableMetricsRequest(t *testing.T) {
	t.Parallel()

	t.Run("with aws region", func(t *testing.T) {
		t.Parallel()
		data := &RestoreAccountMetricsConfigResourceModel{
			RestoreAccountId: types.StringValue("acct-1"),
			Aws: &awsAccountMetricsDestModel{
				Region: types.StringValue("eu-west-1"),
			},
		}
		req := modelToEnableMetricsRequest(data)
		require.True(t, req.HasAws())
		aws := req.GetAws()
		assert.Equal(t, "eu-west-1", aws.GetRegion())
	})

	t.Run("without aws block", func(t *testing.T) {
		t.Parallel()
		data := &RestoreAccountMetricsConfigResourceModel{
			RestoreAccountId: types.StringValue("acct-1"),
		}
		req := modelToEnableMetricsRequest(data)
		assert.False(t, req.HasAws())
	})
}

func TestMetricsConfigToModel(t *testing.T) {
	t.Parallel()

	destination := externalEonSdkAPI.NewAccountMetricsDestination()
	aws := externalEonSdkAPI.NewAwsAccountMetricsDestination()
	aws.SetRegion("ap-southeast-1")
	destination.SetAws(*aws)
	config := externalEonSdkAPI.NewRestoreAccountMetricsConfig("acct-1", true, *destination)

	data := &RestoreAccountMetricsConfigResourceModel{}
	diags := metricsConfigToModel(config, data)
	require.False(t, diags.HasError())
	require.NotNil(t, data.Aws)
	assert.Equal(t, "ap-southeast-1", data.Aws.Region.ValueString())
}

func TestAzureDiskSettingsFromModel(t *testing.T) {
	t.Parallel()

	settings, err := azureDiskSettingsFromModel(
		t.Context(),
		types.StringValue("disk-1"),
		types.StringValue("Premium_LRS"),
		types.StringValue("P10"),
		types.StringValue("V2"),
		types.Int64Value(10737418240),
		types.MapNull(types.StringType),
	)
	require.NoError(t, err)
	assert.Equal(t, "disk-1", settings.Name)
	assert.Equal(t, "Premium_LRS", settings.Type)
	assert.Equal(t, "P10", settings.Tier)
	require.NotNil(t, settings.HyperVGeneration)
	assert.Equal(t, "V2", *settings.HyperVGeneration)
	require.NotNil(t, settings.SizeBytes)
	assert.Equal(t, int64(10737418240), *settings.SizeBytes)
}
