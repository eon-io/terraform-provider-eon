package provider

import (
	"context"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreAccountRegionEncryptionKeysDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewRestoreAccountRegionEncryptionKeysDataSource())
}

func TestRestoreAccountRegionEncryptionKeysDataSource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shouldFail  bool
		expectError bool
	}{
		{"successful list", false, false},
		{"list failure", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailListEncryptionKeys = tt.shouldFail
			mockClient.SetMockEncryptionKeys("acct-1", "us-east-1", externalEonSdkAPI.NewEncryptionKeys(
				[]externalEonSdkAPI.EncryptionKey{
					*externalEonSdkAPI.NewEncryptionKey("key-1", "arn:aws:kms:us-east-1:123456789012:key/key-1", "alias/restore"),
				},
			))

			includeAwsManagedKeys := true
			result, err := mockClient.ListEncryptionKeys(context.Background(), "acct-1", "us-east-1", client.ListEncryptionKeysOptions{
				IncludeAwsManagedKeys: &includeAwsManagedKeys,
			})
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, result)
				require.Len(t, result.GetKeys(), 1)
				assert.Equal(t, "key-1", result.GetKeys()[0].GetId())
				assert.Equal(t, "alias/restore", result.GetKeys()[0].GetAliasName())
			}
			assert.Equal(t, 1, mockClient.ListEncryptionKeysCalls)
		})
	}
}

func TestRestoreAccountRegionEncryptionKeysDataSource_UnknownRegionReturnsEmpty(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()

	result, err := mockClient.ListEncryptionKeys(context.Background(), "acct-1", "us-east-1", client.ListEncryptionKeysOptions{})

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.GetKeys())
	assert.Equal(t, 1, mockClient.ListEncryptionKeysCalls)
}

func TestRestoreAccountRegionEncryptionKeysToModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    []externalEonSdkAPI.EncryptionKey
		expected []RestoreAccountRegionEncryptionKeyModel
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: []RestoreAccountRegionEncryptionKeyModel{},
		},
		{
			name: "keys are mapped in order",
			input: []externalEonSdkAPI.EncryptionKey{
				*externalEonSdkAPI.NewEncryptionKey("key-1", "arn-1", "alias/one"),
				*externalEonSdkAPI.NewEncryptionKey("key-2", "arn-2", "alias/two"),
			},
			expected: []RestoreAccountRegionEncryptionKeyModel{
				{Id: types.StringValue("key-1"), Arn: types.StringValue("arn-1"), AliasName: types.StringValue("alias/one")},
				{Id: types.StringValue("key-2"), Arn: types.StringValue("arn-2"), AliasName: types.StringValue("alias/two")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, restoreAccountRegionEncryptionKeysToModel(tt.input))
		})
	}
}
