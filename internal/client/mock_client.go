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
	BackupPolicies        map[string]*externalEonSdkAPI.BackupPolicy
	BackupPostureControls map[string]*externalEonSdkAPI.BackupPostureControl
	IdpGroups             map[string]*externalEonSdkAPI.IdpGroup
	Idps                  map[string]*externalEonSdkAPI.Idp
	Roles                 map[string]*externalEonSdkAPI.Role

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
	// Backup posture control behavior
	ShouldFailBackupPostureControlList   bool
	ShouldFailBackupPostureControlCreate bool
	ShouldFailBackupPostureControlRead   bool
	ShouldFailBackupPostureControlUpdate bool
	ShouldFailBackupPostureControlDelete bool
	// Identity provider list behavior
	ShouldFailIdpList bool

	// Call tracking
	CreateCalls                     int
	ReadCalls                       int
	UpdateCalls                     int
	DeleteCalls                     int
	ListCalls                       int
	IdpGroupListCalls               int
	IdpGroupCreateCalls             int
	IdpGroupReadCalls               int
	IdpGroupUpdateCalls             int
	IdpGroupDeleteCalls             int
	BackupPostureControlListCalls   int
	BackupPostureControlCreateCalls int
	BackupPostureControlReadCalls   int
	BackupPostureControlUpdateCalls int
	BackupPostureControlDeleteCalls int
	IdpListCalls                    int

	// Mock configuration
	ProjectID string
}

// NewMockEonClient creates a new mock client with default behavior
func NewMockEonClient() *MockEonClient {
	return &MockEonClient{
		BackupPolicies:        make(map[string]*externalEonSdkAPI.BackupPolicy),
		BackupPostureControls: make(map[string]*externalEonSdkAPI.BackupPostureControl),
		IdpGroups:             make(map[string]*externalEonSdkAPI.IdpGroup),
		Idps:                  make(map[string]*externalEonSdkAPI.Idp),
		Roles:                 make(map[string]*externalEonSdkAPI.Role),
		ProjectID:             "mock-project-id",
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

// ExcludeVolumeFromBackup mocks excluding a volume from backup
func (m *MockEonClient) ExcludeVolumeFromBackup(ctx context.Context, resourceId, volumeId string) error {
	return nil
}

// CancelVolumeBackupExclusion mocks cancelling a volume backup exclusion
func (m *MockEonClient) CancelVolumeBackupExclusion(ctx context.Context, resourceId, volumeId string) error {
	return nil
}

// Backup posture control mock state
func (m *MockEonClient) ensureBackupPostureControls() {
	if m.BackupPostureControls == nil {
		m.BackupPostureControls = make(map[string]*externalEonSdkAPI.BackupPostureControl)
	}
}

func (m *MockEonClient) ensureIdps() {
	if m.Idps == nil {
		m.Idps = make(map[string]*externalEonSdkAPI.Idp)
	}
}

// ListBackupPostureControls mocks listing backup posture controls.
func (m *MockEonClient) ListBackupPostureControls(ctx context.Context) ([]externalEonSdkAPI.BackupPostureControl, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BackupPostureControlListCalls++
	m.ensureBackupPostureControls()

	if m.ShouldFailBackupPostureControlList {
		return nil, fmt.Errorf("mock list backup posture controls error")
	}

	out := make([]externalEonSdkAPI.BackupPostureControl, 0, len(m.BackupPostureControls))
	for _, c := range m.BackupPostureControls {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Id < out[j].Id })
	return out, nil
}

// GetBackupPostureControl mocks getting a backup posture control by ID.
func (m *MockEonClient) GetBackupPostureControl(ctx context.Context, controlId string) (*externalEonSdkAPI.BackupPostureControl, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BackupPostureControlReadCalls++
	m.ensureBackupPostureControls()

	if m.ShouldFailBackupPostureControlRead {
		return nil, fmt.Errorf("mock get backup posture control error")
	}

	c, exists := m.BackupPostureControls[controlId]
	if !exists {
		return nil, &APIError{StatusCode: 404, Message: "backup posture control not found"}
	}
	return c, nil
}

// CreateBackupPostureControl mocks creating a backup posture control.
func (m *MockEonClient) CreateBackupPostureControl(ctx context.Context, req externalEonSdkAPI.CreateBackupPostureControlRequest) (*externalEonSdkAPI.BackupPostureControl, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BackupPostureControlCreateCalls++
	m.ensureBackupPostureControls()

	if m.ShouldFailBackupPostureControlCreate {
		return nil, fmt.Errorf("mock create backup posture control error")
	}

	id := fmt.Sprintf("mock-bpc-%d", m.BackupPostureControlCreateCalls)
	control := &externalEonSdkAPI.BackupPostureControl{
		Id:               id,
		Name:             req.Name,
		Severity:         req.Severity,
		ResourceSelector: req.ResourceSelector,
		Rules:            req.Rules,
	}
	m.BackupPostureControls[id] = control
	return control, nil
}

// UpdateBackupPostureControl mocks updating a backup posture control.
func (m *MockEonClient) UpdateBackupPostureControl(ctx context.Context, controlId string, req externalEonSdkAPI.UpdateBackupPostureControlRequest) (*externalEonSdkAPI.BackupPostureControl, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BackupPostureControlUpdateCalls++
	m.ensureBackupPostureControls()

	if m.ShouldFailBackupPostureControlUpdate {
		return nil, fmt.Errorf("mock update backup posture control error")
	}

	c, exists := m.BackupPostureControls[controlId]
	if !exists {
		return nil, &APIError{StatusCode: 404, Message: "backup posture control not found"}
	}
	c.Name = req.Name
	c.Severity = req.Severity
	c.ResourceSelector = req.ResourceSelector
	c.Rules = req.Rules
	m.BackupPostureControls[controlId] = c
	return c, nil
}

// DeleteBackupPostureControl mocks deleting a backup posture control.
func (m *MockEonClient) DeleteBackupPostureControl(ctx context.Context, controlId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BackupPostureControlDeleteCalls++
	m.ensureBackupPostureControls()

	if m.ShouldFailBackupPostureControlDelete {
		return fmt.Errorf("mock delete backup posture control error")
	}

	if _, exists := m.BackupPostureControls[controlId]; !exists {
		return &APIError{StatusCode: 404, Message: "backup posture control not found"}
	}
	delete(m.BackupPostureControls, controlId)
	return nil
}

// AddMockBackupPostureControl adds a pre-defined mock backup posture control.
func (m *MockEonClient) AddMockBackupPostureControl(control *externalEonSdkAPI.BackupPostureControl) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureBackupPostureControls()
	m.BackupPostureControls[control.Id] = control
}

// ListIdps mocks listing identity providers.
func (m *MockEonClient) ListIdps(ctx context.Context) ([]externalEonSdkAPI.Idp, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.IdpListCalls++
	m.ensureIdps()

	if m.ShouldFailIdpList {
		return nil, fmt.Errorf("mock list identity providers error")
	}

	out := make([]externalEonSdkAPI.Idp, 0, len(m.Idps))
	for _, idp := range m.Idps {
		out = append(out, *idp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Id < out[j].Id })
	return out, nil
}

// AddMockIdp adds a pre-defined mock identity provider.
func (m *MockEonClient) AddMockIdp(idp *externalEonSdkAPI.Idp) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureIdps()
	m.Idps[idp.Id] = idp
}
