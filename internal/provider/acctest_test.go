package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	externalEonSdkAPI "github.com/eon-io/eon-sdk-go"
	"github.com/eon-io/terraform-provider-eon/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	testAccProjectID    = "test-project"
	testAccClientID     = "test-client-id"
	testAccClientSecret = "test-client-secret"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server that Terraform can connect to.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"eon": providerserver.NewProtocol6WithError(New("test", client.NewClient)()),
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless TF_ACC=1")
	}
}

// testAccRealEnvPreCheck requires live Eon credentials. Optionally also require
// EON_TEST_RESOURCE_ID when the test mutates/reads a specific inventory resource.
func testAccRealEnvPreCheck(t *testing.T, requireResourceID bool) {
	t.Helper()
	testAccPreCheck(t)

	required := []string{
		"EON_ENDPOINT",
		"EON_CLIENT_ID",
		"EON_CLIENT_SECRET",
		"EON_PROJECT_ID",
	}
	if requireResourceID {
		required = append(required, "EON_TEST_RESOURCE_ID")
	}
	for _, key := range required {
		if os.Getenv(key) == "" {
			t.Skipf("%s must be set for real-environment acceptance tests", key)
		}
	}
}

func testAccRealProviderConfig() string {
	return fmt.Sprintf(`
provider "eon" {
  endpoint      = %q
  client_id     = %q
  client_secret = %q
  project_id    = %q
}
`, os.Getenv("EON_ENDPOINT"), os.Getenv("EON_CLIENT_ID"), os.Getenv("EON_CLIENT_SECRET"), os.Getenv("EON_PROJECT_ID"))
}

// fakeEonServer is an in-memory Eon API used by acceptance tests.
type fakeEonServer struct {
	server    *httptest.Server
	mu        sync.Mutex
	projectID string

	controls             map[string]*externalEonSdkAPI.BackupPostureControl
	idps                 map[string]*externalEonSdkAPI.Idp
	permissions          []externalEonSdkAPI.Permission
	resources            map[string]*externalEonSdkAPI.InventoryResource
	snapshots            map[string][]externalEonSdkAPI.Snapshot
	snapshotsByID        map[string]*externalEonSdkAPI.Snapshot
	metricsConfigs       map[string]*externalEonSdkAPI.RestoreAccountMetricsConfig
	sourceMetricsConfigs map[string]*externalEonSdkAPI.SourceAccountMetricsConfig
	actionApprovalRules  map[string]*externalEonSdkAPI.ActionApprovalRule
	restoreJobs          map[string]*externalEonSdkAPI.RestoreJob
	nextID               int
}

func newFakeEonServer(t *testing.T) *fakeEonServer {
	t.Helper()
	f := &fakeEonServer{
		projectID:            testAccProjectID,
		controls:             make(map[string]*externalEonSdkAPI.BackupPostureControl),
		idps:                 make(map[string]*externalEonSdkAPI.Idp),
		permissions:          []externalEonSdkAPI.Permission{},
		resources:            make(map[string]*externalEonSdkAPI.InventoryResource),
		snapshots:            make(map[string][]externalEonSdkAPI.Snapshot),
		snapshotsByID:        make(map[string]*externalEonSdkAPI.Snapshot),
		metricsConfigs:       make(map[string]*externalEonSdkAPI.RestoreAccountMetricsConfig),
		sourceMetricsConfigs: make(map[string]*externalEonSdkAPI.SourceAccountMetricsConfig),
		actionApprovalRules:  make(map[string]*externalEonSdkAPI.ActionApprovalRule),
		restoreJobs:          make(map[string]*externalEonSdkAPI.RestoreJob),
		nextID:               1,
	}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeEonServer) URL() string {
	return f.server.URL
}

func (f *fakeEonServer) providerConfig() string {
	return fmt.Sprintf(`
provider "eon" {
  endpoint      = %q
  client_id     = %q
  client_secret = %q
  project_id    = %q
}
`, f.URL(), testAccClientID, testAccClientSecret, f.projectID)
}

