package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToSdkUpdateSourceAccountAttributes(t *testing.T) {
	t.Parallel()

	roleArn := "arn:aws:iam::123456789012:role/eon"
	tests := []struct {
		name     string
		in       UpdateSourceAccountAttributes
		wantJSON string
	}{
		{
			name: "nothing to update",
			in:   UpdateSourceAccountAttributes{},
		},
		{
			name:     "aws role only leaves regions out",
			in:       UpdateSourceAccountAttributes{Aws: &UpdateAwsSourceAccountAttributes{RoleArn: &roleArn}},
			wantJSON: `{"aws":{"roleArn":"arn:aws:iam::123456789012:role/eon"}}`,
		},
		{
			name:     "aws regions",
			in:       UpdateSourceAccountAttributes{Aws: &UpdateAwsSourceAccountAttributes{Regions: []string{"eu-west-1", "us-east-1"}}},
			wantJSON: `{"aws":{"regions":["eu-west-1","us-east-1"]}}`,
		},
		{
			name:     "empty azure regions are sent so the account goes back to every region",
			in:       UpdateSourceAccountAttributes{Azure: &UpdateAzureSourceAccountAttributes{Regions: []string{}}},
			wantJSON: `{"azure":{"regions":[]}}`,
		},
		{
			name:     "gcp regions",
			in:       UpdateSourceAccountAttributes{Gcp: &UpdateGcpSourceAccountAttributes{Regions: []string{"us-central1"}}},
			wantJSON: `{"gcp":{"regions":["us-central1"]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			attrs := toSdkUpdateSourceAccountAttributes(tt.in)
			if tt.wantJSON == "" {
				assert.Nil(t, attrs)
				return
			}
			require.NotNil(t, attrs)
			got, err := json.Marshal(attrs)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantJSON, string(got))
		})
	}
}
