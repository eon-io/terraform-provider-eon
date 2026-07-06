package provider

import (
	"testing"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGcsBucketConfigurationModelToUpdateRequest(t *testing.T) {
	t.Parallel()

	model := &GcsBucketConfigurationResourceModel{
		ResourceId:          types.StringValue("resource-1"),
		ScanDetectionMethod: types.StringValue("FORCE_ENABLED"),
		FullScanMethod:      types.StringValue("FORCE_DISABLED"),
	}

	req, diags := gcsBucketConfigurationModelToUpdateRequest(model)
	require.False(t, diags.HasError())

	require.NotNil(t, req.CdcBackup)
	require.NotNil(t, req.CdcBackup.SystemControlled)
	require.NotNil(t, req.CdcBackup.Enabled)
	assert.False(t, *req.CdcBackup.SystemControlled)
	assert.True(t, *req.CdcBackup.Enabled)

	require.NotNil(t, req.InventoryBackup)
	require.NotNil(t, req.InventoryBackup.SystemControlled)
	require.NotNil(t, req.InventoryBackup.Enabled)
	assert.False(t, *req.InventoryBackup.SystemControlled)
	assert.False(t, *req.InventoryBackup.Enabled)
}

func TestBackupMethodToAPIConfig_Auto(t *testing.T) {
	t.Parallel()

	cfg, diags := backupMethodToAPIConfig("auto")
	require.False(t, diags.HasError())
	require.NotNil(t, cfg.SystemControlled)
	assert.True(t, *cfg.SystemControlled)
	assert.Nil(t, cfg.Enabled)
}

func TestBackupMethodToAPIConfig_Invalid(t *testing.T) {
	t.Parallel()

	_, diags := backupMethodToAPIConfig("bad")
	assert.True(t, diags.HasError())
}

func TestCloudResourceConfigurationToModel(t *testing.T) {
	t.Parallel()

	systemControlled := true
	userControlled := false
	enabled := true

	config := &client.CloudResourceConfiguration{
		CdcBackup: client.CloudResourceBackupMethodConfig{
			SystemControlled: &systemControlled,
		},
		InventoryBackup: client.CloudResourceBackupMethodConfig{
			SystemControlled: &userControlled,
			Enabled:          &enabled,
		},
	}

	model := &GcsBucketConfigurationResourceModel{
		ResourceId: types.StringValue("resource-1"),
	}
	diags := cloudResourceConfigurationToModel(config, model)
	require.False(t, diags.HasError())

	assert.Equal(t, "resource-1", model.Id.ValueString())
	assert.Equal(t, gcsBucketConfigurationMethodAuto, model.ScanDetectionMethod.ValueString())
	assert.Equal(t, gcsBucketConfigurationMethodForceEnabled, model.FullScanMethod.ValueString())
}

func TestAPIConfigToBackupMethod_UserControlledMissingEnabled(t *testing.T) {
	t.Parallel()

	systemControlled := false
	_, diags := apiConfigToBackupMethod("full_scan_method", client.CloudResourceBackupMethodConfig{
		SystemControlled: &systemControlled,
	})
	assert.True(t, diags.HasError())
}