func (f *fakeEonServer) DeleteControl(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.controls, id)
}

func (f *fakeEonServer) AddIdp(idp *externalEonSdkAPI.Idp) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.idps[idp.Id] = idp
}

func (f *fakeEonServer) AddPermission(permission *externalEonSdkAPI.Permission) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.permissions = append(f.permissions, *permission)
}

func (f *fakeEonServer) AddResource(resource *externalEonSdkAPI.InventoryResource) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.resources[resource.Id] = resource
}

func (f *fakeEonServer) CancelResourceExclusion(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if res, ok := f.resources[id]; ok {
		res.BackupStatus = externalEonSdkAPI.PROTECTED
	}
}

func (f *fakeEonServer) RemoveDataClassesOverride(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if res, ok := f.resources[id]; ok && res.Classifications != nil {
		details := externalEonSdkAPI.NewDataClassesDetails()
		details.SetIsOverridden(false)
		details.SetDataClasses([]string{})
		res.Classifications.SetDataClassesDetails(*details)
	}
}

func (f *fakeEonServer) RemoveEnvironmentOverride(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if res, ok := f.resources[id]; ok && res.Classifications != nil {
		details := externalEonSdkAPI.NewEnvironmentDetails()
		details.SetIsOverridden(false)
		res.Classifications.SetEnvironmentDetails(*details)
	}
}

func (f *fakeEonServer) AddSnapshot(resourceID string, snapshot *externalEonSdkAPI.Snapshot) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.snapshots[resourceID] = append(f.snapshots[resourceID], *snapshot)
	f.snapshotsByID[snapshot.Id] = snapshot
}

func (f *fakeEonServer) DeleteMetricsConfig(accountID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.metricsConfigs, accountID)
}

func (f *fakeEonServer) DeleteSourceMetricsConfig(accountID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sourceMetricsConfigs, accountID)
}

func (f *fakeEonServer) DeleteActionApprovalRule(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.actionApprovalRules, id)
}

func (f *fakeEonServer) RemoveHold(snapshotID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if snap, ok := f.snapshotsByID[snapshotID]; ok {
		snap.SetOnHold(false)
		snap.HoldDescription = nil
	}
}

func (f *fakeEonServer) DeleteRestoreJob(jobID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.restoreJobs, jobID)
}

