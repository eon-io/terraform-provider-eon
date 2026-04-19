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
	IdpGroups      map[string]*externalEonSdkAPI.IdpGroup
	Roles          map[string]*externalEonSdkAPI.Role

	// Behavior controls
	ShouldFailCreate bool
	ShouldFailRead   bool
	ShouldFailUpdate bool
	ShouldFailDelete bool
	ShouldFailList   bool
	// IDP group behavior (when set, IDP group methods return error)
	ShouldFailIdpGroupList   bool
	ShouldFailIdpGroupCreate bool
	ShouldFailIdpGroupRead   bool
	ShouldFailIdpGroupUpdate bool
	ShouldFailIdpGroupDelete bool

	// Call tracking
	CreateCalls         int
	ReadCalls           int
	UpdateCalls         int
	DeleteCalls         int
	ListCalls           int
	IdpGroupListCalls   int
	IdpGroupCreateCalls int
	IdpGroupReadCalls   int
	IdpGroupUpdateCalls int
	IdpGroupDeleteCalls int

	// Mock configuration
	ProjectID string
}

// NewMockEonClient creates a new mock client with default behavior
func NewMockEonClient() *MockEonClient {
	return &MockEonClient{
		BackupPolicies: make(map[string]*externalEonSdkAPI.BackupPolicy),
		IdpGroups:      make(map[string]*externalEonSdkAPI.IdpGroup),
		Roles:          make(map[string]*externalEonSdkAPI.Role),
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
	m.IdpGroups = make(map[string]*externalEonSdkAPI.IdpGroup)
	m.Roles = make(map[string]*externalEonSdkAPI.Role)
	m.CreateCalls = 0
	m.ReadCalls = 0
	m.UpdateCalls = 0
	m.DeleteCalls = 0
	m.ListCalls = 0
	m.IdpGroupListCalls = 0
	m.IdpGroupCreateCalls = 0
	m.IdpGroupReadCalls = 0
	m.IdpGroupUpdateCalls = 0
	m.IdpGroupDeleteCalls = 0
	m.ShouldFailCreate = false
	m.ShouldFailRead = false
	m.ShouldFailUpdate = false
	m.ShouldFailDelete = false
	m.ShouldFailList = false
	m.ShouldFailIdpGroupList = false
	m.ShouldFailIdpGroupCreate = false
	m.ShouldFailIdpGroupRead = false
	m.ShouldFailIdpGroupUpdate = false
	m.ShouldFailIdpGroupDelete = false
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

// ListIdpGroups mocks listing IDP groups
func (m *MockEonClient) ListIdpGroups(ctx context.Context) ([]externalEonSdkAPI.IdpGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.IdpGroupListCalls++

	if m.ShouldFailIdpGroupList {
		return nil, fmt.Errorf("mock idp group list error")
	}

	groups := make([]externalEonSdkAPI.IdpGroup, 0, len(m.IdpGroups))
	for _, g := range m.IdpGroups {
		groups = append(groups, *g)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Id < groups[j].Id })
	return groups, nil
}

// GetIdpGroup mocks getting an IDP group by ID
func (m *MockEonClient) GetIdpGroup(ctx context.Context, groupId string) (*externalEonSdkAPI.IdpGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.IdpGroupReadCalls++

	if m.ShouldFailIdpGroupRead {
		return nil, fmt.Errorf("mock idp group read error")
	}

	g, exists := m.IdpGroups[groupId]
	if !exists {
		return nil, fmt.Errorf("idp group not found: %s", groupId)
	}
	return g, nil
}

// CreateIdpGroup mocks creating an IDP group
func (m *MockEonClient) CreateIdpGroup(ctx context.Context, req externalEonSdkAPI.CreateIdpGroupRequest) (*externalEonSdkAPI.IdpGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.IdpGroupCreateCalls++

	if m.ShouldFailIdpGroupCreate {
		return nil, fmt.Errorf("mock idp group create error")
	}

	id := fmt.Sprintf("mock-idp-group-%d", m.IdpGroupCreateCalls)
	group := &externalEonSdkAPI.IdpGroup{
		Id:              id,
		IdpId:           req.GetIdpId(),
		ProviderGroupId: req.GetProviderGroupId(),
		RoleIds:         req.GetRoleIds(),
	}
	m.IdpGroups[id] = group
	return group, nil
}

// UpdateIdpGroup mocks updating an IDP group's role assignments
func (m *MockEonClient) UpdateIdpGroup(ctx context.Context, groupId string, req externalEonSdkAPI.UpdateIdpGroupRequest) (*externalEonSdkAPI.IdpGroup, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.IdpGroupUpdateCalls++

	if m.ShouldFailIdpGroupUpdate {
		return nil, fmt.Errorf("mock idp group update error")
	}

	g, exists := m.IdpGroups[groupId]
	if !exists {
		return nil, fmt.Errorf("idp group not found: %s", groupId)
	}
	g.RoleIds = req.GetRoleIds()
	m.IdpGroups[groupId] = g
	return g, nil
}

// DeleteIdpGroup mocks deleting an IDP group
func (m *MockEonClient) DeleteIdpGroup(ctx context.Context, groupId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.IdpGroupDeleteCalls++

	if m.ShouldFailIdpGroupDelete {
		return fmt.Errorf("mock idp group delete error")
	}

	_, exists := m.IdpGroups[groupId]
	if !exists {
		return fmt.Errorf("idp group not found: %s", groupId)
	}
	delete(m.IdpGroups, groupId)
	return nil
}

// AddMockIdpGroup adds a pre-defined mock IDP group for testing
func (m *MockEonClient) AddMockIdpGroup(group *externalEonSdkAPI.IdpGroup) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.IdpGroups[group.Id] = group
}

