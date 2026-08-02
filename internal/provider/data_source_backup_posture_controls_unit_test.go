package provider

import (
	"context"
	"fmt"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackupPostureControlsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewBackupPostureControlsDataSource())
}

func TestBackupPostureControlsDataSource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shouldFail  bool
		numControls int
		expectError bool
	}{
		{"successful list with multiple controls", false, 2, false},
		{"successful list with no controls", false, 0, false},
		{"list failure", true, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailBackupPostureControlList = tt.shouldFail
			for i := 0; i < tt.numControls; i++ {
				mockClient.AddMockBackupPostureControl(&externalEonSdkAPI.BackupPostureControl{
					Id:       fmt.Sprintf("bpc-%d", i+1),
					Name:     fmt.Sprintf("control-%d", i+1),
					Severity: externalEonSdkAPI.HIGH,
				})
			}

			result, err := mockClient.ListBackupPostureControls(context.Background())
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				assert.Len(t, result, tt.numControls)
			}
			assert.Equal(t, 1, mockClient.BackupPostureControlListCalls)
		})
	}
}