func (f *fakeEonServer) handle(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/api")

	switch {
	case r.Method == http.MethodPost && path == "/v1/token":
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"accessToken":       "test-token",
			"expirationSeconds": 43200,
		})
		return
	case r.Method == http.MethodPost && path == "/v1/oauth2/token":
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"access_token": "test-token",
			"expires_in":   43200,
			"token_type":   "Bearer",
		})
		return
	}

	prefix := fmt.Sprintf("/v1/projects/%s/backup-posture-controls", f.projectID)
	switch {
	case r.Method == http.MethodPost && path == prefix:
		f.handleCreateControl(w, r)
		return
	case r.Method == http.MethodPost && path == prefix+"/list":
		f.handleListControls(w, r)
		return
	case strings.HasPrefix(path, prefix+"/"):
		id := strings.TrimPrefix(path, prefix+"/")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			f.handleGetControl(w, id)
		case http.MethodPut:
			f.handleUpdateControl(w, r, id)
		case http.MethodDelete:
			f.handleDeleteControl(w, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	case r.Method == http.MethodPost && path == "/v1/idps/list":
		f.handleListIdps(w, r)
		return
	case r.Method == http.MethodGet && path == "/v1/permissions":
		f.handleListPermissions(w, r)
		return
	}

	actionApprovalRulesPrefix := fmt.Sprintf("/v1/projects/%s/action-approvals/rules", f.projectID)
	switch {
	case path == actionApprovalRulesPrefix && r.Method == http.MethodGet:
		f.handleListActionApprovalRules(w, r)
		return
	case path == actionApprovalRulesPrefix && r.Method == http.MethodPost:
		f.handleCreateActionApprovalRule(w, r)
		return
	case strings.HasPrefix(path, actionApprovalRulesPrefix+"/"):
		id := strings.TrimPrefix(path, actionApprovalRulesPrefix+"/")
		if id == "" || strings.Contains(id, "/") {
			http.NotFound(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			f.handleGetActionApprovalRule(w, id)
		case http.MethodPut:
			f.handleUpdateActionApprovalRule(w, r, id)
		case http.MethodDelete:
			f.handleDeleteActionApprovalRule(w, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	restoreAccountsPrefix := fmt.Sprintf("/v1/projects/%s/restore-accounts/", f.projectID)
	if strings.HasPrefix(path, restoreAccountsPrefix) {
		rest := strings.TrimPrefix(path, restoreAccountsPrefix)
		parts := strings.Split(rest, "/")
		if len(parts) == 2 && parts[1] == "metrics-config" {
			switch r.Method {
			case http.MethodGet:
				f.handleGetMetricsConfig(w, parts[0])
				return
			case http.MethodPut:
				f.handleEnableMetricsConfig(w, r, parts[0])
				return
			case http.MethodDelete:
				f.handleDisableMetricsConfig(w, parts[0])
				return
			}
		}
	}

	sourceAccountsPrefix := fmt.Sprintf("/v1/projects/%s/source-accounts/", f.projectID)
	if strings.HasPrefix(path, sourceAccountsPrefix) {
		rest := strings.TrimPrefix(path, sourceAccountsPrefix)
		parts := strings.Split(rest, "/")
		if len(parts) == 2 && parts[1] == "metrics-config" {
			switch r.Method {
			case http.MethodGet:
				f.handleGetSourceMetricsConfig(w, parts[0])
				return
			case http.MethodPut:
				f.handleEnableSourceMetricsConfig(w, r, parts[0])
				return
			case http.MethodDelete:
				f.handleDisableSourceMetricsConfig(w, parts[0])
				return
			}
		}
	}

	snapshotsPrefix := fmt.Sprintf("/v1/projects/%s/snapshots/", f.projectID)
	if strings.HasPrefix(path, snapshotsPrefix) {
		rest := strings.TrimPrefix(path, snapshotsPrefix)
		parts := strings.Split(rest, "/")
		if len(parts) == 1 && r.Method == http.MethodGet {
			f.handleGetSnapshot(w, parts[0])
			return
		}
		if len(parts) == 2 && parts[1] == "hold" && r.Method == http.MethodPatch {
			f.handleHoldSnapshot(w, r, parts[0])
			return
		}
		if len(parts) == 2 && parts[1] == "remove-hold" && r.Method == http.MethodPatch {
			f.handleRemoveSnapshotHold(w, parts[0])
			return
		}
	}

	restoreJobsPrefix := fmt.Sprintf("/v1/projects/%s/restore-jobs/", f.projectID)
	if strings.HasPrefix(path, restoreJobsPrefix) && r.Method == http.MethodGet {
		jobID := strings.TrimPrefix(path, restoreJobsPrefix)
		if jobID != "" && !strings.Contains(jobID, "/") {
			f.handleGetRestoreJob(w, jobID)
			return
		}
	}

	resourcesPrefix := fmt.Sprintf("/v1/projects/%s/resources/", f.projectID)
	if strings.HasPrefix(path, resourcesPrefix) {
		rest := strings.TrimPrefix(path, resourcesPrefix)
		parts := strings.Split(rest, "/")
		if len(parts) == 1 && r.Method == http.MethodGet {
			f.handleGetResource(w, parts[0])
			return
		}
		if len(parts) == 2 && parts[1] == "exclude" && r.Method == http.MethodPatch {
			f.handleExcludeResource(w, parts[0])
			return
		}
		if len(parts) == 2 && parts[1] == "include" && r.Method == http.MethodPatch {
			f.handleIncludeResource(w, parts[0])
			return
		}
		if len(parts) == 2 && parts[1] == "data-classifications" {
			switch r.Method {
			case http.MethodPatch:
				f.handleOverrideDataClasses(w, r, parts[0])
				return
			case http.MethodDelete:
				f.handleRemoveDataClassesOverride(w, parts[0])
				return
			}
		}
		if len(parts) == 2 && parts[1] == "environments" {
			switch r.Method {
			case http.MethodPatch:
				f.handleOverrideEnvironment(w, r, parts[0])
				return
			case http.MethodDelete:
				f.handleRemoveEnvironmentOverride(w, parts[0])
				return
			}
		}
		if len(parts) == 2 && parts[1] == "snapshots" && r.Method == http.MethodPost {
			f.handleListResourceSnapshots(w, r, parts[0])
			return
		}
		if len(parts) == 4 && parts[1] == "snapshots" && r.Method == http.MethodPost {
			f.handleStartRestore(w, r, parts[0], parts[2], parts[3])
			return
		}
	}

	http.NotFound(w, r)
}

func (f *fakeEonServer) handleCreateControl(w http.ResponseWriter, r *http.Request) {
	var req externalEonSdkAPI.CreateBackupPostureControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	id := fmt.Sprintf("bpc-%d", f.nextID)
	f.nextID++
	control := &externalEonSdkAPI.BackupPostureControl{
		Id:               id,
		Name:             req.GetName(),
		Severity:         req.GetSeverity(),
		ResourceSelector: req.ResourceSelector,
		Rules:            req.Rules,
	}
	f.controls[id] = control
	writeJSON(w, http.StatusCreated, control)
}

func (f *fakeEonServer) handleGetControl(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	control, ok := f.controls[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, control)
}

func (f *fakeEonServer) handleUpdateControl(w http.ResponseWriter, r *http.Request, id string) {
	var req externalEonSdkAPI.UpdateBackupPostureControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	control, ok := f.controls[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	control.Name = req.GetName()
	control.Severity = req.GetSeverity()
	control.ResourceSelector = req.ResourceSelector
	control.Rules = req.Rules
	f.controls[id] = control
	writeJSON(w, http.StatusOK, control)
}

func (f *fakeEonServer) handleDeleteControl(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.controls[id]; !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	delete(f.controls, id)
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeEonServer) handleListControls(w http.ResponseWriter, r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)

	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]externalEonSdkAPI.BackupPostureControl, 0, len(f.controls))
	for _, c := range f.controls {
		items = append(items, *c)
	}
	writeJSON(w, http.StatusOK, externalEonSdkAPI.ListBackupPostureControlsResponse{
		BackupPostureControls: items,
	})
}

func (f *fakeEonServer) handleListIdps(w http.ResponseWriter, r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)

	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]externalEonSdkAPI.Idp, 0, len(f.idps))
	for _, idp := range f.idps {
		items = append(items, *idp)
	}
	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewListIdpsResponse(items, int32(len(items))))
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (f *fakeEonServer) handleListPermissions(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]externalEonSdkAPI.Permission, len(f.permissions))
	copy(items, f.permissions)
	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewListPermissionsResponse(items))
}

