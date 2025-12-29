package client

// ClientConfig holds client configuration
type ClientConfig struct {
	Endpoint       string
	ClientID       string
	ClientSecret   string
	ProjectID      string
	DefaultHeaders map[string]string // Optional additional headers
}

// ClientFactory creates an EonClient from configuration
type ClientFactory func(cfg ClientConfig) (*EonClient, error)
