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
	Permissions           []externalEonSdkAPI.Permission
	ResourceExclusions    map[string]bool
	DataClassesOverrides  map[string][]string
	EnvironmentOverrides  map[string]string
	InventoryResources    map[string]*externalEonSdkAPI.InventoryResource
	ResourceSnapshots     map[string][]externalEonSdkAPI.Snapshot

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
	ShouldFailBackupPostureControlCreate bool
	ShouldFailBackupPostureControlRead   bool
	ShouldFailBackupPostureControlUpdate bool
	ShouldFailBackupPostureControlDelete bool
	ShouldFailBackupPostureControlList   bool
	// IdP list behavior
	ShouldFailIdpList bool
	// Permissions / resource override behavior
	ShouldFailPermissionsList           bool
	ShouldFailExcludeResource           bool
	ShouldFailCancelResourceExclusion   bool
	ShouldFailOverrideDataClasses       bool
	ShouldFailRemoveDataClassesOverride bool
	ShouldFailOverrideEnvironment       bool
	ShouldFailRemoveEnvironmentOverride bool
	ShouldFailListResources             bool
	ShouldFailListResourceSnapshots     bool
	ShouldFailGetResource               bool

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
	BackupPostureControlCreateCalls int
	BackupPostureControlReadCalls   int
	BackupPostureControlUpdateCalls int
	BackupPostureControlDeleteCalls int
	BackupPostureControlListCalls   int
	IdpListCalls                    int
	PermissionsListCalls            int
	ExcludeResourceCalls            int
	CancelResourceExclusionCalls    int
	OverrideDataClassesCalls        int
	RemoveDataClassesOverrideCalls  int
	OverrideEnvironmentCalls        int
	RemoveEnvironmentOverrideCalls  int
	ListResourcesCalls              int
	ListResourceSnapshotsCalls      int
	GetResourceCalls                int

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
		Permissions:           []externalEonSdkAPI.Permission{},
		ResourceExclusions:    make(map[string]bool),
		DataClassesOverrides:  make(map[string][]string),
		EnvironmentOverrides:  make(map[string]string),
		InventoryResources:    make(map[string]*externalEonSdkAPI.InventoryResource),
		ResourceSnapshots:     make(map[string][]externalEonSdkAPI.Snapshot),
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
	m.BackupPostureControls = make(map[string]*externalEonSdkAPI.BackupPostureControl)
	m.IdpGroups = make(map[string]*externalEonSdkAPI.IdpGroup)
	m.Idps = make(map[string]*externalEonSdkAPI.Idp)
	m.Roles = make(map[string]*externalEonSdkAPI.Role)
	m.Permissions = []externalEonSdkAPI.Permission{}
	m.ResourceExclusions = make(map[string]bool)
	m.DataClassesOverrides = make(map[string][]string)
	m.EnvironmentOverrides = make(map[string]string)
	m.InventoryResources = make(map[string]*externalEonSdkAPI.InventoryResource)
	m.ResourceSnapshots = make(map[string][]externalEonSdkAPI.Snapshot)
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
	m.BackupPostureControlCreateCalls = 0
	m.BackupPostureControlReadCalls = 0
	m.BackupPostureControlUpdateCalls = 0
	m.BackupPostureControlDeleteCalls = 0
	m.BackupPostureControlListCalls = 0
	m.IdpListCalls = 0
	m.PermissionsListCalls = 0
	m.ExcludeResourceCalls = 0
	m.CancelResourceExclusionCalls = 0
	m.OverrideDataClassesCalls = 0
	m.RemoveDataClassesOverrideCalls = 0
	m.OverrideEnvironmentCalls = 0
	m.RemoveEnvironmentOverrideCalls = 0
	m.ListResourcesCalls = 0
	m.ListResourceSnapshotsCalls = 0
	m.GetResourceCalls = 0
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
	m.ShouldFailBackupPostureControlCreate = false
	m.ShouldFailBackupPostureControlRead = false
	m.ShouldFailBackupPostureControlUpdate = false
	m.ShouldFailBackupPostureControlDelete = false
	m.ShouldFailBackupPostureControlList = false
	m.ShouldFailIdpList = false
	m.ShouldFailPermissionsList = false
	m.ShouldFailExcludeResource = false
	m.ShouldFailCancelResourceExclusion = false
	m.ShouldFailOverrideDataClasses = false
	m.ShouldFailRemoveDataClassesOverride = false
	m.ShouldFailOverrideEnvironment = false
	m.ShouldFailRemoveEnvironmentOverride = false
	m.ShouldFailListResources = false
	m.ShouldFailListResourceSnapshots = false
	m.ShouldFailGetResource = false
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