// ListRoles mocks listing roles
func (m *MockEonClient) ListRoles(ctx context.Context) ([]externalEonSdkAPI.Role, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	roles := make([]externalEonSdkAPI.Role, 0, len(m.Roles))
	for _, r := range m.Roles {
		roles = append(roles, *r)
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i].Id < roles[j].Id })
	return roles, nil
}

// GetRole mocks getting a role by ID
func (m *MockEonClient) GetRole(ctx context.Context, roleId string) (*externalEonSdkAPI.Role, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	r, exists := m.Roles[roleId]
	if !exists {
		return nil, fmt.Errorf("role not found: %s", roleId)
	}
	return r, nil
}

// CreateRole mocks creating a role
func (m *MockEonClient) CreateRole(ctx context.Context, req externalEonSdkAPI.CreateRoleRequest) (*externalEonSdkAPI.Role, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := fmt.Sprintf("mock-role-%d", len(m.Roles)+1)
	role := &externalEonSdkAPI.Role{
		Id:               id,
		Name:             req.GetName(),
		IsBuiltInRole:    false,
		PermissionGrants: permissionGrantInputToGrant(req.GetPermissionGrants()),
	}
	if req.AccessConditions != nil {
		role.AccessConditions = req.AccessConditions
	}
	if req.HasRestoreDestinationLimits() {
		rdl := req.GetRestoreDestinationLimits()
		role.RestoreDestinationLimits = *externalEonSdkAPI.NewNullableRestoreDestinationLimits(&rdl)
	}
	m.Roles[id] = role
	return role, nil
}

func permissionGrantInputToGrant(in []externalEonSdkAPI.PermissionGrantInput) []externalEonSdkAPI.PermissionGrant {
	out := make([]externalEonSdkAPI.PermissionGrant, 0, len(in))
	for _, p := range in {
		g := externalEonSdkAPI.PermissionGrant{Permission: p.Permission, AccessConditionId: p.AccessConditionId}
		out = append(out, g)
	}
	return out
}

// UpdateRole mocks updating a role
func (m *MockEonClient) UpdateRole(ctx context.Context, roleId string, req externalEonSdkAPI.UpdateRoleRequest) (*externalEonSdkAPI.Role, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	r, exists := m.Roles[roleId]
	if !exists {
		return nil, fmt.Errorf("role not found: %s", roleId)
	}
	r.Name = req.GetName()
	r.PermissionGrants = permissionGrantInputToGrant(req.GetPermissionGrants())
	r.AccessConditions = req.AccessConditions
	if req.HasRestoreDestinationLimits() {
		rdl := req.GetRestoreDestinationLimits()
		r.RestoreDestinationLimits = *externalEonSdkAPI.NewNullableRestoreDestinationLimits(&rdl)
	} else {
		r.RestoreDestinationLimits.Unset()
	}
	m.Roles[roleId] = r
	return r, nil
}

// DeleteRole mocks deleting a role
func (m *MockEonClient) DeleteRole(ctx context.Context, roleId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, exists := m.Roles[roleId]
	if !exists {
		return fmt.Errorf("role not found: %s", roleId)
	}
	delete(m.Roles, roleId)
	return nil
}
