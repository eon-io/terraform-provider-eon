package provider

import (
	"testing"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestS3BucketConfigurationModelToUpdateRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		scanMode            string
		changeDetectionMode string
		wantInventory       client.CloudResourceBackupMethodConfig
		wantCdc             client.CloudResourceBackupMethodConfig
	}{
		{
			name:                "auto",
			scanMode:            "AUTO",
			changeDetectionMode: "auto",
			wantInventory:       client.CloudResourceBackupMethodConfig{SystemControlled: new(true)},
			wantCdc:             client.CloudResourceBackupMethodConfig{SystemControlled: new(true)},
		},
		{
			name:                "inventory with change events",
			scanMode:            "INVENTORY",
			changeDetectionMode: "SCAN_AND_CHANGE_EVENTS",
			wantInventory:       client.CloudResourceBackupMethodConfig{SystemControlled: new(false), Enabled: new(true)},
			wantCdc:             client.CloudResourceBackupMethodConfig{SystemControlled: new(false), Enabled: new(true)},
		},
		{
			name:                "list objects scan only",
			scanMode:            " list_objects ",
			changeDetectionMode: "SCAN_ONLY",
			wantInventory:       client.CloudResourceBackupMethodConfig{SystemControlled: new(false), Enabled: new(false)},
			wantCdc:             client.CloudResourceBackupMethodConfig{SystemControlled: new(false), Enabled: new(false)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req, diags := s3BucketConfigurationModelToUpdateRequest(&S3BucketConfigurationResourceModel{
				ResourceId:          types.StringValue("resource-1"),
				ScanMode:            types.StringValue(tt.scanMode),
				ChangeDetectionMode: types.StringValue(tt.changeDetectionMode),
			})
			require.False(t, diags.HasError(), diags)
			require.NotNil(t, req.InventoryBackup)
			require.NotNil(t, req.CdcBackup)
			assert.Equal(t, tt.wantInventory, *req.InventoryBackup)
			assert.Equal(t, tt.wantCdc, *req.CdcBackup)
		})
	}
}

func TestS3BucketConfigurationModelToUpdateRequest_RejectsOtherAttributesValues(t *testing.T) {
	t.Parallel()

	_, diags := s3BucketConfigurationModelToUpdateRequest(&S3BucketConfigurationResourceModel{
		ScanMode:            types.StringValue("SCAN_ONLY"),
		ChangeDetectionMode: types.StringValue("INVENTORY"),
	})
	require.True(t, diags.HasError())
	assert.Len(t, diags.Errors(), 2)
}

func TestS3BucketConfigurationToModel(t *testing.T) {
	t.Parallel()

	model := &S3BucketConfigurationResourceModel{ResourceId: types.StringValue("resource-1")}
	diags := s3BucketConfigurationToModel(&client.CloudResourceConfiguration{
		InventoryBackup: client.CloudResourceBackupMethodConfig{SystemControlled: new(false), Enabled: new(false)},
		CdcBackup:       client.CloudResourceBackupMethodConfig{SystemControlled: new(true)},
	}, model)
	require.False(t, diags.HasError(), diags)

	assert.Equal(t, "resource-1", model.Id.ValueString())
	assert.Equal(t, s3ScanModeListObjects, model.ScanMode.ValueString())
	assert.Equal(t, s3BucketModeAuto, model.ChangeDetectionMode.ValueString())
}

func TestS3BucketConfigurationToModel_UserControlledWithoutEnabled(t *testing.T) {
	t.Parallel()

	diags := s3BucketConfigurationToModel(&client.CloudResourceConfiguration{
		InventoryBackup: client.CloudResourceBackupMethodConfig{SystemControlled: new(false)},
		CdcBackup:       client.CloudResourceBackupMethodConfig{SystemControlled: new(true)},
	}, &S3BucketConfigurationResourceModel{})
	assert.True(t, diags.HasError())
}