// CreateBackupPostureControl mocks creating a backup posture control.
func (m *MockEonClient) CreateBackupPostureControl(ctx context.Context, req externalEonSdkAPI.CreateBackupPostureControlRequest) (*externalEonSdkAPI.BackupPostureControl, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BackupPostureControlCreateCalls++
	if m.ShouldFailBackupPostureControlCreate {
		return nil, fmt.Errorf("mock create backup posture control error")
	}

	id := fmt.Sprintf("mock-bpc-%d", m.BackupPostureControlCreateCalls)
	control := &externalEonSdkAPI.BackupPostureControl{
		Id:               id,
		Name:             req.GetName(),
		Severity:         req.GetSeverity(),
		ResourceSelector: req.ResourceSelector,
		Rules:            req.Rules,
	}
	m.BackupPostureControls[id] = control
	return control, nil
}

// GetBackupPostureControl mocks getting a backup posture control.
func (m *MockEonClient) GetBackupPostureControl(ctx context.Context, controlId string) (*externalEonSdkAPI.BackupPostureControl, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BackupPostureControlReadCalls++
	if m.ShouldFailBackupPostureControlRead {
		return nil, fmt.Errorf("mock read backup posture control error")
	}
	c, exists := m.BackupPostureControls[controlId]
	if !exists {
		return nil, fmt.Errorf("backup posture control not found: %s", controlId)
	}
	return c, nil
}

// UpdateBackupPostureControl mocks updating a backup posture control.
func (m *MockEonClient) UpdateBackupPostureControl(ctx context.Context, controlId string, req externalEonSdkAPI.UpdateBackupPostureControlRequest) (*externalEonSdkAPI.BackupPostureControl, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BackupPostureControlUpdateCalls++
	if m.ShouldFailBackupPostureControlUpdate {
		return nil, fmt.Errorf("mock update backup posture control error")
	}
	c, exists := m.BackupPostureControls[controlId]
	if !exists {
		return nil, fmt.Errorf("backup posture control not found: %s", controlId)
	}
	c.Name = req.GetName()
	c.Severity = req.GetSeverity()
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
	if m.ShouldFailBackupPostureControlDelete {
		return fmt.Errorf("mock delete backup posture control error")
	}
	if _, exists := m.BackupPostureControls[controlId]; !exists {
		return fmt.Errorf("backup posture control not found: %s", controlId)
	}
	delete(m.BackupPostureControls, controlId)
	return nil
}

// ListBackupPostureControls mocks listing backup posture controls.
func (m *MockEonClient) ListBackupPostureControls(ctx context.Context) ([]externalEonSdkAPI.BackupPostureControl, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BackupPostureControlListCalls++
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

// AddMockBackupPostureControl adds a pre-defined mock backup posture control.
func (m *MockEonClient) AddMockBackupPostureControl(control *externalEonSdkAPI.BackupPostureControl) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BackupPostureControls[control.Id] = control
}

// ListIdps mocks listing identity providers.
func (m *MockEonClient) ListIdps(ctx context.Context) ([]externalEonSdkAPI.Idp, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.IdpListCalls++
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
	m.Idps[idp.Id] = idp
}

// ListPermissions mocks listing permissions.
func (m *MockEonClient) ListPermissions(ctx context.Context) ([]externalEonSdkAPI.Permission, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PermissionsListCalls++
	if m.ShouldFailPermissionsList {
		return nil, fmt.Errorf("mock list permissions error")
	}
	out := make([]externalEonSdkAPI.Permission, len(m.Permissions))
	copy(out, m.Permissions)
	return out, nil
}

// AddMockPermission adds a pre-defined mock permission.
func (m *MockEonClient) AddMockPermission(permission *externalEonSdkAPI.Permission) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Permissions = append(m.Permissions, *permission)
}

// ExcludeResourceFromBackup mocks excluding a resource from backup.
func (m *MockEonClient) ExcludeResourceFromBackup(ctx context.Context, resourceId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ExcludeResourceCalls++
	if m.ShouldFailExcludeResource {
		return fmt.Errorf("mock exclude resource error")
	}
	m.ResourceExclusions[resourceId] = true
	if res, ok := m.InventoryResources[resourceId]; ok {
		res.BackupStatus = externalEonSdkAPI.EXCLUDED_FROM_BACKUP
	}
	return nil
}

// CancelResourceBackupExclusion mocks cancelling a resource backup exclusion.
func (m *MockEonClient) CancelResourceBackupExclusion(ctx context.Context, resourceId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CancelResourceExclusionCalls++
	if m.ShouldFailCancelResourceExclusion {
		return fmt.Errorf("mock cancel resource exclusion error")
	}
	delete(m.ResourceExclusions, resourceId)
	if res, ok := m.InventoryResources[resourceId]; ok {
		res.BackupStatus = externalEonSdkAPI.PROTECTED
	}
	return nil
}

// OverrideDataClasses mocks overriding data classes.
func (m *MockEonClient) OverrideDataClasses(ctx context.Context, resourceId string, dataClasses []string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.OverrideDataClassesCalls++
	if m.ShouldFailOverrideDataClasses {
		return nil, fmt.Errorf("mock override data classes error")
	}
	copied := append([]string{}, dataClasses...)
	m.DataClassesOverrides[resourceId] = copied
	if res, ok := m.InventoryResources[resourceId]; ok {
		details := externalEonSdkAPI.NewDataClassesDetails()
		details.SetDataClasses(copied)
		details.SetIsOverridden(true)
		if res.Classifications == nil {
			res.Classifications = externalEonSdkAPI.NewClassifications()
		}
		res.Classifications.SetDataClassesDetails(*details)
	}
	return copied, nil
}

