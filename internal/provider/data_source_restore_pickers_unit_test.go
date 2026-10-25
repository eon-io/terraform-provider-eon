package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestorePickerDataSources_Unit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		newDataSource func() datasource.DataSource
		requiredAttrs []string
		optionalAttrs []string
		computedAttrs []string
		nestedAttrs   map[string][]string
	}{
		{
			name:          "eon_restore_aws_vpcs",
			newDataSource: NewRestoreAwsVpcsDataSource,
			requiredAttrs: []string{"restore_account_id", "region"},
			computedAttrs: []string{"vpcs"},
			nestedAttrs:   map[string][]string{"vpcs": {"id", "name"}},
		},
		{
			name:          "eon_restore_aws_subnets",
			newDataSource: NewRestoreAwsSubnetsDataSource,
			requiredAttrs: []string{"restore_account_id", "region"},
			computedAttrs: []string{"subnets"},
			nestedAttrs:   map[string][]string{"subnets": {"id", "vpc"}},
		},
		{
			name:          "eon_restore_aws_security_groups",
			newDataSource: NewRestoreAwsSecurityGroupsDataSource,
			requiredAttrs: []string{"restore_account_id", "region"},
			optionalAttrs: []string{"vpc_id", "subnet_id"},
			computedAttrs: []string{"security_groups"},
			nestedAttrs:   map[string][]string{"security_groups": {"id", "name", "vpc"}},
		},
		{
			name:          "eon_restore_aws_instance_types",
			newDataSource: NewRestoreAwsInstanceTypesDataSource,
			requiredAttrs: []string{"restore_account_id", "region", "subnet_id"},
			optionalAttrs: []string{"volume_count", "total_volume_bytes"},
			computedAttrs: []string{"instance_types"},
		},
		{
			name:          "eon_restore_encryption_keys",
			newDataSource: NewRestoreEncryptionKeysDataSource,
			requiredAttrs: []string{"restore_account_id", "region"},
			optionalAttrs: []string{"include_aws_managed_keys"},
			computedAttrs: []string{"keys"},
			nestedAttrs:   map[string][]string{"keys": {"id", "arn", "alias_name"}},
		},
		{
			name:          "eon_restore_s3_buckets",
			newDataSource: NewRestoreS3BucketsDataSource,
			requiredAttrs: []string{"restore_account_id"},
			computedAttrs: []string{"buckets"},
			nestedAttrs:   map[string][]string{"buckets": {"bucket_name", "region_name"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			dataSource := test.newDataSource()
			require.NotNil(t, dataSource)

			var metadata datasource.MetadataResponse
			dataSource.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "eon"}, &metadata)
			assert.Equal(t, test.name, metadata.TypeName)

			var schemaResponse datasource.SchemaResponse
			dataSource.Schema(context.Background(), datasource.SchemaRequest{}, &schemaResponse)
			for _, name := range test.requiredAttrs {
				assertAttributeMode(t, schemaResponse.Schema.Attributes[name], true, false, false)
			}
			for _, name := range test.optionalAttrs {
				assertAttributeMode(t, schemaResponse.Schema.Attributes[name], false, true, false)
			}
			for _, name := range test.computedAttrs {
				assertAttributeMode(t, schemaResponse.Schema.Attributes[name], false, false, true)
			}
			for name, expectedChildren := range test.nestedAttrs {
				nested, ok := schemaResponse.Schema.Attributes[name].(schema.ListNestedAttribute)
				require.True(t, ok, "%s should be a nested list", name)
				childNames := make([]string, 0, len(nested.NestedObject.Attributes))
				for childName := range nested.NestedObject.Attributes {
					childNames = append(childNames, childName)
				}
				assert.ElementsMatch(t, expectedChildren, childNames)
			}
		})
	}
}

func assertAttributeMode(t *testing.T, attribute schema.Attribute, required, optional, computed bool) {
	t.Helper()

	var actualRequired, actualOptional, actualComputed bool
	switch attribute := attribute.(type) {
	case schema.StringAttribute:
		actualRequired, actualOptional, actualComputed = attribute.Required, attribute.Optional, attribute.Computed
	case schema.BoolAttribute:
		actualRequired, actualOptional, actualComputed = attribute.Required, attribute.Optional, attribute.Computed
	case schema.Int32Attribute:
		actualRequired, actualOptional, actualComputed = attribute.Required, attribute.Optional, attribute.Computed
	case schema.Int64Attribute:
		actualRequired, actualOptional, actualComputed = attribute.Required, attribute.Optional, attribute.Computed
	case schema.ListAttribute:
		actualRequired, actualOptional, actualComputed = attribute.Required, attribute.Optional, attribute.Computed
	case schema.ListNestedAttribute:
		actualRequired, actualOptional, actualComputed = attribute.Required, attribute.Optional, attribute.Computed
	default:
		require.FailNow(t, "unsupported schema attribute type", "%T", attribute)
	}

	assert.Equal(t, required, actualRequired)
	assert.Equal(t, optional, actualOptional)
	assert.Equal(t, computed, actualComputed)
}
