package client

// ClientConfig holds client configuration
type ClientConfig struct {
	Endpoint       string
	ClientID       string
	ClientSecret   string // #nosec G117 -- required for provider API authentication // #nosec G117 -- required for provider API authentication
	ProjectID      string
	DefaultHeaders map[string]string // Optional additional headers
}

// ClientFactory creates an EonClient from configuration
type ClientFactory func(cfg ClientConfig) (*EonClient, error)
