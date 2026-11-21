package provider

import (
	"context"
	"testing"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configureProvider runs Configure with the given action_approval_request_id
// (nil leaves it unset) and returns the client config passed to the factory.
func configureProvider(t *testing.T, actionApprovalRequestID *string) client.ClientConfig {
	t.Helper()

	var got client.ClientConfig
	p := New("test", func(cfg client.ClientConfig) (*client.EonClient, error) {
		got = cfg
		return nil, nil
	})()

	ctx := context.Background()
	schemaResp := &provider.SchemaResponse{}
	p.Schema(ctx, provider.SchemaRequest{}, schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError())

	schemaType := schemaResp.Schema.Type().TerraformType(ctx)
	raw := tftypes.NewValue(schemaType, map[string]tftypes.Value{
		"endpoint":                   tftypes.NewValue(tftypes.String, "https://example.console.eon.io"),
		"client_id":                  tftypes.NewValue(tftypes.String, "client-id"),
		"client_secret":              tftypes.NewValue(tftypes.String, "client-secret"),
		"project_id":                 tftypes.NewValue(tftypes.String, "project-1"),
		"action_approval_request_id": tftypes.NewValue(tftypes.String, actionApprovalRequestID),
	})

	resp := &provider.ConfigureResponse{}
	p.Configure(ctx, provider.ConfigureRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: raw}}, resp)
	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	return got
}

func TestProviderConfigure_ActionApprovalRequestID(t *testing.T) {
	fromConfig := "config-id"

	t.Run("unset", func(t *testing.T) {
		assert.Empty(t, configureProvider(t, nil).ActionApprovalRequestID)
	})

	t.Run("from provider block", func(t *testing.T) {
		assert.Equal(t, fromConfig, configureProvider(t, &fromConfig).ActionApprovalRequestID)
	})

	t.Run("environment variable is ignored", func(t *testing.T) {
		t.Setenv("EON_ACTION_APPROVAL_REQUEST_ID", "env-id")
		assert.Empty(t, configureProvider(t, nil).ActionApprovalRequestID)
	})
}