func (f *fakeEonServer) handleGetResource(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	res, ok := f.resources[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewGetResourceResponse(*res))
}

func (f *fakeEonServer) handleExcludeResource(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	res, ok := f.resources[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	res.BackupStatus = externalEonSdkAPI.EXCLUDED_FROM_BACKUP
	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewExcludeFromBackupResponse(true))
}

func (f *fakeEonServer) handleIncludeResource(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	res, ok := f.resources[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	res.BackupStatus = externalEonSdkAPI.PROTECTED
	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewCancelExclusionFromBackupResponse(true))
}

func (f *fakeEonServer) handleOverrideDataClasses(w http.ResponseWriter, r *http.Request, id string) {
	var req externalEonSdkAPI.OverrideDataClassificationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	res, ok := f.resources[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	dataClasses := req.GetDataClasses()
	if dataClasses == nil {
		dataClasses = []string{}
	}
	details := externalEonSdkAPI.NewDataClassesDetails()
	details.SetDataClasses(dataClasses)
	details.SetIsOverridden(true)
	if res.Classifications == nil {
		res.Classifications = externalEonSdkAPI.NewClassifications()
	}
	res.Classifications.SetDataClassesDetails(*details)
	resp := externalEonSdkAPI.NewOverrideDataClassificationsResponse()
	resp.SetDataClasses(dataClasses)
	writeJSON(w, http.StatusOK, resp)
}

func (f *fakeEonServer) handleRemoveDataClassesOverride(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	res, ok := f.resources[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	details := externalEonSdkAPI.NewDataClassesDetails()
	details.SetIsOverridden(false)
	details.SetDataClasses([]string{})
	if res.Classifications == nil {
		res.Classifications = externalEonSdkAPI.NewClassifications()
	}
	res.Classifications.SetDataClassesDetails(*details)
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeEonServer) handleOverrideEnvironment(w http.ResponseWriter, r *http.Request, id string) {
	var req externalEonSdkAPI.OverrideEnvironmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	res, ok := f.resources[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	environment := req.GetEnvironment()
	details := externalEonSdkAPI.NewEnvironmentDetails()
	details.SetEnvironment(environment)
	details.SetIsOverridden(true)
	if res.Classifications == nil {
		res.Classifications = externalEonSdkAPI.NewClassifications()
	}
	res.Classifications.SetEnvironmentDetails(*details)
	resp := externalEonSdkAPI.NewOverrideEnvironmentResponse()
	resp.SetEnvironment(environment)
	writeJSON(w, http.StatusOK, resp)
}

func (f *fakeEonServer) handleRemoveEnvironmentOverride(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	res, ok := f.resources[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	details := externalEonSdkAPI.NewEnvironmentDetails()
	details.SetIsOverridden(false)
	if res.Classifications == nil {
		res.Classifications = externalEonSdkAPI.NewClassifications()
	}
	res.Classifications.SetEnvironmentDetails(*details)
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeEonServer) handleListResourceSnapshots(w http.ResponseWriter, r *http.Request, id string) {
	_, _ = io.Copy(io.Discard, r.Body)

	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.resources[id]; !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	items := append([]externalEonSdkAPI.Snapshot{}, f.snapshots[id]...)
	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewListInventorySnapshotsResponse(items, int32(len(items))))
}

func (f *fakeEonServer) handleGetMetricsConfig(w http.ResponseWriter, accountID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	config, ok := f.metricsConfigs[accountID]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewGetRestoreAccountMetricsConfigResponse(*config))
}

func (f *fakeEonServer) handleEnableMetricsConfig(w http.ResponseWriter, r *http.Request, accountID string) {
	var req externalEonSdkAPI.EnableRestoreAccountMetricsConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	destination := externalEonSdkAPI.NewAccountMetricsDestination()
	if req.HasAws() {
		destination.SetAws(req.GetAws())
	}

	config := externalEonSdkAPI.NewRestoreAccountMetricsConfig(accountID, true, *destination)
	f.mu.Lock()
	f.metricsConfigs[accountID] = config
	f.mu.Unlock()

	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewEnableRestoreAccountMetricsConfigResponse(*config))
}

func (f *fakeEonServer) handleDisableMetricsConfig(w http.ResponseWriter, accountID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.metricsConfigs[accountID]; !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	delete(f.metricsConfigs, accountID)
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeEonServer) handleGetSourceMetricsConfig(w http.ResponseWriter, accountID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	config, ok := f.sourceMetricsConfigs[accountID]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewGetSourceAccountMetricsConfigResponse(*config))
}

