package provider

import (
	"context"
	"fmt"
	"testing"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdpsDataSource_Unit(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, NewIdpsDataSource())
}

func TestIdpsDataSource_ListWithMockClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		shouldFail  bool
		numIdps     int
		expectError bool
	}{
		{name: "successful list with multiple idps", shouldFail: false, numIdps: 2, expectError: false},
		{name: "successful list with no idps", shouldFail: false, numIdps: 0, expectError: false},
		{name: "list failure", shouldFail: true, numIdps: 0, expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := client.NewMockEonClient()
			mockClient.ShouldFailIdpList = tt.shouldFail

			for i := 0; i < tt.numIdps; i++ {
				mockClient.AddMockIdp(&externalEonSdkAPI.Idp{
					Id:           fmt.Sprintf("idp-%d", i+1),
					ProviderName: fmt.Sprintf("Provider %d", i+1),
				})
			}

			result, err := mockClient.ListIdps(context.Background())
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Len(t, result, tt.numIdps)
			}
			assert.Equal(t, 1, mockClient.IdpListCalls)
		})
	}
}

func TestIdpsDataSource_ListWithProviderName(t *testing.T) {
	t.Parallel()

	mockClient := client.NewMockEonClient()
	mockClient.AddMockIdp(&externalEonSdkAPI.Idp{
		Id:           "idp-okta",
		ProviderName: "Okta",
	})

	result, err := mockClient.ListIdps(context.Background())
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "idp-okta", result[0].Id)
	assert.Equal(t, "Okta", result[0].ProviderName)
}