// RemoveDataClassesOverride mocks removing a data classes override.
func (m *MockEonClient) RemoveDataClassesOverride(ctx context.Context, resourceId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RemoveDataClassesOverrideCalls++
	if m.ShouldFailRemoveDataClassesOverride {
		return fmt.Errorf("mock remove data classes override error")
	}
	delete(m.DataClassesOverrides, resourceId)
	if res, ok := m.InventoryResources[resourceId]; ok && res.Classifications != nil {
		details := externalEonSdkAPI.NewDataClassesDetails()
		details.SetIsOverridden(false)
		res.Classifications.SetDataClassesDetails(*details)
	}
	return nil
}

// GetResourceById mocks getting an inventory resource by ID.
func (m *MockEonClient) GetResourceById(ctx context.Context, resourceId string) (*externalEonSdkAPI.InventoryResource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetResourceCalls++
	if m.ShouldFailGetResource {
		return nil, fmt.Errorf("mock get resource error")
	}
	res, ok := m.InventoryResources[resourceId]
	if !ok {
		return nil, &APIError{StatusCode: 404, Message: "resource not found"}
	}
	copied := *res
	return &copied, nil
}

// AddMockInventoryResource adds a pre-defined mock inventory resource.
func (m *MockEonClient) AddMockInventoryResource(resource *externalEonSdkAPI.InventoryResource) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.InventoryResources[resource.Id] = resource
}

// OverrideEnvironment mocks overriding a resource environment.
func (m *MockEonClient) OverrideEnvironment(ctx context.Context, resourceId string, environment string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.OverrideEnvironmentCalls++
	if m.ShouldFailOverrideEnvironment {
		return "", fmt.Errorf("mock override environment error")
	}
	m.EnvironmentOverrides[resourceId] = environment
	if res, ok := m.InventoryResources[resourceId]; ok {
		details := externalEonSdkAPI.NewEnvironmentDetails()
		details.SetEnvironment(externalEonSdkAPI.Environment(environment))
		details.SetIsOverridden(true)
		if res.Classifications == nil {
			res.Classifications = externalEonSdkAPI.NewClassifications()
		}
		res.Classifications.SetEnvironmentDetails(*details)
	}
	return environment, nil
}

// RemoveEnvironmentOverride mocks removing an environment override.
func (m *MockEonClient) RemoveEnvironmentOverride(ctx context.Context, resourceId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RemoveEnvironmentOverrideCalls++
	if m.ShouldFailRemoveEnvironmentOverride {
		return fmt.Errorf("mock remove environment override error")
	}
	delete(m.EnvironmentOverrides, resourceId)
	if res, ok := m.InventoryResources[resourceId]; ok && res.Classifications != nil {
		details := externalEonSdkAPI.NewEnvironmentDetails()
		details.SetIsOverridden(false)
		res.Classifications.SetEnvironmentDetails(*details)
	}
	return nil
}

// ListResources mocks listing inventory resources.
func (m *MockEonClient) ListResources(ctx context.Context, filters *externalEonSdkAPI.InventoryFilterConditions) ([]externalEonSdkAPI.InventoryResource, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ListResourcesCalls++
	if m.ShouldFailListResources {
		return nil, fmt.Errorf("mock list resources error")
	}

	out := make([]externalEonSdkAPI.InventoryResource, 0, len(m.InventoryResources))
	for _, res := range m.InventoryResources {
		if filters != nil && filters.Id != nil && len(filters.Id.GetIn()) > 0 {
			matched := false
			for _, id := range filters.Id.GetIn() {
				if res.GetId() == id {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, *res)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Id < out[j].Id })
	return out, nil
}

// ListResourceSnapshots mocks listing snapshots for a resource.
func (m *MockEonClient) ListResourceSnapshots(ctx context.Context, resourceId string, filters *externalEonSdkAPI.SnapshotFilterConditions) ([]externalEonSdkAPI.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ListResourceSnapshotsCalls++
	if m.ShouldFailListResourceSnapshots {
		return nil, fmt.Errorf("mock list resource snapshots error")
	}
	snapshots := m.ResourceSnapshots[resourceId]
	out := make([]externalEonSdkAPI.Snapshot, len(snapshots))
	copy(out, snapshots)
	return out, nil
}

// AddMockResourceSnapshot adds a pre-defined mock snapshot for a resource.
func (m *MockEonClient) AddMockResourceSnapshot(resourceId string, snapshot *externalEonSdkAPI.Snapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ResourceSnapshots[resourceId] = append(m.ResourceSnapshots[resourceId], *snapshot)
}