func (f *fakeEonServer) handleEnableSourceMetricsConfig(w http.ResponseWriter, r *http.Request, accountID string) {
	var req externalEonSdkAPI.EnableSourceAccountMetricsConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	destination := externalEonSdkAPI.NewAccountMetricsDestination()
	if req.HasAws() {
		destination.SetAws(req.GetAws())
	}

	config := externalEonSdkAPI.NewSourceAccountMetricsConfig(accountID, true, *destination)
	f.mu.Lock()
	f.sourceMetricsConfigs[accountID] = config
	f.mu.Unlock()

	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewEnableSourceAccountMetricsConfigResponse(*config))
}

func (f *fakeEonServer) handleDisableSourceMetricsConfig(w http.ResponseWriter, accountID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.sourceMetricsConfigs[accountID]; !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	delete(f.sourceMetricsConfigs, accountID)
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeEonServer) handleCreateActionApprovalRule(w http.ResponseWriter, r *http.Request) {
	var req externalEonSdkAPI.CreateActionApprovalRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	id := fmt.Sprintf("aar-%d", f.nextID)
	f.nextID++

	required := int32(1)
	if req.HasRequiredApprovals() {
		required = req.GetRequiredApprovals()
	}

	rule := externalEonSdkAPI.NewActionApprovalRule(
		id,
		f.projectID,
		req.GetOperation(),
		required,
		req.GetApprovalWindowHours(),
		req.GetExecutionWindowHours(),
	)
	if req.HasDescription() {
		rule.SetDescription(req.GetDescription())
	}
	if req.HasResourceSelector() {
		rule.SetResourceSelector(req.GetResourceSelector())
	}
	if req.HasApproverIdpId() {
		rule.SetApproverIdpId(req.GetApproverIdpId())
	}
	if req.HasApproverProviderGroupId() {
		rule.SetApproverProviderGroupId(req.GetApproverProviderGroupId())
	}
	if req.HasExemptApiCredentials() {
		rule.SetExemptApiCredentials(req.GetExemptApiCredentials())
	}

	f.actionApprovalRules[id] = rule
	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewCreateActionApprovalRuleResponse(*externalEonSdkAPI.NewNullableActionApprovalRule(rule)))
}

