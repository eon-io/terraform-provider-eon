package provider

import (
	"context"
	"testing"

	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func regionSet(regions ...string) types.Set {
	return regionsToSet(regions)
}

var (
	nullRegions    = types.SetNull(types.StringType)
	unknownRegions = types.SetUnknown(types.StringType)
)

func TestRegionsFromSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		set        types.Set
		want       []string
		wantConfig bool
	}{
		{name: "null is not managed", set: nullRegions},
		{name: "unknown is not managed", set: unknownRegions},
		{name: "empty means every region", set: regionSet(), want: []string{}, wantConfig: true},
		{name: "regions come back sorted", set: regionSet("us-east-1", "eu-west-1"), want: []string{"eu-west-1", "us-east-1"}, wantConfig: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, configured, diags := regionsFromSet(context.Background(), tt.set)
			require.False(t, diags.HasError())
			assert.Equal(t, tt.wantConfig, configured)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRegionsToSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		regions []string
		want    types.Set
	}{
		{name: "no regions is a known empty set", regions: nil, want: types.SetValueMust(types.StringType, []attr.Value{})},
		{name: "regions", regions: []string{"eastus"}, want: types.SetValueMust(types.StringType, []attr.Value{types.StringValue("eastus")})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := regionsToSet(tt.regions)
			assert.False(t, got.IsNull())
			assert.True(t, tt.want.Equal(got))
		})
	}
}

func TestRegionsChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		plan, state types.Set
		want        bool
	}{
		{name: "omitted from config", plan: nullRegions, state: regionSet("us-east-1"), want: false},
		{name: "unknown plan", plan: unknownRegions, state: regionSet("us-east-1"), want: false},
		{name: "same regions in another order", plan: regionSet("b", "a"), state: regionSet("a", "b"), want: false},
		{name: "region added", plan: regionSet("a", "b"), state: regionSet("a"), want: true},
		{name: "cleared to every region", plan: regionSet(), state: regionSet("a"), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, regionsChanged(tt.plan, tt.state))
		})
	}
}

func TestImmutableRegionsModifier(t *testing.T) {
	t.Parallel()

	existingResource := tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})}
	newResource := tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{}, nil)}
	plannedResource := tfsdk.Plan{Raw: tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})}

	tests := []struct {
		name        string
		state       tfsdk.State
		planValue   types.Set
		stateValue  types.Set
		wantErr     bool
		wantErrText string
	}{
		{name: "set when connecting", state: newResource, planValue: regionSet("us-east-1"), stateValue: nullRegions},
		{name: "unchanged", state: existingResource, planValue: regionSet("us-east-1"), stateValue: regionSet("us-east-1")},
		{name: "omitted keeps the current regions", state: existingResource, planValue: nullRegions, stateValue: regionSet("us-east-1")},
		{
			name: "changed on a connected organizational unit", state: existingResource,
			planValue: regionSet("eu-west-1"), stateValue: regionSet("us-east-1", "ap-south-1"),
			wantErr: true, wantErrText: "[ap-south-1, us-east-1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := planmodifier.SetRequest{State: tt.state, Plan: plannedResource, PlanValue: tt.planValue, StateValue: tt.stateValue}
			resp := &planmodifier.SetResponse{PlanValue: tt.planValue}
			immutableRegionsModifier{resourceName: "organizational unit"}.PlanModifySet(context.Background(), req, resp)
			require.Equal(t, tt.wantErr, resp.Diagnostics.HasError())
			if tt.wantErr {
				assert.Contains(t, resp.Diagnostics.Errors()[0].Detail(), tt.wantErrText)
			}
		})
	}
}

func TestBuildSourceAccountUpdateRequest(t *testing.T) {
	t.Parallel()

	account := func(provider CloudProvider, roleArn string, regions types.Set) SourceAccountResourceModel {
		m := SourceAccountResourceModel{Name: types.StringValue("acct"), CloudProvider: types.StringValue(provider.String()), Regions: regions}
		if roleArn != "" {
			m.Aws = awsModel(roleArn)
		}
		return m
	}
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name        string
		plan, state SourceAccountResourceModel
		want        *client.UpdateSourceAccountRequest
	}{
		{
			name:  "no changes",
			plan:  account(CloudProviderAWS, "arn:x", regionSet("us-east-1")),
			state: account(CloudProviderAWS, "arn:x", regionSet("us-east-1")),
		},
		{
			name:  "regions omitted from config are left alone",
			plan:  account(CloudProviderAzure, "", nullRegions),
			state: account(CloudProviderAzure, "", regionSet("eastus")),
		},
		{
			name:  "aws role and regions change together",
			plan:  account(CloudProviderAWS, "arn:new", regionSet("us-east-1", "eu-west-1")),
			state: account(CloudProviderAWS, "arn:old", regionSet("us-east-1")),
			want: &client.UpdateSourceAccountRequest{SourceAccountAttributes: &client.UpdateSourceAccountAttributes{
				Aws: &client.UpdateAwsSourceAccountAttributes{RoleArn: strPtr("arn:new"), Regions: []string{"eu-west-1", "us-east-1"}},
			}},
		},
		{
			name:  "azure regions cleared to every region",
			plan:  account(CloudProviderAzure, "", regionSet()),
			state: account(CloudProviderAzure, "", regionSet("eastus")),
			want: &client.UpdateSourceAccountRequest{SourceAccountAttributes: &client.UpdateSourceAccountAttributes{
				Azure: &client.UpdateAzureSourceAccountAttributes{Regions: []string{}},
			}},
		},
		{
			name:  "gcp regions set",
			plan:  account(CloudProviderGCP, "", regionSet("us-central1")),
			state: account(CloudProviderGCP, "", regionSet()),
			want: &client.UpdateSourceAccountRequest{SourceAccountAttributes: &client.UpdateSourceAccountAttributes{
				Gcp: &client.UpdateGcpSourceAccountAttributes{Regions: []string{"us-central1"}},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, diags := buildSourceAccountUpdateRequest(context.Background(), tt.plan, tt.state)
			require.False(t, diags.HasError())
			assert.Equal(t, tt.want, got)
		})
	}
}
