package client

// ClientConfig holds client configuration
type ClientConfig struct {
	Endpoint       string
	ClientID       string
	ClientSecret   string // #nosec G117 -- required for provider API authentication // #nosec G117 -- required for provider API authentication
	ProjectID      string
	DefaultHeaders map[string]string // Optional additional headers
	// ActionApprovalRequestID is the ID of an approved action approval request.
	// When set, it is sent in the X-Action-Approval-Request-Id header on every
	// operation that an action approval policy can protect.
	ActionApprovalRequestID string
}

// ClientFactory creates an EonClient from configuration
type ClientFactory func(cfg ClientConfig) (*EonClient, error)
