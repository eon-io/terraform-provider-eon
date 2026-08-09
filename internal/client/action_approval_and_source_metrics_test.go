package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sourceMetricsConfigPath = "/api/v1/projects/project-1/source-accounts/account-1/metrics-config"

const sourceMetricsConfigResponse = `{
	"sourceAccountConfig": {
		"sourceAccountId": "account-1",
		"enabled": true,
		"destination": {
			"aws": {"region": "us-east-1"}
		}
	}
}`

func TestGetSourceAccountMetricsConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantRegion string
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body:       sourceMetricsConfigResponse,
			wantRegion: "us-east-1",
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			body:       `{"message":"not found"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, sourceMetricsConfigPath, r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			config, err := c.GetSourceAccountMetricsConfig(context.Background(), "account-1")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, config)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "account-1", config.GetSourceAccountId())
			assert.True(t, config.GetEnabled())
			dest := config.GetDestination()
			aws := dest.GetAws()
			assert.Equal(t, tt.wantRegion, aws.GetRegion())
		})
	}
}

func TestEnableSourceAccountMetricsConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body:       sourceMetricsConfigResponse,
		},
		{
			name:       "bad request",
			statusCode: http.StatusBadRequest,
			body:       `{"message":"invalid"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, sourceMetricsConfigPath, r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			aws := externalEonSdkAPI.NewAwsAccountMetricsDestination()
			aws.SetRegion("us-east-1")
			req := externalEonSdkAPI.EnableSourceAccountMetricsConfigRequest{}
			req.SetAws(*aws)

			config, err := c.EnableSourceAccountMetricsConfig(context.Background(), "account-1", req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, config)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "account-1", config.GetSourceAccountId())
		})
	}
}

func TestDisableSourceAccountMetricsConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusNoContent},
		{name: "success ok", statusCode: http.StatusOK},
		{name: "not found", statusCode: http.StatusNotFound, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, sourceMetricsConfigPath, r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}, nil
			}))

			err := c.DisableSourceAccountMetricsConfig(context.Background(), "account-1")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

const actionApprovalRuleResponse = `{
	"actionApprovalRule": {
		"id": "rule-1",
		"projectId": "project-1",
		"operation": "RESTORE_RESOURCE",
		"requiredApprovals": 1,
		"approvalWindowHours": 24,
		"executionWindowHours": 48,
		"description": "test",
		"exemptApiCredentials": false
	}
}`

func TestCreateActionApprovalRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK, body: actionApprovalRuleResponse},
		{name: "bad request", statusCode: http.StatusBadRequest, body: `{"message":"invalid"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/projects/project-1/action-approvals/rules", r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			req := *externalEonSdkAPI.NewCreateActionApprovalRuleRequest(
				externalEonSdkAPI.ACTION_APPROVAL_OPERATION_RESTORE_RESOURCE,
				24,
				48,
			)
			rule, err := c.CreateActionApprovalRule(context.Background(), req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, rule)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "rule-1", rule.GetId())
		})
	}
}

func TestGetActionApprovalRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK, body: actionApprovalRuleResponse},
		{name: "not found", statusCode: http.StatusNotFound, body: `{"message":"not found"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/v1/projects/project-1/action-approvals/rules/rule-1", r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			rule, err := c.GetActionApprovalRule(context.Background(), "rule-1")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, rule)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "rule-1", rule.GetId())
			assert.Equal(t, externalEonSdkAPI.ACTION_APPROVAL_OPERATION_RESTORE_RESOURCE, rule.GetOperation())
		})
	}
}

func TestUpdateActionApprovalRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusOK, body: actionApprovalRuleResponse},
		{name: "bad request", statusCode: http.StatusBadRequest, body: `{"message":"invalid"}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, "/api/v1/projects/project-1/action-approvals/rules/rule-1", r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			req := *externalEonSdkAPI.NewUpdateActionApprovalRuleRequest()
			req.SetRequiredApprovals(2)
			rule, err := c.UpdateActionApprovalRule(context.Background(), "rule-1", req)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, rule)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "rule-1", rule.GetId())
		})
	}
}

func TestDeleteActionApprovalRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{name: "success", statusCode: http.StatusNoContent},
		{name: "success ok", statusCode: http.StatusOK},
		{name: "not found", statusCode: http.StatusNotFound, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/api/v1/projects/project-1/action-approvals/rules/rule-1", r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{}`)),
				}, nil
			}))

			err := c.DeleteActionApprovalRule(context.Background(), "rule-1")
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestListActionApprovalRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    bool
		wantLen    int
	}{
		{
			name:       "success",
			statusCode: http.StatusOK,
			body: `{
				"actionApprovalRules": [{
					"id": "rule-1",
					"projectId": "project-1",
					"operation": "RESTORE_RESOURCE",
					"requiredApprovals": 1,
					"approvalWindowHours": 24,
					"executionWindowHours": 48
				}],
				"totalCount": 1
			}`,
			wantLen: 1,
		},
		{
			name:       "empty",
			statusCode: http.StatusOK,
			body:       `{"actionApprovalRules": [], "totalCount": 0}`,
			wantLen:    0,
		},
		{
			name:       "error",
			statusCode: http.StatusInternalServerError,
			body:       `{"message":"boom"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c := newTestClient(roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/v1/projects/project-1/action-approvals/rules", r.URL.Path)
				return &http.Response{
					StatusCode: tt.statusCode,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(tt.body)),
				}, nil
			}))

			rules, err := c.ListActionApprovalRules(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, rules)
				return
			}
			require.NoError(t, err)
			assert.Len(t, rules, tt.wantLen)
		})
	}
}
