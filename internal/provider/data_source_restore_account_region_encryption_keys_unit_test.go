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

func TestRestoreAccountRegionEncryptionKeysDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewRestoreAccountRegionEncryptionKeysDataSource())
}

func TestRestoreAccountRegionEncryptionKeysDataSource_MetadataAndSchema(t *testing.T) {
	t.Parallel()

	d := NewRestoreAccountRegionEncryptionKeysDataSource()

	metaResp := &datasource.MetadataResponse{}
	d.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "eon"}, metaResp)
	assert.Equal(t, "eon_restore_account_region_encryption_keys", metaResp.TypeName)

	schemaResp := &datasource.SchemaResponse{}
	d.Schema(context.Background(), datasource.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())
	assert.False(t, schemaResp.Schema.ValidateImplementation(context.Background()).HasError())
	for _, attr := range []string{"account_id", "region", "include_aws_managed_keys", "keys"} {
		assert.Contains(t, schemaResp.Schema.Attributes, attr)
	}
}

func TestRestoreAccountRegionEncryptionKeysDataSource_Configure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		providerData any
		expectError  bool
	}{
		{"nil provider data", nil, false},
		{"wrong provider data type", struct{}{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := &RestoreAccountRegionEncryptionKeysDataSource{}
			resp := &datasource.ConfigureResponse{}
			d.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: tt.providerData}, resp)
			assert.Equal(t, tt.expectError, resp.Diagnostics.HasError())
			assert.Nil(t, d.client)
		})
	}
}

func TestRestoreAccountRegionEncryptionKeysToModel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []externalEonSdkAPI.EncryptionKey
		want  []RestoreAccountRegionEncryptionKeyModel
	}{
		{
			name:  "nil input yields empty list",
			input: nil,
			want:  []RestoreAccountRegionEncryptionKeyModel{},
		},
		{
			name: "keys preserve order and fields",
			input: []externalEonSdkAPI.EncryptionKey{
				*externalEonSdkAPI.NewEncryptionKey("key-1", "arn:aws:kms:us-east-1:123456789012:key/key-1", "alias/restore"),
				*externalEonSdkAPI.NewEncryptionKey("key-2", "arn:aws:kms:us-east-1:123456789012:key/key-2", "alias/aws/ebs"),
			},
			want: []RestoreAccountRegionEncryptionKeyModel{
				{
					Id:        types.StringValue("key-1"),
					Arn:       types.StringValue("arn:aws:kms:us-east-1:123456789012:key/key-1"),
					AliasName: types.StringValue("alias/restore"),
				},
				{
					Id:        types.StringValue("key-2"),
					Arn:       types.StringValue("arn:aws:kms:us-east-1:123456789012:key/key-2"),
					AliasName: types.StringValue("alias/aws/ebs"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := restoreAccountRegionEncryptionKeysToModel(tt.input)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, got)
		})
	}
}
