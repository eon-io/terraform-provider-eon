package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAwsSchemaBlock_AttributesAreOptional verifies that all AWS schema block attributes
// are marked as Optional, not Required. This ensures the AWS block can be omitted when
// using a different cloud provider (Azure or GCP).
func TestAwsSchemaBlock_AttributesAreOptional(t *testing.T) {
	t.Parallel()

	block := awsSchemaBlock()

	// Verify role_arn is Optional
	roleArnAttr, ok := block.Attributes["role_arn"]
	require.True(t, ok, "role_arn attribute should exist in AWS schema block")

	stringAttr, ok := roleArnAttr.(schema.StringAttribute)
	require.True(t, ok, "role_arn should be a StringAttribute")

	assert.True(t, stringAttr.Optional, "role_arn should be Optional")
	assert.False(t, stringAttr.Required, "role_arn should NOT be Required")
}

// TestAzureSchemaBlock_AttributesAreOptional verifies that all Azure schema block attributes
// are marked as Optional, not Required. This ensures the Azure block can be omitted when
// using a different cloud provider (AWS or GCP).
func TestAzureSchemaBlock_AttributesAreOptional(t *testing.T) {
	t.Parallel()

	block := azureSchemaBlock("Test resource group description")

	expectedAttrs := []string{"tenant_id", "subscription_id", "resource_group_name"}

	for _, attrName := range expectedAttrs {
		t.Run(attrName, func(t *testing.T) {
			attr, ok := block.Attributes[attrName]
			require.True(t, ok, "%s attribute should exist in Azure schema block", attrName)

			stringAttr, ok := attr.(schema.StringAttribute)
			require.True(t, ok, "%s should be a StringAttribute", attrName)

			assert.True(t, stringAttr.Optional, "%s should be Optional", attrName)
			assert.False(t, stringAttr.Required, "%s should NOT be Required", attrName)
		})
	}
}

// TestGcpSchemaBlock_AttributesAreOptional verifies that all GCP schema block attributes
// are marked as Optional, not Required. This ensures the GCP block can be omitted when
// using a different cloud provider (AWS or Azure).
//
// This test was added to prevent regression of a bug where project_id and service_account
// were incorrectly marked as Required: true, which caused Terraform validation errors
// when using AWS or Azure cloud providers.
func TestGcpSchemaBlock_AttributesAreOptional(t *testing.T) {
	t.Parallel()

	block := gcpSchemaBlock()

	expectedAttrs := []string{"project_id", "service_account"}

	for _, attrName := range expectedAttrs {
		t.Run(attrName, func(t *testing.T) {
			attr, ok := block.Attributes[attrName]
			require.True(t, ok, "%s attribute should exist in GCP schema block", attrName)

			stringAttr, ok := attr.(schema.StringAttribute)
			require.True(t, ok, "%s should be a StringAttribute", attrName)

			assert.True(t, stringAttr.Optional, "%s should be Optional (not Required) to allow using other cloud providers", attrName)
			assert.False(t, stringAttr.Required, "%s should NOT be Required - this would break Azure and AWS configurations", attrName)
		})
	}
}

// TestAllCloudProviderSchemaBlocks_ConsistentOptionalPattern verifies that all cloud provider
// schema blocks follow the same pattern: all attributes are Optional.
// This ensures consistency across AWS, Azure, and GCP configurations.
func TestAllCloudProviderSchemaBlocks_ConsistentOptionalPattern(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		block      schema.SingleNestedBlock
		attributes []string
	}{
		{
			name:       "AWS",
			block:      awsSchemaBlock(),
			attributes: []string{"role_arn"},
		},
		{
			name:       "Azure",
			block:      azureSchemaBlock("test description"),
			attributes: []string{"tenant_id", "subscription_id", "resource_group_name"},
		},
		{
			name:       "GCP",
			block:      gcpSchemaBlock(),
			attributes: []string{"project_id", "service_account"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, attrName := range tc.attributes {
				attr, ok := tc.block.Attributes[attrName]
				require.True(t, ok, "%s.%s attribute should exist", tc.name, attrName)

				stringAttr, ok := attr.(schema.StringAttribute)
				require.True(t, ok, "%s.%s should be a StringAttribute", tc.name, attrName)

				assert.True(t, stringAttr.Optional,
					"%s.%s should be Optional to allow other cloud providers to be used", tc.name, attrName)
				assert.False(t, stringAttr.Required,
					"%s.%s should NOT be Required - would break other cloud provider configurations", tc.name, attrName)
			}
		})
	}
}

// TestCloudProvider_BlockName verifies that CloudProvider returns correct block names.
func TestCloudProvider_BlockName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		provider CloudProvider
		expected string
	}{
		{CloudProviderAWS, "aws"},
		{CloudProviderAzure, "azure"},
		{CloudProviderGCP, "gcp"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.provider.BlockName())
		})
	}
}

// TestCloudProvider_String verifies that CloudProvider String() returns the correct value.
func TestCloudProvider_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		provider CloudProvider
		expected string
	}{
		{CloudProviderAWS, "AWS"},
		{CloudProviderAzure, "AZURE"},
		{CloudProviderGCP, "GCP"},
	}

	for _, tt := range tests {
		t.Run(string(tt.provider), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.provider.String())
		})
	}
}

// TestConflictErrorMessage verifies the conflict error message generation.
func TestConflictErrorMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		resourceType string
		existingID   string
		wantTitle    string
		wantContains []string
	}{
		{
			name:         "source account with ID",
			resourceType: "Source Account",
			existingID:   "acc-123",
			wantTitle:    "Source Account Already Exists",
			wantContains: []string{"terraform import", "eon_source_account", "acc-123"},
		},
		{
			name:         "restore account with ID",
			resourceType: "Restore Account",
			existingID:   "acc-456",
			wantTitle:    "Restore Account Already Exists",
			wantContains: []string{"terraform import", "eon_restore_account", "acc-456"},
		},
		{
			name:         "source account without ID",
			resourceType: "Source Account",
			existingID:   "",
			wantTitle:    "Source Account Already Exists",
			wantContains: []string{"terraform import", "eon_source_account", "data source"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, detail := conflictErrorMessage(tt.resourceType, tt.existingID)

			assert.Equal(t, tt.wantTitle, title)
			for _, substring := range tt.wantContains {
				assert.Contains(t, detail, substring)
			}
		})
	}
}

// TestResourceTypeTFName verifies the resource type to TF name conversion.
func TestResourceTypeTFName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"Source Account", "source_account"},
		{"Restore Account", "restore_account"},
		{"Unknown", "account"},
		{"", "account"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := resourceTypeTFName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
