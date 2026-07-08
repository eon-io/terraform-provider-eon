package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func awsModel(roleArn string) *AwsAccountConfigModel {
	return &AwsAccountConfigModel{RoleArn: types.StringValue(roleArn)}
}

func gcpModel(projectID, serviceAccount string) *GcpAccountConfigModel {
	return &GcpAccountConfigModel{
		ProjectId:      types.StringValue(projectID),
		ServiceAccount: types.StringValue(serviceAccount),
	}
}

func azureModel(tenant, sub, rg string) *AzureAccountConfigModel {
	return &AzureAccountConfigModel{
		TenantId:          types.StringValue(tenant),
		SubscriptionId:    types.StringValue(sub),
		ResourceGroupName: types.StringValue(rg),
	}
}

func TestRestoreAccount_azureAttributesChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		plan, state *AzureAccountConfigModel
		want        bool
	}{
		{"both nil", nil, nil, false},
		{"plan nil", nil, azureModel("t", "s", "rg"), false},
		{"state nil", azureModel("t", "s", "rg"), nil, false},
		{"identical", azureModel("t", "s", "rg"), azureModel("t", "s", "rg"), false},
		{"tenant changed", azureModel("t2", "s", "rg"), azureModel("t", "s", "rg"), true},
		{"subscription changed", azureModel("t", "s2", "rg"), azureModel("t", "s", "rg"), true},
		{"resource group changed", azureModel("t", "s", "rg2"), azureModel("t", "s", "rg"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, azureAttributesChanged(tt.plan, tt.state))
		})
	}
}

func TestRestoreAccount_buildUpdateRequest(t *testing.T) {
	t.Parallel()
	r := &RestoreAccountResource{}

	t.Run("no changes returns nil", func(t *testing.T) {
		t.Parallel()
		plan := RestoreAccountResourceModel{Name: types.StringValue("acct"), Aws: awsModel("arn:x")}
		state := RestoreAccountResourceModel{Name: types.StringValue("acct"), Aws: awsModel("arn:x")}
		assert.Nil(t, r.buildUpdateRequest(plan, state))
	})

	t.Run("name change", func(t *testing.T) {
		t.Parallel()
		plan := RestoreAccountResourceModel{Name: types.StringValue("new")}
		state := RestoreAccountResourceModel{Name: types.StringValue("old")}
		req := r.buildUpdateRequest(plan, state)
		require.NotNil(t, req)
		require.NotNil(t, req.Name)
		assert.Equal(t, "new", *req.Name)
		assert.Nil(t, req.RestoreAccountAttributes)
	})

	t.Run("aws role_arn change", func(t *testing.T) {
		t.Parallel()
		plan := RestoreAccountResourceModel{Name: types.StringValue("acct"), Aws: awsModel("arn:new")}
		state := RestoreAccountResourceModel{Name: types.StringValue("acct"), Aws: awsModel("arn:old")}
		req := r.buildUpdateRequest(plan, state)
		require.NotNil(t, req)
		require.NotNil(t, req.RestoreAccountAttributes)
		require.NotNil(t, req.RestoreAccountAttributes.Aws)
		assert.Equal(t, "arn:new", *req.RestoreAccountAttributes.Aws.RoleArn)
	})

	t.Run("gcp service_account change", func(t *testing.T) {
		t.Parallel()
		plan := RestoreAccountResourceModel{Name: types.StringValue("acct"), Gcp: gcpModel("p", "sa-new@x")}
		state := RestoreAccountResourceModel{Name: types.StringValue("acct"), Gcp: gcpModel("p", "sa-old@x")}
		req := r.buildUpdateRequest(plan, state)
		require.NotNil(t, req)
		require.NotNil(t, req.RestoreAccountAttributes)
		require.NotNil(t, req.RestoreAccountAttributes.Gcp)
		assert.Equal(t, "sa-new@x", *req.RestoreAccountAttributes.Gcp.ServiceAccount)
	})
}
