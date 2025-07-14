package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEonProvider_Metadata tests the provider metadata
func TestEonProvider_Metadata(t *testing.T) {
	p := &EonProvider{version: "1.0.0"}

	req := provider.MetadataRequest{}
	resp := &provider.MetadataResponse{}

	p.Metadata(context.Background(), req, resp)

	assert.Equal(t, "eon", resp.TypeName)
	assert.Equal(t, "1.0.0", resp.Version)
}

// TestEonProvider_Schema tests the provider schema
func TestEonProvider_Schema(t *testing.T) {
	p := &EonProvider{}

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}

	p.Schema(context.Background(), req, resp)

	require.NotNil(t, resp.Schema)
	assert.False(t, resp.Diagnostics.HasError())

	// Test required attributes
	assert.Contains(t, resp.Schema.Attributes, "endpoint")
	assert.Contains(t, resp.Schema.Attributes, "client_id")
	assert.Contains(t, resp.Schema.Attributes, "client_secret")
	assert.Contains(t, resp.Schema.Attributes, "project_id")
}

// TestEonProvider_Resources tests the provider resources registration
func TestEonProvider_Resources(t *testing.T) {
	p := &EonProvider{}

	resources := p.Resources(context.Background())

	assert.NotEmpty(t, resources)
	assert.Greater(t, len(resources), 0)
}

// TestEonProvider_DataSources tests the provider data sources registration
func TestEonProvider_DataSources(t *testing.T) {
	p := &EonProvider{}

	dataSources := p.DataSources(context.Background())

	assert.NotEmpty(t, dataSources)
	assert.Greater(t, len(dataSources), 0)
}

// TestNew tests the provider factory function
func TestNew(t *testing.T) {
	version := "test-version"

	providerFunc := New(version)
	provider := providerFunc()

	assert.NotNil(t, provider)
	assert.IsType(t, &EonProvider{}, provider)

	eonProvider := provider.(*EonProvider)
	assert.Equal(t, version, eonProvider.version)
}

// TestEonProviderModel tests the provider model structure
func TestEonProviderModel(t *testing.T) {
	model := EonProviderModel{
		Endpoint:     types.StringValue("https://test.eon.io"),
		ClientId:     types.StringValue("test-client-id"),
		ClientSecret: types.StringValue("test-client-secret"),
		ProjectId:    types.StringValue("test-project-id"),
	}

	assert.Equal(t, "https://test.eon.io", model.Endpoint.ValueString())
	assert.Equal(t, "test-client-id", model.ClientId.ValueString())
	assert.Equal(t, "test-client-secret", model.ClientSecret.ValueString())
	assert.Equal(t, "test-project-id", model.ProjectId.ValueString())
}

// TestEonProvider_StringValues tests string value handling
func TestEonProvider_StringValues(t *testing.T) {
	testCases := []struct {
		name     string
		value    string
		expected string
	}{
		{
			name:     "normal_string",
			value:    "test-value",
			expected: "test-value",
		},
		{
			name:     "empty_string",
			value:    "",
			expected: "",
		},
		{
			name:     "url_string",
			value:    "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "uuid_string",
			value:    "123e4567-e89b-12d3-a456-426614174000",
			expected: "123e4567-e89b-12d3-a456-426614174000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stringValue := types.StringValue(tc.value)
			assert.Equal(t, tc.expected, stringValue.ValueString())
		})
	}
}

// TestEonProvider_BoolValues tests boolean value handling
func TestEonProvider_BoolValues(t *testing.T) {
	testCases := []struct {
		name     string
		value    bool
		expected bool
	}{
		{
			name:     "true_value",
			value:    true,
			expected: true,
		},
		{
			name:     "false_value",
			value:    false,
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			boolValue := types.BoolValue(tc.value)
			assert.Equal(t, tc.expected, boolValue.ValueBool())
		})
	}
}

// TestEonProvider_Int64Values tests int64 value handling
func TestEonProvider_Int64Values(t *testing.T) {
	testCases := []struct {
		name     string
		value    int64
		expected int64
	}{
		{
			name:     "positive_int",
			value:    42,
			expected: 42,
		},
		{
			name:     "negative_int",
			value:    -42,
			expected: -42,
		},
		{
			name:     "zero",
			value:    0,
			expected: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			int64Value := types.Int64Value(tc.value)
			assert.Equal(t, tc.expected, int64Value.ValueInt64())
		})
	}
}

// TestEonProvider_TypeInterface tests that provider implements the Provider interface
func TestEonProvider_TypeInterface(t *testing.T) {
	p := &EonProvider{}

	// Test that provider implements the Provider interface
	var _ provider.Provider = p

	// This test passes if compilation succeeds
	assert.NotNil(t, p)
}

// TestEonProvider_ProviderSchema tests detailed schema attributes
func TestEonProvider_ProviderSchema(t *testing.T) {
	p := &EonProvider{}

	req := provider.SchemaRequest{}
	resp := &provider.SchemaResponse{}

	p.Schema(context.Background(), req, resp)

	require.NotNil(t, resp.Schema)
	assert.False(t, resp.Diagnostics.HasError())

	// Test that we have exactly 4 attributes
	assert.Equal(t, 4, len(resp.Schema.Attributes))

	// Test attribute names
	expectedAttributes := []string{"endpoint", "client_id", "client_secret", "project_id"}
	for _, attr := range expectedAttributes {
		assert.Contains(t, resp.Schema.Attributes, attr)
	}
}

// TestEonProvider_ProviderMetadata tests detailed metadata
func TestEonProvider_ProviderMetadata(t *testing.T) {
	testCases := []struct {
		name            string
		version         string
		expectedType    string
		expectedVersion string
	}{
		{
			name:            "version_1.0.0",
			version:         "1.0.0",
			expectedType:    "eon",
			expectedVersion: "1.0.0",
		},
		{
			name:            "version_dev",
			version:         "dev",
			expectedType:    "eon",
			expectedVersion: "dev",
		},
		{
			name:            "version_test",
			version:         "test",
			expectedType:    "eon",
			expectedVersion: "test",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := &EonProvider{version: tc.version}

			req := provider.MetadataRequest{}
			resp := &provider.MetadataResponse{}

			p.Metadata(context.Background(), req, resp)

			assert.Equal(t, tc.expectedType, resp.TypeName)
			assert.Equal(t, tc.expectedVersion, resp.Version)
		})
	}
}