func (f *fakeEonServer) handleGetActionApprovalRule(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rule, ok := f.actionApprovalRules[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewGetActionApprovalRuleResponse(*externalEonSdkAPI.NewNullableActionApprovalRule(rule)))
}

func (f *fakeEonServer) handleUpdateActionApprovalRule(w http.ResponseWriter, r *http.Request, id string) {
	var req externalEonSdkAPI.UpdateActionApprovalRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	rule, ok := f.actionApprovalRules[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}

	if req.HasRequiredApprovals() {
		rule.SetRequiredApprovals(req.GetRequiredApprovals())
	}
	if req.HasApprovalWindowHours() {
		rule.SetApprovalWindowHours(req.GetApprovalWindowHours())
	}
	if req.HasExecutionWindowHours() {
		rule.SetExecutionWindowHours(req.GetExecutionWindowHours())
	}
	if req.HasDescription() {
		rule.SetDescription(req.GetDescription())
	}
	if req.HasResourceSelector() {
		if req.ResourceSelector.Get() == nil {
			rule.UnsetResourceSelector()
		} else {
			rule.SetResourceSelector(req.GetResourceSelector())
		}
	}
	if req.ApproverIdpId.IsSet() {
		if req.ApproverIdpId.Get() == nil {
			rule.SetApproverIdpIdNil()
		} else {
			rule.SetApproverIdpId(req.GetApproverIdpId())
		}
	}
	if req.ApproverProviderGroupId.IsSet() {
		if req.ApproverProviderGroupId.Get() == nil {
			rule.SetApproverProviderGroupIdNil()
		} else {
			rule.SetApproverProviderGroupId(req.GetApproverProviderGroupId())
		}
	}
	if req.HasExemptApiCredentials() {
		rule.SetExemptApiCredentials(req.GetExemptApiCredentials())
	}

	f.actionApprovalRules[id] = rule
	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewUpdateActionApprovalRuleResponse(*externalEonSdkAPI.NewNullableActionApprovalRule(rule)))
}

func (f *fakeEonServer) handleDeleteActionApprovalRule(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.actionApprovalRules[id]; !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	delete(f.actionApprovalRules, id)
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeEonServer) handleListActionApprovalRules(w http.ResponseWriter, r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)

	f.mu.Lock()
	defer f.mu.Unlock()
	rules := make([]externalEonSdkAPI.ActionApprovalRule, 0, len(f.actionApprovalRules))
	for _, rule := range f.actionApprovalRules {
		rules = append(rules, *rule)
	}
	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewListActionApprovalRulesResponse(rules, int32(len(rules))))
}

