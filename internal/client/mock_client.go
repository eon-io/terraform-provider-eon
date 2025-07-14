package client

import (
	"context"
	"fmt"
	"sort"
	"sync"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
)

// MockEonClient implements the EonClient interface with mock data
type MockEonClient struct {
	// Mutex for thread safety
	mu sync.RWMutex

	// Storage for mock data
	BackupPolicies map[string]*externalEonSdkAPI.BackupPolicy

	// Behavior controls
	ShouldFailCreate bool
	ShouldFailRead   bool
	ShouldFailUpdate bool
	ShouldFailDelete bool
	ShouldFailList   bool

	// Call tracking
	CreateCalls int
	ReadCalls   int
	UpdateCalls int
	DeleteCalls int
	ListCalls   int

	// Mock configuration
	ProjectID string
}

// NewMockEonClient creates a new mock client with default behavior
func NewMockEonClient() *MockEonClient {
	return &MockEonClient{
		BackupPolicies: make(map[string]*externalEonSdkAPI.BackupPolicy),
		ProjectID:      "mock-project-id",
	}
}

// CreateBackupPolicy mocks creating a backup policy
func (m *MockEonClient) CreateBackupPolicy(ctx context.Context, req externalEonSdkAPI.CreateBackupPolicyRequest) (*externalEonSdkAPI.BackupPolicy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CreateCalls++

	if m.ShouldFailCreate {
		return nil, fmt.Errorf("mock create error")
	}

	// Generate mock ID
	id := fmt.Sprintf("mock-policy-%d", m.CreateCalls)

	// Create mock policy with only the fields that exist in the actual EON SDK
	policy := &externalEonSdkAPI.BackupPolicy{
		Id:      id,
		Name:    req.Name,
		Enabled: req.GetEnabled(),
	}

	// Store in mock storage
	m.BackupPolicies[id] = policy

	return policy, nil
}

// ReadBackupPolicy mocks reading a backup policy
func (m *MockEonClient) ReadBackupPolicy(ctx context.Context, id string) (*externalEonSdkAPI.BackupPolicy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ReadCalls++

	if m.ShouldFailRead {
		return nil, fmt.Errorf("mock read error")
	}

	policy, exists := m.BackupPolicies[id]
	if !exists {
		return nil, fmt.Errorf("backup policy not found: %s", id)
	}

	return policy, nil
}

// UpdateBackupPolicy mocks updating a backup policy
func (m *MockEonClient) UpdateBackupPolicy(ctx context.Context, id string, req externalEonSdkAPI.UpdateBackupPolicyRequest) (*externalEonSdkAPI.BackupPolicy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.UpdateCalls++

	if m.ShouldFailUpdate {
		return nil, fmt.Errorf("mock update error")
	}

	policy, exists := m.BackupPolicies[id]
	if !exists {
		return nil, fmt.Errorf("backup policy not found: %s", id)
	}

	// Update the policy with the correct field access
	policy.Name = req.Name
	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}

	// Store updated policy
	m.BackupPolicies[id] = policy

	return policy, nil
}

// DeleteBackupPolicy mocks deleting a backup policy
func (m *MockEonClient) DeleteBackupPolicy(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.DeleteCalls++

	if m.ShouldFailDelete {
		return fmt.Errorf("mock delete error")
	}

	_, exists := m.BackupPolicies[id]
	if !exists {
		return fmt.Errorf("backup policy not found: %s", id)
	}

	delete(m.BackupPolicies, id)
	return nil
}

// ListBackupPolicies mocks listing backup policies
func (m *MockEonClient) ListBackupPolicies(ctx context.Context) ([]externalEonSdkAPI.BackupPolicy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ListCalls++

	if m.ShouldFailList {
		return nil, fmt.Errorf("mock list error")
	}

	policies := make([]externalEonSdkAPI.BackupPolicy, 0)
	for _, policy := range m.BackupPolicies {
		policies = append(policies, *policy)
	}

	// Sort policies by ID for consistent ordering
	sort.Slice(policies, func(i, j int) bool {
		return policies[i].Id < policies[j].Id
	})

	return policies, nil
}

// GetBackupPolicy mocks getting a backup policy (alias for ReadBackupPolicy)
func (m *MockEonClient) GetBackupPolicy(ctx context.Context, id string) (*externalEonSdkAPI.BackupPolicy, error) {
	return m.ReadBackupPolicy(ctx, id)
}

// Reset clears all mock data and resets counters
func (m *MockEonClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BackupPolicies = make(map[string]*externalEonSdkAPI.BackupPolicy)
	m.CreateCalls = 0
	m.ReadCalls = 0
	m.UpdateCalls = 0
	m.DeleteCalls = 0
	m.ListCalls = 0
	m.ShouldFailCreate = false
	m.ShouldFailRead = false
	m.ShouldFailUpdate = false
	m.ShouldFailDelete = false
	m.ShouldFailList = false
}

// AddMockPolicy adds a pre-defined mock policy for testing
func (m *MockEonClient) AddMockPolicy(policy *externalEonSdkAPI.BackupPolicy) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BackupPolicies[policy.Id] = policy
}

// GetMockPolicy retrieves a mock policy for testing
func (m *MockEonClient) GetMockPolicy(id string) (*externalEonSdkAPI.BackupPolicy, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policy, exists := m.BackupPolicies[id]
	return policy, exists
}
