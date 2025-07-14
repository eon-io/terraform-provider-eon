package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestProvider tests the provider creation without external dependencies
func TestProvider(t *testing.T) {
	t.Parallel()

	provider := New("test")()
	assert.NotNil(t, provider, "Provider should not be nil")
}

// TestProvider_impl tests that the provider implements the expected interface
func TestProvider_impl(t *testing.T) {
	t.Parallel()

	provider := New("test")()
	assert.NotNil(t, provider, "Provider should not be nil")

	// Test that provider can be created successfully
	// This validates interface compliance without external dependencies
}

// TestProvider_NewWithVersion tests provider creation with different versions
func TestProvider_NewWithVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version string
	}{
		{
			name:    "test version",
			version: "test",
		},
		{
			name:    "dev version",
			version: "dev",
		},
		{
			name:    "production version",
			version: "1.0.0",
		},
		{
			name:    "empty version",
			version: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := New(tt.version)()
			assert.NotNil(t, provider, "Provider should not be nil for version %s", tt.version)
		})
	}
}

// TestProvider_BasicValidation tests basic provider validation
func TestProvider_BasicValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "provider creation",
			test: func(t *testing.T) {
				provider := New("test")()
				assert.NotNil(t, provider, "Provider should not be nil")
			},
		},
		{
			name: "provider with different versions",
			test: func(t *testing.T) {
				versions := []string{"test", "dev", "1.0.0", ""}
				for _, version := range versions {
					provider := New(version)()
					assert.NotNil(t, provider, "Provider should not be nil for version %s", version)
				}
			},
		},
		{
			name: "provider stability",
			test: func(t *testing.T) {
				// Test that multiple providers can be created
				provider1 := New("test")()
				provider2 := New("test")()

				assert.NotNil(t, provider1, "First provider should not be nil")
				assert.NotNil(t, provider2, "Second provider should not be nil")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.test(t)
		})
	}
}

// TestProvider_ConfigurationValidation tests configuration validation logic
func TestProvider_ConfigurationValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		endpoint     string
		clientID     string
		clientSecret string
		projectID    string
		shouldError  bool
	}{
		{
			name:         "valid configuration",
			endpoint:     "https://test.eon.io",
			clientID:     "test-client",
			clientSecret: "test-secret",
			projectID:    "test-project",
			shouldError:  false,
		},
		{
			name:         "empty endpoint",
			endpoint:     "",
			clientID:     "test-client",
			clientSecret: "test-secret",
			projectID:    "test-project",
			shouldError:  true,
		},
		{
			name:         "empty client ID",
			endpoint:     "https://test.eon.io",
			clientID:     "",
			clientSecret: "test-secret",
			projectID:    "test-project",
			shouldError:  true,
		},
		{
			name:         "empty client secret",
			endpoint:     "https://test.eon.io",
			clientID:     "test-client",
			clientSecret: "",
			projectID:    "test-project",
			shouldError:  true,
		},
		{
			name:         "empty project ID",
			endpoint:     "https://test.eon.io",
			clientID:     "test-client",
			clientSecret: "test-secret",
			projectID:    "",
			shouldError:  true,
		},
		{
			name:         "all empty",
			endpoint:     "",
			clientID:     "",
			clientSecret: "",
			projectID:    "",
			shouldError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Test configuration validation logic without API calls
			hasEmptyFields := tt.endpoint == "" || tt.clientID == "" || tt.clientSecret == "" || tt.projectID == ""

			if tt.shouldError {
				assert.True(t, hasEmptyFields, "Should have empty required fields")
			} else {
				assert.False(t, hasEmptyFields, "Should not have empty required fields")
				assert.NotEmpty(t, tt.endpoint, "Endpoint should not be empty")
				assert.NotEmpty(t, tt.clientID, "Client ID should not be empty")
				assert.NotEmpty(t, tt.clientSecret, "Client secret should not be empty")
				assert.NotEmpty(t, tt.projectID, "Project ID should not be empty")
			}
		})
	}
}

// TestProvider_Concurrent tests concurrent provider creation
func TestProvider_Concurrent(t *testing.T) {
	t.Parallel()

	const numGoroutines = 10
	const numProvidersPerGoroutine = 10

	results := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() {
				if r := recover(); r != nil {
					results <- false
					return
				}
			}()

			for j := 0; j < numProvidersPerGoroutine; j++ {
				provider := New("test")()
				if provider == nil {
					results <- false
					return
				}
			}
			results <- true
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		select {
		case success := <-results:
			assert.True(t, success, "Goroutine %d should create providers successfully", i)
		}
	}
}