func (f *fakeEonServer) handleGetSnapshot(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	snap, ok := f.snapshotsByID[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewGetSnapshotResponse(*snap))
}

func (f *fakeEonServer) handleHoldSnapshot(w http.ResponseWriter, r *http.Request, id string) {
	var req externalEonSdkAPI.HoldSnapshotRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	f.mu.Lock()
	defer f.mu.Unlock()
	snap, ok := f.snapshotsByID[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	snap.SetOnHold(true)
	if req.HasDescription() {
		snap.SetHoldDescription(req.GetDescription())
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (f *fakeEonServer) handleRemoveSnapshotHold(w http.ResponseWriter, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	snap, ok := f.snapshotsByID[id]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	snap.SetOnHold(false)
	snap.HoldDescription = nil
	writeJSON(w, http.StatusOK, map[string]interface{}{})
}

func (f *fakeEonServer) handleGetRestoreJob(w http.ResponseWriter, jobID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	job, ok := f.restoreJobs[jobID]
	if !ok {
		http.Error(w, `{"message":"not found"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, externalEonSdkAPI.NewGetRestoreJobResponse(*job))
}

func (f *fakeEonServer) handleStartRestore(w http.ResponseWriter, r *http.Request, resourceID, snapshotID, action string) {
	_, _ = io.Copy(io.Discard, r.Body)

	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.resources[resourceID]; !ok {
		http.Error(w, `{"message":"resource not found"}`, http.StatusNotFound)
		return
	}
	if _, ok := f.snapshotsByID[snapshotID]; !ok {
		http.Error(w, `{"message":"snapshot not found"}`, http.StatusNotFound)
		return
	}

	switch action {
	case "restore-dynamo-db-table",
		"restore-azure-disk",
		"restore-azure-vm-instance",
		"restore-azure-sql-database",
		"convert-ec2-ebs-snapshot",
		"restore-ec2-ebs-volume",
		"restore-ec2-instance",
		"restore-rds-instance",
		"restore-bucket",
		"restore-files",
		"restore-gcp-vm-instance",
		"restore-gcp-disk",
		"restore-gcp-cloudsql",
		"restore-bigquery-dataset":
		// accepted restore actions for the fake server
	default:
		http.Error(w, fmt.Sprintf(`{"message":"unsupported restore action %s"}`, action), http.StatusNotFound)
		return
	}

	jobID := fmt.Sprintf("job-%d", f.nextID)
	f.nextID++
	details := externalEonSdkAPI.NewJobExecutionDetails(jobID, externalEonSdkAPI.JOB_COMPLETED, time.Now().UTC())
	destination := externalEonSdkAPI.NewDestinationDetails("restore-acct-1", "123456789012", externalEonSdkAPI.AWS, "us-east-1")
	job := externalEonSdkAPI.NewRestoreJob(*details, *destination, externalEonSdkAPI.AWS_DYNAMO_DB_TABLE_RESTORE)
	f.restoreJobs[jobID] = job
	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"jobId":                 jobID,
		"actionApprovalRequest": nil,
	})
}

func newTestInventoryResource(id string) *externalEonSdkAPI.InventoryResource {
	return externalEonSdkAPI.NewInventoryResource(
		id,
		externalEonSdkAPI.PROTECTED,
		"i-1234567890abcdef0",
		"demo-resource",
		"123456789012",
		*externalEonSdkAPI.NewSnapshotStorage(),
		*externalEonSdkAPI.NewSourceStorage(),
		map[string]string{},
		externalEonSdkAPI.AWS,
		externalEonSdkAPI.AWS_EC2,
		"us-east-1",
	)
}

func newTestIdp(id, name string) *externalEonSdkAPI.Idp {
	return externalEonSdkAPI.NewIdp(id, name)
}
